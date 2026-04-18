package main

import (
	"os"
	"testing"

	rns "github.com/svanichkin/go-reticulum/rns"
)

func requireIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("RNS_INTEGRATION") == "" {
		t.Skip("set RNS_INTEGRATION=1 to run integration tests")
	}
}

func TestIntegration_Announce_InboundDispatchesHandlers(t *testing.T) {
	requireIntegration(t)

	// Mutates global transport state in rns package; do not run in parallel.
	prevDestinations := rns.Destinations
	prevTransportID := rns.TransportIdentity

	t.Cleanup(func() {
		rns.Destinations = prevDestinations
		rns.TransportIdentity = prevTransportID
	})

	// Ensure inbound processing is active.
	id, err := rns.NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	rns.TransportIdentity = id

	// Create an announce packet, but prevent it from being treated as a local destination.
	// We do that by temporarily clearing the global destination registry.
	appID, err := rns.NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(app): %v", err)
	}
	dest, err := rns.NewDestination(appID, rns.DestinationIN, rns.DestinationSINGLE, appName, "announcesample", "fruits")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}
	announce := dest.Announce([]byte("Peach"), false, nil, nil, false)
	if announce == nil {
		t.Fatalf("Announce returned nil")
	}
	if err := announce.Pack(); err != nil {
		t.Fatalf("Pack: %v", err)
	}

	rns.Destinations = nil

	h := &testAnnounceHandler{filter: "example_utilities.announcesample.fruits"}
	rns.RegisterAnnounceHandler(h)
	t.Cleanup(func() { rns.DeregisterAnnounceHandler(h) })

	ifc := &rns.Interface{Name: "if0", IN: true, OUT: true, IngressControl: false}
	rns.Inbound(append([]byte(nil), announce.Raw...), ifc)

	if h.calls != 1 {
		t.Fatalf("expected handler calls=1 got %d", h.calls)
	}
	if string(h.last.appData) != "Peach" {
		t.Fatalf("expected appData=Peach got %q", string(h.last.appData))
	}
}
