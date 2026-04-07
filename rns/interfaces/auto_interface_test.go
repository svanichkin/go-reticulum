package interfaces

import "testing"

func TestAutoInterface_StringMatchesPython(t *testing.T) {
	t.Parallel()

	iface := &Interface{
		Name: "auto0",
		Type: "AutoInterface",
	}

	if got := iface.String(); got != "AutoInterface[auto0]" {
		t.Fatalf("unexpected interface string %q", got)
	}

	peer := &Interface{
		Name: "AutoInterfacePeer[en0/fe80::1]",
		Type: "AutoInterfacePeer",
	}

	if got := peer.String(); got != "AutoInterfacePeer[en0/fe80::1]" {
		t.Fatalf("unexpected peer string %q", got)
	}
}
