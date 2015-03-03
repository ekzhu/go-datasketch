// HyperLogLog is described here:
// http://algo.inria.fr/flajolet/Publications/FlFuGaMe07.pdf
// The code is shamelessly copied (with some modification) from
// https://github.com/clarkduvall/hyperloglog
// Jaccard similarity measure is computed using Inclusion-Exclusion
// principle.
package hyperloglog

import (
	"errors"
)

const two32 = 1 << 32

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

// Add adds a new item to HyperLogLog h.
func (h *HyperLogLog) Add(item Hash32) {
	x := item.Sum32()
	i := eb32(x, 32, 32-h.P) // {x31,...,x32-p}
	w := x<<h.P | 1<<(h.P-1) // {x32-p,...,x0}

	zeroBits := clz32(w) + 1
	if zeroBits > h.Reg[i] {
		h.Reg[i] = zeroBits
	}
}

// Merge takes another HyperLogLog and combines it with HyperLogLog h.
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

// Return the cardinality of the union
func (h *HyperLogLog) Union(other *HyperLogLog) (float64, error) {
	if h.P != other.P {
		return 0.0, errors.New("must have the same precision to compute Jaccard")
	}
	inverCount := func(val uint8) float64 {
		return 1.0 / float64(uint32(1)<<val)
	}
	sum := 0.0
	for i, v := range other.Reg {
		if v > h.Reg[i] {
			sum += inverCount(v)
		} else {
			sum += inverCount(h.Reg[i])
		}
	}
	fm := float64(h.M)
	est := alpha(h.M) * fm * fm / sum
	return correction(est, fm, h.Reg), nil
}

// Jaccard similarity using Inclusion-Exclusion principle
func (h *HyperLogLog) Jaccard(other *HyperLogLog) (float64, error) {
	u, err := h.Union(other)
	if err != nil {
		return 0.0, nil
	}
	if u == 0.0 {
		return 1.0, nil
	}
	return (h.Count() + other.Count() - u) / u, nil
}
