// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"fmt"
	"math"
	"reflect"
	"slices"
	"strconv"
	"testing"

	"go.sdls.io/rapid/internal/bitstream"
	"go.sdls.io/rapid/internal/engine"
)

const shrinkTestRuns = 10

func TestShrink_IntCmp(t *testing.T) {
	t.Parallel()

	ref := []struct {
		gt bool
		a  int
		b  int
		eq bool
	}{
		{true, 1000000, 1000001, false},
		{true, -1000000, 0, false},
		{true, 0, 0, true},
		{false, 1000000, 0, false},
		{false, -1000000, -1000001, false},
		{false, 0, 0, true},
	}

	for _, testCase := range ref {
		t.Run(fmt.Sprintf("%v", testCase), func(t *testing.T) {
			t.Parallel()

			checkShrink(t, func(t *T) {
				i := Int[int]().Draw(t, "i")
				if ((testCase.gt && i > testCase.a) || (!testCase.gt && i < testCase.a)) || (testCase.eq && i == testCase.a) {
					t.Fail()
				}
			}, testCase.b)
		})
	}
}

func TestShrink_FloatCmp(t *testing.T) {
	t.Parallel()

	type cmp struct {
		gt bool
		a  float64
		b  float64
		eq bool
	}

	ref := []cmp{
		{true, 1000000, 1000000.5, false},
		{true, math.Pi, 3.5, false},
		{true, 1, 1, true},
		{true, -1000000, 1, false},
		{false, -1000000, -1000000.5, false},
		{false, -math.E, -2.75, false},
	}

	for _, testCase := range ref {
		t.Run(fmt.Sprintf("%v", testCase), func(t *testing.T) {
			t.Parallel()

			checkShrink(t, func(t *T) {
				f := Float[float64]().Draw(t, "f")
				if ((testCase.gt && f > testCase.a) || (!testCase.gt && f < testCase.a)) || (testCase.eq && f == testCase.a) {
					t.Fail()
				}
			}, testCase.b)
		})
	}
}

func TestShrink_IntSliceNElemsGt(t *testing.T) {
	t.Parallel()

	checkShrink(t, func(t *T) {
		s := SliceOf(Int[int]()).Draw(t, "s")
		n := 0
		for _, elem := range s {
			if elem > 1000000 {
				n++
			}
		}
		if n > 1 {
			t.Fail()
		}
	}, []int{1000001, 1000001})
}

// Failures recorded with Errorf count as one failure wherever they are recorded, so shrinking may move
// from one call site to another.
func TestShrink_ErrorfAtAnySite(t *testing.T) {
	t.Parallel()

	checkShrink(t, func(t *T) {
		x := Int[int]().Draw(t, "x")
		if x < 1000 {
			return
		}
		if x%2 == 0 {
			t.Errorf("even %d", x)
		} else {
			t.Errorf("odd %d", x)
		}
	}, 1000)
}

func TestShrink_IntSliceElemGe(t *testing.T) {
	t.Parallel()

	checkShrink(t, func(t *T) {
		s := SliceOfN(Int[int](), 1, -1).Draw(t, "s")
		ix := IntRange(0, len(s)-1).Draw(t, "ix")

		if s[ix] >= 100 {
			t.Fail()
		}
	}, []int{100}, 0)
}

func TestShrink_IntSliceElemSpanGe(t *testing.T) {
	t.Parallel()

	checkShrink(t, func(t *T) {
		s := SliceOfN(Int[int](), 4, -1).Draw(t, "s")
		if len(s)%3 == 1 && s[len(s)-1] >= 100 {
			t.Fail()
		}
	}, []int{0, 0, 0, 100})
}

func TestShrink_IntSliceNoDuplicates(t *testing.T) {
	t.Parallel()

	checkShrink(t, func(t *T) {
		s := SliceOfN(IntMin(1), 5, -1).Draw(t, "s")
		slices.Sort(s)
		last := 0
		for _, elem := range s {
			if elem == last {
				return
			}
			last = elem
		}
		t.Fail()
	}, []int{1, 2, 3, 4, 5})
}

func TestShrink_String(t *testing.T) {
	t.Parallel()

	checkShrink(t, func(t *T) {
		s1 := String().Draw(t, "s1")
		s2 := String().Draw(t, "s2")
		if len(s1) > len(s2) {
			t.Fail()
		}
	}, "A", "")
}

func TestShrink_StringOf(t *testing.T) {
	t.Parallel()

	checkShrink(t, func(t *T) {
		s1 := StringOf(RuneFrom([]rune{'X', 'Y', 'Z'})).Draw(t, "s1")
		s2 := StringOf(RuneFrom([]rune{'U', 'V', 'W'})).Draw(t, "s2")
		if len(s1) > len(s2) {
			t.Fail()
		}
	}, "X", "")
}

func checkShrink(t *testing.T, prop func(*T), draws ...any) {
	t.Helper()

	for run := range shrinkTestRuns {
		t.Run(strconv.Itoa(run), func(t *testing.T) {
			t.Helper()
			t.Parallel()

			res := search(t, engine.BaseSeed(), prop)
			seed, err1, err2 := res.Seed, res.Err1, res.Err2
			if err1 == nil {
				t.Fatalf("shrink test did not fail")
			}
			if engine.Traceback(err1) != engine.Traceback(err2) {
				t.Fatalf("flaky shrink test (seed %v)\nTraceback (%v):\n%vOriginal traceback (%v):\n%v", seed, err2, engine.Traceback(err2), err1, engine.Traceback(err1))
			}

			nt := newT(t, bitstream.NewBuffer(res.Buf, false), false, nil)
			nt.onDraw = func(idx int, v any) {
				if idx >= len(draws) || !reflect.DeepEqual(v, draws[idx]) {
					t.Fatalf("draw %v is %#v, want %#v", idx, v, draws)
				}
			}
			_ = checkOnce(nt, prop)
			if nt.draws != len(draws) {
				t.Fatalf("different number of draws: %v vs expected %v", nt.draws, len(draws))
			}
		})
	}
}

// A failure whose Custom tries the generator skipped shrinks to the test case without them: the skipped
// tries are not kept in the shrunk buffer.
func TestShrinkDropsSkippedCustomTries(t *testing.T) {
	t.Parallel()

	gen := Custom(func(t *T) bool {
		b := Bool().Draw(t, "b")
		if b {
			t.SkipNow()
		}
		return b
	})
	for seed := uint64(1); seed <= 20; seed++ {
		res := search(t, seed, func(t *T) {
			_ = gen.Draw(t, "v")
			t.Fatal("fail")
		})
		if !slices.Equal(res.Buf, []uint64{0}) {
			t.Fatalf("seed %v: shrunk to %v, want [0]", seed, res.Buf)
		}
	}
}
