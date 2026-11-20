// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package bitstream_test

import (
	"math"
	"testing"

	"go.sdls.io/rapid"
	"go.sdls.io/rapid/internal/bitstream"
)

func TestFloat32Range(t *testing.T) {
	t.Parallel()

	bounds := map[string]*rapid.Generator[float32]{
		"non-negative": rapid.Float32Min(0),
		"any":          rapid.Float32(),
	}
	for name, bound := range bounds {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rapid.Check(t, func(t *rapid.T) {
				lo, hi := bound.Draw(t, "min"), bound.Draw(t, "max")
				if lo > hi {
					lo, hi = hi, lo
				}

				s := bitstream.NewRandom(rapid.Int[uint64]().Draw(t, "seed"), false)
				f := bitstream.Float32Range(s, float64(lo), float64(hi))
				if f < lo || f > hi {
					t.Fatalf("%v (0x%x) outside of [%v, %v] ([0x%x, 0x%x])",
						f, math.Float32bits(f), lo, hi, math.Float32bits(lo), math.Float32bits(hi))
				}
			})
		})
	}
}

func TestFloat64Range(t *testing.T) {
	t.Parallel()

	bounds := map[string]*rapid.Generator[float64]{
		"non-negative": rapid.Float64Min(0),
		"any":          rapid.Float64(),
	}
	for name, bound := range bounds {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rapid.Check(t, func(t *rapid.T) {
				lo, hi := bound.Draw(t, "min"), bound.Draw(t, "max")
				if lo > hi {
					lo, hi = hi, lo
				}

				s := bitstream.NewRandom(rapid.Int[uint64]().Draw(t, "seed"), false)
				f := bitstream.Float64Range(s, lo, hi)
				if f < lo || f > hi {
					t.Fatalf("%v (0x%x) outside of [%v, %v] ([0x%x, 0x%x])",
						f, math.Float64bits(f), lo, hi, math.Float64bits(lo), math.Float64bits(hi))
				}
			})
		})
	}
}
