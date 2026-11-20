// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package textgen holds what string generation draws from: the default rune palette,
// the runes of unicode tables and regexp character classes, and parsed regexps.
package textgen

import (
	"regexp/syntax"
	"unicode"

	"go.sdls.io/rapid/internal/bitstream"
)

var (
	// DefaultRunes are the runes rapid.Rune favours: ASCII punctuation and whitespace,
	// control characters, and runes that trip up text handling.
	DefaultRunes = []rune{
		'A', 'a', '?',
		'~', '!', '@', '#', '$', '%', '^', '&', '*', '_', '-', '+', '=',
		'.', ',', ':', ';',
		' ', '\t', '\r', '\n',
		'/', '\\', '|',
		'(', '[', '{', '<',
		'\'', '"', '`',
		'\x00', '\x0B', '\x1B', '\x7F', // NUL, VT, ESC, DEL
		'\uFEFF', '\uFFFD', '\u202E', // BOM, replacement character, RTL override
		'Ⱥ', // In UTF-8, Ⱥ increases in length from 2 to 3 bytes when lowercased
	}

	// DefaultTables are the unicode.Categories without surrogates (which are not allowed in UTF-8), ordered by taste.
	DefaultTables = []*unicode.RangeTable{
		unicode.Lu, // Letter, uppercase        (1781)
		unicode.Ll, // Letter, lowercase        (2145)
		unicode.Lt, // Letter, titlecase          (31)
		unicode.Lm, // Letter, modifier          (250)
		unicode.Lo, // Letter, other          (121212)
		unicode.Nd, // Number, decimal digit     (610)
		unicode.Nl, // Number, letter            (236)
		unicode.No, // Number, other             (807)
		unicode.P,  // Punctuation               (788)
		unicode.Sm, // Symbol, math              (948)
		unicode.Sc, // Symbol, currency           (57)
		unicode.Sk, // Symbol, modifier          (121)
		unicode.So, // Symbol, other            (5984)
		unicode.Mn, // Mark, nonspacing         (1805)
		unicode.Me, // Mark, enclosing            (13)
		unicode.Mc, // Mark, spacing combining   (415)
		unicode.Z,  // Separator                  (19)
		unicode.Cc, // Other, control             (65)
		unicode.Cf, // Other, format             (152)
		unicode.Co, // Other, private use     (137468)
	}

	// DefaultSets are the RuneSets of DefaultTables, built once for every Rune generator to share. They hold
	// their runes expanded, as most draws are from them: indexing a rune is faster than searching its range.
	DefaultSets = expandedSets(DefaultTables)
)

func expandedSets(tables []*unicode.RangeTable) []*RuneSet {
	sets := make([]*RuneSet, len(tables))
	for idx, t := range tables {
		sets[idx] = TableSet(t)
		sets[idx].expand()
	}
	return sets
}

// TableSet returns the runes t holds, in table order.
func TableSet(t *unicode.RangeTable) *RuneSet {
	n := len(t.R16) + len(t.R32)
	set := &RuneSet{lo: make([]rune, 0, n), stride: make([]rune, 0, n), ends: make([]int, 0, n)}
	for _, span := range t.R16 {
		set.add(rune(span.Lo), rune(span.Hi), rune(span.Stride))
	}
	for _, span := range t.R32 {
		set.add(rune(span.Lo), rune(span.Hi), rune(span.Stride))
	}
	return set
}

// ClassRunes returns the runes of the character class re, in the order of its ranges.
func ClassRunes(re *syntax.Regexp) *RuneSet {
	set := &RuneSet{}
	for idx := 0; idx < len(re.Rune); idx += 2 {
		set.add(re.Rune[idx], re.Rune[idx+1], 1)
	}
	return set
}

// FoldCase draws how many times to case-fold r when flags ask for case folding, and folds it.
func FoldCase(s bitstream.Stream, r rune, flags syntax.Flags) rune {
	n := uint64(0)
	if flags&syntax.FoldCase != 0 {
		n, _, _ = bitstream.UintN(s, 4, false)
	}

	for range n {
		r = unicode.SimpleFold(r)
	}

	return r
}
