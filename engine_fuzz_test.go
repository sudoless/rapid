// Copyright 2022 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid_test

import (
	"context"
	"testing"

	. "go.sdls.io/rapid"
)

func FuzzContext(f *testing.F) {
	type key struct{}

	var ctx context.Context
	f.Fuzz(MakeFuzz(func(t *T) {
		// Assign to outer variable
		// so we can check it after the fuzzing.
		ctx = context.WithValue(t.Context(), key{}, "value")
		if err := ctx.Err(); err != nil {
			t.Fatalf("context must be valid: %v", err)
		}
	}))

	// ctx is set only if the fuzzing function was called.
	if ctx != nil {
		if err := ctx.Err(); err == nil {
			f.Fatalf("context must be canceled")
		}

		if want, got := "value", ctx.Value(key{}); want != got {
			f.Fatalf("context must have value %q, got %q", want, got)
		}
	}
}

func FuzzCleanup(f *testing.F) {
	var state []bool
	f.Fuzz(MakeFuzz(func(t *T) {
		idx := len(state)
		state = append(state, false)
		t.Cleanup(func() {
			state[idx] = true
		})
	}))

	for _, ok := range state {
		if !ok {
			f.Fatalf("cleanup must be called")
		}
	}
}
