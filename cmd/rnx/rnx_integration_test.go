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

func writeTwoNodeTemplateConfigRNX(t *testing.T, dstDir, templatePath string, sharedPort, controlPort, listenPort, forwardPort int) {
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

func startRNXListener(t *testing.T, ctx context.Context, bin, configDir, workDir string) (*exec.Cmd, string, *bytes.Buffer) {
	t.Helper()

	c := exec.CommandContext(ctx, bin, "--config", configDir, "--listen", "--noauth")
	c.Dir = workDir
	home := filepath.Join(configDir, ".home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	c.Env = append(os.Environ(),
		"HOME="+home,
		"USERPROFILE="+home,
	)

	var out bytes.Buffer
	stdout, err := c.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	stderr, err := c.StderrPipe()
	if err != nil {
		t.Fatalf("stderr pipe: %v", err)
	}
	if err := c.Start(); err != nil {
		t.Fatalf("start rnx listener: %v", err)
	}
	go func() { _, _ = io.Copy(&out, stderr) }()

	destCh := make(chan string, 1)
	go func() {
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			line := sc.Text()
			_, _ = out.Write([]byte(line + "\n"))
			if !strings.Contains(line, "rnx listening for commands on") {
				continue
			}
			start := strings.Index(line, "<")
			end := strings.Index(line, ">")
			if start >= 0 && end > start {
				destCh <- line[start+1 : end]
				return
			}
		}
	}()

	select {
	case dest := <-destCh:
		t.Cleanup(func() {
			if c.Process != nil {
				_ = c.Process.Signal(syscall.SIGTERM)
			}
			_ = c.Wait()
		})
		return c, dest, &out
	case <-time.After(20 * time.Second):
		_ = c.Process.Signal(syscall.SIGTERM)
		_ = c.Wait()
		t.Fatalf("rnx listener did not print destination in time; output:\n%s", out.String())
		return nil, "", nil
	}
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

func TestRNXIntegration_TwoNodeRemoteExec(t *testing.T) {
	root := t.TempDir()
	rnxBin := cmdtest.Build(t, root, "rnx", "./cmd/rnx")
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")

	basePort := freeTCPPortRNX(t)
	sharedA := freeTCPPortRNX(t)
	controlA := freeTCPPortRNX(t)
	sharedB := freeTCPPortRNX(t)
	controlB := freeTCPPortRNX(t)

	nodeADir := filepath.Join(root, "node-a")
	nodeBDir := filepath.Join(root, "node-b")
	writeTwoNodeTemplateConfigRNX(t, nodeADir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_a/config"),
		sharedA, controlA, basePort, basePort+1)
	writeTwoNodeTemplateConfigRNX(t, nodeBDir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_b/config"),
		sharedB, controlB, basePort+1, basePort)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	_, outA := startRNSDServiceRNX(t, ctx, rnsdBin, nodeADir, root)
	_, outB := startRNSDServiceRNX(t, ctx, rnsdBin, nodeBDir, root)
	_ = waitForRNStatusSuccessRNX(t, rnstatusBin, nodeADir, root, 10*time.Second)
	_ = waitForRNStatusSuccessRNX(t, rnstatusBin, nodeBDir, root, 10*time.Second)

	listener, dest, listenerOut := startRNXListener(t, ctx, rnxBin, nodeBDir, root)
	_ = listener

	runCtx, runCancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer runCancel()
	res := cmdtest.Run(t, runCtx, rnxBin, cmdtest.RunOptions{ConfigDir: nodeADir, WorkDir: root},
		"--config", nodeADir, "-d", dest, "echo hello-two-node")
	skipIfReticulumUnavailableRNX(t, outA.String()+outB.String()+listenerOut.String()+res.Output, res.ExitCode)
	if res.ExitCode != 0 {
		t.Fatalf("expected rnx success, got %d\nnodeA:\n%s\nnodeB:\n%s\nlistener:\n%s\nclient:\n%s",
			res.ExitCode, outA.String(), outB.String(), listenerOut.String(), res.Output)
	}
	if !strings.Contains(res.Output, "hello-two-node") {
		t.Fatalf("expected remote command output, got:\n%s", res.Output)
	}
}

func TestRNXIntegration_TwoNodeMirrorRemoteExitCode(t *testing.T) {
	root := t.TempDir()
	rnxBin := cmdtest.Build(t, root, "rnx", "./cmd/rnx")
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")

	basePort := freeTCPPortRNX(t)
	sharedA := freeTCPPortRNX(t)
	controlA := freeTCPPortRNX(t)
	sharedB := freeTCPPortRNX(t)
	controlB := freeTCPPortRNX(t)

	nodeADir := filepath.Join(root, "node-a")
	nodeBDir := filepath.Join(root, "node-b")
	writeTwoNodeTemplateConfigRNX(t, nodeADir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_a/config"),
		sharedA, controlA, basePort, basePort+1)
	writeTwoNodeTemplateConfigRNX(t, nodeBDir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_b/config"),
		sharedB, controlB, basePort+1, basePort)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	_, outA := startRNSDServiceRNX(t, ctx, rnsdBin, nodeADir, root)
	_, outB := startRNSDServiceRNX(t, ctx, rnsdBin, nodeBDir, root)
	_ = waitForRNStatusSuccessRNX(t, rnstatusBin, nodeADir, root, 10*time.Second)
	_ = waitForRNStatusSuccessRNX(t, rnstatusBin, nodeBDir, root, 10*time.Second)

	listener, dest, listenerOut := startRNXListener(t, ctx, rnxBin, nodeBDir, root)
	_ = listener

	runCtx, runCancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer runCancel()
	res := cmdtest.Run(t, runCtx, rnxBin, cmdtest.RunOptions{ConfigDir: nodeADir, WorkDir: root},
		"--config", nodeADir, "-m", "-d", dest, "false")
	skipIfReticulumUnavailableRNX(t, outA.String()+outB.String()+listenerOut.String()+res.Output, res.ExitCode)
	if res.ExitCode != 1 {
		t.Fatalf("expected mirrored exit code 1, got %d\nnodeA:\n%s\nnodeB:\n%s\nlistener:\n%s\nclient:\n%s",
			res.ExitCode, outA.String(), outB.String(), listenerOut.String(), res.Output)
	}
}

func TestRNXIntegration_TwoNodeRemoteExecStderrDetailed(t *testing.T) {
	root := t.TempDir()
	rnxBin := cmdtest.Build(t, root, "rnx", "./cmd/rnx")
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")

	basePort := freeTCPPortRNX(t)
	sharedA := freeTCPPortRNX(t)
	controlA := freeTCPPortRNX(t)
	sharedB := freeTCPPortRNX(t)
	controlB := freeTCPPortRNX(t)

	nodeADir := filepath.Join(root, "node-a")
	nodeBDir := filepath.Join(root, "node-b")
	writeTwoNodeTemplateConfigRNX(t, nodeADir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_a/config"),
		sharedA, controlA, basePort, basePort+1)
	writeTwoNodeTemplateConfigRNX(t, nodeBDir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_b/config"),
		sharedB, controlB, basePort+1, basePort)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	_, outA := startRNSDServiceRNX(t, ctx, rnsdBin, nodeADir, root)
	_, outB := startRNSDServiceRNX(t, ctx, rnsdBin, nodeBDir, root)
	_ = waitForRNStatusSuccessRNX(t, rnstatusBin, nodeADir, root, 10*time.Second)
	_ = waitForRNStatusSuccessRNX(t, rnstatusBin, nodeBDir, root, 10*time.Second)

	listener, dest, listenerOut := startRNXListener(t, ctx, rnxBin, nodeBDir, root)
	_ = listener

	runCtx, runCancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer runCancel()
	res := cmdtest.Run(t, runCtx, rnxBin, cmdtest.RunOptions{ConfigDir: nodeADir, WorkDir: root},
		"--config", nodeADir, "-d", "-detailed", dest, "sh -c 'echo boom >&2'")
	skipIfReticulumUnavailableRNX(t, outA.String()+outB.String()+listenerOut.String()+res.Output, res.ExitCode)
	if res.ExitCode != 0 {
		t.Fatalf("expected detailed stderr success, got %d\nnodeA:\n%s\nnodeB:\n%s\nlistener:\n%s\nclient:\n%s",
			res.ExitCode, outA.String(), outB.String(), listenerOut.String(), res.Output)
	}
	if !strings.Contains(res.Output, "boom") || !strings.Contains(res.Output, "Remote wrote") {
		t.Fatalf("expected stderr and detailed summary, got:\n%s", res.Output)
	}
}

func TestRNXIntegration_TwoNodeSequentialCommands(t *testing.T) {
	root := t.TempDir()
	rnxBin := cmdtest.Build(t, root, "rnx", "./cmd/rnx")
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")

	basePort := freeTCPPortRNX(t)
	sharedA := freeTCPPortRNX(t)
	controlA := freeTCPPortRNX(t)
	sharedB := freeTCPPortRNX(t)
	controlB := freeTCPPortRNX(t)

	nodeADir := filepath.Join(root, "node-a")
	nodeBDir := filepath.Join(root, "node-b")
	writeTwoNodeTemplateConfigRNX(t, nodeADir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_a/config"),
		sharedA, controlA, basePort, basePort+1)
	writeTwoNodeTemplateConfigRNX(t, nodeBDir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_b/config"),
		sharedB, controlB, basePort+1, basePort)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	_, outA := startRNSDServiceRNX(t, ctx, rnsdBin, nodeADir, root)
	_, outB := startRNSDServiceRNX(t, ctx, rnsdBin, nodeBDir, root)
	_ = waitForRNStatusSuccessRNX(t, rnstatusBin, nodeADir, root, 10*time.Second)
	_ = waitForRNStatusSuccessRNX(t, rnstatusBin, nodeBDir, root, 10*time.Second)

	listener, dest, listenerOut := startRNXListener(t, ctx, rnxBin, nodeBDir, root)
	_ = listener

	for _, tc := range []struct {
		cmd  string
		want string
	}{
		{cmd: "echo first-two-node", want: "first-two-node"},
		{cmd: "echo second-two-node", want: "second-two-node"},
	} {
		runCtx, runCancel := context.WithTimeout(context.Background(), 25*time.Second)
		res := cmdtest.Run(t, runCtx, rnxBin, cmdtest.RunOptions{ConfigDir: nodeADir, WorkDir: root},
			"--config", nodeADir, "-d", dest, tc.cmd)
		runCancel()
		skipIfReticulumUnavailableRNX(t, outA.String()+outB.String()+listenerOut.String()+res.Output, res.ExitCode)
		if res.ExitCode != 0 {
			t.Fatalf("command %q failed with %d\nnodeA:\n%s\nnodeB:\n%s\nlistener:\n%s\nclient:\n%s",
				tc.cmd, res.ExitCode, outA.String(), outB.String(), listenerOut.String(), res.Output)
		}
		if !strings.Contains(res.Output, tc.want) {
			t.Fatalf("expected output %q for command %q, got:\n%s", tc.want, tc.cmd, res.Output)
		}
	}
}
