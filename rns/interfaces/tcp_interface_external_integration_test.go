//go:build integration

package interfaces

import (
	"context"
	"net"
	"os"
	"testing"
	"time"
)

func TestTCPIntegration_ExternalNode_212_233_88_164_4242(t *testing.T) {
	if os.Getenv("RNS_EXTERNAL_TCP_21223388164") == "" {
		t.Skip("set RNS_EXTERNAL_TCP_21223388164=1 to run external TCP integration test")
	}

	client := NewTCPClientInitiator(nil, nil, "external-smoke", "212.233.88.164", 4242, false, false)
	client.ConnectTimeout = 5 * time.Second
	client.ReconnectWait = 1 * time.Second
	zero := 0
	client.MaxReconnectTry = &zero
	t.Cleanup(func() { client.Detach() })

	ctx, cancel := context.WithTimeout(context.Background(), client.ConnectTimeout)
	defer cancel()

	if ok := client.connect(ctx, true); !ok {
		t.Fatalf("could not connect TCPClientInterface to 212.233.88.164:4242")
	}
	if !client.online.Load() {
		t.Fatal("client is not marked online after successful external TCP connect")
	}
	conn := client.getConn()
	if conn == nil {
		t.Fatal("client connection is nil after successful external TCP connect")
	}

	// Send one valid HDLC-framed payload and ensure the peer does not close the
	// socket immediately on write. This keeps the test a smoke check without
	// assuming any application-layer response semantics from the remote node.
	if err := client.ProcessOutgoing([]byte("external-smoke")); err != nil {
		t.Fatalf("write framed payload to external TCP node: %v", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(750 * time.Millisecond))
	buf := make([]byte, 1)
	_, err := conn.Read(buf)
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		return
	}
	if err == nil {
		return
	}
	t.Fatalf("external TCP node closed socket or returned unexpected read error after payload write: %v", err)
}
