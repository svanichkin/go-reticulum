package rns

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func resetKnownDestinationsForTestIdentityParity() {
	knownDestinationsLoadMu.Lock()
	defer knownDestinationsLoadMu.Unlock()
	knownDestinations.entries = make(map[string]*knownDestinationEntry)
	knownDestinationsLoaded.Store(true)
	knownDestinationsLoadAttempted.Store(true)
}

func TestIdentityValidateAnnounce_RejectsBlackholedIdentity(t *testing.T) {
	resetKnownDestinationsForTest()

	prevBlackholed := blackholedIdentities
	blackholeMu.Lock()
	blackholedIdentities = make(map[hashKey]*blackholeEntry)
	blackholeMu.Unlock()
	t.Cleanup(func() {
		blackholeMu.Lock()
		blackholedIdentities = prevBlackholed
		blackholeMu.Unlock()
	})

	id, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	dst, err := NewDestination(id, DestinationIN, DestinationSINGLE, "test", "announce")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}
	ap := dst.Announce(nil, false, nil, nil, false)
	if ap == nil {
		t.Fatal("Announce() returned nil")
	}
	if err := ap.Pack(); err != nil {
		t.Fatalf("Pack: %v", err)
	}

	key, ok := makeHashKey(id.Hash)
	if !ok {
		t.Fatalf("makeHashKey(%x) failed", id.Hash)
	}
	blackholeMu.Lock()
	blackholedIdentities[key] = &blackholeEntry{Source: []byte("test-blackhole")}
	blackholeMu.Unlock()

	if ok := IdentityValidateAnnounce(ap, false); ok {
		t.Fatal("IdentityValidateAnnounce() = true, want false for blackholed identity")
	}
}

func TestIdentityValidateAnnounce_LoadsPersistedKnownDestinationsBeforeCollisionCheck(t *testing.T) {
	dir := t.TempDir()
	prevInstance := instance
	instance = &Reticulum{StoragePath: dir}
	t.Cleanup(func() { instance = prevInstance })

	id, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	dst, err := NewDestination(id, DestinationIN, DestinationSINGLE, "test", "announce")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}
	ap := dst.Announce(nil, false, nil, nil, false)
	if ap == nil {
		t.Fatal("Announce() returned nil")
	}
	if err := ap.Pack(); err != nil {
		t.Fatalf("Pack: %v", err)
	}

	other, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(other): %v", err)
	}

	raw, err := encodeKnownDestinations(map[string]*knownDestinationEntry{
		string(ap.DestinationHash): {
			SeenAt:     1,
			PacketHash: []byte("pkt"),
			PublicKey:  other.GetPublicKey(),
		},
	})
	if err != nil {
		t.Fatalf("encodeKnownDestinations: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "known_destinations"), raw, 0o600); err != nil {
		t.Fatalf("WriteFile(known_destinations): %v", err)
	}

	knownDestinationsLoadMu.Lock()
	knownDestinations.entries = make(map[string]*knownDestinationEntry)
	knownDestinationsLoaded.Store(false)
	knownDestinationsLoadAttempted.Store(false)
	knownDestinationsLoadMu.Unlock()
	t.Cleanup(resetKnownDestinationsForTest)

	if ok := IdentityValidateAnnounce(ap, false); ok {
		t.Fatal("IdentityValidateAnnounce() = true, want false for persisted key collision")
	}
}

func TestIdentityRecall_FollowsAnnounceDispatch(t *testing.T) {
	resetKnownDestinationsForTestIdentityParity()
	t.Cleanup(resetKnownDestinationsForTestIdentityParity)

	id, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	dst, err := NewDestination(id, DestinationIN, DestinationSINGLE, "test", "announce")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}
	packet := dst.Announce(nil, false, nil, nil, false)
	if packet == nil {
		t.Fatal("Announce() returned nil")
	}
	if err := packet.Pack(); err != nil {
		t.Fatalf("Pack(): %v", err)
	}

	knownDestinationsLoadMu.Lock()
	knownDestinations.entries = make(map[string]*knownDestinationEntry)
	knownDestinationsLoaded.Store(true)
	knownDestinationsLoadAttempted.Store(true)
	knownDestinationsLoadMu.Unlock()

	NotifyAnnounceHandlers(packet)
	if got := IdentityRecall(packet.DestinationHash); got == nil {
		t.Fatal("IdentityRecall() returned nil after announce dispatch")
	}
}

func TestIdentityDecrypt_CorruptedTokenReturnsNilWithoutError(t *testing.T) {
	id1, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(id1): %v", err)
	}
	id2, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(id2): %v", err)
	}

	token, err := id2.Encrypt([]byte("hello"), nil)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	token[len(token)-1] ^= 0xFF

	plain, err := id1.Decrypt(token, nil, false)
	if err != nil {
		t.Fatalf("Decrypt() err=%v, want nil", err)
	}
	if plain != nil {
		t.Fatalf("Decrypt()=%x, want nil", plain)
	}
}

func TestIdentityKnownDestinations_Encode_UsesPythonBinKeys(t *testing.T) {
	t.Parallel()

	key := bytes.Repeat([]byte{0x01}, ReticulumTruncatedHashLength/8)
	pub := make([]byte, 0, 64)
	pub = append(pub, bytes.Repeat([]byte{0x02}, 32)...)
	pub = append(pub, bytes.Repeat([]byte{0x03}, 32)...)

	raw, err := encodeKnownDestinations(map[string]*knownDestinationEntry{
		string(key): {
			SeenAt:     123,
			PacketHash: []byte("ph"),
			PublicKey:  pub,
			AppData:    []byte("ad"),
		},
	})
	if err != nil {
		t.Fatalf("encodeKnownDestinations: %v", err)
	}
	if len(raw) < 2 || raw[1] != 0xC4 {
		t.Fatalf("encoded key prefix=%#x, want msgpack bin8 (0xC4)", raw)
	}
}
