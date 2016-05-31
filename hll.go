package hll

import (
	"encoding/binary"
	"errors"
)

// HLL is a hybrid hyper-loglog: either sparse or dense, switching from sparse to dense when needed.
// Note, both sparse and dense representation take exactly same space.
// Dense representation performs no allocations, sparse might need some when switching to dense.
//
// Sparse mode estimate is exact.
// HLL is byte buffer friendly (no need to serialize/deserialize).
//
// Layout:
// First 8 bit header.
// Leftmost bit: dirty.
// Second left most: mode (1: dense, 0: sparse).
//
// sparse:
//   Next 30 bits: number of elements (big endian), 32 bits unused, followed by elements, 8 bytes each (uint64 little endian).
//   Note, the header is a part of sparse HLL.
//
// dense:
//   Next 62 bits: previous cardinality esimate (big endian). Valid if !dirty. Followed by dense HLL.
// full: 8 byte header, hll[0]&(1<<6) != 0, followed by dense HLL.
//
// All operations are in place. Add/Merge might allocate a temporary buffer when switching from sparse to dense representation.
// Note, this is not HLL++ - it uses a different sparse representation.
//
// Creating an HLL:
//
// s, err := SizeByP(p) // Or SizeByError.
// if err != nil {
//     log.Panicln(err)
// }
// h := make(HLL, s)
//
// h is empty and sparse at this point.
type HLL []byte

// SizeByError returns a byte size of an HLL for a given errorRate.
// The error must be between 0.0253% and 26% (inclusive).
func SizeByError(errorRate float64) (int, error) {
	s, err := DenseSizeByError(errorRate)
	if err != nil {
		return 0, err
	}
	return s + 8, nil
}

// SizeByP returns a byte size of an HLL for a given precision.
// Precision (p) must be between 4 and 25 (inclusive).
func SizeByP(p int) (int, error) {
	s, err := DenseSizeByP(p)
	if err != nil {
		return 0, err
	}
	return s + 8, nil
}

// Add a hash to an HLL.
// Might allocate a block (with Alloc) if HLL is sparse and it gets full.
// Make sure to use a good hash function.
func (h HLL) Add(hash uint64) {
	if h[0]&(1<<6) != 0 {
		if Dense(h[8:]).Add(hash) {
			h[0] |= 1 << 7 // Mark as dirty.
		}
		return
	}
	s := sparse(h)
	if s.Add(hash) == ok {
		return
	}
	toDense(s)
	Dense(h[8:]).Add(hash)
}

// Merge another HLL (of the same precision) into this.
// Might allocate a block (with Alloc) if HLL is sparse and it gets full.
func (h HLL) Merge(g HLL) error {
	if len(h) != len(g) {
		return errors.New("size mismatch")
	}
	hDense := h[0]&(1<<6) != 0
	gDense := g[0]&(1<<6) != 0
	if hDense && gDense {
		Dense(h[8:]).Merge(Dense(g[8:]))
		h[0] |= 1 << 7 // Mark as dirty.
		return nil
	}
	if !hDense && !gDense {
		if mergeIntoSparse(sparse(h), sparse(g)) == ok {
			return nil
		}
		toDense(sparse(h))
		mergeIntoDense(Dense(h[8:]), sparse(g))
		h[0] = 128 + 64
		return nil
	}
	if hDense { // !g.Dense
		mergeIntoDense(Dense(h[8:]), sparse(g))
		h[0] |= 128
		return nil
	}
	// h is sparse, g is Dense
	toDense(sparse(h))
	Dense(h[8:]).Merge(Dense(g[8:]))
	return nil
}

// IsValid checks whether HLL size makes sense.
func (h HLL) IsValid() error {
	if len(h) < 8 {
		return errors.New("size too small")
	}

	if (len(h)-8)%3 != 0 {
		return errors.New("hll byte size sould be 8 + a multiple of 3")
	}
	m := Dense(h[8:]).m()
	if m&(m-1) != 0 {
		return errors.New("hll byte size sould be 8 + 3 times a power of two")
	}
	var p byte
	for z := m; z != 0; z >>= 1 {
		p++
	}
	p--
	if p < 4 || p > 25 {
		return errors.New("p must be between 4 and 25, inclusive")
	}
	if h[0]&(1<<6) == 0 {
		sz := sparse(h).size()
		if len(h) < 8+8*int(sz) {
			return errors.New("sparse HLL is corrupted")
		}
	}
	return nil
}

// IsSparse returns true iff the underlying HLL is sparse (and thus the cardinality estimate is exact).
func (h HLL) IsSparse() bool {
	return h[0]&64 == 0
}

// EstimateCardinality returns a cardinality estimate.
// Note, EstimateCardinality might (will) modify the HLL iff HLL is dirty.
func (h HLL) EstimateCardinality() uint64 {
	if h[0]&(1<<6) != 0 {
		const mask = uint64(1<<63 + 1<<62)
		if h[0]&(1<<7) == 0 { // Not dirty.
			return binary.BigEndian.Uint64(h) & (^mask)
		}
		card := Dense(h[8:]).EstimateCardinality()
		if card&mask != 0 {
			// Wow. carinality is 2^62+. Keep it marked as dirty, so we keep recomputing this absurd cardinality.
			return card
		}
		binary.BigEndian.PutUint64(h, card|1<<62) // Clear the dirty bit.
		return card
	}
	return uint64(sparse(h).EstimateCardinality())
}

// Reset the HLL.
func (h HLL) Reset() {
	// Technically it is enough to clear the first 8 bytes. Let's be diligent.
	for i := range h {
		h[i] = 0
	}
}

// Alloc allocates the memory blob. It is a variable, so one can change it to use, say, sync.Pool.
var Alloc = func(n int) []byte {
	return make([]byte, n)
}

// Free returns the blob back. It is a variable, so one can change it to use, say, sync.Pool.
var Free = func(blob []byte) {
}

func toDense(s sparse) {
	tmp := Dense(Alloc(len(s) - 8))
	mergeIntoDense(tmp, s)
	copy(s[8:], tmp)
	Free(tmp)
	s[0] = 128 + 64 // dirty + dense
}

func mergeIntoDense(h Dense, s sparse) {
	sz := int(s.size())
	for i := 0; i < sz; i++ {
		s = s[8:]
		hash := binary.LittleEndian.Uint64(s)
		h.Add(hash)
	}
}

func mergeIntoSparse(t sparse, s sparse) addResult {
	sz := int(s.size())
	for i := 0; i < sz; i++ {
		s = s[8:]
		hash := binary.LittleEndian.Uint64(s)
		if t.Add(hash) == full {
			return full
		}
	}
	return ok
}
