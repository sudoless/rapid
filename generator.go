// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"fmt"
	"sync"

	"go.sdls.io/rapid/internal/assert"
	"go.sdls.io/rapid/internal/bitstream"
	"go.sdls.io/rapid/internal/engine"
)

type generatorImpl[V any] interface {
	String() string
	value(t *T) V
}

// Generator describes a generator of values of type V.
type Generator[V any] struct {
	impl    generatorImpl[V]
	strOnce sync.Once
	str     string
}

func newGenerator[V any](impl generatorImpl[V]) *Generator[V] {
	return &Generator[V]{
		impl: impl,
	}
}

func (g *Generator[V]) String() string {
	g.strOnce.Do(func() {
		g.str = g.impl.String()
	})

	return g.str
}

// Draw produces a value from the generator.
func (g *Generator[V]) Draw(t *T, label string) V {
	if t.tbLog {
		t.Helper()
	}

	v := g.value(t)

	if t.onDraw != nil {
		t.onDraw(t.draws, v)
	}

	if t.tbLog || t.rawLog != nil {
		if label == "" {
			label = fmt.Sprintf("#%v", t.draws)
		}

		if t.tbLog {
			t.Helper()
		}
		t.Logf("[rapid] draw %v: %#v", label, v)
	}

	t.draws++

	return v
}

func (g *Generator[V]) value(t *T) V {
	i := t.s.BeginGroup(g.impl, true)
	v := g.impl.value(t)
	t.s.EndGroup(i, false)
	return v
}

// Example produces an example value from the generator. If seed is provided, value is produced deterministically
// based on seed. Example should only be used for examples; always use *Generator.Draw in property-based tests.
func (g *Generator[V]) Example(seed ...int) V {
	s := engine.BaseSeed()
	if len(seed) > 0 {
		s = uint64(seed[0])
	}

	v, n, err := example(g, bitstream.NewRandom(s, false))
	assert.Truef(err == nil, "%v failed to generate an example in %v tries: %v", g, n, err)

	return v
}

// Filter creates a generator producing only values from g for which fn returns true.
func (g *Generator[V]) Filter(fn func(V) bool) *Generator[V] {
	return filter(g, fn)
}

// AsAny creates a generator producing values from g converted to any.
func (g *Generator[V]) AsAny() *Generator[any] {
	return asAny(g)
}

// example draws a value from g on s, giving every attempt a test case of its own.
func example[V any](g *Generator[V], s bitstream.Stream) (V, int, error) {
	for attempt := 1; ; attempt++ {
		r, err := recoverValue(g, newT(nil, s, false, nil))
		if err == nil {
			return r, attempt, nil
		} else if attempt == exampleMaxTries {
			var zero V
			return zero, attempt, err
		}
	}
}

// recoverValue draws a value from g as one test case, and returns how it failed, after its cleanups.
func recoverValue[V any](g *Generator[V], t *T) (v V, err *engine.TestError) {
	defer func() { err = t.settle(panicToError(recover())) }()

	defer t.cleanup()
	return g.value(t), nil
}
