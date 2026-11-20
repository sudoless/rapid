// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"sync/atomic"

	"go.sdls.io/rapid/internal/assert"
	"go.sdls.io/rapid/internal/bitstream"
	"go.sdls.io/rapid/internal/engine"
)

const tryLabel bitstream.Label = "try"

// Custom creates a generator which produces results of calling fn. In fn, values should be generated
// by calling other generators; it is invalid to return a value from fn without using any other generator.
// Custom is a primary way of creating user-defined generators.
//
// The *T fn receives belongs to the test case drawing the value: a failure fn records with Error, Errorf
// or Fail fails that test case. Its Context and Cleanup cover one call of fn.
func Custom[V any](fn func(*T) V) *Generator[V] {
	return newGenerator[V](&customGen[V]{
		fn: fn,
	})
}

type customGen[V any] struct {
	fn func(*T) V
}

func (g *customGen[V]) String() string {
	var v V
	return fmt.Sprintf("Custom(%T)", v)
}

func (g *customGen[V]) value(t *T) V {
	return find(g.maybeValue, t, small)
}

func (g *customGen[V]) maybeValue(t *T) (V, bool) {
	t = t.child(t.tb, engine.Flags.Debug, nil)
	defer t.cleanup()

	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(bitstream.InvalidData); !ok {
				panic(r)
			}
		}
	}()

	return g.fn(t), true
}

// Deferred creates a generator which defers calling fn until attempting to produce a value. This allows
// to define recursive generators. fn is called once, however many goroutines draw from the generator;
// a fn that panics is called again on the next draw, so the failure reproduces.
func Deferred[V any](fn func() *Generator[V]) *Generator[V] {
	return newGenerator[V](&deferredGen[V]{
		fn: fn,
	})
}

type deferredGen[V any] struct {
	fn  func() *Generator[V]
	mu  sync.Mutex                   // serializes calls of fn
	gen atomic.Pointer[Generator[V]] // what fn returned, once it has returned
}

func (g *deferredGen[V]) String() string {
	var v V
	return fmt.Sprintf("Deferred(%T)", v)
}

func (g *deferredGen[V]) value(t *T) V {
	return g.resolve().value(t)
}

func (g *deferredGen[V]) resolve() *Generator[V] {
	if gen := g.gen.Load(); gen != nil {
		return gen
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	if gen := g.gen.Load(); gen != nil {
		return gen
	}
	gen := g.fn()
	g.gen.Store(gen)
	return gen
}

func filter[V any](g *Generator[V], fn func(V) bool) *Generator[V] {
	return newGenerator[V](&filteredGen[V]{
		g:  g,
		fn: fn,
	})
}

type filteredGen[V any] struct {
	g  *Generator[V]
	fn func(V) bool
}

func (g *filteredGen[V]) String() string {
	return fmt.Sprintf("%v.Filter(...)", g.g)
}

func (g *filteredGen[V]) value(t *T) V {
	return find(g.maybeValue, t, small)
}

func (g *filteredGen[V]) maybeValue(t *T) (V, bool) {
	v := g.g.value(t)
	if g.fn(v) {
		return v, true
	} else {
		var zero V
		return zero, false
	}
}

func find[V any](gen func(*T) (V, bool), t *T, tries int) V {
	for range tries {
		i := t.s.BeginGroup(tryLabel, false)
		v, ok := gen(t)
		t.s.EndGroup(i, !ok)
		if ok {
			return v
		}
	}

	panic(bitstream.InvalidData(fmt.Sprintf("no suitable value in %d tries", tries)))
}

// Map creates a generator producing fn(u) for each u produced by g.
func Map[U any, V any](g *Generator[U], fn func(U) V) *Generator[V] {
	return newGenerator[V](&mappedGen[U, V]{
		g:  g,
		fn: fn,
	})
}

type mappedGen[U any, V any] struct {
	g  *Generator[U]
	fn func(U) V
}

func (g *mappedGen[U, V]) String() string {
	return fmt.Sprintf("Map(%v, %T)", g.g, g.fn)
}

func (g *mappedGen[U, V]) value(t *T) V {
	return g.fn(g.g.value(t))
}

// Just creates a generator which always produces the given value.
// Just(val) is a shorthand for [SampledFrom]([]V{val}).
func Just[V any](val V) *Generator[V] {
	return SampledFrom([]V{val})
}

// SampledFrom creates a generator which produces values from the given slice.
// SampledFrom panics if slice is empty.
func SampledFrom[S ~[]E, E any](slice S) *Generator[E] {
	assert.Truef(len(slice) > 0, "slice should not be empty")

	return newGenerator[E](&sampledGen[E]{
		slice: slice,
	})
}

type sampledGen[E any] struct {
	slice []E
}

func (g *sampledGen[E]) String() string {
	if len(g.slice) == 1 {
		return fmt.Sprintf("Just(%v)", g.slice[0])
	} else {
		return fmt.Sprintf("SampledFrom(%v %T)", len(g.slice), g.slice[0])
	}
}

func (g *sampledGen[E]) value(t *T) E {
	i := bitstream.Index(t.s, len(g.slice), true)

	return g.slice[i]
}

// Permutation creates a generator which produces permutations of the given slice.
func Permutation[S ~[]E, E any](slice S) *Generator[S] {
	return newGenerator[S](&permGen[S, E]{
		slice: slice,
	})
}

type permGen[S ~[]E, E any] struct {
	slice S
}

func (g *permGen[S, E]) String() string {
	var zero E
	return fmt.Sprintf("Permutation(%v %T)", len(g.slice), zero)
}

func (g *permGen[S, E]) value(t *T) S {
	s := append(S(nil), g.slice...)
	n := len(s)
	m := max(n-1, 0)

	// shrink-friendly variant of Fisher–Yates shuffle: shrinks to lower number of smaller distance swaps
	repeat := bitstream.NewRepeat(0, m, math.MaxInt, "permute")
	for idx := 0; repeat.More(t.s); idx++ {
		j, _, _ := bitstream.UintRange(t.s, uint64(idx), uint64(n-1), false)
		s[idx], s[j] = s[j], s[idx]
	}

	return s
}

// OneOf creates a generator which produces each value by selecting one of gens and producing a value from it.
// OneOf panics if gens is empty.
func OneOf[V any](gens ...*Generator[V]) *Generator[V] {
	assert.Truef(len(gens) > 0, "at least one generator should be specified")

	return newGenerator[V](&oneOfGen[V]{
		gens: gens,
	})
}

type oneOfGen[V any] struct {
	gens []*Generator[V]
}

func (g *oneOfGen[V]) String() string {
	strs := make([]string, len(g.gens))
	for idx, gen := range g.gens {
		strs[idx] = gen.String()
	}

	return fmt.Sprintf("OneOf(%v)", strings.Join(strs, ", "))
}

func (g *oneOfGen[V]) value(t *T) V {
	i := bitstream.Index(t.s, len(g.gens), true)

	return g.gens[i].value(t)
}

// Ptr creates a *E generator. If allowNil is true, Ptr can return nil pointers.
func Ptr[E any](elem *Generator[E], allowNil bool) *Generator[*E] {
	return newGenerator[*E](&ptrGen[E]{
		elem:     elem,
		allowNil: allowNil,
	})
}

type ptrGen[E any] struct {
	elem     *Generator[E]
	allowNil bool
}

func (g *ptrGen[E]) String() string {
	return fmt.Sprintf("Ptr(%v, allowNil=%v)", g.elem, g.allowNil)
}

func (g *ptrGen[E]) value(t *T) *E {
	pNonNil := float64(1)
	if g.allowNil {
		pNonNil = 0.5
	}

	if bitstream.FlipBiasedCoin(t.s, pNonNil) {
		e := g.elem.value(t)
		return &e
	} else {
		return nil
	}
}

func asAny[V any](g *Generator[V]) *Generator[any] {
	return newGenerator[any](&asAnyGen[V]{
		gen: g,
	})
}

type asAnyGen[V any] struct {
	gen *Generator[V]
}

func (g *asAnyGen[V]) String() string {
	return fmt.Sprintf("%v.AsAny()", g.gen)
}

func (g *asAnyGen[V]) value(t *T) any {
	return g.gen.value(t)
}
