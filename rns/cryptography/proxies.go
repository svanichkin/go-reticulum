package cryptography

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
)

// ===================== X25519 =====================

type X25519PrivateKeyProxy struct {
	k *ecdh.PrivateKey
}

type X25519PublicKeyProxy struct {
	k *ecdh.PublicKey
}

var (
	x25519Curve = ecdh.X25519()

	// ErrInvalidX25519PrivLen = errors.New("x25519: invalid private key length")
	// ErrInvalidX25519PubLen  = errors.New("x25519: invalid public key length")
)

// Generate mirrors X25519PrivateKeyProxy.generate().
func (X25519PrivateKeyProxy) Generate() (*X25519PrivateKeyProxy, error) {
	k, err := x25519Curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &X25519PrivateKeyProxy{k: k}, nil
}

// FromPrivateBytes mirrors from_private_bytes().
func (X25519PrivateKeyProxy) FromPrivateBytes(b []byte) (*X25519PrivateKeyProxy, error) {
	k, err := x25519Curve.NewPrivateKey(b)
	if err != nil {
		return nil, err
	}
	return &X25519PrivateKeyProxy{k: k}, nil
}

// PrivateBytes mirrors private_bytes().
func (p *X25519PrivateKeyProxy) PrivateBytes() []byte {
	return p.k.Bytes()
}

// PublicKey mirrors public_key().
func (p *X25519PrivateKeyProxy) PublicKey() *X25519PublicKeyProxy {
	return &X25519PublicKeyProxy{k: p.k.PublicKey()}
}

// Exchange mirrors exchange().
func (p *X25519PrivateKeyProxy) Exchange(peer *X25519PublicKeyProxy) ([]byte, error) {
	return p.k.ECDH(peer.k)
}

// FromPublicBytes mirrors from_public_bytes().
func (X25519PublicKeyProxy) FromPublicBytes(b []byte) (*X25519PublicKeyProxy, error) {
	k, err := x25519Curve.NewPublicKey(b)
	if err != nil {
		return nil, err
	}
	return &X25519PublicKeyProxy{k: k}, nil
}

// PublicBytes mirrors public_bytes().
func (p *X25519PublicKeyProxy) PublicBytes() []byte {
	return p.k.Bytes()
}

// ===================== Ed25519 =====================

type Ed25519PrivateKeyProxy struct {
	k ed25519.PrivateKey
}

type Ed25519PublicKeyProxy struct {
	k ed25519.PublicKey
}

var (
	ErrInvalidEd25519PrivLen = errors.New("ed25519: invalid private key length")
	ErrInvalidEd25519PubLen  = errors.New("ed25519: invalid public key length")
)

// Generate mirrors Ed25519PrivateKeyProxy.generate().
func (Ed25519PrivateKeyProxy) Generate() (*Ed25519PrivateKeyProxy, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &Ed25519PrivateKeyProxy{k: priv}, nil
}

// FromPrivateBytes mirrors from_private_bytes().
func (Ed25519PrivateKeyProxy) FromPrivateBytes(b []byte) (*Ed25519PrivateKeyProxy, error) {
	if len(b) != ed25519.SeedSize {
		return nil, ErrInvalidEd25519PrivLen
	}
	priv := ed25519.NewKeyFromSeed(b)
	return &Ed25519PrivateKeyProxy{k: priv}, nil
}

// PrivateBytes mirrors private_bytes().
func (p *Ed25519PrivateKeyProxy) PrivateBytes() []byte {
	seed := p.k.Seed()
	out := make([]byte, len(seed))
	copy(out, seed)
	return out
}

// PublicKey mirrors public_key().
func (p *Ed25519PrivateKeyProxy) PublicKey() *Ed25519PublicKeyProxy {
	pub := p.k.Public().(ed25519.PublicKey)
	return &Ed25519PublicKeyProxy{k: pub}
}

// Sign mirrors sign().
func (p *Ed25519PrivateKeyProxy) Sign(message []byte) []byte {
	return ed25519.Sign(p.k, message)
}

// FromPublicBytes mirrors from_public_bytes().
func (Ed25519PublicKeyProxy) FromPublicBytes(b []byte) (*Ed25519PublicKeyProxy, error) {
	if len(b) != ed25519.PublicKeySize {
		return nil, ErrInvalidEd25519PubLen
	}
	pub := ed25519.PublicKey(make([]byte, len(b)))
	copy(pub, b)
	return &Ed25519PublicKeyProxy{k: pub}, nil
}

// PublicBytes mirrors public_bytes().
func (p *Ed25519PublicKeyProxy) PublicBytes() []byte {
	out := make([]byte, len(p.k))
	copy(out, p.k)
	return out
}

// Verify mirrors verify(); returns an error instead of raising.
func (p *Ed25519PublicKeyProxy) Verify(signature, message []byte) error {
	if !ed25519.Verify(p.k, message, signature) {
		return errors.New("ed25519: invalid signature")
	}
	return nil
}
