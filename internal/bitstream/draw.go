// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package bitstream

import (
	"fmt"
	"math"
	"math/bits"

	"go.sdls.io/rapid/internal/assert"
)

// small is the average repetition count NewRepeat aims for when it is not given one.
const small = 5

const (
	biasLabel     Label = "bias"
	intBitsLabel  Label = "intbits"
	coinFlipLabel Label = "coinflip"
	dieRollLabel  Label = "dieroll"
	repeatLabel         = "@repeat"
)

func Bitmask64(n uint) uint64 {
	return uint64(1)<<n - 1
}

func Float01(s Stream) float64 {
	return float64(s.DrawBits(53)) * 0x1.0p-53
}

func geom(s Stream, p float64) uint64 {
	assert.True(p > 0 && p <= 1)

	f := Float01(s)
	n := math.Log1p(-f) / math.Log1p(-p)

	return uint64(n)
}

func UintNNoReject(s Stream, hi uint64) uint64 {
	bitlen := bits.Len64(hi)
	i := s.BeginGroup(intBitsLabel, false)
	u := s.DrawBits(bitlen)
	s.EndGroup(i, false)
	if u > hi {
		u = hi
	}
	return u
}

func genUintNUnbiased(s Stream, hi uint64) uint64 {
	bitlen := bits.Len64(hi)

	for {
		i := s.BeginGroup(intBitsLabel, false)
		u := s.DrawBits(bitlen)
		ok := u <= hi
		s.EndGroup(i, !ok)
		if ok {
			return u
		}
	}
}

func genUintNBiased(s Stream, hi uint64) (uint64, bool, bool) {
	bitlen := bits.Len64(hi)
	i := s.BeginGroup(biasLabel, false)
	m := math.Max(8, (float64(bitlen)+48)/7)
	n := geom(s, 1/(m+1)) + 1
	s.EndGroup(i, false)

	if int(n) < bitlen {
		bitlen = int(n)
	} else if int(n) >= 64-(16-int(m))*4 {
		bitlen = 65
	}

	for {
		i := s.BeginGroup(intBitsLabel, false)
		u := s.DrawBits(bitlen)
		ok := bitlen > 64 || u <= hi
		s.EndGroup(i, !ok)
		if bitlen > 64 {
			u = hi
		}
		if u <= hi {
			return u, u == 0 && n == 1, u == hi && bitlen >= int(n)
		}
	}
}

func UintN(s Stream, hi uint64, bias bool) (uint64, bool, bool) {
	if bias {
		return genUintNBiased(s, hi)
	} else {
		return genUintNUnbiased(s, hi), false, false
	}
}

func UintRange(s Stream, lo, hi uint64, bias bool) (uint64, bool, bool) {
	if lo > hi {
		assert.Truef(false, "invalid range [%v,  %v]", lo, hi) // avoid allocations in the fast path
	}

	u, lOverflow, rOverflow := UintN(s, hi-lo, bias)

	return lo + u, lOverflow, rOverflow
}

func IntRange(s Stream, lo, hi int64, bias bool) (int64, bool, bool) {
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

	if FlipBiasedCoin(s, pNeg) {
		u, lOverflow, rOverflow := UintRange(s, negMin, uint64(-lo), bias)
		return -int64(u), rOverflow, lOverflow && hi <= 0
	} else {
		u, lOverflow, rOverflow := UintRange(s, posMin, uint64(hi), bias)
		return int64(u), lOverflow && lo >= 0, rOverflow
	}
}

func Index(s Stream, n int, bias bool) int {
	assert.True(n > 0)

	u, _, _ := UintN(s, uint64(n-1), bias)

	return int(u)
}

func FlipBiasedCoin(s Stream, p float64) bool {
	assert.True(p >= 0 && p <= 1)

	i := s.BeginGroup(coinFlipLabel, false)
	f := Float01(s)
	s.EndGroup(i, false)

	return f >= 1-p
}

type LoadedDie struct {
	table []int
}

func NewLoadedDie(weights []int) *LoadedDie {
	assert.True(len(weights) > 0)

	if len(weights) == 1 {
		return &LoadedDie{
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

	return &LoadedDie{
		table: table,
	}
}

func (d *LoadedDie) Roll(s Stream) int {
	i := s.BeginGroup(dieRollLabel, false)
	ix := Index(s, len(d.table), false)
	s.EndGroup(i, false)

	return d.table[ix]
}

type Repeat struct {
	minCount   int
	maxCount   int
	avgCount   float64
	pContinue  float64
	count      int
	group      int
	rejected   bool
	rejections int
	forceStop  bool
	label      fmt.Stringer
}

func NewRepeat(minCount, maxCount int, avgCount float64, label fmt.Stringer) *Repeat {
	if minCount < 0 {
		minCount = 0
	}
	if maxCount < 0 {
		maxCount = math.MaxInt
	}
	if avgCount < 0 {
		avgCount = float64(minCount) + math.Min(math.Max(float64(minCount), small), (float64(maxCount)-float64(minCount))/2)
	}

	return &Repeat{
		minCount:  minCount,
		maxCount:  maxCount,
		avgCount:  avgCount,
		pContinue: 1 - 1/(1+avgCount-float64(minCount)), // past minCount, a geometric count of mean avgCount-minCount
		group:     -1,
		label:     label,
	}
}

// String is the label of each repetition's group.
func (r *Repeat) String() string {
	return r.label.String() + repeatLabel
}

func (r *Repeat) Avg() int {
	return int(math.Ceil(r.avgCount))
}

func (r *Repeat) More(s Stream) bool {
	if r.group >= 0 {
		s.EndGroup(r.group, r.rejected)
	}

	r.group = s.BeginGroup(r, true)
	r.rejected = false

	pCont := r.pContinue
	if r.count < r.minCount {
		pCont = 1
	} else if r.forceStop || r.count >= r.maxCount {
		pCont = 0
	}

	cont := FlipBiasedCoin(s, pCont)
	if cont {
		r.count++
	} else {
		s.EndGroup(r.group, false)
	}

	return cont
}

func (r *Repeat) Reject() {
	assert.True(r.count > 0)
	r.count--
	r.rejected = true
	r.rejections++

	if r.rejections > r.count*2 {
		if r.count >= r.minCount {
			r.forceStop = true
		} else {
			panic(InvalidData("too many rejections in repeat"))
		}
	}
}
