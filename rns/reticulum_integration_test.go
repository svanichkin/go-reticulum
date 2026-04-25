package rns

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
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

func TestIntegration_Reticulum_SharedInstanceRPC_Getters(t *testing.T) {
	requireIntegration(t)

	dir, err := os.MkdirTemp("", "rns_reticulum_it_*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	rpcKey := make([]byte, 16)
	for i := range rpcKey {
		rpcKey[i] = byte(i + 1)
	}
	cfg := []byte(
		"[reticulum]\n" +
			"share_instance = yes\n" +
			"shared_instance_type = tcp\n" +
			"shared_instance_port = 0\n" +
			"instance_control_port = 0\n" +
			"rpc_key = " + hex.EncodeToString(rpcKey) + "\n",
	)
	if err := os.WriteFile(filepath.Join(dir, "config"), cfg, 0o644); err != nil {
		t.Fatalf("WriteFile(config): %v", err)
	}

	prevInstance := instance
	instance = nil
	t.Cleanup(func() { instance = prevInstance })

	r, err := NewReticulum(&dir, nil, nil, nil, false, func() *string { s := "tcp"; return &s }())
	if err != nil {
		t.Fatalf("NewReticulum: %v", err)
	}
	if r == nil {
		t.Fatalf("NewReticulum returned nil")
	}
	t.Cleanup(func() {
		if r.rpcLn != nil {
			_ = r.rpcLn.Close()
		}
	})

	if !r.IsSharedInstance {
		t.Skipf("shared instance RPC not available in this environment (shared=%v standalone=%v connected=%v)", r.IsSharedInstance, r.IsStandaloneInstance, r.IsConnectedToSharedInstance)
	}
	if r.rpcLn == nil || r.rpcLn.Addr() == nil {
		t.Fatalf("expected rpc listener addr")
	}

	time.Sleep(10 * time.Millisecond)

	doRoundtrip := func(req any, resp any) error {
		c, err := net.Dial("tcp", r.rpcLn.Addr().String())
		if err != nil {
			if strings.Contains(err.Error(), "operation not permitted") {
				t.Skipf("loopback tcp dial not permitted in this environment: %v", err)
			}
			return err
		}
		defer c.Close()
		handshake := func(conn net.Conn, key []byte) error {
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
		if err := handshake(c, r.RPCKey); err != nil {
			return err
		}

		data, err := vendor.PickleDumps(req)
		if err != nil {
			return err
		}
		if err := func() error {
			var hdr [4]byte
			binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
			if _, err := c.Write(hdr[:]); err != nil {
				return err
			}
			if len(data) > 0 {
				_, err := c.Write(data)
				return err
			}
			return nil
		}(); err != nil {
			return err
		}
		data, err = func() ([]byte, error) {
			var hdr [4]byte
			if _, err := io.ReadFull(c, hdr[:]); err != nil {
				return nil, err
			}
			n := int(binary.BigEndian.Uint32(hdr[:]))
			if n > 0 {
				buf := make([]byte, n)
				if _, err := io.ReadFull(c, buf); err != nil {
					return nil, err
				}
				return buf, nil
			}
			return nil, nil
		}()
		if err != nil {
			return err
		}
		result, err := vendor.PickleLoads(data)
		if err != nil {
			return fmt.Errorf("pickle decode: %w", err)
		}
		return vendor.PickleAssign(result, resp)
	}

	{
		var count int
		if err := doRoundtrip(map[string]any{"get": "link_count"}, &count); err != nil {
			t.Fatalf("rpc link_count: %v", err)
		}
		if count < 0 {
			t.Fatalf("unexpected link_count=%d", count)
		}
	}

	{
		var stats map[string]any
		if err := doRoundtrip(map[string]any{"get": "interface_stats"}, &stats); err != nil {
			t.Fatalf("rpc interface_stats: %v", err)
		}
		if stats == nil {
			t.Fatalf("expected interface_stats map")
		}
	}
}
