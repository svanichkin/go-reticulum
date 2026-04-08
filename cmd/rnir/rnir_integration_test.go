//go:build integration

package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/svanichkin/go-reticulum/internal/cmdtest"
)

func writeMinimalReticulumConfigRNIR(t *testing.T, configDir string) {
	t.Helper()
	// Prefer standalone instance to avoid shared-instance socket/network complexity.
	cmdtest.WriteReticulumConfig(t, configDir, strings.Join([]string{
		"[reticulum]",
		"enable_transport = False",
		"share_instance = No",
		"",
	}, "\n"))
}

func skipIfReticulumUnavailableRNIR(t *testing.T, out string, exitCode int) {
	t.Helper()
	if exitCode == 1 && (strings.Contains(out, "Could not start Reticulum") || strings.Contains(out, "operation not permitted")) {
		t.Skipf("environment does not allow Reticulum startup; skipping rnir integration test\n%s", out)
	}
}

func TestRNIRIntegration_ExampleConfig(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnir", "./cmd/rnir")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: root, WorkDir: root}, "--exampleconfig")
	if res.ExitCode != 0 {
		t.Fatalf("exit=%d\n%s", res.ExitCode, res.Output)
	}
	if res.Output != exampleConfig {
		t.Fatalf("unexpected output:\n%q", res.Output)
	}
}

func TestRNIRIntegration_Version(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnir", "./cmd/rnir")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: root, WorkDir: root}, "--version")
	if res.ExitCode != 0 {
		t.Fatalf("exit=%d\n%s", res.ExitCode, res.Output)
	}
	if !strings.HasPrefix(strings.TrimSpace(res.Output), "rnir ") {
		t.Fatalf("unexpected version output: %q", res.Output)
	}
}

func TestRNIRIntegration_StandaloneStartup(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnir", "./cmd/rnir")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNIR(t, cfg)

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root}, "--config", cfg)
	skipIfReticulumUnavailableRNIR(t, res.Output, res.ExitCode)
	if res.ExitCode != 0 {
		t.Fatalf("exit=%d\n%s", res.ExitCode, res.Output)
	}
	if strings.TrimSpace(res.Output) != "" {
		t.Fatalf("expected no output on successful startup, got:\n%s", res.Output)
	}
}

func TestRNIRIntegration_ServiceModeCreatesLogfile(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnir", "./cmd/rnir")
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNIR(t, cfg)

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root}, "--config", cfg, "--service")
	skipIfReticulumUnavailableRNIR(t, res.Output, res.ExitCode)
	if res.ExitCode != 0 {
		t.Fatalf("exit=%d\n%s", res.ExitCode, res.Output)
	}
	if _, err := os.Stat(filepath.Join(cfg, "logfile")); err != nil {
		t.Fatalf("expected logfile to exist: %v", err)
	}
}

func TestRNIRIntegration_CountFlagsExpand(t *testing.T) {
	root := t.TempDir()
	bin := cmdtest.Build(t, root, "rnir", "./cmd/rnir")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cfg := filepath.Join(root, "cfg")
	writeMinimalReticulumConfigRNIR(t, cfg)

	res := cmdtest.Run(t, ctx, bin, cmdtest.RunOptions{ConfigDir: cfg, WorkDir: root}, "--config", cfg, "-vvv", "-qq")
	skipIfReticulumUnavailableRNIR(t, res.Output, res.ExitCode)
	if res.ExitCode != 0 {
		t.Fatalf("exit=%d\n%s", res.ExitCode, res.Output)
	}
}
