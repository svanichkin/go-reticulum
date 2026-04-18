package rns

import (
	"bytes"
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
	if got := asDiscoveryString(discoveryRawGet(raw, discoveryFieldInterfaceType)); got != "TCPServerInterface" {
		t.Fatalf("interface type=%q, want TCPServerInterface", got)
	}
	if got := asDiscoveryString(discoveryRawGet(raw, discoveryFieldReachableOn)); got != "example.com" {
		t.Fatalf("reachable_on=%q, want example.com", got)
	}
	if got := asDiscoveryInt(discoveryRawGet(raw, discoveryFieldPort)); got != 4242 {
		t.Fatalf("port=%d, want 4242", got)
	}
	if got := asDiscoveryString(discoveryRawGet(raw, discoveryFieldIFACNetname)); got != "mesh" {
		t.Fatalf("ifac_netname=%q, want mesh", got)
	}
	if got := asDiscoveryString(discoveryRawGet(raw, discoveryFieldIFACNetkey)); got != "secret" {
		t.Fatalf("ifac_netkey=%q, want secret", got)
	}
}

func TestDiscoveryInfoFromRaw_MissingLocationFieldsStayNil(t *testing.T) {
	id, err := NewIdentity()
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}

	raw := map[any]any{
		discoveryFieldInterfaceType: "TCPServerInterface",
		discoveryFieldName:          "public-tcp",
		discoveryFieldTransport:     true,
	}

	info := discoveryInfoFromRaw(raw, []byte{0x01, 0x02}, id, []byte{0xaa}, 23)
	if info == nil {
		t.Fatal("discoveryInfoFromRaw returned nil")
	}
	if _, ok := info["latitude"]; !ok {
		t.Fatal("latitude key missing")
	}
	if _, ok := info["longitude"]; !ok {
		t.Fatal("longitude key missing")
	}
	if _, ok := info["height"]; !ok {
		t.Fatal("height key missing")
	}
	if info["latitude"] != nil {
		t.Fatalf("latitude=%v, want nil", info["latitude"])
	}
	if info["longitude"] != nil {
		t.Fatalf("longitude=%v, want nil", info["longitude"])
	}
	if info["height"] != nil {
		t.Fatalf("height=%v, want nil", info["height"])
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

	nameWithIdentity, err := DestinationExpandName(TransportIdentity, TransportAppName, "discovery", "interface")
	if err != nil {
		t.Fatalf("DestinationExpandName(identity): %v", err)
	}
	nameWithoutIdentity, err := DestinationExpandName(nil, TransportAppName, "discovery", "interface")
	if err != nil {
		t.Fatalf("DestinationExpandName(nil): %v", err)
	}
	discoveryHash, err := DestinationHash(TransportIdentity, TransportAppName, "discovery", "interface")
	if err != nil {
		t.Fatalf("DestinationHash: %v", err)
	}
	discoveryDest := &Destination{
		Type:            DestinationSINGLE,
		Direction:       DestinationIN,
		identity:        TransportIdentity,
		name:            nameWithIdentity,
		hash:            discoveryHash,
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
	if got, _ := info["name"].(string); got != "public-tcp" {
		t.Fatalf("name=%q, want public-tcp", got)
	}
	if got, _ := info["reachable_on"].(string); got != "reticulum.example" {
		t.Fatalf("reachable_on=%q, want reticulum.example", got)
	}
	if got := asDiscoveryInt(info["port"]); got != 7777 {
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
	if got := asDiscoveryString(discoveryRawGet(raw, discoveryFieldReachableOn)); got != "exec.example" {
		t.Fatalf("reachable_on=%q, want exec.example", got)
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

	nameWithIdentity, err := DestinationExpandName(remoteID, TransportAppName, "discovery", "interface")
	if err != nil {
		t.Fatalf("DestinationExpandName(identity): %v", err)
	}
	nameWithoutIdentity, err := DestinationExpandName(nil, TransportAppName, "discovery", "interface")
	if err != nil {
		t.Fatalf("DestinationExpandName(nil): %v", err)
	}
	discoveryHash, err := DestinationHash(remoteID, TransportAppName, "discovery", "interface")
	if err != nil {
		t.Fatalf("DestinationHash: %v", err)
	}
	discoveryDest := &Destination{
		Type:            DestinationSINGLE,
		Direction:       DestinationIN,
		identity:        remoteID,
		name:            nameWithIdentity,
		hash:            discoveryHash,
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

	announcer := NewInterfaceAnnouncer()
	if announcer == nil {
		t.Fatal("NewInterfaceAnnouncer() returned nil")
	}
	handler := NewInterfaceAnnounceHandler(14, nil)
	if handler == nil {
		t.Fatal("NewInterfaceAnnounceHandler() returned nil")
	}
	if panics != 2 {
		t.Fatalf("panic calls=%d, want 2", panics)
	}
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

	packed, err := packDiscoveryInfo(info)
	if err != nil {
		t.Fatalf("packDiscoveryInfo: %v", err)
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

	packedA, err := packDiscoveryInfo(infoA)
	if err != nil {
		t.Fatalf("packDiscoveryInfo(infoA): %v", err)
	}
	packedB, err := packDiscoveryInfo(infoB)
	if err != nil {
		t.Fatalf("packDiscoveryInfo(infoB): %v", err)
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
