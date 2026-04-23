package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode"
	"unicode/utf8"

	rns "github.com/svanichkin/go-reticulum/rns"
)

const (
	appName          = "rnx"
	remoteExecGrace  = 2.0
	defaultSizeUnit  = "B"
	defaultBitSuffix = "b"
)

var (
	identity            *rns.Identity
	reticulum           *rns.Reticulum
	allowAll            bool
	allowedIdentityHash [][]byte

	link               *rns.Link
	linkIdentified     bool
	listenerDest       *rns.Destination
	currentProgress    float64
	responseTransferSz int64
	speed              float64
	stats              [][]float64
)

func main() {
	var (
		configDir string
		destHex   string
		command   string

		verbose int
		quiet   int

		printIdentity bool
		listenMode    bool
		identityPath  string
		interactive   bool
		noAnnounce    bool

		stdinStr  optionalStringFlag
		stdoutLim optionalIntFlag
		stderrLim optionalIntFlag

		allowed  multiString
		noAuth   bool
		noID     bool
		detailed bool
		mirror   bool

		timeout       float64
		resultTimeout optionalFloat64Flag
	)

	fs := flag.NewFlagSet("rnx", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.StringVar(&configDir, "config", "", "path to alternative Reticulum config directory")
	fs.StringVar(&destHex, "destination", "", "hexadecimal hash of the listener")
	fs.StringVar(&command, "command", "", "command to be execute")

	// Python argparse uses action='count', meaning users can do -v/-vv/--verbose --verbose etc.
	// Implement count-flags as bool-like flags that increment per occurrence.
	vc := countFlag{p: &verbose}
	qc := countFlag{p: &quiet}
	fs.Var(vc, "v", "increase verbosity (repeatable, eg -vv)")
	fs.Var(vc, "verbose", "increase verbosity (repeatable, eg --verbose --verbose)")
	fs.Var(qc, "q", "decrease verbosity (repeatable, eg -qq)")
	fs.Var(qc, "quiet", "decrease verbosity (repeatable, eg --quiet --quiet)")

	fs.BoolVar(&printIdentity, "p", false, "print identity and destination info and exit")
	fs.BoolVar(&printIdentity, "print-identity", false, "print identity and destination info and exit")
	fs.BoolVar(&listenMode, "l", false, "listen for incoming commands")
	fs.BoolVar(&listenMode, "listen", false, "listen for incoming commands")

	fs.StringVar(&identityPath, "i", "", "path to identity to use")
	fs.BoolVar(&interactive, "x", false, "enter interactive mode")
	fs.BoolVar(&interactive, "interactive", false, "enter interactive mode")

	fs.BoolVar(&noAnnounce, "b", false, "don't announce at program start")
	fs.BoolVar(&noAnnounce, "no-announce", false, "don't announce at program start")

	fs.Var(&allowed, "a", "accept from this identity")
	fs.BoolVar(&noAuth, "n", false, "accept commands from anyone")
	fs.BoolVar(&noAuth, "noauth", false, "accept commands from anyone")

	fs.BoolVar(&noID, "N", false, "don't identify to listener")
	fs.BoolVar(&noID, "noid", false, "don't identify to listener")

	fs.BoolVar(&detailed, "detailed", false, "show detailed result output")
	fs.BoolVar(&detailed, "d", false, "show detailed result output")

	fs.BoolVar(&mirror, "m", false, "mirror exit code of remote command")

	fs.Float64Var(&timeout, "w", rns.TransportPathRequestTimeout, "connect and request timeout before giving up")
	fs.Var(&resultTimeout, "W", "max result download time")

	fs.Var(&stdinStr, "stdin", "pass input to stdin")
	fs.Var(&stdoutLim, "stdout", "max size in bytes of returned stdout")
	fs.Var(&stderrLim, "stderr", "max size in bytes of returned stderr")

	var showVersion bool
	fs.BoolVar(&showVersion, "version", false, "show version and exit")

	var help bool
	fs.BoolVar(&help, "help", false, "show this help message and exit")
	fs.BoolVar(&help, "h", false, "show this help message and exit")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: rnx [-h] [destination] [command] [--config path] [-v] [-q] [-p] [-l] [-i identity] [-x] [-b] [-a allowed_hash] [-n] [-N] [-d] [-m] [-w seconds] [-W seconds] [--stdin stdin] [--stdout stdout] [--stderr stderr] [--version]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Reticulum Remote Execution Utility")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "options:")
		fs.SetOutput(os.Stderr)
		fs.PrintDefaults()
	}

	if err := fs.Parse(expandCountFlags(os.Args[1:])); err != nil {
		fs.Usage()
		msg := strings.TrimSpace(err.Error())
		if msg != "" && !strings.HasPrefix(msg, "flag provided but not defined") {
			fmt.Fprintf(os.Stderr, "rnx: error: %s\n", msg)
		} else if strings.HasPrefix(msg, "flag provided but not defined") {
			parts := strings.SplitN(msg, ":", 2)
			if len(parts) == 2 {
				unk := strings.TrimSpace(parts[1])
				fmt.Fprintf(os.Stderr, "rnx: error: unrecognized arguments: %s\n", unk)
			} else {
				fmt.Fprintf(os.Stderr, "rnx: error: %s\n", msg)
			}
		}
		os.Exit(2)
	}

	if help {
		fs.Usage()
		os.Exit(0)
	}

	// positional args: destination, command (like Python)
	if destHex == "" && fs.NArg() >= 1 {
		destHex = fs.Arg(0)
	}
	if command == "" && fs.NArg() >= 2 {
		command = fs.Arg(1)
	}

	if showVersion {
		fmt.Printf("rnx %s\n", rns.GetVersion())
		return
	}

	setupSignals()

	if listenMode || printIdentity {
		if err := listen(
			configDirOrNil(configDir),
			identityPathOrNil(identityPath),
			verbose,
			quiet,
			allowed,
			printIdentity,
			noAuth,
			noAnnounce,
		); err != nil {
			fmt.Fprintln(os.Stderr, "listen error:", err)
			os.Exit(1)
		}
		return
	}

	// normal/interactive mode
	if destHex != "" && command != "" {
		code := execute(
			configDirOrNil(configDir),
			identityPathOrNil(identityPath),
			verbose,
			quiet,
			detailed,
			mirror,
			noID,
			destHex,
			command,
			optionalStringValue(stdinStr),
			optionalIntValue(stdoutLim),
			optionalIntValue(stderrLim),
			timeout,
			resultTimeoutValue(resultTimeout),
			interactive,
		)
		if !interactive {
			if mirror {
				os.Exit(code)
			}
			os.Exit(0)
		}
	}

	if destHex != "" && interactive {
		reader := bufio.NewReader(os.Stdin)
		var lastCode int
		for {
			codePrefix := ""
			if lastCode != 0 {
				codePrefix = fmt.Sprintf("%d", lastCode)
			}
			fmt.Printf("%s> ", codePrefix)
			line, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println()
				os.Exit(0)
			}
			cmdLine := strings.TrimSpace(line)
			if cmdLine == "" {
				continue
			}
			if cmdLine == "exit" || cmdLine == "quit" {
				os.Exit(0)
			}
			if cmdLine == "clear" {
				fmt.Print("\033c")
				continue
			}
			lastCode = execute(
				configDirOrNil(configDir),
				identityPathOrNil(identityPath),
				verbose,
				quiet,
				detailed,
				mirror,
				noID,
				destHex,
				cmdLine,
				nil,
				optionalIntValue(stdoutLim),
				optionalIntValue(stderrLim),
				timeout,
				resultTimeoutValue(resultTimeout),
				true,
			)
		}
	}

	fmt.Println()
	fs.Usage()
	fmt.Println()
}

type countFlag struct{ p *int }

func (c countFlag) String() string {
	if c.p == nil {
		return "0"
	}
	return strconv.Itoa(*c.p)
}

// IsBoolFlag lets "-v" be specified without a value (like a boolean flag).
func (c countFlag) IsBoolFlag() bool { return true }

func (c countFlag) Set(s string) error {
	if c.p == nil {
		return nil
	}
	s = strings.TrimSpace(s)
	if s == "" || s == "true" {
		*c.p++
		return nil
	}
	if s == "false" {
		return nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("invalid count %q", s)
	}
	*c.p = v
	return nil
}

type optionalFloat64Flag struct {
	set bool
	val float64
}

func (o *optionalFloat64Flag) String() string {
	if o == nil || !o.set {
		return "None"
	}
	return pyFloat(o.val)
}

func (o *optionalFloat64Flag) Set(s string) error {
	if o == nil {
		return nil
	}
	s = strings.TrimSpace(s)
	if s == "" {
		o.set = true
		o.val = 0
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("invalid float %q", s)
	}
	o.set = true
	o.val = v
	return nil
}

type optionalIntFlag struct {
	set bool
	val int
}

func (o *optionalIntFlag) String() string {
	if o == nil || !o.set {
		return "None"
	}
	return strconv.Itoa(o.val)
}

func (o *optionalIntFlag) Set(s string) error {
	if o == nil {
		return nil
	}
	s = strings.TrimSpace(s)
	if s == "" {
		o.set = true
		o.val = 0
		return nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("invalid int %q", s)
	}
	o.set = true
	o.val = v
	return nil
}

type optionalStringFlag struct {
	set bool
	val string
}

func (o *optionalStringFlag) String() string {
	if o == nil || !o.set {
		return "None"
	}
	return o.val
}

func (o *optionalStringFlag) Set(s string) error {
	if o == nil {
		return nil
	}
	o.set = true
	o.val = s
	return nil
}

// expandCountFlags expands Python-style "-vvv" / "-qq" into "-v -v -v" / "-q -q".
// This keeps compatibility with the original rnx utility CLI.
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

func resultTimeoutValue(opt optionalFloat64Flag) float64 {
	if opt.set {
		return opt.val
	}
	return -1
}

func optionalIntValue(opt optionalIntFlag) *int {
	if !opt.set {
		return nil
	}
	v := opt.val
	return &v
}

func optionalStringValue(opt optionalStringFlag) *string {
	if !opt.set {
		return nil
	}
	v := opt.val
	return &v
}

// ---------- listen side ----------

func listen(
	configdir *string,
	identitypath *string,
	verbosity, quietness int,
	allowed multiString,
	printIdentity, disableAuth, disableAnnounce bool,
) error {
	logLevel := 3 + verbosity - quietness
	var err error
	reticulum, err = rns.NewReticulum(configdir, &logLevel, nil, nil, false, nil)
	if err != nil {
		return err
	}

	if identity == nil {
		if err := prepareIdentity(identitypath); err != nil {
			return err
		}
	}

	dest, err := rns.NewDestination(identity, rns.DestinationIN, rns.DestinationSINGLE, appName, "execute")
	if err != nil {
		return err
	}

	if printIdentity {
		fmt.Println("Identity     :", rns.PrettyHex(identity.Hash))
		fmt.Println("Listening on :", rns.PrettyHex(dest.Hash))
		return nil
	}

	if disableAuth {
		allowAll = true
	} else {
		if err := loadAllowedIdentities(allowed); err != nil {
			return err
		}
	}

	if len(allowedIdentityHash) < 1 && !disableAuth {
		fmt.Println("Warning: No allowed identities configured, rncx will not accept any commands!")
	}

	dest.SetLinkEstablishedCallback(commandLinkEstablished)

	if !allowAll {
		if err := dest.RegisterRequestHandler("command", executeReceivedCommand,
			rns.DestinationAllowList, allowedIdentityHash); err != nil {
			return err
		}
	} else {
		if err := dest.RegisterRequestHandler("command", executeReceivedCommand,
			rns.DestinationAllowAll, nil); err != nil {
			return err
		}
	}

	// Python uses RNS.log(..., level=3) by default (Notice).
	rns.Log("rnx listening for commands on "+rns.PrettyHex(dest.Hash), rns.LogNotice)

	if !disableAnnounce {
		dest.Announce(nil, false, nil, nil, true)
	}

	for {
		time.Sleep(time.Second)
	}
}

func commandLinkEstablished(l *rns.Link) {
	l.SetRemoteIdentifiedCallback(initiatorIdentified)
	l.SetLinkClosedCallback(commandLinkClosed)
	rns.Log("Command link "+l.String()+" established", rns.LogInfo)
}

func commandLinkClosed(l *rns.Link) {
	rns.Log("Command link "+l.String()+" closed", rns.LogInfo)
}

func initiatorIdentified(l *rns.Link, id *rns.Identity) {
	if id == nil {
		return
	}
	rns.Log("Initiator "+l.String()+" identified as "+rns.PrettyHex(id.Hash), rns.LogInfo)
	if !allowAll && !inHashList(id.Hash, allowedIdentityHash) {
		rns.Log("Identity "+rns.PrettyHex(id.Hash)+" not allowed, tearing down link", rns.LogInfo)
		l.Teardown()
	}
}

// request handler (listen side)
func executeReceivedCommand(path string, data any, requestID []byte, _ []byte, remoteIdentity *rns.Identity, requestedAt time.Time) any {
	result := []any{
		false,        // 0: executed
		nil,          // 1: return code
		nil,          // 2: stdout
		nil,          // 3: stderr
		nil,          // 4: total stdout len
		nil,          // 5: total stderr len
		nowSeconds(), // 6: started
		nil,          // 7: concluded
	}

	args, ok := data.([]any)
	if !ok || len(args) < 5 {
		rns.Log("Received malformed command request", rns.LogWarning)
		return result
	}

	cmdBytes := bytesFromAny(args[0])
	if len(cmdBytes) == 0 {
		rns.Log("Received empty command in request", rns.LogWarning)
		return result
	}
	if !utf8.Valid(cmdBytes) {
		rns.Log("Received non-UTF-8 command in request", rns.LogWarning)
		return result
	}
	timeout := floatFromAny(args[1])
	oLimit := optionalInt(args[2])
	eLimit := optionalInt(args[3])
	stdinArg := args[4]
	stdinBytes := bytesFromAny(stdinArg)

	command := string(cmdBytes)
	var remoteIDStr string
	if remoteIdentity != nil {
		remoteIDStr = rns.PrettyHex(remoteIdentity.Hash)
	}
	if remoteIDStr != "" {
		rns.Log("Executing command ["+command+"] for "+remoteIDStr, rns.LogInfo)
	} else {
		rns.Log("Executing command ["+command+"] for unknown requestor", rns.LogInfo)
	}

	cmdArgs, err := splitCommand(command)
	if err != nil || len(cmdArgs) == 0 {
		rns.Logf(rns.LogError, "Could not parse command %q: %v", command, err)
		return result
	}

	ctx := context.Background()
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(timeout*float64(time.Second)))
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	if stdinArg != nil {
		cmd.Stdin = bytes.NewReader(stdinBytes)
	}
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		rns.Logf(rns.LogError, "Could not start command %q: %v", command, err)
		return result
	}
	result[0] = true

	_ = cmd.Wait()
	stdout := stdoutBuf.Bytes()
	stderr := stderrBuf.Bytes()
	timedOut := errors.Is(ctx.Err(), context.DeadlineExceeded)

	if !timedOut {
		result[7] = nowSeconds()
	} else {
		rns.Log("Command ["+command+"] timed out and was terminated", rns.LogWarning)
	}

	if cmd.ProcessState != nil {
		result[1] = cmd.ProcessState.ExitCode()
	} else {
		result[1] = -1
	}

	// stdout
	if oLimit != nil && *oLimit >= 0 && len(stdout) > *oLimit {
		if *oLimit == 0 {
			result[2] = []byte{}
		} else {
			result[2] = stdout[:*oLimit]
		}
	} else {
		result[2] = stdout
	}

	// stderr
	if eLimit != nil && *eLimit >= 0 && len(stderr) > *eLimit {
		if *eLimit == 0 {
			result[3] = []byte{}
		} else {
			result[3] = stderr[:*eLimit]
		}
	} else {
		result[3] = stderr
	}

	result[4] = len(stdout)
	result[5] = len(stderr)

	if remoteIDStr != "" {
		rns.Log("Delivering result of command ["+command+"] to "+remoteIDStr, rns.LogInfo)
	} else {
		rns.Log("Delivering result of command ["+command+"] to unknown requestor", rns.LogInfo)
	}

	return result
}

// ---------- execute side ----------

func execute(
	configdir *string,
	identitypath *string,
	verbosity, quietness int,
	detailed, mirror, noID bool,
	destHex, cmd string,
	stdin *string,
	stdoutLim, stderrLim *int,
	timeout, resultTimeout float64,
	interactive bool,
) int {
	destHash, err := parseDest(destHex)
	if err != nil {
		fmt.Println(err)
		os.Exit(241)
	}

	if reticulum == nil {
		logLevel := 3 + verbosity - quietness
		var err error
		reticulum, err = rns.NewReticulum(configdir, &logLevel, nil, nil, false, nil)
		if err != nil {
			fmt.Println("Could not initialise Reticulum:", err)
			os.Exit(241)
		}
	}

	if identity == nil {
		if err := prepareIdentity(identitypath); err != nil {
			fmt.Println(err)
			os.Exit(241)
		}
	}

	if !rns.HasPath(destHash) {
		rns.RequestPath(destHash, nil, nil, false)
		ok := spin(func() bool { return rns.HasPath(destHash) },
			"Path to "+rns.PrettyHex(destHash)+" requested",
			timeout)
		if !ok {
			fmt.Println("Path not found")
			os.Exit(242)
		}
	}

	if listenerDest == nil {
		li := rns.IdentityRecall(destHash)
		if li == nil {
			fmt.Println("Could not recall listener identity")
			os.Exit(243)
		}
		var err error
		listenerDest, err = rns.NewDestination(li, rns.DestinationOUT, rns.DestinationSINGLE, appName, "execute")
		if err != nil {
			fmt.Println("Could not create listener destination:", err)
			os.Exit(243)
		}
	}

	if link == nil || link.Status == rns.LinkClosed || link.Status == rns.LinkPending {
		var err error
		link, err = rns.NewLink(listenerDest, nil, rns.LinkModeDefault, nil, nil)
		if err != nil {
			fmt.Println("Could not create link:", err)
			os.Exit(243)
		}
		linkIdentified = false
	}

	ok := spin(func() bool { return link.Status == rns.LinkActive },
		"Establishing link with "+rns.PrettyHex(destHash),
		timeout)
	if !ok {
		fmt.Println("Could not establish link with " + rns.PrettyHex(destHash))
		os.Exit(243)
	}

	if !noID && !linkIdentified {
		link.Identify(identity)
		linkIdentified = true
		// Ensure the identify packet is sent before the request packet.
		// Python sends identify() immediately before request(), and callers expect
		// allow-list listeners to reliably see the remote identity in time.
		time.Sleep(100 * time.Millisecond)
	}

	var stdinBytes any
	if stdin != nil {
		stdinBytes = []byte(*stdin)
	}

	reqData := []any{
		[]byte(cmd),
		timeout,
		limitValue(stdoutLim),
		limitValue(stderrLim),
		stdinBytes,
	}

	rtt := link.RTT.Seconds()
	rexecTimeout := timeout + rtt*4 + remoteExecGrace

	result := link.Request("command", reqData, remoteExecutionDone, remoteExecutionDone, remoteExecutionProgress, rexecTimeout)
	receipt, _ := result.(*rns.RequestReceipt)
	if receipt == nil {
		fmt.Println("Could not request remote execution")
		if interactive {
			return 0
		}
		os.Exit(244)
	}

	spin(
		func() bool {
			st := receipt.Status()
			return link.Status == rns.LinkClosed ||
				(st != rns.ReceiptFailed && st != rns.ReceiptSent)
		},
		"Sending execution request",
		rexecTimeout+0.5,
	)

	if link.Status == rns.LinkClosed {
		fmt.Println("Could not request remote execution, link was closed")
		if interactive {
			return 0
		}
		os.Exit(244)
	}

	if receipt.Status() == rns.ReceiptFailed {
		fmt.Println("Could not request remote execution")
		if interactive {
			return 0
		}
		os.Exit(244)
	}

	spin(
		func() bool { return receipt.Status() != rns.ReceiptDelivered },
		"Command delivered, awaiting result",
		timeout,
	)

	if receipt.Status() == rns.ReceiptFailed {
		fmt.Println("No result was received")
		if interactive {
			return 0
		}
		os.Exit(245)
	}

	spinStat(func() bool { return receipt.Status() != rns.ReceiptReceiving }, resultTimeout)

	if receipt.Status() == rns.ReceiptFailed {
		fmt.Println("Receiving result failed")
		if interactive {
			return 0
		}
		os.Exit(246)
	}

	resp := receipt.Response()
	if resp == nil {
		fmt.Println("No response")
		if interactive {
			return 0
		}
		os.Exit(249)
	}

	resultFields, ok := resp.([]any)
	if !ok || len(resultFields) < 8 {
		fmt.Println("Received invalid result")
		if interactive {
			return 0
		}
		os.Exit(247)
	}

	executed, _ := resultFields[0].(bool)
	retval, hasRetval := intFromAny(resultFields[1])
	stdoutBytes := bytesFromAny(resultFields[2])
	stderrBytes := bytesFromAny(resultFields[3])
	outLen, _ := intFromAny(resultFields[4])
	errLen, _ := intFromAny(resultFields[5])
	started := floatFromAny(resultFields[6])
	concluded := floatFromAny(resultFields[7])

	if !executed {
		fmt.Println("Remote could not execute command")
		if interactive {
			return 0
		}
		os.Exit(248)
	}

	if detailed {
		if len(stdoutBytes) > 0 {
			fmt.Print(string(stdoutBytes))
		}
		if len(stderrBytes) > 0 {
			fmt.Fprint(os.Stderr, string(stderrBytes))
		}
		os.Stdout.Sync()
		os.Stderr.Sync()

		fmt.Println("\n--- End of remote output, rnx done ---")
		if started != 0 && concluded != 0 {
			cmdDur := roundFloat(concluded-started, 3)
			fmt.Printf("Remote command execution took %s seconds\n", pyFloat(cmdDur))

			totalSize := receipt.ResponseSize()
			if rs := receipt.RequestSize(); rs > 0 {
				totalSize += rs
			}
			tdur := roundFloat(receipt.ResponseConcludedAt()-receipt.SentAt()-cmdDur, 3)
			tdStr := ""
			if tdur == 1 {
				tdStr = " in 1 second"
			} else if tdur < 10 {
				tdStr = " in " + pyFloat(tdur) + " seconds"
			} else {
				tdStr = " in " + prettyTime(tdur, false)
			}
			spdStr := ""
			if tdur > 0 {
				spdStr = ", effective rate " + sizeStr(float64(totalSize)/tdur, defaultBitSuffix) + "ps"
			}
			fmt.Printf("Transferred %s%s%s\n", sizeStr(float64(totalSize), defaultSizeUnit), tdStr, spdStr)
		}
		if outLen > 0 {
			tstr := ""
			if len(stdoutBytes) < outLen {
				tstr = fmt.Sprintf(", %d bytes displayed", len(stdoutBytes))
			}
			fmt.Printf("Remote wrote %d bytes to stdout%s\n", outLen, tstr)
		}
		if errLen > 0 {
			tstr := ""
			if len(stderrBytes) < errLen {
				tstr = fmt.Sprintf(", %d bytes displayed", len(stderrBytes))
			}
			fmt.Printf("Remote wrote %d bytes to stderr%s\n", errLen, tstr)
		}
	} else {
		if len(stdoutBytes) > 0 {
			fmt.Print(string(stdoutBytes))
		}
		if len(stderrBytes) > 0 {
			fmt.Fprint(os.Stderr, string(stderrBytes))
		}
		if ((stdoutLim == nil || *stdoutLim != 0) && len(stdoutBytes) < outLen) ||
			((stderrLim == nil || *stderrLim != 0) && len(stderrBytes) < errLen) {
			os.Stdout.Sync()
			os.Stderr.Sync()
			fmt.Println("\nOutput truncated before being returned:")
			if len(stdoutBytes) != 0 && len(stdoutBytes) < outLen {
				fmt.Printf("  stdout truncated to %d bytes\n", len(stdoutBytes))
			}
			if len(stderrBytes) != 0 && len(stderrBytes) < errLen {
				fmt.Printf("  stderr truncated to %d bytes\n", len(stderrBytes))
			}
		}
	}

	if !interactive {
		link.Teardown()
	}

	if mirror {
		if !hasRetval {
			return 240
		}
		return retval
	}
	return 0
}

// ---------- progress callbacks ----------

func remoteExecutionDone(rr *rns.RequestReceipt) {}

func remoteExecutionProgress(rr *rns.RequestReceipt) {
	const statsMax = 32
	currentProgress = rr.Progress()
	responseTransferSz = int64(rr.ResponseTransferSize())
	now := float64(time.Now().UnixNano()) / 1e9
	got := currentProgress * float64(responseTransferSz)
	entry := []float64{now, got}
	stats = append(stats, entry)
	if len(stats) > statsMax {
		stats = stats[1:]
	}
	span := now - stats[0][0]
	if span == 0 {
		speed = 0
	} else {
		diff := got - stats[0][1]
		speed = diff / span
	}
}

// ---------- utils ----------

type multiString []string

func (m *multiString) String() string     { return strings.Join(*m, ",") }
func (m *multiString) Set(s string) error { *m = append(*m, s); return nil }

func limitValue(lim *int) any {
	if lim == nil {
		return nil
	}
	return *lim
}

func optionalInt(v any) *int {
	if v == nil {
		return nil
	}
	var val int
	switch t := v.(type) {
	case int:
		val = t
	case uint:
		val = int(t)
	case int8:
		val = int(t)
	case int16:
		val = int(t)
	case int32:
		val = int(t)
	case int64:
		val = int(t)
	case uint8:
		val = int(t)
	case uint16:
		val = int(t)
	case uint32:
		val = int(t)
	case uint64:
		val = int(t)
	case float32:
		val = int(t)
	case float64:
		val = int(t)
	default:
		return nil
	}
	return &val
}

func floatFromAny(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case uint:
		return float64(t)
	case int64:
		return float64(t)
	case int32:
		return float64(t)
	case uint32:
		return float64(t)
	case uint64:
		return float64(t)
	default:
		return 0
	}
}

func intFromAny(v any) (int, bool) {
	if v == nil {
		return 0, false
	}
	switch t := v.(type) {
	case int:
		return t, true
	case uint:
		return int(t), true
	case int8:
		return int(t), true
	case uint8:
		return int(t), true
	case int16:
		return int(t), true
	case uint16:
		return int(t), true
	case int32:
		return int(t), true
	case uint32:
		return int(t), true
	case int64:
		return int(t), true
	case uint64:
		return int(t), true
	case float32:
		return int(t), true
	case float64:
		return int(t), true
	default:
		return 0, false
	}
}

func bytesFromAny(v any) []byte {
	switch t := v.(type) {
	case nil:
		return nil
	case []byte:
		return t
	case string:
		return []byte(t)
	default:
		return nil
	}
}

func nowSeconds() float64 {
	return float64(time.Now().UnixNano()) / 1e9
}

func prepareIdentity(identityPath *string) error {
	var path string
	if identityPath != nil && *identityPath != "" {
		path = *identityPath
	} else {
		if reticulum != nil && reticulum.IdentityPath != "" {
			path = filepath.Join(reticulum.IdentityPath, appName)
		} else {
			home, _ := os.UserHomeDir()
			path = filepath.Join(home, ".reticulum", "storage", "identities", appName)
		}
	}
	if fileExists(path) {
		id, err := rns.IdentityFromFile(path)
		if err == nil {
			identity = id
		}
	}
	if identity == nil {
		rns.Log("No valid saved identity found, creating new...", rns.LogInfo)
		id, err := rns.NewIdentity()
		if err != nil {
			return err
		}
		identity = id
		if err := identity.Save(path); err != nil {
			return err
		}
	}
	return nil
}

func loadAllowedIdentities(allowed multiString) error {
	destLen := (rns.ReticulumTruncatedHashLength / 8) * 2
	for _, a := range allowed {
		if len(a) != destLen {
			return fmt.Errorf("Allowed destination length is invalid, must be %d hexadecimal characters (%d bytes).", destLen, destLen/2)
		}
		b, err := hex.DecodeString(a)
		if err != nil {
			return fmt.Errorf("Invalid destination entered. Check your input.")
		}
		allowedIdentityHash = append(allowedIdentityHash, b)
	}

	// + allowed_identities file
	fileName := "allowed_identities"
	paths := []string{
		"/etc/rnx/" + fileName,
		"~/.config/rnx/" + fileName,
		"~/.rnx/" + fileName,
	}
	for _, p := range paths {
		p = expandUser(p)
		if fileExists(p) {
			data, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			lines := strings.Split(strings.ReplaceAll(string(data), "\r", ""), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if len(line) == destLen {
					if b, err := hex.DecodeString(line); err == nil {
						allowedIdentityHash = append(allowedIdentityHash, b)
					}
				}
			}
		}
	}
	return nil
}

func inHashList(h []byte, list [][]byte) bool {
	for _, x := range list {
		if string(x) == string(h) {
			return true
		}
	}
	return false
}

func parseDest(s string) ([]byte, error) {
	destLen := (rns.ReticulumTruncatedHashLength / 8) * 2
	if len(s) != destLen {
		return nil, fmt.Errorf("Allowed destination length is invalid, must be %d hexadecimal characters (%d bytes).", destLen, destLen/2)
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("Invalid destination entered. Check your input.")
	}
	return b, nil
}

func configDirOrNil(path string) *string {
	if path == "" {
		return nil
	}
	return &path
}

func identityPathOrNil(path string) *string {
	if path == "" {
		return nil
	}
	return &path
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func expandUser(path string) string {
	if strings.HasPrefix(path, "~") {
		home, _ := os.UserHomeDir()
		if path == "~" {
			return home
		}
		if len(path) > 1 && (path[1] == '/' || path[1] == '\\') {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func splitCommand(input string) ([]string, error) {
	// Best-effort shlex.split() compatibility (POSIX mode):
	// - Whitespace splits tokens unless inside quotes.
	// - Single quotes preserve everything until next single quote.
	// - Double quotes preserve everything until next double quote, but allow backslash
	//   escaping of: \\ \" \\$ \\` and newline.
	// - Backslash outside quotes escapes next character.
	runes := []rune(input)
	var (
		args    []string
		current strings.Builder
		mode    rune // 0, '\'', '"'
	)

	flush := func() {
		if current.Len() == 0 {
			return
		}
		args = append(args, current.String())
		current.Reset()
	}

	for i := 0; i < len(runes); i++ {
		r := runes[i]

		switch mode {
		case '\'':
			if r == '\'' {
				mode = 0
				continue
			}
			current.WriteRune(r)
			continue

		case '"':
			if r == '"' {
				mode = 0
				continue
			}
			if r == '\\' {
				if i+1 >= len(runes) {
					return nil, errors.New("unterminated escape in command")
				}
				next := runes[i+1]
				switch next {
				case '\\', '"', '$', '`':
					current.WriteRune(next)
					i++
					continue
				case '\n':
					// swallow escaped newline
					i++
					continue
				default:
					// keep the backslash if it doesn't escape a special char
					current.WriteRune('\\')
					current.WriteRune(next)
					i++
					continue
				}
			}
			current.WriteRune(r)
			continue
		}

		// mode == 0 (unquoted)
		switch {
		case unicode.IsSpace(r):
			flush()
		case r == '\'' || r == '"':
			mode = r
		case r == '\\':
			if i+1 >= len(runes) {
				return nil, errors.New("unterminated escape in command")
			}
			current.WriteRune(runes[i+1])
			i++
		default:
			current.WriteRune(r)
		}
	}

	if mode != 0 {
		return nil, errors.New("unterminated quote in command")
	}
	flush()
	return args, nil
}

// spinner
func spin(until func() bool, msg string, timeout float64) bool {
	i := 0
	syms := []rune("⢄⢂⢁⡁⡈⡐⡠")
	var deadline time.Time
	if timeout > 0 {
		deadline = time.Now().Add(time.Duration(timeout * float64(time.Second)))
	}
	fmt.Print(msg + "  ")
	for (timeout == 0 || time.Now().Before(deadline)) && !until() {
		time.Sleep(100 * time.Millisecond)
		fmt.Printf("\b\b%c ", syms[i])
		i = (i + 1) % len(syms)
	}
	fmt.Printf("\r%s  \r", strings.Repeat(" ", len(msg)))
	if timeout > 0 && time.Now().After(deadline) && !until() {
		return false
	}
	return true
}

func spinStat(until func() bool, timeout float64) bool {
	i := 0
	syms := []rune("⢄⢂⢁⡁⡈⡐⡠")
	var deadline time.Time
	if timeout >= 0 {
		deadline = time.Now().Add(time.Duration(timeout * float64(time.Second)))
	}
	for (timeout < 0 || time.Now().Before(deadline)) && !until() {
		time.Sleep(100 * time.Millisecond)
		prg := currentProgress
		percent := roundFloat(prg*100.0, 1)
		stat := fmt.Sprintf("%.1f%% - %s of %s - %sps",
			percent,
			sizeStr(prg*float64(responseTransferSz), defaultSizeUnit),
			sizeStr(float64(responseTransferSz), defaultSizeUnit),
			sizeStr(speed, defaultBitSuffix))
		fmt.Printf("\r                                                                                  \rReceiving result %c %s ", syms[i], stat)
		i = (i + 1) % len(syms)
	}
	fmt.Print("\r                                                                                  \r")
	if timeout >= 0 && time.Now().After(deadline) && !until() {
		return false
	}
	return true
}

func sizeStr(num float64, suffix string) string {
	units := []string{"", "K", "M", "G", "T", "P", "E", "Z"}
	lastUnit := "Y"
	val := num
	if suffix == "b" {
		val *= 8
	}
	for _, u := range units {
		if abs(val) < 1000.0 {
			if u == "" {
				return fmt.Sprintf("%.0f %s", val, suffix)
			}
			return fmt.Sprintf("%.2f %s%s", val, u, suffix)
		}
		val /= 1000.0
	}
	return fmt.Sprintf("%.2f%s%s", val, lastUnit, suffix)
}

func prettyTime(sec float64, verbose bool) string {
	days := int(sec / (24 * 3600))
	sec = math.Mod(sec, 24*3600)
	hours := int(sec / 3600)
	sec = math.Mod(sec, 3600)
	minutes := int(sec / 60)
	sec = math.Mod(sec, 60)
	seconds := roundFloat(sec, 2)

	var parts []string
	if days > 0 {
		if verbose {
			parts = append(parts, fmt.Sprintf("%d day%s", days, plural(days)))
		} else {
			parts = append(parts, fmt.Sprintf("%dd", days))
		}
	}
	if hours > 0 {
		if verbose {
			parts = append(parts, fmt.Sprintf("%d hour%s", hours, plural(hours)))
		} else {
			parts = append(parts, fmt.Sprintf("%dh", hours))
		}
	}
	if minutes > 0 {
		if verbose {
			parts = append(parts, fmt.Sprintf("%d minute%s", minutes, plural(minutes)))
		} else {
			parts = append(parts, fmt.Sprintf("%dm", minutes))
		}
	}
	if seconds > 0 {
		if verbose {
			parts = append(parts, fmt.Sprintf("%s second%s", pyFloat(seconds), pluralFloat(seconds)))
		} else {
			parts = append(parts, fmt.Sprintf("%ss", pyFloat(seconds)))
		}
	}

	if len(parts) == 0 {
		return "0s"
	}

	// Match Python pretty_time(): always uses ", " and " and " between parts.
	tstr := ""
	for i, p := range parts {
		switch {
		case i == 0:
		case i < len(parts)-1:
			tstr += ", "
		default:
			tstr += " and "
		}
		tstr += p
	}
	return tstr
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func roundFloat(v float64, decimals int) float64 {
	if decimals <= 0 {
		return math.Round(v)
	}
	f := math.Pow(10, float64(decimals))
	return math.Round(v*f) / f
}

// pyFloat mimics Python's str(float) for rnexec outputs:
// - no trailing zeros except keeping at least one decimal for integral floats (eg "1.0").
func pyFloat(v float64) string {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
}

func pluralFloat(v float64) string {
	// Python: "" if x == 1 else "s"
	if v == 1 {
		return ""
	}
	return "s"
}

func setupSignals() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for range ch {
			if link != nil {
				link.Teardown()
			}
			os.Exit(0)
		}
	}()
}
