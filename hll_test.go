package hll

import (
	"log"
	"math"
	"math/rand"
	"sort"
	"testing"
)

func TestIsValid(t *testing.T) {
	for p := 4; p < 26; p++ {
		s, err := SizeByP(p)
		if err != nil {
			log.Panicln(err)
		}
		h := make(HLL, s)
		if h.IsValid() != nil {
			t.Fatal(h.IsValid())
		}
		// Dense.
		h[0] = 128 + 64
		if h.IsValid() != nil {
			t.Fatal(h.IsValid())
		}
		h.Reset()
		sparse(h).setSize(1)
		if h.IsValid() != nil {
			t.Fatal(h.IsValid())
		}
		sparse(h).setSize((1 << uint(p) * 3 / 4 / 8))
		if h.IsValid() != nil {
			t.Fatal(h.IsValid())
		}
		sparse(h).setSize((1<<uint(p)*3/4/8 + 1))
		if h.IsValid() == nil {
			t.Fatal("expected error")
		}
	}
	h := make(HLL, 4)
	if h.IsValid() == nil {
		t.Fatal("expected error")
	}
	h = make(HLL, 12)
	if h.IsValid() == nil {
		t.Fatal("expected error")
	}
	h = make(HLL, 14)
	if h.IsValid() == nil {
		t.Fatal("expected error")
	}
	h = make(HLL, 9)
	if h.IsValid() == nil {
		t.Fatal("expected error")
	}
	h = make(HLL, 17)
	if h.IsValid() == nil {
		t.Fatal("expected error")
	}
}

func TestAddMerge(t *testing.T) {
	n := []int{1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024}
	for p := 4; p < 26; p++ {
		s, err := SizeByP(p)
		if err != nil {
			log.Panicln(err)
		}

		dense := make(HLL, s)
		dense[0] = 64
		merged := make(HLL, s)
		all := make(HLL, s)
		tmp := make(HLL, s)
		for k, n := range n {
			tmp.Reset()
			for i := 0; i < n; i++ {
				v := xorShift64StarRound(i*1024 + k)
				all.Add(v)
				tmp.Add(v)
				dense.Add(v)
			}
			merged.Merge(tmp)
		}
		if all.EstimateCardinality() != merged.EstimateCardinality() {
			log.Fatal(p, all.EstimateCardinality(), merged.EstimateCardinality(), dense.EstimateCardinality())
		}
	}
}

func TestMergeSparseDense(t *testing.T) {
	s, err := SizeByP(8)
	if err != nil {
		log.Panicln(err)
	}
	sp := make(HLL, s)
	dense := make(HLL, s)
	dense[0] = 64
	dense.Merge(sp)
	if dense.EstimateCardinality() != 0 {
		t.Fatal()
	}
	s2 := make(HLL, s)
	s2.Merge(dense)
	if s2.EstimateCardinality() != 0 {
		t.Fatal()
	}
	if s2.IsSparse() {
		t.Fatal()
	}
	for i := 0; i < 3; i++ {
		sp.Add(xorShift64StarRound(i))
		dense.Add(xorShift64StarRound(i))
	}
	dc := dense.EstimateCardinality()
	sp.Merge(dense)
	if sp.EstimateCardinality() != dc {
		log.Println(sp)
		log.Fatal(sp.EstimateCardinality(), dc)
	}
	if sp[0]&128 != 0 {
		t.Fatal("dirty", dc)
	}
	for i := 0; i < 3; i++ {
		sp.Add(xorShift64StarRound(i))
	}
	if sp[0]&128 != 0 {
		t.Fatal("dirty")
	}
	s2.Reset()
	s2.Add(xorShift64StarRound(1))
	s2.Add(xorShift64StarRound(4))
	s2.Merge(sp)
	s2.EstimateCardinality()
	for i := 0; i < 4; i++ {
		sp.Add(xorShift64StarRound(i))
	}
	if s2[0]&128 != 0 {
		t.Fatal("dirty")
	}
}

func TestMergeDenseDense(t *testing.T) {
	s, err := SizeByP(8)
	if err != nil {
		log.Panicln(err)
	}
	dense := make(HLL, s)
	dense[0] = 64
	d2 := make(HLL, s)
	d2.Merge(dense)
	if d2.EstimateCardinality() != 0 {
		log.Println(d2)
		t.Fatal(d2.EstimateCardinality())
	}
	if d2.IsSparse() {
		t.Fatal()
	}
	for i := 0; i < 3; i++ {
		dense.Add(xorShift64StarRound(i))
	}
	for i := 1; i < 4; i++ {
		d2.Add(xorShift64StarRound(i))
	}
	d2.Merge(dense)
	d2.EstimateCardinality()
	if d2[0]&128 != 0 {
		t.Fatal("dirty")
	}
	for i := 0; i < 4; i++ {
		d2.Add(xorShift64StarRound(i))
	}
	if d2[0]&128 != 0 {
		t.Fatal("dirty")
	}
}

func TestMergeSparseSparse(t *testing.T) {
	s, err := SizeByP(8)
	if err != nil {
		log.Panicln(err)
	}
	sp := make(HLL, s)
	s2 := make(HLL, s)
	for i := 0; i < 3; i++ {
		sp.Add(xorShift64StarRound(i))
	}
	for i := 0; i < 3; i++ {
		s2.Add(xorShift64StarRound(i))
	}
	sp.Merge(s2)
	if !sp.IsSparse() {
		t.Fatal()
	}
	if sp.EstimateCardinality() != 3 {
		t.Fatal(sp.EstimateCardinality())
	}
	if sparse(sp).size() != 3 {
		t.Fatal(sparse(sp).size())
	}
	s2.Add(xorShift64StarRound(4))
	sp.Merge(s2)
	if !sp.IsSparse() {
		t.Fatal()
	}
	if sp.EstimateCardinality() != 4 {
		t.Fatal(sp.EstimateCardinality())
	}
	if sparse(sp).size() != 4 {
		t.Fatal(sparse(sp).size())
	}
	sp.Add(xorShift64StarRound(5))
	if !sp.IsSparse() {
		t.Fatal()
	}
	if sp.EstimateCardinality() != 5 {
		t.Fatal(sp.EstimateCardinality())
	}
	if sparse(sp).size() != 5 {
		t.Fatal(sparse(sp).size())
	}
	sp.Merge(s2)
	if !sp.IsSparse() {
		t.Fatal()
	}
	if sp.EstimateCardinality() != 5 {
		t.Fatal(sp.EstimateCardinality())
	}
	if sparse(sp).size() != 5 {
		t.Fatal(sparse(sp).size())
	}
}

func TestSizeByErrorErrorTooSmall(t *testing.T) {
	_, err := SizeByError(0.00001)
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestSizeByErrorErrorTooLarge(t *testing.T) {
	_, err := SizeByError(0.3)
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestMergeMismatchingSizes(t *testing.T) {
	s, err := SizeByError(0.001)
	if err != nil {
		log.Panicln(err)
	}
	s2, err := SizeByError(0.2)
	if err != nil {
		log.Panicln(err)
	}
	h := make(HLL, s)
	h2 := make(HLL, s2)
	if err := h.Merge(h2); err == nil {
		t.Fatal("erpected error")
	}
	if err := h2.Merge(h); err == nil {
		t.Fatal("erpected error")
	}
}

func TestEstimateCardinality(t *testing.T) {
	for p := 4; p < 26; p++ {
		s, err := SizeByP(p)
		if err != nil {
			log.Panicln(err)
		}
		h := make(HLL, s)
		for _, n := range []int{1, 2, 10, 100, 500, 1000, 5 * 1000, 10000, 50 * 1000, 100 * 1000, 500 * 1000, 1000 * 1000, 5 * 1000 * 1000} {
			h.Reset()
			for i := 0; i < n; i++ {
				h.Add(randUint64())
			}
			c := h.EstimateCardinality()
			errRate := math.Abs(float64(c)-float64(n)) / float64(n)
			if errRate > ErrFromP(p)*10 {
				t.Errorf("p: %d n: %d error rate: %g expected error rate: %g estimated: %d\n", p, n, errRate, ErrFromP(p), c)
			}
		}
	}
}

func TestEstimateCardinalityCached(t *testing.T) {
	for p := 4; p < 26; p++ {
		s, err := SizeByP(p)
		if err != nil {
			log.Panicln(err)
		}
		h := make(HLL, s)
		for _, n := range []int{5, 1000, 100000} {
			for _, sd := range []byte{0, 64} {
				h.Reset()
				h[0] = sd
				for i := 0; i < n; i++ {
					h.Add(xorShift64StarRound(i))
				}
				c := h.EstimateCardinality()
				c2 := h.EstimateCardinality()
				if c != c2 {
					log.Fatal("cardinality mismatch")
				}
			}
		}
	}
}

func TestBiasCorrecton(t *testing.T) {
	if len(rawEstimateData) != len(biasData) {
		t.Fatal("bias correction data is off")
	}
	for i, r := range rawEstimateData {
		if len(r) != len(biasData[i]) {
			t.Fatalf("bias correction data is off for index %d", i)
		}
		if !sort.IsSorted(sort.Float64Slice(r)) {
			for k := 1; k < len(r); k++ {
				if r[k] <= r[k-1] {
					t.Logf("raw  estimate data is not monotone for index %d:%d %v", i, k, r[k-1:k+1])
				}
			}
			if i > 2 {
				t.Fatalf("raw  estimate data is not sorted for index %d\n%v", i, r)
			}
		}
	}
}

func BenchmarkAdd(b *testing.B) {
	s, _ := SizeByP(14)
	h := make(HLL, s)
	b.ReportAllocs()
	b.ResetTimer()
	for i := uint64(0); i < uint64(b.N); i++ {
		h.Add(i)
	}
}

func BenchmarkEstimate(b *testing.B) {
	s, _ := SizeByP(14)
	h := make(HLL, s)
	h[0] |= 64 // Dense.
	for i := 0; i < 1<<16; i++ {
		h.Add(randUint64())
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.EstimateCardinality()
		h[0] |= 1 << 7 // Make it dirty.
	}
}

func BenchmarkMerge(b *testing.B) {
	s, _ := SizeByP(14)
	h := make(HLL, s)
	g := make(HLL, s)
	for i := 0; i < 1<<16; i++ {
		g.Add(randUint64())
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Merge(g)
		h[0] |= 127 // Make it dirty.
	}
}

func xorShift64StarRound(n int) uint64 {
	x := uint64(n)
	x ^= x >> 12
	x ^= x << 25
	x ^= x >> 27
	return x * 2685821657736338717
}

// WTF: https://groups.google.com/forum/#!topic/golang-nuts/Kle874lT1Eo/discussion
func randUint64() uint64 {
	return uint64(rand.Uint32())<<32 + uint64(rand.Uint32())
}
