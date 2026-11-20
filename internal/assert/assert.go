// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package assert panics on broken internal invariants: a bug in rapid, or a generator misused.
package assert

import "fmt"

// True panics when ok is false.
func True(ok bool) {
	if !ok {
		panic("assertion failed")
	}
}

// Truef panics with the formatted message when ok is false.
func Truef(ok bool, format string, args ...any) {
	if !ok {
		panic(fmt.Sprintf(format, args...))
	}
}
