package rns

import (
	"bytes"
	"crypto/ed25519"
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

func buildAnnounceWithRandomBlob(t *testing.T, dst *Destination, appData []byte, pathResponse bool, randomBlob []byte) *Packet {
	t.Helper()
	if dst == nil {
		t.Fatal("destination is nil")
	}
	if len(randomBlob) != announceRandomHashLen {
		t.Fatalf("random blob length=%d, want %d", len(randomBlob), announceRandomHashLen)
	}
	if dst.identity == nil {
		t.Fatal("destination identity is nil")
	}

	publicKey := dst.identity.GetPublicKey()
	nameHash := copyBytes(dst.nameHash)
	destinationHash := copyBytes(dst.Hash())

	signed := make([]byte, 0, len(destinationHash)+len(publicKey)+len(nameHash)+len(randomBlob)+len(appData))
	signed = append(signed, destinationHash...)
	signed = append(signed, publicKey...)
	signed = append(signed, nameHash...)
	signed = append(signed, randomBlob...)
	signed = append(signed, appData...)

	signature, err := dst.identity.Sign(signed)
	if err != nil {
		t.Fatalf("Sign(): %v", err)
	}
	if len(signature) != ed25519.SignatureSize {
		t.Fatalf("signature length=%d, want %d", len(signature), ed25519.SignatureSize)
	}

	data := make([]byte, 0, len(publicKey)+len(nameHash)+len(randomBlob)+len(signature)+len(appData))
	data = append(data, publicKey...)
	data = append(data, nameHash...)
	data = append(data, randomBlob...)
	data = append(data, signature...)
	data = append(data, appData...)

	packet := NewPacket(
		dst,
		data,
		WithPacketType(PacketANNOUNCE),
		WithPacketContext(func() byte {
			if pathResponse {
				return PacketPATH_RESPONSE
			}
			return PacketNONE
		}()),
		WithHeaderType(HeaderType1),
		WithTransportType(Broadcast),
		WithContextFlag(FlagUnset),
		WithoutReceipt(),
	)
	if packet == nil {
		t.Fatal("NewPacket returned nil")
	}
	if err := packet.Pack(); err != nil {
		t.Fatalf("Pack(): %v", err)
	}
	return packet
}

func TestHandleInboundAnnounce_LocalClientPathResponseRequiresPendingRequest(t *testing.T) {
	resetKnownDestinationsForTest()

	prevTransportEnabled := transportEnabled
	prevPending := pendingLocalPathRequests
	prevAnnounceTable := announceTable
	prevHeldAnnounces := heldAnnounces
	prevLocalClients := LocalClientInterfaces
	prevInterfaces := Interfaces
	prevDestinations := Destinations

	transportEnabled = true
	pendingLocalPathRequests = make(map[hashKey]*Interface)
	announceTable = make(map[hashKey]*announceEntry)
	heldAnnounces = make(map[hashKey]*heldAnnounce)

	localClient := &Interface{Name: "local-client", Type: "LocalInterface"}
	LocalClientInterfaces = []*Interface{localClient}
	Interfaces = []*Interface{localClient}

	t.Cleanup(func() {
		transportEnabled = prevTransportEnabled
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

	handleInboundAnnounce(announce, localClient, true)

	if entry := announceTable[key]; entry != nil {
		t.Fatalf("local PATH_RESPONSE without pending request unexpectedly queued: %+v", entry)
	}
}

func TestHandleInboundAnnounce_LocalClientPathResponseQueuesImmediateAnnounceWhenPending(t *testing.T) {
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
	if entry.AttachedInterface != nil {
		t.Fatalf("queued attached interface=%p, want nil for Python parity", entry.AttachedInterface)
	}
	if entry.BlockRebroadcasts {
		t.Fatal("queued local PATH_RESPONSE announce must not force PATH_RESPONSE context")
	}
	if entry.Retries != pathfinderRetryLimit {
		t.Fatalf("queued retries=%d, want %d", entry.Retries, pathfinderRetryLimit)
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
	if packet.AttachedInterface != nil {
		t.Fatalf("packet attached interface=%p, want nil", packet.AttachedInterface)
	}

	_ = packet.Send()
	if got := len(backend.packets); got != 1 {
		t.Fatalf("backend packets=%d, want 1", got)
	}
	if backend.packets[0].AttachedInterface != nil {
		t.Fatalf("sent attached interface=%p, want nil", backend.packets[0].AttachedInterface)
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
	if entry.Retries != pathfinderRetryLimit {
		t.Fatalf("queued retries=%d, want %d", entry.Retries, pathfinderRetryLimit)
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

func TestJobs_WaitsForInboundReadLockToProcessQueuedAnnounce(t *testing.T) {
	resetKnownDestinationsForTest()

	prevTransport := Transport
	prevTransportEnabled := transportEnabled
	prevTransportIdentity := TransportIdentity
	prevAnnounceTable := announceTable
	prevHeldAnnounces := heldAnnounces
	prevInterfaces := Interfaces
	prevDestinations := Destinations
	prevAnnLast := AnnLast

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
	AnnLast = time.Now().Add(-2 * time.Second)

	localClient := &Interface{Name: "local-client", Type: "LocalInterface"}
	external := &Interface{Name: "tcp-peer", Type: "TCPClientInterface", OUT: true}
	Interfaces = []*Interface{localClient, external}

	t.Cleanup(func() {
		Transport = prevTransport
		transportEnabled = prevTransportEnabled
		TransportIdentity = prevTransportIdentity
		announceTable = prevAnnounceTable
		heldAnnounces = prevHeldAnnounces
		Interfaces = prevInterfaces
		Destinations = prevDestinations
		AnnLast = prevAnnLast
	})

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

	jobsMu.RLock()
	done := make(chan struct{})
	go func() {
		defer close(done)
		Jobs()
	}()

	select {
	case <-done:
		t.Fatal("Jobs returned while transport read lock was still held")
	case <-time.After(50 * time.Millisecond):
	}

	jobsMu.RUnlock()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Jobs did not complete after releasing read lock")
	}

	if len(backend.packets) == 0 {
		t.Fatal("expected queued announce to be transmitted once Jobs acquired the lock")
	}
}

func TestHandleAnnounceRetransmit_ReinsertsHeldAnnounceAfterPathResponse(t *testing.T) {
	resetKnownDestinationsForTest()

	prevTransportIdentity := TransportIdentity
	prevAnnounceTable := announceTable
	prevHeldAnnounces := heldAnnounces

	announceTable = make(map[hashKey]*announceEntry)
	heldAnnounces = make(map[hashKey]*heldAnnounce)

	transportID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(transport): %v", err)
	}
	TransportIdentity = transportID

	t.Cleanup(func() {
		TransportIdentity = prevTransportIdentity
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
	announce := dst.Announce([]byte("payload"), false, nil, nil, false)
	if announce == nil {
		t.Fatal("Announce returned nil")
	}
	if err := announce.Pack(); err != nil {
		t.Fatalf("Pack(): %v", err)
	}

	key, ok := makeHashKey(announce.DestinationHash)
	if !ok {
		t.Fatal("announce destination hash invalid")
	}

	heldEntry := &announceEntry{
		Packet:            cloneAnnouncePacket(announce),
		Next:              time.Now().Add(time.Minute),
		Timestamp:         time.Now(),
		Expires:           time.Now().Add(time.Hour),
		AttachedInterface: &Interface{Name: "held-if"},
	}
	pathResponseEntry := &announceEntry{
		Packet:            cloneAnnouncePacket(announce),
		Next:              time.Now().Add(-time.Second),
		Timestamp:         time.Now(),
		Expires:           time.Now().Add(time.Hour),
		BlockRebroadcasts: true,
		AttachedInterface: &Interface{Name: "resp-if"},
	}

	announceTable[key] = pathResponseEntry
	heldAnnounces[key] = &heldAnnounce{Entry: heldEntry}

	var outgoing []*Packet
	handleAnnounceRetransmit(time.Now(), &outgoing)

	if len(outgoing) != 1 {
		t.Fatalf("outgoing packets=%d, want 1", len(outgoing))
	}
	if outgoing[0].Context != PacketPATH_RESPONSE {
		t.Fatalf("packet context=%d, want PATH_RESPONSE", outgoing[0].Context)
	}
	if announceTable[key] != heldEntry {
		t.Fatal("held announce was not restored after path response retransmit")
	}
	if _, exists := heldAnnounces[key]; exists {
		t.Fatal("held announce entry was not cleared")
	}
}

func TestHandleAnnounceRetransmit_AllowsSecondRetryForNonLocalAnnounce(t *testing.T) {
	resetKnownDestinationsForTest()

	prevAnnounceTable := announceTable
	prevHeldAnnounces := heldAnnounces

	announceTable = make(map[hashKey]*announceEntry)
	heldAnnounces = make(map[hashKey]*heldAnnounce)

	t.Cleanup(func() {
		announceTable = prevAnnounceTable
		heldAnnounces = prevHeldAnnounces
	})

	id, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	dst, err := NewDestination(id, DestinationIN, DestinationSINGLE, "test", "announce")
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

	key, ok := makeHashKey(announce.DestinationHash)
	if !ok {
		t.Fatal("announce destination hash invalid")
	}

	announceTable[key] = &announceEntry{
		Packet:    cloneAnnouncePacket(announce),
		Next:      time.Now().Add(-time.Second),
		Timestamp: time.Now(),
		Expires:   time.Now().Add(time.Hour),
	}

	var first []*Packet
	handleAnnounceRetransmit(time.Now(), &first)
	if len(first) != 1 {
		t.Fatalf("first outgoing packets=%d, want 1", len(first))
	}
	entry := announceTable[key]
	if entry == nil {
		t.Fatal("announce entry disappeared after first retry")
	}
	if entry.Retries != 1 {
		t.Fatalf("retries after first send=%d, want 1", entry.Retries)
	}

	var second []*Packet
	handleAnnounceRetransmit(entry.Next.Add(time.Millisecond), &second)
	if len(second) != 1 {
		t.Fatalf("second outgoing packets=%d, want 1", len(second))
	}
}

func TestOutbound_LocalAnnounceDoesNotQueueTransportRetransmit(t *testing.T) {
	resetKnownDestinationsForTest()

	prevAnnounceTable := announceTable
	prevInterfaces := Interfaces
	prevDestinations := Destinations

	announceTable = make(map[hashKey]*announceEntry)
	Interfaces = []*Interface{
		{
			Name:        "if0",
			Type:        "Test",
			OUT:         true,
			AnnounceCap: 1.0,
		},
	}

	t.Cleanup(func() {
		announceTable = prevAnnounceTable
		Interfaces = prevInterfaces
		Destinations = prevDestinations
	})

	id, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	dst, err := NewDestination(id, DestinationIN, DestinationSINGLE, "test", "announce")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}

	announce := dst.Announce([]byte("payload"), false, nil, nil, false)
	if announce == nil {
		t.Fatal("Announce returned nil")
	}

	if !Outbound(announce) {
		t.Fatal("Outbound returned false")
	}

	key, ok := makeHashKey(dst.Hash())
	if !ok {
		t.Fatal("destination hash invalid")
	}
	if entry := announceTable[key]; entry != nil {
		t.Fatalf("local outbound announce unexpectedly queued: %+v", entry)
	}
}

func TestHandleInboundAnnounce_DuplicateExternalReturnForLocalClientPathIsIgnored(t *testing.T) {
	resetKnownDestinationsForTest()

	prevTransportEnabled := transportEnabled
	prevTransportIdentity := TransportIdentity
	prevAnnounceTable := announceTable
	prevHeldAnnounces := heldAnnounces
	prevPathTable := pathTable
	prevDestinations := Destinations
	prevInterfaces := Interfaces
	prevLocalClients := LocalClientInterfaces
	prevHandlers := announceHandlers

	transportEnabled = true
	announceTable = make(map[hashKey]*announceEntry)
	heldAnnounces = make(map[hashKey]*heldAnnounce)
	pathTable = make(map[hashKey]*PathEntry)
	Destinations = nil
	announceHandlers = nil

	transportID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(transport): %v", err)
	}
	TransportIdentity = transportID

	localClient := &Interface{Name: "local-client", Type: "LocalInterface"}
	external := &Interface{Name: "tcp-peer", Type: "TCPClientInterface", OUT: true}
	LocalClientInterfaces = []*Interface{localClient}
	Interfaces = []*Interface{localClient, external}

	t.Cleanup(func() {
		transportEnabled = prevTransportEnabled
		TransportIdentity = prevTransportIdentity
		announceTable = prevAnnounceTable
		heldAnnounces = prevHeldAnnounces
		pathTable = prevPathTable
		Destinations = prevDestinations
		Interfaces = prevInterfaces
		LocalClientInterfaces = prevLocalClients
		announceHandlers = prevHandlers
	})

	handler := &legacyAnnounceHandler{}
	RegisterAnnounceHandler(handler)

	announceID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(announce): %v", err)
	}
	dst, err := NewDestination(announceID, DestinationIN, DestinationSINGLE, "test", "announce")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}
	localAnnounce := dst.Announce([]byte("payload"), false, nil, nil, false)
	if localAnnounce == nil {
		t.Fatal("Announce returned nil")
	}
	if err := localAnnounce.Pack(); err != nil {
		t.Fatalf("Pack(): %v", err)
	}
	Destinations = nil
	localAnnounce.ReceivingInterface = localClient
	localAnnounce.Hops = 0

	handleInboundAnnounce(localAnnounce, localClient, true)

	key, ok := makeHashKey(localAnnounce.DestinationHash)
	if !ok {
		t.Fatal("announce destination hash invalid")
	}
	entry := pathTable[key]
	if entry == nil {
		t.Fatal("expected local-client announce to populate path table")
	}
	if entry.RecvInterface != localClient {
		t.Fatalf("path recv interface=%v, want local client", entry.RecvInterface)
	}
	if entry.Hops != 0 {
		t.Fatalf("path hops=%d, want 0 for local client path", entry.Hops)
	}
	if handler.calls != 1 {
		t.Fatalf("handler calls after local announce=%d, want 1", handler.calls)
	}

	announceTable = make(map[hashKey]*announceEntry)

	returnedAnnounce := buildTransportAnnouncePacket(localAnnounce, nil, false)
	if returnedAnnounce == nil {
		t.Fatal("buildTransportAnnouncePacket returned nil")
	}
	if err := returnedAnnounce.Pack(); err != nil {
		t.Fatalf("Pack(returned announce): %v", err)
	}
	returnedAnnounce.ReceivingInterface = external
	returnedAnnounce.Hops = 1

	handleInboundAnnounce(returnedAnnounce, external, false)

	if got := len(announceTable); got != 0 {
		t.Fatalf("duplicate external return queued %d announce(s), want 0", got)
	}
	entry = pathTable[key]
	if entry == nil {
		t.Fatal("path entry disappeared after duplicate external return")
	}
	if entry.RecvInterface != localClient {
		t.Fatalf("path recv interface after duplicate=%v, want local client", entry.RecvInterface)
	}
	if entry.Hops != 0 {
		t.Fatalf("path hops after duplicate=%d, want 0", entry.Hops)
	}
	if handler.calls != 1 {
		t.Fatalf("handler calls after duplicate=%d, want still 1", handler.calls)
	}
}

func TestHandleInboundAnnounce_LocalPathResponseThenNormalAnnounceKeepsSingleCallback(t *testing.T) {
	resetKnownDestinationsForTest()

	prevTransportEnabled := transportEnabled
	prevTransportIdentity := TransportIdentity
	prevAnnounceTable := announceTable
	prevHeldAnnounces := heldAnnounces
	prevPathTable := pathTable
	prevDestinations := Destinations
	prevInterfaces := Interfaces
	prevLocalClients := LocalClientInterfaces
	prevHandlers := announceHandlers

	transportEnabled = true
	announceTable = make(map[hashKey]*announceEntry)
	heldAnnounces = make(map[hashKey]*heldAnnounce)
	pathTable = make(map[hashKey]*PathEntry)
	Destinations = nil
	announceHandlers = nil

	transportID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(transport): %v", err)
	}
	TransportIdentity = transportID

	localClient := &Interface{Name: "local-client", Type: "LocalInterface"}
	external := &Interface{Name: "tcp-peer", Type: "TCPClientInterface", OUT: true}
	LocalClientInterfaces = []*Interface{localClient}
	Interfaces = []*Interface{localClient, external}

	t.Cleanup(func() {
		transportEnabled = prevTransportEnabled
		TransportIdentity = prevTransportIdentity
		announceTable = prevAnnounceTable
		heldAnnounces = prevHeldAnnounces
		pathTable = prevPathTable
		Destinations = prevDestinations
		Interfaces = prevInterfaces
		LocalClientInterfaces = prevLocalClients
		announceHandlers = prevHandlers
	})

	handler := &legacyAnnounceHandler{}
	RegisterAnnounceHandler(handler)

	announceID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(announce): %v", err)
	}
	dst, err := NewDestination(announceID, DestinationIN, DestinationSINGLE, "test", "announce")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}

	autoPathResponse := buildAnnounceWithRandomBlob(t, dst, nil, true, bytes.Repeat([]byte{0x00}, announceRandomHashLen))
	autoPathResponse.ReceivingInterface = localClient
	autoPathResponse.Hops = 0
	Destinations = nil
	handleInboundAnnounce(autoPathResponse, localClient, true)

	normalAnnounce := buildAnnounceWithRandomBlob(t, dst, []byte("payload"), false, bytes.Repeat([]byte{0xFF}, announceRandomHashLen))
	normalAnnounce.ReceivingInterface = localClient
	normalAnnounce.Hops = 0
	handleInboundAnnounce(normalAnnounce, localClient, true)

	key, ok := makeHashKey(normalAnnounce.DestinationHash)
	if !ok {
		t.Fatal("announce destination hash invalid")
	}
	entry := pathTable[key]
	if entry == nil {
		t.Fatal("expected normal announce to populate path table")
	}
	if entry.Hops != 0 {
		t.Fatalf("path hops after normal announce=%d, want 0", entry.Hops)
	}
	if len(entry.RandomBlobs) < 1 {
		t.Fatal("expected at least one random blob in path table")
	}
	if handler.calls != 1 {
		t.Fatalf("handler calls after path response + normal announce=%d, want 1", handler.calls)
	}

	announceTable = make(map[hashKey]*announceEntry)

	returnedAnnounce := buildTransportAnnouncePacket(normalAnnounce, nil, false)
	if returnedAnnounce == nil {
		t.Fatal("buildTransportAnnouncePacket returned nil")
	}
	if err := returnedAnnounce.Pack(); err != nil {
		t.Fatalf("Pack(returned announce): %v", err)
	}
	returnedAnnounce.ReceivingInterface = external
	returnedAnnounce.Hops = 2

	handleInboundAnnounce(returnedAnnounce, external, false)

	if got := len(announceTable); got != 0 {
		t.Fatalf("duplicate external return queued %d announce(s), want 0", got)
	}
	entry = pathTable[key]
	if entry == nil {
		t.Fatal("path entry disappeared after duplicate external return")
	}
	if entry.Hops != 0 {
		t.Fatalf("path hops after duplicate external return=%d, want 0", entry.Hops)
	}
	if handler.calls != 1 {
		t.Fatalf("handler calls after duplicate external return=%d, want still 1", handler.calls)
	}
}

func TestLocalClientInterfacesSnapshot_FallsBackToParentInterface(t *testing.T) {
	prevOwner := Owner
	prevInterfaces := Interfaces
	prevLocalClients := LocalClientInterfaces

	shared := &Interface{Name: "Shared Instance[test]", Type: "LocalInterface"}
	local := &Interface{Name: "LocalInterface[12345]", Type: "LocalInterface", Parent: shared}
	Owner = &Reticulum{SharedInstanceInterface: shared}
	Interfaces = []*Interface{shared, local}
	LocalClientInterfaces = nil

	t.Cleanup(func() {
		Owner = prevOwner
		Interfaces = prevInterfaces
		LocalClientInterfaces = prevLocalClients
	})

	snapshot := localClientInterfacesSnapshot()
	if len(snapshot) != 1 {
		t.Fatalf("local client snapshot size=%d, want 1", len(snapshot))
	}
	if snapshot[0] != local {
		t.Fatalf("snapshot local client=%p, want %p", snapshot[0], local)
	}
	if !IsLocalClientInterface(local) {
		t.Fatal("IsLocalClientInterface() = false, want true for parent-derived local client")
	}
	if got := localClientInterfaceCount(); got != 1 {
		t.Fatalf("local client count=%d, want 1", got)
	}
}

func TestInbound_LocalClientHopCorrectionDoesNotDependOnRegistrySlice(t *testing.T) {
	resetKnownDestinationsForTest()

	prevOwner := Owner
	prevTransportIdentity := TransportIdentity
	prevTransportEnabled := transportEnabled
	prevInterfaces := Interfaces
	prevLocalClients := LocalClientInterfaces
	prevDestinations := Destinations
	prevAnnounceTable := announceTable
	prevHeldAnnounces := heldAnnounces

	shared := &Interface{Name: "Shared Instance[test]", Type: "LocalInterface"}
	local := &Interface{Name: "LocalInterface[12345]", Type: "LocalInterface", Parent: shared}

	transportID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(transport): %v", err)
	}

	Owner = &Reticulum{SharedInstanceInterface: shared}
	TransportIdentity = transportID
	transportEnabled = true
	Interfaces = []*Interface{shared, local}
	LocalClientInterfaces = nil
	Destinations = nil
	announceTable = make(map[hashKey]*announceEntry)
	heldAnnounces = make(map[hashKey]*heldAnnounce)

	t.Cleanup(func() {
		Owner = prevOwner
		TransportIdentity = prevTransportIdentity
		transportEnabled = prevTransportEnabled
		Interfaces = prevInterfaces
		LocalClientInterfaces = prevLocalClients
		Destinations = prevDestinations
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
	announce := dst.Announce([]byte("payload"), false, nil, nil, false)
	if announce == nil {
		t.Fatal("Announce returned nil")
	}
	if err := announce.Pack(); err != nil {
		t.Fatalf("Pack(): %v", err)
	}
	Destinations = nil

	Inbound(append([]byte(nil), announce.Raw...), local)

	key, ok := makeHashKey(announce.DestinationHash)
	if !ok {
		t.Fatal("announce destination hash invalid")
	}
	entry := announceTable[key]
	if entry == nil {
		t.Fatal("expected local announce to be queued")
	}
	if entry.Packet == nil {
		t.Fatal("queued announce packet is nil")
	}
	if entry.Packet.Hops != 0 {
		t.Fatalf("queued announce hops=%d, want 0 after local-client hop correction", entry.Packet.Hops)
	}
}
