package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strings"

	"golang.org/x/exp/jsonrpc2"
	errors "golang.org/x/xerrors"
)

// LineFramer returns a JSON-RPC framer that reads and writes one message per line.
//
// This matches the stdio transport used by the current Node MCP SDK.
func LineFramer() jsonrpc2.Framer { return lineFramer{} }

type lineFramer struct{}

type lineReader struct {
	in *bufio.Reader
}

type lineWriter struct {
	out io.Writer
}

func (lineFramer) Reader(rw io.Reader) jsonrpc2.Reader {
	return &lineReader{in: bufio.NewReader(rw)}
}

func (lineFramer) Writer(rw io.Writer) jsonrpc2.Writer {
	return &lineWriter{out: rw}
}

func (r *lineReader) Read(ctx context.Context) (jsonrpc2.Message, int64, error) {
	select {
	case <-ctx.Done():
		return nil, 0, ctx.Err()
	default:
	}
	line, err := r.in.ReadString('\n')
	if err != nil {
		return nil, int64(len(line)), err
	}
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	if line == "" {
		return nil, int64(len(line)), io.EOF
	}
	msg, err := jsonrpc2.DecodeMessage(json.RawMessage(line))
	return msg, int64(len(line)), err
}

func (w *lineWriter) Write(ctx context.Context, msg jsonrpc2.Message) (int64, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}
	data, err := jsonrpc2.EncodeMessage(msg)
	if err != nil {
		return 0, errors.Errorf("marshaling message: %v", err)
	}
	n, err := w.out.Write(append(data, '\n'))
	return int64(n), err
}
