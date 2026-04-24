package rns

import "testing"

func TestResourceCancel_CancelsNextSegmentRecursively(t *testing.T) {
	link := &Link{}
	root := &Resource{
		Status:        ResourceTransferring,
		segmentIndex:  1,
		totalSegments: 2,
		Link:          link,
	}
	next := &Resource{
		Status:        ResourceAdvertised,
		segmentIndex:  2,
		totalSegments: 2,
		Link:          link,
	}
	root.nextSegment = next

	root.Cancel()

	if root.Status != ResourceFailed {
		t.Fatalf("root status=%d, want %d", root.Status, ResourceFailed)
	}
	if next.Status != ResourceFailed {
		t.Fatalf("next segment status=%d, want %d", next.Status, ResourceFailed)
	}
}
