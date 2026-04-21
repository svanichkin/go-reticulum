package rns

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sync"
	"time"

	"github.com/dsnet/compress/bzip2"
)

// ==== StreamDataMessage ======================================================

type StreamDataMessage struct {
	StreamID   *uint16
	Compressed bool
	Data       []byte
	EOF        bool
}

const (
	STREAM_ID_MAX = 0x3fff // 16383

	streamHeaderBytes   = 2
	channelEnvelopeSize = 6
	OVERHEAD            = streamHeaderBytes + channelEnvelopeSize
)

// Pack mirrors Python pack(self) -> bytes.
func (m *StreamDataMessage) Pack() ([]byte, error) {
	if m == nil || m.StreamID == nil {
		return nil, errors.New("stream_id")
	}
	// header_val = (0x3fff & stream_id) | (0x8000 if eof else 0) | (0x4000 if compressed else 0)
	headerVal := (0x3fff & int(*m.StreamID))
	if m.EOF {
		headerVal |= 0x8000
	}
	if m.Compressed {
		headerVal |= 0x4000
	}

	buf := make([]byte, 2+len(m.Data))
	binary.BigEndian.PutUint16(buf[:2], uint16(headerVal))
	copy(buf[2:], m.Data)
	return buf, nil
}

// Unpack mirrors Python unpack(self, raw).
func (m *StreamDataMessage) Unpack(raw []byte) error {
	if len(raw) < 2 {
		return errors.New("stream data too short")
	}

	header := binary.BigEndian.Uint16(raw[:2])
	m.StreamID = &header
	m.EOF = (0x8000 & *m.StreamID) > 0
	m.Compressed = (0x4000 & *m.StreamID) > 0
	*m.StreamID &= 0x3fff
	m.Data = raw[2:]

	if m.Compressed {
		reader, err := bzip2.NewReader(bytes.NewReader(m.Data), nil)
		if err != nil {
			return err
		}
		defer reader.Close()
		decompressed, err := io.ReadAll(reader)
		if err != nil {
			return err
		}
		m.Data = decompressed
	}
	return nil
}

// ==== RawChannelReader =======================================================

var errBufferValueError = errors.New("list.remove(x): x not in list")

// ErrWouldBlock matches Python RawIOBase.readinto() returning None when no data is available yet.
// It is used by RawChannelReader.ReadInto for non-blocking reads.
var ErrWouldBlock = errors.New("would block")

type RawChannelReader struct {
	streamID  int
	channel   *Channel
	lock      sync.Mutex
	buffer    []byte
	eof       bool
	listeners []func(int)
}

func (r *RawChannelReader) AddReadyCallback(cb func(int)) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.listeners = append(r.listeners, cb)
}

func (r *RawChannelReader) RemoveReadyCallback(cb func(int)) {
	r.lock.Lock()
	defer r.lock.Unlock()

	found := false
	for i, f := range r.listeners {
		if reflect.ValueOf(f).Pointer() == reflect.ValueOf(cb).Pointer() {
			r.listeners = append(r.listeners[:i], r.listeners[i+1:]...)
			found = true
			break
		}
	}

	// Python list.remove raises if the callback was not present.
	if !found {
		panic(errBufferValueError)
	}
}

// Mirrors Python _handle_message(self, message).
func (r *RawChannelReader) handleMessage(msg MessageBase) bool {
	sdm, ok := msg.(*StreamDataMessage)
	if !ok {
		return false
	}
	if sdm.StreamID == nil || int(*sdm.StreamID) != r.streamID {
		return false
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	if sdm.Data != nil {
		r.buffer = append(r.buffer, sdm.Data...)
	}
	if sdm.EOF {
		r.eof = true
	}

	// Python parity: ready callbacks receive the total buffered byte count
	// after appending the new StreamDataMessage data.
	readyBytes := len(r.buffer)
	for _, listener := range r.listeners {
		cb := listener
		go func(cb func(int)) {
			defer func() {
				if rec := recover(); rec != nil {
					Log(fmt.Sprintf("Error calling RawChannelReader(%d) callback: %v", r.streamID, rec), LOG_ERROR)
				}
			}()
			cb(readyBytes)
		}(cb)
	}

	return true
}

// Mirrors Python RawChannelReader._read(self, __size).
func (r *RawChannelReader) read(size int) ([]byte, bool) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if len(r.buffer) == 0 && !r.eof {
		return nil, false
	}
	if len(r.buffer) == 0 && r.eof {
		return nil, true
	}

	if size > len(r.buffer) {
		size = len(r.buffer)
	}
	if size < 0 {
		size = 0
	}
	res := make([]byte, size)
	copy(res, r.buffer[:size])
	r.buffer = r.buffer[size:]
	return res, true
}

// ReadInto is a non-blocking read that mirrors Python RawChannelReader.readinto().
// Go cannot return Python's None, so ErrWouldBlock distinguishes "no data yet"
// from EOF.
func (r *RawChannelReader) ReadInto(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	data, ok := r.read(len(p))
	if !ok {
		return 0, ErrWouldBlock
	}
	if len(data) == 0 {
		return 0, nil
	}

	n := copy(p, data)
	return n, nil
}

// Readable mirrors Python RawIOBase contract for RawChannelReader.
func (r *RawChannelReader) Readable() bool { return true }

// Writable mirrors Python RawIOBase contract for RawChannelReader.
func (r *RawChannelReader) Writable() bool { return false }

// Seekable mirrors Python RawIOBase contract for RawChannelReader.
func (r *RawChannelReader) Seekable() bool { return false }

func (r *RawChannelReader) Close() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	defer func() {
		if rec := recover(); rec != nil {
			Log(fmt.Sprintf("Error while closing RawChannelReader(%d): %v", r.streamID, rec), LOG_ERROR)
		}
	}()

	r.channel.RemoveMessageHandler(r.handleMessage)
	r.listeners = nil
	return nil
}

func (r *RawChannelReader) Enter() *RawChannelReader {
	return r
}

func (r *RawChannelReader) Exit(excType any, excValue any, excTraceback any) bool {
	_ = excType
	_ = excValue
	_ = excTraceback
	_ = r.Close()
	return false
}

// ==== RawChannelWriter =======================================================

type RawChannelWriter struct {
	mu         sync.Mutex
	streamID   int
	channel    *Channel
	eof        bool
	maxDataLen int
}

const (
	MAX_CHUNK_LEN     = 1024 * 16
	COMPRESSION_TRIES = 4
	// Python io.BufferedReader/Writer defaults to 8192 bytes.
	defaultBufferSize = 8192
)

var (
	errLinkNotReady = errors.New("channel not ready")
)

// Write mirrors Python write(self, __b).
func (w *RawChannelWriter) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(b) == 0 {
		if w.streamID < 0 || w.streamID > STREAM_ID_MAX {
			return 0, errors.New("stream_id must be 0-16383")
		}
		msg := &StreamDataMessage{
			StreamID:   func(v uint16) *uint16 { return &v }(uint16(w.streamID)),
			Compressed: false,
			Data:       nil,
			EOF:        w.eof,
		}
		if _, err := w.channel.Send(msg); err != nil {
			if cex, ok := err.(*ChannelException); ok && cex.Type == ME_LINK_NOT_READY {
				return 0, nil
			}
			return 0, err
		}
		return 0, nil
	}

	// Python parity: RawChannelWriter.write() sends at most one StreamDataMessage
	// per call and returns the number of input bytes consumed.
	maxDataLen := w.maxDataLen
	if maxDataLen <= 0 {
		if current := LinkMDU - OVERHEAD; current > 0 {
			maxDataLen = current
		} else {
			maxDataLen = MAX_CHUNK_LEN
		}
	}
	payloadLen := len(b)
	if payloadLen > MAX_CHUNK_LEN {
		payloadLen = MAX_CHUNK_LEN
	}
	payload := b[:payloadLen]

	chunkLen := len(payload)
	compSuccess := false
	var (
		compChunk          []byte
		processedLength    = chunkLen
		chunkSegmentLength int
	)
	for compTry := 1; chunkLen > 32 && compTry < COMPRESSION_TRIES; compTry++ {
		chunkSegmentLength = chunkLen / compTry
		if chunkSegmentLength <= 0 {
			continue
		}
		var compressedBuf bytes.Buffer
		writer, err := bzip2.NewWriter(&compressedBuf, nil)
		if err != nil {
			return 0, err
		}
		if _, err := writer.Write(payload[:chunkSegmentLength]); err != nil {
			_ = writer.Close()
			return 0, err
		}
		if err := writer.Close(); err != nil {
			return 0, err
		}
		compressed := compressedBuf.Bytes()
		if len(compressed) < maxDataLen && len(compressed) < chunkSegmentLength {
			compSuccess = true
			compChunk = compressed
			processedLength = chunkSegmentLength
			break
		}
	}

	var chunk []byte
	if compSuccess {
		chunk = compChunk
	} else {
		if processedLength > maxDataLen {
			processedLength = maxDataLen
		}
		chunk = make([]byte, processedLength)
		copy(chunk, payload[:processedLength])
	}

	if w.streamID < 0 || w.streamID > STREAM_ID_MAX {
		return 0, errors.New("stream_id must be 0-16383")
	}
	msg := &StreamDataMessage{
		StreamID:   func(v uint16) *uint16 { return &v }(uint16(w.streamID)),
		Compressed: compSuccess,
		Data:       chunk,
		EOF:        w.eof,
	}
	if _, err := w.channel.Send(msg); err != nil {
		if cex, ok := err.(*ChannelException); ok && cex.Type == ME_LINK_NOT_READY {
			return 0, nil
		}
		return 0, err
	}

	return processedLength, nil
}

func (w *RawChannelWriter) Close() error {
	timeout := 15 * time.Second
	if w != nil && w.channel != nil {
		w.channel.lock.RLock()
		txQueueLen := len(w.channel.txRing)
		outlet, ok := w.channel.outlet.(*LinkChannelOutlet)
		w.channel.lock.RUnlock()
		if ok && outlet != nil && outlet.link != nil {
			timeout = time.Duration(outlet.link.RTT.Seconds() * float64(txQueueLen) * float64(time.Second))
		}
	}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) && !w.channel.IsReadyToSend() {
		time.Sleep(50 * time.Millisecond)
	}

	w.mu.Lock()
	w.eof = true
	w.mu.Unlock()

	_, err := w.Write([]byte{})
	return err
}

func (w *RawChannelWriter) Enter() *RawChannelWriter {
	return w
}

func (w *RawChannelWriter) Exit(excType any, excValue any, excTraceback any) bool {
	_ = excType
	_ = excValue
	_ = excTraceback
	_ = w.Close()
	return false
}

// Seekable mirrors Python RawIOBase contract for RawChannelWriter.
func (w *RawChannelWriter) Seekable() bool { return false }

// Readable mirrors Python RawIOBase contract for RawChannelWriter.
func (w *RawChannelWriter) Readable() bool { return false }

// Writable mirrors Python RawIOBase contract for RawChannelWriter.
func (w *RawChannelWriter) Writable() bool { return true }

func CreateReader(streamID int, ch *Channel, readyCallback func(int)) *BufferedReader {
	reader := &RawChannelReader{
		streamID: streamID,
		channel:  ch,
		buffer:   make([]byte, 0),
		eof:      false,
	}
	if ch != nil {
		if err := channelRegisterMessageTypeLocked(ch, &StreamDataMessage{}, true, func() *uint16 { v := uint16(SMT_STREAM_DATA); return &v }()); err != nil {
			Log(fmt.Sprintf("RawChannelReader failed to register StreamDataMessage: %v", err), LOG_ERROR)
		}
	}
	if ch != nil {
		ch.AddMessageHandler(reader.handleMessage)
	}
	if readyCallback != nil {
		reader.AddReadyCallback(readyCallback)
	}
	return &BufferedReader{raw: reader}
}

func CreateWriter(streamID int, ch *Channel) *BufferedWriter {
	maxLen := LinkMDU - OVERHEAD
	if maxLen <= 0 {
		maxLen = MAX_CHUNK_LEN
	}
	if ch != nil {
		if err := channelRegisterMessageTypeLocked(ch, &StreamDataMessage{}, true, func() *uint16 { v := uint16(SMT_STREAM_DATA); return &v }()); err != nil {
			Log(fmt.Sprintf("RawChannelWriter failed to register StreamDataMessage: %v", err), LOG_ERROR)
		}
	}
	writer := &RawChannelWriter{
		streamID:   streamID,
		channel:    ch,
		eof:        false,
		maxDataLen: maxLen,
	}
	return &BufferedWriter{raw: writer}
}

func CreateBidirectionalBuffer(
	receiveStreamID int,
	sendStreamID int,
	ch *Channel,
	readyCallback func(int),
) *BufferedRWPair {
	reader := CreateReader(receiveStreamID, ch, readyCallback)
	writer := CreateWriter(sendStreamID, ch)
	return &BufferedRWPair{
		Reader: reader,
		Writer: writer,
	}
}
