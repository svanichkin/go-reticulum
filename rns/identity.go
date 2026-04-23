package rns

import (
	"bytes"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
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
	knownDestinationsSaving        atomic.Bool
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
	id.CreateKeys()
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
	if len(id.prvBytes) != x25519KeyLen || len(id.sigPrivBytes) != ed25519SeedLen {
		return errors.New("identity has no private key material")
	}
	return os.WriteFile(path, id.GetPrivateKey(), 0o600)
}

// Load mirrors load(path).
func (id *Identity) Load(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		Log("Error while loading identity from "+path, LogError)
		Log("The contained exception was: "+err.Error(), LogError)
		return false
	}
	if err := id.LoadPrivateKey(data); err != nil {
		Log("Error while loading identity from "+path, LogError)
		Log("The contained exception was: "+err.Error(), LogError)
		return false
	}
	return true
}

// CreateKeys generates X25519 + Ed25519 keypairs.
func (id *Identity) CreateKeys() {
	if id.curve == nil {
		id.curve = ecdh.X25519()
	}

	// X25519
	prv, err := id.curve.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	id.prv = prv
	id.prvBytes = prv.Bytes()
	pub := prv.PublicKey()
	id.pub = pub
	id.pubBytes = pub.Bytes()

	// Ed25519
	pubSig, prvSig, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	id.sigPriv = prvSig
	id.sigPrivBytes = prvSig.Seed()
	id.sigPub = pubSig
	id.sigPubBytes = pubSig

	id.updateHashes()
	Logf(LogVerbose, "Identity keys created for %s", PrettyHexRep(id.Hash))
}

// GetPrivateKey mirrors get_private_key().
func (id *Identity) GetPrivateKey() []byte {
	return append(append([]byte{}, id.prvBytes...), id.sigPrivBytes...)
}

// GetPublicKey mirrors get_public_key().
func (id *Identity) GetPublicKey() []byte {
	return append(append([]byte{}, id.pubBytes...), id.sigPubBytes...)
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

	switch len(suffix) {
	case ed25519SeedLen:
		id.sigPrivBytes = append([]byte{}, suffix...)
		id.sigPriv = ed25519.NewKeyFromSeed(id.sigPrivBytes)
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

// IdentityGetRandomHash returns a random truncated hash.
func IdentityGetRandomHash() []byte {
	buf := make([]byte, truncatedHashBytes)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
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
func IdentityRemember(packetHash, destinationHash, publicKey, appData []byte) {
	if len(publicKey) != identityPubKeyLen {
		panic(fmt.Errorf("Can't remember %s, the public key size of %d is not valid.", PrettyHexRep(destinationHash), len(publicKey)))
	}

	entry := &knownDestinationEntry{
		SeenAt:     float64(time.Now().UnixNano()) / 1e9,
		PacketHash: append([]byte(nil), packetHash...),
		PublicKey:  append([]byte(nil), publicKey...),
		AppData:    append([]byte(nil), appData...),
	}

	key := string(destinationHash)
	knownDestinations.Lock()
	knownDestinations.entries[key] = entry
	knownDestinations.Unlock()
}

// IdentityRecall looks up an identity by destination hash or identity hash.
func IdentityRecall(targetHash []byte, fromIdentityHash ...bool) *Identity {
	searchIdentityHash := len(fromIdentityHash) > 0 && fromIdentityHash[0]

	if searchIdentityHash {
		knownDestinations.RLock()
		for _, entry := range knownDestinations.entries {
			if bytes.Equal(targetHash, TruncatedHash(entry.PublicKey)) {
				id := &Identity{}
				_ = id.LoadPublicKey(entry.PublicKey)
				id.AppData = entry.AppData
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
		id := &Identity{}
		_ = id.LoadPublicKey(entry.PublicKey)
		id.AppData = entry.AppData
		return id
	}

	for _, dst := range Destinations {
		if dst == nil {
			continue
		}
		if bytes.Equal(targetHash, dst.Hash) {
			id := &Identity{}
			_ = id.LoadPublicKey(dst.identity.GetPublicKey())
			id.AppData = nil
			return id
		}
	}

	return nil
}

// IdentityRecallAppData returns the last app_data.
func IdentityRecallAppData(destinationHash []byte) []byte {
	knownDestinations.RLock()
	defer knownDestinations.RUnlock()
	if entry, ok := knownDestinations.entries[string(destinationHash)]; ok {
		return entry.AppData
	}
	return nil
}

// IdentitySaveKnownDestinations writes the map to disk.
func IdentitySaveKnownDestinations() error {
	if knownDestinationsSaving.Load() {
		waitInterval := 200 * time.Millisecond
		waitTimeout := 5 * time.Second
		waitStart := time.Now()
		for knownDestinationsSaving.Load() {
			time.Sleep(waitInterval)
			if time.Since(waitStart) > waitTimeout {
				err := errors.New("Could not save known destinations to storage, waiting for previous save operation timed out.")
				Logf(LogError, "%v", err)
				return err
			}
		}
	}

	knownDestinationsSaveMu.Lock()
	defer knownDestinationsSaveMu.Unlock()
	knownDestinationsSaving.Store(true)
	defer knownDestinationsSaving.Store(false)

	saveStart := time.Now()

	path := filepath.Join(osUserDir(), ".reticulum", "storage", "known_destinations")
	if inst := GetInstance(); inst != nil && inst.StoragePath != "" {
		path = filepath.Join(inst.StoragePath, "known_destinations")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	// Merge on-disk data so we don't lose entries from other processes.
	if data, err := os.ReadFile(path); err == nil {
		var raw map[any]any
		if err := umsgpack.Unpackb(data, &raw); err == nil {
			diskEntries := make(map[string]*knownDestinationEntry, len(raw))
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
					switch val := values[1].(type) {
					case nil:
					case []byte:
						entry.PacketHash = append([]byte(nil), val...)
					case string:
						entry.PacketHash = []byte(val)
					}
				}
				if len(values) > 2 {
					switch val := values[2].(type) {
					case nil:
					case []byte:
						entry.PublicKey = append([]byte(nil), val...)
					case string:
						entry.PublicKey = []byte(val)
					}
				}
				if len(values) > 3 {
					switch val := values[3].(type) {
					case nil:
					case []byte:
						entry.AppData = append([]byte(nil), val...)
					case string:
						entry.AppData = []byte(val)
					}
				}
				diskEntries[string(keyBytes)] = entry
			}

			knownDestinations.Lock()
			for k, v := range diskEntries {
				if _, ok := knownDestinations.entries[k]; !ok {
					knownDestinations.entries[k] = v
				}
			}
			knownDestinations.Unlock()
		}
	}

	knownDestinations.RLock()
	snapshot := make(map[string]*knownDestinationEntry, len(knownDestinations.entries))
	for k, v := range knownDestinations.entries {
		if v == nil {
			continue
		}
		snapshot[k] = &knownDestinationEntry{
			SeenAt:     v.SeenAt,
			PacketHash: append([]byte(nil), v.PacketHash...),
			PublicKey:  append([]byte(nil), v.PublicKey...),
			AppData:    append([]byte(nil), v.AppData...),
		}
	}
	knownDestinations.RUnlock()

	payload := make(map[[truncatedHashBytes]byte][]any, len(snapshot))
	for key, entry := range snapshot {
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

	data, err := umsgpack.Packb(payload)
	if err != nil {
		return err
	}

	Logf(LogDebug, "Saving %d known destinations to storage...", len(snapshot))

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}

	saveTime := time.Since(saveStart).Seconds()
	if saveTime < 1 {
		Logf(LogDebug, "Saved known destinations to storage in %.2fms", saveTime*1000)
	} else {
		Logf(LogDebug, "Saved known destinations to storage in %.2fs", saveTime)
	}
	return nil
}

// IdentityLoadKnownDestinations reads the map from storage/known_destinations.
func IdentityLoadKnownDestinations() error {
	path := filepath.Join(osUserDir(), ".reticulum", "storage", "known_destinations")
	if inst := GetInstance(); inst != nil && inst.StoragePath != "" {
		path = filepath.Join(inst.StoragePath, "known_destinations")
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		Logf(LogVerbose, "Destinations file does not exist, no known destinations loaded")
		return nil
	}
	if err != nil {
		Logf(LogError, "Error loading known destinations from disk, file will be recreated on exit")
		return nil
	}

	var raw map[any]any
	if err := umsgpack.Unpackb(data, &raw); err != nil {
		Logf(LogError, "Error loading known destinations from disk, file will be recreated on exit")
		return nil
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
			switch val := values[1].(type) {
			case nil:
			case []byte:
				entry.PacketHash = append([]byte(nil), val...)
			case string:
				entry.PacketHash = []byte(val)
			}
		}
		if len(values) > 2 {
			switch val := values[2].(type) {
			case nil:
			case []byte:
				entry.PublicKey = append([]byte(nil), val...)
			case string:
				entry.PublicKey = []byte(val)
			}
		}
		if len(values) > 3 {
			switch val := values[3].(type) {
			case nil:
			case []byte:
				entry.AppData = append([]byte(nil), val...)
			case string:
				entry.AppData = []byte(val)
			}
		}
		entries[string(keyBytes)] = entry
	}

	knownDestinations.Lock()
	knownDestinations.entries = entries
	knownDestinations.Unlock()

	Logf(LogVerbose, "Loaded %d known destination from storage", len(entries))
	return nil
}

// IdentityPersistData persists the map if we are a standalone instance.
func IdentityPersistData() {
	if inst := GetInstance(); inst != nil && inst.IsConnectedToSharedInstance {
		return
	}
	if err := IdentitySaveKnownDestinations(); err != nil {
		Logf(LogError, "Error while saving known destinations to disk, the contained exception was: %v", err)
		TraceException(err)
	}
}

// IdentityExitHandler runs during Reticulum shutdown.
func IdentityExitHandler() {
	IdentityPersistData()
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
func ValidateAnnounce(packet *Packet, onlyValidateSignature bool) (valid bool) {
	defer func() {
		if r := recover(); r != nil {
			Logf(LogError, "Error occurred while validating announce. The contained exception was: %v", r)
			valid = false
		}
	}()

	if packet == nil || packet.PacketType != PacketTypeAnnounce {
		return false
	}

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

	appData := []byte{}
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
		Logf(LogError, "Error while loading public key, the contained exception was: %v", err)
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

	if !(announced.pub != nil && announced.Validate(signature, signed)) {
		Logf(LogDebug, "Received invalid announce for %s: Invalid signature.", PrettyHexRep(packet.DestinationHash))
		return false
	}

	if onlyValidateSignature {
		return true
	}

	hashMaterial := append([]byte{}, nameHash...)
	hashMaterial = append(hashMaterial, announced.Hash...)
	expectedHash := FullHash(hashMaterial)[:ReticulumTruncatedHashLength/8]
	if len(packet.DestinationHash) != ReticulumTruncatedHashLength/8 || !bytes.Equal(packet.DestinationHash, expectedHash) {
		Logf(LogDebug, "Received invalid announce for %s: Destination mismatch.", PrettyHexRep(packet.DestinationHash))
		return false
	}

	knownDestinations.RLock()
	if entry, ok := knownDestinations.entries[string(packet.DestinationHash)]; ok && entry != nil && !bytes.Equal(entry.PublicKey, publicKey) {
		knownDestinations.RUnlock()
		Log("Received announce with valid signature and destination hash, but announced public key does not match already known public key.", LogCritical)
		Log("This may indicate an attempt to modify network paths, or a random hash collision. The announce was rejected.", LogCritical)
		return false
	}
	knownDestinations.RUnlock()

	IdentityRemember(packet.GetHash(), packet.DestinationHash, publicKey, appData)

	if len(ratchet) == ratchetLen {
		IdentityRememberRatchet(packet.DestinationHash, ratchet)
	}

	signal := ""
	if packet.RSSI != nil || packet.SNR != nil {
		parts := []string{}
		if packet.RSSI != nil {
			parts = append(parts, fmt.Sprintf("RSSI %.0fdBm", *packet.RSSI))
		}
		if packet.SNR != nil {
			parts = append(parts, fmt.Sprintf("SNR %.0fdB", *packet.SNR))
		}
		if len(parts) > 0 {
			signal = " [" + strings.Join(parts, ", ") + "]"
		}
	}
	if len(packet.TransportID) > 0 {
		Logf(LogExtreme, "Valid announce for %s %d hops away, received via %s on %v%s",
			PrettyHexRep(packet.DestinationHash), packet.Hops, PrettyHexRep(packet.TransportID), packet.ReceivingInterface, signal)
	} else {
		Logf(LogExtreme, "Valid announce for %s %d hops away, received on %v%s",
			PrettyHexRep(packet.DestinationHash), packet.Hops, packet.ReceivingInterface, signal)
	}
	return true
}

// Encrypt mirrors encrypt(self, plaintext, ratchet=None).
// Returns ephemeral_pub || ciphertext.
func (id *Identity) Encrypt(plaintext []byte, ratchet []byte) ([]byte, error) {
	if id.pub == nil {
		panic(errors.New("Encryption failed because identity does not hold a public key"))
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

	derived, err := Cryptography.HKDF(derivedKeyLen, shared, id.GetSalt(), id.GetContext())
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
	derived, err := Cryptography.HKDF(derivedKeyLen, shared, id.GetSalt(), id.GetContext())
	if err != nil {
		return nil, err
	}
	tok, err := Cryptography.NewToken(derived)
	if err != nil {
		return nil, err
	}
	plaintext, err := tok.Decrypt(ciphertext)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

// Decrypt mirrors decrypt(self, ciphertext_token, ratchets=None, enforce_ratchets=False).
func (id *Identity) Decrypt(ciphertextToken []byte, ratchets [][]byte, enforceRatchets bool) ([]byte, error) {
	if id.prv == nil {
		panic(errors.New("Decryption failed because identity does not hold a private key"))
	}
	if len(ciphertextToken) <= x25519KeyLen {
		Log("Decryption failed because the token size was invalid.", LogDebug)
		return nil, nil
	}
	if id.curve == nil {
		id.curve = ecdh.X25519()
	}

	peerPubBytes := ciphertextToken[:x25519KeyLen]
	ciphertext := ciphertextToken[x25519KeyLen:]

	peerPub, err := id.curve.NewPublicKey(peerPubBytes)
	if err != nil {
		Logf(LogDebug, "Decryption by %s failed: %v", PrettyHexRep(id.Hash), err)
		return nil, nil
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
			return pt, nil
		}
	}

	if enforceRatchets {
		Logf(LogDebug, "Decryption with ratchet enforcement by %s failed. Dropping packet.", PrettyHexRep(id.Hash))
		return nil, nil
	}

	shared, err := id.prv.ECDH(peerPub)
	if err != nil {
		Logf(LogDebug, "Decryption by %s failed: %v", PrettyHexRep(id.Hash), err)
		return nil, nil
	}
	pt, err := id.decryptWithShared(shared, ciphertext)
	if err != nil {
		Logf(LogDebug, "Decryption by %s failed: %v", PrettyHexRep(id.Hash), err)
		return nil, nil
	}
	return pt, nil
}

// Sign mirrors sign().
func (id *Identity) Sign(msg []byte) (sig []byte, err error) {
	if id.sigPriv == nil {
		panic(errors.New("Signing failed because identity does not hold a private key"))
	}
	defer func() {
		if r := recover(); r != nil {
			Logf(LogError, "The identity %s could not sign the requested message. The contained exception was: %v", id, r)
			panic(r)
		}
	}()
	return ed25519.Sign(id.sigPriv, msg), nil
}

// Validate mirrors validate().
func (id *Identity) Validate(sig, msg []byte) bool {
	if id.sigPub == nil {
		panic(errors.New("Signature validation failed because identity does not hold a public key"))
	}
	return ed25519.Verify(id.sigPub, msg, sig)
}

// String mirrors __str__().
func (id *Identity) String() string {
	return PrettyHexRep(id.Hash)
}

// Prove sends a proof packet to confirm delivery.
func (id *Identity) Prove(packet *Packet, destination *Destination) {
	if id == nil || packet == nil {
		return
	}

	packetHash := packet.PacketHash
	if len(packetHash) == 0 {
		packetHash = packet.GetHash()
	}

	signature := func() []byte {
		sig, err := id.Sign(packetHash)
		if err != nil {
			panic(err)
		}
		return sig
	}()

	var proofData []byte
	if ShouldUseImplicitProof() {
		proofData = signature
	} else {
		proofData = append(append([]byte{}, packetHash...), signature...)
	}

	if destination == nil {
		destination = packet.GenerateProofDestination()
	}

	proof := NewPacket(
		destination,
		proofData,
		PacketTypeProof,
		PacketCtxNone,
		Broadcast,
		HeaderType1,
		nil,
		packet.ReceivingInterface,
		false,
		FlagUnset,
	)
	proof.FromPacked = true
	proof.Destination = destination
	if proof.Send() == nil {
		Logf(LogDebug, "Sent proof for %s", PrettyHexRep(packetHash))
	}
}

// ---- Ratchet helpers ----

func IdentityGenerateRatchet() []byte {
	curve := ecdh.X25519()
	priv, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	return priv.Bytes()
}

func IdentityRatchetPublicBytes(private []byte) []byte {
	if len(private) != x25519KeyLen {
		panic(errors.New("invalid ratchet length"))
	}
	curve := ecdh.X25519()
	key, err := curve.NewPrivateKey(private)
	if err != nil {
		panic(err)
	}
	return key.PublicKey().Bytes()
}

func IdentityRememberRatchet(destHash, ratchet []byte) {
	defer func() {
		if r := recover(); r != nil {
			Logf(LogError, "Could not persist ratchet for %s to storage.", PrettyHexRep(destHash))
			Logf(LogVerbose, "The contained exception was: %v", r)
			TraceException(r)
		}
	}()

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
		defer func() {
			if r := recover(); r != nil {
				Logf(LogError, "Could not persist ratchet for %s to storage.", PrettyHexRep(destCopy))
				Logf(LogError, "The contained exception was: %v", r)
			}
		}()

		ratchetPersistLock.Lock()
		defer ratchetPersistLock.Unlock()

		dir := filepath.Join(func() string {
			if inst := GetInstance(); inst != nil && inst.StoragePath != "" {
				return inst.StoragePath
			}
			return filepath.Join(osUserDir(), ".reticulum", "storage")
		}(), "ratchets")
		ensureDir(dir)

		hexHash := hex.EncodeToString(destCopy)
		tmpPath := filepath.Join(dir, hexHash+".tmp")
		finalPath := filepath.Join(dir, hexHash)

		payload, err := umsgpack.Packb(map[string]any{
			"ratchet":  ratchetCopy,
			"received": float64(time.Now().UnixNano()) / 1e9,
		})
		if err != nil {
			return
		}
		if err := os.WriteFile(tmpPath, payload, 0o600); err != nil {
			return
		}
		if err := os.Rename(tmpPath, finalPath); err != nil {
			return
		}
	}()
}

func IdentityGetRatchet(destHash []byte) []byte {
	knownRatchets.RLock()
	if ratchet, ok := knownRatchets.store[string(destHash)]; ok {
		knownRatchets.RUnlock()
		return append([]byte{}, ratchet...)
	}
	knownRatchets.RUnlock()

	path := filepath.Join(func() string {
		if inst := GetInstance(); inst != nil && inst.StoragePath != "" {
			return inst.StoragePath
		}
		return filepath.Join(osUserDir(), ".reticulum", "storage")
	}(), "ratchets", hex.EncodeToString(destHash))
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			Logf(LogDebug, "Could not load ratchet for %s", PrettyHexRep(destHash))
			return nil
		}
		Logf(LogError, "An error occurred while loading ratchet data for %s from storage.", PrettyHexRep(destHash))
		Logf(LogError, "The contained exception was: %v", err)
		return nil
	}
	var rec ratchetRecord
	if err := umsgpack.Unpackb(data, &rec); err != nil {
		Logf(LogError, "An error occurred while loading ratchet data for %s from storage.", PrettyHexRep(destHash))
		Logf(LogError, "The contained exception was: %v", err)
		return nil
	}
	if len(rec.Ratchet) != x25519KeyLen {
		Logf(LogDebug, "Could not load ratchet for %s", PrettyHexRep(destHash))
		return nil
	}
	if time.Since(time.Unix(int64(math.Floor(rec.Received)), int64((rec.Received-math.Floor(rec.Received))*1e9))) > ratchetExpiry {
		Logf(LogDebug, "Could not load ratchet for %s", PrettyHexRep(destHash))
		return nil
	}
	ratchet := rec.Ratchet
	if len(ratchet) == 0 {
		Logf(LogDebug, "Could not load ratchet for %s", PrettyHexRep(destHash))
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
	dir := filepath.Join(func() string {
		if inst := GetInstance(); inst != nil && inst.StoragePath != "" {
			return inst.StoragePath
		}
		return filepath.Join(osUserDir(), ".reticulum", "storage")
	}(), "ratchets")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		Logf(LogError, "An error occurred while cleaning ratchets. The contained exception was: %v", err)
		return
	}

	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		func() {
			data, err := os.ReadFile(path)
			if err != nil {
				Logf(LogError, "An error occurred while cleaning ratchets, in the processing of %s.", path)
				Logf(LogError, "The contained exception was: %v", err)
				return
			}

			var rec ratchetRecord
			if err := umsgpack.Unpackb(data, &rec); err != nil {
				Logf(LogError, "Corrupted ratchet data while reading %s, removing file", path)
				if err := os.Remove(path); err != nil {
					Logf(LogError, "Could not remove ratchet file %s: %v", path, err)
				}
				return
			}

			sec, frac := math.Modf(rec.Received)
			stored := time.Unix(int64(sec), int64(frac*1e9))
			if sec < 0 {
				stored = time.Unix(0, 0)
			}
			if now.Sub(stored) > ratchetExpiry {
				if err := os.Remove(path); err != nil {
					Logf(LogError, "Could not remove ratchet file %s: %v", path, err)
				}
			}
		}()
	}
}

func IdentityGetRatchetID(ratchet []byte) []byte {
	hash := FullHash(ratchet)
	size := IdentityNameHashLength / 8
	return append([]byte{}, hash[:size]...)
}
