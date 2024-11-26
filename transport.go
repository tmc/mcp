package mcp

import (
	"context"
	"io"
	"os"
)

// Transport handles communication between client and server
type Transport interface {
	io.ReadWriteCloser
	// Context returns the context for this transport
	Context() context.Context
}

// StdioTransport implements Transport over stdin/stdout
type StdioTransport struct {
	ctx context.Context
	in  io.Reader
	out io.Writer
}

// NewStdioTransport creates a new stdio transport
func NewStdioTransport(ctx context.Context) *StdioTransport {
	return &StdioTransport{
		ctx: ctx,
		in:  os.Stdin,
		out: os.Stdout,
	}
}

func (t *StdioTransport) Read(p []byte) (n int, err error)  { return t.in.Read(p) }
func (t *StdioTransport) Write(p []byte) (n int, err error) { return t.out.Write(p) }
func (t *StdioTransport) Close() error                      { return nil }
func (t *StdioTransport) Context() context.Context          { return t.ctx }

