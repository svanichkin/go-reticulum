package vendor

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"reflect"
	"strings"
)

// Packb encodes the provided value using MessagePack.
func Packb(v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := packAny(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unpackb decodes data into the provided target pointer.
func Unpackb(data []byte, v any) error {
	if v == nil {
		return fmt.Errorf("umsgpack: target cannot be nil")
	}
	dest := reflect.ValueOf(v)
	if dest.Kind() != reflect.Pointer || dest.IsNil() {
		return fmt.Errorf("umsgpack: target must be a non-nil pointer")
	}

	val, err := unpackAny(bytes.NewReader(data))
	if err != nil {
		return err
	}
	return assignValue(dest, reflect.ValueOf(val))
}

// ---- Minimal MessagePack decoder -------------------------------------------
//
// This repo historically used a very old msgpack implementation that panics on
// newer codes such as STR8 (0xD9). We implement a small decoder for the subset
// of types Reticulum uses (maps, arrays, ints, bool, nil, bytes/strings).

const (
	codeNil   = 0xC0
	codeFalse = 0xC2
	codeTrue  = 0xC3

	codeBin8  = 0xC4
	codeBin16 = 0xC5
	codeBin32 = 0xC6

	codeExt8  = 0xC7
	codeExt16 = 0xC8
	codeExt32 = 0xC9

	codeFloat32 = 0xCA
	codeFloat64 = 0xCB

	codeUint8  = 0xCC
	codeUint16 = 0xCD
	codeUint32 = 0xCE
	codeUint64 = 0xCF
	codeInt8   = 0xD0
	codeInt16  = 0xD1
	codeInt32  = 0xD2
	codeInt64  = 0xD3

	codeFixExt1  = 0xD4
	codeFixExt2  = 0xD5
	codeFixExt4  = 0xD6
	codeFixExt8  = 0xD7
	codeFixExt16 = 0xD8

	codeStr8  = 0xD9
	codeStr16 = 0xDA
	codeStr32 = 0xDB

	codeArray16 = 0xDC
	codeArray32 = 0xDD
	codeMap16   = 0xDE
	codeMap32   = 0xDF
)

// ---- Minimal MessagePack encoder -------------------------------------------
//
// We implement encoding for the subset of types Reticulum uses. The previously
// vendored msgpack encoder truncated "raw"/string lengths to uint8 for values
// >255 bytes, breaking resource advertisements (hashmap segments >255 bytes).

// Ext is an application-defined MessagePack extension object, compatible with
// u-msgpack-python's Ext (type + opaque data).
type Ext struct {
	Type int8
	Data []byte
}

func packAny(w *bytes.Buffer, v any) error {
	switch x := v.(type) {
	case nil:
		return w.WriteByte(codeNil)
	case bool:
		if x {
			return w.WriteByte(codeTrue)
		}
		return w.WriteByte(codeFalse)
	case string:
		return packString(w, x)
	case []byte:
		return packBytes(w, x)
	case []any:
		return packArray(w, x)
	case map[string]any:
		return packStringMap(w, x)
	case map[any]any:
		return packAnyMap(w, x)
	case int:
		return packInt64(w, int64(x))
	case int8:
		return packInt64(w, int64(x))
	case int16:
		return packInt64(w, int64(x))
	case int32:
		return packInt64(w, int64(x))
	case int64:
		return packInt64(w, x)
	case uint:
		return packUint64(w, uint64(x))
	case uint8:
		return packUint64(w, uint64(x))
	case uint16:
		return packUint64(w, uint64(x))
	case uint32:
		return packUint64(w, uint64(x))
	case uint64:
		return packUint64(w, x)
	case float32:
		return packFloat64(w, float64(x))
	case float64:
		return packFloat64(w, x)
	case Ext:
		return packExt(w, x.Type, x.Data)
	case *Ext:
		if x == nil {
			return w.WriteByte(codeNil)
		}
		return packExt(w, x.Type, x.Data)
	default:
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Slice:
			n := rv.Len()
			arr := make([]any, 0, n)
			for i := 0; i < n; i++ {
				arr = append(arr, rv.Index(i).Interface())
			}
			return packArray(w, arr)
		case reflect.Array:
			if rv.Type().Elem().Kind() == reflect.Uint8 {
				b := make([]byte, rv.Len())
				for i := 0; i < rv.Len(); i++ {
					b[i] = byte(rv.Index(i).Uint())
				}
				return packBytes(w, b)
			}
			n := rv.Len()
			arr := make([]any, 0, n)
			for i := 0; i < n; i++ {
				arr = append(arr, rv.Index(i).Interface())
			}
			return packArray(w, arr)
		case reflect.Map:
			iter := rv.MapRange()
			m := make(map[any]any)
			for iter.Next() {
				m[iter.Key().Interface()] = iter.Value().Interface()
			}
			return packAnyMap(w, m)
		}
		return fmt.Errorf("umsgpack: unsupported type %T", v)
	}
}

func packString(w *bytes.Buffer, s string) error {
	b := []byte(s)
	n := len(b)
	switch {
	case n <= 31:
		if err := w.WriteByte(byte(0xA0 | n)); err != nil {
			return err
		}
	case n <= 0xFF:
		if err := w.WriteByte(codeStr8); err != nil {
			return err
		}
		if err := w.WriteByte(byte(n)); err != nil {
			return err
		}
	case n <= 0xFFFF:
		if err := w.WriteByte(codeStr16); err != nil {
			return err
		}
		var buf [2]byte
		binary.BigEndian.PutUint16(buf[:], uint16(n))
		if _, err := w.Write(buf[:]); err != nil {
			return err
		}
	default:
		if err := w.WriteByte(codeStr32); err != nil {
			return err
		}
		var buf [4]byte
		binary.BigEndian.PutUint32(buf[:], uint32(n))
		if _, err := w.Write(buf[:]); err != nil {
			return err
		}
	}
	_, err := w.Write(b)
	return err
}

func packBytes(w *bytes.Buffer, b []byte) error {
	n := len(b)
	switch {
	case n <= 0xFF:
		if err := w.WriteByte(codeBin8); err != nil {
			return err
		}
		if err := w.WriteByte(byte(n)); err != nil {
			return err
		}
	case n <= 0xFFFF:
		if err := w.WriteByte(codeBin16); err != nil {
			return err
		}
		var buf [2]byte
		binary.BigEndian.PutUint16(buf[:], uint16(n))
		if _, err := w.Write(buf[:]); err != nil {
			return err
		}
	default:
		if err := w.WriteByte(codeBin32); err != nil {
			return err
		}
		var buf [4]byte
		binary.BigEndian.PutUint32(buf[:], uint32(n))
		if _, err := w.Write(buf[:]); err != nil {
			return err
		}
	}
	_, err := w.Write(b)
	return err
}

func packArray(w *bytes.Buffer, arr []any) error {
	n := len(arr)
	switch {
	case n <= 15:
		if err := w.WriteByte(byte(0x90 | n)); err != nil {
			return err
		}
	case n <= 0xFFFF:
		if err := w.WriteByte(codeArray16); err != nil {
			return err
		}
		var buf [2]byte
		binary.BigEndian.PutUint16(buf[:], uint16(n))
		if _, err := w.Write(buf[:]); err != nil {
			return err
		}
	default:
		if err := w.WriteByte(codeArray32); err != nil {
			return err
		}
		var buf [4]byte
		binary.BigEndian.PutUint32(buf[:], uint32(n))
		if _, err := w.Write(buf[:]); err != nil {
			return err
		}
	}
	for _, v := range arr {
		if err := packAny(w, v); err != nil {
			return err
		}
	}
	return nil
}

func packStringMap(w *bytes.Buffer, m map[string]any) error {
	n := len(m)
	switch {
	case n <= 15:
		if err := w.WriteByte(byte(0x80 | n)); err != nil {
			return err
		}
	case n <= 0xFFFF:
		if err := w.WriteByte(codeMap16); err != nil {
			return err
		}
		var buf [2]byte
		binary.BigEndian.PutUint16(buf[:], uint16(n))
		if _, err := w.Write(buf[:]); err != nil {
			return err
		}
	default:
		if err := w.WriteByte(codeMap32); err != nil {
			return err
		}
		var buf [4]byte
		binary.BigEndian.PutUint32(buf[:], uint32(n))
		if _, err := w.Write(buf[:]); err != nil {
			return err
		}
	}
	for k, v := range m {
		if err := packString(w, k); err != nil {
			return err
		}
		if err := packAny(w, v); err != nil {
			return err
		}
	}
	return nil
}

func packAnyMap(w *bytes.Buffer, m map[any]any) error {
	n := len(m)
	switch {
	case n <= 15:
		if err := w.WriteByte(byte(0x80 | n)); err != nil {
			return err
		}
	case n <= 0xFFFF:
		if err := w.WriteByte(codeMap16); err != nil {
			return err
		}
		var buf [2]byte
		binary.BigEndian.PutUint16(buf[:], uint16(n))
		if _, err := w.Write(buf[:]); err != nil {
			return err
		}
	default:
		if err := w.WriteByte(codeMap32); err != nil {
			return err
		}
		var buf [4]byte
		binary.BigEndian.PutUint32(buf[:], uint32(n))
		if _, err := w.Write(buf[:]); err != nil {
			return err
		}
	}
	for k, v := range m {
		if err := packAny(w, k); err != nil {
			return err
		}
		if err := packAny(w, v); err != nil {
			return err
		}
	}
	return nil
}

func packInt64(w *bytes.Buffer, n int64) error {
	// Prefer fixints when possible.
	if n >= 0 && n <= 127 {
		return w.WriteByte(byte(n))
	}
	if n < 0 && n >= -32 {
		return w.WriteByte(byte(int8(n)))
	}
	switch {
	case n >= math.MinInt8 && n <= math.MaxInt8:
		if err := w.WriteByte(codeInt8); err != nil {
			return err
		}
		return w.WriteByte(byte(int8(n)))
	case n >= math.MinInt16 && n <= math.MaxInt16:
		if err := w.WriteByte(codeInt16); err != nil {
			return err
		}
		var buf [2]byte
		binary.BigEndian.PutUint16(buf[:], uint16(int16(n)))
		_, err := w.Write(buf[:])
		return err
	case n >= math.MinInt32 && n <= math.MaxInt32:
		if err := w.WriteByte(codeInt32); err != nil {
			return err
		}
		var buf [4]byte
		binary.BigEndian.PutUint32(buf[:], uint32(int32(n)))
		_, err := w.Write(buf[:])
		return err
	default:
		if err := w.WriteByte(codeInt64); err != nil {
			return err
		}
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], uint64(n))
		_, err := w.Write(buf[:])
		return err
	}
}

func packUint64(w *bytes.Buffer, n uint64) error {
	if n <= 127 {
		return w.WriteByte(byte(n))
	}
	switch {
	case n <= math.MaxUint8:
		if err := w.WriteByte(codeUint8); err != nil {
			return err
		}
		return w.WriteByte(byte(n))
	case n <= math.MaxUint16:
		if err := w.WriteByte(codeUint16); err != nil {
			return err
		}
		var buf [2]byte
		binary.BigEndian.PutUint16(buf[:], uint16(n))
		_, err := w.Write(buf[:])
		return err
	case n <= math.MaxUint32:
		if err := w.WriteByte(codeUint32); err != nil {
			return err
		}
		var buf [4]byte
		binary.BigEndian.PutUint32(buf[:], uint32(n))
		_, err := w.Write(buf[:])
		return err
	default:
		if err := w.WriteByte(codeUint64); err != nil {
			return err
		}
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], n)
		_, err := w.Write(buf[:])
		return err
	}
}

func packFloat64(w *bytes.Buffer, f float64) error {
	if err := w.WriteByte(codeFloat64); err != nil {
		return err
	}
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], math.Float64bits(f))
	_, err := w.Write(buf[:])
	return err
}

func packExt(w *bytes.Buffer, typ int8, data []byte) error {
	n := len(data)
	switch n {
	case 1:
		if err := w.WriteByte(codeFixExt1); err != nil {
			return err
		}
	case 2:
		if err := w.WriteByte(codeFixExt2); err != nil {
			return err
		}
	case 4:
		if err := w.WriteByte(codeFixExt4); err != nil {
			return err
		}
	case 8:
		if err := w.WriteByte(codeFixExt8); err != nil {
			return err
		}
	case 16:
		if err := w.WriteByte(codeFixExt16); err != nil {
			return err
		}
	default:
		switch {
		case n <= 0xFF:
			if err := w.WriteByte(codeExt8); err != nil {
				return err
			}
			if err := w.WriteByte(byte(n)); err != nil {
				return err
			}
		case n <= 0xFFFF:
			if err := w.WriteByte(codeExt16); err != nil {
				return err
			}
			var buf [2]byte
			binary.BigEndian.PutUint16(buf[:], uint16(n))
			if _, err := w.Write(buf[:]); err != nil {
				return err
			}
		default:
			if err := w.WriteByte(codeExt32); err != nil {
				return err
			}
			var buf [4]byte
			binary.BigEndian.PutUint32(buf[:], uint32(n))
			if _, err := w.Write(buf[:]); err != nil {
				return err
			}
		}
	}

	if err := w.WriteByte(byte(typ)); err != nil {
		return err
	}
	_, err := w.Write(data)
	return err
}

func unpackAny(r *bytes.Reader) (any, error) {
	b, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	// Positive fixint
	if b <= 0x7F {
		return int64(b), nil
	}
	// FixMap
	if b >= 0x80 && b <= 0x8F {
		return unpackMap(r, int(b&0x0F))
	}
	// FixArray
	if b >= 0x90 && b <= 0x9F {
		return unpackArray(r, int(b&0x0F))
	}
	// FixStr
	if b >= 0xA0 && b <= 0xBF {
		n := int(b & 0x1F)
		buf := make([]byte, n)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		return string(buf), nil
	}

	// Negative fixint
	if b >= 0xE0 {
		return int64(int8(b)), nil
	}

	switch b {
	case codeNil:
		return nil, nil
	case codeFalse:
		return false, nil
	case codeTrue:
		return true, nil

	case codeUint8:
		v, err := r.ReadByte()
		return int64(v), err
	case codeUint16:
		var buf [2]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, err
		}
		return int64(binary.BigEndian.Uint16(buf[:])), nil
	case codeUint32:
		var buf [4]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, err
		}
		return int64(binary.BigEndian.Uint32(buf[:])), nil
	case codeUint64:
		var buf [8]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, err
		}
		u := binary.BigEndian.Uint64(buf[:])
		// Best-effort: store as int64 when possible, otherwise uint64.
		if u <= uint64(^uint64(0)>>1) {
			return int64(u), nil
		}
		return u, nil

	case codeInt8:
		v, err := r.ReadByte()
		return int64(int8(v)), err
	case codeInt16:
		var buf [2]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, err
		}
		return int64(int16(binary.BigEndian.Uint16(buf[:]))), nil
	case codeInt32:
		var buf [4]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, err
		}
		return int64(int32(binary.BigEndian.Uint32(buf[:]))), nil
	case codeInt64:
		var buf [8]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, err
		}
		return int64(binary.BigEndian.Uint64(buf[:])), nil

	case codeStr8:
		n, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		buf := make([]byte, int(n))
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		return string(buf), nil
	case codeStr16:
		var buf [2]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, err
		}
		n := int(binary.BigEndian.Uint16(buf[:]))
		out := make([]byte, n)
		if _, err := io.ReadFull(r, out); err != nil {
			return nil, err
		}
		return string(out), nil
	case codeStr32:
		var buf [4]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, err
		}
		n := int(binary.BigEndian.Uint32(buf[:]))
		out := make([]byte, n)
		if _, err := io.ReadFull(r, out); err != nil {
			return nil, err
		}
		return string(out), nil

	case codeBin8:
		n, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		out := make([]byte, int(n))
		if _, err := io.ReadFull(r, out); err != nil {
			return nil, err
		}
		return out, nil
	case codeBin16:
		var buf [2]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, err
		}
		n := int(binary.BigEndian.Uint16(buf[:]))
		out := make([]byte, n)
		if _, err := io.ReadFull(r, out); err != nil {
			return nil, err
		}
		return out, nil
	case codeBin32:
		var buf [4]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, err
		}
		n := int(binary.BigEndian.Uint32(buf[:]))
		out := make([]byte, n)
		if _, err := io.ReadFull(r, out); err != nil {
			return nil, err
		}
		return out, nil

	case codeExt8:
		n, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		return unpackExt(r, int(n))
	case codeExt16:
		var buf [2]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, err
		}
		return unpackExt(r, int(binary.BigEndian.Uint16(buf[:])))
	case codeExt32:
		var buf [4]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, err
		}
		return unpackExt(r, int(binary.BigEndian.Uint32(buf[:])))
	case codeFixExt1:
		return unpackExt(r, 1)
	case codeFixExt2:
		return unpackExt(r, 2)
	case codeFixExt4:
		return unpackExt(r, 4)
	case codeFixExt8:
		return unpackExt(r, 8)
	case codeFixExt16:
		return unpackExt(r, 16)

	case codeArray16:
		var buf [2]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, err
		}
		return unpackArray(r, int(binary.BigEndian.Uint16(buf[:])))
	case codeArray32:
		var buf [4]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, err
		}
		return unpackArray(r, int(binary.BigEndian.Uint32(buf[:])))
	case codeMap16:
		var buf [2]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, err
		}
		return unpackMap(r, int(binary.BigEndian.Uint16(buf[:])))
	case codeMap32:
		var buf [4]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, err
		}
		return unpackMap(r, int(binary.BigEndian.Uint32(buf[:])))

	case codeFloat32:
		var buf [4]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, err
		}
		return float64(math.Float32frombits(binary.BigEndian.Uint32(buf[:]))), nil
	case codeFloat64:
		var buf [8]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, err
		}
		return math.Float64frombits(binary.BigEndian.Uint64(buf[:])), nil
	}

	return nil, fmt.Errorf("umsgpack: unsupported code 0x%02x", b)
}

func unpackExt(r *bytes.Reader, n int) (any, error) {
	if n < 0 {
		return nil, fmt.Errorf("umsgpack: invalid ext length %d", n)
	}
	t, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	out := make([]byte, n)
	if _, err := io.ReadFull(r, out); err != nil {
		return nil, err
	}
	return Ext{Type: int8(t), Data: out}, nil
}

func unpackArray(r *bytes.Reader, n int) ([]any, error) {
	if n < 0 {
		return nil, fmt.Errorf("umsgpack: invalid array length %d", n)
	}
	out := make([]any, 0, n)
	for i := 0; i < n; i++ {
		v, err := unpackAny(r)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func unpackMap(r *bytes.Reader, n int) (map[any]any, error) {
	if n < 0 {
		return nil, fmt.Errorf("umsgpack: invalid map length %d", n)
	}
	out := make(map[any]any, n)
	for i := 0; i < n; i++ {
		k, err := unpackAny(r)
		if err != nil {
			return nil, err
		}
		v, err := unpackAny(r)
		if err != nil {
			return nil, err
		}
		// Go map keys must be comparable. msgpack allows bin keys, which decode as []byte.
		// Convert []byte keys to string for safe storage in map[any]any.
		if kb, ok := k.([]byte); ok {
			k = string(kb)
		}
		out[k] = v
	}
	return out, nil
}

func assignValue(dest, src reflect.Value) error {
	if dest.Kind() == reflect.Pointer {
		if dest.IsNil() {
			dest.Set(reflect.New(dest.Type().Elem()))
		}
		return assignValue(dest.Elem(), src)
	}

	src = indirectValue(src)
	if !src.IsValid() {
		dest.Set(reflect.Zero(dest.Type()))
		return nil
	}

	if src.Type().AssignableTo(dest.Type()) {
		dest.Set(src)
		return nil
	}
	if src.Type().ConvertibleTo(dest.Type()) {
		dest.Set(src.Convert(dest.Type()))
		return nil
	}

	switch dest.Kind() {
	case reflect.Interface:
		dest.Set(src)
		return nil
	case reflect.Map:
		return assignMap(dest, src)
	case reflect.Slice:
		return assignSlice(dest, src)
	case reflect.Struct:
		return assignStruct(dest, src)
	case reflect.String:
		if src.Kind() == reflect.Slice && src.Type().Elem().Kind() == reflect.Uint8 {
			dest.SetString(string(src.Bytes()))
			return nil
		}
	}

	return fmt.Errorf("umsgpack: cannot assign %s to %s", src.Type(), dest.Type())
}

func assignMap(dest, src reflect.Value) error {
	src = indirectValue(src)
	if !src.IsValid() {
		dest.Set(reflect.Zero(dest.Type()))
		return nil
	}
	if src.Kind() != reflect.Map {
		return fmt.Errorf("umsgpack: expected map but got %s", src.Kind())
	}

	newMap := reflect.MakeMap(dest.Type())
	for _, key := range src.MapKeys() {
		dstKey := reflect.New(dest.Type().Key()).Elem()
		if err := assignValue(dstKey, key); err != nil {
			return err
		}
		dstVal := reflect.New(dest.Type().Elem()).Elem()
		if err := assignValue(dstVal, src.MapIndex(key)); err != nil {
			return err
		}
		newMap.SetMapIndex(dstKey, dstVal)
	}
	dest.Set(newMap)
	return nil
}

func assignSlice(dest, src reflect.Value) error {
	src = indirectValue(src)
	if !src.IsValid() {
		dest.Set(reflect.Zero(dest.Type()))
		return nil
	}
	if src.Kind() != reflect.Slice && src.Kind() != reflect.Array {
		return fmt.Errorf("umsgpack: expected slice/array but got %s", src.Kind())
	}

	newSlice := reflect.MakeSlice(dest.Type(), src.Len(), src.Len())
	for i := 0; i < src.Len(); i++ {
		if err := assignValue(newSlice.Index(i), src.Index(i)); err != nil {
			return err
		}
	}
	dest.Set(newSlice)
	return nil
}

func assignStruct(dest, src reflect.Value) error {
	src = indirectValue(src)
	if !src.IsValid() {
		dest.Set(reflect.Zero(dest.Type()))
		return nil
	}
	if src.Kind() != reflect.Map {
		return fmt.Errorf("umsgpack: expected map for struct but got %s", src.Kind())
	}

	for i := 0; i < dest.NumField(); i++ {
		field := dest.Field(i)
		if !field.CanSet() {
			continue
		}
		fieldType := dest.Type().Field(i)
		tag := fieldType.Tag.Get("msgpack")
		key := fieldType.Name
		if tag != "" {
			key = strings.Split(tag, ",")[0]
		}
		if key == "-" {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			key = fieldType.Name
		}
		value, ok := findMapValue(src, key)
		if !ok {
			continue
		}
		if err := assignValue(field, value); err != nil {
			return err
		}
	}
	return nil
}

func findMapValue(src reflect.Value, key string) (reflect.Value, bool) {
	for _, k := range src.MapKeys() {
		if matchesMapKey(k, key) {
			return src.MapIndex(k), true
		}
	}
	return reflect.Value{}, false
}

func matchesMapKey(key reflect.Value, target string) bool {
	key = indirectValue(key)
	if !key.IsValid() {
		return false
	}
	switch key.Kind() {
	case reflect.String:
		return strings.EqualFold(key.String(), target)
	case reflect.Slice:
		if key.Type().Elem().Kind() == reflect.Uint8 {
			return strings.EqualFold(string(key.Bytes()), target)
		}
	}
	return false
}

func indirectValue(v reflect.Value) reflect.Value {
	for v.IsValid() && (v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer) {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}
