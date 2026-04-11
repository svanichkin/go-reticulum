//go:build integration

package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/svanichkin/go-reticulum/internal/cmdtest"
)

var listenLineRe = regexp.MustCompile(`\brncp listening on\s+<?([0-9a-fA-F]+)>?\b`)

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func repoRoot(t *testing.T) string {
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
			t.Fatalf("could not find repo root (go.mod) from %s", wd)
		}
		dir = parent
	}
}

func writeMinimalReticulumConfig(t *testing.T, configDir string) {
	t.Helper()
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir configdir: %v", err)
	}
	// Keep instance name short (macOS UNIX socket paths have tight length limits).
	instanceName := "rncp-" + filepath.Base(configDir)
	cfg := strings.Join([]string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"shared_instance_type = unix",
		"instance_name = " + instanceName,
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(configDir, "config"), []byte(cfg), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func writeReticulumConfigRNCP(t *testing.T, configDir string, lines []string) {
	t.Helper()
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir configdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config"), []byte(strings.Join(lines, "\n")), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func freeTCPPortRNCP(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen ephemeral tcp port: %v", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func writeTwoNodeTemplateConfigRNCP(t *testing.T, dstDir, templatePath string, sharedPort, controlPort, listenPort, forwardPort int) {
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

func startRNSDServiceRNCP(t *testing.T, ctx context.Context, bin, cfg, workDir string) (*exec.Cmd, *lockedBuffer) {
	t.Helper()
	c := exec.CommandContext(ctx, bin, "--config", cfg, "--service")
	c.Dir = workDir
	home := filepath.Join(cfg, ".home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	c.Env = append(os.Environ(),
		"HOME="+home,
		"USERPROFILE="+home,
	)
	var buf lockedBuffer
	c.Stdout = &buf
	c.Stderr = &buf
	if err := c.Start(); err != nil {
		t.Fatalf("start rnsd: %v", err)
	}
	t.Cleanup(func() {
		if c.Process != nil {
			_ = c.Process.Signal(syscall.SIGTERM)
		}
		_ = c.Wait()
	})
	return c, &buf
}

func waitForRNStatusSuccessRNCP(t *testing.T, rnstatusBin, cfg, workDir string, timeout time.Duration) string {
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

func extractProbeHashFromRNStatusRNCP(t *testing.T, out string) string {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "Probe responder at") || !strings.Contains(line, "active") {
			continue
		}
		start := strings.Index(line, "<")
		end := strings.Index(line, ">")
		if start >= 0 && end > start {
			return line[start+1 : end]
		}
	}
	t.Fatalf("could not extract probe responder hash from rnstatus output:\n%s", out)
	return ""
}

func buildRNCP(t *testing.T, binDir string) string {
	t.Helper()
	return cmdtest.Build(t, binDir, "rncp", "./cmd/rncp")
}

func startListener(t *testing.T, ctx context.Context, bin, configDir, saveDir, jailDir string, allowFetch bool) (cmd *exec.Cmd, dest string, out *lockedBuffer) {
	t.Helper()

	args := []string{"--config", configDir, "--listen", "--no-auth", "-b", "0"}
	if saveDir != "" {
		args = append(args, "--save", saveDir)
	}
	if jailDir != "" {
		args = append(args, "--jail", jailDir)
	}
	if allowFetch {
		args = append(args, "--allow-fetch")
	}

	c := exec.CommandContext(ctx, bin, args...)

	// Isolate shared-instance sockets/state to avoid interacting with a user-level daemon.
	home := filepath.Join(configDir, ".home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	c.Env = append(os.Environ(),
		"HOME="+home,
		"USERPROFILE="+home, // windows compatibility
	)

	var buf lockedBuffer
	stdout, err := c.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	stderr, err := c.StderrPipe()
	if err != nil {
		t.Fatalf("stderr pipe: %v", err)
	}
	if err := c.Start(); err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("environment does not allow local Reticulum transport setup; skipping rncp integration test\n%v", err)
		}
		t.Fatalf("start listener: %v", err)
	}

	// Stream stderr for debugging, but don't block if it is noisy.
	go func() {
		_, _ = io.Copy(&buf, stderr)
	}()

	destCh := make(chan string, 1)
	go func() {
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			line := sc.Text()
			_, _ = buf.Write([]byte(line + "\n"))
			if m := listenLineRe.FindStringSubmatch(line); len(m) == 2 {
				destCh <- m[1]
				return
			}
		}
	}()

	select {
	case d := <-destCh:
		return c, d, &buf
	case <-time.After(20 * time.Second):
		_ = c.Process.Signal(syscall.SIGTERM)
		_ = c.Wait()
		if strings.Contains(buf.String(), "operation not permitted") ||
			strings.Contains(buf.String(), "could not be connected") ||
			strings.Contains(buf.String(), "No interfaces could process") ||
			ctx.Err() != nil {
			t.Skipf("environment does not allow local Reticulum transport setup; skipping rncp integration test\n%s", buf.String())
		}
		t.Fatalf("listener did not print destination in time; output:\n%s", buf.String())
		return nil, "", nil
	}
}

func runRNCP(t *testing.T, ctx context.Context, bin string, configDir string, workDir string, args ...string) (string, error) {
	t.Helper()
	c := exec.CommandContext(ctx, bin, args...)
	c.Dir = workDir

	home := filepath.Join(configDir, ".home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	c.Env = append(os.Environ(),
		"HOME="+home,
		"USERPROFILE="+home,
	)

	out, err := c.CombinedOutput()
	return string(out), err
}

func stopProcess(t *testing.T, c *exec.Cmd, buf *lockedBuffer) {
	t.Helper()
	if c == nil || c.Process == nil {
		return
	}
	_ = c.Process.Signal(syscall.SIGINT)
	done := make(chan error, 1)
	go func() { done <- c.Wait() }()
	select {
	case <-time.After(4 * time.Second):
		_ = c.Process.Kill()
		_ = <-done
		t.Fatalf("listener did not exit; output:\n%s", buf.String())
	case <-done:
	}
}

func skipIfReticulumUnavailableRNCP(t *testing.T, out string, err error) {
	t.Helper()
	if strings.Contains(out, "Could not start Reticulum") ||
		strings.Contains(out, "operation not permitted") ||
		strings.Contains(out, "No interfaces could process") {
		t.Skipf("environment does not allow Reticulum startup; skipping rncp integration test\nerr=%v\n%s", err, out)
	}
}

func TestRNCPIntegration_HelpAndVersion(t *testing.T) {
	root := t.TempDir()
	bin := buildRNCP(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, err := runRNCP(t, ctx, bin, root, root, "--help")
	if err != nil {
		t.Fatalf("help failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Reticulum File Transfer Utility") && !strings.Contains(out, "Usage:") {
		t.Fatalf("unexpected help output:\n%s", out)
	}

	out, err = runRNCP(t, ctx, bin, root, root, "--version")
	if err != nil {
		t.Fatalf("version failed: %v\n%s", err, out)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), "rncp ") {
		t.Fatalf("unexpected version output: %q", out)
	}
}

func TestRNCPIntegration_MissingArgsShowsUsageExit0(t *testing.T) {
	root := t.TempDir()
	bin := buildRNCP(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, err := runRNCP(t, ctx, bin, root, root)
	if err != nil {
		t.Fatalf("expected usage exit 0, got %v\n%s", err, out)
	}
	if !strings.Contains(out, "Usage:") || !strings.Contains(out, "rncp --fetch [options] file destination") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestRNCPIntegration_InvalidTimeoutFlagExit2(t *testing.T) {
	root := t.TempDir()
	bin := buildRNCP(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, err := runRNCP(t, ctx, bin, root, root, "--w", "not-a-float")
	if err == nil {
		t.Fatalf("expected invalid timeout failure, got success\n%s", out)
	}
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 2 {
		t.Fatalf("expected exit 2, got %v\n%s", err, out)
	}
	if !strings.Contains(out, "invalid value") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestRNCPIntegration_PrintIdentity(t *testing.T) {
	root := t.TempDir()
	bin := buildRNCP(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	configDir := filepath.Join(root, "cfg")
	writeMinimalReticulumConfig(t, configDir)

	out, err := runRNCP(t, ctx, bin, configDir, root, "--config", configDir, "--print-identity")
	skipIfReticulumUnavailableRNCP(t, out, err)
	if err != nil {
		t.Fatalf("print-identity failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Identity     :") || !strings.Contains(out, "Listening on :") {
		t.Fatalf("unexpected print-identity output:\n%s", out)
	}
}

func TestRNCPIntegration_SharedInstancePrintIdentity(t *testing.T) {
	root := t.TempDir()
	bin := buildRNCP(t, root)
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	configDir := filepath.Join(root, "cfg")
	writeReticulumConfigRNCP(t, configDir, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = rncp-shared-local",
		"",
	})

	_, serviceOut := startRNSDServiceRNCP(t, ctx, rnsdBin, configDir, root)
	_ = waitForRNStatusSuccessRNCP(t, rnstatusBin, configDir, root, 10*time.Second)

	out, err := runRNCP(t, ctx, bin, configDir, root, "--config", configDir, "--print-identity")
	skipIfReticulumUnavailableRNCP(t, serviceOut.String()+out, err)
	if err != nil {
		t.Fatalf("shared-instance print-identity failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Identity     :") || !strings.Contains(out, "Listening on :") {
		t.Fatalf("unexpected print-identity output:\n%s", out)
	}
}

func TestRNCPIntegration_SharedInstanceTCPPrintIdentity(t *testing.T) {
	root := t.TempDir()
	bin := buildRNCP(t, root)
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	configDir := filepath.Join(root, "cfg")
	packetPort := freeTCPPortRNCP(t)
	controlPort := freeTCPPortRNCP(t)
	writeReticulumConfigRNCP(t, configDir, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"shared_instance_type = tcp",
		"shared_instance_port = " + strconv.Itoa(packetPort),
		"instance_control_port = " + strconv.Itoa(controlPort),
		"",
	})

	_, serviceOut := startRNSDServiceRNCP(t, ctx, rnsdBin, configDir, root)
	_ = waitForRNStatusSuccessRNCP(t, rnstatusBin, configDir, root, 10*time.Second)

	out, err := runRNCP(t, ctx, bin, configDir, root, "--config", configDir, "--print-identity")
	skipIfReticulumUnavailableRNCP(t, serviceOut.String()+out, err)
	if err != nil {
		t.Fatalf("shared-instance tcp print-identity failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Identity     :") || !strings.Contains(out, "Listening on :") {
		t.Fatalf("unexpected print-identity output:\n%s", out)
	}
}

func TestRNCPIntegration_PrintIdentityCorruptIdentityExit2(t *testing.T) {
	root := t.TempDir()
	bin := buildRNCP(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	configDir := filepath.Join(root, "cfg")
	writeMinimalReticulumConfig(t, configDir)

	identityPath := filepath.Join(root, "bad.id")
	if err := os.WriteFile(identityPath, []byte("bad-identity"), 0o600); err != nil {
		t.Fatal(err)
	}

	out, err := runRNCP(t, ctx, bin, configDir, root, "--config", configDir, "--identity", identityPath, "--print-identity")
	skipIfReticulumUnavailableRNCP(t, out, err)
	if err == nil {
		t.Fatalf("expected corrupt identity error, got success\n%s", out)
	}
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 2 {
		t.Fatalf("expected exit 2, got %v\n%s", err, out)
	}
	if !strings.Contains(out, "identity error") && !strings.Contains(out, "Could not load identity for rncp") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestRNCPIntegration_ListenMissingSaveDirExit3(t *testing.T) {
	root := t.TempDir()
	bin := buildRNCP(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	configDir := filepath.Join(root, "cfg")
	writeMinimalReticulumConfig(t, configDir)

	out, err := runRNCP(t, ctx, bin, configDir, root, "--config", configDir, "--listen", "--save", filepath.Join(root, "missing"))
	skipIfReticulumUnavailableRNCP(t, out, err)
	if err == nil {
		t.Fatalf("expected save-dir failure, got success\n%s", out)
	}
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 3 {
		t.Fatalf("expected exit 3, got %v\n%s", err, out)
	}
	if !strings.Contains(out, "Output directory not found") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestRNCPIntegration_FetchMissingSaveDirExit3(t *testing.T) {
	root := t.TempDir()
	bin := buildRNCP(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	configDir := filepath.Join(root, "cfg")
	writeMinimalReticulumConfig(t, configDir)

	out, err := runRNCP(t, ctx, bin, configDir, root,
		"--config", configDir,
		"--fetch",
		"--save", filepath.Join(root, "missing"),
		"file.txt",
		strings.Repeat("0", 32),
	)
	skipIfReticulumUnavailableRNCP(t, out, err)
	if err == nil {
		t.Fatalf("expected fetch save-dir failure, got success\n%s", out)
	}
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 3 {
		t.Fatalf("expected exit 3, got %v\n%s", err, out)
	}
	if !strings.Contains(out, "Output directory not found") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestRNCPIntegration_ListenInvalidAllowedIdentityExit1(t *testing.T) {
	root := t.TempDir()
	bin := buildRNCP(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	configDir := filepath.Join(root, "cfg")
	writeMinimalReticulumConfig(t, configDir)

	out, err := runRNCP(t, ctx, bin, configDir, root, "--config", configDir, "--listen", "-a", "abcd")
	skipIfReticulumUnavailableRNCP(t, out, err)
	if err == nil {
		t.Fatalf("expected invalid allowed identity failure, got success\n%s", out)
	}
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 1 {
		t.Fatalf("expected exit 1, got %v\n%s", err, out)
	}
	if !strings.Contains(out, "Allowed destination length is invalid") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestRNCPIntegration_SendFileNotFoundExit1(t *testing.T) {
	root := t.TempDir()
	bin := buildRNCP(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	configDir := filepath.Join(root, "cfg")
	writeMinimalReticulumConfig(t, configDir)

	out, err := runRNCP(t, ctx, bin, configDir, root, "--config", configDir, filepath.Join(root, "missing.txt"), strings.Repeat("0", 32))
	skipIfReticulumUnavailableRNCP(t, out, err)
	if err == nil {
		t.Fatalf("expected missing file failure, got success\n%s", out)
	}
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 1 {
		t.Fatalf("expected exit 1, got %v\n%s", err, out)
	}
	if !strings.Contains(strings.ToLower(out), "file not found") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestRNCPIntegration_SendInvalidDestinationExit1(t *testing.T) {
	root := t.TempDir()
	bin := buildRNCP(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	configDir := filepath.Join(root, "cfg")
	writeMinimalReticulumConfig(t, configDir)

	srcPath := filepath.Join(root, "hello.txt")
	if err := os.WriteFile(srcPath, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runRNCP(t, ctx, bin, configDir, root, "--config", configDir, srcPath, "abcd")
	skipIfReticulumUnavailableRNCP(t, out, err)
	if err == nil {
		t.Fatalf("expected invalid destination failure, got success\n%s", out)
	}
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 1 {
		t.Fatalf("expected exit 1, got %v\n%s", err, out)
	}
	if !strings.Contains(out, "Allowed destination length is invalid") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestRNCPIntegration_CountFlagsExpand(t *testing.T) {
	root := t.TempDir()
	bin := buildRNCP(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	configDir := filepath.Join(root, "cfg")
	writeMinimalReticulumConfig(t, configDir)

	out, err := runRNCP(t, ctx, bin, configDir, root, "--config", configDir, "-vvv", "-qq", "--print-identity")
	skipIfReticulumUnavailableRNCP(t, out, err)
	if err != nil {
		t.Fatalf("count-flags print-identity failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Identity     :") || !strings.Contains(out, "Listening on :") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestRNCPIntegration_SendReceive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	root := t.TempDir()
	bin := buildRNCP(t, root)
	configDir := filepath.Join(root, "cfg")
	writeMinimalReticulumConfig(t, configDir)
	recvDir := filepath.Join(root, "recv")
	sendDir := filepath.Join(root, "send")
	if err := os.MkdirAll(recvDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sendDir, 0o755); err != nil {
		t.Fatal(err)
	}

	listener, dest, buf := startListener(t, ctx, bin, configDir, recvDir, "", false)
	t.Cleanup(func() { stopProcess(t, listener, buf) })

	payload := []byte("hello rncp integration\n")
	srcPath := filepath.Join(sendDir, "hello.txt")
	if err := os.WriteFile(srcPath, payload, 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runRNCP(t, ctx, bin, configDir, sendDir, "--config", configDir, srcPath, dest)
	if err != nil {
		if strings.Contains(out, "No interfaces could process the outbound packet") ||
			strings.Contains(out, "Path not found") ||
			strings.Contains(out, "could not be connected") {
			t.Skipf("environment does not allow local Reticulum transport setup; skipping send/receive parity test\n%s", out)
		}
		t.Fatalf("send failed: %v\n%s", err, out)
	}

	dstPath := filepath.Join(recvDir, "hello.txt")
	got, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("read received file: %v\nlistener output:\n%s", err, buf.String())
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("received payload mismatch: got %q want %q", got, payload)
	}
}

func TestRNCPIntegration_Fetch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()

	root := t.TempDir()
	bin := buildRNCP(t, root)
	configDir := filepath.Join(root, "cfg")
	writeMinimalReticulumConfig(t, configDir)
	serverDir := filepath.Join(root, "server")
	clientDir := filepath.Join(root, "client")
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(clientDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// File served by listener.
	payload := []byte("fetch me\n")
	servedName := "served.txt"
	servedPath := filepath.Join(serverDir, servedName)
	if err := os.WriteFile(servedPath, payload, 0o644); err != nil {
		t.Fatal(err)
	}

	listener, dest, buf := startListener(t, ctx, bin, configDir, "", serverDir, true)
	t.Cleanup(func() { stopProcess(t, listener, buf) })

	out, err := runRNCP(t, ctx, bin, configDir, clientDir,
		"--config", configDir,
		"--fetch",
		"--save", clientDir,
		servedName,
		dest,
	)
	if err != nil {
		if strings.Contains(out, "No interfaces could process the outbound packet") ||
			strings.Contains(out, "Path not found") ||
			strings.Contains(out, "could not be connected") {
			t.Skipf("environment does not allow local Reticulum transport setup; skipping fetch parity test\n%s", out)
		}
	}

	dstPath := filepath.Join(clientDir, servedName)
	got, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("read fetched file: %v\nclient output:\n%s\nlistener output:\n%s", err, out, buf.String())
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("fetched payload mismatch: got %q want %q", got, payload)
	}
}

func TestRNCPIntegration_Fetch_JailTraversalRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()

	root := t.TempDir()
	bin := buildRNCP(t, root)
	configDir := filepath.Join(root, "cfg")
	writeMinimalReticulumConfig(t, configDir)
	serverDir := filepath.Join(root, "server")
	clientDir := filepath.Join(root, "client")
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(clientDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Start listener with a jail, but do not create any files outside it.
	listener, dest, buf := startListener(t, ctx, bin, configDir, "", serverDir, true)
	t.Cleanup(func() { stopProcess(t, listener, buf) })

	// Attempt path traversal outside jail.
	out, _ := runRNCP(t, ctx, bin, configDir, clientDir,
		"--config", configDir,
		"--fetch",
		"--save", clientDir,
		"../nope",
		dest,
	)
	if strings.Contains(out, "No interfaces could process the outbound packet") ||
		strings.Contains(out, "Path not found") ||
		strings.Contains(out, "could not be connected") {
		t.Skipf("environment does not allow local Reticulum transport setup; skipping jail traversal integration test\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(clientDir, "nope")); err == nil {
		t.Fatalf("unexpectedly fetched file outside jail; output:\n%s", out)
	}
}

func TestRNCPIntegration_TwoNodeSendReceive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	root := t.TempDir()
	bin := buildRNCP(t, root)
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")

	basePort := freeTCPPortRNCP(t)
	sharedA := freeTCPPortRNCP(t)
	controlA := freeTCPPortRNCP(t)
	sharedB := freeTCPPortRNCP(t)
	controlB := freeTCPPortRNCP(t)

	nodeADir := filepath.Join(root, "node-a")
	nodeBDir := filepath.Join(root, "node-b")
	writeTwoNodeTemplateConfigRNCP(t, nodeADir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_a/config"),
		sharedA, controlA, basePort, basePort+1)
	writeTwoNodeTemplateConfigRNCP(t, nodeBDir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_b/config"),
		sharedB, controlB, basePort+1, basePort)

	_, outA := startRNSDServiceRNCP(t, ctx, rnsdBin, nodeADir, root)
	_, outB := startRNSDServiceRNCP(t, ctx, rnsdBin, nodeBDir, root)
	_ = waitForRNStatusSuccessRNCP(t, rnstatusBin, nodeADir, root, 20*time.Second)
	_ = waitForRNStatusSuccessRNCP(t, rnstatusBin, nodeBDir, root, 20*time.Second)

	recvDir := filepath.Join(root, "recv")
	sendDir := filepath.Join(root, "send")
	if err := os.MkdirAll(recvDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sendDir, 0o755); err != nil {
		t.Fatal(err)
	}

	listener, dest, buf := startListener(t, ctx, bin, nodeBDir, recvDir, "", false)
	t.Cleanup(func() { stopProcess(t, listener, buf) })

	payload := []byte("hello rncp two-node integration\n")
	srcPath := filepath.Join(sendDir, "hello.txt")
	if err := os.WriteFile(srcPath, payload, 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runRNCP(t, ctx, bin, nodeADir, sendDir, "--config", nodeADir, srcPath, dest)
	if err != nil {
		skipIfReticulumUnavailableRNCP(t, outA.String()+outB.String()+out+buf.String(), err)
		t.Fatalf("two-node send failed: %v\nsender:\n%s\nlistener:\n%s", err, out, buf.String())
	}

	dstPath := filepath.Join(recvDir, "hello.txt")
	got, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("read received file: %v\nlistener output:\n%s\nnodeA:\n%s\nnodeB:\n%s", err, buf.String(), outA.String(), outB.String())
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("received payload mismatch: got %q want %q", got, payload)
	}
}

func TestRNCPIntegration_TwoNodeFetch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	cmdtest.AcquireLock(t, "integration-two-node-shared", 5*time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	root := t.TempDir()
	bin := buildRNCP(t, root)
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")

	basePort := freeTCPPortRNCP(t)
	sharedA := freeTCPPortRNCP(t)
	controlA := freeTCPPortRNCP(t)
	sharedB := freeTCPPortRNCP(t)
	controlB := freeTCPPortRNCP(t)

	nodeADir := filepath.Join(root, "node-a")
	nodeBDir := filepath.Join(root, "node-b")
	writeTwoNodeTemplateConfigRNCP(t, nodeADir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_a/config"),
		sharedA, controlA, basePort, basePort+1)
	writeTwoNodeTemplateConfigRNCP(t, nodeBDir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_b/config"),
		sharedB, controlB, basePort+1, basePort)

	_, outA := startRNSDServiceRNCP(t, ctx, rnsdBin, nodeADir, root)
	_, outB := startRNSDServiceRNCP(t, ctx, rnsdBin, nodeBDir, root)
	_ = waitForRNStatusSuccessRNCP(t, rnstatusBin, nodeADir, root, 20*time.Second)
	_ = waitForRNStatusSuccessRNCP(t, rnstatusBin, nodeBDir, root, 20*time.Second)

	serverDir := filepath.Join(root, "server")
	clientDir := filepath.Join(root, "client")
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(clientDir, 0o755); err != nil {
		t.Fatal(err)
	}

	payload := []byte("fetch me two-node\n")
	servedName := "served.txt"
	servedPath := filepath.Join(serverDir, servedName)
	if err := os.WriteFile(servedPath, payload, 0o644); err != nil {
		t.Fatal(err)
	}

	listener, dest, buf := startListener(t, ctx, bin, nodeBDir, "", serverDir, true)
	t.Cleanup(func() { stopProcess(t, listener, buf) })

	out, err := runRNCP(t, ctx, bin, nodeADir, clientDir,
		"--config", nodeADir,
		"--fetch",
		"--save", clientDir,
		servedName,
		dest,
	)
	if err != nil {
		skipIfReticulumUnavailableRNCP(t, outA.String()+outB.String()+out+buf.String(), err)
		t.Fatalf("two-node fetch failed: %v\nclient:\n%s\nlistener:\n%s", err, out, buf.String())
	}

	dstPath := filepath.Join(clientDir, servedName)
	got, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("read fetched file: %v\nclient output:\n%s\nlistener output:\n%s", err, out, buf.String())
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("fetched payload mismatch: got %q want %q", got, payload)
	}
}

func TestRNCPIntegration_TwoNodeFetchDenied(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	cmdtest.AcquireLock(t, "integration-two-node-shared", 5*time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	root := t.TempDir()
	bin := buildRNCP(t, root)
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")

	basePort := freeTCPPortRNCP(t)
	sharedA := freeTCPPortRNCP(t)
	controlA := freeTCPPortRNCP(t)
	sharedB := freeTCPPortRNCP(t)
	controlB := freeTCPPortRNCP(t)

	nodeADir := filepath.Join(root, "node-a")
	nodeBDir := filepath.Join(root, "node-b")
	writeTwoNodeTemplateConfigRNCP(t, nodeADir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_a/config"),
		sharedA, controlA, basePort, basePort+1)
	writeTwoNodeTemplateConfigRNCP(t, nodeBDir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_b/config"),
		sharedB, controlB, basePort+1, basePort)

	_, outA := startRNSDServiceRNCP(t, ctx, rnsdBin, nodeADir, root)
	_, outB := startRNSDServiceRNCP(t, ctx, rnsdBin, nodeBDir, root)
	_ = waitForRNStatusSuccessRNCP(t, rnstatusBin, nodeADir, root, 20*time.Second)
	_ = waitForRNStatusSuccessRNCP(t, rnstatusBin, nodeBDir, root, 20*time.Second)

	serverDir := filepath.Join(root, "server")
	clientDir := filepath.Join(root, "client")
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(clientDir, 0o755); err != nil {
		t.Fatal(err)
	}
	servedName := "served.txt"
	if err := os.WriteFile(filepath.Join(serverDir, servedName), []byte("fetch denied\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	listener, dest, buf := startListener(t, ctx, bin, nodeBDir, "", serverDir, false)
	t.Cleanup(func() { stopProcess(t, listener, buf) })

	out, err := runRNCP(t, ctx, bin, nodeADir, clientDir,
		"--config", nodeADir,
		"--fetch",
		"--save", clientDir,
		servedName,
		dest,
	)
	if err == nil {
		t.Fatalf("expected denied fetch to fail\nclient:\n%s\nlistener:\n%s", out, buf.String())
	}
	skipIfReticulumUnavailableRNCP(t, outA.String()+outB.String()+out+buf.String(), err)
	if !strings.Contains(out, "was not allowed by the remote") {
		t.Fatalf("unexpected denied fetch output:\n%s", out)
	}
	if _, statErr := os.Stat(filepath.Join(clientDir, servedName)); !os.IsNotExist(statErr) {
		t.Fatalf("unexpected fetched file present after denied fetch: %v", statErr)
	}
}

func TestRNCPIntegration_TwoNodeFetchMissingFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	cmdtest.AcquireLock(t, "integration-two-node-shared", 5*time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	root := t.TempDir()
	bin := buildRNCP(t, root)
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")

	basePort := freeTCPPortRNCP(t)
	sharedA := freeTCPPortRNCP(t)
	controlA := freeTCPPortRNCP(t)
	sharedB := freeTCPPortRNCP(t)
	controlB := freeTCPPortRNCP(t)

	nodeADir := filepath.Join(root, "node-a")
	nodeBDir := filepath.Join(root, "node-b")
	writeTwoNodeTemplateConfigRNCP(t, nodeADir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_a/config"),
		sharedA, controlA, basePort, basePort+1)
	writeTwoNodeTemplateConfigRNCP(t, nodeBDir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_b/config"),
		sharedB, controlB, basePort+1, basePort)

	_, outA := startRNSDServiceRNCP(t, ctx, rnsdBin, nodeADir, root)
	_, outB := startRNSDServiceRNCP(t, ctx, rnsdBin, nodeBDir, root)
	_ = waitForRNStatusSuccessRNCP(t, rnstatusBin, nodeADir, root, 20*time.Second)
	_ = waitForRNStatusSuccessRNCP(t, rnstatusBin, nodeBDir, root, 20*time.Second)

	serverDir := filepath.Join(root, "server")
	clientDir := filepath.Join(root, "client")
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(clientDir, 0o755); err != nil {
		t.Fatal(err)
	}

	listener, dest, buf := startListener(t, ctx, bin, nodeBDir, "", serverDir, true)
	t.Cleanup(func() { stopProcess(t, listener, buf) })

	missingName := "missing.txt"
	out, err := runRNCP(t, ctx, bin, nodeADir, clientDir,
		"--config", nodeADir,
		"--fetch",
		"--save", clientDir,
		missingName,
		dest,
	)
	skipIfReticulumUnavailableRNCP(t, outA.String()+outB.String()+out+buf.String(), err)
	if !strings.Contains(out, "was not found on the remote") {
		t.Fatalf("unexpected missing fetch output:\n%s", out)
	}
	if _, statErr := os.Stat(filepath.Join(clientDir, missingName)); !os.IsNotExist(statErr) {
		t.Fatalf("unexpected fetched file present after missing fetch: %v", statErr)
	}
}

func TestRNCPIntegration_TwoNodeRepeatedFetch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	root := t.TempDir()
	bin := buildRNCP(t, root)
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")

	basePort := freeTCPPortRNCP(t)
	sharedA := freeTCPPortRNCP(t)
	controlA := freeTCPPortRNCP(t)
	sharedB := freeTCPPortRNCP(t)
	controlB := freeTCPPortRNCP(t)

	nodeADir := filepath.Join(root, "node-a")
	nodeBDir := filepath.Join(root, "node-b")
	writeTwoNodeTemplateConfigRNCP(t, nodeADir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_a/config"),
		sharedA, controlA, basePort, basePort+1)
	writeTwoNodeTemplateConfigRNCP(t, nodeBDir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_b/config"),
		sharedB, controlB, basePort+1, basePort)

	_, outA := startRNSDServiceRNCP(t, ctx, rnsdBin, nodeADir, root)
	_, outB := startRNSDServiceRNCP(t, ctx, rnsdBin, nodeBDir, root)
	_ = waitForRNStatusSuccessRNCP(t, rnstatusBin, nodeADir, root, 20*time.Second)
	_ = waitForRNStatusSuccessRNCP(t, rnstatusBin, nodeBDir, root, 20*time.Second)

	serverDir := filepath.Join(root, "server")
	clientDir := filepath.Join(root, "client")
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(clientDir, 0o755); err != nil {
		t.Fatal(err)
	}

	payload := []byte("fetch me twice\n")
	servedName := "twice.txt"
	if err := os.WriteFile(filepath.Join(serverDir, servedName), payload, 0o644); err != nil {
		t.Fatal(err)
	}

	listener, dest, buf := startListener(t, ctx, bin, nodeBDir, "", serverDir, true)
	t.Cleanup(func() { stopProcess(t, listener, buf) })

	for i := 0; i < 2; i++ {
		targetDir := filepath.Join(clientDir, strconv.Itoa(i))
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatal(err)
		}
		out, err := runRNCP(t, ctx, bin, nodeADir, targetDir,
			"--config", nodeADir,
			"--fetch",
			"--save", targetDir,
			servedName,
			dest,
		)
		if err != nil {
			skipIfReticulumUnavailableRNCP(t, outA.String()+outB.String()+out+buf.String(), err)
			t.Fatalf("repeated fetch %d failed: %v\nclient:\n%s\nlistener:\n%s", i+1, err, out, buf.String())
		}
		got, err := os.ReadFile(filepath.Join(targetDir, servedName))
		if err != nil {
			t.Fatalf("read fetched file run %d: %v", i+1, err)
		}
		if !bytes.Equal(got, payload) {
			t.Fatalf("repeated fetch %d mismatch: got %q want %q", i+1, got, payload)
		}
	}
}

func TestRNCPIntegration_TwoNodeLargeSendIntegrity(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	root := t.TempDir()
	bin := buildRNCP(t, root)
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")

	basePort := freeTCPPortRNCP(t)
	sharedA := freeTCPPortRNCP(t)
	controlA := freeTCPPortRNCP(t)
	sharedB := freeTCPPortRNCP(t)
	controlB := freeTCPPortRNCP(t)

	nodeADir := filepath.Join(root, "node-a")
	nodeBDir := filepath.Join(root, "node-b")
	writeTwoNodeTemplateConfigRNCP(t, nodeADir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_a/config"),
		sharedA, controlA, basePort, basePort+1)
	writeTwoNodeTemplateConfigRNCP(t, nodeBDir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_b/config"),
		sharedB, controlB, basePort+1, basePort)

	_, outA := startRNSDServiceRNCP(t, ctx, rnsdBin, nodeADir, root)
	_, outB := startRNSDServiceRNCP(t, ctx, rnsdBin, nodeBDir, root)
	_ = waitForRNStatusSuccessRNCP(t, rnstatusBin, nodeADir, root, 10*time.Second)
	_ = waitForRNStatusSuccessRNCP(t, rnstatusBin, nodeBDir, root, 10*time.Second)

	recvDir := filepath.Join(root, "recv")
	sendDir := filepath.Join(root, "send")
	if err := os.MkdirAll(recvDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sendDir, 0o755); err != nil {
		t.Fatal(err)
	}

	listener, dest, buf := startListener(t, ctx, bin, nodeBDir, recvDir, "", false)
	t.Cleanup(func() { stopProcess(t, listener, buf) })

	payload := bytes.Repeat([]byte("0123456789abcdef"), 64*1024) // 1 MiB
	srcPath := filepath.Join(sendDir, "large.bin")
	if err := os.WriteFile(srcPath, payload, 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runRNCP(t, ctx, bin, nodeADir, sendDir, "--config", nodeADir, srcPath, dest)
	if err != nil {
		skipIfReticulumUnavailableRNCP(t, outA.String()+outB.String()+out+buf.String(), err)
		t.Fatalf("large two-node send failed: %v\nsender:\n%s\nlistener:\n%s", err, out, buf.String())
	}

	got, err := os.ReadFile(filepath.Join(recvDir, "large.bin"))
	if err != nil {
		t.Fatalf("read received large file: %v\nlistener output:\n%s", err, buf.String())
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("large payload mismatch: got %d bytes want %d bytes", len(got), len(payload))
	}
}

func TestRNCPIntegration_TwoNodeLargeFetchAfterPathReuseIntegrity(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	root := t.TempDir()
	bin := buildRNCP(t, root)
	rnpathBin := cmdtest.Build(t, root, "rnpath", "./cmd/rnpath")
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")

	basePort := freeTCPPortRNCP(t)
	sharedA := freeTCPPortRNCP(t)
	controlA := freeTCPPortRNCP(t)
	sharedB := freeTCPPortRNCP(t)
	controlB := freeTCPPortRNCP(t)

	nodeADir := filepath.Join(root, "node-a")
	nodeBDir := filepath.Join(root, "node-b")
	writeTwoNodeTemplateConfigRNCP(t, nodeADir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_a/config"),
		sharedA, controlA, basePort, basePort+1)
	writeTwoNodeTemplateConfigRNCP(t, nodeBDir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_b/config"),
		sharedB, controlB, basePort+1, basePort)

	_, outA := startRNSDServiceRNCP(t, ctx, rnsdBin, nodeADir, root)
	_, outB := startRNSDServiceRNCP(t, ctx, rnsdBin, nodeBDir, root)
	_ = waitForRNStatusSuccessRNCP(t, rnstatusBin, nodeADir, root, 10*time.Second)
	_ = waitForRNStatusSuccessRNCP(t, rnstatusBin, nodeBDir, root, 10*time.Second)

	statusCtx, statusCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer statusCancel()
	statusRes := cmdtest.Run(t, statusCtx, rnstatusBin, cmdtest.RunOptions{ConfigDir: nodeBDir, WorkDir: root},
		"--config", nodeBDir, "-a")
	if statusRes.ExitCode != 0 {
		skipIfReticulumUnavailableRNCP(t, outA.String()+outB.String()+statusRes.Output, errors.New("rnstatus failed"))
		t.Fatalf("rnstatus on node B failed: %d\n%s", statusRes.ExitCode, statusRes.Output)
	}
	probeHash := extractProbeHashFromRNStatusRNCP(t, statusRes.Output)

	pathCtx, pathCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer pathCancel()
	pathRes := cmdtest.Run(t, pathCtx, rnpathBin, cmdtest.RunOptions{ConfigDir: nodeADir, WorkDir: root},
		"--config", nodeADir, "-w", "15", probeHash)
	if pathRes.ExitCode != 0 {
		t.Fatalf("pre-fetch path discovery failed: %d\nnodeA:\n%s\nnodeB:\n%s\nrnpath:\n%s",
			pathRes.ExitCode, outA.String(), outB.String(), pathRes.Output)
	}

	serverDir := filepath.Join(root, "server")
	clientDir := filepath.Join(root, "client")
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(clientDir, 0o755); err != nil {
		t.Fatal(err)
	}

	payload := bytes.Repeat([]byte("fetch-chunk-block-0123456789"), 48*1024)
	servedName := "large-fetch.bin"
	if err := os.WriteFile(filepath.Join(serverDir, servedName), payload, 0o644); err != nil {
		t.Fatal(err)
	}

	listener, dest, buf := startListener(t, ctx, bin, nodeBDir, "", serverDir, true)
	t.Cleanup(func() { stopProcess(t, listener, buf) })

	out, err := runRNCP(t, ctx, bin, nodeADir, clientDir,
		"--config", nodeADir,
		"--fetch",
		"--save", clientDir,
		"-w", "1",
		servedName,
		dest,
	)
	if err != nil {
		skipIfReticulumUnavailableRNCP(t, outA.String()+outB.String()+pathRes.Output+out+buf.String(), err)
		t.Fatalf("large fetch after path reuse failed: %v\nclient:\n%s\nlistener:\n%s", err, out, buf.String())
	}

	got, err := os.ReadFile(filepath.Join(clientDir, servedName))
	if err != nil {
		t.Fatalf("read fetched large file: %v\nclient:\n%s\nlistener:\n%s", err, out, buf.String())
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("large fetch payload mismatch: got %d bytes want %d bytes", len(got), len(payload))
	}
}
