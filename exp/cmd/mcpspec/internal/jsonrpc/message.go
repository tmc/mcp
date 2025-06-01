// Package jsonrpc provides utilities for working with JSON-RPC 2.0 messages in the MCP protocol.
package jsonrpc

import (
	"encoding/json"
	"fmt"
)

// Message represents a JSON-RPC message for MCP.
type Message struct {
	// Version of the JSON-RPC protocol. Always "2.0".
	Version string `json:"jsonrpc"`

	// ID of the message. Null for notifications.
	ID interface{} `json:"id,omitempty"`

	// Method is the name of the method to be invoked.
	Method string `json:"method,omitempty"`

	// Params are the parameters to the method.
	Params json.RawMessage `json:"params,omitempty"`

	// Result is the result of the method call (for responses).
	Result json.RawMessage `json:"result,omitempty"`

	// Error is the error from the method call (for responses).
	Error *Error `json:"error,omitempty"`
}

// Error represents a JSON-RPC error object.
type Error struct {
	// Code is the error code.
	Code int `json:"code"`

	// Message is the error message.
	Message string `json:"message"`

	// Data is optional additional information about the error.
	Data interface{} `json:"data,omitempty"`
}

// Standard error codes defined in the JSON-RPC spec.
const (
	// Standard JSON-RPC 2.0 error codes
	ErrCodeParseError     = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternalError  = -32603

	// MCP specific error codes can be defined in the range -32000 to -32099
	ErrCodeMCPBase = -32000
)

// Protocol constants
const (
	JSONRPCVersion = "2.0"
)

// Standard MCP method names
const (
	MethodInitialize             = "initialize"
	MethodPing                   = "ping"
	MethodResourcesList          = "resources/list"
	MethodResourcesTemplatesList = "resources/templates/list"
	MethodResourcesRead          = "resources/read"
	MethodPromptsList            = "prompts/list"
	MethodPromptsGet             = "prompts/get"
	MethodToolsList              = "tools/list"
	MethodToolsCall              = "tools/call"
)

// NewRequest creates a new JSON-RPC request message with the given method, params, and ID.
func NewRequest(method string, params interface{}, id int) (*Message, error) {
	var paramsJSON json.RawMessage
	if params != nil {
		var err error
		paramsJSON, err = json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal parameters: %w", err)
		}
	}

	return &Message{
		Version: JSONRPCVersion,
		ID:      id,
		Method:  method,
		Params:  paramsJSON,
	}, nil
}

// NewNotification creates a new JSON-RPC notification message with the given method and params.
func NewNotification(method string, params interface{}) (*Message, error) {
	var paramsJSON json.RawMessage
	if params != nil {
		var err error
		paramsJSON, err = json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal parameters: %w", err)
		}
	}

	return &Message{
		Version: JSONRPCVersion,
		Method:  method,
		Params:  paramsJSON,
	}, nil
}

// NewResponse creates a new JSON-RPC response message with the given result and ID.
func NewResponse(result interface{}, id interface{}) (*Message, error) {
	var resultJSON json.RawMessage
	if result != nil {
		var err error
		resultJSON, err = json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result: %w", err)
		}
	}

	return &Message{
		Version: JSONRPCVersion,
		ID:      id,
		Result:  resultJSON,
	}, nil
}

// NewErrorResponse creates a new JSON-RPC error response message with the given error and ID.
func NewErrorResponse(message string, code int, id interface{}) *Message {
	return &Message{
		Version: JSONRPCVersion,
		ID:      id,
		Error: &Error{
			Code:    code,
			Message: message,
		},
	}
}

// IsRequest returns true if the message is a request.
func (m *Message) IsRequest() bool {
	return m.Method != "" && m.ID != nil
}

// IsNotification returns true if the message is a notification.
func (m *Message) IsNotification() bool {
	return m.Method != "" && m.ID == nil
}

// IsResponse returns true if the message is a response.
func (m *Message) IsResponse() bool {
	return m.ID != nil && (m.Result != nil || m.Error != nil)
}

// IsError returns true if the message is an error response.
func (m *Message) IsError() bool {
	return m.ID != nil && m.Error != nil
}

// ParseParams parses the params of the message into the given value.
func (m *Message) ParseParams(v interface{}) error {
	if m.Params == nil {
		return nil
	}
	return json.Unmarshal(m.Params, v)
}

// ParseResult parses the result of the message into the given value.
func (m *Message) ParseResult(v interface{}) error {
	if m.Result == nil {
		return fmt.Errorf("no result to parse")
	}
	return json.Unmarshal(m.Result, v)
}
