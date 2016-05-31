package hll

import "testing"

func BenchmarkSort(b *testing.B) {
	s, _ := DenseSizeByP(14)
	h := make(sparse, s+8)
	l := len(h)/8 - 100
	for i := 0; i <= l; i++ {
		h.Add(uint64(-i))
	}
	h.sort()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i <= b.N; i++ {
		h.sort()
	}
}

func BenchmarkAddSparse(b *testing.B) {
	s, _ := DenseSizeByP(18)
	h := make(sparse, s+8)
	b.ReportAllocs()
	b.ResetTimer()
	l := len(h)/8 - 100
	k := 0
	for i := 0; i <= b.N; i++ {
		h.Add(uint64(-i))
		k++
		if k == l {
			h.setSize(0)
			k = 0
		}
	}
}
