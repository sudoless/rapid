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

		rapid.SyncTest(rt, func(inner *rapid.T) {
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
