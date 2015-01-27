package hllminhash

import (
	"errors"
	"math"
)

const two32 = 1 << 32

func alpha(m uint32) float64 {
	if m == 16 {
		return 0.673
	} else if m == 32 {
		return 0.697
	} else if m == 64 {
		return 0.709
	}
	return 0.7213 / (1 + 1.079/float64(m))
}

var clzLookup = []uint8{
	32, 31, 30, 30, 29, 29, 29, 29, 28, 28, 28, 28, 28, 28, 28, 28,
}

// This optimized clz32 algorithm is from:
//       http://embeddedgurus.com/state-space/2014/09/
//                      fast-deterministic-and-portable-counting-leading-zeros/
func clz32(x uint32) uint8 {
	var n uint8

	if x >= (1 << 16) {
		if x >= (1 << 24) {
			if x >= (1 << 28) {
				n = 28
			} else {
				n = 24
			}
		} else {
			if x >= (1 << 20) {
				n = 20
			} else {
				n = 16
			}
		}
	} else {
		if x >= (1 << 8) {
			if x >= (1 << 12) {
				n = 12
			} else {
				n = 8
			}
		} else {
			if x >= (1 << 4) {
				n = 4
			} else {
				n = 0
			}
		}
	}
	return clzLookup[x>>n] - n
}

func countZeros(s []uint8) uint32 {
	var c uint32
	for _, v := range s {
		if v == 0 {
			c++
		}
	}
	return c
}

// Extract bits from uint32 using LSB 0 numbering, including lo
func eb32(bits uint32, hi uint8, lo uint8) uint32 {
	m := uint32(((1 << (hi - lo)) - 1) << lo)
	return (bits & m) >> lo
}

func linearCounting(m uint32, v uint32) float64 {
	fm := float64(m)
	return fm * math.Log(fm/float64(v))
}

func calculateEstimate(s []uint8) float64 {
	sum := 0.0
	for _, val := range s {
		sum += 1.0 / float64(uint32(1)<<val)
	}

	m := uint32(len(s))
	fm := float64(m)
	return alpha(m) * fm * fm / sum
}

type HllMinHash struct {
	reg   []uint8
	minhv []uint32
	m     uint32
	p     uint8
}

// New returns a new initialized HllMinHash.
func New(precision uint8) (*HllMinHash, error) {
	if precision > 16 || precision < 4 {
		return nil, errors.New("precision must be between 4 and 16")
	}

	h := &HllMinHash{}
	h.p = precision
	h.m = 1 << precision
	h.reg = make([]uint8, h.m)
	h.minhv = make([]uint32, h.m)
	for i := range h.minhv {
		h.minhv[i] = math.MaxUint32
	}
	return h, nil
}

// Clear sets HllMinHash back to its initial state.
func (h *HllMinHash) Clear() {
	h.reg = make([]uint8, h.m)
	h.minhv = make([]uint32, h.m)
	for i := range h.minhv {
		h.minhv[i] = math.MaxUint32
	}
}

// Add adds a new 32 bit hashed value to HllMinHash.
func (h *HllMinHash) Add(hv uint32) {
	j := eb32(hv, 32, 32-h.p) // {x31,...,x32-p}
	w := hv<<h.p | 1<<(h.p-1) // {x32-p,...,x0}

	// HyperLogLog part
	zeroBits := clz32(w) + 1
	if zeroBits > h.reg[j] {
		h.reg[j] = zeroBits
	}

	// MinHash part
	if w < h.minhv[j] {
		h.minhv[j] = w
	}
}

// Merge two HllMinHash instances
func (h *HllMinHash) Merge(other *HllMinHash) error {
	if h.p != other.p {
		return errors.New("Merging instances must have the same precision")
	}
	// Merge the HyperLogLog registers
	for i, v := range other.reg {
		if v > h.reg[i] {
			h.reg[i] = v
		}
	}
	// Merge the MinHash part
	for i, hv := range other.minhv {
		if hv < h.minhv[i] {
			h.minhv[i] = hv
		}
	}
	return nil
}

// Count returns the cardinality estimate
func (h *HllMinHash) Count() uint64 {
	est := calculateEstimate(h.reg)
	if est <= float64(h.m)*2.5 {
		if v := countZeros(h.reg); v != 0 {
			return uint64(linearCounting(h.m, v))
		}
		return uint64(est)
	} else if est < two32/30 {
		return uint64(est)
	}
	return -uint64(two32 * math.Log(1-est/two32))
}

// Jaccard returns the jaccard similarity estimate between
// two HllMinHash instances
func (h *HllMinHash) Jaccard(other *HllMinHash) (float64, error) {
	if h.p != other.p {
		return 0.0, errors.New("Instances must have the same precision to compute Jaccard")
	}
	intersection := 0
	for i, hv := range other.minhv {
		if hv == h.minhv[i] {
			intersection++
		}
	}
	return float64(intersection) / float64(h.m), nil
}
