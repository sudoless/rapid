// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid_test

import (
	"math"
	"testing"

	. "go.sdls.io/rapid"
)

// wrapper to test (*T).Helper()
func fatalf(t *T, format string, args ...any) {
	t.Helper()
	t.Fatalf(format, args...)
}

func TestFailure_ImpossibleData(t *testing.T) {
	t.Parallel()

	valid, invalid, failed := CheckOutcome(t, func(t *T) {
		_ = Int().Filter(func(i int) bool { return false }).Draw(t, "i")
	})
	if valid != 0 || invalid == 0 || failed {
		t.Fatalf("got %v valid and %v invalid test cases, failed %v; want only invalid ones", valid, invalid, failed)
	}
}

func TestFailure_Trivial(t *testing.T) {
	t.Parallel()

	t.Skip("expected failure")

	Check(t, func(t *T) {
		i := Int().Draw(t, "i")
		if i > 1000000000 {
			fatalf(t, "got a huge integer: %v", i)
		}
	})
}

func TestFailure_SimpleCollection(t *testing.T) {
	t.Parallel()

	CheckShrink(t, func(t *T) {
		s := SliceOf(Int().Filter(func(i int) bool { return i%2 == -1 })).Draw(t, "s")
		if len(s) > 3 {
			fatalf(t, "got a long sequence: %v", s)
		}
	}, []int{-1, -1, -1, -1})
}

func TestFailure_CollectionElements(t *testing.T) {
	t.Parallel()

	CheckShrink(t, func(t *T) {
		s := SliceOfN(Int(), 2, -1).Draw(t, "s")

		n := 0
		for _, i := range s {
			if i > 1000000 {
				n++
			}
		}

		if n > 1 {
			fatalf(t, "got %v huge elements", n)
		}
	}, []int{1000001, 1000001})
}

func TestFailure_TrivialString(t *testing.T) {
	t.Parallel()

	_, _, failed := CheckOutcome(t, func(t *T) {
		s := String().Draw(t, "s")
		if len(s) > 7 {
			fatalf(t, "got bad string %v", s)
		}
	})
	if !failed {
		t.Fatalf("no string longer than 7 bytes found")
	}
}

func TestFailure_Make(t *testing.T) {
	t.Parallel()

	t.Skip("expected failure")

	Check(t, func(t *T) {
		n := IntMin(0).Draw(t, "n")
		_ = make([]int, n)
	})
}

func TestFailure_Mean(t *testing.T) {
	t.Parallel()

	t.Skip("expected failure")

	Check(t, func(t *T) {
		s := SliceOf(Float64()).Draw(t, "s")

		mean := 0.0
		for _, f := range s {
			mean += f
		}
		mean /= float64(len(s))

		lo, hi := math.Inf(0), math.Inf(-1)
		for _, f := range s {
			if f < lo {
				lo = f
			}
			if f > hi {
				hi = f
			}
		}

		if mean < lo || mean > hi {
			t.Fatalf("got mean %v for range [%v, %v]", mean, lo, hi)
		}
	})
}

func TestFailure_ExampleParseDate(t *testing.T) {
	t.Parallel()

	if _, _, failed := CheckOutcome(t, testParseDate); !failed {
		t.Fatalf("ParseDate month bug not caught")
	}
}

func TestFailure_ExampleQueue(t *testing.T) {
	t.Parallel()

	if _, _, failed := CheckOutcome(t, testQueue); !failed {
		t.Fatalf("Queue size bug not caught")
	}
}

// LastIndex returns the index of the last instance of x in list, or
// -1 if x is not present. The loop condition has a fault that
// causes some tests to fail. Change it to i >= 0 to see them pass.
func LastIndex(list []int, x int) int {
	for i := len(list) - 1; i > 0; i-- {
		if list[i] == x {
			return i
		}
	}
	return -1
}

// This can be a good example of property-based test; however, it is unclear
// what is the "best" way to generate input. Either it is concise (like in
// the test below), but requires high quality data generation (and even then
// is flaky), or it can be verbose, explicitly covering important input classes --
// however, how do we know them when writing a test?
func TestFailure_LastIndex(t *testing.T) {
	t.Parallel()

	t.Skip("expected failure (flaky)")

	Check(t, func(t *T) {
		s := SliceOf(Int()).Draw(t, "s")
		x := Int().Draw(t, "x")
		ix := LastIndex(s, x)

		// index is either -1 or in bounds
		if ix != -1 && (ix < 0 || ix >= len(s)) {
			t.Fatalf("%v is not a valid last index", ix)
		}

		// index is either -1 or a valid index of x
		if ix != -1 && s[ix] != x {
			t.Fatalf("%v is not a valid index of %v", ix, x)
		}

		// no valid index of x is bigger than ix
		for i := ix + 1; i < len(s); i++ {
			if s[i] == x {
				t.Fatalf("%v is not the last index of %v (%v is bigger)", ix, x, i)
			}
		}
	})
}
