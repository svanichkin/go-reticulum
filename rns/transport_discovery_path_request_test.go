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
	prevInterfaces := Interfaces
	prevLocalClientInterfaces := LocalClientInterfaces
	prevDestinations := Destinations
	prevPathTable := pathTable
	prevDiscoveryPRTags := discoveryPRTags
	prevDiscoveryPRTagFIFO := discoveryPRTagFIFO
	prevDiscoveryPathRequests := discoveryPathRequests
	prevTransportEnabled := transportEnabled
	prevTransportIdentity := TransportIdentity

	transportEnabled = true
	Interfaces = nil
	LocalClientInterfaces = nil
	Destinations = nil
	pathTable = make(map[hashKey]*PathEntry)
	discoveryPRTags = make(map[string]struct{})
	discoveryPRTagFIFO = nil
	discoveryPathRequests = make(map[hashKey]*discoveryPathRequest)
	sink := &outboundCapture{}
	transportID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(transport): %v", err)
	}
	TransportIdentity = transportID

	t.Cleanup(func() {
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

	attached := &Interface{Name: "ap0", Mode: InterfaceModeAccessPoint, OUT: true}
	peerA := &Interface{Name: "peerA", OUT: true}
	peerB := &Interface{Name: "peerB", OUT: true}
	attachOutboundCapture(t, sink, attached)
	attachOutboundCapture(t, sink, peerA)
	attachOutboundCapture(t, sink, peerB)
	Interfaces = []*Interface{attached, peerA, peerB}

	destHash := make([]byte, truncatedHashBytes)
	tagA := make([]byte, truncatedHashBytes)
	tagB := make([]byte, truncatedHashBytes)
	for i := range destHash {
		destHash[i] = byte(0x30 + i)
		tagA[i] = byte(0x80 + i)
		tagB[i] = byte(0x90 + i)
	}

	pathRequestHandler(append(append([]byte(nil), destHash...), tagA...), &Packet{ReceivingInterface: attached})
	if got := len(sink.packets); got != 2 {
		t.Fatalf("first outbound requests=%d, want 2", got)
	}

	pathRequestHandler(append(append([]byte(nil), destHash...), tagB...), &Packet{ReceivingInterface: attached})
	if got := len(sink.packets); got != 2 {
		t.Fatalf("duplicate discovery request reflooded, outbound=%d want 2", got)
	}
}

func TestHandleInboundAnnounce_AnswersWaitingDiscoveryPathRequest(t *testing.T) {
	resetKnownDestinationsForTest()

	prevInterfaces := Interfaces
	prevLocalClientInterfaces := LocalClientInterfaces
	prevDestinations := Destinations
	prevPathTable := pathTable
	prevDiscoveryPRTags := discoveryPRTags
	prevDiscoveryPRTagFIFO := discoveryPRTagFIFO
	prevDiscoveryPathRequests := discoveryPathRequests
	prevTransportEnabled := transportEnabled
	prevTransportIdentity := TransportIdentity

	transportEnabled = true
	Interfaces = nil
	LocalClientInterfaces = nil
	Destinations = nil
	pathTable = make(map[hashKey]*PathEntry)
	discoveryPRTags = make(map[string]struct{})
	discoveryPRTagFIFO = nil
	discoveryPathRequests = make(map[hashKey]*discoveryPathRequest)
	sink := &outboundCapture{}

	transportID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(transport): %v", err)
	}
	TransportIdentity = transportID

	t.Cleanup(func() {
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

	requester := &Interface{Name: "ap0", Mode: InterfaceModeAccessPoint, OUT: true}
	source := &Interface{Name: "peerA", OUT: true}
	other := &Interface{Name: "peerB", OUT: true}
	attachOutboundCapture(t, sink, requester)
	attachOutboundCapture(t, sink, source)
	attachOutboundCapture(t, sink, other)
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

	pathRequestHandler(append(append([]byte(nil), announce.DestinationHash...), tag...), &Packet{ReceivingInterface: requester})
	if got := len(sink.packets); got != 2 {
		t.Fatalf("path discovery outbound=%d, want 2", got)
	}

	key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(announce.DestinationHash)
	if !ok {
		t.Fatal("announce destination hash invalid")
	}
	if _, exists := discoveryPathRequests[key]; !exists {
		t.Fatal("expected discovery path request to be stored")
	}

	Inbound(announce.Raw, source)

	if got := len(sink.packets); got < 2 {
		t.Fatalf("outbound packets=%d, want at least 2 for the discovery request flow", got)
	}
	if got := len(sink.packets); got >= 3 {
		response := sink.packets[2]
		if response.PacketType != PacketANNOUNCE {
			t.Fatalf("response packet type=%d, want announce", response.PacketType)
		}
		if response.Context != PacketPATH_RESPONSE {
			t.Fatalf("response context=%d, want PATH_RESPONSE", response.Context)
		}
		if response.AttachedInterface != nil {
			t.Fatalf("response attached interface=%p, want nil", response.AttachedInterface)
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
	}
	if _, exists := discoveryPathRequests[key]; exists {
		t.Fatal("expected discovery path request to be removed after matching announce")
	}
}

func TestCullDiscoveryPathRequests_RemovesExpiredEntries(t *testing.T) {
	prevDiscoveryPathRequests := discoveryPathRequests
	prevPendingLocalPathRequests := pendingLocalPathRequests
	prevPendingPRsLastChecked := pendingPRsLastChecked
	prevLinksLastChecked := LinksLastChecked
	prevReceiptsLast := ReceiptsLast
	prevAnnLast := AnnLast
	prevTablesLastCulled := TablesLastCulled
	prevLastCacheCleaned := LastCacheCleaned
	prevLastTablesPersisted := lastTablesPersisted
	prevInterfaceLastJobs := InterfaceLastJobs
	prevBlackholeLastChecked := blackholeLastChecked

	expiredKey := hashKey{1}
	liveKey := hashKey{2}
	now := time.Now()
	discoveryPathRequests = map[hashKey]*discoveryPathRequest{
		expiredKey: {Timeout: now.Add(-time.Second), RequestingInterface: &Interface{Name: "expired"}},
		liveKey:    {Timeout: now.Add(time.Second), RequestingInterface: &Interface{Name: "live"}},
	}
	pendingLocalPathRequests = make(map[hashKey]*Interface)
	pendingPRsLastChecked = now.Add(-pendingPRsCheckInterval - time.Second)
	LinksLastChecked = now
	ReceiptsLast = now
	AnnLast = now
	TablesLastCulled = now
	LastCacheCleaned = now
	lastTablesPersisted = now
	InterfaceLastJobs = now
	blackholeLastChecked = now

	t.Cleanup(func() {
		discoveryPathRequests = prevDiscoveryPathRequests
		pendingLocalPathRequests = prevPendingLocalPathRequests
		pendingPRsLastChecked = prevPendingPRsLastChecked
		LinksLastChecked = prevLinksLastChecked
		ReceiptsLast = prevReceiptsLast
		AnnLast = prevAnnLast
		TablesLastCulled = prevTablesLastCulled
		LastCacheCleaned = prevLastCacheCleaned
		lastTablesPersisted = prevLastTablesPersisted
		InterfaceLastJobs = prevInterfaceLastJobs
		blackholeLastChecked = prevBlackholeLastChecked
	})

	Jobs()
	if _, exists := discoveryPathRequests[expiredKey]; exists {
		t.Fatal("expired discovery path request was not removed")
	}
	if _, exists := discoveryPathRequests[liveKey]; !exists {
		t.Fatal("live discovery path request was removed")
	}
}
