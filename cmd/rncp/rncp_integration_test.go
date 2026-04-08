//go:build integration

package main

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
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
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		res := cmdtest.Run(t, ctx, rnstatusBin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: workDir}, "--config", cfg, "--json")
		cancel()
		if res.ExitCode == 0 {
			return res.Output
		}
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

func buildRNCP(t *testing.T, binDir string) string {
	t.Helper()
	name := "rncp"
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
	cmd := exec.Command("go", "build", "-o", out, "./cmd/rncp")
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(),
		"GOCACHE="+gocache,
		"GOTMPDIR="+gotmp,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("build rncp: %v", err)
	}
	return out
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
	if !strings.Contains(out, "file not found") {
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
