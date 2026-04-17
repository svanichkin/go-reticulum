package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	rns "github.com/svanichkin/go-reticulum/rns"
)

const (
	appName            = "rncp"
	reqFetchNotAllowed = 0xF0
	eraseStr           = "\033[2K\r"
	es                 = " "
)

var (
	allowAll                bool
	allowFetch              bool
	allowOverwriteOnReceive bool
	fetchAutoCompress       = true
	fetchJail               string
	savePath                string
	showPhyRates            bool
	maxAccepted             = -1
	acceptedMu              sync.Mutex

	allowedIdentityHashes [][]byte

	resourceDone    bool
	currentResource *rns.Resource
	statsMu         sync.Mutex
	stats           [][]float64 // [time, got, phyGot]
	speed           float64
	phySpeed        float64
	link            *rns.Link
)

func main() {
	var (
		helpFlag      = flag.Bool("h", false, "show this help message and exit")
		configDir     = flag.String("config", "", "path to alternative Reticulum config directory")
		identityPath  = flag.String("i", "", "path to identity to use")
		silent        = flag.Bool("S", false, "disable transfer progress output")
		listenMode    = flag.Bool("l", false, "listen for incoming transfer requests")
		noCompress    = flag.Bool("C", false, "disable automatic compression")
		allowFetchFlg = flag.Bool("F", false, "allow authenticated clients to fetch files")
		fetchMode     = flag.Bool("f", false, "fetch file from remote listener instead of sending")
		jail          = flag.String("j", "", "restrict fetch requests to specified path")
		save          = flag.String("s", "", "save received files in specified path")
		overwrite     = flag.Bool("O", false, "allow overwriting received files, instead of adding postfix")
		announce      = flag.Int("b", -1, "announce interval, 0 to only announce at startup")
		limitValue    = -1
		noAuth        = flag.Bool("n", false, "accept requests from anyone")
		printIdentity = flag.Bool("p", false, "print identity and destination info and exit")
		timeout       = flag.Float64("w", float64(rns.TransportPathRequestTimeout), "sender timeout before giving up")
		phyRates      = flag.Bool("P", false, "display physical layer transfer rates")
		versionFlag   = flag.Bool("version", false, "print version information and exit")
	)

	// Python argparse uses action='count', meaning users can do -vvv / -qq.
	// Expand grouped short flags before parsing.
	var verboseCnt countFlag
	var quietCnt countFlag
	flag.Var(&verboseCnt, "v", "increase verbosity")
	flag.Var(&verboseCnt, "verbose", "increase verbosity")
	flag.Var(&quietCnt, "q", "decrease verbosity")
	flag.Var(&quietCnt, "quiet", "decrease verbosity")

	// Long option aliases for parity with Python.
	flag.BoolVar(helpFlag, "help", *helpFlag, "show this help message and exit")
	flag.BoolVar(silent, "silent", *silent, "disable transfer progress output")
	flag.BoolVar(listenMode, "listen", *listenMode, "listen for incoming transfer requests")
	flag.BoolVar(noCompress, "no-compress", *noCompress, "disable automatic compression")
	flag.BoolVar(allowFetchFlg, "allow-fetch", *allowFetchFlg, "allow authenticated clients to fetch files")
	flag.BoolVar(fetchMode, "fetch", *fetchMode, "fetch file from remote listener instead of sending")
	flag.StringVar(jail, "jail", *jail, "restrict fetch requests to specified path")
	flag.StringVar(save, "save", *save, "save received files in specified path")
	flag.BoolVar(overwrite, "overwrite", *overwrite, "allow overwriting received files, instead of adding postfix")
	flag.BoolVar(noAuth, "no-auth", *noAuth, "accept requests from anyone")
	flag.BoolVar(printIdentity, "print-identity", *printIdentity, "print identity and destination info and exit")
	flag.BoolVar(phyRates, "phy-rates", *phyRates, "display physical layer transfer rates")
	flag.StringVar(identityPath, "identity", *identityPath, "path to identity to use")

	// NOTE: Python rncp.py keeps --limit commented out; keep parity by not exposing it.
	// flag.IntVar(&limitValue, "limit", -1, "maximum number of files to accept before exiting")
	// flag.IntVar(&limitValue, "L", -1, "alias for --limit")

	var allowed multiString
	flag.Var(&allowed, "a", "allow this identity (or add in ~/.rncp/allowed_identities)")

	flag.Usage = printUsage
	flag.CommandLine.Parse(expandCountFlags(os.Args[1:]))
	args := flag.Args()

	if *helpFlag {
		printUsage()
		return
	}

	if *versionFlag {
		fmt.Printf("rncp %s\n", rns.GetVersion())
		return
	}

	setupSignals()

	allowFetch = *allowFetchFlg
	showPhyRates = *phyRates
	allowOverwriteOnReceive = *overwrite

	if *listenMode || *printIdentity {
		err := listen(*configDir, *identityPath, verboseCnt.Value(), quietCnt.Value(), allowed, *printIdentity, &limitValue, *noAuth, *allowFetchFlg, *noCompress, *jail, *save, *announce, *overwrite)
		if err != nil {
			if ee, ok := asExitError(err); ok {
				os.Exit(ee.code)
			}
			fmt.Println("listen error:", err)
			os.Exit(1)
		}
		return
	}

	if len(args) < 2 {
		printUsage()
		return
	}

	file := args[0]
	dest := args[1]

	if *fetchMode {
		if err := fetch(*configDir, *identityPath, verboseCnt.Value(), quietCnt.Value(), dest, file, *timeout, *silent, *phyRates, *save, *overwrite); err != nil {
			if ee, ok := asExitError(err); ok {
				os.Exit(ee.code)
			}
			os.Exit(1)
		}
	} else {
		if err := send(*configDir, *identityPath, verboseCnt.Value(), quietCnt.Value(), dest, file, *timeout, *silent, *phyRates, *noCompress); err != nil {
			if ee, ok := asExitError(err); ok {
				os.Exit(ee.code)
			}
			os.Exit(1)
		}
	}
}

type countFlag int

func (c *countFlag) String() string   { return fmt.Sprint(int(*c)) }
func (c *countFlag) IsBoolFlag() bool { return true }
func (c *countFlag) Set(s string) error {
	s = strings.TrimSpace(s)
	if s == "" || s == "true" {
		*c++
		return nil
	}
	if s == "false" {
		return nil
	}
	return fmt.Errorf("invalid count %q", s)
}
func (c *countFlag) Value() int { return int(*c) }

// expandCountFlags expands Python-style "-vvv" / "-qq" into "-v -v -v" / "-q -q".
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

type exitError struct {
	code int
	err  error
}

func (e exitError) Error() string {
	if e.err == nil {
		return fmt.Sprintf("exit %d", e.code)
	}
	return e.err.Error()
}

func asExitError(err error) (exitError, bool) {
	var ee exitError
	if errors.As(err, &ee) {
		return ee, true
	}
	return exitError{}, false
}

// ----- listen -----

func listen(configdir, identityPath string, verbosity, quietness int, allowed multiString, displayIdentity bool,
	limit *int, disableAuth, fetchAllowed, noCompress bool,
	jail, save string, announce int, allowOverwrite bool) error {

	allowFetch = fetchAllowed
	fetchAutoCompress = !noCompress
	allowOverwriteOnReceive = allowOverwrite
	if limit != nil && *limit >= 0 {
		acceptedMu.Lock()
		maxAccepted = *limit
		acceptedMu.Unlock()
	}

	targetLogLevel := 3 + verbosity - quietness
	ret, err := rns.NewReticulum(&configdir, &targetLogLevel, nil, nil, false, nil)
	if err != nil {
		return err
	}

	if jail != "" {
		abs := absPath(jail)
		fetchJail = abs
		rns.Logf(rns.LogVerbose, "Restricting fetch requests to paths under %q", fetchJail)
	}

	if save != "" {
		sp := absPath(save)
		if !dirExists(sp) {
			rns.Log("Output directory not found", rns.LogError)
			os.Exit(3)
		}
		if !isWritable(sp) {
			rns.Log("Output directory not writable", rns.LogError)
			os.Exit(4)
		}
		savePath = sp
		rns.Logf(rns.LogVerbose, "Saving received files in %q", savePath)
	}

	identity, err := loadOrCreateIdentity(ret.IdentityPath, identityPath)
	if err != nil {
		return err
	}

	dest, err := rns.NewDestination(identity, rns.DestinationIN, rns.DestinationSINGLE, appName, "receive")
	if err != nil {
		return err
	}

	if displayIdentity {
		fmt.Println("Identity     :", rns.PrettyHex(identity.Hash))
		fmt.Println("Listening on :", rns.PrettyHex(dest.Hash()))
		os.Exit(0)
	}

	if disableAuth {
		allowAll = true
	} else {
		if err := loadAllowedIdentities(allowed); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	if len(allowedIdentityHashes) < 1 && !disableAuth {
		fmt.Println("Warning: No allowed identities configured, rncp will not accept any files!")
	}

	dest.SetLinkEstablishedCallback(clientLinkEstablished)

	if allowFetch {
		if allowAll {
			rns.Log("Allowing unauthenticated fetch requests", rns.LogWarning)
			_ = dest.RegisterRequestHandler("fetch_file", fetchRequest, rns.DestinationALLOW_ALL, nil)
		} else {
			_ = dest.RegisterRequestHandler("fetch_file", fetchRequest, rns.DestinationALLOW_LIST, allowedIdentityHashes)
		}
	}

	fmt.Println("rncp listening on", rns.PrettyHex(dest.Hash()))

	if announce >= 0 {
		go func() {
			dest.Announce(nil, false, nil, nil, true)
			if announce > 0 {
				for {
					time.Sleep(time.Duration(announce) * time.Second)
					dest.Announce(nil, false, nil, nil, true)
				}
			}
		}()
	}

	for {
		time.Sleep(time.Second)
	}
}

func fetchRequest(path string, data any, requestID []byte, linkID []byte, remoteIdentity *rns.Identity, requestedAt time.Time) any {
	if !allowFetch {
		return reqFetchNotAllowed
	}

	var reqPath string
	switch v := data.(type) {
	case []byte:
		reqPath = string(v)
	case string:
		reqPath = v
	default:
		return false
	}
	var filePath string

	if fetchJail != "" {
		if strings.HasPrefix(reqPath, fetchJail+"/") {
			reqPath = strings.TrimPrefix(reqPath, fetchJail+"/")
		}
		fp := absPath(filepath.Join(fetchJail, reqPath))
		if !strings.HasPrefix(fp, fetchJail+"/") && fp != fetchJail {
			rns.Logf(rns.LogWarning, "Disallowing fetch request for %s outside of fetch jail %s", fp, fetchJail)
			return reqFetchNotAllowed
		}
		filePath = fp
	} else {
		filePath = absPath(reqPath)
	}

	targetLink := findActiveLink(linkID)

	if !fileExists(filePath) {
		rns.Log("Client-requested file not found: "+filePath, rns.LogVerbose)
		return false
	}

	if targetLink == nil {
		return nil
	}

	rns.Log("Sending file "+filePath+" to client", rns.LogVerbose)
	meta := map[string][]byte{
		"name": []byte(filepath.Base(filePath)),
	}
	f, err := os.Open(filePath)
	if err != nil {
		rns.Logf(rns.LogError, "Could not open file to send: %v", err)
		return false
	}
	defer f.Close()

	_, err = startResourceTransfer(f, targetLink, meta, fetchAutoCompress, nil, nil)
	if err != nil {
		rns.Logf(rns.LogError, "Could not send file to client: %v", err)
		return false
	}
	return true
}

// ----- link & resource callbacks -----

func clientLinkEstablished(l *rns.Link) {
	rns.Log("Incoming link established", rns.LogVerbose)
	l.SetRemoteIdentifiedCallback(receiveSenderIdentified)
	l.SetResourceStrategy(rns.LinkAcceptApp)
	l.SetResourceCallback(receiveResourceAdvertisementCallback)
	l.SetResourceStartedCallback(receiveResourceStarted)
	l.SetResourceConcludedCallback(receiveResourceConcluded)
}

func receiveSenderIdentified(l *rns.Link, identity *rns.Identity) {
	if identity == nil {
		return
	}
	if inHashList(identity.Hash, allowedIdentityHashes) {
		rns.Log("Authenticated sender", rns.LogVerbose)
		return
	}
	if !allowAll {
		rns.Log("Sender not allowed, tearing down link", rns.LogVerbose)
		l.Teardown()
	}
}

func receiveResourceAdvertisementCallback(ad *rns.ResourceAdvertisement) bool {
	if allowAll {
		return true
	}
	if ad == nil || ad.Link == nil {
		return false
	}
	sender := ad.Link.RemoteIdentity()
	return sender != nil && inHashList(sender.Hash, allowedIdentityHashes)
}

func receiveResourceStarted(res *rns.Resource) {
	var idStr string
	if ri := res.Link().RemoteIdentity(); ri != nil {
		idStr = " from " + rns.PrettyHex(ri.Hash)
	}
	fmt.Println("Starting resource transfer " + rns.PrettyHex(res.Hash()) + idStr)
}

func receiveResourceConcluded(res *rns.Resource) {
	if res.Status() != rns.ResourceComplete {
		fmt.Println("Resource failed")
		return
	}
	fmt.Println(res.String(), "completed")

	filename, ok := resourceMetadataName(res.Metadata())
	if !ok {
		fmt.Println("Invalid data received, ignoring resource")
		return
	}

	var savedFilename string
	if savePath != "" {
		savedFilename = absPath(filepath.Join(savePath, filename))
		if !strings.HasPrefix(savedFilename, savePath+"/") && savedFilename != savePath {
			rns.Logf(rns.LogError, "Invalid save path %s, ignoring", savedFilename)
			return
		}
	} else {
		savedFilename = filename
	}

	full := savedFilename
	if allowOverwriteOnReceive {
		if fileExists(full) {
			if err := os.Remove(full); err != nil {
				rns.Logf(rns.LogError, "Could not overwrite existing file %s, renaming instead", full)
			}
		}
	}

	counter := 0
	for fileExists(full) {
		counter++
		full = fmt.Sprintf("%s.%d", savedFilename, counter)
	}

	if err := os.Rename(res.DataFile(), full); err != nil {
		rns.Logf(rns.LogError, "Could not move received file: %v", err)
	}
	noteAcceptedResource()
}

// ----- send / fetch -----

func send(configdir, identityPath string, verbosity, quietness int, destination, file string,
	timeout float64, silent, phyRates, noCompress bool) error {
	resetTransferState()

	targetLogLevel := 3 + verbosity - quietness
	showPhyRates = phyRates

	destHash, err := parseDest(destination)
	if err != nil {
		fmt.Println(err)
		return err
	}

	filePath := expandPath(file)
	if !fileExists(filePath) {
		fmt.Println("File not found")
		return errors.New("file not found")
	}

	meta := map[string][]byte{"name": []byte(filepath.Base(filePath))}
	fmt.Print(eraseStr)

	ret, err := rns.NewReticulum(&configdir, &targetLogLevel, nil, nil, false, nil)
	if err != nil {
		return err
	}

	identity, err := loadOrCreateIdentity(ret.IdentityPath, identityPath)
	if err != nil {
		return err
	}

	if !rns.HasPath(destHash) {
		rns.RequestPath(destHash, nil, nil, false)
		if silent {
			fmt.Println("Path to " + rns.PrettyHex(destHash) + " requested")
		} else {
			fmt.Print("Path to " + rns.PrettyHex(destHash) + " requested  " + es)
		}
	}

	syms := []rune("⢄⢂⢁⡁⡈⡐⡠")
	i := 0
	estabTimeout := time.Now().Add(time.Duration(timeout * float64(time.Second)))

	for !rns.HasPath(destHash) && time.Now().Before(estabTimeout) {
		if !silent {
			time.Sleep(100 * time.Millisecond)
			fmt.Printf("\b\b%c ", syms[i])
			i = (i + 1) % len(syms)
		}
	}

	if !rns.HasPath(destHash) {
		if silent {
			fmt.Println("Path not found")
		} else {
			fmt.Print(eraseStr + "Path not found\n")
		}
		return errors.New("path not found")
	}

	if silent {
		fmt.Println("Establishing link with " + rns.PrettyHex(destHash))
	} else {
		fmt.Print(eraseStr + "Establishing link with " + rns.PrettyHex(destHash) + "  " + es)
	}

	receiverID := rns.IdentityRecall(destHash)
	if receiverID == nil {
		// Python's rncp assumes that once a path exists the destination identity is also known.
		// In practice, identity propagation can lag path discovery slightly, so wait briefly.
		for receiverID == nil && time.Now().Before(estabTimeout) {
			time.Sleep(50 * time.Millisecond)
			receiverID = rns.IdentityRecall(destHash)
		}
	}
	if receiverID == nil {
		if silent {
			fmt.Println("Could not recall identity for " + rns.PrettyHex(destHash))
		} else {
			fmt.Print(eraseStr + "Could not recall identity for " + rns.PrettyHex(destHash) + "\n")
		}
		return errors.New("receiver identity not known")
	}
	receiverDest, err := rns.NewDestination(receiverID, rns.DestinationOUT, rns.DestinationSINGLE, appName, "receive")
	if err != nil {
		return err
	}

	link, err = rns.NewOutgoingLink(receiverDest, rns.LinkModeDefault, nil, nil)
	if err != nil {
		return err
	}
	for link.Status != rns.LinkActive && time.Now().Before(estabTimeout) {
		if !silent {
			time.Sleep(100 * time.Millisecond)
			fmt.Printf("\b\b%c ", syms[i])
			i = (i + 1) % len(syms)
		}
	}

	if time.Now().After(estabTimeout) {
		if silent {
			fmt.Println("Link establishment with " + rns.PrettyHex(destHash) + " timed out")
		} else {
			fmt.Print(eraseStr + "Link establishment with " + rns.PrettyHex(destHash) + " timed out\n")
		}
		return errors.New("link establishment timeout")
	}
	if !rns.HasPath(destHash) {
		if silent {
			fmt.Println("No path found to " + rns.PrettyHex(destHash))
		} else {
			fmt.Print(eraseStr + "No path found to " + rns.PrettyHex(destHash) + "\n")
		}
		return errors.New("no path")
	}

	if silent {
		fmt.Println("Advertising file resource...")
	} else {
		fmt.Print(eraseStr + "Advertising file resource  " + es)
	}

	link.Identify(identity)
	autoCompress := !noCompress

	f, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Could not start transfer: %v\n", err)
		return err
	}
	defer f.Close()

	res, err := startResourceTransfer(f, link, meta, autoCompress, senderProgress, senderProgress)
	if err != nil {
		fmt.Printf("Could not start transfer: %v\n", err)
		return err
	}
	currentResource = res

	for res.Status() < rns.ResourceTransferring {
		if !silent {
			time.Sleep(100 * time.Millisecond)
			fmt.Printf("\b\b%c ", syms[i])
			i = (i + 1) % len(syms)
		}
	}

	start := time.Now()

	if res.Status() > rns.ResourceComplete {
		if silent {
			fmt.Println("File was not accepted by " + rns.PrettyHex(destHash))
		} else {
			fmt.Print(eraseStr + "File was not accepted by " + rns.PrettyHex(destHash) + "\n")
		}
		return errors.New("not accepted")
	}

	if silent {
		fmt.Println("Transferring file...")
	} else {
		fmt.Print(eraseStr + "Transferring file  " + es)
	}

	symsIdx := 0
	for !resourceDone {
		if !silent {
			symsIdx = progressUpdate(syms, symsIdx, false)
		}
	}

	end := time.Now()
	transferTime := end.Sub(start).Seconds()
	if transferTime > 0 && currentResource != nil {
		speed = float64(currentResource.TotalSize()) / transferTime
	}

	if !silent {
		_ = progressUpdate(syms, symsIdx, true)
	}

	if currentResource == nil || currentResource.Status() != rns.ResourceComplete {
		if silent {
			fmt.Println("The transfer failed")
		} else {
			fmt.Print(eraseStr + "The transfer failed\n")
		}
		return errors.New("transfer failed")
	}

	if silent {
		fmt.Println(filePath + " copied to " + rns.PrettyHex(destHash))
	} else {
		fmt.Println("\n" + filePath + " copied to " + rns.PrettyHex(destHash))
	}
	link.Teardown()
	time.Sleep(250 * time.Millisecond)
	return nil
}

func fetch(configdir, identityPath string, verbosity, quietness int,
	destination, file string, timeout float64, silent, phyRates bool, save string, overwrite bool) error {
	resetTransferState()

	showPhyRates = phyRates
	allowOverwriteOnReceive = overwrite

	if save != "" {
		sp := absPath(save)
		if !dirExists(sp) {
			rns.Log("Output directory not found", rns.LogError)
			os.Exit(3)
		}
		if !isWritable(sp) {
			rns.Log("Output directory not writable", rns.LogError)
			os.Exit(4)
		}
		savePath = sp
	}

	destHash, err := parseDest(destination)
	if err != nil {
		fmt.Println(err)
		return err
	}

	targetLogLevel := 3 + verbosity - quietness
	ret, err := rns.NewReticulum(&configdir, &targetLogLevel, nil, nil, false, nil)
	if err != nil {
		return err
	}

	identity, err := loadOrCreateIdentity(ret.IdentityPath, identityPath)
	if err != nil {
		return err
	}

	if !rns.HasPath(destHash) {
		rns.RequestPath(destHash, nil, nil, false)
		if silent {
			fmt.Println("Path to " + rns.PrettyHex(destHash) + " requested")
		} else {
			fmt.Print("Path to " + rns.PrettyHex(destHash) + " requested  " + es)
		}
	}

	syms := []rune("⢄⢂⢁⡁⡈⡐⡠")
	i := 0
	estabTimeout := time.Now().Add(time.Duration(timeout * float64(time.Second)))

	for !rns.HasPath(destHash) && time.Now().Before(estabTimeout) {
		if !silent {
			time.Sleep(100 * time.Millisecond)
			fmt.Printf("\b\b%c ", syms[i])
			i = (i + 1) % len(syms)
		}
	}

	if !rns.HasPath(destHash) {
		if silent {
			fmt.Println("Path not found")
		} else {
			fmt.Print(eraseStr + "Path not found\n")
		}
		return errors.New("path not found")
	}

	if silent {
		fmt.Println("Establishing link with " + rns.PrettyHex(destHash))
	} else {
		fmt.Print(eraseStr + "Establishing link with " + rns.PrettyHex(destHash) + "  " + es)
	}

	listenerID := rns.IdentityRecall(destHash)
	if listenerID == nil {
		// See send(): identity may not be immediately available after path discovery.
		for listenerID == nil && time.Now().Before(estabTimeout) {
			time.Sleep(50 * time.Millisecond)
			listenerID = rns.IdentityRecall(destHash)
		}
	}
	if listenerID == nil {
		if silent {
			fmt.Println("Could not recall identity for " + rns.PrettyHex(destHash))
		} else {
			fmt.Print(eraseStr + "Could not recall identity for " + rns.PrettyHex(destHash) + "\n")
		}
		return errors.New("receiver identity not known")
	}
	listenerDest, err := rns.NewDestination(listenerID, rns.DestinationOUT, rns.DestinationSINGLE, appName, "receive")
	if err != nil {
		return err
	}

	link, err = rns.NewOutgoingLink(listenerDest, rns.LinkModeDefault, nil, nil)
	if err != nil {
		return err
	}
	for link.Status != rns.LinkActive && time.Now().Before(estabTimeout) {
		if !silent {
			time.Sleep(100 * time.Millisecond)
			fmt.Printf("\b\b%c ", syms[i])
			i = (i + 1) % len(syms)
		}
	}

	if !rns.HasPath(destHash) {
		if silent {
			fmt.Println("Could not establish link with " + rns.PrettyHex(destHash))
		} else {
			fmt.Print(eraseStr + "Could not establish link with " + rns.PrettyHex(destHash) + "\n")
		}
		return errors.New("link fail")
	}

	if silent {
		fmt.Println("Requesting file from remote...")
	} else {
		fmt.Print(eraseStr + "Requesting file from remote  " + es)
	}

	link.Identify(identity)

	requestResolved := false
	requestStatus := "unknown"
	resourceResolved := false
	currentTransferStarted := float64(0)

	requestResponse := func(rr *rns.RequestReceipt) {
		switch v := rr.Response().(type) {
		case bool:
			if !v {
				requestStatus = "not_found"
			} else {
				requestStatus = "found"
			}
		case nil:
			requestStatus = "remote_error"
		case int:
			if v == reqFetchNotAllowed {
				requestStatus = "fetch_not_allowed"
			} else {
				requestStatus = "found"
			}
		default:
			requestStatus = "found"
		}
		requestResolved = true
	}

	requestFailed := func(rr *rns.RequestReceipt) {
		requestStatus = "unknown"
		requestResolved = true
	}

	fetchResourceStarted := func(res *rns.Resource) {
		currentResource = res
		res.SetProgressCallback(senderProgress)
		if currentTransferStarted == 0 {
			currentTransferStarted = float64(time.Now().UnixNano()) / 1e9
		}
	}

	fetchResourceConcluded := func(res *rns.Resource) {
		defer func() { resourceResolved = true }()

		if res.Status() != rns.ResourceComplete {
			fmt.Println("Resource failed")
			return
		}

		filename, ok := resourceMetadataName(res.Metadata())
		if !ok {
			fmt.Println("Invalid data received, ignoring resource")
			return
		}

		var savedFilename string
		if savePath != "" {
			savedFilename = absPath(filepath.Join(savePath, filename))
			if !strings.HasPrefix(savedFilename, savePath+"/") && savedFilename != savePath {
				fmt.Printf("Invalid save path %s, ignoring\n", savedFilename)
				return
			}
		} else {
			savedFilename = filename
		}

		full := savedFilename
		if allowOverwriteOnReceive {
			if fileExists(full) {
				if err := os.Remove(full); err != nil {
					fmt.Printf("Could not overwrite existing file %s, renaming instead\n", full)
				}
			}
		}
		counter := 0
		for fileExists(full) {
			counter++
			full = fmt.Sprintf("%s.%d", savedFilename, counter)
		}

		if err := os.Rename(res.DataFile(), full); err != nil {
			fmt.Printf("An error occurred while saving received resource: %v\n", err)
			return
		}
	}

	link.SetResourceStrategy(rns.LinkAcceptAll)
	link.SetResourceStartedCallback(fetchResourceStarted)
	link.SetResourceConcludedCallback(fetchResourceConcluded)
	link.Request("fetch_file", file, requestResponse, requestFailed, nil, timeout)

	for !requestResolved {
		if !silent {
			time.Sleep(100 * time.Millisecond)
			fmt.Printf("\b\b%c ", syms[i])
			i = (i + 1) % len(syms)
		}
	}

	if !silent {
		fmt.Print(eraseStr)
	}

	switch requestStatus {
	case "fetch_not_allowed":
		fmt.Printf("Fetch request failed, fetching the file %s was not allowed by the remote\n", file)
		link.Teardown()
		time.Sleep(150 * time.Millisecond)
		return nil
	case "not_found":
		fmt.Printf("Fetch request failed, the file %s was not found on the remote\n", file)
		link.Teardown()
		time.Sleep(150 * time.Millisecond)
		return nil
	case "remote_error":
		fmt.Println("Fetch request failed due to an error on the remote system")
		link.Teardown()
		time.Sleep(150 * time.Millisecond)
		return nil
	case "unknown":
		fmt.Println("Fetch request failed due to an unknown error (probably not authorised)")
		link.Teardown()
		time.Sleep(150 * time.Millisecond)
		return nil
	case "found":
		// ok
	}

	i = 0
	for !resourceResolved {
		if !silent {
			time.Sleep(100 * time.Millisecond)
			if currentResource != nil {
				prg := currentResource.Progress()
				percent := prg * 100
				var phyStr string
				if showPhyRates {
					phyStr = fmt.Sprintf(" (%sps at physical layer)", sizeStr(int64(phySpeed), 'b'))
				}
				ps := sizeStr(int64(prg*float64(currentResource.TotalSize())), 'B')
				ts := sizeStr(int64(currentResource.TotalSize()), 'B')
				ss := sizeStr(int64(speed), 'b')
				if prg != 1.0 {
					fmt.Printf("%sTransferring file %c %.1f%% - %s of %s - %sps%s%s",
						eraseStr, syms[i], percent, ps, ts, ss, phyStr, es)
				} else {
					delta := float64(time.Now().UnixNano())/1e9 - currentTransferStarted
					if delta > 0 {
						speed = float64(currentResource.TotalSize()) / delta
					}
					dtStr := rns.PrettyTime(delta, false, false)
					ss = sizeStr(int64(speed), 'b')
					fmt.Printf("%sTransfer complete  %.1f%% - %s of %s in %s - %sps%s%s",
						eraseStr, percent, ps, ts, dtStr, ss, phyStr, es)
				}
			} else {
				fmt.Printf("%sWaiting for transfer to start %c %s", eraseStr, syms[i], es)
			}
			i = (i + 1) % len(syms)
		}
	}

	if currentResource == nil || currentResource.Status() != rns.ResourceComplete {
		if silent {
			fmt.Println("The transfer failed")
		} else {
			fmt.Print(eraseStr + "The transfer failed\n")
		}
		return errors.New("transfer failed")
	}

	if silent {
		fmt.Println(file + " fetched from " + rns.PrettyHex(destHash))
	} else {
		fmt.Println("\n" + file + " fetched from " + rns.PrettyHex(destHash))
	}
	link.Teardown()
	time.Sleep(100 * time.Millisecond)
	return nil
}

// ----- helpers -----

func senderProgress(res *rns.Resource) {
	statsMu.Lock()
	defer statsMu.Unlock()

	const statsMax = 32
	currentResource = res

	now := float64(time.Now().UnixNano()) / 1e9
	got := res.Progress() * float64(res.TotalSize())
	phyGot := res.SegmentProgress() * float64(res.GetTransferSize())

	entry := []float64{now, got, phyGot}
	stats = append(stats, entry)
	if len(stats) > statsMax {
		stats = stats[1:]
	}
	span := now - stats[0][0]
	if span == 0 {
		speed = 0
		phySpeed = 0
	} else {
		diff := got - stats[0][1]
		speed = diff / span
		phyDiff := phyGot - stats[0][2]
		if phyDiff > 0 {
			phySpeed = phyDiff / span
		}
	}
	if res.Status() < rns.ResourceComplete {
		resourceDone = false
	} else {
		resourceDone = true
	}
}

func progressUpdate(syms []rune, i int, done bool) int {
	time.Sleep(100 * time.Millisecond)
	if currentResource == nil {
		return i
	}
	prg := currentResource.Progress()
	percent := prg * 100
	var phyStr string
	if showPhyRates && !resourceDone {
		phyStr = fmt.Sprintf(" (%sps at physical layer)", sizeStr(int64(phySpeed), 'b'))
	}
	cs := sizeStr(int64(prg*float64(currentResource.TotalSize())), 'B')
	ts := sizeStr(int64(currentResource.TotalSize()), 'B')
	ss := sizeStr(int64(speed), 'b')
	stat := fmt.Sprintf("%.1f%% - %s of %s - %sps%s", percent, cs, ts, ss, phyStr)
	if !done {
		fmt.Printf("%sTransferring file %c %s%s", eraseStr, syms[i], stat, es)
	} else {
		fmt.Printf("%sTransfer complete  %s%s", eraseStr, stat, es)
	}
	i = (i + 1) % len(syms)
	return i
}

func startResourceTransfer(file *os.File, link *rns.Link, metadata map[string][]byte, autoCompress bool, progressCb func(*rns.Resource), concludedCb func(*rns.Resource)) (*rns.Resource, error) {
	if link == nil {
		return nil, errors.New("link is nil")
	}
	res, err := rns.NewResource(nil, file, link, metadata, true, autoCompress, concludedCb, progressCb, nil, 0, nil, nil, false, 0)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func resourceMetadataName(meta any) (string, bool) {
	var raw any
	switch m := meta.(type) {
	case map[string]any:
		raw = m["name"]
	case map[any]any:
		raw = m["name"]
	case map[string][]byte:
		raw = m["name"]
	}

	switch v := raw.(type) {
	case string:
		return filepath.Base(v), true
	case []byte:
		return filepath.Base(string(v)), true
	default:
		return "", false
	}
}

func findActiveLink(linkID []byte) *rns.Link {
	if len(linkID) == 0 {
		return nil
	}
	for _, l := range rns.TransportActiveLinks() {
		if l == nil || len(l.LinkID) == 0 {
			continue
		}
		if bytes.Equal(l.LinkID, linkID) {
			return l
		}
	}
	return nil
}

func loadAllowedIdentities(allowed multiString) error {
	destLen := (rns.ReticulumTruncatedHashLength / 8) * 2

	// from files
	fileName := "allowed_identities"
	var allowedFile string
	for _, p := range []string{
		"/etc/rncp/" + fileName,
		"~/.config/rncp/" + fileName,
		"~/.rncp/" + fileName,
	} {
		if fileExists(expandPath(p)) {
			allowedFile = expandPath(p)
			break
		}
	}

	if allowedFile != "" {
		data, err := os.ReadFile(allowedFile)
		if err != nil {
			// Python logs parsing errors but continues.
			rns.Logf(rns.LogError, "Error while parsing allowed_identities file %s: %v", allowedFile, err)
		} else {
			lines := strings.Split(strings.ReplaceAll(string(data), "\r", ""), "\n")
			var ali []string
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if len(line) == destLen {
					ali = append(ali, line)
				}
			}
			if len(ali) > 0 {
				if len(allowed) == 0 {
					allowed = ali
				} else {
					allowed = append(allowed, ali...)
				}
				ms := "ies"
				if len(ali) == 1 {
					ms = "y"
				}
				rns.Logf(rns.LogVerbose, "Loaded %d allowed identit%v from %s", len(ali), ms, allowedFile)
			}
		}
	}

	for _, a := range allowed {
		if len(a) != destLen {
			return fmt.Errorf("Allowed destination length is invalid, must be %d hexadecimal characters (%d bytes).", destLen, destLen/2)
		}
		b, err := hexDecode(a)
		if err != nil {
			return fmt.Errorf("Invalid destination entered. Check your input.")
		}
		allowedIdentityHashes = append(allowedIdentityHashes, b)
	}
	return nil
}

func parseDest(dest string) ([]byte, error) {
	destLen := (rns.ReticulumTruncatedHashLength / 8) * 2
	if len(dest) != destLen {
		return nil, fmt.Errorf("Allowed destination length is invalid, must be %d hexadecimal characters (%d bytes).", destLen, destLen/2)
	}
	b, err := hexDecode(dest)
	if err != nil {
		return nil, fmt.Errorf("Invalid destination entered. Check your input.")
	}
	return b, nil
}

func loadOrCreateIdentity(identityPathRoot, identityPath string) (*rns.Identity, error) {
	idPath := resolveIdentityPath(identityPathRoot, identityPath)
	if fileExists(idPath) {
		id, err := rns.IdentityFromFile(idPath)
		if err == nil {
			return id, nil
		}
		rns.Logf(rns.LogError, "Could not load identity for rncp. The identity file at %q may be corrupt or unreadable.", idPath)
		return nil, exitError{code: 2, err: errors.New("identity error")}
	}
	rns.Log("No valid saved identity found, creating new...", rns.LogInfo)
	id, err := rns.NewIdentity()
	if err != nil {
		return nil, err
	}
	if err := id.Save(idPath); err != nil {
		return nil, err
	}
	return id, nil
}

// small utils:

type multiString []string

func (m *multiString) String() string     { return strings.Join(*m, ",") }
func (m *multiString) Set(s string) error { *m = append(*m, s); return nil }

func setupSignals() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for range c {
			if currentResource != nil {
				currentResource.Cancel()
			}
			if link != nil {
				link.Teardown()
			}
			rns.Exit(0)
		}
	}()
}

func resolveIdentityPath(identityPathRoot, identityPath string) string {
	// Match Python rncp semantics:
	// - If no identity is specified, default to <identitypath>/<APP_NAME>.
	// - If specified, use it as-is (with "~" expansion), without forcing absolute paths.
	if identityPath == "" {
		return filepath.Join(identityPathRoot, appName)
	}
	return expandPath(identityPath)
}

func absPath(p string) string {
	// Python rncp uses os.path.abspath(os.path.expanduser(...)).
	expanded := expandPath(p)
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return filepath.Clean(expanded)
	}
	return filepath.Clean(abs)
}
func expandPath(p string) string {
	// Match Python's os.path.expanduser:
	// - "~" -> $HOME
	// - "~/" -> $HOME/...
	// We intentionally do not support "~user" expansion.
	if p == "~" {
		home, _ := os.UserHomeDir()
		if home == "" {
			return p
		}
		return home
	}
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		if home == "" {
			return p
		}
		return filepath.Join(home, p[2:])
	}
	return p
}
func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}
func isWritable(p string) bool {
	f, err := os.CreateTemp(p, ".test")
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(f.Name())
	return true
}
func inHashList(h []byte, lst [][]byte) bool {
	for _, x := range lst {
		if bytes.Equal(x, h) {
			return true
		}
	}
	return false
}
func hexDecode(s string) ([]byte, error) {
	return hex.DecodeString(strings.ToLower(s))
}

func noteAcceptedResource() {
	acceptedMu.Lock()
	defer acceptedMu.Unlock()
	if maxAccepted <= 0 {
		return
	}
	maxAccepted--
	if maxAccepted == 0 {
		fmt.Println("Accept limit reached, exiting")
		go func() {
			time.Sleep(250 * time.Millisecond)
			os.Exit(0)
		}()
	}
}

func resetTransferState() {
	statsMu.Lock()
	stats = nil
	statsMu.Unlock()
	resourceDone = false
	currentResource = nil
	speed = 0
	phySpeed = 0
}

func sizeStr(num int64, suffix rune) string {
	if suffix == 0 {
		suffix = 'B'
	}
	units := []string{"", "K", "M", "G", "T", "P", "E", "Z"}
	lastUnit := "Y"

	val := float64(num)
	if suffix == 'b' {
		val *= 8
	}
	for _, u := range units {
		if val < 1000.0 {
			if u == "" {
				return fmt.Sprintf("%.0f %c", val, suffix)
			}
			return fmt.Sprintf("%.2f %s%c", val, u, suffix)
		}
		val /= 1000.0
	}
	return fmt.Sprintf("%.2f%s%c", val, lastUnit, suffix)
}

func printUsage() {
	fmt.Println("")
	fmt.Println("Reticulum File Transfer Utility")
	fmt.Println("Usage:")
	fmt.Println("  rncp [options] file destination")
	fmt.Println("  rncp --listen [options]")
	fmt.Println("  rncp --fetch [options] file destination")
	fmt.Println("")
	fmt.Println("Options:")
	flag.CommandLine.PrintDefaults()
	fmt.Println("")
}
