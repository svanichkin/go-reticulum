package rns

import (
	"bytes"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type loopPacket struct {
	id  uint64
	raw []byte
}

type loopOutlet struct {
	name string
	peer atomic.Pointer[Channel]
	mdu  int
	rtt  float64

	mu        sync.Mutex
	delivered map[uint64]bool
	dcb       map[uint64]func(any)
}

func newLoopOutlet(name string, mdu int) *loopOutlet {
	return &loopOutlet{
		name:      name,
		mdu:       mdu,
		rtt:       0.05,
		delivered: make(map[uint64]bool),
		dcb:       make(map[uint64]func(any)),
	}
}

var loopPktID atomic.Uint64

func (o *loopOutlet) Send(raw []byte) any {
	p := &loopPacket{id: loopPktID.Add(1), raw: append([]byte(nil), raw...)}

	peer := o.peer.Load()
	if peer != nil {
		peer.Receive(p.raw)
	}

	o.mu.Lock()
	o.delivered[p.id] = true
	cb := o.dcb[p.id]
	o.mu.Unlock()

	if cb != nil {
		cb(p)
	}
	return p
}

func (o *loopOutlet) Resend(packet any) any {
	p, _ := packet.(*loopPacket)
	if p == nil {
		return nil
	}
	peer := o.peer.Load()
	if peer != nil {
		peer.Receive(p.raw)
	}
	return p
}

func (o *loopOutlet) Mdu() int       { return o.mdu }
func (o *loopOutlet) Rtt() float64   { return o.rtt }
func (o *loopOutlet) IsUsable() bool { return true }
func (o *loopOutlet) TimedOut()      {}
func (o *loopOutlet) String() string { return o.name }
func (o *loopOutlet) GetPacketID(packet any) any {
	if p, ok := packet.(*loopPacket); ok && p != nil {
		return p.id
	}
	return nil
}

func (o *loopOutlet) GetPacketState(packet any) MessageState {
	p, _ := packet.(*loopPacket)
	if p == nil {
		return MSGSTATE_FAILED
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.delivered[p.id] {
		return MSGSTATE_DELIVERED
	}
	return MSGSTATE_SENT
}

func (o *loopOutlet) SetPacketTimeoutCallback(packet any, cb func(any), timeout *float64) {
	// No-op for tests; we deliver synchronously.
	_ = packet
	_ = cb
	_ = timeout
}

func (o *loopOutlet) SetPacketDeliveredCallback(packet any, cb func(any)) {
	p, _ := packet.(*loopPacket)
	if p == nil {
		return
	}
	o.mu.Lock()
	o.dcb[p.id] = cb
	already := o.delivered[p.id]
	o.mu.Unlock()
	if already && cb != nil {
		cb(p)
	}
}

func TestRawChannelStream_NoContentLoss(t *testing.T) {
	oa := newLoopOutlet("a", 65535)
	ob := newLoopOutlet("b", 65535)

	ca := NewChannel(oa)
	cb := NewChannel(ob)
	oa.peer.Store(cb)
	ob.peer.Store(ca)

	// Receiver side reader.
	reader := NewRawChannelReader(1, cb)
	defer reader.Close()

	// Sender side writer.
	writer := NewRawChannelWriter(1, ca)
	defer writer.Close()

	// Highly compressible payload to exercise compression path.
	want := bytes.Repeat([]byte("A"), 250_000)

		written := 0
		for written < len(want) {
			n, err := writer.Write(want[written:])
			written += n
			if err != nil {
				t.Fatalf("write error: %v", err)
			}
			if n == 0 {
				t.Fatalf("short write: got %d want %d", written, len(want))
			}
		}
	if err := writer.Close(); err != nil {
		t.Fatalf("close error: %v", err)
	}

	var got bytes.Buffer
	buf := make([]byte, 8192)
	for {
		readN, rerr := reader.Read(buf)
		if readN > 0 {
			got.Write(buf[:readN])
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			t.Fatalf("read error: %v", rerr)
		}
		// Safety: shouldn't spin forever.
		if got.Len() > len(want)+1024 {
			t.Fatalf("read too much data: %d", got.Len())
		}
		// Avoid tight loop if implementation changes.
		time.Sleep(1 * time.Millisecond)
	}

	if !bytes.Equal(got.Bytes(), want) {
		t.Fatalf("payload mismatch: got %d bytes want %d", got.Len(), len(want))
	}
}

func TestRawChannelReader_ReadyCallback_BufferedSize(t *testing.T) {
	oa := newLoopOutlet("a", 65535)
	ob := newLoopOutlet("b", 65535)

	ca := NewChannel(oa)
	cb := NewChannel(ob)
	oa.peer.Store(cb)
	ob.peer.Store(ca)

	reader := NewRawChannelReader(1, cb)
	defer reader.Close()

	ready := make(chan int, 4)
	reader.AddReadyCallback(func(n int) { ready <- n })

	writer := NewRawChannelWriter(1, ca)
	defer writer.Close()

	if _, err := writer.Write([]byte("1234567890")); err != nil {
		t.Fatalf("write1: %v", err)
	}
	if _, err := writer.Write([]byte("abcde")); err != nil {
		t.Fatalf("write2: %v", err)
	}

	var got []int
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	for len(got) < 2 {
		select {
		case n := <-ready:
			got = append(got, n)
		case <-deadline.C:
			t.Fatalf("timeout waiting for callbacks, got=%v", got)
		}
	}

	// Python passes len(self._buffer) after each received message.
	if !((got[0] == 10 && got[1] == 15) || (got[0] == 15 && got[1] == 10)) {
		t.Fatalf("unexpected ready sizes: %v", got)
	}
}

func TestStreamDataMessage_PackWithoutStreamIDFails(t *testing.T) {
	t.Parallel()

	msg := &StreamDataMessage{Data: []byte("abc")}
	if _, err := msg.Pack(); err == nil || err.Error() != "stream_id" {
		t.Fatalf("Pack() err=%v, want stream_id", err)
	}
}

func TestRawChannelReader_RemoveReadyCallback_PanicsLikeValueError(t *testing.T) {
	oa := newLoopOutlet("a", 65535)
	ch := NewChannel(oa)
	reader := NewRawChannelReader(1, ch)
	defer reader.Close()

	cb := func(int) {}

	defer func() {
		rec := recover()
		if rec == nil {
			t.Fatal("expected panic")
		}
		ve, ok := rec.(*BufferValueError)
		if !ok {
			t.Fatalf("panic type=%T, want *BufferValueError", rec)
		}
		if ve.Message != "list.remove(x): x not in list" {
			t.Fatalf("panic message=%q", ve.Message)
		}
	}()

	reader.RemoveReadyCallback(cb)
}

func TestRawChannelWriter_CloseWaitDuration_MatchesPythonFallback(t *testing.T) {
	t.Parallel()

	oa := newLoopOutlet("a", 65535)
	ch := NewChannel(oa)
	writer := NewRawChannelWriter(1, ch)
	if got := writer.closeWaitDuration(); got != 15*time.Second {
		t.Fatalf("closeWaitDuration()=%v, want 15s", got)
	}
}

func TestBufferAliases_CreateHelpersReturnAliasTypes(t *testing.T) {
	t.Parallel()

	oa := newLoopOutlet("a", 65535)
	ch := NewChannel(oa)

	reader := CreateReader(1, ch, nil)
	if reader == nil {
		t.Fatal("CreateReader returned nil")
	}
	writer := CreateWriter(1, ch)
	if writer == nil {
		t.Fatal("CreateWriter returned nil")
	}
	rw := CreateBidirectionalBuffer(1, 1, ch, nil)
	if rw == nil {
		t.Fatal("CreateBidirectionalBuffer returned nil")
	}
}
