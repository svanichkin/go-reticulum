package rns

import (
	"testing"
	"time"
)

func TestShouldAnnounceOnInterface_ModesMatchPython(t *testing.T) {
	resetKnownDestinationsForTest()

	prevDestinations := Destinations
	prevPathTable := pathTable

	t.Cleanup(func() {
		Destinations = prevDestinations
		pathTable = prevPathTable
	})

	announceID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	dst, err := NewDestination(announceID, DestinationIN, DestinationSINGLE, "test", "announce")
	if err != nil {
		t.Fatalf("NewDestination: %v", err)
	}
	dstHash := copyBytes(dst.Hash())

	ap := &Interface{Name: "ap", Mode: InterfaceModeAccessPoint}
	roaming := &Interface{Name: "roaming", Mode: InterfaceModeRoaming}
	boundary := &Interface{Name: "boundary", Mode: InterfaceModeBoundary}
	full := &Interface{Name: "full", Mode: InterfaceModeFull}

	nextFull := &Interface{Name: "next-full", Mode: InterfaceModeFull}
	nextRoaming := &Interface{Name: "next-roaming", Mode: InterfaceModeRoaming}
	nextBoundary := &Interface{Name: "next-boundary", Mode: InterfaceModeBoundary}

	cases := []struct {
		name      string
		ifc       *Interface
		attached  *Interface
		localDest bool
		nextHop   *Interface
		want      bool
	}{
		{
			name: "ap blocks announce without attached interface",
			ifc:  ap,
			want: false,
		},
		{
			name:     "ap allows announce with attached interface",
			ifc:      ap,
			attached: &Interface{Name: "attached"},
			want:     true,
		},
		{
			name:      "roaming allows local destination",
			ifc:       roaming,
			localDest: true,
			nextHop:   nextRoaming,
			want:      true,
		},
		{
			name:    "roaming blocks roaming next hop",
			ifc:     roaming,
			nextHop: nextRoaming,
			want:    false,
		},
		{
			name:    "roaming blocks boundary next hop",
			ifc:     roaming,
			nextHop: nextBoundary,
			want:    false,
		},
		{
			name:    "roaming allows full next hop",
			ifc:     roaming,
			nextHop: nextFull,
			want:    true,
		},
		{
			name:      "boundary allows local destination",
			ifc:       boundary,
			localDest: true,
			nextHop:   nextRoaming,
			want:      true,
		},
		{
			name:    "boundary blocks roaming next hop",
			ifc:     boundary,
			nextHop: nextRoaming,
			want:    false,
		},
		{
			name:    "boundary allows full next hop",
			ifc:     boundary,
			nextHop: nextFull,
			want:    true,
		},
		{
			name:    "full mode allows announce",
			ifc:     full,
			nextHop: nextRoaming,
			want:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			Destinations = nil
			pathTable = make(map[hashKey]*PathEntry)

			if tc.localDest {
				Destinations = []*Destination{dst}
			}
			if tc.nextHop != nil {
				key, ok := makeHashKey(dstHash)
				if !ok {
					t.Fatal("destination hash is invalid")
				}
				pathTable[key] = &PathEntry{RecvInterface: tc.nextHop, Timestamp: time.Now()}
			}

			packet := &Packet{DestinationHash: copyBytes(dstHash), AttachedInterface: tc.attached}
			if got := shouldAnnounceOnInterface(packet, tc.ifc, time.Time{}); got != tc.want {
				t.Fatalf("shouldAnnounceOnInterface() = %v, want %v", got, tc.want)
			}
		})
	}
}
