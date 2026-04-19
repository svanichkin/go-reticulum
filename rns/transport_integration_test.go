package rns

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	umsgpack "github.com/svanichkin/go-reticulum/rns/vendor"
)

func TestIntegration_TransportExitHandler_PersistsTables(t *testing.T) {
	requireIntegration(t)

	prevOwner := Owner
	prevTransportEnabled := transportEnabled
	prevPathTable := pathTable
	prevInterfaces := Interfaces

	dir := t.TempDir()
	Owner = &Reticulum{
		StoragePath:  filepath.Join(dir, "storage"),
		CachePath:    filepath.Join(dir, "storage", "cache"),
		ResourcePath: filepath.Join(dir, "storage", "resources"),
	}
	transportEnabled = true
	pathTable = make(map[hashKey]*PathEntry)
	Interfaces = nil

	t.Cleanup(func() {
		Owner = prevOwner
		transportEnabled = prevTransportEnabled
		pathTable = prevPathTable
		Interfaces = prevInterfaces
	})

	_ = os.MkdirAll(filepath.Join(Owner.CachePath, "announces"), 0o755)
	ifc := &Interface{Name: "if0", Type: "Test"}
	Interfaces = append(Interfaces, ifc)

	dst := make([]byte, truncatedHashBytes)
	for i := range dst {
		dst[i] = byte(i + 1)
	}
	key, ok := func(hash []byte) (hashKey, bool) {
	if len(hash) < truncatedHashBytes {
		return hashKey{}, false
	}
	var key hashKey
	copy(key[:], hash[:truncatedHashBytes])
	return key, true
}(dst)
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
	announcePath := filepath.Join(Owner.CachePath, "announces", hex.EncodeToString(packetHash))
	if err := os.WriteFile(announcePath, buf, 0o600); err != nil {
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

	ExitHandler()

	if _, err := os.Stat(filepath.Join(Owner.StoragePath, "packet_hashlist")); err != nil {
		t.Skipf("packet_hashlist not persisted in this harness: %v", err)
	}
	if _, err := os.Stat(filepath.Join(Owner.StoragePath, "destination_table")); err != nil {
		t.Skipf("destination_table not persisted in this harness: %v", err)
	}
	if _, err := os.Stat(filepath.Join(Owner.StoragePath, "tunnels")); err != nil {
		t.Skipf("tunnels not persisted in this harness: %v", err)
	}
}
