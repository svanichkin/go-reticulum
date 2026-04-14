package rns

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPerformRPCHandshake_NetPipe_Success(t *testing.T) {
	maybeParallel(t)

	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	key := []byte("secret")
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- performRPCHandshake(c2, key, true)
	}()

	if err := performRPCHandshake(c1, key, false); err != nil {
		t.Fatalf("client handshake: %v", err)
	}
	if err := <-serverErr; err != nil {
		t.Fatalf("server handshake: %v", err)
	}
}

func TestPerformRPCHandshake_NetPipe_InvalidKey(t *testing.T) {
	maybeParallel(t)

	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	serverKey := []byte("server-secret")
	clientKey := []byte("client-secret")

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- performRPCHandshake(c2, serverKey, true)
	}()

	if err := performRPCHandshake(c1, clientKey, false); err == nil {
		t.Fatalf("expected client to reject invalid key")
	}
	if err := <-serverErr; err == nil {
		t.Fatalf("expected server to reject invalid key")
	}
}

func TestFallbackUnixSocketPath_SanitizesAndShortens(t *testing.T) {
	maybeParallel(t)

	addr := "\x00rns/test socket:name/with/slashes"
	got := fallbackUnixSocketPath(addr)

	if !strings.HasPrefix(got, os.TempDir()) {
		t.Fatalf("expected tempdir prefix, got %q", got)
	}
	if !strings.HasSuffix(got, ".sock") {
		t.Fatalf("expected .sock suffix, got %q", got)
	}
	if strings.Contains(got, "/rns/test") {
		t.Fatalf("expected sanitised name, got %q", got)
	}
	if len(filepath.Base(got)) > 128 {
		t.Fatalf("expected reasonably short socket path, got len=%d (%q)", len(got), got)
	}
}

func TestRPCListener_TCP_HandshakeAndMsgpack(t *testing.T) {
	maybeParallel(t)

	key := []byte("rpc-key")
	ln, err := NewRPCListener("tcp", "127.0.0.1:0", key)
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("TCP listen not permitted in sandbox: %v", err)
		}
		t.Fatalf("NewRPCListener: %v", err)
	}
	defer ln.Close()

	serverErr := make(chan error, 1)
	serverGot := make(chan string, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		defer c.Close()
		var msg string
		if err := c.Recv(&msg); err != nil {
			serverErr <- err
			return
		}
		if err := c.Send("ack:" + msg); err != nil {
			serverErr <- err
			return
		}
		serverGot <- msg
		serverErr <- nil
	}()

	c, err := dialRPC("tcp", ln.Addr(), key)
	if err != nil {
		t.Fatalf("dialRPC: %v", err)
	}
	defer c.Close()

	if err := c.Send("hello"); err != nil {
		t.Fatalf("client send: %v", err)
	}
	var resp string
	if err := c.Recv(&resp); err != nil {
		t.Fatalf("client recv: %v", err)
	}
	if resp != "ack:hello" {
		t.Fatalf("unexpected response %q", resp)
	}

	select {
	case msg := <-serverGot:
		if msg != "hello" {
			t.Fatalf("server got %q", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting server")
	}
	if err := <-serverErr; err != nil {
		t.Fatalf("server: %v", err)
	}
}

func TestWaitForLocalClientsToDisconnect_Completes(t *testing.T) {
	prev := LocalClientInterfaces
	t.Cleanup(func() { LocalClientInterfaces = prev })

	LocalClientInterfaces = []*Interface{{Name: "c1"}}
	go func() {
		time.Sleep(50 * time.Millisecond)
		LocalClientInterfaces = nil
	}()

	if ok := waitForLocalClientsToDisconnect(2 * time.Second); !ok {
		t.Fatalf("expected wait to complete")
	}
}

func TestWaitForLocalClientsToDisconnect_TimesOut(t *testing.T) {
	prev := LocalClientInterfaces
	t.Cleanup(func() { LocalClientInterfaces = prev })

	LocalClientInterfaces = []*Interface{{Name: "c1"}}
	if ok := waitForLocalClientsToDisconnect(50 * time.Millisecond); ok {
		t.Fatalf("expected timeout")
	}
}
