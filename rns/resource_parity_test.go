package rns

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	umsgpack "github.com/svanichkin/go-reticulum/rns/vendor"
)

func TestResourceRequest_HMUOnlyStillSendsHMU(t *testing.T) {
	requireIntegration(t)
	resetKnownDestinationsForTest()
	withIntegrationTransport(t, func() {
		prvHex := "f8953ffaf607627e615603ff1530c82c434cf87c07179dd7689ea776f30b964cfb7ba6164af00c5111a45e69e57d885e1285f8dbfe3a21e95ae17cf676b0f8b7"
		prv, _ := hex.DecodeString(prvHex)
		id, err := IdentityFromBytes(prv)
		if err != nil {
			t.Fatalf("IdentityFromBytes: %v", err)
		}

		const appName = "rns_resource_parity"
		destOut, err := NewDestination(id, DestinationOUT, DestinationSINGLE, appName, "link", "establish")
		if err != nil {
			t.Fatalf("NewDestination(out): %v", err)
		}
		_, err = NewDestination(id, DestinationIN, DestinationSINGLE, appName, "link", "establish")
		if err != nil {
			t.Fatalf("NewDestination(in): %v", err)
		}

		l, err := NewLink(destOut, nil, LinkModeDefault, nil, nil)
		if err != nil {
			t.Fatalf("NewOutgoingLink: %v", err)
		}
		waitForLinkActiveOrSkip(t, l, 2*time.Second)
		defer l.Teardown()

		data := make([]byte, 256*1000)
		_, _ = rand.Read(data)
		timeoutSeconds := 120.0
		res, err := NewResource(data, nil, l, nil, false, true, nil, nil, &timeoutSeconds, 0, nil, nil, false, 0)
		if err != nil {
			t.Fatalf("NewResource: %v", err)
		}
		res.Status = ResourceTransferring
		res.advSent = time.Now()
		res.retriesLeft = res.maxRetries
		res.receiverMinConsecutiveHeight = 0

		if len(res.hashmap) == 0 || len(res.parts) == 0 {
			t.Fatalf("expected populated hashmap/outgoing parts")
		}

		resetIntegrationTransportStats()
		lastMap := res.hashmap[minInt(res.hashmapMaxLen, len(res.hashmap))-1]
		req := append([]byte{HashmapExhausted}, lastMap...)
		req = append(req, res.hash...)
		res.Request(req)

		it := getIntegrationTransport()
		if it == nil {
			t.Fatalf("expected integration transport")
		}
		it.mu.Lock()
		hmuSent := it.sentByContext[PacketCtxResourceHMU]
		it.mu.Unlock()
		if hmuSent == 0 {
			t.Fatalf("expected HMU packet to be sent for HMU-only request")
		}
	})
}

func TestResourceHandleIncomingCompletion_PreservesNonMapMetadata(t *testing.T) {
	dir := t.TempDir()
	metaPath := filepath.Join(dir, "res.meta")
	storagePath := filepath.Join(dir, "res.data")

	metaValue := []any{"hello", 42}
	packed, err := umsgpack.Packb(metaValue)
	if err != nil {
		t.Fatalf("Packb(metadata): %v", err)
	}
	if err := os.WriteFile(metaPath, packed, 0o600); err != nil {
		t.Fatalf("WriteFile(meta): %v", err)
	}

	randomHash := make([]byte, RandomHashSize)
	if _, err := rand.Read(randomHash); err != nil {
		t.Fatalf("rand.Read(randomHash): %v", err)
	}
	body := append([]byte{
		byte(len(packed) >> 16),
		byte(len(packed) >> 8),
		byte(len(packed)),
	}, packed...)
	body = append(body, []byte("payload")...)
	assembled := append([]byte(nil), body...)
	hash := FullHash(append(append([]byte(nil), assembled...), randomHash...))

	r := &Resource{
		Link:            &Link{Status: LinkClosed, destination: &Destination{Type: DestinationOUT, Hash: make([]byte, truncatedHashBytes)}},
		parts:           []any{append(randomHash, assembled...)},
		randomHash:      randomHash,
		hash:            hash,
		hasMetadata:     true,
		metaStoragePath: metaPath,
		DataFile:        storagePath,
		segmentIndex:    1,
		totalSegments:   1,
		Status:          ResourceComplete,
	}

	r.Assemble()

	if r.Metadata != nil {
		t.Fatalf("metadata = %#v, want nil without callback", r.Metadata)
	}
	if _, err := os.Stat(metaPath); err != nil {
		t.Fatalf("expected metadata file to remain without callback, stat err=%v", err)
	}
}

func TestResourceHandleIncomingCompletion_CleansReceiverFileAfterCallback(t *testing.T) {
	dir := t.TempDir()
	storagePath := filepath.Join(dir, "res.data")
	payload := []byte("payload")
	randomHash := make([]byte, RandomHashSize)
	if _, err := rand.Read(randomHash); err != nil {
		t.Fatalf("rand.Read(randomHash): %v", err)
	}
	hash := FullHash(append(append([]byte(nil), payload...), randomHash...))

	var sawExists atomic.Bool
	var sawData atomic.Bool
	r := &Resource{
		Link:          &Link{Status: LinkClosed, destination: &Destination{Type: DestinationOUT, Hash: make([]byte, truncatedHashBytes)}},
		parts:         []any{append(randomHash, payload...)},
		randomHash:    randomHash,
		hash:          hash,
		DataFile:      storagePath,
		segmentIndex:  1,
		totalSegments: 1,
		Status:        ResourceComplete,
		callback: func(res *Resource) {
			if _, err := os.Stat(res.DataFile); err == nil {
				sawExists.Store(true)
			}
			data, err := os.ReadFile(res.DataFile)
			if err == nil && bytes.Equal(data, []byte("payload")) {
				sawData.Store(true)
			}
		},
	}

	r.Assemble()

	if !sawExists.Load() || !sawData.Load() {
		t.Fatalf("expected callback to access receiver file before cleanup")
	}
	if _, err := os.Stat(storagePath); !os.IsNotExist(err) {
		t.Fatalf("expected receiver file to be deleted after callback, stat err=%v", err)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
