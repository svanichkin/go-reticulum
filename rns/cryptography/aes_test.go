package cryptography

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"testing"
)

func TestAES128CBC_RoundTrip(t *testing.T) {
	maybeParallel(t)

	key := bytes.Repeat([]byte{0x11}, 16)
	iv := bytes.Repeat([]byte{0x22}, 16)
	plain := bytes.Repeat([]byte{0x33}, 32)

	ciphertext, err := AES128CBCEncrypt(plain, key, iv)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	// Cross-check with stdlib.
	block, _ := aes.NewCipher(key)
	want := make([]byte, len(plain))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(want, plain)
	if !bytes.Equal(ciphertext, want) {
		t.Fatalf("ciphertext mismatch")
	}

	out, err := AES128CBCDecrypt(ciphertext, key, iv)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(out, plain) {
		t.Fatalf("roundtrip mismatch")
	}
}

func TestAES256CBC_RoundTrip(t *testing.T) {
	maybeParallel(t)

	key := bytes.Repeat([]byte{0x11}, 32)
	iv := bytes.Repeat([]byte{0x22}, 16)
	plain := bytes.Repeat([]byte{0x33}, 48)

	ciphertext, err := AES256CBCEncrypt(plain, key, iv)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	block, _ := aes.NewCipher(key)
	want := make([]byte, len(plain))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(want, plain)
	if !bytes.Equal(ciphertext, want) {
		t.Fatalf("ciphertext mismatch")
	}

	out, err := AES256CBCDecrypt(ciphertext, key, iv)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(out, plain) {
		t.Fatalf("roundtrip mismatch")
	}
}

func TestAES_Errors(t *testing.T) {
	maybeParallel(t)

	key16 := bytes.Repeat([]byte{0x11}, 16)
	key32 := bytes.Repeat([]byte{0x11}, 32)
	iv := bytes.Repeat([]byte{0x22}, 16)
	badIV := bytes.Repeat([]byte{0x22}, 15)
	plainBad := []byte{1, 2, 3}

	if _, err := AES128CBCEncrypt(bytes.Repeat([]byte{0x33}, 16), key16[:15], iv); err == nil {
		t.Fatalf("expected error for AES-128 key length")
	} else {
		var valueErr *AESValueError
		if !errors.As(err, &valueErr) || valueErr.Error() != "invalid key length 120 for AES_128_CBC" {
			t.Fatalf("unexpected AES-128 key error: %v", err)
		}
	}
	if _, err := AES128CBCEncrypt(bytes.Repeat([]byte{0x33}, 16), key16, badIV); err == nil {
		t.Fatalf("expected error for IV length")
	} else {
		var valueErr *AESValueError
		if !errors.As(err, &valueErr) || valueErr.Error() != "invalid IV length: 15" {
			t.Fatalf("unexpected IV error: %v", err)
		}
	}
	if _, err := AES128CBCEncrypt(plainBad, key16, iv); err == nil {
		t.Fatalf("expected error for non-block-multiple plaintext")
	} else {
		var valueErr *AESValueError
		if !errors.As(err, &valueErr) || valueErr.Error() != "data length must be a multiple of AES block size (16)" {
			t.Fatalf("unexpected plaintext size error: %v", err)
		}
	}

	if _, err := AES256CBCEncrypt(bytes.Repeat([]byte{0x33}, 16), key32[:31], iv); err == nil {
		t.Fatalf("expected error for AES-256 key length")
	} else {
		var valueErr *AESValueError
		if !errors.As(err, &valueErr) || valueErr.Error() != "invalid key length 248 for AES_256_CBC" {
			t.Fatalf("unexpected AES-256 key error: %v", err)
		}
	}
	if _, err := AES256CBCEncrypt(bytes.Repeat([]byte{0x33}, 16), key32, badIV); err == nil {
		t.Fatalf("expected error for IV length")
	} else {
		var valueErr *AESValueError
		if !errors.As(err, &valueErr) || valueErr.Error() != "invalid IV length: 15" {
			t.Fatalf("unexpected AES-256 IV error: %v", err)
		}
	}
	if _, err := AES256CBCEncrypt(plainBad, key32, iv); err == nil {
		t.Fatalf("expected error for non-block-multiple plaintext")
	} else {
		var valueErr *AESValueError
		if !errors.As(err, &valueErr) || valueErr.Error() != "data length must be a multiple of AES block size (16)" {
			t.Fatalf("unexpected AES-256 plaintext size error: %v", err)
		}
	}
}
