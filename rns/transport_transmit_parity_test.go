package rns

import (
	"bytes"
	"testing"

	Cryptography "github.com/svanichkin/go-reticulum/rns/cryptography"
)

type fixedSigner struct {
	signature []byte
}

func (f fixedSigner) Sign([]byte) ([]byte, error) {
	return append([]byte(nil), f.signature...), nil
}

func pythonTransmitReference(raw, signature, ifacKey []byte, ifacSize int) []byte {
	if len(signature) < ifacSize {
		return nil
	}

	ifac := signature[len(signature)-ifacSize:]
	mask, err := Cryptography.HKDF(len(raw)+ifacSize, ifac, ifacKey, nil)
	if err != nil {
		return nil
	}

	newHeader := []byte{raw[0] | 0x80, raw[1]}
	newRaw := append(newHeader, ifac...)
	newRaw = append(newRaw, raw[2:]...)

	masked := make([]byte, len(newRaw))
	for i, b := range newRaw {
		switch {
		case i == 0:
			masked[i] = (b ^ mask[i]) | 0x80
		case i == 1 || i > ifacSize+1:
			masked[i] = b ^ mask[i]
		default:
			masked[i] = b
		}
	}

	return masked
}

func TestProcessOutgoing_MatchesPythonTransmit_NoIFAC(t *testing.T) {
	ifc := &Interface{Type: "Test"}
	var got []byte
	ifc.SetProcessOutgoingFunc(func(data []byte) error {
		got = append([]byte(nil), data...)
		return nil
	})

	raw := []byte{0x01, 0x02, 0x03, 0x04}
	transmit(ifc, raw)

	if !bytes.Equal(got, raw) {
		t.Fatalf("no-IFAC payload mismatch: got %x want %x", got, raw)
	}
}

func TestProcessOutgoing_MatchesPythonTransmit_WithIFAC(t *testing.T) {
	ifc := &Interface{
		Type:     "Test",
		IFACSize: 4,
		IFACKey:  []byte("ifac-key"),
		IFACIdentity: fixedSigner{
			signature: []byte{
				0x10, 0x11, 0x12, 0x13,
				0x20, 0x21, 0x22, 0x23,
				0x30, 0x31, 0x32, 0x33,
				0x40, 0x41, 0x42, 0x43,
			},
		},
	}

	var got []byte
	ifc.SetProcessOutgoingFunc(func(data []byte) error {
		got = append([]byte(nil), data...)
		return nil
	})

	raw := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}
	transmit(ifc, raw)

	want := pythonTransmitReference(raw, []byte{
		0x10, 0x11, 0x12, 0x13,
		0x20, 0x21, 0x22, 0x23,
		0x30, 0x31, 0x32, 0x33,
		0x40, 0x41, 0x42, 0x43,
	}, []byte("ifac-key"), 4)

	if !bytes.Equal(got, want) {
		t.Fatalf("IFAC payload mismatch: got %x want %x", got, want)
	}
}
