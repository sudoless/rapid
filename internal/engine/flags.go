// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package engine

import (
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

func init() {
	defaults := loadDefaults(os.LookupEnv)
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

func loadDefaults(lookup func(string) (string, bool)) Settings {
	defaults := defaultSettings()

	defaults.Checks = envInt(lookup, "RAPID_CHECKS", defaults.Checks)
	defaults.Steps = envInt(lookup, "RAPID_STEPS", defaults.Steps)
	defaults.FailFile = envString(lookup, "RAPID_FAILFILE", defaults.FailFile)
	defaults.NoFailFile = envBool(lookup, "RAPID_NOFAILFILE", defaults.NoFailFile)
	defaults.Seed = envUint64(lookup, "RAPID_SEED", defaults.Seed)
	defaults.Log = envBool(lookup, "RAPID_LOG", defaults.Log)
	defaults.Verbose = envBool(lookup, "RAPID_V", defaults.Verbose)
	defaults.Debug = envBool(lookup, "RAPID_DEBUG", defaults.Debug)
	defaults.DebugVis = envBool(lookup, "RAPID_DEBUGVIS", defaults.DebugVis)
	defaults.ShrinkTime = envDuration(lookup, "RAPID_SHRINKTIME", defaults.ShrinkTime)

	return defaults
}

func envString(lookup func(string) (string, bool), key, defaultValue string) string {
	if v, ok := lookup(key); ok {
		return v
	}
	return defaultValue
}

func envInt(lookup func(string) (string, bool), key string, defaultValue int) int {
	v, ok := lookup(key)
	if !ok {
		return defaultValue
	}
	n, err := strconv.ParseInt(v, 0, strconv.IntSize)
	if err != nil {
		panicInvalidEnv(key, v, err)
	}
	return int(n)
}

func envBool(lookup func(string) (string, bool), key string, defaultValue bool) bool {
	v, ok := lookup(key)
	if !ok {
		return defaultValue
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		panicInvalidEnv(key, v, err)
	}
	return b
}

func envUint64(lookup func(string) (string, bool), key string, defaultValue uint64) uint64 {
	v, ok := lookup(key)
	if !ok {
		return defaultValue
	}
	n, err := strconv.ParseUint(v, 0, 64)
	if err != nil {
		panicInvalidEnv(key, v, err)
	}
	return n
}

func envDuration(lookup func(string) (string, bool), key string, defaultValue time.Duration) time.Duration {
	v, ok := lookup(key)
	if !ok {
		return defaultValue
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		panicInvalidEnv(key, v, err)
	}
	return d
}

func panicInvalidEnv(key, value string, err error) {
	panic(fmt.Sprintf("[rapid] invalid value for %s=%q: %v", key, value, err))
}
