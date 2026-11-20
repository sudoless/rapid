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

// table returns e as lines to append to a message about valid test cases: one per event, the most marked
// first. It returns "" when there is no event to show.
func (e Events) table(valid int) string {
	names := slices.Collect(maps.Keys(e))
	slices.SortFunc(names, func(a, b string) int {
		return cmp.Or(cmp.Compare(e[b], e[a]), strings.Compare(a, b))
	})
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
	}
	return b.String()
}
