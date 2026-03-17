// Package mcpspy records and streams MCP traffic.
package mcpspy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

const defaultBufferSize = 4096

// Options configures a Recorder.
type Options struct {
	PrettyJSON  bool
	PassThrough bool
	BufferSize  int
	Name        string
	SessionID   string
	TimeNow     func() time.Time
}

// Event is a structured view of a single captured message.
type Event struct {
	Seq       uint64          `json:"seq"`
	Time      time.Time       `json:"time"`
	Direction string          `json:"direction"`
	Raw       []byte          `json:"raw"`
	Formatted []byte          `json:"formatted"`
	Parsed    json.RawMessage `json:"parsed,omitempty"`
	Source    string          `json:"source"`
}

// MarshalJSON preserves byte fields as readable strings.
func (e Event) MarshalJSON() ([]byte, error) {
	type payload struct {
		Seq       uint64          `json:"seq"`
		Time      time.Time       `json:"time"`
		Direction string          `json:"direction"`
		Raw       string          `json:"raw"`
		Formatted string          `json:"formatted"`
		Parsed    json.RawMessage `json:"parsed,omitempty"`
		Source    string          `json:"source"`
	}
	return json.Marshal(payload{
		Seq:       e.Seq,
		Time:      e.Time,
		Direction: e.Direction,
		Raw:       string(e.Raw),
		Formatted: string(e.Formatted),
		Parsed:    e.Parsed,
		Source:    e.Source,
	})
}

// Recorder records MCP messages while exposing live subscriptions.
type Recorder struct {
	log  io.Writer
	opts Options

	mu       sync.Mutex
	seq      uint64
	buf      []Event
	bufCap   int
	bufStart int
	bufLen   int
	subs     map[int]chan Event
	nextSub  int
	scanners map[string]*jsonScanner
}

// New constructs a Recorder.
func New(log io.Writer, opts Options) *Recorder {
	if opts.BufferSize <= 0 {
		opts.BufferSize = defaultBufferSize
	}
	if opts.TimeNow == nil {
		opts.TimeNow = time.Now
	}
	return &Recorder{
		log:      log,
		opts:     opts,
		bufCap:   opts.BufferSize,
		buf:      make([]Event, opts.BufferSize),
		subs:     make(map[int]chan Event),
		scanners: make(map[string]*jsonScanner),
	}
}

// Reader wraps src and records messages read from it using dir.
func (r *Recorder) Reader(dir string, src io.Reader) io.Reader {
	return &recordingReader{
		src: src,
		rec: r,
		dir: dir,
	}
}

// Writer wraps dst and records messages written to it using dir.
func (r *Recorder) Writer(dir string, dst io.Writer) io.Writer {
	return &recordingWriter{
		dst: dst,
		rec: r,
		dir: dir,
	}
}

// Snapshot returns the currently buffered events in order.
func (r *Recorder) Snapshot() []Event {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]Event, 0, r.bufLen)
	for i := 0; i < r.bufLen; i++ {
		idx := (r.bufStart + i) % r.bufCap
		out = append(out, cloneEvent(r.buf[idx]))
	}
	return out
}

// Subscribe returns a live event stream and a cancellation function.
func (r *Recorder) Subscribe() (<-chan Event, func()) {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := r.nextSub
	r.nextSub++
	ch := make(chan Event, 128)
	r.subs[id] = ch
	cancel := func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		if ch, ok := r.subs[id]; ok {
			delete(r.subs, id)
			close(ch)
		}
	}
	return ch, cancel
}

// NewSessionID returns a random session identifier.
func NewSessionID() string {
	return newID()
}

type recordingReader struct {
	src io.Reader
	rec *Recorder
	dir string
}

func (r *recordingReader) Read(p []byte) (int, error) {
	n, err := r.src.Read(p)
	if n > 0 {
		r.rec.record(r.dir, p[:n], "reader")
	}
	return n, err
}

type recordingWriter struct {
	dst io.Writer
	rec *Recorder
	dir string
}

func (w *recordingWriter) Write(p []byte) (int, error) {
	n, err := w.dst.Write(p)
	if n > 0 {
		w.rec.record(w.dir, p[:n], "writer")
	}
	return n, err
}

func (r *Recorder) record(dir string, chunk []byte, source string) {
	r.mu.Lock()
	scanner := r.scanners[source+":"+dir]
	if scanner == nil {
		scanner = newJSONScanner()
		r.scanners[source+":"+dir] = scanner
	}
	messages := scanner.add(chunk)
	r.mu.Unlock()

	for _, msg := range messages {
		r.publish(dir, []byte(msg), source)
	}
}

func (r *Recorder) publish(dir string, raw []byte, source string) {
	now := r.opts.TimeNow()
	line, parsed := formatEntry(dir, raw, now, r.opts)
	ev := Event{
		Time:      now,
		Direction: dir,
		Raw:       append([]byte(nil), bytes.TrimSpace(raw)...),
		Formatted: line,
		Parsed:    parsed,
		Source:    source,
	}

	if r.log != nil {
		if _, err := r.log.Write(append(append([]byte(nil), line...), '\n')); err != nil {
			// The CLI handles reporting; dropping write errors here avoids poisoning the stream.
		}
	}

	r.mu.Lock()
	r.seq++
	ev.Seq = r.seq
	if r.bufLen < r.bufCap {
		idx := (r.bufStart + r.bufLen) % r.bufCap
		r.buf[idx] = cloneEvent(ev)
		r.bufLen++
	} else {
		r.buf[r.bufStart] = cloneEvent(ev)
		r.bufStart = (r.bufStart + 1) % r.bufCap
	}
	subs := make([]chan Event, 0, len(r.subs))
	for _, ch := range r.subs {
		subs = append(subs, ch)
	}
	r.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- cloneEvent(ev):
		default:
		}
	}
}

func cloneEvent(ev Event) Event {
	ev.Raw = append([]byte(nil), ev.Raw...)
	ev.Formatted = append([]byte(nil), ev.Formatted...)
	ev.Parsed = append([]byte(nil), ev.Parsed...)
	return ev
}

func formatEntry(dir string, raw []byte, now time.Time, opts Options) ([]byte, json.RawMessage) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, nil
	}
	formatted := formatJSON(trimmed, opts.PrettyJSON)
	var parsed json.RawMessage
	if json.Valid(trimmed) {
		parsed = append(json.RawMessage(nil), trimmed...)
	}
	if opts.PassThrough {
		return formatted, parsed
	}
	unixSec := now.Unix()
	unixMilli := now.UnixMilli() % 1000
	line := fmt.Sprintf("mcp-%s %s # %d.%03d", dir, formatted, unixSec, unixMilli)
	return []byte(line), parsed
}

func formatJSON(raw []byte, pretty bool) []byte {
	if !pretty || !json.Valid(raw) {
		return append([]byte(nil), raw...)
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return append([]byte(nil), raw...)
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return append([]byte(nil), raw...)
	}
	return out
}

type jsonScanner struct {
	buffer []byte
}

func newJSONScanner() *jsonScanner {
	return &jsonScanner{buffer: make([]byte, 0, 4096)}
}

func (s *jsonScanner) add(data []byte) []string {
	s.buffer = append(s.buffer, data...)
	var messages []string

	for {
		start := bytes.IndexByte(s.buffer, '{')
		if start < 0 {
			break
		}
		depth := 0
		inString := false
		escape := false
		end := -1
		for i := start; i < len(s.buffer); i++ {
			c := s.buffer[i]
			if escape {
				escape = false
				continue
			}
			if inString && c == '\\' {
				escape = true
				continue
			}
			if c == '"' {
				inString = !inString
				continue
			}
			if inString {
				continue
			}
			switch c {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					end = i + 1
					break
				}
			}
		}
		if end < 0 {
			break
		}
		candidate := bytes.TrimSpace(s.buffer[start:end])
		if json.Valid(candidate) {
			messages = append(messages, string(candidate))
			s.buffer = s.buffer[end:]
			continue
		}
		s.buffer = s.buffer[start+1:]
	}

	lines := strings.Split(string(s.buffer), "\n")
	if len(lines) > 1 {
		for _, line := range lines[:len(lines)-1] {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			messages = append(messages, line)
		}
		s.buffer = []byte(lines[len(lines)-1])
	}

	return messages
}
