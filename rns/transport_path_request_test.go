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
	prevTransportIdentity := TransportIdentity

	backend := &pathRequestCaptureBackend{}
	Transport = backend
	transportEnabled = true
	Interfaces = nil
	LocalClientInterfaces = nil
	Destinations = nil
	pathTable = make(map[hashKey]*PathEntry)
	discoveryPRTags = make(map[string]struct{})
	discoveryPRTagFIFO = nil
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
		transportEnabled = prevTransportEnabled
		TransportIdentity = prevTransportIdentity
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
	prevTransportIdentity := TransportIdentity

	backend := &pathRequestCaptureBackend{}
	Transport = backend
	transportEnabled = true
	Interfaces = nil
	LocalClientInterfaces = nil
	Destinations = nil
	pathTable = make(map[hashKey]*PathEntry)
	discoveryPRTags = make(map[string]struct{})
	discoveryPRTagFIFO = nil
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
		transportEnabled = prevTransportEnabled
		TransportIdentity = prevTransportIdentity
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
	prevTransportIdentity := TransportIdentity

	backend := &pathRequestCaptureBackend{}
	Transport = backend
	transportEnabled = true
	Interfaces = nil
	LocalClientInterfaces = nil
	Destinations = nil
	pathTable = make(map[hashKey]*PathEntry)
	discoveryPRTags = make(map[string]struct{})
	discoveryPRTagFIFO = nil
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
		transportEnabled = prevTransportEnabled
		TransportIdentity = prevTransportIdentity
	})

	detachedShared := &Interface{Name: "Shared Instance[default]", Type: "LocalInterface", LocalIsSharedInstance: true}
	localClient := &Interface{Name: "client0", Type: "LocalInterface", Mode: InterfaceModeFull, Parent: detachedShared}
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

func TestPathRequestHandler_ExternalForwardsUnknownPathToLocalClients(t *testing.T) {
	prevTransport := Transport
	prevInterfaces := Interfaces
	prevLocalClientInterfaces := LocalClientInterfaces
	prevDestinations := Destinations
	prevPathTable := pathTable
	prevDiscoveryPRTags := discoveryPRTags
	prevDiscoveryPRTagFIFO := discoveryPRTagFIFO
	prevTransportEnabled := transportEnabled
	prevTransportIdentity := TransportIdentity

	backend := &pathRequestCaptureBackend{}
	Transport = backend
	transportEnabled = true
	Interfaces = nil
	LocalClientInterfaces = nil
	Destinations = nil
	pathTable = make(map[hashKey]*PathEntry)
	discoveryPRTags = make(map[string]struct{})
	discoveryPRTagFIFO = nil
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
		transportEnabled = prevTransportEnabled
		TransportIdentity = prevTransportIdentity
	})

	external := &Interface{Name: "peer0", Mode: InterfaceModeFull}
	localClient := &Interface{Name: "client0", Mode: InterfaceModeFull}
	Interfaces = []*Interface{external, localClient}
	LocalClientInterfaces = []*Interface{localClient}

	destHash := make([]byte, truncatedHashBytes)
	tag := make([]byte, truncatedHashBytes)
	for i := range destHash {
		destHash[i] = byte(0x50 + i)
		tag[i] = byte(0xD0 + i)
	}

	pathRequestHandler(append(destHash, tag...), &Packet{ReceivingInterface: external})

	if got := len(backend.attached); got != 1 {
		t.Fatalf("outbound requests=%d, want 1", got)
	}
	if backend.attached[0] != localClient {
		t.Fatalf("unexpected attached interface: %#v", backend.attached[0])
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

	RequestPath(destHash, ifc, tag, true)
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
	if ifc.AnnounceAllowedAtTime().IsZero() {
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
	key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(destHash)
	if !ok {
		t.Fatal("hash key conversion failed")
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

	RequestPath(destHash, ifc, tag, false)
	if got := len(backend.packets); got != 1 {
		t.Fatalf("outbound packets=%d, want 1", got)
	}
	if backend.packets[0].AttachedInterface != ifc {
		t.Fatal("request was not sent on specified interface")
	}
}

func TestCullReverseAndLinkTables_MarksPathUnresponsiveOnPendingLinkTimeout(t *testing.T) {
	prevTransport := Transport
	prevTransportEnabled := transportEnabled
	prevInterfaces := Interfaces
	prevPathTable := pathTable
	prevPathStates := pathStates
	prevLastPathRequest := lastPathRequest
	prevLinkTable := linkTable

	backend := &pathRequestCaptureBackend{}
	Transport = backend
	transportEnabled = true
	Interfaces = nil
	pathTable = make(map[hashKey]*PathEntry)
	pathStates = make(map[hashKey]uint8)
	lastPathRequest = make(map[hashKey]time.Time)
	linkTable = make(map[hashKey]*linkEntry)
	TablesLastCulled = time.Time{}
	LinksLastChecked = time.Now()
	ReceiptsLast = time.Now()
	AnnLast = time.Now()
	LastCacheCleaned = time.Now()
	InterfaceLastJobs = time.Now()
	blackholeLastChecked = time.Now()
	pendingPRsLastChecked = time.Now()

	t.Cleanup(func() {
		Transport = prevTransport
		transportEnabled = prevTransportEnabled
		Interfaces = prevInterfaces
		pathTable = prevPathTable
		pathStates = prevPathStates
		lastPathRequest = prevLastPathRequest
		linkTable = prevLinkTable
	})

	destHash := make([]byte, truncatedHashBytes)
	for i := range destHash {
		destHash[i] = byte(0x70 + i)
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
		t.Fatal("hash key conversion failed")
	}

	recvIfc := &Interface{Name: "tcp-peer", Mode: InterfaceModeFull}
	peerIfc := &Interface{Name: "peer-a", Mode: InterfaceModeFull}
	Interfaces = []*Interface{recvIfc, peerIfc}
	pathTable[key] = &PathEntry{
		NextHop:       append([]byte(nil), destHash...),
		RecvInterface: recvIfc,
		Hops:          1,
		Timestamp:     time.Now(),
		ExpiresAt:     time.Now().Add(time.Hour),
	}
	linkTable[key] = &linkEntry{
		Timestamp:         time.Now(),
		NextHopID:         append([]byte(nil), destHash...),
		NextHopInterface:  recvIfc,
		ReceivedInterface: recvIfc,
		RemainingHops:     1,
		Hops:              1,
		DestinationHash:   append([]byte(nil), destHash...),
		Validated:         false,
		ProofTimeout:      time.Now().Add(-time.Second),
	}

	Jobs()
	if !PathIsUnresponsive(destHash) {
		t.Fatal("expected path to be marked unresponsive after pending link timeout")
	}
	if got := len(backend.packets); got != 1 {
		t.Fatalf("outbound path requests=%d, want 1", got)
	}
	if backend.attached[0] != peerIfc {
		t.Fatalf("request sent on %#v, want %#v", backend.attached[0], peerIfc)
	}
	if _, exists := linkTable[key]; exists {
		t.Fatal("expected expired link to be removed")
	}
}

func TestHandlePendingAndActiveLinks_ThrottlesClosedPendingLinkRediscovery(t *testing.T) {
	prevTransportEnabled := transportEnabled
	prevOwner := Owner
	prevPathTable := pathTable
	prevPendingLinks := PendingLinks
	prevLastPathRequest := lastPathRequest

	transportEnabled = false
	Owner = nil
	pathTable = make(map[hashKey]*PathEntry)
	lastPathRequest = make(map[hashKey]time.Time)
	PendingLinks = nil

	t.Cleanup(func() {
		transportEnabled = prevTransportEnabled
		Owner = prevOwner
		pathTable = prevPathTable
		PendingLinks = prevPendingLinks
		lastPathRequest = prevLastPathRequest
	})

	destHash := make([]byte, truncatedHashBytes)
	for i := range destHash {
		destHash[i] = byte(0x90 + i)
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
		t.Fatal("hash key conversion failed")
	}

	recvIfc := &Interface{Name: "tcp-peer", Mode: InterfaceModeFull}
	pathTable[key] = &PathEntry{
		NextHop:       append([]byte(nil), destHash...),
		RecvInterface: recvIfc,
		Hops:          1,
		Timestamp:     time.Now(),
		ExpiresAt:     time.Now().Add(time.Hour),
	}
	lastPathRequest[key] = time.Now()
	PendingLinks = []*Link{{
		Status:      LinkClosed,
		destination: &Destination{hash: append([]byte(nil), destHash...)},
	}}

	pathReqs := make(map[hashKey]*Interface)
	func() {
		linkMu.Lock()
		defer linkMu.Unlock()

		now := time.Now()

		filterPending := PendingLinks[:0]
		for _, link := range PendingLinks {
			if link == nil {
				continue
			}
			if link.Status == LinkClosed {
				if !TransportEnabled() && link.destination != nil {
					destHash := link.destination.Hash()
					if ExpirePath(destHash) {
						if Owner == nil || !Owner.IsConnectedToSharedInstance {
							if key, ok := func(hash []byte) (hashKey, bool) {
								if len(hash) < truncatedHashBytes {
									return hashKey{}, false
								}
								var key hashKey
								copy(key[:], hash[:truncatedHashBytes])
								return key, true
							}(destHash); ok {
								pathRequestMu.Lock()
								lastRequest, hasLast := lastPathRequest[key]
								pathRequestMu.Unlock()
								if !hasLast || lastRequest.IsZero() || now.Sub(lastRequest) > pathRequestMinInterval {
									if pathReqs != nil {
										if _, exists := pathReqs[key]; !exists {
											pathReqs[key] = nil
										}
									}
								}
							}
						}
					}
				}
				continue
			}
			if link.destination != nil && len(link.destination.Hash()) >= truncatedHashBytes {
				if key, ok := func(hash []byte) (hashKey, bool) {
					if len(hash) < truncatedHashBytes {
						return hashKey{}, false
					}
					var key hashKey
					copy(key[:], hash[:truncatedHashBytes])
					return key, true
				}(link.destination.Hash()); ok {
					pathReqs[key] = nil
				}
			}
			filterPending = append(filterPending, link)
		}
		PendingLinks = filterPending

		filterActive := ActiveLinks[:0]
		for _, link := range ActiveLinks {
			if link == nil || link.Status == LinkClosed {
				continue
			}
			if !link.lastInbound.IsZero() && now.Sub(link.lastInbound) > link.StaleTime {
				go link.Teardown()
				continue
			}
			if link.destination != nil && !HasPath(link.destination.Hash()) {
				if key, ok := func(hash []byte) (hashKey, bool) {
					if len(hash) < truncatedHashBytes {
						return hashKey{}, false
					}
					var key hashKey
					copy(key[:], hash[:truncatedHashBytes])
					return key, true
				}(link.destination.Hash()); ok {
					pathReqs[key] = nil
				}
			}
			filterActive = append(filterActive, link)
		}
		ActiveLinks = filterActive
	}()

	if got := len(pathReqs); got != 0 {
		t.Fatalf("queued path requests=%d, want 0", got)
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
	detachedShared := &Interface{Name: "Shared Instance[default]", Type: "LocalInterface", LocalIsSharedInstance: true}
	localClient := &Interface{Name: "client0", Type: "LocalInterface", Parent: detachedShared}
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

	cases := []struct {
		name      string
		attached  *Interface
		recv      *Interface
		fromLocal bool
	}{
		{
			name:      "request from local client",
			attached:  localClient,
			recv:      external,
			fromLocal: true,
		},
		{
			name:      "destination on local client",
			attached:  external,
			recv:      localClient,
			fromLocal: false,
		},
		{
			name:      "destination on detached local client",
			attached:  external,
			recv:      detachedLocalClient,
			fromLocal: false,
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

			pathRequest(announce.DestinationHash, tc.fromLocal, tc.attached, nil, bytesRepeat(0xA0, truncatedHashBytes))

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
