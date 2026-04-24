package rns

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestLinkTeardown_CancelsTrackedResources(t *testing.T) {
	l := &Link{
		Status:            LinkActive,
		Initiator:         true,
		watchdogStop:      make(chan struct{}),
		incomingResources: make([]*Resource, 0),
		outgoingResources: make([]*Resource, 0),
	}

	var incomingCalled atomic.Int32
	var outgoingCalled atomic.Int32

	incoming := &Resource{
		Link:      l,
		initiator: false,
		Status:    ResourceTransferring,
		callback: func(*Resource) {
			incomingCalled.Add(1)
		},
	}
	outgoing := &Resource{
		Link:      l,
		initiator: true,
		Status:    ResourceTransferring,
		callback: func(*Resource) {
			outgoingCalled.Add(1)
		},
	}

	l.incomingResources = append(l.incomingResources, incoming)
	l.outgoingResources = append(l.outgoingResources, outgoing)

	l.Teardown()

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if incomingCalled.Load() > 0 && outgoingCalled.Load() > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	if incoming.Status != ResourceFailed {
		t.Fatalf("incoming resource status=%d, want failed", incoming.Status)
	}
	if outgoing.Status != ResourceFailed {
		t.Fatalf("outgoing resource status=%d, want failed", outgoing.Status)
	}
	if incomingCalled.Load() == 0 {
		t.Fatalf("expected incoming resource callback after teardown")
	}
	if outgoingCalled.Load() == 0 {
		t.Fatalf("expected outgoing resource callback after teardown")
	}
}

func TestLinkParityHelpers_IdleAndGetterMethods(t *testing.T) {
	now := time.Now()
	l := &Link{
		Status:            LinkActive,
		Mode:              LinkModeDefault,
		MTU:               777,
		MDU:               700,
		ExpectedRate:      1234,
		EstablishmentRate: 100,
		lastInbound:       now.Add(-3 * time.Second),
		lastOutbound:      now.Add(-5 * time.Second),
		lastData:          now.Add(-7 * time.Second),
		activatedAt:       now.Add(-2 * time.Second),
	}

	if got := l.GetMode(); got != LinkModeDefault {
		t.Fatalf("GetMode=%d, want %d", got, LinkModeDefault)
	}
	if got := l.NoInboundFor(); got < 1.5 || got > 2.5 {
		t.Fatalf("NoInboundFor=%f, want about 2s", got)
	}
	if got := l.NoOutboundFor(); got < 4.5 || got > 5.5 {
		t.Fatalf("NoOutboundFor=%f, want about 5s", got)
	}
	if got := l.NoDataFor(); got < 6.5 || got > 7.5 {
		t.Fatalf("NoDataFor=%f, want about 7s", got)
	}
	if got := l.InactiveFor(); got < 1.5 || got > 2.5 {
		t.Fatalf("InactiveFor=%f, want about 2s", got)
	}

	mtu := l.GetMTU()
	if mtu == nil || *mtu != 777 {
		t.Fatalf("GetMTU=%v, want 777", mtu)
	}
	mdu := l.GetMDU()
	if mdu == nil || *mdu != 700 {
		t.Fatalf("GetMDU=%v, want 700", mdu)
	}
	expected := l.GetExpectedRate()
	if expected == nil || *expected != 1234 {
		t.Fatalf("GetExpectedRate=%v, want 1234", expected)
	}
	estRate := l.GetEstablishmentRate()
	if estRate == nil || *estRate != 800 {
		t.Fatalf("GetEstablishmentRate=%v, want 800", estRate)
	}

	l.Status = LinkClosed
	if l.GetMTU() != nil || l.GetMDU() != nil || l.GetExpectedRate() != nil {
		t.Fatalf("expected established-link getters to return nil when closed")
	}
}

func TestLinkRequest_ReturnsFalseOnFailure(t *testing.T) {
	l := &Link{}
	result := l.Request("", nil, nil, nil, nil, 0)
	if b, ok := result.(bool); !ok || b {
		t.Fatalf("Request()=%T(%v), want literal false", result, result)
	}
	if rr, ok := result.(*RequestReceipt); ok && rr != nil {
		t.Fatalf("type assertion on false returned %v, want nil", rr)
	}
}
