package interfaces

import (
	"bytes"
	"encoding/hex"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestWeaveIntegration_ProcessIncoming_SpawnedPeerInheritsIngressSettings(t *testing.T) {
	ifc, err := NewWeaveInterface("w0", map[string]string{"port": "/dev/null"})
	if err != nil {
		t.Fatalf("NewWeaveInterface: %v", err)
	}
	t.Cleanup(func() { ifc.weave.Close() })

	hold := 11.0
	maxHeld := 77
	ifc.IN = false
	ifc.OUT = true
	ifc.IngressControl = false
	ifc.ICBurstHold = &hold
	ifc.ICMaxHeldAnnounces = &maxHeld

	endpoint := []byte{0, 1, 2, 3, 4, 5, 6, 7}
	payload := []byte("hello")
	ifc.weave.ProcessIncoming(endpoint, payload)

	key := hex.EncodeToString(endpoint)
	ifc.weave.mu.Lock()
	peerState := ifc.weave.peers[key]
	ifc.weave.mu.Unlock()
	if peerState == nil || peerState.iface == nil {
		t.Fatalf("expected peer to exist")
	}
	peer := peerState.iface
	if peer.IngressControl != ifc.IngressControl {
		t.Fatalf("peer IngressControl=%v, want %v", peer.IngressControl, ifc.IngressControl)
	}
	if peer.IN != ifc.IN || peer.OUT != ifc.OUT {
		t.Fatalf("peer IN/OUT=(%v,%v), want (%v,%v)", peer.IN, peer.OUT, ifc.IN, ifc.OUT)
	}
	if peer.ICMaxHeldAnnounces == nil || *peer.ICMaxHeldAnnounces != maxHeld {
		t.Fatalf("peer ICMaxHeldAnnounces=%v, want %d", peer.ICMaxHeldAnnounces, maxHeld)
	}
	if peer.ICBurstHold == nil || *peer.ICBurstHold != hold {
		t.Fatalf("peer ICBurstHold=%v, want %v", peer.ICBurstHold, hold)
	}
}

func TestWeaveIntegration_ProcessIncoming_SpawnsPeerAndDelivers(t *testing.T) {
	// Do not run in parallel: overrides global hooks.
	oldInbound := InboundHandler
	t.Cleanup(func() { InboundHandler = oldInbound })

	got := make(chan struct {
		data []byte
		ifc  *Interface
	}, 1)
	InboundHandler = func(raw []byte, ifc *Interface) {
		got <- struct {
			data []byte
			ifc  *Interface
		}{data: append([]byte(nil), raw...), ifc: ifc}
	}

	ifc, err := NewWeaveInterface("w0", map[string]string{"port": "/dev/null"})
	if err != nil {
		t.Fatalf("NewWeaveInterface: %v", err)
	}
	t.Cleanup(func() { ifc.weave.Close() })

	endpoint := []byte{0, 1, 2, 3, 4, 5, 6, 7}
	payload := []byte("hello")
	ifc.weave.ProcessIncoming(endpoint, payload)

	select {
	case r := <-got:
		if !bytes.Equal(r.data, payload) {
			t.Fatalf("unexpected payload: %q", string(r.data))
		}
		if r.ifc == nil || !strings.EqualFold(r.ifc.Type, "WeaveInterfacePeer") {
			t.Fatalf("unexpected iface: %#v", r.ifc)
		}
		wantName := "WeaveInterfacePeer[" + hex.EncodeToString(endpoint) + "]"
		if r.ifc.Name != wantName {
			t.Fatalf("unexpected peer name: %q", r.ifc.Name)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting inbound")
	}
}

func TestWeaveIntegration_ProcessIncoming_DedupsWithinTTL(t *testing.T) {
	// Do not run in parallel: overrides global hooks.
	oldInbound := InboundHandler
	t.Cleanup(func() { InboundHandler = oldInbound })

	var calls atomic.Int32
	InboundHandler = func(_ []byte, _ *Interface) { calls.Add(1) }

	ifc, err := NewWeaveInterface("w0", map[string]string{"port": "/dev/null"})
	if err != nil {
		t.Fatalf("NewWeaveInterface: %v", err)
	}
	t.Cleanup(func() { ifc.weave.Close() })

	endpoint := []byte{7, 6, 5, 4, 3, 2, 1, 0}
	payload := []byte("same-payload")
	ifc.weave.ProcessIncoming(endpoint, payload)
	ifc.weave.ProcessIncoming(endpoint, payload)

	// give callbacks time to fire
	time.Sleep(150 * time.Millisecond)

	if calls.Load() != 1 {
		t.Fatalf("expected 1 inbound call, got %d", calls.Load())
	}
}

func TestWeaveIntegration_PeerTimeout_RemovesPeer(t *testing.T) {
	// Do not run in parallel: overrides global hooks.
	oldRemove := RemoveInterfaceHandler
	t.Cleanup(func() { RemoveInterfaceHandler = oldRemove })

	removed := make(chan *Interface, 1)
	RemoveInterfaceHandler = func(ifc *Interface) { removed <- ifc }

	ifc, err := NewWeaveInterface("w0", map[string]string{"port": "/dev/null"})
	if err != nil {
		t.Fatalf("NewWeaveInterface: %v", err)
	}
	t.Cleanup(func() { ifc.weave.Close() })

	endpoint := []byte{9, 9, 9, 9, 9, 9, 9, 9}
	payload := []byte("hello")
	ifc.weave.ProcessIncoming(endpoint, payload)

	key := hex.EncodeToString(endpoint)

	// Force peer stale.
	ifc.weave.mu.Lock()
	peer := ifc.weave.peers[key]
	if peer == nil {
		ifc.weave.mu.Unlock()
		t.Fatalf("expected peer to exist")
	}
	peer.lastHeard = time.Now().Add(-weavePeerTimeout - 5*time.Second)
	ifc.weave.mu.Unlock()

	// Run cleanup synchronously.
	ifc.weave.cleanupPeers()

	select {
	case removedIf := <-removed:
		if removedIf == nil || removedIf.Name != "WeaveInterfacePeer["+key+"]" {
			t.Fatalf("unexpected removed interface: %#v", removedIf)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting peer removal")
	}

	// Peer should be gone.
	ifc.weave.mu.Lock()
	_, ok := ifc.weave.peers[key]
	ifc.weave.mu.Unlock()
	if ok {
		t.Fatalf("expected peer to be removed")
	}
}
