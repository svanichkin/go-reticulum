package rns

import (
	"bytes"
	"testing"
	"time"
)

func TestRegisterDestination_SetsMTUAndSkipsOutboundRegistration(t *testing.T) {
	prevMTU := MTU
	t.Cleanup(func() { _ = SetMTU(prevMTU) })

	if err := SetMTU(700); err != nil {
		t.Fatalf("SetMTU: %v", err)
	}

	prevDestinations := Destinations
	t.Cleanup(func() { Destinations = prevDestinations })
	Destinations = nil

	d := &Destination{Direction: DestinationOUT, Type: DestinationSINGLE}
	if err := RegisterDestination(d); err != nil {
		t.Fatalf("RegisterDestination: %v", err)
	}

	if d.mtu != MTU {
		t.Fatalf("destination mtu=%d, want %d", d.mtu, MTU)
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
	if first.mtu != MTU {
		t.Fatalf("first destination mtu=%d, want %d", first.mtu, MTU)
	}

	if err := RegisterDestination(second); err == nil {
		t.Fatal("RegisterDestination did not return error on duplicate hash")
	}
	if second.mtu != MTU {
		t.Fatalf("second destination mtu=%d, want %d", second.mtu, MTU)
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
