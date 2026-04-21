package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	rns "github.com/svanichkin/go-reticulum/rns"
)

func destinationHashFromNameAndIdentityHash(fullName string, identityHash []byte) []byte {
	if len(identityHash) != rns.ReticulumTruncatedHashLength/8 {
		return nil
	}
	nameHash := rns.FullHash([]byte(fullName))[:rns.IdentityNameHashLength/8]
	material := append([]byte{}, nameHash...)
	material = append(material, identityHash...)
	full := rns.FullHash(material)
	return append([]byte(nil), full[:rns.ReticulumTruncatedHashLength/8]...)
}

var rnstatusVersion = fmt.Sprintf("rnstatus %s", rns.GetVersion())

type countFlag int

func (c *countFlag) String() string   { return fmt.Sprint(int(*c)) }
func (c *countFlag) IsBoolFlag() bool { return true }
func (c *countFlag) Set(string) error {
	*c++
	return nil
}

type exitError struct {
	code int
	err  error
}

func (e exitError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

func main() {
	var (
		configDir  string
		showAll    bool
		astats     bool
		lstats     bool
		discovered bool
		configEnts bool
		totals     bool
		sortBy     string
		reverse    bool
		jsonOut    bool

		remoteHash string
		identPath  string
		remoteTO   float64

		verbose countFlag
	)

	fs := flag.NewFlagSet("rnstatus", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.StringVar(&configDir, "config", "", "path to alternative Reticulum config directory")

	fs.BoolVar(&showAll, "all", false, "show all interfaces")
	fs.BoolVar(&showAll, "a", false, "show all interfaces")

	fs.BoolVar(&astats, "announce-stats", false, "show announce stats")
	fs.BoolVar(&astats, "A", false, "show announce stats")

	fs.BoolVar(&lstats, "link-stats", false, "show link stats")
	fs.BoolVar(&lstats, "l", false, "show link stats")

	fs.BoolVar(&discovered, "discovered", false, "list discovered interfaces")
	fs.BoolVar(&discovered, "d", false, "list discovered interfaces")

	fs.BoolVar(&configEnts, "D", false, "show details and config entries for discovered interfaces")

	fs.BoolVar(&totals, "totals", false, "display traffic totals")
	fs.BoolVar(&totals, "t", false, "display traffic totals")

	fs.StringVar(&sortBy, "sort", "", "sort interfaces by [rate, traffic, rx, tx, rxs, txs, announces, arx, atx, held]")
	fs.StringVar(&sortBy, "s", "", "sort interfaces by [rate, traffic, rx, tx, rxs, txs, announces, arx, atx, held]")

	fs.BoolVar(&reverse, "reverse", false, "reverse sorting")
	fs.BoolVar(&reverse, "r", false, "reverse sorting")

	fs.BoolVar(&jsonOut, "json", false, "output in JSON format")
	fs.BoolVar(&jsonOut, "j", false, "output in JSON format")

	fs.StringVar(&remoteHash, "R", "", "transport identity hash of remote instance to get status from")
	fs.StringVar(&identPath, "i", "", "path to identity used for remote management")
	fs.Float64Var(&remoteTO, "w", float64(rns.TransportPathRequestTimeout), "timeout before giving up on remote queries")

	fs.Var(&verbose, "verbose", "increase verbosity (repeatable)")
	fs.Var(&verbose, "v", "increase verbosity (repeatable)")

	var version bool
	fs.BoolVar(&version, "version", false, "show version and exit")

	var help bool
	fs.BoolVar(&help, "help", false, "show this help message and exit")
	fs.BoolVar(&help, "h", false, "show this help message and exit")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: rnstatus [-h] [--config CONFIG] [--version] [-a] [-A] [-l] [-d] [-D] [-t] [-s SORT] [-r] [-j] [-R HASH] [-i PATH] [-w SECONDS] [-v] [filter]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Reticulum Network Stack Status")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "options:")
		fs.SetOutput(os.Stderr)
		fs.PrintDefaults()
	}

	if err := fs.Parse(os.Args[1:]); err != nil {
		fs.Usage()
		msg := strings.TrimSpace(err.Error())
		if msg != "" && !strings.HasPrefix(msg, "flag provided but not defined") {
			fmt.Fprintf(os.Stderr, "rnstatus: error: %s\n", msg)
		} else if strings.HasPrefix(msg, "flag provided but not defined") {
			parts := strings.SplitN(msg, ":", 2)
			if len(parts) == 2 {
				unk := strings.TrimSpace(parts[1])
				fmt.Fprintf(os.Stderr, "rnstatus: error: unrecognized arguments: %s\n", unk)
			} else {
				fmt.Fprintf(os.Stderr, "rnstatus: error: %s\n", msg)
			}
		}
		os.Exit(2)
	}

	if help {
		fs.Usage()
		os.Exit(0)
	}

	if version {
		fmt.Println(rnstatusVersion)
		return
	}

	var nameFilter *string
	if fs.NArg() > 0 {
		f := fs.Arg(0)
		nameFilter = &f
	}

	var cfg *string
	if trimmed := strings.TrimSpace(configDir); trimmed != "" {
		cfg = new(string)
		*cfg = trimmed
	}

	if err := programSetup(cfg, showAll, int(verbose), nameFilter, jsonOut,
		astats, lstats, discovered, configEnts, sortBy, reverse, remoteHash, identPath, remoteTO, totals); err != nil {
		if ee, ok := err.(exitError); ok {
			if ee.err != nil && ee.err.Error() != "" {
				fmt.Fprintln(os.Stderr, ee.err)
			}
			os.Exit(ee.code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func programSetup(
	configdir *string,
	dispall bool,
	verbosity int,
	nameFilter *string,
	jsonOut bool,
	astats bool,
	lstats bool,
	discoveredInterfaces bool,
	configEntries bool,
	sorting string,
	sortReverse bool,
	remote string,
	managementIdentity string,
	remoteTimeout float64,
	trafficTotals bool,
) error {
	remote = strings.TrimSpace(remote)
	remotePretty := remote
	requireShared := remote == ""

	logLevel := 3 + verbosity
	var reticulum *rns.Reticulum
	var err error

	reticulum, err = rns.NewReticulum(configdir, &logLevel, nil, nil, requireShared, nil)
	if err != nil {
		return exitError{code: 1, err: fmt.Errorf("no shared RNS instance available to get status from")}
	}

	var (
		stats     map[string]any
		linkCount *int
	)

	details := false
	if configEntries {
		discoveredInterfaces = true
		details = true
	}

	if discoveredInterfaces {
		fmt.Println()
		out, err := renderDiscoveredInterfaces(reticulum.DiscoveredInterfaces(), nameFilter, jsonOut, details)
		if err != nil {
			return err
		}
		fmt.Print(out)
		return nil
	}

	if remote != "" {
		// --- remote mode ---
		if managementIdentity == "" {
			return exitError{code: 20, err: fmt.Errorf("remote management requires an identity file; use -i to specify the path to a management identity")}
		}

		destLen := (rns.ReticulumTruncatedHashLength / 8) * 2
		if len(remote) != destLen {
			return exitError{code: 20, err: fmt.Errorf("destination length is invalid, must be %d hexadecimal characters (%d bytes)", destLen, destLen/2)}
		}

		identityHash, err := hex.DecodeString(remote)
		if err != nil {
			return exitError{code: 20, err: fmt.Errorf("invalid destination entered; check your input")}
		}
		remotePretty = rns.PrettyHex(identityHash)

		destHash := destinationHashFromNameAndIdentityHash("rnstransport.remote.management", identityHash)
		id, err := rns.IdentityFromFile(expandUser(managementIdentity))
		if err != nil || id == nil {
			return exitError{code: 20, err: fmt.Errorf("could not load management identity from %s", managementIdentity)}
		}

		s, lc, err := getRemoteStatus(destHash, lstats, id, jsonOut, remoteTimeout)
		if err != nil {
			if ee, ok := err.(exitError); ok {
				return ee
			}
			return exitError{code: 20, err: err}
		}
		stats = s
		linkCount = lc
	} else {
		// --- local mode ---
		if lstats {
			c := reticulum.GetLinkCount()
			linkCount = &c
		}
		stats = reticulum.GetInterfaceStats()
	}

	if stats == nil {
		if remote == "" {
			return exitError{code: 2, err: fmt.Errorf("could not get RNS status")}
		}
		target := strings.TrimSpace(remotePretty)
		if target == "" {
			target = "remote transport instance"
		}
		return exitError{code: 2, err: fmt.Errorf("could not get RNS status from remote transport instance %s", target)}
	}

	if jsonOut {
		normaliseBytesRoot(stats)
		data, err := json.Marshal(stats)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	ifacesAny, ok := stats["interfaces"]
	if !ok {
		return fmt.Errorf("no interface stats available")
	}

	interfaces := toIfaceSlice(ifacesAny)

	// sorting
	if sorting != "" {
		sortKey := strings.ToLower(sorting)
		sortInterfaces(interfaces, sortKey, !sortReverse)
	}

	for _, ifstat := range interfaces {
		name, _ := ifstat["name"].(string)

		if !dispall {
			if strings.HasPrefix(name, "LocalInterface[") ||
				strings.HasPrefix(name, "TCPInterface[Client") ||
				strings.HasPrefix(name, "BackboneInterface[Client on") ||
				strings.HasPrefix(name, "AutoInterfacePeer[") ||
				strings.HasPrefix(name, "WeaveInterfacePeer[") ||
				strings.HasPrefix(name, "I2PInterfacePeer[Connected peer") ||
				(strings.HasPrefix(name, "I2PInterface[") &&
					!boolField(ifstat, "i2p_connectable")) {
				continue
			}
		}

		if strings.HasPrefix(name, "I2PInterface[") &&
			!boolField(ifstat, "i2p_connectable") {
			continue
		}

		if nameFilter != nil && !strings.Contains(strings.ToLower(name), strings.ToLower(*nameFilter)) {
			continue
		}

		fmt.Println()
		fmt.Printf(" %s\n", name)

		if netname, ok := ifstat["ifac_netname"].(string); ok && netname != "" {
			fmt.Printf("    Network   : %s\n", netname)
		}

		status := "Down"
		if boolField(ifstat, "status") {
			status = "Up"
		}
		fmt.Printf("    Status    : %s\n", status)

		clientsStr := ""
		if c, ok := numField(ifstat, "clients"); ok {
			switch {
			case strings.HasPrefix(name, "Shared Instance["):
				cnum := c
				if cnum > 0 {
					cnum--
				}
				spec := " programs"
				if cnum == 1 {
					spec = " program"
				}
				clientsStr = fmt.Sprintf("Serving   : %d%s", cnum, spec)
			case strings.HasPrefix(name, "I2PInterface["):
				if boolField(ifstat, "i2p_connectable") {
					cnum := c
					spec := " connected I2P endpoints"
					if cnum == 1 {
						spec = " connected I2P endpoint"
					}
					clientsStr = fmt.Sprintf("Peers     : %d%s", cnum, spec)
				}
			default:
				clientsStr = fmt.Sprintf("Clients   : %d", c)
			}
		}
		if clientsStr != "" {
			fmt.Printf("    %s\n", clientsStr)
		}

		omitMode := strings.HasPrefix(name, "Shared Instance[") ||
			strings.HasPrefix(name, "TCPInterface[Client") ||
			strings.HasPrefix(name, "LocalInterface[")
		if !omitMode {

			mode := intField(ifstat, "mode")
			modeStr := "Full"
			switch mode {
			case rns.InterfaceModeAccessPoint:
				modeStr = "Access Point"
			case rns.InterfaceModePointToPoint:
				modeStr = "Point-to-Point"
			case rns.InterfaceModeRoaming:
				modeStr = "Roaming"
			case rns.InterfaceModeBoundary:
				modeStr = "Boundary"
			case rns.InterfaceModeGateway:
				modeStr = "Gateway"
			}
			fmt.Printf("    Mode      : %s\n", modeStr)
		}

		if bitrate, ok := numField(ifstat, "bitrate"); ok {
			fmt.Printf("    Rate      : %s\n", speedStr(float64(bitrate), "bps"))
		}

		// noise floor + interference
		if _, present := ifstat["noise_floor"]; present {
			if nfVal, ok := numField(ifstat, "noise_floor"); ok {
				nstr := ""
				if interVal, okI := numField(ifstat, "interference"); okI {
					if interVal != 0 {
						nstr = fmt.Sprintf("\n    Intrfrnc. : %d dBm", interVal)
					} else {
						nstr = ", no interference"
					}
					if ts, okTS := numField(ifstat, "interference_last_ts"); okTS {
						if dbm, okDBM := numField(ifstat, "interference_last_dbm"); okDBM {
							ago := time.Since(time.Unix(int64(ts), 0))
							nstr = fmt.Sprintf("\n    Intrfrnc. : %d dBm %s ago", dbm, rns.PrettyTime(ago.Seconds(), true, false))
						}
					}
				} else if ts, okTS := numField(ifstat, "interference_last_ts"); okTS {
					if dbm, okDBM := numField(ifstat, "interference_last_dbm"); okDBM {
						ago := time.Since(time.Unix(int64(ts), 0))
						nstr = fmt.Sprintf("\n    Intrfrnc. : %d dBm %s ago", dbm, rns.PrettyTime(ago.Seconds(), true, false))
					}
				}
				fmt.Printf("    Noise Fl. : %d dBm%s\n", nfVal, nstr)
			} else {
				fmt.Println("    Noise Fl. : Unknown")
			}
		}

		if _, present := ifstat["cpu_load"]; present {
			if v, ok := numField(ifstat, "cpu_load"); ok {
				fmt.Printf("    CPU load  : %d %%\n", v)
			} else {
				fmt.Println("    CPU load  : Unknown")
			}
		}
		if _, present := ifstat["cpu_temp"]; present {
			if v, ok := numField(ifstat, "cpu_temp"); ok {
				fmt.Printf("    CPU temp  : %d°C\n", v)
			} else {
				fmt.Println("    CPU temp  : Unknown")
			}
		}
		if _, present := ifstat["mem_load"]; present {
			// Python rnstatus.py uses a (likely unintended) cpu_load guard here:
			// if "mem_load" in ifstat:
			//   if ifstat["cpu_load"] != None: print(mem_load) else Unknown
			// Keep output parity with Python.
			if cpuAny, ok := ifstat["cpu_load"]; ok && cpuAny != nil {
				if v, ok := numField(ifstat, "mem_load"); ok {
					fmt.Printf("    Mem usage : %d %%\n", v)
				} else {
					fmt.Println("    Mem usage : Unknown")
				}
			} else {
				fmt.Println("    Mem usage : Unknown")
			}
		}

		if bp, ok := numField(ifstat, "battery_percent"); ok {
			if st, ok2 := ifstat["battery_state"].(string); ok2 {
				fmt.Printf("    Battery   : %d%% (%s)\n", bp, st)
			}
		}

		if ats, okS := numField(ifstat, "airtime_short"); okS {
			if atl, okL := numField(ifstat, "airtime_long"); okL {
				fmt.Printf("    Airtime   : %d%% (15s), %d%% (1h)\n", ats, atl)
			}
		}
		if cls, okS := numField(ifstat, "channel_load_short"); okS {
			if cll, okL := numField(ifstat, "channel_load_long"); okL {
				fmt.Printf("    Ch. Load  : %d%% (15s), %d%% (1h)\n", cls, cll)
			}
		}

		if val, present := ifstat["switch_id"]; present {
			if v, ok := val.(string); ok && v != "" {
				fmt.Printf("    Switch ID : %s\n", v)
			} else {
				fmt.Println("    Switch ID : Unknown")
			}
		}
		if val, present := ifstat["endpoint_id"]; present {
			if v, ok := val.(string); ok && v != "" {
				fmt.Printf("    Endpoint  : %s\n", v)
			} else {
				fmt.Println("    Endpoint  : Unknown")
			}
		}
		if val, present := ifstat["via_switch_id"]; present {
			if v, ok := val.(string); ok && v != "" {
				fmt.Printf("    Via       : %s\n", v)
			} else {
				fmt.Println("    Via       : Unknown")
			}
		}
		if v, ok := numField(ifstat, "peers"); ok {
			fmt.Printf("    Peers     : %d reachable\n", v)
		}
		if v, ok := ifstat["tunnelstate"].(string); ok && v != "" {
			fmt.Printf("    I2P       : %s\n", v)
		}
		if sig, ok := byteField(ifstat, "ifac_signature"); ok && len(sig) >= 5 {
			nb := intField(ifstat, "ifac_size") * 8
			sigStr := "<…" + rns.HexRep(sig[len(sig)-5:], false) + ">"
			fmt.Printf("    Access    : %d-bit IFAC by %s\n", nb, sigStr)
		}
		if b32, ok := ifstat["i2p_b32"].(string); ok && b32 != "" {
			fmt.Printf("    I2P B32   : %s\n", b32)
		}

		if astats {
			if v, ok := numField(ifstat, "announce_queue"); ok && v > 0 {
				if v == 1 {
					fmt.Printf("    Queued    : %d announce\n", v)
				} else {
					fmt.Printf("    Queued    : %d announces\n", v)
				}
			}
			if v, ok := numField(ifstat, "held_announces"); ok && v > 0 {
				if v == 1 {
					fmt.Printf("    Held      : %d announce\n", v)
				} else {
					fmt.Printf("    Held      : %d announces\n", v)
				}
			}
			if oaf, ok := numField(ifstat, "outgoing_announce_frequency"); ok {
				fmt.Printf("    Announces : %s↑\n", rns.PrettyFrequency(float64(oaf)))
			}
			if iaf, ok := numField(ifstat, "incoming_announce_frequency"); ok {
				fmt.Printf("                %s↓\n", rns.PrettyFrequency(float64(iaf)))
			}
		}

		// Traffic
		rxb, _ := numField(ifstat, "rxb")
		txb, _ := numField(ifstat, "txb")
		rxs, haveRxs := numField(ifstat, "rxs")
		txs, haveTxs := numField(ifstat, "txs")

		rxbStr := "↓" + rns.PrettySize(float64(rxb))
		txbStr := "↑" + rns.PrettySize(float64(txb))
		diff := len(rxbStr) - len(txbStr)
		if diff > 0 {
			txbStr += strings.Repeat(" ", diff)
		} else if diff < 0 {
			rxbStr += strings.Repeat(" ", -diff)
		}
		rxStat := rxbStr
		txStat := txbStr
		if haveRxs && haveTxs {
			rxStat += "  " + rns.PrettySpeed(float64(rxs))
			txStat += "  " + rns.PrettySpeed(float64(txs))
		}

		fmt.Printf("    Traffic   : %s\n", txStat)
		fmt.Printf("                %s\n", rxStat)
	}

	// link table stats
	lstr := ""
	if lstats && linkCount != nil {
		ms := "ies"
		if *linkCount == 1 {
			ms = "y"
		}
		if tid, ok := stats["transport_id"].([]byte); ok && tid != nil {
			lstr = fmt.Sprintf(", %d entr%s in link table", *linkCount, ms)
		} else {
			lstr = fmt.Sprintf(" %d entr%s in link table", *linkCount, ms)
		}
	}

	if trafficTotals {
		rxb, _ := numField(stats, "rxb")
		txb, _ := numField(stats, "txb")
		rxs, _ := numField(stats, "rxs")
		txs, _ := numField(stats, "txs")

		rxbStr := "↓" + rns.PrettySize(float64(rxb))
		txbStr := "↑" + rns.PrettySize(float64(txb))
		diff := len(rxbStr) - len(txbStr)
		if diff > 0 {
			txbStr += strings.Repeat(" ", diff)
		} else if diff < 0 {
			rxbStr += strings.Repeat(" ", -diff)
		}
		rxStat := rxbStr + "  " + rns.PrettySpeed(float64(rxs))
		txStat := txbStr + "  " + rns.PrettySpeed(float64(txs))

		fmt.Printf("\n Totals       : %s\n", txStat)
		fmt.Printf("                %s\n", rxStat)
	}

	if tid, ok := stats["transport_id"].([]byte); ok && tid != nil {
		fmt.Printf("\n Transport Instance %s running\n", rns.PrettyHex(tid))
		if pr, ok := stats["probe_responder"].([]byte); ok && pr != nil {
			fmt.Printf(" Probe responder at %s active\n", rns.PrettyHex(pr))
		}
		if ut, ok := numField(stats, "transport_uptime"); ok {
			fmt.Printf(" Uptime is %s%s\n", rns.PrettyTime(float64(ut), false, false), lstr)
		}
	} else if lstr != "" {
		fmt.Println("\n" + lstr)
	}

	fmt.Println()
	return nil
}

// -------- remote status --------

func getRemoteStatus(destHash []byte, includeLstats bool, identity *rns.Identity, noOutput bool, timeout float64) (map[string]any, *int, error) {
	if !rns.HasPath(destHash) {
		if !noOutput {
			fmt.Print("Path to " + rns.PrettyHex(destHash) + " requested ")
			os.Stdout.Sync()
		}
		rns.RequestPath(destHash, nil, nil, false)
		start := time.Now()
		for !rns.HasPath(destHash) {
			time.Sleep(100 * time.Millisecond)
			if time.Since(start).Seconds() > timeout {
				if !noOutput {
					fmt.Print("\r                                                          \r")
					fmt.Println("Path request timed out")
				}
				// Python already printed a message and exits with code 12.
				return nil, nil, exitError{code: 12, err: nil}
			}
		}
	}

	remoteIdentity := rns.IdentityRecall(destHash, false)
	if remoteIdentity == nil {
		// Parity helper: allow recalling from identity hash if a full destination recall fails.
		remoteIdentity = rns.IdentityRecall(destHash, true)
	}
	if remoteIdentity == nil {
		return nil, nil, fmt.Errorf("could not recall remote identity")
	}

	done := make(chan struct {
		stats     map[string]any
		linkCount *int
		err       error
	}, 1)

	remoteDest, err := rns.NewDestination(remoteIdentity, rns.DestinationOUT, rns.DestinationSINGLE, "rnstransport", "remote", "management")
	if err != nil {
		return nil, nil, err
	}
	link, err := rns.NewOutgoingLink(remoteDest, rns.LinkModeDefault, nil, nil)
	if err != nil {
		return nil, nil, err
	}

	link.SetLinkClosedCallback(func(l *rns.Link) {
		if !noOutput {
			fmt.Print("\r                                                          \r")
			switch l.TeardownReason {
			case rns.LinkTimeout:
				fmt.Println("The link timed out, exiting now")
			case rns.LinkDestinationClose:
				fmt.Println("The link was closed by the server, exiting now")
			default:
				fmt.Println("Link closed unexpectedly, exiting now")
			}
		}
		select {
		case done <- struct {
			stats     map[string]any
			linkCount *int
			err       error
		}{nil, nil, exitError{code: 10, err: nil}}:
		default:
		}
	})

	link.SetLinkEstablishedCallback(func(l *rns.Link) {
		if !noOutput {
			fmt.Print("\r                                                          \r")
			fmt.Print("Sending request... ")
			os.Stdout.Sync()
		}
		l.Identify(identity)
		receipt := rns.RequestReceiptFrom(l.Request("/status", []any{includeLstats}, nil, nil, nil, timeout))
		if receipt == nil {
			done <- struct {
				stats     map[string]any
				linkCount *int
				err       error
			}{nil, nil, nil}
			return
		}
		for !receipt.Concluded() {
			time.Sleep(100 * time.Millisecond)
		}
		resp := receipt.Response()
		if resp == nil {
			if !noOutput {
				fmt.Print("\r                                                          \r")
				fmt.Println("The remote status request failed. Likely authentication failure.")
			}
			// Python returns None here, and the caller exits with code 2.
			done <- struct {
				stats     map[string]any
				linkCount *int
				err       error
			}{nil, nil, nil}
			return
		}

		// expected: response is []any{stats, linkCount?}
		list, ok := resp.([]any)
		if !ok || len(list) == 0 {
			// Python treats this as invalid result and exits with code 2.
			done <- struct {
				stats     map[string]any
				linkCount *int
				err       error
			}{nil, nil, nil}
			return
		}

		stats, _ := list[0].(map[string]any)
		var lc *int
		if len(list) > 1 {
			if n, ok := list[1].(int); ok {
				lc = &n
			}
		}
		done <- struct {
			stats     map[string]any
			linkCount *int
			err       error
		}{stats, lc, nil}
	})

	if !noOutput {
		fmt.Print("\r                                                          \r")
		fmt.Print("Establishing link with remote transport instance... ")
		os.Stdout.Sync()
	}

	res := <-done
	if res.err == nil && !noOutput {
		fmt.Print("\r                                                          \r")
	}
	return res.stats, res.linkCount, res.err
}

// -------- helpers --------

func expandUser(p string) string {
	if p == "" {
		return p
	}
	if p[0] == '~' {
		home, _ := os.UserHomeDir()
		if len(p) == 1 {
			return home
		}
		if p[1] == '/' || p[1] == '\\' {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

func normaliseBytes(v any) any {
	switch x := v.(type) {
	case []byte:
		return rns.HexRep(x, false)
	case map[string]any:
		for k, vv := range x {
			x[k] = normaliseBytes(vv)
		}
		return x
	case []any:
		for i, vv := range x {
			x[i] = normaliseBytes(vv)
		}
		return x
	case []map[string]any:
		for i := range x {
			for k, vv := range x[i] {
				x[i][k] = normaliseBytes(vv)
			}
		}
		return x
	default:
		return v
	}
}

func normaliseBytesRoot(m map[string]any) {
	for k, v := range m {
		m[k] = normaliseBytes(v)
	}
}

func toIfaceSlice(v any) []map[string]any {
	res := []map[string]any{}
	switch t := v.(type) {
	case []map[string]any:
		return t
	case []any:
		for _, el := range t {
			if m, ok := el.(map[string]any); ok {
				res = append(res, m)
			}
		}
	}
	return res
}

func numField(m map[string]any, k string) (int, bool) {
	v, ok := m[k]
	if !ok || v == nil {
		return 0, false
	}
	switch n := v.(type) {
	case int:
		return n, true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case uint:
		if uint64(n) > uint64(math.MaxInt) {
			return math.MaxInt, true
		}
		return int(n), true
	case uint32:
		return int(n), true
	case uint64:
		if n > uint64(math.MaxInt) {
			return math.MaxInt, true
		}
		return int(n), true
	case float32:
		return int(n), true
	case float64:
		return int(n), true
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return int(i), true
		}
		if f, err := n.Float64(); err == nil {
			return int(f), true
		}
		return 0, false
	}
	return 0, false
}

func byteField(m map[string]any, k string) ([]byte, bool) {
	v, ok := m[k]
	if !ok || v == nil {
		return nil, false
	}
	switch b := v.(type) {
	case []byte:
		return b, true
	case string:
		return []byte(b), true
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		if rv.Type().Elem().Kind() != reflect.Uint8 {
			return nil, false
		}
		out := make([]byte, rv.Len())
		reflect.Copy(reflect.ValueOf(out), rv)
		return out, true
	default:
		return nil, false
	}
}

func intField(m map[string]any, k string) int {
	if v, ok := numField(m, k); ok {
		return v
	}
	return 0
}

func boolField(m map[string]any, k string) bool {
	v, ok := m[k]
	if !ok || v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

func speedStr(num float64, suffix string) string {
	units := []string{"", "k", "M", "G", "T", "P", "E", "Z"}
	lastUnit := "Y"
	if suffix == "Bps" {
		num /= 8
		units = []string{"", "K", "M", "G", "T", "P", "E", "Z"}
		lastUnit = "Y"
	}
	for _, u := range units {
		if abs(num) < 1000.0 {
			return fmt.Sprintf("%3.2f %s%s", num, u, suffix)
		}
		num /= 1000.0
	}
	return fmt.Sprintf("%.2f %s%s", num, lastUnit, suffix)
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func renderDiscoveredInterfaces(ifs []map[string]any, nameFilter *string, jsonOut bool, details bool) (string, error) {
	if jsonOut {
		normalised := make([]map[string]any, len(ifs))
		for i, info := range ifs {
			cp := make(map[string]any, len(info))
			for k, v := range info {
				cp[k] = normaliseBytes(v)
			}
			normalised[i] = cp
		}
		data, err := json.Marshal(normalised)
		if err != nil {
			return "", err
		}
		return string(data) + "\n", nil
	}

	filtered := filterDiscoveredInterfaces(ifs, nameFilter)
	if details {
		return renderDiscoveredInterfacesDetailed(filtered), nil
	}
	return renderDiscoveredInterfacesCompact(filtered), nil
}

func filterDiscoveredInterfaces(ifs []map[string]any, nameFilter *string) []map[string]any {
	if nameFilter == nil || strings.TrimSpace(*nameFilter) == "" {
		return ifs
	}
	filter := strings.ToLower(strings.TrimSpace(*nameFilter))
	filtered := make([]map[string]any, 0, len(ifs))
	for _, info := range ifs {
		name, _ := info["name"].(string)
		if strings.Contains(strings.ToLower(name), filter) {
			filtered = append(filtered, info)
		}
	}
	return filtered
}

func renderDiscoveredInterfacesDetailed(ifs []map[string]any) string {
	var b strings.Builder
	for idx, info := range ifs {
		if idx > 0 {
			b.WriteString("\n================================\n\n")
		}

		transportID, _ := info["transport_id"].(string)
		networkID, _ := info["network_id"].(string)
		if networkID != "" && networkID != transportID {
			fmt.Fprintf(&b, "Network   ID : %s\n", networkID)
		}
		if transportID != "" {
			fmt.Fprintf(&b, "Transport ID : %s\n", transportID)
		}

		name, _ := info["name"].(string)
		ifType, _ := info["type"].(string)
		fmt.Fprintf(&b, "Name         : %s\n", name)
		fmt.Fprintf(&b, "Type         : %s\n", ifType)
		fmt.Fprintf(&b, "Status       : %s\n", discoveredStatusDetailed(stringField(info, "status")))
		fmt.Fprintf(&b, "Transport    : %s\n", discoveredTransportStatus(boolField(info, "transport")))

		hops, _ := numField(info, "hops")
		hopSuffix := "s"
		if hops == 1 {
			hopSuffix = ""
		}
		fmt.Fprintf(&b, "Distance     : %d hop%s\n", hops, hopSuffix)

		if discoveredAt, ok := numField(info, "discovered"); ok {
			dago := time.Since(time.Unix(int64(discoveredAt), 0)).Seconds()
			fmt.Fprintf(&b, "Discovered   : %s ago\n", rns.PrettyTime(dago, false, true))
		}
		if lastHeard, ok := numField(info, "last_heard"); ok {
			hago := time.Since(time.Unix(int64(lastHeard), 0)).Seconds()
			fmt.Fprintf(&b, "Last Heard   : %s ago\n", rns.PrettyTime(hago, false, true))
		}

		fmt.Fprintf(&b, "Location     : %s\n", discoveredLocationDetailed(info))

		if v, ok := numField(info, "frequency"); ok {
			fmt.Fprintf(&b, "Frequency    : %s Hz\n", formatThousands(v))
		}
		if v, ok := numField(info, "bandwidth"); ok {
			fmt.Fprintf(&b, "Bandwidth    : %s Hz\n", formatThousands(v))
		}
		if v, ok := numField(info, "sf"); ok {
			fmt.Fprintf(&b, "Sprd. Factor : %d\n", v)
		}
		if v, ok := numField(info, "cr"); ok {
			fmt.Fprintf(&b, "Coding Rate  : %d\n", v)
		}
		if modulation, _ := info["modulation"].(string); modulation != "" {
			fmt.Fprintf(&b, "Modulation   : %s\n", modulation)
		}
		if reachableOn, _ := info["reachable_on"].(string); reachableOn != "" {
			fmt.Fprintf(&b, "Address      : %s\n", reachableOn)
		}
		if port, ok := numField(info, "port"); ok {
			fmt.Fprintf(&b, "Port         : %d\n", port)
		}

		if value, ok := numField(info, "value"); ok {
			fmt.Fprintf(&b, "Stamp Value  : %d\n", value)
		}

		if entry, _ := info["config_entry"].(string); entry != "" {
			b.WriteString("\nConfiguration Entry:\n")
			for _, line := range strings.Split(entry, "\n") {
				fmt.Fprintf(&b, "  %s\n", line)
			}
		}
	}
	return b.String()
}

func renderDiscoveredInterfacesCompact(ifs []map[string]any) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%-25s %-12s %-12s %-12s %-8s %-15s\n", "Name", "Type", "Status", "Last Heard", "Value", "Location")
	b.WriteString(strings.Repeat("-", 89) + "\n")
	for _, info := range ifs {
		name, _ := info["name"].(string)
		if runeCount := len([]rune(name)); runeCount > 24 {
			runes := []rune(name)
			name = string(runes[:24]) + "..."
		}

		ifType, _ := info["type"].(string)
		ifType = strings.ReplaceAll(ifType, "Interface", "")

		status := discoveredStatusCompact(stringField(info, "status"))
		lastHeardDisplay := discoveredLastHeardCompact(info)
		value := ""
		if v, ok := numField(info, "value"); ok {
			value = fmt.Sprintf("%d", v)
		}
		location := discoveredLocationCompact(info)
		fmt.Fprintf(&b, "%-25s %-12s %-12s %-12s %-8s %-15s\n", name, ifType, status, lastHeardDisplay, value, location)
	}
	return b.String()
}

func stringField(m map[string]any, k string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[k].(string); ok {
		return v
	}
	return ""
}

func discoveredStatusDetailed(status string) string {
	switch status {
	case "available":
		return "Available"
	case "unknown":
		return "Unknown"
	case "stale":
		return "Stale"
	default:
		return status
	}
}

func discoveredStatusCompact(status string) string {
	switch status {
	case "available":
		return "✓ Available"
	case "unknown":
		return "? Unknown"
	case "stale":
		return "× Stale"
	default:
		return status
	}
}

func discoveredTransportStatus(enabled bool) string {
	if enabled {
		return "Enabled"
	}
	return "Disabled"
}

func discoveredLocationDetailed(info map[string]any) string {
	lat, latOK := info["latitude"]
	lon, lonOK := info["longitude"]
	if !latOK || !lonOK || lat == nil || lon == nil {
		return "Unknown"
	}
	latf, latIsNum := floatField(lat)
	lonf, lonIsNum := floatField(lon)
	if !latIsNum || !lonIsNum {
		return "Unknown"
	}
	location := fmt.Sprintf("%.4f, %.4f", latf, lonf)
	if heightVal, ok := info["height"]; ok && heightVal != nil {
		if height, ok := floatField(heightVal); ok {
			location += fmt.Sprintf(", %gm h", height)
		}
	}
	return location
}

func discoveredLocationCompact(info map[string]any) string {
	lat, latOK := info["latitude"]
	lon, lonOK := info["longitude"]
	if !latOK || !lonOK || lat == nil || lon == nil {
		return "N/A"
	}
	latf, latIsNum := floatField(lat)
	lonf, lonIsNum := floatField(lon)
	if !latIsNum || !lonIsNum {
		return "N/A"
	}
	return fmt.Sprintf("%.4f, %.4f", latf, lonf)
}

func discoveredLastHeardCompact(info map[string]any) string {
	lastHeard, ok := numField(info, "last_heard")
	if !ok {
		return ""
	}
	diff := time.Since(time.Unix(int64(lastHeard), 0))
	switch {
	case diff < time.Minute:
		return "Just now"
	case diff < time.Hour:
		return fmt.Sprintf("%dm ago", int(diff/time.Minute))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(diff/time.Hour))
	default:
		return fmt.Sprintf("%dd ago", int(diff/(24*time.Hour)))
	}
}

func floatField(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

func formatThousands(v int) string {
	s := fmt.Sprintf("%d", v)
	if len(s) <= 3 {
		return s
	}
	var parts []string
	for len(s) > 3 {
		parts = append(parts, s[len(s)-3:])
		s = s[:len(s)-3]
	}
	parts = append(parts, s)
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, ",")
}

// sort by traffic/bitrate/announce keys
func sortInterfaces(ifs []map[string]any, key string, desc bool) {
	less := func(i, j int) bool {
		a := ifs[i]
		b := ifs[j]
		switch key {
		case "rate", "bitrate":
			ai, _ := numField(a, "bitrate")
			bi, _ := numField(b, "bitrate")
			if desc {
				return ai > bi
			}
			return ai < bi
		case "rx":
			ai, _ := numField(a, "rxb")
			bi, _ := numField(b, "rxb")
			if desc {
				return ai > bi
			}
			return ai < bi
		case "tx":
			ai, _ := numField(a, "txb")
			bi, _ := numField(b, "txb")
			if desc {
				return ai > bi
			}
			return ai < bi
		case "rxs":
			ai, _ := numField(a, "rxs")
			bi, _ := numField(b, "rxs")
			if desc {
				return ai > bi
			}
			return ai < bi
		case "txs":
			ai, _ := numField(a, "txs")
			bi, _ := numField(b, "txs")
			if desc {
				return ai > bi
			}
			return ai < bi
		case "traffic":
			ai, _ := numField(a, "rxb")
			aj, _ := numField(a, "txb")
			bi, _ := numField(b, "rxb")
			bj, _ := numField(b, "txb")
			at := ai + aj
			bt := bi + bj
			if desc {
				return at > bt
			}
			return at < bt
		case "announces", "announce":
			ai, _ := numField(a, "incoming_announce_frequency")
			aj, _ := numField(a, "outgoing_announce_frequency")
			bi, _ := numField(b, "incoming_announce_frequency")
			bj, _ := numField(b, "outgoing_announce_frequency")
			at := ai + aj
			bt := bi + bj
			if desc {
				return at > bt
			}
			return at < bt
		case "arx":
			ai, _ := numField(a, "incoming_announce_frequency")
			bi, _ := numField(b, "incoming_announce_frequency")
			if desc {
				return ai > bi
			}
			return ai < bi
		case "atx":
			ai, _ := numField(a, "outgoing_announce_frequency")
			bi, _ := numField(b, "outgoing_announce_frequency")
			if desc {
				return ai > bi
			}
			return ai < bi
		case "held":
			ai, _ := numField(a, "held_announces")
			bi, _ := numField(b, "held_announces")
			if desc {
				return ai > bi
			}
			return ai < bi
		default:
			return false
		}
	}

	sort.SliceStable(ifs, less)
}
