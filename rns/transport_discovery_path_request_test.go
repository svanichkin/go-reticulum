package rns

import (
	"bytes"
	"testing"
	"time"
)

type discoveryPathRequestCaptureBackend struct {
	packets []*Packet
}

func (b *discoveryPathRequestCaptureBackend) Outbound(p *Packet) bool {
	if p != nil {
		b.packets = append(b.packets, p)
	}
	return true
}

func (b *discoveryPathRequestCaptureBackend) HopsTo(_ []byte) int {
	return PathfinderMaxHops
}

func (b *discoveryPathRequestCaptureBackend) GetFirstHopTimeout(_ []byte) time.Duration {
	return 0
}

func (b *discoveryPathRequestCaptureBackend) GetPacketRSSI(_ []byte) *float64 {
	return nil
}

func (b *discoveryPathRequestCaptureBackend) GetPacketSNR(_ []byte) *float64 {
	return nil
}

func (b *discoveryPathRequestCaptureBackend) GetPacketQ(_ []byte) *float64 {
	return nil
}

func TestPathRequestHandler_DuplicateDiscoveryRequestDoesNotReflood(t *testing.T) {
	prevTransport := Transport
	prevInterfaces := Interfaces
	prevLocalClientInterfaces := LocalClientInterfaces
	prevDestinations := Destinations
	prevPathTable := pathTable
	prevDiscoveryPRTags := discoveryPRTags
	prevDiscoveryPRTagFIFO := discoveryPRTagFIFO
	prevDiscoveryPathRequests := discoveryPathRequests
	prevTransportEnabled := transportEnabled

	backend := &discoveryPathRequestCaptureBackend{}
	Transport = backend
	transportEnabled = true
	Interfaces = nil
	LocalClientInterfaces = nil
	Destinations = nil
	pathTable = make(map[hashKey]*PathEntry)
	discoveryPRTags = make(map[string]struct{})
	discoveryPRTagFIFO = nil
	discoveryPathRequests = make(map[hashKey]*discoveryPathRequest)

	t.Cleanup(func() {
		Transport = prevTransport
		Interfaces = prevInterfaces
		LocalClientInterfaces = prevLocalClientInterfaces
		Destinations = prevDestinations
		pathTable = prevPathTable
		discoveryPRTags = prevDiscoveryPRTags
		discoveryPRTagFIFO = prevDiscoveryPRTagFIFO
		discoveryPathRequests = prevDiscoveryPathRequests
		transportEnabled = prevTransportEnabled
	})

	attached := &Interface{Name: "ap0", Mode: InterfaceModeAccessPoint}
	peerA := &Interface{Name: "peerA"}
	peerB := &Interface{Name: "peerB"}
	Interfaces = []*Interface{attached, peerA, peerB}

	destHash := make([]byte, truncatedHashBytes)
	tagA := make([]byte, truncatedHashBytes)
	tagB := make([]byte, truncatedHashBytes)
	for i := range destHash {
		destHash[i] = byte(0x30 + i)
		tagA[i] = byte(0x80 + i)
		tagB[i] = byte(0x90 + i)
	}

	pathRequestHandler(append(copyBytes(destHash), tagA...), &Packet{ReceivingInterface: attached})
	if got := len(backend.packets); got != 2 {
		t.Fatalf("first outbound requests=%d, want 2", got)
	}

	pathRequestHandler(append(copyBytes(destHash), tagB...), &Packet{ReceivingInterface: attached})
	if got := len(backend.packets); got != 2 {
		t.Fatalf("duplicate discovery request reflooded, outbound=%d want 2", got)
	}
}

func TestHandleInboundAnnounce_AnswersWaitingDiscoveryPathRequest(t *testing.T) {
	resetKnownDestinationsForTest()

	prevTransport := Transport
	prevInterfaces := Interfaces
	prevLocalClientInterfaces := LocalClientInterfaces
	prevDestinations := Destinations
	prevPathTable := pathTable
	prevDiscoveryPRTags := discoveryPRTags
	prevDiscoveryPRTagFIFO := discoveryPRTagFIFO
	prevDiscoveryPathRequests := discoveryPathRequests
	prevTransportEnabled := transportEnabled
	prevTransportIdentity := TransportIdentity

	backend := &discoveryPathRequestCaptureBackend{}
	Transport = backend
	transportEnabled = true
	Interfaces = nil
	LocalClientInterfaces = nil
	Destinations = nil
	pathTable = make(map[hashKey]*PathEntry)
	discoveryPRTags = make(map[string]struct{})
	discoveryPRTagFIFO = nil
	discoveryPathRequests = make(map[hashKey]*discoveryPathRequest)

	transportID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(transport): %v", err)
	}
	TransportIdentity = transportID

	t.Cleanup(func() {
		Transport = prevTransport
		Interfaces = prevInterfaces
		LocalClientInterfaces = prevLocalClientInterfaces
		Destinations = prevDestinations
		pathTable = prevPathTable
		discoveryPRTags = prevDiscoveryPRTags
		discoveryPRTagFIFO = prevDiscoveryPRTagFIFO
		discoveryPathRequests = prevDiscoveryPathRequests
		transportEnabled = prevTransportEnabled
		TransportIdentity = prevTransportIdentity
	})

	requester := &Interface{Name: "ap0", Mode: InterfaceModeAccessPoint}
	source := &Interface{Name: "peerA"}
	other := &Interface{Name: "peerB"}
	Interfaces = []*Interface{requester, source, other}

	announceID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(announce): %v", err)
	}
	dst, err := NewDestination(announceID, DestinationIN, DestinationSINGLE, "test", "announce")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}
	announce := dst.Announce(nil, false, nil, nil, false)
	if announce == nil {
		t.Fatal("Announce returned nil")
	}
	if err := announce.Pack(); err != nil {
		t.Fatalf("Pack(): %v", err)
	}
	announce.ReceivingInterface = source
	announce.Hops = 1
	Destinations = nil

	tag := make([]byte, truncatedHashBytes)
	for i := range tag {
		tag[i] = byte(0xA0 + i)
	}

	pathRequestHandler(append(copyBytes(announce.DestinationHash), tag...), &Packet{ReceivingInterface: requester})
	if got := len(backend.packets); got != 2 {
		t.Fatalf("path discovery outbound=%d, want 2", got)
	}

	key, ok := makeHashKey(announce.DestinationHash)
	if !ok {
		t.Fatal("announce destination hash invalid")
	}
	if _, exists := discoveryPathRequests[key]; !exists {
		t.Fatal("expected discovery path request to be stored")
	}

	handleInboundAnnounce(announce, source, false)

	if got := len(backend.packets); got != 3 {
		t.Fatalf("outbound packets=%d, want 3 including PATH_RESPONSE", got)
	}

	response := backend.packets[2]
	if response.PacketType != PacketANNOUNCE {
		t.Fatalf("response packet type=%d, want announce", response.PacketType)
	}
	if response.Context != PacketPATH_RESPONSE {
		t.Fatalf("response context=%d, want PATH_RESPONSE", response.Context)
	}
	if response.AttachedInterface != requester {
		t.Fatalf("response attached interface=%p, want requester %p", response.AttachedInterface, requester)
	}
	if response.HeaderType != HeaderType2 {
		t.Fatalf("response header type=%d, want HEADER_2", response.HeaderType)
	}
	if response.TransportType != TransportDirect {
		t.Fatalf("response transport type=%d, want transport direct", response.TransportType)
	}
	if !bytes.Equal(response.TransportID, TransportIdentity.Hash) {
		t.Fatalf("response transport id mismatch")
	}
	if !bytes.Equal(response.DestinationHash, announce.DestinationHash) {
		t.Fatalf("response destination hash mismatch")
	}
	if _, exists := discoveryPathRequests[key]; exists {
		t.Fatal("expected discovery path request to be removed after matching announce")
	}
}

func TestCullDiscoveryPathRequests_RemovesExpiredEntries(t *testing.T) {
	prevDiscoveryPathRequests := discoveryPathRequests

	expiredKey := hashKey{1}
	liveKey := hashKey{2}
	now := time.Now()
	discoveryPathRequests = map[hashKey]*discoveryPathRequest{
		expiredKey: {Timeout: now.Add(-time.Second), RequestingInterface: &Interface{Name: "expired"}},
		liveKey:    {Timeout: now.Add(time.Second), RequestingInterface: &Interface{Name: "live"}},
	}

	t.Cleanup(func() {
		discoveryPathRequests = prevDiscoveryPathRequests
	})

	if removed := cullDiscoveryPathRequests(now); !removed {
		t.Fatal("expected expired discovery path request to be removed")
	}
	if _, exists := discoveryPathRequests[expiredKey]; exists {
		t.Fatal("expired discovery path request was not removed")
	}
	if _, exists := discoveryPathRequests[liveKey]; !exists {
		t.Fatal("live discovery path request was removed")
	}
}
