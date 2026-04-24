package rns

import (
	"testing"

	umsgpack "github.com/svanichkin/go-reticulum/rns/vendor"
)

func TestResourceAdvertisement_RoundTrip_HashmapSegment(t *testing.T) {
	// Basic sanity for msgpack pack/unpack of adv payload.
	a := ResourceAdvertisement{
		T:    123,
		D:    456,
		N:    2,
		H:    []byte{1, 2, 3, 4},
		R:    []byte{5, 6, 7, 8},
		O:    []byte{9, 10, 11, 12},
		I:    1,
		L:    1,
		Q:    nil,
		F:    0,
		M:    []byte{0xAA, 0xBB, 0xCC, 0xDD, 0x01, 0x02, 0x03, 0x04},
		Link: nil,
	}

	raw := a.Pack(0)
	if len(raw) == 0 {
		t.Fatalf("Pack returned empty")
	}
	var decoded any
	_ = umsgpack.Unpackb(raw, &decoded)
	t.Logf("decoded type=%T val=%#v", decoded, decoded)
	var decodedMap map[any]any
	_ = umsgpack.Unpackb(raw, &decodedMap)
	t.Logf("decodedMap type=%T val=%#v", decodedMap, decodedMap)
	b, err := (ResourceAdvertisement{}).Unpack(raw)
	if err != nil {
		t.Fatalf("Unpack: %v", err)
	}
	if b == nil {
		t.Fatalf("Unpack returned nil")
	}
	t.Logf("unpacked: T=%d D=%d N=%d len(M)=%d F=%d", b.T, b.D, b.N, len(b.M), b.F)
	if b.T != a.T || b.D != a.D || b.N != a.N {
		t.Fatalf("numeric mismatch")
	}
	if len(b.M) != 8 {
		t.Fatalf("expected hashmap bytes, got len=%d", len(b.M))
	}
}

func TestResourceAdvertisement_RoundTrip_LargeIntegersAndHashmap(t *testing.T) {
	// Exercise STR8/BIN types and uint32/int32 sizes that appear for mini+ resources.
	const (
		transferSize = 256064
		dataSize     = 256000
		parts        = 552
	)
	hashmapLen := 74 // derived from MDU=431 and AdvOverhead=134: floor((431-134)/4)=74
	if hashmapLen <= 0 {
		t.Fatalf("invalid hashmapLen")
	}
	hashmapBytes := make([]byte, hashmapLen*MapHashLen)
	for i := range hashmapBytes {
		hashmapBytes[i] = byte(i)
	}

	a := ResourceAdvertisement{
		T:    transferSize,
		D:    dataSize,
		N:    parts,
		H:    []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		R:    []byte{9, 9, 9, 9},
		O:    []byte{1, 1, 1, 1},
		I:    1,
		L:    1,
		Q:    nil,
		F:    0x01,
		M:    hashmapBytes,
		Link: nil,
	}

	raw := a.Pack(0)
	if len(raw) == 0 {
		t.Fatalf("Pack returned empty")
	}
	var decodedMap map[any]any
	if err := umsgpack.Unpackb(raw, &decodedMap); err != nil {
		t.Fatalf("umsgpack.Unpackb: %v", err)
	}
	t.Logf("decodedMap=%#v", decodedMap)
	b, err := (ResourceAdvertisement{}).Unpack(raw)
	if err != nil {
		t.Fatalf("Unpack: %v", err)
	}
	if b.T != a.T || b.D != a.D || b.N != a.N {
		t.Fatalf("numeric mismatch: got T=%d D=%d N=%d", b.T, b.D, b.N)
	}
	if len(b.M) != hashmapLen*MapHashLen {
		t.Fatalf("hashmap segment mismatch: got len=%d want %d", len(b.M), hashmapLen*MapHashLen)
	}
}
