// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid_test

import (
	"reflect"
	"strconv"
	"testing"

	. "go.sdls.io/rapid"
)

func TestSliceOf(t *testing.T) {
	t.Parallel()

	gens := []*Generator[any]{
		SliceOf(Bool()).AsAny(),
		SliceOf(Int[byte]()).AsAny(),
		SliceOf(Int[int]()).AsAny(),
		SliceOf(Int[uint]()).AsAny(),
	}

	for _, gen := range gens {
		t.Run(gen.String(), func(t *testing.T) {
			t.Parallel()

			Check(t, func(t *T) {
				v := gen.Draw(t, "v")
				if rv(v).Kind() != reflect.Slice {
					t.Fatalf("got not a slice")
				}
				if rv(v).Len() == 0 {
					t.Skip("empty")
				}
			})
		})
	}
}

func TestSliceOfDistinct(t *testing.T) {
	t.Parallel()

	g := SliceOfDistinct(Int[int](), ID[int])

	Check(t, func(t *T) {
		s := g.Draw(t, "s")
		m := map[int]struct{}{}
		for _, elem := range s {
			m[elem] = struct{}{}
		}
		if len(m) != len(s) {
			t.Fatalf("%v unique out of %v", len(m), len(s))
		}
	})
}

func TestSliceOfDistinctBy(t *testing.T) {
	t.Parallel()

	g := SliceOfDistinct(Int[int](), func(i int) string { return strconv.Itoa(i % 5) })

	Check(t, func(t *T) {
		s := g.Draw(t, "s")
		m := map[int]struct{}{}
		for _, elem := range s {
			m[elem%5] = struct{}{}
		}
		if len(m) != len(s) {
			t.Fatalf("%v unique out of %v", len(m), len(s))
		}
	})
}

func TestMapOf(t *testing.T) {
	t.Parallel()

	gens := []*Generator[any]{
		MapOf(Bool(), Int[int]()).AsAny(),
		MapOf(Int[int](), Int[uint]()).AsAny(),
		MapOf(Int[uint](), SliceOf(Bool())).AsAny(),
	}

	for _, gen := range gens {
		t.Run(gen.String(), func(t *testing.T) {
			t.Parallel()

			Check(t, func(t *T) {
				v := gen.Draw(t, "v")
				if rv(v).Kind() != reflect.Map {
					t.Fatalf("got not a map")
				}
				if rv(v).Len() == 0 {
					t.Skip("empty")
				}
			})
		})
	}
}

func TestMapOfValues(t *testing.T) {
	t.Parallel()

	g := MapOfValues(Custom(genStruct), func(s testStruct) int { return s.x })

	Check(t, func(t *T) {
		m := g.Draw(t, "m")
		for key, val := range m {
			if key != val.x {
				t.Fatalf("got key %v with value %v", key, val)
			}
		}
	})
}

func TestCollectionLenLimits(t *testing.T) {
	t.Parallel()

	genFuncs := []func(i, j int) *Generator[any]{
		func(i, j int) *Generator[any] { return StringOfN(IntRange[int32]('A', 'Z'), i, j, -1).AsAny() },
		func(i, j int) *Generator[any] { return SliceOfN(Int[byte](), i, j).AsAny() },
		func(i, j int) *Generator[any] { return SliceOfNDistinct(Int[byte](), i, j, ID[byte]).AsAny() },
		func(i, j int) *Generator[any] {
			return SliceOfNDistinct(Int[int](), i, j, func(n int) int { return n % j }).AsAny()
		},
		func(i, j int) *Generator[any] { return MapOfN(Int[int](), Int[int](), i, j).AsAny() },
		func(i, j int) *Generator[any] { return MapOfNValues(Int[int](), i, j, ID[int]).AsAny() },
		func(i, j int) *Generator[any] {
			return MapOfNValues(Int[int](), i, j, func(n int) int { return n % j }).AsAny()
		},
	}

	for idx, gf := range genFuncs {
		t.Run(strconv.Itoa(idx), func(t *testing.T) {
			t.Parallel()

			Check(t, func(t *T) {
				minLen := IntRange(0, 256).Draw(t, "minLen")
				maxLen := IntMin(minLen).Draw(t, "maxLen")

				s := rv(gf(minLen, maxLen).Draw(t, "s"))
				if s.Len() < minLen {
					t.Fatalf("got collection of length %v with minLen %v", s.Len(), minLen)
				}
				if s.Len() > maxLen {
					t.Fatalf("got collection of length %v with maxLen %v", s.Len(), maxLen)
				}
			})
		})
	}
}
