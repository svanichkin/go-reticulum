package interfaces

import (
	"net"
	"strings"
	"testing"
)

func TestTCPBind_AddressForHost_LiteralIPv4(t *testing.T) {
	addr, err := tcpAddressForHost("127.0.0.1", 1234, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if addr != "127.0.0.1:1234" {
		t.Fatalf("unexpected addr: %q", addr)
	}
}

func TestTCPBind_AddressForHost_LiteralIPv6(t *testing.T) {
	addr, err := tcpAddressForHost("2001:db8::1", 1234, true)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if addr != "[2001:db8::1]:1234" {
		t.Fatalf("unexpected addr: %q", addr)
	}
}

func TestTCPBind_AddressForHost_PreservesHostname(t *testing.T) {
	addr, err := tcpAddressForHost("localhost", 1234, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if addr != "localhost:1234" {
		t.Fatalf("unexpected addr: %q", addr)
	}
}

func TestTCPBind_AddressForHost_NoFallbackOnLookupError(t *testing.T) {
	_, err := tcpAddressForHost("nonexistent.invalid", 1234, false)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestTCPBind_AddressForHost_NoSuitableFamily(t *testing.T) {
	if _, err := tcpAddressForHost("127.0.0.1", 1234, true); err == nil {
		t.Fatalf("expected error for ipv6 preference on ipv4-only literal")
	}
}

func TestTCPBind_AddressForInterface_Validates(t *testing.T) {
	_, err := tcpAddressForInterface("", 1234, false)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTCPBind_AddressForInterface_LoopbackAllowedLikePython(t *testing.T) {
	ifc, err := net.InterfaceByName("lo0")
	if err != nil {
		ifc, err = net.InterfaceByName("lo")
	}
	if err != nil {
		t.Skip("loopback interface not found")
	}

	addr, err := tcpAddressForInterface(ifc.Name, 1234, false)
	if err != nil {
		t.Fatalf("unexpected error for loopback interface %q: %v", ifc.Name, err)
	}
	if !strings.HasSuffix(addr, ":1234") {
		t.Fatalf("unexpected addr: %q", addr)
	}
}
