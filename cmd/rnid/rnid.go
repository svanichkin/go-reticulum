package main

import (
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	rns "github.com/svanichkin/go-reticulum/rns"
)

const (
	appName   = "rnid"
	sigExt    = "rsg"
	encExt    = "rfe"
	chunkSize = 16 * 1024 * 1024
)

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

func spin(until func() bool, msg string, timeout time.Duration) bool {
	syms := []rune("⢄⢂⢁⡁⡈⡐⡠")
	i := 0
	var deadline time.Time
	if timeout > 0 {
		deadline = time.Now().Add(timeout)
	}

	fmt.Print(msg + "  ")
	for (timeout == 0 || time.Now().Before(deadline)) && !until() {
		time.Sleep(100 * time.Millisecond)
		fmt.Printf("\b\b%c ", syms[i])
		os.Stdout.Sync()
		i = (i + 1) % len(syms)
	}
	fmt.Printf("\r%s  \r", strings.Repeat(" ", len(msg)))

	if timeout > 0 && time.Now().After(deadline) && !until() {
		return false
	}
	return true
}

func main() {
	var (
		configDir      string
		identityStr    string
		generatePath   string
		importStr      string
		doExport       bool
		announce       string
		hashAs         string
		encryptPath    string
		decryptPath    string
		signPath       string
		validatePath   string
		readPath       string
		writePath      string
		force          bool
		useStdin       bool
		useStdout      bool
		requestUnknown bool
		timeoutSeconds = float64(rns.TransportPathRequestTimeout)
		printIdentity  bool
		printPrivate   bool
		useBase64      bool
		useBase32      bool
		showVersion    bool
	)

	flag.StringVar(&configDir, "config", "", "path to alternative Reticulum config directory")
	flag.StringVar(&identityStr, "identity", "", "hex identity or destination hash, or path to Identity file")
	flag.StringVar(&identityStr, "i", "", "hex identity or destination hash, or path to Identity file")
	flag.StringVar(&generatePath, "generate", "", "generate a new Identity and write to file")
	flag.StringVar(&generatePath, "g", "", "generate a new Identity and write to file")
	flag.StringVar(&importStr, "import", "", "import identity in hex, base32 or base64 format")
	flag.StringVar(&importStr, "m", "", "import identity in hex, base32 or base64 format")
	flag.BoolVar(&doExport, "export", false, "export identity to hex, base32 or base64")
	flag.BoolVar(&doExport, "x", false, "export identity to hex, base32 or base64")

	var verbose countFlag
	var quiet countFlag

	flag.StringVar(&announce, "announce", "", "announce a destination based on this Identity (app.aspect...)")
	flag.StringVar(&announce, "a", "", "announce a destination based on this Identity (app.aspect...)")
	flag.StringVar(&hashAs, "hash", "", "show destination hashes for other aspects for this Identity")
	flag.StringVar(&hashAs, "H", "", "show destination hashes for other aspects for this Identity")
	flag.StringVar(&encryptPath, "encrypt", "", "encrypt file")
	flag.StringVar(&encryptPath, "e", "", "encrypt file")
	flag.StringVar(&decryptPath, "decrypt", "", "decrypt file")
	flag.StringVar(&decryptPath, "d", "", "decrypt file")
	flag.StringVar(&signPath, "sign", "", "sign file")
	flag.StringVar(&signPath, "s", "", "sign file")
	flag.StringVar(&validatePath, "validate", "", "validate signature (path to .rsg file)")
	flag.StringVar(&validatePath, "V", "", "validate signature (path to .rsg file)")
	flag.StringVar(&readPath, "read", "", "input file path")
	flag.StringVar(&readPath, "r", "", "input file path")
	flag.StringVar(&writePath, "write", "", "output file path")
	flag.StringVar(&writePath, "w", "", "output file path")
	flag.BoolVar(&force, "force", false, "overwrite existing output files")
	flag.BoolVar(&force, "f", false, "overwrite existing output files")
	flag.BoolVar(&useStdin, "stdin", false, "read input from STDIN instead of file")
	flag.BoolVar(&useStdin, "I", false, "read input from STDIN instead of file")
	flag.BoolVar(&useStdout, "stdout", false, "write output to STDOUT instead of file")
	flag.BoolVar(&useStdout, "O", false, "write output to STDOUT instead of file")
	flag.BoolVar(&requestUnknown, "request", false, "request unknown Identities from the network")
	flag.BoolVar(&requestUnknown, "R", false, "request unknown Identities from the network")
	flag.Float64Var(&timeoutSeconds, "t", timeoutSeconds, "identity request timeout in seconds")
	flag.Float64Var(&timeoutSeconds, "timeout", timeoutSeconds, "identity request timeout in seconds")
	flag.BoolVar(&printIdentity, "print-identity", false, "print identity info and exit")
	flag.BoolVar(&printIdentity, "p", false, "print identity info and exit")
	flag.BoolVar(&printPrivate, "print-private", false, "allow displaying private keys")
	flag.BoolVar(&printPrivate, "P", false, "allow displaying private keys")
	flag.BoolVar(&useBase64, "base64", false, "Use base64-encoded input and output")
	flag.BoolVar(&useBase64, "b", false, "Use base64-encoded input and output")
	flag.BoolVar(&useBase32, "base32", false, "Use base32-encoded input and output")
	flag.BoolVar(&useBase32, "B", false, "Use base32-encoded input and output")
	flag.BoolVar(&showVersion, "version", false, "show version and exit")

	flag.Var(&verbose, "verbose", "increase verbosity")
	flag.Var(&verbose, "v", "increase verbosity")
	flag.Var(&quiet, "quiet", "decrease verbosity")
	flag.Var(&quiet, "q", "decrease verbosity")
	flag.CommandLine.Parse(expandCountFlags(os.Args[1:]))

	if showVersion {
		fmt.Printf("rnid %s\n", rns.GetVersion())
		return
	}

	// how many encrypt/decrypt/sign/validate operations are selected at once
	ops := 0
	if encryptPath != "" {
		ops++
	}
	if decryptPath != "" {
		ops++
	}
	if signPath != "" {
		ops++
	}
	if validatePath != "" {
		ops++
	}
	if ops > 1 {
		rns.Log("only one of --encrypt, --decrypt, --sign, --validate can be used at a time", rns.LogError)
		rns.Exit(1)
	}

	// if --read is not set, derive it from encrypt/decrypt/sign flags
	if readPath == "" {
		switch {
		case encryptPath != "":
			readPath = encryptPath
		case decryptPath != "":
			readPath = decryptPath
		case signPath != "":
			readPath = signPath
		}
	}

	// identity import
	if importStr != "" {
		importIdentity(importStr, useBase32, useBase64, &writePath, force, printPrivate)
		return
	}

	// if not generating and no identity is provided, show help
	if generatePath == "" && identityStr == "" {
		fmt.Println()
		fmt.Println("No identity provided, cannot continue")
		fmt.Println()
		flag.Usage()
		fmt.Println()
		rns.Exit(2)
	}

	// log level
	targetLogLevel := 4 + verbose.Value() - quiet.Value()
	var configPtr *string
	if configDir != "" {
		configPtr = &configDir
	}
	if _, err := rns.NewReticulum(configPtr, &targetLogLevel, nil, nil, false, nil); err != nil {
		rns.Log("Could not start Reticulum: "+err.Error(), rns.LogError)
		rns.Exit(101)
	}
	rns.SetCompactLogFormat(true)
	if useStdout {
		rns.SetLogLevel(-1)
	}

	// generate a new identity
	if generatePath != "" {
		if !force && fileExists(generatePath) {
			rns.Log("Identity file "+generatePath+" already exists. Not overwriting.", rns.LogError)
			rns.Exit(3)
		}
		id, err := rns.NewIdentity()
		if err != nil {
			rns.Log("An error occurred while generating a new Identity.", rns.LogError)
			rns.Log("The contained exception was: "+err.Error(), rns.LogError)
			rns.Exit(4)
		}
		if err := id.Save(generatePath); err != nil {
			rns.Log("An error occurred while saving the generated Identity.", rns.LogError)
			rns.Log("The contained exception was: "+err.Error(), rns.LogError)
			rns.Exit(4)
		}
		rns.Log(fmt.Sprintf("New identity %s written to %s", id.String(), generatePath), rns.LogNotice)
		return
	}

	// load/recall identity
	id := loadIdentity(identityStr, requestUnknown, time.Duration(timeoutSeconds*float64(time.Second)))

	// hash/announce/print/export operations
	if hashAs != "" {
		doHash(id, hashAs)
		return
	}
	if announce != "" {
		doAnnounce(id, announce)
		return
	}
	if printIdentity {
		printIdentityKeys(id, printPrivate, useBase32, useBase64)
		return
	}
	if doExport {
		exportIdentity(id, useBase32, useBase64)
		return
	}

	// next: file operations (encrypt/decrypt/sign/validate)
	doIO(id, encryptPath != "", decryptPath != "", signPath != "", validatePath,
		&readPath, &writePath, force, useStdin, useStdout)
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

// ---------- import/export/print ----------

func importIdentity(data string, useB32, useB64 bool, writePath *string, force, printPriv bool) {
	var raw []byte
	var err error

	switch {
	case useB64:
		raw, err = decodeIdentityBase64(data)
	case useB32:
		raw, err = base32.StdEncoding.DecodeString(data)
	default:
		raw, err = hex.DecodeString(data)
	}
	if err != nil {
		fmt.Println("Invalid identity data specified for import:", err)
		rns.Exit(41)
	}

	id, err := rns.IdentityFromBytes(raw)
	if err != nil {
		fmt.Println("Could not create Reticulum identity from specified data:", err)
		rns.Exit(42)
	}

	rns.Log("Identity imported", rns.LogNotice)
	pub := id.GetPublicKey()
	switch {
	case useB64:
		rns.Log("Public Key  : "+encodeIdentityBase64(pub), rns.LogNotice)
	case useB32:
		rns.Log("Public Key  : "+base32.StdEncoding.EncodeToString(pub), rns.LogNotice)
	default:
		rns.Log("Public Key  : "+rns.HexRep(pub, false), rns.LogNotice)
	}

	if len(id.GetPrivateKey()) > 0 {
		if printPriv {
			prv := id.GetPrivateKey()
			switch {
			case useB64:
				rns.Log("Private Key : "+encodeIdentityBase64(prv), rns.LogNotice)
			case useB32:
				rns.Log("Private Key : "+base32.StdEncoding.EncodeToString(prv), rns.LogNotice)
			default:
				rns.Log("Private Key : "+rns.HexRep(prv, false), rns.LogNotice)
			}
		} else {
			rns.Log("Private Key : Hidden", rns.LogNotice)
		}
	}

	if writePath != nil && *writePath != "" {
		wp := expandUser(*writePath)
		if !fileExists(wp) || force {
			if err := id.Save(wp); err != nil {
				fmt.Println("Error while writing imported identity to file:", err)
				rns.Exit(44)
			}
			rns.Log("Wrote imported identity to "+*writePath, rns.LogNotice)
		} else {
			fmt.Println("File", wp, "already exists, not overwriting")
			rns.Exit(43)
		}
	}

	rns.Exit(0)
}

func printIdentityKeys(id *rns.Identity, printPriv, useB32, useB64 bool) {
	pub := id.GetPublicKey()
	switch {
	case useB64:
		rns.Log("Public Key  : "+encodeIdentityBase64(pub), rns.LogNotice)
	case useB32:
		rns.Log("Public Key  : "+base32.StdEncoding.EncodeToString(pub), rns.LogNotice)
	default:
		rns.Log("Public Key  : "+rns.HexRep(pub, false), rns.LogNotice)
	}

	if len(id.GetPrivateKey()) > 0 {
		if printPriv {
			prv := id.GetPrivateKey()
			switch {
			case useB64:
				rns.Log("Private Key : "+encodeIdentityBase64(prv), rns.LogNotice)
			case useB32:
				rns.Log("Private Key : "+base32.StdEncoding.EncodeToString(prv), rns.LogNotice)
			default:
				rns.Log("Private Key : "+rns.HexRep(prv, false), rns.LogNotice)
			}
		} else {
			rns.Log("Private Key : Hidden", rns.LogNotice)
		}
	}
}

// export: always include the private key, like Python
func exportIdentity(id *rns.Identity, useB32, useB64 bool) {
	if len(id.GetPrivateKey()) == 0 {
		rns.Log("Identity doesn't hold a private key, cannot export", rns.LogError)
		rns.Exit(50)
	}
	prv := id.GetPrivateKey()
	switch {
	case useB64:
		rns.Log("Exported Identity : "+encodeIdentityBase64(prv), rns.LogNotice)
	case useB32:
		rns.Log("Exported Identity : "+base32.StdEncoding.EncodeToString(prv), rns.LogNotice)
	default:
		rns.Log("Exported Identity : "+rns.HexRep(prv, false), rns.LogNotice)
	}
	rns.Exit(0)
}

// ---------- hash / announce ----------

func doHash(id *rns.Identity, aspectsStr string) {
	aspects := strings.Split(aspectsStr, ".")
	if len(aspects) == 0 {
		rns.Log("Invalid destination aspects specified", rns.LogError)
		rns.Exit(32)
	}
	app := aspects[0]
	aspects = aspects[1:]

	if len(id.GetPublicKey()) == 0 {
		rns.Log("No public key known for identity", rns.LogError)
		rns.Exit(32)
	}

	dst, err := rns.NewDestination(id, rns.DestinationOUT, rns.DestinationSINGLE, app, aspects...)
	if err != nil {
		rns.Log("Could not create destination: "+err.Error(), rns.LogError)
		rns.Exit(32)
	}
	rns.Log("The "+aspectsStr+" destination for this Identity is "+rns.PrettyHash(dst.Hash), rns.LogNotice)
	rns.Log("The full destination specifier is "+rns.PrettyHex(dst.Hash), rns.LogNotice)
	time.Sleep(250 * time.Millisecond)
	rns.Exit(0)
}

func doAnnounce(id *rns.Identity, aspectsStr string) {
	aspects := strings.Split(aspectsStr, ".")
	if len(aspects) <= 1 {
		rns.Log("Invalid destination aspects specified", rns.LogError)
		rns.Exit(32)
	}
	app := aspects[0]
	aspects = aspects[1:]

	if len(id.GetPrivateKey()) > 0 {
		dst, err := rns.NewDestination(id, rns.DestinationIN, rns.DestinationSINGLE, app, aspects...)
		if err != nil {
			rns.Log("Could not create destination: "+err.Error(), rns.LogError)
			rns.Exit(32)
		}
		rns.Log("Created destination "+rns.PrettyHex(dst.Hash), rns.LogNotice)
		rns.Log("Announcing destination "+rns.PrettyHash(dst.Hash), rns.LogNotice)
		time.Sleep(1100 * time.Millisecond)
		dst.Announce(nil, false, nil, nil, true)
		time.Sleep(250 * time.Millisecond)
		rns.Exit(0)
	} else {
		dst, err := rns.NewDestination(id, rns.DestinationOUT, rns.DestinationSINGLE, app, aspects...)
		if err != nil {
			rns.Log("Could not create destination: "+err.Error(), rns.LogError)
			rns.Exit(32)
		}
		rns.Log("The "+aspectsStr+" destination for this Identity is "+rns.PrettyHash(dst.Hash), rns.LogNotice)
		rns.Log("The full destination specifier is "+rns.PrettyHex(dst.Hash), rns.LogNotice)
		rns.Log("Cannot announce this destination, since the private key is not held", rns.LogWarning)
		time.Sleep(250 * time.Millisecond)
		rns.Exit(33)
	}
}

// ---------- identity load / recall ----------

func loadIdentity(arg string, requestUnknown bool, timeout time.Duration) *rns.Identity {
	// truncated hash length
	destLen := (rns.ReticulumTruncatedHashLength / 8) * 2
	pathCandidate := expandUser(arg)

	// hex hash
	if len(arg) == destLen && !fileExists(pathCandidate) {
		b, err := hex.DecodeString(arg)
		if err != nil {
			rns.Log("Invalid hexadecimal hash provided", rns.LogError)
			rns.Exit(7)
		}
		id := rns.IdentityRecall(b, false)
		if id == nil {
			id = rns.IdentityRecall(b, true)
		}
		if id == nil {
			if !requestUnknown {
				rns.Log("Could not recall Identity for "+rns.PrettyHash(b)+".", rns.LogError)
				rns.Log("You can query the network for unknown Identities with the -R option.", rns.LogError)
				rns.Exit(5)
			}
			rns.RequestPath(b, nil, nil, false)
			ok := spin(func() bool {
				return rns.IdentityRecall(b, false) != nil || rns.IdentityRecall(b, true) != nil
			}, "Requesting unknown Identity for "+rns.PrettyHash(b), timeout)
			if !ok {
				// The transport side may persist the announce just as the request
				// timeout expires. Refresh the on-disk identity cache once before
				// failing so we don't miss a late-arriving but otherwise valid reply.
				if err := rns.IdentityLoadKnownDestinations(); err == nil {
					id = rns.IdentityRecall(b, false)
					if id == nil {
						id = rns.IdentityRecall(b, true)
					}
					if id != nil {
						rns.Log("Received Identity "+id.String()+" for destination "+rns.PrettyHash(b), rns.LogInfo)
						return id
					}
				}
				rns.Log("Identity request timed out", rns.LogError)
				rns.Exit(6)
			}
			id = rns.IdentityRecall(b, false)
			if id == nil {
				id = rns.IdentityRecall(b, true)
			}
			rns.Log("Received Identity "+id.String()+" for destination "+rns.PrettyHash(b), rns.LogInfo)
		} else {
			identStr := id.String()
			hashStr := rns.PrettyHash(b)
			if identStr == hashStr {
				rns.Log("Recalled Identity "+identStr, rns.LogInfo)
			} else {
				rns.Log("Recalled Identity "+identStr+" for destination "+hashStr, rns.LogInfo)
			}
		}
		return id
	}

	// file
	if !fileExists(pathCandidate) {
		rns.Log("Specified Identity file not found", rns.LogError)
		rns.Exit(8)
	}
	id, err := rns.IdentityFromFile(pathCandidate)
	if err != nil {
		rns.Log("Could not decode Identity from specified file", rns.LogError)
		rns.Exit(9)
	}
	rns.Log("Loaded Identity "+id.String()+" from "+pathCandidate, rns.LogInfo)
	return id
}

// ---------- file operations: sign / validate / encrypt / decrypt ----------

func doIO(id *rns.Identity, doEnc, doDec, doSign bool, validatePath string,
	readPath, writePath *string, force bool, useStdin, useStdout bool) {

	// validate: if --read is not set and validate ends with .rsg -> derive input path
	if validatePath != "" {
		if *readPath == "" && strings.HasSuffix(strings.ToLower(validatePath), "."+sigExt) {
			// Python uses replace() here, not just a suffix trim.
			*readPath = strings.Replace(validatePath, "."+sigExt, "", -1)
		}

		// Python parity: check signature file existence first (exit 10).
		sigPath := expandUser(validatePath)
		if !fileExists(sigPath) {
			// Python message uses args.read here (looks like a bug upstream); keep parity.
			rns.Log("Signature file "+*readPath+" not found", rns.LogError)
			rns.Exit(10)
		}
		validatePath = sigPath

		// Python parity: validate also requires an input file (exit 11).
		if *readPath == "" || !fileExists(expandUser(*readPath)) {
			rns.Log("Input file "+*readPath+" not found", rns.LogError)
			rns.Exit(11)
		}
	}

	var in io.ReadCloser
	var out io.WriteCloser
	var err error

	if useStdin {
		in = io.NopCloser(os.Stdin)
	} else if *readPath != "" {
		path := expandUser(*readPath)
		if !fileExists(path) {
			rns.Log("Input file "+path+" not found", rns.LogError)
			rns.Exit(12)
		}
		in, err = os.Open(path)
		if err != nil {
			rns.Log("Could not open input file for reading", rns.LogError)
			rns.Log("The contained exception was: "+err.Error(), rns.LogError)
			rns.Exit(13)
		}
		*readPath = path
	}
	if validatePath != "" && in == nil {
		// Should not happen due to the validate pre-check above, but keep Python parity.
		rns.Log("Signature verification requested, but no input data specified", rns.LogError)
		rns.Exit(20)
	}

	// derive default output
	if doEnc && *writePath == "" && !useStdout && *readPath != "" {
		*writePath = *readPath + "." + encExt
	}
	if doDec && *writePath == "" && !useStdout && *readPath != "" &&
		strings.HasSuffix(strings.ToLower(*readPath), "."+encExt) {
		*writePath = strings.TrimSuffix(*readPath, "."+encExt)
	}
	if doSign && *writePath == "" && !useStdout && *readPath != "" {
		*writePath = *readPath + "." + sigExt
	}

	// sign requires private key
	if doSign && len(id.GetPrivateKey()) == 0 {
		rns.Log("Specified Identity does not hold a private key. Cannot sign.", rns.LogError)
		rns.Exit(14)
	}

	if *writePath != "" && !useStdout {
		outPath := expandUser(*writePath)
		if !force && fileExists(outPath) {
			rns.Log("Output file "+outPath+" already exists. Not overwriting.", rns.LogError)
			rns.Exit(15)
		}
		out, err = os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			rns.Log("Could not open output file for writing", rns.LogError)
			rns.Log("The contained exception was: "+err.Error(), rns.LogError)
			rns.Exit(15)
		}
		*writePath = outPath
	}

	inputName := *readPath
	if useStdin || inputName == "" {
		inputName = "<stdin>"
	}
	outputName := *writePath
	if useStdout || outputName == "" {
		outputName = "<stdout>"
	}

	// SIGN
	if doSign {
		if in == nil {
			rns.Log("Signing requested, but no input data specified", rns.LogError)
			rns.Exit(17)
		}
		if out == nil && !useStdout {
			rns.Log("Signing requested, but no output specified", rns.LogError)
			rns.Exit(18)
		}
		if !useStdout {
			rns.Log("Signing "+inputName, rns.LogInfo)
		}
		data, err := io.ReadAll(in)
		if err != nil {
			failIO(err, in, out, "signing", 19)
		}
		sig, err := id.Sign(data)
		if err != nil {
			rns.Log("Signing failed: "+err.Error(), rns.LogError)
			rns.Exit(19)
		}
		if useStdout {
			if _, err := os.Stdout.Write(sig); err != nil {
				failIO(err, in, out, "signing", 19)
			}
		} else if out != nil {
			if _, err := out.Write(sig); err != nil {
				failIO(err, in, out, "signing", 19)
			}
			out.Close()
		}
		if in != nil {
			in.Close()
		}
		if !useStdout {
			rns.Log("File "+inputName+" signed with "+id.String()+" to "+outputName, rns.LogInfo)
		}
		rns.Exit(0)
	}

	// VALIDATE
	if validatePath != "" {
		if in == nil {
			rns.Log("Signature verification requested, but no input data specified", rns.LogError)
			rns.Exit(20)
		}
		sigFile, err := os.Open(validatePath)
		if err != nil {
			rns.Log("An error occurred while opening "+validatePath+".", rns.LogError)
			rns.Log("The contained exception was: "+err.Error(), rns.LogError)
			rns.Exit(21)
		}
		sigData, _ := io.ReadAll(sigFile)
		data, _ := io.ReadAll(in)
		sigFile.Close()
		in.Close()

		ok := id.Validate(sigData, data)
		if !ok {
			rns.Log("Signature "+validatePath+" for file "+inputName+" is invalid", rns.LogError)
			rns.Exit(22)
		}
		rns.Log("Signature "+validatePath+" for file "+inputName+" made by Identity "+id.String()+" is valid", rns.LogInfo)
		rns.Exit(0)
	}

	// ENCRYPT
	if doEnc {
		if in == nil {
			rns.Log("Encryption requested, but no input data specified", rns.LogError)
			rns.Exit(24)
		}
		if out == nil && !useStdout {
			rns.Log("Encryption requested, but no output specified", rns.LogError)
			rns.Exit(25)
		}
		if !useStdout {
			rns.Log("Encrypting "+inputName, rns.LogInfo)
		}
		if err := streamTransform(in, out, func(chunk []byte) ([]byte, error) {
			return id.Encrypt(chunk, nil)
		}); err != nil {
			failIO(err, in, out, "encrypting data", 26)
		}
		if !useStdout {
			rns.Log("File "+inputName+" encrypted for "+id.String()+" to "+outputName, rns.LogInfo)
		}
		rns.Exit(0)
	}

	// DECRYPT
	if doDec {
		if len(id.GetPrivateKey()) == 0 {
			rns.Log("Specified Identity does not hold a private key. Cannot decrypt.", rns.LogError)
			rns.Exit(27)
		}
		if in == nil {
			rns.Log("Decryption requested, but no input data specified", rns.LogError)
			rns.Exit(28)
		}
		if out == nil && !useStdout {
			rns.Log("Decryption requested, but no output specified", rns.LogError)
			rns.Exit(29)
		}
		if !useStdout {
			rns.Log("Decrypting "+inputName+"...", rns.LogInfo)
		}
		var errCouldNotDecrypt = errors.New("data could not be decrypted with the specified Identity")
		err := streamTransform(in, out, func(chunk []byte) ([]byte, error) {
			// Python uses identity.decrypt(chunk) and treats a nil plaintext as a non-exception failure.
			plain, decErr := id.Decrypt(chunk, nil, false)
			if decErr != nil {
				return nil, decErr
			}
			if plain == nil {
				return nil, errCouldNotDecrypt
			}
			return plain, nil
		})
		if err != nil {
			if errors.Is(err, errCouldNotDecrypt) {
				if !useStdout {
					rns.Log("Data could not be decrypted with the specified Identity", rns.LogError)
				}
				rns.Exit(30)
			}
			failIO(err, in, out, "decrypting data", 31)
		}
		if !useStdout {
			rns.Log("File "+inputName+" decrypted with "+id.String()+" to "+outputName, rns.LogInfo)
		}
		rns.Exit(0)
	}

	// if nothing was selected
	flag.Usage()
	rns.Exit(0)
}

func streamTransform(in io.ReadCloser, out io.WriteCloser, fn func([]byte) ([]byte, error)) error {
	defer in.Close()
	if out != nil {
		defer out.Close()
	}
	buf := make([]byte, chunkSize)
	for {
		n, err := in.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			outChunk, fnErr := fn(chunk)
			if fnErr != nil {
				return fnErr
			}
			if out != nil {
				if _, werr := out.Write(outChunk); werr != nil {
					return werr
				}
			} else {
				if _, werr := os.Stdout.Write(outChunk); werr != nil {
					return werr
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}
	return nil
}

func failIO(err error, in io.ReadCloser, out io.WriteCloser, what string, code int) {
	rns.Log("An error ocurred while "+what+".", rns.LogError)
	rns.Log("The contained exception was: "+err.Error(), rns.LogError)
	if out != nil {
		out.Close()
	}
	if in != nil {
		in.Close()
	}
	rns.Exit(code)
}

// ---------- utils ----------

func decodeIdentityBase64(data string) ([]byte, error) {
	if out, err := base64.URLEncoding.DecodeString(data); err == nil {
		return out, nil
	}
	trimmed := strings.TrimRight(data, "=")
	return base64.RawURLEncoding.DecodeString(trimmed)
}

func encodeIdentityBase64(data []byte) string {
	return base64.URLEncoding.EncodeToString(data)
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func expandUser(path string) string {
	// Match Python's os.path.expanduser:
	// - "~" -> $HOME
	// - "~/" -> $HOME/...
	// We intentionally do not support "~user" expansion.
	if path == "~" {
		home, _ := os.UserHomeDir()
		if home == "" {
			return path
		}
		return home
	}
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		if home == "" {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
