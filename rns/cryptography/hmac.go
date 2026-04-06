package cryptography

import (
	"crypto/sha256"
	"errors"
	"hash"
)

// Precomputed XOR tables like Python: trans_5C and trans_36.
var (
	trans5C [256]byte
	trans36 [256]byte
)

func init() {
	for i := 0; i < 256; i++ {
		trans5C[i] = byte(i) ^ 0x5c
		trans36[i] = byte(i) ^ 0x36
	}
}

type HMAC struct {
	key        []byte // normalized and zero-padded key
	blockSize  int    // effective block_size
	digestSize int    // digest size
	digestmod  func() hash.Hash
	data       []byte // retained for copy/digest fallback when hash state can't be cloned
	inner      hash.Hash
	outer      hash.Hash
}

type binaryCloner interface {
	MarshalBinary() ([]byte, error)
	UnmarshalBinary([]byte) error
}

// NewHMAC mirrors new(key, msg=None, digestmod=sha256).
func NewHMAC(key, msg []byte, digestmod func() hash.Hash) *HMAC {
	if digestmod == nil {
		digestmod = sha256.New
	}

	h0 := digestmod()
	blockSize := h0.BlockSize()
	if blockSize < 16 {
		blockSize = 64 // fallback (like Python)
	}

	// if key is longer than block_size, hash it
	if len(key) > blockSize {
		h := digestmod()
		h.Write(key)
		key = h.Sum(nil)
	}

	// pad with zeros up to block_size
	if len(key) < blockSize {
		k := make([]byte, blockSize)
		copy(k, key)
		key = k
	}

	h := &HMAC{
		key:        key,
		blockSize:  blockSize,
		digestSize: h0.Size(),
		digestmod:  digestmod,
		data:       nil,
	}

	kInner := make([]byte, len(key))
	kOuter := make([]byte, len(key))
	for i, b := range key {
		kInner[i] = trans36[b]
		kOuter[i] = trans5C[b]
	}

	h.inner = digestmod()
	h.outer = digestmod()
	h.inner.Write(kInner)
	h.outer.Write(kOuter)

	if msg != nil {
		h.Update(msg)
	}
	return h
}

// Update mirrors update(msg).
func (h *HMAC) Update(msg []byte) {
	h.data = append(h.data, msg...)
	h.inner.Write(msg)
}

// Copy mirrors copy(), returning a fully independent copy.
func (h *HMAC) Copy() *HMAC {
	kc := make([]byte, len(h.key))
	copy(kc, h.key)

	dc := make([]byte, len(h.data))
	copy(dc, h.data)

	out := &HMAC{
		key:        kc,
		blockSize:  h.blockSize,
		digestSize: h.digestSize,
		digestmod:  h.digestmod,
		data:       dc,
	}
	if inner, err := cloneHash(h.inner, h.digestmod); err == nil {
		out.inner = inner
	}
	if outer, err := cloneHash(h.outer, h.digestmod); err == nil {
		out.outer = outer
	}
	if out.inner == nil || out.outer == nil {
		out.rebuildState()
	}
	return out
}

func cloneHash(src hash.Hash, newHash func() hash.Hash) (hash.Hash, error) {
	cloner, ok := src.(binaryCloner)
	if !ok {
		return nil, errors.New("hash state is not cloneable")
	}
	state, err := cloner.MarshalBinary()
	if err != nil {
		return nil, err
	}
	dst := newHash()
	unmarshaler, ok := dst.(binaryCloner)
	if !ok {
		return nil, errors.New("hash state is not restoreable")
	}
	if err := unmarshaler.UnmarshalBinary(state); err != nil {
		return nil, err
	}
	return dst, nil
}

func (h *HMAC) rebuildState() {
	kInner := make([]byte, len(h.key))
	kOuter := make([]byte, len(h.key))
	for i, b := range h.key {
		kInner[i] = trans36[b]
		kOuter[i] = trans5C[b]
	}
	h.inner = h.digestmod()
	h.outer = h.digestmod()
	h.inner.Write(kInner)
	h.outer.Write(kOuter)
	h.inner.Write(h.data)
}

// compute() is the internal HMAC computation, like digest() in the Python version.
func (h *HMAC) compute() []byte {
	if h.inner == nil || h.outer == nil {
		h.rebuildState()
	}
	inner, err := cloneHash(h.inner, h.digestmod)
	if err != nil {
		h.rebuildState()
		inner, _ = cloneHash(h.inner, h.digestmod)
	}
	innerSum := inner.Sum(nil)
	outer, err := cloneHash(h.outer, h.digestmod)
	if err != nil {
		h.rebuildState()
		outer, _ = cloneHash(h.outer, h.digestmod)
	}
	outer.Write(innerSum)
	return outer.Sum(nil)
}

// Digest mirrors digest().
func (h *HMAC) Digest() []byte {
	sum := h.compute()
	out := make([]byte, len(sum))
	copy(out, sum)
	return out
}

// HexDigest mirrors hexdigest().
func (h *HMAC) HexDigest() string {
	sum := h.Digest()
	const hex = "0123456789abcdef"
	out := make([]byte, len(sum)*2)
	for i, b := range sum {
		out[i*2] = hex[b>>4]
		out[i*2+1] = hex[b&0x0f]
	}
	return string(out)
}

// New is an exact analogue of Python's new(...).
func New(key, msg []byte, digestmod func() hash.Hash) *HMAC {
	return NewHMAC(key, msg, digestmod)
}

// DigestFast mirrors digest(key, msg, digest).
func DigestFast(key, msg []byte, digestmod func() hash.Hash) []byte {
	h := NewHMAC(key, msg, digestmod)
	return h.Digest()
}
