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

	res := doCheck(t, checkDeadline(nil), 100, engine.BaseSeed(), "", false, prop)

	return res.valid, res.invalid, res.err1 != nil
}
