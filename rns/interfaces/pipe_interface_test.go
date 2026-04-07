package interfaces

import (
	"testing"
	"time"
)

func TestPipe_shlexSplit_Basic(t *testing.T) {
	t.Parallel()

	args, err := shlexSplit(`echo hello`)
	if err != nil {
		t.Fatalf("shlexSplit: %v", err)
	}
	if len(args) != 2 || args[0] != "echo" || args[1] != "hello" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestPipe_shlexSplit_QuotesAndEscapes(t *testing.T) {
	t.Parallel()

	args, err := shlexSplit(`cmd "a b" 'c d' e\ f`)
	if err != nil {
		t.Fatalf("shlexSplit: %v", err)
	}
	want := []string{"cmd", "a b", "c d", "e f"}
	if len(args) != len(want) {
		t.Fatalf("unexpected args: %#v", args)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("arg %d mismatch: got=%q want=%q", i, args[i], want[i])
		}
	}
}

func TestPipe_shlexSplit_Errors(t *testing.T) {
	t.Parallel()

	if _, err := shlexSplit(``); err == nil {
		t.Fatalf("expected error for empty command")
	}
	if _, err := shlexSplit(`unterminated "quote`); err == nil {
		t.Fatalf("expected error for unterminated quotes")
	}
	if _, err := shlexSplit(`trailing\`); err == nil {
		t.Fatalf("expected error for trailing escape")
	}
}

func TestPipe_parsePipeConfig(t *testing.T) {
	t.Parallel()

	cfg, err := parsePipeConfig("p", map[string]string{
		"command":       "echo hi",
		"respawn_delay": "0.2",
	})
	if err != nil {
		t.Fatalf("parsePipeConfig: %v", err)
	}
	if cfg.Command != "echo hi" {
		t.Fatalf("unexpected command %q", cfg.Command)
	}
	if cfg.RespawnDelay != 200*time.Millisecond {
		t.Fatalf("unexpected respawn delay %v", cfg.RespawnDelay)
	}
}

func TestPipeInterface_StringMatchesPython(t *testing.T) {
	t.Parallel()

	iface := &Interface{
		Name: "p0",
		Type: "PipeInterface",
	}

	if got := iface.String(); got != "PipeInterface[p0]" {
		t.Fatalf("unexpected string form %q", got)
	}
}
