package rns

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestVersionString_MatchesPython114(t *testing.T) {
	if got := VersionString(); got != "1.1.4" {
		t.Fatalf("VersionString()=%q, want 1.1.4", got)
	}
}

func TestCompiled_DefaultsFalse(t *testing.T) {
	prev := compiledFlag
	compiledFlag = false
	t.Cleanup(func() {
		compiledFlag = prev
	})

	if Compiled() {
		t.Fatal("Compiled()=true, want false by default")
	}
}

func TestGetProfiler_EmptyTagPreserved(t *testing.T) {
	prevRegistry := profilerRegistry
	prevTags := profilerTags
	prevOrder := profilerOrder
	prevRan := profilerHasRun

	profilerRegistry = make(map[string]*Profiler)
	profilerTags = make(map[string]*profilerTagEntry)
	profilerOrder = nil
	profilerHasRun = false
	t.Cleanup(func() {
		profilerRegistry = prevRegistry
		profilerTags = prevTags
		profilerOrder = prevOrder
		profilerHasRun = prevRan
	})

	prof := GetProfiler("")
	if prof == nil {
		t.Fatal("GetProfiler(\"\") returned nil")
	}
	if prof.tag != "" {
		t.Fatalf("profiler tag=%q, want empty string", prof.tag)
	}
}

func TestProfilerResults_PreservesCreationOrder(t *testing.T) {
	prevRegistry := profilerRegistry
	prevTags := profilerTags
	prevOrder := profilerOrder
	prevRan := profilerHasRun
	prevDest := logDest

	profilerRegistry = make(map[string]*Profiler)
	profilerTags = make(map[string]*profilerTagEntry)
	profilerOrder = nil
	profilerHasRun = false
	logDest = LogStdout
	t.Cleanup(func() {
		profilerRegistry = prevRegistry
		profilerTags = prevTags
		profilerOrder = prevOrder
		profilerHasRun = prevRan
		logDest = prevDest
	})

	first := GetProfiler("beta")
	second := GetProfiler("alpha")

	first.Enter()
	time.Sleep(1 * time.Millisecond)
	first.Exit()

	second.Enter()
	time.Sleep(1 * time.Millisecond)
	second.Exit()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(): %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	ProfilerResults()

	_ = w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll(stdout): %v", err)
	}

	text := string(out)
	beta := strings.Index(text, "beta")
	alpha := strings.Index(text, "alpha")
	if beta == -1 || alpha == -1 {
		t.Fatalf("expected profiler output to contain beta and alpha, got %q", text)
	}
	if beta > alpha {
		t.Fatalf("profiler output order mismatch: beta index=%d alpha index=%d output=%q", beta, alpha, text)
	}
}

func TestTraceException_UsesVerboseErrorWhenAvailable(t *testing.T) {
	prevDest := logDest
	prevLevel := logLevel
	logDest = LogStdout
	logLevel = LogError
	t.Cleanup(func() {
		logDest = prevDest
		logLevel = prevLevel
	})

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(): %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	TraceException(verboseTraceError{msg: "boom", trace: "stack line 1\nstack line 2"})

	_ = w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll(stdout): %v", err)
	}

	if !bytes.Contains(out, []byte("stack line 1")) {
		t.Fatalf("TraceException output missing verbose trace: %q", string(out))
	}
}

func TestTraceException_PlainErrorDoesNotLogCurrentGoroutineStack(t *testing.T) {
	prevDest := logDest
	prevLevel := logLevel
	logDest = LogStdout
	logLevel = LogError
	t.Cleanup(func() {
		logDest = prevDest
		logLevel = prevLevel
	})

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(): %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	TraceException(fmt.Errorf("plain boom"))

	_ = w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll(stdout): %v", err)
	}

	text := string(out)
	if strings.Contains(text, "goroutine ") {
		t.Fatalf("TraceException output unexpectedly contains current goroutine stack: %q", text)
	}
	if !strings.Contains(text, "plain boom") {
		t.Fatalf("TraceException output missing error text: %q", text)
	}
}

type verboseTraceError struct {
	msg   string
	trace string
}

func (e verboseTraceError) Error() string { return e.msg }

func (e verboseTraceError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			_, _ = io.WriteString(s, e.trace)
			return
		}
		fallthrough
	case 's':
		_, _ = io.WriteString(s, e.msg)
	case 'q':
		_, _ = io.WriteString(s, `"`+e.msg+`"`)
	}
}
