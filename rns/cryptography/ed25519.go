package cryptography

import (
	"crypto/ed25519"
	"crypto/rand"
	"io"
)

type Ed25519ValueError struct {
	Message string
}

func (e *Ed25519ValueError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

type Ed25519SignatureError struct {
	Message string
}

func (e *Ed25519SignatureError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// ------------------------------------------------------------
// Ed25519PrivateKey (equivalent of Python Ed25519PrivateKey)
// ------------------------------------------------------------

type Ed25519PrivateKey struct {
	seed []byte
	sk   ed25519.PrivateKey
}

func NewEd25519PrivateKey(seed []byte) (*Ed25519PrivateKey, error) {
	if len(seed) != ed25519.SeedSize {
		return nil, &Ed25519ValueError{Message: "seed must be 32 bytes"}
	}
	sk := ed25519.NewKeyFromSeed(seed)
	return &Ed25519PrivateKey{
		seed: seed,
		sk:   sk,
	}, nil
}

func GenerateEd25519PrivateKey() (*Ed25519PrivateKey, error) {
	seed := make([]byte, ed25519.SeedSize)
	if _, err := io.ReadFull(rand.Reader, seed); err != nil {
		return nil, err
	}
	return NewEd25519PrivateKey(seed)
}

func (k *Ed25519PrivateKey) PrivateBytes() []byte {
	return append([]byte(nil), k.seed...)
}

func (k *Ed25519PrivateKey) PublicKey() *Ed25519PublicKey {
	pub := k.sk.Public().(ed25519.PublicKey)
	cp := append([]byte(nil), pub...)
	return &Ed25519PublicKey{vk: cp}
}

func (k *Ed25519PrivateKey) Sign(message []byte) []byte {
	return ed25519.Sign(k.sk, message)
}

// ------------------------------------------------------------
// Ed25519PublicKey (equivalent of Python Ed25519PublicKey)
// ------------------------------------------------------------

type Ed25519PublicKey struct {
	vk ed25519.PublicKey
}

func NewEd25519PublicKey(raw []byte) (*Ed25519PublicKey, error) {
	if len(raw) != ed25519.PublicKeySize {
		return nil, &Ed25519ValueError{Message: "public key must be 32 bytes"}
	}
	cp := append([]byte(nil), raw...)
	return &Ed25519PublicKey{vk: cp}, nil
}

func (k *Ed25519PublicKey) PublicBytes() []byte {
	cp := append([]byte(nil), k.vk...)
	return cp
}

func (k *Ed25519PublicKey) Verify(signature, message []byte) error {
	if !ed25519.Verify(k.vk, message, signature) {
		return &Ed25519SignatureError{Message: "invalid signature"}
	}
	return nil
}
