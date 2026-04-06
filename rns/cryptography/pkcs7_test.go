package cryptography

import (
	"bytes"
	"errors"
	"testing"
)

func TestPKCS7PadUnpad_RoundTrip_DefaultBlockSize(t *testing.T) {
	maybeParallel(t)

	lengths := []int{0, 1, 15, 16, 17, 31, 32, 33, 63, 64}
	for _, n := range lengths {
		t.Run("len="+itoa(n), func(t *testing.T) {
			in := bytes.Repeat([]byte{0xA5}, n)
			padded, err := PKCS7Pad(in, 0)
			if err != nil {
				t.Fatalf("pad: %v", err)
			}
			if len(padded) == 0 || len(padded)%PKCS7BlockSize != 0 {
				t.Fatalf("unexpected padded length %d", len(padded))
			}

			out, err := PKCS7Unpad(padded, 0)
			if err != nil {
				t.Fatalf("unpad: %v", err)
			}
			if !bytes.Equal(out, in) {
				t.Fatalf("roundtrip mismatch")
			}
		})
	}
}

func TestPKCS7Pad_BlockSizeValidation(t *testing.T) {
	maybeParallel(t)

	if _, err := PKCS7Pad([]byte("x"), 256); err == nil {
		t.Fatalf("expected error for bs=256")
	} else {
		var valueErr *PKCS7ValueError
		if !errors.As(err, &valueErr) || valueErr.Error() != "invalid block size 256" {
			t.Fatalf("unexpected pad block size error: %v", err)
		}
	}
	if _, err := PKCS7Unpad([]byte(""), 256); err == nil {
		t.Fatalf("expected error for bs=256")
	} else {
		var valueErr *PKCS7ValueError
		if !errors.As(err, &valueErr) || valueErr.Error() != "invalid block size 256" {
			t.Fatalf("unexpected unpad block size error: %v", err)
		}
	}
}

func TestPKCS7Unpad_Errors(t *testing.T) {
	maybeParallel(t)

	if _, err := PKCS7Unpad(nil, 16); err == nil {
		t.Fatalf("expected error for nil input")
	} else {
		var valueErr *PKCS7ValueError
		if !errors.As(err, &valueErr) || valueErr.Error() != "cannot unpad empty data" {
			t.Fatalf("unexpected empty input error: %v", err)
		}
	}
	if _, err := PKCS7Unpad([]byte{}, 16); err == nil {
		t.Fatalf("expected error for empty input")
	} else {
		var valueErr *PKCS7ValueError
		if !errors.As(err, &valueErr) || valueErr.Error() != "cannot unpad empty data" {
			t.Fatalf("unexpected empty slice error: %v", err)
		}
	}

	badBig := bytes.Repeat([]byte{0}, 16)
	badBig[15] = 17
	if _, err := PKCS7Unpad(badBig, 16); err == nil {
		t.Fatalf("expected error for padLen>bs")
	} else {
		var valueErr *PKCS7ValueError
		if !errors.As(err, &valueErr) || valueErr.Error() != "cannot unpad, invalid padding length of 17 bytes" {
			t.Fatalf("unexpected padLen error: %v", err)
		}
	}
}

func TestPKCS7Unpad_PythonStyleLeniency(t *testing.T) {
	maybeParallel(t)

	badBytes := bytes.Repeat([]byte{0}, 16)
	badBytes[15] = 2
	badBytes[14] = 3
	out, err := PKCS7Unpad(badBytes, 16)
	if err != nil {
		t.Fatalf("unexpected error for python-style lenient unpad: %v", err)
	}
	if len(out) != 14 {
		t.Fatalf("unexpected len after lenient unpad: %d", len(out))
	}

	out, err = PKCS7Unpad([]byte{0x41, 0x42, 0x01}, 16)
	if err != nil {
		t.Fatalf("unexpected error for non-multiple length: %v", err)
	}
	if !bytes.Equal(out, []byte{0x41, 0x42}) {
		t.Fatalf("unexpected non-multiple unpad result: %x", out)
	}
}

func itoa(n int) string {
	// small helper to avoid fmt in tight test loops
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [32]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
