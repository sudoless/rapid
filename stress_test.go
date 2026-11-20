// Copyright (c) 2026 SUDOLESS SRL
// SPDX-License-Identifier: BSD-3-Clause-Clear

//go:build stress

package rapid

import "testing"

// TestStress runs the statistical and coverage checks, which the default suite
// runs on stattest.FixedSeed, on one random seed per run. `make test-stress` repeats
// it; a failing run logs the seed that replays it with -rapid.seed.
func TestStress(t *testing.T) {
	t.Parallel()

	seed := baseSeed()
	t.Logf("seed %v, replay with -rapid.seed=%v", seed, seed)

	checks := map[string]func(*testing.T, uint64){
		"GenGeomMean":    checkGenGeomMean,
		"GenUintN":       checkGenUintN,
		"GenUintRange":   checkGenUintRange,
		"GenIntRange":    checkGenIntRange,
		"FlipBiasedCoin": checkFlipBiasedCoin,
		"LoadedDie":      checkLoadedDie,
		"Repeat":         checkRepeat,
		"ByteCoverage":   checkByteCoverage,
		"IntCoverage":    checkIntCoverage,
		"UintCoverage":   checkUintCoverage,
	}
	for name, check := range checks {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			check(t, seed)
		})
	}
}
