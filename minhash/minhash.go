// Package minhash implements a probabilistic data structure for computing the
// similarity between datasets.
//
// The original MinHash paper:
// http://cs.brown.edu/courses/cs253/papers/nearduplicate.pdf
//
// The b-Bit MinWise Hashing:
// http://research.microsoft.com/pubs/120078/wfc0398-lips.pdf
// paper explains the idea behind the one-bit MinHash and storage space saving.
package minhash

import (
	"encoding/binary"
	"errors"
	"math"
	"math/rand"
)

// Hash32 is a relaxed version of hash.Hash32
type Hash32 interface {
	Sum32() uint32
}

const (
	mersennePrime = (1 << 61) - 1
)

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

// New creates a new MinHash signature.
// `seed` is used to generate random permutation functions.
// `numPerm` number of permuation functions will
// be generated.
// Higher number of permutations results in better estimation,
// but reduces performance. 128 is a good number to start.
func New(numPerm int, seed int64) (*MinHash, error) {
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

// Clear sets the MinHash back to initial state
func (sig *MinHash) Clear() {
	for i := range sig.HashValues {
		sig.HashValues[i] = math.MaxUint32
	}
}

// Digest consumes a 32-bit hash and then computes all permutations and retains
// the minimum value for each permutations.
// Using a good hash function is decisive in estimation accuracy. See
// http://programmers.stackexchange.com/a/145633.
// You can use the murmur3 hash function in /hashfunc/murmur3 directory.
func (sig *MinHash) Digest(item Hash32) {
	hv := item.Sum32()
	var phv uint32
	for i := range sig.Permutations {
		phv = (sig.Permutations[i])(hv)
		if phv < sig.HashValues[i] {
			sig.HashValues[i] = phv
		}
	}
}

// Merge takes another MinHash and combines it with MinHash sig,
// making sig the union of both.
func (sig *MinHash) Merge(other *MinHash) error {
	if sig.Seed != other.Seed {
		return errors.New("Cannot merge MinHashs with different seed.")
	}
	for i, v := range other.HashValues {
		if v < sig.HashValues[i] {
			sig.HashValues[i] = v
		}
	}
	return nil
}

// ByteSize returns the size of the serialized object.
func (sig *MinHash) ByteSize() int {
	return 8 + 4 + 4*len(sig.HashValues)
}

// Serialize the MinHash signature to bytes stored in buffer
func (sig *MinHash) Serialize(buffer []byte) error {
	if len(buffer) < sig.ByteSize() {
		return errors.New("The buffer does not have enough space to " +
			"hold the MinHash signature.")
	}
	b := binary.LittleEndian
	b.PutUint64(buffer, uint64(sig.Seed))
	b.PutUint32(buffer[8:], uint32(len(sig.HashValues)))
	offset := 8 + 4
	for _, v := range sig.HashValues {
		b.PutUint32(buffer[offset:], v)
		offset += 4
	}
	return nil
}

// Deserialize reconstructs a MinHash signature from the buffer
func Deserialize(buffer []byte) (*MinHash, error) {
	if len(buffer) < 12 {
		return nil, errors.New("The buffer does not contain enough bytes to " +
			"reconstruct a MinHash.")
	}
	b := binary.LittleEndian
	seed := int64(b.Uint64(buffer))
	numPerm := int(b.Uint32(buffer[8:]))
	offset := 12
	if len(buffer[offset:]) < numPerm {
		return nil, errors.New("The buffer does not contain enough bytes to " +
			"reconstruct a MinHash.")
	}
	m, err := New(numPerm, seed)
	if err != nil {
		return nil, err
	}
	for i := range m.HashValues {
		m.HashValues[i] = b.Uint32(buffer[offset:])
		offset += 4
	}
	return m, nil
}

// Jaccard computes the estimation of Jaccard Similarity among
// MinHash signatures.
func Jaccard(sigs ...*MinHash) (float64, error) {
	if sigs == nil || len(sigs) < 2 {
		return 0.0, errors.New("Less than 2 MinHash signatures were given")
	}
	numPerm := len(sigs[0].Permutations)
	for _, sig := range sigs[1:] {
		if sigs[0].Seed != sig.Seed {
			return 0.0, errors.New("Cannot compare MinHash signatures with " +
				"different seed")
		}
		if numPerm != len(sig.Permutations) {
			return 0.0, errors.New("Cannot compare MinHash signatures with " +
				"different numbers of permutations")
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
