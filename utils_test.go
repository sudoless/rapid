// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"fmt"
	"math"
	"strconv"
	"testing"
)

// fixedStatSeed seeds the statistical checks in the default suite; TestStress
// (build tag stress) runs the same checks on random seeds.
const fixedStatSeed = 1

// Each statistical check draws statDraws values and allows statSigmas standard
// errors, so a correct generator fails one bound with probability below 1e-6.
const (
	statDraws  = 10000
	statSigmas = 5
)

func createRandomBitStream(t *testing.T) bitStream {
	t.Helper()

	seed := baseSeed()
	t.Logf("random seed %v", seed)

	return newRandomBitStream(seed, false)
}

func TestGenFloat01(t *testing.T) {
	t.Parallel()

	s1 := &bufBitStream{buf: []uint64{0}}
	f1 := genFloat01(s1)
	if f1 != 0 {
		t.Errorf("got %v instead of 0", f1)
	}

	s2 := &bufBitStream{buf: []uint64{math.MaxUint64}}
	f2 := genFloat01(s2)
	if f2 == 1 {
		t.Errorf("got impossible 1")
	}
}

func TestGenGeom(t *testing.T) {
	t.Parallel()

	s1 := &bufBitStream{buf: []uint64{0}}
	i1 := genGeom(s1, 0.1)
	if i1 != 0 {
		t.Errorf("got %v instead of 0 for 0.1", i1)
	}

	s2 := &bufBitStream{buf: []uint64{0}}
	i2 := genGeom(s2, 1)
	if i2 != 0 {
		t.Errorf("got %v instead of 0 for 1", i2)
	}
}

func TestGenGeomMean(t *testing.T) {
	t.Parallel()

	checkGenGeomMean(t, fixedStatSeed)
}

func checkGenGeomMean(t *testing.T, seed uint64) {
	t.Helper()

	s := newRandomBitStream(seed, false)

	// Subtests run in order: they draw from one shared bitstream.
	for i := range 100 {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			p := genFloat01(s)

			var geoms []uint64
			for range statDraws {
				geoms = append(geoms, genGeom(s, p))
			}

			avg := 0.0
			for _, f := range geoms {
				avg += float64(f)
			}
			avg /= float64(len(geoms))

			mean := (1 - p) / p
			stderr := math.Sqrt((1 - p) / (p * p) / float64(len(geoms)))
			if math.Abs(avg-mean) > statSigmas*stderr {
				t.Fatalf("for p=%v geom avg=%v vs expected mean=%v (stderr %v)", p, avg, mean, stderr)
			}
		})
	}
}

func ensureWithinSigmas(t *testing.T, ctx any, y int, p float64) {
	t.Helper()

	n := statDraws
	mu := float64(n) * p
	s := math.Sqrt(float64(n) * p * (1 - p))

	if float64(y) < mu-statSigmas*s || float64(y) > mu+statSigmas*s {
		if ctx != nil {
			t.Errorf("for %v: got %v out of %v (p %v, mu %v, stddev %v)", ctx, y, n, p, mu, s)
		} else {
			t.Errorf("got %v out of %v (p %v, mu %v, stddev %v)", y, n, p, mu, s)
		}
	}
}

func TestGenUintN(t *testing.T) {
	t.Parallel()

	checkGenUintN(t, fixedStatSeed)
}

func checkGenUintN(t *testing.T, seed uint64) {
	t.Helper()

	s := newRandomBitStream(seed, false)
	hi := []uint64{0, 1, 2, 5, 13}

	for _, m := range hi {
		r := make([]int, m+1)
		n := statDraws
		for range n {
			u, _, _ := genUintN(s, m, false)
			r[u]++
		}

		for u := range r {
			ensureWithinSigmas(t, m, r[u], 1/float64(m+1))
		}
	}
}

func TestGenUintRange(t *testing.T) {
	t.Parallel()

	checkGenUintRange(t, fixedStatSeed)
}

func checkGenUintRange(t *testing.T, seed uint64) {
	t.Helper()

	s := newRandomBitStream(seed, false)
	ranges := [][]uint64{
		{0, 0},
		{0, 1},
		{0, 2},
		{1, 1},
		{1, 3},
		{3, 7},
		{math.MaxUint64 - 3, math.MaxUint64},
		{math.MaxUint64 - 1, math.MaxUint64},
		{math.MaxUint64, math.MaxUint64},
	}

	for _, r := range ranges {
		m := map[uint64]int{}
		n := statDraws
		for range n {
			u, _, _ := genUintRange(s, r[0], r[1], false)
			m[u]++
		}

		for u := range m {
			if u < r[0] || u > r[1] {
				t.Errorf("%v out of range [%v, %v]", u, r[0], r[1])
			}
			ensureWithinSigmas(t, fmt.Sprintf("%v from %v", u, r), m[u], 1/float64(r[1]-r[0]+1))
		}
	}
}

func TestGenIntRange(t *testing.T) {
	t.Parallel()

	checkGenIntRange(t, fixedStatSeed)
}

func checkGenIntRange(t *testing.T, seed uint64) {
	t.Helper()

	s := newRandomBitStream(seed, false)
	ranges := [][]int64{
		{0, 0},
		{0, 1},
		{0, 2},
		{1, 1},
		{1, 3},
		{3, 7},
		{math.MaxInt64 - 3, math.MaxInt64},
		{math.MaxInt64 - 1, math.MaxInt64},
		{math.MaxInt64, math.MaxInt64},
		{-1, -1},
		{-2, -1},
		{-3, 0},
		{-1, 1},
		{-1, 3},
		{-3, 7},
		{-7, -3},
		{math.MinInt64, math.MinInt64 + 3},
		{math.MinInt64, math.MinInt64 + 1},
		{math.MinInt64, math.MinInt64},
	}

	for _, r := range ranges {
		m := map[int64]int{}
		n := statDraws
		for range n {
			u, _, _ := genIntRange(s, r[0], r[1], false)
			m[u]++
		}

		for u := range m {
			if u < r[0] || u > r[1] {
				t.Errorf("%v out of range [%v, %v]", u, r[0], r[1])
			}
			ensureWithinSigmas(t, fmt.Sprintf("%v from %v", u, r), m[u], 1/float64(r[1]-r[0]+1))
		}
	}
}

func TestFlipBiasedCoin(t *testing.T) {
	t.Parallel()

	checkFlipBiasedCoin(t, fixedStatSeed)
}

func checkFlipBiasedCoin(t *testing.T, seed uint64) {
	t.Helper()

	s := newRandomBitStream(seed, false)
	ps := []float64{0, 0.3, 0.5, 0.7, 1}

	for _, p := range ps {
		n := statDraws
		y := 0
		for range n {
			if flipBiasedCoin(s, p) {
				y++
			}
		}

		ensureWithinSigmas(t, p, y, p)
	}
}

func TestLoadedDie(t *testing.T) {
	t.Parallel()

	checkLoadedDie(t, fixedStatSeed)
}

func checkLoadedDie(t *testing.T, seed uint64) {
	t.Helper()

	s := newRandomBitStream(seed, false)
	weights := [][]int{
		{1},
		{1, 2},
		{3, 2, 1},
		{1, 2, 4, 2, 1},
	}

	for _, ws := range weights {
		d := newLoadedDie(ws)
		n := statDraws
		r := make([]int, len(ws))

		for range n {
			r[d.roll(s)]++
		}

		total := 0
		for _, w := range ws {
			total += w
		}

		for i, w := range ws {
			ensureWithinSigmas(t, ws, r[i], float64(w)/float64(total))
		}
	}
}

func TestRepeat(t *testing.T) {
	t.Parallel()

	checkRepeat(t, fixedStatSeed)
}

func checkRepeat(t *testing.T, seed uint64) {
	t.Helper()

	mmas := [][3]int{
		{0, 0, 0},
		{0, 1, 0},
		{0, 1, 1},
		{1, 1, 1},
		{3, 3, 3},
		{3, 7, 3},
		{3, 7, 5},
		{3, 7, 7},
		{0, 10, 5},
		{1, 10, 6},
		{0, 50, 5},
		{1, 50, 6},
		{1000, math.MaxInt32, 1000 + 1},
		{1000, math.MaxInt32, 1000 + 2},
		{1000, math.MaxInt32, 1000 + 7},
		{1000, math.MaxInt32, 1000 + 13},
		{1000, math.MaxInt32, 1000 + 100},
		{1000, math.MaxInt32, 1000 + 1000},
	}

	for idx, mma := range mmas {
		t.Run(fmt.Sprint(mma), func(t *testing.T) {
			t.Parallel()

			lo, hi, avg := mma[0], mma[1], mma[2]
			s := newRandomBitStream(seed+uint64(idx), false)

			n := 5000
			c := make([]int, n)
			for i := range n {
				r := newRepeat(lo, hi, float64(avg), "")
				for r.more(s) {
					c[i]++
				}

				if c[i] < lo || c[i] > hi {
					t.Errorf("got %v tries with bounds [%v, %v]", c[i], lo, hi)
				}
			}

			if lo == 1000 && hi == math.MaxInt32 {
				mu := float64(0)
				for _, e := range c {
					mu += float64(e)
				}
				mu /= float64(len(c))

				variance := float64(0)
				for _, e := range c {
					variance += (float64(e) - mu) * (float64(e) - mu)
				}
				variance /= float64(len(c) - 1)
				stderr := math.Sqrt(variance / float64(len(c)))

				diff := math.Abs(mu - float64(avg))
				if diff > statSigmas*stderr {
					t.Errorf("real avg %v vs desired %v, diff %v (stderr %v, %v tries)", mu, avg, diff, stderr, n)
				}
			}
		})
	}
}
