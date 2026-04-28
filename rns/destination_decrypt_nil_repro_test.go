package rns

import (
	"path/filepath"
	"testing"
)

func TestDestinationReceive_NilDecryptWithNilErrorDoesNotPanic(t *testing.T) {
	id, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}

	dst, err := NewDestination(id, DestinationIN, DestinationSINGLE, "repro", "delivery")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}

	ratchetPath := filepath.Join(t.TempDir(), "ratchets.msgpack")
	if ok, err := dst.EnableRatchets(ratchetPath); err != nil {
		t.Fatalf("EnableRatchets: ok=%v err=%v", ok, err)
	}

	// Force the ratchet-enabled branch in Destination.Decrypt(). The ciphertext
	// is intentionally too short, which makes the identity decryption path
	// return (nil, nil, nil). The regression guard here is that Receive() must
	// fail closed instead of panicking on the retry path.
	dst.Ratchets = [][]byte{[]byte("placeholder-ratchet")}

	pkt := &Packet{
		PacketType: PacketDATA,
		Data:       []byte{0x01, 0x02, 0x03},
	}

	if ok := dst.Receive(pkt); ok {
		t.Fatalf("Receive() = true, want false for invalid ciphertext")
	}
}
