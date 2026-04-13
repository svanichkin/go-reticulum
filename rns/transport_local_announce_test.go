package rns

import (
	"bytes"
	"testing"
	"time"
)

type announceCaptureBackend struct {
	packets []*Packet
}

func (b *announceCaptureBackend) Outbound(p *Packet) bool {
	if p != nil {
		b.packets = append(b.packets, p)
	}
	return true
}

func (b *announceCaptureBackend) HopsTo(_ []byte) int { return PathfinderMaxHops }

func (b *announceCaptureBackend) GetFirstHopTimeout(_ []byte) time.Duration { return 0 }

func (b *announceCaptureBackend) GetPacketRSSI(_ []byte) *float64 { return nil }

func (b *announceCaptureBackend) GetPacketSNR(_ []byte) *float64 { return nil }

func (b *announceCaptureBackend) GetPacketQ(_ []byte) *float64 { return nil }

func TestHandleInboundAnnounce_LocalClientPathResponseTargetsRequestingInterface(t *testing.T) {
	resetKnownDestinationsForTest()

	prevTransport := Transport
	prevTransportEnabled := transportEnabled
	prevTransportIdentity := TransportIdentity
	prevPending := pendingLocalPathRequests
	prevAnnounceTable := announceTable
	prevHeldAnnounces := heldAnnounces
	prevLocalClients := LocalClientInterfaces
	prevInterfaces := Interfaces
	prevDestinations := Destinations

	backend := &announceCaptureBackend{}
	Transport = backend
	transportEnabled = true
	pendingLocalPathRequests = make(map[hashKey]*Interface)
	announceTable = make(map[hashKey]*announceEntry)
	heldAnnounces = make(map[hashKey]*heldAnnounce)

	transportID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(transport): %v", err)
	}
	TransportIdentity = transportID

	localClient := &Interface{Name: "local-client", Type: "LocalInterface"}
	external := &Interface{Name: "tcp-peer", Type: "TCPClientInterface"}
	LocalClientInterfaces = []*Interface{localClient}
	Interfaces = []*Interface{localClient, external}

	t.Cleanup(func() {
		Transport = prevTransport
		transportEnabled = prevTransportEnabled
		TransportIdentity = prevTransportIdentity
		pendingLocalPathRequests = prevPending
		announceTable = prevAnnounceTable
		heldAnnounces = prevHeldAnnounces
		LocalClientInterfaces = prevLocalClients
		Interfaces = prevInterfaces
		Destinations = prevDestinations
	})

	announceID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(announce): %v", err)
	}
	dst, err := NewDestination(announceID, DestinationIN, DestinationSINGLE, "test", "announce")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}
	announce := dst.Announce(nil, true, nil, nil, false)
	if announce == nil {
		t.Fatal("Announce returned nil")
	}
	if err := announce.Pack(); err != nil {
		t.Fatalf("Pack(): %v", err)
	}
	Destinations = nil
	announce.ReceivingInterface = localClient
	announce.Hops = 0

	key, ok := makeHashKey(announce.DestinationHash)
	if !ok {
		t.Fatal("announce destination hash invalid")
	}
	pendingLocalPathRequests[key] = external

	handleInboundAnnounce(announce, localClient, true)

	entry := announceTable[key]
	if entry == nil {
		t.Fatal("expected local PATH_RESPONSE announce to be queued")
	}
	if entry.AttachedInterface != external {
		t.Fatalf("queued attached interface=%p, want requester %p", entry.AttachedInterface, external)
	}
	if !entry.BlockRebroadcasts {
		t.Fatal("queued local PATH_RESPONSE announce must block rebroadcasts")
	}

	var outgoing []*Packet
	handleAnnounceRetransmit(time.Now().Add(time.Second), &outgoing)
	if len(outgoing) != 1 {
		t.Fatalf("outgoing packets=%d, want 1", len(outgoing))
	}
	packet := outgoing[0]
	if packet.Context != PacketPATH_RESPONSE {
		t.Fatalf("packet context=%d, want PATH_RESPONSE", packet.Context)
	}
	if packet.AttachedInterface != external {
		t.Fatalf("packet attached interface=%p, want requester %p", packet.AttachedInterface, external)
	}

	_ = packet.Send()
	if got := len(backend.packets); got != 1 {
		t.Fatalf("backend packets=%d, want 1", got)
	}
	if backend.packets[0].AttachedInterface != external {
		t.Fatalf("sent attached interface=%p, want requester %p", backend.packets[0].AttachedInterface, external)
	}
	if !bytes.Equal(backend.packets[0].TransportID, TransportIdentity.Hash) {
		t.Fatal("sent packet transport id mismatch")
	}
}

func TestHandleInboundAnnounce_LocalClientAnnounceQueuesImmediateTransportRebroadcast(t *testing.T) {
	resetKnownDestinationsForTest()

	prevTransport := Transport
	prevTransportEnabled := transportEnabled
	prevTransportIdentity := TransportIdentity
	prevAnnounceTable := announceTable
	prevHeldAnnounces := heldAnnounces
	prevDestinations := Destinations

	backend := &announceCaptureBackend{}
	Transport = backend
	transportEnabled = true
	announceTable = make(map[hashKey]*announceEntry)
	heldAnnounces = make(map[hashKey]*heldAnnounce)

	transportID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(transport): %v", err)
	}
	TransportIdentity = transportID

	t.Cleanup(func() {
		Transport = prevTransport
		transportEnabled = prevTransportEnabled
		TransportIdentity = prevTransportIdentity
		announceTable = prevAnnounceTable
		heldAnnounces = prevHeldAnnounces
		Destinations = prevDestinations
	})

	localClient := &Interface{Name: "local-client", Type: "LocalInterface"}

	announceID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(announce): %v", err)
	}
	dst, err := NewDestination(announceID, DestinationIN, DestinationSINGLE, "test", "announce")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}
	announce := dst.Announce([]byte("payload"), false, nil, nil, false)
	if announce == nil {
		t.Fatal("Announce returned nil")
	}
	if err := announce.Pack(); err != nil {
		t.Fatalf("Pack(): %v", err)
	}
	Destinations = nil
	announce.ReceivingInterface = localClient
	announce.Hops = 0

	handleInboundAnnounce(announce, localClient, true)

	key, ok := makeHashKey(announce.DestinationHash)
	if !ok {
		t.Fatal("announce destination hash invalid")
	}
	entry := announceTable[key]
	if entry == nil {
		t.Fatal("expected local announce to be queued")
	}
	if !entry.Next.IsZero() && entry.Next.After(time.Now().Add(100*time.Millisecond)) {
		t.Fatalf("local announce next send too late: %v", entry.Next)
	}

	var outgoing []*Packet
	handleAnnounceRetransmit(time.Now().Add(time.Second), &outgoing)
	if len(outgoing) != 1 {
		t.Fatalf("outgoing packets=%d, want 1", len(outgoing))
	}
	packet := outgoing[0]
	if packet.Context != PacketNONE {
		t.Fatalf("packet context=%d, want NONE", packet.Context)
	}
	if packet.HeaderType != HeaderType2 {
		t.Fatalf("packet header type=%d, want HEADER_2", packet.HeaderType)
	}
	if !bytes.Equal(packet.TransportID, TransportIdentity.Hash) {
		t.Fatal("packet transport id mismatch")
	}
}
