package rns

import "testing"

type outboundCapture struct {
	packets []*Packet
	ifaces  []*Interface
}

func attachOutboundCapture(t *testing.T, sink *outboundCapture, ifc *Interface) {
	t.Helper()
	if sink == nil || ifc == nil {
		return
	}
	ifc.OUT = true
	ifc.SetProcessOutgoingFunc(func(data []byte) error {
		pkt := &Packet{Raw: append([]byte(nil), data...)}
		if !pkt.Unpack() {
			t.Fatalf("failed to unpack captured packet from %s", ifc)
		}
		sink.packets = append(sink.packets, pkt)
		sink.ifaces = append(sink.ifaces, ifc)
		return nil
	})
}

