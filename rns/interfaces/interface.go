package interfaces

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	IAFreqSamples = 6
	OAFreqSamples = 6

	MaxHeldAnnounces = 256

	// Interface mode definitions (mirrors Python Interface.* constants).
	InterfaceModeFull         = 0x01
	InterfaceModePointToPoint = 0x02
	InterfaceModeAccessPoint  = 0x03
	InterfaceModeRoaming      = 0x04
	InterfaceModeBoundary     = 0x05
	InterfaceModeGateway      = 0x06

	ICNewTime             = 2 * 60 * 60
	ICBurstFreqNew        = 3.5
	ICBurstFreq           = 12.0
	ICBurstHold           = 1 * 60
	ICBurstPenalty        = 5 * 60
	ICHeldReleaseInterval = 30

	MaxFrameLength = 1 << 20
)

// DiscoverPathsFor mirrors Python Interface.DISCOVER_PATHS_FOR.
var DiscoverPathsFor = []int{InterfaceModeAccessPoint, InterfaceModeGateway, InterfaceModeRoaming}

// InboundHandler is set by the rns package to avoid import cycles.
// It is used by LocalInterface to feed received frames into transport.
var InboundHandler func(raw []byte, ifc *Interface)

// RemoveInterfaceHandler is set by the rns package so drivers can drop
// interfaces without importing the transport layer.
var RemoveInterfaceHandler func(ifc *Interface)

// WeaveIdentityProvider is set by the rns package so WeaveInterface can
// use the same persisted Identity semantics as Python (sig_pub_bytes/sign)
// without creating import cycles.
// It returns the Ed25519 public key bytes (32) and a signer func.
var WeaveIdentityProvider func(port string) (sigPub []byte, sign func(msg []byte) ([]byte, error), err error)

// QueuedAnnounceLife mirrors Reticulum.QUEUED_ANNOUNCE_LIFE.
var QueuedAnnounceLife = 24 * time.Hour

// DefaultAnnounceCapProvider is set by the rns package and mirrors
// Reticulum.ANNOUNCE_CAP defaulting in Python.
// When non-nil and returning >0, it is used when Interface.AnnounceCap is unset (<=0).
var DefaultAnnounceCapProvider func() float64

// ---------- logging hooks ----------
//
// DiagLog and DiagLogf are set by the rns package so interface drivers can emit
// diagnostics without importing rns and creating cycles.
var DiagLog func(msg any, level int)
var DiagLogf func(level int, format string, args ...any)

// HeaderMinSize is set by the rns package and mirrors Reticulum.HEADER_MINSIZE.
// When non-zero, LocalInterface framing drops frames shorter than or equal to this value,
// like the Python implementation.
var HeaderMinSize int

// ExitFunc/PanicFunc are set by the rns package to mirror RNS.exit()/RNS.panic()
// without introducing import cycles.
var ExitFunc func(code ...int)
var PanicFunc func()

// PanicOnInterfaceErrorProvider mirrors Reticulum.panic_on_interface_error.
var PanicOnInterfaceErrorProvider func() bool

// Log levels mirror rns.Log* constants.
const (
	LogNone     = -1
	LogCritical = 0
	LogError    = 1
	LogWarning  = 2
	LogNotice   = 3
	LogInfo     = 4
	LogVerbose  = 5
	LogDebug    = 6
	LogExtreme  = 7
)

// HDLC framing, like Python LocalInterface.
const (
	hdlcFlag    = 0x7E
	hdlcEsc     = 0x7D
	hdlcEscMask = 0x20
)

func hdlcEscape(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	out := make([]byte, 0, len(data)+8)
	for _, b := range data {
		switch b {
		case hdlcEsc:
			out = append(out, hdlcEsc, hdlcEsc^hdlcEscMask)
		case hdlcFlag:
			out = append(out, hdlcEsc, hdlcFlag^hdlcEscMask)
		default:
			out = append(out, b)
		}
	}
	return out
}

func hdlcUnescape(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	out := make([]byte, 0, len(data))
	esc := false
	for _, b := range data {
		if esc {
			out = append(out, b^hdlcEscMask)
			esc = false
			continue
		}
		if b == hdlcEsc {
			esc = true
			continue
		}
		out = append(out, b)
	}
	return out
}

// Interface is the shared interface representation used by the rns transport.
// This mirrors the Python Interface base class state and hooks.
type Interface struct {
	Name   string
	Type   string
	Parent *Interface

	IN   bool
	OUT  bool
	FWD  bool
	RPT  bool
	Mode int

	// WantsTunnel mirrors Python interface.wants_tunnel.
	// If true, the transport should synthesize a tunnel for this interface.
	WantsTunnel bool

	Bitrate     int
	HWMTU       int
	AnnounceCap float64

	AnnounceRateTarget  *int
	AnnounceRateGrace   *int
	AnnounceRatePenalty *int

	IngressControl        bool
	ICMaxHeldAnnounces    *int
	ICBurstHold           *float64
	ICBurstFreqNew        *float64
	ICBurstFreq           *float64
	ICNewTime             *float64
	ICBurstPenalty        *float64
	ICHeldReleaseInterval *float64

	DriverImplemented bool
	DriverError       string

	AutoconfigureMTU bool
	FixedMTU         bool

	IFACSize       int
	IFACKey        []byte
	IFACIdentity   interface{ Sign([]byte) ([]byte, error) }
	IFACSignature  []byte
	IFACNetnameVal string
	IFACNetkeyVal  string

	Discoverable              bool
	DiscoveryAnnounceInterval time.Duration
	LastDiscoveryAnnounce     time.Time
	DiscoveryStampValue       *int
	DiscoveryName             string
	DiscoveryEncrypt          bool
	DiscoveryPublishIFAC      bool
	DiscoveryLatitude         *float64
	DiscoveryLongitude        *float64
	DiscoveryHeight           *float64
	DiscoveryReachableOn      string
	DiscoveryPort             *int
	DiscoveryFrequency        *int
	DiscoveryBandwidth        *int
	DiscoveryChannel          *int
	DiscoveryModulation       string
	AutoconnectHash           []byte
	AutoconnectSource         string
	AutoconnectDown           time.Time
	BootstrapOnly             bool

	RXB uint64
	TXB uint64

	CurRxSpeed float64
	CurTxSpeed float64

	TrafficCounter *TrafficCounter

	Online   bool
	Detached bool
	Created  time.Time

	udpBindAddr    *net.UDPAddr
	udpForwardAddr *net.UDPAddr
	udpConn        *net.UDPConn

	TunnelID []byte

	localMu     sync.Mutex
	localConn   net.Conn
	localLn     net.Listener
	localSendMu sync.Mutex

	// ForceBitrateLatency simulates transmit time for interfaces where Python
	// intentionally sleeps based on bitrate (eg. LocalInterface with forced bitrate).
	ForceBitrateLatency bool
	LocalIsSharedClient bool

	icMu              sync.Mutex
	icBurstActive     bool
	icBurstActivated  time.Time
	icHeldRelease     time.Time
	heldAnnounces     map[string]heldAnnounce
	iaFreq            []time.Time
	oaFreq            []time.Time
	announceAllowedAt time.Time
	announceQueue     []announceQueueEntry
	announceRunning   bool

	auto              *autoState
	ax25              *AX25KISSDriver
	kiss              *KISSDriver
	backboneServer    *BackboneInterfaceDriver
	backboneClient    *BackboneClientDriver
	backbonePeer      *BackbonePeer
	i2pClient         *I2PClientDriver
	i2pPeer           *I2PPeer
	pipe              *PipeDriver
	weave             *WeaveInterfaceDriver
	serial            *SerialDriver
	rnodeMulti        *RNodeMultiDriver
	rnodeSub          *RNodeSubDriver
	rnodeSingle       *RNodeInterface
	tcpClient         *TCPClientInterface
	tcpServer         *TCPServerInterface
	clientCount       func() int
	processOutgoingFn func([]byte) error
	detachFn          func()
}

func (i *Interface) SetTCPClient(ci *TCPClientInterface) {
	if i == nil {
		return
	}
	i.tcpClient = ci
	if ci != nil {
		ci.iface = i
	}
}

func (i *Interface) SetTCPServer(s *TCPServerInterface) {
	if i == nil {
		return
	}
	i.tcpServer = s
}

type heldAnnounce struct {
	raw      []byte
	recvIf   *Interface
	hops     uint8
	destHash []byte
}

type announceQueueEntry struct {
	enqueuedAt time.Time
	hops       int
	raw        []byte
}

type TrafficCounter struct {
	TS  time.Time
	RXB uint64
	TXB uint64
}

func (i *Interface) String() string {
	if i == nil {
		return "<nil>"
	}
	if strings.EqualFold(i.Type, "TCPClientInterface") && i.tcpClient != nil {
		return i.tcpClient.String()
	}
	if strings.EqualFold(i.Type, "TCPServerInterface") && i.tcpServer != nil {
		return i.tcpServer.String()
	}
	if strings.EqualFold(i.Type, "UDPInterface") {
		bind := i.udpBindAddr
		fwd := i.udpForwardAddr
		if bind != nil {
			return fmt.Sprintf("UDPInterface[%s/%s]", i.Name, bind.String())
		}
		if fwd != nil {
			return fmt.Sprintf("UDPInterface[%s/%s]", i.Name, fwd.String())
		}
	}
	if strings.EqualFold(i.Type, "SerialInterface") && i.Name != "" {
		return fmt.Sprintf("SerialInterface[%s]", i.Name)
	}
	if strings.EqualFold(i.Type, "RNodeInterface") && i.Name != "" {
		return fmt.Sprintf("RNodeInterface[%s]", i.Name)
	}
	if strings.EqualFold(i.Type, "RNodeMultiInterface") && i.Name != "" {
		return fmt.Sprintf("RNodeMultiInterface[%s]", i.Name)
	}
	if strings.EqualFold(i.Type, "PipeInterface") && i.Name != "" {
		return fmt.Sprintf("PipeInterface[%s]", i.Name)
	}
	if strings.EqualFold(i.Type, "KISSInterface") && i.Name != "" {
		return fmt.Sprintf("KISSInterface[%s]", i.Name)
	}
	if strings.EqualFold(i.Type, "I2PInterface") && i.Name != "" {
		return fmt.Sprintf("I2PInterface[%s]", i.Name)
	}
	if strings.EqualFold(i.Type, "I2PInterfacePeer") && i.Name != "" {
		return fmt.Sprintf("I2PInterfacePeer[%s]", i.Name)
	}
	if strings.EqualFold(i.Type, "AutoInterface") && i.Name != "" {
		return fmt.Sprintf("AutoInterface[%s]", i.Name)
	}
	if strings.EqualFold(i.Type, "AX25KISSInterface") && i.Name != "" {
		return fmt.Sprintf("AX25KISSInterface[%s]", i.Name)
	}
	if strings.EqualFold(i.Type, "BackboneInterface") && i.backboneServer != nil {
		ipStr := i.backboneServer.cfg.Listen
		if ipStr == "" {
			ipStr = i.Name
		}
		if strings.Contains(ipStr, ":") && !strings.HasPrefix(ipStr, "[") {
			ipStr = "[" + ipStr + "]"
		}
		return fmt.Sprintf("BackboneInterface[%s/%s:%d]", i.Name, ipStr, i.backboneServer.cfg.Port)
	}
	if strings.EqualFold(i.Type, "BackboneClientInterface") && i.backboneClient != nil {
		ipStr := i.backboneClient.cfg.TargetHost
		if strings.Contains(ipStr, ":") && !strings.HasPrefix(ipStr, "[") {
			ipStr = "[" + ipStr + "]"
		}
		return fmt.Sprintf("BackboneInterface[%s/%s:%d]", i.Name, ipStr, i.backboneClient.cfg.TargetPort)
	}
	if i.Name != "" {
		return i.Name
	}
	return "Interface"
}

func (i *Interface) Hash() []byte {
	if i == nil {
		return nil
	}
	sum := sha256.Sum256([]byte(i.String()))
	return sum[:]
}

// GetHash mirrors Python Interface.get_hash().
func (i *Interface) GetHash() []byte { return i.Hash() }

func (i *Interface) FinalInit() {
	if i != nil {
		if i.Created.IsZero() {
			i.Created = time.Now()
		}
		i.Detached = false
	}
}

func (i *Interface) SetProcessOutgoingFunc(fn func([]byte) error) {
	if i == nil {
		return
	}
	i.processOutgoingFn = fn
}

func (i *Interface) SetDetachFunc(fn func()) {
	if i == nil {
		return
	}
	i.detachFn = fn
}

func (i *Interface) Detach() {
	if i != nil {
		i.Online = false
		i.Detached = true
		i.localMu.Lock()
		if i.localConn != nil {
			_ = i.localConn.Close()
			i.localConn = nil
		}
		if i.localLn != nil {
			_ = i.localLn.Close()
			i.localLn = nil
		}
		i.localMu.Unlock()
		if i.auto != nil {
			i.auto.stop()
		}
		if i.weave != nil {
			i.weave.Close()
		}
		if i.ax25 != nil {
			i.ax25.Close()
		}
		if i.kiss != nil {
			i.kiss.Close()
		}
		if i.backboneServer != nil {
			i.backboneServer.Close()
		}
		if i.backboneClient != nil {
			i.backboneClient.Close()
		}
		if i.backbonePeer != nil {
			i.backbonePeer.Close()
		}
		if i.pipe != nil {
			i.pipe.Close()
		}
		if i.i2pClient != nil {
			i.i2pClient.Close()
		}
		if i.i2pPeer != nil {
			i.i2pPeer.Close()
		}
		if i.serial != nil {
			i.serial.Close()
		}
		if i.rnodeMulti != nil {
			i.rnodeMulti.Close()
		}
		if i.rnodeSingle != nil {
			_ = i.rnodeSingle.Stop()
		}
		if i.tcpClient != nil {
			i.tcpClient.Detach()
		}
		if i.tcpServer != nil {
			i.tcpServer.Detach()
		}
		if i.detachFn != nil {
			i.detachFn()
		}
	}
}

func (i *Interface) Age() time.Duration {
	if i == nil {
		return 0
	}
	if i.Created.IsZero() {
		i.Created = time.Now()
		return 0
	}
	return time.Since(i.Created)
}

func (i *Interface) AutoPeerCount() *int {
	if i == nil || !strings.EqualFold(i.Type, "AutoInterface") || i.auto == nil {
		return nil
	}
	i.auto.mu.Lock()
	n := len(i.auto.spawned)
	i.auto.mu.Unlock()
	return &n
}

func (i *Interface) WeavePeerCount() *int {
	if i == nil || !strings.EqualFold(i.Type, "WeaveInterface") || i.weave == nil {
		return nil
	}
	i.weave.mu.Lock()
	n := len(i.weave.peers)
	i.weave.mu.Unlock()
	return &n
}

func (i *Interface) ClientCount() *int {
	if i == nil || i.clientCount == nil {
		return nil
	}
	c := i.clientCount()
	return &c
}

func (i *Interface) SupportsDiscovery() bool {
	if i == nil {
		return false
	}
	switch i.Type {
	case "BackboneInterface", "TCPServerInterface", "TCPClientInterface", "RNodeInterface", "WeaveInterface", "I2PInterface", "KISSInterface":
		return true
	default:
		return false
	}
}

func (i *Interface) DiscoveryNameValue() string {
	if i == nil {
		return ""
	}
	if strings.TrimSpace(i.DiscoveryName) != "" {
		return i.DiscoveryName
	}
	if strings.TrimSpace(i.Name) != "" {
		return i.Name
	}
	return i.String()
}

func (i *Interface) DiscoveryReachableOnValue() string {
	if i == nil {
		return ""
	}
	if strings.TrimSpace(i.DiscoveryReachableOn) != "" {
		return strings.TrimSpace(i.DiscoveryReachableOn)
	}
	if i.backboneClient != nil {
		return strings.TrimSpace(i.backboneClient.cfg.TargetHost)
	}
	if i.tcpClient != nil && i.tcpClient.Initiator {
		return strings.TrimSpace(i.tcpClient.TargetHost)
	}
	if i.tcpServer != nil {
		host, _, err := net.SplitHostPort(i.tcpServer.ListenAddr)
		if err == nil {
			return strings.Trim(strings.TrimSpace(host), "[]")
		}
	}
	return ""
}

func (i *Interface) DiscoveryPortValue() *int {
	if i == nil {
		return nil
	}
	if i.DiscoveryPort != nil && *i.DiscoveryPort > 0 {
		v := *i.DiscoveryPort
		return &v
	}
	if i.backboneClient != nil && i.backboneClient.cfg.TargetPort > 0 {
		v := i.backboneClient.cfg.TargetPort
		return &v
	}
	if i.tcpClient != nil && i.tcpClient.Initiator && strings.TrimSpace(i.tcpClient.TargetPort) != "" {
		if p, err := strconv.Atoi(i.tcpClient.TargetPort); err == nil && p > 0 {
			v := p
			return &v
		}
	}
	if i.tcpServer != nil {
		_, port, err := net.SplitHostPort(i.tcpServer.ListenAddr)
		if err == nil {
			if p, err := strconv.Atoi(port); err == nil && p > 0 {
				return &p
			}
		}
	}
	return nil
}

func (i *Interface) DiscoveryKISSFraming() bool {
	if i == nil {
		return false
	}
	if i.tcpClient != nil {
		return i.tcpClient.KISSFraming
	}
	if i.tcpServer != nil {
		return i.tcpServer.KISSFraming
	}
	if i.i2pClient != nil {
		return i.i2pClient.cfg.KISSFraming
	}
	return strings.EqualFold(i.Type, "KISSInterface")
}

func (i *Interface) DiscoveryRNodeRadioParams() (frequency, bandwidth int, sf, cr int, ok bool) {
	if i == nil {
		return 0, 0, 0, 0, false
	}
	if strings.EqualFold(i.Type, "RNodeInterface") && i.rnodeSingle != nil {
		return int(i.rnodeSingle.Frequency), int(i.rnodeSingle.Bandwidth), int(i.rnodeSingle.SF), int(i.rnodeSingle.CR), true
	}
	if strings.EqualFold(i.Type, "RNodeMultiInterface") && i.rnodeSub != nil {
		return int(i.rnodeSub.frequency), int(i.rnodeSub.bandwidth), int(i.rnodeSub.sf), int(i.rnodeSub.cr), true
	}
	return 0, 0, 0, 0, false
}

func (i *Interface) DiscoveryWeaveParams() (frequency, bandwidth, channel *int, modulation string) {
	if i == nil {
		return nil, nil, nil, ""
	}
	return i.DiscoveryFrequency, i.DiscoveryBandwidth, i.DiscoveryChannel, strings.TrimSpace(i.DiscoveryModulation)
}

func (i *Interface) I2PConnectable() *bool {
	if i == nil || !strings.EqualFold(i.Type, "I2PInterface") || i.i2pClient == nil {
		return nil
	}
	v := i.i2pClient.connectable.Load()
	return &v
}

func (i *Interface) I2PB32() *string {
	if i == nil || !strings.EqualFold(i.Type, "I2PInterface") || i.i2pClient == nil {
		return nil
	}
	if s, ok := i.i2pClient.b32.Load().(string); ok && strings.TrimSpace(s) != "" {
		if strings.HasSuffix(s, ".b32.i2p") {
			return &s
		}
		withSuffix := s + ".b32.i2p"
		return &withSuffix
	}
	return nil
}

func (i *Interface) DiscoveryTargetHost() *string {
	if i == nil {
		return nil
	}
	if i.backboneClient != nil && strings.TrimSpace(i.backboneClient.cfg.TargetHost) != "" {
		v := strings.TrimSpace(i.backboneClient.cfg.TargetHost)
		return &v
	}
	if i.tcpClient != nil && i.tcpClient.Initiator && strings.TrimSpace(i.tcpClient.TargetHost) != "" {
		v := strings.TrimSpace(i.tcpClient.TargetHost)
		return &v
	}
	if b32 := i.I2PB32(); b32 != nil && strings.TrimSpace(*b32) != "" {
		v := strings.TrimSpace(*b32)
		return &v
	}
	return nil
}

func (i *Interface) DiscoveryTargetPort() *int {
	if i == nil {
		return nil
	}
	if i.backboneClient != nil && i.backboneClient.cfg.TargetPort > 0 {
		v := i.backboneClient.cfg.TargetPort
		return &v
	}
	if i.tcpClient != nil && i.tcpClient.Initiator && strings.TrimSpace(i.tcpClient.TargetPort) != "" {
		if p, err := strconv.Atoi(i.tcpClient.TargetPort); err == nil && p > 0 {
			v := p
			return &v
		}
	}
	return nil
}

func (i *Interface) I2PTunnelState() *string {
	if i == nil || !strings.EqualFold(i.Type, "I2PInterfacePeer") || i.i2pPeer == nil {
		return nil
	}
	switch i.i2pPeer.tunnelState.Load() {
	case i2pTunnelStateAlive:
		s := "Tunnel Active"
		return &s
	case i2pTunnelStateInit:
		s := "Creating Tunnel"
		return &s
	case i2pTunnelStateStale:
		s := "Tunnel Unresponsive"
		return &s
	default:
		s := "Unknown State"
		return &s
	}
}

func (i *Interface) RNodeAirtimeShort() *float64 {
	if i == nil {
		return nil
	}
	if strings.EqualFold(i.Type, "RNodeInterface") && i.rnodeSingle != nil {
		v := float64(i.rnodeSingle.rAirtimeShort)
		return &v
	}
	if strings.EqualFold(i.Type, "RNodeMultiInterface") && i.rnodeMulti != nil {
		v := float64(math.Float32frombits(i.rnodeMulti.rAirtimeShortBits.Load()))
		return &v
	}
	return nil
}

func (i *Interface) RNodeAirtimeLong() *float64 {
	if i == nil {
		return nil
	}
	if strings.EqualFold(i.Type, "RNodeInterface") && i.rnodeSingle != nil {
		v := float64(i.rnodeSingle.rAirtimeLong)
		return &v
	}
	if strings.EqualFold(i.Type, "RNodeMultiInterface") && i.rnodeMulti != nil {
		v := float64(math.Float32frombits(i.rnodeMulti.rAirtimeLongBits.Load()))
		return &v
	}
	return nil
}

func (i *Interface) RNodeChannelLoadShort() *float64 {
	if i == nil {
		return nil
	}
	if strings.EqualFold(i.Type, "RNodeInterface") && i.rnodeSingle != nil {
		v := float64(i.rnodeSingle.rChannelLoadShort)
		return &v
	}
	if strings.EqualFold(i.Type, "RNodeMultiInterface") && i.rnodeMulti != nil {
		v := float64(math.Float32frombits(i.rnodeMulti.rChannelLoadShortBits.Load()))
		return &v
	}
	return nil
}

func (i *Interface) RNodeChannelLoadLong() *float64 {
	if i == nil {
		return nil
	}
	if strings.EqualFold(i.Type, "RNodeInterface") && i.rnodeSingle != nil {
		v := float64(i.rnodeSingle.rChannelLoadLong)
		return &v
	}
	if strings.EqualFold(i.Type, "RNodeMultiInterface") && i.rnodeMulti != nil {
		v := float64(math.Float32frombits(i.rnodeMulti.rChannelLoadLongBits.Load()))
		return &v
	}
	return nil
}

func (i *Interface) RNodeNoiseFloor() *int {
	if i == nil {
		return nil
	}
	if strings.EqualFold(i.Type, "RNodeInterface") && i.rnodeSingle != nil {
		if i.rnodeSingle.rNoiseFloor == 0 {
			return nil
		}
		v := int(i.rnodeSingle.rNoiseFloor)
		return &v
	}
	if strings.EqualFold(i.Type, "RNodeMultiInterface") && i.rnodeMulti != nil {
		nf := i.rnodeMulti.rNoiseFloor.Load()
		if nf == 0 {
			return nil
		}
		v := int(nf)
		return &v
	}
	return nil
}

func (i *Interface) RNodeInterference() *int {
	if i == nil {
		return nil
	}
	if strings.EqualFold(i.Type, "RNodeInterface") && i.rnodeSingle != nil {
		if i.rnodeSingle.rInterference == nil {
			return nil
		}
		v := int(*i.rnodeSingle.rInterference)
		return &v
	}
	if strings.EqualFold(i.Type, "RNodeMultiInterface") && i.rnodeMulti != nil {
		vv := i.rnodeMulti.rInterference.Load()
		if vv == math.MinInt32 {
			return nil
		}
		v := int(vv)
		return &v
	}
	return nil
}

func (i *Interface) RNodeInterferenceLast() (ts *float64, dbm *int) {
	if i == nil {
		return nil, nil
	}
	if strings.EqualFold(i.Type, "RNodeInterface") && i.rnodeSingle != nil {
		if i.rnodeSingle.rInterference == nil || i.rnodeSingle.rInterferenceAt.IsZero() {
			return nil, nil
		}
		dbmVal := int(*i.rnodeSingle.rInterference)
		tsVal := float64(i.rnodeSingle.rInterferenceAt.Unix())
		return &tsVal, &dbmVal
	}
	if strings.EqualFold(i.Type, "RNodeMultiInterface") && i.rnodeMulti != nil {
		vv := i.rnodeMulti.rInterference.Load()
		tsUnix := i.rnodeMulti.rInterferenceAtUnixSec.Load()
		if vv == math.MinInt32 || tsUnix <= 0 {
			return nil, nil
		}
		dbmVal := int(vv)
		tsVal := float64(tsUnix)
		return &tsVal, &dbmVal
	}
	return nil, nil
}

func (i *Interface) RNodeBatteryState() *string {
	if i == nil || !strings.EqualFold(i.Type, "RNodeInterface") || i.rnodeSingle == nil {
		return nil
	}
	if i.rnodeSingle.batteryState == BatteryUnknown {
		return nil
	}
	s := i.rnodeSingle.BatteryStateString()
	return &s
}

func (i *Interface) RNodeBatteryPercent() *int {
	if i == nil || !strings.EqualFold(i.Type, "RNodeInterface") || i.rnodeSingle == nil {
		return nil
	}
	v := int(i.rnodeSingle.batteryPercent)
	return &v
}

func (i *Interface) RNodeCPUTemp() *int {
	if i == nil || !strings.EqualFold(i.Type, "RNodeInterface") || i.rnodeSingle == nil || i.rnodeSingle.temperatureC == nil {
		return nil
	}
	v := int(*i.rnodeSingle.temperatureC)
	return &v
}

func (i *Interface) SetClientCountFunc(fn func() int) {
	if i == nil {
		return
	}
	i.clientCount = fn
}

func (i *Interface) WeaveSwitchID() *string {
	if i == nil || !strings.EqualFold(i.Type, "WeaveInterface") || i.weave == nil || i.weave.device == nil {
		return nil
	}
	i.weave.device.mu.Lock()
	defer i.weave.device.mu.Unlock()
	if strings.TrimSpace(i.weave.device.switchID) == "" {
		return nil
	}
	s := i.weave.device.switchID
	return &s
}

func (i *Interface) WeaveEndpointID() *string {
	if i == nil || !strings.EqualFold(i.Type, "WeaveInterface") || i.weave == nil || i.weave.device == nil {
		return nil
	}
	i.weave.device.mu.Lock()
	defer i.weave.device.mu.Unlock()
	if strings.TrimSpace(i.weave.device.endpointID) == "" {
		return nil
	}
	s := i.weave.device.endpointID
	return &s
}

func (i *Interface) WeaveCPULoad() *float64 {
	if i == nil || !strings.EqualFold(i.Type, "WeaveInterface") || i.weave == nil || i.weave.device == nil {
		return nil
	}
	i.weave.device.mu.Lock()
	defer i.weave.device.mu.Unlock()
	if len(i.weave.device.cpuStats) == 0 {
		return nil
	}
	v := i.weave.device.cpuStats[len(i.weave.device.cpuStats)-1].Value
	return &v
}

func (i *Interface) WeaveMemLoad() *float64 {
	if i == nil || !strings.EqualFold(i.Type, "WeaveInterface") || i.weave == nil || i.weave.device == nil {
		return nil
	}
	i.weave.device.mu.Lock()
	defer i.weave.device.mu.Unlock()
	if i.weave.device.memoryTotal == 0 {
		return nil
	}
	v := i.weave.device.memoryUsedPct
	return &v
}

func (i *Interface) WeaveActiveTasksDetailed() map[string]WeaveTaskStat {
	if i == nil || !strings.EqualFold(i.Type, "WeaveInterface") || i.weave == nil || i.weave.device == nil {
		return nil
	}
	return i.weave.device.ActiveTasks(5 * time.Second)
}

func (i *Interface) WeaveLogHistory() []string {
	if i == nil || !strings.EqualFold(i.Type, "WeaveInterface") || i.weave == nil || i.weave.device == nil {
		return nil
	}
	return i.weave.device.LogHistory()
}

func (i *Interface) WeaveCPUHistory() []StatEntry {
	if i == nil || !strings.EqualFold(i.Type, "WeaveInterface") || i.weave == nil || i.weave.device == nil {
		return nil
	}
	return i.weave.device.CPUHistory()
}

func (i *Interface) WeaveMemoryHistory() []StatEntry {
	if i == nil || !strings.EqualFold(i.Type, "WeaveInterface") || i.weave == nil || i.weave.device == nil {
		return nil
	}
	return i.weave.device.MemoryHistory()
}

func (i *Interface) WeaveCPUStatsUI() *WeaveStatsSeries {
	if i == nil || !strings.EqualFold(i.Type, "WeaveInterface") || i.weave == nil || i.weave.device == nil {
		return nil
	}
	return i.weave.device.CPUStatsUI()
}

func (i *Interface) WeaveMemoryStatsUI() *WeaveStatsSeries {
	if i == nil || !strings.EqualFold(i.Type, "WeaveInterface") || i.weave == nil || i.weave.device == nil {
		return nil
	}
	return i.weave.device.MemoryStatsUI()
}

func (i *Interface) WeaveEndpointQueues() map[string][][]byte {
	if i == nil || !strings.EqualFold(i.Type, "WeaveInterface") || i.weave == nil || i.weave.device == nil {
		return nil
	}
	return i.weave.device.EndpointQueues()
}

func (i *Interface) WeaveSetRemoteDisplay(enabled bool) {
	if i == nil || !strings.EqualFold(i.Type, "WeaveInterface") || i.weave == nil {
		return
	}
	i.weave.SetRemoteDisplay(enabled)
}

func (i *Interface) WeaveSendRemoteInput(payload []byte) {
	if i == nil || !strings.EqualFold(i.Type, "WeaveInterface") || i.weave == nil {
		return
	}
	i.weave.SendRemoteInput(payload)
}

func (i *Interface) WeaveRequestEndpointsList() {
	if i == nil || !strings.EqualFold(i.Type, "WeaveInterface") || i.weave == nil {
		return
	}
	i.weave.RequestEndpointsList()
}

func (i *Interface) WeaveViaSwitchID() *string {
	if i == nil || !strings.EqualFold(i.Type, "WeaveInterfacePeer") || i.Parent == nil || i.Parent.weave == nil || i.Parent.weave.device == nil {
		return nil
	}
	name := i.Name
	hexID := strings.TrimPrefix(name, "WeaveInterfacePeer[")
	hexID = strings.TrimSuffix(hexID, "]")
	if strings.TrimSpace(hexID) == "" {
		return nil
	}
	i.Parent.weave.device.mu.Lock()
	defer i.Parent.weave.device.mu.Unlock()
	ep := i.Parent.weave.device.endpoints[hexID]
	if ep == nil || strings.TrimSpace(ep.Via) == "" {
		return nil
	}
	s := ep.Via
	return &s
}

func (i *Interface) WeaveActiveTasks() map[string]WeaveTaskStat {
	if i == nil || !strings.EqualFold(i.Type, "WeaveInterface") || i.weave == nil || i.weave.device == nil {
		return nil
	}
	return i.weave.device.ActiveTasks(5 * time.Second)
}

func (i *Interface) IFACNetname() string {
	if i == nil {
		return ""
	}
	return i.IFACNetnameVal
}

func (i *Interface) IFACNetkey() string {
	if i == nil {
		return ""
	}
	return i.IFACNetkeyVal
}

func (i *Interface) SetAnnounceRateConfig(target, grace, penalty *int) {
	if i == nil {
		return
	}
	i.AnnounceRateTarget = target
	i.AnnounceRateGrace = grace
	i.AnnounceRatePenalty = penalty
}

func (i *Interface) SetAnnounceRate(_ *int, _ *int, _ *int) {
	// Python exposes this hook, but AnnounceCap/Target/Grace/Penalty are applied elsewhere.
}

func (i *Interface) OptimiseMTU() {
	if i == nil || !i.AutoconfigureMTU || i.FixedMTU {
		return
	}
	br := i.Bitrate
	prev := i.HWMTU
	switch {
	case br >= 1_000_000_000:
		i.HWMTU = 524288
	case br > 750_000_000:
		i.HWMTU = 262144
	case br > 400_000_000:
		i.HWMTU = 131072
	case br > 200_000_000:
		i.HWMTU = 65536
	case br > 100_000_000:
		i.HWMTU = 32768
	case br > 10_000_000:
		i.HWMTU = 16384
	case br > 5_000_000:
		i.HWMTU = 8192
	case br > 2_000_000:
		i.HWMTU = 4096
	case br > 1_000_000:
		i.HWMTU = 2048
	case br > 62_500:
		i.HWMTU = 1024
	default:
		i.HWMTU = 0
	}
	if DiagLogf != nil && prev != i.HWMTU {
		DiagLogf(LogDebug, "%s hardware MTU set to %d", i, i.HWMTU)
	}
}

func removeInterface(ifc *Interface) {
	if ifc == nil {
		return
	}
	if RemoveInterfaceHandler != nil {
		RemoveInterfaceHandler(ifc)
		return
	}
	ifc.Detach()
}

func (i *Interface) ProcessOutgoing(data []byte) {
	if i == nil || len(data) == 0 {
		return
	}
	if i.processOutgoingFn != nil {
		if err := i.processOutgoingFn(data); err != nil && DiagLogf != nil {
			DiagLogf(LogError, "external send error on %s: %v", i, err)
		}
		return
	}
	if strings.EqualFold(i.Type, "LocalInterface") {
		i.localMu.Lock()
		conn := i.localConn
		i.localMu.Unlock()
		if conn == nil {
			return
		}
		i.localSendMu.Lock()
		if i.ForceBitrateLatency && i.Bitrate > 0 {
			// Python: sleep(len(data)/bitrate*8) to simulate latency.
			secs := (float64(len(data)) * 8.0) / float64(i.Bitrate)
			if secs > 0 {
				time.Sleep(time.Duration(secs * float64(time.Second)))
			}
		}
		framed := append([]byte{hdlcFlag}, hdlcEscape(data)...)
		framed = append(framed, hdlcFlag)
		n, err := conn.Write(framed)
		i.localSendMu.Unlock()
		if err != nil || n != len(framed) {
			if DiagLogf != nil {
				DiagLogf(LogError, "Exception occurred while transmitting via %s, tearing down interface", i)
				if err != nil {
					DiagLogf(LogError, "The contained exception was: %v", err)
				} else {
					DiagLogf(LogError, "The contained exception was: short write (%d/%d)", n, len(framed))
				}
			}
			i.setLocalConn(nil)
			i.Online = false
			if i.LocalIsSharedClient {
				if DiagLogf != nil {
					DiagLogf(LogCritical, "Permanently lost connection to local shared RNS instance. Exiting now.")
				}
				if PanicOnInterfaceErrorProvider != nil && PanicOnInterfaceErrorProvider() {
					if PanicFunc != nil {
						PanicFunc()
					}
				}
				if ExitFunc != nil {
					ExitFunc()
				}
			}
			return
		}
		if n > 0 {
			atomic.AddUint64(&i.TXB, uint64(n))
			if parent := i.Parent; parent != nil {
				atomic.AddUint64(&parent.TXB, uint64(n))
			}
		}
		return
	}
	if strings.EqualFold(i.Type, "UDPInterface") {
		i.udpProcessOutgoing(data)
		return
	}
	if strings.EqualFold(i.Type, "AX25KISSInterface") && i.ax25 != nil {
		i.ax25.ProcessOutgoing(data)
		return
	}
	if strings.EqualFold(i.Type, "KISSInterface") && i.kiss != nil {
		i.kiss.ProcessOutgoing(data)
		return
	}
	if (strings.EqualFold(i.Type, "BackboneInterfacePeer") || strings.EqualFold(i.Type, "BackboneClientInterface")) && i.backbonePeer != nil {
		i.backbonePeer.ProcessOutgoing(data)
		return
	}
	if strings.EqualFold(i.Type, "I2PInterface") && i.i2pClient != nil {
		i.i2pClient.Send(data)
		return
	}
	if strings.EqualFold(i.Type, "I2PInterfacePeer") && i.i2pPeer != nil {
		i.i2pPeer.ProcessOutgoing(data)
		return
	}
	if strings.EqualFold(i.Type, "PipeInterface") && i.pipe != nil {
		i.pipe.ProcessOutgoing(data)
		return
	}
	if strings.EqualFold(i.Type, "SerialInterface") && i.serial != nil {
		i.serial.ProcessOutgoing(data)
		return
	}
	if strings.EqualFold(i.Type, "RNodeInterface") && i.rnodeSingle != nil {
		_ = i.rnodeSingle.SendData(data)
		atomic.AddUint64(&i.TXB, uint64(len(data)))
		return
	}
	if strings.EqualFold(i.Type, "RNodeSubInterface") {
		if i.Parent != nil && i.Parent.rnodeMulti != nil {
			i.Parent.rnodeMulti.ProcessOutgoing(i, data)
		}
		return
	}
	if strings.EqualFold(i.Type, "AutoInterfacePeer") {
		if i.Parent != nil && i.Parent.auto != nil {
			i.Parent.auto.sendToPeer(i, data)
		}
		return
	}
	if strings.EqualFold(i.Type, "WeaveInterfacePeer") {
		if i.Parent != nil && i.Parent.weave != nil {
			i.Parent.weave.ProcessOutgoing(i, data)
		}
		return
	}
	if strings.EqualFold(i.Type, "TCPClientInterface") && i.tcpClient != nil {
		if err := i.tcpClient.ProcessOutgoing(data); err != nil {
			if DiagLogf != nil {
				level := LogError
				if errors.Is(err, errTCPOfflineDetached) || errors.Is(err, errTCPConnNil) {
					level = LogWarning
				}
				DiagLogf(level, "TCP send error on %s: %v", i, err)
			}
			return
		}
		atomic.AddUint64(&i.TXB, uint64(len(data)))
		return
	}
}

// ProcessAnnounceRaw implements Python Interface.process_announce_queue behaviour:
// rate-limit outgoing announces according to announce_cap, preferring lowest hop count.
func (i *Interface) ProcessAnnounceRaw(raw []byte, hops int) {
	if i == nil || len(raw) == 0 {
		return
	}
	now := time.Now()

	cap := i.AnnounceCap
	if cap <= 0 && DefaultAnnounceCapProvider != nil {
		cap = DefaultAnnounceCapProvider()
	}

	if cap <= 0 {
		i.ProcessOutgoing(raw)
		i.SentAnnounce()
		return
	}

	i.icMu.Lock()
	i.announceQueue = append(i.announceQueue, announceQueueEntry{
		enqueuedAt: now,
		hops:       hops,
		raw:        append([]byte(nil), raw...),
	})
	alreadyRunning := i.announceRunning
	if !alreadyRunning {
		i.announceRunning = true
	}
	i.icMu.Unlock()

	if !alreadyRunning {
		go i.processAnnounceQueueLoop()
	}
}

func (i *Interface) processAnnounceQueueLoop() {
	defer func() {
		if r := recover(); r != nil {
			i.icMu.Lock()
			i.announceQueue = nil
			i.announceRunning = false
			i.icMu.Unlock()
			if DiagLogf != nil {
				DiagLogf(LogError, "Error while processing announce queue on %s. The contained exception was: %v", i, r)
				DiagLogf(LogError, "The announce queue for this interface has been cleared.")
			}
		}
	}()

	for {
		var (
			entry   announceQueueEntry
			hasWork bool
			waitFor time.Duration
		)

		now := time.Now()
		i.icMu.Lock()
		// TTL cleanup, like Python.
		if QueuedAnnounceLife > 0 && len(i.announceQueue) > 0 {
			kept := i.announceQueue[:0]
			for _, e := range i.announceQueue {
				if now.After(e.enqueuedAt.Add(QueuedAnnounceLife)) {
					continue
				}
				kept = append(kept, e)
			}
			i.announceQueue = kept
		}
		if len(i.announceQueue) == 0 {
			i.announceRunning = false
			i.icMu.Unlock()
			return
		}

		if !i.announceAllowedAt.IsZero() && now.Before(i.announceAllowedAt) {
			waitFor = time.Until(i.announceAllowedAt)
			i.icMu.Unlock()
			time.Sleep(waitFor)
			continue
		}

		// Select lowest-hop, oldest entry among those.
		selectedIdx := -1
		minHops := int(^uint(0) >> 1)
		var oldest time.Time
		for idx, e := range i.announceQueue {
			if e.hops < minHops {
				minHops = e.hops
				oldest = e.enqueuedAt
				selectedIdx = idx
			} else if e.hops == minHops && (selectedIdx < 0 || e.enqueuedAt.Before(oldest)) {
				oldest = e.enqueuedAt
				selectedIdx = idx
			}
		}
		if selectedIdx >= 0 {
			entry = i.announceQueue[selectedIdx]
			i.announceQueue = append(i.announceQueue[:selectedIdx], i.announceQueue[selectedIdx+1:]...)
			hasWork = true
		}
		i.icMu.Unlock()

		if !hasWork || len(entry.raw) == 0 {
			continue
		}

		cap := i.AnnounceCap
		if cap <= 0 && DefaultAnnounceCapProvider != nil {
			cap = DefaultAnnounceCapProvider()
		}
		if cap <= 0 {
			i.ProcessOutgoing(entry.raw)
			i.SentAnnounce()
			continue
		}

		br := i.Bitrate
		if br <= 0 {
			br = 62500
		}
		// tx_time = (len(raw)*8) / bitrate; wait_time = tx_time / announce_cap
		txTime := (float64(len(entry.raw)) * 8.0) / float64(br)
		waitTime := txTime / cap
		if waitTime < 0 {
			waitTime = 0
		}
		i.icMu.Lock()
		i.announceAllowedAt = time.Now().Add(time.Duration(waitTime * float64(time.Second)))
		i.icMu.Unlock()

		i.ProcessOutgoing(entry.raw)
		i.SentAnnounce()

		// Python uses a timer to re-run processing after wait_time if queue still has items.
		// Our loop continues, but sleep is governed by announceAllowedAt, which is equivalent.
	}
}

func (i *Interface) ReceivedAnnounce() {
	if i == nil {
		return
	}
	now := time.Now()
	i.icMu.Lock()
	i.iaFreq = append(i.iaFreq, now)
	if len(i.iaFreq) > IAFreqSamples {
		i.iaFreq = i.iaFreq[len(i.iaFreq)-IAFreqSamples:]
	}
	i.icMu.Unlock()
	if i.Parent != nil {
		i.Parent.ReceivedAnnounce()
	}
}

func (i *Interface) SentAnnounce() {
	if i == nil {
		return
	}
	now := time.Now()
	i.icMu.Lock()
	i.oaFreq = append(i.oaFreq, now)
	if len(i.oaFreq) > OAFreqSamples {
		i.oaFreq = i.oaFreq[len(i.oaFreq)-OAFreqSamples:]
	}
	i.icMu.Unlock()
	if i.Parent != nil {
		i.Parent.SentAnnounce()
	}
}

func (i *Interface) OutgoingAnnounceFrequency() float64 {
	if i == nil {
		return 0
	}
	i.icMu.Lock()
	defer i.icMu.Unlock()
	if len(i.oaFreq) <= 1 {
		return 0
	}
	delta := 0.0
	for idx := 1; idx < len(i.oaFreq); idx++ {
		delta += i.oaFreq[idx].Sub(i.oaFreq[idx-1]).Seconds()
	}
	delta += time.Since(i.oaFreq[len(i.oaFreq)-1]).Seconds()
	if delta <= 0 {
		return 0
	}
	return 1.0 / (delta / float64(len(i.oaFreq)))
}

func (i *Interface) incomingAnnounceFrequencyLocked(now time.Time) float64 {
	if len(i.iaFreq) <= 1 {
		return 0
	}
	delta := 0.0
	for idx := 1; idx < len(i.iaFreq); idx++ {
		delta += i.iaFreq[idx].Sub(i.iaFreq[idx-1]).Seconds()
	}
	delta += now.Sub(i.iaFreq[len(i.iaFreq)-1]).Seconds()
	if delta <= 0 {
		return 0
	}
	return 1.0 / (delta / float64(len(i.iaFreq)))
}

func (i *Interface) IncomingAnnounceFrequency() float64 {
	if i == nil {
		return 0
	}
	now := time.Now()
	i.icMu.Lock()
	defer i.icMu.Unlock()
	return i.incomingAnnounceFrequencyLocked(now)
}

func (i *Interface) shouldIngressLimitLocked(now time.Time) bool {
	if i == nil || !i.IngressControl {
		return false
	}

	newTimeSeconds := float64(ICNewTime)
	if i.ICNewTime != nil {
		newTimeSeconds = *i.ICNewTime
	}

	freqThreshold := ICBurstFreq
	if i.Age() < time.Duration(newTimeSeconds*float64(time.Second)) {
		freqThreshold = ICBurstFreqNew
		if i.ICBurstFreqNew != nil {
			freqThreshold = *i.ICBurstFreqNew
		}
	} else if i.ICBurstFreq != nil {
		freqThreshold = *i.ICBurstFreq
	}

	iaFreq := i.incomingAnnounceFrequencyLocked(now)

	if i.icBurstActive {
		hold := time.Duration(float64(time.Second) * float64(ICBurstHold))
		if i.ICBurstHold != nil {
			hold = time.Duration(float64(time.Second) * *i.ICBurstHold)
		}
		if iaFreq < freqThreshold && now.After(i.icBurstActivated.Add(hold)) {
			i.icBurstActive = false
			penalty := time.Duration(float64(time.Second) * float64(ICBurstPenalty))
			if i.ICBurstPenalty != nil {
				penalty = time.Duration(float64(time.Second) * *i.ICBurstPenalty)
			}
			i.icHeldRelease = now.Add(penalty)
		}
		return true
	}

	if iaFreq > freqThreshold {
		i.icBurstActive = true
		i.icBurstActivated = now
		return true
	}

	return false
}

func (i *Interface) ShouldIngressLimit() bool {
	if i == nil {
		return false
	}
	i.icMu.Lock()
	defer i.icMu.Unlock()
	return i.shouldIngressLimitLocked(time.Now())
}

func (i *Interface) HoldAnnounce(raw []byte, recvIf *Interface, destinationHash []byte, hops uint8) {
	if i == nil || len(raw) == 0 || len(destinationHash) == 0 {
		return
	}
	i.icMu.Lock()
	defer i.icMu.Unlock()
	if i.heldAnnounces == nil {
		i.heldAnnounces = make(map[string]heldAnnounce)
	}
	key := string(destinationHash)
	// Replace existing or add if under cap.
	if _, ok := i.heldAnnounces[key]; ok {
		i.heldAnnounces[key] = heldAnnounce{raw: raw, recvIf: recvIf, hops: hops, destHash: destinationHash}
		return
	}
	maxHeld := MaxHeldAnnounces
	if i.ICMaxHeldAnnounces != nil {
		maxHeld = *i.ICMaxHeldAnnounces
	}
	if len(i.heldAnnounces) >= maxHeld {
		return
	}
	i.heldAnnounces[key] = heldAnnounce{raw: raw, recvIf: recvIf, hops: hops, destHash: destinationHash}
}

func (i *Interface) ProcessHeldAnnounces(maxHops int) {
	if i == nil || InboundHandler == nil {
		return
	}
	now := time.Now()

	i.icMu.Lock()
	defer i.icMu.Unlock()

	if i.shouldIngressLimitLocked(now) {
		return
	}
	if len(i.heldAnnounces) == 0 {
		return
	}
	if !i.icHeldRelease.IsZero() && now.Before(i.icHeldRelease) {
		return
	}

	newTimeSeconds := float64(ICNewTime)
	if i.ICNewTime != nil {
		newTimeSeconds = *i.ICNewTime
	}
	freqThreshold := ICBurstFreq
	if i.Age() < time.Duration(newTimeSeconds*float64(time.Second)) {
		freqThreshold = ICBurstFreqNew
		if i.ICBurstFreqNew != nil {
			freqThreshold = *i.ICBurstFreqNew
		}
	} else if i.ICBurstFreq != nil {
		freqThreshold = *i.ICBurstFreq
	}
	if i.incomingAnnounceFrequencyLocked(now) >= freqThreshold {
		return
	}

	var selected *heldAnnounce
	var selectedKey string
	minH := maxHops
	for k, ap := range i.heldAnnounces {
		if int(ap.hops) < minH {
			minH = int(ap.hops)
			tmp := ap
			selected = &tmp
			selectedKey = k
		}
	}
	if selected == nil {
		return
	}

	interval := time.Duration(float64(time.Second) * float64(ICHeldReleaseInterval))
	if i.ICHeldReleaseInterval != nil {
		interval = time.Duration(float64(time.Second) * *i.ICHeldReleaseInterval)
	}
	i.icHeldRelease = now.Add(interval)
	delete(i.heldAnnounces, selectedKey)

	if DiagLogf != nil {
		DiagLogf(LogExtreme, "Releasing held announce packet %x (%d hops) from %s", selected.destHash, selected.hops, i)
	}

	raw := append([]byte(nil), selected.raw...)
	recvIf := selected.recvIf
	go InboundHandler(raw, recvIf)
}

func (i *Interface) setLocalConn(conn net.Conn) {
	i.localMu.Lock()
	old := i.localConn
	i.localConn = conn
	i.localMu.Unlock()
	if old != nil && old != conn {
		_ = old.Close()
	}
}

func (i *Interface) HasQueuedAnnounces() bool {
	if i == nil {
		return false
	}
	i.icMu.Lock()
	defer i.icMu.Unlock()
	return len(i.announceQueue) > 0 || i.announceRunning
}

func (i *Interface) AnnounceAllowedAt() time.Time {
	if i == nil {
		return time.Time{}
	}
	i.icMu.Lock()
	defer i.icMu.Unlock()
	return i.announceAllowedAt
}

func (i *Interface) HeldAnnouncesCount() int {
	if i == nil {
		return 0
	}
	i.icMu.Lock()
	defer i.icMu.Unlock()
	return len(i.heldAnnounces)
}

func (i *Interface) AnnounceQueueCount() int {
	if i == nil {
		return 0
	}
	i.icMu.Lock()
	defer i.icMu.Unlock()
	return len(i.announceQueue)
}

func (i *Interface) SetAnnounceAllowedAt(t time.Time) {
	if i == nil {
		return
	}
	i.icMu.Lock()
	i.announceAllowedAt = t
	i.icMu.Unlock()
}

func (i *Interface) readLocalFramesLoop() {
	i.localMu.Lock()
	conn := i.localConn
	i.localMu.Unlock()
	if conn == nil || InboundHandler == nil {
		return
	}
	buf := make([]byte, 4096)
	var frameBuf []byte
	for {
		n, err := conn.Read(buf)
		if err != nil || n <= 0 {
			return
		}
		frameBuf = append(frameBuf, buf[:n]...)
		for {
			start := indexByte(frameBuf, hdlcFlag)
			if start < 0 {
				frameBuf = nil
				break
			}
			end := indexByteFrom(frameBuf, hdlcFlag, start+1)
			if end < 0 {
				if start > 0 {
					frameBuf = frameBuf[start:]
				}
				break
			}
			frame := frameBuf[start+1 : end]
			frameBuf = frameBuf[end:]
			unescaped := hdlcUnescape(frame)
			if HeaderMinSize > 0 && len(unescaped) <= HeaderMinSize {
				continue
			}
			if len(unescaped) > 0 && len(unescaped) <= MaxFrameLength {
				atomic.AddUint64(&i.RXB, uint64(len(unescaped)))
				if parent := i.Parent; parent != nil {
					atomic.AddUint64(&parent.RXB, uint64(len(unescaped)))
				}
				InboundHandler(unescaped, i)
			}
		}
	}
}

func indexByte(b []byte, v byte) int {
	for i := range b {
		if b[i] == v {
			return i
		}
	}
	return -1
}

func indexByteFrom(b []byte, v byte, start int) int {
	for i := start; i < len(b); i++ {
		if b[i] == v {
			return i
		}
	}
	return -1
}
