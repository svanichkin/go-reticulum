package vendor

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type PackException struct{ Message string }

func (e *PackException) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

type UnpackException struct{ Message string }

func (e *UnpackException) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

type UnsupportedTypeException struct{ *PackException }
type InsufficientDataException struct{ *UnpackException }
type InvalidStringException struct{ *UnpackException }
type UnsupportedTimestampException struct{ *UnpackException }
type ReservedCodeException struct{ *UnpackException }
type UnhashableKeyException struct{ *UnpackException }
type DuplicateKeyException struct{ *UnpackException }

type KeyNotPrimitiveException = UnhashableKeyException
type KeyDuplicateException = DuplicateKeyException

type InvalidString []byte
type BinaryKey string
type Tuple []any
type TupleKey string

type OrderedMapEntry struct {
	Key   any
	Value any
}

type OrderedMap []OrderedMapEntry

type ExtPackHandler func(any) (Ext, error)
type ExtUnpackHandler func(Ext) (any, error)

type ExtSerializable interface {
	Packb() ([]byte, error)
}

type extClassRegistration struct {
	typ    reflect.Type
	unpack func([]byte) (any, error)
}

type options struct {
	compatibility       bool
	forceFloatPrecision string
	useOrderedMap       bool
	useTuple            bool
	allowInvalidUTF8    bool
	packExtHandlers     map[reflect.Type]ExtPackHandler
	unpackExtHandlers   map[int8]ExtUnpackHandler
}

type Option func(*options)

var Compatibility bool

var (
	extRegistryMu     sync.RWMutex
	extTypeToClass    = map[int8]extClassRegistration{}
	extClassToType    = map[reflect.Type]int8{}
	timeType          = reflect.TypeOf(time.Time{})
	binaryKeyType     = reflect.TypeOf(BinaryKey(""))
	orderedMapType    = reflect.TypeOf(OrderedMap{})
	invalidStringType = reflect.TypeOf(InvalidString{})
	tupleType         = reflect.TypeOf(Tuple{})
)

func defaultOptions() options {
	return options{
		compatibility:       Compatibility,
		forceFloatPrecision: "double",
	}
}

func WithCompatibility(v bool) Option {
	return func(o *options) { o.compatibility = v }
}

func WithForceFloatPrecision(v string) Option {
	return func(o *options) { o.forceFloatPrecision = v }
}

func WithUseOrderedMap(v bool) Option {
	return func(o *options) { o.useOrderedMap = v }
}

func WithUseTuple(v bool) Option {
	return func(o *options) { o.useTuple = v }
}

func WithAllowInvalidUTF8(v bool) Option {
	return func(o *options) { o.allowInvalidUTF8 = v }
}

func WithPackExtHandlers(m map[reflect.Type]ExtPackHandler) Option {
	return func(o *options) { o.packExtHandlers = m }
}

func WithUnpackExtHandlers(m map[int8]ExtUnpackHandler) Option {
	return func(o *options) { o.unpackExtHandlers = m }
}

func resolveOptions(opts []Option) options {
	o := defaultOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	return o
}

func RegisterExtSerializable(extType int8, sample any, unpack func([]byte) (any, error)) error {
	if sample == nil {
		return fmt.Errorf("umsgpack: ext sample cannot be nil")
	}
	if unpack == nil {
		return fmt.Errorf("umsgpack: ext unpack callback cannot be nil")
	}
	rt := reflect.TypeOf(sample)
	extRegistryMu.Lock()
	defer extRegistryMu.Unlock()
	if existing, ok := extTypeToClass[extType]; ok {
		return fmt.Errorf("umsgpack: ext type %d already registered for %s", extType, existing.typ)
	}
	if existing, ok := extClassToType[rt]; ok {
		return fmt.Errorf("umsgpack: type %s already registered for ext type %d", rt, existing)
	}
	extTypeToClass[extType] = extClassRegistration{typ: rt, unpack: unpack}
	extClassToType[rt] = extType
	return nil
}

func ResetExtSerializables() {
	extRegistryMu.Lock()
	defer extRegistryMu.Unlock()
	extTypeToClass = map[int8]extClassRegistration{}
	extClassToType = map[reflect.Type]int8{}
}

func Dumps(v any, opts ...Option) ([]byte, error) {
	return Packb(v, opts...)
}

func Loads(data []byte, opts ...Option) (any, error) {
	var out any
	if err := Unpackb(data, &out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

func Dump(w io.Writer, v any, opts ...Option) error {
	data, err := Packb(v, opts...)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func Load(r io.Reader, opts ...Option) (any, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return Loads(data, opts...)
}

// Packb encodes the provided value using MessagePack.
func Packb(v any, opts ...Option) ([]byte, error) {
	var buf bytes.Buffer
	options := resolveOptions(opts)
	if err := packAny(&buf, v, options); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unpackb decodes data into the provided target pointer.
func Unpackb(data []byte, v any, opts ...Option) error {
	if v == nil {
		return fmt.Errorf("umsgpack: target cannot be nil")
	}
	dest := reflect.ValueOf(v)
	if dest.Kind() != reflect.Pointer || dest.IsNil() {
		return fmt.Errorf("umsgpack: target must be a non-nil pointer")
	}

	options := resolveOptions(opts)
	val, err := unpackAny(bytes.NewReader(data), options)
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

func packAny(w *bytes.Buffer, v any, opts options) error {
	switch x := v.(type) {
	case nil:
		return w.WriteByte(codeNil)
	case bool:
		if x {
			return w.WriteByte(codeTrue)
		}
		return w.WriteByte(codeFalse)
	case string:
		if opts.compatibility {
			return packOldSpecRaw(w, []byte(x))
		}
		return packString(w, x)
	case []byte:
		if opts.compatibility {
			return packOldSpecRaw(w, x)
		}
		return packBytes(w, x)
	case []any:
		return packArray(w, x, opts)
	case Tuple:
		return packArray(w, []any(x), opts)
	case map[string]any:
		return packStringMap(w, x, opts)
	case map[any]any:
		return packAnyMap(w, x, opts)
	case OrderedMap:
		return packOrderedMap(w, x, opts)
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
		return packFloat64(w, float64(x), opts)
	case float64:
		return packFloat64(w, x, opts)
	case Ext:
		return packExt(w, x.Type, x.Data)
	case *Ext:
		if x == nil {
			return w.WriteByte(codeNil)
		}
		return packExt(w, x.Type, x.Data)
	case time.Time:
		return packExtTimestamp(w, x)
	default:
		if ext, ok, err := tryPackRegisteredExt(v); ok || err != nil {
			if err != nil {
				return err
			}
			return packExt(w, ext.Type, ext.Data)
		}
		if ext, ok, err := tryPackExtHandler(v, opts); ok || err != nil {
			if err != nil {
				return err
			}
			return packExt(w, ext.Type, ext.Data)
		}
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Slice:
			n := rv.Len()
			arr := make([]any, 0, n)
			for i := 0; i < n; i++ {
				arr = append(arr, rv.Index(i).Interface())
			}
			return packArray(w, arr, opts)
		case reflect.Array:
			if rv.Type().Elem().Kind() == reflect.Uint8 {
				b := make([]byte, rv.Len())
				for i := 0; i < rv.Len(); i++ {
					b[i] = byte(rv.Index(i).Uint())
				}
				if opts.compatibility {
					return packOldSpecRaw(w, b)
				}
				return packBytes(w, b)
			}
			n := rv.Len()
			arr := make([]any, 0, n)
			for i := 0; i < n; i++ {
				arr = append(arr, rv.Index(i).Interface())
			}
			return packArray(w, arr, opts)
		case reflect.Map:
			iter := rv.MapRange()
			m := make(map[any]any)
			for iter.Next() {
				m[iter.Key().Interface()] = iter.Value().Interface()
			}
			return packAnyMap(w, m, opts)
		}
		return &UnsupportedTypeException{&PackException{Message: fmt.Sprintf("unsupported type: %s", reflect.TypeOf(v))}}
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

func packArray(w *bytes.Buffer, arr []any, opts options) error {
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
		if err := packAny(w, v, opts); err != nil {
			return err
		}
	}
	return nil
}

func packStringMap(w *bytes.Buffer, m map[string]any, opts options) error {
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
		if opts.compatibility {
			if err := packOldSpecRaw(w, []byte(k)); err != nil {
				return err
			}
		} else if err := packString(w, k); err != nil {
			return err
		}
		if err := packAny(w, v, opts); err != nil {
			return err
		}
	}
	return nil
}

func packAnyMap(w *bytes.Buffer, m map[any]any, opts options) error {
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
		if err := packAny(w, k, opts); err != nil {
			return err
		}
		if err := packAny(w, v, opts); err != nil {
			return err
		}
	}
	return nil
}

func packOrderedMap(w *bytes.Buffer, m OrderedMap, opts options) error {
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
	for _, entry := range m {
		if err := packAny(w, entry.Key, opts); err != nil {
			return err
		}
		if err := packAny(w, entry.Value, opts); err != nil {
			return err
		}
	}
	return nil
}

func packOldSpecRaw(w *bytes.Buffer, b []byte) error {
	n := len(b)
	switch {
	case n <= 31:
		if err := w.WriteByte(byte(0xA0 | n)); err != nil {
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

func packFloat64(w *bytes.Buffer, f float64, opts options) error {
	switch opts.forceFloatPrecision {
	case "", "double":
		if err := w.WriteByte(codeFloat64); err != nil {
			return err
		}
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], math.Float64bits(f))
		_, err := w.Write(buf[:])
		return err
	case "single":
		if err := w.WriteByte(codeFloat32); err != nil {
			return err
		}
		var buf [4]byte
		binary.BigEndian.PutUint32(buf[:], math.Float32bits(float32(f)))
		_, err := w.Write(buf[:])
		return err
	default:
		return fmt.Errorf("invalid float precision")
	}
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

func packExtTimestamp(w *bytes.Buffer, t time.Time) error {
	if t.Location() != time.UTC {
		t = t.UTC()
	}
	seconds := t.Unix()
	nanos := t.Nanosecond()
	switch {
	case nanos == 0 && seconds >= 0 && seconds <= math.MaxUint32:
		if err := w.WriteByte(codeFixExt4); err != nil {
			return err
		}
		if err := w.WriteByte(0xFF); err != nil {
			return err
		}
		var buf [4]byte
		binary.BigEndian.PutUint32(buf[:], uint32(seconds))
		_, err := w.Write(buf[:])
		return err
	case seconds >= 0 && seconds <= (1<<34)-1:
		if err := w.WriteByte(codeFixExt8); err != nil {
			return err
		}
		if err := w.WriteByte(0xFF); err != nil {
			return err
		}
		value := (uint64(nanos) << 34) | uint64(seconds)
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], value)
		_, err := w.Write(buf[:])
		return err
	default:
		if err := w.WriteByte(codeExt8); err != nil {
			return err
		}
		if err := w.WriteByte(12); err != nil {
			return err
		}
		if err := w.WriteByte(0xFF); err != nil {
			return err
		}
		var buf [12]byte
		binary.BigEndian.PutUint32(buf[:4], uint32(nanos))
		binary.BigEndian.PutUint64(buf[4:], uint64(seconds))
		_, err := w.Write(buf[:])
		return err
	}
}

func unpackAny(r *bytes.Reader, opts options) (any, error) {
	b, err := r.ReadByte()
	if err != nil {
		return nil, wrapReadErr(err)
	}

	// Positive fixint
	if b <= 0x7F {
		return int64(b), nil
	}
	// FixMap
	if b >= 0x80 && b <= 0x8F {
		return unpackMap(r, int(b&0x0F), opts)
	}
	// FixArray
	if b >= 0x90 && b <= 0x9F {
		return unpackArray(r, int(b&0x0F), opts)
	}
	// FixStr
	if b >= 0xA0 && b <= 0xBF {
		n := int(b & 0x1F)
		buf := make([]byte, n)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, wrapReadErr(err)
		}
		return decodeString(buf, opts)
	}

	// Negative fixint
	if b >= 0xE0 {
		return int64(int8(b)), nil
	}

	switch b {
	case 0xC1:
		return nil, &ReservedCodeException{&UnpackException{Message: "encountered reserved code: 0xc1"}}
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
			return nil, wrapReadErr(err)
		}
		return int64(binary.BigEndian.Uint16(buf[:])), nil
	case codeUint32:
		var buf [4]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, wrapReadErr(err)
		}
		return int64(binary.BigEndian.Uint32(buf[:])), nil
	case codeUint64:
		var buf [8]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, wrapReadErr(err)
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
			return nil, wrapReadErr(err)
		}
		return int64(int16(binary.BigEndian.Uint16(buf[:]))), nil
	case codeInt32:
		var buf [4]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, wrapReadErr(err)
		}
		return int64(int32(binary.BigEndian.Uint32(buf[:]))), nil
	case codeInt64:
		var buf [8]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, wrapReadErr(err)
		}
		return int64(binary.BigEndian.Uint64(buf[:])), nil

	case codeStr8:
		n, err := r.ReadByte()
		if err != nil {
			return nil, wrapReadErr(err)
		}
		buf := make([]byte, int(n))
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, wrapReadErr(err)
		}
		return decodeString(buf, opts)
	case codeStr16:
		var buf [2]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, wrapReadErr(err)
		}
		n := int(binary.BigEndian.Uint16(buf[:]))
		out := make([]byte, n)
		if _, err := io.ReadFull(r, out); err != nil {
			return nil, wrapReadErr(err)
		}
		return decodeString(out, opts)
	case codeStr32:
		var buf [4]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, wrapReadErr(err)
		}
		n := int(binary.BigEndian.Uint32(buf[:]))
		out := make([]byte, n)
		if _, err := io.ReadFull(r, out); err != nil {
			return nil, wrapReadErr(err)
		}
		return decodeString(out, opts)

	case codeBin8:
		n, err := r.ReadByte()
		if err != nil {
			return nil, wrapReadErr(err)
		}
		out := make([]byte, int(n))
		if _, err := io.ReadFull(r, out); err != nil {
			return nil, wrapReadErr(err)
		}
		return out, nil
	case codeBin16:
		var buf [2]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, wrapReadErr(err)
		}
		n := int(binary.BigEndian.Uint16(buf[:]))
		out := make([]byte, n)
		if _, err := io.ReadFull(r, out); err != nil {
			return nil, wrapReadErr(err)
		}
		return out, nil
	case codeBin32:
		var buf [4]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, wrapReadErr(err)
		}
		n := int(binary.BigEndian.Uint32(buf[:]))
		out := make([]byte, n)
		if _, err := io.ReadFull(r, out); err != nil {
			return nil, wrapReadErr(err)
		}
		return out, nil

	case codeExt8:
		n, err := r.ReadByte()
		if err != nil {
			return nil, wrapReadErr(err)
		}
		return unpackExt(r, int(n), opts)
	case codeExt16:
		var buf [2]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, wrapReadErr(err)
		}
		return unpackExt(r, int(binary.BigEndian.Uint16(buf[:])), opts)
	case codeExt32:
		var buf [4]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, wrapReadErr(err)
		}
		return unpackExt(r, int(binary.BigEndian.Uint32(buf[:])), opts)
	case codeFixExt1:
		return unpackExt(r, 1, opts)
	case codeFixExt2:
		return unpackExt(r, 2, opts)
	case codeFixExt4:
		return unpackExt(r, 4, opts)
	case codeFixExt8:
		return unpackExt(r, 8, opts)
	case codeFixExt16:
		return unpackExt(r, 16, opts)

	case codeArray16:
		var buf [2]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, wrapReadErr(err)
		}
		return unpackArray(r, int(binary.BigEndian.Uint16(buf[:])), opts)
	case codeArray32:
		var buf [4]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, wrapReadErr(err)
		}
		return unpackArray(r, int(binary.BigEndian.Uint32(buf[:])), opts)
	case codeMap16:
		var buf [2]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, err
		}
		return unpackMap(r, int(binary.BigEndian.Uint16(buf[:])), opts)
	case codeMap32:
		var buf [4]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, wrapReadErr(err)
		}
		return unpackMap(r, int(binary.BigEndian.Uint32(buf[:])), opts)

	case codeFloat32:
		var buf [4]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, err
		}
		return float64(math.Float32frombits(binary.BigEndian.Uint32(buf[:]))), nil
	case codeFloat64:
		var buf [8]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return nil, wrapReadErr(err)
		}
		return math.Float64frombits(binary.BigEndian.Uint64(buf[:])), nil
	}

	return nil, fmt.Errorf("umsgpack: unsupported code 0x%02x", b)
}

func unpackExt(r *bytes.Reader, n int, opts options) (any, error) {
	if n < 0 {
		return nil, fmt.Errorf("umsgpack: invalid ext length %d", n)
	}
	t, err := r.ReadByte()
	if err != nil {
		return nil, wrapReadErr(err)
	}
	out := make([]byte, n)
	if _, err := io.ReadFull(r, out); err != nil {
		return nil, wrapReadErr(err)
	}
	ext := Ext{Type: int8(t), Data: out}
	if opts.unpackExtHandlers != nil {
		if handler, ok := opts.unpackExtHandlers[ext.Type]; ok && handler != nil {
			return handler(ext)
		}
	}
	if ext.Type == -1 {
		return unpackTimestampExt(out)
	}
	if unpacked, ok, err := tryUnpackRegisteredExt(ext); ok || err != nil {
		return unpacked, err
	}
	return ext, nil
}

func unpackArray(r *bytes.Reader, n int, opts options) (any, error) {
	if n < 0 {
		return nil, fmt.Errorf("umsgpack: invalid array length %d", n)
	}
	out := make([]any, 0, n)
	for i := 0; i < n; i++ {
		v, err := unpackAny(r, opts)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	if opts.useTuple {
		return Tuple(out), nil
	}
	return out, nil
}

func unpackMap(r *bytes.Reader, n int, opts options) (any, error) {
	if n < 0 {
		return nil, fmt.Errorf("umsgpack: invalid map length %d", n)
	}
	out := make(map[any]any, n)
	ordered := make(OrderedMap, 0, n)
	for i := 0; i < n; i++ {
		k, err := unpackAny(r, opts)
		if err != nil {
			return nil, err
		}
		k, err = normalizeMapKey(k)
		if err != nil {
			return nil, err
		}
		if _, exists := out[k]; exists {
			return nil, &DuplicateKeyException{&UnpackException{Message: fmt.Sprintf("encountered duplicate key: %v (%T)", k, k)}}
		}
		v, err := unpackAny(r, opts)
		if err != nil {
			return nil, err
		}
		out[k] = v
		ordered = append(ordered, OrderedMapEntry{Key: k, Value: v})
	}
	if opts.useOrderedMap {
		return ordered, nil
	}
	return out, nil
}

func normalizeMapKey(k any) (any, error) {
	switch x := k.(type) {
	case []byte:
		return BinaryKey(string(x)), nil
	case []any:
		return newTupleKey(x)
	case Tuple:
		return newTupleKey([]any(x))
	case OrderedMap, map[any]any, map[string]any:
		return nil, &UnhashableKeyException{&UnpackException{Message: fmt.Sprintf("encountered unhashable key: %v (%T)", k, k)}}
	default:
		rv := reflect.ValueOf(k)
		if !rv.IsValid() {
			return nil, nil
		}
		if !rv.Type().Comparable() {
			return nil, &UnhashableKeyException{&UnpackException{Message: fmt.Sprintf("encountered unhashable key: %v (%T)", k, k)}}
		}
		return k, nil
	}
}

func newTupleKey(items []any) (TupleKey, error) {
	normalised := make([]any, len(items))
	for i, item := range items {
		v, err := normaliseTupleElement(item)
		if err != nil {
			return "", err
		}
		normalised[i] = v
	}
	packed, err := Packb(normalised, WithAllowInvalidUTF8(true))
	if err != nil {
		return "", err
	}
	return TupleKey(string(packed)), nil
}

func normaliseTupleElement(v any) (any, error) {
	switch x := v.(type) {
	case []any:
		out := make([]any, len(x))
		for i, elem := range x {
			norm, err := normaliseTupleElement(elem)
			if err != nil {
				return nil, err
			}
			out[i] = norm
		}
		return out, nil
	case Tuple:
		return normaliseTupleElement([]any(x))
	case map[any]any, map[string]any, OrderedMap:
		return nil, &UnhashableKeyException{&UnpackException{Message: fmt.Sprintf("encountered unhashable key: %v (%T)", v, v)}}
	default:
		rv := reflect.ValueOf(v)
		if !rv.IsValid() {
			return nil, nil
		}
		if rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() == reflect.Uint8 {
			return v, nil
		}
		if !rv.Type().Comparable() {
			return nil, &UnhashableKeyException{&UnpackException{Message: fmt.Sprintf("encountered unhashable key: %v (%T)", v, v)}}
		}
		return v, nil
	}
}

func decodeString(data []byte, opts options) (any, error) {
	if opts.compatibility {
		return data, nil
	}
	if !utf8.Valid(data) {
		if opts.allowInvalidUTF8 {
			return InvalidString(append([]byte(nil), data...)), nil
		}
		return nil, &InvalidStringException{&UnpackException{Message: "unpacked string is invalid utf-8"}}
	}
	return string(data), nil
}

func wrapReadErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return &InsufficientDataException{&UnpackException{Message: "insufficient data"}}
	}
	return err
}

func unpackTimestampExt(data []byte) (time.Time, error) {
	switch len(data) {
	case 4:
		seconds := int64(binary.BigEndian.Uint32(data))
		return time.Unix(seconds, 0).UTC(), nil
	case 8:
		value := binary.BigEndian.Uint64(data)
		seconds := int64(value & 0x3ffffffff)
		nanos := int64(value >> 34)
		return time.Unix(seconds, nanos).UTC(), nil
	case 12:
		nanos := int64(binary.BigEndian.Uint32(data[:4]))
		seconds := int64(binary.BigEndian.Uint64(data[4:]))
		return time.Unix(seconds, nanos).UTC(), nil
	default:
		return time.Time{}, &UnsupportedTimestampException{&UnpackException{Message: fmt.Sprintf("unsupported timestamp with data length %d", len(data))}}
	}
}

func tryPackExtHandler(v any, opts options) (Ext, bool, error) {
	if opts.packExtHandlers == nil || v == nil {
		return Ext{}, false, nil
	}
	rv := reflect.ValueOf(v)
	rt := rv.Type()
	if handler, ok := opts.packExtHandlers[rt]; ok && handler != nil {
		ext, err := handler(v)
		return ext, true, err
	}
	for t, handler := range opts.packExtHandlers {
		if handler == nil {
			continue
		}
		if rt.AssignableTo(t) || (t.Kind() == reflect.Interface && rt.Implements(t)) {
			ext, err := handler(v)
			return ext, true, err
		}
	}
	return Ext{}, false, nil
}

func tryPackRegisteredExt(v any) (Ext, bool, error) {
	if v == nil {
		return Ext{}, false, nil
	}
	rv := reflect.ValueOf(v)
	rt := rv.Type()

	extRegistryMu.RLock()
	defer extRegistryMu.RUnlock()

	if extType, ok := extClassToType[rt]; ok {
		return packRegisteredExtValue(extType, rv)
	}
	for candidate, extType := range extClassToType {
		if rt.AssignableTo(candidate) || (candidate.Kind() == reflect.Interface && rt.Implements(candidate)) {
			return packRegisteredExtValue(extType, rv)
		}
	}
	return Ext{}, false, nil
}

func packRegisteredExtValue(extType int8, rv reflect.Value) (Ext, bool, error) {
	packer, ok := rv.Interface().(ExtSerializable)
	if !ok && rv.CanAddr() {
		packer, ok = rv.Addr().Interface().(ExtSerializable)
	}
	if !ok {
		return Ext{}, false, fmt.Errorf("umsgpack: registered ext type %d does not implement Packb()", extType)
	}
	data, err := packer.Packb()
	if err != nil {
		return Ext{}, true, err
	}
	return Ext{Type: extType, Data: data}, true, nil
}

func tryUnpackRegisteredExt(ext Ext) (any, bool, error) {
	extRegistryMu.RLock()
	reg, ok := extTypeToClass[ext.Type]
	extRegistryMu.RUnlock()
	if !ok {
		return nil, false, nil
	}
	out, err := reg.unpack(ext.Data)
	return out, true, err
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
		if dest.Type() == timeType && src.Type() == timeType {
			dest.Set(src)
			return nil
		}
		return assignStruct(dest, src)
	case reflect.String:
		if src.Type() == binaryKeyType {
			dest.SetString(string(src.Interface().(BinaryKey)))
			return nil
		}
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
	if src.Type() == orderedMapType {
		newMap := reflect.MakeMap(dest.Type())
		for i := 0; i < src.Len(); i++ {
			entry := src.Index(i)
			dstKey := reflect.New(dest.Type().Key()).Elem()
			if err := assignValue(dstKey, entry.FieldByName("Key")); err != nil {
				return err
			}
			dstVal := reflect.New(dest.Type().Elem()).Elem()
			if err := assignValue(dstVal, entry.FieldByName("Value")); err != nil {
				return err
			}
			newMap.SetMapIndex(dstKey, dstVal)
		}
		dest.Set(newMap)
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
	if src.Type() == invalidStringType {
		if dest.Type().Elem().Kind() == reflect.Uint8 {
			raw := []byte(src.Interface().(InvalidString))
			newSlice := reflect.MakeSlice(dest.Type(), len(raw), len(raw))
			reflect.Copy(newSlice, reflect.ValueOf(raw))
			dest.Set(newSlice)
			return nil
		}
	}
	if src.Type() == tupleType {
		src = reflect.ValueOf([]any(src.Interface().(Tuple)))
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
	if key.Type() == binaryKeyType {
		return strings.EqualFold(string(key.Interface().(BinaryKey)), target)
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
