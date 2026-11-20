// Copyright (c) 2026 SUDOLESS SRL
// SPDX-License-Identifier: BSD-3-Clause-Clear

package textgen

import (
	"fmt"
	"testing"
)

// Regexps built from drawn data do not grow the compile cache past its bound, and a regexp compiled again
// soon after is served from it.
func TestCompileRegexpCacheBounded(t *testing.T) {
	t.Parallel()

	for idx := range maxCompiled + 10 {
		if _, err := CompileRegexp(fmt.Sprintf("[^%c]x%d", rune('a'+idx%26), idx)); err != nil {
			t.Fatal(err)
		}
	}

	compiled.mu.Lock()
	size := len(compiled.byExpr)
	compiled.mu.Unlock()
	if size > maxCompiled {
		t.Fatalf("the cache holds %v regexps, more than %v", size, maxCompiled)
	}

	first, err := CompileRegexp("[a-z]+cached")
	if err != nil {
		t.Fatal(err)
	}
	again, err := CompileRegexp("[a-z]+cached")
	if err != nil {
		t.Fatal(err)
	}
	if first.Syntax != again.Syntax {
		t.Fatalf("a regexp compiled twice in a row was not served from the cache")
	}
}
