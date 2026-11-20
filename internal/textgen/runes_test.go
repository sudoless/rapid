// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package textgen

import (
	"regexp/syntax"
	"testing"
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
