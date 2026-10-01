package mcpfixture

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tmc/mcp/mcptrace"
)

// Prerequisite records a read-only catalog list operation before the selected
// call. Check executes it and checks its observed result before calling the tool.
type Prerequisite struct {
	Method   string          `json:"method"`
	Params   json.RawMessage `json:"params"`
	Expected json.RawMessage `json:"expected"`
}

// ClientOptions returns the recorded client configuration. Callback capabilities
// are refused because a fixture cannot reconstruct the corresponding handlers.
func ClientOptions(f Fixture) (*sdk.ClientOptions, error) {
	raw := f.ClientCapabilities
	if len(raw) == 0 {
		raw = json.RawMessage(`{}`)
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) != nil || object == nil {
		return nil, fmt.Errorf("invalid client capabilities")
	}
	for key, rawValue := range object {
		if key != "experimental" && key != "extensions" {
			return nil, fmt.Errorf("client capabilities require unsupported fixture prerequisites: %s", key)
		}
		var settings map[string]json.RawMessage
		if json.Unmarshal(rawValue, &settings) != nil || len(settings) == 0 {
			return nil, fmt.Errorf("client capability %s cannot be reproduced faithfully", key)
		}
	}
	var caps sdk.ClientCapabilities
	if err := decode(raw, &caps); err != nil {
		return nil, fmt.Errorf("client capabilities: %w", err)
	}
	return &sdk.ClientOptions{Capabilities: &caps}, nil
}
func decode(raw []byte, out any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	d.UseNumber()
	return d.Decode(out)
}
func handshake(records []mcptrace.Record, messages []message) (Fixture, int, error) {
	var f Fixture
	if len(records) < 2 || records[0].Direction != "send" || records[1].Direction != "recv" {
		return f, 0, fmt.Errorf("missing sequential handshake")
	}
	first := messages[0]
	if first.Method == "server/discover" {
		if err := request(first, "server/discover"); err != nil {
			return f, 0, err
		}
		if err := response(messages[1], first.ID); err != nil {
			return f, 0, err
		}
		var params sdk.DiscoverParams
		if decode(first.Params, &params) != nil {
			return f, 0, fmt.Errorf("invalid discovery parameters")
		}
		meta := params.Meta
		for key := range meta {
			if key != sdk.MetaKeyClientInfo && key != sdk.MetaKeyClientCapabilities && key != sdk.MetaKeyProtocolVersion {
				return f, 0, fmt.Errorf("unsupported discovery metadata prerequisite: %s", key)
			}
		}
		requested, ok := meta[sdk.MetaKeyProtocolVersion].(string)
		if !ok || requested < "2026-07-28" {
			return f, 0, fmt.Errorf("invalid discovery protocol version")
		}
		raw, _ := json.Marshal(meta[sdk.MetaKeyClientInfo])
		if json.Unmarshal(raw, &f.ClientInfo) != nil || f.ClientInfo == nil {
			return f, 0, fmt.Errorf("discovery client identity missing")
		}
		f.ClientCapabilities, _ = json.Marshal(meta[sdk.MetaKeyClientCapabilities])
		var result sdk.DiscoverResult
		if json.Unmarshal(messages[1].Result, &result) != nil || result.Capabilities == nil || result.Capabilities.Tools == nil {
			return f, 0, fmt.Errorf("discovery result lacks tools capability")
		}
		supported := sdk.SupportedProtocolVersions()
		if slices.Contains(result.SupportedVersions, requested) && slices.Contains(supported, requested) {
			f.ProtocolVersion = requested
		} else {
			for _, version := range supported {
				if slices.Contains(result.SupportedVersions, version) {
					f.ProtocolVersion = version
					break
				}
			}
		}
		if f.ProtocolVersion < "2026-07-28" {
			return f, 0, fmt.Errorf("discovery has no supported modern protocol")
		}
		f.RequestedProtocolVersion = requested
		if _, err := ClientOptions(f); err != nil {
			return f, 0, err
		}
		return f, 2, nil
	}
	if err := request(first, "initialize"); err != nil {
		return f, 0, err
	}
	if err := response(messages[1], first.ID); err != nil {
		return f, 0, err
	}
	var params struct {
		ProtocolVersion string              `json:"protocolVersion"`
		Capabilities    json.RawMessage     `json:"capabilities"`
		ClientInfo      *sdk.Implementation `json:"clientInfo"`
	}
	if decode(first.Params, &params) != nil || params.ProtocolVersion == "" || params.ClientInfo == nil || len(params.Capabilities) == 0 {
		return f, 0, fmt.Errorf("invalid initialize parameters")
	}
	var result sdk.InitializeResult
	if json.Unmarshal(messages[1].Result, &result) != nil || result.ProtocolVersion == "" || result.Capabilities == nil || result.Capabilities.Tools == nil || result.ServerInfo == nil {
		return f, 0, fmt.Errorf("initialize result lacks protocol version or tools capability")
	}
	if !slices.Contains(sdk.SupportedProtocolVersions(), result.ProtocolVersion) || result.ProtocolVersion >= "2026-07-28" {
		return f, 0, fmt.Errorf("modern versions require discovery lifecycle")
	}
	if len(records) < 3 || records[2].Direction != "send" {
		return f, 0, fmt.Errorf("missing initialized notification")
	}
	m := messages[2]
	if m.Method != "notifications/initialized" || len(m.ID) != 0 || len(m.Result) != 0 || len(m.Error) != 0 {
		return f, 0, fmt.Errorf("missing initialized notification")
	}
	if len(m.Params) > 0 {
		var object map[string]json.RawMessage
		if json.Unmarshal(m.Params, &object) != nil || object == nil || len(object) != 0 {
			return f, 0, fmt.Errorf("initialized notification has unsupported parameters")
		}
	}
	f.ClientInfo = params.ClientInfo
	f.ClientCapabilities = params.Capabilities
	f.ProtocolVersion = result.ProtocolVersion
	f.RequestedProtocolVersion = params.ProtocolVersion
	if _, err := ClientOptions(f); err != nil {
		return f, 0, err
	}
	return f, 3, nil
}
func requestContext(raw []byte, f Fixture) error {
	var params map[string]json.RawMessage
	if json.Unmarshal(raw, &params) != nil || params == nil {
		return fmt.Errorf("request parameters must be an object")
	}
	var meta map[string]json.RawMessage
	if rawMeta, exists := params["_meta"]; exists {
		if json.Unmarshal(rawMeta, &meta) != nil || meta == nil {
			return fmt.Errorf("invalid request metadata")
		}
	}
	if meta["progressToken"] != nil {
		return fmt.Errorf("progress callbacks require unsupported prerequisites")
	}
	expected := map[string]any{sdk.MetaKeyProtocolVersion: f.ProtocolVersion, sdk.MetaKeyClientInfo: f.ClientInfo, sdk.MetaKeyClientCapabilities: json.RawMessage(f.ClientCapabilities)}
	if len(f.ClientCapabilities) == 0 {
		expected[sdk.MetaKeyClientCapabilities] = json.RawMessage(`{}`)
	}
	for key, value := range expected {
		captured, present := meta[key]
		if !present {
			if f.ProtocolVersion >= "2026-07-28" {
				return fmt.Errorf("missing required request context %s", key)
			}
			continue
		}
		encoded, err := json.Marshal(value)
		if err != nil || !sameJSON(captured, encoded) {
			return fmt.Errorf("conflicting request context %s", key)
		}
	}
	return nil
}
func listMethod(method string) bool {
	return method == "tools/list" || method == "prompts/list" || method == "resources/templates/list" || method == "resources/list"
}
func validatePrerequisite(p Prerequisite) error {
	if !listMethod(p.Method) || !json.Valid(p.Expected) {
		return fmt.Errorf("unsupported fixture prerequisite")
	}
	var params sdk.ListToolsParams
	if err := decode(p.Params, &params); err != nil {
		return fmt.Errorf("invalid list parameters: %w", err)
	}
	var result map[string]json.RawMessage
	if json.Unmarshal(p.Expected, &result) != nil || result == nil {
		return fmt.Errorf("invalid list result")
	}
	field := map[string]string{"tools/list": "tools", "prompts/list": "prompts", "resources/templates/list": "resourceTemplates", "resources/list": "resources"}[p.Method]
	var items []json.RawMessage
	if json.Unmarshal(result[field], &items) != nil || items == nil {
		return fmt.Errorf("list result lacks %s array", field)
	}
	return nil
}
func checkPrerequisite(ctx context.Context, session *sdk.ClientSession, p Prerequisite) error {
	var result any
	var err error
	switch p.Method {
	case "tools/list":
		var params sdk.ListToolsParams
		if err = decode(p.Params, &params); err == nil {
			result, err = session.ListTools(ctx, &params)
		}
	case "prompts/list":
		var params sdk.ListPromptsParams
		if err = decode(p.Params, &params); err == nil {
			result, err = session.ListPrompts(ctx, &params)
		}
	case "resources/templates/list":
		var params sdk.ListResourceTemplatesParams
		if err = decode(p.Params, &params); err == nil {
			result, err = session.ListResourceTemplates(ctx, &params)
		}
	case "resources/list":
		var params sdk.ListResourcesParams
		if err = decode(p.Params, &params); err == nil {
			result, err = session.ListResources(ctx, &params)
		}
	default:
		return fmt.Errorf("unsupported fixture prerequisite")
	}
	if err != nil {
		return fmt.Errorf("prerequisite %s: %w", p.Method, err)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return err
	}
	if !sameJSON(raw, p.Expected) {
		return fmt.Errorf("prerequisite %s observed result changed", p.Method)
	}
	return nil
}

func validateCursors(prerequisites []Prerequisite) error {
	next := map[string]string{}
	for _, p := range prerequisites {
		var params map[string]json.RawMessage
		json.Unmarshal(p.Params, &params)
		cursor, err := cursorValue(params["cursor"])
		if err != nil {
			return err
		}
		if cursor != "" && next[p.Method] != cursor {
			return fmt.Errorf("list cursor has incomplete or conflicting prerequisites")
		}
		var result map[string]json.RawMessage
		json.Unmarshal(p.Expected, &result)
		next[p.Method], err = cursorValue(result["nextCursor"])
		if err != nil {
			return err
		}
	}
	return nil
}
func cursorValue(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}
	var value *string
	if json.Unmarshal(raw, &value) != nil || value == nil {
		return "", fmt.Errorf("list cursor must be a string")
	}
	return *value, nil
}
