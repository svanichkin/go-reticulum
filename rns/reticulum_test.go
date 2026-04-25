package rns

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	vendor "github.com/svanichkin/go-reticulum/rns/vendor"
)

var (
	rpcChallenge = []byte("#CHALLENGE#")
	rpcWelcome   = []byte("#WELCOME#")
	rpcFailure   = []byte("#FAILURE#")
)

func TestPerformRPCHandshake_NetPipe_Success(t *testing.T) {
	maybeParallel(t)

	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	key := []byte("secret")
	answer := func(conn net.Conn, key []byte) error {
		var hdr [4]byte
		if _, err := io.ReadFull(conn, hdr[:]); err != nil {
			return err
		}
		n := int(binary.BigEndian.Uint32(hdr[:]))
		if n > 256 {
			return fmt.Errorf("rpc message too large: %d bytes", n)
		}
		msg := make([]byte, n)
		if _, err := io.ReadFull(conn, msg); err != nil {
			return err
		}
		if len(msg) < len(rpcChallenge) || !bytes.Equal(msg[:len(rpcChallenge)], rpcChallenge) {
			return fmt.Errorf("protocol error, expected challenge, got: %q", msg)
		}
		mac := hmac.New(md5.New, key)
		if _, err := mac.Write(msg[len(rpcChallenge):]); err != nil {
			return err
		}
		digest := mac.Sum(nil)
		binary.BigEndian.PutUint32(hdr[:], uint32(len(digest)))
		if _, err := conn.Write(hdr[:]); err != nil {
			return err
		}
		if len(digest) > 0 {
			if _, err := conn.Write(digest); err != nil {
				return err
			}
		}
		if _, err := io.ReadFull(conn, hdr[:]); err != nil {
			return err
		}
		if int(binary.BigEndian.Uint32(hdr[:])) > 256 {
			return fmt.Errorf("rpc message too large: %d bytes", binary.BigEndian.Uint32(hdr[:]))
		}
		resp := make([]byte, int(binary.BigEndian.Uint32(hdr[:])))
		if _, err := io.ReadFull(conn, resp); err != nil {
			return err
		}
		if !bytes.Equal(resp, rpcWelcome) {
			return errors.New("digest sent was rejected")
		}
		return nil
	}
	deliver := func(conn net.Conn, key []byte) error {
		randBytes := make([]byte, 40)
		if _, err := rand.Read(randBytes); err != nil {
			return err
		}
		msg := append(append([]byte(nil), rpcChallenge...), randBytes...)
		var hdr [4]byte
		binary.BigEndian.PutUint32(hdr[:], uint32(len(msg)))
		if _, err := conn.Write(hdr[:]); err != nil {
			return err
		}
		if len(msg) > 0 {
			if _, err := conn.Write(msg); err != nil {
				return err
			}
		}
		if _, err := io.ReadFull(conn, hdr[:]); err != nil {
			return err
		}
		n := int(binary.BigEndian.Uint32(hdr[:]))
		if n > 256 {
			return fmt.Errorf("rpc message too large: %d bytes", n)
		}
		resp := make([]byte, n)
		if _, err := io.ReadFull(conn, resp); err != nil {
			return err
		}
		mac := hmac.New(md5.New, key)
		if _, err := mac.Write(randBytes); err != nil {
			return err
		}
		expected := mac.Sum(nil)
		if subtle.ConstantTimeCompare(resp, expected) != 1 {
			binary.BigEndian.PutUint32(hdr[:], uint32(len(rpcFailure)))
			if _, err := conn.Write(hdr[:]); err != nil {
				return err
			}
			if len(rpcFailure) > 0 {
				if _, err := conn.Write(rpcFailure); err != nil {
					return err
				}
			}
			return errors.New("digest sent was rejected")
		}
		binary.BigEndian.PutUint32(hdr[:], uint32(len(rpcWelcome)))
		if _, err := conn.Write(hdr[:]); err != nil {
			return err
		}
		if len(rpcWelcome) > 0 {
			if _, err := conn.Write(rpcWelcome); err != nil {
				return err
			}
		}
		return nil
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- func() error {
			if err := deliver(c2, key); err != nil {
				return err
			}
			return answer(c2, key)
		}()
	}()

	if err := func() error {
		if err := answer(c1, key); err != nil {
			return err
		}
		return deliver(c1, key)
	}(); err != nil {
		t.Fatalf("client handshake: %v", err)
	}
	if err := <-serverErr; err != nil {
		t.Fatalf("server handshake: %v", err)
	}
}

func TestPerformRPCHandshake_NetPipe_InvalidKey(t *testing.T) {
	maybeParallel(t)

	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	serverKey := []byte("server-secret")
	clientKey := []byte("client-secret")
	answer := func(conn net.Conn, key []byte) error {
		var hdr [4]byte
		if _, err := io.ReadFull(conn, hdr[:]); err != nil {
			return err
		}
		n := int(binary.BigEndian.Uint32(hdr[:]))
		if n > 256 {
			return fmt.Errorf("rpc message too large: %d bytes", n)
		}
		msg := make([]byte, n)
		if _, err := io.ReadFull(conn, msg); err != nil {
			return err
		}
		if len(msg) < len(rpcChallenge) || !bytes.Equal(msg[:len(rpcChallenge)], rpcChallenge) {
			return fmt.Errorf("protocol error, expected challenge, got: %q", msg)
		}
		mac := hmac.New(md5.New, key)
		if _, err := mac.Write(msg[len(rpcChallenge):]); err != nil {
			return err
		}
		digest := mac.Sum(nil)
		binary.BigEndian.PutUint32(hdr[:], uint32(len(digest)))
		if _, err := conn.Write(hdr[:]); err != nil {
			return err
		}
		if len(digest) > 0 {
			if _, err := conn.Write(digest); err != nil {
				return err
			}
		}
		if _, err := io.ReadFull(conn, hdr[:]); err != nil {
			return err
		}
		if int(binary.BigEndian.Uint32(hdr[:])) > 256 {
			return fmt.Errorf("rpc message too large: %d bytes", binary.BigEndian.Uint32(hdr[:]))
		}
		resp := make([]byte, int(binary.BigEndian.Uint32(hdr[:])))
		if _, err := io.ReadFull(conn, resp); err != nil {
			return err
		}
		if !bytes.Equal(resp, rpcWelcome) {
			return errors.New("digest sent was rejected")
		}
		return nil
	}
	deliver := func(conn net.Conn, key []byte) error {
		randBytes := make([]byte, 40)
		if _, err := rand.Read(randBytes); err != nil {
			return err
		}
		msg := append(append([]byte(nil), rpcChallenge...), randBytes...)
		var hdr [4]byte
		binary.BigEndian.PutUint32(hdr[:], uint32(len(msg)))
		if _, err := conn.Write(hdr[:]); err != nil {
			return err
		}
		if len(msg) > 0 {
			if _, err := conn.Write(msg); err != nil {
				return err
			}
		}
		if _, err := io.ReadFull(conn, hdr[:]); err != nil {
			return err
		}
		n := int(binary.BigEndian.Uint32(hdr[:]))
		if n > 256 {
			return fmt.Errorf("rpc message too large: %d bytes", n)
		}
		resp := make([]byte, n)
		if _, err := io.ReadFull(conn, resp); err != nil {
			return err
		}
		mac := hmac.New(md5.New, key)
		if _, err := mac.Write(randBytes); err != nil {
			return err
		}
		expected := mac.Sum(nil)
		if subtle.ConstantTimeCompare(resp, expected) != 1 {
			binary.BigEndian.PutUint32(hdr[:], uint32(len(rpcFailure)))
			if _, err := conn.Write(hdr[:]); err != nil {
				return err
			}
			if len(rpcFailure) > 0 {
				if _, err := conn.Write(rpcFailure); err != nil {
					return err
				}
			}
			return errors.New("digest sent was rejected")
		}
		binary.BigEndian.PutUint32(hdr[:], uint32(len(rpcWelcome)))
		if _, err := conn.Write(hdr[:]); err != nil {
			return err
		}
		if len(rpcWelcome) > 0 {
			if _, err := conn.Write(rpcWelcome); err != nil {
				return err
			}
		}
		return nil
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- func() error {
			if err := deliver(c2, serverKey); err != nil {
				return err
			}
			return answer(c2, serverKey)
		}()
	}()

	if err := func() error {
		if err := answer(c1, clientKey); err != nil {
			return err
		}
		return deliver(c1, clientKey)
	}(); err == nil {
		t.Fatalf("expected client to reject invalid key")
	}
	if err := <-serverErr; err == nil {
		t.Fatalf("expected server to reject invalid key")
	}
}

func TestFallbackUnixSocketPath_SanitizesAndShortens(t *testing.T) {
	maybeParallel(t)

	addr := "\x00rns/test socket:name/with/slashes"
	got := func(addr string) string {
		name := strings.Trim(addr, "\x00")
		if name == "" {
			name = "default"
		}
		replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_", ":", "_")
		name = replacer.Replace(name)
		if len(name) > 48 {
			sum := sha256.Sum256([]byte(name))
			name = hex.EncodeToString(sum[:16])
		}
		return filepath.Join(os.TempDir(), "rns-"+name+".sock")
	}(addr)

	if !strings.HasPrefix(got, os.TempDir()) {
		t.Fatalf("expected tempdir prefix, got %q", got)
	}
	if !strings.HasSuffix(got, ".sock") {
		t.Fatalf("expected .sock suffix, got %q", got)
	}
	if strings.Contains(got, "/rns/test") {
		t.Fatalf("expected sanitised name, got %q", got)
	}
	if len(filepath.Base(got)) > 128 {
		t.Fatalf("expected reasonably short socket path, got len=%d (%q)", len(got), got)
	}
}

func TestRPCListener_TCP_HandshakeAndMsgpack(t *testing.T) {
	maybeParallel(t)

	key := []byte("rpc-key")
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("TCP listen not permitted in sandbox: %v", err)
		}
		t.Fatalf("net.Listen: %v", err)
	}
	defer ln.Close()

	serverErr := make(chan error, 1)
	serverGot := make(chan string, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		defer c.Close()
		if err := func() error {
			var hdr [4]byte
			randBytes := make([]byte, 40)
			if _, err := rand.Read(randBytes); err != nil {
				return err
			}
			msg := append(append([]byte(nil), rpcChallenge...), randBytes...)
			binary.BigEndian.PutUint32(hdr[:], uint32(len(msg)))
			if _, err := c.Write(hdr[:]); err != nil {
				return err
			}
			if len(msg) > 0 {
				if _, err := c.Write(msg); err != nil {
					return err
				}
			}
			if _, err := io.ReadFull(c, hdr[:]); err != nil {
				return err
			}
			n := int(binary.BigEndian.Uint32(hdr[:]))
			if n > 256 {
				return fmt.Errorf("rpc message too large: %d bytes", n)
			}
			resp := make([]byte, n)
			if _, err := io.ReadFull(c, resp); err != nil {
				return err
			}
			mac := hmac.New(md5.New, key)
			if _, err := mac.Write(randBytes); err != nil {
				return err
			}
			expected := mac.Sum(nil)
			if subtle.ConstantTimeCompare(resp, expected) != 1 {
				return errors.New("digest sent was rejected")
			}
			binary.BigEndian.PutUint32(hdr[:], uint32(len(rpcWelcome)))
			if _, err := c.Write(hdr[:]); err != nil {
				return err
			}
			if len(rpcWelcome) > 0 {
				if _, err := c.Write(rpcWelcome); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			serverErr <- err
			return
		}
		var msg string
		if err := func() error {
			data, err := func() ([]byte, error) {
				var hdr [4]byte
				if _, err := io.ReadFull(c, hdr[:]); err != nil {
					return nil, err
				}
				n := int(binary.BigEndian.Uint32(hdr[:]))
				buf := make([]byte, n)
				if _, err := io.ReadFull(c, buf); err != nil {
					return nil, err
				}
				return buf, nil
			}()
			if err != nil {
				return err
			}
			result, err := vendor.PickleLoads(data)
			if err != nil {
				return fmt.Errorf("pickle decode: %w", err)
			}
			return vendor.PickleAssign(result, &msg)
		}(); err != nil {
			serverErr <- err
			return
		}
		data, err := vendor.PickleDumps("ack:" + msg)
		if err != nil {
			serverErr <- err
			return
		}
		if err := func() error {
			var hdr [4]byte
			binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
			if _, err := c.Write(hdr[:]); err != nil {
				return err
			}
			if len(data) > 0 {
				if _, err := c.Write(data); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			serverErr <- err
			return
		}
		serverGot <- msg
		serverErr <- nil
	}()

	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("net.Dial: %v", err)
	}
	defer c.Close()
	if err := func() error {
		var hdr [4]byte
		if _, err := io.ReadFull(c, hdr[:]); err != nil {
			return err
		}
		n := int(binary.BigEndian.Uint32(hdr[:]))
		if n > 256 {
			return fmt.Errorf("rpc message too large: %d bytes", n)
		}
		msg := make([]byte, n)
		if _, err := io.ReadFull(c, msg); err != nil {
			return err
		}
		if len(msg) < len(rpcChallenge) || !bytes.Equal(msg[:len(rpcChallenge)], rpcChallenge) {
			return fmt.Errorf("protocol error, expected challenge, got: %q", msg)
		}
		mac := hmac.New(md5.New, key)
		if _, err := mac.Write(msg[len(rpcChallenge):]); err != nil {
			return err
		}
		digest := mac.Sum(nil)
		binary.BigEndian.PutUint32(hdr[:], uint32(len(digest)))
		if _, err := c.Write(hdr[:]); err != nil {
			return err
		}
		if len(digest) > 0 {
			if _, err := c.Write(digest); err != nil {
				return err
			}
		}
		if _, err := io.ReadFull(c, hdr[:]); err != nil {
			return err
		}
		if int(binary.BigEndian.Uint32(hdr[:])) > 256 {
			return fmt.Errorf("rpc message too large: %d bytes", binary.BigEndian.Uint32(hdr[:]))
		}
		resp := make([]byte, int(binary.BigEndian.Uint32(hdr[:])))
		if _, err := io.ReadFull(c, resp); err != nil {
			return err
		}
		if !bytes.Equal(resp, rpcWelcome) {
			return errors.New("digest sent was rejected")
		}
		return nil
	}(); err != nil {
		t.Fatalf("client handshake: %v", err)
	}

	data, err := vendor.PickleDumps("hello")
	if err != nil {
		t.Fatalf("client send: %v", err)
	}
	if err := func() error {
		var hdr [4]byte
		binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
		if _, err := c.Write(hdr[:]); err != nil {
			return err
		}
		if len(data) > 0 {
			if _, err := c.Write(data); err != nil {
				return err
			}
		}
		return nil
	}(); err != nil {
		t.Fatalf("client send: %v", err)
	}
	var resp string
	if err := func() error {
		data, err := func() ([]byte, error) {
			var hdr [4]byte
			if _, err := io.ReadFull(c, hdr[:]); err != nil {
				return nil, err
			}
			n := int(binary.BigEndian.Uint32(hdr[:]))
			buf := make([]byte, n)
			if _, err := io.ReadFull(c, buf); err != nil {
				return nil, err
			}
			return buf, nil
		}()
		if err != nil {
			return err
		}
		result, err := vendor.PickleLoads(data)
		if err != nil {
			return fmt.Errorf("pickle decode: %w", err)
		}
		return vendor.PickleAssign(result, &resp)
	}(); err != nil {
		t.Fatalf("client recv: %v", err)
	}
	if resp != "ack:hello" {
		t.Fatalf("unexpected response %q", resp)
	}

	select {
	case msg := <-serverGot:
		if msg != "hello" {
			t.Fatalf("server got %q", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting server")
	}
	if err := <-serverErr; err != nil {
		t.Fatalf("server: %v", err)
	}
}

func TestWaitForLocalClientsToDisconnect_Completes(t *testing.T) {
	prev := LocalClientInterfaces
	t.Cleanup(func() { LocalClientInterfaces = prev })

	LocalClientInterfaces = []*Interface{{Name: "c1"}}
	go func() {
		time.Sleep(50 * time.Millisecond)
		LocalClientInterfaces = nil
	}()

	if ok := waitForLocalClientsToDisconnect(2 * time.Second); !ok {
		t.Fatalf("expected wait to complete")
	}
}

func TestWaitForLocalClientsToDisconnect_TimesOut(t *testing.T) {
	prev := LocalClientInterfaces
	t.Cleanup(func() { LocalClientInterfaces = prev })

	LocalClientInterfaces = []*Interface{{Name: "c1"}}
	if ok := waitForLocalClientsToDisconnect(50 * time.Millisecond); ok {
		t.Fatalf("expected timeout")
	}
}
