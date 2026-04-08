//go:build integration

package main

import (
	"bytes"
	"context"
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
)

func writeMinimalReticulumConfigRNX(t *testing.T, configDir string) {
	t.Helper()
	cmdtest.WriteReticulumConfig(t, configDir, strings.Join([]string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = No",
		"",
	}, "\n"))
}

func skipIfReticulumUnavailableRNX(t *testing.T, out string, exitCode int) {
	t.Helper()
	if exitCode == 1 && (strings.Contains(out, "listen error:") || strings.Contains(out, "operation not permitted")) {
		t.Skipf("environment does not allow Reticulum startup; skipping rnx integration test\n%s", out)
	}
	if exitCode == 241 && (strings.Contains(out, "Could not initialise Reticulum") || strings.Contains(out, "operation not permitted")) {
		t.Skipf("environment does not allow Reticulum startup; skipping rnx integration test\n%s", out)
	}
}

func writeReticulumConfigRNX(t *testing.T, configDir string, lines []string) {
	t.Helper()
	cmdtest.WriteReticulumConfig(t, configDir, strings.Join(lines, "\n"))
}

func freeTCPPortRNX(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen ephemeral tcp port: %v", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func startRNSDServiceRNX(t *testing.T, ctx context.Context, bin, cfg, workDir string) (*exec.Cmd, *bytes.Buffer) {
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

func waitForRNStatusSuccessRNX(t *testing.T, rnstatusBin, cfg, workDir string, timeout time.Duration) string {
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

func TestRNXIntegration_HelpAndVersion(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnx", "./cmd/rnx")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: root, WorkDir: root}, "--help")
	if res.ExitCode != 0 {
		t.Fatalf("help exit=%d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "Reticulum Remote Execution Utility") {
		t.Fatalf("unexpected help output:\n%s", res.Output)
	}

	res = cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: root, WorkDir: root}, "--version")
	if res.ExitCode != 0 {
		t.Fatalf("version exit=%d\n%s", res.ExitCode, res.Output)
	}
	if !strings.HasPrefix(res.Output, "rnx ") {
		t.Fatalf("unexpected version output: %q", res.Output)
	}
}

func TestRNXIntegration_MissingArgsShowsUsageExit0(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnx", "./cmd/rnx")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: root, WorkDir: root})
	if res.ExitCode != 0 {
		t.Fatalf("expected exit 0, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "Usage:") || !strings.Contains(res.Output, "-destination") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNXIntegration_InteractiveWithoutDestinationShowsUsageExit0(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnx", "./cmd/rnx")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: root, WorkDir: root}, "--interactive")
	if res.ExitCode != 0 {
		t.Fatalf("expected exit 0, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "Usage:") || !strings.Contains(res.Output, "-destination") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNXIntegration_InvalidTimeoutFlagExit2(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnx", "./cmd/rnx")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: root, WorkDir: root}, "--w", "not-a-float")
	if res.ExitCode != 2 {
		t.Fatalf("expected exit 2, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "invalid value") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNXIntegration_UnknownFlagExit2(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnx", "./cmd/rnx")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: root, WorkDir: root}, "--nope")
	if res.ExitCode != 2 {
		t.Fatalf("expected exit 2, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "unrecognized arguments") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNXIntegration_PrintIdentity(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnx", "./cmd/rnx")
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNX(t, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root}, "--config", cfg, "--print-identity")
	skipIfReticulumUnavailableRNX(t, res.Output, res.ExitCode)
	if res.ExitCode != 0 {
		t.Fatalf("print-identity exit=%d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "Identity     :") || !strings.Contains(res.Output, "Listening on :") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNXIntegration_SharedInstancePrintIdentity(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnx", "./cmd/rnx")
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNX(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = rnx-shared-local",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, out := startRNSDServiceRNX(t, ctx, rnsdBin, cfg, root)
	_ = waitForRNStatusSuccessRNX(t, rnstatusBin, cfg, root, 10*time.Second)

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root}, "--config", cfg, "--print-identity")
	skipIfReticulumUnavailableRNX(t, out.String()+res.Output, res.ExitCode)
	if res.ExitCode != 0 {
		t.Fatalf("shared-instance print-identity exit=%d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "Identity     :") || !strings.Contains(res.Output, "Listening on :") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNXIntegration_SharedInstanceTCPPrintIdentity(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnx", "./cmd/rnx")
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	packetPort := freeTCPPortRNX(t)
	controlPort := freeTCPPortRNX(t)
	writeReticulumConfigRNX(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"shared_instance_type = tcp",
		"shared_instance_port = " + strconv.Itoa(packetPort),
		"instance_control_port = " + strconv.Itoa(controlPort),
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, out := startRNSDServiceRNX(t, ctx, rnsdBin, cfg, root)
	_ = waitForRNStatusSuccessRNX(t, rnstatusBin, cfg, root, 10*time.Second)

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root}, "--config", cfg, "--print-identity")
	skipIfReticulumUnavailableRNX(t, out.String()+res.Output, res.ExitCode)
	if res.ExitCode != 0 {
		t.Fatalf("shared-instance tcp print-identity exit=%d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "Identity     :") || !strings.Contains(res.Output, "Listening on :") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNXIntegration_ListenInvalidAllowedIdentityExit1(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnx", "./cmd/rnx")
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNX(t, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root}, "--config", cfg, "--listen", "-a", "abcd")
	skipIfReticulumUnavailableRNX(t, res.Output, res.ExitCode)
	if res.ExitCode != 1 {
		t.Fatalf("expected exit 1, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "Allowed destination length is invalid") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNXIntegration_InvalidDestinationExit241(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnx", "./cmd/rnx")
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNX(t, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root}, "--config", cfg, "abcd", "echo hi")
	if res.ExitCode != 241 {
		t.Fatalf("expected exit 241, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "Allowed destination length is invalid") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNXIntegration_PathNotFoundExit242(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnx", "./cmd/rnx")
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNX(t, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root},
		"--config", cfg, "--w", "0.2", strings.Repeat("0", 32), "echo hi")
	skipIfReticulumUnavailableRNX(t, res.Output, res.ExitCode)
	if res.ExitCode != 242 {
		t.Fatalf("expected exit 242, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "Path not found") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNXIntegration_InteractiveQuit(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnx", "./cmd/rnx")
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNX(t, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{
		ConfigDir: cfg,
		WorkDir:   root,
		Stdin:     strings.NewReader("quit\n"),
	}, "--config", cfg, "--interactive", strings.Repeat("0", 32))
	if res.ExitCode != 0 {
		t.Fatalf("interactive quit exit=%d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "> ") {
		t.Fatalf("expected prompt in output:\n%s", res.Output)
	}
}

func TestRNXIntegration_CountFlagsExpandWithPrintIdentity(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnx", "./cmd/rnx")
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNX(t, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root}, "--config", cfg, "-vvv", "-qq", "--print-identity")
	skipIfReticulumUnavailableRNX(t, res.Output, res.ExitCode)
	if res.ExitCode != 0 {
		t.Fatalf("count-flags print-identity exit=%d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "Identity     :") || !strings.Contains(res.Output, "Listening on :") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}
