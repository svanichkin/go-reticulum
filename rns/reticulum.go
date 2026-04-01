package rns

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/svanichkin/go-reticulum/rns/cryptography"
	ifaces "github.com/svanichkin/go-reticulum/rns/interfaces"
	vendor "github.com/svanichkin/go-reticulum/rns/vendor"

	configobj "github.com/svanichkin/configobj"
)

// ConfigObj equivalent.

const (
	DefaultMTU              = 500
	LINK_MTU_DISCOVERY      = true
	MAX_QUEUED_ANNOUNCES    = 16384
	QUEUED_ANNOUNCE_LIFE    = 60 * 60 * 24
	ANNOUNCE_CAP            = 2 // percent
	MINIMUM_BITRATE         = 5
	DEFAULT_PER_HOP_TIMEOUT = 6
	TRUNCATED_HASHLENGTH    = 128

	HEADER_MINSIZE = 2 + 1 + (TRUNCATED_HASHLENGTH/8)*1
	HEADER_MAXSIZE = 2 + 1 + (TRUNCATED_HASHLENGTH/8)*2
	IFAC_MIN_SIZE  = 1

	RESOURCE_CACHE            = 24 * 60 * 60
	JOB_INTERVAL              = 5 * 60
	CLEAN_INTERVAL            = 15 * 60
	PERSIST_INTERVAL          = 60 * 60 * 12
	GRACIOUS_PERSIST_INTERVAL = 60 * 5
	sha256Bits                = 256
)

// Interface modes (match RNS.Interfaces.Interface.MODE_*).
const (
	InterfaceModeFull         = 0x01
	InterfaceModePointToPoint = 0x02
	InterfaceModeAccessPoint  = 0x03
	InterfaceModeRoaming      = 0x04
	InterfaceModeBoundary     = 0x05
	InterfaceModeGateway      = 0x06
)

var (
	IFAC_SALT = mustHex("adf54d882c9a9b80771eb4995d702d4a3e733391b2a0f53f416d9f907e55cff8")

	MTU = DefaultMTU
	MDU = MTU - HEADER_MAXSIZE - IFAC_MIN_SIZE

	instance     *Reticulum
	instanceOnce sync.Once

	// Global flags, like in the Python class.
	transportEnabled        = false
	linkMTUDiscovery        = LINK_MTU_DISCOVERY
	remoteManagementEnabled = false
	useImplicitProof        = true
	allowProbes             = false
)

func init() {
	gob.Register(map[string]any{})
	gob.Register([]map[string]any{})
	gob.Register([]any{})
	gob.Register([]byte{})

	// Mirror Python's defaulting of Interface.announce_cap to Reticulum.ANNOUNCE_CAP.
	ifaces.DefaultAnnounceCapProvider = func() float64 { return float64(ANNOUNCE_CAP) / 100.0 }

	// Mirror Python's IFAC derivation for TCPServerInterface spawning, without import cycles.
	ifaces.TCPIFACDeriver = func(ifacNetname, ifacNetkey string) ([]byte, interface{ Sign([]byte) ([]byte, error) }, []byte, error) {
		origin := make([]byte, 0)
		if strings.TrimSpace(ifacNetname) != "" {
			origin = append(origin, FullHash([]byte(ifacNetname))...)
		}
		if strings.TrimSpace(ifacNetkey) != "" {
			origin = append(origin, FullHash([]byte(ifacNetkey))...)
		}
		if len(origin) == 0 {
			return nil, nil, nil, nil
		}
		originHash := FullHash(origin)
		ifacKey, err := cryptography.HKDF(64, originHash, IFAC_SALT, nil)
		if err != nil {
			return nil, nil, nil, err
		}
		ifacIdentity, _ := IdentityFromBytes(ifacKey)
		ifacSignature, _ := ifacIdentity.Sign(FullHash(ifacKey))
		return ifacKey, ifacIdentity, ifacSignature, nil
	}
}

type Reticulum struct {
	// Global paths.
	UserDir       string
	ConfigDir     string
	ConfigPath    string
	StoragePath   string
	CachePath     string
	ResourcePath  string
	IdentityPath  string
	InterfacePath string

	Config *configobj.Config

	// Settings/state.
	localInterfacePort int
	localControlPort   int
	LocalSocketPath    string
	ShareInstance      bool
	SharedInstanceType string // "tcp" / "unix"
	RPCKey             []byte
	UseAFUnix          bool

	RequestedLoglevel  *int
	RequestedVerbosity *int

	IsSharedInstance            bool
	SharedInstanceInterface     *Interface
	RequireShared               bool
	IsConnectedToSharedInstance bool
	IsStandaloneInstance        bool

	// LocalInterface packet IPC address (unix path or tcp addr).
	localPktAddr string

	PanicOnInterfaceError bool

	jobsStarted     bool
	lastDataPersist time.Time
	lastCacheClean  time.Time

	ifacSalt []byte

	// RPC listener + address (TCP/Unix).
	rpcAddr    string
	rpcNetwork string      // "tcp" / "unix"
	rpcLn      RPCListener // wrapper (see interface below)
}

// Optional API parity with Python Reticulum/RNS daemon management calls.
// These manage interfaces by config-section name (interface short name).
func (r *Reticulum) HaltInterface(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("missing interface name")
	}
	if r.IsConnectedToSharedInstance {
		return r.rpcCallInterfaceMgmt("halt_interface", name)
	}
	ifc := findInterfaceByName(name)
	if ifc == nil {
		return fmt.Errorf("interface %q not found", name)
	}
	removeInterface(ifc)
	return nil
}

func (r *Reticulum) ResumeInterface(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("missing interface name")
	}
	if r.IsConnectedToSharedInstance {
		return r.rpcCallInterfaceMgmt("resume_interface", name)
	}
	if findInterfaceByName(name) != nil {
		return fmt.Errorf("interface %q already running", name)
	}
	kv, ok := r.getInterfaceConfigByName(name)
	if !ok {
		return fmt.Errorf("interface %q not found in config", name)
	}
	started, err := r.startInterfaceFromConfig(name, kv)
	if err != nil {
		return err
	}
	if !started {
		return fmt.Errorf("interface %q is disabled", name)
	}
	return nil
}

func (r *Reticulum) ReloadInterface(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("missing interface name")
	}
	if r.IsConnectedToSharedInstance {
		return r.rpcCallInterfaceMgmt("reload_interface", name)
	}
	if ifc := findInterfaceByName(name); ifc != nil {
		removeInterface(ifc)
	}
	kv, ok := r.getInterfaceConfigByName(name)
	if !ok {
		return fmt.Errorf("interface %q not found in config", name)
	}
	started, err := r.startInterfaceFromConfig(name, kv)
	if err != nil {
		return err
	}
	if !started {
		return fmt.Errorf("interface %q is disabled", name)
	}
	return nil
}

type tcpLogAdapter struct{}

func (tcpLogAdapter) Debugf(format string, args ...any) { Logf(LogDebug, format, args...) }
func (tcpLogAdapter) Infof(format string, args ...any)  { Logf(LogInfo, format, args...) }
func (tcpLogAdapter) Warnf(format string, args ...any)  { Logf(LogWarning, format, args...) }
func (tcpLogAdapter) Errorf(format string, args ...any) { Logf(LogError, format, args...) }

type tcpOwnerAdapter struct{ ifc *Interface }

func (o tcpOwnerAdapter) Inbound(data []byte, _ *ifaces.TCPClientInterface) {
	if o.ifc == nil || len(data) == 0 {
		return
	}
	o.ifc.RXB += uint64(len(data))
	if ifaces.InboundHandler != nil {
		ifaces.InboundHandler(data, o.ifc)
	}
}

// RPCListener is an abstraction over TCP/Unix listeners.
// The implementation is intentionally flexible.
type RPCListener interface {
	Accept() (RPCConn, error)
	Close() error
	Addr() string
}

type RPCConn interface {
	Recv(v interface{}) error
	Send(v interface{}) error
	Close() error
}

type gobRPCListener struct {
	net.Listener
	authKey []byte
	cleanup func()
	addr    string
}

type gobRPCConn struct {
	conn net.Conn
	enc  *gob.Encoder
	dec  *gob.Decoder
}

func NewRPCListener(network, addr string, key []byte) (RPCListener, error) {
	switch network {
	case "unix":
		ln, resolved, cleanup, err := listenUnix(addr)
		if err != nil {
			return nil, err
		}
		return &gobRPCListener{
			Listener: ln,
			authKey:  append([]byte(nil), key...),
			cleanup:  cleanup,
			addr:     resolved,
		}, nil
	default:
		ln, err := net.Listen(network, addr)
		if err != nil {
			return nil, err
		}
		return &gobRPCListener{
			Listener: ln,
			authKey:  append([]byte(nil), key...),
			addr:     ln.Addr().String(),
		}, nil
	}
}

func dialRPC(network, addr string, key []byte) (RPCConn, error) {
	var (
		conn net.Conn
		err  error
	)
	switch network {
	case "unix":
		conn, err = dialUnix(addr)
	default:
		conn, err = net.Dial(network, addr)
	}
	if err != nil {
		return nil, err
	}
	if err := performRPCHandshake(conn, key, false); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return newGobRPCConn(conn), nil
}

func listenUnix(addr string) (net.Listener, string, func(), error) {
	if isAbstractUnix(addr) && supportsAbstractUnixSockets() {
		ln, err := net.Listen("unix", addr)
		if err == nil {
			return ln, addr, func() {}, nil
		}
		// Abstract UNIX socket bind failed. On Linux, abstract sockets have no
		// filesystem presence and cannot go stale, so a bind failure means
		// another process definitively holds the socket. Return the error
		// immediately so the caller connects as a client; no dial-probe needed.
		return nil, "", nil, err
	}
	path := fallbackUnixSocketPath(addr)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		Log("Could not remove stale RPC socket: "+err.Error(), LogDebug)
	}
	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, "", nil, err
	}
	cleanup := func() {
		_ = os.Remove(path)
	}
	return ln, path, cleanup, nil
}

func dialUnix(addr string) (net.Conn, error) {
	if isAbstractUnix(addr) {
		if supportsAbstractUnixSockets() {
			return net.Dial("unix", addr)
		}
		return net.Dial("unix", fallbackUnixSocketPath(addr))
	}
	return net.Dial("unix", addr)
}

func fallbackUnixSocketPath(addr string) string {
	name := strings.Trim(addr, "\x00")
	if name == "" {
		name = "default"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_", ":", "_")
	name = replacer.Replace(name)
	// Keep paths short to avoid UNIX socket path length limits (notably on macOS).
	if len(name) > 48 {
		sum := sha256.Sum256([]byte(name))
		name = hex.EncodeToString(sum[:16])
	}
	return filepath.Join(os.TempDir(), "rns-"+name+".sock")
}

func isAbstractUnix(addr string) bool {
	return len(addr) > 0 && addr[0] == 0
}

func supportsAbstractUnixSockets() bool {
	// Abstract UNIX sockets are Linux/Android-specific.
	return vendor.IsLinux() || vendor.IsAndroid()
}

func newGobRPCConn(conn net.Conn) *gobRPCConn {
	return &gobRPCConn{
		conn: conn,
		enc:  gob.NewEncoder(conn),
		dec:  gob.NewDecoder(conn),
	}
}

func (l *gobRPCListener) Accept() (RPCConn, error) {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}
		if err := performRPCHandshake(conn, l.authKey, true); err != nil {
			Log("Rejected RPC connection: "+err.Error(), LogWarning)
			_ = conn.Close()
			continue
		}
		return newGobRPCConn(conn), nil
	}
}

func (l *gobRPCListener) Close() error {
	if l.cleanup != nil {
		defer l.cleanup()
	}
	return l.Listener.Close()
}

func (l *gobRPCListener) Addr() string {
	return l.addr
}

func (c *gobRPCConn) Recv(v interface{}) error {
	return c.dec.Decode(v)
}

func (c *gobRPCConn) Send(v interface{}) error {
	return c.enc.Encode(v)
}

func (c *gobRPCConn) Close() error {
	return c.conn.Close()
}

// rpcSendBytes sends data using Python multiprocessing.connection wire format:
// [4-byte signed big-endian length][data]
func rpcSendBytes(conn net.Conn, data []byte) error {
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
	if _, err := conn.Write(hdr[:]); err != nil {
		return err
	}
	if len(data) > 0 {
		_, err := conn.Write(data)
		return err
	}
	return nil
}

// rpcRecvBytes reads data using Python multiprocessing.connection wire format.
func rpcRecvBytes(conn net.Conn, maxLen int) ([]byte, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(conn, hdr[:]); err != nil {
		return nil, err
	}
	n := int(binary.BigEndian.Uint32(hdr[:]))
	if maxLen > 0 && n > maxLen {
		return nil, fmt.Errorf("rpc message too large: %d bytes", n)
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// performRPCHandshake implements the Python multiprocessing.connection
// HMAC challenge-response authentication protocol.
//
// Server sequence: deliver_challenge → answer_challenge
// Client sequence: answer_challenge → deliver_challenge
func performRPCHandshake(conn net.Conn, key []byte, server bool) error {
	if server {
		if err := rpcDeliverChallenge(conn, key); err != nil {
			return fmt.Errorf("deliver challenge: %w", err)
		}
		return rpcAnswerChallenge(conn, key)
	}
	if err := rpcAnswerChallenge(conn, key); err != nil {
		return fmt.Errorf("answer challenge: %w", err)
	}
	return rpcDeliverChallenge(conn, key)
}

var (
	rpcChallenge = []byte("#CHALLENGE#")
	rpcWelcome   = []byte("#WELCOME#")
	rpcFailure   = []byte("#FAILURE#")
)

// rpcDeliverChallenge sends a challenge and verifies the peer's response.
func rpcDeliverChallenge(conn net.Conn, key []byte) error {
	randBytes := make([]byte, 40)
	if _, err := rand.Read(randBytes); err != nil {
		return err
	}
	digestPrefix := []byte("{sha256}")
	challengeMsg := append(digestPrefix, randBytes...)
	msg := append(rpcChallenge, challengeMsg...)
	if err := rpcSendBytes(conn, msg); err != nil {
		return err
	}
	resp, err := rpcRecvBytes(conn, 256)
	if err != nil {
		return err
	}
	expected, err := rpcCreateResponse(key, challengeMsg)
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare(resp, expected) != 1 {
		_ = rpcSendBytes(conn, rpcFailure)
		return errors.New("digest sent was rejected")
	}
	return rpcSendBytes(conn, rpcWelcome)
}

// rpcAnswerChallenge reads a server challenge and responds with HMAC.
func rpcAnswerChallenge(conn net.Conn, key []byte) error {
	msg, err := rpcRecvBytes(conn, 256)
	if err != nil {
		return err
	}
	if len(msg) < len(rpcChallenge) || !bytes.Equal(msg[:len(rpcChallenge)], rpcChallenge) {
		return fmt.Errorf("protocol error, expected challenge, got: %q", msg)
	}
	challengeMsg := msg[len(rpcChallenge):]
	digest, err := rpcCreateResponse(key, challengeMsg)
	if err != nil {
		return err
	}
	if err := rpcSendBytes(conn, digest); err != nil {
		return err
	}
	resp, err := rpcRecvBytes(conn, 256)
	if err != nil {
		return err
	}
	if !bytes.Equal(resp, rpcWelcome) {
		return errors.New("digest sent was rejected")
	}
	return nil
}

// rpcCreateResponse computes the HMAC response for a challenge message.
// It handles both legacy (HMAC-MD5) and modern (prefixed HMAC-SHA256) formats.
func rpcCreateResponse(key, message []byte) ([]byte, error) {
	digestPrefix := []byte("{sha256}")
	if len(message) >= len(digestPrefix) && bytes.Equal(message[:len(digestPrefix)], digestPrefix) {
		mac := hmac.New(sha256.New, key)
		mac.Write(message)
		sum := mac.Sum(nil)
		return append(digestPrefix, sum...), nil
	}
	mac := hmac.New(md5.New, key)
	mac.Write(message)
	return mac.Sum(nil), nil
}

// helper hex parser
func mustHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

// GetInstance mirrors get_instance().
func GetInstance() *Reticulum {
	return instance
}

var (
	errAlreadyRunning = errors.New("Reticulum is already running")
)

// NewReticulum mirrors __init__.
func NewReticulum(configDir *string, loglevel *int, logdest any, verbosity *int,
	requireSharedInstance bool, sharedInstanceType *string) (ret *Reticulum, retErr error) {

	if instance != nil {
		return nil, errAlreadyRunning
	}

	// Python parity: run platform checks early (mostly a no-op in Go).
	vendor.PlatformChecks()

	r := &Reticulum{
		UserDir:            osUserDir(),
		localInterfacePort: 37428,
		localControlPort:   37429,
		ShareInstance:      true,
		SharedInstanceType: "",
		RequireShared:      requireSharedInstance,
		ifacSalt:           IFAC_SALT,
		lastDataPersist:    time.Now(),
		lastCacheClean:     time.Unix(0, 0),
		// Python parity: runtime interface panic behaviour is disabled by default.
		PanicOnInterfaceError: false,
	}

	// Python parity: set the singleton instance early so identity/storage helpers
	// use the configured config dir during initialisation.
	instance = r
	defer func() {
		if retErr != nil && instance == r {
			instance = nil
		}
	}()

	// config dir
	if configDir != nil {
		r.ConfigDir = *configDir
	} else {
		if dirExists("/etc/reticulum") && fileExists("/etc/reticulum/config") {
			r.ConfigDir = "/etc/reticulum"
		} else if dirExists(filepath.Join(r.UserDir, ".config/reticulum")) &&
			fileExists(filepath.Join(r.UserDir, ".config/reticulum/config")) {
			r.ConfigDir = filepath.Join(r.UserDir, ".config/reticulum")
		} else {
			r.ConfigDir = filepath.Join(r.UserDir, ".reticulum")
		}
	}

	r.ConfigPath = filepath.Join(r.ConfigDir, "config")
	r.StoragePath = filepath.Join(r.ConfigDir, "storage")
	r.CachePath = filepath.Join(r.ConfigDir, "storage/cache")
	r.ResourcePath = filepath.Join(r.ConfigDir, "storage/resources")
	r.IdentityPath = filepath.Join(r.ConfigDir, "storage/identities")
	r.InterfacePath = filepath.Join(r.ConfigDir, "interfaces")

	// logging (adapt log destination to your logger)
	if lf, ok := logdest.(LogFileMarker); ok && lf == LogFile {
		SetLogDestFile(filepath.Join(r.ConfigDir, "logfile"))
	} else if cb, ok := logdest.(func(level int, msg string)); ok {
		SetLogDestCallback(cb)
	}

	// log level
	if loglevel != nil {
		ll := *loglevel
		if ll > LOG_EXTREME {
			ll = LOG_EXTREME
		}
		if ll < LOG_CRITICAL {
			ll = LOG_CRITICAL
		}
		SetLogLevel(ll)
		r.RequestedLoglevel = &ll
	}
	if verbosity != nil {
		v := *verbosity
		r.RequestedVerbosity = &v
	}

	// dirs
	ensureDir(r.StoragePath)
	ensureDir(r.CachePath)
	ensureDir(filepath.Join(r.CachePath, "announces"))
	ensureDir(r.ResourcePath)
	ensureDir(r.IdentityPath)
	ensureDir(r.InterfacePath)

	// Provide persisted Weave Identity semantics to interfaces package
	// without introducing import cycles.
	ifaces.WeaveIdentityProvider = func(port string) ([]byte, func(msg []byte) ([]byte, error), error) {
		if strings.TrimSpace(port) == "" {
			return nil, nil, errors.New("weave: empty port")
		}
		sum := sha256.Sum256([]byte(port))
		keyPath := filepath.Join(r.IdentityPath, "weave_"+hex.EncodeToString(sum[:16])+".id")
		var id *Identity
		if b, err := os.ReadFile(keyPath); err == nil && len(b) > 0 {
			loaded, lerr := IdentityFromBytes(b)
			if lerr != nil {
				return nil, nil, lerr
			}
			id = loaded
		} else {
			created, cerr := NewIdentity()
			if cerr != nil {
				return nil, nil, cerr
			}
			id = created
			_ = id.Save(keyPath)
		}
		pub := id.GetPublicKey()
		if len(pub) != 64 {
			return nil, nil, errors.New("weave: invalid identity public key size")
		}
		sigPub := append([]byte(nil), pub[32:]...)
		sign := func(msg []byte) ([]byte, error) { return id.Sign(msg) }
		return sigPub, sign, nil
	}

	// config
	if fileExists(r.ConfigPath) {
		cfg, err := configobj.Load(r.ConfigPath)
		if err != nil {
			Log("Could not parse configuration at "+r.ConfigPath, LogError)
			Log("Check your configuration file for errors!", LogError)
			Panic()
		}
		r.Config = cfg
	} else {
		Log("Could not load config file, creating default configuration file...", LogInfo)
		if err := r.createDefaultConfig(); err != nil {
			return nil, err
		}
		Log("Default config file created. Edit "+r.ConfigPath+" and restart Reticulum if needed.", LogInfo)
		time.Sleep(1500 * time.Millisecond)
	}

	if sharedInstanceType != nil {
		r.SharedInstanceType = *sharedInstanceType
	}

	if err := r.applyConfig(); err != nil {
		return nil, err
	}

	Logf(LogDebug, "Utilising cryptography backend %q", cryptography.ProviderBackend())
	Log("Configuration loaded from "+r.ConfigPath, LogVerbose)

	_ = IdentityLoadKnownDestinations()
	Start(r) // mirrors RNS.Transport.start(self)

	// choose AF_UNIX vs TCP
	if vendor.UseAFUnix() {
		if r.SharedInstanceType == "tcp" {
			r.UseAFUnix = false
		} else {
			r.UseAFUnix = true
		}
	} else {
		r.SharedInstanceType = "tcp"
		r.UseAFUnix = false
	}

	// Abstract AF_UNIX socket addresses are Linux-only. Fall back to TCP on other platforms.
	if r.UseAFUnix && runtime.GOOS != "linux" {
		r.UseAFUnix = false
		r.SharedInstanceType = "tcp"
	}

	if r.LocalSocketPath == "" && r.UseAFUnix {
		r.LocalSocketPath = "default"
	}

	if r.UseAFUnix {
		r.rpcNetwork = "unix"
		r.rpcAddr = "\x00rns/" + r.LocalSocketPath + "/rpc"
	} else {
		r.rpcNetwork = "tcp"
		r.rpcAddr = fmt.Sprintf("127.0.0.1:%d", r.localControlPort)
	}

	// rpc key
	if r.RPCKey == nil {
		// Python defaults to full_hash(Transport.identity.get_private_key()).
		if TransportIdentity != nil {
			if pk := TransportIdentity.GetPrivateKey(); len(pk) > 0 {
				r.RPCKey = FullHash(pk)
			}
		}
		if r.RPCKey == nil {
			r.RPCKey = FullHash([]byte("reticulum"))
		}
	}

	// In Python, local shared instance setup happens after Transport.start() and after
	// rpc addr/key selection.
	if err := r.startLocalInterface(); err != nil {
		return nil, err
	}

	// If this is shared/standalone — bring up system interfaces.
	if r.IsSharedInstance || r.IsStandaloneInstance {
		Log("Bringing up system interfaces...", LogVerbose)
		if err := r.bringUpSystemInterfaces(); err != nil {
			return nil, err
		}
		Log("System interfaces are ready", LogVerbose)
	}

	// signals and exit handler
	setupExitHandlers()
	SetExitHandler(exitHandler)

	return r, nil
}

func osUserDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "~"
	}
	return home
}

func ensureDir(path string) {
	_ = os.MkdirAll(path, 0o755)
}

func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

// ---------------- exit / signals ----------------

var (
	exitHandlerRan         bool
	interfaceDetachHandler bool
)

func setupExitHandlers() {
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for sig := range ch {
			switch sig {
			case syscall.SIGINT:
				sigintHandler()
			case syscall.SIGTERM:
				sigtermHandler()
			}
		}
	}()
}

func exitHandler() {
	if exitHandlerRan {
		return
	}
	exitHandlerRan = true

	// Python parity: the shared/master instance should not exit while local clients
	// are still connected (unless forcibly terminated). Allow a configurable timeout
	// for test/sandbox environments via `RNS_EXIT_WAIT_TIMEOUT` (seconds).
	if instance != nil && instance.IsSharedInstance {
		waitForLocalClientsToDisconnect(exitWaitTimeoutFromEnv())
	}

	if !interfaceDetachHandler {
		DetachInterfaces()
	}
	// Transport/Identity persistence hooks (best-effort).
	TransportExitHandler()
	IdentityExitHandler()

	if ProfilerRan() {
		ProfilerResults()
	}
	SetLogLevel(-1)
}

func sigintHandler() {
	DetachInterfaces()
	interfaceDetachHandler = true
	Exit()
}

func sigtermHandler() {
	DetachInterfaces()
	interfaceDetachHandler = true
	Exit()
}

func exitWaitTimeoutFromEnv() time.Duration {
	raw := strings.TrimSpace(os.Getenv("RNS_EXIT_WAIT_TIMEOUT"))
	if raw == "" {
		return 0
	}
	seconds, err := strconv.ParseFloat(raw, 64)
	if err != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds * float64(time.Second))
}

func waitForLocalClientsToDisconnect(timeout time.Duration) bool {
	started := time.Now()
	reported := false
	for len(LocalClientInterfaces) > 0 {
		if !reported {
			Logf(LogDebug, "Waiting for %d local client(s) to disconnect before exiting", len(LocalClientInterfaces))
			reported = true
		}
		if timeout > 0 && time.Since(started) > timeout {
			Logf(LogWarning, "Timed out waiting for local clients to disconnect (%d remaining)", len(LocalClientInterfaces))
			return false
		}
		time.Sleep(50 * time.Millisecond)
	}
	return true
}

// ---------------- config ----------------

func (r *Reticulum) applyConfig() error {
	// [logging]
	if r.Config != nil && r.Config.HasSection("logging") {
		sec := r.Config.Section("logging")
		if r.RequestedLoglevel == nil { // like Python: only if not explicitly set
			if v, err := sec.AsInt("loglevel"); err == nil {
				ll := v
				if r.RequestedVerbosity != nil {
					ll += *r.RequestedVerbosity
				}
				if ll < 0 {
					ll = 0
				}
				if ll > 7 {
					ll = 7
				}
				SetLogLevel(ll)
			}
		}
	}

	// [reticulum]
	if r.Config != nil && r.Config.HasSection("reticulum") {
		sec := r.Config.Section("reticulum")
		if mtu, err := sec.AsInt("mtu"); err == nil && mtu > 0 {
			if err := SetMTU(mtu); err != nil {
				Logf(LogWarning, "Ignoring invalid MTU value %d: %v", mtu, err)
			} else {
				Logf(LogDebug, "Configured Reticulum MTU set to %d bytes", mtu)
			}
		}
		if _, ok := sec.Get("share_instance"); ok {
			v, err := sec.AsBool("share_instance")
			if err == nil {
				r.ShareInstance = v
			}
		}
		if vendor.UseAFUnix() {
			if n, ok := sec.Get("instance_name"); ok {
				r.LocalSocketPath = n
			}
		}
		if r.SharedInstanceType == "" {
			if v, ok := sec.Get("shared_instance_type"); ok {
				v = strings.ToLower(v)
				if v == "tcp" || v == "unix" {
					r.SharedInstanceType = v
				}
			}
		}
		if p, err := sec.AsInt("shared_instance_port"); err == nil {
			r.localInterfacePort = p
		}
		if p, err := sec.AsInt("instance_control_port"); err == nil {
			r.localControlPort = p
		}
		if s, ok := sec.Get("rpc_key"); ok && s != "" {
			b, err := hex.DecodeString(s)
			if err != nil {
				Log("Invalid shared instance RPC key, falling back to default", LogError)
				r.RPCKey = nil
			} else {
				r.RPCKey = b
			}
		}
		if v, _ := sec.AsBool("enable_transport"); v {
			transportEnabled = true
		}
		if v, _ := sec.AsBool("link_mtu_discovery"); v {
			linkMTUDiscovery = true
		}
		if v, _ := sec.AsBool("enable_remote_management"); v {
			remoteManagementEnabled = true
		}
		if l := sec.AsList("remote_management_allowed"); len(l) > 0 {
			for _, hexhash := range l {
				destLen := (TRUNCATED_HASHLENGTH / 8) * 2
				if len(hexhash) != destLen {
					return fmt.Errorf("identity hash length for remote management ACL %s is invalid, must be %d hexadecimal characters (%d bytes)", hexhash, destLen, destLen/2)
				}
				b, err := hex.DecodeString(hexhash)
				if err != nil {
					return fmt.Errorf("invalid identity hash for remote management ACL: %s", hexhash)
				}
				if !RemoteManagementAllowedContains(b) {
					AddRemoteManagementAllowed(b)
				}
			}
		}
		if v, _ := sec.AsBool("respond_to_probes"); v {
			allowProbes = true
		}
		if v, err := sec.AsInt("force_shared_instance_bitrate"); err == nil {
			ForceSharedInstanceBitrate(v)
			if v > 0 {
				Logf(LogWarning, "Forcing shared instance bitrate of %s", PrettySpeed(float64(v)))
			}
		}
		if v, err := sec.AsBool("panic_on_interface_error"); err == nil {
			r.PanicOnInterfaceError = v
		}
		if v, err := sec.AsBool("use_implicit_proof"); err == nil {
			useImplicitProof = v
		}
	}

	if Compiled() {
		Log("Reticulum running in compiled mode", LogDebug)
	} else {
		Log("Reticulum running in interpreted mode", LogDebug)
	}
	return nil
}

func (r *Reticulum) createDefaultConfig() error {
	cfg, err := configobj.LoadReader(strings.NewReader(strings.Join(defaultConfigLines, "\n")))
	if err != nil {
		return err
	}
	r.Config = cfg
	ensureDir(r.ConfigDir)
	return cfg.Save(r.ConfigPath)
}

// ---------------- local interface + jobs ----------------

func (r *Reticulum) startJobs() {
	if r.jobsStarted {
		return
	}
	IdentityCleanRatchets()
	r.jobsStarted = true
	go r.jobsLoop()
}

func (r *Reticulum) jobsLoop() {
	for {
		now := time.Now()

		if now.After(r.lastCacheClean.Add(CLEAN_INTERVAL * time.Second)) {
			r.cleanCaches()
			r.lastCacheClean = time.Now()
		}
		if now.After(r.lastDataPersist.Add(PERSIST_INTERVAL * time.Second)) {
			r.persistData()
		}
		time.Sleep(JOB_INTERVAL * time.Second)
	}
}

func (r *Reticulum) startLocalInterface() error {
	if !r.ShareInstance {
		r.IsSharedInstance = false
		r.IsStandaloneInstance = true
		r.IsConnectedToSharedInstance = false
		r.startJobs()
		return nil
	}

	// Try to become the shared instance by binding the RPC listener.
	ln, err := NewRPCListener(r.rpcNetwork, r.rpcAddr, r.RPCKey)
	if err == nil {
		if r.RequireShared {
			_ = ln.Close()
			// Python aborts startup if an existing shared instance was required,
			// but we ended up becoming the shared instance (meaning none existed).
			Log("Existing shared instance required, but this instance started as shared instance. Aborting startup.", LogVerbose)
			return errors.New("no shared instance available, but application that started Reticulum required it")
		}
		r.rpcLn = ln
		r.IsSharedInstance = true
		r.IsStandaloneInstance = false
		r.IsConnectedToSharedInstance = false
		go r.rpcLoop()
		Logf(LogDebug, "Started shared instance RPC listener at %s (%s)", r.rpcLn.Addr(), r.rpcNetwork)

		// Python parity: the first instance creates a local shared-instance interface.
		siName := "default"
		if r.UseAFUnix {
			if strings.TrimSpace(r.LocalSocketPath) != "" {
				siName = strings.TrimSpace(r.LocalSocketPath)
			}
		} else {
			// Python uses the shared instance port in the name when shared instance runs over TCP.
			siName = fmt.Sprintf("%d", r.localInterfacePort)
		}
		si := &Interface{
			Name:              fmt.Sprintf("Shared Instance[%s]", siName),
			Type:              "LocalInterface",
			IN:                true,
			OUT:               true,
			DriverImplemented: true,
			Online:            true,
			AutoconfigureMTU:  true,
		}
		// Python reports the shared instance interface as a 1 Gbps local link by default.
		si.Bitrate = 1000000000
		if br := SharedInstanceForcedBitrate(); br > 0 {
			si.Bitrate = br
		}
		si.ForceBitrateLatency = true
		si.OptimiseMTU()
		AddInterface(si)
		si.SetClientCountFunc(func() int {
			return len(LocalClientInterfaces)
		})
		r.SharedInstanceInterface = si

		// Start LocalInterface packet IPC server (Python local_client_interfaces).
		ifaces.SharedConnectionDisappeared = SharedConnectionDisappeared
		ifaces.SharedConnectionReappeared = SharedConnectionReappeared
		if _, err := ifaces.StartLocalInterfaceServer(ifaces.LocalConfig{
			UseAFUnix:          r.UseAFUnix,
			LocalSocketPath:    r.LocalSocketPath,
			LocalInterfacePort: r.localInterfacePort,
			Parent:             r.SharedInstanceInterface,
			OnClientDisconnect: func(cif *ifaces.Interface) {
				removeLocalClientInterface(cif)
				removeInterface(cif)
			},
		}, func(cif *ifaces.Interface) {
			AddInterface(cif)
			LocalClientInterfaces = append(LocalClientInterfaces, cif)
		}); err != nil {
			Logf(LogError, "Could not start LocalInterface IPC server: %v", err)
		}

		r.startJobs()
		return nil
	}

	// Could not bind: try to connect to an existing shared instance.
	client, dialErr := dialRPC(r.rpcNetwork, r.rpcAddr, r.RPCKey)
	if dialErr == nil {
		_ = client.Close()
		r.IsSharedInstance = false
		r.IsStandaloneInstance = false
		r.IsConnectedToSharedInstance = true
		transportEnabled = false
		remoteManagementEnabled = false
		allowProbes = false
		Logf(LogNotice, "Connected to locally available Reticulum shared instance via RPC")

		// Python parity: create a LocalInterface client interface placeholder.
		siName := "default"
		if r.UseAFUnix {
			if strings.TrimSpace(r.LocalSocketPath) != "" {
				siName = strings.TrimSpace(r.LocalSocketPath)
			}
		} else {
			// Python uses the shared instance port in the name when shared instance runs over TCP.
			siName = fmt.Sprintf("%d", r.localInterfacePort)
		}
		si := &Interface{
			Name:                fmt.Sprintf("LocalInterface[%s]", siName),
			Type:                "LocalInterface",
			IN:                  true,
			OUT:                 true,
			DriverImplemented:   true,
			Online:              true,
			AutoconfigureMTU:    true,
			LocalIsSharedClient: true,
		}
		si.Bitrate = 1000000000
		if br := SharedInstanceForcedBitrate(); br > 0 {
			si.Bitrate = br
		}
		si.ForceBitrateLatency = true
		si.OptimiseMTU()
		AddInterface(si)
		r.SharedInstanceInterface = si

		// Connect LocalInterface packet IPC client to shared instance.
		ifaces.SharedConnectionDisappeared = SharedConnectionDisappeared
		ifaces.SharedConnectionReappeared = SharedConnectionReappeared
		if err := ifaces.ConnectLocalInterfaceClient(ifaces.LocalConfig{
			UseAFUnix:          r.UseAFUnix,
			LocalSocketPath:    r.LocalSocketPath,
			LocalInterfacePort: r.localInterfacePort,
		}, si); err != nil {
			Logf(LogError, "Could not connect to LocalInterface IPC server: %v", err)
		} else {
			Log("Connected to shared instance LocalInterface packet socket", LogNotice)
		}

		return nil
	}

	Log("Local shared instance appears to be running, but it could not be connected", LogError)
	Log("The contained exception was: "+dialErr.Error(), LogError)

	// No shared instance available, fall back to standalone.
	r.IsSharedInstance = false
	r.IsStandaloneInstance = true
	r.IsConnectedToSharedInstance = false
	r.startJobs()
	return nil
}

func (r *Reticulum) bringUpSystemInterfaces() error {
	if r.Config == nil {
		Log("No interfaces configured in reticulum.conf", LogVerbose)
		return nil
	}

	var configs []interfaceConfigEntry
	if ordered, err := parseConfigObjInterfacesOrdered(r.ConfigPath); err == nil && len(ordered) > 0 {
		configs = ordered
	} else {
		ini := parseINIFallbackInterfaces(r.Config)
		if len(ini) == 0 {
			Log("No interfaces configured in reticulum.conf", LogVerbose)
			return nil
		}
		names := make([]string, 0, len(ini))
		for name := range ini {
			names = append(names, name)
		}
		sort.Strings(names)
		configs = make([]interfaceConfigEntry, 0, len(names))
		for _, name := range names {
			configs = append(configs, interfaceConfigEntry{Name: name, KV: ini[name]})
		}
	}

	seenNames := make(map[string]struct{}, len(configs))
	broughtUp := 0

	for _, entry := range configs {
		name := strings.TrimSpace(entry.Name)
		if name == "" {
			continue
		}
		if _, ok := seenNames[name]; ok {
			msg := fmt.Sprintf("The interface name %q was already used. Check your configuration file for errors!", name)
			Log(msg, LogError)
			return errors.New(msg)
		}
		seenNames[name] = struct{}{}

		kv := entry.KV

		// Python parity: an interface is only started if it is explicitly enabled.
		// Missing keys mean disabled.
		enabled := parseTruthy(getFirst(kv, "interface_enabled", "enabled", "enable"), false)
		if !enabled {
			Logf(LogDebug, "Skipping disabled interface %q", name)
			continue
		}

		mode := parseInterfaceMode(getFirst(kv, "interface_mode", "selected_interface_mode", "mode"))

		var bitrate *int
		if v, ok := parseInt(getFirst(kv, "configured_bitrate", "bitrate")); ok && v >= MINIMUM_BITRATE {
			bitrate = &v
		}
		// Python sets interface_config["configured_bitrate"] for all interface drivers.
		if bitrate != nil && strings.TrimSpace(getFirst(kv, "configured_bitrate")) == "" {
			kv["configured_bitrate"] = strconv.Itoa(*bitrate)
		}

		// Python parity: announce_cap is specified as percent (0 < v <= 100) and stored as fraction.
		var announceCap *float64
		if v, ok := parseFloat(getFirst(kv, "announce_cap")); ok && v > 0 && v <= 100 {
			f := v / 100.0
			announceCap = &f
		}

		var ifacSize *int
		if v, ok := parseInt(getFirst(kv, "ifac_size")); ok && v >= IFAC_MIN_SIZE*8 {
			sz := v / 8
			ifacSize = &sz
		}

		netName := getFirst(kv, "networkname", "network_name")
		var ifacNetname *string
		if strings.TrimSpace(netName) != "" {
			ifacNetname = &netName
		}

		netKey := getFirst(kv, "passphrase", "pass_phrase")
		var ifacNetkey *string
		if strings.TrimSpace(netKey) != "" {
			ifacNetkey = &netKey
		}

		var announceRateTarget *int
		if v, ok := parseInt(getFirst(kv, "announce_rate_target")); ok && v > 0 {
			announceRateTarget = &v
		}
		var announceRateGrace *int
		if v, ok := parseInt(getFirst(kv, "announce_rate_grace")); ok && v >= 0 {
			announceRateGrace = &v
		}
		var announceRatePenalty *int
		if v, ok := parseInt(getFirst(kv, "announce_rate_penalty")); ok && v >= 0 {
			announceRatePenalty = &v
		}
		if announceRateTarget != nil && announceRateGrace == nil {
			zero := 0
			announceRateGrace = &zero
		}
		if announceRateTarget != nil && announceRatePenalty == nil {
			zero := 0
			announceRatePenalty = &zero
		}

		ingressControl := parseTruthy(getFirst(kv, "ingress_control"), true)
		var icMaxHeld *int
		if v, ok := parseInt(getFirst(kv, "ic_max_held_announces")); ok {
			icMaxHeld = &v
		}
		var icBurstHold *float64
		if v, ok := parseFloat(getFirst(kv, "ic_burst_hold")); ok {
			icBurstHold = &v
		}
		var icBurstFreqNew *float64
		if v, ok := parseFloat(getFirst(kv, "ic_burst_freq_new")); ok {
			icBurstFreqNew = &v
		}
		var icBurstFreq *float64
		if v, ok := parseFloat(getFirst(kv, "ic_burst_freq")); ok {
			icBurstFreq = &v
		}
		var icNewTime *float64
		if v, ok := parseFloat(getFirst(kv, "ic_new_time")); ok {
			icNewTime = &v
		}
		var icBurstPenalty *float64
		if v, ok := parseFloat(getFirst(kv, "ic_burst_penalty")); ok {
			icBurstPenalty = &v
		}
		var icHeldRelease *float64
		if v, ok := parseFloat(getFirst(kv, "ic_held_release_interval")); ok {
			icHeldRelease = &v
		}

		ifType := strings.TrimSpace(getFirst(kv, "type"))
		if ifType == "" {
			msg := fmt.Sprintf("The interface %q could not be created. Check your configuration file for errors!", name)
			Log(msg, LogError)
			Log("The contained exception was: missing interface type", LogError)
			return errors.New("missing interface type")
		}

		// Python compatibility: normalise Backbone key aliases.
		if strings.EqualFold(ifType, "BackboneInterface") || strings.EqualFold(ifType, "BackboneClientInterface") {
			if v := strings.TrimSpace(getFirst(kv, "port")); v != "" {
				if strings.TrimSpace(getFirst(kv, "listen_port")) == "" {
					kv["listen_port"] = v
				}
				if strings.TrimSpace(getFirst(kv, "target_port")) == "" {
					kv["target_port"] = v
				}
			}
			if v := strings.TrimSpace(getFirst(kv, "remote")); v != "" && strings.TrimSpace(getFirst(kv, "target_host")) == "" {
				kv["target_host"] = v
			}
			if v := strings.TrimSpace(getFirst(kv, "listen_on")); v != "" && strings.TrimSpace(getFirst(kv, "listen_ip")) == "" {
				kv["listen_ip"] = v
			}
		}

		// Python parity: BackboneInterface behaves like BackboneClientInterface
		// when a target host is specified.
		if strings.EqualFold(ifType, "BackboneInterface") && strings.TrimSpace(getFirst(kv, "target_host", "remote")) != "" {
			ifType = "BackboneClientInterface"
		}

		// Config parity convenience: allow `TCPInterface` to mean either a
		// client (when target_host/target_port is provided) or a server
		// (when listen_ip/device is provided).
		if strings.EqualFold(ifType, "TCPInterface") {
			if strings.TrimSpace(getFirst(kv, "target_host", "remote")) != "" {
				ifType = "TCPClientInterface"
			} else {
				ifType = "TCPServerInterface"
			}
		}

		knownType := false
		switch strings.ToLower(ifType) {
		case "ax25kissinterface", "autointerface", "udpinterface", "tcpclientinterface", "tcpserverinterface", "serialinterface", "kissinterface", "rnodeinterface", "rnodemultiinterface", "i2pinterface", "weaveinterface", "backboneinterface", "backboneclientinterface", "pipeinterface":
			knownType = true
		}
		if !knownType {
			// Python parity: if no internal interface matched, try interfacepath.
			if strings.TrimSpace(r.InterfacePath) != "" {
				py := filepath.Join(r.InterfacePath, ifType+".py")
				goSrc := filepath.Join(r.InterfacePath, ifType+".go")
				if fileExists(py) {
					msg := fmt.Sprintf("External interface initialisation failed for %s / %s (external Python interfaces are not supported in Go port)", ifType, name)
					Log(msg, LogError)
					return errors.New(msg)
				}
				if fileExists(goSrc) {
					msg := fmt.Sprintf("External interface %q found at %s but external Go interfaces are not supported yet", ifType, goSrc)
					Log(msg, LogError)
					return errors.New(msg)
				}
			}
			Logf(LogError, "Could not locate external interface module %q in %q", ifType+".py", r.InterfacePath)
			continue
		}

		base, err := buildInterfaceFromType(strings.TrimSpace(name), ifType)
		if err != nil {
			msg := fmt.Sprintf("The interface %q could not be created. Check your configuration file for errors!", name)
			Log(msg, LogError)
			Log("The contained exception was: "+err.Error(), LogError)
			return err
		}
		if base == nil {
			return fmt.Errorf("interface %q type %q initialisation returned nil", name, ifType)
		}

		// Python parity: default outgoing is enabled, but can be disabled via `outgoing = False`.
		base.OUT = parseTruthy(getFirst(kv, "outgoing"), true)
		base.IngressControl = ingressControl
		base.ICMaxHeldAnnounces = icMaxHeld
		base.ICBurstHold = icBurstHold
		base.ICBurstFreqNew = icBurstFreqNew
		base.ICBurstFreq = icBurstFreq
		base.ICNewTime = icNewTime
		base.ICBurstPenalty = icBurstPenalty
		base.ICHeldReleaseInterval = icHeldRelease
		base.SetAnnounceRateConfig(announceRateTarget, announceRateGrace, announceRatePenalty)

		ifc := base

		if strings.EqualFold(ifType, "UDPInterface") {
			// Python: IN=True, BITRATE_GUESS=10Mbps, DEFAULT_IFAC_SIZE=16, HW_MTU=1064
			ifc.IN = true
			if ifc.Bitrate == 0 {
				ifc.Bitrate = 10 * 1000 * 1000
			}
			if ifc.HWMTU == 0 {
				ifc.HWMTU = 1064
			}
			if ifc.IFACSize == 0 {
				ifc.IFACSize = 16
			}
			ifc.FixedMTU = true

			device := getFirst(kv, "device")
			port, _ := parseInt(getFirst(kv, "port"))
			listenIP := getFirst(kv, "listen_ip")
			listenPort, _ := parseInt(getFirst(kv, "listen_port"))
			forwardIP := getFirst(kv, "forward_ip")
			forwardPort, _ := parseInt(getFirst(kv, "forward_port"))
			if port > 0 {
				if listenPort == 0 {
					listenPort = port
				}
				if forwardPort == 0 {
					forwardPort = port
				}
			}
			if strings.TrimSpace(device) != "" {
				bcast, derr := broadcastForInterface(device)
				if derr != nil {
					return fmt.Errorf("Interface %q device %q error: %w", name, device, derr)
				}
				if listenIP == "" {
					listenIP = bcast.String()
				}
				if forwardIP == "" {
					forwardIP = bcast.String()
				}
			}
			if err := ifc.ConfigureUDP(listenIP, listenPort, forwardIP, forwardPort); err != nil {
				return fmt.Errorf("Interface %q UDP config error: %w", name, err)
			}
			if err := ifc.StartUDP(); err != nil {
				return fmt.Errorf("Interface %q UDP listener error: %w", name, err)
			}
		}

		if strings.EqualFold(ifType, "AutoInterface") {
			desiredOut := ifc.OUT
			if err := ifc.ConfigureAutoInterface(kv); err != nil {
				return fmt.Errorf("Interface %q AutoInterface config error: %w", name, err)
			}
			ifc.OUT = desiredOut
			if err := ifc.StartAutoInterface(); err != nil {
				return fmt.Errorf("Interface %q AutoInterface start error: %w", name, err)
			}
		}

		if strings.EqualFold(ifType, "AX25KISSInterface") {
			axIf, err := ifaces.NewAX25KISSInterface(strings.TrimSpace(name), kv)
			if err != nil {
				return fmt.Errorf("Interface %q AX25KISS config error: %w", name, err)
			}
			inheritInterfaceConfig(axIf, ifc)
			ifc = axIf
		}

		if strings.EqualFold(ifType, "KISSInterface") {
			kIf, err := ifaces.NewKISSInterface(strings.TrimSpace(name), kv)
			if err != nil {
				return fmt.Errorf("Interface %q KISS config error: %w", name, err)
			}
			inheritInterfaceConfig(kIf, ifc)
			ifc = kIf
		}

		if strings.EqualFold(ifType, "BackboneInterface") {
			bbIf, err := ifaces.NewBackboneInterface(strings.TrimSpace(name), kv)
			if err != nil {
				return fmt.Errorf("Interface %q Backbone config error: %w", name, err)
			}
			inheritInterfaceConfig(bbIf, ifc)
			ifc = bbIf
		}

		if strings.EqualFold(ifType, "BackboneClientInterface") {
			bcIf, err := ifaces.NewBackboneClientInterface(strings.TrimSpace(name), kv)
			if err != nil {
				return fmt.Errorf("Interface %q Backbone client config error: %w", name, err)
			}
			inheritInterfaceConfig(bcIf, ifc)
			ifc = bcIf
		}

		if strings.EqualFold(ifType, "WeaveInterface") {
			wIf, err := ifaces.NewWeaveInterface(strings.TrimSpace(name), kv)
			if err != nil {
				return fmt.Errorf("Interface %q Weave config error: %w", name, err)
			}
			inheritInterfaceConfig(wIf, ifc)
			// Python parity: WeaveInterface.should_ingress_limit() always false.
			wIf.IngressControl = false
			ifc = wIf
		}

		if strings.EqualFold(ifType, "I2PInterface") {
			// Python injects Reticulum.storagepath into I2PInterface config.
			if strings.TrimSpace(getFirst(kv, "storagepath")) == "" && strings.TrimSpace(r.StoragePath) != "" {
				kv["storagepath"] = r.StoragePath
			}
			i2pIf, err := ifaces.NewI2PInterface(strings.TrimSpace(name), kv)
			if err != nil {
				return fmt.Errorf("Interface %q I2P config error: %w", name, err)
			}
			inheritInterfaceConfig(i2pIf, ifc)
			ifc = i2pIf
		}

		if strings.EqualFold(ifType, "PipeInterface") {
			pIf, err := ifaces.NewPipeInterface(strings.TrimSpace(name), kv)
			if err != nil {
				return fmt.Errorf("Interface %q Pipe config error: %w", name, err)
			}
			inheritInterfaceConfig(pIf, ifc)
			ifc = pIf
		}

		if strings.EqualFold(ifType, "SerialInterface") {
			sIf, err := ifaces.NewSerialInterface(strings.TrimSpace(name), kv)
			if err != nil {
				return fmt.Errorf("Interface %q Serial config error: %w", name, err)
			}
			inheritInterfaceConfig(sIf, ifc)
			ifc = sIf
		}

		if strings.EqualFold(ifType, "RNodeMultiInterface") {
			rnmIf, err := ifaces.NewRNodeMultiInterface(strings.TrimSpace(name), kv)
			if err != nil {
				return fmt.Errorf("Interface %q RNodeMulti config error: %w", name, err)
			}
			inheritInterfaceConfig(rnmIf, ifc)
			ifc = rnmIf
		}

		if strings.EqualFold(ifType, "RNodeInterface") {
			rnIf, err := ifaces.NewRNodeInterfaceFromConfig(strings.TrimSpace(name), kv)
			if err != nil {
				return fmt.Errorf("Interface %q RNode config error: %w", name, err)
			}
			inheritInterfaceConfig(rnIf, ifc)
			ifc = rnIf
		}

		if strings.EqualFold(ifType, "TCPClientInterface") {
			host := getFirst(kv, "target_host", "remote")
			port, _ := parseInt(getFirst(kv, "target_port", "port"))
			kiss := parseTruthy(getFirst(kv, "kiss_framing"), false)
			i2p := parseTruthy(getFirst(kv, "i2p_tunneled"), false)
			fixedMTU, _ := parseInt(getFirst(kv, "fixed_mtu", "mtu"))
			connectTimeoutS, _ := parseFloat(getFirst(kv, "connect_timeout"))
			reconnectWaitS, _ := parseFloat(getFirst(kv, "reconnect_wait"))
			var maxReconnect *int
			if v, ok := parseInt(getFirst(kv, "max_reconnect_tries")); ok {
				maxReconnect = &v
			}

			ifacSz := 0
			if ifacSize != nil {
				ifacSz = *ifacSize
			}
			ifacNetnameVal := ""
			if ifacNetname != nil {
				ifacNetnameVal = *ifacNetname
			}
			ifacNetkeyVal := ""
			if ifacNetkey != nil {
				ifacNetkeyVal = *ifacNetkey
			}

			tcpIf, err := ifaces.NewTCPClientInterfaceFromConfig(ifaces.TCPClientConfig{
				Name:            strings.TrimSpace(name),
				TargetHost:      host,
				TargetPort:      port,
				KISSFraming:     kiss,
				I2PTunneled:     i2p,
				FixedMTU:        fixedMTU,
				ConnectTimeout:  time.Duration(connectTimeoutS * float64(time.Second)),
				ReconnectWait:   time.Duration(reconnectWaitS * float64(time.Second)),
				MaxReconnectTry: maxReconnect,
				IFACSize:        ifacSz,
				IFACNetname:     ifacNetnameVal,
				IFACNetkey:      ifacNetkeyVal,
			})
			if err != nil {
				return fmt.Errorf("Interface %q TCP client config error: %w", name, err)
			}
			inheritInterfaceConfig(tcpIf, ifc)
			ifc = tcpIf
		}

		if strings.EqualFold(ifType, "TCPServerInterface") {
			listenIP := getFirst(kv, "listen_ip", "listen_on", "bind_ip")
			listenPort, _ := parseInt(getFirst(kv, "listen_port", "port"))
			device := getFirst(kv, "device")
			preferIPv6 := parseTruthy(getFirst(kv, "prefer_ipv6"), false)
			kiss := parseTruthy(getFirst(kv, "kiss_framing"), false)
			i2p := parseTruthy(getFirst(kv, "i2p_tunneled"), false)
			fixedMTU, _ := parseInt(getFirst(kv, "fixed_mtu", "mtu"))

			ifacSz := 0
			if ifacSize != nil {
				ifacSz = *ifacSize
			}
			ifacNetnameVal := ""
			if ifacNetname != nil {
				ifacNetnameVal = *ifacNetname
			}
			ifacNetkeyVal := ""
			if ifacNetkey != nil {
				ifacNetkeyVal = *ifacNetkey
			}

			logger := tcpLogAdapter{}
			server, err := ifaces.NewTCPServerFromConfig(nil, logger, ifaces.TCPServerConfig{
				Name:        strings.TrimSpace(name),
				Device:      device,
				ListenIP:    listenIP,
				ListenPort:  listenPort,
				PreferIPv6:  preferIPv6,
				KISSFraming: kiss,
				I2PTunneled: i2p,
				FixedMTU:    fixedMTU,
				IFACSize:    ifacSz,
				IFACNetname: ifacNetnameVal,
				IFACNetkey:  ifacNetkeyVal,
			})
			if err != nil {
				return fmt.Errorf("Interface %q TCP server config error: %w", name, err)
			}

			parent := &Interface{
				Name:              strings.TrimSpace(name),
				Type:              "TCPServerInterface",
				IN:                true,
				OUT:               false,
				DriverImplemented: true,
				Online:            true,
				Bitrate:           server.Bitrate,
				HWMTU:             server.HWMTU,
				AutoconfigureMTU:  !server.FixedMTU,
				FixedMTU:          server.FixedMTU,
				IFACSize:          server.IFACSize,
				IFACNetnameVal:    server.IFACNetnameVal,
				IFACNetkeyVal:     server.IFACNetkeyVal,
				IFACKey:           append([]byte(nil), server.IFACKey...),
				IFACIdentity:      server.IFACIdentity,
				IFACSignature:     append([]byte(nil), server.IFACSignature...),
			}
			parent.SetTCPServer(server)
			parent.SetClientCountFunc(server.Clients)

			// Wrap accepted clients as Interfaces for Transport.
			server.OnNewClient = func(ci *ifaces.TCPClientInterface) {
				if ci == nil {
					return
				}
				peer := &Interface{
					Name:              ci.String(),
					Type:              "TCPClientInterface",
					Parent:            parent,
					IN:                true,
					OUT:               true,
					DriverImplemented: true,
					Online:            true,
					Bitrate:           parent.Bitrate,
					HWMTU:             parent.HWMTU,
					AutoconfigureMTU:  parent.AutoconfigureMTU,
					FixedMTU:          parent.FixedMTU,
					IFACSize:          parent.IFACSize,
					IFACNetnameVal:    parent.IFACNetnameVal,
					IFACNetkeyVal:     parent.IFACNetkeyVal,
					IFACKey:           append([]byte(nil), parent.IFACKey...),
					IFACIdentity:      parent.IFACIdentity,
					IFACSignature:     append([]byte(nil), parent.IFACSignature...),
				}
				peer.SetTCPClient(ci)
				ci.Owner = tcpOwnerAdapter{ifc: peer}
				AddInterface(peer)
			}

			if err := server.Start(); err != nil {
				return fmt.Errorf("Interface %q TCP server start error: %w", name, err)
			}

			inheritInterfaceConfig(parent, ifc)
			ifc = parent
		}

		r.AddInterface(ifc, mode, bitrate, ifacSize, ifacNetname, ifacNetkey, announceCap, announceRateTarget, announceRateGrace, announceRatePenalty)
		broughtUp++
	}

	if broughtUp == 0 {
		Log("No enabled interfaces could be brought up from config; interface drivers are not ported yet.", LogWarning)
		return nil
	}
	Logf(LogDebug, "Configured %d interface(s) from config", broughtUp)
	return nil
}

func (r *Reticulum) startInterfaceFromConfig(name string, kv map[string]string) (bool, error) {
	if r == nil {
		return false, errors.New("nil reticulum")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return false, errors.New("missing interface name")
	}
	if kv == nil {
		return false, errors.New("missing interface config")
	}

	enabled := parseTruthy(getFirst(kv, "interface_enabled", "enabled", "enable"), false)
	if !enabled {
		return false, nil
	}

	mode := parseInterfaceMode(getFirst(kv, "interface_mode", "selected_interface_mode", "mode"))

	var bitrate *int
	if v, ok := parseInt(getFirst(kv, "configured_bitrate", "bitrate")); ok && v >= MINIMUM_BITRATE {
		bitrate = &v
	}
	// Python sets interface_config["configured_bitrate"] for all interface drivers.
	if bitrate != nil && strings.TrimSpace(getFirst(kv, "configured_bitrate")) == "" {
		kv["configured_bitrate"] = strconv.Itoa(*bitrate)
	}

	var announceCap *float64
	if v, ok := parseFloat(getFirst(kv, "announce_cap")); ok && v > 0 && v <= 100 {
		f := v / 100.0
		announceCap = &f
	}

	var ifacSize *int
	if v, ok := parseInt(getFirst(kv, "ifac_size")); ok && v >= IFAC_MIN_SIZE*8 {
		sz := v / 8
		ifacSize = &sz
	}

	netName := getFirst(kv, "networkname", "network_name")
	var ifacNetname *string
	if strings.TrimSpace(netName) != "" {
		ifacNetname = &netName
	}

	netKey := getFirst(kv, "passphrase", "pass_phrase")
	var ifacNetkey *string
	if strings.TrimSpace(netKey) != "" {
		ifacNetkey = &netKey
	}

	var announceRateTarget *int
	if v, ok := parseInt(getFirst(kv, "announce_rate_target")); ok && v > 0 {
		announceRateTarget = &v
	}
	var announceRateGrace *int
	if v, ok := parseInt(getFirst(kv, "announce_rate_grace")); ok && v >= 0 {
		announceRateGrace = &v
	}
	var announceRatePenalty *int
	if v, ok := parseInt(getFirst(kv, "announce_rate_penalty")); ok && v >= 0 {
		announceRatePenalty = &v
	}
	if announceRateTarget != nil && announceRateGrace == nil {
		zero := 0
		announceRateGrace = &zero
	}
	if announceRateTarget != nil && announceRatePenalty == nil {
		zero := 0
		announceRatePenalty = &zero
	}

	ingressControl := parseTruthy(getFirst(kv, "ingress_control"), true)
	var icMaxHeld *int
	if v, ok := parseInt(getFirst(kv, "ic_max_held_announces")); ok {
		icMaxHeld = &v
	}
	var icBurstHold *float64
	if v, ok := parseFloat(getFirst(kv, "ic_burst_hold")); ok {
		icBurstHold = &v
	}
	var icBurstFreqNew *float64
	if v, ok := parseFloat(getFirst(kv, "ic_burst_freq_new")); ok {
		icBurstFreqNew = &v
	}
	var icBurstFreq *float64
	if v, ok := parseFloat(getFirst(kv, "ic_burst_freq")); ok {
		icBurstFreq = &v
	}
	var icNewTime *float64
	if v, ok := parseFloat(getFirst(kv, "ic_new_time")); ok {
		icNewTime = &v
	}
	var icBurstPenalty *float64
	if v, ok := parseFloat(getFirst(kv, "ic_burst_penalty")); ok {
		icBurstPenalty = &v
	}
	var icHeldRelease *float64
	if v, ok := parseFloat(getFirst(kv, "ic_held_release_interval")); ok {
		icHeldRelease = &v
	}

	ifType := strings.TrimSpace(getFirst(kv, "type"))
	if ifType == "" {
		return false, errors.New("missing interface type")
	}

	if strings.EqualFold(ifType, "BackboneInterface") || strings.EqualFold(ifType, "BackboneClientInterface") {
		if v := strings.TrimSpace(getFirst(kv, "port")); v != "" {
			if strings.TrimSpace(getFirst(kv, "listen_port")) == "" {
				kv["listen_port"] = v
			}
			if strings.TrimSpace(getFirst(kv, "target_port")) == "" {
				kv["target_port"] = v
			}
		}
		if v := strings.TrimSpace(getFirst(kv, "remote")); v != "" && strings.TrimSpace(getFirst(kv, "target_host")) == "" {
			kv["target_host"] = v
		}
		if v := strings.TrimSpace(getFirst(kv, "listen_on")); v != "" && strings.TrimSpace(getFirst(kv, "listen_ip")) == "" {
			kv["listen_ip"] = v
		}
	}

	if strings.EqualFold(ifType, "BackboneInterface") && strings.TrimSpace(getFirst(kv, "target_host", "remote")) != "" {
		ifType = "BackboneClientInterface"
	}

	if strings.EqualFold(ifType, "TCPInterface") {
		if strings.TrimSpace(getFirst(kv, "target_host", "remote")) != "" {
			ifType = "TCPClientInterface"
		} else {
			ifType = "TCPServerInterface"
		}
	}

	if !strings.EqualFold(ifType, "UDPInterface") &&
		!strings.EqualFold(ifType, "TCPClientInterface") &&
		!strings.EqualFold(ifType, "TCPServerInterface") &&
		!strings.EqualFold(ifType, "AutoInterface") &&
		!strings.EqualFold(ifType, "AX25KISSInterface") &&
		!strings.EqualFold(ifType, "KISSInterface") &&
		!strings.EqualFold(ifType, "BackboneInterface") &&
		!strings.EqualFold(ifType, "BackboneClientInterface") &&
		!strings.EqualFold(ifType, "WeaveInterface") &&
		!strings.EqualFold(ifType, "I2PInterface") &&
		!strings.EqualFold(ifType, "PipeInterface") &&
		!strings.EqualFold(ifType, "SerialInterface") &&
		!strings.EqualFold(ifType, "RNodeInterface") &&
		!strings.EqualFold(ifType, "RNodeMultiInterface") {
		if strings.TrimSpace(r.InterfacePath) != "" {
			pyPath := filepath.Join(r.InterfacePath, ifType+".py")
			goPath := filepath.Join(r.InterfacePath, ifType+".go")
			if _, err := os.Stat(pyPath); err == nil {
				return false, fmt.Errorf("external Python interfaces are not supported: found %q", pyPath)
			}
			if _, err := os.Stat(goPath); err == nil {
				return false, fmt.Errorf("external Go interfaces are not supported yet: found %q", goPath)
			}
		}
		Logf(LogError, "Could not locate external interface module %q in %q", ifType+".py", r.InterfacePath)
		return false, nil
	}

	base, err := buildInterfaceFromType(strings.TrimSpace(name), ifType)
	if err != nil {
		return false, err
	}
	if base == nil {
		return false, fmt.Errorf("interface %q type %q initialisation returned nil", name, ifType)
	}

	base.OUT = parseTruthy(getFirst(kv, "outgoing"), true)
	base.IngressControl = ingressControl
	base.ICMaxHeldAnnounces = icMaxHeld
	base.ICBurstHold = icBurstHold
	base.ICBurstFreqNew = icBurstFreqNew
	base.ICBurstFreq = icBurstFreq
	base.ICNewTime = icNewTime
	base.ICBurstPenalty = icBurstPenalty
	base.ICHeldReleaseInterval = icHeldRelease
	base.SetAnnounceRateConfig(announceRateTarget, announceRateGrace, announceRatePenalty)

	ifc := base

	if strings.EqualFold(ifType, "UDPInterface") {
		ifc.IN = true
		if ifc.Bitrate == 0 {
			ifc.Bitrate = 10 * 1000 * 1000
		}
		if ifc.HWMTU == 0 {
			ifc.HWMTU = 1064
		}
		if ifc.IFACSize == 0 {
			ifc.IFACSize = 16
		}
		ifc.FixedMTU = true

		device := getFirst(kv, "device")
		port, _ := parseInt(getFirst(kv, "port"))
		listenIP := getFirst(kv, "listen_ip")
		listenPort, _ := parseInt(getFirst(kv, "listen_port"))
		forwardIP := getFirst(kv, "forward_ip")
		forwardPort, _ := parseInt(getFirst(kv, "forward_port"))
		if port > 0 {
			if listenPort == 0 {
				listenPort = port
			}
			if forwardPort == 0 {
				forwardPort = port
			}
		}
		if strings.TrimSpace(device) != "" {
			bcast, derr := broadcastForInterface(device)
			if derr != nil {
				return false, fmt.Errorf("Interface %q device %q error: %w", name, device, derr)
			}
			if listenIP == "" {
				listenIP = bcast.String()
			}
			if forwardIP == "" {
				forwardIP = bcast.String()
			}
		}
		if err := ifc.ConfigureUDP(listenIP, listenPort, forwardIP, forwardPort); err != nil {
			return false, fmt.Errorf("Interface %q UDP config error: %w", name, err)
		}
		if err := ifc.StartUDP(); err != nil {
			return false, fmt.Errorf("Interface %q UDP listener error: %w", name, err)
		}
	}

	if strings.EqualFold(ifType, "AutoInterface") {
		desiredOut := ifc.OUT
		if err := ifc.ConfigureAutoInterface(kv); err != nil {
			return false, fmt.Errorf("Interface %q AutoInterface config error: %w", name, err)
		}
		ifc.OUT = desiredOut
		if err := ifc.StartAutoInterface(); err != nil {
			return false, fmt.Errorf("Interface %q AutoInterface start error: %w", name, err)
		}
	}

	if strings.EqualFold(ifType, "AX25KISSInterface") {
		axIf, err := ifaces.NewAX25KISSInterface(strings.TrimSpace(name), kv)
		if err != nil {
			return false, fmt.Errorf("Interface %q AX25KISS config error: %w", name, err)
		}
		inheritInterfaceConfig(axIf, ifc)
		ifc = axIf
	}

	if strings.EqualFold(ifType, "KISSInterface") {
		kIf, err := ifaces.NewKISSInterface(strings.TrimSpace(name), kv)
		if err != nil {
			return false, fmt.Errorf("Interface %q KISS config error: %w", name, err)
		}
		inheritInterfaceConfig(kIf, ifc)
		ifc = kIf
	}

	if strings.EqualFold(ifType, "BackboneInterface") {
		bbIf, err := ifaces.NewBackboneInterface(strings.TrimSpace(name), kv)
		if err != nil {
			return false, fmt.Errorf("Interface %q Backbone config error: %w", name, err)
		}
		inheritInterfaceConfig(bbIf, ifc)
		ifc = bbIf
	}

	if strings.EqualFold(ifType, "BackboneClientInterface") {
		bcIf, err := ifaces.NewBackboneClientInterface(strings.TrimSpace(name), kv)
		if err != nil {
			return false, fmt.Errorf("Interface %q Backbone client config error: %w", name, err)
		}
		inheritInterfaceConfig(bcIf, ifc)
		ifc = bcIf
	}

	if strings.EqualFold(ifType, "WeaveInterface") {
		wIf, err := ifaces.NewWeaveInterface(strings.TrimSpace(name), kv)
		if err != nil {
			return false, fmt.Errorf("Interface %q Weave config error: %w", name, err)
		}
		inheritInterfaceConfig(wIf, ifc)
		wIf.IngressControl = false
		ifc = wIf
	}

	if strings.EqualFold(ifType, "I2PInterface") {
		if strings.TrimSpace(getFirst(kv, "storagepath")) == "" && strings.TrimSpace(r.StoragePath) != "" {
			kv["storagepath"] = r.StoragePath
		}
		i2pIf, err := ifaces.NewI2PInterface(strings.TrimSpace(name), kv)
		if err != nil {
			return false, fmt.Errorf("Interface %q I2P config error: %w", name, err)
		}
		inheritInterfaceConfig(i2pIf, ifc)
		ifc = i2pIf
	}

	if strings.EqualFold(ifType, "PipeInterface") {
		pIf, err := ifaces.NewPipeInterface(strings.TrimSpace(name), kv)
		if err != nil {
			return false, fmt.Errorf("Interface %q Pipe config error: %w", name, err)
		}
		inheritInterfaceConfig(pIf, ifc)
		ifc = pIf
	}

	if strings.EqualFold(ifType, "SerialInterface") {
		sIf, err := ifaces.NewSerialInterface(strings.TrimSpace(name), kv)
		if err != nil {
			return false, fmt.Errorf("Interface %q Serial config error: %w", name, err)
		}
		inheritInterfaceConfig(sIf, ifc)
		ifc = sIf
	}

	if strings.EqualFold(ifType, "RNodeMultiInterface") {
		rnmIf, err := ifaces.NewRNodeMultiInterface(strings.TrimSpace(name), kv)
		if err != nil {
			return false, fmt.Errorf("Interface %q RNodeMulti config error: %w", name, err)
		}
		inheritInterfaceConfig(rnmIf, ifc)
		ifc = rnmIf
	}

	if strings.EqualFold(ifType, "RNodeInterface") {
		rnIf, err := ifaces.NewRNodeInterfaceFromConfig(strings.TrimSpace(name), kv)
		if err != nil {
			return false, fmt.Errorf("Interface %q RNode config error: %w", name, err)
		}
		inheritInterfaceConfig(rnIf, ifc)
		ifc = rnIf
	}

	if strings.EqualFold(ifType, "TCPClientInterface") {
		host := getFirst(kv, "target_host", "remote")
		port, _ := parseInt(getFirst(kv, "target_port", "port"))
		kiss := parseTruthy(getFirst(kv, "kiss_framing"), false)
		i2p := parseTruthy(getFirst(kv, "i2p_tunneled"), false)
		fixedMTU, _ := parseInt(getFirst(kv, "fixed_mtu", "mtu"))
		connectTimeoutS, _ := parseFloat(getFirst(kv, "connect_timeout"))
		reconnectWaitS, _ := parseFloat(getFirst(kv, "reconnect_wait"))
		var maxReconnect *int
		if v, ok := parseInt(getFirst(kv, "max_reconnect_tries")); ok {
			maxReconnect = &v
		}

		ifacSz := 0
		if ifacSize != nil {
			ifacSz = *ifacSize
		}
		ifacNetnameVal := ""
		if ifacNetname != nil {
			ifacNetnameVal = *ifacNetname
		}
		ifacNetkeyVal := ""
		if ifacNetkey != nil {
			ifacNetkeyVal = *ifacNetkey
		}

		tcpIf, err := ifaces.NewTCPClientInterfaceFromConfig(ifaces.TCPClientConfig{
			Name:            strings.TrimSpace(name),
			TargetHost:      host,
			TargetPort:      port,
			KISSFraming:     kiss,
			I2PTunneled:     i2p,
			FixedMTU:        fixedMTU,
			ConnectTimeout:  time.Duration(connectTimeoutS * float64(time.Second)),
			ReconnectWait:   time.Duration(reconnectWaitS * float64(time.Second)),
			MaxReconnectTry: maxReconnect,
			IFACSize:        ifacSz,
			IFACNetname:     ifacNetnameVal,
			IFACNetkey:      ifacNetkeyVal,
		})
		if err != nil {
			return false, fmt.Errorf("Interface %q TCP client config error: %w", name, err)
		}
		inheritInterfaceConfig(tcpIf, ifc)
		ifc = tcpIf
	}

	if strings.EqualFold(ifType, "TCPServerInterface") {
		listenIP := getFirst(kv, "listen_ip", "listen_on", "bind_ip")
		listenPort, _ := parseInt(getFirst(kv, "listen_port", "port"))
		device := getFirst(kv, "device")
		preferIPv6 := parseTruthy(getFirst(kv, "prefer_ipv6"), false)
		kiss := parseTruthy(getFirst(kv, "kiss_framing"), false)
		i2p := parseTruthy(getFirst(kv, "i2p_tunneled"), false)
		fixedMTU, _ := parseInt(getFirst(kv, "fixed_mtu", "mtu"))

		ifacSz := 0
		if ifacSize != nil {
			ifacSz = *ifacSize
		}
		ifacNetnameVal := ""
		if ifacNetname != nil {
			ifacNetnameVal = *ifacNetname
		}
		ifacNetkeyVal := ""
		if ifacNetkey != nil {
			ifacNetkeyVal = *ifacNetkey
		}

		logger := tcpLogAdapter{}
		server, err := ifaces.NewTCPServerFromConfig(nil, logger, ifaces.TCPServerConfig{
			Name:        strings.TrimSpace(name),
			Device:      device,
			ListenIP:    listenIP,
			ListenPort:  listenPort,
			PreferIPv6:  preferIPv6,
			KISSFraming: kiss,
			I2PTunneled: i2p,
			FixedMTU:    fixedMTU,
			IFACSize:    ifacSz,
			IFACNetname: ifacNetnameVal,
			IFACNetkey:  ifacNetkeyVal,
		})
		if err != nil {
			return false, fmt.Errorf("Interface %q TCP server config error: %w", name, err)
		}

		parent := &Interface{
			Name:              strings.TrimSpace(name),
			Type:              "TCPServerInterface",
			IN:                true,
			OUT:               false,
			DriverImplemented: true,
			Online:            true,
			Bitrate:           server.Bitrate,
			HWMTU:             server.HWMTU,
			AutoconfigureMTU:  !server.FixedMTU,
			FixedMTU:          server.FixedMTU,
			IFACSize:          server.IFACSize,
			IFACNetnameVal:    server.IFACNetnameVal,
			IFACNetkeyVal:     server.IFACNetkeyVal,
			IFACKey:           append([]byte(nil), server.IFACKey...),
			IFACIdentity:      server.IFACIdentity,
			IFACSignature:     append([]byte(nil), server.IFACSignature...),
		}
		parent.SetTCPServer(server)
		parent.SetClientCountFunc(server.Clients)

		server.OnNewClient = func(ci *ifaces.TCPClientInterface) {
			if ci == nil {
				return
			}
			peer := &Interface{
				Name:              ci.String(),
				Type:              "TCPClientInterface",
				Parent:            parent,
				IN:                true,
				OUT:               true,
				DriverImplemented: true,
				Online:            true,
				Bitrate:           parent.Bitrate,
				HWMTU:             parent.HWMTU,
				AutoconfigureMTU:  parent.AutoconfigureMTU,
				FixedMTU:          parent.FixedMTU,
				IFACSize:          parent.IFACSize,
				IFACNetnameVal:    parent.IFACNetnameVal,
				IFACNetkeyVal:     parent.IFACNetkeyVal,
				IFACKey:           append([]byte(nil), parent.IFACKey...),
				IFACIdentity:      parent.IFACIdentity,
				IFACSignature:     append([]byte(nil), parent.IFACSignature...),
			}
			peer.SetTCPClient(ci)
			ci.Owner = tcpOwnerAdapter{ifc: peer}
			AddInterface(peer)
		}

		if err := server.Start(); err != nil {
			return false, fmt.Errorf("Interface %q TCP server start error: %w", name, err)
		}

		inheritInterfaceConfig(parent, ifc)
		ifc = parent
	}

	r.AddInterface(ifc, mode, bitrate, ifacSize, ifacNetname, ifacNetkey, announceCap, announceRateTarget, announceRateGrace, announceRatePenalty)
	return true, nil
}

func buildInterfaceFromType(name, ifType string) (*Interface, error) {
	ifType = strings.TrimSpace(ifType)
	if ifType == "" {
		return nil, errors.New("missing interface type")
	}

	// Keep type names for Python config parity.
	switch strings.ToLower(ifType) {
	case "udpinterface":
		return &Interface{
			Name:              name,
			Type:              "UDPInterface",
			DriverImplemented: true,
		}, nil
	case "tcpclientinterface":
		return &Interface{
			Name:              name,
			Type:              "TCPClientInterface",
			DriverImplemented: true,
		}, nil
	case "tcpserverinterface":
		return &Interface{
			Name:              name,
			Type:              "TCPServerInterface",
			DriverImplemented: true,
		}, nil
	case "serialinterface":
		return &Interface{
			Name:              name,
			Type:              "SerialInterface",
			DriverImplemented: true,
		}, nil
	case "ax25kissinterface":
		return &Interface{
			Name:              name,
			Type:              "AX25KISSInterface",
			DriverImplemented: true,
		}, nil
	case "backboneinterface":
		return &Interface{
			Name:              name,
			Type:              "BackboneInterface",
			DriverImplemented: true,
		}, nil
	case "backboneclientinterface":
		return &Interface{
			Name:              name,
			Type:              "BackboneClientInterface",
			DriverImplemented: true,
		}, nil
	case "kissinterface":
		return &Interface{
			Name:              name,
			Type:              "KISSInterface",
			DriverImplemented: true,
		}, nil
	case "i2pinterface":
		return &Interface{
			Name:              name,
			Type:              "I2PInterface",
			DriverImplemented: true,
		}, nil
	case "autointerface":
		return &Interface{
			Name:              name,
			Type:              "AutoInterface",
			DriverImplemented: true,
		}, nil
	case "pipeinterface":
		return &Interface{
			Name:              name,
			Type:              "PipeInterface",
			DriverImplemented: true,
		}, nil
	case "rnodemultiinterface":
		return &Interface{
			Name:              name,
			Type:              "RNodeMultiInterface",
			DriverImplemented: true,
		}, nil
	case "rnodeinterface":
		return &Interface{
			Name:              name,
			Type:              "RNodeInterface",
			DriverImplemented: true,
		}, nil
	case "weaveinterface":
		return &Interface{
			Name:              name,
			Type:              "WeaveInterface",
			DriverImplemented: true,
		}, nil
	default:
		return &Interface{
			Name:              name,
			Type:              ifType,
			DriverImplemented: false,
			DriverError:       "unknown interface type",
		}, errors.New("unknown interface type")
	}
}

func inheritInterfaceConfig(dst, src *Interface) {
	if dst == nil || src == nil {
		return
	}
	dst.IN = src.IN
	dst.OUT = src.OUT
	dst.IngressControl = src.IngressControl
	dst.ICMaxHeldAnnounces = src.ICMaxHeldAnnounces
	dst.ICBurstHold = src.ICBurstHold
	dst.ICBurstFreqNew = src.ICBurstFreqNew
	dst.ICBurstFreq = src.ICBurstFreq
	dst.ICNewTime = src.ICNewTime
	dst.ICBurstPenalty = src.ICBurstPenalty
	dst.ICHeldReleaseInterval = src.ICHeldReleaseInterval
	dst.AnnounceCap = src.AnnounceCap
	dst.SetAnnounceRateConfig(src.AnnounceRateTarget, src.AnnounceRateGrace, src.AnnounceRatePenalty)
}

func parseINIFallbackInterfaces(cfg *configobj.Config) map[string]map[string]string {
	if cfg == nil {
		return nil
	}
	out := map[string]map[string]string{}
	for _, name := range cfg.Sections() {
		if !strings.HasPrefix(name, "interfaces.") {
			continue
		}
		sec := cfg.Section(name)
		if sec == nil {
			continue
		}
		m := map[string]string{}
		for _, k := range []string{"enabled", "mode", "interface_mode", "bitrate", "announce_cap", "ifac_size", "networkname", "network_name", "passphrase", "pass_phrase", "type"} {
			if v, ok := sec.Get(k); ok {
				m[k] = v
			}
		}
		out[strings.TrimPrefix(name, "interfaces.")] = m
	}
	return out
}

func parseConfigObjInterfaces(path string) (map[string]map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")

	inInterfaces := false
	currentName := ""
	currentSub := ""
	out := map[string]map[string]string{}

	flush := func() {
		currentName = strings.TrimSpace(currentName)
		if currentName == "" {
			return
		}
		if _, ok := out[currentName]; !ok {
			out[currentName] = map[string]string{}
		}
	}

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Section header: [interfaces]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") && !strings.HasPrefix(line, "[[") {
			section := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			inInterfaces = strings.EqualFold(section, "interfaces")
			currentName = ""
			currentSub = ""
			continue
		}

		if !inInterfaces {
			continue
		}

		// Subsection header: [[Name]]
		if strings.HasPrefix(line, "[[") && strings.HasSuffix(line, "]]") {
			// Sub-subsection header: [[[Name]]]
			if strings.HasPrefix(line, "[[[") && strings.HasSuffix(line, "]]]") {
				sub := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "[[["), "]]]"))
				currentSub = sub
				flush()
				continue
			}
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "[["), "]]"))
			currentName = name
			currentSub = ""
			flush()
			continue
		}

		if currentName == "" {
			// Ignore keys at the [interfaces] level for now.
			continue
		}

		key, val, ok := splitKeyValue(line)
		if !ok {
			continue
		}
		if _, ok := out[currentName]; !ok {
			out[currentName] = map[string]string{}
		}
		lkey := strings.ToLower(key)
		if currentSub != "" {
			lkey = "sub." + strings.ToLower(currentSub) + "." + lkey
		}
		out[currentName][lkey] = val
	}

	return out, nil
}

type interfaceConfigEntry struct {
	Name string
	KV   map[string]string
}

func parseConfigObjInterfacesOrdered(path string) ([]interfaceConfigEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")

	inInterfaces := false
	currentSub := ""
	currentIdx := -1
	out := make([]interfaceConfigEntry, 0)

	startInterface := func(name string) {
		name = strings.TrimSpace(name)
		currentSub = ""
		if name == "" {
			currentIdx = -1
			return
		}
		out = append(out, interfaceConfigEntry{Name: name, KV: map[string]string{}})
		currentIdx = len(out) - 1
	}

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Section header: [interfaces]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") && !strings.HasPrefix(line, "[[") {
			section := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			inInterfaces = strings.EqualFold(section, "interfaces")
			currentSub = ""
			currentIdx = -1
			continue
		}

		if !inInterfaces {
			continue
		}

		// Subsection header: [[Name]]
		if strings.HasPrefix(line, "[[") && strings.HasSuffix(line, "]]") {
			// Sub-subsection header: [[[Name]]]
			if strings.HasPrefix(line, "[[[") && strings.HasSuffix(line, "]]]") {
				sub := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "[[["), "]]]"))
				currentSub = sub
				continue
			}
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "[["), "]]"))
			startInterface(name)
			continue
		}

		if currentIdx < 0 || currentIdx >= len(out) {
			// Ignore keys at the [interfaces] level.
			continue
		}

		key, val, ok := splitKeyValue(line)
		if !ok {
			continue
		}
		lkey := strings.ToLower(key)
		if currentSub != "" {
			lkey = "sub." + strings.ToLower(currentSub) + "." + lkey
		}
		out[currentIdx].KV[lkey] = val
	}

	return out, nil
}

func findInterfaceByName(name string) *Interface {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	for _, ifc := range Interfaces {
		if ifc == nil {
			continue
		}
		if ifc.Name == name || ifc.String() == name {
			return ifc
		}
	}
	return nil
}

func (r *Reticulum) getInterfaceConfigByName(name string) (map[string]string, bool) {
	name = strings.TrimSpace(name)
	if name == "" || r == nil {
		return nil, false
	}

	if ordered, err := parseConfigObjInterfacesOrdered(r.ConfigPath); err == nil && len(ordered) > 0 {
		for _, entry := range ordered {
			if strings.EqualFold(strings.TrimSpace(entry.Name), name) {
				return entry.KV, true
			}
		}
		return nil, false
	}

	ini := parseINIFallbackInterfaces(r.Config)
	for k, v := range ini {
		if strings.EqualFold(strings.TrimSpace(k), name) {
			return v, true
		}
	}
	return nil, false
}

func splitKeyValue(line string) (key, value string, ok bool) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	key = strings.TrimSpace(parts[0])
	value = strings.TrimSpace(parts[1])
	if key == "" {
		return "", "", false
	}
	// Strip surrounding quotes.
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
	}
	return key, value, true
}

func getFirst(m map[string]string, keys ...string) string {
	for _, k := range keys {
		if m == nil {
			return ""
		}
		if v, ok := m[strings.ToLower(k)]; ok {
			return v
		}
		// allow raw keys if caller already lowercased
		if v, ok := m[k]; ok {
			return v
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

func parseInterfaceMode(v string) int {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "access_point", "accesspoint", "ap":
		return InterfaceModeAccessPoint
	case "pointtopoint", "ptp":
		return InterfaceModePointToPoint
	case "roaming":
		return InterfaceModeRoaming
	case "boundary":
		return InterfaceModeBoundary
	case "gateway", "gw":
		return InterfaceModeGateway
	default:
		return InterfaceModeFull
	}
}

func parseInt(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	iv, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return iv, true
}

func parseFloat(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	fv, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return fv, true
}

// ---------------- interface setup and IFAC ----------------

func (r *Reticulum) AddInterface(
	ifc *Interface,
	mode int,
	configuredBitrate *int,
	ifacSize *int,
	ifacNetname *string,
	ifacNetkey *string,
	announceCap *float64,
	announceRateTarget *int,
	announceRateGrace *int,
	announceRatePenalty *int,
) {
	if r.IsConnectedToSharedInstance {
		return
	}
	if ifc == nil {
		return
	}

	if mode == 0 {
		mode = InterfaceModeFull
	}
	ifc.Mode = mode

	if configuredBitrate != nil {
		ifc.Bitrate = *configuredBitrate
	}
	ifc.OptimiseMTU()

	if ifacSize != nil {
		ifc.IFACSize = *ifacSize
	} else {
		// Python parity: keep the interface-type default IFAC size when unset,
		// falling back to 8 only if the interface has no default.
		if ifc.IFACSize == 0 {
			ifc.IFACSize = 8
		}
	}

	ac := float64(ANNOUNCE_CAP) / 100.0
	if announceCap != nil {
		ac = *announceCap
	}
	ifc.AnnounceCap = ac
	ifc.SetAnnounceRateConfig(announceRateTarget, announceRateGrace, announceRatePenalty)
	ifc.SetAnnounceRate(announceRateTarget, announceRateGrace, announceRatePenalty)

	if ifacNetname != nil {
		ifc.IFACNetnameVal = *ifacNetname
	}
	if ifacNetkey != nil {
		ifc.IFACNetkeyVal = *ifacNetkey
	}

	if ifc.IFACNetname() != "" || ifc.IFACNetkey() != "" {
		origin := make([]byte, 0)
		if n := ifc.IFACNetname(); n != "" {
			origin = append(origin, FullHash([]byte(n))...)
		}
		if k := ifc.IFACNetkey(); k != "" {
			origin = append(origin, FullHash([]byte(k))...)
		}
		originHash := FullHash(origin)
		ifacKey, err := cryptography.HKDF(64, originHash, r.ifacSalt, nil)
		if err != nil {
			Logf(LogError, "Could not derive IFAC key: %v", err)
			return
		}
		ifacIdentity, _ := IdentityFromBytes(ifacKey)
		ifacSignature, _ := ifacIdentity.Sign(FullHash(ifacKey))

		ifc.IFACKey = ifacKey
		ifc.IFACIdentity = ifacIdentity
		ifc.IFACSignature = ifacSignature
	}

	AddInterface(ifc)
	ifc.FinalInit()
}

// ---------------- persist / clean ----------------

func (r *Reticulum) ShouldPersistData() {
	if time.Since(r.lastDataPersist) > GRACIOUS_PERSIST_INTERVAL*time.Second {
		r.persistData()
	}
}

func (r *Reticulum) persistData() {
	IdentityPersistData()
	TransportPersistData()
	r.lastDataPersist = time.Now()
}

func (r *Reticulum) cleanCaches() {
	Log("Cleaning resource and packet caches...", LogExtreme)
	now := time.Now()

	// resources
	entries, _ := os.ReadDir(r.ResourcePath)
	for _, e := range entries {
		name := e.Name()
		if len(name) == (IdentityHashLength/8)*2 {
			fp := filepath.Join(r.ResourcePath, name)
			st, err := os.Stat(fp)
			if err != nil {
				Log("Error while cleaning resources cache, the contained exception was: "+err.Error(), LogError)
				continue
			}
			age := now.Sub(st.ModTime())
			if age > RESOURCE_CACHE*time.Second {
				if err := os.Remove(fp); err != nil {
					Log("Error while cleaning resources cache, the contained exception was: "+err.Error(), LogError)
				}
			}
		}
	}

	// packets
	entries, _ = os.ReadDir(r.CachePath)
	for _, e := range entries {
		name := e.Name()
		if len(name) == (IdentityHashLength/8)*2 {
			fp := filepath.Join(r.CachePath, name)
			st, err := os.Stat(fp)
			if err != nil {
				Log("Error while cleaning packet cache, the contained exception was: "+err.Error(), LogError)
				continue
			}
			age := now.Sub(st.ModTime())
			if age > DestinationTimeout {
				if err := os.Remove(fp); err != nil {
					Log("Error while cleaning packet cache, the contained exception was: "+err.Error(), LogError)
				}
			}
		}
	}
}

// ---------------- RPC loop + public methods ----------------

// Local RPC uses a small auth handshake plus Go `encoding/gob` for call/response
// serialization. The `RPCListener`/`RPCConn` interfaces are kept so we can swap
// the wire format if needed, but the current implementation targets Go↔Go.

func (r *Reticulum) rpcLoop() {
	if r == nil || r.rpcLn == nil {
		return
	}
	for {
		conn, err := r.rpcLn.Accept()
		if err != nil {
			// During shutdown the listener will be closed; don't spin.
			if strings.Contains(strings.ToLower(err.Error()), "closed") {
				return
			}
			Log("Error accepting RPC: "+err.Error(), LogError)
			continue
		}
		go r.handleRPC(conn)
	}
}

func (r *Reticulum) handleRPC(conn RPCConn) {
	defer conn.Close()

	var call map[string]any
	if err := conn.Recv(&call); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			// Normal case: client connects and closes without sending anything.
			return
		}
		Log("RPC recv error: "+err.Error(), LogError)
		return
	}

	if get, ok := call["get"].(string); ok {
		switch get {
		case "interface_stats":
			_ = conn.Send(r.GetInterfaceStats())
		case "path_table":
			mh, _ := call["max_hops"].(int)
			_ = conn.Send(r.GetPathTable(mh))
		case "rate_table":
			_ = conn.Send(r.GetRateTable())
		case "next_hop_if_name":
			dst := rpcBytes(call["destination_hash"])
			_ = conn.Send(r.GetNextHopIfName(dst))
		case "next_hop":
			dst := rpcBytes(call["destination_hash"])
			_ = conn.Send(r.GetNextHop(dst))
		case "first_hop_timeout":
			dst := rpcBytes(call["destination_hash"])
			_ = conn.Send(r.GetFirstHopTimeout(dst))
		case "link_count":
			_ = conn.Send(r.GetLinkCount())
		case "packet_rssi":
			hash := rpcBytes(call["packet_hash"])
			rpcSendFloat(conn, Transport.GetPacketRSSI(hash))
		case "packet_snr":
			hash := rpcBytes(call["packet_hash"])
			rpcSendFloat(conn, Transport.GetPacketSNR(hash))
		case "packet_q":
			hash := rpcBytes(call["packet_hash"])
			rpcSendFloat(conn, Transport.GetPacketQ(hash))
		}
	}
	if action, ok := call["call"].(string); ok {
		name, _ := call["interface"].(string)
		if name == "" {
			name, _ = call["name"].(string)
		}
		switch action {
		case "halt_interface":
			rpcSendError(conn, r.HaltInterface(name))
		case "resume_interface":
			rpcSendError(conn, r.ResumeInterface(name))
		case "reload_interface":
			rpcSendError(conn, r.ReloadInterface(name))
		}
	}
	if drop, ok := call["drop"].(string); ok {
		switch drop {
		case "path":
			dst := rpcBytes(call["destination_hash"])
			_ = conn.Send(r.DropPath(dst))
		case "all_via":
			dst := rpcBytes(call["destination_hash"])
			_ = conn.Send(r.DropAllVia(dst))
		case "announce_queues":
			_ = conn.Send(r.DropAnnounceQueues())
		}
	}
}

func rpcBytes(value any) []byte {
	switch v := value.(type) {
	case []byte:
		return v
	default:
		return nil
	}
}

func rpcSendFloat(conn RPCConn, value *float64) {
	if value == nil {
		_ = conn.Send(nil)
		return
	}
	_ = conn.Send(*value)
}

func rpcSendError(conn RPCConn, err error) {
	if err == nil {
		_ = conn.Send(nil)
		return
	}
	_ = conn.Send(err.Error())
}

func (r *Reticulum) getRPCClient() (RPCConn, error) {
	if r.rpcAddr == "" || r.rpcNetwork == "" {
		return nil, errors.New("rpc endpoint not configured")
	}
	return dialRPC(r.rpcNetwork, r.rpcAddr, r.RPCKey)
}

func (r *Reticulum) rpcCallInterfaceMgmt(action, name string) error {
	client, err := r.getRPCClient()
	if err != nil {
		return err
	}
	defer client.Close()
	if err := client.Send(map[string]any{"call": action, "interface": name}); err != nil {
		return err
	}
	var resp any
	if err := client.Recv(&resp); err != nil {
		return err
	}
	if resp == nil {
		return nil
	}
	if s, ok := resp.(string); ok && strings.TrimSpace(s) != "" {
		return errors.New(s)
	}
	return nil
}

// The methods below mirror Python behaviour closely (signatures + core logic).
// Internally they either call via RPC or call transport/*/identity directly.

func (r *Reticulum) GetInterfaceStats() map[string]any {
	if r.IsConnectedToSharedInstance {
		client, err := r.getRPCClient()
		if err != nil {
			Log("Could not contact shared instance for interface stats: "+err.Error(), LogError)
			return map[string]any{}
		}
		defer client.Close()
		if err := client.Send(map[string]any{"get": "interface_stats"}); err != nil {
			Log("RPC request failed: "+err.Error(), LogError)
			return map[string]any{}
		}
		var resp map[string]any
		if err := client.Recv(&resp); err != nil {
			Log("RPC response failed: "+err.Error(), LogError)
			return map[string]any{}
		}
		return resp
	}
	stats := map[string]any{}
	entriesShared := make([]map[string]any, 0, 1)
	entriesLocal := make([]map[string]any, 0, 1)
	entriesOther := make([]map[string]any, 0, len(Interfaces))

	for _, ifc := range Interfaces {
		if ifc == nil {
			continue
		}

		var parentName any = nil
		var parentHash any = nil
		if ifc.Parent != nil {
			parentName = ifc.Parent.String()
			if h := ifc.Parent.GetHash(); len(h) > 0 {
				parentHash = h
			}
		}

		name := ifc.String()
		ifHash := any(nil)
		if h := ifc.GetHash(); len(h) > 0 {
			ifHash = h
		}

		entry := map[string]any{
			"bitrate":                     ifc.Bitrate,
			"rxs":                         ifc.CurRxSpeed,
			"txs":                         ifc.CurTxSpeed,
			"ifac_signature":              ifc.IFACSignature,
			"ifac_size":                   ifc.IFACSize,
			"ifac_netname":                ifc.IFACNetnameVal,
			"name":                        name,
			"short_name":                  ifc.Name,
			"hash":                        ifHash,
			"type":                        ifc.Type,
			"rxb":                         ifc.RXB,
			"txb":                         ifc.TXB,
			"incoming_announce_frequency": ifc.IncomingAnnounceFrequency(),
			"outgoing_announce_frequency": ifc.OutgoingAnnounceFrequency(),
			"held_announces":              ifc.HeldAnnouncesCount(),
			"status":                      ifc.Online,
			"mode":                        ifc.Mode,
		}
		if cc := ifc.ClientCount(); cc != nil {
			entry["clients"] = *cc
		}
		if parentName != nil {
			entry["parent_interface_name"] = parentName
		}
		if parentHash != nil {
			entry["parent_interface_hash"] = parentHash
		}
		if pc := ifc.AutoPeerCount(); pc != nil {
			entry["peers"] = *pc
		}
		if pc := ifc.WeavePeerCount(); pc != nil {
			entry["peers"] = *pc
		}
		if sid := ifc.WeaveSwitchID(); sid != nil {
			entry["switch_id"] = *sid
		}
		if eid := ifc.WeaveEndpointID(); eid != nil {
			entry["endpoint_id"] = *eid
		}
		if v := ifc.WeaveCPULoad(); v != nil {
			entry["cpu_load"] = *v
		}
		if v := ifc.WeaveMemLoad(); v != nil {
			entry["mem_load"] = *v
		}
		if tasks := ifc.WeaveActiveTasks(); tasks != nil {
			entry["active_tasks"] = tasks
		}
		if via := ifc.WeaveViaSwitchID(); via != nil {
			entry["via_switch_id"] = *via
		}
		if v := ifc.I2PConnectable(); v != nil {
			entry["i2p_connectable"] = *v
		}
		if v := ifc.I2PB32(); v != nil {
			entry["i2p_b32"] = *v
		}
		if v := ifc.I2PTunnelState(); v != nil {
			entry["tunnelstate"] = *v
		}
		if v := ifc.RNodeAirtimeShort(); v != nil {
			entry["airtime_short"] = *v
		}
		if v := ifc.RNodeAirtimeLong(); v != nil {
			entry["airtime_long"] = *v
		}
		if v := ifc.RNodeChannelLoadShort(); v != nil {
			entry["channel_load_short"] = *v
		}
		if v := ifc.RNodeChannelLoadLong(); v != nil {
			entry["channel_load_long"] = *v
		}
		if v := ifc.RNodeNoiseFloor(); v != nil {
			entry["noise_floor"] = *v
		}
		if v := ifc.RNodeInterference(); v != nil {
			entry["interference"] = *v
		}
		if ts, dbm := ifc.RNodeInterferenceLast(); ts != nil && dbm != nil {
			entry["interference_last_ts"] = *ts
			entry["interference_last_dbm"] = *dbm
		}
		if v := ifc.RNodeCPUTemp(); v != nil {
			entry["cpu_temp"] = *v
		}
		if v := ifc.RNodeBatteryState(); v != nil {
			entry["battery_state"] = *v
		}
		if v := ifc.RNodeBatteryPercent(); v != nil {
			entry["battery_percent"] = *v
		}
		entry["announce_queue"] = ifc.AnnounceQueueCount()

		switch {
		case strings.HasPrefix(name, "Shared Instance["):
			entriesShared = append(entriesShared, entry)
		case strings.HasPrefix(name, "LocalInterface["):
			entriesLocal = append(entriesLocal, entry)
		default:
			entriesOther = append(entriesOther, entry)
		}
	}

	stats["interfaces"] = append(append(entriesShared, entriesOther...), entriesLocal...)
	stats["rxb"] = TrafficRXB
	stats["txb"] = TrafficTXB
	stats["rxs"] = SpeedRX
	stats["txs"] = SpeedTX
	if TransportEnabled() && TransportIdentity != nil {
		stats["transport_id"] = TransportIdentity.Hash
		stats["transport_uptime"] = time.Since(StartTime).Seconds()
	} else {
		stats["transport_id"] = nil
		stats["transport_uptime"] = nil
	}
	if ProbeDestinationEnabled() && ProbeDestination != nil {
		stats["probe_responder"] = ProbeDestination.Hash()
	} else {
		stats["probe_responder"] = nil
	}

	stats["rss"] = processRSSBytes()
	return stats
}

func processRSSBytes() any {
	return processRSSBytesPlatform()
}

func (r *Reticulum) GetPathTable(maxHops int) []map[string]any {
	if r.IsConnectedToSharedInstance {
		client, err := r.getRPCClient()
		if err != nil {
			Log("Could not contact shared instance for path table: "+err.Error(), LogError)
			return nil
		}
		defer client.Close()
		if err := client.Send(map[string]any{"get": "path_table", "max_hops": maxHops}); err != nil {
			Log("RPC request for path table failed: "+err.Error(), LogError)
			return nil
		}
		var resp []map[string]any
		if err := client.Recv(&resp); err != nil {
			Log("RPC response for path table failed: "+err.Error(), LogError)
			return nil
		}
		return resp
	}
	return GetPathTable(maxHops)
}

func (r *Reticulum) GetRateTable() []map[string]any {
	if r.IsConnectedToSharedInstance {
		client, err := r.getRPCClient()
		if err != nil {
			Log("Could not contact shared instance for rate table: "+err.Error(), LogError)
			return nil
		}
		defer client.Close()
		if err := client.Send(map[string]any{"get": "rate_table"}); err != nil {
			Log("RPC request for rate table failed: "+err.Error(), LogError)
			return nil
		}
		var resp []map[string]any
		if err := client.Recv(&resp); err != nil {
			Log("RPC response for rate table failed: "+err.Error(), LogError)
			return nil
		}
		return resp
	}
	return GetRateTable()
}

func (r *Reticulum) DropPath(destination []byte) bool {
	if r.IsConnectedToSharedInstance {
		client, err := r.getRPCClient()
		if err != nil {
			Log("Could not contact shared instance to drop path: "+err.Error(), LogError)
			return false
		}
		defer client.Close()
		req := map[string]any{"drop": "path", "destination_hash": destination}
		if err := client.Send(req); err != nil {
			Log("RPC request to drop path failed: "+err.Error(), LogError)
			return false
		}
		var resp bool
		if err := client.Recv(&resp); err != nil {
			Log("RPC response to drop path failed: "+err.Error(), LogError)
			return false
		}
		return resp
	}
	return DropPath(destination)
}

func (r *Reticulum) DropAllVia(transportHash []byte) int {
	if r.IsConnectedToSharedInstance {
		client, err := r.getRPCClient()
		if err != nil {
			Log("Could not contact shared instance to drop via: "+err.Error(), LogError)
			return 0
		}
		defer client.Close()
		req := map[string]any{"drop": "all_via", "destination_hash": transportHash}
		if err := client.Send(req); err != nil {
			Log("RPC request to drop via failed: "+err.Error(), LogError)
			return 0
		}
		var resp int
		if err := client.Recv(&resp); err != nil {
			Log("RPC response to drop via failed: "+err.Error(), LogError)
			return 0
		}
		return resp
	}
	return DropAllVia(transportHash)
}

func (r *Reticulum) DropAnnounceQueues() int {
	if r.IsConnectedToSharedInstance {
		client, err := r.getRPCClient()
		if err != nil {
			Log("Could not contact shared instance to drop announce queues: "+err.Error(), LogError)
			return 0
		}
		defer client.Close()
		if err := client.Send(map[string]any{"drop": "announce_queues"}); err != nil {
			Log("RPC request to drop announce queues failed: "+err.Error(), LogError)
			return 0
		}
		var resp int
		if err := client.Recv(&resp); err != nil {
			Log("RPC response to drop announce queues failed: "+err.Error(), LogError)
			return 0
		}
		return resp
	}
	return DropAnnounceQueues()
}

func (r *Reticulum) GetNextHop(destination []byte) []byte {
	if r.IsConnectedToSharedInstance {
		client, err := r.getRPCClient()
		if err != nil {
			Log("Could not contact shared instance for next hop: "+err.Error(), LogError)
			return nil
		}
		defer client.Close()
		if err := client.Send(map[string]any{"get": "next_hop", "destination_hash": destination}); err != nil {
			Log("RPC request for next hop failed: "+err.Error(), LogError)
			return nil
		}
		var resp []byte
		if err := client.Recv(&resp); err != nil {
			Log("RPC response for next hop failed: "+err.Error(), LogError)
			return nil
		}
		return resp
	}
	return NextHop(destination)
}

func (r *Reticulum) GetNextHopIfName(destination []byte) string {
	if r.IsConnectedToSharedInstance {
		client, err := r.getRPCClient()
		if err != nil {
			Log("Could not contact shared instance for next hop interface: "+err.Error(), LogError)
			return ""
		}
		defer client.Close()
		if err := client.Send(map[string]any{"get": "next_hop_if_name", "destination_hash": destination}); err != nil {
			Log("RPC request for next hop interface failed: "+err.Error(), LogError)
			return ""
		}
		var resp string
		if err := client.Recv(&resp); err != nil {
			Log("RPC response for next hop interface failed: "+err.Error(), LogError)
			return ""
		}
		return resp
	}
	return NextHopInterfaceName(destination)
}

func (r *Reticulum) rpcGetFloat(kind string, packetHash []byte) (*float64, bool) {
	client, err := r.getRPCClient()
	if err != nil {
		Log("Could not contact shared instance for "+kind+": "+err.Error(), LogError)
		return nil, false
	}
	defer client.Close()

	req := map[string]any{"get": kind}
	if len(packetHash) > 0 {
		req["packet_hash"] = packetHash
	}
	if err := client.Send(req); err != nil {
		Log("RPC request for "+kind+" failed: "+err.Error(), LogError)
		return nil, false
	}

	var raw any
	if err := client.Recv(&raw); err != nil {
		Log("RPC response for "+kind+" failed: "+err.Error(), LogError)
		return nil, false
	}

	return decodeFloatPointer(raw), true
}

func decodeFloatPointer(raw any) *float64 {
	switch v := raw.(type) {
	case nil:
		return nil
	case float64:
		val := v
		return &val
	case *float64:
		if v == nil {
			return nil
		}
		val := *v
		return &val
	default:
		return nil
	}
}

func (r *Reticulum) GetFirstHopTimeout(destination []byte) float64 {
	if r.IsConnectedToSharedInstance {
		client, err := r.getRPCClient()
		if err != nil {
			Log("Could not contact shared instance for first hop timeout: "+err.Error(), LogError)
			return DEFAULT_PER_HOP_TIMEOUT
		}
		defer client.Close()
		if err := client.Send(map[string]any{"get": "first_hop_timeout", "destination_hash": destination}); err != nil {
			Log("RPC request for first hop timeout failed: "+err.Error(), LogError)
			return DEFAULT_PER_HOP_TIMEOUT
		}
		var resp float64
		if err := client.Recv(&resp); err != nil {
			Log("RPC response for first hop timeout failed: "+err.Error(), LogError)
			return DEFAULT_PER_HOP_TIMEOUT
		}
		if bitrate := SharedInstanceForcedBitrate(); bitrate > 0 {
			simulatedLatency := (float64(MTU) * 8.0) / float64(bitrate)
			Logf(LogDebug, "Adding simulated latency of %s to first hop timeout", PrettyTime(simulatedLatency, false, false))
			resp += simulatedLatency
		}
		return resp
	}
	return Transport.GetFirstHopTimeout(destination).Seconds()
}

func (r *Reticulum) GetLinkCount() int {
	if r.IsConnectedToSharedInstance {
		client, err := r.getRPCClient()
		if err != nil {
			Log("Could not contact shared instance for link count: "+err.Error(), LogError)
			return 0
		}
		defer client.Close()
		if err := client.Send(map[string]any{"get": "link_count"}); err != nil {
			Log("RPC request for link count failed: "+err.Error(), LogError)
			return 0
		}
		var resp int
		if err := client.Recv(&resp); err != nil {
			Log("RPC response for link count failed: "+err.Error(), LogError)
			return 0
		}
		return resp
	}
	return len(TransportActiveLinks())
}

func (r *Reticulum) GetPacketRSSI(packetHash []byte) *float64 {
	if r.IsConnectedToSharedInstance {
		raw, ok := r.rpcGetFloat("packet_rssi", packetHash)
		if !ok {
			return nil
		}
		return raw
	}
	return Transport.GetPacketRSSI(packetHash)
}

func (r *Reticulum) GetPacketSNR(packetHash []byte) *float64 {
	if r.IsConnectedToSharedInstance {
		raw, ok := r.rpcGetFloat("packet_snr", packetHash)
		if !ok {
			return nil
		}
		return raw
	}
	return Transport.GetPacketSNR(packetHash)
}

func (r *Reticulum) GetPacketQ(packetHash []byte) *float64 {
	if r.IsConnectedToSharedInstance {
		raw, ok := r.rpcGetFloat("packet_q", packetHash)
		if !ok {
			return nil
		}
		return raw
	}
	return Transport.GetPacketQ(packetHash)
}

// Other methods (GetPathTable, GetRateTable, DropPath, DropAllVia,
// DropAnnounceQueues, GetNextHopIfName, GetFirstHopTimeout,
// GetNextHop, GetLinkCount, GetPacketRSSI/SNR/Q) follow the same pattern.

// ---------------- statics (like Python) ----------------

func ShouldUseImplicitProof() bool {
	return useImplicitProof
}

func TransportEnabled() bool {
	return transportEnabled
}

func LinkMTUDiscovery() bool {
	return linkMTUDiscovery
}

func RemoteManagementEnabled() bool {
	return remoteManagementEnabled
}

func ProbeDestinationEnabled() bool {
	return allowProbes
}

// defaultConfigLines is a direct copy of __default_rns_config__.
var defaultConfigLines = []string{
	"# This is the default Reticulum config file.",
	"# You should probably edit it to include any additional,",
	"# interfaces and settings you might need.",
	"",
	"# Only the most basic options are included in this default",
	"# configuration. To see a more verbose, and much longer,",
	"# configuration example, you can run the command:",
	"# rnsd --exampleconfig",
	"",
	"",
	"[reticulum]",
	"",
	"# If you enable Transport, your system will route traffic",
	"# for other peers, pass announces and serve path requests.",
	"# This should only be done for systems that are suited to",
	"# act as transport nodes, ie. if they are stationary and",
	"# always-on. This directive is optional and can be removed",
	"# for brevity.",
	"",
	"enable_transport = False",
	"",
	"",
	"# By default, the first program to launch the Reticulum",
	"# Network Stack will create a shared instance, that other",
	"# programs can communicate with. Only the shared instance",
	"# opens all the configured interfaces directly, and other",
	"# local programs communicate with the shared instance over",
	"# a local socket. This is completely transparent to the",
	"# user, and should generally be turned on. This directive",
	"# is optional and can be removed for brevity.",
	"",
	"share_instance = Yes",
	"",
	"",
	"# If you want to run multiple *different* shared instances",
	"# on the same system, you will need to specify different",
	"# instance names for each. On platforms supporting domain",
	"# sockets, this can be done with the instance_name option:",
	"",
	"instance_name = default",
	"",
	"# Some platforms don't support domain sockets, and if that",
	"# is the case, you can isolate different instances by",
	"# specifying a unique set of ports for each:",
	"",
	"# shared_instance_port = 37428",
	"# instance_control_port = 37429",
	"",
	"",
	"# If you want to explicitly use TCP for shared instance",
	"# communication, instead of domain sockets, this is also",
	"# possible, by using the following option:",
	"",
	"# shared_instance_type = tcp",
	"",
	"",
	"# You can configure Reticulum to panic and forcibly close",
	"# if an unrecoverable interface error occurs, such as the",
	"# hardware device for an interface disappearing. This is",
	"# an optional directive, and can be left out for brevity.",
	"# This behaviour is disabled by default.",
	"",
	"# panic_on_interface_error = No",
	"",
	"",
	"[logging]",
	"# Valid log levels are 0 through 7:",
	"#   0: Log only critical information",
	"#   1: Log errors and lower log levels",
	"#   2: Log warnings and lower log levels",
	"#   3: Log notices and lower log levels",
	"#   4: Log info and lower (this is the default)",
	"#   5: Verbose logging",
	"#   6: Debug logging",
	"#   7: Extreme logging",
	"",
	"loglevel = 4",
	"",
	"",
	"# The interfaces section defines the physical and virtual",
	"# interfaces Reticulum will use to communicate on. This",
	"# section will contain examples for a variety of interface",
	"# types. You can modify these or use them as a basis for",
	"# your own config, or simply remove the unused ones.",
	"",
	"[interfaces]",
	"",
	"  # This interface enables communication with other",
	"  # link-local Reticulum nodes over UDP. It does not",
	"  # need any functional IP infrastructure like routers",
	"  # or DHCP servers, but will require that at least link-",
	"  # local IPv6 is enabled in your operating system, which",
	"  # should be enabled by default in almost any OS. See",
	"  # the Reticulum Manual for more configuration options.",
	"",
	"  [[Default Interface]]",
	"    type = AutoInterface",
	"    enabled = Yes",
}
