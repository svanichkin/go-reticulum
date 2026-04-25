package rns

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	configobj "github.com/svanichkin/configobj"
	ifaces "github.com/svanichkin/go-reticulum/rns/interfaces"
)

func TestBringUpSystemInterfaces_LoadsExternalPythonInterface(t *testing.T) {
	if _, err := externalInterfacePythonLookup(); err != nil {
		t.Skipf("python3 not available: %v", err)
	}

	cfgDir := t.TempDir()
	ifDir := filepath.Join(cfgDir, "interfaces")
	if err := os.MkdirAll(ifDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	module := `
class FooInterface(Interface):
    DEFAULT_IFAC_SIZE = 11

    def __init__(self, owner, configuration):
        super().__init__()
        self.owner = owner
        self.name = configuration["name"]
        self.online = True
        self.bitrate = 4242
        self.HW_MTU = 333

    def process_outgoing(self, data):
        self.owner.inbound(b"py:" + data, self)

interface_class = FooInterface
`
	if err := os.WriteFile(filepath.Join(ifDir, "FooInterface.py"), []byte(module), 0o600); err != nil {
		t.Fatalf("WriteFile(interface): %v", err)
	}

	cfgPath := filepath.Join(cfgDir, "config")
	cfg := `
[interfaces]
  [[Foo]]
    enabled = True
    type = FooInterface
    selected_interface_mode = gateway
    outgoing = False
`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatalf("WriteFile(config): %v", err)
	}
	parsed, err := configobj.Load(cfgPath)
	if err != nil {
		t.Fatalf("configobj.Load: %v", err)
	}

	prevInterfaces := Interfaces
	Interfaces = nil
	t.Cleanup(func() {
		for _, ifc := range Interfaces {
			if ifc != nil {
				ifc.Detach()
			}
		}
		Interfaces = prevInterfaces
	})

	prevInbound := ifaces.InboundHandler
	gotInbound := make(chan []byte, 1)
	ifaces.InboundHandler = func(raw []byte, _ *ifaces.Interface) {
		gotInbound <- append([]byte(nil), raw...)
	}
	t.Cleanup(func() {
		ifaces.InboundHandler = prevInbound
	})

	r := &Reticulum{
		Config:                parsed,
		ConfigPath:            cfgPath,
		InterfacePath:         ifDir,
		PanicOnInterfaceError: false,
	}
	if err := r.synthesizeConfiguredInterfaces(); err != nil {
		t.Fatalf("synthesizeConfiguredInterfaces(): %v", err)
	}
	if len(Interfaces) != 1 {
		t.Fatalf("expected 1 interface, got %d", len(Interfaces))
	}

	ifc := Interfaces[0]
	if ifc == nil {
		t.Fatal("expected non-nil interface")
	}
	if ifc.Type != "FooInterface" {
		t.Fatalf("interface type=%q, want FooInterface", ifc.Type)
	}
	if ifc.Mode != InterfaceModeGateway {
		t.Fatalf("interface mode=%d, want %d", ifc.Mode, InterfaceModeGateway)
	}
	if ifc.OUT {
		t.Fatal("expected outgoing=false to be applied to external Python interface")
	}
	if !ifc.Online {
		t.Fatal("expected external Python interface to be online")
	}
	if ifc.Bitrate != 4242 {
		t.Fatalf("interface bitrate=%d, want 4242", ifc.Bitrate)
	}
	if ifc.HWMTU != 333 {
		t.Fatalf("interface hw mtu=%d, want 333", ifc.HWMTU)
	}
	if ifc.IFACSize != 11 {
		t.Fatalf("interface IFAC size=%d, want 11", ifc.IFACSize)
	}

	ifc.ProcessOutgoing([]byte("ping"))

	select {
	case raw := <-gotInbound:
		if string(raw) != "py:ping" {
			t.Fatalf("unexpected inbound payload %q", string(raw))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for inbound payload from external Python interface")
	}
}
