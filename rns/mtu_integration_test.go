package rns

import (
	"encoding/hex"
	"testing"
	"time"
)

func TestIntegration_MTU_PropagatesToLinkAndResource(t *testing.T) {
	requireIntegration(t)
	resetKnownDestinationsForTest()

	prevMTU := MTU
	t.Cleanup(func() { _ = SetMTU(prevMTU) })

	if err := SetMTU(700); err != nil {
		t.Fatalf("SetMTU: %v", err)
	}

	withIntegrationTransport(t, func() {
		prvHex := "f8953ffaf607627e615603ff1530c82c434cf87c07179dd7689ea776f30b964cfb7ba6164af00c5111a45e69e57d885e1285f8dbfe3a21e95ae17cf676b0f8b7"
		prv, _ := hex.DecodeString(prvHex)
		id, err := IdentityFromBytes(prv)
		if err != nil {
			t.Fatalf("IdentityFromBytes: %v", err)
		}

		const appName = "rns_unit_tests"
		destOut, err := NewDestination(id, DestinationOUT, DestinationSINGLE, appName, "mtu", "test")
		if err != nil {
			t.Fatalf("NewDestination(out): %v", err)
		}
		_, err = NewDestination(id, DestinationIN, DestinationSINGLE, appName, "mtu", "test")
		if err != nil {
			t.Fatalf("NewDestination(in): %v", err)
		}

		l, err := NewLink(destOut, nil, LinkModeDefault, nil, nil)
		if err != nil {
			t.Fatalf("NewOutgoingLink: %v", err)
		}
		waitForLinkActiveOrSkip(t, l, 2*time.Second)
		if l.MTU != MTU {
			t.Fatalf("expected link MTU=%d, got %d", MTU, l.MTU)
		}
		if l.MDU != LinkMDU {
			t.Fatalf("expected link MDU=%d, got %d", LinkMDU, l.MDU)
		}

		// Sanity: send packets at the new MDU and ensure receipts are delivered.
		numPackets := 5
		packetSize := l.MDU
		if packetSize <= 0 {
			packetSize = 64
		}
		receipts := make([]*PacketReceipt, 0, numPackets)
		for i := 0; i < numPackets; i++ {
			data := make([]byte, packetSize)
			pkt := NewPacket(l, data, PacketTypeData, PacketCtxNone, Broadcast, HeaderType1, nil, nil, true, FlagUnset)
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
