package mcptrace

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"
)

// RoundTripper records JSON messages sent and received by an HTTP client.
// It preserves request and response streaming and recognizes JSON bodies and
// server-sent-event data lines.
type RoundTripper struct {
	Transport http.RoundTripper
	Trace     *Writer
}

// RoundTrip implements http.RoundTripper.
func (t *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	transport := t.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	if t.Trace == nil {
		return nil, fmt.Errorf("nil trace writer")
	}
	copyReq := req.Clone(req.Context())
	if req.Body != nil {
		copyReq.Body = newCaptureBody(req.Body, t.Trace, "send")
	}
	resp, err := transport.RoundTrip(copyReq)
	if err != nil {
		return resp, err
	}
	if resp.Body != nil {
		resp.Body = newCaptureBody(resp.Body, t.Trace, "recv")
	}
	return resp, nil
}

// Handler records JSON messages received and sent by an HTTP server handler.
func Handler(next http.Handler, trace *Writer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if trace == nil {
			http.Error(w, "trace writer unavailable", http.StatusInternalServerError)
			return
		}
		if req.Body != nil {
			req.Body = newCaptureBody(req.Body, trace, "recv")
		}
		capture := &captureResponseWriter{ResponseWriter: w, trace: trace}
		next.ServeHTTP(capture, req)
		if err := capture.flush(); err != nil {
			// The handler has already returned; HTTP has no way to report this error.
			// The underlying response has still been delivered to the caller.
			return
		}
	})
}

type captureBody struct {
	io.ReadCloser
	trace     *Writer
	direction string
	mu        sync.Mutex
	buf       []byte
	err       error
	closed    bool
}

func newCaptureBody(src io.ReadCloser, trace *Writer, direction string) *captureBody {
	return &captureBody{ReadCloser: src, trace: trace, direction: direction}
}

func (b *captureBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	b.mu.Lock()
	defer b.mu.Unlock()
	if n > 0 {
		b.buf = append(b.buf, p[:n]...)
		if len(b.buf) > 16<<20 {
			b.err = fmt.Errorf("HTTP trace message exceeds 16 MiB")
		}
		if recErr := b.lines(false); recErr != nil && b.err == nil {
			b.err = recErr
		}
	}
	if err == io.EOF {
		if recErr := b.lines(true); recErr != nil && b.err == nil {
			b.err = recErr
		}
	}
	if b.err != nil {
		return n, fmt.Errorf("record HTTP body: %w", b.err)
	}
	return n, err
}

func (b *captureBody) lines(final bool) error {
	for {
		i := bytes.IndexByte(b.buf, '\n')
		if i < 0 {
			if final && len(bytes.TrimSpace(b.buf)) > 0 {
				if err := recordLine(b.trace, b.direction, b.buf); err != nil {
					return err
				}
				b.buf = nil
			}
			return nil
		}
		line := append([]byte(nil), b.buf[:i]...)
		b.buf = b.buf[i+1:]
		if err := recordLine(b.trace, b.direction, line); err != nil {
			return err
		}
	}
}

func (b *captureBody) Close() error {
	b.mu.Lock()
	if !b.closed {
		b.closed = true
		if err := b.lines(true); err != nil && b.err == nil {
			b.err = err
		}
	}
	err := b.err
	b.mu.Unlock()
	closeErr := b.ReadCloser.Close()
	if err != nil {
		return fmt.Errorf("record HTTP body: %w", err)
	}
	return closeErr
}

type captureResponseWriter struct {
	http.ResponseWriter
	trace *Writer
	mu    sync.Mutex
	buf   []byte
	err   error
}

func (w *captureResponseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *captureResponseWriter) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	if n > 0 {
		w.mu.Lock()
		w.buf = append(w.buf, p[:n]...)
		if len(w.buf) > 16<<20 {
			w.err = fmt.Errorf("HTTP trace message exceeds 16 MiB")
		} else if recErr := w.lines(false); recErr != nil && w.err == nil {
			w.err = recErr
		}
		w.mu.Unlock()
	}
	return n, err
}

func (w *captureResponseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *captureResponseWriter) lines(final bool) error {
	for {
		i := bytes.IndexByte(w.buf, '\n')
		if i < 0 {
			if final && len(bytes.TrimSpace(w.buf)) > 0 {
				if err := recordLine(w.trace, "send", w.buf); err != nil {
					return err
				}
				w.buf = nil
			}
			return nil
		}
		line := append([]byte(nil), w.buf[:i]...)
		w.buf = w.buf[i+1:]
		if err := recordLine(w.trace, "send", line); err != nil {
			return err
		}
	}
}

func (w *captureResponseWriter) flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.lines(true); err != nil && w.err == nil {
		w.err = err
	}
	if w.err != nil {
		return fmt.Errorf("record HTTP response: %w", w.err)
	}
	return nil
}
