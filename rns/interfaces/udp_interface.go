package interfaces

import (
	"fmt"
	"net"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

// Minimal UDP outgoing support mirroring the old rns/interface_udp.go behaviour.
// Full parity with Python UDPInterface lives in rns/interfaces/UDPInterface.py.

func (i *Interface) udpProcessOutgoing(data []byte) {
	if i == nil || len(data) == 0 {
		return
	}
	if i.udpForwardAddr == nil {
		return
	}

	// Python parity: process_outgoing uses a fresh UDP socket each time and does
	// not depend on the receive socket being active.
	conn := i.udpConn
	ephemeral := false
	if conn == nil {
		c, err := net.ListenUDP("udp", nil)
		if err != nil {
			return
		}
		enableUDPBroadcast(c)
		conn = c
		ephemeral = true
	}

	_, _ = conn.WriteToUDP(data, i.udpForwardAddr)
	if ephemeral {
		_ = conn.Close()
	}
	atomic.AddUint64(&i.TXB, uint64(len(data)))
	if parent := i.Parent; parent != nil {
		atomic.AddUint64(&parent.TXB, uint64(len(data)))
	}
}

// ConfigureUDP mirrors the Go-port config format used in reticulum.go.
func (i *Interface) ConfigureUDP(listenIP string, listenPort int, forwardIP string, forwardPort int) error {
	if i == nil {
		return nil
	}
	if strings.TrimSpace(i.Type) == "" {
		i.Type = "UDPInterface"
	}

	// Python parity: allow forward-only interfaces (no bind/listen).
	if listenPort > 0 {
		if strings.TrimSpace(listenIP) == "" {
			listenIP = "0.0.0.0"
		}
		bindAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(listenIP, fmt.Sprint(listenPort)))
		if err != nil {
			return err
		}
		i.udpBindAddr = bindAddr
	} else {
		i.udpBindAddr = nil
	}

	// Forward config can exist with or without a bind port.
	if forwardPort <= 0 {
		forwardPort = listenPort
	}
	if forwardPort > 0 {
		if strings.TrimSpace(forwardIP) == "" {
			forwardIP = "255.255.255.255"
		}
		fwdAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(forwardIP, fmt.Sprint(forwardPort)))
		if err != nil {
			return err
		}
		i.udpForwardAddr = fwdAddr
	} else {
		i.udpForwardAddr = nil
	}
	return nil
}

func (i *Interface) StartUDP() error {
	if i == nil || i.udpBindAddr == nil {
		return nil
	}
	conn, err := net.ListenUDP("udp", i.udpBindAddr)
	if err != nil {
		return err
	}
	_ = conn.SetWriteBuffer(1 << 20)
	_ = conn.SetReadBuffer(1 << 20)
	enableUDPBroadcast(conn)
	i.udpConn = conn
	i.Online = true
	go i.udpReadLoop()
	i.udpConn = conn
	return nil
}

func (i *Interface) udpReadLoop() {
	bufSize := 4096
	if i != nil && i.HWMTU > bufSize {
		bufSize = i.HWMTU
	}
	buf := make([]byte, bufSize)
	for i.Online {
		i.udpConn.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
		n, _, err := i.udpConn.ReadFromUDP(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			return
		}
		if n == 0 {
			continue
		}
		data := append([]byte(nil), buf[:n]...)
		atomic.AddUint64(&i.RXB, uint64(len(data)))
		if parent := i.Parent; parent != nil {
			atomic.AddUint64(&parent.RXB, uint64(len(data)))
		}
		if InboundHandler != nil {
			InboundHandler(data, i)
		}
	}
}

func (i *Interface) StopUDP() {
	if i.udpConn != nil {
		_ = i.udpConn.Close()
		i.udpConn = nil
		i.Online = false
	}
}

func enableUDPBroadcast(conn *net.UDPConn) {
	if conn == nil {
		return
	}
	rawConn, err := conn.SyscallConn()
	if err != nil {
		return
	}
	_ = rawConn.Control(func(fd uintptr) {
		_ = setSockoptIntFD(fd, syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
	})
}
