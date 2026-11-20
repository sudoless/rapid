// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"math"
	"testing"
)

func TestFloatConversionRoundtrip(t *testing.T) {
	t.Parallel()

	Check(t, func(t *T) {
		u := uint32(t.s.drawBits(32))
		f := math.Float32frombits(u)
		if math.IsNaN(float64(f)) {
			t.Skip("NaN") // we can get NaNs with different bit patterns back
		}
		g := float32(float64(f))
		if g != f {
			t.Fatalf("got %v (0x%x) back from %v (0x%x)", g, math.Float32bits(g), f, math.Float32bits(f))
		}
	})
}

func TestUfloat32FromParts(t *testing.T) {
	t.Parallel()

	Check(t, func(t *T) {
		f := Float32Min(0).Draw(t, "f")
		g := ufloat32FromParts(ufloat32Parts(f))
		if g != f {
			t.Fatalf("got %v (0x%x) back from %v (0x%x)", g, math.Float32bits(g), f, math.Float32bits(f))
		}
	})
}

func TestUfloat64FromParts(t *testing.T) {
	t.Parallel()

	Check(t, func(t *T) {
		f := Float64Min(0).Draw(t, "f")
		g := ufloat64FromParts(ufloat64Parts(f))
		if g != f {
			t.Fatalf("got %v (0x%x) back from %v (0x%x)", g, math.Float64bits(g), f, math.Float64bits(f))
		}
	})
}

func TestGenUfloat32Range(t *testing.T) {
	t.Parallel()

	Check(t, func(t *T) {
		lo := Float32Min(0).Draw(t, "min")
		hi := Float32Min(0).Draw(t, "max")
		if lo > hi {
			lo, hi = hi, lo
		}

		f := ufloat32FromParts(genUfloatRange(t.s, float64(lo), float64(hi), float32SignifBits))
		if f < lo || f > hi {
			t.Fatalf("%v (0x%x) outside of [%v, %v] ([0x%x, 0x%x])", f, math.Float32bits(f), lo, hi, math.Float32bits(lo), math.Float32bits(hi))
		}
	})
}

func TestGenUfloat64Range(t *testing.T) {
	t.Parallel()

	Check(t, func(t *T) {
		lo := Float64Min(0).Draw(t, "min")
		hi := Float64Min(0).Draw(t, "max")
		if lo > hi {
			lo, hi = hi, lo
		}

		f := ufloat64FromParts(genUfloatRange(t.s, lo, hi, float64SignifBits))
		if f < lo || f > hi {
			t.Fatalf("%v (0x%x) outside of [%v, %v] ([0x%x, 0x%x])", f, math.Float64bits(f), lo, hi, math.Float64bits(lo), math.Float64bits(hi))
		}
	})
}

func TestGenFloat32Range(t *testing.T) {
	t.Parallel()

	Check(t, func(t *T) {
		lo := Float32().Draw(t, "min")
		hi := Float32().Draw(t, "max")
		if lo > hi {
			lo, hi = hi, lo
		}

		f := float32FromParts(genFloatRange(t.s, float64(lo), float64(hi), float32SignifBits))
		if f < lo || f > hi {
			t.Fatalf("%v (0x%x) outside of [%v, %v] ([0x%x, 0x%x])", f, math.Float32bits(f), lo, hi, math.Float32bits(lo), math.Float32bits(hi))
		}
	})
}

func TestGenFloat64Range(t *testing.T) {
	t.Parallel()

	Check(t, func(t *T) {
		lo := Float64().Draw(t, "min")
		hi := Float64().Draw(t, "max")
		if lo > hi {
			lo, hi = hi, lo
		}

		f := float64FromParts(genFloatRange(t.s, lo, hi, float64SignifBits))
		if f < lo || f > hi {
			t.Fatalf("%v (0x%x) outside of [%v, %v] ([0x%x, 0x%x])", f, math.Float64bits(f), lo, hi, math.Float64bits(lo), math.Float64bits(hi))
		}
	})
}
