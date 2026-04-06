package cryptography

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
)

const (
	Overhead   = 48 // 16 IV + 32 HMAC
	blockSize  = aes.BlockSize
	hmacSize   = 32
	key128Size = 32 // 16+16
	key256Size = 64 // 32+32
)

type TokenTypeError struct {
	Message string
}

func (e *TokenTypeError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

type TokenValueError struct {
	Message string
}

func (e *TokenValueError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// GenerateKey mirrors Token.generate_key(mode=AES_256_CBC by default).
// aesKeyBytes = 16 -> AES-128 (final key 32 bytes)
// aesKeyBytes = 32 -> AES-256 (final key 64 bytes)
func GenerateKey(aesKeyBytes int) ([]byte, error) {
	var total int
	switch aesKeyBytes {
	case 16:
		total = key128Size
	case 32:
		total = key256Size
	default:
		return nil, &TokenTypeError{Message: fmt.Sprintf("invalid token mode: %d", aesKeyBytes)}
	}

	k := make([]byte, total)
	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		return nil, err
	}
	return k, nil
}

type Token struct {
	signingKey    []byte
	encryptionKey []byte
	aesKeySize    int // 16 or 32
}

// NewToken mirrors __init__(key, mode=AES).
func NewToken(key []byte) (*Token, error) {
	if key == nil {
		return nil, &TokenValueError{Message: "Token key cannot be None"}
	}

	switch len(key) {
	case key128Size:
		return &Token{
			signingKey:    key[:16],
			encryptionKey: key[16:],
			aesKeySize:    16,
		}, nil
	case key256Size:
		return &Token{
			signingKey:    key[:32],
			encryptionKey: key[32:],
			aesKeySize:    32,
		}, nil
	default:
		return nil, &TokenValueError{Message: fmt.Sprintf("Token key must be 128 or 256 bits, not %d", len(key)*8)}
	}
}

// verifyHMAC mirrors verify_hmac().
func (t *Token) verifyHMAC(tok []byte) (bool, error) {
	if len(tok) <= hmacSize {
		return false, &TokenValueError{Message: fmt.Sprintf("Cannot verify HMAC on token of only %d bytes", len(tok))}
	}

	data := tok[:len(tok)-hmacSize]
	rec := tok[len(tok)-hmacSize:]

	mac := hmac.New(sha256.New, t.signingKey)
	_, _ = mac.Write(data)
	exp := mac.Sum(nil)[:hmacSize]

	return hmac.Equal(rec, exp), nil
}

// Encrypt mirrors encrypt().
func (t *Token) Encrypt(data []byte) ([]byte, error) {
	if data == nil {
		return nil, &TokenTypeError{Message: "Token plaintext input must be bytes"}
	}

	iv := make([]byte, blockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	padded, err := PKCS7Pad(data, blockSize)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(t.encryptionKey)
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, len(padded))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, padded)

	signedParts := append(iv, ciphertext...)

	mac := hmac.New(sha256.New, t.signingKey)
	_, _ = mac.Write(signedParts)
	sig := mac.Sum(nil)[:hmacSize]

	return append(signedParts, sig...), nil
}

// Decrypt mirrors decrypt().
func (t *Token) Decrypt(tok []byte) ([]byte, error) {
	if tok == nil {
		return nil, &TokenTypeError{Message: "Token must be bytes"}
	}
	valid, err := t.verifyHMAC(tok)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, &TokenValueError{Message: "Token HMAC was invalid"}
	}

	iv := tok[:blockSize]
	ciphertext := tok[blockSize : len(tok)-hmacSize]

	if len(ciphertext)%blockSize != 0 {
		return nil, &TokenValueError{Message: "Could not decrypt token: ciphertext length not multiple of block size"}
	}

	block, err := aes.NewCipher(t.encryptionKey)
	if err != nil {
		return nil, err
	}

	plaintextPadded := make([]byte, len(ciphertext))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(plaintextPadded, ciphertext)

	plaintext, err := PKCS7Unpad(plaintextPadded, blockSize)
	if err != nil {
		return nil, &TokenValueError{Message: fmt.Sprintf("Could not decrypt token: %v", err)}
	}

	return plaintext, nil
}
