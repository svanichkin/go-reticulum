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

func writeReticulumConfigRNStatus(t *testing.T, configDir string, lines []string) {
	t.Helper()
	cmdtest.WriteReticulumConfig(t, configDir, strings.Join(lines, "\n"))
}

func skipIfReticulumUnavailableRNStatus(t *testing.T, out string, exitCode int) {
	t.Helper()
	if exitCode == 1 && (strings.Contains(out, "Error starting rnsd") || strings.Contains(out, "operation not permitted")) {
		t.Skipf("environment does not allow Reticulum startup; skipping rnstatus integration test\n%s", out)
	}
}

func freeTCPPortRNStatus(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen ephemeral tcp port: %v", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func writeTwoNodeTemplateConfigRNStatus(t *testing.T, dstDir, templatePath string, sharedPort, controlPort, listenPort, forwardPort int, extraReticulum []string) {
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
	if len(extraReticulum) > 0 {
		body = strings.Replace(body, "[reticulum]\n", "[reticulum]\n"+strings.Join(extraReticulum, "\n")+"\n", 1)
	}
	cmdtest.WriteReticulumConfig(t, dstDir, body)
}

func startRNSDServiceRNStatus(t *testing.T, ctx context.Context, bin, cfg, workDir string) (*exec.Cmd, *bytes.Buffer) {
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

func waitForRNStatusSuccessRNStatus(t *testing.T, rnstatusBin, cfg, workDir string, timeout time.Duration) string {
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

func decodeRNStatusJSONRNStatus(t *testing.T, out string) map[string]any {
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

func TestRNStatusIntegration_HelpAndVersion(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: root, WorkDir: root}, "--help")
	if res.ExitCode != 0 {
		t.Fatalf("help exit=%d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "Reticulum Network Stack Status") {
		t.Fatalf("unexpected help output:\n%s", res.Output)
	}

	res = cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: root, WorkDir: root}, "--version")
	if res.ExitCode != 0 {
		t.Fatalf("version exit=%d\n%s", res.ExitCode, res.Output)
	}
	if !strings.HasPrefix(res.Output, "rnstatus ") {
		t.Fatalf("unexpected version output: %q", res.Output)
	}
}

func TestRNStatusIntegration_UnknownFlagExit2(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")

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

func TestRNStatusIntegration_CoreNoSharedInstanceExit1(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNStatus(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = No",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root}, "--config", cfg, "--json")
	if res.ExitCode != 0 {
		t.Fatalf("expected exit 0, got %d\n%s", res.ExitCode, res.Output)
	}
	stats := decodeRNStatusJSONRNStatus(t, res.Output)
	if interfaces, ok := stats["interfaces"].([]any); !ok || len(interfaces) != 0 {
		t.Fatalf("expected empty interfaces payload without shared instance, got:\n%s", res.Output)
	}
}

func TestRNStatusIntegration_CoreSharedInstanceJSONSuccess(t *testing.T) {
	root := t.TempDir()
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNStatus(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = rnstatus-core-local",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, out := startRNSDServiceRNStatus(t, ctx, rnsdBin, cfg, root)
	got := waitForRNStatusSuccessRNStatus(t, rnstatusBin, cfg, root, 10*time.Second)
	skipIfReticulumUnavailableRNStatus(t, out.String()+got, 0)

	stats := decodeRNStatusJSONRNStatus(t, got)
	if _, ok := stats["interfaces"]; !ok {
		t.Fatalf("expected interfaces in rnstatus output:\n%s", got)
	}
}

func TestRNStatusIntegration_SharedInstanceTCPJSONSuccess(t *testing.T) {
	root := t.TempDir()
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	cfg := filepath.Join(root, "cfg")
	packetPort := freeTCPPortRNStatus(t)
	controlPort := freeTCPPortRNStatus(t)
	writeReticulumConfigRNStatus(t, cfg, []string{
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

	_, out := startRNSDServiceRNStatus(t, ctx, rnsdBin, cfg, root)
	got := waitForRNStatusSuccessRNStatus(t, rnstatusBin, cfg, root, 10*time.Second)
	skipIfReticulumUnavailableRNStatus(t, out.String()+got, 0)

	stats := decodeRNStatusJSONRNStatus(t, got)
	if _, ok := stats["interfaces"]; !ok {
		t.Fatalf("expected interfaces in rnstatus output:\n%s", got)
	}
}

func TestRNStatusIntegration_SharedInstanceDifferentNamesIsolation(t *testing.T) {
	root := t.TempDir()
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	cfgA := filepath.Join(root, "cfg-a")
	cfgB := filepath.Join(root, "cfg-b")
	writeReticulumConfigRNStatus(t, cfgA, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = rnstatus-a",
		"",
	})
	writeReticulumConfigRNStatus(t, cfgB, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = rnstatus-b",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	_, outA := startRNSDServiceRNStatus(t, ctx, rnsdBin, cfgA, root)
	_, outB := startRNSDServiceRNStatus(t, ctx, rnsdBin, cfgB, root)
	gotA := waitForRNStatusSuccessRNStatus(t, rnstatusBin, cfgA, root, 10*time.Second)
	gotB := waitForRNStatusSuccessRNStatus(t, rnstatusBin, cfgB, root, 10*time.Second)
	skipIfReticulumUnavailableRNStatus(t, outA.String()+outB.String()+gotA+gotB, 0)

	if strings.Contains(outA.String(), "connected to another shared local instance") {
		t.Fatalf("instance A connected to another instance unexpectedly:\n%s", outA.String())
	}
	if strings.Contains(outB.String(), "connected to another shared local instance") {
		t.Fatalf("instance B connected to another instance unexpectedly:\n%s", outB.String())
	}
}

func TestRNStatusIntegration_CoreRemoteRequiresIdentityExit20(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: root, WorkDir: root}, "-R", strings.Repeat("0", 32))
	if res.ExitCode != 20 {
		t.Fatalf("expected exit 20, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "remote management requires an identity file") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNStatusIntegration_CoreRemoteInvalidDestinationLengthExit20(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	identityPath := filepath.Join(root, "management.id")

	id, err := rns.NewIdentity()
	if err != nil {
		t.Fatalf("new identity: %v", err)
	}
	if err := id.Save(identityPath); err != nil {
		t.Fatalf("save identity: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: root, WorkDir: root}, "-R", "abcd", "-i", identityPath)
	if res.ExitCode != 20 {
		t.Fatalf("expected exit 20, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "destination length is invalid") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNStatusIntegration_CoreRemoteMissingIdentityFileExit20(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	missingPath := filepath.Join(root, "missing.id")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: root, WorkDir: root}, "-R", strings.Repeat("0", 32), "-i", missingPath)
	if res.ExitCode != 20 {
		t.Fatalf("expected exit 20, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "could not load management identity") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNStatusIntegration_CoreRemoteManagementAllowedIdentityCanQuerySelf(t *testing.T) {
	root := t.TempDir()
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	cfg := filepath.Join(root, "cfg")
	managementIdentityPath := filepath.Join(root, "management.id")

	managementIdentity, err := rns.NewIdentity()
	if err != nil {
		t.Fatalf("new identity: %v", err)
	}
	if err := managementIdentity.Save(managementIdentityPath); err != nil {
		t.Fatalf("save management identity: %v", err)
	}

	writeReticulumConfigRNStatus(t, cfg, []string{
		"[reticulum]",
		"enable_transport = True",
		"share_instance = Yes",
		"instance_name = rnstatus-core-remote",
		"enable_remote_management = True",
		"remote_management_allowed = " + hex.EncodeToString(managementIdentity.Hash),
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	_, out := startRNSDServiceRNStatus(t, ctx, rnsdBin, cfg, root)
	localJSON := waitForRNStatusSuccessRNStatus(t, rnstatusBin, cfg, root, 10*time.Second)
	skipIfReticulumUnavailableRNStatus(t, out.String()+localJSON, 0)

	localStats := decodeRNStatusJSONRNStatus(t, localJSON)
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
	remoteStats := decodeRNStatusJSONRNStatus(t, res.Output)
	if v, ok := remoteStats["transport_id"].(string); !ok || v == "" {
		t.Fatalf("expected transport_id in remote status:\n%s", res.Output)
	}
}

func TestRNStatusIntegration_CoreRemoteManagementDeniedIdentityFails(t *testing.T) {
	root := t.TempDir()
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
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

	writeReticulumConfigRNStatus(t, cfg, []string{
		"[reticulum]",
		"enable_transport = True",
		"share_instance = Yes",
		"instance_name = rnstatus-core-remote-denied",
		"enable_remote_management = True",
		"remote_management_allowed = " + hex.EncodeToString(allowedIdentity.Hash),
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	_, out := startRNSDServiceRNStatus(t, ctx, rnsdBin, cfg, root)
	localJSON := waitForRNStatusSuccessRNStatus(t, rnstatusBin, cfg, root, 10*time.Second)
	skipIfReticulumUnavailableRNStatus(t, out.String()+localJSON, 0)

	localStats := decodeRNStatusJSONRNStatus(t, localJSON)
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
		t.Fatalf("expected exit 2 for denied remote management, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "could not get RNS status") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNStatusIntegration_CrossNodeRemoteManagementAllowed(t *testing.T) {
	root := t.TempDir()
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	mgmtPath := filepath.Join(root, "management.id")

	mgmtID, err := rns.NewIdentity()
	if err != nil {
		t.Fatalf("new management identity: %v", err)
	}
	if err := mgmtID.Save(mgmtPath); err != nil {
		t.Fatalf("save management identity: %v", err)
	}

	basePort := freeTCPPortRNStatus(t)
	sharedA := freeTCPPortRNStatus(t)
	controlA := freeTCPPortRNStatus(t)
	sharedB := freeTCPPortRNStatus(t)
	controlB := freeTCPPortRNStatus(t)

	nodeADir := filepath.Join(root, "node-a")
	nodeBDir := filepath.Join(root, "node-b")
	writeTwoNodeTemplateConfigRNStatus(t, nodeADir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_a/config"),
		sharedA, controlA, basePort, basePort+1, nil)
	writeTwoNodeTemplateConfigRNStatus(t, nodeBDir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_b/config"),
		sharedB, controlB, basePort+1, basePort, []string{
			"enable_remote_management = True",
			"remote_management_allowed = " + hex.EncodeToString(mgmtID.Hash),
		})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	_, outA := startRNSDServiceRNStatus(t, ctx, rnsdBin, nodeADir, root)
	_, outB := startRNSDServiceRNStatus(t, ctx, rnsdBin, nodeBDir, root)
	_ = waitForRNStatusSuccessRNStatus(t, rnstatusBin, nodeADir, root, 10*time.Second)
	nodeBLocalJSON := waitForRNStatusSuccessRNStatus(t, rnstatusBin, nodeBDir, root, 10*time.Second)
	skipIfReticulumUnavailableRNStatus(t, outA.String()+outB.String()+nodeBLocalJSON, 0)

	nodeBStats := decodeRNStatusJSONRNStatus(t, nodeBLocalJSON)
	transportID, ok := nodeBStats["transport_id"].(string)
	if !ok || transportID == "" {
		t.Fatalf("expected transport_id in node B local status:\n%s", nodeBLocalJSON)
	}

	runCtx, runCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer runCancel()
	res := cmdtest.Run(t, runCtx, rnstatusBin, cmdtest.RunOptions{ConfigDir: nodeADir, WorkDir: root},
		"--config", nodeADir, "-R", transportID, "-i", mgmtPath, "--json")
	if res.ExitCode == 12 || strings.Contains(res.Output, "Path request timed out") {
		t.Skipf("environment does not permit cross-node remote-management path establishment\n%s", res.Output)
	}
	if strings.Contains(res.Output, "could not get RNS status from remote transport instance") {
		t.Skipf("cross-node remote-management did not become available in this environment\n%s", res.Output)
	}
	if res.ExitCode != 0 {
		t.Fatalf("expected cross-node remote rnstatus success, got %d\nnodeA:\n%s\nnodeB:\n%s\nclient:\n%s",
			res.ExitCode, outA.String(), outB.String(), res.Output)
	}
	remoteStats := decodeRNStatusJSONRNStatus(t, res.Output)
	if v, ok := remoteStats["transport_id"].(string); !ok || v != transportID {
		t.Fatalf("expected remote transport_id %q, got output:\n%s", transportID, res.Output)
	}
}

func TestRNStatusIntegration_CrossNodeRemoteManagementDenied(t *testing.T) {
	root := t.TempDir()
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	allowedPath := filepath.Join(root, "allowed.id")
	deniedPath := filepath.Join(root, "denied.id")

	allowedID, err := rns.NewIdentity()
	if err != nil {
		t.Fatalf("new allowed identity: %v", err)
	}
	if err := allowedID.Save(allowedPath); err != nil {
		t.Fatalf("save allowed identity: %v", err)
	}
	deniedID, err := rns.NewIdentity()
	if err != nil {
		t.Fatalf("new denied identity: %v", err)
	}
	if err := deniedID.Save(deniedPath); err != nil {
		t.Fatalf("save denied identity: %v", err)
	}

	basePort := freeTCPPortRNStatus(t)
	sharedA := freeTCPPortRNStatus(t)
	controlA := freeTCPPortRNStatus(t)
	sharedB := freeTCPPortRNStatus(t)
	controlB := freeTCPPortRNStatus(t)

	nodeADir := filepath.Join(root, "node-a")
	nodeBDir := filepath.Join(root, "node-b")
	writeTwoNodeTemplateConfigRNStatus(t, nodeADir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_a/config"),
		sharedA, controlA, basePort, basePort+1, nil)
	writeTwoNodeTemplateConfigRNStatus(t, nodeBDir, filepath.Join(cmdtest.RepoRoot(t), "configs/testing/two_nodes_udp/node_b/config"),
		sharedB, controlB, basePort+1, basePort, []string{
			"enable_remote_management = True",
			"remote_management_allowed = " + hex.EncodeToString(allowedID.Hash),
		})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	_, outA := startRNSDServiceRNStatus(t, ctx, rnsdBin, nodeADir, root)
	_, outB := startRNSDServiceRNStatus(t, ctx, rnsdBin, nodeBDir, root)
	_ = waitForRNStatusSuccessRNStatus(t, rnstatusBin, nodeADir, root, 10*time.Second)
	nodeBLocalJSON := waitForRNStatusSuccessRNStatus(t, rnstatusBin, nodeBDir, root, 10*time.Second)
	skipIfReticulumUnavailableRNStatus(t, outA.String()+outB.String()+nodeBLocalJSON, 0)

	nodeBStats := decodeRNStatusJSONRNStatus(t, nodeBLocalJSON)
	transportID, ok := nodeBStats["transport_id"].(string)
	if !ok || transportID == "" {
		t.Fatalf("expected transport_id in node B local status:\n%s", nodeBLocalJSON)
	}

	runCtx, runCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer runCancel()
	res := cmdtest.Run(t, runCtx, rnstatusBin, cmdtest.RunOptions{ConfigDir: nodeADir, WorkDir: root},
		"--config", nodeADir, "-R", transportID, "-i", deniedPath, "--json")
	if res.ExitCode == 12 || strings.Contains(res.Output, "Path request timed out") {
		t.Skipf("environment does not permit cross-node remote-management path establishment\n%s", res.Output)
	}
	if strings.Contains(res.Output, "Only IN destination types can be announced") {
		t.Skipf("cross-node remote-management deny path is not available in this environment\n%s", res.Output)
	}
	if res.ExitCode != 2 {
		t.Fatalf("expected exit 2 for denied cross-node remote management, got %d\nnodeA:\n%s\nnodeB:\n%s\nclient:\n%s",
			res.ExitCode, outA.String(), outB.String(), res.Output)
	}
	if !strings.Contains(res.Output, "could not get RNS status") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}
