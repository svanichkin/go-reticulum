package vendor

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
	"time"
)

type customSerializable struct {
	Value string
}

func (c customSerializable) Packb() ([]byte, error) {
	return []byte(c.Value), nil
}

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

	// We can't construct a Go map with []byte keys (not comparable), so craft
	// the msgpack payload directly:
	//   fixmap(1) + bin8("k") + fixint(1)
	b := []byte{0x81, codeBin8, 0x01, 'k', 0x01}

	var out map[any]any
	if err := Unpackb(b, &out); err != nil {
		t.Fatalf("Unpackb: %v", err)
	}
	if v, ok := out[BinaryKey("k")]; !ok || v.(int64) != 1 {
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

func TestUMsgpack_CompatibilityMode_RoundTripsRawAsBytes(t *testing.T) {
	t.Parallel()

	b, err := Packb("hello", WithCompatibility(true))
	if err != nil {
		t.Fatalf("Packb: %v", err)
	}
	if got := firstByte(b); got != 0xA5 {
		t.Fatalf("expected fixraw/fixstr prefix, got %02x", got)
	}

	var out any
	if err := Unpackb(b, &out, WithCompatibility(true)); err != nil {
		t.Fatalf("Unpackb: %v", err)
	}
	raw, ok := out.([]byte)
	if !ok || !bytes.Equal(raw, []byte("hello")) {
		t.Fatalf("unexpected compatibility decode: %#v", out)
	}
}

func TestUMsgpack_Unpackb_InvalidUTF8_DefaultErrors(t *testing.T) {
	t.Parallel()

	err := Unpackb([]byte{codeStr8, 0x02, 0xC3, 0x28}, new(any))
	var invalid *InvalidStringException
	if !errors.As(err, &invalid) {
		t.Fatalf("expected InvalidStringException, got %v", err)
	}
}

func TestUMsgpack_Unpackb_InvalidUTF8_AllowReturnsInvalidString(t *testing.T) {
	t.Parallel()

	var out any
	if err := Unpackb([]byte{codeStr8, 0x02, 0xC3, 0x28}, &out, WithAllowInvalidUTF8(true)); err != nil {
		t.Fatalf("Unpackb: %v", err)
	}
	s, ok := out.(InvalidString)
	if !ok || !bytes.Equal([]byte(s), []byte{0xC3, 0x28}) {
		t.Fatalf("unexpected invalid string payload: %#v", out)
	}
}

func TestUMsgpack_Unpackb_ReservedCode(t *testing.T) {
	t.Parallel()

	err := Unpackb([]byte{0xC1}, new(any))
	var reserved *ReservedCodeException
	if !errors.As(err, &reserved) {
		t.Fatalf("expected ReservedCodeException, got %v", err)
	}
}

func TestUMsgpack_Unpackb_DuplicateKeyErrors(t *testing.T) {
	t.Parallel()

	err := Unpackb([]byte{0x82, 0xA1, 'a', 0x01, 0xA1, 'a', 0x02}, new(any))
	var dup *DuplicateKeyException
	if !errors.As(err, &dup) {
		t.Fatalf("expected DuplicateKeyException, got %v", err)
	}
}

func TestUMsgpack_Unpackb_ListKeyBecomesTupleKey(t *testing.T) {
	t.Parallel()

	// { [1, 2]: 3 }
	b := []byte{0x81, 0x92, 0x01, 0x02, 0x03}
	var out map[any]any
	if err := Unpackb(b, &out); err != nil {
		t.Fatalf("Unpackb: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("unexpected map len: %#v", out)
	}
	for key, value := range out {
		if _, ok := key.(TupleKey); !ok {
			t.Fatalf("expected TupleKey, got %T", key)
		}
		if value.(int64) != 3 {
			t.Fatalf("unexpected value: %#v", value)
		}
	}
}

func TestUMsgpack_Unpackb_UseTuple(t *testing.T) {
	t.Parallel()

	var out any
	if err := Unpackb([]byte{0x92, 0x01, 0x02}, &out, WithUseTuple(true)); err != nil {
		t.Fatalf("Unpackb: %v", err)
	}
	tuple, ok := out.(Tuple)
	if !ok || len(tuple) != 2 || tuple[0].(int64) != 1 || tuple[1].(int64) != 2 {
		t.Fatalf("unexpected tuple: %#v", out)
	}
}

func TestUMsgpack_Unpackb_UseOrderedMap(t *testing.T) {
	t.Parallel()

	var out any
	if err := Unpackb([]byte{0x82, 0xA1, 'b', 0x01, 0xA1, 'a', 0x02}, &out, WithUseOrderedMap(true)); err != nil {
		t.Fatalf("Unpackb: %v", err)
	}
	ordered, ok := out.(OrderedMap)
	if !ok || len(ordered) != 2 {
		t.Fatalf("unexpected ordered map: %#v", out)
	}
	if ordered[0].Key != "b" || ordered[1].Key != "a" {
		t.Fatalf("unexpected key order: %#v", ordered)
	}
}

func TestUMsgpack_Packb_ForceFloatPrecisionSingle(t *testing.T) {
	t.Parallel()

	b, err := Packb(1.25, WithForceFloatPrecision("single"))
	if err != nil {
		t.Fatalf("Packb: %v", err)
	}
	if got := firstByte(b); got != codeFloat32 {
		t.Fatalf("expected float32 prefix, got %02x", got)
	}
}

func TestUMsgpack_Packb_InvalidFloatPrecision(t *testing.T) {
	t.Parallel()

	err := func() error {
		_, err := Packb(1.25, WithForceFloatPrecision("weird"))
		return err
	}()
	if err == nil || err.Error() != "invalid float precision" {
		t.Fatalf("expected invalid float precision error, got %v", err)
	}
}

func TestUMsgpack_TimestampExt_RoundTrip(t *testing.T) {
	t.Parallel()

	in := time.Unix(1700000000, 123456000).UTC()
	b, err := Packb(in)
	if err != nil {
		t.Fatalf("Packb: %v", err)
	}

	var out time.Time
	if err := Unpackb(b, &out); err != nil {
		t.Fatalf("Unpackb: %v", err)
	}
	if !out.Equal(in) {
		t.Fatalf("unexpected time: got %s want %s", out, in)
	}
}

func TestUMsgpack_Unpackb_TimestampInvalidLength(t *testing.T) {
	t.Parallel()

	err := Unpackb([]byte{codeExt8, 0x03, 0xFF, 0x01, 0x02, 0x03}, new(any))
	var tsErr *UnsupportedTimestampException
	if !errors.As(err, &tsErr) {
		t.Fatalf("expected UnsupportedTimestampException, got %v", err)
	}
}

func TestUMsgpack_RegisteredExtSerializable_RoundTrip(t *testing.T) {
	t.Cleanup(ResetExtSerializables)

	if err := RegisterExtSerializable(7, customSerializable{}, func(data []byte) (any, error) {
		return customSerializable{Value: string(data)}, nil
	}); err != nil {
		t.Fatalf("RegisterExtSerializable: %v", err)
	}

	in := customSerializable{Value: "hello"}
	b, err := Packb(in)
	if err != nil {
		t.Fatalf("Packb: %v", err)
	}
	var out any
	if err := Unpackb(b, &out); err != nil {
		t.Fatalf("Unpackb: %v", err)
	}
	got, ok := out.(customSerializable)
	if !ok || got.Value != "hello" {
		t.Fatalf("unexpected custom ext object: %#v", out)
	}
}

func TestUMsgpack_DumpsLoadsAndIOWrappers(t *testing.T) {
	t.Parallel()

	payload := map[string]any{"a": int64(1)}
	data, err := Dumps(payload)
	if err != nil {
		t.Fatalf("Dumps: %v", err)
	}
	var buf bytes.Buffer
	if err := Dump(&buf, payload); err != nil {
		t.Fatalf("Dump: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), data) {
		t.Fatalf("Dump mismatch")
	}
	loaded, err := Load(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	loaded2, err := Loads(data)
	if err != nil {
		t.Fatalf("Loads: %v", err)
	}
	if !reflect.DeepEqual(loaded, loaded2) {
		t.Fatalf("load/loads mismatch: %#v %#v", loaded, loaded2)
	}
}

func TestUMsgpack_Unpackb_InsufficientDataException(t *testing.T) {
	t.Parallel()

	err := Unpackb([]byte{codeStr8, 0x03, 'a'}, new(any))
	var insufficient *InsufficientDataException
	if !errors.As(err, &insufficient) {
		t.Fatalf("expected InsufficientDataException, got %v", err)
	}
}

func firstByte(b []byte) byte {
	if len(b) == 0 {
		return 0
	}
	return b[0]
}
