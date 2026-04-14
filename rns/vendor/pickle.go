package vendor

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"reflect"

	mpickle "github.com/MacIt/pickle"
	gpickle "github.com/nlpodyssey/gopickle/pickle"
	gtypes "github.com/nlpodyssey/gopickle/types"
)

// PickleDumps serializes a Go value to Python pickle protocol 4 format.
func PickleDumps(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := mpickle.NewEncoderWithConfig(&buf, &mpickle.EncoderConfig{Protocol: 4})
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// PickleLoads deserializes Python pickle data to Go values.
//
// The returned value is normalized to stay compatible with the rest of the
// codebase:
//   - dict -> map[string]any
//   - list/tuple -> []any
//   - bytes -> []byte
//   - pickle.None -> nil
//   - integers -> int when representable, otherwise int64
func PickleLoads(data []byte) (any, error) {
	dec := gpickle.NewUnpickler(bytes.NewReader(data))
	dec.FindClass = func(module, name string) (interface{}, error) {
		if (module == "builtins" || module == "__builtin__") && name == "bytearray" {
			return bytearrayCallable{}, nil
		}
		return nil, nil
	}
	v, err := dec.Load()
	if err != nil {
		return nil, err
	}
	return normalizePickleValue(v), nil
}

// PickleAssign assigns a pickle-decoded value to the target pointed to by v.
// v must be a non-nil pointer.
func PickleAssign(result any, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("pickle: target must be a non-nil pointer")
	}
	target := rv.Elem()

	if result == nil {
		target.Set(reflect.Zero(target.Type()))
		return nil
	}

	resultRV := reflect.ValueOf(result)
	resultType := resultRV.Type()
	targetType := target.Type()

	if resultType.AssignableTo(targetType) {
		target.Set(resultRV)
		return nil
	}

	switch targetType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch r := result.(type) {
		case int:
			target.SetInt(int64(r))
		case int64:
			target.SetInt(r)
		case float64:
			target.SetInt(int64(r))
		case int32:
			target.SetInt(int64(r))
		default:
			return fmt.Errorf("pickle: cannot assign %T to %s", result, targetType)
		}

	case reflect.Float32, reflect.Float64:
		switch r := result.(type) {
		case float64:
			target.SetFloat(r)
		case int:
			target.SetFloat(float64(r))
		case int64:
			target.SetFloat(float64(r))
		default:
			return fmt.Errorf("pickle: cannot assign %T to %s", result, targetType)
		}

	case reflect.Bool:
		switch r := result.(type) {
		case bool:
			target.SetBool(r)
		default:
			return fmt.Errorf("pickle: cannot assign %T to %s", result, targetType)
		}

	case reflect.String:
		switch r := result.(type) {
		case string:
			target.SetString(r)
		default:
			return fmt.Errorf("pickle: cannot assign %T to %s", result, targetType)
		}

	case reflect.Slice:
		if targetType.Elem().Kind() == reflect.Uint8 {
			if b, ok := result.([]byte); ok {
				target.SetBytes(b)
				return nil
			}
			if s, ok := result.(string); ok {
				target.SetBytes([]byte(s))
				return nil
			}
		}
		if arr, ok := result.([]any); ok {
			slice := reflect.MakeSlice(targetType, len(arr), len(arr))
			for i, elem := range arr {
				if err := PickleAssign(elem, slice.Index(i).Addr().Interface()); err != nil {
					return fmt.Errorf("pickle: slice element %d: %w", i, err)
				}
			}
			target.Set(slice)
			return nil
		}

	case reflect.Map:
		if m, ok := result.(map[string]any); ok {
			newMap := reflect.MakeMapWithSize(targetType, len(m))
			for k, v := range m {
				keyRV := reflect.ValueOf(k)
				valRV := reflect.New(targetType.Elem())
				if err := PickleAssign(v, valRV.Interface()); err != nil {
					return fmt.Errorf("pickle: map key %q: %w", k, err)
				}
				newMap.SetMapIndex(keyRV, valRV.Elem())
			}
			target.Set(newMap)
			return nil
		}

	case reflect.Interface:
		target.Set(resultRV)
		return nil
	}

	return fmt.Errorf("pickle: cannot assign %T to %s", result, targetType)
}

func normalizePickleValue(v any) any {
	switch x := v.(type) {
	case nil, bool, string, []byte:
		return x
	case mpickle.None:
		return nil
	case int:
		return x
	case int8:
		return int(x)
	case int16:
		return int(x)
	case int32:
		return int(x)
	case int64:
		if x >= math.MinInt && x <= math.MaxInt {
			return int(x)
		}
		return x
	case uint:
		if uint64(x) <= uint64(math.MaxInt) {
			return int(x)
		}
		return int64(x)
	case uint8:
		return int(x)
	case uint16:
		return int(x)
	case uint32:
		if uint64(x) <= uint64(math.MaxInt) {
			return int(x)
		}
		return int64(x)
	case uint64:
		if x <= uint64(math.MaxInt) {
			return int(x)
		}
		return int64(x)
	case float32:
		return float64(x)
	case float64:
		return x
	case gtypes.ByteArray:
		b := make([]byte, len(x))
		copy(b, x)
		return b
	case *gtypes.ByteArray:
		if x == nil {
			return nil
		}
		b := make([]byte, len(*x))
		copy(b, *x)
		return b
	case gtypes.Dict:
		out := make(map[string]any, len(x))
		for _, entry := range x {
			out[pickleKeyToString(entry.Key)] = normalizePickleValue(entry.Value)
		}
		return out
	case *gtypes.Dict:
		if x == nil {
			return map[string]any{}
		}
		out := make(map[string]any, len(*x))
		for _, entry := range *x {
			out[pickleKeyToString(entry.Key)] = normalizePickleValue(entry.Value)
		}
		return out
	case gtypes.List:
		out := make([]any, len(x))
		for i := range x {
			out[i] = normalizePickleValue(x[i])
		}
		return out
	case *gtypes.List:
		if x == nil {
			return []any{}
		}
		out := make([]any, len(*x))
		for i := range *x {
			out[i] = normalizePickleValue((*x)[i])
		}
		return out
	case gtypes.Tuple:
		out := make([]any, len(x))
		for i := range x {
			out[i] = normalizePickleValue(x[i])
		}
		return out
	case *gtypes.Tuple:
		if x == nil {
			return []any{}
		}
		out := make([]any, len(*x))
		for i := range *x {
			out[i] = normalizePickleValue((*x)[i])
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i := range x {
			out[i] = normalizePickleValue(x[i])
		}
		return out
	case mpickle.Tuple:
		out := make([]any, len(x))
		for i := range x {
			out[i] = normalizePickleValue(x[i])
		}
		return out
	case map[interface{}]interface{}:
		out := make(map[string]any, len(x))
		for k, v := range x {
			out[pickleKeyToString(k)] = normalizePickleValue(v)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, v := range x {
			out[k] = normalizePickleValue(v)
		}
		return out
	default:
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Slice, reflect.Array:
			out := make([]any, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				out[i] = normalizePickleValue(rv.Index(i).Interface())
			}
			return out
		case reflect.Map:
			out := make(map[string]any, rv.Len())
			for _, key := range rv.MapKeys() {
				out[pickleKeyToString(key.Interface())] = normalizePickleValue(rv.MapIndex(key).Interface())
			}
			return out
		default:
			return v
		}
	}
}

func pickleKeyToString(key any) string {
	switch k := key.(type) {
	case string:
		return k
	case []byte:
		return string(k)
	case mpickle.Bytes:
		return string(k)
	case fmt.Stringer:
		return k.String()
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", k)
	}
}

type bytearrayCallable struct{}

func (bytearrayCallable) Call(args ...interface{}) (interface{}, error) {
	switch len(args) {
	case 0:
		return []byte{}, nil
	case 1:
		return toBytes(args[0])
	case 2:
		if enc, ok := args[1].(string); ok && (enc == "latin1" || enc == "latin-1") {
			return toBytes(args[0])
		}
	}
	return nil, fmt.Errorf("pickle: unsupported bytearray invocation: %#v", args)
}

func toBytes(v any) ([]byte, error) {
	switch x := v.(type) {
	case nil:
		return nil, nil
	case []byte:
		b := make([]byte, len(x))
		copy(b, x)
		return b, nil
	case string:
		return []byte(x), nil
	case gtypes.ByteArray:
		b := make([]byte, len(x))
		copy(b, x)
		return b, nil
	case *gtypes.ByteArray:
		if x == nil {
			return nil, nil
		}
		b := make([]byte, len(*x))
		copy(b, *x)
		return b, nil
	case []any:
		b := make([]byte, len(x))
		for i, elem := range x {
			switch n := elem.(type) {
			case int:
				b[i] = byte(n)
			case int64:
				b[i] = byte(n)
			case uint:
				b[i] = byte(n)
			case uint64:
				b[i] = byte(n)
			default:
				return nil, fmt.Errorf("pickle: unsupported byte element %T", elem)
			}
		}
		return b, nil
	default:
		rv := reflect.ValueOf(v)
		if !rv.IsValid() {
			return nil, nil
		}
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
			if rv.Type().Elem().Kind() != reflect.Uint8 {
				return nil, fmt.Errorf("pickle: unsupported bytearray source %T", v)
			}
			b := make([]byte, rv.Len())
			reflect.Copy(reflect.ValueOf(b), rv)
			return b, nil
		}
	}
	return nil, fmt.Errorf("pickle: unsupported bytearray source %T", v)
}
