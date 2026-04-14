package interfaces

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	vendor "github.com/svanichkin/go-reticulum/rns/vendor"
)

// SpawnHandler is set by the rns package to register dynamically spawned sub-interfaces.
// This avoids import cycles.
var SpawnHandler func(ifc *Interface)

type autoConfig struct {
	Name                   string
	GroupID                []byte
	DiscoveryScope         string
	DiscoveryPort          int
	MulticastAddressType   string
	DataPort               int
	AllowedInterfaces      map[string]bool
	IgnoredInterfaces      map[string]bool
	ConfiguredBitrate      *int
	FixedMTU               bool
	HWMTU                  int
	PeeringTimeout         time.Duration
	AnnounceInterval       time.Duration
	ReversePeeringInterval time.Duration
	PeerJobInterval        time.Duration
	MultiIFDequeLen        int
	MultiIFDequeTTL        time.Duration
	MulticastEchoTimeout   time.Duration
	MulticastDiscoveryAddr string
}

type autoState struct {
	cfg autoConfig

	mu            sync.Mutex
	finalInitDone atomic.Bool
	carrierChanged atomic.Bool

	adopted map[string]*net.Interface // ifname -> interface
	llip    map[string]net.IP         // ifname -> link-local IPv6
	llstr   map[string]string         // ifname -> link-local string (post descopeLinkLocal)
	llset   map[string]bool           // set of link-local strings (like Python link_local_addresses)
	llrev   map[string]string         // link-local string -> ifname (like Python adopted_interfaces reverse lookup)
	peers   map[string]*autoPeerState // ip string -> state
	spawned map[string]*Interface     // ip string -> peer ifc
	discPC  map[string]net.PacketConn // ifname -> discovery PacketConn
	dataUC  map[string]*net.UDPConn   // ifname -> data UDPConn
	annStop map[string]chan struct{}  // ifname -> stop announce loop
	mcastE  map[string]time.Time      // ifname -> last multicast echo
	initE   map[string]time.Time      // ifname -> first multicast echo (only set once echo seen)
	timedOU map[string]bool           // ifname -> carrier timed out
	mifQ    []autoSeen                // ring buffer
	mifIdx  int
	outConn *net.UDPConn

	stopCh chan struct{}
	wg     sync.WaitGroup
}

type autoSeen struct {
	sum [32]byte
	ts  time.Time
}

type autoPeerState struct {
	addr         net.IP
	ifname       string
	lastSeen     time.Time
	lastOutbound time.Time
}

const (
	autoDefaultDiscoveryPort = 29716
	autoDefaultDataPort      = 42671
	autoDefaultIFACSize      = 16
	autoDefaultMTU           = 1196

	autoScopeLink         = "2"
	autoScopeAdmin        = "4"
	autoScopeSite         = "5"
	autoScopeOrganisation = "8"
	autoScopeGlobal       = "e"

	autoMcastPermanent = "0"
	autoMcastTemporary = "1"
)

func defaultAutoConfig(name string) autoConfig {
	cfg := autoConfig{
		Name:                 name,
		GroupID:              []byte("reticulum"),
		DiscoveryScope:       autoScopeLink,
		DiscoveryPort:        autoDefaultDiscoveryPort,
		MulticastAddressType: autoMcastTemporary,
		DataPort:             autoDefaultDataPort,
		AllowedInterfaces:    map[string]bool{},
		IgnoredInterfaces:    map[string]bool{},
		FixedMTU:             true,
		HWMTU:                autoDefaultMTU,
		PeeringTimeout:       22 * time.Second,
		AnnounceInterval:     time.Duration(float64(time.Second) * 1.6),
		ReversePeeringInterval: time.Duration(float64(time.Second) * 1.6 * 3.25),
		PeerJobInterval:      4 * time.Second,
		MultiIFDequeLen:      48,
		MultiIFDequeTTL:      time.Duration(float64(time.Second) * 0.75),
		MulticastEchoTimeout: time.Duration(float64(time.Second) * 6.5),
	}
	// Match Python: increase peering timeout on Android due to low-power modes.
	if vendor.IsAndroid() {
		cfg.PeeringTimeout = time.Duration(float64(cfg.PeeringTimeout) * 1.25)
	}
	return cfg
}

func (c *autoConfig) buildMulticastAddress() string {
	g := sha256.Sum256(c.GroupID)
	gt := "0"
	gt += ":" + fmt.Sprintf("%02x", uint16(g[2])<<8|uint16(g[3]))
	gt += ":" + fmt.Sprintf("%02x", uint16(g[4])<<8|uint16(g[5]))
	gt += ":" + fmt.Sprintf("%02x", uint16(g[6])<<8|uint16(g[7]))
	gt += ":" + fmt.Sprintf("%02x", uint16(g[8])<<8|uint16(g[9]))
	gt += ":" + fmt.Sprintf("%02x", uint16(g[10])<<8|uint16(g[11]))
	gt += ":" + fmt.Sprintf("%02x", uint16(g[12])<<8|uint16(g[13]))
	return "ff" + c.MulticastAddressType + c.DiscoveryScope + ":" + gt
}

func (i *Interface) ConfigureAutoInterface(cfg map[string]string) error {
	if i == nil {
		return errors.New("nil interface")
	}
	st := &autoState{
		cfg:     defaultAutoConfig(i.Name),
		adopted: map[string]*net.Interface{},
		llip:    map[string]net.IP{},
		llstr:   map[string]string{},
		llset:   map[string]bool{},
		llrev:   map[string]string{},
		peers:   map[string]*autoPeerState{},
		spawned: map[string]*Interface{},
		discPC:  map[string]net.PacketConn{},
		dataUC:  map[string]*net.UDPConn{},
		annStop: map[string]chan struct{}{},
		mcastE:  map[string]time.Time{},
		initE:   map[string]time.Time{},
		timedOU: map[string]bool{},
		stopCh:  make(chan struct{}),
	}

	if v := first(cfg, "group_id"); v != "" {
		st.cfg.GroupID = []byte(v)
	}
	if v := strings.ToLower(first(cfg, "discovery_scope")); v != "" {
		switch v {
		case "link":
			st.cfg.DiscoveryScope = autoScopeLink
		case "admin":
			st.cfg.DiscoveryScope = autoScopeAdmin
		case "site":
			st.cfg.DiscoveryScope = autoScopeSite
		case "organisation":
			st.cfg.DiscoveryScope = autoScopeOrganisation
		case "global":
			st.cfg.DiscoveryScope = autoScopeGlobal
		default:
			// accept raw nibble like Python
			st.cfg.DiscoveryScope = v
		}
	}
	if v := first(cfg, "discovery_port"); v != "" {
		if p, ok := parseInt(v); ok && p > 0 {
			st.cfg.DiscoveryPort = p
		}
	}
	if v := strings.TrimSpace(first(cfg, "multicast_address_type")); v != "" {
		switch strings.ToLower(v) {
		case "temporary":
			st.cfg.MulticastAddressType = autoMcastTemporary
		case "permanent":
			st.cfg.MulticastAddressType = autoMcastPermanent
		default:
			st.cfg.MulticastAddressType = v
		}
	}
	if v := first(cfg, "data_port"); v != "" {
		if p, ok := parseInt(v); ok && p > 0 {
			st.cfg.DataPort = p
		}
	}
	if v := first(cfg, "configured_bitrate"); v != "" {
		if b, ok := parseInt(v); ok && b > 0 {
			st.cfg.ConfiguredBitrate = &b
		}
	}

	for _, d := range list(cfg, "devices") {
		st.cfg.AllowedInterfaces[strings.TrimSpace(d)] = true
	}
	for _, d := range list(cfg, "ignored_devices") {
		st.cfg.IgnoredInterfaces[strings.TrimSpace(d)] = true
	}

	st.cfg.MulticastDiscoveryAddr = st.cfg.buildMulticastAddress()

	i.Type = "AutoInterface"
	i.IN = true
	i.OUT = false
	i.DriverImplemented = true
	i.Bitrate = 10 * 1000 * 1000
	if st.cfg.ConfiguredBitrate != nil {
		i.Bitrate = *st.cfg.ConfiguredBitrate
	}
	if i.IFACSize == 0 {
		i.IFACSize = autoDefaultIFACSize
	}
	i.Online = false
	i.auto = st
	return nil
}

func (i *Interface) StartAutoInterface() error {
	if i == nil || i.auto == nil {
		return errors.New("autointerface not configured")
	}
	st := i.auto
	st.finalInitDone.Store(false)

	peeringWait := time.Duration(float64(st.cfg.AnnounceInterval) * 1.2)
	if DiagLogf != nil {
		DiagLogf(LogVerbose, "%s discovering peers for %.2f seconds...", i, peeringWait.Seconds())
	}

	interfaces, err := net.Interfaces()
	if err != nil {
		return err
	}

	adopted := 0
	for _, nif := range interfaces {
		ifname := nif.Name
		if shouldIgnoreAutoInterface(ifname, st.cfg.AllowedInterfaces, st.cfg.IgnoredInterfaces) {
			if DiagLogf != nil {
				DiagLogf(LogExtreme, "%s skipping interface %s", i, ifname)
			}
			continue
		}
		ll, llStr := findLinkLocalIPv6(&nif)
		if ll == nil {
			if DiagLogf != nil {
				DiagLogf(LogExtreme, "%s no link-local IPv6 on %s; skipping", i, ifname)
			}
			continue
		}
		if err := st.startOrRestartInterface(i, nif, ll, llStr, false); err == nil {
			adopted++
		}
	}

	if adopted == 0 && DiagLogf != nil {
		DiagLogf(LogWarning, "%s could not autoconfigure. This interface currently provides no connectivity.", i)
	}

	// Start peer job loop even if we have zero adopted interfaces (Python still exists but offline).
	st.wg.Add(1)
	go func() {
		defer st.wg.Done()
		st.autoPeerJobsLoop(i)
	}()

	// In Python final_init waits ~announce_interval*1.2, then marks online.
	time.AfterFunc(peeringWait, func() {
		i.Online = true
		st.finalInitDone.Store(true)
	})

	return nil
}

func findLinkLocalIPv6(nif *net.Interface) (net.IP, string) {
	if nif == nil {
		return nil, ""
	}
	addrs, err := nif.Addrs()
	if err != nil {
		return nil, ""
	}
	for _, a := range addrs {
		// Preserve Python descope_linklocal() semantics:
		// - drop "%ifname" zone suffix (macOS)
		// - drop embedded scope specifier (NetBSD/OpenBSD)
		host := a.String()
		if slash := strings.IndexByte(host, '/'); slash >= 0 {
			host = host[:slash]
		}
		host = descopeLinkLocal(host)
		ip := net.ParseIP(host)
		if ip == nil {
			continue
		}
		ip = ip.To16()
		if ip == nil || ip.To4() != nil {
			continue
		}
		if strings.HasPrefix(strings.ToLower(host), "fe80:") {
			return ip, host
		}
	}
	return nil, ""
}

func descopeLinkLocal(linkLocalAddr string) string {
	// Drop scope specifier expressed as %ifname (macOS)
	if idx := strings.IndexByte(linkLocalAddr, '%'); idx >= 0 {
		linkLocalAddr = linkLocalAddr[:idx]
	}

	// Drop embedded scope specifier (NetBSD/OpenBSD), mirroring:
	// re.sub(r"fe80:[0-9a-f]*::","fe80::", link_local_addr)
	s := strings.ToLower(linkLocalAddr)
	if !strings.HasPrefix(s, "fe80:") {
		return linkLocalAddr
	}
	rest := s[len("fe80:"):]
	if rest == "" {
		return linkLocalAddr
	}
	dbl := strings.Index(rest, "::")
	if dbl <= 0 {
		return linkLocalAddr
	}
	seg := rest[:dbl]
	for i := 0; i < len(seg); i++ {
		c := seg[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return linkLocalAddr
		}
	}
	// Replace fe80:<hex>:: with fe80::
	return "fe80::" + linkLocalAddr[len("fe80:")+len(seg)+len("::"):]
}

func isClosedConnErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, net.ErrClosed) {
		return true
	}
	// Compat with older net errors.
	return strings.Contains(strings.ToLower(err.Error()), "closed network connection")
}

func listenDiscoveryPacket(mcastAddr string, linkScope bool, ifname string, port int) (net.PacketConn, error) {
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var ctrlErr error
			if err := c.Control(func(fd uintptr) {
				_ = setSockoptIntFD(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
				// Best-effort; may not exist on some platforms.
				_ = setSockoptIntFD(fd, syscall.SOL_SOCKET, 0x0200 /* SO_REUSEPORT */, 1)
			}); err != nil {
				ctrlErr = err
			}
			return ctrlErr
		},
	}

	// Windows cannot bind to multicast host or with interface specifier.
	if vendor.IsWindows() {
		return lc.ListenPacket(context.TODO(), "udp6", fmt.Sprintf("[::]:%d", port))
	}

	host := mcastAddr
	zone := ""
	if linkScope {
		zone = ifname
	}
	addr := (&net.UDPAddr{IP: net.ParseIP(host), Zone: zone, Port: port}).String()
	return lc.ListenPacket(context.TODO(), "udp6", addr)
}

func (st *autoState) stop() {
	if st == nil {
		return
	}
	select {
	case <-st.stopCh:
		return
	default:
		close(st.stopCh)
	}
	st.wg.Wait()
	if st.outConn != nil {
		_ = st.outConn.Close()
	}

	st.mu.Lock()
	spawned := make([]*Interface, 0, len(st.spawned))
	for _, peer := range st.spawned {
		if peer != nil {
			spawned = append(spawned, peer)
		}
	}
	st.spawned = make(map[string]*Interface)
	st.peers = make(map[string]*autoPeerState)
	st.mu.Unlock()

	for _, peer := range spawned {
		removeInterface(peer)
	}
}

func (st *autoState) stopInterface(ifname string) {
	if st == nil || strings.TrimSpace(ifname) == "" {
		return
	}
	var (
		pc      net.PacketConn
		uc      *net.UDPConn
		annStop chan struct{}
	)
	st.mu.Lock()
	pc = st.discPC[ifname]
	uc = st.dataUC[ifname]
	annStop = st.annStop[ifname]
	oldLLStr := st.llstr[ifname]
	delete(st.discPC, ifname)
	delete(st.dataUC, ifname)
	delete(st.annStop, ifname)
	delete(st.adopted, ifname)
	delete(st.llip, ifname)
	delete(st.llstr, ifname)
	if oldLLStr != "" {
		delete(st.llset, oldLLStr)
		delete(st.llrev, oldLLStr)
	}
	delete(st.mcastE, ifname)
	delete(st.initE, ifname)
	delete(st.timedOU, ifname)
	st.mu.Unlock()

	if DiagLogf != nil {
		DiagLogf(LogDebug, "AutoInterface stopping listeners on %s", ifname)
	}
	if annStop != nil {
		close(annStop)
	}
	if pc != nil {
		_ = pc.Close()
	}
	if uc != nil {
		_ = uc.Close()
	}
}

func (st *autoState) startOrRestartInterface(parent *Interface, nif net.Interface, ll net.IP, llStr string, force bool) error {
	if st == nil || parent == nil {
		return errors.New("autointerface: nil state")
	}
	ifname := nif.Name
	if strings.TrimSpace(ifname) == "" || ll == nil {
		return errors.New("autointerface: invalid interface parameters")
	}
	if strings.TrimSpace(llStr) == "" {
		llStr = ll.String()
	}

	st.mu.Lock()
	oldLL := st.llip[ifname]
	_, already := st.adopted[ifname]
	st.mu.Unlock()

	if already && !force && oldLL != nil && oldLL.Equal(ll) {
		return nil
	}
	if already {
		if DiagLogf != nil {
			DiagLogf(LogDebug, "%s restarting listeners on %s (link-local or carrier change)", parent, ifname)
		}
		st.stopInterface(ifname)
	}

	// Discovery multicast listener.
	pc, err := listenDiscoveryPacket(st.cfg.MulticastDiscoveryAddr, st.cfg.DiscoveryScope == autoScopeLink, ifname, st.cfg.DiscoveryPort)
	if err != nil {
		if DiagLogf != nil {
			DiagLogf(LogError, "%s could not open discovery socket on %s: %v", parent, ifname, err)
		}
		return err
	}
	if err := joinIPv6Multicast(pc, st.cfg.MulticastDiscoveryAddr, nif.Index); err != nil {
		_ = pc.Close()
		if DiagLogf != nil {
			DiagLogf(LogError, "%s could not join multicast group on %s: %v", parent, ifname, err)
		}
		return err
	}

	// Data socket per adopted interface.
	laddr := &net.UDPAddr{IP: ll, Zone: ifname, Port: st.cfg.DataPort}
	uc, err := net.ListenUDP("udp6", laddr)
	if err != nil {
		_ = pc.Close()
		if DiagLogf != nil {
			DiagLogf(LogError, "%s could not open data socket on %s: %v", parent, ifname, err)
		}
		return err
	}

	if DiagLogf != nil {
		DiagLogf(LogExtreme, "%s Selecting link-local address %s for interface %s", parent, llStr, ifname)
	}

	annStop := make(chan struct{})
	now := time.Now()
	nifCopy := nif
	st.mu.Lock()
	st.adopted[ifname] = &nifCopy
	st.llip[ifname] = ll
	st.llstr[ifname] = llStr
	st.llset[llStr] = true
	st.llrev[llStr] = ifname
	st.discPC[ifname] = pc
	st.dataUC[ifname] = uc
	st.annStop[ifname] = annStop
	st.mcastE[ifname] = now
	st.mu.Unlock()

	st.wg.Add(1)
	go func(ifname string, pc net.PacketConn, annStop <-chan struct{}) {
		defer st.wg.Done()
		defer pc.Close()
		st.autoDiscoveryLoop(parent, ifname, pc, annStop)
	}(ifname, pc, annStop)

	st.wg.Add(1)
	go func(ifname string, c *net.UDPConn) {
		defer st.wg.Done()
		defer c.Close()
		st.autoDataLoop(parent, ifname, c)
	}(ifname, uc)

	return nil
}

func (st *autoState) autoDiscoveryLoop(parent *Interface, ifname string, pc net.PacketConn, annStop <-chan struct{}) {
	// Announce loop.
	st.wg.Add(1)
	go func() {
		defer st.wg.Done()
		t := time.NewTicker(st.cfg.AnnounceInterval)
		defer t.Stop()
		for {
			select {
			case <-st.stopCh:
				return
			case <-annStop:
				return
			case <-t.C:
				st.sendDiscoveryAnnounce(ifname)
			}
		}
	}()

	buf := make([]byte, 2048)
	for {
		_ = pc.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, addr, err := pc.ReadFrom(buf)
		select {
		case <-st.stopCh:
			return
		default:
		}
		if err != nil {
			if isClosedConnErr(err) {
				return
			}
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			continue
		}
		if n < sha256.Size {
			continue
		}
		// Python ignores discovery packets until final_init_done.
		if !st.finalInitDone.Load() {
			continue
		}
		ua, ok := addr.(*net.UDPAddr)
		if !ok || ua.IP == nil {
			continue
		}
		peerStr := ua.IP.String()
		peerIP := ua.IP.To16()
		if peerIP == nil {
			continue
		}
		// Python uses ipv6_src[0] string as received from OS for the token hash.
		expected := sha256.Sum256(append(append([]byte{}, st.cfg.GroupID...), []byte(peerStr)...))
		if !equal32(buf[:sha256.Size], expected[:]) {
			if DiagLogf != nil {
				DiagLogf(LogDebug, "%s received peering packet on %s from %s, but authentication hash was incorrect", parent, ifname, peerStr)
			}
			continue
		}
		st.addOrRefreshPeer(parent, peerIP, peerStr, ifname)
	}
}

func (st *autoState) autoDataLoop(parent *Interface, ifname string, c *net.UDPConn) {
	buf := make([]byte, MaxFrameLength)
	for {
		_ = c.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, from, err := c.ReadFromUDPAddrPort(buf)
		select {
		case <-st.stopCh:
			return
		default:
		}
		if err != nil {
			if isClosedConnErr(err) {
				return
			}
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			continue
		}
		if n <= 0 {
			continue
		}
		ip := from.Addr().AsSlice()
		peerIP := net.IP(ip).To16()
		if peerIP == nil {
			continue
		}
		peerKey := peerIP.String()

		st.mu.Lock()
		peerIfc := st.spawned[peerKey]
		st.mu.Unlock()
		if peerIfc == nil || !peerIfc.Online || !parent.Online {
			continue
		}

		// Multi-interface de-duplication.
		sum := sha256.Sum256(buf[:n])
		if st.seenRecently(sum) {
			continue
		}
		st.markSeen(sum)

		st.addOrRefreshPeer(parent, peerIP, peerKey, ifname)

		atomic.AddUint64(&peerIfc.RXB, uint64(n))
		atomic.AddUint64(&parent.RXB, uint64(n))
		if InboundHandler != nil {
			InboundHandler(append([]byte{}, buf[:n]...), peerIfc)
		}
	}
}

func (st *autoState) autoPeerJobsLoop(parent *Interface) {
	t := time.NewTicker(st.cfg.PeerJobInterval)
	defer t.Stop()
	for {
		select {
		case <-st.stopCh:
			return
		case <-t.C:
			now := time.Now()
			var toRemove []string
			var dead []*Interface
			var checkIfs []string
			oldLL := map[string]net.IP{}
			oldLLStr := map[string]string{}
			prevTO := map[string]bool{}
			curTO := map[string]bool{}
			echoSeen := map[string]bool{}
			st.mu.Lock()
			for k, p := range st.peers {
				if now.After(p.lastSeen.Add(st.cfg.PeeringTimeout)) {
					toRemove = append(toRemove, k)
				}
			}
			for ifname := range st.adopted {
				prevTO[ifname] = st.timedOU[ifname]
				last := st.mcastE[ifname]
				timedOut := true
				if !last.IsZero() && now.Sub(last) <= st.cfg.MulticastEchoTimeout {
					timedOut = false
				}
				st.timedOU[ifname] = timedOut
				curTO[ifname] = timedOut
				echoSeen[ifname] = !st.initE[ifname].IsZero()
				checkIfs = append(checkIfs, ifname)
				if st.llip[ifname] != nil {
					oldLL[ifname] = append(net.IP(nil), st.llip[ifname]...)
				}
				if st.llstr[ifname] != "" {
					oldLLStr[ifname] = st.llstr[ifname]
				}
			}
			for _, k := range toRemove {
				ifc := st.spawned[k]
				if ifc != nil {
					dead = append(dead, ifc)
				}
				delete(st.spawned, k)
				delete(st.peers, k)
			}
			for _, p := range st.peers {
				if now.After(p.lastOutbound.Add(st.cfg.ReversePeeringInterval)) {
					p.lastOutbound = now
					peerIP := append(net.IP(nil), p.addr...)
					peerIf := p.ifname
					go st.reverseAnnounce(peerIf, peerIP)
				}
			}
			st.mu.Unlock()
			for _, ifc := range dead {
				removeInterface(ifc)
			}

			// Check that the link-local address has not changed, and if it has,
			// reconfigure per-interface sockets like Python peer_jobs.
			for _, ifname := range checkIfs {
				nif, err := net.InterfaceByName(ifname)
				if err != nil || nif == nil {
					continue
				}
				newLL, newLLStr := findLinkLocalIPv6(nif)
				prev := oldLL[ifname]
				wasTimedOut := prevTO[ifname]
				isTimedOut := curTO[ifname]
				if newLL == nil {
					// Interface lost link-local; stop using it.
					st.stopInterface(ifname)
					continue
				}
				if prev == nil || !prev.Equal(newLL) {
					if DiagLogf != nil {
						prevStr := oldLLStr[ifname]
						if prevStr != "" && newLLStr != "" && prevStr != newLLStr {
							DiagLogf(LogDebug, "Replacing link-local address %s for %s with %s", prevStr, ifname, newLLStr)
						}
					}
					_ = st.startOrRestartInterface(parent, *nif, newLL, newLLStr, true)
					st.carrierChanged.Store(true)
					continue
				}

				if DiagLogf != nil && wasTimedOut != isTimedOut {
					if isTimedOut {
						DiagLogf(LogWarning, "Multicast echo timeout for %s. Carrier lost.", ifname)
					} else {
						DiagLogf(LogWarning, "%s Carrier recovered on %s", parent, ifname)
					}
					st.carrierChanged.Store(true)
				}

				if DiagLogf != nil && !echoSeen[ifname] {
					DiagLogf(LogError, "%s No multicast echoes received on %s. The networking hardware or a firewall may be blocking multicast traffic.", parent, ifname)
				}

				// Python does not restart listeners on carrier-timeout transitions.
				_ = wasTimedOut
			}
		}
	}
}

func (st *autoState) sendDiscoveryAnnounce(ifname string) {
	st.mu.Lock()
	ll := st.llip[ifname]
	llStr := st.llstr[ifname]
	timedOut := st.timedOU[ifname]
	st.mu.Unlock()
	if ll == nil || strings.TrimSpace(llStr) == "" {
		return
	}
	// Python hashes the link-local address string from OS enumeration after descope_linklocal().
	token := sha256.Sum256(append(append([]byte{}, st.cfg.GroupID...), []byte(llStr)...))
	dst := &net.UDPAddr{IP: net.ParseIP(st.cfg.MulticastDiscoveryAddr), Zone: ifname, Port: st.cfg.DiscoveryPort}

	conn, err := net.DialUDP("udp6", nil, dst)
	if err != nil {
		if DiagLogf != nil && !timedOut {
			DiagLogf(LogWarning, "%s detected possible carrier loss on %s: %v", st.cfg.Name, ifname, err)
		}
		return
	}
	if _, err := conn.Write(token[:]); err != nil {
		if DiagLogf != nil && !timedOut {
			DiagLogf(LogWarning, "%s detected possible carrier loss on %s: %v", st.cfg.Name, ifname, err)
		}
	}
	_ = conn.Close()
}

func (st *autoState) reverseAnnounce(ifname string, peerIP net.IP) {
	st.mu.Lock()
	ll := st.llip[ifname]
	st.mu.Unlock()
	if ll == nil || peerIP == nil {
		return
	}
	dst := &net.UDPAddr{IP: peerIP.To16(), Zone: ifname, Port: st.cfg.DiscoveryPort + 1}
	if dst.IP == nil {
		return
	}
	conn, err := net.DialUDP("udp6", nil, dst)
	if err != nil {
		if DiagLogf != nil {
			DiagLogf(LogError, "Could not send reverse peering packet to %s on %s: %v", peerIP.String(), ifname, err)
		}
		return
	}
	defer conn.Close()
	token := sha256.Sum256(append(append([]byte{}, st.cfg.GroupID...), []byte(ll.String())...))
	_, _ = conn.Write(token[:])
}

func (st *autoState) addOrRefreshPeer(parent *Interface, peerIP net.IP, peerStr string, ifname string) {
	if parent == nil {
		return
	}
	if strings.TrimSpace(peerStr) == "" {
		peerStr = peerIP.String()
	}
	key := peerStr
	now := time.Now()

	var retired *Interface

	st.mu.Lock()
	// Python treats discovery packets sourced from our own link-local address
	// as multicast echoes.
	if st.llset[key] {
		if echoIf := st.llrev[key]; echoIf != "" {
			st.mcastE[echoIf] = now
			if _, ok := st.initE[echoIf]; !ok {
				st.initE[echoIf] = now
			}
			st.mu.Unlock()
			return
		}
		if DiagLogf != nil {
			DiagLogf(LogWarning, "%s received multicast echo on unexpected interface %s", parent, ifname)
		}
		st.mu.Unlock()
		return
	}

	if p, ok := st.peers[key]; ok {
		p.lastSeen = now
		st.mu.Unlock()
		return
	}

	if existing := st.spawned[key]; existing != nil {
		retired = existing
		delete(st.spawned, key)
	}

	st.peers[key] = &autoPeerState{addr: peerIP, ifname: ifname, lastSeen: now, lastOutbound: now}

	peerIfc := &Interface{
		Name:                  fmt.Sprintf("AutoInterfacePeer[%s/%s]", ifname, key),
		Type:                  "AutoInterfacePeer",
		Parent:                parent,
		IN:                    parent.IN,
		OUT:                   parent.OUT,
		Mode:                  parent.Mode,
		Bitrate:               parent.Bitrate,
		IngressControl:        parent.IngressControl,
		ICMaxHeldAnnounces:    parent.ICMaxHeldAnnounces,
		ICBurstHold:           parent.ICBurstHold,
		ICBurstFreqNew:        parent.ICBurstFreqNew,
		ICBurstFreq:           parent.ICBurstFreq,
		ICNewTime:             parent.ICNewTime,
		ICBurstPenalty:        parent.ICBurstPenalty,
		ICHeldReleaseInterval: parent.ICHeldReleaseInterval,
		AnnounceCap:           parent.AnnounceCap,
		AnnounceRateTarget:    parent.AnnounceRateTarget,
		AnnounceRateGrace:     parent.AnnounceRateGrace,
		AnnounceRatePenalty:   parent.AnnounceRatePenalty,
		HWMTU:                 parent.HWMTU,
		AutoconfigureMTU:      parent.AutoconfigureMTU,
		FixedMTU:              parent.FixedMTU,
		DriverImplemented:     true,
		Online:                true,
		Created:               time.Now(),
	}
	peerIfc.IFACSize = parent.IFACSize
	peerIfc.IFACKey = parent.IFACKey
	peerIfc.IFACIdentity = parent.IFACIdentity
	peerIfc.IFACSignature = parent.IFACSignature
	peerIfc.IFACNetnameVal = parent.IFACNetnameVal
	peerIfc.IFACNetkeyVal = parent.IFACNetkeyVal

	st.spawned[key] = peerIfc
	st.mu.Unlock()

	if retired != nil {
		removeInterface(retired)
	}
	if SpawnHandler != nil {
		SpawnHandler(peerIfc)
	}
	if DiagLogf != nil {
		DiagLogf(LogDebug, "%s added peer %s on %s", parent, key, ifname)
	}
}

func (st *autoState) seenRecently(sum [32]byte) bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	now := time.Now()
	for _, s := range st.mifQ {
		if s.sum == sum && now.Before(s.ts.Add(st.cfg.MultiIFDequeTTL)) {
			return true
		}
	}
	return false
}

func (st *autoState) markSeen(sum [32]byte) {
	st.mu.Lock()
	defer st.mu.Unlock()
	now := time.Now()
	if len(st.mifQ) < st.cfg.MultiIFDequeLen {
		st.mifQ = append(st.mifQ, autoSeen{sum: sum, ts: now})
		return
	}
	st.mifQ[st.mifIdx] = autoSeen{sum: sum, ts: now}
	st.mifIdx = (st.mifIdx + 1) % st.cfg.MultiIFDequeLen
}

func (st *autoState) sendToPeer(peer *Interface, data []byte) {
	if peer == nil || peer.Parent == nil || peer.Parent.auto == nil {
		return
	}
	parent := peer.Parent
	st = parent.auto
	if st == nil || len(data) == 0 {
		return
	}
	ipStr := ""
	if strings.HasPrefix(peer.Name, "AutoInterfacePeer[") {
		parts := strings.TrimSuffix(strings.TrimPrefix(peer.Name, "AutoInterfacePeer["), "]")
		p := strings.SplitN(parts, "/", 2)
		if len(p) == 2 {
			ipStr = p[1]
		}
	}
	if ipStr == "" {
		return
	}

	st.mu.Lock()
	ps := st.peers[ipStr]
	st.mu.Unlock()
	if ps == nil {
		return
	}
	dst := &net.UDPAddr{IP: ps.addr, Zone: ps.ifname, Port: st.cfg.DataPort}

	st.mu.Lock()
	oc := st.outConn
	if oc == nil {
		oc, _ = net.ListenUDP("udp6", nil)
		st.outConn = oc
	}
	st.mu.Unlock()
	if oc == nil {
		return
	}

	_, _ = oc.WriteToUDP(data, dst)
	atomic.AddUint64(&peer.TXB, uint64(len(data)))
	atomic.AddUint64(&parent.TXB, uint64(len(data)))
}

func shouldIgnoreAutoInterface(ifname string, allowed, ignored map[string]bool) bool {
	if ifname == "" {
		return true
	}
	if ignored[ifname] {
		return true
	}
	// Python has platform specific defaults; we replicate minimal common ones.
	if ifname == "lo0" || ifname == "lo" {
		if !allowed[ifname] {
			return true
		}
	}
	if vendor.IsDarwin() {
		switch ifname {
		case "awdl0", "llw0", "en5":
			if !allowed[ifname] {
				return true
			}
		}
	}
	if vendor.IsAndroid() {
		switch ifname {
		case "dummy0", "tun0":
			if !allowed[ifname] {
				return true
			}
		}
	}
	if len(allowed) > 0 && !allowed[ifname] {
		return true
	}
	return false
}

func joinIPv6Multicast(pc net.PacketConn, group string, ifIndex int) error {
	c, ok := pc.(*net.UDPConn)
	if !ok {
		return errors.New("unsupported PacketConn")
	}
	raw, err := c.SyscallConn()
	if err != nil {
		return err
	}
	var sockErr error
	err = raw.Control(func(fd uintptr) {
		ip := net.ParseIP(group)
		if ip == nil {
			sockErr = errors.New("invalid multicast group")
			return
		}
		var mreq syscall.IPv6Mreq
		copy(mreq.Multiaddr[:], ip.To16())
		mreq.Interface = uint32(ifIndex)
		sockErr = setSockoptIPv6MreqFD(fd, syscall.IPPROTO_IPV6, syscall.IPV6_JOIN_GROUP, &mreq)
	})
	if err != nil {
		return err
	}
	return sockErr
}

func equal32(a, b []byte) bool {
	if len(a) != 32 || len(b) != 32 {
		return false
	}
	for i := 0; i < 32; i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func first(m map[string]string, key string) string {
	if m == nil {
		return ""
	}
	// ConfigObj style sometimes uses different casings
	for k, v := range m {
		if strings.EqualFold(k, key) {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func list(m map[string]string, key string) []string {
	if m == nil {
		return nil
	}
	var res []string
	for k, vals := range m {
		if !strings.EqualFold(k, key) {
			continue
		}
		// accept comma-separated too
		for _, part := range strings.Split(vals, ",") {
			p := strings.TrimSpace(part)
			if p != "" {
				res = append(res, p)
			}
		}
	}
	return res
}

func parseInt(s string) (int, bool) {
	var n int
	_, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &n)
	return n, err == nil
}
