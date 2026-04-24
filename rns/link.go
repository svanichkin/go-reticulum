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

func linkIDFromLinkRequestPacket(p *Packet) []byte {
	hashable := p.getHashablePart()
	if len(p.Data) > linkEcPubSize {
		diff := len(p.Data) - linkEcPubSize
		hashable = hashable[:len(hashable)-diff]
	}
	return TruncatedHash(hashable)
}

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
	watchdogLock   bool
	watchdogOnce   sync.Once
	watchdogStop   chan struct{}
	remoteIdentity *Identity

	trackPhyStats bool
	rssi          *float64
	snr           *float64
	q             *float64
}

const identityPublicKeyLength = x25519KeyLen + ed25519.PublicKeySize

// ==== Link creation ====

func NewLink(destination *Destination, owner *Destination, mode int, establishedCB, closedCB func(*Link)) (*Link, error) {
	if mode < 0 {
		mode = linkDefaultMode
	}
	if destination != nil && destination.Type != DestinationSINGLE {
		return nil, &DestinationTypeError{Message: "Links can only be established to the \"single\" destination type"}
	}

	l := &Link{
		Mode:      mode,
		Status:    LinkPending,
		Initiator: destination != nil,
		MTU: func() int {
			mtu := MTU
			if phyParamsSnapshot.PhysicalLayerMTU > 0 {
				mtu = phyParamsSnapshot.PhysicalLayerMTU
			}
			return mtu
		}(),
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
		mtu := MTU
		if desc, ok := linkModeDescriptions[l.Mode]; ok {
			Logf(LOG_DEBUG, "Establishing link with mode %s", desc)
		}
		if destination != nil {
			l.expectedHops = HopsTo(destination.Hash)
			hopsForTimeout := l.expectedHops
			if hopsForTimeout < 1 {
				hopsForTimeout = 1
			}
			// Python: get_first_hop_timeout + ESTABLISHMENT_TIMEOUT_PER_HOP*hops
			baseTimeout := FirstHopTimeout(destination.Hash)
			if Owner != nil {
				baseTimeout = time.Duration(Owner.GetFirstHopTimeout(destination.Hash) * float64(time.Second))
			}
			l.estTimeout = baseTimeout + linkDefaultPerHop*time.Duration(hopsForTimeout)
		}
		if LinkMTUDiscovery() && destination != nil {
			if nh := NextHopInterfaceHWMTU(destination.Hash); nh != nil {
				mtu = *nh
				Logf(LOG_DEBUG, "Signalling link MTU of %s for link", PrettySize(float64(mtu)))
			}
		}
		signalling, err := linkSignallingBytes(mtu, l.Mode)
		if err != nil {
			return nil, err
		}
		payload := make([]byte, 0, len(l.pub)+len(l.sigPub)+len(signalling))
		payload = append(payload, l.pub...)
		payload = append(payload, l.sigPub...)
		payload = append(payload, signalling...)
		l.requestData = payload

		packet := NewPacket(
			destination,
			payload,
			PacketTypeLinkRequest,
			PacketCtxNone,
			Broadcast,
			HeaderType1,
			nil,
			nil,
			true,
			FlagUnset,
		)
		if packet == nil {
			return nil, errors.New("could not build link request packet")
		}
		if err := packet.Pack(); err != nil {
			return nil, err
		}

		l.EstablishmentCost += len(packet.Raw)
		l.setLinkID(packet)
		// Python parity: register the outgoing link after set_link_id() and before
		// the request is sent, so the pending entry mirrors Link.__init__ exactly.
		registerLink(l)
		l.requestTime = time.Now()
		l.startWatchdog()
		l.packet = packet

		receipt := packet.Send()
		now := time.Now()
		l.mu.Lock()
		l.lastOutbound = now
		l.lastData = now
		l.mu.Unlock()
		Logf(LOG_DEBUG, "Link request %s sent to %v", PrettyHexRep(l.LinkID), destination)
		Logf(LOG_EXTREME, "Establishment timeout is %s for link request %s", PrettyTime(l.estTimeout.Seconds(), true, true), PrettyHexRep(l.LinkID))
		if receipt == nil && !packet.Sent {
			return nil, errors.New("link request send failed")
		}
		l.hadOutbound(false)
	}
	return l, nil
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

func (l *Link) GetRemoteIdentity() *Identity {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.remoteIdentity
}

func (l *Link) GetAge() *float64 {
	if l == nil || l.activatedAt.IsZero() {
		return nil
	}
	v := time.Since(l.activatedAt).Seconds()
	return &v
}

func (l *Link) hadOutbound(isKeepalive bool) {
	now := time.Now()
	l.mu.Lock()
	l.lastOutbound = now
	if isKeepalive {
		l.lastKeepalive = now
	} else {
		l.lastData = now
	}
	l.mu.Unlock()
}

func (l *Link) NoInboundFor() float64 {
	if l == nil {
		return 0
	}
	l.mu.Lock()
	activatedAt := l.activatedAt
	lastInbound := l.lastInbound
	l.mu.Unlock()
	if activatedAt.After(lastInbound) {
		lastInbound = activatedAt
	}
	return time.Since(lastInbound).Seconds()
}

func (l *Link) NoOutboundFor() float64 {
	if l == nil {
		return 0
	}
	l.mu.Lock()
	lastOutbound := l.lastOutbound
	l.mu.Unlock()
	return time.Since(lastOutbound).Seconds()
}

func (l *Link) NoDataFor() float64 {
	if l == nil {
		return 0
	}
	l.mu.Lock()
	lastData := l.lastData
	l.mu.Unlock()
	return time.Since(lastData).Seconds()
}

func (l *Link) InactiveFor() float64 {
	return min(l.NoInboundFor(), l.NoOutboundFor())
}

func (l *Link) GetEstablishmentRate() *float64 {
	if l == nil || l.EstablishmentRate <= 0 {
		return nil
	}
	v := l.EstablishmentRate * 8
	return &v
}

func (l *Link) GetMTU() *int {
	if l == nil || l.Status != LinkActive {
		return nil
	}
	v := l.MTU
	return &v
}

func (l *Link) GetMDU() *int {
	if l == nil || l.Status != LinkActive {
		return nil
	}
	v := l.MDU
	return &v
}

func (l *Link) GetExpectedRate() *float64 {
	if l == nil || l.Status != LinkActive {
		return nil
	}
	v := l.ExpectedRate
	return &v
}

func (l *Link) GetMode() int {
	return l.Mode
}

func (l *Link) Request(
	path string,
	data any,
	responseCb func(*RequestReceipt),
	failedCb func(*RequestReceipt),
	progressCb func(*RequestReceipt),
	timeout float64,
) any {
	if timeout <= 0 {
		timeout = l.RTT.Seconds()*l.TrafficTimeoutFactor + 1.125*ResponseMaxGraceTime.Seconds()
	}

	pathHash := TruncatedHash([]byte(path))
	unpacked := []any{float64(time.Now().UnixNano()) / 1e9, pathHash, data}
	packedRequest, err := umsgpack.Packb(unpacked)
	if err != nil {
		Log(fmt.Sprintf("Could not pack request payload for %s: %v", path, err), LOG_ERROR)
		return false
	}

	if len(packedRequest) <= l.MDU {
		packet := NewPacket(
			l,
			packedRequest,
			PacketTypeData,
			PacketCtxRequest,
			Broadcast,
			HeaderType1,
			nil,
			nil,
			true,
			FlagUnset,
		)
		if packet == nil {
			Log("Failed to build request packet", LOG_ERROR)
			return false
		}
		packetReceipt := packet.Send()
		if packetReceipt == nil {
			if !packet.Sent {
				Log("Request packet could not be sent", LOG_WARNING)
				return false
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
	Log(fmt.Sprintf("Sending request %s as resource.", PrettyHexRep(requestID)), LOG_DEBUG)
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
		return false
	}
	if res == nil {
		Log("NewResource returned nil for request", LOG_ERROR)
		return false
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

func (l *Link) SetResourceStrategy(strategy int) {
	if strategy != LinkAcceptNone && strategy != LinkAcceptApp && strategy != LinkAcceptAll {
		panic("unsupported resource strategy")
	}
	l.mu.Lock()
	l.resourceStrategy = strategy
	l.mu.Unlock()
}

// ==== Resource handling ====

func (l *Link) RegisterIncomingResource(res *Resource) {
	l.mu.Lock()
	l.incomingResources = append(l.incomingResources, res)
	l.mu.Unlock()
}

func (l *Link) RegisterOutgoingResource(res *Resource) {
	l.mu.Lock()
	l.outgoingResources = append(l.outgoingResources, res)
	l.mu.Unlock()
}

func (l *Link) CancelIncomingResource(res *Resource) {
	l.mu.Lock()
	found := false
	for i, candidate := range l.incomingResources {
		if candidate == res {
			l.incomingResources = append(l.incomingResources[:i], l.incomingResources[i+1:]...)
			found = true
			break
		}
	}
	l.mu.Unlock()
	if !found {
		Log("Attempt to cancel a non-existing incoming resource", LOG_ERROR)
	}
}

func (l *Link) CancelOutgoingResource(res *Resource) {
	l.mu.Lock()
	found := false
	for i, candidate := range l.outgoingResources {
		if candidate == res {
			l.outgoingResources = append(l.outgoingResources[:i], l.outgoingResources[i+1:]...)
			found = true
			break
		}
	}
	l.mu.Unlock()
	if !found {
		Log("Attempt to cancel a non-existing outgoing resource", LOG_ERROR)
	}
}

func (l *Link) HasIncomingResource(res *Resource) bool {
	if res == nil {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, incoming := range l.incomingResources {
		if incoming != nil && bytes.Equal(incoming.hash, res.hash) {
			return true
		}
	}
	return false
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
	l.mu.Lock()
	removedIncoming := false
	for i, candidate := range l.incomingResources {
		if candidate == res {
			l.incomingResources = append(l.incomingResources[:i], l.incomingResources[i+1:]...)
			removedIncoming = true
			break
		}
	}
	removedOutgoing := false
	for i, candidate := range l.outgoingResources {
		if candidate == res {
			l.outgoingResources = append(l.outgoingResources[:i], l.outgoingResources[i+1:]...)
			removedOutgoing = true
			break
		}
	}
	if removedIncoming {
		l.lastResourceWindow = res.window
		l.lastResourceEIFR = res.eifr
	}
	if removedIncoming || removedOutgoing {
		d := time.Since(res.startedTransferring).Seconds()
		if d < 0.0001 {
			d = 0.0001
		}
		l.ExpectedRate = float64(res.size*8) / d
	}
	l.mu.Unlock()
}

func (l *Link) handleResponse(requestID []byte, response any, responseSize int, responseTransferSize int, metadata any) {
	if l == nil || l.Status != LinkActive {
		return
	}
	if len(requestID) == 0 {
		return
	}
	l.mu.Lock()
	var pending *RequestReceipt
	for _, candidate := range l.pendingRequests {
		if candidate != nil && bytes.Equal(candidate.RequestID(), requestID) {
			pending = candidate
			break
		}
	}
	l.mu.Unlock()
	if pending == nil {
		return
	}
	pending.responseReceived(response, metadata, responseSize, responseTransferSize)
}

func (l *Link) requestResourceConcluded(res *Resource) {
	if res == nil {
		return
	}
	if res.Status != ResourceComplete {
		Log(fmt.Sprintf("Incoming request resource failed with status: %x", []byte{byte(res.Status)}), LOG_DEBUG)
		return
	}
	data, err := os.ReadFile(res.DataFile)
	if err != nil {
		Log(fmt.Sprintf("Could not read completed request resource: %v", err), LOG_ERROR)
		return
	}
	var unpacked []any
	if err := umsgpack.Unpackb(data, &unpacked); err != nil {
		Log(fmt.Sprintf("Could not decode request resource payload: %v", err), LOG_ERROR)
		return
	}
	l.handleRequest(append([]byte(nil), res.requestID...), unpacked, nil)
}

func (l *Link) responseResourceConcluded(res *Resource) {
	if res == nil {
		return
	}
	reqID := append([]byte(nil), res.requestID...)
	l.mu.Lock()
	var pending *RequestReceipt
	for _, candidate := range l.pendingRequests {
		if candidate != nil && bytes.Equal(candidate.RequestID(), reqID) {
			pending = candidate
			break
		}
	}
	l.mu.Unlock()
	if res.Status != ResourceComplete {
		Log(fmt.Sprintf("Incoming response resource failed with status: %x", []byte{byte(res.Status)}), LOG_DEBUG)
		if pending != nil {
			pending.requestTimedOut()
		}
		return
	}
	if pending == nil {
		return
	}
	if res.hasMetadata {
		pending.responseReceived(res, res.Metadata, res.GetDataSize(), res.size)
		return
	}
	data, err := os.ReadFile(res.DataFile)
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
	response := unpacked[1]
	pending.responseReceived(response, nil, res.GetDataSize(), res.size)
}

func (l *Link) responseResourceProgress(res *Resource) {
	if res == nil || res.requestID == nil {
		return
	}
	l.mu.Lock()
	var pending *RequestReceipt
	for _, candidate := range l.pendingRequests {
		if candidate != nil && bytes.Equal(candidate.RequestID(), res.requestID) {
			pending = candidate
			break
		}
	}
	l.mu.Unlock()
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

	link, err := NewLink(nil, owner, linkDefaultMode, nil, nil)
	if err != nil {
		Log(fmt.Sprintf("Validating link request failed: %v", err), LOG_VERBOSE)
		return nil
	}
	if err := link.loadPeer(peerPub, peerSig); err != nil {
		Log(fmt.Sprintf("Validating link request failed: %v", err), LOG_VERBOSE)
		return nil
	}

	link.setLinkID(packet)

	if len(data) == linkEcPubSize+linkSignalSize {
		Log("Link request includes MTU signalling", LOG_DEBUG)
		mtu := MTU
		if mtus, ok := linkMTUFromLRPacket(packet); ok {
			mtu = mtus
		}
		link.MTU = mtu
	}

	link.Mode = linkModeFromLRPacket(packet)

	link.updateMDU()
	if desc, ok := linkModeDescriptions[link.Mode]; ok {
		Log(fmt.Sprintf("Incoming link request with mode %s", desc), LOG_DEBUG)
	}
	if packet != nil {
		link.attachedInterface = packet.ReceivingInterface
		link.destination = packet.Destination
		link.EstablishmentCost += len(packet.Raw)
	}

	// Python: establishment_timeout = PER_HOP*max(1, packet.hops) + KEEPALIVE
	hops := 1
	if packet != nil && packet.Hops > 0 {
		hops = int(packet.Hops)
	}
	link.estTimeout = linkDefaultPerHop*time.Duration(hops) + linkKeepaliveMax

	Logf(LogDebug, "Validating link request %s", PrettyHexRep(link.LinkID))
	Logf(LogExtreme, "Link MTU configured to %s", PrettySize(float64(link.MTU)))
	Logf(LogExtreme, "Establishment timeout is %s for incoming link request %s", PrettyTime(link.estTimeout.Seconds(), true, true), PrettyHexRep(link.LinkID))

	if err := link.Handshake(); err != nil {
		Log(fmt.Sprintf("Handshake failed: %v", err), LOG_ERROR)
		return nil
	}
	link.prove()
	link.requestTime = time.Now()
	registerLink(link)
	link.lastInbound = time.Now()
	if packet != nil {
		rssi := packet.GetRSSI()
		snr := packet.GetSNR()
		q := packet.GetQ()
		link.mu.Lock()
		if rssi != nil {
			v := *rssi
			link.rssi = &v
		}
		if snr != nil {
			v := *snr
			link.snr = &v
		}
		if q != nil {
			v := *q
			link.q = &v
		}
		link.mu.Unlock()
	}

	link.startWatchdog()
	Log(fmt.Sprintf("Incoming link request %s accepted on %v", link, link.attachedInterface), LOG_DEBUG)
	return link
}

// Receive processes an inbound packet on this link, updating timers and
// handing channel payloads to the Channel implementation.
func (l *Link) Receive(packet *Packet) {
	if packet == nil {
		return
	}
	l.mu.Lock()
	l.watchdogLock = true
	l.mu.Unlock()
	defer func() {
		l.mu.Lock()
		l.watchdogLock = false
		l.mu.Unlock()
	}()
	if l.Status == LinkClosed {
		return
	}
	if l.Initiator && packet.Context == PacketCtxKeepalive && len(packet.Data) == 1 && packet.Data[0] == 0xFF {
		return
	}
	if packet.ReceivingInterface != l.attachedInterface {
		Log(fmt.Sprintf("Link-associated packet received on unexpected interface %v instead of %v! Someone might be trying to manipulate your communication!", packet.ReceivingInterface, l.attachedInterface), LOG_ERROR)
		return
	}
	now := time.Now()
	activate := false
	l.mu.Lock()
	l.lastInbound = now
	if packet.Context != PacketCtxKeepalive {
		l.lastData = now
	}
	l.rx++
	if len(packet.Data) > 0 {
		l.rxBytes += uint64(len(packet.Data))
	}
	if l.Status == LinkStale {
		activate = true
		l.Status = LinkActive
	}
	l.mu.Unlock()
	if activate {
		activateLink(l)
	}

	// Link packet proofs: update matching receipts (Python Transport.receipts parity).
	if packet.PacketType == PacketTypeProof && packet.Context == PacketCtxNone {
		// Proof payload is explicit: packet_hash || signature
		receiptsMu.Lock()
		defer receiptsMu.Unlock()
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
			return
		}
		packet.RatchetID = append([]byte(nil), l.LinkID...)
		l.updatePhyStats(packet)
		if cb := l.callbacks.Packet; cb != nil {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						Log(fmt.Sprintf("Packet callback panic on %s: %v", l, r), LOG_ERROR)
					}
				}()
				cb(plaintext, packet)
			}()
		}
		if l.destination != nil {
			switch l.destination.proofStrategy {
			case DestinationPROVE_ALL:
				l.ProvePacket(packet)
			case DestinationPROVE_APP:
				if l.destination.callbacks.ProofRequested != nil {
					func() {
						defer func() {
							if r := recover(); r != nil {
								Log(fmt.Sprintf("Error while executing proof request callback from %s. The contained exception was: %v", l, r), LOG_ERROR)
							}
						}()
						if l.destination.callbacks.ProofRequested(packet) {
							l.ProvePacket(packet)
						}
					}()
				}
			}
		}
		return
	}

	switch packet.Context {
	case PacketCtxChannel:
		l.mu.Lock()
		ch := l.channel
		l.mu.Unlock()
		if ch == nil {
			Log("Channel data received without open channel", LOG_DEBUG)
			return
		}
		l.ProvePacket(packet)
		payload := packet.Data
		if pt, err := l.Decrypt(packet.Data); err == nil && len(pt) > 0 {
			payload = pt
		} else if err != nil {
			return
		}
		defer func() {
			if r := recover(); r != nil {
				Log(fmt.Sprintf("%s panic while delivering channel data: %v", l, r), LOG_ERROR)
			}
		}()
		ch.Receive(payload)
	case PacketCtxKeepalive:
		// Python: the destination replies to initiator keepalive 0xFF with 0xFE.
		if !l.Initiator && bytes.Equal(packet.Data, []byte{0xFF}) {
			go func() {
				p := NewPacket(
					l,
					[]byte{0xFE},
					PacketTypeData,
					PacketCtxKeepalive,
					Broadcast,
					HeaderType1,
					nil,
					nil,
					false,
					FlagUnset,
				)
				if p == nil {
					return
				}
				_ = p.Send()
				now := time.Now()
				l.mu.Lock()
				l.lastOutbound = now
				l.lastKeepalive = now
				l.mu.Unlock()
			}()
		}
	case PacketCtxLinkIdentify:
		payload := packet.Data
		if pt, err := l.Decrypt(packet.Data); err == nil && len(pt) > 0 {
			payload = pt
		} else if err != nil {
			return
		}
		if len(payload) < identityPublicKeyLength+ed25519.SignatureSize {
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
						Log(fmt.Sprintf("Error while executing remote identified callback from %s. The contained exception was: %v", l, r), LOG_ERROR)
					}
				}()
				cb(l, remote)
			}()
		}
		l.updatePhyStats(packet)
	case PacketCtxRequest:
		payload, err := l.Decrypt(packet.Data)
		if err != nil || len(payload) == 0 {
			return
		}
		var unpacked []any
		if err := umsgpack.Unpackb(payload, &unpacked); err != nil {
			Log(fmt.Sprintf("%s received malformed request payload: %v", l, err), LOG_WARNING)
			return
		}
		l.updatePhyStats(packet)
		go l.handleRequest(packet.GetTruncatedHash(), unpacked, packet)
	case PacketCtxLRProof:
		l.validateProof(packet)
	case PacketCtxLRRTT:
		if !l.Initiator {
			l.rttPacket(packet)
			l.updatePhyStats(packet)
		}
	case PacketCtxResponse:
		payload, err := l.Decrypt(packet.Data)
		if err != nil || len(payload) == 0 {
			return
		}
		var unpacked []any
		if err := umsgpack.Unpackb(payload, &unpacked); err != nil || len(unpacked) < 2 {
			Log(fmt.Sprintf("%s received malformed response payload: %v", l, err), LOG_WARNING)
			return
		}
		var requestID []byte
		if v, ok := unpacked[0].([]byte); ok {
			requestID = append([]byte(nil), v...)
		}
		response := unpacked[1]
		packedResponse, err := umsgpack.Packb(response)
		if err != nil {
			Log(fmt.Sprintf("%s failed to pack response payload: %v", l, err), LOG_WARNING)
			return
		}
		responseSize := len(packedResponse) - 2
		if responseSize < 0 {
			responseSize = 0
		}
		l.updatePhyStats(packet)
		go l.handleResponse(requestID, response, responseSize, responseSize, nil)
	case PacketCtxResourceAdv:
		plaintext, err := l.Decrypt(packet.Data)
		if err != nil || len(plaintext) == 0 {
			return
		}
		packet.Plaintext = plaintext
		packet.Link = l

		adv, err := (ResourceAdvertisement{}).Unpack(plaintext)
		if err != nil {
			Log(fmt.Sprintf("Could not unpack resource advertisement: %v", err), LOG_DEBUG)
			return
		}
		if len(adv.H) == 0 || len(adv.R) == 0 || len(adv.M) == 0 || adv.N <= 0 {
			Log(fmt.Sprintf("Malformed resource advertisement on %s (H=%d R=%d M=%d N=%d)", l, len(adv.H), len(adv.R), len(adv.M), adv.N), LOG_ERROR)
			return
		}
		l.updatePhyStats(packet)
		adv.Link = l

		switch {
		case adv.IsRequest():
			(&Resource{}).accept(
				packet,
				l.requestResourceConcluded,
				nil,
				adv.Q,
			)
		case adv.IsResponse():
			l.mu.Lock()
			var pending *RequestReceipt
			for _, candidate := range l.pendingRequests {
				if candidate != nil && bytes.Equal(candidate.RequestID(), adv.Q) {
					pending = candidate
					break
				}
			}
			l.mu.Unlock()
			if pending == nil {
				(&Resource{}).reject(packet)
				return
			}
			pending.noteResponseAdvertisement(adv)
			res := (&Resource{}).accept(
				packet,
				l.responseResourceConcluded,
				l.responseResourceProgress,
				adv.Q,
			)
			if res == nil {
				return
			}
			if res != nil {
				pending.responseResourceProgress(res)
			}
		default:
			switch l.resourceStrategy {
			case LinkAcceptNone:
				return
			case LinkAcceptApp:
				if l.callbacks.Resource == nil {
					return
				}
				func() {
					defer func() {
						if rec := recover(); rec != nil {
							Log(fmt.Sprintf("Error while executing resource accept callback from %s. The contained exception was: %v", l, rec), LOG_ERROR)
						}
					}()
					if l.callbacks.Resource(adv) {
						(&Resource{}).accept(packet, l.callbacks.ResourceConcluded, nil, nil)
					} else {
						(&Resource{}).reject(packet)
					}
				}()
			case LinkAcceptAll:
				(&Resource{}).accept(packet, l.callbacks.ResourceConcluded, nil, nil)
			}
		}
	case PacketCtxResourceReq:
		payload, err := l.Decrypt(packet.Data)
		if err != nil || len(payload) == 0 {
			Log(fmt.Sprintf("%s failed to decrypt resource request packet: %v", l, err), LOG_WARNING)
			return
		}
		l.updatePhyStats(packet)
		if len(payload) < 1 {
			return
		}
		resourceHashLen := sha256Bits / 8
		if len(payload) > 0 && payload[0] == HashmapExhausted {
			resourceHashLen = ReticulumTruncatedHashLength / 8
		}
		if len(payload) < 1+resourceHashLen {
			return
		}
		resourceHash := payload[1 : 1+resourceHashLen]
		l.mu.Lock()
		var resource *Resource
		for _, candidate := range l.outgoingResources {
			if candidate != nil && bytes.Equal(candidate.hash, resourceHash) {
				resource = candidate
				break
			}
		}
		l.mu.Unlock()
		if resource != nil {
			resource.Request(payload)
		}
	case PacketCtxResourceHMU:
		payload, err := l.Decrypt(packet.Data)
		if err != nil || len(payload) == 0 {
			return
		}
		l.updatePhyStats(packet)
		hashLen := sha256Bits / 8
		if len(payload) < hashLen {
			return
		}
		l.mu.Lock()
		var resource *Resource
		for _, candidate := range l.incomingResources {
			if candidate != nil && bytes.Equal(candidate.hash, payload[:hashLen]) {
				resource = candidate
				break
			}
		}
		l.mu.Unlock()
		if resource != nil {
			resource.HashmapUpdatePacket(payload)
		}
	case PacketCtxResourceICL:
		payload, err := l.Decrypt(packet.Data)
		if err != nil || len(payload) == 0 {
			return
		}
		l.updatePhyStats(packet)
		hashLen := sha256Bits / 8
		if len(payload) < hashLen {
			return
		}
		l.mu.Lock()
		var resource *Resource
		for _, candidate := range l.incomingResources {
			if candidate != nil && bytes.Equal(candidate.hash, payload[:hashLen]) {
				resource = candidate
				break
			}
		}
		l.mu.Unlock()
		if resource != nil {
			resource.Cancel()
		}
	case PacketCtxResourceRCL:
		payload, err := l.Decrypt(packet.Data)
		if err != nil || len(payload) == 0 {
			return
		}
		l.updatePhyStats(packet)
		hashLen := sha256Bits / 8
		if len(payload) < hashLen {
			return
		}
		l.mu.Lock()
		var resource *Resource
		for _, candidate := range l.outgoingResources {
			if candidate != nil && bytes.Equal(candidate.hash, payload[:hashLen]) {
				resource = candidate
				break
			}
		}
		l.mu.Unlock()
		if resource != nil {
			resource.rejected()
		}
	case PacketCtxResource:
		l.mu.Lock()
		incoming := make([]*Resource, len(l.incomingResources))
		copy(incoming, l.incomingResources)
		l.mu.Unlock()
		for _, res := range incoming {
			if res != nil {
				res.ReceivePart(packet)
				l.updatePhyStats(packet)
			}
		}
	default:
		if packet.PacketType == PacketTypeProof && packet.Context == PacketCtxResourcePrf {
			hashLen := sha256Bits / 8
			if len(packet.Data) < hashLen {
				return
			}
			l.mu.Lock()
			var resource *Resource
			for _, candidate := range l.outgoingResources {
				if candidate != nil && bytes.Equal(candidate.hash, packet.Data[:hashLen]) {
					resource = candidate
					break
				}
			}
			l.mu.Unlock()
			if resource == nil {
				return
			}
			go func(res *Resource, data []byte) {
				defer func() {
					if rec := recover(); rec != nil {
						Log(fmt.Sprintf("Resource proof callback panic on %s: %v", l, rec), LOG_ERROR)
					}
				}()
				res.ValidateProof(data)
			}(resource, append([]byte(nil), packet.Data...))
			l.updatePhyStats(packet)
		} else if packet.Context == PacketCtxLinkClose {
			l.teardownPacket(packet)
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
	if !l.trackPhyStats {
		return nil
	}
	return l.rssi
}

func (l *Link) GetSNR() *float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.trackPhyStats {
		return nil
	}
	return l.snr
}

func (l *Link) GetQ() *float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.trackPhyStats {
		return nil
	}
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
		PacketTypeData,
		PacketCtxKeepalive,
		Broadcast,
		HeaderType1,
		nil,
		nil,
		false,
		FlagUnset,
	)
	if p == nil {
		return
	}
	_ = p.Send()
	l.hadOutbound(true)
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
		PacketTypeProof,
		PacketCtxLRProof,
		Broadcast,
		HeaderType1,
		nil,
		nil,
		true,
		FlagUnset,
	)
	if proof == nil {
		return
	}
	_ = proof.Send()
	if len(proof.Raw) > 0 {
		l.EstablishmentCost += len(proof.Raw)
	}
	l.hadOutbound(false)
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
		mtu = MTU
		if phyParamsSnapshot.PhysicalLayerMTU > 0 {
			mtu = phyParamsSnapshot.PhysicalLayerMTU
		}
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

func (l *Link) validateProof(packet *Packet) {
	defer func() {
		if r := recover(); r != nil {
			l.Status = LinkClosed
			Log(fmt.Sprintf("An error ocurred while validating link request proof on %s.", l), LOG_ERROR)
			Log(fmt.Sprintf("The contained exception was: %v", r), LOG_ERROR)
		}
	}()
	if l == nil || packet == nil || l.Status != LinkPending || !l.Initiator || l.destination == nil || l.destination.identity == nil {
		return
	}

	signalling := []byte{}
	confirmedMTU := 0
	mode := linkModeFromProofPacket(packet)
	if desc, ok := linkModeDescriptions[mode]; ok {
		Log(fmt.Sprintf("Validating link request proof with mode %s", desc), LOG_DEBUG)
	}
	if mode != l.Mode {
		panic(fmt.Sprintf("Invalid link mode %d in link request proof", mode))
	}

	proofData := packet.Data
	if len(packet.Data) == ed25519.SignatureSize+linkEcPubSize/2+linkSignalSize {
		if mtu, ok := linkMTUFromProofPacket(packet); ok {
			confirmedMTU = mtu
			sb, err := linkSignallingBytes(confirmedMTU, mode)
			if err != nil {
				panic(err)
			}
			signalling = sb
			proofData = packet.Data[:ed25519.SignatureSize+linkEcPubSize/2]
			Log(fmt.Sprintf("Destination confirmed link MTU of %s", PrettySize(float64(confirmedMTU))), LOG_DEBUG)
		}
	}

	if len(proofData) != ed25519.SignatureSize+linkEcPubSize/2 {
		return
	}

	peerPub := proofData[ed25519.SignatureSize : ed25519.SignatureSize+linkEcPubSize/2]
	peerSigPub := l.destination.identity.GetPublicKey()[linkEcPubSize/2 : linkEcPubSize]
	if err := l.loadPeer(peerPub, peerSigPub); err != nil {
		panic(err)
	}
	if err := l.Handshake(); err != nil {
		panic(err)
	}

	l.EstablishmentCost += len(packet.Raw)
	signed := make([]byte, 0, len(l.LinkID)+len(l.peerPubBytes)+len(peerSigPub)+len(signalling))
	signed = append(signed, l.LinkID...)
	signed = append(signed, l.peerPubBytes...)
	signed = append(signed, peerSigPub...)
	signed = append(signed, signalling...)
	signature := proofData[:ed25519.SignatureSize]

	if !l.destination.identity.Validate(signature, signed) {
		Log(fmt.Sprintf("Invalid link proof signature received by %s. Ignoring.", l), LOG_DEBUG)
		return
	}
	if l.Status != LinkHandshake {
		panic(fmt.Sprintf("Invalid link state for proof validation: %d", l.Status))
	}

	l.RTT = time.Since(l.requestTime)
	l.attachedInterface = packet.ReceivingInterface
	l.remoteIdentity = l.destination.identity
	if confirmedMTU > 0 {
		l.MTU = confirmedMTU
	} else {
		l.MTU = MTU
	}
	l.updateMDU()
	l.Status = LinkActive
	l.activatedAt = time.Now()
	l.lastProof = l.activatedAt
	activateLink(l)
	Log(fmt.Sprintf("Link %s established with %v, RTT is %s", l, l.destination, PrettyShortTime(l.RTT.Seconds(), true, false)), LOG_DEBUG)

	if l.RTT > 0 && l.EstablishmentCost > 0 {
		l.EstablishmentRate = float64(l.EstablishmentCost) / l.RTT.Seconds()
	}

	l.updateKeepalive()
	if rttData, err := umsgpack.Packb(l.RTT.Seconds()); err == nil {
		rttPacket := NewPacket(l, rttData, PacketTypeData, PacketCtxLRRTT, Broadcast, HeaderType1, nil, nil, true, FlagUnset)
		if rttPacket != nil {
			_ = rttPacket.Send()
			l.hadOutbound(false)
			l.updatePhyStats(packet)
		}
	}

	if l.callbacks.LinkEstablished != nil {
		go l.callbacks.LinkEstablished(l)
	}
}

func (l *Link) rttPacket(packet *Packet) {
	defer func() {
		if r := recover(); r != nil {
			Log(fmt.Sprintf("Error occurred while processing RTT packet, tearing down link. The contained exception was: %v", r), LOG_ERROR)
			l.Teardown()
		}
	}()
	if l == nil || packet == nil {
		return
	}
	measuredRTT := time.Since(l.requestTime).Seconds()
	plaintext, err := l.Decrypt(packet.Data)
	if err != nil || len(plaintext) == 0 {
		return
	}
	var rttSec float64
	if err := umsgpack.Unpackb(plaintext, &rttSec); err != nil {
		panic(err)
	}
	if measuredRTT > rttSec {
		rttSec = measuredRTT
	}
	l.RTT = time.Duration(rttSec * float64(time.Second))
	l.Status = LinkActive
	l.activatedAt = time.Now()

	if l.RTT > 0 && l.EstablishmentCost > 0 {
		l.EstablishmentRate = float64(l.EstablishmentCost) / l.RTT.Seconds()
	}

	l.updateKeepalive()

	if l.owner != nil && l.owner.callbacks.LinkEstablished != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					Log(fmt.Sprintf("Error occurred in external link establishment callback. The contained exception was: %v", r), LOG_ERROR)
				}
			}()
			l.owner.callbacks.LinkEstablished(l)
		}()
	}
}

func (l *Link) Handshake() error {
	if l == nil || l.Status != LinkPending || l.priv == nil || l.peerPub == nil {
		Log(fmt.Sprintf("Handshake attempt on %s with invalid state %d", l, l.Status), LOG_ERROR)
		return errors.New("invalid handshake state")
	}
	l.Status = LinkHandshake
	shared, err := l.priv.ECDH(l.peerPub)
	if err != nil {
		return err
	}
	l.sharedKey = shared

	derivedLen := 64
	switch l.Mode {
	case LinkModeAES128CBC:
		derivedLen = 32
	case LinkModeAES256CBC:
		derivedLen = 64
	default:
		return fmt.Errorf("invalid link mode %d on %s", l.Mode, l)
	}
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
	return nil
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
		tok, err := Cryptography.NewToken(l.derivedKey)
		if err != nil {
			Log(fmt.Sprintf("Could not instantiate token while performing encryption on link %s. The contained exception was: %v", l, err), LogError)
			return nil, err
		}
		l.token = tok
	}
	ct, err := l.token.Encrypt(plaintext)
	if err != nil {
		Log(fmt.Sprintf("Encryption on link %s failed. The contained exception was: %v", l, err), LogError)
		return nil, err
	}
	return ct, nil
}

func (l *Link) Decrypt(ciphertext []byte) ([]byte, error) {
	if l.token == nil {
		tok, err := Cryptography.NewToken(l.derivedKey)
		if err != nil {
			return nil, err
		}
		l.token = tok
	}
	pt, err := l.token.Decrypt(ciphertext)
	if err != nil {
		Log(fmt.Sprintf("Decryption failed on link %s. The contained exception was: %v", l, err), LogError)
		return nil, err
	}
	return pt, nil
}

func (l *Link) Sign(message []byte) ([]byte, error) {
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
	proof := NewPacket(l, proofData, PacketTypeProof, PacketCtxNone, Broadcast, HeaderType1, nil, nil, false, FlagUnset)
	if proof == nil {
		return
	}
	_ = proof.Send()
	l.hadOutbound(false)
}

func (l *Link) setLinkID(packet *Packet) {
	if packet == nil {
		return
	}
	// Python parity: Link.link_id_from_lr_packet(packet)
	hashable := packet.getHashablePart()
	if len(packet.Data) > linkEcPubSize {
		diff := len(packet.Data) - linkEcPubSize
		if diff > 0 {
			hashable = hashable[:len(hashable)-diff]
		}
	}
	l.LinkID = TruncatedHash(hashable)
	l.Hash = l.LinkID
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
		PacketTypeData,
		PacketCtxLinkIdentify,
		Broadcast,
		HeaderType1,
		nil,
		nil,
		false,
		FlagUnset,
	)
	if packet == nil {
		Log("Failed to create link identify packet", LOG_ERROR)
		return
	}
	if receipt := packet.Send(); receipt == nil && !packet.Sent {
		Log("Link identify packet could not be sent", LOG_WARNING)
	}
	l.hadOutbound(false)
}

// ==== Watchdog ====

func (l *Link) startWatchdog() {
	l.watchdogOnce.Do(func() {
		go func() {
			for {
				l.mu.Lock()
				if l.Status == LinkClosed {
					l.mu.Unlock()
					return
				}
				locked := l.watchdogLock
				rtt := l.RTT
				l.mu.Unlock()

				if locked {
					sleepFor := 25 * time.Millisecond
					if rtt > 0 && rtt > sleepFor {
						sleepFor = rtt
					}
					time.Sleep(sleepFor)
					continue
				}

				sleepTime := 1 * time.Millisecond

				// Python: in PENDING/HANDSHAKE, establishment timeout is based on request_time.
				if (l.Status == LinkPending || l.Status == LinkHandshake) && !l.requestTime.IsZero() && l.estTimeout > 0 {
					nextCheck := l.requestTime.Add(l.estTimeout)
					sleepTime = time.Until(nextCheck)
					if time.Now().After(nextCheck) || time.Now().Equal(nextCheck) {
						if l.Status == LinkPending {
							Log("Link establishment timed out", LOG_VERBOSE)
						}
						l.Status = LinkClosed
						l.TeardownReason = LinkTimeout
						initiator := l.Initiator
						l.linkClosed()
						if initiator {
							Log("Timeout waiting for link request proof", LOG_DEBUG)
						} else {
							Log("Timeout waiting for RTT packet from link initiator", LOG_DEBUG)
						}
						continue
					}
				} else if l.Status == LinkActive {
					activatedAt := l.activatedAt
					lastInbound := l.lastInbound
					if l.lastProof.After(lastInbound) {
						lastInbound = l.lastProof
					}
					if activatedAt.After(lastInbound) {
						lastInbound = activatedAt
					}

					if lastInbound.IsZero() {
						l.mu.Unlock()
						time.Sleep(linkWatchdogMaxSleep)
						continue
					}

					if l.Keepalive <= 0 {
						l.Keepalive = linkKeepaliveMax
					}
					if l.StaleTime <= 0 {
						l.StaleTime = l.Keepalive * linkStaleFactor
					}

					now := time.Now()
					if !now.Before(lastInbound.Add(l.Keepalive)) {
						if l.Initiator && !now.Before(l.lastKeepalive.Add(l.Keepalive)) {
							go l.sendKeepalive()
						}
						if !now.Before(lastInbound.Add(l.StaleTime)) {
							rtt := l.RTT
							if rtt <= 0 {
								rtt = 500 * time.Millisecond
							}
							mult := l.KeepaliveTimeoutFactor
							if mult <= 0 {
								mult = linkKeepaliveFact
							}
							sleepTime = time.Duration(mult*float64(rtt)) + linkStaleGrace
							l.Status = LinkStale
						} else {
							sleepTime = lastInbound.Add(l.Keepalive).Sub(now)
						}
					} else {
						sleepTime = lastInbound.Add(l.Keepalive).Sub(now)
					}
				} else if l.Status == LinkStale {
					l.teardown(LinkTimeout)
					time.Sleep(time.Millisecond)
					continue
				}

				if sleepTime == 0 {
					Log("Warning! Link watchdog sleep time of 0!", LOG_ERROR)
				}
				if sleepTime < 0 {
					Log(fmt.Sprintf("Timing error! Tearing down link %s now.", l), LOG_ERROR)
					l.Teardown()
					sleepTime = 100 * time.Millisecond
					time.Sleep(sleepTime)
					if !l.trackPhyStats {
						l.mu.Lock()
						l.rssi = nil
						l.snr = nil
						l.q = nil
						l.mu.Unlock()
					}
					continue
				}
				if sleepTime > linkWatchdogMaxSleep {
					sleepTime = linkWatchdogMaxSleep
				}
				time.Sleep(sleepTime)
				if !l.trackPhyStats {
					l.mu.Lock()
					l.rssi = nil
					l.snr = nil
					l.q = nil
					l.mu.Unlock()
				}
			}
		}()
	})
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

func (l *Link) teardownPacket(packet *Packet) {
	if l == nil || packet == nil {
		return
	}
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
	l.Status = LinkClosed
	if initiator {
		l.TeardownReason = LinkDestinationClose
	} else {
		l.TeardownReason = LinkInitiatorClose
	}
	l.updatePhyStats(packet)
	l.linkClosed()
}

func (l *Link) teardown(reason int) {
	if l == nil {
		return
	}
	l.mu.Lock()
	sendClose := l.Status != LinkPending && l.Status != LinkClosed
	linkID := append([]byte(nil), l.LinkID...)
	l.mu.Unlock()
	if sendClose {
		p := NewPacket(l, linkID, PacketTypeData, PacketCtxLinkClose, Broadcast, HeaderType1, nil, nil, false, FlagUnset)
		if p != nil {
			_ = p.Send()
			l.hadOutbound(false)
		}
	}
	l.TeardownReason = reason
	l.Status = LinkClosed
	l.linkClosed()
}

func (l *Link) linkClosed() {
	if l == nil {
		return
	}
	l.mu.Lock()
	incomingResources := append([]*Resource(nil), l.incomingResources...)
	outgoingResources := append([]*Resource(nil), l.outgoingResources...)
	if l.channel != nil {
		channelClose(l.channel)
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
	destination := l.destination
	l.mu.Unlock()

	for _, resource := range incomingResources {
		if resource != nil {
			resource.Cancel()
		}
	}
	for _, resource := range outgoingResources {
		if resource != nil {
			resource.Cancel()
		}
	}

	if destination != nil && destination.Direction == DestinationIN {
		destination.linksMu.Lock()
		for i, existing := range destination.links {
			if existing == l {
				destination.links = append(destination.links[:i], destination.links[i+1:]...)
				break
			}
		}
		destination.linksMu.Unlock()
	}

	if cb != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					Log(fmt.Sprintf("Error while executing link closed callback from %s. The contained exception was: %v", l, r), LOG_ERROR)
				}
			}()
			cb(l)
		}()
	}
}

func (l *Link) String() string {
	return PrettyHexRep(l.LinkID)
}

func (l *Link) handleRequest(requestID []byte, unpacked []any, packet *Packet) {
	if len(unpacked) < 3 {
		Log("Malformed request payload received, ignoring", LOG_WARNING)
		return
	}

	requestedAt := time.Unix(0, 0)
	switch v := unpacked[0].(type) {
	case float64:
		requestedAt = time.Unix(0, int64(v*float64(time.Second)))
	case float32:
		requestedAt = time.Unix(0, int64(float64(v)*float64(time.Second)))
	case int:
		requestedAt = time.Unix(0, int64(float64(v)*float64(time.Second)))
	case int64:
		requestedAt = time.Unix(0, int64(float64(v)*float64(time.Second)))
	case uint64:
		requestedAt = time.Unix(0, int64(float64(v)*float64(time.Second)))
	case uint32:
		requestedAt = time.Unix(0, int64(float64(v)*float64(time.Second)))
	}
	var pathHash []byte
	if v, ok := unpacked[1].([]byte); ok {
		pathHash = append([]byte(nil), v...)
	}
	if len(pathHash) == 0 {
		Log("Request without path hash received, ignoring", LOG_WARNING)
		return
	}
	requestData := unpacked[2]

	l.mu.Lock()
	dest := l.destination
	remoteID := l.remoteIdentity
	linkID := append([]byte{}, l.LinkID...)
	l.mu.Unlock()

	if dest == nil {
		return
	}

	handler, ok := dest.requestHandlers[string(pathHash)]
	if !ok || handler == nil {
		return
	}

	resp, allowed := destinationDispatchRequest(dest, handler.Path, requestData, requestID, linkID, remoteID, requestedAt)
	if !allowed {
		return
	}
	if resp == nil {
		// Allowed request with nil response still counts as delivered.
		if packet != nil {
			l.ProvePacket(packet)
		}
		return
	}

	payload, err := umsgpack.Packb([]any{requestID, resp})
	if err != nil {
		Log(fmt.Sprintf("Could not pack response for %s: %v", handler.Path, err), LOG_ERROR)
		if packet != nil {
			l.ProvePacket(packet)
		}
		return
	}
	if len(payload) <= l.MDU {
		responsePacket := NewPacket(
			l,
			payload,
			PacketTypeData,
			PacketCtxResponse,
			Broadcast,
			HeaderType1,
			nil,
			nil,
			false,
			FlagUnset,
		)
		if responsePacket == nil {
			Log("Failed to create response packet", LOG_ERROR)
			if packet != nil {
				l.ProvePacket(packet)
			}
			return
		}
		if receipt := responsePacket.Send(); receipt == nil && !responsePacket.Sent {
			Log("Response packet could not be sent", LOG_WARNING)
		}
		if packet != nil {
			l.ProvePacket(packet)
		}
		return
	}

	timeout := time.Duration(float64(l.RTT)*l.TrafficTimeoutFactor) + ResponseMaxGraceTime
	timeoutSeconds := timeout.Seconds()
	callback := func(res *Resource) {
		l.ResourceConcluded(res)
		if cb := l.callbacks.ResourceConcluded; cb != nil {
			cb(res)
		}
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
	if packet != nil {
		l.ProvePacket(packet)
	}
}

// ==== helper functions ====

func linkMTUFromLRPacket(packet *Packet) (int, bool) {
	if packet == nil {
		return 0, false
	}
	if len(packet.Data) != linkEcPubSize+linkSignalSize {
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
	if mode != linkEnabledModes[0] {
		return nil, fmt.Errorf("link mode %d disabled", mode)
	}
	if mtu <= 0 {
		mtu = MTU
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
	if len(packet.Data) <= ed25519.SignatureSize+linkEcPubSize/2 {
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
