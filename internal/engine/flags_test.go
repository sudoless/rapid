// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package engine

import (
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestLoadCmdlineDefaultsUsesBaseValues(t *testing.T) {
	t.Parallel()

	got, errs := loadDefaults(func(string) (string, bool) {
		return "", false
	})
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	want := Settings{Checks: 100, Steps: 30, ShrinkTime: 30 * time.Second}

	if got != want {
		t.Fatalf("defaults mismatch: got %+v, want %+v", got, want)
	}
}

func TestLoadCmdlineDefaultsFromEnv(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"RAPID_CHECKS":     "0xc8",
		"RAPID_STEPS":      "40_000",
		"RAPID_FAILFILE":   "/tmp/failfile",
		"RAPID_NOFAILFILE": "true",
		"RAPID_SEED":       "0x1234",
		"RAPID_LOG":        "true",
		"RAPID_V":          "true",
		"RAPID_DEBUG":      "true",
		"RAPID_DEBUGVIS":   "true",
		"RAPID_SHRINKTIME": "45s",
	}

	got, errs := loadDefaults(func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	})
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	if got.Checks != 200 {
		t.Fatalf("checks: got %d, want %d", got.Checks, 200)
	}
	if got.Steps != 40000 {
		t.Fatalf("steps: got %d, want %d", got.Steps, 40000)
	}
	if got.FailFile != "/tmp/failfile" {
		t.Fatalf("failfile: got %q, want %q", got.FailFile, "/tmp/failfile")
	}
	if !got.NoFailFile {
		t.Fatalf("nofailfile: expected true")
	}
	if got.Seed != 0x1234 {
		t.Fatalf("seed: got %d, want %d", got.Seed, 0x1234)
	}
	if !got.Log || !got.Verbose || !got.Debug || !got.DebugVis {
		t.Fatalf("expected all bool flags true, got %+v", got)
	}
	if got.ShrinkTime != 45*time.Second {
		t.Fatalf("shrinkTime: got %v, want %v", got.ShrinkTime, 45*time.Second)
	}
}

// A variable that does not parse keeps its default, and its error names it and its flag; the others still apply.
func TestLoadCmdlineDefaultsInvalidEnv(t *testing.T) {
	t.Parallel()

	env := map[string]string{"RAPID_CHECKS": "abc", "RAPID_V": "maybe", "RAPID_STEPS": "40"}
	got, errs := loadDefaults(func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	})

	if got.Checks != 100 || got.Verbose || got.Steps != 40 {
		t.Fatalf("got checks %v, verbose %v, steps %v; want 100, false, 40", got.Checks, got.Verbose, got.Steps)
	}
	want := map[string]string{"rapid.checks": "RAPID_CHECKS", "rapid.v": "RAPID_V"}
	if len(errs) != len(want) {
		t.Fatalf("errors %v, want one for each of %v", errs, want)
	}
	for _, e := range errs {
		key, ok := want[e.flag]
		if !ok || !strings.Contains(e.err.Error(), key) || !errors.Is(e.err, strconv.ErrSyntax) {
			t.Errorf("error %v of flag %v; want flags %v, each error naming its variable and wrapping %v", e.err, e.flag, want, strconv.ErrSyntax)
		}
	}
}

func BenchmarkBaseSeed(b *testing.B) {
	for b.Loop() {
		BaseSeed()
	}
}
