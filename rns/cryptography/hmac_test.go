package cryptography

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"hash"
	"testing"
)

type tinyHash struct {
	buf []byte
}

func (h *tinyHash) Write(p []byte) (int, error) {
	h.buf = append(h.buf, p...)
	return len(p), nil
}

func (h *tinyHash) Sum(b []byte) []byte {
	out := make([]byte, 4)
	copy(out, h.buf)
	return append(b, out...)
}

func (h *tinyHash) Reset()         { h.buf = nil }
func (h *tinyHash) Size() int      { return 4 }
func (h *tinyHash) BlockSize() int { return 8 }

func newTinyHash() hash.Hash { return &tinyHash{} }

func TestHMAC_MatchesStdlib_SHA256(t *testing.T) {
	maybeParallel(t)

	cases := []struct {
		name string
		key  []byte
		msg  []byte
	}{
		{"empty", []byte{}, []byte{}},
		{"short_key", []byte("k"), []byte("msg")},
		{"exact_block", bytes.Repeat([]byte{0x11}, 64), []byte("msg")},
		{"long_key", bytes.Repeat([]byte{0x22}, 200), bytes.Repeat([]byte("a"), 1000)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			std := hmac.New(sha256.New, tc.key)
			std.Write(tc.msg)
			want := std.Sum(nil)

			got := DigestFast(tc.key, tc.msg, sha256.New)
			if !bytes.Equal(got, want) {
				t.Fatalf("digest mismatch")
			}
		})
	}
}

func TestHMAC_CopyIsIndependent(t *testing.T) {
	maybeParallel(t)

	h := NewHMAC([]byte("key"), []byte("a"), sha256.New)
	c := h.Copy()

	h.Update([]byte("b"))
	c.Update([]byte("c"))

	if bytes.Equal(h.Digest(), c.Digest()) {
		t.Fatalf("expected different digests after divergent updates")
	}
}

func TestHMAC_DigestDoesNotMutateState(t *testing.T) {
	maybeParallel(t)

	h := NewHMAC([]byte("key"), []byte("a"), sha256.New)
	first := h.Digest()
	second := h.Digest()
	if !bytes.Equal(first, second) {
		t.Fatalf("Digest changed state")
	}

	h.Update([]byte("b"))
	got := h.Digest()

	std := hmac.New(sha256.New, []byte("key"))
	std.Write([]byte("ab"))
	want := std.Sum(nil)
	if !bytes.Equal(got, want) {
		t.Fatalf("digest after update mismatch")
	}
}

func TestHMAC_BlockSizeFallbackTo64(t *testing.T) {
	maybeParallel(t)

	h := NewHMAC([]byte("key"), nil, newTinyHash)
	if h.blockSize != 64 {
		t.Fatalf("blockSize=%d, want 64", h.blockSize)
	}
}

func TestHMAC_IncrementalUpdatesMatchOneShot(t *testing.T) {
	maybeParallel(t)

	parts := [][]byte{[]byte("reti"), []byte("cul"), []byte("um")}
	inc := New([]byte("key"), nil, sha256.New)
	for _, part := range parts {
		inc.Update(part)
	}
	oneShot := DigestFast([]byte("key"), bytes.Join(parts, nil), sha256.New)
	if !bytes.Equal(inc.Digest(), oneShot) {
		t.Fatalf("incremental digest mismatch")
	}
}

func TestHMAC_NewAliasAndHexDigest(t *testing.T) {
	maybeParallel(t)

	h := New([]byte("key"), []byte("msg"), sha256.New)
	if h == nil {
		t.Fatalf("New returned nil")
	}
	if len(h.HexDigest()) != h.digestSize*2 {
		t.Fatalf("unexpected hexdigest length: %d", len(h.HexDigest()))
	}
}
