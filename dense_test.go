package hll

import (
	"log"
	"testing"

	"github.com/ScaledInference/si-srv/rng"
)

func TestVerifyDense(t *testing.T) {
	for p := 4; p < 26; p++ {
		s, err := DenseSizeByP(p)
		if err != nil {
			log.Panicln(err)
		}
		h := make(Dense, s)
		if h.IsValid() != nil {
			t.Fatal(h.IsValid)
		}

	}
}

func BenchmarkAddDense(b *testing.B) {
	s, _ := DenseSizeByP(14)
	h := make(Dense, s)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i <= b.N; i++ {
		h.Add(uint64(i))
	}
}

func BenchmarkEstimateDense(b *testing.B) {
	s, _ := DenseSizeByP(14)
	h := make(Dense, s)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i <= b.N; i++ {
		h.EstimateCardinality()
	}
}

func BenchmarkMergeDense(b *testing.B) {
	s, _ := DenseSizeByP(14)
	h := make(Dense, s)
	g := make(Dense, s)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i <= b.N; i++ {
		h.Merge(g)
	}
}

func TestGetSet(t *testing.T) {
	for p := 4; p <= 25; p++ {
		max := 1 << byte(p)
		m := map[int]byte{}
		s, err := DenseSizeByP(p)
		if err != nil {
			panic(p)
		}
		h := make(Dense, s)
		for i := 0; i < 100000; i++ {
			r := byte(rng.Get() & 63)
			idx := i % max
			h.set(idx, r)
			if h.get(idx) != r {
				t.Fatal(idx, r, h.get(idx))
			}
			m[idx] = r
		}
		for idx := 0; idx < max; idx++ {
			if m[idx] != h.get(idx) {
				t.Fatal(idx, m[idx], h.get(idx))
			}
		}
	}
}
