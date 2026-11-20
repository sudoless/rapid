// Copyright 2020 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package engine

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	persistDirMode     = 0o775
	failfileTmpPattern = ".rapid-failfile-tmp-*"
)

// https://learn.microsoft.com/en-us/windows/win32/fileio/naming-a-file
var windowsReservedNames = []string{
	"CON", "PRN", "AUX", "NUL",
	"COM0", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "COM¹", "COM²", "COM³",
	"LPT0", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9", "LPT¹", "LPT²", "LPT³",
}

func kindaSafeFilename(f string) string {
	var s strings.Builder
	for _, char := range f {
		if unicode.IsLetter(char) || unicode.IsDigit(char) || char == '-' || char == '_' {
			s.WriteRune(char)
		} else {
			s.WriteRune('_')
		}
	}
	name := s.String()
	nameUpper := strings.ToUpper(name)
	if slices.Contains(windowsReservedNames, nameUpper) {
		return name + "_"
	}
	return name
}

// FailFileName returns a new fail file path for the test, unique to this second and process.
func FailFileName(testName string) string {
	ts := time.Now().Format("20060102150405")
	fileName := fmt.Sprintf("%s-%s-%d.fail", kindaSafeFilename(testName), ts, os.Getpid())
	return filepath.Join("testdata", "rapid", kindaSafeFilename(testName), fileName)
}

// FailFilePattern returns the glob that matches every fail file FailFileName returns for the test.
func FailFilePattern(testName string) string {
	fileName := kindaSafeFilename(testName) + "-*.fail"
	dirName := filepath.Join("testdata", "rapid", kindaSafeFilename(testName))
	return filepath.Join(dirName, fileName)
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
	f, err := os.Open(filename) //nolint:gosec // G304: the fail file is named by -rapid.failfile or the test's own fail-file pattern
	if err != nil {
		return "", 0, nil, fmt.Errorf("failed to open fail file: %w", err)
	}
	defer func() { _ = f.Close() }()

	var data []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		s := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(s, "#") || s == "" {
			continue
		}
		data = append(data, s)
	}
	if err := scanner.Err(); err != nil {
		return "", 0, nil, fmt.Errorf("failed to load fail file %q: %w", filename, err)
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
