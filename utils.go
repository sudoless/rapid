// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"math"
	"math/bits"

	"go.sdls.io/rapid/internal/assert"
)

const (
	biasLabel     = "bias"
	intBitsLabel  = "intbits"
	coinFlipLabel = "coinflip"
	dieRollLabel  = "dieroll"
	repeatLabel   = "@repeat"
)

func bitmask64(n uint) uint64 {
	return uint64(1)<<n - 1
}

func genFloat01(s bitStream) float64 {
	return float64(s.drawBits(53)) * 0x1.0p-53
}

func genGeom(s bitStream, p float64) uint64 {
	assert.True(p > 0 && p <= 1)

	f := genFloat01(s)
	n := math.Log1p(-f) / math.Log1p(-p)

	return uint64(n)
}

func genUintNNoReject(s bitStream, hi uint64) uint64 {
	bitlen := bits.Len64(hi)
	i := s.beginGroup(intBitsLabel, false)
	u := s.drawBits(bitlen)
	s.endGroup(i, false)
	if u > hi {
		u = hi
	}
	return u
}

func genUintNUnbiased(s bitStream, hi uint64) uint64 {
	bitlen := bits.Len64(hi)

	for {
		i := s.beginGroup(intBitsLabel, false)
		u := s.drawBits(bitlen)
		ok := u <= hi
		s.endGroup(i, !ok)
		if ok {
			return u
		}
	}
}

func genUintNBiased(s bitStream, hi uint64) (uint64, bool, bool) {
	bitlen := bits.Len64(hi)
	i := s.beginGroup(biasLabel, false)
	m := math.Max(8, (float64(bitlen)+48)/7)
	n := genGeom(s, 1/(m+1)) + 1
	s.endGroup(i, false)

	if int(n) < bitlen {
		bitlen = int(n)
	} else if int(n) >= 64-(16-int(m))*4 {
		bitlen = 65
	}

	for {
		i := s.beginGroup(intBitsLabel, false)
		u := s.drawBits(bitlen)
		ok := bitlen > 64 || u <= hi
		s.endGroup(i, !ok)
		if bitlen > 64 {
			u = hi
		}
		if u <= hi {
			return u, u == 0 && n == 1, u == hi && bitlen >= int(n)
		}
	}
}

func genUintN(s bitStream, hi uint64, bias bool) (uint64, bool, bool) {
	if bias {
		return genUintNBiased(s, hi)
	} else {
		return genUintNUnbiased(s, hi), false, false
	}
}

func genUintRange(s bitStream, lo, hi uint64, bias bool) (uint64, bool, bool) {
	if lo > hi {
		assert.Truef(false, "invalid range [%v,  %v]", lo, hi) // avoid allocations in the fast path
	}

	u, lOverflow, rOverflow := genUintN(s, hi-lo, bias)

	return lo + u, lOverflow, rOverflow
}

func genIntRange(s bitStream, lo, hi int64, bias bool) (int64, bool, bool) {
	if lo > hi {
		assert.Truef(false, "invalid range [%v,  %v]", lo, hi) // avoid allocations in the fast path
	}

	var posMin, negMin uint64
	var pNeg float64
	switch {
	case lo >= 0:
		posMin = uint64(lo)
		pNeg = 0
	case hi <= 0:
		negMin = uint64(-hi)
		pNeg = 1
	default:
		posMin = 0
		negMin = 1
		pos := uint64(hi) + 1
		neg := uint64(-lo)
		pNeg = float64(neg) / (float64(neg) + float64(pos))
		if bias {
			pNeg = 0.5
		}
	}

	if flipBiasedCoin(s, pNeg) {
		u, lOverflow, rOverflow := genUintRange(s, negMin, uint64(-lo), bias)
		return -int64(u), rOverflow, lOverflow && hi <= 0
	} else {
		u, lOverflow, rOverflow := genUintRange(s, posMin, uint64(hi), bias)
		return int64(u), lOverflow && lo >= 0, rOverflow
	}
}

func genIndex(s bitStream, n int, bias bool) int {
	assert.True(n > 0)

	u, _, _ := genUintN(s, uint64(n-1), bias)

	return int(u)
}

func flipBiasedCoin(s bitStream, p float64) bool {
	assert.True(p >= 0 && p <= 1)

	i := s.beginGroup(coinFlipLabel, false)
	f := genFloat01(s)
	s.endGroup(i, false)

	return f >= 1-p
}

type loadedDie struct {
	table []int
}

func newLoadedDie(weights []int) *loadedDie {
	assert.True(len(weights) > 0)

	if len(weights) == 1 {
		return &loadedDie{
			table: []int{0},
		}
	}

	total := 0
	for _, weight := range weights {
		assert.True(weight > 0 && weight < 100)
		total += weight
	}

	table := make([]int, total)
	i := 0
	for face, weight := range weights {
		for start := i; i < start+weight; i++ {
			table[i] = face
		}
	}

	return &loadedDie{
		table: table,
	}
}

func (d *loadedDie) roll(s bitStream) int {
	i := s.beginGroup(dieRollLabel, false)
	ix := genIndex(s, len(d.table), false)
	s.endGroup(i, false)

	return d.table[ix]
}

type repeat struct {
	minCount   int
	maxCount   int
	avgCount   float64
	pContinue  float64
	count      int
	group      int
	rejected   bool
	rejections int
	forceStop  bool
	label      string
}

func newRepeat(minCount, maxCount int, avgCount float64, label string) *repeat {
	if minCount < 0 {
		minCount = 0
	}
	if maxCount < 0 {
		maxCount = math.MaxInt
	}
	if avgCount < 0 {
		avgCount = float64(minCount) + math.Min(math.Max(float64(minCount), small), (float64(maxCount)-float64(minCount))/2)
	}

	return &repeat{
		minCount:  minCount,
		maxCount:  maxCount,
		avgCount:  avgCount,
		pContinue: 1 - 1/(1+avgCount-float64(minCount)), // TODO was no -minCount intentional?
		group:     -1,
		label:     label + repeatLabel,
	}
}

func (r *repeat) avg() int {
	return int(math.Ceil(r.avgCount))
}

func (r *repeat) more(s bitStream) bool {
	if r.group >= 0 {
		s.endGroup(r.group, r.rejected)
	}

	r.group = s.beginGroup(r.label, true)
	r.rejected = false

	pCont := r.pContinue
	if r.count < r.minCount {
		pCont = 1
	} else if r.forceStop || r.count >= r.maxCount {
		pCont = 0
	}

	cont := flipBiasedCoin(s, pCont)
	if cont {
		r.count++
	} else {
		s.endGroup(r.group, false)
	}

	return cont
}

func (r *repeat) reject() {
	assert.True(r.count > 0)
	r.count--
	r.rejected = true
	r.rejections++

	if r.rejections > r.count*2 {
		if r.count >= r.minCount {
			r.forceStop = true
		} else {
			panic(invalidData("too many rejections in repeat"))
		}
	}
}
