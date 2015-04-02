package minhash

import (
	"errors"
	"math/big"
)

const (
	// The maximum size (in bit) for the 1-bit minhash
	bitArraySize = 128
	onebitMask   = uint32(0x1)
)

func popCount(bits uint32) uint32 {
	bits = (bits & 0x55555555) + (bits >> 1 & 0x55555555)
	bits = (bits & 0x33333333) + (bits >> 2 & 0x33333333)
	bits = (bits & 0x0f0f0f0f) + (bits >> 4 & 0x0f0f0f0f)
	bits = (bits & 0x00ff00ff) + (bits >> 8 & 0x00ff00ff)
	return (bits & 0x0000ffff) + (bits >> 16 & 0x0000ffff)
}

func popCountBig(bits *big.Int) int {
	result := 0
	for _, v := range bits.Bytes() {
		result += int(popCount(uint32(v)))
	}
	return result
}

// OneBitMinHash see:
// http://research.microsoft.com/pubs/120078/wfc0398-lips.pdf
// This is the 1-bit signature that can be used to compute
// similarity with other 1-bit signatures.
// Using signature incurs some loss of precision, but reduced storage cost.
// Usually a good idea to use this when the actual Jaccard is > 0.5.
type OneBitMinHash struct {
	Size     int
	BitArray *big.Int
	Seed     int64
}

// ExportOneBit exports the full MinHash signature to OneBitMinHash.
// Keeping only the lowest bit of every hash value.
// If the number of hash permutation functions exceeds
// the maximum size of the bit array `bitArraySize`,
// only the first
// `bitArraySize` number of hash values will be exported.
func (sig *MinHash) ExportOneBit() *OneBitMinHash {
	var numExportedHashValues int
	if len(sig.Permutations) > bitArraySize {
		numExportedHashValues = bitArraySize
	} else {
		numExportedHashValues = len(sig.Permutations)
	}
	sigOneBit := OneBitMinHash{
		BitArray: big.NewInt(0),
		Seed:     sig.Seed,
		Size:     numExportedHashValues,
	}
	for i := 0; i < numExportedHashValues; i++ {
		sigOneBit.BitArray.SetBit(sigOneBit.BitArray, i,
			uint(sig.HashValues[i]&onebitMask))
	}
	return &sigOneBit
}

// EstimateJaccardOneBit estimates Jaccard similarity of OneBitMinHash signatures
func EstimateJaccardOneBit(sigs ...*OneBitMinHash) (float64, error) {
	if sigs == nil || len(sigs) == 0 {
		return 0.0, errors.New("Less than 2 OneBitMinHash signatures were given")
	}
	for _, sig := range sigs[1:] {
		if sigs[0].Seed != sig.Seed {
			return 0.0, errors.New("Cannot compare OneBitMinHash signatures " +
				"with different seed")
		}
		if sigs[0].Size != sig.Size {
			return 0.0, errors.New("Cannot compare OneBitMinHash signatures " +
				"with different numbers of permutations")
		}
	}
	commonBits := big.NewInt(0)
	for _, sig := range sigs {
		commonBits.Xor(commonBits, sig.BitArray)
	}
	return 2.0 * (float64((sigs[0].Size-popCountBig(commonBits)))/
		float64(sigs[0].Size) - 0.5), nil
}
