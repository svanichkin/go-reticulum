package cryptography

import (
	"bytes"
	"testing"
)

func TestCryptoFacade_HashAliases(t *testing.T) {
	maybeParallel(t)

	data := []byte("reticulum")
	if !bytes.Equal(HashSHA256(data), Sha256(data)) {
		t.Fatalf("HashSHA256 alias mismatch")
	}
	if !bytes.Equal(HashSHA512(data), Sha512(data)) {
		t.Fatalf("HashSHA512 alias mismatch")
	}
}

func TestCryptoFacade_HKDFWrapperParity(t *testing.T) {
	maybeParallel(t)

	salt := []byte("salt")
	ikm := []byte("ikm")
	info := []byte("context")

	got, err := HKDFSHA256(salt, ikm, info, 42)
	if err != nil {
		t.Fatalf("HKDFSHA256: %v", err)
	}
	want, err := HKDF(42, ikm, salt, info)
	if err != nil {
		t.Fatalf("HKDF: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("HKDF facade mismatch")
	}

	got512, err := HKDFSHA512(nil, ikm, nil, 42)
	if err != nil {
		t.Fatalf("HKDFSHA512: %v", err)
	}
	if len(got512) != 42 {
		t.Fatalf("HKDFSHA512 len=%d, want 42", len(got512))
	}
}

func TestCryptoFacade_TokenWrappers(t *testing.T) {
	maybeParallel(t)

	key, err := GenerateTokenKey(32)
	if err != nil {
		t.Fatalf("GenerateTokenKey: %v", err)
	}
	if len(key) != 64 {
		t.Fatalf("GenerateTokenKey len=%d, want 64", len(key))
	}
	token, err := NewTokenCipher(key)
	if err != nil {
		t.Fatalf("NewTokenCipher: %v", err)
	}
	if token == nil {
		t.Fatalf("NewTokenCipher returned nil token")
	}
}

func TestCryptoFacade_ProviderBackend(t *testing.T) {
	maybeParallel(t)

	if got := ProviderBackend(); got != "internal" {
		t.Fatalf("ProviderBackend=%q, want %q", got, "internal")
	}
}

func TestCryptoFacade_X25519Wrappers(t *testing.T) {
	maybeParallel(t)

	priv, err := X25519Generate()
	if err != nil {
		t.Fatalf("X25519Generate: %v", err)
	}
	pub, err := priv.PublicKey()
	if err != nil {
		t.Fatalf("PublicKey: %v", err)
	}
	priv2, err := X25519FromPrivateBytes(priv.PrivateBytes())
	if err != nil {
		t.Fatalf("X25519FromPrivateBytes: %v", err)
	}
	pub2, err := X25519FromPublicBytes(pub.PublicBytes())
	if err != nil {
		t.Fatalf("X25519FromPublicBytes: %v", err)
	}
	shared1, err := priv.Exchange(pub2)
	if err != nil {
		t.Fatalf("Exchange 1: %v", err)
	}
	shared2, err := priv2.Exchange(pub)
	if err != nil {
		t.Fatalf("Exchange 2: %v", err)
	}
	if !bytes.Equal(shared1, shared2) {
		t.Fatalf("shared key mismatch")
	}
}

func TestCryptoFacade_Ed25519Wrappers(t *testing.T) {
	maybeParallel(t)

	priv, err := Ed25519Generate()
	if err != nil {
		t.Fatalf("Ed25519Generate: %v", err)
	}
	pub := priv.PublicKey()
	pub2, err := Ed25519FromPublicBytes(pub.PublicBytes())
	if err != nil {
		t.Fatalf("Ed25519FromPublicBytes: %v", err)
	}
	sig := priv.Sign([]byte("msg"))
	if err := pub2.Verify(sig, []byte("msg")); err != nil {
		t.Fatalf("Verify: %v", err)
	}

	seed := priv.PrivateBytes()
	priv2, err := Ed25519FromSeed(seed)
	if err != nil {
		t.Fatalf("Ed25519FromSeed: %v", err)
	}
	if !bytes.Equal(priv2.PrivateBytes(), seed) {
		t.Fatalf("seed roundtrip mismatch")
	}
}
