// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package engine

import (
	"errors"
	"flag"
	"fmt"
	"hash/maphash"
	"os"
	"strconv"
	"time"
)

// Settings are the values of the -rapid.* command-line flags.
type Settings struct {
	Checks     int
	Steps      int
	FailFile   string
	NoFailFile bool
	Seed       uint64
	Log        bool
	Verbose    bool
	Debug      bool
	DebugVis   bool
	ShrinkTime time.Duration
}

// Flags holds the -rapid.* flags, registered on the default flag set when the package initializes;
// the matching RAPID_* environment variables set their defaults.
var Flags Settings

// errEnv says which RAPID_* variables could not be parsed, and so kept their default.
// Check and Fuzz fail with it, rather than the package failing to initialize.
var errEnv error

func init() {
	var defaults Settings
	defaults, errEnv = loadDefaults(os.LookupEnv)
	flag.IntVar(&Flags.Checks, "rapid.checks", defaults.Checks, "rapid: number of checks to perform")
	flag.IntVar(&Flags.Steps, "rapid.steps", defaults.Steps, "rapid: average number of Repeat actions to execute")
	flag.StringVar(&Flags.FailFile, "rapid.failfile", defaults.FailFile, "rapid: fail file to use to reproduce test failure")
	flag.BoolVar(&Flags.NoFailFile, "rapid.nofailfile", defaults.NoFailFile, "rapid: do not write fail files on test failures")
	flag.Uint64Var(&Flags.Seed, "rapid.seed", defaults.Seed, "rapid: PRNG seed to start with (0 to use a random one)")
	flag.BoolVar(&Flags.Log, "rapid.log", defaults.Log, "rapid: eager verbose output to stdout (to aid with unrecoverable test failures)")
	flag.BoolVar(&Flags.Verbose, "rapid.v", defaults.Verbose, "rapid: verbose output")
	flag.BoolVar(&Flags.Debug, "rapid.debug", defaults.Debug, "rapid: debugging output")
	flag.BoolVar(&Flags.DebugVis, "rapid.debugvis", defaults.DebugVis, "rapid: debugging visualization")
	flag.DurationVar(&Flags.ShrinkTime, "rapid.shrinktime", defaults.ShrinkTime, "rapid: maximum time to spend on test case minimization")
}

// BaseSeed returns -rapid.seed when set, and a random seed otherwise.
func BaseSeed() uint64 {
	if Flags.Seed != 0 {
		return Flags.Seed
	}

	return new(maphash.Hash).Sum64()
}

func defaultSettings() Settings {
	return Settings{
		Checks:     100,
		Steps:      30,
		ShrinkTime: 30 * time.Second,
	}
}

// loadDefaults returns the settings the RAPID_* variables lookup finds give. A variable that does not parse
// keeps its default, and the error says which and why.
func loadDefaults(lookup func(string) (string, bool)) (Settings, error) {
	defaults := defaultSettings()
	env := envReader{lookup: lookup}

	defaults.Checks = envValue(&env, "RAPID_CHECKS", defaults.Checks, parseInt)
	defaults.Steps = envValue(&env, "RAPID_STEPS", defaults.Steps, parseInt)
	defaults.FailFile = envValue(&env, "RAPID_FAILFILE", defaults.FailFile, func(s string) (string, error) { return s, nil })
	defaults.NoFailFile = envValue(&env, "RAPID_NOFAILFILE", defaults.NoFailFile, strconv.ParseBool)
	defaults.Seed = envValue(&env, "RAPID_SEED", defaults.Seed, func(s string) (uint64, error) { return strconv.ParseUint(s, 0, 64) })
	defaults.Log = envValue(&env, "RAPID_LOG", defaults.Log, strconv.ParseBool)
	defaults.Verbose = envValue(&env, "RAPID_V", defaults.Verbose, strconv.ParseBool)
	defaults.Debug = envValue(&env, "RAPID_DEBUG", defaults.Debug, strconv.ParseBool)
	defaults.DebugVis = envValue(&env, "RAPID_DEBUGVIS", defaults.DebugVis, strconv.ParseBool)
	defaults.ShrinkTime = envValue(&env, "RAPID_SHRINKTIME", defaults.ShrinkTime, time.ParseDuration)

	return defaults, errors.Join(env.errs...)
}

// envReader reads RAPID_* variables through lookup, and collects the errors of those that do not parse.
type envReader struct {
	lookup func(string) (string, bool)
	errs   []error
}

// envValue returns the variable key parsed by parse, or defaultValue when key is unset or does not parse.
func envValue[V any](env *envReader, key string, defaultValue V, parse func(string) (V, error)) V {
	s, ok := env.lookup(key)
	if !ok {
		return defaultValue
	}
	v, err := parse(s)
	if err != nil {
		env.errs = append(env.errs, fmt.Errorf("parsing %s=%q: %w", key, s, err))
		return defaultValue
	}
	return v
}

func parseInt(s string) (int, error) {
	n, err := strconv.ParseInt(s, 0, strconv.IntSize)
	return int(n), err
}
