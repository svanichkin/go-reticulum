package rns

import (
	"os"
	"path/filepath"
	"testing"

	configobj "github.com/svanichkin/configobj"
)

func TestBringUpSystemInterfaces_ModeAndBitrateAliases(t *testing.T) {
	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "config")
	cfg := `
[interfaces]
  [[Alias]]
    enabled = True
    type = WeaveInterface
    port = fake
    selected_interface_mode = gateway
    configured_bitrate = 12345
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

	r := &Reticulum{
		Config:                parsed,
		ConfigPath:            cfgPath,
		PanicOnInterfaceError: false,
	}
	if err := r.synthesizeConfiguredInterfaces(); err != nil {
		t.Fatalf("synthesizeConfiguredInterfaces: %v", err)
	}
	if len(Interfaces) != 1 {
		t.Fatalf("expected 1 interface, got %d", len(Interfaces))
	}
	ifc := Interfaces[0]
	if ifc == nil {
		t.Fatalf("expected non-nil interface")
	}
	if ifc.Mode != InterfaceModeGateway {
		t.Fatalf("expected mode=%d got %d", InterfaceModeGateway, ifc.Mode)
	}
	if ifc.Bitrate != 12345 {
		t.Fatalf("expected bitrate=12345 got %d", ifc.Bitrate)
	}
}

func TestBringUpSystemInterfaces_ExternalInterfacePyInitFailureIsSkipped(t *testing.T) {
	cfgDir := t.TempDir()
	ifDir := filepath.Join(cfgDir, "interfaces")
	if err := os.MkdirAll(ifDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	module := "raise RuntimeError('boom')\n"
	if err := os.WriteFile(filepath.Join(ifDir, "FooInterface.py"), []byte(module), 0o600); err != nil {
		t.Fatalf("WriteFile(interface): %v", err)
	}

	cfgPath := filepath.Join(cfgDir, "config")
	cfg := `
[interfaces]
  [[Foo]]
    enabled = True
    type = FooInterface
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

	r := &Reticulum{
		Config:                parsed,
		ConfigPath:            cfgPath,
		InterfacePath:         ifDir,
		PanicOnInterfaceError: false,
	}
	if err := r.synthesizeConfiguredInterfaces(); err != nil {
		t.Fatalf("synthesizeConfiguredInterfaces(): %v", err)
	}
	if len(Interfaces) != 0 {
		t.Fatalf("expected failed external Python interface to be skipped, got %d interfaces", len(Interfaces))
	}
}
