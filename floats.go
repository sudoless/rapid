// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"fmt"
	"math"
	"reflect"

	"go.sdls.io/rapid/internal/assert"
	"go.sdls.io/rapid/internal/bitstream"
)

// Floating is the constraint of the floating-point generators: float32, float64 and the types built on them.
type Floating interface {
	~float32 | ~float64
}

// Float creates a generator of finite floating-point numbers of type F:
// Float[float64](), Float[float32]().
func Float[F Floating]() *Generator[F] {
	return newGenerator[F](newFloatGen[F](false, 0, false, 0))
}

// FloatMin creates a generator of floating-point numbers of type F that are at least lo.
// lo can be -Inf.
func FloatMin[F Floating](lo F) *Generator[F] {
	return newGenerator[F](newFloatGen(true, lo, false, 0))
}

// FloatMax creates a generator of floating-point numbers of type F that are at most hi.
// hi can be +Inf.
func FloatMax[F Floating](hi F) *Generator[F] {
	return newGenerator[F](newFloatGen(false, 0, true, hi))
}

// FloatRange creates a generator of floating-point numbers of type F in [lo, hi].
// Both lo and hi can be infinite. FloatRange panics when lo exceeds hi.
func FloatRange[F Floating](lo, hi F) *Generator[F] {
	return newGenerator[F](newFloatGen(true, lo, true, hi))
}

// newFloatGen stops short of the Generator, so the constructors stay small enough to inline
// and a generator built and drawn from in one expression stays off the heap.
func newFloatGen[F Floating](hasMin bool, lo F, hasMax bool, hi F) *floatGen[F] {
	typ := reflect.TypeFor[F]()
	g := &floatGen[F]{typ: typ.String(), single: typ.Bits() == 32, hasMin: hasMin, hasMax: hasMax}
	g.max = math.MaxFloat64
	if g.single {
		g.max = math.MaxFloat32
	}
	g.min = -g.max
	if hasMin {
		g.min = float64(lo)
	}
	if hasMax {
		g.max = float64(hi)
	}

	assert.Truef(g.min == g.min, "lo should not be a NaN")
	assert.Truef(g.max == g.max, "hi should not be a NaN")
	assert.Truef(g.min <= g.max, "invalid range [%v, %v]", F(g.min), F(g.max))

	return g
}

type floatGen[F Floating] struct {
	typ    string
	single bool
	min    float64
	max    float64
	hasMin bool
	hasMax bool
}

func (g *floatGen[F]) String() string {
	switch {
	case g.hasMin && g.hasMax:
		return fmt.Sprintf("FloatRange[%s](%g, %g)", g.typ, g.min, g.max)
	case g.hasMin:
		return fmt.Sprintf("FloatMin[%s](%g)", g.typ, g.min)
	case g.hasMax:
		return fmt.Sprintf("FloatMax[%s](%g)", g.typ, g.max)
	}

	return "Float[" + g.typ + "]()"
}

func (g *floatGen[F]) value(t *T) F {
	if g.single {
		return F(bitstream.Float32Range(t.s, g.min, g.max))
	}

	return F(bitstream.Float64Range(t.s, g.min, g.max))
}
