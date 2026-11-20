// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"fmt"
	"reflect"

	"go.sdls.io/rapid/internal/assert"
	"go.sdls.io/rapid/internal/bitstream"
)

type boolGen struct{}

func Bool() *Generator[bool]       { return newGenerator[bool](&boolGen{}) }
func (g *boolGen) String() string  { return "Bool()" }
func (g *boolGen) value(t *T) bool { return t.s.DrawBits(1) == 1 }

// Integer is the constraint of the integer generators: every integer type, named ones included.
type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// Int creates a generator of integers of type I across its whole range:
// Int[int](), Int[uint8](), Int[time.Duration]().
func Int[I Integer]() *Generator[I] {
	return newGenerator[I](newIntegerGen[I](false, 0, false, 0))
}

// IntMin creates a generator of integers of type I that are at least lo.
func IntMin[I Integer](lo I) *Generator[I] {
	return newGenerator[I](newIntegerGen(true, lo, false, 0))
}

// IntMax creates a generator of integers of type I that are at most hi.
func IntMax[I Integer](hi I) *Generator[I] {
	return newGenerator[I](newIntegerGen(false, 0, true, hi))
}

// IntRange creates a generator of integers of type I in [lo, hi]. IntRange panics when lo exceeds hi.
func IntRange[I Integer](lo, hi I) *Generator[I] {
	assert.Truef(lo <= hi, "invalid integer range [%v, %v]", lo, hi)

	return newGenerator[I](newIntegerGen(true, lo, true, hi))
}

// newIntegerGen stops short of the Generator, so the constructors stay small enough to inline
// and a generator built and drawn from in one expression stays off the heap.
func newIntegerGen[I Integer](hasMin bool, lo I, hasMax bool, hi I) *integerGen[I] {
	g := &integerGen[I]{lo: 0, hi: ^I(0), hasMin: hasMin, hasMax: hasMax}
	if ^I(0) < 0 {
		g.hi = I(uint64(1)<<(reflect.TypeFor[I]().Bits()-1) - 1)
		g.lo = -g.hi - 1
	}
	if hasMin {
		g.lo = lo
	}
	if hasMax {
		g.hi = hi
	}

	return g
}

type integerGen[I Integer] struct {
	lo     I
	hi     I
	hasMin bool
	hasMax bool
}

func (g *integerGen[I]) String() string {
	typ := reflect.TypeFor[I]().String()
	switch {
	case g.hasMin && g.hasMax:
		return fmt.Sprintf("IntRange[%s](%d, %d)", typ, g.lo, g.hi)
	case g.hasMin:
		return fmt.Sprintf("IntMin[%s](%d)", typ, g.lo)
	case g.hasMax:
		return fmt.Sprintf("IntMax[%s](%d)", typ, g.hi)
	}

	return "Int[" + typ + "]()"
}

func (g *integerGen[I]) value(t *T) I {
	if ^I(0) < 0 {
		i, _, _ := bitstream.IntRange(t.s, int64(g.lo), int64(g.hi), true)
		return I(i)
	}

	u, _, _ := bitstream.UintRange(t.s, uint64(g.lo), uint64(g.hi), true)
	return I(u)
}
