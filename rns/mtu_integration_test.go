package rns

import (
	"encoding/hex"
	"math"
	"testing"
	"time"

	cryptography "github.com/svanichkin/go-reticulum/rns/cryptography"
)

func TestIntegration_MTU_PropagatesToLinkAndResource(t *testing.T) {
	requireIntegration(t)
	resetKnownDestinationsForTest()

	prevMTU := MTU
	t.Cleanup(func() {
		mtuMu.Lock()
		prevPlain := PacketPlainMDU
		prevEncrypted := PacketEncryptedMDU
		prevLink := LinkMDU
		MTU = prevMTU
		plain := MTU - HEADER_MAXSIZE - IFAC_MIN_SIZE
		if plain < 0 {
			plain = 0
		}
		link := func() int {
			payload := MTU - IFAC_MIN_SIZE - HEADER_MINSIZE - cryptography.Overhead
			if payload <= 0 {
				return 0
			}
			blocks := payload / identityAESBlockSize
			if blocks <= 0 {
				return 0
			}
			return blocks*identityAESBlockSize - 1
		}()
		hashLen := int(math.Floor(float64(link-AdvOverhead) / float64(MapHashLen)))
		encrypted := func() int {
			usable := plain - cryptography.Overhead - (identityPubKeyLen / 2)
			if usable <= 0 {
				return 0
			}
			blocks := usable / identityAESBlockSize
			if blocks <= 0 {
				return 0
			}
			return blocks*identityAESBlockSize - 1
		}()
		MDU = plain
		PacketMDU = plain
		PacketPLAIN_MDU = plain
		PacketENCRYPTED_MDU = encrypted
		PacketPlainMDU = plain
		PacketEncryptedMDU = encrypted
		LinkMDU = link
		HashmapMaxLen = hashLen
		CollisionGuardSize = 2*ResourceWindowMax + HashmapMaxLen
		if plain != prevPlain || encrypted != prevEncrypted || link != prevLink {
			go func(prevPlain, newPlain, prevLink, newLink int) {
				for _, dst := range Destinations {
					if dst == nil {
						continue
					}
					if dst.mtu == 0 || dst.mtu == prevPlain {
						dst.mtu = newPlain
					}
				}
				linkMu.Lock()
				for _, l := range PendingLinks {
					if l == nil {
						continue
					}
					if prevLink == 0 || l.MTU == prevLink {
						l.MTU = newLink
						l.updateMDU()
					}
				}
				for _, l := range ActiveLinks {
					if l == nil {
						continue
					}
					if prevLink == 0 || l.MTU == prevLink {
						l.MTU = newLink
						l.updateMDU()
					}
				}
				linkMu.Unlock()
			}(prevPlain, plain, prevLink, link)
		}
		mtuMu.Unlock()
	})

	mtuMu.Lock()
	prevPlain := PacketPlainMDU
	prevEncrypted := PacketEncryptedMDU
	prevLink := LinkMDU
	MTU = 700
	plain := MTU - HEADER_MAXSIZE - IFAC_MIN_SIZE
	if plain < 0 {
		plain = 0
	}
	link := func() int {
		payload := MTU - IFAC_MIN_SIZE - HEADER_MINSIZE - cryptography.Overhead
		if payload <= 0 {
			return 0
		}
		blocks := payload / identityAESBlockSize
		if blocks <= 0 {
			return 0
		}
		return blocks*identityAESBlockSize - 1
	}()
	hashLen := int(math.Floor(float64(link-AdvOverhead) / float64(MapHashLen)))
	encrypted := func() int {
		usable := plain - cryptography.Overhead - (identityPubKeyLen / 2)
		if usable <= 0 {
			return 0
		}
		blocks := usable / identityAESBlockSize
		if blocks <= 0 {
			return 0
		}
		return blocks*identityAESBlockSize - 1
	}()
	MDU = plain
	PacketMDU = plain
	PacketPLAIN_MDU = plain
	PacketENCRYPTED_MDU = encrypted
	PacketPlainMDU = plain
	PacketEncryptedMDU = encrypted
	LinkMDU = link
	HashmapMaxLen = hashLen
	CollisionGuardSize = 2*ResourceWindowMax + HashmapMaxLen
	if plain != prevPlain || encrypted != prevEncrypted || link != prevLink {
		go func(prevPlain, newPlain, prevLink, newLink int) {
			for _, dst := range Destinations {
				if dst == nil {
					continue
				}
				if dst.mtu == 0 || dst.mtu == prevPlain {
					dst.mtu = newPlain
				}
			}
			linkMu.Lock()
			for _, l := range PendingLinks {
				if l == nil {
					continue
				}
				if prevLink == 0 || l.MTU == prevLink {
					l.MTU = newLink
					l.updateMDU()
				}
			}
			for _, l := range ActiveLinks {
				if l == nil {
					continue
				}
				if prevLink == 0 || l.MTU == prevLink {
					l.MTU = newLink
					l.updateMDU()
				}
			}
			linkMu.Unlock()
		}(prevPlain, plain, prevLink, link)
	}
	mtuMu.Unlock()

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
				if r == nil || r.Status != PacketReceiptDELIVERED {
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
			if r == nil || r.Status != PacketReceiptDELIVERED {
				t.Fatalf("receipt not delivered")
			}
		}

		l.Teardown()
	})
}
