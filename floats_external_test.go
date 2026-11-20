// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid_test

import (
	"math"
	"testing"

	. "go.sdls.io/rapid"
)

func TestFloatNoInf(t *testing.T) {
	t.Parallel()

	gens := []*Generator[any]{
		Float32().AsAny(),
		Float32Min(0).AsAny(),
		Float32Max(0).AsAny(),
		Float64().AsAny(),
		Float64Min(0).AsAny(),
		Float64Max(0).AsAny(),
	}

	for _, gen := range gens {
		t.Run(gen.String(), func(t *testing.T) {
			t.Parallel()

			Check(t, func(t *T) {
				f := gen.Draw(t, "f")
				if math.IsInf(rv(f).Float(), 0) {
					t.Fatalf("got infinity: %v", f)
				}
			})
		})
	}
}

func TestFloat32BoundCoverage(t *testing.T) {
	t.Parallel()

	Check(t, func(t *T) {
		lo := Float32().Draw(t, "min")
		hi := Float32().Draw(t, "max")
		if lo > hi {
			lo, hi = hi, lo
		}

		g := Float32Range(lo, hi)
		var gotMin, gotMax, gotZero bool
		for seed := range 400 {
			f := g.Example(seed)

			gotMin = gotMin || f == lo
			gotMax = gotMax || f == hi
			gotZero = gotZero || f == 0

			if gotMin && gotMax && (lo > 0 || hi < 0 || gotZero) {
				return
			}
		}

		t.Fatalf("[%v, %v]: got min %v, got max %v, got zero %v", lo, hi, gotMin, gotMax, gotZero)
	})
}

func TestFloat64BoundCoverage(t *testing.T) {
	t.Parallel()

	Check(t, func(t *T) {
		lo := Float64().Draw(t, "min")
		hi := Float64().Draw(t, "max")
		if lo > hi {
			lo, hi = hi, lo
		}

		g := Float64Range(lo, hi)
		var gotMin, gotMax, gotZero bool
		for seed := range 400 {
			f := g.Example(seed)

			gotMin = gotMin || f == lo
			gotMax = gotMax || f == hi
			gotZero = gotZero || f == 0

			if gotMin && gotMax && (lo > 0 || hi < 0 || gotZero) {
				return
			}
		}

		t.Fatalf("[%v, %v]: got min %v, got max %v, got zero %v", lo, hi, gotMin, gotMax, gotZero)
	})
}
