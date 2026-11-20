// Copyright (c) 2026 SUDOLESS SRL
// SPDX-License-Identifier: BSD-3-Clause-Clear

package rapid_test

import (
	"strings"
	"testing"

	. "go.sdls.io/rapid"
)

// checkQuiet runs prop through Check on a TB that records its outcome, and fails t if prop failed.
func checkQuiet(t *testing.T, prop func(*T), opts ...CheckOption) *quietTB {
	t.Helper()

	tb := &quietTB{name: t.Name()}
	Check(tb, prop, opts...)
	if tb.failed {
		t.Fatalf("Check failed: %v", tb.msg)
	}
	return tb
}

// A passing Check logs, after its OK line, each event with how many valid test cases marked it, the most
// marked first; a test case marking an event twice counts once. Without events, there is no table.
func TestEventTable(t *testing.T) {
	t.Parallel()

	runs := 0
	tb := checkQuiet(t, func(t *T) {
		runs++
		t.Event("all")
		t.Event("all")
		if runs%2 == 0 {
			t.Event("half")
		}
	}, Checks(10))

	want := "\n    events, of 10 valid tests:\n    10  100.0%  all\n     5   50.0%  half\n"
	if testing.Short() {
		want = "\n    events, of 2 valid tests:\n    2  100.0%  all\n    1   50.0%  half\n"
	}
	if _, table, _ := strings.Cut(tb.log.String(), ")"); table != want {
		t.Fatalf("Check logged\n%v\nwant the table\n%v", tb.log.String(), want)
	}

	tb = checkQuiet(t, func(*T) {}, Checks(10))
	if strings.Contains(tb.log.String(), "events") {
		t.Fatalf("Check of a property without events logged a table:\n%v", tb.log.String())
	}
}

// Only valid test cases count: one that marks an event and then skips does not.
func TestEventCountsValidTestCases(t *testing.T) {
	t.Parallel()

	runs := 0
	tb := checkQuiet(t, func(t *T) {
		runs++
		t.Event("marked")
		if runs%2 == 0 {
			t.SkipNow()
		}
	}, Checks(10))

	if !strings.Contains(tb.log.String(), "100.0%  marked") {
		t.Fatalf("Check logged\n%v\nwant every valid test case, and no other, counted for marked", tb.log.String())
	}
}

// A value that is discarded takes the events it marked with it.
func TestEventDiscardedValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		prop func(t *T, discard func())
	}{
		{"Custom try that skips", func(t *T, discard func()) {
			tries := 0
			Custom(func(t *T) int {
				Bool().Draw(t, "b")
				tries++
				if tries == 1 {
					discard()
					t.SkipNow()
				}
				return 0
			}).Draw(t, "v")
		}},
		{"value Filter rejects", func(t *T, discard func()) {
			calls := 0
			Custom(func(t *T) int {
				Bool().Draw(t, "b")
				calls++
				if calls == 1 {
					discard()
					return 1
				}
				return 0
			}).Filter(func(v int) bool { return v == 0 }).Draw(t, "v")
		}},
		{"duplicate in SliceOfDistinct", func(t *T, discard func()) {
			calls := 0
			SliceOfNDistinct(Custom(func(t *T) int {
				Bool().Draw(t, "b")
				calls++
				if calls == 2 {
					discard()
					return 1
				}
				return calls
			}), 3, 3).Draw(t, "s")
		}},
		{"duplicate in MapOfValues", func(t *T, discard func()) {
			calls := 0
			MapOfNValues(Custom(func(t *T) int {
				Bool().Draw(t, "b")
				calls++
				if calls == 2 {
					discard()
					return 1
				}
				return calls
			}), 3, 3, func(v int) int { return v }).Draw(t, "m")
		}},
		{"rune StringOfN rejects", func(t *T, discard func()) {
			calls := 0
			StringOfN(Custom(func(t *T) rune {
				Bool().Draw(t, "b")
				calls++
				if calls == 2 {
					discard()
					return -1
				}
				return 'a'
			}), 2, 2, -1).Draw(t, "s")
		}},
		{"Repeat action that skips", func(t *T, discard func()) {
			skipped := false
			t.Repeat(map[string]func(*T){
				"skip": func(t *T) {
					if !skipped {
						skipped = true
						discard()
						t.SkipNow()
					}
				},
				"": func(*T) {},
			})
		}},
		{"Repeat action that draws, then skips", func(t *T, discard func()) {
			skipped := false
			t.Repeat(map[string]func(*T){
				"reject": func(t *T) {
					Bool().Draw(t, "b")
					if !skipped {
						skipped = true
						discard()
						t.SkipNow()
					}
				},
				"": func(*T) {},
			})
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			discards := 0
			tb := checkQuiet(t, func(t *T) {
				tc.prop(t, func() {
					discards++
					t.Event("discarded")
				})
				t.Event("kept")
			}, Checks(20))

			log := tb.log.String()
			if discards == 0 || !strings.Contains(log, "kept") {
				t.Fatalf("%v discards; want some, and a table of events in the log:\n%v", discards, log)
			}
			if strings.Contains(log, "discarded") {
				t.Fatalf("a discarded value's event was counted:\n%v", log)
			}
		})
	}
}
