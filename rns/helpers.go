package rns

func asFloat64(v any) float64 {
	switch val := v.(type) {
	case nil:
		return 0
	case float64:
		return val
	case *float64:
		if val != nil {
			return *val
		}
		return 0
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int8:
		return float64(val)
	case int16:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case uint:
		return float64(val)
	case uint8:
		return float64(val)
	case uint16:
		return float64(val)
	case uint32:
		return float64(val)
	case uint64:
		return float64(val)
	default:
		return 0
	}
}
