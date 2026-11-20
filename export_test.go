// Copyright (c) 2026 SUDOLESS SRL
// SPDX-License-Identifier: BSD-3-Clause-Clear

package rapid

import (
	"testing"

	"go.sdls.io/rapid/internal/engine"
)

// CheckShrink exposes checkShrink to package rapid_test.
var CheckShrink = checkShrink

// CheckOutcome runs prop for 100 checks, without reading or writing fail files,
// and reports how many test cases were valid and invalid, and whether prop failed.
func CheckOutcome(t *testing.T, prop func(*T)) (valid int, invalid int, failed bool) {
	t.Helper()

	res := search(t, engine.BaseSeed(), prop)

	return res.Valid, res.Invalid, res.Err1 != nil
}
