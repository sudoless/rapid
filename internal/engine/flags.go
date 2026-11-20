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
	"strings"
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

// envErrs are the RAPID_* variables that could not be parsed, and so kept their default; EnvErr reports them.
var envErrs []envError

func init() {
	var defaults Settings
	defaults, envErrs = loadDefaults(os.LookupEnv)
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

// EnvErr says which RAPID_* variables could not be parsed, and so left the default of their flag. A variable
// whose -rapid.* flag is set is left out: the flag overrides it.
func EnvErr() error {
	if len(envErrs) == 0 {
		return nil
	}

	set := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { set[f.Name] = true })

	var errs []error
	for i := range envErrs {
		if !set[envErrs[i].flag] {
			errs = append(errs, envErrs[i].err)
		}
	}
	return errors.Join(errs...)
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
// keeps its default, and is returned with why.
func loadDefaults(lookup func(string) (string, bool)) (Settings, []envError) {
	defaults := defaultSettings()
	env := envReader{lookup: lookup}

	defaults.Checks = envValue(&env, "rapid.checks", defaults.Checks, parseInt)
	defaults.Steps = envValue(&env, "rapid.steps", defaults.Steps, parseInt)
	defaults.FailFile = envValue(&env, "rapid.failfile", defaults.FailFile, func(s string) (string, error) { return s, nil })
	defaults.NoFailFile = envValue(&env, "rapid.nofailfile", defaults.NoFailFile, strconv.ParseBool)
	defaults.Seed = envValue(&env, "rapid.seed", defaults.Seed, func(s string) (uint64, error) { return strconv.ParseUint(s, 0, 64) })
	defaults.Log = envValue(&env, "rapid.log", defaults.Log, strconv.ParseBool)
	defaults.Verbose = envValue(&env, "rapid.v", defaults.Verbose, strconv.ParseBool)
	defaults.Debug = envValue(&env, "rapid.debug", defaults.Debug, strconv.ParseBool)
	defaults.DebugVis = envValue(&env, "rapid.debugvis", defaults.DebugVis, strconv.ParseBool)
	defaults.ShrinkTime = envValue(&env, "rapid.shrinktime", defaults.ShrinkTime, time.ParseDuration)

	return defaults, env.errs
}

// envError is a RAPID_* variable that does not parse.
type envError struct {
	flag string // the -rapid.* flag the variable sets the default of
	err  error
}

// envReader reads RAPID_* variables through lookup, and collects those that do not parse.
type envReader struct {
	lookup func(string) (string, bool)
	errs   []envError
}

// envValue returns the variable of the flag name, parsed by parse, or defaultValue when the variable is unset
// or does not parse. The variable of rapid.checks is RAPID_CHECKS.
func envValue[V any](env *envReader, name string, defaultValue V, parse func(string) (V, error)) V {
	key := strings.ToUpper(strings.ReplaceAll(name, ".", "_"))
	s, ok := env.lookup(key)
	if !ok {
		return defaultValue
	}
	v, err := parse(s)
	if err != nil {
		env.errs = append(env.errs, envError{flag: name, err: fmt.Errorf("parsing %s=%q: %w", key, s, err)})
		return defaultValue
	}
	return v
}

func parseInt(s string) (int, error) {
	n, err := strconv.ParseInt(s, 0, strconv.IntSize)
	return int(n), err
}
