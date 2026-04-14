package interfaces

import (
	"crypto/sha256"
	"net"
	"sync"
	"testing"
	"time"
)

func TestAutoInterface_DiscoveryLoop_SpawnsPeer(t *testing.T) {
	// Uses UDP6 loopback; skip if unavailable.

	pc, err := net.ListenPacket("udp6", "[::1]:0")
	if err != nil {
		t.Skipf("udp6 not available: %v", err)
	}
	defer pc.Close()

	parent := &Interface{Name: "AutoParent", IN: true, OUT: false, Online: true}

	st := &autoState{
		cfg:     defaultAutoConfig(parent.Name),
		adopted: map[string]*net.Interface{},
		llip:    map[string]net.IP{},
		llstr:   map[string]string{},
		llset:   map[string]bool{},
		llrev:   map[string]string{},
		peers:   map[string]*autoPeerState{},
		spawned: map[string]*Interface{},
		discPC:  map[string]net.PacketConn{},
		dataUC:  map[string]*net.UDPConn{},
		annStop: map[string]chan struct{}{},
		mcastE:  map[string]time.Time{},
		initE:   map[string]time.Time{},
		timedOU: map[string]bool{},
		stopCh:  make(chan struct{}),
	}
	// Avoid the announce ticker hitting sendDiscoveryAnnounce().
	st.cfg.AnnounceInterval = time.Hour
	st.finalInitDone.Store(true)
	parent.auto = st

	var (
		gotMu sync.Mutex
		got   *Interface
	)
	prevSpawn := SpawnHandler
	t.Cleanup(func() { SpawnHandler = prevSpawn })
	SpawnHandler = func(ifc *Interface) {
		gotMu.Lock()
		got = ifc
		gotMu.Unlock()
		close(st.stopCh)
	}

	annStop := make(chan struct{})
	go st.autoDiscoveryLoop(parent, "lo", pc, annStop)

	dst := pc.LocalAddr().(*net.UDPAddr)
	c, err := net.DialUDP("udp6", nil, dst)
	if err != nil {
		t.Fatalf("DialUDP: %v", err)
	}
	defer c.Close()

	peerStr := "::1"
	token := sha256.Sum256(append(append([]byte{}, st.cfg.GroupID...), []byte(peerStr)...))
	if _, err := c.Write(token[:]); err != nil {
		t.Fatalf("write token: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		gotMu.Lock()
		spawned := got
		gotMu.Unlock()
		if spawned != nil {
			if spawned.IngressControl != parent.IngressControl {
				t.Fatalf("peer IngressControl=%v, want %v", spawned.IngressControl, parent.IngressControl)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected peer spawn")
}

func TestAutoInterface_DiscoveryLoop_SpawnedPeerInheritsIngressControl(t *testing.T) {
	pc, err := net.ListenPacket("udp6", "[::1]:0")
	if err != nil {
		t.Skipf("udp6 not available: %v", err)
	}
	defer pc.Close()

	parent := &Interface{Name: "AutoParent", IN: true, OUT: false, Online: true, IngressControl: false}

	st := &autoState{
		cfg:     defaultAutoConfig(parent.Name),
		adopted: map[string]*net.Interface{},
		llip:    map[string]net.IP{},
		llstr:   map[string]string{},
		llset:   map[string]bool{},
		llrev:   map[string]string{},
		peers:   map[string]*autoPeerState{},
		spawned: map[string]*Interface{},
		discPC:  map[string]net.PacketConn{},
		dataUC:  map[string]*net.UDPConn{},
		annStop: map[string]chan struct{}{},
		mcastE:  map[string]time.Time{},
		initE:   map[string]time.Time{},
		timedOU: map[string]bool{},
		stopCh:  make(chan struct{}),
	}
	st.cfg.AnnounceInterval = time.Hour
	st.finalInitDone.Store(true)
	parent.auto = st

	gotCh := make(chan *Interface, 1)
	prevSpawn := SpawnHandler
	t.Cleanup(func() { SpawnHandler = prevSpawn })
	SpawnHandler = func(ifc *Interface) {
		select {
		case gotCh <- ifc:
		default:
		}
		close(st.stopCh)
	}

	annStop := make(chan struct{})
	go st.autoDiscoveryLoop(parent, "lo", pc, annStop)

	dst := pc.LocalAddr().(*net.UDPAddr)
	c, err := net.DialUDP("udp6", nil, dst)
	if err != nil {
		t.Fatalf("DialUDP: %v", err)
	}
	defer c.Close()

	peerStr := "::1"
	token := sha256.Sum256(append(append([]byte{}, st.cfg.GroupID...), []byte(peerStr)...))
	if _, err := c.Write(token[:]); err != nil {
		t.Fatalf("write token: %v", err)
	}

	select {
	case peer := <-gotCh:
		if peer.IngressControl != parent.IngressControl {
			t.Fatalf("peer IngressControl=%v, want %v", peer.IngressControl, parent.IngressControl)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("expected peer spawn")
	}
}

func TestAutoInterface_DataLoop_DeduplicatesMultiIF(t *testing.T) {
	pc, err := net.ListenUDP("udp6", &net.UDPAddr{IP: net.IPv6loopback, Port: 0})
	if err != nil {
		t.Skipf("udp6 not available: %v", err)
	}
	defer pc.Close()

	parent := &Interface{Name: "AutoParent", IN: true, OUT: false, Online: true}
	peerIfc := &Interface{Name: "AutoInterfacePeer[test/::1]", Parent: parent, Online: true}

	st := &autoState{
		cfg:     defaultAutoConfig(parent.Name),
		peers:   map[string]*autoPeerState{},
		spawned: map[string]*Interface{},
		stopCh:  make(chan struct{}),
	}
	st.cfg.MultiIFDequeLen = 4
	st.cfg.MultiIFDequeTTL = 2 * time.Second
	parent.auto = st

	st.peers["::1"] = &autoPeerState{addr: net.IPv6loopback, ifname: "lo", lastSeen: time.Now()}
	st.spawned["::1"] = peerIfc

	var (
		callsMu sync.Mutex
		calls   int
	)
	prevInbound := InboundHandler
	t.Cleanup(func() { InboundHandler = prevInbound })
	InboundHandler = func(raw []byte, ifc *Interface) {
		callsMu.Lock()
		calls++
		callsMu.Unlock()
	}

	go st.autoDataLoop(parent, "lo", pc)

	dst := pc.LocalAddr().(*net.UDPAddr)
	s, err := net.DialUDP("udp6", nil, dst)
	if err != nil {
		t.Fatalf("DialUDP: %v", err)
	}
	defer s.Close()

	payload := []byte("hello")
	if _, err := s.Write(payload); err != nil {
		t.Fatalf("write1: %v", err)
	}
	if _, err := s.Write(payload); err != nil {
		t.Fatalf("write2: %v", err)
	}

	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		callsMu.Lock()
		n := calls
		callsMu.Unlock()
		if n >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Allow dedup logic to run.
	time.Sleep(100 * time.Millisecond)

	callsMu.Lock()
	n := calls
	callsMu.Unlock()
	if n != 1 {
		t.Fatalf("expected 1 inbound call due to dedup, got %d", n)
	}

	close(st.stopCh)
}
