// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package engine

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"go.sdls.io/rapid/internal/bitstream"
	"go.sdls.io/rapid/internal/stattest"
)

func TestWriteVisHTML(t *testing.T) {
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
		{Begin: 0, End: 13, Label: bitstream.Label("")},
		{Begin: 1, End: 1 + 3, Label: bitstream.Label("8-bit")},
		{Begin: 3, End: 4, Label: bitstream.Label("0xff"), Discard: true},
		{Begin: 4, End: 4 + 3, Label: bitstream.Label("16-bit")},
		{Begin: 7, End: 13, Label: bitstream.Label("big integers")},
		{Begin: 7, End: 7 + 3, Label: bitstream.Label("32-bit")},
		{Begin: 10, End: 10 + 3, Label: bitstream.Label("64-bit")},
	}

	// A drawn recording: a repetition whose odd elements are rejected, and a group left open.
	s := bitstream.NewRandom(stattest.Seed(t), true)
	elems := bitstream.NewRepeat(0, 8, -1, "elems")
	for elems.More(s) {
		if u, _, _ := bitstream.UintN(s, 100, true); u%2 == 1 {
			elems.Reject()
		}
	}
	s.BeginGroup(bitstream.Label("unfinished"), false)
	s.DrawBits(8)

	rd := []bitstream.Recorded{
		{Data: data, Groups: groups},
		{Data: s.Data, Groups: s.Groups},
	}

	var html bytes.Buffer
	if err := writeVisHTML(&html, "vis title", rd); err != nil {
		t.Fatalf("writeVisHTML error: %v", err)
	}
	for _, want := range []string{"<h1>vis title (", "8-bit", "big integers", "discard", "endless", "elems@repeat"} {
		if !strings.Contains(html.String(), want) {
			t.Errorf("rendered HTML does not contain %q", want)
		}
	}
}

// A generator that fails before its first draw leaves its group open past the last word.
func TestWriteVisHTMLGroupOpenedAfterLastDraw(t *testing.T) {
	t.Parallel()

	s := bitstream.NewRandom(stattest.FixedSeed, true)
	_ = s.DrawBits(8)
	s.BeginGroup(bitstream.Label("Custom(int)"), false)

	var html bytes.Buffer
	if err := writeVisHTML(&html, "vis title", []bitstream.Recorded{s.Recorded}); err != nil {
		t.Fatalf("writeVisHTML error: %v", err)
	}
}
