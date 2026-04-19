package rns

import (
	"testing"
	"time"
)

type awaitPathTransportBackend struct {
	outboundCalls int
}

func (b *awaitPathTransportBackend) Outbound(p *Packet) bool {
	b.outboundCalls++
	return p != nil
}

func (b *awaitPathTransportBackend) HopsTo(_ []byte) int {
	return PathfinderMaxHops
}

func (b *awaitPathTransportBackend) GetFirstHopTimeout(_ []byte) time.Duration {
	return 0
}

func (b *awaitPathTransportBackend) GetPacketRSSI(_ []byte) *float64 {
	return nil
}

func (b *awaitPathTransportBackend) GetPacketSNR(_ []byte) *float64 {
	return nil
}

func (b *awaitPathTransportBackend) GetPacketQ(_ []byte) *float64 {
	return nil
}

func TestAwaitPath_ReturnsImmediatelyWhenPathExists(t *testing.T) {
	prevPathTable := pathTable
	prevLastPathRequest := lastPathRequest
	prevInterfaces := Interfaces

	pathTable = make(map[hashKey]*PathEntry)
	lastPathRequest = make(map[hashKey]time.Time)
	Interfaces = nil

	t.Cleanup(func() {
		pathTable = prevPathTable
		lastPathRequest = prevLastPathRequest
		Interfaces = prevInterfaces
	})

	destHash := make([]byte, truncatedHashBytes)
	for i := range destHash {
		destHash[i] = byte(i + 1)
	}
	key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(destHash)
	if !ok {
		t.Fatalf("makeHashKey failed")
	}
	ifc := &Interface{Name: "await-if0", Type: "Test", OUT: true}
	sink := &outboundCapture{}
	attachOutboundCapture(t, sink, ifc)
	Interfaces = []*Interface{ifc}
	pathTableMu.Lock()
	pathTable[key] = &PathEntry{Timestamp: time.Now()}
	pathTableMu.Unlock()

	if !AwaitPath(destHash, 0.1, nil) {
		t.Fatalf("AwaitPath() = false, want true")
	}
	if len(sink.packets) != 0 {
		t.Fatalf("expected no outbound requests, got %d", len(sink.packets))
	}
}

func TestAwaitPath_RequestsAndWaitsForPath(t *testing.T) {
	prevPathTable := pathTable
	prevLastPathRequest := lastPathRequest
	prevInterfaces := Interfaces

	pathTable = make(map[hashKey]*PathEntry)
	lastPathRequest = make(map[hashKey]time.Time)
	Interfaces = nil

	t.Cleanup(func() {
		pathTable = prevPathTable
		lastPathRequest = prevLastPathRequest
		Interfaces = prevInterfaces
	})

	destHash := make([]byte, truncatedHashBytes)
	for i := range destHash {
		destHash[i] = byte(0x20 + i)
	}
	key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(destHash)
	if !ok {
		t.Fatalf("makeHashKey failed")
	}
	ifc := &Interface{Name: "await-if1", Type: "Test", OUT: true}
	sink := &outboundCapture{}
	attachOutboundCapture(t, sink, ifc)
	Interfaces = []*Interface{ifc}

	go func() {
		time.Sleep(20 * time.Millisecond)
		pathTableMu.Lock()
		pathTable[key] = &PathEntry{Timestamp: time.Now()}
		pathTableMu.Unlock()
	}()

	if !AwaitPath(destHash, 0.25, nil) {
		t.Fatalf("AwaitPath() = false, want true")
	}
	if len(sink.packets) != 1 {
		t.Fatalf("expected 1 outbound request, got %d", len(sink.packets))
	}
	if _, ok := lastPathRequest[key]; !ok {
		t.Fatalf("expected lastPathRequest to be updated")
	}
}

func TestAwaitPath_TimesOutWhenPathNeverAppears(t *testing.T) {
	prevPathTable := pathTable
	prevLastPathRequest := lastPathRequest
	prevInterfaces := Interfaces

	pathTable = make(map[hashKey]*PathEntry)
	lastPathRequest = make(map[hashKey]time.Time)
	Interfaces = nil

	t.Cleanup(func() {
		pathTable = prevPathTable
		lastPathRequest = prevLastPathRequest
		Interfaces = prevInterfaces
	})

	destHash := make([]byte, truncatedHashBytes)
	for i := range destHash {
		destHash[i] = byte(0x40 + i)
	}
	ifc := &Interface{Name: "await-if2", Type: "Test", OUT: true}
	sink := &outboundCapture{}
	attachOutboundCapture(t, sink, ifc)
	Interfaces = []*Interface{ifc}

	if AwaitPath(destHash, 0.06, nil) {
		t.Fatalf("AwaitPath() = true, want false")
	}
	if len(sink.packets) != 1 {
		t.Fatalf("expected 1 outbound request, got %d", len(sink.packets))
	}
}
