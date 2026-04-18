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
		identity:  remoteID,
		hash:      append([]byte(nil), remoteID.Hash...),
		hexhash:   PrettyHexRep(remoteID.Hash),
	}
	Destinations = []*Destination{dest}

	linkID := bytes.Repeat([]byte{0x44}, truncatedHashBytes)
	initPub := bytes.Repeat([]byte{0x22}, linkEcPubSize/2)
	initSigPub := bytes.Repeat([]byte{0x33}, linkEcPubSize/2)
	mtu := defaultLinkMTU()
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
	packet := NewPacket(link, proofData, WithPacketType(PacketTypeProof), WithPacketContext(PacketCtxLRProof), WithCreateReceipt(false))
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
		DestinationHash:   append([]byte(nil), dest.hash...),
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

	Inbound(packet.RawBytes(), external)

	if got := len(captured); got != 1 {
		t.Fatalf("captured packets=%d, want 1", got)
	}

	expected := append([]byte(nil), packet.RawBytes()...)
	expected[1] = 1
	if !bytes.Equal(captured[0], expected) {
		t.Fatalf("forwarded raw=%x, want %x", captured[0], expected)
	}

	if entry := linkTable[key]; entry == nil || !entry.Validated {
		t.Fatal("linkTable entry was not marked validated")
	}
}
