// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package stattest holds the bounds rapid's statistical and coverage tests share.
package stattest

import (
	"math"
	"testing"
)

// FixedSeed seeds the statistical checks in the default suite; TestStress
// (build tag stress) runs the same checks on random seeds.
const FixedSeed = 1

// Each statistical check draws Draws values and allows Sigmas standard
// errors, so a correct generator fails one bound with probability below 1e-6.
const (
	Draws  = 10000
	Sigmas = 5
)

// CoverageDraws bounds how many values a coverage check draws before it gives up.
const CoverageDraws = 100000

// WithinSigmas fails t when y hits out of Draws trials, each a hit with probability p,
// stray more than Sigmas standard deviations from the expected count; ctx, if not nil, names the case.
func WithinSigmas(t *testing.T, ctx any, y int, p float64) {
	t.Helper()

	n := Draws
	mu := float64(n) * p
	s := math.Sqrt(float64(n) * p * (1 - p))

	if float64(y) < mu-Sigmas*s || float64(y) > mu+Sigmas*s {
		if ctx != nil {
			t.Errorf("for %v: got %v out of %v (p %v, mu %v, stddev %v)", ctx, y, n, p, mu, s)
		} else {
			t.Errorf("got %v out of %v (p %v, mu %v, stddev %v)", y, n, p, mu, s)
		}
	}
}
