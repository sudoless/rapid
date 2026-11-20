// Copyright (c) 2026 SUDOLESS SRL
// SPDX-License-Identifier: BSD-3-Clause-Clear

package rapid_test

import (
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	. "go.sdls.io/rapid"
)

// short returns n as -short cuts it: to a fifth, though not below one.
func short(n int) int {
	if testing.Short() {
		return max(n/5, 1)
	}
	return n
}

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

// Events marked inside a SyncTest bubble, or from other goroutines, are the test case's.
func TestEventScopes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		prop func(*T)
	}{
		{"SyncTest", func(t *T) { t.SyncTest(func(t *T) { t.Event("marked") }) }},
		{"goroutines", func(t *T) {
			var wg sync.WaitGroup
			for range 2 {
				wg.Go(func() { t.Event("marked") })
			}
			wg.Wait()
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			Check(t, tc.prop, Checks(10), Cover("marked", 10))
		})
	}
}

// Cover runs the property past its checks until enough valid test cases marked each covered event, and up
// to ten times its checks or its largest n; -short cuts the counts to a fifth.
func TestCover(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts []CheckOption
		late int // the run that marks "late", if any
		want int // runs
	}{
		{"met by the checks", []CheckOption{Checks(10), Cover("always", 3)}, 0, short(10)},
		{"past the checks", []CheckOption{Checks(10), Cover("always", 30)}, 0, short(30)},
		{"until a late event", []CheckOption{Checks(10), Cover("late", 1)}, 15, 15},
		{"up to ten times the checks", []CheckOption{Checks(9), Cover("always", 90)}, 0, short(90)},
		{"n above ten times the checks", []CheckOption{Checks(5), Cover("always", 60)}, 0, short(60)},
		{"each event on its own", []CheckOption{Checks(10), Cover("always", 40), Cover("late", 1)}, 5, short(40)},
		{"the last option for an event wins", []CheckOption{Checks(10), Cover("always", 40), Cover("always", 20)}, 0, short(20)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			runs := 0
			tb := checkQuiet(t, func(t *T) {
				runs++
				t.Event("always")
				if runs == tc.late {
					t.Event("late")
				}
			}, tc.opts...)
			if runs != tc.want {
				t.Fatalf("the property ran %v times, want %v; Check logged\n%v", runs, tc.want, tb.log.String())
			}
		})
	}
}

// A Cover not met fails the test, saying what stopped the search, with the table of events.
func TestCoverNotMet(t *testing.T) {
	t.Parallel()

	sleep := func(*T) { time.Sleep(time.Millisecond) }
	tests := []struct {
		name string
		prop func(*T)
		opts []CheckOption
		want string
	}{
		{"ten times the checks", func(*T) {}, []CheckOption{Checks(10)}, "Cover not met after " + strconv.Itoa(short(100)) + " valid tests"},
		{"time limit", sleep, []CheckOption{Checks(1_000_000), SearchTime(50 * time.Millisecond)}, "the search reached its time limit"},
		{"invalid test cases", func() func(*T) {
			runs := 0
			return func(t *T) {
				runs++
				if runs > 2 {
					t.SkipNow()
				}
			}
		}(), []CheckOption{Checks(2)}, "the search stopped at " + strconv.Itoa(short(2)*10) + " invalid tests, after 2 valid ones"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tb := &quietTB{name: t.Name()}
			Check(tb, tc.prop, append(tc.opts, Cover("never", 1))...)
			if !tb.failed {
				t.Fatalf("Check passed with Cover unmet; it logged\n%v", tb.log.String())
			}
			if !strings.Contains(tb.msg, tc.want) || !strings.Contains(tb.msg, "0.0%  never (cover 1)") {
				t.Fatalf("Check failed with\n%v\nwant %q and the table of events", tb.msg, tc.want)
			}
		})
	}
}

// A search the time limit stops passes when its Cover is met.
func TestCoverMetBeforeTimeLimit(t *testing.T) {
	t.Parallel()

	checkQuiet(t, func(t *T) {
		t.Event("always")
		time.Sleep(time.Millisecond)
	}, Checks(1_000_000), SearchTime(50*time.Millisecond), Cover("always", 1))
}
