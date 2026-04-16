package rns

import "testing"

func TestDestinationAnnounce_CachesAnnounceDataForTagWithoutPathResponse(t *testing.T) {
	id, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	dst, err := NewDestination(id, DestinationIN, DestinationSINGLE, "test", "announce")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}

	tag := []byte{0x01, 0x02, 0x03}
	packet := dst.Announce([]byte("payload"), false, nil, tag, false)
	if packet == nil {
		t.Fatal("Announce returned nil")
	}

	entry := dst.pathResponses[string(tag)]
	if entry == nil {
		t.Fatal("announce data was not cached for tag")
	}
	if len(entry.Data) == 0 {
		t.Fatal("cached announce data is empty")
	}
}
