package rns

import ifaces "github.com/svanichkin/go-reticulum/rns/interfaces"
import "time"

// Interface is re-exported from the interfaces subpackage so Go ports can live next to
// the original Python drivers in rns/interfaces/.
type Interface = ifaces.Interface

func init() {
	// Provide inbound callback to interfaces package without introducing import cycles.
	ifaces.InboundHandler = func(raw []byte, ifc *ifaces.Interface) {
		Inbound(raw, ifc)
	}
	ifaces.DiagLog = Log
	ifaces.DiagLogf = Logf
	ifaces.ExitFunc = Exit
	ifaces.PanicFunc = Panic
	ifaces.PanicOnInterfaceErrorProvider = func() bool {
		if instance == nil {
			return false
		}
		return instance.PanicOnInterfaceError
	}
	ifaces.SpawnHandler = func(ifc *ifaces.Interface) {
		// Register spawned sub-interfaces like AutoInterface peers.
		if ifc == nil {
			return
		}
		interfacesMu.Lock()
		Interfaces = append(Interfaces, ifc)
		interfacesMu.Unlock()
	}
	ifaces.RemoveInterfaceHandler = func(ifc *ifaces.Interface) {
		if ifc == nil {
			return
		}
		if len(ifc.TunnelID) == HashLengthBytes {
			VoidTunnelInterface(ifc.TunnelID)
		}
		ifc.Detach()
		interfacesMu.Lock()
		for idx, existing := range LocalClientInterfaces {
			if existing == ifc {
				LocalClientInterfaces = append(LocalClientInterfaces[:idx], LocalClientInterfaces[idx+1:]...)
				interfacesMu.Unlock()
				return
			}
		}
		for idx, existing := range Interfaces {
			if existing == ifc {
				Interfaces = append(Interfaces[:idx], Interfaces[idx+1:]...)
				interfacesMu.Unlock()
				return
			}
		}
		interfacesMu.Unlock()
	}
	ifaces.LocalInterfaceTeardownHandler = func(ifc *ifaces.Interface) {
		if ifc == nil {
			return
		}
		interfacesMu.Lock()
		for idx, existing := range LocalClientInterfaces {
			if existing == ifc {
				LocalClientInterfaces = append(LocalClientInterfaces[:idx], LocalClientInterfaces[idx+1:]...)
				break
			}
		}
		for idx, existing := range Interfaces {
			if existing == ifc {
				Interfaces = append(Interfaces[:idx], Interfaces[idx+1:]...)
				break
			}
		}
		interfacesMu.Unlock()
		if ifc.Parent != nil && Owner != nil {
			Owner.ShouldPersistData()
		}
	}
	ifaces.QueuedAnnounceLife = time.Duration(QUEUED_ANNOUNCE_LIFE) * time.Second
	ifaces.HeaderMinSize = HEADER_MINSIZE
	ifaces.TransportIdentityHashProvider = func() []byte {
		if TransportIdentity == nil {
			return nil
		}
		return TransportIdentity.Hash
	}
	ifaces.TunnelSynthesizer = func(ifc *ifaces.Interface) {
		SynthesizeTunnel(ifc)
	}
}
