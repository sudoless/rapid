// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	. "go.sdls.io/rapid"
)

var rv = reflect.ValueOf

func TestIntMinMaxRange(t *testing.T) {
	t.Parallel()

	data := []struct {
		g      *Generator[any]
		min    func(any) *Generator[any]
		max    func(any) *Generator[any]
		range_ func(any, any) *Generator[any]
	}{
		{
			Int[int]().AsAny(),
			func(i any) *Generator[any] { return IntMin(i.(int)).AsAny() },
			func(i any) *Generator[any] { return IntMax(i.(int)).AsAny() },
			func(i, j any) *Generator[any] { return IntRange(i.(int), j.(int)).AsAny() },
		},
		{
			Int[int8]().AsAny(),
			func(i any) *Generator[any] { return IntMin[int8](i.(int8)).AsAny() },
			func(i any) *Generator[any] { return IntMax[int8](i.(int8)).AsAny() },
			func(i, j any) *Generator[any] { return IntRange[int8](i.(int8), j.(int8)).AsAny() },
		},
		{
			Int[int16]().AsAny(),
			func(i any) *Generator[any] { return IntMin[int16](i.(int16)).AsAny() },
			func(i any) *Generator[any] { return IntMax[int16](i.(int16)).AsAny() },
			func(i, j any) *Generator[any] { return IntRange[int16](i.(int16), j.(int16)).AsAny() },
		},
		{
			Int[int32]().AsAny(),
			func(i any) *Generator[any] { return IntMin[int32](i.(int32)).AsAny() },
			func(i any) *Generator[any] { return IntMax[int32](i.(int32)).AsAny() },
			func(i, j any) *Generator[any] { return IntRange[int32](i.(int32), j.(int32)).AsAny() },
		},
		{
			Int[int64]().AsAny(),
			func(i any) *Generator[any] { return IntMin[int64](i.(int64)).AsAny() },
			func(i any) *Generator[any] { return IntMax[int64](i.(int64)).AsAny() },
			func(i, j any) *Generator[any] { return IntRange[int64](i.(int64), j.(int64)).AsAny() },
		},
	}

	for _, testCase := range data {
		t.Run(testCase.g.String(), func(t *testing.T) {
			t.Parallel()

			Check(t, func(t *T) {
				lo := testCase.g.Draw(t, "min")
				hi := testCase.g.Draw(t, "max")
				if rv(lo).Int() > rv(hi).Int() {
					t.Skip("min > max")
				}

				i := testCase.min(lo).Draw(t, "i")
				if rv(i).Int() < rv(lo).Int() {
					t.Fatalf("got %v which is less than min %v", i, lo)
				}

				j := testCase.max(hi).Draw(t, "j")
				if rv(j).Int() > rv(hi).Int() {
					t.Fatalf("got %v which is more than max %v", j, hi)
				}

				k := testCase.range_(lo, hi).Draw(t, "k")
				if rv(k).Int() < rv(lo).Int() || rv(k).Int() > rv(hi).Int() {
					t.Fatalf("got %v which is out of bounds [%v, %v]", k, lo, hi)
				}
			})
		})
	}
}

func TestUintMinMaxRange(t *testing.T) {
	t.Parallel()

	data := []struct {
		g      *Generator[any]
		min    func(any) *Generator[any]
		max    func(any) *Generator[any]
		range_ func(any, any) *Generator[any]
	}{
		{
			Int[byte]().AsAny(),
			func(i any) *Generator[any] { return IntMin[byte](i.(byte)).AsAny() },
			func(i any) *Generator[any] { return IntMax[byte](i.(byte)).AsAny() },
			func(i, j any) *Generator[any] { return IntRange[byte](i.(byte), j.(byte)).AsAny() },
		},
		{
			Int[uint]().AsAny(),
			func(i any) *Generator[any] { return IntMin[uint](i.(uint)).AsAny() },
			func(i any) *Generator[any] { return IntMax[uint](i.(uint)).AsAny() },
			func(i, j any) *Generator[any] { return IntRange[uint](i.(uint), j.(uint)).AsAny() },
		},
		{
			Int[uint8]().AsAny(),
			func(i any) *Generator[any] { return IntMin[uint8](i.(uint8)).AsAny() },
			func(i any) *Generator[any] { return IntMax[uint8](i.(uint8)).AsAny() },
			func(i, j any) *Generator[any] { return IntRange[uint8](i.(uint8), j.(uint8)).AsAny() },
		},
		{
			Int[uint16]().AsAny(),
			func(i any) *Generator[any] { return IntMin[uint16](i.(uint16)).AsAny() },
			func(i any) *Generator[any] { return IntMax[uint16](i.(uint16)).AsAny() },
			func(i, j any) *Generator[any] { return IntRange[uint16](i.(uint16), j.(uint16)).AsAny() },
		},
		{
			Int[uint32]().AsAny(),
			func(i any) *Generator[any] { return IntMin[uint32](i.(uint32)).AsAny() },
			func(i any) *Generator[any] { return IntMax[uint32](i.(uint32)).AsAny() },
			func(i, j any) *Generator[any] { return IntRange[uint32](i.(uint32), j.(uint32)).AsAny() },
		},
		{
			Int[uint64]().AsAny(),
			func(i any) *Generator[any] { return IntMin[uint64](i.(uint64)).AsAny() },
			func(i any) *Generator[any] { return IntMax[uint64](i.(uint64)).AsAny() },
			func(i, j any) *Generator[any] { return IntRange[uint64](i.(uint64), j.(uint64)).AsAny() },
		},
		{
			Int[uintptr]().AsAny(),
			func(i any) *Generator[any] { return IntMin[uintptr](i.(uintptr)).AsAny() },
			func(i any) *Generator[any] { return IntMax[uintptr](i.(uintptr)).AsAny() },
			func(i, j any) *Generator[any] { return IntRange[uintptr](i.(uintptr), j.(uintptr)).AsAny() },
		},
	}

	for _, testCase := range data {
		t.Run(testCase.g.String(), func(t *testing.T) {
			t.Parallel()

			Check(t, func(t *T) {
				lo := testCase.g.Draw(t, "min")
				hi := testCase.g.Draw(t, "max")
				if rv(lo).Uint() > rv(hi).Uint() {
					t.Skip("min > max")
				}

				i := testCase.min(lo).Draw(t, "i")
				if rv(i).Uint() < rv(lo).Uint() {
					t.Fatalf("got %v which is less than min %v", i, lo)
				}

				j := testCase.max(hi).Draw(t, "j")
				if rv(j).Uint() > rv(hi).Uint() {
					t.Fatalf("got %v which is more than max %v", j, hi)
				}

				k := testCase.range_(lo, hi).Draw(t, "k")
				if rv(k).Uint() < rv(lo).Uint() || rv(k).Uint() > rv(hi).Uint() {
					t.Fatalf("got %v which is out of bounds [%v, %v]", k, lo, hi)
				}
			})
		})
	}
}

func TestIntBoundCoverage(t *testing.T) {
	t.Parallel()

	Check(t, func(t *T) {
		lo := Int[int]().Draw(t, "min")
		hi := Int[int]().Draw(t, "max")
		if lo > hi {
			lo, hi = hi, lo
		}

		g := IntRange(lo, hi)
		var gotMin, gotMax, gotZero bool
		for seed := range 250 {
			n := g.Example(seed)

			gotMin = gotMin || n == lo
			gotMax = gotMax || n == hi
			gotZero = gotZero || n == 0

			if gotMin && gotMax && (lo > 0 || hi < 0 || gotZero) {
				return
			}
		}

		t.Fatalf("[%v, %v]: got min %v, got max %v, got zero %v", lo, hi, gotMin, gotMax, gotZero)
	})
}

type level uint8

func TestIntNamedType(t *testing.T) {
	t.Parallel()

	Check(t, func(t *T) {
		lo := Int[time.Duration]().Draw(t, "lo")
		hi := IntMin(lo).Draw(t, "hi")
		d := IntRange(lo, hi).Draw(t, "d")
		if d < lo || d > hi {
			t.Fatalf("%d outside of [%d, %d]", d, lo, hi)
		}

		l := IntMax[level](5).Draw(t, "l")
		if l > 5 {
			t.Fatalf("level %d above 5", l)
		}
	})
}

func TestIntString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		gen  fmt.Stringer
		want string
	}{
		{Int[int](), "Int[int]()"},
		{IntMin[uint8](3), "IntMin[uint8](3)"},
		{IntMax(-7), "IntMax[int](-7)"},
		{IntRange[int64](-1, 1<<40), "IntRange[int64](-1, 1099511627776)"},
		{IntRange[time.Duration](-1, 2), "IntRange[time.Duration](-1, 2)"},
		{Int[level](), "Int[rapid_test.level]()"},
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
