// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"testing"

	"go.sdls.io/rapid/internal/bitstream"
	"go.sdls.io/rapid/internal/engine"
	"go.sdls.io/rapid/internal/stattest"
)

type intPair struct {
	x int
	y int
}

func BenchmarkHeavyChain3(b *testing.B) {
	t := newT(nil, bitstream.NewRandom(engine.BaseSeed(), false), false, nil)
	g1 := Custom(func(t *T) int { return Int[int]().Draw(t, "") })
	g2 := Map(g1, func(i int) intPair { return intPair{i, i << 13} })
	g3 := Map(g2, func(p intPair) int { return p.x + p.y })

	for b.Loop() {
		g3.value(t)
	}
}

// A Deferred whose fn panics fails the same way on every draw, so the failure reproduces and is shrunk
// rather than reported as flaky.
func TestDeferredPanicReproduces(t *testing.T) {
	t.Parallel()

	gen := Deferred(func() *Generator[int] { panic("deferred fn is broken") })
	res := search(t, stattest.FixedSeed, func(t *T) { _ = gen.Draw(t, "v") })

	if res.Err1 == nil || res.Err2 == nil {
		t.Fatalf("the property did not fail: Err1 %v, Err2 %v", res.Err1, res.Err2)
	}
	if tb1, tb2 := engine.Traceback(res.Err1), engine.Traceback(res.Err2); tb1 != tb2 {
		t.Fatalf("the replay panicked from another stack, so the failure reads as flaky\nreplay:\n%voriginal:\n%v", tb2, tb1)
	}
}
