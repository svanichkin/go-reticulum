package vendor

import (
	"bytes"
	"testing"
)

func TestUMsgpack_Packb_StringLengthEncodings(t *testing.T) {
	t.Parallel()

	short := bytes.Repeat([]byte("a"), 100)
	b, err := Packb(string(short))
	if err != nil {
		t.Fatalf("Packb: %v", err)
	}
	if len(b) == 0 || b[0] != codeStr8 {
		t.Fatalf("expected STR8 (0xD9) prefix, got %02x", firstByte(b))
	}

	long := bytes.Repeat([]byte("a"), 300)
	b, err = Packb(string(long))
	if err != nil {
		t.Fatalf("Packb: %v", err)
	}
	if len(b) == 0 || b[0] != codeStr16 {
		t.Fatalf("expected STR16 (0xDA) prefix, got %02x", firstByte(b))
	}
}

func TestUMsgpack_Packb_BinLengthEncodings(t *testing.T) {
	t.Parallel()

	short := bytes.Repeat([]byte{0xAB}, 200)
	b, err := Packb(short)
	if err != nil {
		t.Fatalf("Packb: %v", err)
	}
	if len(b) == 0 || b[0] != codeBin8 {
		t.Fatalf("expected BIN8 (0xC4) prefix, got %02x", firstByte(b))
	}

	long := bytes.Repeat([]byte{0xCD}, 300)
	b, err = Packb(long)
	if err != nil {
		t.Fatalf("Packb: %v", err)
	}
	if len(b) == 0 || b[0] != codeBin16 {
		t.Fatalf("expected BIN16 (0xC5) prefix, got %02x", firstByte(b))
	}
}

func TestUMsgpack_Packb_ByteArrayEncodesAsBin(t *testing.T) {
	t.Parallel()

	in := [3]byte{0xAA, 0xBB, 0xCC}
	b, err := Packb(in)
	if err != nil {
		t.Fatalf("Packb: %v", err)
	}
	if len(b) == 0 || b[0] != codeBin8 {
		t.Fatalf("expected BIN8 (0xC4) prefix, got %02x", firstByte(b))
	}

	var out []byte
	if err := Unpackb(b, &out); err != nil {
		t.Fatalf("Unpackb: %v", err)
	}
	if !bytes.Equal(out, in[:]) {
		t.Fatalf("unexpected bytes: %x", out)
	}
}

func TestUMsgpack_RoundTrip_StructAssignment(t *testing.T) {
	t.Parallel()

	type payload struct {
		Name  string `msgpack:"name"`
		Count int    `msgpack:"count"`
	}

	in := map[string]any{
		"NAME":  "alice", // case-insensitive matching
		"count": int64(7),
	}
	b, err := Packb(in)
	if err != nil {
		t.Fatalf("Packb: %v", err)
	}

	var out payload
	if err := Unpackb(b, &out); err != nil {
		t.Fatalf("Unpackb: %v", err)
	}
	if out.Name != "alice" || out.Count != 7 {
		t.Fatalf("unexpected struct: %+v", out)
	}
}

func TestUMsgpack_Unpackb_TargetValidation(t *testing.T) {
	t.Parallel()

	if err := Unpackb([]byte{codeNil}, nil); err == nil {
		t.Fatalf("expected error for nil target")
	}
	var notPtr int
	if err := Unpackb([]byte{codeNil}, notPtr); err == nil {
		t.Fatalf("expected error for non-pointer target")
	}
	var nilPtr *int
	if err := Unpackb([]byte{codeNil}, nilPtr); err == nil {
		t.Fatalf("expected error for nil pointer target")
	}
}

func TestUMsgpack_BinMapKeyDecodesToComparableKey(t *testing.T) {
	t.Parallel()

	// MessagePack allows bin keys; our decoder converts []byte keys to string
	// for safe use in a Go map[any]any.
	//
	// We can't construct a Go map with []byte keys (not comparable), so craft
	// the msgpack payload directly:
	//   fixmap(1) + bin8("k") + fixint(1)
	b := []byte{0x81, codeBin8, 0x01, 'k', 0x01}

	var out map[any]any
	if err := Unpackb(b, &out); err != nil {
		t.Fatalf("Unpackb: %v", err)
	}
	if v, ok := out["k"]; !ok || v.(int64) != 1 {
		t.Fatalf("unexpected map: %#v", out)
	}
}

func TestUMsgpack_Ext_RoundTrip(t *testing.T) {
	t.Parallel()

	in := Ext{Type: 5, Data: bytes.Repeat([]byte{0xAB}, 3)}
	b, err := Packb(in)
	if err != nil {
		t.Fatalf("Packb: %v", err)
	}

	var out Ext
	if err := Unpackb(b, &out); err != nil {
		t.Fatalf("Unpackb: %v", err)
	}
	if out.Type != in.Type || !bytes.Equal(out.Data, in.Data) {
		t.Fatalf("ext mismatch: got %#v want %#v", out, in)
	}
}

func TestUMsgpack_Ext_FixExt8Decode(t *testing.T) {
	t.Parallel()

	// fixext 8: 0xD7 + type + 8 bytes
	b := append([]byte{codeFixExt8, 0xFE}, bytes.Repeat([]byte{0x11}, 8)...)
	var out Ext
	if err := Unpackb(b, &out); err != nil {
		t.Fatalf("Unpackb: %v", err)
	}
	// 0xFE interpreted as signed int8 equals -2.
	if out.Type != int8(-2) || len(out.Data) != 8 {
		t.Fatalf("unexpected ext: %#v", out)
	}
}

func firstByte(b []byte) byte {
	if len(b) == 0 {
		return 0
	}
	return b[0]
}
