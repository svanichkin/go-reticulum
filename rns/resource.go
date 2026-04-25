package rns

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dsnet/compress/bzip2"
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
	compressedSize   int
	hash             []byte
	truncatedHash    []byte
	originalHash     []byte
	randomHash       []byte
	hashmapRaw       []byte
	hashmap          [][]byte
	hashmapHeight    int
	expectedProof    []byte
	encr, comp       bool
	encrypted        bool
	compressed       bool
	initiator        bool
	hasMetadata      bool
	split            bool
	segmentIndex     int
	totalSegments    int
	requestID        []byte
	isResponse       bool
	reqHashlist      [][]byte
	data             []byte
	uncompressedData []byte
	compressedData   []byte

	Link *Link
	// Python parity: HASHMAP_MAX_LEN/COLLISION_GUARD_SIZE are global and based
	// on Link.MDU, not per-link negotiated MDU.
	hashmapMaxLen     int
	collisionGuardLen int

	// SDU and parts
	sdu                   int
	totalParts            int
	parts                 []any
	outgoingPartByMapHash map[string]*Packet
	sentParts             int
	receivedCount         int
	outstanding           int

	// window
	window                       int
	windowMax                    int
	windowMin                    int
	windowFlexibility            int
	waitingForHMU                bool
	receivingPart                bool
	consecutiveHeight            int
	receiverMinConsecutiveHeight int

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
	DataFile        string
	metaStoragePath string
	inputFile       *os.File
	tempInputPath   string

	metadata     []byte
	metadataSize int
	Metadata     any
	dataFile     *os.File

	Status byte

	callback         func(*Resource)
	progressCallback func(*Resource)

	advPacket          *Packet
	advSent            time.Time
	processedParts     float64
	progressTotalParts float64

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

const (
	AdvOverhead = 134
)

var (
	// HASHMAP_MAX_LEN depends on MTU. In Python: floor((Link.MDU-OVERHEAD)/MAPHASH_LEN)
	// In Go this can be computed dynamically from link.MDU during pack(), but we keep it
	// global for now (works fine when MDU is fixed).
	HashmapMaxLen = int(math.Floor(float64(LinkMDU-AdvOverhead) / float64(MapHashLen)))

	CollisionGuardSize = 2*ResourceWindowMax + HashmapMaxLen
)

// ---------------- static operations ----------------

// reject mirrors Resource.reject().
func (r *Resource) reject(advPkt *Packet) {
	defer func() {
		if r := recover(); r != nil {
			Log(fmt.Sprintf("An error ocurred while rejecting advertised resource: %v", r), LOG_ERROR)
			TraceException(r)
		}
	}()
	adv, err := (ResourceAdvertisement{}).Unpack(advPkt.Plaintext)
	if err != nil {
		Log(fmt.Sprintf("An error ocurred while rejecting advertised resource: %v", err), LOG_ERROR)
		TraceException(err)
		return
	}
	resHash := adv.H
	reject := NewPacket(
		advPkt.Link,
		resHash,
		PacketTypeData,
		PacketCONTEXT_RESOURCE_RCL,
		Broadcast,
		HeaderType1,
		nil,
		nil,
		true,
		FlagUnset,
	)
	_ = reject.Send()
}

// accept mirrors Resource.accept().
func (r *Resource) accept(advPkt *Packet, cb func(*Resource), progCb func(*Resource), reqID []byte) (res *Resource) {
	defer func() {
		if rec := recover(); rec != nil {
			Log("Could not decode resource advertisement, dropping resource", LOG_DEBUG)
			res = nil
		}
	}()
	adv, err := (ResourceAdvertisement{}).Unpack(advPkt.Plaintext)
	if err != nil {
		Log("Could not decode resource advertisement, dropping resource", LOG_DEBUG)
		return nil
	}

	res, err = NewResource(nil, nil, advPkt.Link, nil, false, true, nil, progCb, nil, 1, nil, reqID, false, 0)
	if err != nil {
		Log("Could not decode resource advertisement, dropping resource", LOG_DEBUG)
		return nil
	}
	res.flags = adv.F
	res.size = adv.T
	res.totalSize = adv.D
	res.uncompressedSize = adv.D
	res.hash = adv.H
	res.originalHash = adv.O
	res.randomHash = adv.R
	res.hashmapRaw = adv.M
	res.encr = (adv.F & 0x01) != 0
	res.comp = ((adv.F >> 1) & 0x01) != 0
	res.encrypted = (adv.F & 0x01) != 0
	res.compressed = ((adv.F >> 1) & 0x01) != 0
	res.initiator = false
	res.callback = cb
	res.progressCallback = progCb
	res.Link = advPkt.Link
	res.segmentIndex = adv.I
	res.totalSegments = adv.L
	res.requestID = reqID
	res.Status = ResourceTransferring
	res.window = ResourceWindow
	res.windowMax = ResourceWindowMaxSlow
	res.windowMin = ResourceWindowMin
	res.windowFlexibility = ResourceWindowFlexibility
	res.consecutiveHeight = -1
	res.totalParts = int(math.Ceil(float64(res.size) / float64(res.sdu)))
	res.receivedCount = 0
	res.outstanding = 0
	res.parts = make([]any, res.totalParts)
	res.lastActivity = time.Now()
	res.startedTransferring = res.lastActivity
	basePath := ""
	if instance != nil && instance.ResourcePath != "" {
		basePath = instance.ResourcePath
	} else if Owner != nil && Owner.ResourcePath != "" {
		basePath = Owner.ResourcePath
	}
	res.DataFile = basePath + "/" + hex.EncodeToString(res.originalHash)
	res.metaStoragePath = res.DataFile + ".meta"

	// split / metadata
	res.split = adv.L > 1
	res.hasMetadata = adv.X

	res.hashmap = make([][]byte, res.totalParts)
	res.hashmapHeight = 0
	res.waitingForHMU = false
	res.receivingPart = false

	res.hashmapMaxLen = HashmapMaxLen
	res.collisionGuardLen = CollisionGuardSize

	prevWin := advPkt.Link.GetLastResourceWindow()
	prevEIFR := advPkt.Link.GetLastResourceEIFR()
	if prevWin > 0 {
		res.window = prevWin
	}
	if prevEIFR > 0 {
		res.previousEIFR = prevEIFR
	}

	if advPkt.Link.HasIncomingResource(res) {
		Log("Ignoring resource advertisement for "+PrettyHex(res.hash)+", resource already transferring", LOG_DEBUG)
		return nil
	}

	advPkt.Link.RegisterIncomingResource(res)
	Log(fmt.Sprintf(
		"Accepting resource advertisement for %s. Transfer size is %s in %d parts.",
		PrettyHex(res.hash), PrettySize(float64(res.size)), res.totalParts,
	), LOG_DEBUG)

	if advPkt.Link.callbacks.ResourceStarted != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					Log(fmt.Sprintf("Error while executing resource started callback from %s. The contained exception was: %v", res, r), LOG_ERROR)
				}
			}()
			advPkt.Link.callbacks.ResourceStarted(res)
		}()
	}

	res.HashmapUpdate(0, res.hashmapRaw)
	res.WatchdogJob()
	return res
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
		Link:                         link,
		assemblyLock:                 false,
		preparingNext:                false,
		nextSegment:                  nil,
		hasMetadata:                  sentMetadataSize > 0,
		metadataSize:                 sentMetadataSize,
		maxRetries:                   MaxRetries,
		maxAdvRetries:                MaxAdvRetries,
		retriesLeft:                  MaxRetries,
		timeoutFactor:                link.TrafficTimeoutFactor,
		partTimeoutFactor:            PartTimeoutFactor,
		senderGraceTime:              SenderGraceTime,
		hmuRetryOK:                   false,
		progressCallback:             progCb,
		requestID:                    requestID,
		isResponse:                   isResponse,
		reqHashlist:                  make([][]byte, 0),
		autoCompressLimit:            AutoCompressMaxSize,
		autoCompressOption:           autoCompress,
		receiverMinConsecutiveHeight: 0,
		consecutiveHeight:            -1,
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
		packed, err := umsgpack.Packb(metadata)
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
	} else {
		res.metadata = []byte{}
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
			n, errRead := io.ReadFull(file, buf)
			if errRead != nil && errRead != io.EOF && errRead != io.ErrUnexpectedEOF {
				return nil, errRead
			}
			if n < len(buf) {
				buf = buf[:n]
			}
			if errRead != nil && errRead != io.EOF && errRead != io.ErrUnexpectedEOF {
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
			n, errRead := io.ReadFull(file, buf)
			if errRead != nil && errRead != io.EOF && errRead != io.ErrUnexpectedEOF {
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

	res.Status = ResourceNone
	if res.Link != nil && res.Link.MTU > 0 {
		res.sdu = res.Link.MTU - HEADER_MAXSIZE - IFAC_MIN_SIZE
	} else if res.Link != nil && res.Link.MDU > 0 {
		res.sdu = res.Link.MDU
	} else if PacketMDU > 0 {
		res.sdu = PacketMDU
	}
	res.timeout = float64(link.RTT) * link.TrafficTimeoutFactor
	if timeout != nil {
		res.timeout = *timeout
	}

	res.hashmapMaxLen = HashmapMaxLen
	res.collisionGuardLen = CollisionGuardSize

	if fullData != nil {
		res.initiator = true
		res.callback = cb

		// auto-compression
		res.uncompressedData = append([]byte(nil), fullData...)
		uncompressed := res.uncompressedData
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
			var compressedBuf bytes.Buffer
			writer, err := bzip2.NewWriter(&compressedBuf, nil)
			if err != nil {
				return nil, err
			}
			if _, err := writer.Write(uncompressed); err != nil {
				_ = writer.Close()
				return nil, err
			}
			if err := writer.Close(); err != nil {
				return nil, err
			}
			compressed := compressedBuf.Bytes()
			res.compressedSize = len(compressed)
			res.compressedData = append([]byte(nil), compressed...)
			res.comp = len(compressed) < len(uncompressed)
			res.compressed = res.comp
			if res.comp {
				toSend = res.compressedData
				Log(fmt.Sprintf("Compression saved %d bytes, sending compressed",
					len(uncompressed)-len(compressed)), LOG_EXTREME)
			} else {
				res.compressedData = append([]byte(nil), uncompressed...)
				res.compressedSize = len(uncompressed)
				toSend = res.compressedData
				Log("Compression did not decrease size, sending uncompressed", LOG_EXTREME)
			}
		} else {
			res.comp = false
			res.compressed = false
			res.compressedData = append([]byte(nil), uncompressed...)
			res.compressedSize = len(uncompressed)
			toSend = uncompressed
		}

		// random-hash prefix
		// Python uses two random values:
		// - a 4-byte prefix embedded in the encrypted stream (stripped by receiver)
		// - a 4-byte random hash used for map hashes and resource hash validation
		streamPrefix := IdentityGetRandomHash()[:RandomHashSize]

		payload := append(streamPrefix, toSend...)

		// link encryption
		enc, err := link.Encrypt(payload)
		if err != nil {
			return nil, err
		}
		res.encr = true
		res.encrypted = true
		res.size = len(enc)
		res.data = append([]byte(nil), enc...)

		// parts and hashmap
		hashmapEntries := int(math.Ceil(float64(res.size) / float64(res.sdu)))
		res.totalParts = hashmapEntries
		hashmapOk := false
		for !hashmapOk {
			hashmapComputationBegan := time.Now()
			Log(fmt.Sprintf("Starting resource hashmap computation with %d entries...", hashmapEntries), LOG_EXTREME)
			res.randomHash = IdentityGetRandomHash()[:RandomHashSize]
			hashInput := append(append([]byte(nil), fullData...), res.randomHash...)
			res.hash = FullHash(hashInput)
			res.truncatedHash = TruncatedHash(hashInput)
			proofInput := append(append([]byte(nil), fullData...), res.hash...)
			res.expectedProof = FullHash(proofInput)
			if originalHash != nil {
				res.originalHash = originalHash
			} else {
				res.originalHash = res.hash
			}

			res.parts = make([]any, 0, hashmapEntries)
			res.hashmap = make([][]byte, 0, hashmapEntries)
			res.hashmapHeight = 0
			res.outgoingPartByMapHash = make(map[string]*Packet, hashmapEntries)
			collisionGuard := make([][]byte, 0)
			hashmapOk = true

			for i := 0; i < hashmapEntries; i++ {
				start := i * res.sdu
				end := start + res.sdu
				if end > len(res.data) {
					end = len(res.data)
				}
				chunk := res.data[start:end]
				mapHash := res.getMapHash(chunk)

				for _, h := range collisionGuard {
					if bytes.Equal(h, mapHash) {
						Log("Found hash collision in resource map, remapping...", LOG_DEBUG)
						hashmapOk = false
						break
					}
				}
				if !hashmapOk {
					break
				}

				collisionGuard = append(collisionGuard, mapHash)
				if len(collisionGuard) > res.collisionGuardLen {
					collisionGuard = collisionGuard[1:]
				}

				partPkt := NewPacket(
					res.Link,
					chunk,
					PacketTypeData,
					PacketCONTEXT_RESOURCE,
					Broadcast,
					HeaderType1,
					nil,
					nil,
					true,
					FlagUnset,
				)
				if partPkt == nil {
					return nil, errors.New("failed to create resource part packet")
				}
				if err := partPkt.Pack(); err != nil {
					return nil, err
				}
				partPkt.MapHash = mapHash

				res.hashmap = append(res.hashmap, partPkt.MapHash)
				res.parts = append(res.parts, partPkt)
				res.outgoingPartByMapHash[string(mapHash)] = partPkt
			}
			Log(fmt.Sprintf("Hashmap computation concluded in %.3f seconds",
				time.Since(hashmapComputationBegan).Seconds()), LOG_EXTREME)
		}
		res.data = nil
		res.uncompressedData = nil
		res.compressedData = nil

		if advertise {
			res.Advertise()
		}
	}

	return res, nil
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
	if r.Status == ResourceFailed {
		return
	}
	r.lastActivity = time.Now()
	r.retriesLeft = r.maxRetries

	off := sha256Bits / 8
	payload := plaintext
	if off > len(payload) {
		payload = payload[:0]
	} else {
		payload = payload[off:]
	}
	var update []any
	if err := umsgpack.Unpackb(payload, &update); err != nil {
		panic(err)
	}
	var seg int
	switch v := update[0].(type) {
	case int:
		seg = v
	case int8:
		seg = int(v)
	case int16:
		seg = int(v)
	case int32:
		seg = int(v)
	case int64:
		seg = int(v)
	case uint:
		seg = int(v)
	case uint8:
		seg = int(v)
	case uint16:
		seg = int(v)
	case uint32:
		seg = int(v)
	case uint64:
		seg = int(v)
	case float32:
		seg = int(v)
	case float64:
		seg = int(v)
	default:
		panic(fmt.Sprintf("unexpected hashmap update segment type %T", update[0]))
	}
	switch hm := update[1].(type) {
	case []byte:
		r.HashmapUpdate(seg, hm)
	case string:
		r.HashmapUpdate(seg, []byte(hm))
	default:
		panic(fmt.Sprintf("unexpected hashmap update payload type %T", update[1]))
	}
}

func (r *Resource) HashmapUpdate(segment int, hashmap []byte) {
	if r.Status == ResourceFailed {
		return
	}
	r.Status = ResourceTransferring
	segLen := r.hashmapMaxLen
	hashes := len(hashmap) / MapHashLen
	for i := 0; i < hashes; i++ {
		idx := i + segment*segLen
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

func (r *Resource) advertiseJob() {
	adv := ResourceAdvertisement{
		T: r.size,
		D: r.totalSize,
		N: r.totalParts,
		H: r.hash,
		R: r.randomHash,
		O: r.originalHash,
		M: func() []byte {
			var b []byte
			for _, h := range r.hashmap {
				b = append(b, h...)
			}
			return b
		}(),
		F: func() byte {
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
			return f
		}(),
		I:    r.segmentIndex,
		L:    r.totalSegments,
		Q:    r.requestID,
		Link: r.Link,
		E:    r.encr,
		C:    r.comp,
		S:    r.split,
		U:    r.requestID != nil && !r.isResponse,
		P:    r.requestID != nil && r.isResponse,
		X:    r.hasMetadata,
	}
	pkt := NewPacket(
		r.Link,
		adv.Pack(0),
		PacketTypeData,
		PacketCONTEXT_RESOURCE_ADV,
		Broadcast,
		HeaderType1,
		nil,
		nil,
		true,
		FlagUnset,
	)
	if pkt == nil {
		Log("Could not build resource advertisement packet", LOG_ERROR)
		r.Cancel()
		return
	}
	r.advPacket = pkt

	for !r.Link.ReadyForNewResource() {
		r.Status = ResourceQueued
		time.Sleep(250 * time.Millisecond)
	}

	func() {
		defer func() {
			if rec := recover(); rec != nil {
				Log("Could not advertise resource, the contained exception was: "+fmt.Sprint(rec), LOG_ERROR)
				r.Cancel()
			}
		}()
		pkt.Send()
	}()
	if r.Status == ResourceFailed {
		return
	}
	now := time.Now()
	r.lastActivity = now
	r.startedTransferring = now
	r.advSent = now
	r.rtt = 0
	r.Status = ResourceAdvertised
	r.retriesLeft = r.maxAdvRetries
	r.Link.RegisterOutgoingResource(r)
	Log("Sent resource advertisement for "+PrettyHex(r.hash), LOG_EXTREME)
	r.WatchdogJob()
}

// ---------------- EIFR ----------------

func (r *Resource) updateEIFR() {
	var rtt float64
	if r.rtt == 0 {
		rtt = r.Link.RTT.Seconds()
	} else {
		rtt = r.rtt
	}
	var expected float64
	if r.reqDataRTTRate != 0 {
		expected = r.reqDataRTTRate * 8
	} else if r.previousEIFR != 0 {
		expected = r.previousEIFR
	} else {
		if rtt <= 0 {
			panic("division by zero in updateEIFR")
		}
		expected = float64(r.Link.EstablishmentCost*8) / rtt
	}
	r.eifr = expected
	r.Link.ExpectedRate = expected
}

// ---------------- watchdog ----------------

func (r *Resource) WatchdogJob() {
	go func() {
		r.watchdogMu.Lock()
		r.watchdogJobID++
		jobID := r.watchdogJobID
		r.watchdogMu.Unlock()

		for r.Status < ResourceAssembling && jobID == r.watchdogJobID {
			for r.watchdogLock {
				time.Sleep(25 * time.Millisecond)
			}

			var sleepTime time.Duration
			sleepSet := false
			now := time.Now()

			switch r.Status {
			case ResourceAdvertised:
				sleepSet = true
				exp := r.advSent.Add(time.Duration(r.timeout*float64(time.Second)) +
					time.Duration(ProcessingGrace*float64(time.Second)))
				if now.After(exp) {
					if r.retriesLeft <= 0 {
						Log("Resource transfer timeout after sending advertisement", LOG_DEBUG)
						r.Cancel()
						sleepTime = time.Millisecond
					} else {
						Log("No part requests received, retrying resource advertisement...", LOG_DEBUG)
						r.retriesLeft--
						adv := ResourceAdvertisement{
							T: r.size,
							D: r.totalSize,
							N: r.totalParts,
							H: r.hash,
							R: r.randomHash,
							O: r.originalHash,
							M: func() []byte {
								var b []byte
								for _, h := range r.hashmap {
									b = append(b, h...)
								}
								return b
							}(),
							F: func() byte {
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
								return f
							}(),
							I:    r.segmentIndex,
							L:    r.totalSegments,
							Q:    r.requestID,
							Link: r.Link,
							E:    r.encr,
							C:    r.comp,
							S:    r.split,
							U:    r.requestID != nil && !r.isResponse,
							P:    r.requestID != nil && r.isResponse,
							X:    r.hasMetadata,
						}
						pkt := NewPacket(
							r.Link,
							adv.Pack(0),
							PacketTypeData,
							PacketCONTEXT_RESOURCE_ADV,
							Broadcast,
							HeaderType1,
							nil,
							nil,
							true,
							FlagUnset,
						)
						if pkt == nil {
							Log("Could not build resource advertisement packet", LOG_ERROR)
							r.Cancel()
							sleepTime = time.Millisecond
							break
						}
						r.advPacket = pkt
						func() {
							defer func() {
								if rec := recover(); rec != nil {
									Log("Could not resend advertisement packet, cancelling resource. The contained exception was: "+fmt.Sprint(rec), LOG_VERBOSE)
									r.Cancel()
								}
							}()
							pkt.Send()
						}()
						if r.Status == ResourceFailed {
							sleepTime = time.Millisecond
							break
						}
						r.lastActivity = time.Now()
						r.advSent = r.lastActivity
						sleepTime = time.Millisecond
					}
				} else {
					sleepTime = exp.Sub(now)
				}

			case ResourceTransferring:
				sleepSet = true
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
						time.Duration(r.senderGraceTime*float64(time.Second)) + maxExtra
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
				sleepSet = true
				r.timeoutFactor = ProofTimeoutFactor
				exp := r.lastPartSent.Add(
					time.Duration(r.rtt*r.timeoutFactor*float64(time.Second) + r.senderGraceTime*float64(time.Second)),
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
							r.Link,
							expectedData,
							PacketTypeProof,
							PacketCONTEXT_RESOURCE_PRF,
							Broadcast,
							HeaderType1,
							nil,
							nil,
							true,
							FlagUnset,
						)
						p.Pack()
						CacheRequest(p.PacketHash, r.Link)
						r.lastPartSent = time.Now()
						sleepTime = time.Millisecond
					}
				} else {
					sleepTime = exp.Sub(now)
				}

			case ResourceRejected:
				sleepSet = true
				sleepTime = time.Millisecond
			}

			if !sleepSet || sleepTime < 0 {
				Log("Timing error, cancelling resource transfer.", LOG_ERROR)
				r.Cancel()
				sleepTime = time.Millisecond
			}
			if sleepTime == 0 {
				Log("Warning! Link watchdog sleep time of 0!", LOG_DEBUG)
			}
			if sleepTime > WatchdogMaxSleep {
				sleepTime = WatchdogMaxSleep
			}
			time.Sleep(sleepTime)
		}
	}()
}

// ---------------- assemble ----------------

func (r *Resource) Assemble() {
	if r.Status == ResourceFailed {
		return
	}

	defer func() {
		if r.Link != nil {
			r.Link.ResourceConcluded(r)
		}

		if r.segmentIndex != r.totalSegments {
			Log(fmt.Sprintf("Resource segment %d of %d received, waiting for next segment to be announced", r.segmentIndex, r.totalSegments), LOG_DEBUG)
			return
		}

		if r.callback != nil {
			if r.hasMetadata {
				if _, err := os.Stat(r.metaStoragePath); err != nil {
					if !os.IsNotExist(err) {
						panic(err)
					}
					r.Metadata = nil
				} else if data, err := os.ReadFile(r.metaStoragePath); err == nil {
					var meta any
					if err := umsgpack.Unpackb(data, &meta); err == nil {
						r.Metadata = meta
					} else {
						panic(err)
					}
					_ = os.Remove(r.metaStoragePath)
				} else {
					panic(err)
				}
			}

			file, err := os.Open(r.DataFile)
			if err != nil {
				panic(err)
			}
			r.dataFile = file

			func() {
				defer func() {
					if rec := recover(); rec != nil {
						Log("Error while executing resource assembled callback from "+fmt.Sprint(r)+". The contained exception was: "+fmt.Sprint(rec), LOG_ERROR)
					}
				}()
				r.callback(r)
			}()

			if r.dataFile != nil {
				_ = r.dataFile.Close()
				r.dataFile = nil
			}
		}
		if r.DataFile != "" {
			_ = os.Remove(r.DataFile)
		}
	}()
	defer func() {
		if rec := recover(); rec != nil {
			Log("Error while assembling received resource.", LOG_ERROR)
			Log("The contained exception was: "+fmt.Sprint(rec), LOG_ERROR)
			r.Status = ResourceCorrupt
		}
	}()

	r.Status = ResourceAssembling
	stream := make([]byte, 0)
	for _, part := range r.parts {
		data, ok := part.([]byte)
		if !ok || len(data) == 0 {
			continue
		}
		stream = append(stream, data...)
	}

	var data []byte
	if r.encr {
		var err error
		data, err = r.Link.Decrypt(stream)
		if err != nil {
			panic(err)
		}
	} else {
		data = stream
	}
	if len(data) < RandomHashSize {
		r.Status = ResourceCorrupt
		return
	}
	data = data[RandomHashSize:]

	if r.comp {
		reader, err := bzip2.NewReader(bytes.NewReader(data), nil)
		if err != nil {
			panic(err)
		}
		decompressed, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			panic(err)
		}
		data = decompressed
	}

	fullData := append([]byte(nil), data...)
	calculated := FullHash(append(fullData, r.randomHash...))
	if !bytes.Equal(calculated, r.hash) {
		r.Status = ResourceCorrupt
		return
	}

	payload := fullData
	if r.hasMetadata && r.segmentIndex == 1 {
		if len(fullData) < 3 {
			r.Status = ResourceCorrupt
			return
		}
		metaSize := int(fullData[0])<<16 | int(fullData[1])<<8 | int(fullData[2])
		if len(fullData) < 3+metaSize {
			r.Status = ResourceCorrupt
			return
		}
		packedMeta := fullData[3 : 3+metaSize]
		if err := os.WriteFile(r.metaStoragePath, packedMeta, 0o644); err != nil {
			panic(err)
		}
		payload = fullData[3+metaSize:]
	}

	if dir := filepath.Dir(r.DataFile); dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}
	f, err := os.OpenFile(r.DataFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		panic(err)
	}
	if _, err = f.Write(payload); err != nil {
		_ = f.Close()
		panic(err)
	}
	_ = f.Close()

	r.Status = ResourceComplete
	r.Prove(fullData)
	for i := range fullData {
		fullData[i] = 0
	}
}

// Prove sends the proof.
func (r *Resource) Prove(data []byte) {
	if r.Status == ResourceFailed {
		return
	}
	proof := FullHash(append(data, r.hash...))
	proofData := append(r.hash, proof...)
	p := NewPacket(
		r.Link,
		proofData,
		PacketTypeProof,
		PacketCONTEXT_RESOURCE_PRF,
		Broadcast,
		HeaderType1,
		nil,
		nil,
		true,
		FlagUnset,
	)
	func() {
		defer func() {
			if rec := recover(); rec != nil {
				Log("Could not send proof packet, cancelling resource", LOG_DEBUG)
				Log("The contained exception was: "+fmt.Sprint(rec), LOG_DEBUG)
				r.Cancel()
			}
		}()
		p.Send()
	}()
	if r.Status == ResourceFailed {
		return
	}
	Cache(p, true)
}

// prepareNextSegment prepares the next segment for large resources.
func (r *Resource) prepareNextSegment() {
	Log(fmt.Sprintf("Preparing segment %d of %d for resource %s", r.segmentIndex+1, r.totalSegments, r), LOG_DEBUG)
	r.preparingNext = true
	next, err := NewResource(
		nil,
		r.inputFile,
		r.Link,
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
		panic(err)
	}
	r.nextSegment = next
}

// ValidateProof validates a proof from the resource receiver.
func (r *Resource) ValidateProof(proofData []byte) {
	if r.Status == ResourceFailed {
		return
	}

	hashLen := sha256Bits / 8
	if len(proofData) != hashLen*2 {
		return
	}

	if !bytes.Equal(proofData[hashLen:], r.expectedProof) {
		return
	}

	r.Status = ResourceComplete
	if r.Link != nil {
		r.Link.ResourceConcluded(r)
	}

	if r.segmentIndex == r.totalSegments {
		if r.callback != nil {
			func() {
				defer func() {
					if rec := recover(); rec != nil {
						Log("Error while executing resource concluded callback from "+fmt.Sprint(r)+". The contained exception was: "+fmt.Sprint(rec), LOG_ERROR)
					}
				}()
				r.callback(r)
			}()
		}
		if r.inputFile != nil {
			if err := r.inputFile.Close(); err != nil {
				Log("Error while closing resource input file: "+err.Error(), LOG_ERROR)
			}
			r.inputFile = nil
		}
		if r.tempInputPath != "" {
			_ = os.Remove(r.tempInputPath)
			r.tempInputPath = ""
		}
	} else {
		if r.nextSegment == nil && !r.preparingNext {
			Log(fmt.Sprintf("Next segment preparation for resource %s was not started yet, manually preparing now. This will cause transfer slowdown.", r), LOG_WARNING)
			r.prepareNextSegment()
		}
		for r.nextSegment == nil {
			time.Sleep(50 * time.Millisecond)
		}
		r.data = nil
		r.metadata = nil
		r.parts = nil
		r.inputFile = nil
		r.Link = nil
		r.reqHashlist = nil
		r.hashmap = nil
		r.nextSegment.Advertise()
	}
}

// ReceivePart handles an incoming resource packet.
func (r *Resource) ReceivePart(packet *Packet) {
	r.receiveLock.Lock()
	r.receivingPart = true
	locked := true
	defer func() {
		if locked {
			r.receivingPart = false
			r.receiveLock.Unlock()
		}
	}()

	now := time.Now()
	r.lastActivity = now
	r.retriesLeft = r.maxRetries

	if r.reqResp.IsZero() {
		r.reqResp = now
		rtt := now.Sub(r.reqSent).Seconds()
		r.partTimeoutFactor = PartTimeoutFactorAfterRTT
		switch {
		case r.rtt == 0:
			r.rtt = r.Link.RTT.Seconds()
			r.WatchdogJob()
		case rtt < r.rtt:
			r.rtt = math.Max(r.rtt-r.rtt*0.05, rtt)
		case rtt > r.rtt:
			r.rtt = math.Min(r.rtt+r.rtt*0.05, rtt)
		}

		if rtt > 0 {
			reqRespCost := len(packet.Raw)
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

	if r.Status == ResourceFailed {
		return
	}

	r.Status = ResourceTransferring

	partData := packet.Data
	partHash := r.getMapHash(partData)

	start := r.consecutiveHeight
	if start < 0 {
		start = 0
	}
	windowEnd := start + r.window
	if windowEnd > len(r.hashmap) {
		windowEnd = len(r.hashmap)
	}

	for idx := start; idx < windowEnd; idx++ {
		mapHash := r.hashmap[idx]
		if mapHash == nil || !bytes.Equal(mapHash, partHash) {
			continue
		}
		if r.parts[idx] == nil {
			r.parts[idx] = append([]byte(nil), partData...)
			r.rttRxBytes += len(partData)
			r.receivedCount++
			r.outstanding--

			if idx == r.consecutiveHeight+1 {
				r.consecutiveHeight = idx
			}
			cp := r.consecutiveHeight + 1
			for cp < len(r.parts) && r.parts[cp] != nil {
				r.consecutiveHeight = cp
				cp++
			}

			if r.progressCallback != nil {
				func() {
					defer func() {
						if rec := recover(); rec != nil {
							Log("Error while executing progress callback from "+fmt.Sprint(r)+". The contained exception was: "+fmt.Sprint(rec), LOG_ERROR)
						}
					}()
					r.progressCallback(r)
				}()
			}
		}
	}

	if r.receivedCount == r.totalParts && !r.assemblyLock {
		r.assemblyLock = true
		r.receivingPart = false
		r.receiveLock.Unlock()
		locked = false
		go r.Assemble()
		return
	}

	shouldRequest := r.outstanding == 0
	r.receivingPart = false
	r.receiveLock.Unlock()
	locked = false

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
	for r.receivingPart {
		time.Sleep(1 * time.Millisecond)
	}

	if r.Status == ResourceFailed {
		return
	}

	if !r.waitingForHMU {
		r.outstanding = 0
		hashmapState := HashmapNotExhausted
		requested := make([]byte, 0, r.window*MapHashLen)
		requestedCount := 0

		pn := r.consecutiveHeight + 1
		searchEnd := pn + r.window
		if searchEnd > len(r.parts) {
			searchEnd = len(r.parts)
		}

		for _, part := range r.parts[pn:searchEnd] {
			if part == nil {
				if r.hashmap[pn] != nil {
					requested = append(requested, r.hashmap[pn]...)
					r.outstanding++
					requestedCount++
				} else {
					hashmapState = HashmapExhausted
				}
			}
			pn++
			if requestedCount >= r.window || hashmapState == HashmapExhausted {
				break
			}
		}

		hmuPart := []byte{hashmapState}
		if hashmapState == HashmapExhausted {
			lastIndex := r.hashmapHeight - 1
			if lastIndex < 0 {
				lastIndex = len(r.hashmap) - 1
			}
			last := r.hashmap[lastIndex]
			hmuPart = append(hmuPart, last...)
			r.waitingForHMU = true
		}

		requestData := append(hmuPart, r.hash...)
		requestData = append(requestData, requested...)

		pkt := NewPacket(
			r.Link,
			requestData,
			PacketTypeData,
			PacketCONTEXT_RESOURCE_REQ,
			Broadcast,
			HeaderType1,
			nil,
			nil,
			true,
			FlagUnset,
		)
		sentOK := true
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					sentOK = false
					Log("Could not send resource request packet, cancelling resource", LOG_DEBUG)
					Log("The contained exception was: "+fmt.Sprint(rec), LOG_DEBUG)
					r.Cancel()
				}
			}()
			pkt.Send()
		}()
		if sentOK {
			r.lastActivity = time.Now()
			r.reqSent = r.lastActivity
			r.reqSentBytes = len(pkt.Raw)
			r.reqResp = time.Time{}
		}
	}
}

// Request handles an incoming request for parts on the receiver side.
func (r *Resource) Request(requestData []byte) {
	if r.Status == ResourceFailed {
		return
	}

	rtt := time.Since(r.advSent).Seconds()
	if r.rtt == 0 {
		r.rtt = rtt
	}

	if r.Status != ResourceTransferring {
		r.Status = ResourceTransferring
		r.WatchdogJob()
	}

	r.retriesLeft = r.maxRetries

	wantsHMU := requestData[0] == HashmapExhausted
	pad := 1
	if wantsHMU {
		pad += MapHashLen
	}

	requestedHashStart := pad + sha256Bits/8
	if requestedHashStart > len(requestData) {
		requestedHashStart = len(requestData)
	}
	requestedHashes := requestData[requestedHashStart:]

	searchStart := r.receiverMinConsecutiveHeight
	searchEnd := r.receiverMinConsecutiveHeight + CollisionGuardSize
	if searchStart > len(r.parts) {
		searchStart = len(r.parts)
	}
	if searchEnd > len(r.parts) {
		searchEnd = len(r.parts)
	}

	mapHashes := make([][]byte, 0, len(requestedHashes)/MapHashLen)
	for i := 0; i+MapHashLen <= len(requestedHashes); i += MapHashLen {
		mapHashes = append(mapHashes, append([]byte(nil), requestedHashes[i:i+MapHashLen]...))
	}

	requestedParts := make([]*Packet, 0, len(mapHashes))
	for _, item := range r.parts[searchStart:searchEnd] {
		part := item.(*Packet)
		for _, mapHash := range mapHashes {
			if bytes.Equal(part.MapHash, mapHash) {
				requestedParts = append(requestedParts, part)
				break
			}
		}
	}

	for _, partPkt := range requestedParts {
		sentOK := true
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					sentOK = false
					Log("Resource could not send parts, cancelling transfer!", LOG_DEBUG)
					Log("The contained exception was: "+fmt.Sprint(rec), LOG_DEBUG)
					r.Cancel()
				}
			}()
			if !partPkt.Sent {
				partPkt.Send()
				r.sentParts++
			} else {
				partPkt.Resend()
			}
		}()
		if sentOK {
			r.lastActivity = time.Now()
			r.lastPartSent = r.lastActivity
		}
	}

	if wantsHMU {
		lastMapStart := 1
		if lastMapStart > len(requestData) {
			lastMapStart = len(requestData)
		}
		lastMapEnd := 1 + MapHashLen
		if lastMapEnd > len(requestData) {
			lastMapEnd = len(requestData)
		}
		lastMap := requestData[lastMapStart:lastMapEnd]
		collisionGuardLen := r.collisionGuardLen
		segLen := r.hashmapMaxLen
		partIndex := r.receiverMinConsecutiveHeight
		searchStart := partIndex
		searchEnd := searchStart + collisionGuardLen
		if searchStart > len(r.parts) {
			searchStart = len(r.parts)
		}
		if searchEnd > len(r.parts) {
			searchEnd = len(r.parts)
		}

		foundIndex := -1
		for idx := searchStart; idx < searchEnd; idx++ {
			partIndex++
			part := r.parts[idx].(*Packet)
			if bytes.Equal(part.MapHash, lastMap) {
				foundIndex = partIndex
				break
			}
		}

		if foundIndex == -1 {
			Log("Resource sequencing error, cancelling transfer!", LOG_ERROR)
			r.Cancel()
			return
		}

		r.receiverMinConsecutiveHeight = max(foundIndex-1-ResourceWindowMax, 0)
		if foundIndex%segLen != 0 {
			Log("Resource sequencing error, cancelling transfer!", LOG_ERROR)
			r.Cancel()
			return
		}

		segment := foundIndex / segLen
		hashmapStart := segment * segLen
		hashmapEnd := hashmapStart + segLen
		if hashmapEnd > len(r.parts) {
			hashmapEnd = len(r.parts)
		}

		hm := make([]byte, 0, (hashmapEnd-hashmapStart)*MapHashLen)
		for i := hashmapStart; i < hashmapEnd; i++ {
			hm = append(hm, r.hashmap[i]...)
		}

		hmuPayload, err := umsgpack.Packb([]any{segment, hm})
		if err != nil {
			panic(err)
		}

		hmu := append(r.hash, hmuPayload...)
		hmuPacket := NewPacket(
			r.Link,
			hmu,
			PacketTypeData,
			PacketCONTEXT_RESOURCE_HMU,
			Broadcast,
			HeaderType1,
			nil,
			nil,
			true,
			FlagUnset,
		)
		sentOK := true
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					sentOK = false
					Log("Could not send resource HMU packet, cancelling resource", LOG_DEBUG)
					Log("The contained exception was: "+fmt.Sprint(rec), LOG_DEBUG)
					r.Cancel()
				}
			}()
			hmuPacket.Send()
		}()
		if sentOK {
			r.lastActivity = time.Now()
		}
	}

	if r.sentParts == len(r.parts) {
		r.Status = ResourceAwaitingProof
		r.retriesLeft = ProofTimeoutFactor
	}

	if r.progressCallback != nil {
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					Log("Error while executing progress callback from "+fmt.Sprint(r)+". The contained exception was: "+fmt.Sprint(rec), LOG_ERROR)
				}
			}()
			r.progressCallback(r)
		}()
	}
}

// Cancel cancels the resource transfer.
func (r *Resource) Cancel() {
	if r.nextSegment != nil {
		r.nextSegment.Cancel()
	}
	if r.Status >= ResourceComplete {
		return
	}
	r.Status = ResourceFailed

	if r.initiator {
		if r.Link.Status == LinkActive {
			func() {
				defer func() {
					if rec := recover(); rec != nil {
						Log("Could not send resource cancel packet, the contained exception was: "+fmt.Sprint(rec), LOG_ERROR)
					}
				}()
				cancel := NewPacket(
					r.Link,
					r.hash,
					PacketTypeData,
					PacketCONTEXT_RESOURCE_ICL,
					Broadcast,
					HeaderType1,
					nil,
					nil,
					true,
					FlagUnset,
				)
				cancel.Send()
			}()
		}
		r.Link.CancelOutgoingResource(r)
	} else {
		r.Link.CancelIncomingResource(r)
	}

	if r.callback != nil {
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					Log("Error while executing callbacks on resource cancel from "+fmt.Sprint(r)+". The contained exception was: "+fmt.Sprint(rec), LOG_ERROR)
				}
			}()
			r.Link.ResourceConcluded(r)
			r.callback(r)
		}()
	}
}

func (r *Resource) rejected() {
	if r.Status >= ResourceComplete {
		return
	}
	if r.initiator {
		r.Status = ResourceRejected
		r.Link.CancelOutgoingResource(r)
		if r.callback != nil {
			func() {
				defer func() {
					if rec := recover(); rec != nil {
						Log("Error while executing callbacks on resource reject from "+fmt.Sprint(r)+". The contained exception was: "+fmt.Sprint(rec), LOG_ERROR)
					}
				}()
				r.Link.ResourceConcluded(r)
				go func() {
					r.callback(r)
				}()
			}()
		}
	}
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
	if r.Status == ResourceComplete && r.segmentIndex == r.totalSegments {
		return 1.0
	}

	var processed, total float64
	if r.initiator {
		if !r.split {
			processed = float64(r.sentParts)
			total = float64(r.totalParts)
		} else {
			if r.sdu == 0 {
				panic("division by zero")
			}
			maxPartsPerSegment := math.Ceil(float64(MaxEfficientSize) / float64(r.sdu))
			prevSegments := float64(r.segmentIndex - 1)
			prevParts := prevSegments * maxPartsPerSegment
			currentParts := float64(r.totalParts)
			currentFactor := 1.0
			if currentParts < maxPartsPerSegment {
				if currentParts == 0 {
					panic("division by zero")
				}
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
			if r.sdu == 0 {
				panic("division by zero")
			}
			maxPartsPerSegment := math.Ceil(float64(MaxEfficientSize) / float64(r.sdu))
			prevSegments := float64(r.segmentIndex - 1)
			prevParts := prevSegments * maxPartsPerSegment
			currentParts := float64(r.totalParts)
			currentFactor := 1.0
			if currentParts < maxPartsPerSegment {
				if currentParts == 0 {
					panic("division by zero")
				}
				currentFactor = maxPartsPerSegment / currentParts
			}
			processed = prevParts + float64(r.receivedCount)*currentFactor
			total = float64(r.totalSegments) * maxPartsPerSegment
		}
	}
	r.processedParts = processed
	r.progressTotalParts = total
	if total == 0 {
		panic("division by zero")
	}
	return math.Min(1.0, processed/total)
}

// GetSegmentProgress returns progress for the current segment.
func (r *Resource) GetSegmentProgress() float64 {
	if r.Status == ResourceComplete && r.segmentIndex == r.totalSegments {
		return 1.0
	}
	var processed float64
	if r.initiator {
		processed = float64(r.sentParts)
	} else {
		processed = float64(r.receivedCount)
	}
	if r.totalParts == 0 {
		panic("division by zero")
	}
	return math.Min(1.0, processed/float64(r.totalParts))
}

func (r *Resource) GetTransferSize() int { return r.size }
func (r *Resource) GetDataSize() int     { return r.totalSize }
func (r *Resource) GetParts() int        { return r.totalParts }
func (r *Resource) GetSegments() int     { return r.totalSegments }
func (r *Resource) GetHash() []byte      { return append([]byte(nil), r.hash...) }
func (r *Resource) IsCompressed() bool   { return r.comp }

func (r *Resource) String() string {
	return "<" + HexRep(r.hash, false) + "/" + HexRep(r.Link.LinkID, false) + ">"
}

func (a ResourceAdvertisement) Pack(segment int) []byte {
	segLen := HashmapMaxLen

	hashmapStart := segment * segLen
	hashmapEnd := hashmapStart + segLen
	if hashmapEnd > a.N {
		hashmapEnd = a.N
	}
	var hm []byte
	for i := hashmapStart; i < hashmapEnd; i++ {
		start := i * MapHashLen
		end := start + MapHashLen
		if start >= len(a.M) {
			continue
		}
		if end > len(a.M) {
			end = len(a.M)
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

	b, err := umsgpack.Packb(dict)
	if err != nil {
		Log(fmt.Sprintf("Could not pack resource advertisement: %v", err), LOG_ERROR)
		return []byte{}
	}
	return b
}

func (ResourceAdvertisement) Unpack(data []byte) (*ResourceAdvertisement, error) {
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
		return umsgpack.Unpackb(data, &dictAny)
	}(); err != nil {
		return nil, err
	}

	get := func(key string) (any, bool) {
		if v, ok := dictAny[key]; ok {
			return v, true
		}
		// Some decoders use []byte keys for msgpack "str" types.
		for k, v := range dictAny {
			switch ks := k.(type) {
			case string:
				if ks == key {
					return v, true
				}
			case []byte:
				if string(ks) == key {
					return v, true
				}
			}
		}
		return nil, false
	}

	readInt := func(key string) (int, error) {
		v, ok := get(key)
		if !ok {
			return 0, fmt.Errorf("missing msgpack key %q", key)
		}
		switch i := v.(type) {
		case int:
			return i, nil
		case int8:
			return int(i), nil
		case int16:
			return int(i), nil
		case int32:
			return int(i), nil
		case int64:
			return int(i), nil
		case uint:
			return int(i), nil
		case uint8:
			return int(i), nil
		case uint16:
			return int(i), nil
		case uint32:
			return int(i), nil
		case uint64:
			return int(i), nil
		case float32:
			return int(i), nil
		case float64:
			return int(i), nil
		default:
			return 0, fmt.Errorf("unexpected msgpack int type %T for key %q", v, key)
		}
	}
	readBytes := func(key string) ([]byte, error) {
		v, ok := get(key)
		if !ok {
			return nil, fmt.Errorf("missing msgpack key %q", key)
		}
		switch b := v.(type) {
		case []byte:
			return append([]byte(nil), b...), nil
		case string:
			return append([]byte(nil), []byte(b)...), nil
		default:
			return nil, fmt.Errorf("unexpected msgpack bytes type %T for key %q", v, key)
		}
	}
	readOptionalBytes := func(key string) ([]byte, error) {
		v, ok := get(key)
		if !ok {
			return nil, fmt.Errorf("missing msgpack key %q", key)
		}
		if v == nil {
			return nil, nil
		}
		switch b := v.(type) {
		case []byte:
			return append([]byte(nil), b...), nil
		case string:
			return append([]byte(nil), []byte(b)...), nil
		default:
			return nil, fmt.Errorf("unexpected msgpack optional bytes type %T for key %q", v, key)
		}
	}
	readByte := func(key string) (byte, error) {
		v, ok := get(key)
		if !ok {
			return 0, fmt.Errorf("missing msgpack key %q", key)
		}
		switch i := v.(type) {
		case int:
			return byte(i), nil
		case int8:
			return byte(i), nil
		case int16:
			return byte(i), nil
		case int32:
			return byte(i), nil
		case int64:
			return byte(i), nil
		case uint:
			return byte(i), nil
		case uint8:
			return byte(i), nil
		case uint16:
			return byte(i), nil
		case uint32:
			return byte(i), nil
		case uint64:
			return byte(i), nil
		case float32:
			return byte(i), nil
		case float64:
			return byte(i), nil
		default:
			return 0, fmt.Errorf("unexpected msgpack byte type %T for key %q", v, key)
		}
	}

	t, err := readInt("t")
	if err != nil {
		return nil, err
	}
	d, err := readInt("d")
	if err != nil {
		return nil, err
	}
	n, err := readInt("n")
	if err != nil {
		return nil, err
	}
	h, err := readBytes("h")
	if err != nil {
		return nil, err
	}
	r, err := readBytes("r")
	if err != nil {
		return nil, err
	}
	o, err := readBytes("o")
	if err != nil {
		return nil, err
	}
	i, err := readInt("i")
	if err != nil {
		return nil, err
	}
	l, err := readInt("l")
	if err != nil {
		return nil, err
	}
	q, err := readOptionalBytes("q")
	if err != nil {
		return nil, err
	}
	f, err := readByte("f")
	if err != nil {
		return nil, err
	}
	m, err := readBytes("m")
	if err != nil {
		return nil, err
	}

	adv := &ResourceAdvertisement{
		T: t,
		D: d,
		N: n,
		H: h,
		R: r,
		O: o,
		I: i,
		L: l,
		Q: q,
		F: f,
		M: m,
	}

	adv.E = (adv.F & 0x01) == 0x01
	adv.C = ((adv.F >> 1) & 0x01) == 0x01
	adv.S = ((adv.F >> 2) & 0x01) == 0x01
	adv.U = ((adv.F >> 3) & 0x01) == 0x01
	adv.P = ((adv.F >> 4) & 0x01) == 0x01
	adv.X = ((adv.F >> 5) & 0x01) == 0x01
	return adv, nil
}

func (a ResourceAdvertisement) GetTransferSize() int { return a.T }
func (a ResourceAdvertisement) GetDataSize() int     { return a.D }
func (a ResourceAdvertisement) GetParts() int        { return a.N }
func (a ResourceAdvertisement) GetSegments() int     { return a.L }
func (a ResourceAdvertisement) GetHash() []byte      { return append([]byte(nil), a.H...) }
func (a ResourceAdvertisement) IsCompressed() bool   { return a.C }
func (a ResourceAdvertisement) HasMetadata() bool    { return a.X }
func (a ResourceAdvertisement) GetLink() *Link       { return a.Link }

func (a *ResourceAdvertisement) IsRequest() bool {
	return a.Q != nil && a.U
}

func (a *ResourceAdvertisement) IsResponse() bool {
	return a.Q != nil && a.P
}

func (ResourceAdvertisement) ReadRequestID(advertisementPacket *Packet) []byte {
	adv, err := (ResourceAdvertisement{}).Unpack(advertisementPacket.Plaintext)
	if err != nil {
		panic(err)
	}
	return append([]byte(nil), adv.Q...)
}

func (ResourceAdvertisement) ReadTransferSize(advertisementPacket *Packet) int {
	adv, err := (ResourceAdvertisement{}).Unpack(advertisementPacket.Plaintext)
	if err != nil {
		panic(err)
	}
	return adv.T
}

func (ResourceAdvertisement) ReadSize(advertisementPacket *Packet) int {
	adv, err := (ResourceAdvertisement{}).Unpack(advertisementPacket.Plaintext)
	if err != nil {
		panic(err)
	}
	return adv.D
}
