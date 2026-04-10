package interfaces

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	errTCPOfflineDetached = errors.New("tcp iface offline/detached")
	errTCPConnNil         = errors.New("tcp conn is nil")
)

// ---- Framing ----
type HDLC struct{}

const (
	HDLC_FLAG     = 0x7E
	HDLC_ESC      = 0x7D
	HDLC_ESC_MASK = 0x20
)

func (HDLC) Escape(data []byte) []byte {
	out := make([]byte, 0, len(data)+8)
	for _, b := range data {
		switch b {
		case HDLC_ESC:
			out = append(out, HDLC_ESC, HDLC_ESC^HDLC_ESC_MASK)
		case HDLC_FLAG:
			out = append(out, HDLC_ESC, HDLC_FLAG^HDLC_ESC_MASK)
		default:
			out = append(out, b)
		}
	}
	return out
}

type KISS struct{}

const (
	KISS_FEND        = 0xC0
	KISS_FESC        = 0xDB
	KISS_TFEND       = 0xDC
	KISS_TFESC       = 0xDD
	KISS_CMD_DATA    = 0x00
	KISS_CMD_UNKNOWN = 0xFE
)

func (KISS) Escape(data []byte) []byte {
	out := make([]byte, 0, len(data)+8)
	for _, b := range data {
		switch b {
		case KISS_FESC:
			out = append(out, KISS_FESC, KISS_TFESC)
		case KISS_FEND:
			out = append(out, KISS_FESC, KISS_TFEND)
		default:
			out = append(out, b)
		}
	}
	return out
}

// ---- Common ----
type TCPLog interface {
	Debugf(string, ...any)
	Infof(string, ...any)
	Warnf(string, ...any)
	Errorf(string, ...any)
}

type diagTCPLog struct{}

func (diagTCPLog) Debugf(format string, args ...any) { //nolint:revive
	if DiagLogf != nil {
		DiagLogf(LogDebug, format, args...)
	}
}
func (diagTCPLog) Infof(format string, args ...any) { //nolint:revive
	if DiagLogf != nil {
		DiagLogf(LogInfo, format, args...)
	}
}
func (diagTCPLog) Warnf(format string, args ...any) { //nolint:revive
	if DiagLogf != nil {
		DiagLogf(LogWarning, format, args...)
	}
}
func (diagTCPLog) Errorf(format string, args ...any) { //nolint:revive
	if DiagLogf != nil {
		DiagLogf(LogError, format, args...)
	}
}

type TCPOwner interface {
	Inbound(data []byte, iface *TCPClientInterface)
}

const TCP_HW_MTU = 262144
const TCP_BITRATE_GUESS = 10 * 1000 * 1000
const TCP_DEFAULT_IFAC_SIZE = 16

// TCPTunnelSynthesizer can be set by the rns package (or callers) to mirror
// Python Transport.synthesize_tunnel(interface) behaviour for non-KISS TCP links.
// It is invoked after reconnecting a TCP initiator when WantsTunnel() is true.
var TCPTunnelSynthesizer func(iface *TCPClientInterface)

// ---- TCP Client Interface ----
type TCPClientInterface struct {
	Owner TCPOwner
	Log   TCPLog

	Name string

	TargetHost string
	TargetPort int

	KISSFraming bool
	I2PTunneled bool

	Initiator       bool
	ReconnectWait   time.Duration
	MaxReconnectTry *int // nil = infinite
	ConnectTimeout  time.Duration

	HWMTU          int
	Bitrate        int
	IFACSize       int
	IFACNetnameVal string
	IFACNetkeyVal  string
	IFACKey        []byte
	IFACIdentity   interface{ Sign([]byte) ([]byte, error) }
	IFACSignature  []byte

	AutoconfigureMTU bool
	FixedMTU         bool

	parent *TCPServerInterface

	mu         sync.Mutex
	conn       net.Conn
	writing    atomic.Bool
	online     atomic.Bool
	detached   atomic.Bool
	reconnect  atomic.Bool
	neverConn  atomic.Bool
	reading    atomic.Bool
	writeMutex sync.Mutex

	wantsTunnel atomic.Bool

	rxb atomic.Uint64
	txb atomic.Uint64

	iface *Interface
}

func NewTCPClientInitiator(owner TCPOwner, log TCPLog, name, host string, port int, kiss, i2p bool) *TCPClientInterface {
	iface := &TCPClientInterface{
		Owner:            owner,
		Log:              log,
		Name:             name,
		TargetHost:       host,
		TargetPort:       port,
		KISSFraming:      kiss,
		I2PTunneled:      i2p,
		Initiator:        true,
		ReconnectWait:    5 * time.Second,
		ConnectTimeout:   5 * time.Second,
		HWMTU:            TCP_HW_MTU,
		Bitrate:          TCP_BITRATE_GUESS,
		IFACSize:         TCP_DEFAULT_IFAC_SIZE,
		AutoconfigureMTU: true,
	}
	iface.neverConn.Store(true)
	return iface
}

func NewTCPClientFromAccepted(owner TCPOwner, log TCPLog, name string, c net.Conn, kiss, i2p bool) *TCPClientInterface {
	iface := &TCPClientInterface{
		Owner:            owner,
		Log:              log,
		Name:             name,
		KISSFraming:      kiss,
		I2PTunneled:      i2p,
		Initiator:        false,
		HWMTU:            TCP_HW_MTU,
		Bitrate:          TCP_BITRATE_GUESS,
		IFACSize:         TCP_DEFAULT_IFAC_SIZE,
		AutoconfigureMTU: true,
	}
	iface.setConn(c)
	iface.online.Store(true)
	iface.neverConn.Store(false)
	return iface
}

func (t *TCPClientInterface) OptimiseMTU() {
	if t == nil || !t.AutoconfigureMTU || t.FixedMTU {
		return
	}
	br := t.Bitrate
	switch {
	case br >= 1_000_000_000:
		t.HWMTU = 524288
	case br > 750_000_000:
		t.HWMTU = 262144
	case br > 400_000_000:
		t.HWMTU = 131072
	case br > 200_000_000:
		t.HWMTU = 65536
	case br > 100_000_000:
		t.HWMTU = 32768
	case br > 10_000_000:
		t.HWMTU = 16384
	case br > 5_000_000:
		t.HWMTU = 8192
	case br > 2_000_000:
		t.HWMTU = 4096
	case br > 1_000_000:
		t.HWMTU = 2048
	case br > 62_500:
		t.HWMTU = 1024
	default:
		t.HWMTU = 0
	}
}

func (t *TCPClientInterface) String() string {
	host := t.TargetHost
	if host == "" && t.conn != nil {
		host = t.conn.RemoteAddr().String()
	}
	return fmt.Sprintf("TCPInterface[%s/%s:%d]", t.Name, host, t.TargetPort)
}

func (t *TCPClientInterface) setConn(c net.Conn) {
	t.mu.Lock()
	t.conn = c
	t.mu.Unlock()

	// Nagle off
	if tc, ok := c.(*net.TCPConn); ok {
		_ = tc.SetNoDelay(true)
		_ = tc.SetKeepAlive(true)
		// python: probe after ~5s (tcp keepidle / keepalive)
		_ = tc.SetKeepAlivePeriod(5 * time.Second)
	}

	// Best-effort: Linux TCP_USER_TIMEOUT + keepalive params
	_ = setTCPTimeoutsBestEffort(c, t.I2PTunneled)
}

func (t *TCPClientInterface) getConn() net.Conn {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.conn
}

func (t *TCPClientInterface) Detach() {
	t.online.Store(false)
	t.detached.Store(true)

	c := t.getConn()
	if c != nil {
		_ = c.Close()
	}
	t.mu.Lock()
	t.conn = nil
	t.mu.Unlock()
}

func (t *TCPClientInterface) InitialConnect(ctx context.Context) {
	ok := t.connect(ctx, true)
	if !ok {
		go t.reconnectLoop()
		return
	}
	t.synthesizeTunnelIfNeeded()
	go t.readLoop()
	// Python: if not KISS, wants_tunnel = True (handled at a higher level).
}

func (t *TCPClientInterface) connect(ctx context.Context, initial bool) bool {
	if t.TargetHost == "" || t.TargetPort == 0 {
		return false
	}
	addr := net.JoinHostPort(t.TargetHost, fmt.Sprintf("%d", t.TargetPort))
	if initial && t.Log != nil {
		t.Log.Debugf("Establishing TCP connection for %s...", t.String())
	}

	d := net.Dialer{Timeout: t.ConnectTimeout}
	c, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		if initial && t.Log != nil {
			t.Log.Errorf("Initial connection for %s could not be established: %v", t.String(), err)
			t.Log.Errorf("Leaving unconnected and retrying connection in %s.", t.ReconnectWait)
		}
		t.online.Store(false)
		return false
	}

	t.setConn(c)
	t.online.Store(true)
	t.writing.Store(false)
	t.neverConn.Store(false)

	if initial && t.Log != nil {
		t.Log.Debugf("TCP connection for %s established", t.String())
	}
	if !t.KISSFraming {
		t.wantsTunnel.Store(true)
	}
	return true
}

func (t *TCPClientInterface) reconnectLoop() {
	if !t.Initiator {
		if t.Log != nil {
			t.Log.Errorf("Attempt to reconnect on a non-initiator TCP interface: %s", t.String())
		}
		return
	}
	if !t.reconnect.CompareAndSwap(false, true) {
		return
	}
	defer t.reconnect.Store(false)

	attempts := 0
	for !t.online.Load() && !t.detached.Load() {
		time.Sleep(t.ReconnectWait)
		attempts++

		if t.MaxReconnectTry != nil && attempts > *t.MaxReconnectTry {
			if t.Log != nil {
				t.Log.Errorf("Max reconnection attempts reached for %s", t.String())
			}
			t.teardown()
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), t.ConnectTimeout)
		ok := t.connect(ctx, false)
		cancel()
		if !ok {
			if t.Log != nil {
				t.Log.Debugf("Connection attempt for %s failed", t.String())
			}
			continue
		}
	}

	if !t.neverConn.Load() && t.Log != nil {
		t.Log.Infof("Reconnected socket for %s.", t.String())
	}

	// Python parity: if non-KISS framing is used, a tunnel may need to be
	// synthesized again after reconnect.
	t.synthesizeTunnelIfNeeded()

	go t.readLoop()
}

func (t *TCPClientInterface) synthesizeTunnelIfNeeded() {
	if t == nil || t.KISSFraming || !t.WantsTunnel() {
		return
	}

	called := false
	if TCPTunnelSynthesizer != nil {
		TCPTunnelSynthesizer(t)
		called = true
	} else if TunnelSynthesizer != nil && t.iface != nil {
		TunnelSynthesizer(t.iface)
		called = true
	}

	if called {
		if t.iface != nil {
			t.iface.WantsTunnel = false
		}
		t.wantsTunnel.Store(false)
	}
}

func (t *TCPClientInterface) ProcessIncoming(data []byte) {
	if !t.online.Load() || t.detached.Load() {
		return
	}
	t.rxb.Add(uint64(len(data)))
	if t.parent != nil {
		t.parent.rxb.Add(uint64(len(data)))
	}
	if t.Owner != nil && len(data) > 0 {
		cp := append([]byte(nil), data...)
		t.Owner.Inbound(cp, t)
	}
}

func (t *TCPClientInterface) ProcessOutgoing(data []byte) error {
	if !t.online.Load() || t.detached.Load() {
		t.startReconnectAsync()
		return errTCPOfflineDetached
	}

	c := t.getConn()
	if c == nil {
		t.startReconnectAsync()
		return errTCPConnNil
	}

	var framed []byte
	if t.KISSFraming {
		framed = make([]byte, 0, len(data)+4)
		framed = append(framed, KISS_FEND, KISS_CMD_DATA)
		framed = append(framed, KISS{}.Escape(data)...)
		framed = append(framed, KISS_FEND)
	} else {
		framed = make([]byte, 0, len(data)+2+8)
		framed = append(framed, HDLC_FLAG)
		framed = append(framed, HDLC{}.Escape(data)...)
		framed = append(framed, HDLC_FLAG)
	}

	t.writeMutex.Lock()
	defer t.writeMutex.Unlock()

	t.writing.Store(true)
	_, err := c.Write(framed) // net.Conn.Write typically writes all or returns an error
	t.writing.Store(false)

	if err != nil {
		if t.Log != nil {
			t.Log.Errorf("Exception occurred while transmitting via %s, tearing down interface", t.String())
			t.Log.Errorf("The contained exception was: %v", err)
		}
		t.teardown()
		t.startReconnectAsync()
		return err
	}

	t.txb.Add(uint64(len(framed)))
	if t.parent != nil {
		t.parent.txb.Add(uint64(len(framed)))
	}
	return nil
}

func (t *TCPClientInterface) WantsTunnel() bool {
	return t.wantsTunnel.Load()
}

func (t *TCPClientInterface) readLoop() {
	// Allow a new read loop after reconnect, but never run two in parallel.
	if !t.reading.CompareAndSwap(false, true) {
		return
	}
	defer func() {
		t.reading.Store(false)
		t.online.Store(false)
	}()

	c := t.getConn()
	if c == nil {
		return
	}

	buf := make([]byte, 4096)

	// --- KISS state ---
	inFrame := false
	escape := false
	command := byte(KISS_CMD_UNKNOWN)
	var dataBuf bytes.Buffer

	// --- HDLC buffer ---
	var frameBuf bytes.Buffer

	for {
		n, err := c.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) || t.detached.Load() {
				// like Python: socket closed
			} else if t.Log != nil {
				t.Log.Warnf("An interface error occurred for %s: %v", t.String(), err)
			}

			t.online.Store(false)
			if t.Initiator && !t.detached.Load() {
				if t.Log != nil {
					t.Log.Warnf("The socket for %s was closed, attempting to reconnect...", t.String())
				}
				t.reconnectLoop()
			} else {
				if t.Log != nil {
					t.Log.Debugf("The socket for remote client %s was closed.", t.String())
				}
				t.teardown()
			}
			return
		}

		if n == 0 {
			t.online.Store(false)
			if t.Initiator && !t.detached.Load() {
				if t.Log != nil {
					t.Log.Warnf("The socket for %s was closed, attempting to reconnect...", t.String())
				}
				t.reconnectLoop()
			} else {
				t.teardown()
			}
			return
		}

		chunk := buf[:n]

		if t.KISSFraming {
			// bytewise parser, like Python
			for i := 0; i < len(chunk); i++ {
				b := chunk[i]
				if inFrame && b == KISS_FEND && command == KISS_CMD_DATA {
					inFrame = false
					t.ProcessIncoming(dataBuf.Bytes())
					dataBuf.Reset()
					continue
				} else if b == KISS_FEND {
					inFrame = true
					command = KISS_CMD_UNKNOWN
					dataBuf.Reset()
					escape = false
					continue
				} else if inFrame && dataBuf.Len() < t.HWMTU {
					if dataBuf.Len() == 0 && command == KISS_CMD_UNKNOWN {
						// strip port nibble
						command = b & 0x0F
						continue
					}
					if command == KISS_CMD_DATA {
						if b == KISS_FESC {
							escape = true
							continue
						}
						if escape {
							if b == KISS_TFEND {
								b = KISS_FEND
							} else if b == KISS_TFESC {
								b = KISS_FESC
							}
							escape = false
						}
						dataBuf.WriteByte(b)
					}
				}
			}
		} else {
			// HDLC framing: accumulate and extract FLAG..FLAG
			frameBuf.Write(chunk)

			for {
				all := frameBuf.Bytes()
				start := bytes.IndexByte(all, HDLC_FLAG)
				if start < 0 {
					// nothing useful
					frameBuf.Reset()
					break
				}
				end := bytes.IndexByte(all[start+1:], HDLC_FLAG)
				if end < 0 {
					// wait for the second flag
					if start > 0 {
						frameBuf.Next(start) // drop garbage before the first flag
					}
					break
				}
				end = start + 1 + end
				frame := append([]byte(nil), all[start+1:end]...)

				// unescape like Python (replace in pairs)
				frame = bytes.ReplaceAll(frame, []byte{HDLC_ESC, HDLC_FLAG ^ HDLC_ESC_MASK}, []byte{HDLC_FLAG})
				frame = bytes.ReplaceAll(frame, []byte{HDLC_ESC, HDLC_ESC ^ HDLC_ESC_MASK}, []byte{HDLC_ESC})

				// Python: if len(frame) > HEADER_MINSIZE -> process
				if HeaderMinSize > 0 && len(frame) <= HeaderMinSize {
					// Drop processed bytes, otherwise we'll loop on the same frame.
					// Python: frame_buffer = frame_buffer[frame_end:]
					frameBuf.Next(end)
					continue
				}
				if len(frame) > 0 {
					t.ProcessIncoming(frame)
				}

				// truncate up to end (keep end as the new start, like Python frame_buffer = frame_buffer[frame_end:])
				frameBuf.Next(end)
			}
		}
	}
}

func (t *TCPClientInterface) teardown() {
	t.online.Store(false)
	c := t.getConn()
	if c != nil {
		_ = c.Close()
	}
	t.mu.Lock()
	t.conn = nil
	t.mu.Unlock()

	// if spawned by the server, remove from the list
	if t.parent != nil {
		t.parent.removeClient(t)
	}
}

func (t *TCPClientInterface) startReconnectAsync() {
	if t == nil || !t.Initiator || t.detached.Load() {
		return
	}
	if t.online.Load() {
		return
	}
	go t.reconnectLoop()
}

// ---- TCP Server Interface ----
type TCPServerInterface struct {
	Owner TCPOwner
	Log   TCPLog

	Name string

	ListenAddr  string // "ip:port" or "[ip%if]:port"
	I2PTunneled bool
	KISSFraming bool

	HWMTU    int
	FixedMTU bool
	Bitrate  int

	PreferIPv6 bool
	Device     string

	// IFAC parity (mirrors Python TCPServerInterface spawning behaviour).
	IFACSize       int
	IFACNetnameVal string
	IFACNetkeyVal  string
	IFACKey        []byte
	IFACIdentity   interface{ Sign([]byte) ([]byte, error) }
	IFACSignature  []byte

	ln       net.Listener
	online   atomic.Bool
	detached atomic.Bool

	mu      sync.Mutex
	clients []*TCPClientInterface

	// OnNewClient is an optional callback invoked before the spawned client
	// starts its read loop. It allows the rns package to wrap the connection
	// as a transport Interface without import cycles.
	OnNewClient func(ci *TCPClientInterface)

	rxb atomic.Uint64
	txb atomic.Uint64
}

func (t *TCPServerInterface) String() string {
	if t == nil {
		return "<nil>"
	}
	return fmt.Sprintf("TCPServerInterface[%s/%s]", t.Name, t.ListenAddr)
}

func NewTCPServer(owner TCPOwner, log TCPLog, name, listenAddr string, kiss, i2p bool) *TCPServerInterface {
	return &TCPServerInterface{
		Owner:       owner,
		Log:         log,
		Name:        name,
		ListenAddr:  listenAddr,
		I2PTunneled: i2p,
		KISSFraming: kiss,
		HWMTU:       TCP_HW_MTU,
		Bitrate:     TCP_BITRATE_GUESS,
		IFACSize:    TCP_DEFAULT_IFAC_SIZE,
	}
}

type TCPServerConfig struct {
	Name string

	Device     string
	ListenIP   string
	ListenPort int
	Port       int
	PreferIPv6 bool

	I2PTunneled bool
	KISSFraming bool

	FixedMTU int // 0 = default TCP_HW_MTU

	IFACSize    int
	IFACNetname string
	IFACNetkey  string
}

type TCPClientConfig struct {
	Name string

	TargetHost string
	TargetPort int
	Port       int

	I2PTunneled bool
	KISSFraming bool

	ReconnectWait   time.Duration
	MaxReconnectTry *int
	ConnectTimeout  time.Duration

	FixedMTU int // 0 = default TCP_HW_MTU

	IFACSize    int
	IFACNetname string
	IFACNetkey  string
}

type tcpOwnerFunc func([]byte, *TCPClientInterface)

func (f tcpOwnerFunc) Inbound(data []byte, iface *TCPClientInterface) { f(data, iface) }

// TCPIFACDeriver can be set by the rns package (or callers) to derive IFAC keying
// material for TCP interfaces, mirroring Python interface-level IFAC behaviour.
// It returns the IFAC key (64 bytes), the signing identity, plus the interface signature bytes.
var TCPIFACDeriver func(ifacNetname, ifacNetkey string) (ifacKey []byte, ifacIdentity interface{ Sign([]byte) ([]byte, error) }, ifacSignature []byte, err error)

func NewTCPClientInterfaceFromConfig(cfg TCPClientConfig) (*Interface, error) {
	if strings.TrimSpace(cfg.Name) == "" {
		return nil, errors.New("tcp client config missing name")
	}
	targetPort := cfg.TargetPort
	if targetPort == 0 {
		targetPort = cfg.Port
	}
	if targetPort <= 0 || targetPort > 65535 {
		return nil, fmt.Errorf("invalid target port: %d", targetPort)
	}
	if strings.TrimSpace(cfg.TargetHost) == "" {
		return nil, errors.New("tcp client config missing target_host")
	}

	ifc := &Interface{
		Name:              cfg.Name,
		Type:              "TCPClientInterface",
		IN:                true,
		OUT:               true,
		DriverImplemented: true,
		Online:            false,
		Bitrate:           TCP_BITRATE_GUESS,
		HWMTU:             TCP_HW_MTU,
		AutoconfigureMTU:  true,
	}
	// Python parity: non-KISS TCP interfaces request a tunnel by default.
	ifc.WantsTunnel = !cfg.KISSFraming
	// Match Python TCPInterface.DEFAULT_IFAC_SIZE.
	ifc.IFACSize = TCP_DEFAULT_IFAC_SIZE
	if cfg.FixedMTU > 0 {
		ifc.HWMTU = cfg.FixedMTU
		ifc.FixedMTU = true
		ifc.AutoconfigureMTU = false
	}
	if cfg.IFACSize > 0 {
		ifc.IFACSize = cfg.IFACSize
	}
	ifc.IFACNetnameVal = cfg.IFACNetname
	ifc.IFACNetkeyVal = cfg.IFACNetkey

	owner := tcpOwnerFunc(func(raw []byte, _ *TCPClientInterface) {
		if len(raw) == 0 {
			return
		}
		atomic.AddUint64(&ifc.RXB, uint64(len(raw)))
		if InboundHandler != nil {
			InboundHandler(raw, ifc)
		}
	})
	var log TCPLog
	if DiagLogf != nil {
		log = diagTCPLog{}
	}
	client := NewTCPClientInitiator(owner, log, cfg.Name, cfg.TargetHost, targetPort, cfg.KISSFraming, cfg.I2PTunneled)
	if cfg.ReconnectWait > 0 {
		client.ReconnectWait = cfg.ReconnectWait
	}
	if cfg.ConnectTimeout > 0 {
		client.ConnectTimeout = cfg.ConnectTimeout
	}
	client.MaxReconnectTry = cfg.MaxReconnectTry

	if (ifc.IFACNetnameVal != "" || ifc.IFACNetkeyVal != "") && TCPIFACDeriver != nil {
		key, id, sig, err := TCPIFACDeriver(ifc.IFACNetnameVal, ifc.IFACNetkeyVal)
		if err != nil {
			return nil, err
		}
		ifc.IFACKey = key
		ifc.IFACIdentity = id
		ifc.IFACSignature = sig
		client.IFACKey = key
		client.IFACIdentity = id
		client.IFACSignature = sig
	}

	ifc.SetTCPClient(client)
	ifc.OptimiseMTU()

	go func() {
		client.InitialConnect(context.Background())
		t := time.NewTicker(750 * time.Millisecond)
		defer t.Stop()
		for range t.C {
			if ifc.Detached {
				return
			}
			ifc.Online = client.online.Load() && !client.detached.Load()
		}
	}()

	return ifc, nil
}

// NewTCPServerFromConfig mirrors Python TCPServerInterface configuration semantics:
// - bind can be specified via kernel interface name ("device") or explicit host ("listen_ip")
// - prefer_ipv6 controls address family selection
// - port is an alias for listen_port
func NewTCPServerFromConfig(owner TCPOwner, log TCPLog, cfg TCPServerConfig) (*TCPServerInterface, error) {
	if cfg.Name == "" {
		return nil, errors.New("tcp server config missing name")
	}

	listenPort := cfg.ListenPort
	if listenPort == 0 {
		listenPort = cfg.Port
	}
	if listenPort <= 0 || listenPort > 65535 {
		return nil, fmt.Errorf("invalid listen port: %d", listenPort)
	}

	var listenAddr string
	if cfg.Device != "" {
		addr, err := tcpAddressForInterface(cfg.Device, listenPort, cfg.PreferIPv6)
		if err != nil {
			return nil, err
		}
		listenAddr = addr
	} else {
		if cfg.ListenIP == "" {
			return nil, errors.New("tcp server config missing listen_ip/device")
		}
		addr, err := tcpAddressForHost(cfg.ListenIP, listenPort, cfg.PreferIPv6)
		if err != nil {
			return nil, err
		}
		listenAddr = addr
	}

	s := NewTCPServer(owner, log, cfg.Name, listenAddr, cfg.KISSFraming, cfg.I2PTunneled)
	s.Device = cfg.Device
	s.PreferIPv6 = cfg.PreferIPv6

	if cfg.FixedMTU > 0 {
		s.HWMTU = cfg.FixedMTU
		s.FixedMTU = true
	}

	if cfg.IFACSize > 0 {
		s.IFACSize = cfg.IFACSize
	}
	s.IFACNetnameVal = cfg.IFACNetname
	s.IFACNetkeyVal = cfg.IFACNetkey

	if (s.IFACNetnameVal != "" || s.IFACNetkeyVal != "") && TCPIFACDeriver != nil {
		key, id, sig, err := TCPIFACDeriver(s.IFACNetnameVal, s.IFACNetkeyVal)
		if err != nil {
			return nil, err
		}
		s.IFACKey = key
		s.IFACIdentity = id
		s.IFACSignature = sig
	}

	return s, nil
}

func (s *TCPServerInterface) Clients() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.clients)
}

func (s *TCPServerInterface) Start() error {
	// Python parity: allow_reuse_address = True
	ln, err := listenWithReuseAddr("tcp", s.ListenAddr)
	if err != nil {
		return err
	}
	s.ln = ln
	s.online.Store(true)

	if s.Log != nil {
		s.Log.Infof("Listening on %s", s.String())
	}

	go s.acceptLoop()
	return nil
}

func (s *TCPServerInterface) Detach() {
	s.detached.Store(true)
	s.online.Store(false)
	if s.ln != nil {
		_ = s.ln.Close()
	}
	// close clients
	s.mu.Lock()
	cs := append([]*TCPClientInterface(nil), s.clients...)
	s.clients = nil
	s.mu.Unlock()

	for _, c := range cs {
		c.Detach()
	}
}

func (s *TCPServerInterface) acceptLoop() {
	for s.online.Load() && !s.detached.Load() {
		c, err := s.ln.Accept()
		if err != nil {
			if s.detached.Load() {
				return
			}
			if s.Log != nil {
				s.Log.Warnf("Accept error on %s: %v", s.String(), err)
			}
			continue
		}

		if s.Log != nil {
			s.Log.Debugf("Accepting incoming TCP connection")
		}

		ra := c.RemoteAddr().(*net.TCPAddr)
		iface := s.spawnClient(c, ra)

		if s.OnNewClient != nil {
			s.OnNewClient(iface)
		}

		s.mu.Lock()
		s.clients = append(s.clients, iface)
		s.mu.Unlock()

		go iface.readLoop()
	}
}

func (s *TCPServerInterface) spawnClient(c net.Conn, ra *net.TCPAddr) *TCPClientInterface {
	iface := NewTCPClientFromAccepted(s.Owner, s.Log, "Client on "+s.Name, c, s.KISSFraming, s.I2PTunneled)
	iface.parent = s
	if ra != nil {
		iface.TargetHost = ra.IP.String()
		iface.TargetPort = ra.Port
	}
	iface.HWMTU = s.HWMTU
	iface.FixedMTU = s.FixedMTU
	iface.AutoconfigureMTU = !s.FixedMTU
	iface.Bitrate = s.Bitrate
	iface.OptimiseMTU()

	iface.IFACSize = s.IFACSize
	iface.IFACNetnameVal = s.IFACNetnameVal
	iface.IFACNetkeyVal = s.IFACNetkeyVal
	iface.IFACKey = append([]byte(nil), s.IFACKey...)
	iface.IFACIdentity = s.IFACIdentity
	iface.IFACSignature = append([]byte(nil), s.IFACSignature...)
	return iface
}

func (s *TCPServerInterface) removeClient(ci *TCPClientInterface) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.clients[:0]
	for _, c := range s.clients {
		if c != ci {
			out = append(out, c)
		}
	}
	s.clients = out
}

// ---- Best-effort socket options ----
