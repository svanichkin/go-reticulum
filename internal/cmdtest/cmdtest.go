package cmdtest

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

type RunOptions struct {
	ConfigDir string
	WorkDir   string
	Env       []string
	Stdin     io.Reader
}

type RunResult struct {
	Output   string
	ExitCode int
}

func RepoRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repo root from %s", wd)
		}
		dir = parent
	}
}

func Build(t *testing.T, binDir, name, pkg string) string {
	t.Helper()

	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	out := filepath.Join(binDir, name)
	gocache := filepath.Join(binDir, ".gocache")
	gotmp := filepath.Join(binDir, ".gotmp")
	if err := os.MkdirAll(gocache, 0o755); err != nil {
		t.Fatalf("mkdir gocache: %v", err)
	}
	if err := os.MkdirAll(gotmp, 0o755); err != nil {
		t.Fatalf("mkdir gotmp: %v", err)
	}

	cmd := exec.Command("go", "build", "-o", out, pkg)
	cmd.Dir = RepoRoot(t)
	cmd.Env = append(os.Environ(),
		"GOCACHE="+gocache,
		"GOTMPDIR="+gotmp,
	)
	if err := cmd.Run(); err != nil {
		t.Fatalf("build %s: %v", name, err)
	}
	return out
}

func Run(t *testing.T, ctx context.Context, bin string, opts RunOptions, args ...string) RunResult {
	t.Helper()

	c := exec.CommandContext(ctx, bin, args...)
	if opts.WorkDir != "" {
		c.Dir = opts.WorkDir
	}
	if opts.Stdin != nil {
		c.Stdin = opts.Stdin
	}

	env := append([]string{}, os.Environ()...)
	if opts.ConfigDir != "" {
		home := filepath.Join(opts.ConfigDir, ".home")
		if err := os.MkdirAll(home, 0o755); err != nil {
			t.Fatalf("mkdir home: %v", err)
		}
		env = append(env,
			"HOME="+home,
			"USERPROFILE="+home,
		)
	}
	env = append(env, opts.Env...)
	c.Env = env

	out, err := c.CombinedOutput()
	if err == nil {
		return RunResult{Output: string(out), ExitCode: 0}
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return RunResult{Output: string(out), ExitCode: ee.ExitCode()}
	}
	t.Fatalf("run %s: %v\n%s", bin, err, string(out))
	return RunResult{}
}

func WriteReticulumConfig(t *testing.T, configDir string, body string) {
	t.Helper()

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir configdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config"), []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}
