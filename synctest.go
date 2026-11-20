// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"testing"
	"testing/synctest"
)

// SyncTest runs prop within a testing/synctest bubble.
// Callers must already be executing inside a rapid.Check-style helper;
// SyncTest forwards failures to the parent *rapid.T and restores its state afterwards.
func SyncTest(t *T, prop func(*T)) {
	if t == nil {
		panic("rapid.SyncTest requires *rapid.T")
	}

	t.Helper()

	testT, ok := underlyingTestingT(t.tb)
	if !ok {
		t.Fatalf("[rapid] SyncTest requires a *testing.T backing the current rapid test")
		return
	}

	syncTestWithinRapid(t, testT, prop)
}

func syncTestWithinRapid(t *T, parent *testing.T, prop func(*T)) {
	// synctest.Test converts failures inside the bubble into parent.FailNow (runtime.Goexit),
	// which would bypass rapid's panic-based failure capture/shrinking. Run the bubble in a
	// separate goroutine, swallow failures inside the bubble, and re-panic outside as a
	// *testError so checkOnce can shrink and generate failfiles as usual.
	resultCh := make(chan *testError, 1)

	go func() {
		var captured *testError
		returned := false
		defer func() {
			if r := recover(); r != nil {
				captured = panicToError(r)
			} else if !returned && captured == nil {
				captured = panicToError(stopTest("[rapid] SyncTest aborted via testing.FailNow"))
			}
			resultCh <- captured
		}()

		synctest.Test(parent, func(st *testing.T) {
			st.Helper()

			restore := t.enterBubble(st)
			defer func() {
				panicValue := recover()
				// Always run rapid cleanups, even if the property panicked.
				if r := t.cleanupRecovered(); r != nil {
					panicValue = r
				}
				restore()

				if panicValue != nil {
					captured = panicToError(panicValue)
				}
			}()

			prop(t)
			t.failOnError()
		})

		returned = true
	}()

	if err := <-resultCh; err != nil {
		panic(err)
	}
}

// enterBubble points t at st, the *testing.T of a synctest bubble, with fresh per-run state,
// and returns the function that restores the state it replaced.
// No lock is needed because no other goroutine touches t before control passes to the property.
func (t *T) enterBubble(st *testing.T) (restore func()) {
	prevTB := t.tb
	prevTBLog := t.tbLog // preserved so we keep the original logging behaviour
	prevCtx := t.ctx
	prevCancel := t.cancelCtx
	prevCleanups := t.cleanups
	prevCleaning := t.cleaning.Load()

	t.tb = st
	t.ctx = nil
	t.cancelCtx = nil
	t.cleanups = nil
	t.cleaning.Store(false)

	return func() {
		t.tb = prevTB
		t.tbLog = prevTBLog
		t.ctx = prevCtx
		t.cancelCtx = prevCancel
		t.cleanups = prevCleanups
		t.cleaning.Store(prevCleaning)
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
