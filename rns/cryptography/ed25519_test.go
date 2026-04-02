package cryptography

import (
	"bytes"
	"errors"
	"testing"
)

func TestEd25519_SignVerify(t *testing.T) {
	maybeParallel(t)

	seed := bytes.Repeat([]byte{0x01}, 32)
	priv, err := NewEd25519PrivateKey(seed)
	if err != nil {
		t.Fatalf("NewEd25519PrivateKey: %v", err)
	}
	pub := priv.PublicKey()

	msg := []byte("hello")
	sig := priv.Sign(msg)

	if err := pub.Verify(sig, msg); err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if err := pub.Verify(sig, []byte("hello!")); err == nil {
		t.Fatalf("expected verify failure for modified message")
	} else {
		var sigErr *Ed25519SignatureError
		if !errors.As(err, &sigErr) || sigErr.Error() != "invalid signature" {
			t.Fatalf("unexpected verify error: %v", err)
		}
	}
}

func TestEd25519_KeyLengthErrors(t *testing.T) {
	maybeParallel(t)

	if _, err := NewEd25519PrivateKey([]byte("short")); err == nil {
		t.Fatalf("expected seed length error")
	} else {
		var valueErr *Ed25519ValueError
		if !errors.As(err, &valueErr) || valueErr.Error() != "seed must be 32 bytes" {
			t.Fatalf("unexpected seed error: %v", err)
		}
	}
	if _, err := NewEd25519PublicKey([]byte("short")); err == nil {
		t.Fatalf("expected public key length error")
	} else {
		var valueErr *Ed25519ValueError
		if !errors.As(err, &valueErr) || valueErr.Error() != "public key must be 32 bytes" {
			t.Fatalf("unexpected public key error: %v", err)
		}
	}
}

func TestEd25519_PrivateBytesAreSeed(t *testing.T) {
	maybeParallel(t)

	seed := bytes.Repeat([]byte{0x42}, 32)
	priv, err := NewEd25519PrivateKey(seed)
	if err != nil {
		t.Fatalf("NewEd25519PrivateKey: %v", err)
	}
	if !bytes.Equal(priv.PrivateBytes(), seed) {
		t.Fatalf("PrivateBytes did not return original seed")
	}
}
