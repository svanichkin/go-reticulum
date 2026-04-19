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
	if _, err := rns.NewReticulum(nil, nil, nil, nil, false, nil); err != nil {
		t.Fatalf("NewReticulum: %v", err)
	}
	id, err := rns.NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	rns.TransportIdentity = id

	// Create an announce packet, but prevent it from being treated as a local destination.
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

	h := &testAnnounceHandler{filter: "example_utilities.announcesample.fruits"}
	rns.RegisterAnnounceHandler(h)
	t.Cleanup(func() { rns.DeregisterAnnounceHandler(h) })

	dispatchAnnounceForExampleTest(t, announce, appID, h)
	if h.calls != 1 {
		t.Fatalf("expected handler calls=1 got %d", h.calls)
	}
}
