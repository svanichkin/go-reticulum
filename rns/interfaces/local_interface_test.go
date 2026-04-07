package interfaces

import (
	"runtime"
	"strings"
	"testing"
)

func TestLocal_packetAddr_Override(t *testing.T) {
	t.Parallel()

	network, addr := packetAddr(LocalConfig{PktAddrOverride: "tcp:127.0.0.1:1234"})
	if network != "tcp" || addr != "127.0.0.1:1234" {
		t.Fatalf("unexpected override result: %s %q", network, addr)
	}
	network, addr = packetAddr(LocalConfig{PktAddrOverride: "unix:/tmp/sock"})
	if network != "unix" || addr != "/tmp/sock" {
		t.Fatalf("unexpected override result: %s %q", network, addr)
	}
}

func TestLocal_packetAddr_AFUnixAbstract(t *testing.T) {
	t.Parallel()

	network, addr := packetAddr(LocalConfig{UseAFUnix: true, LocalSocketPath: "default"})
	if network != "unix" {
		t.Fatalf("expected unix, got %q", network)
	}
	if runtime.GOOS != "linux" {
		t.Skipf("abstract AF_UNIX addresses are Linux-only (got %s)", runtime.GOOS)
	}
	if !strings.HasPrefix(addr, "\x00rns/") {
		t.Fatalf("expected abstract socket prefix, got %q", addr)
	}
}

func TestLocal_packetAddr_TCPFallback(t *testing.T) {
	t.Parallel()

	network, addr := packetAddr(LocalConfig{UseAFUnix: false, LocalSocketPath: "x", LocalInterfacePort: 4242})
	if network != "tcp" || addr != "127.0.0.1:4242" {
		t.Fatalf("unexpected tcp fallback: %s %q", network, addr)
	}
}

func TestLocal_localClientDisplayName_TCP(t *testing.T) {
	t.Parallel()

	got := localClientDisplayName(LocalConfig{LocalInterfacePort: 4242})
	if got != "LocalInterface[4242]" {
		t.Fatalf("unexpected display name %q", got)
	}
}

func TestLocal_localClientDisplayName_Unix(t *testing.T) {
	t.Parallel()

	got := localClientDisplayName(LocalConfig{UseAFUnix: true, LocalSocketPath: "default"})
	if runtime.GOOS == "linux" {
		if got != "LocalInterface[rns/default]" {
			t.Fatalf("unexpected display name %q", got)
		}
		return
	}
	if !strings.HasPrefix(got, "LocalInterface[") {
		t.Fatalf("unexpected display name %q", got)
	}
}
