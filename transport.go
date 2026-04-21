package mcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
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
	if t.ReadWriteCloser == nil {
		return nil, transportClosedError("transport dial")
	}
	return t.ReadWriteCloser, nil
}

func transportClosedError(op string) error {
	return fmt.Errorf("%s: %w", op, ErrTransportClosed)
}

func wrapTransportClosed(op string, err error) error {
	if errors.Is(err, ErrTransportClosed) {
		return err
	}
	if err == nil {
		return nil
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe) || errors.Is(err, net.ErrClosed) {
		return transportClosedError(op)
	}
	if strings.Contains(err.Error(), "use of closed network connection") {
		return transportClosedError(op)
	}
	return err
}
