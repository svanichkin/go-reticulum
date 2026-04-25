package rns

import (
	"testing"
	"time"
)

func TestRequestReceiptResourceLifecycle_MatchesPythonResponseTimeSemantics(t *testing.T) {
	l := &Link{
		pendingRequests: make([]*RequestReceipt, 0),
	}

	requestSize := 123
	rr := newRequestReceipt(l, nil, []byte("request-id"), 0.05, &requestSize, nil, nil, nil)
	if rr == nil {
		t.Fatal("newRequestReceipt returned nil")
	}

	if rr.GetResponseTime() != nil {
		t.Fatalf("responseTime=%v, want nil before resource completion", rr.GetResponseTime())
	}

	time.Sleep(10 * time.Millisecond)
	rr.mu.Lock()
	if rr.status != RequestReceiptSent {
		rr.mu.Unlock()
		t.Fatalf("status=%d, want sent before resource completion", rr.GetStatus())
	}
	if rr.startedAt == nil {
		v := float64(time.Now().UnixNano()) / 1e9
		rr.startedAt = &v
	}
	rr.status = RequestReceiptDelivered
	rr.mu.Unlock()
	time.AfterFunc(time.Duration(rr.timeout), func() {
		rr.requestTimedOut()
	})

	if rr.GetStatus() != RequestReceiptDelivered {
		t.Fatalf("status=%d, want delivered", rr.GetStatus())
	}
	if rr.GetResponseTime() != nil {
		t.Fatalf("responseTime=%v, want nil before response is received", rr.GetResponseTime())
	}
	if rr.GetResponse() != nil {
		t.Fatalf("expected nil response before response is received, got %v", rr.GetResponse())
	}
	if rr.Concluded() {
		t.Fatal("request should not be concluded before response is received")
	}

	rr.responseSize = new(int)
	*rr.responseSize = 2
	rr.responseTransferSize = new(int)
	*rr.responseTransferSize = 3
	rr.responseReceived([]byte("ok"), map[string]any{"kind": "test"})

	if rr.GetStatus() != RequestReceiptReady {
		t.Fatalf("status=%d, want ready", rr.GetStatus())
	}
	responseTime := rr.GetResponseTime()
	if responseTime == nil || *responseTime <= 0 {
		t.Fatalf("responseTime=%v, want > 0 after response is received", responseTime)
	}
	if rr.GetResponse() == nil {
		t.Fatal("expected response to be stored")
	}
	if rr.Metadata() == nil {
		t.Fatal("expected metadata to be stored")
	}
	if rr.ResponseSize() == nil || *rr.ResponseSize() != 2 {
		t.Fatalf("response size=%v, want 2", rr.ResponseSize())
	}
	if rr.ResponseTransferSize() == nil || *rr.ResponseTransferSize() != 3 {
		t.Fatalf("response transfer size=%v, want 3", rr.ResponseTransferSize())
	}
	if rr.ResponseConcludedAt() == nil {
		t.Fatal("expected response concluded time to be set")
	}

	if len(l.pendingRequests) != 1 {
		t.Fatalf("pending requests len=%d, want 1 after response", len(l.pendingRequests))
	}

	time.Sleep(100 * time.Millisecond)
	if rr.GetStatus() != RequestReceiptReady {
		t.Fatalf("status=%d after timeout window, want ready", rr.GetStatus())
	}
}
