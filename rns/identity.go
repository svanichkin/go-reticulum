package rns

import (
	"bytes"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	Cryptography "github.com/svanichkin/go-reticulum/rns/cryptography"
	umsgpack "github.com/svanichkin/go-reticulum/rns/vendor"
)

// Constants for X25519/Ed25519.
const (
	x25519KeyLen          = 32 // length of X25519 private/public keys
	truncatedHashBytes    = 16 // 128 bits, like LinkID
	derivedKeyLen         = 64 // 64 bytes for Token (32 signing + 32 enc)
	derivedKeyLenLegacy   = 32 // legacy Token length (16 signing + 16 enc)
	ratchetExpiry         = 30 * 24 * time.Hour
	identityPubKeyLen     = x25519KeyLen + ed25519.PublicKeySize
	announceRandomHashLen = 10
	ed25519SeedLen        = ed25519.SeedSize // Python Identity stores 32-byte seed
)

// Identity mirrors RNS.Identity.
type Identity struct {
	curve ecdh.Curve

	// encryption (X25519)
	prv      *ecdh.PrivateKey
	prvBytes []byte
	pub      *ecdh.PublicKey
	pubBytes []byte

	// signature (Ed25519)
	sigPriv      ed25519.PrivateKey
	sigPrivBytes []byte
	sigPub       ed25519.PublicKey
	sigPubBytes  []byte

	AppData []byte

	// hashes
	Hash    []byte
	HexHash string
}

var (
	knownRatchets = struct {
		sync.RWMutex
		store map[string][]byte
	}{
		store: make(map[string][]byte),
	}
	ratchetPersistLock sync.Mutex

	knownDestinations = struct {
		sync.RWMutex
		entries map[string]*knownDestinationEntry
	}{
		entries: make(map[string]*knownDestinationEntry),
	}
	knownDestinationsSaveMu        sync.Mutex
	knownDestinationsLoadMu        sync.Mutex
	knownDestinationsLoaded        atomic.Bool
	knownDestinationsLoadAttempted atomic.Bool
)

type ratchetRecord struct {
	Ratchet  []byte  `msgpack:"ratchet"`
	Received float64 `msgpack:"received"`
}

type knownDestinationEntry struct {
	SeenAt     float64
	PacketHash []byte
	PublicKey  []byte
	AppData    []byte
}

// NewIdentity creates a new identity with fresh keys.
func NewIdentity() (*Identity, error) {
	id := &Identity{
		curve: ecdh.X25519(),
	}
	if err := id.CreateKeys(); err != nil {
		return nil, err
	}
	return id, nil
}

// IdentityFromBytes loads from private bytes (like from_bytes).
func IdentityFromBytes(prvBytes []byte) (*Identity, error) {
	id := &Identity{
		curve: ecdh.X25519(),
	}
	if err := id.LoadPrivateKey(prvBytes); err != nil {
		return nil, err
	}
	return id, nil
}

// IdentityFromFile loads from file (like from_file).
func IdentityFromFile(path string) (*Identity, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return IdentityFromBytes(b)
}

// Save mirrors to_file(path).
func (id *Identity) Save(path string) error {
	if id.prvBytes == nil || len(id.prvBytes) != x25519KeyLen {
		return errors.New("identity has no private key material")
	}

	seed := id.sigPrivBytes
	if len(seed) == 0 && len(id.sigPriv) > 0 {
		seed = id.sigPriv.Seed()
	}
	if len(seed) != ed25519SeedLen {
		return errors.New("identity has no private key material")
	}

	all := append([]byte{}, id.prvBytes...)
	all = append(all, seed...)

	return os.WriteFile(path, all, 0o600)
}

// CreateKeys generates X25519 + Ed25519 keypairs.
func (id *Identity) CreateKeys() error {
	if id.curve == nil {
		id.curve = ecdh.X25519()
	}

	// X25519
	prv, err := id.curve.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	id.prv = prv
	id.prvBytes = prv.Bytes()
	pub := prv.PublicKey()
	id.pub = pub
	id.pubBytes = pub.Bytes()

	// Ed25519
	pubSig, prvSig, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	id.sigPriv = prvSig
	id.sigPrivBytes = prvSig.Seed()
	id.sigPub = pubSig
	id.sigPubBytes = pubSig

	id.updateHashes()
	return nil
}

// GetPrivateKey mirrors get_private_key().
func (id *Identity) GetPrivateKey() []byte {
	if id.prvBytes == nil || len(id.prvBytes) != x25519KeyLen {
		return nil
	}

	seed := id.sigPrivBytes
	if len(seed) == 0 && len(id.sigPriv) > 0 {
		seed = id.sigPriv.Seed()
	}
	if len(seed) != ed25519SeedLen {
		return nil
	}
	out := append([]byte{}, id.prvBytes...)
	out = append(out, seed...)
	return out
}

// GetPublicKey mirrors get_public_key().
func (id *Identity) GetPublicKey() []byte {
	if id.pubBytes == nil || id.sigPubBytes == nil {
		return nil
	}
	out := append([]byte{}, id.pubBytes...)
	out = append(out, id.sigPubBytes...)
	return out
}

// LoadPrivateKey mirrors load_private_key().
func (id *Identity) LoadPrivateKey(all []byte) error {
	if len(all) < x25519KeyLen+ed25519SeedLen {
		return errors.New("invalid private key length")
	}
	id.prvBytes = all[:x25519KeyLen]
	suffix := all[x25519KeyLen:]

	var err error
	if id.curve == nil {
		id.curve = ecdh.X25519()
	}
	id.prv, err = id.curve.NewPrivateKey(id.prvBytes)
	if err != nil {
		return err
	}

	// Python stores 32-byte Ed25519 seed. For backwards compatibility with older
	// Go versions of this port, also accept a full 64-byte Ed25519 private key.
	switch len(suffix) {
	case ed25519SeedLen:
		id.sigPrivBytes = append([]byte{}, suffix...)
		id.sigPriv = ed25519.NewKeyFromSeed(id.sigPrivBytes)
	case ed25519.PrivateKeySize:
		id.sigPriv = ed25519.PrivateKey(append([]byte{}, suffix...))
		id.sigPrivBytes = id.sigPriv.Seed()
	default:
		return errors.New("invalid private key length")
	}

	// rebuild public keys
	pub := id.prv.PublicKey()
	id.pub = pub
	id.pubBytes = pub.Bytes()
	id.sigPub = id.sigPriv.Public().(ed25519.PublicKey)
	id.sigPubBytes = id.sigPub

	id.updateHashes()
	return nil
}

// LoadPublicKey mirrors load_public_key().
func (id *Identity) LoadPublicKey(pubBytes []byte) error {
	if len(pubBytes) != identityPubKeyLen {
		return errors.New("invalid public key length")
	}
	if id.curve == nil {
		id.curve = ecdh.X25519()
	}

	id.pubBytes = append([]byte{}, pubBytes[:x25519KeyLen]...)
	id.sigPubBytes = append([]byte{}, pubBytes[x25519KeyLen:x25519KeyLen+ed25519.PublicKeySize]...)

	var err error
	id.pub, err = id.curve.NewPublicKey(id.pubBytes)
	if err != nil {
		return err
	}
	id.sigPub = ed25519.PublicKey(id.sigPubBytes)

	id.updateHashes()
	return nil
}

// updateHashes mirrors update_hashes().
func (id *Identity) updateHashes() {
	pubAll := id.GetPublicKey()
	h := TruncatedHash(pubAll)
	id.Hash = h
	id.HexHash = hex.EncodeToString(h)
}

// FullHash — SHA-256
func FullHash(data []byte) []byte {
	sum := sha256.Sum256(data)
	return sum[:]
}

// TruncatedHash is a truncated SHA-256.
func TruncatedHash(data []byte) []byte {
	h := FullHash(data)
	return h[:truncatedHashBytes]
}

func copyBytes(b []byte) []byte {
	if len(b) == 0 {
		return nil
	}
	out := make([]byte, len(b))
	copy(out, b)
	return out
}

func nowSeconds() float64 {
	return float64(time.Now().UnixNano()) / 1e9
}

func identityStorageBasePath() string {
	if inst := GetInstance(); inst != nil && inst.StoragePath != "" {
		return inst.StoragePath
	}
	return filepath.Join(osUserDir(), ".reticulum", "storage")
}

func knownDestinationsPath() string {
	return filepath.Join(identityStorageBasePath(), "known_destinations")
}

func ensureKnownDestinationsLoaded() {
	if knownDestinationsLoaded.Load() || knownDestinationsLoadAttempted.Load() {
		return
	}

	knownDestinationsLoadMu.Lock()
	defer knownDestinationsLoadMu.Unlock()

	if knownDestinationsLoaded.Load() || knownDestinationsLoadAttempted.Load() {
		return
	}

	knownDestinationsLoadAttempted.Store(true)
	if err := IdentityLoadKnownDestinations(); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			Logf(LogWarning, "Could not load known destinations: %v", err)
		}
	}
}

// IdentityGetRandomHash returns a random truncated hash.
func IdentityGetRandomHash() []byte {
	buf := make([]byte, truncatedHashBytes)
	if _, err := rand.Read(buf); err != nil {
		// fallback: deterministic hash of timestamp
		ts := time.Now().UnixNano()
		var tmp [8]byte
		binary.BigEndian.PutUint64(tmp[:], uint64(ts))
		copy(buf, tmp[:])
	}
	return TruncatedHash(buf)
}

// GetSalt mirrors get_salt().
func (id *Identity) GetSalt() []byte {
	return id.Hash
}

// GetContext mirrors get_context().
func (id *Identity) GetContext() []byte {
	return nil
}

// ---- Known destination storage ----

// IdentityRemember persists a public key and app_data for a destination.
func IdentityRemember(packetHash, destinationHash, publicKey, appData []byte) error {
	ensureKnownDestinationsLoaded()

	if len(publicKey) != identityPubKeyLen {
		return fmt.Errorf("can't remember %s, public key length %d is invalid", PrettyHexRep(destinationHash), len(publicKey))
	}

	entry := &knownDestinationEntry{
		SeenAt:     nowSeconds(),
		PacketHash: copyBytes(packetHash),
		PublicKey:  copyBytes(publicKey),
		AppData:    copyBytes(appData),
	}

	key := string(destinationHash)
	knownDestinations.Lock()
	knownDestinations.entries[key] = entry
	knownDestinations.Unlock()
	return nil
}

// IdentityRecall looks up an identity by destination hash or identity hash.
func IdentityRecall(targetHash []byte, fromIdentityHash ...bool) *Identity {
	ensureKnownDestinationsLoaded()

	if len(targetHash) == 0 {
		return nil
	}
	searchIdentityHash := len(fromIdentityHash) > 0 && fromIdentityHash[0]

	if searchIdentityHash {
		knownDestinations.RLock()
		for _, entry := range knownDestinations.entries {
			if entry == nil || len(entry.PublicKey) == 0 {
				continue
			}
			if bytes.Equal(targetHash, TruncatedHash(entry.PublicKey)) {
				id := identityFromKnownEntry(entry)
				knownDestinations.RUnlock()
				return id
			}
		}
		knownDestinations.RUnlock()
		return nil
	}

	knownDestinations.RLock()
	entry := knownDestinations.entries[string(targetHash)]
	knownDestinations.RUnlock()
	if entry != nil {
		return identityFromKnownEntry(entry)
	}

	for _, dst := range Destinations {
		if dst == nil || dst.identity == nil || len(dst.Hash) == 0 {
			continue
		}
		if bytes.Equal(targetHash, dst.Hash) {
			pub := dst.identity.GetPublicKey()
			if len(pub) == 0 {
				return nil
			}
			id := &Identity{curve: ecdh.X25519()}
			if err := id.LoadPublicKey(pub); err != nil {
				return nil
			}
			return id
		}
	}

	return nil
}

// IdentityRecallAppData returns the last app_data.
func IdentityRecallAppData(destinationHash []byte) []byte {
	ensureKnownDestinationsLoaded()

	knownDestinations.RLock()
	defer knownDestinations.RUnlock()
	if entry, ok := knownDestinations.entries[string(destinationHash)]; ok {
		return copyBytes(entry.AppData)
	}
	return nil
}

// IdentitySaveKnownDestinations writes the map to disk.
func IdentitySaveKnownDestinations() error {
	ensureKnownDestinationsLoaded()

	knownDestinationsSaveMu.Lock()
	defer knownDestinationsSaveMu.Unlock()

	path := knownDestinationsPath()
	if err := ensureParentDir(path); err != nil {
		return err
	}

	// Merge on-disk data so we don't lose entries from other processes.
	diskEntries, err := readKnownDestinationsFromDisk(path)
	if err == nil {
		knownDestinations.Lock()
		for k, v := range diskEntries {
			if _, ok := knownDestinations.entries[k]; !ok {
				knownDestinations.entries[k] = v
			}
		}
		knownDestinations.Unlock()
	} else if !errors.Is(err, os.ErrNotExist) {
		Logf(LogWarning, "Could not merge known destinations from disk: %v", err)
	}

	knownDestinations.RLock()
	snapshot := make(map[string]*knownDestinationEntry, len(knownDestinations.entries))
	for k, v := range knownDestinations.entries {
		if v == nil {
			continue
		}
		snapshot[k] = &knownDestinationEntry{
			SeenAt:     v.SeenAt,
			PacketHash: copyBytes(v.PacketHash),
			PublicKey:  copyBytes(v.PublicKey),
			AppData:    copyBytes(v.AppData),
		}
	}
	knownDestinations.RUnlock()

	data, err := encodeKnownDestinations(snapshot)
	if err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}

	Logf(LogDebug, "Saved %d known destinations to storage", len(snapshot))
	return nil
}

// IdentityLoadKnownDestinations reads the map from storage/known_destinations.
func IdentityLoadKnownDestinations() error {
	path := knownDestinationsPath()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		Logf(LogVerbose, "Destinations file does not exist, no known destinations loaded")
		knownDestinationsLoaded.Store(true)
		return nil
	}
	if err != nil {
		return err
	}

	entries, err := decodeKnownDestinations(data)
	if err != nil {
		return err
	}

	knownDestinations.Lock()
	knownDestinations.entries = entries
	knownDestinations.Unlock()

	Logf(LogVerbose, "Loaded %d known destinations from storage", len(entries))
	knownDestinationsLoaded.Store(true)
	return nil
}

// IdentityPersistData persists the map if we are a standalone instance.
func IdentityPersistData() {
	if inst := GetInstance(); inst != nil && inst.IsConnectedToSharedInstance {
		return
	}
	if err := IdentitySaveKnownDestinations(); err != nil {
		Logf(LogError, "Error while saving known destinations to disk: %v", err)
	}
}

// IdentityExitHandler runs during Reticulum shutdown.
func IdentityExitHandler() {
	IdentityPersistData()
}

func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0o755)
}

func readKnownDestinationsFromDisk(path string) (map[string]*knownDestinationEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return decodeKnownDestinations(data)
}

func encodeKnownDestinations(entries map[string]*knownDestinationEntry) ([]byte, error) {
	// Persist with fixed-size byte-array keys so msgpack encodes them as binary,
	// matching Python's bytes-keyed storage format.
	payload := make(map[[truncatedHashBytes]byte][]any, len(entries))
	for key, entry := range entries {
		if len(key) != truncatedHashBytes {
			continue
		}
		var keyBytes [truncatedHashBytes]byte
		copy(keyBytes[:], []byte(key))
		payload[keyBytes] = []any{
			entry.SeenAt,
			entry.PacketHash,
			entry.PublicKey,
			entry.AppData,
		}
	}
	return umsgpack.Packb(payload)
}

func decodeKnownDestinations(data []byte) (map[string]*knownDestinationEntry, error) {
	// Python compatibility: keys can be bytes; older Go versions might have written string keys.
	var raw map[any]any
	if err := umsgpack.Unpackb(data, &raw); err != nil {
		return nil, err
	}

	entries := make(map[string]*knownDestinationEntry, len(raw))
	for k, v := range raw {
		var keyBytes []byte
		switch kt := k.(type) {
		case []byte:
			keyBytes = kt
		case umsgpack.BinaryKey:
			keyBytes = []byte(string(kt))
		case string:
			keyBytes = []byte(kt)
		default:
			continue
		}
		if len(keyBytes) != ReticulumTruncatedHashLength/8 {
			continue
		}
		key := string(keyBytes)

		var values []any
		switch vt := v.(type) {
		case []any:
			values = vt
		default:
			continue
		}

		entry := &knownDestinationEntry{}
		if len(values) > 0 {
			entry.SeenAt = asFloat64(values[0])
		}
		if len(values) > 1 {
			entry.PacketHash = asBytes(values[1])
		}
		if len(values) > 2 {
			entry.PublicKey = asBytes(values[2])
		}
		if len(values) > 3 {
			entry.AppData = asBytes(values[3])
		}
		if len(entry.PublicKey) != identityPubKeyLen {
			continue
		}
		entries[key] = entry
	}
	return entries, nil
}

func asBytes(v interface{}) []byte {
	switch val := v.(type) {
	case nil:
		return nil
	case []byte:
		return copyBytes(val)
	case string:
		return []byte(val)
	default:
		return nil
	}
}

func identityFromKnownEntry(entry *knownDestinationEntry) *Identity {
	if entry == nil || len(entry.PublicKey) != identityPubKeyLen {
		return nil
	}
	id := &Identity{curve: ecdh.X25519()}
	if err := id.LoadPublicKey(entry.PublicKey); err != nil {
		return nil
	}
	id.AppData = copyBytes(entry.AppData)
	return id
}

func logAnnounceReception(packet *Packet) {
	if packet == nil {
		return
	}
	dest := PrettyHexRep(packet.DestinationHash)
	signal := packetSignalString(packet)
	if len(packet.TransportID) > 0 {
		Logf(LogExtreme, "Valid announce for %s %d hops away, received via %s on %v%s",
			dest, packet.Hops, PrettyHexRep(packet.TransportID), packet.ReceivingInterface, signal)
	} else {
		Logf(LogExtreme, "Valid announce for %s %d hops away, received on %v%s",
			dest, packet.Hops, packet.ReceivingInterface, signal)
	}
}

func packetSignalString(packet *Packet) string {
	if packet == nil {
		return ""
	}
	parts := []string{}
	if packet.RSSI != nil {
		parts = append(parts, fmt.Sprintf("RSSI %.0fdBm", *packet.RSSI))
	}
	if packet.SNR != nil {
		parts = append(parts, fmt.Sprintf("SNR %.0fdB", *packet.SNR))
	}
	if len(parts) == 0 {
		return ""
	}
	return " [" + strings.Join(parts, ", ") + "]"
}

// IdentityCurrentRatchetID returns the ID of the current ratchet if known.
func IdentityCurrentRatchetID(destinationHash []byte) []byte {
	ratchet := IdentityGetRatchet(destinationHash)
	if ratchet == nil {
		return nil
	}
	return IdentityGetRatchetID(ratchet)
}

// ValidateAnnounce mirrors Python's validate_announce().
func ValidateAnnounce(packet *Packet, onlyValidateSignature bool) bool {
	if packet == nil || packet.PacketType != PacketTypeAnnounce {
		return false
	}
	ensureKnownDestinationsLoaded()

	data := packet.Data
	keySize := identityPubKeyLen
	nameHashLen := IdentityNameHashLength / 8
	sigLen := ed25519.SignatureSize
	ratchetLen := x25519KeyLen
	if len(data) < keySize+nameHashLen+announceRandomHashLen+sigLen {
		return false
	}

	publicKey := data[:keySize]
	offset := keySize
	nameHash := data[offset : offset+nameHashLen]
	offset += nameHashLen
	randomHash := data[offset : offset+announceRandomHashLen]
	offset += announceRandomHashLen

	var ratchet []byte
	if packet.ContextFlag == FlagSet {
		if len(data) < offset+ratchetLen+sigLen {
			return false
		}
		ratchet = data[offset : offset+ratchetLen]
		offset += ratchetLen
	}

	if len(data) < offset+sigLen {
		return false
	}

	signature := data[offset : offset+sigLen]
	offset += sigLen

	var appData []byte
	if len(data) > offset {
		appData = append([]byte(nil), data[offset:]...)
	}
	if len(data) <= keySize+nameHashLen+announceRandomHashLen+sigLen {
		appData = nil
	}

	signed := make([]byte, 0, len(packet.DestinationHash)+len(publicKey)+len(nameHash)+len(randomHash)+len(ratchet)+len(appData))
	signed = append(signed, packet.DestinationHash...)
	signed = append(signed, publicKey...)
	signed = append(signed, nameHash...)
	signed = append(signed, randomHash...)
	signed = append(signed, ratchet...)
	if len(appData) > 0 {
		signed = append(signed, appData...)
	}

	announced := &Identity{curve: ecdh.X25519()}
	if err := announced.LoadPublicKey(publicKey); err != nil {
		Logf(LogDebug, "Received invalid announce for %s: %v", PrettyHexRep(packet.DestinationHash), err)
		return false
	}
	if key, ok := func(hash []byte) (hashKey, bool) {
		if len(hash) < truncatedHashBytes {
			return hashKey{}, false
		}
		var key hashKey
		copy(key[:], hash[:truncatedHashBytes])
		return key, true
	}(announced.Hash); ok {
		blackholeMu.RLock()
		_, blackholed := blackholedIdentities[key]
		blackholeMu.RUnlock()
		if blackholed {
			Logf(LogExtreme, "Invalidated and dropped announce from blackholed identity %s", PrettyHexRep(announced.Hash))
			return false
		}
	}

	if !announced.Validate(signature, signed) {
		Logf(LogDebug, "Received invalid announce for %s: signature mismatch", PrettyHexRep(packet.DestinationHash))
		return false
	}

	if onlyValidateSignature {
		return true
	}

	hashMaterial := append([]byte{}, nameHash...)
	hashMaterial = append(hashMaterial, announced.Hash...)
	expectedHash := FullHash(hashMaterial)[:ReticulumTruncatedHashLength/8]
	if len(packet.DestinationHash) != ReticulumTruncatedHashLength/8 || !bytes.Equal(packet.DestinationHash, expectedHash) {
		Logf(LogDebug, "Received invalid announce for %s: destination mismatch", PrettyHexRep(packet.DestinationHash))
		return false
	}

	knownDestinations.RLock()
	if entry, ok := knownDestinations.entries[string(packet.DestinationHash)]; ok && entry != nil && len(entry.PublicKey) > 0 && !bytes.Equal(entry.PublicKey, publicKey) {
		knownDestinations.RUnlock()
		Log("Received announce with mismatched public key; rejecting", LogCritical)
		return false
	}
	knownDestinations.RUnlock()

	packetHash := packet.PacketHash
	if len(packetHash) == 0 {
		packetHash = packet.GetHash()
	}
	if err := IdentityRemember(packetHash, packet.DestinationHash, publicKey, appData); err != nil {
		Logf(LogWarning, "Could not remember announce for %s: %v", PrettyHexRep(packet.DestinationHash), err)
	}

	if len(ratchet) == ratchetLen {
		IdentityRememberRatchet(packet.DestinationHash, ratchet)
	}

	logAnnounceReception(packet)
	return true
}

// deriveKey — HKDF(shared, salt, info)
func deriveKey(shared, salt, info []byte, length int) ([]byte, error) {
	return Cryptography.HKDF(length, shared, salt, info)
}

// Encrypt mirrors encrypt(self, plaintext, ratchet=None).
// Returns ephemeral_pub || ciphertext.
func (id *Identity) Encrypt(plaintext []byte, ratchet []byte) ([]byte, error) {
	if id.pub == nil {
		return nil, errors.New("identity has no public key")
	}
	if id.curve == nil {
		id.curve = ecdh.X25519()
	}

	// ephemeral key
	eph, err := id.curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	ephPubBytes := eph.PublicKey().Bytes()

	var targetPub *ecdh.PublicKey
	if ratchet != nil {
		targetPub, err = id.curve.NewPublicKey(ratchet)
		if err != nil {
			return nil, err
		}
	} else {
		targetPub = id.pub
	}

	shared, err := eph.ECDH(targetPub)
	if err != nil {
		return nil, err
	}

	derived, err := deriveKey(shared, id.GetSalt(), id.GetContext(), derivedKeyLen)
	if err != nil {
		return nil, err
	}

	tok, err := Cryptography.NewToken(derived) // AES Token from crypto package
	if err != nil {
		return nil, err
	}
	ct, err := tok.Encrypt(plaintext)
	if err != nil {
		return nil, err
	}

	return append(ephPubBytes, ct...), nil
}

// decryptWithShared mirrors __decrypt().
func (id *Identity) decryptWithShared(shared, ciphertext []byte) ([]byte, error) {
	derived, err := deriveKey(shared, id.GetSalt(), id.GetContext(), derivedKeyLen)
	if err != nil {
		return nil, err
	}
	tok, err := Cryptography.NewToken(derived)
	if err != nil {
		return nil, err
	}
	plaintext, err := tok.Decrypt(ciphertext)
	if err == nil {
		return plaintext, nil
	}

	// Python has DERIVED_KEY_LENGTH_LEGACY for backwards compatibility.
	derivedLegacy, derr := deriveKey(shared, id.GetSalt(), id.GetContext(), derivedKeyLenLegacy)
	if derr != nil {
		return nil, err
	}
	tokLegacy, terr := Cryptography.NewToken(derivedLegacy)
	if terr != nil {
		return nil, err
	}
	if ptLegacy, lerr := tokLegacy.Decrypt(ciphertext); lerr == nil {
		return ptLegacy, nil
	}
	return nil, err
}

// Decrypt mirrors decrypt(self, ciphertext_token, ratchets=None, enforce_ratchets=False).
func (id *Identity) Decrypt(ciphertextToken []byte, ratchets [][]byte, enforceRatchets bool) ([]byte, error) {
	plaintext, _, err := id.decryptWithRatchetID(ciphertextToken, ratchets, enforceRatchets)
	return plaintext, err
}

func (id *Identity) DecryptWithRatchetID(ciphertextToken []byte, ratchets [][]byte, enforceRatchets bool) ([]byte, []byte, error) {
	return id.decryptWithRatchetID(ciphertextToken, ratchets, enforceRatchets)
}

func (id *Identity) DecryptWithRatchetReceiver(ciphertextToken []byte, ratchets [][]byte, enforceRatchets bool, setLatestRatchetID func([]byte)) ([]byte, error) {
	plaintext, ratchetID, err := id.decryptWithRatchetID(ciphertextToken, ratchets, enforceRatchets)
	if setLatestRatchetID != nil {
		setLatestRatchetID(ratchetID)
	}
	return plaintext, err
}

func (id *Identity) decryptWithRatchetID(ciphertextToken []byte, ratchets [][]byte, enforceRatchets bool) ([]byte, []byte, error) {
	if id.prv == nil {
		return nil, nil, errors.New("identity has no private key")
	}
	if len(ciphertextToken) <= x25519KeyLen {
		Log("Decryption failed because the token size was invalid.", LogDebug)
		return nil, nil, nil
	}
	if id.curve == nil {
		id.curve = ecdh.X25519()
	}

	peerPubBytes := ciphertextToken[:x25519KeyLen]
	ciphertext := ciphertextToken[x25519KeyLen:]

	peerPub, err := id.curve.NewPublicKey(peerPubBytes)
	if err != nil {
		Logf(LogDebug, "Decryption by %s failed: %v", PrettyHexRep(id.Hash), err)
		return nil, nil, nil
	}

	for _, ratchet := range ratchets {
		if ratchet == nil {
			continue
		}
		ratchetPrv, err := id.curve.NewPrivateKey(ratchet)
		if err != nil {
			continue
		}
		shared, err := ratchetPrv.ECDH(peerPub)
		if err != nil {
			continue
		}
		pt, err := id.decryptWithShared(shared, ciphertext)
		if err == nil {
			ratchetPub := ratchetPrv.PublicKey().Bytes()
			ratchetID := IdentityGetRatchetID(ratchetPub)
			return pt, ratchetID, nil
		}
	}

	if enforceRatchets {
		Logf(LogDebug, "Decryption with ratchet enforcement by %s failed. Dropping packet.", PrettyHexRep(id.Hash))
		return nil, nil, nil
	}

	shared, err := id.prv.ECDH(peerPub)
	if err != nil {
		Logf(LogDebug, "Decryption by %s failed: %v", PrettyHexRep(id.Hash), err)
		return nil, nil, nil
	}
	pt, err := id.decryptWithShared(shared, ciphertext)
	if err != nil {
		Logf(LogDebug, "Decryption by %s failed: %v", PrettyHexRep(id.Hash), err)
		return nil, nil, nil
	}
	return pt, nil, nil
}

// Sign mirrors sign().
func (id *Identity) Sign(msg []byte) ([]byte, error) {
	if id.sigPriv == nil {
		return nil, errors.New("identity has no signing key")
	}
	return ed25519.Sign(id.sigPriv, msg), nil
}

// Validate mirrors validate().
func (id *Identity) Validate(sig, msg []byte) bool {
	if id.sigPub == nil {
		return false
	}
	return ed25519.Verify(id.sigPub, msg, sig)
}

// String mirrors __str__().
func (id *Identity) String() string {
	if id.HexHash == "" {
		return "<identity>"
	}
	return id.HexHash
}

// Prove sends a proof packet to confirm delivery.
func (id *Identity) Prove(packet *Packet, destination *Destination) {
	if id == nil || packet == nil {
		return
	}
	if id.sigPriv == nil {
		Log("Identity cannot send proof without a signing key", LogError)
		return
	}

	packetHash := packet.PacketHash
	if len(packetHash) == 0 {
		packetHash = packet.GetHash()
	}

	signature, err := id.Sign(packetHash)
	if err != nil {
		Logf(LogError, "Could not sign proof for %s: %v", PrettyHexRep(packetHash), err)
		return
	}

	var proofData []byte
	if ShouldUseImplicitProof() {
		proofData = signature
	} else {
		proofData = append(append([]byte{}, packetHash...), signature...)
	}

	if destination == nil {
		destination = packet.GenerateProofDestination()
	}
	if destination == nil {
		Log("Could not determine proof destination", LogError)
		return
	}

	proof := NewPacket(
		destination,
		proofData,
		WithPacketType(PacketTypeProof),
		WithPacketContext(PacketCtxNone),
		WithTransportType(Broadcast),
		WithAttachedInterface(packet.ReceivingInterface),
		WithCreateReceipt(false),
	)
	proof.FromPacked = true
	proof.Destination = destination
	if proof.Send() == nil {
		Logf(LogDebug, "Sent proof for %s", PrettyHexRep(packetHash))
	}
}

// ---- Ratchet helpers ----

func IdentityGenerateRatchet() ([]byte, error) {
	curve := ecdh.X25519()
	priv, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return priv.Bytes(), nil
}

func IdentityRatchetPublicBytes(private []byte) ([]byte, error) {
	if len(private) != x25519KeyLen {
		return nil, errors.New("invalid ratchet length")
	}
	curve := ecdh.X25519()
	key, err := curve.NewPrivateKey(private)
	if err != nil {
		return nil, err
	}
	return key.PublicKey().Bytes(), nil
}

func IdentityRememberRatchet(destHash, ratchet []byte) {
	if len(destHash) == 0 || len(ratchet) == 0 {
		return
	}

	var needsPersist bool
	knownRatchets.Lock()
	key := string(destHash)
	if existing, ok := knownRatchets.store[key]; !ok || !bytes.Equal(existing, ratchet) {
		knownRatchets.store[key] = append([]byte{}, ratchet...)
		needsPersist = true
	}
	knownRatchets.Unlock()

	if !needsPersist {
		return
	}

	ratchetID := IdentityGetRatchetID(ratchet)
	Logf(LogExtreme, "Remembering ratchet %s for %s", PrettyHexRep(ratchetID), PrettyHexRep(destHash))

	if inst := GetInstance(); inst != nil && inst.IsConnectedToSharedInstance {
		return
	}

	destCopy := append([]byte{}, destHash...)
	ratchetCopy := append([]byte{}, ratchet...)
	go func() {
		if err := persistRatchet(destCopy, ratchetCopy); err != nil {
			Logf(LogError, "Could not persist ratchet for %s: %v", PrettyHexRep(destCopy), err)
		}
	}()
}

func IdentityGetRatchet(destHash []byte) []byte {
	if len(destHash) == 0 {
		return nil
	}
	knownRatchets.RLock()
	if ratchet, ok := knownRatchets.store[string(destHash)]; ok {
		knownRatchets.RUnlock()
		return append([]byte{}, ratchet...)
	}
	knownRatchets.RUnlock()

	ratchet, err := loadRatchetFromDisk(destHash)
	if err != nil {
		Logf(LogError, "Could not load ratchet for %s: %v", PrettyHexRep(destHash), err)
		return nil
	}
	if len(ratchet) == 0 {
		return nil
	}

	knownRatchets.Lock()
	knownRatchets.store[string(destHash)] = append([]byte{}, ratchet...)
	knownRatchets.Unlock()
	return append([]byte{}, ratchet...)
}

// IdentityCleanRatchets removes expired or corrupted records from storage/ratchets.
func IdentityCleanRatchets() {
	Log("Cleaning ratchets...", LogDebug)
	dir := ratchetDirectory()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		Logf(LogError, "An error occurred while cleaning ratchets: %v", err)
		return
	}

	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			Logf(LogError, "Could not read ratchet file %s: %v", path, err)
			continue
		}

		remove := false
		var rec ratchetRecord
		if err := umsgpack.Unpackb(data, &rec); err != nil {
			Logf(LogError, "Corrupted ratchet data while reading %s, removing file", path)
			remove = true
		} else {
			if rec.Received == 0 {
				rec.Received = float64(now.Unix())
			}
			if len(rec.Ratchet) != x25519KeyLen {
				remove = true
			} else {
				sec, frac := math.Modf(rec.Received)
				stored := time.Unix(int64(sec), int64(frac*1e9))
				if sec < 0 {
					stored = time.Unix(0, 0)
				}
				if now.Sub(stored) > ratchetExpiry {
					remove = true
				}
			}
		}

		if remove {
			if err := os.Remove(path); err != nil {
				Logf(LogError, "Could not remove ratchet file %s: %v", path, err)
			}
		}
	}
}

func IdentityGetRatchetID(ratchet []byte) []byte {
	if len(ratchet) == 0 {
		return nil
	}
	hash := FullHash(ratchet)
	size := IdentityNameHashLength / 8
	if size <= 0 || size > len(hash) {
		return append([]byte{}, hash...)
	}
	return append([]byte{}, hash[:size]...)
}

func ratchetDirectory() string {
	if inst := GetInstance(); inst != nil && inst.StoragePath != "" {
		return filepath.Join(inst.StoragePath, "ratchets")
	}
	return filepath.Join(osUserDir(), ".reticulum", "storage", "ratchets")
}

func persistRatchet(destHash, ratchet []byte) error {
	ratchetPersistLock.Lock()
	defer ratchetPersistLock.Unlock()

	dir := ratchetDirectory()
	ensureDir(dir)

	hexHash := hex.EncodeToString(destHash)
	tmpPath := filepath.Join(dir, hexHash+".tmp")
	finalPath := filepath.Join(dir, hexHash)

	payload, err := umsgpack.Packb(map[string]any{
		"ratchet":  append([]byte{}, ratchet...),
		"received": float64(time.Now().UnixNano()) / 1e9,
	})
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmpPath, payload, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpPath, finalPath)
}

func loadRatchetFromDisk(destHash []byte) ([]byte, error) {
	ratchetPersistLock.Lock()
	defer ratchetPersistLock.Unlock()

	path := filepath.Join(ratchetDirectory(), hex.EncodeToString(destHash))
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var rec ratchetRecord
	if err := umsgpack.Unpackb(data, &rec); err != nil {
		_ = os.Remove(path)
		return nil, err
	}
	if len(rec.Ratchet) != x25519KeyLen {
		_ = os.Remove(path)
		return nil, errors.New("invalid ratchet size on disk")
	}
	if rec.Received == 0 {
		rec.Received = float64(time.Now().UnixNano()) / 1e9
	}
	sec, frac := math.Modf(rec.Received)
	stored := time.Unix(int64(sec), int64(frac*1e9))
	if sec < 0 {
		stored = time.Unix(0, 0)
	}
	if time.Since(stored) > ratchetExpiry {
		_ = os.Remove(path)
		return nil, nil
	}
	return rec.Ratchet, nil
}
