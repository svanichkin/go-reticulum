package rns

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	umsgpack "github.com/svanichkin/go-reticulum/rns/vendor"
)

// Porting notes: this file assumes these types/values already exist:
//  - type Link
//  - type Packet
//  - type Identity
//  - var Reticulum struct{ ResourcePath string; HEADER_MAXSIZE, IFAC_MIN_SIZE int }
//  - Packet contexts: PacketRESOURCE, PacketRESOURCE_ADV, PacketRESOURCE_REQ, ...
//  - Transport.cache, Transport.cache_request, etc.
//  - a msgpack library

const (
	// window
	ResourceWindow            = 4
	ResourceWindowMin         = 2
	ResourceWindowMaxSlow     = 10
	ResourceWindowMaxVerySlow = 4
	ResourceWindowMaxFast     = 75
	ResourceWindowMax         = ResourceWindowMaxFast

	ResourceFastRateThreshold         = ResourceWindowMaxSlow - ResourceWindow - 2
	VerySlowRateThreshold             = 2
	RateFast                  float64 = (50 * 1000) / 8
	RateVerySlow              float64 = (2 * 1000) / 8

	ResourceWindowFlexibility = 4

	MapHashLen           = 4
	RandomHashSize       = 4
	MaxEfficientSize     = 1*1024*1024 - 1
	ResponseMaxGraceTime = 10 * time.Second
	MetadataMaxSize      = 16*1024*1024 - 1
	AutoCompressMaxSize  = 64 * 1024 * 1024

	PartTimeoutFactor         = 4
	PartTimeoutFactorAfterRTT = 2
	ProofTimeoutFactor        = 3
	MaxRetries                = 16
	MaxAdvRetries             = 4
	SenderGraceTime           = 10.0
	ProcessingGrace           = 1.0
	RetryGraceTime            = 0.25
	PerRetryDelay             = 0.5
	WatchdogMaxSleep          = 1 * time.Second

	HashmapNotExhausted byte = 0x00
	HashmapExhausted    byte = 0xFF

	// statuses
	ResourceNone          = 0x00
	ResourceQueued        = 0x01
	ResourceAdvertised    = 0x02
	ResourceTransferring  = 0x03
	ResourceAwaitingProof = 0x04
	ResourceAssembling    = 0x05
	ResourceComplete      = 0x06
	ResourceFailed        = 0x07
	ResourceCorrupt       = 0x08
	ResourceRejected      = 0x00
)

// Resource is a port of the Python Resource class.
type Resource struct {
	// immutable resource parameters
	flags            byte
	size             int
	totalSize        int
	uncompressedSize int
	hash             []byte
	originalHash     []byte
	randomHash       []byte
	hashmapRaw       []byte
	hashmap          [][]byte
	hashmapHeight    int
	expectedProof    []byte
	encr, comp       bool
	initiator        bool
	hasMetadata      bool
	split            bool
	segmentIndex     int
	totalSegments    int
	requestID        []byte
	isResponse       bool

	link *Link
	// Python parity: HASHMAP_MAX_LEN/COLLISION_GUARD_SIZE depend on link MDU.
	hashmapMaxLen     int
	collisionGuardLen int

	// SDU and parts
	sdu                   int
	totalParts            int
	parts                 [][]byte // payload only, without headers
	outgoingParts         []*resourcePart
	outgoingPartByMapHash map[string]*Packet
	sentParts             int
	receivedCount         int
	outstanding           int

	// window
	window            int
	windowMax         int
	windowMin         int
	windowFlexibility int
	waitingForHMU     bool
	receivingPart     bool
	consecutiveHeight int
	receiverMinHeight int

	// timing/RTT
	timeout             float64
	timeoutFactor       float64
	partTimeoutFactor   float64
	senderGraceTime     float64
	lastActivity        time.Time
	startedTransferring time.Time
	lastPartSent        time.Time
	retriesLeft         int
	maxRetries          int
	maxAdvRetries       int
	hmuRetryOK          bool
	watchdogLock        bool
	watchdogJobID       int

	rtt                float64
	rttRxBytes         int
	rttRxBytesAtReq    int
	reqSent            time.Time
	reqResp            time.Time
	reqSentBytes       int
	reqRespRTTRate     float64
	reqDataRTTRate     float64
	eifr               float64
	previousEIFR       float64
	fastRateRounds     int
	verySlowRateRounds int

	// auto-compression
	autoCompress       bool
	autoCompressLimit  int
	autoCompressOption any

	// files/paths
	storagePath     string
	metaStoragePath string
	inputFile       *os.File
	tempInputPath   string

	metadata     []byte
	metadataSize int
	metadataMap  map[string]any
	dataFile     *os.File

	status byte

	callback         func(*Resource)
	progressCallback func(*Resource)

	advPacket       *Packet
	advSent         time.Time
	lastRequestData []byte

	// helper fields
	assemblyLock  bool
	preparingNext bool
	nextSegment   *Resource
	receiveLock   sync.Mutex
	watchdogMu    sync.Mutex
}

// -------- ResourceAdvertisement --------

type ResourceAdvertisement struct {
	T    int    // transfer size
	D    int    // data size
	N    int    // num parts
	H    []byte // hash
	R    []byte // random hash
	O    []byte // original hash
	M    []byte // hashmap bytes
	F    byte   // flags
	I    int    // segment index
	L    int    // total segments
	Q    []byte // request id
	Link *Link

	E bool // encrypted
	C bool // compressed
	S bool // split
	U bool // is request
	P bool // is response
	X bool // has metadata
}

// helpers mirroring Python ResourceAdvertisement static methods

func unpackResourceAdvertisement(packet *Packet) *ResourceAdvertisement {
	if packet == nil {
		return nil
	}
	data := packet.Plaintext
	if len(data) == 0 {
		data = packet.Data
		if len(data) == 0 {
			return nil
		}
	}
	adv, err := ResourceAdvertisementUnpack(data)
	if err != nil {
		return nil
	}
	if adv.Link == nil {
		adv.Link = packet.Link
	}
	return adv
}

func ResourceAdvertisementIsRequest(packet *Packet) bool {
	adv := unpackResourceAdvertisement(packet)
	if adv == nil {
		return false
	}
	return adv.IsRequest()
}

func ResourceAdvertisementIsResponse(packet *Packet) bool {
	adv := unpackResourceAdvertisement(packet)
	if adv == nil {
		return false
	}
	return adv.IsResponse()
}

func ResourceAdvertisementReadRequestID(packet *Packet) []byte {
	adv := unpackResourceAdvertisement(packet)
	if adv == nil {
		return nil
	}
	return adv.RequestID()
}

func ResourceAdvertisementReadTransferSize(packet *Packet) int {
	if adv := unpackResourceAdvertisement(packet); adv != nil {
		return adv.TransferSize()
	}
	return 0
}

func ResourceAdvertisementReadDataSize(packet *Packet) int {
	if adv := unpackResourceAdvertisement(packet); adv != nil {
		return adv.DataSize()
	}
	return 0
}

const (
	AdvOverhead = 134
)

var (
	// HASHMAP_MAX_LEN depends on MTU. In Python: floor((Link.MDU-OVERHEAD)/MAPHASH_LEN)
	// In Go this can be computed dynamically from link.MDU during pack(), but we keep it
	// global for now (works fine when MDU is fixed).
	HashmapMaxLen = resourceHashmapCapacity(LinkMDU)

	CollisionGuardSize = 2*ResourceWindowMax + HashmapMaxLen
)

func resourceHashmapCapacity(linkMDU int) int {
	usable := linkMDU - AdvOverhead
	if usable <= 0 {
		return 0
	}
	return int(math.Floor(float64(usable) / float64(MapHashLen)))
}

func resourceHashmapCapacityForLink(link *Link) int {
	if link == nil {
		return resourceHashmapCapacity(LinkMDU)
	}
	if link.MDU > 0 {
		return resourceHashmapCapacity(link.MDU)
	}
	return resourceHashmapCapacity(LinkMDU)
}

func setResourceSizing(hashLen int) {
	HashmapMaxLen = hashLen
	CollisionGuardSize = 2*ResourceWindowMax + HashmapMaxLen
}

type resourcePart struct {
	packet  *Packet
	mapHash []byte
}

func init() {
	if HashmapMaxLen <= 0 {
		// Avoid panicking on import; align with Python behaviour where this would
		// just make resources unusable until configuration is fixed.
		Log("Configured MTU is too small to include any map hashes in resource advertisements; forcing HashmapMaxLen=1", LogError)
		HashmapMaxLen = 1
		CollisionGuardSize = 2*ResourceWindowMax + HashmapMaxLen
	}
}

// ---------------- static operations ----------------

// ResourceReject mirrors Resource.reject().
func ResourceReject(advPkt *Packet) {
	defer func() {
		if r := recover(); r != nil {
			Log(fmt.Sprintf("panic in ResourceReject: %v", r), LOG_ERROR)
		}
	}()
	adv, err := ResourceAdvertisementUnpack(advPkt.Plaintext)
	if err != nil {
		Log(fmt.Sprintf("Error rejecting resource: %v", err), LOG_ERROR)
		return
	}
	adv.Link = advPkt.Link
	resHash := adv.H
	reject := NewPacket(
		advPkt.Link,
		resHash,
		WithPacketContext(PacketCONTEXT_RESOURCE_RCL),
	)
	_ = reject.Send()
}

// ResourceAccept mirrors Resource.accept().
func ResourceAccept(advPkt *Packet, cb func(*Resource), progCb func(*Resource), reqID []byte) (*Resource, error) {
	adv, err := ResourceAdvertisementUnpack(advPkt.Plaintext)
	if err != nil {
		Log("Could not decode resource advertisement, dropping resource", LOG_DEBUG)
		return nil, err
	}
	adv.Link = advPkt.Link

	res := &Resource{
		flags:             adv.F,
		size:              adv.T,
		totalSize:         adv.D,
		uncompressedSize:  adv.D,
		hash:              adv.H,
		originalHash:      adv.O,
		randomHash:        adv.R,
		hashmapRaw:        adv.M,
		encr:              (adv.F & 0x01) != 0,
		comp:              ((adv.F >> 1) & 0x01) != 0,
		initiator:         false,
		callback:          cb,
		progressCallback:  progCb,
		link:              advPkt.Link,
		segmentIndex:      adv.I,
		totalSegments:     adv.L,
		requestID:         reqID,
		status:            ResourceTransferring,
		window:            ResourceWindow,
		windowMax:         ResourceWindowMaxSlow,
		windowMin:         ResourceWindowMin,
		windowFlexibility: ResourceWindowFlexibility,
		consecutiveHeight: -1,
	}

	res.totalParts = int(math.Ceil(float64(res.size) / float64(res.sduValue())))
	res.sdu = res.sduValue()
	res.receivedCount = 0
	res.outstanding = 0
	res.parts = make([][]byte, res.totalParts)
	res.lastActivity = time.Now()
	res.startedTransferring = res.lastActivity
	if inst := GetInstance(); inst != nil && inst.ResourcePath != "" {
		res.storagePath = filepath.Join(inst.ResourcePath, hex.EncodeToString(res.originalHash))
	} else if Owner != nil && Owner.ResourcePath != "" {
		res.storagePath = filepath.Join(Owner.ResourcePath, hex.EncodeToString(res.originalHash))
	} else {
		res.storagePath = filepath.Join(os.TempDir(), "rns_resources", hex.EncodeToString(res.originalHash))
	}
	res.metaStoragePath = res.storagePath + ".meta"

	// split / metadata
	res.split = adv.L > 1
	res.hasMetadata = adv.X

	res.hashmap = make([][]byte, res.totalParts)
	res.hashmapHeight = 0
	res.waitingForHMU = false
	res.receivingPart = false

	res.hashmapMaxLen = resourceHashmapCapacityForLink(res.link)
	if res.hashmapMaxLen <= 0 {
		return nil, fmt.Errorf("the configured MTU is too small to include any map hashes in resource advertisements")
	}
	res.collisionGuardLen = 2*ResourceWindowMax + res.hashmapMaxLen

	prevWin := advPkt.Link.GetLastResourceWindow()
	prevEIFR := advPkt.Link.GetLastResourceEIFR()
	if prevWin > 0 {
		res.window = prevWin
	}
	if prevEIFR > 0 {
		res.previousEIFR = prevEIFR
	}

	if advPkt.Link.HasIncomingResource(res.hash) {
		Log("Ignoring resource advertisement, already transferring "+PrettyHex(res.hash), LOG_DEBUG)
		return nil, nil
	}

	advPkt.Link.RegisterIncomingResource(res)
	Log(fmt.Sprintf(
		"Accepting resource advertisement for %s. Transfer size %s in %d parts.",
		PrettyHex(res.hash), PrettySize(float64(res.size)), res.totalParts,
	), LOG_DEBUG)

	if advPkt.Link.callbacks.ResourceStarted != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					Log(fmt.Sprintf("Error in resource started callback: %v", r), LOG_ERROR)
				}
			}()
			advPkt.Link.callbacks.ResourceStarted(res)
		}()
	}

	res.HashmapUpdate(0, res.hashmapRaw)
	res.WatchdogJob()
	return res, nil
}

// ---------------- outgoing resource constructor ----------------

func NewResource(
	data []byte,
	file *os.File,
	link *Link,
	metadata any,
	advertise bool,
	autoCompress any, // bool or int
	cb func(*Resource),
	progCb func(*Resource),
	timeout *float64,
	segmentIndex int,
	originalHash []byte,
	requestID []byte,
	isResponse bool,
	sentMetadataSize int,
) (*Resource, error) {

	res := &Resource{
		link:               link,
		assemblyLock:       false,
		preparingNext:      false,
		nextSegment:        nil,
		hasMetadata:        sentMetadataSize > 0,
		metadataSize:       sentMetadataSize,
		maxRetries:         MaxRetries,
		maxAdvRetries:      MaxAdvRetries,
		retriesLeft:        MaxRetries,
		timeoutFactor:      link.TrafficTimeoutFactor,
		partTimeoutFactor:  PartTimeoutFactor,
		senderGraceTime:    SenderGraceTime,
		hmuRetryOK:         false,
		progressCallback:   progCb,
		requestID:          requestID,
		isResponse:         isResponse,
		autoCompressLimit:  AutoCompressMaxSize,
		autoCompressOption: autoCompress,
		receiverMinHeight:  0,
		consecutiveHeight:  -1,
	}

	// Python parity: if data is large bytes, wrap in a temp file so we can segment.
	if data != nil && file == nil && res.metadataSize+len(data) > MaxEfficientSize {
		tmp, err := os.CreateTemp("", "rns_resource_*")
		if err != nil {
			return nil, err
		}
		if _, err := tmp.Write(data); err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
			return nil, err
		}
		if _, err := tmp.Seek(0, 0); err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
			return nil, err
		}
		res.tempInputPath = tmp.Name()
		file = tmp
		data = nil
	}

	// metadata
	if metadata != nil && sentMetadataSize == 0 {
		packed, err := msgpackMarshal(metadata)
		if err != nil {
			return nil, err
		}
		if len(packed) > MetadataMaxSize {
			return nil, errors.New("Resource metadata size exceeded")
		}
		// first 3 bytes are the length
		sizeBytes := []byte{0, 0, 0}
		size := len(packed)
		sizeBytes[0] = byte((size >> 16) & 0xff)
		sizeBytes[1] = byte((size >> 8) & 0xff)
		sizeBytes[2] = byte(size & 0xff)
		res.metadata = append(sizeBytes, packed...)
		res.metadataSize = len(res.metadata)
		res.hasMetadata = true
	} else if sentMetadataSize > 0 {
		res.hasMetadata = true
	}

	// compute total size and segmentation
	var totalSize int
	var segmentData []byte
	var inputSize int
	var err error

	if file != nil {
		info, errStat := file.Stat()
		if errStat != nil {
			return nil, errStat
		}
		inputSize = int(info.Size())
	} else if data != nil {
		inputSize = len(data)
	}

	totalSize = inputSize + res.metadataSize
	res.totalSize = totalSize

	if file != nil {
		if totalSize <= MaxEfficientSize {
			res.totalSegments = 1
			res.segmentIndex = 1
			res.split = false
			buf := make([]byte, inputSize)
			_, err = file.Read(buf)
			if err != nil && err.Error() != "EOF" {
				return nil, err
			}
			segmentData = buf
			_ = file.Close()
			if res.tempInputPath != "" {
				_ = os.Remove(res.tempInputPath)
				res.tempInputPath = ""
			}
		} else {
			res.totalSegments = (totalSize-1)/MaxEfficientSize + 1
			if segmentIndex == 0 {
				segmentIndex = 1
			}
			res.segmentIndex = segmentIndex
			res.split = true

			seekIndex := segmentIndex - 1
			firstReadSize := MaxEfficientSize - res.metadataSize
			var seekPos int
			var readSize int
			if segmentIndex == 1 {
				seekPos = 0
				readSize = firstReadSize
			} else {
				seekPos = firstReadSize + (seekIndex-1)*MaxEfficientSize
				readSize = MaxEfficientSize
			}

			_, err = file.Seek(int64(seekPos), 0)
			if err != nil {
				return nil, err
			}
			buf := make([]byte, readSize)
			n, errRead := file.Read(buf)
			if errRead != nil && errRead.Error() != "EOF" {
				return nil, errRead
			}
			segmentData = buf[:n]
			res.inputFile = file
		}
	} else if data != nil {
		totalSize = len(data) + res.metadataSize
		res.totalSize = totalSize
		res.totalSegments = 1
		res.segmentIndex = 1
		res.split = false
		segmentData = data
	} else {
		// incoming resource (accept)
	}

	// add metadata to the segment
	var fullData []byte
	if segmentData != nil {
		if res.hasMetadata && res.segmentIndex == 1 {
			fullData = append(res.metadata, segmentData...)
		} else {
			fullData = segmentData
		}
	}

	res.status = ResourceNone
	res.sdu = res.sduValue()
	res.timeout = float64(link.RTT) * link.TrafficTimeoutFactor
	if timeout != nil {
		res.timeout = *timeout
	}

	res.hashmapMaxLen = resourceHashmapCapacityForLink(res.link)
	if res.hashmapMaxLen <= 0 {
		return nil, fmt.Errorf("the configured MTU is too small to include any map hashes in resource advertisements")
	}
	res.collisionGuardLen = 2*ResourceWindowMax + res.hashmapMaxLen

	if fullData != nil {
		res.initiator = true
		res.callback = cb

		// auto-compression
		uncompressed := fullData
		res.uncompressedSize = len(uncompressed)

		switch v := autoCompress.(type) {
		case bool:
			res.autoCompress = v
		case int:
			res.autoCompress = true
			res.autoCompressLimit = v
		default:
			return nil, fmt.Errorf("invalid type %T for auto_compress", autoCompress)
		}

		var toSend []byte
		if res.autoCompress && len(uncompressed) <= res.autoCompressLimit {
			Log("Compressing resource data...", LOG_EXTREME)
			compressed, err := bz2Compress(uncompressed)
			if err != nil {
				return nil, err
			}
			res.comp = len(compressed) < len(uncompressed)
			if res.comp {
				toSend = compressed
				Log(fmt.Sprintf("Compression saved %d bytes, sending compressed",
					len(uncompressed)-len(compressed)), LOG_EXTREME)
			} else {
				toSend = uncompressed
				Log("Compression did not decrease size, sending uncompressed", LOG_EXTREME)
			}
		} else {
			res.comp = false
			toSend = uncompressed
		}

		// random-hash prefix
		// Python uses two random values:
		// - a 4-byte prefix embedded in the encrypted stream (stripped by receiver)
		// - a 4-byte random hash used for map hashes and resource hash validation
		streamPrefix := IdentityGetRandomHash()[:RandomHashSize]
		res.randomHash = IdentityGetRandomHash()[:RandomHashSize]

		// Resource hash and expected proof are computed over the decrypted payload
		// (without stream prefix), plus randomHash.
		hashInput := append(append([]byte(nil), toSend...), res.randomHash...)
		res.hash = FullHash(hashInput)
		proofInput := append(append([]byte(nil), toSend...), res.hash...)
		res.expectedProof = FullHash(proofInput)

		payload := append(streamPrefix, toSend...)

		// link encryption
		enc, err := link.Encrypt(payload)
		if err != nil {
			return nil, err
		}
		res.encr = true
		res.size = len(enc)

		if originalHash != nil {
			res.originalHash = originalHash
		} else {
			res.originalHash = res.hash
		}

		// parts and hashmap
		hashmapEntries := int(math.Ceil(float64(res.size) / float64(res.sdu)))
		res.totalParts = hashmapEntries
		for {
			ok, err := res.buildHashmap(enc, hashmapEntries)
			if err != nil {
				return nil, err
			}
			if ok {
				break
			}
			// retry randomHash on collisions
		}

		if advertise {
			res.Advertise()
		}
	}

	return res, nil
}

// buildHashmap mirrors the collision_guard_list loop.
func (r *Resource) buildHashmap(enc []byte, entries int) (bool, error) {
	hashmapStart := time.Now()
	if len(r.randomHash) == 0 {
		r.randomHash = IdentityGetRandomHash()[:RandomHashSize]
	}
	r.hashmap = make([][]byte, 0, entries)
	r.outgoingParts = make([]*resourcePart, 0, entries)
	r.outgoingPartByMapHash = make(map[string]*Packet, entries)
	var collisionGuard [][]byte
	collisionGuardLen := r.collisionGuardLen
	if collisionGuardLen <= 0 {
		collisionGuardLen = CollisionGuardSize
	}

	ok := true
	for i := 0; i < entries; i++ {
		start := i * r.sdu
		end := start + r.sdu
		if end > len(enc) {
			end = len(enc)
		}
		chunk := enc[start:end]
		mapHash := r.getMapHash(chunk)

		// collision check
		for _, h := range collisionGuard {
			if bytes.Equal(h, mapHash) {
				Log("Found hash collision in resource map, remapping...", LOG_DEBUG)
				ok = false
				break
			}
		}
		if !ok {
			break
		}
		collisionGuard = append(collisionGuard, mapHash)
		if len(collisionGuard) > collisionGuardLen {
			collisionGuard = collisionGuard[1:]
		}
		partPkt := NewPacket(
			r.link,
			chunk,
			WithPacketContext(PacketCONTEXT_RESOURCE),
		)
		if partPkt == nil {
			return false, errors.New("failed to create resource part packet")
		}
		partPkt.MapHash = mapHash
		if err := partPkt.Pack(); err != nil {
			return false, err
		}

		r.outgoingParts = append(r.outgoingParts, &resourcePart{
			packet:  partPkt,
			mapHash: mapHash,
		})
		r.hashmap = append(r.hashmap, mapHash)
		r.outgoingPartByMapHash[string(mapHash)] = partPkt
	}
	r.hashmapHeight = len(r.hashmap)
	Log(fmt.Sprintf("Hashmap computation concluded in %.3f seconds",
		time.Since(hashmapStart).Seconds()), LOG_EXTREME)
	return ok, nil
}

// sduValue computes SDU like in __init__.
func (r *Resource) sduValue() int {
	mtu := 0
	if r.link != nil {
		mtu = r.link.MTU
	}
	if mtu <= 0 {
		mtu = defaultLinkMTU()
	}
	if mtu > 0 {
		sdu := mtu - HEADER_MAXSIZE - IFAC_MIN_SIZE
		if sdu > 0 {
			return sdu
		}
	}
	if r.link != nil && r.link.MDU > 0 {
		return r.link.MDU
	}
	if PacketMDU > 0 {
		return PacketMDU
	}
	return 1
}

func (r *Resource) getMapHash(data []byte) []byte {
	// Important: avoid mutating the underlying buffer of data (data may be a slice
	// of a shared backing array, eg. the encrypted resource buffer).
	hashInput := append(append([]byte(nil), data...), r.randomHash...)
	h := FullHash(hashInput)
	return h[:MapHashLen]
}

// ---------------- Hashmap update ----------------

func (r *Resource) HashmapUpdatePacket(plaintext []byte) {
	if r.status == ResourceFailed {
		return
	}
	r.lastActivity = time.Now()
	r.retriesLeft = r.maxRetries

	off := sha256Bits / 8
	if len(plaintext) <= off {
		return
	}
	update, err := msgpackUnmarshal(plaintext[off:])
	if err != nil {
		return
	}
	if len(update) < 2 {
		return
	}
	seg, ok := asIntValue(update[0])
	if !ok {
		return
	}
	hm, ok := asBytesValue(update[1])
	if !ok {
		return
	}
	r.HashmapUpdate(seg, hm)
}

func (r *Resource) HashmapUpdate(segment int, hashmap []byte) {
	if r.status == ResourceFailed {
		return
	}
	r.status = ResourceTransferring
	segLen := r.hashmapMaxLen
	if segLen <= 0 {
		segLen = HashmapMaxLen
	}
	hashes := len(hashmap) / MapHashLen
	for i := 0; i < hashes; i++ {
		idx := i + segment*segLen
		if idx < 0 || idx >= len(r.hashmap) {
			break
		}
		if r.hashmap[idx] == nil {
			r.hashmapHeight++
		}
		r.hashmap[idx] = hashmap[i*MapHashLen : (i+1)*MapHashLen]
	}
	r.waitingForHMU = false
	r.RequestNext()
}

// ---------------- advertise ----------------

func (r *Resource) Advertise() {
	go r.advertiseJob()
	if r.segmentIndex < r.totalSegments {
		go r.prepareNextSegment()
	}
}

func (r *Resource) resendAdvertisement() {
	adv := NewResourceAdvertisementFromResource(r)
	pkt := NewPacket(
		r.link,
		adv.Pack(0),
		WithPacketContext(PacketCONTEXT_RESOURCE_ADV),
	)
	if pkt == nil {
		Log("Could not build resource advertisement packet", LOG_ERROR)
		r.Cancel()
		return
	}
	pkt.Send()
	if !pkt.Sent {
		Log("Could not resend advertisement packet, cancelling resource", LOG_VERBOSE)
		r.Cancel()
		return
	}

	now := time.Now()
	r.lastActivity = now
	r.advSent = now
	r.advPacket = pkt
	Log("Sent resource advertisement for "+PrettyHex(r.hash), LOG_EXTREME)
}

func (r *Resource) advertiseJob() {
	adv := &ResourceAdvertisement{ /* filled below */ }
	*adv = NewResourceAdvertisementFromResource(r)
	pkt := NewPacket(
		r.link,
		adv.Pack(0),
		WithPacketContext(PacketCONTEXT_RESOURCE_ADV),
	)
	if pkt == nil {
		Log("Could not build resource advertisement packet", LOG_ERROR)
		r.Cancel()
		return
	}
	r.status = ResourceQueued

	for !r.link.ReadyForNewResource() {
		r.status = ResourceQueued
		time.Sleep(250 * time.Millisecond)
	}

	// Register before sending so part requests that race in right after the
	// advertisement can be resolved to this resource.
	r.link.RegisterOutgoingResource(r)

	now := time.Now()
	r.lastActivity = now
	r.startedTransferring = now
	r.advSent = now
	r.rtt = 0
	r.status = ResourceAdvertised
	r.retriesLeft = r.maxAdvRetries
	r.advPacket = pkt

	pkt.Send()
	if !pkt.Sent {
		Log("Could not advertise resource, transport refused packet", LOG_ERROR)
		r.link.CancelOutgoingResource(r)
		r.Cancel()
		return
	}
	Log("Sent resource advertisement for "+PrettyHex(r.hash), LOG_EXTREME)
	r.WatchdogJob()
}

// ---------------- EIFR ----------------

func (r *Resource) updateEIFR() {
	var rtt float64
	if r.rtt == 0 {
		rtt = r.link.RTT.Seconds()
	} else {
		rtt = r.rtt
	}
	var expected float64
	if r.reqDataRTTRate != 0 {
		expected = r.reqDataRTTRate * 8
	} else if r.previousEIFR != 0 {
		expected = r.previousEIFR
	} else {
		expected = float64(r.link.EstablishmentCost*8) / rtt
	}
	r.eifr = expected
	r.link.ExpectedRate = expected
}

// ---------------- watchdog ----------------

func (r *Resource) WatchdogJob() {
	go r.watchdog()
}

func (r *Resource) watchdog() {
	r.watchdogMu.Lock()
	r.watchdogJobID++
	jobID := r.watchdogJobID
	r.watchdogMu.Unlock()

	for r.status < ResourceAssembling && jobID == r.watchdogJobID {
		for r.watchdogLock {
			time.Sleep(25 * time.Millisecond)
		}

		var sleepTime time.Duration
		now := time.Now()

		switch r.status {
		case ResourceAdvertised:
			exp := r.advSentTime().Add(time.Duration(r.timeout*float64(time.Second)) +
				time.Duration(ProcessingGrace*float64(time.Second)))
			if now.After(exp) {
				if r.retriesLeft <= 0 {
					Log("Resource transfer timeout after sending advertisement", LOG_DEBUG)
					r.Cancel()
					sleepTime = time.Millisecond
				} else {
					Log("No part requests received, retrying resource advertisement...", LOG_DEBUG)
					r.retriesLeft--
					r.resendAdvertisement()
					sleepTime = time.Millisecond
				}
			} else {
				sleepTime = exp.Sub(now)
			}

		case ResourceTransferring:
			if !r.initiator {
				retriesUsed := r.maxRetries - r.retriesLeft
				extraWait := time.Duration(float64(retriesUsed) * PerRetryDelay * float64(time.Second))
				r.updateEIFR()
				expectedTOF := float64(r.outstanding*r.sdu*8) / r.eifr
				var base time.Time
				if r.reqRespRTTRate != 0 {
					base = r.lastActivity.Add(time.Duration(
						r.partTimeoutFactor*expectedTOF*float64(time.Second) +
							RetryGraceTime*float64(time.Second)))
				} else {
					base = r.lastActivity.Add(time.Duration(
						r.partTimeoutFactor*((3*float64(r.sdu))/r.eifr)*float64(time.Second) +
							RetryGraceTime*float64(time.Second)))
				}
				exp := base.Add(extraWait)
				if now.After(exp) {
					if r.retriesLeft > 0 {
						ms := ""
						if r.outstanding != 1 {
							ms = "s"
						}
						Log(fmt.Sprintf(
							"Timed out waiting for %d part%s, requesting retry on %v",
							r.outstanding, ms, r,
						), LOG_DEBUG)
						if r.window > r.windowMin {
							r.window--
							if r.windowMax > r.windowMin {
								r.windowMax--
								if (r.windowMax - r.window) > (r.windowFlexibility - 1) {
									r.windowMax--
								}
							}
						}
						r.retriesLeft--
						r.waitingForHMU = false
						r.RequestNext()
						sleepTime = time.Millisecond
					} else {
						r.Cancel()
						sleepTime = time.Millisecond
					}
				} else {
					sleepTime = exp.Sub(now)
				}
			} else {
				maxExtra := time.Duration(0)
				for rtry := 0; rtry < MaxRetries; rtry++ {
					maxExtra += time.Duration(float64(rtry+1) * PerRetryDelay * float64(time.Second))
				}
				maxWait := time.Duration(r.rtt*float64(r.timeoutFactor)*float64(time.Second)*float64(r.maxRetries)) +
					time.Duration(SenderGraceTime*float64(time.Second)) + maxExtra
				exp := r.lastActivity.Add(maxWait)
				if now.After(exp) {
					Log("Resource timed out waiting for part requests", LOG_DEBUG)
					r.Cancel()
					sleepTime = time.Millisecond
				} else {
					sleepTime = exp.Sub(now)
				}
			}

		case ResourceAwaitingProof:
			r.timeoutFactor = ProofTimeoutFactor
			exp := r.lastPartSent.Add(
				time.Duration(r.rtt*r.timeoutFactor*float64(time.Second) + SenderGraceTime*float64(time.Second)),
			)
			if now.After(exp) {
				if r.retriesLeft <= 0 {
					Log("Resource timed out waiting for proof", LOG_DEBUG)
					r.Cancel()
					sleepTime = time.Millisecond
				} else {
					Log("All parts sent, but no resource proof received, querying network cache...", LOG_DEBUG)
					r.retriesLeft--
					expectedData := append(r.hash, r.expectedProof...)
					p := NewPacket(
						r.link,
						expectedData,
						WithPacketType(PacketTypeProof),
						WithPacketContext(PacketCONTEXT_RESOURCE_PRF),
					)
					p.Pack()
					CacheRequest(p.PacketHash, r.link)
					r.lastPartSent = time.Now()
					sleepTime = time.Millisecond
				}
			} else {
				sleepTime = exp.Sub(now)
			}

		case ResourceRejected:
			sleepTime = time.Millisecond
		default:
			sleepTime = WatchdogMaxSleep
		}

		if sleepTime < 0 {
			Log("Timing error, cancelling resource transfer.", LOG_ERROR)
			r.Cancel()
			sleepTime = time.Millisecond
		}
		if sleepTime == 0 {
			sleepTime = time.Millisecond
		}
		if sleepTime > WatchdogMaxSleep {
			sleepTime = WatchdogMaxSleep
		}
		time.Sleep(sleepTime)
	}
}

func (r *Resource) advSentTime() time.Time {
	if !r.advSent.IsZero() {
		return r.advSent
	}
	return r.startedTransferring
}

// ---------------- assemble ----------------

func (r *Resource) Assemble() {
	if r.status == ResourceFailed {
		return
	}

	concluded := false
	defer func() {
		if !concluded && r.link != nil {
			r.link.ResourceConcluded(r)
		}
	}()

	r.status = ResourceAssembling
	stream := bytes.Join(r.parts, nil)

	var data []byte
	if r.encr {
		var err error
		data, err = r.link.Decrypt(stream)
		if err != nil {
			r.status = ResourceCorrupt
			return
		}
	} else {
		data = stream
	}
	if len(data) < RandomHashSize {
		r.status = ResourceCorrupt
		return
	}
	data = data[RandomHashSize:]

	if r.comp {
		var err error
		data, err = bz2Decompress(data)
		if err != nil {
			Log(fmt.Sprintf("Error while bz2 decompress resource: %v", err), LOG_ERROR)
			r.status = ResourceCorrupt
			return
		}
	}

	fullData := append([]byte(nil), data...)
	calculated := FullHash(append(fullData, r.randomHash...))
	if !bytes.Equal(calculated, r.hash) {
		r.status = ResourceCorrupt
		return
	}

	payload := fullData
	if r.hasMetadata && r.segmentIndex == 1 {
		if len(fullData) < 3 {
			r.status = ResourceCorrupt
			return
		}
		metaSize := int(fullData[0])<<16 | int(fullData[1])<<8 | int(fullData[2])
		if len(fullData) < 3+metaSize {
			r.status = ResourceCorrupt
			return
		}
		packedMeta := fullData[3 : 3+metaSize]
		_ = os.WriteFile(r.metaStoragePath, packedMeta, 0o644)
		payload = fullData[3+metaSize:]
	}

	if dir := filepath.Dir(r.storagePath); dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}
	f, err := os.OpenFile(r.storagePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		Log(fmt.Sprintf("Error opening resource file: %v", err), LOG_ERROR)
		r.status = ResourceCorrupt
		return
	}
	if _, err = f.Write(payload); err != nil {
		Log(fmt.Sprintf("Error writing resource file: %v", err), LOG_ERROR)
		r.status = ResourceCorrupt
		_ = f.Close()
		return
	}
	_ = f.Close()

	r.status = ResourceComplete
	r.Prove(fullData)
	if r.link != nil {
		concluded = true
		r.link.ResourceConcluded(r)
	}
	r.handleIncomingCompletion()
	for i := range fullData {
		fullData[i] = 0
	}
}

// Prove sends the proof.
func (r *Resource) Prove(data []byte) {
	if r.status == ResourceFailed {
		return
	}
	proof := FullHash(append(data, r.hash...))
	proofData := append(r.hash, proof...)
	p := NewPacket(
		r.link,
		proofData,
		WithPacketType(PacketTypeProof),
		WithPacketContext(PacketCONTEXT_RESOURCE_PRF),
	)
	p.Send()
	if p.Sent {
		Cache(p, true)
		return
	}
	Log("Could not send proof packet, cancelling resource", LOG_DEBUG)
	r.Cancel()
}

func (r *Resource) handleIncomingCompletion() {
	if r.segmentIndex != r.totalSegments {
		Log(fmt.Sprintf("Resource segment %d of %d received, waiting for next segment to be announced", r.segmentIndex, r.totalSegments), LOG_DEBUG)
		return
	}

	if r.hasMetadata {
		if data, err := os.ReadFile(r.metaStoragePath); err == nil {
			var meta map[string]any
			if err := umsgpack.Unpackb(data, &meta); err == nil {
				r.metadataMap = meta
			} else {
				Log(fmt.Sprintf("Error decoding resource metadata: %v", err), LOG_ERROR)
			}
			_ = os.Remove(r.metaStoragePath)
		}
	}

	file, err := os.Open(r.storagePath)
	if err != nil {
		Log(fmt.Sprintf("Error opening completed resource file: %v", err), LOG_ERROR)
		return
	}
	r.dataFile = file

	if r.callback != nil {
		safeResourceCallback(r.callback, r)
	}

	if r.dataFile != nil {
		_ = r.dataFile.Close()
		r.dataFile = nil
	}
}

// prepareNextSegment prepares the next segment for large resources.
func (r *Resource) prepareNextSegment() {
	if r.preparingNext || r.inputFile == nil || r.segmentIndex >= r.totalSegments {
		return
	}
	Log(fmt.Sprintf("Preparing segment %d of %d for resource %s", r.segmentIndex+1, r.totalSegments, r), LOG_DEBUG)
	r.preparingNext = true
	defer func() { r.preparingNext = false }()
	next, err := NewResource(
		nil,
		r.inputFile,
		r.link,
		nil,
		false,
		r.autoCompressOption,
		r.callback,
		r.progressCallback,
		nil,
		r.segmentIndex+1,
		r.originalHash,
		r.requestID,
		r.isResponse,
		r.metadataSize,
	)
	if err != nil {
		Log(fmt.Sprintf("Error preparing next resource segment: %v", err), LOG_ERROR)
		return
	}
	r.nextSegment = next
}

// ValidateProof validates a proof from the resource receiver.
func (r *Resource) ValidateProof(proofData []byte) {
	if r.status == ResourceFailed {
		return
	}

	hashLen := sha256Bits / 8
	if len(proofData) != hashLen*2 {
		return
	}

	if !bytes.Equal(proofData[hashLen:], r.expectedProof) {
		return
	}

	r.status = ResourceComplete
	if r.link != nil {
		r.link.ResourceConcluded(r)
	}

	if r.segmentIndex == r.totalSegments {
		r.finishSender()
	} else {
		if !r.preparingNext {
			Log(fmt.Sprintf("Next segment preparation for resource %s was not started yet, preparing now. This will slow down transfer.", r), LOG_WARNING)
			r.prepareNextSegment()
		}
		for r.nextSegment == nil {
			time.Sleep(50 * time.Millisecond)
		}
		r.releaseSenderState()
		r.nextSegment.Advertise()
	}
}

func (r *Resource) finishSender() {
	if r.callback != nil {
		safeResourceCallback(r.callback, r)
	}
	if r.inputFile != nil {
		_ = r.inputFile.Close()
		r.inputFile = nil
	}
	if r.tempInputPath != "" {
		_ = os.Remove(r.tempInputPath)
		r.tempInputPath = ""
	}
}

func (r *Resource) releaseSenderState() {
	r.metadata = nil
	r.parts = nil
	r.outgoingParts = nil
	r.inputFile = nil
	r.link = nil
	r.hashmap = nil
}

// ReceivePart handles an incoming resource packet.
func (r *Resource) ReceivePart(packet *Packet) {
	r.receiveLock.Lock()
	r.receivingPart = true

	now := time.Now()
	r.lastActivity = now
	r.retriesLeft = r.maxRetries

	if r.reqResp.IsZero() {
		r.reqResp = now
		rtt := now.Sub(r.reqSent).Seconds()
		if rtt < 0 {
			rtt = 0
		}
		r.partTimeoutFactor = PartTimeoutFactorAfterRTT
		switch {
		case r.rtt == 0:
			r.rtt = r.link.RTT.Seconds()
			r.WatchdogJob()
		case rtt < r.rtt:
			r.rtt = math.Max(r.rtt-r.rtt*0.05, rtt)
		case rtt > r.rtt:
			r.rtt = math.Min(r.rtt+r.rtt*0.05, rtt)
		}

		if rtt > 0 {
			reqRespCost := len(packet.Raw)
			if reqRespCost == 0 {
				reqRespCost = len(packet.Data)
			}
			reqRespCost += r.reqSentBytes
			r.reqRespRTTRate = float64(reqRespCost) / rtt
			if r.reqRespRTTRate > RateFast && r.fastRateRounds < ResourceFastRateThreshold {
				r.fastRateRounds++
				if r.fastRateRounds == ResourceFastRateThreshold {
					r.windowMax = ResourceWindowMaxFast
				}
			}
		}
	}

	if r.status == ResourceFailed {
		return
	}

	r.status = ResourceTransferring

	partData := packet.Data
	if len(partData) == 0 {
		partData = packet.Ciphertext
	}
	partHash := r.getMapHash(partData)

	start := r.consecutiveHeight
	if start < 0 {
		start = 0
	}
	windowEnd := start + r.window
	if windowEnd > len(r.hashmap) {
		windowEnd = len(r.hashmap)
	}

	updated := false
	for idx := start; idx < windowEnd; idx++ {
		mapHash := r.hashmap[idx]
		if mapHash == nil || !bytes.Equal(mapHash, partHash) {
			continue
		}
		if r.parts[idx] == nil {
			r.parts[idx] = append([]byte(nil), partData...)
			r.rttRxBytes += len(partData)
			r.receivedCount++
			if r.outstanding > 0 {
				r.outstanding--
			}

			if idx == r.consecutiveHeight+1 {
				r.consecutiveHeight = idx
			}
			cp := r.consecutiveHeight + 1
			for cp < len(r.parts) && r.parts[cp] != nil {
				r.consecutiveHeight = cp
				cp++
			}

			updated = true
		}
		break
	}

	if updated && r.progressCallback != nil {
		go safeResourceCallback(r.progressCallback, r)
	}

	if r.receivedCount == r.totalParts && !r.assemblyLock {
		r.assemblyLock = true
		r.receivingPart = false
		r.receiveLock.Unlock()
		go r.Assemble()
		return
	}

	shouldRequest := r.outstanding == 0
	r.receivingPart = false
	r.receiveLock.Unlock()

	if shouldRequest {
		if r.window < r.windowMax {
			r.window++
			if (r.window - r.windowMin) > (r.windowFlexibility - 1) {
				r.windowMin++
			}
		}

		if !r.reqSent.IsZero() {
			rtt := time.Since(r.reqSent).Seconds()
			if rtt != 0 {
				reqTransferred := r.rttRxBytes - r.rttRxBytesAtReq
				r.reqDataRTTRate = float64(reqTransferred) / rtt
				r.updateEIFR()
				r.rttRxBytesAtReq = r.rttRxBytes

				if r.reqDataRTTRate > RateFast && r.fastRateRounds < ResourceFastRateThreshold {
					r.fastRateRounds++
					if r.fastRateRounds == ResourceFastRateThreshold {
						r.windowMax = ResourceWindowMaxFast
					}
				}

				if r.fastRateRounds == 0 && r.reqDataRTTRate < RateVerySlow && r.verySlowRateRounds < VerySlowRateThreshold {
					r.verySlowRateRounds++
					if r.verySlowRateRounds == VerySlowRateThreshold {
						r.windowMax = ResourceWindowMaxVerySlow
					}
				} else if r.reqDataRTTRate >= RateVerySlow {
					r.verySlowRateRounds = 0
				}
			}
		}

		r.RequestNext()
	}
}

// RequestNext requests subsequent parts from the initiator.
func (r *Resource) RequestNext() {
	if r.status == ResourceFailed || r.waitingForHMU {
		return
	}

	r.outstanding = 0
	hashmapState := HashmapNotExhausted
	requested := make([]byte, 0, r.window*MapHashLen)

	pn := r.consecutiveHeight + 1
	searchEnd := pn + r.window
	if searchEnd > len(r.parts) {
		searchEnd = len(r.parts)
	}

	for idx := pn; idx < searchEnd; idx++ {
		if r.parts[idx] == nil {
			if r.hashmap[idx] != nil {
				requested = append(requested, r.hashmap[idx]...)
				r.outstanding++
			} else {
				hashmapState = HashmapExhausted
				break
			}
		}
	}

	hmuPart := []byte{hashmapState}
	if hashmapState == HashmapExhausted {
		if r.hashmapHeight == 0 {
			return
		}
		last := r.hashmap[r.hashmapHeight-1]
		if last == nil {
			return
		}
		hmuPart = append(hmuPart, last...)
		r.waitingForHMU = true
	}

	requestData := append(hmuPart, r.hash...)
	requestData = append(requestData, requested...)

	pkt := NewPacket(
		r.link,
		requestData,
		WithPacketContext(PacketCONTEXT_RESOURCE_REQ),
	)
	if pkt == nil {
		return
	}

	pkt.Send()
	if !pkt.Sent {
		Log("Could not send resource request packet, cancelling resource", LOG_DEBUG)
		r.Cancel()
		return
	}

	r.lastActivity = time.Now()
	r.reqSent = r.lastActivity
	if len(pkt.Raw) > 0 {
		r.reqSentBytes = len(pkt.Raw)
	} else {
		r.reqSentBytes = len(requestData)
	}
	r.reqResp = time.Time{}
}

// Request handles an incoming request for parts on the receiver side.
func (r *Resource) Request(requestData []byte) {
	if r.status == ResourceFailed {
		return
	}
	r.lastRequestData = copyBytes(requestData)

	rtt := time.Since(r.advSent).Seconds()
	if r.rtt == 0 {
		r.rtt = rtt
	}

	if r.status != ResourceTransferring {
		r.status = ResourceTransferring
		r.WatchdogJob()
	}

	r.retriesLeft = r.maxRetries

	wantsHMU := len(requestData) > 0 && requestData[0] == HashmapExhausted
	pad := 1
	if wantsHMU {
		pad += MapHashLen
	}

	hashLen := sha256Bits / 8
	if len(requestData) < pad+hashLen {
		return
	}

	requestedHashes := requestData[pad+hashLen:]
	if len(requestedHashes) == 0 {
		return
	}

	for i := 0; i+MapHashLen <= len(requestedHashes); i += MapHashLen {
		partPkt := r.outgoingPartByMapHash[string(requestedHashes[i:i+MapHashLen])]
		if partPkt == nil {
			continue
		}

		if !partPkt.Sent {
			partPkt.Send()
		} else {
			partPkt.Resend()
		}

		if !partPkt.Sent {
			Log("Resource could not send parts, cancelling transfer", LOG_DEBUG)
			r.Cancel()
			return
		}

		r.sentParts++
		r.lastActivity = time.Now()
		r.lastPartSent = r.lastActivity
	}
	if r.sentParts == 0 && len(requestedHashes) > 0 {
		Log(fmt.Sprintf("Resource %s got part request but matched 0 hashes (requested_len=%d)", r, len(requestedHashes)), LOG_ERROR)
	}

	if wantsHMU {
		if len(requestData) < 1+MapHashLen {
			return
		}
		lastMap := requestData[1 : 1+MapHashLen]
		collisionGuardLen := r.collisionGuardLen
		if collisionGuardLen <= 0 {
			collisionGuardLen = CollisionGuardSize
		}
		segLen := r.hashmapMaxLen
		if segLen <= 0 {
			segLen = HashmapMaxLen
		}
		partIndex := r.receiverMinHeight
		searchStart := partIndex
		searchEnd := searchStart + collisionGuardLen
		if searchEnd > len(r.outgoingParts) {
			searchEnd = len(r.outgoingParts)
		}

		foundIndex := -1
		for idx := searchStart; idx < searchEnd; idx++ {
			partIndex++
			part := r.outgoingParts[idx]
			if part != nil && bytes.Equal(part.mapHash, lastMap) {
				foundIndex = partIndex
				break
			}
		}

		if foundIndex == -1 {
			Log("Resource sequencing error, cancelling transfer!", LOG_ERROR)
			r.Cancel()
			return
		}

		r.receiverMinHeight = max(foundIndex-1-ResourceWindowMax, 0)
		if foundIndex%segLen != 0 {
			Log("Resource sequencing error, cancelling transfer!", LOG_ERROR)
			r.Cancel()
			return
		}

		segment := foundIndex / segLen
		hashmapStart := segment * segLen
		hashmapEnd := hashmapStart + segLen
		if hashmapEnd > len(r.hashmap) {
			hashmapEnd = len(r.hashmap)
		}

		hm := make([]byte, 0, (hashmapEnd-hashmapStart)*MapHashLen)
		for i := hashmapStart; i < hashmapEnd; i++ {
			if i >= len(r.hashmap) || r.hashmap[i] == nil {
				break
			}
			hm = append(hm, r.hashmap[i]...)
		}

		hmuPayload, err := msgpackMarshal([]any{segment, hm})
		if err != nil {
			Log(fmt.Sprintf("Could not encode HMU payload: %v", err), LOG_ERROR)
			r.Cancel()
			return
		}

		hmu := append(r.hash, hmuPayload...)
		hmuPacket := NewPacket(
			r.link,
			hmu,
			WithPacketContext(PacketCONTEXT_RESOURCE_HMU),
		)
		if hmuPacket == nil {
			return
		}
		hmuPacket.Send()
		if !hmuPacket.Sent {
			Log("Could not send resource HMU packet, cancelling resource", LOG_DEBUG)
			r.Cancel()
			return
		}
		r.lastActivity = time.Now()
	}

	if r.sentParts == len(r.outgoingParts) {
		r.status = ResourceAwaitingProof
		r.retriesLeft = ProofTimeoutFactor
	}

	if r.progressCallback != nil {
		go safeResourceCallback(r.progressCallback, r)
	}
}

// Cancel cancels the resource transfer.
func (r *Resource) Cancel() {
	if r.status >= ResourceComplete {
		return
	}
	r.status = ResourceFailed

	if r.initiator {
		if r.link != nil && r.link.Status == LinkActive {
			cancel := NewPacket(
				r.link,
				r.hash,
				WithPacketContext(PacketCONTEXT_RESOURCE_ICL),
			)
			if cancel != nil {
				cancel.Send()
			}
		}
		if r.link != nil {
			r.link.CancelOutgoingResource(r)
		}
	} else if r.link != nil {
		r.link.CancelIncomingResource(r)
	}

	if r.link != nil {
		r.link.ResourceConcluded(r)
	}

	if r.callback != nil {
		go safeResourceCallback(r.callback, r)
	}
	if r.inputFile != nil {
		_ = r.inputFile.Close()
		r.inputFile = nil
	}
	if r.tempInputPath != "" {
		_ = os.Remove(r.tempInputPath)
		r.tempInputPath = ""
	}
}

func (r *Resource) rejected() {
	if r.status >= ResourceComplete {
		return
	}
	if r.initiator {
		r.status = ResourceRejected
		if r.link != nil {
			r.link.CancelOutgoingResource(r)
		}
		if r.callback != nil {
			go safeResourceCallback(r.callback, r)
		}
	}
}

func safeResourceCallback(cb func(*Resource), res *Resource) {
	defer func() {
		if rec := recover(); rec != nil {
			Log(fmt.Sprintf("panic in resource callback: %v", rec), LOG_ERROR)
		}
	}()
	cb(res)
}

// SetCallback sets the completion callback.
func (r *Resource) SetCallback(cb func(*Resource)) {
	r.callback = cb
}

// SetProgressCallback sets the progress callback.
func (r *Resource) SetProgressCallback(cb func(*Resource)) {
	r.progressCallback = cb
}

// GetProgress returns the overall resource progress.
func (r *Resource) GetProgress() float64 {
	if r.status == ResourceComplete && r.segmentIndex == r.totalSegments {
		return 1.0
	}

	var processed, total float64
	maxPartsPerSegment := math.Ceil(float64(MaxEfficientSize) / float64(r.sdu))

	if r.initiator {
		if !r.split {
			processed = float64(r.sentParts)
			total = float64(r.totalParts)
		} else {
			prevSegments := float64(r.segmentIndex - 1)
			prevParts := prevSegments * maxPartsPerSegment
			currentParts := float64(r.totalParts)
			currentFactor := 1.0
			if currentParts < maxPartsPerSegment && currentParts > 0 {
				currentFactor = maxPartsPerSegment / currentParts
			}
			processed = prevParts + float64(r.sentParts)*currentFactor
			total = float64(r.totalSegments) * maxPartsPerSegment
		}
	} else {
		if !r.split {
			processed = float64(r.receivedCount)
			total = float64(r.totalParts)
		} else {
			prevSegments := float64(r.segmentIndex - 1)
			prevParts := prevSegments * maxPartsPerSegment
			currentParts := float64(r.totalParts)
			currentFactor := 1.0
			if currentParts < maxPartsPerSegment && currentParts > 0 {
				currentFactor = maxPartsPerSegment / currentParts
			}
			processed = prevParts + float64(r.receivedCount)*currentFactor
			total = float64(r.totalSegments) * maxPartsPerSegment
		}
	}

	if total == 0 {
		return 0
	}
	return math.Min(1.0, processed/total)
}

// GetSegmentProgress returns progress for the current segment.
func (r *Resource) GetSegmentProgress() float64 {
	if r.status == ResourceComplete && r.segmentIndex == r.totalSegments {
		return 1.0
	}
	var processed float64
	if r.initiator {
		processed = float64(r.sentParts)
	} else {
		processed = float64(r.receivedCount)
	}
	if r.totalParts == 0 {
		return 0
	}
	return math.Min(1.0, processed/float64(r.totalParts))
}

func (r *Resource) GetTransferSize() int { return r.size }
func (r *Resource) GetDataSize() int     { return r.totalSize }
func (r *Resource) GetParts() int        { return r.totalParts }
func (r *Resource) GetSegments() int     { return r.totalSegments }
func (r *Resource) GetHash() []byte      { return copyBytes(r.hash) }
func (r *Resource) IsCompressed() bool   { return r.comp }

func (r *Resource) Progress() float64        { return r.GetProgress() }
func (r *Resource) SegmentProgress() float64 { return r.GetSegmentProgress() }
func (r *Resource) TotalSize() int           { return r.totalSize }
func (r *Resource) Status() byte             { return r.status }
func (r *Resource) Hash() []byte             { return r.GetHash() }
func (r *Resource) Link() *Link              { return r.link }

func (r *Resource) Metadata() map[string]any {
	if r.metadataMap == nil {
		return nil
	}
	meta := make(map[string]any, len(r.metadataMap))
	for k, v := range r.metadataMap {
		meta[k] = v
	}
	return meta
}

func (r *Resource) DataFile() string {
	return r.storagePath
}

func (r *Resource) String() string {
	linkID := "unknown"
	if r.link != nil && len(r.link.LinkID) > 0 {
		linkID = PrettyHex(r.link.LinkID)
	}
	return fmt.Sprintf("<%s/%s>", PrettyHex(r.hash), linkID)
}

func NewResourceAdvertisementFromResource(r *Resource) ResourceAdvertisement {
	// flags
	var f byte
	if r.encr {
		f |= 0x01
	}
	if r.comp {
		f |= 0x02
	}
	if r.split {
		f |= 0x04
	}
	if r.requestID != nil && !r.isResponse {
		f |= 0x08
	}
	if r.requestID != nil && r.isResponse {
		f |= 0x10
	}
	if r.hasMetadata {
		f |= 0x20
	}

	return ResourceAdvertisement{
		T:    r.size,
		D:    r.totalSize,
		N:    r.totalParts,
		H:    r.hash,
		R:    r.randomHash,
		O:    r.originalHash,
		M:    flattenHashmap(r.hashmap),
		F:    f,
		I:    r.segmentIndex,
		L:    r.totalSegments,
		Q:    r.requestID,
		Link: r.link,
		E:    r.encr,
		C:    r.comp,
		S:    r.split,
		U:    r.requestID != nil && !r.isResponse,
		P:    r.requestID != nil && r.isResponse,
		X:    r.hasMetadata,
	}
}

func (a ResourceAdvertisement) Pack(segment int) []byte {
	segLen := HashmapMaxLen
	if a.Link != nil {
		if v := resourceHashmapCapacityForLink(a.Link); v > 0 {
			segLen = v
		}
	}

	hashmapStart := segment * segLen
	hashmapEnd := hashmapStart + segLen
	if hashmapEnd > a.N {
		hashmapEnd = a.N
	}
	var hm []byte
	for i := hashmapStart; i < hashmapEnd; i++ {
		start := i * MapHashLen
		end := start + MapHashLen
		if end > len(a.M) {
			break
		}
		hm = append(hm, a.M[start:end]...)
	}

	dict := map[string]any{
		"t": a.T,
		"d": a.D,
		"n": a.N,
		"h": a.H,
		"r": a.R,
		"o": a.O,
		"i": a.I,
		"l": a.L,
		"q": a.Q,
		"f": a.F,
		"m": hm,
	}

	b, err := msgpackMarshal(dict)
	if err != nil {
		Log(fmt.Sprintf("Could not pack resource advertisement: %v", err), LOG_ERROR)
		return []byte{}
	}
	return b
}

func ResourceAdvertisementUnpack(data []byte) (*ResourceAdvertisement, error) {
	// Python parity: the advertisement is a msgpack dict with short keys.
	// The bundled umsgpack implementation does not reliably decode into tagged structs,
	// so unpack into a map and extract the fields.
	var dictAny map[any]any
	if err := func() (err error) {
		defer func() {
			if rec := recover(); rec != nil {
				err = fmt.Errorf("msgpack unpack panic: %v", rec)
			}
		}()
		return msgpackUnmarshalInto(data, &dictAny)
	}(); err != nil {
		return nil, err
	}

	get := func(key string) (any, bool) {
		if v, ok := dictAny[key]; ok {
			return v, true
		}
		// Some decoders use []byte keys for msgpack "str" types.
		for k, v := range dictAny {
			ks, ok := asStringValue(k)
			if ok && ks == key {
				return v, true
			}
		}
		return nil, false
	}

	readInt := func(key string) int {
		if v, ok := get(key); ok {
			if i, ok := asIntValue(v); ok {
				return i
			}
		}
		return 0
	}
	readBytes := func(key string) []byte {
		if v, ok := get(key); ok {
			if b, ok := asBytesValue(v); ok {
				return append([]byte(nil), b...)
			}
		}
		return nil
	}
	readByte := func(key string) byte {
		if v, ok := get(key); ok {
			if i, ok := asIntValue(v); ok {
				return byte(i)
			}
		}
		return 0
	}

	adv := &ResourceAdvertisement{
		T: readInt("t"),
		D: readInt("d"),
		N: readInt("n"),
		H: readBytes("h"),
		R: readBytes("r"),
		O: readBytes("o"),
		I: readInt("i"),
		L: readInt("l"),
		Q: readBytes("q"),
		F: readByte("f"),
		M: readBytes("m"),
	}

	adv.E = (adv.F & 0x01) == 0x01
	adv.C = ((adv.F >> 1) & 0x01) == 0x01
	adv.S = ((adv.F >> 2) & 0x01) == 0x01
	adv.U = ((adv.F >> 3) & 0x01) == 0x01
	adv.P = ((adv.F >> 4) & 0x01) == 0x01
	adv.X = ((adv.F >> 5) & 0x01) == 0x01
	return adv, nil
}

func flattenHashmap(hm [][]byte) []byte {
	var b []byte
	for _, h := range hm {
		b = append(b, h...)
	}
	return b
}

func (a ResourceAdvertisement) GetTransferSize() int { return a.TransferSize() }
func (a ResourceAdvertisement) GetDataSize() int     { return a.DataSize() }
func (a ResourceAdvertisement) GetParts() int        { return a.N }
func (a ResourceAdvertisement) GetSegments() int     { return a.L }
func (a ResourceAdvertisement) GetHash() []byte      { return a.H }
func (a ResourceAdvertisement) IsCompressed() bool   { return a.C }
func (a ResourceAdvertisement) HasMetadata() bool    { return a.X }

func (a *ResourceAdvertisement) IsRequest() bool {
	if a == nil {
		return false
	}
	return len(a.Q) > 0 && a.U
}

func (a *ResourceAdvertisement) IsResponse() bool {
	if a == nil {
		return false
	}
	return len(a.Q) > 0 && a.P
}

func (a *ResourceAdvertisement) RequestID() []byte {
	if a == nil || len(a.Q) == 0 {
		return nil
	}
	return append([]byte(nil), a.Q...)
}

func (a *ResourceAdvertisement) TransferSize() int {
	if a == nil {
		return 0
	}
	return a.T
}

func (a *ResourceAdvertisement) DataSize() int {
	if a == nil {
		return 0
	}
	return a.D
}

// etc.

// ---------------- msgpack helpers ----------------

func msgpackMarshal(v any) ([]byte, error) {
	return umsgpack.Packb(v)
}

func msgpackUnmarshal(data []byte) ([]any, error) {
	var out []any
	if err := umsgpack.Unpackb(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func msgpackUnmarshalInto(data []byte, v interface{}) error {
	return umsgpack.Unpackb(data, v)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
