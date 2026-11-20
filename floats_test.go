// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"testing"

	"go.sdls.io/rapid/internal/stattest"
)

// FloatRange produces both bounds of a range, and zero when the range holds it, on ranges drawn from the
// fixed seed.
func TestFloatBoundCoverage(t *testing.T) {
	t.Parallel()

	checks := map[string]func(*testing.T, uint64){
		"float32": checkFloat32BoundCoverage,
		"float64": checkFloat64BoundCoverage,
	}
	for name, check := range checks {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			check(t, stattest.FixedSeed)
		})
	}
}

func checkFloat32BoundCoverage(t *testing.T, seed uint64) {
	t.Helper()
	checkBoundCoverage(t, seed, Float[float32](), FloatRange[float32], 400)
}

func checkFloat64BoundCoverage(t *testing.T, seed uint64) {
	t.Helper()
	checkBoundCoverage(t, seed, Float[float64](), FloatRange[float64], 400)
}
