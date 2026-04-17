package rns

import (
	"os"
	"path/filepath"
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

	if sent := RequestPath(destHash, ifc, tag, true); !sent {
		t.Fatal("RequestPath() = false, want true")
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

func TestRequestPathOnInterface_SendsEvenWhenPathExists(t *testing.T) {
	prevTransport := Transport
	prevTransportEnabled := transportEnabled
	prevPathTable := pathTable

	backend := &pathRequestCaptureBackend{}
	Transport = backend
	transportEnabled = false
	pathTable = make(map[hashKey]*PathEntry)

	t.Cleanup(func() {
		Transport = prevTransport
		transportEnabled = prevTransportEnabled
		pathTable = prevPathTable
	})

	destHash := []byte("request-path-api")
	key, ok := makeHashKey(destHash)
	if !ok {
		t.Fatal("makeHashKey() = false")
	}
	pathTable[key] = &PathEntry{
		NextHop:       []byte("next-hop-desthash"),
		RecvInterface: &Interface{Name: "known0"},
		Hops:          1,
		Timestamp:     time.Now(),
		ExpiresAt:     time.Now().Add(time.Minute),
	}

	tag := []byte("refresh-path-tag")
	ifc := &Interface{Name: "peer0"}

	if sent := RequestPath(destHash, ifc, tag, false); !sent {
		t.Fatal("RequestPath() = false, want true")
	}
	if got := len(backend.packets); got != 1 {
		t.Fatalf("outbound packets=%d, want 1", got)
	}
	if backend.packets[0].AttachedInterface != ifc {
		t.Fatal("request was not sent on specified interface")
	}
}

func TestAnswerPathRequest_LocalClientCasesQueueImmediateResponse(t *testing.T) {
	resetKnownDestinationsForTest()

	prevOwner := Owner
	prevTransport := Transport
	prevTransportEnabled := transportEnabled
	prevTransportIdentity := TransportIdentity
	prevInterfaces := Interfaces
	prevLocalClients := LocalClientInterfaces
	prevDestinations := Destinations
	prevPathTable := pathTable
	prevAnnounceTable := announceTable
	prevHeldAnnounces := heldAnnounces

	cacheRoot := t.TempDir()
	Owner = &Reticulum{
		StoragePath: filepath.Join(cacheRoot, "storage"),
		CachePath:   filepath.Join(cacheRoot, "storage", "cache"),
	}
	if err := os.MkdirAll(filepath.Join(Owner.CachePath, "announces"), 0o755); err != nil {
		t.Fatalf("MkdirAll(announces): %v", err)
	}

	backend := &pathRequestCaptureBackend{}
	Transport = backend
	transportEnabled = true
	pathTable = make(map[hashKey]*PathEntry)
	announceTable = make(map[hashKey]*announceEntry)
	heldAnnounces = make(map[hashKey]*heldAnnounce)

	transportID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(transport): %v", err)
	}
	TransportIdentity = transportID

	external := &Interface{Name: "peer0", Type: "TCPClientInterface"}
	localClient := &Interface{Name: "client0", Type: "LocalInterface"}
	detachedShared := &Interface{Name: "Shared Instance[default]", Type: "LocalInterface"}
	detachedLocalClient := &Interface{Name: "client-detached", Type: "LocalInterface", Parent: detachedShared}
	Interfaces = []*Interface{external, localClient}
	LocalClientInterfaces = []*Interface{localClient}
	Owner.SharedInstanceInterface = detachedShared

	t.Cleanup(func() {
		Owner = prevOwner
		Transport = prevTransport
		transportEnabled = prevTransportEnabled
		TransportIdentity = prevTransportIdentity
		Interfaces = prevInterfaces
		LocalClientInterfaces = prevLocalClients
		Destinations = prevDestinations
		pathTable = prevPathTable
		announceTable = prevAnnounceTable
		heldAnnounces = prevHeldAnnounces
	})

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
		t.Fatal("Announce() returned nil")
	}
	if err := announce.Pack(); err != nil {
		t.Fatalf("Pack(): %v", err)
	}
	Destinations = nil
	announce.ReceivingInterface = external
	announce.Hops = 1
	Cache(announce, true)

	key, ok := makeHashKey(announce.DestinationHash)
	if !ok {
		t.Fatal("announce destination hash invalid")
	}

	cases := []struct {
		name     string
		attached *Interface
		recv     *Interface
	}{
		{
			name:     "request from local client",
			attached: localClient,
			recv:     external,
		},
		{
			name:     "destination on local client",
			attached: external,
			recv:     localClient,
		},
		{
			name:     "destination on detached local client",
			attached: external,
			recv:     detachedLocalClient,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pathTable[key] = &PathEntry{
				NextHop:       copyBytes(announce.DestinationHash),
				RecvInterface: tc.recv,
				Hops:          1,
				Timestamp:     time.Now(),
				ExpiresAt:     time.Now().Add(time.Minute),
				PacketHash:    copyBytes(announce.PacketHash),
			}
			announceTable = make(map[hashKey]*announceEntry)
			heldAnnounces = make(map[hashKey]*heldAnnounce)

			if ok := answerPathRequest(announce.DestinationHash, tc.attached, nil, bytesRepeat(0xA0, truncatedHashBytes)); !ok {
				t.Fatal("answerPathRequest() = false, want true")
			}

			entry := announceTable[key]
			if entry == nil {
				t.Fatal("expected queued PATH_RESPONSE announce")
			}
			if entry.AttachedInterface != tc.attached {
				t.Fatalf("attached interface=%p, want %p", entry.AttachedInterface, tc.attached)
			}
			if entry.Next.After(time.Now().Add(100 * time.Millisecond)) {
				t.Fatalf("queued response was not immediate: next=%v", entry.Next)
			}
		})
	}
}
