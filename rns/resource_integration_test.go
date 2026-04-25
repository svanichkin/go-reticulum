package rns

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"sync"
	"testing"
	"time"
)

func TestIntegration_Resource_MicroMiniSmall(t *testing.T) {
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

		l, err := NewLink(destOut, nil, LinkModeDefault, nil, nil)
		if err != nil {
			t.Fatalf("NewOutgoingLink: %v", err)
		}
		waitForLinkActiveOrSkip(t, l, 2*time.Second)
		peer := findPeerLinkTest(l)
		if peer == nil {
			t.Fatalf("expected peer link")
		}
		peer.SetResourceStrategy(LinkAcceptAll)

		var (
			peerMu        sync.Mutex
			lastIncoming  *Resource
			lastData      []byte
			lastMeta      any
			incomingCount int
			started       *Resource
		)
		peer.SetResourceStartedCallback(func(res *Resource) {
			peerMu.Lock()
			started = res
			peerMu.Unlock()
		})
		peer.SetResourceConcludedCallback(func(res *Resource) {
			var cp []byte
			var meta any
			if res != nil && res.Status == ResourceComplete {
				if data, err := os.ReadFile(res.DataFile); err == nil {
					cp = append([]byte(nil), data...)
				}
				meta = res.Metadata
			}
			peerMu.Lock()
			lastIncoming = res
			lastData = cp
			lastMeta = meta
			incomingCount++
			peerMu.Unlock()
		})

		sendAndVerify := func(size int, withMeta bool) {
			t.Helper()

			resetIntegrationTransportStats()
			peerMu.Lock()
			started = nil
			lastIncoming = nil
			lastData = nil
			lastMeta = nil
			peerMu.Unlock()

			data := make([]byte, size)
			_, _ = rand.Read(data)

			var meta any
			if withMeta {
				blob := make([]byte, 32)
				_, _ = rand.Read(blob)
				meta = map[string]any{
					"text":    "Some text",
					"numbers": []any{1, 2, 3, 4},
					"blob":    blob,
				}
			}

			timeoutSeconds := 120.0
			res, err := NewResource(
				data,
				nil,
				l,
				meta,
				true,
				true,
				nil,
				nil,
				&timeoutSeconds,
				0,
				nil,
				nil,
				false,
				0,
			)
			if err != nil || res == nil {
				t.Fatalf("NewResource(size=%d, meta=%v): %v", size, withMeta, err)
			}
			// Allow a short grace period for the receiver to accept the advertisement and request parts.
			{
				deadline := time.Now().Add(5 * time.Second)
				for time.Now().Before(deadline) {
					peerMu.Lock()
					s := started
					peerMu.Unlock()
					if s != nil && len(s.hashmapRaw) > 0 && s.hashmapHeight > 0 {
						break
					}
					time.Sleep(5 * time.Millisecond)
				}
			}

			waitUntil := time.Now().Add(30 * time.Second)
			for time.Now().Before(waitUntil) {
				if res.Status == ResourceComplete {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if res.Status != ResourceComplete {
				sentReq, sentData, delReq, delData := 0, 0, 0, 0
				delReqInit, delReqResp := 0, 0
				if it := getIntegrationTransport(); it != nil {
					it.mu.Lock()
					sentReq = it.sentByContext[PacketCtxResourceReq]
					sentData = it.sentByContext[PacketCtxResource]
					delReq = it.deliveredByContext[PacketCtxResourceReq]
					delData = it.deliveredByContext[PacketCtxResource]
					delReqInit = it.deliveredByContextToInitiator[PacketCtxResourceReq]
					delReqResp = it.deliveredByContextToResponder[PacketCtxResourceReq]
					it.mu.Unlock()
				}
				linkMTU := 0
				linkMDU := 0
				if res.Link != nil {
					linkMTU = res.Link.MTU
					linkMDU = res.Link.MDU
				}
				peerMu.Lock()
				s := started
				peerMu.Unlock()
				peerHash := 0
				peerParts := 0
				peerHashmapHeight := 0
				peerReqSent := false
				peerSize := 0
				peerSDU := 0
				peerOutstanding := 0
				peerReceived := 0
				peerConsecutive := 0
				peerAdvT := 0
				peerAdvN := 0
				peerAdvMLen := 0
				if s != nil {
					peerHash = len(s.hash)
					peerParts = s.totalParts
					peerHashmapHeight = s.hashmapHeight
					peerReqSent = !s.reqSent.IsZero()
					peerSize = s.size
					peerSDU = s.sdu
					peerOutstanding = s.outstanding
					peerReceived = s.receivedCount
					peerConsecutive = s.consecutiveHeight
				}
				if res.advPacket != nil {
					raw := res.advPacket.Data
					if len(raw) == 0 {
						raw = res.advPacket.Plaintext
					}
					if len(raw) > 0 {
						if adv, err := (ResourceAdvertisement{}).Unpack(raw); err == nil && adv != nil {
							peerAdvT = adv.T
							peerAdvN = adv.N
							peerAdvMLen = len(adv.M)
						}
					}
				}
				missing := 0
				if len(res.hashmap) > 0 && len(res.outgoingPartByMapHash) > 0 {
					maxCheck := 4
					if len(res.hashmap) < maxCheck {
						maxCheck = len(res.hashmap)
					}
					for i := 0; i < maxCheck; i++ {
						h := res.hashmap[i]
						if len(h) != MapHashLen {
							continue
						}
						if res.outgoingPartByMapHash[string(h)] == nil {
							missing++
						}
					}
				}
				t.Fatalf("resource status=%d want %d (sent_req=%d sent_data=%d delivered_req=%d delivered_data=%d delivered_req_init=%d delivered_req_resp=%d sentParts=%d totalParts=%d size=%d sdu=%d link.MTU=%d link.MDU=%d adv_t=%d adv_n=%d adv_m_len=%d sender_missing_first4=%d peer_hash_len=%d peer_size=%d peer_sdu=%d peer_totalParts=%d peer_hashmapHeight=%d peer_outstanding=%d peer_received=%d peer_consecutive=%d peer_reqSent=%v)", res.Status, ResourceComplete, sentReq, sentData, delReq, delData, delReqInit, delReqResp, res.sentParts, res.totalParts, res.size, res.sdu, linkMTU, linkMDU, peerAdvT, peerAdvN, peerAdvMLen, missing, peerHash, peerSize, peerSDU, peerParts, peerHashmapHeight, peerOutstanding, peerReceived, peerConsecutive, peerReqSent)
			}

			// The sender may mark complete slightly before the receiver callback fires.
			var pr *Resource
			{
				deadline := time.Now().Add(2 * time.Second)
				for time.Now().Before(deadline) {
					peerMu.Lock()
					pr = lastIncoming
					peerMu.Unlock()
					if pr != nil && pr.Status == ResourceComplete {
						break
					}
					time.Sleep(5 * time.Millisecond)
				}
			}
			if pr == nil || pr.Status != ResourceComplete {
				t.Fatalf("peer did not complete incoming resource (size=%d meta=%v)", size, withMeta)
			}
			peerMu.Lock()
			got := append([]byte(nil), lastData...)
			gotMeta := lastMeta
			peerMu.Unlock()
			if len(got) != len(data) {
				t.Fatalf("peer data length mismatch: got %d want %d", len(got), len(data))
			}
			for i := range data {
				if got[i] != data[i] {
					t.Fatalf("peer data mismatch at %d", i)
				}
			}

			if withMeta {
				if gotMeta == nil {
					t.Fatalf("expected metadata on peer")
				}
				switch m := gotMeta.(type) {
				case map[string]any:
					if m["text"] == nil {
						t.Fatalf("expected metadata key 'text'")
					}
				case map[any]any:
					if m["text"] == nil {
						t.Fatalf("expected metadata key 'text'")
					}
				}
			}
		}

		// Micro resource (128 B) + metadata + invalid metadata size.
		sendAndVerify(128, false)
		sendAndVerify(128, true)
		{
			data := make([]byte, 128)
			_, _ = rand.Read(data)
			tooBig := make([]byte, MetadataMaxSize+1)
			_, _ = rand.Read(tooBig)
			timeoutSeconds := 120.0
			if _, err := NewResource(
				data,
				nil,
				l,
				tooBig,
				true,
				true,
				nil,
				nil,
				&timeoutSeconds,
				0,
				nil,
				nil,
				false,
				0,
			); err == nil {
				t.Fatalf("expected metadata size error")
			}
		}

		// Mini resource (256 KB) + metadata.
		sendAndVerify(256*1000, false)
		sendAndVerify(256*1000, true)

		// Small resource (1 MB) + metadata.
		sendAndVerify(1000*1000, false)
		sendAndVerify(1000*1000, true)

		peerMu.Lock()
		cnt := incomingCount
		peerMu.Unlock()
		if cnt < 6 {
			t.Fatalf("expected at least 6 incoming resources, got %d", cnt)
		}

		l.Teardown()
	})
}

func TestIntegration_Resource_MediumLarge_Slow(t *testing.T) {
	requireIntegration(t)
	if os.Getenv("RUN_SLOW_TESTS") == "" {
		t.Skip("set RUN_SLOW_TESTS=1 to run medium/large resource integration tests")
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

		l, err := NewLink(destOut, nil, LinkModeDefault, nil, nil)
		if err != nil {
			t.Fatalf("NewOutgoingLink: %v", err)
		}
		waitForLinkActiveOrSkip(t, l, 2*time.Second)
		peer := findPeerLinkTest(l)
		if peer == nil {
			t.Fatalf("expected peer link")
		}
		peer.SetResourceStrategy(LinkAcceptAll)

		send := func(size int) {
			t.Helper()
			data := make([]byte, size)
			_, _ = rand.Read(data)
			timeoutSeconds := 120.0
			done := make(chan struct{})
			res, err := NewResource(
				data,
				nil,
				l,
				nil,
				true,
				false, // auto_compress=False (Python parity for medium/large)
				func(*Resource) { close(done) },
				nil,
				&timeoutSeconds,
				0,
				nil,
				nil,
				false,
				0,
			)
			if err != nil || res == nil {
				t.Fatalf("NewResource: %v", err)
			}
			select {
			case <-done:
			case <-time.After(120 * time.Second):
				t.Fatalf("timeout waiting resource completion (size=%d) status=%d", size, res.Status)
			}
			if res.Status != ResourceComplete {
				t.Fatalf("resource status=%d want %d", res.Status, ResourceComplete)
			}
		}

		send(5 * 1000 * 1000)
		send(50 * 1000 * 1000)

		l.Teardown()
	})
}

func TestIntegration_Resource_RejectStrategy(t *testing.T) {
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
		destOut, err := NewDestination(id, DestinationOUT, DestinationSINGLE, appName, "resource", "reject")
		if err != nil {
			t.Fatalf("NewDestination(out): %v", err)
		}
		_, err = NewDestination(id, DestinationIN, DestinationSINGLE, appName, "resource", "reject")
		if err != nil {
			t.Fatalf("NewDestination(in): %v", err)
		}

		l, err := NewLink(destOut, nil, LinkModeDefault, nil, nil)
		if err != nil {
			t.Fatalf("NewOutgoingLink: %v", err)
		}
		waitForLinkActiveOrSkip(t, l, 2*time.Second)

		peer := findPeerLinkTest(l)
		if peer == nil {
			t.Fatalf("expected peer link")
		}
		peer.SetResourceStrategy(LinkAcceptNone)

		done := make(chan *Resource, 1)
		data := make([]byte, 1024)
		_, _ = rand.Read(data)
		timeoutSeconds := 10.0
		res, err := NewResource(
			data,
			nil,
			l,
			nil,
			true,
			true,
			func(r *Resource) { done <- r },
			nil,
			&timeoutSeconds,
			0,
			nil,
			nil,
			false,
			0,
		)
		if err != nil || res == nil {
			t.Fatalf("NewResource: %v", err)
		}

		select {
		case r := <-done:
			if r == nil {
				t.Fatalf("callback returned nil resource")
			}
			if r.Status != ResourceRejected {
				t.Fatalf("expected resource rejected, got status=%d", r.Status)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timeout waiting resource rejection, status=%d", res.Status)
		}

		l.Teardown()
	})
}

func TestIntegration_Resource_ResponseToRequest_AsResource(t *testing.T) {
	requireIntegration(t)
	t.Skip("TODO: response-as-resource transfer parity (ResourceAdv/Req flow) not fully implemented yet")
	resetKnownDestinationsForTest()
	withIntegrationTransport(t, func() {
		prvHex := "f8953ffaf607627e615603ff1530c82c434cf87c07179dd7689ea776f30b964cfb7ba6164af00c5111a45e69e57d885e1285f8dbfe3a21e95ae17cf676b0f8b7"
		prv, _ := hex.DecodeString(prvHex)
		id, err := IdentityFromBytes(prv)
		if err != nil {
			t.Fatalf("IdentityFromBytes: %v", err)
		}

		const appName = "rns_unit_tests"
		destOut, err := NewDestination(id, DestinationOUT, DestinationSINGLE, appName, "resource", "response")
		if err != nil {
			t.Fatalf("NewDestination(out): %v", err)
		}
		destIn, err := NewDestination(id, DestinationIN, DestinationSINGLE, appName, "resource", "response")
		if err != nil {
			t.Fatalf("NewDestination(in): %v", err)
		}

		l, err := NewLink(destOut, nil, LinkModeDefault, nil, nil)
		if err != nil {
			t.Fatalf("NewOutgoingLink: %v", err)
		}
		waitForLinkActiveOrSkip(t, l, 2*time.Second)

		peer := findPeerLinkTest(l)
		if peer == nil {
			t.Fatalf("expected peer link")
		}
		peer.SetResourceStrategy(LinkAcceptAll)

		var expected []byte
		if err := destIn.RegisterRequestHandler(
			"/big",
			func(_ string, _ any, _ []byte, _ []byte, _ *Identity, _ time.Time) any {
				// Force a response resource by exceeding the MDU.
				payload := make([]byte, l.MDU+64)
				_, _ = rand.Read(payload)
				expected = payload
				return payload
			},
			DestinationALLOW_ALL,
			nil,
			true,
		); err != nil {
			t.Fatalf("RegisterRequestHandler: %v", err)
		}

		done := make(chan *RequestReceipt, 1)
		result := l.Request(
			"/big",
			map[string]any{"ping": "pong"},
			func(r *RequestReceipt) { done <- r },
			func(r *RequestReceipt) { done <- r },
			nil,
			5,
		)
		rr, _ := result.(*RequestReceipt)
		if rr == nil {
			t.Fatalf("Request returned nil")
		}

		select {
		case r := <-done:
			if r.GetStatus() != RequestReceiptReady {
				stats := getIntegrationTransport()
				if stats != nil {
					stats.mu.Lock()
					sent := make(map[byte]int, len(stats.sentByContext))
					for k, v := range stats.sentByContext {
						sent[k] = v
					}
					delivered := make(map[byte]int, len(stats.deliveredByContext))
					for k, v := range stats.deliveredByContext {
						delivered[k] = v
					}
					stats.mu.Unlock()
					t.Fatalf("expected receipt ready, got %d (progress=%.3f) sent=%v delivered=%v", r.GetStatus(), r.GetProgress(), sent, delivered)
				}
				t.Fatalf("expected receipt ready, got %d (progress=%.3f)", r.GetStatus(), r.GetProgress())
			}
			gotAny := r.GetResponse()
			got, ok := gotAny.([]byte)
			if !ok {
				t.Fatalf("expected []byte response, got %T", gotAny)
			}
			if len(got) != len(expected) {
				t.Fatalf("response size mismatch: got %d want %d", len(got), len(expected))
			}
			for i := range expected {
				if got[i] != expected[i] {
					t.Fatalf("response mismatch at %d", i)
				}
			}
			if r.ResponseTransferSize() == nil || *r.ResponseTransferSize() == 0 {
				t.Fatalf("expected non-zero response transfer size")
			}
		case <-time.After(10 * time.Second):
			var prStatus byte
			if rr != nil && rr.packetReceipt != nil {
				prStatus = rr.packetReceipt.Status
			}
			stats := getIntegrationTransport()
			if stats != nil {
				stats.mu.Lock()
				sent := make(map[byte]int, len(stats.sentByContext))
				for k, v := range stats.sentByContext {
					sent[k] = v
				}
				delivered := make(map[byte]int, len(stats.deliveredByContext))
				for k, v := range stats.deliveredByContext {
					delivered[k] = v
				}
				stats.mu.Unlock()
				t.Fatalf("timeout waiting request response, status=%d packet_receipt=%d progress=%.3f sent=%v delivered=%v", rr.GetStatus(), prStatus, rr.GetProgress(), sent, delivered)
			}
			t.Fatalf("timeout waiting request response, status=%d packet_receipt=%d progress=%.3f", rr.GetStatus(), prStatus, rr.GetProgress())
		}

		l.Teardown()
	})
}
