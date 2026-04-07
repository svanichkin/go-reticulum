package interfaces

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
)

type rnodeLogAdapter struct{}

func (rnodeLogAdapter) Debugf(format string, args ...any) { if DiagLogf != nil { DiagLogf(LogDebug, format, args...) } }
func (rnodeLogAdapter) Infof(format string, args ...any)  { if DiagLogf != nil { DiagLogf(LogInfo, format, args...) } }
func (rnodeLogAdapter) Warnf(format string, args ...any)  { if DiagLogf != nil { DiagLogf(LogWarning, format, args...) } }
func (rnodeLogAdapter) Errorf(format string, args ...any) { if DiagLogf != nil { DiagLogf(LogError, format, args...) } }

type rnodeOwnerAdapter struct{ ifc *Interface }

func (o rnodeOwnerAdapter) Inbound(data []byte, _ *RNodeInterface) {
	if o.ifc == nil || len(data) == 0 {
		return
	}
	o.ifc.RXB += uint64(len(data))
	if InboundHandler != nil {
		InboundHandler(data, o.ifc)
	}
}

var rnodeTransportFromPort = NewRNodeInterfaceFromPort
var rnodeStartInterface = func(rn *RNodeInterface, ctx context.Context) error { return rn.Start(ctx) }
var rnodeStartReconnectLoop = func(rn *RNodeInterface) { go rn.reconnectLoop() }

// NewRNodeInterfaceFromConfig builds and starts a single RNodeInterface driver from a config map,
// mirroring Python's RNodeInterface(Interface.get_config_obj(...)).
func NewRNodeInterfaceFromConfig(name string, kv map[string]string) (*Interface, error) {
	get := func(key string) string {
		if kv == nil {
			return ""
		}
		if v, ok := kv[key]; ok {
			return strings.TrimSpace(v)
		}
		lower := strings.ToLower(key)
		for k, v := range kv {
			if strings.ToLower(k) == lower {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}
	parseInt := func(key string) (int, bool) {
		raw := strings.TrimSpace(get(key))
		if raw == "" {
			return 0, false
		}
		v, err := strconv.Atoi(raw)
		return v, err == nil
	}
	parseFloat := func(key string) (float64, bool) {
		raw := strings.TrimSpace(get(key))
		if raw == "" {
			return 0, false
		}
		v, err := strconv.ParseFloat(raw, 64)
		return v, err == nil
	}
	parseBool := func(key string, def bool) bool {
		raw := strings.TrimSpace(get(key))
		if raw == "" {
			return def
		}
		switch strings.ToLower(raw) {
		case "1", "true", "yes", "y", "on":
			return true
		case "0", "false", "no", "n", "off":
			return false
		default:
			return def
		}
	}

	iface := &Interface{
		Name:              strings.TrimSpace(name),
		Type:              "RNodeInterface",
		IN:                true,
		OUT:               true,
		DriverImplemented: true,
		HWMTU:             508, // Python: HW_MTU = 508
	}

	port := get("port")
	if port == "" {
		return nil, errors.New("no port specified for RNode interface")
	}

	rn, err := rnodeTransportFromPort(rnodeOwnerAdapter{ifc: iface}, rnodeLogAdapter{}, iface.Name, port)
	if err != nil {
		return nil, err
	}

	if v, ok := parseInt("frequency"); ok {
		rn.Frequency = uint32(v)
	}
	if v, ok := parseInt("bandwidth"); ok {
		rn.Bandwidth = uint32(v)
	}
	if v, ok := parseInt("txpower"); ok {
		if v < 0 {
			v = 0
		}
		rn.TXPower = byte(v)
	}
	if v, ok := parseInt("spreadingfactor"); ok {
		if v < 0 {
			v = 0
		}
		rn.SF = byte(v)
	}
	if v, ok := parseInt("codingrate"); ok {
		if v < 0 {
			v = 0
		}
		rn.CR = byte(v)
	}

	rn.FlowCtrl = parseBool("flow_control", false)

	if v, ok := parseInt("id_interval"); ok && v > 0 {
		rn.idInterval = time.Duration(v) * time.Second
	}
	if cs := get("id_callsign"); cs != "" {
		rn.idCallsign = []byte(cs)
	}
	if v, ok := parseFloat("airtime_limit_short"); ok {
		rn.ShortAirtimeLimit = v
	}
	if v, ok := parseFloat("airtime_limit_long"); ok {
		rn.LongAirtimeLimit = v
	}

	iface.rnodeSingle = rn

	if !rn.ValidateConfig() {
		iface.rnodeSingle = nil
		return nil, errors.New("invalid configuration")
	}

	if err := rnodeStartInterface(rn, context.Background()); err != nil {
		if DiagLogf != nil {
			DiagLogf(LogError, "Could not open serial port for interface %s", iface.String())
			DiagLogf(LogError, "The contained exception was: %v", err)
			DiagLogf(LogError, "Reticulum will attempt to bring up this interface periodically")
		}
		rnodeStartReconnectLoop(rn)
		return iface, nil
	}

	// Best-effort bitrate estimate for stats/mtu tuning.
	if br := rn.BitrateEstimate(); br > 0 {
		iface.Bitrate = int(br)
	}

	return iface, nil
}
