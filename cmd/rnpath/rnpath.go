package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
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

const programName = "rnpath"

var (
	reticulum  *rns.Reticulum
	remoteLink *rns.Link
)

type exitError struct {
	code int
	msg  string
}

func (e exitError) Error() string { return e.msg }

func setupSignals() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for range c {
			if remoteLink != nil {
				remoteLink.Teardown()
			}
			os.Exit(0)
		}
	}()
}

type countFlag int

func (c *countFlag) String() string { return fmt.Sprint(int(*c)) }
func (c *countFlag) Set(string) error {
	*c++
	return nil
}
func (c *countFlag) IsBoolFlag() bool { return true }

func main() {
	var (
		configDir    string
		table        bool
		maxHops      int
		rates        bool
		drop         bool
		dropQueues   bool
		dropVia      bool
		timeout      float64
		remoteHash   string
		identityPath string
		remoteTO     float64
		jsonOut      bool
		showVersion  bool
		verbose      countFlag
	)

	flag.StringVar(&configDir, "config", "", "path to alternative Reticulum config directory")
	flag.BoolVar(&table, "table", false, "show all known paths")
	flag.BoolVar(&table, "t", false, "show all known paths")

	flag.IntVar(&maxHops, "max", -1, "maximum hops to filter path table by")
	flag.IntVar(&maxHops, "m", -1, "maximum hops to filter path table by")

	flag.BoolVar(&rates, "rates", false, "show announce rate info")
	flag.BoolVar(&rates, "r", false, "show announce rate info")

	flag.BoolVar(&drop, "drop", false, "remove the path to a destination")
	flag.BoolVar(&drop, "d", false, "remove the path to a destination")

	flag.BoolVar(&dropQueues, "drop-announces", false, "drop all queued announces")
	flag.BoolVar(&dropQueues, "D", false, "drop all queued announces")

	flag.BoolVar(&dropVia, "drop-via", false, "drop all paths via specified transport instance")
	flag.BoolVar(&dropVia, "x", false, "drop all paths via specified transport instance")

	flag.Float64Var(&timeout, "w", float64(rns.TransportPathRequestTimeout), "timeout before giving up")

	flag.StringVar(&remoteHash, "R", "", "transport identity hash of remote instance to manage")
	flag.StringVar(&identityPath, "i", "", "path to identity used for remote management")
	flag.Float64Var(&remoteTO, "W", float64(rns.TransportPathRequestTimeout), "timeout before giving up on remote queries")

	flag.BoolVar(&jsonOut, "json", false, "output in JSON format")
	flag.BoolVar(&jsonOut, "j", false, "output in JSON format")

	flag.BoolVar(&showVersion, "version", false, "print program version and exit")
	flag.BoolVar(&showVersion, "V", false, "print program version and exit")

	flag.Var(&verbose, "v", "increase verbosity")
	flag.Var(&verbose, "verbose", "increase verbosity")

	flag.Usage = func() {
		fmt.Println("")
		fmt.Println("Reticulum Path Discovery Utility")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  rnpath [options] <destination>")
		fmt.Println("")
		flag.PrintDefaults()
	}

	flag.Parse()

	if showVersion {
		fmt.Printf("%s %s\n", programName, rns.GetVersion())
		return
	}

	destStr := ""
	if flag.NArg() > 0 {
		destStr = flag.Arg(0)
	}

	// Python rnpath has a slightly odd help condition: it does not consider --drop,
	// so we match it for 1:1 parity.
	if !dropQueues && !table && !rates && destStr == "" && !dropVia {
		fmt.Println()
		flag.Usage()
		fmt.Println()
		return
	}

	setupSignals()

	var cfgPtr *string
	if strings.TrimSpace(configDir) != "" {
		cfgPtr = &configDir
	}

	if err := programSetup(
		cfgPtr,
		table,
		rates,
		drop,
		destStr,
		int(verbose),
		timeout,
		dropQueues,
		dropVia,
		maxHops,
		remoteHash,
		identityPath,
		remoteTO,
		false,
		jsonOut,
	); err != nil {
		if ee, ok := err.(exitError); ok {
			if ee.msg != "" {
				fmt.Fprintln(os.Stderr, ee.msg)
			}
			os.Exit(ee.code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// -------- core ----------

func pythonJSONDumpListOfMaps(maps []map[string]any, keyOrder []string) (string, error) {
	// Match Python json.dumps() formatting (", ", ": ") and stable key order
	// from Reticulum.get_path_table()/get_rate_table().
	//
	// Go's encoding/json sorts map keys lexicographically, which drifts from
	// Python's insertion-order dict serialisation.
	keySet := make(map[string]struct{}, len(keyOrder))
	for _, k := range keyOrder {
		keySet[k] = struct{}{}
	}

	var b strings.Builder
	b.Grow(len(maps) * 64)
	b.WriteByte('[')
	for i, m := range maps {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteByte('{')

		wrote := false
		writeKV := func(k string, v any) error {
			kb, err := json.Marshal(k)
			if err != nil {
				return err
			}
			vb, err := json.Marshal(v)
			if err != nil {
				return err
			}
			if wrote {
				b.WriteString(", ")
			}
			wrote = true
			b.Write(kb)
			b.WriteString(": ")
			b.Write(vb)
			return nil
		}

		for _, k := range keyOrder {
			v, ok := m[k]
			if !ok {
				continue
			}
			if err := writeKV(k, v); err != nil {
				return "", err
			}
		}

		// Any unknown keys (eg remote extensions) go last, sorted for determinism.
		extras := make([]string, 0, len(m))
		for k := range m {
			if _, ok := keySet[k]; ok {
				continue
			}
			extras = append(extras, k)
		}
		sort.Strings(extras)
		for _, k := range extras {
			if err := writeKV(k, m[k]); err != nil {
				return "", err
			}
		}

		b.WriteByte('}')
	}
	b.WriteByte(']')
	return b.String(), nil
}

func programSetup(
	configdir *string,
	table, rates, drop bool,
	destinationHex string,
	verbosity int,
	timeout float64,
	dropQueues, dropVia bool,
	maxHops int,
	remote, managementIdentity string,
	remoteTimeout float64,
	noOutput, jsonOut bool,
) error {
	logLevel := 3 + verbosity
	var err error
	reticulum, err = rns.NewReticulum(configdir, &logLevel, nil, nil, false, nil)
	if err != nil {
		return err
	}

	// remote management
	if remote != "" {
		destLen := (rns.ReticulumTruncatedHashLength / 8) * 2
		if len(remote) != destLen {
			return exitError{code: 20, msg: fmt.Sprintf("Destination length is invalid, must be %d hexadecimal characters (%d bytes).", destLen, destLen/2)}
		}
		identityHash, err := hex.DecodeString(remote)
		if err != nil {
			return exitError{code: 20, msg: "Invalid destination entered. Check your input."}
		}
		remoteHash := destinationHashFromNameAndIdentityHash("rnstransport.remote.management", identityHash)

		if managementIdentity == "" {
			return exitError{code: 20, msg: "Management identity path required (-i)"}
		}
		id, err := rns.IdentityFromFile(expandUser(managementIdentity))
		if err != nil || id == nil {
			return exitError{code: 20, msg: fmt.Sprintf("Could not load management identity from %s", managementIdentity)}
		}

		if err := connectRemote(remoteHash, id, remoteTimeout, noOutput); err != nil {
			return err
		}
		for remoteLink == nil {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// path table
	switch {
	case table:
		return handleTable(destinationHex, maxHops, jsonOut, noOutput)
	case rates:
		return handleRates(destinationHex, jsonOut, noOutput)
	case dropQueues:
		return handleDropQueues(noOutput)
	case drop:
		return handleDropPath(destinationHex, noOutput)
	case dropVia:
		return handleDropVia(destinationHex, noOutput)
	default:
		return handleDiscover(destinationHex, timeout, noOutput)
	}

	return nil
}

// -------- remote link ----------

func connectRemote(destHash []byte, auth *rns.Identity, timeout float64, noOutput bool) error {
	if !rns.HasPath(destHash) {
		if !noOutput {
			fmt.Print("Path to " + rns.PrettyHex(destHash) + " requested ")
		}
		rns.RequestPath(destHash, nil, nil, false)
		start := time.Now()
		for !rns.HasPath(destHash) {
			time.Sleep(100 * time.Millisecond)
			if time.Since(start).Seconds() > timeout {
				if !noOutput {
					fmt.Print("\r                                                          \r")
				}
				return exitError{code: 12, msg: "Path request timed out"}
			}
		}
	}

	remoteIdentity := rns.IdentityRecall(destHash, false)
	if remoteIdentity == nil {
		// Parity helper: allow recalling from identity hash if a full destination recall fails.
		remoteIdentity = rns.IdentityRecall(destHash, true)
	}
	if remoteIdentity == nil {
		return errors.New("could not recall remote identity")
	}

	closed := func(l *rns.Link) {
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
		os.Exit(10)
	}
	established := func(l *rns.Link) {
		if auth != nil {
			l.Identify(auth)
		}
		remoteLink = l
	}

	if !noOutput {
		fmt.Print("\r                                                          \r")
		fmt.Print("Establishing link with remote transport instance... ")
	}

	dest, err := rns.NewDestination(remoteIdentity, rns.DestinationOUT, rns.DestinationSINGLE, "rnstransport", "remote", "management")
	if err != nil {
		return err
	}
	if _, err := rns.NewLink(dest, nil, rns.LinkModeDefault, established, closed); err != nil {
		return err
	}

	return nil
}

// -------- operations ----------

func parseDestHex(s string) ([]byte, error) {
	destLen := (rns.ReticulumTruncatedHashLength / 8) * 2
	if len(s) != destLen {
		return nil, fmt.Errorf("Destination length is invalid, must be %d hexadecimal characters (%d bytes).", destLen, destLen/2)
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("Invalid destination entered. Check your input.")
	}
	return b, nil
}

func parseOptionalDest(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	return parseDestHex(s)
}

func handleTable(destinationHex string, maxHops int, jsonOut, noOutput bool) error {
	destHash, err := parseOptionalDest(destinationHex)
	if err != nil {
		return err
	}

	table, err := obtainPathTable(destHash, maxHops, noOutput)
	if err != nil {
		return err
	}
	table = filterByHops(table, maxHops)

	if len(table) == 0 && jsonOut {
		fmt.Println("[]")
		return nil
	}

	sort.SliceStable(table, func(i, j int) bool {
		ifaceI, _ := table[i]["interface"].(string)
		ifaceJ, _ := table[j]["interface"].(string)
		if ifaceI == ifaceJ {
			return intValue(table[i]["hops"]) < intValue(table[j]["hops"])
		}
		return ifaceI < ifaceJ
	})

	if jsonOut {
		normalized := normalizeForJSON(table)
		out, err := pythonJSONDumpListOfMaps(normalized, []string{"hash", "timestamp", "via", "hops", "expires", "interface"})
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	}

	displayed := 0
	for _, path := range table {
		hash, _ := path["hash"].([]byte)
		if destHash != nil && !equalBytes(destHash, hash) {
			continue
		}
		displayed++

		via, _ := path["via"].([]byte)
		ifName, _ := path["interface"].(string)
		hops := intValue(path["hops"])
		expires := int64Value(path["expires"])

		ms := "s"
		if hops == 1 {
			ms = " "
		}

		fmt.Printf("%s is %d hop%s away via %s on %s expires %s\n",
			rns.PrettyHex(hash),
			hops,
			ms,
			rns.PrettyHex(via),
			ifName,
			rns.TimestampStr(float64(expires)),
		)
	}
	if destHash != nil && displayed == 0 {
		fmt.Println("No path known")
		return exitError{code: 1, msg: ""}
	}
	return nil
}

func obtainPathTable(destHash []byte, maxHops int, noOutput bool) ([]map[string]any, error) {
	if maxHops < 0 {
		if remoteLink == nil {
			return reticulum.GetPathTable(nil), nil
		}
	} else if remoteLink == nil {
		reqMaxPtr := &maxHops
		return reticulum.GetPathTable(reqMaxPtr), nil
	}

	if !noOutput {
		fmt.Print("\r                                                          \r")
		fmt.Print("Sending request... ")
	}
	var requestMax any = nil
	if maxHops >= 0 {
		requestMax = maxHops
	}
	requestData := []any{"table", destHash, requestMax}
	result := remoteLink.Request("/path", requestData, nil, nil, nil, 0)
	receipt, _ := result.(*rns.RequestReceipt)
	if receipt == nil {
		if !noOutput {
			fmt.Print("\r                                                          \r")
			fmt.Println("The remote request could not be started.")
		}
		return nil, exitError{code: 10, msg: ""}
	}
	for !receipt.Concluded() {
		time.Sleep(100 * time.Millisecond)
	}
	resp := receipt.GetResponse()
	if !noOutput {
		fmt.Print("\r                                                          \r")
	}
	if resp == nil {
		if !noOutput {
			fmt.Println("The remote request failed. Likely authentication failure.")
		}
		return nil, exitError{code: 10, msg: ""}
	}
	return decodeTableResponse(resp)
}

func obtainRateTable(destHash []byte, noOutput bool) ([]map[string]any, error) {
	if remoteLink == nil {
		return reticulum.GetRateTable(), nil
	}
	if !noOutput {
		fmt.Print("\r                                                          \r")
		fmt.Print("Sending request... ")
	}
	result := remoteLink.Request("/path", []any{"rates", destHash}, nil, nil, nil, 0)
	receipt, _ := result.(*rns.RequestReceipt)
	if receipt == nil {
		if !noOutput {
			fmt.Print("\r                                                          \r")
			fmt.Println("The remote request could not be started.")
		}
		return nil, exitError{code: 10, msg: ""}
	}
	for !receipt.Concluded() {
		time.Sleep(100 * time.Millisecond)
	}
	resp := receipt.GetResponse()
	if !noOutput {
		fmt.Print("\r                                                          \r")
	}
	if resp == nil {
		if !noOutput {
			fmt.Println("The remote request failed. Likely authentication failure.")
		}
		return nil, exitError{code: 10, msg: ""}
	}
	return decodeTableResponse(resp)
}

func decodeTableResponse(resp any) ([]map[string]any, error) {
	switch v := resp.(type) {
	case []map[string]any:
		return v, nil
	case []any:
		out := make([]map[string]any, 0, len(v))
		for _, entry := range v {
			m, ok := entry.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("unexpected remote entry type %T", entry)
			}
			out = append(out, m)
		}
		return out, nil
	case []byte:
		var parsed []map[string]any
		if err := json.Unmarshal(v, &parsed); err != nil {
			return nil, fmt.Errorf("could not decode remote response: %w", err)
		}
		return parsed, nil
	default:
		return nil, fmt.Errorf("unexpected remote response type %T", resp)
	}
}

func normalizeForJSON(src []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(src))
	for _, entry := range src {
		clone := make(map[string]any, len(entry))
		for k, v := range entry {
			if b, ok := v.([]byte); ok {
				clone[k] = hex.EncodeToString(b)
			} else {
				clone[k] = v
			}
		}
		out = append(out, clone)
	}
	return out
}

func filterByHops(paths []map[string]any, maxHops int) []map[string]any {
	if maxHops < 0 {
		return paths
	}
	filtered := make([]map[string]any, 0, len(paths))
	for _, path := range paths {
		if intValue(path["hops"]) <= maxHops {
			filtered = append(filtered, path)
		}
	}
	return filtered
}

func intValue(v any) int {
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
	case float64:
		return int(val)
	case float32:
		return int(val)
	default:
		return 0
	}
}

func int64Value(v any) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case int32:
		return int64(val)
	case uint64:
		return int64(val)
	case uint32:
		return int64(val)
	case float64:
		return int64(val)
	case float32:
		return int64(val)
	default:
		return 0
	}
}

func extractTimestamps(v any) []int64 {
	switch val := v.(type) {
	case []float64:
		out := make([]int64, 0, len(val))
		for _, f := range val {
			out = append(out, int64(f))
		}
		return out
	case []int64:
		return val
	case []any:
		out := make([]int64, 0, len(val))
		for _, entry := range val {
			out = append(out, int64Value(entry))
		}
		return out
	default:
		return nil
	}
}

func handleRates(destinationHex string, jsonOut, noOutput bool) error {
	destHash, err := parseOptionalDest(destinationHex)
	if err != nil {
		return err
	}

	table, err := obtainRateTable(destHash, noOutput)
	if err != nil {
		return err
	}

	if len(table) == 0 {
		if jsonOut {
			fmt.Println("[]")
			return nil
		}
		fmt.Println("No information available")
		return nil
	}

	sort.SliceStable(table, func(i, j int) bool {
		return int64Value(table[i]["last"]) < int64Value(table[j]["last"])
	})

	if jsonOut {
		normalized := normalizeForJSON(table)
		out, err := pythonJSONDumpListOfMaps(normalized, []string{"hash", "last", "rate_violations", "blocked_until", "timestamps"})
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	}

	displayed := 0
	for _, entry := range table {
		hash, _ := entry["hash"].([]byte)
		if destHash != nil && !equalBytes(destHash, hash) {
			continue
		}
		displayed++

		last := int64Value(entry["last"])
		lastStr := prettyDate(last)

		timestamps := extractTimestamps(entry["timestamps"])
		if len(timestamps) == 0 {
			continue
		}
		startTs := timestamps[0]
		spanSeconds := time.Since(time.Unix(startTs, 0)).Seconds()
		if spanSeconds < 3600 {
			spanSeconds = 3600
		}
		hourRate := float64(len(timestamps)) / (spanSeconds / 3600.0)
		hourRateRounded := math.Round(hourRate*1000) / 1000
		var rateStr string
		if math.Mod(hourRateRounded, 1) == 0 {
			rateStr = fmt.Sprintf("%.0f", hourRateRounded)
		} else {
			rateStr = strconv.FormatFloat(hourRateRounded, 'f', -1, 64)
		}

		rateViol := intValue(entry["rate_violations"])
		blockedUntil := int64Value(entry["blocked_until"])

		rvStr := ""
		if rateViol > 0 {
			suffix := "s"
			if rateViol == 1 {
				suffix = ""
			}
			rvStr = fmt.Sprintf(", %d active rate violation%s", rateViol, suffix)
		}

		blStr := ""
		now := time.Now().Unix()
		if blockedUntil > now {
			bli := now - (blockedUntil - now)
			blStr = ", new announces allowed in " + prettyDate(bli)
		}

		fmt.Printf("%s last heard %s ago, %s announces/hour in the last %s%s%s\n",
			rns.PrettyHex(hash),
			lastStr,
			rateStr,
			prettyDate(startTs),
			rvStr,
			blStr,
		)
	}

	if destHash != nil && displayed == 0 {
		fmt.Println("No information available")
		return exitError{code: 1, msg: ""}
	}
	return nil
}

func handleDropQueues(noOutput bool) error {
	if remoteLink != nil {
		if !noOutput {
			fmt.Print("\r                                                          \r")
			fmt.Println("Dropping announce queues on remote instances not yet implemented")
		}
		return exitError{code: 255, msg: ""}
	}
	fmt.Println("Dropping announce queues on all interfaces...")
	reticulum.DropAnnounceQueues()
	return nil
}

func handleDropPath(destinationHex string, noOutput bool) error {
	if remoteLink != nil {
		if !noOutput {
			fmt.Print("\r                                                          \r")
			fmt.Println("Dropping path on remote instances not yet implemented")
		}
		return exitError{code: 255, msg: ""}
	}
	destHash, err := parseDestHex(destinationHex)
	if err != nil {
		return err
	}
	if reticulum.DropPath(destHash) {
		fmt.Println("Dropped path to " + rns.PrettyHex(destHash))
	} else {
		fmt.Println("Unable to drop path to " + rns.PrettyHex(destHash) + ". Does it exist?")
		return exitError{code: 1, msg: ""}
	}
	return nil
}

func handleDropVia(destinationHex string, noOutput bool) error {
	if remoteLink != nil {
		if !noOutput {
			fmt.Print("\r                                                          \r")
			fmt.Println("Dropping all paths via specific transport instance on remote instances yet not implemented")
		}
		return exitError{code: 255, msg: ""}
	}
	destHash, err := parseDestHex(destinationHex)
	if err != nil {
		return err
	}
	if count := reticulum.DropAllVia(destHash); count > 0 {
		fmt.Println("Dropped all paths via " + rns.PrettyHex(destHash))
	} else {
		fmt.Println("Unable to drop paths via " + rns.PrettyHex(destHash) + ". Does the transport instance exist?")
		return exitError{code: 1, msg: ""}
	}
	return nil
}

func handleDiscover(destinationHex string, timeout float64, noOutput bool) error {
	if remoteLink != nil {
		if !noOutput {
			fmt.Print("\r                                                          \r")
			fmt.Println("Requesting paths on remote instances not implemented")
		}
		return exitError{code: 255, msg: ""}
	}
	destHash, err := parseDestHex(destinationHex)
	if err != nil {
		return err
	}

	if !rns.HasPath(destHash) {
		rns.RequestPath(destHash, nil, nil, false)
		// Python prints: "requested  " with end=" ", so there is one space
		// before the 2-char spinner field.
		fmt.Print("Path to " + rns.PrettyHex(destHash) + " requested   ")
	}
	syms := []rune("⢄⢂⢁⡁⡈⡐⡠")
	i := 0
	limit := time.Now().Add(time.Duration(timeout * float64(time.Second)))

	for !rns.HasPath(destHash) && time.Now().Before(limit) {
		time.Sleep(100 * time.Millisecond)
		fmt.Printf("\b\b%c ", syms[i])
		i = (i + 1) % len(syms)
	}

	if rns.HasPath(destHash) {
		nextHopBytes := reticulum.GetNextHop(destHash)
		if nextHopBytes == nil {
			fmt.Print("\r                                                       \r")
			fmt.Println("Error: Invalid path data returned")
			return exitError{code: 1, msg: ""}
		}
		nextHop := rns.PrettyHex(nextHopBytes)
		hops := rns.HopsTo(destHash)
		ifName := reticulum.GetNextHopIfName(destHash)

		// The path entry can become visible before hop count is finalised.
		// Wait briefly for a stable hop count unless this is clearly a local-client route.
		if hops == 0 && ifName != "" && ifName != "None" && !strings.HasPrefix(ifName, "LocalInterface") {
			stableDeadline := time.Now().Add(750 * time.Millisecond)
			for hops == 0 && time.Now().Before(stableDeadline) {
				time.Sleep(50 * time.Millisecond)
				hops = rns.HopsTo(destHash)
			}
		}

		ms := ""
		if hops != 1 {
			ms = "s"
		}
		fmt.Printf("\rPath found, destination %s is %d hop%s away via %s on %s\n",
			rns.PrettyHex(destHash), hops, ms, nextHop, ifName)
		return nil
	}
	fmt.Print("\r                                                       \rPath not found\n")
	return exitError{code: 1, msg: ""}
}

func pythonJSONSpacing(s string) string {
	// Python json.dumps() uses separators (", ", ": ") by default.
	inString := false
	esc := false
	var b strings.Builder
	b.Grow(len(s) + len(s)/8)
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inString {
			b.WriteByte(ch)
			if esc {
				esc = false
				continue
			}
			if ch == '\\' {
				esc = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
			b.WriteByte(ch)
		case ':':
			b.WriteByte(':')
			b.WriteByte(' ')
		case ',':
			b.WriteByte(',')
			b.WriteByte(' ')
		default:
			b.WriteByte(ch)
		}
	}
	return b.String()
}

// -------- helpers ----------

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func expandUser(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		if len(path) == 1 {
			return home
		}
		if path[1] == '/' || path[1] == '\\' {
			return home + path[1:]
		}
	}
	return path
}

func prettyDate(ts int64) string {
	now := time.Now()
	t := time.Unix(ts, 0)
	diff := now.Sub(t)
	sec := int(diff.Seconds())
	days := int(diff.Hours() / 24)

	if days < 0 {
		return ""
	}
	if days == 0 {
		switch {
		case sec < 10:
			return fmt.Sprintf("%d seconds", sec)
		case sec < 60:
			return fmt.Sprintf("%d seconds", sec)
		case sec < 120:
			return "1 minute"
		case sec < 3600:
			return fmt.Sprintf("%d minutes", sec/60)
		case sec < 7200:
			return "an hour"
		case sec < 86400:
			return fmt.Sprintf("%d hours", sec/3600)
		}
	}
	if days == 1 {
		return "1 day"
	}
	if days < 7 {
		return fmt.Sprintf("%d days", days)
	}
	if days < 31 {
		return fmt.Sprintf("%d weeks", days/7)
	}
	if days < 365 {
		return fmt.Sprintf("%d months", days/30)
	}
	return fmt.Sprintf("%d years", days/365)
}
