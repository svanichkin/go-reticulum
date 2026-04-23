package rns

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	ifaces "github.com/svanichkin/go-reticulum/rns/interfaces"
	umsgpack "github.com/svanichkin/go-reticulum/rns/vendor"
)

const (
	discoveryFieldName          = 0xFF
	discoveryFieldTransportID   = 0xFE
	discoveryFieldInterfaceType = 0x00
	discoveryFieldTransport     = 0x01
	discoveryFieldReachableOn   = 0x02
	discoveryFieldLatitude      = 0x03
	discoveryFieldLongitude     = 0x04
	discoveryFieldHeight        = 0x05
	discoveryFieldPort          = 0x06
	discoveryFieldIFACNetname   = 0x07
	discoveryFieldIFACNetkey    = 0x08
	discoveryFieldFrequency     = 0x09
	discoveryFieldBandwidth     = 0x0A
	discoveryFieldSpreading     = 0x0B
	discoveryFieldCodingRate    = 0x0C
	discoveryFieldModulation    = 0x0D
	discoveryFieldChannel       = 0x0E
	discoveryFlagSigned         = 0b00000001
	discoveryFlagEncrypted      = 0b00000010
	discoveryAnnouncerInterval  = 60 * time.Second
	discoveryStampDefaultValue  = 14
	discoveryWorkblockRounds    = 20
	discoveryThresholdUnknown   = 24 * time.Hour
	discoveryThresholdStale     = 3 * 24 * time.Hour
	discoveryThresholdRemove    = 7 * 24 * time.Hour
	discoveryMonitorInterval    = 5 * time.Second
	discoveryDetachThreshold    = 12 * time.Second
	discoveryAutoconnectBitrate = 5_000_000
	blackholeUpdaterInitialWait = 20 * time.Second
	blackholeUpdaterJobInterval = 60 * time.Second
	blackholeUpdateInterval     = 1 * time.Hour
	blackholeSourceTimeout      = 25 * time.Second
)

var discoveryAnnouncerInterfaceTypes = map[string]struct{}{
	"BackboneInterface":  {},
	"TCPServerInterface": {},
	"TCPClientInterface": {},
	"RNodeInterface":     {},
	"WeaveInterface":     {},
	"I2PInterface":       {},
	"KISSInterface":      {},
}
var discoveryInterfaceTypes = map[string]struct{}{
	"BackboneInterface":  {},
	"TCPServerInterface": {},
	"I2PInterface":       {},
	"RNodeInterface":     {},
	"WeaveInterface":     {},
	"KISSInterface":      {},
}

type DiscoveryStamper interface {
	StampSize() int
	GenerateStamp(infoHash []byte, stampCost int, expandRounds int) ([]byte, int, error)
	StampWorkblock(infoHash []byte, expandRounds int) []byte
	StampValue(workblock, stamp []byte) int
	StampValid(stamp []byte, requiredValue int, workblock []byte) bool
}

// DiscoveryStampProvider is the pluggable proof-of-work backend used by
// Discovery.py in Python via LXMF. The runtime compiles without it, but actual
// discovery announce generation/validation requires a compatible implementation.
var DiscoveryStampProvider DiscoveryStamper
var discoveryAutoconnectInterfaceFactory = func(info map[string]any) (*Interface, error) {
	name, _ := info["name"].(string)
	reachableOn, _ := info["reachable_on"].(string)
	port := func(v any) int {
		switch x := v.(type) {
		case int:
			return x
		case int64:
			return int(x)
		case uint64:
			return int(x)
		default:
			return 0
		}
	}(info["port"])
	if strings.TrimSpace(name) == "" || strings.TrimSpace(reachableOn) == "" || port <= 0 {
		return nil, errors.New("discovered interface missing reachable_on or port")
	}
	return ifaces.NewBackboneClientInterface(name, map[string]string{
		"target_host": reachableOn,
		"target_port": strconv.Itoa(port),
	})
}
var discoveryPanic = Panic

type InterfaceAnnouncer struct {
	shouldRun  bool
	mu         sync.Mutex
	dest       *Destination
	stampCache map[string][]byte
}

type InterfaceAnnounceHandler struct {
	requiredValue int
	callback      func(map[string]any)
}

type InterfaceDiscovery struct {
	requiredValue int
	callback      func(map[string]any)
	storagePath   string
	handler       *InterfaceAnnounceHandler
	mu            sync.Mutex
	monitored     []*Interface
	monitoring    bool
	monitorEvery  time.Duration
	detachAfter   time.Duration
	initialAuto   bool
}

type BlackholeUpdater struct {
	mu             sync.Mutex
	shouldRun      bool
	lastUpdates    map[hashKey]time.Time
	initialWait    time.Duration
	jobInterval    time.Duration
	updateInterval time.Duration
	sourceTimeout  time.Duration
	awaitPath      func([]byte, float64, *Interface) bool
	fetchList      func([]byte, time.Duration) (any, error)
	sleep          func(time.Duration)
}

func init() {
	interfaceAnnouncerFactory = func() transportBackgroundWorker {
		return NewInterfaceAnnouncer()
	}
	discoveryHandlerFactory = func(requiredValue int, discoverInterfaces bool) any {
		d, err := NewInterfaceDiscovery(requiredValue, nil, discoverInterfaces)
		if err != nil {
			Logf(LogError, "Could not initialise interface discovery: %v", err)
			return nil
		}
		return d
	}
	blackholeUpdaterFactory = func() transportBackgroundWorker {
		return NewBlackholeUpdater()
	}
}

func NewInterfaceAnnouncer() *InterfaceAnnouncer {
	if DiscoveryStampProvider == nil {
		Log("Using on-network interface discovery requires the LXMF module to be installed.", LogCritical)
		Log("You can install it with the command: pip install lxmf", LogCritical)
		if discoveryPanic != nil {
			discoveryPanic()
		}
		panic(errors.New("Using on-network interface discovery requires the LXMF module to be installed."))
	}
	var identity *Identity
	if HasNetworkIdentity() {
		identity = NetworkIdentity
	} else {
		identity = TransportIdentity
	}
	if identity == nil {
		return &InterfaceAnnouncer{stampCache: make(map[string][]byte)}
	}
	dest, err := NewDestination(identity, DestinationIN, DestinationSINGLE, TransportAppName, "discovery", "interface")
	if err != nil {
		Logf(LogError, "Could not create discovery destination: %v", err)
		return &InterfaceAnnouncer{stampCache: make(map[string][]byte)}
	}
	return &InterfaceAnnouncer{dest: dest, stampCache: make(map[string][]byte)}
}

func (a *InterfaceAnnouncer) Start() {
	if a == nil {
		return
	}
	a.mu.Lock()
	if a.shouldRun {
		a.mu.Unlock()
		return
	}
	a.shouldRun = true
	a.mu.Unlock()

	go a.job()
}

func (a *InterfaceAnnouncer) Stop() {
	if a == nil {
		return
	}
	a.mu.Lock()
	a.shouldRun = false
	a.mu.Unlock()
}

func (a *InterfaceAnnouncer) job() {
	for {
		a.mu.Lock()
		shouldRun := a.shouldRun
		a.mu.Unlock()
		if !shouldRun {
			return
		}
		time.Sleep(discoveryAnnouncerInterval)
		a.mu.Lock()
		shouldRun = a.shouldRun
		a.mu.Unlock()
		if !shouldRun {
			return
		}
		if a == nil || a.dest == nil {
			continue
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					Logf(LogError, "Error while preparing interface discovery announces: %v", r)
					TraceException(r)
				}
			}()
			now := time.Now()
			due := make([]*Interface, 0)
			for _, ifc := range Interfaces {
				if ifc == nil || !ifc.SupportsDiscovery() || !ifc.Discoverable {
					continue
				}
				if ifc.LastDiscoveryAnnounce.IsZero() || now.Sub(ifc.LastDiscoveryAnnounce) > ifc.DiscoveryAnnounceInterval {
					due = append(due, ifc)
				}
			}
			sort.SliceStable(due, func(i, j int) bool {
				return now.Sub(due[i].LastDiscoveryAnnounce) > now.Sub(due[j].LastDiscoveryAnnounce)
			})
			if len(due) == 0 {
				return
			}
			selected := due[0]
			selected.LastDiscoveryAnnounce = time.Now()
			Logf(LogDebug, "Preparing interface discovery announce for %s", selected.Name)
			appData, err := a.GetInterfaceAnnounceData(selected)
			if err != nil {
				Logf(LogError, "Could not generate interface discovery announce data for %s", selected.Name)
				return
			}
			Logf(LogDebug, "Sending interface discovery announce for %s with %dB payload", selected.Name, len(appData))
			a.dest.Announce(appData, false, nil, nil, true)
		}()
	}
}

func (a *InterfaceAnnouncer) GetInterfaceAnnounceData(ifc *Interface) ([]byte, error) {
	if ifc == nil {
		return nil, errors.New("nil interface")
	}
	if DiscoveryStampProvider == nil {
		return nil, errors.New("discovery stamper provider is not configured")
	}

	ifType := ifc.Type
	if _, ok := discoveryAnnouncerInterfaceTypes[ifType]; !ok {
		return nil, fmt.Errorf("interface type %q does not support discovery", ifType)
	}
	if ifType == "TCPClientInterface" && !ifc.DiscoveryKISSFraming() {
		Logf(LogError, "Invalid interface discovery configuration for %s, aborting discovery announce", ifc.Name)
		return nil, fmt.Errorf("invalid interface discovery configuration for %s", ifc.Name)
	}

	flags := byte(0)
	info := map[any]any{
		discoveryFieldInterfaceType: ifType,
		discoveryFieldTransport:     TransportEnabled(),
		discoveryFieldTransportID:   append([]byte(nil), TransportIdentity.Hash...),
		discoveryFieldName:          sanitizeDiscoveryString(ifc.DiscoveryNameValue()),
		discoveryFieldLatitude: func() any {
			if ifc.DiscoveryLatitude == nil {
				return nil
			}
			return *ifc.DiscoveryLatitude
		}(),
		discoveryFieldLongitude: func() any {
			if ifc.DiscoveryLongitude == nil {
				return nil
			}
			return *ifc.DiscoveryLongitude
		}(),
		discoveryFieldHeight: func() any {
			if ifc.DiscoveryHeight == nil {
				return nil
			}
			return *ifc.DiscoveryHeight
		}(),
	}

	switch ifType {
	case "BackboneInterface", "TCPServerInterface":
		reachableOn := sanitizeDiscoveryString(ifc.DiscoveryReachableOnValue())
		if !umsgpack.IsWindows() {
			resolveExecutablePath := func(v string) (string, bool) {
				if strings.TrimSpace(v) == "" {
					return "", false
				}
				expanded := v
				if strings.HasPrefix(expanded, "~") {
					if home, err := os.UserHomeDir(); err == nil {
						switch {
						case expanded == "~":
							expanded = home
						case strings.HasPrefix(expanded, "~/"):
							expanded = filepath.Join(home, strings.TrimPrefix(expanded, "~/"))
						}
					}
				}
				info, err := os.Stat(expanded)
				if err != nil || info.IsDir() {
					return "", false
				}
				if info.Mode()&0o111 == 0 {
					return "", false
				}
				return expanded, true
			}
			if execPath, ok := resolveExecutablePath(reachableOn); ok {
				Logf(LogDebug, "Evaluating reachable_on from executable at %s", execPath)
				out, err := exec.Command(execPath).Output()
				if err != nil {
					Logf(LogError, "Error while getting reachable_on from executable at %s: %v", ifc.DiscoveryReachableOnValue(), err)
					Log("Aborting discovery announce", LogError)
					return nil, fmt.Errorf("error while getting reachable_on from executable at %s: %w", ifc.DiscoveryReachableOnValue(), err)
				}
				reachableOn = sanitizeDiscoveryString(string(out))
				if !(isIPAddress(reachableOn) || isHostname(reachableOn)) {
					execErr := fmt.Errorf("Valid IP address or hostname was not found in external script output %q", reachableOn)
					Logf(LogError, "Error while getting reachable_on from executable at %s: %v", ifc.DiscoveryReachableOnValue(), execErr)
					Log("Aborting discovery announce", LogError)
					return nil, execErr
				}
			}
			if !(isIPAddress(reachableOn) || isHostname(reachableOn)) {
				Logf(LogError, "The configured reachable_on parameter \"%s\" for %s is not a valid IP address or hostname", reachableOn, ifc.Name)
				Log("Aborting discovery announce", LogError)
				return nil, fmt.Errorf("invalid reachable_on %q for %s", reachableOn, ifc.Name)
			}
		}
		port := ifc.DiscoveryPortValue()
		if port == nil || *port <= 0 {
			return nil, fmt.Errorf("missing discovery port for %s", ifc.Name)
		}
		info[discoveryFieldReachableOn] = reachableOn
		info[discoveryFieldPort] = *port
	case "I2PInterface":
		if connectable := ifc.I2PConnectable(); connectable != nil && *connectable {
			if b32 := ifc.I2PB32(); b32 != nil && strings.TrimSpace(*b32) != "" {
				info[discoveryFieldReachableOn] = sanitizeDiscoveryString(*b32)
			}
		}
	case "RNodeInterface":
		frequency, bandwidth, sf, cr, ok := ifc.DiscoveryRNodeRadioParams()
		if ok {
			info[discoveryFieldFrequency] = frequency
			info[discoveryFieldBandwidth] = bandwidth
			info[discoveryFieldSpreading] = sf
			info[discoveryFieldCodingRate] = cr
		}
	case "WeaveInterface":
		frequency, bandwidth, channel, modulation := ifc.DiscoveryWeaveParams()
		if frequency != nil {
			info[discoveryFieldFrequency] = *frequency
		}
		if bandwidth != nil {
			info[discoveryFieldBandwidth] = *bandwidth
		}
		if channel != nil {
			info[discoveryFieldChannel] = *channel
		}
		if modulation != "" {
			info[discoveryFieldModulation] = sanitizeDiscoveryString(modulation)
		}
	}

	if ifType == "KISSInterface" || (ifType == "TCPClientInterface" && ifc.DiscoveryKISSFraming()) {
		info[discoveryFieldInterfaceType] = "KISSInterface"
		if ifc.DiscoveryFrequency != nil {
			info[discoveryFieldFrequency] = *ifc.DiscoveryFrequency
		} else {
			info[discoveryFieldFrequency] = nil
		}
		if ifc.DiscoveryBandwidth != nil {
			info[discoveryFieldBandwidth] = *ifc.DiscoveryBandwidth
		} else {
			info[discoveryFieldBandwidth] = nil
		}
		info[discoveryFieldModulation] = sanitizeDiscoveryString(ifc.DiscoveryModulation)
	}

	if ifc.DiscoveryPublishIFAC {
		info[discoveryFieldIFACNetname] = sanitizeDiscoveryString(ifc.IFACNetname())
		info[discoveryFieldIFACNetkey] = sanitizeDiscoveryString(ifc.IFACNetkey())
	}

	packedInfo := bytes.Buffer{}
	stringOf := func(v any) string {
		switch x := v.(type) {
		case string:
			return x
		case []byte:
			return string(x)
		default:
			return ""
		}
	}
	lookup := func(raw map[any]any, want int) (any, bool) {
		for k, v := range raw {
			switch x := k.(type) {
			case int:
				if x == want {
					return v, true
				}
			case int64:
				if int(x) == want {
					return v, true
				}
			case uint64:
				if int(x) == want {
					return v, true
				}
			case uint8:
				if int(x) == want {
					return v, true
				}
			}
		}
		return nil, false
	}
	writeHeader := func(buf *bytes.Buffer, n int) error {
		switch {
		case n <= 15:
			return buf.WriteByte(byte(0x80 | n))
		case n <= 0xFFFF:
			if err := buf.WriteByte(0xDE); err != nil {
				return err
			}
			var raw [2]byte
			binary.BigEndian.PutUint16(raw[:], uint16(n))
			_, err := buf.Write(raw[:])
			return err
		default:
			if err := buf.WriteByte(0xDF); err != nil {
				return err
			}
			var raw [4]byte
			binary.BigEndian.PutUint32(raw[:], uint32(n))
			_, err := buf.Write(raw[:])
			return err
		}
	}
	keys := []int{
		discoveryFieldInterfaceType,
		discoveryFieldTransport,
		discoveryFieldTransportID,
		discoveryFieldName,
		discoveryFieldLatitude,
		discoveryFieldLongitude,
		discoveryFieldHeight,
	}
	ifTypeVal, _ := lookup(info, discoveryFieldInterfaceType)
	switch sanitizeDiscoveryString(stringOf(ifTypeVal)) {
	case "BackboneInterface", "TCPServerInterface":
		keys = append(keys, discoveryFieldReachableOn, discoveryFieldPort)
	case "I2PInterface":
		keys = append(keys, discoveryFieldReachableOn)
	case "RNodeInterface":
		keys = append(keys, discoveryFieldFrequency, discoveryFieldBandwidth, discoveryFieldSpreading, discoveryFieldCodingRate)
	case "WeaveInterface":
		keys = append(keys, discoveryFieldFrequency, discoveryFieldBandwidth, discoveryFieldChannel, discoveryFieldModulation)
	case "KISSInterface":
		keys = append(keys, discoveryFieldFrequency, discoveryFieldBandwidth, discoveryFieldModulation)
	}
	if _, ok := lookup(info, discoveryFieldIFACNetname); ok {
		keys = append(keys, discoveryFieldIFACNetname)
	}
	if _, ok := lookup(info, discoveryFieldIFACNetkey); ok {
		keys = append(keys, discoveryFieldIFACNetkey)
	}
	filtered := make([]int, 0, len(keys))
	seen := make(map[int]struct{}, len(keys))
	for _, key := range keys {
		if _, ok := seen[key]; ok {
			continue
		}
		if _, ok := lookup(info, key); !ok {
			continue
		}
		seen[key] = struct{}{}
		filtered = append(filtered, key)
	}
	keys = filtered
	if err := writeHeader(&packedInfo, len(keys)); err != nil {
		return nil, err
	}
	for _, key := range keys {
		value, ok := lookup(info, key)
		if !ok {
			continue
		}
		packedKey, err := umsgpack.Packb(key)
		if err != nil {
			return nil, err
		}
		if _, err := packedInfo.Write(packedKey); err != nil {
			return nil, err
		}
		packedValue, err := umsgpack.Packb(value)
		if err != nil {
			return nil, err
		}
		if _, err := packedInfo.Write(packedValue); err != nil {
			return nil, err
		}
	}
	infoHash := FullHash(packedInfo.Bytes())
	stampCost := discoveryStampDefaultValue
	if ifc.DiscoveryStampValue != nil && *ifc.DiscoveryStampValue > 0 {
		stampCost = *ifc.DiscoveryStampValue
	}
	stampKey := string(infoHash)
	var stamp []byte
	var err error
	a.mu.Lock()
	if cached, ok := a.stampCache[stampKey]; ok && len(cached) > 0 {
		stamp = append([]byte(nil), cached...)
	}
	a.mu.Unlock()
	if len(stamp) == 0 {
		stamp, _, err = DiscoveryStampProvider.GenerateStamp(infoHash, stampCost, discoveryWorkblockRounds)
		if err != nil {
			return nil, err
		}
		a.mu.Lock()
		if a.stampCache == nil {
			a.stampCache = make(map[string][]byte)
		}
		a.stampCache[stampKey] = append([]byte(nil), stamp...)
		a.mu.Unlock()
	}
	payload := append(append([]byte(nil), packedInfo.Bytes()...), stamp...)
	if ifc.DiscoveryEncrypt {
		if !HasNetworkIdentity() {
			Logf(LogError, "Discovery encryption requested for %s, but no network identity configured. Aborting discovery announce.", ifc.Name)
			return nil, errors.New("discovery encryption requested without network identity")
		}
		flags |= discoveryFlagEncrypted
		payload, err = NetworkIdentity.Encrypt(payload, nil)
		if err != nil {
			return nil, err
		}
	}
	return append([]byte{flags}, payload...), nil
}

func NewInterfaceAnnounceHandler(requiredValue int, callback func(map[string]any)) *InterfaceAnnounceHandler {
	if DiscoveryStampProvider == nil {
		Log("Using on-network interface discovery requires the LXMF module to be installed.", LogCritical)
		Log("You can install it with the command: pip install lxmf", LogCritical)
		if discoveryPanic != nil {
			discoveryPanic()
		}
		panic(errors.New("Using on-network interface discovery requires the LXMF module to be installed."))
	}
	if requiredValue <= 0 {
		requiredValue = discoveryStampDefaultValue
	}
	return &InterfaceAnnounceHandler{requiredValue: requiredValue, callback: callback}
}

func (h *InterfaceAnnounceHandler) AspectFilter() any {
	return TransportAppName + ".discovery.interface"
}

func (h *InterfaceAnnounceHandler) ReceivedAnnounce(destinationHash []byte, announcedIdentity *Identity, appData []byte) {
	defer func() {
		if r := recover(); r != nil {
			Logf(LogDebug, "An error occurred while trying to decode discovered interface. The contained exception was: %v", r)
		}
	}()
	if h == nil || announcedIdentity == nil || len(appData) <= 1 || DiscoveryStampProvider == nil {
		return
	}
	sources := InterfaceDiscoverySources()
	if len(sources) > 0 {
		allowed := false
		for _, src := range sources {
			if bytesEqual(src, announcedIdentity.Hash) {
				allowed = true
				break
			}
		}
		if !allowed {
			Logf(LogDebug, "Interface discovered from non-authorized network identity %s, ignoring", PrettyHexRep(announcedIdentity.Hash))
			return
		}
	}

	flags := appData[0]
	_ = flags & discoveryFlagSigned
	payload := append([]byte(nil), appData[1:]...)
	if flags&discoveryFlagEncrypted != 0 {
		if !HasNetworkIdentity() {
			return
		}
		plain, err := NetworkIdentity.Decrypt(payload, nil, false)
		if err != nil {
			return
		}
		payload = plain
	}

	stampSize := DiscoveryStampProvider.StampSize()
	if stampSize <= 0 || len(payload) <= stampSize {
		return
	}
	stamp := append([]byte(nil), payload[len(payload)-stampSize:]...)
	packed := append([]byte(nil), payload[:len(payload)-stampSize]...)
	infoHash := FullHash(packed)
	workblock := DiscoveryStampProvider.StampWorkblock(infoHash, discoveryWorkblockRounds)
	if !DiscoveryStampProvider.StampValid(stamp, h.requiredValue, workblock) {
		Log("Ignored discovered interface with invalid stamp", LogDebug)
		return
	}
	value := DiscoveryStampProvider.StampValue(workblock, stamp)
	if value < h.requiredValue {
		Logf(LogDebug, "Ignored discovered interface with stamp value %d", value)
		return
	}

	raw := map[any]any{}
	if err := umsgpack.Unpackb(packed, &raw); err != nil {
		return
	}
	if raw == nil || announcedIdentity == nil {
		return
	}
	lookup := func(raw map[any]any, want int) (any, bool) {
		for k, v := range raw {
			switch x := k.(type) {
			case int:
				if x == want {
					return v, true
				}
			case int64:
				if int(x) == want {
					return v, true
				}
			case uint64:
				if int(x) == want {
					return v, true
				}
			case uint8:
				if int(x) == want {
					return v, true
				}
			}
		}
		return nil, false
	}
	stringOf := func(v any) string {
		switch x := v.(type) {
		case string:
			return x
		case []byte:
			return string(x)
		default:
			return ""
		}
	}
	intOf := func(v any) int {
		switch x := v.(type) {
		case int:
			return x
		case int64:
			return int(x)
		case uint64:
			return int(x)
		case float64:
			return int(x)
		default:
			return 0
		}
	}
	maybeInt := func(v any) *int {
		switch x := v.(type) {
		case int:
			return &x
		case int64:
			n := int(x)
			return &n
		case uint64:
			n := int(x)
			return &n
		case float64:
			n := int(x)
			return &n
		default:
			return nil
		}
	}
	floatOf := func(v any) float64 {
		switch x := v.(type) {
		case float64:
			return x
		case float32:
			return float64(x)
		case int:
			return float64(x)
		case int64:
			return float64(x)
		default:
			return 0
		}
	}
	ifTypeVal, _ := lookup(raw, discoveryFieldInterfaceType)
	ifType := sanitizeDiscoveryString(stringOf(ifTypeVal))
	if ifType == "" {
		return
	}
	nameVal, _ := lookup(raw, discoveryFieldName)
	name := stringOf(nameVal)
	if name == "" {
		name = "Discovered " + ifType
	}
	transportIDVal, _ := lookup(raw, discoveryFieldTransportID)
	transportID := func(v any) []byte {
		switch x := v.(type) {
		case []byte:
			return append([]byte(nil), x...)
		case string:
			return []byte(x)
		default:
			return nil
		}
	}(transportIDVal)
	info := map[string]any{
		"type": ifType,
		"transport": func() bool {
			value, _ := lookup(raw, discoveryFieldTransport)
			if x, ok := value.(bool); ok {
				return x
			}
			return false
		}(),
		"name":         name,
		"received":     float64(time.Now().UnixNano()) / 1e9,
		"stamp":        stamp,
		"value":        value,
		"transport_id": strings.ToLower(hex.EncodeToString(transportID)),
		"network_id":   strings.ToLower(hex.EncodeToString(announcedIdentity.Hash)),
		"hops": func() int {
			if len(destinationHash) == 0 {
				return 0
			}
			return HopsTo(destinationHash)
		}(),
	}
	if latRaw, ok := lookup(raw, discoveryFieldLatitude); ok {
		if latRaw == nil {
			info["latitude"] = nil
		} else {
			info["latitude"] = floatOf(latRaw)
		}
	} else {
		info["latitude"] = nil
	}
	if lonRaw, ok := lookup(raw, discoveryFieldLongitude); ok {
		if lonRaw == nil {
			info["longitude"] = nil
		} else {
			info["longitude"] = floatOf(lonRaw)
		}
	} else {
		info["longitude"] = nil
	}
	if heightRaw, ok := lookup(raw, discoveryFieldHeight); ok {
		if heightRaw == nil {
			info["height"] = nil
		} else {
			info["height"] = floatOf(heightRaw)
		}
	} else {
		info["height"] = nil
	}
	if reachableOnVal, ok := lookup(raw, discoveryFieldReachableOn); ok {
		reachableOn := stringOf(reachableOnVal)
		if !(isIPAddress(reachableOn) || isHostname(reachableOn)) {
			panic(errors.New("Invalid data in reachable_on field of announce"))
		}
		info["reachable_on"] = reachableOn
	}
	if portVal, _ := lookup(raw, discoveryFieldPort); maybeInt(portVal) != nil {
		port := maybeInt(portVal)
		info["port"] = *port
	}
	if netnameVal, ok := lookup(raw, discoveryFieldIFACNetname); ok {
		info["ifac_netname"] = stringOf(netnameVal)
	}
	if netkeyVal, ok := lookup(raw, discoveryFieldIFACNetkey); ok {
		info["ifac_netkey"] = stringOf(netkeyVal)
	}
	if vVal, _ := lookup(raw, discoveryFieldFrequency); maybeInt(vVal) != nil {
		v := maybeInt(vVal)
		info["frequency"] = *v
	}
	if vVal, _ := lookup(raw, discoveryFieldBandwidth); maybeInt(vVal) != nil {
		v := maybeInt(vVal)
		info["bandwidth"] = *v
	}
	if vVal, _ := lookup(raw, discoveryFieldSpreading); maybeInt(vVal) != nil {
		v := maybeInt(vVal)
		info["sf"] = *v
	}
	if vVal, _ := lookup(raw, discoveryFieldCodingRate); maybeInt(vVal) != nil {
		v := maybeInt(vVal)
		info["cr"] = *v
	}
	if vVal, _ := lookup(raw, discoveryFieldChannel); maybeInt(vVal) != nil {
		v := maybeInt(vVal)
		info["channel"] = *v
	}
	if modulationVal, _ := lookup(raw, discoveryFieldModulation); sanitizeDiscoveryString(stringOf(modulationVal)) != "" {
		modulation := sanitizeDiscoveryString(stringOf(modulationVal))
		info["modulation"] = modulation
	}
	cfgName, _ := info["name"].(string)
	cfgTransportID, _ := info["transport_id"].(string)
	cfgNetname, _ := info["ifac_netname"].(string)
	cfgNetkey, _ := info["ifac_netkey"].(string)
	cfgNetnameStr := ""
	if strings.TrimSpace(cfgNetname) != "" {
		cfgNetnameStr = "\n  network_name = " + cfgNetname
	}
	cfgNetkeyStr := ""
	if strings.TrimSpace(cfgNetkey) != "" {
		cfgNetkeyStr = "\n  passphrase = " + cfgNetkey
	}
	cfgIdentityStr := ""
	if strings.TrimSpace(cfgTransportID) != "" {
		cfgIdentityStr = "\n  transport_identity = " + cfgTransportID
	}

	var configEntry string
	switch ifType {
	case "BackboneInterface", "TCPServerInterface":
		reachableOn, ok := info["reachable_on"].(string)
		if !ok {
			return
		}
		portValue, ok := info["port"]
		if !ok {
			return
		}
		port := intOf(portValue)
		connectionType := "BackboneInterface"
		remoteKey := "remote"
		if umsgpack.IsWindows() {
			connectionType = "TCPClientInterface"
			remoteKey = "target_host"
		}
		configEntry = fmt.Sprintf("[[%s]]\n  type = %s\n  enabled = yes\n  %s = %s\n  target_port = %d%s%s%s",
			cfgName, connectionType, remoteKey, reachableOn, port, cfgIdentityStr, cfgNetnameStr, cfgNetkeyStr)
	case "I2PInterface":
		reachableOn, ok := info["reachable_on"].(string)
		if !ok {
			return
		}
		configEntry = fmt.Sprintf("[[%s]]\n  type = I2PInterface\n  enabled = yes\n  peers = %s%s%s%s",
			cfgName, reachableOn, cfgIdentityStr, cfgNetnameStr, cfgNetkeyStr)
	case "RNodeInterface":
		if _, ok := info["frequency"]; !ok {
			return
		}
		if _, ok := info["bandwidth"]; !ok {
			return
		}
		if _, ok := info["sf"]; !ok {
			return
		}
		if _, ok := info["cr"]; !ok {
			return
		}
		configEntry = fmt.Sprintf("[[%s]]\n  type = RNodeInterface\n  enabled = yes\n  port = \n  frequency = %d\n  bandwidth = %d\n  spreadingfactor = %d\n  codingrate = %d\n  txpower = %s%s",
			cfgName, intOf(info["frequency"]), intOf(info["bandwidth"]), intOf(info["sf"]), intOf(info["cr"]), cfgNetnameStr, cfgNetkeyStr)
	case "WeaveInterface":
		if _, ok := info["frequency"]; !ok {
			return
		}
		if _, ok := info["bandwidth"]; !ok {
			return
		}
		if _, ok := info["channel"]; !ok {
			return
		}
		if _, ok := info["modulation"]; !ok {
			return
		}
		configEntry = fmt.Sprintf("[[%s]]\n  type = WeaveInterface\n  enabled = yes\n  port = %s%s",
			cfgName, cfgNetnameStr, cfgNetkeyStr)
	case "KISSInterface":
		if _, ok := info["frequency"]; !ok {
			return
		}
		if _, ok := info["bandwidth"]; !ok {
			return
		}
		if _, ok := info["modulation"]; !ok {
			return
		}
		configEntry = fmt.Sprintf("[[%s]]\n  type = KISSInterface\n  enabled = yes\n  port = \n  # Frequency: %d\n  # Bandwidth: %d\n  # Modulation: %s%s%s%s",
			cfgName, intOf(info["frequency"]), intOf(info["bandwidth"]), func() string {
				switch x := info["modulation"].(type) {
				case string:
					return x
				case []byte:
					return string(x)
				default:
					return ""
				}
			}(), cfgIdentityStr, cfgNetnameStr, cfgNetkeyStr)
	}
	if configEntry != "" {
		info["config_entry"] = configEntry
	}
	info["discovery_hash"] = FullHash([]byte(info["transport_id"].(string) + info["name"].(string)))
	if h.callback != nil {
		h.callback(info)
	}
}

func NewInterfaceDiscovery(requiredValue int, callback func(map[string]any), discoverInterfaces bool) (*InterfaceDiscovery, error) {
	if Owner == nil {
		return nil, errors.New("transport owner/storage path is not initialised")
	}
	if strings.TrimSpace(Owner.StoragePath) == "" {
		return nil, errors.New("transport owner/storage path is not initialised")
	}
	d := &InterfaceDiscovery{
		requiredValue: requiredValue,
		callback:      callback,
		storagePath:   filepath.Join(Owner.StoragePath, "discovery", "interfaces"),
		monitorEvery:  discoveryMonitorInterval,
		detachAfter:   discoveryDetachThreshold,
	}
	if d.requiredValue <= 0 {
		d.requiredValue = discoveryStampDefaultValue
	}
	if err := os.MkdirAll(d.storagePath, 0o755); err != nil {
		return nil, err
	}
	if discoverInterfaces {
		d.handler = NewInterfaceAnnounceHandler(d.requiredValue, d.interfaceDiscovered)
		RegisterAnnounceHandler(d.handler)
		go d.connectDiscovered()
	}
	return d, nil
}

func (d *InterfaceDiscovery) interfaceDiscovered(info map[string]any) {
	defer func() {
		if r := recover(); r != nil {
			Logf(LogError, "Error processing discovered interface data: %v", r)
			TraceException(r)
		}
	}()
	if d == nil || len(info) == 0 {
		return
	}
	if _, ok := discoveryInterfaceTypes[strings.TrimSpace(func() string {
		switch x := info["type"].(type) {
		case string:
			return x
		case []byte:
			return string(x)
		default:
			return ""
		}
	}())]; !ok {
		return
	}
	hash, _ := info["discovery_hash"].([]byte)
	if len(hash) == 0 {
		return
	}
	path := filepath.Join(d.storagePath, hex.EncodeToString(hash))
	received := float64(time.Now().UnixNano()) / 1e9
	hops, _ := info["hops"].(int)
	stampValue, _ := info["value"].(int)
	ifName, _ := info["name"].(string)
	ifType, _ := info["type"].(string)
	ms := "s"
	if hops == 1 {
		ms = ""
	}
	Logf(LogDebug, "Discovered %s %d hop%s away with stamp value %d: %s", ifType, hops, ms, stampValue, ifName)
	switch x := info["received"].(type) {
	case float64:
		received = x
	case float32:
		received = float64(x)
	case int:
		received = float64(x)
	case int64:
		received = float64(x)
	}

	merged := map[string]any{}
	newlyDiscovered := true
	if raw, err := os.ReadFile(path); err == nil {
		newlyDiscovered = false
		_ = umsgpack.Unpackb(raw, &merged)
	}
	for k, v := range info {
		merged[k] = v
	}
	if _, ok := merged["discovered"]; !ok {
		merged["discovered"] = received
	}
	merged["last_heard"] = received
	if newlyDiscovered {
		merged["heard_count"] = 0
	} else {
		heardCount := 0
		if prev, ok := merged["heard_count"].(int); ok {
			heardCount = prev
		} else if prev, ok := merged["heard_count"].(int64); ok {
			heardCount = int(prev)
		}
		merged["heard_count"] = heardCount + 1
	}

	buf, err := umsgpack.Packb(merged)
	if err != nil {
		Logf(LogError, "Error while persisting discovered interface data: %v", err)
		TraceException(err)
		return
	}
	if err := os.WriteFile(path, buf, 0o600); err != nil {
		Logf(LogError, "Error while persisting discovered interface data: %v", err)
		TraceException(err)
	}

	d.autoconnect(merged)
	if d.callback != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					Logf(LogError, "Error while processing external interface discovery callback: %v", r)
				}
			}()
			d.callback(merged)
		}()
	}
}

func (d *InterfaceDiscovery) ListDiscoveredInterfaces(onlyAvailable, onlyTransport bool) []map[string]any {
	if d == nil || strings.TrimSpace(d.storagePath) == "" {
		return nil
	}
	entries, err := os.ReadDir(d.storagePath)
	if err != nil {
		return nil
	}

	now := float64(time.Now().UnixNano()) / 1e9
	floatOf := func(v any) float64 {
		switch x := v.(type) {
		case float64:
			return x
		case float32:
			return float64(x)
		case int:
			return float64(x)
		case int64:
			return float64(x)
		default:
			return 0
		}
	}
	intOf := func(v any) int {
		switch x := v.(type) {
		case int:
			return x
		case int64:
			return int(x)
		case uint64:
			return int(x)
		case float64:
			return int(x)
		default:
			return 0
		}
	}
	out := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(d.storagePath, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			Logf(LogError, "Error while loading discovered interface data: %v", err)
			Logf(LogError, "The interface data file %s may be corrupt", path)
			TraceException(err)
			continue
		}
		info := map[string]any{}
		if err := umsgpack.Unpackb(raw, &info); err != nil {
			Logf(LogError, "Error while loading discovered interface data: %v", err)
			Logf(LogError, "The interface data file %s may be corrupt", path)
			TraceException(err)
			continue
		}
		lastHeard := floatOf(info["last_heard"])
		if lastHeard <= 0 {
			continue
		}
		networkID, _ := info["network_id"].(string)
		networkID = strings.TrimSpace(networkID)
		if len(InterfaceDiscoverySources()) > 0 {
			if networkID == "" {
				_ = os.Remove(path)
				continue
			}
			decoded, err := hex.DecodeString(networkID)
			if err != nil {
				_ = os.Remove(path)
				continue
			}
			allowed := false
			for _, src := range InterfaceDiscoverySources() {
				if bytesEqual(src, decoded) {
					allowed = true
					break
				}
			}
			if !allowed {
				_ = os.Remove(path)
				continue
			}
		}
		if _, ok := discoveryInterfaceTypes[strings.TrimSpace(func() string {
			switch x := info["type"].(type) {
			case string:
				return x
			case []byte:
				return string(x)
			default:
				return ""
			}
		}())]; !ok {
			_ = os.Remove(path)
			continue
		}
		if reachableOnValue, ok := info["reachable_on"]; ok {
			var reachableOn string
			switch x := reachableOnValue.(type) {
			case string:
				reachableOn = x
			case []byte:
				reachableOn = string(x)
			default:
				reachableOn = fmt.Sprint(x)
			}
			if !(isIPAddress(reachableOn) || isHostname(reachableOn)) {
				_ = os.Remove(path)
				continue
			}
		}
		delta := time.Duration((now - lastHeard) * float64(time.Second))
		if delta > discoveryThresholdRemove {
			_ = os.Remove(path)
			continue
		}
		status := "available"
		statusCode := 1000
		switch {
		case delta > discoveryThresholdStale:
			status = "stale"
			statusCode = 0
		case delta > discoveryThresholdUnknown:
			status = "unknown"
			statusCode = 100
		}
		info["status"] = status
		info["status_code"] = statusCode

		if onlyAvailable && status != "available" {
			continue
		}
		if onlyTransport {
			if transport, ok := info["transport"].(bool); !ok || !transport {
				continue
			}
		}
		out = append(out, info)
	}

	sort.SliceStable(out, func(i, j int) bool {
		icode := intOf(out[i]["status_code"])
		jcode := intOf(out[j]["status_code"])
		if icode != jcode {
			return icode > jcode
		}
		iv := intOf(out[i]["value"])
		jv := intOf(out[j]["value"])
		if iv != jv {
			return iv > jv
		}
		return floatOf(out[i]["last_heard"]) > floatOf(out[j]["last_heard"])
	})
	return out
}

func (d *InterfaceDiscovery) connectDiscovered() {
	defer func() {
		if r := recover(); r != nil {
			Logf(LogError, "Error while reconnecting discovered interfaces: %v", r)
		}
	}()
	if d == nil || !ShouldAutoconnectDiscoveredInterfaces() {
		return
	}
	for _, info := range d.ListDiscoveredInterfaces(false, true) {
		if d.autoconnectCount() >= MaxAutoconnectedInterfaces() {
			break
		}
		d.autoconnect(info)
	}
	d.mu.Lock()
	d.initialAuto = true
	d.mu.Unlock()
}

func (d *InterfaceDiscovery) endpointHash(info map[string]any) []byte {
	spec := ""
	if info != nil {
		if reachableOn, ok := info["reachable_on"]; ok {
			spec += fmt.Sprint(reachableOn)
		}
		if _, ok := info["port"]; ok {
			spec += ":" + fmt.Sprint(info["port"])
		}
	}
	return FullHash([]byte(spec))
}

func (d *InterfaceDiscovery) interfaceExists(info map[string]any) bool {
	if info == nil {
		return false
	}
	endpointHash := d.endpointHash(info)
	reachableOn, hasReachableOn := info["reachable_on"]
	_, hasPort := info["port"]
	port := func(v any) int {
		switch x := v.(type) {
		case int:
			return x
		case int64:
			return int(x)
		case uint64:
			return int(x)
		case float64:
			return int(x)
		default:
			return 0
		}
	}(info["port"])
	reachableOnStr := fmt.Sprint(reachableOn)
	for _, ifc := range Interfaces {
		if ifc == nil {
			continue
		}
		if len(ifc.AutoconnectHash) > 0 && bytesEqual(ifc.AutoconnectHash, endpointHash) {
			return true
		}
		targetHost := ifc.DiscoveryTargetHost()
		targetPort := ifc.DiscoveryTargetPort()
		destMatch := hasReachableOn && targetHost != nil && *targetHost == reachableOnStr
		portMatch := !hasPort || (targetPort != nil && *targetPort == port)
		b32 := ifc.I2PB32()
		b32Match := hasReachableOn && b32 != nil && *b32 == reachableOnStr
		if (destMatch && portMatch) || b32Match {
			return true
		}
	}
	return false
}

func (d *InterfaceDiscovery) autoconnectCount() int {
	count := 0
	for _, ifc := range Interfaces {
		if ifc != nil && len(ifc.AutoconnectHash) > 0 {
			count++
		}
	}
	return count
}

func (d *InterfaceDiscovery) autoconnect(info map[string]any) {
	defer func() {
		if r := recover(); r != nil {
			Logf(LogError, "Error while auto-connecting discovered interface: %v", r)
			TraceException(r)
		}
	}()
	if d == nil || !ShouldAutoconnectDiscoveredInterfaces() {
		return
	}
	if d.autoconnectCount() >= MaxAutoconnectedInterfaces() {
		return
	}
	interfaceType, _ := info["type"].(string)
	if interfaceType != "BackboneInterface" && interfaceType != "TCPServerInterface" {
		return
	}
	if umsgpack.IsWindows() {
		Log("Your operating system does not support the Backbone interface type, and must degrade to using TCPClientInterface instead", LogWarning)
		Log("Auto-connecting discovered TCPClient interfaces is not yet implemented, aborting auto-connect", LogWarning)
		Log("You can obtain the configuration entry and add this interface manually instead using rnstatus -D", LogWarning)
		return
	}
	if d.interfaceExists(info) {
		Logf(LogDebug, "Discovered %s already exists, not auto-connecting", interfaceType)
		return
	}

	name, _ := info["name"].(string)
	reachableOn, _ := info["reachable_on"].(string)
	port := func(v any) int {
		switch x := v.(type) {
		case int:
			return x
		case int64:
			return int(x)
		case uint64:
			return int(x)
		case float64:
			return int(x)
		default:
			return 0
		}
	}(info["port"])
	if strings.TrimSpace(name) == "" || strings.TrimSpace(reachableOn) == "" || port <= 0 {
		Logf(LogWarning, "Could not auto-connect discovered %s %q: %v", interfaceType, name, errors.New("discovered interface missing reachable_on or port"))
		return
	}
	ifc, err := discoveryAutoconnectInterfaceFactory(info)
	if err != nil {
		Logf(LogWarning, "Could not auto-connect discovered %s %q: %v", interfaceType, name, err)
		return
	}
	endpointHash := d.endpointHash(info)
	ifc.AutoconnectHash = append([]byte(nil), endpointHash...)
	if networkID, _ := info["network_id"].(string); strings.TrimSpace(networkID) != "" {
		ifc.AutoconnectSource = networkID
	}
	ifc.OUT = true
	Logf(LogVerbose, "Auto-connecting discovered %s %s", interfaceType, name)

	var bitrate *int
	configuredBitrate := discoveryAutoconnectBitrate
	bitrate = &configuredBitrate
	var ifacNetname *string
	if v, _ := info["ifac_netname"].(string); strings.TrimSpace(v) != "" {
		ifacNetname = &v
	}
	var ifacNetkey *string
	if v, _ := info["ifac_netkey"].(string); strings.TrimSpace(v) != "" {
		ifacNetkey = &v
	}
	Owner.AddInterface(ifc, 0, bitrate, nil, ifacNetname, ifacNetkey, nil, nil, nil, nil)
	d.monitorInterface(ifc)
}

func (d *InterfaceDiscovery) monitorInterface(ifc *Interface) {
	if d == nil || ifc == nil {
		return
	}
	d.mu.Lock()
	found := false
	for _, existing := range d.monitored {
		if existing == ifc {
			found = true
			break
		}
	}
	if !found {
		d.monitored = append(d.monitored, ifc)
	}
	shouldStart := !d.monitoring
	if shouldStart {
		d.monitoring = true
	}
	d.mu.Unlock()

	if shouldStart {
		go d.monitorJob()
	}
}

func (d *InterfaceDiscovery) teardownInterface(ifc *Interface) {
	if ifc == nil {
		return
	}
	if handler := ifaces.RemoveInterfaceHandler; handler != nil {
		handler(ifc)
	} else {
		ifc.Detach()
	}
	if d == nil {
		return
	}
	d.mu.Lock()
	filtered := d.monitored[:0]
	for _, existing := range d.monitored {
		if existing != nil && existing != ifc {
			filtered = append(filtered, existing)
		}
	}
	d.monitored = filtered
	d.mu.Unlock()
}

func (d *InterfaceDiscovery) bootstrapInterfaceCount() int {
	count := 0
	for _, ifc := range Interfaces {
		if ifc != nil && ifc.BootstrapOnly {
			count++
		}
	}
	return count
}

func (d *InterfaceDiscovery) monitorJob() {
	for {
		d.mu.Lock()
		running := d.monitoring
		interval := d.monitorEvery
		d.mu.Unlock()
		if !running {
			return
		}
		if interval <= 0 {
			interval = discoveryMonitorInterval
		}
		time.Sleep(interval)
		if d == nil {
			return
		}
		now := time.Now()
		d.mu.Lock()
		monitored := append([]*Interface(nil), d.monitored...)
		initialAuto := d.initialAuto
		detachAfter := d.detachAfter
		d.mu.Unlock()
		if detachAfter <= 0 {
			detachAfter = discoveryDetachThreshold
		}

		detached := make([]*Interface, 0)
		online := 0
		for _, ifc := range monitored {
			if ifc == nil {
				continue
			}
			func() {
				defer func() {
					if r := recover(); r != nil {
						Logf(LogError, "Error while checking auto-connected interface state for %s: %v", ifc, r)
					}
				}()
				if ifc.Online {
					online++
					if !ifc.AutoconnectDown.IsZero() {
						Logf(LogVerbose, "Auto-discovered interface %s reconnected", ifc)
						ifc.AutoconnectDown = time.Time{}
					}
					return
				}
				if ifc.AutoconnectDown.IsZero() {
					Logf(LogDebug, "Auto-discovered interface %s disconnected", ifc)
					ifc.AutoconnectDown = now
					return
				}
				if downFor := now.Sub(ifc.AutoconnectDown); downFor >= detachAfter {
					Logf(LogDebug, "Auto-discovered interface %s has been down for %s, detaching", ifc, PrettyTime(downFor.Seconds(), false, false))
					detached = append(detached, ifc)
				}
			}()
		}

		maxAuto := MaxAutoconnectedInterfaces()
		freeSlots := maxAuto - d.autoconnectCount()
		if freeSlots < 0 {
			freeSlots = 0
		}
		reservedSlots := maxAuto / 4

		if online >= maxAuto {
			for _, ifc := range Interfaces {
				if ifc == nil || !ifc.BootstrapOnly {
					continue
				}
				Logf(LogInfo, "Tearing down bootstrap-only %s since target connected auto-discovered interface count has been reached", ifc)
				alreadyDetached := false
				for _, gone := range detached {
					if gone == ifc {
						alreadyDetached = true
						break
					}
				}
				if !alreadyDetached {
					detached = append(detached, ifc)
				}
			}
		}

		if online == 0 && Owner != nil && d.bootstrapInterfaceCount() == 0 {
			Log("No auto-discovered interfaces connected, re-enabling bootstrap interfaces", LogNotice)
			Owner.reenableBootstrapInterfaces()
		}

		if initialAuto && freeSlots > reservedSlots {
			candidates := d.ListDiscoveredInterfaces(true, true)
			if len(candidates) > 0 {
				selected := candidates[rand.Intn(len(candidates))]
				if !d.interfaceExists(selected) {
					d.autoconnect(selected)
				}
			}
		}

		for _, ifc := range detached {
			func() {
				defer func() {
					if r := recover(); r != nil {
						Logf(LogError, "Error while de-registering auto-connected interface from transport: %v", r)
					}
				}()
				d.teardownInterface(ifc)
			}()
		}

		d.mu.Lock()
		filtered := d.monitored[:0]
		for _, ifc := range d.monitored {
			if ifc == nil {
				continue
			}
			keep := true
			for _, gone := range detached {
				if gone == ifc {
					keep = false
					break
				}
			}
			if keep {
				filtered = append(filtered, ifc)
			}
		}
		d.monitored = filtered
		d.mu.Unlock()
	}
}

func NewBlackholeUpdater() *BlackholeUpdater {
	return &BlackholeUpdater{
		lastUpdates:    make(map[hashKey]time.Time),
		initialWait:    blackholeUpdaterInitialWait,
		jobInterval:    blackholeUpdaterJobInterval,
		updateInterval: blackholeUpdateInterval,
		sourceTimeout:  blackholeSourceTimeout,
		awaitPath:      AwaitPath,
		sleep:          time.Sleep,
	}
}

func (u *BlackholeUpdater) Start() {
	if u == nil {
		return
	}
	u.mu.Lock()
	if u.shouldRun {
		u.mu.Unlock()
		return
	}
	sourceCount := len(BlackholeSources())
	ms := ""
	if sourceCount != 1 {
		ms = "s"
	}
	Logf(LogDebug, "Starting blackhole updater with %d source%s", sourceCount, ms)
	u.shouldRun = true
	u.mu.Unlock()
	go u.job()
}

func (u *BlackholeUpdater) Stop() {
	if u == nil {
		return
	}
	u.mu.Lock()
	u.shouldRun = false
	u.mu.Unlock()
}

func (u *BlackholeUpdater) job() {
	if u == nil {
		return
	}
	if u.initialWait > 0 {
		u.sleep(u.initialWait)
	}
	for {
		u.mu.Lock()
		shouldRun := u.shouldRun
		u.mu.Unlock()
		if !shouldRun {
			return
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					Logf(LogError, "Error in blackhole list updater job: %v", r)
					TraceException(r)
				}
			}()
			now := time.Now()
			for _, sourceHash := range BlackholeSources() {
				if len(sourceHash) < truncatedHashBytes {
					continue
				}
				var sourceKey hashKey
				copy(sourceKey[:], sourceHash[:truncatedHashBytes])
				u.mu.Lock()
				last := u.lastUpdates[sourceKey]
				interval := u.updateInterval
				u.mu.Unlock()
				if interval <= 0 {
					interval = blackholeUpdateInterval
				}
				if now.Sub(last) < interval {
					continue
				}
				destinationHash, err := (&Destination{}).HashFromNameAndIdentity("rnstransport.info.blackhole", &Identity{Hash: sourceHash})
				if err != nil || len(destinationHash) == 0 {
					continue
				}
				Logf(LogDebug, "Attempting blackhole list update from %s...", PrettyHexRep(sourceHash))
				if u.awaitPath != nil && !u.awaitPath(destinationHash, 0, nil) {
					Logf(LogVerbose, "No path available for blackhole list update from %s, retrying later", PrettyHexRep(sourceHash))
					continue
				}
				timeout := u.sourceTimeout
				if timeout <= 0 {
					timeout = blackholeSourceTimeout
				}
				var response any
				if u.fetchList != nil {
					response, err = u.fetchList(sourceHash, timeout)
				} else {
					remoteIdentity := IdentityRecall(destinationHash)
					if remoteIdentity == nil {
						remoteIdentity = IdentityRecall(sourceHash, true)
					}
					if remoteIdentity == nil {
						err = errors.New("remote blackhole source identity not known")
					} else {
						destination, derr := NewDestination(remoteIdentity, DestinationOUT, DestinationSINGLE, TransportAppName, "info", "blackhole")
						if derr != nil {
							err = derr
						} else {
							establishedCh := make(chan *Link, 1)
							var established *Link
							link, derr := NewLink(destination, nil, LinkModeDefault, func(l *Link) {
								select {
								case establishedCh <- l:
								default:
								}
							}, nil)
							if derr != nil {
								err = derr
							} else {
								select {
								case established = <-establishedCh:
								case <-time.After(timeout):
									if link != nil {
										link.Teardown()
									}
									err = errors.New("timed out establishing blackhole update link")
								}
								if err == nil {
									if established == nil {
										err = errors.New("blackhole update link was not established")
									} else {
										defer established.Teardown()
										result := established.Request("/list", nil, nil, nil, nil, timeout.Seconds())
										rr, _ := result.(*RequestReceipt)
										if rr == nil {
											err = errors.New("blackhole list request could not be started")
										} else {
											deadline := time.Now().Add(timeout)
											for !rr.Concluded() && time.Now().Before(deadline) {
												time.Sleep(200 * time.Millisecond)
											}
											if rr.Status() != ReceiptReady {
												err = errors.New("blackhole list request timed out or failed")
											} else {
												response = rr.Response()
											}
										}
									}
								}
							}
						}
					}
				}
				if err != nil {
					Logf(LogError, "Error while establishing link for blackhole list update from %s: %v", PrettyHexRep(sourceHash), err)
					continue
				}
				decoded := make(map[hashKey]*blackholeEntry)
				switch typed := response.(type) {
				case map[hashKey]map[string]any:
					for entryKey, entryValue := range typed {
						if entryValue == nil {
							continue
						}
						entry := &blackholeEntry{Source: append([]byte(nil), sourceHash...)}
						if len(entry.Source) == 0 {
							switch source := entryValue["source"].(type) {
							case []byte:
								if len(source) > 0 {
									entry.Source = source
								}
							case string:
								if len(source) > 0 {
									entry.Source = []byte(source)
								}
							}
						}
						if untilVal, exists := entryValue["until"]; exists && untilVal != nil {
							untilUnix := asFloat64(untilVal)
							if untilUnix > 0 {
								sec, frac := math.Modf(untilUnix)
								until := time.Unix(int64(sec), int64(frac*1e9))
								if sec < 0 {
									until = time.Unix(0, 0)
								}
								if now.After(until) {
									continue
								}
								entry.Until = &until
							}
						}
						if reasonVal, exists := entryValue["reason"]; exists && reasonVal != nil {
							switch reason := reasonVal.(type) {
							case string:
								entry.Reason = &reason
							case []byte:
								if len(reason) > 0 {
									reasonStr := string(reason)
									entry.Reason = &reasonStr
								}
							}
						}
						decoded[entryKey] = entry
					}
				case map[hashKey]any:
					for entryKey, entryValue := range typed {
						raw, ok := entryValue.(map[any]any)
						if !ok {
							if typedRaw, ok := entryValue.(map[string]any); ok {
								raw = make(map[any]any, len(typedRaw))
								for k, v := range typedRaw {
									raw[k] = v
								}
							} else {
								continue
							}
						}
						entry := &blackholeEntry{Source: append([]byte(nil), sourceHash...)}
						if len(entry.Source) == 0 {
							switch source := raw["source"].(type) {
							case []byte:
								if len(source) > 0 {
									entry.Source = source
								}
							case string:
								if len(source) > 0 {
									entry.Source = []byte(source)
								}
							}
						}
						if untilVal, exists := raw["until"]; exists && untilVal != nil {
							untilUnix := asFloat64(untilVal)
							if untilUnix > 0 {
								sec, frac := math.Modf(untilUnix)
								until := time.Unix(int64(sec), int64(frac*1e9))
								if sec < 0 {
									until = time.Unix(0, 0)
								}
								if now.After(until) {
									continue
								}
								entry.Until = &until
							}
						}
						if reasonVal, exists := raw["reason"]; exists && reasonVal != nil {
							switch reason := reasonVal.(type) {
							case string:
								entry.Reason = &reason
							case []byte:
								if len(reason) > 0 {
									reasonStr := string(reason)
									entry.Reason = &reasonStr
								}
							}
						}
						decoded[entryKey] = entry
					}
				case map[any]any:
					for rawKey, rawValue := range typed {
						var entryKey hashKey
						var ok bool
						switch v := rawKey.(type) {
						case string:
							entryKey, ok = func(hash []byte) (hashKey, bool) {
								if len(hash) < truncatedHashBytes {
									return hashKey{}, false
								}
								var key hashKey
								copy(key[:], hash[:truncatedHashBytes])
								return key, true
							}([]byte(v))
						case umsgpack.BinaryKey:
							entryKey, ok = func(hash []byte) (hashKey, bool) {
								if len(hash) < truncatedHashBytes {
									return hashKey{}, false
								}
								var key hashKey
								copy(key[:], hash[:truncatedHashBytes])
								return key, true
							}([]byte(string(v)))
						case []byte:
							entryKey, ok = func(hash []byte) (hashKey, bool) {
								if len(hash) < truncatedHashBytes {
									return hashKey{}, false
								}
								var key hashKey
								copy(key[:], hash[:truncatedHashBytes])
								return key, true
							}(v)
						default:
							ok = false
						}
						if !ok {
							continue
						}
						raw, ok := rawValue.(map[any]any)
						if !ok {
							if typedRaw, ok := rawValue.(map[string]any); ok {
								raw = make(map[any]any, len(typedRaw))
								for k, v := range typedRaw {
									raw[k] = v
								}
							} else {
								continue
							}
						}
						entry := &blackholeEntry{Source: append([]byte(nil), sourceHash...)}
						if len(entry.Source) == 0 {
							switch source := raw["source"].(type) {
							case []byte:
								if len(source) > 0 {
									entry.Source = source
								}
							case string:
								if len(source) > 0 {
									entry.Source = []byte(source)
								}
							}
						}
						if untilVal, exists := raw["until"]; exists && untilVal != nil {
							untilUnix := asFloat64(untilVal)
							if untilUnix > 0 {
								sec, frac := math.Modf(untilUnix)
								until := time.Unix(int64(sec), int64(frac*1e9))
								if sec < 0 {
									until = time.Unix(0, 0)
								}
								if now.After(until) {
									continue
								}
								entry.Until = &until
							}
						}
						if reasonVal, exists := raw["reason"]; exists && reasonVal != nil {
							switch reason := reasonVal.(type) {
							case string:
								entry.Reason = &reason
							case []byte:
								if len(reason) > 0 {
									reasonStr := string(reason)
									entry.Reason = &reasonStr
								}
							}
						}
						decoded[entryKey] = entry
					}
				case map[string]any:
					for rawKey, rawValue := range typed {
						entryKey, ok := func(hash []byte) (hashKey, bool) {
							if len(hash) < truncatedHashBytes {
								return hashKey{}, false
							}
							var key hashKey
							copy(key[:], hash[:truncatedHashBytes])
							return key, true
						}([]byte(rawKey))
						if !ok {
							continue
						}
						raw, ok := rawValue.(map[any]any)
						if !ok {
							if typedRaw, ok := rawValue.(map[string]any); ok {
								raw = make(map[any]any, len(typedRaw))
								for k, v := range typedRaw {
									raw[k] = v
								}
							} else {
								continue
							}
						}
						entry := &blackholeEntry{Source: append([]byte(nil), sourceHash...)}
						if len(entry.Source) == 0 {
							switch source := raw["source"].(type) {
							case []byte:
								if len(source) > 0 {
									entry.Source = source
								}
							case string:
								if len(source) > 0 {
									entry.Source = []byte(source)
								}
							}
						}
						if untilVal, exists := raw["until"]; exists && untilVal != nil {
							untilUnix := asFloat64(untilVal)
							if untilUnix > 0 {
								sec, frac := math.Modf(untilUnix)
								until := time.Unix(int64(sec), int64(frac*1e9))
								if sec < 0 {
									until = time.Unix(0, 0)
								}
								if now.After(until) {
									continue
								}
								entry.Until = &until
							}
						}
						if reasonVal, exists := raw["reason"]; exists && reasonVal != nil {
							switch reason := reasonVal.(type) {
							case string:
								entry.Reason = &reason
							case []byte:
								if len(reason) > 0 {
									reasonStr := string(reason)
									entry.Reason = &reasonStr
								}
							}
						}
						decoded[entryKey] = entry
					}
				}
				if len(decoded) == 0 {
					continue
				}
				added := 0
				serialisable := make(map[hashKey]map[string]any, len(decoded))
				blackholeMu.Lock()
				for entryKey, entry := range decoded {
					if entry == nil {
						continue
					}
					serialised := map[string]any{"source": nil, "until": nil, "reason": nil}
					serialised["source"] = append([]byte(nil), entry.Source...)
					if entry.Until != nil && !entry.Until.IsZero() {
						serialised["until"] = float64(entry.Until.UnixNano()) / 1e9
					}
					if entry.Reason != nil {
						serialised["reason"] = *entry.Reason
					}
					serialisable[entryKey] = serialised
					if _, exists := blackholedIdentities[entryKey]; exists {
						continue
					}
					blackholedIdentities[entryKey] = entry
					added++
				}
				blackholeMu.Unlock()

				if added > 0 {
					removeBlackholedPaths()
					if Owner != nil && strings.TrimSpace(Owner.StoragePath) != "" {
						sourcelistpath := filepath.Join(Owner.StoragePath, "blackhole", hex.EncodeToString(sourceHash))
						tmppath := sourcelistpath + ".tmp"
						payload := make(map[any]any, len(serialisable))
						for entryKey, entry := range serialisable {
							payload[umsgpack.BinaryKey(string(entryKey[:]))] = entry
						}
						packed, err := umsgpack.Packb(payload)
						if err != nil {
							Logf(LogError, "Error while persisting blackhole list from %s: %v", PrettyHexRep(sourceHash), err)
						} else {
							if err := os.WriteFile(tmppath, packed, 0o600); err != nil {
								Logf(LogError, "Error while persisting blackhole list from %s: %v", PrettyHexRep(sourceHash), err)
							} else {
								if fileExists(sourcelistpath) {
									_ = os.Remove(sourcelistpath)
								}
								if err := os.Rename(tmppath, sourcelistpath); err != nil {
									Logf(LogError, "Error while persisting blackhole list from %s: %v", PrettyHexRep(sourceHash), err)
								}
							}
						}
					}
				}
				spec := "identities"
				if added == 1 {
					spec = "identity"
				}
				Logf(LogDebug, "Added %d blackholed %s from %s", added, spec, PrettyHexRep(sourceHash))
				u.mu.Lock()
				if u.lastUpdates == nil {
					u.lastUpdates = make(map[hashKey]time.Time)
				}
				u.lastUpdates[sourceKey] = now
				u.mu.Unlock()
				Logf(LogDebug, "Blackhole list update from %s completed", PrettyHexRep(sourceHash))
			}
		}()
		interval := u.jobInterval
		if interval <= 0 {
			interval = blackholeUpdaterJobInterval
		}
		u.sleep(interval)
	}
}

func sanitizeDiscoveryString(in string) string {
	s := strings.ReplaceAll(in, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	return strings.TrimSpace(s)
}

func isIPAddress(v string) bool {
	if ip := net.ParseIP(v); ip != nil {
		return true
	}
	return false
}

func isHostname(v string) bool {
	v = strings.TrimSuffix(v, ".")
	if len(v) == 0 || len(v) > 253 {
		return false
	}
	parts := strings.Split(v, ".")
	if len(parts) == 0 {
		return false
	}
	if _, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
		return false
	}
	for _, part := range parts {
		if len(part) == 0 || len(part) > 63 {
			return false
		}
		if part[0] == '-' || part[len(part)-1] == '-' {
			return false
		}
		for _, r := range part {
			isLetter := r >= 'a' && r <= 'z'
			isUpper := r >= 'A' && r <= 'Z'
			isDigit := r >= '0' && r <= '9'
			if !isLetter && !isUpper && !isDigit && r != '-' {
				return false
			}
		}
		if part == "" {
			return false
		}
	}
	return true
}
