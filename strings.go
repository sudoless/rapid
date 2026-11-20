// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"bytes"
	"fmt"
	"math"
	"regexp"
	"regexp/syntax"
	"strings"
	"unicode"
	"unicode/utf8"

	"go.sdls.io/rapid/internal/assert"
	"go.sdls.io/rapid/internal/bitstream"
	"go.sdls.io/rapid/internal/textgen"
)

var (
	anyRuneGen     = Rune()
	anyRuneGenNoNL = Rune().Filter(func(r rune) bool { return r != '\n' })
)

// Rune creates a rune generator. Rune is equivalent to [RuneFrom] with default set of runes and tables.
func Rune() *Generator[rune] {
	return newGenerator[rune](newRuneGen(true, textgen.DefaultRunes, textgen.DefaultSets))
}

// RuneFrom creates a rune generator from provided runes and tables.
// RuneFrom panics if both runes and tables are empty. RuneFrom panics if tables contain an empty table.
func RuneFrom(runes []rune, tables ...*unicode.RangeTable) *Generator[rune] {
	if len(tables) == 0 {
		assert.Truef(len(runes) > 0, "at least one rune should be specified")
	}
	if len(runes) == 0 {
		assert.Truef(len(tables) > 0, "at least one *unicode.RangeTable should be specified")
	}

	sets := make([]*textgen.RuneSet, len(tables))
	for idx := range tables {
		sets[idx] = textgen.TableSet(tables[idx])
		assert.Truef(sets[idx].Len() > 0, "empty *unicode.RangeTable %v", idx)
	}

	return newGenerator[rune](newRuneGen(false, runes, sets))
}

func newRuneGen(default_ bool, runes []rune, tables []*textgen.RuneSet) *runeGen {
	var weights []int
	if len(runes) > 0 {
		weights = append(weights, len(tables))
	}
	for range tables {
		weights = append(weights, 1)
	}

	return &runeGen{
		die:      bitstream.NewLoadedDie(weights),
		runes:    runes,
		tables:   tables,
		default_: default_,
	}
}

type runeGen struct {
	die      *bitstream.LoadedDie
	runes    []rune
	tables   []*textgen.RuneSet
	default_ bool
}

func (g *runeGen) String() string {
	if g.default_ {
		return "Rune()"
	} else {
		return fmt.Sprintf("Rune(%v runes, %v tables)", len(g.runes), len(g.tables))
	}
}

func (g *runeGen) value(t *T) rune {
	return g.draw(t.s, g.tables)
}

// draw rolls g's die for its runes or one of tables, which stand in for g's own, and draws a rune from it.
func (g *runeGen) draw(s bitstream.Stream, tables []*textgen.RuneSet) rune {
	n := g.die.Roll(s)
	if len(g.runes) > 0 {
		if n == 0 {
			return g.runes[bitstream.Index(s, len(g.runes), true)]
		}
		n--
	}

	set := tables[n]
	return set.At(bitstream.Index(s, set.Len(), true))
}

// String is a shorthand for [StringOf]([Rune]()).
func String() *Generator[string] {
	return StringOf(anyRuneGen)
}

// StringN is a shorthand for [StringOfN]([Rune](), minRunes, maxRunes, maxLen).
func StringN(minRunes, maxRunes, maxLen int) *Generator[string] {
	return StringOfN(anyRuneGen, minRunes, maxRunes, maxLen)
}

// StringOf is a shorthand for [StringOfN](elem, -1, -1, -1).
func StringOf(elem *Generator[rune]) *Generator[string] {
	return StringOfN(elem, -1, -1, -1)
}

// StringOfN creates a UTF-8 string generator.
// If minRunes >= 0, generated strings have minimum minRunes runes.
// If maxRunes >= 0, generated strings have maximum maxRunes runes.
// If maxLen >= 0, generates strings have maximum length of maxLen.
// StringOfN panics if maxRunes >= 0 and minRunes > maxRunes.
// StringOfN panics if maxLen >= 0 and maxLen < maxRunes.
func StringOfN(elem *Generator[rune], minRunes, maxRunes, maxLen int) *Generator[string] {
	assert.ValidRange(minRunes, maxRunes)
	assert.Truef(maxLen < 0 || maxLen >= maxRunes,
		"maximum length (%v) should not be less than maximum number of runes (%v)", maxLen, maxRunes)

	return newGenerator[string](&stringGen{
		elem:     elem,
		minRunes: minRunes,
		maxRunes: maxRunes,
		maxLen:   maxLen,
	})
}

type stringGen struct {
	elem     *Generator[rune]
	minRunes int
	maxRunes int
	maxLen   int
}

func (g *stringGen) String() string {
	unbounded := g.minRunes < 0 && g.maxRunes < 0 && g.maxLen < 0
	switch {
	case g.elem == anyRuneGen && unbounded:
		return "String()"
	case g.elem == anyRuneGen:
		return fmt.Sprintf("StringN(minRunes=%v, maxRunes=%v, maxLen=%v)", g.minRunes, g.maxRunes, g.maxLen)
	case unbounded:
		return fmt.Sprintf("StringOf(%v)", g.elem)
	default:
		return fmt.Sprintf("StringOfN(%v, minRunes=%v, maxRunes=%v, maxLen=%v)", g.elem, g.minRunes, g.maxRunes, g.maxLen)
	}
}

func (g *stringGen) value(t *T) string {
	repeat := bitstream.NewRepeat(g.minRunes, g.maxRunes, -1, g.elem)

	var b strings.Builder
	b.Grow(repeat.Avg())

	maxLen := g.maxLen
	if maxLen < 0 {
		maxLen = math.MaxInt
	}

	for repeat.More(t.s) {
		mark := t.tc.eventMark()
		r := g.elem.value(t)
		n := utf8.RuneLen(r)

		if n < 0 || b.Len()+n > maxLen {
			repeat.Reject()
			t.tc.unmarkSince(mark)
		} else {
			b.WriteRune(r)
		}
	}

	return b.String()
}

// StringMatching creates a UTF-8 string generator matching the provided [syntax.Perl] regular expression.
func StringMatching(expr string) *Generator[string] {
	return newGenerator[string](&regexpStringGen{newRegexpGen(expr)})
}

// SliceOfBytesMatching creates a UTF-8 byte slice generator matching the provided [syntax.Perl] regular expression.
func SliceOfBytesMatching(expr string) *Generator[[]byte] {
	return newGenerator[[]byte](&regexpSliceGen{newRegexpGen(expr)})
}

type runeWriter interface {
	WriteRune(r rune) (int, error)
}

type regexpGen struct {
	expr    string
	syn     *syntax.Regexp
	re      *regexp.Regexp
	labels  map[*syntax.Regexp]fmt.Stringer
	classes map[*syntax.Regexp]*textgen.RuneSet
}

func newRegexpGen(expr string) regexpGen {
	compiled, err := textgen.CompileRegexp(expr)
	assert.Truef(err == nil, "%v", err)

	return regexpGen{
		expr:    expr,
		syn:     compiled.Syntax,
		re:      compiled.Matcher,
		labels:  compiled.Labels,
		classes: compiled.Classes,
	}
}

type (
	regexpStringGen struct{ regexpGen }
	regexpSliceGen  struct{ regexpGen }
)

func (g *regexpStringGen) String() string {
	return fmt.Sprintf("StringMatching(%q)", g.expr)
}

func (g *regexpSliceGen) String() string {
	return fmt.Sprintf("SliceOfBytesMatching(%q)", g.expr)
}

func (g *regexpStringGen) maybeString(t *T) (string, bool) {
	b := &strings.Builder{}
	g.build(b, g.syn, t)
	v := b.String()

	if g.re.MatchString(v) {
		return v, true
	} else {
		return "", false
	}
}

func (g *regexpSliceGen) maybeSlice(t *T) ([]byte, bool) {
	b := &bytes.Buffer{}
	g.build(b, g.syn, t)
	v := b.Bytes()

	if g.re.Match(v) {
		return v, true
	} else {
		return nil, false
	}
}

func (g *regexpStringGen) value(t *T) string {
	return find(g.maybeString, t, small)
}

func (g *regexpSliceGen) value(t *T) []byte {
	return find(g.maybeSlice, t, small)
}

func (g *regexpGen) build(w runeWriter, re *syntax.Regexp, t *T) {
	i := t.s.BeginGroup(re.Op, false)

	switch re.Op {
	case syntax.OpNoMatch:
		panic(bitstream.InvalidData("no possible regexp match"))
	case syntax.OpEmptyMatch:
		t.s.DrawBits(0)
	case syntax.OpLiteral:
		t.s.DrawBits(0)
		for _, char := range re.Rune {
			_, _ = w.WriteRune(textgen.FoldCase(t.s, char, re.Flags))
		}
	case syntax.OpCharClass, syntax.OpAnyCharNotNL, syntax.OpAnyChar:
		var r rune
		switch re.Op {
		case syntax.OpCharClass:
			class := g.classes[re]
			if class.Len() == 0 { // such as \P{Any}, which matches nothing as OpNoMatch does
				panic(bitstream.InvalidData("no possible regexp match"))
			}
			r = drawClass(t, class)
		case syntax.OpAnyCharNotNL:
			r = anyRuneGenNoNL.value(t)
		default:
			r = anyRuneGen.value(t)
		}
		_, _ = w.WriteRune(textgen.FoldCase(t.s, r, re.Flags))
	case syntax.OpBeginLine, syntax.OpEndLine,
		syntax.OpBeginText, syntax.OpEndText,
		syntax.OpWordBoundary, syntax.OpNoWordBoundary:
		t.s.DrawBits(0) // do nothing and hope that find() is enough
	case syntax.OpCapture:
		g.build(w, re.Sub[0], t)
	case syntax.OpStar, syntax.OpPlus, syntax.OpQuest, syntax.OpRepeat:
		lo, hi := re.Min, re.Max
		switch re.Op {
		case syntax.OpStar:
			lo, hi = 0, -1
		case syntax.OpPlus:
			lo, hi = 1, -1
		case syntax.OpQuest:
			lo, hi = 0, 1
		}
		repeat := bitstream.NewRepeat(lo, hi, -1, g.labels[re.Sub[0]])
		for repeat.More(t.s) {
			g.build(w, re.Sub[0], t)
		}
	case syntax.OpConcat:
		for _, sub := range re.Sub {
			g.build(w, sub, t)
		}
	case syntax.OpAlternate:
		ix := bitstream.Index(t.s, len(re.Sub), true)
		g.build(w, re.Sub[ix], t)
	default:
		assert.Truef(false, "invalid regexp op %v", re.Op)
	}

	t.s.EndGroup(i, false)
}

// classRunes is RuneFrom of one table, as a character class draws: in a standalone group labelled by
// classRunes, through its die. Classes so shrink as RuneFrom groups do.
var classRunes = newRuneGen(false, nil, make([]*textgen.RuneSet, 1))

func drawClass(t *T, runes *textgen.RuneSet) rune {
	i := t.s.BeginGroup(classRunes, true)
	r := classRunes.draw(t.s, []*textgen.RuneSet{runes})
	t.s.EndGroup(i, false)
	return r
}
