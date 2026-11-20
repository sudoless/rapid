// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package bitstream is the source of every value rapid generates: a stream of random or replayed
// 64-bit words, recorded in labelled groups so the shrinker can edit them, and the primitives
// that turn words into integers, coin flips, die rolls and repetition counts.
package bitstream

import (
	"math"
	"math/bits"

	"go.sdls.io/rapid/internal/assert"
)

// InvalidData is the panic value of a draw that cannot produce a value from the stream's data:
// a replayed stream ran out, or a generator rejected too many values in a row.
// The engine discards the test case instead of failing it.
type InvalidData string

// Stream is where generators draw their bits from.
type Stream interface {
	DrawBits(n int) uint64
	BeginGroup(label string, standalone bool) int
	EndGroup(i int, discard bool)
}

type Random struct {
	ctx jsf64ctx
	Recorded
}

func NewRandom(seed uint64, persist bool) *Random {
	s := &Random{}
	s.Init(seed)
	s.persist = persist
	return s
}

func (s *Random) Init(seed uint64) {
	s.ctx.init(seed)
}

func (s *Random) DrawBits(n int) uint64 {
	assert.True(n >= 0)

	var u uint64
	if n <= 64 {
		u = s.ctx.rand() & Bitmask64(uint(n))
	} else {
		u = math.MaxUint64
	}
	s.record(u)

	return u
}

type Buffer struct {
	buf []uint64
	Recorded
}

func NewBuffer(buf []uint64, persist bool) *Buffer {
	s := &Buffer{
		buf: buf,
	}
	s.persist = persist
	return s
}

func (s *Buffer) DrawBits(n int) uint64 {
	assert.True(n >= 0)

	if len(s.buf) == 0 {
		panic(InvalidData("overrun"))
	}

	u := s.buf[0] & Bitmask64(uint(n))
	s.record(u)
	s.buf = s.buf[1:]

	return u
}

type Group struct {
	Begin      int
	End        int
	Label      string
	Standalone bool
	Discard    bool
}

type Recorded struct {
	Data    []uint64
	Groups  []Group
	dataLen int
	persist bool
}

func (rec *Recorded) record(u uint64) {
	if rec.persist {
		rec.Data = append(rec.Data, u)
	} else {
		rec.dataLen++
	}
}

func (rec *Recorded) BeginGroup(label string, standalone bool) int {
	if !rec.persist {
		return rec.dataLen
	}

	rec.Groups = append(rec.Groups, Group{
		Begin:      len(rec.Data),
		End:        -1,
		Label:      label,
		Standalone: standalone,
	})

	return len(rec.Groups) - 1
}

func (rec *Recorded) EndGroup(i int, discard bool) {
	assert.Truef(discard || (!rec.persist && rec.dataLen > i) || (rec.persist && len(rec.Data) > rec.Groups[i].Begin),
		"group did not use any data from bitstream; this is likely a result of Custom generator not calling any of the built-in generators")

	if !rec.persist {
		return
	}

	rec.Groups[i].End = len(rec.Data)
	rec.Groups[i].Discard = discard
}

func (rec *Recorded) Prune() {
	assert.True(rec.persist)

	for idx := 0; idx < len(rec.Groups); {
		if rec.Groups[idx].Discard {
			rec.removeGroup(idx) // O(n^2)
		} else {
			idx++
		}
	}

	for idx := range rec.Groups {
		group := &rec.Groups[idx]
		assert.True(group.Begin != group.End)
	}
}

func (rec *Recorded) removeGroup(i int) {
	g := rec.Groups[i]
	assert.True(g.End >= 0)

	j := i + 1
	for j < len(rec.Groups) && rec.Groups[j].End <= g.End {
		j++
	}

	rec.Data = append(rec.Data[:g.Begin], rec.Data[g.End:]...)
	rec.Groups = append(rec.Groups[:i], rec.Groups[j:]...)

	n := g.End - g.Begin
	for idx := range rec.Groups {
		if rec.Groups[idx].Begin >= g.End {
			rec.Groups[idx].Begin -= n
		}
		if rec.Groups[idx].End >= g.End {
			rec.Groups[idx].End -= n
		}
	}
}

// "A Small Noncryptographic PRNG" by Bob Jenkins
// See http://www.pcg-random.org/posts/bob-jenkins-small-prng-passes-practrand.html for some recent analysis.
type jsf64ctx struct {
	a uint64
	b uint64
	c uint64
	d uint64
}

func (x *jsf64ctx) init(seed uint64) {
	x.a = 0xf1ea5eed
	x.b = seed
	x.c = seed
	x.d = seed

	for range 20 {
		x.rand()
	}
}

func (x *jsf64ctx) rand() uint64 {
	e := x.a - bits.RotateLeft64(x.b, 7)
	x.a = x.b ^ bits.RotateLeft64(x.c, 13)
	x.b = x.c + bits.RotateLeft64(x.d, 37)
	x.c = x.d + e
	x.d = e + x.a
	return x.d
}
