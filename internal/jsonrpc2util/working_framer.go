package jsonrpc2util

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync/atomic"

	"golang.org/x/exp/jsonrpc2"
)

// WorkingFramer implements a framer that works around jsonrpc2.ID marshaling issues
type WorkingFramer struct {
	DebugLogger *slog.Logger
	idCounter   atomic.Int64
}

type workingReader struct {
	r           *bufio.Reader
	debugLogger *slog.Logger
}

type workingWriter struct {
	w           *bufio.Writer
	debugLogger *slog.Logger
	idCounter   *atomic.Int64
}

// wireRequest represents the JSON structure we send on the wire
type wireRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// wireResponse represents the JSON structure we receive on the wire
type wireResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
}

func (f *WorkingFramer) Reader(r io.Reader) jsonrpc2.Reader {
	return &workingReader{r: bufio.NewReader(r), debugLogger: f.DebugLogger}
}

func (f *WorkingFramer) Writer(w io.Writer) jsonrpc2.Writer {
	return &workingWriter{w: bufio.NewWriter(w), debugLogger: f.DebugLogger, idCounter: &f.idCounter}
}

func (r *workingReader) Read(ctx context.Context) (jsonrpc2.Message, int64, error) {
	select {
	case <-ctx.Done():
		return nil, 0, ctx.Err()
	default:
	}

	line, err := r.r.ReadBytes('\n')
	if err != nil {
		return nil, 0, err
	}

	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return r.Read(ctx)
	}

	if r.debugLogger != nil {
		r.debugLogger.DebugContext(ctx, "WorkingFramer.Read: raw", "line", string(line))
	}

	// First check if it's a request or response
	var probe struct {
		Method *string `json:"method"`
	}
	if err := json.Unmarshal(line, &probe); err != nil {
		return nil, 0, err
	}

	if probe.Method != nil {
		// It's a request - unmarshal to our wire format first
		var wire wireRequest
		if err := json.Unmarshal(line, &wire); err != nil {
			return nil, 0, err
		}

		// Convert to jsonrpc2.Request
		req := jsonrpc2.Request{
			Method: wire.Method,
			Params: wire.Params,
		}

		// Handle ID conversion
		if wire.ID != nil {
			switch v := wire.ID.(type) {
			case float64:
				req.ID = jsonrpc2.Int64ID(int64(v))
			case string:
				req.ID = jsonrpc2.StringID(v)
			}
		}

		return &req, int64(len(line)), nil
	} else {
		// It's a response - unmarshal to our wire format first
		var wire wireResponse
		if err := json.Unmarshal(line, &wire); err != nil {
			return nil, 0, err
		}

		// Convert to jsonrpc2.Response
		resp := jsonrpc2.Response{
			Result: wire.Result,
		}

		// Handle ID conversion
		if wire.ID != nil {
			switch v := wire.ID.(type) {
			case float64:
				resp.ID = jsonrpc2.Int64ID(int64(v))
			case string:
				resp.ID = jsonrpc2.StringID(v)
			}
		}

		// Handle error
		if wire.Error != nil {
			resp.Error = fmt.Errorf("remote error: %s", string(wire.Error))
		}

		return &resp, int64(len(line)), nil
	}
}

func (w *workingWriter) Write(ctx context.Context, msg jsonrpc2.Message) (int64, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	var wireMsg interface{}

	switch m := msg.(type) {
	case *jsonrpc2.Request:
		wire := wireRequest{
			JSONRPC: "2.0",
			Method:  m.Method,
			Params:  m.Params,
		}

		// Generate ID if needed
		if !m.ID.IsValid() {
			id := w.idCounter.Add(1)
			wire.ID = id
			if w.debugLogger != nil {
				w.debugLogger.DebugContext(ctx, "WorkingFramer.Write: generated ID", "id", id, "method", m.Method)
			}
		} else {
			// Extract the actual ID value
			raw := m.ID.Raw()
			if raw != nil {
				wire.ID = raw
			}
		}

		wireMsg = wire

	case *jsonrpc2.Response:
		wire := wireResponse{
			JSONRPC: "2.0",
			Result:  m.Result,
		}

		// Extract the actual ID value
		if m.ID.IsValid() {
			raw := m.ID.Raw()
			if raw != nil {
				wire.ID = raw
			}
		}

		if m.Error != nil {
			// Convert error to JSON
			errMsg := map[string]interface{}{
				"message": m.Error.Error(),
			}
			wire.Error, _ = json.Marshal(errMsg)
		}

		wireMsg = wire

	default:
		return 0, fmt.Errorf("unsupported message type: %T", msg)
	}

	data, err := json.Marshal(wireMsg)
	if err != nil {
		return 0, err
	}

	if w.debugLogger != nil {
		w.debugLogger.DebugContext(ctx, "WorkingFramer.Write: raw", "data", string(data))
	}

	n, err := w.w.Write(data)
	if err != nil {
		return int64(n), err
	}

	n2, err := w.w.Write([]byte{'\n'})
	n += n2

	if flushErr := w.w.Flush(); flushErr != nil && err == nil {
		err = flushErr
	}

	return int64(n), err
}
