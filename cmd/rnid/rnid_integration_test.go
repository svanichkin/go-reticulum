//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	rns "github.com/svanichkin/go-reticulum/rns"
)

func repoRootRNID(t *testing.T) string {
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

func buildRNID(t *testing.T, binDir string) string {
	t.Helper()
	name := "rnid"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	out := filepath.Join(binDir, name)
	gocache := filepath.Join(binDir, ".gocache")
	gotmp := filepath.Join(binDir, ".gotmp")
	if err := os.MkdirAll(gocache, 0o755); err != nil {
		t.Fatalf("mkdir gocache: %v", err)
	}
	if err := os.MkdirAll(gotmp, 0o755); err != nil {
		t.Fatalf("mkdir gotmp: %v", err)
	}
	cmd := exec.Command("go", "build", "-o", out, "./cmd/rnid")
	cmd.Dir = repoRootRNID(t)
	cmd.Env = append(os.Environ(),
		"GOCACHE="+gocache,
		"GOTMPDIR="+gotmp,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("build rnid: %v", err)
	}
	return out
}

func writeMinimalReticulumConfigRNID(t *testing.T, configDir string) {
	t.Helper()
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir configdir: %v", err)
	}
	// Keep it offline and avoid shared instance RPC to reduce sandbox friction.
	cfg := strings.Join([]string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = No",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(configDir, "config"), []byte(cfg), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func runRNID(t *testing.T, ctx context.Context, bin, configDir, workDir string, args ...string) (stdout string, exitCode int) {
	t.Helper()
	c := exec.CommandContext(ctx, bin, args...)
	c.Dir = workDir
	home := filepath.Join(configDir, ".home")
	_ = os.MkdirAll(home, 0o755)
	c.Env = append(os.Environ(),
		"HOME="+home,
		"USERPROFILE="+home,
	)
	out, err := c.CombinedOutput()
	if err == nil {
		return string(out), 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return string(out), ee.ExitCode()
	}
	t.Fatalf("run rnid: %v\n%s", err, string(out))
	return "", -1
}

func runRNIDWithInput(t *testing.T, ctx context.Context, bin, configDir, workDir string, stdin io.Reader, args ...string) (stdout string, exitCode int) {
	t.Helper()
	c := exec.CommandContext(ctx, bin, args...)
	c.Dir = workDir
	c.Stdin = stdin
	home := filepath.Join(configDir, ".home")
	_ = os.MkdirAll(home, 0o755)
	c.Env = append(os.Environ(),
		"HOME="+home,
		"USERPROFILE="+home,
	)
	out, err := c.CombinedOutput()
	if err == nil {
		return string(out), 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return string(out), ee.ExitCode()
	}
	t.Fatalf("run rnid with input: %v\n%s", err, string(out))
	return "", -1
}

func skipIfReticulumUnavailable(t *testing.T, out string, exitCode int) {
	t.Helper()
	if exitCode == 101 || strings.Contains(out, "Could not start Reticulum") ||
		strings.Contains(out, "operation not permitted") ||
		strings.Contains(out, "No interfaces could process") {
		t.Skipf("environment does not allow Reticulum startup; skipping rnid integration test\n%s", out)
	}
}

func rememberPublicDestinationRNID(t *testing.T, configDir string, destinationHash []byte, publicKey []byte) {
	t.Helper()

	logLevel := 0
	cfg := configDir
	ret, err := rns.NewReticulum(&cfg, &logLevel, nil, nil, false, nil)
	if err != nil {
		t.Fatalf("start reticulum for known destination prep: %v", err)
	}
	if ret == nil {
		t.Fatalf("reticulum instance is nil")
	}
	if err := rns.IdentityRemember([]byte("pkt"), destinationHash, publicKey, nil); err != nil {
		t.Fatalf("remember public destination: %v", err)
	}
	if err := rns.IdentitySaveKnownDestinations(); err != nil {
		t.Fatalf("save known destinations: %v", err)
	}
}

func TestRNIDIntegration_EncryptDecrypt_SignValidate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	root := t.TempDir()
	bin := buildRNID(t, root)
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNID(t, cfg)

	work := filepath.Join(root, "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}

	identityPath := filepath.Join(work, "id")
	out, code := runRNID(t, ctx, bin, cfg, work, "--config", cfg, "--generate", identityPath)
	skipIfReticulumUnavailable(t, out, code)
	if code != 0 {
		t.Fatalf("generate exit=%d\n%s", code, out)
	}
	if _, err := os.Stat(identityPath); err != nil {
		t.Fatalf("expected identity file: %v", err)
	}

	plain := []byte("hello rnid integration\n")
	inPath := filepath.Join(work, "in.txt")
	if err := os.WriteFile(inPath, plain, 0o644); err != nil {
		t.Fatal(err)
	}

	encPath := filepath.Join(work, "in.txt."+encExt)
	out, code = runRNID(t, ctx, bin, cfg, work,
		"--config", cfg,
		"--identity", identityPath,
		"--encrypt", inPath,
		"--write", encPath,
	)
	skipIfReticulumUnavailable(t, out, code)
	if code != 0 {
		t.Fatalf("encrypt exit=%d\n%s", code, out)
	}

	decPath := filepath.Join(work, "out.txt")
	out, code = runRNID(t, ctx, bin, cfg, work,
		"--config", cfg,
		"--identity", identityPath,
		"--decrypt", encPath,
		"--write", decPath,
	)
	skipIfReticulumUnavailable(t, out, code)
	if code != 0 {
		t.Fatalf("decrypt exit=%d\n%s", code, out)
	}
	got, err := os.ReadFile(decPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("decrypt mismatch: got %q want %q", got, plain)
	}

	sigPath := filepath.Join(work, "in.txt."+sigExt)
	out, code = runRNID(t, ctx, bin, cfg, work,
		"--config", cfg,
		"--identity", identityPath,
		"--sign", inPath,
		"--write", sigPath,
	)
	skipIfReticulumUnavailable(t, out, code)
	if code != 0 {
		t.Fatalf("sign exit=%d\n%s", code, out)
	}

	// Validate with explicit read.
	out, code = runRNID(t, ctx, bin, cfg, work,
		"--config", cfg,
		"--identity", identityPath,
		"--validate", sigPath,
		"--read", inPath,
	)
	skipIfReticulumUnavailable(t, out, code)
	if code != 0 {
		t.Fatalf("validate(exit=%d) expected 0\n%s", code, out)
	}

	// Validate without --read (derive from .rsg).
	out, code = runRNID(t, ctx, bin, cfg, work,
		"--config", cfg,
		"--identity", identityPath,
		"--validate", sigPath,
	)
	skipIfReticulumUnavailable(t, out, code)
	if code != 0 {
		t.Fatalf("validate(derive) exit=%d expected 0\n%s", code, out)
	}

	// Tamper input and ensure invalid signature maps to exit 22 (Python parity).
	badPath := filepath.Join(work, "bad.txt")
	if err := os.WriteFile(badPath, []byte("tampered\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code = runRNID(t, ctx, bin, cfg, work,
		"--config", cfg,
		"--identity", identityPath,
		"--validate", sigPath,
		"--read", badPath,
	)
	skipIfReticulumUnavailable(t, out, code)
	if code != 22 {
		t.Fatalf("validate(tampered) exit=%d expected 22\n%s", code, out)
	}
}

func TestRNIDIntegration_Version(t *testing.T) {
	root := t.TempDir()
	bin := buildRNID(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, code := runRNID(t, ctx, bin, root, root, "--version")
	if code != 0 {
		t.Fatalf("version exit=%d\n%s", code, out)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), "rnid ") {
		t.Fatalf("unexpected version output: %q", out)
	}
}

func TestRNIDIntegration_NoIdentityExit2(t *testing.T) {
	root := t.TempDir()
	bin := buildRNID(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNID(t, cfg)

	out, code := runRNID(t, ctx, bin, cfg, root, "--config", cfg)
	if code != 2 {
		t.Fatalf("expected exit 2, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "No identity provided, cannot continue") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestRNIDIntegration_GenerateExistingFileWithoutForceExit3(t *testing.T) {
	root := t.TempDir()
	bin := buildRNID(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNID(t, cfg)

	identityPath := filepath.Join(root, "id")
	if err := os.WriteFile(identityPath, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}

	out, code := runRNID(t, ctx, bin, cfg, root, "--config", cfg, "--generate", identityPath)
	skipIfReticulumUnavailable(t, out, code)
	if code != 3 {
		t.Fatalf("expected exit 3, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "already exists") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestRNIDIntegration_ExportImportAndPrintIdentity(t *testing.T) {
	root := t.TempDir()
	bin := buildRNID(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNID(t, cfg)

	identityPath := filepath.Join(root, "id")
	out, code := runRNID(t, ctx, bin, cfg, root, "--config", cfg, "--generate", identityPath)
	skipIfReticulumUnavailable(t, out, code)
	if code != 0 {
		t.Fatalf("generate exit=%d\n%s", code, out)
	}

	out, code = runRNID(t, ctx, bin, cfg, root, "--config", cfg, "--identity", identityPath, "--export")
	skipIfReticulumUnavailable(t, out, code)
	if code != 0 {
		t.Fatalf("export exit=%d\n%s", code, out)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	var exported string
	for _, line := range lines {
		if strings.Contains(line, "Exported Identity : ") {
			exported = strings.TrimSpace(strings.SplitN(line, "Exported Identity : ", 2)[1])
			break
		}
	}
	if exported == "" {
		t.Fatalf("could not find exported identity in output:\n%s", out)
	}

	importedPath := filepath.Join(root, "imported.id")
	out, code = runRNID(t, ctx, bin, cfg, root,
		"--import", exported,
		"--write", importedPath,
		"--print-private",
	)
	if code != 0 {
		t.Fatalf("import exit=%d\n%s", code, out)
	}
	if _, err := os.Stat(importedPath); err != nil {
		t.Fatalf("expected imported identity file: %v", err)
	}
	if !strings.Contains(out, "Identity imported") {
		t.Fatalf("unexpected import output:\n%s", out)
	}

	out, code = runRNID(t, ctx, bin, cfg, root, "--config", cfg, "--identity", importedPath, "--print-identity")
	skipIfReticulumUnavailable(t, out, code)
	if code != 0 {
		t.Fatalf("print-identity exit=%d\n%s", code, out)
	}
	if !strings.Contains(out, "Public Key") || !strings.Contains(out, "Private Key : Hidden") {
		t.Fatalf("unexpected print-identity output:\n%s", out)
	}
}

func TestRNIDIntegration_ConflictingFileOpsExit1(t *testing.T) {
	root := t.TempDir()
	bin := buildRNID(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, code := runRNID(t, ctx, bin, root, root, "--identity", "deadbeef", "--encrypt", "a", "--sign", "b")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "only one of --encrypt, --decrypt, --sign, --validate can be used at a time") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestRNIDIntegration_ForceOverwriteGenerate(t *testing.T) {
	root := t.TempDir()
	bin := buildRNID(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNID(t, cfg)

	identityPath := filepath.Join(root, "id")
	if err := os.WriteFile(identityPath, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}

	out, code := runRNID(t, ctx, bin, cfg, root, "--config", cfg, "--generate", identityPath, "--force")
	skipIfReticulumUnavailable(t, out, code)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d\n%s", code, out)
	}
	info, err := os.Stat(identityPath)
	if err != nil {
		t.Fatalf("stat identity: %v", err)
	}
	if info.Size() == int64(len("existing")) {
		t.Fatalf("expected generated identity to overwrite existing file")
	}
}

func TestRNIDIntegration_Base64AndBase32ExportImport(t *testing.T) {
	root := t.TempDir()
	bin := buildRNID(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNID(t, cfg)

	identityPath := filepath.Join(root, "id")
	out, code := runRNID(t, ctx, bin, cfg, root, "--config", cfg, "--generate", identityPath)
	skipIfReticulumUnavailable(t, out, code)
	if code != 0 {
		t.Fatalf("generate exit=%d\n%s", code, out)
	}

	for _, tc := range []struct {
		name    string
		flag    string
		prefix  string
		outPath string
	}{
		{name: "base64", flag: "--base64", prefix: "Exported Identity : ", outPath: filepath.Join(root, "imported.b64.id")},
		{name: "base32", flag: "--base32", prefix: "Exported Identity : ", outPath: filepath.Join(root, "imported.b32.id")},
	} {
		out, code = runRNID(t, ctx, bin, cfg, root, "--config", cfg, "--identity", identityPath, "--export", tc.flag)
		skipIfReticulumUnavailable(t, out, code)
		if code != 0 {
			t.Fatalf("%s export exit=%d\n%s", tc.name, code, out)
		}
		var exported string
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			if strings.Contains(line, tc.prefix) {
				exported = strings.TrimSpace(strings.SplitN(line, tc.prefix, 2)[1])
				break
			}
		}
		if exported == "" {
			t.Fatalf("%s exported identity not found in output:\n%s", tc.name, out)
		}

		out, code = runRNID(t, ctx, bin, cfg, root, "--import", exported, tc.flag, "--write", tc.outPath)
		if code != 0 {
			t.Fatalf("%s import exit=%d\n%s", tc.name, code, out)
		}
		if _, err := os.Stat(tc.outPath); err != nil {
			t.Fatalf("%s imported file missing: %v", tc.name, err)
		}
	}
}

func TestRNIDIntegration_PrintPrivate(t *testing.T) {
	root := t.TempDir()
	bin := buildRNID(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNID(t, cfg)

	identityPath := filepath.Join(root, "id")
	out, code := runRNID(t, ctx, bin, cfg, root, "--config", cfg, "--generate", identityPath)
	skipIfReticulumUnavailable(t, out, code)
	if code != 0 {
		t.Fatalf("generate exit=%d\n%s", code, out)
	}

	out, code = runRNID(t, ctx, bin, cfg, root, "--config", cfg, "--identity", identityPath, "--print-identity", "--print-private")
	skipIfReticulumUnavailable(t, out, code)
	if code != 0 {
		t.Fatalf("print-private exit=%d\n%s", code, out)
	}
	if !strings.Contains(out, "Public Key") || !strings.Contains(out, "Private Key : ") || strings.Contains(out, "Private Key : Hidden") {
		t.Fatalf("unexpected print-private output:\n%s", out)
	}
}

func TestRNIDIntegration_SignStdoutAndValidate(t *testing.T) {
	root := t.TempDir()
	bin := buildRNID(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNID(t, cfg)

	identityPath := filepath.Join(root, "id")
	out, code := runRNID(t, ctx, bin, cfg, root, "--config", cfg, "--generate", identityPath)
	skipIfReticulumUnavailable(t, out, code)
	if code != 0 {
		t.Fatalf("generate exit=%d\n%s", code, out)
	}

	plain := []byte("stdin stdout rnid\n")
	inPath := filepath.Join(root, "in.txt")
	if err := os.WriteFile(inPath, plain, 0o644); err != nil {
		t.Fatal(err)
	}

	sigOut, code := runRNID(t, ctx, bin, cfg, root,
		"--config", cfg,
		"--identity", identityPath,
		"--sign", inPath,
		"--stdout",
	)
	skipIfReticulumUnavailable(t, sigOut, code)
	if code != 0 {
		t.Fatalf("sign stdout exit=%d\n%s", code, sigOut)
	}

	sigPath := filepath.Join(root, "sig.rsg")
	if err := os.WriteFile(sigPath, []byte(sigOut), 0o644); err != nil {
		t.Fatal(err)
	}

	out, code = runRNID(t, ctx, bin, cfg, root,
		"--config", cfg,
		"--identity", identityPath,
		"--validate", sigPath,
		"--read", inPath,
	)
	skipIfReticulumUnavailable(t, out, code)
	if code != 0 {
		t.Fatalf("validate stdout signature exit=%d\n%s", code, out)
	}
}

func TestRNIDIntegration_EncryptDecryptViaStdinStdout(t *testing.T) {
	root := t.TempDir()
	bin := buildRNID(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNID(t, cfg)

	identityPath := filepath.Join(root, "id")
	out, code := runRNID(t, ctx, bin, cfg, root, "--config", cfg, "--generate", identityPath)
	skipIfReticulumUnavailable(t, out, code)
	if code != 0 {
		t.Fatalf("generate exit=%d\n%s", code, out)
	}

	plain := []byte("streamed rnid payload\n")
	ciphertext, code := runRNIDWithInput(t, ctx, bin, cfg, root, bytes.NewReader(plain),
		"--config", cfg,
		"--identity", identityPath,
		"--encrypt", "ignored",
		"--stdin",
		"--stdout",
	)
	skipIfReticulumUnavailable(t, ciphertext, code)
	if code != 0 {
		t.Fatalf("encrypt stdin/stdout exit=%d\n%s", code, ciphertext)
	}

	decrypted, code := runRNIDWithInput(t, ctx, bin, cfg, root, strings.NewReader(ciphertext),
		"--config", cfg,
		"--identity", identityPath,
		"--decrypt", "ignored",
		"--stdin",
		"--stdout",
	)
	skipIfReticulumUnavailable(t, decrypted, code)
	if code != 0 {
		t.Fatalf("decrypt stdin/stdout exit=%d\n%s", code, decrypted)
	}
	if decrypted != string(plain) {
		t.Fatalf("decrypt stdin/stdout mismatch: got %q want %q", decrypted, plain)
	}
}

func TestRNIDIntegration_ValidateMissingSignatureExit10(t *testing.T) {
	root := t.TempDir()
	bin := buildRNID(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNID(t, cfg)

	identityPath := filepath.Join(root, "id")
	out, code := runRNID(t, ctx, bin, cfg, root, "--config", cfg, "--generate", identityPath)
	skipIfReticulumUnavailable(t, out, code)
	if code != 0 {
		t.Fatalf("generate exit=%d\n%s", code, out)
	}

	inPath := filepath.Join(root, "in.txt")
	if err := os.WriteFile(inPath, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, code = runRNID(t, ctx, bin, cfg, root,
		"--config", cfg,
		"--identity", identityPath,
		"--validate", filepath.Join(root, "missing.rsg"),
		"--read", inPath,
	)
	skipIfReticulumUnavailable(t, out, code)
	if code != 10 {
		t.Fatalf("expected exit 10, got %d\n%s", code, out)
	}
}

func TestRNIDIntegration_InvalidHexIdentityExit7(t *testing.T) {
	root := t.TempDir()
	bin := buildRNID(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNID(t, cfg)

	out, code := runRNID(t, ctx, bin, cfg, root,
		"--config", cfg,
		"--identity", strings.Repeat("z", 32),
		"--print-identity",
	)
	skipIfReticulumUnavailable(t, out, code)
	if code != 7 {
		t.Fatalf("expected exit 7, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "Invalid hexadecimal hash provided") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestRNIDIntegration_UnknownHashWithoutRequestExit5(t *testing.T) {
	root := t.TempDir()
	bin := buildRNID(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNID(t, cfg)

	out, code := runRNID(t, ctx, bin, cfg, root,
		"--config", cfg,
		"--identity", strings.Repeat("0", 32),
		"--print-identity",
	)
	skipIfReticulumUnavailable(t, out, code)
	if code != 5 {
		t.Fatalf("expected exit 5, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "Could not recall Identity") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestRNIDIntegration_UnknownHashWithRequestTimesOutExit6(t *testing.T) {
	root := t.TempDir()
	bin := buildRNID(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNID(t, cfg)

	out, code := runRNID(t, ctx, bin, cfg, root,
		"--config", cfg,
		"--identity", strings.Repeat("0", 32),
		"--request",
		"--timeout", "0.2",
		"--print-identity",
	)
	skipIfReticulumUnavailable(t, out, code)
	if code != 6 {
		t.Fatalf("expected exit 6, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "Identity request timed out") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestRNIDIntegration_MissingIdentityFileExit8(t *testing.T) {
	root := t.TempDir()
	bin := buildRNID(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNID(t, cfg)

	out, code := runRNID(t, ctx, bin, cfg, root,
		"--config", cfg,
		"--identity", filepath.Join(root, "missing.id"),
		"--print-identity",
	)
	skipIfReticulumUnavailable(t, out, code)
	if code != 8 {
		t.Fatalf("expected exit 8, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "Specified Identity file not found") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestRNIDIntegration_InvalidIdentityFileExit9(t *testing.T) {
	root := t.TempDir()
	bin := buildRNID(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNID(t, cfg)

	identityPath := filepath.Join(root, "bad.id")
	if err := os.WriteFile(identityPath, []byte("not-an-identity"), 0o600); err != nil {
		t.Fatal(err)
	}

	out, code := runRNID(t, ctx, bin, cfg, root,
		"--config", cfg,
		"--identity", identityPath,
		"--print-identity",
	)
	skipIfReticulumUnavailable(t, out, code)
	if code != 9 {
		t.Fatalf("expected exit 9, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "Could not decode Identity from specified file") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestRNIDIntegration_HashSuccess(t *testing.T) {
	root := t.TempDir()
	bin := buildRNID(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNID(t, cfg)

	identityPath := filepath.Join(root, "id")
	out, code := runRNID(t, ctx, bin, cfg, root, "--config", cfg, "--generate", identityPath)
	skipIfReticulumUnavailable(t, out, code)
	if code != 0 {
		t.Fatalf("generate exit=%d\n%s", code, out)
	}

	out, code = runRNID(t, ctx, bin, cfg, root,
		"--config", cfg,
		"--identity", identityPath,
		"--hash", "app.aspect",
	)
	skipIfReticulumUnavailable(t, out, code)
	if code != 0 {
		t.Fatalf("hash exit=%d\n%s", code, out)
	}
	if !strings.Contains(out, "The app.aspect destination for this Identity is") ||
		!strings.Contains(out, "The full destination specifier is") {
		t.Fatalf("unexpected hash output:\n%s", out)
	}
}

func TestRNIDIntegration_AnnounceInvalidAspectsExit32(t *testing.T) {
	root := t.TempDir()
	bin := buildRNID(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNID(t, cfg)

	identityPath := filepath.Join(root, "id")
	out, code := runRNID(t, ctx, bin, cfg, root, "--config", cfg, "--generate", identityPath)
	skipIfReticulumUnavailable(t, out, code)
	if code != 0 {
		t.Fatalf("generate exit=%d\n%s", code, out)
	}

	out, code = runRNID(t, ctx, bin, cfg, root,
		"--config", cfg,
		"--identity", identityPath,
		"--announce", "invalid",
	)
	skipIfReticulumUnavailable(t, out, code)
	if code != 32 {
		t.Fatalf("expected exit 32, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "Invalid destination aspects specified") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestRNIDIntegration_ImportInvalidDataExit41(t *testing.T) {
	root := t.TempDir()
	bin := buildRNID(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, code := runRNID(t, ctx, bin, root, root, "--import", "not-hex-data")
	if code != 41 {
		t.Fatalf("expected exit 41, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "Invalid identity data specified for import") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestRNIDIntegration_ImportExistingFileWithoutForceExit43(t *testing.T) {
	root := t.TempDir()
	bin := buildRNID(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNID(t, cfg)

	identityPath := filepath.Join(root, "id")
	out, code := runRNID(t, ctx, bin, cfg, root, "--config", cfg, "--generate", identityPath)
	skipIfReticulumUnavailable(t, out, code)
	if code != 0 {
		t.Fatalf("generate exit=%d\n%s", code, out)
	}

	out, code = runRNID(t, ctx, bin, cfg, root, "--config", cfg, "--identity", identityPath, "--export")
	skipIfReticulumUnavailable(t, out, code)
	if code != 0 {
		t.Fatalf("export exit=%d\n%s", code, out)
	}
	var exported string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if strings.Contains(line, "Exported Identity : ") {
			exported = strings.TrimSpace(strings.SplitN(line, "Exported Identity : ", 2)[1])
			break
		}
	}
	if exported == "" {
		t.Fatalf("could not find exported identity in output:\n%s", out)
	}

	importPath := filepath.Join(root, "existing.id")
	if err := os.WriteFile(importPath, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}

	out, code = runRNID(t, ctx, bin, cfg, root, "--import", exported, "--write", importPath)
	if code != 43 {
		t.Fatalf("expected exit 43, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "already exists, not overwriting") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestRNIDIntegration_AnnounceSuccess(t *testing.T) {
	root := t.TempDir()
	bin := buildRNID(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNID(t, cfg)

	identityPath := filepath.Join(root, "id")
	out, code := runRNID(t, ctx, bin, cfg, root, "--config", cfg, "--generate", identityPath)
	skipIfReticulumUnavailable(t, out, code)
	if code != 0 {
		t.Fatalf("generate exit=%d\n%s", code, out)
	}

	out, code = runRNID(t, ctx, bin, cfg, root,
		"--config", cfg,
		"--identity", identityPath,
		"--announce", "app.aspect",
	)
	skipIfReticulumUnavailable(t, out, code)
	if code != 0 {
		t.Fatalf("announce exit=%d\n%s", code, out)
	}
	if !strings.Contains(out, "Created destination") || !strings.Contains(out, "Announcing destination") {
		t.Fatalf("unexpected announce output:\n%s", out)
	}
}

func TestRNIDIntegration_AnnouncePublicOnlyIdentityExit33(t *testing.T) {
	root := t.TempDir()
	bin := buildRNID(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNID(t, cfg)

	id, err := rns.NewIdentity()
	if err != nil {
		t.Fatalf("new identity: %v", err)
	}
	dst, err := rns.NewDestination(id, rns.DestinationIN, rns.DestinationSINGLE, "app", "aspect")
	if err != nil {
		t.Fatalf("new destination: %v", err)
	}
	rememberPublicDestinationRNID(t, cfg, dst.Hash(), id.GetPublicKey())

	out, code := runRNID(t, ctx, bin, cfg, root,
		"--config", cfg,
		"--identity", hex.EncodeToString(dst.Hash()),
		"--announce", "app.aspect",
	)
	skipIfReticulumUnavailable(t, out, code)
	if code != 33 {
		t.Fatalf("expected exit 33, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "Cannot announce this destination, since the private key is not held") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}
