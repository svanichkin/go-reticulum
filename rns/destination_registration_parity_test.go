package rns

import (
	"bytes"
	"math"
	"testing"
	"time"

	cryptography "github.com/svanichkin/go-reticulum/rns/cryptography"
)

func TestRegisterDestination_SetsMTUAndSkipsOutboundRegistration(t *testing.T) {
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
					if dst.MTU == 0 || dst.MTU == prevPlain {
						dst.MTU = newPlain
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
				if dst.MTU == 0 || dst.MTU == prevPlain {
					dst.MTU = newPlain
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

	prevDestinations := Destinations
	t.Cleanup(func() { Destinations = prevDestinations })
	Destinations = nil

	d := &Destination{Direction: DestinationOUT, Type: DestinationSINGLE}
	if err := RegisterDestination(d); err != nil {
		t.Fatalf("RegisterDestination: %v", err)
	}

	if d.MTU != MTU {
		t.Fatalf("destination mtu=%d, want %d", d.MTU, MTU)
	}
	if len(Destinations) != 0 {
		t.Fatalf("destinations len=%d, want 0 for outbound destination", len(Destinations))
	}
}

func TestRegisterDestination_DuplicateHashPanics(t *testing.T) {
	prevDestinations := Destinations
	t.Cleanup(func() { Destinations = prevDestinations })
	Destinations = nil

	hash := bytes.Repeat([]byte{0x21}, truncatedHashBytes)
	first := &Destination{Direction: DestinationIN, Type: DestinationSINGLE, Hash: append([]byte(nil), hash...)}
	second := &Destination{Direction: DestinationIN, Type: DestinationSINGLE, Hash: append([]byte(nil), hash...)}

	if err := RegisterDestination(first); err != nil {
		t.Fatalf("RegisterDestination(first): %v", err)
	}
	if first.MTU != MTU {
		t.Fatalf("first destination mtu=%d, want %d", first.MTU, MTU)
	}

	if err := RegisterDestination(second); err == nil {
		t.Fatal("RegisterDestination did not return error on duplicate hash")
	}
	if second.MTU != MTU {
		t.Fatalf("second destination mtu=%d, want %d", second.MTU, MTU)
	}
}

func TestRegisterDestination_SharedClientRegistrationDoesNotAutoAnnounce(t *testing.T) {
	prevOwner := Owner
	prevInterfaces := Interfaces
	prevDestinations := Destinations

	clientIfc := &Interface{
		Name:                "shared-client",
		Type:                "LocalInterface",
		IN:                  true,
		OUT:                 true,
		LocalIsSharedClient: true,
	}
	sink := &outboundCapture{}
	attachOutboundCapture(t, sink, clientIfc)

	Owner = &Reticulum{IsConnectedToSharedInstance: true, SharedInstanceInterface: clientIfc}
	Interfaces = []*Interface{clientIfc}
	Destinations = nil

	t.Cleanup(func() {
		Owner = prevOwner
		Interfaces = prevInterfaces
		Destinations = prevDestinations
	})

	id, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	if _, err := NewDestination(id, DestinationIN, DestinationSINGLE, "test", "shared", "register"); err != nil {
		t.Fatalf("NewDestination: %v", err)
	}

	time.Sleep(400 * time.Millisecond)

	if got := len(sink.packets); got != 0 {
		t.Fatalf("automatic shared-client announces=%d, want 0", got)
	}
}

func TestDeregisterDestination_RemovesExactObjectOnly(t *testing.T) {
	prevDestinations := Destinations
	t.Cleanup(func() { Destinations = prevDestinations })

	first := &Destination{Direction: DestinationIN, Type: DestinationSINGLE, Hash: bytes.Repeat([]byte{0x31}, truncatedHashBytes)}
	second := &Destination{Direction: DestinationIN, Type: DestinationSINGLE, Hash: bytes.Repeat([]byte{0x31}, truncatedHashBytes)}
	third := &Destination{Direction: DestinationIN, Type: DestinationSINGLE, Hash: bytes.Repeat([]byte{0x32}, truncatedHashBytes)}

	Destinations = []*Destination{first, second, third}

	DeregisterDestination(first)

	if len(Destinations) != 2 {
		t.Fatalf("destinations len=%d, want 2", len(Destinations))
	}
	if Destinations[0] != second || Destinations[1] != third {
		t.Fatalf("unexpected destination order after deregister: %#v", Destinations)
	}

	DeregisterDestination(first)
	if len(Destinations) != 2 {
		t.Fatalf("destinations len=%d, want 2 after removing missing object", len(Destinations))
	}
}
