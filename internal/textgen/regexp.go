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
)

// Regexp holds a regular expression in both forms string generation uses:
// the syntax tree it generates strings from, and the matcher it checks them with.
type Regexp struct {
	Syntax  *syntax.Regexp
	Matcher *regexp.Regexp
}

var (
	compiledRegexps = sync.Map{} // expr -> Regexp
	regexpNames     = sync.Map{} // *syntax.Regexp -> string
)

// CompileRegexp parses and compiles expr, once per expr.
func CompileRegexp(expr string) (Regexp, error) {
	cached, ok := compiledRegexps.Load(expr)
	if ok {
		return cached.(Regexp), nil
	}

	syn, err := syntax.Parse(expr, syntax.Perl)
	if err != nil {
		return Regexp{}, fmt.Errorf("failed to parse regexp %q: %w", expr, err)
	}

	re, err := regexp.Compile(expr)
	if err != nil {
		return Regexp{}, fmt.Errorf("failed to compile regexp %q: %w", expr, err)
	}

	ret := Regexp{syn, re}
	compiledRegexps.Store(expr, ret)

	return ret, nil
}

// RegexpName returns re.String(), computed once per re.
func RegexpName(re *syntax.Regexp) string {
	cached, ok := regexpNames.Load(re)
	if ok {
		return cached.(string)
	}

	s := re.String()
	regexpNames.Store(re, s)

	return s
}
