// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"testing"

	"go.sdls.io/rapid/internal/bitstream"
	"go.sdls.io/rapid/internal/stattest"
)

func TestCollectionsWithImpossibleMinSize(t *testing.T) {
	t.Parallel()

	gens := []*Generator[any]{
		MapOfN(Bool(), Int[int](), 10, -1).AsAny(),
		SliceOfNDistinct(Int[int](), 10, -1, func(i int) int { return i % 5 }).AsAny(),
	}

	for _, gen := range gens {
		t.Run(gen.String(), func(t *testing.T) {
			t.Parallel()

			s := bitstream.NewRandom(stattest.Seed(t), false)
			_, err := recoverValue(gen, newT(nil, s, false, nil))
			if err == nil || !err.IsInvalidData() {
				t.Fatalf("got error %v instead of invalid data", err)
			}
		})
	}
}
