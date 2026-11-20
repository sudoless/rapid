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

// SliceOf is a shorthand for [SliceOfN](elem, -1, -1).
func SliceOf[E any](elem *Generator[E]) *Generator[[]E] {
	return SliceOfN(elem, -1, -1)
}

// SliceOfN creates a []E generator. If minLen >= 0, generated slices have minimum length of minLen.
// If maxLen >= 0, generated slices have maximum length of maxLen. SliceOfN panics if maxLen >= 0
// and minLen > maxLen.
func SliceOfN[E any](elem *Generator[E], minLen, maxLen int) *Generator[[]E] {
	assert.ValidRange(minLen, maxLen)

	return newGenerator[[]E](&sliceGen[E, struct{}]{
		minLen: minLen,
		maxLen: maxLen,
		elem:   elem,
	})
}

// SliceOfDistinct is a shorthand for [SliceOfNDistinct](elem, -1, -1).
func SliceOfDistinct[E comparable](elem *Generator[E]) *Generator[[]E] {
	return SliceOfNDistinct(elem, -1, -1)
}

// SliceOfNDistinct creates a []E generator whose slices hold no element twice.
// If minLen >= 0, generated slices have minimum length of minLen. If maxLen >= 0, generated slices
// have maximum length of maxLen. SliceOfNDistinct panics if maxLen >= 0 and minLen > maxLen.
func SliceOfNDistinct[E comparable](elem *Generator[E], minLen, maxLen int) *Generator[[]E] {
	assert.ValidRange(minLen, maxLen)

	return newGenerator[[]E](&sliceGen[E, E]{
		minLen:    minLen,
		maxLen:    maxLen,
		elem:      elem,
		keyFn:     func(e E) E { return e },
		checkKeys: mayBeUncomparable(reflect.TypeFor[E]()),
	})
}

// SliceOfDistinctFunc is a shorthand for [SliceOfNDistinctFunc](elem, -1, -1, keyFn).
func SliceOfDistinctFunc[E any, K comparable](elem *Generator[E], keyFn func(E) K) *Generator[[]E] {
	return SliceOfNDistinctFunc(elem, -1, -1, keyFn)
}

// SliceOfNDistinctFunc creates a []E generator whose slices hold no two elements with the same keyFn key.
// If minLen >= 0, generated slices have minimum length of minLen. If maxLen >= 0, generated slices
// have maximum length of maxLen. SliceOfNDistinctFunc panics if maxLen >= 0 and minLen > maxLen.
func SliceOfNDistinctFunc[E any, K comparable](elem *Generator[E], minLen, maxLen int, keyFn func(E) K) *Generator[[]E] {
	assert.ValidRange(minLen, maxLen)

	return newGenerator[[]E](&sliceGen[E, K]{
		minLen:    minLen,
		maxLen:    maxLen,
		elem:      elem,
		keyFn:     keyFn,
		byFunc:    true,
		checkKeys: mayBeUncomparable(reflect.TypeFor[K]()),
	})
}

type sliceGen[E any, K comparable] struct {
	minLen    int
	maxLen    int
	elem      *Generator[E]
	keyFn     func(E) K
	byFunc    bool // keyFn came from the caller, rather than being the identity
	checkKeys bool // K can hold a value that cannot be compared
}

func (g *sliceGen[E, K]) String() string {
	unbounded := g.minLen < 0 && g.maxLen < 0
	switch {
	case g.keyFn == nil && unbounded:
		return fmt.Sprintf("SliceOf(%v)", g.elem)
	case g.keyFn == nil:
		return fmt.Sprintf("SliceOfN(%v, minLen=%v, maxLen=%v)", g.elem, g.minLen, g.maxLen)
	case !g.byFunc && unbounded:
		return fmt.Sprintf("SliceOfDistinct(%v)", g.elem)
	case !g.byFunc:
		return fmt.Sprintf("SliceOfNDistinct(%v, minLen=%v, maxLen=%v)", g.elem, g.minLen, g.maxLen)
	case unbounded:
		return fmt.Sprintf("SliceOfDistinctFunc(%v, key=%T)", g.elem, g.keyFn)
	default:
		return fmt.Sprintf("SliceOfNDistinctFunc(%v, minLen=%v, maxLen=%v, key=%T)", g.elem, g.minLen, g.maxLen, g.keyFn)
	}
}

func (g *sliceGen[E, K]) value(t *T) []E {
	repeat := bitstream.NewRepeat(g.minLen, g.maxLen, -1, g.elem)

	var seen map[K]struct{}
	if g.keyFn != nil {
		seen = make(map[K]struct{}, repeat.Avg())
	}

	sl := make([]E, 0, repeat.Avg())
	for repeat.More(t.s) {
		mark := t.tc.eventMark()
		e := g.elem.value(t)
		if g.keyFn == nil {
			sl = append(sl, e)
		} else {
			k := g.keyFn(e)
			if g.checkKeys {
				checkKey(g, k)
			}
			if _, ok := seen[k]; ok {
				repeat.Reject()
				t.tc.unmarkSince(mark)
			} else {
				seen[k] = struct{}{}
				sl = append(sl, e)
			}
		}
	}

	return sl
}

// MapOf is a shorthand for [MapOfN](key, val, -1, -1).
func MapOf[K comparable, V any](key *Generator[K], val *Generator[V]) *Generator[map[K]V] {
	return MapOfN(key, val, -1, -1)
}

// MapOfN creates a map[K]V generator. If minLen >= 0, generated maps have minimum length of minLen.
// If maxLen >= 0, generated maps have maximum length of maxLen. MapOfN panics if maxLen >= 0
// and minLen > maxLen.
func MapOfN[K comparable, V any](key *Generator[K], val *Generator[V], minLen, maxLen int) *Generator[map[K]V] {
	assert.ValidRange(minLen, maxLen)

	return newGenerator[map[K]V](&mapGen[K, V]{
		minLen:    minLen,
		maxLen:    maxLen,
		key:       key,
		val:       val,
		entry:     &mapEntryLabel{key: key, val: val},
		checkKeys: mayBeUncomparable(reflect.TypeFor[K]()),
	})
}

// MapOfValues is a shorthand for [MapOfNValues](val, -1, -1, keyFn).
func MapOfValues[K comparable, V any](val *Generator[V], keyFn func(V) K) *Generator[map[K]V] {
	return MapOfNValues(val, -1, -1, keyFn)
}

// MapOfNValues creates a map[K]V generator, where keys are generated by applying keyFn to values.
// If minLen >= 0, generated maps have minimum length of minLen. If maxLen >= 0, generated maps
// have maximum length of maxLen. MapOfNValues panics if maxLen >= 0 and minLen > maxLen.
func MapOfNValues[K comparable, V any](val *Generator[V], minLen, maxLen int, keyFn func(V) K) *Generator[map[K]V] {
	assert.ValidRange(minLen, maxLen)

	return newGenerator[map[K]V](&mapGen[K, V]{
		minLen:    minLen,
		maxLen:    maxLen,
		val:       val,
		keyFn:     keyFn,
		entry:     &mapEntryLabel{val: val},
		checkKeys: mayBeUncomparable(reflect.TypeFor[K]()),
	})
}

type mapGen[K comparable, V any] struct {
	minLen    int
	maxLen    int
	key       *Generator[K]
	val       *Generator[V]
	keyFn     func(V) K
	entry     *mapEntryLabel
	checkKeys bool // K can hold a value that cannot be compared
}

func (g *mapGen[K, V]) String() string {
	unbounded := g.minLen < 0 && g.maxLen < 0
	switch {
	case g.key != nil && unbounded:
		return fmt.Sprintf("MapOf(%v, %v)", g.key, g.val)
	case g.key != nil:
		return fmt.Sprintf("MapOfN(%v, %v, minLen=%v, maxLen=%v)", g.key, g.val, g.minLen, g.maxLen)
	case unbounded:
		return fmt.Sprintf("MapOfValues(%v, key=%T)", g.val, g.keyFn)
	default:
		return fmt.Sprintf("MapOfNValues(%v, minLen=%v, maxLen=%v, key=%T)", g.val, g.minLen, g.maxLen, g.keyFn)
	}
}

func (g *mapGen[K, V]) value(t *T) map[K]V {
	repeat := bitstream.NewRepeat(g.minLen, g.maxLen, -1, g.entry)

	m := make(map[K]V, repeat.Avg())
	for repeat.More(t.s) {
		mark := t.tc.eventMark()
		var k K
		var v V
		if g.key != nil {
			k = g.key.value(t)
			v = g.val.value(t)
		} else {
			v = g.val.value(t)
			k = g.keyFn(v)
		}

		if g.checkKeys {
			checkKey(g, k)
		}
		if _, ok := m[k]; ok {
			repeat.Reject()
			t.tc.unmarkSince(mark)
		} else {
			m[k] = v
		}
	}

	return m
}

// mapEntryLabel labels the repetition of a map's entries by its key and value generators: "key,value", or
// the value generator alone when key is nil, as when the keys are made from the values.
type mapEntryLabel struct {
	key, val fmt.Stringer
}

func (l *mapEntryLabel) String() string {
	if l.key == nil {
		return l.val.String()
	}
	return l.key.String() + "," + l.val.String()
}
