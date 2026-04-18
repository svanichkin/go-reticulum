package rns

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReticulumShouldPersistData_PersistsIdentityAndTransport(t *testing.T) {
	dir := t.TempDir()

	prevInstance := instance
	prevOwner := Owner
	prevTransportEnabled := transportEnabled
	prevPacketHashSet := PacketHashSet
	prevPacketHashSet2 := PacketHashSet2

	r := &Reticulum{
		StoragePath:  filepath.Join(dir, "storage"),
		CachePath:    filepath.Join(dir, "storage", "cache"),
		ResourcePath: filepath.Join(dir, "storage", "resources"),
		lastDataPersist: time.Now().Add(
			-(time.Duration(GRACIOUS_PERSIST_INTERVAL + 1)) * time.Second,
		),
	}

	instance = r
	Owner = r
	transportEnabled = true
	PacketHashSet = make(map[hashKey]struct{})
	PacketHashSet2 = make(map[hashKey]struct{})

	t.Cleanup(func() {
		instance = prevInstance
		Owner = prevOwner
		transportEnabled = prevTransportEnabled
		PacketHashSet = prevPacketHashSet
		PacketHashSet2 = prevPacketHashSet2
	})

	if err := os.MkdirAll(r.StoragePath, 0o755); err != nil {
		t.Fatalf("mkdir storage: %v", err)
	}
	if key, ok := func(hash []byte) (hashKey, bool) {
	if len(hash) < truncatedHashBytes {
		return hashKey{}, false
	}
	var key hashKey
	copy(key[:], hash[:truncatedHashBytes])
	return key, true
}(bytesRepeat(0xAB, truncatedHashBytes)); ok {
		PacketHashSet[key] = struct{}{}
	}

	r.ShouldPersistData()

	if _, err := os.Stat(filepath.Join(r.StoragePath, "known_destinations")); err != nil {
		t.Fatalf("expected known_destinations file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(r.StoragePath, "packet_hashlist")); err != nil {
		t.Fatalf("expected packet_hashlist file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(r.StoragePath, "destination_table")); err != nil {
		t.Fatalf("expected destination_table file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(r.StoragePath, "tunnels")); err != nil {
		t.Fatalf("expected tunnels file: %v", err)
	}
}

func bytesRepeat(b byte, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}
