package interfaces

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	serial "go.bug.st/serial"
)

const (
	rnodeMultiHWMTU = 508

	rnodeMultiRequiredFWVerMaj = 1
	rnodeMultiRequiredFWVerMin = 74

	rnodeMultiCallsignMaxLen = 32
)

// RNode multi uses the same type IDs as Python.
const (
	sx127x = 0x00
	sx1276 = 0x01
	sx1278 = 0x02
	sx126x = 0x10
	sx1262 = 0x11
	sx128x = 0x20
	sx1280 = 0x21
)

// RNode multi-specific commands/values (not present in rnode_Interface.go).
const (
	cmdInterfaces = 0x71
	cmdSelInt     = 0x1F

	cmdInt0Data  = 0x00
	cmdInt1Data  = 0x10
	cmdInt2Data  = 0x20
	cmdInt3Data  = 0x70
	cmdInt4Data  = 0x75
	cmdInt5Data  = 0x90
	cmdInt6Data  = 0xA0
	cmdInt7Data  = 0xB0
	cmdInt8Data  = 0xC0
	cmdInt9Data  = 0xD0
	cmdInt10Data = 0xE0
	cmdInt11Data = 0xF0
)

func isIntDataCmd(cmd byte) bool {
	switch cmd {
	case cmdInt0Data, cmdInt1Data, cmdInt2Data, cmdInt3Data, cmdInt4Data, cmdInt5Data,
		cmdInt6Data, cmdInt7Data, cmdInt8Data, cmdInt9Data, cmdInt10Data, cmdInt11Data:
		return true
	default:
		return false
	}
}

func intDataCmdToIndex(cmd byte) (byte, bool) {
	switch cmd {
	case cmdInt0Data:
		return 0, true
	case cmdInt1Data:
		return 1, true
	case cmdInt2Data:
		return 2, true
	case cmdInt3Data:
		return 3, true
	case cmdInt4Data:
		return 4, true
	case cmdInt5Data:
		return 5, true
	case cmdInt6Data:
		return 6, true
	case cmdInt7Data:
		return 7, true
	case cmdInt8Data:
		return 8, true
	case cmdInt9Data:
		return 9, true
	case cmdInt10Data:
		return 10, true
	case cmdInt11Data:
		return 11, true
	default:
		return 0, false
	}
}

func interfaceTypeToStr(t byte) string {
	switch t {
	case sx126x, sx1262:
		return "SX126X"
	case sx127x, sx1276, sx1278:
		return "SX127X"
	case sx128x, sx1280:
		return "SX128X"
	default:
		return "SX127X"
	}
}

type rnodeMultiSubConfig struct {
	Name        string
	VPort       byte
	Outgoing    bool
	FlowControl bool

	Frequency uint32
	Bandwidth uint32
	TXPower   int8
	SF        byte
	CR        byte

	STALock *float64
	LTALock *float64
}

type rnodeMultiConfig struct {
	Name string
	Port string

	Speed int

	IDCallsign []byte
	IDInterval time.Duration

	Subinterfaces []rnodeMultiSubConfig
}

type RNodeMultiDriver struct {
	parent *Interface
	cfg    rnodeMultiConfig

	serialMu sync.Mutex
	port     serial.Port

	sendMu sync.Mutex

	stopCh chan struct{}

	detected    atomic.Bool
	firmwareOK  atomic.Bool
	majVer      atomic.Uint32
	minVer      atomic.Uint32
	platform    atomic.Uint32
	mcu         atomic.Uint32
	display     atomic.Bool
	randomByte  atomic.Uint32
	selectedIdx atomic.Int32
	firstTxAt   atomic.Int64 // unixnano; like Python first_tx

	interfaceTypesMu sync.Mutex
	interfaceTypes   []string

	subMu sync.Mutex
	subs  map[byte]*Interface

	reconnecting atomic.Bool

	// Channel statistics reported by the RNode (global, not per subinterface in Python).
	rAirtimeShortBits     atomic.Uint32 // float32bits (percent)
	rAirtimeLongBits      atomic.Uint32 // float32bits (percent)
	rChannelLoadShortBits atomic.Uint32 // float32bits (percent)
	rChannelLoadLongBits  atomic.Uint32 // float32bits (percent)

	rCurrentRSSI           atomic.Int32
	rNoiseFloor            atomic.Int32
	rInterference          atomic.Int32 // RSSI-based, MinInt32 means nil
	rInterferenceAtUnixSec atomic.Int64
}

var rnodeMultiStartDriver = func(d *RNodeMultiDriver) error { return d.start() }
var rnodeMultiStartReconnectLoop = func(d *RNodeMultiDriver) { go d.reconnectLoop() }

func NewRNodeMultiInterface(name string, kv map[string]string) (*Interface, error) {
	cfg, err := parseRNodeMultiConfig(strings.TrimSpace(name), kv)
	if err != nil {
		return nil, err
	}

	parent := &Interface{
		Name:              cfg.Name,
		Type:              "RNodeMultiInterface",
		IN:                false,
		OUT:               false,
		DriverImplemented: true,
		Online:            false,
		HWMTU:             rnodeMultiHWMTU,
		FixedMTU:          true,
	}

	driver := &RNodeMultiDriver{
		parent: parent,
		cfg:    cfg,
		stopCh: make(chan struct{}),
		subs:   make(map[byte]*Interface),
	}
	parent.rnodeMulti = driver

	if err := rnodeMultiStartDriver(driver); err != nil {
		if DiagLogf != nil {
			DiagLogf(LogError, "Could not open serial port for interface %s", parent)
			DiagLogf(LogError, "The contained exception was: %v", err)
			DiagLogf(LogError, "Reticulum will attempt to bring up this interface periodically")
		}
		rnodeMultiStartReconnectLoop(driver)
		return parent, nil
	}
	return parent, nil
}

func parseRNodeMultiConfig(name string, kv map[string]string) (rnodeMultiConfig, error) {
	get := func(key string) string {
		if kv == nil {
			return ""
		}
		if v, ok := kv[key]; ok {
			return strings.TrimSpace(v)
		}
		lower := strings.ToLower(key)
		for k, v := range kv {
			if strings.ToLower(k) == lower {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}

	cfg := rnodeMultiConfig{
		Name:  name,
		Port:  get("port"),
		Speed: parseIntOr(get("speed"), 115200),
	}
	if strings.TrimSpace(cfg.Port) == "" {
		return cfg, fmt.Errorf("no port specified for %s", name)
	}
	if cfg.Speed <= 0 {
		cfg.Speed = 115200
	}

	if idInterval := parseIntOr(get("id_interval"), 0); idInterval > 0 {
		if idCalls := strings.TrimSpace(get("id_callsign")); idCalls != "" {
			enc := []byte(idCalls)
			if len(enc) > rnodeMultiCallsignMaxLen {
				return cfg, fmt.Errorf("id_callsign exceeds %d bytes when encoded", rnodeMultiCallsignMaxLen)
			}
			cfg.IDCallsign = enc
			cfg.IDInterval = time.Duration(idInterval) * time.Second
		}
	}

	// Parse subinterfaces encoded by reticulum.go ConfigObj parser:
	// key format: "sub.<SubName>.<key>".
	subKV := map[string]map[string]string{}
	for k, v := range kv {
		lk := strings.ToLower(strings.TrimSpace(k))
		if !strings.HasPrefix(lk, "sub.") {
			continue
		}
		rest := strings.TrimPrefix(lk, "sub.")
		parts := strings.SplitN(rest, ".", 2)
		if len(parts) != 2 {
			continue
		}
		subName := strings.TrimSpace(parts[0])
		subKey := strings.TrimSpace(parts[1])
		if subName == "" || subKey == "" {
			continue
		}
		if _, ok := subKV[subName]; !ok {
			subKV[subName] = map[string]string{}
		}
		subKV[subName][subKey] = strings.TrimSpace(v)
	}

	if len(subKV) == 0 {
		return cfg, fmt.Errorf("no subinterfaces configured for %s (expected [[[Sub]]] sections)", name)
	}

	// Stable ordering for reproducible behaviour/logging.
	subNames := make([]string, 0, len(subKV))
	for n := range subKV {
		subNames = append(subNames, n)
	}
	sort.Strings(subNames)

	parentEnabled := parseTruthy(get("enabled"), true)

	for _, subName := range subNames {
		sub := subKV[subName]

		enabledExplicit := false
		enabled := parentEnabled
		if raw := getFirst(sub, "interface_enabled", "enabled"); strings.TrimSpace(raw) != "" {
			enabledExplicit = true
			enabled = parseTruthy(raw, true)
		}
		if enabledExplicit && !enabled {
			continue
		}
		if !enabledExplicit && !parentEnabled {
			continue
		}

		vport := parseIntOr(getFirst(sub, "vport"), -1)
		if vport < 0 || vport > 255 {
			return cfg, fmt.Errorf("invalid vport for subinterface %q", subName)
		}

		outgoing := true
		if raw := getFirst(sub, "outgoing"); strings.TrimSpace(raw) != "" {
			outgoing = parseTruthy(raw, true)
		}

		freq := uint32(parseIntOr(getFirst(sub, "frequency"), 0))
		bw := uint32(parseIntOr(getFirst(sub, "bandwidth"), 0))
		txp := int8(parseIntOr(getFirst(sub, "txpower"), 0))
		sf := byte(parseIntOr(getFirst(sub, "spreadingfactor"), 0))
		cr := byte(parseIntOr(getFirst(sub, "codingrate"), 0))

		if freq == 0 || bw == 0 || sf == 0 || cr == 0 {
			return cfg, fmt.Errorf("missing radio parameters for subinterface %q", subName)
		}

		flowControl := parseBoolOr(getFirst(sub, "flow_control"), false)

		var sta *float64
		if raw := strings.TrimSpace(getFirst(sub, "airtime_limit_short")); raw != "" {
			if v, ok := parseFloatSimple(raw); ok {
				sta = &v
			}
		}
		var lta *float64
		if raw := strings.TrimSpace(getFirst(sub, "airtime_limit_long")); raw != "" {
			if v, ok := parseFloatSimple(raw); ok {
				lta = &v
			}
		}

		cfg.Subinterfaces = append(cfg.Subinterfaces, rnodeMultiSubConfig{
			Name:        subName,
			VPort:       byte(vport),
			Outgoing:    outgoing,
			FlowControl: flowControl,
			Frequency:   freq,
			Bandwidth:   bw,
			TXPower:     txp,
			SF:          sf,
			CR:          cr,
			STALock:     sta,
			LTALock:     lta,
		})
	}

	if len(cfg.Subinterfaces) == 0 {
		return cfg, fmt.Errorf("no subinterfaces enabled for %s", name)
	}

	return cfg, nil
}

func getFirst(m map[string]string, keys ...string) string {
	for _, k := range keys {
		if m == nil {
			return ""
		}
		lk := strings.ToLower(k)
		if v, ok := m[lk]; ok {
			return strings.TrimSpace(v)
		}
		for mk, mv := range m {
			if strings.ToLower(mk) == lk {
				return strings.TrimSpace(mv)
			}
		}
	}
	return ""
}

func parseTruthy(v string, defaultVal bool) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return defaultVal
	}
	switch v {
	case "yes", "on", "1", "true", "enabled":
		return true
	case "no", "off", "0", "false", "disabled":
		return false
	default:
		return defaultVal
	}
}

func parseFloatSimple(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	var v float64
	if _, err := fmt.Sscanf(s, "%f", &v); err != nil {
		return 0, false
	}
	return v, true
}

func (d *RNodeMultiDriver) start() error {
	if DiagLogf != nil {
		DiagLogf(LogVerbose, "Opening serial port %s for %s...", d.cfg.Port, d.parent)
	}
	if err := d.openPort(); err != nil {
		return err
	}

	// Python: sleep(2.0) before reader/detect.
	time.Sleep(2 * time.Second)

	d.parent.Online = true
	go d.readLoop()

	// Detect.
	if err := d.detect(); err != nil {
		d.parent.Online = false
		d.closePort()
		return err
	}

	// Wait a bit for detect/metadata.
	detectDeadline := time.Now().Add(1500 * time.Millisecond)
	for !d.detected.Load() && time.Now().Before(detectDeadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if !d.detected.Load() {
		d.parent.Online = false
		d.closePort()
		return fmt.Errorf("could not detect device for %s", d.parent)
	}

	// Python: sleep(0.2) after detect to allow metadata to arrive.
	time.Sleep(200 * time.Millisecond)

	typesDeadline := time.Now().Add(800 * time.Millisecond)
	for {
		d.interfaceTypesMu.Lock()
		n := len(d.interfaceTypes)
		d.interfaceTypesMu.Unlock()
		if n > 0 || time.Now().After(typesDeadline) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Spawn and configure subinterfaces.
	if err := d.spawnSubinterfaces(); err != nil {
		d.parent.Online = false
		d.closePort()
		d.teardownSubinterfaces()
		return err
	}

	// Optional ID beacons (Python sends on all online subinterfaces).
	if len(d.cfg.IDCallsign) > 0 && d.cfg.IDInterval > 0 {
		go d.idBeaconLoop()
	}

	return nil
}

func (d *RNodeMultiDriver) Close() {
	if d == nil {
		return
	}
	select {
	case <-d.stopCh:
	default:
		close(d.stopCh)
	}
	d.parent.Online = false
	d.disableExternalFramebuffer()
	d.teardownSubinterfaces()
	_ = d.leave()
	d.closePort()
}

func (d *RNodeMultiDriver) getPort() serial.Port {
	d.serialMu.Lock()
	p := d.port
	d.serialMu.Unlock()
	return p
}

func (d *RNodeMultiDriver) setPort(p serial.Port) {
	d.serialMu.Lock()
	old := d.port
	d.port = p
	d.serialMu.Unlock()
	if old != nil && old != p {
		_ = old.Close()
	}
}

func (d *RNodeMultiDriver) openPort() error {
	mode := &serial.Mode{BaudRate: d.cfg.Speed}
	p, err := serial.Open(d.cfg.Port, mode)
	if err != nil {
		return err
	}
	if err := p.SetReadTimeout(0); err != nil {
		_ = p.Close()
		return err
	}
	d.setPort(p)
	return nil
}

func (d *RNodeMultiDriver) closePort() {
	d.setPort(nil)
}

func (d *RNodeMultiDriver) writeBytes(b []byte) error {
	p := d.getPort()
	if p == nil {
		return errors.New("serial closed")
	}
	d.sendMu.Lock()
	n, err := p.Write(b)
	d.sendMu.Unlock()
	if err != nil {
		return err
	}
	if n != len(b) {
		return fmt.Errorf("short write: %d/%d", n, len(b))
	}
	return nil
}

func (d *RNodeMultiDriver) detect() error {
	// Python: FEND DETECT req FEND FW 0 FEND PLATFORM 0 FEND MCU 0 FEND INTERFACES 0 FEND
	cmd := []byte{
		FEND, CMD_DETECT, DETECT_REQ, FEND,
		CMD_FW_VERSION, 0x00, FEND,
		CMD_PLATFORM, 0x00, FEND,
		CMD_MCU, 0x00, FEND,
		cmdInterfaces, 0x00, FEND,
	}
	return d.writeBytes(cmd)
}

func (d *RNodeMultiDriver) leave() error {
	return d.writeBytes([]byte{FEND, CMD_LEAVE, 0xFF, FEND})
}

func (d *RNodeMultiDriver) disableExternalFramebuffer() {
	if !d.display.Load() {
		return
	}
	_ = d.writeBytes([]byte{FEND, CMD_FB_EXT, 0x00, FEND})
}

func (d *RNodeMultiDriver) validateFirmware() {
	maj := d.majVer.Load()
	min := d.minVer.Load()

	if maj > rnodeMultiRequiredFWVerMaj || (maj == rnodeMultiRequiredFWVerMaj && min >= rnodeMultiRequiredFWVerMin) {
		d.firmwareOK.Store(true)
		return
	}

	if DiagLogf != nil {
		DiagLogf(LogError, "The firmware version of the connected RNode is %d.%d", maj, min)
		DiagLogf(LogError, "This version of Reticulum requires at least version %d.%d", rnodeMultiRequiredFWVerMaj, rnodeMultiRequiredFWVerMin)
		DiagLogf(LogError, "Please update your RNode firmware with rnodeconf from the Reticulum repository")
	}
	if PanicFunc != nil {
		PanicFunc()
		return
	}
	panic(fmt.Errorf("rnode firmware too old: %d.%d", maj, min))
}

func (d *RNodeMultiDriver) setU32(cmd byte, v uint32, idx byte) error {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], v)
	data := kissEscape(b[:])
	// FEND SEL idx FEND  FEND cmd data FEND
	frame := append([]byte{FEND, cmdSelInt, idx, FEND, FEND, cmd}, data...)
	frame = append(frame, FEND)
	return d.writeBytes(frame)
}

func (d *RNodeMultiDriver) setI8(cmd byte, v int8, idx byte) error {
	frame := []byte{FEND, cmdSelInt, idx, FEND, FEND, cmd, byte(v), FEND}
	return d.writeBytes(frame)
}

func (d *RNodeMultiDriver) setU8(cmd byte, v byte, idx byte) error {
	frame := []byte{FEND, cmdSelInt, idx, FEND, FEND, cmd, v, FEND}
	return d.writeBytes(frame)
}

func (d *RNodeMultiDriver) setAirtime(cmd byte, percent float64, idx byte) error {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	at := uint16(percent * 100)
	data := kissEscape([]byte{byte(at >> 8), byte(at)})
	frame := append([]byte{FEND, cmdSelInt, idx, FEND, FEND, cmd}, data...)
	frame = append(frame, FEND)
	return d.writeBytes(frame)
}

func makeSelDataFrame(idx byte, payload []byte) (frame []byte, escapedLen int) {
	// Python escapes the payload and accounts txb using the escaped length.
	escaped := kissEscape(payload)
	frame = append([]byte{FEND, cmdSelInt, idx, FEND, FEND, CMD_DATA}, escaped...)
	frame = append(frame, FEND)
	return frame, len(escaped)
}

func (d *RNodeMultiDriver) addSub(idx byte, ifc *Interface) {
	d.subMu.Lock()
	d.subs[idx] = ifc
	d.subMu.Unlock()
}

func (d *RNodeMultiDriver) getSub(idx byte) *Interface {
	d.subMu.Lock()
	ifc := d.subs[idx]
	d.subMu.Unlock()
	return ifc
}

func (d *RNodeMultiDriver) teardownSubinterfaces() {
	d.subMu.Lock()
	subs := make([]*Interface, 0, len(d.subs))
	for _, ifc := range d.subs {
		subs = append(subs, ifc)
	}
	d.subs = make(map[byte]*Interface)
	d.subMu.Unlock()

	for _, ifc := range subs {
		removeInterface(ifc)
	}
}

func (d *RNodeMultiDriver) spawnSubinterfaces() error {
	d.interfaceTypesMu.Lock()
	types := append([]string(nil), d.interfaceTypes...)
	d.interfaceTypesMu.Unlock()

	if len(types) == 0 {
		return errors.New("device did not report any subinterfaces")
	}

	for _, sub := range d.cfg.Subinterfaces {
		if int(sub.VPort) >= len(types) {
			return fmt.Errorf("virtual port %d for subinterface %q does not exist on %s", sub.VPort, sub.Name, d.cfg.Name)
		}
		ifType := types[int(sub.VPort)]

		child := &Interface{
			Name:              fmt.Sprintf("%s[%s]", d.cfg.Name, sub.Name),
			Type:              "RNodeSubInterface",
			Parent:            d.parent,
			IN:                true,
			OUT:               sub.Outgoing,
			DriverImplemented: true,
			Online:            false,
			HWMTU:             rnodeMultiHWMTU,
			FixedMTU:          true,
		}

		// Copy selected common configuration like Python does.
		child.Mode = d.parent.Mode
		child.AnnounceCap = d.parent.AnnounceCap
		child.SetAnnounceRateConfig(d.parent.AnnounceRateTarget, d.parent.AnnounceRateGrace, d.parent.AnnounceRatePenalty)
		child.SetAnnounceRate(d.parent.AnnounceRateTarget, d.parent.AnnounceRateGrace, d.parent.AnnounceRatePenalty)

		driver := &RNodeSubDriver{
			parent:      d,
			iface:       child,
			name:        sub.Name,
			index:       sub.VPort,
			ifType:      ifType,
			flowControl: sub.FlowControl,
			frequency:   sub.Frequency,
			bandwidth:   sub.Bandwidth,
			txpower:     sub.TXPower,
			sf:          sub.SF,
			cr:          sub.CR,
			stalock:     sub.STALock,
			ltalock:     sub.LTALock,
		}
		child.rnodeSub = driver

		d.addSub(sub.VPort, child)

		if SpawnHandler != nil {
			SpawnHandler(child)
		}

		// Configure radio and validate.
		if err := driver.configure(); err != nil {
			if DiagLogf != nil {
				DiagLogf(LogError, "Failed to configure %s: %v", child, err)
			}
			continue
		}
	}

	return nil
}

func (d *RNodeMultiDriver) processQueueAll() {
	d.subMu.Lock()
	subs := make([]*Interface, 0, len(d.subs))
	for _, ifc := range d.subs {
		subs = append(subs, ifc)
	}
	d.subMu.Unlock()
	for _, ifc := range subs {
		if ifc == nil || ifc.rnodeSub == nil {
			continue
		}
		ifc.rnodeSub.processQueue()
	}
}

func (d *RNodeMultiDriver) ProcessOutgoing(sub *Interface, data []byte) {
	if d == nil || sub == nil || len(data) == 0 {
		return
	}
	if !d.parent.Online {
		return
	}
	if sub.rnodeSub == nil {
		return
	}
	drv := sub.rnodeSub

	if drv.flowControl && !drv.ready.Load() {
		drv.enqueue(data)
		return
	}
	if drv.flowControl {
		drv.ready.Store(false)
	}

	// Python: first_tx is set on first non-beacon TX, and cleared when sending the beacon payload.
	if len(d.cfg.IDCallsign) > 0 && bytes.Equal(data, d.cfg.IDCallsign) {
		d.firstTxAt.Store(0)
	} else if d.firstTxAt.Load() == 0 {
		d.firstTxAt.Store(time.Now().UnixNano())
	}

	frame, escapedLen := makeSelDataFrame(drv.index, data)
	if err := d.writeBytes(frame); err != nil {
		if DiagLogf != nil {
			DiagLogf(LogError, "Serial error while transmitting via %s: %v", sub, err)
		}
		d.parent.Online = false
		d.closePort()
		go d.reconnectLoop()
		return
	}

	atomic.AddUint64(&sub.TXB, uint64(len(data)))
	// Python parent txb uses the escaped payload length (not raw payload bytes).
	atomic.AddUint64(&d.parent.TXB, uint64(escapedLen))
	if parent := d.parent.Parent; parent != nil {
		atomic.AddUint64(&parent.TXB, uint64(escapedLen))
	}
}

func (d *RNodeMultiDriver) shouldSendBeacon(now time.Time) bool {
	if d == nil {
		return false
	}
	if len(d.cfg.IDCallsign) == 0 || d.cfg.IDInterval <= 0 {
		return false
	}
	first := d.firstTxAt.Load()
	if first == 0 {
		return false
	}
	return now.After(time.Unix(0, first).Add(d.cfg.IDInterval))
}

func (d *RNodeMultiDriver) idBeaconLoop() {
	t := time.NewTicker(80 * time.Millisecond) // Python checks beacon condition in read loop idle path
	defer t.Stop()
	for {
		select {
		case <-d.stopCh:
			return
		case <-t.C:
			if !d.parent.Online {
				continue
			}
			if !d.shouldSendBeacon(time.Now()) {
				continue
			}

			// Send on all online subinterfaces (and reset firstTx via ProcessOutgoing beacon handling).
			d.subMu.Lock()
			subs := make([]*Interface, 0, len(d.subs))
			for _, ifc := range d.subs {
				subs = append(subs, ifc)
			}
			d.subMu.Unlock()
			for _, ifc := range subs {
				if ifc == nil || ifc.rnodeSub == nil || !ifc.Online || !ifc.OUT {
					continue
				}
				ifc.ProcessOutgoing(d.cfg.IDCallsign)
			}
		}
	}
}

func (d *RNodeMultiDriver) handleSerialError(err error) {
	d.parent.Online = false
	if DiagLogf != nil {
		DiagLogf(LogError, "A serial port error occurred, the contained exception was: %v", err)
		DiagLogf(LogError, "The interface %s experienced an unrecoverable error and is now offline.", d.parent)
	}
	if PanicOnInterfaceErrorProvider != nil && PanicOnInterfaceErrorProvider() {
		if PanicFunc != nil {
			PanicFunc()
			return
		}
		panic(err)
	}
	if DiagLogf != nil {
		DiagLogf(LogError, "Reticulum will attempt to reconnect the interface periodically.")
	}
	d.teardownSubinterfaces()
}

func (d *RNodeMultiDriver) readLoop() {
	defer func() {
		d.parent.Online = false
		d.closePort()
		if DiagLogf != nil {
			DiagLogf(LogWarning, "%s offline (read loop ended)", d.parent)
		}
		go d.reconnectLoop()
	}()

	p := d.getPort()
	if p == nil {
		return
	}
	br := bufio.NewReaderSize(p, 4096)

	inFrame := false
	escape := false
	command := byte(CMD_UNKNOWN)
	var dataBuf bytes.Buffer
	var cmdBuf bytes.Buffer

	var lastReadMS int64

	for {
		select {
		case <-d.stopCh:
			return
		default:
		}

		b, err := br.ReadByte()
		if err != nil {
			d.handleSerialError(err)
			return
		}
		lastReadMS = time.Now().UnixMilli()

		// End of frame.
		if inFrame && b == FEND && (command == CMD_DATA || isIntDataCmd(command)) {
			inFrame = false
			idx := byte(d.selectedIdx.Load())
			if command != CMD_DATA {
				if v, ok := intDataCmdToIndex(command); ok {
					idx = v
				}
			}
			if ifc := d.getSub(idx); ifc != nil && ifc.rnodeSub != nil {
				ifc.rnodeSub.processIncoming(dataBuf.Bytes())
			}
			dataBuf.Reset()
			cmdBuf.Reset()
			command = CMD_UNKNOWN
			escape = false
			continue
		}

		// Start new frame.
		if b == FEND {
			inFrame = true
			command = CMD_UNKNOWN
			dataBuf.Reset()
			cmdBuf.Reset()
			escape = false
			continue
		}

		if !inFrame || dataBuf.Len() >= rnodeMultiHWMTU {
			continue
		}

		// First byte after FEND is the command byte.
		if command == CMD_UNKNOWN {
			command = b
			continue
		}

		// Data payload.
		if command == CMD_DATA || isIntDataCmd(command) {
			kissUnescapeStream(&dataBuf, b, &escape)
			continue
		}

		// Non-data commands.
		switch command {
		case cmdSelInt:
			d.selectedIdx.Store(int32(b))

		case CMD_FREQUENCY:
			kissUnescapeStream(&cmdBuf, b, &escape)
			if cmdBuf.Len() == 4 {
				v := binary.BigEndian.Uint32(cmdBuf.Bytes())
				if ifc := d.getSub(byte(d.selectedIdx.Load())); ifc != nil && ifc.rnodeSub != nil {
					ifc.rnodeSub.rFrequency.Store(v)
					ifc.rnodeSub.repFreq.Store(true)
					ifc.rnodeSub.updateBitrateFromReported()
				}
				cmdBuf.Reset()
			}

		case CMD_BANDWIDTH:
			kissUnescapeStream(&cmdBuf, b, &escape)
			if cmdBuf.Len() == 4 {
				v := binary.BigEndian.Uint32(cmdBuf.Bytes())
				if ifc := d.getSub(byte(d.selectedIdx.Load())); ifc != nil && ifc.rnodeSub != nil {
					ifc.rnodeSub.rBandwidth.Store(v)
					ifc.rnodeSub.repBW.Store(true)
					ifc.rnodeSub.updateBitrateFromReported()
				}
				cmdBuf.Reset()
			}

		case CMD_TXPOWER:
			if ifc := d.getSub(byte(d.selectedIdx.Load())); ifc != nil && ifc.rnodeSub != nil {
				ifc.rnodeSub.rTXPower.Store(int32(int8(b)))
				ifc.rnodeSub.repTXP.Store(true)
			}

		case CMD_SF:
			if ifc := d.getSub(byte(d.selectedIdx.Load())); ifc != nil && ifc.rnodeSub != nil {
				ifc.rnodeSub.rSF.Store(uint32(b))
				ifc.rnodeSub.repSF.Store(true)
				ifc.rnodeSub.updateBitrateFromReported()
			}

		case CMD_CR:
			if ifc := d.getSub(byte(d.selectedIdx.Load())); ifc != nil && ifc.rnodeSub != nil {
				ifc.rnodeSub.rCR.Store(uint32(b))
				ifc.rnodeSub.updateBitrateFromReported()
			}

		case CMD_RADIO_STATE:
			if ifc := d.getSub(byte(d.selectedIdx.Load())); ifc != nil && ifc.rnodeSub != nil {
				ifc.rnodeSub.rState.Store(uint32(b))
				ifc.rnodeSub.repState.Store(true)
			}

		case CMD_RADIO_LOCK:
			if ifc := d.getSub(byte(d.selectedIdx.Load())); ifc != nil && ifc.rnodeSub != nil {
				ifc.rnodeSub.rLock.Store(uint32(b))
			}

		case CMD_FW_VERSION:
			kissUnescapeStream(&cmdBuf, b, &escape)
			if cmdBuf.Len() == 2 {
				d.majVer.Store(uint32(cmdBuf.Bytes()[0]))
				d.minVer.Store(uint32(cmdBuf.Bytes()[1]))
				cmdBuf.Reset()
				d.validateFirmware()
			}

		case CMD_STAT_CHTM:
			// Python multi: 8 bytes (ats, atl, cus, cul) in centi-percent.
			// Some firmwares include extra fields like current RSSI/noise floor/interference (11 bytes total).
			kissUnescapeStream(&cmdBuf, b, &escape)
			if cmdBuf.Len() == 8 || cmdBuf.Len() == 11 {
				cb := cmdBuf.Bytes()
				ats := binary.BigEndian.Uint16(cb[0:2])
				atl := binary.BigEndian.Uint16(cb[2:4])
				cus := binary.BigEndian.Uint16(cb[4:6])
				cul := binary.BigEndian.Uint16(cb[6:8])
				d.rAirtimeShortBits.Store(math.Float32bits(float32(ats) / 100.0))
				d.rAirtimeLongBits.Store(math.Float32bits(float32(atl) / 100.0))
				d.rChannelLoadShortBits.Store(math.Float32bits(float32(cus) / 100.0))
				d.rChannelLoadLongBits.Store(math.Float32bits(float32(cul) / 100.0))
				if cmdBuf.Len() == 11 {
					crs := cb[8]
					nfl := cb[9]
					ntf := cb[10]

					d.rCurrentRSSI.Store(int32(int(crs) - RSSI_OFFSET))
					d.rNoiseFloor.Store(int32(int(nfl) - RSSI_OFFSET))
					if ntf == 0xFF {
						d.rInterference.Store(math.MinInt32)
					} else {
						d.rInterference.Store(int32(int(ntf) - RSSI_OFFSET))
						d.rInterferenceAtUnixSec.Store(time.Now().Unix())
					}
				}
				cmdBuf.Reset()
			}

		case CMD_STAT_PHYPRM:
			// Python multi: 10 bytes (lst, lsr, prs, prt, cst). Some firmwares include 12 bytes (difs).
			kissUnescapeStream(&cmdBuf, b, &escape)
			if cmdBuf.Len() == 10 || cmdBuf.Len() == 12 {
				cb := cmdBuf.Bytes()
				lst := float32(binary.BigEndian.Uint16(cb[0:2])) / 1000.0
				lsr := binary.BigEndian.Uint16(cb[2:4])
				prs := binary.BigEndian.Uint16(cb[4:6])
				prt := binary.BigEndian.Uint16(cb[6:8])
				cst := binary.BigEndian.Uint16(cb[8:10])
				var difs uint16
				if cmdBuf.Len() == 12 {
					difs = binary.BigEndian.Uint16(cb[10:12])
				}

				if ifc := d.getSub(byte(d.selectedIdx.Load())); ifc != nil && ifc.rnodeSub != nil {
					s := ifc.rnodeSub
					s.rSymbolTimeMS.Store(math.Float32bits(lst))
					s.rSymbolRate.Store(uint32(lsr))
					s.rPreambleSymbols.Store(uint32(prs))
					s.rPreambleTimeMS.Store(uint32(prt))
					s.rCSMASlotTimeMS.Store(uint32(cst))
					if cmdBuf.Len() == 12 {
						s.rCSMADIFSMS.Store(uint32(difs))
					}
				}
				cmdBuf.Reset()
			}

		case CMD_STAT_RSSI:
			if ifc := d.getSub(byte(d.selectedIdx.Load())); ifc != nil && ifc.rnodeSub != nil {
				ifc.rnodeSub.rRSSI.Store(int32(int(b) - RSSI_OFFSET))
			}

		case CMD_STAT_SNR:
			if ifc := d.getSub(byte(d.selectedIdx.Load())); ifc != nil && ifc.rnodeSub != nil {
				snr := float32(int8(b)) * 0.25
				ifc.rnodeSub.rSNR.Store(math.Float32bits(snr))
				ifc.rnodeSub.computeQuality()
			}

		case CMD_ST_ALOCK:
			kissUnescapeStream(&cmdBuf, b, &escape)
			if cmdBuf.Len() == 2 {
				at := uint16(cmdBuf.Bytes()[0])<<8 | uint16(cmdBuf.Bytes()[1])
				if ifc := d.getSub(byte(d.selectedIdx.Load())); ifc != nil && ifc.rnodeSub != nil {
					ifc.rnodeSub.rSTALock.Store(uint32(at))
				}
				cmdBuf.Reset()
			}

		case CMD_LT_ALOCK:
			kissUnescapeStream(&cmdBuf, b, &escape)
			if cmdBuf.Len() == 2 {
				at := uint16(cmdBuf.Bytes()[0])<<8 | uint16(cmdBuf.Bytes()[1])
				if ifc := d.getSub(byte(d.selectedIdx.Load())); ifc != nil && ifc.rnodeSub != nil {
					ifc.rnodeSub.rLTALock.Store(uint32(at))
				}
				cmdBuf.Reset()
			}

		case CMD_PLATFORM:
			d.platform.Store(uint32(b))
			if b == PlatformESP32 || b == PlatformNRF52 {
				d.display.Store(true)
			}

		case CMD_MCU:
			d.mcu.Store(uint32(b))

		case CMD_RANDOM:
			d.randomByte.Store(uint32(b))

		case CMD_READY:
			d.processQueueAll()

		case CMD_DETECT:
			d.detected.Store(b == DETECT_RESP)

		case cmdInterfaces:
			// Python: buffers 2 bytes, uses the second byte as interface type.
			cmdBuf.WriteByte(b)
			if cmdBuf.Len() == 2 {
				t := cmdBuf.Bytes()[1]
				d.interfaceTypesMu.Lock()
				d.interfaceTypes = append(d.interfaceTypes, interfaceTypeToStr(t))
				d.interfaceTypesMu.Unlock()
				cmdBuf.Reset()
			}

		case CMD_ERROR:
			if DiagLogf != nil {
				DiagLogf(LogError, "%s hw error 0x%02X", d.parent, b)
			}
			return

		case CMD_RESET:
			// Python: if 0xF8 and ESP32 -> reinit path.
			if b == 0xF8 && d.platform.Load() == PlatformESP32 {
				return
			}

		default:
			// Ignore unknown commands.
		}

		// Timeout behaviour like Python: clear partially received frames.
		if (dataBuf.Len() > 0 || cmdBuf.Len() > 0) && (time.Now().UnixMilli()-lastReadMS) > 100 {
			dataBuf.Reset()
			cmdBuf.Reset()
			inFrame = false
			command = CMD_UNKNOWN
			escape = false
		}
	}
}

func (d *RNodeMultiDriver) reconnectLoop() {
	if d == nil {
		return
	}
	if !d.reconnecting.CompareAndSwap(false, true) {
		return
	}
	defer d.reconnecting.Store(false)

	for {
		select {
		case <-d.stopCh:
			return
		default:
		}

		time.Sleep(5 * time.Second)

		if DiagLogf != nil {
			DiagLogf(LogVerbose, "Attempting to reconnect serial port %s for %s...", d.cfg.Port, d.parent)
		}

		d.closePort()
		if err := d.openPort(); err != nil {
			if DiagLogf != nil {
				DiagLogf(LogError, "Error while reconnecting port, the contained exception was: %v", err)
			}
			continue
		}

		time.Sleep(2 * time.Second)
		d.parent.Online = true
		go d.readLoop()

		_ = d.detect()

		if DiagLogf != nil {
			DiagLogf(LogInfo, "Reconnected serial port for %s", d.parent)
		}
		return
	}
}

// -------- Subinterface driver --------

type RNodeSubDriver struct {
	parent *RNodeMultiDriver
	iface  *Interface

	name   string
	index  byte
	ifType string

	flowControl bool
	ready       atomic.Bool

	frequency uint32
	bandwidth uint32
	txpower   int8
	sf        byte
	cr        byte

	stalock *float64
	ltalock *float64

	rFrequency atomic.Uint32
	rBandwidth atomic.Uint32
	rTXPower   atomic.Int32
	rSF        atomic.Uint32
	rCR        atomic.Uint32
	rState     atomic.Uint32
	rLock      atomic.Uint32
	repFreq    atomic.Bool
	repBW      atomic.Bool
	repTXP     atomic.Bool
	repSF      atomic.Bool
	repState   atomic.Bool

	desiredState atomic.Uint32

	rRSSI    atomic.Int32
	rSNR     atomic.Uint32
	rQuality atomic.Uint32 // float32bits (percent)
	rSTALock atomic.Uint32
	rLTALock atomic.Uint32

	rSymbolTimeMS    atomic.Uint32 // float32bits
	rSymbolRate      atomic.Uint32
	rPreambleSymbols atomic.Uint32
	rPreambleTimeMS  atomic.Uint32
	rCSMASlotTimeMS  atomic.Uint32
	rCSMADIFSMS      atomic.Uint32

	qMu   sync.Mutex
	queue [][]byte

	bitrate atomic.Uint64 // float64bits
}

func (d *RNodeSubDriver) configure() error {
	if d.parent == nil || d.iface == nil {
		return errors.New("missing parent")
	}
	if DiagLogf != nil {
		DiagLogf(LogVerbose, "Configuring RNode subinterface %s...", d.iface)
	}

	// Reset reported values.
	d.rFrequency.Store(0)
	d.rBandwidth.Store(0)
	d.rTXPower.Store(0)
	d.rSF.Store(0)
	d.rCR.Store(0)
	d.rState.Store(0)
	d.rLock.Store(0)
	d.repFreq.Store(false)
	d.repBW.Store(false)
	d.repTXP.Store(false)
	d.repSF.Store(false)
	d.repState.Store(false)
	d.desiredState.Store(0)

	// Python: sleep(2.0) before initRadio.
	time.Sleep(2 * time.Second)

	if err := d.parent.setU32(CMD_FREQUENCY, d.frequency, d.index); err != nil {
		return err
	}
	if err := d.parent.setU32(CMD_BANDWIDTH, d.bandwidth, d.index); err != nil {
		return err
	}
	if err := d.parent.setI8(CMD_TXPOWER, d.txpower, d.index); err != nil {
		return err
	}
	if err := d.parent.setU8(CMD_SF, d.sf, d.index); err != nil {
		return err
	}
	if err := d.parent.setU8(CMD_CR, d.cr, d.index); err != nil {
		return err
	}
	if d.stalock != nil {
		if err := d.parent.setAirtime(CMD_ST_ALOCK, *d.stalock, d.index); err != nil {
			return err
		}
	}
	if d.ltalock != nil {
		if err := d.parent.setAirtime(CMD_LT_ALOCK, *d.ltalock, d.index); err != nil {
			return err
		}
	}
	d.desiredState.Store(uint32(RADIO_STATE_ON))
	if err := d.parent.setU8(CMD_RADIO_STATE, RADIO_STATE_ON, d.index); err != nil {
		return err
	}

	// Validate radio state (best-effort like Python).
	time.Sleep(250 * time.Millisecond)

	if err := d.validateReported(); err != nil {
		if DiagLogf != nil {
			DiagLogf(LogError, "After configuring %s, reported radio parameters did not match configuration: %v", d.iface, err)
		}
		d.iface.Online = false
		return err
	}

	d.ready.Store(true)
	d.iface.Online = true
	return nil
}

func (d *RNodeSubDriver) validateReported() error {
	// Python treats missing reported values (None) as mismatches.
	// We do the same via rep* flags.
	if d.frequency != 0 && !d.repFreq.Load() {
		return errors.New("frequency report missing")
	}
	if d.bandwidth != 0 && !d.repBW.Load() {
		return errors.New("bandwidth report missing")
	}
	if !d.repTXP.Load() {
		return errors.New("txpower report missing")
	}
	if d.sf != 0 && !d.repSF.Load() {
		return errors.New("spreading factor report missing")
	}
	if want := d.desiredState.Load(); want != 0 && !d.repState.Load() {
		return errors.New("radio state report missing")
	}

	if rf := d.rFrequency.Load(); d.repFreq.Load() && d.frequency != 0 {
		diff := int64(rf) - int64(d.frequency)
		if diff < 0 {
			diff = -diff
		}
		if diff > 100 {
			return fmt.Errorf("frequency mismatch %d vs %d", d.frequency, rf)
		}
	}
	if rbw := d.rBandwidth.Load(); d.repBW.Load() && d.bandwidth != 0 && rbw != d.bandwidth {
		return fmt.Errorf("bandwidth mismatch %d vs %d", d.bandwidth, rbw)
	}
	if rtx := d.rTXPower.Load(); d.repTXP.Load() && rtx != int32(d.txpower) {
		return fmt.Errorf("txpower mismatch %d vs %d", d.txpower, rtx)
	}
	if rsf := d.rSF.Load(); d.repSF.Load() && d.sf != 0 && byte(rsf) != d.sf {
		return fmt.Errorf("sf mismatch %d vs %d", d.sf, byte(rsf))
	}
	if want := d.desiredState.Load(); want != 0 && d.repState.Load() {
		if got := d.rState.Load(); got != want {
			return fmt.Errorf("radio state mismatch %d vs %d", want, got)
		}
	}
	return nil
}

func (d *RNodeSubDriver) processIncoming(data []byte) {
	if d == nil || d.iface == nil || len(data) == 0 {
		return
	}
	cp := append([]byte(nil), data...)
	atomic.AddUint64(&d.iface.RXB, uint64(len(cp)))
	if parent := d.iface.Parent; parent != nil {
		atomic.AddUint64(&parent.RXB, uint64(len(cp)))
	}
	if InboundHandler != nil {
		InboundHandler(cp, d.iface)
	}

	// Python clears transient stat fields on receive.
	d.rRSSI.Store(0)
	d.rSNR.Store(0)
}

func (d *RNodeSubDriver) enqueue(data []byte) {
	d.qMu.Lock()
	d.queue = append(d.queue, append([]byte(nil), data...))
	d.qMu.Unlock()
}

func (d *RNodeSubDriver) processQueue() {
	d.qMu.Lock()
	if len(d.queue) == 0 {
		d.qMu.Unlock()
		d.ready.Store(true)
		return
	}
	data := d.queue[0]
	d.queue = d.queue[1:]
	d.qMu.Unlock()
	d.ready.Store(true)
	d.iface.ProcessOutgoing(data)
}

func (d *RNodeSubDriver) updateBitrateFromReported() {
	sf := float64(d.rSF.Load())
	cr := float64(d.rCR.Load())
	bw := float64(d.rBandwidth.Load())
	if sf == 0 || cr == 0 || bw == 0 {
		return
	}
	rate := sf * ((4.0 / cr) / (math.Pow(2, sf) / (bw / 1000.0))) * 1000.0
	if rate <= 0 {
		return
	}
	d.bitrate.Store(math.Float64bits(rate))
	d.iface.Bitrate = int(rate) // best-effort for MTU heuristics.
	if DiagLogf != nil {
		DiagLogf(LogVerbose, "%s on-air bitrate is now %.2f kbps", d.iface, rate/1000.0)
	}
}

func (d *RNodeSubDriver) computeQuality() {
	snrBits := d.rSNR.Load()
	if snrBits == 0 {
		d.rQuality.Store(0)
		return
	}
	snr := math.Float32frombits(snrBits)

	// Match Python quality metric.
	sf := float32(d.rSF.Load())
	sfs := sf - 7
	qMin := float32(QSNRMinBase) - sfs*float32(QSNRStep)
	qMax := float32(QSNRMax)
	span := qMax - qMin
	if span <= 0 {
		d.rQuality.Store(0)
		return
	}
	quality := ((snr - qMin) / span) * 100
	if quality > 100 {
		quality = 100
	}
	if quality < 0 {
		quality = 0
	}
	d.rQuality.Store(math.Float32bits(quality))
}
