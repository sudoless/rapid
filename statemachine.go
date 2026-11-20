// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"reflect"
	"slices"
	"testing"

	"go.sdls.io/rapid/internal/assert"
	"go.sdls.io/rapid/internal/bitstream"
	"go.sdls.io/rapid/internal/engine"
)

const (
	actionLabel        bitstream.Label = "action"
	repeatActionsLabel bitstream.Label = "Repeat"
)

const (
	validActionTries  = 100 // hack, but probably good enough for now
	checkMethodName   = "Check"
	noValidActionsMsg = "can't find a valid (non-skipped) action"
)

// Repeat executes a random sequence of actions (often called a "state machine" test).
// actions[""], if set, is executed before/after every other action invocation
// and should only contain invariant checking code.
//
// For complex state machines, it can be more convenient to specify actions as
// methods of a special state machine type. In this case, [StateMachineActions]
// can be used to create an actions map from state machine methods using reflection.
//
// Repeat runs an average of -rapid.steps actions, half as many under -short; [*T.RepeatSteps] sets the number.
func (t *T) Repeat(actions map[string]func(*T)) {
	t.Helper()

	steps := engine.Flags.Steps
	if testing.Short() {
		steps /= 2
	}
	t.repeat(actions, steps)
}

// RepeatSteps is [*T.Repeat] with an average of steps actions, whatever -rapid.steps or -short say. A failure
// replays from a fail file or a seed only with the step count it was found with, so a fixed count keeps it
// replayable across runs with other flags. RepeatSteps panics if steps < 1.
func (t *T) RepeatSteps(actions map[string]func(*T), steps int) {
	t.Helper()
	assert.Truef(steps >= 1, "RepeatSteps(%v): want at least 1 step", steps)
	t.repeat(actions, steps)
}

func (t *T) repeat(actions map[string]func(*T), steps int) {
	t.Helper()

	check := func(*T) {}
	actionKeys := make([]string, 0, len(actions))
	for key, action := range actions {
		if key != "" {
			actionKeys = append(actionKeys, key)
		} else {
			check = action
		}
	}
	if len(actionKeys) == 0 {
		return
	}
	slices.Sort(actionKeys)

	repeat := bitstream.NewRepeat(-1, -1, float64(steps), repeatActionsLabel)
	sm := stateMachine{
		check:      check,
		actionKeys: SampledFrom(actionKeys),
		actions:    actions,
	}

	sm.check(t)
	t.failOnError()
	for repeat.More(t.s) {
		ok := sm.executeAction(t)
		if ok {
			sm.check(t)
			t.failOnError()
		} else {
			repeat.Reject()
		}
	}
}

type StateMachine interface {
	// Check is ran after every action and should contain invariant checks.
	//
	// All other public methods should have a form ActionName(t *rapid.T)
	// or ActionName(t rapid.TB) and are used as possible actions.
	// At least one action has to be specified.
	Check(t *T)
}

// StateMachineActions creates an actions map for [*T.Repeat]
// from methods of a [StateMachine] type instance using reflection.
func StateMachineActions(sm StateMachine) map[string]func(*T) {
	var (
		v = reflect.ValueOf(sm)
		t = v.Type()
		n = t.NumMethod()
	)

	actions := make(map[string]func(*T), n)
	for idx := range n {
		name := t.Method(idx).Name

		if name == checkMethodName {
			continue
		}

		m, ok := reflect.TypeAssert[func(*T)](v.Method(idx))
		if ok {
			actions[name] = m
		}

		m2, ok := reflect.TypeAssert[func(TB)](v.Method(idx))
		if ok {
			actions[name] = func(t *T) {
				m2(t)
			}
		}
	}

	assert.Truef(len(actions) > 0, "state machine of type %v has no actions specified", t)
	actions[""] = sm.Check

	return actions
}

type stateMachine struct {
	check      func(*T)
	actionKeys *Generator[string]
	actions    map[string]func(*T)
}

func (sm *stateMachine) executeAction(t *T) bool {
	t.Helper()

	for range validActionTries {
		i := t.s.BeginGroup(actionLabel, false)
		action := sm.actions[sm.actionKeys.Draw(t, "action")]
		invalid, skipped := runAction(t, action)
		t.s.EndGroup(i, false)

		if skipped {
			continue
		} else {
			return !invalid
		}
	}

	panic(engine.StopTest(noValidActionsMsg))
}

func runAction(t *T, action func(*T)) (invalid, skipped bool) {
	defer func(draws int) {
		if r := recover(); r != nil {
			if _, ok := r.(bitstream.InvalidData); ok {
				invalid = true
				skipped = t.draws == draws
			} else {
				panic(r)
			}
		}
	}(t.draws)

	action(t)
	t.failOnError()

	return false, false
}
