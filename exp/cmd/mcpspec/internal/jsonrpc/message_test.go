package jsonrpc

import (
	"encoding/json"
	"testing"
)

func TestNewRequest(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		params   interface{}
		id       int
		wantJSON string
		wantErr  bool
	}{
		{
			name:     "basic request",
			method:   "test",
			params:   nil,
			id:       1,
			wantJSON: `{"jsonrpc":"2.0","id":1,"method":"test"}`,
			wantErr:  false,
		},
		{
			name:     "request with params",
			method:   "test",
			params:   map[string]string{"foo": "bar"},
			id:       2,
			wantJSON: `{"jsonrpc":"2.0","id":2,"method":"test","params":{"foo":"bar"}}`,
			wantErr:  false,
		},
		{
			name:     "tools/list request",
			method:   MethodToolsList,
			params:   map[string]string{"cursor": ""},
			id:       3,
			wantJSON: `{"jsonrpc":"2.0","id":3,"method":"tools/list","params":{"cursor":""}}`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := NewRequest(tt.method, tt.params, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			bytes, err := json.Marshal(msg)
			if err != nil {
				t.Errorf("Failed to marshal message: %v", err)
				return
			}

			if string(bytes) != tt.wantJSON {
				t.Errorf("NewRequest() JSON = %v, want %v", string(bytes), tt.wantJSON)
			}
		})
	}
}

func TestNewNotification(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		params   interface{}
		wantJSON string
		wantErr  bool
	}{
		{
			name:     "basic notification",
			method:   "test",
			params:   nil,
			wantJSON: `{"jsonrpc":"2.0","method":"test"}`,
			wantErr:  false,
		},
		{
			name:     "notification with params",
			method:   "test",
			params:   map[string]string{"foo": "bar"},
			wantJSON: `{"jsonrpc":"2.0","method":"test","params":{"foo":"bar"}}`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := NewNotification(tt.method, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewNotification() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			bytes, err := json.Marshal(msg)
			if err != nil {
				t.Errorf("Failed to marshal message: %v", err)
				return
			}

			if string(bytes) != tt.wantJSON {
				t.Errorf("NewNotification() JSON = %v, want %v", string(bytes), tt.wantJSON)
			}
		})
	}
}

func TestNewResponse(t *testing.T) {
	tests := []struct {
		name     string
		result   interface{}
		id       interface{}
		wantJSON string
		wantErr  bool
	}{
		{
			name:     "basic response",
			result:   map[string]string{"result": "value"},
			id:       1,
			wantJSON: `{"jsonrpc":"2.0","id":1,"result":{"result":"value"}}`,
			wantErr:  false,
		},
		{
			name:     "tools list response",
			result:   map[string]interface{}{"tools": []map[string]string{{"name": "test"}}},
			id:       2,
			wantJSON: `{"jsonrpc":"2.0","id":2,"result":{"tools":[{"name":"test"}]}}`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := NewResponse(tt.result, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			bytes, err := json.Marshal(msg)
			if err != nil {
				t.Errorf("Failed to marshal message: %v", err)
				return
			}

			if string(bytes) != tt.wantJSON {
				t.Errorf("NewResponse() JSON = %v, want %v", string(bytes), tt.wantJSON)
			}
		})
	}
}

func TestNewErrorResponse(t *testing.T) {
	tests := []struct {
		name    string
		message string
		code    int
		id      interface{}
		wantErr string
	}{
		{
			name:    "basic error",
			message: "test error",
			code:    -32000,
			id:      1,
			wantErr: "test error",
		},
		{
			name:    "invalid params error",
			message: "invalid params",
			code:    -32602,
			id:      2,
			wantErr: "invalid params",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := NewErrorResponse(tt.message, tt.code, tt.id)

			if msg.Error == nil {
				t.Errorf("NewErrorResponse() did not set Error field")
				return
			}

			if msg.Error.Message != tt.wantErr {
				t.Errorf("NewErrorResponse() Error.Message = %v, want %v", msg.Error.Message, tt.wantErr)
			}

			if msg.Error.Code != tt.code {
				t.Errorf("NewErrorResponse() Error.Code = %v, want %v", msg.Error.Code, tt.code)
			}

			if msg.ID != tt.id {
				t.Errorf("NewErrorResponse() ID = %v, want %v", msg.ID, tt.id)
			}
		})
	}
}

func TestMessageType(t *testing.T) {
	tests := []struct {
		name           string
		message        Message
		isRequest      bool
		isNotification bool
		isResponse     bool
		isError        bool
	}{
		{
			name: "request",
			message: Message{
				Version: "2.0",
				ID:      1,
				Method:  "test",
				Params:  json.RawMessage(`{}`),
			},
			isRequest:      true,
			isNotification: false,
			isResponse:     false,
			isError:        false,
		},
		{
			name: "notification",
			message: Message{
				Version: "2.0",
				Method:  "test",
				Params:  json.RawMessage(`{}`),
			},
			isRequest:      false,
			isNotification: true,
			isResponse:     false,
			isError:        false,
		},
		{
			name: "response",
			message: Message{
				Version: "2.0",
				ID:      1,
				Result:  json.RawMessage(`{}`),
			},
			isRequest:      false,
			isNotification: false,
			isResponse:     true,
			isError:        false,
		},
		{
			name: "error",
			message: Message{
				Version: "2.0",
				ID:      1,
				Error:   &Error{Code: -32000, Message: "test error"},
			},
			isRequest:      false,
			isNotification: false,
			isResponse:     true,
			isError:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.message.IsRequest(); got != tt.isRequest {
				t.Errorf("Message.IsRequest() = %v, want %v", got, tt.isRequest)
			}
			if got := tt.message.IsNotification(); got != tt.isNotification {
				t.Errorf("Message.IsNotification() = %v, want %v", got, tt.isNotification)
			}
			if got := tt.message.IsResponse(); got != tt.isResponse {
				t.Errorf("Message.IsResponse() = %v, want %v", got, tt.isResponse)
			}
			if got := tt.message.IsError(); got != tt.isError {
				t.Errorf("Message.IsError() = %v, want %v", got, tt.isError)
			}
		})
	}
}

func TestParseParams(t *testing.T) {
	type ListToolsRequest struct {
		Cursor string `json:"cursor,omitempty"`
	}

	type CallToolRequest struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments,omitempty"`
	}

	tests := []struct {
		name    string
		params  json.RawMessage
		target  interface{}
		wantErr bool
		check   func(interface{}) bool
	}{
		{
			name:    "parse tools/list params",
			params:  json.RawMessage(`{"cursor":"test"}`),
			target:  &ListToolsRequest{},
			wantErr: false,
			check: func(v interface{}) bool {
				req, ok := v.(*ListToolsRequest)
				return ok && req.Cursor == "test"
			},
		},
		{
			name:    "parse tools/call params",
			params:  json.RawMessage(`{"name":"test","arguments":{"foo":"bar"}}`),
			target:  &CallToolRequest{},
			wantErr: false,
			check: func(v interface{}) bool {
				req, ok := v.(*CallToolRequest)
				return ok && req.Name == "test"
			},
		},
		{
			name:    "nil params",
			params:  nil,
			target:  &struct{}{},
			wantErr: false,
			check:   func(v interface{}) bool { return true },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &Message{Params: tt.params}
			err := msg.ParseParams(tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("Message.ParseParams() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && !tt.check(tt.target) {
				t.Errorf("Message.ParseParams() didn't properly parse into target object")
			}
		})
	}
}

func TestParseResult(t *testing.T) {
	type ListToolsResult struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}

	tests := []struct {
		name    string
		result  json.RawMessage
		target  interface{}
		wantErr bool
		check   func(interface{}) bool
	}{
		{
			name:    "parse tools/list result",
			result:  json.RawMessage(`{"tools":[{"name":"test"}]}`),
			target:  &ListToolsResult{},
			wantErr: false,
			check: func(v interface{}) bool {
				res, ok := v.(*ListToolsResult)
				return ok && len(res.Tools) == 1 && res.Tools[0].Name == "test"
			},
		},
		{
			name:    "parse empty result",
			result:  json.RawMessage(`{}`),
			target:  &struct{}{},
			wantErr: false,
			check:   func(v interface{}) bool { return true },
		},
		{
			name:    "nil result",
			result:  nil,
			target:  &struct{}{},
			wantErr: true,
			check:   func(v interface{}) bool { return false },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &Message{Result: tt.result}
			err := msg.ParseResult(tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("Message.ParseResult() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && !tt.check(tt.target) {
				t.Errorf("Message.ParseResult() didn't properly parse into target object")
			}
		})
	}
}
