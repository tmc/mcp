package mcp

import (
	"context"
	"io"

	"golang.org/x/exp/jsonrpc2"
	errors "golang.org/x/xerrors"
)

func defaultFramer() jsonrpc2.Framer { return lineFramer{} }

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
