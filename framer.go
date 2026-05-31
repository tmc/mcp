package mcp

import (
	"context"
	"io"

	"golang.org/x/exp/jsonrpc2"
	errors "golang.org/x/xerrors"
)

// LineFramer returns a JSON-RPC framer that writes one message per line.
//
// The reader accepts ordinary JSON-RPC values as well as newline-delimited
// values, which preserves compatibility with older raw-framed peers. The writer
// uses newline-delimited JSON, matching MCP stdio and the official Go and
// TypeScript SDKs.
func LineFramer() jsonrpc2.Framer { return lineFramer{} }

// RawFramer returns the undelimited JSON-RPC framer used by older versions of
// this package.
func RawFramer() jsonrpc2.Framer { return jsonrpc2.RawFramer() }

func defaultFramer() jsonrpc2.Framer { return LineFramer() }

type lineFramer struct{}

type lineWriter struct {
	out io.Writer
}

func (lineFramer) Reader(rw io.Reader) jsonrpc2.Reader {
	return jsonrpc2.RawFramer().Reader(rw)
}

func (lineFramer) Writer(rw io.Writer) jsonrpc2.Writer {
	return &lineWriter{out: rw}
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
