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
	StreamID   uint16
	Compressed bool
	Data       []byte
	EOF        bool
}

func (m *StreamDataMessage) MsgType() uint16 { return uint16(SMT_STREAM_DATA) }

const (
	STREAM_ID_MAX = 0x3fff // 16383

	streamHeaderBytes   = 2
	channelEnvelopeSize = 6
	OVERHEAD            = streamHeaderBytes + channelEnvelopeSize
)

func streamMaxDataLen(_ *Channel) int {
	// Python reference uses Link.MDU (global) to compute StreamDataMessage.MAX_DATA_LEN.
	if LinkMDU <= OVERHEAD {
		return 0
	}
	return LinkMDU - OVERHEAD
}

func NewStreamDataMessage(streamID int, data []byte, eof bool, compressed bool) (*StreamDataMessage, error) {
	if streamID < 0 || streamID > STREAM_ID_MAX {
		return nil, errors.New("stream_id must be 0-16383")
	}

	m := &StreamDataMessage{
		StreamID:   uint16(streamID),
		Compressed: compressed,
		Data:       data,
		EOF:        eof,
	}
	return m, nil
}

// Pack mirrors Python pack(self) -> bytes.
func (m *StreamDataMessage) Pack() ([]byte, error) {
	// header_val = (0x3fff & stream_id) | (0x8000 if eof else 0) | (0x4000 if compressed else 0)
	headerVal := (0x3fff & int(m.StreamID))
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
	m.EOF = (0x8000 & header) > 0
	m.Compressed = (0x4000 & header) > 0
	m.StreamID = header & 0x3fff
	m.Data = raw[2:]

	if m.Compressed {
		decompressed, err := bz2Decompress(m.Data)
		if err != nil {
			return err
		}
		m.Data = decompressed
	}
	return nil
}

// ==== RawChannelReader =======================================================

type ReadyCallback func(readyBytes int)

// ErrWouldBlock matches Python RawIOBase.readinto() returning None when no data is available yet.
// It is used by RawChannelReader.ReadInto for non-blocking reads.
var ErrWouldBlock = errors.New("would block")

type RawChannelReader struct {
	streamID  int
	channel   *Channel
	lock      sync.Mutex
	cond      *sync.Cond
	buffer    []byte
	eof       bool
	closed    bool
	listeners []ReadyCallback
}

func NewRawChannelReader(streamID int, ch *Channel) *RawChannelReader {
	r := &RawChannelReader{
		streamID: streamID,
		channel:  ch,
		buffer:   make([]byte, 0),
		eof:      false,
	}
	r.cond = sync.NewCond(&r.lock)

	// ch._register_message_type(StreamDataMessage, is_system_type=True)
	if err := ch._register_message_type(&StreamDataMessage{}, true); err != nil {
		Log(fmt.Sprintf("RawChannelReader failed to register StreamDataMessage: %v", err), LOG_ERROR)
	}

	// ch.add_message_handler(self._handle_message)
	ch.AddMessageHandler(r.handleMessage)

	return r
}

func (r *RawChannelReader) AddReadyCallback(cb ReadyCallback) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.listeners = append(r.listeners, cb)
}

func (r *RawChannelReader) RemoveReadyCallback(cb ReadyCallback) {
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
		panic("ready callback not registered")
	}
}

// Mirrors Python _handle_message(self, message).
func (r *RawChannelReader) handleMessage(msg MessageBase) bool {
	sdm, ok := msg.(*StreamDataMessage)
	if !ok {
		return false
	}
	if int(sdm.StreamID) != r.streamID {
		return false
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	if r.closed {
		return true
	}

	if sdm.Data != nil {
		r.buffer = append(r.buffer, sdm.Data...)
		r.cond.Broadcast()
	}
	if sdm.EOF {
		r.eof = true
		r.cond.Broadcast()
	}

	// Python parity: ready callbacks receive the total buffered byte count
	// after appending the new StreamDataMessage data.
	readyBytes := len(r.buffer)
	for _, listener := range r.listeners {
		cb := listener
		go func(cb ReadyCallback) {
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

// Mirrors Python _read(self, __size).
func (r *RawChannelReader) readN(size int, block bool) ([]byte, bool, bool) {
	r.lock.Lock()
	defer r.lock.Unlock()

	for len(r.buffer) == 0 && !r.eof && !r.closed && block {
		r.cond.Wait()
	}

	if len(r.buffer) == 0 && (r.eof || r.closed) {
		return nil, true, true
	}

	if len(r.buffer) == 0 {
		return nil, false, false
	}

	if size > len(r.buffer) {
		size = len(r.buffer)
	}

	res := make([]byte, size)
	copy(res, r.buffer[:size])
	r.buffer = r.buffer[size:]
	return res, true, false
}

// Read implements io.Reader (mirrors Python readinto).
func (r *RawChannelReader) Read(p []byte) (int, error) {
	data, ok, eof := r.readN(len(p), true)
	if data == nil && eof {
		return 0, io.EOF
	}
	if !ok || data == nil {
		return 0, nil
	}
	copy(p, data)
	return len(data), nil
}

// ReadInto is a non-blocking read that mirrors Python RawChannelReader.readinto().
// If no data is available and EOF has not been received, it returns ErrWouldBlock.
func (r *RawChannelReader) ReadInto(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	return r.tryRead(p)
}

func (r *RawChannelReader) tryRead(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	if len(r.buffer) == 0 && !r.eof && !r.closed {
		return 0, ErrWouldBlock
	}
	if len(r.buffer) == 0 && (r.eof || r.closed) {
		return 0, io.EOF
	}

	n := len(p)
	if n > len(r.buffer) {
		n = len(r.buffer)
	}
	copy(p[:n], r.buffer[:n])
	r.buffer = r.buffer[n:]
	return n, nil
}

// TryRead mirrors Python RawChannelReader._read/readinto by returning ErrWouldBlock if no data.
func (r *RawChannelReader) TryRead(p []byte) (int, error) {
	data, ok, eof := r.readN(len(p), false)
	if data == nil && eof {
		return 0, io.EOF
	}
	if data == nil && !ok {
		return 0, ErrWouldBlock
	}
	copy(p, data)
	return len(data), nil
}
func (r *RawChannelReader) Close() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	defer func() {
		if rec := recover(); rec != nil {
			Log(fmt.Sprintf("Error while closing RawChannelReader(%d): %v", r.streamID, rec), LOG_ERROR)
		}
	}()

	r.closed = true
	r.channel.RemoveMessageHandler(r.handleMessage)
	r.listeners = nil
	if r.cond != nil {
		r.cond.Broadcast()
	}
	return nil
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
	errLinkNotReady    = errors.New("channel not ready")
	errChannelUnusable = errors.New("channel outlet unusable")
)

func NewRawChannelWriter(streamID int, ch *Channel) *RawChannelWriter {
	maxLen := streamMaxDataLen(ch)
	if maxLen <= 0 {
		maxLen = MAX_CHUNK_LEN
	}
	return &RawChannelWriter{
		streamID:   streamID,
		channel:    ch,
		eof:        false,
		maxDataLen: maxLen,
	}
}

// Write mirrors Python write(self, __b).
func (w *RawChannelWriter) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(b) == 0 {
		if w.eof {
			return 0, w.sendEmptyChunk()
		}
		return 0, nil
	}

	// Python parity: RawChannelWriter.write() is non-blocking and returns 0
	// when the channel/link is not ready.
	return w.writeInternal(b, false)
}

func (w *RawChannelWriter) WriteNonBlocking(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.writeInternal(b, false)
}

// WriteBlocking is used by buffered writers to block until progress can be made.
func (w *RawChannelWriter) WriteBlocking(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.writeInternal(b, true)
}

func (w *RawChannelWriter) writeInternal(b []byte, block bool) (int, error) {
	if len(b) == 0 {
		if w.eof {
			return 0, w.sendEmptyChunk()
		}
		return 0, nil
	}

	// Python parity: RawChannelWriter.write() sends at most one StreamDataMessage
	// per call and returns the number of input bytes consumed. Buffered writers
	// are responsible for calling write() repeatedly until all data is sent.
	if current := streamMaxDataLen(w.channel); current > 0 {
		w.maxDataLen = current
	}
	payloadLen := len(b)
	if payloadLen > MAX_CHUNK_LEN {
		payloadLen = MAX_CHUNK_LEN
	}

	// Attempt compression on up to payloadLen bytes, but never accept a compressed
	// chunk that cannot fit in a single StreamDataMessage frame.
	payload := b[:payloadLen]

	for {
		sent, processed, err := w.processChunk(payload)
		if errors.Is(err, errLinkNotReady) && block {
			if waitErr := w.waitUntilReady(); waitErr != nil {
				return 0, waitErr
			}
			continue
		}
		if errors.Is(err, errLinkNotReady) && !block {
			return 0, nil
		}
		if err != nil {
			return 0, err
		}
		_ = sent
		return processed, nil
	}
}

func (w *RawChannelWriter) processChunk(data []byte) (int, int, error) {
	chunkLen := len(data)
	compSuccess := false
	var (
		compChunk          []byte
		processedLength    = chunkLen
		chunkSegmentLength int
	)

	// Python reference tries 1..(COMPRESSION_TRIES-1).
	for compTry := 1; chunkLen > 32 && compTry < COMPRESSION_TRIES; compTry++ {
		chunkSegmentLength = chunkLen / compTry
		if chunkSegmentLength <= 0 {
			continue
		}
		compressed, err := bz2Compress(data[:chunkSegmentLength])
		if err != nil {
			return 0, 0, err
		}
		if len(compressed) < w.maxDataLen && len(compressed) < chunkSegmentLength {
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
		// Python parity: when not compressing, cap to StreamDataMessage.MAX_DATA_LEN.
		if processedLength > w.maxDataLen {
			processedLength = w.maxDataLen
		}
		chunk = make([]byte, processedLength)
		copy(chunk, data[:processedLength])
	}

	msg, err := NewStreamDataMessage(w.streamID, chunk, false, compSuccess)
	if err != nil {
		return 0, 0, err
	}
	if _, err := w.channel.TrySend(msg); err != nil {
		if cex, ok := err.(*ChannelException); ok && cex.Type == ME_LINK_NOT_READY {
			return 0, 0, errLinkNotReady
		}
		return 0, 0, err
	}

	return len(chunk), processedLength, nil
}

func (w *RawChannelWriter) sendEmptyChunk() error {
	msg, err := NewStreamDataMessage(w.streamID, nil, w.eof, false)
	if err != nil {
		return err
	}
	for {
		if _, err := w.channel.TrySend(msg); err != nil {
			if cex, ok := err.(*ChannelException); ok && cex.Type == ME_LINK_NOT_READY {
				if waitErr := w.waitUntilReady(); waitErr != nil {
					return waitErr
				}
				continue
			}
			return err
		}
		return nil
	}
}

func (w *RawChannelWriter) Close() error {
	var timeout time.Time

	func() {
		defer func() {
			if rec := recover(); rec != nil {
				timeout = time.Now().Add(15 * time.Second)
			}
		}()

		linkRTT := w.channel.Outlet().Rtt()
		if linkRTT <= 0 {
			linkRTT = 0.1
		}
		txLen := w.channel.TxQueueLen()
		if txLen <= 0 {
			txLen = 1
		}
		timeout = time.Now().Add(time.Duration(linkRTT*float64(txLen)) * time.Second)
	}()

	for time.Now().Before(timeout) && !w.channel.IsReadyToSend() {
		time.Sleep(50 * time.Millisecond)
	}

	w.mu.Lock()
	w.eof = true
	w.mu.Unlock()

	_, err := w.Write([]byte{})
	return err
}

func (w *RawChannelWriter) waitUntilReady() error {
	for {
		if w.channel.IsReadyToSend() {
			return nil
		}
		outlet := w.channel.Outlet()
		if outlet == nil || !outlet.IsUsable() {
			return errChannelUnusable
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// ---- bzip2 helpers ---------------------------------------------------------

func bz2Decompress(data []byte) ([]byte, error) {
	reader, err := bzip2.NewReader(bytes.NewReader(data), nil)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func bz2Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := bzip2.NewWriter(&buf, nil)
	if err != nil {
		return nil, err
	}
	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ==== Buffer API =============================================================

type ChannelBufferedReader struct {
	raw *RawChannelReader
	buf []byte
}

func (r *ChannelBufferedReader) Close() error {
	if r == nil || r.raw == nil {
		return nil
	}
	err := r.raw.Close()
	r.raw = nil
	return err
}

func (r *ChannelBufferedReader) Raw() *RawChannelReader {
	return r.raw
}

func (r *ChannelBufferedReader) Read(p []byte) (int, error) {
	if r == nil || r.raw == nil {
		return 0, io.EOF
	}
	if len(p) == 0 {
		return 0, nil
	}

	// Ensure we can satisfy up to len(p) from the internal buffer by pulling
	// non-blocking chunks from the raw reader. This is important for Python parity:
	// ready callbacks provide the message payload length, and buffer.read(ready_bytes)
	// should return that entire segment if available.
	for len(r.buf) < len(p) {
		need := len(p) - len(r.buf)
		chunk := defaultBufferSize
		if need < chunk {
			chunk = need
		}
		tmp := make([]byte, chunk)
		n, err := r.raw.TryRead(tmp)
		if err != nil {
			if len(r.buf) > 0 {
				break
			}
			return 0, err
		}
		if n == 0 {
			break
		}
		r.buf = append(r.buf, tmp[:n]...)
		if n < chunk {
			break
		}
	}

	if len(r.buf) == 0 {
		return 0, ErrWouldBlock
	}

	n := copy(p, r.buf)
	r.buf = r.buf[n:]
	return n, nil
}

type ChannelBufferedWriter struct {
	raw *RawChannelWriter
	buf []byte
}

func (w *ChannelBufferedWriter) Close() error {
	if w == nil {
		return nil
	}

	var firstErr error
	if err := w.Flush(); err != nil && firstErr == nil {
		firstErr = err
	}
	if w.raw != nil {
		if err := w.raw.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		w.raw = nil
	}
	return firstErr
}

func (w *ChannelBufferedWriter) Raw() *RawChannelWriter {
	return w.raw
}

func (w *ChannelBufferedWriter) Write(p []byte) (int, error) {
	if w == nil || w.raw == nil {
		return 0, io.ErrClosedPipe
	}
	if len(p) == 0 {
		return 0, nil
	}

	// Python parity: io.BufferedWriter writes large chunks directly to the raw
	// stream (bypassing the internal buffer), which keeps StreamDataMessage
	// segmentation aligned to MAX_DATA_LEN boundaries instead of buffer-size
	// boundaries.
	if len(p) >= defaultBufferSize {
		if len(w.buf) > 0 {
			if err := w.Flush(); err != nil {
				return 0, err
			}
		}
		written := 0
		remaining := p
		for len(remaining) > 0 {
			n, err := w.raw.WriteBlocking(remaining)
			written += n
			if err != nil {
				return written, err
			}
			if n == 0 {
				return written, errChannelUnusable
			}
			remaining = remaining[n:]
		}
		return written, nil
	}

	written := 0
	for len(p) > 0 {
		space := defaultBufferSize - len(w.buf)
		if space == 0 {
			if err := w.Flush(); err != nil {
				return written, err
			}
			space = defaultBufferSize - len(w.buf)
			if space == 0 {
				return written, ErrWouldBlock
			}
		}

		chunk := len(p)
		if chunk > space {
			chunk = space
		}
		w.buf = append(w.buf, p[:chunk]...)
		written += chunk
		p = p[chunk:]

		if len(w.buf) == defaultBufferSize {
			if err := w.Flush(); err != nil {
				return written, err
			}
		}
	}

	return written, nil
}

func (w *ChannelBufferedWriter) Flush() error {
	if w == nil || w.raw == nil {
		return io.ErrClosedPipe
	}
	for len(w.buf) > 0 {
		// Python parity: BufferedWriter.flush() blocks until all buffered bytes
		// are written (or an error occurs).
		n, err := w.raw.WriteBlocking(w.buf)
		if err != nil {
			return err
		}
		if n == 0 {
			// RawChannelWriter.Write() should not return 0 for non-empty buffers
			// unless the channel becomes unusable.
			return errChannelUnusable
		}
		w.buf = w.buf[n:]
	}
	return nil
}

type ChannelBufferedReadWriter struct {
	reader *ChannelBufferedReader
	writer *ChannelBufferedWriter
}

func (rw *ChannelBufferedReadWriter) Close() error {
	if rw == nil {
		return nil
	}
	var firstErr error
	if rw.writer != nil {
		if err := rw.writer.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if rw.reader != nil {
		if err := rw.reader.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (rw *ChannelBufferedReadWriter) Read(p []byte) (int, error) {
	if rw == nil || rw.reader == nil {
		return 0, io.EOF
	}
	return rw.reader.Read(p)
}

func (rw *ChannelBufferedReadWriter) Write(p []byte) (int, error) {
	if rw == nil || rw.writer == nil {
		return 0, io.ErrClosedPipe
	}
	return rw.writer.Write(p)
}

func (rw *ChannelBufferedReadWriter) Flush() error {
	if rw == nil || rw.writer == nil {
		return io.ErrClosedPipe
	}
	return rw.writer.Flush()
}

func (rw *ChannelBufferedReadWriter) RawReader() *RawChannelReader {
	if rw == nil {
		return nil
	}
	return rw.reader.Raw()
}

func (rw *ChannelBufferedReadWriter) RawWriter() *RawChannelWriter {
	if rw == nil {
		return nil
	}
	return rw.writer.Raw()
}

// CreateReader mirrors Buffer.create_reader(...).
func CreateReader(streamID int, ch *Channel, readyCallback ReadyCallback) *ChannelBufferedReader {
	reader := NewRawChannelReader(streamID, ch)
	if readyCallback != nil {
		reader.AddReadyCallback(readyCallback)
	}
	return &ChannelBufferedReader{
		raw: reader,
	}
}

// CreateWriter mirrors Buffer.create_writer(...).
func CreateWriter(streamID int, ch *Channel) *ChannelBufferedWriter {
	writer := NewRawChannelWriter(streamID, ch)
	return &ChannelBufferedWriter{
		raw: writer,
	}
}

// CreateBidirectionalBuffer mirrors create_bidirectional_buffer(...).
func CreateBidirectionalBuffer(
	receiveStreamID int,
	sendStreamID int,
	ch *Channel,
	readyCallback ReadyCallback,
) *ChannelBufferedReadWriter {
	reader := CreateReader(receiveStreamID, ch, readyCallback)
	writer := CreateWriter(sendStreamID, ch)
	return &ChannelBufferedReadWriter{
		reader: reader,
		writer: writer,
	}
}

// Buffer mirrors the Python helper namespace so callers can use rns.Buffer.CreateReader(...).
var Buffer bufferAPI

type bufferAPI struct{}

func (bufferAPI) CreateReader(streamID int, ch *Channel, ready ReadyCallback) *ChannelBufferedReader {
	return CreateReader(streamID, ch, ready)
}

func (bufferAPI) CreateWriter(streamID int, ch *Channel) *ChannelBufferedWriter {
	return CreateWriter(streamID, ch)
}

func (bufferAPI) CreateBidirectionalBuffer(
	receiveStreamID int,
	sendStreamID int,
	ch *Channel,
	ready ReadyCallback,
) *ChannelBufferedReadWriter {
	return CreateBidirectionalBuffer(receiveStreamID, sendStreamID, ch, ready)
}
