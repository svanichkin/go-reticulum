package interfaces

import (
	"testing"
	"time"
)

func TestKISS_msToKISS(t *testing.T) {
	t.Parallel()

	if got := msToKISS(-5); got != 0 {
		t.Fatalf("msToKISS(-5)=%d want 0", got)
	}
	if got := msToKISS(0); got != 0 {
		t.Fatalf("msToKISS(0)=%d want 0", got)
	}
	if got := msToKISS(9); got != 0 {
		t.Fatalf("msToKISS(9)=%d want 0", got)
	}
	if got := msToKISS(10); got != 1 {
		t.Fatalf("msToKISS(10)=%d want 1", got)
	}
}

func TestKISS_parseKISSConfig_DefaultsAndCaseInsensitiveKeys(t *testing.T) {
	t.Parallel()

	cfg, err := parseKISSConfig("k", map[string]string{
		"PORT":         "/dev/ttyUSB0",
		"SpEeD":        "115200",
		"parity":       "even",
		"stopbits":     "2",
		"id_interval":  "5",
		"id_callsign":  "CALL",
	})
	if err != nil {
		t.Fatalf("parseKISSConfig: %v", err)
	}
	if cfg.Port != "/dev/ttyUSB0" {
		t.Fatalf("unexpected port %q", cfg.Port)
	}
	if cfg.Speed != 115200 {
		t.Fatalf("unexpected speed %d", cfg.Speed)
	}
	if cfg.ReadTimeout != 0 {
		t.Fatalf("expected read timeout default 0, got %v", cfg.ReadTimeout)
	}
	if cfg.FrameTimeout != 100*time.Millisecond {
		t.Fatalf("expected frame timeout default 100ms, got %v", cfg.FrameTimeout)
	}
	if cfg.BeaconEvery != 5*time.Second {
		t.Fatalf("expected beacon every 5s, got %v", cfg.BeaconEvery)
	}
	if string(cfg.BeaconData) != "CALL" {
		t.Fatalf("unexpected beacon data %q", string(cfg.BeaconData))
	}
}

func TestKISS_parseKISSConfig_MissingPort(t *testing.T) {
	t.Parallel()

	if _, err := parseKISSConfig("k", map[string]string{}); err == nil {
		t.Fatalf("expected error when port is missing")
	}
}

func TestKISSInterface_StringMatchesPython(t *testing.T) {
	t.Parallel()

	iface := &Interface{
		Name: "k0",
		Type: "KISSInterface",
	}

	if got := iface.String(); got != "KISSInterface[k0]" {
		t.Fatalf("unexpected string form %q", got)
	}
}
