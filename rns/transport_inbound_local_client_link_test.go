package rns

import (
	"bytes"
	"testing"
	"time"
)

func TestInbound_ForLocalClientLinkRoutesLRProofWhenTransportDisabled(t *testing.T) {
	prevTransportEnabled := transportEnabled
	prevTransportIdentity := TransportIdentity
	prevInterfaces := Interfaces
	prevLocalClients := LocalClientInterfaces
	prevDestinations := Destinations
	prevLinkTable := linkTable

	transportEnabled = false
	TransportIdentity = nil
	Interfaces = nil
	LocalClientInterfaces = nil
	Destinations = nil
	linkTable = make(map[hashKey]*linkEntry)

	t.Cleanup(func() {
		transportEnabled = prevTransportEnabled
		TransportIdentity = prevTransportIdentity
		Interfaces = prevInterfaces
		LocalClientInterfaces = prevLocalClients
		Destinations = prevDestinations
		linkTable = prevLinkTable
	})

	shared := &Interface{Name: "Shared Instance[test]", Type: "LocalInterface", LocalIsSharedInstance: true}
	localClient := &Interface{Name: "local-client", Type: "LocalInterface", Parent: shared}
	external := &Interface{Name: "external", Type: "UDPInterface"}
	captured := make([][]byte, 0, 1)
	localClient.SetProcessOutgoingFunc(func(data []byte) error {
		captured = append(captured, append([]byte(nil), data...))
		return nil
	})
	Interfaces = []*Interface{shared, localClient, external}
	LocalClientInterfaces = []*Interface{localClient}

	remoteID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(remote): %v", err)
	}
	dest := &Destination{
		Type:      DestinationPLAIN,
		Direction: DestinationOUT,
		Identity:  remoteID,
		Hash:      append([]byte(nil), remoteID.Hash...),
		HexHash:   PrettyHexRep(remoteID.Hash),
	}
	Destinations = []*Destination{dest}

	linkID := bytes.Repeat([]byte{0x44}, truncatedHashBytes)
	initPub := bytes.Repeat([]byte{0x22}, linkEcPubSize/2)
	initSigPub := bytes.Repeat([]byte{0x33}, linkEcPubSize/2)
	mtu := MTU
	if phyParamsSnapshot.PhysicalLayerMTU > 0 {
		mtu = phyParamsSnapshot.PhysicalLayerMTU
	}
	signalling, err := linkSignallingBytes(mtu, linkDefaultMode)
	if err != nil {
		t.Fatalf("linkSignallingBytes(): %v", err)
	}
	peerPub := remoteID.GetPublicKey()[linkEcPubSize/2 : linkEcPubSize]
	signedData := make([]byte, 0, len(linkID)+len(initPub)+len(peerPub)+len(signalling))
	signedData = append(signedData, linkID...)
	signedData = append(signedData, initPub...)
	signedData = append(signedData, peerPub...)
	signedData = append(signedData, signalling...)
	signature, err := remoteID.Sign(signedData)
	if err != nil {
		t.Fatalf("remoteID.Sign(): %v", err)
	}
	if !remoteID.Validate(signature, signedData) {
		t.Fatal("setup signature did not validate")
	}
	proofData := make([]byte, 0, len(signature)+len(initPub)+len(signalling))
	proofData = append(proofData, signature...)
	proofData = append(proofData, initPub...)
	proofData = append(proofData, signalling...)

	link := &Link{
		Mode:        linkDefaultMode,
		MTU:         mtu,
		LinkID:      append([]byte(nil), linkID...),
		owner:       dest,
		destination: dest,
		pub:         append([]byte(nil), initPub...),
		sigPub:      append([]byte(nil), initSigPub...),
	}
	packet := NewPacket(link, proofData, PacketTypeProof, PacketCtxLRProof, Broadcast, HeaderType1, nil, nil, false, FlagUnset)
	if packet == nil {
		t.Fatal("NewPacket returned nil")
	}
	if err := packet.Pack(); err != nil {
		t.Fatalf("Pack(): %v", err)
	}
	transportID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(transport): %v", err)
	}
	TransportIdentity = transportID

	key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(linkID)
	if !ok {
		t.Fatal("hash key conversion failed")
	}
	linkTable[key] = &linkEntry{
		Timestamp:         time.Now(),
		NextHopID:         append([]byte(nil), linkID...),
		NextHopInterface:  external,
		RemainingHops:     1,
		ReceivedInterface: localClient,
		Hops:              1,
		DestinationHash:   append([]byte(nil), dest.Hash...),
		Validated:         false,
		ProofTimeout:      time.Now().Add(time.Hour),
	}
	if !func(p *Packet) bool {
		if p.Type == PacketAnnounce {
			return false
		}
		key, ok := func(hash []byte) (hashKey, bool) {
			if len(hash) < truncatedHashBytes {
				return hashKey{}, false
			}
			var key hashKey
			copy(key[:], hash[:truncatedHashBytes])
			return key, true
		}(p.DestinationHash)
		if !ok {
			return false
		}
		linkTableMu.RLock()
		entry := linkTable[key]
		linkTableMu.RUnlock()
		if entry == nil {
			return false
		}
		for _, cif := range LocalClientInterfaces {
			if entry.ReceivedInterface == cif || entry.NextHopInterface == cif {
				return true
			}
		}
		return false
	}(packet) {
		t.Fatal("expected packet to be for local-client link")
	}

	Inbound(packet.Raw, external)

	if got := len(captured); got != 1 {
		t.Fatalf("captured packets=%d, want 1", got)
	}

	expected := append([]byte(nil), packet.Raw...)
	expected[1] = 1
	if !bytes.Equal(captured[0], expected) {
		t.Fatalf("forwarded raw=%x, want %x", captured[0], expected)
	}

	if entry := linkTable[key]; entry == nil || !entry.Validated {
		t.Fatal("linkTable entry was not marked validated")
	}
}

func TestInbound_SharedInstanceClientReceivesDirectLRProofForPendingLink(t *testing.T) {
	prevTransportEnabled := transportEnabled
	prevTransportIdentity := TransportIdentity
	prevInterfaces := Interfaces
	prevLocalClients := LocalClientInterfaces
	prevDestinations := Destinations
	prevPathTable := pathTable
	prevPending := PendingLinks
	prevActive := ActiveLinks
	prevOwner := Owner
	prevPacketHashSet := PacketHashSet
	prevPacketHashSet2 := PacketHashSet2

	transportEnabled = false
	Interfaces = nil
	LocalClientInterfaces = nil
	Destinations = nil
	pathTable = make(map[hashKey]*PathEntry)
	PendingLinks = nil
	ActiveLinks = nil
	Owner = nil
	PacketHashSet = make(map[hashKey]struct{})
	PacketHashSet2 = make(map[hashKey]struct{})

	t.Cleanup(func() {
		transportEnabled = prevTransportEnabled
		TransportIdentity = prevTransportIdentity
		Interfaces = prevInterfaces
		LocalClientInterfaces = prevLocalClients
		Destinations = prevDestinations
		pathTable = prevPathTable
		PendingLinks = prevPending
		ActiveLinks = prevActive
		Owner = prevOwner
		PacketHashSet = prevPacketHashSet
		PacketHashSet2 = prevPacketHashSet2
	})

	transportID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(transport): %v", err)
	}
	TransportIdentity = transportID

	clientIfc := &Interface{Name: "LocalInterface[shared]", Type: "LocalInterface", LocalIsSharedClient: true, OUT: true}
	clientIfc.SetProcessOutgoingFunc(func([]byte) error { return nil })
	Interfaces = []*Interface{clientIfc}

	serverID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(server): %v", err)
	}
	serverDest := &Destination{
		Type:      DestinationSINGLE,
		Direction: DestinationOUT,
		Identity:  serverID,
		Hash:      append([]byte(nil), serverID.Hash...),
		HexHash:   PrettyHexRep(serverID.Hash),
	}

	key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(serverDest.Hash)
	if !ok {
		t.Fatal("hash key conversion failed")
	}
	pathTable[key] = &PathEntry{
		NextHop:       append([]byte(nil), serverDest.Hash...),
		RecvInterface: clientIfc,
		Hops:          0,
		ExpiresAt:     time.Now().Add(time.Hour),
	}

	initiator, err := NewLink(serverDest, nil, LinkModeDefault, nil, nil)
	if err != nil {
		t.Fatalf("NewLink(initiator): %v", err)
	}
	if initiator == nil {
		t.Fatal("NewLink(initiator) returned nil")
	}
	t.Cleanup(func() { initiator.Status = LinkClosed })
	if initiator.expectedHops != 0 {
		t.Fatalf("initiator.expectedHops=%d, want 0 for shared-instance local path", initiator.expectedHops)
	}

	serverOwner := &Destination{
		Type:      DestinationSINGLE,
		Direction: DestinationIN,
		Identity:  serverID,
		Hash:      append([]byte(nil), serverDest.Hash...),
		HexHash:   PrettyHexRep(serverDest.Hash),
	}
	responder, err := NewLink(nil, serverOwner, LinkModeDefault, nil, nil)
	if err != nil {
		t.Fatalf("NewLink(responder): %v", err)
	}
	if responder == nil {
		t.Fatal("NewLink(responder) returned nil")
	}
	responder.LinkID = append([]byte(nil), initiator.LinkID...)
	responder.destination = serverOwner
	serverSide := &Interface{Name: "server-side-local", Type: "LocalInterface", Parent: &Interface{LocalIsSharedInstance: true}, OUT: true}
	var captured [][]byte
	serverSide.SetProcessOutgoingFunc(func(data []byte) error {
		captured = append(captured, append([]byte(nil), data...))
		return nil
	})
	responder.attachedInterface = serverSide
	if err := responder.loadPeer(append([]byte(nil), initiator.pub...), append([]byte(nil), initiator.sigPub...)); err != nil {
		t.Fatalf("responder.loadPeer(): %v", err)
	}
	if err := responder.Handshake(); err != nil {
		t.Fatalf("responder.Handshake(): %v", err)
	}
	responder.prove()

	if len(captured) != 1 {
		t.Fatalf("captured proofs=%d, want 1", len(captured))
	}

	pkt := NewPacket(nil, captured[0], PacketTypeData, PacketCtxNone, Broadcast, HeaderType1, nil, nil, true, FlagUnset)
	if pkt == nil || !pkt.Unpack() {
		t.Fatal("captured LRPROOF could not be unpacked")
	}
	if !bytes.Equal(pkt.DestinationHash, initiator.LinkID) {
		t.Fatalf("proof destination hash=%x, want link id %x", pkt.DestinationHash, initiator.LinkID)
	}
	if len(PendingLinks) != 1 || PendingLinks[0] != initiator {
		t.Fatalf("pending links=%d, want 1 matching initiator", len(PendingLinks))
	}
	PacketHashSet = make(map[hashKey]struct{})
	PacketHashSet2 = make(map[hashKey]struct{})

	Inbound(captured[0], clientIfc)

	if initiator.Status != LinkActive {
		t.Fatalf("initiator.Status=%d, want %d (proof hops=%d expected_hops=%d)", initiator.Status, LinkActive, pkt.Hops, initiator.expectedHops)
	}
}
