// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"math/bits"
	"testing"
)

// coverageDraws bounds how many values a coverage check draws before it gives up.
const coverageDraws = 100000

func TestByteCoverage(t *testing.T) {
	t.Parallel()

	checkByteCoverage(t, fixedStatSeed)
}

func checkByteCoverage(t *testing.T, seed uint64) {
	t.Helper()

	s := newRandomBitStream(seed, false)
	seen := map[byte]bool{}
	for range coverageDraws {
		seen[drawFrom(t, s, Byte())] = true
		if len(seen) == 256 {
			return
		}
	}
	t.Fatalf("got %v of 256 byte values in %v draws", len(seen), coverageDraws)
}

func TestIntCoverage(t *testing.T) {
	t.Parallel()

	checkIntCoverage(t, fixedStatSeed)
}

func checkIntCoverage(t *testing.T, seed uint64) {
	t.Helper()

	targets := []coverageTarget[int]{
		{"0", func(i int) bool { return i == 0 }},
		{"1", func(i int) bool { return i == 1 }},
		{"-1", func(i int) bool { return i == -1 }},
		{"even", func(i int) bool { return i%2 == 0 }},
		{"multiple of 17", func(i int) bool { return i%17 == 0 }},
		{"(0, 100)", func(i int) bool { return i > 0 && i < 100 }},
		{"(-100, 0)", func(i int) bool { return i < 0 && i > -100 }},
		{"> 1<<30", func(i int) bool { return i > 1<<30 }},
		{"< -(1<<30)", func(i int) bool { return i < -(1 << 30) }},
	}
	if bits.UintSize == 64 {
		targets = append(targets, coverageTarget[int]{"> 1<<62", func(i int) bool { return i > 1<<62 }})
		targets = append(targets, coverageTarget[int]{"< -(1<<62)", func(i int) bool { return i < -(1 << 62) }})
	}

	s := newRandomBitStream(seed, false)
	for _, target := range targets {
		if !drawsHit(t, s, Int(), target.hit) {
			t.Errorf("no int %v in %v draws", target.name, coverageDraws)
		}
	}
}

func TestUintCoverage(t *testing.T) {
	t.Parallel()

	checkUintCoverage(t, fixedStatSeed)
}

func checkUintCoverage(t *testing.T, seed uint64) {
	t.Helper()

	targets := []coverageTarget[uint]{
		{"0", func(i uint) bool { return i == 0 }},
		{"1", func(i uint) bool { return i == 1 }},
		{"even", func(i uint) bool { return i%2 == 0 }},
		{"multiple of 17", func(i uint) bool { return i%17 == 0 }},
		{"(0, 100)", func(i uint) bool { return i > 0 && i < 100 }},
		{"> 1<<31", func(i uint) bool { return i > 1<<31 }},
	}
	if bits.UintSize == 64 {
		targets = append(targets, coverageTarget[uint]{"> 1<<63", func(i uint) bool { return i > 1<<63 }})
	}

	s := newRandomBitStream(seed, false)
	for _, target := range targets {
		if !drawsHit(t, s, Uint(), target.hit) {
			t.Errorf("no uint %v in %v draws", target.name, coverageDraws)
		}
	}
}

type coverageTarget[V any] struct {
	name string
	hit  func(V) bool
}

// drawsHit reports whether one of up to coverageDraws values drawn from g satisfies hit.
func drawsHit[V any](t *testing.T, s bitStream, g *Generator[V], hit func(V) bool) bool {
	t.Helper()

	for range coverageDraws {
		if hit(drawFrom(t, s, g)) {
			return true
		}
	}
	return false
}

func drawFrom[V any](t *testing.T, s bitStream, g *Generator[V]) V {
	t.Helper()

	v, err := recoverValue(g, newT(nil, s, false, nil))
	if err != nil {
		t.Fatalf("drawing from %v: %v", g, err)
	}
	return v
}
