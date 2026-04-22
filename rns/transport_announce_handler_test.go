package rns

import (
	"bytes"
	"crypto/ed25519"
	"testing"
	"time"
)

type legacyAnnounceHandler struct {
	calls int
}

func (h *legacyAnnounceHandler) AspectFilter() any { return nil }

func (h *legacyAnnounceHandler) ReceivedAnnounce(_ []byte, _ *Identity, _ []byte) {
	h.calls++
}

type filteredAnnounceHandler struct {
	filter string
	calls  int
}

func (h *filteredAnnounceHandler) AspectFilter() any { return h.filter }

func (h *filteredAnnounceHandler) ReceivedAnnounce(_ []byte, _ *Identity, _ []byte) {
	h.calls++
}

type hashAwareAnnounceHandler struct {
	calls       int
	lastHash    []byte
	lastDest    []byte
	lastAppData []byte
}

func (h *hashAwareAnnounceHandler) AspectFilter() any { return nil }

func (h *hashAwareAnnounceHandler) ReceivedAnnounce(_ []byte, _ *Identity, _ []byte) {}

func (h *hashAwareAnnounceHandler) ReceivedAnnounceWithPacketHash(destinationHash []byte, _ *Identity, appData []byte, announcePacketHash []byte) {
	h.calls++
	h.lastHash = append([]byte(nil), announcePacketHash...)
	h.lastDest = append([]byte(nil), destinationHash...)
	h.lastAppData = append([]byte(nil), appData...)
}

type fullAnnounceHandler struct {
	calls              int
	receivePathAnswers bool
	lastHash           []byte
	lastIsPathResponse bool
}

func (h *fullAnnounceHandler) AspectFilter() any { return nil }

func (h *fullAnnounceHandler) ReceivePathResponses() bool { return h.receivePathAnswers }

func (h *fullAnnounceHandler) ReceivedAnnounce(_ []byte, _ *Identity, _ []byte) {}

func (h *fullAnnounceHandler) ReceivedAnnounceWithPacketInfo(_ []byte, _ *Identity, _ []byte, announcePacketHash []byte, isPathResponse bool) {
	h.calls++
	h.lastHash = append([]byte(nil), announcePacketHash...)
	h.lastIsPathResponse = isPathResponse
}

type invalidAnnounceHandler struct{}

func waitForCalls(t *testing.T, calls func() int, want int) {
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
	if ok := ValidateAnnounce(packet, false); !ok {
		t.Fatal("ValidateAnnounce returned false for test packet")
	}
	return packet
}

func notifyAnnounceHandlersForTest(packet *Packet) {
	if packet == nil {
		return
	}
	if packet.PacketType != PacketTypeAnnounce {
		return
	}
	data := packet.Data
	nameHashLen := IdentityNameHashLength / 8
	minLen := identityPubKeyLen + nameHashLen + announceRandomHashLen + ed25519.SignatureSize
	if len(data) < minLen {
		return
	}

	announceHandlersMu.RLock()
	handlers := append([]any(nil), announceHandlers...)
	announceHandlersMu.RUnlock()

	for _, handler := range handlers {
		if handler == nil {
			continue
		}

		announceHandler, ok := handler.(AnnounceHandler)
		if !ok {
			continue
		}

		announced := IdentityRecall(packet.DestinationHash)
		filterValue := announceHandler.AspectFilter()
		if filterValue != nil {
			filter, ok := filterValue.(string)
			if !ok {
				continue
			}
			expectedHash, err := (&Destination{}).HashFromNameAndIdentity(filter, announced)
			if err != nil || !bytes.Equal(expectedHash, packet.DestinationHash) {
				continue
			}
		}

		if packet.Context == PacketPATH_RESPONSE {
			prHandler, ok := handler.(PathResponseAnnounceHandler)
			if !ok || !prHandler.ReceivePathResponses() {
				continue
			}
		}

		go func(handler any) {
			defer func() {
				if rec := recover(); rec != nil {
					Logf(LogError, "Announce handler panic: %v", rec)
				}
			}()

			appData := IdentityRecallAppData(packet.DestinationHash)
			packetHash := append([]byte(nil), packet.PacketHash...)
			if len(packetHash) == 0 {
				packetHash = append([]byte(nil), packet.GetHash()...)
			}

			switch typed := handler.(type) {
			case AnnounceHandlerWithPacketInfo:
				typed.ReceivedAnnounceWithPacketInfo(
					append([]byte(nil), packet.DestinationHash...),
					announced,
					appData,
					packetHash,
					packet.Context == PacketPATH_RESPONSE,
				)
			case AnnounceHandlerWithPacketHash:
				typed.ReceivedAnnounceWithPacketHash(
					append([]byte(nil), packet.DestinationHash...),
					announced,
					appData,
					packetHash,
				)
			default:
				announceHandler.ReceivedAnnounce(append([]byte(nil), packet.DestinationHash...), announced, appData)
			}
		}(handler)
	}
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
	if !ValidateAnnounce(packet, false) {
		t.Fatal("ValidateAnnounce returned false for PATH_RESPONSE announce")
	}
	notifyAnnounceHandlersForTest(packet)
	time.Sleep(20 * time.Millisecond)
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
	if !ValidateAnnounce(packet, false) {
		t.Fatal("ValidateAnnounce returned false for announce handler hash test")
	}
	notifyAnnounceHandlersForTest(packet)
	waitForCalls(t, func() int { return handler.calls }, 1)
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

func TestRegisterAnnounceHandler_AllowsDuplicatesAndDeregisterRemovesAllMatches(t *testing.T) {
	prevHandlers := announceHandlers
	announceHandlers = nil
	t.Cleanup(func() {
		announceHandlers = prevHandlers
	})

	handler := &legacyAnnounceHandler{}
	RegisterAnnounceHandler(handler)
	RegisterAnnounceHandler(handler)

	if got := len(announceHandlers); got != 2 {
		t.Fatalf("announceHandlers len=%d, want 2", got)
	}

	DeregisterAnnounceHandler(handler)
	if got := len(announceHandlers); got != 0 {
		t.Fatalf("announceHandlers len after deregister=%d, want 0", got)
	}
}

func TestRegisterAnnounceHandler_IgnoresInvalidObjects(t *testing.T) {
	prevHandlers := announceHandlers
	announceHandlers = nil
	t.Cleanup(func() {
		announceHandlers = prevHandlers
	})

	RegisterAnnounceHandler(invalidAnnounceHandler{})
	var nilHandler *legacyAnnounceHandler
	RegisterAnnounceHandler(nilHandler)
	if got := len(announceHandlers); got != 0 {
		t.Fatalf("announceHandlers len=%d, want 0 for invalid handlers", got)
	}

	DeregisterAnnounceHandler(invalidAnnounceHandler{})
	if got := len(announceHandlers); got != 0 {
		t.Fatalf("announceHandlers len=%d, want 0 after invalid deregister", got)
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
	if !ValidateAnnounce(packet, false) {
		t.Fatal("ValidateAnnounce returned false for PATH_RESPONSE packet info test")
	}
	notifyAnnounceHandlersForTest(packet)
	waitForCalls(t, func() int { return handler.calls }, 1)
	if !handler.lastIsPathResponse {
		t.Fatal("handler did not receive isPathResponse=true")
	}
	if !bytes.Equal(handler.lastHash, packet.PacketHash) {
		t.Fatal("handler did not receive announce packet hash")
	}
}

func TestNotifyAnnounceHandlers_AspectFilterMatchesExactly(t *testing.T) {
	prevHandlers := announceHandlers
	announceHandlers = nil
	t.Cleanup(func() {
		announceHandlers = prevHandlers
	})

	handler := &filteredAnnounceHandler{filter: " "}
	RegisterAnnounceHandler(handler)

	packet := buildAnnouncePacketForHandlerTest(t, false, []byte("exact"))
	if !ValidateAnnounce(packet, false) {
		t.Fatal("ValidateAnnounce returned false for aspect filter test")
	}
	notifyAnnounceHandlersForTest(packet)
	time.Sleep(20 * time.Millisecond)
	if handler.calls != 0 {
		t.Fatalf("handler calls=%d, want 0 for whitespace aspect filter", handler.calls)
	}
}

type explicitEmptyFilterAnnounceHandler struct {
	calls int
}

func (h *explicitEmptyFilterAnnounceHandler) AspectFilter() any { return "" }

func (h *explicitEmptyFilterAnnounceHandler) ReceivedAnnounce(_ []byte, _ *Identity, _ []byte) {
	h.calls++
}

func TestNotifyAnnounceHandlers_ExplicitEmptyFilterDoesNotWildcard(t *testing.T) {
	prevHandlers := announceHandlers
	announceHandlers = nil
	t.Cleanup(func() {
		announceHandlers = prevHandlers
	})

	handler := &explicitEmptyFilterAnnounceHandler{}
	RegisterAnnounceHandler(handler)

	packet := buildAnnouncePacketForHandlerTest(t, false, []byte("explicit-empty"))
	if !ValidateAnnounce(packet, false) {
		t.Fatal("ValidateAnnounce returned false for explicit empty filter test")
	}
	notifyAnnounceHandlersForTest(packet)

	time.Sleep(20 * time.Millisecond)
	if handler.calls != 0 {
		t.Fatalf("handler calls=%d, want 0 for explicit empty filter", handler.calls)
	}
}
