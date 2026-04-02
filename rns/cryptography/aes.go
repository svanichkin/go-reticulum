package cryptography

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
)

const BlockSize = aes.BlockSize // 16

type AESValueError struct {
	Message string
}

func (e *AESValueError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func aesKeyLengthError(mode string, got int) error {
	return &AESValueError{Message: fmt.Sprintf("invalid key length %d for %s", got*8, mode)}
}

func aesIVLengthError(got int) error {
	return &AESValueError{Message: fmt.Sprintf("invalid IV length: %d", got)}
}

func aesBlockLengthError() error {
	return &AESValueError{Message: "data length must be a multiple of AES block size (16)"}
}

// AES128CBCEncrypt/AES128CBCDecrypt are equivalent to Python AES_128_CBC.
func AES128CBCEncrypt(plaintext, key, iv []byte) ([]byte, error) {
	if len(key) != 16 {
		return nil, aesKeyLengthError("AES_128_CBC", len(key))
	}
	if len(iv) != BlockSize {
		return nil, aesIVLengthError(len(iv))
	}
	if len(plaintext)%BlockSize != 0 {
		return nil, aesBlockLengthError()
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, len(plaintext))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, plaintext)

	return ciphertext, nil
}

func AES128CBCDecrypt(ciphertext, key, iv []byte) ([]byte, error) {
	if len(key) != 16 {
		return nil, aesKeyLengthError("AES_128_CBC", len(key))
	}
	if len(iv) != BlockSize {
		return nil, aesIVLengthError(len(iv))
	}
	if len(ciphertext)%BlockSize != 0 {
		return nil, aesBlockLengthError()
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	plaintext := make([]byte, len(ciphertext))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(plaintext, ciphertext)

	return plaintext, nil
}

// AES256CBCEncrypt/AES256CBCDecrypt are equivalent to Python AES_256_CBC.
func AES256CBCEncrypt(plaintext, key, iv []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, aesKeyLengthError("AES_256_CBC", len(key))
	}
	if len(iv) != BlockSize {
		return nil, aesIVLengthError(len(iv))
	}
	if len(plaintext)%BlockSize != 0 {
		return nil, aesBlockLengthError()
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, len(plaintext))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, plaintext)

	return ciphertext, nil
}

func AES256CBCDecrypt(ciphertext, key, iv []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, aesKeyLengthError("AES_256_CBC", len(key))
	}
	if len(iv) != BlockSize {
		return nil, aesIVLengthError(len(iv))
	}
	if len(ciphertext)%BlockSize != 0 {
		return nil, aesBlockLengthError()
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	plaintext := make([]byte, len(ciphertext))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(plaintext, ciphertext)

	return plaintext, nil
}
