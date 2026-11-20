// Copyright (c) 2026 SUDOLESS SRL
// SPDX-License-Identifier: BSD-3-Clause-Clear

package engine

import (
	"log"
	"os"
	"testing"

	"go.sdls.io/rapid/internal/bitstream"
)

// Under -rapid.log, every run Search makes is given a logger, so the property's output goes where Search's
// progress lines go.
func TestSearchLogGivesRunsALogger(t *testing.T) { //nolint:paralleltest // sets Flags.Log and os.Stdout
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	flagLog, stdout := Flags.Log, os.Stdout
	Flags.Log, os.Stdout = true, devNull
	t.Cleanup(func() {
		Flags.Log, os.Stdout = flagLog, stdout
		_ = devNull.Close()
	})

	runs, logged := 0, 0
	Search(t, Limits{Checks: 3, Deadline: Deadline(nil), ShrinkTime: Flags.ShrinkTime}, 1, "", false, func(_ bitstream.Stream, _ bool, out *log.Logger) (*TestError, []string) {
		runs++
		if out != nil {
			logged++
		}
		return nil, nil
	})

	if runs == 0 || logged != runs {
		t.Fatalf("%v of %v runs had a logger", logged, runs)
	}
}
