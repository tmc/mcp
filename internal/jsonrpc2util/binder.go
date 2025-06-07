package jsonrpc2util

import (
	"context"

	"golang.org/x/exp/jsonrpc2"
)

// ConnectionBinder implements jsonrpc2.Binder for both client and server connections
type ConnectionBinder struct {
	Handler jsonrpc2.Handler
}

// Bind implements the jsonrpc2.Binder interface
func (b ConnectionBinder) Bind(ctx context.Context, conn *jsonrpc2.Connection) (jsonrpc2.ConnectionOptions, error) {
	return jsonrpc2.ConnectionOptions{
		Handler: b.Handler,
	}, nil
}
