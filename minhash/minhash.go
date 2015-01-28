// Implementation of MinHash with random permutation functions.
//
// The original MinHash paper:
// http://cs.brown.edu/courses/cs253/papers/nearduplicate.pdf
//
// The b-Bit MinWise Hashing:
// http://research.microsoft.com/pubs/120078/wfc0398-lips.pdf
// paper explains the idea behind the one-bit MinHash and storage space saving.
package minhash

import (
	"errors"
	"hash"
	"math"
	"math/big"
	"math/rand"
)

const (
	// The maximum size (in bit) for the 1-bit minhash
	BIT_ARRAY_SIZE = 128
	onebitMask     = uint32(0x1)
	mersennePrime  = (1 << 61) - 1
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

// http://en.wikipedia.org/wiki/Universal_hashing
type permutation func(uint32) uint32

func createPermutation(a, b uint32, p uint64, m int) permutation {
	return func(x uint32) uint32 {
		return uint32((uint(a*x+b) % uint(p)) % uint(m))
	}
}

// The MinHash signagure
type MinHash struct {
	Permutations []permutation
	HashValues   []uint32
	Seed         int64
}

// Create a new MinHash signature.
// `seed` is used to generate random permutation functions.
// `numPerm` number of permuation functions will
// be generated.
// Higher number of permutations results in better estimation,
// but reduces performance. 128 is a good number to start.
func New(seed int64, numPerm int) (*MinHash, error) {
	if numPerm <= 0 {
		return nil, errors.New("Cannot have non-positive number of permutations")
	}
	s := new(MinHash)
	s.HashValues = make([]uint32, numPerm)
	s.Permutations = make([]permutation, numPerm)
	s.Seed = seed
	rand.Seed(s.Seed)
	var a uint32
	for i := 0; i < numPerm; i++ {
		s.HashValues[i] = math.MaxUint32
		for {
			a = rand.Uint32()
			if a != 0 {
				break
			}
		}
		s.Permutations[i] = createPermutation(a,
			rand.Uint32(), mersennePrime, (1 << 32))
	}
	return s, nil
}

// Consumes a 32-bit hash and then computes all permutations and retains
// the minimum value for each permutations.
// Using a good hash function is decisive in estimation accuracy. See
// http://programmers.stackexchange.com/a/145633.
// You can use the murmur3 hash function in /hashfunc/murmur3 directory.
func (sig *MinHash) Digest(item hash.Hash32) {
	hv := item.Sum32()
	var phv uint32
	for i := range sig.Permutations {
		phv = (sig.Permutations[i])(hv)
		if phv < sig.HashValues[i] {
			sig.HashValues[i] = phv
		}
	}
}

// Compute the estimation of Jaccard Similarity among MinHash signatures.
func EstimateJaccard(sigs ...*MinHash) (float64, error) {
	if sigs == nil || len(sigs) == 0 {
		return 0.0, errors.New("Less than 2 MinHash signatures were given")
	}
	numPerm := len(sigs[0].Permutations)
	for _, sig := range sigs[1:] {
		if sigs[0].Seed != sig.Seed {
			return 0.0, errors.New("Cannot compare MinHash signatures with different seed")
		}
		if numPerm != len(sig.Permutations) {
			return 0.0, errors.New("Cannot compare MinHash signatures with different numbers of permutations")
		}
	}
	intersection := 0
	var currRowAgree int
	for i := 0; i < numPerm; i++ {
		currRowAgree = 1
		for _, sig := range sigs[1:] {
			if sigs[0].HashValues[i] != sig.HashValues[i] {
				currRowAgree = 0
				break
			}
		}
		intersection += currRowAgree
	}
	return float64(intersection) / float64(numPerm), nil
}

// For b-Bit MinHash see:
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

// Export the full MinHash signature to OneBitMinHash.
// Keeping only the lowest bit of every hash value.
// If the number of hash permutation functions exceeds
// the maximum size of the bit array `BIT_ARRAY_SIZE`,
// only the first
// `BIT_ARRAY_SIZE` number of hash values will be exported.
func (sig *MinHash) ExportOneBit() *OneBitMinHash {
	var numExportedHashValues int
	if len(sig.Permutations) > BIT_ARRAY_SIZE {
		numExportedHashValues = BIT_ARRAY_SIZE
	} else {
		numExportedHashValues = len(sig.Permutations)
	}
	sigOneBit := OneBitMinHash{
		BitArray: big.NewInt(0),
		Seed:     sig.Seed,
		Size:     numExportedHashValues,
	}
	for i := 0; i < numExportedHashValues; i++ {
		sigOneBit.BitArray.SetBit(sigOneBit.BitArray, i, uint(sig.HashValues[i]&onebitMask))
	}
	return &sigOneBit
}

// Estimate Jaccard similarity of OneBitMinHash signatures
func EstimateJaccardOneBit(sigs ...*OneBitMinHash) (float64, error) {
	if sigs == nil || len(sigs) == 0 {
		return 0.0, errors.New("Less than 2 OneBitMinHash signatures were given")
	}
	for _, sig := range sigs[1:] {
		if sigs[0].Seed != sig.Seed {
			return 0.0, errors.New("Cannot compare OneBitMinHash signatures with different seed")
		}
		if sigs[0].Size != sig.Size {
			return 0.0, errors.New("Cannot compare OneBitMinHash signatures with different numbers of permutations")
		}
	}
	commonBits := big.NewInt(0)
	for _, sig := range sigs {
		commonBits.Xor(commonBits, sig.BitArray)
	}
	return 2.0 * (float64((sigs[0].Size-popCountBig(commonBits)))/float64(sigs[0].Size) - 0.5), nil
}
