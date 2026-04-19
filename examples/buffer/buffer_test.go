package main

import (
	"bytes"
	"sync"
	"testing"
	"time"

	rns "github.com/svanichkin/go-reticulum/rns"
)

type loopbackPacket struct {
	id uint64
}

type loopbackOutlet struct {
	name string

	mdu int
	rtt float64

	mu            sync.Mutex
	peer          *rns.Channel
	nextID        uint64
	deliveredCBBy map[any]func(any)
	timeoutCBBy   map[any]func(any)
}

func newLoopbackOutlet(name string) *loopbackOutlet {
	return &loopbackOutlet{
		name:          name,
		mdu:           4096,
		rtt:           0.01,
		deliveredCBBy: make(map[any]func(any)),
		timeoutCBBy:   make(map[any]func(any)),
	}
}

func (o *loopbackOutlet) connect(peer *rns.Channel) {
	o.mu.Lock()
	o.peer = peer
	o.mu.Unlock()
}

func (o *loopbackOutlet) Send(raw []byte) any {
	o.mu.Lock()
	o.nextID++
	p := &loopbackPacket{id: o.nextID}
	peer := o.peer
	cb := o.deliveredCBBy[p]
	o.mu.Unlock()

	if peer != nil {
		peer.Receive(append([]byte(nil), raw...))
	}
	if cb != nil {
		go cb(p)
	}
	return p
}

func (o *loopbackOutlet) Resend(packet any) any {
	// For this harness, resend is treated as immediate delivery.
	lp, ok := packet.(*loopbackPacket)
	if !ok || lp == nil {
		return nil
	}
	o.mu.Lock()
	cb := o.deliveredCBBy[lp]
	o.mu.Unlock()
	if cb != nil {
		go cb(lp)
	}
	return lp
}

func (o *loopbackOutlet) Mdu() int                   { return o.mdu }
func (o *loopbackOutlet) Rtt() float64               { return o.rtt }
func (o *loopbackOutlet) IsUsable() bool             { return true }
func (o *loopbackOutlet) TimedOut()                  {}
func (o *loopbackOutlet) String() string             { return o.name }
func (o *loopbackOutlet) GetPacketID(packet any) any { return packet }

func (o *loopbackOutlet) GetPacketState(packet any) rns.MessageState {
	// Treat all packets as delivered instantly.
	return rns.MSGSTATE_DELIVERED
}

func (o *loopbackOutlet) SetPacketTimeoutCallback(packet any, cb func(any), _ *float64) {
	o.mu.Lock()
	o.timeoutCBBy[packet] = cb
	o.mu.Unlock()
}

func (o *loopbackOutlet) SetPacketDeliveredCallback(packet any, cb func(any)) {
	o.mu.Lock()
	o.deliveredCBBy[packet] = cb
	o.mu.Unlock()
}

func TestBufferExample_BidirectionalEcho(t *testing.T) {
	// A unit-style test that exercises Buffer + Channel with an in-memory outlet.
	outA := newLoopbackOutlet("a")
	outB := newLoopbackOutlet("b")

	chA := rns.NewChannel(outA)
	chB := rns.NewChannel(outB)
	outA.connect(chB)
	outB.connect(chA)

	var (
		serverBuf *rns.BufferedRWPair
		clientBuf *rns.BufferedRWPair
		mu        sync.Mutex
		got       []string
	)

	serverBuf = rns.CreateBidirectionalBuffer(0, 0, chB, func(readyBytes int) {
		if readyBytes <= 0 {
			return
		}
		data := make([]byte, readyBytes)
		n, _ := serverBuf.Reader.ReadInto(data)
		if n == 0 {
			return
		}
		reply := []byte("I received \"" + string(data[:n]) + "\" over the buffer")
		_, _ = serverBuf.Writer.Write(reply)
	})

	clientBuf = rns.CreateBidirectionalBuffer(0, 0, chA, func(readyBytes int) {
		if readyBytes <= 0 {
			return
		}
		data := make([]byte, readyBytes)
		n, _ := clientBuf.Reader.ReadInto(data)
		if n == 0 {
			return
		}
		mu.Lock()
		got = append(got, string(data[:n]))
		mu.Unlock()
	})

	_, _ = clientBuf.Writer.Write([]byte("hello"))

	waitUntil := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(waitUntil) {
		mu.Lock()
		ok := len(got) == 1 && got[0] == "I received \"hello\" over the buffer"
		mu.Unlock()
		if ok {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(got) != 1 {
		t.Fatalf("expected 1 reply, got %d (%v)", len(got), got)
	}
	if got[0] != "I received \"hello\" over the buffer" {
		t.Fatalf("unexpected reply %q", got[0])
	}
	if bytes.Contains([]byte(got[0]), []byte{0}) {
		t.Fatalf("unexpected NUL bytes in reply")
	}
}
