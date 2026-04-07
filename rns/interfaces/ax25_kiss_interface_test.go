package interfaces

import (
	"testing"
	"time"
)

func TestAX25_parseIntOr_parseBoolOr(t *testing.T) {
	t.Parallel()

	if got := parseIntOr(" 123 ", 7); got != 123 {
		t.Fatalf("parseIntOr=%d want 123", got)
	}
	if got := parseIntOr("nope", 7); got != 7 {
		t.Fatalf("parseIntOr=%d want 7", got)
	}
	if got := parseBoolOr("true", false); got != true {
		t.Fatalf("parseBoolOr=true want true")
	}
	if got := parseBoolOr("1", false); got != true {
		t.Fatalf("parseBoolOr=1 want true")
	}
	if got := parseBoolOr("0", true); got != false {
		t.Fatalf("parseBoolOr=0 want false")
	}
	if got := parseBoolOr("nope", true); got != true {
		t.Fatalf("parseBoolOr=nope want default")
	}
}

func TestAX25_parseAX25KISSConfig_Defaults(t *testing.T) {
	t.Parallel()

	cfg, err := parseAX25KISSConfig("a", map[string]string{
		"port":     "/dev/ttyUSB0",
		"callsign": "N0CALL",
		"ssid":     "0",
	})
	if err != nil {
		t.Fatalf("parseAX25KISSConfig: %v", err)
	}
	if cfg.Port != "/dev/ttyUSB0" {
		t.Fatalf("unexpected port %q", cfg.Port)
	}
	if cfg.ReadTimeout != 100*time.Millisecond {
		t.Fatalf("unexpected read timeout %v", cfg.ReadTimeout)
	}
	if cfg.FrameTimeout != 100*time.Millisecond {
		t.Fatalf("unexpected frame timeout %v", cfg.FrameTimeout)
	}
}

func TestAX25_parseAX25KISSConfig_MissingPort(t *testing.T) {
	t.Parallel()

	if _, err := parseAX25KISSConfig("a", map[string]string{"callsign": "N0CALL"}); err == nil {
		t.Fatalf("expected error when port is missing")
	}
}

func TestAX25KISSInterface_StringMatchesPython(t *testing.T) {
	t.Parallel()

	iface := &Interface{
		Name: "ax0",
		Type: "AX25KISSInterface",
	}

	if got := iface.String(); got != "AX25KISSInterface[ax0]" {
		t.Fatalf("unexpected string form %q", got)
	}
}
