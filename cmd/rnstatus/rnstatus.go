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
	"sort"
	"strings"
	"time"

	rns "github.com/svanichkin/go-reticulum/rns"
)

var rnstatusVersion = fmt.Sprintf("rnstatus %s", rns.GetVersion())

type countFlag int

func (c *countFlag) String() string { return fmt.Sprint(int(*c)) }
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
		configDir string
		showAll   bool
		astats    bool
		lstats    bool
		totals    bool
		sortBy    string
		reverse   bool
		jsonOut   bool

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
		fmt.Fprintln(os.Stderr, "usage: rnstatus [-h] [--config CONFIG] [--version] [-a] [-A] [-l] [-t] [-s SORT] [-r] [-j] [-R HASH] [-i PATH] [-w SECONDS] [-v] [filter]")
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
		astats, lstats, sortBy, reverse, remoteHash, identPath, remoteTO, totals); err != nil {
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

		destHash := rns.HashFromNameAndIdentity("rnstransport.remote.management", identityHash)
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
		if sig, ok := ifstat["ifac_signature"].([]byte); ok && len(sig) >= 5 {
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
	if !rns.TransportHasPath(destHash) {
		if !noOutput {
			fmt.Print("Path to " + rns.PrettyHex(destHash) + " requested ")
			os.Stdout.Sync()
		}
		rns.TransportRequestPath(destHash)
		start := time.Now()
		for !rns.TransportHasPath(destHash) {
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
	link, err := rns.NewOutgoingLink(remoteDest, 0, nil, nil)
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
