package hll

import (
	"encoding/binary"
	"sort"
)

// sparse keeps a set of untique hashes.
// if sparse if dirty `s[0]&(1<<7) != 0`, hashes are unordered, maybe even with duplicates.
// if sparse is not dirty, hashes are sorted and duplicates are removed.
type sparse []byte

func (s sparse) dirty() bool {
	return s[0]&(1<<7) != 0
}

func (s sparse) EstimateCardinality() int {
	if s.dirty() {
		s.sort() // clears dirty.
	}
	return int(s.size())
}

func (s sparse) Add(hash uint64) addResult {
	sz := s.size()
	sz++
	if sz < uint32(len(s))>>3 {
		binary.LittleEndian.PutUint64(s[sz<<3:], hash)
		s.setSize(sz | 1<<31)
		return ok
	}
	if !s.dirty() {
		return full
	}
	s.sort()
	sz = s.size()
	// Add a 100 byte padding so we do not re-sort on every add once the sparse buffer is almost full.
	if sz+100 < uint32(len(s))>>3 {
		sz++
		binary.LittleEndian.PutUint64(s[sz<<3:], hash)
		s.setSize(sz | 1<<31)
		return ok
	}
	return full
}

type sortable sparse

func (s sortable) Len() int {
	return len(s) >> 3
}

func (s sortable) Swap(i, j int) {
	i <<= 3
	j <<= 3
	for k := 0; k < 8; k++ {
		s[i], s[j] = s[j], s[i]
		i++
		j++
	}
}

func (s sortable) Less(i, j int) bool {
	i <<= 3
	j <<= 3
	for k := 0; k < 8; k++ {
		a := s[i]
		b := s[j]
		if a < b {
			return true
		}
		if a > b {
			return false
		}
		i++
		j++
	}
	return false
}

type addResult int

const (
	ok   addResult = 0
	full addResult = 1
)

func (s sparse) size() uint32 {
	return binary.BigEndian.Uint32(s) & ^(uint32(1) << 31)
}

func (s sparse) setSize(sz uint32) {
	binary.BigEndian.PutUint32(s, sz)
}

func (s sortable) l() int {
	return len(s)
}

func (s sparse) sort() {
	sz := s.size()
	end := sz<<3 + 8
	t := sortable(s[8:end])
	sort.Sort(t)
	// Remove dups.
	to := 0
	from := 0
outer:
	for ; from < len(t); from += 8 {
		if from == 0 {
			to += 8
			continue
		}
		prev := from - 8
		i := from
		for k := 0; k < 8; k++ {
			if t[prev] != t[i] {
				if from == to {
					to += 8
					continue outer
				}
				j := from
				for k := 0; k < 8; k++ {
					t[to] = t[j]
					j++
					to++
				}
				continue outer
			}
			i++
			prev++
		}
	}
	// Clear the slack (having zeroes at the end could make hll compress better).
	s.setSize(uint32(to >> 3))
	for i := to; i < len(t); i++ {
		t[i] = 0
	}
}
