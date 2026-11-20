// Copyright (c) 2026 SUDOLESS SRL
// SPDX-License-Identifier: BSD-3-Clause-Clear

// Package engine checks a property: it searches for a test case the property fails on,
// replays and shrinks it, and reports it. It also holds the -rapid.* settings a run takes,
// the failure itself (TestError), the fail files that save it for replay,
// and the -rapid.debugvis page that shows how it was shrunk.
package engine
