package rns

import (
	"os"
	"path/filepath"
	"testing"

	configobj "github.com/svanichkin/configobj"
)

func TestReticulumBringUpSystemInterfaces_FatalOnErrorByDefault(t *testing.T) {
	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "config")
	cfg := `
[interfaces]
  [[Bad]]
    enabled = True
    type = RNodeInterface
`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatalf("WriteFile(config): %v", err)
	}

	parsed, err := configobj.Load(cfgPath)
	if err != nil {
		t.Fatalf("configobj.Load: %v", err)
	}

	r := &Reticulum{
		Config:                parsed,
		ConfigPath:            cfgPath,
		PanicOnInterfaceError: true,
	}

	if err := r.synthesizeConfiguredInterfaces(); err == nil {
		t.Fatalf("expected synthesizeConfiguredInterfaces to fail for enabled misconfigured interface")
	}
}

func TestReticulumBringUpSystemInterfaces_FatalEvenWhenPanicOnInterfaceErrorFalse(t *testing.T) {
	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "config")
	cfg := `
[interfaces]
  [[Bad]]
    enabled = True
    type = RNodeInterface
`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatalf("WriteFile(config): %v", err)
	}

	parsed, err := configobj.Load(cfgPath)
	if err != nil {
		t.Fatalf("configobj.Load: %v", err)
	}

	r := &Reticulum{
		Config:                parsed,
		ConfigPath:            cfgPath,
		PanicOnInterfaceError: false,
	}

	if err := r.synthesizeConfiguredInterfaces(); err == nil {
		t.Fatalf("expected synthesizeConfiguredInterfaces to fail even when PanicOnInterfaceError is false")
	}
}
