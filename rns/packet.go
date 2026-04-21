package rns

import (
	"errors"
	"fmt"
	"time"
)

// ===== Packet types =====

const (
	PacketTypeData        = 0x00
	PacketTypeAnnounce    = 0x01
	PacketTypeLinkRequest = 0x02
	PacketTypeProof       = 0x03
)

// Header types
const (
	HeaderType1 = 0x00
	HeaderType2 = 0x01
)

// Contexts
const (
	PacketCtxNone          = 0x00
	PacketCtxResource      = 0x01
	PacketCtxResourceAdv   = 0x02
	PacketCtxResourceReq   = 0x03
	PacketCtxResourceHMU   = 0x04
	PacketCtxResourcePrf   = 0x05
	PacketCtxResourceICL   = 0x06
	PacketCtxResourceRCL   = 0x07
	PacketCtxCacheRequest  = 0x08
	PacketCtxRequest       = 0x09
	PacketCtxResponse      = 0x0A
	PacketCtxPathResponse  = 0x0B
	PacketCtxCommand       = 0x0C
	PacketCtxCommandStatus = 0x0D
	PacketCtxChannel       = 0x0E
	PacketCtxKeepalive     = 0xFA
	PacketCtxLinkIdentify  = 0xFB
	PacketCtxLinkClose     = 0xFC
	PacketCtxLinkProof     = 0xFD
	PacketCtxLRRTT         = 0xFE
	PacketCtxLRProof       = 0xFF
)

// Context flags
const (
	FlagSet   = 0x01
	FlagUnset = 0x00
)

// Exported aliases (matching python naming)
const (
	PacketDATA        = PacketTypeData
	PacketANNOUNCE    = PacketTypeAnnounce
	PacketLINKREQUEST = PacketTypeLinkRequest
	PacketPROOF       = PacketTypeProof

	// Backwards/porting aliases (older Go code used these names).
	PacketData     = PacketDATA
	PacketAnnounce = PacketANNOUNCE
	PacketProof    = PacketPROOF
)

const (
	PacketNONE           = PacketCtxNone
	PacketRESOURCE       = PacketCtxResource
	PacketRESOURCE_ADV   = PacketCtxResourceAdv
	PacketRESOURCE_REQ   = PacketCtxResourceReq
	PacketRESOURCE_HMU   = PacketCtxResourceHMU
	PacketRESOURCE_PRF   = PacketCtxResourcePrf
	PacketRESOURCE_ICL   = PacketCtxResourceICL
	PacketRESOURCE_RCL   = PacketCtxResourceRCL
	PacketCACHE_REQUEST  = PacketCtxCacheRequest
	PacketPATH_RESPONSE  = PacketCtxPathResponse
	PacketCOMMAND        = PacketCtxCommand
	PacketCOMMAND_STATUS = PacketCtxCommandStatus
	PacketCHANNEL        = PacketCtxChannel
	PacketKEEPALIVE      = PacketCtxKeepalive
	PacketLINKIDENTIFY   = PacketCtxLinkIdentify
	PacketLINKCLOSE      = PacketCtxLinkClose
	PacketLINKPROOF      = PacketCtxLinkProof
	PacketLRRTT          = PacketCtxLRRTT
	PacketLRPROOF        = PacketCtxLRProof
)

// Backwards/porting aliases for context constants.
const (
	PacketKeepalive    = PacketKEEPALIVE
	PacketLRProof      = PacketLRPROOF
	PacketResource     = PacketRESOURCE
	PacketResourceRCL  = PacketRESOURCE_RCL
	PacketResourceReq  = PacketRESOURCE_REQ
	PacketResourcePrf  = PacketRESOURCE_PRF
	PacketResourceHMU  = PacketRESOURCE_HMU
	PacketResourceICL  = PacketRESOURCE_ICL
	PacketCacheRequest = PacketCACHE_REQUEST
	PacketChannel      = PacketCHANNEL
)

// Backwards/porting aliases for header type constants.
const (
	Header1 = HeaderType1
	Header2 = HeaderType2
)

const (
	PacketFLAG_SET   = FlagSet
	PacketFLAG_UNSET = FlagUnset
)

const (
	PacketCONTEXT_RESOURCE_ADV  = PacketCtxResourceAdv
	PacketCONTEXT_RESOURCE      = PacketCtxResource
	PacketCONTEXT_RESOURCE_REQ  = PacketCtxResourceReq
	PacketCONTEXT_RESOURCE_HMU  = PacketCtxResourceHMU
	PacketCONTEXT_RESOURCE_PRF  = PacketCtxResourcePrf
	PacketCONTEXT_RESOURCE_ICL  = PacketCtxResourceICL
	PacketCONTEXT_RESOURCE_RCL  = PacketCtxResourceRCL
	PacketCONTEXT_CACHE_REQUEST = PacketCtxCacheRequest
)

var (
	PacketMDU          = MDU
	PacketPlainMDU     = MDU
	PacketEncryptedMDU = computeEncryptedPacketMDU(MDU)
)

type packetOptions struct {
	packetType    byte
	context       byte
	transportType byte
	headerType    byte
	transportID   []byte
	attached      *Interface
	createReceipt bool
	contextFlag   byte
}

type PacketOption func(*packetOptions)

func defaultPacketOptions() packetOptions {
	return packetOptions{
		packetType:    PacketTypeData,
		context:       PacketCtxNone,
		transportType: Broadcast,
		headerType:    HeaderType1,
		createReceipt: true,
		contextFlag:   FlagUnset,
	}
}

func WithPacketType(t byte) PacketOption {
	return func(o *packetOptions) {
		o.packetType = t
	}
}

func WithPacketContext(ctx byte) PacketOption {
	return func(o *packetOptions) {
		o.context = ctx
	}
}

func WithTransportType(t byte) PacketOption {
	return func(o *packetOptions) {
		o.transportType = t
	}
}

func WithHeaderType(t byte) PacketOption {
	return func(o *packetOptions) {
		o.headerType = t
	}
}

func WithTransportID(id []byte) PacketOption {
	return func(o *packetOptions) {
		o.transportID = copyBytes(id)
	}
}

func WithAttachedInterface(ifc *Interface) PacketOption {
	return func(o *packetOptions) {
		o.attached = ifc
	}
}

func WithCreateReceipt(enable bool) PacketOption {
	return func(o *packetOptions) {
		o.createReceipt = enable
	}
}

func WithoutReceipt() PacketOption {
	return WithCreateReceipt(false)
}

func WithContextFlag(flag byte) PacketOption {
	return func(o *packetOptions) {
		o.contextFlag = flag
	}
}

// Python: Packet.TIMEOUT_PER_HOP = Reticulum.DEFAULT_PER_HOP_TIMEOUT
const TimeoutPerHop = float64(DEFAULT_PER_HOP_TIMEOUT)

// Python parity: Link.TRAFFIC_TIMEOUT_MIN = 5 seconds.
// This is a lower bound for per-packet receipt timeouts on links, ensuring that
// early link RTT estimates (often 0 at startup) don't cause immediate timeouts.
const trafficTimeoutMin = 5 * time.Second

// ===== Minimal transport interface =====

type TransportBackend interface {
}

// Default transport backend is wired at package init time via the concrete
// zero-value backend, matching Python module-level static state.
var Transport TransportBackend = defaultTransportBackend{}

// ===== Packet =====

type Packet struct {
	Hops          uint8
	Header        []byte
	HeaderType    byte
	PacketType    byte
	Type          byte // alias for legacy callers expecting "Type"
	TransportType byte
	Context       byte
	ContextFlag   byte

	Destination *Destination
	TransportID []byte
	Data        []byte
	Flags       byte
	Raw         []byte
	Packed      bool
	Sent        bool

	CreateReceipt bool
	Receipt       *PacketReceipt
	FromPacked    bool

	MTU        int
	SentAt     time.Time
	PacketHash []byte
	RatchetID  []byte

	AttachedInterface  *Interface
	ReceivingInterface *Interface

	RSSI *float64
	SNR  *float64
	Q    *float64

	Ciphertext      []byte
	Plaintext       []byte
	DestinationHash []byte
	DestinationType byte
	Link            *Link
	MapHash         []byte
}

type PacketStateError struct {
	Message string
}

func (e *PacketStateError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func panicPacketState(message string) {
	panic(&PacketStateError{Message: message})
}

// NewPacket constructs a packet destined for either a Destination or Link.
// When target is nil, the packet represents already-packed raw data.
func NewPacket(target interface{}, data []byte, opts ...PacketOption) *Packet {
	if target == nil {
		return &Packet{
			Raw:           copyBytes(data),
			Data:          copyBytes(data),
			Packed:        true,
			FromPacked:    true,
			CreateReceipt: false,
		}
	}

	cfg := defaultPacketOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	var (
		dest *Destination
		link *Link
	)

	switch v := target.(type) {
	case *Destination:
		dest = v
	case *Link:
		if v == nil || len(v.LinkID) == 0 {
			Log("Cannot create packet for nil link or link without link ID", LogError)
			return nil
		}
		link = v
		// For link packets, the destination is still the owning Destination.
		// The packet header uses the link ID as destination hash.
		dest = v.destination
	default:
		Log(fmt.Sprintf("Unsupported packet target %T", target), LogError)
		return nil
	}

	if dest == nil {
		return nil
	}

	packet := &Packet{
		HeaderType:        cfg.headerType,
		PacketType:        cfg.packetType,
		Type:              cfg.packetType,
		TransportType:     cfg.transportType,
		Context:           cfg.context,
		ContextFlag:       cfg.contextFlag,
		Hops:              0,
		Destination:       dest,
		TransportID:       copyBytes(cfg.transportID),
		Data:              copyBytes(data),
		CreateReceipt:     cfg.createReceipt,
		FromPacked:        false,
		AttachedInterface: cfg.attached,
		Link:              link,
	}

	packet.Flags = packet.getPackedFlags()

	// Python: Packet.MTU is the physical-layer MTU (raw bytes), except for link
	// packets where it uses the per-link MTU. Destination.mtu in this Go port is
	// used for payload sizing (MDU), not raw packet length.
	if link != nil {
		packet.MTU = link.MTU
	} else {
		packet.MTU = MTU
	}

	return packet
}

// NewRawPacket mirrors Python Packet(None, raw) construction for already-packed bytes.
func NewRawPacket(raw []byte) *Packet {
	return NewPacket(nil, raw)
}

func (p *Packet) getPackedFlags() byte {
	if p.Context == PacketCtxLRProof {
		// LINK is always for LRPROOF
		return (p.HeaderType << 6) |
			(p.ContextFlag << 5) |
			(p.TransportType << 4) |
			(byte(DestinationLINK) << 2) |
			p.PacketType
	}
	destType := byte(0)
	if p.Link != nil {
		destType = byte(DestinationLINK)
	} else if p.Destination != nil {
		destType = byte(p.Destination.Type)
	}
	return (p.HeaderType << 6) |
		(p.ContextFlag << 5) |
		(p.TransportType << 4) |
		(destType << 2) |
		p.PacketType
}

// Pack mirrors pack().
func (p *Packet) Pack() error {
	if p.Destination == nil && p.Link == nil {
		return errors.New("cannot pack packet without destination")
	}

	if p.Link != nil {
		p.DestinationHash = copyBytes(p.Link.LinkID)
	} else {
		p.DestinationHash = copyBytes(p.Destination.Hash)
	}
	if len(p.DestinationHash) != truncatedHashBytes {
		return fmt.Errorf("invalid destination hash length %d (expected %d)", len(p.DestinationHash), truncatedHashBytes)
	}
	header := make([]byte, 0, 64)

	header = append(header, p.Flags)
	header = append(header, byte(p.Hops))

	if p.Context == PacketCtxLRProof {
		if p.Link == nil || len(p.Link.LinkID) == 0 {
			return errors.New("LRPROOF packet requires link_id")
		}
		if len(p.Link.LinkID) != truncatedHashBytes {
			return fmt.Errorf("invalid link_id length %d (expected %d)", len(p.Link.LinkID), truncatedHashBytes)
		}
		header = append(header, p.Link.LinkID...)
		p.Ciphertext = p.Data
	} else {
		if p.HeaderType == HeaderType1 {
			header = append(header, p.DestinationHash...)

			switch {
			case p.PacketType == PacketTypeAnnounce:
				p.Ciphertext = p.Data
			case p.PacketType == PacketTypeLinkRequest:
				p.Ciphertext = p.Data
			case p.PacketType == PacketTypeProof && p.Context == PacketCtxResourcePrf:
				p.Ciphertext = p.Data
			case p.PacketType == PacketTypeProof && p.Link != nil:
				p.Ciphertext = p.Data
			case p.Context == PacketCtxResource:
				p.Ciphertext = p.Data
			case p.Context == PacketCtxKeepalive:
				p.Ciphertext = p.Data
			case p.Context == PacketCtxCacheRequest:
				p.Ciphertext = p.Data
			default:
				ct, err := p.encryptForDestination(p.Data)
				if err != nil {
					return err
				}
				p.Ciphertext = ct
				if p.Destination != nil && len(p.Destination.latestRatchetID) > 0 {
					p.RatchetID = append([]byte{}, p.Destination.latestRatchetID...)
				}
			}
		}

		if p.HeaderType == HeaderType2 {
			if len(p.TransportID) == 0 {
				return errors.New("header type 2 packet must have transportID")
			}
			if len(p.TransportID) != truncatedHashBytes {
				return fmt.Errorf("invalid transportID length %d (expected %d)", len(p.TransportID), truncatedHashBytes)
			}
			header = append(header, p.TransportID...)
			header = append(header, p.DestinationHash...)
			// Python only uses Packet.pack() with HEADER_2 for announces; other
			// transported packets are inserted into transport by rewriting raw bytes.
			if p.PacketType != PacketTypeAnnounce {
				return fmt.Errorf("header type 2 Pack() is only supported for announce packets (got type 0x%02x)", p.PacketType)
			}
			// Announce packets are not encrypted.
			p.Ciphertext = p.Data
		}
	}

	header = append(header, p.Context)
	p.Header = header
	p.Raw = append(header, p.Ciphertext...)

	if p.MTU > 0 && len(p.Raw) > p.MTU {
		return fmt.Errorf("packet size of %d exceeds MTU of %d bytes", len(p.Raw), p.MTU)
	}

	p.Packed = true
	p.UpdateHash()
	return nil
}

func (p *Packet) encryptForDestination(plaintext []byte) ([]byte, error) {
	if p.Link != nil {
		return p.Link.Encrypt(plaintext)
	}
	if p.Destination == nil {
		return nil, errors.New("no destination")
	}
	// ProofDestination compatibility: Python uses a special Destination-like object
	// with type SINGLE that performs no encryption.
	if p.Destination.Type == DestinationSINGLE && p.Destination.identity == nil {
		return plaintext, nil
	}
	ct := p.Destination.Encrypt(plaintext)
	if ct == nil && plaintext != nil {
		return nil, errors.New("destination encryption failed")
	}
	return ct, nil
}

// Unpack mirrors unpack().
func (p *Packet) Unpack() bool {
	defer func() {
		if r := recover(); r != nil {
			// swallow like Python, just return false
		}
	}()

	if len(p.Raw) < 2 {
		return false
	}

	p.Flags = p.Raw[0]
	p.Hops = p.Raw[1]

	p.HeaderType = (p.Flags & 0b01000000) >> 6
	p.ContextFlag = (p.Flags & 0b00100000) >> 5
	p.TransportType = (p.Flags & 0b00010000) >> 4
	p.DestinationType = (p.Flags & 0b00001100) >> 2
	p.PacketType = (p.Flags & 0b00000011)
	p.Type = p.PacketType

	dstLen := truncatedHashBytes

	if p.HeaderType == HeaderType2 {
		if len(p.Raw) < 2+2*dstLen+1 {
			return false
		}
		p.TransportID = copyBytes(p.Raw[2 : dstLen+2])
		p.DestinationHash = copyBytes(p.Raw[dstLen+2 : 2*dstLen+2])
		p.Context = p.Raw[2*dstLen+2]
		p.Data = copyBytes(p.Raw[2*dstLen+3:])
	} else {
		if len(p.Raw) < 2+dstLen+1 {
			return false
		}
		p.TransportID = nil
		p.DestinationHash = copyBytes(p.Raw[2 : dstLen+2])
		p.Context = p.Raw[dstLen+2]
		p.Data = copyBytes(p.Raw[dstLen+3:])
	}

	p.Packed = false
	p.UpdateHash()
	return true
}

// Send mirrors send().
func (p *Packet) Send() *PacketReceipt {
	if p.Sent {
		panicPacketState("Packet was already sent")
	}
	if p.Destination == nil && p.Link == nil {
		Log("Cannot send packet without destination", LogError)
		return nil
	}
	if p.Link != nil {
		if p.Link.Status == LinkClosed {
			Log("Attempt to transmit over a closed link, dropping packet", LogDebug)
			// just exit
			p.Sent = false
			p.Receipt = nil
			return nil
		}
		p.Link.noteOutbound(p.Context, len(p.Data))
	}

	if !p.Packed {
		if err := p.Pack(); err != nil {
			Logf(LogError, "Could not pack packet: %v", err)
			return nil
		}
	}

	if Transport == nil {
		Log("Transport backend not initialised", LogError)
		return nil
	}

	if Outbound(p) {
		return p.Receipt
	}

	Log("No interfaces could process the outbound packet", LogError)
	p.Sent = false
	p.SentAt = time.Time{}
	p.Receipt = nil
	return nil
}

// Resend mirrors resend().
func (p *Packet) Resend() *PacketReceipt {
	if !p.Sent {
		panicPacketState("Packet was not sent yet")
	}
	if err := p.Pack(); err != nil {
		Logf(LogError, "Could not repack packet before resend: %v", err)
		return nil
	}

	if Transport == nil {
		Log("Transport backend not initialised", LogError)
		return nil
	}

	if Outbound(p) {
		return p.Receipt
	}

	Log("No interfaces could process the outbound packet", LogError)
	p.Sent = false
	p.SentAt = time.Time{}
	p.Receipt = nil
	return nil
}

// Prove / ProofDestination

func (p *Packet) Prove(dest *Destination) {
	if p.FromPacked && p.Destination != nil && p.Destination.identity != nil && p.Destination.identity.prv != nil {
		p.Destination.identity.Prove(p, dest)
	} else if p.FromPacked && p.Link != nil {
		p.Link.ProvePacket(p)
	} else {
		Log("Could not prove packet without an associated destination or link", LogError)
	}
}

func (p *Packet) GenerateProofDestination() *Destination {
	// ProofDestination mirrors rns/packet.py: it is type SINGLE but performs no encryption.
	return &Destination{
		Type:      DestinationSINGLE,
		Direction: DestinationOUT,
		Hash:      append([]byte{}, p.GetHash()[:ReticulumTruncatedHashLength/8]...),
	}
}

func (p *Packet) ValidateProofPacket(proof *Packet) bool {
	if p == nil || p.Receipt == nil || proof == nil {
		return false
	}
	return p.Receipt.ValidateProofPacket(proof)
}

func (p *Packet) ValidateProof(proof []byte) bool {
	if p == nil || p.Receipt == nil {
		return false
	}
	return p.Receipt.ValidateProof(proof, nil)
}

// Hashing

func (p *Packet) UpdateHash() {
	p.PacketHash = p.GetHash()
}

func (p *Packet) GetHash() []byte {
	return FullHash(p.getHashablePart())
}

func (p *Packet) GetTruncatedHash() []byte {
	return TruncatedHash(p.getHashablePart())
}

func (p *Packet) getHashablePart() []byte {
	if len(p.Raw) == 0 {
		return nil
	}
	hashable := []byte{p.Raw[0] & 0x0F}
	if p.HeaderType == HeaderType2 {
		hashable = append(hashable, p.Raw[truncatedHashBytes+2:]...)
	} else {
		hashable = append(hashable, p.Raw[2:]...)
	}
	return hashable
}

// Raw returns a copy of the packed bytes. If the packet has not been packed yet,
// it returns nil.
func (p *Packet) RawBytes() []byte {
	return copyBytes(p.Raw)
}

// Metrics

func (p *Packet) GetRSSI() *float64 {
	if p.RSSI != nil {
		return p.RSSI
	}
	localStatsMu.RLock()
	v, ok := localRSSICache[string(p.PacketHash)]
	localStatsMu.RUnlock()
	if ok {
		return &v
	}
	return nil
}

func (p *Packet) GetSNR() *float64 {
	if p.SNR != nil {
		return p.SNR
	}
	localStatsMu.RLock()
	v, ok := localSNRCache[string(p.PacketHash)]
	localStatsMu.RUnlock()
	if ok {
		return &v
	}
	return nil
}

func (p *Packet) GetQ() *float64 {
	if p.Q != nil {
		return p.Q
	}
	localStatsMu.RLock()
	v, ok := localQCache[string(p.PacketHash)]
	localStatsMu.RUnlock()
	if ok {
		return &v
	}
	return nil
}

// ===== PacketReceipt =====

type PacketReceipt struct {
	Hash          []byte
	TruncatedHash []byte
	Sent          bool
	SentAt        time.Time
	Proved        bool
	Status        byte
	Destination   *Destination
	Link          *Link
	Callbacks     PacketReceiptCallbacks
	ConcludedAt   time.Time
	ProofPacket   *Packet
	Timeout       float64
}

const (
	ReceiptFailed    = 0x00
	ReceiptSent      = 0x01
	ReceiptDelivered = 0x02
	ReceiptReceiving = 0x03
	ReceiptReady     = 0x04
	ReceiptCulled    = 0xFF
)

const (
	ReceiptExplLength = HashLengthBytes + SigLengthBytes
	ReceiptImplLength = SigLengthBytes
)

type PacketReceiptCallbacks struct {
	Delivery func(*PacketReceipt)
	Timeout  func(*PacketReceipt)
}

func NewPacketReceipt(p *Packet) *PacketReceipt {
	r := &PacketReceipt{
		Hash:          p.GetHash(),
		TruncatedHash: p.GetTruncatedHash(),
		Sent:          true,
		SentAt:        time.Now(),
		Proved:        false,
		Status:        ReceiptSent,
		Destination:   p.Destination,
		Link:          p.Link,
		Callbacks:     PacketReceiptCallbacks{},
	}

	if p.Link != nil {
		r.Timeout = maxFloat(p.Link.RTT.Seconds()*p.Link.TrafficTimeoutFactor, trafficTimeoutMin.Seconds())
	} else if Transport != nil && p.Destination != nil {
		base := FirstHopTimeout(p.Destination.Hash).Seconds()
		if Owner != nil {
			base = Owner.GetFirstHopTimeout(p.Destination.Hash)
		}
		hops := HopsTo(p.Destination.Hash)
		if hops <= 0 {
			hops = 1
		}
		r.Timeout = base + TimeoutPerHop*float64(hops)
	} else {
		r.Timeout = 10.0
	}

	return r
}

func (r *PacketReceipt) GetStatus() byte {
	return r.Status
}

func (r *PacketReceipt) ValidateProofPacket(proofPacket *Packet) bool {
	if proofPacket.Link != nil {
		return r.validateLinkProof(proofPacket.Data, proofPacket.Link, proofPacket)
	}
	return r.ValidateProof(proofPacket.Data, proofPacket)
}

func (r *PacketReceipt) validateLinkProof(proof []byte, link *Link, proofPacket *Packet) bool {
	if len(proof) != ReceiptExplLength {
		return false
	}
	proofHash := proof[:HashLengthBytes]
	sig := proof[HashLengthBytes : HashLengthBytes+SigLengthBytes]

	if !bytesEqual(proofHash, r.Hash) {
		return false
	}

	if !link.Validate(sig, r.Hash) {
		return false
	}

	r.Status = ReceiptDelivered
	r.Proved = true
	r.ConcludedAt = time.Now()
	r.ProofPacket = proofPacket
	link.mu.Lock()
	link.lastProof = r.ConcludedAt
	link.mu.Unlock()
	if r.Link != nil {
		r.Link.observeRTTSeconds(r.GetRTT())
	}

	if r.Callbacks.Delivery != nil {
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					Logf(LogError, "Error while executing packet receipt delivery callback: %v", rec)
				}
			}()
			r.Callbacks.Delivery(r)
		}()
	}
	return true
}

func (r *PacketReceipt) ValidateProof(proof []byte, proofPacket *Packet) bool {
	if len(proof) == ReceiptExplLength {
		proofHash := proof[:HashLengthBytes]
		sig := proof[HashLengthBytes : HashLengthBytes+SigLengthBytes]

		if !bytesEqual(proofHash, r.Hash) || r.Destination == nil || r.Destination.identity == nil {
			return false
		}
		if !r.Destination.identity.Validate(sig, r.Hash) {
			return false
		}
		r.Status = ReceiptDelivered
		r.Proved = true
		r.ConcludedAt = time.Now()
		r.ProofPacket = proofPacket
		if r.Link != nil {
			r.Link.observeRTTSeconds(r.GetRTT())
		}
		if r.Callbacks.Delivery != nil {
			func() {
				defer func() {
					if rec := recover(); rec != nil {
						Logf(LogError, "Error while executing packet receipt delivery callback: %v", rec)
					}
				}()
				r.Callbacks.Delivery(r)
			}()
		}
		return true
	}

	if len(proof) == ReceiptImplLength {
		if r.Destination == nil || r.Destination.identity == nil {
			return false
		}
		sig := proof[:SigLengthBytes]
		if !r.Destination.identity.Validate(sig, r.Hash) {
			return false
		}
		r.Status = ReceiptDelivered
		r.Proved = true
		r.ConcludedAt = time.Now()
		r.ProofPacket = proofPacket
		if r.Link != nil {
			r.Link.observeRTTSeconds(r.GetRTT())
		}
		if r.Callbacks.Delivery != nil {
			func() {
				defer func() {
					if rec := recover(); rec != nil {
						Logf(LogError, "Error while executing packet receipt delivery callback: %v", rec)
					}
				}()
				r.Callbacks.Delivery(r)
			}()
		}
		return true
	}

	return false
}

func (r *PacketReceipt) GetRTT() float64 {
	if r.ConcludedAt.IsZero() {
		return 0
	}
	return r.ConcludedAt.Sub(r.SentAt).Seconds()
}

func (r *PacketReceipt) IsTimedOut() bool {
	return time.Since(r.SentAt).Seconds() > r.Timeout
}

func (r *PacketReceipt) CheckTimeout() {
	if r.Status == ReceiptSent && r.IsTimedOut() {
		if r.Timeout < 0 {
			r.Status = ReceiptCulled
		} else {
			r.Status = ReceiptFailed
		}
		r.ConcludedAt = time.Now()
		if r.Callbacks.Timeout != nil {
			go func() {
				defer func() {
					if rec := recover(); rec != nil {
						Logf(LogError, "Error while executing packet receipt timeout callback: %v", rec)
					}
				}()
				r.Callbacks.Timeout(r)
			}()
		}
	}
}

func (r *PacketReceipt) SetTimeout(timeout float64) {
	r.Timeout = timeout
}

func (r *PacketReceipt) SetDeliveryCallback(cb func(*PacketReceipt)) {
	r.Callbacks.Delivery = cb
}

func (r *PacketReceipt) SetTimeoutCallback(cb func(*PacketReceipt)) {
	r.Callbacks.Timeout = cb
}

// ===== small utilities =====

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
