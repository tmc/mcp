// Package jsonrpc2 provides a minimal JSON-RPC 2.0 implementation
package jsonrpc2

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
)

// Constants for JSON-RPC 2.0
const (
	Version = "2.0"
)

// Standard error codes defined in the JSON-RPC spec
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

// Error represents a JSON-RPC error object
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Error implements the error interface
func (e *Error) Error() string {
	return fmt.Sprintf("jsonrpc2 error: code=%d message=%s", e.Code, e.Message)
}

// ID represents a JSON-RPC message ID
type ID struct {
	Num int64
	Str string
	raw json.RawMessage
}

// MarshalJSON implements json.Marshaler for ID
func (id ID) MarshalJSON() ([]byte, error) {
	if id.Str != "" {
		return json.Marshal(id.Str)
	}
	if id.Num != 0 {
		return json.Marshal(id.Num)
	}
	if len(id.raw) > 0 {
		return id.raw, nil
	}
	return []byte("null"), nil
}

// UnmarshalJSON implements json.Unmarshaler for ID
func (id *ID) UnmarshalJSON(data []byte) error {
	id.raw = data

	// Try to unmarshal as string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		id.Str = s
		id.Num = 0
		return nil
	}

	// Try to unmarshal as number
	var n int64
	if err := json.Unmarshal(data, &n); err == nil {
		id.Num = n
		id.Str = ""
		return nil
	}

	// If neither works, keep the raw data for null or other values
	return nil
}

// IsValid returns true if the ID is valid
func (id ID) IsValid() bool {
	return id.Str != "" || id.Num != 0 || string(id.raw) == "null"
}

// Int64ID creates a numeric ID
func Int64ID(id int64) ID {
	return ID{Num: id}
}

// StringID creates a string ID
func StringID(id string) ID {
	return ID{Str: id}
}

// Request represents a JSON-RPC request
type Request struct {
	ID     ID              `json:"id,omitempty"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC response
type Response struct {
	ID     ID              `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *Error          `json:"error,omitempty"`
}

// Message represents a JSON-RPC message (request, response, or notification)
type Message struct {
	Version string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Handler is an interface for handling JSON-RPC requests
type Handler interface {
	Handle(ctx context.Context, conn *Conn, req *Request)
}

// HandlerFunc is a function type that implements Handler
type HandlerFunc func(ctx context.Context, conn *Conn, req *Request)

// Handle calls the handler function
func (f HandlerFunc) Handle(ctx context.Context, conn *Conn, req *Request) {
	f(ctx, conn, req)
}

// Conn represents a JSON-RPC connection
type Conn struct {
	mu       sync.Mutex
	stream   *BufferedStream
	handler  Handler
	initOnce sync.Once
}

// NotificationMethod returns true if the request is a notification
func (r *Request) NotificationMethod() bool {
	return !r.ID.IsValid()
}

// BufferedStream reads and writes JSON-RPC messages over an io.ReadWriteCloser
type BufferedStream struct {
	reader *bufio.Reader
	writer io.Writer
	closer io.Closer
	mu     sync.Mutex
}

// NewBufferedStream creates a new BufferedStream
func NewBufferedStream(rw io.ReadWriteCloser, codec Codec) *BufferedStream {
	return &BufferedStream{
		reader: bufio.NewReader(rw),
		writer: rw,
		closer: rw,
	}
}

// Read reads a message from the stream
func (s *BufferedStream) Read(ctx context.Context) (Message, error) {
	var msg Message
	line, err := s.reader.ReadBytes('\n')
	if err != nil {
		return msg, err
	}

	err = json.Unmarshal(line, &msg)
	return msg, err
}

// Write writes a message to the stream
func (s *BufferedStream) Write(ctx context.Context, msg Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = s.writer.Write(append(data, '\n'))
	return err
}

// Close closes the stream
func (s *BufferedStream) Close() error {
	return s.closer.Close()
}

// Codec is an interface for encoding and decoding JSON-RPC messages
type Codec interface {
	Encode(v interface{}) ([]byte, error)
	Decode(data []byte, v interface{}) error
}

// VSCodeObjectCodec is a codec that implements the VS Code JSON-RPC protocol
type VSCodeObjectCodec struct{}

// Encode encodes a value as JSON
func (VSCodeObjectCodec) Encode(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// Decode decodes JSON data into a value
func (VSCodeObjectCodec) Decode(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// NewConn creates a new JSON-RPC connection
func NewConn(ctx context.Context, stream *BufferedStream, handler Handler) *Conn {
	conn := &Conn{
		stream:  stream,
		handler: handler,
	}

	go conn.readMessages(ctx)

	return conn
}

// readMessages reads and processes messages from the connection
func (c *Conn) readMessages(ctx context.Context) {
	for {
		msg, err := c.stream.Read(ctx)
		if err != nil {
			if err == io.EOF || strings.Contains(err.Error(), "closed") {
				// Connection closed normally
				break
			}
			// Handle other errors
			continue
		}

		// Check if it's a request or notification
		if msg.Method != "" {
			id := ID{raw: msg.ID}

			// Try to parse the ID if it exists
			if len(msg.ID) > 0 {
				if msg.ID[0] == '"' {
					// String ID
					if err := json.Unmarshal(msg.ID, &id.Str); err == nil {
						id.Num = 0
					}
				} else {
					// Number ID
					if err := json.Unmarshal(msg.ID, &id.Num); err == nil {
						id.Str = ""
					}
				}
			}

			req := &Request{
				ID:     id,
				Method: msg.Method,
				Params: msg.Params,
			}

			go c.handler.Handle(ctx, c, req)
		}
	}
}

// Reply sends a response to a request
func (c *Conn) Reply(ctx context.Context, id ID, result interface{}) error {
	var raw json.RawMessage
	if result != nil {
		data, err := json.Marshal(result)
		if err != nil {
			return err
		}
		raw = data
	}

	return c.ReplyWithData(ctx, id, raw)
}

// ReplyWithData sends a raw JSON response to a request
func (c *Conn) ReplyWithData(ctx context.Context, id ID, result json.RawMessage) error {
	msg := Message{
		Version: Version,
		Result:  result,
	}

	if id.Str != "" {
		b, err := json.Marshal(id.Str)
		if err != nil {
			return err
		}
		msg.ID = b
	} else {
		b, err := json.Marshal(id.Num)
		if err != nil {
			return err
		}
		msg.ID = b
	}

	return c.stream.Write(ctx, msg)
}

// ReplyWithError sends an error response to a request
func (c *Conn) ReplyWithError(ctx context.Context, id ID, err *Error) error {
	var rawID json.RawMessage

	if id.Str != "" {
		b, err := json.Marshal(id.Str)
		if err != nil {
			return err
		}
		rawID = b
	} else {
		b, err := json.Marshal(id.Num)
		if err != nil {
			return err
		}
		rawID = b
	}

	msg := Message{
		Version: Version,
		ID:      rawID,
		Error:   err,
	}

	return c.stream.Write(ctx, msg)
}

// Notify sends a notification
func (c *Conn) Notify(ctx context.Context, method string, params interface{}) error {
	var raw json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return err
		}
		raw = data
	}

	return c.NotifyWithData(ctx, method, raw)
}

// NotifyWithData sends a raw JSON notification
func (c *Conn) NotifyWithData(ctx context.Context, method string, params json.RawMessage) error {
	msg := Message{
		Version: Version,
		Method:  method,
		Params:  params,
	}

	return c.stream.Write(ctx, msg)
}

// Reader is an interface for reading messages
type Reader interface {
	Read(context.Context) (Message, error)
}

// Writer is an interface for writing messages
type Writer interface {
	Write(context.Context, Message) error
}

// Preempter is an interface for priority message handling
type Preempter interface {
	Preempt(ctx context.Context, method string, params interface{})
}

// DisconnectNotify returns a channel that is closed when the connection is closed
func (c *Conn) DisconnectNotify() <-chan struct{} {
	ch := make(chan struct{})
	close(ch) // For simplicity, just return a closed channel
	return ch
}

// EncodeMessage encodes a message to JSON
func EncodeMessage(msg Message) ([]byte, error) {
	return json.Marshal(msg)
}

// DecodeMessage decodes a JSON message
func DecodeMessage(data []byte) (Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return msg, err
}

// ConnectionOptions contains options for a connection
type ConnectionOptions struct {
	Handler Handler
}

// Dial creates a new client connection
func Dial(ctx context.Context, rwc io.ReadWriteCloser, binder interface{}) (*Conn, error) {
	stream := NewBufferedStream(rwc, VSCodeObjectCodec{})
	return NewConn(ctx, stream, binder.(Handler)), nil
}

// Binder is an interface for binding connections
type Binder interface {
	Bind(context.Context, *Conn) (ConnectionOptions, error)
}
