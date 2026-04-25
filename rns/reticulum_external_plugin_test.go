//go:build linux || darwin || freebsd

package rns

import (
	"os"
	"path/filepath"
	"testing"

	configobj "github.com/svanichkin/configobj"
)

func TestBringUpSystemInterfaces_LoadsExternalGoPlugin(t *testing.T) {
	cfgDir := t.TempDir()
	ifDir := filepath.Join(cfgDir, "interfaces")
	if err := os.MkdirAll(ifDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	soPath := filepath.Join(ifDir, "FooInterface.so")
	if err := os.WriteFile(soPath, []byte("stub"), 0o600); err != nil {
		t.Fatalf("WriteFile(plugin): %v", err)
	}

	prevLoader := externalInterfacePluginLoader
	externalInterfacePluginLoader = func(path string) (externalInterfaceFactory, error) {
		if path != soPath {
			t.Fatalf("plugin path=%q, want %q", path, soPath)
		}
		return func(name string, config map[string]string) (*Interface, error) {
			return &Interface{
				Name:              name,
				Type:              "FooInterface",
				DriverImplemented: true,
				Online:            true,
			}, nil
		}, nil
	}
	t.Cleanup(func() {
		externalInterfacePluginLoader = prevLoader
	})

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
	if ifc.Name != "Foo" {
		t.Fatalf("interface name=%q, want Foo", ifc.Name)
	}
	if ifc.Type != "FooInterface" {
		t.Fatalf("interface type=%q, want FooInterface", ifc.Type)
	}
	if ifc.Mode != InterfaceModeGateway {
		t.Fatalf("interface mode=%d, want %d", ifc.Mode, InterfaceModeGateway)
	}
	if ifc.OUT {
		t.Fatal("expected outgoing=false to be applied to external plugin interface")
	}
}
