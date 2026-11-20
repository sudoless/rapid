// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package engine

import (
	"hash/maphash"
	"log"
	"math"
	"math/bits"
	"strconv"
	"testing"

	"go.sdls.io/rapid/internal/bitstream"
	"go.sdls.io/rapid/internal/stattest"
)

// minimizeInputs returns edge values and random words of every bit length.
func minimizeInputs(t *testing.T) []uint64 {
	t.Helper()

	inputs := []uint64{0, 1, 2, smallBlock, smallBlock + 1, 1 << 32, 1 << 63, math.MaxUint64 - 1, math.MaxUint64}
	s := bitstream.NewRandom(stattest.Seed(t), false)
	for range 1000 {
		width := int(s.DrawBits(6)) + 1
		inputs = append(inputs, s.DrawBits(width))
	}

	return inputs
}

func TestMinimize_UnsetBits(t *testing.T) {
	t.Parallel()

	for _, mask := range minimizeInputs(t) {
		best := minimize(math.MaxUint64, func(x uint64, _ string) bool { return x&mask == mask })
		if best != mask {
			t.Fatalf("unset to %v instead of %v", bin(best), bin(mask))
		}
	}
}

func TestMinimize_SortBits(t *testing.T) {
	t.Parallel()

	for _, u := range minimizeInputs(t) {
		n := bits.OnesCount64(u)
		v := uint64(1<<uint(n) - 1)

		best := minimize(u, func(x uint64, _ string) bool { return bits.OnesCount64(x) == n })
		if best != v {
			t.Fatalf("minimized %v to %v instead of %v (%v bits set)", bin(u), bin(best), bin(v), n)
		}
	}
}

func TestMinimize_LowerBound(t *testing.T) {
	t.Parallel()

	inputs := minimizeInputs(t)
	for idx, lo := range inputs {
		u := max(lo, inputs[(idx+1)%len(inputs)])

		best := minimize(u, func(x uint64, _ string) bool { return x >= lo })
		if best != lo {
			t.Fatalf("minimized %v to %v instead of %v", u, best, lo)
		}
	}
}

func bin(u uint64) string {
	return "0b" + strconv.FormatUint(u, 2)
}

// After the shrinker accepts a smaller test case, the sort pass compares that test case's labels.
func TestShrinkerLabelsFollowAcceptedTestCase(t *testing.T) {
	t.Parallel()

	failure := &TestError{Data: "fail", Trace: "trace"}
	s := &shrinker{
		rec: bitstream.Recorded{
			Data: []uint64{1, 1},
			Groups: []bitstream.Group{
				{Begin: 0, End: 1, Label: bitstream.Label("before"), Standalone: true},
				{Begin: 1, End: 2, Label: bitstream.Label("before"), Standalone: true},
			},
		},
		err: failure,
		run: func(st bitstream.Stream, _ bool, _ *log.Logger) (*TestError, []string) {
			i := st.BeginGroup(bitstream.Label("after"), true)
			st.DrawBits(1)
			st.EndGroup(i, false)
			return failure, nil
		},
		tries:     map[string]int{},
		cacheSeed: maphash.MakeSeed(),
		cache:     map[uint64]struct{}{},
	}

	if got := s.label(0); got != "before" {
		t.Fatalf("label before accepting = %q, want %q", got, "before")
	}
	if !s.accept([]uint64{0}, labelRemoveGroup, "remove a group") {
		t.Fatal("a smaller test case that fails the same way was not accepted")
	}
	if got := s.label(0); got != "after" {
		t.Fatalf("label after accepting = %q, want %q", got, "after")
	}
}
