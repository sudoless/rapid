// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"strings"
	"testing"

	"go.sdls.io/rapid/internal/bitstream"
	"go.sdls.io/rapid/internal/engine"
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
	if !strings.Contains(engine.Traceback(err), "synctest_internal_test.go") {
		t.Fatalf("traceback does not include property call site:\n%v", engine.Traceback(err))
	}
}
