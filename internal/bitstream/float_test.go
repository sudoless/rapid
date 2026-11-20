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

// The words IsFloat and LowerFloat rely on are where a float draw records them: the shrinker lowers
// floats through these two functions alone.
func TestFloatLayout(t *testing.T) {
	t.Parallel()

	draws := map[string]func(Stream){
		"float32":          func(s Stream) { Float32Range(s, -1e9, 1e9) },
		"float64":          func(s Stream) { Float64Range(s, -1e9, 1e9) },
		"float64 [0, 1]":   func(s Stream) { Float64Range(s, 0, 1) },
		"float64 [-5, -1]": func(s Stream) { Float64Range(s, -5, -1) },
		"float64 [0, 0]":   func(s Stream) { Float64Range(s, 0, 0) },
		"float64 [-inf, inf]": func(s Stream) {
			Float64Range(s, math.Inf(-1), math.Inf(1))
		},
	}
	for name, draw := range draws {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			s := NewRandom(stattest.FixedSeed, true)
			_ = s.DrawBits(8) // the float does not start the data
			i := s.BeginGroup(Label("float"), true)
			draw(s)
			s.EndGroup(i, false)

			g := s.Groups[i]
			if !IsFloat(s.Groups, i) {
				t.Fatalf("IsFloat does not accept the group of a float draw: %+v", s.Groups)
			}
			want := map[Label][2]int{
				floatExpLabel:    {g.Begin + floatExpBegin, g.Begin + floatExpWord + 1},
				floatSignifLabel: {g.Begin + floatExpWord + 1, g.End},
			}
			for _, inner := range s.Groups[i+1:] {
				if span, ok := want[inner.Label.(Label)]; ok && (inner.Begin != span[0] || inner.End != span[1]) {
					t.Errorf("group %v spans [%v, %v), want [%v, %v)", inner.Label, inner.Begin, inner.End, span[0], span[1])
				}
			}

			for step := range FloatLowerSteps {
				buf := LowerFloat(s.Data, g, step)
				word := g.Begin + floatExpWord + step
				for idx := range buf {
					switch {
					case idx == word && buf[idx] != s.Data[idx]-1:
						t.Errorf("step %v: word %v is %#x, want %#x", step, idx, buf[idx], s.Data[idx]-1)
					case idx > word && idx < g.End && buf[idx] != math.MaxUint64:
						t.Errorf("step %v: word %v is %#x, want the maximum", step, idx, buf[idx])
					case (idx < word || idx >= g.End) && buf[idx] != s.Data[idx]:
						t.Errorf("step %v: word %v changed", step, idx)
					}
				}
			}
		})
	}
}

// A standalone group of a float draw's size that draws no float is not one.
func TestIsFloatRejectsOtherGroups(t *testing.T) {
	t.Parallel()

	s := NewRandom(stattest.FixedSeed, true)
	i := s.BeginGroup(Label("seven words"), true)
	for range floatWords {
		_ = s.DrawBits(64)
	}
	s.EndGroup(i, false)

	if IsFloat(s.Groups, i) {
		t.Fatalf("IsFloat accepts a group of %v plain words", floatWords)
	}
}
