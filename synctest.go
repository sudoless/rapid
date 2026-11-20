// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"testing"
	"testing/synctest"

	"go.sdls.io/rapid/internal/engine"
)

// SyncTest runs prop inside a [testing/synctest] bubble, as part of t's test case.
//
// prop receives a T of its own for the bubble. Its Context and Cleanup belong to the bubble, and its
// cleanups run before the bubble ends, so inside prop use that T rather than t. A failure prop records
// with Error, Errorf or Fail fails t's test case; Fatal, FailNow, Skip or a panic in prop stops it.
// Values prop draws count as drawn by t. SyncTest needs a [*testing.T] behind t.
func (t *T) SyncTest(prop func(*T)) {
	t.Helper()

	testT, ok := underlyingTestingT(t.tb)
	if !ok {
		t.Fatalf("[rapid] SyncTest requires a *testing.T backing the current rapid test")
		return
	}

	t.syncTest(testT, prop)
}

func (t *T) syncTest(parent *testing.T, prop func(*T)) {
	// synctest.Test turns a failure inside the bubble into parent.FailNow (runtime.Goexit), which would bypass
	// rapid's panic-based failure capture and shrinking. Run the bubble in a goroutine of its own, catch what
	// stops prop inside it, and panic with it outside, so checkOnce can shrink and save it as usual.
	resultCh := make(chan *engine.TestError, 1)
	draws := t.draws

	go func() {
		var captured *engine.TestError
		returned := false
		defer func() {
			if r := recover(); r != nil {
				captured = panicToError(r)
			} else if !returned && captured == nil {
				captured = panicToError(engine.StopTest("[rapid] SyncTest aborted via testing.FailNow"))
			}
			resultCh <- captured
		}()

		synctest.Test(parent, func(st *testing.T) {
			st.Helper()

			bubble := t.child(st, t.tbLog, t.rawLog)
			bubble.draws = draws
			bubble.onDraw = t.onDraw
			defer func() {
				panicValue := recover()
				// Always run the bubble's cleanups, even if the property panicked.
				if r := bubble.cleanupRecovered(); r != nil {
					panicValue = r
				}
				draws = bubble.draws

				if panicValue != nil {
					captured = panicToError(panicValue)
				}
			}()

			prop(bubble)
		})

		returned = true
	}()

	err := <-resultCh
	t.draws = draws
	if err != nil {
		panic(err)
	}
}

// cleanupRecovered runs t's cleanups and returns the value of a panic they raised, or nil.
func (t *T) cleanupRecovered() (panicValue any) {
	defer func() { panicValue = recover() }()
	t.cleanup()

	return nil
}

// underlyingTestingT returns the *testing.T associated with tb, if any.
func underlyingTestingT(tbValue TB) (*testing.T, bool) {
	if tbValue == nil {
		return nil, false
	}
	return underlyingTestingTPrivate(tb(tbValue))
}

func underlyingTestingTPrivate(tb tb) (*testing.T, bool) {
	// Some rapid helpers clone the TB they receive by wrapping it in a new *rapid.T.
	// This happens, for example, when Custom generators spin up helper *T instances.
	// When SyncTest needs the underlying *testing.T we peel through any number of *rapid.T
	// layers until we reach the real testing object.
	switch t := any(tb).(type) {
	case *testing.T:
		return t, true
	case *T:
		return underlyingTestingTPrivate(t.tb)
	case nilTB:
		return nil, false
	default:
		return nil, false
	}
}
