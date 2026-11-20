// Copyright (c) 2026 SUDOLESS SRL
// SPDX-License-Identifier: BSD-3-Clause-Clear

package engine

import "time"

// Limits bound how long Search looks for a failure, and how long it shrinks one.
type Limits struct {
	Checks     int           // valid test cases to generate
	Deadline   time.Time     // when the test has to end; shrinking stops early enough to end before it
	SearchTime time.Duration // how long to generate test cases for; 0 for no bound but Deadline
	ShrinkTime time.Duration // how long to shrink a failure for
}

// limits returns lim with its zero fields taken from the -rapid.* flags and the deadline of tb. When short,
// the number of checks is cut to a fifth, though not below one.
func limits(tb TB, lim Limits, short bool) Limits {
	if lim.Checks == 0 {
		lim.Checks = Flags.Checks
	}
	if short && lim.Checks > 0 {
		lim.Checks = max(lim.Checks/5, 1)
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
