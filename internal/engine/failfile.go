// Copyright 2020 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package engine

import (
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	persistDirMode     = 0o775
	failfileTmpPattern = ".rapid-failfile-tmp-*"
)

// testFileName returns the name the files of the test testName go by. Runes other than letters, digits,
// '-' and '_' become '_', and the result is cut to a length every file system takes. A hash of the full name
// follows, so names that sanitise alike, such as TestCalc/1+2 and TestCalc/1*2, stay apart.
func testFileName(testName string) string {
	const maxSafeBytes = 100

	var safe strings.Builder
	for _, char := range testName {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) && char != '-' {
			char = '_'
		}
		if safe.Len()+utf8.RuneLen(char) > maxSafeBytes {
			break
		}
		safe.WriteRune(char)
	}

	h := fnv.New32a()
	_, _ = h.Write([]byte(testName)) // a hash.Hash never returns an error
	return fmt.Sprintf("%s-%08x", safe.String(), h.Sum32())
}

// failFileSeq numbers the fail file names this process hands out.
var failFileSeq atomic.Uint64

// FailFileName returns a new fail file path for the test: the time, process ID and a number this process
// never hands out twice make it unique.
func FailFileName(testName string) string {
	name := testFileName(testName)
	ts := time.Now().Format("20060102150405")
	fileName := fmt.Sprintf("%s-%s-%d-%d.fail", name, ts, os.Getpid(), failFileSeq.Add(1))
	return filepath.Join("testdata", "rapid", name, fileName)
}

// FailFilePattern returns the glob that matches every fail file FailFileName returns for the test.
func FailFilePattern(testName string) string {
	name := testFileName(testName)
	return filepath.Join("testdata", "rapid", name, name+"-*.fail")
}

// SaveFailFile writes the failing test case buf, drawn from seed, to filename, with output as comments.
// version identifies the format, so a later rapid can skip files it cannot replay.
func SaveFailFile(filename, version string, output []byte, seed uint64, buf []uint64) error {
	dir := filepath.Dir(filename)
	err := os.MkdirAll(dir, persistDirMode)
	if err != nil {
		return fmt.Errorf("failed to create directory for fail file %q: %w", filename, err)
	}

	f, err := os.CreateTemp(dir, failfileTmpPattern)
	if err != nil {
		return fmt.Errorf("failed to create temporary file for fail file %q: %w", filename, err)
	}
	defer func() { _ = os.Remove(f.Name()) }()
	defer func() { _ = f.Close() }()

	for line := range strings.SplitSeq(string(output), "\n") {
		_, err := f.WriteString("# " + line + "\n")
		if err != nil {
			return fmt.Errorf("failed to write data to fail file %q: %w", filename, err)
		}
	}

	bs := []string{fmt.Sprintf("%v#%v", version, seed)}
	for _, word := range buf {
		bs = append(bs, fmt.Sprintf("0x%x", word))
	}

	_, err = f.WriteString(strings.Join(bs, "\n"))
	if err != nil {
		return fmt.Errorf("failed to write data to fail file %q: %w", filename, err)
	}

	_ = f.Close() // early close, otherwise os.Rename will fail on Windows
	err = os.Rename(f.Name(), filename)
	if err != nil {
		return fmt.Errorf("failed to save fail file %q: %w", filename, err)
	}

	return nil
}

// LoadFailFile reads back what SaveFailFile wrote to filename: the version, seed and test case.
func LoadFailFile(filename string) (string, uint64, []uint64, error) {
	content, err := os.ReadFile(filename) //nolint:gosec // G304: the fail file is named by -rapid.failfile or the test's own fail-file pattern
	if err != nil {
		return "", 0, nil, fmt.Errorf("failed to open fail file: %w", err)
	}

	var data []string
	for line := range strings.Lines(string(content)) {
		s := strings.TrimSpace(line)
		if strings.HasPrefix(s, "#") || s == "" {
			continue
		}
		data = append(data, s)
	}

	if len(data) == 0 {
		return "", 0, nil, fmt.Errorf("no data in fail file %q", filename)
	}

	version, seedField, ok := strings.Cut(data[0], "#")
	if !ok {
		return "", 0, nil, fmt.Errorf("invalid version/seed field %q in %q", data[0], filename)
	}
	seed, err := strconv.ParseUint(seedField, 10, 64)
	if err != nil {
		return "", 0, nil, fmt.Errorf("invalid seed %q in %q", seedField, filename)
	}

	var buf []uint64
	for _, field := range data[1:] {
		u, err := strconv.ParseUint(field, 0, 64)
		if err != nil {
			return "", 0, nil, fmt.Errorf("failed to load fail file %q: %w", filename, err)
		}
		buf = append(buf, u)
	}

	return version, seed, buf, nil
}
