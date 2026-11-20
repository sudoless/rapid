// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid_test

import (
	"reflect"
	"testing"

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
			Int().AsAny(),
			func(i any) *Generator[any] { return IntMin(i.(int)).AsAny() },
			func(i any) *Generator[any] { return IntMax(i.(int)).AsAny() },
			func(i, j any) *Generator[any] { return IntRange(i.(int), j.(int)).AsAny() },
		},
		{
			Int8().AsAny(),
			func(i any) *Generator[any] { return Int8Min(i.(int8)).AsAny() },
			func(i any) *Generator[any] { return Int8Max(i.(int8)).AsAny() },
			func(i, j any) *Generator[any] { return Int8Range(i.(int8), j.(int8)).AsAny() },
		},
		{
			Int16().AsAny(),
			func(i any) *Generator[any] { return Int16Min(i.(int16)).AsAny() },
			func(i any) *Generator[any] { return Int16Max(i.(int16)).AsAny() },
			func(i, j any) *Generator[any] { return Int16Range(i.(int16), j.(int16)).AsAny() },
		},
		{
			Int32().AsAny(),
			func(i any) *Generator[any] { return Int32Min(i.(int32)).AsAny() },
			func(i any) *Generator[any] { return Int32Max(i.(int32)).AsAny() },
			func(i, j any) *Generator[any] { return Int32Range(i.(int32), j.(int32)).AsAny() },
		},
		{
			Int64().AsAny(),
			func(i any) *Generator[any] { return Int64Min(i.(int64)).AsAny() },
			func(i any) *Generator[any] { return Int64Max(i.(int64)).AsAny() },
			func(i, j any) *Generator[any] { return Int64Range(i.(int64), j.(int64)).AsAny() },
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
			Byte().AsAny(),
			func(i any) *Generator[any] { return ByteMin(i.(byte)).AsAny() },
			func(i any) *Generator[any] { return ByteMax(i.(byte)).AsAny() },
			func(i, j any) *Generator[any] { return ByteRange(i.(byte), j.(byte)).AsAny() },
		},
		{
			Uint().AsAny(),
			func(i any) *Generator[any] { return UintMin(i.(uint)).AsAny() },
			func(i any) *Generator[any] { return UintMax(i.(uint)).AsAny() },
			func(i, j any) *Generator[any] { return UintRange(i.(uint), j.(uint)).AsAny() },
		},
		{
			Uint8().AsAny(),
			func(i any) *Generator[any] { return Uint8Min(i.(uint8)).AsAny() },
			func(i any) *Generator[any] { return Uint8Max(i.(uint8)).AsAny() },
			func(i, j any) *Generator[any] { return Uint8Range(i.(uint8), j.(uint8)).AsAny() },
		},
		{
			Uint16().AsAny(),
			func(i any) *Generator[any] { return Uint16Min(i.(uint16)).AsAny() },
			func(i any) *Generator[any] { return Uint16Max(i.(uint16)).AsAny() },
			func(i, j any) *Generator[any] { return Uint16Range(i.(uint16), j.(uint16)).AsAny() },
		},
		{
			Uint32().AsAny(),
			func(i any) *Generator[any] { return Uint32Min(i.(uint32)).AsAny() },
			func(i any) *Generator[any] { return Uint32Max(i.(uint32)).AsAny() },
			func(i, j any) *Generator[any] { return Uint32Range(i.(uint32), j.(uint32)).AsAny() },
		},
		{
			Uint64().AsAny(),
			func(i any) *Generator[any] { return Uint64Min(i.(uint64)).AsAny() },
			func(i any) *Generator[any] { return Uint64Max(i.(uint64)).AsAny() },
			func(i, j any) *Generator[any] { return Uint64Range(i.(uint64), j.(uint64)).AsAny() },
		},
		{
			Uintptr().AsAny(),
			func(i any) *Generator[any] { return UintptrMin(i.(uintptr)).AsAny() },
			func(i any) *Generator[any] { return UintptrMax(i.(uintptr)).AsAny() },
			func(i, j any) *Generator[any] { return UintptrRange(i.(uintptr), j.(uintptr)).AsAny() },
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
		lo := Int().Draw(t, "min")
		hi := Int().Draw(t, "max")
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
