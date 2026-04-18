package rns

import (
	"encoding/hex"
	"testing"
)

func TestLink_ValidateAnnounce_Valid(t *testing.T) {
	// Uses global identity storage; don't run in parallel with other tests that might
	// manipulate persistence state.
	resetKnownDestinationsForTest()

	// Mirrors `tests/link.py::test_00_valid_announce`
	prvHex := "b82c7a4f047561d974de7e38538281d7f005d3663615f30d9663bad35a716063c931672cd452175d55bcdd70bb7aa35a9706872a97963dc52029938ea7341b39"
	prv, _ := hex.DecodeString(prvHex)
	id, err := IdentityFromBytes(prv)
	if err != nil {
		t.Fatalf("IdentityFromBytes: %v", err)
	}
	dst, err := NewDestination(id, DestinationIN, DestinationSINGLE, "test", "announce")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}
	ap := dst.Announce(nil, false, nil, nil, false)
	if ap == nil {
		t.Fatalf("Announce returned nil")
	}
	if err := ap.Pack(); err != nil {
		t.Fatalf("Pack: %v", err)
	}
	if ok := ValidateAnnounce(ap, false); !ok {
		if sigOK := ValidateAnnounce(ap, true); !sigOK {
			t.Fatalf("expected valid announce (signature validation failed)")
		}
		t.Fatalf("expected valid announce (destination/hash validation failed)")
	}
}

func TestLink_ValidateAnnounce_InvalidDestination(t *testing.T) {
	resetKnownDestinationsForTest()

	// Mirrors `tests/link.py::test_01_invalid_announce`
	prvHex := "08bb35f92b06a0832991165a0d9b4fd91af7b7765ce4572aa6222070b11b767092b61b0fd18b3a59cae6deb9db6d4bfb1c7fcfe076cfd66eea7ddd5f877543b9"
	prv, _ := hex.DecodeString(prvHex)
	id, err := IdentityFromBytes(prv)
	if err != nil {
		t.Fatalf("IdentityFromBytes: %v", err)
	}
	dst, err := NewDestination(id, DestinationIN, DestinationSINGLE, "test", "announce")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}
	ap := dst.Announce(nil, false, nil, nil, false)
	if ap == nil {
		t.Fatalf("Announce returned nil")
	}

	// Mutate the announce payload (destination hash field inside the signed material)
	// similarly to Python: fake_dst + ap.data[16:]
	fakeDst, _ := hex.DecodeString("1333b911fa8ebb16726996adbe3c6262")
	if len(ap.Data) < 16 {
		t.Fatalf("announce data too short")
	}
	preLen := len(ap.Data)
	ap.Data = append(append([]byte{}, fakeDst...), ap.Data[16:]...)
	if len(ap.Data) != preLen {
		t.Fatalf("announce data length changed")
	}

	if err := ap.Pack(); err != nil {
		t.Fatalf("Pack: %v", err)
	}
	if ok := ValidateAnnounce(ap, false); ok {
		t.Fatalf("expected invalid announce")
	}
}

func resetKnownDestinationsForTest() {
	knownDestinationsLoadMu.Lock()
	defer knownDestinationsLoadMu.Unlock()

	knownDestinations.entries = make(map[string]*knownDestinationEntry)
	knownDestinationsLoaded.Store(true)
	knownDestinationsLoadAttempted.Store(true)
}
