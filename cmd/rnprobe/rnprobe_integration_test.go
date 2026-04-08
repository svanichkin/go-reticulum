//go:build integration

package main

import (
	"context"
	"path/filepath"
	"strings"
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
