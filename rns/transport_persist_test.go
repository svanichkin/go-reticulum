package rns

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	umsgpack "github.com/svanichkin/go-reticulum/rns/vendor"
)

func TestSaveDestinationTable_WritesFiles(t *testing.T) {
	prevOwner := Owner
	prevTransportEnabled := transportEnabled
	prevInterfaces := Interfaces
	prevPathTable := pathTable

	dir := t.TempDir()
	Owner = &Reticulum{
		StoragePath:  filepath.Join(dir, "storage"),
		CachePath:    filepath.Join(dir, "storage", "cache"),
		ResourcePath: filepath.Join(dir, "storage", "resources"),
	}
	transportEnabled = true
	Interfaces = nil
	pathTable = make(map[hashKey]*PathEntry)

	t.Cleanup(func() {
		Owner = prevOwner
		transportEnabled = prevTransportEnabled
		Interfaces = prevInterfaces
		pathTable = prevPathTable
	})

	_ = os.MkdirAll(filepath.Join(Owner.CachePath, "announces"), 0o755)

	ifc := &Interface{Name: "if0", Type: "Test"}
	Interfaces = append(Interfaces, ifc)

	dst := make([]byte, truncatedHashBytes)
	for i := range dst {
		dst[i] = byte(i + 1)
	}
	key, ok := makeHashKey(dst)
	if !ok {
		t.Fatalf("makeHashKey failed")
	}

	packetHash := make([]byte, sha256Bits/8)
	for i := range packetHash {
		packetHash[i] = byte(0xAA + i)
	}
	rawAnnounce := []byte("not-a-real-packet")
	buf, err := umsgpack.Packb([]any{rawAnnounce, nil})
	if err != nil {
		t.Fatalf("pack cached announce: %v", err)
	}
	if err := os.WriteFile(announceCachePath(packetHash), buf, 0o600); err != nil {
		t.Fatalf("write cached announce: %v", err)
	}

	pathTable[key] = &PathEntry{
		NextHop:       []byte{1, 2, 3},
		RecvInterface: ifc,
		Hops:          1,
		Timestamp:     time.Now(),
		ExpiresAt:     time.Now().Add(time.Hour),
		RandomBlobs:   [][]byte{[]byte("x")},
		PacketHash:    packetHash,
	}

	if err := saveDestinationTable(); err != nil {
		t.Fatalf("saveDestinationTable: %v", err)
	}
	if _, err := os.Stat(filepath.Join(Owner.StoragePath, "destination_table")); err != nil {
		t.Fatalf("expected destination_table file: %v", err)
	}
	if _, err := os.Stat(announceCachePath(packetHash)); err != nil {
		t.Fatalf("expected announce cache file: %v", err)
	}
}

func TestSaveTunnelTable_TruncatesPersistedBlobs(t *testing.T) {
	prevOwner := Owner
	prevTransportEnabled := transportEnabled
	prevTunnels := tunnels

	dir := t.TempDir()
	Owner = &Reticulum{
		StoragePath: filepath.Join(dir, "storage"),
		CachePath:   filepath.Join(dir, "storage", "cache"),
	}
	transportEnabled = true
	tunnels = make(map[string]*tunnelEntry)

	t.Cleanup(func() {
		Owner = prevOwner
		transportEnabled = prevTransportEnabled
		tunnels = prevTunnels
	})

	tunnelID := make([]byte, HashLengthBytes)
	for i := range tunnelID {
		tunnelID[i] = byte(0x10 + i)
	}
	dstHash := make([]byte, truncatedHashBytes)
	for i := range dstHash {
		dstHash[i] = byte(0x22 + i)
	}
	packetHash := make([]byte, sha256Bits/8)
	for i := range packetHash {
		packetHash[i] = byte(0x33 + i)
	}

	blobs := make([][]byte, 0, persistRandomBlobs+5)
	for i := 0; i < persistRandomBlobs+5; i++ {
		blobs = append(blobs, []byte{byte(i)})
	}

	te := &tunnelEntry{
		ID:        tunnelID,
		ExpiresAt: time.Now().Add(time.Hour),
		Paths: map[string]*tunnelPathEntry{
			string(dstHash): {
				Timestamp:    time.Now(),
				ReceivedFrom: []byte{1},
				Hops:         1,
				ExpiresAt:    time.Now().Add(time.Hour),
				RandomBlobs:  blobs,
				PacketHash:   packetHash,
			},
		},
	}
	tunnels[string(tunnelID)] = te

	if err := saveTunnelTable(); err != nil {
		t.Fatalf("saveTunnelTable: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(Owner.StoragePath, "tunnels"))
	if err != nil {
		t.Fatalf("ReadFile(tunnels): %v", err)
	}
	var decoded []any
	if err := umsgpack.Unpackb(raw, &decoded); err != nil {
		t.Fatalf("umsgpack.Unpackb: %v", err)
	}
	if len(decoded) != 1 {
		t.Fatalf("expected 1 tunnel entry, got %d", len(decoded))
	}
	t0, ok := decoded[0].([]any)
	if !ok || len(t0) < 3 {
		t.Fatalf("unexpected decoded tunnel entry: %#v", decoded[0])
	}
	pathsAny, ok := t0[2].([]any)
	if !ok || len(pathsAny) != 1 {
		t.Fatalf("unexpected paths: %#v", t0[2])
	}
	p0, ok := pathsAny[0].([]any)
	if !ok || len(p0) < 6 {
		t.Fatalf("unexpected path entry: %#v", pathsAny[0])
	}
	blobsAny, ok := p0[5].([]any)
	if !ok {
		t.Fatalf("unexpected blobs type: %T", p0[5])
	}
	if len(blobsAny) != persistRandomBlobs {
		t.Fatalf("expected persisted blobs=%d got %d", persistRandomBlobs, len(blobsAny))
	}
}
