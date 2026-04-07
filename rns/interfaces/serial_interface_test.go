package interfaces

import (
	"testing"
	"time"
)

func TestSerial_parseSerialConfig_Defaults(t *testing.T) {
	t.Parallel()

	cfg, err := parseSerialConfig("s", map[string]string{
		"port": "/dev/ttyUSB0",
	})
	if err != nil {
		t.Fatalf("parseSerialConfig: %v", err)
	}
	if cfg.Port != "/dev/ttyUSB0" {
		t.Fatalf("unexpected port %q", cfg.Port)
	}
	if cfg.Timeout != 100*time.Millisecond {
		t.Fatalf("unexpected timeout %v", cfg.Timeout)
	}
}

func TestSerial_parseSerialConfig_MissingPort(t *testing.T) {
	t.Parallel()

	if _, err := parseSerialConfig("s", map[string]string{}); err == nil {
		t.Fatalf("expected error when port is missing")
	}
}

func TestSerialInterface_StringMatchesPython(t *testing.T) {
	t.Parallel()

	iface := &Interface{
		Name: "tty0",
		Type: "SerialInterface",
	}

	if got := iface.String(); got != "SerialInterface[tty0]" {
		t.Fatalf("unexpected string form %q", got)
	}
}
