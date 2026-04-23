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
	nameHash := append([]byte(nil), dst.nameHash...)
	destinationHash := append([]byte(nil), dst.Hash...)

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
	context := byte(PacketNONE)
	if pathResponse {
		context = byte(PacketPATH_RESPONSE)
	}

	packet := NewPacket(
		dst,
		data,
		PacketANNOUNCE,
		context,
		Broadcast,
		HeaderType1,
		nil,
		nil,
		false,
		FlagUnset,
	)
	if packet == nil {
		t.Fatal("NewPacket returned nil")
	}
	if err := packet.Pack(); err != nil {
		t.Fatalf("Pack(): %v", err)
	}
	return packet
}

func waitForTestCalls(t *testing.T, calls func() int, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if calls() == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("handler calls=%d, want %d", calls(), want)
}

func TestHandleInboundAnnounce_LocalClientPathResponseRequiresPendingRequest(t *testing.T) {
	resetKnownDestinationsForTest()

	prevOwner := Owner
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

	shared := &Interface{Name: "Shared Instance[test]", Type: "LocalInterface", LocalIsSharedInstance: true}
	Owner = &Reticulum{SharedInstanceInterface: shared}
	localClient := &Interface{Name: "local-client", Type: "LocalInterface", Parent: shared}
	LocalClientInterfaces = []*Interface{localClient}
	Interfaces = []*Interface{localClient}

	t.Cleanup(func() {
		Owner = prevOwner
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

	Inbound(announce.Raw, localClient)

	if entry := announceTable[key]; entry != nil {
		t.Fatalf("local PATH_RESPONSE without pending request unexpectedly queued: %+v", entry)
	}
}

func TestHandleInboundAnnounce_LocalClientPathResponseQueuesImmediateAnnounceWhenPending(t *testing.T) {
	resetKnownDestinationsForTest()

	prevOwner := Owner
	prevTransportEnabled := transportEnabled
	prevTransportIdentity := TransportIdentity
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
	sink := &outboundCapture{}

	transportID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(transport): %v", err)
	}
	TransportIdentity = transportID

	shared := &Interface{Name: "Shared Instance[test]", Type: "LocalInterface", LocalIsSharedInstance: true}
	Owner = &Reticulum{SharedInstanceInterface: shared}
	localClient := &Interface{Name: "local-client", Type: "LocalInterface", Parent: shared}
	external := &Interface{Name: "tcp-peer", Type: "TCPClientInterface", OUT: true}
	attachOutboundCapture(t, sink, external)
	LocalClientInterfaces = []*Interface{localClient}
	Interfaces = []*Interface{localClient, external}

	t.Cleanup(func() {
		Owner = prevOwner
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
	pendingLocalPathRequests[key] = external

	Inbound(announce.Raw, localClient)

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
	if entry.Next.IsZero() || entry.Next.After(time.Now().Add(500*time.Millisecond)) {
		t.Fatalf("queued local PATH_RESPONSE announce next send too late: %v", entry.Next)
	}
}

func TestHandleInboundAnnounce_LocalClientAnnounceQueuesImmediateTransportRebroadcast(t *testing.T) {
	resetKnownDestinationsForTest()

	prevTransportEnabled := transportEnabled
	prevTransportIdentity := TransportIdentity
	prevAnnounceTable := announceTable
	prevHeldAnnounces := heldAnnounces
	prevDestinations := Destinations

	transportEnabled = true
	announceTable = make(map[hashKey]*announceEntry)
	heldAnnounces = make(map[hashKey]*heldAnnounce)
	sink := &outboundCapture{}

	transportID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(transport): %v", err)
	}
	TransportIdentity = transportID
	outIfc := &Interface{Name: "tcp-peer", Type: "TCPClientInterface", OUT: true}
	attachOutboundCapture(t, sink, outIfc)
	Interfaces = []*Interface{outIfc}

	t.Cleanup(func() {
		transportEnabled = prevTransportEnabled
		TransportIdentity = prevTransportIdentity
		announceTable = prevAnnounceTable
		heldAnnounces = prevHeldAnnounces
		Destinations = prevDestinations
	})

	shared := &Interface{Name: "Shared Instance[test]", Type: "LocalInterface", LocalIsSharedInstance: true}
	localClient := &Interface{Name: "local-client", Type: "LocalInterface", Parent: shared}

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

	Inbound(announce.Raw, localClient)

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
	entry := announceTable[key]
	if entry == nil {
		t.Fatal("expected local announce to be queued")
	}
	if !entry.Next.IsZero() && entry.Next.After(time.Now().Add(500*time.Millisecond)) {
		t.Fatalf("local announce next send too late: %v", entry.Next)
	}
}

func TestHandleInboundAnnounce_SharedInstanceClientProcessesAnnounceWhenTransportDisabled(t *testing.T) {
	resetKnownDestinationsForTest()

	prevOwner := Owner
	prevTransportEnabled := transportEnabled
	prevInterfaces := Interfaces
	prevLocalClients := LocalClientInterfaces
	prevDestinations := Destinations
	prevPathTable := pathTable

	Owner = &Reticulum{IsConnectedToSharedInstance: true}
	transportEnabled = false
	Interfaces = nil
	LocalClientInterfaces = nil
	Destinations = nil
	pathTable = make(map[hashKey]*PathEntry)

	t.Cleanup(func() {
		Owner = prevOwner
		transportEnabled = prevTransportEnabled
		Interfaces = prevInterfaces
		LocalClientInterfaces = prevLocalClients
		Destinations = prevDestinations
		pathTable = prevPathTable
	})

	clientIfc := &Interface{
		Name:                "shared-client",
		Type:                "LocalInterface",
		IN:                  true,
		OUT:                 true,
		LocalIsSharedClient: true,
	}

	announceID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(announce): %v", err)
	}
	dst, err := NewDestination(announceID, DestinationIN, DestinationSINGLE, "test", "shared", "client")
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

	forwardedDst := &Destination{
		Type:      DestinationSINGLE,
		Direction: DestinationOUT,
		Hash:      append([]byte(nil), dst.Hash...),
		hexhash:   PrettyHexRep(dst.Hash),
	}
	forwarded := NewPacket(
		forwardedDst,
		append([]byte(nil), announce.Data...),
		PacketANNOUNCE,
		PacketNONE,
		TransportDirect,
		HeaderType2,
		bytes.Repeat([]byte{0x42}, truncatedHashBytes),
		nil,
		false,
		announce.ContextFlag,
	)
	if forwarded == nil {
		t.Fatal("NewPacket returned nil")
	}
	forwarded.Hops = 1
	if err := forwarded.Pack(); err != nil {
		t.Fatalf("forwarded.Pack(): %v", err)
	}

	Inbound(forwarded.Raw, clientIfc)

	recalled := IdentityRecall(dst.Hash)
	if recalled == nil {
		t.Fatal("expected shared-instance client announce to be validated and remembered")
	}
	if !bytes.Equal(recalled.GetPublicKey(), announceID.GetPublicKey()) {
		t.Fatal("recalled public key does not match announced identity")
	}
}

func TestJobs_WaitsForInboundReadLockToProcessQueuedAnnounce(t *testing.T) {
	resetKnownDestinationsForTest()

	prevOwner := Owner
	prevTransportEnabled := transportEnabled
	prevTransportIdentity := TransportIdentity
	prevAnnounceTable := announceTable
	prevHeldAnnounces := heldAnnounces
	prevInterfaces := Interfaces
	prevDestinations := Destinations
	prevAnnLast := AnnLast

	transportEnabled = true
	Owner = &Reticulum{}
	announceTable = make(map[hashKey]*announceEntry)
	heldAnnounces = make(map[hashKey]*heldAnnounce)
	sink := &outboundCapture{}

	transportID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(transport): %v", err)
	}
	TransportIdentity = transportID
	AnnLast = time.Now().Add(-2 * time.Second)

	shared := &Interface{Name: "Shared Instance[test]", Type: "LocalInterface", LocalIsSharedInstance: true}
	localClient := &Interface{Name: "local-client", Type: "LocalInterface", Parent: shared}
	external := &Interface{Name: "tcp-peer", Type: "TCPClientInterface", OUT: true}
	attachOutboundCapture(t, sink, shared)
	attachOutboundCapture(t, sink, external)
	Interfaces = []*Interface{shared, localClient, external}

	t.Cleanup(func() {
		Owner = prevOwner
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

	Inbound(announce.Raw, localClient)

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
}

func TestHandleAnnounceRetransmit_ReinsertsHeldAnnounceAfterPathResponse(t *testing.T) {
	resetKnownDestinationsForTest()

	prevTransportEnabled := transportEnabled
	prevTransportIdentity := TransportIdentity
	prevAnnounceTable := announceTable
	prevHeldAnnounces := heldAnnounces
	prevAnnLast := AnnLast
	prevLastTablesPersisted := lastTablesPersisted
	prevTablesLastCulled := TablesLastCulled
	prevLastCacheCleaned := LastCacheCleaned
	prevBlackholeLastChecked := blackholeLastChecked
	prevInterfaceLastJobs := InterfaceLastJobs
	prevMgmtDestinations := mgmtDestinations
	prevInterfaces := Interfaces
	prevOwner := Owner

	transportEnabled = true
	announceTable = make(map[hashKey]*announceEntry)
	heldAnnounces = make(map[hashKey]*heldAnnounce)
	sink := &outboundCapture{}

	transportID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(transport): %v", err)
	}
	TransportIdentity = transportID
	outIfc := &Interface{Name: "tcp-peer", Type: "TCPClientInterface", OUT: true}
	attachOutboundCapture(t, sink, outIfc)
	Interfaces = []*Interface{outIfc}
	Owner = &Reticulum{}

	t.Cleanup(func() {
		transportEnabled = prevTransportEnabled
		TransportIdentity = prevTransportIdentity
		announceTable = prevAnnounceTable
		heldAnnounces = prevHeldAnnounces
		AnnLast = prevAnnLast
		lastTablesPersisted = prevLastTablesPersisted
		TablesLastCulled = prevTablesLastCulled
		LastCacheCleaned = prevLastCacheCleaned
		blackholeLastChecked = prevBlackholeLastChecked
		InterfaceLastJobs = prevInterfaceLastJobs
		mgmtDestinations = prevMgmtDestinations
		Owner = prevOwner
		Interfaces = prevInterfaces
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

	heldEntry := &announceEntry{
		Packet:            announce,
		Next:              time.Now().Add(time.Minute),
		Timestamp:         time.Now(),
		Expires:           time.Now().Add(time.Hour),
		AttachedInterface: &Interface{Name: "held-if"},
	}
	pathResponseEntry := &announceEntry{
		Packet:            announce,
		Next:              time.Now().Add(-time.Second),
		Timestamp:         time.Now(),
		Expires:           time.Now().Add(time.Hour),
		BlockRebroadcasts: true,
		AttachedInterface: &Interface{Name: "resp-if"},
	}

	announceTable[key] = pathResponseEntry
	heldAnnounces[key] = &heldAnnounce{Entry: heldEntry}

	AnnLast = time.Now().Add(-2 * time.Second)
	LastCacheCleaned = time.Now()
	TablesLastCulled = time.Now()
	blackholeLastChecked = time.Now()
	InterfaceLastJobs = time.Now()
	Jobs()

	if announceTable[key] != heldEntry {
		t.Fatal("held announce was not restored after path response retransmit")
	}
	if _, exists := heldAnnounces[key]; exists {
		t.Fatal("held announce entry was not cleared")
	}
}

func TestHandleAnnounceRetransmit_AllowsSecondRetryForNonLocalAnnounce(t *testing.T) {
	resetKnownDestinationsForTest()

	prevTransportEnabled := transportEnabled
	prevTransportIdentity := TransportIdentity
	prevAnnounceTable := announceTable
	prevHeldAnnounces := heldAnnounces
	prevAnnLast := AnnLast
	prevLastTablesPersisted := lastTablesPersisted
	prevTablesLastCulled := TablesLastCulled
	prevLastCacheCleaned := LastCacheCleaned
	prevBlackholeLastChecked := blackholeLastChecked
	prevInterfaceLastJobs := InterfaceLastJobs
	prevMgmtDestinations := mgmtDestinations
	prevInterfaces := Interfaces

	transportEnabled = true
	announceTable = make(map[hashKey]*announceEntry)
	heldAnnounces = make(map[hashKey]*heldAnnounce)
	sink := &outboundCapture{}

	transportID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(transport): %v", err)
	}
	TransportIdentity = transportID
	outIfc := &Interface{Name: "tcp-peer", Type: "TCPClientInterface", OUT: true}
	attachOutboundCapture(t, sink, outIfc)
	Interfaces = []*Interface{outIfc}

	t.Cleanup(func() {
		transportEnabled = prevTransportEnabled
		TransportIdentity = prevTransportIdentity
		announceTable = prevAnnounceTable
		heldAnnounces = prevHeldAnnounces
		AnnLast = prevAnnLast
		lastTablesPersisted = prevLastTablesPersisted
		TablesLastCulled = prevTablesLastCulled
		LastCacheCleaned = prevLastCacheCleaned
		blackholeLastChecked = prevBlackholeLastChecked
		InterfaceLastJobs = prevInterfaceLastJobs
		mgmtDestinations = prevMgmtDestinations
		Interfaces = prevInterfaces
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

	announceTable[key] = &announceEntry{
		Packet:    announce,
		Next:      time.Now().Add(-time.Second),
		Timestamp: time.Now(),
		Expires:   time.Now().Add(time.Hour),
	}

	AnnLast = time.Now().Add(-2 * time.Second)
	lastTablesPersisted = time.Now()
	TablesLastCulled = time.Now()
	LastCacheCleaned = time.Now()
	blackholeLastChecked = time.Now()
	InterfaceLastJobs = time.Now()
	mgmtDestinations = nil

	Jobs()
	if len(sink.packets) != 1 {
		t.Fatalf("first outgoing packets=%d, want 1", len(sink.packets))
	}
	entry := announceTable[key]
	if entry == nil {
		t.Fatal("announce entry disappeared after first retry")
	}
	if entry.Retries != 1 {
		t.Fatalf("retries after first send=%d, want 1", entry.Retries)
	}

	if entry.Retries != 1 {
		t.Fatalf("retries after first send=%d, want 1", entry.Retries)
	}
	entry.Next = time.Now().Add(-time.Millisecond)
	announceTable[key] = entry
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

	key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(dst.Hash)
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

	shared := &Interface{Name: "Shared Instance[test]", Type: "LocalInterface", LocalIsSharedInstance: true}
	localClient := &Interface{Name: "local-client", Type: "LocalInterface", Parent: shared}
	external := &Interface{Name: "tcp-peer", Type: "TCPClientInterface", OUT: true}
	LocalClientInterfaces = []*Interface{localClient}
	Interfaces = []*Interface{shared, localClient, external}

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

	Inbound(localAnnounce.Raw, localClient)

	key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(localAnnounce.DestinationHash)
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
	waitForTestCalls(t, func() int { return handler.calls }, 1)

	announceTable = make(map[hashKey]*announceEntry)

	announceIdentity := IdentityRecall(localAnnounce.DestinationHash)
	announceDestination := &Destination{
		Type:      DestinationSINGLE,
		Direction: DestinationOUT,
		identity:  announceIdentity,
		Hash:      append([]byte(nil), localAnnounce.DestinationHash...),
		hexhash:   PrettyHexRep(localAnnounce.DestinationHash),
	}
	returnedAnnounce := NewPacket(
		announceDestination,
		append([]byte(nil), localAnnounce.Data...),
		PacketANNOUNCE,
		PacketNONE,
		TransportDirect,
		HeaderType2,
		nil,
		nil,
		true,
		localAnnounce.ContextFlag,
	)
	if returnedAnnounce == nil {
		t.Fatal("returned announce was nil")
	}
	if TransportIdentity != nil && len(TransportIdentity.Hash) > 0 {
		returnedAnnounce.TransportID = append([]byte(nil), TransportIdentity.Hash...)
	}
	returnedAnnounce.DestinationHash = append([]byte(nil), localAnnounce.DestinationHash...)
	returnedAnnounce.DestinationType = byte(DestinationSINGLE)
	returnedAnnounce.Hops = localAnnounce.Hops
	if err := returnedAnnounce.Pack(); err != nil {
		t.Fatalf("Pack(returned announce): %v", err)
	}
	returnedAnnounce.ReceivingInterface = external
	returnedAnnounce.Hops = 1

	Inbound(returnedAnnounce.Raw, external)

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
	waitForTestCalls(t, func() int { return handler.calls }, 1)
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

	shared := &Interface{Name: "Shared Instance[test]", Type: "LocalInterface", LocalIsSharedInstance: true}
	localClient := &Interface{Name: "local-client", Type: "LocalInterface", Parent: shared}
	external := &Interface{Name: "tcp-peer", Type: "TCPClientInterface", OUT: true}
	LocalClientInterfaces = []*Interface{localClient}
	Interfaces = []*Interface{shared, localClient, external}

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
	Inbound(autoPathResponse.Raw, localClient)

	normalAnnounce := buildAnnounceWithRandomBlob(t, dst, []byte("payload"), false, bytes.Repeat([]byte{0xFF}, announceRandomHashLen))
	normalAnnounce.ReceivingInterface = localClient
	normalAnnounce.Hops = 0
	Inbound(normalAnnounce.Raw, localClient)

	key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(normalAnnounce.DestinationHash)
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
	waitForTestCalls(t, func() int { return handler.calls }, 1)

	announceTable = make(map[hashKey]*announceEntry)

	announceIdentity := IdentityRecall(normalAnnounce.DestinationHash)
	announceDestination := &Destination{
		Type:      DestinationSINGLE,
		Direction: DestinationOUT,
		identity:  announceIdentity,
		Hash:      append([]byte(nil), normalAnnounce.DestinationHash...),
		hexhash:   PrettyHexRep(normalAnnounce.DestinationHash),
	}
	returnedAnnounce := NewPacket(
		announceDestination,
		append([]byte(nil), normalAnnounce.Data...),
		PacketANNOUNCE,
		PacketNONE,
		TransportDirect,
		HeaderType2,
		nil,
		nil,
		true,
		normalAnnounce.ContextFlag,
	)
	if returnedAnnounce == nil {
		t.Fatal("returned announce was nil")
	}
	if TransportIdentity != nil && len(TransportIdentity.Hash) > 0 {
		returnedAnnounce.TransportID = append([]byte(nil), TransportIdentity.Hash...)
	}
	returnedAnnounce.DestinationHash = append([]byte(nil), normalAnnounce.DestinationHash...)
	returnedAnnounce.DestinationType = byte(DestinationSINGLE)
	returnedAnnounce.Hops = normalAnnounce.Hops
	if err := returnedAnnounce.Pack(); err != nil {
		t.Fatalf("Pack(returned announce): %v", err)
	}
	returnedAnnounce.ReceivingInterface = external
	returnedAnnounce.Hops = 2

	Inbound(returnedAnnounce.Raw, external)

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
	waitForTestCalls(t, func() int { return handler.calls }, 1)
}

func TestLocalClientInterfacesSnapshot_UsesExplicitRegistryOnly(t *testing.T) {
	prevOwner := Owner
	prevInterfaces := Interfaces
	prevLocalClients := LocalClientInterfaces

	shared := &Interface{Name: "Shared Instance[test]", Type: "LocalInterface"}
	shared.LocalIsSharedInstance = true
	local := &Interface{Name: "LocalInterface[12345]", Type: "LocalInterface", Parent: shared}
	Owner = &Reticulum{SharedInstanceInterface: shared}
	Interfaces = []*Interface{shared, local}
	LocalClientInterfaces = nil

	t.Cleanup(func() {
		Owner = prevOwner
		Interfaces = prevInterfaces
		LocalClientInterfaces = prevLocalClients
	})

	if len(LocalClientInterfaces) != 0 {
		t.Fatalf("local client slice size=%d, want 0", len(LocalClientInterfaces))
	}
	if !IsLocalClientInterface(local) {
		t.Fatal("IsLocalClientInterface() = false, want true for parent-derived local client")
	}
	if got := len(LocalClientInterfaces); got != 0 {
		t.Fatalf("local client count=%d, want 0", got)
	}
}

func TestInbound_LocalClientHopCorrection_FollowsPythonRegistryGate(t *testing.T) {
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
	shared.LocalIsSharedInstance = true
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
	entry := announceTable[key]
	if entry == nil {
		t.Fatal("expected local announce to be queued")
	}
	if entry.Packet == nil {
		t.Fatal("queued announce packet is nil")
	}
	if entry.Packet.Hops != 1 {
		t.Fatalf("queued announce hops=%d, want 1 when local-client registry slice is empty", entry.Packet.Hops)
	}
}

func TestOutbound_SharedInstanceOneHopRewritesHeader2(t *testing.T) {
	resetKnownDestinationsForTest()

	prevOwner := Owner
	prevPathTable := pathTable
	prevInterfaces := Interfaces

	Owner = &Reticulum{IsConnectedToSharedInstance: true}
	pathTable = make(map[hashKey]*PathEntry)
	Interfaces = nil

	t.Cleanup(func() {
		Owner = prevOwner
		pathTable = prevPathTable
		Interfaces = prevInterfaces
	})

	destID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(dest): %v", err)
	}
	dst, err := NewDestination(destID, DestinationOUT, DestinationSINGLE, "test", "shared", "onehop")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}

	nextHop := bytes.Repeat([]byte{0x42}, truncatedHashBytes)
	outIfc := &Interface{Name: "shared-client", Type: "TCPClientInterface", OUT: true}
	var captured []byte
	outIfc.SetProcessOutgoingFunc(func(data []byte) error {
		captured = append([]byte(nil), data...)
		return nil
	})
	Interfaces = []*Interface{outIfc}

	key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(dst.Hash)
	if !ok {
		t.Fatal("destination hash invalid")
	}
	pathTable[key] = &PathEntry{
		NextHop:       append([]byte(nil), nextHop...),
		RecvInterface: outIfc,
		Hops:          1,
		Timestamp:     time.Now(),
		ExpiresAt:     time.Now().Add(time.Hour),
	}

	packet := NewPacket(dst, []byte("payload"), PacketTypeData, PacketCtxNone, Broadcast, HeaderType1, nil, nil, true, FlagUnset)
	if packet == nil {
		t.Fatal("NewPacket returned nil")
	}
	if err := packet.Pack(); err != nil {
		t.Fatalf("Pack(): %v", err)
	}

	if !Outbound(packet) {
		t.Fatal("Outbound returned false")
	}
	if len(captured) == 0 {
		t.Fatal("expected packet to be transmitted to outbound interface")
	}
	if got := captured[0] >> 6; got != HeaderType2 {
		t.Fatalf("captured header type=%d, want HEADER_2", got)
	}
	if got := (captured[0] >> 4) & 0x01; got != TransportDirect {
		t.Fatalf("captured transport type=%d, want TRANSPORT", got)
	}
	if !bytes.Equal(captured[2:2+len(nextHop)], nextHop) {
		t.Fatal("captured next hop mismatch")
	}
}
