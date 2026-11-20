// Copyright (c) 2026 SUDOLESS SRL
// SPDX-License-Identifier: BSD-3-Clause-Clear

package rapid

import (
	"regexp/syntax"
	"testing"
	"unicode"

	"go.sdls.io/rapid/internal/bitstream"
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
