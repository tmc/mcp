package mcptracediff

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tmc/mcp/mcptrace"
)

// Options configures comparison. Session supplies the anonymous stream scope;
// explicit session= metadata overrides it. Timing requires both durations known.
type Options struct {
	Session string
	Timing  bool
}

// Difference identifies a changed, missing, or additional observed call group.
type Difference struct {
	Kind          string          `json:"kind"`
	Session       string          `json:"session,omitempty"`
	Direction     string          `json:"direction"`
	Method        string          `json:"method"`
	LeftRecord    int             `json:"leftRecord,omitempty"`
	RightRecord   int             `json:"rightRecord,omitempty"`
	Detail        string          `json:"detail"`
	Params        json.RawMessage `json:"params,omitempty"`
	LeftResponse  json.RawMessage `json:"leftResponse,omitempty"`
	RightResponse json.RawMessage `json:"rightResponse,omitempty"`
}

type identity struct{ session, direction, method, params string }
type call struct {
	params   json.RawMessage
	response json.RawMessage
	record   int
	result   string
	duration *time.Duration
}
type callSet map[identity][]*call

// Compare reads both traces and returns differences in deterministic order.
// An error means pairing or comparison could not be established. A clean EOF
// with outstanding calls is incomplete, not proof that an operation completed.
func Compare(left, right io.Reader, opts Options) ([]Difference, error) {
	a, err := read(left, opts)
	if err != nil {
		return nil, fmt.Errorf("left trace: %w", err)
	}
	b, err := read(right, opts)
	if err != nil {
		return nil, fmt.Errorf("right trace: %w", err)
	}
	keys := make([]identity, 0, len(a)+len(b))
	for key := range a {
		keys = append(keys, key)
	}
	for key := range b {
		if _, ok := a[key]; !ok {
			keys = append(keys, key)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		if a.session != b.session {
			return a.session < b.session
		}
		if a.direction != b.direction {
			return a.direction < b.direction
		}
		if a.method != b.method {
			return a.method < b.method
		}
		return a.params < b.params
	})
	differences := make([]Difference, 0)
	for _, key := range keys {
		x, y := a[key], b[key]
		base := Difference{Session: key.session, Direction: key.direction, Method: key.method}
		if len(x) > 0 {
			base.Params = x[0].params
		} else if len(y) > 0 {
			base.Params = y[0].params
		}
		if len(x) != len(y) {
			base.Kind = "count"
			base.Detail = fmt.Sprintf("call count changed: %d -> %d; individual alignment unavailable", len(x), len(y))
			if len(x) > 0 {
				base.LeftRecord = x[0].record
			}
			if len(y) > 0 {
				base.RightRecord = y[0].record
			}
			differences = append(differences, base)
			continue
		}
		for index, p := range x {
			q := y[index]
			d := base
			d.LeftRecord = p.record
			d.RightRecord = q.record
			if p.result != q.result {
				d.LeftResponse = p.response
				d.RightResponse = q.response
				d.Kind = "result"
				d.Detail = "response result or error changed"
				differences = append(differences, d)
			}
			if opts.Timing && *p.duration != *q.duration {
				d.Kind = "duration"
				d.Detail = fmt.Sprintf("duration changed: %s -> %s", *p.duration, *q.duration)
				differences = append(differences, d)
			}
		}
	}
	return differences, nil
}

func read(input io.Reader, opts Options) (callSet, error) {
	var records []mcptrace.Record
	var trace bytes.Buffer
	reader := mcptrace.NewReader(io.TeeReader(input, &trace))
	for {
		r, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("incomplete or invalid input: %w", err)
		}
		if _, err := canonical(r.Message); err != nil {
			return nil, fmt.Errorf("record %d: %w", len(records)+1, err)
		}
		records = append(records, r)
	}
	conversation := mcptrace.NewConversation(&trace, opts.Session)
	calls := make(callSet)
	pending := make(map[int]*call)
	requestKeys := make(map[int]identity)
	active := make(map[identity]int)
	for {
		event, err := conversation.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if event.Diagnostic != "" && !(event.Diagnostic == "response timestamp precedes request" && !opts.Timing) {
			return nil, fmt.Errorf("record %d: %s", event.Record, event.Diagnostic)
		}
		switch event.Kind {
		case "request":
			var message map[string]json.RawMessage
			if err := json.Unmarshal(records[event.Record-1].Message, &message); err != nil {
				return nil, err
			}
			params := "absent"
			if raw, ok := message["params"]; ok {
				params, err = canonical(raw)
				if err != nil {
					return nil, err
				}
			}
			key := identity{event.Session, event.Direction, event.Method, params}
			if active[key] > 0 {
				return nil, fmt.Errorf("record %d: overlapping identical %s calls are ambiguous", event.Record, event.Method)
			}
			observed := &call{record: event.Record, params: message["params"]}
			calls[key] = append(calls[key], observed)
			pending[event.Record] = observed
			requestKeys[event.Record] = key
			active[key]++
		case "response":
			observed := pending[event.RequestRecord]
			if observed == nil {
				return nil, fmt.Errorf("record %d: response has no correlated request", event.Record)
			}
			var message map[string]json.RawMessage
			if err := json.Unmarshal(records[event.Record-1].Message, &message); err != nil {
				return nil, err
			}
			payload := map[string]json.RawMessage{}
			if result, ok := message["result"]; ok {
				payload["result"] = result
			} else {
				payload["error"] = message["error"]
			}
			raw, err := json.Marshal(payload)
			if err != nil {
				return nil, err
			}
			observed.response = raw
			observed.result, err = canonical(raw)
			if err != nil {
				return nil, err
			}
			observed.duration = event.Duration
			if opts.Timing && event.Duration == nil {
				return nil, fmt.Errorf("record %d: duration unavailable; both timestamps are required for -timing", event.Record)
			}
			active[requestKeys[event.RequestRecord]]--
			delete(pending, event.RequestRecord)
		}
	}
	return calls, nil
}

// canonical uses type tags and normalized decimal coefficients/exponents. It
// never rounds through float64 or expands an exponent into a huge integer.
func canonical(data []byte) (string, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value func() (string, error)
	value = func() (string, error) {
		token, err := decoder.Token()
		if err != nil {
			return "", err
		}
		switch v := token.(type) {
		case nil:
			return "null", nil
		case bool:
			return strconv.FormatBool(v), nil
		case string:
			return "s" + strconv.Quote(v), nil
		case json.Number:
			return number(v.String()), nil
		case json.Delim:
			switch v {
			case '{':
				fields := make(map[string]string)
				for decoder.More() {
					token, err := decoder.Token()
					if err != nil {
						return "", err
					}
					key := token.(string)
					if _, ok := fields[key]; ok {
						return "", fmt.Errorf("duplicate JSON object key %q", key)
					}
					item, err := value()
					if err != nil {
						return "", err
					}
					fields[key] = item
				}
				if _, err := decoder.Token(); err != nil {
					return "", err
				}
				keys := make([]string, 0, len(fields))
				for key := range fields {
					keys = append(keys, key)
				}
				sort.Strings(keys)
				var out strings.Builder
				out.WriteByte('{')
				for _, key := range keys {
					out.WriteString(strconv.Quote(key))
					out.WriteByte(':')
					out.WriteString(fields[key])
					out.WriteByte(',')
				}
				out.WriteByte('}')
				return out.String(), nil
			case '[':
				var out strings.Builder
				out.WriteByte('[')
				for decoder.More() {
					item, err := value()
					if err != nil {
						return "", err
					}
					out.WriteString(item)
					out.WriteByte(',')
				}
				if _, err := decoder.Token(); err != nil {
					return "", err
				}
				out.WriteByte(']')
				return out.String(), nil
			}
		}
		return "", fmt.Errorf("invalid JSON value")
	}
	result, err := value()
	if err != nil {
		return "", err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return "", fmt.Errorf("extra JSON data")
	}
	return result, nil
}

func number(text string) string {
	sign := ""
	if strings.HasPrefix(text, "-") {
		sign = "-"
		text = text[1:]
	}
	exponent := new(big.Int)
	if index := strings.IndexAny(text, "eE"); index >= 0 {
		exponent.SetString(text[index+1:], 10)
		text = text[:index]
	}
	if index := strings.IndexByte(text, '.'); index >= 0 {
		exponent.Sub(exponent, big.NewInt(int64(len(text)-index-1)))
		text = text[:index] + text[index+1:]
	}
	text = strings.TrimLeft(text, "0")
	if text == "" {
		return "n0"
	}
	trimmed := strings.TrimRight(text, "0")
	exponent.Add(exponent, big.NewInt(int64(len(text)-len(trimmed))))
	return "n" + sign + trimmed + "e" + exponent.String()
}
