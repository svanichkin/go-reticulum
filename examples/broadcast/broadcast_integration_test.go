package main

import (
	"os"
	"testing"
	"time"

	rns "github.com/svanichkin/go-reticulum/rns"
)

func requireIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("RNS_INTEGRATION") == "" {
		t.Skip("set RNS_INTEGRATION=1 to run integration tests")
	}
}

func TestIntegration_Broadcast_InboundDeliversToPlainDestination(t *testing.T) {
	requireIntegration(t)

	// Mutates global transport state.
	prevDestinations := rns.Destinations
	prevTransportID := rns.TransportIdentity
	t.Cleanup(func() {
		rns.Destinations = prevDestinations
		rns.TransportIdentity = prevTransportID
	})

	if _, err := rns.NewReticulum(nil, nil, nil, nil, false, nil); err != nil {
		t.Fatalf("NewReticulum: %v", err)
	}
	id, err := rns.NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	rns.Destinations = nil
	rns.TransportIdentity = id

	dest, err := rns.NewDestination(nil, rns.DestinationIN, rns.DestinationPLAIN, appName, "broadcast", "public_information")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}

	got := make(chan string, 1)
	dest.SetPacketCallback(func(data []byte, _ *rns.Packet) {
		got <- string(data)
	})

	pkt := rns.NewPacket(dest, []byte("hello"), rns.PacketTypeData, rns.PacketCtxNone, rns.Broadcast, rns.HeaderType1, nil, nil, true, rns.FlagUnset)
	if pkt == nil {
		t.Fatalf("NewPacket returned nil")
	}
	if err := pkt.Pack(); err != nil {
		t.Fatalf("Pack: %v", err)
	}

	if ok := dest.Receive(pkt); !ok {
		t.Fatalf("Receive returned false")
	}

	select {
	case v := <-got:
		if v != "hello" {
			t.Fatalf("expected hello got %q", v)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("expected callback")
	}
}
