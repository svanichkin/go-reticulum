package rns

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBlackholeUpdater_UpdateOnce_MergesAndPersistsRemoteList(t *testing.T) {
	prevOwner := Owner
	prevTransportIdentity := TransportIdentity
	prevBlackholed := blackholedIdentities
	prevSources := BlackholeSources()

	Owner = &Reticulum{StoragePath: t.TempDir()}
	transportID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(transport): %v", err)
	}
	TransportIdentity = transportID
	blackholedIdentities = make(map[hashKey]*blackholeEntry)

	sourceHash := []byte("remote-bh-source")
	blackholeSources = [][]byte{append([]byte(nil), sourceHash...)}

	t.Cleanup(func() {
		Owner = prevOwner
		TransportIdentity = prevTransportIdentity
		blackholedIdentities = prevBlackholed
		blackholeSources = prevSources
	})

	victimHash := []byte("remote-bh-victim")
	victimKey, _ := makeHashKey(victimHash)
	fetchCalls := 0

	updater := NewBlackholeUpdater()
	updater.awaitPath = func(hash []byte, timeout time.Duration, onInterface *Interface) bool {
		return len(hash) == truncatedHashBytes && timeout == 0 && onInterface == nil
	}
	updater.fetchList = func(source []byte, timeout time.Duration) (any, error) {
		fetchCalls++
		if !bytesEqual(source, sourceHash) {
			t.Fatalf("fetch source=%x, want %x", source, sourceHash)
		}
		if timeout != blackholeSourceTimeout {
			t.Fatalf("fetch timeout=%v, want %v", timeout, blackholeSourceTimeout)
		}
		return map[hashKey]map[string]any{
			victimKey: {
				"source": sourceHash,
				"reason": "remote-test",
			},
		}, nil
	}

	now := time.Now()
	updater.updateOnce(now)
	updater.updateOnce(now.Add(time.Minute))

	if fetchCalls != 1 {
		t.Fatalf("fetch calls=%d, want 1", fetchCalls)
	}
	if !isBlackholedIdentity(victimHash) {
		t.Fatal("expected remote blackhole entry to be merged")
	}

	persistedPath := filepath.Join(Owner.StoragePath, "blackhole", hex.EncodeToString(sourceHash))
	if _, err := os.Stat(persistedPath); err != nil {
		t.Fatalf("Stat(remote blackhole file): %v", err)
	}
}
