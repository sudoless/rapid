// Copyright 2019 Gregory Petrosyan <gregory.petrosyan@gmail.com>
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package engine

import (
	"encoding/binary"
	"fmt"
	"hash/maphash"
	"log"
	"math/bits"
	"os"
	"slices"
	"strings"
	"time"

	"go.sdls.io/rapid/internal/assert"
	"go.sdls.io/rapid/internal/bitstream"
)

// smallBlock bounds the block values minimize tries one by one before it shifts and clears bits.
const smallBlock = 5

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

// Logger is where shrinkTestCase reports its progress: the test running the property.
type Logger interface {
	Helper()
	Logf(format string, args ...any)
	Name() string
}

// Run runs the property on the test case s draws and returns how it failed, or nil when it passed.
// The property's own output goes to out when it is set, else to the test log when log is set.
type Run func(s bitstream.Stream, log bool, out *log.Logger) *TestError

// shrinkTestCase minimises rec, a test case that failed with err, until deadline: it keeps every smaller
// test case that run fails on in the same way. It returns the smallest such test case and its failure,
// or the test case and failure of a replay that failed differently.
// Under -rapid.debugvis it also writes every step to vis-<test name>.html, the name made safe as for fail files.
func shrinkTestCase(tb Logger, deadline time.Time, rec bitstream.Recorded, err *TestError, run Run) ([]uint64, *TestError) {
	best := slices.Clone(rec.Data)
	rec.Prune()

	s := &shrinker{
		tb:        tb,
		best:      best,
		rec:       rec,
		err:       err,
		run:       run,
		visBits:   []bitstream.Recorded{rec},
		tries:     map[string]int{},
		cacheSeed: maphash.MakeSeed(),
		cache:     map[uint64]struct{}{},
	}

	buf, err := s.shrink(deadline)

	if Flags.DebugVis {
		if err := writeVisFile(tb.Name(), s.visBits); err != nil {
			tb.Logf("[rapid] %v", err)
		}
	}

	return buf, err
}

// writeVisFile writes the -rapid.debugvis page of the shrink steps of the test testName.
func writeVisFile(testName string, steps []bitstream.Recorded) error {
	name := "vis-" + testFileName(testName) + ".html"
	f, err := os.Create(name) //nolint:gosec // G304: the name derives from the test's own name, under -rapid.debugvis
	if err != nil {
		return fmt.Errorf("creating debugvis file: %w", err)
	}

	if err := writeVisHTML(f, testName, steps); err != nil {
		_ = f.Close() // the write error is the one to report
		return fmt.Errorf("writing debugvis file %q: %w", name, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("closing debugvis file %q: %w", name, err)
	}

	return nil
}

type shrinker struct {
	tb        Logger
	best      []uint64           // the last test case run, which failed with err
	rec       bitstream.Recorded // best's recording, pruned of discarded groups: what the shrinker edits
	labels    []string           // rec's group labels, resolved on first use
	err       *TestError
	run       Run
	visBits   []bitstream.Recorded
	tries     map[string]int
	shrinks   int
	cacheSeed maphash.Seed
	cache     map[uint64]struct{}
	hits      int
}

func (s *shrinker) debugf(verbose_ bool, format string, args ...any) {
	if Flags.Debug && (!verbose_ || Flags.Verbose) {
		s.tb.Helper()
		s.tb.Logf("[shrink] "+format, args...)
	}
}

func (s *shrinker) shrink(deadline time.Time) (buf []uint64, err *TestError) {
	defer func() {
		if r := recover(); r != nil {
			buf, err = s.best, r.(*TestError)
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
	s.tryPruned(deadline)

	tries := 0
	for _, count := range s.tries {
		tries += count
	}
	s.debugf(false, "done, %v rounds total (%v tries, %v shrinks, %v cache hits):\n%v", i, tries, s.shrinks, s.hits, s.tries)

	return s.best, s.err
}

// tryPruned keeps the recording the shrinker edits in place of the last test case run, when it fails the same
// way: pruned of the groups the test case discarded, it is smaller, and accept only runs smaller ones.
func (s *shrinker) tryPruned(deadline time.Time) {
	if compareData(s.rec.Data, s.best) >= 0 || !time.Now().Before(deadline) {
		return
	}

	s.tries["pruned"]++
	err := s.run(bitstream.NewBuffer(s.rec.Data, false), Flags.Debug && Flags.Verbose, nil)
	if Traceback(err) == Traceback(s.err) {
		s.best, s.err = s.rec.Data, err
	}
}

func (s *shrinker) removeGroups(deadline time.Time) {
	for idx := 0; idx < len(s.rec.Groups) && time.Now().Before(deadline); idx++ {
		g := s.rec.Groups[idx]
		if !g.Standalone || g.End < 0 {
			continue
		}

		if s.accept(without(s.rec.Data, g), labelRemoveGroup, "remove group %q at %v: [%v, %v)", g.Label, idx, g.Begin, g.End) {
			idx--
		}
	}
}

func (s *shrinker) minimizeBlocks(deadline time.Time) {
	for idx := 0; idx < len(s.rec.Data) && time.Now().Before(deadline); idx++ {
		minimize(s.rec.Data[idx], func(u uint64, label string) bool {
			buf := slices.Clone(s.rec.Data)
			buf[idx] = u
			return s.accept(buf, label, "minimize block %v: %v to %v", idx, s.rec.Data[idx], u)
		})
	}
}

func (s *shrinker) lowerFloatHack(deadline time.Time) {
	steps := [bitstream.FloatLowerSteps]string{labelLowerFloatExp, labelLowerFloatSignif, labelLowerFloatFrac}
	for idx := 0; idx < len(s.rec.Groups) && time.Now().Before(deadline); idx++ {
		if !bitstream.IsFloat(s.rec.Groups, idx) {
			continue
		}

		g := s.rec.Groups[idx]
		for step, label := range steps {
			buf := bitstream.LowerFloat(s.rec.Data, g, step)
			if s.accept(buf, label, "%v of group %q at %v", label, g.Label, idx) {
				break
			}
		}
	}
}

func (s *shrinker) removeGroupsAndLower(deadline time.Time) {
	for idx := 0; idx < len(s.rec.Data) && time.Now().Before(deadline); idx++ {
		if s.rec.Data[idx] == 0 {
			continue
		}

		buf := slices.Clone(s.rec.Data)
		buf[idx] -= 1

		for jdx := range len(s.rec.Groups) {
			g := s.rec.Groups[jdx]
			if !g.Standalone || g.End < 0 || (idx >= g.Begin && idx < g.End) {
				continue
			}

			if s.accept(without(buf, g), labelRemoveGroupAndLower, "lower block %v to %v and remove group %q at %v: [%v, %v)",
				idx, buf[idx], g.Label, jdx, g.Begin, g.End) {
				idx--
				break
			}
		}
	}
}

// label returns the label of rec's group i. Labels are resolved together, the first time one is asked for,
// so a recording that the sort pass never reaches formats none.
func (s *shrinker) label(i int) string {
	if s.labels == nil {
		s.labels = make([]string, len(s.rec.Groups))
		for idx := range s.rec.Groups {
			s.labels[idx] = s.rec.Groups[idx].Label.String()
		}
	}

	return s.labels[i]
}

func (s *shrinker) sortGroups(deadline time.Time) {
	for idx := 1; idx < len(s.rec.Groups) && time.Now().Before(deadline); idx++ {
		for jdx := idx; jdx > 0; {
			g := s.rec.Groups[jdx]
			if !g.Standalone || g.End < 0 {
				break
			}

			jdx = s.swapWithEarlier(jdx, g)
		}
	}
}

// swapWithEarlier tries to swap group g, at index jdx, with an earlier standalone group of the same label
// that ends before g begins, nearest first. It returns the index g moved to, or -1 when no swap was accepted.
func (s *shrinker) swapWithEarlier(jdx int, g bitstream.Group) int {
	for kdx := jdx - 1; kdx >= 0; kdx-- {
		h := s.rec.Groups[kdx]
		if !h.Standalone || h.End < 0 || h.End > g.Begin || s.label(kdx) != s.label(jdx) {
			continue
		}

		if s.accept(swapSpans(s.rec.Data, h, g), labelSortGroups, "swap groups %q at %v: [%v, %v) and %q at %v: [%v, %v)",
			g.Label, jdx, g.Begin, g.End, h.Label, kdx, h.Begin, h.End) {
			return kdx
		}
	}

	return -1
}

// swapSpans returns a copy of data with the spans of groups a and b exchanged; a ends before b begins.
func swapSpans(data []uint64, a, b bitstream.Group) []uint64 {
	buf := make([]uint64, 0, len(data))
	buf = append(buf, data[:a.Begin]...)
	buf = append(buf, data[b.Begin:b.End]...)
	buf = append(buf, data[a.End:b.Begin]...)
	buf = append(buf, data[a.Begin:a.End]...)

	return append(buf, data[b.End:]...)
}

func (s *shrinker) removeGroupSpans(deadline time.Time) {
	for idx := 0; idx < len(s.rec.Groups) && time.Now().Before(deadline); idx++ {
		g := s.rec.Groups[idx]
		if !g.Standalone || g.End < 0 {
			continue
		}

		groups := []bitstream.Group{g}
		for jdx := idx + 1; jdx < len(s.rec.Groups); jdx++ {
			h := s.rec.Groups[jdx]
			if !h.Standalone || h.End < 0 || h.Begin < groups[len(groups)-1].End {
				continue
			}

			groups = append(groups, h)
			buf := without(s.rec.Data, groups...)

			if s.accept(buf, labelRemoveGroupSpan, "remove %v groups %v", len(groups), groups) {
				idx--
				break
			}
		}
	}
}

func (s *shrinker) accept(buf []uint64, label, format string, args ...any) bool {
	if compareData(buf, s.rec.Data) >= 0 {
		return false
	}
	bufHash := maphash.String(s.cacheSeed, dataStr(buf))
	if _, ok := s.cache[bufHash]; ok {
		s.hits++
		return false
	}

	s.debugf(true, label+": trying to reproduce the failure with a smaller test case: "+format, args...)
	s.tries[label]++
	s1 := bitstream.NewBuffer(buf, false)
	err1 := s.run(s1, Flags.Debug && Flags.Verbose, nil)
	if Traceback(err1) != Traceback(s.err) {
		s.cache[bufHash] = struct{}{}
		return false
	}

	s.debugf(true, label+": trying to reproduce the failure")
	s.tries[label]++
	s.err = err1
	s2 := bitstream.NewBuffer(buf, true)
	err2 := s.run(s2, Flags.Debug && Flags.Verbose, nil)
	s.best = buf // a pruned recording can leave out what made buf fail, so only buf is known to fail
	s.rec = s2.Recorded
	s.rec.Prune()
	s.labels = nil
	assert.True(compareData(s.rec.Data, buf) <= 0)
	if Flags.DebugVis {
		s.visBits = append(s.visBits, s.rec)
	}
	if !SameError(err1, err2) {
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
	for candidate := uint64(0); candidate < u && candidate < smallBlock; candidate++ {
		if cond(candidate, labelMinBlockTrySmall) {
			return candidate
		}
	}
	if u <= smallBlock {
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
	if u >= m.best || u < smallBlock || !m.cond(u, label) {
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

func without(data []uint64, groups ...bitstream.Group) []uint64 {
	buf := slices.Clone(data)

	for idx := range slices.Backward(groups) {
		group := &groups[idx]
		buf = append(buf[:group.Begin], buf[group.End:]...)
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
