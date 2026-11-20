// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"strings"
	"testing"

	"go.sdls.io/rapid/internal/bitstream"
	"go.sdls.io/rapid/internal/engine"
	"go.sdls.io/rapid/internal/stattest"
)

func TestSyncTest_FailureIsCapturedForShrinking(t *testing.T) {
	t.Parallel()

	rt := newT(tb(t), bitstream.NewRandom(1, true), false, nil)

	err := checkOnce(rt, func(t *T) {
		t.SyncTest(func(inner *T) {
			inner.Fatalf("boom")
		})
	})

	if err == nil {
		t.Fatalf("checkOnce did not report failure from SyncTest")
	}
	if !err.IsStopTest() {
		t.Fatalf("expected stopTest failure, got %T (%v)", err.Data, err)
	}
	if !strings.Contains(engine.ErrorString(err), "boom") {
		t.Fatalf("missing failure message: %q", engine.ErrorString(err))
	}
	if !strings.Contains(engine.Traceback(err), "synctest_test.go") {
		t.Fatalf("traceback does not include property call site:\n%v", engine.Traceback(err))
	}
}

// A skip inside SyncTest skips what the T running the bubble would skip: a Repeat action it marks not
// applicable, and a Custom try it rejects, rather than discarding the whole test case.
func TestSyncTestSkipScopedLikeItsT(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		prop func(*T)
	}{
		{"Repeat action", func(t *T) {
			t.Repeat(map[string]func(*T){
				"Skip": func(t *T) { t.SyncTest(func(bt *T) { bt.SkipNow() }) },
				"Ok":   func(*T) {},
			})
		}},
		{"Custom try", func(t *T) {
			Custom(func(t *T) int {
				x := IntRange(0, 9).Draw(t, "x")
				if x == 0 {
					t.SyncTest(func(bt *T) { bt.SkipNow() })
				}
				return x
			}).Draw(t, "v")
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res := search(t, stattest.FixedSeed, tc.prop)
			if res.Err1 != nil || res.Invalid != 0 {
				t.Fatalf("got %d valid and %d invalid test cases, failure %v; want 100 valid", res.Valid, res.Invalid, res.Err1)
			}
		})
	}
}
