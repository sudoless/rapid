// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid_test

import (
	"fmt"
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

	g := SliceOfDistinct(Int[int]())

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

func TestSliceOfDistinctFunc(t *testing.T) {
	t.Parallel()

	g := SliceOfDistinctFunc(Int[int](), func(i int) string { return strconv.Itoa(i % 5) })

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
		func(i, j int) *Generator[any] { return SliceOfNDistinct(Int[byte](), i, j).AsAny() },
		func(i, j int) *Generator[any] {
			return SliceOfNDistinctFunc(Int[int](), i, j, func(n int) int { return n % j }).AsAny()
		},
		func(i, j int) *Generator[any] { return MapOfN(Int[int](), Int[int](), i, j).AsAny() },
		func(i, j int) *Generator[any] {
			return MapOfNValues(Int[int](), i, j, func(n int) int { return n }).AsAny()
		},
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

func TestSliceOfDistinctString(t *testing.T) {
	t.Parallel()

	mod5 := func(i int) int { return i % 5 }
	tests := []struct {
		gen  fmt.Stringer
		want string
	}{
		{SliceOfDistinct(IntRange(0, 9)), "SliceOfDistinct(IntRange[int](0, 9))"},
		{SliceOfNDistinct(IntRange(0, 9), 1, 3), "SliceOfNDistinct(IntRange[int](0, 9), minLen=1, maxLen=3)"},
		{SliceOfDistinctFunc(IntRange(0, 9), mod5), "SliceOfDistinctFunc(IntRange[int](0, 9), key=func(int) int)"},
		{SliceOfNDistinctFunc(IntRange(0, 9), 1, 3, mod5), "SliceOfNDistinctFunc(IntRange[int](0, 9), minLen=1, maxLen=3, key=func(int) int)"},
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
