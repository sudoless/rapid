// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"context"
	"fmt"
	"io"
	"log"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"go.sdls.io/rapid/internal/assert"
	"go.sdls.io/rapid/internal/bitstream"
	"go.sdls.io/rapid/internal/engine"
)

const (
	small           = 5
	exampleMaxTries = 1000

	tracebackLen  = 32
	tracebackStop = "go.sdls.io/rapid.checkOnce"
	runtimePrefix = "runtime."
)

var tracebackBlacklist = map[string]bool{
	"go.sdls.io/rapid.(*customGen[...]).maybeValue.func1": true,
	"go.sdls.io/rapid.runAction.func1":                    true,
}

// Check fails the current test if rapid can find a test case which falsifies prop. Options set how long
// it looks, in place of the -rapid.* flags.
//
// Property is falsified in case of a panic or a call to
// [*T.Fatalf], [*T.Fatal], [*T.Errorf], [*T.Error], [*T.FailNow] or [*T.Fail].
func Check(t TB, prop func(*T), opts ...CheckOption) {
	t.Helper()
	engine.Check(t, checkLimits(opts), runner(t, prop))
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
func MakeCheck(prop func(*T), opts ...CheckOption) func(*testing.T) {
	lim := checkLimits(opts)
	return func(t *testing.T) {
		t.Helper()
		engine.Check(t, lim, runner(t, prop))
	}
}

// Fuzz makes prop the fuzz target of f, each fuzz input encoding one test case. Fuzz calls [*testing.F.Fuzz],
// so seed inputs are added with [*testing.F.Add] before it:
//
//	func FuzzFoo(f *testing.F) {
//	    rapid.Fuzz(f, func(t *rapid.T) {
//	        // test code
//	    })
//	}
func Fuzz(f *testing.F, prop func(*T)) {
	f.Helper()
	engine.Fuzz(f, func(t *testing.T) engine.Run { return runner(t, prop) })
}

// runner returns prop as the engine runs it in the test tb: each run is one test case, with a T of its own.
func runner(tb tb, prop func(*T)) engine.Run {
	return func(s bitstream.Stream, tbLog bool, out *log.Logger) (*engine.TestError, []string) {
		if tbLog {
			tb.Helper()
		}
		t := newT(tb, s, tbLog, out)
		err := checkOnce(t, prop)
		return err, t.tc.marked()
	}
}

// checkOnce runs prop as one test case, and returns how it failed, after its cleanups.
func checkOnce(t *T, prop func(*T)) (err *engine.TestError) {
	if t.tbLog {
		t.Helper()
	}
	var propErr *engine.TestError
	defer func() { err = t.settle(firstFailure(propErr, panicToError(recover()))) }()

	defer t.cleanup()
	defer func() { propErr = panicToError(recover()) }() // before the cleanups, which can panic in turn
	prop(t)

	return nil
}

// firstFailure returns the property's outcome, unless only its cleanups failed: a skip in a cleanup does not
// hide a failure of the property.
func firstFailure(propErr, cleanupErr *engine.TestError) *engine.TestError {
	if propErr != nil && !propErr.IsInvalidData() || cleanupErr == nil {
		return propErr
	}
	return cleanupErr
}

func panicToError(p any) *engine.TestError {
	if p == nil {
		return nil
	}

	if err, ok := p.(*engine.TestError); ok {
		return err
	}
	if invalid, ok := p.(bitstream.InvalidData); ok {
		return &engine.TestError{Data: invalid} // an invalid test case is discarded, so its traceback would go unread
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
// A property receives a new T for every test case it runs.
//
// For tests to be reproducible, they should generally run in a single goroutine.
// As with testing.T, a test case ends when the property returns or calls FailNow, Fatal, Fatalf,
// SkipNow, Skip or Skipf, and those methods must be called only from the goroutine running the property
// (or the Custom function or SyncTest bubble the T was given to). The other methods, such as the
// variations of Log and Error, and Event, may be called from several goroutines at once.
// *Generator.Draw from a given T may not.
type T struct {
	helperTB // embedded so t.Helper marks its caller; a method that called tb.Helper would mark itself
	tb       tb

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

var _ TB = (*T)(nil)

// helperTB is the part of the test's TB that T exposes as is.
type helperTB interface {
	Helper()
	Name() string
}

// testCase is the outcome of one test case, shared by the Ts its property and generators receive.
type testCase struct {
	mu      sync.Mutex
	failed  *engine.TestError // the last failure recorded with Error, Errorf, Fail, Fatal, Fatalf or FailNow
	events  []string          // the events marked with Event, each once
	nEvents atomic.Int64      // len(events), read without mu
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
		helperTB: tb,
		tb:       tb,
		tbLog:    tbLog,
		rawLog:   rawLog,
		s:        s,
	}
	t.tc = &t.own

	return t
}

// child returns a T for a nested scope of t's test case, such as one Custom value: it draws from t's stream
// and records failures to t's test case, with its own Context, Cleanups and draw count.
func (t *T) child(tb tb, tbLog bool, rawLog *log.Logger) *T {
	return &T{
		helperTB: tb,
		tb:       tb,
		tbLog:    tbLog,
		rawLog:   rawLog,
		s:        t.s,
		tc:       t.tc,
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

// Context returns a context.Context that is canceled
// after the property function exits,
// before Cleanup-registered functions are run.
//
// For [Check], [Fuzz], and similar functions,
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

	// Once the property has exited and cleanup has begun, the context has been canceled and cleared.
	// Don't create a new one, for a cleanup or a goroutine the property left running: return a canceled one.
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
// For [Check], [Fuzz], and similar functions,
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
	t.cleaning.Store(true) // for good: the test case is over, and so is its context

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
// and will not be shown in test logs. A test case that has already recorded a failure
// with Error, Errorf or Fail stays failed, as a testing.T does.
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

// recordedTrace is the traceback of every failure recorded with Error, Errorf or Fail. The shrinker keeps
// a smaller test case only when it fails with the same traceback, so these failures all count as one,
// wherever they were recorded; a Fatal or panic keeps the traceback of where it happened.
const recordedTrace = "    recorded with (*T).Error, (*T).Errorf or (*T).Fail\n"

// fail records the failure msg for the test case, and stops the test case when now is set.
func (t *T) fail(now bool, msg string) {
	if !now {
		t.tc.record(&engine.TestError{Data: engine.StopTest(msg), Trace: recordedTrace})
		return
	}

	err := newTestError(engine.StopTest(msg), 4) // skip runtime.Callers, newTestError, fail and the exported method
	t.tc.record(err)
	panic(err)
}

// failOnError stops the test case if it has recorded a failure.
func (t *T) failOnError() {
	if err := t.tc.recorded(); err != nil {
		panic(err)
	}
}
