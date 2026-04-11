//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/svanichkin/go-reticulum/internal/cmdtest"
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
	return cmdtest.Build(t, binDir, "rnid", "./cmd/rnid")
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

	helperBin := cmdtest.Build(t, configDir, "go_known_dest_helper", "./tests/support/tools/go_known_dest_helper")
	cmd := exec.Command(helperBin, configDir, hex.EncodeToString(destinationHash), hex.EncodeToString(publicKey))
	cmd.Dir = repoRootRNID(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("remember public destination: %v\n%s", err, string(out))
	}
}

func writeReticulumConfigSharedRNID(t *testing.T, configDir string, lines []string) {
	t.Helper()
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir configdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config"), []byte(strings.Join(lines, "\n")), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func freeTCPPortRNID(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen ephemeral tcp port: %v", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func writeTwoNodeTemplateConfigRNID(t *testing.T, dstDir, templatePath string, sharedPort, controlPort, listenPort, forwardPort int) {
	t.Helper()
	bodyBytes, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("read template config: %v", err)
	}
	body := string(bodyBytes)
	replacements := map[string]string{
		"shared_instance_port = 37430":  "shared_instance_port = " + strconv.Itoa(sharedPort),
		"instance_control_port = 37431": "instance_control_port = " + strconv.Itoa(controlPort),
		"shared_instance_port = 37432":  "shared_instance_port = " + strconv.Itoa(sharedPort),
		"instance_control_port = 37433": "instance_control_port = " + strconv.Itoa(controlPort),
		"listen_port = 50000":           "listen_port = " + strconv.Itoa(listenPort),
		"forward_port = 50001":          "forward_port = " + strconv.Itoa(forwardPort),
		"listen_port = 50001":           "listen_port = " + strconv.Itoa(listenPort),
		"forward_port = 50000":          "forward_port = " + strconv.Itoa(forwardPort),
	}
	for old, newVal := range replacements {
		body = strings.ReplaceAll(body, old, newVal)
	}
	cmdtest.WriteReticulumConfig(t, dstDir, body)
}

func startRNSDServiceRNID(t *testing.T, ctx context.Context, bin, cfg, workDir string) (*exec.Cmd, *bytes.Buffer) {
	t.Helper()
	c := exec.CommandContext(ctx, bin, "--config", cfg, "--service")
	c.Dir = workDir
	home := filepath.Join(cfg, ".home")
	_ = os.MkdirAll(home, 0o755)
	c.Env = append(os.Environ(),
		"HOME="+home,
		"USERPROFILE="+home,
	)
	var out bytes.Buffer
	c.Stdout = &out
	c.Stderr = &out
	if err := c.Start(); err != nil {
		t.Fatalf("start rnsd: %v", err)
	}
	t.Cleanup(func() {
		if c.Process != nil {
			_ = c.Process.Signal(syscall.SIGTERM)
		}
		_ = c.Wait()
	})
	return c, &out
}

func waitForRNStatusSuccessRNID(t *testing.T, rnstatusBin, cfg, workDir string, timeout time.Duration) string {
	t.Helper()
	if timeout < 90*time.Second {
		timeout = 90 * time.Second
	}
	deadline := time.Now().Add(timeout)
	var last string
	consecutive := 0
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		res := cmdtest.Run(t, ctx, rnstatusBin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: workDir}, "--config", cfg, "--json")
		cancel()
		if res.ExitCode == 0 {
			last = res.Output
			consecutive++
			if consecutive >= 2 {
				time.Sleep(500 * time.Millisecond)
				return last
			}
			time.Sleep(250 * time.Millisecond)
			continue
		}
		consecutive = 0
		if !strings.Contains(res.Output, "no shared RNS instance available") &&
			!strings.Contains(res.Output, "could not get RNS status") &&
			!strings.Contains(res.Output, "operation not permitted") {
			t.Fatalf("unexpected rnstatus failure while waiting for rnsd:\n%s", res.Output)
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("rnstatus did not succeed within %s", timeout)
	return ""
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

func TestRNIDIntegration_SharedInstanceHashSuccess(t *testing.T) {
	root := t.TempDir()
	rnidBin := buildRNID(t, root)
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigSharedRNID(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = rnid-shared-hash",
		"",
	})

	identityPath := filepath.Join(root, "id")
	out, code := runRNID(t, ctx, rnidBin, cfg, root, "--config", cfg, "--generate", identityPath)
	skipIfReticulumUnavailable(t, out, code)
	if code != 0 {
		t.Fatalf("generate exit=%d\n%s", code, out)
	}

	_, serviceOut := startRNSDServiceRNID(t, ctx, rnsdBin, cfg, root)
	_ = waitForRNStatusSuccessRNID(t, rnstatusBin, cfg, root, 10*time.Second)

	out, code = runRNID(t, ctx, rnidBin, cfg, root,
		"--config", cfg,
		"--identity", identityPath,
		"--hash", "app.aspect",
	)
	skipIfReticulumUnavailable(t, serviceOut.String()+out, code)
	if code != 0 {
		t.Fatalf("hash exit=%d\n%s", code, out)
	}
	if !strings.Contains(out, "The app.aspect destination for this Identity is") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestRNIDIntegration_SharedInstanceTCPAnnounceSuccess(t *testing.T) {
	root := t.TempDir()
	rnidBin := buildRNID(t, root)
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	cfg := filepath.Join(root, "cfg")
	packetPort := freeTCPPortRNID(t)
	controlPort := freeTCPPortRNID(t)
	writeReticulumConfigSharedRNID(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"shared_instance_type = tcp",
		"shared_instance_port = " + strconv.Itoa(packetPort),
		"instance_control_port = " + strconv.Itoa(controlPort),
		"",
	})

	identityPath := filepath.Join(root, "id")
	out, code := runRNID(t, ctx, rnidBin, cfg, root, "--config", cfg, "--generate", identityPath)
	skipIfReticulumUnavailable(t, out, code)
	if code != 0 {
		t.Fatalf("generate exit=%d\n%s", code, out)
	}

	_, serviceOut := startRNSDServiceRNID(t, ctx, rnsdBin, cfg, root)
	_ = waitForRNStatusSuccessRNID(t, rnstatusBin, cfg, root, 10*time.Second)

	out, code = runRNID(t, ctx, rnidBin, cfg, root,
		"--config", cfg,
		"--identity", identityPath,
		"--announce", "app.aspect",
	)
	skipIfReticulumUnavailable(t, serviceOut.String()+out, code)
	if code != 0 {
		t.Fatalf("announce exit=%d\n%s", code, out)
	}
	if !strings.Contains(out, "Created destination") || !strings.Contains(out, "Announcing destination") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestRNIDIntegration_SharedInstancePublicOnlyAnnounceExit33(t *testing.T) {
	root := t.TempDir()
	rnidBin := buildRNID(t, root)
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigSharedRNID(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = rnid-shared-public-only",
		"",
	})

	id, err := rns.NewIdentity()
	if err != nil {
		t.Fatalf("new identity: %v", err)
	}
	dst, err := rns.NewDestination(id, rns.DestinationIN, rns.DestinationSINGLE, "app", "aspect")
	if err != nil {
		t.Fatalf("new destination: %v", err)
	}
	rememberPublicDestinationRNID(t, cfg, dst.Hash(), id.GetPublicKey())

	_, serviceOut := startRNSDServiceRNID(t, ctx, rnsdBin, cfg, root)
	_ = waitForRNStatusSuccessRNID(t, rnstatusBin, cfg, root, 10*time.Second)

	out, code := runRNID(t, ctx, rnidBin, cfg, root,
		"--config", cfg,
		"--identity", hex.EncodeToString(dst.Hash()),
		"--announce", "app.aspect",
	)
	skipIfReticulumUnavailable(t, serviceOut.String()+out, code)
	if code != 33 {
		t.Fatalf("expected exit 33, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "Cannot announce this destination, since the private key is not held") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestRNIDIntegration_TwoNodeRequestSuccess(t *testing.T) {
	cmdtest.AcquireLock(t, "integration-two-node-shared", 5*time.Minute)
	root := t.TempDir()
	rnidBin := buildRNID(t, root)
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	basePort := freeTCPPortRNID(t)
	sharedA := freeTCPPortRNID(t)
	controlA := freeTCPPortRNID(t)
	sharedB := freeTCPPortRNID(t)
	controlB := freeTCPPortRNID(t)

	nodeADir := filepath.Join(root, "node-a")
	nodeBDir := filepath.Join(root, "node-b")
	writeTwoNodeTemplateConfigRNID(t, nodeADir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_a/config"),
		sharedA, controlA, basePort, basePort+1)
	writeTwoNodeTemplateConfigRNID(t, nodeBDir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_b/config"),
		sharedB, controlB, basePort+1, basePort)

	_, outA := startRNSDServiceRNID(t, ctx, rnsdBin, nodeADir, root)
	_, outB := startRNSDServiceRNID(t, ctx, rnsdBin, nodeBDir, root)
	_ = waitForRNStatusSuccessRNID(t, rnstatusBin, nodeADir, root, 20*time.Second)
	_ = waitForRNStatusSuccessRNID(t, rnstatusBin, nodeBDir, root, 20*time.Second)

	identityPath := filepath.Join(root, "node-b.id")
	out, code := runRNID(t, ctx, rnidBin, nodeBDir, root, "--config", nodeBDir, "--generate", identityPath)
	skipIfReticulumUnavailable(t, outB.String()+out, code)
	if code != 0 {
		t.Fatalf("generate exit=%d\n%s", code, out)
	}

	hashOut, hashCode := runRNID(t, ctx, rnidBin, nodeBDir, root,
		"--config", nodeBDir,
		"--identity", identityPath,
		"--hash", "app.aspect",
	)
	skipIfReticulumUnavailable(t, outB.String()+hashOut, hashCode)
	if hashCode != 0 {
		t.Fatalf("hash exit=%d\n%s", hashCode, hashOut)
	}
	var destHash string
	for _, line := range strings.Split(hashOut, "\n") {
		if !strings.Contains(line, "destination for this Identity is") {
			continue
		}
		start := strings.Index(line, "<")
		end := strings.Index(line, ">")
		if start >= 0 && end > start {
			destHash = line[start+1 : end]
			break
		}
	}
	if destHash == "" {
		t.Fatalf("could not extract destination hash from output:\n%s", hashOut)
	}

	announceOut, announceCode := runRNID(t, ctx, rnidBin, nodeBDir, root,
		"--config", nodeBDir,
		"--identity", identityPath,
		"--announce", "app.aspect",
	)
	skipIfReticulumUnavailable(t, outB.String()+announceOut, announceCode)
	if announceCode != 0 {
		t.Fatalf("announce exit=%d\n%s", announceCode, announceOut)
	}

	requestOut, requestCode := runRNID(t, ctx, rnidBin, nodeADir, root,
		"--config", nodeADir,
		"--identity", destHash,
		"--request",
		"--timeout", "15",
		"--print-identity",
	)
	skipIfReticulumUnavailable(t, outA.String()+outB.String()+requestOut, requestCode)
	if requestCode != 0 {
		t.Fatalf("request exit=%d\nnodeA:\n%s\nnodeB:\n%s\nrequest:\n%s", requestCode, outA.String(), outB.String(), requestOut)
	}
	if !strings.Contains(requestOut, "Identity     :") {
		t.Fatalf("expected printed identity after successful request, got:\n%s", requestOut)
	}
}

func TestRNIDIntegration_TwoNodeAnnounceIsReceived(t *testing.T) {
	cmdtest.AcquireLock(t, "integration-two-node-shared", 5*time.Minute)
	root := t.TempDir()
	rnidBin := buildRNID(t, root)
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	basePort := freeTCPPortRNID(t)
	sharedA := freeTCPPortRNID(t)
	controlA := freeTCPPortRNID(t)
	sharedB := freeTCPPortRNID(t)
	controlB := freeTCPPortRNID(t)

	nodeADir := filepath.Join(root, "node-a")
	nodeBDir := filepath.Join(root, "node-b")
	writeTwoNodeTemplateConfigRNID(t, nodeADir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_a/config"),
		sharedA, controlA, basePort, basePort+1)
	writeTwoNodeTemplateConfigRNID(t, nodeBDir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_b/config"),
		sharedB, controlB, basePort+1, basePort)

	_, outA := startRNSDServiceRNID(t, ctx, rnsdBin, nodeADir, root)
	_, outB := startRNSDServiceRNID(t, ctx, rnsdBin, nodeBDir, root)
	_ = waitForRNStatusSuccessRNID(t, rnstatusBin, nodeADir, root, 20*time.Second)
	_ = waitForRNStatusSuccessRNID(t, rnstatusBin, nodeBDir, root, 20*time.Second)

	identityPath := filepath.Join(root, "node-b.id")
	out, code := runRNID(t, ctx, rnidBin, nodeBDir, root, "--config", nodeBDir, "--generate", identityPath)
	skipIfReticulumUnavailable(t, outB.String()+out, code)
	if code != 0 {
		t.Fatalf("generate exit=%d\n%s", code, out)
	}

	hashOut, hashCode := runRNID(t, ctx, rnidBin, nodeBDir, root,
		"--config", nodeBDir,
		"--identity", identityPath,
		"--hash", "app.aspect",
	)
	skipIfReticulumUnavailable(t, outB.String()+hashOut, hashCode)
	if hashCode != 0 {
		t.Fatalf("hash exit=%d\n%s", hashCode, hashOut)
	}
	var destHash string
	for _, line := range strings.Split(hashOut, "\n") {
		if !strings.Contains(line, "destination for this Identity is") {
			continue
		}
		start := strings.Index(line, "<")
		end := strings.Index(line, ">")
		if start >= 0 && end > start {
			destHash = line[start+1 : end]
			break
		}
	}
	if destHash == "" {
		t.Fatalf("could not extract destination hash from output:\n%s", hashOut)
	}

	announceOut, announceCode := runRNID(t, ctx, rnidBin, nodeBDir, root,
		"--config", nodeBDir,
		"--identity", identityPath,
		"--announce", "app.aspect",
	)
	skipIfReticulumUnavailable(t, outB.String()+announceOut, announceCode)
	if announceCode != 0 {
		t.Fatalf("announce exit=%d\n%s", announceCode, announceOut)
	}

	deadline := time.Now().Add(15 * time.Second)
	var recallOut string
	var recallCode int
	for time.Now().Before(deadline) {
		recallOut, recallCode = runRNID(t, ctx, rnidBin, nodeADir, root,
			"--config", nodeADir,
			"--identity", destHash,
			"--print-identity",
		)
		skipIfReticulumUnavailable(t, outA.String()+outB.String()+recallOut, recallCode)
		if recallCode == 0 && strings.Contains(recallOut, "Identity     :") {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}

	t.Fatalf("announce was not received without --request\nnodeA:\n%s\nnodeB:\n%s\nlast recall exit=%d\n%s",
		outA.String(), outB.String(), recallCode, recallOut)
}

func TestRNIDIntegration_TwoNodeMultiAspectAnnounce(t *testing.T) {
	root := t.TempDir()
	rnidBin := buildRNID(t, root)
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	basePort := freeTCPPortRNID(t)
	sharedA := freeTCPPortRNID(t)
	controlA := freeTCPPortRNID(t)
	sharedB := freeTCPPortRNID(t)
	controlB := freeTCPPortRNID(t)

	nodeADir := filepath.Join(root, "node-a")
	nodeBDir := filepath.Join(root, "node-b")
	writeTwoNodeTemplateConfigRNID(t, nodeADir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_a/config"),
		sharedA, controlA, basePort, basePort+1)
	writeTwoNodeTemplateConfigRNID(t, nodeBDir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_b/config"),
		sharedB, controlB, basePort+1, basePort)

	_, outA := startRNSDServiceRNID(t, ctx, rnsdBin, nodeADir, root)
	_, outB := startRNSDServiceRNID(t, ctx, rnsdBin, nodeBDir, root)
	_ = waitForRNStatusSuccessRNID(t, rnstatusBin, nodeADir, root, 10*time.Second)
	_ = waitForRNStatusSuccessRNID(t, rnstatusBin, nodeBDir, root, 10*time.Second)

	identityPath := filepath.Join(root, "node-b.id")
	out, code := runRNID(t, ctx, rnidBin, nodeBDir, root, "--config", nodeBDir, "--generate", identityPath)
	skipIfReticulumUnavailable(t, outB.String()+out, code)
	if code != 0 {
		t.Fatalf("generate exit=%d\n%s", code, out)
	}

	hashForAspect := func(aspect string) string {
		t.Helper()
		hashOut, hashCode := runRNID(t, ctx, rnidBin, nodeBDir, root,
			"--config", nodeBDir,
			"--identity", identityPath,
			"--hash", aspect,
		)
		skipIfReticulumUnavailable(t, outB.String()+hashOut, hashCode)
		if hashCode != 0 {
			t.Fatalf("hash(%s) exit=%d\n%s", aspect, hashCode, hashOut)
		}
		for _, line := range strings.Split(hashOut, "\n") {
			if !strings.Contains(line, "destination for this Identity is") {
				continue
			}
			start := strings.Index(line, "<")
			end := strings.Index(line, ">")
			if start >= 0 && end > start {
				return line[start+1 : end]
			}
		}
		t.Fatalf("could not extract destination hash for %s from output:\n%s", aspect, hashOut)
		return ""
	}

	hashOne := hashForAspect("app.one")
	hashTwo := hashForAspect("app.two")
	if hashOne == hashTwo {
		t.Fatalf("expected distinct hashes for different aspects, got %s", hashOne)
	}

	for _, aspect := range []string{"app.one", "app.two"} {
		announceOut, announceCode := runRNID(t, ctx, rnidBin, nodeBDir, root,
			"--config", nodeBDir,
			"--identity", identityPath,
			"--announce", aspect,
		)
		skipIfReticulumUnavailable(t, outB.String()+announceOut, announceCode)
		if announceCode != 0 {
			t.Fatalf("announce(%s) exit=%d\n%s", aspect, announceCode, announceOut)
		}
	}

	for _, tc := range []struct {
		name string
		hash string
	}{
		{name: "app.one", hash: hashOne},
		{name: "app.two", hash: hashTwo},
	} {
		deadline := time.Now().Add(15 * time.Second)
		var recallOut string
		var recallCode int
		for time.Now().Before(deadline) {
			recallOut, recallCode = runRNID(t, ctx, rnidBin, nodeADir, root,
				"--config", nodeADir,
				"--identity", tc.hash,
				"--print-identity",
			)
			skipIfReticulumUnavailable(t, outA.String()+outB.String()+recallOut, recallCode)
			if recallCode == 0 && strings.Contains(recallOut, "Identity     :") {
				break
			}
			time.Sleep(250 * time.Millisecond)
		}
		if recallCode != 0 || !strings.Contains(recallOut, "Identity     :") {
			t.Fatalf("aspect %s was not recalled after announce\nnodeA:\n%s\nnodeB:\n%s\nlast recall exit=%d\n%s",
				tc.name, outA.String(), outB.String(), recallCode, recallOut)
		}
	}
}

func TestRNIDIntegration_TwoNodeReannounceKeepsRecallAndPath(t *testing.T) {
	root := t.TempDir()
	rnidBin := buildRNID(t, root)
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	rnpathBin := cmdtest.Build(t, root, "rnpath", "./cmd/rnpath")
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	basePort := freeTCPPortRNID(t)
	sharedA := freeTCPPortRNID(t)
	controlA := freeTCPPortRNID(t)
	sharedB := freeTCPPortRNID(t)
	controlB := freeTCPPortRNID(t)

	nodeADir := filepath.Join(root, "node-a")
	nodeBDir := filepath.Join(root, "node-b")
	writeTwoNodeTemplateConfigRNID(t, nodeADir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_a/config"),
		sharedA, controlA, basePort, basePort+1)
	writeTwoNodeTemplateConfigRNID(t, nodeBDir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_b/config"),
		sharedB, controlB, basePort+1, basePort)

	_, outA := startRNSDServiceRNID(t, ctx, rnsdBin, nodeADir, root)
	_, outB := startRNSDServiceRNID(t, ctx, rnsdBin, nodeBDir, root)
	_ = waitForRNStatusSuccessRNID(t, rnstatusBin, nodeADir, root, 10*time.Second)
	_ = waitForRNStatusSuccessRNID(t, rnstatusBin, nodeBDir, root, 10*time.Second)

	identityPath := filepath.Join(root, "node-b.id")
	out, code := runRNID(t, ctx, rnidBin, nodeBDir, root, "--config", nodeBDir, "--generate", identityPath)
	skipIfReticulumUnavailable(t, outB.String()+out, code)
	if code != 0 {
		t.Fatalf("generate exit=%d\n%s", code, out)
	}

	hashOut, hashCode := runRNID(t, ctx, rnidBin, nodeBDir, root,
		"--config", nodeBDir,
		"--identity", identityPath,
		"--hash", "app.aspect",
	)
	skipIfReticulumUnavailable(t, outB.String()+hashOut, hashCode)
	if hashCode != 0 {
		t.Fatalf("hash exit=%d\n%s", hashCode, hashOut)
	}
	var destHash string
	for _, line := range strings.Split(hashOut, "\n") {
		if !strings.Contains(line, "destination for this Identity is") {
			continue
		}
		start := strings.Index(line, "<")
		end := strings.Index(line, ">")
		if start >= 0 && end > start {
			destHash = line[start+1 : end]
			break
		}
	}
	if destHash == "" {
		t.Fatalf("could not extract destination hash from output:\n%s", hashOut)
	}

	announce := func() {
		t.Helper()
		announceOut, announceCode := runRNID(t, ctx, rnidBin, nodeBDir, root,
			"--config", nodeBDir,
			"--identity", identityPath,
			"--announce", "app.aspect",
		)
		skipIfReticulumUnavailable(t, outB.String()+announceOut, announceCode)
		if announceCode != 0 {
			t.Fatalf("announce exit=%d\n%s", announceCode, announceOut)
		}
	}

	recallOnA := func() string {
		t.Helper()
		deadline := time.Now().Add(15 * time.Second)
		var recallOut string
		var recallCode int
		for time.Now().Before(deadline) {
			recallOut, recallCode = runRNID(t, ctx, rnidBin, nodeADir, root,
				"--config", nodeADir,
				"--identity", destHash,
				"--print-identity",
			)
			skipIfReticulumUnavailable(t, outA.String()+outB.String()+recallOut, recallCode)
			if recallCode == 0 && strings.Contains(recallOut, "Identity     :") {
				return recallOut
			}
			time.Sleep(250 * time.Millisecond)
		}
		t.Fatalf("destination was not recalled on node A\nnodeA:\n%s\nnodeB:\n%s\nlast recall exit=%d\n%s",
			outA.String(), outB.String(), recallCode, recallOut)
		return ""
	}

	announce()
	firstRecall := recallOnA()
	if !strings.Contains(firstRecall, "Identity     :") {
		t.Fatalf("unexpected first recall output:\n%s", firstRecall)
	}

	announce()
	secondRecall := recallOnA()
	if !strings.Contains(secondRecall, "Identity     :") {
		t.Fatalf("unexpected second recall output:\n%s", secondRecall)
	}

	pathCtx, pathCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer pathCancel()
	pathRes := cmdtest.Run(t, pathCtx, rnpathBin, cmdtest.RunOptions{ConfigDir: nodeADir, WorkDir: root},
		"--config", nodeADir, "-w", "15", destHash)
	if pathRes.ExitCode != 0 {
		t.Fatalf("expected path after re-announce, got %d\nnodeA:\n%s\nnodeB:\n%s\nrnpath:\n%s",
			pathRes.ExitCode, outA.String(), outB.String(), pathRes.Output)
	}
	if !strings.Contains(pathRes.Output, "Path found, destination") {
		t.Fatalf("unexpected rnpath output after re-announce:\n%s", pathRes.Output)
	}

	tableCtx, tableCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer tableCancel()
	tableRes := cmdtest.Run(t, tableCtx, rnpathBin, cmdtest.RunOptions{ConfigDir: nodeADir, WorkDir: root},
		"--config", nodeADir, "--table", "--json")
	if tableRes.ExitCode != 0 {
		t.Fatalf("rnpath --table --json failed after re-announce: %d\n%s", tableRes.ExitCode, tableRes.Output)
	}
	if strings.TrimSpace(tableRes.Output) == "[]" {
		t.Fatalf("expected non-empty path table after re-announce, got:\n%s", tableRes.Output)
	}
}

func TestRNIDIntegration_TwoNodeRepeatedRequestUsesRecall(t *testing.T) {
	root := t.TempDir()
	rnidBin := buildRNID(t, root)
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	basePort := freeTCPPortRNID(t)
	sharedA := freeTCPPortRNID(t)
	controlA := freeTCPPortRNID(t)
	sharedB := freeTCPPortRNID(t)
	controlB := freeTCPPortRNID(t)

	nodeADir := filepath.Join(root, "node-a")
	nodeBDir := filepath.Join(root, "node-b")
	writeTwoNodeTemplateConfigRNID(t, nodeADir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_a/config"),
		sharedA, controlA, basePort, basePort+1)
	writeTwoNodeTemplateConfigRNID(t, nodeBDir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_b/config"),
		sharedB, controlB, basePort+1, basePort)

	cmdA, outA := startRNSDServiceRNID(t, ctx, rnsdBin, nodeADir, root)
	_ = cmdA
	cmdB, outB := startRNSDServiceRNID(t, ctx, rnsdBin, nodeBDir, root)
	_ = waitForRNStatusSuccessRNID(t, rnstatusBin, nodeADir, root, 10*time.Second)
	_ = waitForRNStatusSuccessRNID(t, rnstatusBin, nodeBDir, root, 10*time.Second)

	identityPath := filepath.Join(root, "node-b.id")
	out, code := runRNID(t, ctx, rnidBin, nodeBDir, root, "--config", nodeBDir, "--generate", identityPath)
	skipIfReticulumUnavailable(t, outB.String()+out, code)
	if code != 0 {
		t.Fatalf("generate exit=%d\n%s", code, out)
	}

	hashOut, hashCode := runRNID(t, ctx, rnidBin, nodeBDir, root,
		"--config", nodeBDir,
		"--identity", identityPath,
		"--hash", "app.aspect",
	)
	skipIfReticulumUnavailable(t, outB.String()+hashOut, hashCode)
	if hashCode != 0 {
		t.Fatalf("hash exit=%d\n%s", hashCode, hashOut)
	}
	var destHash string
	for _, line := range strings.Split(hashOut, "\n") {
		if !strings.Contains(line, "destination for this Identity is") {
			continue
		}
		start := strings.Index(line, "<")
		end := strings.Index(line, ">")
		if start >= 0 && end > start {
			destHash = line[start+1 : end]
			break
		}
	}
	if destHash == "" {
		t.Fatalf("could not extract destination hash from output:\n%s", hashOut)
	}

	announceOut, announceCode := runRNID(t, ctx, rnidBin, nodeBDir, root,
		"--config", nodeBDir,
		"--identity", identityPath,
		"--announce", "app.aspect",
	)
	skipIfReticulumUnavailable(t, outB.String()+announceOut, announceCode)
	if announceCode != 0 {
		t.Fatalf("announce exit=%d\n%s", announceCode, announceOut)
	}

	firstRequestOut, firstRequestCode := runRNID(t, ctx, rnidBin, nodeADir, root,
		"--config", nodeADir,
		"--identity", destHash,
		"--request",
		"--timeout", "15",
		"--print-identity",
	)
	skipIfReticulumUnavailable(t, outA.String()+outB.String()+firstRequestOut, firstRequestCode)
	if firstRequestCode != 0 {
		t.Fatalf("first request exit=%d\n%s", firstRequestCode, firstRequestOut)
	}

	if cmdB.Process != nil {
		_ = cmdB.Process.Signal(syscall.SIGTERM)
		_ = cmdB.Wait()
	}

	secondRequestOut, secondRequestCode := runRNID(t, ctx, rnidBin, nodeADir, root,
		"--config", nodeADir,
		"--identity", destHash,
		"--request",
		"--timeout", "1",
		"--print-identity",
	)
	if secondRequestCode != 0 {
		t.Fatalf("expected repeated request to succeed from recall, got exit=%d\n%s", secondRequestCode, secondRequestOut)
	}
	if !strings.Contains(secondRequestOut, "Identity     :") {
		t.Fatalf("expected recalled identity output, got:\n%s", secondRequestOut)
	}
}

func TestRNIDIntegration_TwoNodeRequestUnknownHashTimesOut(t *testing.T) {
	cmdtest.AcquireLock(t, "integration-two-node-shared", 5*time.Minute)
	root := t.TempDir()
	rnidBin := buildRNID(t, root)
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	basePort := freeTCPPortRNID(t)
	sharedA := freeTCPPortRNID(t)
	controlA := freeTCPPortRNID(t)
	sharedB := freeTCPPortRNID(t)
	controlB := freeTCPPortRNID(t)

	nodeADir := filepath.Join(root, "node-a")
	nodeBDir := filepath.Join(root, "node-b")
	writeTwoNodeTemplateConfigRNID(t, nodeADir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_a/config"),
		sharedA, controlA, basePort, basePort+1)
	writeTwoNodeTemplateConfigRNID(t, nodeBDir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_b/config"),
		sharedB, controlB, basePort+1, basePort)

	_, outA := startRNSDServiceRNID(t, ctx, rnsdBin, nodeADir, root)
	_, outB := startRNSDServiceRNID(t, ctx, rnsdBin, nodeBDir, root)
	_ = waitForRNStatusSuccessRNID(t, rnstatusBin, nodeADir, root, 10*time.Second)
	_ = waitForRNStatusSuccessRNID(t, rnstatusBin, nodeBDir, root, 10*time.Second)

	requestOut, requestCode := runRNID(t, ctx, rnidBin, nodeADir, root,
		"--config", nodeADir,
		"--identity", strings.Repeat("a", 32),
		"--request",
		"--timeout", "1",
		"--print-identity",
	)
	skipIfReticulumUnavailable(t, outA.String()+outB.String()+requestOut, requestCode)
	if requestCode != 6 {
		t.Fatalf("expected unknown-hash request exit 6, got %d\n%s", requestCode, requestOut)
	}
}
