package mcptrace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// Event describes one observed message or an unresolved request at end of input.
// It describes JSON-RPC exchanges, not MCP protocol lifecycle validity.
type Event struct {
	Kind          string          `json:"kind"`
	Session       string          `json:"session,omitempty"`
	Direction     string          `json:"direction,omitempty"`
	ID            json.RawMessage `json:"id,omitempty"`
	Method        string          `json:"method,omitempty"`
	Record        int             `json:"record"`
	RequestRecord int             `json:"requestRecord,omitempty"`
	Duration      *time.Duration  `json:"duration,omitempty"`
	Diagnostic    string          `json:"diagnostic,omitempty"`
}

type callKey struct{ session, direction, id string }
type observedCall struct {
	event     Event
	time      time.Time
	ambiguous bool
}

// Conversation reads messages incrementally and pairs responses with requests.
// Construct it with NewConversation. It retains outstanding requests in memory.
// Session metadata of the form session=NAME overrides the default session.
// Without session metadata, input must contain exactly one session; the analyzer
// cannot discover session boundaries from request IDs or initialization messages.
// String IDs compare by decoded value; numeric IDs compare by their JSON
// spelling, without conversion to floating point.
// Only send/recv (and matching send-SUFFIX/recv-SUFFIX) directions are paired.
type Conversation struct {
	reader  *Reader
	session string
	record  int
	pending map[callKey]observedCall
	tail    []Event
	done    bool
}

// NewConversation analyzes records from r within the explicitly supplied session
// scope. An empty session names a single anonymous session, not all sessions.
func NewConversation(r io.Reader, session string) *Conversation {
	return &Conversation{reader: NewReader(r), session: session, pending: make(map[callKey]observedCall)}
}

// Next returns the next observation, or io.EOF after reporting outstanding calls.
// Invalid trace input returns an error; callers must then treat input as truncated.
// Invalid JSON-RPC messages produce diagnostic events and do not alter pairing.
func (c *Conversation) Next() (Event, error) {
	if len(c.tail) > 0 {
		e := c.tail[0]
		c.tail = c.tail[1:]
		return e, nil
	}
	if c.done {
		return Event{}, io.EOF
	}
	r, err := c.reader.Read()
	if err != nil {
		c.done = true
		if err != io.EOF {
			return Event{}, err
		}
		for _, p := range c.pending {
			e := p.event
			e.Kind = "outstanding"
			e.Diagnostic = "no response observed before end of input"
			if p.ambiguous {
				e.Diagnostic = "duplicate outstanding request ID; pairing is ambiguous"
			}
			c.tail = append(c.tail, e)
		}
		sort.Slice(c.tail, func(i, j int) bool { return c.tail[i].Record < c.tail[j].Record })
		if len(c.tail) > 0 {
			e := c.tail[0]
			c.tail = c.tail[1:]
			return e, nil
		}
		return Event{}, io.EOF
	}
	c.record++
	e := Event{Session: c.session, Direction: r.Direction, Record: c.record}
	sessionSeen := false
	sessionConflict := false
	for _, metadata := range r.Metadata {
		if strings.HasPrefix(metadata, "session=") {
			session := strings.TrimPrefix(metadata, "session=")
			if session == "" || (sessionSeen && session != e.Session) {
				sessionConflict = true
			}
			e.Session = session
			sessionSeen = true
		}
	}
	var m map[string]json.RawMessage
	diagnostic := func(s string) (Event, error) { e.Kind = "diagnostic"; e.Diagnostic = s; return e, nil }
	if sessionConflict {
		return diagnostic("empty or conflicting session metadata; pairing skipped")
	}
	if err := json.Unmarshal(r.Message, &m); err != nil || m == nil {
		return diagnostic("expected a JSON-RPC object; batches are not analyzed")
	}
	var version string
	if json.Unmarshal(m["jsonrpc"], &version) != nil || version != "2.0" {
		return diagnostic("expected jsonrpc version 2.0")
	}
	id, hasID := m["id"]
	identity := ""
	if hasID {
		var valid bool
		identity, valid = idIdentity(id)
		if !valid {
			return diagnostic("request ID must be a string or number")
		}
		e.ID = append(json.RawMessage(nil), id...)
	}
	if params, exists := m["params"]; exists {
		params = bytes.TrimSpace(params)
		if len(params) == 0 || (params[0] != '{' && params[0] != '[') {
			return diagnostic("params must be an object or array")
		}
	}
	opposite, ok := oppositeDirection(r.Direction)
	if !ok {
		return diagnostic("direction has no known opposite")
	}
	_, hasResult := m["result"]
	_, hasError := m["error"]
	method, hasMethod := m["method"]
	if hasMethod {
		if json.Unmarshal(method, &e.Method) != nil || e.Method == "" || hasResult || hasError {
			return diagnostic("invalid request or notification shape")
		}
		if hasID {
			e.Kind = "request"
			k := callKey{e.Session, r.Direction, identity}
			if p, exists := c.pending[k]; exists {
				p.ambiguous = true
				c.pending[k] = p
				e.Diagnostic = "duplicate outstanding request ID; pairing is ambiguous"
			} else {
				c.pending[k] = observedCall{event: e, time: r.Time}
			}
		} else {
			e.Kind = "notification"
			if e.Method == "notifications/cancelled" {
				var params struct {
					RequestID json.RawMessage `json:"requestId"`
				}
				if json.Unmarshal(m["params"], &params) != nil || len(params.RequestID) == 0 {
					return diagnostic("cancellation has no requestId")
				}
				target, valid := idIdentity(params.RequestID)
				if !valid {
					return diagnostic("cancellation requestId must be a string or number")
				}
				e.Kind = "cancellation"
				e.ID = params.RequestID
				if p, exists := c.pending[callKey{e.Session, r.Direction, target}]; exists && !p.ambiguous {
					e.RequestRecord = p.event.Record
				} else {
					e.Diagnostic = "cancellation target is absent or ambiguous"
				}
			}
		}
		return e, nil
	}
	if !hasID || hasResult == hasError {
		return diagnostic("invalid response shape")
	}
	if hasError {
		var responseError struct {
			Code    *int    `json:"code"`
			Message *string `json:"message"`
		}
		if json.Unmarshal(m["error"], &responseError) != nil || responseError.Code == nil || responseError.Message == nil {
			return diagnostic("invalid response error")
		}
	}
	e.Kind = "response"
	k := callKey{e.Session, opposite, identity}
	p, exists := c.pending[k]
	if !exists {
		e.Diagnostic = "no matching request observed"
		return e, nil
	}
	if p.ambiguous {
		e.Diagnostic = "duplicate outstanding request ID; pairing is ambiguous"
		return e, nil
	}
	delete(c.pending, k)
	e.Method = p.event.Method
	e.RequestRecord = p.event.Record
	if !p.time.IsZero() && !r.Time.IsZero() {
		d := r.Time.Sub(p.time)
		if d < 0 {
			e.Diagnostic = "response timestamp precedes request"
		} else {
			e.Duration = &d
		}
	}
	return e, nil
}

func oppositeDirection(direction string) (string, bool) {
	for _, pair := range [][2]string{{"send", "recv"}, {"recv", "send"}} {
		if direction == pair[0] {
			return pair[1], true
		}
		if strings.HasPrefix(direction, pair[0]+"-") {
			return pair[1] + strings.TrimPrefix(direction, pair[0]), true
		}
	}
	return "", false
}

// String formats an event as one human-readable line.
func (e Event) String() string {
	s := fmt.Sprintf("%d %s %s", e.Record, e.Direction, e.Kind)
	if e.Session != "" {
		s += " session=" + e.Session
	}
	if len(e.ID) > 0 {
		s += " id=" + string(e.ID)
	}
	if e.Method != "" {
		s += " " + e.Method
	}
	if e.RequestRecord > 0 {
		s += fmt.Sprintf(" request=%d", e.RequestRecord)
	}
	if e.Duration != nil {
		s += " duration=" + e.Duration.String()
	}
	if e.Diagnostic != "" {
		s += " (" + e.Diagnostic + ")"
	}
	return s
}

func idIdentity(raw json.RawMessage) (string, bool) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if decoder.Decode(&value) != nil {
		return "", false
	}
	switch v := value.(type) {
	case string:
		return "s:" + v, true
	case json.Number:
		return "n:" + v.String(), true
	}
	return "", false
}
