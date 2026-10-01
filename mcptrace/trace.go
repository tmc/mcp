package mcptrace

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Record is one direction-tagged JSON-RPC message.
type Record struct {
	// Direction is send, recv, or a direction extension such as send-shadow.
	Direction string
	// Message is the JSON-RPC message as it appeared in the trace.
	Message json.RawMessage
	// Time is the optional Unix timestamp. The zero value means no timestamp.
	Time time.Time
	// Metadata contains optional whitespace-separated fields after the timestamp.
	Metadata []string
}

// Reader reads one record per line. The zero value is not usable; construct it
// with NewReader.
type Reader struct {
	scanner *bufio.Scanner
	line    int
	err     error
}

// NewReader returns a reader for an MCP trace.
func NewReader(r io.Reader) *Reader {
	if r == nil {
		return &Reader{err: fmt.Errorf("nil trace reader")}
	}
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 4096), 16<<20)
	return &Reader{scanner: s}
}

// Read returns the next record, or io.EOF at the end of the trace.
func (r *Reader) Read() (Record, error) {
	if r.err != nil {
		return Record{}, r.err
	}
	for r.scanner.Scan() {
		r.line++
		line := strings.TrimSpace(r.scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		rec, err := ParseRecord(line)
		if err != nil {
			return Record{}, fmt.Errorf("line %d: %w", r.line, err)
		}
		return rec, nil
	}
	if err := r.scanner.Err(); err != nil {
		return Record{}, fmt.Errorf("read trace: %w", err)
	}
	return Record{}, io.EOF
}

// ParseRecord parses one record line without consuming a stream.
func ParseRecord(line string) (Record, error) {
	line = strings.TrimSpace(line)
	fields := strings.SplitN(line, " ", 2)
	if len(fields) != 2 || !strings.HasPrefix(fields[0], "mcp-") {
		return Record{}, fmt.Errorf("invalid record")
	}
	direction := strings.TrimPrefix(fields[0], "mcp-")
	if !validDirection(direction) {
		return Record{}, fmt.Errorf("invalid direction")
	}
	payload := fields[1]
	for i := strings.LastIndex(payload, " # "); i >= 0; i = strings.LastIndex(payload[:i], " # ") {
		message := strings.TrimSpace(payload[:i])
		if !json.Valid([]byte(message)) {
			continue
		}
		suffix := strings.Fields(payload[i+3:])
		if len(suffix) == 0 {
			return Record{}, fmt.Errorf("invalid timestamp")
		}
		stamp, err := parseTimestamp(suffix[0])
		if err != nil {
			return Record{}, fmt.Errorf("invalid timestamp: %w", err)
		}
		return Record{
			Direction: direction,
			Message:   append(json.RawMessage(nil), message...),
			Time:      stamp,
			Metadata:  append([]string(nil), suffix[1:]...),
		}, nil
	}
	if !json.Valid([]byte(payload)) {
		return Record{}, fmt.Errorf("invalid JSON")
	}
	return Record{Direction: direction, Message: append(json.RawMessage(nil), payload...)}, nil
}

func parseTimestamp(text string) (time.Time, error) {
	parts := strings.SplitN(text, ".", 2)
	seconds, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	var nanos int64
	if len(parts) == 2 {
		fraction := parts[1]
		if len(fraction) == 0 || len(fraction) > 9 {
			return time.Time{}, fmt.Errorf("fraction must contain 1 to 9 digits")
		}
		fraction += strings.Repeat("0", 9-len(fraction))
		nanos, err = strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return time.Time{}, err
		}
	}
	return time.Unix(seconds, nanos), nil
}

// Writer writes records in MCP trace format. Construct it with NewWriter.
type Writer struct {
	w  io.Writer
	mu sync.Mutex
}

// ReadCloser records newline-delimited JSON messages read from src while
// returning the original bytes to the caller.
func ReadCloser(src io.ReadCloser, trace *Writer, direction string) io.ReadCloser {
	return &recordingReader{ReadCloser: src, trace: trace, direction: direction}
}

// WriteCloser records newline-delimited JSON messages written to dst while
// forwarding the original bytes.
func WriteCloser(dst io.WriteCloser, trace *Writer, direction string) io.WriteCloser {
	return &recordingWriter{WriteCloser: dst, trace: trace, direction: direction}
}

type recordingReader struct {
	io.ReadCloser
	trace     *Writer
	direction string
	buf       []byte
	err       error
}

func (r *recordingReader) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	if n > 0 {
		r.buf = append(r.buf, p[:n]...)
		if recErr := r.recordLines(false); recErr != nil {
			r.err = recErr
		}
	}
	if err == io.EOF {
		if recErr := r.recordLines(true); recErr != nil {
			r.err = recErr
		}
	}
	if r.err != nil {
		return n, fmt.Errorf("record stream: %w", r.err)
	}
	return n, err
}

func (r *recordingReader) recordLines(final bool) error {
	for {
		i := bytes.IndexByte(r.buf, '\n')
		if i < 0 {
			if final && len(strings.TrimSpace(string(r.buf))) > 0 {
				if err := r.recordLine(r.buf); err != nil {
					return err
				}
				r.buf = nil
			}
			if len(r.buf) > 16<<20 {
				return fmt.Errorf("stream line exceeds 16 MiB")
			}
			return nil
		}
		line := append([]byte(nil), r.buf[:i]...)
		r.buf = r.buf[i+1:]
		if err := r.recordLine(line); err != nil {
			return err
		}
	}
}

func (r *recordingReader) recordLine(line []byte) error {
	line = bytes.TrimSpace(line)
	if bytes.HasPrefix(line, []byte("data:")) {
		line = bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
	}
	if len(line) == 0 || !json.Valid(line) {
		return nil
	}
	return r.trace.Write(Record{Direction: r.direction, Message: line, Time: time.Now()})
}

func (r *recordingReader) Close() error {
	if len(r.buf) > 0 {
		if err := r.recordLines(true); err != nil && r.err == nil {
			r.err = err
		}
	}
	closeErr := r.ReadCloser.Close()
	if r.err != nil {
		return fmt.Errorf("record stream: %w", r.err)
	}
	return closeErr
}

type recordingWriter struct {
	io.WriteCloser
	trace     *Writer
	direction string
	mu        sync.Mutex
	buf       []byte
	err       error
}

func (w *recordingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n, err := w.WriteCloser.Write(p)
	if n > 0 {
		w.buf = append(w.buf, p[:n]...)
		for {
			i := bytes.IndexByte(w.buf, '\n')
			if i < 0 {
				break
			}
			line := append([]byte(nil), w.buf[:i]...)
			w.buf = w.buf[i+1:]
			if err := recordLine(w.trace, w.direction, line); err != nil {
				w.err = err
				break
			}
		}
	}
	if w.err != nil {
		return n, fmt.Errorf("record stream: %w", w.err)
	}
	return n, err
}

func (w *recordingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.buf) > 0 {
		if err := recordLine(w.trace, w.direction, w.buf); err != nil && w.err == nil {
			w.err = err
		}
		w.buf = nil
	}
	closeErr := w.WriteCloser.Close()
	if w.err != nil {
		return fmt.Errorf("record stream: %w", w.err)
	}
	return closeErr
}

func recordLine(trace *Writer, direction string, line []byte) error {
	line = bytes.TrimSpace(line)
	if bytes.HasPrefix(line, []byte("data:")) {
		line = bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
	}
	if len(line) == 0 || !json.Valid(line) {
		return nil
	}
	return trace.Write(Record{Direction: direction, Message: line, Time: time.Now()})
}

// NewWriter returns a writer for an MCP trace.
func NewWriter(w io.Writer) *Writer { return &Writer{w: w} }

// Write writes one record. A nonzero Time is serialized as a Unix timestamp.
func (w *Writer) Write(rec Record) error {
	if w == nil || w.w == nil {
		return fmt.Errorf("nil trace writer")
	}
	if !validDirection(rec.Direction) {
		return fmt.Errorf("invalid direction %q", rec.Direction)
	}
	if !json.Valid(rec.Message) {
		return fmt.Errorf("invalid JSON message")
	}
	line := fmt.Sprintf("mcp-%s %s", rec.Direction, bytes.TrimSpace(rec.Message))
	if !rec.Time.IsZero() {
		line += fmt.Sprintf(" # %d.%03d", rec.Time.Unix(), rec.Time.Nanosecond()/1e6)
		if len(rec.Metadata) > 0 {
			line += " " + strings.Join(rec.Metadata, " ")
		}
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, err := io.WriteString(w.w, line+"\n"); err != nil {
		return fmt.Errorf("write trace: %w", err)
	}
	return nil
}

func validDirection(direction string) bool {
	if direction == "" {
		return false
	}
	for _, r := range direction {
		if (r < 'a' || r > 'z') && r != '-' {
			return false
		}
	}
	return direction[0] != '-' && direction[len(direction)-1] != '-'
}
