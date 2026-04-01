package rns

import (
	"testing"
	"time"
)

type pathRequestCaptureBackend struct {
	attached []*Interface
	packets  []*Packet
}

func (b *pathRequestCaptureBackend) Outbound(p *Packet) bool {
	if p != nil {
		b.attached = append(b.attached, p.AttachedInterface)
		b.packets = append(b.packets, p)
	}
	return true
}

func (b *pathRequestCaptureBackend) HopsTo(_ []byte) int {
	return PathfinderMaxHops
}

func (b *pathRequestCaptureBackend) GetFirstHopTimeout(_ []byte) time.Duration {
	return 0
}

func (b *pathRequestCaptureBackend) GetPacketRSSI(_ []byte) *float64 {
	return nil
}

func (b *pathRequestCaptureBackend) GetPacketSNR(_ []byte) *float64 {
	return nil
}

func (b *pathRequestCaptureBackend) GetPacketQ(_ []byte) *float64 {
	return nil
}

func TestPathRequestHandler_DiscoverModeForwardsUnknownPath(t *testing.T) {
	prevTransport := Transport
	prevInterfaces := Interfaces
	prevLocalClientInterfaces := LocalClientInterfaces
	prevDestinations := Destinations
	prevPathTable := pathTable
	prevDiscoveryPRTags := discoveryPRTags
	prevDiscoveryPRTagFIFO := discoveryPRTagFIFO
	prevTransportEnabled := transportEnabled

	backend := &pathRequestCaptureBackend{}
	Transport = backend
	transportEnabled = true
	Interfaces = nil
	LocalClientInterfaces = nil
	Destinations = nil
	pathTable = make(map[hashKey]*PathEntry)
	discoveryPRTags = make(map[string]struct{})
	discoveryPRTagFIFO = nil

	t.Cleanup(func() {
		Transport = prevTransport
		Interfaces = prevInterfaces
		LocalClientInterfaces = prevLocalClientInterfaces
		Destinations = prevDestinations
		pathTable = prevPathTable
		discoveryPRTags = prevDiscoveryPRTags
		discoveryPRTagFIFO = prevDiscoveryPRTagFIFO
		transportEnabled = prevTransportEnabled
	})

	attached := &Interface{Name: "ap0", Mode: InterfaceModeAccessPoint}
	peerA := &Interface{Name: "peerA"}
	peerB := &Interface{Name: "peerB"}
	Interfaces = []*Interface{attached, peerA, peerB}

	destHash := make([]byte, truncatedHashBytes)
	tag := make([]byte, truncatedHashBytes)
	for i := range destHash {
		destHash[i] = byte(i + 1)
		tag[i] = byte(0x80 + i)
	}

	pathRequestHandler(append(destHash, tag...), &Packet{ReceivingInterface: attached})

	if got := len(backend.attached); got != 2 {
		t.Fatalf("outbound requests=%d, want 2", got)
	}
	if backend.attached[0] != peerA || backend.attached[1] != peerB {
		t.Fatalf("unexpected attached interfaces: %#v", backend.attached)
	}
}

func TestPathRequestHandler_NonDiscoverModeDoesNotForwardUnknownPath(t *testing.T) {
	prevTransport := Transport
	prevInterfaces := Interfaces
	prevLocalClientInterfaces := LocalClientInterfaces
	prevDestinations := Destinations
	prevPathTable := pathTable
	prevDiscoveryPRTags := discoveryPRTags
	prevDiscoveryPRTagFIFO := discoveryPRTagFIFO
	prevTransportEnabled := transportEnabled

	backend := &pathRequestCaptureBackend{}
	Transport = backend
	transportEnabled = true
	Interfaces = nil
	LocalClientInterfaces = nil
	Destinations = nil
	pathTable = make(map[hashKey]*PathEntry)
	discoveryPRTags = make(map[string]struct{})
	discoveryPRTagFIFO = nil

	t.Cleanup(func() {
		Transport = prevTransport
		Interfaces = prevInterfaces
		LocalClientInterfaces = prevLocalClientInterfaces
		Destinations = prevDestinations
		pathTable = prevPathTable
		discoveryPRTags = prevDiscoveryPRTags
		discoveryPRTagFIFO = prevDiscoveryPRTagFIFO
		transportEnabled = prevTransportEnabled
	})

	attached := &Interface{Name: "full0", Mode: InterfaceModeFull}
	peerA := &Interface{Name: "peerA"}
	peerB := &Interface{Name: "peerB"}
	Interfaces = []*Interface{attached, peerA, peerB}

	destHash := make([]byte, truncatedHashBytes)
	tag := make([]byte, truncatedHashBytes)
	for i := range destHash {
		destHash[i] = byte(0x20 + i)
		tag[i] = byte(0xA0 + i)
	}

	pathRequestHandler(append(destHash, tag...), &Packet{ReceivingInterface: attached})

	if got := len(backend.attached); got != 0 {
		t.Fatalf("outbound requests=%d, want 0", got)
	}
}

func TestPathRequestHandler_LocalClientStillForwardsUnknownPath(t *testing.T) {
	prevTransport := Transport
	prevInterfaces := Interfaces
	prevLocalClientInterfaces := LocalClientInterfaces
	prevDestinations := Destinations
	prevPathTable := pathTable
	prevDiscoveryPRTags := discoveryPRTags
	prevDiscoveryPRTagFIFO := discoveryPRTagFIFO
	prevTransportEnabled := transportEnabled

	backend := &pathRequestCaptureBackend{}
	Transport = backend
	transportEnabled = true
	Interfaces = nil
	LocalClientInterfaces = nil
	Destinations = nil
	pathTable = make(map[hashKey]*PathEntry)
	discoveryPRTags = make(map[string]struct{})
	discoveryPRTagFIFO = nil

	t.Cleanup(func() {
		Transport = prevTransport
		Interfaces = prevInterfaces
		LocalClientInterfaces = prevLocalClientInterfaces
		Destinations = prevDestinations
		pathTable = prevPathTable
		discoveryPRTags = prevDiscoveryPRTags
		discoveryPRTagFIFO = prevDiscoveryPRTagFIFO
		transportEnabled = prevTransportEnabled
	})

	localClient := &Interface{Name: "client0", Mode: InterfaceModeFull}
	peerA := &Interface{Name: "peerA"}
	peerB := &Interface{Name: "peerB"}
	Interfaces = []*Interface{localClient, peerA, peerB}
	LocalClientInterfaces = []*Interface{localClient}

	destHash := make([]byte, truncatedHashBytes)
	tag := make([]byte, truncatedHashBytes)
	for i := range destHash {
		destHash[i] = byte(0x40 + i)
		tag[i] = byte(0xC0 + i)
	}

	pathRequestHandler(append(destHash, tag...), &Packet{ReceivingInterface: localClient})

	if got := len(backend.attached); got != 2 {
		t.Fatalf("outbound requests=%d, want 2", got)
	}
	if backend.attached[0] != peerA || backend.attached[1] != peerB {
		t.Fatalf("unexpected attached interfaces: %#v", backend.attached)
	}
}

func TestRequestPathOnInterface_UsesProvidedTagAndRecursive(t *testing.T) {
	prevTransport := Transport
	prevTransportEnabled := transportEnabled
	prevLastPathRequest := lastPathRequest

	backend := &pathRequestCaptureBackend{}
	Transport = backend
	transportEnabled = false
	lastPathRequest = make(map[hashKey]time.Time)

	t.Cleanup(func() {
		Transport = prevTransport
		transportEnabled = prevTransportEnabled
		lastPathRequest = prevLastPathRequest
	})

	destHash := []byte("request-path-api")
	tag := []byte("custom-path-tag!")
	ifc := &Interface{Name: "peer0", Bitrate: 1000000, AnnounceCap: 0.5}

	if sent := RequestPathOnInterface(destHash, ifc, tag, true); !sent {
		t.Fatal("RequestPathOnInterface() = false, want true")
	}
	if got := len(backend.packets); got != 1 {
		t.Fatalf("outbound packets=%d, want 1", got)
	}
	packet := backend.packets[0]
	if packet.AttachedInterface != ifc {
		t.Fatal("request was not sent on specified interface")
	}
	wantPayload := append(append([]byte(nil), destHash...), tag...)
	if string(packet.Data) != string(wantPayload) {
		t.Fatalf("payload=%x, want %x", packet.Data, wantPayload)
	}
	if ifc.AnnounceAllowedAt().IsZero() {
		t.Fatal("recursive request did not update announce_allowed_at")
	}
}
