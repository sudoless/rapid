// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid_test

import (
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"go.sdls.io/rapid"
)

func TestSyncTest(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		var cleaned atomic.Bool

		rt.SyncTest(func(inner *rapid.T) {
			inner.Cleanup(func() {
				cleaned.Store(true)
			})

			const sleep = 2 * time.Second
			start := time.Now()
			time.Sleep(sleep)
			if got := time.Since(start); got != sleep {
				inner.Fatalf("virtual time advanced by %v, want %v", got, sleep)
			}

			done := make(chan struct{})
			go func() { close(done) }()
			synctest.Wait()
			select {
			case <-done:
			default:
				inner.Fatalf("goroutine did not finish inside synctest bubble")
			}
		})

		if !cleaned.Load() {
			rt.Fatalf("cleanup registered inside SyncTest did not run")
		}
	})
}

// Context and Cleanup are safe to call from another goroutine while the property runs a SyncTest bubble.
func TestSyncTestWhileAnotherGoroutineUsesT(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		done := make(chan struct{})
		go func() {
			defer close(done)
			for range 100 {
				_ = rt.Context()
				rt.Cleanup(func() {})
			}
		}()
		rt.SyncTest(func(*rapid.T) {})
		<-done
	})
}

// A Repeat action whose only draws happen inside SyncTest has drawn: skipping it afterwards rejects
// the action, rather than marking it not applicable until Repeat finds no valid action.
func TestSyncTestDrawsCountForRepeat(t *testing.T) {
	t.Parallel()

	_, _, failed := rapid.CheckOutcome(t, func(rt *rapid.T) {
		rt.Repeat(map[string]func(*rapid.T){
			"DrawThenSkip": func(rt *rapid.T) {
				rt.SyncTest(func(bt *rapid.T) { _ = rapid.IntRange(0, 9).Draw(bt, "v") })
				rt.SkipNow()
			},
		})
	})
	if failed {
		t.Fatal("an action that drew inside SyncTest and then skipped counted as not applicable")
	}
}

// A failure recorded inside SyncTest fails the test case that ran the bubble.
func TestSyncTestRecordedFailureFailsTestCase(t *testing.T) {
	t.Parallel()

	_, _, failed := rapid.CheckOutcome(t, func(rt *rapid.T) {
		rt.SyncTest(func(bt *rapid.T) { bt.Errorf("bad") })
	})
	if !failed {
		t.Fatal("Errorf inside SyncTest did not fail the test case")
	}
}

// Check runs inside a testing/synctest bubble, where time is fake: a property that sleeps an hour per test
// case finishes at once.
func TestCheckInsideSynctestBubble(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		runs := 0
		rapid.Check(t, func(*rapid.T) {
			runs++
			time.Sleep(time.Hour)
		}, rapid.Checks(10))

		want := 10
		if testing.Short() {
			want = 2
		}
		if runs != want {
			t.Fatalf("the property ran %v times, want %v", runs, want)
		}
	})
}
