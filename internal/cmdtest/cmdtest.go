package cmdtest

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
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

	repoRoot := RepoRoot(t)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	cacheRoot := filepath.Join(repoRoot, ".cmdtestbin", runtime.GOOS+"_"+runtime.GOARCH)
	if err := os.MkdirAll(cacheRoot, 0o755); err != nil {
		t.Fatalf("mkdir cmdtest cache: %v", err)
	}
	key := strings.NewReplacer("/", "_", "\\", "_", ".", "_").Replace(pkg)
	out := filepath.Join(cacheRoot, fmt.Sprintf("%s__%s", name, key))
	gocache := filepath.Join(binDir, ".gocache")
	gotmp := filepath.Join(binDir, ".gotmp")
	gopath := filepath.Join(binDir, ".gopath")
	gomodcache := filepath.Join(binDir, ".gomodcache")
	if err := os.MkdirAll(gocache, 0o755); err != nil {
		t.Fatalf("mkdir gocache: %v", err)
	}
	if err := os.MkdirAll(gotmp, 0o755); err != nil {
		t.Fatalf("mkdir gotmp: %v", err)
	}
	if err := os.MkdirAll(gopath, 0o755); err != nil {
		t.Fatalf("mkdir gopath: %v", err)
	}
	if err := os.MkdirAll(gomodcache, 0o755); err != nil {
		t.Fatalf("mkdir gomodcache: %v", err)
	}
	lockPath := out + ".lock"
	lockDeadline := time.Now().Add(2 * time.Minute)
	for {
		lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_ = lock.Close()
			defer os.Remove(lockPath)
			break
		}
		if !os.IsExist(err) {
			t.Fatalf("create build lock: %v", err)
		}
		if _, statErr := os.Stat(out); statErr == nil {
			return out
		}
		if time.Now().After(lockDeadline) {
			t.Fatalf("timed out waiting for build lock %s", lockPath)
		}
		time.Sleep(200 * time.Millisecond)
	}
	if _, err := os.Stat(out); err == nil {
		return out
	}

	cmd := exec.Command("go", "build", "-o", out, pkg)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(),
		"GOCACHE="+gocache,
		"GOTMPDIR="+gotmp,
		"GOPATH="+gopath,
		"GOMODCACHE="+gomodcache,
		"GOFLAGS=-modcacherw",
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

func AcquireLock(t *testing.T, name string, timeout time.Duration) {
	t.Helper()

	repoRoot := RepoRoot(t)
	lockDir := filepath.Join(repoRoot, ".cmdtestlocks")
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		t.Fatalf("mkdir lockdir: %v", err)
	}

	lockPath := filepath.Join(lockDir, name+".lock")
	deadline := time.Now().Add(timeout)
	for {
		lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_ = lock.Close()
			t.Cleanup(func() { _ = os.Remove(lockPath) })
			return
		}
		if !os.IsExist(err) {
			t.Fatalf("create test lock: %v", err)
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for test lock %s", lockPath)
		}
		time.Sleep(200 * time.Millisecond)
	}
}
