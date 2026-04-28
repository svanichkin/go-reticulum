package rns

import (
	"bytes"
	"crypto/ecdh"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	Cryptography "github.com/svanichkin/go-reticulum/rns/cryptography"
	umsgpack "github.com/svanichkin/go-reticulum/rns/vendor"
)

// ---- helper types ----

type Callbacks struct {
	LinkEstablished func(*Link)
	Packet          func([]byte, *Packet)
	ProofRequested  func(*Packet) bool
}

// Request policies
const (
	DestinationSINGLE = 0x00
	DestinationGROUP  = 0x01
	DestinationPLAIN  = 0x02
	DestinationLINK   = 0x03

	DestinationPROVE_NONE = 0x21
	DestinationPROVE_APP  = 0x22
	DestinationPROVE_ALL  = 0x23

	DestinationALLOW_NONE = 0x00
	DestinationALLOW_ALL  = 0x01
	DestinationALLOW_LIST = 0x02

	DestinationIN  = 0x11
	DestinationOUT = 0x12

	DestinationPR_TAG_WINDOW = 30

	DestinationRATCHET_COUNT    = 512
	DestinationRATCHET_INTERVAL = 30 * 60
)

// Backwards/porting aliases used by older code.
const (
	DestPlain  = DestinationPLAIN
	DestSingle = DestinationSINGLE
	DestGroup  = DestinationGROUP
	DestLink   = DestinationLINK
)

// ---- Request handler ----

type RequestHandler struct {
	Path         string
	ResponseGen  func(path string, data any, requestID []byte, linkID []byte, remoteIdentity *Identity, requestedAt time.Time) any
	AllowPolicy  int
	AllowedList  [][]byte
	AutoCompress interface{} // bool or int
}

// ---- Destination ----

type Destination struct {
	// configuration
	Type      int
	Direction int

	AcceptLinkRequests bool
	Callbacks          Callbacks

	ProofStrategy int

	Identity *Identity
	Name     string
	Hash     []byte
	NameHash []byte
	HexHash  string

	DefaultAppData interface{}
	Callback       any
	ProofCallback  any
	// StampCost mirrors Python Destination.stamp_cost and intentionally allows
	// three states: nil, int, and false.
	StampCost any

	// ratchets
	Ratchets          [][]byte
	RatchetsPath      string
	RatchetInterval   int
	ratchetFileLock   sync.Mutex
	RetainedRatchets  int
	LatestRatchetTime float64
	LatestRatchetID   []byte
	enforceRatchets   bool

	// misc
	MTU             int
	PathResponses   map[string]*pathResponseEntry
	RequestHandlers map[string]*RequestHandler
	Links           []*Link
	linksMu         sync.Mutex

	groupTokenMu    sync.Mutex
	groupTokenBytes []byte
	groupToken      *Cryptography.Token
	PrvBytes        []byte
	Prv             *Cryptography.Token
}

type pathResponseEntry struct {
	Timestamp float64
	Data      []byte
}

type DestinationTypeError struct {
	Message string
}

func (e *DestinationTypeError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

type DestinationValueError struct {
	Message string
}

func (e *DestinationValueError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func str(d *Destination) string {
	if d == nil {
		return "<nil>"
	}
	return "<" + d.Name + ":" + d.HexHash + ">"
}

func pathResponseKey(tag []byte) string {
	if tag == nil {
		return "\x00nil"
	}
	return string(tag)
}

// ---- static helpers (internal to package) ----

func (*Destination) ExpandName(identity *Identity, appName string, aspects ...string) (string, error) {
	if strings.Contains(appName, ".") {
		return "", errors.New("dots can't be used in app names")
	}
	name := appName
	for _, a := range aspects {
		if strings.Contains(a, ".") {
			return "", errors.New("dots can't be used in aspects")
		}
		name += "." + a
	}
	if identity != nil {
		name += "." + identity.HexHash
	}
	return name, nil
}

func destinationHash(identity interface{}, appName string, aspects ...string) ([]byte, error) {
	// Python parity: Destination.hash() computes name_hash from expand_name(None,...),
	// ie. without appending identity hexhash, even when an identity is supplied.
	name, err := (&Destination{}).ExpandName(nil, appName, aspects...)
	if err != nil {
		return nil, err
	}
	nameHash := FullHash([]byte(name))[:IdentityNameHashLength/8]
	addrHashMaterial := append([]byte{}, nameHash...)

	if identity != nil {
		switch v := identity.(type) {
		case *Identity:
			if v == nil {
				break
			}
			addrHashMaterial = append(addrHashMaterial, v.Hash...)
		case []byte:
			if len(v) == ReticulumTruncatedHashLength/8 {
				addrHashMaterial = append(addrHashMaterial, v...)
			} else {
				return nil, errors.New("invalid material supplied for destination hash calculation")
			}
		default:
			return nil, errors.New("invalid material supplied for destination hash calculation")
		}
	}

	full := FullHash(addrHashMaterial)
	return full[:ReticulumTruncatedHashLength/8], nil
}

func (*Destination) AppAndAspectsFromName(fullName string) (string, []string) {
	parts := strings.Split(fullName, ".")
	if len(parts) == 0 {
		return "", nil
	}
	app := parts[0]
	aspects := []string{}
	if len(parts) > 1 {
		aspects = parts[1:]
	}
	return app, aspects
}

func (*Destination) HashFromNameAndIdentity(fullName string, identity *Identity) ([]byte, error) {
	app, aspects := (&Destination{}).AppAndAspectsFromName(fullName)
	return destinationHash(identity, app, aspects...)
}

// ---- constructor ----

func NewDestination(identity *Identity, direction int, dstType int, appName string, aspects ...string) (*Destination, error) {
	if strings.Contains(appName, ".") {
		return nil, errors.New("dots can't be used in app names")
	}
	if dstType != DestinationSINGLE && dstType != DestinationGROUP && dstType != DestinationPLAIN && dstType != DestinationLINK {
		return nil, errors.New("unknown destination type")
	}
	if direction != DestinationIN && direction != DestinationOUT {
		return nil, errors.New("unknown destination direction")
	}

	d := &Destination{
		Type:               dstType,
		Direction:          direction,
		AcceptLinkRequests: true,
		Callbacks:          Callbacks{},
		ProofStrategy:      DestinationPROVE_NONE,
		Ratchets:           nil,
		RatchetsPath:       "",
		RatchetInterval:    DestinationRATCHET_INTERVAL,
		RetainedRatchets:   DestinationRATCHET_COUNT,
		enforceRatchets:    false,
		MTU:                0,
		PathResponses:      make(map[string]*pathResponseEntry),
		RequestHandlers:    make(map[string]*RequestHandler),
		Links:              []*Link{},
	}

	if identity == nil && direction == DestinationIN && dstType != DestinationPLAIN {
		newIdentity, err := NewIdentity()
		if err != nil {
			return nil, fmt.Errorf("could not create identity for destination: %w", err)
		}
		identity = newIdentity
		aspects = append(aspects, identity.HexHash)
	}

	if identity == nil && direction == DestinationOUT && dstType != DestinationPLAIN {
		return nil, errors.New("can't create outbound SINGLE destination without an identity")
	}

	if identity != nil && dstType == DestinationPLAIN {
		return nil, errors.New("selected destination type PLAIN cannot hold an identity")
	}

	d.Identity = identity

	nameWithIdentity, err := (&Destination{}).ExpandName(identity, appName, aspects...)
	if err != nil {
		return nil, err
	}
	d.Name = nameWithIdentity

	hash, err := destinationHash(identity, appName, aspects...)
	if err != nil {
		return nil, err
	}
	d.Hash = hash

	nameWithoutIdentity, err := (&Destination{}).ExpandName(nil, appName, aspects...)
	if err != nil {
		return nil, err
	}
	d.NameHash = FullHash([]byte(nameWithoutIdentity))[:IdentityNameHashLength/8]
	d.HexHash = HexRep(hash, false)

	// registration
	if err := RegisterDestination(d); err != nil {
		return nil, err
	}

	return d, nil
}

// ---- methods ----

// AcceptsLinks sets or queries whether the destination allows incoming link
// requests. Call without arguments to query, or pass a bool to update.
func (d *Destination) AcceptsLinks(accept ...bool) bool {
	if len(accept) == 0 {
		return d.AcceptLinkRequests
	}
	d.AcceptLinkRequests = accept[0]
	return d.AcceptLinkRequests
}

func (d *Destination) SetLinkEstablishedCallback(cb func(*Link)) {
	d.Callbacks.LinkEstablished = cb
}

func (d *Destination) SetPacketCallback(cb func([]byte, *Packet)) {
	d.Callbacks.Packet = cb
}

func (d *Destination) SetProofRequestedCallback(cb func(*Packet) bool) {
	d.Callbacks.ProofRequested = cb
}

func (d *Destination) SetProofStrategy(strategy int) {
	if strategy != DestinationPROVE_NONE && strategy != DestinationPROVE_APP && strategy != DestinationPROVE_ALL {
		panic(&DestinationTypeError{Message: "Unsupported proof strategy"})
	}
	d.ProofStrategy = strategy
}

func (d *Destination) GetProofStrategy() int {
	return d.ProofStrategy
}

func (d *Destination) SetStampCost(stampCost any) {
	switch v := stampCost.(type) {
	case nil:
		d.StampCost = nil
		return
	case bool:
		if v {
			panic(&DestinationValueError{Message: "Stamp cost can only be false, nil, or an integer"})
		}
		d.StampCost = false
		return
	}

	value := reflect.ValueOf(stampCost)
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		cost := int(value.Int())
		if cost < 1 {
			d.StampCost = nil
		} else if cost >= 255 {
			d.StampCost = false
		} else {
			d.StampCost = cost
		}
		return

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		cost := value.Uint()
		if cost < 1 {
			d.StampCost = nil
		} else if cost >= 255 {
			d.StampCost = false
		} else {
			d.StampCost = int(cost)
		}
		return
	}

	panic(&DestinationValueError{Message: "Stamp cost can only be false, nil, or an integer"})
}

// IncomingLinkRequest mirrors Destination.incoming_link_request() in Python.
// It validates the link request and appends it to the destination if accepted.
func (d *Destination) IncomingLinkRequest(data []byte, packet *Packet) {
	if !d.AcceptLinkRequests {
		return
	}
	destinationIncomingLinkRequest(d, data, packet)
}

// internal
func (d *Destination) cleanRatchets() {
	if d.Ratchets != nil && len(d.Ratchets) > d.RetainedRatchets {
		n := len(d.Ratchets)
		if n > DestinationRATCHET_COUNT {
			n = DestinationRATCHET_COUNT
		}
		d.Ratchets = d.Ratchets[:n]
	}
}

func (d *Destination) persistRatchets() error {
	if d.RatchetsPath == "" {
		return errors.New("no ratchets path set")
	}

	d.ratchetFileLock.Lock()
	defer d.ratchetFileLock.Unlock()

	tempPath := d.RatchetsPath + ".tmp"

	packedRatchets, err := umsgpack.Packb(d.Ratchets)
	if err != nil {
		return err
	}

	signature := d.Sign(packedRatchets)
	persisted := map[string]interface{}{
		"signature": signature,
		"ratchets":  packedRatchets,
	}

	buf, err := umsgpack.Packb(persisted)
	if err != nil {
		return err
	}

	if err := os.WriteFile(tempPath, buf, 0o600); err != nil {
		TraceException(err)
		d.Ratchets = nil
		d.RatchetsPath = ""
		Log("Could not write ratchet file contents for "+str(d)+". The contained exception was: "+err.Error(), LOG_ERROR)
		return errors.New("could not write ratchet file contents for " + str(d) + ": " + err.Error())
	}

	if _, err := os.Stat(d.RatchetsPath); err == nil {
		_ = os.Remove(d.RatchetsPath)
	}
	if err := os.Rename(tempPath, d.RatchetsPath); err != nil {
		TraceException(err)
		d.Ratchets = nil
		d.RatchetsPath = ""
		Log("Could not write ratchet file contents for "+str(d)+". The contained exception was: "+err.Error(), LOG_ERROR)
		return errors.New("could not move ratchet temp file for " + str(d) + ": " + err.Error())
	}

	return nil
}

func (d *Destination) RotateRatchets() bool {
	if d.Ratchets == nil {
		panic(errors.New("Cannot rotate ratchet on " + str(d) + ", ratchets are not enabled"))
	}
	// Python uses time.time() (float seconds).
	now := float64(time.Now().UnixNano()) / 1e9
	if now > d.LatestRatchetTime+float64(d.RatchetInterval) {
		Log("Rotating ratchets for "+str(d), LOG_DEBUG)
		newRatchet := IdentityGenerateRatchet()
		d.Ratchets = append([][]byte{newRatchet}, d.Ratchets...)
		d.LatestRatchetTime = now
		d.cleanRatchets()
		if err := d.persistRatchets(); err != nil {
			panic(err)
		}
	}
	return true
}

func (d *Destination) Announce(appData []byte, pathResponse bool, attachedInterface *Interface, tag []byte, send bool) *Packet {
	if d.Type != DestinationSINGLE {
		panic(&DestinationTypeError{Message: "Only SINGLE destination types can be announced"})
	}
	if d.Direction != DestinationIN {
		panic(&DestinationTypeError{Message: "Only IN destination types can be announced"})
	}

	var ratchetPub []byte
	// Python uses time.time() (float seconds).
	now := float64(time.Now().UnixNano()) / 1e9

	// purge old path_responses
	for k, v := range d.PathResponses {
		if now > v.Timestamp+DestinationPR_TAG_WINDOW {
			delete(d.PathResponses, k)
		}
	}

	var announceData []byte

	if pathResponse && tag != nil {
		if entry, ok := d.PathResponses[pathResponseKey(tag)]; ok {
			Log("Using cached announce data for answering path request with tag "+PrettyHexRep(tag), LOG_EXTREME)
			announceData = entry.Data
		}
	}

	if announceData == nil {
		destHash := d.Hash
		randomBytes := IdentityGetRandomHash()
		randomHash := make([]byte, 10)
		copy(randomHash[:5], randomBytes[:5])
		var tsBuf [8]byte
		binary.BigEndian.PutUint64(tsBuf[:], uint64(time.Now().Unix()))
		copy(randomHash[5:], tsBuf[3:])

		if d.Ratchets != nil {
			_ = d.RotateRatchets()
			if len(d.Ratchets) > 0 {
				ratchetPub = IdentityRatchetPublicBytes(d.Ratchets[0])
				IdentityRememberRatchet(destHash, ratchetPub)
			}
		}

		if appData == nil && d.DefaultAppData != nil {
			switch v := d.DefaultAppData.(type) {
			case []byte:
				appData = v
			case func() []byte:
				res := v()
				if res != nil {
					appData = res
				}
			}
		}

		signed := append([]byte{}, destHash...)
		signed = append(signed, d.Identity.GetPublicKey()...)
		signed = append(signed, d.NameHash...)
		signed = append(signed, randomHash...)
		signed = append(signed, ratchetPub...)
		if appData != nil {
			signed = append(signed, appData...)
		}

		signature, err := d.Identity.Sign(signed)
		if err != nil {
			panic(errors.New("Failed to sign announce for " + str(d) + ": " + err.Error()))
		}

		announceData = append([]byte{}, d.Identity.GetPublicKey()...)
		announceData = append(announceData, d.NameHash...)
		announceData = append(announceData, randomHash...)
		announceData = append(announceData, ratchetPub...)
		announceData = append(announceData, signature...)
		if appData != nil {
			announceData = append(announceData, appData...)
		}

		key := pathResponseKey(tag)
		d.PathResponses[key] = &pathResponseEntry{
			Timestamp: float64(time.Now().UnixNano()) / 1e9,
			Data:      announceData,
		}
	}

	var announceContext byte
	if pathResponse {
		announceContext = PacketPATH_RESPONSE
	} else {
		announceContext = PacketNONE
	}

	var contextFlag byte
	if len(ratchetPub) > 0 {
		contextFlag = PacketFLAG_SET
	} else {
		contextFlag = PacketFLAG_UNSET
	}

	pkt := NewPacket(
		d,
		announceData,
		PacketANNOUNCE,
		announceContext,
		Broadcast,
		HeaderType1,
		nil,
		attachedInterface,
		true,
		contextFlag,
	)
	if send {
		_ = pkt.Send()
		return nil
	}
	return pkt
}

func (d *Destination) RegisterRequestHandler(
	path string,
	responseGen func(path string, data any, requestID []byte, linkID []byte, remoteIdentity *Identity, requestedAt time.Time) any,
	allow int,
	allowedList [][]byte,
	autoCompress ...interface{},
) error {
	if path == "" {
		panic(&DestinationValueError{Message: "Invalid path specified"})
	}
	if responseGen == nil {
		panic(&DestinationValueError{Message: "Invalid response generator specified"})
	}
	if allow != DestinationALLOW_NONE && allow != DestinationALLOW_ALL && allow != DestinationALLOW_LIST {
		panic(&DestinationValueError{Message: "Invalid request policy"})
	}
	pathHash := TruncatedHash([]byte(path))
	var auto interface{} = true
	if len(autoCompress) > 0 {
		auto = autoCompress[0]
	}

	d.RequestHandlers[string(pathHash)] = &RequestHandler{
		Path:         path,
		ResponseGen:  responseGen,
		AllowPolicy:  allow,
		AllowedList:  allowedList,
		AutoCompress: auto,
	}
	return nil
}

func (d *Destination) DeregisterRequestHandler(path string) bool {
	pathHash := TruncatedHash([]byte(path))
	key := string(pathHash)
	if _, ok := d.RequestHandlers[key]; ok {
		delete(d.RequestHandlers, key)
		return true
	}
	return false
}

func (d *Destination) Receive(packet *Packet) bool {
	if packet.PacketType == PacketLINKREQUEST {
		plaintext := packet.Data
		destinationIncomingLinkRequest(d, plaintext, packet)
		return true
	}

	plaintext := d.Decrypt(packet.Data)
	packet.RatchetID = d.LatestRatchetID
	if plaintext == nil {
		return false
	}

	if packet.PacketType == PacketDATA {
		if d.Callbacks.Packet != nil {
			func() {
				defer func() {
					if r := recover(); r != nil {
						Log(fmt.Sprintf("Error while executing receive callback from %s. The contained exception was: %v", str(d), r), LOG_ERROR)
					}
				}()
				d.Callbacks.Packet(plaintext, packet)
			}()
		}
	}
	return true
}

func destinationIncomingLinkRequest(d *Destination, data []byte, packet *Packet) {
	if !d.AcceptLinkRequests {
		return
	}
	link := LinkValidateRequest(d, data, packet)
	if link == nil {
		return
	}
	d.linksMu.Lock()
	d.Links = append(d.Links, link)
	d.linksMu.Unlock()
}

func destinationDispatchRequest(d *Destination, path string, data any, requestID []byte, linkID []byte, remoteIdentity *Identity, requestedAt time.Time) (any, bool) {
	handler, ok := d.RequestHandlers[string(TruncatedHash([]byte(path)))]
	if !ok || handler == nil {
		return nil, false
	}
	if !destinationRequestAllowed(handler, remoteIdentity) {
		return nil, false
	}
	resp := handler.ResponseGen(path, data, requestID, linkID, remoteIdentity, requestedAt)
	return resp, true
}

func destinationRequestAllowed(handler *RequestHandler, remote *Identity) bool {
	switch handler.AllowPolicy {
	case DestinationALLOW_ALL:
		return true
	case DestinationALLOW_NONE:
		return false
	case DestinationALLOW_LIST:
		if remote == nil {
			return false
		}
		for _, entry := range handler.AllowedList {
			if bytes.Equal(entry, remote.Hash) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func (d *Destination) reloadRatchets(path string) error {
	if _, err := os.Stat(path); err != nil {
		Log("No existing ratchet data found, initialising new ratchet file for "+str(d), LOG_DEBUG)
		d.Ratchets = [][]byte{}
		d.RatchetsPath = path
		return d.persistRatchets()
	}

	d.ratchetFileLock.Lock()
	defer d.ratchetFileLock.Unlock()

	loadAttempt := func() error {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var persisted map[string]interface{}
		if err := umsgpack.Unpackb(data, &persisted); err != nil {
			return err
		}
		sig, ok1 := persisted["signature"].([]byte)
		rawRatchets, ok2 := persisted["ratchets"].([]byte)
		if !ok1 || !ok2 {
			return errors.New("invalid ratchet file format")
		}
		if !d.Identity.Validate(sig, rawRatchets) {
			return errors.New("invalid ratchet file signature")
		}
		var ratchets [][]byte
		if err := umsgpack.Unpackb(rawRatchets, &ratchets); err != nil {
			return err
		}
		d.Ratchets = ratchets
		d.RatchetsPath = path
		return nil
	}

	if err := loadAttempt(); err != nil {
		TraceException(err)
		Log("First ratchet reload attempt for "+str(d)+" failed. Possible I/O conflict. Retrying in 500ms.", LOG_ERROR)
		time.Sleep(500 * time.Millisecond)
		if err2 := loadAttempt(); err2 != nil {
			d.Ratchets = nil
			d.RatchetsPath = ""
			TraceException(err2)
			Log("The ratchet file located at "+path+" could not be loaded. This could indicate that the ratchet file has become corrupt.", LOG_CRITICAL)
			Log("You can attempt to manually recover the ratchet file, or simply remove it to have Reticulum recreate it on the next use.", LOG_CRITICAL)
			Log("If re-initialize this ratchet file, make sure to send an announce for the relevant destination as soon as possible,", LOG_CRITICAL)
			Log("so that the new ratchet information is synchronized to the network.", LOG_CRITICAL)
			return errors.New("could not read ratchet file contents for " + str(d) + ": " + err2.Error())
		}
		Log("Ratchet reload retry succeeded", LOG_DEBUG)
	}
	return nil
}

func (d *Destination) EnableRatchets(path string) (bool, error) {
	if path == "" {
		panic(&DestinationValueError{Message: "No ratchet file path specified for " + str(d)})
	}
	d.LatestRatchetTime = 0
	if err := d.reloadRatchets(path); err != nil {
		panic(err)
	}
	Log("Ratchets enabled on "+str(d), LOG_DEBUG)
	return true, nil
}

func (d *Destination) EnforceRatchets() bool {
	if d.Ratchets == nil {
		return false
	}
	d.enforceRatchets = true
	Log("Ratchets enforced on "+str(d), LOG_DEBUG)
	return true
}

func (d *Destination) SetRetainedRatchets(n int) bool {
	if n <= 0 {
		return false
	}
	d.RetainedRatchets = n
	d.cleanRatchets()
	return true
}

func (d *Destination) SetRatchetInterval(interval int) bool {
	if interval <= 0 {
		return false
	}
	d.RatchetInterval = interval
	return true
}

func (d *Destination) CreateKeys() error {
	if d.Type == DestinationPLAIN {
		panic(&DestinationTypeError{Message: "A plain destination does not hold any keys"})
	}
	if d.Type == DestinationSINGLE {
		panic(&DestinationTypeError{Message: "A single destination holds keys through an Identity instance"})
	}
	if d.Type == DestinationGROUP {
		key, err := Cryptography.GenerateKey(32)
		if err != nil {
			panic(err)
		}
		d.LoadPrivateKey(key)
	}
	return nil
}

func (d *Destination) Encrypt(plaintext []byte) []byte {
	if d.Type == DestinationPLAIN {
		return plaintext
	}
	if d.Type == DestinationSINGLE && d.Identity != nil {
		selectedRatchet := IdentityGetRatchet(d.Hash)
		if selectedRatchet != nil {
			d.LatestRatchetID = IdentityGetRatchetID(selectedRatchet)
		}
		ct, err := d.Identity.Encrypt(plaintext, selectedRatchet)
		if err != nil {
			Log("Failed to encrypt payload for "+str(d)+": "+err.Error(), LOG_ERROR)
			return nil
		}
		return ct
	}
	if d.Type == DestinationGROUP {
		d.groupTokenMu.Lock()
		tok := d.groupToken
		d.groupTokenMu.Unlock()
		if tok != nil {
			ct, err := tok.Encrypt(plaintext)
			if err != nil {
				Log("The GROUP destination could not encrypt data", LOG_ERROR)
				Log("The contained exception was: "+err.Error(), LOG_ERROR)
				return nil
			}
			return ct
		}
		Log("No private key held by GROUP destination. Did you create or load one?", LOG_ERROR)
		panic(&DestinationValueError{Message: "No private key held by GROUP destination. Did you create or load one?"})
	}
	return nil
}

func (d *Destination) Decrypt(ciphertext []byte) []byte {
	if d.Type == DestinationPLAIN {
		return ciphertext
	}
	if d.Type == DestinationSINGLE && d.Identity != nil {
		d.LatestRatchetID = nil
		if len(d.Ratchets) > 0 {
			if d.Identity.prv == nil {
				panic(errors.New("Decryption failed because identity does not hold a private key"))
			}
			if len(ciphertext) <= x25519KeyLen {
				Log("Decryption failed because the token size was invalid.", LogDebug)
				return nil
			}
			if d.Identity.curve == nil {
				d.Identity.curve = ecdh.X25519()
			}

			peerPubBytes := ciphertext[:x25519KeyLen]
			ciphertextBody := ciphertext[x25519KeyLen:]

			peerPub, err := d.Identity.curve.NewPublicKey(peerPubBytes)
			if err != nil {
				Log(fmt.Sprintf("Decryption by %s failed: %v", PrettyHexRep(d.Identity.Hash), err), LogDebug)
				return nil
			}

			decrypted := []byte(nil)
			for _, ratchet := range d.Ratchets {
				if ratchet == nil {
					continue
				}
				ratchetPrv, err := d.Identity.curve.NewPrivateKey(ratchet)
				if err != nil {
					continue
				}
				shared, err := ratchetPrv.ECDH(peerPub)
				if err != nil {
					continue
				}
				pt, err := d.Identity.decryptWithShared(shared, ciphertextBody)
				if err == nil {
					ratchetPub := ratchetPrv.PublicKey().Bytes()
					ratchetID := IdentityGetRatchetID(ratchetPub)
					if len(ratchetID) == 0 {
						d.LatestRatchetID = nil
					} else {
						d.LatestRatchetID = append([]byte{}, ratchetID...)
					}
					decrypted = pt
					break
				}
			}

			if d.enforceRatchets && decrypted == nil {
				Log(fmt.Sprintf("Decryption with ratchet enforcement by %s failed. Dropping packet.", PrettyHexRep(d.Identity.Hash)), LogDebug)
				d.LatestRatchetID = nil
				return nil
			}

			if decrypted == nil {
				shared, err := d.Identity.prv.ECDH(peerPub)
				if err != nil {
					Log(fmt.Sprintf("Decryption by %s failed: %v", PrettyHexRep(d.Identity.Hash), err), LogDebug)
					d.LatestRatchetID = nil
					return nil
				}
				pt, err := d.Identity.decryptWithShared(shared, ciphertextBody)
				if err != nil {
					Log(fmt.Sprintf("Decryption by %s failed: %v", PrettyHexRep(d.Identity.Hash), err), LogDebug)
					d.LatestRatchetID = nil
					return nil
				}
				d.LatestRatchetID = nil
				return pt
			}
			return decrypted
		}
		decrypted, _ := d.Identity.Decrypt(ciphertext, nil, d.enforceRatchets)
		d.LatestRatchetID = nil
		return decrypted
	}

	if d.Type == DestinationGROUP {
		d.groupTokenMu.Lock()
		tok := d.groupToken
		d.groupTokenMu.Unlock()
		if tok != nil {
			pt, err := tok.Decrypt(ciphertext)
			if err != nil {
				Log("The GROUP destination could not decrypt data", LOG_ERROR)
				Log("The contained exception was: "+err.Error(), LOG_ERROR)
				return nil
			}
			return pt
		}
		Log("No private key held by GROUP destination. Did you create or load one?", LOG_ERROR)
		panic(&DestinationValueError{Message: "No private key held by GROUP destination. Did you create or load one?"})
	}
	return nil
}

func (d *Destination) Sign(message []byte) []byte {
	if d.Type == DestinationSINGLE && d.Identity != nil {
		sig, err := d.Identity.Sign(message)
		if err != nil {
			Log("Failed to sign message for "+str(d)+": "+err.Error(), LOG_ERROR)
			return nil
		}
		return sig
	}
	return nil
}

func (d *Destination) SetDefaultAppData(appData interface{}) {
	d.DefaultAppData = appData
}

func (d *Destination) ClearDefaultAppData() {
	d.SetDefaultAppData(nil)
}

func (d *Destination) LoadPrivateKey(key []byte) {
	if d.Type == DestinationPLAIN {
		panic(&DestinationTypeError{Message: "A plain destination does not hold any keys"})
	}
	if d.Type == DestinationSINGLE {
		panic(&DestinationTypeError{Message: "A single destination holds keys through an Identity instance"})
	}
	tok, err := Cryptography.NewToken(key)
	if err != nil {
		panic(err)
	}
	d.groupTokenMu.Lock()
	defer d.groupTokenMu.Unlock()
	d.groupTokenBytes = append([]byte{}, key...)
	d.groupToken = tok
	d.PrvBytes = append([]byte{}, key...)
	d.Prv = tok
}

func (d *Destination) GetPrivateKey() []byte {
	if d.Type == DestinationPLAIN {
		panic(&DestinationTypeError{Message: "A plain destination does not hold any keys"})
	}
	if d.Type == DestinationSINGLE {
		panic(&DestinationTypeError{Message: "A single destination holds keys through an Identity instance"})
	}
	d.groupTokenMu.Lock()
	defer d.groupTokenMu.Unlock()
	return append([]byte{}, d.groupTokenBytes...)
}

func (d *Destination) LoadPublicKey(_ []byte) {
	if d.Type != DestinationSINGLE {
		panic(&DestinationTypeError{Message: "Only the \"single\" destination type can hold a public key"})
	}
	panic(&DestinationTypeError{Message: "A single destination holds keys through an Identity instance"})
}
