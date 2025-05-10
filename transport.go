package mcp

import (
	"context"
	"io"
)

// Transport accepts a context and returns a ReadWriteCloser.
// Either:
// 1. A function that takes a context and returns a ReadWriteCloser and error, or
// 2. An object that implements Dial(context.Context) (io.ReadWriteCloser, error)
type Transport interface {
	Dial(context.Context) (io.ReadWriteCloser, error)
}

// TransportFunc implements Transport with a function.
type TransportFunc func(context.Context) (io.ReadWriteCloser, error)

// Dial implements Transport interface.
func (t TransportFunc) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	return t(ctx)
}

type ReadWriteCloserTransport struct {
	io.ReadWriteCloser
}

func (t *ReadWriteCloserTransport) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	return t.ReadWriteCloser, nil
}
