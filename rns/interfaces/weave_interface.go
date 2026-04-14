package interfaces

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	serial "go.bug.st/serial"
)

const (
	weaveDefaultHWMTU     = 1024
	weaveBitrateGuess     = 250_000
	weavePeerTimeout      = 20 * time.Second
	weavePeerJobInterval  = time.Duration(float64(weavePeerTimeout) * 1.1)
	weaveMultiIFDequeLen  = 48
	weaveMultiIFDequeTTL  = time.Duration(float64(time.Second) * 0.75)
	weaveEndpointQueueLen = 1024
	weaveStatUpdateMin    = 500 * time.Millisecond
)

const (
	wdclTSDiscover    = 0x00
	wdclTSConnect     = 0x01
	wdclTSCmd         = 0x02
	wdclTSLog         = 0x03
	wdclTSDisp        = 0x04
	wdclTSEndpointPkt = 0x05
	wdclTSEncapProto  = 0x06

	wdclHeaderMinSize = 4 + 1
	wdclSwitchIDLen   = 4
	wdclEndpointIDLen = 8
	wdclPubKeySize    = ed25519.PublicKeySize
	wdclSignatureSize = ed25519.SignatureSize

	wdclCmdEndpointPkt = 0x0001
	wdclCmdEndpoints   = 0x0100
	wdclCmdRemoteInput = 0x0A01

	weaveHandshakeTimeout = 2 * time.Second
)

var wdclBroadcast = [4]byte{0xFF, 0xFF, 0xFF, 0xFF}

const (
	wdclCmdRemoteDisplay uint16 = 0x0A00

	// Event IDs mirror Evt.* from Python weaveInterface.py.
	evtMsg                 WeaveEventID = 0x0000
	evtSystemBoot          WeaveEventID = 0x0001
	evtCoreInit            WeaveEventID = 0x0002
	evtDrvUARTInit         WeaveEventID = 0x1000
	evtDrvUSBCDCInit       WeaveEventID = 0x1010
	evtDrvUSBCDCAvail      WeaveEventID = 0x1011
	evtDrvUSBCDCSPnd       WeaveEventID = 0x1012
	evtDrvUSBCDCRes        WeaveEventID = 0x1013
	evtDrvUSBCDCConn       WeaveEventID = 0x1014
	evtDrvUSBCDCReadEr     WeaveEventID = 0x1015
	evtDrvUSBCDCOvf        WeaveEventID = 0x1016
	evtDrvUSBCDCDrop       WeaveEventID = 0x1017
	evtDrvUSBCDCTxTo       WeaveEventID = 0x1018
	evtDrvI2CInit          WeaveEventID = 0x1020
	evtDrvNVSInit          WeaveEventID = 0x1030
	evtDrvNVSErase         WeaveEventID = 0x1031
	evtDrvCryptoInit       WeaveEventID = 0x1040
	evtDrvDisplayInit      WeaveEventID = 0x1050
	evtDrvDisplayBus       WeaveEventID = 0x1051
	evtDrvDisplayIO        WeaveEventID = 0x1052
	evtDrvDisplayPanel     WeaveEventID = 0x1053
	evtDrvDisplayRst       WeaveEventID = 0x1054
	evtDrvDisplayInit2     WeaveEventID = 0x1055
	evtDrvDisplayEn        WeaveEventID = 0x1056
	evtDrvDisplayRemEn     WeaveEventID = 0x1057
	evtDrvW80211Init       WeaveEventID = 0x1060
	evtDrvW80211Init2      WeaveEventID = 0x1061
	evtDrvW80211Chan       WeaveEventID = 0x1062
	evtDrvW80211Power      WeaveEventID = 0x1063
	evtKrnLoggerInit       WeaveEventID = 0x2000
	evtKrnLoggerOut        WeaveEventID = 0x2001
	evtKrnUIInit           WeaveEventID = 0x2010
	evtProtoWDCLInit       WeaveEventID = 0x3000
	evtProtoWDCLRun        WeaveEventID = 0x3001
	evtProtoWDCLConnection WeaveEventID = 0x3002
	evtProtoWDCLHostEP     WeaveEventID = 0x3003
	evtProtoWeaveInit      WeaveEventID = 0x3100
	evtProtoWeaveRun       WeaveEventID = 0x3101
	evtProtoWeaveEPAlive   WeaveEventID = 0x3102
	evtProtoWeaveEPTimeout WeaveEventID = 0x3103
	evtProtoWeaveEPVia     WeaveEventID = 0x3104
	evtSrvctlRemoteDisplay WeaveEventID = 0xA000
	evtInterfaceRegistered WeaveEventID = 0xD000
	evtStatState           WeaveEventID = 0xE000
	evtStatUptime          WeaveEventID = 0xE001
	evtStatTimebase        WeaveEventID = 0xE002
	evtStatCPU             WeaveEventID = 0xE003
	evtStatTaskCPU         WeaveEventID = 0xE004
	evtStatMemory          WeaveEventID = 0xE005
	evtStatStorage         WeaveEventID = 0xE006
	evtSyserrMemExhausted  WeaveEventID = 0xF000
)

var weaveEventDescriptions = map[WeaveEventID]string{
	evtSystemBoot:          "System boot",
	evtCoreInit:            "Core initialization",
	evtDrvUARTInit:         "UART driver initialization",
	evtDrvUSBCDCInit:       "USB CDC driver initialization",
	evtDrvUSBCDCAvail:      "USB CDC host became available",
	evtDrvUSBCDCSPnd:       "USB CDC host suspend",
	evtDrvUSBCDCRes:        "USB CDC host resume",
	evtDrvUSBCDCConn:       "USB CDC host connection",
	evtDrvUSBCDCReadEr:     "USB CDC read error",
	evtDrvUSBCDCOvf:        "USB CDC overflow occurred",
	evtDrvUSBCDCDrop:       "USB CDC dropped bytes",
	evtDrvUSBCDCTxTo:       "USB CDC TX flush timeout",
	evtDrvI2CInit:          "I2C driver initialization",
	evtDrvNVSInit:          "NVS driver initialization",
	evtDrvCryptoInit:       "Cryptography driver initialization",
	evtDrvW80211Init:       "W802.11 driver initialization",
	evtDrvW80211Init2:      "W802.11 driver initialization",
	evtDrvW80211Chan:       "W802.11 channel configuration",
	evtDrvW80211Power:      "W802.11 TX power configuration",
	evtDrvDisplayInit:      "Display driver initialization",
	evtDrvDisplayBus:       "Display bus availability",
	evtDrvDisplayIO:        "Display I/O configuration",
	evtDrvDisplayPanel:     "Display panel allocation",
	evtDrvDisplayRst:       "Display panel reset",
	evtDrvDisplayInit2:     "Display panel initialization",
	evtDrvDisplayEn:        "Display panel activation",
	evtDrvDisplayRemEn:     "Remote display output activation",
	evtKrnLoggerInit:       "Logging service initialization",
	evtKrnLoggerOut:        "Logging service output activation",
	evtKrnUIInit:           "User interface service initialization",
	evtProtoWDCLInit:       "WDCL protocol initialization",
	evtProtoWDCLRun:        "WDCL protocol activation",
	evtProtoWDCLConnection: "WDCL host connection",
	evtProtoWDCLHostEP:     "Weave host endpoint",
	evtProtoWeaveInit:      "Weave protocol initialization",
	evtProtoWeaveRun:       "Weave protocol activation",
	evtProtoWeaveEPAlive:   "Weave endpoint alive",
	evtProtoWeaveEPTimeout: "Weave endpoint disappeared",
	evtSrvctlRemoteDisplay: "Remote display service control event",
	evtInterfaceRegistered: "Interface registration",
	evtSyserrMemExhausted:  "System memory exhausted",
}

var weaveInterfaceTypes = map[byte]string{
	0x01: "usb",
	0x02: "uart",
	0x03: "mw",
	0x04: "ble",
	0x05: "lora",
	0x06: "eth",
	0x07: "wifi",
	0x08: "tcp",
	0x09: "udp",
	0x0A: "ir",
	0x0B: "afsk",
	0x0C: "gpio",
	0x0D: "spi",
	0x0E: "i2c",
	0x0F: "can",
	0x10: "dma",
}

var weaveChannelDescriptions = map[byte]string{
	1:  "Channel 1 (2412 MHz)",
	2:  "Channel 2 (2417 MHz)",
	3:  "Channel 3 (2422 MHz)",
	4:  "Channel 4 (2427 MHz)",
	5:  "Channel 5 (2432 MHz)",
	6:  "Channel 6 (2437 MHz)",
	7:  "Channel 7 (2442 MHz)",
	8:  "Channel 8 (2447 MHz)",
	9:  "Channel 9 (2452 MHz)",
	10: "Channel 10 (2457 MHz)",
	11: "Channel 11 (2462 MHz)",
	12: "Channel 12 (2467 MHz)",
	13: "Channel 13 (2472 MHz)",
	14: "Channel 14 (2484 MHz)",
}

var weaveLevelNames = map[WeaveEventLevel]string{
	0: "Forced",
	1: "Critical",
	2: "Error",
	3: "Warning",
	4: "Notice",
	5: "Info",
	6: "Verbose",
	7: "Debug",
	8: "Extreme",
	9: "System",
}

const (
	weaveDisplayWidth  = 128
	weaveDisplayHeight = 64
)

var WeaveRemoteDisplayHandler func(data []byte, width, height int, device *WeaveDevice)

var weaveTaskDescriptions = map[string]string{
	"taskLVGL":       "Driver: UI Renderer",
	"ui_service":     "Service: User Interface",
	"TinyUSB":        "Driver: USB",
	"drv_w80211":     "Driver: W802.11",
	"system_stats":   "System: Stats",
	"core":           "System: Core",
	"protocol_wdcl":  "Protocol: WDCL",
	"protocol_weave": "Protocol: Weave",
	"tiT":            "Protocol: TCP/IP",
	"ipc0":           "System: CPU 0 IPC",
	"ipc1":           "System: CPU 1 IPC",
	"esp_timer":      "Driver: Timers",
	"Tmr Svc":        "Service: Timers",
	"kernel_logger":  "Service: Logging",
	"remote_display": "Service: Remote Display",
	"wifi":           "System: WiFi Hardware",
	"sys_evt":        "System: Kernel Events",
}

type WeaveEventLevel byte
type WeaveEventID uint16

type LogFrame struct {
	Timestamp time.Time
	Uptime    time.Duration
	Level     WeaveEventLevel
	Event     WeaveEventID
	Data      []byte
	Rendered  string
}

type StatEntry struct {
	Timestamp time.Time
	Value     float64
}

type WeaveStatsSeries struct {
	Timestamps []float64 `json:"timestamps"`
	Values     []float64 `json:"values"`
	Max        float64   `json:"max"`
	Unit       string    `json:"unit"`
}

type WeaveTaskStat struct {
	CPULoad    float64 `json:"cpu_load"`
	Timestamp  float64 `json:"timestamp"`
	ReceivedAt time.Time
}

type WeaveEndpoint struct {
	ID        string
	LastSeen  time.Time
	Via       string
	Received  [][]byte
	Available bool
}

type WeaveDevice struct {
	mu            sync.Mutex
	endpoints     map[string]*WeaveEndpoint
	cpuLoad       float64
	cpuStats      []StatEntry
	memoryTotal   uint32
	memoryFree    uint32
	memoryUsed    uint32
	memoryUsedPct float64
	memoryStats   []StatEntry
	logQueue      []LogFrame
	connected     bool
	switchID      string
	endpointID    string
	nextUpdateCPU time.Time
	nextUpdateMem time.Time
	activeTasks   map[string]WeaveTaskStat

	displayMu     sync.Mutex
	displayBuf    []byte
	displaySize   int
	displayOffset int
	displayDirty  bool
}

func NewWeaveDevice() *WeaveDevice {
	return &WeaveDevice{
		endpoints:   make(map[string]*WeaveEndpoint),
		cpuStats:    make([]StatEntry, 0, 64),
		memoryStats: make([]StatEntry, 0, 64),
		logQueue:    make([]LogFrame, 0, 128),
		activeTasks: make(map[string]WeaveTaskStat),
	}
}

func weaveLevelName(level WeaveEventLevel) string {
	if s, ok := weaveLevelNames[level]; ok {
		return s
	}
	return "Unknown"
}

func weaveEventDescription(event WeaveEventID) string {
	if s, ok := weaveEventDescriptions[event]; ok {
		return s
	}
	if event == evtMsg {
		return "Message"
	}
	return fmt.Sprintf("0x%04x", uint16(event))
}

func hexBytes(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return hex.EncodeToString(b)
}

func renderWeaveLogFrame(frame LogFrame) string {
	ts := frame.Uptime
	tsStr := fmt.Sprintf("%.2fs", ts.Seconds())
	levelStr := weaveLevelName(frame.Level)

	if frame.Event == evtMsg {
		msg := ""
		if len(frame.Data) > 0 {
			msg = string(frame.Data)
		}
		return fmt.Sprintf("[%s] [%s]: %s", tsStr, levelStr, msg)
	}

	desc := weaveEventDescription(frame.Event)
	dataString := ""

	switch frame.Event {
	case evtInterfaceRegistered:
		if len(frame.Data) >= 2 {
			ifIndex := frame.Data[0]
			ifType := frame.Data[1]
			typeName := "phy"
			if t, ok := weaveInterfaceTypes[ifType]; ok {
				typeName = t
			}
			dataString = fmt.Sprintf(": %s%d", typeName, ifIndex)
		}
	case evtDrvUSBCDCConn:
		if len(frame.Data) >= 1 {
			switch frame.Data[0] {
			case 0x01:
				dataString = ": Connected"
			case 0x00:
				dataString = ": Disconnected"
			}
		}
	case evtDrvW80211Chan:
		if len(frame.Data) >= 1 {
			if s, ok := weaveChannelDescriptions[frame.Data[0]]; ok {
				dataString = ": " + s
			} else {
				dataString = ": " + hexBytes(frame.Data)
			}
		}
	case evtDrvW80211Power:
		if len(frame.Data) >= 1 {
			txPower := float64(frame.Data[0]) * 0.25
			mw := int(math.Pow(10, txPower/10.0))
			dataString = fmt.Sprintf(": %.2f dBm (%d mW)", txPower, mw)
		}
	default:
		// Core/proto init range: show success/failure if data[0] present.
		if frame.Event >= evtCoreInit && frame.Event <= evtProtoWeaveRun && len(frame.Data) >= 1 {
			switch frame.Data[0] {
			case 0x01:
				dataString = ": Success"
			case 0x00:
				if frame.Level == 2 { // LOG_ERROR in Python
					dataString = ": Failure"
				} else {
					dataString = ": Stopped"
				}
			default:
				dataString = ": " + hexBytes(frame.Data)
			}
		} else if len(frame.Data) > 0 {
			dataString = ": " + hexBytes(frame.Data)
		}
	}

	return fmt.Sprintf("[%s] [%s] [%s]%s", tsStr, levelStr, desc, dataString)
}

func (d *WeaveDevice) EndpointAlive(endpoint string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	ep, ok := d.endpoints[endpoint]
	if !ok {
		ep = &WeaveEndpoint{
			ID:       endpoint,
			Received: make([][]byte, 0, 4),
		}
		d.endpoints[endpoint] = ep
	}
	ep.LastSeen = time.Now()
	ep.Available = true
}

func (d *WeaveDevice) EndpointVia(endpoint, via string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if ep, ok := d.endpoints[endpoint]; ok {
		ep.Via = via
	}
}

func (d *WeaveDevice) EndpointTimeout(endpoint string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if ep, ok := d.endpoints[endpoint]; ok {
		ep.Available = false
	}
}

func (d *WeaveDevice) ReceiveFrame(endpoint string, payload []byte) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if ep, ok := d.endpoints[endpoint]; ok {
		ep.Received = append(ep.Received, append([]byte(nil), payload...))
		if len(ep.Received) > weaveEndpointQueueLen {
			start := len(ep.Received) - weaveEndpointQueueLen
			ep.Received = ep.Received[start:]
		}
	}
}

func (d *WeaveDevice) CaptureCPU(value float64) {
	now := time.Now()
	if now.Before(d.nextUpdateCPU) {
		return
	}
	d.nextUpdateCPU = now.Add(weaveStatUpdateMin)
	d.mu.Lock()
	d.cpuLoad = value
	d.cpuStats = append(d.cpuStats, StatEntry{Timestamp: now, Value: value})
	if len(d.cpuStats) > 256 {
		d.cpuStats = d.cpuStats[len(d.cpuStats)-256:]
	}
	d.mu.Unlock()
}

func (d *WeaveDevice) CaptureMemory(free, total uint32) {
	now := time.Now()
	if now.Before(d.nextUpdateMem) {
		return
	}
	d.nextUpdateMem = now.Add(weaveStatUpdateMin)
	d.mu.Lock()
	d.memoryFree = free
	d.memoryTotal = total
	if total > 0 && free <= total {
		d.memoryUsed = total - free
		d.memoryUsedPct = (float64(d.memoryUsed) / float64(total)) * 100.0
		d.memoryStats = append(d.memoryStats, StatEntry{Timestamp: now, Value: float64(d.memoryUsed)})
	}
	if len(d.memoryStats) > 256 {
		d.memoryStats = d.memoryStats[len(d.memoryStats)-256:]
	}
	d.mu.Unlock()
}

func (d *WeaveDevice) CaptureTask(name string, load float64) {
	if name == "" {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.activeTasks == nil {
		d.activeTasks = make(map[string]WeaveTaskStat)
	}
	now := time.Now()
	d.activeTasks[name] = WeaveTaskStat{
		CPULoad:    load,
		Timestamp:  float64(now.UnixNano()) / 1e9,
		ReceivedAt: now,
	}
}

func (d *WeaveDevice) ActiveTasks(maxAge time.Duration) map[string]WeaveTaskStat {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.activeTasks) == 0 {
		return nil
	}
	now := time.Now()
	kept := make(map[string]WeaveTaskStat, len(d.activeTasks))
	for name, entry := range d.activeTasks {
		if name == "" || strings.HasPrefix(name, "IDLE") {
			continue
		}
		if maxAge > 0 && now.Sub(entry.ReceivedAt) > maxAge {
			continue
		}
		kept[name] = WeaveTaskStat{CPULoad: entry.CPULoad, Timestamp: entry.Timestamp}
	}
	if len(kept) == 0 {
		return nil
	}
	return kept
}

func (d *WeaveDevice) LogHistory() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.logQueue) == 0 {
		return nil
	}
	out := make([]string, len(d.logQueue))
	for i, frame := range d.logQueue {
		out[i] = frame.Rendered
	}
	return out
}

func (d *WeaveDevice) CPUHistory() []StatEntry {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.cpuStats) == 0 {
		return nil
	}
	out := make([]StatEntry, len(d.cpuStats))
	copy(out, d.cpuStats)
	return out
}

func (d *WeaveDevice) MemoryHistory() []StatEntry {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.memoryStats) == 0 {
		return nil
	}
	out := make([]StatEntry, len(d.memoryStats))
	copy(out, d.memoryStats)
	return out
}

func (d *WeaveDevice) CPUStatsUI() *WeaveStatsSeries {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.cpuStats) == 0 {
		return nil
	}
	tbegin := d.cpuStats[len(d.cpuStats)-1].Timestamp
	out := &WeaveStatsSeries{
		Timestamps: make([]float64, 0, len(d.cpuStats)),
		Values:     make([]float64, 0, len(d.cpuStats)),
		Max:        100,
		Unit:       "%",
	}
	for _, entry := range d.cpuStats {
		out.Timestamps = append(out.Timestamps, entry.Timestamp.Sub(tbegin).Seconds())
		out.Values = append(out.Values, entry.Value)
	}
	return out
}

func (d *WeaveDevice) MemoryStatsUI() *WeaveStatsSeries {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.memoryStats) == 0 || d.memoryTotal == 0 {
		return nil
	}
	tbegin := d.memoryStats[len(d.memoryStats)-1].Timestamp
	out := &WeaveStatsSeries{
		Timestamps: make([]float64, 0, len(d.memoryStats)),
		Values:     make([]float64, 0, len(d.memoryStats)),
		Max:        float64(d.memoryTotal),
		Unit:       "B",
	}
	for _, entry := range d.memoryStats {
		out.Timestamps = append(out.Timestamps, entry.Timestamp.Sub(tbegin).Seconds())
		out.Values = append(out.Values, entry.Value)
	}
	return out
}

func (d *WeaveDevice) EndpointQueues() map[string][][]byte {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.endpoints) == 0 {
		return nil
	}
	out := make(map[string][][]byte, len(d.endpoints))
	for id, ep := range d.endpoints {
		if len(ep.Received) == 0 {
			continue
		}
		queue := make([][]byte, len(ep.Received))
		for idx, pkt := range ep.Received {
			queue[idx] = append([]byte(nil), pkt...)
		}
		out[id] = queue
	}
	return out
}

func (d *WeaveDevice) EnqueueLog(frame LogFrame) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.logQueue = append(d.logQueue, frame)
	if len(d.logQueue) > 256 {
		d.logQueue = d.logQueue[len(d.logQueue)-256:]
	}
}

func (d *WeaveDevice) UpdateDisplayChunk(offset int, total int, chunk []byte) {
	if total <= 0 || offset < 0 || len(chunk) == 0 {
		return
	}
	if offset+len(chunk) > total {
		return
	}
	d.displayMu.Lock()
	defer d.displayMu.Unlock()
	if total != d.displaySize || len(d.displayBuf) < total {
		d.displayBuf = make([]byte, total)
		d.displaySize = total
		d.displayOffset = 0
	}
	copy(d.displayBuf[offset:offset+len(chunk)], chunk)
	d.displayOffset = offset + len(chunk)
	d.displayDirty = true
}

func (d *WeaveDevice) DisplayComplete() bool {
	d.displayMu.Lock()
	defer d.displayMu.Unlock()
	return d.displayDirty && d.displaySize > 0 && d.displayOffset >= d.displaySize
}

func (d *WeaveDevice) ResetDisplay() {
	d.displayMu.Lock()
	defer d.displayMu.Unlock()
	d.displayBuf = nil
	d.displaySize = 0
	d.displayOffset = 0
	d.displayDirty = false
}

func (d *WeaveDevice) DisplaySnapshot() ([]byte, bool) {
	d.displayMu.Lock()
	defer d.displayMu.Unlock()
	if !d.displayDirty || d.displaySize == 0 || d.displayOffset < d.displaySize {
		return nil, false
	}
	buf := append([]byte(nil), d.displayBuf...)
	d.displayDirty = false
	return buf, true
}

type weaveConfig struct {
	name    string
	port    string
	bitrate int
}

type weavePeerState struct {
	endpoint  string
	iface     *Interface
	lastHeard time.Time
}

type weaveSeen struct {
	sum [32]byte
	ts  time.Time
}

type WeaveInterfaceDriver struct {
	iface *Interface
	cfg   weaveConfig

	mu       sync.Mutex
	peers    map[string]*weavePeerState
	mifDeque []weaveSeen
	mifIdx   int

	stopCh chan struct{}
	device *WeaveDevice

	serialMu     sync.Mutex
	serialPort   serial.Port
	serialOnline atomic.Bool
	writeMu      sync.Mutex

	localSigPub []byte
	localSign   func(msg []byte) ([]byte, error)
	localSID    [wdclSwitchIDLen]byte

	remoteMu      sync.Mutex
	remotePub     ed25519.PublicKey
	remoteSID     [wdclSwitchIDLen]byte
	remoteKnown   bool
	wdclOK        atomic.Bool
	remoteDisplay atomic.Bool
	reconnecting  atomic.Bool
}

// WeaveEncapProtoHandler is an optional hook for handling WDCL_T_ENCAP_PROTO frames.
// The Python driver defines this frame type but does not currently process it.
// If set, it will be invoked with the proto byte and payload, and the parent interface.
var WeaveEncapProtoHandler func(proto byte, payload []byte, ifc *Interface)

// NewWeaveInterface implements the placeholder high-level structure of the Python
// WeaveInterface so the Go port can track peers and dedup inbound frames while the
// actual WDCL transport remains to be implemented.
func NewWeaveInterface(name string, kv map[string]string) (*Interface, error) {
	cfg, err := parseWeaveConfig(name, kv)
	if err != nil {
		return nil, err
	}

	iface := &Interface{
		Name:              name,
		Type:              "WeaveInterface",
		IN:                true,
		OUT:               false,
		DriverImplemented: true,
		Bitrate:           cfg.bitrate,
		HWMTU:             weaveDefaultHWMTU,
		FixedMTU:          true,
		IngressControl:    false, // Python: should_ingress_limit() always false
		Created:           time.Now(),
	}
	// Match Python default.
	if iface.IFACSize == 0 {
		iface.IFACSize = 16
	}

	driver := &WeaveInterfaceDriver{
		iface:    iface,
		cfg:      cfg,
		peers:    make(map[string]*weavePeerState),
		mifDeque: make([]weaveSeen, weaveMultiIFDequeLen),
		stopCh:   make(chan struct{}),
		device:   NewWeaveDevice(),
	}

	if err := driver.initLocalIdentity(); err != nil {
		return nil, err
	}

	iface.weave = driver
	go driver.peerJobsLoop()
	go driver.connectLoop()
	return iface, nil
}

func parseWeaveConfig(name string, kv map[string]string) (weaveConfig, error) {
	cfg := weaveConfig{name: name}
	cfg.port = first(kv, "port")
	if strings.TrimSpace(cfg.port) == "" {
		return cfg, errors.New("WeaveInterface requires port configuration")
	}
	if v, ok := parseInt(first(kv, "configured_bitrate")); ok && v > 0 {
		cfg.bitrate = v
	} else {
		cfg.bitrate = weaveBitrateGuess
	}
	return cfg, nil
}

func (d *WeaveInterfaceDriver) initLocalIdentity() error {
	if WeaveIdentityProvider != nil {
		sigPub, sign, err := WeaveIdentityProvider(d.cfg.port)
		if err != nil {
			return err
		}
		if len(sigPub) != wdclPubKeySize || sign == nil {
			return errors.New("invalid WeaveIdentityProvider result")
		}
		d.localSigPub = append([]byte(nil), sigPub...)
		d.localSign = sign
		copy(d.localSID[:], sigPub[len(sigPub)-wdclSwitchIDLen:])
		d.device.switchID = hex.EncodeToString(d.localSID[:])
		return nil
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	d.localSigPub = append([]byte(nil), pub...)
	d.localSign = func(msg []byte) ([]byte, error) {
		return ed25519.Sign(priv, msg), nil
	}
	copy(d.localSID[:], pub[len(pub)-wdclSwitchIDLen:])
	d.device.switchID = hex.EncodeToString(d.localSID[:])
	return nil
}

func (d *WeaveInterfaceDriver) peerJobsLoop() {
	ticker := time.NewTicker(weavePeerJobInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			d.cleanupPeers()
		case <-d.stopCh:
			return
		}
	}
}

func (d *WeaveInterfaceDriver) cleanupPeers() {
	threshold := time.Now().Add(-weavePeerTimeout)
	var stale []*weavePeerState
	d.mu.Lock()
	for key, peer := range d.peers {
		if peer.lastHeard.Before(threshold) {
			stale = append(stale, peer)
			delete(d.peers, key)
		}
	}
	d.mu.Unlock()
	for _, peer := range stale {
		if d.device != nil {
			d.device.EndpointTimeout(peer.endpoint)
		}
		removeInterface(peer.iface)
	}
}

func (d *WeaveInterfaceDriver) Close() {
	select {
	case <-d.stopCh:
		return
	default:
		close(d.stopCh)
	}
	if d.device != nil {
		d.device.connected = false
	}
	d.closeSerial()
}

func (d *WeaveInterfaceDriver) ProcessIncoming(endpoint, data []byte) {
	if d == nil || len(endpoint) == 0 || len(data) == 0 {
		return
	}
	sum := sha256.Sum256(data)
	if d.seenRecently(sum) {
		return
	}
	d.markSeen(sum)

	peer := d.addOrRefreshPeer(endpoint)
	if peer == nil {
		return
	}
	key := encodeEndpoint(endpoint)
	d.device.EndpointAlive(key)
	d.device.ReceiveFrame(key, data)
	atomic.AddUint64(&peer.RXB, uint64(len(data)))
	if parent := peer.Parent; parent != nil {
		atomic.AddUint64(&parent.RXB, uint64(len(data)))
	}
	if InboundHandler != nil {
		InboundHandler(append([]byte(nil), data...), peer)
	}
}

func (d *WeaveInterfaceDriver) ProcessOutgoing(peer *Interface, data []byte) {
	if peer == nil || len(data) == 0 {
		return
	}
	atomic.AddUint64(&peer.TXB, uint64(len(data)))
	if parent := peer.Parent; parent != nil {
		atomic.AddUint64(&parent.TXB, uint64(len(data)))
	}
	endpointHex := strings.TrimPrefix(peer.Name, "WeaveInterfacePeer[")
	endpointHex = strings.TrimSuffix(endpointHex, "]")
	endpoint, err := hex.DecodeString(endpointHex)
	if err != nil || len(endpoint) != wdclEndpointIDLen {
		return
	}
	d.sendEndpointPacket(endpoint, data)
}

func (d *WeaveInterfaceDriver) addOrRefreshPeer(endpoint []byte) *Interface {
	key := encodeEndpoint(endpoint)
	now := time.Now()
	d.mu.Lock()
	defer d.mu.Unlock()
	if existing, ok := d.peers[key]; ok {
		existing.lastHeard = now
		return existing.iface
	}

	peer := &Interface{
		Name:              fmt.Sprintf("WeaveInterfacePeer[%s]", key),
		Type:              "WeaveInterfacePeer",
		Parent:            d.iface,
		DriverImplemented: true,
		Bitrate:           d.iface.Bitrate,
		HWMTU:             d.iface.HWMTU,
		FixedMTU:          d.iface.FixedMTU,
		Created:           now,
		Online:            true,
	}
	peer.IN = d.iface.IN
	peer.OUT = d.iface.OUT
	peer.Mode = d.iface.Mode
	peer.IngressControl = d.iface.IngressControl
	peer.ICMaxHeldAnnounces = d.iface.ICMaxHeldAnnounces
	peer.ICBurstHold = d.iface.ICBurstHold
	peer.ICBurstFreqNew = d.iface.ICBurstFreqNew
	peer.ICBurstFreq = d.iface.ICBurstFreq
	peer.ICNewTime = d.iface.ICNewTime
	peer.ICBurstPenalty = d.iface.ICBurstPenalty
	peer.ICHeldReleaseInterval = d.iface.ICHeldReleaseInterval
	peer.AnnounceCap = d.iface.AnnounceCap
	peer.IFACSize = d.iface.IFACSize
	peer.IFACKey = d.iface.IFACKey
	peer.IFACIdentity = d.iface.IFACIdentity
	peer.IFACSignature = d.iface.IFACSignature
	peer.IFACNetnameVal = d.iface.IFACNetnameVal
	peer.IFACNetkeyVal = d.iface.IFACNetkeyVal
	peer.AnnounceRateTarget = d.iface.AnnounceRateTarget
	peer.AnnounceRateGrace = d.iface.AnnounceRateGrace
	peer.AnnounceRatePenalty = d.iface.AnnounceRatePenalty

	if SpawnHandler != nil {
		SpawnHandler(peer)
	}
	d.peers[key] = &weavePeerState{
		endpoint:  key,
		iface:     peer,
		lastHeard: now,
	}
	return peer
}

func (d *WeaveInterfaceDriver) seenRecently(sum [32]byte) bool {
	if len(d.mifDeque) == 0 {
		return false
	}
	now := time.Now()
	for _, entry := range d.mifDeque {
		if entry.ts.IsZero() {
			continue
		}
		if entry.sum == sum && now.Sub(entry.ts) <= weaveMultiIFDequeTTL {
			return true
		}
	}
	return false
}

func (d *WeaveInterfaceDriver) markSeen(sum [32]byte) {
	if len(d.mifDeque) == 0 {
		return
	}
	d.mifDeque[d.mifIdx] = weaveSeen{sum: sum, ts: time.Now()}
	d.mifIdx++
	if d.mifIdx >= len(d.mifDeque) {
		d.mifIdx = 0
	}
}

func encodeEndpoint(endpoint []byte) string {
	return hex.EncodeToString(endpoint)
}

func (d *WeaveInterfaceDriver) connectLoop() {
	for {
		select {
		case <-d.stopCh:
			return
		default:
		}
		if d.serialOnline.Load() {
			time.Sleep(250 * time.Millisecond)
			continue
		}
		if d.reconnecting.Load() {
			time.Sleep(5 * time.Second)
			continue
		}
		d.reconnecting.Store(true)
		if DiagLogf != nil {
			DiagLogf(LogDebug, "Attempting to open serial port %s...", d.cfg.port)
		}
		_ = d.openSerial()
		d.reconnecting.Store(false)
		time.Sleep(5 * time.Second)
	}
}

func (d *WeaveInterfaceDriver) openSerial() error {
	if strings.TrimSpace(d.cfg.port) == "" {
		return errors.New("missing serial port")
	}
	mode := &serial.Mode{
		BaudRate: 3_000_000,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}
	p, err := serial.Open(d.cfg.port, mode)
	if err != nil {
		if DiagLogf != nil {
			DiagLogf(LogError, "Could not open serial port %s: %v", d.cfg.port, err)
		}
		return err
	}
	_ = p.SetReadTimeout(250 * time.Millisecond)
	d.serialMu.Lock()
	d.serialPort = p
	d.serialMu.Unlock()
	d.serialOnline.Store(true)
	// Python's interface.online reflects WDCL connection state, not just "serial open".
	// We mark Online once we receive ET_PROTO_WDCL_CONNECTION.
	d.iface.Online = false
	if DiagLogf != nil {
		DiagLogf(LogVerbose, "Serial port %s opened for WeaveInterface", d.cfg.port)
	}
	go d.readLoop()
	d.sendDiscover()
	return nil
}

func (d *WeaveInterfaceDriver) closeSerial() {
	d.serialOnline.Store(false)
	d.wdclOK.Store(false)
	if d.iface != nil {
		d.iface.Online = false
	}
	if DiagLogf != nil {
		DiagLogf(LogVerbose, "Serial port %s closed", d.cfg.port)
	}
	d.remoteMu.Lock()
	d.remoteKnown = false
	d.remotePub = nil
	d.remoteSID = [wdclSwitchIDLen]byte{}
	d.remoteMu.Unlock()
	if d.device != nil {
		d.device.connected = false
		d.device.ResetDisplay()
	}
	d.serialMu.Lock()
	defer d.serialMu.Unlock()
	if d.serialPort != nil {
		_ = d.serialPort.Close()
		d.serialPort = nil
	}
}

func (d *WeaveInterfaceDriver) writeRaw(payload []byte) error {
	d.serialMu.Lock()
	p := d.serialPort
	d.serialMu.Unlock()
	if p == nil {
		return errors.New("serial closed")
	}
	framed := append([]byte{hdlcFlag}, hdlcEscape(payload)...)
	framed = append(framed, hdlcFlag)
	d.writeMu.Lock()
	defer d.writeMu.Unlock()
	_, err := p.Write(framed)
	return err
}

func (d *WeaveInterfaceDriver) readLoop() {
	defer d.closeSerial()
	buf := make([]byte, 1500)
	var frameBuf []byte
	for d.serialOnline.Load() {
		n, err := func() (int, error) {
			d.serialMu.Lock()
			p := d.serialPort
			d.serialMu.Unlock()
			if p == nil {
				return 0, errors.New("serial closed")
			}
			return p.Read(buf)
		}()
		if err != nil {
			if DiagLogf != nil {
				DiagLogf(LogError, "Serial port read error on %s: %v", d.cfg.port, err)
			}
			return
		}
		if n <= 0 {
			continue
		}
		frameBuf = append(frameBuf, buf[:n]...)
		for {
			start := bytes.IndexByte(frameBuf, hdlcFlag)
			if start < 0 {
				if len(frameBuf) > 4096 {
					frameBuf = frameBuf[len(frameBuf)-1024:]
				}
				break
			}
			relEnd := bytes.IndexByte(frameBuf[start+1:], hdlcFlag)
			if relEnd < 0 {
				if start > 0 {
					frameBuf = frameBuf[start:]
				}
				break
			}
			end := start + 1 + relEnd
			raw := frameBuf[start+1 : end]
			frameBuf = frameBuf[end:]
			payload := hdlcUnescape(raw)
			if len(payload) >= wdclHeaderMinSize {
				d.handleWDCL(payload)
			}
		}
	}
}

func (d *WeaveInterfaceDriver) handleWDCL(frame []byte) {
	if len(frame) < wdclHeaderMinSize {
		return
	}
	var dst [wdclSwitchIDLen]byte
	copy(dst[:], frame[:wdclSwitchIDLen])
	pktType := frame[wdclSwitchIDLen]
	payload := frame[wdclHeaderMinSize:]

	// Python WeaveInterface accepts DISCOVER/LOG/DISP frames without requiring
	// dst match (the first 4 bytes can be the signed_id / remote id).
	// For endpoint packets, dst must match our local switch ID.
	if pktType == wdclTSEndpointPkt || pktType == wdclTSCmd || pktType == wdclTSConnect || pktType == wdclTSEncapProto {
		if dst != d.localSID && dst != wdclBroadcast {
			return
		}
	}

	switch pktType {
	case wdclTSDiscover:
		d.handleDiscover(dst, payload)
	case wdclTSLog:
		d.handleLog(payload)
	case wdclTSDisp:
		d.handleDisp(payload)
	case wdclTSEndpointPkt:
		d.handleEndpointPkt(payload)
	case wdclTSCmd:
		d.handleCmd(payload)
	case wdclTSEncapProto:
		d.handleEncapProto(payload)
	}
}

func (d *WeaveInterfaceDriver) handleEncapProto(payload []byte) {
	// Frame type exists in Python (WDCL_T_ENCAP_PROTO) but is not yet processed there.
	// We implement a minimal, forward-compatible handler:
	// payload = proto(1) + data(...)
	if len(payload) < 1 {
		return
	}
	proto := payload[0]
	data := payload[1:]
	if WeaveEncapProtoHandler != nil {
		WeaveEncapProtoHandler(proto, append([]byte(nil), data...), d.iface)
		return
	}
}

func (d *WeaveInterfaceDriver) handleDiscover(dst [wdclSwitchIDLen]byte, payload []byte) {
	// Python discovery response layout is:
	//   [signed_id (also used as dst)][type][remote_pub][remote_sig]
	// where remote_sig validates signed_id.
	//
	// Some implementations may also include signed_id in payload, so accept:
	//   payload = [signed_id][remote_pub][remote_sig]
	var (
		signedID  []byte
		remotePub ed25519.PublicKey
		sig       []byte
	)

	switch len(payload) {
	case wdclPubKeySize + wdclSignatureSize:
		signedID = dst[:]
		remotePub = ed25519.PublicKey(payload[:wdclPubKeySize])
		sig = payload[wdclPubKeySize:]
	case wdclSwitchIDLen + wdclPubKeySize + wdclSignatureSize:
		signedID = payload[:wdclSwitchIDLen]
		remotePub = ed25519.PublicKey(payload[wdclSwitchIDLen : wdclSwitchIDLen+wdclPubKeySize])
		sig = payload[wdclSwitchIDLen+wdclPubKeySize:]
	default:
		return
	}
	if !ed25519.Verify(remotePub, signedID, sig) {
		return
	}
	var remoteSID [wdclSwitchIDLen]byte
	copy(remoteSID[:], remotePub[len(remotePub)-wdclSwitchIDLen:])
	d.remoteMu.Lock()
	d.remotePub = append(ed25519.PublicKey(nil), remotePub...)
	d.remoteSID = remoteSID
	d.remoteKnown = true
	d.remoteMu.Unlock()
	d.sendConnectHandshake()
}

func (d *WeaveInterfaceDriver) handleEndpointPkt(payload []byte) {
	if len(payload) <= wdclEndpointIDLen {
		return
	}
	src := payload[len(payload)-wdclEndpointIDLen:]
	data := payload[:len(payload)-wdclEndpointIDLen]
	d.ProcessIncoming(append([]byte(nil), src...), append([]byte(nil), data...))
}

func (d *WeaveInterfaceDriver) handleCmd(payload []byte) {
	// Format: cmd(2) + cmd_data(...)
	if len(payload) < 2 {
		return
	}
	cmd := binary.BigEndian.Uint16(payload[:2])
	cmdData := payload[2:]
	switch cmd {
	case wdclCmdEndpointPkt:
		// cmd_data: endpoint_id(8) + packet_data(...)
		if len(cmdData) <= wdclEndpointIDLen {
			return
		}
		endpoint := cmdData[:wdclEndpointIDLen]
		data := cmdData[wdclEndpointIDLen:]
		d.ProcessIncoming(append([]byte(nil), endpoint...), append([]byte(nil), data...))
	case wdclCmdRemoteDisplay:
		// cmd_data: enabled(1)
		return
	case wdclCmdRemoteInput:
		// Host->device only in Python. Ignore if received from device.
		return
	case wdclCmdEndpoints:
		// Host->device only in Python. Ignore if received from device.
		return
	default:
		return
	}
}

func (d *WeaveInterfaceDriver) handleLog(payload []byte) {
	tryParse := func(off int) (LogFrame, bool) {
		if len(payload) < off+1+4+1+2 {
			return LogFrame{}, false
		}
		tsMs := binary.BigEndian.Uint32(payload[off+1 : off+5])
		lvl := payload[off+5]
		evt := binary.BigEndian.Uint16(payload[off+6 : off+8])
		data := append([]byte(nil), payload[off+8:]...)
		uptime := time.Duration(tsMs) * time.Millisecond
		return LogFrame{
			Timestamp: time.Now(),
			Uptime:    uptime,
			Level:     WeaveEventLevel(lvl),
			Event:     WeaveEventID(evt),
			Data:      data,
		}, true
	}

	frame, ok := tryParse(0)
	if !ok {
		frame, ok = tryParse(1)
	}
	if !ok {
		return
	}

	frame.Rendered = renderWeaveLogFrame(frame)
	d.device.EnqueueLog(frame)

	switch frame.Event {
	case evtProtoWDCLConnection:
		d.wdclOK.Store(true)
		d.device.connected = true
		if d.iface != nil {
			d.iface.Online = true
		}
	case evtProtoWDCLHostEP:
		if len(frame.Data) == wdclEndpointIDLen {
			d.device.endpointID = hex.EncodeToString(frame.Data)
		}
	case evtProtoWeaveEPAlive:
		if len(frame.Data) == wdclEndpointIDLen {
			d.addOrRefreshPeer(frame.Data)
			d.device.EndpointAlive(hex.EncodeToString(frame.Data))
		}
	case evtProtoWeaveEPTimeout:
		if len(frame.Data) == wdclEndpointIDLen {
			key := hex.EncodeToString(frame.Data)
			d.device.EndpointTimeout(key)
			d.mu.Lock()
			st := d.peers[key]
			if st != nil {
				delete(d.peers, key)
			}
			d.mu.Unlock()
			if st != nil {
				removeInterface(st.iface)
			}
		}
	case evtProtoWeaveEPVia:
		if len(frame.Data) == wdclEndpointIDLen+wdclSwitchIDLen {
			ep := frame.Data[:wdclEndpointIDLen]
			via := frame.Data[wdclEndpointIDLen:]
			d.device.EndpointVia(hex.EncodeToString(ep), hex.EncodeToString(via))
		}
	case evtStatTaskCPU:
		if len(frame.Data) > 1 {
			name := strings.TrimSpace(string(frame.Data[1:]))
			if name != "" {
				if desc, ok := weaveTaskDescriptions[name]; ok {
					name = desc
				}
				d.device.CaptureTask(name, float64(frame.Data[0]))
			}
		}
	case evtStatCPU:
		if len(frame.Data) >= 1 {
			d.device.CaptureCPU(float64(frame.Data[0]))
		}
	case evtStatMemory:
		if len(frame.Data) >= 8 {
			free := binary.BigEndian.Uint32(frame.Data[:4])
			total := binary.BigEndian.Uint32(frame.Data[4:8])
			d.device.CaptureMemory(free, total)
		}
	case evtMsg:
	default:
	}
}

func (d *WeaveInterfaceDriver) handleDisp(payload []byte) {
	// Python layout:
	// cf(1) + ofs(4) + dsz(4) + data(...)
	if len(payload) < 1+4+4 {
		return
	}
	ofs := int(binary.BigEndian.Uint32(payload[1:5]))
	dsz := int(binary.BigEndian.Uint32(payload[5:9]))
	if dsz <= 0 || dsz > 10_000_000 {
		return
	}
	chunk := payload[9:]
	d.device.UpdateDisplayChunk(ofs, dsz, chunk)

	if d.remoteDisplay.Load() && WeaveRemoteDisplayHandler != nil {
		if buf, ok := d.device.DisplaySnapshot(); ok {
			WeaveRemoteDisplayHandler(buf, weaveDisplayWidth, weaveDisplayHeight, d.device)
		}
	}
}

func (d *WeaveInterfaceDriver) sendDiscover() {
	frame := append(append([]byte(nil), wdclBroadcast[:]...), wdclTSDiscover)
	frame = append(frame, d.localSID[:]...)
	_ = d.writeRaw(frame)
}

func (d *WeaveInterfaceDriver) sendConnectHandshake() {
	d.remoteMu.Lock()
	remoteKnown := d.remoteKnown
	remoteSID := d.remoteSID
	d.remoteMu.Unlock()
	if !remoteKnown {
		return
	}
	if d.localSign == nil || len(d.localSigPub) != wdclPubKeySize {
		return
	}
	// Python signs the remote switch_id (self.switch_id) with the local switch identity.
	sig, err := d.localSign(remoteSID[:])
	if err != nil || len(sig) != wdclSignatureSize {
		return
	}
	if DiagLogf != nil {
		DiagLogf(LogVerbose, "WDCL connection handshake sent to %s", hex.EncodeToString(remoteSID[:]))
	}
	payload := append([]byte(nil), d.localSigPub...)
	payload = append(payload, sig...)
	frame := append([]byte(nil), remoteSID[:]...)
	frame = append(frame, wdclTSConnect)
	frame = append(frame, payload...)
	_ = d.writeRaw(frame)

	go func() {
		t := time.NewTimer(weaveHandshakeTimeout)
		defer t.Stop()
		select {
		case <-t.C:
			if !d.wdclOK.Load() {
				if DiagLogf != nil {
					DiagLogf(LogWarning, "WDCL handshake timed out for %s", d.cfg.port)
				}
				// Reconnect from scratch like Python behaviour on handshake timeout.
				d.closeSerial()
			}
		case <-d.stopCh:
		}
	}()
}

func (d *WeaveInterfaceDriver) sendEndpointPacket(endpoint []byte, data []byte) {
	d.remoteMu.Lock()
	remoteKnown := d.remoteKnown
	remoteSID := d.remoteSID
	d.remoteMu.Unlock()
	if !remoteKnown || !d.serialOnline.Load() || !d.wdclOK.Load() {
		return
	}

	const maxChunk = 32768
	for offset := 0; offset < len(data); offset += maxChunk {
		end := offset + maxChunk
		if end > len(data) {
			end = len(data)
		}
		chunk := data[offset:end]

		cmdPayload := make([]byte, 2+len(endpoint)+len(chunk))
		binary.BigEndian.PutUint16(cmdPayload[:2], wdclCmdEndpointPkt)
		copy(cmdPayload[2:], endpoint)
		copy(cmdPayload[2+len(endpoint):], chunk)

		frame := append([]byte(nil), remoteSID[:]...)
		frame = append(frame, wdclTSCmd)
		frame = append(frame, cmdPayload...)
		_ = d.writeRaw(frame)
	}
}

func (d *WeaveInterfaceDriver) SetRemoteDisplay(enabled bool) {
	d.remoteMu.Lock()
	remoteKnown := d.remoteKnown
	remoteSID := d.remoteSID
	d.remoteMu.Unlock()
	d.remoteDisplay.Store(enabled)
	if !remoteKnown || !d.serialOnline.Load() || !d.wdclOK.Load() {
		return
	}
	cmdPayload := make([]byte, 2+1)
	binary.BigEndian.PutUint16(cmdPayload[:2], wdclCmdRemoteDisplay)
	if enabled {
		cmdPayload[2] = 0x01
	} else {
		cmdPayload[2] = 0x00
	}
	frame := append([]byte(nil), remoteSID[:]...)
	frame = append(frame, wdclTSCmd)
	frame = append(frame, cmdPayload...)
	_ = d.writeRaw(frame)
}

func (d *WeaveInterfaceDriver) SendRemoteInput(payload []byte) {
	d.remoteMu.Lock()
	remoteKnown := d.remoteKnown
	remoteSID := d.remoteSID
	d.remoteMu.Unlock()
	if !remoteKnown || !d.serialOnline.Load() || !d.wdclOK.Load() {
		return
	}
	cmdPayload := make([]byte, 2+len(payload))
	binary.BigEndian.PutUint16(cmdPayload[:2], wdclCmdRemoteInput)
	copy(cmdPayload[2:], payload)
	frame := append([]byte(nil), remoteSID[:]...)
	frame = append(frame, wdclTSCmd)
	frame = append(frame, cmdPayload...)
	_ = d.writeRaw(frame)
}

func (d *WeaveInterfaceDriver) RequestEndpointsList() {
	d.remoteMu.Lock()
	remoteKnown := d.remoteKnown
	remoteSID := d.remoteSID
	d.remoteMu.Unlock()
	if !remoteKnown || !d.serialOnline.Load() || !d.wdclOK.Load() {
		return
	}
	cmdPayload := make([]byte, 2)
	binary.BigEndian.PutUint16(cmdPayload[:2], wdclCmdEndpoints)
	frame := append([]byte(nil), remoteSID[:]...)
	frame = append(frame, wdclTSCmd)
	frame = append(frame, cmdPayload...)
	_ = d.writeRaw(frame)
}
