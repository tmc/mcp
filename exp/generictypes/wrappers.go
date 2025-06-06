package generictypes

import (
	"encoding/json"

	"github.com/tmc/mcp/modelcontextprotocol"
)

// Request is a generic wrapper for MCP requests with metadata.
// It simplifies the creation of request types that all follow the same pattern.
type Request[T any] struct {
	Meta   *modelcontextprotocol.RequestMeta `json:"_meta,omitempty"`
	Params T                                 `json:",inline"`
}

// WithMeta returns a new Request with the specified metadata.
func (r Request[T]) WithMeta(meta modelcontextprotocol.RequestMeta) Request[T] {
	r.Meta = &meta
	return r
}

// WithProgressToken returns a new Request with a progress token.
func (r Request[T]) WithProgressToken(token modelcontextprotocol.ProgressToken) Request[T] {
	if r.Meta == nil {
		r.Meta = &modelcontextprotocol.RequestMeta{}
	}
	r.Meta.ProgressToken = &token
	return r
}

// Result is a generic wrapper for MCP results with metadata.
// It provides a consistent structure for all result types.
type Result[T any] struct {
	Meta map[string]any `json:"_meta,omitempty"`
	Data T              `json:",inline"`
}

// WithMeta returns a new Result with the specified metadata.
func (r Result[T]) WithMeta(key string, value any) Result[T] {
	if r.Meta == nil {
		r.Meta = make(map[string]any)
	}
	r.Meta[key] = value
	return r
}

// Notification is a generic wrapper for notifications.
type Notification[T any] struct {
	Meta map[string]any `json:"_meta,omitempty"`
	Data T              `json:",inline"`
}

// --- JSON marshaling helpers ---

// MarshalAs marshals the data as the wrapped type T.
func (r Request[T]) MarshalJSON() ([]byte, error) {
	type wrapper struct {
		Meta *modelcontextprotocol.RequestMeta `json:"_meta,omitempty"`
		Data T                                 `json:"-"`
	}
	w := wrapper{Meta: r.Meta, Data: r.Params}

	// Marshal the wrapper first
	wrapperBytes, err := json.Marshal(w)
	if err != nil {
		return nil, err
	}

	// Marshal the params
	paramBytes, err := json.Marshal(r.Params)
	if err != nil {
		return nil, err
	}

	// Merge the two JSON objects
	if len(paramBytes) > 0 && paramBytes[0] == '{' && len(wrapperBytes) > 0 && wrapperBytes[0] == '{' {
		// Remove the closing brace from wrapper and opening brace from params
		merged := append(wrapperBytes[:len(wrapperBytes)-1], ',')
		merged = append(merged, paramBytes[1:]...)
		return merged, nil
	}

	// Can't use embedded type parameter, so we'll return the marshaled params directly
	return json.Marshal(r.Params)
}

// UnmarshalJSON implements custom unmarshaling for Request types.
func (r *Request[T]) UnmarshalJSON(data []byte) error {
	type wrapper struct {
		Meta *modelcontextprotocol.RequestMeta `json:"_meta,omitempty"`
	}

	var w wrapper
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	r.Meta = w.Meta

	// Unmarshal directly into Params
	return json.Unmarshal(data, &r.Params)
}

// Similar marshaling for Result type
func (r Result[T]) MarshalJSON() ([]byte, error) {
	type wrapper struct {
		Meta map[string]any `json:"_meta,omitempty"`
		Data T              `json:"-"`
	}
	w := wrapper{Meta: r.Meta, Data: r.Data}

	// Marshal the wrapper first
	wrapperBytes, err := json.Marshal(w)
	if err != nil {
		return nil, err
	}

	// Marshal the data
	dataBytes, err := json.Marshal(r.Data)
	if err != nil {
		return nil, err
	}

	// Merge the two JSON objects
	if len(dataBytes) > 0 && dataBytes[0] == '{' && len(wrapperBytes) > 0 && wrapperBytes[0] == '{' {
		// Remove the closing brace from wrapper and opening brace from data
		merged := append(wrapperBytes[:len(wrapperBytes)-1], ',')
		merged = append(merged, dataBytes[1:]...)
		return merged, nil
	}

	// Can't use embedded type parameter, so we'll return the marshaled data directly
	return json.Marshal(r.Data)
}

// --- Example type aliases showing how these could be used ---

// InitializeRequest replaces InitializeRequestParams
type InitializeRequest = Request[struct {
	ProtocolVersion string                                  `json:"protocolVersion"`
	Capabilities    modelcontextprotocol.ClientCapabilities `json:"capabilities"`
	ClientInfo      modelcontextprotocol.Implementation     `json:"clientInfo"`
}]

// InitializeResponse replaces InitializeResult
type InitializeResponse = Result[struct {
	ProtocolVersion string                                  `json:"protocolVersion"`
	Capabilities    modelcontextprotocol.ServerCapabilities `json:"capabilities"`
	ServerInfo      modelcontextprotocol.Implementation     `json:"serverInfo"`
	Instructions    *string                                 `json:"instructions,omitempty"`
}]

// ProgressNotification replaces ProgressNotificationParams
type ProgressNotification = Notification[struct {
	ProgressToken modelcontextprotocol.ProgressToken `json:"progressToken"`
	Progress      float64                            `json:"progress"`
	Total         *float64                           `json:"total,omitempty"`
	Message       *string                            `json:"message,omitempty"`
}]
