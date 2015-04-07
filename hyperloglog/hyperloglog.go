// Package hyperloglog implements a probabilistic data structure for estimating
// cardinality.
// HyperLogLog is described here:
// http://algo.inria.fr/flajolet/Publications/FlFuGaMe07.pdf.
// The code is shamelessly modified based on
// https://github.com/clarkduvall/hyperloglog.
package hyperloglog

import (
	"errors"
	"math"
)

const two32 = 1 << 32

// HyperLogLog data structure
type HyperLogLog struct {
	Reg []uint8
	M   uint32
	P   uint8
}

// New returns a new initialized HyperLogLog.
func New(precision uint8) (*HyperLogLog, error) {
	if precision > 16 || precision < 4 {
		return nil, errors.New("precision must be between 4 and 16")
	}

	h := &HyperLogLog{}
	h.P = precision
	h.M = 1 << precision
	h.Reg = make([]uint8, h.M)
	return h, nil
}

// Clear sets HyperLogLog h back to its initial state.
func (h *HyperLogLog) Clear() {
	h.Reg = make([]uint8, h.M)
}

// Digest adds a new item to HyperLogLog h.
func (h *HyperLogLog) Digest(item Hash32) {
	x := item.Sum32()
	i := eb32(x, 32, 32-h.P) // {x31,...,x32-p}
	w := x<<h.P | 1<<(h.P-1) // {x32-p,...,x0}

	zeroBits := clz32(w) + 1
	if zeroBits > h.Reg[i] {
		h.Reg[i] = zeroBits
	}
}

// Merge takes another HyperLogLog and combines it with HyperLogLog h,
// making h the union of both.
func (h *HyperLogLog) Merge(other *HyperLogLog) error {
	if h.P != other.P {
		return errors.New("precisions must be equal")
	}

	for i, v := range other.Reg {
		if v > h.Reg[i] {
			h.Reg[i] = v
		}
	}
	return nil
}

// Count returns the cardinality estimate.
func (h *HyperLogLog) Count() float64 {
	est := calculateEstimate(h.Reg)
	return correction(est, float64(h.M), h.Reg)
}

// ByteSize returns the size of the HyperLogLog h in bytes
func (h *HyperLogLog) ByteSize() int {
	return 1 + int(h.M)
}

// Serialize the HyperLogLog h into bytes and store in the buffer
func (h *HyperLogLog) Serialize(buffer []byte) error {
	if len(buffer) < h.ByteSize() {
		return errors.New("buffer does not have enough space for holding" +
			" this HyperLogLog.")
	}
	buffer[0] = h.P
	offset := 1
	for _, v := range h.Reg {
		buffer[offset] = v
		offset++
	}
	return nil
}

// Deserialize reconstruct a HyperLogLog from the buffer
func Deserialize(buffer []byte) (*HyperLogLog, error) {
	p := buffer[0]
	m := 1 << p
	if len(buffer) < int(m)+1 {
		return nil, errors.New("buffer doesn't contain enough space for " +
			"reconstructing a HyperLogLog.")
	}
	h, err := New(p)
	if err != nil {
		return nil, err
	}
	offset := 1
	for i := range h.Reg {
		h.Reg[i] = buffer[offset]
		offset++
	}
	return h, nil
}

// UnionCount returns the cardinality of the union of all the HyperLogLogs.
// This is more memory efficient than creating a new HyperLogLog and merging
// with others.
func UnionCount(hlls ...*HyperLogLog) (float64, error) {
	if hlls == nil || len(hlls) < 2 {
		return 0.0, errors.New("Less than 2 HyperLogLogs were given.")
	}
	p := hlls[0].P
	for _, h := range hlls[1:] {
		if h.P != p {
			return 0.0, errors.New("Cannot union HyperLogLogs with different" +
				"precision parameters.")
		}
	}
	inverCount := func(val uint8) float64 {
		return 1.0 / float64(uint32(1)<<val)
	}
	sum := 0.0
	var numZero uint32
	for i, v := range hlls[0].Reg {
		maxV := v
		for _, h := range hlls[1:] {
			if h.Reg[i] > v {
				maxV = h.Reg[i]
			}
		}
		sum += inverCount(maxV)
		if maxV == 0 {
			numZero++
		}
	}
	fm := float64(hlls[0].M)
	est := alpha(hlls[0].M) * fm * fm / sum
	if est <= fm*2.5 {
		if numZero != 0 {
			return linearCounting(fm, numZero), nil
		}
		return est, nil
	} else if est < two32/30 {
		return est, nil
	}
	return float64(-uint64(two32 * math.Log(1-est/two32))), nil
}

// IntersectionCount returns the cardinality estimation of the intersection
// of the two HyperLogLogs.
// This uses the Inclusion-Exclusion Principle.
// The value may be negative due to cardinality estimation error
func IntersectionCount(h1, h2 *HyperLogLog) (float64, error) {
	u, err := UnionCount(h1, h2)
	if err != nil {
		return 0.0, err
	}
	return (h1.Count() + h2.Count() - u), nil
}

// Jaccard returns the estimated Jaccard similarity between the two HyperLogLogs.
// The value may be negative due to cardinality estimation error
func Jaccard(h1, h2 *HyperLogLog) (float64, error) {
	u, err := UnionCount(h1, h2)
	if err != nil {
		return 0.0, err
	}
	if u == 0.0 {
		return 1.0, nil
	}
	ic := h1.Count() + h2.Count() - u
	return ic / u, nil
}

// Inclusion returns the estimated inclusion score of h1 against h2.
// It measures the fraction of the multiset counted by h1
// overlapping with the multiset counted by h2.
// The value may be negative due to estimation error.
func Inclusion(h1, h2 *HyperLogLog) (float64, error) {
	u, err := UnionCount(h1, h2)
	if err != nil {
		return 0.0, err
	}
	if u == 0.0 {
		return 1.0, nil
	}
	c := h1.Count()
	ic := c + h2.Count() - u
	return ic / c, nil
}
