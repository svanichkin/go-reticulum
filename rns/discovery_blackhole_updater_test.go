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
	if err := os.MkdirAll(filepath.Join(Owner.StoragePath, "blackhole"), 0o755); err != nil {
		t.Fatalf("MkdirAll(blackhole): %v", err)
	}

	sourceHash := []byte("remote-bh-source")
	blackholeSources = [][]byte{append([]byte(nil), sourceHash...)}

	t.Cleanup(func() {
		Owner = prevOwner
		TransportIdentity = prevTransportIdentity
		blackholedIdentities = prevBlackholed
		blackholeSources = prevSources
	})

	victimHash := []byte("remote-bh-victim")
	victimKey, _ := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(victimHash)
	fetchCalls := 0

	updater := NewBlackholeUpdater()
	updater.awaitPath = func(hash []byte, timeout float64, onInterface *Interface) bool {
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

	updater.initialWait = 0
	updater.sleep = func(time.Duration) {
		updater.mu.Lock()
		updater.shouldRun = false
		updater.mu.Unlock()
	}
	updater.mu.Lock()
	updater.shouldRun = true
	updater.mu.Unlock()

	finished := make(chan struct{})
	go func() {
		defer close(finished)
		updater.job()
	}()

	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for blackhole updater job")
	}

	if fetchCalls != 1 {
		t.Fatalf("fetch calls=%d, want 1", fetchCalls)
	}
	if key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(victimHash); ok {
		blackholeMu.RLock()
		_, blackholed := blackholedIdentities[key]
		blackholeMu.RUnlock()
		if !blackholed {
			t.Fatal("expected remote blackhole entry to be merged")
		}
	} else {
		t.Fatal("expected remote blackhole entry to be merged")
	}

	persistedPath := filepath.Join(Owner.StoragePath, "blackhole", hex.EncodeToString(sourceHash))
	if _, err := os.Stat(persistedPath); err != nil {
		t.Fatalf("Stat(remote blackhole file): %v", err)
	}
}
