package rns

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	Cryptography "github.com/svanichkin/go-reticulum/rns/cryptography"
	ifaces "github.com/svanichkin/go-reticulum/rns/interfaces"
	umsgpack "github.com/svanichkin/go-reticulum/rns/vendor"
)

const (
	Broadcast       = 0x00
	TransportDirect = 0x01
	Relay           = 0x02
	Tunnel          = 0x03
)

// Python parity: Transport reachability values.
const (
	ReachabilityUnreachable = 0x00
	ReachabilityDirect      = 0x01
	ReachabilityTransport   = 0x02
)

// Python parity: Transport path state values.
const (
	TransportStateUnknown      = 0x00
	TransportStateUnresponsive = 0x01
	TransportStateResponsive   = 0x02
)

// TransportAppName mirrors Transport.APP_NAME in the Python implementation.
const TransportAppName = "rnstransport"

const (
	MaxReceipts                   = 1024
	TransportPathRequestTimeout   = 15.0
	DefaultDiscoveryRequiredValue = 14
	// DestinationTimeout mirrors Python Transport.DESTINATION_TIMEOUT.
	DestinationTimeout      = 7 * 24 * time.Hour
	PathfinderMaxHops       = 128
	pathfinderRetryLimit    = 1
	pathfinderRetryGrace    = 5 * time.Second
	pathfinderRandomWindow  = 500 * time.Millisecond
	localRebroadcastsMax    = 2
	pathRequestGrace        = 400 * time.Millisecond
	pathRequestRoamingGrace = 1500 * time.Millisecond
	apPathTime              = 24 * time.Hour
	roamingPathTime         = 6 * time.Hour
	maxRandomBlobs          = 64
	persistRandomBlobs      = 32
	maxRateTimestamps       = 16
	mgmtAnnounceInterval    = 2 * time.Hour
	initialMgmtAnnounceWait = 15 * time.Second
	blackholeCheckInterval  = 60 * time.Second
)

var (
	announceQueueTTL = time.Duration(QUEUED_ANNOUNCE_LIFE) * time.Second
)

type hashKey [truncatedHashBytes]byte

type transportBackgroundWorker interface {
	Start()
}

type noopTransportWorker struct{}

func (noopTransportWorker) Start() {}

// AnnounceHandler mirrors the Python announce handler callback.
// If AspectFilter returns a non-empty string, only announces whose NAME_HASH
// matches the filter will be delivered.
type AnnounceHandler interface {
	AspectFilter() any
	ReceivedAnnounce(destinationHash []byte, announcedIdentity *Identity, appData []byte)
}

// AnnounceHandlerWithPacketHash receives the announce packet hash in addition
// to the legacy callback parameters, matching the Python 4-argument callback.
type AnnounceHandlerWithPacketHash interface {
	AnnounceHandler
	ReceivedAnnounceWithPacketHash(destinationHash []byte, announcedIdentity *Identity, appData []byte, announcePacketHash []byte)
}

// AnnounceHandlerWithPacketInfo receives both announce_packet_hash and
// is_path_response, matching the Python 5-argument callback.
type AnnounceHandlerWithPacketInfo interface {
	AnnounceHandler
	ReceivedAnnounceWithPacketInfo(destinationHash []byte, announcedIdentity *Identity, appData []byte, announcePacketHash []byte, isPathResponse bool)
}

// PathResponseAnnounceHandler opts a handler into receiving PATH_RESPONSE
// announces, mirroring Python's receive_path_responses flag.
type PathResponseAnnounceHandler interface {
	ReceivePathResponses() bool
}

var (
	Owner *Reticulum

	// TransportIdentity is the identity used for transport control destinations
	// like remote management. It is initialised in Start() if missing.
	TransportIdentity *Identity
	NetworkIdentity   *Identity

	ProbeDestination *Destination

	Interfaces      []*Interface
	Destinations    []*Destination
	PendingLinks    []*Link
	ActiveLinks     []*Link
	PacketHashSet   = make(map[hashKey]struct{})
	PacketHashSet2  = make(map[hashKey]struct{})
	HashlistMaxSize = 1_000_000

	Receipts []*PacketReceipt

	LocalClientInterfaces []*Interface
	interfacesMu          sync.RWMutex

	// jobsMu serialises Jobs() (write-lock) against Inbound/Outbound
	// (read-lock).  This replaces the old JobsLocked/JobsRunning booleans
	// which were bare flags with no memory barriers — a data race that
	// caused silent deadlocks when multiple goroutines called Outbound
	// concurrently (e.g. teardown packets from watchdog goroutines).
	jobsMu                sync.RWMutex
	JobInterval           = 250 * time.Millisecond
	LinksLastChecked      time.Time
	LinksCheckInt         = time.Second
	ReceiptsLast          time.Time
	ReceiptsCheckInt      = time.Second
	AnnLast               time.Time
	AnnCheckInt           = time.Second
	InterfaceLastJobs     time.Time
	InterfaceJobsInterval = 5 * time.Second

	pathTable         = make(map[hashKey]*PathEntry)
	pathTableMu       sync.RWMutex
	packetHashMu      sync.RWMutex
	pathRequestMu     sync.Mutex
	linkMu            sync.Mutex
	linkTableMu       sync.RWMutex
	receiptsMu        sync.Mutex
	announceMu        sync.Mutex
	inboundAnnounceMu sync.Mutex
	lastPathRequest   = make(map[hashKey]time.Time)
	announceTable     = make(map[hashKey]*announceEntry)
	linkTable         = make(map[hashKey]*linkEntry)

	announceRateTable = make(map[hashKey]*announceRateEntry)
	announceRateOrder = make([]hashKey, 0)
	announceRateMu    sync.RWMutex

	TrafficRXB uint64
	TrafficTXB uint64
	SpeedRX    float64
	SpeedTX    float64

	StartTime time.Time

	sharedInstanceForcedBitrate int
	sharedInstanceMu            sync.RWMutex

	reverseTableMu      sync.Mutex
	reverseTable        = make(map[hashKey]*reverseEntry)
	heldAnnounces       = make(map[hashKey]*heldAnnounce)
	LastCacheCleaned    time.Time
	TablesLastCulled    time.Time
	lastTablesPersisted time.Time

	controlHashesMu sync.RWMutex
	controlHashes   = make(map[string]struct{})

	remoteManagementAllowedMu sync.RWMutex
	remoteManagementAllowed   = make(map[string][]byte)
	remoteManagementDest      *Destination
	instanceDestination       *Destination
	networkDestination        *Destination
	pathRequestDest           *Destination
	tunnelSynthesizeDest      *Destination
	controlDestinations       []*Destination
	mgmtDestinations          []*Destination
	mgmtHashes                [][]byte
	remoteManagementActive    bool
	blackholeDestination      *Destination
	LastMgmtAnnounce          time.Time
	discoveryRequiredValue    = DefaultDiscoveryRequiredValue
	blackholeLastChecked      time.Time
	interfaceAnnouncer        transportBackgroundWorker
	discoveryHandler          any
	blackholeUpdater          transportBackgroundWorker
	interfaceAnnouncerFactory = func() transportBackgroundWorker { return noopTransportWorker{} }
	discoveryHandlerFactory   = func(requiredValue int, discoverInterfaces bool) any {
		return struct {
			RequiredValue      int
			DiscoverInterfaces bool
		}{RequiredValue: requiredValue, DiscoverInterfaces: discoverInterfaces}
	}
	blackholeUpdaterFactory = func() transportBackgroundWorker { return noopTransportWorker{} }

	tunnelsMu sync.RWMutex
	tunnels   = make(map[string]*tunnelEntry)

	destinationsMu sync.Mutex

	pathStatesMu sync.RWMutex
	pathStates   = make(map[hashKey]uint8)

	packetHashlistSaveMu sync.Mutex
	savingPacketHashlist bool

	pathTableSaveMu sync.Mutex
	savingPathTable bool

	tunnelTableSaveMu sync.Mutex
	savingTunnelTable bool

	discoveryTagsMu    sync.Mutex
	discoveryPRTags    = make(map[string]struct{})
	discoveryPRTagFIFO []string
	maxPRTags          = 32000

	discoveryPathRequestsMu sync.Mutex
	discoveryPathRequests   = make(map[hashKey]*discoveryPathRequest)

	blackholeMu          sync.RWMutex
	blackholedIdentities = make(map[hashKey]*blackholeEntry)

	pendingLocalPathRequestsMu sync.Mutex
	pendingLocalPathRequests   = make(map[hashKey]*Interface)
	pendingPRsLastChecked      time.Time
	pendingPRsCheckInterval    = 30 * time.Second

	localStatsMu   sync.RWMutex
	localRSSICache = make(map[string]float64)
	localSNRCache  = make(map[string]float64)
	localQCache    = make(map[string]float64)
	localRSSIFIFO  []string
	localSNRFIFO   []string
	localQFIFO     []string
	localStatsMax  = 512

	announceHandlersMu sync.RWMutex
	announceHandlers   []any

	// Store additional transport tables here as well:
	// AnnounceTable, PathTable, ReverseTable, LinkTable, DiscoveryPathRequests, ...
)

// RegisterAnnounceHandler registers an announce handler with the transport.
// Mirrors Python: checks received_announce (callable) first, then aspect_filter (attribute).
func RegisterAnnounceHandler(handler any) {
	if handler == nil {
		return
	}
	val := reflect.ValueOf(handler)
	if val.Kind() == reflect.Pointer && val.IsNil() {
		return
	}
	if !val.MethodByName("ReceivedAnnounce").IsValid() {
		return
	}
	if !val.MethodByName("AspectFilter").IsValid() {
		return
	}
	announceHandlersMu.Lock()
	announceHandlers = append(announceHandlers, handler)
	announceHandlersMu.Unlock()
}

// DeregisterAnnounceHandler removes a previously registered announce handler.
func DeregisterAnnounceHandler(handler any) {
	if handler == nil {
		return
	}
	announceHandlersMu.Lock()
	filtered := make([]any, 0, len(announceHandlers))
	for _, existing := range announceHandlers {
		if existing == handler {
			continue
		}
		filtered = append(filtered, existing)
	}
	announceHandlers = filtered
	announceHandlersMu.Unlock()
	runtime.GC()
}

// SetNetworkIdentity mirrors Python Transport.set_network_identity().
// The identity is only set once, matching the Python behaviour.
func SetNetworkIdentity(identity *Identity) {
	if identity == nil || NetworkIdentity != nil {
		return
	}
	NetworkIdentity = identity
}

// HasNetworkIdentity mirrors Python Transport.has_network_identity().
func HasNetworkIdentity() bool {
	return NetworkIdentity != nil
}

// EnableDiscovery mirrors Python Transport.enable_discovery().
func EnableDiscovery() {
	if interfaceAnnouncer != nil {
		return
	}
	interfaceAnnouncer = interfaceAnnouncerFactory()
	if interfaceAnnouncer != nil {
		interfaceAnnouncer.Start()
	}
}

// DiscoverInterfaces mirrors Python Transport.discover_interfaces().
func DiscoverInterfaces() {
	if discoveryHandler != nil {
		return
	}
	discoveryHandler = discoveryHandlerFactory(discoveryRequiredValue, true)
}

// EnableBlackholeUpdater mirrors Python Transport.enable_blackhole_updater().
func EnableBlackholeUpdater() {
	if blackholeUpdater != nil {
		return
	}
	blackholeUpdater = blackholeUpdaterFactory()
	if blackholeUpdater != nil {
		blackholeUpdater.Start()
	}
}

// RegisterDestination registers a destination with the transport.
// The Python implementation maintains multiple internal maps; for now we keep
// a simple list and ensure uniqueness.
func RegisterDestination(d *Destination) error {
	if d == nil {
		return nil
	}
	d.MTU = MTU
	if d.Direction != DestinationIN {
		return nil
	}

	destinationsMu.Lock()
	for _, existing := range Destinations {
		if existing != nil && len(existing.Hash) > 0 && bytes.Equal(existing.Hash, d.Hash) {
			destinationsMu.Unlock()
			return errors.New("Attempt to register an already registered destination.")
		}
	}
	Destinations = append(Destinations, d)
	destinationsMu.Unlock()
	return nil
}

// DeregisterDestination removes a destination from the transport registry.
// Mirrors Python Transport.deregister_destination().
func DeregisterDestination(d *Destination) {
	if d == nil {
		return
	}
	destinationsMu.Lock()
	for i, existing := range Destinations {
		if existing == d {
			Destinations = append(Destinations[:i], Destinations[i+1:]...)
			break
		}
	}
	destinationsMu.Unlock()
}

func registerLink(l *Link) {
	if l == nil {
		return
	}
	Log(fmt.Sprintf("Registering link %v", l), LogExtreme)
	linkMu.Lock()
	defer linkMu.Unlock()
	if l.Initiator {
		PendingLinks = append(PendingLinks, l)
	} else {
		ActiveLinks = append(ActiveLinks, l)
	}
}

func activateLink(l *Link) {
	if l == nil {
		return
	}
	Log(fmt.Sprintf("Activating link %v", l), LogExtreme)
	linkMu.Lock()
	defer linkMu.Unlock()
	found := false
	for _, existing := range PendingLinks {
		if existing == l {
			found = true
			break
		}
	}
	if !found {
		Log("Attempted to activate a link that was not in the pending table", LogError)
		return
	}
	if l.Status != LinkActive {
		Log( // Python parity: raise IOError rather than panic for invalid link state.
			fmt.Sprintf("Invalid link state for link activation: %d", l.Status), LogError)
		return
	}
	for i, existing := range PendingLinks {
		if existing == l {
			PendingLinks = append(PendingLinks[:i], PendingLinks[i+1:]...)
			break
		}
	}
	ActiveLinks = append(ActiveLinks, l)
	l.Status = LinkActive
}

// DetachInterfaces mirrors Python Transport.detach_interfaces().
func DetachInterfaces() {
	detachableInterfaces := make([]*Interface, 0, len(Interfaces)+len(LocalClientInterfaces))
	for _, ifc := range Interfaces {
		if ifc != nil && !ifc.Detached {
			detachableInterfaces = append(detachableInterfaces, ifc)
		}
	}
	for _, ifc := range LocalClientInterfaces {
		if ifc != nil && !ifc.Detached {
			detachableInterfaces = append(detachableInterfaces, ifc)
		}
	}

	sharedInstanceMaster := (*Interface)(nil)
	localInterfaces := make([]*Interface, 0, len(detachableInterfaces))
	detachThreads := make([]chan struct{}, 0, len(detachableInterfaces))

	Log("Detaching interfaces", LogDebug)
	for _, ifc := range detachableInterfaces {
		if ifc == nil {
			continue
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					Log(fmt.Sprintf("An error occurred while detaching %s. The contained exception was: %v", ifc, r), LogError)
				}
			}()
			switch {
			case strings.EqualFold(ifc.Type, "LocalInterface") && ifc.Parent == nil && !ifc.LocalIsSharedClient:
				sharedInstanceMaster = ifc
			case strings.EqualFold(ifc.Type, "LocalInterface") && (ifc.Parent != nil || ifc.LocalIsSharedClient):
				localInterfaces = append(localInterfaces, ifc)
			default:
				wgDone := make(chan struct{})
				detachThreads = append(detachThreads, wgDone)
				go func(interfaceToDetach *Interface, done chan struct{}) {
					defer close(done)
					Log(fmt.Sprintf("Detaching %s", interfaceToDetach), LogExtreme)
					interfaceToDetach.Detach()
				}(ifc, wgDone)
			}
		}()
	}

	for _, done := range detachThreads {
		<-done
	}

	Log("Detaching local clients", LogDebug)
	for _, li := range localInterfaces {
		if li != nil {
			li.Detach()
		}
	}

	Log("Detaching shared instance", LogDebug)
	if sharedInstanceMaster != nil {
		sharedInstanceMaster.Detach()
	}

	ifaces.DeregisterListeners()

	Log("All interfaces detached", LogDebug)
}

const (
	pathExpiration         = 7 * 24 * time.Hour
	pathRequestMinInterval = 20 * time.Second
	reverseTimeout         = 8 * time.Minute
	tablesCullInterval     = 5 * time.Second
	cacheCleanInterval     = 5 * time.Minute
	tablesPersistInterval  = 12 * time.Hour
	linkTimeout            = 900 * time.Second // Python: Link.STALE_TIME * 1.25
)

type PathEntry struct {
	NextHop       []byte
	RecvInterface *Interface
	Hops          int
	Timestamp     time.Time
	ExpiresAt     time.Time
	RandomBlobs   [][]byte
	AnnounceAt    uint64
	PacketHash    []byte
}

type announceEntry struct {
	Packet            *Packet
	Next              time.Time
	Retries           int
	Timestamp         time.Time
	Expires           time.Time
	LocalRebroadcasts int
	BlockRebroadcasts bool
	AttachedInterface *Interface
}

type reverseEntry struct {
	ReceivedIf *Interface
	OutboundIf *Interface
	Timestamp  time.Time
}

type linkEntry struct {
	Timestamp         time.Time
	NextHopID         []byte
	NextHopInterface  *Interface
	RemainingHops     int
	ReceivedInterface *Interface
	Hops              int
	DestinationHash   []byte
	Validated         bool
	ProofTimeout      time.Time
}

type heldAnnounce struct {
	Entry *announceEntry
}

type announceRateEntry struct {
	Last           time.Time
	RateViolations int
	BlockedUntil   time.Time
	Timestamps     []time.Time
}

type discoveryPathRequest struct {
	Timeout             time.Time
	RequestingInterface *Interface
}

type blackholeEntry struct {
	Source []byte
	Until  *time.Time
	Reason *string
}

// Python parity: Transport.path_table entry indices.
const (
	IdxPTDstHash   = 0
	IdxPTTimestamp = 1
	IdxPTNextHop   = 2
	IdxPTHops      = 3
	IdxPTExpires   = 4
	IdxPTRandBlobs = 5
	IdxPTRcvdIf    = 6
	IdxPTPacket    = 7
)

// Python parity: Transport.reverse_table entry indices.
const (
	IdxRTRcvdIf    = 0
	IdxRTOutbIf    = 1
	IdxRTTimestamp = 2
)

// Python parity: Transport.announce_table entry indices.
const (
	IdxATTimestamp = 0
	IdxATRtrnsTmo  = 1
	IdxATRetries   = 2
	IdxATRcvdIf    = 3
	IdxATHops      = 4
	IdxATPacket    = 5
	IdxATLclRbrd   = 6
	IdxATBlckRbrd  = 7
	IdxATAttchdIf  = 8
)

// Python parity: Transport.link_table entry indices.
const (
	IdxLTTimestamp = 0
	IdxLTNhTrid    = 1
	IdxLTNhIf      = 2
	IdxLTRemHops   = 3
	IdxLTRcvdIf    = 4
	IdxLTHops      = 5
	IdxLTDstHash   = 6
	IdxLTValidated = 7
	IdxLTProofTmo  = 8
)

// Python parity: Transport.tunnels entry indices.
const (
	IdxTTTunnelID = 0
	IdxTTIf       = 1
	IdxTTPaths    = 2
	IdxTTExpires  = 3
)

type tunnelEntry struct {
	ID        []byte
	Interface *Interface
	ExpiresAt time.Time
	Paths     map[string]*tunnelPathEntry
}

type tunnelPathEntry struct {
	Timestamp    time.Time
	ReceivedFrom []byte
	Hops         int
	ExpiresAt    time.Time
	RandomBlobs  [][]byte
	PacketHash   []byte
}

func Start(owner *Reticulum) {
	Owner = owner
	if TransportIdentity == nil && Owner != nil {
		identityPath := filepath.Join(Owner.StoragePath, "transport_identity")
		if st, err := os.Stat(identityPath); err == nil && !st.IsDir() {
			if id, err := IdentityFromFile(identityPath); err == nil {
				TransportIdentity = id
				Log("Loaded Transport Identity from storage", LogVerbose)
			}
		}
		if TransportIdentity == nil {
			Log("No valid Transport Identity in storage, creating...", LogVerbose)
			if id, err := NewIdentity(); err == nil {
				TransportIdentity = id
				_ = os.MkdirAll(Owner.StoragePath, 0o755)
				_ = TransportIdentity.Save(identityPath)
			}
		}
	}

	if Owner != nil && !Owner.IsConnectedToSharedInstance {
		packetHashlistPath := filepath.Join(Owner.StoragePath, "packet_hashlist")
		if st, err := os.Stat(packetHashlistPath); err == nil && !st.IsDir() {
			data, err := os.ReadFile(packetHashlistPath)
			if err != nil {
				Log(fmt.Sprintf("Could not load packet hashlist from storage: %v", err), LogError)
			} else {
				var hashes [][]byte
				if err := umsgpack.Unpackb(data, &hashes); err != nil {
					if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
						backupPath := fmt.Sprintf("%s.corrupt.%d", packetHashlistPath, time.Now().UnixNano())
						if renameErr := os.Rename(packetHashlistPath, backupPath); renameErr != nil {
							_ = os.Remove(packetHashlistPath)
						} else {
							Log(fmt.Sprintf("Packet hashlist storage was truncated; moved to %s", backupPath), LogWarning)
						}
					} else {
						Log(fmt.Sprintf("Could not decode packet hashlist from storage: %v", err), LogError)
					}
				} else {
					packetHashMu.Lock()
					for _, h := range hashes {
						if len(h) == 0 {
							continue
						}
						var k hashKey
						copy(k[:], h)
						PacketHashSet[k] = struct{}{}
					}
					packetHashMu.Unlock()
				}
			}
		}
	}
	reloadBlackhole()

	dest, err := NewDestination(nil, DestinationIN, DestinationPLAIN, "rnstransport", "path", "request")
	if err == nil {
		pathRequestDest = dest
		pathRequestDest.SetPacketCallback(pathRequestHandler)
		controlDestinations = append(controlDestinations, pathRequestDest)
		controlHashesMu.Lock()
		controlHashes[string(pathRequestDest.Hash)] = struct{}{}
		controlHashesMu.Unlock()
	} else {
		Log(fmt.Sprintf("Could not create path request destination: %v", err), LogError)
	}

	dest, err = NewDestination(nil, DestinationIN, DestinationPLAIN, "rnstransport", "tunnel", "synthesize")
	if err == nil {
		tunnelSynthesizeDest = dest
		tunnelSynthesizeDest.SetPacketCallback(tunnelSynthesizeHandler)
		controlDestinations = append(controlDestinations, tunnelSynthesizeDest)
		controlHashesMu.Lock()
		controlHashes[string(tunnelSynthesizeDest.Hash)] = struct{}{}
		controlHashesMu.Unlock()
	} else {
		Log(fmt.Sprintf("Could not create tunnel synthesize destination: %v", err), LogError)
	}

	remoteManagementActive = false
	if RemoteManagementEnabled() && Owner != nil && !Owner.IsConnectedToSharedInstance && TransportIdentity != nil {
		if remoteManagementDest == nil {
			dest, err := NewDestination(TransportIdentity, DestinationIN, DestinationSINGLE, "rnstransport", "remote", "management")
			if err != nil {
				Log(fmt.Sprintf("Could not create remote management destination: %v", err), LogError)
			} else {
				remoteManagementDest = dest
			}
		}
		if remoteManagementDest != nil {
			remoteManagementAllowedMu.RLock()
			allowed := make([][]byte, 0, len(remoteManagementAllowed))
			for _, v := range remoteManagementAllowed {
				allowed = append(allowed, append([]byte(nil), v...))
			}
			remoteManagementAllowedMu.RUnlock()

			if err := remoteManagementDest.RegisterRequestHandler("/status", remoteStatusHandler, DestinationALLOW_LIST, allowed); err != nil {
				Log(fmt.Sprintf("Could not register remote status handler: %v", err), LogError)
			} else if err := remoteManagementDest.RegisterRequestHandler("/path", remotePathHandler, DestinationALLOW_LIST, allowed); err != nil {
				Log(fmt.Sprintf("Could not register remote path handler: %v", err), LogError)
				remoteManagementDest.DeregisterRequestHandler("/status")
			} else {
				mgmtHashes = append(mgmtHashes, append([]byte(nil), remoteManagementDest.Hash...))
				Log(fmt.Sprintf("Enabled remote management on %s", str(remoteManagementDest)), LogNotice)
				remoteManagementActive = true
			}
		}
	}

	if !PublishBlackholeEnabled() || Owner == nil || Owner.IsConnectedToSharedInstance || TransportIdentity == nil {
		if blackholeDestination != nil {
			blackholeDestination.DeregisterRequestHandler("/list")
		}
		blackholeDestination = nil
	} else {
		if blackholeDestination == nil {
			dest, err := NewDestination(TransportIdentity, DestinationIN, DestinationSINGLE, TransportAppName, "info", "blackhole")
			if err != nil {
				Log(fmt.Sprintf("Could not create blackhole destination: %v", err), LogError)
			} else {
				blackholeDestination = dest
			}
		}
		if blackholeDestination != nil {
			if err := blackholeDestination.RegisterRequestHandler("/list", blackholeListHandler, DestinationALLOW_ALL, nil); err != nil {
				Log(fmt.Sprintf("Could not register blackhole list handler: %v", err), LogError)
			} else {
				mgmtHashes = append(mgmtHashes, append([]byte(nil), blackholeDestination.Hash...))
				if TransportIdentity != nil {
					Log(fmt.Sprintf("Enabled blackhole list publishing for transport identity %s", PrettyHexRep(TransportIdentity.Hash)), LogNotice)
				}
			}
		}
	}

	if Owner != nil && !Owner.IsConnectedToSharedInstance && NetworkIdentity != nil {
		if instanceDestination == nil {
			dest, err := NewDestination(NetworkIdentity, DestinationIN, DestinationSINGLE, TransportAppName, "network", "instance", strings.ToLower(hex.EncodeToString(NetworkIdentity.Hash)))
			if err != nil {
				Log(fmt.Sprintf("Could not create network instance destination: %v", err), LogError)
			} else {
				instanceDestination = dest
			}
		}
		if networkDestination == nil {
			dest, err := NewDestination(NetworkIdentity, DestinationIN, DestinationSINGLE, TransportAppName, "network")
			if err != nil {
				Log(fmt.Sprintf("Could not create network destination: %v", err), LogError)
			} else {
				networkDestination = dest
			}
		}
		if instanceDestination != nil {
			mgmtHashes = append(mgmtHashes, append([]byte(nil), instanceDestination.Hash...))
		}
		if networkDestination != nil {
			mgmtHashes = append(mgmtHashes, append([]byte(nil), networkDestination.Hash...))
		}
	}

	// Python: Transport.cache_last_cleaned = time.time() + 60 (deferred 60s, set after destination setup)
	LastCacheCleaned = time.Now().Add(60 * time.Second)
	LastMgmtAnnounce = time.Now().Add(-(mgmtAnnounceInterval - initialMgmtAnnounceWait))

	// start background loops
	go JobLoop()
	go CountTrafficLoop()

	if TransportEnabled() && Owner != nil && !Owner.IsConnectedToSharedInstance {
		pathTablePath := filepath.Join(Owner.StoragePath, "destination_table")
		if st, err := os.Stat(pathTablePath); err == nil && !st.IsDir() {
			data, err := os.ReadFile(pathTablePath)
			if err != nil {
				Log(fmt.Sprintf("Could not load destination table from storage: %v", err), LogError)
			} else {
				var rawEntries [][]any
				if err := umsgpack.Unpackb(data, &rawEntries); err != nil {
					if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
						backupPath := fmt.Sprintf("%s.corrupt.%d", pathTablePath, time.Now().UnixNano())
						if renameErr := os.Rename(pathTablePath, backupPath); renameErr != nil {
							_ = os.Remove(pathTablePath)
						} else {
							Log(fmt.Sprintf("Destination table storage was truncated; moved to %s", backupPath), LogWarning)
						}
					} else {
						Log(fmt.Sprintf("Could not decode destination table from storage: %v", err), LogError)
					}
				} else {
					loaded := 0
					for _, re := range rawEntries {
						if len(re) <= IdxPTPacket {
							continue
						}
						dst, _ := re[IdxPTDstHash].([]byte)
						if len(dst) != truncatedHashBytes {
							continue
						}
						ts := asFloat64(re[IdxPTTimestamp])
						nextHop, _ := re[IdxPTNextHop].([]byte)
						hops := func(v any) int {
							switch val := v.(type) {
							case int:
								return val
							case int8:
								return int(val)
							case int16:
								return int(val)
							case int32:
								return int(val)
							case int64:
								return int(val)
							case uint:
								return int(val)
							case uint8:
								return int(val)
							case uint16:
								return int(val)
							case uint32:
								return int(val)
							case uint64:
								return int(val)
							case float32:
								return int(val)
							case float64:
								return int(val)
							default:
								return 0
							}
						}(re[IdxPTHops])
						exp := asFloat64(re[IdxPTExpires])

						var randomBlobs [][]byte
						if rb, ok := re[IdxPTRandBlobs].([]any); ok {
							for _, item := range rb {
								if b, ok := item.([]byte); ok && len(b) > 0 {
									randomBlobs = append(randomBlobs, b)
								}
							}
						}

						ifHash, _ := re[IdxPTRcvdIf].([]byte)
						recvIf := findInterfaceFromHash(ifHash)
						packetHash, _ := re[IdxPTPacket].([]byte)

						// Load announce packet from cache (nil if not available)
						var announce *Packet
						if Owner != nil && len(packetHash) > 0 {
							path := filepath.Join(Owner.CachePath, "announces", hex.EncodeToString(packetHash))
							if fileData, err := os.ReadFile(path); err == nil && len(fileData) > 0 {
								var cached []any
								if err := umsgpack.Unpackb(fileData, &cached); err == nil && len(cached) >= 1 {
									if rawAnnounce, ok := cached[0].([]byte); ok && len(rawAnnounce) > 0 {
										pkt := NewPacket(nil, rawAnnounce, PacketTypeData, PacketCtxNone, Broadcast, HeaderType1, nil, nil, true, FlagUnset)
										if pkt != nil && pkt.Unpack() {
											announce = pkt
										}
									}
								}
							}
						}

						// Check blackhole
						blackholed := false
						blackholeMu.RLock()
						blen := len(blackholedIdentities)
						blackholeMu.RUnlock()
						if blen > 0 {
							if identity := IdentityRecall(dst); identity != nil {
								var k hashKey
								if len(identity.Hash) >= truncatedHashBytes {
									copy(k[:], identity.Hash[:truncatedHashBytes])
									blackholeMu.RLock()
									_, blackholed = blackholedIdentities[k]
									blackholeMu.RUnlock()
								}
							}
						}

						if announce != nil && recvIf != nil && !blackholed {
							announce.Hops += 1
							key, ok := func(hash []byte) (hashKey, bool) {
								if len(hash) < truncatedHashBytes {
									return hashKey{}, false
								}
								var k hashKey
								copy(k[:], hash[:truncatedHashBytes])
								return k, true
							}(dst)
							if !ok {
								continue
							}
							entry := &PathEntry{
								NextHop:       append([]byte(nil), nextHop...),
								RecvInterface: recvIf,
								Hops:          hops,
								Timestamp: func() time.Time {
									sec, frac := math.Modf(ts)
									if sec < 0 {
										return time.Unix(0, 0)
									}
									return time.Unix(int64(sec), int64(frac*1e9))
								}(),
								ExpiresAt: func() time.Time {
									sec, frac := math.Modf(exp)
									if sec < 0 {
										return time.Unix(0, 0)
									}
									return time.Unix(int64(sec), int64(frac*1e9))
								}(),
								RandomBlobs: randomBlobs,
								PacketHash:  append([]byte(nil), announce.PacketHash...),
							}
							pathTableMu.Lock()
							pathTable[key] = entry
							pathTableMu.Unlock()
							Log(fmt.Sprintf("Loaded path table entry for %s from storage", PrettyHexRep(dst)), LogDebug)
							loaded++
						} else {
							Log(fmt.Sprintf("Could not reconstruct path table entry from storage for %s", PrettyHexRep(dst)), LogDebug)
							if announce == nil {
								Log("The announce packet could not be loaded from cache", LogDebug)
							}
							if recvIf == nil {
								Log("The interface is no longer available", LogDebug)
							}
							if blackholed {
								Log("The associated identity is blackholed", LogDebug)
							}
						}
					}
					pathTableMu.RLock()
					pathCount := len(pathTable)
					pathTableMu.RUnlock()
					specifier := "entries"
					if pathCount == 1 {
						specifier = "entry"
					}
					Log(fmt.Sprintf("Loaded %d path table %s from storage", pathCount, specifier), LogVerbose)
				}
			}
		}

		tunnelTablePath := filepath.Join(Owner.StoragePath, "tunnels")
		if st, err := os.Stat(tunnelTablePath); err == nil && !st.IsDir() {
			data, err := os.ReadFile(tunnelTablePath)
			if err != nil {
				Log(fmt.Sprintf("Could not load tunnel table from storage: %v", err), LogError)
			} else {
				var rawTunnels []any
				if err := umsgpack.Unpackb(data, &rawTunnels); err != nil {
					if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
						backupPath := fmt.Sprintf("%s.corrupt.%d", tunnelTablePath, time.Now().UnixNano())
						if renameErr := os.Rename(tunnelTablePath, backupPath); renameErr != nil {
							_ = os.Remove(tunnelTablePath)
						} else {
							Log(fmt.Sprintf("Tunnel table storage was truncated; moved to %s", backupPath), LogWarning)
						}
					} else {
						Log(fmt.Sprintf("Could not decode tunnel table from storage: %v", err), LogError)
					}
				} else {
					loaded := 0
					for _, t := range rawTunnels {
						tlist, ok := t.([]any)
						if !ok || len(tlist) <= IdxTTExpires {
							continue
						}
						tunnelID, _ := tlist[IdxTTTunnelID].([]byte)
						if len(tunnelID) != HashLengthBytes {
							continue
						}
						ifHash, _ := tlist[IdxTTIf].([]byte)
						serialisedPathsAny := tlist[IdxTTPaths]
						expires := asFloat64(tlist[IdxTTExpires])
						ifc := findInterfaceFromHash(ifHash)

						te := &tunnelEntry{
							ID:        append([]byte(nil), tunnelID...),
							Interface: ifc,
							ExpiresAt: func() time.Time {
								sec, frac := math.Modf(expires)
								if sec < 0 {
									return time.Unix(0, 0)
								}
								return time.Unix(int64(sec), int64(frac*1e9))
							}(),
							Paths: make(map[string]*tunnelPathEntry),
						}

						if serialisedPaths, ok := serialisedPathsAny.([]any); ok && len(serialisedPaths) > 0 {
							for _, sp := range serialisedPaths {
								elist, ok := sp.([]any)
								if !ok || len(elist) <= IdxPTPacket {
									continue
								}
								dstHash, _ := elist[IdxPTDstHash].([]byte)
								if len(dstHash) != truncatedHashBytes {
									continue
								}
								ts := asFloat64(elist[IdxPTTimestamp])
								receivedFrom, _ := elist[IdxPTNextHop].([]byte)
								hops := func(v any) int {
									switch val := v.(type) {
									case int:
										return val
									case int8:
										return int(val)
									case int16:
										return int(val)
									case int32:
										return int(val)
									case int64:
										return int(val)
									case uint:
										return int(val)
									case uint8:
										return int(val)
									case uint16:
										return int(val)
									case uint32:
										return int(val)
									case uint64:
										return int(val)
									case float32:
										return int(val)
									case float64:
										return int(val)
									default:
										return 0
									}
								}(elist[IdxPTHops])
								exp := asFloat64(elist[IdxPTExpires])

								blobs := make([][]byte, 0)
								if blobsAny, ok := elist[IdxPTRandBlobs].([]any); ok {
									seen := make(map[string]struct{})
									for _, b := range blobsAny {
										bb, ok := b.([]byte)
										if !ok || len(bb) == 0 {
											continue
										}
										key := string(bb)
										if _, exists := seen[key]; exists {
											continue
										}
										seen[key] = struct{}{}
										blobs = append(blobs, append([]byte(nil), bb...))
									}
								}

								packetHash, _ := elist[IdxPTPacket].([]byte)
								if len(packetHash) == 0 {
									continue
								}

								// Python parity: only add tunnel path if announce packet is loadable from cache.
								var tunnelAnnounce *Packet
								if Owner != nil {
									cachePath := filepath.Join(Owner.CachePath, "announces", hex.EncodeToString(packetHash))
									if fileData, err := os.ReadFile(cachePath); err == nil && len(fileData) > 0 {
										var cached []any
										if err := umsgpack.Unpackb(fileData, &cached); err == nil && len(cached) >= 1 {
											if rawAnnounce, ok := cached[0].([]byte); ok && len(rawAnnounce) > 0 {
												pkt := NewPacket(nil, rawAnnounce, PacketTypeData, PacketCtxNone, Broadcast, HeaderType1, nil, nil, true, FlagUnset)
												if pkt != nil && pkt.Unpack() {
													tunnelAnnounce = pkt
												}
											}
										}
									}
								}
								if tunnelAnnounce == nil {
									continue
								}
								tunnelAnnounce.Hops += 1

								te.Paths[string(dstHash)] = &tunnelPathEntry{
									Timestamp: func() time.Time {
										sec, frac := math.Modf(ts)
										if sec < 0 {
											return time.Unix(0, 0)
										}
										return time.Unix(int64(sec), int64(frac*1e9))
									}(),
									ReceivedFrom: append([]byte(nil), receivedFrom...),
									Hops:         hops,
									ExpiresAt: func() time.Time {
										sec, frac := math.Modf(exp)
										if sec < 0 {
											return time.Unix(0, 0)
										}
										return time.Unix(int64(sec), int64(frac*1e9))
									}(),
									RandomBlobs: blobs,
									PacketHash:  append([]byte(nil), tunnelAnnounce.PacketHash...),
								}
							}
							if len(te.Paths) == 0 {
								continue
							}
						} else {
							te.Paths = nil
						}

						tunnelsMu.Lock()
						tunnels[string(tunnelID)] = te
						tunnelsMu.Unlock()
						if ifc != nil {
							ifc.TunnelID = append([]byte(nil), tunnelID...)
						}
						loaded++
					}
					tunnelsMu.RLock()
					tunnelCount := len(tunnels)
					tunnelsMu.RUnlock()
					specifier := "entries"
					if tunnelCount == 1 {
						specifier = "entry"
					}
					Log(fmt.Sprintf("Loaded %d tunnel table %s from storage", tunnelCount, specifier), LogVerbose)
				}
			}
		}
	}

	if TransportEnabled() && ProbeDestinationEnabled() && Owner != nil && !Owner.IsConnectedToSharedInstance && TransportIdentity != nil {
		if ProbeDestination == nil {
			dest, err := NewDestination(TransportIdentity, DestinationIN, DestinationSINGLE, TransportAppName, "probe")
			if err == nil {
				ProbeDestination = dest
			} else {
				Log(fmt.Sprintf("Could not create probe destination: %v", err), LogError)
			}
		}
		if ProbeDestination != nil {
			ProbeDestination.AcceptsLinks(false)
			ProbeDestination.SetProofStrategy(DestinationPROVE_ALL)
			Log(fmt.Sprintf("Transport Instance will respond to probe requests on %s", str(ProbeDestination)), LogNotice)
		}
	} else if ProbeDestination != nil {
		DeregisterDestination(ProbeDestination)
		ProbeDestination = nil
	}
	if remoteManagementActive && remoteManagementDest != nil {
		mgmtDestinations = append(mgmtDestinations, remoteManagementDest)
	}
	if blackholeDestination != nil {
		mgmtDestinations = append(mgmtDestinations, blackholeDestination)
	}
	if Owner != nil && !Owner.IsConnectedToSharedInstance && NetworkIdentity != nil {
		if instanceDestination != nil {
			mgmtDestinations = append(mgmtDestinations, instanceDestination)
		}
		if networkDestination != nil {
			mgmtDestinations = append(mgmtDestinations, networkDestination)
		}
	}
	if ProbeDestination != nil {
		mgmtDestinations = append(mgmtDestinations, ProbeDestination)
	}
	if TransportEnabled() {
		if TransportIdentity != nil {
			Log(fmt.Sprintf("Transport instance %s started", TransportIdentity), LogVerbose)
		}
		StartTime = time.Now()
	}

	// sort interfaces by bitrate
	PrioritizeInterfaces()

	// Python parity: reset tunnel_id for all interfaces, then synthesize for those wanting it.
	for _, ifc := range Interfaces {
		ifc.TunnelID = nil
	}
	for _, ifc := range Interfaces {
		if ifc.WantsTunnel {
			SynthesizeTunnel(ifc)
		}
	}
}

func findInterfaceFromHash(hash []byte) *Interface {
	for _, ifc := range Interfaces {
		if ifc != nil && bytes.Equal(ifc.GetHash(), hash) {
			return ifc
		}
	}
	return nil
}

func reloadBlackhole() {
	basePath := ""
	if Owner != nil && strings.TrimSpace(Owner.StoragePath) != "" {
		basePath = filepath.Join(Owner.StoragePath, "blackhole")
	}
	if basePath == "" {
		return
	}
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return
	}

	now := time.Now()
	for _, entry := range entries {
		if entry == nil || entry.IsDir() {
			continue
		}
		name := entry.Name()
		func() {
			defer func() {
				if r := recover(); r != nil {
					Log(fmt.Sprintf("Could not load blackholed identities from source file %s: %v", name, r), LogError)
				}
			}()
			var sourceIdentityHash []byte
			if name == "local" {
				if TransportIdentity == nil {
					return
				}
				sourceIdentityHash = TransportIdentity.Hash
			} else {
				if len(name) != truncatedHashBytes*2 {
					err := fmt.Errorf("Identity hash length for blackhole source %s is invalid", name)
					Log(fmt.Sprintf("Could not load blackholed identities from source file %s: %v", name, err), LogError)
					TraceException(err)
					return
				}
				decoded, err := hex.DecodeString(name)
				if err != nil || len(decoded) != truncatedHashBytes {
					if err == nil {
						err = fmt.Errorf("Identity hash length for blackhole source %s is invalid", name)
					}
					Log(fmt.Sprintf("Could not load blackholed identities from source file %s: %v", name, err), LogError)
					TraceException(err)
					return
				}
				enabled := false
				for _, source := range BlackholeSources() {
					if bytes.Equal(source, decoded) {
						enabled = true
						break
					}
				}
				if !enabled {
					Log(fmt.Sprintf("Skipping disabled blackhole source %s", PrettyHexRep(decoded)), LogVerbose)
					return
				}
				sourceIdentityHash = decoded
			}

			path := filepath.Join(basePath, name)
			packed, err := os.ReadFile(path)
			if err != nil {
				Log(fmt.Sprintf("Could not load blackholed identities from source file %s: %v", name, err), LogError)
				TraceException(err)
				return
			}
			raw := map[any]any{}
			if err := umsgpack.Unpackb(packed, &raw); err != nil {
				Log(fmt.Sprintf("Could not load blackholed identities from source file %s: %v", name, err), LogError)
				TraceException(err)
				return
			}
			blackholeMu.Lock()
			defer blackholeMu.Unlock()
			for rawKey, rawValue := range raw {
				var key hashKey
				var ok bool
				switch v := rawKey.(type) {
				case string:
					if len(v) != truncatedHashBytes {
						continue
					}
					key, ok = func(hash []byte) (hashKey, bool) {
						if len(hash) < truncatedHashBytes {
							return hashKey{}, false
						}
						var key hashKey
						copy(key[:], hash[:truncatedHashBytes])
						return key, true
					}([]byte(v))
				case umsgpack.BinaryKey:
					if len(v) != truncatedHashBytes {
						continue
					}
					key, ok = func(hash []byte) (hashKey, bool) {
						if len(hash) < truncatedHashBytes {
							return hashKey{}, false
						}
						var key hashKey
						copy(key[:], hash[:truncatedHashBytes])
						return key, true
					}([]byte(string(v)))
				case []byte:
					if len(v) != truncatedHashBytes {
						continue
					}
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
				if existing := blackholedIdentities[key]; existing != nil && TransportIdentity != nil && bytes.Equal(existing.Source, TransportIdentity.Hash) {
					continue
				}
				entryMap, ok := rawValue.(map[any]any)
				if !ok {
					continue
				}
				decodedEntry := &blackholeEntry{Source: append([]byte(nil), sourceIdentityHash...)}
				if untilVal, exists := entryMap["until"]; exists && untilVal != nil {
					untilUnix := asFloat64(untilVal)
					if untilUnix > 0 {
						sec, frac := math.Modf(untilUnix)
						until := time.Unix(int64(sec), int64(frac*1e9))
						if sec < 0 {
							until = time.Unix(0, 0)
						}
						decodedEntry.Until = &until
					}
				}
				if reasonVal, exists := entryMap["reason"]; exists && reasonVal != nil {
					switch reason := reasonVal.(type) {
					case string:
						decodedEntry.Reason = &reason
					case []byte:
						if len(reason) > 0 {
							reasonStr := string(reason)
							decodedEntry.Reason = &reasonStr
						}
					}
				}
				if decodedEntry.Until == nil || now.Before(*decodedEntry.Until) {
					blackholedIdentities[key] = decodedEntry
				}
			}
		}()
	}

	removeBlackholedPaths()
}

func removeBlackholedPaths() {
	pathTableMu.RLock()
	keys := make([]hashKey, 0, len(pathTable))
	for key := range pathTable {
		keys = append(keys, key)
	}
	pathTableMu.RUnlock()

	drop := make([]hashKey, 0)
	for _, key := range keys {
		func() {
			defer func() {
				if r := recover(); r != nil {
					Log(fmt.Sprintf("Error while enumerating blackhole-associated destinations: %v", r), LogError)
				}
			}()
			identity := IdentityRecall(key[:])
			if identity != nil {
				if hk, ok := func(hash []byte) (hashKey, bool) {
					if len(hash) < truncatedHashBytes {
						return hashKey{}, false
					}
					var k hashKey
					copy(k[:], hash[:truncatedHashBytes])
					return k, true
				}(identity.Hash); ok {
					blackholeMu.RLock()
					_, blackholed := blackholedIdentities[hk]
					blackholeMu.RUnlock()
					if blackholed {
						drop = append(drop, key)
					}
				}
			}
		}()
	}

	if len(drop) == 0 {
		return
	}
	pathTableMu.Lock()
	for _, key := range drop {
		func() {
			defer func() {
				if r := recover(); r != nil {
					Log(fmt.Sprintf("Error while dropping blackhole-associated destination from path table: %v", r), LogError)
				}
			}()
			delete(pathTable, key)
		}()
	}
	pathTableMu.Unlock()
	ms := ""
	if len(drop) != 1 {
		ms = "s"
	}
	Log(fmt.Sprintf("Removed %d destination%s associated with blackholed identities from path table", len(drop), ms), LogInfo)
}

func BlackholeIdentity(identityHash []byte, until *time.Time, reason *string) (result any) {
	defer func() {
		if r := recover(); r != nil {
			Log(fmt.Sprintf("Error while blackholing identity: %v", r), LogError)
			result = false
		}
	}()
	key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) != truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash)
		return key, true
	}(identityHash)
	if !ok {
		return nil
	}

	entry := &blackholeEntry{Source: append([]byte(nil), TransportIdentity.Hash...)}
	if until != nil && !until.IsZero() {
		t := *until
		entry.Until = &t
	}
	if reason != nil {
		r := *reason
		entry.Reason = &r
	}

	blackholeMu.Lock()
	if _, exists := blackholedIdentities[key]; exists {
		blackholeMu.Unlock()
		return nil
	}
	blackholedIdentities[key] = entry
	blackholeMu.Unlock()

	persistBlackhole()
	removeBlackholedPaths()
	Log(fmt.Sprintf("Blackholed identity %s", PrettyHexRep(identityHash)), LogInfo)
	return true
}

func UnblackholeIdentity(identityHash []byte) (result any) {
	defer func() {
		if r := recover(); r != nil {
			Log(fmt.Sprintf("Error while unblackholing identity: %v", r), LogError)
			result = false
		}
	}()
	key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) != truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash)
		return key, true
	}(identityHash)
	if !ok {
		return nil
	}

	blackholeMu.Lock()
	if _, exists := blackholedIdentities[key]; !exists {
		blackholeMu.Unlock()
		return nil
	}
	delete(blackholedIdentities, key)
	blackholeMu.Unlock()

	persistBlackhole()
	Log(fmt.Sprintf("Lifted blackhole for identity %s", PrettyHexRep(identityHash)), LogInfo)
	return true
}

func persistBlackhole() {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("%v", r)
			Log(fmt.Sprintf("Error while persisting blackhole list: %v", err), LogError)
			TraceException(err)
		}
	}()
	blackholeMu.RLock()
	localBlackhole := make(map[any]any)
	for key, entry := range blackholedIdentities {
		if entry == nil || TransportIdentity == nil || !bytes.Equal(entry.Source, TransportIdentity.Hash) {
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
		localBlackhole[umsgpack.BinaryKey(string(key[:]))] = serialised
	}
	blackholeMu.RUnlock()
	packed, err := umsgpack.Packb(localBlackhole)
	if err != nil {
		Log(fmt.Sprintf("Error while persisting blackhole list: %v", err), LogError)
		TraceException(err)
		return
	}
	if Owner == nil || len(Owner.StoragePath) == 0 {
		return
	}
	localPath := filepath.Join(Owner.StoragePath, "blackhole", "local")
	tmpPath := localPath + ".tmp"
	if err := os.WriteFile(tmpPath, packed, 0o600); err != nil {
		Log(fmt.Sprintf("Error while persisting blackhole list: %v", err), LogError)
		TraceException(err)
		return
	}
	if _, err := os.Stat(localPath); err == nil {
		_ = os.Remove(localPath)
	}
	if err := os.Rename(tmpPath, localPath); err != nil {
		Log(fmt.Sprintf("Error while persisting blackhole list: %v", err), LogError)
		TraceException(err)
		return
	}
}

func blackholeListHandler(_ string, _ any, _ []byte, _ []byte, remoteIdentity *Identity, _ time.Time) (result any) {
	defer func() {
		if r := recover(); r != nil {
			Log(fmt.Sprintf("An error occurred while processing blackhole list request from %v", remoteIdentity), LogError)
			Log(fmt.Sprintf("The contained exception was: %v", r), LogError)
			result = nil
		}
	}()
	blackholeMu.RLock()
	defer blackholeMu.RUnlock()
	return blackholedIdentities
}

// ExitHandler best-effort persistence hook mirroring Transport.exit_handler() in Python.
func ExitHandler() {
	if instance != nil && !instance.IsConnectedToSharedInstance {
		PersistData()
	}
}

// PersistData persists transport tables, mirroring Python's
// Transport.persist_data() / Reticulum.__persist_data() behaviour.
func PersistData() {
	if instance != nil && instance.IsConnectedToSharedInstance {
		return
	}
	savePacketHashlist()
	SavePathTable()
	saveTunnelTable()
	lastTablesPersisted = time.Now()
}

func savePacketHashlist() {
	if Owner == nil || Owner.IsConnectedToSharedInstance {
		return
	}

	waitInterval := 200 * time.Millisecond
	waitTimeout := 5 * time.Second
	waitStart := time.Now()
	for {
		packetHashlistSaveMu.Lock()
		if !savingPacketHashlist {
			savingPacketHashlist = true
			packetHashlistSaveMu.Unlock()
			break
		}
		packetHashlistSaveMu.Unlock()
		time.Sleep(waitInterval)
		if time.Since(waitStart) > waitTimeout {
			Log(fmt.Sprintf("Could not save packet hashlist to storage, waiting for previous save operation timed out."), LogError)
			return
		}
	}
	defer func() {
		packetHashlistSaveMu.Lock()
		savingPacketHashlist = false
		packetHashlistSaveMu.Unlock()
	}()

	func() {
		defer func() {
			if r := recover(); r != nil {
				Log(fmt.Sprintf("Could not save packet hashlist to storage, the contained exception was: %v", r), LogError)
			}
		}()

		saveStart := time.Now()

		if !TransportEnabled() {
			packetHashMu.Lock()
			PacketHashSet = make(map[hashKey]struct{})
			PacketHashSet2 = make(map[hashKey]struct{})
			packetHashMu.Unlock()
		} else {
			Log(fmt.Sprintf("Saving packet hashlist to storage..."), LogDebug)
		}

		packetHashMu.RLock()
		hashes := make([][]byte, 0, len(PacketHashSet))
		for k := range PacketHashSet {
			b := make([]byte, truncatedHashBytes)
			copy(b, k[:])
			hashes = append(hashes, b)
		}
		packetHashMu.RUnlock()

		buf, err := umsgpack.Packb(hashes)
		if err != nil {
			Log(fmt.Sprintf("Could not save packet hashlist to storage, the contained exception was: %v", err), LogError)
			return
		}
		path := filepath.Join(Owner.StoragePath, "packet_hashlist")
		if err := os.WriteFile(path, buf, 0o600); err != nil {
			Log(fmt.Sprintf("Could not save packet hashlist to storage, the contained exception was: %v", err), LogError)
			return
		}

		saveTime := time.Since(saveStart)
		var timeStr string
		if saveTime < time.Second {
			timeStr = fmt.Sprintf("%.2fms", float64(saveTime.Microseconds())/1000.0)
		} else {
			timeStr = fmt.Sprintf("%.2fs", saveTime.Seconds())
		}
		Log(fmt.Sprintf("Saved packet hashlist in %s", timeStr), LogDebug)
	}()
}

func SavePathTable() {
	if Owner == nil || Owner.IsConnectedToSharedInstance || !TransportEnabled() {
		return
	}

	// Python parity: serialize concurrent saves via a flag with a bounded wait.
	pathTableSaveMu.Lock()
	if savingPathTable {
		pathTableSaveMu.Unlock()
		waitInterval := 200 * time.Millisecond
		waitTimeout := 5 * time.Second
		start := time.Now()
		for {
			time.Sleep(waitInterval)
			pathTableSaveMu.Lock()
			busy := savingPathTable
			pathTableSaveMu.Unlock()
			if !busy {
				break
			}
			if time.Since(start) > waitTimeout {
				Log(fmt.Sprintf("Could not save path table to storage, waiting for previous save operation timed out."), LogError)
				return
			}
		}
		pathTableSaveMu.Lock()
	}
	savingPathTable = true
	pathTableSaveMu.Unlock()
	defer func() {
		pathTableSaveMu.Lock()
		savingPathTable = false
		pathTableSaveMu.Unlock()
		runtime.GC()
	}()

	func() {
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("%v", r)
				Log(fmt.Sprintf("Could not save path table to storage, the contained exception was: %v", err), LogError)
				TraceException(err)
			}
		}()

		saveStart := time.Now()
		Log(fmt.Sprintf("Saving path table to storage..."), LogDebug)

		// Take a snapshot of the path table.
		pathTableMu.RLock()
		snapshot := make(map[hashKey]*PathEntry, len(pathTable))
		for k, v := range pathTable {
			snapshot[k] = v
		}
		pathTableMu.RUnlock()

		entries := make([][]any, 0)
		now := time.Now()

		for key, entry := range snapshot {
			func() {
				defer func() {
					if r := recover(); r != nil {
						Log(fmt.Sprintf("Skipping persist for path table entry due to error: %v", r), LogError)
					}
				}()
				if entry == nil {
					return
				}
				if !entry.ExpiresAt.IsZero() && entry.ExpiresAt.Before(now) {
					return
				}
				if entry.RecvInterface == nil {
					return
				}
				ifHash := entry.RecvInterface.GetHash()

				// Only store entry if the associated interface is still active.
				if findInterfaceFromHash(ifHash) == nil {
					return
				}

				dst := make([]byte, truncatedHashBytes)
				copy(dst, key[:])

				entries = append(entries, []any{
					dst,
					float64(entry.Timestamp.UnixNano()) / 1e9,
					append([]byte(nil), entry.NextHop...),
					entry.Hops,
					float64(entry.ExpiresAt.UnixNano()) / 1e9,
					func(in [][]byte) [][]byte {
						if len(in) == 0 {
							return nil
						}
						out := make([][]byte, len(in))
						for i, blob := range in {
							out[i] = append([]byte(nil), blob...)
						}
						return out
					}(entry.RandomBlobs),
					ifHash,
					append([]byte(nil), entry.PacketHash...),
				})
			}()
		}

		buf, err := umsgpack.Packb(entries)
		if err != nil {
			Log(fmt.Sprintf("Could not save path table to storage, the contained exception was: %v", err), LogError)
			return
		}
		path := filepath.Join(Owner.StoragePath, "destination_table")
		if err := os.WriteFile(path, buf, 0o600); err != nil {
			Log(fmt.Sprintf("Could not save path table to storage, the contained exception was: %v", err), LogError)
			return
		}

		saveTime := time.Since(saveStart)
		var timeStr string
		if saveTime < time.Second {
			timeStr = fmt.Sprintf("%.2fms", float64(saveTime.Microseconds())/1000.0)
		} else {
			timeStr = fmt.Sprintf("%.2fs", saveTime.Seconds())
		}
		Log(fmt.Sprintf("Saved %d path table entries in %s", len(entries), timeStr), LogDebug)
	}()
}

func saveTunnelTable() {
	if Owner == nil || Owner.IsConnectedToSharedInstance || !TransportEnabled() {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			Log(fmt.Sprintf("Could not save tunnel table to storage, the contained exception was: %v", r), LogError)
		}
	}()
	// Python parity: serialize concurrent saves via a flag with a bounded wait.
	tunnelTableSaveMu.Lock()
	if savingTunnelTable {
		tunnelTableSaveMu.Unlock()
		waitInterval := 200 * time.Millisecond
		waitTimeout := 5 * time.Second
		start := time.Now()
		for {
			time.Sleep(waitInterval)
			tunnelTableSaveMu.Lock()
			busy := savingTunnelTable
			tunnelTableSaveMu.Unlock()
			if !busy {
				break
			}
			if time.Since(start) > waitTimeout {
				Log(fmt.Sprintf("Could not save tunnel table to storage, waiting for previous save operation timed out."), LogError)
				return
			}
		}
		tunnelTableSaveMu.Lock()
	}
	savingTunnelTable = true
	tunnelTableSaveMu.Unlock()
	defer func() {
		tunnelTableSaveMu.Lock()
		savingTunnelTable = false
		tunnelTableSaveMu.Unlock()
		runtime.GC()
	}()

	saveStart := time.Now()
	Log(fmt.Sprintf("Saving tunnel table to storage..."), LogDebug)

	now := time.Now()
	serialised := make([][]any, 0)

	tunnelsMu.RLock()
	snapshot := make(map[string]*tunnelEntry, len(tunnels))
	for k, v := range tunnels {
		snapshot[k] = v
	}
	tunnelsMu.RUnlock()

	for _, te := range snapshot {
		if !te.ExpiresAt.IsZero() && te.ExpiresAt.Before(now) {
			continue
		}
		var ifHash []byte
		if te.Interface != nil {
			ifHash = te.Interface.GetHash()
		}
		serialisedPaths := make([][]any, 0)
		for dstKey, pe := range te.Paths {
			dstHash := []byte(dstKey)
			if !pe.ExpiresAt.IsZero() && pe.ExpiresAt.Before(now) {
				continue
			}

			blobs := pe.RandomBlobs
			if len(blobs) > persistRandomBlobs {
				blobs = blobs[len(blobs)-persistRandomBlobs:]
			}

			serialisedPaths = append(serialisedPaths, []any{
				append([]byte(nil), dstHash...),
				float64(pe.Timestamp.UnixNano()) / 1e9,
				append([]byte(nil), pe.ReceivedFrom...),
				pe.Hops,
				float64(pe.ExpiresAt.UnixNano()) / 1e9,
				func(in [][]byte) [][]byte {
					if len(in) == 0 {
						return nil
					}
					out := make([][]byte, len(in))
					for i, blob := range in {
						out[i] = append([]byte(nil), blob...)
					}
					return out
				}(blobs),
				ifHash,
				append([]byte(nil), pe.PacketHash...),
			})
		}
		serialised = append(serialised, []any{
			append([]byte(nil), te.ID...),
			ifHash,
			serialisedPaths,
			float64(te.ExpiresAt.UnixNano()) / 1e9,
		})
	}

	buf, err := umsgpack.Packb(serialised)
	if err != nil {
		Log(fmt.Sprintf("Could not save tunnel table to storage, the contained exception was: %v", err), LogError)
		return
	}
	path := filepath.Join(Owner.StoragePath, "tunnels")
	if err := os.WriteFile(path, buf, 0o600); err != nil {
		Log(fmt.Sprintf("Could not save tunnel table to storage, the contained exception was: %v", err), LogError)
		return
	}

	saveTime := time.Since(saveStart)
	var timeStr string
	if saveTime < time.Second {
		timeStr = fmt.Sprintf("%.2fms", float64(saveTime.Microseconds())/1000.0)
	} else {
		timeStr = fmt.Sprintf("%.2fs", saveTime.Seconds())
	}
	Log(fmt.Sprintf("Saved %d tunnel table entries in %s", len(serialised), timeStr), LogDebug)
}

// -------- prioritisation and traffic counters --------

func PrioritizeInterfaces() {
	defer func() {
		if r := recover(); r != nil {
			Log(fmt.Sprintf("Could not prioritize interfaces according to bitrate. The contained exception was: %v", r), LogError)
		}
	}()

	interfacesMu.Lock()
	defer interfacesMu.Unlock()
	// sort by bitrate descending
	sort.SliceStable(Interfaces, func(i, j int) bool {
		return Interfaces[i].Bitrate > Interfaces[j].Bitrate
	})
}

func CountTrafficLoop() {
	for {
		time.Sleep(time.Second)
		func() {
			defer func() {
				if r := recover(); r != nil {
					Log(fmt.Sprintf("An error occurred while counting interface traffic: %v", r), LogError)
				}
			}()

			var rxb, txb uint64
			var rxs, txs float64

			for _, ifc := range Interfaces {
				if ifc.Parent != nil {
					continue
				}
				now := time.Now()

				if ifc.TrafficCounter == nil {
					tc := TrafficCounter{
						"ts":  now,
						"rxb": ifc.RXB,
						"txb": ifc.TXB,
					}
					ifc.TrafficCounter = &tc
					continue
				}

				tc := *ifc.TrafficCounter
				tcTS, _ := tc["ts"].(time.Time)
				tcRXB, _ := tc["rxb"].(uint64)
				tcTXB, _ := tc["txb"].(uint64)
				rxDiff := ifc.RXB - tcRXB
				txDiff := ifc.TXB - tcTXB
				tsDiff := now.Sub(tcTS).Seconds()
				if tsDiff <= 0 {
					continue
				}

				rxb += rxDiff
				txb += txDiff
				crxs := float64(rxDiff*8) / tsDiff
				ctxs := float64(txDiff*8) / tsDiff
				ifc.CurRxSpeed = crxs
				ifc.CurTxSpeed = ctxs
				rxs += crxs
				txs += ctxs

				tc["rxb"] = ifc.RXB
				tc["txb"] = ifc.TXB
				tc["ts"] = now
			}

			TrafficRXB += rxb
			TrafficTXB += txb
			SpeedRX = rxs
			SpeedTX = txs
		}()
	}
}

// -------- main job loop --------

func JobLoop() {
	for {
		Jobs()
		time.Sleep(JobInterval)
	}
}

func Jobs() {
	var outgoing []*Packet
	pathRequests := make(map[hashKey]*Interface)
	var mgmtAnnouncements []*Destination
	var culled bool

	// Python parity: jobs() takes the lock and waits rather than silently
	// skipping a cycle while IO is active. Skipped cycles can strand
	// short-lived local-client announces in announceTable long enough for
	// their associated path entries to disappear before rebroadcast.
	jobsMu.Lock()
	defer func() {
		jobsMu.Unlock()

		// send collected packets (outside the lock so Outbound can proceed)
		for _, p := range outgoing {
			_ = p.Send()
		}
		// path requests
		for dst, blocked := range pathRequests {
			if _, ok := func(hash []byte) (hashKey, bool) {
				if len(hash) < truncatedHashBytes {
					return hashKey{}, false
				}
				var key hashKey
				copy(key[:], hash[:truncatedHashBytes])
				return key, true
			}(dst[:]); !ok {
				continue
			}

			tag := IdentityGetRandomHash()
			if blocked == nil {
				RequestPath(dst[:], nil, tag, false)
				continue
			}
			for _, ifc := range Interfaces {
				if ifc == nil || ifc == blocked {
					continue
				}
				RequestPath(dst[:], ifc, tag, false)
			}
		}
		if len(mgmtAnnouncements) > 0 {
			announcements := append([]*Destination(nil), mgmtAnnouncements...)
			go func() {
				for _, dest := range announcements {
					if dest == nil {
						continue
					}
					dest.Announce(nil, false, nil, nil, true)
				}
			}()
		}
	}()
	defer func() {
		if r := recover(); r != nil {
			Log(fmt.Sprintf("An exception occurred while running Transport jobs."), LogError)
			Log(fmt.Sprintf("The contained exception was: %v", r), LogError)
		}
	}()
	shouldGC := false

	now := time.Now()

	// ---- pending and active links ----
	if now.Sub(LinksLastChecked) > LinksCheckInt {
		// pending_links / active_links mirror Python closely:
		// check status, remove CLOSED, rediscover paths, etc.
		linkMu.Lock()
		nowLinks := time.Now()

		filterPending := PendingLinks[:0]
		for _, link := range PendingLinks {
			if link == nil {
				continue
			}
			if link.Status == LinkClosed {
				// Python parity: if a pending link closes without being activated on a
				// non-transport instance, expire the associated path and try rediscovery.
				if !TransportEnabled() && link.Destination != nil {
					destHash := link.Destination.Hash
					if ExpirePath(destHash) {
						if Owner == nil || !Owner.IsConnectedToSharedInstance {
							if key, ok := func(hash []byte) (hashKey, bool) {
								if len(hash) < truncatedHashBytes {
									return hashKey{}, false
								}
								var key hashKey
								copy(key[:], hash[:truncatedHashBytes])
								return key, true
							}(destHash); ok {
								pathRequestMu.Lock()
								lastRequest, hasLast := lastPathRequest[key]
								pathRequestMu.Unlock()
								if !hasLast || lastRequest.IsZero() || nowLinks.Sub(lastRequest) > pathRequestMinInterval {
									Log(fmt.Sprintf("Trying to rediscover path for %s since an attempted link was never established", PrettyHexRep(destHash)), LogDebug)
									if pathRequests != nil {
										if _, exists := pathRequests[key]; !exists {
											pathRequests[key] = nil
										}
									}
								}
							}
						}
					}
				}
				continue
			}
			filterPending = append(filterPending, link)
		}
		PendingLinks = filterPending

		filterActive := ActiveLinks[:0]
		for _, link := range ActiveLinks {
			if link == nil || link.Status == LinkClosed {
				continue
			}
			if !link.lastInbound.IsZero() && nowLinks.Sub(link.lastInbound) > link.StaleTime {
				// Teardown in a goroutine: this function runs under jobsMu
				// write lock, and Teardown → sendTeardownPacket → Outbound
				// needs jobsMu read lock — calling it here would deadlock.
				go link.Teardown()
				continue
			}
			filterActive = append(filterActive, link)
		}
		ActiveLinks = filterActive
		linkMu.Unlock()
		LinksLastChecked = now
	}

	// ---- receipts ----
	if now.Sub(ReceiptsLast) > ReceiptsCheckInt {
		receiptsMu.Lock()
		for len(Receipts) > MaxReceipts {
			r := Receipts[0]
			Receipts = Receipts[1:]
			r.Timeout = -1
			r.CheckTimeout()
			shouldGC = true
		}
		for i := 0; i < len(Receipts); {
			rc := Receipts[i]
			rc.CheckTimeout()
			if rc.Status != PacketReceiptSENT {
				Receipts = append(Receipts[:i], Receipts[i+1:]...)
			} else {
				i++
			}
		}
		receiptsMu.Unlock()
		ReceiptsLast = now
	}

	// ---- announces retransmit ----
	if now.Sub(AnnLast) > AnnCheckInt {
		announceMu.Lock()
		for key, entry := range announceTable {
			if entry == nil || entry.Packet == nil {
				delete(announceTable, key)
				continue
			}
			// Python parity: check local rebroadcast limit first, then global retry limit.
			if entry.Retries > 0 && entry.Retries >= localRebroadcastsMax {
				Log(fmt.Sprintf("Completed announce processing for %s, local rebroadcast limit reached", PrettyHexRep(entry.Packet.DestinationHash)), LogExtreme)
				delete(announceTable, key)
				continue
			}
			if entry.Retries > pathfinderRetryLimit {
				Log(fmt.Sprintf("Completed announce processing for %s, retry limit reached", PrettyHexRep(entry.Packet.DestinationHash)), LogExtreme)
				delete(announceTable, key)
				continue
			}
			if entry.Next.After(now) {
				continue
			}
			announceIdentity := IdentityRecall(entry.Packet.DestinationHash)
			announceDestination := &Destination{
				Type:      DestinationSINGLE,
				Direction: DestinationOUT,
				Identity:  announceIdentity,
				Hash:      append([]byte(nil), entry.Packet.DestinationHash...),
				HexHash:   PrettyHexRep(entry.Packet.DestinationHash),
			}
			announceContext := byte(PacketNONE)
			if entry.BlockRebroadcasts {
				announceContext = byte(PacketPATH_RESPONSE)
			}
			send := NewPacket(
				announceDestination,
				append([]byte(nil), entry.Packet.Data...),
				PacketANNOUNCE,
				announceContext,
				TransportDirect,
				HeaderType2,
				append([]byte(nil), entry.Packet.TransportID...),
				entry.AttachedInterface,
				true,
				entry.Packet.ContextFlag,
			)
			if send == nil {
				delete(announceTable, key)
				continue
			}
			if TransportIdentity != nil && len(TransportIdentity.Hash) > 0 {
				send.TransportID = append([]byte(nil), TransportIdentity.Hash...)
			}
			send.Hops = entry.Packet.Hops
			send.DestinationHash = append([]byte(nil), entry.Packet.DestinationHash...)
			send.DestinationType = byte(DestinationSINGLE)
			if entry.BlockRebroadcasts {
				Log(fmt.Sprintf("Rebroadcasting announce as path response for %s with hop count %d", PrettyHexRep(entry.Packet.DestinationHash), send.Hops), LogDebug)
			} else {
				Log(fmt.Sprintf("Rebroadcasting announce for %s with hop count %d", PrettyHexRep(entry.Packet.DestinationHash), send.Hops), LogDebug)
			}
			outgoing = append(outgoing, send)
			if held := heldAnnounces[key]; held != nil {
				delete(heldAnnounces, key)
				if held.Entry != nil {
					announceTable[key] = held.Entry
					Log(fmt.Sprintf("Reinserting held announce into table"), LogDebug)
					continue
				}
			}

			entry.Retries++
			// Python parity: retry timeout = G + RW (fixed, not random).
			entry.Next = now.Add(pathfinderRetryGrace + pathfinderRandomWindow)
			announceTable[key] = entry
		}
		announceMu.Unlock()
		AnnLast = now
	}

	// ---- hashlist rotation ----
	if len(PacketHashSet) > HashlistMaxSize/2 {
		PacketHashSet2 = PacketHashSet
		PacketHashSet = make(map[hashKey]struct{})
	}

	// Cull invalidated path requests (Python: pending_prs_last_checked / pending_prs_check_interval).
	if now.Sub(pendingPRsLastChecked) > pendingPRsCheckInterval {
		pendingLocalPathRequestsMu.Lock()
		for key, desired := range pendingLocalPathRequests {
			if desired != nil {
				present := false
				for _, existing := range Interfaces {
					if existing == desired {
						present = true
						break
					}
				}
				if present {
					continue
				}
			}
			delete(pendingLocalPathRequests, key)
		}
		pendingLocalPathRequestsMu.Unlock()

		removed := 0
		discoveryPathRequestsMu.Lock()
		staleDiscovery := make([]hashKey, 0)
		for key, entry := range discoveryPathRequests {
			if entry == nil || (!entry.Timeout.IsZero() && !now.Before(entry.Timeout)) {
				Log(fmt.Sprintf("Waiting path request for %s timed out and was removed", PrettyHexRep(key[:])), LogDebug)
				staleDiscovery = append(staleDiscovery, key)
			}
		}
		for _, key := range staleDiscovery {
			delete(discoveryPathRequests, key)
			removed++
		}
		discoveryPathRequestsMu.Unlock()
		if removed > 0 {
			if removed == 1 {
				Log(fmt.Sprintf("Removed %d waiting path request", removed), LogExtreme)
			} else {
				Log(fmt.Sprintf("Removed %d waiting path requests", removed), LogExtreme)
			}
		}
		pendingPRsLastChecked = now
	}

	discoveryTagsMu.Lock()
	if len(discoveryPRTagFIFO) > maxPRTags {
		kept := append([]string(nil), discoveryPRTagFIFO[len(discoveryPRTagFIFO)-maxPRTags:len(discoveryPRTagFIFO)-1]...)
		discoveryPRTagFIFO = kept
		discoveryPRTags = make(map[string]struct{}, len(discoveryPRTagFIFO))
		for _, tag := range discoveryPRTagFIFO {
			discoveryPRTags[tag] = struct{}{}
		}
	}
	discoveryTagsMu.Unlock()

	// periodic culling of tables (Python: tables_last_culled / tables_cull_interval)
	if now.Sub(TablesLastCulled) > tablesCullInterval {
		tunnelsMu.Lock()
		removedTunnels := 0
		staleTunnelPaths := 0
		for key, te := range tunnels {
			if te == nil {
				delete(tunnels, key)
				removedTunnels++
				continue
			}
			// Python parity: remove expired tunnels.
			if !te.ExpiresAt.IsZero() && now.After(te.ExpiresAt) {
				Log(fmt.Sprintf("Tunnel %s timed out and was removed", PrettyHexRep([]byte(key))), LogExtreme)
				delete(tunnels, key)
				removedTunnels++
				continue
			}
			// Python parity: void (nil) the tunnel interface if its interface no
			// longer exists — but keep the tunnel and its paths.
			if te.Interface != nil {
				present := false
				for _, existing := range Interfaces {
					if existing == te.Interface {
						present = true
						break
					}
				}
				if !present {
					Log(fmt.Sprintf("Removing non-existent tunnel interface %v", te.Interface), LogExtreme)
					te.Interface = nil
					tunnels[key] = te
				}
			}
			// Python parity: cull individual stale tunnel paths
			// (tunnel_path_entry[0] + DESTINATION_TIMEOUT).
			if te.Paths != nil {
				for dstKey, pe := range te.Paths {
					if pe != nil && !pe.Timestamp.IsZero() && now.Sub(pe.Timestamp) > pathExpiration {
						Log(fmt.Sprintf("Tunnel path to %s timed out and was removed", PrettyHexRep([]byte(dstKey))), LogExtreme)
						delete(te.Paths, dstKey)
						staleTunnelPaths++
					}
				}
			}
		}
		tunnelsMu.Unlock()
		if removedTunnels > 0 {
			Log(fmt.Sprintf("Removed %d tunnels", removedTunnels), LogExtreme)
		}
		if staleTunnelPaths > 0 {
			if staleTunnelPaths == 1 {
				Log(fmt.Sprintf("Removed %d tunnel path", staleTunnelPaths), LogExtreme)
			} else {
				Log(fmt.Sprintf("Removed %d tunnel paths", staleTunnelPaths), LogExtreme)
			}
		}
		removed := false
		reverseRemovedN := 0

		reverseTableMu.Lock()
		for key, entry := range reverseTable {
			if entry == nil {
				delete(reverseTable, key)
				removed = true
				reverseRemovedN++
				continue
			}
			if now.Sub(entry.Timestamp) > reverseTimeout {
				delete(reverseTable, key)
				removed = true
				reverseRemovedN++
				continue
			}
			// Python removes entries if interfaces disappear.
			if entry.OutboundIf != nil {
				present := false
				for _, existing := range Interfaces {
					if existing == entry.OutboundIf {
						present = true
						break
					}
				}
				if !present {
					delete(reverseTable, key)
					removed = true
					reverseRemovedN++
					continue
				}
			}
			if entry.ReceivedIf != nil {
				present := false
				for _, existing := range Interfaces {
					if existing == entry.ReceivedIf {
						present = true
						break
					}
				}
				if !present {
					delete(reverseTable, key)
					removed = true
					reverseRemovedN++
				}
			}
		}
		reverseTableMu.Unlock()
		if reverseRemovedN > 0 {
			if reverseRemovedN == 1 {
				Log(fmt.Sprintf("Released 1 reverse table entry"), LogExtreme)
			} else {
				Log(fmt.Sprintf("Released %d reverse table entries", reverseRemovedN), LogExtreme)
			}
		}

		linkRemovedN := 0
		linkTableMu.Lock()
		for key, entry := range linkTable {
			if entry == nil {
				delete(linkTable, key)
				removed = true
				linkRemovedN++
				continue
			}
			if entry.Validated {
				// Python parity: for validated links, check interface presence AND timeout.
				if entry.NextHopInterface != nil {
					present := false
					for _, existing := range Interfaces {
						if existing == entry.NextHopInterface {
							present = true
							break
						}
					}
					if !present {
						delete(linkTable, key)
						removed = true
						linkRemovedN++
						continue
					}
				}
				if entry.ReceivedInterface != nil {
					present := false
					for _, existing := range Interfaces {
						if existing == entry.ReceivedInterface {
							present = true
							break
						}
					}
					if !present {
						delete(linkTable, key)
						removed = true
						linkRemovedN++
						continue
					}
				}
				if !entry.Timestamp.IsZero() && now.Sub(entry.Timestamp) > linkTimeout {
					delete(linkTable, key)
					removed = true
					linkRemovedN++
					continue
				}
			}
			// Proof timeouts expire pending (unvalidated) links.
			// Python parity: for unvalidated links, only check proof timeout.
			if !entry.Validated && !entry.ProofTimeout.IsZero() && now.After(entry.ProofTimeout) {
				if len(entry.DestinationHash) > 0 {
					var blockedIf *Interface
					pathRequestConditions := false
					pathRequestThrottle := false
					if dstKey, ok := func(hash []byte) (hashKey, bool) {
						if len(hash) < truncatedHashBytes {
							return hashKey{}, false
						}
						var key hashKey
						copy(key[:], hash[:truncatedHashBytes])
						return key, true
					}(entry.DestinationHash); ok {
						pathRequestMu.Lock()
						lastRequest, hasLast := lastPathRequest[dstKey]
						pathRequestMu.Unlock()
						if hasLast && !lastRequest.IsZero() && now.Sub(lastRequest) < pathRequestMinInterval {
							pathRequestThrottle = true
						}
					}

					switch {
					case !HasPath(entry.DestinationHash):
						Log(fmt.Sprintf("Trying to rediscover path for %s since an attempted link was never established, and path is now missing", PrettyHexRep(entry.DestinationHash)), LogDebug)
						pathRequestConditions = true
					case !pathRequestThrottle && entry.Hops == 0:
						Log(fmt.Sprintf("Trying to rediscover path for %s since an attempted local client link was never established", PrettyHexRep(entry.DestinationHash)), LogDebug)
						pathRequestConditions = true
					case !pathRequestThrottle && HopsTo(entry.DestinationHash) == 1:
						Log(fmt.Sprintf("Trying to rediscover path for %s since an attempted link was never established, and destination was previously local to an interface on this instance", PrettyHexRep(entry.DestinationHash)), LogDebug)
						pathRequestConditions = true
						blockedIf = entry.ReceivedInterface
						if TransportEnabled() && entry.ReceivedInterface != nil && entry.ReceivedInterface.Mode != InterfaceModeBoundary {
							MarkPathUnresponsive(entry.DestinationHash)
						}
					case !pathRequestThrottle && entry.Hops == 1:
						Log(fmt.Sprintf("Trying to rediscover path for %s since an attempted link was never established, and link initiator is local to an interface on this instance", PrettyHexRep(entry.DestinationHash)), LogDebug)
						pathRequestConditions = true
						blockedIf = entry.ReceivedInterface
						if TransportEnabled() && entry.ReceivedInterface != nil && entry.ReceivedInterface.Mode != InterfaceModeBoundary {
							MarkPathUnresponsive(entry.DestinationHash)
						}
					}

					if pathRequestConditions {
						if pathRequests != nil {
							if dstKey, ok := func(hash []byte) (hashKey, bool) {
								if len(hash) < truncatedHashBytes {
									return hashKey{}, false
								}
								var key hashKey
								copy(key[:], hash[:truncatedHashBytes])
								return key, true
							}(entry.DestinationHash); ok {
								if _, exists := pathRequests[dstKey]; !exists {
									pathRequests[dstKey] = blockedIf
								}
							}
						}

						if !TransportEnabled() {
							ExpirePath(entry.DestinationHash)
						}
					}
				}
				delete(linkTable, key)
				removed = true
				linkRemovedN++
				continue
			}
		}
		linkTableMu.Unlock()
		if removed {
			culled = true
		}
		if linkRemovedN > 0 {
			if linkRemovedN == 1 {
				Log(fmt.Sprintf("Released 1 link"), LogExtreme)
			} else {
				Log(fmt.Sprintf("Released %d links", linkRemovedN), LogExtreme)
			}
		}
		pathTableMu.Lock()
		removedPathTable := false
		pathRemovedN := 0
		for key, entry := range pathTable {
			if entry == nil {
				delete(pathTable, key)
				removedPathTable = true
				pathRemovedN++
				continue
			}
			// Python parity: expiry is always recomputed from Timestamp + timeout,
			// so that expire_path() (which sets Timestamp=0) causes immediate culling.
			var expires time.Time
			if !entry.Timestamp.IsZero() {
				if entry.RecvInterface != nil {
					switch entry.RecvInterface.Mode {
					case InterfaceModeAccessPoint:
						expires = entry.Timestamp.Add(apPathTime)
					case InterfaceModeRoaming:
						expires = entry.Timestamp.Add(roamingPathTime)
					default:
						expires = entry.Timestamp.Add(pathExpiration)
					}
				} else {
					expires = entry.Timestamp.Add(pathExpiration)
				}
			}
			// Fallback: use stored ExpiresAt if Timestamp is zero.
			if expires.IsZero() {
				expires = entry.ExpiresAt
			}
			if !expires.IsZero() && expires.Before(now) {
				Log(fmt.Sprintf("Path to %s timed out and was removed", PrettyHexRep(key[:])), LogDebug)
				delete(pathTable, key)
				removedPathTable = true
				pathRemovedN++
				continue
			}
			if entry.RecvInterface != nil {
				present := false
				for _, existing := range Interfaces {
					if existing == entry.RecvInterface {
						present = true
						break
					}
				}
				if !present {
					Log(fmt.Sprintf("Path to %s was removed since the attached interface no longer exists", PrettyHexRep(key[:])), LogDebug)
					delete(pathTable, key)
					removedPathTable = true
					pathRemovedN++
				}
			}
		}
		pathTableMu.Unlock()
		if removedPathTable {
			culled = true
		}
		if pathRemovedN > 0 {
			if pathRemovedN == 1 {
				Log(fmt.Sprintf("Removed 1 path"), LogExtreme)
			} else {
				Log(fmt.Sprintf("Removed %d paths", pathRemovedN), LogExtreme)
			}
		}
		pathTableMu.RLock()
		pathStatesMu.Lock()
		removedPathStates := false
		pathStatesRemovedN := 0
		for key := range pathStates {
			if _, ok := pathTable[key]; ok {
				continue
			}
			delete(pathStates, key)
			removedPathStates = true
			pathStatesRemovedN++
		}
		pathStatesMu.Unlock()
		pathTableMu.RUnlock()
		if removedPathStates {
			culled = true
		}
		if pathStatesRemovedN > 0 {
			if pathStatesRemovedN == 1 {
				Log(fmt.Sprintf("Removed 1 path state entry"), LogExtreme)
			} else {
				Log(fmt.Sprintf("Removed %d path state entries", pathStatesRemovedN), LogExtreme)
			}
		}
		TablesLastCulled = now
	}

	// periodic cleanup of reverse/link tables and caches
	if now.Sub(LastCacheCleaned) > cacheCleanInterval {
		cleanCache()
		culled = true
	}

	// Periodic persistence of transport tables (Python persist_data()).
	if now.Sub(lastTablesPersisted) > tablesPersistInterval {
		SavePathTable()
		savePacketHashlist()
		saveTunnelTable()
		lastTablesPersisted = now
	}

	// Similarly here: pending_local_path_requests, discovery_pr_tags, table culling
	// reverse_table, link_table, path_table, discovery_path_requests, tunnels, path_states
	// and interface.process_held_announces().
	if !InterfaceLastJobs.IsZero() && now.Sub(InterfaceLastJobs) <= InterfaceJobsInterval {
	} else {
		PrioritizeInterfaces()
		func() {
			defer func() {
				if r := recover(); r != nil {
					Log(fmt.Sprintf("Error while processing held per-interface announces: %v", r), LogWarning)
					Log("Postponing until next job run", LogWarning)
				}
			}()
			for _, ifc := range Interfaces {
				if ifc != nil {
					ifc.ProcessHeldAnnounces(PathfinderMaxHops)
				}
			}
			InterfaceLastJobs = now
		}()
	}

	if len(mgmtDestinations) > 0 && (LastMgmtAnnounce.IsZero() || now.Sub(LastMgmtAnnounce) > mgmtAnnounceInterval) {
		LastMgmtAnnounce = now
		mgmtAnnouncements = append([]*Destination(nil), mgmtDestinations...)
	}

	if now.Sub(blackholeLastChecked) > blackholeCheckInterval {
		blackholeMu.RLock()
		keys := make([]hashKey, 0, len(blackholedIdentities))
		for key := range blackholedIdentities {
			keys = append(keys, key)
		}
		blackholeMu.RUnlock()

		stale := make([]hashKey, 0, len(keys))
		for _, key := range keys {
			func() {
				defer func() {
					if r := recover(); r != nil {
						Log(fmt.Sprintf("Error while checking blackhole expiry for %s: %v", PrettyHexRep(key[:]), r), LogError)
					}
				}()
				blackholeMu.RLock()
				entry := blackholedIdentities[key]
				blackholeMu.RUnlock()
				if entry.Until != nil && !entry.Until.IsZero() && now.After(*entry.Until) {
					stale = append(stale, key)
				}
			}()
		}

		removed := 0
		blackholeMu.Lock()
		for _, key := range stale {
			if _, exists := blackholedIdentities[key]; exists {
				delete(blackholedIdentities, key)
				removed++
			}
		}
		blackholeMu.Unlock()
		if removed > 0 {
			if removed == 1 {
				Log(fmt.Sprintf("Removed 1 blackholed identity"), LogVerbose)
			} else {
				Log(fmt.Sprintf("Removed %d blackholed identities", removed), LogVerbose)
			}
			culled = true
		}
		blackholeLastChecked = now
	}

	if shouldGC || culled {
		runtime.GC()
	}
}

// VoidTunnelInterface mirrors Python Transport.void_tunnel_interface().
func VoidTunnelInterface(tunnelID []byte) {
	key := string(tunnelID)
	tunnelsMu.Lock()
	if te := tunnels[key]; te != nil {
		Log(fmt.Sprintf("Voiding tunnel interface %v", te.Interface), LogExtreme)
		te.Interface = nil
		tunnels[key] = te
	}
	tunnelsMu.Unlock()
}

// DropAnnounceQueues mirrors Python Transport.drop_announce_queues().
func DropAnnounceQueues() {
	for _, ifc := range Interfaces {
		if ifc == nil {
			continue
		}
		ifc.DropAnnounceQueue()
	}
}

func remoteStatusHandler(_ string, data any, _ []byte, _ []byte, remoteIdentity *Identity, _ time.Time) any {
	if remoteIdentity == nil || Owner == nil {
		return nil
	}
	var response []any
	defer func() {
		if r := recover(); r != nil {
			Log(fmt.Sprintf("An error occurred while processing remote status request from %v", remoteIdentity), LogError)
			Log(fmt.Sprintf("The contained exception was: %v", r), LogError)
			response = nil
		}
	}()
	args, ok := data.([]any)
	if !ok || len(args) == 0 {
		return nil
	}
	response = []any{Owner.GetInterfaceStats()}
	if includeLinks, ok := args[0].(bool); ok && includeLinks {
		response = append(response, Owner.GetLinkCount())
	}
	return response
}

func remotePathHandler(_ string, data any, _ []byte, _ []byte, remoteIdentity *Identity, _ time.Time) any {
	if remoteIdentity == nil || Owner == nil {
		return nil
	}
	var response any
	defer func() {
		if r := recover(); r != nil {
			Log(fmt.Sprintf("An error occurred while processing remote status request from %v", remoteIdentity), LogError)
			Log(fmt.Sprintf("The contained exception was: %v", r), LogError)
			response = nil
		}
	}()
	if args, ok := data.([]any); ok && len(args) > 0 {
		command, _ := args[0].(string)
		var destHash []byte
		var maxHops *int
		if len(args) > 1 {
			if hash, ok := args[1].([]byte); ok {
				destHash = hash
			}
		}
		if len(args) > 2 {
			maxHops = func(v any) *int {
				switch val := v.(type) {
				case int:
					return &val
				case int8:
					vv := int(val)
					return &vv
				case int16:
					vv := int(val)
					return &vv
				case int32:
					vv := int(val)
					return &vv
				case int64:
					vv := int(val)
					return &vv
				case uint:
					vv := int(val)
					return &vv
				case uint8:
					vv := int(val)
					return &vv
				case uint16:
					vv := int(val)
					return &vv
				case uint32:
					vv := int(val)
					return &vv
				case uint64:
					vv := int(val)
					return &vv
				case float32:
					vv := int(val)
					return &vv
				case float64:
					vv := int(val)
					return &vv
				default:
					return nil
				}
			}(args[2])
		}

		switch command {
		case "table":
			table := Owner.GetPathTable(maxHops)
			filtered := make([]map[string]any, 0, len(table))
			for _, entry := range table {
				raw, _ := entry["hash"].([]byte)
				if len(destHash) == 0 || bytes.Equal(raw, destHash) {
					filtered = append(filtered, entry)
				}
			}
			response = filtered
		case "rates":
			table := Owner.GetRateTable()
			filtered := make([]map[string]any, 0, len(table))
			for _, entry := range table {
				raw, _ := entry["hash"].([]byte)
				if len(destHash) == 0 || bytes.Equal(raw, destHash) {
					filtered = append(filtered, entry)
				}
			}
			response = filtered
		}

		return response
	}
	return nil
}

func pathRequestHandler(data []byte, packet *Packet) {
	defer func() {
		if r := recover(); r != nil {
			Log(fmt.Sprintf("Error while handling path request. The contained exception was: %v", r), LogError)
		}
	}()
	if len(data) < truncatedHashBytes {
		return
	}

	destinationHash := append([]byte(nil), data[:truncatedHashBytes]...)
	var requestingTransportID []byte
	var tagBytes []byte

	if len(data) > truncatedHashBytes*2 {
		requestingTransportID = append([]byte(nil), data[truncatedHashBytes:truncatedHashBytes*2]...)
		tagBytes = data[truncatedHashBytes*2:]
	} else if len(data) > truncatedHashBytes {
		tagBytes = data[truncatedHashBytes:]
	}

	if len(tagBytes) == 0 {
		Log(fmt.Sprintf("Ignoring tagless path request for %s", PrettyHexRep(destinationHash)), LogDebug)
		return
	}
	if len(tagBytes) > truncatedHashBytes {
		tagBytes = tagBytes[:truncatedHashBytes]
	}

	unique := append(append([]byte(nil), destinationHash...), tagBytes...)
	uniqueKey := string(unique)
	discoveryTagsMu.Lock()
	if _, ok := discoveryPRTags[uniqueKey]; ok {
		discoveryTagsMu.Unlock()
		Log(fmt.Sprintf("Ignoring duplicate path request for %s with tag %s", PrettyHexRep(destinationHash), PrettyHexRep(unique)), LogDebug)
		return
	}
	discoveryPRTags[uniqueKey] = struct{}{}
	discoveryPRTagFIFO = append(discoveryPRTagFIFO, uniqueKey)
	discoveryTagsMu.Unlock()

	var attached *Interface
	if packet != nil {
		attached = packet.ReceivingInterface
	}
	pathRequest(destinationHash, FromLocalClient(packet), attached, requestingTransportID, tagBytes)
}

func pathRequest(destinationHash []byte, isFromLocalClient bool, attachedInterface *Interface, requestorTransportID []byte, tag []byte) {
	shouldSearchForUnknown := false
	if attachedInterface != nil {
		if TransportEnabled() {
			switch attachedInterface.Mode {
			case InterfaceModeAccessPoint, InterfaceModeGateway, InterfaceModeRoaming:
				shouldSearchForUnknown = true
			}
		}
	}
	Log(fmt.Sprintf("Path request for %s%s", PrettyHexRep(destinationHash), fmt.Sprint(attachedInterface)), LogDebug)

	// Python parity: If the destination exists on a local client, but it has not
	// been announced yet, remember which external interface wants it so the later
	// PATH_RESPONSE can be forwarded to that interface.
	var pathEntry *PathEntry
	if key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(destinationHash); ok {
		pathTableMu.RLock()
		pathEntry = pathTable[key]
		pathTableMu.RUnlock()
	}
	if len(LocalClientInterfaces) > 0 && pathEntry != nil && pathEntry.RecvInterface != nil && IsLocalClientInterface(pathEntry.RecvInterface) {
		if key, ok := func(hash []byte) (hashKey, bool) {
			if len(hash) < truncatedHashBytes {
				return hashKey{}, false
			}
			var key hashKey
			copy(key[:], hash[:truncatedHashBytes])
			return key, true
		}(destinationHash); ok {
			pendingLocalPathRequestsMu.Lock()
			pendingLocalPathRequests[key] = attachedInterface
			pendingLocalPathRequestsMu.Unlock()
		}
	}

	var dst *Destination
	for _, candidate := range Destinations {
		if candidate != nil && len(candidate.Hash) > 0 && bytes.Equal(candidate.Hash, destinationHash) {
			dst = candidate
			break
		}
	}
	if dst != nil {
		dst.Announce(nil, true, attachedInterface, tag, true)
		Log(fmt.Sprintf("Answering path request for %s on %s, destination is local to this system", PrettyHexRep(destinationHash), fmt.Sprint(attachedInterface)), LogDebug)
		return
	}

	if (TransportEnabled() || isFromLocalClient) && pathEntry != nil {
		entry := pathEntry

		// Python parity: get the cached packet first, then check nil, then roaming, then unpack.
		announcePacket := getCachedPacket(entry.PacketHash, "announce")
		if announcePacket == nil {
			Log(fmt.Sprintf("Could not retrieve announce packet from cache while answering path request for %s", PrettyHexRep(destinationHash)), LogError)
			return
		}

		if attachedInterface != nil && attachedInterface.Mode == InterfaceModeRoaming && entry.RecvInterface == attachedInterface {
			Log(fmt.Sprintf("Not answering path request on roaming-mode interface, since next hop is on same roaming-mode interface"), LogDebug)
			return
		}

		if !announcePacket.Unpack() {
			return
		}

		if len(requestorTransportID) == truncatedHashBytes && len(entry.NextHop) == truncatedHashBytes && bytes.Equal(entry.NextHop, requestorTransportID) {
			Log(fmt.Sprintf("Not answering path request for %s%s, since next hop is the requestor", PrettyHexRep(destinationHash), fmt.Sprint(attachedInterface)), LogDebug)
			return
		}
		Log(fmt.Sprintf("Answering path request for %s%s, path is known", PrettyHexRep(destinationHash), fmt.Sprint(attachedInterface)), LogDebug)

		now := time.Now()
		delay := pathRequestGrace
		switch {
		case isFromLocalClient:
			delay = 0
		case entry.RecvInterface != nil && IsLocalClientInterface(entry.RecvInterface):
			Log(fmt.Sprintf("Path request destination %s is on a local client interface, rebroadcasting immediately", PrettyHexRep(destinationHash)), LogExtreme)
			delay = 0
		case attachedInterface != nil && attachedInterface.Mode == InterfaceModeRoaming:
			delay += pathRequestRoamingGrace
		}

		dest := &Destination{
			Type:      DestinationSINGLE,
			Direction: DestinationOUT,
			Hash:      append([]byte(nil), destinationHash...),
			HexHash:   PrettyHexRep(destinationHash),
		}
		resp := NewPacket(
			dest,
			append([]byte(nil), announcePacket.Data...),
			PacketANNOUNCE,
			PacketPATH_RESPONSE,
			TransportDirect,
			HeaderType2,
			append([]byte(nil), TransportIdentity.Hash...),
			attachedInterface,
			false,
			announcePacket.ContextFlag,
		)
		if resp == nil {
			Log(fmt.Sprintf("Could not construct path response packet for %s", PrettyHexRep(destinationHash)), LogDebug)
			return
		}
		h := entry.Hops
		if h < 0 {
			h = 0
		}
		if h > 255 {
			h = 255
		}
		resp.Hops = uint8(h)
		resp.DestinationHash = append([]byte(nil), destinationHash...)
		resp.DestinationType = byte(DestinationSINGLE)

		if key, ok := func(hash []byte) (hashKey, bool) {
			if len(hash) < truncatedHashBytes {
				return hashKey{}, false
			}
			var key hashKey
			copy(key[:], hash[:truncatedHashBytes])
			return key, true
		}(destinationHash); ok {
			announceMu.Lock()
			if existing := announceTable[key]; existing != nil {
				heldAnnounces[key] = &heldAnnounce{Entry: existing}
			}
			queued := &announceEntry{
				Packet:            resp,
				Next:              now.Add(delay),
				Retries:           pathfinderRetryLimit,
				Timestamp:         now,
				Expires:           now.Add(announceQueueTTL),
				LocalRebroadcasts: 0,
				BlockRebroadcasts: true,
				AttachedInterface: attachedInterface,
			}
			announceTable[key] = queued
			announceMu.Unlock()
		}
		return
	}

	if isFromLocalClient {
		Log(fmt.Sprintf("Forwarding path request from local client for %s%s to all other interfaces", PrettyHexRep(destinationHash), fmt.Sprint(attachedInterface)), LogDebug)
		requestTag := IdentityGetRandomHash()
		for _, ifc := range Interfaces {
			if ifc == nil || ifc == attachedInterface {
				continue
			}
			RequestPath(destinationHash, ifc, requestTag, false)
		}
		return
	}

	if shouldSearchForUnknown && attachedInterface != nil {
		if key, ok := func(hash []byte) (hashKey, bool) {
			if len(hash) < truncatedHashBytes {
				return hashKey{}, false
			}
			var key hashKey
			copy(key[:], hash[:truncatedHashBytes])
			return key, true
		}(destinationHash); ok {
			now := time.Now()
			discoveryPathRequestsMu.Lock()
			if entry := discoveryPathRequests[key]; entry != nil {
				discoveryPathRequestsMu.Unlock()
				Log(fmt.Sprintf("There is already a waiting path request for %s on behalf of path request on %s",
					PrettyHexRep(destinationHash), fmt.Sprint(attachedInterface)), LogDebug,
				)
				return
			}
			discoveryPathRequests[key] = &discoveryPathRequest{
				Timeout:             now.Add(time.Duration(TransportPathRequestTimeout * float64(time.Second))),
				RequestingInterface: attachedInterface,
			}
			discoveryPathRequestsMu.Unlock()
			Log(fmt.Sprintf("Attempting to discover unknown path to %s on behalf of path request on %s",
				PrettyHexRep(destinationHash), fmt.Sprint(attachedInterface)), LogDebug,
			)
			for _, ifc := range Interfaces {
				if ifc == nil || ifc == attachedInterface {
					continue
				}
				RequestPath(destinationHash, ifc, tag, true)
			}
			return
		}
	}

	if !isFromLocalClient && len(LocalClientInterfaces) > 0 {
		Log(fmt.Sprintf("Forwarding path request for %s%s to local clients", PrettyHexRep(destinationHash), fmt.Sprint(attachedInterface)), LogDebug)
		for _, ifc := range LocalClientInterfaces {
			if ifc == nil {
				continue
			}
			RequestPath(destinationHash, ifc, nil, false)
		}
		return
	}
	Log(fmt.Sprintf("Ignoring path request for %s%s, no path known", PrettyHexRep(destinationHash), fmt.Sprint(attachedInterface)), LogDebug)
}

func tunnelSynthesizeHandler(data []byte, packet *Packet) {
	defer func() {
		if r := recover(); r != nil {
			Log(fmt.Sprintf("An error occurred while validating tunnel establishment packet."), LogDebug)
			Log(fmt.Sprintf("The contained exception was: %v", r), LogDebug)
		}
	}()
	// Python expected_length:
	// KEYSIZE//8 (64) + HASHLENGTH//8 (32) + TRUNCATED_HASHLENGTH//8 (16) + SIGLENGTH//8 (64) = 176
	expected := (IdentityKeySize / 8) + (IdentityHashLength / 8) + (ReticulumTruncatedHashLength / 8) + (IdentitySigLength / 8)
	if len(data) != expected {
		return
	}

	off := 0
	pubKey := data[off : off+(IdentityKeySize/8)]
	off += IdentityKeySize / 8
	ifaceHash := data[off : off+(IdentityHashLength/8)]
	off += IdentityHashLength / 8
	randomHash := data[off : off+(ReticulumTruncatedHashLength/8)]
	off += ReticulumTruncatedHashLength / 8
	signature := data[off:]

	tunnelIDData := append(append([]byte(nil), pubKey...), ifaceHash...)
	tunnelID := FullHash(tunnelIDData)
	signedData := append(append([]byte(nil), tunnelIDData...), randomHash...)

	remote := &Identity{}
	if err := remote.LoadPublicKey(pubKey); err != nil {
		panic(err)
	}
	if !remote.Validate(signature, signedData) {
		return
	}

	handleTunnel(tunnelID, packet.ReceivingInterface)
}

// SynthesizeTunnel mirrors Python Transport.synthesize_tunnel(interface). It sends a
// tunnel establishment packet attached to the specified interface.
func SynthesizeTunnel(ifc *Interface) {
	pub := TransportIdentity.GetPublicKey()
	ifaceHash := ifc.Hash()
	randomHash := IdentityGetRandomHash()

	tunnelIDData := make([]byte, 0, len(pub)+len(ifaceHash))
	tunnelIDData = append(tunnelIDData, pub...)
	tunnelIDData = append(tunnelIDData, ifaceHash...)

	signedData := make([]byte, 0, len(tunnelIDData)+len(randomHash))
	signedData = append(signedData, tunnelIDData...)
	signedData = append(signedData, randomHash...)

	sig, err := TransportIdentity.Sign(signedData)
	if err != nil {
		panic(err)
	}

	data := make([]byte, 0, len(signedData)+len(sig))
	data = append(data, signedData...)
	data = append(data, sig...)

	dst, err := NewDestination(nil, DestinationOUT, DestinationPLAIN, "rnstransport", "tunnel", "synthesize")
	if err != nil {
		panic(err)
	}
	p := NewPacket(
		dst,
		data,
		PacketTypeData,
		PacketCtxNone,
		Broadcast,
		HeaderType1,
		nil,
		ifc,
		false,
		FlagUnset,
	)
	_ = p.Send()
	ifc.WantsTunnel = false
}

func handleTunnel(tunnelID []byte, ifc *Interface) {
	expiresAt := time.Now().Add(pathExpiration)

	key := string(tunnelID)
	tunnelsMu.Lock()
	existing := tunnels[key]
	if existing != nil {
		Log(fmt.Sprintf("Tunnel endpoint %s reappeared. Restoring paths...", PrettyHexRep(tunnelID)), LogDebug)
		existing.Interface = ifc
		existing.ExpiresAt = expiresAt
		ifc.TunnelID = append([]byte(nil), tunnelID...)
		if existing.Paths == nil {
			existing.Paths = make(map[string]*tunnelPathEntry)
		}
		tunnels[key] = existing
		tunnelsMu.Unlock()

		now := time.Now()
		deprecated := make([]string, 0)

		for dstKey, entry := range existing.Paths {
			if entry == nil {
				deprecated = append(deprecated, dstKey)
				continue
			}
			if !entry.ExpiresAt.IsZero() && now.After(entry.ExpiresAt) {
				deprecated = append(deprecated, dstKey)
				continue
			}
			dstHash := []byte(dstKey)
			if len(dstHash) != truncatedHashBytes {
				deprecated = append(deprecated, dstKey)
				continue
			}

			pathKey, ok := func(hash []byte) (hashKey, bool) {
				if len(hash) < truncatedHashBytes {
					return hashKey{}, false
				}
				var key hashKey
				copy(key[:], hash[:truncatedHashBytes])
				return key, true
			}(dstHash)
			if !ok {
				continue
			}
			randomBlobs := func(in [][]byte) [][]byte {
				if len(in) == 0 {
					return nil
				}
				out := make([][]byte, len(in))
				for i, blob := range in {
					out[i] = append([]byte(nil), blob...)
				}
				return out
			}(entry.RandomBlobs)
			if len(randomBlobs) > 0 {
				seen := make(map[string]struct{}, len(randomBlobs))
				deduped := make([][]byte, 0, len(randomBlobs))
				for _, blob := range randomBlobs {
					if len(blob) == 0 {
						continue
					}
					key := string(blob)
					if _, ok := seen[key]; ok {
						continue
					}
					seen[key] = struct{}{}
					deduped = append(deduped, append([]byte(nil), blob...))
				}
				if len(deduped) > maxRandomBlobs {
					deduped = deduped[len(deduped)-maxRandomBlobs:]
				}
				randomBlobs = deduped
			}
			newEntry := &PathEntry{
				NextHop:       append([]byte(nil), entry.ReceivedFrom...),
				RecvInterface: ifc,
				Hops:          entry.Hops,
				Timestamp:     now,
				ExpiresAt:     entry.ExpiresAt,
				RandomBlobs:   randomBlobs,
				PacketHash:    append([]byte(nil), entry.PacketHash...),
			}

			shouldAdd := false
			var old *PathEntry
			if pathKey, ok := func(hash []byte) (hashKey, bool) {
				if len(hash) < truncatedHashBytes {
					return hashKey{}, false
				}
				var key hashKey
				copy(key[:], hash[:truncatedHashBytes])
				return key, true
			}(dstHash); ok {
				pathTableMu.RLock()
				old = pathTable[pathKey]
				pathTableMu.RUnlock()
			}
			if old != nil {
				if newEntry.Hops <= old.Hops || now.After(old.ExpiresAt) {
					shouldAdd = true
				} else {
					Log(fmt.Sprintf("Did not restore path to %s because a newer path with fewer hops exist", PrettyHexRep(dstHash)), LogDebug)
				}
			} else {
				if now.Before(entry.ExpiresAt) {
					shouldAdd = true
				} else {
					Log(fmt.Sprintf("Did not restore path to %s because it has expired", PrettyHexRep(dstHash)), LogDebug)
				}
			}

			if shouldAdd {
				pathTableMu.Lock()
				pathTable[pathKey] = newEntry
				pathTableMu.Unlock()
				Log(fmt.Sprintf("Restored path to %s is now %d hops away via %s on %v",
					PrettyHexRep(dstHash), newEntry.Hops, PrettyHexRep(newEntry.NextHop), ifc), LogDebug,
				)
			} else {
				deprecated = append(deprecated, dstKey)
			}
		}

		if len(deprecated) > 0 {
			tunnelsMu.Lock()
			for _, k := range deprecated {
				Log(fmt.Sprintf("Removing path to %s from tunnel %s", PrettyHexRep([]byte(k)), PrettyHexRep(existing.ID)), LogDebug)
				delete(existing.Paths, k)
			}
			tunnelsMu.Unlock()
		}
		return
	}
	Log(fmt.Sprintf("Tunnel endpoint %s established.", PrettyHexRep(tunnelID)), LogDebug)
	ifc.TunnelID = append([]byte(nil), tunnelID...)
	te := &tunnelEntry{
		ID:        append([]byte(nil), tunnelID...),
		Interface: ifc,
		ExpiresAt: expiresAt,
		Paths:     make(map[string]*tunnelPathEntry),
	}
	tunnels[key] = te
	tunnelsMu.Unlock()
}

// -------- interface transmission (IFAC) --------

func transmit(ifc *Interface, raw []byte) {
	defer func() {
		if r := recover(); r != nil {
			Log(fmt.Sprintf("Error while transmitting on %v. The contained exception was: %v", ifc, r), LogError)
		}
	}()

	if ifc.IFACIdentity != nil {
		// ifac = Sign(raw)[-ifacSize:]
		sig, err := ifc.IFACIdentity.Sign(raw)
		if err != nil {
			panic(err)
		}
		ifac := sig[len(sig)-ifc.IFACSize:]

		// mask = hkdf(len(raw)+ifacSize, ifac, ifacKey)
		mask, err := Cryptography.HKDF(len(raw)+ifc.IFACSize, ifac, ifc.IFACKey, nil)
		if err != nil {
			panic(err)
		}

		// new header with IFAC flag
		newHeader := []byte{raw[0] | 0x80, raw[1]}
		newRaw := append(newHeader, ifac...)
		newRaw = append(newRaw, raw[2:]...)

		masked := make([]byte, len(newRaw))
		for i, b := range newRaw {
			switch {
			case i == 0:
				masked[i] = (b ^ mask[i]) | 0x80
			case i == 1 || i > ifc.IFACSize+1:
				masked[i] = b ^ mask[i]
			default:
				masked[i] = b
			}
		}
		ifc.ProcessOutgoing(masked)
		return
	}

	ifc.ProcessOutgoing(raw)
}

// -------- outbound packets --------

func Outbound(p *Packet) bool {
	jobsMu.RLock()
	var sent bool
	defer func() {
		jobsMu.RUnlock()
		if sent && Owner != nil {
			Owner.ShouldPersistData()
		}
	}()

	genReceipt := false
	outboundTime := time.Now()
	if p.CreateReceipt &&
		p.Type == PacketData &&
		p.Destination.Type != DestPlain &&
		!(p.Context >= PacketKeepalive && p.Context <= PacketLRProof) &&
		!(p.Context >= PacketResource && p.Context <= PacketResourceRCL) {
		genReceipt = true
	}

	packetSent := func(p *Packet) {
		p.Sent = true
		p.SentAt = time.Now()
		if genReceipt {
			rc := NewPacketReceipt(p)
			p.Receipt = rc
			receiptsMu.Lock()
			Receipts = append(Receipts, rc)
			receiptsMu.Unlock()
		}
	}

	linkAttachedInterface := func() *Interface {
		if p == nil || p.Link == nil {
			return nil
		}
		if attached, ok := p.Link.AttachedInterface.(*Interface); ok {
			return attached
		}
		return nil
	}

	// is the path known?
	sendBroadcast := true
	if p.Type != PacketAnnounce &&
		p.Destination.Type != DestPlain &&
		p.Destination.Type != DestGroup &&
		p.Link == nil &&
		HasPath(p.DestinationHash) {

		var entry *PathEntry
		if key, ok := func(hash []byte) (hashKey, bool) {
			if len(hash) < truncatedHashBytes {
				return hashKey{}, false
			}
			var key hashKey
			copy(key[:], hash[:truncatedHashBytes])
			return key, true
		}(p.DestinationHash); ok {
			pathTableMu.RLock()
			entry = pathTable[key]
			pathTableMu.RUnlock()
		}
		if entry == nil {
			Log(fmt.Sprintf("Dropped packet since path table entry disappeared during outbound processing"), LogWarning)
			return false
		}
		sendBroadcast = false
		outIfc := entry.RecvInterface
		hops := entry.Hops

		connectedShared := Owner != nil && Owner.IsConnectedToSharedInstance
		// Inject multi-hop packets into transport by rewriting to HEADER_2 with
		// the next-hop transport ID. Single-hop packets are transmitted directly.
		if hops > 1 {
			if p.HeaderType == Header1 {
				flags := byte(Header2)<<6 |
					byte(TransportDirect)<<4 |
					(p.Flags & 0x0f)
				newRaw := make([]byte, 0, len(p.Raw)+truncatedHashBytes)
				newRaw = append(newRaw, flags)
				newRaw = append(newRaw, p.Raw[1])
				newRaw = append(newRaw, entry.NextHop...)
				newRaw = append(newRaw, p.Raw[2:]...)
				packetSent(p)
				transmit(outIfc, newRaw)
				entry.Timestamp = time.Now()
				sent = true
			}
		} else if hops == 1 && connectedShared {
			if p.HeaderType == Header1 {
				flags := byte(Header2)<<6 |
					byte(TransportDirect)<<4 |
					(p.Flags & 0x0f)
				newRaw := make([]byte, 0, len(p.Raw)+truncatedHashBytes)
				newRaw = append(newRaw, flags)
				newRaw = append(newRaw, p.Raw[1])
				newRaw = append(newRaw, entry.NextHop...)
				newRaw = append(newRaw, p.Raw[2:]...)
				packetSent(p)
				transmit(outIfc, newRaw)
				entry.Timestamp = time.Now()
				sent = true
			}
		} else {
			// single hop: send directly
			packetSent(p)
			transmit(outIfc, p.Raw)
			sent = true
		}
	}

	// Python parity: LINK-type packets (with attached link interface) are transmitted
	// directly via the link's attached interface, bypassing the broadcast loop.
	// This covers LRPROOF and other link-level packets sent back to the peer.
	if sendBroadcast {
		if attached := linkAttachedInterface(); attached != nil {
			AddPacketHash(p.PacketHash)
			transmit(attached, p.Raw)
			packetSent(p)
			sent = true
			return sent
		}
		// Fast path: Python's broadcast loop only sends on interfaces where
		// interface.OUT is True AND interface == packet.attached_interface.
		// This early return avoids iterating all interfaces for that common case.
		if p.AttachedInterface != nil {
			AddPacketHash(p.PacketHash)
			transmit(p.AttachedInterface, p.Raw)
			packetSent(p)
			sent = true
			return sent
		}
	}

	if sendBroadcast {
		// path unknown: broadcast via all OUT interfaces
		storedHash := false

		for _, ifc := range Interfaces {
			if !ifc.OUT {
				continue
			}
			shouldSend := true

			// Python parity: don't transmit to a LINK-type destination if the link is CLOSED.
			if p.Destination != nil && p.Destination.Type == DestLink && p.Link != nil && p.Link.Status == LinkClosed {
				shouldSend = false
			}

			if attached := linkAttachedInterface(); attached != nil && ifc != attached {
				shouldSend = false
			}

			if p.AttachedInterface != nil && ifc != p.AttachedInterface {
				shouldSend = false
			}

			// announce logic with AP/ROAMING/BOUNDARY and queues —
			// all mode/cap checks only apply when packet is not attached to a specific interface.
			if p.Type == PacketAnnounce {
				if p.AttachedInterface == nil {
					switch ifc.Mode {
					case InterfaceModeAccessPoint:
						Log(fmt.Sprintf("Blocking announce broadcast on %v due to AP mode", ifc), LogExtreme)
						shouldSend = false
					case InterfaceModeRoaming:
						var dst *Destination
						destinationsMu.Lock()
						for _, candidate := range Destinations {
							if candidate != nil && len(candidate.Hash) > 0 && bytes.Equal(candidate.Hash, p.DestinationHash) {
								dst = candidate
								break
							}
						}
						destinationsMu.Unlock()
						if dst == nil {
							fromIfc := NextHopInterface(p.DestinationHash)
							if fromIfc == nil {
								Log(fmt.Sprintf("Blocking announce broadcast on %v since next hop interface doesn't exist", ifc), LogExtreme)
								shouldSend = false
							} else {
								switch fromIfc.Mode {
								case InterfaceModeRoaming:
									Log(fmt.Sprintf("Blocking announce broadcast on %v due to roaming-mode next-hop interface", ifc), LogExtreme)
									shouldSend = false
								case InterfaceModeBoundary:
									Log(fmt.Sprintf("Blocking announce broadcast on %v due to boundary-mode next-hop interface", ifc), LogExtreme)
									shouldSend = false
								}
							}
						}
					case InterfaceModeBoundary:
						var dst *Destination
						destinationsMu.Lock()
						for _, candidate := range Destinations {
							if candidate != nil && len(candidate.Hash) > 0 && bytes.Equal(candidate.Hash, p.DestinationHash) {
								dst = candidate
								break
							}
						}
						destinationsMu.Unlock()
						if dst == nil {
							fromIfc := NextHopInterface(p.DestinationHash)
							if fromIfc == nil {
								Log(fmt.Sprintf("Blocking announce broadcast on %v since next hop interface doesn't exist", ifc), LogExtreme)
								shouldSend = false
							} else {
								if fromIfc.Mode == InterfaceModeRoaming {
									Log(fmt.Sprintf("Blocking announce broadcast on %v due to roaming-mode next-hop interface", ifc), LogExtreme)
									shouldSend = false
								}
							}
						}
					default:
						// Normal interface: apply announce cap (only for relayed announces, not local).
						if int(p.Hops) > 0 {
							cap := ifc.AnnounceCap
							if cap <= 0 && ifaces.DefaultAnnounceCapProvider != nil {
								cap = ifaces.DefaultAnnounceCapProvider()
							}
							if cap > 0 {
								ifc.ICMu.Lock()
								queuedAnnounces := len(ifc.AnnounceQueue) > 0
								allowedAt := ifc.AnnounceAllowedAt
								shouldQueue := queuedAnnounces || (!allowedAt.IsZero() && outboundTime.Before(allowedAt)) || ifc.Bitrate == 0
								if !shouldQueue {
									txTime := (float64(len(p.Raw)) * 8.0) / float64(ifc.Bitrate)
									waitTime := txTime / cap
									if waitTime < 0 {
										waitTime = 0
									}
									ifc.AnnounceAllowedAt = outboundTime.Add(time.Duration(waitTime * float64(time.Second)))
								} else {
									shouldSend = false
									if len(ifc.AnnounceQueue) < MAX_QUEUED_ANNOUNCES {
										emitted := AnnounceEmitted(p)
										alreadyQueued := false
										var existing *ifaces.AnnounceQueueEntry
										for idx := range ifc.AnnounceQueue {
											if bytes.Equal(ifc.AnnounceQueue[idx].Destination, p.DestinationHash) {
												alreadyQueued = true
												existing = &ifc.AnnounceQueue[idx]
											}
										}
										if alreadyQueued {
											if emitted > existing.Emitted {
												existing.Time = outboundTime
												existing.Hops = int(p.Hops)
												existing.Emitted = emitted
												existing.Raw = append([]byte(nil), p.Raw...)
											}
										} else {
											entry := ifaces.AnnounceQueueEntry{
												Time:        outboundTime,
												Destination: append([]byte(nil), p.DestinationHash...),
												Hops:        int(p.Hops),
												Emitted:     emitted,
												Raw:         append([]byte(nil), p.Raw...),
											}
											queuedBefore := len(ifc.AnnounceQueue) > 0
											ifc.AnnounceQueue = append(ifc.AnnounceQueue, entry)
											waitUntil := time.Until(ifc.AnnounceAllowedAt)
											if waitUntil < 0 {
												waitUntil = 0
											}
											if !queuedBefore {
												time.AfterFunc(waitUntil, func() {
													ifc.ProcessAnnounceQueue()
												})
											}
											var waitTimeStr string
											if waitUntil.Seconds() < 1 {
												waitTimeStr = fmt.Sprintf("%.2fms", waitUntil.Seconds()*1000)
											} else {
												waitTimeStr = fmt.Sprintf("%.2fs", waitUntil.Seconds())
											}
											Log(fmt.Sprintf("Added announce to queue (height %d) on %v for processing in %s", len(ifc.AnnounceQueue), ifc, waitTimeStr), LogExtreme)
										}
									}
								}
								ifc.ICMu.Unlock()
							}
						}
					}
				}
			}

			if !shouldSend {
				continue
			}

			if !storedHash {
				AddPacketHash(p.PacketHash)
				storedHash = true
			}
			transmit(ifc, p.Raw)
			if p.Type == PacketAnnounce {
				ifc.SentAnnounce()
			}
			packetSent(p)
			sent = true
		}
	}

	return sent
}

// -------- inbound filtering --------

func PacketFilter(p *Packet) bool {
	// shared instance decides for itself
	if Owner != nil && Owner.IsConnectedToSharedInstance {
		return true
	}

	// foreign transport_id (except announces)
	if p.TransportID != nil && p.Type != PacketAnnounce {
		if TransportIdentity != nil && !bytes.Equal(p.TransportID, TransportIdentity.Hash) {
			Log(fmt.Sprintf("Ignored packet %s in transport for other transport instance",
				PrettyHexRep(p.PacketHash)), LogExtreme,
			)
			return false
		}
	}

	switch p.Context {
	case PacketKeepalive,
		PacketResourceReq,
		PacketResourcePrf,
		PacketResource,
		PacketCacheRequest,
		PacketChannel:
		return true
	}

	if p.DestinationType == DestPlain {
		if p.Type != PacketAnnounce {
			if p.Hops > 1 {
				Log(fmt.Sprintf("Dropped PLAIN packet %s with %d hops",
					PrettyHexRep(p.PacketHash), p.Hops), LogDebug,
				)
				return false
			}
			return true
		}
		Log("Dropped invalid PLAIN announce packet", LogDebug)
		return false
	}

	if p.DestinationType == DestGroup {
		if p.Type != PacketAnnounce {
			if p.Hops > 1 {
				Log(fmt.Sprintf("Dropped GROUP packet %s with %d hops",
					PrettyHexRep(p.PacketHash), p.Hops), LogDebug,
				)
				return false
			}
			return true
		}
		Log("Dropped invalid GROUP announce packet", LogDebug)
		return false
	}

	if len(p.PacketHash) < truncatedHashBytes {
		return true
	}
	var packetHashKey hashKey
	copy(packetHashKey[:], p.PacketHash[:truncatedHashBytes])
	packetHashMu.RLock()
	_, seen := PacketHashSet[packetHashKey]
	if !seen {
		_, seen = PacketHashSet2[packetHashKey]
	}
	packetHashMu.RUnlock()
	if !seen {
		return true
	}
	if p.Type == PacketAnnounce && p.DestinationType == DestSingle {
		return true
	}

	if p.Type == PacketAnnounce {
		Log("Dropped invalid announce packet", LogDebug)
		return false
	}
	Log(fmt.Sprintf("Filtered packet with hash %s", PrettyHexRep(p.PacketHash)), LogExtreme)
	return false
}

// -------- inbound packets --------

func Inbound(raw []byte, ifc *Interface) {
	if len(raw) <= 2 {
		return
	}

	// IFAC authentication
	if ifc != nil && ifc.IFACIdentity != nil {
		if raw[0]&0x80 == 0 {
			return
		}
		if len(raw) <= 2+ifc.IFACSize {
			return
		}
		ifac := raw[2 : 2+ifc.IFACSize]
		mask, err := Cryptography.HKDF(len(raw), ifac, ifc.IFACKey, nil)
		if err != nil {
			return
		}

		unmasked := make([]byte, len(raw))
		for i, b := range raw {
			if i <= 1 || i > ifc.IFACSize+1 {
				unmasked[i] = b ^ mask[i]
			} else {
				unmasked[i] = b
			}
		}
		newHeader := []byte{unmasked[0] & 0x7f, unmasked[1]}
		newRaw := append(newHeader, unmasked[2+ifc.IFACSize:]...)
		expSig, err := ifc.IFACIdentity.Sign(newRaw)
		if err != nil {
			return
		}
		if len(expSig) < ifc.IFACSize {
			return
		}
		expIFAC := expSig[len(expSig)-ifc.IFACSize:]
		if !bytes.Equal(ifac, expIFAC) {
			return
		}
		raw = newRaw
	} else {
		if raw[0]&0x80 == 0x80 {
			// interface without IFAC, but flag set -> drop
			return
		}
	}

	jobsMu.RLock()
	defer func() {
		jobsMu.RUnlock()
		if Owner != nil {
			Owner.ShouldPersistData()
		}
	}()

	if TransportIdentity == nil {
		return
	}

	p := NewPacket(nil, raw, PacketTypeData, PacketCtxNone, Broadcast, HeaderType1, nil, nil, true, FlagUnset)
	if !p.Unpack() {
		return
	}

	p.ReceivingInterface = ifc
	p.Hops++

	// Cache RSSI/SNR/Q for local-client lookups, mirroring Python's
	// Transport.inbound() local_client_*_cache updates.
	if ifc != nil {
		if p.RSSI != nil {
			localStatsMu.Lock()
			if len(p.PacketHash) == 0 {
				p.PacketHash = p.GetHash()
			}
			if len(p.PacketHash) != 0 {
				key := string(p.PacketHash)
				localRSSICache[key] = *p.RSSI
				localRSSIFIFO = append(localRSSIFIFO, key)
				if len(localRSSIFIFO) > localStatsMax {
					evict := localRSSIFIFO[0]
					localRSSIFIFO = localRSSIFIFO[1:]
					delete(localRSSICache, evict)
				}
			}
			localStatsMu.Unlock()
		}
		if p.SNR != nil {
			localStatsMu.Lock()
			if len(p.PacketHash) == 0 {
				p.PacketHash = p.GetHash()
			}
			if len(p.PacketHash) != 0 {
				key := string(p.PacketHash)
				localSNRCache[key] = *p.SNR
				localSNRFIFO = append(localSNRFIFO, key)
				if len(localSNRFIFO) > localStatsMax {
					evict := localSNRFIFO[0]
					localSNRFIFO = localSNRFIFO[1:]
					delete(localSNRCache, evict)
				}
			}
			localStatsMu.Unlock()
		}
		if p.Q != nil {
			localStatsMu.Lock()
			if len(p.PacketHash) == 0 {
				p.PacketHash = p.GetHash()
			}
			if len(p.PacketHash) != 0 {
				key := string(p.PacketHash)
				localQCache[key] = *p.Q
				localQFIFO = append(localQFIFO, key)
				if len(localQFIFO) > localStatsMax {
					evict := localQFIFO[0]
					localQFIFO = localQFIFO[1:]
					delete(localQCache, evict)
				}
			}
			localStatsMu.Unlock()
		}
	}

	// hop correction for local clients / shared instance
	if len(LocalClientInterfaces) > 0 {
		if IsLocalClientInterface(ifc) {
			p.Hops--
		}
	} else if InterfaceToSharedInstance(ifc) {
		p.Hops--
	}

	if !PacketFilter(p) {
		return
	}

	rememberHash := true
	if key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(p.DestinationHash); ok {
		linkTableMu.RLock()
		_, exists := linkTable[key]
		linkTableMu.RUnlock()
		if exists {
			rememberHash = false
		}
	}
	if p.Type == PacketProof && p.Context == PacketLRProof {
		rememberHash = false
	}

	if rememberHash {
		AddPacketHash(p.PacketHash)
		// also cache the packet here, if enabled
	}

	fromLocal := false
	interfacesMu.RLock()
	for _, cif := range LocalClientInterfaces {
		if cif == ifc {
			fromLocal = true
			break
		}
	}
	interfacesMu.RUnlock()
	forLocalClient := func(p *Packet) bool {
		if p.Type == PacketAnnounce {
			return false
		}
		var entry *PathEntry
		if key, ok := func(hash []byte) (hashKey, bool) {
			if len(hash) < truncatedHashBytes {
				return hashKey{}, false
			}
			var key hashKey
			copy(key[:], hash[:truncatedHashBytes])
			return key, true
		}(p.DestinationHash); ok {
			pathTableMu.RLock()
			entry = pathTable[key]
			pathTableMu.RUnlock()
		}
		return entry != nil && entry.Hops == 0
	}(p)
	forLocalClientLink := func(p *Packet) bool {
		if p.Type == PacketAnnounce {
			return false
		}
		key, ok := func(hash []byte) (hashKey, bool) {
			if len(hash) < truncatedHashBytes {
				return hashKey{}, false
			}
			var key hashKey
			copy(key[:], hash[:truncatedHashBytes])
			return key, true
		}(p.DestinationHash)
		if !ok {
			return false
		}
		linkTableMu.RLock()
		entry := linkTable[key]
		linkTableMu.RUnlock()
		if entry == nil {
			return false
		}
		for _, cif := range LocalClientInterfaces {
			if entry.ReceivedInterface == cif || entry.NextHopInterface == cif {
				return true
			}
		}
		return false
	}(p)
	proofForLocalClient := func(p *Packet) bool {
		key, ok := func(hash []byte) (hashKey, bool) {
			if len(hash) < truncatedHashBytes {
				return hashKey{}, false
			}
			var key hashKey
			copy(key[:], hash[:truncatedHashBytes])
			return key, true
		}(p.DestinationHash)
		if !ok {
			return false
		}
		reverseTableMu.Lock()
		entry := reverseTable[key]
		reverseTableMu.Unlock()
		if entry == nil {
			return false
		}
		for _, cif := range LocalClientInterfaces {
			if entry.ReceivedIf == cif {
				return true
			}
		}
		return false
	}(p)
	transportHandling := TransportEnabled() || fromLocal || forLocalClient || forLocalClientLink

	// Python parity (Transport.py:1404): inject our identity as transport ID so the
	// routing block below can forward the packet to the local client.  Track whether
	// we injected it so we can fall through to local delivery afterwards (Python does
	// not return after the routing transmit – execution continues to packet-type
	// delivery).
	injectedTransportID := false
	if p.TransportID == nil && forLocalClient {
		p.TransportID = append([]byte(nil), TransportIdentity.Hash...)
		injectedTransportID = true
	}

	if transportHandling && p.Context == PacketCacheRequest {
		if cacheRequestPacket(p) {
			return
		}
	}

	if p.TransportID != nil && p.Type != PacketAnnounce && TransportIdentity != nil && bytes.Equal(p.TransportID, TransportIdentity.Hash) {
		key, ok := func(hash []byte) (hashKey, bool) {
			if len(hash) < truncatedHashBytes {
				return hashKey{}, false
			}
			var key hashKey
			copy(key[:], hash[:truncatedHashBytes])
			return key, true
		}(p.DestinationHash)
		if !ok {
			return
		}

		var entry *PathEntry
		if key, ok := func(hash []byte) (hashKey, bool) {
			if len(hash) < truncatedHashBytes {
				return hashKey{}, false
			}
			var key hashKey
			copy(key[:], hash[:truncatedHashBytes])
			return key, true
		}(p.DestinationHash); ok {
			pathTableMu.RLock()
			entry = pathTable[key]
			pathTableMu.RUnlock()
		}
		if entry == nil || entry.RecvInterface == nil {
			Log(fmt.Sprintf("Got packet in transport, but no known path to final destination %s. Dropping packet.", PrettyHexRep(p.DestinationHash)), LogExtreme)
			return
		}

		nextHop := entry.NextHop
		remainingHops := entry.Hops
		if remainingHops < 0 {
			remainingHops = 0
		}
		var outRaw []byte
		switch {
		case remainingHops > 1 && len(nextHop) == truncatedHashBytes:
			// Just increase hop count and transmit with updated next hop.
			outRaw = make([]byte, 0, len(p.Raw))
			outRaw = append(outRaw, p.Raw[0])
			outRaw = append(outRaw, p.Raw[1])
			outRaw = append(outRaw, nextHop...)
			outRaw = append(outRaw, p.Raw[truncatedHashBytes+2:]...)

		case remainingHops == 1:
			// Strip transport headers (header2 -> header1) and transmit.
			newFlags := (HeaderType1 << 6) | (Broadcast << 4) | (p.Flags & 0b00001111)
			outRaw = make([]byte, 0, len(p.Raw)-truncatedHashBytes)
			outRaw = append(outRaw, newFlags)
			outRaw = append(outRaw, p.Raw[1])
			outRaw = append(outRaw, p.Raw[truncatedHashBytes+2:]...)

		case remainingHops == 0:
			// Just increase hop count and transmit (header1 packets behind shared instances).
			outRaw = make([]byte, 0, len(p.Raw))
			outRaw = append(outRaw, p.Raw[0])
			outRaw = append(outRaw, p.Raw[1])
			outRaw = append(outRaw, p.Raw[2:]...)

		default:
			return
		}

		// LINKREQUEST: MTU clamping for forwarded packets (Python lines 1447-1469),
		// then create link table entry, else create reverse table entry.
		if p.PacketType == PacketTypeLinkRequest {
			if pathMTU, hasMTU := linkMTUFromLRPacket(p); hasMTU {
				mode := linkModeFromLRPacket(p)
				phMTU := 0
				if ifc != nil {
					phMTU = ifc.HWMTU
				}
				outbound := entry.RecvInterface
				nhMTU := outbound.HWMTU
				if nhMTU == 0 {
					Log("No next-hop HW MTU, disabling link MTU upgrade", LogDebug)
					if len(outRaw) >= linkSignalSize {
						outRaw = outRaw[:len(outRaw)-linkSignalSize]
					}
				} else if !outbound.AutoconfigureMTU && !outbound.FixedMTU {
					Log("Outbound interface doesn't support MTU autoconfiguration, disabling link MTU upgrade", LogDebug)
					if len(outRaw) >= linkSignalSize {
						outRaw = outRaw[:len(outRaw)-linkSignalSize]
					}
				} else if nhMTU < pathMTU || (phMTU > 0 && phMTU < pathMTU) {
					clampMTU := nhMTU
					if phMTU > 0 && phMTU < clampMTU {
						clampMTU = phMTU
					}
					clamped, err := linkSignallingBytes(clampMTU, mode)
					if err != nil {
						Log(fmt.Sprintf("Dropping link request packet. The contained exception was: %v", err), LogWarning)
						return
					}
					Log(fmt.Sprintf("Clamping link MTU to %s", PrettySize(float64(clampMTU))), LogDebug)
					if len(outRaw) >= linkSignalSize {
						outRaw = append(outRaw[:len(outRaw)-linkSignalSize], clamped...)
					}
				}
			}
			linkID := linkIDFromLinkRequestPacket(p)
			if lidKey, ok := func(hash []byte) (hashKey, bool) {
				if len(hash) < truncatedHashBytes {
					return hashKey{}, false
				}
				var key hashKey
				copy(key[:], hash[:truncatedHashBytes])
				return key, true
			}(linkID); ok {
				now := time.Now()
				rem := remainingHops
				if rem < 1 {
					rem = 1
				}
				proofTimeout := now.Add(time.Duration(DEFAULT_PER_HOP_TIMEOUT*rem)*time.Second + ExtraLinkProofTimeout(entry.RecvInterface))
				le := &linkEntry{
					Timestamp:         now,
					NextHopID:         append([]byte(nil), nextHop...),
					NextHopInterface:  entry.RecvInterface,
					RemainingHops:     remainingHops,
					ReceivedInterface: ifc,
					Hops:              int(p.Hops),
					DestinationHash:   append([]byte(nil), p.DestinationHash...),
					Validated:         false,
					ProofTimeout:      proofTimeout,
				}
				linkMu.Lock()
				linkTable[lidKey] = le
				linkMu.Unlock()
			}
		} else {
			reverseTableMu.Lock()
			reverseTable[key] = &reverseEntry{ReceivedIf: ifc, OutboundIf: entry.RecvInterface, Timestamp: time.Now()}
			reverseTableMu.Unlock()
		}

		transmit(entry.RecvInterface, outRaw)
		pathTableMu.Lock()
		if cur := pathTable[key]; cur == entry {
			entry.Timestamp = time.Now()
			pathTable[key] = entry
		}
		pathTableMu.Unlock()
		// Python parity: Python does NOT return after the routing transmit — execution
		// continues to local delivery (announce/data/proof handlers below).  In Go we
		// only need this fall-through when we injected the transport ID ourselves (the
		// for_local_client case); packets that already carried a transport ID from the
		// network are fully handled by routing and should return here.
		if !injectedTransportID {
			return
		}
		p.TransportID = nil // clear injected ID so local delivery code treats it as a direct packet
	}

	// Local client routing: if a destination is behind a local client (path hops==0),
	// forward packets to the local client interface.
	if forLocalClient && !fromLocal {
		var entry *PathEntry
		if key, ok := func(hash []byte) (hashKey, bool) {
			if len(hash) < truncatedHashBytes {
				return hashKey{}, false
			}
			var key hashKey
			copy(key[:], hash[:truncatedHashBytes])
			return key, true
		}(p.DestinationHash); ok {
			pathTableMu.RLock()
			entry = pathTable[key]
			pathTableMu.RUnlock()
		}
		if entry != nil && entry.RecvInterface != nil && IsLocalClientInterface(entry.RecvInterface) {
			// Python parity: when forwarding LINKREQUEST packets to a local client, we
			// must create a link_table entry so that subsequent LINK/LRPROOF packets
			// can be routed back to the original interface.
			if p.PacketType == PacketTypeLinkRequest {
				linkID := linkIDFromLinkRequestPacket(p)
				if lidKey, ok := func(hash []byte) (hashKey, bool) {
					if len(hash) < truncatedHashBytes {
						return hashKey{}, false
					}
					var key hashKey
					copy(key[:], hash[:truncatedHashBytes])
					return key, true
				}(linkID); ok {
					now := time.Now()
					rem := entry.Hops
					if rem < 1 {
						rem = 1
					}
					proofTimeout := now.Add(time.Duration(DEFAULT_PER_HOP_TIMEOUT*rem)*time.Second + ExtraLinkProofTimeout(entry.RecvInterface))
					le := &linkEntry{
						Timestamp:         now,
						NextHopID:         append([]byte(nil), entry.NextHop...),
						NextHopInterface:  entry.RecvInterface,
						RemainingHops:     entry.Hops,
						ReceivedInterface: ifc,
						Hops:              int(p.Hops),
						DestinationHash:   append([]byte(nil), p.DestinationHash...),
						Validated:         false,
						ProofTimeout:      proofTimeout,
					}
					linkTableMu.Lock()
					linkTable[lidKey] = le
					linkTableMu.Unlock()
				}
			}
			transmit(entry.RecvInterface, p.Raw)
			return
		}
	}

	// PLAIN BROADCAST from a client -> forward across interfaces
	var isControl bool
	controlHashesMu.RLock()
	_, isControl = controlHashes[string(p.DestinationHash)]
	controlHashesMu.RUnlock()
	if !isControl &&
		p.DestinationType == DestPlain &&
		p.TransportType == Broadcast {

		if fromLocal {
			for _, oif := range Interfaces {
				if oif != ifc {
					transmit(oif, p.Raw)
				}
			}
		} else {
			for _, cif := range LocalClientInterfaces {
				transmit(cif, p.Raw)
			}
		}
	}

	if p.Type == PacketAnnounce {
		if p == nil {
			return
		}

		if ifc != nil && ValidateAnnounce(p, true) {
			ifc.ReceivedAnnounce()
		}

		// Python parity: apply ingress limiting for unknown destinations.
		if ifc != nil && !HasPath(p.DestinationHash) {
			if ifc.ShouldIngressLimit() {
				ifc.HoldAnnounce(append([]byte(nil), p.Raw...), p.ReceivingInterface, append([]byte(nil), p.DestinationHash...), p.Hops)
				return
			}
		}

		var dst *Destination
		destinationsMu.Lock()
		for _, candidate := range Destinations {
			if candidate != nil && len(candidate.Hash) > 0 && bytes.Equal(candidate.Hash, p.DestinationHash) {
				dst = candidate
				break
			}
		}
		destinationsMu.Unlock()
		if dst != nil {
			return
		}

		if !ValidateAnnounce(p, false) {
			return
		}

		// Python keeps the full announce-ingress critical section under a single
		// inbound_announce_lock. Without that serialization, two near-simultaneous
		// arrivals of the same announce can both observe stale path state and
		// re-accept the packet on TCP rings.
		inboundAnnounceMu.Lock()
		defer inboundAnnounceMu.Unlock()

		now := time.Now()
		if len(p.TransportID) > 0 && TransportEnabled() {
			if key, ok := func(hash []byte) (hashKey, bool) {
				if len(hash) < truncatedHashBytes {
					return hashKey{}, false
				}
				var key hashKey
				copy(key[:], hash[:truncatedHashBytes])
				return key, true
			}(p.DestinationHash); ok {
				announceMu.Lock()
				entry := announceTable[key]
				if entry != nil && entry.Packet != nil {
					expected := entry.Packet.Hops
					if p.Hops-1 == expected {
						entry.LocalRebroadcasts++
						Log(fmt.Sprintf("Heard a rebroadcast of announce for %s on %v", PrettyHexRep(p.DestinationHash), p.ReceivingInterface), LogExtreme)
						if entry.Retries > 0 {
							if entry.LocalRebroadcasts >= localRebroadcastsMax {
								Log(fmt.Sprintf("Completed announce processing for %s, local rebroadcast limit reached", PrettyHexRep(p.DestinationHash)), LogExtreme)
								delete(announceTable, key)
								announceMu.Unlock()
								return
							}
						}
						announceTable[key] = entry
					} else if p.Hops-1 == expected+1 && entry.Retries > 0 && now.Before(entry.Next) {
						Log(fmt.Sprintf("Rebroadcasted announce for %s has been passed on to another node, no further tries needed", PrettyHexRep(p.DestinationHash)), LogExtreme)
						delete(announceTable, key)
					}
				}
				announceMu.Unlock()
			}
		}

		if p.Hops >= PathfinderMaxHops+1 {
			return
		}

		updated := false
		if pathKey, ok := func(hash []byte) (hashKey, bool) {
			if len(hash) < truncatedHashBytes {
				return hashKey{}, false
			}
			var key hashKey
			copy(key[:], hash[:truncatedHashBytes])
			return key, true
		}(p.DestinationHash); ok {
			offset := identityPubKeyLen + IdentityNameHashLength/8
			var randomBlob []byte
			if p != nil && len(p.Data) > offset {
				end := offset + announceRandomHashLen
				if end > len(p.Data) {
					end = len(p.Data)
				}
				randomBlob = append([]byte(nil), p.Data[offset:end]...)
			}
			var existing *PathEntry
			if key, ok := func(hash []byte) (hashKey, bool) {
				if len(hash) < truncatedHashBytes {
					return hashKey{}, false
				}
				var key hashKey
				copy(key[:], hash[:truncatedHashBytes])
				return key, true
			}(p.DestinationHash); ok {
				pathTableMu.RLock()
				existing = pathTable[key]
				pathTableMu.RUnlock()
			}

			var blobs [][]byte
			if existing != nil && len(existing.RandomBlobs) > 0 {
				blobs = func(in [][]byte) [][]byte {
					if len(in) == 0 {
						return nil
					}
					out := make([][]byte, len(in))
					for i, blob := range in {
						out[i] = append([]byte(nil), blob...)
					}
					return out
				}(existing.RandomBlobs)
			}

			blobSeen := false
			if len(randomBlob) > 0 {
				for _, existingBlob := range blobs {
					if bytes.Equal(existingBlob, randomBlob) {
						blobSeen = true
						break
					}
				}
			}
			emitted := timebaseFromRandomBlob(randomBlob)
			shouldAdd := false

			switch {
			case existing == nil:
				shouldAdd = true
			case int(p.Hops) <= existing.Hops:
				if !blobSeen && emitted > timebaseFromRandomBlobs(blobs) {
					MarkPathUnknownState(p.DestinationHash)
					shouldAdd = true
				}
			default:
				expired := now.After(existing.ExpiresAt)
				pathEmitted := timebaseFromRandomBlobs(blobs)
				newer := emitted > pathEmitted
				if expired && !blobSeen {
					MarkPathUnknownState(p.DestinationHash)
					Log(fmt.Sprintf("Replacing destination table entry for %s with new announce due to expired path", PrettyHexRep(p.DestinationHash)), LogDebug)
					shouldAdd = true
				} else if newer && !blobSeen {
					MarkPathUnknownState(p.DestinationHash)
					Log(fmt.Sprintf("Replacing destination table entry for %s with new announce, since it was more recently emitted", PrettyHexRep(p.DestinationHash)), LogDebug)
					shouldAdd = true
				} else if emitted == pathEmitted && PathIsUnresponsive(p.DestinationHash) {
					Log(fmt.Sprintf("Replacing destination table entry for %s with new announce, since previously tried path was unresponsive", PrettyHexRep(p.DestinationHash)), LogDebug)
					shouldAdd = true
				}
			}

			if shouldAdd {
				if len(randomBlob) > 0 {
					blobs = append(blobs, randomBlob)
					if len(blobs) > maxRandomBlobs {
						blobs = blobs[len(blobs)-maxRandomBlobs:]
					}
				}
				nextHop := p.TransportID
				if len(nextHop) == 0 {
					nextHop = p.DestinationHash
				}
				packetHash := p.PacketHash
				if len(packetHash) == 0 {
					packetHash = p.GetHash()
				}

				expiresAt := now.Add(pathExpiration)
				if ifc != nil {
					switch ifc.Mode {
					case InterfaceModeAccessPoint:
						expiresAt = now.Add(apPathTime)
					case InterfaceModeRoaming:
						expiresAt = now.Add(roamingPathTime)
					}
				}

				entry := &PathEntry{
					NextHop:       append([]byte(nil), nextHop...),
					RecvInterface: ifc,
					Hops:          int(p.Hops),
					Timestamp:     now,
					ExpiresAt:     expiresAt,
					RandomBlobs:   blobs,
					AnnounceAt:    emitted,
					PacketHash:    append([]byte(nil), packetHash...),
				}

				pathTableMu.Lock()
				pathTable[pathKey] = entry
				pathTableMu.Unlock()

				if ifc != nil && len(ifc.TunnelID) == HashLengthBytes {
					tunnelsMu.Lock()
					if te := tunnels[string(ifc.TunnelID)]; te != nil {
						if te.Paths == nil {
							te.Paths = make(map[string]*tunnelPathEntry)
						}
						te.Paths[string(p.DestinationHash)] = &tunnelPathEntry{
							Timestamp:    entry.Timestamp,
							ReceivedFrom: append([]byte(nil), entry.NextHop...),
							Hops:         entry.Hops,
							ExpiresAt:    entry.ExpiresAt,
							RandomBlobs: func(in [][]byte) [][]byte {
								if len(in) == 0 {
									return nil
								}
								out := make([][]byte, len(in))
								for i, blob := range in {
									out[i] = append([]byte(nil), blob...)
								}
								return out
							}(entry.RandomBlobs),
							PacketHash: append([]byte(nil), entry.PacketHash...),
						}
						te.ExpiresAt = time.Now().Add(DestinationTimeout)
						tunnels[string(ifc.TunnelID)] = te
					}
					tunnelsMu.Unlock()
					Log(fmt.Sprintf("Path to %s associated with tunnel %s", PrettyHexRep(p.DestinationHash), PrettyHexRep(ifc.TunnelID)), LogDebug)
				}

				// Python parity: cache announce with force_cache=True when adding/updating path table.
				Cache(p, true)
				updated = true
			}
		}
		if updated {
			receivedFrom := p.TransportID
			if len(receivedFrom) == 0 {
				receivedFrom = p.DestinationHash
			}
			Log(fmt.Sprintf("Destination %s is now %d hops away via %s on %s",
				PrettyHexRep(p.DestinationHash), p.Hops, PrettyHexRep(receivedFrom), fmt.Sprint(ifc)), LogDebug,
			)
		}
		if !updated {
			return
		}

		if len(LocalClientInterfaces) > 0 {
			announceIdentity := IdentityRecall(p.DestinationHash)
			announceDestination := &Destination{
				Type:      DestinationSINGLE,
				Direction: DestinationOUT,
				Identity:  announceIdentity,
				Hash:      append([]byte(nil), p.DestinationHash...),
				HexHash:   PrettyHexRep(p.DestinationHash),
			}
			for _, cif := range LocalClientInterfaces {
				if cif == nil || p.ReceivingInterface == cif {
					continue
				}
				send := NewPacket(
					announceDestination,
					append([]byte(nil), p.Data...),
					PacketANNOUNCE,
					PacketNONE,
					TransportDirect,
					HeaderType2,
					append([]byte(nil), p.TransportID...),
					cif,
					true,
					p.ContextFlag,
				)
				if send == nil {
					continue
				}
				if TransportIdentity != nil && len(TransportIdentity.Hash) > 0 {
					send.TransportID = append([]byte(nil), TransportIdentity.Hash...)
				}
				send.Hops = p.Hops
				send.DestinationHash = append([]byte(nil), p.DestinationHash...)
				send.DestinationType = byte(DestinationSINGLE)
				if err := send.Pack(); err != nil {
					continue
				}
				raw := append([]byte(nil), send.Raw...)
				transmit(cif, raw)
			}
		}

		if TransportIdentity != nil {
			if key, ok := func(hash []byte) (hashKey, bool) {
				if len(hash) < truncatedHashBytes {
					return hashKey{}, false
				}
				var key hashKey
				copy(key[:], hash[:truncatedHashBytes])
				return key, true
			}(p.DestinationHash); ok {
				discoveryPathRequestsMu.Lock()
				entry := discoveryPathRequests[key]
				if entry != nil {
					if !entry.Timeout.IsZero() && !now.Before(entry.Timeout) {
						delete(discoveryPathRequests, key)
						entry = nil
					} else {
						delete(discoveryPathRequests, key)
					}
				}
				discoveryPathRequestsMu.Unlock()

				if entry != nil && entry.RequestingInterface != nil {
					Log(fmt.Sprintf("Got matching announce, answering waiting discovery path request for %s on %s",
						PrettyHexRep(p.DestinationHash), fmt.Sprint(entry.RequestingInterface)), LogDebug,
					)

					dest := &Destination{
						Type:      DestinationSINGLE,
						Direction: DestinationOUT,
						Hash:      append([]byte(nil), p.DestinationHash...),
						HexHash:   PrettyHexRep(p.DestinationHash),
					}

					response := NewPacket(
						dest,
						append([]byte(nil), p.Data...),
						PacketANNOUNCE,
						PacketPATH_RESPONSE,
						TransportDirect,
						HeaderType2,
						append([]byte(nil), TransportIdentity.Hash...),
						entry.RequestingInterface,
						false,
						p.ContextFlag,
					)
					if response != nil {
						response.Hops = p.Hops
						response.DestinationHash = append([]byte(nil), p.DestinationHash...)
						response.DestinationType = byte(DestinationSINGLE)
						_ = response.Send()
					}
				}
			}
		}

		// Python parity: only newly accepted announces propagate to handlers and
		// transport rebroadcast.
		if p != nil && p.PacketType == PacketTypeAnnounce && len(p.Data) >= identityPubKeyLen+IdentityNameHashLength/8+announceRandomHashLen+ed25519.SignatureSize {
			announceHandlersMu.RLock()
			handlers := append([]any(nil), announceHandlers...)
			announceHandlersMu.RUnlock()

			for _, handler := range handlers {
				if handler == nil {
					continue
				}

				announceHandler, ok := handler.(AnnounceHandler)
				if !ok {
					continue
				}

				announced := IdentityRecall(p.DestinationHash)
				filterValue := announceHandler.AspectFilter()
				if filterValue != nil {
					filter, ok := filterValue.(string)
					if !ok {
						continue
					}
					expectedHash, err := (&Destination{}).HashFromNameAndIdentity(filter, announced)
					if err != nil || !bytes.Equal(expectedHash, p.DestinationHash) {
						continue
					}
				}

				if p.Context == PacketPATH_RESPONSE {
					prHandler, ok := handler.(PathResponseAnnounceHandler)
					if !ok || !prHandler.ReceivePathResponses() {
						continue
					}
				}

				go func(handler any) {
					defer func() {
						if rec := recover(); rec != nil {
							Log("Error while processing external announce callback.", LogError)
							Log(fmt.Sprintf("The contained exception was: %v", rec), LogError)
							TraceException(rec)
						}
					}()

					appData := IdentityRecallAppData(p.DestinationHash)
					packetHash := append([]byte(nil), p.PacketHash...)
					if len(packetHash) == 0 {
						packetHash = append([]byte(nil), p.GetHash()...)
					}

					switch typed := handler.(type) {
					case AnnounceHandlerWithPacketInfo:
						typed.ReceivedAnnounceWithPacketInfo(
							append([]byte(nil), p.DestinationHash...),
							announced,
							appData,
							packetHash,
							p.Context == PacketPATH_RESPONSE,
						)
					case AnnounceHandlerWithPacketHash:
						typed.ReceivedAnnounceWithPacketHash(
							append([]byte(nil), p.DestinationHash...),
							announced,
							appData,
							packetHash,
						)
					default:
						announceHandler.ReceivedAnnounce(append([]byte(nil), p.DestinationHash...), announced, appData)
					}
				}(handler)
			}
		}

		retransmitAllowed := TransportEnabled() || fromLocal
		if !retransmitAllowed {
			return
		}

		if p.Context == PacketPATH_RESPONSE && !fromLocal {
			return
		}

		announceKey, ok := func(hash []byte) (hashKey, bool) {
			if len(hash) < truncatedHashBytes {
				return hashKey{}, false
			}
			var key hashKey
			copy(key[:], hash[:truncatedHashBytes])
			return key, true
		}(p.DestinationHash)
		if !ok {
			return
		}

		rateBlocked := false
		if p.Context != PacketPATH_RESPONSE {
			type announceRateTargetGetter interface {
				AnnounceRateTarget() time.Duration
			}
			if getter, ok := any(ifc).(announceRateTargetGetter); ok {
				target := getter.AnnounceRateTarget()
				if target > 0 {
					type announceRateGraceGetter interface {
						AnnounceRateGrace() int
					}
					type announceRatePenaltyGetter interface {
						AnnounceRatePenalty() time.Duration
					}

					grace := 0
					penalty := time.Duration(0)
					if getter, ok := any(ifc).(announceRateGraceGetter); ok {
						grace = getter.AnnounceRateGrace()
					}
					if getter, ok := any(ifc).(announceRatePenaltyGetter); ok {
						penalty = getter.AnnounceRatePenalty()
					}

					announceRateMu.Lock()
					entry := announceRateTable[announceKey]
					if entry == nil {
						announceRateTable[announceKey] = &announceRateEntry{
							Last:       now,
							Timestamps: []time.Time{now},
						}
						announceRateOrder = append(announceRateOrder, announceKey)
					} else {
						entry.Timestamps = append(entry.Timestamps, now)
						if len(entry.Timestamps) > maxRateTimestamps {
							entry.Timestamps = entry.Timestamps[len(entry.Timestamps)-maxRateTimestamps:]
						}

						currentRate := now.Sub(entry.Last)
						if now.After(entry.BlockedUntil) {
							if currentRate < target {
								entry.RateViolations++
							} else if entry.RateViolations > 0 {
								entry.RateViolations--
							}

							if entry.RateViolations > grace {
								entry.BlockedUntil = entry.Last.Add(target + penalty)
								rateBlocked = true
							} else {
								entry.Last = now
							}
						} else {
							rateBlocked = true
						}
					}
					announceRateMu.Unlock()
				}
			}
		}
		if rateBlocked {
			Log(fmt.Sprintf("Blocking rebroadcast of announce from %s due to excessive announce rate",
				PrettyHexRep(p.DestinationHash)), LogDebug,
			)
			return
		}

		if p.Context == PacketPATH_RESPONSE {
			// Python only queues local-client PATH_RESPONSE announces when an
			// external path request is pending. These are rebroadcast as normal
			// announces, not as PATH_RESPONSE packets, and are not pinned to the
			// requesting interface.
			if !fromLocal {
				return
			}

			queuePathResponse := false
			if key, ok := func(hash []byte) (hashKey, bool) {
				if len(hash) < truncatedHashBytes {
					return hashKey{}, false
				}
				var key hashKey
				copy(key[:], hash[:truncatedHashBytes])
				return key, true
			}(p.DestinationHash); ok {
				pendingLocalPathRequestsMu.Lock()
				if _, exists := pendingLocalPathRequests[key]; exists {
					delete(pendingLocalPathRequests, key)
					queuePathResponse = true
				}
				pendingLocalPathRequestsMu.Unlock()
			}
			if !queuePathResponse {
				return
			}

			if keyHash, ok := func(hash []byte) (hashKey, bool) {
				if len(hash) < truncatedHashBytes {
					return hashKey{}, false
				}
				var key hashKey
				copy(key[:], hash[:truncatedHashBytes])
				return key, true
			}(p.DestinationHash); ok {
				now := time.Now()
				entry := &announceEntry{
					Packet:            p,
					Next:              now,
					Retries:           pathfinderRetryLimit,
					Timestamp:         now,
					Expires:           now.Add(announceQueueTTL),
					LocalRebroadcasts: 0,
					BlockRebroadcasts: false,
					AttachedInterface: nil,
				}
				announceMu.Lock()
				announceTable[keyHash] = entry
				announceMu.Unlock()
			}
			return
		}

		if keyHash, ok := func(hash []byte) (hashKey, bool) {
			if len(hash) < truncatedHashBytes {
				return hashKey{}, false
			}
			var key hashKey
			copy(key[:], hash[:truncatedHashBytes])
			return key, true
		}(p.DestinationHash); ok {
			now := time.Now()
			delay := pathfinderRetryGrace
			if pathfinderRandomWindow > 0 {
				delay = time.Duration(Rand() * float64(pathfinderRandomWindow))
			}
			entry := &announceEntry{
				Packet:            p,
				Next:              now.Add(delay),
				Retries:           0,
				Timestamp:         now,
				Expires:           now.Add(announceQueueTTL),
				LocalRebroadcasts: 0,
				BlockRebroadcasts: false,
				AttachedInterface: nil,
			}
			if fromLocal {
				entry.Retries = pathfinderRetryLimit
				entry.Next = now
			}
			announceMu.Lock()
			announceTable[keyHash] = entry
			announceMu.Unlock()
		}
	}

	// Link transport handling (Python: routes packets according to link_table).
	if p.Type != PacketAnnounce && p.Type != PacketLINKREQUEST && p.Context != PacketCtxLRProof {
		// If this link packet is for a local link, deliver it before any routing
		// decisions. Shared instances will typically have no local link, and will
		// fall back to link_table forwarding below.
		if p.DestinationType == DestLink {
			var link *Link
			linkMu.Lock()
			for _, candidate := range ActiveLinks {
				if candidate != nil && bytes.Equal(candidate.LinkID, p.DestinationHash) {
					link = candidate
					break
				}
			}
			if link == nil {
				for _, candidate := range PendingLinks {
					if candidate != nil && bytes.Equal(candidate.LinkID, p.DestinationHash) {
						link = candidate
						break
					}
				}
			}
			linkMu.Unlock()
			if link != nil {
				// Python parity: for DATA packets addressed to a link, only deliver
				// if the packet arrived on the link's attached interface. If not, remove
				// the packet hash from the filter so the link can receive it when it
				// finally arrives over the correct path (Python lines 1977-1986).
				if p.Type == PacketData && link.AttachedInterface != nil && link.AttachedInterface != p.ReceivingInterface {
					packetHashMu.Lock()
					if len(p.PacketHash) >= truncatedHashBytes {
						var k hashKey
						copy(k[:], p.PacketHash[:truncatedHashBytes])
						delete(PacketHashSet, k)
						delete(PacketHashSet2, k)
					}
					packetHashMu.Unlock()
				} else {
					// Python parity (lines 1968-1973): DATA+LINK CACHE_REQUEST packets
					// must look up the cached packet and deliver it to the link, not
					// pass the request packet itself to link.Receive.
					if p.Type == PacketData && p.Context == PacketCacheRequest {
						cacheRequestPacket(p)
						return
					}
					link.Receive(p)
					return
				}
			}
		}

		if p.Type != PacketAnnounce && p.PacketType != PacketTypeLinkRequest && p.Context != PacketLRProof {
			key, ok := func(hash []byte) (hashKey, bool) {
				if len(hash) < truncatedHashBytes {
					return hashKey{}, false
				}
				var key hashKey
				copy(key[:], hash[:truncatedHashBytes])
				return key, true
			}(p.DestinationHash)
			if ok {
				linkTableMu.RLock()
				entry := linkTable[key]
				linkTableMu.RUnlock()
				if entry != nil && entry.NextHopInterface != nil && entry.ReceivedInterface != nil {
					var outbound *Interface
					if entry.NextHopInterface == entry.ReceivedInterface {
						// Direction doesn't matter, but ensure hop count matches expectations.
						if int(p.Hops) == entry.RemainingHops || int(p.Hops) == entry.Hops {
							outbound = entry.NextHopInterface
						}
					} else {
						// Direction matters; transmit on opposite interface to where it was received.
						if ifc == entry.NextHopInterface {
							if int(p.Hops) == entry.RemainingHops {
								outbound = entry.ReceivedInterface
							}
						} else if ifc == entry.ReceivedInterface {
							if int(p.Hops) == entry.Hops {
								outbound = entry.NextHopInterface
							}
						}
					}

					if outbound != nil && outbound != ifc {
						// Add this packet to the filter hashlist if we have determined that it's our turn.
						AddPacketHash(p.PacketHash)

						outRaw := append([]byte{p.Raw[0], p.Raw[1]}, p.Raw[2:]...)
						transmit(outbound, outRaw)

						linkTableMu.Lock()
						if cur := linkTable[key]; cur == entry {
							if p.Type == PacketProof && p.Context == PacketCtxLRProof {
								entry.Validated = true
								entry.ProofTimeout = time.Time{}
							}
							entry.Timestamp = time.Now()
							linkTable[key] = entry
						}
						linkTableMu.Unlock()
						return
					}

				}
			}
		}
	}

	// ---- basic delivery pipeline (Destination/Link/Proof) ----

	// Proof packets are handled before generic destination routing.
	if p.Type == PacketProof {
		if p.Context == PacketLRProof {
			handledTransport := false
			if (TransportEnabled() || forLocalClientLink || fromLocal) && len(p.DestinationHash) > 0 {
				key, ok := func(hash []byte) (hashKey, bool) {
					if len(hash) < truncatedHashBytes {
						return hashKey{}, false
					}
					var key hashKey
					copy(key[:], hash[:truncatedHashBytes])
					return key, true
				}(p.DestinationHash)
				if ok {
					linkTableMu.RLock()
					entry := linkTable[key]
					linkTableMu.RUnlock()
					if entry != nil {
						handledTransport = true
						if int(p.Hops) == entry.RemainingHops {
							if p.ReceivingInterface == entry.NextHopInterface {
								tryValidate := false
								if len(p.Data) == ed25519.SignatureSize+linkEcPubSize/2 || len(p.Data) == ed25519.SignatureSize+linkEcPubSize/2+linkSignalSize {
									tryValidate = true
								}
								if tryValidate {
									// Python parity: try/except wraps the entire validation block
									// (Transport.py:2010-2035). Without trace_exception → simple recover.
									func() {
										defer func() {
											if rec := recover(); rec != nil {
												Log(fmt.Sprintf("Error while transporting link request proof. The contained exception was: %v", rec), LogError)
											}
										}()
										signalling := []byte{}
										if len(p.Data) == ed25519.SignatureSize+linkEcPubSize/2+linkSignalSize {
											mtu, ok := linkMTUFromProofPacket(p)
											if ok {
												mode := linkModeFromProofPacket(p)
												if sb, err := linkSignallingBytes(mtu, mode); err == nil {
													signalling = sb
												} else {
													Log(fmt.Sprintf("Error while transporting link request proof. The contained exception was: %v", err), LogError)
												}
											}
										}

										peerPubBytes := p.Data[ed25519.SignatureSize : ed25519.SignatureSize+linkEcPubSize/2]
										peerIdentity := IdentityRecall(entry.DestinationHash)
										if peerIdentity != nil {
											peerPubKey := peerIdentity.GetPublicKey()
											if len(peerPubKey) >= linkEcPubSize {
												peerSigPubBytes := peerPubKey[linkEcPubSize/2 : linkEcPubSize]
												signedData := make([]byte, 0, len(p.DestinationHash)+len(peerPubBytes)+len(peerSigPubBytes)+len(signalling))
												signedData = append(signedData, p.DestinationHash...)
												signedData = append(signedData, peerPubBytes...)
												signedData = append(signedData, peerSigPubBytes...)
												signedData = append(signedData, signalling...)
												signature := p.Data[:ed25519.SignatureSize]

												if peerIdentity.Validate(signature, signedData) {
													Log(fmt.Sprintf("Link request proof validated for transport via %v", entry.ReceivedInterface), LogExtreme)
													newRaw := []byte{p.Raw[0]}
													newRaw = append(newRaw, byte(p.Hops))
													newRaw = append(newRaw, p.Raw[2:]...)
													linkTableMu.Lock()
													if cur := linkTable[key]; cur == entry {
														entry.Validated = true
														entry.Timestamp = time.Now()
														linkTable[key] = entry
													}
													linkTableMu.Unlock()
													transmit(entry.ReceivedInterface, newRaw)
												} else {
													Log(fmt.Sprintf("Invalid link request proof in transport for link %s, dropping proof.", PrettyHexRep(p.DestinationHash)), LogDebug)
												}
											}
										}
									}()
								}
							} else {
								Log(fmt.Sprintf("Link request proof received on wrong interface, not transporting it."), LogDebug)
							}
						} else {
							Log(fmt.Sprintf("Received link request proof with hop mismatch, not transporting it"), LogDebug)
						}
					}
				}
			}
			if !handledTransport {
				linkMu.Lock()
				pendingLinks := append([]*Link(nil), PendingLinks...)
				linkMu.Unlock()
				for _, link := range pendingLinks {
					if link == nil || !bytes.Equal(link.LinkID, p.DestinationHash) {
						continue
					}
					if p.Hops == uint8(link.ExpectedHops) || link.ExpectedHops == PathfinderMaxHops {
						AddPacketHash(p.PacketHash)
						p.Link = link
						link.validateProof(p)
						return
					}
				}
			}
			return
		}
		if (TransportEnabled() || fromLocal || proofForLocalClient) && len(p.DestinationHash) > 0 {
			if key, ok := func(hash []byte) (hashKey, bool) {
				if len(hash) < truncatedHashBytes {
					return hashKey{}, false
				}
				var key hashKey
				copy(key[:], hash[:truncatedHashBytes])
				return key, true
			}(p.DestinationHash); ok {
				reverseTableMu.Lock()
				entry := reverseTable[key]
				delete(reverseTable, key)
				reverseTableMu.Unlock()
				if entry != nil && entry.ReceivedIf != nil {
					if ifc == entry.OutboundIf {
						Log(fmt.Sprintf("Proof received on correct interface, transporting it via %v", entry.ReceivedIf), LogExtreme)
						newRaw := append([]byte{p.Raw[0], byte(p.Hops)}, p.Raw[2:]...)
						transmit(entry.ReceivedIf, newRaw)
					} else {
						Log(fmt.Sprintf("Proof received on wrong interface, not transporting it."), LogDebug)
					}
				}
			}
		}
		proofHash := []byte(nil)
		if len(p.Data) == ReceiptExplLength {
			proofHash = p.Data[:HashLengthBytes]
		}
		receiptsMu.Lock()
		for i := 0; i < len(Receipts); {
			rc := Receipts[i]
			receiptValidated := false
			if rc != nil && rc.Status == PacketReceiptSENT {
				if proofHash != nil {
					if bytes.Equal(rc.Hash, proofHash) {
						receiptValidated = rc.ValidateProofPacket(p)
					}
				} else {
					receiptValidated = rc.ValidateProofPacket(p)
				}
			}
			if receiptValidated {
				if i < len(Receipts) && Receipts[i] == rc {
					Receipts = append(Receipts[:i], Receipts[i+1:]...)
				} else {
					i++
				}
			} else {
				i++
			}
		}
		receiptsMu.Unlock()
	}

	// Deliver to link if addressed to a link ID.
	if p.DestinationType == DestLink {
		var link *Link
		linkMu.Lock()
		for _, candidate := range ActiveLinks {
			if candidate != nil && bytes.Equal(candidate.LinkID, p.DestinationHash) {
				link = candidate
				break
			}
		}
		if link == nil {
			for _, candidate := range PendingLinks {
				if candidate != nil && bytes.Equal(candidate.LinkID, p.DestinationHash) {
					link = candidate
					break
				}
			}
		}
		linkMu.Unlock()
		if link != nil {
			p.Link = link
			link.Receive(p)
			return
		}
		// Python parity: RESOURCE_PRF packets are delivered to active links before
		// generic destination routing. In shared-instance mode, the active link can
		// live in a local client rather than in this transport process, so forward
		// unmatched resource proofs to local clients.
		if p.Type == PacketProof && p.Context == PacketCtxResourcePrf && !fromLocal {
			for _, cif := range LocalClientInterfaces {
				if cif != nil && cif != ifc {
					transmit(cif, p.Raw)
				}
			}
			return
		}
		// Not a local link; allow routing logic below to handle it.
	}

	// Deliver to a registered destination (including control destinations).
	// Python parity: match both hash AND destination type (Transport.py:1934, 1989).
	var dst *Destination
	destinationsMu.Lock()
	for _, candidate := range Destinations {
		if candidate != nil && len(candidate.Hash) > 0 &&
			bytes.Equal(candidate.Hash, p.DestinationHash) &&
			candidate.Type == int(p.DestinationType) {
			dst = candidate
			break
		}
	}
	destinationsMu.Unlock()
	if dst != nil {
		p.Destination = dst
		// Python parity: LINKREQUEST packets are only delivered to local destinations
		// if transport_id is absent or matches our own identity (Transport.py:1932).
		if p.PacketType == PacketLINKREQUEST && len(p.TransportID) > 0 &&
			(TransportIdentity == nil || !bytes.Equal(p.TransportID, TransportIdentity.Hash)) {
			return
		}
		// Python parity: for LINKREQUEST packets, clamp the path MTU to the
		// receiving interface's hardware MTU before delivering (Python lines 1931-1959).
		if p.PacketType == PacketLINKREQUEST && p.ReceivingInterface != nil {
			pathMTU, hasMTU := linkMTUFromLRPacket(p)
			if hasMTU {
				rifc := p.ReceivingInterface
				mode := linkModeFromLRPacket(p)
				var nhMTU int
				if rifc.AutoconfigureMTU || rifc.FixedMTU {
					nhMTU = rifc.HWMTU
				} else {
					nhMTU = MTU
				}
				if rifc.HWMTU == 0 {
					Log("No next-hop HW MTU, disabling link MTU upgrade", LogDebug)
					if len(p.Data) >= linkSignalSize {
						p.Data = p.Data[:len(p.Data)-linkSignalSize]
					}
				} else if nhMTU < pathMTU {
					clamped, err := linkSignallingBytes(nhMTU, mode)
					if err != nil {
						Log(fmt.Sprintf("Dropping link request packet to local destination. The contained exception was: %v", err), LogWarning)
						return
					}
					Log(fmt.Sprintf("Clamping link MTU to %s", PrettySize(float64(nhMTU))), LogDebug)
					if len(p.Data) >= linkSignalSize {
						p.Data = append(p.Data[:len(p.Data)-linkSignalSize], clamped...)
					}
				}
			}
		}
		// Python parity (Transport.py:1991-1999): after destination.receive(), prove
		// the packet if the destination's proof strategy requires it.
		// Python always calls packet.prove() which routes through
		// packet.generate_proof_destination() — a ProofDestination with no identity
		// and therefore no encryption.  We must pass nil here so that Identity.Prove
		// uses GenerateProofDestination() instead of the server destination (which
		// would encrypt the proof data with the server's public key, breaking receipt
		// validation on the remote side).
		if dst.Receive(p) {
			switch dst.GetProofStrategy() {
			case DestinationPROVE_ALL:
				p.Prove(nil)
			case DestinationPROVE_APP:
				if dst.Callbacks.ProofRequested != nil {
					func() {
						defer func() {
							if r := recover(); r != nil {
								Log(fmt.Sprintf("Error while executing proof request callback. The contained exception was: %v", r), LogError)
							}
						}()
						if dst.Callbacks.ProofRequested(p) {
							p.Prove(nil)
						}
					}()
				}
			}
		}
		return
	}

	// Minimal routing pipeline: forward packets not for local system along a known path.
	if p.Type != PacketAnnounce && p.TransportType != Broadcast && p.DestinationType != DestSingle && p.DestinationType != DestGroup && p.DestinationType != DestPlain {
		// Only addressable destination types are eligible for path-based forwarding.
		return
	}
	if p.Type != PacketAnnounce && p.Type != PacketProof && p.DestinationType == DestPlain && p.TransportType == Broadcast {
		return
	}
	if p.Type != PacketAnnounce && p.Hops < PathfinderMaxHops+1 {
		var entry *PathEntry
		if key, ok := func(hash []byte) (hashKey, bool) {
			if len(hash) < truncatedHashBytes {
				return hashKey{}, false
			}
			var key hashKey
			copy(key[:], hash[:truncatedHashBytes])
			return key, true
		}(p.DestinationHash); ok {
			pathTableMu.RLock()
			entry = pathTable[key]
			pathTableMu.RUnlock()
		}
		if entry != nil && entry.RecvInterface != nil && entry.RecvInterface != ifc {
			// Remaining hops from path table (Python: IDX_PT_HOPS).
			remainingHops := entry.Hops
			if remainingHops < 0 {
				remainingHops = 0
			}
			if remainingHops > 255 {
				remainingHops = 255
			}

			flags := p.Flags
			flags &^= 0b01000000                     // clear header type bit
			flags &^= 0b00010000                     // clear transport type bit
			flags |= (TransportDirect & 0x01) << 4   // set transport type (1 bit encoding)
			flags |= (p.ContextFlag & 0x01) << 5     // preserve context flag
			flags &^= 0b00000011                     // clear packet type bits
			flags |= (p.PacketType & 0x03)           // restore packet type bits
			flags &^= 0b00001100                     // clear destination type bits
			flags |= (p.DestinationType & 0x03) << 2 // restore destination type bits

			outRaw := make([]byte, 0, len(p.Raw)+truncatedHashBytes)
			if remainingHops > 1 && len(entry.NextHop) == truncatedHashBytes {
				// Header 2: flags,hops,next_hop_id,destination_hash,context,data
				flags |= 0b01000000
				outRaw = append(outRaw, flags, p.Hops)
				outRaw = append(outRaw, entry.NextHop...)
				outRaw = append(outRaw, p.DestinationHash...)
				outRaw = append(outRaw, p.Context)
				outRaw = append(outRaw, p.Data...)
			} else {
				// Header 1: flags,hops,destination_hash,context,data
				// Python uses BROADCAST transport type for header1 forwarding.
				flags &^= 0b00010000
				outRaw = append(outRaw, flags, p.Hops)
				outRaw = append(outRaw, p.DestinationHash...)
				outRaw = append(outRaw, p.Context)
				outRaw = append(outRaw, p.Data...)
			}

			// Update reverse table entry to assist replies/proofs returning.
			if key, ok := func(hash []byte) (hashKey, bool) {
				if len(hash) < truncatedHashBytes {
					return hashKey{}, false
				}
				var key hashKey
				copy(key[:], hash[:truncatedHashBytes])
				return key, true
			}(p.GetTruncatedHash()); ok && ifc != nil {
				reverseTableMu.Lock()
				if rev := reverseTable[key]; rev != nil {
					rev.OutboundIf = entry.RecvInterface
					reverseTable[key] = rev
				} else {
					reverseTable[key] = &reverseEntry{ReceivedIf: ifc, OutboundIf: entry.RecvInterface, Timestamp: time.Now()}
				}
				reverseTableMu.Unlock()
			}

			transmit(entry.RecvInterface, outRaw)

			// Create link table entry for link requests so subsequent link packets can be forwarded.
			if p.PacketType == PacketTypeLinkRequest {
				linkID := linkIDFromLinkRequestPacket(p)
				if lidKey, ok := func(hash []byte) (hashKey, bool) {
					if len(hash) < truncatedHashBytes {
						return hashKey{}, false
					}
					var key hashKey
					copy(key[:], hash[:truncatedHashBytes])
					return key, true
				}(linkID); ok {
					rem := remainingHops
					if rem < 1 {
						rem = 1
					}
					proofTmo := time.Now().Add(time.Duration(DEFAULT_PER_HOP_TIMEOUT)*time.Second*time.Duration(rem) + ExtraLinkProofTimeout(entry.RecvInterface))
					le := &linkEntry{
						Timestamp:         time.Now(),
						NextHopID:         append([]byte(nil), entry.NextHop...),
						NextHopInterface:  entry.RecvInterface,
						RemainingHops:     remainingHops,
						ReceivedInterface: ifc,
						Hops:              int(p.Hops),
						DestinationHash:   append([]byte(nil), p.DestinationHash...),
						Validated:         false,
						ProofTimeout:      proofTmo,
					}
					linkTableMu.Lock()
					linkTable[lidKey] = le
					linkTableMu.Unlock()
				}
			}
			return
		}
	}
}

func timebaseFromRandomBlob(blob []byte) uint64 {
	if len(blob) < announceRandomHashLen {
		return 0
	}
	var out uint64
	for _, b := range blob[5:10] {
		out = (out << 8) | uint64(b)
	}
	return out
}

func timebaseFromRandomBlobs(blobs [][]byte) uint64 {
	var max uint64
	for _, blob := range blobs {
		if val := timebaseFromRandomBlob(blob); val > max {
			max = val
		}
	}
	return max
}

// AnnounceEmitted mirrors Python Transport.announce_emitted().
// Returns the emission timebase encoded in the announce packet's random blob.
func AnnounceEmitted(p *Packet) uint64 {
	offset := identityPubKeyLen + IdentityNameHashLength/8
	end := offset + announceRandomHashLen
	if end > len(p.Data) {
		end = len(p.Data)
	}
	return timebaseFromRandomBlob(p.Data[offset:end])
}

func AddPacketHash(hash []byte) {
	if Owner != nil && Owner.IsConnectedToSharedInstance {
		return
	}
	key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(hash)
	if !ok {
		return
	}
	packetHashMu.Lock()
	PacketHashSet[key] = struct{}{}
	packetHashMu.Unlock()
}

func HasPath(hash []byte) bool {
	if key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(hash); ok {
		pathTableMu.RLock()
		entry := pathTable[key]
		pathTableMu.RUnlock()
		return entry != nil
	}
	return false
}

// ExpirePath mirrors Python Transport.expire_path().
func ExpirePath(hash []byte) bool {
	key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(hash)
	if !ok {
		return false
	}
	pathTableMu.Lock()
	entry, exists := pathTable[key]
	if !exists {
		pathTableMu.Unlock()
		return false
	}
	if entry != nil {
		entry.Timestamp = time.Unix(0, 0)
	}
	TablesLastCulled = time.Time{}
	pathTableMu.Unlock()
	return true
}

// MarkPathResponsive mirrors Python Transport.mark_path_responsive().
func MarkPathResponsive(hash []byte) bool {
	key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(hash)
	if !ok {
		return false
	}
	pathTableMu.RLock()
	_, exists := pathTable[key]
	pathTableMu.RUnlock()
	if !exists {
		return false
	}
	pathStatesMu.Lock()
	pathStates[key] = TransportStateResponsive
	pathStatesMu.Unlock()
	return true
}

// MarkPathUnknownState mirrors Python Transport.mark_path_unknown_state().
func MarkPathUnknownState(hash []byte) bool {
	key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(hash)
	if !ok {
		return false
	}
	pathTableMu.RLock()
	_, exists := pathTable[key]
	pathTableMu.RUnlock()
	if !exists {
		return false
	}
	pathStatesMu.Lock()
	pathStates[key] = TransportStateUnknown
	pathStatesMu.Unlock()
	return true
}

// PathIsUnresponsive mirrors Python Transport.path_is_unresponsive().
func PathIsUnresponsive(hash []byte) bool {
	key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(hash)
	if !ok {
		return false
	}
	pathStatesMu.RLock()
	state, ok := pathStates[key]
	pathStatesMu.RUnlock()
	return ok && state == TransportStateUnresponsive
}

// MarkPathUnresponsive mirrors Python Transport.mark_path_unresponsive().
func MarkPathUnresponsive(hash []byte) bool {
	key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(hash)
	if !ok {
		return false
	}
	pathTableMu.RLock()
	_, exists := pathTable[key]
	pathTableMu.RUnlock()
	if !exists {
		return false
	}
	pathStatesMu.Lock()
	pathStates[key] = TransportStateUnresponsive
	pathStatesMu.Unlock()
	return true
}

func HopsTo(hash []byte) int {
	var entry *PathEntry
	if key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(hash); ok {
		pathTableMu.RLock()
		entry = pathTable[key]
		pathTableMu.RUnlock()
	}
	if entry != nil {
		return entry.Hops
	}
	return PathfinderMaxHops
}

func NextHopInterfaceBitrate(destinationHash []byte) *int {
	var entry *PathEntry
	if key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(destinationHash); ok {
		pathTableMu.RLock()
		entry = pathTable[key]
		pathTableMu.RUnlock()
	}
	if entry == nil || entry.RecvInterface == nil {
		return nil
	}
	b := entry.RecvInterface.Bitrate
	return &b
}

func NextHopPerBitLatency(destinationHash []byte) *float64 {
	br := NextHopInterfaceBitrate(destinationHash)
	if br == nil || *br == 0 {
		return nil
	}
	v := 1.0 / float64(*br)
	return &v
}

func NextHopPerByteLatency(destinationHash []byte) *float64 {
	perBit := NextHopPerBitLatency(destinationHash)
	if perBit == nil {
		return nil
	}
	v := (*perBit) * 8.0
	return &v
}

func FirstHopTimeout(destinationHash []byte) time.Duration {
	perByte := NextHopPerByteLatency(destinationHash)
	if perByte == nil {
		return time.Duration(DEFAULT_PER_HOP_TIMEOUT * float64(time.Second))
	}
	timeout := float64(MTU)*(*perByte) + DEFAULT_PER_HOP_TIMEOUT
	return time.Duration(timeout * float64(time.Second))
}

// ExtraLinkProofTimeout mirrors Python Transport.extra_link_proof_timeout().
func ExtraLinkProofTimeout(ifc *Interface) time.Duration {
	if ifc == nil || ifc.Bitrate == 0 {
		return 0
	}
	seconds := (float64(MTU) * 8.0) / float64(ifc.Bitrate)
	return time.Duration(seconds * float64(time.Second))
}

// AwaitPath mirrors Python Transport.await_path().
// It requests a path if needed and blocks until the path is available or the
// timeout expires. A timeout <= 0 behaves like Python's None default.
func AwaitPath(destinationHash []byte, timeout float64, onInterface *Interface) bool {
	if HasPath(destinationHash) {
		return true
	}

	if timeout <= 0 {
		timeout = TransportPathRequestTimeout
	}

	if onInterface != nil {
		RequestPath(destinationHash, onInterface, nil, false)
	} else {
		RequestPath(destinationHash, nil, nil, false)
	}

	deadline := time.Now().Add(time.Duration(timeout * float64(time.Second)))
	for !HasPath(destinationHash) && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}

	return HasPath(destinationHash)
}

// RequestPath mirrors Python Transport.request_path().
// Python does not suppress explicit request_path() calls just because a local
// path entry already exists. Callers like rnid rely on this to refresh an
// identity from the network when a path is known but the announce data is not.
func RequestPath(hash []byte, onInterface *Interface, tag []byte, recursive bool) {
	if len(tag) == 0 {
		tag = IdentityGetRandomHash()
	}

	var payload []byte
	if TransportEnabled() && TransportIdentity != nil {
		payload = make([]byte, 0, truncatedHashBytes*3)
		payload = append(payload, hash...)
		payload = append(payload, TransportIdentity.Hash...)
		payload = append(payload, tag...)
	} else {
		payload = make([]byte, 0, truncatedHashBytes*2)
		payload = append(payload, hash...)
		payload = append(payload, tag...)
	}

	// If this is a recursive path request on a specific interface, respect announce cap.
	if onInterface != nil && recursive {
		// Python behaviour:
		// - Block recursive path requests if the interface has queued announces.
		// - Block if announce cap is currently active (now < announce_allowed_at).
		// - Otherwise, update announce_allowed_at based on tx_time/announce_cap.
		if onInterface.HasQueuedAnnounces() {
			Log(fmt.Sprintf("Blocking recursive path request on %v due to queued announces", onInterface), LogExtreme)
			return
		}
		now := time.Now()
		if allowedAt := onInterface.AnnounceAllowedAtTime(); !allowedAt.IsZero() && now.Before(allowedAt) {
			Log(fmt.Sprintf("Blocking recursive path request on %v due to active announce cap", onInterface), LogExtreme)
			return
		}

		// tx_time = ((len(data)+HEADER_MINSIZE)*8)/bitrate
		txTime := (float64(len(payload)+HEADER_MINSIZE) * 8.0) / float64(onInterface.Bitrate)
		waitTime := txTime / onInterface.AnnounceCap
		onInterface.SetAnnounceAllowedAt(now.Add(time.Duration(waitTime * float64(time.Second))))
	}

	dst, err := NewDestination(nil, DestinationOUT, DestinationPLAIN, "rnstransport", "path", "request")
	if err != nil {
		Log(fmt.Sprintf("Could not create path request destination: %v", err), LogError)
		return
	}

	p := NewPacket(dst, payload,
		PacketDATA,
		PacketNONE,
		Broadcast,
		HeaderType1,
		nil,
		onInterface,
		false,
		FlagUnset,
	)
	if p == nil {
		return
	}
	_ = p.Send()
	var key hashKey
	copy(key[:], hash)
	pathRequestMu.Lock()
	lastPathRequest[key] = time.Now()
	pathRequestMu.Unlock()
}

// SharedConnectionDisappeared mirrors the Python behaviour when the local
// shared-instance connection drops.
func SharedConnectionDisappeared() {
	linkMu.Lock()
	for _, link := range ActiveLinks {
		if link != nil {
			link.Teardown()
		}
	}
	for _, link := range PendingLinks {
		if link != nil {
			link.Teardown()
		}
	}
	linkMu.Unlock()

	announceMu.Lock()
	announceTable = make(map[hashKey]*announceEntry)
	heldAnnounces = make(map[hashKey]*heldAnnounce)
	announceMu.Unlock()

	pathTableMu.Lock()
	pathTable = make(map[hashKey]*PathEntry)
	pathTableMu.Unlock()

	reverseTableMu.Lock()
	reverseTable = make(map[hashKey]*reverseEntry)
	reverseTableMu.Unlock()

	tunnelsMu.Lock()
	tunnels = make(map[string]*tunnelEntry)
	tunnelsMu.Unlock()

	discoveryPathRequestsMu.Lock()
	discoveryPathRequests = make(map[hashKey]*discoveryPathRequest)
	discoveryPathRequestsMu.Unlock()

	linkTableMu.Lock()
	linkTable = make(map[hashKey]*linkEntry)
	linkTableMu.Unlock()
}

// SharedConnectionReappeared mirrors the Python behaviour when a shared
// connection returns and single destinations should re-announce.
func SharedConnectionReappeared() {
	if Owner == nil || !Owner.IsConnectedToSharedInstance {
		return
	}
	for _, dst := range Destinations {
		if dst.Type == DestinationSINGLE {
			dst.Announce(nil, true, nil, nil, true)
		}
	}
}

// TransportActiveLinks is a compatibility shim for callers that still use the
// old helper name.
func TransportActiveLinks() []*Link {
	linkMu.Lock()
	defer linkMu.Unlock()
	return append([]*Link(nil), ActiveLinks...)
}

type defaultTransportBackend struct{}

func Cache(p *Packet, force bool) {
	if p == nil {
		return
	}
	if !force && !ShouldCache(p) {
		return
	}
	if Owner == nil || len(Owner.CachePath) == 0 {
		return
	}
	packetHash := p.GetHash()
	if len(packetHash) == 0 {
		return
	}

	// Python parity: cache = umsgpack([packet.raw, interface_reference])
	var ifaceRef any
	if p.ReceivingInterface != nil {
		ifaceRef = p.ReceivingInterface.String()
	}
	buf, err := umsgpack.Packb([]any{p.Raw, ifaceRef})
	if err != nil {
		return
	}

	// Announces are stored in cachepath/announces/, everything else in cachepath/.
	var path string
	if p.Type == PacketANNOUNCE {
		path = filepath.Join(Owner.CachePath, "announces", hex.EncodeToString(packetHash))
	} else {
		path = filepath.Join(Owner.CachePath, hex.EncodeToString(packetHash))
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, buf, 0o600); err != nil {
		Log(fmt.Sprintf("Error writing packet to cache. The contained exception was: %v", err), LogError)
	}
}

func cleanCache() {
	if Owner.IsConnectedToSharedInstance {
		return
	}
	cleanAnnounceCache()
	LastCacheCleaned = time.Now()
}

func cleanAnnounceCache() {
	if Owner == nil || len(Owner.CachePath) == 0 {
		return
	}
	targetPath := filepath.Join(Owner.CachePath, "announces")
	if st, err := os.Stat(targetPath); err != nil || st.IsDir() {
		return
	}
	st := time.Now()
	activePaths := make(map[string]struct{})
	pathTableMu.RLock()
	for _, entry := range pathTable {
		if entry == nil || len(entry.PacketHash) == 0 {
			continue
		}
		activePaths[string(entry.PacketHash)] = struct{}{}
	}
	pathTableMu.RUnlock()
	tunnelsMu.RLock()
	for _, te := range tunnels {
		if te == nil || te.Paths == nil {
			continue
		}
		for _, pe := range te.Paths {
			if pe == nil || len(pe.PacketHash) == 0 {
				continue
			}
			activePaths[string(pe.PacketHash)] = struct{}{}
		}
	}
	tunnelsMu.RUnlock()

	removed := 0
	_ = filepath.WalkDir(targetPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil
		}
		name := filepath.Base(path)
		hash, decodeErr := hex.DecodeString(name)
		if decodeErr != nil || len(hash) == 0 {
			_ = os.Remove(path)
			removed++
			return nil
		}
		if _, ok := activePaths[string(hash)]; !ok {
			_ = os.Remove(path)
			removed++
		}
		return nil
	})
	if removed > 0 {
		Log(fmt.Sprintf("Removed %d cached announces in %s", removed, PrettyTime(time.Since(st).Seconds(), true, false)), LogDebug)
	}
}

// ShouldCache mirrors Python Transport.should_cache() in spirit. The Go port
// currently only guarantees cache semantics for announce packets and explicit
// forced caching.
func ShouldCache(p *Packet) bool {
	if p == nil {
		return false
	}
	// Python parity: currently returns False for all packets.
	return false
}

func CacheRequest(hash []byte, link *Link) {
	if cached := getCachedPacket(hash, ""); cached != nil {
		Inbound(cached.Raw, cached.ReceivingInterface)
		return
	}
	if link == nil {
		return
	}
	var attached *Interface
	if ifc, ok := link.AttachedInterface.(*Interface); ok && ifc != nil {
		attached = ifc
	} else if link.Destination != nil {
		var entry *PathEntry
		if key, ok := func(hash []byte) (hashKey, bool) {
			if len(hash) < truncatedHashBytes {
				return hashKey{}, false
			}
			var key hashKey
			copy(key[:], hash[:truncatedHashBytes])
			return key, true
		}(link.Destination.Hash); ok {
			pathTableMu.RLock()
			entry = pathTable[key]
			pathTableMu.RUnlock()
		}
		if entry != nil && entry.RecvInterface != nil {
			attached = entry.RecvInterface
		}
	}
	req := NewPacket(link, hash, PacketTypeData, PacketCONTEXT_CACHE_REQUEST, Broadcast, HeaderType1, nil, attached, false, FlagUnset)
	if req != nil {
		req.Send()
	}
}

func cacheRequestPacket(packet *Packet) bool {
	if len(packet.Data) != HashLengthBytes {
		return false
	}
	cached := getCachedPacket(packet.Data, "")
	if cached == nil {
		return false
	}
	Inbound(cached.Raw, cached.ReceivingInterface)
	return true
}

func getCachedPacket(hash []byte, packetType string) (pkt *Packet) {
	defer func() {
		if rec := recover(); rec != nil {
			Log("Exception occurred while getting cached packet.", LogError)
			Log(fmt.Sprintf("The contained exception was: %v", rec), LogError)
			pkt = nil
		}
	}()
	path := filepath.Join(Owner.CachePath, hex.EncodeToString(hash))
	if packetType == "announce" {
		path = filepath.Join(Owner.CachePath, "announces", hex.EncodeToString(hash))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cached []any
	if err := umsgpack.Unpackb(data, &cached); err != nil {
		return nil
	}
	raw, ok := cached[0].([]byte)
	if !ok {
		return nil
	}
	var recvIf *Interface
	if ref, ok := cached[1].(string); ok {
		for _, ifc := range Interfaces {
			if ifc != nil && ifc.String() == ref {
				recvIf = ifc
				break
			}
		}
	}
	pkt = &Packet{Raw: raw, Data: raw, Packed: true, FromPacked: true, ReceivingInterface: recvIf}
	return pkt
}

func InterfaceToSharedInstance(ifc *Interface) bool {
	return ifc != nil && ifc.LocalIsSharedClient
}

func IsLocalClientInterface(ifc *Interface) bool {
	if ifc == nil {
		return false
	}
	if ifc.Parent != nil && ifc.Parent.LocalIsSharedInstance {
		return true
	}
	return false
}

// FromLocalClient mirrors Python Transport.from_local_client().
// Returns true if the packet was received from a local client interface.
func FromLocalClient(p *Packet) bool {
	return IsLocalClientInterface(p.ReceivingInterface)
}

func NextHop(hash []byte) []byte {
	var entry *PathEntry
	if key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(hash); ok {
		pathTableMu.RLock()
		entry = pathTable[key]
		pathTableMu.RUnlock()
	}
	if entry != nil {
		return append([]byte(nil), entry.NextHop...)
	}
	return nil
}

// NextHopInterfaceHWMTU mirrors Python Transport.next_hop_interface_hw_mtu().
// Returns nil if unknown.
func NextHopInterfaceHWMTU(hash []byte) *int {
	var entry *PathEntry
	if key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(hash); ok {
		pathTableMu.RLock()
		entry = pathTable[key]
		pathTableMu.RUnlock()
	}
	if entry != nil && entry.RecvInterface != nil {
		if entry.RecvInterface.AutoconfigureMTU || entry.RecvInterface.FixedMTU {
			mtu := entry.RecvInterface.HWMTU
			return &mtu
		}
	}
	return nil
}

// NextHopInterface mirrors Python Transport.next_hop_interface().
// Returns the interface for the next hop to the specified destination, or nil
// if the interface is unknown.
func NextHopInterface(hash []byte) *Interface {
	var entry *PathEntry
	if key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(hash); ok {
		pathTableMu.RLock()
		entry = pathTable[key]
		pathTableMu.RUnlock()
	}
	if entry != nil {
		return entry.RecvInterface
	}
	return nil
}
