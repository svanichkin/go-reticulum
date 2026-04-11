//go:build integration

package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/svanichkin/go-reticulum/internal/cmdtest"
)

func repoRootRNodeconf(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repo root from %s", wd)
		}
		dir = parent
	}
}

var (
	rnodeconfBinOnce sync.Once
	rnodeconfBinPath string
	rnodeconfBinErr  error
)

func getRNodeconfBin(t *testing.T) string {
	t.Helper()
	rnodeconfBinOnce.Do(func() {
		repo := repoRootRNodeconf(t)
		binDir, err := os.MkdirTemp(filepath.Join(repo, ".tmp"), "rnodeconf-bin-")
		if err != nil {
			rnodeconfBinErr = err
			return
		}
		out := cmdtest.Build(t, binDir, "rnodeconf", "./cmd/rnodeconf")
		if _, err := os.Stat(out); err != nil {
			rnodeconfBinErr = err
			return
		}
		rnodeconfBinPath = out
	})
	if rnodeconfBinErr != nil {
		t.Fatalf("build rnodeconf: %v", rnodeconfBinErr)
	}
	return rnodeconfBinPath
}

func newITestRoot(t *testing.T) string {
	t.Helper()
	repo := repoRootRNodeconf(t)
	root, err := os.MkdirTemp(filepath.Join(repo, ".tmp"), "rnodeconf-it-")
	if err != nil {
		t.Fatalf("mktemp: %v", err)
	}
	return root
}

func runRNodeconf(t *testing.T, ctx context.Context, bin, configDir, workDir string, args ...string) (string, int) {
	t.Helper()
	c := exec.CommandContext(ctx, bin, args...)
	c.Dir = workDir

	home := filepath.Join(configDir, ".home")
	rnodeconfDir := filepath.Join(configDir, "cfg")
	_ = os.MkdirAll(home, 0o755)
	_ = os.MkdirAll(rnodeconfDir, 0o755)
	c.Env = append(os.Environ(),
		"HOME="+home,
		"USERPROFILE="+home,
		"RNODECONF_DIR="+rnodeconfDir,
	)
	// Allow commands that prompt for Enter (Python-style UX) to proceed in tests.
	c.Stdin = strings.NewReader("\n")

	out, err := c.CombinedOutput()
	if err == nil {
		return string(out), 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return string(out), ee.ExitCode()
	}
	t.Fatalf("run rnodeconf: %v\n%s", err, string(out))
	return "", -1
}

func TestRNodeconfIntegration_HelpAndVersion(t *testing.T) {
	root := newITestRoot(t)
	bin := getRNodeconfBin(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, code := runRNodeconf(t, ctx, bin, root, root, "--help")
	if code != 0 {
		t.Fatalf("help exit=%d\n%s", code, out)
	}
	// Hidden flags should not appear.
	if strings.Contains(out, "get-target-firmware-hash") || strings.Contains(out, "get-firmware-hash") {
		t.Fatalf("expected hidden flags not to appear in help:\n%s", out)
	}
	if !strings.Contains(out, "RNode Configuration and firmware utility") {
		t.Fatalf("unexpected help output:\n%s", out)
	}

	out, code = runRNodeconf(t, ctx, bin, root, root, "--version")
	if code != 0 {
		t.Fatalf("version exit=%d\n%s", code, out)
	}
	if !strings.HasPrefix(out, "rnodeconf ") {
		t.Fatalf("unexpected version output: %q", out)
	}
}

func TestRNodeconfIntegration_UnknownFlagExit2(t *testing.T) {
	root := newITestRoot(t)
	bin := getRNodeconfBin(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, code := runRNodeconf(t, ctx, bin, root, root, "--nope")
	if code != 2 {
		t.Fatalf("expected exit 2, got %d\n%s", code, out)
	}
}

func TestRNodeconfIntegration_FlashMissingParamsExit68(t *testing.T) {
	root := newITestRoot(t)
	bin := getRNodeconfBin(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, code := runRNodeconf(t, ctx, bin, root, root, "--flash")
	if code != 68 {
		t.Fatalf("expected exit 68, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "Missing parameters") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestRNodeconfIntegration_RNodeconfDirNoAccessExit99(t *testing.T) {
	repo := repoRootRNodeconf(t)
	root, err := os.MkdirTemp(filepath.Join(repo, ".tmp"), "rnodeconf-it-")
	if err != nil {
		t.Fatalf("mktemp: %v", err)
	}
	bin := getRNodeconfBin(t)

	// Create a file where RNODECONF_DIR expects a directory.
	bad := filepath.Join(root, "cfg")
	if err := os.WriteFile(bad, []byte("nope"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c := exec.CommandContext(ctx, bin, "--clear-cache")
	c.Dir = root
	home := filepath.Join(root, ".home")
	_ = os.MkdirAll(home, 0o755)
	c.Env = append(os.Environ(),
		"HOME="+home,
		"USERPROFILE="+home,
		"RNODECONF_DIR="+bad,
	)
	outBytes, err := c.CombinedOutput()
	out := string(outBytes)
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			t.Fatalf("run rnodeconf: %v\n%s", err, out)
		}
	}
	if code != 99 {
		t.Fatalf("expected exit 99, got %d\n%s", code, out)
	}
}

func TestRNodeconfIntegration_TrustKeyStoresFile(t *testing.T) {
	root := newITestRoot(t)
	bin := getRNodeconfBin(t)

	k, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("gen rsa: %v", err)
	}
	pubDER, err := x509.MarshalPKIXPublicKey(&k.PublicKey)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, code := runRNodeconf(t, ctx, bin, root, root, "--trust-key", hex.EncodeToString(pubDER))
	if code != 0 {
		t.Fatalf("trust-key exit=%d\n%s", code, out)
	}
	if !strings.Contains(out, "Trusting key:") || !strings.Contains(out, "Stored at:") {
		t.Fatalf("unexpected output:\n%s", out)
	}
	// Ensure something was written to trusted_keys.
	tkDir := filepath.Join(root, "cfg", "trusted_keys")
	entries, err := os.ReadDir(tkDir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	found := false
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".pubkey") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected a .pubkey file in %s", tkDir)
	}
}

func TestRNodeconfIntegration_TrustKeyInvalidDERExit1(t *testing.T) {
	root := newITestRoot(t)
	bin := getRNodeconfBin(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, code := runRNodeconf(t, ctx, bin, root, root, "--trust-key", "00")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "invalid trusted key") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestRNodeconfIntegration_ListModelsAndShowModel(t *testing.T) {
	root := newITestRoot(t)
	bin := getRNodeconfBin(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, code := runRNodeconf(t, ctx, bin, root, root, "--list-models")
	if code != 0 {
		t.Fatalf("list-models exit=%d\n%s", code, out)
	}
	if !strings.Contains(out, "Model") && !strings.Contains(out, "model") {
		t.Fatalf("unexpected list-models output:\n%s", out)
	}

	out, code = runRNodeconf(t, ctx, bin, root, root, "--show-model", "A1")
	if code != 0 {
		t.Fatalf("show-model exit=%d\n%s", code, out)
	}
	if !strings.Contains(out, "Model") && !strings.Contains(out, "model") {
		t.Fatalf("unexpected show-model output:\n%s", out)
	}

	out, code = runRNodeconf(t, ctx, bin, root, root, "--show-model", "NOPE")
	if code != 1 {
		t.Fatalf("show-model invalid expected exit 1, got %d\n%s", code, out)
	}
}

func TestRNodeconfIntegration_KeyAndShowSigningKey(t *testing.T) {
	root := newITestRoot(t)
	bin := getRNodeconfBin(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, code := runRNodeconf(t, ctx, bin, root, root, "--key")
	if code != 0 {
		t.Fatalf("key exit=%d\n%s", code, out)
	}

	// Key paths under RNODECONF_DIR.
	signerPath := filepath.Join(root, "cfg", "signing.key")
	if _, err := os.Stat(signerPath); err != nil {
		t.Fatalf("expected signing.key to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "cfg", "device.key")); err != nil {
		t.Fatalf("expected device.key to exist: %v", err)
	}

	signerDER, err := os.ReadFile(signerPath)
	if err != nil {
		t.Fatalf("read signing.key: %v", err)
	}
	anyKey, err := x509.ParsePKCS8PrivateKey(signerDER)
	if err != nil {
		t.Fatalf("expected PKCS8 signing.key, got parse error: %v", err)
	}
	priv, ok := anyKey.(*rsa.PrivateKey)
	if !ok {
		t.Fatalf("expected RSA private key, got %T", anyKey)
	}
	if priv.N.BitLen() != 1024 {
		t.Fatalf("expected 1024-bit signing key, got %d", priv.N.BitLen())
	}

	out, code = runRNodeconf(t, ctx, bin, root, root, "--show-signing-key")
	if code != 0 {
		t.Fatalf("show-signing-key exit=%d\n%s", code, out)
	}
	if !strings.Contains(out, "EEPROM Signing Public key:") {
		t.Fatalf("expected EEPROM signing key output, got:\n%s", out)
	}
	if !strings.Contains(out, "Device Signing Public key:") {
		t.Fatalf("expected device signing key output, got:\n%s", out)
	}

	lines := strings.Split(out, "\n")
	var eepromHex, deviceHex string
	for i := 0; i < len(lines); i++ {
		switch strings.TrimSpace(lines[i]) {
		case "EEPROM Signing Public key:":
			for j := i + 1; j < len(lines); j++ {
				val := strings.TrimSpace(lines[j])
				if val != "" {
					eepromHex = val
					break
				}
			}
		case "Device Signing Public key:":
			for j := i + 1; j < len(lines); j++ {
				val := strings.TrimSpace(lines[j])
				if val != "" {
					deviceHex = val
					break
				}
			}
		}
	}
	if eepromHex == "" || deviceHex == "" {
		t.Fatalf("could not parse show-signing-key output:\n%s", out)
	}

	eepromDER, err := hex.DecodeString(eepromHex)
	if err != nil || len(eepromDER) == 0 {
		t.Fatalf("expected DER hex for EEPROM public key, got %q", eepromHex)
	}
	if _, err := x509.ParsePKIXPublicKey(eepromDER); err != nil {
		t.Fatalf("expected valid DER SubjectPublicKeyInfo, got parse error: %v", err)
	}

	deviceDER, err := hex.DecodeString(strings.ReplaceAll(deviceHex, ":", ""))
	if err != nil || len(deviceDER) != 32 {
		t.Fatalf("expected 32-byte colon-delimited device key, got %q", deviceHex)
	}
}

func TestRNodeconfIntegration_ShowSigningKey_NoKeysDoesNotCreateKeys(t *testing.T) {
	root := newITestRoot(t)
	bin := getRNodeconfBin(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, code := runRNodeconf(t, ctx, bin, root, root, "--show-signing-key")
	if code != 0 {
		t.Fatalf("show-signing-key exit=%d\n%s", code, out)
	}
	if !strings.Contains(out, "Could not load EEPROM signing key") {
		t.Fatalf("expected missing EEPROM signing key message, got:\n%s", out)
	}
	if !strings.Contains(out, "Could not load device signing key") {
		t.Fatalf("expected missing device signing key message, got:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(root, "cfg", "signing.key")); err == nil {
		t.Fatalf("did not expect signing.key to be created")
	}
	if _, err := os.Stat(filepath.Join(root, "cfg", "device.key")); err == nil {
		t.Fatalf("did not expect device.key to be created")
	}
}

func TestRNodeconfIntegration_SimPort_InfoAndConfig(t *testing.T) {
	root := newITestRoot(t)
	bin := getRNodeconfBin(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	out, code := runRNodeconf(t, ctx, bin, root, root, "--info", "--port", "sim://esp32")
	if code != 0 {
		t.Fatalf("info exit=%d\n%s", code, out)
	}
	if !strings.Contains(out, "Device:") || !strings.Contains(out, "Firmware:") {
		t.Fatalf("unexpected info output:\n%s", out)
	}
	if !strings.Contains(out, "Device signature: Unverified") {
		t.Fatalf("expected unverified signature in output:\n%s", out)
	}

	out, code = runRNodeconf(t, ctx, bin, root, root, "--config", "--port", "sim://esp32")
	if code != 0 {
		t.Fatalf("config exit=%d\n%s", code, out)
	}
	if !strings.Contains(out, "Device configuration") && !strings.Contains(out, "WiFi") && !strings.Contains(out, "Config") {
		// output format is verbose and may change; just ensure command didn't fail.
		t.Fatalf("unexpected config output:\n%s", out)
	}
}

func TestRNodeconfIntegration_SimPort_ConfigReflectsWiFiAndMasksPSK(t *testing.T) {
	root := newITestRoot(t)
	bin := getRNodeconfBin(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Without --show-psk: PSK should be masked.
	out, code := runRNodeconf(t, ctx, bin, root, root,
		"--port", "sim://esp32",
		"--wifi", "sta",
		"--channel", "6",
		"--ssid", "TestNet",
		"--psk", "password123",
		"--ip", "192.168.1.10",
		"--nm", "255.255.255.0",
		"--bluetooth-on",
		"--ia-disable",
		"--display", "10",
		"--timeout", "5",
		"--rotation", "1",
		"--display-addr", "0x42",
		"--np", "7",
		"--config",
	)
	if code != 0 {
		t.Fatalf("config exit=%d\n%s", code, out)
	}
	if !strings.Contains(out, "WiFi") || !strings.Contains(out, "Enabled (Station)") {
		t.Fatalf("expected station mode in output:\n%s", out)
	}
	if !strings.Contains(out, "SSID") || !strings.Contains(out, "TestNet") {
		t.Fatalf("expected SSID in output:\n%s", out)
	}
	if !strings.Contains(out, "PSK") || !strings.Contains(out, "***********") {
		t.Fatalf("expected masked PSK in output:\n%s", out)
	}
	if !strings.Contains(out, "IP Address") || !strings.Contains(out, "192.168.1.10") {
		t.Fatalf("expected IP in output:\n%s", out)
	}
	if !strings.Contains(out, "Network Mask") || !strings.Contains(out, "255.255.255.0") {
		t.Fatalf("expected NM in output:\n%s", out)
	}
	if !strings.Contains(out, "Bluetooth") || !strings.Contains(out, "Enabled") {
		t.Fatalf("expected bluetooth enabled in output:\n%s", out)
	}
	if !strings.Contains(out, "Interference avoidance") || !strings.Contains(out, "Disabled") {
		t.Fatalf("expected IA disabled in output:\n%s", out)
	}

	// With --show-psk: PSK should be visible.
	out, code = runRNodeconf(t, ctx, bin, root, root,
		"--port", "sim://esp32",
		"--wifi", "sta",
		"--ssid", "TestNet",
		"--psk", "password123",
		"--config",
		"--show-psk",
	)
	if code != 0 {
		t.Fatalf("config(show-psk) exit=%d\n%s", code, out)
	}
	if !strings.Contains(out, "PSK") || !strings.Contains(out, "password123") {
		t.Fatalf("expected unmasked PSK in output:\n%s", out)
	}
}

func TestRNodeconfIntegration_SimPort_WiFiAPModeConfig(t *testing.T) {
	root := newITestRoot(t)
	bin := getRNodeconfBin(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	out, code := runRNodeconf(t, ctx, bin, root, root,
		"--port", "sim://esp32",
		"--wifi", "ap",
		"--ssid", "APNET",
		"--psk", "password123",
		"--config",
	)
	if code != 0 {
		t.Fatalf("wifi ap exit=%d\n%s", code, out)
	}
	if !strings.Contains(out, "Enabled (AP)") {
		t.Fatalf("expected AP mode in output:\n%s", out)
	}
	// AP mode uses the built-in IP/mask defaults in the config view.
	if !strings.Contains(out, "10.0.0.1") || !strings.Contains(out, "255.255.255.0") {
		t.Fatalf("expected AP IP defaults in output:\n%s", out)
	}
}

func TestRNodeconfIntegration_ValidationErrors(t *testing.T) {
	root := newITestRoot(t)
	bin := getRNodeconfBin(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, code := runRNodeconf(t, ctx, bin, root, root, "--ia-enable", "--ia-disable")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d\n%s", code, out)
	}

	out, code = runRNodeconf(t, ctx, bin, root, root, "--display", "300")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d\n%s", code, out)
	}

	out, code = runRNodeconf(t, ctx, bin, root, root, "--ip", "999.1.1.1")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d\n%s", code, out)
	}

	// Channel parsing allows larger values, but device setter only accepts 1..14 -> runtime error.
	out, code = runRNodeconf(t, ctx, bin, root, root, "--port", "sim://esp32", "--channel", "20")
	if code != 2 {
		t.Fatalf("expected exit 2, got %d\n%s", code, out)
	}
}

func TestRNodeconfIntegration_SimPort_ModesAndROMBootstrap(t *testing.T) {
	root := newITestRoot(t)
	bin := getRNodeconfBin(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	out, code := runRNodeconf(t, ctx, bin, root, root,
		"--port", "sim://esp32",
		"--normal",
	)
	if code != 0 {
		t.Fatalf("normal exit=%d\n%s", code, out)
	}
	if !strings.Contains(out, "Device set to normal") {
		t.Fatalf("unexpected output:\n%s", out)
	}

	out, code = runRNodeconf(t, ctx, bin, root, root,
		"--port", "sim://esp32",
		"--tnc",
		"--freq", "868000000",
		"--bw", "125000",
		"--txp", "10",
		"--sf", "7",
		"--cr", "5",
	)
	if code != 0 {
		t.Fatalf("tnc exit=%d\n%s", code, out)
	}
	if !strings.Contains(out, "Device set to TNC operating mode") {
		t.Fatalf("unexpected output:\n%s", out)
	}

	out, code = runRNodeconf(t, ctx, bin, root, root,
		"--port", "sim://esp32",
		"--rom",
		"--product", "0x03",
		"--model", "A1",
		"--hwrev", "1",
	)
	if code != 0 {
		t.Fatalf("rom bootstrap exit=%d\n%s", code, out)
	}
	if !strings.Contains(out, "Bootstrapping device EEPROM") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestRNodeconfIntegration_SimPort_ExtractBranches(t *testing.T) {
	root := newITestRoot(t)
	bin := getRNodeconfBin(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	out, code := runRNodeconf(t, ctx, bin, root, root, "--extract", "--port", "sim://esp32?extract=fail182")
	if code != 182 {
		t.Fatalf("expected exit 182, got %d\n%s", code, out)
	}

	out, code = runRNodeconf(t, ctx, bin, root, root, "--extract", "--port", "sim://esp32?extract=fail180")
	if code != 180 {
		t.Fatalf("expected exit 180, got %d\n%s", code, out)
	}

	out, code = runRNodeconf(t, ctx, bin, root, root, "--extract", "--port", "sim://avr")
	if code != 170 {
		t.Fatalf("expected exit 170, got %d\n%s", code, out)
	}
}

func TestRNodeconfIntegration_SimPort_FlashBranchExit1(t *testing.T) {
	root := newITestRoot(t)
	bin := getRNodeconfBin(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// flash requires extracted firmware presence; create it via simulated extraction first.
	_, code := runRNodeconf(t, ctx, bin, root, root, "--extract", "--port", "sim://esp32")
	if code != 0 {
		t.Fatalf("extract setup failed exit=%d", code)
	}
	out, code := runRNodeconf(t, ctx, bin, root, root, "--flash", "--use-extracted", "--port", "sim://esp32?flash=fail1")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d\n%s", code, out)
	}
}

func TestRNodeconfIntegration_SimPort_EEPROMWipe(t *testing.T) {
	root := newITestRoot(t)
	bin := getRNodeconfBin(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	out, code := runRNodeconf(t, ctx, bin, root, root, "--eeprom-wipe", "--port", "sim://esp32")
	if code != 0 {
		t.Fatalf("eeprom-wipe exit=%d\n%s", code, out)
	}
	if !strings.Contains(out, "EEPROM wipe completed") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestRNodeconfIntegration_SimPort_NotProvisionedExit77_79(t *testing.T) {
	root := newITestRoot(t)
	bin := getRNodeconfBin(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	out, code := runRNodeconf(t, ctx, bin, root, root, "--firmware-hash", strings.Repeat("00", 32), "--port", "sim://esp32?provisioned=0")
	if code != 77 {
		t.Fatalf("expected exit 77, got %d\n%s", code, out)
	}

	out, code = runRNodeconf(t, ctx, bin, root, root, "--sign", "--port", "sim://esp32?provisioned=0")
	if code != 79 {
		t.Fatalf("expected exit 79, got %d\n%s", code, out)
	}
}

func TestRNodeconfIntegration_SimPort_HiddenHashFlagsWork(t *testing.T) {
	root := newITestRoot(t)
	bin := getRNodeconfBin(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	out, code := runRNodeconf(t, ctx, bin, root, root, "-K", "--port", "sim://esp32")
	if code != 0 {
		t.Fatalf("-K exit=%d\n%s", code, out)
	}
	out, code = runRNodeconf(t, ctx, bin, root, root, "-L", "--port", "sim://esp32")
	if code != 0 {
		t.Fatalf("-L exit=%d\n%s", code, out)
	}
}

func TestRNodeconfIntegration_SimPort_UpdateAndAutoinstall(t *testing.T) {
	root := newITestRoot(t)
	bin := getRNodeconfBin(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Prepare extracted firmware.
	_, code := runRNodeconf(t, ctx, bin, root, root, "--extract", "--port", "sim://esp32")
	if code != 0 {
		t.Fatalf("extract setup failed exit=%d", code)
	}

	out, code := runRNodeconf(t, ctx, bin, root, root, "--update", "--use-extracted", "--port", "sim://esp32")
	if code != 0 {
		t.Fatalf("update exit=%d\n%s", code, out)
	}

	// Autoinstall also ROM bootstraps when parameters provided.
	// NOTE: use a fresh sim port instance to avoid cross-test state issues.
	out, code = runRNodeconf(t, ctx, bin, root, root,
		"--autoinstall", "--use-extracted",
		"--product", "0x03", "--model", "A1", "--hwrev", "1",
		"--port", "sim://esp32?session=autoinstall",
	)
	if code != 0 {
		t.Fatalf("autoinstall exit=%d\n%s", code, out)
	}
}

func TestRNodeconfIntegration_SimPort_SettingsAndEEPROMBackup(t *testing.T) {
	root := newITestRoot(t)
	bin := getRNodeconfBin(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	out, code := runRNodeconf(t, ctx, bin, root, root,
		"--bluetooth-on",
		"--wifi", "OFF",
		"--display", "10",
		"--np", "10",
		"--eeprom-backup",
		"--port", "sim://esp32",
	)
	if code != 0 {
		t.Fatalf("settings exit=%d\n%s", code, out)
	}

	// Ensure EEPROM backup was created in RNODECONF_DIR.
	eepromDir := filepath.Join(root, "cfg", "eeprom")
	entries, err := os.ReadDir(eepromDir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	found := false
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "eeprom_") && strings.HasSuffix(e.Name(), ".bin") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected EEPROM backup file in %s", eepromDir)
	}
}

func TestRNodeconfIntegration_SimPort_MissingExternalFlashTools(t *testing.T) {
	root := newITestRoot(t)
	bin := getRNodeconfBin(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// AVR: simulate missing avrdude.
	out, code := runRNodeconf(t, ctx, bin, root, root, "--flash", "--fw-url", "file:///tmp/rnode_firmware.hex", "--port", "sim://avr?flash=missing_avrdude")
	if code != 0 {
		t.Fatalf("expected exit 0 (python graceful_exit default), got %d\n%s", code, out)
	}
	if !strings.Contains(out, "avrdude") {
		t.Fatalf("expected avrdude guidance:\n%s", out)
	}

	// NRF52: simulate missing adafruit-nrfutil.
	out, code = runRNodeconf(t, ctx, bin, root, root, "--flash", "--fw-url", "file:///tmp/fw.zip", "--port", "sim://nrf52?flash=missing_nrfutil")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "adafruit-nrfutil") {
		t.Fatalf("expected nrfutil guidance:\n%s", out)
	}
}
