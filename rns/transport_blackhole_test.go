package rns

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	umsgpack "github.com/svanichkin/go-reticulum/rns/vendor"
)

func persistedBlackholeLookup(m map[any]any, key []byte) (any, bool) {
	if v, ok := m[string(key)]; ok {
		return v, true
	}
	if v, ok := m[umsgpack.BinaryKey(string(key))]; ok {
		return v, true
	}
	return nil, false
}

func stashKnownDestinationsForBlackholeTest() func() {
	knownDestinationsLoadMu.Lock()
	prevEntries := knownDestinations.entries
	prevLoaded := knownDestinationsLoaded.Load()
	prevAttempted := knownDestinationsLoadAttempted.Load()
	knownDestinations.entries = make(map[string]*knownDestinationEntry)
	knownDestinationsLoaded.Store(true)
	knownDestinationsLoadAttempted.Store(true)
	knownDestinationsLoadMu.Unlock()
	return func() {
		knownDestinationsLoadMu.Lock()
		knownDestinations.entries = prevEntries
		knownDestinationsLoaded.Store(prevLoaded)
		knownDestinationsLoadAttempted.Store(prevAttempted)
		knownDestinationsLoadMu.Unlock()
	}
}

func TestBlackholeIdentity_PersistsAndDropsPaths(t *testing.T) {
	prevOwner := Owner
	prevTransportIdentity := TransportIdentity
	prevPathTable := pathTable
	prevBlackholed := blackholedIdentities

	Owner = &Reticulum{StoragePath: t.TempDir()}
	TransportIdentity, _ = NewIdentity()
	pathTable = make(map[hashKey]*PathEntry)
	blackholedIdentities = make(map[hashKey]*blackholeEntry)
	restoreKnown := stashKnownDestinationsForBlackholeTest()

	t.Cleanup(func() {
		Owner = prevOwner
		TransportIdentity = prevTransportIdentity
		pathTable = prevPathTable
		blackholedIdentities = prevBlackholed
		restoreKnown()
	})

	remoteID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(remote): %v", err)
	}
	destHash := make([]byte, truncatedHashBytes)
	copy(destHash, []byte("dest-blackhole-1"))
	if err := IdentityRemember([]byte("pkt"), destHash, remoteID.GetPublicKey(), nil); err != nil {
		t.Fatalf("IdentityRemember: %v", err)
	}
	key, ok := makeHashKey(destHash)
	if !ok {
		t.Fatal("makeHashKey(destHash) failed")
	}
	pathTable[key] = &PathEntry{Timestamp: time.Now(), ExpiresAt: time.Now().Add(time.Hour)}

	reason := "test"
	if added := BlackholeIdentity(remoteID.Hash, nil, &reason); !added {
		t.Fatal("BlackholeIdentity() = false, want true")
	}
	if _, exists := pathTable[key]; exists {
		t.Fatal("expected blackholed destination to be removed from path table")
	}
	if !isBlackholedIdentity(remoteID.Hash) {
		t.Fatal("expected identity to be blackholed")
	}

	raw, err := os.ReadFile(filepath.Join(Owner.StoragePath, "blackhole", "local"))
	if err != nil {
		t.Fatalf("ReadFile(local blackhole): %v", err)
	}
	var persisted map[any]any
	if err := umsgpack.Unpackb(raw, &persisted); err != nil {
		t.Fatalf("Unpackb(local blackhole): %v", err)
	}
	entryAny, ok := persistedBlackholeLookup(persisted, remoteID.Hash)
	if !ok {
		t.Fatalf("persisted blackhole missing identity key")
	}
	entry, ok := entryAny.(map[any]any)
	if !ok {
		t.Fatalf("persisted entry type=%T", entryAny)
	}
	source, _ := asBytesValue(entry["source"])
	if !bytesEqual(source, TransportIdentity.Hash) {
		t.Fatalf("persisted source=%x, want %x", source, TransportIdentity.Hash)
	}
	if got, _ := asStringValue(entry["reason"]); got != "test" {
		t.Fatalf("persisted reason=%q, want test", got)
	}
}

func TestReloadBlackhole_LoadsEnabledSourcesAndDropsPaths(t *testing.T) {
	prevOwner := Owner
	prevTransportIdentity := TransportIdentity
	prevPathTable := pathTable
	prevBlackholed := blackholedIdentities
	prevSources := BlackholeSources()

	Owner = &Reticulum{StoragePath: t.TempDir()}
	TransportIdentity, _ = NewIdentity()
	pathTable = make(map[hashKey]*PathEntry)
	blackholedIdentities = make(map[hashKey]*blackholeEntry)
	restoreKnown := stashKnownDestinationsForBlackholeTest()

	enabledSource := []byte{0xA1, 0xB2, 0xC3, 0xD4, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB, 0xCC}
	blackholeSources = [][]byte{append([]byte(nil), enabledSource...)}

	t.Cleanup(func() {
		Owner = prevOwner
		TransportIdentity = prevTransportIdentity
		pathTable = prevPathTable
		blackholedIdentities = prevBlackholed
		blackholeSources = prevSources
		restoreKnown()
	})

	base := filepath.Join(Owner.StoragePath, "blackhole")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatalf("MkdirAll(blackhole): %v", err)
	}

	localVictim, _ := NewIdentity()
	remoteVictim, _ := NewIdentity()
	disabledVictim, _ := NewIdentity()

	localKey, _ := makeHashKey(localVictim.Hash)
	remoteKey, _ := makeHashKey(remoteVictim.Hash)
	disabledKey, _ := makeHashKey(disabledVictim.Hash)

	localPacked, _ := umsgpack.Packb(map[hashKey]map[string]any{
		localKey: {"source": TransportIdentity.Hash, "reason": "local"},
	})
	if err := os.WriteFile(filepath.Join(base, "local"), localPacked, 0o600); err != nil {
		t.Fatalf("WriteFile(local): %v", err)
	}

	remotePacked, _ := umsgpack.Packb(map[hashKey]map[string]any{
		remoteKey: {"source": enabledSource, "reason": "remote"},
	})
	if err := os.WriteFile(filepath.Join(base, hex.EncodeToString(enabledSource)), remotePacked, 0o600); err != nil {
		t.Fatalf("WriteFile(remote): %v", err)
	}

	disabledSource := []byte{0x10, 0x20, 0x30, 0x40, 0x50, 0x60, 0x70, 0x80, 0x90, 0xA0, 0xB0, 0xC0, 0xD0, 0xE0, 0xF0, 0x01}
	disabledPacked, _ := umsgpack.Packb(map[hashKey]map[string]any{
		disabledKey: {"source": disabledSource, "reason": "disabled"},
	})
	if err := os.WriteFile(filepath.Join(base, hex.EncodeToString(disabledSource)), disabledPacked, 0o600); err != nil {
		t.Fatalf("WriteFile(disabled): %v", err)
	}

	destHash := make([]byte, truncatedHashBytes)
	copy(destHash, []byte("dest-blackhole-2"))
	if err := IdentityRemember([]byte("pkt2"), destHash, remoteVictim.GetPublicKey(), nil); err != nil {
		t.Fatalf("IdentityRemember(remote): %v", err)
	}
	pathKey, _ := makeHashKey(destHash)
	pathTable[pathKey] = &PathEntry{Timestamp: time.Now(), ExpiresAt: time.Now().Add(time.Hour)}

	reloadBlackhole()

	if !isBlackholedIdentity(localVictim.Hash) {
		t.Fatal("expected local blackhole entry to be loaded")
	}
	if !isBlackholedIdentity(remoteVictim.Hash) {
		t.Fatal("expected enabled remote blackhole entry to be loaded")
	}
	if isBlackholedIdentity(disabledVictim.Hash) {
		t.Fatal("expected disabled remote blackhole entry to be skipped")
	}
	if _, exists := pathTable[pathKey]; exists {
		t.Fatal("expected blackholed path to be removed on reload")
	}
}

func TestCullBlackholedIdentities_RemovesExpiredAndPersists(t *testing.T) {
	prevOwner := Owner
	prevTransportIdentity := TransportIdentity
	prevBlackholed := blackholedIdentities

	Owner = &Reticulum{StoragePath: t.TempDir()}
	TransportIdentity, _ = NewIdentity()
	blackholedIdentities = make(map[hashKey]*blackholeEntry)

	t.Cleanup(func() {
		Owner = prevOwner
		TransportIdentity = prevTransportIdentity
		blackholedIdentities = prevBlackholed
	})

	expiredID := []byte("expired-blackhol")
	activeID := []byte("active-blackhol-")
	expiredKey, _ := makeHashKey(expiredID)
	activeKey, _ := makeHashKey(activeID)
	expiredAt := time.Now().Add(-time.Minute)
	activeUntil := time.Now().Add(time.Hour)
	blackholedIdentities[expiredKey] = &blackholeEntry{Source: TransportIdentity.Hash, Until: &expiredAt}
	blackholedIdentities[activeKey] = &blackholeEntry{Source: TransportIdentity.Hash, Until: &activeUntil}

	if changed := cullBlackholedIdentities(time.Now()); !changed {
		t.Fatal("cullBlackholedIdentities() = false, want true")
	}
	if _, exists := blackholedIdentities[expiredKey]; exists {
		t.Fatal("expected expired entry to be removed")
	}
	if _, exists := blackholedIdentities[activeKey]; !exists {
		t.Fatal("expected active entry to remain")
	}

	raw, err := os.ReadFile(filepath.Join(Owner.StoragePath, "blackhole", "local"))
	if err != nil {
		t.Fatalf("ReadFile(local blackhole): %v", err)
	}
	var persisted map[any]any
	if err := umsgpack.Unpackb(raw, &persisted); err != nil {
		t.Fatalf("Unpackb(local blackhole): %v", err)
	}
	if _, exists := persistedBlackholeLookup(persisted, expiredID); exists {
		t.Fatal("expected expired entry to be removed from persisted local list")
	}
	if _, exists := persistedBlackholeLookup(persisted, activeID); !exists {
		t.Fatal("expected active entry to remain in persisted local list")
	}
}

func TestEnsureBlackholeDestination_PublishesList(t *testing.T) {
	prevOwner := Owner
	prevTransportIdentity := TransportIdentity
	prevBlackholeDestination := blackholeDestination
	prevMgmtDestinations := mgmtDestinations
	prevPublish := publishBlackholeEnabled
	prevBlackholed := blackholedIdentities

	Owner = &Reticulum{}
	TransportIdentity, _ = NewIdentity()
	blackholeDestination = nil
	mgmtDestinations = nil
	publishBlackholeEnabled = true
	blackholedIdentities = make(map[hashKey]*blackholeEntry)

	t.Cleanup(func() {
		Owner = prevOwner
		TransportIdentity = prevTransportIdentity
		blackholeDestination = prevBlackholeDestination
		mgmtDestinations = prevMgmtDestinations
		publishBlackholeEnabled = prevPublish
		blackholedIdentities = prevBlackholed
	})

	victim := []byte("victim-blackhol-")
	key, _ := makeHashKey(victim)
	blackholedIdentities[key] = &blackholeEntry{Source: []byte("source-blackhol")}

	ensureBlackholeDestination()
	refreshManagementDestinations()

	if blackholeDestination == nil {
		t.Fatal("expected blackhole destination to be created")
	}
	if len(mgmtDestinations) != 1 || mgmtDestinations[0] != blackholeDestination {
		t.Fatalf("unexpected management destinations: %#v", mgmtDestinations)
	}
	resp, ok := blackholeDestination.DispatchRequest("/list", nil, nil, nil, nil, time.Time{})
	if !ok {
		t.Fatal("expected /list request handler to be registered")
	}
	list, ok := resp.(map[hashKey]map[string]any)
	if !ok {
		t.Fatalf("unexpected response type %T", resp)
	}
	if _, exists := list[key]; !exists {
		t.Fatal("expected blackholed identity in published list")
	}
}

func TestBlackholedIdentities_ReturnsSnapshot(t *testing.T) {
	prevBlackholed := blackholedIdentities
	blackholedIdentities = make(map[hashKey]*blackholeEntry)
	t.Cleanup(func() {
		blackholedIdentities = prevBlackholed
	})

	identityHash := []byte("snapshot-blackhol")
	key, _ := makeHashKey(identityHash)
	reason := "snapshot"
	blackholedIdentities[key] = &blackholeEntry{
		Source: []byte("source-blackhol"),
		Reason: &reason,
	}

	snapshot := BlackholedIdentities()
	if len(snapshot) != 1 {
		t.Fatalf("snapshot size=%d, want 1", len(snapshot))
	}
	entry := snapshot[key]
	entry["reason"] = "mutated"
	delete(snapshot, key)

	current := BlackholedIdentities()
	if len(current) != 1 {
		t.Fatalf("current size=%d, want 1", len(current))
	}
	if got, _ := current[key]["reason"].(string); got != "snapshot" {
		t.Fatalf("current reason=%q, want snapshot", got)
	}
}
