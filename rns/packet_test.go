package rns

import (
	"bytes"
	"testing"
)

func TestPacket_PackUnpack_Header1_Announce(t *testing.T) {
	t.Parallel()

	id, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	dst, err := NewDestination(id, DestinationIN, DestinationSINGLE, "test", "packet")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}

	p := NewPacket(dst, []byte("announce"), PacketTypeAnnounce, PacketCtxNone, Broadcast, HeaderType1, nil, nil, true, FlagUnset)
	if p == nil {
		t.Fatalf("NewPacket returned nil")
	}
	if err := p.Pack(); err != nil {
		t.Fatalf("Pack: %v", err)
	}

	var q Packet
	q.Raw = append([]byte(nil), p.Raw...)
	if ok := q.Unpack(); !ok {
		t.Fatalf("Unpack failed")
	}
	if q.HeaderType != HeaderType1 || q.PacketType != PacketTypeAnnounce || q.Context != PacketCtxNone {
		t.Fatalf("unexpected parsed fields: header=%d type=%d ctx=%d", q.HeaderType, q.PacketType, q.Context)
	}
	if !bytes.Equal(q.DestinationHash, p.DestinationHash) {
		t.Fatalf("destination hash mismatch")
	}
	if !bytes.Equal(q.Data, []byte("announce")) {
		t.Fatalf("data mismatch: %q", string(q.Data))
	}
}

func TestPacket_HashIgnoresHops(t *testing.T) {
	t.Parallel()

	dst, err := NewDestination(nil, DestinationOUT, DestinationPLAIN, "test", "packet", "hash")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}

	p1 := NewPacket(dst, []byte("data"), PacketTypeData, PacketCtxNone, Broadcast, HeaderType1, nil, nil, true, FlagUnset)
	if err := p1.Pack(); err != nil {
		t.Fatalf("Pack1: %v", err)
	}
	h1 := p1.GetHash()

	p2 := NewPacket(dst, []byte("data"), PacketTypeData, PacketCtxNone, Broadcast, HeaderType1, nil, nil, true, FlagUnset)
	p2.Hops = 3
	if err := p2.Pack(); err != nil {
		t.Fatalf("Pack2: %v", err)
	}
	h2 := p2.GetHash()

	if !bytes.Equal(h1, h2) {
		t.Fatalf("expected hash to ignore hops")
	}
}

func TestPacket_Header2_TransportID_NotInHash(t *testing.T) {
	t.Parallel()

	dst, err := NewDestination(nil, DestinationOUT, DestinationPLAIN, "test", "packet", "hdr2")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}

	transportA := bytes.Repeat([]byte{0xAA}, ReticulumTruncatedHashLength/8)
	transportB := bytes.Repeat([]byte{0xBB}, ReticulumTruncatedHashLength/8)

	p1 := NewPacket(dst, []byte("a"), PacketTypeAnnounce, PacketCtxNone, Broadcast, HeaderType2, transportA, nil, true, FlagUnset)
	if err := p1.Pack(); err != nil {
		t.Fatalf("Pack1: %v", err)
	}
	p2 := NewPacket(dst, []byte("a"), PacketTypeAnnounce, PacketCtxNone, Broadcast, HeaderType2, transportB, nil, true, FlagUnset)
	if err := p2.Pack(); err != nil {
		t.Fatalf("Pack2: %v", err)
	}

	if !bytes.Equal(p1.GetHash(), p2.GetHash()) {
		t.Fatalf("expected header2 hash to ignore transport_id")
	}
}

func TestPacket_Send_PanicsWhenAlreadySent(t *testing.T) {
	t.Parallel()

	dst, err := NewDestination(nil, DestinationOUT, DestinationPLAIN, "test", "packet", "sendpanic")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}

	p := NewPacket(dst, []byte("data"), PacketTypeData, PacketCtxNone, Broadcast, HeaderType1, nil, nil, true, FlagUnset)
	p.Sent = true

	defer func() {
		rec := recover()
		if rec == nil {
			t.Fatalf("expected panic")
		}
		stateErr, ok := rec.(*PacketStateError)
		if !ok {
			t.Fatalf("unexpected panic type %T", rec)
		}
		if stateErr.Error() != "Packet was already sent" {
			t.Fatalf("unexpected panic message %q", stateErr.Error())
		}
	}()

	_ = p.Send()
}

func TestPacket_Resend_PanicsWhenNotSentYet(t *testing.T) {
	t.Parallel()

	dst, err := NewDestination(nil, DestinationOUT, DestinationPLAIN, "test", "packet", "resendpanic")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}

	p := NewPacket(dst, []byte("data"), PacketTypeData, PacketCtxNone, Broadcast, HeaderType1, nil, nil, true, FlagUnset)

	defer func() {
		rec := recover()
		if rec == nil {
			t.Fatalf("expected panic")
		}
		stateErr, ok := rec.(*PacketStateError)
		if !ok {
			t.Fatalf("unexpected panic type %T", rec)
		}
		if stateErr.Error() != "Packet was not sent yet" {
			t.Fatalf("unexpected panic message %q", stateErr.Error())
		}
	}()

	_ = p.Resend()
}

func TestNewRawPacket_MirrorsPackedConstructor(t *testing.T) {
	t.Parallel()

	raw := []byte{1, 2, 3, 4}
	p := NewPacket(nil, raw, PacketTypeData, PacketCtxNone, Broadcast, HeaderType1, nil, nil, true, FlagUnset)
	if p == nil {
		t.Fatalf("expected packet")
	}
	if !p.Packed || !p.FromPacked || p.CreateReceipt {
		t.Fatalf("unexpected raw packet state: packed=%v fromPacked=%v createReceipt=%v", p.Packed, p.FromPacked, p.CreateReceipt)
	}
	if !bytes.Equal(p.Raw, raw) {
		t.Fatalf("raw mismatch")
	}
	if p.Data != nil {
		t.Fatalf("expected raw packet data to be nil before unpack, got %v", p.Data)
	}
}
