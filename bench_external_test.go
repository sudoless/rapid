// Copyright (c) 2026 SUDOLESS SRL
// SPDX-License-Identifier: BSD-3-Clause-Clear

package rapid_test

import (
	"context"
	"flag"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"go.sdls.io/rapid"
)

// The benchmarks below use the public API only, so the same file runs against other versions of rapid.
// Every run draws the same values: BenchmarkDraw cycles through fixed example seeds, and BenchmarkCheck
// and BenchmarkShrink fix -rapid.seed.

// drawSeeds is how many distinct example seeds BenchmarkDraw cycles through.
const drawSeeds = 1024

// BenchmarkDraw measures one value from each generator: a fresh T and stream, and the draw.
func BenchmarkDraw(b *testing.B) {
	type point struct {
		X, Y int
		Name string
	}

	benchDraw(b, "Bool", rapid.Bool())
	benchDraw(b, "Int", rapid.Int[int]())
	benchDraw(b, "IntRange", rapid.IntRange(-1000, 1000))
	benchDraw(b, "Int8", rapid.Int[int8]())
	benchDraw(b, "Uint64", rapid.Int[uint64]())
	benchDraw(b, "Float64", rapid.Float[float64]())
	benchDraw(b, "Float32Range", rapid.FloatRange[float32](-1, 1))
	benchDraw(b, "Rune", rapid.Rune())
	benchDraw(b, "String", rapid.String())
	benchDraw(b, "StringN", rapid.StringN(5, 20, -1))
	benchDraw(b, "StringMatching", rapid.StringMatching(`[a-z]{3,8}@[a-z]{2,5}\.(com|org)`))
	benchDraw(b, "SliceOfBytesMatching", rapid.SliceOfBytesMatching(`[0-9a-f]{8,16}`))
	benchDraw(b, "SliceOf", rapid.SliceOf(rapid.Int[int]()))
	benchDraw(b, "SliceOfN100", rapid.SliceOfN(rapid.Int[int](), 100, 100))
	benchDraw(b, "SliceOfDistinct", rapid.SliceOfDistinct(rapid.IntRange(0, 1000)))
	benchDraw(b, "SliceOfDistinctFunc", rapid.SliceOfDistinctFunc(rapid.String(), func(s string) int { return len(s) }))
	benchDraw(b, "MapOf", rapid.MapOf(rapid.String(), rapid.Int[int]()))
	benchDraw(b, "MapOfValues", rapid.MapOfValues(rapid.Int[int](), func(i int) int { return i % 100 }))
	benchDraw(b, "Permutation", rapid.Permutation([]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}))
	benchDraw(b, "SampledFrom", rapid.SampledFrom([]string{"a", "b", "c", "d", "e"}))
	benchDraw(b, "Just", rapid.Just(42))
	benchDraw(b, "OneOf", rapid.OneOf(rapid.IntRange(0, 9), rapid.IntRange(100, 109), rapid.Int[int]()))
	benchDraw(b, "Map", rapid.Map(rapid.Int[int](), strconv.Itoa))
	benchDraw(b, "Filter", rapid.Int[int]().Filter(func(i int) bool { return i%2 == 0 }))
	benchDraw(b, "Ptr", rapid.Ptr(rapid.Int[int](), true))
	benchDraw(b, "Custom", rapid.Custom(func(t *rapid.T) point {
		return point{
			X:    rapid.Int[int]().Draw(t, "x"),
			Y:    rapid.Int[int]().Draw(t, "y"),
			Name: rapid.String().Draw(t, "name"),
		}
	}))
	benchDraw(b, "Make", rapid.Make[point]())
}

func benchDraw[V any](b *testing.B, name string, gen *rapid.Generator[V]) {
	b.Helper()

	b.Run(name, func(b *testing.B) {
		seed := 0
		for b.Loop() {
			gen.Example(seed % drawSeeds)
			seed++
		}
	})
}

// BenchmarkCheck measures Check on properties that hold: 100 test cases each.
func BenchmarkCheck(b *testing.B) {
	benchCheck(b, "Ints", func(t *rapid.T) {
		a := rapid.Int[int]().Draw(t, "a")
		c := rapid.Int[int]().Draw(t, "c")
		if a+c != c+a {
			t.Fatalf("addition does not commute")
		}
	})
	benchCheck(b, "SortSlice", func(t *rapid.T) {
		s := rapid.SliceOf(rapid.Int[int]()).Draw(t, "s")
		slices.Sort(s)
		if !slices.IsSorted(s) {
			t.Fatalf("not sorted")
		}
	})
	benchCheck(b, "StringRoundTrip", func(t *rapid.T) {
		s := rapid.String().Draw(t, "s")
		u, err := strconv.Unquote(strconv.Quote(s))
		if err != nil || u != s {
			t.Fatalf("round trip of %q gave %q, %v", s, u, err)
		}
	})
	benchCheck(b, "StateMachine", func(t *rapid.T) {
		var stack []int
		pushed := 0
		t.Repeat(map[string]func(*rapid.T){
			"Push": func(t *rapid.T) {
				stack = append(stack, rapid.Int[int]().Draw(t, "v"))
				pushed++
			},
			"Pop": func(t *rapid.T) {
				if len(stack) == 0 {
					t.Skip("empty")
				}
				stack = stack[:len(stack)-1]
				pushed--
			},
			"": func(t *rapid.T) {
				if len(stack) != pushed {
					t.Fatalf("stack holds %d values, %d pushed", len(stack), pushed)
				}
			},
		})
	})
}

// BenchmarkShrink measures Check on properties that fail: finding the failure, minimising it and reporting it.
// runs/op is how many times the property ran, which does not depend on the machine.
func BenchmarkShrink(b *testing.B) {
	benchShrink(b, "IntThreshold", func(t *rapid.T) {
		if rapid.Int[int]().Draw(t, "x") > 1000 {
			t.Fatalf("too big")
		}
	})
	benchShrink(b, "TwoInts", func(t *rapid.T) {
		a := rapid.IntRange(0, 100).Draw(t, "a")
		c := rapid.IntRange(0, 1000).Draw(t, "c")
		if a > 10 || c > 10 {
			t.Fatalf("too big")
		}
	})
	benchShrink(b, "SliceSum", func(t *rapid.T) {
		sum := 0
		for _, elem := range rapid.SliceOf(rapid.IntRange(0, 100)).Draw(t, "s") {
			sum += elem
		}
		if sum > 200 {
			t.Fatalf("sum %d", sum)
		}
	})
	benchShrink(b, "Unsorted", func(t *rapid.T) {
		s := rapid.SliceOfN(rapid.IntRange(0, 1000), 20, 40).Draw(t, "s")
		if !slices.IsSorted(s) {
			t.Fatalf("unsorted")
		}
	})
	benchShrink(b, "String", func(t *rapid.T) {
		s := rapid.String().Draw(t, "s")
		if utf8.RuneCountInString(s) > 4 && strings.IndexFunc(s, unicode.IsLetter) >= 0 {
			t.Fatalf("long, with a letter")
		}
	})
	benchShrink(b, "Float", func(t *rapid.T) {
		f := rapid.Float[float64]().Draw(t, "f")
		if f > 1.5 && f < 1e6 {
			t.Fatalf("in range")
		}
	})
	benchShrink(b, "Nested", func(t *rapid.T) {
		ss := rapid.SliceOf(rapid.SliceOf(rapid.Int[uint8]())).Draw(t, "ss")
		long := 0
		for _, inner := range ss {
			if len(inner) > 2 {
				long++
			}
		}
		if long >= 2 {
			t.Fatalf("two long slices")
		}
	})
	benchShrink(b, "StateMachine", func(t *rapid.T) {
		var stack []int
		t.Repeat(map[string]func(*rapid.T){
			"Push": func(t *rapid.T) { stack = append(stack, rapid.IntRange(0, 9).Draw(t, "v")) },
			"Pop": func(t *rapid.T) {
				if len(stack) == 0 {
					t.Skip("empty")
				}
				stack = stack[:len(stack)-1]
			},
			"": func(t *rapid.T) {
				if len(stack) > 3 && stack[len(stack)-1] > 5 {
					t.Fatalf("stack %v", stack)
				}
			},
		})
	})
}

func benchCheck(b *testing.B, name string, prop func(*rapid.T)) {
	b.Helper()

	b.Run(name, func(b *testing.B) {
		fixFlags(b)

		for b.Loop() {
			tb := &quietTB{name: name}
			rapid.Check(tb, prop)
			if tb.failed {
				b.Fatalf("property failed: %s", tb.msg)
			}
		}
	})
}

func benchShrink(b *testing.B, name string, prop func(*rapid.T)) {
	b.Helper()

	b.Run(name, func(b *testing.B) {
		fixFlags(b)

		runs := 0
		counted := func(t *rapid.T) {
			runs++
			prop(t)
		}
		for b.Loop() {
			tb := &quietTB{name: name}
			rapid.Check(tb, counted)
			if !tb.failed {
				b.Fatal("property did not fail")
			}
		}
		b.ReportMetric(float64(runs)/float64(b.N), "runs/op")
	})
}

// fixFlags fixes the seed and turns off fail files for b, and restores both when b ends.
func fixFlags(b *testing.B) {
	b.Helper()

	for name, value := range map[string]string{"rapid.seed": "1", "rapid.nofailfile": "true"} {
		f := flag.Lookup(name)
		if f == nil {
			b.Fatalf("flag -%s is not registered", name)
		}
		old := f.Value.String()
		if err := f.Value.Set(value); err != nil {
			b.Fatal(err)
		}
		b.Cleanup(func() { _ = f.Value.Set(old) })
	}
}

// quietTB is a rapid.TB that records a failure instead of failing the benchmark.
type quietTB struct {
	name   string
	failed bool
	msg    string
}

func (tb *quietTB) Helper()                           {}
func (tb *quietTB) Name() string                      { return tb.name }
func (tb *quietTB) Logf(string, ...any)               {}
func (tb *quietTB) Log(...any)                        {}
func (tb *quietTB) Skipf(string, ...any)              {}
func (tb *quietTB) Skip(...any)                       {}
func (tb *quietTB) SkipNow()                          {}
func (tb *quietTB) Errorf(format string, args ...any) { tb.fail(fmt.Sprintf(format, args...)) }
func (tb *quietTB) Error(args ...any)                 { tb.fail(fmt.Sprint(args...)) }
func (tb *quietTB) Fatalf(format string, args ...any) { tb.fail(fmt.Sprintf(format, args...)) }
func (tb *quietTB) Fatal(args ...any)                 { tb.fail(fmt.Sprint(args...)) }
func (tb *quietTB) FailNow()                          { tb.fail("FailNow") }
func (tb *quietTB) Fail()                             { tb.fail("Fail") }
func (tb *quietTB) Failed() bool                      { return tb.failed }
func (tb *quietTB) Context() context.Context          { return context.Background() }
func (tb *quietTB) Output() io.Writer                 { return io.Discard }

func (tb *quietTB) fail(msg string) {
	if !tb.failed {
		tb.failed, tb.msg = true, msg
	}
}
