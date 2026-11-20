// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid_test

import (
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
