package rns

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
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
	IFAC_SALT = func() []byte {
		b, err := hex.DecodeString("adf54d882c9a9b80771eb4995d702d4a3e733391b2a0f53f416d9f907e55cff8")
		if err != nil {
			panic(err)
		}
		return b
	}()

	MTU = DefaultMTU
	MDU = MTU - HEADER_MAXSIZE - IFAC_MIN_SIZE

	instance *Reticulum

	// Global flags, like in the Python class.
	transportEnabled                = false
	linkMTUDiscovery                = LINK_MTU_DISCOVERY
	remoteManagementEnabled         = false
	useImplicitProof                = true
	allowProbes                     = false
	discoveryEnabled                = false
	discoverInterfacesMode          = false
	requiredDiscoveryValue          *int
	interfaceDiscoverySources       [][]byte
	autoconnectDiscoveredInterfaces int
	publishBlackholeEnabled         bool
	blackholeSources                [][]byte
)

func init() {
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

	PanicOnInterfaceError bool

	jobsStarted     bool
	lastDataPersist time.Time
	lastCacheClean  time.Time

	ifacSalt []byte

	BootstrapConfigs []map[string]string

	// RPC listener + address (TCP/Unix).
	rpcAddr    string
	rpcNetwork string // "tcp" / "unix"
	rpcLn      net.Listener
}

// GetInstance mirrors Python's Reticulum.get_instance().
func GetInstance() *Reticulum {
	return instance
}

// Python parity: these are no-ops in Reticulum.py.
func (r *Reticulum) HaltInterface(name string) error {
	return nil
}

func (r *Reticulum) ResumeInterface(name string) error {
	return nil
}

func (r *Reticulum) ReloadInterface(name string) error {
	return nil
}

func (r *Reticulum) DiscoveredInterfaces() []map[string]any {
	if r == nil {
		return nil
	}
	prevOwner := Owner
	Owner = r
	defer func() {
		Owner = prevOwner
	}()
	requiredDiscoveryValue := DefaultDiscoveryRequiredValue
	if v := RequiredDiscoveryValue(); v != nil && *v > 0 {
		requiredDiscoveryValue = *v
	}
	discovery, err := NewInterfaceDiscovery(requiredDiscoveryValue, nil, false)
	if err != nil {
		return nil
	}
	return discovery.ListDiscoveredInterfaces(false, false)
}

func (r *Reticulum) GetBlackholedIdentities() map[hashKey]map[string]any {
	defer func() {
		if rec := recover(); rec != nil {
			Logf(LogError, "Error while getting blackholed identities: %v", rec)
			panic(rec)
		}
	}()
	if r == nil {
		return nil
	}
	if r.IsConnectedToSharedInstance {
		client, err := r.getRPCClient()
		if err != nil {
			panic(err)
		}
		defer client.Close()
		data, err := vendor.PickleDumps(map[string]any{"get": "blackholed_identities"})
		if err != nil {
			panic(err)
		}
		if err := func() error {
			var hdr [4]byte
			binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
			if _, err := client.Write(hdr[:]); err != nil {
				return err
			}
			if len(data) > 0 {
				_, err := client.Write(data)
				return err
			}
			return nil
		}(); err != nil {
			panic(err)
		}
		var resp map[string]map[string]any
		if err := func() error {
			data, err := func() ([]byte, error) {
				var hdr [4]byte
				if _, err := io.ReadFull(client, hdr[:]); err != nil {
					return nil, err
				}
				n := int(binary.BigEndian.Uint32(hdr[:]))
				buf := make([]byte, n)
				if _, err := io.ReadFull(client, buf); err != nil {
					return nil, err
				}
				return buf, nil
			}()
			if err != nil {
				return err
			}
			result, err := vendor.PickleLoads(data)
			if err != nil {
				return fmt.Errorf("pickle decode: %w", err)
			}
			return vendor.PickleAssign(result, &resp)
		}(); err != nil {
			panic(err)
		}
		out := make(map[hashKey]map[string]any, len(resp))
		for k, entry := range resp {
			rawKey, err := hex.DecodeString(k)
			if err != nil || len(rawKey) < truncatedHashBytes {
				continue
			}
			var key hashKey
			copy(key[:], rawKey[:truncatedHashBytes])
			out[key] = entry
		}
		return out
	}
	blackholeMu.RLock()
	defer blackholeMu.RUnlock()
	out := make(map[hashKey]map[string]any, len(blackholedIdentities))
	for key, entry := range blackholedIdentities {
		serialised := map[string]any{"source": nil, "until": nil, "reason": nil}
		if entry != nil {
			serialised["source"] = append([]byte(nil), entry.Source...)
			if entry.Until != nil && !entry.Until.IsZero() {
				serialised["until"] = float64(entry.Until.UnixNano()) / 1e9
			}
			if entry.Reason != nil {
				serialised["reason"] = *entry.Reason
			}
		}
		out[key] = serialised
	}
	return out
}

func (r *Reticulum) BlackholeIdentity(identityHash []byte, until *time.Time, reason *string) any {
	if len(identityHash) != ReticulumTruncatedHashLength/8 {
		return false
	}
	if r != nil && r.IsConnectedToSharedInstance {
		client, err := r.getRPCClient()
		if err != nil {
			panic(err)
		}
		defer client.Close()
		req := map[string]any{
			"blackhole_identity": identityHash,
			"until":              nil,
			"reason":             nil,
		}
		if until != nil && !until.IsZero() {
			req["until"] = float64(until.UnixNano()) / 1e9
		}
		if reason != nil {
			req["reason"] = *reason
		}
		data, err := vendor.PickleDumps(req)
		if err != nil {
			panic(err)
		}
		if err := func() error {
			var hdr [4]byte
			binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
			if _, err := client.Write(hdr[:]); err != nil {
				return err
			}
			if len(data) > 0 {
				_, err := client.Write(data)
				return err
			}
			return nil
		}(); err != nil {
			panic(err)
		}
		var resp any
		if err := func() error {
			data, err := func() ([]byte, error) {
				var hdr [4]byte
				if _, err := io.ReadFull(client, hdr[:]); err != nil {
					return nil, err
				}
				n := int(binary.BigEndian.Uint32(hdr[:]))
				buf := make([]byte, n)
				if _, err := io.ReadFull(client, buf); err != nil {
					return nil, err
				}
				return buf, nil
			}()
			if err != nil {
				return err
			}
			result, err := vendor.PickleLoads(data)
			if err != nil {
				return fmt.Errorf("pickle decode: %w", err)
			}
			return vendor.PickleAssign(result, &resp)
		}(); err != nil {
			panic(err)
		}
		return resp
	}
	return BlackholeIdentity(identityHash, until, reason)
}

func (r *Reticulum) UnblackholeIdentity(identityHash []byte) any {
	if len(identityHash) != ReticulumTruncatedHashLength/8 {
		return false
	}
	if r != nil && r.IsConnectedToSharedInstance {
		client, err := r.getRPCClient()
		if err != nil {
			panic(err)
		}
		defer client.Close()
		data, err := vendor.PickleDumps(map[string]any{"unblackhole_identity": identityHash})
		if err != nil {
			panic(err)
		}
		if err := func() error {
			var hdr [4]byte
			binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
			if _, err := client.Write(hdr[:]); err != nil {
				return err
			}
			if len(data) > 0 {
				_, err := client.Write(data)
				return err
			}
			return nil
		}(); err != nil {
			panic(err)
		}
		var resp any
		if err := func() error {
			data, err := func() ([]byte, error) {
				var hdr [4]byte
				if _, err := io.ReadFull(client, hdr[:]); err != nil {
					return nil, err
				}
				n := int(binary.BigEndian.Uint32(hdr[:]))
				buf := make([]byte, n)
				if _, err := io.ReadFull(client, buf); err != nil {
					return nil, err
				}
				return buf, nil
			}()
			if err != nil {
				return err
			}
			result, err := vendor.PickleLoads(data)
			if err != nil {
				return fmt.Errorf("pickle decode: %w", err)
			}
			return vendor.PickleAssign(result, &resp)
		}(); err != nil {
			panic(err)
		}
		return resp
	}
	return UnblackholeIdentity(identityHash)
}

// NewReticulum mirrors __init__.
func NewReticulum(configDir *string, loglevel *int, logdest any, verbosity *int,
	requireSharedInstance bool, sharedInstanceType *string) (ret *Reticulum, retErr error) {

	if instance != nil {
		return nil, errors.New("Reticulum is already running")
	}

	// Python parity: run platform checks early (mostly a no-op in Go).
	vendor.PlatformChecks()

	userDir, err := os.UserHomeDir()
	if err != nil {
		userDir = "~"
	}

	r := &Reticulum{
		UserDir:            userDir,
		localInterfacePort: 37428,
		localControlPort:   37429,
		ShareInstance:      true,
		SharedInstanceType: "",
		RequireShared:      requireSharedInstance,
		ifacSalt:           IFAC_SALT,
		lastDataPersist:    time.Now(),
		lastCacheClean:     time.Time{},
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
		if st, err := os.Stat("/etc/reticulum"); err == nil && st.IsDir() {
			if st, err := os.Stat("/etc/reticulum/config"); err == nil && !st.IsDir() {
				r.ConfigDir = "/etc/reticulum"
			}
		}
		if r.ConfigDir == "" {
			configDir := filepath.Join(r.UserDir, ".config/reticulum")
			if st, err := os.Stat(configDir); err == nil && st.IsDir() {
				if st, err := os.Stat(filepath.Join(configDir, "config")); err == nil && !st.IsDir() {
					r.ConfigDir = configDir
				}
			}
		}
		if r.ConfigDir == "" {
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
	_ = os.MkdirAll(r.StoragePath, 0o755)
	_ = os.MkdirAll(r.CachePath, 0o755)
	_ = os.MkdirAll(filepath.Join(r.CachePath, "announces"), 0o755)
	_ = os.MkdirAll(r.ResourcePath, 0o755)
	_ = os.MkdirAll(r.IdentityPath, 0o755)
	_ = os.MkdirAll(filepath.Join(r.StoragePath, "blackhole"), 0o755)
	_ = os.MkdirAll(r.InterfacePath, 0o755)

	// config
	if st, err := os.Stat(r.ConfigPath); err == nil && !st.IsDir() {
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
		Log("Default config file created. Make any necessary changes in "+r.ConfigDir+"/config and restart Reticulum if needed.", LogInfo)
		time.Sleep(1500 * time.Millisecond)
	}

	if sharedInstanceType != nil {
		r.SharedInstanceType = *sharedInstanceType
	}

	if err := r.applyConfig(); err != nil {
		return nil, err
	}

	// Python config application also decides the local shared-instance transport
	// before Transport.start() and before any RPC listener is created.
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

	if r.LocalSocketPath == "" && r.UseAFUnix {
		r.LocalSocketPath = "default"
	}

	if err := r.startLocalInterface(); err != nil {
		return nil, err
	}

	// Python parity: __apply_config() synthesizes configured interfaces before
	// Transport.start() is invoked.
	if (r.IsSharedInstance || r.IsStandaloneInstance) && !r.IsConnectedToSharedInstance {
		Log("Bringing up system interfaces...", LogVerbose)
		if err := r.synthesizeConfiguredInterfaces(); err != nil {
			return nil, err
		}
		Log("System interfaces are ready", LogVerbose)
	}

	Logf(LogDebug, "Utilising cryptography backend %q", cryptography.ProviderBackend())
	Log("Configuration loaded from "+r.ConfigPath, LogVerbose)

	_ = IdentityLoadKnownDestinations()
	Start(r) // mirrors RNS.Transport.start(self)

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
		// Even when connected as a client to a shared instance, Python loads
		// the transport identity from the shared storage directory to derive
		// the RPC key. We must do the same for cross-language compatibility.
		if TransportIdentity != nil {
			if pk := TransportIdentity.GetPrivateKey(); len(pk) > 0 {
				r.RPCKey = FullHash(pk)
			}
		}
		if r.RPCKey == nil {
			// Try loading transport identity from storage (same as Python client)
			tiPath := filepath.Join(r.StoragePath, "transport_identity")
			if st, err := os.Stat(tiPath); err == nil && !st.IsDir() {
				if idData, err := os.ReadFile(tiPath); err == nil && len(idData) > 0 {
					if ti, err := IdentityFromBytes(idData); err == nil && ti != nil {
						if pk := ti.GetPrivateKey(); len(pk) > 0 {
							r.RPCKey = FullHash(pk)
						}
					}
				}
			}
		}
		if r.RPCKey == nil {
			r.RPCKey = FullHash([]byte("reticulum"))
		}
	}

	// Python only creates the RPC listener for the shared-instance owner, after
	// Transport.start() has loaded the transport identity used for the auth key.
	if r.IsSharedInstance {
		switch r.rpcNetwork {
		case "unix":
			if len(r.rpcAddr) > 0 && r.rpcAddr[0] == 0 && (vendor.IsLinux() || vendor.IsAndroid()) {
				ln, err := net.Listen("unix", r.rpcAddr)
				if err != nil {
					return nil, err
				}
				r.rpcLn = ln
			} else {
				name := strings.Trim(r.rpcAddr, "\x00")
				if name == "" {
					name = "default"
				}
				replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_", ":", "_")
				name = replacer.Replace(name)
				if len(name) > 48 {
					sum := sha256.Sum256([]byte(name))
					name = hex.EncodeToString(sum[:16])
				}
				path := filepath.Join(os.TempDir(), "rns-"+name+".sock")
				if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
					Log("Could not remove stale RPC socket: "+err.Error(), LogDebug)
				}
				ln, err := net.Listen("unix", path)
				if err != nil {
					return nil, err
				}
				r.rpcLn = ln
				r.rpcAddr = path
			}
		default:
			ln, err := net.Listen(r.rpcNetwork, r.rpcAddr)
			if err != nil {
				return nil, err
			}
			r.rpcLn = ln
			r.rpcAddr = ln.Addr().String()
		}
		go r.rpcLoop()
	}

	if r.IsSharedInstance || r.IsStandaloneInstance {
		if InterfaceDiscoveryEnabled() {
			EnableDiscovery()
		}
		if discoverInterfacesMode {
			DiscoverInterfaces()
		}
		if len(BlackholeSources()) > 0 {
			EnableBlackholeUpdater()
		}
	}

	// signals and exit handler
	exitMu.Lock()
	exitCallback = exitHandler
	exitMu.Unlock()

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

	return r, nil
}

// broadcastForInterface returns an IPv4 broadcast address for the named interface.
// Used for UDPInterface convenience config, similar to Python behavior.
func broadcastForInterface(name string) (net.IP, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return nil, err
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP == nil || ipnet.Mask == nil {
			continue
		}
		ip4 := ipnet.IP.To4()
		if ip4 == nil {
			continue
		}
		bcast := make(net.IP, 4)
		for i := 0; i < 4; i++ {
			bcast[i] = ip4[i] | ^ipnet.Mask[i]
		}
		return bcast, nil
	}
	return nil, errors.New("no IPv4 address found on interface")
}

// ---------------- exit / signals ----------------

var (
	exitHandlerRan         bool
	interfaceDetachHandler bool
)

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
	ExitHandler()
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
		remaining := len(LocalClientInterfaces)
		if !reported {
			Logf(LogDebug, "Waiting for %d local client(s) to disconnect before exiting", remaining)
			reported = true
		}
		if timeout > 0 && time.Since(started) > timeout {
			Logf(LogWarning, "Timed out waiting for local clients to disconnect (%d remaining)", remaining)
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
		if path, ok := sec.Get("network_identity"); ok && strings.TrimSpace(path) != "" && !HasNetworkIdentity() {
			identityPath := filepath.Clean(os.ExpandEnv(path))
			if strings.HasPrefix(identityPath, "~") {
				identityPath = filepath.Join(r.UserDir, strings.TrimPrefix(identityPath, "~"))
			}
			_ = os.MkdirAll(filepath.Dir(identityPath), 0o755)
			var networkIdentity *Identity
			if st, err := os.Stat(identityPath); err == nil && !st.IsDir() {
				id, err := IdentityFromFile(identityPath)
				if err != nil {
					return fmt.Errorf("could not set network identity from %s: %w", path, err)
				}
				networkIdentity = id
			} else {
				id, err := NewIdentity()
				if err != nil {
					return fmt.Errorf("could not create network identity for %s: %w", path, err)
				}
				if err := id.Save(identityPath); err != nil {
					return fmt.Errorf("could not persist network identity to %s: %w", identityPath, err)
				}
				networkIdentity = id
			}
			SetNetworkIdentity(networkIdentity)
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
				remoteManagementAllowedMu.Lock()
				if _, ok := remoteManagementAllowed[string(b)]; !ok {
					remoteManagementAllowed[string(b)] = append([]byte(nil), b...)
				}
				remoteManagementAllowedMu.Unlock()
			}
		}
		if v, _ := sec.AsBool("respond_to_probes"); v {
			allowProbes = true
		}
		if v, err := sec.AsInt("force_shared_instance_bitrate"); err == nil {
			sharedInstanceMu.Lock()
			sharedInstanceForcedBitrate = v
			sharedInstanceMu.Unlock()
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
		if v, err := sec.AsBool("discover_interfaces"); err == nil {
			discoverInterfacesMode = v
		}
		if v, err := sec.AsInt("required_discovery_value"); err == nil && v > 0 {
			requiredDiscoveryValue = &v
			discoveryRequiredValue = v
		} else if _, ok := sec.Get("required_discovery_value"); ok {
			requiredDiscoveryValue = nil
		}
		if l := sec.AsList("interface_discovery_sources"); len(l) > 0 {
			interfaceDiscoverySources = interfaceDiscoverySources[:0]
			for _, hexhash := range l {
				destLen := (TRUNCATED_HASHLENGTH / 8) * 2
				if len(hexhash) != destLen {
					return fmt.Errorf("identity hash length for interface discovery source %s is invalid, must be %d hexadecimal characters (%d bytes)", hexhash, destLen, destLen/2)
				}
				b, err := hex.DecodeString(hexhash)
				if err != nil {
					return fmt.Errorf("invalid identity hash for interface discovery source: %s", hexhash)
				}
				interfaceDiscoverySources = append(interfaceDiscoverySources, b)
			}
		}
		if v, err := sec.AsInt("autoconnect_discovered_interfaces"); err == nil && v > 0 {
			autoconnectDiscoveredInterfaces = v
		}
		if v, err := sec.AsBool("publish_blackhole"); err == nil {
			publishBlackholeEnabled = v
		}
		if l := sec.AsList("blackhole_sources"); len(l) > 0 {
			blackholeSources = blackholeSources[:0]
			for _, hexhash := range l {
				destLen := (TRUNCATED_HASHLENGTH / 8) * 2
				if len(hexhash) != destLen {
					return fmt.Errorf("identity hash length for blackhole source %s is invalid, must be %d hexadecimal characters (%d bytes)", hexhash, destLen, destLen/2)
				}
				b, err := hex.DecodeString(hexhash)
				if err != nil {
					return fmt.Errorf("invalid identity hash for remote blackhole source: %s", hexhash)
				}
				blackholeSources = append(blackholeSources, b)
			}
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
	_ = os.MkdirAll(r.ConfigDir, 0o755)
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

	siName := "default"
	if r.UseAFUnix {
		if strings.TrimSpace(r.LocalSocketPath) != "" {
			siName = strings.TrimSpace(r.LocalSocketPath)
		}
	} else {
		siName = fmt.Sprintf("%d", r.localInterfacePort)
	}

	sharedIfc := &Interface{
		Name:              fmt.Sprintf("Shared Instance[%s]", siName),
		Type:              "LocalInterface",
		IN:                true,
		OUT:               true,
		DriverImplemented: true,
		Online:            true,
		AutoconfigureMTU:  true,
	}
	sharedIfc.Bitrate = 1000000000
	sharedInstanceMu.RLock()
	br := sharedInstanceForcedBitrate
	sharedInstanceMu.RUnlock()
	if br > 0 {
		sharedIfc.Bitrate = br
		Logf(LogWarning, "Forcing shared instance bitrate of %s", PrettySpeed(float64(sharedIfc.Bitrate)))
	}
	sharedIfc.ForceBitrateLatency = true

	ifaces.SharedConnectionDisappeared = SharedConnectionDisappeared
	ifaces.SharedConnectionReappeared = SharedConnectionReappeared

	ln, serverErr := ifaces.StartLocalInterfaceServer(ifaces.LocalConfig{
		UseAFUnix:          r.UseAFUnix,
		LocalSocketPath:    r.LocalSocketPath,
		LocalInterfacePort: r.localInterfacePort,
		Parent:             sharedIfc,
		OnClientDisconnect: func(cif *ifaces.Interface) {
			cif.Teardown()
		},
	}, func(cif *ifaces.Interface) {
		if cif != nil {
			interfacesMu.Lock()
			Interfaces = append(Interfaces, cif)
			interfacesMu.Unlock()
		}
		interfacesMu.Lock()
		LocalClientInterfaces = append(LocalClientInterfaces, cif)
		interfacesMu.Unlock()
	})
	if serverErr == nil {
		if r.RequireShared {
			_ = ln.Close()
			r.IsSharedInstance = true
			Log("Existing shared instance required, but this instance started as shared instance. Aborting startup.", LogVerbose)
			return errors.New("no shared instance available, but application that started Reticulum required it")
		}

		sharedIfc.OptimiseMTU()
		interfacesMu.Lock()
		Interfaces = append(Interfaces, sharedIfc)
		interfacesMu.Unlock()
		sharedIfc.SetClientCountFunc(func() int {
			return len(LocalClientInterfaces)
		})
		sharedIfc.LocalIsSharedInstance = true
		r.SharedInstanceInterface = sharedIfc
		r.IsSharedInstance = true
		r.IsStandaloneInstance = false
		r.IsConnectedToSharedInstance = false
		Logf(LogDebug, "Started shared instance interface: %s", r.SharedInstanceInterface)
		r.startJobs()
		return nil
	}

	clientIfc := &Interface{
		Name:                fmt.Sprintf("LocalInterface[%s]", siName),
		Type:                "LocalInterface",
		IN:                  true,
		OUT:                 true,
		DriverImplemented:   true,
		Online:              true,
		AutoconfigureMTU:    true,
		LocalIsSharedClient: true,
	}
	clientIfc.Bitrate = 1000000000
	sharedInstanceMu.RLock()
	clientBitrate := sharedInstanceForcedBitrate
	sharedInstanceMu.RUnlock()
	if clientBitrate > 0 {
		clientIfc.Bitrate = clientBitrate
		Logf(LogWarning, "Forcing shared instance bitrate of %s", PrettySpeed(float64(clientIfc.Bitrate)))
	}
	clientIfc.ForceBitrateLatency = true

	clientErr := ifaces.ConnectLocalInterfaceClient(ifaces.LocalConfig{
		UseAFUnix:          r.UseAFUnix,
		LocalSocketPath:    r.LocalSocketPath,
		LocalInterfacePort: r.localInterfacePort,
	}, clientIfc)
	if clientErr == nil {
		interfacesMu.Lock()
		Interfaces = append(Interfaces, clientIfc)
		interfacesMu.Unlock()
		r.SharedInstanceInterface = clientIfc
		r.IsSharedInstance = false
		r.IsStandaloneInstance = false
		r.IsConnectedToSharedInstance = true
		transportEnabled = false
		remoteManagementEnabled = false
		allowProbes = false
		Logf(LogDebug, "Connected to locally available Reticulum instance via: %s", clientIfc)
		return nil
	}

	Log("Local shared instance appears to be running, but it could not be connected", LogError)
	Log("The contained exception was: "+clientErr.Error(), LogError)
	r.IsSharedInstance = false
	r.IsStandaloneInstance = true
	r.IsConnectedToSharedInstance = false
	r.startJobs()
	return nil
}

func (r *Reticulum) synthesizeConfiguredInterfaces() error {
	if r.Config == nil {
		return nil
	}

	interfaces := r.Config.Section("interfaces")
	if interfaces == nil {
		return nil
	}
	seenNames := make(map[string]struct{}, len(interfaces.Sections()))

	for _, name := range interfaces.Sections() {
		sub := interfaces.Subsection(name)
		if sub == nil {
			continue
		}
		kv := map[string]string{}
		var collect func(sec *configobj.Section, prefix string)
		collect = func(sec *configobj.Section, prefix string) {
			if sec == nil {
				return
			}
			for _, key := range sec.Keys() {
				if val, ok := sec.Get(key); ok {
					if prefix != "" {
						kv[prefix+strings.ToLower(key)] = val
					} else {
						kv[strings.ToLower(key)] = val
					}
				}
			}
			for _, subName := range sec.Sections() {
				child := sec.Subsection(subName)
				if child == nil {
					continue
				}
				childPrefix := prefix
				if childPrefix != "" {
					childPrefix += "sub."
				} else {
					childPrefix = "sub."
				}
				childPrefix += strings.ToLower(subName) + "."
				collect(child, childPrefix)
			}
		}
		collect(sub, "")
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if kv == nil {
			continue
		}
		if _, exists := kv["name"]; !exists {
			kv["name"] = name
		}
		if strings.TrimSpace(kv["name"]) == "" {
			kv["name"] = name
		}
		if _, ok := seenNames[name]; ok {
			msg := fmt.Sprintf("The interface name %q was already used. Check your configuration file for errors!", name)
			Log(msg, LogError)
			return errors.New(msg)
		}
		seenNames[name] = struct{}{}
		if _, err := r.synthesizeInterface(name, kv, true); err != nil {
			return err
		}
	}
	return nil
}

func (r *Reticulum) synthesizeInterface(name string, kv map[string]string, instanceInit bool) (bool, error) {
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

	first := func(keys ...string) string {
		for _, k := range keys {
			lk := strings.ToLower(k)
			if v, ok := kv[lk]; ok {
				return strings.TrimSpace(v)
			}
			for mk, mv := range kv {
				if strings.ToLower(mk) == lk {
					return strings.TrimSpace(mv)
				}
			}
		}
		return ""
	}

	truthy := func(v string, defaultVal bool) bool {
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
	interfaceMode := func(v string) int {
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
	intValue := func(s string) (int, bool) {
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
	floatValue := func(s string) (float64, bool) {
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

	enabled := truthy(first("interface_enabled", "enabled", "enable"), false)
	if !enabled {
		Logf(LogDebug, "Skipping disabled interface %q", name)
		return false, nil
	}

	mode := interfaceMode(first("interface_mode", "selected_interface_mode", "mode"))

	var bitrate *int
	if v, ok := intValue(first("configured_bitrate", "bitrate")); ok && v >= MINIMUM_BITRATE {
		bitrate = &v
	}
	// Python sets interface_config["configured_bitrate"] for all interface drivers.
	if bitrate != nil && strings.TrimSpace(first("configured_bitrate")) == "" {
		kv["configured_bitrate"] = strconv.Itoa(*bitrate)
	}

	var announceCap *float64
	if v, ok := floatValue(first("announce_cap")); ok && v > 0 && v <= 100 {
		f := v / 100.0
		announceCap = &f
	}

	var ifacSize *int
	if v, ok := intValue(first("ifac_size")); ok && v >= IFAC_MIN_SIZE*8 {
		sz := v / 8
		ifacSize = &sz
	}

	netName := first("networkname", "network_name")
	var ifacNetname *string
	if strings.TrimSpace(netName) != "" {
		ifacNetname = &netName
	}

	netKey := first("passphrase", "pass_phrase")
	var ifacNetkey *string
	if strings.TrimSpace(netKey) != "" {
		ifacNetkey = &netKey
	}

	var announceRateTarget *int
	if v, ok := intValue(first("announce_rate_target")); ok && v > 0 {
		announceRateTarget = &v
	}
	var announceRateGrace *int
	if v, ok := intValue(first("announce_rate_grace")); ok && v >= 0 {
		announceRateGrace = &v
	}
	var announceRatePenalty *int
	if v, ok := intValue(first("announce_rate_penalty")); ok && v >= 0 {
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

	bootstrapOnly := truthy(first("bootstrap_only"), false)
	ignoreConfigWarnings := truthy(first("ignore_config_warnings"), false)

	ingressControl := truthy(first("ingress_control"), true)
	var icMaxHeld *int
	if v, ok := intValue(first("ic_max_held_announces")); ok {
		icMaxHeld = &v
	}
	var icBurstHold *float64
	if v, ok := floatValue(first("ic_burst_hold")); ok {
		icBurstHold = &v
	}
	var icBurstFreqNew *float64
	if v, ok := floatValue(first("ic_burst_freq_new")); ok {
		icBurstFreqNew = &v
	}
	var icBurstFreq *float64
	if v, ok := floatValue(first("ic_burst_freq")); ok {
		icBurstFreq = &v
	}
	var icNewTime *float64
	if v, ok := floatValue(first("ic_new_time")); ok {
		icNewTime = &v
	}
	var icBurstPenalty *float64
	if v, ok := floatValue(first("ic_burst_penalty")); ok {
		icBurstPenalty = &v
	}
	var icHeldRelease *float64
	if v, ok := floatValue(first("ic_held_release_interval")); ok {
		icHeldRelease = &v
	}

	ifType := strings.TrimSpace(first("type"))
	if ifType == "" {
		return false, errors.New("missing interface type")
	}

	if strings.EqualFold(ifType, "BackboneInterface") || strings.EqualFold(ifType, "BackboneClientInterface") {
		if v := strings.TrimSpace(first("port")); v != "" {
			if strings.TrimSpace(first("listen_port")) == "" {
				kv["listen_port"] = v
			}
			if strings.TrimSpace(first("target_port")) == "" {
				kv["target_port"] = v
			}
		}
		if v := strings.TrimSpace(first("remote")); v != "" && strings.TrimSpace(first("target_host")) == "" {
			kv["target_host"] = v
		}
		if v := strings.TrimSpace(first("listen_on")); v != "" && strings.TrimSpace(first("listen_ip")) == "" {
			kv["listen_ip"] = v
		}
	}

	if strings.EqualFold(ifType, "BackboneInterface") && strings.TrimSpace(first("target_host", "remote")) != "" {
		ifType = "BackboneClientInterface"
	}

	if strings.EqualFold(ifType, "TCPInterface") {
		if strings.TrimSpace(first("target_host", "remote")) != "" {
			ifType = "TCPClientInterface"
		} else {
			ifType = "TCPServerInterface"
		}
	}

	var externalIfc *Interface
	if loaded, handled, err := loadExternalInterfacePlugin(r.InterfacePath, ifType, name, kv); handled {
		if err != nil {
			Logf(LogError, "External interface initialisation failed for %s / %s: %v", ifType, name, err)
			return false, nil
		}
		externalIfc = loaded
	}
	if externalIfc == nil {
		if loaded, handled, err := loadExternalInterfacePython(r.InterfacePath, ifType, name, kv); handled {
			if err != nil {
				Logf(LogError, "External interface initialisation failed for %s / %s: %v", ifType, name, err)
				return false, nil
			}
			externalIfc = loaded
		}
	}

	if externalIfc == nil &&
		!strings.EqualFold(ifType, "UDPInterface") &&
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
			goPath := filepath.Join(r.InterfacePath, ifType+".go")
			if _, err := os.Stat(goPath); err == nil {
				Logf(LogError, "External interface %q found at %s but source-based Go interfaces are not supported, build a .so plugin instead", ifType, goPath)
				return false, nil
			}
		}
		Logf(LogError, "Could not locate external interface module %q in %q", ifType+".py", r.InterfacePath)
		return false, nil
	}

	discoverable := false
	discoveryAnnounceInterval := 6 * time.Hour
	var discoveryStampValue *int
	discoveryName := strings.TrimSpace(first("discovery_name"))
	discoveryEncrypt := truthy(first("discovery_encrypt"), false)
	reachableOn := strings.TrimSpace(first("reachable_on"))
	publishIFAC := truthy(first("publish_ifac"), false)
	var discoveryLatitude *float64
	var discoveryLongitude *float64
	var discoveryHeight *float64
	var discoveryFrequency *int
	var discoveryBandwidth *int
	var discoveryChannel *int
	discoveryModulation := strings.TrimSpace(first("discovery_modulation"))
	if truthy(first("discoverable"), false) {
		discoverable = true
		discoveryEnabled = true
		if v, ok := intValue(first("announce_interval")); ok {
			discoveryAnnounceInterval = time.Duration(v) * time.Minute
			if discoveryAnnounceInterval < 5*time.Minute {
				discoveryAnnounceInterval = 5 * time.Minute
			}
		}
		if v, ok := intValue(first("discovery_stamp_value")); ok && v > 0 {
			discoveryStampValue = &v
		}
		if v, ok := floatValue(first("latitude")); ok {
			discoveryLatitude = &v
		}
		if v, ok := floatValue(first("longitude")); ok {
			discoveryLongitude = &v
		}
		if v, ok := floatValue(first("height")); ok {
			discoveryHeight = &v
		}
		if v, ok := intValue(first("discovery_frequency")); ok {
			discoveryFrequency = &v
		}
		if v, ok := intValue(first("discovery_bandwidth")); ok {
			discoveryBandwidth = &v
		}
		if v, ok := intValue(first("discovery_channel")); ok {
			discoveryChannel = &v
		}
		if mode != InterfaceModeGateway && mode != InterfaceModeAccessPoint {
			if strings.EqualFold(ifType, "RNodeInterface") || strings.EqualFold(ifType, "RNodeMultiInterface") {
				mode = InterfaceModeAccessPoint
				if !ignoreConfigWarnings {
					Logf(LogNotice, "Discovery enabled on interface %s without gateway or AP mode. Auto-configured to AP mode.", name)
				}
			} else {
				mode = InterfaceModeGateway
				if !ignoreConfigWarnings {
					Logf(LogNotice, "Discovery enabled on interface %s without gateway or AP mode. Auto-configured to gateway mode.", name)
				}
			}
		}
	}

	var ifc *Interface
	if externalIfc != nil {
		ifc = externalIfc
	} else {
		switch strings.ToLower(ifType) {
		case "udpinterface":
			ifc = &Interface{Name: name, Type: "UDPInterface", DriverImplemented: true}
		case "tcpclientinterface":
			ifc = &Interface{Name: name, Type: "TCPClientInterface", DriverImplemented: true}
		case "tcpserverinterface":
			ifc = &Interface{Name: name, Type: "TCPServerInterface", DriverImplemented: true}
		case "serialinterface":
			ifc = &Interface{Name: name, Type: "SerialInterface", DriverImplemented: true}
		case "ax25kissinterface":
			ifc = &Interface{Name: name, Type: "AX25KISSInterface", DriverImplemented: true}
		case "backboneinterface":
			ifc = &Interface{Name: name, Type: "BackboneInterface", DriverImplemented: true}
		case "backboneclientinterface":
			ifc = &Interface{Name: name, Type: "BackboneClientInterface", DriverImplemented: true}
		case "kissinterface":
			ifc = &Interface{Name: name, Type: "KISSInterface", DriverImplemented: true}
		case "i2pinterface":
			ifc = &Interface{Name: name, Type: "I2PInterface", DriverImplemented: true}
		case "autointerface":
			ifc = &Interface{Name: name, Type: "AutoInterface", DriverImplemented: true}
		case "pipeinterface":
			ifc = &Interface{Name: name, Type: "PipeInterface", DriverImplemented: true}
		case "rnodemultiinterface":
			ifc = &Interface{Name: name, Type: "RNodeMultiInterface", DriverImplemented: true}
		case "rnodeinterface":
			ifc = &Interface{Name: name, Type: "RNodeInterface", DriverImplemented: true}
		case "weaveinterface":
			ifc = &Interface{Name: name, Type: "WeaveInterface", DriverImplemented: true}
		default:
			return false, fmt.Errorf("unknown interface type %q", ifType)
		}
	}

	ifc.OUT = truthy(first("outgoing"), true)
	ifc.IngressControl = ingressControl
	ifc.ICMaxHeldAnnounces = icMaxHeld
	ifc.ICBurstHold = icBurstHold
	ifc.ICBurstFreqNew = icBurstFreqNew
	ifc.ICBurstFreq = icBurstFreq
	ifc.ICNewTime = icNewTime
	ifc.ICBurstPenalty = icBurstPenalty
	ifc.ICHeldReleaseInterval = icHeldRelease
	ifc.SetAnnounceRateConfig(announceRateTarget, announceRateGrace, announceRatePenalty)

	if externalIfc == nil && strings.EqualFold(ifType, "UDPInterface") {
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

		device := first("device")
		port, _ := intValue(first("port"))
		listenIP := first("listen_ip")
		listenPort, _ := intValue(first("listen_port"))
		forwardIP := first("forward_ip")
		forwardPort, _ := intValue(first("forward_port"))
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

	if externalIfc == nil && strings.EqualFold(ifType, "AutoInterface") {
		desiredOut := ifc.OUT
		if err := ifc.ConfigureAutoInterface(kv); err != nil {
			return false, fmt.Errorf("Interface %q AutoInterface config error: %w", name, err)
		}
		ifc.OUT = desiredOut
		if err := ifc.StartAutoInterface(); err != nil {
			return false, fmt.Errorf("Interface %q AutoInterface start error: %w", name, err)
		}
	}

	if externalIfc == nil && strings.EqualFold(ifType, "AX25KISSInterface") {
		axIf, err := ifaces.NewAX25KISSInterface(strings.TrimSpace(name), kv)
		if err != nil {
			return false, fmt.Errorf("Interface %q AX25KISS config error: %w", name, err)
		}
		axIf.IN = ifc.IN
		axIf.OUT = ifc.OUT
		axIf.IngressControl = ifc.IngressControl
		axIf.ICMaxHeldAnnounces = ifc.ICMaxHeldAnnounces
		axIf.ICBurstHold = ifc.ICBurstHold
		axIf.ICBurstFreqNew = ifc.ICBurstFreqNew
		axIf.ICBurstFreq = ifc.ICBurstFreq
		axIf.ICNewTime = ifc.ICNewTime
		axIf.ICBurstPenalty = ifc.ICBurstPenalty
		axIf.ICHeldReleaseInterval = ifc.ICHeldReleaseInterval
		axIf.AnnounceCap = ifc.AnnounceCap
		axIf.SetAnnounceRateConfig(ifc.AnnounceRateTarget, ifc.AnnounceRateGrace, ifc.AnnounceRatePenalty)
		ifc = axIf
	}

	if externalIfc == nil && strings.EqualFold(ifType, "KISSInterface") {
		kIf, err := ifaces.NewKISSInterface(strings.TrimSpace(name), kv)
		if err != nil {
			return false, fmt.Errorf("Interface %q KISS config error: %w", name, err)
		}
		kIf.IN = ifc.IN
		kIf.OUT = ifc.OUT
		kIf.IngressControl = ifc.IngressControl
		kIf.ICMaxHeldAnnounces = ifc.ICMaxHeldAnnounces
		kIf.ICBurstHold = ifc.ICBurstHold
		kIf.ICBurstFreqNew = ifc.ICBurstFreqNew
		kIf.ICBurstFreq = ifc.ICBurstFreq
		kIf.ICNewTime = ifc.ICNewTime
		kIf.ICBurstPenalty = ifc.ICBurstPenalty
		kIf.ICHeldReleaseInterval = ifc.ICHeldReleaseInterval
		kIf.AnnounceCap = ifc.AnnounceCap
		kIf.SetAnnounceRateConfig(ifc.AnnounceRateTarget, ifc.AnnounceRateGrace, ifc.AnnounceRatePenalty)
		ifc = kIf
	}

	if externalIfc == nil && strings.EqualFold(ifType, "BackboneInterface") {
		bbIf, err := ifaces.NewBackboneInterface(strings.TrimSpace(name), kv)
		if err != nil {
			return false, fmt.Errorf("Interface %q Backbone config error: %w", name, err)
		}
		bbIf.IN = ifc.IN
		bbIf.OUT = ifc.OUT
		bbIf.IngressControl = ifc.IngressControl
		bbIf.ICMaxHeldAnnounces = ifc.ICMaxHeldAnnounces
		bbIf.ICBurstHold = ifc.ICBurstHold
		bbIf.ICBurstFreqNew = ifc.ICBurstFreqNew
		bbIf.ICBurstFreq = ifc.ICBurstFreq
		bbIf.ICNewTime = ifc.ICNewTime
		bbIf.ICBurstPenalty = ifc.ICBurstPenalty
		bbIf.ICHeldReleaseInterval = ifc.ICHeldReleaseInterval
		bbIf.AnnounceCap = ifc.AnnounceCap
		bbIf.SetAnnounceRateConfig(ifc.AnnounceRateTarget, ifc.AnnounceRateGrace, ifc.AnnounceRatePenalty)
		ifc = bbIf
	}

	if externalIfc == nil && strings.EqualFold(ifType, "BackboneClientInterface") {
		bcIf, err := ifaces.NewBackboneClientInterface(strings.TrimSpace(name), kv)
		if err != nil {
			return false, fmt.Errorf("Interface %q Backbone client config error: %w", name, err)
		}
		bcIf.IN = ifc.IN
		bcIf.OUT = ifc.OUT
		bcIf.IngressControl = ifc.IngressControl
		bcIf.ICMaxHeldAnnounces = ifc.ICMaxHeldAnnounces
		bcIf.ICBurstHold = ifc.ICBurstHold
		bcIf.ICBurstFreqNew = ifc.ICBurstFreqNew
		bcIf.ICBurstFreq = ifc.ICBurstFreq
		bcIf.ICNewTime = ifc.ICNewTime
		bcIf.ICBurstPenalty = ifc.ICBurstPenalty
		bcIf.ICHeldReleaseInterval = ifc.ICHeldReleaseInterval
		bcIf.AnnounceCap = ifc.AnnounceCap
		bcIf.SetAnnounceRateConfig(ifc.AnnounceRateTarget, ifc.AnnounceRateGrace, ifc.AnnounceRatePenalty)
		ifc = bcIf
	}

	if externalIfc == nil && strings.EqualFold(ifType, "WeaveInterface") {
		if strings.TrimSpace(first("identitypath")) == "" && strings.TrimSpace(r.IdentityPath) != "" {
			kv["identitypath"] = r.IdentityPath
		}
		wIf, err := ifaces.NewWeaveInterface(strings.TrimSpace(name), kv)
		if err != nil {
			return false, fmt.Errorf("Interface %q Weave config error: %w", name, err)
		}
		wIf.IN = ifc.IN
		wIf.OUT = ifc.OUT
		wIf.IngressControl = ifc.IngressControl
		wIf.ICMaxHeldAnnounces = ifc.ICMaxHeldAnnounces
		wIf.ICBurstHold = ifc.ICBurstHold
		wIf.ICBurstFreqNew = ifc.ICBurstFreqNew
		wIf.ICBurstFreq = ifc.ICBurstFreq
		wIf.ICNewTime = ifc.ICNewTime
		wIf.ICBurstPenalty = ifc.ICBurstPenalty
		wIf.ICHeldReleaseInterval = ifc.ICHeldReleaseInterval
		wIf.AnnounceCap = ifc.AnnounceCap
		wIf.SetAnnounceRateConfig(ifc.AnnounceRateTarget, ifc.AnnounceRateGrace, ifc.AnnounceRatePenalty)
		wIf.IngressControl = false
		ifc = wIf
	}

	if externalIfc == nil && strings.EqualFold(ifType, "I2PInterface") {
		if strings.TrimSpace(first("storagepath")) == "" && strings.TrimSpace(r.StoragePath) != "" {
			kv["storagepath"] = r.StoragePath
		}
		i2pIf, err := ifaces.NewI2PInterface(strings.TrimSpace(name), kv)
		if err != nil {
			return false, fmt.Errorf("Interface %q I2P config error: %w", name, err)
		}
		i2pIf.IN = ifc.IN
		i2pIf.OUT = ifc.OUT
		i2pIf.IngressControl = ifc.IngressControl
		i2pIf.ICMaxHeldAnnounces = ifc.ICMaxHeldAnnounces
		i2pIf.ICBurstHold = ifc.ICBurstHold
		i2pIf.ICBurstFreqNew = ifc.ICBurstFreqNew
		i2pIf.ICBurstFreq = ifc.ICBurstFreq
		i2pIf.ICNewTime = ifc.ICNewTime
		i2pIf.ICBurstPenalty = ifc.ICBurstPenalty
		i2pIf.ICHeldReleaseInterval = ifc.ICHeldReleaseInterval
		i2pIf.AnnounceCap = ifc.AnnounceCap
		i2pIf.SetAnnounceRateConfig(ifc.AnnounceRateTarget, ifc.AnnounceRateGrace, ifc.AnnounceRatePenalty)
		ifc = i2pIf
	}

	if externalIfc == nil && strings.EqualFold(ifType, "PipeInterface") {
		pIf, err := ifaces.NewPipeInterface(strings.TrimSpace(name), kv)
		if err != nil {
			return false, fmt.Errorf("Interface %q Pipe config error: %w", name, err)
		}
		pIf.IN = ifc.IN
		pIf.OUT = ifc.OUT
		pIf.IngressControl = ifc.IngressControl
		pIf.ICMaxHeldAnnounces = ifc.ICMaxHeldAnnounces
		pIf.ICBurstHold = ifc.ICBurstHold
		pIf.ICBurstFreqNew = ifc.ICBurstFreqNew
		pIf.ICBurstFreq = ifc.ICBurstFreq
		pIf.ICNewTime = ifc.ICNewTime
		pIf.ICBurstPenalty = ifc.ICBurstPenalty
		pIf.ICHeldReleaseInterval = ifc.ICHeldReleaseInterval
		pIf.AnnounceCap = ifc.AnnounceCap
		pIf.SetAnnounceRateConfig(ifc.AnnounceRateTarget, ifc.AnnounceRateGrace, ifc.AnnounceRatePenalty)
		ifc = pIf
	}

	if externalIfc == nil && strings.EqualFold(ifType, "SerialInterface") {
		sIf, err := ifaces.NewSerialInterface(strings.TrimSpace(name), kv)
		if err != nil {
			return false, fmt.Errorf("Interface %q Serial config error: %w", name, err)
		}
		sIf.IN = ifc.IN
		sIf.OUT = ifc.OUT
		sIf.IngressControl = ifc.IngressControl
		sIf.ICMaxHeldAnnounces = ifc.ICMaxHeldAnnounces
		sIf.ICBurstHold = ifc.ICBurstHold
		sIf.ICBurstFreqNew = ifc.ICBurstFreqNew
		sIf.ICBurstFreq = ifc.ICBurstFreq
		sIf.ICNewTime = ifc.ICNewTime
		sIf.ICBurstPenalty = ifc.ICBurstPenalty
		sIf.ICHeldReleaseInterval = ifc.ICHeldReleaseInterval
		sIf.AnnounceCap = ifc.AnnounceCap
		sIf.SetAnnounceRateConfig(ifc.AnnounceRateTarget, ifc.AnnounceRateGrace, ifc.AnnounceRatePenalty)
		ifc = sIf
	}

	if externalIfc == nil && strings.EqualFold(ifType, "RNodeMultiInterface") {
		rnmIf, err := ifaces.NewRNodeMultiInterface(strings.TrimSpace(name), kv)
		if err != nil {
			return false, fmt.Errorf("Interface %q RNodeMulti config error: %w", name, err)
		}
		rnmIf.IN = ifc.IN
		rnmIf.OUT = ifc.OUT
		rnmIf.IngressControl = ifc.IngressControl
		rnmIf.ICMaxHeldAnnounces = ifc.ICMaxHeldAnnounces
		rnmIf.ICBurstHold = ifc.ICBurstHold
		rnmIf.ICBurstFreqNew = ifc.ICBurstFreqNew
		rnmIf.ICBurstFreq = ifc.ICBurstFreq
		rnmIf.ICNewTime = ifc.ICNewTime
		rnmIf.ICBurstPenalty = ifc.ICBurstPenalty
		rnmIf.ICHeldReleaseInterval = ifc.ICHeldReleaseInterval
		rnmIf.AnnounceCap = ifc.AnnounceCap
		rnmIf.SetAnnounceRateConfig(ifc.AnnounceRateTarget, ifc.AnnounceRateGrace, ifc.AnnounceRatePenalty)
		ifc = rnmIf
	}

	if externalIfc == nil && strings.EqualFold(ifType, "RNodeInterface") {
		rnIf, err := ifaces.NewRNodeInterfaceFromConfig(strings.TrimSpace(name), kv)
		if err != nil {
			return false, fmt.Errorf("Interface %q RNode config error: %w", name, err)
		}
		rnIf.IN = ifc.IN
		rnIf.OUT = ifc.OUT
		rnIf.IngressControl = ifc.IngressControl
		rnIf.ICMaxHeldAnnounces = ifc.ICMaxHeldAnnounces
		rnIf.ICBurstHold = ifc.ICBurstHold
		rnIf.ICBurstFreqNew = ifc.ICBurstFreqNew
		rnIf.ICBurstFreq = ifc.ICBurstFreq
		rnIf.ICNewTime = ifc.ICNewTime
		rnIf.ICBurstPenalty = ifc.ICBurstPenalty
		rnIf.ICHeldReleaseInterval = ifc.ICHeldReleaseInterval
		rnIf.AnnounceCap = ifc.AnnounceCap
		rnIf.SetAnnounceRateConfig(ifc.AnnounceRateTarget, ifc.AnnounceRateGrace, ifc.AnnounceRatePenalty)
		ifc = rnIf
	}

	if externalIfc == nil && strings.EqualFold(ifType, "TCPClientInterface") {
		host := first("target_host", "remote")
		port, _ := intValue(first("target_port", "port"))
		kiss := truthy(first("kiss_framing"), false)
		i2p := truthy(first("i2p_tunneled"), false)
		fixedMTU, _ := intValue(first("fixed_mtu", "mtu"))
		connectTimeoutS, _ := floatValue(first("connect_timeout"))
		reconnectWaitS, _ := floatValue(first("reconnect_wait"))
		var maxReconnect *int
		if v, ok := intValue(first("max_reconnect_tries")); ok {
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
		tcpIf.IN = ifc.IN
		tcpIf.OUT = ifc.OUT
		tcpIf.IngressControl = ifc.IngressControl
		tcpIf.ICMaxHeldAnnounces = ifc.ICMaxHeldAnnounces
		tcpIf.ICBurstHold = ifc.ICBurstHold
		tcpIf.ICBurstFreqNew = ifc.ICBurstFreqNew
		tcpIf.ICBurstFreq = ifc.ICBurstFreq
		tcpIf.ICNewTime = ifc.ICNewTime
		tcpIf.ICBurstPenalty = ifc.ICBurstPenalty
		tcpIf.ICHeldReleaseInterval = ifc.ICHeldReleaseInterval
		tcpIf.AnnounceCap = ifc.AnnounceCap
		tcpIf.SetAnnounceRateConfig(ifc.AnnounceRateTarget, ifc.AnnounceRateGrace, ifc.AnnounceRatePenalty)
		ifc = tcpIf
	}

	if externalIfc == nil && strings.EqualFold(ifType, "TCPServerInterface") {
		listenIP := first("listen_ip", "listen_on", "bind_ip")
		listenPort, _ := intValue(first("listen_port", "port"))
		device := first("device")
		preferIPv6 := truthy(first("prefer_ipv6"), false)
		kiss := truthy(first("kiss_framing"), false)
		i2p := truthy(first("i2p_tunneled"), false)
		fixedMTU, _ := intValue(first("fixed_mtu", "mtu"))

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

		logger := Logf
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
			peer.IN = parent.IN
			peer.OUT = parent.OUT
			peer.IngressControl = parent.IngressControl
			peer.ICMaxHeldAnnounces = parent.ICMaxHeldAnnounces
			peer.ICBurstHold = parent.ICBurstHold
			peer.ICBurstFreqNew = parent.ICBurstFreqNew
			peer.ICBurstFreq = parent.ICBurstFreq
			peer.ICNewTime = parent.ICNewTime
			peer.ICBurstPenalty = parent.ICBurstPenalty
			peer.ICHeldReleaseInterval = parent.ICHeldReleaseInterval
			peer.AnnounceCap = parent.AnnounceCap
			peer.SetAnnounceRateConfig(parent.AnnounceRateTarget, parent.AnnounceRateGrace, parent.AnnounceRatePenalty)
			peer.SetTCPClient(ci)
			ci.Owner = func(data []byte, _ *ifaces.TCPClientInterface) {
				if peer == nil || len(data) == 0 {
					return
				}
				peer.RXB += uint64(len(data))
				if ifaces.InboundHandler != nil {
					ifaces.InboundHandler(data, peer)
				}
			}
			interfacesMu.Lock()
			Interfaces = append(Interfaces, peer)
			interfacesMu.Unlock()
		}

		if err := server.Start(); err != nil {
			return false, fmt.Errorf("Interface %q TCP server start error: %w", name, err)
		}

		parent.IN = ifc.IN
		parent.OUT = ifc.OUT
		parent.IngressControl = ifc.IngressControl
		parent.ICMaxHeldAnnounces = ifc.ICMaxHeldAnnounces
		parent.ICBurstHold = ifc.ICBurstHold
		parent.ICBurstFreqNew = ifc.ICBurstFreqNew
		parent.ICBurstFreq = ifc.ICBurstFreq
		parent.ICNewTime = ifc.ICNewTime
		parent.ICBurstPenalty = ifc.ICBurstPenalty
		parent.ICHeldReleaseInterval = ifc.ICHeldReleaseInterval
		parent.AnnounceCap = ifc.AnnounceCap
		parent.SetAnnounceRateConfig(ifc.AnnounceRateTarget, ifc.AnnounceRateGrace, ifc.AnnounceRatePenalty)
		ifc = parent
	}

	if discoverable {
		ifc.Discoverable = true
		ifc.DiscoveryAnnounceInterval = discoveryAnnounceInterval
		ifc.DiscoveryPublishIFAC = publishIFAC
		ifc.DiscoveryReachableOn = reachableOn
		ifc.DiscoveryName = discoveryName
		ifc.DiscoveryEncrypt = discoveryEncrypt
		ifc.DiscoveryStampValue = discoveryStampValue
		ifc.DiscoveryLatitude = discoveryLatitude
		ifc.DiscoveryLongitude = discoveryLongitude
		ifc.DiscoveryHeight = discoveryHeight
		ifc.DiscoveryFrequency = discoveryFrequency
		ifc.DiscoveryBandwidth = discoveryBandwidth
		ifc.DiscoveryChannel = discoveryChannel
		ifc.DiscoveryModulation = discoveryModulation
	}
	ifc.BootstrapOnly = bootstrapOnly
	kv["name"] = name
	kv["selected_interface_mode"] = strconv.Itoa(mode)
	kv["configured_bitrate"] = ""
	if bitrate != nil {
		kv["configured_bitrate"] = strconv.Itoa(*bitrate)
	}
	r.AddInterface(ifc, mode, bitrate, ifacSize, ifacNetname, ifacNetkey, announceCap, announceRateTarget, announceRateGrace, announceRatePenalty)
	if bootstrapOnly && instanceInit {
		r.BootstrapConfigs = append(r.BootstrapConfigs, kv)
	}
	return true, nil
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

	interfacesMu.Lock()
	Interfaces = append(Interfaces, ifc)
	interfacesMu.Unlock()
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
	PersistData()
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

// Local RPC uses a small auth handshake plus msgpack for call/response
// serialization. The listener and conn are concrete. The current implementation
// targets Go↔Python.

func (r *Reticulum) rpcLoop() {
	if r == nil || r.rpcLn == nil {
		return
	}
	for {
		conn, err := r.rpcLn.Accept()
		if err != nil {
			// During shutdown the listener will be closed; don't spin.
			if strings.Contains(strings.ToLower(err.Error()), "closed") {
				if r.rpcNetwork == "unix" && len(r.rpcAddr) > 0 && r.rpcAddr[0] != 0 {
					_ = os.Remove(r.rpcAddr)
				}
				return
			}
			Log("Error accepting RPC: "+err.Error(), LogError)
			continue
		}
		if err := func() error {
			var hdr [4]byte
			// deliver_challenge: server sends its nonce, verifies client digest
			randBytes := make([]byte, 20)
			if _, err := rand.Read(randBytes); err != nil {
				return err
			}
			msg := append([]byte("#CHALLENGE#"), randBytes...)
			binary.BigEndian.PutUint32(hdr[:], uint32(len(msg)))
			if _, err := conn.Write(hdr[:]); err != nil {
				return err
			}
			if _, err := conn.Write(msg); err != nil {
				return err
			}
			if _, err := io.ReadFull(conn, hdr[:]); err != nil {
				return err
			}
			n := int(binary.BigEndian.Uint32(hdr[:]))
			if n > 256 {
				return fmt.Errorf("rpc message too large: %d bytes", n)
			}
			resp := make([]byte, n)
			if _, err := io.ReadFull(conn, resp); err != nil {
				return err
			}
			mac := hmac.New(md5.New, r.RPCKey)
			if _, err := mac.Write(randBytes); err != nil {
				return err
			}
			expected := mac.Sum(nil)
			if subtle.ConstantTimeCompare(resp, expected) != 1 {
				binary.BigEndian.PutUint32(hdr[:], uint32(len("#FAILURE#")))
				_, _ = conn.Write(hdr[:])
				_, _ = conn.Write([]byte("#FAILURE#"))
				return errors.New("digest sent was rejected")
			}
			binary.BigEndian.PutUint32(hdr[:], uint32(len("#WELCOME#")))
			if _, err := conn.Write(hdr[:]); err != nil {
				return err
			}
			if _, err := conn.Write([]byte("#WELCOME#")); err != nil {
				return err
			}
			// answer_challenge: server receives client nonce and sends digest
			if _, err := io.ReadFull(conn, hdr[:]); err != nil {
				return err
			}
			n = int(binary.BigEndian.Uint32(hdr[:]))
			if n > 256 {
				return fmt.Errorf("rpc message too large: %d bytes", n)
			}
			clientMsg := make([]byte, n)
			if _, err := io.ReadFull(conn, clientMsg); err != nil {
				return err
			}
			const challengePrefix = "#CHALLENGE#"
			if len(clientMsg) <= len(challengePrefix) || string(clientMsg[:len(challengePrefix)]) != challengePrefix {
				return errors.New("expected challenge from client")
			}
			clientNonce := clientMsg[len(challengePrefix):]
			mac2 := hmac.New(md5.New, r.RPCKey)
			if _, err := mac2.Write(clientNonce); err != nil {
				return err
			}
			digest := mac2.Sum(nil)
			binary.BigEndian.PutUint32(hdr[:], uint32(len(digest)))
			if _, err := conn.Write(hdr[:]); err != nil {
				return err
			}
			if _, err := conn.Write(digest); err != nil {
				return err
			}
			if _, err := io.ReadFull(conn, hdr[:]); err != nil {
				return err
			}
			n = int(binary.BigEndian.Uint32(hdr[:]))
			if n > 256 {
				return fmt.Errorf("rpc message too large: %d bytes", n)
			}
			finalResp := make([]byte, n)
			if _, err := io.ReadFull(conn, finalResp); err != nil {
				return err
			}
			if string(finalResp) != "#WELCOME#" {
				return errors.New("digest sent was rejected")
			}
			return nil
		}(); err != nil {
			Log("Rejected RPC connection: "+err.Error(), LogWarning)
			_ = conn.Close()
			continue
		}
		go r.handleRPC(conn)
	}
}

func (r *Reticulum) handleRPC(conn net.Conn) {
	defer func() {
		if rec := recover(); rec != nil {
			Log("An error ocurred while handling RPC call from local client: "+fmt.Sprint(rec), LogError)
		}
	}()
	defer conn.Close()

	var call map[string]any
	if err := func() error {
		data, err := func() ([]byte, error) {
			var hdr [4]byte
			if _, err := io.ReadFull(conn, hdr[:]); err != nil {
				return nil, err
			}
			n := int(binary.BigEndian.Uint32(hdr[:]))
			buf := make([]byte, n)
			if _, err := io.ReadFull(conn, buf); err != nil {
				return nil, err
			}
			return buf, nil
		}()
		if err != nil {
			return err
		}
		result, err := vendor.PickleLoads(data)
		if err != nil {
			return fmt.Errorf("pickle decode: %w", err)
		}
		return vendor.PickleAssign(result, &call)
	}(); err != nil {
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
			if err := func() error {
				data, err := vendor.PickleDumps(r.GetInterfaceStats())
				if err != nil {
					return err
				}
				return func() error {
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
				}()
			}(); err != nil {
				panic(err)
			}
		case "blackholed_identities":
			snapshot := r.GetBlackholedIdentities()
			out := make(map[string]map[string]any, len(snapshot))
			for key, entry := range snapshot {
				out[hex.EncodeToString(key[:])] = entry
			}
			if err := func() error {
				data, err := vendor.PickleDumps(out)
				if err != nil {
					return err
				}
				return func() error {
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
				}()
			}(); err != nil {
				panic(err)
			}
		case "path_table":
			var mh *int
			switch v := call["max_hops"].(type) {
			case int:
				mh = &v
			case int8:
				vv := int(v)
				mh = &vv
			case int16:
				vv := int(v)
				mh = &vv
			case int32:
				vv := int(v)
				mh = &vv
			case int64:
				vv := int(v)
				mh = &vv
			case uint:
				vv := int(v)
				mh = &vv
			case uint8:
				vv := int(v)
				mh = &vv
			case uint16:
				vv := int(v)
				mh = &vv
			case uint32:
				vv := int(v)
				mh = &vv
			case uint64:
				vv := int(v)
				mh = &vv
			case float32:
				vv := int(v)
				mh = &vv
			case float64:
				vv := int(v)
				mh = &vv
			}
			if err := func() error {
				data, err := vendor.PickleDumps(r.GetPathTable(mh))
				if err != nil {
					return err
				}
				return func() error {
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
				}()
			}(); err != nil {
				panic(err)
			}
		case "rate_table":
			if err := func() error {
				data, err := vendor.PickleDumps(r.GetRateTable())
				if err != nil {
					return err
				}
				return func() error {
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
				}()
			}(); err != nil {
				panic(err)
			}
		case "next_hop_if_name":
			dst, _ := call["destination_hash"].([]byte)
			if err := func() error {
				data, err := vendor.PickleDumps(r.GetNextHopIfName(dst))
				if err != nil {
					return err
				}
				return func() error {
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
				}()
			}(); err != nil {
				panic(err)
			}
		case "next_hop":
			dst, _ := call["destination_hash"].([]byte)
			if err := func() error {
				data, err := vendor.PickleDumps(r.GetNextHop(dst))
				if err != nil {
					return err
				}
				return func() error {
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
				}()
			}(); err != nil {
				panic(err)
			}
		case "first_hop_timeout":
			dst, _ := call["destination_hash"].([]byte)
			if err := func() error {
				data, err := vendor.PickleDumps(r.GetFirstHopTimeout(dst))
				if err != nil {
					return err
				}
				return func() error {
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
				}()
			}(); err != nil {
				panic(err)
			}
		case "link_count":
			if err := func() error {
				data, err := vendor.PickleDumps(r.GetLinkCount())
				if err != nil {
					return err
				}
				return func() error {
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
				}()
			}(); err != nil {
				panic(err)
			}
		case "packet_rssi":
			hash, _ := call["packet_hash"].([]byte)
			localStatsMu.RLock()
			v, ok := localRSSICache[string(hash)]
			localStatsMu.RUnlock()
			var data []byte
			var err error
			if ok {
				data, err = vendor.PickleDumps(v)
			} else {
				data, err = vendor.PickleDumps(nil)
			}
			if err != nil {
				panic(err)
			}
			if err := func() error {
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
			}(); err != nil {
				panic(err)
			}
		case "packet_snr":
			hash, _ := call["packet_hash"].([]byte)
			localStatsMu.RLock()
			v, ok := localSNRCache[string(hash)]
			localStatsMu.RUnlock()
			var data []byte
			var err error
			if ok {
				data, err = vendor.PickleDumps(v)
			} else {
				data, err = vendor.PickleDumps(nil)
			}
			if err != nil {
				panic(err)
			}
			if err := func() error {
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
			}(); err != nil {
				panic(err)
			}
		case "packet_q":
			hash, _ := call["packet_hash"].([]byte)
			localStatsMu.RLock()
			v, ok := localQCache[string(hash)]
			localStatsMu.RUnlock()
			var data []byte
			var err error
			if ok {
				data, err = vendor.PickleDumps(v)
			} else {
				data, err = vendor.PickleDumps(nil)
			}
			if err != nil {
				panic(err)
			}
			if err := func() error {
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
			}(); err != nil {
				panic(err)
			}
		}
	}
	if drop, ok := call["drop"].(string); ok {
		switch drop {
		case "path":
			dst, _ := call["destination_hash"].([]byte)
			if err := func() error {
				data, err := vendor.PickleDumps(r.DropPath(dst))
				if err != nil {
					return err
				}
				return func() error {
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
				}()
			}(); err != nil {
				panic(err)
			}
		case "all_via":
			dst, _ := call["destination_hash"].([]byte)
			if err := func() error {
				data, err := vendor.PickleDumps(r.DropAllVia(dst))
				if err != nil {
					return err
				}
				return func() error {
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
				}()
			}(); err != nil {
				panic(err)
			}
		case "announce_queues":
			if err := func() error {
				data, err := vendor.PickleDumps(r.DropAnnounceQueues())
				if err != nil {
					return err
				}
				return func() error {
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
				}()
			}(); err != nil {
				panic(err)
			}
		}
	}
	if raw, ok := call["blackhole_identity"]; ok {
		identityHash, _ := raw.([]byte)
		until := func() *time.Time {
			value, ok := call["until"]
			if !ok || value == nil {
				return nil
			}
			switch v := value.(type) {
			case time.Time:
				t := v
				return &t
			case *time.Time:
				if v == nil {
					return nil
				}
				t := *v
				return &t
			case float64:
				sec, frac := math.Modf(v)
				if sec < 0 {
					t := time.Unix(0, 0)
					return &t
				}
				t := time.Unix(int64(sec), int64(frac*1e9))
				return &t
			case float32:
				fv := float64(v)
				sec, frac := math.Modf(fv)
				if sec < 0 {
					t := time.Unix(0, 0)
					return &t
				}
				t := time.Unix(int64(sec), int64(frac*1e9))
				return &t
			case int:
				t := time.Unix(int64(v), 0)
				return &t
			case int64:
				t := time.Unix(v, 0)
				return &t
			case int32:
				t := time.Unix(int64(v), 0)
				return &t
			case uint64:
				t := time.Unix(int64(v), 0)
				return &t
			case uint32:
				t := time.Unix(int64(v), 0)
				return &t
			default:
				return nil
			}
		}()
		reason := func() *string {
			value, ok := call["reason"]
			if !ok || value == nil {
				return nil
			}
			switch v := value.(type) {
			case string:
				s := v
				return &s
			case *string:
				if v == nil {
					return nil
				}
				s := *v
				return &s
			default:
				s := fmt.Sprint(v)
				return &s
			}
		}()
		data, err := vendor.PickleDumps(r.BlackholeIdentity(identityHash, until, reason))
		if err != nil {
			panic(err)
		}
		if err := func() error {
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
		}(); err != nil {
			panic(err)
		}
	}
	if raw, ok := call["unblackhole_identity"]; ok {
		identityHash, _ := raw.([]byte)
		data, err := vendor.PickleDumps(r.UnblackholeIdentity(identityHash))
		if err != nil {
			panic(err)
		}
		if err := func() error {
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
		}(); err != nil {
			panic(err)
		}
	}
}

func (r *Reticulum) getRPCClient() (net.Conn, error) {
	conn, err := net.Dial(r.rpcNetwork, r.rpcAddr)
	if err != nil {
		return nil, err
	}
	if err := func() error {
		var hdr [4]byte
		// answer_challenge: client receives server nonce, sends digest
		if _, err := io.ReadFull(conn, hdr[:]); err != nil {
			return err
		}
		n := int(binary.BigEndian.Uint32(hdr[:]))
		if n > 256 {
			return fmt.Errorf("rpc message too large: %d bytes", n)
		}
		serverMsg := make([]byte, n)
		if _, err := io.ReadFull(conn, serverMsg); err != nil {
			return err
		}
		const challengePrefix = "#CHALLENGE#"
		if len(serverMsg) <= len(challengePrefix) || string(serverMsg[:len(challengePrefix)]) != challengePrefix {
			return errors.New("expected challenge from server")
		}
		serverNonce := serverMsg[len(challengePrefix):]
		mac := hmac.New(md5.New, r.RPCKey)
		if _, err := mac.Write(serverNonce); err != nil {
			return err
		}
		digest := mac.Sum(nil)
		binary.BigEndian.PutUint32(hdr[:], uint32(len(digest)))
		if _, err := conn.Write(hdr[:]); err != nil {
			return err
		}
		if _, err := conn.Write(digest); err != nil {
			return err
		}
		if _, err := io.ReadFull(conn, hdr[:]); err != nil {
			return err
		}
		n = int(binary.BigEndian.Uint32(hdr[:]))
		if n > 256 {
			return fmt.Errorf("rpc message too large: %d bytes", n)
		}
		serverResp := make([]byte, n)
		if _, err := io.ReadFull(conn, serverResp); err != nil {
			return err
		}
		if string(serverResp) != "#WELCOME#" {
			return errors.New("digest sent was rejected")
		}
		// deliver_challenge: client sends its nonce, verifies server digest
		randBytes := make([]byte, 20)
		if _, err := rand.Read(randBytes); err != nil {
			return err
		}
		msg := append([]byte("#CHALLENGE#"), randBytes...)
		binary.BigEndian.PutUint32(hdr[:], uint32(len(msg)))
		if _, err := conn.Write(hdr[:]); err != nil {
			return err
		}
		if _, err := conn.Write(msg); err != nil {
			return err
		}
		if _, err := io.ReadFull(conn, hdr[:]); err != nil {
			return err
		}
		n = int(binary.BigEndian.Uint32(hdr[:]))
		if n > 256 {
			return fmt.Errorf("rpc message too large: %d bytes", n)
		}
		serverDigest := make([]byte, n)
		if _, err := io.ReadFull(conn, serverDigest); err != nil {
			return err
		}
		mac2 := hmac.New(md5.New, r.RPCKey)
		if _, err := mac2.Write(randBytes); err != nil {
			return err
		}
		expected := mac2.Sum(nil)
		if subtle.ConstantTimeCompare(serverDigest, expected) != 1 {
			binary.BigEndian.PutUint32(hdr[:], uint32(len("#FAILURE#")))
			_, _ = conn.Write(hdr[:])
			_, _ = conn.Write([]byte("#FAILURE#"))
			return errors.New("digest received was wrong")
		}
		binary.BigEndian.PutUint32(hdr[:], uint32(len("#WELCOME#")))
		if _, err := conn.Write(hdr[:]); err != nil {
			return err
		}
		if _, err := conn.Write([]byte("#WELCOME#")); err != nil {
			return err
		}
		return nil
	}(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
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
		if err := func() error {
			data, err := vendor.PickleDumps(map[string]any{"get": "interface_stats"})
			if err != nil {
				return err
			}
			return func() error {
				var hdr [4]byte
				binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
				if _, err := client.Write(hdr[:]); err != nil {
					return err
				}
				if len(data) > 0 {
					_, err := client.Write(data)
					return err
				}
				return nil
			}()
		}(); err != nil {
			Log("RPC request failed: "+err.Error(), LogError)
			return map[string]any{}
		}
		var resp map[string]any
		if err := func() error {
			data, err := func() ([]byte, error) {
				var hdr [4]byte
				if _, err := io.ReadFull(client, hdr[:]); err != nil {
					return nil, err
				}
				n := int(binary.BigEndian.Uint32(hdr[:]))
				buf := make([]byte, n)
				if _, err := io.ReadFull(client, buf); err != nil {
					return nil, err
				}
				return buf, nil
			}()
			if err != nil {
				return err
			}
			result, err := vendor.PickleLoads(data)
			if err != nil {
				return fmt.Errorf("pickle decode: %w", err)
			}
			return vendor.PickleAssign(result, &resp)
		}(); err != nil {
			Log("RPC response failed: "+err.Error(), LogError)
			return map[string]any{}
		}
		return resp
	}
	stats := map[string]any{}
	entries := make([]map[string]any, 0, len(Interfaces))

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
			"ifac_signature":              nil,
			"ifac_size":                   nil,
			"ifac_netname":                nil,
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
			"autoconnect_source":          nil,
			"clients":                     nil,
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
		} else if strings.EqualFold(ifc.Type, "AutoInterface") {
			entry["peers"] = nil
		}
		if pc := ifc.WeavePeerCount(); pc != nil {
			entry["peers"] = *pc
		} else if strings.EqualFold(ifc.Type, "WeaveInterface") {
			entry["peers"] = nil
		}
		if sid := ifc.WeaveSwitchID(); sid != nil {
			entry["switch_id"] = *sid
		} else if strings.EqualFold(ifc.Type, "WeaveInterface") {
			entry["switch_id"] = nil
		}
		if eid := ifc.WeaveEndpointID(); eid != nil {
			entry["endpoint_id"] = *eid
		} else if strings.EqualFold(ifc.Type, "WeaveInterface") {
			entry["endpoint_id"] = nil
		}
		if v := ifc.WeaveCPULoad(); v != nil {
			entry["cpu_load"] = *v
		}
		if v := ifc.WeaveMemLoad(); v != nil {
			entry["mem_load"] = *v
		}
		if via := ifc.WeaveViaSwitchID(); via != nil {
			entry["via_switch_id"] = *via
		} else if strings.EqualFold(ifc.Type, "WeaveInterfacePeer") {
			entry["via_switch_id"] = nil
		}
		if v := ifc.I2PConnectable(); v != nil {
			entry["i2p_connectable"] = *v
		}
		if v := ifc.I2PB32(); v != nil {
			entry["i2p_b32"] = *v
		} else if strings.EqualFold(ifc.Type, "I2PInterface") {
			entry["i2p_b32"] = nil
		}
		if v := ifc.I2PTunnelState(); v != nil {
			entry["tunnelstate"] = *v
		} else if strings.EqualFold(ifc.Type, "I2PInterfacePeer") {
			entry["tunnelstate"] = nil
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
			if p := ifc.RNodeBatteryPercent(); p != nil {
				entry["battery_percent"] = *p
			}
		}
		if strings.TrimSpace(ifc.AutoconnectSource) != "" {
			entry["autoconnect_source"] = ifc.AutoconnectSource
		}
		if ifc.AnnounceQueue != nil {
			entry["announce_queue"] = ifc.AnnounceQueueCount()
		}
		entries = append(entries, entry)
	}

	stats["interfaces"] = entries
	stats["rxb"] = TrafficRXB
	stats["txb"] = TrafficTXB
	stats["rxs"] = SpeedRX
	stats["txs"] = SpeedTX
	if TransportEnabled() && TransportIdentity != nil {
		stats["transport_id"] = TransportIdentity.Hash
		if NetworkIdentity != nil {
			stats["network_id"] = NetworkIdentity.Hash
		} else {
			stats["network_id"] = nil
		}
		stats["transport_uptime"] = time.Since(StartTime).Seconds()
		if ProbeDestinationEnabled() && ProbeDestination != nil {
			stats["probe_responder"] = ProbeDestination.Hash
		} else {
			stats["probe_responder"] = nil
		}
	}

	stats["rss"] = processRSSBytesPlatform()
	return stats
}

func (r *Reticulum) GetPathTable(maxHops *int) []map[string]any {
	if r.IsConnectedToSharedInstance {
		client, err := r.getRPCClient()
		if err != nil {
			Log("Could not contact shared instance for path table: "+err.Error(), LogError)
			return nil
		}
		defer client.Close()
		if err := func() error {
			data, err := vendor.PickleDumps(map[string]any{"get": "path_table", "max_hops": maxHops})
			if err != nil {
				return err
			}
			return func() error {
				var hdr [4]byte
				binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
				if _, err := client.Write(hdr[:]); err != nil {
					return err
				}
				if len(data) > 0 {
					_, err := client.Write(data)
					return err
				}
				return nil
			}()
		}(); err != nil {
			Log("RPC request for path table failed: "+err.Error(), LogError)
			return nil
		}
		var resp []map[string]any
		if err := func() error {
			data, err := func() ([]byte, error) {
				var hdr [4]byte
				if _, err := io.ReadFull(client, hdr[:]); err != nil {
					return nil, err
				}
				n := int(binary.BigEndian.Uint32(hdr[:]))
				buf := make([]byte, n)
				if _, err := io.ReadFull(client, buf); err != nil {
					return nil, err
				}
				return buf, nil
			}()
			if err != nil {
				return err
			}
			result, err := vendor.PickleLoads(data)
			if err != nil {
				return fmt.Errorf("pickle decode: %w", err)
			}
			return vendor.PickleAssign(result, &resp)
		}(); err != nil {
			Log("RPC response for path table failed: "+err.Error(), LogError)
			return nil
		}
		return resp
	}
	pathTableMu.RLock()
	defer pathTableMu.RUnlock()
	table := make([]map[string]any, 0, len(pathTable))
	for key, entry := range pathTable {
		if entry == nil {
			continue
		}
		if maxHops != nil && entry.Hops > *maxHops {
			continue
		}
		hashCopy := make([]byte, truncatedHashBytes)
		copy(hashCopy, key[:])
		via := append([]byte(nil), entry.NextHop...)
		var expires float64
		if !entry.ExpiresAt.IsZero() {
			expires = float64(entry.ExpiresAt.UnixNano()) / 1e9
		}
		timestamp := float64(entry.Timestamp.UnixNano()) / 1e9
		table = append(table, map[string]any{
			"hash":      hashCopy,
			"timestamp": timestamp,
			"via":       via,
			"hops":      entry.Hops,
			"expires":   expires,
			"interface": fmt.Sprint(entry.RecvInterface),
		})
	}
	return table
}

func (r *Reticulum) GetRateTable() []map[string]any {
	if r.IsConnectedToSharedInstance {
		client, err := r.getRPCClient()
		if err != nil {
			Log("Could not contact shared instance for rate table: "+err.Error(), LogError)
			return nil
		}
		defer client.Close()
		if err := func() error {
			data, err := vendor.PickleDumps(map[string]any{"get": "rate_table"})
			if err != nil {
				return err
			}
			return func() error {
				var hdr [4]byte
				binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
				if _, err := client.Write(hdr[:]); err != nil {
					return err
				}
				if len(data) > 0 {
					_, err := client.Write(data)
					return err
				}
				return nil
			}()
		}(); err != nil {
			Log("RPC request for rate table failed: "+err.Error(), LogError)
			return nil
		}
		var resp []map[string]any
		if err := func() error {
			data, err := func() ([]byte, error) {
				var hdr [4]byte
				if _, err := io.ReadFull(client, hdr[:]); err != nil {
					return nil, err
				}
				n := int(binary.BigEndian.Uint32(hdr[:]))
				buf := make([]byte, n)
				if _, err := io.ReadFull(client, buf); err != nil {
					return nil, err
				}
				return buf, nil
			}()
			if err != nil {
				return err
			}
			result, err := vendor.PickleLoads(data)
			if err != nil {
				return fmt.Errorf("pickle decode: %w", err)
			}
			return vendor.PickleAssign(result, &resp)
		}(); err != nil {
			Log("RPC response for rate table failed: "+err.Error(), LogError)
			return nil
		}
		return resp
	}
	announceRateMu.RLock()
	defer announceRateMu.RUnlock()
	table := make([]map[string]any, 0, len(announceRateOrder))
	for _, key := range announceRateOrder {
		entry := announceRateTable[key]
		if entry == nil {
			continue
		}
		hashCopy := make([]byte, truncatedHashBytes)
		copy(hashCopy, key[:])
		timestamps := make([]float64, len(entry.Timestamps))
		for i, ts := range entry.Timestamps {
			timestamps[i] = float64(ts.UnixNano()) / 1e9
		}
		table = append(table, map[string]any{
			"hash":            hashCopy,
			"last":            float64(entry.Last.UnixNano()) / 1e9,
			"rate_violations": entry.RateViolations,
			"blocked_until":   float64(entry.BlockedUntil.UnixNano()) / 1e9,
			"timestamps":      timestamps,
		})
	}
	return table
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
		if err := func() error {
			data, err := vendor.PickleDumps(req)
			if err != nil {
				return err
			}
			return func() error {
				var hdr [4]byte
				binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
				if _, err := client.Write(hdr[:]); err != nil {
					return err
				}
				if len(data) > 0 {
					_, err := client.Write(data)
					return err
				}
				return nil
			}()
		}(); err != nil {
			Log("RPC request to drop path failed: "+err.Error(), LogError)
			return false
		}
		var resp bool
		if err := func() error {
			data, err := func() ([]byte, error) {
				var hdr [4]byte
				if _, err := io.ReadFull(client, hdr[:]); err != nil {
					return nil, err
				}
				n := int(binary.BigEndian.Uint32(hdr[:]))
				buf := make([]byte, n)
				if _, err := io.ReadFull(client, buf); err != nil {
					return nil, err
				}
				return buf, nil
			}()
			if err != nil {
				return err
			}
			result, err := vendor.PickleLoads(data)
			if err != nil {
				return fmt.Errorf("pickle decode: %w", err)
			}
			return vendor.PickleAssign(result, &resp)
		}(); err != nil {
			Log("RPC response to drop path failed: "+err.Error(), LogError)
			return false
		}
		return resp
	}
	return ExpirePath(destination)
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
		if err := func() error {
			data, err := vendor.PickleDumps(req)
			if err != nil {
				return err
			}
			return func() error {
				var hdr [4]byte
				binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
				if _, err := client.Write(hdr[:]); err != nil {
					return err
				}
				if len(data) > 0 {
					_, err := client.Write(data)
					return err
				}
				return nil
			}()
		}(); err != nil {
			Log("RPC request to drop via failed: "+err.Error(), LogError)
			return 0
		}
		var resp int
		if err := func() error {
			data, err := func() ([]byte, error) {
				var hdr [4]byte
				if _, err := io.ReadFull(client, hdr[:]); err != nil {
					return nil, err
				}
				n := int(binary.BigEndian.Uint32(hdr[:]))
				buf := make([]byte, n)
				if _, err := io.ReadFull(client, buf); err != nil {
					return nil, err
				}
				return buf, nil
			}()
			if err != nil {
				return err
			}
			result, err := vendor.PickleLoads(data)
			if err != nil {
				return fmt.Errorf("pickle decode: %w", err)
			}
			return vendor.PickleAssign(result, &resp)
		}(); err != nil {
			Log("RPC response to drop via failed: "+err.Error(), LogError)
			return 0
		}
		return resp
	}
	if len(transportHash) == 0 {
		return 0
	}
	pathTableMu.Lock()
	defer pathTableMu.Unlock()
	droppedCount := 0
	for _, entry := range pathTable {
		if entry == nil {
			continue
		}
		if len(entry.NextHop) == 0 {
			continue
		}
		if bytes.Equal(entry.NextHop, transportHash) {
			entry.Timestamp = time.Unix(0, 0)
			droppedCount++
		}
	}
	if droppedCount > 0 {
		TablesLastCulled = time.Time{}
	}
	return droppedCount
}

func (r *Reticulum) DropAnnounceQueues() any {
	if r.IsConnectedToSharedInstance {
		client, err := r.getRPCClient()
		if err != nil {
			Log("Could not contact shared instance to drop announce queues: "+err.Error(), LogError)
			return nil
		}
		defer client.Close()
		if err := func() error {
			data, err := vendor.PickleDumps(map[string]any{"drop": "announce_queues"})
			if err != nil {
				return err
			}
			return func() error {
				var hdr [4]byte
				binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
				if _, err := client.Write(hdr[:]); err != nil {
					return err
				}
				if len(data) > 0 {
					_, err := client.Write(data)
					return err
				}
				return nil
			}()
		}(); err != nil {
			Log("RPC request to drop announce queues failed: "+err.Error(), LogError)
			return nil
		}
		var resp any
		if err := func() error {
			data, err := func() ([]byte, error) {
				var hdr [4]byte
				if _, err := io.ReadFull(client, hdr[:]); err != nil {
					return nil, err
				}
				n := int(binary.BigEndian.Uint32(hdr[:]))
				buf := make([]byte, n)
				if _, err := io.ReadFull(client, buf); err != nil {
					return nil, err
				}
				return buf, nil
			}()
			if err != nil {
				return err
			}
			result, err := vendor.PickleLoads(data)
			if err != nil {
				return fmt.Errorf("pickle decode: %w", err)
			}
			return vendor.PickleAssign(result, &resp)
		}(); err != nil {
			Log("RPC response to drop announce queues failed: "+err.Error(), LogError)
			return nil
		}
		return resp
	}
	DropAnnounceQueues()
	return nil
}

func (r *Reticulum) GetNextHop(destination []byte) []byte {
	if r.IsConnectedToSharedInstance {
		client, err := r.getRPCClient()
		if err != nil {
			Log("Could not contact shared instance for next hop: "+err.Error(), LogError)
			return nil
		}
		defer client.Close()
		if err := func() error {
			data, err := vendor.PickleDumps(map[string]any{"get": "next_hop", "destination_hash": destination})
			if err != nil {
				return err
			}
			return func() error {
				var hdr [4]byte
				binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
				if _, err := client.Write(hdr[:]); err != nil {
					return err
				}
				if len(data) > 0 {
					_, err := client.Write(data)
					return err
				}
				return nil
			}()
		}(); err != nil {
			Log("RPC request for next hop failed: "+err.Error(), LogError)
			return nil
		}
		var resp []byte
		if err := func() error {
			data, err := func() ([]byte, error) {
				var hdr [4]byte
				if _, err := io.ReadFull(client, hdr[:]); err != nil {
					return nil, err
				}
				n := int(binary.BigEndian.Uint32(hdr[:]))
				buf := make([]byte, n)
				if _, err := io.ReadFull(client, buf); err != nil {
					return nil, err
				}
				return buf, nil
			}()
			if err != nil {
				return err
			}
			result, err := vendor.PickleLoads(data)
			if err != nil {
				return fmt.Errorf("pickle decode: %w", err)
			}
			return vendor.PickleAssign(result, &resp)
		}(); err != nil {
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
		if err := func() error {
			data, err := vendor.PickleDumps(map[string]any{"get": "next_hop_if_name", "destination_hash": destination})
			if err != nil {
				return err
			}
			return func() error {
				var hdr [4]byte
				binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
				if _, err := client.Write(hdr[:]); err != nil {
					return err
				}
				if len(data) > 0 {
					_, err := client.Write(data)
					return err
				}
				return nil
			}()
		}(); err != nil {
			Log("RPC request for next hop interface failed: "+err.Error(), LogError)
			return ""
		}
		var resp string
		if err := func() error {
			data, err := func() ([]byte, error) {
				var hdr [4]byte
				if _, err := io.ReadFull(client, hdr[:]); err != nil {
					return nil, err
				}
				n := int(binary.BigEndian.Uint32(hdr[:]))
				buf := make([]byte, n)
				if _, err := io.ReadFull(client, buf); err != nil {
					return nil, err
				}
				return buf, nil
			}()
			if err != nil {
				return err
			}
			result, err := vendor.PickleLoads(data)
			if err != nil {
				return fmt.Errorf("pickle decode: %w", err)
			}
			return vendor.PickleAssign(result, &resp)
		}(); err != nil {
			Log("RPC response for next hop interface failed: "+err.Error(), LogError)
			return ""
		}
		return resp
	}
	if ifc := NextHopInterface(destination); ifc != nil {
		return fmt.Sprint(ifc)
	}
	return "None"
}

func (r *Reticulum) GetFirstHopTimeout(destination []byte) float64 {
	if r.IsConnectedToSharedInstance {
		client, err := r.getRPCClient()
		if err != nil {
			Log("Could not contact shared instance for first hop timeout: "+err.Error(), LogError)
			return DEFAULT_PER_HOP_TIMEOUT
		}
		defer client.Close()
		if err := func() error {
			data, err := vendor.PickleDumps(map[string]any{"get": "first_hop_timeout", "destination_hash": destination})
			if err != nil {
				return err
			}
			return func() error {
				var hdr [4]byte
				binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
				if _, err := client.Write(hdr[:]); err != nil {
					return err
				}
				if len(data) > 0 {
					_, err := client.Write(data)
					return err
				}
				return nil
			}()
		}(); err != nil {
			Log("RPC request for first hop timeout failed: "+err.Error(), LogError)
			return DEFAULT_PER_HOP_TIMEOUT
		}
		var resp float64
		if err := func() error {
			data, err := func() ([]byte, error) {
				var hdr [4]byte
				if _, err := io.ReadFull(client, hdr[:]); err != nil {
					return nil, err
				}
				n := int(binary.BigEndian.Uint32(hdr[:]))
				buf := make([]byte, n)
				if _, err := io.ReadFull(client, buf); err != nil {
					return nil, err
				}
				return buf, nil
			}()
			if err != nil {
				return err
			}
			result, err := vendor.PickleLoads(data)
			if err != nil {
				return fmt.Errorf("pickle decode: %w", err)
			}
			return vendor.PickleAssign(result, &resp)
		}(); err != nil {
			Log("RPC response for first hop timeout failed: "+err.Error(), LogError)
			return DEFAULT_PER_HOP_TIMEOUT
		}
		sharedInstanceMu.RLock()
		bitrate := sharedInstanceForcedBitrate
		sharedInstanceMu.RUnlock()
		if bitrate > 0 {
			simulatedLatency := (float64(MTU) * 8.0) / float64(bitrate)
			Logf(LogDebug, "Adding simulated latency of %s to first hop timeout", PrettyTime(simulatedLatency, false, false))
			resp += simulatedLatency
		}
		return resp
	}
	return FirstHopTimeout(destination).Seconds()
}

func (r *Reticulum) GetLinkCount() int {
	if r.IsConnectedToSharedInstance {
		client, err := r.getRPCClient()
		if err != nil {
			Log("Could not contact shared instance for link count: "+err.Error(), LogError)
			return 0
		}
		defer client.Close()
		if err := func() error {
			data, err := vendor.PickleDumps(map[string]any{"get": "link_count"})
			if err != nil {
				return err
			}
			return func() error {
				var hdr [4]byte
				binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
				if _, err := client.Write(hdr[:]); err != nil {
					return err
				}
				if len(data) > 0 {
					_, err := client.Write(data)
					return err
				}
				return nil
			}()
		}(); err != nil {
			Log("RPC request for link count failed: "+err.Error(), LogError)
			return 0
		}
		var resp int
		if err := func() error {
			data, err := func() ([]byte, error) {
				var hdr [4]byte
				if _, err := io.ReadFull(client, hdr[:]); err != nil {
					return nil, err
				}
				n := int(binary.BigEndian.Uint32(hdr[:]))
				buf := make([]byte, n)
				if _, err := io.ReadFull(client, buf); err != nil {
					return nil, err
				}
				return buf, nil
			}()
			if err != nil {
				return err
			}
			result, err := vendor.PickleLoads(data)
			if err != nil {
				return fmt.Errorf("pickle decode: %w", err)
			}
			return vendor.PickleAssign(result, &resp)
		}(); err != nil {
			Log("RPC response for link count failed: "+err.Error(), LogError)
			return 0
		}
		return resp
	}
	linkMu.Lock()
	count := len(ActiveLinks)
	linkMu.Unlock()
	return count
}

func (r *Reticulum) GetPacketRSSI(packetHash []byte) *float64 {
	if r.IsConnectedToSharedInstance {
		client, err := r.getRPCClient()
		if err != nil {
			Log("Could not contact shared instance for packet_rssi: "+err.Error(), LogError)
			return nil
		}
		defer client.Close()
		if err := func() error {
			data, err := vendor.PickleDumps(map[string]any{"get": "packet_rssi", "packet_hash": packetHash})
			if err != nil {
				return err
			}
			return func() error {
				var hdr [4]byte
				binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
				if _, err := client.Write(hdr[:]); err != nil {
					return err
				}
				if len(data) > 0 {
					_, err := client.Write(data)
					return err
				}
				return nil
			}()
		}(); err != nil {
			Log("RPC request for packet_rssi failed: "+err.Error(), LogError)
			return nil
		}
		var resp *float64
		if err := func() error {
			data, err := func() ([]byte, error) {
				var hdr [4]byte
				if _, err := io.ReadFull(client, hdr[:]); err != nil {
					return nil, err
				}
				n := int(binary.BigEndian.Uint32(hdr[:]))
				buf := make([]byte, n)
				if _, err := io.ReadFull(client, buf); err != nil {
					return nil, err
				}
				return buf, nil
			}()
			if err != nil {
				return err
			}
			result, err := vendor.PickleLoads(data)
			if err != nil {
				return fmt.Errorf("pickle decode: %w", err)
			}
			return vendor.PickleAssign(result, &resp)
		}(); err != nil {
			Log("RPC response for packet_rssi failed: "+err.Error(), LogError)
			return nil
		}
		return resp
	}
	localStatsMu.RLock()
	v, ok := localRSSICache[string(packetHash)]
	localStatsMu.RUnlock()
	if !ok {
		return nil
	}
	return &v
}

func (r *Reticulum) GetPacketSNR(packetHash []byte) *float64 {
	if r.IsConnectedToSharedInstance {
		client, err := r.getRPCClient()
		if err != nil {
			Log("Could not contact shared instance for packet_snr: "+err.Error(), LogError)
			return nil
		}
		defer client.Close()
		if err := func() error {
			data, err := vendor.PickleDumps(map[string]any{"get": "packet_snr", "packet_hash": packetHash})
			if err != nil {
				return err
			}
			return func() error {
				var hdr [4]byte
				binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
				if _, err := client.Write(hdr[:]); err != nil {
					return err
				}
				if len(data) > 0 {
					_, err := client.Write(data)
					return err
				}
				return nil
			}()
		}(); err != nil {
			Log("RPC request for packet_snr failed: "+err.Error(), LogError)
			return nil
		}
		var resp *float64
		if err := func() error {
			data, err := func() ([]byte, error) {
				var hdr [4]byte
				if _, err := io.ReadFull(client, hdr[:]); err != nil {
					return nil, err
				}
				n := int(binary.BigEndian.Uint32(hdr[:]))
				buf := make([]byte, n)
				if _, err := io.ReadFull(client, buf); err != nil {
					return nil, err
				}
				return buf, nil
			}()
			if err != nil {
				return err
			}
			result, err := vendor.PickleLoads(data)
			if err != nil {
				return fmt.Errorf("pickle decode: %w", err)
			}
			return vendor.PickleAssign(result, &resp)
		}(); err != nil {
			Log("RPC response for packet_snr failed: "+err.Error(), LogError)
			return nil
		}
		return resp
	}
	localStatsMu.RLock()
	v, ok := localSNRCache[string(packetHash)]
	localStatsMu.RUnlock()
	if !ok {
		return nil
	}
	return &v
}

func (r *Reticulum) GetPacketQ(packetHash []byte) *float64 {
	if r.IsConnectedToSharedInstance {
		client, err := r.getRPCClient()
		if err != nil {
			Log("Could not contact shared instance for packet_q: "+err.Error(), LogError)
			return nil
		}
		defer client.Close()
		if err := func() error {
			data, err := vendor.PickleDumps(map[string]any{"get": "packet_q", "packet_hash": packetHash})
			if err != nil {
				return err
			}
			return func() error {
				var hdr [4]byte
				binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
				if _, err := client.Write(hdr[:]); err != nil {
					return err
				}
				if len(data) > 0 {
					_, err := client.Write(data)
					return err
				}
				return nil
			}()
		}(); err != nil {
			Log("RPC request for packet_q failed: "+err.Error(), LogError)
			return nil
		}
		var resp *float64
		if err := func() error {
			data, err := func() ([]byte, error) {
				var hdr [4]byte
				if _, err := io.ReadFull(client, hdr[:]); err != nil {
					return nil, err
				}
				n := int(binary.BigEndian.Uint32(hdr[:]))
				buf := make([]byte, n)
				if _, err := io.ReadFull(client, buf); err != nil {
					return nil, err
				}
				return buf, nil
			}()
			if err != nil {
				return err
			}
			result, err := vendor.PickleLoads(data)
			if err != nil {
				return fmt.Errorf("pickle decode: %w", err)
			}
			return vendor.PickleAssign(result, &resp)
		}(); err != nil {
			Log("RPC response for packet_q failed: "+err.Error(), LogError)
			return nil
		}
		return resp
	}
	localStatsMu.RLock()
	v, ok := localQCache[string(packetHash)]
	localStatsMu.RUnlock()
	if !ok {
		return nil
	}
	return &v
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

func RequiredDiscoveryValue() *int {
	return requiredDiscoveryValue
}

func InterfaceDiscoveryEnabled() bool {
	return discoveryEnabled
}

func InterfaceDiscoverySources() [][]byte {
	return interfaceDiscoverySources
}

func PublishBlackholeEnabled() bool {
	return publishBlackholeEnabled
}

func BlackholeSources() [][]byte {
	return blackholeSources
}

func ShouldAutoconnectDiscoveredInterfaces() bool {
	return autoconnectDiscoveredInterfaces > 0
}

func MaxAutoconnectedInterfaces() int {
	return autoconnectDiscoveredInterfaces
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
	"# You can configure whether Reticulum should discover",
	"# available interfaces from other Transport Instances over",
	"# the network. If this option is enabled, Reticulum will",
	"# collect interface information discovered from the network.",
	"",
	"# discover_interfaces = No",
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
	"# If you're connecting to a large external network, you",
	"# can use one or more external blackhole list to block",
	"# spammy and excessive announces onto your network. This",
	"# funtionality is especially useful if you're hosting public",
	"# entrypoints or gateways. The list source below provides a",
	"# functional example, but better, more timely maintained",
	"# lists probably exist in the community.",
	"",
	"# blackhole_sources = 521c87a83afb8f29e4455e77930b973b",
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
