// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"fmt"
	"math"

	"go.sdls.io/rapid/internal/assert"
	"go.sdls.io/rapid/internal/bitstream"
)

// Float32 is a shorthand for [Float32Range](-[math.MaxFloat32], [math.MaxFloat32]).
func Float32() *Generator[float32] {
	return Float32Range(-math.MaxFloat32, math.MaxFloat32)
}

// Float32Min is a shorthand for [Float32Range](lo, [math.MaxFloat32]).
func Float32Min(lo float32) *Generator[float32] {
	return Float32Range(lo, math.MaxFloat32)
}

// Float32Max is a shorthand for [Float32Range](-[math.MaxFloat32], hi).
func Float32Max(hi float32) *Generator[float32] {
	return Float32Range(-math.MaxFloat32, hi)
}

// Float32Range creates a generator of 32-bit floating-point numbers in range [lo, hi].
// Both lo and hi can be infinite.
func Float32Range(lo, hi float32) *Generator[float32] {
	assert.Truef(lo == lo, "lo should not be a NaN")
	assert.Truef(hi == hi, "hi should not be a NaN")
	assert.Truef(lo <= hi, "invalid range [%v, %v]", lo, hi)

	return newGenerator[float32](&float32Gen{
		floatGen{
			min:    float64(lo),
			max:    float64(hi),
			minVal: -math.MaxFloat32,
			maxVal: math.MaxFloat32,
		},
	})
}

// Float64 is a shorthand for [Float64Range](-[math.MaxFloat64], [math.MaxFloat64]).
func Float64() *Generator[float64] {
	return Float64Range(-math.MaxFloat64, math.MaxFloat64)
}

// Float64Min is a shorthand for [Float64Range](lo, [math.MaxFloat64]).
func Float64Min(lo float64) *Generator[float64] {
	return Float64Range(lo, math.MaxFloat64)
}

// Float64Max is a shorthand for [Float64Range](-[math.MaxFloat64], hi).
func Float64Max(hi float64) *Generator[float64] {
	return Float64Range(-math.MaxFloat64, hi)
}

// Float64Range creates a generator of 64-bit floating-point numbers in range [lo, hi].
// Both lo and hi can be infinite.
func Float64Range(lo, hi float64) *Generator[float64] {
	assert.Truef(lo == lo, "lo should not be a NaN")
	assert.Truef(hi == hi, "hi should not be a NaN")
	assert.Truef(lo <= hi, "invalid range [%v, %v]", lo, hi)

	return newGenerator[float64](&float64Gen{
		floatGen{
			min:    lo,
			max:    hi,
			minVal: -math.MaxFloat64,
			maxVal: math.MaxFloat64,
		},
	})
}

type floatGen struct {
	min    float64
	max    float64
	minVal float64
	maxVal float64
}
type (
	float32Gen struct{ floatGen }
	float64Gen struct{ floatGen }
)

func (g *floatGen) stringImpl(kind string) string {
	switch {
	case g.min != g.minVal && g.max != g.maxVal:
		return fmt.Sprintf("%sRange(%g, %g)", kind, g.min, g.max)
	case g.min != g.minVal:
		return fmt.Sprintf("%sMin(%g)", kind, g.min)
	case g.max != g.maxVal:
		return fmt.Sprintf("%sMax(%g)", kind, g.max)
	}

	return kind + "()"
}

func (g *float32Gen) String() string {
	return g.stringImpl("Float32")
}

func (g *float64Gen) String() string {
	return g.stringImpl("Float64")
}

func (g *float32Gen) value(t *T) float32 {
	return bitstream.Float32Range(t.s, g.min, g.max)
}

func (g *float64Gen) value(t *T) float64 {
	return bitstream.Float64Range(t.s, g.min, g.max)
}
