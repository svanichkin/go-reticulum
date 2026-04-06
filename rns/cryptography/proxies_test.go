package cryptography

import (
	"bytes"
	"testing"
)

func TestProxies_Ed25519_RoundTrip(t *testing.T) {
	maybeParallel(t)

	priv, err := (Ed25519PrivateKeyProxy{}).Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	pub := priv.PublicKey()

	msg := []byte("hello")
	sig := priv.Sign(msg)
	if err := pub.Verify(sig, msg); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if err := pub.Verify(sig, []byte("tampered")); err == nil {
		t.Fatalf("expected verify error")
	}
}

func TestProxies_Ed25519_PrivateBytesUseSeedFormat(t *testing.T) {
	maybeParallel(t)

	seed := bytes.Repeat([]byte{0x42}, 32)
	priv, err := (Ed25519PrivateKeyProxy{}).FromPrivateBytes(seed)
	if err != nil {
		t.Fatalf("FromPrivateBytes: %v", err)
	}
	if !bytes.Equal(priv.PrivateBytes(), seed) {
		t.Fatalf("PrivateBytes did not return original seed")
	}
}

func TestProxies_X25519_ExchangeSymmetry(t *testing.T) {
	maybeParallel(t)

	a, err := (X25519PrivateKeyProxy{}).Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	b, err := (X25519PrivateKeyProxy{}).Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	ap := a.PublicKey()
	bp := b.PublicKey()

	s1, err := a.Exchange(bp)
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	s2, err := b.Exchange(ap)
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if !bytes.Equal(s1, s2) {
		t.Fatalf("shared secret mismatch")
	}
}

func TestProxies_LengthErrors(t *testing.T) {
	maybeParallel(t)

	if _, err := (Ed25519PrivateKeyProxy{}).FromPrivateBytes([]byte("short")); err == nil {
		t.Fatalf("expected ed25519 priv length error")
	}
	if _, err := (Ed25519PublicKeyProxy{}).FromPublicBytes([]byte("short")); err == nil {
		t.Fatalf("expected ed25519 pub length error")
	}
	if _, err := (X25519PrivateKeyProxy{}).FromPrivateBytes([]byte("short")); err == nil {
		t.Fatalf("expected x25519 priv length error")
	}
	if _, err := (X25519PublicKeyProxy{}).FromPublicBytes([]byte("short")); err == nil {
		t.Fatalf("expected x25519 pub length error")
	}
}
