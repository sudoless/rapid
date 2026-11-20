// Copyright (c) 2026 SUDOLESS SRL
// SPDX-License-Identifier: BSD-3-Clause-Clear

package rapid

import (
	"slices"
)

// Event marks the current test case with name. Once the property passes, [Check] logs a table of the events
// and how many valid test cases marked each.
//
// A test case counts once for each name it marks, however often it marks it, and only when it is valid.
// A value that is discarded takes its marks with it: a [Custom] try that skips, a value [*Generator.Filter]
// rejects, an element a distinct collection rejects as a duplicate, a [*T.Repeat] action that skips.
// Event logs the mark with the test case's draws. Under [Fuzz], it only logs.
func (t *T) Event(name string) {
	if t.tbLog {
		t.Helper()
	}

	t.tc.mark(name)

	if t.tbLog || t.rawLog != nil {
		if t.tbLog {
			t.Helper()
		}
		t.Logf("[rapid] event %v", name)
	}
}

// mark adds name to the events of tc, unless it is there already.
func (tc *testCase) mark(name string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if !slices.Contains(tc.events, name) {
		tc.events = append(tc.events, name)
		tc.nEvents.Store(int64(len(tc.events)))
	}
}

// eventMark returns where the events of tc end now, for unmarkSince to drop the ones marked after.
func (tc *testCase) eventMark() int {
	return int(tc.nEvents.Load())
}

// unmarkSince drops the events of tc marked after mark, as the scope that marked them was discarded.
func (tc *testCase) unmarkSince(mark int) {
	if tc.eventMark() <= mark {
		return
	}

	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.events = tc.events[:mark]
	tc.nEvents.Store(int64(mark))
}

// marked returns the events of tc.
func (tc *testCase) marked() []string {
	if tc.eventMark() == 0 {
		return nil
	}

	tc.mu.Lock()
	defer tc.mu.Unlock()

	return slices.Clone(tc.events)
}
