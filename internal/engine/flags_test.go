// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package engine

import (
	"testing"
	"time"
)

func TestLoadCmdlineDefaultsUsesBaseValues(t *testing.T) {
	t.Parallel()

	got := loadDefaults(func(string) (string, bool) {
		return "", false
	})
	want := defaultSettings()

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

	got := loadDefaults(func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	})

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

func TestLoadCmdlineDefaultsInvalidEnvPanics(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for invalid env value")
		}
	}()

	loadDefaults(func(string) (string, bool) {
		return "not-an-int", true
	})
}

func BenchmarkBaseSeed(b *testing.B) {
	for b.Loop() {
		BaseSeed()
	}
}
