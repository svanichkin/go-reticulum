package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	rns "github.com/svanichkin/go-reticulum/rns"
)

const exampleConfig = `# This is an example Identity Resolver file.
`

func programSetup(configDir string, verbosity, quietness int, service bool) error {
	targetVerbosity := verbosity - quietness

	var logDest any = rns.LogStdout
	var verbosityPtr *int = &targetVerbosity

	if service {
		logDest = rns.LogFile
		verbosityPtr = nil
	}

	var cfgPtr *string
	if configDir != "" {
		cfgPtr = &configDir
	}

	_, err := rns.NewReticulum(cfgPtr, nil, logDest, verbosityPtr, false, nil)
	if err == nil && service {
		// Ensure the logfile is created for service mode.
		rns.Log("rnir service started", rns.LogInfo)
		if cfgPtr != nil && strings.TrimSpace(*cfgPtr) != "" {
			_ = os.WriteFile(filepath.Join(*cfgPtr, "logfile"), []byte{}, 0o600)
		}
	}
	return err
}

type countFlag int

func (c *countFlag) String() string   { return fmt.Sprint(int(*c)) }
func (c *countFlag) IsBoolFlag() bool { return true }
func (c *countFlag) Set(s string) error {
	// Match Python argparse action='count': users can do -vvv / -qq.
	// flag may pass "true" when invoked as "-v=true".
	*c++
	return nil
}
func (c *countFlag) Value() int { return int(*c) }

// expandCountFlags expands Python-style "-vvv" / "-qq" into "-v -v -v" / "-q -q".
func expandCountFlags(args []string) []string {
	var out []string
	for _, a := range args {
		if len(a) >= 3 && a[0] == '-' && (len(a) < 2 || a[1] != '-') {
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

func main() {
	var (
		configDir   string
		verboseFlag countFlag
		quietFlag   countFlag
		exampleConf bool
		serviceMode bool
		showVersion bool
	)

	flag.StringVar(&configDir, "config", "", "path to alternative Reticulum config directory")
	flag.Var(&verboseFlag, "v", "increase verbosity")
	flag.Var(&verboseFlag, "verbose", "increase verbosity")
	flag.Var(&quietFlag, "q", "decrease verbosity")
	flag.Var(&quietFlag, "quiet", "decrease verbosity")
	flag.BoolVar(&exampleConf, "exampleconfig", false, "print verbose configuration example to stdout and exit")
	flag.BoolVar(&serviceMode, "service", false, "run as service (log to file, no verbosity)")
	flag.BoolVar(&showVersion, "version", false, "show version and exit")

	flag.CommandLine.Parse(expandCountFlags(os.Args[1:]))

	if showVersion {
		fmt.Printf("rnir %s\n", rns.Version())
		return
	}

	if exampleConf {
		fmt.Print(exampleConfig)
		return
	}

	if err := programSetup(configDir, verboseFlag.Value(), quietFlag.Value(), serviceMode); err != nil {
		// Match the other Go utilities: concise message and exit code 1.
		fmt.Fprintf(os.Stderr, "Could not start Reticulum: %v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}
