// Copyright 2022 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	. "go.sdls.io/rapid"
)

func FuzzContext(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})

	type key struct{}

	var ctx context.Context
	Fuzz(f, func(t *T) {
		// Assign to outer variable
		// so we can check it after the fuzzing.
		ctx = context.WithValue(t.Context(), key{}, "value")
		if err := ctx.Err(); err != nil {
			t.Fatalf("context must be valid: %v", err)
		}
	})

	// ctx is set only if the fuzzing function was called.
	if ctx != nil {
		if err := ctx.Err(); err == nil {
			f.Fatalf("context must be canceled")
		}

		if want, got := "value", ctx.Value(key{}); want != got {
			f.Fatalf("context must have value %q, got %q", want, got)
		}
	}
}

func FuzzCleanup(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})

	var state []bool
	Fuzz(f, func(t *T) {
		idx := len(state)
		state = append(state, false)
		t.Cleanup(func() {
			state[idx] = true
		})
	})

	for _, ok := range state {
		if !ok {
			f.Fatalf("cleanup must be called")
		}
	}
}

// envChild names, in a test process TestMalformedEnv starts, what TestMalformedEnv does there instead.
const envChild = "RAPIDTEST_ENV_CHILD"

// A RAPID_* variable that does not parse fails whatever reads the settings, unless its -rapid.* flag is set.
// The variables are read once per process, so each case runs this test binary again.
func TestMalformedEnv(t *testing.T) {
	switch os.Getenv(envChild) {
	case "check":
		Check(t, func(t *T) { _ = Int[int]().Draw(t, "x") })
		return
	case "example":
		_ = Int[int]().Example()
		return
	}
	t.Parallel()

	tests := []struct {
		name   string
		key    string // set to a value that does not parse
		child  string // what TestMalformedEnv does in the child; empty to run the test args name instead
		args   []string
		failed bool
		absent string // in the output: Fuzz fails before any input runs, so no input such as seed#0 fails
	}{
		{"Check", "RAPID_CHECKS", "check", nil, true, ""},
		{"flag overrides", "RAPID_CHECKS", "check", []string{"-rapid.checks=5"}, false, ""},
		{"Fuzz", "RAPID_V", "", []string{"-test.run=^FuzzCleanup$", "-test.v"}, true, "seed#"},
		{"Example", "RAPID_SEED", "example", nil, true, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			args := tc.args
			if tc.child != "" {
				args = append([]string{"-test.run=^TestMalformedEnv$"}, args...)
			}
			cmd := exec.CommandContext(t.Context(), os.Args[0], args...) //nolint:gosec // this test binary, with the case's args
			for _, kv := range os.Environ() {
				if !strings.HasPrefix(kv, "RAPID_") && !strings.HasPrefix(kv, envChild+"=") {
					cmd.Env = append(cmd.Env, kv)
				}
			}
			cmd.Env = append(cmd.Env, tc.key+"=malformed", envChild+"="+tc.child)

			out, err := cmd.CombinedOutput()
			switch {
			case tc.failed && err == nil:
				t.Fatalf("passed with a malformed %v; want it to fail naming it\n%s", tc.key, out)
			case tc.failed && !strings.Contains(string(out), tc.key):
				t.Fatalf("failed with a malformed %v without naming it\n%s", tc.key, out)
			case !tc.failed && err != nil:
				t.Fatalf("failed with a malformed %v: %v\n%s", tc.key, err, out)
			case tc.absent != "" && strings.Contains(string(out), tc.absent):
				t.Fatalf("with a malformed %v, the output holds %q\n%s", tc.key, tc.absent, out)
			}
		})
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

// A property that panics fails, whatever its cleanups do: a cleanup's skip does not hide the panic.
func TestCheckPanicNotMaskedByCleanupSkip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		prop func(*T)
	}{
		{"property", func(t *T) {
			t.Cleanup(func() { t.SkipNow() })
			panic("boom")
		}},
		{"SyncTest bubble", func(t *T) {
			t.SyncTest(func(t *T) {
				t.Cleanup(func() { t.SkipNow() })
				panic("boom")
			})
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if valid, invalid, failed := CheckOutcome(t, tc.prop); !failed {
				t.Fatalf("panic not reported (valid %d, invalid %d)", valid, invalid)
			}
		})
	}
}
