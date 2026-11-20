// Copyright (c) 2026 SUDOLESS SRL
// SPDX-License-Identifier: BSD-3-Clause-Clear

package engine

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// coverChecksMult bounds the valid test cases a search runs to meet Limits.Cover, as a multiple of Checks.
const coverChecksMult = 10

// Events counts, for each event, the valid test cases that marked it.
type Events map[string]int

// add counts one more test case for each of names.
func (e *Events) add(names []string) {
	if len(names) == 0 {
		return
	}
	if *e == nil {
		*e = Events{}
	}
	for _, name := range names {
		(*e)[name]++
	}
}

// covers reports whether each event in cover was marked by at least as many test cases as cover asks for.
func (e Events) covers(cover map[string]int) bool {
	for name, want := range cover {
		if e[name] < want {
			return false
		}
	}
	return true
}

// table returns e as lines to append to a message about valid test cases: one per event, the most marked
// first, each event in cover with what it asks for. It returns "" when there is no event to show.
func (e Events) table(valid int, cover map[string]int) string {
	names := slices.AppendSeq(slices.Collect(maps.Keys(e)), maps.Keys(cover))
	slices.SortFunc(names, func(a, b string) int {
		return cmp.Or(cmp.Compare(e[b], e[a]), strings.Compare(a, b))
	})
	names = slices.Compact(names)
	if len(names) == 0 {
		return ""
	}

	b := &strings.Builder{}
	fmt.Fprintf(b, "\n    events, of %v valid tests:", valid)
	width := len(strconv.Itoa(valid))
	for _, name := range names {
		pct := 0.0
		if valid > 0 {
			pct = float64(e[name]) / float64(valid) * 100
		}
		fmt.Fprintf(b, "\n    %*d  %5.1f%%  %s", width, e[name], pct, name)
		if n, ok := cover[name]; ok {
			fmt.Fprintf(b, " (cover %v)", n)
		}
	}
	return b.String()
}

// coverFailure says why the search res ended with lim.Cover unmet.
func coverFailure(res *Result, lim *Limits) string {
	switch {
	case res.EarlyExit:
		return fmt.Sprintf("Cover not met: the search reached its time limit after %v valid tests", res.Valid)
	case res.Valid < max(lim.Checks, lim.CoverChecks):
		return fmt.Sprintf("Cover not met: the search stopped at %v invalid tests, after %v valid ones", res.Invalid, res.Valid)
	default:
		return fmt.Sprintf("Cover not met after %v valid tests", res.Valid)
	}
}
