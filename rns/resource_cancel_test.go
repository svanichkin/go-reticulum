package rns

import "testing"

func TestResourceCancel_CancelsNextSegmentRecursively(t *testing.T) {
	root := &Resource{
		status:       ResourceTransferring,
		segmentIndex: 1,
		totalSegments: 2,
	}
	next := &Resource{
		status:       ResourceAdvertised,
		segmentIndex: 2,
		totalSegments: 2,
	}
	root.nextSegment = next

	root.Cancel()

	if root.status != ResourceFailed {
		t.Fatalf("root status=%d, want %d", root.status, ResourceFailed)
	}
	if next.status != ResourceFailed {
		t.Fatalf("next segment status=%d, want %d", next.status, ResourceFailed)
	}
}
