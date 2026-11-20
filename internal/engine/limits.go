// Copyright (c) 2026 SUDOLESS SRL
// SPDX-License-Identifier: BSD-3-Clause-Clear

package engine

import (
	"maps"
	"slices"
	"time"
)

// Limits bound how long Search looks for a failure, and how long it shrinks one.
type Limits struct {
	Checks      int            // valid test cases to generate
	Cover       map[string]int // for each event, the valid test cases that have to mark it
	CoverChecks int            // valid test cases to generate at most while Cover is unmet
	Deadline    time.Time      // when the test has to end; shrinking stops early enough to end before it
	SearchTime  time.Duration  // how long to generate test cases for; 0 for no bound but Deadline
	ShrinkTime  time.Duration  // how long to shrink a failure for
}

// limits returns lim with its zero fields taken from the -rapid.* flags and the deadline of tb, and
// CoverChecks a multiple of Checks, or of the most that Cover asks for, when lim has a Cover. When short, the counts of test cases, Cover's
// included, are cut to a fifth, though not below one.
func limits(tb TB, lim Limits, short bool) Limits {
	if lim.Checks == 0 {
		lim.Checks = Flags.Checks
	}
	if len(lim.Cover) > 0 && lim.CoverChecks == 0 {
		lim.CoverChecks = max(lim.Checks, slices.Max(slices.Collect(maps.Values(lim.Cover)))) * coverChecksMult
	}
	if short && lim.Checks > 0 {
		fifth := func(n int) int { return max(n/5, 1) }
		lim.Checks = fifth(lim.Checks)
		if lim.CoverChecks > 0 {
			lim.CoverChecks = fifth(lim.CoverChecks)
		}
		if len(lim.Cover) > 0 {
			cover := make(map[string]int, len(lim.Cover)) // lim.Cover may be shared with other tests
			for event, n := range lim.Cover {
				cover[event] = fifth(n)
			}
			lim.Cover = cover
		}
	}
	if lim.Deadline.IsZero() {
		lim.Deadline = Deadline(tb)
	}
	if lim.ShrinkTime == 0 {
		lim.ShrinkTime = Flags.ShrinkTime
	}
	return lim
}

// searchDeadline returns when to stop generating test cases, for a search started at start.
func (lim *Limits) searchDeadline(start time.Time) time.Time {
	if lim.SearchTime > 0 {
		if d := start.Add(lim.SearchTime); d.Before(lim.Deadline) {
			return d
		}
	}
	return lim.Deadline
}

// shrinkDeadline returns when to stop shrinking a failure found now: after ShrinkTime, or early enough that
// the last shrink step, checked only before it starts, ends before Deadline.
func (lim *Limits) shrinkDeadline() time.Time {
	d := time.Now().Add(lim.ShrinkTime)
	if latest := lim.Deadline.Add(-shrinkStepBound); d.After(latest) {
		d = latest
	}
	return d
}
