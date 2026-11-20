// Copyright (c) 2026 SUDOLESS SRL
// SPDX-License-Identifier: BSD-3-Clause-Clear

package rapid

import (
	"time"

	"go.sdls.io/rapid/internal/assert"
	"go.sdls.io/rapid/internal/engine"
)

// CheckOption sets how [Check] and [MakeCheck] run one property, in place of the -rapid.* flag or RAPID_*
// variable that sets it for every property. When options set the same thing, the last one wins.
//
// A slice of options shared by a group of tests gives them a common default:
//
//	var e2e = []rapid.CheckOption{rapid.Checks(20), rapid.ShrinkTime(10 * time.Second)}
//
//	rapid.Check(t, prop, e2e...)
type CheckOption struct {
	set func(*engine.Limits)
}

// Checks makes the property run on n valid test cases, whatever -rapid.checks says, and on more when [Cover]
// asks for them. Under -short it runs on a fifth of them, and on at least one. Checks panics if n < 1.
func Checks(n int) CheckOption {
	assert.Truef(n >= 1, "Checks(%v): want at least 1", n)
	return CheckOption{func(lim *engine.Limits) { lim.Checks = n }}
}

// SearchTime stops generating test cases for the property after d, so an expensive property is bounded by
// time as well as by count. The property passes if at least one valid test case ran, and [Cover] is met.
// Neither replaying fail files nor shrinking a failure counts towards d. SearchTime panics if d <= 0.
func SearchTime(d time.Duration) CheckOption {
	assert.Truef(d > 0, "SearchTime(%v): want a positive duration", d)
	return CheckOption{func(lim *engine.Limits) { lim.SearchTime = d }}
}

// ShrinkTime bounds how long shrinking a failure of the property takes, whatever -rapid.shrinktime says.
// ShrinkTime panics if d <= 0.
func ShrinkTime(d time.Duration) CheckOption {
	assert.Truef(d > 0, "ShrinkTime(%v): want a positive duration", d)
	return CheckOption{func(lim *engine.Limits) { lim.ShrinkTime = d }}
}

// checkLimits returns the limits opts set, leaving the others zero for the engine to take from the flags.
func checkLimits(opts []CheckOption) engine.Limits {
	var lim engine.Limits
	for _, opt := range opts {
		opt.set(&lim)
	}
	return lim
}
