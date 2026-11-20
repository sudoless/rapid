// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"bytes"
	"context"
	"errors"
	"log"
	"reflect"
	"strings"
	"testing"

	"go.sdls.io/rapid/internal/bitstream"
	"go.sdls.io/rapid/internal/engine"
	"go.sdls.io/rapid/internal/stattest"
)

// search runs prop for 100 test cases generated from seed, without fail files.
func search(tb tb, seed uint64, prop func(*T)) engine.Result {
	tb.Helper()
	return engine.Search(tb, engine.Deadline(nil), 100, seed, "", false, runner(tb, prop))
}

func brokenGen(*T) int { panic("this generator is not working") }

type brokenMachine struct{}

func (m *brokenMachine) DoNothing(_ *T) { panic("this state machine is not working") }
func (m *brokenMachine) Check(_ *T)     {}

func TestPanicTraceback(t *testing.T) {
	t.Parallel()

	testData := []struct {
		name       string
		suffix     string
		canSucceed bool
		fail       func(*T) *engine.TestError
	}{
		{
			"broken custom generator",
			"go.sdls.io/rapid.brokenGen",
			false,
			func(t *T) *engine.TestError {
				g := Custom(brokenGen)
				_, err := recoverValue(g, t)
				return err
			},
		},
		{
			"broken state machine",
			"go.sdls.io/rapid.(*brokenMachine).DoNothing",
			true,
			func(t *T) *engine.TestError {
				return checkOnce(t, func(t *T) {
					var sm brokenMachine
					t.Repeat(StateMachineActions(&sm))
				})
			},
		},
	}

	for _, td := range testData {
		t.Run(td.name, func(t *testing.T) {
			t.Parallel()

			s := bitstream.NewRandom(stattest.Seed(t), false)
			nt := newT(t, s, false, nil)

			err := td.fail(nt)
			if err == nil {
				if td.canSucceed {
					t.SkipNow()
				}
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
		engine.Check(b, runner(b, f))
	}
}

func TestCheckContext(t *testing.T) {
	t.Parallel()

	type key struct{}

	var ctx context.Context
	Check(t, func(t *T) {
		ctx = context.WithValue(t.Context(), key{}, Int[int]().Draw(t, "x"))
		if err := ctx.Err(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if err := ctx.Err(); err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context to be canceled, got: %v", err)
	}

	if _, ok := ctx.Value(key{}).(int); !ok {
		t.Fatalf("context must have a value")
	}
}

func TestCheckCleanup(t *testing.T) {
	t.Parallel()

	// Each Check iteration will append a true to indicate "open",
	// and flip it to false on cleanup.
	//
	// After the check is done, we expect all values to be false.
	var state []bool

	Check(t, func(t *T) {
		idx := len(state)
		state = append(state, true)
		t.Cleanup(func() {
			state[idx] = false
		})
	})

	for _, value := range state {
		if value {
			t.Fatalf("expected all values to be false")
		}
	}
}

func TestCheckCleanupMultipleOrder(t *testing.T) {
	t.Parallel()

	// If multiple cleanups are appended during a Check,
	// they must run in reverse order.
	var state []int
	Check(t, func(t *T) {
		// We just want to capture the result of one iteration,
		// so we'll keep resetting the state.
		state = nil
		t.Cleanup(func() {
			state = append(state, 1)
		})
		t.Cleanup(func() {
			state = append(state, 2)
		})
		t.Cleanup(func() {
			state = append(state, 3)
		})
	})

	if !reflect.DeepEqual(state, []int{3, 2, 1}) {
		t.Fatalf("expected cleanups to run in reverse order, got: %v", state)
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

func TestCheckCleanupNewCleanupsDuringCleanup(t *testing.T) {
	t.Parallel()

	// Cleanups can be added during cleanup.
	var state []bool
	Check(t, func(t *T) {
		idx := len(state)
		state = append(state, true)
		t.Cleanup(func() {
			// Odd numbered events will add a new cleanup.
			if idx%2 == 0 {
				state[idx] = false
			} else {
				t.Cleanup(func() {
					state[idx] = false
				})
			}
		})
	})

	for idx, open := range state {
		if open {
			t.Fatalf("cleanup %v registered during cleanup did not run", idx)
		}
	}
}

func TestCheckCleanupContextIsCanceled(t *testing.T) {
	t.Parallel()

	// Context created during Check is canceled by the time Cleanup is run.
	Check(t, func(t *T) {
		ctx := t.Context()
		t.Cleanup(func() {
			if err := ctx.Err(); err == nil || !errors.Is(err, context.Canceled) {
				t.Fatalf("expected context to be canceled, got: %v", ctx)
			}
		})
	})
}

func TestCheckCleanupContextCreatedInCleanup(t *testing.T) {
	t.Parallel()

	// Context created during Cleanup is already canceled.
	Check(t, func(t *T) {
		ctx := t.Context()
		t.Cleanup(func() {
			// ctx is already cleared on rapid.T by now,
			// so this will request a new context.
			newCtx := t.Context()
			if ctx == newCtx {
				t.Fatalf("expected new context")
			}

			if err := newCtx.Err(); err == nil || !errors.Is(err, context.Canceled) {
				t.Fatalf("expected context to be canceled, got: %v", newCtx)
			}
		})
	})
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
