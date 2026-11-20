// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build !go1.25

package rapid

// SyncTest is only available on Go 1.25+.
func SyncTest(t *T, _ func(*T)) {
	if t == nil {
		panic("rapid.SyncTest requires *rapid.T")
	}
	t.Helper()
	t.Fatalf("[rapid] SyncTest requires Go 1.25 or newer")
}
