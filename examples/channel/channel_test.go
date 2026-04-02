package main

import (
	"sync"
	"testing"
	"time"

	rns "github.com/svanichkin/go-reticulum/rns"
)

type loopbackPacket struct{ id uint64 }

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

func (o *loopbackOutlet) Mdu() int       { return o.mdu }
func (o *loopbackOutlet) Rtt() float64   { return o.rtt }
func (o *loopbackOutlet) IsUsable() bool { return true }
func (o *loopbackOutlet) TimedOut()      {}
func (o *loopbackOutlet) String() string { return o.name }

func (o *loopbackOutlet) GetPacketState(any) rns.MessageState { return rns.MSGSTATE_DELIVERED }
func (o *loopbackOutlet) GetPacketID(packet any) any          { return packet }

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

func TestChannelExample_StringMessage_RoundTrip(t *testing.T) {
	outA := newLoopbackOutlet("a")
	outB := newLoopbackOutlet("b")

	chA := rns.NewChannel(outA)
	chB := rns.NewChannel(outB)
	outA.connect(chB)
	outB.connect(chA)

	if err := chA.TryRegisterMessageType(&StringMessage{}); err != nil {
		t.Fatalf("RegisterMessageType a: %v", err)
	}
	if err := chB.TryRegisterMessageType(&StringMessage{}); err != nil {
		t.Fatalf("RegisterMessageType b: %v", err)
	}

	got := make(chan string, 1)
	chB.AddMessageHandler(func(m rns.MessageBase) bool {
		sm, ok := m.(*StringMessage)
		if !ok {
			return false
		}
		reply := &StringMessage{Data: "I received \"" + sm.Data + "\" over the link"}
		_ = chB.Send(reply)
		return true
	})
	chA.AddMessageHandler(func(m rns.MessageBase) bool {
		sm, ok := m.(*StringMessage)
		if !ok {
			return false
		}
		got <- sm.Data
		return true
	})

	_, err := chA.TrySend(&StringMessage{Data: "hello", Timestamp: time.Now()})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	select {
	case v := <-got:
		if v != "I received \"hello\" over the link" {
			t.Fatalf("unexpected reply %q", v)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timeout waiting reply")
	}
}
