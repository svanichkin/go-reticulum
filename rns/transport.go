package rns

import (
	"bytes"
	"crypto/ecdh"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	Cryptography "github.com/svanichkin/go-reticulum/rns/cryptography"
	umsgpack "github.com/svanichkin/go-reticulum/rns/vendor"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
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
	MaxReceipts                 = 1024
	TransportPathRequestTimeout = 15.0
	// DestinationTimeout mirrors Python Transport.DESTINATION_TIMEOUT.
	DestinationTimeout      = 7 * 24 * time.Hour
	packetCacheMaxEntries   = 512
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
)

var (
	announceQueueTTL = time.Duration(QUEUED_ANNOUNCE_LIFE) * time.Second
)

type hashKey [truncatedHashBytes]byte

// AnnounceHandler mirrors the Python announce handler callback.
// If AspectFilter returns a non-empty string, only announces whose NAME_HASH
// matches the filter will be delivered.
type AnnounceHandler interface {
	AspectFilter() string
	ReceivedAnnounce(destinationHash []byte, announcedIdentity *Identity, appData []byte)
}

var (
	Owner *Reticulum

	// TransportIdentity is the identity used for transport control destinations
	// like remote management. It is initialised in Start() if missing.
	TransportIdentity *Identity

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

	// jobsMu serialises Jobs() (write-lock) against Inbound/Outbound
	// (read-lock).  This replaces the old JobsLocked/JobsRunning booleans
	// which were bare flags with no memory barriers — a data race that
	// caused silent deadlocks when multiple goroutines called Outbound
	// concurrently (e.g. teardown packets from watchdog goroutines).
	jobsMu sync.RWMutex
	JobInterval      = 250 * time.Millisecond
	LinksLastChecked time.Time
	LinksCheckInt    = time.Second
	ReceiptsLast     time.Time
	ReceiptsCheckInt = time.Second
	AnnLast          time.Time
	AnnCheckInt      = time.Second

	pathTable       = make(map[hashKey]*PathEntry)
	pathTableMu     sync.RWMutex
	packetHashMu    sync.RWMutex
	pathRequestMu   sync.Mutex
	linkMu          sync.Mutex
	announceMu      sync.Mutex
	lastPathRequest = make(map[hashKey]time.Time)
	announceTable   = make(map[hashKey]*announceEntry)
	linkTable       = make(map[hashKey]*linkEntry)

	packetCache   = make(map[string]*cachedPacket)
	packetCacheMu sync.RWMutex

	announceRateTable = make(map[hashKey]*announceRateEntry)
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
	pathRequestDest           *Destination
	tunnelSynthesizeDest      *Destination
	controlDestinations       []*Destination
	remoteManagementActive    bool

	tunnelsMu sync.RWMutex
	tunnels   = make(map[string]*tunnelEntry)

	destinationsMu sync.Mutex

	pathStatesMu sync.RWMutex
	pathStates   = make(map[hashKey]uint8)

	packetHashlistSaveMu sync.Mutex
	savingPacketHashlist bool

	discoveryTagsMu    sync.Mutex
	discoveryPRTags    = make(map[string]struct{})
	discoveryPRTagFIFO []string
	maxPRTags          = 32000

	pendingLocalPathRequestsMu sync.Mutex
	pendingLocalPathRequests   = make(map[hashKey]*Interface)
	pendingPRsLastChecked      time.Time
	pendingPRsCheckInterval    = 30 * time.Second

	localStatsMu   sync.RWMutex
	localRSSICache = make(map[string]float64)
	localSNRCache  = make(map[string]float64)
	localQCache    = make(map[string]float64)
	localStatsFIFO []string
	localStatsMax  = 512

	announceHandlersMu sync.RWMutex
	announceHandlers   []AnnounceHandler

	// Store additional transport tables here as well:
	// AnnounceTable, PathTable, ReverseTable, LinkTable, DiscoveryPathRequests, ...
)

// RegisterAnnounceHandler registers an announce handler with the transport.
// This is the Go equivalent of `RNS.Transport.register_announce_handler()`.
func RegisterAnnounceHandler(handler AnnounceHandler) {
	if handler == nil {
		return
	}
	announceHandlersMu.Lock()
	for _, existing := range announceHandlers {
		if existing == handler {
			announceHandlersMu.Unlock()
			return
		}
	}
	announceHandlers = append(announceHandlers, handler)
	announceHandlersMu.Unlock()
}

// DeregisterAnnounceHandler removes a previously registered announce handler.
func DeregisterAnnounceHandler(handler AnnounceHandler) bool {
	if handler == nil {
		return false
	}
	announceHandlersMu.Lock()
	defer announceHandlersMu.Unlock()
	for i, existing := range announceHandlers {
		if existing == handler {
			announceHandlers = append(announceHandlers[:i], announceHandlers[i+1:]...)
			return true
		}
	}
	return false
}

func announceNameHashAndAppData(packet *Packet) (nameHash []byte, appData []byte, announced *Identity, ok bool) {
	if packet == nil || packet.PacketType != PacketTypeAnnounce {
		return nil, nil, nil, false
	}
	data := packet.Data
	nameHashLen := IdentityNameHashLength / 8
	minLen := identityPubKeyLen + nameHashLen + announceRandomHashLen + ed25519.SignatureSize
	if len(data) < minLen {
		return nil, nil, nil, false
	}

	publicKey := data[:identityPubKeyLen]
	offset := identityPubKeyLen
	nameHash = data[offset : offset+nameHashLen]
	offset += nameHashLen
	offset += announceRandomHashLen

	if packet.ContextFlag == FlagSet {
		if len(data) < offset+x25519KeyLen+ed25519.SignatureSize {
			return nil, nil, nil, false
		}
		offset += x25519KeyLen
	}
	if len(data) < offset+ed25519.SignatureSize {
		return nil, nil, nil, false
	}
	offset += ed25519.SignatureSize
	if len(data) > offset {
		appData = data[offset:]
	}

	announced = &Identity{curve: ecdh.X25519()}
	if err := announced.LoadPublicKey(publicKey); err != nil {
		return nil, nil, nil, false
	}
	return append([]byte(nil), nameHash...), append([]byte(nil), appData...), announced, true
}

func announceHandlerWants(handler AnnounceHandler, announceNameHash []byte) bool {
	if handler == nil {
		return false
	}
	filter := strings.TrimSpace(handler.AspectFilter())
	if filter == "" {
		return true
	}
	want := FullHash([]byte(filter))[:IdentityNameHashLength/8]
	return bytes.Equal(want, announceNameHash)
}

func dispatchAnnounceHandlers(packet *Packet) {
	if packet == nil {
		return
	}
	nameHash, appData, announced, ok := announceNameHashAndAppData(packet)
	if !ok {
		return
	}

	announceHandlersMu.RLock()
	handlers := append([]AnnounceHandler(nil), announceHandlers...)
	announceHandlersMu.RUnlock()

	for _, h := range handlers {
		if !announceHandlerWants(h, nameHash) {
			continue
		}
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					Logf(LogError, "Announce handler panic: %v", rec)
				}
			}()
			h.ReceivedAnnounce(copyBytes(packet.DestinationHash), announced, appData)
		}()
	}
}

// NotifyAnnounceHandlers triggers delivery to registered announce handlers for
// a decoded announce packet. This is primarily useful for applications/tests
// that already have a parsed announce packet and want to reuse the handler
// mechanism without going through full transport ingress.
func NotifyAnnounceHandlers(packet *Packet) {
	dispatchAnnounceHandlers(packet)
}

func removeLocalClientInterface(ifc *Interface) {
	if ifc == nil {
		return
	}
	for idx, existing := range LocalClientInterfaces {
		if existing == ifc {
			LocalClientInterfaces = append(LocalClientInterfaces[:idx], LocalClientInterfaces[idx+1:]...)
			return
		}
	}
}

func removeInterface(ifc *Interface) {
	if ifc == nil {
		return
	}
	if len(ifc.TunnelID) == HashLengthBytes {
		VoidTunnelInterface(ifc.TunnelID)
	}
	ifc.Detach()
	for idx, existing := range Interfaces {
		if existing == ifc {
			Interfaces = append(Interfaces[:idx], Interfaces[idx+1:]...)
			return
		}
	}
}

// TransportRegisterDestination registers a destination with the transport.
// The Python implementation maintains multiple internal maps; for now we keep
// a simple list and ensure uniqueness.
func TransportRegisterDestination(d *Destination) {
	if d == nil {
		return
	}
	destinationsMu.Lock()
	defer destinationsMu.Unlock()
	for _, existing := range Destinations {
		if existing == d {
			return
		}
		if existing != nil && len(existing.Hash()) > 0 && bytes.Equal(existing.Hash(), d.Hash()) {
			return
		}
	}
	Destinations = append(Destinations, d)
}

// TransportDeregisterDestination removes a destination from the transport registry.
// Mirrors Python Transport.deregister_destination().
func TransportDeregisterDestination(d *Destination) bool {
	if d == nil {
		return false
	}
	destinationsMu.Lock()
	removed := false
	for i := 0; i < len(Destinations); i++ {
		existing := Destinations[i]
		if existing == d || (existing != nil && len(existing.Hash()) > 0 && bytes.Equal(existing.Hash(), d.Hash())) {
			Destinations = append(Destinations[:i], Destinations[i+1:]...)
			removed = true
			i--
		}
	}
	destinationsMu.Unlock()
	if removed && Owner != nil && !Owner.IsConnectedToSharedInstance {
		Owner.ShouldPersistData()
	}
	return removed
}

// AddInterface appends an interface to the transport interface list.
// The Go port currently supports only the shared in-memory representation.
func AddInterface(ifc *Interface) {
	if ifc == nil {
		return
	}
	Interfaces = append(Interfaces, ifc)
	PrioritiseInterfaces()
}

// DetachInterfaces best-effort detaches all interfaces.
func DetachInterfaces() {
	for _, ifc := range Interfaces {
		if ifc == nil {
			continue
		}
		ifc.Detach()
	}
}

// AddRemoteManagementAllowed adds an identity hash to the remote-management ACL.
func AddRemoteManagementAllowed(hash []byte) {
	if len(hash) == 0 {
		return
	}
	key := string(hash)
	remoteManagementAllowedMu.Lock()
	if _, ok := remoteManagementAllowed[key]; !ok {
		remoteManagementAllowed[key] = append([]byte(nil), hash...)
	}
	remoteManagementAllowedMu.Unlock()
}

// RemoteManagementAllowedContains reports whether the hash exists in the ACL.
func RemoteManagementAllowedContains(hash []byte) bool {
	if len(hash) == 0 {
		return false
	}
	key := string(hash)
	remoteManagementAllowedMu.RLock()
	_, ok := remoteManagementAllowed[key]
	remoteManagementAllowedMu.RUnlock()
	return ok
}

func remoteManagementAllowedList() [][]byte {
	remoteManagementAllowedMu.RLock()
	defer remoteManagementAllowedMu.RUnlock()
	out := make([][]byte, 0, len(remoteManagementAllowed))
	for _, v := range remoteManagementAllowed {
		out = append(out, append([]byte(nil), v...))
	}
	return out
}

const (
	pathExpiration         = 7 * 24 * time.Hour
	pathRequestMinInterval = 20 * time.Second
	reverseTimeout         = 8 * time.Minute
	tablesCullInterval     = 5 * time.Second
	cacheCleanInterval     = 5 * time.Minute
	packetCacheLifetime    = 10 * time.Minute
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
	Packet  *Packet
	Release time.Time
}

type announceRateEntry struct {
	Last           time.Time
	RateViolations int
	BlockedUntil   time.Time
	Timestamps     []time.Time
}

type cachedPacket struct {
	Raw        []byte
	Interface  *Interface
	StoredAt   time.Time
	PacketHash []byte
}

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

// -------- announce queue helpers --------

type announceEnqueueOptions struct {
	delay            time.Duration
	blockRebroadcast bool
	attached         *Interface
}

// AnnounceOption configures how QueueAnnounce behaves.
type AnnounceOption func(*announceEnqueueOptions)

// WithAnnounceDelay schedules announce retransmission after a fixed delay.
func WithAnnounceDelay(d time.Duration) AnnounceOption {
	return func(o *announceEnqueueOptions) {
		o.delay = d
	}
}

// WithAnnounceImmediate schedules the announce for immediate retransmission.
func WithAnnounceImmediate() AnnounceOption {
	return func(o *announceEnqueueOptions) {
		o.delay = 0
	}
}

// WithAnnounceBlockRebroadcasts forces queued packets to use PATH_RESPONSE context.
func WithAnnounceBlockRebroadcasts(block bool) AnnounceOption {
	return func(o *announceEnqueueOptions) {
		o.blockRebroadcast = block
	}
}

// WithAnnounceAttachedInterface tags the retransmission with a specific interface.
func WithAnnounceAttachedInterface(ifc *Interface) AnnounceOption {
	return func(o *announceEnqueueOptions) {
		o.attached = ifc
	}
}

func Start(owner *Reticulum) {
	StartTime = time.Now()
	Owner = owner
	// jobsMu is initialized at declaration; no special startup needed.
	// Python: Transport.cache_last_cleaned = time.time() + 60
	LastCacheCleaned = time.Now().Add(60 * time.Second)
	ensureTransportIdentity()
	loadPacketHashlist()
	loadDestinationTable()
	loadTunnelTable()
	configureControlDestinations()

	// ... the rest of start() is ported here:
	// loading identities, tables, probe destination, etc.

	// sort interfaces by bitrate
	PrioritiseInterfaces()

	// Python parity: Synthesize tunnels for any interfaces wanting it.
	for _, ifc := range Interfaces {
		if ifc == nil {
			continue
		}
		ifc.TunnelID = nil
		if ifc.WantsTunnel {
			SynthesizeTunnel(ifc)
			ifc.WantsTunnel = false
		}
	}

	// start background loops
	go JobLoop()
	go CountTrafficLoop()
}

type serialisedPathEntry struct {
	DestinationHash []byte
	Timestamp       float64
	NextHop         []byte
	Hops            int
	Expires         float64
	RandomBlobs     [][]byte
	InterfaceHash   []byte
	PacketHash      []byte
}

func interfaceHash(ifc *Interface) []byte {
	if ifc == nil {
		return nil
	}
	return FullHash([]byte(ifc.String()))
}

func findInterfaceFromHash(hash []byte) *Interface {
	if len(hash) == 0 {
		return nil
	}
	for _, ifc := range Interfaces {
		if ifc == nil {
			continue
		}
		if bytes.Equal(interfaceHash(ifc), hash) {
			return ifc
		}
	}
	return nil
}

func announceCachePath(packetHash []byte) string {
	if Owner == nil || len(packetHash) == 0 {
		return ""
	}
	return filepath.Join(Owner.CachePath, "announces", hex.EncodeToString(packetHash))
}

func loadDestinationTable() {
	if Owner == nil || Owner.IsConnectedToSharedInstance || !TransportEnabled() {
		return
	}
	path := filepath.Join(Owner.StoragePath, "destination_table")
	if !fileExists(path) {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		Logf(LogError, "Could not load destination table from storage: %v", err)
		return
	}

	var rawEntries [][]any
	if err := umsgpack.Unpackb(data, &rawEntries); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			backupPath := fmt.Sprintf("%s.corrupt.%d", path, time.Now().UnixNano())
			if renameErr := os.Rename(path, backupPath); renameErr != nil {
				_ = os.Remove(path)
			} else {
				Logf(LogWarning, "Destination table storage was truncated; moved to %s", backupPath)
			}
			return
		}
		Logf(LogError, "Could not decode destination table from storage: %v", err)
		return
	}

	loaded := 0
	for _, re := range rawEntries {
		if len(re) < 8 {
			continue
		}
		dst, _ := asBytesValue(re[0])
		if len(dst) != truncatedHashBytes {
			continue
		}
		ts, _ := asFloatValue(re[1])
		nextHop, _ := asBytesValue(re[2])
		hops, _ := asIntValue(re[3])
		exp, _ := asFloatValue(re[4])

		var randomBlobs [][]byte
		if rb, ok := re[5].([]any); ok {
			for _, item := range rb {
				if b, ok := asBytesValue(item); ok && len(b) > 0 {
					randomBlobs = append(randomBlobs, b)
				}
			}
		}

		ifHash, _ := asBytesValue(re[6])
		recvIf := findInterfaceFromHash(ifHash)
		packetHash, _ := asBytesValue(re[7])

		rawAnnounce, err := os.ReadFile(announceCachePath(packetHash))
		if err != nil || len(rawAnnounce) == 0 || recvIf == nil {
			continue
		}

		announce := NewPacket(nil, rawAnnounce)
		if announce == nil || !announce.Unpack() {
			continue
		}
		announce.Hops += 1

		key, ok := makeHashKey(dst)
		if !ok {
			continue
		}
		entry := &PathEntry{
			NextHop:       append([]byte(nil), nextHop...),
			RecvInterface: recvIf,
			Hops:          hops,
			Timestamp:     time.Unix(int64(ts), 0),
			ExpiresAt:     time.Unix(int64(exp), 0),
			RandomBlobs:   randomBlobs,
			PacketHash:    append([]byte(nil), packetHash...),
		}

		pathTableMu.Lock()
		pathTable[key] = entry
		pathTableMu.Unlock()
		loaded++
	}
	if loaded > 0 {
		Logf(LogVerbose, "Loaded %d path table entries from storage", loaded)
	}
}

func loadTunnelTable() {
	if Owner == nil || Owner.IsConnectedToSharedInstance || !TransportEnabled() {
		return
	}
	path := filepath.Join(Owner.StoragePath, "tunnels")
	if !fileExists(path) {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		Logf(LogError, "Could not load tunnel table from storage: %v", err)
		return
	}

	var rawTunnels []any
	if err := umsgpack.Unpackb(data, &rawTunnels); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			backupPath := fmt.Sprintf("%s.corrupt.%d", path, time.Now().UnixNano())
			if renameErr := os.Rename(path, backupPath); renameErr != nil {
				_ = os.Remove(path)
			} else {
				Logf(LogWarning, "Tunnel table storage was truncated; moved to %s", backupPath)
			}
			return
		}
		Logf(LogError, "Could not decode tunnel table from storage: %v", err)
		return
	}

	loaded := 0
	for _, t := range rawTunnels {
		tlist, ok := t.([]any)
		if !ok || len(tlist) < 4 {
			continue
		}
		tunnelID, _ := asBytesValue(tlist[0])
		if len(tunnelID) != HashLengthBytes {
			// Python uses full hash (32 bytes) for tunnel_id
			continue
		}
		ifHash, _ := asBytesValue(tlist[1])
		serialisedPathsAny := tlist[2]
		expires, _ := asFloatValue(tlist[3])
		ifc := findInterfaceFromHash(ifHash)

		te := &tunnelEntry{
			ID:        append([]byte(nil), tunnelID...),
			Interface: ifc,
			ExpiresAt: time.Unix(int64(expires), 0),
			Paths:     make(map[string]*tunnelPathEntry),
		}

		if serialisedPaths, ok := serialisedPathsAny.([]any); ok && len(serialisedPaths) > 0 {
			for _, sp := range serialisedPaths {
				elist, ok := sp.([]any)
				if !ok || len(elist) < 8 {
					continue
				}
				dstHash, _ := asBytesValue(elist[0])
				if len(dstHash) != truncatedHashBytes {
					continue
				}
				ts, _ := asFloatValue(elist[1])
				receivedFrom, _ := asBytesValue(elist[2])
				hops, _ := asIntValue(elist[3])
				exp, _ := asFloatValue(elist[4])

				blobs := make([][]byte, 0)
				if blobsAny, ok := elist[5].([]any); ok {
					seen := make(map[string]struct{})
					for _, b := range blobsAny {
						bb, ok := asBytesValue(b)
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

				packetHash, _ := asBytesValue(elist[7])
				if len(packetHash) == 0 {
					continue
				}

				te.Paths[string(dstHash)] = &tunnelPathEntry{
					Timestamp:    time.Unix(int64(ts), 0),
					ReceivedFrom: append([]byte(nil), receivedFrom...),
					Hops:         hops,
					ExpiresAt:    time.Unix(int64(exp), 0),
					RandomBlobs:  blobs,
					PacketHash:   append([]byte(nil), packetHash...),
				}
			}
			if len(te.Paths) == 0 {
				// Match Python: only keep tunnel entries with at least one path.
				continue
			}
		} else {
			// Keep empty path table entries.
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
	if loaded > 0 {
		Logf(LogVerbose, "Loaded %d tunnel table entries from storage", loaded)
	}
}

func ensureTransportIdentity() {
	if TransportIdentity != nil || Owner == nil {
		return
	}
	identityPath := filepath.Join(Owner.StoragePath, "transport_identity")
	if fileExists(identityPath) {
		if id, err := IdentityFromFile(identityPath); err == nil {
			TransportIdentity = id
			Log("Loaded Transport Identity from storage", LogVerbose)
			return
		}
	}

	Log("No valid Transport Identity in storage, creating...", LogVerbose)
	if id, err := NewIdentity(); err == nil {
		TransportIdentity = id
		_ = os.MkdirAll(Owner.StoragePath, 0o755)
		_ = TransportIdentity.Save(identityPath)
	}
}

func loadPacketHashlist() {
	if Owner == nil || Owner.IsConnectedToSharedInstance {
		return
	}
	if !TransportEnabled() {
		packetHashMu.Lock()
		PacketHashSet = make(map[hashKey]struct{})
		PacketHashSet2 = make(map[hashKey]struct{})
		packetHashMu.Unlock()
		return
	}
	path := filepath.Join(Owner.StoragePath, "packet_hashlist")
	if !fileExists(path) {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		Logf(LogError, "Could not load packet hashlist from storage: %v", err)
		return
	}
	var hashes [][]byte
	if err := umsgpack.Unpackb(data, &hashes); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			backupPath := fmt.Sprintf("%s.corrupt.%d", path, time.Now().UnixNano())
			if renameErr := os.Rename(path, backupPath); renameErr != nil {
				_ = os.Remove(path)
			} else {
				Logf(LogWarning, "Packet hashlist storage was truncated; moved to %s", backupPath)
			}
			return
		}
		Logf(LogError, "Could not decode packet hashlist from storage: %v", err)
		return
	}
	packetHashMu.Lock()
	for _, h := range hashes {
		if len(h) != truncatedHashBytes {
			continue
		}
		var k hashKey
		copy(k[:], h)
		PacketHashSet[k] = struct{}{}
	}
	packetHashMu.Unlock()
}

// TransportExitHandler best-effort persistence hook mirroring Transport.exit_handler() in Python.
func TransportExitHandler() {
	_ = savePacketHashlist()
	_ = saveDestinationTable()
	_ = saveTunnelTable()
}

// TransportPersistData persists transport tables, mirroring Python's
// Transport.persist_data() / Reticulum.__persist_data() behaviour.
func TransportPersistData() {
	if inst := GetInstance(); inst != nil && inst.IsConnectedToSharedInstance {
		return
	}
	if err := savePacketHashlist(); err != nil {
		Logf(LogError, "Error while saving packet hashlist to disk: %v", err)
	}
	if err := saveDestinationTable(); err != nil {
		Logf(LogError, "Error while saving destination table to disk: %v", err)
	}
	if err := saveTunnelTable(); err != nil {
		Logf(LogError, "Error while saving tunnel table to disk: %v", err)
	}
	lastTablesPersisted = time.Now()
}

func savePacketHashlist() error {
	if Owner == nil || Owner.IsConnectedToSharedInstance {
		return nil
	}
	packetHashlistSaveMu.Lock()
	if savingPacketHashlist {
		packetHashlistSaveMu.Unlock()
		return nil
	}
	savingPacketHashlist = true
	packetHashlistSaveMu.Unlock()
	defer func() {
		packetHashlistSaveMu.Lock()
		savingPacketHashlist = false
		packetHashlistSaveMu.Unlock()
	}()

	if !TransportEnabled() {
		packetHashMu.Lock()
		PacketHashSet = make(map[hashKey]struct{})
		PacketHashSet2 = make(map[hashKey]struct{})
		packetHashMu.Unlock()
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
		return err
	}
	_ = os.MkdirAll(Owner.StoragePath, 0o755)
	path := filepath.Join(Owner.StoragePath, "packet_hashlist")
	return os.WriteFile(path, buf, 0o600)
}

func saveDestinationTable() error {
	if Owner == nil || Owner.IsConnectedToSharedInstance || !TransportEnabled() {
		return nil
	}

	entries := make([][]any, 0)
	now := time.Now()

	pathTableMu.RLock()
	for key, entry := range pathTable {
		if entry == nil {
			continue
		}
		if !entry.ExpiresAt.IsZero() && entry.ExpiresAt.Before(now) {
			continue
		}

		dst := make([]byte, truncatedHashBytes)
		copy(dst, key[:])

		ifHash := interfaceHash(entry.RecvInterface)
		if len(ifHash) == 0 {
			continue
		}
		if len(entry.PacketHash) == 0 {
			continue
		}

		// Ensure announce cache exists on disk.
		cachePath := announceCachePath(entry.PacketHash)
		if cachePath != "" && !fileExists(cachePath) {
			packetCacheMu.RLock()
			cp := packetCache[string(entry.PacketHash)]
			packetCacheMu.RUnlock()
			if cp != nil && len(cp.Raw) > 0 {
				_ = os.MkdirAll(filepath.Dir(cachePath), 0o755)
				_ = os.WriteFile(cachePath, cp.Raw, 0o600)
			}
		}

		entries = append(entries, []any{
			dst,
			float64(entry.Timestamp.Unix()),
			append([]byte(nil), entry.NextHop...),
			entry.Hops,
			float64(entry.ExpiresAt.Unix()),
			dedupeAndTailRandomBlobs(entry.RandomBlobs, persistRandomBlobs),
			ifHash,
			append([]byte(nil), entry.PacketHash...),
		})
	}
	pathTableMu.RUnlock()

	buf, err := umsgpack.Packb(entries)
	if err != nil {
		return err
	}
	_ = os.MkdirAll(Owner.StoragePath, 0o755)
	path := filepath.Join(Owner.StoragePath, "destination_table")
	return os.WriteFile(path, buf, 0o600)
}

func saveTunnelTable() error {
	if Owner == nil || Owner.IsConnectedToSharedInstance || !TransportEnabled() {
		return nil
	}
	now := time.Now()
	serialised := make([][]any, 0)

	tunnelsMu.RLock()
	for _, te := range tunnels {
		if te == nil || len(te.ID) == 0 {
			continue
		}
		if !te.ExpiresAt.IsZero() && te.ExpiresAt.Before(now) {
			continue
		}
		var ifHash []byte
		if te.Interface != nil {
			ifHash = interfaceHash(te.Interface)
		}
		serialisedPaths := make([][]any, 0)
		for dstKey, pe := range te.Paths {
			if pe == nil {
				continue
			}
			dstHash := []byte(dstKey)
			if len(dstHash) != truncatedHashBytes {
				continue
			}
			if !pe.ExpiresAt.IsZero() && pe.ExpiresAt.Before(now) {
				continue
			}
			if len(pe.PacketHash) == 0 {
				continue
			}

			// Ensure announce cache exists on disk, mirroring destination_table persistence.
			cachePath := announceCachePath(pe.PacketHash)
			if cachePath != "" && !fileExists(cachePath) {
				packetCacheMu.RLock()
				cp := packetCache[string(pe.PacketHash)]
				packetCacheMu.RUnlock()
				if cp != nil && len(cp.Raw) > 0 {
					_ = os.MkdirAll(filepath.Dir(cachePath), 0o755)
					_ = os.WriteFile(cachePath, cp.Raw, 0o600)
				}
			}

			blobs := pe.RandomBlobs
			if len(blobs) > persistRandomBlobs {
				blobs = blobs[len(blobs)-persistRandomBlobs:]
			}

			serialisedPaths = append(serialisedPaths, []any{
				append([]byte(nil), dstHash...),
				float64(pe.Timestamp.Unix()),
				append([]byte(nil), pe.ReceivedFrom...),
				pe.Hops,
				float64(pe.ExpiresAt.Unix()),
				copyRandomBlobSlice(blobs),
				ifHash,
				append([]byte(nil), pe.PacketHash...),
			})
		}

		serialised = append(serialised, []any{
			append([]byte(nil), te.ID...),
			ifHash,
			serialisedPaths,
			float64(te.ExpiresAt.Unix()),
		})
	}
	tunnelsMu.RUnlock()

	buf, err := umsgpack.Packb(serialised)
	if err != nil {
		return err
	}
	_ = os.MkdirAll(Owner.StoragePath, 0o755)
	path := filepath.Join(Owner.StoragePath, "tunnels")
	return os.WriteFile(path, buf, 0o600)
}

// -------- prioritisation and traffic counters --------

func PrioritiseInterfaces() {
	defer func() {
		if r := recover(); r != nil {
			Logf(LogError, "Could not prioritise interfaces: %v", r)
		}
	}()

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
					Logf(LogError, "Error while counting traffic: %v", r)
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
					ifc.TrafficCounter = &TrafficCounter{
						TS:  now,
						RXB: ifc.RXB,
						TXB: ifc.TXB,
					}
					continue
				}

				tc := ifc.TrafficCounter
				rxDiff := ifc.RXB - tc.RXB
				txDiff := ifc.TXB - tc.TXB
				tsDiff := now.Sub(tc.TS).Seconds()
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

				tc.RXB = ifc.RXB
				tc.TXB = ifc.TXB
				tc.TS = now
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
	var culled bool

	// TryLock: if Inbound/Outbound is active, skip this job cycle rather
	// than blocking.  This preserves the original semantics where Jobs()
	// yielded to IO traffic (the old "if JobsLocked { return }" pattern)
	// and avoids writer-starvation of the RWMutex blocking all new readers.
	if !jobsMu.TryLock() {
		return
	}
	defer func() {
		jobsMu.Unlock()

		// send collected packets (outside the lock so Outbound can proceed)
		for _, p := range outgoing {
			_ = p.Send()
		}
		// path requests
		for dst, blocked := range pathRequests {
			if blocked == nil {
				RequestPath(dst[:], nil)
			} else {
				for _, ifc := range Interfaces {
					if ifc != blocked {
						RequestPath(dst[:], ifc)
					}
				}
			}
		}
	}()
	shouldGC := false

	now := time.Now()

	// ---- pending and active links ----
	if now.Sub(LinksLastChecked) > LinksCheckInt {
		// pending_links / active_links mirror Python closely:
		// check status, remove CLOSED, rediscover paths, etc.
		handlePendingAndActiveLinks(pathRequests)
		LinksLastChecked = now
	}

	// ---- receipts ----
	if now.Sub(ReceiptsLast) > ReceiptsCheckInt {
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
			if rc.Status != ReceiptSent {
				Receipts = append(Receipts[:i], Receipts[i+1:]...)
			} else {
				i++
			}
		}
		ReceiptsLast = now
	}

	// ---- announces retransmit ----
	if now.Sub(AnnLast) > AnnCheckInt {
		handleAnnounceRetransmit(now, &outgoing)
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
			if desired != nil && interfacePresent(desired) {
				continue
			}
			delete(pendingLocalPathRequests, key)
		}
		pendingLocalPathRequestsMu.Unlock()
		pendingPRsLastChecked = now
	}

	// periodic culling of tables (Python: tables_last_culled / tables_cull_interval)
	if now.Sub(TablesLastCulled) > tablesCullInterval {
		cullTunnels(now)
		if cullReverseAndLinkTables(now) {
			culled = true
		}
		if cullPathTable(now) {
			culled = true
		}
		if cullPathStates() {
			culled = true
		}
		TablesLastCulled = now
	}

	// periodic cleanup of reverse/link tables and caches
	if now.Sub(LastCacheCleaned) > cacheCleanInterval {
		if cleanAnnounceCache(now) {
			culled = true
		}
		if cleanPacketCache(now) {
			culled = true
		}
		LastCacheCleaned = now
	}

	// Periodic persistence of transport tables (Python persist_data()).
	if now.Sub(lastTablesPersisted) > tablesPersistInterval {
		_ = saveDestinationTable()
		_ = savePacketHashlist()
		_ = saveTunnelTable()
		lastTablesPersisted = now
	}

	// Similarly here: pending_local_path_requests, discovery_pr_tags, table culling
	// reverse_table, link_table, path_table, discovery_path_requests, tunnels, path_states
	// and interface.process_held_announces().
	if len(heldAnnounces) > 0 {
		processHeldAnnounces(now, &outgoing)
	}
	for _, ifc := range Interfaces {
		if ifc != nil {
			ifc.ProcessHeldAnnounces(PathfinderMaxHops)
		}
	}

	if shouldGC || culled {
		runtime.GC()
	}
}

func cullTunnels(now time.Time) {
	tunnelsMu.Lock()
	defer tunnelsMu.Unlock()
	removed := 0
	for key, te := range tunnels {
		if te == nil {
			delete(tunnels, key)
			removed++
			continue
		}
		if te.Interface == nil || (!te.ExpiresAt.IsZero() && now.After(te.ExpiresAt)) {
			delete(tunnels, key)
			removed++
		}
	}
	if removed > 0 {
		Logf(LogExtreme, "Removed %d tunnels", removed)
	}
}

// VoidTunnelInterface mirrors Python Transport.void_tunnel_interface().
func VoidTunnelInterface(tunnelID []byte) {
	if len(tunnelID) != HashLengthBytes {
		return
	}
	key := string(tunnelID)
	tunnelsMu.Lock()
	if te := tunnels[key]; te != nil {
		Logf(LogExtreme, "Voiding tunnel interface %v", te.Interface)
		te.Interface = nil
		tunnels[key] = te
	}
	tunnelsMu.Unlock()
}

func handleAnnounceRetransmit(now time.Time, outgoing *[]*Packet) {
	announceMu.Lock()
	defer announceMu.Unlock()

	for key, entry := range announceTable {
		if entry == nil || entry.Packet == nil {
			delete(announceTable, key)
			continue
		}
		if now.After(entry.Expires) {
			delete(announceTable, key)
			continue
		}
		if entry.Next.After(now) {
			continue
		}
		if entry.Retries >= pathfinderRetryLimit {
			delete(announceTable, key)
			continue
		}
		send := cloneAnnouncePacket(entry.Packet)
		if send == nil {
			delete(announceTable, key)
			continue
		}
		if entry.BlockRebroadcasts {
			send.Context = PacketPATH_RESPONSE
		}
		if entry.AttachedInterface != nil {
			send.AttachedInterface = entry.AttachedInterface
		}
		*outgoing = append(*outgoing, send)

		entry.Retries++
		entry.Next = now.Add(pathfinderRetryGrace + randAnnounceDelay())
		announceTable[key] = entry
	}
}

func processHeldAnnounces(now time.Time, outgoing *[]*Packet) {
	announceMu.Lock()
	defer announceMu.Unlock()
	for key, entry := range heldAnnounces {
		if entry == nil || entry.Packet == nil {
			delete(heldAnnounces, key)
			continue
		}
		if entry.Release.After(now) {
			continue
		}
		*outgoing = append(*outgoing, entry.Packet)
		delete(heldAnnounces, key)
	}
}

// QueueAnnounce schedules a packet for rebroadcast, mirroring the behaviour of
// the Python transport announce table. Only ANNOUNCE packets are queued.
func QueueAnnounce(p *Packet, opts ...AnnounceOption) {
	if p == nil || p.Type != PacketANNOUNCE {
		return
	}
	keyHash := announceHash(p)
	if keyHash == nil {
		return
	}

	params := announceEnqueueOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&params)
		}
	}

	clone := cloneAnnouncePacket(p)
	if clone == nil {
		return
	}

	now := time.Now()
	delay := params.delay
	if delay == 0 {
		delay = randAnnounceDelay()
	}

	entry := &announceEntry{
		Packet:            clone,
		Next:              now.Add(delay),
		Retries:           0,
		Timestamp:         now,
		Expires:           now.Add(announceQueueTTL),
		LocalRebroadcasts: 0,
		BlockRebroadcasts: params.blockRebroadcast,
		AttachedInterface: params.attached,
	}

	key := *keyHash
	announceMu.Lock()
	if len(announceTable) >= MAX_QUEUED_ANNOUNCES {
		evictOldestAnnounceLocked()
	}
	announceTable[key] = entry
	announceMu.Unlock()
}

// DropAnnounceQueues clears queued announces and returns the number of entries removed.
func DropAnnounceQueues() int {
	dropped := 0
	announceMu.Lock()
	dropped += len(announceTable)
	announceTable = make(map[hashKey]*announceEntry)
	dropped += len(heldAnnounces)
	heldAnnounces = make(map[hashKey]*heldAnnounce)
	announceMu.Unlock()

	for _, ifc := range Interfaces {
		if ifc == nil {
			continue
		}
		if dropper, ok := any(ifc).(interface{ DropAnnounceQueue() int }); ok {
			dropped += dropper.DropAnnounceQueue()
		}
	}
	return dropped
}

// -------- remote management --------

func configureControlDestinations() {
	if Owner == nil {
		return
	}
	ensureCoreControlDestinations()
	ensureProbeDestination()
	if !RemoteManagementEnabled() || Owner.IsConnectedToSharedInstance || TransportIdentity == nil {
		disableRemoteManagementDestination()
		return
	}
	enableRemoteManagementDestination()
}

func ensureProbeDestination() {
	if Owner == nil {
		return
	}
	if !ProbeDestinationEnabled() || Owner.IsConnectedToSharedInstance || TransportIdentity == nil {
		ProbeDestination = nil
		return
	}
	if ProbeDestination != nil {
		return
	}
	dest, err := NewDestination(TransportIdentity, DestinationIN, DestinationSINGLE, TransportAppName, "probe")
	if err != nil {
		Logf(LogError, "Could not create probe destination: %v", err)
		return
	}
	dest.AcceptsLinks(false)
	_ = dest.SetProofStrategy(DestinationPROVE_ALL)
	dest.Announce(nil, false, nil, nil, true)
	ProbeDestination = dest
	Logf(LogNotice, "Transport Instance will respond to probe requests on %s", dest)
}

func ensureCoreControlDestinations() {
	if pathRequestDest == nil {
		dest, err := NewDestination(nil, DestinationIN, DestinationPLAIN, "rnstransport", "path", "request")
		if err == nil {
			pathRequestDest = dest
			pathRequestDest.SetPacketCallback(pathRequestHandler)
			controlDestinations = append(controlDestinations, dest)
			registerControlHash(dest.Hash())
		} else {
			Logf(LogError, "Could not create path request destination: %v", err)
		}
	}

	if tunnelSynthesizeDest == nil {
		dest, err := NewDestination(nil, DestinationIN, DestinationPLAIN, "rnstransport", "tunnel", "synthesize")
		if err == nil {
			tunnelSynthesizeDest = dest
			tunnelSynthesizeDest.SetPacketCallback(tunnelSynthesizeHandler)
			controlDestinations = append(controlDestinations, dest)
			registerControlHash(dest.Hash())
		} else {
			Logf(LogError, "Could not create tunnel synthesize destination: %v", err)
		}
	}
}

func enableRemoteManagementDestination() {
	if remoteManagementDest == nil {
		dest, err := NewDestination(TransportIdentity, DestinationIN, DestinationSINGLE, "rnstransport", "remote", "management")
		if err != nil {
			Logf(LogError, "Could not create remote management destination: %v", err)
			return
		}
		remoteManagementDest = dest
		controlDestinations = append(controlDestinations, dest)
	}

	allowed := remoteManagementAllowedList()
	if err := remoteManagementDest.RegisterRequestHandler("/status", remoteStatusHandler,
		DestinationALLOW_LIST, allowed); err != nil {
		Logf(LogError, "Could not register remote status handler: %v", err)
		return
	}
	if err := remoteManagementDest.RegisterRequestHandler("/path", remotePathHandler,
		DestinationALLOW_LIST, allowed); err != nil {
		Logf(LogError, "Could not register remote path handler: %v", err)
		remoteManagementDest.DeregisterRequestHandler("/status")
		return
	}

	if !remoteManagementActive {
		registerControlHash(remoteManagementDest.Hash())
		Logf(LogNotice, "Enabled remote management on %s", remoteManagementDest)
	}
	remoteManagementActive = true
}

func disableRemoteManagementDestination() {
	if remoteManagementDest == nil || !remoteManagementActive {
		return
	}
	remoteManagementDest.DeregisterRequestHandler("/status")
	remoteManagementDest.DeregisterRequestHandler("/path")
	unregisterControlHash(remoteManagementDest.Hash())
	remoteManagementActive = false
}

func remoteStatusHandler(_ string, data any, _ []byte, _ []byte, remoteIdentity *Identity, _ time.Time) any {
	if remoteIdentity == nil || Owner == nil {
		return nil
	}
	args, ok := toInterfaceSlice(data)
	if !ok || len(args) == 0 {
		return nil
	}
	includeLinks, _ := asBoolValue(args[0])
	response := []any{Owner.GetInterfaceStats()}
	if includeLinks {
		response = append(response, Owner.GetLinkCount())
	}
	return response
}

func remotePathHandler(_ string, data any, _ []byte, _ []byte, remoteIdentity *Identity, _ time.Time) any {
	if remoteIdentity == nil || Owner == nil {
		return nil
	}
	args, ok := toInterfaceSlice(data)
	if !ok || len(args) == 0 {
		return nil
	}
	command, ok := asStringValue(args[0])
	if !ok {
		return nil
	}
	command = strings.ToLower(command)

	var destHash []byte
	if len(args) > 1 {
		if hash, ok := asBytesValue(args[1]); ok {
			destHash = hash
		}
	}

	maxHops := -1
	if len(args) > 2 {
		if mh, ok := asIntValue(args[2]); ok {
			maxHops = mh
		}
	}

	switch command {
	case "table":
		table := Owner.GetPathTable(maxHops)
		return filterTableByHash(table, destHash)
	case "rates":
		table := Owner.GetRateTable()
		return filterTableByHash(table, destHash)
	case "drop_announces", "drop_queues":
		// Mirrors rnpath --drop-announces, but executed on the remote instance.
		return Owner.DropAnnounceQueues()
	case "drop":
		// Mirrors rnpath --drop, but executed on the remote instance.
		if len(destHash) == 0 {
			return nil
		}
		return Owner.DropPath(destHash)
	case "drop_via":
		// Mirrors rnpath --drop-via, but executed on the remote instance.
		if len(destHash) == 0 {
			return nil
		}
		return Owner.DropAllVia(destHash)
	case "discover", "request_path":
		// Requests a path on the remote instance.
		if len(destHash) == 0 {
			return nil
		}
		TransportRequestPath(destHash)
		return true
	default:
		return nil
	}
}

func pathRequestHandler(data []byte, packet *Packet) {
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
		Logf(LogDebug, "Ignoring tagless path request for %s", PrettyHash(destinationHash))
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
		Logf(LogDebug, "Ignoring duplicate path request for %s with tag %s", PrettyHash(destinationHash), PrettyHash(unique))
		return
	}
	discoveryPRTags[uniqueKey] = struct{}{}
	discoveryPRTagFIFO = append(discoveryPRTagFIFO, uniqueKey)
	if len(discoveryPRTagFIFO) > maxPRTags {
		evict := discoveryPRTagFIFO[0]
		discoveryPRTagFIFO = discoveryPRTagFIFO[1:]
		delete(discoveryPRTags, evict)
	}
	discoveryTagsMu.Unlock()

	var attached *Interface
	if packet != nil {
		attached = packet.ReceivingInterface
	}

	// Python parity: If the destination exists on a local client, but it has not been
	// announced yet, the shared instance should still forward PATH_REQUEST packets to
	// local clients so they can answer with a PATH_RESPONSE.
	if attached != nil && len(LocalClientInterfaces) > 0 && !IsLocalClientInterface(attached) {
		if key, ok := makeHashKey(destinationHash); ok {
			pendingLocalPathRequestsMu.Lock()
			if _, exists := pendingLocalPathRequests[key]; !exists {
				pendingLocalPathRequests[key] = attached
			}
			pendingLocalPathRequestsMu.Unlock()
		}
		if packet != nil && len(packet.Raw) > 0 {
			forwardPathRequestToLocalClients(packet.Raw)
		}
	}

	if answerPathRequest(destinationHash, attached, requestingTransportID, tagBytes) {
		return
	}

	// Forward the request while preserving the tag to avoid loops (Python behaviour).
	if attached == nil {
		requestPathOnInterface(destinationHash, nil, tagBytes, true)
		return
	}
	for _, ifc := range Interfaces {
		if ifc == nil || ifc == attached {
			continue
		}
		requestPathOnInterface(destinationHash, ifc, tagBytes, true)
	}
}

func forwardPathRequestToLocalClients(raw []byte) {
	if len(raw) == 0 || len(LocalClientInterfaces) == 0 {
		return
	}
	send := append([]byte(nil), raw...)
	for _, cif := range LocalClientInterfaces {
		if cif == nil {
			continue
		}
		Transmit(cif, send)
	}
}

func tunnelSynthesizeHandler(data []byte, packet *Packet) {
	// Python expected_length:
	// KEYSIZE//8 (64) + HASHLENGTH//8 (32) + TRUNCATED_HASHLENGTH//8 (16) + SIGLENGTH//8 (64) = 176
	expected := (IdentityKeySize / 8) + (IdentityHashLength / 8) + (ReticulumTruncatedHashLength / 8) + (IdentitySigLength / 8)
	if len(data) != expected {
		return
	}
	if packet == nil || packet.ReceivingInterface == nil {
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
		return
	}
	if !remote.Validate(signature, signedData) {
		return
	}

	// Python parity: if a tunnel with same ID already exists, void the old interface.
	tunnelsMu.Lock()
	if existing := tunnels[string(tunnelID)]; existing != nil && existing.Interface != nil && existing.Interface != packet.ReceivingInterface {
		Logf(LogExtreme, "Voiding tunnel interface %v", existing.Interface)
		existing.Interface.TunnelID = nil
		existing.Interface = nil
		tunnels[string(tunnelID)] = existing
	}
	tunnelsMu.Unlock()

	handleTunnel(tunnelID, packet.ReceivingInterface)
}

// SynthesizeTunnel mirrors Python Transport.synthesize_tunnel(interface). It sends a
// tunnel establishment packet attached to the specified interface.
func SynthesizeTunnel(ifc *Interface) {
	if ifc == nil || TransportIdentity == nil {
		return
	}
	pub := TransportIdentity.GetPublicKey()
	if len(pub) != IdentityKeySize/8 {
		return
	}
	ifaceHash := ifc.Hash()
	if len(ifaceHash) != IdentityHashLength/8 {
		return
	}
	randomHash := IdentityGetRandomHash()
	if len(randomHash) != ReticulumTruncatedHashLength/8 {
		return
	}

	tunnelIDData := make([]byte, 0, len(pub)+len(ifaceHash))
	tunnelIDData = append(tunnelIDData, pub...)
	tunnelIDData = append(tunnelIDData, ifaceHash...)

	signedData := make([]byte, 0, len(tunnelIDData)+len(randomHash))
	signedData = append(signedData, tunnelIDData...)
	signedData = append(signedData, randomHash...)

	sig, err := TransportIdentity.Sign(signedData)
	if err != nil || len(sig) != IdentitySigLength/8 {
		return
	}

	data := make([]byte, 0, len(signedData)+len(sig))
	data = append(data, signedData...)
	data = append(data, sig...)

	dst, err := NewDestination(nil, DestinationOUT, DestinationPLAIN, "rnstransport", "tunnel", "synthesize")
	if err != nil {
		return
	}
	p := NewPacket(
		dst,
		data,
		WithPacketType(PacketTypeData),
		WithTransportType(Broadcast),
		WithHeaderType(HeaderType1),
		WithAttachedInterface(ifc),
		WithoutReceipt(),
	)
	if p == nil {
		return
	}
	_ = p.Send()
}

func handleTunnel(tunnelID []byte, ifc *Interface) {
	if len(tunnelID) != HashLengthBytes || ifc == nil {
		return
	}
	expiresAt := time.Now().Add(pathExpiration)

	key := string(tunnelID)
	tunnelsMu.Lock()
	existing := tunnels[key]
	if existing != nil {
		Logf(LogDebug, "Tunnel endpoint %s reappeared. Restoring paths...", PrettyHexRep(tunnelID))
		existing.Interface = ifc
		existing.ExpiresAt = expiresAt
		if existing.Paths == nil {
			existing.Paths = make(map[string]*tunnelPathEntry)
		}
		tunnels[key] = existing
		tunnelsMu.Unlock()

		ifc.TunnelID = append([]byte(nil), tunnelID...)
		restoreTunnelPaths(existing, ifc)
		return
	}
	te := &tunnelEntry{
		ID:        append([]byte(nil), tunnelID...),
		Interface: ifc,
		ExpiresAt: expiresAt,
		Paths:     make(map[string]*tunnelPathEntry),
	}
	tunnels[key] = te
	tunnelsMu.Unlock()
	Logf(LogDebug, "Tunnel endpoint %s established.", PrettyHexRep(tunnelID))
	ifc.TunnelID = append([]byte(nil), tunnelID...)
}

func restoreTunnelPaths(te *tunnelEntry, ifc *Interface) {
	if te == nil || ifc == nil || len(te.Paths) == 0 {
		return
	}
	now := time.Now()
	deprecated := make([]string, 0)

	for dstKey, entry := range te.Paths {
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

		pathKey, ok := makeHashKey(dstHash)
		if !ok {
			continue
		}
		newEntry := &PathEntry{
			NextHop:       copyBytes(entry.ReceivedFrom),
			RecvInterface: ifc,
			Hops:          entry.Hops,
			Timestamp:     now,
			ExpiresAt:     entry.ExpiresAt,
			RandomBlobs:   copyRandomBlobSlice(entry.RandomBlobs),
			PacketHash:    copyBytes(entry.PacketHash),
		}

		shouldAdd := false
		old := getPathEntry(dstHash)
		if old != nil {
			if newEntry.Hops <= old.Hops || now.After(old.ExpiresAt) {
				shouldAdd = true
			} else {
				Logf(LogDebug, "Did not restore path to %s because a newer path with fewer hops exist", PrettyHexRep(dstHash))
			}
		} else {
			shouldAdd = true
		}

		if shouldAdd {
			pathTableMu.Lock()
			pathTable[pathKey] = newEntry
			pathTableMu.Unlock()
			Logf(LogDebug, "Restored path to %s is now %d hops away via %s on %v",
				PrettyHexRep(dstHash), newEntry.Hops, PrettyHexRep(newEntry.NextHop), ifc)
		} else {
			deprecated = append(deprecated, dstKey)
		}
	}

	if len(deprecated) > 0 {
		tunnelsMu.Lock()
		for _, k := range deprecated {
			delete(te.Paths, k)
		}
		tunnelsMu.Unlock()
	}
}

func answerPathRequest(destinationHash []byte, attachedInterface *Interface, requestorTransportID []byte, tag []byte) bool {
	// Local destination? Answer by emitting a path response announce.
	if dst := destinationByHash(destinationHash); dst != nil {
		dst.Announce(nil, true, attachedInterface, tag, true)
		return true
	}

	// Known path? Re-announce cached announce as path response.
	if !TransportEnabled() {
		return false
	}
	entry := getPathEntry(destinationHash)
	if entry == nil {
		return false
	}

	// Python parity: If the destination exists on a local client interface, remember which
	// external interface desires the path so a subsequent PATH_RESPONSE from the local client
	// can be forwarded.
	if attachedInterface != nil && len(LocalClientInterfaces) > 0 && IsLocalClientInterface(entry.RecvInterface) {
		if key, ok := makeHashKey(destinationHash); ok {
			pendingLocalPathRequestsMu.Lock()
			pendingLocalPathRequests[key] = attachedInterface
			pendingLocalPathRequestsMu.Unlock()
		}
	}

	// Don't answer path requests on roaming interfaces if next hop is on the same roaming interface.
	if attachedInterface != nil && attachedInterface.Mode == InterfaceModeRoaming && entry.RecvInterface == attachedInterface {
		return true
	}

	// If requestor transport id is the next hop, skip (Python behaviour).
	if len(requestorTransportID) == truncatedHashBytes && len(entry.NextHop) == truncatedHashBytes && bytes.Equal(entry.NextHop, requestorTransportID) {
		return true
	}

	raw := getCachedAnnounceRaw(entry.PacketHash)
	if len(raw) == 0 {
		return false
	}
	cached := NewPacket(nil, raw)
	if cached == nil || !cached.Unpack() {
		return false
	}

	dest := &Destination{
		Type:      DestinationSINGLE,
		Direction: DestinationOUT,
		hash:      copyBytes(destinationHash),
		hexhash:   PrettyHexRep(destinationHash),
	}

	resp := NewPacket(
		dest,
		copyBytes(cached.Data),
		WithPacketType(PacketANNOUNCE),
		WithPacketContext(PacketPATH_RESPONSE),
		WithTransportType(TransportDirect),
		WithHeaderType(HeaderType2),
		WithTransportID(copyBytes(TransportIdentity.Hash)),
		WithContextFlag(cached.ContextFlag),
		WithAttachedInterface(attachedInterface),
		WithoutReceipt(),
	)
	if resp == nil {
		return false
	}
	h := entry.Hops
	if h < 0 {
		h = 0
	}
	if h > 255 {
		h = 255
	}
	resp.Hops = uint8(h)
	resp.DestinationHash = copyBytes(destinationHash)
	resp.DestinationType = byte(DestinationSINGLE)

	delay := pathRequestGrace
	if attachedInterface != nil && attachedInterface.Mode == InterfaceModeRoaming {
		delay += pathRequestRoamingGrace
	}
	QueueAnnounce(resp, WithAnnounceDelay(delay), WithAnnounceBlockRebroadcasts(true), WithAnnounceAttachedInterface(attachedInterface))
	return true
}

func getCachedAnnounceRaw(packetHash []byte) []byte {
	if len(packetHash) == 0 {
		return nil
	}
	path := announceCachePath(packetHash)
	if path != "" && fileExists(path) {
		if b, err := os.ReadFile(path); err == nil && len(b) > 0 {
			return b
		}
	}
	packetCacheMu.RLock()
	entry := packetCache[string(packetHash)]
	packetCacheMu.RUnlock()
	if entry != nil && len(entry.Raw) > 0 {
		return append([]byte(nil), entry.Raw...)
	}
	return nil
}

func filterTableByHash(table []map[string]any, hash []byte) []map[string]any {
	if len(hash) == 0 {
		return table
	}
	filtered := make([]map[string]any, 0, len(table))
	for _, entry := range table {
		raw, _ := entry["hash"].([]byte)
		if len(raw) > 0 && bytes.Equal(raw, hash) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func toInterfaceSlice(value any) ([]any, bool) {
	switch v := value.(type) {
	case []any:
		return v, true
	default:
		return nil, false
	}
}

func asBoolValue(value any) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case *bool:
		if v == nil {
			return false, false
		}
		return *v, true
	default:
		return false, false
	}
}

func asBytesValue(value any) ([]byte, bool) {
	switch v := value.(type) {
	case []byte:
		return v, true
	case *[]byte:
		if v == nil {
			return nil, false
		}
		return *v, true
	case string:
		return []byte(v), true
	default:
		return nil, false
	}
}

func asStringValue(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case []byte:
		return string(v), true
	default:
		return "", false
	}
}

func asIntValue(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int8:
		return int(v), true
	case int16:
		return int(v), true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case uint:
		return int(v), true
	case uint8:
		return int(v), true
	case uint16:
		return int(v), true
	case uint32:
		return int(v), true
	case uint64:
		return int(v), true
	case float32:
		return int(v), true
	case float64:
		return int(v), true
	case *int:
		if v == nil {
			return 0, false
		}
		return *v, true
	case *float64:
		if v == nil {
			return 0, false
		}
		return int(*v), true
	default:
		return 0, false
	}
}

func asFloatValue(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint64:
		return float64(v), true
	case *float64:
		if v == nil {
			return 0, false
		}
		return *v, true
	default:
		return 0, false
	}
}

func registerControlHash(hash []byte) {
	if len(hash) == 0 {
		return
	}
	controlHashesMu.Lock()
	controlHashes[string(hash)] = struct{}{}
	controlHashesMu.Unlock()
}

func unregisterControlHash(hash []byte) {
	if len(hash) == 0 {
		return
	}
	controlHashesMu.Lock()
	delete(controlHashes, string(hash))
	controlHashesMu.Unlock()
}

func cullReverseAndLinkTables(now time.Time) bool {
	var removed bool

	reverseTableMu.Lock()
	for key, entry := range reverseTable {
		if entry == nil {
			delete(reverseTable, key)
			removed = true
			continue
		}
		if now.Sub(entry.Timestamp) > reverseTimeout {
			delete(reverseTable, key)
			removed = true
			continue
		}
		// Python removes entries if interfaces disappear.
		if entry.OutboundIf != nil && !interfacePresent(entry.OutboundIf) {
			delete(reverseTable, key)
			removed = true
			continue
		}
		if entry.ReceivedIf != nil && !interfacePresent(entry.ReceivedIf) {
			delete(reverseTable, key)
			removed = true
		}
	}
	reverseTableMu.Unlock()

	linkMu.Lock()
	for key, entry := range linkTable {
		if entry == nil {
			delete(linkTable, key)
			removed = true
			continue
		}
		if entry.NextHopInterface != nil && !interfacePresent(entry.NextHopInterface) {
			delete(linkTable, key)
			removed = true
			continue
		}
		if entry.ReceivedInterface != nil && !interfacePresent(entry.ReceivedInterface) {
			delete(linkTable, key)
			removed = true
			continue
		}
		// Validated links have a timeout separate from path table expiration.
		if entry.Validated && !entry.Timestamp.IsZero() && now.Sub(entry.Timestamp) > linkTimeout {
			delete(linkTable, key)
			removed = true
			continue
		}
		// Proof timeouts expire pending links.
		if !entry.ProofTimeout.IsZero() && now.After(entry.ProofTimeout) {
			delete(linkTable, key)
			removed = true
			continue
		}
		if !entry.Timestamp.IsZero() && now.Sub(entry.Timestamp) > pathExpiration {
			delete(linkTable, key)
			removed = true
		}
	}
	linkMu.Unlock()

	return removed
}

func cullPathTable(now time.Time) bool {
	pathTableMu.Lock()
	defer pathTableMu.Unlock()
	removed := false
	for key, entry := range pathTable {
		if entry == nil {
			delete(pathTable, key)
			removed = true
			continue
		}
		if !entry.ExpiresAt.IsZero() && entry.ExpiresAt.Before(now) {
			delete(pathTable, key)
			removed = true
			continue
		}
		if !entry.Timestamp.IsZero() && now.Sub(entry.Timestamp) > pathExpiration {
			delete(pathTable, key)
			removed = true
		}
	}
	return removed
}

func cullPathStates() bool {
	pathTableMu.RLock()
	defer pathTableMu.RUnlock()
	pathStatesMu.Lock()
	defer pathStatesMu.Unlock()
	removed := false
	for key := range pathStates {
		if _, ok := pathTable[key]; ok {
			continue
		}
		delete(pathStates, key)
		removed = true
	}
	return removed
}

func interfacePresent(ifc *Interface) bool {
	if ifc == nil {
		return false
	}
	for _, existing := range Interfaces {
		if existing == ifc {
			return true
		}
	}
	return false
}

func cleanPacketCache(now time.Time) bool {
	packetCacheMu.Lock()
	removed := false
	for key, entry := range packetCache {
		if entry == nil {
			delete(packetCache, key)
			removed = true
			continue
		}
		if now.Sub(entry.StoredAt) > packetCacheLifetime {
			delete(packetCache, key)
			removed = true
		}
	}
	packetCacheMu.Unlock()
	return removed
}

func cleanAnnounceCache(now time.Time) bool {
	if Owner == nil || len(Owner.CachePath) == 0 {
		return false
	}
	announceDir := filepath.Join(Owner.CachePath, "announces")
	if !fileExists(announceDir) {
		return false
	}
	removed := false
	_ = filepath.WalkDir(announceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, statErr := d.Info()
		if statErr != nil {
			return nil
		}
		if now.Sub(info.ModTime()) > packetCacheLifetime {
			if rmErr := os.Remove(path); rmErr == nil {
				removed = true
			}
		}
		return nil
	})
	return removed
}

// -------- interface transmission (IFAC) --------

func Transmit(ifc *Interface, raw []byte) {
	defer func() {
		if r := recover(); r != nil {
			Logf(LogError, "Error while transmitting on %v: %v", ifc, r)
		}
	}()

	if ifc.IFACIdentity != nil {
		// ifac = Sign(raw)[-ifacSize:]
		sig, err := ifc.IFACIdentity.Sign(raw)
		if err != nil {
			return
		}
		if len(sig) < ifc.IFACSize {
			return
		}
		ifac := sig[len(sig)-ifc.IFACSize:]

		// mask = hkdf(len(raw)+ifacSize, ifac, ifacKey)
		mask, err := Cryptography.HKDF(len(raw)+ifc.IFACSize, ifac, ifc.IFACKey, nil)
		if err != nil {
			return
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

func transmitAnnounce(ifc *Interface, raw []byte, hops uint8) {
	if ifc == nil || len(raw) == 0 {
		return
	}
	// IFAC is applied inside Transmit(), so enqueue raw through Transmit path first
	// when interface uses IFAC. For announce-cap throttling we need the final raw.
	if ifc.IFACIdentity != nil {
		Transmit(ifc, raw)
		return
	}
	ifc.ProcessAnnounceRaw(raw, int(hops))
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

	outTime := time.Now()

	genReceipt := false
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
			Receipts = append(Receipts, rc)
		}
	}

	// is the path known?
	sendBroadcast := true
	if p.Type != PacketAnnounce &&
		p.Destination.Type != DestPlain &&
		p.Destination.Type != DestGroup &&
		HasPath(p.DestinationHash) {

		entry := getPathEntry(p.DestinationHash)
		if entry != nil {
			outIfc := entry.RecvInterface
			hops := entry.Hops

			connectedShared := Owner != nil && Owner.IsConnectedToSharedInstance
			// When the path has more than one hop, inject the packet into transport
			// by rewriting to HEADER_2 with the next-hop transport ID.
			// When connected to a shared instance, we also rewrite single-hop
			// packets to HEADER_2 because rnsd does not forward plain HEADER_1
			// packets from local clients to remote destinations.
			if hops > 1 || connectedShared {
				// add transport header (HEADER_2 + next hop id)
				Logf(LogNotice, "Outbound: path-based HEADER_2 rewrite (hops=%d, shared=%v, nextHop=%x, dest=%x, ifc=%s)",
					hops, connectedShared, entry.NextHop, p.DestinationHash, outIfc)
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
					Transmit(outIfc, newRaw)
					entry.Timestamp = time.Now()
					sent = true
					sendBroadcast = false
				}
			} else {
				// single hop: send directly
				Logf(LogNotice, "Outbound: direct HEADER_1 send (hops=%d, ifc=%s)", hops, outIfc)
				packetSent(p)
				Transmit(outIfc, p.Raw)
				sent = true
				sendBroadcast = false
			}
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

			if p.AttachedInterface != nil && ifc != p.AttachedInterface {
				shouldSend = false
			}

			// announce logic with AP/ROAMING/BOUNDARY and queues
			if p.Type == PacketAnnounce {
				if !shouldAnnounceOnInterface(p, ifc, outTime) {
					shouldSend = false
				}
			}

			if !shouldSend {
				continue
			}

			if !storedHash {
				AddPacketHash(p.PacketHash)
				storedHash = true
			}

			if p.Type == PacketAnnounce {
				transmitAnnounce(ifc, p.Raw, p.Hops)
			} else {
				Transmit(ifc, p.Raw)
			}
			packetSent(p)
			sent = true
		}
	}

	if sent && p.Type == PacketAnnounce && p.Context != PacketPATH_RESPONSE {
		QueueAnnounce(p, WithAnnounceAttachedInterface(p.AttachedInterface))
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
			Logf(LogExtreme, "Ignoring packet %s for other transport instance",
				PrettyHash(p.PacketHash))
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
				Logf(LogDebug, "Dropped PLAIN packet %s with %d hops",
					PrettyHash(p.PacketHash), p.Hops)
				return false
			}
			return true
		}
		Log("Dropped invalid PLAIN announce", LogDebug)
		return false
	}

	if p.DestinationType == DestGroup {
		if p.Type != PacketAnnounce {
			if p.Hops > 1 {
				Logf(LogDebug, "Dropped GROUP packet %s with %d hops",
					PrettyHash(p.PacketHash), p.Hops)
				return false
			}
			return true
		}
		Log("Dropped invalid GROUP announce", LogDebug)
		return false
	}

	if !HasPacketHash(p.PacketHash) {
		return true
	}

	if p.Type == PacketAnnounce && p.DestinationType == DestSingle {
		return true
	}

	Logf(LogExtreme, "Filtered packet %s", PrettyHash(p.PacketHash))
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

	p := NewPacket(nil, raw)
	if !p.Unpack() {
		return
	}

	p.ReceivingInterface = ifc
	p.Hops++

	// Record reverse table entry for transport-direct packets (minimal parity).
	if p.HeaderType == HeaderType2 && ifc != nil {
		// Python stores reverse_table entries keyed by packet.getTruncatedHash().
		// These are used for forwarding proofs and replies back towards the origin.
		if key, ok := makeHashKey(p.GetTruncatedHash()); ok {
			reverseTableMu.Lock()
			reverseTable[key] = &reverseEntry{ReceivedIf: ifc, Timestamp: time.Now()}
			reverseTableMu.Unlock()
		}
	}

	// cache RSSI/SNR/Q for clients
	cacheLocalStats(p, ifc)

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
	if key, ok := makeHashKey(p.DestinationHash); ok {
		if _, exists := linkTable[key]; exists {
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

	if p.Context == PacketCacheRequest {
		if cacheRequestPacket(p) {
			return
		}
	}

	fromLocal := IsLocalClientInterface(ifc)
	forLocalClient := isForLocalClient(p)
	proofForLocalClient := isProofForLocal(p)

	if p.TransportID == nil && forLocalClient && p.Type != PacketAnnounce && TransportIdentity != nil {
		p.TransportID = copyBytes(TransportIdentity.Hash)
	}

	if p.TransportID != nil && p.Type != PacketAnnounce && TransportIdentity != nil && bytes.Equal(p.TransportID, TransportIdentity.Hash) {
		if forwardDesignatedTransportPacket(p, ifc) {
			return
		}
	}

	// Local client routing: if a destination is behind a local client (path hops==0),
	// forward packets to the local client interface.
	if forLocalClient && !fromLocal {
		if entry := getPathEntry(p.DestinationHash); entry != nil && entry.RecvInterface != nil && IsLocalClientInterface(entry.RecvInterface) {
			// Python parity: when forwarding LINKREQUEST packets to a local client, we
			// must create a link_table entry so that subsequent LINK/LRPROOF packets
			// can be routed back to the original interface.
			if p.PacketType == PacketTypeLinkRequest {
				linkID := linkIDFromLinkRequestPacket(p)
				if lidKey, ok := makeHashKey(linkID); ok {
					now := time.Now()
					proofTimeout := now.Add(time.Duration(DEFAULT_PER_HOP_TIMEOUT*maxInt(1, entry.Hops))*time.Second + TransportExtraLinkProofTimeout(entry.RecvInterface))
					le := &linkEntry{
						Timestamp:         now,
						NextHopID:         copyBytes(entry.NextHop),
						NextHopInterface:  entry.RecvInterface,
						RemainingHops:     entry.Hops,
						ReceivedInterface: ifc,
						Hops:              int(p.Hops),
						DestinationHash:   copyBytes(p.DestinationHash),
						Validated:         false,
						ProofTimeout:      proofTimeout,
					}
					linkMu.Lock()
					linkTable[lidKey] = le
					linkMu.Unlock()
				}
			}
			Transmit(entry.RecvInterface, p.Raw)
			return
		}
	}

	// Reverse table forwarding (Python: return proofs and replies).
	if p.Type != PacketAnnounce && p.DestinationType == DestSingle {
		if forwardViaReverseTable(p, ifc) {
			return
		}
	}

	// PLAIN BROADCAST from a client -> forward across interfaces
	if !isControlHash(p.DestinationHash) &&
		p.DestinationType == DestPlain &&
		p.TransportType == Broadcast {

		if fromLocal {
			for _, oif := range Interfaces {
				if oif != ifc {
					Transmit(oif, p.Raw)
				}
			}
		} else {
			for _, cif := range LocalClientInterfaces {
				Transmit(cif, p.Raw)
			}
		}
	}

	if p.Type == PacketAnnounce {
		handleInboundAnnounce(p, ifc, fromLocal)
	}

	// Link transport handling (Python: routes packets according to link_table).
	// This must include LRPROOF packets so link establishment can complete across hops.
	if p.Type != PacketAnnounce && p.Type != PacketLINKREQUEST {
		// If this link packet is for a local link, deliver it before any routing
		// decisions. Shared instances will typically have no local link, and will
		// fall back to link_table forwarding below.
		if p.DestinationType == DestLink {
			link := findLinkByID(p.DestinationHash)
			if link != nil {
				link.Receive(p)
				return
			}
			Logf(LogDebug, "Inbound DestLink: no local link for %x ctx=0x%02x (pending=%d active=%d)",
				p.DestinationHash, p.Context, len(PendingLinks), len(ActiveLinks))
			for i, pl := range PendingLinks {
				if pl != nil {
					Logf(LogExtreme, "  PendingLink[%d]: linkID=%x status=%d", i, pl.LinkID, pl.Status)
				}
			}
		}

		if forwardViaLinkTable(p, ifc) {
			return
		}
	}

	// ---- basic delivery pipeline (Destination/Link/Proof) ----

	// ProofDestination packets are addressed to truncated packet hashes and are not registered Destinations.
	if p.Type == PacketProof && p.DestinationType == DestSingle {
		if handleInboundProof(p) {
			return
		}
		// If this proof is not for us, try to forward via reverse table (Python behaviour).
		if forwardProofViaReverseTable(p, ifc) {
			return
		}
		if proofForLocalClient && len(LocalClientInterfaces) > 0 {
			for _, cif := range LocalClientInterfaces {
				if cif != nil && cif != ifc {
					Transmit(cif, p.Raw)
				}
			}
			return
		}
	}

	// Deliver to link if addressed to a link ID.
	if p.DestinationType == DestLink {
		if link := findLinkByID(p.DestinationHash); link != nil {
			link.Receive(p)
			return
		}
		// Not a local link; allow routing logic below to handle it.
	}

	// Deliver to a registered destination (including control destinations).
	if dst := destinationByHash(p.DestinationHash); dst != nil {
		_ = dst.Receive(p)
		return
	}

	// Minimal routing pipeline: forward packets not for local system along a known path.
	if TransportEnabled() {
		if forwardTransportPacket(p, ifc) {
			return
		}
	}
}

func forwardViaLinkTable(p *Packet, receivedOn *Interface) bool {
	if p == nil || len(p.DestinationHash) == 0 {
		return false
	}
	key, ok := makeHashKey(p.DestinationHash)
	if !ok {
		return false
	}

	linkMu.Lock()
	entry := linkTable[key]
	linkMu.Unlock()
	if entry == nil || entry.NextHopInterface == nil || entry.ReceivedInterface == nil {
		return false
	}

	var outbound *Interface
	if entry.NextHopInterface == entry.ReceivedInterface {
		// Direction doesn't matter, but ensure hop count matches expectations.
		if int(p.Hops) == entry.RemainingHops || int(p.Hops) == entry.Hops {
			outbound = entry.NextHopInterface
		}
	} else {
		// Direction matters; transmit on opposite interface to where it was received.
		if receivedOn == entry.NextHopInterface {
			if int(p.Hops) == entry.RemainingHops {
				outbound = entry.ReceivedInterface
			}
		} else if receivedOn == entry.ReceivedInterface {
			if int(p.Hops) == entry.Hops {
				outbound = entry.NextHopInterface
			}
		}
	}

	if outbound == nil || outbound == receivedOn {
		return false
	}

	// Add this packet to the filter hashlist if we have determined that it's our turn.
	AddPacketHash(p.PacketHash)

	outRaw := append([]byte{p.Raw[0], p.Raw[1]}, p.Raw[2:]...)
	Transmit(outbound, outRaw)

	now := time.Now()
	linkMu.Lock()
	if cur := linkTable[key]; cur == entry {
		entry.Timestamp = now
		linkTable[key] = entry
	}
	linkMu.Unlock()
	return true
}

func forwardDesignatedTransportPacket(p *Packet, receivedOn *Interface) bool {
	if p == nil || p.Type == PacketAnnounce {
		return false
	}
	// Only transport packets (header2) are routed here.
	if p.HeaderType != HeaderType2 || len(p.TransportID) != truncatedHashBytes {
		return false
	}

	entry := getPathEntry(p.DestinationHash)
	if entry == nil || entry.RecvInterface == nil {
		Logf(LogExtreme, "Got packet in transport, but no known path to final destination %s. Dropping packet.", PrettyHexRep(p.DestinationHash))
		return true
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
		return false
	}

	// LINKREQUEST: create link table entry, else create reverse table entry.
	if p.PacketType == PacketTypeLinkRequest {
		linkID := linkIDFromLinkRequestPacket(p)
		if lidKey, ok := makeHashKey(linkID); ok {
			now := time.Now()
			proofTimeout := now.Add(time.Duration(DEFAULT_PER_HOP_TIMEOUT*maxInt(1, remainingHops))*time.Second + TransportExtraLinkProofTimeout(entry.RecvInterface))
			le := &linkEntry{
				Timestamp:         now,
				NextHopID:         copyBytes(nextHop),
				NextHopInterface:  entry.RecvInterface,
				RemainingHops:     remainingHops,
				ReceivedInterface: receivedOn,
				Hops:              int(p.Hops),
				DestinationHash:   copyBytes(p.DestinationHash),
				Validated:         false,
				ProofTimeout:      proofTimeout,
			}
			linkMu.Lock()
			linkTable[lidKey] = le
			linkMu.Unlock()
		}
	} else {
		if key, ok := makeHashKey(p.GetTruncatedHash()); ok {
			reverseTableMu.Lock()
			reverseTable[key] = &reverseEntry{ReceivedIf: receivedOn, OutboundIf: entry.RecvInterface, Timestamp: time.Now()}
			reverseTableMu.Unlock()
		}
	}

	Transmit(entry.RecvInterface, outRaw)

	// Update timestamp to keep path alive (Python).
	pathTableMu.Lock()
	if k, ok := makeHashKey(p.DestinationHash); ok {
		if cur := pathTable[k]; cur != nil {
			cur.Timestamp = time.Now()
			pathTable[k] = cur
		}
	}
	pathTableMu.Unlock()

	return true
}

func forwardTransportPacket(p *Packet, receivedOn *Interface) bool {
	if p == nil {
		return false
	}
	// Do not forward announces; they are handled via announce queue.
	if p.Type == PacketAnnounce {
		return false
	}
	// Do not forward broadcast plain/group packets here; those are handled separately.
	if p.TransportType == Broadcast && (p.DestinationType == DestPlain || p.DestinationType == DestGroup) {
		return false
	}
	// Only forward addressable destination types.
	if p.DestinationType != DestSingle && p.DestinationType != DestGroup && p.DestinationType != DestPlain {
		return false
	}
	if p.Hops >= PathfinderMaxHops+1 {
		return false
	}

	entry := getPathEntry(p.DestinationHash)
	if entry == nil || entry.RecvInterface == nil {
		return false
	}
	if receivedOn != nil && entry.RecvInterface == receivedOn {
		return false
	}

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
	if key, ok := makeHashKey(p.GetTruncatedHash()); ok && receivedOn != nil {
		reverseTableMu.Lock()
		if rev := reverseTable[key]; rev != nil {
			rev.OutboundIf = entry.RecvInterface
			reverseTable[key] = rev
		} else {
			reverseTable[key] = &reverseEntry{ReceivedIf: receivedOn, OutboundIf: entry.RecvInterface, Timestamp: time.Now()}
		}
		reverseTableMu.Unlock()
	}

	Transmit(entry.RecvInterface, outRaw)

	// Create link table entry for link requests so subsequent link packets can be forwarded.
	if p.PacketType == PacketTypeLinkRequest {
		linkID := linkIDFromLinkRequestPacket(p)
		if lidKey, ok := makeHashKey(linkID); ok {
			proofTmo := time.Now().Add(time.Duration(DEFAULT_PER_HOP_TIMEOUT)*time.Second*time.Duration(maxInt(1, remainingHops)) + TransportExtraLinkProofTimeout(entry.RecvInterface))
			le := &linkEntry{
				Timestamp:         time.Now(),
				NextHopID:         copyBytes(entry.NextHop),
				NextHopInterface:  entry.RecvInterface,
				RemainingHops:     remainingHops,
				ReceivedInterface: receivedOn,
				Hops:              int(p.Hops),
				DestinationHash:   copyBytes(p.DestinationHash),
				Validated:         false,
				ProofTimeout:      proofTmo,
			}
			linkMu.Lock()
			linkTable[lidKey] = le
			linkMu.Unlock()
		}
	}
	return true
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// linkIDFromLinkRequestPacket computes the link ID in the same way as Python
// Link.link_id_from_lr_packet(), ensuring that link-table routing works for
// LRPROOF and subsequent link packets across transport hops.
func linkIDFromLinkRequestPacket(p *Packet) []byte {
	if p == nil || p.PacketType != PacketTypeLinkRequest {
		return nil
	}
	// The link ID is the truncated hash of the packet hashable part, excluding
	// optional signalling bytes beyond Link.ECPUBSIZE (64 bytes).
	hashable := p.getHashablePart()
	if len(p.Data) > linkEcPubSize {
		diff := len(p.Data) - linkEcPubSize
		if diff > 0 && diff <= len(hashable) {
			hashable = hashable[:len(hashable)-diff]
		}
	}
	return TruncatedHash(hashable)
}

func forwardProofViaReverseTable(p *Packet, receivedOn *Interface) bool {
	if p == nil || p.Type != PacketProof || len(p.DestinationHash) == 0 {
		return false
	}
	key, ok := makeHashKey(p.DestinationHash)
	if !ok {
		return false
	}

	reverseTableMu.Lock()
	entry := reverseTable[key]
	// Proofs are one-shot routed in Python; pop the entry.
	delete(reverseTable, key)
	reverseTableMu.Unlock()
	if entry == nil || entry.ReceivedIf == nil {
		return false
	}
	if receivedOn != nil && entry.ReceivedIf == receivedOn {
		return false
	}
	Transmit(entry.ReceivedIf, p.Raw)
	return true
}

func forwardViaReverseTable(p *Packet, receivedOn *Interface) bool {
	if p == nil || len(p.DestinationHash) == 0 {
		return false
	}
	key, ok := makeHashKey(p.DestinationHash)
	if !ok {
		return false
	}

	reverseTableMu.Lock()
	entry := reverseTable[key]
	if entry != nil {
		entry.Timestamp = time.Now()
		reverseTable[key] = entry
	}
	reverseTableMu.Unlock()
	if entry == nil || entry.ReceivedIf == nil {
		return false
	}

	var outbound *Interface
	if entry.OutboundIf != nil {
		switch receivedOn {
		case entry.OutboundIf:
			outbound = entry.ReceivedIf
		case entry.ReceivedIf:
			outbound = entry.OutboundIf
		default:
			outbound = entry.ReceivedIf
		}
	} else {
		outbound = entry.ReceivedIf
	}

	if outbound == nil || outbound == receivedOn {
		return false
	}

	Transmit(outbound, p.Raw)
	return true
}

func findLinkByID(linkID []byte) *Link {
	if len(linkID) == 0 {
		return nil
	}
	linkMu.Lock()
	defer linkMu.Unlock()
	for _, l := range ActiveLinks {
		if l != nil && bytes.Equal(l.LinkID, linkID) {
			return l
		}
	}
	for _, l := range PendingLinks {
		if l != nil && bytes.Equal(l.LinkID, linkID) {
			return l
		}
	}
	return nil
}

func handleInboundProof(p *Packet) bool {
	if p == nil || len(p.DestinationHash) == 0 {
		return false
	}
	for _, rc := range Receipts {
		if rc == nil || rc.Status != ReceiptSent {
			continue
		}
		if bytes.Equal(rc.TruncatedHash, p.DestinationHash) {
			if rc.ValidateProofPacket(p) {
				return true
			}
		}
	}
	return false
}

func handleInboundAnnounce(p *Packet, ifc *Interface, fromLocal bool) {
	if p == nil {
		return
	}

	if ifc != nil && IdentityValidateAnnounce(p, true) {
		noteInterfaceAnnounce(ifc)
	}

	// Python parity: apply ingress limiting for unknown destinations.
	if ifc != nil && !HasPath(p.DestinationHash) {
		if ifc.ShouldIngressLimit() {
			ifc.HoldAnnounce(append([]byte(nil), p.Raw...), p.ReceivingInterface, copyBytes(p.DestinationHash), p.Hops)
			return
		}
	}

	if destinationByHash(p.DestinationHash) != nil {
		return
	}

	if !IdentityValidateAnnounce(p, false) {
		return
	}

	// Notify application-level announce handlers (Python parity).
	dispatchAnnounceHandlers(p)

	now := time.Now()
	recordAnnounceRebroadcast(p, now)

	if p.Hops >= PathfinderMaxHops+1 {
		return
	}

	updated := updatePathFromAnnounce(p, ifc, now)
	if updated {
		Logf(LogDebug, "Updated path to %s via %s (%d hops)",
			PrettyHexRep(p.DestinationHash), interfaceName(ifc), p.Hops)
	}

	if len(LocalClientInterfaces) > 0 {
		forwardAnnounceToLocalClients(p)
	}

	retransmitAllowed := TransportEnabled() || fromLocal
	if !retransmitAllowed {
		return
	}

	if p.Context == PacketPATH_RESPONSE && !fromLocal {
		return
	}

	key := announceHash(p)
	if key == nil {
		return
	}

	if shouldRateBlockAnnounce(*key, ifc, now, p.Context) {
		Logf(LogDebug, "Blocking rebroadcast of announce from %s due to rate limiting",
			PrettyHexRep(p.DestinationHash))
		return
	}

	opts := []AnnounceOption{}
	if fromLocal {
		opts = append(opts, WithAnnounceImmediate())
	}
	if p.Context == PacketPATH_RESPONSE {
		opts = append(opts, WithAnnounceBlockRebroadcasts(true))
		attachedIF := p.AttachedInterface
		if attachedIF == nil {
			attachedIF = ifc
		}
		// Python parity: For PATH_RESPONSE from local clients, prefer any pending external
		// interface that requested the path.
		if fromLocal {
			if key, ok := makeHashKey(p.DestinationHash); ok {
				pendingLocalPathRequestsMu.Lock()
				if desired, exists := pendingLocalPathRequests[key]; exists {
					delete(pendingLocalPathRequests, key)
					attachedIF = desired
				}
				pendingLocalPathRequestsMu.Unlock()
			}
		}
		if attachedIF != nil {
			opts = append(opts, WithAnnounceAttachedInterface(attachedIF))
		}
	}

	QueueAnnounce(p, opts...)
}

func destinationByHash(hash []byte) *Destination {
	for _, d := range Destinations {
		if d == nil || len(d.Hash()) == 0 {
			continue
		}
		if bytes.Equal(d.Hash(), hash) {
			return d
		}
	}
	return nil
}

func forwardAnnounceToLocalClients(p *Packet) {
	if p == nil || len(LocalClientInterfaces) == 0 {
		return
	}
	send := cloneAnnouncePacket(p)
	if send == nil {
		return
	}
	if err := send.Pack(); err != nil {
		return
	}
	raw := append([]byte(nil), send.Raw...)
	for _, cif := range LocalClientInterfaces {
		if cif == nil {
			continue
		}
		Transmit(cif, raw)
	}
}

func noteInterfaceAnnounce(ifc *Interface) {
	if ifc == nil {
		return
	}
	ifc.ReceivedAnnounce()
}

func recordAnnounceRebroadcast(p *Packet, now time.Time) {
	key := announceHash(p)
	if key == nil {
		return
	}
	announceMu.Lock()
	defer announceMu.Unlock()
	entry := announceTable[*key]
	if entry == nil || entry.Packet == nil {
		return
	}
	expected := entry.Packet.Hops
	if p.Hops-1 == expected {
		entry.LocalRebroadcasts++
		if entry.LocalRebroadcasts >= localRebroadcastsMax {
			delete(announceTable, *key)
			return
		}
		announceTable[*key] = entry
		return
	}
	if p.Hops-1 == expected+1 && entry.Retries > 0 && now.Before(entry.Next) {
		delete(announceTable, *key)
	}
}

func updatePathFromAnnounce(p *Packet, ifc *Interface, now time.Time) bool {
	if p == nil {
		return false
	}
	key, ok := makeHashKey(p.DestinationHash)
	if !ok {
		return false
	}

	randomBlob := announceRandomBlob(p)
	existing := getPathEntry(p.DestinationHash)

	var blobs [][]byte
	var prevEmitted uint64
	if existing != nil && len(existing.RandomBlobs) > 0 {
		blobs = copyRandomBlobSlice(existing.RandomBlobs)
		prevEmitted = existing.AnnounceAt
	}

	blobSeen := randomBlobSeen(blobs, randomBlob)
	emitted := timebaseFromRandomBlob(randomBlob)
	shouldAdd := false

	switch {
	case existing == nil:
		shouldAdd = true
	case int(p.Hops) <= existing.Hops:
		if !blobSeen && emitted > timebaseFromRandomBlobs(blobs) {
			shouldAdd = true
		}
	default:
		expired := now.After(existing.ExpiresAt)
		newer := emitted > prevEmitted
		if (expired || newer) && !blobSeen {
			shouldAdd = true
		}
	}

	if !shouldAdd {
		return false
	}

	blobs = appendRandomBlob(blobs, randomBlob)
	nextHop := p.TransportID
	if len(nextHop) == 0 {
		nextHop = p.DestinationHash
	}
	packetHash := p.PacketHash
	if len(packetHash) == 0 {
		packetHash = p.GetHash()
	}

	entry := &PathEntry{
		NextHop:       copyBytes(nextHop),
		RecvInterface: ifc,
		Hops:          int(p.Hops),
		Timestamp:     now,
		ExpiresAt:     pathExpiryFromInterface(ifc, now),
		RandomBlobs:   blobs,
		AnnounceAt:    emitted,
		PacketHash:    copyBytes(packetHash),
	}

	pathTableMu.Lock()
	pathTable[key] = entry
	pathTableMu.Unlock()
	_ = MarkPathResponsive(p.DestinationHash)

	if ifc != nil && len(ifc.TunnelID) == HashLengthBytes {
		tunnelsMu.Lock()
		if te := tunnels[string(ifc.TunnelID)]; te != nil {
			if te.Paths == nil {
				te.Paths = make(map[string]*tunnelPathEntry)
			}
			te.Paths[string(p.DestinationHash)] = &tunnelPathEntry{
				Timestamp:    entry.Timestamp,
				ReceivedFrom: copyBytes(entry.NextHop),
				Hops:         entry.Hops,
				ExpiresAt:    entry.ExpiresAt,
				RandomBlobs:  copyRandomBlobSlice(entry.RandomBlobs),
				PacketHash:   copyBytes(entry.PacketHash),
			}
			tunnels[string(ifc.TunnelID)] = te
		}
		tunnelsMu.Unlock()
	}

	Cache(p, false)
	return true
}

func pathExpiryFromInterface(ifc *Interface, now time.Time) time.Time {
	switch interfaceMode(ifc) {
	case InterfaceModeAccessPoint:
		return now.Add(apPathTime)
	case InterfaceModeRoaming:
		return now.Add(roamingPathTime)
	default:
		return now.Add(pathExpiration)
	}
}

func interfaceMode(ifc *Interface) int {
	if ifc == nil {
		return InterfaceModeFull
	}
	type modeGetter interface {
		Mode() int
	}
	if getter, ok := any(ifc).(modeGetter); ok {
		return getter.Mode()
	}
	type modeGetterAlt interface {
		GetMode() int
	}
	if getter, ok := any(ifc).(modeGetterAlt); ok {
		return getter.GetMode()
	}
	return InterfaceModeFull
}

func shouldRateBlockAnnounce(key hashKey, ifc *Interface, now time.Time, ctx byte) bool {
	if ctx == PacketPATH_RESPONSE {
		return false
	}
	target, grace, penalty, ok := getAnnounceRateParams(ifc)
	if !ok || target <= 0 {
		return false
	}

	announceRateMu.Lock()
	defer announceRateMu.Unlock()

	entry := announceRateTable[key]
	if entry == nil {
		entry = &announceRateEntry{
			Last:       now,
			Timestamps: []time.Time{now},
		}
		announceRateTable[key] = entry
		return false
	}

	entry.Timestamps = append(entry.Timestamps, now)
	if len(entry.Timestamps) > maxRateTimestamps {
		entry.Timestamps = entry.Timestamps[len(entry.Timestamps)-maxRateTimestamps:]
	}

	if now.After(entry.BlockedUntil) {
		if now.Sub(entry.Last) < target {
			entry.RateViolations++
		} else if entry.RateViolations > 0 {
			entry.RateViolations--
		}
		if entry.RateViolations > grace {
			entry.BlockedUntil = entry.Last.Add(target + penalty)
			return true
		}
		entry.Last = now
		return false
	}

	return true
}

func getAnnounceRateParams(ifc *Interface) (time.Duration, int, time.Duration, bool) {
	target, ok := callDurationMethod(ifc, "AnnounceRateTarget")
	if !ok || target <= 0 {
		return 0, 0, 0, false
	}
	grace, _ := callIntMethod(ifc, "AnnounceRateGrace")
	penalty, _ := callDurationMethod(ifc, "AnnounceRatePenalty")
	return target, grace, penalty, true
}

func callDurationMethod(ifc *Interface, name string) (time.Duration, bool) {
	if ifc == nil {
		return 0, false
	}
	val := reflect.ValueOf(ifc)
	if val.Kind() == reflect.Pointer && val.IsNil() {
		return 0, false
	}
	method := val.MethodByName(name)
	if !method.IsValid() || method.IsZero() || method.Type().NumIn() != 0 || method.Type().NumOut() == 0 {
		return 0, false
	}
	out := method.Call(nil)
	if len(out) == 0 {
		return 0, false
	}
	return convertToDuration(out[0].Interface())
}

func callIntMethod(ifc *Interface, name string) (int, bool) {
	if ifc == nil {
		return 0, false
	}
	val := reflect.ValueOf(ifc)
	if val.Kind() == reflect.Pointer && val.IsNil() {
		return 0, false
	}
	method := val.MethodByName(name)
	if !method.IsValid() || method.IsZero() || method.Type().NumIn() != 0 || method.Type().NumOut() == 0 {
		return 0, false
	}
	out := method.Call(nil)
	if len(out) == 0 {
		return 0, false
	}
	return convertToInt(out[0].Interface())
}

func convertToDuration(value any) (time.Duration, bool) {
	switch v := value.(type) {
	case time.Duration:
		return v, true
	case *time.Duration:
		if v == nil {
			return 0, false
		}
		return *v, true
	case int:
		return time.Duration(v) * time.Second, true
	case *int:
		if v == nil {
			return 0, false
		}
		return time.Duration(*v) * time.Second, true
	case float64:
		return time.Duration(v * float64(time.Second)), true
	case *float64:
		if v == nil {
			return 0, false
		}
		return time.Duration(*v * float64(time.Second)), true
	}
	return 0, false
}

func convertToInt(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case *int:
		if v == nil {
			return 0, false
		}
		return *v, true
	case float64:
		return int(v), true
	case *float64:
		if v == nil {
			return 0, false
		}
		return int(*v), true
	}
	return 0, false
}

func announceRandomBlob(p *Packet) []byte {
	if p == nil {
		return nil
	}
	offset := identityPubKeyLen + IdentityNameHashLength/8
	if len(p.Data) < offset+announceRandomHashLen {
		return nil
	}
	return copyBytes(p.Data[offset : offset+announceRandomHashLen])
}

func appendRandomBlob(blobs [][]byte, blob []byte) [][]byte {
	if len(blob) == 0 {
		return blobs
	}
	blobs = append(blobs, blob)
	if len(blobs) > maxRandomBlobs {
		blobs = blobs[len(blobs)-maxRandomBlobs:]
	}
	return blobs
}

func randomBlobSeen(blobs [][]byte, blob []byte) bool {
	if len(blob) == 0 {
		return false
	}
	for _, existing := range blobs {
		if bytes.Equal(existing, blob) {
			return true
		}
	}
	return false
}

func dedupeAndTailRandomBlobs(blobs [][]byte, max int) [][]byte {
	if len(blobs) == 0 || max <= 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(blobs))
	out := make([][]byte, 0, len(blobs))
	for _, blob := range blobs {
		if len(blob) == 0 {
			continue
		}
		key := string(blob)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, append([]byte(nil), blob...))
	}
	if len(out) > max {
		out = out[len(out)-max:]
	}
	return out
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

func copyRandomBlobSlice(in [][]byte) [][]byte {
	if len(in) == 0 {
		return nil
	}
	out := make([][]byte, len(in))
	for i, blob := range in {
		out[i] = copyBytes(blob)
	}
	return out
}

func handlePendingAndActiveLinks(pathReqs map[hashKey]*Interface) {
	linkMu.Lock()
	defer linkMu.Unlock()

	now := time.Now()

	filterPending := PendingLinks[:0]
	for _, link := range PendingLinks {
		if link == nil {
			continue
		}
		if link.Status == LinkClosed {
			// Python parity: if a pending link closes without being activated on a
			// non-transport instance, expire the associated path and try rediscovery.
			if !TransportEnabled() && link.destination != nil {
				destHash := link.destination.Hash()
				if ExpirePath(destHash) {
					if Owner != nil && Owner.IsConnectedToSharedInstance {
						// Shared instance will handle path request.
					} else {
						RequestPath(destHash, nil)
					}
				}
			}
			continue
		}
		if link.destination != nil && len(link.destination.Hash()) >= truncatedHashBytes {
			if key, ok := makeHashKey(link.destination.Hash()); ok {
				pathReqs[key] = nil
			}
		}
		filterPending = append(filterPending, link)
	}
	PendingLinks = filterPending

	filterActive := ActiveLinks[:0]
	for _, link := range ActiveLinks {
		if link == nil || link.Status == LinkClosed {
			continue
		}
		if !link.lastInbound.IsZero() && now.Sub(link.lastInbound) > link.StaleTime {
			// Teardown in a goroutine: this function runs under jobsMu
			// write lock, and Teardown → sendTeardownPacket → Outbound
			// needs jobsMu read lock — calling it here would deadlock.
			go link.Teardown()
			continue
		}
		if link.destination != nil && !HasPath(link.destination.Hash()) {
			if key, ok := makeHashKey(link.destination.Hash()); ok {
				pathReqs[key] = nil
			}
		}
		filterActive = append(filterActive, link)
	}
	ActiveLinks = filterActive
}

// ---- helpers ----

func getPathEntry(hash []byte) *PathEntry {
	key, ok := makeHashKey(hash)
	if !ok {
		return nil
	}

	pathTableMu.RLock()
	entry := pathTable[key]
	pathTableMu.RUnlock()
	if entry == nil {
		return nil
	}
	if time.Since(entry.Timestamp) > pathExpiration {
		pathTableMu.Lock()
		if cur := pathTable[key]; cur == entry {
			delete(pathTable, key)
		}
		pathTableMu.Unlock()
		return nil
	}
	return entry
}

func makeHashKey(hash []byte) (hashKey, bool) {
	if len(hash) < truncatedHashBytes {
		return hashKey{}, false
	}
	var key hashKey
	copy(key[:], hash[:truncatedHashBytes])
	return key, true
}

func AddPacketHash(hash []byte) {
	key, ok := makeHashKey(hash)
	if !ok {
		return
	}
	packetHashMu.Lock()
	PacketHashSet[key] = struct{}{}
	packetHashMu.Unlock()
}

func HasPacketHash(hash []byte) bool {
	key, ok := makeHashKey(hash)
	if !ok {
		return false
	}
	packetHashMu.RLock()
	_, ok = PacketHashSet[key]
	if !ok {
		_, ok = PacketHashSet2[key]
	}
	packetHashMu.RUnlock()
	return ok
}

func HasPath(hash []byte) bool {
	return getPathEntry(hash) != nil
}

// ExpirePath mirrors Python Transport.expire_path().
func ExpirePath(hash []byte) bool {
	key, ok := makeHashKey(hash)
	if !ok {
		return false
	}
	pathTableMu.Lock()
	entry := pathTable[key]
	if entry != nil {
		entry.Timestamp = time.Unix(0, 0)
	}
	pathTableMu.Unlock()
	if entry == nil {
		return false
	}
	// Force a near-term cull pass.
	TablesLastCulled = time.Time{}
	return true
}

// MarkPathUnresponsive mirrors Python Transport.mark_path_unresponsive().
func MarkPathUnresponsive(hash []byte) bool {
	key, ok := makeHashKey(hash)
	if !ok || !HasPath(hash) {
		return false
	}
	pathStatesMu.Lock()
	pathStates[key] = TransportStateUnresponsive
	pathStatesMu.Unlock()
	return true
}

// MarkPathResponsive mirrors Python Transport.mark_path_responsive().
func MarkPathResponsive(hash []byte) bool {
	key, ok := makeHashKey(hash)
	if !ok || !HasPath(hash) {
		return false
	}
	pathStatesMu.Lock()
	pathStates[key] = TransportStateResponsive
	pathStatesMu.Unlock()
	return true
}

// MarkPathUnknownState mirrors Python Transport.mark_path_unknown_state().
func MarkPathUnknownState(hash []byte) bool {
	key, ok := makeHashKey(hash)
	if !ok || !HasPath(hash) {
		return false
	}
	pathStatesMu.Lock()
	pathStates[key] = TransportStateUnknown
	pathStatesMu.Unlock()
	return true
}

// PathIsUnresponsive mirrors Python Transport.path_is_unresponsive().
func PathIsUnresponsive(hash []byte) bool {
	key, ok := makeHashKey(hash)
	if !ok {
		return false
	}
	pathStatesMu.RLock()
	state, ok := pathStates[key]
	pathStatesMu.RUnlock()
	return ok && state == TransportStateUnresponsive
}

func announceHash(p *Packet) *hashKey {
	if p == nil {
		return nil
	}
	hash := p.DestinationHash
	if len(hash) == 0 && p.Destination != nil {
		hash = p.Destination.Hash()
	}
	key, ok := makeHashKey(hash)
	if !ok {
		return nil
	}
	return &key
}

func cloneAnnouncePacket(p *Packet) *Packet {
	if p == nil {
		return nil
	}
	dest := p.Destination
	if dest == nil {
		dest = &Destination{
			hash: copyBytes(p.DestinationHash),
			Type: int(p.DestinationType),
		}
	}
	clone := NewPacket(
		dest,
		copyBytes(p.Data),
		WithPacketType(p.PacketType),
		WithPacketContext(p.Context),
		WithHeaderType(p.HeaderType),
		WithTransportType(p.TransportType),
		WithContextFlag(p.ContextFlag),
	)
	clone.AttachedInterface = p.AttachedInterface
	if len(p.TransportID) > 0 {
		clone.TransportID = copyBytes(p.TransportID)
	}
	clone.Hops = p.Hops
	clone.DestinationHash = copyBytes(p.DestinationHash)
	clone.DestinationType = p.DestinationType
	return clone
}

func randAnnounceDelay() time.Duration {
	if pathfinderRandomWindow <= 0 {
		return pathfinderRetryGrace
	}
	return time.Duration(Rand() * float64(pathfinderRandomWindow))
}

func evictOldestAnnounceLocked() {
	if len(announceTable) == 0 {
		return
	}
	var oldestKey hashKey
	oldestTime := time.Now()
	found := false
	for key, entry := range announceTable {
		if entry == nil {
			oldestKey = key
			found = true
			break
		}
		if !found || entry.Timestamp.Before(oldestTime) {
			oldestTime = entry.Timestamp
			oldestKey = key
			found = true
		}
	}
	if found {
		delete(announceTable, oldestKey)
	}
}

func TransportHasPath(hash []byte) bool {
	return HasPath(hash)
}

func TransportHopsTo(hash []byte) int {
	if entry := getPathEntry(hash); entry != nil {
		return entry.Hops
	}
	return PathfinderMaxHops
}

func TransportNextHopInterfaceBitrate(destinationHash []byte) *int {
	entry := getPathEntry(destinationHash)
	if entry == nil || entry.RecvInterface == nil {
		return nil
	}
	if InterfaceToSharedInstance(entry.RecvInterface) {
		if forced := SharedInstanceForcedBitrate(); forced > 0 {
			return &forced
		}
	}
	if entry.RecvInterface.Bitrate > 0 {
		b := entry.RecvInterface.Bitrate
		return &b
	}
	return nil
}

func TransportNextHopPerBitLatency(destinationHash []byte) *float64 {
	br := TransportNextHopInterfaceBitrate(destinationHash)
	if br == nil || *br <= 0 {
		return nil
	}
	v := 1.0 / float64(*br)
	return &v
}

func TransportNextHopPerByteLatency(destinationHash []byte) *float64 {
	perBit := TransportNextHopPerBitLatency(destinationHash)
	if perBit == nil {
		return nil
	}
	v := (*perBit) * 8.0
	return &v
}

func TransportFirstHopTimeout(destinationHash []byte) time.Duration {
	perByte := TransportNextHopPerByteLatency(destinationHash)
	if perByte == nil {
		return time.Duration(DEFAULT_PER_HOP_TIMEOUT * float64(time.Second))
	}
	timeout := float64(MTU)*(*perByte) + DEFAULT_PER_HOP_TIMEOUT
	if timeout < 0 {
		timeout = 0
	}
	return time.Duration(timeout * float64(time.Second))
}

// TransportExtraLinkProofTimeout mirrors Python Transport.extra_link_proof_timeout().
func TransportExtraLinkProofTimeout(ifc *Interface) time.Duration {
	if ifc == nil {
		return 0
	}
	br := ifc.Bitrate
	if InterfaceToSharedInstance(ifc) {
		if forced := SharedInstanceForcedBitrate(); forced > 0 {
			br = forced
		}
	}
	if br <= 0 {
		return 0
	}
	seconds := (float64(MTU) * 8.0) / float64(br)
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds * float64(time.Second))
}

func RequestPath(hash []byte, blocked *Interface) {
	key, ok := makeHashKey(hash)
	if !ok {
		return
	}
	if HasPath(hash) && !PathIsUnresponsive(hash) {
		return
	}

	now := time.Now()
	pathRequestMu.Lock()
	last := lastPathRequest[key]
	if now.Sub(last) < pathRequestMinInterval {
		pathRequestMu.Unlock()
		return
	}
	lastPathRequest[key] = now
	pathRequestMu.Unlock()

	Logf(LogDebug, "Requesting path to %s", PrettyHash(hash))

	tag := IdentityGetRandomHash()
	if blocked == nil {
		requestPathOnInterface(hash, nil, tag, false)
		return
	}
	// Send on all interfaces except the blocked one (Python: discovery forwarding behaviour).
	for _, ifc := range Interfaces {
		if ifc == nil || ifc == blocked {
			continue
		}
		requestPathOnInterface(hash, ifc, tag, false)
	}
}

func TransportRequestPath(hash []byte) {
	RequestPath(hash, nil)
}

func requestPathOnInterface(destinationHash []byte, onInterface *Interface, tag []byte, recursive bool) {
	if len(destinationHash) != truncatedHashBytes {
		return
	}
	if len(tag) == 0 {
		tag = IdentityGetRandomHash()
	}

	var payload []byte
	if TransportEnabled() && TransportIdentity != nil && len(TransportIdentity.Hash) == truncatedHashBytes {
		payload = make([]byte, 0, truncatedHashBytes*3)
		payload = append(payload, destinationHash...)
		payload = append(payload, TransportIdentity.Hash...)
		payload = append(payload, tag...)
	} else {
		payload = make([]byte, 0, truncatedHashBytes*2)
		payload = append(payload, destinationHash...)
		payload = append(payload, tag...)
	}

	// If this is a recursive path request on a specific interface, respect announce cap.
	if onInterface != nil && recursive {
		// Python behaviour:
		// - Block recursive path requests if the interface has queued announces.
		// - Block if announce cap is currently active (now < announce_allowed_at).
		// - Otherwise, update announce_allowed_at based on tx_time/announce_cap.
		if onInterface.HasQueuedAnnounces() {
			return
		}
		now := time.Now()
		if allowedAt := onInterface.AnnounceAllowedAt(); !allowedAt.IsZero() && now.Before(allowedAt) {
			return
		}

		if onInterface.Bitrate > 0 && onInterface.AnnounceCap > 0 {
			// tx_time = ((len(data)+HEADER_MINSIZE)*8)/bitrate
			txTime := (float64(len(payload)+HEADER_MINSIZE) * 8.0) / float64(onInterface.Bitrate)
			waitTime := txTime / onInterface.AnnounceCap
			if waitTime > 0 {
				onInterface.SetAnnounceAllowedAt(now.Add(time.Duration(waitTime * float64(time.Second))))
			}
		}
	}

	dst, err := NewDestination(nil, DestinationOUT, DestinationPLAIN, "rnstransport", "path", "request")
	if err != nil {
		Logf(LogError, "Could not create path request destination: %v", err)
		return
	}

	p := NewPacket(dst, payload,
		WithPacketType(PacketDATA),
		WithTransportType(Broadcast),
		WithHeaderType(HeaderType1),
		WithAttachedInterface(onInterface),
		WithoutReceipt(),
	)
	if p == nil {
		return
	}
	_ = p.Send()
}

func ForceSharedInstanceBitrate(bitrate int) {
	sharedInstanceMu.Lock()
	defer sharedInstanceMu.Unlock()
	sharedInstanceForcedBitrate = bitrate
}

func SharedInstanceForcedBitrate() int {
	sharedInstanceMu.RLock()
	defer sharedInstanceMu.RUnlock()
	return sharedInstanceForcedBitrate
}

func TransportActiveLinks() []*Link {
	linkMu.Lock()
	defer linkMu.Unlock()
	out := make([]*Link, len(ActiveLinks))
	copy(out, ActiveLinks)
	return out
}

// SharedConnectionDisappeared mirrors the Python behaviour when the local
// shared-instance connection drops.
func SharedConnectionDisappeared() {
	linkMu.Lock()
	pending := append([]*Link(nil), PendingLinks...)
	active := append([]*Link(nil), ActiveLinks...)
	linkMu.Unlock()

	for _, link := range append(active, pending...) {
		if link != nil {
			link.Teardown()
		}
	}

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

	linkTable = make(map[hashKey]*linkEntry)
	configureControlDestinations()
}

// SharedConnectionReappeared mirrors the Python behaviour when a shared
// connection returns and single destinations should re-announce.
func SharedConnectionReappeared() {
	if Owner == nil || !Owner.IsConnectedToSharedInstance {
		return
	}
	for _, dst := range Destinations {
		if dst == nil || dst.Type != DestinationSINGLE {
			continue
		}
		dst.Announce(nil, true, nil, nil, true)
	}
	configureControlDestinations()
}

type defaultTransportBackend struct{}

func (defaultTransportBackend) Outbound(p *Packet) bool {
	return Outbound(p)
}

func (defaultTransportBackend) HopsTo(hash []byte) int {
	return TransportHopsTo(hash)
}

func (defaultTransportBackend) GetFirstHopTimeout(hash []byte) time.Duration {
	return TransportFirstHopTimeout(hash)
}

func (defaultTransportBackend) GetPacketRSSI(hash []byte) *float64 {
	if len(hash) == 0 {
		return nil
	}
	localStatsMu.RLock()
	v, ok := localRSSICache[string(hash)]
	localStatsMu.RUnlock()
	if !ok {
		return nil
	}
	return &v
}

func (defaultTransportBackend) GetPacketSNR(hash []byte) *float64 {
	if len(hash) == 0 {
		return nil
	}
	localStatsMu.RLock()
	v, ok := localSNRCache[string(hash)]
	localStatsMu.RUnlock()
	if !ok {
		return nil
	}
	return &v
}

func (defaultTransportBackend) GetPacketQ(hash []byte) *float64 {
	if len(hash) == 0 {
		return nil
	}
	localStatsMu.RLock()
	v, ok := localQCache[string(hash)]
	localStatsMu.RUnlock()
	if !ok {
		return nil
	}
	return &v
}

func init() {
	Transport = defaultTransportBackend{}
}

func Cache(p *Packet, force bool) {
	if p == nil || len(p.PacketHash) == 0 {
		return
	}
	if !force && !ShouldCache(p) {
		return
	}
	if len(p.Raw) == 0 {
		if err := p.Pack(); err != nil {
			return
		}
	}
	// Persist announce packets to disk so destination_table can be reconstructed on startup.
	if Owner != nil && len(Owner.CachePath) > 0 && p.Type == PacketANNOUNCE {
		path := announceCachePath(p.PacketHash)
		if path != "" {
			_ = os.MkdirAll(filepath.Dir(path), 0o755)
			_ = os.WriteFile(path, p.Raw, 0o600)
		}
	}
	entry := &cachedPacket{
		Raw:        append([]byte(nil), p.Raw...),
		Interface:  p.ReceivingInterface,
		StoredAt:   time.Now(),
		PacketHash: append([]byte(nil), p.PacketHash...),
	}
	packetCacheMu.Lock()
	if len(packetCache) >= packetCacheMaxEntries {
		var oldestKey string
		oldest := time.Now()
		for k, v := range packetCache {
			if v.StoredAt.Before(oldest) {
				oldest = v.StoredAt
				oldestKey = k
			}
		}
		if oldestKey != "" {
			delete(packetCache, oldestKey)
		}
	}
	packetCache[string(p.PacketHash)] = entry
	packetCacheMu.Unlock()
}

// ShouldCache mirrors Python Transport.should_cache() in spirit. The Go port
// currently only guarantees cache semantics for announce packets and explicit
// forced caching.
func ShouldCache(p *Packet) bool {
	if p == nil {
		return false
	}
	return p.Type == PacketANNOUNCE
}

// CleanCache mirrors Python Transport.clean_cache()/clean_announce_cache().
func CleanCache() {
	now := time.Now()
	_ = cleanAnnounceCache(now)
	_ = cleanPacketCache(now)
}

// GetCachedPacket mirrors Python Transport.get_cached_packet().
// packetType is currently only used for "announce" to enable disk-backed lookups.
func GetCachedPacket(packetHash []byte, packetType string) *Packet {
	if len(packetHash) == 0 {
		return nil
	}
	var raw []byte
	if strings.EqualFold(packetType, "announce") || packetType == "" {
		raw = getCachedAnnounceRaw(packetHash)
	}
	if len(raw) == 0 {
		packetCacheMu.RLock()
		entry := packetCache[string(packetHash)]
		packetCacheMu.RUnlock()
		if entry != nil && len(entry.Raw) > 0 {
			raw = append([]byte(nil), entry.Raw...)
		}
	}
	if len(raw) == 0 {
		return nil
	}
	p := NewPacket(nil, raw)
	if p == nil || !p.Unpack() {
		return nil
	}
	return p
}

func CacheRequest(hash []byte, link *Link) {
	if deliverCachedPacket(hash) {
		return
	}
	if link == nil {
		return
	}
	req := NewPacket(
		link,
		hash,
		WithPacketContext(PacketCONTEXT_CACHE_REQUEST),
		WithoutReceipt(),
	)
	if req != nil {
		req.Send()
	}
}

func cacheRequestPacket(packet *Packet) bool {
	if len(packet.Data) == 0 {
		return false
	}
	return deliverCachedPacket(packet.Data)
}

func deliverCachedPacket(hash []byte) bool {
	packetCacheMu.RLock()
	entry := packetCache[string(hash)]
	packetCacheMu.RUnlock()
	var raw []byte
	var recvIf *Interface
	if entry != nil {
		raw = append([]byte(nil), entry.Raw...)
		recvIf = entry.Interface
	} else {
		// Best-effort disk-backed cache for announce packets.
		raw = getCachedAnnounceRaw(hash)
	}
	if len(raw) == 0 {
		return false
	}
	go Inbound(raw, recvIf)
	return true
}

func shouldAnnounceOnInterface(_ *Packet, _ *Interface, _ time.Time) bool {
	return true
}

func cacheLocalStats(p *Packet, _ *Interface) {
	if p == nil {
		return
	}
	if len(p.PacketHash) == 0 {
		p.PacketHash = p.GetHash()
	}
	if len(p.PacketHash) == 0 {
		return
	}
	if p.RSSI == nil && p.SNR == nil && p.Q == nil {
		return
	}

	key := string(p.PacketHash)
	localStatsMu.Lock()
	if p.RSSI != nil {
		localRSSICache[key] = *p.RSSI
	}
	if p.SNR != nil {
		localSNRCache[key] = *p.SNR
	}
	if p.Q != nil {
		localQCache[key] = *p.Q
	}
	localStatsFIFO = append(localStatsFIFO, key)
	if len(localStatsFIFO) > localStatsMax {
		evict := localStatsFIFO[0]
		localStatsFIFO = localStatsFIFO[1:]
		delete(localRSSICache, evict)
		delete(localSNRCache, evict)
		delete(localQCache, evict)
	}
	localStatsMu.Unlock()
}

func InterfaceToSharedInstance(ifc *Interface) bool {
	if ifc == nil || Owner == nil {
		return false
	}
	if Owner.SharedInstanceInterface != nil && ifc == Owner.SharedInstanceInterface {
		return true
	}
	// Best-effort name/type fallback for older initialisation paths.
	if strings.EqualFold(ifc.Type, "LocalInterface") && strings.Contains(strings.ToLower(ifc.Name), "shared instance") {
		return true
	}
	return false
}

func IsLocalClientInterface(ifc *Interface) bool {
	if ifc == nil {
		return false
	}
	for _, cif := range LocalClientInterfaces {
		if cif == ifc {
			return true
		}
	}
	return false
}

func isForLocalClient(p *Packet) bool {
	if p == nil || p.Type == PacketAnnounce {
		return false
	}
	// Python: for_local_client if destination_hash in path_table and hops == 0.
	entry := getPathEntry(p.DestinationHash)
	return entry != nil && entry.Hops == 0
}

func isForLocalClientLink(p *Packet) bool {
	if p == nil || p.Type == PacketAnnounce || len(LocalClientInterfaces) == 0 {
		return false
	}
	key, ok := makeHashKey(p.DestinationHash)
	if !ok {
		return false
	}
	linkMu.Lock()
	entry := linkTable[key]
	linkMu.Unlock()
	if entry == nil {
		return false
	}
	return IsLocalClientInterface(entry.ReceivedInterface) || IsLocalClientInterface(entry.NextHopInterface)
}

func isProofForLocal(p *Packet) bool {
	if p == nil || len(LocalClientInterfaces) == 0 || len(p.DestinationHash) == 0 {
		return false
	}
	key, ok := makeHashKey(p.DestinationHash)
	if !ok {
		return false
	}
	reverseTableMu.Lock()
	entry := reverseTable[key]
	reverseTableMu.Unlock()
	if entry == nil {
		return false
	}
	return IsLocalClientInterface(entry.ReceivedIf)
}

func isControlHash(hash []byte) bool {
	if len(hash) == 0 {
		return false
	}
	controlHashesMu.RLock()
	_, ok := controlHashes[string(hash)]
	controlHashesMu.RUnlock()
	return ok
}

func interfaceName(ifc *Interface) string {
	if ifc == nil {
		return ""
	}
	return fmt.Sprint(ifc)
}

// -------- exported helpers for Reticulum --------

func GetPathTable(maxHops int) []map[string]any {
	pathTableMu.RLock()
	defer pathTableMu.RUnlock()
	res := make([]map[string]any, 0, len(pathTable))
	for key, entry := range pathTable {
		if entry == nil {
			continue
		}
		if maxHops >= 0 && entry.Hops > maxHops {
			continue
		}
		hashCopy := make([]byte, truncatedHashBytes)
		copy(hashCopy, key[:])
		via := append([]byte(nil), entry.NextHop...)
		var expires int64
		if !entry.ExpiresAt.IsZero() {
			expires = entry.ExpiresAt.Unix()
		}
		timestamp := entry.Timestamp.Unix()
		res = append(res, map[string]any{
			"hash":      hashCopy,
			"timestamp": timestamp,
			"via":       via,
			"hops":      entry.Hops,
			"expires":   expires,
			"interface": interfaceName(entry.RecvInterface),
		})
	}
	return res
}

func GetRateTable() []map[string]any {
	announceRateMu.RLock()
	defer announceRateMu.RUnlock()
	res := make([]map[string]any, 0, len(announceRateTable))
	for key, entry := range announceRateTable {
		if entry == nil {
			continue
		}
		hashCopy := make([]byte, truncatedHashBytes)
		copy(hashCopy, key[:])
		timestamps := make([]int64, len(entry.Timestamps))
		for i, ts := range entry.Timestamps {
			timestamps[i] = ts.Unix()
		}
		res = append(res, map[string]any{
			"hash":            hashCopy,
			"last":            entry.Last.Unix(),
			"rate_violations": entry.RateViolations,
			"blocked_until":   entry.BlockedUntil.Unix(),
			"timestamps":      timestamps,
		})
	}
	return res
}

func DropPath(hash []byte) bool {
	// Python calls this "expire_path": keep the entry but mark it stale by setting timestamp to 0.
	return ExpirePath(hash)
}

func DropAllVia(via []byte) int {
	if len(via) == 0 {
		return 0
	}
	pathTableMu.Lock()
	defer pathTableMu.Unlock()
	removed := 0
	for _, entry := range pathTable {
		if entry == nil {
			continue
		}
		if len(entry.NextHop) == 0 {
			continue
		}
		if bytes.Equal(entry.NextHop, via) {
			entry.Timestamp = time.Unix(0, 0)
			removed++
		}
	}
	if removed > 0 {
		TablesLastCulled = time.Time{}
	}
	return removed
}

func NextHop(hash []byte) []byte {
	if entry := getPathEntry(hash); entry != nil {
		return append([]byte(nil), entry.NextHop...)
	}
	return nil
}

func NextHopInterfaceName(hash []byte) string {
	if entry := getPathEntry(hash); entry != nil {
		return interfaceName(entry.RecvInterface)
	}
	return ""
}

// NextHopInterfaceHWMTU mirrors Python Transport.next_hop_interface_hw_mtu().
// Returns 0 if unknown.
func NextHopInterfaceHWMTU(hash []byte) int {
	if entry := getPathEntry(hash); entry != nil && entry.RecvInterface != nil {
		return entry.RecvInterface.HWMTU
	}
	return 0
}
