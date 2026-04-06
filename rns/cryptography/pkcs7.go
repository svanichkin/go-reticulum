package cryptography

import (
	"fmt"
)

const PKCS7BlockSize = 16

type PKCS7ValueError struct {
	Message string
}

func (e *PKCS7ValueError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// PKCS7Pad mirrors PKCS7.pad(data, bs=BLOCKSIZE).
func PKCS7Pad(data []byte, bs int) ([]byte, error) {
	if bs <= 0 {
		bs = PKCS7BlockSize
	}
	if bs >= 256 {
		return nil, &PKCS7ValueError{Message: fmt.Sprintf("invalid block size %d", bs)}
	}
	padLen := bs - (len(data) % bs)
	if padLen == 0 {
		padLen = bs
	}
	out := make([]byte, len(data)+padLen)
	copy(out, data)
	for i := len(data); i < len(out); i++ {
		out[i] = byte(padLen)
	}
	return out, nil
}

// PKCS7Unpad mirrors PKCS7.unpad(data, bs=BLOCKSIZE).
func PKCS7Unpad(data []byte, bs int) ([]byte, error) {
	if bs <= 0 {
		bs = PKCS7BlockSize
	}
	if bs <= 0 || bs >= 256 {
		return nil, &PKCS7ValueError{Message: fmt.Sprintf("invalid block size %d", bs)}
	}
	if len(data) == 0 {
		return nil, &PKCS7ValueError{Message: "cannot unpad empty data"}
	}
	padLen := int(data[len(data)-1])
	if padLen > bs {
		return nil, &PKCS7ValueError{Message: fmt.Sprintf("cannot unpad, invalid padding length of %d bytes", padLen)}
	}
	if padLen < 0 || padLen > len(data) {
		return nil, &PKCS7ValueError{Message: fmt.Sprintf("cannot unpad, invalid padding length of %d bytes", padLen)}
	}
	return data[:len(data)-padLen], nil
}
