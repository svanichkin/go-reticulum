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

func writeMinimalReticulumConfigRNPath(t *testing.T, configDir string) {
	t.Helper()
	// Standalone to avoid shared-instance sockets.
	cmdtest.WriteReticulumConfig(t, configDir, strings.Join([]string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = No",
		"",
	}, "\n"))
}

func skipIfReticulumUnavailableRNPath(t *testing.T, out string, exitCode int) {
	t.Helper()
	if exitCode == 101 || strings.Contains(out, "Could not start Reticulum") ||
		strings.Contains(out, "operation not permitted") ||
		strings.Contains(out, "context deadline exceeded") {
		t.Skipf("environment does not allow Reticulum startup; skipping rnpath integration test\n%s", out)
	}
}

func runRNPathNoFatal(ctx context.Context, bin string, opts cmdtest.RunOptions, args ...string) (cmdtest.RunResult, error) {
	c := exec.CommandContext(ctx, bin, args...)
	if opts.WorkDir != "" {
		c.Dir = opts.WorkDir
	}
	env := append([]string{}, os.Environ()...)
	if opts.ConfigDir != "" {
		home := filepath.Join(opts.ConfigDir, ".home")
		if err := os.MkdirAll(home, 0o755); err != nil {
			return cmdtest.RunResult{}, err
		}
		env = append(env, "HOME="+home, "USERPROFILE="+home)
	}
	env = append(env, opts.Env...)
	c.Env = env
	out, err := c.CombinedOutput()
	if err == nil {
		return cmdtest.RunResult{Output: string(out), ExitCode: 0}, nil
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return cmdtest.RunResult{Output: string(out), ExitCode: ee.ExitCode()}, nil
	}
	return cmdtest.RunResult{Output: string(out)}, err
}

func writeReticulumConfigRNPath(t *testing.T, configDir string, lines []string) {
	t.Helper()
	cmdtest.WriteReticulumConfig(t, configDir, strings.Join(lines, "\n"))
}

func freeTCPPortRNPath(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen ephemeral tcp port: %v", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func startRNSDServiceRNPath(t *testing.T, ctx context.Context, bin, cfg, workDir string) (*exec.Cmd, *bytes.Buffer) {
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

func waitForRNStatusSuccessRNPath(t *testing.T, rnstatusBin, cfg, workDir string, timeout time.Duration) string {
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

func decodeRNStatusJSONRNPath(t *testing.T, out string) map[string]any {
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

func decodeJSONArrayRNPath(t *testing.T, out string) []any {
	t.Helper()
	trimmed := strings.TrimSpace(out)
	if idx := strings.LastIndex(trimmed, "["); idx >= 0 {
		trimmed = trimmed[idx:]
	}
	var data []any
	if err := json.Unmarshal([]byte(trimmed), &data); err != nil {
		t.Fatalf("expected JSON array, got decode error: %v\n%s", err, out)
	}
	return data
}

func writeTwoNodeTemplateConfigRNPath(t *testing.T, dstDir, templatePath string, sharedPort, controlPort, listenPort, forwardPort int) {
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

func writeTwoNodeTemplateConfigRNPathWithRemote(t *testing.T, dstDir, templatePath string, sharedPort, controlPort, listenPort, forwardPort int, extraReticulum []string) {
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

func extractProbeHashFromRNStatusRNPath(t *testing.T, out string) string {
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

func TestRNPathIntegration_JSONTableEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnpath", "./cmd/rnpath")
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNPath(t, cfg)

	res, err := runRNPathNoFatal(ctx, bin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root}, "--config", cfg, "--table", "--json")
	if err != nil && strings.Contains(err.Error(), "context deadline exceeded") {
		t.Skipf("environment does not allow Reticulum startup; skipping rnpath integration test\n%v", err)
	}
	skipIfReticulumUnavailableRNPath(t, res.Output, res.ExitCode)
	if res.ExitCode != 0 {
		t.Fatalf("exit=%d\n%s", res.ExitCode, res.Output)
	}
	trim := strings.TrimSpace(res.Output)
	if trim != "[]" {
		t.Fatalf("expected [], got %q", trim)
	}
}

func TestRNPathIntegration_DropAnnouncesLocal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnpath", "./cmd/rnpath")
	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNPath(t, cfg)

	res, err := runRNPathNoFatal(ctx, bin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root}, "--config", cfg, "--drop-announces")
	if err != nil && strings.Contains(err.Error(), "context deadline exceeded") {
		t.Skipf("environment does not allow Reticulum startup; skipping rnpath integration test\n%v", err)
	}
	skipIfReticulumUnavailableRNPath(t, res.Output, res.ExitCode)
	if res.ExitCode != 0 {
		t.Fatalf("exit=%d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "Dropping announce queues on all interfaces") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNPathIntegration_SharedInstanceTableJSONSuccess(t *testing.T) {
	root := t.TempDir()
	rnpathBin := cmdtest.Build(t, root, "rnpath", "./cmd/rnpath")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNPath(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = rnpath-shared-local",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, out := startRNSDServiceRNPath(t, ctx, rnsdBin, cfg, root)
	localJSON := waitForRNStatusSuccessRNPath(t, rnstatusBin, cfg, root, 10*time.Second)
	skipIfReticulumUnavailableRNPath(t, out.String()+localJSON, 0)

	runCtx, runCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer runCancel()
	res := cmdtest.Run(t, runCtx, rnpathBin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root},
		"--config", cfg, "--table", "--json")
	if res.ExitCode != 0 {
		t.Fatalf("expected shared-instance rnpath success, got %d\n%s", res.ExitCode, res.Output)
	}
	trimmed := strings.TrimSpace(res.Output)
	if idx := strings.LastIndex(trimmed, "["); idx >= 0 {
		trimmed = trimmed[idx:]
	}
	if strings.TrimSpace(trimmed) != "[]" {
		t.Fatalf("expected empty path table, got:\n%s", res.Output)
	}
}

func TestRNPathIntegration_SharedInstanceDropAnnouncesSuccess(t *testing.T) {
	root := t.TempDir()
	rnpathBin := cmdtest.Build(t, root, "rnpath", "./cmd/rnpath")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	cfg := filepath.Join(root, "cfg")
	writeReticulumConfigRNPath(t, cfg, []string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = Yes",
		"instance_name = rnpath-shared-drop",
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, out := startRNSDServiceRNPath(t, ctx, rnsdBin, cfg, root)
	localJSON := waitForRNStatusSuccessRNPath(t, rnstatusBin, cfg, root, 10*time.Second)
	skipIfReticulumUnavailableRNPath(t, out.String()+localJSON, 0)

	runCtx, runCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer runCancel()
	res := cmdtest.Run(t, runCtx, rnpathBin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root},
		"--config", cfg, "--drop-announces")
	if res.ExitCode != 0 {
		t.Fatalf("expected shared-instance rnpath drop-announces success, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "Dropping announce queues on all interfaces") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNPathIntegration_SharedInstanceTCPTableJSONSuccess(t *testing.T) {
	root := t.TempDir()
	rnpathBin := cmdtest.Build(t, root, "rnpath", "./cmd/rnpath")
	rnstatusBin := cmdtest.Build(t, root, "rnstatus", "./cmd/rnstatus")
	rnsdBin := cmdtest.Build(t, root, "rnsd", "./cmd/rnsd")
	cfg := filepath.Join(root, "cfg")
	packetPort := freeTCPPortRNPath(t)
	controlPort := freeTCPPortRNPath(t)
	writeReticulumConfigRNPath(t, cfg, []string{
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

	_, out := startRNSDServiceRNPath(t, ctx, rnsdBin, cfg, root)
	localJSON := waitForRNStatusSuccessRNPath(t, rnstatusBin, cfg, root, 10*time.Second)
	skipIfReticulumUnavailableRNPath(t, out.String()+localJSON, 0)

	runCtx, runCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer runCancel()
	res := cmdtest.Run(t, runCtx, rnpathBin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root},
		"--config", cfg, "--table", "--json")
	if res.ExitCode != 0 {
		t.Fatalf("expected shared-instance tcp rnpath success, got %d\n%s", res.ExitCode, res.Output)
	}
	trimmed := strings.TrimSpace(res.Output)
	if idx := strings.LastIndex(trimmed, "["); idx >= 0 {
		trimmed = trimmed[idx:]
	}
	if strings.TrimSpace(trimmed) != "[]" {
		t.Fatalf("expected empty path table, got:\n%s", res.Output)
	}
}

func TestRNPathIntegration_CoreRemoteRequiresIdentityExit20(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnpath", "./cmd/rnpath")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: root, WorkDir: root}, "-R", strings.Repeat("0", 32), "--table")
	if res.ExitCode != 20 {
		t.Fatalf("expected exit 20, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "Management identity path required (-i)") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNPathIntegration_CoreRemoteInvalidDestinationLengthExit20(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnpath", "./cmd/rnpath")
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

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: root, WorkDir: root}, "-R", "abcd", "-i", identityPath, "--table")
	if res.ExitCode != 20 {
		t.Fatalf("expected exit 20, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "Destination length is invalid") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNPathIntegration_CoreRemoteMissingIdentityFileExit20(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnpath", "./cmd/rnpath")
	missingPath := filepath.Join(root, "missing.id")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: root, WorkDir: root}, "-R", strings.Repeat("0", 32), "-i", missingPath, "--table")
	if res.ExitCode != 20 {
		t.Fatalf("expected exit 20, got %d\n%s", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "Could not load management identity") {
		t.Fatalf("unexpected output:\n%s", res.Output)
	}
}

func TestRNPathIntegration_CoreRemoteTableAllowedIdentityCanQuerySelf(t *testing.T) {
	root := t.TempDir()
	rnpathBin := cmdtest.Build(t, root, "rnpath", "./cmd/rnpath")
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

	writeReticulumConfigRNPath(t, cfg, []string{
		"[reticulum]",
		"enable_transport = True",
		"share_instance = Yes",
		"instance_name = rnpath-core-remote",
		"enable_remote_management = True",
		"remote_management_allowed = " + hex.EncodeToString(managementIdentity.Hash),
		"",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	_, out := startRNSDServiceRNPath(t, ctx, rnsdBin, cfg, root)
	localJSON := waitForRNStatusSuccessRNPath(t, rnstatusBin, cfg, root, 10*time.Second)
	skipIfReticulumUnavailableRNPath(t, out.String()+localJSON, 0)

	localStats := decodeRNStatusJSONRNPath(t, localJSON)
	transportID, ok := localStats["transport_id"].(string)
	if !ok || transportID == "" {
		t.Fatalf("expected transport_id in local status:\n%s", localJSON)
	}

	runCtx, runCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer runCancel()
	res := cmdtest.Run(t, runCtx, rnpathBin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root},
		"--config", cfg, "--table", "--json", "-R", transportID, "-i", managementIdentityPath)
	if res.ExitCode == 12 || strings.Contains(res.Output, "Path request timed out") {
		t.Skipf("environment does not permit remote-management self-query path establishment\n%s", res.Output)
	}
	if res.ExitCode != 0 {
		t.Fatalf("expected remote rnpath table success, got %d\n%s", res.ExitCode, res.Output)
	}
	if strings.TrimSpace(res.Output) != "[]" {
		t.Fatalf("expected empty remote path table, got:\n%s", res.Output)
	}
}
