package main

import (
	"testing"

	rns "github.com/svanichkin/go-reticulum/rns"
)

func TestBroadcast_PlainDestinationReceivesPacket(t *testing.T) {
	// No t.Parallel: uses package-level singletons in rns in other tests.
	prevDestinations := rns.Destinations
	rns.Destinations = nil
	t.Cleanup(func() { rns.Destinations = prevDestinations })

	dest, err := rns.NewDestination(nil, rns.DestinationIN, rns.DestinationPLAIN, appName, "broadcast", "public_information")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}

	got := make(chan string, 1)
	dest.SetPacketCallback(func(data []byte, _ *rns.Packet) {
		got <- string(data)
	})

	pkt := rns.NewPacket(dest, []byte("hello"))
	if pkt == nil {
		t.Fatalf("NewPacket returned nil")
	}
	if err := pkt.Pack(); err != nil {
		t.Fatalf("Pack: %v", err)
	}

	// Deliver directly to destination handler.
	if ok := dest.Receive(pkt); !ok {
		t.Fatalf("Receive returned false")
	}

	select {
	case v := <-got:
		if v != "hello" {
			t.Fatalf("expected hello got %q", v)
		}
	default:
		t.Fatalf("expected callback")
	}
}
