package rns

import (
	"bytes"
	"testing"
	"time"
)

type testTransport struct {
	sent []*Packet
}

func (t *testTransport) Outbound(p *Packet) bool {
	if p == nil {
		return false
	}
	if !p.Sent {
		p.Sent = true
		p.SentAt = time.Now()
	}
	if p.CreateReceipt && p.Receipt == nil {
		p.Receipt = NewPacketReceipt(p)
	}
	t.sent = append(t.sent, p)
	return true
}

func (t *testTransport) HopsTo(_ []byte) int { return 1 }
func (t *testTransport) GetFirstHopTimeout(_ []byte) time.Duration {
	return 0
}
func (t *testTransport) GetPacketRSSI(_ []byte) *float64 { return nil }
func (t *testTransport) GetPacketSNR(_ []byte) *float64  { return nil }
func (t *testTransport) GetPacketQ(_ []byte) *float64    { return nil }

func TestPacketIntegration_ExplicitProof_ValidatesReceipt(t *testing.T) {
	prevInterfaces := Interfaces
	Interfaces = nil
	t.Cleanup(func() {
		Interfaces = prevInterfaces
	})

	id, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}

	dst, err := NewDestination(id, DestinationIN, DestinationSINGLE, "test", "packet", "proof")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}

	sink := &outboundCapture{}
	ifc := &Interface{Name: "packet-if0", Type: "Test", OUT: true}
	attachOutboundCapture(t, sink, ifc)
	Interfaces = []*Interface{ifc}

	p := NewPacket(dst, []byte("payload"), WithCreateReceipt(true))
	if p == nil {
		t.Fatalf("NewPacket returned nil")
	}
	if p.Send() == nil || p.Receipt == nil {
		t.Fatalf("expected receipt")
	}
	if p.Receipt.Status != ReceiptSent {
		t.Fatalf("unexpected receipt status: %d", p.Receipt.Status)
	}
	if len(sink.packets) != 1 {
		t.Fatalf("expected 1 outbound packet, got %d", len(sink.packets))
	}

	// Create a proof packet like Identity.Prove() would.
	packetHash := p.GetHash()
	sig, err := id.Sign(packetHash)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	proofData := append(append([]byte(nil), packetHash...), sig...)

	proof := NewPacket(p.GenerateProofDestination(), proofData, WithPacketType(PacketTypeProof), WithCreateReceipt(false))
	if proof == nil {
		t.Fatalf("proof packet nil")
	}
	if err := proof.Pack(); err != nil {
		t.Fatalf("proof pack: %v", err)
	}

	if ok := p.Receipt.ValidateProofPacket(proof); !ok {
		t.Fatalf("expected proof validation ok")
	}
	if p.Receipt.Status != ReceiptDelivered || !p.Receipt.Proved {
		t.Fatalf("expected delivered/proved")
	}
}

func TestPacketIntegration_ImplicitProof_ValidatesReceipt(t *testing.T) {
	prevInterfaces := Interfaces
	Interfaces = nil
	t.Cleanup(func() {
		Interfaces = prevInterfaces
	})

	id, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}

	dst, err := NewDestination(id, DestinationIN, DestinationSINGLE, "test", "packet", "iproof")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}

	sink := &outboundCapture{}
	ifc := &Interface{Name: "packet-if1", Type: "Test", OUT: true}
	attachOutboundCapture(t, sink, ifc)
	Interfaces = []*Interface{ifc}

	p := NewPacket(dst, []byte("payload"), WithCreateReceipt(true))
	if p.Send() == nil || p.Receipt == nil {
		t.Fatalf("expected receipt")
	}
	if len(sink.packets) != 1 {
		t.Fatalf("expected 1 outbound packet, got %d", len(sink.packets))
	}

	packetHash := p.GetHash()
	sig, err := id.Sign(packetHash)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	// Implicit proof = signature only.
	proof := NewPacket(p.GenerateProofDestination(), sig, WithPacketType(PacketTypeProof), WithCreateReceipt(false))
	if err := proof.Pack(); err != nil {
		t.Fatalf("proof pack: %v", err)
	}
	if ok := p.Receipt.ValidateProofPacket(proof); !ok {
		t.Fatalf("expected proof validation ok")
	}
	if p.Receipt.Status != ReceiptDelivered || !p.Receipt.Proved {
		t.Fatalf("expected delivered/proved")
	}
}

func TestPacketIntegration_Header2_Announce_RoundTripUnpack(t *testing.T) {
	t.Parallel()

	id, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	dst, err := NewDestination(id, DestinationIN, DestinationSINGLE, "test", "packet", "hdr2")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}

	transportID := bytes.Repeat([]byte{0x10}, ReticulumTruncatedHashLength/8)
	p := NewPacket(dst, []byte("announce"), WithHeaderType(HeaderType2), WithTransportID(transportID), WithPacketType(PacketTypeAnnounce))
	if err := p.Pack(); err != nil {
		t.Fatalf("pack: %v", err)
	}

	var q Packet
	q.Raw = p.RawBytes()
	if ok := q.Unpack(); !ok {
		t.Fatalf("unpack failed")
	}
	if q.HeaderType != HeaderType2 {
		t.Fatalf("unexpected header type: %d", q.HeaderType)
	}
	if !bytes.Equal(q.TransportID, transportID) {
		t.Fatalf("transport id mismatch")
	}
	if !bytes.Equal(q.Data, []byte("announce")) {
		t.Fatalf("data mismatch")
	}
}
