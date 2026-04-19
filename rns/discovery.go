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

var discoveryAllowedInterfaceTypes = map[string]struct{}{
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
var discoveryAutoconnectInterfaceFactory = defaultDiscoveryAutoconnectInterfaceFactory
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
	if !requireDiscoveryStampProvider() {
		return &InterfaceAnnouncer{stampCache: make(map[string][]byte)}
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

func (a *InterfaceAnnouncer) running() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.shouldRun
}

func (a *InterfaceAnnouncer) job() {
	for a.running() {
		time.Sleep(discoveryAnnouncerInterval)
		if !a.running() {
			return
		}
		a.runOnce(time.Now())
	}
}

func (a *InterfaceAnnouncer) runOnce(now time.Time) {
	if a == nil || a.dest == nil {
		return
	}
	due := make([]*Interface, 0)
	for _, ifc := range Interfaces {
		if ifc == nil || !ifc.SupportsDiscovery() || !ifc.Discoverable {
			continue
		}
		interval := ifc.DiscoveryAnnounceInterval
		if interval <= 0 {
			interval = 6 * time.Hour
		}
		if ifc.LastDiscoveryAnnounce.IsZero() || now.Sub(ifc.LastDiscoveryAnnounce) > interval {
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
	appData, err := a.GetInterfaceAnnounceData(selected)
	if err != nil {
		Logf(LogError, "Could not generate interface discovery announce data for %s: %v", selected.Name, err)
		return
	}
	selected.LastDiscoveryAnnounce = now
	a.dest.Announce(appData, false, nil, nil, true)
}

func (a *InterfaceAnnouncer) GetInterfaceAnnounceData(ifc *Interface) ([]byte, error) {
	if ifc == nil {
		return nil, errors.New("nil interface")
	}
	if DiscoveryStampProvider == nil {
		return nil, errors.New("discovery stamper provider is not configured")
	}

	ifType := ifc.Type
	if !ifc.SupportsDiscovery() {
		return nil, fmt.Errorf("interface type %q does not support discovery", ifType)
	}
	if ifType == "TCPClientInterface" && !ifc.DiscoveryKISSFraming() {
		return nil, fmt.Errorf("invalid interface discovery configuration for %s", ifc.Name)
	}

	flags := byte(0)
	info := map[any]any{
		discoveryFieldInterfaceType: ifType,
		discoveryFieldTransport:     TransportEnabled(),
		discoveryFieldTransportID:   copyBytes(TransportIdentity.Hash),
		discoveryFieldName:          sanitizeDiscoveryString(ifc.DiscoveryNameValue()),
		discoveryFieldLatitude:      derefFloat64(ifc.DiscoveryLatitude),
		discoveryFieldLongitude:     derefFloat64(ifc.DiscoveryLongitude),
		discoveryFieldHeight:        derefFloat64(ifc.DiscoveryHeight),
	}

	switch ifType {
	case "BackboneInterface", "TCPServerInterface":
		reachableOn, err := resolveDiscoveryReachableOn(ifc, sanitizeDiscoveryString(ifc.DiscoveryReachableOnValue()))
		if err != nil {
			return nil, err
		}
		if !isDiscoveryAddress(reachableOn) {
			return nil, fmt.Errorf("invalid reachable_on %q for %s", reachableOn, ifc.Name)
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
	case "RNodeInterface", "RNodeMultiInterface":
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

	if ifc.DiscoveryKISSFraming() {
		info[discoveryFieldInterfaceType] = "KISSInterface"
		if ifc.DiscoveryFrequency != nil {
			info[discoveryFieldFrequency] = *ifc.DiscoveryFrequency
		}
		if ifc.DiscoveryBandwidth != nil {
			info[discoveryFieldBandwidth] = *ifc.DiscoveryBandwidth
		}
		if strings.TrimSpace(ifc.DiscoveryModulation) != "" {
			info[discoveryFieldModulation] = sanitizeDiscoveryString(ifc.DiscoveryModulation)
		}
	}

	if ifc.DiscoveryPublishIFAC {
		if netname := sanitizeDiscoveryString(ifc.IFACNetname()); netname != "" {
			info[discoveryFieldIFACNetname] = netname
		}
		if netkey := sanitizeDiscoveryString(ifc.IFACNetkey()); netkey != "" {
			info[discoveryFieldIFACNetkey] = netkey
		}
	}

	packed, err := packDiscoveryInfo(info)
	if err != nil {
		return nil, err
	}
	infoHash := FullHash(packed)
	stampCost := discoveryStampDefaultValue
	if ifc.DiscoveryStampValue != nil && *ifc.DiscoveryStampValue > 0 {
		stampCost = *ifc.DiscoveryStampValue
	}
	stampKey := string(packed)
	var stamp []byte
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
	payload := append(append([]byte(nil), packed...), stamp...)
	if ifc.DiscoveryEncrypt {
		if !HasNetworkIdentity() {
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
	if !requireDiscoveryStampProvider() {
		return &InterfaceAnnounceHandler{requiredValue: requiredValue, callback: callback}
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
	if h == nil || announcedIdentity == nil || len(appData) <= 1 || DiscoveryStampProvider == nil {
		return
	}
	if !discoverySourceAllowed(announcedIdentity.Hash) {
		Logf(LogDebug, "Interface discovered from non-authorized network identity %s, ignoring", PrettyHexRep(announcedIdentity.Hash))
		return
	}

	flags := appData[0]
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
		return
	}

	raw := map[any]any{}
	if err := umsgpack.Unpackb(packed, &raw); err != nil {
		return
	}
	info := discoveryInfoFromRaw(raw, destinationHash, announcedIdentity, stamp, DiscoveryStampProvider.StampValue(workblock, stamp))
	if info == nil {
		return
	}
	if h.callback != nil {
		h.callback(info)
	}
}

func NewInterfaceDiscovery(requiredValue int, callback func(map[string]any), discoverInterfaces bool) (*InterfaceDiscovery, error) {
	if Owner == nil {
		return nil, errors.New("transport owner/storage path is not initialised")
	}
	return newInterfaceDiscoveryWithStorage(Owner.StoragePath, requiredValue, callback, discoverInterfaces)
}

func newInterfaceDiscoveryWithStorage(storageBase string, requiredValue int, callback func(map[string]any), discoverInterfaces bool) (*InterfaceDiscovery, error) {
	if strings.TrimSpace(storageBase) == "" {
		return nil, errors.New("transport owner/storage path is not initialised")
	}
	d := &InterfaceDiscovery{
		requiredValue: requiredValue,
		callback:      callback,
		storagePath:   filepath.Join(storageBase, "discovery", "interfaces"),
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
	if d == nil || len(info) == 0 {
		return
	}
	if !discoveryInterfaceTypeAllowed(asDiscoveryString(info["type"])) {
		return
	}
	hash, _ := info["discovery_hash"].([]byte)
	if len(hash) == 0 {
		return
	}
	path := filepath.Join(d.storagePath, hex.EncodeToString(hash))
	now := time.Now().Unix()

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
		merged["discovered"] = now
	}
	merged["last_heard"] = now
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
		return
	}
	_ = os.WriteFile(path, buf, 0o600)

	d.autoconnect(merged)
	if d.callback != nil {
		d.callback(merged)
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

	now := time.Now()
	out := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(d.storagePath, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		info := map[string]any{}
		if err := umsgpack.Unpackb(raw, &info); err != nil {
			continue
		}
		lastHeardUnix, _ := info["last_heard"].(int64)
		if lastHeardUnix == 0 {
			if v, ok := info["last_heard"].(int); ok {
				lastHeardUnix = int64(v)
			}
		}
		if lastHeardUnix == 0 {
			continue
		}
		networkID, _ := info["network_id"].(string)
		if !discoveryAllowedSource(networkID) {
			_ = os.Remove(path)
			continue
		}
		if !discoveryInterfaceTypeAllowed(asDiscoveryString(info["type"])) {
			_ = os.Remove(path)
			continue
		}
		if reachableOn, _ := info["reachable_on"].(string); strings.TrimSpace(reachableOn) != "" && !isDiscoveryAddress(strings.TrimSpace(reachableOn)) {
			_ = os.Remove(path)
			continue
		}
		delta := now.Sub(time.Unix(lastHeardUnix, 0))
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
		icode := asDiscoveryInt(out[i]["status_code"])
		jcode := asDiscoveryInt(out[j]["status_code"])
		if icode != jcode {
			return icode > jcode
		}
		iv := asDiscoveryInt(out[i]["value"])
		jv := asDiscoveryInt(out[j]["value"])
		if iv != jv {
			return iv > jv
		}
		return asDiscoveryInt64(out[i]["last_heard"]) > asDiscoveryInt64(out[j]["last_heard"])
	})
	return out
}

func discoveryInfoFromRaw(raw map[any]any, destinationHash []byte, announcedIdentity *Identity, stamp []byte, value int) map[string]any {
	if raw == nil || announcedIdentity == nil {
		return nil
	}
	ifType := sanitizeDiscoveryString(asDiscoveryString(discoveryRawGet(raw, discoveryFieldInterfaceType)))
	if ifType == "" {
		return nil
	}
	if !discoveryInterfaceTypeAllowed(ifType) {
		return nil
	}
	name := sanitizeDiscoveryString(asDiscoveryString(discoveryRawGet(raw, discoveryFieldName)))
	if name == "" {
		name = "Discovered " + ifType
	}
	transportID := asDiscoveryBytes(discoveryRawGet(raw, discoveryFieldTransportID))
	info := map[string]any{
		"type":         ifType,
		"transport":    asDiscoveryBool(discoveryRawGet(raw, discoveryFieldTransport)),
		"name":         name,
		"received":     time.Now().Unix(),
		"stamp":        stamp,
		"value":        value,
		"transport_id": strings.ToLower(hex.EncodeToString(transportID)),
		"network_id":   strings.ToLower(hex.EncodeToString(announcedIdentity.Hash)),
		"hops":         discoveryHopsTo(destinationHash),
	}
	if latRaw := discoveryRawGet(raw, discoveryFieldLatitude); latRaw != nil {
		info["latitude"] = asDiscoveryFloat(latRaw)
	} else {
		info["latitude"] = nil
	}
	if lonRaw := discoveryRawGet(raw, discoveryFieldLongitude); lonRaw != nil {
		info["longitude"] = asDiscoveryFloat(lonRaw)
	} else {
		info["longitude"] = nil
	}
	if heightRaw := discoveryRawGet(raw, discoveryFieldHeight); heightRaw != nil {
		info["height"] = asDiscoveryFloat(heightRaw)
	} else {
		info["height"] = nil
	}
	if reachableOn := sanitizeDiscoveryString(asDiscoveryString(discoveryRawGet(raw, discoveryFieldReachableOn))); reachableOn != "" {
		info["reachable_on"] = reachableOn
	}
	if port := asDiscoveryMaybeInt(discoveryRawGet(raw, discoveryFieldPort)); port != nil {
		info["port"] = *port
	}
	if netname := sanitizeDiscoveryString(asDiscoveryString(discoveryRawGet(raw, discoveryFieldIFACNetname))); netname != "" {
		info["ifac_netname"] = netname
	}
	if netkey := sanitizeDiscoveryString(asDiscoveryString(discoveryRawGet(raw, discoveryFieldIFACNetkey))); netkey != "" {
		info["ifac_netkey"] = netkey
	}
	if v := asDiscoveryMaybeInt(discoveryRawGet(raw, discoveryFieldFrequency)); v != nil {
		info["frequency"] = *v
	}
	if v := asDiscoveryMaybeInt(discoveryRawGet(raw, discoveryFieldBandwidth)); v != nil {
		info["bandwidth"] = *v
	}
	if v := asDiscoveryMaybeInt(discoveryRawGet(raw, discoveryFieldSpreading)); v != nil {
		info["sf"] = *v
	}
	if v := asDiscoveryMaybeInt(discoveryRawGet(raw, discoveryFieldCodingRate)); v != nil {
		info["cr"] = *v
	}
	if v := asDiscoveryMaybeInt(discoveryRawGet(raw, discoveryFieldChannel)); v != nil {
		info["channel"] = *v
	}
	if modulation := sanitizeDiscoveryString(asDiscoveryString(discoveryRawGet(raw, discoveryFieldModulation))); modulation != "" {
		info["modulation"] = modulation
	}
	if configEntry := discoveryConfigEntry(info); configEntry != "" {
		info["config_entry"] = configEntry
	}
	info["discovery_hash"] = FullHash([]byte(info["transport_id"].(string) + info["name"].(string)))
	return info
}

func discoveryInterfaceTypeAllowed(ifType string) bool {
	_, ok := discoveryAllowedInterfaceTypes[strings.TrimSpace(ifType)]
	return ok
}

func discoveryAllowedSource(networkID string) bool {
	sources := InterfaceDiscoverySources()
	if len(sources) == 0 {
		return true
	}
	networkID = strings.TrimSpace(networkID)
	if networkID == "" {
		return false
	}
	decoded, err := hex.DecodeString(networkID)
	if err != nil {
		return false
	}
	for _, src := range sources {
		if bytesEqual(src, decoded) {
			return true
		}
	}
	return false
}

func discoverySourceAllowed(identityHash []byte) bool {
	sources := InterfaceDiscoverySources()
	if len(sources) == 0 {
		return true
	}
	for _, src := range sources {
		if bytesEqual(src, identityHash) {
			return true
		}
	}
	return false
}

func discoveryConfigEntry(info map[string]any) string {
	if len(info) == 0 {
		return ""
	}
	interfaceType, _ := info["type"].(string)
	name, _ := info["name"].(string)
	transportID, _ := info["transport_id"].(string)
	ifacNetname, _ := info["ifac_netname"].(string)
	ifacNetkey, _ := info["ifac_netkey"].(string)
	cfgIdentityStr := ""
	if strings.TrimSpace(transportID) != "" {
		cfgIdentityStr = "\n  transport_identity = " + transportID
	}
	cfgNetnameStr := ""
	if strings.TrimSpace(ifacNetname) != "" {
		cfgNetnameStr = "\n  network_name = " + ifacNetname
	}
	cfgNetkeyStr := ""
	if strings.TrimSpace(ifacNetkey) != "" {
		cfgNetkeyStr = "\n  passphrase = " + ifacNetkey
	}

	switch interfaceType {
	case "BackboneInterface", "TCPServerInterface":
		reachableOn, _ := info["reachable_on"].(string)
		port := asDiscoveryInt(info["port"])
		if strings.TrimSpace(reachableOn) == "" || port <= 0 {
			return ""
		}
		connectionType := "BackboneInterface"
		remoteKey := "remote"
		if umsgpack.IsWindows() {
			connectionType = "TCPClientInterface"
			remoteKey = "target_host"
		}
		return fmt.Sprintf("[[%s]]\n  type = %s\n  enabled = yes\n  %s = %s\n  target_port = %d%s%s%s",
			name, connectionType, remoteKey, reachableOn, port, cfgIdentityStr, cfgNetnameStr, cfgNetkeyStr)
	case "I2PInterface":
		reachableOn, _ := info["reachable_on"].(string)
		if strings.TrimSpace(reachableOn) == "" {
			return ""
		}
		return fmt.Sprintf("[[%s]]\n  type = I2PInterface\n  enabled = yes\n  peers = %s%s%s%s",
			name, reachableOn, cfgIdentityStr, cfgNetnameStr, cfgNetkeyStr)
	case "RNodeInterface":
		frequency := asDiscoveryInt(info["frequency"])
		bandwidth := asDiscoveryInt(info["bandwidth"])
		sf := asDiscoveryInt(info["sf"])
		cr := asDiscoveryInt(info["cr"])
		return fmt.Sprintf("[[%s]]\n  type = RNodeInterface\n  enabled = yes\n  port = \n  frequency = %d\n  bandwidth = %d\n  spreadingfactor = %d\n  codingrate = %d\n  txpower = %s%s%s",
			name, frequency, bandwidth, sf, cr, cfgIdentityStr, cfgNetnameStr, cfgNetkeyStr)
	case "WeaveInterface":
		return fmt.Sprintf("[[%s]]\n  type = WeaveInterface\n  enabled = yes\n  port = %s%s%s",
			name, cfgIdentityStr, cfgNetnameStr, cfgNetkeyStr)
	case "KISSInterface":
		frequency := asDiscoveryInt(info["frequency"])
		bandwidth := asDiscoveryInt(info["bandwidth"])
		modulation, _ := info["modulation"].(string)
		return fmt.Sprintf("[[%s]]\n  type = KISSInterface\n  enabled = yes\n  port = \n  # Frequency: %d\n  # Bandwidth: %d\n  # Modulation: %s%s%s%s",
			name, frequency, bandwidth, modulation, cfgIdentityStr, cfgNetnameStr, cfgNetkeyStr)
	default:
		return ""
	}
}

func (d *InterfaceDiscovery) connectDiscovered() {
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
	if len(info) == 0 {
		return nil
	}
	spec := ""
	if reachableOn, _ := info["reachable_on"].(string); strings.TrimSpace(reachableOn) != "" {
		spec += reachableOn
	}
	if port := asDiscoveryInt(info["port"]); port > 0 {
		spec += ":" + strconv.Itoa(port)
	}
	if spec == "" {
		return nil
	}
	return FullHash([]byte(spec))
}

func (d *InterfaceDiscovery) interfaceExists(info map[string]any) bool {
	endpointHash := d.endpointHash(info)
	reachableOn, _ := info["reachable_on"].(string)
	port := asDiscoveryInt(info["port"])
	for _, ifc := range Interfaces {
		if ifc == nil {
			continue
		}
		if len(endpointHash) > 0 && len(ifc.AutoconnectHash) > 0 && bytesEqual(ifc.AutoconnectHash, endpointHash) {
			return true
		}
		targetHost := ifc.DiscoveryTargetHost()
		if targetHost == nil || strings.TrimSpace(*targetHost) == "" || strings.TrimSpace(reachableOn) == "" {
			continue
		}
		if *targetHost != reachableOn {
			continue
		}
		targetPort := ifc.DiscoveryTargetPort()
		if port <= 0 || targetPort == nil || *targetPort == port {
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
	if d == nil || !ShouldAutoconnectDiscoveredInterfaces() || Owner == nil || Owner.IsConnectedToSharedInstance {
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
		return
	}

	ifc, err := discoveryAutoconnectInterfaceFactory(info)
	if err != nil {
		Logf(LogWarning, "Could not auto-connect discovered %s %q: %v", interfaceType, info["name"], err)
		return
	}
	if ifc == nil {
		return
	}

	endpointHash := d.endpointHash(info)
	if len(endpointHash) > 0 {
		ifc.AutoconnectHash = append([]byte(nil), endpointHash...)
	}
	if networkID, _ := info["network_id"].(string); strings.TrimSpace(networkID) != "" {
		ifc.AutoconnectSource = networkID
	}
	ifc.OUT = true

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

func defaultDiscoveryAutoconnectInterfaceFactory(info map[string]any) (*Interface, error) {
	name, _ := info["name"].(string)
	reachableOn, _ := info["reachable_on"].(string)
	port := asDiscoveryInt(info["port"])
	if strings.TrimSpace(name) == "" || strings.TrimSpace(reachableOn) == "" || port <= 0 {
		return nil, errors.New("discovered interface missing reachable_on or port")
	}
	return ifaces.NewBackboneClientInterface(name, map[string]string{
		"target_host": reachableOn,
		"target_port": strconv.Itoa(port),
	})
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
		if !d.monitorOnce(time.Now()) {
			return
		}
	}
}

func (d *InterfaceDiscovery) monitorOnce(now time.Time) bool {
	if d == nil {
		return false
	}
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
		if ifc.Online {
			online++
			if !ifc.AutoconnectDown.IsZero() {
				ifc.AutoconnectDown = time.Time{}
			}
			continue
		}
		if ifc.AutoconnectDown.IsZero() {
			ifc.AutoconnectDown = now
			continue
		}
		if now.Sub(ifc.AutoconnectDown) >= detachAfter {
			detached = append(detached, ifc)
		}
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

	if online == 0 && Owner != nil && Owner.bootstrapInterfaceCount() == 0 {
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
		if handler := ifaces.RemoveInterfaceHandler; handler != nil {
			handler(ifc)
		}
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
	if len(d.monitored) == 0 && online == 0 {
		d.monitoring = false
	}
	running := d.monitoring
	d.mu.Unlock()
	return running
}

func NewBlackholeUpdater() *BlackholeUpdater {
	return &BlackholeUpdater{
		lastUpdates:    make(map[hashKey]time.Time),
		initialWait:    blackholeUpdaterInitialWait,
		jobInterval:    blackholeUpdaterJobInterval,
		updateInterval: blackholeUpdateInterval,
		sourceTimeout:  blackholeSourceTimeout,
		awaitPath:      AwaitPath,
		fetchList:      fetchRemoteBlackholeList,
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

func (u *BlackholeUpdater) running() bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.shouldRun
}

func (u *BlackholeUpdater) job() {
	if u == nil {
		return
	}
	if u.initialWait > 0 {
		u.sleep(u.initialWait)
	}
	for u.running() {
		u.updateOnce(time.Now())
		if !u.running() {
			return
		}
		interval := u.jobInterval
		if interval <= 0 {
			interval = blackholeUpdaterJobInterval
		}
		u.sleep(interval)
	}
}

func (u *BlackholeUpdater) due(now time.Time, sourceHash []byte) bool {
	key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(sourceHash)
	if !ok {
		return false
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	last := u.lastUpdates[key]
	interval := u.updateInterval
	if interval <= 0 {
		interval = blackholeUpdateInterval
	}
	return now.Sub(last) >= interval
}

func (u *BlackholeUpdater) markUpdated(sourceHash []byte, now time.Time) {
	key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(sourceHash)
	if !ok {
		return
	}
	u.mu.Lock()
	if u.lastUpdates == nil {
		u.lastUpdates = make(map[hashKey]time.Time)
	}
	u.lastUpdates[key] = now
	u.mu.Unlock()
}

func (u *BlackholeUpdater) updateOnce(now time.Time) {
	if u == nil {
		return
	}
	for _, sourceHash := range BlackholeSources() {
		if !u.due(now, sourceHash) {
			continue
		}
		destinationHash := HashFromNameAndIdentity("rnstransport.info.blackhole", sourceHash)
		if len(destinationHash) == 0 {
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
		response, err := u.fetchList(sourceHash, timeout)
		if err != nil {
			Logf(LogError, "Error while establishing link for blackhole list update from %s: %v", PrettyHexRep(sourceHash), err)
			continue
		}
		if len(sourceHash) != truncatedHashBytes {
			continue
		}
		decoded := make(map[hashKey]*blackholeEntry)
		switch typed := response.(type) {
		case map[hashKey]map[string]any:
			for key, entryValue := range typed {
				if entryValue == nil {
					continue
				}
				entry := &blackholeEntry{Source: copyBytes(sourceHash)}
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
				decoded[key] = entry
			}
		case map[hashKey]any:
			for key, entryValue := range typed {
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
				entry := &blackholeEntry{Source: copyBytes(sourceHash)}
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
				decoded[key] = entry
			}
		case map[any]any:
			for rawKey, rawValue := range typed {
				var key hashKey
				var ok bool
				switch v := rawKey.(type) {
				case string:
					key, ok = func(hash []byte) (hashKey, bool) {
						if len(hash) < truncatedHashBytes {
							return hashKey{}, false
						}
						var key hashKey
						copy(key[:], hash[:truncatedHashBytes])
						return key, true
					}([]byte(v))
				case umsgpack.BinaryKey:
					key, ok = func(hash []byte) (hashKey, bool) {
						if len(hash) < truncatedHashBytes {
							return hashKey{}, false
						}
						var key hashKey
						copy(key[:], hash[:truncatedHashBytes])
						return key, true
					}([]byte(string(v)))
				case []byte:
					key, ok = func(hash []byte) (hashKey, bool) {
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
				entry := &blackholeEntry{Source: copyBytes(sourceHash)}
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
				decoded[key] = entry
			}
		case map[string]any:
			for rawKey, rawValue := range typed {
				key, ok := func(hash []byte) (hashKey, bool) {
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
				entry := &blackholeEntry{Source: copyBytes(sourceHash)}
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
				decoded[key] = entry
			}
		}
		if len(decoded) == 0 {
			continue
		}
		added := 0
		serialisable := make(map[hashKey]map[string]any, len(decoded))
		blackholeMu.Lock()
		for key, entry := range decoded {
			if entry == nil {
				continue
			}
			serialised := map[string]any{"source": nil, "until": nil, "reason": nil}
			serialised["source"] = copyBytes(entry.Source)
			if entry.Until != nil && !entry.Until.IsZero() {
				serialised["until"] = float64(entry.Until.UnixNano()) / 1e9
			}
			if entry.Reason != nil {
				serialised["reason"] = *entry.Reason
			}
			serialisable[key] = serialised
			if _, exists := blackholedIdentities[key]; exists {
				continue
			}
			blackholedIdentities[key] = entry
			added++
		}
		blackholeMu.Unlock()

		if added > 0 {
			removeBlackholedPaths()
			if Owner != nil && strings.TrimSpace(Owner.StoragePath) != "" {
				sourcelistpath := filepath.Join(Owner.StoragePath, "blackhole", hex.EncodeToString(sourceHash))
				tmppath := sourcelistpath + ".tmp"
				payload := make(map[any]any, len(serialisable))
				for key, entry := range serialisable {
					payload[umsgpack.BinaryKey(string(key[:]))] = entry
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
		u.markUpdated(sourceHash, now)
		Logf(LogDebug, "Blackhole list update from %s completed", PrettyHexRep(sourceHash))
	}
}

func fetchRemoteBlackholeList(sourceHash []byte, timeout time.Duration) (any, error) {
	destinationHash := HashFromNameAndIdentity("rnstransport.info.blackhole", sourceHash)
	if len(destinationHash) == 0 {
		return nil, errors.New("invalid blackhole source identity hash")
	}
	remoteIdentity := IdentityRecall(destinationHash)
	if remoteIdentity == nil {
		remoteIdentity = IdentityRecall(sourceHash, true)
	}
	if remoteIdentity == nil {
		return nil, errors.New("remote blackhole source identity not known")
	}

	destination, err := NewDestination(remoteIdentity, DestinationOUT, DestinationSINGLE, TransportAppName, "info", "blackhole")
	if err != nil {
		return nil, err
	}
	defer DeregisterDestination(destination)

	establishedCh := make(chan *Link, 1)
	link, err := NewOutgoingLink(destination, LinkModeDefault, func(l *Link) {
		select {
		case establishedCh <- l:
		default:
		}
	}, nil)
	if err != nil {
		return nil, err
	}

	var established *Link
	select {
	case established = <-establishedCh:
	case <-time.After(timeout):
		if link != nil {
			link.Teardown()
		}
		return nil, errors.New("timed out establishing blackhole update link")
	}
	if established == nil {
		return nil, errors.New("blackhole update link was not established")
	}
	defer established.Teardown()

	rr := RequestReceiptFrom(established.Request("/list", nil, nil, nil, nil, timeout.Seconds()))
	if rr == nil {
		return nil, errors.New("blackhole list request could not be started")
	}
	deadline := time.Now().Add(timeout)
	for !rr.Concluded() && time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
	}
	if rr.Status() != ReceiptReady {
		return nil, errors.New("blackhole list request timed out or failed")
	}
	return rr.Response(), nil
}

func discoveryRawGet(raw map[any]any, want int) any {
	for k, v := range raw {
		switch x := k.(type) {
		case int:
			if x == want {
				return v
			}
		case int64:
			if int(x) == want {
				return v
			}
		case uint64:
			if int(x) == want {
				return v
			}
		case uint8:
			if int(x) == want {
				return v
			}
		}
	}
	return nil
}

func packDiscoveryInfo(info map[any]any) ([]byte, error) {
	if len(info) == 0 {
		return umsgpack.Packb(map[any]any{})
	}

	keys := orderedDiscoveryInfoKeys(info)
	var buf bytes.Buffer
	if err := writeDiscoveryMapHeader(&buf, len(keys)); err != nil {
		return nil, err
	}
	for _, key := range keys {
		value, ok := discoveryRawLookup(info, key)
		if !ok {
			continue
		}
		packedKey, err := umsgpack.Packb(key)
		if err != nil {
			return nil, err
		}
		if _, err := buf.Write(packedKey); err != nil {
			return nil, err
		}
		packedValue, err := umsgpack.Packb(value)
		if err != nil {
			return nil, err
		}
		if _, err := buf.Write(packedValue); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func orderedDiscoveryInfoKeys(info map[any]any) []int {
	keys := []int{
		discoveryFieldInterfaceType,
		discoveryFieldTransport,
		discoveryFieldTransportID,
		discoveryFieldName,
		discoveryFieldLatitude,
		discoveryFieldLongitude,
		discoveryFieldHeight,
	}

	ifType := sanitizeDiscoveryString(asDiscoveryString(discoveryRawGet(info, discoveryFieldInterfaceType)))
	switch ifType {
	case "BackboneInterface", "TCPServerInterface":
		keys = append(keys, discoveryFieldReachableOn, discoveryFieldPort)
	case "I2PInterface":
		keys = append(keys, discoveryFieldReachableOn)
	case "RNodeInterface", "RNodeMultiInterface":
		keys = append(keys, discoveryFieldFrequency, discoveryFieldBandwidth, discoveryFieldSpreading, discoveryFieldCodingRate)
	case "WeaveInterface":
		keys = append(keys, discoveryFieldFrequency, discoveryFieldBandwidth, discoveryFieldChannel, discoveryFieldModulation)
	case "KISSInterface":
		keys = append(keys, discoveryFieldFrequency, discoveryFieldBandwidth, discoveryFieldModulation)
	}

	if _, ok := discoveryRawLookup(info, discoveryFieldIFACNetname); ok {
		keys = append(keys, discoveryFieldIFACNetname)
	}
	if _, ok := discoveryRawLookup(info, discoveryFieldIFACNetkey); ok {
		keys = append(keys, discoveryFieldIFACNetkey)
	}

	filtered := make([]int, 0, len(keys))
	seen := make(map[int]struct{}, len(keys))
	for _, key := range keys {
		if _, ok := seen[key]; ok {
			continue
		}
		if _, ok := discoveryRawLookup(info, key); !ok {
			continue
		}
		seen[key] = struct{}{}
		filtered = append(filtered, key)
	}
	return filtered
}

func discoveryRawLookup(raw map[any]any, want int) (any, bool) {
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

func writeDiscoveryMapHeader(buf *bytes.Buffer, n int) error {
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

func sanitizeDiscoveryString(in string) string {
	s := strings.ReplaceAll(in, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	return strings.TrimSpace(s)
}

func isDiscoveryAddress(v string) bool {
	if ip := net.ParseIP(v); ip != nil {
		return true
	}
	v = strings.TrimSuffix(v, ".")
	if len(v) == 0 || len(v) > 253 {
		return false
	}
	parts := strings.Split(v, ".")
	if len(parts) == 0 {
		return false
	}
	if _, err := strconvAtoi(parts[len(parts)-1]); err == nil {
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

func resolveDiscoveryReachableOn(ifc *Interface, reachableOn string) (string, error) {
	reachableOn = sanitizeDiscoveryString(reachableOn)
	if umsgpack.IsWindows() {
		return reachableOn, nil
	}
	execPath, ok := expandDiscoveryExecutablePath(reachableOn)
	if !ok {
		return reachableOn, nil
	}
	out, err := exec.Command(execPath).Output()
	if err != nil {
		return "", fmt.Errorf("error while getting reachable_on from executable at %s: %w", ifc.DiscoveryReachableOnValue(), err)
	}
	resolved := sanitizeDiscoveryString(string(out))
	if !isDiscoveryAddress(resolved) {
		return "", fmt.Errorf("valid IP address or hostname was not found in external script output %q", resolved)
	}
	return resolved, nil
}

func expandDiscoveryExecutablePath(v string) (string, bool) {
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

func requireDiscoveryStampProvider() bool {
	if DiscoveryStampProvider != nil {
		return true
	}
	Log("Using on-network interface discovery requires the LXMF module to be installed.", LogCritical)
	Log("You can install it with the command: pip install lxmf", LogCritical)
	if discoveryPanic != nil {
		discoveryPanic()
	}
	return false
}

func discoveryHopsTo(destinationHash []byte) int {
	if len(destinationHash) == 0 {
		return 0
	}
	return HopsTo(destinationHash)
}

func asDiscoveryString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	default:
		return ""
	}
}

func asDiscoveryBytes(v any) []byte {
	switch x := v.(type) {
	case []byte:
		return append([]byte(nil), x...)
	case string:
		return []byte(x)
	default:
		return nil
	}
}

func asDiscoveryBool(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	default:
		return false
	}
}

func asDiscoveryFloat(v any) float64 {
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

func asDiscoveryInt(v any) int {
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
}

func asDiscoveryInt64(v any) int64 {
	switch x := v.(type) {
	case int:
		return int64(x)
	case int64:
		return x
	case uint64:
		return int64(x)
	default:
		return 0
	}
}

func asDiscoveryMaybeInt(v any) *int {
	val := asDiscoveryInt(v)
	if val == 0 {
		return nil
	}
	return &val
}

func derefFloat64(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

func strconvAtoi(s string) (int, error) {
	var sign int = 1
	if strings.HasPrefix(s, "-") {
		sign = -1
		s = s[1:]
	}
	if s == "" {
		return 0, errors.New("empty")
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, errors.New("not int")
		}
		n = n*10 + int(r-'0')
	}
	return sign * n, nil
}
