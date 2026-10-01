package mcptest

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Fixture holds connected SDK sessions. Construct it with New or Connect;
// the zero value is not usable. Do not assign to its session fields.
type Fixture struct {
	// Client is the initialized client session.
	Client *mcp.ClientSession
	// Server is the corresponding server session.
	Server *mcp.ServerSession
	once   sync.Once
	done   chan struct{}
	err    error
}

// New connects server and client through SDK in-memory transports. The context
// bounds connection establishment. The caller must close the fixture.
// Both server and client must be non-nil.
func New(ctx context.Context, server *mcp.Server, client *mcp.Client) (*Fixture, error) {
	if server == nil || client == nil {
		return nil, fmt.Errorf("connect fixture: server and client must be non-nil")
	}
	st, ct := mcp.NewInMemoryTransports()
	ss, err := server.Connect(ctx, st, nil)
	if err != nil {
		return nil, fmt.Errorf("connect server: %w", err)
	}
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		// A failed client handshake must not leave the server running.
		go ss.Close()
		return nil, fmt.Errorf("connect client: %w", err)
	}
	return &Fixture{Client: cs, Server: ss, done: make(chan struct{})}, nil
}

// Close starts graceful shutdown of both sessions and waits until they finish
// or ctx is canceled. A timed-out close continues in the background; another
// Close call may wait for completion. Handlers must return for shutdown to
// finish. Close is safe for concurrent use.
func (f *Fixture) Close(ctx context.Context) error {
	if f == nil || f.done == nil || f.Client == nil || f.Server == nil {
		return fmt.Errorf("close fixture: uninitialized fixture")
	}
	f.once.Do(func() {
		go func() {
			errs := make(chan error, 2)
			go func() { errs <- f.Client.Close() }()
			go func() { errs <- f.Server.Close() }()
			for range 2 {
				if err := <-errs; err != nil && f.err == nil {
					f.err = err
				}
			}
			close(f.done)
		}()
	})
	select {
	case <-f.done:
		if f.err != nil {
			return fmt.Errorf("close fixture: %w", f.err)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("close fixture: %w", ctx.Err())
	}
}

type testTB interface {
	Helper()
	Cleanup(func())
	Fatalf(string, ...any)
	Errorf(string, ...any)
}

// Connect connects server and client and registers fixture cleanup with t.
// It bounds startup and cleanup to five seconds each, and reports failures
// through t. A *testing.T or *testing.B satisfies the test interface.
func Connect(t testTB, server *mcp.Server, client *mcp.Client) *Fixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	fixture, err := New(ctx, server, client)
	if err != nil {
		t.Fatalf("mcptest: %v", err)
		return nil
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := fixture.Close(ctx); err != nil {
			t.Errorf("mcptest: %v", err)
		}
	})
	return fixture
}
