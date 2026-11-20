// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package engine

import (
	"fmt"

	"go.sdls.io/rapid/internal/bitstream"
)

// StopTest is the panic value of a property that failed through T.Fatal, T.Error and the like;
// it carries the failure message.
type StopTest string

// TestError is what a failing test case left: the value it panicked with
// (a StopTest, a bitstream.InvalidData or the property's own panic) and the traceback to it.
type TestError struct {
	Data  any
	Trace string
}

func (err *TestError) Error() string {
	if msg, ok := err.Data.(StopTest); ok {
		return string(msg)
	}

	if msg, ok := err.Data.(bitstream.InvalidData); ok {
		return "invalid data: " + string(msg)
	}

	return fmt.Sprintf("%v", err.Data)
}

// IsInvalidData reports whether the test case was discarded rather than failed.
func (err *TestError) IsInvalidData() bool {
	_, ok := err.Data.(bitstream.InvalidData)
	return ok
}

// IsStopTest reports whether the property failed through T.Fatal, T.Error and the like, rather than by panicking.
func (err *TestError) IsStopTest() bool {
	_, ok := err.Data.(StopTest)
	return ok
}

// SameError reports whether err1 and err2 have the same message and traceback; either may be nil.
func SameError(err1, err2 *TestError) bool {
	return ErrorString(err1) == ErrorString(err2) && Traceback(err1) == Traceback(err2)
}

// ErrorString returns err's message, or "" for nil.
func ErrorString(err *TestError) string {
	if err == nil {
		return ""
	}

	return err.Error()
}

// Traceback returns err's traceback, or a placeholder for nil.
func Traceback(err *TestError) string {
	if err == nil {
		return "    <no error>\n"
	}

	return err.Trace
}
