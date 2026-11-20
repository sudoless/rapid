// Copyright (c) 2026 SUDOLESS SRL
// SPDX-License-Identifier: BSD-3-Clause-Clear

package textgen

import (
	"slices"
)

// RuneSet is a set of runes held as ranges, in order, so a large class such as a negated one costs
// its ranges rather than its runes.
type RuneSet struct {
	lo   []rune // the first rune of each range
	ends []int  // the count of runes in each range and the ranges before it
}

// Len returns the number of runes in s.
func (s RuneSet) Len() int {
	if len(s.ends) == 0 {
		return 0
	}
	return s.ends[len(s.ends)-1]
}

// At returns the rune at index i of s, for 0 <= i < s.Len().
func (s RuneSet) At(i int) rune {
	k, _ := slices.BinarySearch(s.ends, i+1) // the first range whose end is past i
	start := 0
	if k > 0 {
		start = s.ends[k-1]
	}
	return s.lo[k] + rune(i-start)
}
