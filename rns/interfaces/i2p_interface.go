package interfaces

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	i2plib "github.com/svanichkin/i2plib"
)

const (
	i2pInterfaceHWMTU      = 1064
	i2pBitrateGuess        = 256_000
	i2pReconnectWait       = 15 * time.Second
	i2pTunnelWait          = 8 * time.Second
	i2pKeepaliveAfter      = 10 * time.Second
	i2pProbeInterval       = 9 * time.Second
	i2pProbes              = 5
	i2pReadTimeout         = (i2pProbeInterval*i2pProbes + i2pKeepaliveAfter) * 2
	i2pServerRetryInterval = 15 * time.Second
	i2pTunnelSetupTimeout  = 90 * time.Second
)

const (
	i2pTunnelStateInit  = 0x00
	i2pTunnelStateAlive = 0x01
	i2pTunnelStateStale = 0x02
)

const (
	i2pKISSFEND       = 0xC0
	i2pKISSFESC       = 0xDB
	i2pKISSTFEND      = 0xDC
	i2pKISSTFESC      = 0xDD
	i2pKISSCmdData    = 0x00
	i2pKISSCmdUnknown = 0xFE
)

// TransportIdentityHashProvider is set by the rns package to avoid import cycles.
// It should return the current transport identity hash bytes (may be truncated).
var TransportIdentityHashProvider func() []byte

// TunnelSynthesizer is set by the rns package to avoid import cycles.
// For Python parity, non-KISS tunnelled interfaces should request tunnel synthesis
// once a connection is established (Transport.synthesize_tunnel()).
var TunnelSynthesizer func(ifc *Interface)

type i2pInterfaceConfig struct {
	Name        string
	StoragePath string
	Peers       []string
	Connectable bool

	SAMHost string
	SAMPort int

	KISSFraming       bool
	MaxReconnectTries int // <0 means unlimited (Python: None)
}

// I2PClientDriver corresponds to Python I2PInterface (the parent interface).
// It accepts incoming I2P connections via a local TCP server behind a ServerTunnel,
// and spawns I2PInterfacePeer instances for each incoming connection.
type I2PClientDriver struct {
	iface *Interface
	cfg   i2pInterfaceConfig

	samAddr i2plib.Address
	sam     *i2plib.DefaultSAMClient

	localLn net.Listener

	// connectable endpoint
	connectable atomic.Bool
	b32         atomic.Value // string
	serverMu    sync.Mutex
	serverTun   *i2plib.ServerTunnel

	stopCh chan struct{}

	mu      sync.Mutex
	spawned []*Interface // incoming peers (clients property)
}

// I2PPeer corresponds to Python I2PInterfacePeer (an actual data-carrying interface).
type I2PPeer struct {
	parent *I2PClientDriver
	iface  *Interface

	initiator   bool
	parentCount bool
	remoteDest  string
	kissFraming bool
	maxRetries  int

	samAddr i2plib.Address
	sam     *i2plib.DefaultSAMClient

	// client tunnel state (initiator only)
	tunMu sync.Mutex
	tun   *i2plib.ClientTunnel

	connMu sync.Mutex
	conn   net.Conn

	sendMu sync.Mutex

	connGen atomic.Uint64

	closed atomic.Bool

	lastRead  atomic.Int64 // unixnano
	lastWrite atomic.Int64 // unixnano

	tunnelState atomic.Int32
}

func i2pLogSAMError(prefix string, err error) {
	if DiagLogf == nil || err == nil {
		return
	}
	var se *i2plib.SAMError
	if errors.As(err, &se) {
		switch se.Code {
		case i2plib.SAMErrCantReachPeer:
			DiagLogf(LogError, "%s The I2P daemon can't reach peer", prefix)
		case i2plib.SAMErrDuplicatedDest:
			DiagLogf(LogError, "%s The I2P daemon reported that the destination is already in use", prefix)
		case i2plib.SAMErrDuplicatedID:
			DiagLogf(LogError, "%s The I2P daemon reported that the ID is already in use", prefix)
		case i2plib.SAMErrInvalidID:
			DiagLogf(LogError, "%s The I2P daemon reported that the stream session ID doesn't exist", prefix)
		case i2plib.SAMErrInvalidKey:
			DiagLogf(LogError, "%s The I2P daemon reported that the key is invalid", prefix)
		case i2plib.SAMErrKeyNotFound:
			DiagLogf(LogError, "%s The I2P daemon could not find the key", prefix)
		case i2plib.SAMErrPeerNotFound:
			DiagLogf(LogError, "%s The I2P daemon could not find the peer", prefix)
		case i2plib.SAMErrI2PError:
			DiagLogf(LogError, "%s The I2P daemon experienced an unspecified error", prefix)
		case i2plib.SAMErrTimeout:
			DiagLogf(LogError, "%s I2P daemon timed out", prefix)
		default:
			DiagLogf(LogError, "%s I2P SAM error: %v", prefix, err)
		}
	}
}

func contextWithStop(parent context.Context, stopCh <-chan struct{}, timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	go func() {
		select {
		case <-stopCh:
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, cancel
}

func waitTunnelStatus(desc string, setupRan func() bool, setupFailed func() bool, setupErr func() error) error {
	// Python waits a short time for control status to appear.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if setupRan() {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !setupRan() {
		return fmt.Errorf("%s I2P tunnel setup did not complete", desc)
	}
	if setupFailed() || setupErr() != nil {
		if setupErr() != nil {
			return setupErr()
		}
		return fmt.Errorf("%s unspecified I2P tunnel setup error", desc)
	}
	return nil
}

func NewI2PInterface(name string, kv map[string]string) (*Interface, error) {
	cfg, err := parseI2PInterfaceConfig(name, kv)
	if err != nil {
		return nil, err
	}

	samAddr := i2plib.Address{Host: cfg.SAMHost, Port: cfg.SAMPort}
	sam := i2plib.NewDefaultSAMClient(samAddr)

	iface := &Interface{
		Name:              name,
		Type:              "I2PInterface",
		IN:                true,
		OUT:               false, // Python: I2PInterface.OUT = False
		DriverImplemented: true,
		Bitrate:           i2pBitrateGuess,
		HWMTU:             i2pInterfaceHWMTU,
		IFACSize:          16, // Python: DEFAULT_IFAC_SIZE = 16
	}

	driver := &I2PClientDriver{
		iface:   iface,
		cfg:     cfg,
		samAddr: samAddr,
		sam:     sam,
		stopCh:  make(chan struct{}),
	}
	driver.b32.Store("")
	iface.i2pClient = driver
	iface.clientCount = driver.ClientCount
	iface.clientCount = driver.ClientCount

	if err := driver.start(); err != nil {
		return iface, err
	}

	// Spawn configured peers (initiators).
	for _, remote := range cfg.Peers {
		peerIface, err := driver.NewInitiatorPeer(normaliseI2PDest(remote))
		if err != nil {
			if DiagLogf != nil {
				DiagLogf(LogError, "I2PInterface %s could not create peer %s: %v", name, remote, err)
			}
			continue
		}
		peerIface.OUT = true
		peerIface.IN = true
		peerIface.Parent = iface
		peerIface.Online = true
		if SpawnHandler != nil {
			SpawnHandler(peerIface)
		}
	}

	return iface, nil
}

func parseI2PInterfaceConfig(name string, kv map[string]string) (i2pInterfaceConfig, error) {
	cfg := i2pInterfaceConfig{
		Name:        name,
		StoragePath: first(kv, "storagepath"),
		Peers:       list(kv, "peers"),
		Connectable: parseBoolOr(first(kv, "connectable"), false),
		SAMHost:     first(kv, "sam_host"),
		SAMPort:     parseIntOr(first(kv, "sam_port"), i2plib.DefaultSAMPort),
		KISSFraming: parseBoolOr(first(kv, "kiss_framing"), false),
		MaxReconnectTries: parseIntOr(
			first(kv, "max_reconnect_tries"),
			-1,
		),
	}
	if strings.TrimSpace(cfg.SAMHost) == "" {
		cfg.SAMHost = i2plib.DefaultSAMHost
	}
	return cfg, nil
}

func (d *I2PClientDriver) samIsReachable(timeout time.Duration) bool {
	addr := net.JoinHostPort(d.samAddr.Host, fmt.Sprintf("%d", d.samAddr.Port))
	c, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}

func (d *I2PClientDriver) waitForSAMReady() bool {
	// Python waits for the controller event loop to become ready before continuing.
	time.Sleep(250 * time.Millisecond)
	if d.samIsReachable(750 * time.Millisecond) {
		return true
	}
	if DiagLogf != nil {
		DiagLogf(LogVerbose, "I2P controller did not become available in time, waiting for controller")
	}
	for !d.samIsReachable(750 * time.Millisecond) {
		select {
		case <-d.stopCh:
			return false
		default:
		}
		time.Sleep(250 * time.Millisecond)
	}
	if DiagLogf != nil {
		DiagLogf(LogVerbose, "I2P controller ready, continuing setup")
	}
	return true
}

func (d *I2PClientDriver) start() error {
	// Local TCP server behind the server tunnel.
	port, err := i2plib.GetFreePort()
	if err != nil {
		return err
	}
	addr := net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	d.localLn = ln

	// Only do the Python-style "controller wait" if we'll actually use SAM.
	if d.cfg.Connectable || len(d.cfg.Peers) > 0 {
		_ = d.waitForSAMReady()
	}

	go d.acceptLoop()

	if d.cfg.Connectable {
		go d.connectableLoop(port)
	}

	// For the parent interface, Online only signals connectable readiness in Python.
	if !d.cfg.Connectable {
		d.iface.Online = true
	}

	return nil
}

func (d *I2PClientDriver) ClientCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.spawned)
}

// Send mirrors the call site in Interface.ProcessOutgoing(). In Python,
// I2PInterface.process_outgoing() is a no-op; traffic is carried by peer interfaces.
func (d *I2PClientDriver) Send(_ []byte) {}

func (d *I2PClientDriver) Close() {
	select {
	case <-d.stopCh:
		return
	default:
		close(d.stopCh)
	}
	if d.localLn != nil {
		_ = d.localLn.Close()
	}
	d.mu.Lock()
	spawned := append([]*Interface(nil), d.spawned...)
	d.mu.Unlock()
	for _, ifc := range spawned {
		if ifc != nil && ifc.i2pPeer != nil {
			ifc.i2pPeer.Close()
		}
	}
	d.serverMu.Lock()
	if d.serverTun != nil {
		d.serverTun.Stop()
		d.serverTun = nil
	}
	d.serverMu.Unlock()
	d.iface.Online = false
}

func (d *I2PClientDriver) acceptLoop() {
	for {
		conn, err := d.localLn.Accept()
		if err != nil {
			select {
			case <-d.stopCh:
				return
			default:
				time.Sleep(100 * time.Millisecond)
				continue
			}
		}
		if DiagLogf != nil {
			DiagLogf(LogVerbose, "Accepting incoming I2P connection")
		}
		d.spawnIncoming(conn)
	}
}

func (d *I2PClientDriver) spawnIncoming(conn net.Conn) {
	parent := d.iface
	peerIface := &Interface{
		Name:              "Connected peer on " + parent.Name,
		Type:              "I2PInterfacePeer",
		Parent:            parent,
		IN:                true,
		OUT:               true,
		DriverImplemented: true,
		Bitrate:           parent.Bitrate,
		HWMTU:             parent.HWMTU,
		Mode:              parent.Mode,

		IngressControl:        parent.IngressControl,
		ICMaxHeldAnnounces:    parent.ICMaxHeldAnnounces,
		ICBurstHold:           parent.ICBurstHold,
		ICBurstFreqNew:        parent.ICBurstFreqNew,
		ICBurstFreq:           parent.ICBurstFreq,
		ICNewTime:             parent.ICNewTime,
		ICBurstPenalty:        parent.ICBurstPenalty,
		ICHeldReleaseInterval: parent.ICHeldReleaseInterval,
		AnnounceCap:           parent.AnnounceCap,
		AnnounceRateTarget:    parent.AnnounceRateTarget,
		AnnounceRateGrace:     parent.AnnounceRateGrace,
		AnnounceRatePenalty:   parent.AnnounceRatePenalty,

		IFACSize:       parent.IFACSize,
		IFACNetnameVal: parent.IFACNetnameVal,
		IFACNetkeyVal:  parent.IFACNetkeyVal,
		IFACKey:        parent.IFACKey,
		IFACIdentity:   parent.IFACIdentity,
		IFACSignature:  parent.IFACSignature,
		Online:         true,
		Created:        time.Now(),
	}

	peer := &I2PPeer{
		parent:      d,
		iface:       peerIface,
		initiator:   false,
		parentCount: true,
		kissFraming: d.cfg.KISSFraming,
		maxRetries:  d.cfg.MaxReconnectTries,
		samAddr:     d.samAddr,
		sam:         d.sam,
	}
	peer.tunnelState.Store(i2pTunnelStateInit)
	peer.setConn(conn)
	peerIface.i2pPeer = peer

	d.mu.Lock()
	d.spawned = append(d.spawned, peerIface)
	d.mu.Unlock()

	if SpawnHandler != nil {
		SpawnHandler(peerIface)
	}
	if DiagLogf != nil {
		DiagLogf(LogVerbose, "Spawned new I2PInterface Peer: %s", peerIface.Name)
	}

	go peer.readLoop()
	go peer.watchdogLoop()
}

func (d *I2PClientDriver) NewInitiatorPeer(remoteDest string) (*Interface, error) {
	parent := d.iface
	if strings.TrimSpace(remoteDest) == "" {
		return nil, errors.New("empty peer destination")
	}

	peerIface := &Interface{
		Name:              parent.Name + " to " + remoteDest,
		Type:              "I2PInterfacePeer",
		Parent:            parent,
		IN:                true,
		OUT:               true,
		DriverImplemented: true,
		Bitrate:           parent.Bitrate,
		HWMTU:             parent.HWMTU,
		Mode:              parent.Mode,

		IngressControl:        parent.IngressControl,
		ICMaxHeldAnnounces:    parent.ICMaxHeldAnnounces,
		ICBurstHold:           parent.ICBurstHold,
		ICBurstFreqNew:        parent.ICBurstFreqNew,
		ICBurstFreq:           parent.ICBurstFreq,
		ICNewTime:             parent.ICNewTime,
		ICBurstPenalty:        parent.ICBurstPenalty,
		ICHeldReleaseInterval: parent.ICHeldReleaseInterval,
		AnnounceCap:           parent.AnnounceCap,
		AnnounceRateTarget:    parent.AnnounceRateTarget,
		AnnounceRateGrace:     parent.AnnounceRateGrace,
		AnnounceRatePenalty:   parent.AnnounceRatePenalty,

		IFACSize:       parent.IFACSize,
		IFACNetnameVal: parent.IFACNetnameVal,
		IFACNetkeyVal:  parent.IFACNetkeyVal,
		IFACKey:        parent.IFACKey,
		IFACIdentity:   parent.IFACIdentity,
		IFACSignature:  parent.IFACSignature,
		Online:         true,
		Created:        time.Now(),
	}

	peer := &I2PPeer{
		parent:      d,
		iface:       peerIface,
		initiator:   true,
		parentCount: false, // Python sets parent_count=False for configured peers
		remoteDest:  remoteDest,
		kissFraming: d.cfg.KISSFraming,
		maxRetries:  d.cfg.MaxReconnectTries,
		samAddr:     d.samAddr,
		sam:         d.sam,
	}
	peer.tunnelState.Store(i2pTunnelStateInit)
	peerIface.i2pPeer = peer
	go peer.initiatorLoop()
	return peerIface, nil
}

func (d *I2PClientDriver) connectableLoop(localPort int) {
	for {
		select {
		case <-d.stopCh:
			return
		default:
		}
		if err := d.ensureServerTunnel(localPort); err != nil {
			if DiagLogf != nil {
				DiagLogf(LogError, "Error while configuring %s: %v", d.iface, err)
				DiagLogf(LogError, "Check that I2P is installed and running, and that SAM is enabled. Retrying tunnel setup later.")
			}
			d.connectable.Store(false)
			time.Sleep(i2pServerRetryInterval)
			continue
		}
		time.Sleep(i2pServerRetryInterval)
	}
}

func hexNoDelim(b []byte) string {
	return hex.EncodeToString(b)
}

func fullHash(b []byte) []byte {
	sum := sha256.Sum256(b)
	return sum[:]
}

func (d *I2PClientDriver) ensureServerTunnel(localPort int) error {
	if strings.TrimSpace(d.cfg.StoragePath) == "" {
		return errors.New("missing storagepath for connectable I2PInterface")
	}

	// Python waits until Transport.identity exists before selecting new-format key.
	if TransportIdentityHashProvider != nil {
		for {
			if tid := TransportIdentityHashProvider(); len(tid) > 0 {
				break
			}
			// We can still proceed with old-format if it already exists; otherwise wait.
			if _, err := os.Stat(filepath.Join(d.cfg.StoragePath, "i2p")); err == nil {
				break
			}
			time.Sleep(1 * time.Second)
			select {
			case <-d.stopCh:
				return errors.New("stopped")
			default:
			}
		}
	}

	// Mimic Python keyfile naming: old format and new format.
	base := fullHash(fullHash([]byte(d.iface.Name)))
	oldKey := filepath.Join(d.cfg.StoragePath, "i2p", hexNoDelim(base)+".i2p")

	newKey := ""
	if TransportIdentityHashProvider != nil {
		tid := TransportIdentityHashProvider()
		if len(tid) > 0 {
			nh := fullHash(append(append([]byte(nil), base...), fullHash(tid)...))
			newKey = filepath.Join(d.cfg.StoragePath, "i2p", hexNoDelim(nh)+".i2p")
		}
	}

	keyfile := oldKey
	if newKey != "" {
		// Use old format if a key is already present.
		if _, err := os.Stat(oldKey); err == nil {
			keyfile = oldKey
		} else {
			keyfile = newKey
		}
	}

	if err := os.MkdirAll(filepath.Dir(keyfile), 0o755); err != nil {
		return err
	}

	var dest *i2plib.Destination
	if _, err := os.Stat(keyfile); err == nil {
		b, err := os.ReadFile(keyfile)
		if err != nil {
			return err
		}
		privB64 := strings.TrimSpace(string(b))
		if privB64 == "" {
			return errors.New("empty i2p keyfile")
		}
		dest, err = i2plib.DestinationFromBase64(privB64, true)
		if err != nil {
			return err
		}
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		dest, err = d.sam.NewDestination(ctx, nil)
		if err != nil {
			return err
		}
		priv := ""
		if dest.PrivKey != nil {
			priv = dest.PrivKey.Base64
		}
		if strings.TrimSpace(priv) == "" {
			return errors.New("SAM did not return private key")
		}
		if err := os.WriteFile(keyfile, []byte(priv), 0o600); err != nil {
			return err
		}
	}

	b32 := dest.Base32()
	d.b32.Store(b32)

	local := i2plib.Address{Host: "127.0.0.1", Port: localPort}
	d.serverMu.Lock()
	existing := d.serverTun
	d.serverMu.Unlock()
	if existing != nil && existing.Status.SetupRan && !existing.Status.SetupFailed && existing.Status.Err == nil {
		d.connectable.Store(true)
		d.iface.Online = true
		return nil
	}
	if existing != nil {
		existing.Stop()
	}

	tun := i2plib.NewServerTunnel(local, d.samAddr, d.sam, dest, "", nil)

	if DiagLogf != nil {
		DiagLogf(LogInfo, "%s Bringing up I2P endpoint, this may take a while...", d.iface)
	}

	ctx, cancel := contextWithStop(context.Background(), d.stopCh, i2pTunnelSetupTimeout)
	defer cancel()
	if err := tun.Run(ctx); err != nil {
		i2pLogSAMError(fmt.Sprintf("%s:", d.iface), err)
		return err
	}
	if err := waitTunnelStatus(
		d.iface.String(),
		func() bool { return tun.Status.SetupRan },
		func() bool { return tun.Status.SetupFailed },
		func() error { return tun.Status.Err },
	); err != nil {
		if DiagLogf != nil {
			DiagLogf(LogError, "%v", err)
			i2pLogSAMError(fmt.Sprintf("%s:", d.iface), err)
			DiagLogf(LogError, "Resetting I2P tunnel and retrying later")
		}
		tun.Stop()
		return err
	}
	d.serverMu.Lock()
	d.serverTun = tun
	d.serverMu.Unlock()
	// Tunnel runs in background. Mark connectable and emit the same "reachable at" log.
	d.connectable.Store(true)
	d.iface.Online = true
	if DiagLogf != nil {
		DiagLogf(LogVerbose, "%s endpoint setup complete. Now reachable at: %s.b32.i2p", d.iface, b32)
	}
	return nil
}

func (p *I2PPeer) setConn(c net.Conn) {
	p.connMu.Lock()
	old := p.conn
	p.conn = c
	p.connMu.Unlock()
	if old != nil && old != c {
		_ = old.Close()
	}

	// Python sets TCP keepalive parameters; mirror best-effort where possible.
	if tc, ok := c.(*net.TCPConn); ok {
		_ = tc.SetNoDelay(true)
		_ = tc.SetKeepAlive(true)
		_ = tc.SetKeepAlivePeriod(i2pKeepaliveAfter)
	}
	_ = setTCPTimeoutsBestEffort(c, true)

	p.connGen.Add(1)
	now := time.Now().UnixNano()
	p.lastRead.Store(now)
	p.lastWrite.Store(now)
}

func (p *I2PPeer) getConn() net.Conn {
	p.connMu.Lock()
	defer p.connMu.Unlock()
	return p.conn
}

func (p *I2PPeer) dropConnection() {
	if c := p.getConn(); c != nil {
		_ = c.Close()
	}
	p.connMu.Lock()
	p.conn = nil
	p.connMu.Unlock()
	p.connGen.Add(1)

	p.tunMu.Lock()
	if p.tun != nil {
		p.tun.Stop()
		p.tun = nil
	}
	p.tunMu.Unlock()

	if p.iface != nil {
		p.iface.Online = false
	}
}

func (d *I2PClientDriver) removeSpawned(ifc *Interface) {
	if d == nil || ifc == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	for i := 0; i < len(d.spawned); i++ {
		if d.spawned[i] == ifc {
			copy(d.spawned[i:], d.spawned[i+1:])
			d.spawned = d.spawned[:len(d.spawned)-1]
			return
		}
	}
}

func (p *I2PPeer) Close() {
	if !p.closed.CompareAndSwap(false, true) {
		return
	}
	p.dropConnection()
	if p.iface != nil {
		p.iface.Online = false
		p.iface.i2pPeer = nil
	}

	// Python tears down spawned (non-initiator) peers and removes them from transport.
	if !p.initiator && p.parent != nil && p.iface != nil {
		p.parent.removeSpawned(p.iface)
		removeInterface(p.iface)
	}
}

var errAwaitingI2PTunnel = errors.New("still waiting for I2P tunnel to appear")

func (p *I2PPeer) initiatorLoop() {
	// Wait for I2P tunnel to appear, then connect + read loop; reconnect on failure.
	if p == nil || p.iface == nil {
		return
	}

	neverConnected := true
	attempts := 0

	for !p.closed.Load() {
		time.Sleep(i2pTunnelWait)
		attempts++

		if DiagLogf != nil {
			DiagLogf(LogDebug, "Establishing I2P connection for %s...", p.iface.Name)
		}

		if err := p.connectViaClientTunnel(); err != nil {
			if DiagLogf != nil {
				if errors.Is(err, errAwaitingI2PTunnel) {
					DiagLogf(LogVerbose, "%s still waiting for I2P tunnel to appear", p.iface.Name)
				} else {
					i2pLogSAMError(fmt.Sprintf("%s:", p.iface.Name), err)
					DiagLogf(LogDebug, "Connection attempt for %s failed: %v", p.iface.Name, err)
					if neverConnected {
						DiagLogf(LogError, "Initial connection for %s could not be established: %v", p.iface.Name, err)
						DiagLogf(LogError, "Leaving unconnected and retrying connection in %s.", i2pReconnectWait)
					}
				}
			}
			if p.maxRetries >= 0 && attempts > p.maxRetries {
				if DiagLogf != nil {
					DiagLogf(LogError, "Max reconnection attempts reached for %s", p.iface.Name)
				}
				p.Close()
				return
			}
			time.Sleep(i2pReconnectWait)
			continue
		}

		if !neverConnected && DiagLogf != nil {
			DiagLogf(LogInfo, "%s Re-established connection via I2P tunnel", p.iface.Name)
		}
		neverConnected = false
		attempts = 0

		connGen := p.connGen.Load()
		go p.watchdogLoopGen(connGen)
		p.readLoopGen(connGen)

		// Connection dropped.
		p.iface.Online = false
		if DiagLogf != nil {
			DiagLogf(LogWarning, "Socket for %s was closed, attempting to reconnect...", p.iface.Name)
		}
		p.dropConnection()
		time.Sleep(i2pReconnectWait)
	}
}

func (p *I2PPeer) connectViaClientTunnel() error {
	if strings.TrimSpace(p.remoteDest) == "" {
		return errors.New("missing remote destination")
	}
	if p.parent != nil && !p.parent.samIsReachable(750*time.Millisecond) {
		return errAwaitingI2PTunnel
	}

	port, err := i2plib.GetFreePort()
	if err != nil {
		return err
	}
	local := i2plib.Address{Host: "127.0.0.1", Port: port}
	tun := i2plib.NewClientTunnel(local, p.remoteDest, p.samAddr, p.sam, nil, "", nil)

	stopCh := (<-chan struct{})(nil)
	if p.parent != nil {
		stopCh = p.parent.stopCh
	}
	if stopCh == nil {
		stopCh = make(chan struct{})
	}
	ctx, cancel := contextWithStop(context.Background(), stopCh, i2pTunnelSetupTimeout)
	defer cancel()
	if err := tun.Run(ctx); err != nil {
		i2pLogSAMError(fmt.Sprintf("%s:", p.iface.Name), err)
		return err
	}
	if err := waitTunnelStatus(
		p.iface.Name,
		func() bool { return tun.Status.SetupRan },
		func() bool { return tun.Status.SetupFailed },
		func() error { return tun.Status.Err },
	); err != nil {
		if DiagLogf != nil {
			DiagLogf(LogError, "%v", err)
			i2pLogSAMError(fmt.Sprintf("%s:", p.iface.Name), err)
			DiagLogf(LogError, "Resetting I2P tunnel and retrying later")
		}
		tun.Stop()
		return err
	}
	// Give listener a moment to bind.
	time.Sleep(200 * time.Millisecond)

	c, err := (&net.Dialer{Timeout: 5 * time.Second}).Dial("tcp", net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", port)))
	if err != nil {
		tun.Stop()
		return err
	}

	p.tunMu.Lock()
	if p.tun != nil {
		p.tun.Stop()
	}
	p.tun = tun
	p.tunMu.Unlock()

	p.setConn(c)
	p.iface.Online = true
	if !p.kissFraming && TunnelSynthesizer != nil {
		TunnelSynthesizer(p.iface)
	}
	return nil
}

func (p *I2PPeer) ProcessOutgoing(data []byte) {
	if p == nil || p.closed.Load() || len(data) == 0 || p.iface == nil || !p.iface.Online {
		return
	}
	var frame []byte
	if p.kissFraming {
		frame = append([]byte{i2pKISSFEND, i2pKISSCmdData}, i2pKISSEscape(data)...)
		frame = append(frame, i2pKISSFEND)
	} else {
		frame = append([]byte{hdlcFlag}, hdlcEscape(data)...)
		frame = append(frame, hdlcFlag)
	}
	framedLen := len(frame)
	p.sendMu.Lock()
	defer p.sendMu.Unlock()
	c := p.getConn()
	if c == nil {
		return
	}
	for len(frame) > 0 {
		n, err := c.Write(frame)
		if err != nil || n <= 0 {
			return
		}
		frame = frame[n:]
	}
	p.iface.TXB += uint64(framedLen)
	if p.parentCount && p.iface.Parent != nil {
		p.iface.Parent.TXB += uint64(framedLen)
	}
	p.lastWrite.Store(time.Now().UnixNano())
}

func (p *I2PPeer) watchdogLoop() {
	// Incoming peers: one watchdog for lifetime.
	p.watchdogLoopGen(p.connGen.Load())
}

func (p *I2PPeer) watchdogLoopGen(gen uint64) {
	t := time.NewTicker(1 * time.Second)
	defer t.Stop()

	for {
		if p.closed.Load() {
			return
		}
		if p.connGen.Load() != gen {
			return
		}
		select {
		case <-t.C:
		default:
			time.Sleep(50 * time.Millisecond)
			continue
		}

		lastRead := time.Unix(0, p.lastRead.Load())
		lastWrite := time.Unix(0, p.lastWrite.Load())

		// Python marks tunnel stale when reads stop.
		if time.Since(lastRead) > i2pKeepaliveAfter*2 {
			if p.tunnelState.Load() != i2pTunnelStateStale {
				if DiagLogf != nil {
					DiagLogf(LogDebug, "I2P tunnel became unresponsive")
				}
			}
			p.tunnelState.Store(i2pTunnelStateStale)
		} else {
			p.tunnelState.Store(i2pTunnelStateAlive)
		}

		if time.Since(lastWrite) > i2pKeepaliveAfter {
			c := p.getConn()
			if c != nil {
				_, err := c.Write([]byte{hdlcFlag, hdlcFlag})
				if err != nil {
					if DiagLogf != nil {
						DiagLogf(LogError, "An error ocurred while sending I2P keepalive. The contained exception was: %v", err)
					}
					_ = c.Close()
					return
				}
			}
		}

		if time.Since(lastRead) > i2pReadTimeout {
			if DiagLogf != nil {
				DiagLogf(LogWarning, "I2P socket is unresponsive, restarting...")
			}
			c := p.getConn()
			if c != nil {
				_ = c.Close()
			}
			return
		}
	}
}

func (p *I2PPeer) readLoop() {
	p.readLoopGen(p.connGen.Load())
}

func (p *I2PPeer) readLoopGen(gen uint64) {
	defer func() {
		// For incoming peers, teardown is permanent.
		if !p.initiator {
			p.Close()
		}
	}()

	buf := make([]byte, 4096)

	// --- KISS state (Python bytewise parser) ---
	inFrame := false
	escape := false
	command := byte(i2pKISSCmdUnknown)
	var kissBuf bytes.Buffer

	// --- HDLC buffer ---
	var frameBuf []byte

	for {
		c := p.getConn()
		if c == nil {
			return
		}
		n, err := c.Read(buf)
		if err != nil || n <= 0 {
			if err != nil && !errors.Is(err, io.EOF) && DiagLogf != nil && p.initiator {
				DiagLogf(LogWarning, "An interface error occurred for %s, the contained exception was: %v", p.iface.Name, err)
			}
			return
		}
		if p.connGen.Load() != gen {
			return
		}
		p.lastRead.Store(time.Now().UnixNano())

		chunk := buf[:n]
		if p.kissFraming {
			for i := 0; i < len(chunk); i++ {
				b := chunk[i]
				if inFrame && b == i2pKISSFEND && command == i2pKISSCmdData {
					inFrame = false
					payload := append([]byte(nil), kissBuf.Bytes()...)
					kissBuf.Reset()
					if len(payload) > 0 && len(payload) <= MaxFrameLength {
						p.iface.RXB += uint64(len(payload))
						if p.parentCount && p.iface.Parent != nil {
							p.iface.Parent.RXB += uint64(len(payload))
						}
						if InboundHandler != nil {
							InboundHandler(payload, p.iface)
						}
					}
					continue
				} else if b == i2pKISSFEND {
					inFrame = true
					command = i2pKISSCmdUnknown
					kissBuf.Reset()
					escape = false
					continue
				} else if inFrame && kissBuf.Len() < p.iface.HWMTU {
					if kissBuf.Len() == 0 && command == i2pKISSCmdUnknown {
						command = b & 0x0F // strip port nibble
						continue
					}
					if command == i2pKISSCmdData {
						if b == i2pKISSFESC {
							escape = true
							continue
						}
						if escape {
							if b == i2pKISSTFEND {
								b = i2pKISSFEND
							} else if b == i2pKISSTFESC {
								b = i2pKISSFESC
							}
							escape = false
						}
						_ = kissBuf.WriteByte(b)
					}
				}
			}
		} else {
			frameBuf = append(frameBuf, chunk...)
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
				p.iface.RXB += uint64(len(payload))
				if p.parentCount && p.iface.Parent != nil {
					p.iface.Parent.RXB += uint64(len(payload))
				}
				if InboundHandler != nil {
					InboundHandler(append([]byte(nil), payload...), p.iface)
				}
			}
		}
	}
}

func i2pKISSEscape(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	out := make([]byte, 0, len(data)+8)
	for _, b := range data {
		switch b {
		case i2pKISSFESC:
			out = append(out, i2pKISSFESC, i2pKISSTFESC)
		case i2pKISSFEND:
			out = append(out, i2pKISSFESC, i2pKISSTFEND)
		default:
			out = append(out, b)
		}
	}
	return out
}

// Helper: normalise peer addresses in config; allow ".b32.i2p" suffix without changing bytes.
func normaliseI2PDest(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, ".b32.i2p") {
		return s
	}
	if strings.HasSuffix(s, ".i2p") {
		return s
	}
	return s
}
