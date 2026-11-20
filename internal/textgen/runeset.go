// Copyright (c) 2026 SUDOLESS SRL
// SPDX-License-Identifier: BSD-3-Clause-Clear

package textgen

import (
	"slices"
)

// RuneSet is a set of runes held as ranges, in order, so a large table or class costs its ranges rather
// than its runes.
type RuneSet struct {
	lo     []rune // the first rune of each range
	stride []rune // the step between the runes of each range
	ends   []int  // the count of runes in each range and the ranges before it
	runes  []rune // every rune, once expanded
}

// add appends the runes lo, lo+stride, ... up to hi.
func (s *RuneSet) add(lo, hi, stride rune) {
	s.lo = append(s.lo, lo)
	s.stride = append(s.stride, stride)
	s.ends = append(s.ends, s.Len()+int((hi-lo)/stride)+1)
}

// Len returns the number of runes in s.
func (s *RuneSet) Len() int {
	if len(s.ends) == 0 {
		return 0
	}
	return s.ends[len(s.ends)-1]
}

// At returns the rune at index i of s, for 0 <= i < s.Len().
func (s *RuneSet) At(i int) rune {
	if s.runes != nil {
		return s.runes[i]
	}

	k, _ := slices.BinarySearch(s.ends, i+1) // the first range whose end is past i
	start := 0
	if k > 0 {
		start = s.ends[k-1]
	}
	return s.lo[k] + rune(i-start)*s.stride[k]
}

// expand stores every rune of s, for At to index.
func (s *RuneSet) expand() {
	runes := make([]rune, 0, s.Len())
	start := 0
	for k, end := range s.ends {
		for idx := range end - start {
			runes = append(runes, s.lo[k]+rune(idx)*s.stride[k])
		}
		start = end
	}
	s.runes = runes
}
