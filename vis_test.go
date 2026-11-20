// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"go.sdls.io/rapid/internal/bitstream"
)

func TestDataVis(t *testing.T) {
	t.Parallel()

	data := []uint64{
		0,
		0x55,
		0xaa,
		math.MaxUint8,
		0x5555,
		0xaaaa,
		math.MaxUint16,
		0x55555555,
		0xaaaaaaaa,
		math.MaxUint32,
		0x5555555555555555,
		0xaaaaaaaaaaaaaaaa,
		math.MaxUint64,
	}

	groups := []bitstream.Group{
		{Begin: 0, End: 13, Label: ""},
		{Begin: 1, End: 1 + 3, Label: "8-bit"},
		{Begin: 3, End: 4, Label: "0xff", Discard: true},
		{Begin: 4, End: 4 + 3, Label: "16-bit"},
		{Begin: 7, End: 13, Label: "big integers"},
		{Begin: 7, End: 7 + 3, Label: "32-bit"},
		{Begin: 10, End: 10 + 3, Label: "64-bit"},
	}

	rd := []bitstream.Recorded{
		{Data: data, Groups: groups},
	}

	g := SliceOf(SliceOf(Uint().Filter(func(i uint) bool { return i%2 == 1 }))).Filter(func(s [][]uint) bool { return len(s) > 0 })
	// Advance the seed, so a fixed -rapid.seed cannot redraw the same invalid value forever.
	for seed := baseSeed(); ; seed++ {
		s := bitstream.NewRandom(seed, true)
		_, err := recoverValue(g, newT(nil, s, false, nil))
		if err != nil && !err.isInvalidData() {
			t.Errorf("unexpected error %v", err)
		}

		rd = append(rd, bitstream.Recorded{Data: s.Data, Groups: s.Groups})

		if err == nil {
			break
		}
	}

	var html bytes.Buffer
	if err := visWriteHTML(&html, "vis title", rd); err != nil {
		t.Fatalf("visWriteHTML error: %v", err)
	}
	for _, want := range []string{"<h1>vis title (", "8-bit", "big integers"} {
		if !strings.Contains(html.String(), want) {
			t.Errorf("rendered HTML does not contain %q", want)
		}
	}
}
