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

// IDGeneratingFramer implements jsonrpc2.Framer with automatic ID generation for requests.
type IDGeneratingFramer struct {
	DebugLogger *slog.Logger
	idCounter   atomic.Int64
}

type idGeneratingReader struct {
	r           *bufio.Reader
	debugLogger *slog.Logger
	framer      *IDGeneratingFramer
}

type idGeneratingWriter struct {
	w           *bufio.Writer
	debugLogger *slog.Logger
	framer      *IDGeneratingFramer
}

func (f *IDGeneratingFramer) Reader(r io.Reader) jsonrpc2.Reader {
	return &idGeneratingReader{r: bufio.NewReader(r), debugLogger: f.DebugLogger, framer: f}
}

func (f *IDGeneratingFramer) Writer(w io.Writer) jsonrpc2.Writer {
	return &idGeneratingWriter{w: bufio.NewWriter(w), debugLogger: f.DebugLogger, framer: f}
}

func (r *idGeneratingReader) Read(ctx context.Context) (jsonrpc2.Message, int64, error) {
	// Same as LineFramer.Read
	select {
	case <-ctx.Done():
		return nil, 0, ctx.Err()
	default:
	}

	line, err := r.r.ReadBytes('\n')
	if err != nil {
		if r.debugLogger != nil && err != io.EOF {
			r.debugLogger.DebugContext(ctx, "IDGeneratingFramer.Read: error reading line", "error", err)
		}
		return nil, 0, err
	}
	trimmedLine := bytes.TrimSpace(line)
	if len(trimmedLine) == 0 {
		return r.Read(ctx)
	}

	if r.debugLogger != nil {
		r.debugLogger.DebugContext(ctx, "IDGeneratingFramer.Read: raw", "line", string(trimmedLine))
	}

	var probe struct {
		Method *string `json:"method"`
	}
	if err := json.Unmarshal(trimmedLine, &probe); err != nil {
		if r.debugLogger != nil {
			r.debugLogger.ErrorContext(ctx, "IDGeneratingFramer.Read: error probing message type", "error", err, "line", string(trimmedLine))
		}
		return nil, 0, fmt.Errorf("%w: %s on line: %s", jsonrpc2.ErrParse, err.Error(), string(trimmedLine))
	}

	var msg jsonrpc2.Message
	if probe.Method != nil {
		var req jsonrpc2.Request
		if err := json.Unmarshal(trimmedLine, &req); err != nil {
			if r.debugLogger != nil {
				r.debugLogger.ErrorContext(ctx, "IDGeneratingFramer.Read: error unmarshaling request", "error", err, "line", string(trimmedLine))
			}
			return nil, 0, fmt.Errorf("%w: %s for line: %s", jsonrpc2.ErrInvalidRequest, err.Error(), string(trimmedLine))
		}
		msg = &req
		if r.debugLogger != nil {
			if req.IsCall() {
				r.debugLogger.DebugContext(ctx, "IDGeneratingFramer.Read: parsed request call", "method", req.Method, "id", req.ID)
			} else {
				r.debugLogger.DebugContext(ctx, "IDGeneratingFramer.Read: parsed request notification", "method", req.Method)
			}
		}
	} else {
		var resp jsonrpc2.Response
		if err := json.Unmarshal(trimmedLine, &resp); err != nil {
			if r.debugLogger != nil {
				r.debugLogger.ErrorContext(ctx, "IDGeneratingFramer.Read: error unmarshaling response", "error", err, "line", string(trimmedLine))
			}
			return nil, 0, fmt.Errorf("%w: %s for line: %s", jsonrpc2.ErrParse, err.Error(), string(trimmedLine))
		}
		msg = &resp
		if r.debugLogger != nil {
			r.debugLogger.DebugContext(ctx, "IDGeneratingFramer.Read: parsed response", "id", resp.ID, "has_result", resp.Result != nil, "has_error", resp.Error != nil)
		}
	}
	return msg, int64(len(line)), nil
}

func (w *idGeneratingWriter) Write(ctx context.Context, msg jsonrpc2.Message) (int64, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	if w.debugLogger != nil {
		w.debugLogger.DebugContext(ctx, "IDGeneratingFramer.Write: message type", "type", fmt.Sprintf("%T", msg))
	}

	// Check if this is a request
	if req, ok := msg.(*jsonrpc2.Request); ok {
		if w.debugLogger != nil {
			w.debugLogger.DebugContext(ctx, "IDGeneratingFramer.Write: request details", "method", req.Method, "id_valid", req.ID.IsValid())
		}
		// Check if the ID is invalid or empty
		if !req.ID.IsValid() {
			// Generate a new ID
			newID := w.framer.idCounter.Add(1)
			req.ID = jsonrpc2.Int64ID(newID)
			if w.debugLogger != nil {
				w.debugLogger.DebugContext(ctx, "IDGeneratingFramer.Write: generated ID for request", "id", newID, "method", req.Method)
			}
		}
	}

	data, err := json.Marshal(msg)
	if err != nil {
		if w.debugLogger != nil {
			w.debugLogger.ErrorContext(ctx, "IDGeneratingFramer.Write: error marshaling message", "error", err, "type", fmt.Sprintf("%T", msg))
		}
		return 0, err
	}

	if w.debugLogger != nil {
		w.debugLogger.DebugContext(ctx, "IDGeneratingFramer.Write: raw", "data", string(data))
	}

	nTotal := 0
	n, err := w.w.Write(data)
	nTotal += n
	if err != nil {
		if w.debugLogger != nil {
			w.debugLogger.ErrorContext(ctx, "IDGeneratingFramer.Write: error writing message data", "error", err)
		}
		return int64(nTotal), err
	}

	n, err = w.w.Write([]byte{'\n'})
	nTotal += n

	flushErr := w.w.Flush()
	if err == nil {
		err = flushErr
	}
	if flushErr != nil && w.debugLogger != nil {
		w.debugLogger.ErrorContext(ctx, "IDGeneratingFramer.Write: error flushing", "error", flushErr)
	}

	return int64(nTotal), err
}
