package rns

import (
	"strings"
	"testing"
	"time"
)

func TestPrettyHexRep_Format(t *testing.T) {
	maybeParallel(t)

	if got := PrettyHexRep([]byte{0x01, 0xAB, 0x00}); got != "<01ab00>" {
		t.Fatalf("unexpected PrettyHexRep: %q", got)
	}
}

func TestHexRep_DelimitToggle(t *testing.T) {
	maybeParallel(t)

	if got := HexRep([]byte{0x01, 0x02, 0x03}); got != "01:02:03" {
		t.Fatalf("unexpected HexRep default: %q", got)
	}
	if got := HexRep([]byte{0x01, 0x02, 0x03}, false); got != "010203" {
		t.Fatalf("unexpected HexRep no-delimit: %q", got)
	}
}

func TestPrettySize_AndSpeed(t *testing.T) {
	maybeParallel(t)

	// Matches Python behaviour: 1024 -> 1.0 KB.
	if got := PrettySize(1024); !strings.Contains(got, "KB") {
		t.Fatalf("expected KB in %q", got)
	}
	if got := PrettySpeed(8); !strings.HasSuffix(got, "bps") {
		t.Fatalf("expected bps suffix in %q", got)
	}
}

func TestSetLogTimeFormat_ResetsOnEmpty(t *testing.T) {
	maybeParallel(t)

	std, prec := LogTimeFormats()
	_ = prec
	SetLogTimeFormat("%Y")
	changed, _ := LogTimeFormats()
	if changed == std {
		t.Fatalf("expected format to change")
	}
	SetLogTimeFormat("")
	reset, _ := LogTimeFormats()
	if reset != std {
		t.Fatalf("expected reset to %q, got %q", std, reset)
	}
}

func TestLogOutputSuppressed_DropsStdoutButKeepsCallback(t *testing.T) {
	prevLevel := LogLevel()
	prevDest := LogDest()
	SetLogLevel(LogDebug)
	SetLogOutputSuppressed(true)
	t.Cleanup(func() {
		SetLogOutputSuppressed(false)
		SetLogDest(prevDest)
		SetLogLevel(prevLevel)
		SetLogDestCallback(nil)
	})

	called := false
	SetLogDestCallback(func(_ int, msg string) {
		called = strings.Contains(msg, "callback-visible")
	})
	Log("callback-visible", LogInfo)
	if !called {
		t.Fatal("expected callback logging to remain active while output is suppressed")
	}
}

func TestTimestampStr_AndPreciseTimestampStr(t *testing.T) {
	maybeParallel(t)

	ts := TimestampStr(0)
	if ts == "" {
		t.Fatalf("expected timestamp string")
	}
	pt := PreciseTimestampStr(float64(time.Now().Unix()))
	if pt == "" {
		t.Fatalf("expected precise timestamp string")
	}
}

func TestSetMTU_Validation(t *testing.T) {
	if err := SetMTU(0); err == nil {
		t.Fatalf("expected error for mtu=0")
	}
	if err := SetMTU(-1); err == nil {
		t.Fatalf("expected error for mtu<0")
	}
}

func TestSetMTU_UpdatesDerivedValues(t *testing.T) {
	prevMTU := MTU
	prevMDU := MDU
	prevLinkMDU := LinkMDU
	t.Cleanup(func() { _ = SetMTU(prevMTU) })

	if err := SetMTU(700); err != nil {
		t.Fatalf("SetMTU: %v", err)
	}
	if MTU != 700 {
		t.Fatalf("expected MTU=700 got %d", MTU)
	}
	if MDU <= 0 || MDU == prevMDU {
		t.Fatalf("expected MDU to change and be >0, got %d", MDU)
	}
	if LinkMDU <= 0 || LinkMDU == prevLinkMDU {
		t.Fatalf("expected LinkMDU to change and be >0, got %d", LinkMDU)
	}
}
