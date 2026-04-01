package interfaces

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	vendor "github.com/svanichkin/go-reticulum/rns/vendor"
)

var localClientSeq atomic.Int64

const localInterfaceHWMTU = 262144

type LocalConfig struct {
	UseAFUnix          bool
	LocalSocketPath    string
	LocalInterfacePort int
	PktAddrOverride    string // optional: "unix:/path" or "tcp:host:port"
	Parent             *Interface
	OnClientDisconnect func(*Interface)
}

// SharedConnectionDisappeared/Reappeared are optional callbacks set by rns package.
var SharedConnectionDisappeared func()
var SharedConnectionReappeared func()

func packetAddr(cfg LocalConfig) (network, addr string) {
	if cfg.PktAddrOverride != "" {
		if strings.HasPrefix(cfg.PktAddrOverride, "unix:") {
			return "unix", strings.TrimPrefix(cfg.PktAddrOverride, "unix:")
		}
		if strings.HasPrefix(cfg.PktAddrOverride, "tcp:") {
			return "tcp", strings.TrimPrefix(cfg.PktAddrOverride, "tcp:")
		}
	}

	socketName := strings.TrimSpace(cfg.LocalSocketPath)
	if socketName == "" {
		socketName = "default"
	}
	// Abstract AF_UNIX addresses ("\x00...") are only supported on Linux.
	if cfg.UseAFUnix && vendor.UseAFUnix() && runtime.GOOS == "linux" {
		// Python RNS uses "\x00rns/{name}" (no "/local" suffix) for the packet IPC socket.
		return "unix", "\x00rns/" + socketName
	}
	// Python falls back to TCP when AF_UNIX isn't used/available.
	if cfg.LocalInterfacePort > 0 {
		return "tcp", fmt.Sprintf("127.0.0.1:%d", cfg.LocalInterfacePort)
	}
	return "unix", filepath.Join(os.TempDir(), "rns_"+socketName+"_local.sock")
}

func StartLocalInterfaceServer(cfg LocalConfig, onNewClient func(*Interface)) (net.Listener, error) {
	network, addr := packetAddr(cfg)
	displayAddr := addr
	if strings.HasPrefix(displayAddr, "\x00") {
		displayAddr = strings.TrimPrefix(displayAddr, "\x00")
	}
	var ln net.Listener
	var err error
	if network == "unix" && strings.HasPrefix(addr, "\x00") {
		ln, err = net.Listen("unix", addr)
	} else if network == "unix" {
		_ = os.Remove(addr)
		_ = os.MkdirAll(filepath.Dir(addr), 0o755)
		ln, err = net.Listen("unix", addr)
	} else {
		ln, err = listenWithReuseAddr("tcp", addr)
	}
	if err != nil {
		return nil, err
	}
	if onNewClient == nil {
		_ = ln.Close()
		return nil, errors.New("missing onNewClient callback")
	}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			if tc, ok := conn.(*net.TCPConn); ok {
				_ = tc.SetNoDelay(true)
			}
			id := localClientSeq.Add(1)
			ifName := ""
			if ta, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
				ifName = fmt.Sprintf("%d", ta.Port)
			} else if ua, ok := conn.RemoteAddr().(*net.UnixAddr); ok {
				_ = ua
				ifName = fmt.Sprintf("%d@%s", id, displayAddr)
			} else {
				ifName = fmt.Sprintf("%d", id)
			}
			ifc := &Interface{
				Name:              fmt.Sprintf("LocalInterface[%s]", ifName),
				Type:              "LocalInterface",
				IN:                true,
				OUT:               false,
				DriverImplemented: true,
				Online:            true,
				Bitrate:           1_000_000_000,
				HWMTU:             localInterfaceHWMTU,
				AutoconfigureMTU:  true,
			}
			if cfg.Parent != nil {
				ifc.Parent = cfg.Parent
				// Mirror Python: spawned interfaces inherit IN/OUT from the server interface.
				ifc.IN = cfg.Parent.IN
				ifc.OUT = cfg.Parent.OUT
				if cfg.Parent.Bitrate > 0 {
					ifc.Bitrate = cfg.Parent.Bitrate
				}
				ifc.ForceBitrateLatency = cfg.Parent.ForceBitrateLatency
			}
			ifc.OptimiseMTU()
			ifc.setLocalConn(conn)
			onNewClient(ifc)
			go func() {
				ifc.readLocalFramesLoop()
				ifc.setLocalConn(nil)
				ifc.Online = false
				if cfg.OnClientDisconnect != nil {
					cfg.OnClientDisconnect(ifc)
				}
			}()
		}
	}()

	return ln, nil
}

func ConnectLocalInterfaceClient(cfg LocalConfig, ifc *Interface) error {
	if ifc == nil {
		return errors.New("nil interface")
	}
	ifc.IN = true
	// Python Reticulum sets OUT=True for the LocalClientInterface used to connect to
	// the local shared instance.
	ifc.OUT = true
	if ifc.Bitrate == 0 {
		ifc.Bitrate = 1_000_000_000
	}
	if ifc.HWMTU == 0 {
		ifc.HWMTU = localInterfaceHWMTU
	}
	if !ifc.AutoconfigureMTU {
		ifc.AutoconfigureMTU = true
	}
	ifc.OptimiseMTU()

	connectOnce := func() (net.Conn, error) {
		network, addr := packetAddr(cfg)
		if network == "unix" && strings.HasPrefix(addr, "\x00") {
			return net.Dial("unix", addr)
		}
		if network == "unix" {
			return net.Dial("unix", addr)
		}
		c, err := net.Dial("tcp", addr)
		if err != nil {
			return nil, err
		}
		if tc, ok := c.(*net.TCPConn); ok {
			_ = tc.SetNoDelay(true)
		}
		return c, nil
	}

	conn, err := connectOnce()
	if err != nil {
		return err
	}
	ifc.setLocalConn(conn)

	go func() {
		for {
			ifc.readLocalFramesLoop()
			ifc.setLocalConn(nil)
			// Python only attempts reconnect for shared-instance clients.
			if !ifc.LocalIsSharedClient || ifc.Detached {
				ifc.Online = false
				return
			}
			if DiagLogf != nil {
				DiagLogf(LogWarning, "Socket for %s was closed, attempting to reconnect...", ifc)
			}
			if SharedConnectionDisappeared != nil {
				SharedConnectionDisappeared()
			}
			// Python LocalClientInterface.RECONNECT_WAIT = 8
			time.Sleep(8 * time.Second)
			if DiagLogf != nil {
				DiagLogf(LogVerbose, "Attempting to reconnect local interface for %s...", ifc)
			}
			newConn, err := connectOnce()
			if err != nil {
				if DiagLogf != nil {
					DiagLogf(LogDebug, "Connection attempt for %s failed: %v", ifc, err)
				}
				continue
			}
			ifc.setLocalConn(newConn)
			if DiagLogf != nil {
				DiagLogf(LogInfo, "Reconnected socket for %s.", ifc)
			}
			if SharedConnectionReappeared != nil {
				// Python schedules this as RECONNECT_WAIT+2 after reconnect.
				go func() {
					time.Sleep(10 * time.Second)
					SharedConnectionReappeared()
				}()
			}
		}
	}()

	return nil
}
