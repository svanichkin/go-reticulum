package rns

import (
	"bytes"
	"sync/atomic"
	"testing"
	"time"
)

func TestChannelIntegration_EndToEndDeliveryAndOrdering(t *testing.T) {
	oa := newLoopOutlet("a", 65535)
	ob := newLoopOutlet("b", 65535)

	ca := NewChannel(oa)
	cb := NewChannel(ob)
	oa.peer.Store(cb)
	ob.peer.Store(ca)

	if err := ca.RegisterMessageType(&messageTest{}); err != nil {
		t.Fatalf("RegisterMessageType a: %v", err)
	}
	if err := cb.RegisterMessageType(&messageTest{}); err != nil {
		t.Fatalf("RegisterMessageType b: %v", err)
	}

	var recv atomic.Int32
	got := make(chan string, 16)
	cb.AddMessageHandler(func(m MessageBase) bool {
		mt, ok := m.(*messageTest)
		if !ok {
			return false
		}
		recv.Add(1)
		got <- mt.ID
		return true
	})

	// Send 3 messages in order.
	if _, err := ca.Send(&messageTest{ID: "1", Data: "a"}); err != nil {
		t.Fatalf("send1: %v", err)
	}
	if _, err := ca.Send(&messageTest{ID: "2", Data: "b"}); err != nil {
		t.Fatalf("send2: %v", err)
	}
	if _, err := ca.Send(&messageTest{ID: "3", Data: "c"}); err != nil {
		t.Fatalf("send3: %v", err)
	}

	// Wait for reception.
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	var ids []string
	for len(ids) < 3 {
		select {
		case id := <-got:
			ids = append(ids, id)
		case <-deadline.C:
			t.Fatalf("timeout, got=%v recv=%d", ids, recv.Load())
		}
	}

	if !bytes.Equal([]byte(ids[0]+ids[1]+ids[2]), []byte("123")) {
		t.Fatalf("unexpected order: %v", ids)
	}
}

func TestChannelIntegration_IsReadyToSend_WindowBlocks(t *testing.T) {
	// Use an outlet that never marks packets as delivered, so outstanding grows.
	o := &stuckOutlet{name: "stuck", mdu: 65535, rtt: 0.05}
	ch := NewChannel(o)

	if err := ch.RegisterMessageType(&messageTest{}); err != nil {
		t.Fatalf("RegisterMessageType: %v", err)
	}

	// Default window is 2 on non-slow RTT, so first two sends should succeed.
	if _, err := ch.Send(&messageTest{ID: "1", Data: "a"}); err != nil {
		t.Fatalf("send1: %v", err)
	}
	if _, err := ch.Send(&messageTest{ID: "2", Data: "b"}); err != nil {
		t.Fatalf("send2: %v", err)
	}

	if ch.IsReadyToSend() {
		t.Fatalf("expected not ready after filling window")
	}
	if _, err := ch.Send(&messageTest{ID: "3", Data: "c"}); err == nil {
		t.Fatalf("expected ME_LINK_NOT_READY on third send")
	}
}
