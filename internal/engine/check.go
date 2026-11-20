// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package engine

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"go.sdls.io/rapid/internal/assert"
	"go.sdls.io/rapid/internal/bitstream"
)

const (
	version = "v0.4.8" // written to fail files; a fail file of another version is ignored

	invalidChecksMult = 10

	maxTestTimeout  = 24 * time.Hour
	shrinkStepBound = 10 * time.Second // can be improved by taking average run time into account
)

// TB is the test a property runs in.
type TB interface {
	Logger
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
	Failed() bool
	FailNow()
	SkipNow()
}

// NewLog returns a logger that writes to w the way -rapid.log does, prefixed with the test's name.
func NewLog(w io.Writer, testName string) *log.Logger {
	return log.New(w, fmt.Sprintf("[%v] ", testName), log.Lmsgprefix|log.Ldate|log.Ltime|log.Lmicroseconds)
}

// Deadline returns when the test tb has to end: the deadline of a *testing.T that has one,
// else a day from now.
func Deadline(tb TB) time.Time {
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
	d := time.Now().Add(Flags.ShrinkTime)
	latest := deadline.Add(-shrinkStepBound) // account for the fact that shrink deadline is checked before the step
	if d.After(latest) {
		d = latest
	}
	return d
}

// Check fails tb when run fails on a test case: one from a fail file, or one generated from the
// -rapid.* settings. It reports the failure shrunk, with how to reproduce it, and saves it to a fail file.
func Check(tb TB, run Run) {
	tb.Helper()

	checks := Flags.Checks
	if testing.Short() {
		checks /= 5
	}

	start := time.Now()
	res := Search(tb, Deadline(tb), checks, BaseSeed(), Flags.FailFile, true, run)
	dt := time.Since(start)

	switch {
	case res.Err1 != nil || res.Err2 != nil:
		report(tb, run, &res)
	case res.Valid == checks || (res.EarlyExit && res.Valid > 0):
		tb.Logf("[rapid] OK, passed %v tests (%v)", res.Valid, dt)
	default:
		tb.Errorf("[rapid] only generated %v valid tests from %v total (%v)", res.Valid, res.Valid+res.Invalid, dt)
	}

	if tb.Failed() {
		tb.FailNow() // do not try to run any checks after the first failed one
	}
}

// Fuzz runs run on the test case input encodes, as a fuzz target: it skips tb when the test case is invalid,
// and fails it when run fails.
func Fuzz(tb TB, input []byte, run Run) {
	tb.Helper()

	var buf []uint64
	for len(input) > 0 {
		var tmp [8]byte
		n := copy(tmp[:], input)
		buf = append(buf, binary.LittleEndian.Uint64(tmp[:]))
		input = input[n:]
	}

	err := run(bitstream.NewBuffer(buf, false), true, nil)

	switch {
	case err == nil:
		// do nothing
	case err.IsInvalidData():
		tb.SkipNow()
	case err.IsStopTest():
		tb.Fatalf("[rapid] failed: %v", err)
	default:
		tb.Fatalf("[rapid] panic: %v\nTraceback:\n%v", err, Traceback(err))
	}
}

// report saves a fail file for res when there is none, reports the failure with how to reproduce it,
// and replays the failing test case so its output appears in the test log.
func report(tb TB, run Run, res *Result) {
	tb.Helper()

	failPath := res.FailFile
	if failPath == "" && !Flags.NoFailFile {
		failPath = FailFileName(tb.Name())
		var out bytes.Buffer
		_ = run(bitstream.NewBuffer(res.Buf, false), false, NewLog(&out, tb.Name()))
		err := SaveFailFile(failPath, version, out.Bytes(), res.Seed, res.Buf)
		if err != nil {
			tb.Logf("[rapid] %v", err)
			failPath = ""
		}
	}

	repr := reproFlags(failPath, res.Seed)
	name := regexp.QuoteMeta(tb.Name())
	err1, err2 := res.Err1, res.Err2
	switch {
	case Traceback(err1) != Traceback(err2):
		tb.Errorf("[rapid] flaky test, can not reproduce a failure\n"+
			"To try to reproduce, specify -run=%q %v\nTraceback (%v):\n%vOriginal traceback (%v):\n%vFailed test output:",
			name, repr, err2, Traceback(err2), err1, Traceback(err1))
	case err2.IsStopTest():
		tb.Errorf("[rapid] failed after %v tests: %v\nTo reproduce, specify -run=%q %v\nFailed test output:", res.Valid, err2, name, repr)
	default:
		tb.Errorf("[rapid] panic after %v tests: %v\nTo reproduce, specify -run=%q %v\nTraceback:\n%vFailed test output:",
			res.Valid, err2, name, repr, Traceback(err2))
	}

	_ = run(bitstream.NewBuffer(res.Buf, false), true, nil) // output using (*testing.T).Log for proper line numbers
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

// Result is what Search learned about a property.
type Result struct {
	Valid     int
	Invalid   int
	EarlyExit bool     // the deadline stopped the search before checks valid test cases ran
	Seed      uint64   // the seed of the failing test case; 0 when it came from a fail file or nothing failed
	FailFile  string   // the fail file the failure was replayed from
	Buf       []uint64 // the failing test case, shrunk when the failure reproduced
	Err1      *TestError
	Err2      *TestError // the failure on replay; differs from Err1 when the test is flaky
}

// Search looks for a test case run fails on: first in the fail file failPath and, with globFailFiles,
// in every fail file saved for tb, then among checks test cases generated from seed, until deadline.
// It replays a failure it finds, and shrinks it when the replay fails the same way.
func Search(tb TB, deadline time.Time, checks int, seed uint64, failPath string, globFailFiles bool, run Run) Result {
	tb.Helper()

	assert.Truef(!tb.Failed(), "check function called with *testing.T which has already failed")

	v := newVerbose(tb)

	var failfiles []string
	if failPath != "" {
		failfiles = []string{failPath}
	}
	if globFailFiles {
		matches, _ := filepath.Glob(FailFilePattern(tb.Name()))
		failfiles = append(failfiles, matches...)
	}
	for _, failPath := range failfiles {
		buf, err1, err2 := checkFailFile(tb, v, failPath, run)
		if err1 != nil || err2 != nil {
			return Result{FailFile: failPath, Buf: buf, Err1: err1, Err2: err2}
		}
	}

	valid, invalid, earlyExit, seed, err1 := findBug(tb, v, deadline, checks, seed, run)
	if err1 == nil {
		return Result{Valid: valid, Invalid: invalid, EarlyExit: earlyExit}
	}

	s := bitstream.NewRandom(seed, true)
	v.logf("[rapid] trying to reproduce the failure")
	err2 := run(s, Flags.Verbose, nil)
	if !SameError(err1, err2) {
		return Result{Valid: valid, Invalid: invalid, Seed: seed, Buf: s.Data, Err1: err1, Err2: err2}
	}

	v.logf("[rapid] trying to minimize the failing test case")
	buf, err3 := Shrink(tb, shrinkDeadline(deadline), s.Recorded, err2, run)

	return Result{Valid: valid, Invalid: invalid, Seed: seed, Buf: buf, Err1: err2, Err2: err3}
}

func checkFailFile(tb TB, v verbose, failPath string, run Run) ([]uint64, *TestError, *TestError) {
	tb.Helper()

	fileVersion, _, buf, err := LoadFailFile(failPath)
	if err != nil {
		tb.Logf("[rapid] ignoring fail file: %v", err)
		return nil, nil, nil
	}
	if fileVersion != version {
		tb.Logf("[rapid] ignoring fail file: version %q differs from rapid version %q", fileVersion, version)
		return nil, nil, nil
	}

	err1 := run(bitstream.NewBuffer(buf, false), Flags.Verbose, nil)
	if err1 == nil {
		return nil, nil, nil
	}
	if err1.IsInvalidData() {
		tb.Logf("[rapid] fail file %q is no longer valid", failPath)
		return nil, nil, nil
	}

	v.logf("[rapid] trying to reproduce the failure")
	err2 := run(bitstream.NewBuffer(buf, false), Flags.Verbose, nil)

	return buf, err1, err2
}

func findBug(tb TB, v verbose, deadline time.Time, checks int, seed uint64, run Run) (int, int, bool, uint64, *TestError) {
	tb.Helper()

	var (
		r       = bitstream.NewRandom(0, false)
		valid   = 0
		invalid = 0
	)

	var total time.Duration
	for valid < checks && invalid < checks*invalidChecksMult {
		iter := valid + invalid
		if iter > 0 && time.Until(deadline) < total/time.Duration(iter)*5 {
			if v.logs() {
				v.logf("[rapid] early exit after test #%v (%v)", iter, total)
			}
			return valid, invalid, true, 0, nil
		}

		seed += uint64(iter)
		r.Init(seed)
		dt, err := runLogged(v, run, r, iter+1, seed)
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

// runLogged runs test case number num, generated from seed, and logs its outcome when v logs.
// The logging is guarded so a quiet run does not build the log arguments.
func runLogged(v verbose, run Run, s bitstream.Stream, num int, seed uint64) (time.Duration, *TestError) {
	if v.tbLog {
		v.tb.Helper()
	}
	logs := v.logs()
	start := time.Now()
	if logs {
		v.logf("[rapid] test #%v start (seed %v)", num, seed)
	}

	err := run(s, v.tbLog, nil)
	dt := time.Since(start)
	if !logs {
		return dt, err
	}

	switch {
	case err == nil:
		v.logf("[rapid] test #%v OK (%v)", num, dt)
	case err.IsInvalidData():
		v.logf("[rapid] test #%v invalid (%v)", num, dt)
	default:
		v.logf("[rapid] test #%v failed: %v", num, err)
	}

	return dt, err
}

// verbose is where Search reports its progress, as a property's own output goes:
// to stdout under -rapid.log, else to the test log under -rapid.v.
type verbose struct {
	tb    TB
	tbLog bool
	out   *log.Logger
}

func newVerbose(tb TB) verbose {
	v := verbose{tb: tb, tbLog: Flags.Verbose}
	if Flags.Log {
		v.out = NewLog(os.Stdout, tb.Name())
	}
	return v
}

func (v verbose) logs() bool {
	return v.out != nil || v.tbLog
}

func (v verbose) logf(format string, args ...any) {
	if v.out != nil {
		v.out.Printf(format, args...)
	} else if v.tbLog {
		v.tb.Helper()
		v.tb.Logf(format, args...)
	}
}
