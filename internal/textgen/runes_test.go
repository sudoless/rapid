// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package textgen

import (
	"fmt"
	"regexp/syntax"
	"testing"
	"unicode"
)

// A class's RuneSet holds the runes the class matches, in the order of its ranges.
func TestClassRunes(t *testing.T) {
	t.Parallel()

	for _, expr := range []string{`[ab]`, `[a-c]`, `[a-cx-z0-9]`, `[^a-c]`, `[^\n\t]`, `(?i)[k]`, `[\p{Greek}]`} {
		t.Run(expr, func(t *testing.T) {
			t.Parallel()

			re, err := syntax.Parse(expr, syntax.Perl)
			if err != nil {
				t.Fatal(err)
			}
			re = re.Simplify()
			if re.Op != syntax.OpCharClass {
				t.Fatalf("%q parses to %v, not a character class", expr, re.Op)
			}

			var want []rune
			for idx := 0; idx < len(re.Rune); idx += 2 {
				for r := re.Rune[idx]; r <= re.Rune[idx+1]; r++ {
					want = append(want, r)
				}
			}

			set := ClassRunes(re)
			if set.Len() != len(want) {
				t.Fatalf("Len() = %v, want %v", set.Len(), len(want))
			}
			for idx, r := range want {
				if got := set.At(idx); got != r {
					t.Fatalf("At(%v) = %q, want %q", idx, got, r)
				}
			}
		})
	}
}

// A table's RuneSet holds the runes the table does, in table order, strided ranges included, expanded or not.
func TestTableSet(t *testing.T) {
	t.Parallel()

	tables := append([]*unicode.RangeTable{
		{R32: []unicode.Range32{{Lo: 0x10000, Hi: 0x10010, Stride: 4}, {Lo: 0x10FFFE, Hi: 0x10FFFF, Stride: 1}}},
	}, DefaultTables...)
	for idx, table := range tables {
		t.Run(fmt.Sprint(idx), func(t *testing.T) {
			t.Parallel()

			var want []rune
			for _, span := range table.R16 {
				for r := uint32(span.Lo); r <= uint32(span.Hi); r += uint32(span.Stride) {
					want = append(want, rune(r))
				}
			}
			for _, span := range table.R32 {
				for r := uint64(span.Lo); r <= uint64(span.Hi); r += uint64(span.Stride) {
					want = append(want, rune(r))
				}
			}

			expanded := TableSet(table)
			expanded.expand()
			for _, set := range []*RuneSet{TableSet(table), expanded} {
				if set.Len() != len(want) {
					t.Fatalf("Len() = %v, want %v", set.Len(), len(want))
				}
				for idx, r := range want {
					if got := set.At(idx); got != r {
						t.Fatalf("At(%v) = %q, want %q", idx, got, r)
					}
				}
			}
		})
	}
}
