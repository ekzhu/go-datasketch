package minhash

import "testing"

type fakeHash32 uint32

func (f fakeHash32) Sum32() uint32 { return uint32(f) }

func TestMinHash(t *testing.T) {
	m1, _ := New(128, 1)
	m2, _ := New(128, 1)

	m1.Digest(fakeHash32(0x00010fff))
	m2.Digest(fakeHash32(0x00010fff))

	est, _ := Jaccard(m1, m2)
	if est != 1.0 {
		t.Error(est)
	}

	m3, _ := New(128, 1)
	m3.Digest(fakeHash32(0x00010fff))
	m2.Digest(fakeHash32(0x01001fff))
	est, _ = Jaccard(m1, m2, m3)
	if est == 1.0 {
		t.Error(est)
	}
}

func TestMinHashClear(t *testing.T) {
	m1, _ := New(128, 1)
	m2, _ := New(128, 1)

	m1.Digest(fakeHash32(0x00010fff))
	m2.Digest(fakeHash32(0x00010fff))

	m1.Clear()

	est, _ := Jaccard(m1, m2)
	if est != 0.0 {
		t.Error(est)
	}
}

func TestMinHashSerialization(t *testing.T) {
	m, _ := New(4, 1)
	m.Digest(fakeHash32(0x00010fff))
	m.Digest(fakeHash32(0x02010fff))
	buf := make([]byte, m.ByteSize())
	err := m.Serialize(buf)
	if err != nil {
		t.Error(err)
	}
	if len(buf) != m.ByteSize() {
		t.Error("Size of the buffer is changed.")
	}
	d, err := Deserialize(buf)
	if err != nil {
		t.Error(err)
	}
	if d.Seed != m.Seed {
		t.Error("Did not get back the same seed")
	}
	for i := range m.HashValues {
		if m.HashValues[i] != d.HashValues[i] {
			t.Error("Did not get back the same hash value")
		}
	}
}

func TestMinHashError(t *testing.T) {
	_, err := New(0, 0)
	if err == nil {
		t.Error("should return error if number of permutations is set to 0")
	}

	m1, _ := New(128, 1)
	m2, _ := New(128, 2)
	_, err = Jaccard(m1, m2)
	if err == nil {
		t.Error("should return error if seeds don't match")
	}

	m3, _ := New(256, 1)
	_, err = Jaccard(m1, m3)
	if err == nil {
		t.Error("should return error if number of permutations don't match")
	}
}
