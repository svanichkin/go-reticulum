package rns

import (
	"bytes"
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
	PacketHEADER_1 = HeaderType1
	PacketHEADER_2 = HeaderType2
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
	PacketREQUEST        = PacketCtxRequest
	PacketRESPONSE       = PacketCtxResponse
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
	PacketHEADER_MAXSIZE = HEADER_MAXSIZE
	PacketMDU            = MDU
	PacketPLAIN_MDU      = MDU
	PacketENCRYPTED_MDU  = computeEncryptedPacketMDU(MDU)
	PacketPlainMDU       = MDU
	PacketEncryptedMDU   = computeEncryptedPacketMDU(MDU)
)

// Python: Packet.TIMEOUT_PER_HOP = Reticulum.DEFAULT_PER_HOP_TIMEOUT
const TimeoutPerHop = float64(DEFAULT_PER_HOP_TIMEOUT)
const PacketTIMEOUT_PER_HOP = TimeoutPerHop

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
func NewPacket(target interface{}, data []byte, packetType byte, context byte, transportType byte, headerType byte, transportID []byte, attachedInterface *Interface, createReceipt bool, contextFlag byte) *Packet {
	if target == nil {
		return &Packet{
			Raw:                append([]byte(nil), data...),
			Packed:             true,
			FromPacked:         true,
			CreateReceipt:      false,
			MTU:                MTU,
			AttachedInterface:  attachedInterface,
			ReceivingInterface: nil,
			RSSI:               nil,
			SNR:                nil,
			Q:                  nil,
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
		if v == nil {
			Log("Cannot create packet for nil link", LogError)
			return nil
		}
		link = v
		dest = v.destination
	default:
		Log(fmt.Sprintf("Unsupported packet target %T", target), LogError)
		return nil
	}

	if dest == nil {
		return nil
	}

	packet := &Packet{
		HeaderType:        headerType,
		PacketType:        packetType,
		Type:              packetType,
		TransportType:     transportType,
		Context:           context,
		ContextFlag:       contextFlag,
		Hops:              0,
		Destination:       dest,
		TransportID:       append([]byte(nil), transportID...),
		Data:              append([]byte(nil), data...),
		CreateReceipt:     createReceipt,
		FromPacked:        false,
		AttachedInterface: attachedInterface,
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
		p.DestinationHash = append([]byte(nil), p.Link.LinkID...)
	} else {
		p.DestinationHash = append([]byte(nil), p.Destination.Hash...)
	}
	header := make([]byte, 0, 64)

	header = append(header, p.Flags)
	header = append(header, byte(p.Hops))

	if p.Context == PacketCtxLRProof {
		if p.Link == nil {
			return errors.New("LRPROOF packet requires link_id")
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
				if p.Link != nil {
					ct, err := p.Link.Encrypt(p.Data)
					if err != nil {
						return err
					}
					p.Ciphertext = ct
				} else if p.Destination != nil && p.Destination.Type == DestinationSINGLE && p.Destination.identity == nil {
					// Python ProofDestination: type SINGLE, but no encryption.
					p.Ciphertext = p.Data
				} else if p.Destination != nil {
					ct := p.Destination.Encrypt(p.Data)
					if ct == nil && p.Data != nil {
						return errors.New("destination encryption failed")
					}
					p.Ciphertext = ct
				} else {
					return errors.New("no destination")
				}
				if p.Destination != nil && len(p.Destination.latestRatchetID) > 0 {
					p.RatchetID = append([]byte{}, p.Destination.latestRatchetID...)
				}
			}
		}

		if p.HeaderType == HeaderType2 {
			if len(p.TransportID) == 0 {
				return errors.New("header type 2 packet must have transportID")
			}
			header = append(header, p.TransportID...)
			header = append(header, p.DestinationHash...)
			if p.PacketType == PacketTypeAnnounce {
				// Announce packets are not encrypted.
				p.Ciphertext = p.Data
			}
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

// Unpack mirrors unpack().
func (p *Packet) Unpack() (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			Logf(LogExtreme, "Received malformed packet, dropping it. The contained exception was: %v", r)
		}
	}()

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
		p.TransportID = append([]byte(nil), p.Raw[2:dstLen+2]...)
		p.DestinationHash = append([]byte(nil), p.Raw[dstLen+2:2*dstLen+2]...)
		p.Context = p.Raw[2*dstLen+2]
		p.Data = append([]byte(nil), p.Raw[2*dstLen+3:]...)
	} else {
		p.TransportID = nil
		p.DestinationHash = append([]byte(nil), p.Raw[2:dstLen+2]...)
		p.Context = p.Raw[dstLen+2]
		p.Data = append([]byte(nil), p.Raw[dstLen+3:]...)
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
	if p.Link != nil {
		if p.Link.Status == LinkClosed {
			Log("Attempt to transmit over a closed link, dropping packet", LogDebug)
			p.Sent = false
			p.Receipt = nil
			return nil
		}
		now := time.Now()
		p.Link.mu.Lock()
		p.Link.lastOutbound = now
		p.Link.tx++
		p.Link.txBytes += uint64(len(p.Data))
		p.Link.mu.Unlock()
	}

	if !p.Packed {
		if err := p.Pack(); err != nil {
			panic(err)
		}
	}

	if Outbound(p) {
		return p.Receipt
	}

	Log("No interfaces could process the outbound packet", LogError)
	p.Sent = false
	p.Receipt = nil
	return nil
}

// Resend mirrors resend().
func (p *Packet) Resend() *PacketReceipt {
	if !p.Sent {
		panicPacketState("Packet was not sent yet")
	}
	if err := p.Pack(); err != nil {
		panic(err)
	}

	if Outbound(p) {
		return p.Receipt
	}

	Log("No interfaces could process the outbound packet", LogError)
	p.Sent = false
	p.Receipt = nil
	return nil
}

type ProofDestination = Destination

// Prove / ProofDestination

func (p *Packet) Prove(dest *Destination) {
	if p.FromPacked && p.Destination != nil && p.Destination.identity != nil && p.Destination.identity.prv != nil {
		p.Destination.identity.Prove(p, dest)
	} else if p.FromPacked && p.Link != nil {
		p.Link.ProvePacket(p)
	} else {
		Log("Could not prove packet associated with neither a destination nor a link", LogError)
	}
}

func (p *Packet) GenerateProofDestination() *ProofDestination {
	// ProofDestination mirrors rns/packet.py: it is type SINGLE but performs no encryption.
	return &Destination{
		Type:      DestinationSINGLE,
		Direction: DestinationOUT,
		Hash:      append([]byte{}, p.GetHash()[:ReticulumTruncatedHashLength/8]...),
	}
}

func (p *Packet) ValidateProofPacket(proof *Packet) bool {
	return p.Receipt.ValidateProofPacket(proof)
}

func (p *Packet) ValidateProof(proof []byte) bool {
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
	hashable := []byte{p.Raw[0] & 0x0F}
	if p.HeaderType == HeaderType2 {
		hashable = append(hashable, p.Raw[truncatedHashBytes+2:]...)
	} else {
		hashable = append(hashable, p.Raw[2:]...)
	}
	return hashable
}

// Metrics

func (p *Packet) GetRSSI() *float64 {
	if p.RSSI != nil {
		return p.RSSI
	}
	if Owner != nil {
		return Owner.GetPacketRSSI(p.PacketHash)
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
	if Owner != nil {
		return Owner.GetPacketSNR(p.PacketHash)
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
	if Owner != nil {
		return Owner.GetPacketQ(p.PacketHash)
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
		rttTimeout := p.Link.RTT.Seconds() * p.Link.TrafficTimeoutFactor
		if rttTimeout < linkTrafficTimeoutMin.Seconds() {
			rttTimeout = linkTrafficTimeoutMin.Seconds()
		}
		r.Timeout = rttTimeout
	} else {
		base := FirstHopTimeout(p.Destination.Hash).Seconds()
		if Owner != nil {
			base = Owner.GetFirstHopTimeout(p.Destination.Hash)
		}
		hops := HopsTo(p.Destination.Hash)
		r.Timeout = base + TimeoutPerHop*float64(hops)
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
	proofHash := proof
	if len(proofHash) > HashLengthBytes {
		proofHash = proofHash[:HashLengthBytes]
	}
	sig := proof[0:0]
	if len(proof) > HashLengthBytes {
		end := HashLengthBytes + SigLengthBytes
		if len(proof) < end {
			sig = proof[HashLengthBytes:]
		} else {
			sig = proof[HashLengthBytes:end]
		}
	}

	if !bytes.Equal(proofHash, r.Hash) {
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

	if r.Callbacks.Delivery != nil {
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					Logf(LogError, "An error occurred while evaluating external delivery callback for %v", link)
					Logf(LogError, "The contained exception was: %v", rec)
					TraceException(rec)
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

		if !bytes.Equal(proofHash, r.Hash) || r.Destination == nil || r.Destination.identity == nil {
			return false
		}
		if !r.Destination.identity.Validate(sig, r.Hash) {
			return false
		}
		r.Status = ReceiptDelivered
		r.Proved = true
		r.ConcludedAt = time.Now()
		r.ProofPacket = proofPacket
		if r.Callbacks.Delivery != nil {
			func() {
				defer func() {
					if rec := recover(); rec != nil {
						Logf(LogError, "Error while executing proof validated callback. The contained exception was: %v", rec)
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
		if r.Callbacks.Delivery != nil {
			func() {
				defer func() {
					if rec := recover(); rec != nil {
						Logf(LogError, "Error while executing proof validated callback. The contained exception was: %v", rec)
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
	if r.SentAt.IsZero() || r.ConcludedAt.IsZero() {
		panic("packet receipt RTT is unavailable before conclusion")
	}
	return r.ConcludedAt.Sub(r.SentAt).Seconds()
}

func (r *PacketReceipt) IsTimedOut() bool {
	return time.Since(r.SentAt).Seconds() > r.Timeout
}

func (r *PacketReceipt) CheckTimeout() {
	if r.Status == ReceiptSent && r.IsTimedOut() {
		if r.Timeout == -1 {
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
