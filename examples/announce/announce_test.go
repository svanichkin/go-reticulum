package main

import (
	"bytes"
	"testing"

	rns "github.com/svanichkin/go-reticulum/rns"
)

type testAnnounceHandler struct {
	filter string
	calls  int
	last   struct {
		dstHash []byte
		appData []byte
		idHash  []byte
	}
}

func (h *testAnnounceHandler) AspectFilter() string { return h.filter }
func (h *testAnnounceHandler) ReceivedAnnounce(destinationHash []byte, announcedIdentity *rns.Identity, appData []byte) {
	h.calls++
	h.last.dstHash = append([]byte(nil), destinationHash...)
	h.last.appData = append([]byte(nil), appData...)
	if announcedIdentity != nil {
		h.last.idHash = append([]byte(nil), announcedIdentity.Hash...)
	}
}

func dispatchAnnounceForExampleTest(t *testing.T, pkt *rns.Packet, announcedIdentity *rns.Identity, handler *testAnnounceHandler) {
	t.Helper()
	if pkt == nil || handler == nil {
		return
	}
	if handler.filter != "" {
		expected, err := rns.DestinationHashFromNameAndIdentity(handler.filter, announcedIdentity)
		if err != nil || !bytes.Equal(expected, pkt.DestinationHash) {
			return
		}
	}
	handler.ReceivedAnnounce(pkt.DestinationHash, announcedIdentity, rns.IdentityRecallAppData(pkt.DestinationHash))
}

func TestAnnounceHandlers_FilterAndDispatch(t *testing.T) {
	// Global announce handler registry; don't run in parallel.
	id, err := rns.NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}

	const appName = "example_utilities"
	destFruits, err := rns.NewDestination(id, rns.DestinationIN, rns.DestinationSINGLE, appName, "announcesample", "fruits")
	if err != nil {
		t.Fatalf("NewDestination(fruits): %v", err)
	}
	destGas, err := rns.NewDestination(id, rns.DestinationIN, rns.DestinationSINGLE, appName, "announcesample", "noble_gases")
	if err != nil {
		t.Fatalf("NewDestination(gases): %v", err)
	}

	fruitData := []byte("Peach")
	pktFruit := destFruits.Announce(fruitData, false, nil, nil, false)
	if pktFruit == nil {
		t.Fatalf("Announce(fruits) returned nil")
	}
	if err := pktFruit.Pack(); err != nil {
		t.Fatalf("Pack(fruits): %v", err)
	}
	if !rns.ValidateAnnounce(pktFruit, false) {
		t.Fatalf("ValidateAnnounce failed for fruits packet")
	}

	pktGas := destGas.Announce([]byte("Neon"), false, nil, nil, false)
	if pktGas == nil {
		t.Fatalf("Announce(gases) returned nil")
	}
	if err := pktGas.Pack(); err != nil {
		t.Fatalf("Pack(gases): %v", err)
	}
	if !rns.ValidateAnnounce(pktGas, false) {
		t.Fatalf("ValidateAnnounce failed for gases packet")
	}

	hAll := &testAnnounceHandler{filter: ""}
	hFruits := &testAnnounceHandler{filter: "example_utilities.announcesample.fruits"}

	rns.RegisterAnnounceHandler(hAll)
	rns.RegisterAnnounceHandler(hFruits)
	t.Cleanup(func() {
		rns.DeregisterAnnounceHandler(hAll)
		rns.DeregisterAnnounceHandler(hFruits)
	})

	dispatchAnnounceForExampleTest(t, pktFruit, id, hAll)
	dispatchAnnounceForExampleTest(t, pktFruit, id, hFruits)
	dispatchAnnounceForExampleTest(t, pktGas, id, hAll)
	dispatchAnnounceForExampleTest(t, pktGas, id, hFruits)

	if hAll.calls != 2 {
		t.Fatalf("expected hAll calls=2 got %d", hAll.calls)
	}
	if hFruits.calls != 1 {
		t.Fatalf("expected hFruits calls=1 got %d", hFruits.calls)
	}
	if string(hFruits.last.appData) != string(fruitData) {
		t.Fatalf("expected appData=%q got %q", fruitData, hFruits.last.appData)
	}
	if len(hFruits.last.dstHash) != rns.ReticulumTruncatedHashLength/8 {
		t.Fatalf("expected dst hash length %d got %d", rns.ReticulumTruncatedHashLength/8, len(hFruits.last.dstHash))
	}
	if len(hFruits.last.idHash) == 0 {
		t.Fatalf("expected announced identity hash")
	}
}
