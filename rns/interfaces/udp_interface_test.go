package interfaces

import (
	"testing"
)

func TestUDP_ConfigureUDP_Defaults(t *testing.T) {
	t.Parallel()

	var ifc Interface
	if err := ifc.ConfigureUDP("", 4242, "", 0); err != nil {
		t.Fatalf("ConfigureUDP: %v", err)
	}
	if ifc.udpBindAddr == nil || ifc.udpBindAddr.Port != 4242 {
		t.Fatalf("unexpected bind addr: %#v", ifc.udpBindAddr)
	}
	if ifc.udpForwardAddr == nil || ifc.udpForwardAddr.Port != 4242 {
		t.Fatalf("unexpected fwd addr: %#v", ifc.udpForwardAddr)
	}
}

func TestUDP_ConfigureUDP_InvalidListenPort(t *testing.T) {
	t.Parallel()

	var ifc Interface
	// Python parity: allow forward-only interfaces (no bind/listen).
	if err := ifc.ConfigureUDP("0.0.0.0", 0, "255.255.255.255", 0); err != nil {
		t.Fatalf("unexpected error for listen port 0: %v", err)
	}
	if ifc.udpBindAddr != nil {
		t.Fatalf("expected nil bind addr")
	}
	if ifc.udpForwardAddr != nil {
		t.Fatalf("expected nil forward addr when forward port is 0")
	}
}

func TestUDP_String_MatchesPythonStyle(t *testing.T) {
	t.Parallel()

	ifc := &Interface{Name: "u0"}
	if err := ifc.ConfigureUDP("127.0.0.1", 4242, "127.0.0.1", 4242); err != nil {
		t.Fatalf("ConfigureUDP: %v", err)
	}
	if got, want := ifc.String(), "UDPInterface[u0/127.0.0.1:4242]"; got != want {
		t.Fatalf("String()=%q, want %q", got, want)
	}
}
