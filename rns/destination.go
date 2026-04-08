package rns

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
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

	acceptLinkRequests bool
	callbacks          Callbacks

	proofStrategy int

	identity *Identity

	name     string
	hash     []byte
	nameHash []byte
	hexhash  string

	defaultAppData interface{}

	// ratchets
	ratchets          [][]byte
	ratchetsPath      string
	ratchetInterval   int
	ratchetFileLock   sync.Mutex
	retainedRatchets  int
	latestRatchetTime float64
	latestRatchetID   []byte
	enforceRatchets   bool

	// misc
	mtu             int
	pathResponses   map[string]*pathResponseEntry
	requestHandlers map[string]*RequestHandler
	links           []*Link
	linksMu         sync.Mutex

	groupTokenMu    sync.Mutex
	groupTokenBytes []byte
	groupToken      *Cryptography.Token
}

type pathResponseEntry struct {
	Timestamp float64
	Data      []byte
}

type DestinationStateError struct {
	Message string
}

func (e *DestinationStateError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func panicDestinationState(message string) {
	panic(&DestinationStateError{Message: message})
}

// ---- static helpers (like @staticmethod in Python) ----

// ExpandName = Destination.expand_name(...)
func DestinationExpandName(identity *Identity, appName string, aspects ...string) (string, error) {
	if containsDot(appName) {
		return "", errors.New("dots can't be used in app names")
	}
	name := appName
	for _, a := range aspects {
		if containsDot(a) {
			return "", errors.New("dots can't be used in aspects")
		}
		name += "." + a
	}
	if identity != nil {
		name += "." + identity.HexHash
	}
	return name, nil
}

// Hash = Destination.hash(...)
func DestinationHash(identity interface{}, appName string, aspects ...string) ([]byte, error) {
	// Python parity: Destination.hash() computes name_hash from expand_name(None,...),
	// ie. without appending identity hexhash, even when an identity is supplied.
	name, err := DestinationExpandName(nil, appName, aspects...)
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

// AppAndAspectsFromName = Destination.app_and_aspects_from_name(...)
func DestinationAppAndAspectsFromName(fullName string) (string, []string) {
	parts := splitDot(fullName)
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

// HashFromNameAndIdentity = Destination.hash_from_name_and_identity(...)
func DestinationHashFromNameAndIdentity(fullName string, identity *Identity) ([]byte, error) {
	app, aspects := DestinationAppAndAspectsFromName(fullName)
	return DestinationHash(identity, app, aspects...)
}

// HashFromNameAndIdentity is a compatibility helper used by some Go utilities.
// The Python tooling takes the identity hash bytes directly; for hashing we only
// need the Identity.Hash field, not the key material.
func HashFromNameAndIdentity(fullName string, identityHash []byte) []byte {
	if len(identityHash) == 0 {
		return nil
	}
	id := &Identity{Hash: identityHash}
	out, err := DestinationHashFromNameAndIdentity(fullName, id)
	if err != nil {
		return nil
	}
	return out
}

// ---- constructor ----

func NewDestination(identity *Identity, direction int, dstType int, appName string, aspects ...string) (*Destination, error) {
	if containsDot(appName) {
		return nil, errors.New("dots can't be used in app names")
	}
	if !containsInt(dstType, []int{DestinationSINGLE, DestinationGROUP, DestinationPLAIN, DestinationLINK}) {
		return nil, errors.New("unknown destination type")
	}
	if !containsInt(direction, []int{DestinationIN, DestinationOUT}) {
		return nil, errors.New("unknown destination direction")
	}

	d := &Destination{
		Type:               dstType,
		Direction:          direction,
		acceptLinkRequests: true,
		callbacks:          Callbacks{},
		proofStrategy:      DestinationPROVE_NONE,
		ratchets:           nil,
		ratchetsPath:       "",
		ratchetInterval:    DestinationRATCHET_INTERVAL,
		retainedRatchets:   DestinationRATCHET_COUNT,
		enforceRatchets:    false,
		mtu:                0,
		pathResponses:      make(map[string]*pathResponseEntry),
		requestHandlers:    make(map[string]*RequestHandler),
		links:              []*Link{},
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

	d.identity = identity

	nameWithIdentity, err := DestinationExpandName(identity, appName, aspects...)
	if err != nil {
		return nil, err
	}
	d.name = nameWithIdentity

	hash, err := DestinationHash(identity, appName, aspects...)
	if err != nil {
		return nil, err
	}
	d.hash = hash

	nameWithoutIdentity, err := DestinationExpandName(nil, appName, aspects...)
	if err != nil {
		return nil, err
	}
	d.nameHash = FullHash([]byte(nameWithoutIdentity))[:IdentityNameHashLength/8]
	d.hexhash = PrettyHexRep(hash) // or hex.EncodeToString(hash)

	// registration
	TransportRegisterDestination(d)

	return d, nil
}

// ---- methods ----

func (d *Destination) String() string {
	return "<" + d.name + ":" + d.hexhash + ">"
}

// AcceptsLinks sets or queries whether the destination allows incoming link
// requests. Call without arguments to query, or pass a bool to update.
func (d *Destination) AcceptsLinks(accept ...bool) bool {
	if len(accept) == 0 {
		return d.acceptLinkRequests
	}
	d.acceptLinkRequests = accept[0]
	return d.acceptLinkRequests
}

func (d *Destination) SetLinkEstablishedCallback(cb func(*Link)) {
	d.callbacks.LinkEstablished = cb
}

func (d *Destination) SetPacketCallback(cb func([]byte, *Packet)) {
	d.callbacks.Packet = cb
}

func (d *Destination) SetProofRequestedCallback(cb func(*Packet) bool) {
	d.callbacks.ProofRequested = cb
}

func (d *Destination) SetProofStrategy(strategy int) error {
	if !containsInt(strategy, []int{DestinationPROVE_NONE, DestinationPROVE_APP, DestinationPROVE_ALL}) {
		panicDestinationState("Unsupported proof strategy")
	}
	d.proofStrategy = strategy
	return nil
}

// Links returns a snapshot of links currently associated with this destination.
func (d *Destination) Links() []*Link {
	d.pruneLinks()
	d.linksMu.Lock()
	defer d.linksMu.Unlock()
	out := make([]*Link, len(d.links))
	copy(out, d.links)
	return out
}

// IncomingLinkRequest mirrors Destination.incoming_link_request() in Python.
// It validates the link request and appends it to the destination if accepted.
func (d *Destination) IncomingLinkRequest(data []byte, packet *Packet) {
	if !d.acceptLinkRequests {
		return
	}
	link := LinkValidateRequest(d, data, packet)
	if link == nil {
		return
	}

	d.linksMu.Lock()
	d.links = append(d.links, link)
	d.linksMu.Unlock()
}

// internal
func (d *Destination) cleanRatchets() {
	if d.ratchets != nil && len(d.ratchets) > d.retainedRatchets {
		if len(d.ratchets) > DestinationRATCHET_COUNT {
			d.ratchets = d.ratchets[:DestinationRATCHET_COUNT]
		}
	}
}

func (d *Destination) persistRatchets() error {
	if d.ratchetsPath == "" {
		return errors.New("no ratchets path set")
	}

	d.ratchetFileLock.Lock()
	defer d.ratchetFileLock.Unlock()

	tempPath := d.ratchetsPath + ".tmp"

	packedRatchets, err := umsgpack.Packb(d.ratchets)
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
		d.ratchets = nil
		d.ratchetsPath = ""
		Log("Could not write ratchet file contents for "+d.String()+". The contained exception was: "+err.Error(), LOG_ERROR)
		return errors.New("could not write ratchet file contents for " + d.String() + ": " + err.Error())
	}

	if _, err := os.Stat(d.ratchetsPath); err == nil {
		_ = os.Remove(d.ratchetsPath)
	}
	if err := os.Rename(tempPath, d.ratchetsPath); err != nil {
		TraceException(err)
		d.ratchets = nil
		d.ratchetsPath = ""
		Log("Could not write ratchet file contents for "+d.String()+". The contained exception was: "+err.Error(), LOG_ERROR)
		return errors.New("could not move ratchet temp file for " + d.String() + ": " + err.Error())
	}

	return nil
}

func (d *Destination) RotateRatchets() (bool, error) {
	if d.ratchets == nil {
		return false, errors.New("cannot rotate ratchet, ratchets are not enabled")
	}
	now := float64(time.Now().Unix())
	if now > d.latestRatchetTime+float64(d.ratchetInterval) {
		Log("Rotating ratchets for "+d.String(), LOG_DEBUG)
		newRatchet, err := IdentityGenerateRatchet()
		if err != nil {
			return false, err
		}
		pub, err := IdentityRatchetPublicBytes(newRatchet)
		if err != nil {
			return false, err
		}
		d.ratchets = append([][]byte{newRatchet}, d.ratchets...)
		d.latestRatchetTime = now
		d.cleanRatchets()
		if err := d.persistRatchets(); err != nil {
			return false, err
		}
		IdentityRememberRatchet(d.hash, pub)
	}
	return true, nil
}

func (d *Destination) Announce(appData []byte, pathResponse bool, attachedInterface *Interface, tag []byte, send bool) *Packet {
	if d.Type != DestinationSINGLE {
		Log("Only SINGLE destination types can be announced", LOG_ERROR)
		return nil
	}
	if d.Direction != DestinationIN {
		Log("Only IN destination types can be announced", LOG_ERROR)
		return nil
	}

	var ratchetPub []byte
	now := float64(time.Now().Unix())

	// purge old path_responses
	for k, v := range d.pathResponses {
		if now > v.Timestamp+DestinationPR_TAG_WINDOW {
			delete(d.pathResponses, k)
		}
	}

	var announceData []byte

	if pathResponse {
		if entry, ok := d.pathResponses[string(tag)]; ok {
			Log("Using cached announce data for answering path request with tag "+PrettyHexRep(tag), LOG_EXTREME)
			announceData = entry.Data
		}
	}

	if announceData == nil {
		destHash := d.hash
		randomHash := announceRandomHash()

		if d.ratchets != nil {
			if ok, err := d.RotateRatchets(); !ok && err != nil {
				Log("Ratchet rotation failed for "+d.String()+": "+err.Error(), LOG_ERROR)
			}
			if len(d.ratchets) > 0 {
				if pub, err := IdentityRatchetPublicBytes(d.ratchets[0]); err == nil {
					ratchetPub = pub
					IdentityRememberRatchet(destHash, ratchetPub)
				} else {
					Log("Could not derive ratchet pub for "+d.String()+": "+err.Error(), LOG_ERROR)
				}
			}
		}

		if appData == nil && d.defaultAppData != nil {
			switch v := d.defaultAppData.(type) {
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
		signed = append(signed, d.identity.GetPublicKey()...)
		signed = append(signed, d.nameHash...)
		signed = append(signed, randomHash...)
		signed = append(signed, ratchetPub...)
		if appData != nil {
			signed = append(signed, appData...)
		}

		signature, err := d.identity.Sign(signed)
		if err != nil {
			Log("Failed to sign announce for "+d.String()+": "+err.Error(), LOG_ERROR)
			return nil
		}

		announceData = append([]byte{}, d.identity.GetPublicKey()...)
		announceData = append(announceData, d.nameHash...)
		announceData = append(announceData, randomHash...)
		announceData = append(announceData, ratchetPub...)
		announceData = append(announceData, signature...)
		if appData != nil {
			announceData = append(announceData, appData...)
		}

		if pathResponse {
			key := string(tag)
			d.pathResponses[key] = &pathResponseEntry{
				Timestamp: float64(time.Now().Unix()),
				Data:      announceData,
			}
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
		WithPacketType(PacketANNOUNCE),
		WithPacketContext(announceContext),
		WithAttachedInterface(attachedInterface),
		WithContextFlag(contextFlag),
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
		return errors.New("invalid path specified")
	}
	if responseGen == nil {
		return errors.New("invalid response generator specified")
	}
	if !containsInt(allow, []int{DestinationALLOW_NONE, DestinationALLOW_ALL, DestinationALLOW_LIST}) {
		return errors.New("invalid request policy")
	}
	pathHash := TruncatedHash([]byte(path))
	var auto interface{} = true
	if len(autoCompress) > 0 {
		auto = autoCompress[0]
	}

	d.requestHandlers[string(pathHash)] = &RequestHandler{
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
	if _, ok := d.requestHandlers[key]; ok {
		delete(d.requestHandlers, key)
		return true
	}
	return false
}

func (d *Destination) DispatchRequest(path string, data any, requestID []byte, linkID []byte, remoteIdentity *Identity, requestedAt time.Time) (any, bool) {
	handler, ok := d.requestHandlers[string(TruncatedHash([]byte(path)))]
	if !ok {
		return nil, false
	}
	if !d.requestAllowed(handler, remoteIdentity) {
		return nil, false
	}
	resp := handler.ResponseGen(path, data, requestID, linkID, remoteIdentity, requestedAt)
	return resp, true
}

func (d *Destination) requestAllowed(handler *RequestHandler, remote *Identity) bool {
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
			if bytesEqual(entry, remote.Hash) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func (d *Destination) Receive(packet *Packet) bool {
	// Ensure the packet is associated with this destination. This is required
	// for destination-side proof generation (Packet.Prove).
	if packet != nil && packet.Destination == nil {
		packet.Destination = d
	}

	if packet.PacketType == PacketLINKREQUEST {
		plaintext := packet.Data
		d.incomingLinkRequest(plaintext, packet)
		return true
	}

	plaintext := d.Decrypt(packet.Data)
	packet.RatchetID = d.latestRatchetID
	if plaintext == nil {
		return false
	}

	if packet.PacketType == PacketDATA {
		if d.callbacks.Packet != nil {
			func() {
				defer func() {
					if r := recover(); r != nil {
						Log("Error while executing receive callback from "+d.String(), LOG_ERROR)
					}
				}()
				d.callbacks.Packet(plaintext, packet)
			}()
		}
		// Python parity: apply proof strategy for incoming data packets.
		d.handleProofStrategy(packet)
	}
	return true
}

func (d *Destination) incomingLinkRequest(data []byte, packet *Packet) {
	if !d.acceptLinkRequests {
		return
	}
	link := LinkValidateRequest(d, data, packet)
	if link != nil {
		d.linksMu.Lock()
		d.links = append(d.links, link)
		d.linksMu.Unlock()
	}
}

func (d *Destination) reloadRatchets(path string) error {
	if _, err := os.Stat(path); err != nil {
		Log("No existing ratchet data found, initialising new ratchet file for "+d.String(), LOG_DEBUG)
		d.ratchets = [][]byte{}
		d.ratchetsPath = path
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
		if !d.identity.Validate(sig, rawRatchets) {
			return errors.New("invalid ratchet file signature")
		}
		var ratchets [][]byte
		if err := umsgpack.Unpackb(rawRatchets, &ratchets); err != nil {
			return err
		}
		d.ratchets = ratchets
		d.ratchetsPath = path
		return nil
	}

	if err := loadAttempt(); err != nil {
		TraceException(err)
		Log("First ratchet reload attempt for "+d.String()+" failed. Possible I/O conflict. Retrying in 500ms.", LOG_ERROR)
		time.Sleep(500 * time.Millisecond)
		if err2 := loadAttempt(); err2 != nil {
			d.ratchets = nil
			d.ratchetsPath = ""
			TraceException(err2)
			Log("The ratchet file located at "+path+" could not be loaded. This could indicate that the ratchet file has become corrupt.", LOG_CRITICAL)
			Log("You can attempt to manually recover the ratchet file, or simply remove it to have Reticulum recreate it on the next use.", LOG_CRITICAL)
			Log("If re-initialize this ratchet file, make sure to send an announce for the relevant destination as soon as possible,", LOG_CRITICAL)
			Log("so that the new ratchet information is synchronized to the network.", LOG_CRITICAL)
			return errors.New("could not read ratchet file contents for " + d.String() + ": " + err2.Error())
		}
		Log("Ratchet reload retry succeeded", LOG_DEBUG)
	}
	return nil
}

func (d *Destination) EnableRatchets(path string) (bool, error) {
	if path == "" {
		return false, errors.New("no ratchet file path specified for " + d.String())
	}
	d.latestRatchetTime = 0
	if err := d.reloadRatchets(path); err != nil {
		return false, err
	}
	Log("Ratchets enabled on "+d.String(), LOG_DEBUG)
	return true, nil
}

func (d *Destination) EnforceRatchets() bool {
	if d.ratchets == nil {
		return false
	}
	d.enforceRatchets = true
	Log("Ratchets enforced on "+d.String(), LOG_DEBUG)
	return true
}

func (d *Destination) SetRetainedRatchets(n int) bool {
	if n <= 0 {
		return false
	}
	d.retainedRatchets = n
	d.cleanRatchets()
	return true
}

func (d *Destination) SetRatchetInterval(interval int) bool {
	if interval <= 0 {
		return false
	}
	d.ratchetInterval = interval
	return true
}

func (d *Destination) CreateKeys() error {
	if d.Type == DestinationPLAIN {
		return errors.New("a plain destination does not hold any keys")
	}
	if d.Type == DestinationSINGLE {
		return errors.New("a single destination holds keys through an Identity instance")
	}
	if d.Type == DestinationGROUP {
		key, err := Cryptography.GenerateKey(32)
		if err != nil {
			return fmt.Errorf("failed to generate group token: %w", err)
		}
		if err := d.LoadPrivateKey(key); err != nil {
			return fmt.Errorf("failed to initialise group token: %w", err)
		}
		d.identity = nil
	}
	return nil
}

func (d *Destination) Encrypt(plaintext []byte) []byte {
	if d.Type == DestinationPLAIN {
		return plaintext
	}
	if d.Type == DestinationSINGLE && d.identity != nil {
		selectedRatchet := IdentityGetRatchet(d.hash)
		if selectedRatchet != nil {
			d.latestRatchetID = IdentityGetRatchetID(selectedRatchet)
		}
		ct, err := d.identity.Encrypt(plaintext, selectedRatchet)
		if err != nil {
			Log("Failed to encrypt payload for "+d.String()+": "+err.Error(), LOG_ERROR)
			return nil
		}
		return ct
	}
	if d.Type == DestinationGROUP {
		if tok, ok := d.getToken(); ok && tok != nil {
			ct, err := tok.Encrypt(plaintext)
			if err != nil {
				Log("The GROUP destination could not encrypt data", LOG_ERROR)
				Log("The contained exception was: "+err.Error(), LOG_ERROR)
				return nil
			}
			return ct
		}
		panicDestinationState("No private key held by GROUP destination. Did you create or load one?")
	}
	return nil
}

func (d *Destination) Decrypt(ciphertext []byte) []byte {
	if d.Type == DestinationPLAIN {
		return ciphertext
	}
	if d.Type == DestinationSINGLE && d.identity != nil {
		d.latestRatchetID = nil
		if d.ratchets != nil {
			var (
				decrypted []byte
				err       error
				ratchetID []byte
			)
			decrypted, ratchetID, err = d.identity.DecryptWithRatchetID(ciphertext, d.ratchets, d.enforceRatchets)
			if err != nil || decrypted == nil {
				Log("Decryption with ratchets failed on "+d.String()+", reloading ratchets from storage and retrying", LOG_ERROR)
				if err2 := d.reloadRatchets(d.ratchetsPath); err2 != nil {
					Log("Decryption still failing after ratchet reload. "+err2.Error(), LOG_ERROR)
					return nil
				}
				decrypted, ratchetID, err = d.identity.DecryptWithRatchetID(ciphertext, d.ratchets, d.enforceRatchets)
				if err != nil || decrypted == nil {
					if err != nil {
						Log("Decryption still failing after ratchet reload. "+err.Error(), LOG_ERROR)
					} else {
						Log("Decryption still failing after ratchet reload.", LOG_ERROR)
					}
					return nil
				}
				Log("Decryption succeeded after ratchet reload", LOG_NOTICE)
			}
			if len(ratchetID) > 0 {
				d.latestRatchetID = ratchetID
			}
			return decrypted
		}
		decrypted, ratchetID, _ := d.identity.DecryptWithRatchetID(ciphertext, nil, d.enforceRatchets)
		if len(ratchetID) > 0 {
			d.latestRatchetID = ratchetID
		}
		return decrypted
	}

	if d.Type == DestinationGROUP {
		if tok, ok := d.getToken(); ok && tok != nil {
			pt, err := tok.Decrypt(ciphertext)
			if err != nil {
				Log("The GROUP destination could not decrypt data", LOG_ERROR)
				Log("The contained exception was: "+err.Error(), LOG_ERROR)
				return nil
			}
			return pt
		}
		panicDestinationState("No private key held by GROUP destination. Did you create or load one?")
	}
	return nil
}

func (d *Destination) Sign(message []byte) []byte {
	if d.Type == DestinationSINGLE && d.identity != nil {
		sig, err := d.identity.Sign(message)
		if err != nil {
			Log("Failed to sign message for "+d.String()+": "+err.Error(), LOG_ERROR)
			return nil
		}
		return sig
	}
	return nil
}

func (d *Destination) SetDefaultAppData(appData interface{}) {
	d.defaultAppData = appData
}

func (d *Destination) ClearDefaultAppData() {
	d.SetDefaultAppData(nil)
}

func (d *Destination) SetGroupKey(key []byte) error {
	return d.LoadPrivateKey(key)
}

func (d *Destination) GroupKeyBytes() []byte {
	d.groupTokenMu.Lock()
	defer d.groupTokenMu.Unlock()
	if len(d.groupTokenBytes) == 0 {
		return nil
	}
	cp := make([]byte, len(d.groupTokenBytes))
	copy(cp, d.groupTokenBytes)
	return cp
}

func (d *Destination) GetPrivateKey() ([]byte, error) {
	switch d.Type {
	case DestinationPLAIN:
		return nil, errors.New("plain destination does not hold a private key")
	case DestinationSINGLE:
		return nil, errors.New("single destination holds keys through Identity")
	case DestinationGROUP:
		d.groupTokenMu.Lock()
		defer d.groupTokenMu.Unlock()
		if len(d.groupTokenBytes) == 0 {
			return nil, errors.New("group destination has no private key")
		}
		cp := make([]byte, len(d.groupTokenBytes))
		copy(cp, d.groupTokenBytes)
		return cp, nil
	default:
		return nil, errors.New("unknown destination type")
	}
}

func (d *Destination) LoadPrivateKey(key []byte) error {
	if d.Type == DestinationPLAIN {
		return errors.New("plain destination does not hold a private key")
	}
	if d.Type == DestinationSINGLE {
		return errors.New("single destination holds keys through Identity")
	}
	if d.Type != DestinationGROUP {
		return errors.New("unsupported destination type for private key load")
	}
	if len(key) == 0 {
		return errors.New("group key cannot be empty")
	}
	tok, err := Cryptography.NewToken(key)
	if err != nil {
		return err
	}
	d.groupTokenMu.Lock()
	defer d.groupTokenMu.Unlock()
	d.groupTokenBytes = append([]byte{}, key...)
	d.groupToken = tok
	return nil
}

func (d *Destination) LoadPublicKey(_ []byte) error {
	return errors.New("destination public keys are managed by Identity")
}

func (d *Destination) pruneLinks() {
	d.linksMu.Lock()
	defer d.linksMu.Unlock()
	if len(d.links) == 0 {
		return
	}
	filtered := d.links[:0]
	for _, l := range d.links {
		if l == nil {
			continue
		}
		if l.Status == LinkClosed {
			continue
		}
		filtered = append(filtered, l)
	}
	if len(filtered) == len(d.links) {
		return
	}
	d.links = append([]*Link(nil), filtered...)
}

func (d *Destination) removeLink(target *Link) {
	if target == nil {
		return
	}
	d.linksMu.Lock()
	defer d.linksMu.Unlock()
	for i, l := range d.links {
		if l == target {
			d.links = append(d.links[:i], d.links[i+1:]...)
			break
		}
	}
}

// ---- small utilities ----

func containsDot(s string) bool {
	return strings.Contains(s, ".")
}

func containsInt(v int, arr []int) bool {
	for _, x := range arr {
		if x == v {
			return true
		}
	}
	return false
}

// Simple alternative to full_name.split(".").
func splitDot(s string) []string {
	return strings.Split(s, ".")
}

// Stub for GROUP Token.
func (d *Destination) getToken() (*Cryptography.Token, bool) {
	d.groupTokenMu.Lock()
	defer d.groupTokenMu.Unlock()

	if d.groupToken != nil {
		return d.groupToken, true
	}
	if len(d.groupTokenBytes) == 0 {
		return nil, false
	}
	token, err := Cryptography.NewToken(d.groupTokenBytes)
	if err != nil {
		Log("Failed to initialise group token for "+d.String()+": "+err.Error(), LOG_ERROR)
		return nil, false
	}
	d.groupToken = token
	return token, true
}

func announceRandomHash() []byte {
	const segmentLen = 5
	randomBytes := IdentityGetRandomHash()
	randPart := make([]byte, segmentLen)
	copy(randPart, randomBytes[:segmentLen])
	var tsBuf [8]byte
	binary.BigEndian.PutUint64(tsBuf[:], uint64(time.Now().Unix()))
	return append(randPart, tsBuf[len(tsBuf)-segmentLen:]...)
}

func (d *Destination) handleProofStrategy(packet *Packet) {
	if packet == nil {
		return
	}
	switch d.proofStrategy {
	case DestinationPROVE_ALL:
		// Python parity: proofs are sent to the packet's ProofDestination (hash=packet hash),
		// not back to the destination itself.
		packet.Prove(nil)
	case DestinationPROVE_APP:
		if cb := d.callbacks.ProofRequested; cb != nil {
			prove := false
			func() {
				defer func() {
					if r := recover(); r != nil {
						Log("Error while executing proof request callback", LOG_ERROR)
					}
				}()
				prove = cb(packet)
			}()
			if prove {
				packet.Prove(nil)
			}
		}
	}
}

func (d *Destination) Hash() []byte {
	if d == nil || d.hash == nil {
		return nil
	}
	cp := make([]byte, len(d.hash))
	copy(cp, d.hash)
	return cp
}

func (d *Destination) Name() string {
	if d == nil {
		return ""
	}
	return d.name
}

func (d *Destination) NameHash() []byte {
	if d == nil || d.nameHash == nil {
		return nil
	}
	cp := make([]byte, len(d.nameHash))
	copy(cp, d.nameHash)
	return cp
}

func (d *Destination) HexHash() string {
	if d == nil {
		return ""
	}
	return d.hexhash
}

func (d *Destination) Identity() *Identity {
	if d == nil {
		return nil
	}
	return d.identity
}

func (d *Destination) ProofStrategy() int {
	if d == nil {
		return DestinationPROVE_NONE
	}
	return d.proofStrategy
}
