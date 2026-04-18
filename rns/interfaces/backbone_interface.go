package interfaces

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	vendor "github.com/svanichkin/go-reticulum/rns/vendor"
)

const (
	backboneDefaultHWMTU       = 1 << 20
	backboneBitrateGuess       = 1_000_000_000
	backboneClientBitrateGuess = 100_000_000
)

type backboneServerConfig struct {
	Name       string
	Listen     string
	Port       int
	Device     string
	PreferIPv6 bool
}

type BackboneInterfaceDriver struct {
	iface   *Interface
	cfg     backboneServerConfig
	lns     []net.Listener
	stopCh  chan struct{}
	clients sync.Map
}

var (
	backboneDriversMu sync.Mutex
	backboneDrivers   = make(map[*BackboneInterfaceDriver]struct{})
)

type backboneClientConfig struct {
	Name           string
	TargetHost     string
	TargetPort     int
	ConnectTimeout time.Duration
	ReconnectWait  time.Duration
	MaxReconnect   int
	PreferIPv6     bool
	I2PTunneled    bool
}

type BackboneClientDriver struct {
	iface  *Interface
	cfg    backboneClientConfig
	stopCh chan struct{}

	neverConnected atomic.Bool
}

type BackbonePeer struct {
	parent    *BackboneInterfaceDriver
	iface     *Interface
	conn      net.Conn
	sendOnce  sync.Once
	txMu      sync.Mutex
	txCond    *sync.Cond
	txQueue   []backboneTxFrame
	txBytes   int
	closed    atomic.Bool
	onClose   func()
	closeOnce sync.Once
}

type backboneTxFrame struct {
	frame      []byte
	payloadLen int
}

const backboneTXMaxBytes = 4 << 20

// Python checks `attempts > max_reconnect_tries` where attempts starts at 0 and
// is incremented before each retry (after the initial connect attempt).
func backboneExceededMaxReconnect(attempts, maxReconnect int) bool {
	return maxReconnect > 0 && attempts > maxReconnect
}

const backboneWriteTimeout = 200 * time.Millisecond
const backboneWriteBackoff = 10 * time.Millisecond
const backboneWriteBackoffMax = 250 * time.Millisecond

func NewBackboneInterface(name string, kv map[string]string) (*Interface, error) {
	// Python only supports BackboneInterface on Linux-based OSes.
	if !vendor.IsLinux() && !vendor.IsAndroid() {
		return nil, errors.New("BackboneInterface is only supported on Linux-based operating systems")
	}

	cfg, err := parseBackboneConfig(name, kv)
	if err != nil {
		return nil, err
	}
	var targets []listenTarget
	if cfg.Listen != "" {
		targets, err = selectListenTargets(cfg.Listen, cfg.Port, cfg.PreferIPv6)
	} else {
		targets, err = selectListenTargetsForDevice(cfg.Device, cfg.Port, cfg.PreferIPv6)
	}
	if err != nil {
		return nil, err
	}
	lns := make([]net.Listener, 0, len(targets))
	for _, t := range targets {
		if t.network == "unix" {
			if err := removeStaleUnixSocket(t.addr); err != nil {
				for _, l := range lns {
					_ = l.Close()
				}
				return nil, err
			}
		}
		ln, err := listenWithReuseAddr(t.network, t.addr)
		if err != nil {
			for _, l := range lns {
				_ = l.Close()
			}
			return nil, fmt.Errorf("backbone listen failed (%s %s): %w", t.network, t.addr, err)
		}
		lns = append(lns, ln)
	}

	iface := &Interface{
		Name:              name,
		Type:              "BackboneInterface",
		IN:                true,
		OUT:               false,
		DriverImplemented: true,
		Bitrate:           backboneBitrateGuess,
		HWMTU:             backboneDefaultHWMTU,
		AutoconfigureMTU:  true,
	}
	// Match Python DEFAULT_IFAC_SIZE.
	if iface.IFACSize == 0 {
		iface.IFACSize = 16
	}

	driver := &BackboneInterfaceDriver{
		iface:  iface,
		cfg:    cfg,
		lns:    lns,
		stopCh: make(chan struct{}),
	}
	registerBackboneDriver(driver)
	iface.backboneServer = driver
	iface.clientCount = driver.ClientCount
	iface.Online = true

	if DiagLogf != nil {
		for _, t := range targets {
			DiagLogf(LogInfo, "BackboneInterface %s listening on %s (%s)", name, t.addr, t.network)
		}
	}

	for _, ln := range driver.lns {
		go driver.acceptLoop(ln)
	}
	return iface, nil
}

func parseBackboneConfig(name string, kv map[string]string) (backboneServerConfig, error) {
	get := func(key string) string {
		if kv == nil {
			return ""
		}
		for k, v := range kv {
			if strings.EqualFold(k, key) {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}

	cfg := backboneServerConfig{
		Name:       name,
		Listen:     stripIPv6Brackets(get("listen_ip")),
		Port:       parseIntOr(get("listen_port"), 0),
		Device:     get("device"),
		PreferIPv6: parseBoolOr(get("prefer_ipv6"), false),
	}
	if cfg.Listen == "" {
		cfg.Listen = stripIPv6Brackets(get("bind_ip"))
	}
	if cfg.Port == 0 {
		cfg.Port = parseIntOr(get("port"), 0)
	}
	if cfg.Listen == "" {
		if cfg.Device != "" {
			// Defer address selection to listener setup so we can multi-listen
			// on both IPv4 and IPv6 addresses when available.
		}
	}
	if cfg.Port == 0 {
		if !isUnixListen(cfg.Listen) {
			return cfg, errors.New("BackboneInterface requires listen_port")
		}
	}
	if cfg.Listen == "" && cfg.Device == "" {
		return cfg, fmt.Errorf("BackboneInterface %q requires listen_ip or device", name)
	}
	return cfg, nil
}

func stripIPv6Brackets(host string) string {
	host = strings.TrimSpace(host)
	if len(host) >= 2 && strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		return host[1 : len(host)-1]
	}
	return host
}

func isUnixListen(listen string) bool {
	s := strings.TrimSpace(listen)
	if s == "" {
		return false
	}
	if strings.HasPrefix(strings.ToLower(s), "unix:") {
		return true
	}
	return strings.HasPrefix(s, "/")
}

func removeStaleUnixSocket(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("missing unix socket path")
	}
	st, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if st.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("refusing to remove non-socket file at %q", path)
	}
	return os.Remove(path)
}

func unixListenPath(listen string) (string, bool) {
	s := strings.TrimSpace(listen)
	if strings.HasPrefix(strings.ToLower(s), "unix:") {
		path := strings.TrimSpace(s[len("unix:"):])
		return path, path != ""
	}
	if strings.HasPrefix(s, "/") {
		return s, true
	}
	return "", false
}

func (d *BackboneInterfaceDriver) Close() {
	select {
	case <-d.stopCh:
		return
	default:
		close(d.stopCh)
		unregisterBackboneDriver(d)
		for _, ln := range d.lns {
			if ln != nil {
				_ = ln.Close()
			}
		}
		d.clients.Range(func(key, value any) bool {
			if peer, ok := key.(*BackbonePeer); ok {
				peer.Close()
			}
			return true
		})
	}
}

func registerBackboneDriver(d *BackboneInterfaceDriver) {
	if d == nil {
		return
	}
	backboneDriversMu.Lock()
	backboneDrivers[d] = struct{}{}
	backboneDriversMu.Unlock()
}

func unregisterBackboneDriver(d *BackboneInterfaceDriver) {
	if d == nil {
		return
	}
	backboneDriversMu.Lock()
	delete(backboneDrivers, d)
	backboneDriversMu.Unlock()
}

func (d *BackboneInterfaceDriver) CloseListeners() {
	select {
	case <-d.stopCh:
		return
	default:
		close(d.stopCh)
		unregisterBackboneDriver(d)
		for _, ln := range d.lns {
			if ln != nil {
				_ = ln.Close()
			}
		}
	}
}

// DeregisterListeners mirrors Python BackboneInterface.deregister_listeners().
func DeregisterListeners() {
	backboneDriversMu.Lock()
	drivers := make([]*BackboneInterfaceDriver, 0, len(backboneDrivers))
	for driver := range backboneDrivers {
		drivers = append(drivers, driver)
	}
	backboneDriversMu.Unlock()
	for _, driver := range drivers {
		driver.CloseListeners()
	}
}

func (d *BackboneInterfaceDriver) acceptLoop(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-d.stopCh:
				return
			default:
				if DiagLogf != nil {
					DiagLogf(LogDebug, "Backbone accept error on %s: %v", d.iface.Name, err)
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}
		}
		if err := d.handleConn(conn); err != nil {
			if DiagLogf != nil {
				DiagLogf(LogDebug, "Backbone handleConn error on %s: %v", d.iface.Name, err)
			}
			_ = conn.Close()
		}
	}
}

func (d *BackboneInterfaceDriver) handleConn(conn net.Conn) error {
	if InboundHandler == nil {
		return errors.New("inbound handler not ready")
	}
	tuneTCPBackbone(conn)
	peerName := fmt.Sprintf("BackboneInterfacePeer[%s/%s]", d.iface.Name, conn.RemoteAddr())
	peerIface := &Interface{
		Name:                  peerName,
		Type:                  "BackboneInterfacePeer",
		Parent:                d.iface,
		IN:                    d.iface.IN,
		OUT:                   d.iface.OUT, // Python spawns peers with OUT=self.OUT (False)
		DriverImplemented:     true,
		Bitrate:               d.iface.Bitrate,
		HWMTU:                 d.iface.HWMTU,
		Mode:                  d.iface.Mode,
		IngressControl:        d.iface.IngressControl,
		ICMaxHeldAnnounces:    d.iface.ICMaxHeldAnnounces,
		ICBurstHold:           d.iface.ICBurstHold,
		ICBurstFreqNew:        d.iface.ICBurstFreqNew,
		ICBurstFreq:           d.iface.ICBurstFreq,
		ICNewTime:             d.iface.ICNewTime,
		ICBurstPenalty:        d.iface.ICBurstPenalty,
		ICHeldReleaseInterval: d.iface.ICHeldReleaseInterval,
		AnnounceCap:           d.iface.AnnounceCap,
		AnnounceRateTarget:    d.iface.AnnounceRateTarget,
		AnnounceRateGrace:     d.iface.AnnounceRateGrace,
		AnnounceRatePenalty:   d.iface.AnnounceRatePenalty,
		IFACSize:              d.iface.IFACSize,
		IFACNetnameVal:        d.iface.IFACNetnameVal,
		IFACNetkeyVal:         d.iface.IFACNetkeyVal,
		IFACKey:               d.iface.IFACKey,
		IFACIdentity:          d.iface.IFACIdentity,
		IFACSignature:         d.iface.IFACSignature,
	}
	peer := &BackbonePeer{
		parent: d,
		iface:  peerIface,
		conn:   conn,
	}
	peer.ensureWriter()
	peer.onClose = func() {
		d.clients.Delete(peer)
	}
	peerIface.backbonePeer = peer

	d.clients.Store(peer, struct{}{})
	if SpawnHandler != nil {
		SpawnHandler(peerIface)
	}
	if DiagLogf != nil {
		DiagLogf(LogVerbose, "Spawned new Backbone client: %s", peerName)
	}

	go peer.readLoop()
	return nil
}

func (d *BackboneInterfaceDriver) ClientCount() int {
	count := 0
	d.clients.Range(func(key, value any) bool {
		count++
		return true
	})
	return count
}

func (p *BackbonePeer) Close() {
	if !p.closed.CompareAndSwap(false, true) {
		return
	}
	ifc := p.iface
	p.txMu.Lock()
	if p.txCond != nil {
		p.txCond.Broadcast()
	}
	p.txMu.Unlock()
	if p.conn != nil {
		_ = p.conn.Close()
	}
	if p.iface != nil {
		p.iface.Online = false
		p.iface.backbonePeer = nil
	}
	p.closeOnce.Do(func() {
		if p.onClose != nil {
			p.onClose()
		}
		// Server-spawned peer interfaces should be removed from transport.
		if p.parent != nil && ifc != nil {
			removeInterface(ifc)
		}
	})
}

func (p *BackbonePeer) ProcessOutgoing(data []byte) {
	if p == nil || p.closed.Load() || len(data) == 0 {
		return
	}
	frame := append([]byte{hdlcFlag}, hdlcEscape(data)...)
	frame = append(frame, hdlcFlag)

	p.ensureWriter()
	p.txMu.Lock()
	defer p.txMu.Unlock()
	if p.closed.Load() {
		return
	}
	need := len(frame)
	deadline := time.Now().Add(200 * time.Millisecond)
	for p.txBytes+need > backboneTXMaxBytes && !p.closed.Load() {
		if time.Now().After(deadline) {
			if DiagLogf != nil && p.iface != nil {
				DiagLogf(LogWarning, "Backbone TX buffer full for %s, dropping %d bytes", p.iface.Name, len(data))
			}
			return
		}
		p.txCond.Wait()
	}
	if p.closed.Load() {
		return
	}
	p.txQueue = append(p.txQueue, backboneTxFrame{frame: frame, payloadLen: len(data)})
	p.txBytes += need
	p.txCond.Signal()
}

func (p *BackbonePeer) ensureWriter() {
	p.sendOnce.Do(func() {
		p.txCond = sync.NewCond(&p.txMu)
		go p.writeLoop()
	})
}

func (p *BackbonePeer) writeLoop() {
	for {
		timeoutBackoff := backboneWriteBackoff

		p.txMu.Lock()
		for len(p.txQueue) == 0 && !p.closed.Load() {
			p.txCond.Wait()
		}
		if p.closed.Load() {
			p.txMu.Unlock()
			return
		}
		tx := p.txQueue[0]
		p.txQueue = p.txQueue[1:]
		p.txBytes -= len(tx.frame)
		p.txCond.Signal()
		p.txMu.Unlock()

		if p.closed.Load() || p.conn == nil || p.iface == nil {
			return
		}
		frame := tx.frame
		for len(frame) > 0 {
			_ = p.conn.SetWriteDeadline(time.Now().Add(backboneWriteTimeout))
			n, err := p.conn.Write(frame)
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					time.Sleep(timeoutBackoff)
					timeoutBackoff *= 2
					if timeoutBackoff > backboneWriteBackoffMax {
						timeoutBackoff = backboneWriteBackoffMax
					}
					continue
				}
				if DiagLogf != nil && p.iface != nil {
					DiagLogf(LogDebug, "Backbone write error on %s: %v", p.iface.Name, err)
				}
				p.Close()
				return
			}
			if n <= 0 {
				p.Close()
				return
			}
			frame = frame[n:]
			timeoutBackoff = backboneWriteBackoff
		}
		atomic.AddUint64(&p.iface.TXB, uint64(tx.payloadLen))
		if parent := p.iface.Parent; parent != nil {
			atomic.AddUint64(&parent.TXB, uint64(tx.payloadLen))
		}
	}
}

func (p *BackbonePeer) readLoop() {
	defer p.Close()
	readSize := 4096
	if p.iface != nil && p.iface.HWMTU > 0 && p.iface.HWMTU < 1<<20 {
		readSize = p.iface.HWMTU
	} else if p.iface != nil && p.iface.HWMTU >= 1<<20 {
		readSize = 1 << 20
	}
	buf := make([]byte, readSize)
	var frameBuf []byte
	for {
		n, err := p.conn.Read(buf)
		if err != nil || n <= 0 {
			if DiagLogf != nil && p.iface != nil && err != nil {
				DiagLogf(LogDebug, "Backbone read error on %s: %v", p.iface.Name, err)
			}
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
			payload := hdlcUnescape(frame)
			if len(payload) == 0 || len(payload) > MaxFrameLength {
				continue
			}
			atomic.AddUint64(&p.iface.RXB, uint64(len(payload)))
			if parent := p.iface.Parent; parent != nil {
				atomic.AddUint64(&parent.RXB, uint64(len(payload)))
			}
			if InboundHandler != nil {
				InboundHandler(payload, p.iface)
			}
		}
	}
}

func NewBackboneClientInterface(name string, kv map[string]string) (*Interface, error) {
	if !vendor.IsLinux() && !vendor.IsAndroid() {
		return nil, errors.New("BackboneClientInterface is only supported on Linux-based operating systems")
	}

	cfg, err := parseBackboneClientConfig(name, kv)
	if err != nil {
		return nil, err
	}

	iface := &Interface{
		Name:              name,
		Type:              "BackboneClientInterface",
		IN:                true,
		OUT:               false, // Python BackboneClientInterface.OUT = False
		DriverImplemented: true,
		Bitrate:           backboneClientBitrateGuess,
		HWMTU:             backboneDefaultHWMTU,
		AutoconfigureMTU:  true,
	}
	// Match Python DEFAULT_IFAC_SIZE.
	if iface.IFACSize == 0 {
		iface.IFACSize = 16
	}

	driver := &BackboneClientDriver{
		iface:  iface,
		cfg:    cfg,
		stopCh: make(chan struct{}),
	}
	driver.neverConnected.Store(true)
	iface.backboneClient = driver

	// Python starts with a synchronous initial_connect.
	conn, err := driver.dial(true)
	if err != nil {
		go driver.run()
		return iface, nil
	}
	peer := &BackbonePeer{
		iface: iface,
		conn:  conn,
	}
	peer.ensureWriter()
	done := make(chan struct{})
	peer.onClose = func() {
		close(done)
	}
	iface.backbonePeer = peer
	iface.Online = true
	driver.neverConnected.Store(false)
	go peer.readLoop()

	// After first successful connect, monitor for disconnect and then continue reconnect loop.
	go func() {
		select {
		case <-driver.stopCh:
			peer.Close()
			<-done
			return
		case <-done:
			iface.Online = false
			go driver.run()
			return
		}
	}()

	return iface, nil
}

func parseBackboneClientConfig(name string, kv map[string]string) (backboneClientConfig, error) {
	get := func(key string) string {
		if kv == nil {
			return ""
		}
		for k, v := range kv {
			if strings.EqualFold(k, key) {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}

	cfg := backboneClientConfig{
		Name:           name,
		TargetHost:     get("target_host"),
		TargetPort:     parseIntOr(get("target_port"), 0),
		ConnectTimeout: time.Duration(parseIntOr(get("connect_timeout"), 5)) * time.Second,
		ReconnectWait:  time.Duration(parseIntOr(get("reconnect_interval"), 5)) * time.Second,
		MaxReconnect:   parseIntOr(get("max_reconnect_tries"), 0),
		PreferIPv6:     parseBoolOr(get("prefer_ipv6"), false),
		I2PTunneled:    parseBoolOr(get("i2p_tunneled"), false),
	}
	if cfg.TargetHost == "" {
		cfg.TargetHost = get("host")
	}
	if cfg.TargetPort == 0 {
		cfg.TargetPort = parseIntOr(get("port"), 0)
	}
	if cfg.TargetHost == "" || cfg.TargetPort == 0 {
		return cfg, errors.New("BackboneClientInterface requires target_host and target_port")
	}
	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = 5 * time.Second
	}
	if cfg.ReconnectWait <= 0 {
		cfg.ReconnectWait = 5 * time.Second
	}
	return cfg, nil
}

func (d *BackboneClientDriver) run() {
	attempts := 0
	for {
		select {
		case <-d.stopCh:
			return
		default:
		}

		conn, err := d.dial(true)
		if err != nil {
			if attempts == 0 && DiagLogf != nil {
				DiagLogf(LogError, "Initial connection for %s could not be established: %v", d.iface.Name, err)
				DiagLogf(LogError, "Leaving unconnected and retrying connection in %s.", d.cfg.ReconnectWait)
			} else if DiagLogf != nil {
				DiagLogf(LogDebug, "Connection attempt for %s failed: %v", d.iface.Name, err)
			}
			if !d.waitReconnect() {
				return
			}
			attempts++
			if backboneExceededMaxReconnect(attempts, d.cfg.MaxReconnect) {
				if DiagLogf != nil {
					DiagLogf(LogError, "Max reconnection attempts reached for %s", d.iface.Name)
				}
				d.teardown()
				return
			}
			continue
		}
		attempts = 0

		peer := &BackbonePeer{
			iface: d.iface,
			conn:  conn,
		}
		peer.ensureWriter()
		done := make(chan struct{})
		peer.onClose = func() {
			close(done)
		}

		d.iface.backbonePeer = peer
		d.iface.Online = true
		if !d.neverConnected.Load() && DiagLogf != nil {
			DiagLogf(LogInfo, "Reconnected socket for %s.", d.iface.Name)
		}
		// Mirror Python BackboneClientInterface.reconnect(): synthesize tunnel after reconnect.
		if !d.neverConnected.Load() && TunnelSynthesizer != nil {
			TunnelSynthesizer(d.iface)
		}
		d.neverConnected.Store(false)
		if DiagLogf != nil {
			DiagLogf(LogDebug, "TCP connection for %s established", d.iface.Name)
		}

		go peer.readLoop()

		select {
		case <-d.stopCh:
			peer.Close()
			<-done
			return
		case <-done:
			d.iface.Online = false
			if DiagLogf != nil {
				DiagLogf(LogWarning, "The socket for %s was closed, attempting to reconnect...", d.iface.Name)
			}
			if !d.waitReconnect() {
				return
			}
		}
	}
}

func (d *BackboneClientDriver) teardown() {
	if d == nil || d.iface == nil {
		return
	}
	// Match Python BackboneClientInterface.teardown(): mark unusable but keep the
	// interface instance (do not remove from transport automatically).
	d.iface.Online = false
	d.iface.IN = false
	d.iface.OUT = false
	d.Close()
}

func (d *BackboneClientDriver) waitReconnect() bool {
	select {
	case <-d.stopCh:
		return false
	case <-time.After(d.cfg.ReconnectWait):
		return true
	}
}

func (d *BackboneClientDriver) dial(initial bool) (net.Conn, error) {
	addr, network, err := selectDialTarget(d.cfg.TargetHost, d.cfg.TargetPort, d.cfg.PreferIPv6)
	if err != nil {
		return nil, err
	}
	dialer := &net.Dialer{Timeout: d.cfg.ConnectTimeout}
	if initial && DiagLogf != nil {
		DiagLogf(LogDebug, "Establishing TCP connection for %s...", d.iface.Name)
	}
	conn, err := backboneDial(dialer, network, addr)
	if err != nil {
		return nil, err
	}
	// Python BackboneClientInterface does not vary TCP tuning based on i2p_tunneled.
	tuneTCPBackbone(conn)
	return conn, nil
}

func selectDialTarget(host string, port int, preferIPv6 bool) (addr string, network string, err error) {
	host = stripIPv6Brackets(host)
	if strings.TrimSpace(host) == "" || port <= 0 {
		return "", "", errors.New("invalid target host/port")
	}
	network = "tcp"

	// Literal IPv4.
	if ip := net.ParseIP(host); ip != nil && ip.To4() != nil {
		return net.JoinHostPort(host, strconv.Itoa(port)), "tcp4", nil
	}

	// Literal IPv6 (possibly scoped).
	if strings.Contains(host, ":") {
		return net.JoinHostPort(host, strconv.Itoa(port)), "tcp6", nil
	}

	// Hostname: resolve and pick family like Python getaddrinfo loop.
	addrs, err := net.DefaultResolver.LookupIPAddr(context.Background(), host)
	if err != nil || len(addrs) == 0 {
		// Fall back to system default.
		return net.JoinHostPort(host, strconv.Itoa(port)), "tcp", err
	}
	picked := pickPreferredIPAddr(addrs, preferIPv6)
	if picked == nil {
		return net.JoinHostPort(host, strconv.Itoa(port)), "tcp", err
	}
	if picked.IP.To4() == nil {
		network = "tcp6"
	} else {
		network = "tcp4"
	}
	return net.JoinHostPort(picked.String(), strconv.Itoa(port)), network, nil
}

func selectListenTarget(host string, port int, preferIPv6 bool) (addr string, network string, err error) {
	host = stripIPv6Brackets(host)
	if strings.TrimSpace(host) == "" || port <= 0 {
		return "", "", errors.New("invalid listen host/port")
	}
	network = "tcp"

	// Literal IPv4.
	if ip := net.ParseIP(host); ip != nil && ip.To4() != nil {
		return net.JoinHostPort(host, strconv.Itoa(port)), "tcp4", nil
	}

	// Literal IPv6 (possibly scoped).
	if strings.Contains(host, ":") {
		return net.JoinHostPort(host, strconv.Itoa(port)), "tcp6", nil
	}

	// Hostname: resolve and pick family like Python getaddrinfo loop.
	addrs, err := net.DefaultResolver.LookupIPAddr(context.Background(), host)
	if err != nil || len(addrs) == 0 {
		return net.JoinHostPort(host, strconv.Itoa(port)), "tcp", err
	}
	picked := pickPreferredIPAddr(addrs, preferIPv6)
	if picked == nil {
		return net.JoinHostPort(host, strconv.Itoa(port)), "tcp", err
	}
	if picked.IP.To4() == nil {
		network = "tcp6"
	} else {
		network = "tcp4"
	}
	return net.JoinHostPort(picked.String(), strconv.Itoa(port)), network, nil
}

type listenTarget struct {
	network string
	addr    string
}

func selectListenTargets(host string, port int, preferIPv6 bool) ([]listenTarget, error) {
	host = strings.TrimSpace(host)

	if path, ok := unixListenPath(host); ok {
		return []listenTarget{{network: "unix", addr: path}}, nil
	}

	addr, network, err := selectListenTarget(host, port, preferIPv6)
	if err != nil {
		return nil, err
	}

	trimmed := stripIPv6Brackets(host)
	// Literal IPs (incl IPv6 scoped) only yield a single listener.
	if ip := net.ParseIP(trimmed); ip != nil || strings.Contains(trimmed, ":") {
		return []listenTarget{{network: network, addr: addr}}, nil
	}

	// Hostname: if it resolves to both v4 and v6, listen on both.
	addrs, rerr := net.DefaultResolver.LookupIPAddr(context.Background(), trimmed)
	if rerr != nil {
		return []listenTarget{{network: network, addr: addr}}, rerr
	}
	v4, v6 := splitFirstV4V6(addrs)
	if v4 == nil || v6 == nil {
		return []listenTarget{{network: network, addr: addr}}, nil
	}

	t4 := listenTarget{network: "tcp4", addr: net.JoinHostPort(v4.String(), strconv.Itoa(port))}
	t6 := listenTarget{network: "tcp6", addr: net.JoinHostPort(v6.String(), strconv.Itoa(port))}
	if preferIPv6 {
		return []listenTarget{t6, t4}, nil
	}
	return []listenTarget{t4, t6}, nil
}

func pickPreferredIPAddr(addrs []net.IPAddr, preferIPv6 bool) *net.IPAddr {
	for i := range addrs {
		ip := addrs[i].IP
		if ip == nil {
			continue
		}
		if preferIPv6 {
			if ip.To4() == nil {
				return &addrs[i]
			}
		} else {
			if ip.To4() != nil {
				return &addrs[i]
			}
		}
	}
	for i := range addrs {
		if addrs[i].IP != nil {
			return &addrs[i]
		}
	}
	return nil
}

func splitFirstV4V6(addrs []net.IPAddr) (v4, v6 *net.IPAddr) {
	for i := range addrs {
		ip := addrs[i].IP
		if ip == nil {
			continue
		}
		if ip.To4() != nil {
			if v4 == nil {
				v4 = &addrs[i]
			}
		} else {
			if v6 == nil {
				v6 = &addrs[i]
			}
		}
		if v4 != nil && v6 != nil {
			break
		}
	}
	return v4, v6
}

func selectListenTargetsForDevice(dev string, port int, preferIPv6 bool) ([]listenTarget, error) {
	if strings.TrimSpace(dev) == "" || port <= 0 {
		return nil, errors.New("invalid device/port")
	}
	ifi, err := net.InterfaceByName(dev)
	if err != nil {
		return nil, err
	}
	addrs, err := ifi.Addrs()
	if err != nil {
		return nil, err
	}
	return selectListenTargetsForDeviceAddrs(addrs, dev, port, preferIPv6)
}

func selectListenTargetsForDeviceAddrs(addrs []net.Addr, dev string, port int, preferIPv6 bool) ([]listenTarget, error) {
	var v4, v6, v6ll string
	for _, a := range addrs {
		ipNet, ok := a.(*net.IPNet)
		if !ok || ipNet.IP == nil {
			continue
		}
		ip := ipNet.IP
		if ip.To4() != nil {
			if v4 == "" {
				v4 = ip.String()
			}
			continue
		}
		ipStr := ip.String()
		if strings.HasPrefix(strings.ToLower(ipStr), "fe80:") {
			if v6ll == "" {
				v6ll = ipStr + "%" + dev
			}
		} else if v6 == "" {
			v6 = ipStr
		}
	}
	if v4 == "" && v6 == "" && v6ll == "" {
		return nil, fmt.Errorf("no suitable address on device %s", dev)
	}

	var targets []listenTarget
	addV4 := func() {
		if v4 != "" {
			targets = append(targets, listenTarget{network: "tcp4", addr: net.JoinHostPort(v4, strconv.Itoa(port))})
		}
	}
	addV6 := func() {
		if v6 != "" {
			targets = append(targets, listenTarget{network: "tcp6", addr: net.JoinHostPort(v6, strconv.Itoa(port))})
		}
		if v6ll != "" {
			targets = append(targets, listenTarget{network: "tcp6", addr: net.JoinHostPort(v6ll, strconv.Itoa(port))})
		}
	}

	if preferIPv6 {
		addV6()
		addV4()
	} else {
		addV4()
		addV6()
	}
	return targets, nil
}

func (d *BackboneClientDriver) Close() {
	select {
	case <-d.stopCh:
		return
	default:
		close(d.stopCh)
	}
	if d.iface != nil && d.iface.backbonePeer != nil {
		d.iface.backbonePeer.Close()
	}
}

var backboneDial = func(dialer *net.Dialer, network, address string) (net.Conn, error) {
	if dialer == nil {
		dialer = &net.Dialer{}
	}
	return dialer.Dial(network, address)
}

// tuneTCPBackbone matches Python BackboneInterface/BackboneClientInterface TCP settings:
// TCP_USER_TIMEOUT=24s, keepalive idle=5s, interval=2s, count=12 (Linux), and TCP_NODELAY.
func tuneTCPBackbone(conn net.Conn) {
	if tc, ok := conn.(*net.TCPConn); ok {
		_ = tc.SetKeepAlive(true)
		_ = tc.SetNoDelay(true)
		_ = tc.SetKeepAlivePeriod(5 * time.Second)
	}
	_ = setTCPTimeoutsBestEffort(conn, false)
}
