package mcpfixture

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tmc/mcp/mcptrace"
)

// Options identifies the tools/call request to extract. Record is the one-based
// record index in the complete input, excluding comments and blank lines.
// Session selects session= metadata; empty selects the anonymous single session.
type Options struct {
	Record  int
	Session string
}

// Fixture contains the captured parameters and observed result. Expected is an
// assertion candidate that the developer must review, not a correctness claim.
type Fixture struct {
	ProtocolVersion          string              `json:"protocolVersion"`
	RequestedProtocolVersion string              `json:"requestedProtocolVersion,omitempty"`
	ClientInfo               *sdk.Implementation `json:"clientInfo"`
	ClientCapabilities       json.RawMessage     `json:"clientCapabilities,omitempty"`
	Prerequisites            []Prerequisite      `json:"prerequisites,omitempty"`
	Params                   *sdk.CallToolParams `json:"params"`
	Expected                 json.RawMessage     `json:"expected"`
	SourceRecord             int                 `json:"sourceRecord"`
	SourceSession            string              `json:"sourceSession,omitempty"`
}

type message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	Result  json.RawMessage `json:"result"`
	Error   json.RawMessage `json:"error"`
}

// Convert writes an executable txtar fixture for the explicitly selected request.
// It validates the handshake and sequential prerequisites through the selected
// response, and reads the full trace for syntax errors before writing output.
// Later exchanges are outside the extracted boundary. The
// generated script requires mcpfixture on PATH and MCP_SERVER set by its runner.
func Convert(in io.Reader, out io.Writer, opts Options) error {
	if opts.Record < 1 {
		return fmt.Errorf("request record must be positive")
	}
	reader := mcptrace.NewReader(in)
	var records []mcptrace.Record
	selectedPos := -1
	for index := 1; ; index++ {
		r, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("incomplete trace: %w", err)
		}
		scope := ""
		seen := false
		for _, metadata := range r.Metadata {
			if strings.HasPrefix(metadata, "session=") {
				next := strings.TrimPrefix(metadata, "session=")
				if next == "" || (seen && scope != next) {
					return fmt.Errorf("record %d: ambiguous session metadata", index)
				}
				scope = next
				seen = true
			}
		}
		if index == opts.Record {
			if scope != opts.Session {
				return fmt.Errorf("selected record belongs to session %q, not %q", scope, opts.Session)
			}
			selectedPos = len(records)
		}
		if scope == opts.Session {
			if err := uniqueKeys(r.Message); err != nil {
				return fmt.Errorf("record %d: %w", index, err)
			}
			records = append(records, r)
		}
	}
	if selectedPos < 0 {
		return fmt.Errorf("request record %d not found", opts.Record)
	}
	messages := make([]message, len(records))
	for i, r := range records {
		if err := json.Unmarshal(r.Message, &messages[i]); err != nil || messages[i].JSONRPC != "2.0" {
			return fmt.Errorf("record sequence %d: expected JSON-RPC 2.0 object", i+1)
		}
	}
	if selectedPos < 2 {
		return fmt.Errorf("selected call has unsupported preceding interactions")
	}
	if selectedPos+1 >= len(records) {
		return fmt.Errorf("missing selected response; got %d records", len(records))
	}
	fixture, first, err := handshake(records, messages)
	if err != nil {
		return err
	}
	if selectedPos < first {
		return fmt.Errorf("selected call has unsupported preceding interactions")
	}
	for i := first; i < selectedPos; i += 2 {
		if i+1 >= selectedPos {
			return fmt.Errorf("extra interactions: incomplete prerequisite or concurrency")
		}
		if records[i].Direction != "send" || records[i+1].Direction != "recv" {
			return fmt.Errorf("extra interactions: callbacks and concurrency are unsupported")
		}
		m := messages[i]
		if !listMethod(m.Method) {
			return fmt.Errorf("unsupported preceding interactions: %s", m.Method)
		}
		if err := request(m, m.Method); err != nil {
			return err
		}
		if err := response(messages[i+1], m.ID); err != nil {
			return err
		}
		if fixture.ProtocolVersion < "2026-07-28" && (len(m.Params) == 0 || bytes.Equal(bytes.TrimSpace(m.Params), []byte("null"))) {
			m.Params = json.RawMessage(`{}`)
		}
		if err := requestContext(m.Params, fixture); err != nil {
			return err
		}
		prerequisite := Prerequisite{Method: m.Method, Params: m.Params, Expected: messages[i+1].Result}
		if err := validatePrerequisite(prerequisite); err != nil {
			return err
		}
		fixture.Prerequisites = append(fixture.Prerequisites, prerequisite)
	}
	if records[selectedPos].Direction != "send" || records[selectedPos+1].Direction != "recv" {
		return fmt.Errorf("extra interactions: callbacks and concurrency are unsupported")
	}
	if err := request(messages[selectedPos], "tools/call"); err != nil {
		return err
	}
	if err := response(messages[selectedPos+1], messages[selectedPos].ID); err != nil {
		return err
	}
	if err := requestContext(messages[selectedPos].Params, fixture); err != nil {
		return err
	}
	var rawParams map[string]json.RawMessage
	if json.Unmarshal(messages[selectedPos].Params, &rawParams) != nil {
		return fmt.Errorf("tool parameters must be an object")
	}
	if arguments, ok := rawParams["arguments"]; ok {
		var object map[string]json.RawMessage
		if json.Unmarshal(arguments, &object) != nil || object == nil {
			return fmt.Errorf("tool arguments must be an object when present")
		}
	}
	var params sdk.CallToolParams
	decoder := json.NewDecoder(bytes.NewReader(messages[selectedPos].Params))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	if err := decoder.Decode(&params); err != nil || params.Name == "" {
		return fmt.Errorf("invalid or unsupported tools/call parameters")
	}
	fixture.Params = &params
	fixture.Expected = messages[selectedPos+1].Result
	fixture.SourceRecord = opts.Record
	fixture.SourceSession = opts.Session
	if err := validateFixture(fixture); err != nil {
		return err
	}
	data, err := json.MarshalIndent(fixture, "", "  ")
	if err != nil {
		return fmt.Errorf("encode fixture: %w", err)
	}
	script := "# Observed response: review before treating it as the expected behavior.\n# From the toolkit checkout: go install ./cmd/mcpfixture\n# Set MCP_SERVER to the server executable; edit this command for arguments.\nexec mcpfixture check fixture.json -- $MCP_SERVER\n\n-- fixture.json --\n"
	if _, err = io.WriteString(out, script+string(data)+"\n"); err != nil {
		return fmt.Errorf("write fixture: %w", err)
	}
	return nil
}

func request(m message, method string) error {
	if m.Method != method || len(m.Result) != 0 || len(m.Error) != 0 || !validID(m.ID) {
		return fmt.Errorf("expected %s request with a string or numeric ID", method)
	}
	return nil
}
func response(m message, id json.RawMessage) error {
	if m.Method != "" || len(m.Params) != 0 || len(m.Error) != 0 || len(m.Result) == 0 || !sameJSON(m.ID, id) {
		return fmt.Errorf("expected matching successful response")
	}
	return nil
}
func validID(raw json.RawMessage) bool {
	var value any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if d.Decode(&value) != nil {
		return false
	}
	switch value.(type) {
	case string, json.Number:
		return true
	}
	return false
}
func sameJSON(a, b []byte) bool {
	decode := func(data []byte) (any, error) {
		var value any
		d := json.NewDecoder(bytes.NewReader(data))
		d.UseNumber()
		err := d.Decode(&value)
		return value, err
	}
	av, ae := decode(a)
	bv, be := decode(b)
	return ae == nil && be == nil && reflect.DeepEqual(av, bv)
}

// Check calls the selected tool through an already connected session and compares
// the complete result with the observed expectation. Construct the session client
// with fixture.ClientInfo and ClientOptions to preserve its recorded configuration.
// Calling Check executes the
// tool; only call it against a server approved for this regression test.
// Object key order is ignored; JSON number spellings remain significant.
func Check(ctx context.Context, session *sdk.ClientSession, fixture Fixture) error {
	if session == nil {
		return fmt.Errorf("invalid fixture or session")
	}
	if err := validateFixture(fixture); err != nil {
		return err
	}
	if session.InitializeResult() == nil {
		return fmt.Errorf("session is not initialized")
	}
	if session.InitializeResult().ProtocolVersion != fixture.ProtocolVersion {
		return fmt.Errorf("protocol version differs: got %s, want %s", session.InitializeResult().ProtocolVersion, fixture.ProtocolVersion)
	}
	for _, p := range fixture.Prerequisites {
		if err := checkPrerequisite(ctx, session, p); err != nil {
			return err
		}
	}
	result, err := session.CallTool(ctx, fixture.Params)
	if err != nil {
		return fmt.Errorf("call tool: %w", err)
	}
	actual, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("encode result: %w", err)
	}
	if !sameJSON(actual, fixture.Expected) {
		return fmt.Errorf("observed result changed: got %s, want %s", actual, fixture.Expected)
	}
	return nil
}

// uniqueKeys rejects ambiguous JSON objects, including nested protocol data.
func uniqueKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value func() error
	value = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delimiter {
		case '{':
			keys := make(map[string]bool)
			for decoder.More() {
				key, err := decoder.Token()
				if err != nil {
					return err
				}
				name := key.(string)
				if keys[name] {
					return fmt.Errorf("duplicate JSON object key %q", name)
				}
				keys[name] = true
				if err := value(); err != nil {
					return err
				}
			}
		case '[':
			for decoder.More() {
				if err := value(); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("unexpected JSON delimiter")
		}
		_, err = decoder.Token()
		return err
	}
	if err := value(); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return fmt.Errorf("extra JSON data")
	}
	return nil
}

// Read decodes and validates a fixture JSON artifact without executing it.
// It preserves JSON numbers exactly and rejects duplicate object keys.
func Read(r io.Reader) (Fixture, error) {
	var fixture Fixture
	data, err := io.ReadAll(io.LimitReader(r, 16<<20+1))
	if err != nil {
		return fixture, fmt.Errorf("read fixture: %w", err)
	}
	if len(data) > 16<<20 {
		return fixture, fmt.Errorf("fixture exceeds 16 MiB")
	}
	if err := uniqueKeys(data); err != nil {
		return fixture, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fixture); err != nil {
		return fixture, fmt.Errorf("decode fixture: %w", err)
	}
	var raw struct {
		Params map[string]json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fixture, err
	}
	if arguments, exists := raw.Params["arguments"]; exists {
		var object map[string]json.RawMessage
		if json.Unmarshal(arguments, &object) != nil || object == nil {
			return fixture, fmt.Errorf("tool arguments must be an object when present")
		}
	}
	return fixture, validateFixture(fixture)
}

func validateFixture(f Fixture) error {
	if f.Params == nil || f.Params.Name == "" || f.ProtocolVersion == "" || !json.Valid(f.Expected) {
		return fmt.Errorf("invalid fixture or session")
	}
	if !slices.Contains(sdk.SupportedProtocolVersions(), f.ProtocolVersion) {
		return fmt.Errorf("unsupported fixture protocol version")
	}
	if f.RequestedProtocolVersion != "" && !slices.Contains(sdk.SupportedProtocolVersions(), f.RequestedProtocolVersion) {
		return fmt.Errorf("unsupported requested fixture protocol version")
	}
	if f.ClientInfo == nil || f.ClientInfo.Name == "" || f.ClientInfo.Version == "" {
		return fmt.Errorf("fixture clientInfo is required")
	}
	if len(f.Params.InputResponses) > 0 || f.Params.RequestState != "" {
		return fmt.Errorf("tool metadata or continuation state requires unsupported prerequisites")
	}
	if f.Params.Arguments != nil {
		data, err := json.Marshal(f.Params.Arguments)
		if err != nil {
			return fmt.Errorf("encode arguments: %w", err)
		}
		var object map[string]json.RawMessage
		if json.Unmarshal(data, &object) != nil || object == nil {
			return fmt.Errorf("tool arguments must be an object when present")
		}
	}
	if _, err := ClientOptions(f); err != nil {
		return err
	}
	for _, p := range f.Prerequisites {
		if err := validatePrerequisite(p); err != nil {
			return err
		}
		if err := requestContext(p.Params, f); err != nil {
			return err
		}
	}
	if err := validateCursors(f.Prerequisites); err != nil {
		return err
	}
	raw, err := json.Marshal(f.Params)
	if err != nil {
		return err
	}
	if err := requestContext(raw, f); err != nil {
		return err
	}
	var result sdk.CallToolResult
	if json.Unmarshal(f.Expected, &result) != nil || result.Content == nil {
		return fmt.Errorf("tool result must contain a content array")
	}
	return nil
}
