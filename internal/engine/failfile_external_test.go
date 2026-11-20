// Copyright 2020 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package engine_test

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

// Go keeps these subtest names distinct, so each test replays only its own fail files.
func TestFailFilePatternSeparatesDistinctTestNames(t *testing.T) {
	t.Parallel()

	names := []string{
		"TestCalc/1+2", "TestCalc/1*2", "TestCalc/1.2", "TestCalc/1_2", "TestCalc_1/2",
		"TestCalc/)@+$+#(@!#,:", "TestCalc/.((.(#?!)*$(", // sanitise alike, and share a 32-bit FNV-1a hash
	}
	for _, name := range names {
		for _, other := range names {
			if name == other {
				continue
			}
			pattern := engine.FailFilePattern(name)
			fileName := engine.FailFileName(other)
			if match, err := filepath.Match(pattern, fileName); match || err != nil {
				t.Errorf("fail file pattern of %q matches fail file %q of %q; err %v", name, fileName, other, err)
			}
		}
	}
}

// Two failures of one test saved in quick succession, as with -count, keep a fail file each.
func TestFailFileNameUnique(t *testing.T) {
	t.Parallel()

	if a, b := engine.FailFileName("TestUnique"), engine.FailFileName("TestUnique"); a == b {
		t.Fatalf("two fail files of one test share the name %q", a)
	}
}

// A test name far longer than a file name may be still names a fail file that can be saved.
func TestFailFileNameLongTestName(t *testing.T) {
	t.Parallel()

	testName := "TestLong/" + strings.Repeat("ä", 300)
	fileName := filepath.Join(t.TempDir(), engine.FailFileName(testName))
	if err := engine.SaveFailFile(fileName, "v1", nil, 1, []uint64{1}); err != nil {
		t.Fatalf("saving the fail file of a long test name: %v", err)
	}
	if match, err := filepath.Match(filepath.Join(filepath.Dir(fileName), filepath.Base(engine.FailFilePattern(testName))), fileName); !match || err != nil {
		t.Fatalf("pattern does not match %q; err %v", fileName, err)
	}
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

// A failing test case may log a line far longer than a bufio.Scanner token, such as a large drawn value.
func TestFailFileRoundtripLongOutputLine(t *testing.T) {
	t.Parallel()

	fileName := filepath.Join(t.TempDir(), "long.fail")
	output := []byte(strings.Repeat("x", 1<<20))
	if err := engine.SaveFailFile(fileName, "v1", output, 42, []uint64{1}); err != nil {
		t.Fatal(err)
	}

	version, seed, buf, err := engine.LoadFailFile(fileName)
	if err != nil {
		t.Fatalf("loading a fail file with a 1 MiB output line: %v", err)
	}
	if version != "v1" || seed != 42 || len(buf) != 1 || buf[0] != 1 {
		t.Fatalf("got %q, %v, %v; want \"v1\", 42, [1]", version, seed, buf)
	}
}

func TestLoadFailFileHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		header  string
		wantErr bool
		cause   error // the error a failure wraps, when it has one
	}{
		{"version and seed", "v1#42", false, nil},
		{"no separator", "v1-42", true, nil},
		{"seed not a number", "v1#x", true, strconv.ErrSyntax},
		{"two separators", "v1#4#2", true, strconv.ErrSyntax},
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
			if tc.cause != nil && !errors.Is(err, tc.cause) {
				t.Fatalf("err = %v, want it to wrap %v", err, tc.cause)
			}
			if err == nil && (version != "v1" || seed != 42 || len(buf) != 1 || buf[0] != 1) {
				t.Fatalf("got %q, %v, %v; want \"v1\", 42, [1]", version, seed, buf)
			}
		})
	}
}
