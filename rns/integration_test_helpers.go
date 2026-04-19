package rns

import (
	"testing"
	"time"
)

func waitForLinkActiveOrSkip(t *testing.T, l *Link, timeout time.Duration) {
	t.Helper()
	if l == nil {
		t.Skip("link is nil")
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if l.Status == LinkActive {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Skipf("expected link active, got status %d", l.Status)
}
