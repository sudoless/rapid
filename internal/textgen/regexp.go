// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package textgen

import (
	"fmt"
	"regexp"
	"regexp/syntax"
	"sync"

	"go.sdls.io/rapid/internal/bitstream"
)

// Regexp holds a regular expression in the forms string generation uses: the syntax tree it generates
// strings from, the matcher it checks them with, and what generation needs at nodes of the tree.
type Regexp struct {
	Syntax  *syntax.Regexp
	Matcher *regexp.Regexp
	Labels  map[*syntax.Regexp]fmt.Stringer // the label of each node a repetition repeats: its regexp text
	Classes map[*syntax.Regexp]*RuneSet     // the runes of each character class
}

// maxCompiled bounds the regexps CompileRegexp keeps, so a property that builds regexps from drawn data
// does not grow the cache without end; a full cache is emptied and starts over.
const maxCompiled = 1024

var compiled struct {
	mu     sync.Mutex
	byExpr map[string]Regexp
}

// CompileRegexp parses and compiles expr. It keeps the result for later calls with the same expr,
// up to maxCompiled regexps.
func CompileRegexp(expr string) (Regexp, error) {
	compiled.mu.Lock()
	cached, ok := compiled.byExpr[expr]
	compiled.mu.Unlock()
	if ok {
		return cached, nil
	}

	syn, err := syntax.Parse(expr, syntax.Perl)
	if err != nil {
		return Regexp{}, fmt.Errorf("parsing regexp %q: %w", expr, err)
	}

	re, err := regexp.Compile(expr)
	if err != nil {
		return Regexp{}, fmt.Errorf("compiling regexp %q: %w", expr, err)
	}

	ret := Regexp{
		Syntax:  syn,
		Matcher: re,
		Labels:  map[*syntax.Regexp]fmt.Stringer{},
		Classes: map[*syntax.Regexp]*RuneSet{},
	}
	ret.index(syn)

	compiled.mu.Lock()
	defer compiled.mu.Unlock()
	if len(compiled.byExpr) >= maxCompiled || compiled.byExpr == nil {
		compiled.byExpr = make(map[string]Regexp, maxCompiled)
	}
	compiled.byExpr[expr] = ret

	return ret, nil
}

// index records the label of every repeated node under re, and the runes of every character class.
func (r *Regexp) index(re *syntax.Regexp) {
	switch re.Op {
	case syntax.OpCharClass:
		r.Classes[re] = ClassRunes(re)
	case syntax.OpStar, syntax.OpPlus, syntax.OpQuest, syntax.OpRepeat:
		r.Labels[re.Sub[0]] = bitstream.Label(re.Sub[0].String())
	}
	for _, sub := range re.Sub {
		r.index(sub)
	}
}
