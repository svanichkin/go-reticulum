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

func writeMinimalReticulumConfigRNProbe(t *testing.T, configDir string) {
	t.Helper()
	cmdtest.WriteReticulumConfig(t, configDir, strings.Join([]string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = No",
		"",
	}, "\n"))
}

func skipIfReticulumUnavailableRNProbe(t *testing.T, out string, exitCode int) {
	t.Helper()
	if exitCode == 101 || strings.Contains(out, "Could not start Reticulum") ||
		strings.Contains(out, "operation not permitted") {
		t.Skipf("environment does not allow Reticulum startup; skipping rnprobe integration test\n%s", out)
	}
}

func writeReticulumConfigRNProbe(t *testing.T, configDir string, lines []string) {
	t.Helper()
	cmdtest.WriteReticulumConfig(t, configDir, strings.Join(lines, "\n"))
}

func freeTCPPortRNProbe(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen ephemeral tcp port: %v", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func startRNSDServiceRNProbe(t *testing.T, ctx context.Context, bin, cfg, workDir string) (*exec.Cmd, *bytes.Buffer) {
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

func waitForRNStatusSuccessRNProbe(t *testing.T, rnstatusBin, cfg, workDir string, timeout time.Duration) string {
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

func TestRNProbeIntegration_InvalidArgsExit0(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnprobe", "./cmd/rnprobe")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNProbe(t, cfg)

	// Invalid destination length should be exit code 0 (Python exit()).
	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root}, "--config", cfg, "app.aspect", "aa")
	if res.ExitCode != 0 {
		t.Fatalf("expected exit 0, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "Destination length is invalid") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNProbeIntegration_HelpAndVersion(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnprobe", "./cmd/rnprobe")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: root, WorkDir: root}, "--help")
	if res.ExitCode != 0 {
		t.Fatalf("help exit=%d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "Reticulum Probe Utility") {
		t.Fatalf("unexpected help output:\n%s", res.Output)
	}

	res = cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: root, WorkDir: root}, "--version")
	if res.ExitCode != 0 {
		t.Fatalf("version exit=%d\n%s", res.ExitCode, res.Output)
	}
	if !strings.HasPrefix(strings.TrimSpace(res.Output), "rnprobe ") {
		t.Fatalf("unexpected version output: %q", res.Output)
	}
}

func TestRNProbeIntegration_EmptyFullNameExit0(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnprobe", "./cmd/rnprobe")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNProbe(t, cfg)

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root}, "--config", cfg, "", strings.Repeat("0", 32))
	if res.ExitCode != 0 {
		t.Fatalf("expected exit 0, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "full destination name") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNProbeIntegration_InvalidDestinationHexExit0(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnprobe", "./cmd/rnprobe")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNProbe(t, cfg)

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root}, "--config", cfg, "app.aspect", strings.Repeat("z", 32))
	if res.ExitCode != 0 {
		t.Fatalf("expected exit 0, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "Invalid destination entered") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNProbeIntegration_PathTimeoutExit1(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnprobe", "./cmd/rnprobe")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNProbe(t, cfg)

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root},
		"--config", cfg, "--timeout", "0.2", "app.aspect", strings.Repeat("0", 32))
	skipIfReticulumUnavailableRNProbe(t, res.Output, res.ExitCode)
	if res.ExitCode != 1 {
		t.Fatalf("expected exit 1, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "Path request timed out") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNProbeIntegration_SharedInstancePathTimeoutExit1(t *testing.T) {
	root := t.TempDir()
	rnprobeBin := cmdtest.Build(t, root, "rnprobe", "./cmd/rnprobe")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNProbe(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = rnprobe-shared-local",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, out := startRNSDServiceRNProbe(t, ctx, rnsdBin, cfg, root)
	_ = waitForRNStatusSuccessRNProbe(t, rnstatusBin, cfg, root, 10*time.Second)

	runCtx, runCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer runCancel()
	res := cmdtest.Run(t, runCtx, rnprobeBin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root},
		"--config", cfg, "--timeout", "0.2", "app.aspect", strings.Repeat("0", 32))
	skipIfReticulumUnavailableRNProbe(t, out.String()+res.Output, res.ExitCode)
	if res.ExitCode != 1 {
		t.Fatalf("expected exit 1, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "Path request timed out") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNProbeIntegration_SharedInstanceTCPPathTimeoutExit1(t *testing.T) {
	root := t.TempDir()
	rnprobeBin := cmdtest.Build(t, root, "rnprobe", "./cmd/rnprobe")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	cfg := filepath.Join(root, "cfg")
	packetPort := freeTCPPortRNProbe(t)
	controlPort := freeTCPPortRNProbe(t)
	writeReticulumConfigRNProbe(t, cfg, []string{
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

	_, out := startRNSDServiceRNProbe(t, ctx, rnsdBin, cfg, root)
	_ = waitForRNStatusSuccessRNProbe(t, rnstatusBin, cfg, root, 10*time.Second)

	runCtx, runCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer runCancel()
	res := cmdtest.Run(t, runCtx, rnprobeBin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root},
		"--config", cfg, "--timeout", "0.2", "app.aspect", strings.Repeat("0", 32))
	skipIfReticulumUnavailableRNProbe(t, out.String()+res.Output, res.ExitCode)
	if res.ExitCode != 1 {
		t.Fatalf("expected exit 1, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "Path request timed out") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}
