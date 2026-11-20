// Copyright 2020 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"path/filepath"
	"testing"
)

func TestFailFilePattern(t *testing.T) {
	t.Parallel()

	Check(t, func(t *T) {
		testName := String().Draw(t, "testName")
		fileName := failFileName(testName)
		pattern := failFilePattern(testName)
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
			seed     = Uint64().Draw(t, "seed")
			output   = SliceOf(Byte()).Draw(t, "output")
			buf      = SliceOf(Uint64()).Draw(t, "buf")
		)

		fileName := failFileName(testName)
		fileName = filepath.Join(root, fileName)
		err := saveFailFile(fileName, version, output, seed, buf)
		if err != nil {
			t.Fatal(err)
		}

		version2, seed2, buf2, err := loadFailFile(fileName)
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
