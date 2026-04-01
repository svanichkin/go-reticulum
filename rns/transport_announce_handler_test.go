package rns

import (
	"bytes"
	"testing"
)

type legacyAnnounceHandler struct {
	calls int
}

func (h *legacyAnnounceHandler) AspectFilter() string { return "" }

func (h *legacyAnnounceHandler) ReceivedAnnounce(_ []byte, _ *Identity, _ []byte) {
	h.calls++
}

type hashAwareAnnounceHandler struct {
	calls       int
	lastHash    []byte
	lastDest    []byte
	lastAppData []byte
}

func (h *hashAwareAnnounceHandler) AspectFilter() string { return "" }

func (h *hashAwareAnnounceHandler) ReceivedAnnounce(_ []byte, _ *Identity, _ []byte) {}

func (h *hashAwareAnnounceHandler) ReceivedAnnounceWithPacketHash(destinationHash []byte, _ *Identity, appData []byte, announcePacketHash []byte) {
	h.calls++
	h.lastHash = copyBytes(announcePacketHash)
	h.lastDest = copyBytes(destinationHash)
	h.lastAppData = copyBytes(appData)
}

type fullAnnounceHandler struct {
	calls              int
	receivePathAnswers bool
	lastHash           []byte
	lastIsPathResponse bool
}

func (h *fullAnnounceHandler) AspectFilter() string { return "" }

func (h *fullAnnounceHandler) ReceivePathResponses() bool { return h.receivePathAnswers }

func (h *fullAnnounceHandler) ReceivedAnnounce(_ []byte, _ *Identity, _ []byte) {}

func (h *fullAnnounceHandler) ReceivedAnnounceWithPacketInfo(_ []byte, _ *Identity, _ []byte, announcePacketHash []byte, isPathResponse bool) {
	h.calls++
	h.lastHash = copyBytes(announcePacketHash)
	h.lastIsPathResponse = isPathResponse
}

func buildAnnouncePacketForHandlerTest(t *testing.T, pathResponse bool, appData []byte) *Packet {
	t.Helper()

	id, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	dst, err := NewDestination(id, DestinationIN, DestinationSINGLE, "test", "announce")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}
	packet := dst.Announce(appData, pathResponse, nil, nil, false)
	if packet == nil {
		t.Fatal("Announce returned nil")
	}
	if err := packet.Pack(); err != nil {
		t.Fatalf("Pack(): %v", err)
	}
	return packet
}

func TestNotifyAnnounceHandlers_PathResponseRequiresOptIn(t *testing.T) {
	prevHandlers := announceHandlers
	announceHandlers = nil
	t.Cleanup(func() {
		announceHandlers = prevHandlers
	})

	legacy := &legacyAnnounceHandler{}
	RegisterAnnounceHandler(legacy)

	packet := buildAnnouncePacketForHandlerTest(t, true, []byte("path-response"))
	NotifyAnnounceHandlers(packet)

	if legacy.calls != 0 {
		t.Fatalf("legacy handler calls=%d, want 0 for PATH_RESPONSE without opt-in", legacy.calls)
	}
}

func TestNotifyAnnounceHandlers_WithPacketHashReceivesHash(t *testing.T) {
	prevHandlers := announceHandlers
	announceHandlers = nil
	t.Cleanup(func() {
		announceHandlers = prevHandlers
	})

	handler := &hashAwareAnnounceHandler{}
	RegisterAnnounceHandler(handler)

	packet := buildAnnouncePacketForHandlerTest(t, false, []byte("hello"))
	NotifyAnnounceHandlers(packet)

	if handler.calls != 1 {
		t.Fatalf("handler calls=%d, want 1", handler.calls)
	}
	if !bytes.Equal(handler.lastHash, packet.PacketHash) {
		t.Fatal("handler did not receive announce packet hash")
	}
	if !bytes.Equal(handler.lastDest, packet.DestinationHash) {
		t.Fatal("handler did not receive destination hash")
	}
	if !bytes.Equal(handler.lastAppData, []byte("hello")) {
		t.Fatalf("handler app data=%q, want %q", handler.lastAppData, "hello")
	}
}

func TestNotifyAnnounceHandlers_WithPacketInfoReceivesPathResponseFlag(t *testing.T) {
	prevHandlers := announceHandlers
	announceHandlers = nil
	t.Cleanup(func() {
		announceHandlers = prevHandlers
	})

	handler := &fullAnnounceHandler{receivePathAnswers: true}
	RegisterAnnounceHandler(handler)

	packet := buildAnnouncePacketForHandlerTest(t, true, []byte("path"))
	NotifyAnnounceHandlers(packet)

	if handler.calls != 1 {
		t.Fatalf("handler calls=%d, want 1", handler.calls)
	}
	if !handler.lastIsPathResponse {
		t.Fatal("handler did not receive isPathResponse=true")
	}
	if !bytes.Equal(handler.lastHash, packet.PacketHash) {
		t.Fatal("handler did not receive announce packet hash")
	}
}
