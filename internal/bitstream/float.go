// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package bitstream

import (
	"math"
	"math/bits"

	"go.sdls.io/rapid/internal/assert"
)

const (
	float32ExpBits    = 8
	float32SignifBits = 23

	float64ExpBits    = 11
	float64SignifBits = 52

	floatExpLabel    Label = "floatexp"
	floatSignifLabel Label = "floatsignif"
)

// Float32Range draws a float32 in [lo, hi]; both bounds may be infinite.
func Float32Range(s Stream, lo, hi float64) float32 {
	return float32FromParts(floatRange(s, lo, hi, float32SignifBits))
}

// Float64Range draws a float64 in [lo, hi]; both bounds may be infinite.
func Float64Range(s Stream, lo, hi float64) float64 {
	return float64FromParts(floatRange(s, lo, hi, float64SignifBits))
}

func ufloatFracBits(e int32, signifBits uint) uint {
	switch {
	case e <= 0:
		return signifBits
	case uint(e) < signifBits:
		return signifBits - uint(e)
	default:
		return 0
	}
}

func ufloat32Parts(f float32) (int32, uint64, uint64) {
	u := math.Float32bits(f) & math.MaxInt32

	e := int32(u>>float32SignifBits) - int32(Bitmask64(float32ExpBits-1))
	s := uint64(u) & Bitmask64(float32SignifBits)
	n := ufloatFracBits(e, float32SignifBits)

	return e, s >> n, s & Bitmask64(n)
}

func ufloat64Parts(f float64) (int32, uint64, uint64) {
	u := math.Float64bits(f) & math.MaxInt64

	e := int32(u>>float64SignifBits) - int32(Bitmask64(float64ExpBits-1))
	s := u & Bitmask64(float64SignifBits)
	n := ufloatFracBits(e, float64SignifBits)

	return e, s >> n, s & Bitmask64(n)
}

func ufloat32FromParts(e int32, si, sf uint64) float32 {
	e_ := (uint32(e) + uint32(Bitmask64(float32ExpBits-1))) << float32SignifBits
	s_ := (uint32(si) << ufloatFracBits(e, float32SignifBits)) | uint32(sf)

	return math.Float32frombits(e_ | s_)
}

func ufloat64FromParts(e int32, si, sf uint64) float64 {
	e_ := (uint64(e) + Bitmask64(float64ExpBits-1)) << float64SignifBits
	s_ := (si << ufloatFracBits(e, float64SignifBits)) | sf

	return math.Float64frombits(e_ | s_)
}

func float32FromParts(sign bool, e int32, si, sf uint64) float32 {
	f := ufloat32FromParts(e, si, sf)
	if sign {
		return -f
	} else {
		return f
	}
}

func float64FromParts(sign bool, e int32, si, sf uint64) float64 {
	f := ufloat64FromParts(e, si, sf)
	if sign {
		return -f
	} else {
		return f
	}
}

func ufloatRange(s Stream, lo, hi float64, signifBits uint) (int32, uint64, uint64) {
	assert.True(lo >= 0 && lo <= hi)

	var (
		minExp, maxExp                                 int32
		minSignifI, maxSignifI, minSignifF, maxSignifF uint64
	)
	if signifBits == float32SignifBits {
		minExp, minSignifI, minSignifF = ufloat32Parts(float32(lo))
		maxExp, maxSignifI, maxSignifF = ufloat32Parts(float32(hi))
	} else {
		minExp, minSignifI, minSignifF = ufloat64Parts(lo)
		maxExp, maxSignifI, maxSignifF = ufloat64Parts(hi)
	}

	i := s.BeginGroup(floatExpLabel, false)
	e, lOverflow, rOverflow := IntRange(s, int64(minExp), int64(maxExp), true)
	s.EndGroup(i, false)

	fracBits := ufloatFracBits(int32(e), signifBits)

	j := s.BeginGroup(floatSignifLabel, false)
	var siMin, siMax uint64
	switch {
	case lOverflow:
		siMin, siMax = minSignifI, minSignifI
	case rOverflow:
		siMin, siMax = maxSignifI, maxSignifI
	case minExp == maxExp:
		siMin, siMax = minSignifI, maxSignifI
	case e == int64(minExp):
		siMin, siMax = minSignifI, Bitmask64(signifBits-fracBits)
	case e == int64(maxExp):
		siMin, siMax = 0, maxSignifI
	default:
		siMin, siMax = 0, Bitmask64(signifBits-fracBits)
	}
	si, _, _ := UintRange(s, siMin, siMax, false)
	var sfMin, sfMax uint64
	switch {
	case lOverflow:
		sfMin, sfMax = minSignifF, minSignifF
	case rOverflow:
		sfMin, sfMax = maxSignifF, maxSignifF
	case minExp == maxExp && minSignifI == maxSignifI:
		sfMin, sfMax = minSignifF, maxSignifF
	case e == int64(minExp) && si == minSignifI:
		sfMin, sfMax = minSignifF, Bitmask64(fracBits)
	case e == int64(maxExp) && si == maxSignifI:
		sfMin, sfMax = 0, maxSignifF
	default:
		sfMin, sfMax = 0, Bitmask64(fracBits)
	}
	maxR := bits.Len64(sfMax - sfMin)
	r := UintNNoReject(s, uint64(maxR))
	sf, _, _ := UintRange(s, sfMin, sfMax, false)
	s.EndGroup(j, false)

	for bit := range uint(maxR) - uint(r) {
		mask := ^(uint64(1) << bit)
		if sf&mask < sfMin {
			break
		}
		sf &= mask
	}

	return int32(e), si, sf
}

func floatRange(s Stream, lo, hi float64, signifBits uint) (bool, int32, uint64, uint64) {
	var posMin, negMin, pNeg float64
	switch {
	case lo >= 0:
		posMin = lo
		pNeg = 0
	case hi <= 0:
		negMin = -hi
		pNeg = 1
	default:
		pNeg = 0.5
	}

	if FlipBiasedCoin(s, pNeg) {
		e, si, sf := ufloatRange(s, negMin, -lo, signifBits)
		return true, e, si, sf
	} else {
		e, si, sf := ufloatRange(s, posMin, hi, signifBits)
		return false, e, si, sf
	}
}
