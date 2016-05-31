package hll

import (
	"errors"
	"math"

	"github.com/dgryski/go-bits"
)

// Dense implements hyper-log log from http://research.google.com/pubs/pub40671.html
// With bias correction and linear counting, but not the HLL++ part (not sparse).
// HLL uses constant memory (0.75 * 2^p bytes.)
// All operations are non-allocating.
type Dense []byte

// DenseSizeByError returns a byte size of a dense HLL for a given errorRate.
// The error must be between 0.0253% and 26% (inclusive).
func DenseSizeByError(errorRate float64) (int, error) {
	if errorRate < 0.00025390625 || errorRate > 0.26 {
		return 0, errors.New("error rate must be between 0.00025390625 and 0.26 (inclusive)")
	}
	p := int(math.Ceil(math.Log2(math.Pow(1.04/errorRate, 2))))
	return DenseSizeByP(p)
}

// ErrFromP returns an expected error rate for a given p.
func ErrFromP(p int) float64 {
	return 1.04 / math.Sqrt(math.Pow(2, float64(p)))
}

// DenseSizeByP returns a byte size of a dense HLL for a given precision.
// Precision (p) must be between 4 and 25 (inclusive).
func DenseSizeByP(p int) (int, error) {
	if p < 4 || p > 25 {
		return 0, errors.New("p must be between 4 and 25, inclusive")
	}
	return (1 << byte(p) * 3) >> 2, nil
}

// Clear resets the HLL.
func (h Dense) Clear() {
	for i := range h {
		h[i] = 0
	}
}

// IsValid checks whether HLL size makes sense.
func (h Dense) IsValid() error {
	if len(h)%3 != 0 {
		return errors.New("hll byte size sould be a multiple of 3")
	}
	m := h.m()
	if m&(m-1) != 0 {
		return errors.New("hll byte size sould be 3 times a power of two")
	}
	var p byte
	for z := m; z != 0; z >>= 1 {
		p++
	}
	p--
	if p < 4 || p > 25 {
		return errors.New("p must be between 4 and 25, inclusive")
	}
	return nil
}

// Add a hash to an HLL.
// Returns true if cardinality esimate changed.
func (h Dense) Add(hash uint64) bool {
	mask := uint64(h.m()) - 1
	idx := hash & mask
	urho := bits.Clz(hash) + 1
	// We are using low bits of hash to get the index.
	// We also count the number of leading zeroes in the whole hash.
	// This skews the distribution of values somewhat, seems to have a minor effect on cardinality estimation error rate.
	if urho > 63 {
		urho = 63
	}
	rho := byte(urho)

	bp := idx >> 2
	bp *= 3
	s := idx & 3
	// We store 4 6-bit registers in 3 consecutive bytes.
	// Layout:
	// (a5, a4, a3, a2, a1, a0, d5, d4) -- byte 0
	// (b5, b4, b3, b2, b1, b0, d3, d2) -- byte 1
	// (c5, c4, c3, c2, c1, c0, d1, d0) -- byte 2
	// Here a, b, c, and d are the registers (in order).
	// a5 is the highest bit, a0 is the lowest.
	if s != 3 {
		b := h[bp+s]
		v := b >> 2
		if v < rho {
			h[bp+s] = (b & 3) | (rho << 2)
			return true
		}
		return false
	}
	x0, x1, x2 := h[bp], h[bp+1], h[bp+2]
	const m2 = ^byte(3)
	xL0, xL1, xL2 := x0&^m2, x1&^m2, x2&^m2
	v := xL0<<4 ^ xL1<<2 ^ xL2
	if v >= rho {
		return false
	}
	h[bp] = (x0 & m2) ^ (rho >> 4)
	h[bp+1] = (x1 & m2) ^ ((rho >> 2) & 3)
	h[bp+2] = (x2 & m2) ^ (rho & 3)
	return true
}

// Merge another HLL (of the same precision) into this.
func (h Dense) Merge(g Dense) error {
	if len(h) != len(g) {
		return errors.New("size mismatch")
	}
	for i := 0; i < len(h); i += 3 {
		x0, x1, x2 := h[i], h[i+1], h[i+2]
		y0, y1, y2 := g[i], g[i+1], g[i+2]
		const m = ^byte(3)
		xH0, xH1, xH2 := x0&m, x1&m, x2&m
		xL0, xL1, xL2 := x0&^m, x1&^m, x2&^m
		yH0, yH1, yH2 := y0&m, y1&m, y2&m
		yL0, yL1, yL2 := y0&^m, y1&^m, y2&^m
		var r0, r1, r2 byte
		if xH0 > yH0 {
			r0 = xH0
		} else {
			r0 = yH0
		}
		if xH1 > yH1 {
			r1 = xH1
		} else {
			r1 = yH1
		}
		if xH2 > yH2 {
			r2 = xH2
		} else {
			r2 = yH2
		}
		xx := xL0<<4 ^ xL1<<2 ^ xL2
		yy := yL0<<4 ^ yL1<<2 ^ yL2
		if xx > yy {
			h[i] = r0 ^ xL0
			h[i+1] = r1 ^ xL1
			h[i+2] = r2 ^ xL2
		} else {
			h[i] = r0 ^ yL0
			h[i+1] = r1 ^ yL1
			h[i+2] = r2 ^ yL2
		}
	}
	return nil
}

func (h Dense) get(idx int) byte {
	bp := idx >> 2
	bp *= 3
	s := idx & 3
	if s != 3 {
		b := h[bp+s]
		v := b >> 2
		return v
	}
	x0, x1, x2 := h[bp], h[bp+1], h[bp+2]
	const mask = ^byte(3)
	xL0, xL1, xL2 := x0&^mask, x1&^mask, x2&^mask
	v := xL0<<4 ^ xL1<<2 ^ xL2
	return v
}

func (h Dense) set(idx int, v byte) {
	bp := idx >> 2
	bp *= 3
	s := idx & 3
	if s != 3 {
		b := h[bp+s]
		h[bp+s] = b&3 ^ v<<2
	} else {
		x0, x1, x2 := h[bp], h[bp+1], h[bp+2]
		const mask = ^byte(3)
		h[bp] = (x0 & mask) ^ (v >> 4)
		h[bp+1] = (x1 & mask) ^ ((v >> 2) & 3)
		h[bp+2] = (x2 & mask) ^ (v & 3)
	}
}

func (h Dense) addSlow(hash uint64) {
	mask := uint64(h.m()) - 1
	idx := int(hash & mask)
	urho := bits.Clz(hash) + 1
	if urho > 63 {
		urho = 63
	}
	rho := byte(urho)

	v := h.get(idx)
	if v >= rho {
		return
	}
	h.set(idx, rho)
}

// EstimateCardinality returns a cardinality estimate.
func (h Dense) EstimateCardinality() uint64 {
	var V int
	var invSum float64
	for i := 0; i < len(h); i += 3 {
		x0, x1, x2 := h[i], h[i+1], h[i+2]
		v0, v1, v2 := x0>>2, x1>>2, x2>>2
		xL0, xL1, xL2 := x0&3, x1&3, x2&3
		v3 := xL0<<4 ^ xL1<<2 ^ xL2
		invSum += lookup[v0]
		invSum += lookup[v1]
		invSum += lookup[v2]
		invSum += lookup[v3]
		if v0 == 0 {
			V++
		}
		if v1 == 0 {
			V++
		}
		if v2 == 0 {
			V++
		}
		if v2 == 0 {
			V++
		}
	}

	est := h.correctedEstimate(invSum, V)
	card := math.Floor(est + 0.5)
	if card > math.MaxUint64 {
		return math.MaxUint64
	}
	return uint64(card)
}

func (h *Dense) correctedEstimate(invSum float64, V int) float64 {
	m := h.m()
	mf := float64(m)
	e := h.alpha() * mf * mf / invSum

	var p byte
	for z := m; z != 0; z >>= 1 {
		p++
	}
	p--
	// bias
	if e < 5*mf {
		e -= estimateBias(e, p)
	}

	H := e
	if V != 0 {
		H = linearCounting(m, V)
	}

	if H <= threshold(m) {
		return H
	}
	return e
}

var lookup [256]float64

func init() {
	for i := 0; i < 256; i++ {
		lookup[i] = math.Pow(2, -float64(i))
	}
}

func threshold(m int) float64 {
	// Those thresholds are not from the original article.
	switch m {
	case 1 << 4:
		return 13
	case 1 << 5:
		return 40
	case 1 << 6:
		return 70
	case 1 << 7:
		return 180
	case 1 << 8:
		return 225
	case 1 << 9:
		return 1000
	case 1 << 10:
		return 1750
	case 1 << 11:
		return 4600
	case 1 << 12:
		return 10 * 1000
	case 1 << 13:
		return 22 * 1000
	case 1 << 14:
		return 45 * 1000
	case 1 << 15:
		return 80 * 1000
	case 1 << 16:
		return 150 * 1000
	case 1 << 17:
		return 400 * 1000
	case 1 << 18:
		return 700 * 1000
	case 1 << 19:
		return 1850 * 1000
	case 1 << 20:
		return 4200 * 1000
	default:
		return float64(m) * 8
	}
}

// linearCounting performs linear counting given the number of registers, m1, and the number
// of empty registers, V
func linearCounting(m int, V int) float64 {
	return float64(m) * math.Log(float64(m)/float64(V))
}

// estimateBias estimates the amount of bias in cardinality
// with an estimator value of e and precision p.
func estimateBias(x float64, p byte) float64 {
	if p > 18 {
		return 0.0
	}
	estimateVector := rawEstimateData[p-4]
	N := len(estimateVector)
	if x < estimateVector[0] || x > estimateVector[N-1] {
		return 0.0
	}
	biasV := biasData[p-4]
	// rawEstimates are almost monotone -- there are a few small inversions for p=2 and p=3.
	// It does not matter much.
	for i := 0; i < len(estimateVector)-1; i++ {
		a := estimateVector[i]
		if x == a {
			return biasV[i]
		}
		if x > a {
			continue
		}
		b := estimateVector[i+1]
		vb := biasV[i+1]
		va := biasV[i]
		r := (x-a)*vb + (b-x)*va
		return r / (b - a)
	}
	return 0
}

func (h Dense) m() int {
	return (len(h) / 3) << 2
}

func (h Dense) alpha() float64 {
	m := h.m()
	switch m {
	case 16:
		return 0.673
	case 32:
		return 0.697
	case 64:
		return 0.709
	default:
		return 0.7213 / (1 + 1.079/float64(m))
	}
}
