// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package bitstream

import (
	"math"
	"testing"

	"go.sdls.io/rapid/internal/stattest"
)

func TestFloatConversionRoundtrip(t *testing.T) {
	t.Parallel()

	s := NewRandom(stattest.Seed(t), false)
	for range stattest.Draws {
		f := math.Float32frombits(uint32(s.DrawBits(32)))
		if math.IsNaN(float64(f)) {
			continue // we can get NaNs with different bit patterns back
		}
		g := float32(float64(f))
		if g != f {
			t.Fatalf("got %v (0x%x) back from %v (0x%x)", g, math.Float32bits(g), f, math.Float32bits(f))
		}
	}
}

func TestUfloat32FromParts(t *testing.T) {
	t.Parallel()

	s := NewRandom(stattest.Seed(t), false)
	values := []float32{0, math.SmallestNonzeroFloat32, 1, math.MaxFloat32, float32(math.Inf(1))}
	for range stattest.Draws {
		values = append(values, math.Float32frombits(uint32(s.DrawBits(31)))) // sign bit clear
	}
	for _, f := range values {
		if math.IsNaN(float64(f)) {
			continue
		}
		g := ufloat32FromParts(ufloat32Parts(f))
		if g != f {
			t.Fatalf("got %v (0x%x) back from %v (0x%x)", g, math.Float32bits(g), f, math.Float32bits(f))
		}
	}
}

func TestUfloat64FromParts(t *testing.T) {
	t.Parallel()

	s := NewRandom(stattest.Seed(t), false)
	values := []float64{0, math.SmallestNonzeroFloat64, 1, math.MaxFloat64, math.Inf(1)}
	for range stattest.Draws {
		values = append(values, math.Float64frombits(s.DrawBits(63))) // sign bit clear
	}
	for _, f := range values {
		if math.IsNaN(f) {
			continue
		}
		g := ufloat64FromParts(ufloat64Parts(f))
		if g != f {
			t.Fatalf("got %v (0x%x) back from %v (0x%x)", g, math.Float64bits(g), f, math.Float64bits(f))
		}
	}
}
