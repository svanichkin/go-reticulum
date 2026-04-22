package rns

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"

	umsgpack "github.com/svanichkin/go-reticulum/rns/vendor"
)

func pythonKnownDestinationsMsgpack(key16, packetHash, publicKey, appData []byte, seenAt float64) []byte {
	// msgpack encoding for: { <bin16 key>: [seenAt, packetHash, publicKey, appData] }
	//
	// fixmap(1)         0x81
	// bin8 + len        0xc4 0x10 + 16 bytes
	// fixarray(4)       0x94
	// float64           0xcb + 8
	// bin8 fields...
	buf := make([]byte, 0, 128)
	buf = append(buf, 0x81)
	buf = append(buf, 0xc4, byte(len(key16)))
	buf = append(buf, key16...)
	buf = append(buf, 0x94)
	buf = append(buf, 0xcb)
	var f [8]byte
	binary.BigEndian.PutUint64(f[:], math.Float64bits(seenAt))
	buf = append(buf, f[:]...)
	for _, b := range [][]byte{packetHash, publicKey, appData} {
		if len(b) < 256 {
			buf = append(buf, 0xc4, byte(len(b)))
		} else {
			buf = append(buf, 0xc5, byte(len(b)>>8), byte(len(b)))
		}
		buf = append(buf, b...)
	}
	return buf
}

func TestIdentityKnownDestinations_EncodeDecode_PythonCompatibleKeys(t *testing.T) {
	t.Parallel()

	key := bytes.Repeat([]byte{0x01}, ReticulumTruncatedHashLength/8)
	pub := make([]byte, 0, 64)
	pub = append(pub, bytes.Repeat([]byte{0x02}, 32)...)
	pub = append(pub, bytes.Repeat([]byte{0x03}, 32)...)

	in := map[string]*knownDestinationEntry{
		string(key): {
			SeenAt:     123.0,
			PacketHash: []byte("ph"),
			PublicKey:  pub,
			AppData:    []byte("ad"),
		},
	}

	payload := make(map[[truncatedHashBytes]byte][]any, len(in))
	for key, entry := range in {
		var keyBytes [truncatedHashBytes]byte
		copy(keyBytes[:], []byte(key))
		payload[keyBytes] = []any{entry.SeenAt, entry.PacketHash, entry.PublicKey, entry.AppData}
	}

	enc, err := umsgpack.Packb(payload)
	if err != nil {
		t.Fatalf("Packb: %v", err)
	}

	decode := func(raw []byte) map[string]*knownDestinationEntry {
		var unpacked map[any]any
		if err := umsgpack.Unpackb(raw, &unpacked); err != nil {
			t.Fatalf("Unpackb: %v", err)
		}
		out := make(map[string]*knownDestinationEntry, len(unpacked))
		for k, v := range unpacked {
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
			values, ok := v.([]any)
			if !ok {
				continue
			}
			entry := &knownDestinationEntry{}
			if len(values) > 0 {
				entry.SeenAt = asFloat64(values[0])
			}
			if len(values) > 1 {
				switch val := values[1].(type) {
				case []byte:
					entry.PacketHash = append([]byte(nil), val...)
				case string:
					entry.PacketHash = []byte(val)
				}
			}
			if len(values) > 2 {
				switch val := values[2].(type) {
				case []byte:
					entry.PublicKey = append([]byte(nil), val...)
				case string:
					entry.PublicKey = []byte(val)
				}
			}
			if len(values) > 3 {
				switch val := values[3].(type) {
				case []byte:
					entry.AppData = append([]byte(nil), val...)
				case string:
					entry.AppData = []byte(val)
				}
			}
			if len(entry.PublicKey) != identityPubKeyLen {
				continue
			}
			out[string(keyBytes)] = entry
		}
		return out
	}

	out := decode(enc)

	got := out[string(key)]
	if got == nil {
		t.Fatalf("missing decoded entry")
	}
	if got.SeenAt != 123.0 {
		t.Fatalf("unexpected SeenAt: %v", got.SeenAt)
	}
	if !bytes.Equal(got.PublicKey, pub) {
		t.Fatalf("unexpected PublicKey")
	}
	if !bytes.Equal(got.AppData, []byte("ad")) {
		t.Fatalf("unexpected AppData")
	}
}

func TestIdentityKnownDestinations_Decode_AcceptsStringKeys(t *testing.T) {
	t.Parallel()

	keyBytes := bytes.Repeat([]byte{0x01}, ReticulumTruncatedHashLength/8)
	keyStr := string(keyBytes)
	pub := make([]byte, 0, 64)
	pub = append(pub, bytes.Repeat([]byte{0x02}, 32)...)
	pub = append(pub, bytes.Repeat([]byte{0x03}, 32)...)

	legacy := map[any]any{
		keyStr: []any{1.0, []byte("ph"), pub, []byte("ad")},
	}
	raw, err := umsgpack.Packb(legacy)
	if err != nil {
		t.Fatalf("pack: %v", err)
	}

	decode := func(raw []byte) map[string]*knownDestinationEntry {
		var unpacked map[any]any
		if err := umsgpack.Unpackb(raw, &unpacked); err != nil {
			t.Fatalf("Unpackb: %v", err)
		}
		out := make(map[string]*knownDestinationEntry, len(unpacked))
		for k, v := range unpacked {
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
			values, ok := v.([]any)
			if !ok {
				continue
			}
			entry := &knownDestinationEntry{}
			if len(values) > 0 {
				entry.SeenAt = asFloat64(values[0])
			}
			if len(values) > 1 {
				switch val := values[1].(type) {
				case []byte:
					entry.PacketHash = append([]byte(nil), val...)
				case string:
					entry.PacketHash = []byte(val)
				}
			}
			if len(values) > 2 {
				switch val := values[2].(type) {
				case []byte:
					entry.PublicKey = append([]byte(nil), val...)
				case string:
					entry.PublicKey = []byte(val)
				}
			}
			if len(values) > 3 {
				switch val := values[3].(type) {
				case []byte:
					entry.AppData = append([]byte(nil), val...)
				case string:
					entry.AppData = []byte(val)
				}
			}
			if len(entry.PublicKey) != identityPubKeyLen {
				continue
			}
			out[string(keyBytes)] = entry
		}
		return out
	}

	out := decode(raw)
	if out[keyStr] == nil {
		t.Fatalf("missing decoded entry")
	}
}

func TestIdentityKnownDestinations_Decode_PythonBytesKeys(t *testing.T) {
	t.Parallel()

	key := bytes.Repeat([]byte{0x01}, ReticulumTruncatedHashLength/8)
	pub := make([]byte, 0, 64)
	pub = append(pub, bytes.Repeat([]byte{0x02}, 32)...)
	pub = append(pub, bytes.Repeat([]byte{0x03}, 32)...)

	raw := pythonKnownDestinationsMsgpack(key, []byte("ph"), pub, []byte("ad"), 1.0)
	var unpacked map[any]any
	if err := umsgpack.Unpackb(raw, &unpacked); err != nil {
		t.Fatalf("Unpackb: %v", err)
	}
	out := make(map[string]*knownDestinationEntry, len(unpacked))
	for k, v := range unpacked {
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
		values, ok := v.([]any)
		if !ok {
			continue
		}
		entry := &knownDestinationEntry{}
		if len(values) > 0 {
			entry.SeenAt = asFloat64(values[0])
		}
		if len(values) > 1 {
			switch val := values[1].(type) {
			case []byte:
				entry.PacketHash = append([]byte(nil), val...)
			case string:
				entry.PacketHash = []byte(val)
			}
		}
		if len(values) > 2 {
			switch val := values[2].(type) {
			case []byte:
				entry.PublicKey = append([]byte(nil), val...)
			case string:
				entry.PublicKey = []byte(val)
			}
		}
		if len(values) > 3 {
			switch val := values[3].(type) {
			case []byte:
				entry.AppData = append([]byte(nil), val...)
			case string:
				entry.AppData = []byte(val)
			}
		}
		if len(entry.PublicKey) != identityPubKeyLen {
			continue
		}
		out[string(keyBytes)] = entry
	}
	if out[string(key)] == nil {
		t.Fatalf("missing decoded entry")
	}
}
