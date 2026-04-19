package rns

import (
	"errors"
	"io"
)

type BufferedReader struct {
	raw *RawChannelReader
	buf []byte
}

func (r *BufferedReader) Close() error {
	if r == nil || r.raw == nil {
		return nil
	}
	err := r.raw.Close()
	r.raw = nil
	r.buf = nil
	return err
}

func (r *BufferedReader) Enter() *BufferedReader {
	return r
}

func (r *BufferedReader) Exit(excType any, excValue any, excTraceback any) bool {
	_ = excType
	_ = excValue
	_ = excTraceback
	_ = r.Close()
	return false
}

func (r *BufferedReader) ReadInto(p []byte) (int, error) {
	if r == nil || r.raw == nil {
		return 0, nil
	}
	return r.raw.ReadInto(p)
}

func (r *BufferedReader) Read(p []byte) (int, error) {
	if r == nil || r.raw == nil {
		return 0, nil
	}
	if len(p) == 0 {
		return 0, nil
	}

	for len(r.buf) < len(p) {
		need := len(p) - len(r.buf)
		chunk := defaultBufferSize
		if need < chunk {
			chunk = need
		}
		data, ok := r.raw.read(chunk)
		if !ok {
			if len(r.buf) > 0 {
				break
			}
			return 0, nil
		}
		if len(data) == 0 {
			break
		}
		r.buf = append(r.buf, data...)
		if len(data) < chunk {
			break
		}
	}

	if len(r.buf) == 0 {
		return 0, nil
	}

	n := copy(p, r.buf)
	r.buf = r.buf[n:]
	return n, nil
}

type BufferedWriter struct {
	raw *RawChannelWriter
	buf []byte
}

var errBufferWouldBlock = errors.New("write could not complete without blocking")

func (w *BufferedWriter) Close() error {
	if w == nil {
		return nil
	}

	var firstErr error
	if w.raw != nil {
		flush := func() error {
			for len(w.buf) > 0 {
				n, err := w.raw.Write(w.buf)
				if err != nil {
					return err
				}
				if n == 0 {
					return errBufferWouldBlock
				}
				w.buf = w.buf[n:]
			}
			return nil
		}
		if err := flush(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if w.raw != nil {
		if err := w.raw.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		w.raw = nil
	}
	return firstErr
}

func (w *BufferedWriter) Write(p []byte) (int, error) {
	if w == nil || w.raw == nil {
		return 0, io.ErrClosedPipe
	}
	if len(p) == 0 {
		return 0, nil
	}

	if len(p) >= defaultBufferSize {
		flush := func() error {
			for len(w.buf) > 0 {
				n, err := w.raw.Write(w.buf)
				if err != nil {
					return err
				}
				if n == 0 {
					return errBufferWouldBlock
				}
				w.buf = w.buf[n:]
			}
			return nil
		}
		if len(w.buf) > 0 {
			if err := flush(); err != nil {
				return 0, err
			}
		}
		written := 0
		remaining := p
		for len(remaining) > 0 {
			n, err := w.raw.Write(remaining)
			written += n
			if err != nil {
				return written, err
			}
			if n == 0 {
				return written, errBufferWouldBlock
			}
			remaining = remaining[n:]
		}
		return written, nil
	}

	written := 0
	for len(p) > 0 {
		space := defaultBufferSize - len(w.buf)
		if space == 0 {
			flush := func() error {
				for len(w.buf) > 0 {
					n, err := w.raw.Write(w.buf)
					if err != nil {
						return err
					}
					if n == 0 {
						return errBufferWouldBlock
					}
					w.buf = w.buf[n:]
				}
				return nil
			}
			if err := flush(); err != nil {
				return written, err
			}
			space = defaultBufferSize - len(w.buf)
			if space == 0 {
				return written, errBufferWouldBlock
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
			for len(w.buf) > 0 {
				n, err := w.raw.Write(w.buf)
				if err != nil {
					return written, err
				}
				if n == 0 {
					return written, errBufferWouldBlock
				}
				w.buf = w.buf[n:]
			}
		}
	}

	return written, nil
}

func (w *BufferedWriter) Flush() error {
	if w == nil || w.raw == nil {
		return io.ErrClosedPipe
	}
	for len(w.buf) > 0 {
		n, err := w.raw.Write(w.buf)
		if err != nil {
			return err
		}
		if n == 0 {
			return errBufferWouldBlock
		}
		w.buf = w.buf[n:]
	}
	return nil
}

func (w *BufferedWriter) Enter() *BufferedWriter {
	return w
}

func (w *BufferedWriter) Exit(excType any, excValue any, excTraceback any) bool {
	_ = excType
	_ = excValue
	_ = excTraceback
	_ = w.Close()
	return false
}

type BufferedRWPair struct {
	Reader *BufferedReader
	Writer *BufferedWriter
}

func (rw *BufferedRWPair) Close() error {
	if rw == nil {
		return nil
	}
	var firstErr error
	if rw.Writer != nil {
		if err := rw.Writer.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if rw.Reader != nil {
		if err := rw.Reader.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (rw *BufferedRWPair) Enter() *BufferedRWPair {
	return rw
}

func (rw *BufferedRWPair) Exit(excType any, excValue any, excTraceback any) bool {
	_ = excType
	_ = excValue
	_ = excTraceback
	_ = rw.Close()
	return false
}

func (rw *BufferedRWPair) Read(p []byte) (int, error) {
	if rw == nil || rw.Reader == nil {
		return 0, nil
	}
	return rw.Reader.Read(p)
}

func (rw *BufferedRWPair) Write(p []byte) (int, error) {
	if rw == nil || rw.Writer == nil {
		return 0, io.ErrClosedPipe
	}
	return rw.Writer.Write(p)
}

func (rw *BufferedRWPair) Flush() error {
	if rw == nil || rw.Writer == nil {
		return io.ErrClosedPipe
	}
	return rw.Writer.Flush()
}
