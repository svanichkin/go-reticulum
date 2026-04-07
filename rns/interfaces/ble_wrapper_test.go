//go:build !ios

package interfaces

import (
	"context"
	"io"
	"testing"
)

type stubBLETransport struct {
	openCalled  bool
	closeCalled bool
	readCalled  bool
	writeCalled bool
	openErr     error
	closeErr    error
	readN       int
	readErr     error
	writeN      int
	writeErr    error
	openState   bool
	disappeared bool
	name        string
}

func (s *stubBLETransport) Open(context.Context) error { s.openCalled = true; return s.openErr }
func (s *stubBLETransport) Close() error               { s.closeCalled = true; return s.closeErr }
func (s *stubBLETransport) Read(p []byte) (int, error) {
	s.readCalled = true
	if s.readN > 0 && len(p) >= s.readN {
		for i := 0; i < s.readN; i++ {
			p[i] = byte(i + 1)
		}
	}
	return s.readN, s.readErr
}
func (s *stubBLETransport) Write([]byte) (int, error) { s.writeCalled = true; return s.writeN, s.writeErr }
func (s *stubBLETransport) IsOpen() bool              { return s.openState }
func (s *stubBLETransport) DeviceDisappeared() bool   { return s.disappeared }
func (s *stubBLETransport) String() string            { return s.name }

func TestBLETransportAdapter_ProxiesBLETransportSurface(t *testing.T) {
	inner := &stubBLETransport{
		readN:       3,
		readErr:     io.EOF,
		writeN:      5,
		writeErr:    io.ErrShortWrite,
		openState:   true,
		disappeared: true,
		name:        "ble://demo",
	}
	adapter := &bleTransportAdapter{inner: inner}

	if err := adapter.Open(context.Background()); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if !inner.openCalled {
		t.Fatal("Open() was not proxied")
	}

	buf := make([]byte, 8)
	n, err := adapter.Read(buf)
	if n != 3 || err != io.EOF {
		t.Fatalf("Read() = (%d, %v), want (3, %v)", n, err, io.EOF)
	}
	if !inner.readCalled {
		t.Fatal("Read() was not proxied")
	}

	n, err = adapter.Write([]byte("hello"))
	if n != 5 || err != io.ErrShortWrite {
		t.Fatalf("Write() = (%d, %v), want (5, %v)", n, err, io.ErrShortWrite)
	}
	if !inner.writeCalled {
		t.Fatal("Write() was not proxied")
	}

	if !adapter.IsOpen() {
		t.Fatal("IsOpen() = false, want true")
	}
	if !adapter.DeviceDisappeared() {
		t.Fatal("DeviceDisappeared() = false, want true")
	}
	if got := adapter.String(); got != "ble://demo" {
		t.Fatalf("String() = %q, want %q", got, "ble://demo")
	}

	if err := adapter.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if !inner.closeCalled {
		t.Fatal("Close() was not proxied")
	}
}
