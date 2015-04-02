package hyperloglog

import "testing"

type fakeHash32 uint32

func (f fakeHash32) Sum32() uint32 { return uint32(f) }

func TestHLLDigest(t *testing.T) {
	h, _ := New(16)

	h.Digest(fakeHash32(0x00010fff))
	n := h.Reg[1]
	if n != 5 {
		t.Error(n)
	}

	h.Digest(fakeHash32(0x0002ffff))
	n = h.Reg[2]
	if n != 1 {
		t.Error(n)
	}

	h.Digest(fakeHash32(0x00030000))
	n = h.Reg[3]
	if n != 17 {
		t.Error(n)
	}

	h.Digest(fakeHash32(0x00030001))
	n = h.Reg[3]
	if n != 17 {
		t.Error(n)
	}

	h.Digest(fakeHash32(0xff037000))
	n = h.Reg[0xff03]
	if n != 2 {
		t.Error(n)
	}

	h.Digest(fakeHash32(0xff030800))
	n = h.Reg[0xff03]
	if n != 5 {
		t.Error(n)
	}
}

func TestHLLCount(t *testing.T) {
	h, _ := New(16)

	n := h.Count()
	if n != 0 {
		t.Error(n)
	}

	h.Digest(fakeHash32(0x00010fff))
	h.Digest(fakeHash32(0x00020fff))
	h.Digest(fakeHash32(0x00030fff))
	h.Digest(fakeHash32(0x00040fff))
	h.Digest(fakeHash32(0x00050fff))
	h.Digest(fakeHash32(0x00050fff))

	n = h.Count()
	if int(n) != 5 {
		t.Error(n)
	}
}

func TestHLLMergeError(t *testing.T) {
	h, _ := New(16)
	h2, _ := New(10)

	err := h.Merge(h2)
	if err == nil {
		t.Error("different precision should return error")
	}
}

func TestHLLMerge(t *testing.T) {
	h, _ := New(16)
	h.Digest(fakeHash32(0x00010fff))
	h.Digest(fakeHash32(0x00020fff))
	h.Digest(fakeHash32(0x00030fff))
	h.Digest(fakeHash32(0x00040fff))
	h.Digest(fakeHash32(0x00050fff))
	h.Digest(fakeHash32(0x00050fff))

	h2, _ := New(16)
	h2.Merge(h)
	n := h2.Count()
	if int(n) != 5 {
		t.Error(n)
	}

	h2.Merge(h)
	n = h2.Count()
	if int(n) != 5 {
		t.Error(n)
	}

	h.Digest(fakeHash32(0x00060fff))
	h.Digest(fakeHash32(0x00070fff))
	h.Digest(fakeHash32(0x00080fff))
	h.Digest(fakeHash32(0x00090fff))
	h.Digest(fakeHash32(0x000a0fff))
	h.Digest(fakeHash32(0x000a0fff))
	n = h.Count()
	if int(n) != 10 {
		t.Error(n)
	}

	h2.Merge(h)
	n = h2.Count()
	if int(n) != 10 {
		t.Error(n)
	}
}

func TestHLLClear(t *testing.T) {
	h, _ := New(16)
	h.Digest(fakeHash32(0x00010fff))

	n := h.Count()
	if int(n) != 1 {
		t.Error(n)
	}
	h.Clear()

	n = h.Count()
	if int(n) != 0 {
		t.Error(n)
	}

	h.Digest(fakeHash32(0x00010fff))
	n = h.Count()
	if int(n) != 1 {
		t.Error(n)
	}
}

func TestHLLPrecision(t *testing.T) {
	h, _ := New(4)

	h.Digest(fakeHash32(0x1fffffff))
	n := h.Reg[1]
	if n != 1 {
		t.Error(n)
	}

	h.Digest(fakeHash32(0xffffffff))
	n = h.Reg[0xf]
	if n != 1 {
		t.Error(n)
	}

	h.Digest(fakeHash32(0x00ffffff))
	n = h.Reg[0]
	if n != 5 {
		t.Error(n)
	}
}

func TestHLLError(t *testing.T) {
	_, err := New(3)
	if err == nil {
		t.Error("precision 3 should return error")
	}

	_, err = New(17)
	if err == nil {
		t.Error("precision 17 should return error")
	}
}

func TestHLLSerialization(t *testing.T) {
	h, _ := New(4)
	h.Digest(fakeHash32(0x00ffffff))
	h.Digest(fakeHash32(0x10ffabc0))
	buffer := make([]byte, h.ByteSize())
	err := h.Serialize(buffer)
	if err != nil {
		t.Error(err)
	}
	d, err := Deserialize(buffer)
	if err != nil {
		t.Error(err)
	}
	if d.P != h.P {
		t.Error("Did not get back the same precision value.")
	}
	for i := range h.Reg {
		if h.Reg[i] != d.Reg[i] {
			t.Error("Did not get back the same register value.")
		}
	}
}

func TestHLLUnionCount(t *testing.T) {
	h1, _ := New(4)
	h1.Digest(fakeHash32(0x00ffffff))
	h1.Digest(fakeHash32(0x10ffabc0))
	h2, _ := New(4)
	h2.Digest(fakeHash32(0x00111111))
	h2.Digest(fakeHash32(0x1abcdef0))

	uc, err := UnionCount(h1, h2)
	if err != nil {
		t.Error(err)
	}
	err = h1.Merge(h2)
	if err != nil {
		t.Error(err)
	}
	uc2 := h1.Count()
	if uc != uc2 {
		t.Error("UnionCount did not return the same result as using " +
			"merge.")
	}
}

func TestHLLJaccard(t *testing.T) {
	h1, _ := New(4)
	h1.Digest(fakeHash32(0x00ffffff))
	h1.Digest(fakeHash32(0x10ffabc0))
	h2, _ := New(4)
	h2.Digest(fakeHash32(0x00111111))
	h2.Digest(fakeHash32(0x1abcdef0))

	j, err := Jaccard(h1, h2)
	if err != nil {
		t.Error(err)
	}
	if j > 1.0 {
		t.Error(j)
	}
}
