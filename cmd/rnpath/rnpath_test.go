package main

import (
	"encoding/hex"
	"testing"

	rns "github.com/svanichkin/go-reticulum/rns"
)

func TestParseDestHex_LengthAndHex(t *testing.T) {
	destLen := (rns.ReticulumTruncatedHashLength / 8) * 2

	_, err := parseDestHex("")
	if err == nil {
		t.Fatalf("expected error")
	}

	_, err = parseDestHex("aa")
	if err == nil {
		t.Fatalf("expected error for short length")
	}

	good := make([]byte, destLen/2)
	for i := range good {
		good[i] = 0xaa
	}
	hexGood := hex.EncodeToString(good)

	out, err := parseDestHex(hexGood)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hex.EncodeToString(out) != hexGood {
		t.Fatalf("got %x want %s", out, hexGood)
	}

	_, err = parseDestHex(hexGood[:destLen-1] + "z")
	if err == nil {
		t.Fatalf("expected error for invalid hex")
	}
}

func TestParseOptionalDest(t *testing.T) {
	b, err := parseOptionalDest("  ")
	if err != nil || b != nil {
		t.Fatalf("got (%v,%v)", b, err)
	}
}

func TestRemoteHashFromNameAndIdentity(t *testing.T) {
	// Mirrors rnpath.py use of Destination.hash_from_name_and_identity("rnstransport.remote.management", identity_hash)
	destLen := rns.ReticulumTruncatedHashLength / 8
	identityHash := make([]byte, destLen)
	for i := range identityHash {
		identityHash[i] = byte(i + 1)
	}
	h := destinationHashFromNameAndIdentityHash("rnstransport.remote.management", identityHash)
	if len(h) != destLen {
		t.Fatalf("got hash len %d want %d", len(h), destLen)
	}
	if hex.EncodeToString(h) == hex.EncodeToString(identityHash) {
		t.Fatalf("expected derived destination hash, not identity hash")
	}
}
