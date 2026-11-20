// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"context"
	"errors"
	"slices"
	"testing"

	"go.sdls.io/rapid/internal/bitstream"
	"go.sdls.io/rapid/internal/engine"
)

type trivialGenImpl struct{}

func (trivialGenImpl) String() string    { return "" }
func (trivialGenImpl) value(t *T) uint64 { return t.s.DrawBits(64) }

func BenchmarkTrivialGenImplValue(b *testing.B) {
	t := newT(nil, bitstream.NewRandom(engine.BaseSeed(), false), false, nil)
	g := trivialGenImpl{}

	for b.Loop() {
		g.value(t)
	}
}

func BenchmarkGenerator_Value(b *testing.B) {
	t := newT(nil, bitstream.NewRandom(engine.BaseSeed(), false), false, nil)
	g := newGenerator[uint64](trivialGenImpl{})

	for b.Loop() {
		g.value(t)
	}
}

func TestExampleHelper(t *testing.T) {
	t.Parallel()

	g := Custom(func(t *T) int {
		t.Helper()
		return Int[int]().Draw(t, t.Name())
	})

	g.Example(0)
}

func TestCustomExampleContext(t *testing.T) {
	t.Parallel()

	type key struct{}

	g := Custom(func(t *T) context.Context {
		ctx := context.WithValue(t.Context(), key{}, Int[int]().Draw(t, "x"))
		if err := ctx.Err(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return ctx
	})

	ctx := g.Example(0)

	if err := ctx.Err(); err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context to be canceled, got: %v", err)
	}

	if _, ok := ctx.Value(key{}).(int); !ok {
		t.Fatalf("context must have a value")
	}
}

func TestCustomExampleCleanup(t *testing.T) {
	t.Parallel()

	var state bool
	g := Custom(func(t *T) int {
		t.Cleanup(func() { state = false })
		return Int[int]().Draw(t, "x")
	})

	state = true
	_ = g.Example(0)
	if state {
		t.Fatalf("cleanup must be called")
	}
}

// The shrinker swaps only groups whose labels match, so a generator's group carries its label
// whether or not anything has asked for the generator's String yet.
func TestDrawLabelsGroupWithGenerator(t *testing.T) {
	t.Parallel()

	s := bitstream.NewRandom(1, true)
	nt := newT(nil, s, false, nil)
	IntRange(0, 9).Draw(nt, "a")
	Bool().Draw(nt, "b")
	IntRange(0, 9).Draw(nt, "c")

	var got []string
	for _, g := range s.Groups {
		if g.Standalone {
			got = append(got, g.Label.String())
		}
	}

	want := []string{"IntRange[int](0, 9)", "Bool()", "IntRange[int](0, 9)"}
	if !slices.Equal(got, want) {
		t.Fatalf("standalone group labels %q, want %q", got, want)
	}
}
