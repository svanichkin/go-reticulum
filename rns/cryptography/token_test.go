package cryptography

import (
	"bytes"
	crand "crypto/rand"
	"errors"
	"testing"
)

func TestToken_EncryptDecrypt_RoundTrip(t *testing.T) {
	maybeParallel(t)

	key, err := GenerateKey(32)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	tok, err := NewToken(key)
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}

	plain := []byte("hello token")
	enc, err := tok.Encrypt(plain)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	out, err := tok.Decrypt(enc)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(out, plain) {
		t.Fatalf("roundtrip mismatch")
	}
}

func TestToken_TamperDetection(t *testing.T) {
	maybeParallel(t)

	key := bytes.Repeat([]byte{0x42}, 64)
	tok, err := NewToken(key)
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}

	enc, err := tok.Encrypt([]byte("hello"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	flip := func(i int) []byte {
		cp := append([]byte(nil), enc...)
		cp[i] ^= 0x01
		return cp
	}

	// Flip IV byte.
	if _, err := tok.Decrypt(flip(0)); err == nil {
		t.Fatalf("expected HMAC error on IV tamper")
	} else {
		var valueErr *TokenValueError
		if !errors.As(err, &valueErr) || valueErr.Error() != "Token HMAC was invalid" {
			t.Fatalf("unexpected IV tamper error: %v", err)
		}
	}
	// Flip ciphertext byte.
	if _, err := tok.Decrypt(flip(blockSize)); err == nil {
		t.Fatalf("expected HMAC error on ciphertext tamper")
	} else {
		var valueErr *TokenValueError
		if !errors.As(err, &valueErr) || valueErr.Error() != "Token HMAC was invalid" {
			t.Fatalf("unexpected ciphertext tamper error: %v", err)
		}
	}
	// Flip HMAC byte.
	if _, err := tok.Decrypt(flip(len(enc) - 1)); err == nil {
		t.Fatalf("expected HMAC error on signature tamper")
	} else {
		var valueErr *TokenValueError
		if !errors.As(err, &valueErr) || valueErr.Error() != "Token HMAC was invalid" {
			t.Fatalf("unexpected signature tamper error: %v", err)
		}
	}
}

func TestToken_Errors(t *testing.T) {
	maybeParallel(t)

	if _, err := NewToken(nil); err == nil {
		t.Fatalf("expected error for nil key")
	} else {
		var valueErr *TokenValueError
		if !errors.As(err, &valueErr) || valueErr.Error() != "Token key cannot be None" {
			t.Fatalf("unexpected nil key error: %v", err)
		}
	}
	if _, err := NewToken(bytes.Repeat([]byte{0x00}, 10)); err == nil {
		t.Fatalf("expected error for wrong key length")
	} else {
		var valueErr *TokenValueError
		if !errors.As(err, &valueErr) || valueErr.Error() != "Token key must be 128 or 256 bits, not 80" {
			t.Fatalf("unexpected wrong key length error: %v", err)
		}
	}

	key := bytes.Repeat([]byte{0x11}, 64)
	tok, _ := NewToken(key)

	if _, err := tok.Encrypt(nil); err == nil {
		t.Fatalf("expected error for nil plaintext")
	} else {
		var typeErr *TokenTypeError
		if !errors.As(err, &typeErr) || typeErr.Error() != "Token plaintext input must be bytes" {
			t.Fatalf("unexpected nil plaintext error: %v", err)
		}
	}
	if _, err := tok.Decrypt(nil); err == nil {
		t.Fatalf("expected error for nil token")
	} else {
		var typeErr *TokenTypeError
		if !errors.As(err, &typeErr) || typeErr.Error() != "Token must be bytes" {
			t.Fatalf("unexpected nil token error: %v", err)
		}
	}
	if _, err := tok.Decrypt([]byte{1, 2, 3}); err == nil {
		t.Fatalf("expected error for too-short token")
	} else {
		var valueErr *TokenValueError
		if !errors.As(err, &valueErr) || valueErr.Error() != "Cannot verify HMAC on token of only 3 bytes" {
			t.Fatalf("unexpected too-short token error: %v", err)
		}
	}
}

func TestToken_Decrypt_CiphertextLengthCheckAfterValidHMAC(t *testing.T) {
	// Do not run in parallel: modifies crypto/rand.Reader.

	// Build a token with a deterministic IV to keep the test stable.
	orig := crand.Reader
	defer func() { crand.Reader = orig }()
	crand.Reader = bytes.NewReader(bytes.Repeat([]byte{0xAA}, 64))

	key := bytes.Repeat([]byte{0x22}, 64)
	tok, err := NewToken(key)
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}

	// Create a valid token, then truncate ciphertext length by 1 and recompute HMAC.
	enc, err := tok.Encrypt([]byte("hello"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	iv := enc[:blockSize]
	ciphertext := enc[blockSize : len(enc)-hmacSize]
	if len(ciphertext) < blockSize {
		t.Fatalf("unexpected ciphertext length %d", len(ciphertext))
	}
	ciphertext = ciphertext[:len(ciphertext)-1] // break block alignment

	// Recompute valid HMAC for malformed body.
	body := append(append([]byte(nil), iv...), ciphertext...)
	mac := tok.signingKey // tests are in-package, ok to access
	bad := append(append([]byte(nil), body...), computeHMACSHA256(mac, body)...)

	if _, err := tok.Decrypt(bad); err == nil {
		t.Fatalf("expected error for non-block-multiple ciphertext")
	} else {
		var valueErr *TokenValueError
		if !errors.As(err, &valueErr) || valueErr.Error() != "Could not decrypt token: ciphertext length not multiple of block size" {
			t.Fatalf("unexpected ciphertext length error: %v", err)
		}
	}
}

func TestToken_FormatAndKeySplitting(t *testing.T) {
	orig := crand.Reader
	defer func() { crand.Reader = orig }()
	crand.Reader = bytes.NewReader(bytes.Repeat([]byte{0xAB}, 64))

	key := bytes.Repeat([]byte{0x11}, 64)
	tok, err := NewToken(key)
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	if !bytes.Equal(tok.signingKey, key[:32]) || !bytes.Equal(tok.encryptionKey, key[32:]) {
		t.Fatalf("unexpected key split")
	}

	enc, err := tok.Encrypt([]byte("hello"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if len(enc) < Overhead {
		t.Fatalf("token shorter than overhead: %d", len(enc))
	}
	iv := enc[:blockSize]
	if !bytes.Equal(iv, bytes.Repeat([]byte{0xAB}, blockSize)) {
		t.Fatalf("unexpected IV prefix: %x", iv)
	}
	body := enc[:len(enc)-hmacSize]
	gotHMAC := enc[len(enc)-hmacSize:]
	wantHMAC := computeHMACSHA256(tok.signingKey, body)
	if !bytes.Equal(gotHMAC, wantHMAC) {
		t.Fatalf("unexpected token hmac")
	}
}

func computeHMACSHA256(key, data []byte) []byte {
	h := NewHMAC(key, data, nil)
	sum := h.Digest()
	return sum[:hmacSize]
}
