// Copyright 2020 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package engine_test

import (
	"os"
	"path/filepath"
	"testing"

	. "go.sdls.io/rapid"
	"go.sdls.io/rapid/internal/engine"
)

func TestFailFilePattern(t *testing.T) {
	t.Parallel()

	Check(t, func(t *T) {
		testName := String().Draw(t, "testName")
		fileName := engine.FailFileName(testName)
		pattern := engine.FailFilePattern(testName)
		match, err := filepath.Match(pattern, fileName)
		if !match || err != nil {
			t.Fatalf("pattern %q does not match %q; err %v", pattern, fileName, err)
		}
	})
}

func TestFailFileRoundtrip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	Check(t, func(t *T) {
		var (
			// OS X seems to have issues with Go 1.16 and String(), reporting "illegal byte sequence" when trying to rename the file
			testName = StringMatching(`[a-zA-Z0-9._-]+`).Draw(t, "testName")
			version  = StringMatching(`[a-zA-Z0-9._-]+`).Draw(t, "version")
			seed     = Int[uint64]().Draw(t, "seed")
			output   = SliceOf(Int[byte]()).Draw(t, "output")
			buf      = SliceOf(Int[uint64]()).Draw(t, "buf")
		)

		fileName := engine.FailFileName(testName)
		fileName = filepath.Join(root, fileName)
		err := engine.SaveFailFile(fileName, version, output, seed, buf)
		if err != nil {
			t.Fatal(err)
		}

		version2, seed2, buf2, err := engine.LoadFailFile(fileName)
		if err != nil {
			t.Fatal(err)
		}

		if version2 != version {
			t.Fatalf("got version %q instead of %q", version2, version)
		}
		if seed2 != seed {
			t.Fatalf("got seed %v instead of %v", seed2, seed)
		}
		if len(buf2) != len(buf) {
			t.Fatalf("got buf of length %v instead of %v", len(buf2), len(buf))
		}
		for idx, word := range buf {
			if buf2[idx] != word {
				t.Fatalf("got %v instead of %v at %v", buf2[idx], word, idx)
			}
		}
	})
}

func TestLoadFailFileHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		header  string
		wantErr bool
	}{
		{"version and seed", "v1#42", false},
		{"no separator", "v1-42", true},
		{"seed not a number", "v1#x", true},
		{"two separators", "v1#4#2", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fileName := filepath.Join(t.TempDir(), "fail")
			if err := os.WriteFile(fileName, []byte(tc.header+"\n0x1\n"), 0o600); err != nil {
				t.Fatal(err)
			}

			version, seed, buf, err := engine.LoadFailFile(fileName)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, want error %v", err, tc.wantErr)
			}
			if err == nil && (version != "v1" || seed != 42 || len(buf) != 1 || buf[0] != 1) {
				t.Fatalf("got %q, %v, %v; want \"v1\", 42, [1]", version, seed, buf)
			}
		})
	}
}
