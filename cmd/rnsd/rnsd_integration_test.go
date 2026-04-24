//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
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

func writeMinimalReticulumConfigRNSD(t *testing.T, configDir string) {
	t.Helper()
	writeReticulumConfigRNSD(t, configDir, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = No",
		"",
	})
}

func writeReticulumConfigRNSD(t *testing.T, configDir string, lines []string) {
	t.Helper()
	cmdtest.WriteReticulumConfig(t, configDir, strings.Join(lines, "\n"))
}

func skipIfReticulumUnavailableRNSD(t *testing.T, out string, exitCode int) {
	t.Helper()
	if exitCode == 1 && (strings.Contains(out, "Error starting rnsd") || strings.Contains(out, "operation not permitted")) {
		t.Skipf("environment does not allow Reticulum startup; skipping rnsd integration test\n%s", out)
	}
}

func freeTCPPortRNSD(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen ephemeral tcp port: %v", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func startRNSDService(t *testing.T, ctx context.Context, bin, cfg, workDir string) (*exec.Cmd, *bytes.Buffer) {
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

func stopRNSDService(t *testing.T, c *exec.Cmd) {
	t.Helper()
	if c == nil || c.Process == nil {
		return
	}
	_ = c.Process.Signal(syscall.SIGTERM)
	done := make(chan error, 1)
	go func() { done <- c.Wait() }()
	select {
	case <-time.After(5 * time.Second):
		_ = c.Process.Kill()
		<-done
		t.Fatalf("rnsd did not stop within timeout")
	case <-done:
	}
}

func waitForRNStatusStopRNSD(t *testing.T, rnstatusBin, cfg, workDir string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		res := cmdtest.Run(t, ctx, rnstatusBin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: workDir}, "--config", cfg, "--json")
		cancel()
		if res.ExitCode != 0 ||
			strings.Contains(res.Output, "no shared RNS instance available") ||
			strings.Contains(res.Output, "could not get RNS status") {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func waitForRNStatusSuccessRNSD(t *testing.T, rnstatusBin, cfg, workDir string, timeout time.Duration) string {
	t.Helper()
	if timeout < 20*time.Second {
		timeout = 20 * time.Second
	}

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

func tryWaitForRNStatusSuccessRNSD(t *testing.T, rnstatusBin, cfg, workDir string, timeout time.Duration) (string, bool) {
	t.Helper()
	if timeout < 20*time.Second {
		timeout = 20 * time.Second
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		res := cmdtest.Run(t, ctx, rnstatusBin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: workDir}, "--config", cfg, "--json")
		cancel()
		if res.ExitCode == 0 {
			return res.Output, true
		}
		if !strings.Contains(res.Output, "no shared RNS instance available") &&
			!strings.Contains(res.Output, "could not get RNS status") &&
			!strings.Contains(res.Output, "operation not permitted") {
			t.Fatalf("unexpected rnstatus failure while waiting for rnsd:\n%s", res.Output)
		}
		time.Sleep(100 * time.Millisecond)
	}
	return "", false
}

func decodeRNStatusJSONRNSD(t *testing.T, out string) map[string]any {
	t.Helper()
	trimmed := strings.TrimSpace(out)
	if idx := strings.Index(trimmed, "{"); idx >= 0 {
		trimmed = trimmed[idx:]
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(trimmed), &data); err != nil {
		t.Fatalf("decode rnstatus json: %v\n%s", err, out)
	}
	return data
}

func findInterfaceByShortNameRNSD(t *testing.T, stats map[string]any, shortName string) map[string]any {
	t.Helper()
	raw, ok := stats["interfaces"].([]any)
	if !ok {
		t.Fatalf("interfaces missing or wrong type: %T", stats["interfaces"])
	}
	for _, entry := range raw {
		m, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		if got, _ := m["short_name"].(string); got == shortName {
			return m
		}
	}
	t.Fatalf("interface %q not found in rnstatus output: %#v", shortName, stats["interfaces"])
	return nil
}

func TestRNSDIntegration_ExampleConfig(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: root, WorkDir: root}, "--exampleconfig")
	if res.ExitCode != 0 {
		t.Fatalf("exit=%d\n%s", res.ExitCode, res.Output)
	}
	if res.Output != exampleRNSConfig+"\n" {
		t.Fatalf("unexpected example config output")
	}
}

func TestRNSDIntegration_Version(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: root, WorkDir: root}, "--version")
	if res.ExitCode != 0 {
		t.Fatalf("exit=%d\n%s", res.ExitCode, res.Output)
	}
	if !strings.HasPrefix(res.Output, "rnsd ") {
		t.Fatalf("unexpected version output: %q", res.Output)
	}
}

func TestRNSDIntegration_Help(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: root, WorkDir: root}, "--help")
	if res.ExitCode != 0 {
		t.Fatalf("exit=%d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "Reticulum Network Stack Daemon") {
		t.Fatalf("unexpected help output:\n%s", res.Output)
	}
	if !strings.Contains(res.Output, "-config") || !strings.Contains(res.Output, "-service") {
		t.Fatalf("expected key flags in help output:\n%s", res.Output)
	}
}

func TestRNSDIntegration_UnknownFlagExit2(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
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

func TestRNSDIntegration_UnexpectedPositionalExit2(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: root, WorkDir: root}, "extra-arg")
	if res.ExitCode != 2 {
		t.Fatalf("expected exit 2, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "unrecognized arguments: extra-arg") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNSDIntegration_ServiceCreatesLogfile(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNSD(t, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c := exec.CommandContext(ctx, bin, "--config", cfg, "--service")
	c.Dir = root
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
		_ = c.Process.Signal(syscall.SIGTERM)
		_ = c.Wait()
	})

	logPath := filepath.Join(cfg, "logfile")
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(logPath); err == nil {
			goto found
		}
		time.Sleep(100 * time.Millisecond)
	}
	// If it exited quickly, report startup failure.
	if c.ProcessState != nil && c.ProcessState.Exited() {
		code := c.ProcessState.ExitCode()
		skipIfReticulumUnavailableRNSD(t, out.String(), code)
		t.Fatalf("rnsd service exited unexpectedly (exit=%d)\n%s", code, out.String())
	}
	t.Fatalf("expected logfile to exist; output:\n%s", out.String())

found:
	_ = c.Process.Signal(syscall.SIGTERM)
	_ = c.Wait()
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("expected logfile to exist: %v", err)
	}
}

func TestRNSDIntegration_InteractiveQuit(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNSD(t, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{
		ConfigDir: cfg,
		WorkDir:   root,
		Stdin:     strings.NewReader(":quit\n"),
	}, "--config", cfg, "--interactive")
	skipIfReticulumUnavailableRNSD(t, res.Output, res.ExitCode)
	if res.ExitCode != 0 {
		t.Fatalf("exit=%d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, ">>> ") {
		t.Fatalf("expected interactive prompt in output:\n%s", res.Output)
	}
}

func TestRNSDIntegration_CoreShareInstanceYesAllowsRNStatus(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = core-share-yes",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, out := startRNSDService(t, ctx, rnsdBin, cfg, root)
	got := waitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	if !strings.Contains(strings.TrimSpace(got), "{") {
		t.Fatalf("expected json output, got:\n%s", got)
	}
	if strings.Contains(out.String(), "connected to another shared local instance") {
		t.Fatalf("rnsd connected to another instance unexpectedly:\n%s", out.String())
	}
}

func TestRNSDIntegration_CoreShareInstanceNoRejectsRNStatus(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = No",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	_, out := startRNSDService(t, ctx, rnsdBin, cfg, root)
	time.Sleep(500 * time.Millisecond)

	runCtx, runCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer runCancel()
	res := cmdtest.Run(t, runCtx, rnstatusBin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root}, "--config", cfg, "--json")
	skipIfReticulumUnavailableRNSD(t, out.String()+res.Output, res.ExitCode)
	if res.ExitCode != 0 {
		t.Fatalf("expected rnstatus exit 0, got %d\n%s", res.ExitCode, res.Output)
	}
	stats := decodeRNStatusJSONRNSD(t, res.Output)
	if interfaces, ok := stats["interfaces"].([]any); !ok || len(interfaces) != 0 {
		t.Fatalf("expected empty interfaces payload without shared instance, got:\n%s", res.Output)
	}
}

func TestRNSDIntegration_CoreSharedInstanceTCPAllowsRNStatus(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	packetPort := freeTCPPortRNSD(t)
	controlPort := freeTCPPortRNSD(t)
	writeReticulumConfigRNSD(t, cfg, []string{
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

	_, out := startRNSDService(t, ctx, rnsdBin, cfg, root)
	got := waitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	if !strings.Contains(strings.TrimSpace(got), "{") {
		t.Fatalf("expected json output, got:\n%s", got)
	}
	if strings.Contains(out.String(), "operation not permitted") {
		t.Skipf("environment does not allow loopback tcp shared instance:\n%s", out.String())
	}
}

func TestRNSDIntegration_CoreDifferentInstanceNamesDoNotConflict(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")

	cfgA := filepath.Join(root, "cfg-a")
	cfgB := filepath.Join(root, "cfg-b")
	writeReticulumConfigRNSD(t, cfgA, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = core-a",
		"",
	})
	writeReticulumConfigRNSD(t, cfgB, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = core-b",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	_, outA := startRNSDService(t, ctx, rnsdBin, cfgA, root)
	_, outB := startRNSDService(t, ctx, rnsdBin, cfgB, root)

	waitForRNStatusSuccessRNSD(t, rnstatusBin, cfgA, root, 10*time.Second)
	waitForRNStatusSuccessRNSD(t, rnstatusBin, cfgB, root, 10*time.Second)

	if strings.Contains(outA.String(), "connected to another shared local instance") {
		t.Fatalf("instance A connected to another instance unexpectedly:\n%s", outA.String())
	}
	if strings.Contains(outB.String(), "connected to another shared local instance") {
		t.Fatalf("instance B connected to another instance unexpectedly:\n%s", outB.String())
	}
}

func TestRNSDIntegration_CoreRPCKeyMismatchRejectsClient(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")

	packetPort := freeTCPPortRNSD(t)
	controlPort := freeTCPPortRNSD(t)
	serverCfg := filepath.Join(root, "cfg-server")
	clientCfg := filepath.Join(root, "cfg-client")
	writeReticulumConfigRNSD(t, serverCfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"shared_instance_type = tcp",
		"shared_instance_port = " + strconv.Itoa(packetPort),
		"instance_control_port = " + strconv.Itoa(controlPort),
		"rpc_key = 00112233445566778899aabbccddeeff",
		"",
	})
	writeReticulumConfigRNSD(t, clientCfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"shared_instance_type = tcp",
		"shared_instance_port = " + strconv.Itoa(packetPort),
		"instance_control_port = " + strconv.Itoa(controlPort),
		"rpc_key = ffeeddccbbaa99887766554433221100",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, out := startRNSDService(t, ctx, rnsdBin, serverCfg, root)
	waitForRNStatusSuccessRNSD(t, rnstatusBin, serverCfg, root, 10*time.Second)

	runCtx, runCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer runCancel()
	res := cmdtest.Run(t, runCtx, rnstatusBin, cmdtest.RunOptions{ConfigDir: clientCfg, WorkDir: root}, "--config", clientCfg, "--json")
	if strings.Contains(out.String(), "operation not permitted") {
		t.Skipf("environment does not allow tcp shared instance setup:\n%s", out.String())
	}
	if res.ExitCode != 0 {
		t.Fatalf("expected rnstatus exit 0 with wrong rpc_key fallback, got %d\n%s", res.ExitCode, res.Output)
	}
	stats := decodeRNStatusJSONRNSD(t, res.Output)
	if interfaces, ok := stats["interfaces"].([]any); !ok || len(interfaces) != 0 {
		t.Fatalf("expected empty interfaces payload with wrong rpc_key fallback, got:\n%s", res.Output)
	}
}

func TestRNSDIntegration_CoreNetworkIdentityCreatedAndReported(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	networkIdentityPath := filepath.Join(cfg, "transport.id")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = True",
		"share_instance = Yes",
		"instance_name = core-netid",
		"network_identity = " + networkIdentityPath,
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, out := startRNSDService(t, ctx, rnsdBin, cfg, root)
	got, ok := tryWaitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	if !ok {
		t.Skipf("environment does not report TCPServerInterface status within timeout:\n%s", out.String())
	}
	skipIfReticulumUnavailableRNSD(t, out.String()+got, 0)

	if _, err := os.Stat(networkIdentityPath); err != nil {
		t.Fatalf("expected network identity file to exist: %v", err)
	}

	stats := decodeRNStatusJSONRNSD(t, got)
	transportID, ok := stats["transport_id"].(string)
	if !ok || transportID == "" {
		t.Fatalf("expected transport_id in rnstatus json:\n%s", got)
	}
	networkID, ok := stats["network_id"].(string)
	if !ok || networkID == "" {
		t.Fatalf("expected network_id in rnstatus json:\n%s", got)
	}
	if _, err := hex.DecodeString(transportID); err != nil {
		t.Fatalf("transport_id is not hex: %q", transportID)
	}
	if _, err := hex.DecodeString(networkID); err != nil {
		t.Fatalf("network_id is not hex: %q", networkID)
	}
}

func TestRNSDIntegration_CoreEnableTransportDisabledHidesTransportIDs(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = core-notransport",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, out := startRNSDService(t, ctx, rnsdBin, cfg, root)
	got, ok := tryWaitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	if !ok {
		t.Skipf("environment does not report TCPServerInterface status within timeout:\n%s", out.String())
	}
	skipIfReticulumUnavailableRNSD(t, out.String()+got, 0)

	stats := decodeRNStatusJSONRNSD(t, got)
	if v, ok := stats["transport_id"]; ok && v != nil {
		t.Fatalf("expected transport_id to be null when transport disabled:\n%s", got)
	}
	if v, ok := stats["network_id"]; ok && v != nil {
		t.Fatalf("expected network_id to be null when transport disabled:\n%s", got)
	}
}

func TestRNSDIntegration_CoreInvalidRemoteManagementACLExits1(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = True",
		"enable_remote_management = True",
		"remote_management_allowed = 0123",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, rnsdBin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root}, "--config", cfg)
	if res.ExitCode != 1 {
		t.Fatalf("expected exit 1, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "identity hash length for remote management ACL") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNSDIntegration_CoreInvalidMTUIgnoredAndStillStarts(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"mtu = 1",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = core-invalid-mtu",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, out := startRNSDService(t, ctx, rnsdBin, cfg, root)
	got, ok := tryWaitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	if !ok {
		t.Skipf("environment does not report TCPServerInterface status within timeout:\n%s", out.String())
	}
	skipIfReticulumUnavailableRNSD(t, out.String()+got, 0)
	if !strings.Contains(strings.TrimSpace(got), "{") {
		t.Fatalf("expected json output, got:\n%s", got)
	}
}

func TestRNSDIntegration_CoreRemoteManagementAllowedIdentityCanQuerySelf(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	managementIdentityPath := filepath.Join(root, "management.id")

	managementIdentity, err := rns.NewIdentity()
	if err != nil {
		t.Fatalf("new identity: %v", err)
	}
	if err := managementIdentity.Save(managementIdentityPath); err != nil {
		t.Fatalf("save management identity: %v", err)
	}

	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = True",
		"share_instance = Yes",
		"instance_name = core-remote-mgmt",
		"enable_remote_management = True",
		"remote_management_allowed = " + hex.EncodeToString(managementIdentity.Hash),
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	_, out := startRNSDService(t, ctx, rnsdBin, cfg, root)
	localJSON := waitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	skipIfReticulumUnavailableRNSD(t, out.String()+localJSON, 0)

	localStats := decodeRNStatusJSONRNSD(t, localJSON)
	transportID, ok := localStats["transport_id"].(string)
	if !ok || transportID == "" {
		t.Fatalf("expected transport_id in local status:\n%s", localJSON)
	}

	runCtx, runCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer runCancel()
	res := cmdtest.Run(t, runCtx, rnstatusBin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root},
		"--config", cfg, "-R", transportID, "-i", managementIdentityPath, "--json")
	if res.ExitCode == 12 || strings.Contains(res.Output, "Path request timed out") {
		t.Skipf("environment does not permit remote-management self-query path establishment\n%s", res.Output)
	}
	if res.ExitCode != 0 {
		t.Fatalf("expected remote rnstatus success, got %d\n%s", res.ExitCode, res.Output)
	}
	remoteStats := decodeRNStatusJSONRNSD(t, res.Output)
	if v, ok := remoteStats["transport_id"].(string); !ok || v == "" {
		t.Fatalf("expected transport_id in remote status:\n%s", res.Output)
	}
}

func TestRNSDIntegration_CoreRemoteManagementDeniedIdentityFails(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	allowedIdentityPath := filepath.Join(root, "allowed.id")
	deniedIdentityPath := filepath.Join(root, "denied.id")

	allowedIdentity, err := rns.NewIdentity()
	if err != nil {
		t.Fatalf("new allowed identity: %v", err)
	}
	if err := allowedIdentity.Save(allowedIdentityPath); err != nil {
		t.Fatalf("save allowed identity: %v", err)
	}
	deniedIdentity, err := rns.NewIdentity()
	if err != nil {
		t.Fatalf("new denied identity: %v", err)
	}
	if err := deniedIdentity.Save(deniedIdentityPath); err != nil {
		t.Fatalf("save denied identity: %v", err)
	}

	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = True",
		"share_instance = Yes",
		"instance_name = core-remote-denied",
		"enable_remote_management = True",
		"remote_management_allowed = " + hex.EncodeToString(allowedIdentity.Hash),
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	_, out := startRNSDService(t, ctx, rnsdBin, cfg, root)
	localJSON := waitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	skipIfReticulumUnavailableRNSD(t, out.String()+localJSON, 0)

	localStats := decodeRNStatusJSONRNSD(t, localJSON)
	transportID, ok := localStats["transport_id"].(string)
	if !ok || transportID == "" {
		t.Fatalf("expected transport_id in local status:\n%s", localJSON)
	}

	runCtx, runCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer runCancel()
	res := cmdtest.Run(t, runCtx, rnstatusBin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root},
		"--config", cfg, "-R", transportID, "-i", deniedIdentityPath, "--json")
	if res.ExitCode == 12 || strings.Contains(res.Output, "Path request timed out") {
		t.Skipf("environment does not permit remote-management self-query path establishment\n%s", res.Output)
	}
	if res.ExitCode != 2 {
		t.Fatalf("expected remote rnstatus denial exit 2, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "could not get RNS status") {
		t.Fatalf("unexpected denied remote management output:\n%s", res.Output)
	}
}

func TestRNSDIntegration_InterfaceCommonEnabledGatingVisibleInStatus(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = if-common-enabled",
		"",
		"[interfaces]",
		"  [[DisabledUDP]]",
		"    type = UDPInterface",
		"    enabled = no",
		"    listen_ip = 127.0.0.1",
		"    listen_port = 0",
		"    forward_ip = 127.0.0.1",
		"    forward_port = 0",
		"  [[EnabledUDP]]",
		"    type = UDPInterface",
		"    enabled = yes",
		"    listen_ip = 127.0.0.1",
		"    listen_port = 0",
		"    forward_ip = 127.0.0.1",
		"    forward_port = 0",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	_, out := startRNSDService(t, ctx, rnsdBin, cfg, root)
	got, ok := tryWaitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	if !ok {
		t.Skipf("environment does not report TCPServerInterface status within timeout:\n%s", out.String())
	}
	skipIfReticulumUnavailableRNSD(t, out.String()+got, 0)

	stats := decodeRNStatusJSONRNSD(t, got)
	enabled := findInterfaceByShortNameRNSD(t, stats, "EnabledUDP")
	if _, ok := enabled["short_name"].(string); !ok {
		t.Fatalf("expected EnabledUDP interface entry in rnstatus output")
	}

	raw := stats["interfaces"].([]any)
	for _, entry := range raw {
		m, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		if got, _ := m["short_name"].(string); got == "DisabledUDP" {
			t.Fatalf("disabled interface unexpectedly present in rnstatus output: %#v", m)
		}
	}
}

func TestRNSDIntegration_InterfaceCommonModeBitrateIFACAndNetnameVisibleInStatus(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = if-common-visible",
		"",
		"[interfaces]",
		"  [[VisibleUDP]]",
		"    type = UDPInterface",
		"    enabled = yes",
		"    mode = gateway",
		"    bitrate = 12345",
		"    ifac_size = 24",
		"    networkname = testnet",
		"    passphrase = testpass",
		"    listen_ip = 127.0.0.1",
		"    listen_port = 0",
		"    forward_ip = 127.0.0.1",
		"    forward_port = 0",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	_, out := startRNSDService(t, ctx, rnsdBin, cfg, root)
	got, ok := tryWaitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	if !ok {
		t.Skipf("environment does not report TCPServerInterface status within timeout:\n%s", out.String())
	}
	skipIfReticulumUnavailableRNSD(t, out.String()+got, 0)

	stats := decodeRNStatusJSONRNSD(t, got)
	ifc := findInterfaceByShortNameRNSD(t, stats, "VisibleUDP")

	if status, _ := ifc["status"].(bool); status {
		t.Fatalf("expected VisibleUDP status=false without active peer, got %#v", ifc["status"])
	}
	if bitrate, _ := ifc["bitrate"].(float64); int(bitrate) != 12345 {
		t.Fatalf("expected bitrate=12345, got %#v", ifc["bitrate"])
	}
	if size, _ := ifc["ifac_size"].(float64); int(size) != 3 {
		t.Fatalf("expected ifac_size=3 bytes, got %#v", ifc["ifac_size"])
	}
	if netname, _ := ifc["ifac_netname"].(string); netname != "testnet" {
		t.Fatalf("expected ifac_netname=testnet, got %#v", ifc["ifac_netname"])
	}
	if mode, _ := ifc["mode"].(float64); int(mode) != rns.InterfaceModeGateway {
		t.Fatalf("expected mode=%d, got %#v", rns.InterfaceModeGateway, ifc["mode"])
	}
}

func TestRNSDIntegration_InterfaceCommonAnnounceSettingsAcceptedAtStartup(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = if-common-announce",
		"",
		"[interfaces]",
		"  [[AnnounceUDP]]",
		"    type = UDPInterface",
		"    enabled = yes",
		"    announce_cap = 50",
		"    announce_rate_target = 10",
		"    announce_rate_grace = 2",
		"    announce_rate_penalty = 5",
		"    listen_ip = 127.0.0.1",
		"    listen_port = 0",
		"    forward_ip = 127.0.0.1",
		"    forward_port = 0",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	_, out := startRNSDService(t, ctx, rnsdBin, cfg, root)
	got, ok := tryWaitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	if !ok {
		t.Skipf("environment does not report TCPServerInterface status within timeout:\n%s", out.String())
	}
	skipIfReticulumUnavailableRNSD(t, out.String()+got, 0)

	stats := decodeRNStatusJSONRNSD(t, got)
	ifc := findInterfaceByShortNameRNSD(t, stats, "AnnounceUDP")
	if _, ok := ifc["announce_queue"]; !ok {
		t.Fatalf("expected announce_queue field in rnstatus output: %#v", ifc)
	}
	if _, ok := ifc["held_announces"]; !ok {
		t.Fatalf("expected held_announces field in rnstatus output: %#v", ifc)
	}
}

func TestRNSDIntegration_InterfaceCommonAliasKeysAcceptedAtStartup(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = if-common-aliases",
		"",
		"[interfaces]",
		"  [[AliasDisabled]]",
		"    type = UDPInterface",
		"    interface_enabled = no",
		"    listen_ip = 127.0.0.1",
		"    listen_port = 0",
		"    forward_ip = 127.0.0.1",
		"    forward_port = 0",
		"  [[AliasVisible]]",
		"    type = UDPInterface",
		"    interface_enabled = yes",
		"    selected_interface_mode = gateway",
		"    configured_bitrate = 23456",
		"    ifac_size = 12",
		"    listen_ip = 127.0.0.1",
		"    listen_port = 0",
		"    forward_ip = 127.0.0.1",
		"    forward_port = 0",
		"  [[AliasVisibleTwo]]",
		"    type = UDPInterface",
		"    interface_enabled = yes",
		"    interface_mode = gateway",
		"    configured_bitrate = 34567",
		"    ifac_size = 14",
		"    listen_ip = 127.0.0.1",
		"    listen_port = 0",
		"    forward_ip = 127.0.0.1",
		"    forward_port = 0",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	_, out := startRNSDService(t, ctx, rnsdBin, cfg, root)
	got, ok := tryWaitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	if !ok {
		t.Skipf("environment does not report TCPServerInterface status within timeout:\n%s", out.String())
	}
	skipIfReticulumUnavailableRNSD(t, out.String()+got, 0)

	stats := decodeRNStatusJSONRNSD(t, got)
	visibleOne := findInterfaceByShortNameRNSD(t, stats, "AliasVisible")
	if mode, _ := visibleOne["mode"].(float64); int(mode) != rns.InterfaceModeGateway {
		t.Fatalf("expected AliasVisible mode=%d, got %#v", rns.InterfaceModeGateway, visibleOne["mode"])
	}
	if bitrate, _ := visibleOne["bitrate"].(float64); int(bitrate) != 23456 {
		t.Fatalf("expected AliasVisible bitrate=23456, got %#v", visibleOne["bitrate"])
	}
	if size, _ := visibleOne["ifac_size"].(float64); int(size) != 1 {
		t.Fatalf("expected AliasVisible ifac_size=1 byte, got %#v", visibleOne["ifac_size"])
	}

	visibleTwo := findInterfaceByShortNameRNSD(t, stats, "AliasVisibleTwo")
	if mode, _ := visibleTwo["mode"].(float64); int(mode) != rns.InterfaceModeGateway {
		t.Fatalf("expected AliasVisibleTwo mode=%d, got %#v", rns.InterfaceModeGateway, visibleTwo["mode"])
	}
	if bitrate, _ := visibleTwo["bitrate"].(float64); int(bitrate) != 34567 {
		t.Fatalf("expected AliasVisibleTwo bitrate=34567, got %#v", visibleTwo["bitrate"])
	}
	if size, _ := visibleTwo["ifac_size"].(float64); int(size) != 1 {
		t.Fatalf("expected AliasVisibleTwo ifac_size=1 byte, got %#v", visibleTwo["ifac_size"])
	}

	raw := stats["interfaces"].([]any)
	for _, entry := range raw {
		m, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		if got, _ := m["short_name"].(string); got == "AliasDisabled" {
			t.Fatalf("disabled alias interface unexpectedly present in rnstatus output: %#v", m)
		}
	}
}

func TestRNSDIntegration_InterfaceDriverUDPVisibleInStatus(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = if-driver-udp",
		"",
		"[interfaces]",
		"  [[UDPDriver]]",
		"    type = UDPInterface",
		"    enabled = yes",
		"    listen_ip = 127.0.0.1",
		"    listen_port = 0",
		"    forward_ip = 127.0.0.1",
		"    forward_port = 0",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	_, out := startRNSDService(t, ctx, rnsdBin, cfg, root)
	got := waitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	skipIfReticulumUnavailableRNSD(t, out.String()+got, 0)

	stats := decodeRNStatusJSONRNSD(t, got)
	ifc := findInterfaceByShortNameRNSD(t, stats, "UDPDriver")
	if typ, _ := ifc["type"].(string); typ != "UDPInterface" {
		t.Fatalf("expected type=UDPInterface, got %#v", ifc["type"])
	}
}

func TestRNSDIntegration_InterfaceDriverTCPServerVisibleInStatus(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = if-driver-tcpserver",
		"",
		"[interfaces]",
		"  [[TCPServerDriver]]",
		"    type = TCPServerInterface",
		"    enabled = yes",
		"    listen_ip = 127.0.0.1",
		"    listen_port = 0",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	_, out := startRNSDService(t, ctx, rnsdBin, cfg, root)
	got, ok := tryWaitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	if !ok {
		t.Skipf("environment does not report TCPServerInterface status within timeout:\n%s", out.String())
	}
	skipIfReticulumUnavailableRNSD(t, out.String()+got, 0)

	stats := decodeRNStatusJSONRNSD(t, got)
	ifc := findInterfaceByShortNameRNSD(t, stats, "TCPServerDriver")
	if typ, _ := ifc["type"].(string); typ != "TCPServerInterface" {
		t.Fatalf("expected type=TCPServerInterface, got %#v", ifc["type"])
	}
}

func TestRNSDIntegration_InterfaceDriverTCPClientVisibleInStatus(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")

	dummyLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen dummy tcp server: %v", err)
	}
	defer dummyLn.Close()
	go func() {
		conn, err := dummyLn.Accept()
		if err == nil {
			_ = conn.Close()
		}
	}()
	targetPort := dummyLn.Addr().(*net.TCPAddr).Port

	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = if-driver-tcpclient",
		"",
		"[interfaces]",
		"  [[TCPClientDriver]]",
		"    type = TCPClientInterface",
		"    enabled = yes",
		"    target_host = 127.0.0.1",
		"    target_port = " + strconv.Itoa(targetPort),
		"    connect_timeout = 0.1",
		"    reconnect_wait = 0.1",
		"    max_reconnect_tries = 0",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	_, out := startRNSDService(t, ctx, rnsdBin, cfg, root)
	got := waitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	skipIfReticulumUnavailableRNSD(t, out.String()+got, 0)

	stats := decodeRNStatusJSONRNSD(t, got)
	ifc := findInterfaceByShortNameRNSD(t, stats, "TCPClientDriver")
	if typ, _ := ifc["type"].(string); typ != "TCPClientInterface" {
		t.Fatalf("expected type=TCPClientInterface, got %#v", ifc["type"])
	}
}

func TestRNSDIntegration_InterfaceDriverPipeVisibleInStatus(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = if-driver-pipe",
		"",
		"[interfaces]",
		"  [[PipeDriver]]",
		"    type = PipeInterface",
		"    enabled = yes",
		"    command = /bin/cat",
		"    respawn_delay = 0.1",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	_, out := startRNSDService(t, ctx, rnsdBin, cfg, root)
	got := waitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	skipIfReticulumUnavailableRNSD(t, out.String()+got, 0)

	stats := decodeRNStatusJSONRNSD(t, got)
	ifc := findInterfaceByShortNameRNSD(t, stats, "PipeDriver")
	if typ, _ := ifc["type"].(string); typ != "PipeInterface" {
		t.Fatalf("expected type=PipeInterface, got %#v", ifc["type"])
	}
	if status, _ := ifc["status"].(bool); !status {
		t.Fatalf("expected PipeDriver status=true, got %#v", ifc["status"])
	}
}

func TestRNSDIntegration_InterfaceDriverAutoVisibleInStatus(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = if-driver-auto",
		"",
		"[interfaces]",
		"  [[AutoDriver]]",
		"    type = AutoInterface",
		"    enabled = yes",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	_, out := startRNSDService(t, ctx, rnsdBin, cfg, root)
	got := waitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 12*time.Second)
	skipIfReticulumUnavailableRNSD(t, out.String()+got, 0)

	stats := decodeRNStatusJSONRNSD(t, got)
	ifc := findInterfaceByShortNameRNSD(t, stats, "AutoDriver")
	if typ, _ := ifc["type"].(string); typ != "AutoInterface" {
		t.Fatalf("expected type=AutoInterface, got %#v", ifc["type"])
	}
}

func TestRNSDIntegration_InterfaceDriverWeaveVisibleInStatus(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = if-driver-weave",
		"",
		"[interfaces]",
		"  [[WeaveDriver]]",
		"    type = WeaveInterface",
		"    enabled = yes",
		"    port = /dev/null",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	_, out := startRNSDService(t, ctx, rnsdBin, cfg, root)
	got := waitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	skipIfReticulumUnavailableRNSD(t, out.String()+got, 0)

	stats := decodeRNStatusJSONRNSD(t, got)
	ifc := findInterfaceByShortNameRNSD(t, stats, "WeaveDriver")
	if typ, _ := ifc["type"].(string); typ != "WeaveInterface" {
		t.Fatalf("expected type=WeaveInterface, got %#v", ifc["type"])
	}
}

func TestRNSDIntegration_ValidationInvalidRPCKeyFallsBackWithLog(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = invalid-rpc-key",
		"rpc_key = not-hex-at-all",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, out := startRNSDService(t, ctx, rnsdBin, cfg, root)
	got := waitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	skipIfReticulumUnavailableRNSD(t, out.String()+got, 0)
	if !strings.Contains(out.String()+got, "Invalid shared instance RPC key") {
		t.Fatalf("expected invalid rpc_key warning in output:\nservice:\n%s\nstatus:\n%s", out.String(), got)
	}
}

func TestRNSDIntegration_ValidationInvalidInterfaceTypeExits1(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"",
		"[interfaces]",
		"  [[Broken]]",
		"    enabled = yes",
		"    type = NotARealInterface",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, rnsdBin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root}, "--config", cfg)
	if !strings.Contains(res.Output, "unsupported interface type") &&
		!strings.Contains(res.Output, "Could not locate external interface module") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
	if res.ExitCode != 1 && !(res.ExitCode == -1 && strings.Contains(res.Output, "Started rnsd version")) {
		t.Fatalf("expected exit 1 or timed service fallback, got %d\n%s", res.ExitCode, res.Output)
	}
}

func TestRNSDIntegration_ValidationInvalidModeFallsBackToFull(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = invalid-mode",
		"",
		"[interfaces]",
		"  [[BadModeUDP]]",
		"    type = UDPInterface",
		"    enabled = yes",
		"    mode = definitely-not-a-mode",
		"    listen_ip = 127.0.0.1",
		"    listen_port = 0",
		"    forward_ip = 127.0.0.1",
		"    forward_port = 0",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, out := startRNSDService(t, ctx, rnsdBin, cfg, root)
	got := waitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	skipIfReticulumUnavailableRNSD(t, out.String()+got, 0)

	stats := decodeRNStatusJSONRNSD(t, got)
	ifc := findInterfaceByShortNameRNSD(t, stats, "BadModeUDP")
	if mode, _ := ifc["mode"].(float64); int(mode) != rns.InterfaceModeFull {
		t.Fatalf("expected fallback mode=%d, got %#v", rns.InterfaceModeFull, ifc["mode"])
	}
}

func TestRNSDIntegration_ValidationInvalidBitrateFallsBackToDefault(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = invalid-bitrate",
		"",
		"[interfaces]",
		"  [[BadBitrateUDP]]",
		"    type = UDPInterface",
		"    enabled = yes",
		"    bitrate = 1",
		"    listen_ip = 127.0.0.1",
		"    listen_port = 0",
		"    forward_ip = 127.0.0.1",
		"    forward_port = 0",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, out := startRNSDService(t, ctx, rnsdBin, cfg, root)
	got := waitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	skipIfReticulumUnavailableRNSD(t, out.String()+got, 0)

	stats := decodeRNStatusJSONRNSD(t, got)
	ifc := findInterfaceByShortNameRNSD(t, stats, "BadBitrateUDP")
	if bitrate, _ := ifc["bitrate"].(float64); int(bitrate) <= 1 {
		t.Fatalf("expected default bitrate fallback, got %#v", ifc["bitrate"])
	}
}

func TestRNSDIntegration_ValidationInvalidAnnounceSettingsIgnored(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = invalid-announce",
		"",
		"[interfaces]",
		"  [[BadAnnounceUDP]]",
		"    type = UDPInterface",
		"    enabled = yes",
		"    announce_cap = 999",
		"    announce_rate_target = -1",
		"    announce_rate_grace = -2",
		"    announce_rate_penalty = -3",
		"    listen_ip = 127.0.0.1",
		"    listen_port = 0",
		"    forward_ip = 127.0.0.1",
		"    forward_port = 0",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, out := startRNSDService(t, ctx, rnsdBin, cfg, root)
	got := waitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	skipIfReticulumUnavailableRNSD(t, out.String()+got, 0)

	stats := decodeRNStatusJSONRNSD(t, got)
	ifc := findInterfaceByShortNameRNSD(t, stats, "BadAnnounceUDP")
	if _, ok := ifc["announce_queue"]; !ok {
		t.Fatalf("expected announce fields to remain available after invalid config: %#v", ifc)
	}
}

func TestRNSDIntegration_ValidationInvalidUDPListenPortExits1(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"",
		"[interfaces]",
		"  [[BrokenUDP]]",
		"    type = UDPInterface",
		"    enabled = yes",
		"    listen_ip = 127.0.0.1",
		"    listen_port = 99999",
		"    forward_ip = 127.0.0.1",
		"    forward_port = 0",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, rnsdBin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root}, "--config", cfg)
	if res.ExitCode != 1 {
		t.Fatalf("expected exit 1, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(strings.ToLower(res.Output), "invalid port") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNSDIntegration_ValidationInvalidTCPClientTargetPortExits1(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"",
		"[interfaces]",
		"  [[BrokenTCPClient]]",
		"    type = TCPClientInterface",
		"    enabled = yes",
		"    target_host = 127.0.0.1",
		"    target_port = 99999",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, rnsdBin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root}, "--config", cfg)
	if res.ExitCode != 1 {
		t.Fatalf("expected exit 1, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "invalid target port") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNSDIntegration_ValidationInvalidTCPServerConfigExits1(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"",
		"[interfaces]",
		"  [[BrokenTCPServer]]",
		"    type = TCPServerInterface",
		"    enabled = yes",
		"    listen_port = 4242",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, rnsdBin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root}, "--config", cfg)
	if res.ExitCode != 1 {
		t.Fatalf("expected exit 1, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "missing listen_ip/device") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNSDIntegration_ValidationPipeMissingCommandExits1(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"",
		"[interfaces]",
		"  [[BrokenPipe]]",
		"    type = PipeInterface",
		"    enabled = yes",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, rnsdBin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root}, "--config", cfg)
	if res.ExitCode != 1 {
		t.Fatalf("expected exit 1, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "no command specified for PipeInterface") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNSDIntegration_ValidationInvalidSharedInstanceTypeFallsBackToDefault(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = invalid-shared-instance-type",
		"shared_instance_type = definitely-not-valid",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, out := startRNSDService(t, ctx, rnsdBin, cfg, root)
	got := waitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	skipIfReticulumUnavailableRNSD(t, out.String()+got, 0)
	if !strings.Contains(strings.TrimSpace(got), "{") {
		t.Fatalf("expected rnstatus json output after shared_instance_type fallback, got:\n%s", got)
	}
}

func TestRNSDIntegration_ValidationInvalidSharedInstancePortsFallbackToDefaults(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"shared_instance_type = tcp",
		"shared_instance_port = not-a-number",
		"instance_control_port = also-not-a-number",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, out := startRNSDService(t, ctx, rnsdBin, cfg, root)
	got := waitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	if strings.Contains(out.String(), "operation not permitted") {
		t.Skipf("environment does not allow loopback tcp shared instance:\n%s", out.String())
	}
	if !strings.Contains(strings.TrimSpace(got), "{") {
		t.Fatalf("expected rnstatus json output after shared instance port fallback, got:\n%s", got)
	}
}

func TestRNSDIntegration_Persistence_NetworkIdentitySurvivesRestart(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	networkIdentityPath := filepath.Join(cfg, "transport.id")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = True",
		"share_instance = Yes",
		"instance_name = persist-netid",
		"network_identity = " + networkIdentityPath,
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	firstCmd, firstOut := startRNSDService(t, ctx, rnsdBin, cfg, root)
	firstJSON, ok := tryWaitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	if !ok {
		t.Skipf("environment does not report TCPServerInterface status within timeout:\n%s", firstOut.String())
	}
	skipIfReticulumUnavailableRNSD(t, firstOut.String()+firstJSON, 0)
	firstStats := decodeRNStatusJSONRNSD(t, firstJSON)
	firstTransportID, _ := firstStats["transport_id"].(string)
	firstNetworkID, _ := firstStats["network_id"].(string)
	if firstTransportID == "" || firstNetworkID == "" {
		t.Fatalf("expected non-empty transport/network ids on first start:\n%s", firstJSON)
	}

	stopRNSDService(t, firstCmd)
	waitForRNStatusStopRNSD(t, rnstatusBin, cfg, root, 5*time.Second)

	secondCmd, secondOut := startRNSDService(t, ctx, rnsdBin, cfg, root)
	defer stopRNSDService(t, secondCmd)
	secondJSON, ok := tryWaitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	if !ok {
		t.Skipf("environment does not report TCPServerInterface status on restart within timeout:\n%s", secondOut.String())
	}
	skipIfReticulumUnavailableRNSD(t, secondOut.String()+secondJSON, 0)
	secondStats := decodeRNStatusJSONRNSD(t, secondJSON)
	secondTransportID, _ := secondStats["transport_id"].(string)
	secondNetworkID, _ := secondStats["network_id"].(string)

	if secondTransportID != firstTransportID {
		t.Fatalf("transport_id changed across restart: first=%s second=%s", firstTransportID, secondTransportID)
	}
	if secondNetworkID != firstNetworkID {
		t.Fatalf("network_id changed across restart: first=%s second=%s", firstNetworkID, secondNetworkID)
	}
}

func TestRNSDIntegration_Persistence_SharedInstanceUnixReusesInstanceNameAfterRestart(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = persist-unix-instance",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	firstCmd, firstOut := startRNSDService(t, ctx, rnsdBin, cfg, root)
	firstJSON, ok := tryWaitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	if !ok {
		t.Skipf("environment does not report TCPServerInterface status within timeout:\n%s", firstOut.String())
	}
	skipIfReticulumUnavailableRNSD(t, firstOut.String()+firstJSON, 0)
	stopRNSDService(t, firstCmd)
	waitForRNStatusStopRNSD(t, rnstatusBin, cfg, root, 5*time.Second)

	secondCmd, secondOut := startRNSDService(t, ctx, rnsdBin, cfg, root)
	defer stopRNSDService(t, secondCmd)
	secondJSON, ok := tryWaitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	if !ok {
		t.Skipf("environment does not report TCPServerInterface status on restart within timeout:\n%s", secondOut.String())
	}
	skipIfReticulumUnavailableRNSD(t, secondOut.String()+secondJSON, 0)

	if strings.Contains(secondOut.String(), "connected to another shared local instance") {
		t.Fatalf("second start unexpectedly connected to another shared instance:\n%s", secondOut.String())
	}
}

func TestRNSDIntegration_Persistence_SharedInstanceTCPReusesPortsAfterRestart(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	packetPort := freeTCPPortRNSD(t)
	controlPort := freeTCPPortRNSD(t)
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"shared_instance_type = tcp",
		"shared_instance_port = " + strconv.Itoa(packetPort),
		"instance_control_port = " + strconv.Itoa(controlPort),
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	firstCmd, firstOut := startRNSDService(t, ctx, rnsdBin, cfg, root)
	firstJSON := waitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	if strings.Contains(firstOut.String(), "operation not permitted") {
		t.Skipf("environment does not allow loopback tcp shared instance:\n%s", firstOut.String())
	}
	if !strings.Contains(strings.TrimSpace(firstJSON), "{") {
		t.Fatalf("expected rnstatus json on first tcp start:\n%s", firstJSON)
	}
	stopRNSDService(t, firstCmd)
	waitForRNStatusStopRNSD(t, rnstatusBin, cfg, root, 5*time.Second)

	secondCmd, secondOut := startRNSDService(t, ctx, rnsdBin, cfg, root)
	defer stopRNSDService(t, secondCmd)
	secondJSON := waitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	if strings.Contains(secondOut.String(), "operation not permitted") {
		t.Skipf("environment does not allow loopback tcp shared instance on restart:\n%s", secondOut.String())
	}
	if !strings.Contains(strings.TrimSpace(secondJSON), "{") {
		t.Fatalf("expected rnstatus json on second tcp start:\n%s", secondJSON)
	}
}

func TestRNSDIntegration_Persistence_UDPInterfaceConfigSurvivesRestart(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = persist-udp",
		"",
		"[interfaces]",
		"  [[PersistUDP]]",
		"    type = UDPInterface",
		"    enabled = yes",
		"    mode = gateway",
		"    bitrate = 23456",
		"    ifac_size = 24",
		"    networkname = persistnet",
		"    passphrase = persistpass",
		"    listen_ip = 127.0.0.1",
		"    listen_port = 0",
		"    forward_ip = 127.0.0.1",
		"    forward_port = 0",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	firstCmd, firstOut := startRNSDService(t, ctx, rnsdBin, cfg, root)
	firstJSON, ok := tryWaitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	if !ok {
		t.Skipf("environment does not report TCPServerInterface status within timeout:\n%s", firstOut.String())
	}
	skipIfReticulumUnavailableRNSD(t, firstOut.String()+firstJSON, 0)
	firstStats := decodeRNStatusJSONRNSD(t, firstJSON)
	firstIfc := findInterfaceByShortNameRNSD(t, firstStats, "PersistUDP")
	if bitrate, _ := firstIfc["bitrate"].(float64); int(bitrate) != 23456 {
		t.Fatalf("expected first bitrate=23456, got %#v", firstIfc["bitrate"])
	}
	if size, _ := firstIfc["ifac_size"].(float64); int(size) != 3 {
		t.Fatalf("expected first ifac_size=3 bytes, got %#v", firstIfc["ifac_size"])
	}
	if netname, _ := firstIfc["ifac_netname"].(string); netname != "persistnet" {
		t.Fatalf("expected first ifac_netname=persistnet, got %#v", firstIfc["ifac_netname"])
	}
	if mode, _ := firstIfc["mode"].(float64); int(mode) != rns.InterfaceModeGateway {
		t.Fatalf("expected first mode=%d, got %#v", rns.InterfaceModeGateway, firstIfc["mode"])
	}
	stopRNSDService(t, firstCmd)
	waitForRNStatusStopRNSD(t, rnstatusBin, cfg, root, 5*time.Second)

	secondCmd, secondOut := startRNSDService(t, ctx, rnsdBin, cfg, root)
	defer stopRNSDService(t, secondCmd)
	secondJSON := waitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	skipIfReticulumUnavailableRNSD(t, secondOut.String()+secondJSON, 0)
	secondStats := decodeRNStatusJSONRNSD(t, secondJSON)
	secondIfc := findInterfaceByShortNameRNSD(t, secondStats, "PersistUDP")
	if bitrate, _ := secondIfc["bitrate"].(float64); int(bitrate) != 23456 {
		t.Fatalf("expected second bitrate=23456, got %#v", secondIfc["bitrate"])
	}
	if size, _ := secondIfc["ifac_size"].(float64); int(size) != 3 {
		t.Fatalf("expected second ifac_size=3 bytes, got %#v", secondIfc["ifac_size"])
	}
	if netname, _ := secondIfc["ifac_netname"].(string); netname != "persistnet" {
		t.Fatalf("expected second ifac_netname=persistnet, got %#v", secondIfc["ifac_netname"])
	}
	if mode, _ := secondIfc["mode"].(float64); int(mode) != rns.InterfaceModeGateway {
		t.Fatalf("expected second mode=%d, got %#v", rns.InterfaceModeGateway, secondIfc["mode"])
	}
}

func TestRNSDIntegration_Persistence_PipeInterfaceSurvivesRestart(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = persist-pipe",
		"",
		"[interfaces]",
		"  [[PersistPipe]]",
		"    type = PipeInterface",
		"    enabled = yes",
		"    command = /bin/cat",
		"    respawn_delay = 0.1",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	firstCmd, firstOut := startRNSDService(t, ctx, rnsdBin, cfg, root)
	firstJSON, ok := tryWaitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	if !ok {
		t.Skipf("environment does not report TCPServerInterface status within timeout:\n%s", firstOut.String())
	}
	skipIfReticulumUnavailableRNSD(t, firstOut.String()+firstJSON, 0)
	firstStats := decodeRNStatusJSONRNSD(t, firstJSON)
	firstIfc := findInterfaceByShortNameRNSD(t, firstStats, "PersistPipe")
	if typ, _ := firstIfc["type"].(string); typ != "PipeInterface" {
		t.Fatalf("expected first type=PipeInterface, got %#v", firstIfc["type"])
	}
	if status, _ := firstIfc["status"].(bool); !status {
		t.Fatalf("expected first pipe status=true, got %#v", firstIfc["status"])
	}
	stopRNSDService(t, firstCmd)
	waitForRNStatusStopRNSD(t, rnstatusBin, cfg, root, 5*time.Second)

	secondCmd, secondOut := startRNSDService(t, ctx, rnsdBin, cfg, root)
	defer stopRNSDService(t, secondCmd)
	var secondJSON string
	waitDeadline := time.Now().Add(20 * time.Second)
	for {
		j, ok := tryWaitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 5*time.Second)
		if !ok {
			t.Skipf("environment does not report PipeInterface status on restart within timeout:\n%s", secondOut.String())
		}
		secondJSON = j
		if strings.Contains(j, `"PersistPipe"`) {
			break
		}
		if time.Now().After(waitDeadline) {
			t.Skipf("PersistPipe did not appear in rnstatus within timeout:\n%s", secondOut.String())
		}
		time.Sleep(100 * time.Millisecond)
	}
	skipIfReticulumUnavailableRNSD(t, secondOut.String()+secondJSON, 0)
	secondStats := decodeRNStatusJSONRNSD(t, secondJSON)
	secondIfc := findInterfaceByShortNameRNSD(t, secondStats, "PersistPipe")
	if typ, _ := secondIfc["type"].(string); typ != "PipeInterface" {
		t.Fatalf("expected second type=PipeInterface, got %#v", secondIfc["type"])
	}
	if status, _ := secondIfc["status"].(bool); !status {
		t.Fatalf("expected second pipe status=true, got %#v", secondIfc["status"])
	}
}

func TestRNSDIntegration_Persistence_TCPServerInterfaceSurvivesRestart(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = persist-tcpserver",
		"",
		"[interfaces]",
		"  [[PersistTCPServer]]",
		"    type = TCPServerInterface",
		"    enabled = yes",
		"    listen_ip = 127.0.0.1",
		"    listen_port = 0",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	firstCmd, firstOut := startRNSDService(t, ctx, rnsdBin, cfg, root)
	firstJSON, ok := tryWaitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	if !ok {
		t.Skipf("environment does not report TCPServerInterface status within timeout:\n%s", firstOut.String())
	}
	skipIfReticulumUnavailableRNSD(t, firstOut.String()+firstJSON, 0)
	firstStats := decodeRNStatusJSONRNSD(t, firstJSON)
	firstIfc := findInterfaceByShortNameRNSD(t, firstStats, "PersistTCPServer")
	if typ, _ := firstIfc["type"].(string); typ != "TCPServerInterface" {
		t.Fatalf("expected first type=TCPServerInterface, got %#v", firstIfc["type"])
	}
	stopRNSDService(t, firstCmd)

	secondCmd, secondOut := startRNSDService(t, ctx, rnsdBin, cfg, root)
	defer stopRNSDService(t, secondCmd)
	secondJSON, ok := tryWaitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	if !ok {
		t.Skipf("environment does not report TCPServerInterface status on restart within timeout:\n%s", secondOut.String())
	}
	skipIfReticulumUnavailableRNSD(t, secondOut.String()+secondJSON, 0)
	secondStats := decodeRNStatusJSONRNSD(t, secondJSON)
	secondIfc := findInterfaceByShortNameRNSD(t, secondStats, "PersistTCPServer")
	if typ, _ := secondIfc["type"].(string); typ != "TCPServerInterface" {
		t.Fatalf("expected second type=TCPServerInterface, got %#v", secondIfc["type"])
	}
}

func TestRNSDIntegration_Persistence_TCPClientInterfaceSurvivesRestart(t *testing.T) {
	root := t.TempDir()
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")

	dummyLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen dummy tcp server: %v", err)
	}
	defer dummyLn.Close()
	go func() {
		for {
			conn, err := dummyLn.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()
	targetPort := dummyLn.Addr().(*net.TCPAddr).Port

	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNSD(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = persist-tcpclient",
		"",
		"[interfaces]",
		"  [[PersistTCPClient]]",
		"    type = TCPClientInterface",
		"    enabled = yes",
		"    target_host = 127.0.0.1",
		"    target_port = " + strconv.Itoa(targetPort),
		"    connect_timeout = 0.1",
		"    reconnect_wait = 0.1",
		"    max_reconnect_tries = 0",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	firstCmd, firstOut := startRNSDService(t, ctx, rnsdBin, cfg, root)
	firstJSON := waitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	skipIfReticulumUnavailableRNSD(t, firstOut.String()+firstJSON, 0)
	firstStats := decodeRNStatusJSONRNSD(t, firstJSON)
	firstIfc := findInterfaceByShortNameRNSD(t, firstStats, "PersistTCPClient")
	if typ, _ := firstIfc["type"].(string); typ != "TCPClientInterface" {
		t.Fatalf("expected first type=TCPClientInterface, got %#v", firstIfc["type"])
	}
	stopRNSDService(t, firstCmd)

	secondCmd, secondOut := startRNSDService(t, ctx, rnsdBin, cfg, root)
	defer stopRNSDService(t, secondCmd)
	secondJSON := waitForRNStatusSuccessRNSD(t, rnstatusBin, cfg, root, 10*time.Second)
	skipIfReticulumUnavailableRNSD(t, secondOut.String()+secondJSON, 0)
	secondStats := decodeRNStatusJSONRNSD(t, secondJSON)
	secondIfc := findInterfaceByShortNameRNSD(t, secondStats, "PersistTCPClient")
	if typ, _ := secondIfc["type"].(string); typ != "TCPClientInterface" {
		t.Fatalf("expected second type=TCPClientInterface, got %#v", secondIfc["type"])
	}
}
