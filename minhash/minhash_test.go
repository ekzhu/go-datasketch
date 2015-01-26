package minhash

import "testing"

func TestMinHash(t *testing.T) {
	m1, _ := New(1, 128)
	m2, _ := New(1, 128)

	m1.Digest(0x00010fff)
	m2.Digest(0x00010fff)

	est, _ := EstimateJaccard(m1, m2)
	if est != 1.0 {
		t.Error(est)
	}

	m3, _ := New(1, 128)
	m3.Digest(0x00010fff)
	m2.Digest(0x01001fff)
	est, _ = EstimateJaccard(m1, m2, m3)
	if est == 1.0 {
		t.Error(est)
	}
}

func TestMinHashError(t *testing.T) {
	_, err := New(0, 0)
	if err == nil {
		t.Error("should return error if number of permutations is set to 0")
	}

	m1, _ := New(1, 128)
	m2, _ := New(2, 128)
	_, err = EstimateJaccard(m1, m2)
	if err == nil {
		t.Error("should return error if seeds don't match")
	}

	m3, _ := New(1, 256)
	_, err = EstimateJaccard(m1, m3)
	if err == nil {
		t.Error("should return error if number of permutations don't match")
	}
}
