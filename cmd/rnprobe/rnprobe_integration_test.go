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

func writeTwoNodeTemplateConfigRNProbe(t *testing.T, dstDir, templatePath string, sharedPort, controlPort, listenPort, forwardPort int) {
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

func extractProbeHashFromRNStatusRNProbe(t *testing.T, out string) string {
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

func TestRNProbeIntegration_MissingArgsShowsUsageExit0(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnprobe", "./cmd/rnprobe")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: root, WorkDir: root}, "--help=false")
	if res.ExitCode != 0 {
		t.Fatalf("expected exit 0, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "Usage:") || !strings.Contains(res.Output, "<full_name> <destination_hash>") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNProbeIntegration_InvalidTimeoutFlagExit2(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnprobe", "./cmd/rnprobe")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: root, WorkDir: root}, "--timeout", "not-a-float")
	if res.ExitCode != 2 {
		t.Fatalf("expected exit 2, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "invalid value") {
		t.Fatalf("unexpected output:\n%s", res.Output)
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

func TestRNProbeIntegration_TwoNodeUDPSuccess(t *testing.T) {
	cmdtest.AcquireLock(t, "integration-two-node-shared", 5*time.Minute)
	root := t.TempDir()
	rnprobeBin := cmdtest.Build(t, root, "rnprobe", "./cmd/rnprobe")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")

	basePort := freeTCPPortRNProbe(t)
	sharedA := freeTCPPortRNProbe(t)
	controlA := freeTCPPortRNProbe(t)
	sharedB := freeTCPPortRNProbe(t)
	controlB := freeTCPPortRNProbe(t)

	nodeADir := filepath.Join(root, "node-a")
	nodeBDir := filepath.Join(root, "node-b")
	writeTwoNodeTemplateConfigRNProbe(t, nodeADir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_a/config"),
		sharedA, controlA, basePort, basePort+1)
	writeTwoNodeTemplateConfigRNProbe(t, nodeBDir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_b/config"),
		sharedB, controlB, basePort+1, basePort)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmdA, outA := startRNSDServiceRNProbe(t, ctx, rnsdBin, nodeADir, root)
	defer func() {
		if cmdA.Process != nil {
			_ = cmdA.Process.Signal(syscall.SIGTERM)
		}
		_ = cmdA.Wait()
	}()
	cmdB, outB := startRNSDServiceRNProbe(t, ctx, rnsdBin, nodeBDir, root)
	defer func() {
		if cmdB.Process != nil {
			_ = cmdB.Process.Signal(syscall.SIGTERM)
		}
		_ = cmdB.Wait()
	}()

	_ = waitForRNStatusSuccessRNProbe(t, rnstatusBin, nodeADir, root, 10*time.Second)
	_ = waitForRNStatusSuccessRNProbe(t, rnstatusBin, nodeBDir, root, 10*time.Second)

	statusCtx, statusCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer statusCancel()
	statusRes := cmdtest.Run(t, statusCtx, rnstatusBin, cmdtest.RunOptions{ConfigDir: nodeBDir, WorkDir: root},
		"--config", nodeBDir, "-a")
	if statusRes.ExitCode != 0 {
		skipIfReticulumUnavailableRNProbe(t, outA.String()+outB.String()+statusRes.Output, statusRes.ExitCode)
		t.Fatalf("rnstatus on node B failed: %d\n%s", statusRes.ExitCode, statusRes.Output)
	}
	probeHash := extractProbeHashFromRNStatusRNProbe(t, statusRes.Output)

	runCtx, runCancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer runCancel()
	res := cmdtest.Run(t, runCtx, rnprobeBin, cmdtest.RunOptions{ConfigDir: nodeADir, WorkDir: root},
		"--config", nodeADir, "-n", "3", "-w", "0.2", "-t", "15", "rnstransport.probe", probeHash)
	skipIfReticulumUnavailableRNProbe(t, outA.String()+outB.String()+res.Output, res.ExitCode)
	if res.ExitCode != 0 {
		t.Fatalf("expected rnprobe success, got %d\nnodeA:\n%s\nnodeB:\n%s\nrnprobe:\n%s", res.ExitCode, outA.String(), outB.String(), res.Output)
	}
	if !strings.Contains(res.Output, "packet loss 0") {
		t.Fatalf("expected successful probe summary, got:\n%s", res.Output)
	}
}

func TestRNProbeIntegration_TwoNodeUDPRepeatedProbes(t *testing.T) {
	root := t.TempDir()
	rnprobeBin := cmdtest.Build(t, root, "rnprobe", "./cmd/rnprobe")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")

	basePort := freeTCPPortRNProbe(t)
	sharedA := freeTCPPortRNProbe(t)
	controlA := freeTCPPortRNProbe(t)
	sharedB := freeTCPPortRNProbe(t)
	controlB := freeTCPPortRNProbe(t)

	nodeADir := filepath.Join(root, "node-a")
	nodeBDir := filepath.Join(root, "node-b")
	writeTwoNodeTemplateConfigRNProbe(t, nodeADir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_a/config"),
		sharedA, controlA, basePort, basePort+1)
	writeTwoNodeTemplateConfigRNProbe(t, nodeBDir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_b/config"),
		sharedB, controlB, basePort+1, basePort)

	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()

	cmdA, outA := startRNSDServiceRNProbe(t, ctx, rnsdBin, nodeADir, root)
	defer func() {
		if cmdA.Process != nil {
			_ = cmdA.Process.Signal(syscall.SIGTERM)
		}
		_ = cmdA.Wait()
	}()
	cmdB, outB := startRNSDServiceRNProbe(t, ctx, rnsdBin, nodeBDir, root)
	defer func() {
		if cmdB.Process != nil {
			_ = cmdB.Process.Signal(syscall.SIGTERM)
		}
		_ = cmdB.Wait()
	}()

	_ = waitForRNStatusSuccessRNProbe(t, rnstatusBin, nodeADir, root, 10*time.Second)
	_ = waitForRNStatusSuccessRNProbe(t, rnstatusBin, nodeBDir, root, 10*time.Second)

	statusCtx, statusCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer statusCancel()
	statusRes := cmdtest.Run(t, statusCtx, rnstatusBin, cmdtest.RunOptions{ConfigDir: nodeBDir, WorkDir: root},
		"--config", nodeBDir, "-a")
	if statusRes.ExitCode != 0 {
		skipIfReticulumUnavailableRNProbe(t, outA.String()+outB.String()+statusRes.Output, statusRes.ExitCode)
		t.Fatalf("rnstatus on node B failed: %d\n%s", statusRes.ExitCode, statusRes.Output)
	}
	probeHash := extractProbeHashFromRNStatusRNProbe(t, statusRes.Output)

	for i := 0; i < 2; i++ {
		runCtx, runCancel := context.WithTimeout(context.Background(), 25*time.Second)
		res := cmdtest.Run(t, runCtx, rnprobeBin, cmdtest.RunOptions{ConfigDir: nodeADir, WorkDir: root},
			"--config", nodeADir, "-n", "2", "-w", "0.2", "-t", "15", "rnstransport.probe", probeHash)
		runCancel()
		skipIfReticulumUnavailableRNProbe(t, outA.String()+outB.String()+res.Output, res.ExitCode)
		if res.ExitCode != 0 {
			t.Fatalf("probe run %d failed: %d\nnodeA:\n%s\nnodeB:\n%s\nrnprobe:\n%s", i+1, res.ExitCode, outA.String(), outB.String(), res.Output)
		}
		if !strings.Contains(res.Output, "packet loss 0") {
			t.Fatalf("expected successful probe summary on run %d, got:\n%s", i+1, res.Output)
		}
	}
}
