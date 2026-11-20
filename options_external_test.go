// Copyright (c) 2026 SUDOLESS SRL
// SPDX-License-Identifier: BSD-3-Clause-Clear

package rapid_test

import (
	"testing"
	"time"

	. "go.sdls.io/rapid"
)

// Checks sets how many test cases the property runs, the last option winning, through Check and MakeCheck
// alike; -short runs a fifth of them, and at least one.
func TestChecks(t *testing.T) {
	t.Parallel()

	short := func(n int) int {
		if testing.Short() {
			return max(n/5, 1)
		}
		return n
	}
	tests := []struct {
		name string
		opts []CheckOption
		want int
	}{
		{"Checks", []CheckOption{Checks(7)}, short(7)},
		{"last wins", []CheckOption{Checks(50), Checks(12)}, short(12)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			runs := 0
			Check(t, func(*T) { runs++ }, tc.opts...)
			if runs != tc.want {
				t.Errorf("Check ran the property %v times, want %v", runs, tc.want)
			}

			runs = 0
			t.Run("MakeCheck", MakeCheck(func(*T) { runs++ }, tc.opts...))
			if runs != tc.want {
				t.Errorf("MakeCheck ran the property %v times, want %v", runs, tc.want)
			}
		})
	}
}

// SearchTime stops an expensive property after the time given, and the property passes on the test cases
// that ran.
func TestSearchTime(t *testing.T) {
	t.Parallel()

	runs := 0
	Check(t, func(*T) {
		runs++
		time.Sleep(time.Millisecond)
	}, Checks(1_000_000), SearchTime(100*time.Millisecond))

	if runs < 1 || runs > 100 {
		t.Fatalf("the property ran %v times in a 100ms search of test cases taking at least 1ms", runs)
	}
}

// An option that cannot be met panics where it is made, as a generator with an empty range does.
func TestCheckOptionPanics(t *testing.T) {
	t.Parallel()

	for name, opt := range map[string]func(){
		"Checks(0)":       func() { Checks(0) },
		"SearchTime(0)":   func() { SearchTime(0) },
		"ShrinkTime(-1s)": func() { ShrinkTime(-time.Second) },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				if recover() == nil {
					t.Fatalf("%v did not panic", name)
				}
			}()
			opt()
		})
	}
}
