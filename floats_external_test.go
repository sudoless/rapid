// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid_test

import (
	"fmt"
	"math"
	"testing"

	. "go.sdls.io/rapid"
)

func TestFloatNoInf(t *testing.T) {
	t.Parallel()

	gens := []*Generator[any]{
		Float[float32]().AsAny(),
		FloatMin[float32](0).AsAny(),
		FloatMax[float32](0).AsAny(),
		Float[float64]().AsAny(),
		FloatMin[float64](0).AsAny(),
		FloatMax[float64](0).AsAny(),
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
		lo := Float[float32]().Draw(t, "min")
		hi := Float[float32]().Draw(t, "max")
		if lo > hi {
			lo, hi = hi, lo
		}

		g := FloatRange[float32](lo, hi)
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
		lo := Float[float64]().Draw(t, "min")
		hi := Float[float64]().Draw(t, "max")
		if lo > hi {
			lo, hi = hi, lo
		}

		g := FloatRange[float64](lo, hi)
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

type celsius float32

func TestFloatNamedType(t *testing.T) {
	t.Parallel()

	Check(t, func(t *T) {
		c := FloatRange[celsius](-273.15, 100).Draw(t, "c")
		if c < -273.15 || c > 100 {
			t.Fatalf("%v outside of [-273.15, 100]", c)
		}
	})
}

func TestFloatString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		gen  fmt.Stringer
		want string
	}{
		{Float[float64](), "Float[float64]()"},
		{FloatMin[float32](0.5), "FloatMin[float32](0.5)"},
		{FloatMax(math.Inf(1)), "FloatMax[float64](+Inf)"},
		{FloatRange(-1.5, 2.0), "FloatRange[float64](-1.5, 2)"},
		{Float[celsius](), "Float[rapid_test.celsius]()"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()

			if got := tc.gen.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}
