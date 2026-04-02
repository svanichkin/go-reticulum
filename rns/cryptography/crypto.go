package cryptography

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"hash"
)

// ===== Hash helpers =====

// HashSHA256 is a convenience alias for Sha256().
func HashSHA256(data []byte) []byte {
	return Sha256(data)
}

// HashSHA512 is a convenience alias for Sha512().
func HashSHA512(data []byte) []byte {
	return Sha512(data)
}

// ===== HKDF =====

// HKDFSHA256/512 are wrappers for cases where streaming output is needed
// (eg. like Python via hashlib).
func HKDFSHA256(salt, ikm, info []byte, length int) ([]byte, error) {
	return hkdfExpand(sha256.New, salt, ikm, info, length)
}

func HKDFSHA512(salt, ikm, info []byte, length int) ([]byte, error) {
	return hkdfExpand(sha512.New, salt, ikm, info, length)
}

func hkdfExpand(newHash func() hash.Hash, salt, ikm, info []byte, length int) ([]byte, error) {
	if length <= 0 {
		return nil, errors.New("hkdf: invalid length")
	}

	hashLen := newHash().Size()
	if salt == nil {
		salt = make([]byte, hashLen)
	}

	prk := hkdfExtract(newHash, salt, ikm)

	var (
		t       []byte
		derived []byte
	)
	nBlocks := (length + hashLen - 1) / hashLen
	if nBlocks > 255 {
		return nil, errors.New("hkdf: length too large")
	}

	for i := 1; i <= nBlocks; i++ {
		mac := hmac.New(newHash, prk)
		mac.Write(t)
		mac.Write(info)
		mac.Write([]byte{byte(i)})
		t = mac.Sum(nil)
		derived = append(derived, t...)
	}

	return derived[:length], nil
}

func hkdfExtract(newHash func() hash.Hash, salt, ikm []byte) []byte {
	mac := hmac.New(newHash, salt)
	mac.Write(ikm)
	return mac.Sum(nil)
}

// ===== Token helpers =====

type TokenCipher = Token

func NewTokenCipher(key []byte) (*TokenCipher, error) {
	return NewToken(key)
}

func GenerateTokenKey(aesKeyBytes int) ([]byte, error) {
	return GenerateKey(aesKeyBytes)
}

// ===== Provider helpers =====

func ProviderBackend() string {
	return "internal"
}

// ===== X25519 helpers =====

type X25519Private = PrivateKey
type X25519Public = PublicKey

func X25519Generate() (*X25519Private, error) {
	return Generate()
}

func X25519FromPrivateBytes(b []byte) (*X25519Private, error) {
	return FromPrivateBytes(b)
}

func X25519FromPublicBytes(b []byte) (*X25519Public, error) {
	return FromPublicBytes(b)
}

// ===== Ed25519 helpers =====

type Ed25519Private = Ed25519PrivateKey
type Ed25519Public = Ed25519PublicKey

func Ed25519Generate() (*Ed25519Private, error) {
	return GenerateEd25519PrivateKey()
}

func Ed25519FromSeed(seed []byte) (*Ed25519Private, error) {
	return NewEd25519PrivateKey(seed)
}

func Ed25519FromPublicBytes(b []byte) (*Ed25519Public, error) {
	return NewEd25519PublicKey(b)
}
