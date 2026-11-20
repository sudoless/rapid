// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.sdls.io/rapid/internal/assert"
	"go.sdls.io/rapid/internal/bitstream"
	"go.sdls.io/rapid/internal/engine"
)

const (
	rapidVersion = "v0.4.8"

	small             = 5
	invalidChecksMult = 10
	exampleMaxTries   = 1000

	maxTestTimeout  = 24 * time.Hour
	shrinkStepBound = 10 * time.Second // can be improved by taking average checkOnce runtime into account

	tracebackLen  = 32
	tracebackStop = "go.sdls.io/rapid.checkOnce"
	runtimePrefix = "runtime."
)

var tracebackBlacklist = map[string]bool{
	"go.sdls.io/rapid.(*customGen[...]).maybeValue.func1": true,
	"go.sdls.io/rapid.runAction.func1":                    true,
}

func checkDeadline(tb tb) time.Time {
	t, ok := tb.(*testing.T)
	if !ok {
		return time.Now().Add(maxTestTimeout)
	}
	d, ok := t.Deadline()
	if !ok {
		return time.Now().Add(maxTestTimeout)
	}
	return d
}

func shrinkDeadline(deadline time.Time) time.Time {
	d := time.Now().Add(engine.Flags.ShrinkTime)
	latest := deadline.Add(-shrinkStepBound) // account for the fact that shrink deadline is checked before the step
	if d.After(latest) {
		d = latest
	}
	return d
}

// Check fails the current test if rapid can find a test case which falsifies prop.
//
// Property is falsified in case of a panic or a call to
// [*T.Fatalf], [*T.Fatal], [*T.Errorf], [*T.Error], [*T.FailNow] or [*T.Fail].
func Check(t TB, prop func(*T)) {
	t.Helper()
	checkTB(t, checkDeadline(t), prop)
}

// MakeCheck is a convenience function for defining subtests suitable for
// [*testing.T.Run]. It allows you to write this:
//
//	t.Run("subtest name", rapid.MakeCheck(func(t *rapid.T) {
//	    // test code
//	}))
//
// instead of this:
//
//	t.Run("subtest name", func(t *testing.T) {
//	    rapid.Check(t, func(t *rapid.T) {
//	        // test code
//	    })
//	})
func MakeCheck(prop func(*T)) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		checkTB(t, checkDeadline(t), prop)
	}
}

// MakeFuzz creates a fuzz target for [*testing.F.Fuzz]:
//
//	func FuzzFoo(f *testing.F) {
//	    f.Fuzz(rapid.MakeFuzz(func(t *rapid.T) {
//	        // test code
//	    }))
//	}
func MakeFuzz(prop func(*T)) func(*testing.T, []byte) {
	return func(t *testing.T, input []byte) {
		t.Helper()
		checkFuzz(t, prop, input)
	}
}

func checkFuzz(tb tb, prop func(*T), input []byte) {
	tb.Helper()

	var buf []uint64
	for len(input) > 0 {
		var tmp [8]byte
		n := copy(tmp[:], input)
		buf = append(buf, binary.LittleEndian.Uint64(tmp[:]))
		input = input[n:]
	}

	t := newT(tb, bitstream.NewBuffer(buf, false), true, nil)
	err := checkOnce(t, prop)

	switch {
	case err == nil:
		// do nothing
	case err.IsInvalidData():
		tb.SkipNow()
	case err.IsStopTest():
		tb.Fatalf("[rapid] failed: %v", err)
	default:
		tb.Fatalf("[rapid] panic: %v\nTraceback:\n%v", err, engine.Traceback(err))
	}
}

func checkTB(tb tb, deadline time.Time, prop func(*T)) {
	tb.Helper()

	checks := engine.Flags.Checks
	if testing.Short() {
		checks /= 5
	}

	start := time.Now()
	res := doCheck(tb, deadline, checks, engine.BaseSeed(), engine.Flags.FailFile, true, prop)
	dt := time.Since(start)

	switch {
	case res.err1 != nil || res.err2 != nil:
		reportFailure(tb, prop, &res)
	case res.valid == checks || (res.earlyExit && res.valid > 0):
		tb.Logf("[rapid] OK, passed %v tests (%v)", res.valid, dt)
	default:
		tb.Errorf("[rapid] only generated %v valid tests from %v total (%v)", res.valid, res.valid+res.invalid, dt)
	}

	if tb.Failed() {
		tb.FailNow() // do not try to run any checks after the first failed one
	}
}

// reportFailure saves a fail file for res when there is none, reports the failure with how to reproduce it,
// and replays the failing test case so its output appears in the test log.
func reportFailure(tb tb, prop func(*T), res *checkResult) {
	tb.Helper()

	failPath := res.failfile
	if failPath == "" && !engine.Flags.NoFailFile {
		failPath = engine.FailFileName(tb.Name())
		out := captureTestOutput(tb, prop, res.buf)
		err := engine.SaveFailFile(failPath, rapidVersion, out, res.seed, res.buf)
		if err != nil {
			tb.Logf("[rapid] %v", err)
			failPath = ""
		}
	}

	repr := reproFlags(failPath, res.seed)
	name := regexp.QuoteMeta(tb.Name())
	err1, err2 := res.err1, res.err2
	switch {
	case engine.Traceback(err1) != engine.Traceback(err2):
		tb.Errorf("[rapid] flaky test, can not reproduce a failure\n"+
			"To try to reproduce, specify -run=%q %v\nTraceback (%v):\n%vOriginal traceback (%v):\n%vFailed test output:",
			name, repr, err2, engine.Traceback(err2), err1, engine.Traceback(err1))
	case err2.IsStopTest():
		tb.Errorf("[rapid] failed after %v tests: %v\nTo reproduce, specify -run=%q %v\nFailed test output:", res.valid, err2, name, repr)
	default:
		tb.Errorf("[rapid] panic after %v tests: %v\nTo reproduce, specify -run=%q %v\nTraceback:\n%vFailed test output:",
			res.valid, err2, name, repr, engine.Traceback(err2))
	}

	_ = checkOnce(newT(tb, bitstream.NewBuffer(res.buf, false), true, nil), prop) // output using (*testing.T).Log for proper line numbers
}

// reproFlags returns the flags that replay a failure from failPath or seed; either may be unset.
func reproFlags(failPath string, seed uint64) string {
	switch {
	case failPath != "" && seed != 0:
		return fmt.Sprintf("-rapid.failfile=%q (or -rapid.seed=%d)", failPath, seed)
	case failPath != "":
		return fmt.Sprintf("-rapid.failfile=%q", failPath)
	case seed != 0:
		return fmt.Sprintf("-rapid.seed=%d", seed)
	default:
		return ""
	}
}

// checkResult is what doCheck learned about a property.
type checkResult struct {
	valid     int
	invalid   int
	earlyExit bool     // the deadline stopped the search before checks valid test cases ran
	seed      uint64   // the seed of the failing test case; 0 when it came from a fail file or nothing failed
	failfile  string   // the fail file the failure was replayed from
	buf       []uint64 // the failing test case, shrunk when the failure reproduced
	err1      *engine.TestError
	err2      *engine.TestError // the failure on replay; differs from err1 when the test is flaky
}

func doCheck(tb tb, deadline time.Time, checks int, seed uint64, failPath string, globFailFiles bool, prop func(*T)) checkResult {
	tb.Helper()

	assert.Truef(!tb.Failed(), "check function called with *testing.T which has already failed")

	var failfiles []string
	if failPath != "" {
		failfiles = []string{failPath}
	}
	if globFailFiles {
		matches, _ := filepath.Glob(engine.FailFilePattern(tb.Name()))
		failfiles = append(failfiles, matches...)
	}
	for _, failPath := range failfiles {
		buf, err1, err2 := checkFailFile(tb, failPath, prop)
		if err1 != nil || err2 != nil {
			return checkResult{failfile: failPath, buf: buf, err1: err1, err2: err2}
		}
	}

	valid, invalid, earlyExit, seed, err1 := findBug(tb, deadline, checks, seed, prop)
	if err1 == nil {
		return checkResult{valid: valid, invalid: invalid, earlyExit: earlyExit}
	}

	s := bitstream.NewRandom(seed, true)
	t := newT(tb, s, engine.Flags.Verbose, nil)
	t.Logf("[rapid] trying to reproduce the failure")
	err2 := checkOnce(t, prop)
	if !engine.SameError(err1, err2) {
		return checkResult{valid: valid, invalid: invalid, seed: seed, buf: s.Data, err1: err1, err2: err2}
	}

	t.Logf("[rapid] trying to minimize the failing test case")
	run := func(s bitstream.Stream, log bool) *engine.TestError { return checkOnce(newT(tb, s, log, nil), prop) }
	buf, err3 := engine.Shrink(tb, shrinkDeadline(deadline), s.Recorded, err2, run)

	return checkResult{valid: valid, invalid: invalid, seed: seed, buf: buf, err1: err2, err2: err3}
}

func checkFailFile(tb tb, failPath string, prop func(*T)) ([]uint64, *engine.TestError, *engine.TestError) {
	tb.Helper()

	version, _, buf, err := engine.LoadFailFile(failPath)
	if err != nil {
		tb.Logf("[rapid] ignoring fail file: %v", err)
		return nil, nil, nil
	}
	if version != rapidVersion {
		tb.Logf("[rapid] ignoring fail file: version %q differs from rapid version %q", version, rapidVersion)
		return nil, nil, nil
	}

	s1 := bitstream.NewBuffer(buf, false)
	t1 := newT(tb, s1, engine.Flags.Verbose, nil)
	err1 := checkOnce(t1, prop)
	if err1 == nil {
		return nil, nil, nil
	}
	if err1.IsInvalidData() {
		tb.Logf("[rapid] fail file %q is no longer valid", failPath)
		return nil, nil, nil
	}

	s2 := bitstream.NewBuffer(buf, false)
	t2 := newT(tb, s2, engine.Flags.Verbose, nil)
	t2.Logf("[rapid] trying to reproduce the failure")
	err2 := checkOnce(t2, prop)

	return buf, err1, err2
}

func findBug(tb tb, deadline time.Time, checks int, seed uint64, prop func(*T)) (int, int, bool, uint64, *engine.TestError) {
	tb.Helper()

	var (
		r       = bitstream.NewRandom(0, false)
		valid   = 0
		invalid = 0
	)

	var total time.Duration
	for valid < checks && invalid < checks*invalidChecksMult {
		iter := valid + invalid
		t := newT(tb, r, engine.Flags.Verbose, nil)
		if iter > 0 && time.Until(deadline) < total/time.Duration(iter)*5 {
			if t.shouldLog() {
				t.Logf("[rapid] early exit after test #%v (%v)", iter, total)
			}
			return valid, invalid, true, 0, nil
		}

		seed += uint64(iter)
		r.Init(seed)
		dt, err := checkLogged(t, prop, iter+1, seed)
		total += dt
		switch {
		case err == nil:
			valid++
		case err.IsInvalidData():
			invalid++
		default:
			return valid, invalid, false, seed, err
		}
	}

	return valid, invalid, false, 0, nil
}

// checkLogged runs test case number num, generated from seed, and logs its outcome when t logs.
// The logging is guarded so a quiet run does not build the log arguments.
func checkLogged(t *T, prop func(*T), num int, seed uint64) (time.Duration, *engine.TestError) {
	if t.tbLog {
		t.Helper()
	}
	logs := t.shouldLog()
	start := time.Now()
	if logs {
		t.Logf("[rapid] test #%v start (seed %v)", num, seed)
	}

	err := checkOnce(t, prop)
	dt := time.Since(start)
	if !logs {
		return dt, err
	}

	switch {
	case err == nil:
		t.Logf("[rapid] test #%v OK (%v)", num, dt)
	case err.IsInvalidData():
		t.Logf("[rapid] test #%v invalid (%v)", num, dt)
	default:
		t.Logf("[rapid] test #%v failed: %v", num, err)
	}

	return dt, err
}

// checkOnce runs prop as one test case, and returns how it failed, after its cleanups.
func checkOnce(t *T, prop func(*T)) (err *engine.TestError) {
	if t.tbLog {
		t.Helper()
	}
	defer func() { err = t.settle(panicToError(recover())) }()

	defer t.cleanup()
	prop(t)

	return nil
}

func captureTestOutput(tb tb, prop func(*T), buf []uint64) []byte {
	var b bytes.Buffer
	l := log.New(&b, fmt.Sprintf("[%v] ", tb.Name()), log.Lmsgprefix|log.Ldate|log.Ltime|log.Lmicroseconds)
	_ = checkOnce(newT(tb, bitstream.NewBuffer(buf, false), false, l), prop)
	return b.Bytes()
}

func panicToError(p any) *engine.TestError {
	if p == nil {
		return nil
	}

	if err, ok := p.(*engine.TestError); ok {
		return err
	}

	return newTestError(p, 4) // skip runtime.Callers, newTestError, panicToError and the deferred recover
}

// newTestError returns p with the traceback of the goroutine, from skip frames up to checkOnce.
func newTestError(p any, skip int) *engine.TestError {
	callers := make([]uintptr, tracebackLen)
	callers = callers[:runtime.Callers(skip, callers)]
	frames := runtime.CallersFrames(callers)

	b := &strings.Builder{}
	f, more, skipSpecial := runtime.Frame{}, true, true
	for more && !strings.HasSuffix(f.Function, tracebackStop) {
		f, more = frames.Next()

		if skipSpecial && (tracebackBlacklist[f.Function] || strings.HasPrefix(f.Function, runtimePrefix)) {
			continue
		}
		skipSpecial = false

		_, err := fmt.Fprintf(b, "    %s:%d in %s\n", f.File, f.Line, f.Function)
		assert.True(err == nil)
	}

	return &engine.TestError{
		Data:  p,
		Trace: b.String(),
	}
}

// TB is a common interface between [*testing.T], [*testing.B] and [*T].
type TB interface {
	Helper()
	Name() string
	Logf(format string, args ...any)
	Log(args ...any)
	Skipf(format string, args ...any)
	Skip(args ...any)
	SkipNow()
	Errorf(format string, args ...any)
	Error(args ...any)
	Fatalf(format string, args ...any)
	Fatal(args ...any)
	FailNow()
	Fail()
	Failed() bool
	Context() context.Context
	Output() io.Writer
}

type tb TB // tb is a private copy of TB, made to avoid T having public fields

type nilTB struct{}

func (nilTB) Helper()                  {}
func (nilTB) Name() string             { return "" }
func (nilTB) Logf(string, ...any)      {}
func (nilTB) Log(...any)               {}
func (nilTB) Skipf(string, ...any)     { panic("call to TB.Skipf() outside a test") }
func (nilTB) Skip(...any)              { panic("call to TB.Skip() outside a test") }
func (nilTB) SkipNow()                 { panic("call to TB.SkipNow() outside a test") }
func (nilTB) Errorf(string, ...any)    { panic("call to TB.Errorf() outside a test") }
func (nilTB) Error(...any)             { panic("call to TB.Error() outside a test") }
func (nilTB) Fatalf(string, ...any)    { panic("call to TB.Fatalf() outside a test") }
func (nilTB) Fatal(...any)             { panic("call to TB.Fatal() outside a test") }
func (nilTB) FailNow()                 { panic("call to TB.FailNow() outside a test") }
func (nilTB) Fail()                    { panic("call to TB.Fail() outside a test") }
func (nilTB) Failed() bool             { panic("call to TB.Failed() outside a test") }
func (nilTB) Context() context.Context { return context.Background() }
func (nilTB) Output() io.Writer        { return io.Discard }

// T is similar to [testing.T], but with extra bookkeeping for property-based tests.
//
// For tests to be reproducible, they should generally run in a single goroutine.
// If concurrency is unavoidable, methods on *T, such as [*testing.T.Helper] and [*T.Errorf],
// are safe for concurrent calls, but *Generator.Draw from a given *T is not.
type T struct {
	tb // unnamed to force re-export of (*T).Helper()

	ctx       context.Context //nolint:containedctx // backs T.Context, like testing.T
	cancelCtx context.CancelFunc
	cleanups  []func()
	cleaning  atomic.Bool

	tbLog  bool
	rawLog *log.Logger
	s      bitstream.Stream
	draws  int
	onDraw func(idx int, v any) // called with every value drawn through t; set by tests
	mu     sync.RWMutex

	tc  *testCase // the test case this T belongs to: own, or its parent's for a Custom value or SyncTest bubble
	own testCase
}

// testCase is the outcome of one test case, shared by the Ts its property and generators receive.
type testCase struct {
	mu     sync.Mutex
	failed *engine.TestError // the last failure recorded with Error, Errorf, Fail, Fatal, Fatalf or FailNow
}

func (tc *testCase) record(err *engine.TestError) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.failed = err
}

func (tc *testCase) recorded() *engine.TestError {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	return tc.failed
}

func newT(tb tb, s bitstream.Stream, tbLog bool, rawLog *log.Logger) *T {
	if tb == nil {
		tb = nilTB{}
	}

	t := &T{
		tb:     tb,
		tbLog:  tbLog,
		rawLog: rawLog,
		s:      s,
	}
	t.tc = &t.own

	if rawLog == nil && engine.Flags.Log {
		testName := "rapid test"
		if tb != nil {
			testName = tb.Name()
		}

		t.rawLog = log.New(os.Stdout, fmt.Sprintf("[%v] ", testName), log.Lmsgprefix|log.Ldate|log.Ltime|log.Lmicroseconds)
	}

	return t
}

// child returns a T for a nested scope of t's test case, such as one Custom value: it draws from t's stream
// and records failures to t's test case, with its own Context, Cleanups and draw count.
func (t *T) child(tb tb, tbLog bool, rawLog *log.Logger) *T {
	return &T{
		tb:     tb,
		tbLog:  tbLog,
		rawLog: rawLog,
		s:      t.s,
		tc:     t.tc,
	}
}

// settle returns the outcome of t's test case, given err, what its run panicked with.
// A recorded failure outranks an invalid test case, as a testing.T that fails and then skips stays failed.
func (t *T) settle(err *engine.TestError) *engine.TestError {
	if err == nil || err.IsInvalidData() {
		if recorded := t.tc.recorded(); recorded != nil {
			return recorded
		}
	}

	return err
}

func (t *T) shouldLog() bool {
	return t.rawLog != nil || t.tbLog
}

// Context returns a context.Context that is canceled
// after the property function exits,
// before Cleanup-registered functions are run.
//
// For [Check], [MakeFuzz], and similar functions,
// each call to the property function gets a unique context
// that is canceled after that property function exits.
//
// For [Custom], each time a new value is generated,
// the generator function gets a unique context
// that is canceled after the generator function exits.
func (t *T) Context() context.Context {
	// Fast path: no need to lock if the context is already set.
	t.mu.RLock()
	ctx := t.ctx
	t.mu.RUnlock()
	if ctx != nil {
		return ctx
	}

	// If we're in the middle of cleaning up
	// and the context has already been canceled and cleared,
	// don't create a new one. Return a canceled context instead.
	if t.cleaning.Load() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx
	}

	// Slow path: lock and check again, create new context if needed.
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.ctx != nil {
		// Another goroutine set the context
		// while we were waiting for the lock.
		return t.ctx
	}

	ctx, cancel := context.WithCancel(t.tb.Context())
	t.ctx = ctx
	t.cancelCtx = cancel
	return ctx
}

// Cleanup registers a function to be called
// when a property function finishes running.
//
// For [Check], [MakeFuzz], and similar functions,
// each call to the property function registers its cleanup functions,
// which are called after the property function exits.
//
// For [Custom], each time a new value is generated,
// the generator function registers its cleanup functions,
// which are called after the generator function exits.
//
// Cleanup functions are called in last-in, first-out order.
//
// If [T.Context] is used, the context is canceled
// before the Cleanup functions are executed.
func (t *T) Cleanup(f func()) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.cleanups = append(t.cleanups, f)
}

// cleanup runs any cleanup tasks associated with the property check.
// It is safe to call multiple times.
func (t *T) cleanup() {
	t.cleaning.Store(true)
	defer t.cleaning.Store(false)

	// If a cleanup function panics,
	// we still want to run the remaining cleanup functions.
	defer func() {
		t.mu.Lock()
		recurse := len(t.cleanups) > 0
		t.mu.Unlock()

		if recurse {
			t.cleanup()
		}
	}()

	// Context must be closed before t.Cleanup functions are run.
	t.mu.Lock()
	if t.cancelCtx != nil {
		t.cancelCtx()
		t.cancelCtx = nil
		t.ctx = nil
	}
	t.mu.Unlock()

	for {
		var cleanup func()
		t.mu.Lock()
		if len(t.cleanups) > 0 {
			last := len(t.cleanups) - 1
			cleanup = t.cleanups[last]
			t.cleanups = t.cleanups[:last]
		}
		t.mu.Unlock()

		if cleanup == nil {
			break
		}

		cleanup()
	}
}

func (t *T) Logf(format string, args ...any) {
	if t.rawLog != nil {
		t.rawLog.Printf(format, args...)
	} else if t.tbLog {
		t.Helper()
		t.tb.Logf(format, args...)
	}
}

func (t *T) Log(args ...any) {
	if t.rawLog != nil {
		t.rawLog.Print(args...)
	} else if t.tbLog {
		t.Helper()
		t.tb.Log(args...)
	}
}

// Output returns a Writer that writes to the same test output stream as T.Log.
// The output is indented like T.Log lines, but Output does not
// add source locations or newlines. The output is internally line
// buffered, and a call to T.Log or the end of the test will implicitly
// flush the buffer, followed by a newline. After a test function and all its
// parents return, neither Output nor the Write method may be called.
func (t *T) Output() io.Writer {
	t.Helper()

	switch {
	case t.rawLog != nil:
		return t.rawLog.Writer()
	case t.tbLog:
		return t.tb.Output()
	default:
		return io.Discard
	}
}

// Skipf is equivalent to [T.Logf] followed by [T.SkipNow].
func (t *T) Skipf(format string, args ...any) {
	if t.tbLog {
		t.Helper()
	}
	t.Logf(format, args...)
	t.skip(fmt.Sprintf(format, args...))
}

// Skip is equivalent to [T.Log] followed by [T.SkipNow].
func (t *T) Skip(args ...any) {
	if t.tbLog {
		t.Helper()
	}
	t.Log(args...)
	t.skip(fmt.Sprint(args...))
}

// SkipNow marks the current test case as invalid (except in [T.Repeat]
// actions, where it marks current action as non-applicable instead).
// If too many test cases are skipped, rapid will mark the test as failing
// due to inability to generate enough valid test cases.
//
// The test case or action will be treated like it had never been drawn
// and will not be shown in test logs.
// Therefore, to avoid confusing test failures later on,
// [SkipNow] must not be called after the action has already mutated shared state.
//
// Prefer *Generator.Filter to SkipNow, and prefer generators that always produce
// valid test cases to Filter.
func (t *T) SkipNow() {
	t.skip("(*T).SkipNow() called")
}

// Errorf is equivalent to [T.Logf] followed by [T.Fail].
func (t *T) Errorf(format string, args ...any) {
	if t.tbLog {
		t.Helper()
	}
	t.Logf(format, args...)
	t.fail(false, fmt.Sprintf(format, args...))
}

// Error is equivalent to [T.Log] followed by [T.Fail].
func (t *T) Error(args ...any) {
	if t.tbLog {
		t.Helper()
	}
	t.Log(args...)
	t.fail(false, fmt.Sprint(args...))
}

// Fatalf is equivalent to [T.Logf] followed by [T.FailNow].
func (t *T) Fatalf(format string, args ...any) {
	if t.tbLog {
		t.Helper()
	}
	t.Logf(format, args...)
	t.fail(true, fmt.Sprintf(format, args...))
}

// Fatal is equivalent to [T.Log] followed by [T.FailNow].
func (t *T) Fatal(args ...any) {
	if t.tbLog {
		t.Helper()
	}
	t.Log(args...)
	t.fail(true, fmt.Sprint(args...))
}

func (t *T) FailNow() {
	t.fail(true, "(*T).FailNow() called")
}

func (t *T) Fail() {
	t.fail(false, "(*T).Fail() called")
}

// Failed reports whether the current test case has failed.
func (t *T) Failed() bool {
	return t.tc.recorded() != nil
}

func (t *T) skip(msg string) {
	panic(bitstream.InvalidData(msg))
}

// fail records the failure msg for the test case, traced from the exported method's caller,
// and stops the test case when now is set.
func (t *T) fail(now bool, msg string) {
	err := newTestError(engine.StopTest(msg), 4) // skip runtime.Callers, newTestError, fail and the exported method
	t.tc.record(err)
	if now {
		panic(err)
	}
}

// failOnError stops the test case if it has recorded a failure.
func (t *T) failOnError() {
	if err := t.tc.recorded(); err != nil {
		panic(err)
	}
}
