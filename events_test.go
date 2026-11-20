// Copyright (c) 2026 SUDOLESS SRL
// SPDX-License-Identifier: BSD-3-Clause-Clear

package rapid

import (
	"bytes"
	"log"
	"slices"
	"strings"
	"testing"

	"go.sdls.io/rapid/internal/bitstream"
)

// Event logs each mark with the test case's output, so the replay of a failure shows it, and a run returns
// the events its test case marked, each once.
func TestEventLogs(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	run := runner(t, func(t *T) {
		t.Event("first")
		t.Event("second")
		t.Event("first")
	})
	err, events := run(bitstream.NewBuffer(nil, false), false, log.New(&out, "", 0))

	if err != nil {
		t.Fatalf("the run failed: %v", err)
	}
	if want := []string{"first", "second"}; !slices.Equal(events, want) {
		t.Fatalf("the run returned the events %q, want %q", events, want)
	}
	if want := "[rapid] event first\n[rapid] event second\n[rapid] event first\n"; out.String() != want {
		t.Fatalf("the run logged\n%v\nwant\n%v", out.String(), strings.TrimSpace(want))
	}
}
