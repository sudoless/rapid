// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"fmt"
	"math"
)

const (
	byteKind    = "Byte"
	intKind     = "Int"
	int8Kind    = "Int8"
	int16Kind   = "Int16"
	int32Kind   = "Int32"
	int64Kind   = "Int64"
	uintKind    = "Uint"
	uint8Kind   = "Uint8"
	uint16Kind  = "Uint16"
	uint32Kind  = "Uint32"
	uint64Kind  = "Uint64"
	uintptrKind = "Uintptr"

	uintptrSize = 32 << (^uintptr(0) >> 32 & 1)
	uintSize    = 32 << (^uint(0) >> 32 & 1)
	intSize     = uintSize

	maxUintptr = 1<<uint(uintptrSize) - 1
)

var integerKindToInfo = map[string]integerKindInfo{
	byteKind:    {size: 1, umax: math.MaxUint8},
	intKind:     {signed: true, size: intSize / 8, smin: math.MinInt, smax: math.MaxInt},
	int8Kind:    {signed: true, size: 1, smin: math.MinInt8, smax: math.MaxInt8},
	int16Kind:   {signed: true, size: 2, smin: math.MinInt16, smax: math.MaxInt16},
	int32Kind:   {signed: true, size: 4, smin: math.MinInt32, smax: math.MaxInt32},
	int64Kind:   {signed: true, size: 8, smin: math.MinInt64, smax: math.MaxInt64},
	uintKind:    {size: uintSize / 8, umax: math.MaxUint},
	uint8Kind:   {size: 1, umax: math.MaxUint8},
	uint16Kind:  {size: 2, umax: math.MaxUint16},
	uint32Kind:  {size: 4, umax: math.MaxUint32},
	uint64Kind:  {size: 8, umax: math.MaxUint64},
	uintptrKind: {size: uintptrSize / 8, umax: maxUintptr},
}

type integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

type integerKindInfo struct {
	signed bool
	size   int
	smin   int64
	smax   int64
	umax   uint64
}

type boolGen struct{}

func Bool() *Generator[bool]       { return newGenerator[bool](&boolGen{}) }
func (g *boolGen) String() string  { return "Bool()" }
func (g *boolGen) value(t *T) bool { return t.s.drawBits(1) == 1 }

func Byte() *Generator[byte]       { return newIntegerGen[byte](byteKind) }
func Int() *Generator[int]         { return newIntegerGen[int](intKind) }
func Int8() *Generator[int8]       { return newIntegerGen[int8](int8Kind) }
func Int16() *Generator[int16]     { return newIntegerGen[int16](int16Kind) }
func Int32() *Generator[int32]     { return newIntegerGen[int32](int32Kind) }
func Int64() *Generator[int64]     { return newIntegerGen[int64](int64Kind) }
func Uint() *Generator[uint]       { return newIntegerGen[uint](uintKind) }
func Uint8() *Generator[uint8]     { return newIntegerGen[uint8](uint8Kind) }
func Uint16() *Generator[uint16]   { return newIntegerGen[uint16](uint16Kind) }
func Uint32() *Generator[uint32]   { return newIntegerGen[uint32](uint32Kind) }
func Uint64() *Generator[uint64]   { return newIntegerGen[uint64](uint64Kind) }
func Uintptr() *Generator[uintptr] { return newIntegerGen[uintptr](uintptrKind) }

func ByteMin(lo byte) *Generator[byte]    { return newUintMinGen[byte](byteKind, uint64(lo)) }
func IntMin(lo int) *Generator[int]       { return newIntMinGen[int](intKind, int64(lo)) }
func Int8Min(lo int8) *Generator[int8]    { return newIntMinGen[int8](int8Kind, int64(lo)) }
func Int16Min(lo int16) *Generator[int16] { return newIntMinGen[int16](int16Kind, int64(lo)) }
func Int32Min(lo int32) *Generator[int32] { return newIntMinGen[int32](int32Kind, int64(lo)) }
func Int64Min(lo int64) *Generator[int64] { return newIntMinGen[int64](int64Kind, lo) }
func UintMin(lo uint) *Generator[uint]    { return newUintMinGen[uint](uintKind, uint64(lo)) }
func Uint8Min(lo uint8) *Generator[uint8] { return newUintMinGen[uint8](uint8Kind, uint64(lo)) }

func Uint16Min(lo uint16) *Generator[uint16] { return newUintMinGen[uint16](uint16Kind, uint64(lo)) }

func Uint32Min(lo uint32) *Generator[uint32] { return newUintMinGen[uint32](uint32Kind, uint64(lo)) }
func Uint64Min(lo uint64) *Generator[uint64] { return newUintMinGen[uint64](uint64Kind, lo) }
func UintptrMin(lo uintptr) *Generator[uintptr] {
	return newUintMinGen[uintptr](uintptrKind, uint64(lo))
}

func ByteMax(hi byte) *Generator[byte]    { return newUintMaxGen[byte](byteKind, uint64(hi)) }
func IntMax(hi int) *Generator[int]       { return newIntMaxGen[int](intKind, int64(hi)) }
func Int8Max(hi int8) *Generator[int8]    { return newIntMaxGen[int8](int8Kind, int64(hi)) }
func Int16Max(hi int16) *Generator[int16] { return newIntMaxGen[int16](int16Kind, int64(hi)) }
func Int32Max(hi int32) *Generator[int32] { return newIntMaxGen[int32](int32Kind, int64(hi)) }
func Int64Max(hi int64) *Generator[int64] { return newIntMaxGen[int64](int64Kind, hi) }
func UintMax(hi uint) *Generator[uint]    { return newUintMaxGen[uint](uintKind, uint64(hi)) }
func Uint8Max(hi uint8) *Generator[uint8] { return newUintMaxGen[uint8](uint8Kind, uint64(hi)) }

func Uint16Max(hi uint16) *Generator[uint16] { return newUintMaxGen[uint16](uint16Kind, uint64(hi)) }

func Uint32Max(hi uint32) *Generator[uint32] { return newUintMaxGen[uint32](uint32Kind, uint64(hi)) }
func Uint64Max(hi uint64) *Generator[uint64] { return newUintMaxGen[uint64](uint64Kind, hi) }
func UintptrMax(hi uintptr) *Generator[uintptr] {
	return newUintMaxGen[uintptr](uintptrKind, uint64(hi))
}

func ByteRange(lo, hi byte) *Generator[byte] {
	return newUintRangeGen[byte](byteKind, uint64(lo), uint64(hi))
}

func IntRange(lo, hi int) *Generator[int] {
	return newIntRangeGen[int](intKind, int64(lo), int64(hi))
}

func Int8Range(lo, hi int8) *Generator[int8] {
	return newIntRangeGen[int8](int8Kind, int64(lo), int64(hi))
}

func Int16Range(lo, hi int16) *Generator[int16] {
	return newIntRangeGen[int16](int16Kind, int64(lo), int64(hi))
}

func Int32Range(lo, hi int32) *Generator[int32] {
	return newIntRangeGen[int32](int32Kind, int64(lo), int64(hi))
}

func Int64Range(lo, hi int64) *Generator[int64] {
	return newIntRangeGen[int64](int64Kind, lo, hi)
}

func UintRange(lo, hi uint) *Generator[uint] {
	return newUintRangeGen[uint](uintKind, uint64(lo), uint64(hi))
}

func Uint8Range(lo, hi uint8) *Generator[uint8] {
	return newUintRangeGen[uint8](uint8Kind, uint64(lo), uint64(hi))
}

func Uint16Range(lo, hi uint16) *Generator[uint16] {
	return newUintRangeGen[uint16](uint16Kind, uint64(lo), uint64(hi))
}

func Uint32Range(lo, hi uint32) *Generator[uint32] {
	return newUintRangeGen[uint32](uint32Kind, uint64(lo), uint64(hi))
}

func Uint64Range(lo, hi uint64) *Generator[uint64] {
	return newUintRangeGen[uint64](uint64Kind, lo, hi)
}

func UintptrRange(lo, hi uintptr) *Generator[uintptr] {
	return newUintRangeGen[uintptr](uintptrKind, uint64(lo), uint64(hi))
}

func newIntegerGen[I integer](kind string) *Generator[I] {
	return newGenerator[I](&integerGen[I]{
		integerKindInfo: integerKindToInfo[kind],
		kind:            kind,
	})
}

func newIntRangeGen[I integer](kind string, lo, hi int64) *Generator[I] {
	assertf(lo <= hi, "invalid integer range [%v, %v]", lo, hi)

	g := &integerGen[I]{
		integerKindInfo: integerKindToInfo[kind],
		kind:            kind,
		hasMin:          true,
		hasMax:          true,
	}
	g.smin = lo
	g.smax = hi

	return newGenerator[I](g)
}

func newIntMinGen[I integer](kind string, lo int64) *Generator[I] {
	g := &integerGen[I]{
		integerKindInfo: integerKindToInfo[kind],
		kind:            kind,
		hasMin:          true,
	}
	g.smin = lo

	return newGenerator[I](g)
}

func newIntMaxGen[I integer](kind string, hi int64) *Generator[I] {
	g := &integerGen[I]{
		integerKindInfo: integerKindToInfo[kind],
		kind:            kind,
		hasMax:          true,
	}
	g.smax = hi

	return newGenerator[I](g)
}

func newUintRangeGen[I integer](kind string, lo, hi uint64) *Generator[I] {
	assertf(lo <= hi, "invalid integer range [%v, %v]", lo, hi)

	g := &integerGen[I]{
		integerKindInfo: integerKindToInfo[kind],
		kind:            kind,
		hasMin:          true,
		hasMax:          true,
	}
	g.umin = lo
	g.umax = hi

	return newGenerator[I](g)
}

func newUintMinGen[I integer](kind string, lo uint64) *Generator[I] {
	g := &integerGen[I]{
		integerKindInfo: integerKindToInfo[kind],
		kind:            kind,
		hasMin:          true,
	}
	g.umin = lo

	return newGenerator[I](g)
}

func newUintMaxGen[I integer](kind string, hi uint64) *Generator[I] {
	g := &integerGen[I]{
		integerKindInfo: integerKindToInfo[kind],
		kind:            kind,
		hasMax:          true,
	}
	g.umax = hi

	return newGenerator[I](g)
}

type integerGen[I integer] struct {
	integerKindInfo
	kind   string
	umin   uint64
	hasMin bool
	hasMax bool
}

func (g *integerGen[I]) String() string {
	switch {
	case g.hasMin && g.hasMax:
		if g.signed {
			return fmt.Sprintf("%sRange(%d, %d)", g.kind, g.smin, g.smax)
		} else {
			return fmt.Sprintf("%sRange(%d, %d)", g.kind, g.umin, g.umax)
		}
	case g.hasMin:
		if g.signed {
			return fmt.Sprintf("%sMin(%d)", g.kind, g.smin)
		} else {
			return fmt.Sprintf("%sMin(%d)", g.kind, g.umin)
		}
	case g.hasMax:
		if g.signed {
			return fmt.Sprintf("%sMax(%d)", g.kind, g.smax)
		} else {
			return fmt.Sprintf("%sMax(%d)", g.kind, g.umax)
		}
	}

	return fmt.Sprintf("%s()", g.kind)
}

func (g *integerGen[I]) value(t *T) I {
	if g.signed {
		i, _, _ := genIntRange(t.s, g.smin, g.smax, true)
		return I(i)
	} else {
		u, _, _ := genUintRange(t.s, g.umin, g.umax, true)
		return I(u)
	}
}
