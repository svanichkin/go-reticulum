package cryptography

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestX25519_ExchangeSymmetry(t *testing.T) {
	maybeParallel(t)

	a, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	b, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	ap, err := a.PublicKey()
	if err != nil {
		t.Fatalf("PublicKey: %v", err)
	}
	bp, err := b.PublicKey()
	if err != nil {
		t.Fatalf("PublicKey: %v", err)
	}

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

func TestX25519_ClampFromPrivateBytes(t *testing.T) {
	maybeParallel(t)

	raw := bytes.Repeat([]byte{0xFF}, 32)
	k, err := FromPrivateBytes(raw)
	if err != nil {
		t.Fatalf("FromPrivateBytes: %v", err)
	}
	p := k.PrivateBytes()
	if len(p) != 32 {
		t.Fatalf("unexpected private length %d", len(p))
	}
	// clampSecret invariants
	if p[0]&7 != 0 {
		t.Fatalf("expected low bits cleared")
	}
	if p[31]&0x80 != 0 {
		t.Fatalf("expected high bit cleared")
	}
	if p[31]&0x40 == 0 {
		t.Fatalf("expected bit 6 set")
	}
}

func TestX25519_Errors(t *testing.T) {
	maybeParallel(t)

	if _, err := FromPrivateBytes([]byte("short")); err == nil {
		t.Fatalf("expected private length error")
	}
	if _, err := FromPublicBytes([]byte("short")); err == nil {
		t.Fatalf("expected public length error")
	}
	k, _ := Generate()
	if _, err := k.Exchange(123); err == nil {
		t.Fatalf("expected unsupported peer type error")
	}
}

func TestX25519_KnownVector_RFC7748(t *testing.T) {
	maybeParallel(t)

	alicePriv, _ := hex.DecodeString("77076d0a7318a57d3c16c17251b26645df4c2f87ebc0992ab177fba51db92c2a")
	alicePubWant, _ := hex.DecodeString("8520f0098930a754748b7ddcb43ef75a0dbf3a0d26381af4eba4a98eaa9b4e6a")
	bobPriv, _ := hex.DecodeString("5dab087e624a8a4b79e17f8b83800ee66f3bb1292618b6fd1c2f8b27ff88e0eb")
	bobPubWant, _ := hex.DecodeString("de9edb7d7b7dc1b4d35b61c2ece435373f8343c85b78674dadfc7e146f882b4f")
	sharedWant, _ := hex.DecodeString("4a5d9d5ba4ce2de1728e3bf480350f25e07e21c947d19e3376f09b3c1e161742")

	a, err := FromPrivateBytes(alicePriv)
	if err != nil {
		t.Fatalf("FromPrivateBytes(alice): %v", err)
	}
	ap, err := a.PublicKey()
	if err != nil {
		t.Fatalf("PublicKey(alice): %v", err)
	}
	if !bytes.Equal(ap.PublicBytes(), alicePubWant) {
		t.Fatalf("alice public mismatch: got %x want %x", ap.PublicBytes(), alicePubWant)
	}

	b, err := FromPrivateBytes(bobPriv)
	if err != nil {
		t.Fatalf("FromPrivateBytes(bob): %v", err)
	}
	bp, err := b.PublicKey()
	if err != nil {
		t.Fatalf("PublicKey(bob): %v", err)
	}
	if !bytes.Equal(bp.PublicBytes(), bobPubWant) {
		t.Fatalf("bob public mismatch: got %x want %x", bp.PublicBytes(), bobPubWant)
	}

	shared, err := a.Exchange(bp.PublicBytes())
	if err != nil {
		t.Fatalf("Exchange(bytes): %v", err)
	}
	if !bytes.Equal(shared, sharedWant) {
		t.Fatalf("shared mismatch: got %x want %x", shared, sharedWant)
	}
}
