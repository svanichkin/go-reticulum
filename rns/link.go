package rns

import (
	"bytes"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	Cryptography "github.com/svanichkin/go-reticulum/rns/cryptography"
	umsgpack "github.com/svanichkin/go-reticulum/rns/vendor"
)

const (
	linkCurveName          = "X25519"
	linkEcPubSize          = 64
	linkKeySize            = 32
	linkMTUMask            = 0x1FFFFF
	linkModeMask           = 0xE0
	linkSignalSize         = 3
	linkTrafficTimeoutMin  = 5 * time.Millisecond
	linkTrafficTimeoutFact = 6
	linkKeepaliveMaxRTT    = 1.75
	linkKeepaliveFact      = 4
	linkStaleGrace         = 5 * time.Second
	linkKeepaliveMax       = 360 * time.Second
	linkKeepaliveMin       = 5 * time.Second
	linkStaleFactor        = 2
	linkWatchdogMaxSleep   = 5 * time.Second
	// Python parity: default link mode is AES_256_CBC (0x01) since RNS 0.9.x.
	linkDefaultMode   = LinkModeAES256CBC
	linkDefaultPerHop = time.Duration(DEFAULT_PER_HOP_TIMEOUT) * time.Second
)

const (
	LinkPending   = 0x00
	LinkHandshake = 0x01
	LinkActive    = 0x02
	LinkStale     = 0x03
	LinkClosed    = 0x04
)

const (
	LinkTimeout          = 0x01
	LinkInitiatorClose   = 0x02
	LinkDestinationClose = 0x03
)

const (
	LinkAcceptNone = 0x00
	LinkAcceptApp  = 0x01
	LinkAcceptAll  = 0x02
)

const (
	LinkModeAES128CBC = 0x00
	LinkModeAES256CBC = 0x01
	LinkModeAES256GCM = 0x02
	LinkModeReserved  = 0x03

	// LinkModeDefault is a Go-port convenience for "use the default mode".
	// Python represents AES128CBC as 0x00, so callers must not use 0 as a sentinel.
	LinkModeDefault = -1
)

var (
	// Python parity: only AES_256_CBC is enabled since RNS 0.9.x.
	linkEnabledModes     = []int{LinkModeAES256CBC}
	linkModeDescriptions = map[int]string{
		LinkModeAES128CBC: "AES_128_CBC",
		LinkModeAES256CBC: "AES_256_CBC",
		LinkModeAES256GCM: "AES_256_GCM",
	}
)

type LinkCallbacks struct {
	LinkEstablished   func(*Link)
	LinkClosed        func(*Link)
	Packet            func([]byte, *Packet)
	Resource          func(*ResourceAdvertisement) bool
	ResourceStarted   func(*Resource)
	ResourceConcluded func(*Resource)
	RemoteIdentified  func(*Link, *Identity)
}

type Link struct {
	Mode      int
	Status    int
	Initiator bool
	// TeardownReason mirrors Python Link.teardown_reason, but is best-effort.
	// Values: LinkTimeout, LinkInitiatorClose, LinkDestinationClose.
	TeardownReason int

	LinkID []byte
	Hash   []byte

	RTT               time.Duration
	MTU               int
	MDU               int
	EstablishmentCost int
	EstablishmentRate float64
	ExpectedRate      float64

	lastInbound   time.Time
	lastOutbound  time.Time
	lastKeepalive time.Time
	lastProof     time.Time
	lastData      time.Time
	requestTime   time.Time
	estTimeout    time.Duration
	activatedAt   time.Time
	establishedCB bool
	tx            uint64
	rx            uint64
	txBytes       uint64
	rxBytes       uint64

	TrafficTimeoutFactor   float64
	KeepaliveTimeoutFactor float64
	Keepalive              time.Duration
	StaleTime              time.Duration

	owner             *Destination
	destination       *Destination
	expectedHops      int
	attachedInterface any

	requestData []byte
	packet      *Packet
	channel     *Channel

	callbacks LinkCallbacks

	resourceStrategy   int
	lastResourceWindow int
	lastResourceEIFR   float64
	outgoingResources  []*Resource
	incomingResources  []*Resource
	pendingRequests    []*RequestReceipt

	curve   ecdh.Curve
	priv    *ecdh.PrivateKey
	pub     []byte
	sigPriv ed25519.PrivateKey
	sigPub  ed25519.PublicKey

	peerPub      *ecdh.PublicKey
	peerPubBytes []byte
	peerSigPub   ed25519.PublicKey

	sharedKey  []byte
	derivedKey []byte
	token      *Cryptography.Token

	mu             sync.Mutex
	watchdogOnce   sync.Once
	watchdogStop   chan struct{}
	remoteIdentity *Identity

	trackPhyStats bool
	rssi          *float64
	snr           *float64
	q             *float64
}

const identityPublicKeyLength = x25519KeyLen + ed25519.PublicKeySize

func (l *Link) noteOutbound(context byte, size int) {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lastOutbound = now
	if context == PacketCtxKeepalive {
		l.lastKeepalive = now
	} else {
		l.lastData = now
	}
	if context != PacketCtxKeepalive {
		l.tx++
		if size > 0 {
			l.txBytes += uint64(size)
		}
	}
}

func (l *Link) noteInbound(context byte, size int) {
	now := time.Now()
	l.mu.Lock()
	activate := false
	l.lastInbound = now
	if context == PacketCtxKeepalive {
		l.lastKeepalive = now
	} else {
		l.lastData = now
	}
	if context != PacketCtxKeepalive {
		l.rx++
		if size > 0 {
			l.rxBytes += uint64(size)
		}
	}
	// Python parity: pending/handshake links must not become ACTIVE merely by receiving
	// inbound packets; activation happens after proof/RTT handling.
	// Only revive stale links on inbound traffic.
	if l.Status == LinkStale {
		activate = true
		l.Status = LinkActive
	}
	l.mu.Unlock()
	if activate {
		activateLinkInTransport(l)
	}
}

func (l *Link) observeRTT(d time.Duration) {
	if d <= 0 {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.RTT <= 0 {
		l.RTT = d
	} else {
		l.RTT = (l.RTT*3 + d) / 4
	}
}

func (l *Link) observeRTTSeconds(sec float64) {
	if sec <= 0 {
		return
	}
	l.observeRTT(time.Duration(sec * float64(time.Second)))
}

// ==== Link creation ====

func NewLink(destination *Destination, owner *Destination, mode int, establishedCB, closedCB func(*Link)) (*Link, error) {
	if mode < 0 {
		mode = linkDefaultMode
	}
	if !containsInt(mode, linkEnabledModes) {
		return nil, fmt.Errorf("link mode %d disabled", mode)
	}

	l := &Link{
		Mode:                   mode,
		Status:                 LinkPending,
		Initiator:              destination != nil,
		MTU:                    defaultLinkMTU(),
		Keepalive:              linkKeepaliveMax,
		StaleTime:              linkKeepaliveMax * linkStaleFactor,
		TrafficTimeoutFactor:   linkTrafficTimeoutFact,
		KeepaliveTimeoutFactor: linkKeepaliveFact,
		owner:                  owner,
		destination:            destination,
		curve:                  ecdh.X25519(),
		resourceStrategy:       LinkAcceptNone,
		outgoingResources:      make([]*Resource, 0),
		incomingResources:      make([]*Resource, 0),
	}

	if establishedCB != nil {
		l.callbacks.LinkEstablished = establishedCB
	}
	if closedCB != nil {
		l.callbacks.LinkClosed = closedCB
	}

	priv, err := l.curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	l.priv = priv
	l.pub = priv.PublicKey().Bytes()

	if l.Initiator {
		_, sig, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, err
		}
		l.sigPriv = sig
		l.sigPub = sig.Public().(ed25519.PublicKey)
	} else if owner != nil && owner.identity != nil {
		l.sigPriv = owner.identity.sigPriv
		l.sigPub = owner.identity.sigPub
	} else {
		return nil, errors.New("owner identity required for incoming link")
	}

	l.updateMDU()
	l.watchdogStop = make(chan struct{})
	if l.Initiator {
		if destination != nil && len(destination.hash) > 0 {
			l.expectedHops = Transport.HopsTo(destination.hash)
			if l.expectedHops <= 0 {
				l.expectedHops = 1
			}
			// Python: get_first_hop_timeout + ESTABLISHMENT_TIMEOUT_PER_HOP*hops
			l.estTimeout = Transport.GetFirstHopTimeout(destination.hash) + linkDefaultPerHop*time.Duration(l.expectedHops)
		}
		l.startWatchdog()
		if err := l.sendLinkRequest(); err != nil {
			Log(fmt.Sprintf("Could not send link request: %v", err), LOG_ERROR)
		} else {
			registerLinkWithTransport(l)
		}
	} else {
		registerLinkWithTransport(l)
	}
	return l, nil
}

func NewOutgoingLink(destination *Destination, mode int, establishedCB, closedCB func(*Link)) (*Link, error) {
	return NewLink(destination, nil, mode, establishedCB, closedCB)
}

func NewIncomingLink(owner *Destination, peerPub, peerSigPub []byte, mode int) (*Link, error) {
	l, err := NewLink(nil, owner, mode, nil, nil)
	if err != nil {
		return nil, err
	}
	if err := l.loadPeer(peerPub, peerSigPub); err != nil {
		return nil, err
	}
	return l, nil
}

func (l *Link) sendLinkRequest() error {
	if l.destination == nil {
		return errors.New("link has no destination")
	}
	if len(l.pub) == 0 || len(l.sigPub) == 0 {
		return errors.New("link key material missing")
	}
	// Python: if link_mtu_discovery() and next-hop hw mtu is known -> signal that MTU, else signal Reticulum MTU.
	mtu := defaultLinkMTU()
	if LinkMTUDiscovery() && l.destination != nil {
		if nh := NextHopInterfaceHWMTU(l.destination.hash); nh > 0 {
			mtu = nh
		}
	}
	signalling, err := linkSignallingBytes(mtu, l.Mode)
	if err != nil {
		return err
	}
	payload := make([]byte, 0, len(l.pub)+len(l.sigPub)+len(signalling))
	payload = append(payload, l.pub...)
	payload = append(payload, l.sigPub...)
	payload = append(payload, signalling...)

	packet := NewPacket(
		l.destination,
		payload,
		WithPacketType(PacketTypeLinkRequest),
		WithTransportType(TransportDirect),
		WithCreateReceipt(true),
	)
	if packet == nil {
		return errors.New("could not build link request packet")
	}
	if err := packet.Pack(); err != nil {
		return err
	}

	l.setLinkID(packet)
	l.packet = packet
	l.requestData = payload
	l.EstablishmentCost += len(packet.Raw)

	Logf(LogDebug, "LINKREQUEST: linkID=%x dest=%x mode=%d hops=%d timeout=%v",
		l.LinkID, packet.DestinationHash, l.Mode, l.expectedHops, l.estTimeout)

	receipt := packet.Send()
	l.requestTime = time.Now()
	if l.estTimeout <= 0 {
		l.estTimeout = linkDefaultPerHop + linkKeepaliveMax
	}
	l.noteOutbound(PacketCtxNone, len(payload))
	if receipt == nil && !packet.Sent {
		return errors.New("link request send failed")
	}
	return nil
}

func (l *Link) SetLinkEstablishedCallback(cb func(*Link)) {
	l.callbacks.LinkEstablished = cb
}

func (l *Link) SetLinkClosedCallback(cb func(*Link)) {
	l.callbacks.LinkClosed = cb
}

func (l *Link) SetPacketCallback(cb func([]byte, *Packet)) {
	l.callbacks.Packet = cb
}

func (l *Link) SetResourceCallback(cb func(*ResourceAdvertisement) bool) {
	l.callbacks.Resource = cb
}

func (l *Link) SetResourceStartedCallback(cb func(*Resource)) {
	l.callbacks.ResourceStarted = cb
}

func (l *Link) SetResourceConcludedCallback(cb func(*Resource)) {
	l.callbacks.ResourceConcluded = cb
}

func (l *Link) SetRemoteIdentifiedCallback(cb func(*Link, *Identity)) {
	l.callbacks.RemoteIdentified = cb
}

// Channel returns (and lazily creates) the channel for this link.
func (l *Link) Channel() *Channel {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.channel == nil {
		l.channel = NewChannel(NewLinkChannelOutlet(l))
	}
	return l.channel
}

func (l *Link) RemoteIdentity() *Identity {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.remoteIdentity
}

func (l *Link) Request(
	path string,
	data any,
	responseCb func(*RequestReceipt),
	failedCb func(*RequestReceipt),
	progressCb func(*RequestReceipt),
	timeout float64,
) *RequestReceipt {
	if path == "" {
		Log("Request path cannot be empty", LOG_WARNING)
		return nil
	}

	if timeout <= 0 {
		timeout = l.RTT.Seconds()*l.TrafficTimeoutFactor + 1.125*ResponseMaxGraceTime.Seconds()
	}

	pathHash := TruncatedHash([]byte(path))
	unpacked := []any{nowSeconds(), pathHash, data}
	packedRequest, err := umsgpack.Packb(unpacked)
	if err != nil {
		Log(fmt.Sprintf("Could not pack request payload for %s: %v", path, err), LOG_ERROR)
		return nil
	}

	if len(packedRequest) <= l.MDU {
		packet := NewPacket(
			l,
			packedRequest,
			WithPacketContext(PacketCtxRequest),
		)
		if packet == nil {
			Log("Failed to build request packet", LOG_ERROR)
			return nil
		}
		packetReceipt := packet.Send()
		if packetReceipt == nil {
			if !packet.Sent {
				Log("Request packet could not be sent", LOG_WARNING)
				return nil
			}
			if packet.Receipt != nil {
				packetReceipt = packet.Receipt
			}
		}
		if packetReceipt == nil {
			packetReceipt = NewPacketReceipt(packet)
		}
		return newRequestReceipt(
			l,
			packetReceipt,
			nil,
			timeout,
			len(packedRequest),
			responseCb,
			failedCb,
			progressCb,
		)
	}

	requestID := TruncatedHash(packedRequest)
	timeoutCopy := timeout
	callback := func(res *Resource) {
		l.ResourceConcluded(res)
		l.requestResourceConcluded(res)
	}
	res, err := NewResource(
		packedRequest,
		nil,
		l,
		nil,
		true,
		true,
		callback,
		nil,
		&timeoutCopy,
		0,
		nil,
		requestID,
		false,
		0,
	)
	if err != nil {
		Log(fmt.Sprintf("Could not send request as resource: %v", err), LOG_ERROR)
		return nil
	}
	if res == nil {
		Log("NewResource returned nil for request", LOG_ERROR)
		return nil
	}
	return newRequestReceipt(
		l,
		nil,
		requestID,
		timeout,
		len(packedRequest),
		responseCb,
		failedCb,
		progressCb,
	)
}

func (l *Link) SetResourceStrategy(strategy int) error {
	if strategy != LinkAcceptNone && strategy != LinkAcceptApp && strategy != LinkAcceptAll {
		return fmt.Errorf("unsupported resource strategy %d", strategy)
	}
	l.mu.Lock()
	l.resourceStrategy = strategy
	l.mu.Unlock()
	return nil
}

// ==== Resource handling ====

func (l *Link) RegisterIncomingResource(res *Resource) {
	if res == nil {
		return
	}
	l.mu.Lock()
	l.incomingResources = append(l.incomingResources, res)
	l.mu.Unlock()
}

func (l *Link) RegisterOutgoingResource(res *Resource) {
	if res == nil {
		return
	}
	l.mu.Lock()
	l.outgoingResources = append(l.outgoingResources, res)
	l.mu.Unlock()
}

func (l *Link) CancelIncomingResource(res *Resource) {
	if res == nil {
		return
	}
	l.mu.Lock()
	removed := l.removeResourceLocked(&l.incomingResources, res)
	l.mu.Unlock()
	if !removed {
		Log(fmt.Sprintf("Attempt to cancel non-existing incoming resource on %s", l), LOG_ERROR)
	}
}

func (l *Link) CancelOutgoingResource(res *Resource) {
	if res == nil {
		return
	}
	l.mu.Lock()
	removed := l.removeResourceLocked(&l.outgoingResources, res)
	l.mu.Unlock()
	if !removed {
		Log(fmt.Sprintf("Attempt to cancel non-existing outgoing resource on %s", l), LOG_ERROR)
	}
}

func (l *Link) HasIncomingResource(hash []byte) bool {
	if len(hash) == 0 {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, res := range l.incomingResources {
		if res != nil && bytes.Equal(res.hash, hash) {
			return true
		}
	}
	return false
}

func (l *Link) addPendingRequest(rr *RequestReceipt) {
	if rr == nil {
		return
	}
	l.mu.Lock()
	l.pendingRequests = append(l.pendingRequests, rr)
	l.mu.Unlock()
}

func (l *Link) removePendingRequest(rr *RequestReceipt) {
	if rr == nil {
		return
	}
	l.mu.Lock()
	for i, pending := range l.pendingRequests {
		if pending == rr {
			l.pendingRequests = append(l.pendingRequests[:i], l.pendingRequests[i+1:]...)
			break
		}
	}
	l.mu.Unlock()
}

func (l *Link) pendingRequestByID(requestID []byte) *RequestReceipt {
	if len(requestID) == 0 {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, pending := range l.pendingRequests {
		if pending == nil {
			continue
		}
		if bytes.Equal(pending.RequestID(), requestID) {
			return pending
		}
	}
	return nil
}

func (l *Link) GetLastResourceWindow() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.lastResourceWindow
}

func (l *Link) GetLastResourceEIFR() float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.lastResourceEIFR
}

func (l *Link) ReadyForNewResource() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.outgoingResources) == 0
}

func (l *Link) ResourceConcluded(res *Resource) {
	if res == nil {
		return
	}
	var cb func(*Resource)
	l.mu.Lock()
	removedIncoming := l.removeResourceLocked(&l.incomingResources, res)
	removedOutgoing := l.removeResourceLocked(&l.outgoingResources, res)
	if removedIncoming {
		l.lastResourceWindow = res.window
		l.lastResourceEIFR = res.eifr
	}
	if removedIncoming || removedOutgoing {
		d := time.Since(res.startedTransferring)
		if d <= 0 {
			d = time.Millisecond
		}
		l.ExpectedRate = float64(res.size*8) / d.Seconds()
	}
	cb = l.callbacks.ResourceConcluded
	l.mu.Unlock()

	if cb != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					Log(fmt.Sprintf("resource concluded callback panic on %s: %v", l, r), LOG_ERROR)
				}
			}()
			cb(res)
		}()
	}
}

func (l *Link) removeResourceLocked(list *[]*Resource, target *Resource) bool {
	if list == nil || target == nil {
		return false
	}
	for i, res := range *list {
		if res == target {
			*list = append((*list)[:i], (*list)[i+1:]...)
			return true
		}
	}
	return false
}

func (l *Link) findIncomingResource(hash []byte) *Resource {
	if len(hash) == 0 {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, res := range l.incomingResources {
		if res != nil && bytes.Equal(res.hash, hash) {
			return res
		}
	}
	return nil
}

func (l *Link) findOutgoingResource(hash []byte) *Resource {
	if len(hash) == 0 {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, res := range l.outgoingResources {
		if res != nil && bytes.Equal(res.hash, hash) {
			return res
		}
	}
	return nil
}

func (l *Link) handleRequest(requestID []byte, unpacked []any) {
	_ = l.handleRequestAndReportAllowed(requestID, unpacked, nil)
}

func (l *Link) sendResponsePayload(handler *RequestHandler, requestID []byte, response any) {
	payload, err := umsgpack.Packb([]any{requestID, response})
	if err != nil {
		Log(fmt.Sprintf("Could not pack response for %s: %v", handler.Path, err), LOG_ERROR)
		return
	}

	if len(payload) <= l.MDU {
		packet := NewPacket(
			l,
			payload,
			WithPacketContext(PacketCtxResponse),
			WithoutReceipt(),
		)
		if packet == nil {
			Log("Failed to create response packet", LOG_ERROR)
			return
		}
		if receipt := packet.Send(); receipt == nil && !packet.Sent {
			Log("Response packet could not be sent", LOG_WARNING)
		}
		return
	}

	timeout := time.Duration(float64(l.RTT)*l.TrafficTimeoutFactor) + ResponseMaxGraceTime
	timeoutSeconds := timeout.Seconds()
	callback := func(res *Resource) {
		l.ResourceConcluded(res)
	}
	if _, err := NewResource(
		payload,
		nil,
		l,
		nil,
		true,
		handler.AutoCompress,
		callback,
		nil,
		&timeoutSeconds,
		0,
		nil,
		requestID,
		true,
		0,
	); err != nil {
		Log(fmt.Sprintf("Could not send response as resource: %v", err), LOG_ERROR)
	}
}

func (l *Link) handleResponse(requestID []byte, response any, transferSize int) {
	if len(requestID) == 0 {
		return
	}
	pending := l.pendingRequestByID(requestID)
	if pending == nil {
		return
	}
	pending.responseReceived(response, nil, transferSize)
}

func (l *Link) requestResourceConcluded(res *Resource) {
	if res == nil {
		return
	}
	if res.Status() != ResourceComplete {
		Log("Incoming request resource failed, ignoring", LOG_DEBUG)
		return
	}
	data, err := os.ReadFile(res.DataFile())
	if err != nil {
		Log(fmt.Sprintf("Could not read completed request resource: %v", err), LOG_ERROR)
		return
	}
	var unpacked []any
	if err := umsgpack.Unpackb(data, &unpacked); err != nil {
		Log(fmt.Sprintf("Could not decode request resource payload: %v", err), LOG_ERROR)
		return
	}
	l.handleRequest(copyBytes(res.requestID), unpacked)
}

func (l *Link) responseResourceConcluded(res *Resource) {
	if res == nil {
		return
	}
	reqID := copyBytes(res.requestID)
	pending := l.pendingRequestByID(reqID)
	if pending == nil {
		return
	}
	if res.Status() != ResourceComplete {
		pending.requestTimedOut()
		return
	}
	if res.hasMetadata {
		pending.responseReceived(res, res.Metadata(), res.TotalSize())
		return
	}
	data, err := os.ReadFile(res.DataFile())
	if err != nil {
		Log(fmt.Sprintf("Could not read response resource data: %v", err), LOG_ERROR)
		pending.requestTimedOut()
		return
	}
	var unpacked []any
	if err := umsgpack.Unpackb(data, &unpacked); err != nil || len(unpacked) < 2 {
		Log("Malformed response resource payload", LOG_ERROR)
		pending.requestTimedOut()
		return
	}
	requestID := bytesFromAny(unpacked[0])
	if len(requestID) == 0 {
		requestID = reqID
	}
	response := unpacked[1]
	pending.responseReceived(response, nil, res.TotalSize())
}

func (l *Link) responseResourceProgress(res *Resource) {
	if res == nil || res.requestID == nil {
		return
	}
	pending := l.pendingRequestByID(res.requestID)
	if pending == nil {
		return
	}
	pending.responseResourceProgress(res)
}

// ==== Incoming request validation ====

func LinkValidateRequest(owner *Destination, data []byte, packet *Packet) *Link {
	if owner == nil {
		return nil
	}

	if len(data) != linkEcPubSize && len(data) != linkEcPubSize+linkSignalSize {
		Log(fmt.Sprintf("Invalid link request payload size of %d bytes, dropping request", len(data)), LOG_DEBUG)
		return nil
	}

	peerPub := data[:linkEcPubSize/2]
	peerSig := data[linkEcPubSize/2 : linkEcPubSize]

	link, err := NewIncomingLink(owner, peerPub, peerSig, linkDefaultMode)
	if err != nil {
		Log(fmt.Sprintf("Validating link request failed: %v", err), LOG_VERBOSE)
		return nil
	}

	link.setLinkID(packet)

	if len(data) == linkEcPubSize+linkSignalSize {
		Log("Link request includes MTU signalling", LOG_DEBUG)
		mtu := defaultLinkMTU()
		if mtus, ok := linkMTUFromLRPacket(packet); ok {
			mtu = mtus
		}
		link.MTU = mtu
	}

	if mode := linkModeFromLRPacket(packet); containsInt(mode, linkEnabledModes) {
		link.Mode = mode
	}

	link.updateMDU()
	if desc, ok := linkModeDescriptions[link.Mode]; ok {
		Log(fmt.Sprintf("Incoming link request with mode %s", desc), LOG_DEBUG)
	}
	link.attachedInterface = packet.ReceivingInterface
	link.destination = packet.Destination

	// Python: establishment_timeout = PER_HOP*max(1, packet.hops) + KEEPALIVE
	hops := 1
	if packet != nil && packet.Hops > 0 {
		hops = int(packet.Hops)
	}
	link.estTimeout = linkDefaultPerHop*time.Duration(hops) + linkKeepaliveMax

	if err := link.Handshake(); err != nil {
		Log(fmt.Sprintf("Handshake failed: %v", err), LOG_ERROR)
		return nil
	}
	if packet != nil {
		link.EstablishmentCost += len(packet.Raw)
	}
	link.requestTime = time.Now()
	link.prove()
	link.lastInbound = time.Now()
	link.updatePhyStatsForce(packet)

	link.startWatchdog()
	Log(fmt.Sprintf("Incoming link request %s accepted", link), LOG_DEBUG)
	return link
}

// Receive processes an inbound packet on this link, updating timers and
// handing channel payloads to the Channel implementation.
func (l *Link) Receive(packet *Packet) {
	if packet == nil {
		return
	}
	l.updatePhyStats(packet)
	l.noteInbound(packet.Context, len(packet.Data))

	// Link packet proofs: update matching receipts (Python Transport.receipts parity).
	if packet.PacketType == PacketTypeProof && packet.Context == PacketCtxNone {
		// Proof payload is explicit: packet_hash || signature
		for _, rc := range Receipts {
			if rc == nil || rc.Status != ReceiptSent {
				continue
			}
			if rc.Link != l {
				continue
			}
			_ = rc.validateLinkProof(packet.Data, l, packet)
		}
		return
	}

	// Plain link data packets (PacketCtxNone) are delivered to the Packet callback.
	// This mirrors Python Link.set_packet_callback() behaviour.
	if packet.PacketType == PacketTypeData && packet.Context == PacketCtxNone {
		plaintext := packet.Data
		if pt, err := l.Decrypt(packet.Data); err == nil && len(pt) > 0 {
			plaintext = pt
		} else if err != nil {
			Log(fmt.Sprintf("%s failed to decrypt packet: %v", l, err), LOG_WARNING)
			return
		}
		if cb := l.callbacks.Packet; cb != nil {
			func() {
				defer func() {
					if r := recover(); r != nil {
						Log(fmt.Sprintf("Packet callback panic on %s: %v", l, r), LOG_ERROR)
					}
				}()
				cb(plaintext, packet)
			}()
		}
		// Python parity: link data packets are always proven back to the sender to
		// satisfy packet receipts, regardless of whether a packet callback is set.
		l.ProvePacket(packet)
		return
	}

	// For link packets, always prove received data packets (receiver-side acknowledgement).
	// This mirrors Python's link/destination proof behaviour for packets that generate receipts.
	if packet.PacketType == PacketTypeData &&
		packet.Context != PacketCtxKeepalive &&
		packet.Context != PacketCtxLRProof &&
		packet.Context != PacketCtxLRRTT &&
		packet.Context != PacketCtxLinkClose &&
		packet.Context != PacketCtxLinkIdentify &&
		// Python parity: requests that are not allowed should not be proven as delivered.
		packet.Context != PacketCtxRequest {
		l.ProvePacket(packet)
	}

	switch packet.Context {
	case PacketCtxChannel:
		l.handleChannelPacket(packet)
	case PacketCtxKeepalive:
		// Python: the destination replies to initiator keepalive 0xFF with 0xFE.
		if !l.Initiator && bytes.Equal(packet.Data, []byte{0xFF}) {
			go l.sendKeepaliveReply()
		}
	case PacketCtxLRProof:
		l.handleLRProof(packet)
	case PacketCtxLRRTT:
		l.handleLRRTT(packet)
	case PacketCtxLinkIdentify:
		l.handleLinkIdentify(packet)
	case PacketCtxRequest:
		l.handleRequestPacket(packet)
	case PacketCtxResponse:
		l.handleResponsePacket(packet)
	case PacketCtxResourceAdv:
		l.handleResourceAdvertisement(packet)
	case PacketCtxResourceReq:
		l.handleResourceRequestPacket(packet)
	case PacketCtxResourceHMU:
		l.handleResourceHashmapUpdate(packet)
	case PacketCtxResourceICL:
		l.handleResourceCancelIncoming(packet)
	case PacketCtxResourceRCL:
		l.handleResourceRejection(packet)
	case PacketCtxResource:
		l.handleResourceData(packet)
	default:
		if packet.PacketType == PacketTypeProof && packet.Context == PacketCtxResourcePrf {
			l.handleResourceProof(packet)
		} else if packet.Context == PacketCtxLinkClose {
			l.handleLinkClose(packet)
		}
	}
}

func (l *Link) TrackPhyStats(track bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.trackPhyStats = track
}

func (l *Link) GetRSSI() *float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.rssi
}

func (l *Link) GetSNR() *float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.snr
}

func (l *Link) GetQ() *float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.q
}

func (l *Link) updatePhyStats(packet *Packet) {
	if l == nil || packet == nil {
		return
	}
	if !l.trackPhyStats {
		return
	}
	rssi := packet.GetRSSI()
	snr := packet.GetSNR()
	q := packet.GetQ()

	l.mu.Lock()
	defer l.mu.Unlock()
	if rssi != nil {
		v := *rssi
		l.rssi = &v
	}
	if snr != nil {
		v := *snr
		l.snr = &v
	}
	if q != nil {
		v := *q
		l.q = &v
	}
}

func (l *Link) updatePhyStatsForce(packet *Packet) {
	if l == nil || packet == nil {
		return
	}
	rssi := packet.GetRSSI()
	snr := packet.GetSNR()
	q := packet.GetQ()

	l.mu.Lock()
	defer l.mu.Unlock()
	if rssi != nil {
		v := *rssi
		l.rssi = &v
	}
	if snr != nil {
		v := *snr
		l.snr = &v
	}
	if q != nil {
		v := *q
		l.q = &v
	}
}

func (l *Link) updateKeepalive() {
	l.mu.Lock()
	defer l.mu.Unlock()
	rtt := l.RTT
	if rtt <= 0 {
		return
	}
	keep := time.Duration(float64(rtt) * (float64(linkKeepaliveMax) / linkKeepaliveMaxRTT))
	if keep > linkKeepaliveMax {
		keep = linkKeepaliveMax
	}
	if keep < linkKeepaliveMin {
		keep = linkKeepaliveMin
	}
	l.Keepalive = keep
	l.StaleTime = keep * linkStaleFactor
}

func (l *Link) sendKeepalive() {
	p := NewPacket(
		l,
		[]byte{0xFF},
		WithPacketContext(PacketCtxKeepalive),
		WithoutReceipt(),
	)
	if p == nil {
		return
	}
	_ = p.Send()
	l.noteOutbound(PacketCtxKeepalive, 1)
}

func (l *Link) sendKeepaliveReply() {
	p := NewPacket(
		l,
		[]byte{0xFE},
		WithPacketContext(PacketCtxKeepalive),
		WithoutReceipt(),
	)
	if p == nil {
		return
	}
	_ = p.Send()
	l.noteOutbound(PacketCtxKeepalive, 1)
}

func (l *Link) prove() {
	// Only the destination side proves link requests.
	if l == nil || l.Initiator || l.owner == nil || l.owner.identity == nil {
		return
	}
	if len(l.LinkID) == 0 || len(l.pub) == 0 || len(l.sigPub) == 0 {
		return
	}
	signalling, err := linkSignallingBytes(l.MTU, l.Mode)
	if err != nil {
		return
	}
	signed := make([]byte, 0, len(l.LinkID)+len(l.pub)+len(l.sigPub)+len(signalling))
	signed = append(signed, l.LinkID...)
	signed = append(signed, l.pub...)
	signed = append(signed, l.sigPub...)
	signed = append(signed, signalling...)
	sig, err := l.owner.identity.Sign(signed)
	if err != nil {
		return
	}
	proofData := make([]byte, 0, len(sig)+len(l.pub)+len(signalling))
	proofData = append(proofData, sig...)
	proofData = append(proofData, l.pub...)
	proofData = append(proofData, signalling...)

	proof := NewPacket(
		l,
		proofData,
		WithPacketType(PacketTypeProof),
		WithPacketContext(PacketCtxLRProof),
		WithoutReceipt(),
	)
	if proof == nil {
		return
	}
	_ = proof.Send()
	if len(proof.Raw) > 0 {
		l.EstablishmentCost += len(proof.Raw)
	}
	l.noteOutbound(PacketCtxLRProof, len(proofData))
}

func (l *Link) handleLRProof(packet *Packet) {
	if l == nil || packet == nil || !l.Initiator {
		return
	}
	if l.Status != LinkPending || l.destination == nil || l.destination.identity == nil {
		return
	}

	mode := linkModeFromProofPacket(packet)
	if mode != l.Mode {
		l.teardown(LinkTimeout)
		return
	}

	confirmedMTU := 0
	signalling := []byte{}
	if mtu, ok := linkMTUFromProofPacket(packet); ok {
		confirmedMTU = mtu
		sb, err := linkSignallingBytes(confirmedMTU, mode)
		if err == nil {
			signalling = sb
		}
	}

	if len(packet.Data) < ed25519.SignatureSize+linkEcPubSize/2 {
		return
	}
	signature := packet.Data[:ed25519.SignatureSize]
	peerPub := packet.Data[ed25519.SignatureSize : ed25519.SignatureSize+linkEcPubSize/2]
	peerSigPub := l.destination.identity.GetPublicKey()[linkEcPubSize/2 : linkEcPubSize]

	_ = l.loadPeer(peerPub, peerSigPub)
	if err := l.Handshake(); err != nil {
		Log(fmt.Sprintf("Handshake failed: %v", err), LOG_ERROR)
		l.teardown(LinkTimeout)
		return
	}
	l.EstablishmentCost += len(packet.Raw)

	signed := make([]byte, 0, len(l.LinkID)+len(peerPub)+len(peerSigPub)+len(signalling))
	signed = append(signed, l.LinkID...)
	signed = append(signed, peerPub...)
	signed = append(signed, peerSigPub...)
	signed = append(signed, signalling...)

	if !l.destination.identity.Validate(signature, signed) {
		Log("Invalid link proof signature received, ignoring", LOG_DEBUG)
		return
	}

	now := time.Now()
	if !l.requestTime.IsZero() {
		l.observeRTT(now.Sub(l.requestTime))
	}
	l.attachedInterface = packet.ReceivingInterface
	l.remoteIdentity = l.destination.identity
	if confirmedMTU > 0 {
		l.MTU = confirmedMTU
	}
	l.updateMDU()

	l.Status = LinkActive
	l.activatedAt = now
	l.lastProof = now
	activateLinkInTransport(l)
	l.updateKeepalive()

	// Send RTT packet to the destination side.
	if payload, err := umsgpack.Packb(l.RTT.Seconds()); err == nil {
		rttPkt := NewPacket(l, payload, WithPacketContext(PacketCtxLRRTT), WithoutReceipt())
		if rttPkt != nil {
			_ = rttPkt.Send()
			l.noteOutbound(PacketCtxLRRTT, len(payload))
		}
	}

	if l.RTT > 0 && l.EstablishmentCost > 0 {
		l.EstablishmentRate = float64(l.EstablishmentCost) / l.RTT.Seconds()
	}

	if !l.establishedCB {
		l.establishedCB = true
		if cb := l.callbacks.LinkEstablished; cb != nil {
			go cb(l)
		}
	}
}

func (l *Link) handleLRRTT(packet *Packet) {
	if l == nil || packet == nil || l.Initiator {
		return
	}
	plaintext, err := l.Decrypt(packet.Data)
	if err != nil || len(plaintext) == 0 {
		return
	}
	var rttSec float64
	if err := umsgpack.Unpackb(plaintext, &rttSec); err != nil {
		return
	}
	measured := 0.0
	if !l.requestTime.IsZero() {
		measured = time.Since(l.requestTime).Seconds()
	}
	if rttSec < measured {
		rttSec = measured
	}
	l.observeRTTSeconds(rttSec)
	l.Status = LinkActive
	l.activatedAt = time.Now()
	l.lastProof = l.activatedAt
	activateLinkInTransport(l)
	l.updateKeepalive()

	if l.RTT > 0 && l.EstablishmentCost > 0 {
		l.EstablishmentRate = float64(l.EstablishmentCost) / l.RTT.Seconds()
	}

	if l.owner != nil && l.owner.callbacks.LinkEstablished != nil && !l.establishedCB {
		l.establishedCB = true
		go l.owner.callbacks.LinkEstablished(l)
	}
}

// ==== Crypto and helper methods ====

func (l *Link) loadPeer(pub, sig []byte) error {
	if len(pub) == 0 || len(sig) == 0 {
		return errors.New("peer key material missing")
	}
	peerPub, err := l.curve.NewPublicKey(pub)
	//nolint:wrapcheck
	if err != nil {
		return err
	}
	l.peerPub = peerPub
	l.peerPubBytes = append([]byte{}, pub...)
	l.peerSigPub = ed25519.PublicKey(append([]byte{}, sig...))
	return nil
}

func (l *Link) updateMDU() {
	mtu := l.MTU
	if mtu == 0 {
		mtu = defaultLinkMTU()
	}
	// Python:
	// floor((mtu-IFAC_MIN_SIZE-HEADER_MINSIZE-TOKEN_OVERHEAD)/AES128_BLOCKSIZE)*AES128_BLOCKSIZE - 1
	overhead := IFAC_MIN_SIZE + HEADER_MINSIZE + Cryptography.Overhead
	if mtu <= overhead {
		l.MDU = 0
		return
	}
	block := 16
	md := ((mtu - overhead) / block) * block
	if md <= 0 {
		l.MDU = 0
		return
	}
	l.MDU = md - 1
}

func (l *Link) Handshake() error {
	if l.peerPub == nil {
		return errors.New("peer public key missing")
	}
	shared, err := l.priv.ECDH(l.peerPub)
	if err != nil {
		return err
	}
	l.sharedKey = shared

	derivedLen := l.derivedKeyLength()
	buf, err := Cryptography.HKDF(derivedLen, shared, l.getSalt(), l.getContext())
	if err != nil {
		return err
	}
	l.derivedKey = buf

	key := buf
	tok, err := Cryptography.NewToken(key)
	if err != nil {
		return err
	}
	l.token = tok
	l.Status = LinkHandshake
	return nil
}

func (l *Link) derivedKeyLength() int {
	switch l.Mode {
	case LinkModeAES128CBC:
		return 32
	case LinkModeAES256CBC, LinkModeAES256GCM:
		return 64
	default:
		return 64
	}
}

func (l *Link) getSalt() []byte {
	// Python: get_salt() -> self.link_id
	return l.LinkID
}

func (l *Link) getContext() []byte {
	return nil
}

func (l *Link) Encrypt(plaintext []byte) ([]byte, error) {
	if l.token == nil {
		return nil, errors.New("link token missing")
	}
	return l.token.Encrypt(plaintext)
}

func (l *Link) Decrypt(ciphertext []byte) ([]byte, error) {
	if l.token == nil {
		return nil, errors.New("link token missing")
	}
	return l.token.Decrypt(ciphertext)
}

func (l *Link) Sign(message []byte) ([]byte, error) {
	if l.sigPriv == nil {
		return nil, errors.New("link has no signing key")
	}
	sig := ed25519.Sign(l.sigPriv, message)
	return sig, nil
}

func (l *Link) Validate(signature, message []byte) bool {
	if l.peerSigPub == nil {
		return false
	}
	return ed25519.Verify(l.peerSigPub, message, signature)
}

// ProvePacket mirrors Python Link.prove_packet(): it sends an explicit proof
// (packet_hash + signature) back over the link.
func (l *Link) ProvePacket(packet *Packet) {
	if l == nil || packet == nil {
		return
	}
	hash := packet.GetHash()
	if len(hash) == 0 {
		return
	}
	sig, err := l.Sign(hash)
	if err != nil {
		return
	}
	proofData := make([]byte, 0, len(hash)+len(sig))
	proofData = append(proofData, hash...)
	proofData = append(proofData, sig...)
	proof := NewPacket(l, proofData, WithPacketType(PacketTypeProof), WithoutReceipt())
	if proof == nil {
		return
	}
	_ = proof.Send()
}

func (l *Link) setLinkID(packet *Packet) {
	if packet == nil {
		return
	}
	// Python parity: Link.link_id_from_lr_packet(packet)
	// Uses packet.get_hashable_part(), and if packet.data contains signalling bytes
	// beyond Link.ECPUBSIZE, those bytes are excluded from the hashable material.
	hashable := packet.getHashablePart()
	if len(packet.Data) > linkEcPubSize {
		diff := len(packet.Data) - linkEcPubSize
		if diff > 0 && diff <= len(hashable) {
			hashable = hashable[:len(hashable)-diff]
		}
	}
	l.LinkID = TruncatedHash(hashable)
	l.Hash = append([]byte{}, l.LinkID...)
}

func (l *Link) Identify(identity *Identity) {
	if identity == nil {
		Log("Identify called with nil identity", LOG_WARNING)
		return
	}

	l.mu.Lock()
	if !l.Initiator {
		l.mu.Unlock()
		Log("Only the link initiator can identify towards the remote peer", LOG_DEBUG)
		return
	}
	if l.Status != LinkActive || len(l.LinkID) == 0 {
		l.mu.Unlock()
		Log("Link is not ready to send identity information", LOG_DEBUG)
		return
	}
	linkID := append([]byte{}, l.LinkID...)
	l.mu.Unlock()

	pub := identity.GetPublicKey()
	if len(pub) == 0 {
		Log("Identity has no public key material available", LOG_ERROR)
		return
	}

	signed := append(append([]byte{}, linkID...), pub...)
	sig, err := identity.Sign(signed)
	if err != nil {
		Log(fmt.Sprintf("Could not sign identify payload: %v", err), LOG_ERROR)
		return
	}

	payload := append(append([]byte{}, pub...), sig...)
	packet := NewPacket(
		l,
		payload,
		WithPacketContext(PacketCtxLinkIdentify),
		WithCreateReceipt(false),
	)
	if packet == nil {
		Log("Failed to create link identify packet", LOG_ERROR)
		return
	}
	if receipt := packet.Send(); receipt == nil && !packet.Sent {
		Log("Link identify packet could not be sent", LOG_WARNING)
	}
}

// ==== Watchdog ====

func (l *Link) startWatchdog() {
	l.watchdogOnce.Do(func() {
		go l.watchdogLoop()
	})
}

func (l *Link) watchdogLoop() {
	t := time.NewTicker(linkWatchdogMaxSleep)
	defer t.Stop()
	for {
		select {
		case <-l.watchdogStop:
			return
		case <-t.C:
			l.checkTimeouts()
		}
	}
}

func (l *Link) checkTimeouts() {
	l.mu.Lock()

	if l.Status == LinkClosed {
		l.mu.Unlock()
		return
	}

	// Python: in PENDING/HANDSHAKE, establishment timeout is based on request_time.
	if (l.Status == LinkPending || l.Status == LinkHandshake) && !l.requestTime.IsZero() && l.estTimeout > 0 {
		if time.Since(l.requestTime) >= l.estTimeout {
			Logf(LogVerbose, "Link establishment timed out: linkID=%x elapsed=%v timeout=%v", l.LinkID, time.Since(l.requestTime), l.estTimeout)
			l.mu.Unlock()
			go l.teardown(LinkTimeout)
			return
		}
		l.mu.Unlock()
		return
	}

	// Python: ACTIVE watchdog uses max(last_inbound, last_proof, activated_at).
	last := l.lastInbound
	if l.lastProof.After(last) {
		last = l.lastProof
	}
	if l.activatedAt.After(last) {
		last = l.activatedAt
	}
	if last.IsZero() {
		l.mu.Unlock()
		return
	}

	timeout := l.rttTimeout()

	if l.Keepalive <= 0 {
		l.Keepalive = linkKeepaliveMax
	}
	if l.StaleTime <= 0 {
		l.StaleTime = l.Keepalive * linkStaleFactor
	}

	if l.Status == LinkActive {
		// Python: only initiator sends keepalives (0xFF).
		if time.Since(last) >= l.Keepalive && l.Initiator && time.Since(l.lastKeepalive) >= l.Keepalive {
			go l.sendKeepalive()
		}
		// Python: mark STALE after stale_time, but only close after an additional timeout interval.
		if time.Since(last) >= l.StaleTime {
			l.Status = LinkStale
		}
	}

	// Python: once stale, close after an additional timeout interval.
	if l.Status == LinkStale && time.Since(last) >= l.StaleTime+timeout {
		Log("Link watchdog stale timeout reached, closing link", LOG_WARNING)
		l.mu.Unlock()
		go l.teardown(LinkTimeout)
		return
	}

	l.mu.Unlock()
}

func (l *Link) rttTimeout() time.Duration {
	rtt := l.RTT
	if rtt <= 0 {
		rtt = 500 * time.Millisecond
	}
	mult := l.KeepaliveTimeoutFactor
	if mult <= 0 {
		mult = linkKeepaliveFact
	}
	return time.Duration(mult*float64(rtt)) + linkStaleGrace
}

func (l *Link) Teardown() {
	if l == nil {
		return
	}
	l.mu.Lock()
	initiator := l.Initiator
	l.mu.Unlock()
	if initiator {
		l.teardown(LinkInitiatorClose)
		return
	}
	l.teardown(LinkDestinationClose)
}

func (l *Link) teardown(reason int) {
	l.teardownWithOptions(reason, true)
}

func (l *Link) teardownWithOptions(reason int, sendClose bool) {
	if l == nil {
		return
	}

	// Decide if we should send a teardown packet without holding the link lock,
	// since sending calls back into noteOutbound() which also locks.
	l.mu.Lock()
	if l.Status == LinkClosed {
		l.mu.Unlock()
		return
	}
	shouldSend := sendClose && l.Status != LinkPending
	l.mu.Unlock()

	if shouldSend {
		l.sendTeardownPacket()
	}

	l.mu.Lock()
	if l.Status == LinkClosed {
		l.mu.Unlock()
		return
	}
	l.TeardownReason = reason
	l.Status = LinkClosed
	if l.channel != nil {
		l.channel.Close()
		l.channel = nil
	}
	if l.watchdogStop != nil {
		close(l.watchdogStop)
		l.watchdogStop = nil
	}
	l.priv = nil
	l.sharedKey = nil
	l.derivedKey = nil
	l.token = nil
	l.incomingResources = nil
	l.outgoingResources = nil
	cb := l.callbacks.LinkClosed
	owner := l.owner
	l.mu.Unlock()

	if cb != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					Log("Link closed callback panic", LOG_ERROR)
				}
			}()
			cb(l)
		}()
	}
	if owner != nil {
		owner.removeLink(l)
	}
}

func (l *Link) String() string {
	if len(l.LinkID) == 0 {
		return "<nil-link>"
	}
	return fmt.Sprintf("<Link %s>", PrettyHexRep(l.LinkID))
}

func (l *Link) handleChannelPacket(packet *Packet) {
	l.mu.Lock()
	ch := l.channel
	l.mu.Unlock()

	if ch == nil {
		Log(fmt.Sprintf("%s received channel data without open channel", l), LOG_DEBUG)
		return
	}

	payload := packet.Data
	if pt, err := l.Decrypt(packet.Data); err == nil && len(pt) > 0 {
		payload = pt
	} else if err != nil {
		Log(fmt.Sprintf("%s failed to decrypt channel packet: %v", l, err), LOG_WARNING)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			Log(fmt.Sprintf("%s panic while delivering channel data: %v", l, r), LOG_ERROR)
		}
	}()
	ch.Receive(payload)
}

func (l *Link) handleLinkIdentify(packet *Packet) {
	payload := packet.Data
	if pt, err := l.Decrypt(packet.Data); err == nil && len(pt) > 0 {
		payload = pt
	} else if err != nil {
		Log(fmt.Sprintf("%s failed to decrypt identify packet: %v", l, err), LOG_WARNING)
		return
	}

	minLen := identityPublicKeyLength + ed25519.SignatureSize
	if len(payload) < minLen {
		Log(fmt.Sprintf("%s received malformed identify payload (%d bytes)", l, len(payload)), LOG_DEBUG)
		return
	}

	pubBytes := append([]byte{}, payload[:identityPublicKeyLength]...)
	signature := append([]byte{}, payload[identityPublicKeyLength:]...)

	remote := &Identity{}
	if err := remote.LoadPublicKey(pubBytes); err != nil {
		Log(fmt.Sprintf("Could not load remote identity from identify payload: %v", err), LOG_ERROR)
		return
	}

	signed := append(append([]byte{}, l.LinkID...), pubBytes...)
	if !remote.Validate(signature, signed) {
		Log(fmt.Sprintf("%s received invalid identity signature", l), LOG_WARNING)
		return
	}

	l.mu.Lock()
	if l.remoteIdentity != nil {
		l.mu.Unlock()
		return
	}
	l.remoteIdentity = remote
	cb := l.callbacks.RemoteIdentified
	l.mu.Unlock()

	if cb != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					Log(fmt.Sprintf("Remote identified callback panic on %s: %v", l, r), LOG_ERROR)
				}
			}()
			cb(l, remote)
		}()
	}
	Log(fmt.Sprintf("%s identified remote peer as %s", l, remote), LOG_DEBUG)
}

func (l *Link) handleRequestPacket(packet *Packet) {
	payload, err := l.Decrypt(packet.Data)
	if err != nil || len(payload) == 0 {
		Log(fmt.Sprintf("%s failed to decrypt request packet: %v", l, err), LOG_WARNING)
		return
	}
	var unpacked []any
	if err := umsgpack.Unpackb(payload, &unpacked); err != nil {
		Log(fmt.Sprintf("%s received malformed request payload: %v", l, err), LOG_WARNING)
		return
	}
	// Dispatch the request handler in a goroutine so that slow response
	// generators (e.g. database queries) do not block the transport's
	// packet processing. This is safe because Inbound/Outbound use an
	// RWMutex that allows concurrent read-side callers.
	go func() {
		if l.handleRequestAndReportAllowed(packet.GetTruncatedHash(), unpacked, packet) {
			l.ProvePacket(packet)
		}
	}()
}

func (l *Link) handleRequestAndReportAllowed(requestID []byte, unpacked []any, packet *Packet) bool {
	if len(unpacked) < 3 {
		Log("Malformed request payload received, ignoring", LOG_WARNING)
		return false
	}

	requestedAt := timeFromAny(unpacked[0])
	pathHash := bytesFromAny(unpacked[1])
	if len(pathHash) == 0 {
		Log("Request without path hash received, ignoring", LOG_WARNING)
		return false
	}
	requestData := unpacked[2]

	l.mu.Lock()
	dest := l.destination
	remoteID := l.remoteIdentity
	linkID := append([]byte{}, l.LinkID...)
	l.mu.Unlock()

	if dest == nil {
		return false
	}

	handler, ok := dest.requestHandlers[string(pathHash)]
	if !ok || handler == nil {
		return false
	}

	resp, allowed := dest.DispatchRequest(handler.Path, requestData, requestID, linkID, remoteID, requestedAt)
	if !allowed {
		return false
	}
	if resp == nil {
		// Allowed request with nil response still counts as delivered.
		return true
	}

	l.sendResponsePayload(handler, requestID, resp)
	return true
}

func (l *Link) handleResponsePacket(packet *Packet) {
	payload, err := l.Decrypt(packet.Data)
	if err != nil || len(payload) == 0 {
		Log(fmt.Sprintf("%s failed to decrypt response packet: %v", l, err), LOG_WARNING)
		return
	}
	var unpacked []any
	if err := umsgpack.Unpackb(payload, &unpacked); err != nil || len(unpacked) < 2 {
		Log(fmt.Sprintf("%s received malformed response payload: %v", l, err), LOG_WARNING)
		return
	}
	requestID := bytesFromAny(unpacked[0])
	response := unpacked[1]
	l.handleResponse(requestID, response, len(payload))
}

func (l *Link) handleResourceAdvertisement(packet *Packet) {
	plaintext, err := l.Decrypt(packet.Data)
	if err != nil || len(plaintext) == 0 {
		Log(fmt.Sprintf("%s failed to decrypt resource advertisement: %v", l, err), LOG_WARNING)
		return
	}
	packet.Plaintext = plaintext
	packet.Link = l

	adv, err := ResourceAdvertisementUnpack(plaintext)
	if err != nil {
		Log(fmt.Sprintf("Could not unpack resource advertisement: %v", err), LOG_DEBUG)
		return
	}
	if len(adv.H) == 0 || len(adv.R) == 0 || len(adv.M) == 0 || adv.N <= 0 {
		Log(fmt.Sprintf("Malformed resource advertisement on %s (H=%d R=%d M=%d N=%d)", l, len(adv.H), len(adv.R), len(adv.M), adv.N), LOG_ERROR)
		return
	}
	adv.Link = l

	switch {
	case adv.IsRequest():
		_, err := ResourceAccept(
			packet,
			func(res *Resource) {
				l.ResourceConcluded(res)
				l.requestResourceConcluded(res)
			},
			nil,
			adv.Q,
		)
		if err != nil {
			Log(fmt.Sprintf("Could not accept request resource: %v", err), LOG_DEBUG)
		}
	case adv.IsResponse():
		pending := l.pendingRequestByID(adv.Q)
		if pending == nil {
			ResourceReject(packet)
			return
		}
		pending.noteResponseAdvertisement(adv)
		res, err := ResourceAccept(
			packet,
			func(res *Resource) {
				l.ResourceConcluded(res)
				l.responseResourceConcluded(res)
			},
			func(res *Resource) {
				l.responseResourceProgress(res)
			},
			adv.Q,
		)
		if err != nil || res == nil {
			Log(fmt.Sprintf("Could not accept response resource: %v", err), LOG_DEBUG)
			return
		}
		if res != nil {
			pending.responseResourceProgress(res)
		}
	default:
		switch l.resourceStrategy {
		case LinkAcceptNone:
			ResourceReject(packet)
		case LinkAcceptApp:
			if l.callbacks.Resource == nil {
				ResourceReject(packet)
				return
			}
			func() {
				defer func() {
					if rec := recover(); rec != nil {
						Log(fmt.Sprintf("Resource callback panic on %s: %v", l, rec), LOG_ERROR)
					}
				}()
				if l.callbacks.Resource(adv) {
					ResourceAccept(packet, l.ResourceConcluded, nil, nil)
				} else {
					ResourceReject(packet)
				}
			}()
		case LinkAcceptAll:
			ResourceAccept(packet, l.ResourceConcluded, nil, nil)
		}
	}
}

func (l *Link) handleResourceRequestPacket(packet *Packet) {
	payload, err := l.Decrypt(packet.Data)
	if err != nil || len(payload) == 0 {
		Log(fmt.Sprintf("Failed to decrypt resource request on %s: %v", l, err), LOG_DEBUG)
		return
	}
	// Resource hashes are full SHA256 in the Python reference, but keep a truncated
	// fallback for compatibility with any legacy senders.
	hashLens := []int{sha256Bits / 8, ReticulumTruncatedHashLength / 8}
	offset := 1
	if len(payload) < offset+1 {
		return
	}
	if payload[0] == HashmapExhausted {
		offset += MapHashLen
	}
	for _, hashLen := range hashLens {
		if hashLen <= 0 {
			continue
		}
		if len(payload) < offset+hashLen {
			continue
		}
		hashStart := offset
		hashEnd := hashStart + hashLen
		if hashEnd > len(payload) {
			continue
		}
		hash := payload[hashStart:hashEnd]
		resource := l.findOutgoingResource(hash)
		if resource == nil {
			continue
		}
		resource.Request(payload)
		return
	}

	// Debug aid: resource request could not be matched.
	func() {
		l.mu.Lock()
		out := make([]*Resource, len(l.outgoingResources))
		copy(out, l.outgoingResources)
		l.mu.Unlock()
		if len(out) == 0 {
			return
		}
		Log(fmt.Sprintf("Resource request on %s did not match any outgoing resource (len=%d)", l, len(payload)), LOG_DEBUG)
	}()

	// Fallback: some malformed/legacy senders may not include a resource hash at the
	// expected offset. Since only one outgoing resource is allowed at a time, try to
	// resolve by matching any requested map hash against the advertised part map.
	l.mu.Lock()
	outgoing := make([]*Resource, len(l.outgoingResources))
	copy(outgoing, l.outgoingResources)
	l.mu.Unlock()
	for _, hashLen := range hashLens {
		if hashLen <= 0 || len(payload) < offset+hashLen {
			continue
		}
		requested := payload[offset+hashLen:]
		for i := 0; i+MapHashLen <= len(requested); i += MapHashLen {
			key := string(requested[i : i+MapHashLen])
			for _, res := range outgoing {
				if res == nil || res.outgoingPartByMapHash == nil {
					continue
				}
				if _, ok := res.outgoingPartByMapHash[key]; ok {
					res.Request(payload)
					return
				}
			}
		}
	}
}

func (l *Link) handleResourceHashmapUpdate(packet *Packet) {
	payload, err := l.Decrypt(packet.Data)
	if err != nil || len(payload) == 0 {
		return
	}
	hashLen := sha256Bits / 8
	if len(payload) < hashLen {
		return
	}
	resource := l.findIncomingResource(payload[:hashLen])
	if resource == nil {
		return
	}
	resource.HashmapUpdatePacket(payload)
}

func (l *Link) handleResourceCancelIncoming(packet *Packet) {
	payload, err := l.Decrypt(packet.Data)
	if err != nil || len(payload) == 0 {
		return
	}
	hashLen := sha256Bits / 8
	if len(payload) < hashLen {
		return
	}
	if res := l.findIncomingResource(payload[:hashLen]); res != nil {
		res.Cancel()
	}
}

func (l *Link) handleResourceRejection(packet *Packet) {
	payload, err := l.Decrypt(packet.Data)
	if err != nil || len(payload) == 0 {
		return
	}
	hashLen := sha256Bits / 8
	if len(payload) < hashLen {
		return
	}
	if res := l.findOutgoingResource(payload[:hashLen]); res != nil {
		res.rejected()
	}
}

func (l *Link) handleResourceData(packet *Packet) {
	l.mu.Lock()
	incoming := make([]*Resource, len(l.incomingResources))
	copy(incoming, l.incomingResources)
	l.mu.Unlock()
	for _, res := range incoming {
		if res != nil {
			res.ReceivePart(packet)
		}
	}
}

func (l *Link) handleResourceProof(packet *Packet) {
	hashLen := sha256Bits / 8
	if len(packet.Data) < hashLen {
		return
	}
	res := l.findOutgoingResource(packet.Data[:hashLen])
	if res == nil {
		return
	}
	res.ValidateProof(packet.Data)
}

func (l *Link) handleLinkClose(packet *Packet) {
	payload, err := l.Decrypt(packet.Data)
	if err != nil || len(payload) == 0 {
		return
	}
	l.mu.Lock()
	linkID := append([]byte(nil), l.LinkID...)
	initiator := l.Initiator
	l.mu.Unlock()
	if !bytes.Equal(payload, linkID) {
		return
	}
	// Mirror Python: initiator sees destination close, destination sees initiator close.
	if initiator {
		l.teardownWithOptions(LinkDestinationClose, false)
	} else {
		l.teardownWithOptions(LinkInitiatorClose, false)
	}
}

func (l *Link) sendTeardownPacket() {
	if l == nil || len(l.LinkID) == 0 {
		return
	}
	p := NewPacket(l, append([]byte(nil), l.LinkID...), WithPacketContext(PacketCtxLinkClose), WithoutReceipt())
	if p == nil {
		return
	}
	_ = p.Send()
	l.noteOutbound(PacketCtxLinkClose, len(l.LinkID))
}

func (l *Link) sendChannelPacket(raw []byte) *Packet {
	l.mu.Lock()
	dest := l.destination
	status := l.Status
	l.mu.Unlock()

	if dest == nil {
		Log("Channel send attempted on link without destination", LOG_WARNING)
		return nil
	}

	l.noteOutbound(PacketCtxChannel, len(raw))
	packet := NewPacket(
		dest,
		raw,
		WithPacketContext(PacketCtxChannel),
	)
	packet.Link = l
	if status != LinkActive {
		Log("Sending channel data on non-active link", LOG_DEBUG)
	}

	if receipt := packet.Send(); receipt != nil {
		packet.Receipt = receipt
	} else if packet.CreateReceipt && packet.Receipt == nil {
		packet.Receipt = NewPacketReceipt(packet)
	}
	return packet
}

// ==== helper functions ====

func bytesFromAny(v any) []byte {
	switch val := v.(type) {
	case []byte:
		return append([]byte(nil), val...)
	default:
		return nil
	}
}

func floatFromAny(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case uint64:
		return float64(val)
	case uint32:
		return float64(val)
	default:
		return 0
	}
}

func timeFromAny(v any) time.Time {
	f := floatFromAny(v)
	if f == 0 {
		return time.Unix(0, 0)
	}
	return time.Unix(0, int64(f*float64(time.Second)))
}

func defaultLinkMTU() int {
	phyParamsMu.RLock()
	defer phyParamsMu.RUnlock()
	if phyParamsSnapshot.PhysicalLayerMTU > 0 {
		return phyParamsSnapshot.PhysicalLayerMTU
	}
	return MTU
}

func registerLinkWithTransport(l *Link) {
	if l == nil {
		return
	}
	linkMu.Lock()
	defer linkMu.Unlock()
	if l.Initiator {
		if !linkSliceContains(PendingLinks, l) {
			PendingLinks = append(PendingLinks, l)
			Logf(LogDebug, "registerLinkWithTransport: added initiator link %x to PendingLinks (now %d)", l.LinkID, len(PendingLinks))
		}
	} else {
		if !linkSliceContains(ActiveLinks, l) {
			ActiveLinks = append(ActiveLinks, l)
		}
	}
}

func activateLinkInTransport(l *Link) {
	if l == nil {
		return
	}
	linkMu.Lock()
	defer linkMu.Unlock()
	for i, existing := range PendingLinks {
		if existing == l {
			PendingLinks = append(PendingLinks[:i], PendingLinks[i+1:]...)
			break
		}
	}
	if !linkSliceContains(ActiveLinks, l) {
		ActiveLinks = append(ActiveLinks, l)
	}
}

func linkSliceContains(list []*Link, target *Link) bool {
	for _, l := range list {
		if l == target {
			return true
		}
	}
	return false
}

func linkMTUFromLRPacket(packet *Packet) (int, bool) {
	if packet == nil {
		return 0, false
	}
	if len(packet.Data) < linkEcPubSize+linkSignalSize {
		return 0, false
	}
	offset := linkEcPubSize
	mtuBytes := packet.Data[offset : offset+linkSignalSize]
	mtu := (int(mtuBytes[0])<<16 | int(mtuBytes[1])<<8 | int(mtuBytes[2])) & linkMTUMask
	return mtu, true
}

func linkMTUFromProofPacket(packet *Packet) (int, bool) {
	if packet == nil {
		return 0, false
	}
	expected := ed25519.SignatureSize + linkEcPubSize/2 + linkSignalSize
	if len(packet.Data) != expected {
		return 0, false
	}
	offset := ed25519.SignatureSize + linkEcPubSize/2
	mtuBytes := packet.Data[offset : offset+linkSignalSize]
	mtu := (int(mtuBytes[0])<<16 | int(mtuBytes[1])<<8 | int(mtuBytes[2])) & linkMTUMask
	return mtu, true
}

func linkSignallingBytes(mtu int, mode int) ([]byte, error) {
	if !containsInt(mode, linkEnabledModes) {
		return nil, fmt.Errorf("link mode %d disabled", mode)
	}
	if mtu <= 0 {
		mtu = defaultLinkMTU()
	}
	signallingValue := (mtu & linkMTUMask) | (((mode << 5) & linkModeMask) << 16)
	return []byte{
		byte((signallingValue >> 16) & 0xff),
		byte((signallingValue >> 8) & 0xff),
		byte(signallingValue & 0xff),
	}, nil
}

func linkModeFromLRPacket(packet *Packet) int {
	if packet == nil {
		return linkDefaultMode
	}
	if len(packet.Data) <= linkEcPubSize {
		return linkDefaultMode
	}
	modeBits := (packet.Data[linkEcPubSize] & linkModeMask) >> 5
	if modeBits == 0 {
		return linkDefaultMode
	}
	return int(modeBits)
}

func linkModeFromProofPacket(packet *Packet) int {
	if packet == nil {
		return linkDefaultMode
	}
	expected := ed25519.SignatureSize + linkEcPubSize/2 + linkSignalSize
	if len(packet.Data) != expected {
		return linkDefaultMode
	}
	offset := ed25519.SignatureSize + linkEcPubSize/2
	mtuBytes := packet.Data[offset : offset+linkSignalSize]
	modeBits := (mtuBytes[0] & linkModeMask) >> 5
	if modeBits == 0 {
		return linkDefaultMode
	}
	return int(modeBits)
}
