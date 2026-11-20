// Copyright (c) 2026 SUDOLESS SRL
// SPDX-License-Identifier: BSD-3-Clause-Clear

package engine

import (
	"log"
	"maps"
	"reflect"
	"testing"
	"testing/synctest"
	"time"

	"go.sdls.io/rapid/internal/bitstream"
)

// A limit a test sets wins over its -rapid.* flag, a limit it leaves zero takes the flag, and -short cuts the
// counts of test cases to a fifth without reaching zero, leaving the Cover it was given as it was.
func TestLimits(t *testing.T) {
	t.Parallel()

	deadline := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		lim   Limits
		short bool
		want  Limits
	}{
		{"zero takes the flags", Limits{Deadline: deadline}, false, Limits{Checks: Flags.Checks, Deadline: deadline, ShrinkTime: Flags.ShrinkTime}},
		{"set wins", Limits{Checks: 20, Deadline: deadline, SearchTime: time.Minute, ShrinkTime: time.Second}, false, Limits{Checks: 20, Deadline: deadline, SearchTime: time.Minute, ShrinkTime: time.Second}},
		{"short cuts to a fifth", Limits{Checks: 20, Deadline: deadline}, true, Limits{Checks: 4, Deadline: deadline, ShrinkTime: Flags.ShrinkTime}},
		{"short keeps one", Limits{Checks: 4, Deadline: deadline}, true, Limits{Checks: 1, Deadline: deadline, ShrinkTime: Flags.ShrinkTime}},
		{"Cover runs up to ten times the checks", Limits{Checks: 9, Cover: map[string]int{"e": 5}, Deadline: deadline}, false, Limits{Checks: 9, Cover: map[string]int{"e": 5}, CoverChecks: 90, Deadline: deadline, ShrinkTime: Flags.ShrinkTime}},
		{"Cover runs up to ten times the most it asks for", Limits{Checks: 5, Cover: map[string]int{"e": 60, "f": 1}, Deadline: deadline}, false, Limits{Checks: 5, Cover: map[string]int{"e": 60, "f": 1}, CoverChecks: 600, Deadline: deadline, ShrinkTime: Flags.ShrinkTime}},
		{"short cuts Cover and its bound alike", Limits{Checks: 9, Cover: map[string]int{"e": 90}, Deadline: deadline}, true, Limits{Checks: 1, Cover: map[string]int{"e": 18}, CoverChecks: 180, Deadline: deadline, ShrinkTime: Flags.ShrinkTime}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cover := maps.Clone(tc.lim.Cover)
			if got := limits(t, tc.lim, tc.short); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("limits(%+v, short %v) = %+v, want %+v", tc.lim, tc.short, got, tc.want)
			}
			if !maps.Equal(tc.lim.Cover, cover) {
				t.Fatalf("limits changed the Cover it was given from %v to %v", cover, tc.lim.Cover)
			}
		})
	}
}

// The search stops after SearchTime, and the test passes on the valid test cases that ran by then.
func TestSearchTimeStopsSearch(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		lim := Limits{Checks: 1_000_000, Deadline: time.Now().Add(time.Hour), SearchTime: 50 * time.Millisecond, ShrinkTime: time.Second}
		res := Search(t, lim, 1, "", false, func(bitstream.Stream, bool, *log.Logger) (*TestError, []string) {
			time.Sleep(time.Millisecond)
			return nil, nil
		})

		if !res.EarlyExit || res.Valid < 1 || res.Valid >= lim.Checks {
			t.Fatalf("early exit %v after %v valid test cases; want an early exit after at least 1 and fewer than %v", res.EarlyExit, res.Valid, lim.Checks)
		}
	})
}

// SearchTime bounds finding a failure, not shrinking it; ShrinkTime bounds shrinking it.
func TestSearchAndShrinkTime(t *testing.T) {
	t.Parallel()

	failure := &TestError{Data: "at least 1000", Trace: "trace"}
	var run Run = func(s bitstream.Stream, _ bool, _ *log.Logger) (*TestError, []string) {
		if s.DrawBits(16) >= 1000 {
			return failure, nil
		}
		return nil, nil
	}
	tests := []struct {
		name       string
		searchTime time.Duration
		shrinkTime time.Duration
		shrunk     bool
	}{
		{"a short search still shrinks", time.Nanosecond, time.Minute, true},
		{"a short shrink stops shrinking", 0, time.Nanosecond, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			lim := Limits{Checks: 100, Deadline: time.Now().Add(time.Hour), SearchTime: tc.searchTime, ShrinkTime: tc.shrinkTime}
			res := Search(t, lim, 1, "", false, run)
			if res.Err1 == nil {
				t.Fatal("no failure found")
			}
			if shrunk := res.Buf[0] == 1000; shrunk != tc.shrunk {
				t.Fatalf("failure shrunk to %v; want shrunk to 1000 %v", res.Buf[0], tc.shrunk)
			}
		})
	}
}
