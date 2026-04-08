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
