package interfaces

import (
	"net"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func startLocalTestServer(t *testing.T) (ln net.Listener, cfg LocalConfig, spawned <-chan *Interface) {
	t.Helper()

	spawnCh := make(chan *Interface, 1)
	// Use filesystem AF_UNIX sockets in tests (abstract namespace is Linux-only).
	cfg = LocalConfig{LocalSocketPath: "t" + strconv.FormatInt(time.Now().UnixNano(), 36), UseAFUnix: false}

	if runtime.GOOS == "windows" {
		cfg.LocalInterfacePort = 0
		cfg.PktAddrOverride = "tcp:127.0.0.1:0"
	}

	var err error
	ln, err = StartLocalInterfaceServer(cfg, func(ifc *Interface) { spawnCh <- ifc })
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("LocalInterface bind not permitted in sandbox: %v", err)
		}
		t.Fatalf("StartLocalInterfaceServer: %v", err)
	}

	// If using TCP :0, feed the chosen port back into the config for clients.
	if ta, ok := ln.Addr().(*net.TCPAddr); ok {
		cfg.UseAFUnix = false
		cfg.LocalInterfacePort = ta.Port
		cfg.PktAddrOverride = ""
	}

	return ln, cfg, spawnCh
}

func TestLocalIntegration_ServerSpawnsClientWithOUTFalse(t *testing.T) {
	oldInbound := InboundHandler
	t.Cleanup(func() { InboundHandler = oldInbound })
	InboundHandler = func([]byte, *Interface) {}

	ln, _, spawned := startLocalTestServer(t)
	t.Cleanup(func() { _ = ln.Close() })

	c, err := net.Dial(ln.Addr().Network(), ln.Addr().String())
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	select {
	case ifc := <-spawned:
		if ifc == nil {
			t.Fatalf("expected spawned interface")
		}
		if ifc.Type != "LocalInterface" {
			t.Fatalf("unexpected type %q", ifc.Type)
		}
		if !ifc.IN {
			t.Fatalf("expected IN=true")
		}
		if ifc.OUT {
			t.Fatalf("expected OUT=false (Python parity)")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for spawned client")
	}
}

func TestLocalIntegration_ClientToServer_FramesAndInbound(t *testing.T) {
	var (
		gotMu sync.Mutex
		got   []byte
	)

	oldInbound := InboundHandler
	t.Cleanup(func() { InboundHandler = oldInbound })
	InboundHandler = func(raw []byte, _ *Interface) {
		gotMu.Lock()
		defer gotMu.Unlock()
		if got == nil {
			got = append([]byte(nil), raw...)
		}
	}

	ln, cfg, spawned := startLocalTestServer(t)
	t.Cleanup(func() { _ = ln.Close() })

	client := &Interface{Name: "c0", Type: "LocalInterface"}
	if err := ConnectLocalInterfaceClient(cfg, client); err != nil {
		t.Fatalf("ConnectLocalInterfaceClient: %v", err)
	}
	t.Cleanup(func() { client.Detach() })
	if !strings.HasPrefix(client.Name, "LocalInterface[") {
		t.Fatalf("expected python-style local client name, got %q", client.Name)
	}

	select {
	case srv := <-spawned:
		t.Cleanup(func() { srv.Detach() })
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for spawned server-side client")
	}

	payload := []byte("hello-local")
	client.ProcessOutgoing(payload)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		gotMu.Lock()
		ok := got != nil
		gotMu.Unlock()
		if ok {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	gotMu.Lock()
	defer gotMu.Unlock()
	if got == nil || string(got) != string(payload) {
		t.Fatalf("unexpected inbound payload: got %q want %q", string(got), string(payload))
	}
}

func TestLocalIntegration_DisconnectCallbacks_SharedVsNonShared(t *testing.T) {
	oldInbound := InboundHandler
	oldGone := SharedConnectionDisappeared
	t.Cleanup(func() {
		InboundHandler = oldInbound
		SharedConnectionDisappeared = oldGone
	})
	InboundHandler = func([]byte, *Interface) {}

	gone := make(chan struct{}, 2)
	SharedConnectionDisappeared = func() { gone <- struct{}{} }

	ln, cfg, spawned := startLocalTestServer(t)
	t.Cleanup(func() { _ = ln.Close() })

	// Non-shared client: should not call SharedConnectionDisappeared.
	nonShared := &Interface{Name: "ns", Type: "LocalInterface", LocalIsSharedClient: false}
	if err := ConnectLocalInterfaceClient(cfg, nonShared); err != nil {
		t.Fatalf("ConnectLocalInterfaceClient(nonShared): %v", err)
	}
	defer nonShared.Detach()

	var serverSide *Interface
	select {
	case serverSide = <-spawned:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for spawned interface")
	}
	serverSide.Detach()

	select {
	case <-gone:
		t.Fatalf("did not expect SharedConnectionDisappeared for non-shared client")
	case <-time.After(200 * time.Millisecond):
	}

	// Shared client: should call SharedConnectionDisappeared on disconnect.
	shared := &Interface{Name: "s", Type: "LocalInterface", LocalIsSharedClient: true}
	if err := ConnectLocalInterfaceClient(cfg, shared); err != nil {
		t.Fatalf("ConnectLocalInterfaceClient(shared): %v", err)
	}
	defer shared.Detach()

	select {
	case serverSide = <-spawned:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for spawned interface")
	}
	serverSide.Detach()

	select {
	case <-gone:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected SharedConnectionDisappeared for shared client")
	}
}
