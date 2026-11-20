// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"bytes"
	"log"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.sdls.io/rapid/internal/bitstream"
	"go.sdls.io/rapid/internal/engine"
	"go.sdls.io/rapid/internal/stattest"
)

// search runs prop for 100 test cases generated from seed, without fail files.
func search(tb tb, seed uint64, prop func(*T)) engine.Result {
	tb.Helper()
	lim := engine.Limits{Checks: 100, Deadline: engine.Deadline(nil), ShrinkTime: engine.Flags.ShrinkTime}
	return engine.Search(tb, lim, seed, "", false, runner(tb, prop))
}

func brokenGen(*T) int { panic("this generator is not working") }

type brokenMachine struct{}

func (m *brokenMachine) DoNothing(_ *T) { panic("this state machine is not working") }
func (m *brokenMachine) Check(_ *T)     {}

func TestPanicTraceback(t *testing.T) {
	t.Parallel()

	testData := []struct {
		name   string
		suffix string
		fail   func(*T) *engine.TestError
	}{
		{
			"broken custom generator",
			"go.sdls.io/rapid.brokenGen",
			func(t *T) *engine.TestError {
				g := Custom(brokenGen)
				_, err := recoverValue(g, t)
				return err
			},
		},
		{
			"broken state machine",
			"go.sdls.io/rapid.(*brokenMachine).DoNothing",
			func(t *T) *engine.TestError {
				return checkOnce(t, func(t *T) {
					var sm brokenMachine
					t.RepeatSteps(StateMachineActions(&sm), 30) // the broken action runs whatever -rapid.steps says
				})
			},
		},
	}

	for _, td := range testData {
		t.Run(td.name, func(t *testing.T) {
			t.Parallel()

			s := bitstream.NewRandom(stattest.FixedSeed, false) // a seed whose state machine reaches DoNothing
			nt := newT(t, s, false, nil)

			err := td.fail(nt)
			if err == nil {
				t.Fatalf("test case did not fail")
			}

			lines := strings.Split(err.Trace, "\n")
			if !strings.HasSuffix(lines[0], td.suffix) {
				t.Errorf("bad traceback:\n%v", err.Trace)
			}
		})
	}
}

func BenchmarkCheckOverhead(b *testing.B) {
	g := Int[uint]()
	f := func(t *T) {
		g.Draw(t, "")
	}
	for b.Loop() {
		engine.Check(b, engine.Limits{}, runner(b, f))
	}
}

// A cleanup that panics fails the test case, and the cleanups registered before it still run, in reverse order.
func TestCheckCleanupPanic(t *testing.T) {
	t.Parallel()

	var ran []int
	prop := func(t *T) {
		for idx := range 5 {
			t.Cleanup(func() {
				ran = append(ran, idx)
				if idx == 2 {
					panic("cleanup panic")
				}
			})
		}
	}

	err := checkOnce(newT(t, bitstream.NewRandom(stattest.FixedSeed, false), false, nil), prop)
	if err == nil {
		t.Fatalf("cleanup panic not reported")
	}
	if want := []int{4, 3, 2, 1, 0}; !reflect.DeepEqual(ran, want) {
		t.Fatalf("cleanups ran %v, want %v", ran, want)
	}
}

func TestOutputRawLog(t *testing.T) {
	t.Parallel()

	msg := []byte("Hello World")

	var out bytes.Buffer
	run := runner(t, func(t *T) {
		if _, err := t.Output().Write(msg); err != nil {
			t.Fatal(err)
		}
	})
	_ = run(bitstream.NewBuffer(nil, false), false, log.New(&out, "", 0))

	if !bytes.Contains(out.Bytes(), msg) {
		t.Errorf("expected output to contain %q, got: %q", msg, out.Bytes())
	}
}

// A failure the property records, with Error, Errorf or Fail, belongs to the test case that recorded it:
// rapid reports it, and replaying that test case reproduces it.
func TestCheckRecordedFailureBelongsToItsTestCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		prop func(*T)
	}{
		{"in a cleanup", func(t *T) {
			x := IntRange(0, 9).Draw(t, "x")
			t.Cleanup(func() {
				if x == 3 {
					t.Errorf("leak at %d", x)
				}
			})
		}},
		{"before a skip", func(t *T) {
			if x := IntRange(0, 9).Draw(t, "x"); x == 3 {
				t.Errorf("bad %d", x)
				t.SkipNow()
			}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res := search(t, stattest.FixedSeed, tc.prop)
			if res.Err1 == nil {
				t.Fatalf("recorded failure was not reported (valid %d, invalid %d)", res.Valid, res.Invalid)
			}
			if !engine.SameError(res.Err1, res.Err2) {
				t.Fatalf("recorded failure did not reproduce: found %v, replay %v", res.Err1, res.Err2)
			}
		})
	}
}

// The test case Search reports, which Check saves to a fail file, fails the way Search reports when replayed.
func TestSearchReportedTestCaseReplaysItsFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		deadline time.Duration // from now; under 10s leaves no time to shrink
		prop     func(*T)
	}{
		{"recorded in a skipped Repeat action", time.Hour, func(t *T) {
			t.RepeatSteps(map[string]func(*T){"act": func(t *T) {
				if IntRange(0, 9).Draw(t, "x") == 3 {
					t.Errorf("recorded")
					t.SkipNow()
				}
			}}, 5)
		}},
		{"recorded in a skipped Custom try", time.Hour, func(t *T) {
			Custom(func(t *T) int {
				x := IntRange(0, 9).Draw(t, "x")
				if x == 3 {
					t.Errorf("recorded")
					t.SkipNow()
				}
				return x
			}).Draw(t, "v")
		}},
		{"Repeat stopped by rejected actions, not shrunk", 5 * time.Second, func(t *T) {
			n := 0
			actions := map[string]func(*T){"inc": func(*T) { n++ }}
			for _, key := range []string{"rej0", "rej1", "rej2", "rej3", "rej4"} {
				actions[key] = func(t *T) {
					Bool().Draw(t, "b")
					t.SkipNow()
				}
			}
			t.RepeatSteps(actions, 1000)
			if n >= 1 {
				t.Fatalf("n = %d", n)
			}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			lim := engine.Limits{Checks: 100, Deadline: time.Now().Add(tc.deadline), ShrinkTime: engine.Flags.ShrinkTime}
			res := engine.Search(t, lim, stattest.FixedSeed, "", false, runner(t, tc.prop))
			if res.Err2 == nil {
				t.Fatalf("no failure found")
			}

			replay := checkOnce(newT(t, bitstream.NewBuffer(res.Buf, false), false, nil), tc.prop)
			if !engine.SameError(res.Err2, replay) {
				t.Fatalf("reported %q on test case %v, which replays as %v", res.Err2, res.Buf, replay)
			}
		})
	}
}
