package rns

import (
	"bytes"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type stuckOutlet struct {
	name string
	mdu  int
	rtt  float64
	id   atomic.Uint64
}

func (o *stuckOutlet) Send(raw []byte) any {
	_ = raw
	return o.id.Add(1)
}
func (o *stuckOutlet) Resend(packet any) any { return packet }
func (o *stuckOutlet) Mdu() int              { return o.mdu }
func (o *stuckOutlet) Rtt() float64          { return o.rtt }
func (o *stuckOutlet) IsUsable() bool        { return true }
func (o *stuckOutlet) TimedOut()             {}
func (o *stuckOutlet) String() string        { return o.name }
func (o *stuckOutlet) GetPacketID(packet any) any {
	return packet
}
func (o *stuckOutlet) GetPacketState(packet any) MessageState {
	_ = packet
	// Never delivered, so channel window will fill.
	return MSGSTATE_SENT
}
func (o *stuckOutlet) SetPacketTimeoutCallback(packet any, cb func(any), timeout *float64) {
	_ = packet
	_ = cb
	_ = timeout
}
func (o *stuckOutlet) SetPacketDeliveredCallback(packet any, cb func(any)) {
	_ = packet
	_ = cb
}

func TestBufferIntegration_RawChannelWriter_NonBlockingOnLinkNotReady(t *testing.T) {
	o := &stuckOutlet{name: "stuck", mdu: 65535, rtt: 0.05}
	ch := NewChannel(o)

	writer := newTestRawChannelWriter(1, ch)

	// First writes should succeed until the channel window is full.
	n1, err := writer.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("write1 err: %v", err)
	}
	if n1 == 0 {
		t.Fatalf("write1 expected progress")
	}

	n2, err := writer.Write([]byte("world"))
	if err != nil {
		t.Fatalf("write2 err: %v", err)
	}
	if n2 == 0 {
		t.Fatalf("write2 expected progress")
	}

	// Third write should remain non-blocking and report no progress once the
	// channel window is exhausted.
	start := time.Now()
	n3, err := writer.Write([]byte("!"))
	if err != nil {
		t.Fatalf("write3 err: %v", err)
	}
	if n3 != 0 {
		t.Fatalf("write3 n=%d, want 0 once channel is full", n3)
	}
	if time.Since(start) > 100*time.Millisecond {
		t.Fatalf("write3 unexpectedly blocked")
	}
}

func TestBufferIntegration_ChannelBufferedWriter_FlushBlocksUntilProgress(t *testing.T) {
	// This is a smoke test that raw write sends StreamDataMessage and the
	// receiving side can unpack it.
	oa := newLoopOutlet("a", 65535)
	ob := newLoopOutlet("b", 65535)
	ca := NewChannel(oa)
	cb := NewChannel(ob)
	oa.peer.Store(cb)
	ob.peer.Store(ca)

	reader := CreateReader(1, cb, nil)
	defer reader.Close()
	writer := CreateWriter(1, ca)
	defer writer.Close()

	payload := bytes.Repeat([]byte("A"), 10000)
	if _, err := writer.Write(payload); err != nil {
		t.Fatalf("write: %v", err)
	}
	buf := make([]byte, len(payload))
	var got bytes.Buffer
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		n, err := reader.ReadInto(buf)
		if err != nil && !errors.Is(err, ErrWouldBlock) {
			t.Fatalf("read: %v", err)
		}
		if n > 0 {
			got.Write(buf[:n])
		}
		if got.Len() >= len(payload) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !bytes.Equal(got.Bytes(), payload) {
		t.Fatalf("payload mismatch: got %d want %d", got.Len(), len(payload))
	}
}
