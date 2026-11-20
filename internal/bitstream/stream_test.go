// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package bitstream

import (
	"fmt"
	"math/bits"
	"math/rand"
	"testing"

	"go.sdls.io/rapid/internal/stattest"
)

func TestJsfRand(t *testing.T) {
	t.Parallel()

	// using https://gist.github.com/imneme/85cff47d4bad8de6bdeb671f9c76c814
	golden := [10]uint64{
		0xe7ac7348cb3c6182,
		0xe20e62c321f18c3f,
		0x592927f9846891ae,
		0xda5c2b6e56ace47a,
		0x3c5987be726a7740,
		0x1463137b89c7292a,
		0xd118e05a46bc8156,
		0xeb72c3391969bc15,
		0xe94f306afee04198,
		0x0f57e93805e22a54,
	}

	ctx := &jsf64ctx{}
	ctx.init(0xcafe5eed00000001)

	for _, want := range golden {
		u := ctx.rand()
		if u != want {
			t.Errorf("0x%x instead of golden 0x%x", u, want)
		}
	}
}

func BenchmarkJsfRand(b *testing.B) {
	ctx := &jsf64ctx{}
	ctx.init(1)
	b.ResetTimer()

	for range b.N {
		ctx.rand()
	}
}

func BenchmarkMathRand(b *testing.B) {
	s := rand.NewSource(1).(rand.Source64)
	b.ResetTimer()

	for range b.N {
		s.Uint64()
	}
}

func TestRandomBitSteam_DrawBits(t *testing.T) {
	t.Parallel()

	s := NewRandom(stattest.Seed(t), false)

	for width := 1; width <= 64; width++ {
		for idx := range 100 {
			t.Run(fmt.Sprintf("%v bits #%v", width, idx), func(t *testing.T) {
				b := s.DrawBits(width)
				if bits.Len64(b) > width {
					t.Errorf("%v: bitlen too big for %v bits", b, width)
				}
				if bits.OnesCount64(b) > width {
					t.Errorf("%v: too much ones for %v bits", b, width)
				}
			})
		}
	}
}

func BenchmarkRandomBitStream_DrawBits1(b *testing.B) {
	s := NewRandom(1, false)
	b.ResetTimer()

	for range b.N {
		s.DrawBits(1)
	}
}

func BenchmarkRandomBitStream_DrawBits64(b *testing.B) {
	s := NewRandom(1, false)
	b.ResetTimer()

	for range b.N {
		s.DrawBits(64)
	}
}
