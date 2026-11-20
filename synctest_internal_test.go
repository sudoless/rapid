// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"strings"
	"testing"
)

func TestSyncTest_FailureIsCapturedForShrinking(t *testing.T) {
	t.Parallel()

	rt := newT(tb(t), newRandomBitStream(1, true), false, nil)

	err := checkOnce(rt, func(t *T) {
		SyncTest(t, func(inner *T) {
			inner.Fatalf("boom")
		})
	})

	if err == nil {
		t.Fatalf("checkOnce did not report failure from SyncTest")
	}
	if !err.isStopTest() {
		t.Fatalf("expected stopTest failure, got %T (%v)", err.data, err)
	}
	if !strings.Contains(errorString(err), "boom") {
		t.Fatalf("missing failure message: %q", errorString(err))
	}
	if !strings.Contains(traceback(err), "synctest_internal_test.go") {
		t.Fatalf("traceback does not include property call site:\n%v", traceback(err))
	}
}
