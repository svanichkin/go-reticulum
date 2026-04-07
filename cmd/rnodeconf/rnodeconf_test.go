package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareStoragePaths_EnvOverride(t *testing.T) {
	base := t.TempDir()
	t.Setenv("RNODECONF_DIR", base)

	paths, err := prepareStoragePaths()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if paths.ConfigDir != base {
		t.Fatalf("ConfigDir got %q want %q", paths.ConfigDir, base)
	}
	for _, dir := range []string{
		paths.ConfigDir,
		paths.UpdateDir,
		paths.FirmwareDir,
		paths.ExtractedDir,
		paths.TrustedKeysDir,
		paths.EEPROMDir,
		paths.DeviceDBDir,
	} {
		st, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("expected %s to exist: %v", dir, err)
		}
		if !st.IsDir() {
			t.Fatalf("expected %s to be dir", dir)
		}
	}
}

func TestLocalFirmwarePathFromURL(t *testing.T) {
	if p, ok := localFirmwarePathFromURL("file:///tmp/fw.zip"); !ok || p != "/tmp/fw.zip" {
		t.Fatalf("got (%q,%v)", p, ok)
	}
	if p, ok := localFirmwarePathFromURL("/tmp/fw.zip"); !ok || p != "/tmp/fw.zip" {
		t.Fatalf("got (%q,%v)", p, ok)
	}
	if _, ok := localFirmwarePathFromURL("https://example.com/fw.zip"); ok {
		t.Fatalf("expected non-local url")
	}
}

func TestParseByteSpec(t *testing.T) {
	if b, err := parseByteSpec("0x1b"); err != nil || b != 0x1b {
		t.Fatalf("got (%v,%v)", b, err)
	}
	// Without prefix, values are treated as hex first for parity with upstream CLI usage.
	if b, err := parseByteSpec("27"); err != nil || b != 0x27 {
		t.Fatalf("got (%v,%v)", b, err)
	}
	if b, err := parseByteSpec("0b1010"); err != nil || b != 10 {
		t.Fatalf("got (%v,%v)", b, err)
	}
	if _, err := parseByteSpec("nope"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestNextDeviceSerial(t *testing.T) {
	p := filepath.Join(t.TempDir(), "serial.counter")

	first, err := nextDeviceSerial(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if first != 1 {
		t.Fatalf("got %d want 1", first)
	}

	second, err := nextDeviceSerial(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if second != 2 {
		t.Fatalf("got %d want 2", second)
	}
}

func TestExtractedFirmwareHash(t *testing.T) {
	base := t.TempDir()
	paths := storagePaths{ExtractedDir: base}

	hash := make([]byte, 32)
	for i := range hash {
		hash[i] = byte(i)
	}
	line := "1.2.3 " + hex.EncodeToString(hash) + "\n"
	if err := os.WriteFile(filepath.Join(base, "extracted_rnode_firmware.version"), []byte(line), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := extractedFirmwareHash(paths)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hex.EncodeToString(got) != hex.EncodeToString(hash) {
		t.Fatalf("got %x want %x", got, hash)
	}
}

func TestEncodeIPv4(t *testing.T) {
	if got, err := encodeIPv4("none"); err != nil || hex.EncodeToString(got) != "00000000" {
		t.Fatalf("got (%x,%v)", got, err)
	}
	got, err := encodeIPv4("192.168.0.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hex.EncodeToString(got) != "c0a80001" {
		t.Fatalf("got %x want c0a80001", got)
	}
}

func TestParseWifiMode(t *testing.T) {
	if got, err := parseWifiMode("sta"); err != nil || got != "STATION" {
		t.Fatalf("got (%q,%v)", got, err)
	}
	if got, err := parseWifiMode("AP"); err != nil || got != "AP" {
		t.Fatalf("got (%q,%v)", got, err)
	}
	if got, err := parseWifiMode("off"); err != nil || got != "OFF" {
		t.Fatalf("got (%q,%v)", got, err)
	}
	if _, err := parseWifiMode("wat"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestParseModelCode(t *testing.T) {
	if got, err := parseModelCode("A1"); err != nil || got != 0xA1 {
		t.Fatalf("got (%v,%v)", got, err)
	}
	if _, err := parseModelCode(""); err == nil {
		t.Fatalf("expected error")
	}
}

func TestSerialPortDisplayLabel(t *testing.T) {
	t.Run("by-id includes basename label", func(t *testing.T) {
		in := "/dev/serial/by-id/usb-Silicon_Labs_CP2102N_USB_to_UART-if00-port0"
		want := in + " (usb-Silicon_Labs_CP2102N_USB_to_UART-if00-port0)"
		if got := serialPortDisplayLabel(in); got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})

	t.Run("plain path unchanged", func(t *testing.T) {
		in := "/dev/ttyUSB0"
		if got := serialPortDisplayLabel(in); got != in {
			t.Fatalf("got %q want %q", got, in)
		}
	})
}

func TestStoreTrustedKey(t *testing.T) {
	k, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("gen rsa: %v", err)
	}
	key, err := x509.MarshalPKIXPublicKey(&k.PublicKey)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	sum := sha256.Sum256(key)
	wantFP := hex.EncodeToString(sum[:])

	dir := t.TempDir()
	path, fp, err := storeTrustedKey(hex.EncodeToString(key), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fp != wantFP {
		t.Fatalf("fp got %q want %q", fp, wantFP)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(b) != string(key) {
		t.Fatalf("file contents mismatch")
	}
}

func TestStoreTrustedKey_InvalidDER(t *testing.T) {
	// Valid hex, but not DER SPKI.
	dir := t.TempDir()
	_, _, err := storeTrustedKey(hex.EncodeToString([]byte("nope")), dir)
	if err == nil {
		t.Fatalf("expected error")
	}
}
