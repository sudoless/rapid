// Copyright (c) 2026 SUDOLESS SRL
// SPDX-License-Identifier: BSD-3-Clause-Clear

//go:build stress

package rapid

import (
	"testing"

	"go.sdls.io/rapid/internal/stattest"
)

// TestStress runs the coverage checks, which the default suite runs on
// stattest.FixedSeed, on one random seed per run. `make test-stress` repeats it;
// a failing run logs the seed, and RAPID_SEED set to it replays the run.
func TestStress(t *testing.T) {
	t.Parallel()

	seed := stattest.Seed(t)

	checks := map[string]func(*testing.T, uint64){
		"ByteCoverage": checkByteCoverage,
		"IntCoverage":  checkIntCoverage,
		"UintCoverage": checkUintCoverage,
	}
	for name, check := range checks {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			check(t, seed)
		})
	}
}
