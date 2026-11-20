// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rapid

import (
	"encoding/binary"
	"fmt"
	"hash/maphash"
	"math"
	"math/bits"
	"os"
	"slices"
	"strings"
	"time"

	"go.sdls.io/rapid/internal/assert"
)

const (
	labelLowerFloatExp       = "lower_float_exp"
	labelLowerFloatSignif    = "lower_float_signif"
	labelLowerFloatFrac      = "lower_float_frac"
	labelMinBlockBinSearch   = "minblock_binsearch"
	labelMinBlockShift       = "minblock_shift"
	labelMinBlockSort        = "minblock_sort"
	labelMinBlockTrySmall    = "minblock_trysmall"
	labelMinBlockUnset       = "minblock_unset"
	labelRemoveGroup         = "remove_group"
	labelRemoveGroupAndLower = "remove_group_lower"
	labelRemoveGroupSpan     = "remove_groupspan"
	labelSortGroups          = "sort_groups"
)

func shrink(tb tb, deadline time.Time, rec recordedBits, err *testError, prop func(*T)) ([]uint64, *testError) {
	rec.prune()

	s := &shrinker{
		tb:        tb,
		rec:       rec,
		err:       err,
		prop:      prop,
		visBits:   []recordedBits{rec},
		tries:     map[string]int{},
		cacheSeed: maphash.MakeSeed(),
		cache:     map[uint64]struct{}{},
	}

	buf, err := s.shrink(deadline)

	if flags.debugvis {
		name := fmt.Sprintf("vis-%v.html", strings.ReplaceAll(tb.Name(), "/", "_"))
		f, err := os.Create(name) //nolint:gosec // G304: the name derives from the test's own name, under -rapid.debugvis
		if err != nil {
			tb.Logf("failed to create debugvis file %v: %v", name, err)
		} else {
			defer func() { _ = f.Close() }()

			if err = visWriteHTML(f, tb.Name(), s.visBits); err != nil {
				tb.Logf("failed to write debugvis file %v: %v", name, err)
			}
		}
	}

	return buf, err
}

type shrinker struct {
	tb        tb
	rec       recordedBits
	err       *testError
	prop      func(*T)
	visBits   []recordedBits
	tries     map[string]int
	shrinks   int
	cacheSeed maphash.Seed
	cache     map[uint64]struct{}
	hits      int
}

func (s *shrinker) debugf(verbose_ bool, format string, args ...any) {
	if flags.debug && (!verbose_ || flags.verbose) {
		s.tb.Helper()
		s.tb.Logf("[shrink] "+format, args...)
	}
}

func (s *shrinker) shrink(deadline time.Time) (buf []uint64, err *testError) {
	defer func() {
		if r := recover(); r != nil {
			buf, err = s.rec.data, r.(*testError)
		}
	}()

	i := 0
	for shrinks := -1; s.shrinks > shrinks && time.Now().Before(deadline); i++ {
		shrinks = s.shrinks

		s.debugf(false, "round %v start", i)
		s.removeGroups(deadline)
		s.minimizeBlocks(deadline)

		if s.shrinks == shrinks {
			s.debugf(false, "trying expensive algorithms for round %v", i)
			s.lowerFloatHack(deadline)
			s.removeGroupsAndLower(deadline)
			s.sortGroups(deadline)
			s.removeGroupSpans(deadline)
		}
	}

	tries := 0
	for _, count := range s.tries {
		tries += count
	}
	s.debugf(false, "done, %v rounds total (%v tries, %v shrinks, %v cache hits):\n%v", i, tries, s.shrinks, s.hits, s.tries)

	return s.rec.data, s.err
}

func (s *shrinker) removeGroups(deadline time.Time) {
	for idx := 0; idx < len(s.rec.groups) && time.Now().Before(deadline); idx++ {
		g := s.rec.groups[idx]
		if !g.standalone || g.end < 0 {
			continue
		}

		if s.accept(without(s.rec.data, g), labelRemoveGroup, "remove group %q at %v: [%v, %v)", g.label, idx, g.begin, g.end) {
			idx--
		}
	}
}

func (s *shrinker) minimizeBlocks(deadline time.Time) {
	for idx := 0; idx < len(s.rec.data) && time.Now().Before(deadline); idx++ {
		minimize(s.rec.data[idx], func(u uint64, label string) bool {
			buf := append([]uint64(nil), s.rec.data...)
			buf[idx] = u
			return s.accept(buf, label, "minimize block %v: %v to %v", idx, s.rec.data[idx], u)
		})
	}
}

func (s *shrinker) lowerFloatHack(deadline time.Time) {
	for idx := 0; idx < len(s.rec.groups) && time.Now().Before(deadline); idx++ {
		g := s.rec.groups[idx]
		if !g.standalone || g.end != g.begin+7 {
			continue
		}

		buf := append([]uint64(nil), s.rec.data...)
		buf[g.begin+3] -= 1
		buf[g.begin+4] = math.MaxUint64
		buf[g.begin+5] = math.MaxUint64
		buf[g.begin+6] = math.MaxUint64

		if !s.accept(buf, labelLowerFloatExp, "lower float exponent of group %q at %v to %v", g.label, idx, buf[g.begin+3]) {
			buf := append([]uint64(nil), s.rec.data...)
			buf[g.begin+4] -= 1
			buf[g.begin+5] = math.MaxUint64
			buf[g.begin+6] = math.MaxUint64

			if !s.accept(buf, labelLowerFloatSignif, "lower float significant of group %q at %v to %v", g.label, idx, buf[g.begin+4]) {
				buf := append([]uint64(nil), s.rec.data...)
				buf[g.begin+5] -= 1
				buf[g.begin+6] = math.MaxUint64

				s.accept(buf, labelLowerFloatFrac, "lower float frac of group %q at %v to %v", g.label, idx, buf[g.begin+5])
			}
		}
	}
}

func (s *shrinker) removeGroupsAndLower(deadline time.Time) {
	for idx := 0; idx < len(s.rec.data) && time.Now().Before(deadline); idx++ {
		if s.rec.data[idx] == 0 {
			continue
		}

		buf := append([]uint64(nil), s.rec.data...)
		buf[idx] -= 1

		for jdx := range len(s.rec.groups) {
			g := s.rec.groups[jdx]
			if !g.standalone || g.end < 0 || (idx >= g.begin && idx < g.end) {
				continue
			}

			if s.accept(without(buf, g), labelRemoveGroupAndLower, "lower block %v to %v and remove group %q at %v: [%v, %v)",
				idx, buf[idx], g.label, jdx, g.begin, g.end) {
				idx--
				break
			}
		}
	}
}

func (s *shrinker) sortGroups(deadline time.Time) {
	for idx := 1; idx < len(s.rec.groups) && time.Now().Before(deadline); idx++ {
		for jdx := idx; jdx > 0; {
			g := s.rec.groups[jdx]
			if !g.standalone || g.end < 0 {
				break
			}

			jdx = s.swapWithEarlier(jdx, g)
		}
	}
}

// swapWithEarlier tries to swap group g, at index jdx, with an earlier standalone group of the same label
// that ends before g begins, nearest first. It returns the index g moved to, or -1 when no swap was accepted.
func (s *shrinker) swapWithEarlier(jdx int, g groupInfo) int {
	for kdx := jdx - 1; kdx >= 0; kdx-- {
		h := s.rec.groups[kdx]
		if !h.standalone || h.end < 0 || h.end > g.begin || h.label != g.label {
			continue
		}

		if s.accept(swapSpans(s.rec.data, h, g), labelSortGroups, "swap groups %q at %v: [%v, %v) and %q at %v: [%v, %v)",
			g.label, jdx, g.begin, g.end, h.label, kdx, h.begin, h.end) {
			return kdx
		}
	}

	return -1
}

// swapSpans returns a copy of data with the spans of groups a and b exchanged; a ends before b begins.
func swapSpans(data []uint64, a, b groupInfo) []uint64 {
	buf := append([]uint64(nil), data[:a.begin]...)
	buf = append(buf, data[b.begin:b.end]...)
	buf = append(buf, data[a.end:b.begin]...)
	buf = append(buf, data[a.begin:a.end]...)

	return append(buf, data[b.end:]...)
}

func (s *shrinker) removeGroupSpans(deadline time.Time) {
	for idx := 0; idx < len(s.rec.groups) && time.Now().Before(deadline); idx++ {
		g := s.rec.groups[idx]
		if !g.standalone || g.end < 0 {
			continue
		}

		groups := []groupInfo{g}
		for jdx := idx + 1; jdx < len(s.rec.groups); jdx++ {
			h := s.rec.groups[jdx]
			if !h.standalone || h.end < 0 || h.begin < groups[len(groups)-1].end {
				continue
			}

			groups = append(groups, h)
			buf := without(s.rec.data, groups...)

			if s.accept(buf, labelRemoveGroupSpan, "remove %v groups %v", len(groups), groups) {
				idx--
				break
			}
		}
	}
}

func (s *shrinker) accept(buf []uint64, label, format string, args ...any) bool {
	if compareData(buf, s.rec.data) >= 0 {
		return false
	}
	bufHash := maphash.String(s.cacheSeed, dataStr(buf))
	if _, ok := s.cache[bufHash]; ok {
		s.hits++
		return false
	}

	s.debugf(true, label+": trying to reproduce the failure with a smaller test case: "+format, args...)
	s.tries[label]++
	s1 := newBufBitStream(buf, false)
	err1 := checkOnce(newT(s.tb, s1, flags.debug && flags.verbose, nil), s.prop)
	if traceback(err1) != traceback(s.err) {
		s.cache[bufHash] = struct{}{}
		return false
	}

	s.debugf(true, label+": trying to reproduce the failure")
	s.tries[label]++
	s.err = err1
	s2 := newBufBitStream(buf, true)
	err2 := checkOnce(newT(s.tb, s2, flags.debug && flags.verbose, nil), s.prop)
	s.rec = s2.recordedBits
	s.rec.prune()
	assert.True(compareData(s.rec.data, buf) <= 0)
	if flags.debugvis {
		s.visBits = append(s.visBits, s.rec)
	}
	if !sameError(err1, err2) {
		panic(err2)
	}

	s.debugf(false, label+" success: "+format, args...)
	s.shrinks++

	return true
}

func minimize(u uint64, cond func(uint64, string) bool) uint64 {
	if u == 0 {
		return 0
	}
	for candidate := uint64(0); candidate < u && candidate < small; candidate++ {
		if cond(candidate, labelMinBlockTrySmall) {
			return candidate
		}
	}
	if u <= small {
		return u
	}

	m := &minimizer{best: u, cond: cond}

	m.rShift()
	m.unsetBits()
	m.sortBits()
	m.binSearch()

	return m.best
}

type minimizer struct {
	best uint64
	cond func(uint64, string) bool
}

func (m *minimizer) accept(u uint64, label string) bool {
	if u >= m.best || u < small || !m.cond(u, label) {
		return false
	}
	m.best = u
	return true
}

func (m *minimizer) rShift() {
	for m.accept(m.best>>1, labelMinBlockShift) {
	}
}

func (m *minimizer) unsetBits() {
	size := bits.Len64(m.best)

	for bit := size - 1; bit >= 0; bit-- {
		m.accept(m.best^1<<uint(bit), labelMinBlockUnset)
	}
}

func (m *minimizer) sortBits() {
	size := bits.Len64(m.best)

	for bit := size - 1; bit >= 0; bit-- {
		h := uint64(1 << uint(bit))
		if m.best&h != 0 {
			for lowerBit := range bit {
				l := uint64(1 << uint(lowerBit))
				if m.best&l == 0 {
					if m.accept(m.best^(l|h), labelMinBlockSort) {
						break
					}
				}
			}
		}
	}
}

func (m *minimizer) binSearch() {
	if !m.accept(m.best-1, labelMinBlockBinSearch) {
		return
	}

	i := uint64(0)
	j := m.best
	for i < j {
		h := i + (j-i)/2
		if m.accept(h, labelMinBlockBinSearch) {
			j = h
		} else {
			i = h + 1
		}
	}
}

func without(data []uint64, groups ...groupInfo) []uint64 {
	buf := append([]uint64(nil), data...)

	for idx := range slices.Backward(groups) {
		group := &groups[idx]
		buf = append(buf[:group.begin], buf[group.end:]...)
	}

	return buf
}

func dataStr(data []uint64) string {
	b := &strings.Builder{}
	err := binary.Write(b, binary.LittleEndian, data)
	assert.True(err == nil)
	return b.String()
}

func compareData(a, b []uint64) int {
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}

	for idx := range a {
		if a[idx] < b[idx] {
			return -1
		}
		if a[idx] > b[idx] {
			return 1
		}
	}

	return 0
}
