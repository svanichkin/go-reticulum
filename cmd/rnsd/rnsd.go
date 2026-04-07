package main

import (
	"bufio"
	_ "embed"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"syscall"
	"time"

	rns "github.com/svanichkin/go-reticulum/rns"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

var rnsdVersion = rns.GetVersion()

type countFlag int

func (c *countFlag) String() string   { return fmt.Sprint(int(*c)) }
func (c *countFlag) IsBoolFlag() bool { return true }
func (c *countFlag) Set(string) error {
	*c++
	return nil
}
func (c *countFlag) Value() int { return int(*c) }

//go:embed example_config.txt
var exampleRNSConfig string

func main() {
	var (
		configDir   string
		verboseCnt  countFlag
		quietCnt    countFlag
		asService   bool
		interactive bool
		exampleConf bool
		showVersion bool
		showHelp    bool
	)

	// Use a local FlagSet to avoid pulling in global flags from dependencies
	// (eg `testing/quick` via Yaegi, which otherwise exposes `-quickchecks`).
	fs := flag.NewFlagSet("rnsd", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.StringVar(&configDir, "config", "", "path to alternative Reticulum config directory")
	fs.Var(&verboseCnt, "verbose", "increase verbosity")
	fs.Var(&verboseCnt, "v", "increase verbosity")
	fs.Var(&quietCnt, "quiet", "decrease verbosity")
	fs.Var(&quietCnt, "q", "decrease verbosity")
	fs.BoolVar(&asService, "service", false, "rnsd is running as a service and should log to file")
	fs.BoolVar(&asService, "s", false, "rnsd is running as a service and should log to file")
	fs.BoolVar(&interactive, "interactive", false, "drop into interactive shell after initialisation")
	fs.BoolVar(&interactive, "i", false, "drop into interactive shell after initialisation")
	fs.BoolVar(&exampleConf, "exampleconfig", false, "print verbose configuration example to stdout and exit")
	fs.BoolVar(&showVersion, "version", false, "show version and exit")
	fs.BoolVar(&showHelp, "help", false, "show this help message and exit")
	fs.BoolVar(&showHelp, "h", false, "show this help message and exit")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Reticulum Network Stack Daemon")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "options:")
		fs.SetOutput(os.Stderr)
		fs.PrintDefaults()
	}

	if err := fs.Parse(expandCountFlags(os.Args[1:])); err != nil {
		fs.Usage()
		msg := strings.TrimSpace(err.Error())
		if msg != "" {
			if strings.HasPrefix(msg, "flag provided but not defined") {
				parts := strings.SplitN(msg, ":", 2)
				unk := msg
				if len(parts) == 2 {
					unk = strings.TrimSpace(parts[1])
				}
				fmt.Fprintf(os.Stderr, "rnsd: error: unrecognized arguments: %s\n", unk)
			} else {
				fmt.Fprintf(os.Stderr, "rnsd: error: %s\n", msg)
			}
		}
		os.Exit(2)
	}

	if showHelp {
		fs.Usage()
		return
	}

	if fs.NArg() > 0 {
		fs.Usage()
		fmt.Fprintf(os.Stderr, "rnsd: error: unrecognized arguments: %s\n", strings.Join(fs.Args(), " "))
		os.Exit(2)
	}

	if showVersion {
		fmt.Printf("rnsd %s\n", rnsdVersion)
		return
	}

	if exampleConf {
		fmt.Println(exampleRNSConfig)
		return
	}

	var cfg *string
	if strings.TrimSpace(configDir) != "" {
		cfg = &configDir
	}

	if err := programSetup(cfg, verboseCnt.Value(), quietCnt.Value(), asService, interactive); err != nil {
		fmt.Fprintln(os.Stderr, "Error starting rnsd:", err)
		os.Exit(1)
	}
}

// expandCountFlags expands Python-style "-vvv" / "-qq" into "-v -v -v" / "-q -q".
// This keeps compatibility with the upstream argparse action='count' semantics.
func expandCountFlags(args []string) []string {
	var out []string
	for _, a := range args {
		if len(a) >= 3 && strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") {
			rest := a[1:]
			if allSameRune(rest, 'v') {
				for range rest {
					out = append(out, "-v")
				}
				continue
			}
			if allSameRune(rest, 'q') {
				for range rest {
					out = append(out, "-q")
				}
				continue
			}
		}
		out = append(out, a)
	}
	return out
}

func allSameRune(s string, r rune) bool {
	for _, rr := range s {
		if rr != r {
			return false
		}
	}
	return s != ""
}

func programSetup(configdir *string, verbosity, quietness int, service, interactive bool) error {
	targetVerbosity := verbosity - quietness

	var logDest any
	var verbPtr *int

	if service {
		logDest = rns.LogFile
		verbPtr = nil
	} else {
		logDest = rns.LogStdout
		verbPtr = &targetVerbosity
	}

	ret, err := rns.NewReticulum(configdir, nil, logDest, verbPtr, false, nil)
	if err != nil {
		return err
	}

	if ret.IsConnectedToSharedInstance {
		rns.Logf(rns.LogWarning,
			"Started rnsd version %s connected to another shared local instance, this is probably NOT what you want!",
			rnsdVersion)
	} else {
		rns.Logf(rns.LogNotice, "Started rnsd version %s", rnsdVersion)
	}

	if interactive {
		runGoInteractive(ret)
		return nil
	}

	for {
		// Match Python behaviour: `while True: time.sleep(1)`.
		time.Sleep(time.Second)
	}
}

func runGoInteractive(ret *rns.Reticulum) {
	// Python uses `code.interact(local=globals())` without a banner.
	// Keep Go output minimal and just provide a prompt.

	i := interp.New(interp.Options{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
	_ = i.Use(stdlib.Symbols)
	_ = i.Use(interp.Exports{
		"rnsd": map[string]reflect.Value{
			"Ret": reflect.ValueOf(ret),
		},
	})

	scanner := bufio.NewScanner(os.Stdin)
	var pending strings.Builder

	// Ensure Ctrl+C exits like Python's KeyboardInterrupt handler.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		fmt.Println()
		os.Exit(0)
	}()

	for {
		if pending.Len() == 0 {
			fmt.Print(">>> ")
		} else {
			fmt.Print("... ")
		}
		if !scanner.Scan() {
			fmt.Println()
			return
		}

		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if pending.Len() == 0 {
			switch strings.ToLower(trimmed) {
			case ":quit", ":q", "quit()", "exit()":
				return
			case ":help":
				fmt.Println("Commands:")
				fmt.Println("  :help            - show this message")
				fmt.Println("  :quit / :q       - exit")
				continue
			}
		}

		pending.WriteString(line)
		pending.WriteString("\n")

		v, err := i.Eval(pending.String())
		if err != nil {
			msg := err.Error()
			if strings.Contains(msg, "unexpected EOF") ||
				strings.Contains(msg, "expected '}'") ||
				strings.Contains(msg, "expected ')'") ||
				strings.Contains(msg, "expected ']'") {
				continue
			}
			fmt.Fprintln(os.Stderr, msg)
			pending.Reset()
			continue
		}

		pending.Reset()
		if v.IsValid() && v.Kind() != reflect.Invalid && v.Kind() != reflect.Func {
			fmt.Println(v.Interface())
		}
	}
}

func toInt(v any) int {
	switch val := v.(type) {
	case int:
		return val
	case int32:
		return int(val)
	case int64:
		return int(val)
	case uint:
		return int(val)
	case uint32:
		return int(val)
	case uint64:
		return int(val)
	case float32:
		return int(val)
	case float64:
		return int(val)
	default:
		return 0
	}
}

func toUint64(v any) uint64 {
	switch val := v.(type) {
	case uint64:
		return val
	case uint32:
		return uint64(val)
	case uint:
		return uint64(val)
	case int64:
		if val < 0 {
			return 0
		}
		return uint64(val)
	case int:
		if val < 0 {
			return 0
		}
		return uint64(val)
	case float64:
		if val < 0 {
			return 0
		}
		return uint64(val)
	case float32:
		if val < 0 {
			return 0
		}
		return uint64(val)
	default:
		return 0
	}
}

func toFloat(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case uint:
		return float64(val)
	case uint64:
		return float64(val)
	default:
		return 0
	}
}
