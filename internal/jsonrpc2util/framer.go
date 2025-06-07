package jsonrpc2util

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	"golang.org/x/exp/jsonrpc2"
)

// LineFramer implements jsonrpc2.Framer for newline-delimited JSON messages.
type LineFramer struct {
	DebugLogger *slog.Logger // Optional logger for debugging raw messages
}

type lineFramerReader struct {
	r           *bufio.Reader
	debugLogger *slog.Logger
}

type lineFramerWriter struct {
	w           *bufio.Writer // Use bufio.Writer for efficient flushing
	debugLogger *slog.Logger
}

func (f *LineFramer) Reader(r io.Reader) jsonrpc2.Reader {
	return &lineFramerReader{r: bufio.NewReader(r), debugLogger: f.DebugLogger}
}

func (f *LineFramer) Writer(w io.Writer) jsonrpc2.Writer {
	return &lineFramerWriter{w: bufio.NewWriter(w), debugLogger: f.DebugLogger}
}

func (r *lineFramerReader) Read(ctx context.Context) (jsonrpc2.Message, int64, error) {
	select {
	case <-ctx.Done():
		return nil, 0, ctx.Err()
	default:
	}

	line, err := r.r.ReadBytes('\n')
	if err != nil {
		if r.debugLogger != nil && err != io.EOF {
			r.debugLogger.DebugContext(ctx, "LineFramer.Read: error reading line", "error", err)
		}
		return nil, 0, err
	}
	if r.debugLogger != nil {
		r.debugLogger.DebugContext(ctx, "LineFramer.Read: raw message", "raw", string(line))
	}

	// Parse the JSON message
	msg, err := jsonrpc2.DecodeMessage(bytes.TrimSpace(line))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to decode message: %v", err)
	}

	return msg, int64(len(line)), nil
}

func (w *lineFramerWriter) Write(ctx context.Context, msg jsonrpc2.Message) (int64, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return 0, err
	}

	n, err := w.w.Write(data)
	if err != nil {
		return int64(n), err
	}

	n2, err := w.w.Write([]byte("\n"))
	if err != nil {
		return int64(n + n2), err
	}

	// Flush the buffer to ensure the message is sent immediately
	if err := w.w.Flush(); err != nil {
		return int64(n + n2), err
	}

	if w.debugLogger != nil {
		w.debugLogger.DebugContext(ctx, "LineFramer.Write: raw message", "raw", string(data))
	}

	return int64(n + n2), nil
}
