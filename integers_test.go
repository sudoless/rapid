// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"math"
	"math/bits"
	"testing"

	"go.sdls.io/rapid/internal/bitstream"
	"go.sdls.io/rapid/internal/stattest"
)

func TestByteCoverage(t *testing.T) {
	t.Parallel()

	checkByteCoverage(t, stattest.FixedSeed)
}

func checkByteCoverage(t *testing.T, seed uint64) {
	t.Helper()

	s := bitstream.NewRandom(seed, false)
	seen := map[byte]bool{}
	for range stattest.CoverageDraws {
		seen[drawFrom(t, s, Int[byte]())] = true
		if len(seen) == 256 {
			return
		}
	}
	t.Fatalf("got %v of 256 byte values in %v draws", len(seen), stattest.CoverageDraws)
}

func TestIntCoverage(t *testing.T) {
	t.Parallel()

	checkIntCoverage(t, stattest.FixedSeed)
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

	s := bitstream.NewRandom(seed, false)
	for _, target := range targets {
		if !drawsHit(t, s, Int[int](), target.hit) {
			t.Errorf("no int %v in %v draws", target.name, stattest.CoverageDraws)
		}
	}
}

// IntRange produces both bounds of a range, and zero when the range holds it, on ranges drawn from the
// fixed seed.
func TestIntBoundCoverage(t *testing.T) {
	t.Parallel()
	checkIntBoundCoverage(t, stattest.FixedSeed)
}

func checkIntBoundCoverage(t *testing.T, seed uint64) {
	t.Helper()
	checkBoundCoverage(t, seed, Int[int](), IntRange[int], 250)
}

// checkBoundCoverage checks, on 100 ranges drawn with bound from seed, that within examples Example calls
// the generator inRange builds for each range produces both bounds, and zero when the range holds it.
func checkBoundCoverage[N Integer | Floating](t *testing.T, seed uint64, bound *Generator[N], inRange func(lo, hi N) *Generator[N], examples int) {
	t.Helper()

	res := search(t, seed, func(t *T) {
		lo := bound.Draw(t, "min")
		hi := bound.Draw(t, "max")
		if lo > hi {
			lo, hi = hi, lo
		}

		g := inRange(lo, hi)
		var gotMin, gotMax, gotZero bool
		for seed := range examples {
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
	if res.Err1 != nil {
		t.Fatalf("%v (seed %v)", res.Err2, seed)
	}
}

func TestUintCoverage(t *testing.T) {
	t.Parallel()

	checkUintCoverage(t, stattest.FixedSeed)
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

	s := bitstream.NewRandom(seed, false)
	for _, target := range targets {
		if !drawsHit(t, s, Int[uint](), target.hit) {
			t.Errorf("no uint %v in %v draws", target.name, stattest.CoverageDraws)
		}
	}
}

// Int draws across the whole range of its type (integers.go:29), and IntRange across [lo, hi], so values
// in the top half of a wide range, short of its bound, are reachable.
func TestIntWideRangeTopHalfCoverage(t *testing.T) {
	t.Parallel()

	checkIntWideRangeTopHalfCoverage(t, stattest.FixedSeed)
}

func checkIntWideRangeTopHalfCoverage(t *testing.T, seed uint64) {
	t.Helper()

	tests := []struct {
		name string
		gen  *Generator[uint64]
		hit  func(uint64) bool
	}{
		{
			"Int[uint64] in [1<<63, MaxUint64)",
			Int[uint64](),
			func(u uint64) bool { return u >= 1<<63 && u != math.MaxUint64 },
		},
		{
			"Int[int64] in [1<<62, MaxInt64)",
			Map(Int[int64](), func(i int64) uint64 { return uint64(i) }),
			func(u uint64) bool { return int64(u) >= 1<<62 && int64(u) != math.MaxInt64 },
		},
		{
			"Int[int64] in (MinInt64, -(1<<62)]",
			Map(Int[int64](), func(i int64) uint64 { return uint64(i) }),
			func(u uint64) bool { return int64(u) <= -(1<<62) && int64(u) != math.MinInt64 },
		},
		{
			"IntRange[uint64](0, 1<<57-1) in [1<<56, 1<<57-1)",
			IntRange[uint64](0, 1<<57-1),
			func(u uint64) bool { return u >= 1<<56 && u != 1<<57-1 },
		},
		{
			"IntRange[uint64](0, 1<<56-1) in [1<<55, 1<<56-1)",
			IntRange[uint64](0, 1<<56-1),
			func(u uint64) bool { return u >= 1<<55 && u != 1<<56-1 },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if !drawsHit(t, bitstream.NewRandom(seed, false), tc.gen, tc.hit) {
				t.Errorf("no value in %v draws", stattest.CoverageDraws)
			}
		})
	}
}

type coverageTarget[V any] struct {
	name string
	hit  func(V) bool
}

// drawsHit reports whether one of up to stattest.CoverageDraws values drawn from g satisfies hit.
func drawsHit[V any](t *testing.T, s bitstream.Stream, g *Generator[V], hit func(V) bool) bool {
	t.Helper()

	for range stattest.CoverageDraws {
		if hit(drawFrom(t, s, g)) {
			return true
		}
	}
	return false
}

func drawFrom[V any](t *testing.T, s bitstream.Stream, g *Generator[V]) V {
	t.Helper()

	v, err := recoverValue(g, newT(nil, s, false, nil))
	if err != nil {
		t.Fatalf("drawing from %v: %v", g, err)
	}
	return v
}
