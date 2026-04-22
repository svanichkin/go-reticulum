package rns

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"

	umsgpack "github.com/svanichkin/go-reticulum/rns/vendor"
)

type fakeDiscoveryStamper struct {
	stamp []byte
	value int
}

type countingDiscoveryStamper struct {
	fakeDiscoveryStamper
	calls int
}

func (f fakeDiscoveryStamper) StampSize() int {
	return len(f.stamp)
}

func (f fakeDiscoveryStamper) GenerateStamp(_ []byte, _ int, _ int) ([]byte, int, error) {
	return append([]byte(nil), f.stamp...), f.value, nil
}

func (f fakeDiscoveryStamper) StampWorkblock(_ []byte, _ int) []byte {
	return []byte("work")
}

func (f fakeDiscoveryStamper) StampValue(_ []byte, _ []byte) int {
	return f.value
}

func (f fakeDiscoveryStamper) StampValid(stamp []byte, requiredValue int, _ []byte) bool {
	return bytes.Equal(stamp, f.stamp) && f.value >= requiredValue
}

func (f *countingDiscoveryStamper) GenerateStamp(_ []byte, _ int, _ int) ([]byte, int, error) {
	f.calls++
	return append([]byte(nil), f.stamp...), f.value, nil
}

func TestInterfaceAnnouncer_GetInterfaceAnnounceData_TCPServer(t *testing.T) {
	prevStamper := DiscoveryStampProvider
	prevTransportIdentity := TransportIdentity
	prevNetworkIdentity := NetworkIdentity
	prevDestinations := Destinations

	DiscoveryStampProvider = fakeDiscoveryStamper{stamp: []byte{1, 2, 3, 4}, value: 21}
	NetworkIdentity = nil
	id, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	TransportIdentity = id

	t.Cleanup(func() {
		DiscoveryStampProvider = prevStamper
		TransportIdentity = prevTransportIdentity
		NetworkIdentity = prevNetworkIdentity
		Destinations = prevDestinations
	})
	Destinations = nil

	port := 4242
	ifc := &Interface{
		Type:                      "TCPServerInterface",
		Name:                      "srv0",
		Discoverable:              true,
		DiscoveryReachableOn:      "example.com",
		DiscoveryPort:             &port,
		DiscoveryPublishIFAC:      true,
		IFACNetnameVal:            "mesh",
		IFACNetkeyVal:             "secret",
		DiscoveryAnnounceInterval: 6 * time.Hour,
	}

	announcer := NewInterfaceAnnouncer()
	data, err := announcer.GetInterfaceAnnounceData(ifc)
	if err != nil {
		t.Fatalf("GetInterfaceAnnounceData: %v", err)
	}
	if len(data) < 1+4 {
		t.Fatalf("announce data too short: %d", len(data))
	}
	if data[0] != 0 {
		t.Fatalf("flags=%d, want 0", data[0])
	}

	raw := map[any]any{}
	if err := umsgpack.Unpackb(data[1:len(data)-4], &raw); err != nil {
		t.Fatalf("Unpackb: %v", err)
	}
	lookup := func(raw map[any]any, want int) any {
		for k, v := range raw {
			switch x := k.(type) {
			case int:
				if x == want {
					return v
				}
			case int64:
				if int(x) == want {
					return v
				}
			case uint64:
				if int(x) == want {
					return v
				}
			case uint8:
				if int(x) == want {
					return v
				}
			}
		}
		return nil
	}
	stringOf := func(v any) string {
		switch x := v.(type) {
		case string:
			return x
		case []byte:
			return string(x)
		default:
			return ""
		}
	}
	if got := stringOf(lookup(raw, discoveryFieldInterfaceType)); got != "TCPServerInterface" {
		t.Fatalf("interface type=%q, want TCPServerInterface", stringOf(got))
	}
	if got := stringOf(lookup(raw, discoveryFieldReachableOn)); got != "example.com" {
		t.Fatalf("reachable_on=%q, want example.com", stringOf(got))
	}
	intOf := func(v any) int {
		switch x := v.(type) {
		case int:
			return x
		case int64:
			return int(x)
		case uint64:
			return int(x)
		case float64:
			return int(x)
		default:
			return 0
		}
	}
	if got := intOf(lookup(raw, discoveryFieldPort)); got != 4242 {
		t.Fatalf("port=%d, want 4242", intOf(got))
	}
	if got := stringOf(lookup(raw, discoveryFieldIFACNetname)); got != "mesh" {
		t.Fatalf("ifac_netname=%q, want mesh", stringOf(got))
	}
	if got := stringOf(lookup(raw, discoveryFieldIFACNetkey)); got != "secret" {
		t.Fatalf("ifac_netkey=%q, want secret", stringOf(got))
	}
}

func TestInterfaceAnnounceHandler_ReceivedAnnounce_MissingLocationFieldsStayNil(t *testing.T) {
	prevStamper := DiscoveryStampProvider
	prevTransportIdentity := TransportIdentity
	prevNetworkIdentity := NetworkIdentity

	DiscoveryStampProvider = fakeDiscoveryStamper{stamp: []byte{7, 7, 7, 7}, value: 23}
	NetworkIdentity = nil
	id, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	TransportIdentity = id

	t.Cleanup(func() {
		DiscoveryStampProvider = prevStamper
		TransportIdentity = prevTransportIdentity
		NetworkIdentity = prevNetworkIdentity
	})

	captured := map[string]any{}
	handler := &InterfaceAnnounceHandler{
		requiredValue: 14,
		callback: func(info map[string]any) {
			captured = info
		},
	}

	raw := map[any]any{
		discoveryFieldInterfaceType: "TCPServerInterface",
		discoveryFieldName:          "public-tcp",
		discoveryFieldTransport:     true,
		discoveryFieldReachableOn:   "reticulum.example",
		discoveryFieldPort:          4242,
	}
	packed, err := umsgpack.Packb(raw)
	if err != nil {
		t.Fatalf("Packb: %v", err)
	}
	appData := append([]byte{0}, append(packed, []byte{7, 7, 7, 7}...)...)

	handler.ReceivedAnnounce([]byte{0x01, 0x02}, id, appData)
	if len(captured) == 0 {
		t.Fatal("callback was not invoked")
	}
	if _, ok := captured["latitude"]; !ok {
		t.Fatal("latitude key missing")
	}
	if _, ok := captured["longitude"]; !ok {
		t.Fatal("longitude key missing")
	}
	if _, ok := captured["height"]; !ok {
		t.Fatal("height key missing")
	}
	if captured["latitude"] != nil {
		t.Fatalf("latitude=%v, want nil", captured["latitude"])
	}
	if captured["longitude"] != nil {
		t.Fatalf("longitude=%v, want nil", captured["longitude"])
	}
	if captured["height"] != nil {
		t.Fatalf("height=%v, want nil", captured["height"])
	}
}

func TestInterfaceDiscovery_PersistsDiscoveredInterfaceFromAnnounce(t *testing.T) {
	prevStamper := DiscoveryStampProvider
	prevTransportIdentity := TransportIdentity
	prevNetworkIdentity := NetworkIdentity
	prevOwner := Owner
	prevHandlers := announceHandlers

	DiscoveryStampProvider = fakeDiscoveryStamper{stamp: []byte{9, 9, 9, 9}, value: 22}
	announceHandlers = nil
	NetworkIdentity = nil
	id, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	TransportIdentity = id
	Owner = &Reticulum{StoragePath: t.TempDir()}

	t.Cleanup(func() {
		DiscoveryStampProvider = prevStamper
		TransportIdentity = prevTransportIdentity
		NetworkIdentity = prevNetworkIdentity
		Owner = prevOwner
		announceHandlers = prevHandlers
	})

	discovery, err := NewInterfaceDiscovery(14, nil, true)
	if err != nil {
		t.Fatalf("NewInterfaceDiscovery: %v", err)
	}
	t.Cleanup(func() {
		if discovery.handler != nil {
			DeregisterAnnounceHandler(discovery.handler)
		}
	})

	port := 7777
	ifc := &Interface{
		Type:                 "TCPServerInterface",
		Name:                 "public-tcp",
		Discoverable:         true,
		DiscoveryReachableOn: "reticulum.example",
		DiscoveryPort:        &port,
	}
	announcer := NewInterfaceAnnouncer()
	appData, err := announcer.GetInterfaceAnnounceData(ifc)
	if err != nil {
		t.Fatalf("GetInterfaceAnnounceData: %v", err)
	}

	nameWithIdentity, err := (&Destination{}).ExpandName(TransportIdentity, TransportAppName, "discovery", "interface")
	if err != nil {
		t.Fatalf("DestinationExpandName(identity): %v", err)
	}
	nameWithoutIdentity, err := (&Destination{}).ExpandName(nil, TransportAppName, "discovery", "interface")
	if err != nil {
		t.Fatalf("DestinationExpandName(nil): %v", err)
	}
	discoveryHash, err := destinationHash(TransportIdentity, TransportAppName, "discovery", "interface")
	if err != nil {
		t.Fatalf("DestinationHash: %v", err)
	}
	discoveryDest := &Destination{
		Type:            DestinationSINGLE,
		Direction:       DestinationIN,
		identity:        TransportIdentity,
		name:            nameWithIdentity,
		Hash:            discoveryHash,
		nameHash:        FullHash([]byte(nameWithoutIdentity))[:IdentityNameHashLength/8],
		hexhash:         PrettyHexRep(discoveryHash),
		pathResponses:   make(map[string]*pathResponseEntry),
		requestHandlers: make(map[string]*RequestHandler),
		links:           []*Link{},
	}
	packet := discoveryDest.Announce(appData, false, nil, nil, false)
	if packet == nil {
		t.Fatal("Announce returned nil")
	}
	if err := packet.Pack(); err != nil {
		t.Fatalf("Pack(): %v", err)
	}
	if ok := ValidateAnnounce(packet, false); !ok {
		t.Fatal("ValidateAnnounce returned false for discovery test packet")
	}

	notifyAnnounceHandlersForTest(packet)
	deadline := time.Now().Add(2 * time.Second)
	var list []map[string]any
	for time.Now().Before(deadline) {
		list = discovery.ListDiscoveredInterfaces(false, false)
		if len(list) == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(list) != 1 {
		t.Fatalf("discovered interfaces=%d, want 1", len(list))
	}
	info := list[0]
	intOf := func(v any) int {
		switch x := v.(type) {
		case int:
			return x
		case int64:
			return int(x)
		case uint64:
			return int(x)
		case float64:
			return int(x)
		default:
			return 0
		}
	}
	if got, _ := info["name"].(string); got != "public-tcp" {
		t.Fatalf("name=%q, want public-tcp", got)
	}
	if got, _ := info["reachable_on"].(string); got != "reticulum.example" {
		t.Fatalf("reachable_on=%q, want reticulum.example", got)
	}
	if got := intOf(info["port"]); got != 7777 {
		t.Fatalf("port=%v, want 7777", info["port"])
	}
}

func TestInterfaceAnnouncer_GetInterfaceAnnounceData_UsesStampCache(t *testing.T) {
	prevStamper := DiscoveryStampProvider
	prevTransportIdentity := TransportIdentity
	prevNetworkIdentity := NetworkIdentity

	stamper := &countingDiscoveryStamper{fakeDiscoveryStamper: fakeDiscoveryStamper{stamp: []byte{5, 5, 5, 5}, value: 30}}
	DiscoveryStampProvider = stamper
	NetworkIdentity = nil
	id, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	TransportIdentity = id

	t.Cleanup(func() {
		DiscoveryStampProvider = prevStamper
		TransportIdentity = prevTransportIdentity
		NetworkIdentity = prevNetworkIdentity
	})

	port := 4242
	ifc := &Interface{
		Type:                 "TCPServerInterface",
		Name:                 "srv-cache",
		Discoverable:         true,
		DiscoveryReachableOn: "example.com",
		DiscoveryPort:        &port,
	}

	announcer := NewInterfaceAnnouncer()
	if _, err := announcer.GetInterfaceAnnounceData(ifc); err != nil {
		t.Fatalf("GetInterfaceAnnounceData #1: %v", err)
	}
	if _, err := announcer.GetInterfaceAnnounceData(ifc); err != nil {
		t.Fatalf("GetInterfaceAnnounceData #2: %v", err)
	}
	if stamper.calls != 1 {
		t.Fatalf("GenerateStamp calls=%d, want 1", stamper.calls)
	}
}

func TestInterfaceAnnouncer_GetInterfaceAnnounceData_TCPClientWithoutKISSRejected(t *testing.T) {
	prevStamper := DiscoveryStampProvider
	prevTransportIdentity := TransportIdentity
	prevNetworkIdentity := NetworkIdentity

	DiscoveryStampProvider = fakeDiscoveryStamper{stamp: []byte{1, 2, 3, 4}, value: 21}
	NetworkIdentity = nil
	id, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	TransportIdentity = id

	t.Cleanup(func() {
		DiscoveryStampProvider = prevStamper
		TransportIdentity = prevTransportIdentity
		NetworkIdentity = prevNetworkIdentity
	})

	port := 4242
	ifc := &Interface{
		Type:                 "TCPClientInterface",
		Name:                 "tcp-client-no-kiss",
		Discoverable:         true,
		DiscoveryReachableOn: "example.com",
		DiscoveryPort:        &port,
	}

	announcer := NewInterfaceAnnouncer()
	if _, err := announcer.GetInterfaceAnnounceData(ifc); err == nil {
		t.Fatalf("GetInterfaceAnnounceData() error = nil, want invalid TCP discovery configuration")
	}
}

func TestInterfaceAnnouncer_GetInterfaceAnnounceData_ReachableOnExecutable(t *testing.T) {
	if umsgpack.IsWindows() {
		t.Skip("executable reachable_on flow is non-Windows only")
	}

	prevStamper := DiscoveryStampProvider
	prevTransportIdentity := TransportIdentity
	prevNetworkIdentity := NetworkIdentity

	DiscoveryStampProvider = fakeDiscoveryStamper{stamp: []byte{1, 2, 3, 4}, value: 21}
	NetworkIdentity = nil
	id, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	TransportIdentity = id

	t.Cleanup(func() {
		DiscoveryStampProvider = prevStamper
		TransportIdentity = prevTransportIdentity
		NetworkIdentity = prevNetworkIdentity
	})

	scriptPath := filepath.Join(t.TempDir(), "reachable_on.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\necho exec.example\n"), 0o755); err != nil {
		t.Fatalf("WriteFile(script): %v", err)
	}

	port := 4242
	ifc := &Interface{
		Type:                 "TCPServerInterface",
		Name:                 "srv-exec",
		Discoverable:         true,
		DiscoveryReachableOn: scriptPath,
		DiscoveryPort:        &port,
	}

	announcer := NewInterfaceAnnouncer()
	data, err := announcer.GetInterfaceAnnounceData(ifc)
	if err != nil {
		t.Fatalf("GetInterfaceAnnounceData: %v", err)
	}

	raw := map[any]any{}
	if err := umsgpack.Unpackb(data[1:len(data)-4], &raw); err != nil {
		t.Fatalf("Unpackb: %v", err)
	}
	lookup := func(raw map[any]any, want int) any {
		for k, v := range raw {
			switch x := k.(type) {
			case int:
				if x == want {
					return v
				}
			case int64:
				if int(x) == want {
					return v
				}
			case uint64:
				if int(x) == want {
					return v
				}
			case uint8:
				if int(x) == want {
					return v
				}
			}
		}
		return nil
	}
	stringOf := func(v any) string {
		switch x := v.(type) {
		case string:
			return x
		case []byte:
			return string(x)
		default:
			return ""
		}
	}
	if got := stringOf(lookup(raw, discoveryFieldReachableOn)); got != "exec.example" {
		t.Fatalf("reachable_on=%q, want exec.example", stringOf(got))
	}
}

func TestInterfaceDiscovery_SourceFilterSkipsUnauthorizedAnnounce(t *testing.T) {
	prevStamper := DiscoveryStampProvider
	prevTransportIdentity := TransportIdentity
	prevNetworkIdentity := NetworkIdentity
	prevOwner := Owner
	prevHandlers := announceHandlers
	prevSources := InterfaceDiscoverySources()
	prevDestinations := Destinations

	DiscoveryStampProvider = fakeDiscoveryStamper{stamp: []byte{9, 9, 9, 9}, value: 22}
	announceHandlers = nil
	NetworkIdentity = nil
	interfaceDiscoverySources = [][]byte{{0xAA, 0xBB}}
	id, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	TransportIdentity = id
	Owner = &Reticulum{StoragePath: t.TempDir()}

	t.Cleanup(func() {
		DiscoveryStampProvider = prevStamper
		TransportIdentity = prevTransportIdentity
		NetworkIdentity = prevNetworkIdentity
		Destinations = prevDestinations
		Owner = prevOwner
		announceHandlers = prevHandlers
		interfaceDiscoverySources = prevSources
	})
	Destinations = nil

	discovery, err := NewInterfaceDiscovery(14, nil, true)
	if err != nil {
		t.Fatalf("NewInterfaceDiscovery: %v", err)
	}
	t.Cleanup(func() {
		if discovery.handler != nil {
			DeregisterAnnounceHandler(discovery.handler)
		}
	})

	remoteID, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity(remote): %v", err)
	}
	port := 7777
	ifc := &Interface{
		Type:                 "TCPServerInterface",
		Name:                 "public-tcp",
		Discoverable:         true,
		DiscoveryReachableOn: "reticulum.example",
		DiscoveryPort:        &port,
	}
	announcer := NewInterfaceAnnouncer()
	appData, err := announcer.GetInterfaceAnnounceData(ifc)
	if err != nil {
		t.Fatalf("GetInterfaceAnnounceData: %v", err)
	}

	nameWithIdentity, err := (&Destination{}).ExpandName(remoteID, TransportAppName, "discovery", "interface")
	if err != nil {
		t.Fatalf("DestinationExpandName(identity): %v", err)
	}
	nameWithoutIdentity, err := (&Destination{}).ExpandName(nil, TransportAppName, "discovery", "interface")
	if err != nil {
		t.Fatalf("DestinationExpandName(nil): %v", err)
	}
	discoveryHash, err := destinationHash(remoteID, TransportAppName, "discovery", "interface")
	if err != nil {
		t.Fatalf("DestinationHash: %v", err)
	}
	discoveryDest := &Destination{
		Type:            DestinationSINGLE,
		Direction:       DestinationIN,
		identity:        remoteID,
		name:            nameWithIdentity,
		Hash:            discoveryHash,
		nameHash:        FullHash([]byte(nameWithoutIdentity))[:IdentityNameHashLength/8],
		hexhash:         PrettyHexRep(discoveryHash),
		pathResponses:   make(map[string]*pathResponseEntry),
		requestHandlers: make(map[string]*RequestHandler),
		links:           []*Link{},
	}
	packet := discoveryDest.Announce(appData, false, nil, nil, false)
	if packet == nil {
		t.Fatal("Announce returned nil")
	}
	if err := packet.Pack(); err != nil {
		t.Fatalf("Pack(): %v", err)
	}
	if ok := ValidateAnnounce(packet, false); !ok {
		t.Fatal("ValidateAnnounce returned false for discovery test packet")
	}

	notifyAnnounceHandlersForTest(packet)
	time.Sleep(20 * time.Millisecond)
	list := discovery.ListDiscoveredInterfaces(false, false)
	if len(list) != 0 {
		t.Fatalf("discovered interfaces=%d, want 0 for unauthorized source", len(list))
	}
}

func TestDiscoveryConstructors_PanicWithoutStampProvider(t *testing.T) {
	prevStamper := DiscoveryStampProvider
	prevPanic := discoveryPanic
	prevTransportIdentity := TransportIdentity
	prevNetworkIdentity := NetworkIdentity

	DiscoveryStampProvider = nil
	panics := 0
	discoveryPanic = func() { panics++ }
	NetworkIdentity = nil
	id, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	TransportIdentity = id

	t.Cleanup(func() {
		DiscoveryStampProvider = prevStamper
		discoveryPanic = prevPanic
		TransportIdentity = prevTransportIdentity
		NetworkIdentity = prevNetworkIdentity
	})

	expectPanic := func(name string, fn func()) {
		t.Helper()
		defer func() {
			if rec := recover(); rec == nil {
				t.Fatalf("%s did not panic", name)
			}
		}()
		fn()
	}

	expectPanic("NewInterfaceAnnouncer", func() { _ = NewInterfaceAnnouncer() })
	expectPanic("NewInterfaceAnnounceHandler", func() { _ = NewInterfaceAnnounceHandler(14, nil) })
	if panics != 2 {
		t.Fatalf("panic calls=%d, want 2", panics)
	}
}

func packDiscoveryInfoForTest(info map[any]any) ([]byte, error) {
	lookup := func(raw map[any]any, want int) (any, bool) {
		for k, v := range raw {
			switch x := k.(type) {
			case int:
				if x == want {
					return v, true
				}
			case int64:
				if int(x) == want {
					return v, true
				}
			case uint64:
				if int(x) == want {
					return v, true
				}
			case uint8:
				if int(x) == want {
					return v, true
				}
			}
		}
		return nil, false
	}
	stringOf := func(v any) string {
		switch x := v.(type) {
		case string:
			return x
		case []byte:
			return string(x)
		default:
			return ""
		}
	}
	writeHeader := func(buf *bytes.Buffer, n int) error {
		switch {
		case n <= 15:
			return buf.WriteByte(byte(0x80 | n))
		case n <= 0xFFFF:
			if err := buf.WriteByte(0xDE); err != nil {
				return err
			}
			var raw [2]byte
			binary.BigEndian.PutUint16(raw[:], uint16(n))
			_, err := buf.Write(raw[:])
			return err
		default:
			if err := buf.WriteByte(0xDF); err != nil {
				return err
			}
			var raw [4]byte
			binary.BigEndian.PutUint32(raw[:], uint32(n))
			_, err := buf.Write(raw[:])
			return err
		}
	}
	keys := []int{
		discoveryFieldInterfaceType,
		discoveryFieldTransport,
		discoveryFieldTransportID,
		discoveryFieldName,
		discoveryFieldLatitude,
		discoveryFieldLongitude,
		discoveryFieldHeight,
	}
	ifTypeVal, _ := lookup(info, discoveryFieldInterfaceType)
	switch sanitizeDiscoveryString(stringOf(ifTypeVal)) {
	case "BackboneInterface", "TCPServerInterface":
		keys = append(keys, discoveryFieldReachableOn, discoveryFieldPort)
	case "I2PInterface":
		keys = append(keys, discoveryFieldReachableOn)
	case "RNodeInterface":
		keys = append(keys, discoveryFieldFrequency, discoveryFieldBandwidth, discoveryFieldSpreading, discoveryFieldCodingRate)
	case "WeaveInterface":
		keys = append(keys, discoveryFieldFrequency, discoveryFieldBandwidth, discoveryFieldChannel, discoveryFieldModulation)
	case "KISSInterface":
		keys = append(keys, discoveryFieldFrequency, discoveryFieldBandwidth, discoveryFieldModulation)
	}
	if _, ok := lookup(info, discoveryFieldIFACNetname); ok {
		keys = append(keys, discoveryFieldIFACNetname)
	}
	if _, ok := lookup(info, discoveryFieldIFACNetkey); ok {
		keys = append(keys, discoveryFieldIFACNetkey)
	}
	filtered := make([]int, 0, len(keys))
	seen := make(map[int]struct{}, len(keys))
	for _, key := range keys {
		if _, ok := seen[key]; ok {
			continue
		}
		if _, ok := lookup(info, key); !ok {
			continue
		}
		seen[key] = struct{}{}
		filtered = append(filtered, key)
	}
	keys = filtered
	var buf bytes.Buffer
	if err := writeHeader(&buf, len(keys)); err != nil {
		return nil, err
	}
	for _, key := range keys {
		value, ok := lookup(info, key)
		if !ok {
			continue
		}
		packedKey, err := umsgpack.Packb(key)
		if err != nil {
			return nil, err
		}
		if _, err := buf.Write(packedKey); err != nil {
			return nil, err
		}
		packedValue, err := umsgpack.Packb(value)
		if err != nil {
			return nil, err
		}
		if _, err := buf.Write(packedValue); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func TestPackDiscoveryInfo_UsesPythonFieldOrder(t *testing.T) {
	info := map[any]any{
		discoveryFieldHeight:        float64(7),
		discoveryFieldIFACNetkey:    "secret",
		discoveryFieldPort:          4242,
		discoveryFieldTransport:     true,
		discoveryFieldLatitude:      float64(55.5),
		discoveryFieldInterfaceType: "TCPServerInterface",
		discoveryFieldTransportID:   []byte{0xAA, 0xBB},
		discoveryFieldLongitude:     float64(37.6),
		discoveryFieldReachableOn:   "example.com",
		discoveryFieldName:          "srv0",
		discoveryFieldIFACNetname:   "mesh",
	}

	packed, err := packDiscoveryInfoForTest(info)
	if err != nil {
		t.Fatalf("packDiscoveryInfoForTest: %v", err)
	}

	expectedKeys := []int{
		discoveryFieldInterfaceType,
		discoveryFieldTransport,
		discoveryFieldTransportID,
		discoveryFieldName,
		discoveryFieldLatitude,
		discoveryFieldLongitude,
		discoveryFieldHeight,
		discoveryFieldReachableOn,
		discoveryFieldPort,
		discoveryFieldIFACNetname,
		discoveryFieldIFACNetkey,
	}
	expectedValues := []any{
		"TCPServerInterface",
		true,
		[]byte{0xAA, 0xBB},
		"srv0",
		float64(55.5),
		float64(37.6),
		float64(7),
		"example.com",
		4242,
		"mesh",
		"secret",
	}

	var expected bytes.Buffer
	if err := expected.WriteByte(byte(0x80 | len(expectedKeys))); err != nil {
		t.Fatalf("WriteByte(map header): %v", err)
	}
	for idx, key := range expectedKeys {
		keyPacked, err := umsgpack.Packb(key)
		if err != nil {
			t.Fatalf("Packb(key %d): %v", key, err)
		}
		if _, err := expected.Write(keyPacked); err != nil {
			t.Fatalf("Write(key %d): %v", key, err)
		}
		valuePacked, err := umsgpack.Packb(expectedValues[idx])
		if err != nil {
			t.Fatalf("Packb(value %d): %v", idx, err)
		}
		if _, err := expected.Write(valuePacked); err != nil {
			t.Fatalf("Write(value %d): %v", idx, err)
		}
	}

	if !bytes.Equal(packed, expected.Bytes()) {
		t.Fatalf("packed discovery info order mismatch\n got: %x\nwant: %x", packed, expected.Bytes())
	}
}

func TestPackDiscoveryInfo_DeterministicAcrossMapInsertionOrder(t *testing.T) {
	infoA := map[any]any{
		discoveryFieldInterfaceType: "KISSInterface",
		discoveryFieldTransport:     true,
		discoveryFieldTransportID:   []byte{0x01},
		discoveryFieldName:          "kiss-a",
		discoveryFieldLatitude:      float64(0),
		discoveryFieldLongitude:     float64(0),
		discoveryFieldHeight:        float64(0),
		discoveryFieldFrequency:     123,
		discoveryFieldBandwidth:     456,
		discoveryFieldModulation:    "fsk",
	}
	infoB := map[any]any{
		discoveryFieldModulation:    "fsk",
		discoveryFieldBandwidth:     456,
		discoveryFieldFrequency:     123,
		discoveryFieldHeight:        float64(0),
		discoveryFieldLongitude:     float64(0),
		discoveryFieldLatitude:      float64(0),
		discoveryFieldName:          "kiss-a",
		discoveryFieldTransportID:   []byte{0x01},
		discoveryFieldTransport:     true,
		discoveryFieldInterfaceType: "KISSInterface",
	}

	packedA, err := packDiscoveryInfoForTest(infoA)
	if err != nil {
		t.Fatalf("packDiscoveryInfoForTest(infoA): %v", err)
	}
	packedB, err := packDiscoveryInfoForTest(infoB)
	if err != nil {
		t.Fatalf("packDiscoveryInfoForTest(infoB): %v", err)
	}
	if !bytes.Equal(packedA, packedB) {
		t.Fatalf("packed outputs differ across insertion order\nA: %x\nB: %x", packedA, packedB)
	}

	if len(packedA) < 1 {
		t.Fatal("packed output unexpectedly empty")
	}
	if got := int(packedA[0] & 0x0F); got != 10 {
		t.Fatalf("map size=%d, want 10", got)
	}
}
