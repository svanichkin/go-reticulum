package rns

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

type integrationTransport struct {
	mu                            sync.Mutex
	pendingByLink                 map[string][][]byte
	sentByContext                 map[byte]int
	deliveredByContext            map[byte]int
	deliveredByContextToInitiator map[byte]int
	deliveredByContextToResponder map[byte]int
}

type integrationTransportProvider interface {
	baseTransport() *integrationTransport
}

func (it *integrationTransport) baseTransport() *integrationTransport { return it }

func (integrationTransport) GetFirstHopTimeout(dstHash []byte) time.Duration {
	return 10 * time.Millisecond
}
func (integrationTransport) HopsTo(dstHash []byte) int          { return 1 }
func (integrationTransport) GetPacketRSSI(hash []byte) *float64 { return nil }
func (integrationTransport) GetPacketSNR(hash []byte) *float64  { return nil }
func (integrationTransport) GetPacketQ(hash []byte) *float64    { return nil }

type integrationTransportDelayed struct {
	base    *integrationTransport
	bitrate int // bits per second
}

func (it *integrationTransportDelayed) baseTransport() *integrationTransport { return it.base }

func (it *integrationTransportDelayed) GetFirstHopTimeout(dstHash []byte) time.Duration {
	return it.base.GetFirstHopTimeout(dstHash)
}
func (it *integrationTransportDelayed) HopsTo(dstHash []byte) int { return it.base.HopsTo(dstHash) }
func (it *integrationTransportDelayed) GetPacketRSSI(hash []byte) *float64 {
	return it.base.GetPacketRSSI(hash)
}
func (it *integrationTransportDelayed) GetPacketSNR(hash []byte) *float64 {
	return it.base.GetPacketSNR(hash)
}
func (it *integrationTransportDelayed) GetPacketQ(hash []byte) *float64 {
	return it.base.GetPacketQ(hash)
}

func (it *integrationTransportDelayed) Outbound(p *Packet) bool {
	// Simulate a slow local interface by adding serialization delay based on frame size.
	if it.bitrate > 0 && p != nil {
		if !p.Packed {
			_ = p.Pack()
		}
		rawLen := len(p.Raw)
		if rawLen == 0 {
			rawLen = len(p.Data)
		}
		// bits / bps = seconds
		delay := time.Duration((float64(rawLen) * 8.0 / float64(it.bitrate)) * float64(time.Second)) //nolint:durationcheck
		if delay > 0 {
			time.Sleep(delay)
		}
	}
	return it.base.Outbound(p)
}

func (it *integrationTransport) Outbound(p *Packet) bool {
	if p == nil {
		return false
	}
	it.mu.Lock()
	if it.sentByContext == nil {
		it.sentByContext = make(map[byte]int)
	}
	it.sentByContext[p.Context]++
	it.mu.Unlock()
	if !p.Packed {
		if err := p.Pack(); err != nil {
			return false
		}
	}
	p.Sent = true
	p.SentAt = time.Now()
	if p.CreateReceipt {
		rc := NewPacketReceipt(p)
		p.Receipt = rc
		Receipts = append(Receipts, rc)
	}

	in := NewPacket(nil, p.Raw)
	if in == nil || !in.Unpack() {
		return false
	}

	var deliver func(pkt *Packet) bool
	deliver = func(pkt *Packet) bool {
		// If this packet is associated with a link, we can deterministically route
		// it to the peer link (initiator <-> responder) in this in-process harness.
		if p.Link != nil && len(p.Link.LinkID) > 0 && bytesEqual(pkt.DestinationHash, p.Link.LinkID) {
			if p.Link.Initiator {
				if responder := findResponderLinkByIDTest(p.Link.LinkID); responder != nil {
					pkt.Link = responder
					pkt.Destination = responder.destination
				}
			} else {
				if initiator := findInitiatorLinkByIDTest(p.Link.LinkID); initiator != nil {
					pkt.Link = initiator
					pkt.Destination = initiator.destination
				}
			}
		}

		// If the harness already resolved the link, don't override it with generic lookup
		// (since both initiator and responder share the same link ID).
		if pkt.Link == nil {
			// Resolve destination/link for the inbound copy.
			if pkt.DestinationType == byte(DestinationLINK) || pkt.Context == PacketCtxLRProof {
				// Find link by link ID (destination hash is link_id for link packets / lrproof).
				var l *Link
				if pkt.Context == PacketCtxLRProof {
					l = findInitiatorLinkByIDTest(pkt.DestinationHash)
				} else {
					l = findLinkByIDTest(pkt.DestinationHash)
				}
				if l != nil {
					pkt.Link = l
					pkt.Destination = l.destination
				} else if pkt.Context == PacketCtxLRProof {
					// Outgoing link might not be registered yet; stash and retry later.
					rawCopy := append([]byte(nil), pkt.Raw...)
					it.mu.Lock()
					if it.pendingByLink == nil {
						it.pendingByLink = make(map[string][][]byte)
					}
					key := string(pkt.DestinationHash)
					it.pendingByLink[key] = append(it.pendingByLink[key], rawCopy)
					it.mu.Unlock()
					destHash := append([]byte(nil), pkt.DestinationHash...)
					go func() {
						deadline := time.Now().Add(200 * time.Millisecond)
						for time.Now().Before(deadline) {
							if findInitiatorLinkByIDTest(destHash) != nil {
								retry := NewPacket(nil, rawCopy)
								if retry == nil || !retry.Unpack() {
									return
								}
								_ = deliver(retry)
								return
							}
							time.Sleep(2 * time.Millisecond)
						}
					}()
					return true
				}
			} else {
				pkt.Destination = findDestinationByHash(pkt.DestinationHash)
			}
		}

		// Proofs are handled by transport receipt logic first.
		if pkt.PacketType == PacketTypeProof {
			// Link proofs are always destined for the initiator (the sender that
			// created the receipt), so route them deterministically to avoid
			// relying on link list ordering.
			if pkt.DestinationType == byte(DestinationLINK) && pkt.Context == PacketCtxNone {
				if initiator := findInitiatorLinkByIDTest(pkt.DestinationHash); initiator != nil {
					pkt.Link = initiator
					pkt.Destination = initiator.destination
				}
			}
			if handled := handleInboundProof(pkt); handled {
				return true
			}
		}

		if pkt.Link != nil {
			it.mu.Lock()
			if it.deliveredByContext == nil {
				it.deliveredByContext = make(map[byte]int)
			}
			it.deliveredByContext[pkt.Context]++
			if pkt.Link.Initiator {
				if it.deliveredByContextToInitiator == nil {
					it.deliveredByContextToInitiator = make(map[byte]int)
				}
				it.deliveredByContextToInitiator[pkt.Context]++
			} else {
				if it.deliveredByContextToResponder == nil {
					it.deliveredByContextToResponder = make(map[byte]int)
				}
				it.deliveredByContextToResponder[pkt.Context]++
			}
			it.mu.Unlock()
			pkt.Link.Receive(pkt)
			return true
		}
		if pkt.Destination != nil {
			it.mu.Lock()
			if it.deliveredByContext == nil {
				it.deliveredByContext = make(map[byte]int)
			}
			it.deliveredByContext[pkt.Context]++
			it.mu.Unlock()
			return pkt.Destination.Receive(pkt)
		}
		return false
	}

	_ = deliver(in)

	// Try to flush any stashed LRPROOF frames that now have a registered link.
	it.mu.Lock()
	pending := it.pendingByLink
	it.pendingByLink = nil
	it.mu.Unlock()
	for _, raws := range pending {
		for _, raw := range raws {
			pkt := NewPacket(nil, raw)
			if pkt == nil || !pkt.Unpack() {
				continue
			}
			_ = deliver(pkt)
		}
	}

	return true
}

func (it *integrationTransport) resetStatsLocked() {
	it.sentByContext = make(map[byte]int)
	it.deliveredByContext = make(map[byte]int)
	it.deliveredByContextToInitiator = make(map[byte]int)
	it.deliveredByContextToResponder = make(map[byte]int)
}

func resetIntegrationTransportStats() {
	if it := getIntegrationTransport(); it != nil {
		it.mu.Lock()
		it.resetStatsLocked()
		it.mu.Unlock()
	}
}

func findDestinationByHash(hash []byte) *Destination {
	// Prefer inbound destinations when both IN/OUT share the same hash (common in tests
	// where we simulate both sides inside one process).
	var fallback *Destination
	for _, d := range Destinations {
		if d == nil || len(d.hash) == 0 {
			continue
		}
		if bytesEqual(d.hash, hash) {
			if d.Direction == DestinationIN {
				return d
			}
			if fallback == nil {
				fallback = d
			}
		}
	}
	return fallback
}

func findLinkByIDTest(linkID []byte) *Link {
	linkMu.Lock()
	defer linkMu.Unlock()
	for _, l := range ActiveLinks {
		if l != nil && l.Status != LinkClosed && bytesEqual(l.LinkID, linkID) {
			return l
		}
	}
	for _, l := range PendingLinks {
		if l != nil && l.Status != LinkClosed && bytesEqual(l.LinkID, linkID) {
			return l
		}
	}
	return nil
}

func findInitiatorLinkByIDTest(linkID []byte) *Link {
	linkMu.Lock()
	defer linkMu.Unlock()
	for _, l := range ActiveLinks {
		if l != nil && l.Status != LinkClosed && l.Initiator && bytesEqual(l.LinkID, linkID) {
			return l
		}
	}
	for _, l := range PendingLinks {
		if l != nil && l.Status != LinkClosed && l.Initiator && bytesEqual(l.LinkID, linkID) {
			return l
		}
	}
	return nil
}

func findResponderLinkByIDTest(linkID []byte) *Link {
	linkMu.Lock()
	defer linkMu.Unlock()
	for _, l := range ActiveLinks {
		if l != nil && l.Status != LinkClosed && !l.Initiator && bytesEqual(l.LinkID, linkID) {
			return l
		}
	}
	for _, l := range PendingLinks {
		if l != nil && l.Status != LinkClosed && !l.Initiator && bytesEqual(l.LinkID, linkID) {
			return l
		}
	}
	return nil
}

func findPeerLinkTest(self *Link) *Link {
	if self == nil || len(self.LinkID) == 0 {
		return nil
	}
	linkMu.Lock()
	defer linkMu.Unlock()
	for _, l := range ActiveLinks {
		if l != nil && l != self && l.Status != LinkClosed && bytesEqual(l.LinkID, self.LinkID) {
			return l
		}
	}
	for _, l := range PendingLinks {
		if l != nil && l != self && l.Status != LinkClosed && bytesEqual(l.LinkID, self.LinkID) {
			return l
		}
	}
	return nil
}

func withIntegrationTransport(t *testing.T, fn func()) {
	t.Helper()
	prev := Transport
	// Store stateful transport instance behind the interface.
	it := &integrationTransport{}
	Transport = it
	defer func() { Transport = prev }()
	fn()
}

func getIntegrationTransport() *integrationTransport {
	if provider, ok := Transport.(integrationTransportProvider); ok {
		return provider.baseTransport()
	}
	return nil
}

func requireIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("RNS_INTEGRATION") == "" {
		t.Skip("set RNS_INTEGRATION=1 to run integration tests")
	}
}

func TestIntegration_LinkEstablish_DefaultMode(t *testing.T) {
	requireIntegration(t)
	resetKnownDestinationsForTest()
	withIntegrationTransport(t, func() {
		// Mirrors core behaviour from `tests/link.py::test_02_establish`, but uses an in-process transport harness.
		prvHex := "f8953ffaf607627e615603ff1530c82c434cf87c07179dd7689ea776f30b964cfb7ba6164af00c5111a45e69e57d885e1285f8dbfe3a21e95ae17cf676b0f8b7"
		prv, _ := hex.DecodeString(prvHex)
		id, err := IdentityFromBytes(prv)
		if err != nil {
			t.Fatalf("IdentityFromBytes: %v", err)
		}

		const appName = "rns_unit_tests"
		destOut, err := NewDestination(id, DestinationOUT, DestinationSINGLE, appName, "link", "establish")
		if err != nil {
			t.Fatalf("NewDestination(out): %v", err)
		}
		destIn, err := NewDestination(id, DestinationIN, DestinationSINGLE, appName, "link", "establish")
		if err != nil {
			t.Fatalf("NewDestination(in): %v", err)
		}
		_ = destIn

		l, err := NewOutgoingLink(destOut, LinkModeDefault, nil, nil)
		if err != nil {
			t.Fatalf("NewOutgoingLink: %v", err)
		}

		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if l.Status == LinkActive {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		if l.Status != LinkActive {
			t.Fatalf("expected link active, got status %d", l.Status)
		}

		l.Teardown()
		deadline = time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if l.Status == LinkClosed {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		if l.Status != LinkClosed {
			t.Fatalf("expected link closed, got status %d", l.Status)
		}
	})
}

func TestIntegration_LinkEstablish_AES256CBC_Mode(t *testing.T) {
	requireIntegration(t)
	resetKnownDestinationsForTest()
	withIntegrationTransport(t, func() {
		prvHex := "f8953ffaf607627e615603ff1530c82c434cf87c07179dd7689ea776f30b964cfb7ba6164af00c5111a45e69e57d885e1285f8dbfe3a21e95ae17cf676b0f8b7"
		prv, _ := hex.DecodeString(prvHex)
		id, err := IdentityFromBytes(prv)
		if err != nil {
			t.Fatalf("IdentityFromBytes: %v", err)
		}

		const appName = "rns_unit_tests"
		destOut, err := NewDestination(id, DestinationOUT, DestinationSINGLE, appName, "link", "establish")
		if err != nil {
			t.Fatalf("NewDestination(out): %v", err)
		}
		_, err = NewDestination(id, DestinationIN, DestinationSINGLE, appName, "link", "establish")
		if err != nil {
			t.Fatalf("NewDestination(in): %v", err)
		}

		l, err := NewOutgoingLink(destOut, LinkModeAES256CBC, nil, nil)
		if err != nil {
			t.Fatalf("NewOutgoingLink: %v", err)
		}

		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if l.Status == LinkActive {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		if l.Status != LinkActive {
			t.Fatalf("expected link active, got status %d", l.Status)
		}
		if l.Mode != LinkModeAES256CBC {
			t.Fatalf("expected mode AES256CBC, got %d", l.Mode)
		}
		if got := len(l.derivedKey); got != 64 {
			t.Fatalf("expected derived key length 64, got %d", got)
		}

		l.Teardown()
	})
}

func TestIntegration_LinkEstablish_AES128CBC(t *testing.T) {
	requireIntegration(t)
	resetKnownDestinationsForTest()
	withIntegrationTransport(t, func() {
		prvHex := "f8953ffaf607627e615603ff1530c82c434cf87c07179dd7689ea776f30b964cfb7ba6164af00c5111a45e69e57d885e1285f8dbfe3a21e95ae17cf676b0f8b7"
		prv, _ := hex.DecodeString(prvHex)
		id, err := IdentityFromBytes(prv)
		if err != nil {
			t.Fatalf("IdentityFromBytes: %v", err)
		}

		const appName = "rns_unit_tests"
		destOut, err := NewDestination(id, DestinationOUT, DestinationSINGLE, appName, "link", "establish")
		if err != nil {
			t.Fatalf("NewDestination(out): %v", err)
		}
		_, err = NewDestination(id, DestinationIN, DestinationSINGLE, appName, "link", "establish")
		if err != nil {
			t.Fatalf("NewDestination(in): %v", err)
		}

		_, err = NewOutgoingLink(destOut, LinkModeAES128CBC, nil, nil)
		if err == nil {
			t.Fatalf("expected AES128CBC link mode to be disabled for Python parity")
		}
		if !strings.Contains(err.Error(), "disabled") {
			t.Fatalf("expected disabled error, got: %v", err)
		}
	})
}

func TestIntegration_LinkPackets_WithReceipts(t *testing.T) {
	requireIntegration(t)
	resetKnownDestinationsForTest()
	withIntegrationTransport(t, func() {
		prvHex := "f8953ffaf607627e615603ff1530c82c434cf87c07179dd7689ea776f30b964cfb7ba6164af00c5111a45e69e57d885e1285f8dbfe3a21e95ae17cf676b0f8b7"
		prv, _ := hex.DecodeString(prvHex)
		id, err := IdentityFromBytes(prv)
		if err != nil {
			t.Fatalf("IdentityFromBytes: %v", err)
		}

		const appName = "rns_unit_tests"
		destOut, err := NewDestination(id, DestinationOUT, DestinationSINGLE, appName, "link", "establish")
		if err != nil {
			t.Fatalf("NewDestination(out): %v", err)
		}
		_, err = NewDestination(id, DestinationIN, DestinationSINGLE, appName, "link", "establish")
		if err != nil {
			t.Fatalf("NewDestination(in): %v", err)
		}

		l, err := NewOutgoingLink(destOut, LinkModeDefault, nil, nil)
		if err != nil {
			t.Fatalf("NewOutgoingLink: %v", err)
		}
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if l.Status == LinkActive {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		if l.Status != LinkActive {
			t.Fatalf("expected link active, got status %d", l.Status)
		}

		numPackets := 25
		if testing.Short() {
			numPackets = 10
		}
		packetSize := l.MDU
		if packetSize <= 0 {
			packetSize = 64
		}

		receipts := make([]*PacketReceipt, 0, numPackets)
		for i := 0; i < numPackets; i++ {
			data := make([]byte, packetSize)
			_, _ = rand.Read(data)
			pkt := NewPacket(l, data)
			if pkt == nil {
				t.Fatalf("NewPacket returned nil")
			}
			rc := pkt.Send()
			if rc == nil {
				t.Fatalf("expected receipt")
			}
			receipts = append(receipts, rc)
		}

		waitUntil := time.Now().Add(3 * time.Second)
		for time.Now().Before(waitUntil) {
			allOK := true
			for _, r := range receipts {
				if r == nil || r.Status != ReceiptDelivered {
					allOK = false
					break
				}
			}
			if allOK {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}

		for _, r := range receipts {
			if r == nil || r.Status != ReceiptDelivered {
				t.Fatalf("receipt not delivered")
			}
		}

		l.Teardown()
	})
}

func TestIntegration_BufferRoundTrip_Small(t *testing.T) {
	requireIntegration(t)
	resetKnownDestinationsForTest()
	withIntegrationTransport(t, func() {
		prvHex := "f8953ffaf607627e615603ff1530c82c434cf87c07179dd7689ea776f30b964cfb7ba6164af00c5111a45e69e57d885e1285f8dbfe3a21e95ae17cf676b0f8b7"
		prv, _ := hex.DecodeString(prvHex)
		id, err := IdentityFromBytes(prv)
		if err != nil {
			t.Fatalf("IdentityFromBytes: %v", err)
		}

		const appName = "rns_unit_tests"
		destOut, err := NewDestination(id, DestinationOUT, DestinationSINGLE, appName, "link", "establish")
		if err != nil {
			t.Fatalf("NewDestination(out): %v", err)
		}
		_, err = NewDestination(id, DestinationIN, DestinationSINGLE, appName, "link", "establish")
		if err != nil {
			t.Fatalf("NewDestination(in): %v", err)
		}

		l, err := NewOutgoingLink(destOut, LinkModeDefault, nil, nil)
		if err != nil {
			t.Fatalf("NewOutgoingLink: %v", err)
		}
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if l.Status == LinkActive {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		if l.Status != LinkActive {
			t.Fatalf("expected link active, got status %d", l.Status)
		}

		peer := findPeerLinkTest(l)
		if peer == nil {
			t.Fatalf("expected peer link")
		}

		// Responder side: echo back incoming data with " back at you".
		peerCh := peer.Channel()
		var peerBuf *ChannelBufferedReadWriter
		peerBuf = Buffer.CreateBidirectionalBuffer(0, 0, peerCh, func(readyBytes int) {
			data := make([]byte, readyBytes)
			n, rerr := peerBuf.Read(data)
			if rerr != nil || n == 0 {
				return
			}
			_, _ = peerBuf.Write(append(data[:n], []byte(" back at you")...))
			_ = peerBuf.Flush()
		})

		// Initiator side: collect received response.
		initCh := l.Channel()
		var (
			mu       sync.Mutex
			received [][]byte
		)
		var initBuf *ChannelBufferedReadWriter
		initBuf = Buffer.CreateBidirectionalBuffer(0, 0, initCh, func(readyBytes int) {
			data := make([]byte, readyBytes)
			n, rerr := initBuf.Read(data)
			if rerr != nil || n == 0 {
				return
			}
			mu.Lock()
			received = append(received, append([]byte(nil), data[:n]...))
			mu.Unlock()
		})

		_, _ = initBuf.Write([]byte("Hi there"))
		_ = initBuf.Flush()

		waitUntil := time.Now().Add(2 * time.Second)
		for time.Now().Before(waitUntil) {
			mu.Lock()
			count := len(received)
			var msg []byte
			if count > 0 {
				msg = received[0]
			}
			mu.Unlock()
			if count == 1 && string(msg) == "Hi there back at you" {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}

		mu.Lock()
		defer mu.Unlock()
		if len(received) != 1 {
			t.Fatalf("expected 1 received chunk, got %d", len(received))
		}
		if string(received[0]) != "Hi there back at you" {
			t.Fatalf("unexpected response: %q", string(received[0]))
		}

		l.Teardown()
	})
}

func TestIntegration_BufferRoundTrip_Big(t *testing.T) {
	requireIntegration(t)
	t.Skip("TODO: channel/buffer large transfer parity (currently flaky in in-process harness)")
	if testing.Short() {
		t.Skip("skipping big buffer test in -short")
	}
	resetKnownDestinationsForTest()
	withIntegrationTransport(t, func() {
		prvHex := "f8953ffaf607627e615603ff1530c82c434cf87c07179dd7689ea776f30b964cfb7ba6164af00c5111a45e69e57d885e1285f8dbfe3a21e95ae17cf676b0f8b7"
		prv, _ := hex.DecodeString(prvHex)
		id, err := IdentityFromBytes(prv)
		if err != nil {
			t.Fatalf("IdentityFromBytes: %v", err)
		}

		const appName = "rns_unit_tests"
		destOut, err := NewDestination(id, DestinationOUT, DestinationSINGLE, appName, "link", "establish")
		if err != nil {
			t.Fatalf("NewDestination(out): %v", err)
		}
		_, err = NewDestination(id, DestinationIN, DestinationSINGLE, appName, "link", "establish")
		if err != nil {
			t.Fatalf("NewDestination(in): %v", err)
		}

		l, err := NewOutgoingLink(destOut, LinkModeDefault, nil, nil)
		if err != nil {
			t.Fatalf("NewOutgoingLink: %v", err)
		}
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if l.Status == LinkActive {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		if l.Status != LinkActive {
			t.Fatalf("expected link active, got status %d", l.Status)
		}

		peer := findPeerLinkTest(l)
		if peer == nil {
			t.Fatalf("expected peer link")
		}

		targetBytes := 32000
		msg := make([]byte, targetBytes)
		_, _ = rand.Read(msg)

		// Build expected response: insert " back at you" every MAX_DATA_LEN and at the end.
		maxDataLen := streamMaxDataLen(l.Channel())
		if maxDataLen <= 0 {
			t.Fatalf("invalid stream max data len")
		}
		suffix := []byte(" back at you")
		expected := make([]byte, 0, len(msg)+((len(msg)/maxDataLen)+2)*len(suffix))
		for i := 0; i < len(msg); i++ {
			if i > 0 && (i%maxDataLen) == 0 {
				expected = append(expected, suffix...)
			}
			expected = append(expected, msg[i])
		}
		expected = append(expected, suffix...)

		// Responder side: aggregate until full message received, then echo segments+suffix.
		peerCh := peer.Channel()
		if peerCh == nil {
			t.Fatalf("peer channel nil")
		}
		var (
			peerMu      sync.Mutex
			peerBuf     *ChannelBufferedReadWriter
			peerAcc     []byte
			peerChunks  [][]byte
			peerAccSize int
			peerRxCount int
			peerRxSizes []int
		)
		peerBuf = Buffer.CreateBidirectionalBuffer(0, 0, peerCh, func(readyBytes int) {
			// Debug: show progress if needed.
			_ = readyBytes
			if readyBytes <= 0 {
				return
			}
			data := make([]byte, readyBytes)
			n, rerr := peerBuf.Read(data)
			if rerr != nil || n == 0 {
				return
			}
			peerMu.Lock()
			peerRxCount++
			peerRxSizes = append(peerRxSizes, n)
			peerAcc = append(peerAcc, data[:n]...)
			peerChunks = append(peerChunks, append([]byte(nil), data[:n]...))
			peerAccSize = len(peerAcc)
			ready := peerAccSize >= len(msg)
			peerMu.Unlock()
			if !ready {
				return
			}
			peerMu.Lock()
			chunks := peerChunks
			peerChunks = nil
			peerAcc = nil
			peerAccSize = 0
			peerMu.Unlock()
			for _, c := range chunks {
				_, _ = peerBuf.Write(append(c, suffix...))
				_ = peerBuf.Flush()
			}
		})

		// Initiator side: collect response bytes.
		initCh := l.Channel()
		if initCh == nil {
			t.Fatalf("init channel nil")
		}
		var (
			initMu      sync.Mutex
			initBuf     *ChannelBufferedReadWriter
			got         []byte
			initRxCount int
		)
		initBuf = Buffer.CreateBidirectionalBuffer(0, 0, initCh, func(readyBytes int) {
			data := make([]byte, readyBytes)
			n, rerr := initBuf.Read(data)
			if rerr != nil || n == 0 {
				return
			}
			initMu.Lock()
			initRxCount++
			got = append(got, data[:n]...)
			initMu.Unlock()
		})

		_, _ = initBuf.Write(msg)
		_ = initBuf.Flush()

		waitUntil := time.Now().Add(5 * time.Second)
		for time.Now().Before(waitUntil) {
			initMu.Lock()
			done := len(got) >= len(expected)
			initMu.Unlock()
			if done {
				break
			}
			peerMu.Lock()
			peerSeen := peerAccSize
			peerCnt := peerRxCount
			peerMu.Unlock()
			initMu.Lock()
			initCnt := initRxCount
			initMu.Unlock()
			_ = initCnt
			_ = peerCnt
			_ = peerSeen
			time.Sleep(50 * time.Millisecond)
		}

		initMu.Lock()
		defer initMu.Unlock()
		if len(got) != len(expected) {
			t.Fatalf("length mismatch: got %d want %d", len(got), len(expected))
		}
		for i := range expected {
			if got[i] != expected[i] {
				t.Fatalf("byte mismatch at %d", i)
			}
		}

		l.Teardown()
	})
}

func TestIntegration_BufferRoundTrip_Big_Slow(t *testing.T) {
	requireIntegration(t)
	if os.Getenv("RUN_SLOW_TESTS") == "" {
		t.Skip("set RUN_SLOW_TESTS=1 to run slow buffer test")
	}
	if testing.Short() {
		t.Skip("skipping slow buffer test in -short")
	}
	resetKnownDestinationsForTest()
	prev := Transport
	base := &integrationTransport{}
	Transport = &integrationTransportDelayed{base: base, bitrate: 410}
	defer func() { Transport = prev }()

	prvHex := "f8953ffaf607627e615603ff1530c82c434cf87c07179dd7689ea776f30b964cfb7ba6164af00c5111a45e69e57d885e1285f8dbfe3a21e95ae17cf676b0f8b7"
	prv, _ := hex.DecodeString(prvHex)
	id, err := IdentityFromBytes(prv)
	if err != nil {
		t.Fatalf("IdentityFromBytes: %v", err)
	}

	const appName = "rns_unit_tests"
	destOut, err := NewDestination(id, DestinationOUT, DestinationSINGLE, appName, "link", "establish")
	if err != nil {
		t.Fatalf("NewDestination(out): %v", err)
	}
	_, err = NewDestination(id, DestinationIN, DestinationSINGLE, appName, "link", "establish")
	if err != nil {
		t.Fatalf("NewDestination(in): %v", err)
	}

	l, err := NewOutgoingLink(destOut, LinkModeDefault, nil, nil)
	if err != nil {
		t.Fatalf("NewOutgoingLink: %v", err)
	}
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if l.Status == LinkActive {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if l.Status != LinkActive {
		t.Fatalf("expected link active, got status %d", l.Status)
	}

	peer := findPeerLinkTest(l)
	if peer == nil {
		t.Fatalf("expected peer link")
	}
	// A very low bitrate implies per-frame delays in seconds; increase RTT/timeout
	// factors so the channel does not tear down due to retries.
	l.RTT = 5 * time.Second
	l.TrafficTimeoutFactor = 20
	peer.RTT = 5 * time.Second
	peer.TrafficTimeoutFactor = 20

	targetBytes := 3000 // Python parity for bitrate < 1000 bps
	msg := make([]byte, targetBytes)
	_, _ = rand.Read(msg)

	maxDataLen := streamMaxDataLen(l.Channel())
	if maxDataLen <= 0 {
		t.Fatalf("invalid stream max data len")
	}
	suffix := []byte(" back at you")
	expected := make([]byte, 0, len(msg)+((len(msg)/maxDataLen)+2)*len(suffix))
	for i := 0; i < len(msg); i++ {
		if i > 0 && (i%maxDataLen) == 0 {
			expected = append(expected, suffix...)
		}
		expected = append(expected, msg[i])
	}
	expected = append(expected, suffix...)

	peerCh := peer.Channel()
	var (
		peerMu     sync.Mutex
		peerBuf    *ChannelBufferedReadWriter
		peerAcc    []byte
		peerChunks [][]byte
	)
	peerBuf = Buffer.CreateBidirectionalBuffer(0, 0, peerCh, func(readyBytes int) {
		if readyBytes <= 0 {
			return
		}
		data := make([]byte, readyBytes)
		n, rerr := peerBuf.Read(data)
		if rerr != nil || n == 0 {
			return
		}
		peerMu.Lock()
		peerAcc = append(peerAcc, data[:n]...)
		peerChunks = append(peerChunks, append([]byte(nil), data[:n]...))
		ready := len(peerAcc) >= len(msg)
		peerMu.Unlock()
		if !ready {
			return
		}
		peerMu.Lock()
		chunks := peerChunks
		peerChunks = nil
		peerAcc = nil
		peerMu.Unlock()
		for _, c := range chunks {
			_, _ = peerBuf.Write(append(c, suffix...))
			_ = peerBuf.Flush()
		}
	})

	initCh := l.Channel()
	var (
		initMu  sync.Mutex
		initBuf *ChannelBufferedReadWriter
		got     []byte
	)
	initBuf = Buffer.CreateBidirectionalBuffer(0, 0, initCh, func(readyBytes int) {
		data := make([]byte, readyBytes)
		n, rerr := initBuf.Read(data)
		if rerr != nil || n == 0 {
			return
		}
		initMu.Lock()
		got = append(got, data[:n]...)
		initMu.Unlock()
	})

	_, _ = initBuf.Write(msg)
	_ = initBuf.Flush()

	waitUntil := time.Now().Add(180 * time.Second)
	for time.Now().Before(waitUntil) {
		initMu.Lock()
		done := len(got) >= len(expected)
		initMu.Unlock()
		if done {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	initMu.Lock()
	defer initMu.Unlock()
	if len(got) != len(expected) {
		t.Fatalf("length mismatch: got %d want %d", len(got), len(expected))
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("byte mismatch at %d", i)
		}
	}

	l.Teardown()
}
