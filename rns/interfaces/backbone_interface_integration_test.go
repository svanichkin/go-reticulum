package interfaces

import (
	"bytes"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type backboneFakeListener struct {
	closed chan struct{}
}

func (l *backboneFakeListener) Accept() (net.Conn, error) { <-l.closed; return nil, net.ErrClosed }
func (l *backboneFakeListener) Close() error {
	select {
	case <-l.closed:
	default:
		close(l.closed)
	}
	return nil
}
func (l *backboneFakeListener) Addr() net.Addr { return &net.TCPAddr{} }

type backboneTimeoutError struct{}

func (backboneTimeoutError) Error() string   { return "timeout" }
func (backboneTimeoutError) Timeout() bool   { return true }
func (backboneTimeoutError) Temporary() bool { return true }

type backboneTimeoutConn struct {
	writes      atomic.Int32
	closed      atomic.Bool
	wrote       chan []byte
	writeDLUsed atomic.Bool
	alwaysTO    atomic.Bool
	toCount     atomic.Int32
}

func (c *backboneTimeoutConn) Read([]byte) (int, error)  { return 0, net.ErrClosed }
func (c *backboneTimeoutConn) Close() error              { c.closed.Store(true); return nil }
func (c *backboneTimeoutConn) LocalAddr() net.Addr       { return &net.TCPAddr{} }
func (c *backboneTimeoutConn) RemoteAddr() net.Addr      { return &net.TCPAddr{} }
func (c *backboneTimeoutConn) SetDeadline(time.Time) error      { return nil }
func (c *backboneTimeoutConn) SetReadDeadline(time.Time) error  { return nil }
func (c *backboneTimeoutConn) SetWriteDeadline(time.Time) error { c.writeDLUsed.Store(true); return nil }

func (c *backboneTimeoutConn) Write(p []byte) (int, error) {
	if c.closed.Load() {
		return 0, net.ErrClosed
	}
	if c.alwaysTO.Load() {
		c.toCount.Add(1)
		return 0, backboneTimeoutError{}
	}
	if c.writes.Add(1) == 1 {
		return 0, backboneTimeoutError{}
	}
	cp := make([]byte, len(p))
	copy(cp, p)
	select {
	case c.wrote <- cp:
	default:
	}
	return len(p), nil
}

func TestBackboneIntegration_HandleConn_SpawnsPeerAndCleansUp(t *testing.T) {
	oldInbound := InboundHandler
	oldSpawn := SpawnHandler
	oldRemove := RemoveInterfaceHandler
	t.Cleanup(func() {
		InboundHandler = oldInbound
		SpawnHandler = oldSpawn
		RemoveInterfaceHandler = oldRemove
	})

	InboundHandler = func([]byte, *Interface) {}
	spawned := make(chan *Interface, 1)
	SpawnHandler = func(iface *Interface) { spawned <- iface }
	RemoveInterfaceHandler = func(*Interface) {}

	parent := &Interface{
		Name:              "bb0",
		Type:              "BackboneInterface",
		IN:                true,
		OUT:               false,
		IngressControl:    false,
		DriverImplemented: true,
		HWMTU:             1 << 16,
		Bitrate:           backboneBitrateGuess,
	}
	driver := &BackboneInterfaceDriver{iface: parent, stopCh: make(chan struct{})}

	serverSide, clientSide := net.Pipe()
	defer clientSide.Close()

	if err := driver.handleConn(serverSide); err != nil {
		t.Fatalf("handleConn: %v", err)
	}

	var peerIface *Interface
	select {
	case peerIface = <-spawned:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for SpawnHandler")
	}
	if peerIface == nil || peerIface.Parent != parent || peerIface.Type != "BackboneInterfacePeer" {
		t.Fatalf("unexpected peer iface: %#v", peerIface)
	}
	if peerIface.IngressControl != parent.IngressControl {
		t.Fatalf("peer IngressControl=%v, want %v", peerIface.IngressControl, parent.IngressControl)
	}
	if driver.ClientCount() != 1 {
		t.Fatalf("expected 1 client, got %d", driver.ClientCount())
	}

	_ = clientSide.Close()

	deadline := time.Now().Add(2 * time.Second)
	for driver.ClientCount() != 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if driver.ClientCount() != 0 {
		t.Fatalf("expected client cleanup, still have %d", driver.ClientCount())
	}
}

func TestBackboneIntegration_ProcessOutgoing_FramesHDLC(t *testing.T) {
	oldRemove := RemoveInterfaceHandler
	t.Cleanup(func() { RemoveInterfaceHandler = oldRemove })
	RemoveInterfaceHandler = func(*Interface) {}

	serverSide, clientSide := net.Pipe()
	defer serverSide.Close()
	defer clientSide.Close()

	iface := &Interface{Name: "peer", Type: "BackboneInterfacePeer", HWMTU: 1 << 16}
	peer := &BackbonePeer{iface: iface, conn: serverSide}

	payload := []byte{0x01, hdlcFlag, 0x02, hdlcEsc, 0x03}
	want := append([]byte{hdlcFlag}, hdlcEscape(payload)...)
	want = append(want, hdlcFlag)

	var gotMu sync.Mutex
	var got []byte
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 64)
		for {
			_ = clientSide.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
			n, err := clientSide.Read(buf)
			if n > 0 {
				gotMu.Lock()
				got = append(got, buf[:n]...)
				gotMu.Unlock()
				if len(got) >= len(want) {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	peer.ProcessOutgoing(payload)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for framed bytes")
	}
	gotMu.Lock()
	defer gotMu.Unlock()
	if !bytes.Equal(got[:len(want)], want) {
		t.Fatalf("unexpected frame:\nwant %x\ngot  %x", want, got)
	}
}

func TestBackboneIntegration_ReadLoop_DeframesHDLC(t *testing.T) {
	oldInbound := InboundHandler
	oldRemove := RemoveInterfaceHandler
	t.Cleanup(func() {
		InboundHandler = oldInbound
		RemoveInterfaceHandler = oldRemove
	})
	RemoveInterfaceHandler = func(*Interface) {}

	serverSide, clientSide := net.Pipe()
	defer clientSide.Close()

	iface := &Interface{Name: "peer", Type: "BackboneInterfacePeer", HWMTU: 1 << 16}
	peer := &BackbonePeer{iface: iface, conn: serverSide}

	payload := []byte("hello\x7e\x7dworld")
	framed := append([]byte{hdlcFlag}, hdlcEscape(payload)...)
	framed = append(framed, hdlcFlag)

	recv := make(chan []byte, 1)
	InboundHandler = func(b []byte, from *Interface) {
		if from != iface {
			return
		}
		cp := make([]byte, len(b))
		copy(cp, b)
		recv <- cp
	}

	go peer.readLoop()

	if _, err := clientSide.Write(framed); err != nil {
		t.Fatalf("write framed: %v", err)
	}

	select {
	case got := <-recv:
		if !bytes.Equal(got, payload) {
			t.Fatalf("unexpected payload: want %x got %x", payload, got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for inbound payload")
	}

	_ = clientSide.Close()
}

func TestBackboneIntegration_ServerClose_ClosesListeners(t *testing.T) {
	ln := &backboneFakeListener{closed: make(chan struct{})}
	driver := &BackboneInterfaceDriver{iface: &Interface{Name: "bb0"}, lns: []net.Listener{ln}, stopCh: make(chan struct{})}

	driver.Close()

	select {
	case <-ln.closed:
	default:
		t.Fatal("expected listener to be closed")
	}
}

func TestBackboneIntegration_PeerClose_RemovesInterface(t *testing.T) {
	oldRemove := RemoveInterfaceHandler
	t.Cleanup(func() { RemoveInterfaceHandler = oldRemove })

	removed := make(chan *Interface, 1)
	RemoveInterfaceHandler = func(iface *Interface) { removed <- iface }

	parent := &Interface{Name: "bb0"}
	driver := &BackboneInterfaceDriver{iface: parent, stopCh: make(chan struct{})}

	serverSide, clientSide := net.Pipe()
	defer clientSide.Close()

	peerIface := &Interface{Name: "peer0", Type: "BackboneInterfacePeer", Parent: parent}
	peer := &BackbonePeer{parent: driver, iface: peerIface, conn: serverSide}
	peer.ensureWriter()

	peer.Close()

	select {
	case got := <-removed:
		if got != peerIface {
			t.Fatalf("unexpected removed iface: %#v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for RemoveInterfaceHandler")
	}
}

func TestBackboneIntegration_WriteLoop_TimeoutDoesNotClosePeer(t *testing.T) {
	oldRemove := RemoveInterfaceHandler
	t.Cleanup(func() { RemoveInterfaceHandler = oldRemove })
	RemoveInterfaceHandler = func(*Interface) {}

	c := &backboneTimeoutConn{wrote: make(chan []byte, 1)}
	iface := &Interface{Name: "peer", Type: "BackboneInterfacePeer"}
	peer := &BackbonePeer{iface: iface, conn: c}
	peer.ensureWriter()

	peer.ProcessOutgoing([]byte("hi"))

	select {
	case <-c.wrote:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for successful write after initial timeout")
	}
	if peer.closed.Load() {
		t.Fatal("peer must not be closed on write timeout")
	}
	if !c.writeDLUsed.Load() {
		t.Fatal("expected SetWriteDeadline to be used")
	}
}

func TestBackboneIntegration_WriteLoop_NoBusyLoopOnSustainedTimeouts(t *testing.T) {
	oldRemove := RemoveInterfaceHandler
	t.Cleanup(func() { RemoveInterfaceHandler = oldRemove })
	RemoveInterfaceHandler = func(*Interface) {}

	c := &backboneTimeoutConn{wrote: make(chan []byte, 1)}
	c.alwaysTO.Store(true)
	iface := &Interface{Name: "peer", Type: "BackboneInterfacePeer"}
	peer := &BackbonePeer{iface: iface, conn: c}
	peer.ensureWriter()

	peer.ProcessOutgoing([]byte("hi"))

	time.Sleep(120 * time.Millisecond)
	attempts := c.toCount.Load()
	// With exponential backoff starting at 10ms, attempts over ~120ms should stay low.
	if attempts > 25 {
		t.Fatalf("expected low write attempts under sustained timeouts, got %d", attempts)
	}
	if peer.closed.Load() {
		t.Fatal("peer must not be closed on write timeouts")
	}
}

func TestBackboneIntegration_WriteLoop_PreservesOrderAfterTimeouts(t *testing.T) {
	oldRemove := RemoveInterfaceHandler
	t.Cleanup(func() { RemoveInterfaceHandler = oldRemove })
	RemoveInterfaceHandler = func(*Interface) {}

	c := &backboneTimeoutConn{wrote: make(chan []byte, 4)}
	iface := &Interface{Name: "peer", Type: "BackboneInterfacePeer"}
	peer := &BackbonePeer{iface: iface, conn: c}
	peer.ensureWriter()

	peer.ProcessOutgoing([]byte("one"))
	peer.ProcessOutgoing([]byte("two"))

	got1 := <-c.wrote
	got2 := <-c.wrote
	if bytes.IndexByte(got1, hdlcFlag) != 0 || bytes.IndexByte(got2, hdlcFlag) != 0 {
		t.Fatalf("unexpected framing")
	}
	// Minimal check: payloads appear in order within framed data.
	if !bytes.Contains(got1, []byte("one")) || !bytes.Contains(got2, []byte("two")) {
		t.Fatalf("unexpected order or payloads: %q %q", got1, got2)
	}
}

type backboneWriteFailConn struct {
	closed atomic.Bool
}

func (c *backboneWriteFailConn) Read([]byte) (int, error)         { return 0, net.ErrClosed }
func (c *backboneWriteFailConn) Write([]byte) (int, error)        { return 0, errors.New("write failed") }
func (c *backboneWriteFailConn) Close() error                     { c.closed.Store(true); return nil }
func (c *backboneWriteFailConn) LocalAddr() net.Addr              { return &net.TCPAddr{} }
func (c *backboneWriteFailConn) RemoteAddr() net.Addr             { return &net.TCPAddr{} }
func (c *backboneWriteFailConn) SetDeadline(time.Time) error      { return nil }
func (c *backboneWriteFailConn) SetReadDeadline(time.Time) error  { return nil }
func (c *backboneWriteFailConn) SetWriteDeadline(time.Time) error { return nil }

func TestBackboneIntegration_Detach_ClosesPeersAndRemovesInterfaces(t *testing.T) {
	oldRemove := RemoveInterfaceHandler
	t.Cleanup(func() { RemoveInterfaceHandler = oldRemove })

	removed := make(chan *Interface, 4)
	RemoveInterfaceHandler = func(iface *Interface) { removed <- iface }

	parent := &Interface{Name: "bb0", Type: "BackboneInterface"}
	driver := &BackboneInterfaceDriver{
		iface:  parent,
		lns:    []net.Listener{&backboneFakeListener{closed: make(chan struct{})}},
		stopCh: make(chan struct{}),
	}
	parent.backboneServer = driver

	p1 := &BackbonePeer{parent: driver, iface: &Interface{Name: "p1", Type: "BackboneInterfacePeer", Parent: parent}, conn: &backboneTimeoutConn{wrote: make(chan []byte, 1)}}
	p2 := &BackbonePeer{parent: driver, iface: &Interface{Name: "p2", Type: "BackboneInterfacePeer", Parent: parent}, conn: &backboneTimeoutConn{wrote: make(chan []byte, 1)}}
	p1.ensureWriter()
	p2.ensureWriter()
	driver.clients.Store(p1, struct{}{})
	driver.clients.Store(p2, struct{}{})
	p1.onClose = func() { driver.clients.Delete(p1) }
	p2.onClose = func() { driver.clients.Delete(p2) }

	parent.Detach()

	// Expect both spawned peer interfaces removed.
	got := map[*Interface]bool{}
	deadline := time.After(2 * time.Second)
	for len(got) < 2 {
		select {
		case ifc := <-removed:
			got[ifc] = true
		case <-deadline:
			t.Fatalf("timeout waiting for removed interfaces, got=%d", len(got))
		}
	}
	if !got[p1.iface] || !got[p2.iface] {
		t.Fatalf("unexpected removed set: %#v", got)
	}
	if driver.ClientCount() != 0 {
		t.Fatalf("expected 0 clients after detach, got %d", driver.ClientCount())
	}
}

func TestBackboneIntegration_WriteError_RemovesPeerInterface(t *testing.T) {
	oldRemove := RemoveInterfaceHandler
	t.Cleanup(func() { RemoveInterfaceHandler = oldRemove })
	RemoveInterfaceHandler = func(*Interface) {}

	removed := make(chan *Interface, 1)
	RemoveInterfaceHandler = func(iface *Interface) { removed <- iface }

	parent := &Interface{Name: "bb0", Type: "BackboneInterface"}
	driver := &BackboneInterfaceDriver{iface: parent, stopCh: make(chan struct{})}

	peerIface := &Interface{Name: "peer0", Type: "BackboneInterfacePeer", Parent: parent}
	peer := &BackbonePeer{parent: driver, iface: peerIface, conn: &backboneWriteFailConn{}}
	peer.ensureWriter()

	peer.ProcessOutgoing([]byte("data"))

	select {
	case got := <-removed:
		if got != peerIface {
			t.Fatalf("unexpected removed iface: %#v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for RemoveInterfaceHandler after write error")
	}
}
