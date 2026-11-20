// Copyright (c) 2026 SUDOLESS SRL
// SPDX-License-Identifier: BSD-3-Clause-Clear

package rapid_test

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"

	. "go.sdls.io/rapid"
)

type sliceHolder struct{ s any }

// A key whose dynamic value cannot be compared, such as a slice held in an interface, makes the generator
// panic with a message that names it and the key's type, rather than with the runtime's hash error.
func TestDistinctUncomparableKey(t *testing.T) {
	t.Parallel()

	asKey := func(i int) any { return []int{i} }
	tests := []struct {
		name    string
		gen     *Generator[any]
		keyType string
	}{
		{"SliceOfDistinct", SliceOfNDistinct(Map(Int[int](), asKey), 1, -1).AsAny(), "[]int"},
		{"SliceOfDistinctFunc", SliceOfNDistinctFunc(Int[int](), 1, -1, asKey).AsAny(), "[]int"},
		{"MapOf", MapOfN(Map(Int[int](), asKey), Int[int](), 1, -1).AsAny(), "[]int"},
		{"MapOfValues", MapOfNValues(Int[int](), 1, -1, asKey).AsAny(), "[]int"},
		{"struct holding a slice", SliceOfNDistinct(Map(Int[int](), func(i int) sliceHolder { return sliceHolder{[]int{i}} }), 1, -1).AsAny(), "rapid_test.sliceHolder"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			Check(t, func(t *T) {
				defer func() {
					r := recover()
					if _, ok := r.(runtime.Error); ok || !strings.Contains(fmt.Sprint(r), "a key of type "+tc.keyType+" ") {
						t.Fatalf("panic %v, want one that names the key type %v", r, tc.keyType)
					}
				}()
				tc.gen.Draw(t, "v")
			})
		})
	}
}

// A map built by Make whose key is drawn as a value that cannot be compared makes the generator panic with
// a message that names it and the key's type, as MapOf does, rather than with the runtime's hash error.
func TestMakeMapUncomparableKey(t *testing.T) {
	t.Parallel()

	gen := MakeCustom[map[any]int](MakeConfig{Kinds: map[reflect.Kind]*Generator[any]{reflect.Interface: SliceOf(Int[int]()).AsAny()}})
	drewKey := false
	Check(t, func(t *T) {
		defer func() {
			r := recover()
			if r == nil {
				return // the map drawn was empty
			}
			drewKey = true
			if _, ok := r.(runtime.Error); ok || !strings.Contains(fmt.Sprint(r), "a key of type []int ") {
				t.Fatalf("panic %v, want one that names the key type []int", r)
			}
		}()
		gen.Draw(t, "v")
	})
	if !drewKey {
		t.Fatal("no draw held a key")
	}
}
