// Copyright (c) 2026 SUDOLESS SRL
// SPDX-License-Identifier: BSD-3-Clause-Clear

package rapid

import (
	"regexp/syntax"
	"testing"
	"unicode"

	"go.sdls.io/rapid/internal/bitstream"
	"go.sdls.io/rapid/internal/stattest"
	"go.sdls.io/rapid/internal/textgen"
)

// A character class draws as RuneFrom of one table holding its runes does: under the same label and from
// the same bits, so the two shrink alike.
func TestClassDrawsAsRuneFrom(t *testing.T) {
	t.Parallel()

	re, err := syntax.Parse(`[0-9a-z]`, syntax.Perl)
	if err != nil {
		t.Fatal(err)
	}
	class := textgen.ClassRunes(re)
	gen := RuneFrom(nil, &unicode.RangeTable{R16: []unicode.Range16{{Lo: '0', Hi: '9', Stride: 1}, {Lo: 'a', Hi: 'z', Stride: 1}}})

	if got, want := classRunes.String(), gen.String(); got != want {
		t.Fatalf("a class draws under the label %q, RuneFrom under %q", got, want)
	}
	for seed := range uint64(100) {
		want := drawFrom(t, bitstream.NewRandom(seed, false), gen)
		got := drawClass(newT(nil, bitstream.NewRandom(seed, false), false, nil), class)
		if got != want {
			t.Fatalf("seed %v: the class drew %q, RuneFrom %q", seed, got, want)
		}
	}
}

// StringMatching generates strings matching its regexp (strings.go:183). A character class that holds no
// rune matches nothing, as OpNoMatch does, so a regexp that holds one where it may be skipped still has
// matches to generate, and a draw that reaches it is at worst invalid data, never a failed property.
func TestStringMatchingEmptyClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		expr string
	}{
		{"optional", `a\P{Any}?`},
		{"star", `a[^\x00-\x{10FFFF}]*`},
		{"alternative", `a|b\P{Any}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res := search(t, stattest.FixedSeed, func(t *T) {
				if s := StringMatching(tc.expr).Draw(t, "s"); s != "a" {
					t.Fatalf("%q matches %q only as %q", s, tc.expr, "a")
				}
			})
			if res.Err1 != nil {
				t.Fatalf("StringMatching(%q) failed the property: %v", tc.expr, res.Err1)
			}
		})
	}
}
