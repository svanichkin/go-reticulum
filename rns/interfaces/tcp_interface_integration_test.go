package interfaces

import (
	"bytes"
	"net"
	"testing"
	"time"
)

func TestTCPIntegration_HDLC_HeaderMinSize_DropsShortFrames(t *testing.T) {
	// Do not run in parallel: overrides global hook.

	oldHeader := HeaderMinSize
	t.Cleanup(func() { HeaderMinSize = oldHeader })
	HeaderMinSize = 5

	serverSide, clientSide := net.Pipe()
	defer serverSide.Close()
	defer clientSide.Close()

	got := make(chan []byte, 1)
	owner := TCPOwner(func(b []byte, _ *TCPClientInterface) { got <- append([]byte(nil), b...) })
	iface := NewTCPClientFromAccepted(owner, nil, "c0", serverSide, false, false)
	t.Cleanup(func() { iface.Detach() })

	// Start reader.
	go iface.readLoop()

	// Frame with payload len=5 should be dropped (<= HeaderMinSize).
	short := []byte("12345")
	framed := append([]byte{HDLC_FLAG}, (HDLC{}).Escape(short)...)
	framed = append(framed, HDLC_FLAG)
	_ = clientSide.SetWriteDeadline(time.Now().Add(1 * time.Second))
	if _, err := clientSide.Write(framed); err != nil {
		t.Fatalf("write short: %v", err)
	}
	select {
	case <-got:
		t.Fatalf("expected short frame to be dropped")
	case <-time.After(200 * time.Millisecond):
	}

	// Frame with payload len=6 should pass.
	long := []byte("123456")
	framed2 := append([]byte{HDLC_FLAG}, (HDLC{}).Escape(long)...)
	framed2 = append(framed2, HDLC_FLAG)
	_ = clientSide.SetWriteDeadline(time.Now().Add(1 * time.Second))
	if _, err := clientSide.Write(framed2); err != nil {
		t.Fatalf("write long: %v", err)
	}
	select {
	case b := <-got:
		if !bytes.Equal(b, long) {
			t.Fatalf("unexpected payload: want %q got %q", string(long), string(b))
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for long frame")
	}
}
