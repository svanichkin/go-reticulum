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

		l, err := NewOutgoingLink(destOut, LinkModeDefault, nil, nil)
		if err != nil {
			t.Fatalf("NewOutgoingLink: %v", err)
		}
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if l.Status == LinkActive {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		if l.Status != LinkActive {
			t.Fatalf("expected link active, got %d", l.Status)
		}
		defer l.Teardown()

		data := make([]byte, 256*1000)
		_, _ = rand.Read(data)
		timeoutSeconds := 120.0
		res, err := NewResource(data, nil, l, nil, false, true, nil, nil, &timeoutSeconds, 0, nil, nil, false, 0)
		if err != nil {
			t.Fatalf("NewResource: %v", err)
		}
		res.status = ResourceTransferring
		res.advSent = time.Now()
		res.retriesLeft = res.maxRetries
		res.receiverMinHeight = 0

		if len(res.hashmap) == 0 || len(res.outgoingParts) == 0 {
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
	if err := os.WriteFile(storagePath, []byte("payload"), 0o600); err != nil {
		t.Fatalf("WriteFile(storage): %v", err)
	}

	metaValue := []any{"hello", 42}
	packed, err := umsgpack.Packb(metaValue)
	if err != nil {
		t.Fatalf("Packb(metadata): %v", err)
	}
	if err := os.WriteFile(metaPath, packed, 0o600); err != nil {
		t.Fatalf("WriteFile(meta): %v", err)
	}

	r := &Resource{
		hasMetadata:     true,
		metaStoragePath: metaPath,
		storagePath:     storagePath,
		segmentIndex:    1,
		totalSegments:   1,
		status:          ResourceComplete,
	}

	r.handleIncomingCompletion()

	got, ok := r.Metadata().([]any)
	if !ok {
		t.Fatalf("metadata type = %T, want []any", r.Metadata())
	}
	if len(got) != 2 || got[0] != "hello" {
		t.Fatalf("unexpected metadata %#v", got)
	}
}

func TestResourceHandleIncomingCompletion_CleansReceiverFileAfterCallback(t *testing.T) {
	dir := t.TempDir()
	storagePath := filepath.Join(dir, "res.data")
	if err := os.WriteFile(storagePath, []byte("payload"), 0o600); err != nil {
		t.Fatalf("WriteFile(storage): %v", err)
	}

	var sawExists atomic.Bool
	var sawData atomic.Bool
	r := &Resource{
		storagePath:   storagePath,
		segmentIndex:  1,
		totalSegments: 1,
		status:        ResourceComplete,
		callback: func(res *Resource) {
			if _, err := os.Stat(res.DataFile()); err == nil {
				sawExists.Store(true)
			}
			data, err := os.ReadFile(res.DataFile())
			if err == nil && bytes.Equal(data, []byte("payload")) {
				sawData.Store(true)
			}
		},
	}

	r.handleIncomingCompletion()

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
