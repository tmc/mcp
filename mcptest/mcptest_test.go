package mcptest_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tmc/mcp/mcptest"
)

func peers() (*mcp.Server, *mcp.Client) {
	return mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1"}, nil), mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
}

func TestCloseUninitialized(t *testing.T) {
	for _, fixture := range []*mcptest.Fixture{nil, {}} {
		if err := fixture.Close(context.Background()); err == nil {
			t.Fatal("Close accepted an uninitialized fixture")
		}
	}
}

func TestCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		server, client := peers()
		started, canceled := make(chan struct{}), make(chan struct{})
		mcp.AddTool(server, &mcp.Tool{Name: "wait"}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, struct{}, error) {
			close(started)
			<-ctx.Done()
			close(canceled)
			return nil, struct{}{}, ctx.Err()
		})
		fixture := mcptest.Connect(t, server, client)
		ctx, cancel := context.WithCancel(context.Background())
		result := make(chan error, 1)
		go func() { _, err := fixture.Client.CallTool(ctx, &mcp.CallToolParams{Name: "wait"}); result <- err }()
		<-started
		cancel()
		if err := <-result; !errors.Is(err, context.Canceled) {
			t.Fatalf("CallTool error = %v, want context.Canceled", err)
		}
		<-canceled
	})
}

func TestCleanup(t *testing.T) {
	server, client := peers()
	var fixture *mcptest.Fixture
	t.Run("fixture", func(t *testing.T) { fixture = mcptest.Connect(t, server, client) })
	if _, err := fixture.Client.ListTools(context.Background(), nil); !errors.Is(err, mcp.ErrConnectionClosed) {
		t.Fatalf("ListTools after cleanup = %v, want ErrConnectionClosed", err)
	}
	if err := fixture.Server.Wait(); err != nil {
		t.Fatalf("Server.Wait: %v", err)
	}
}

func TestConcurrentClose(t *testing.T) {
	server, client := peers()
	fixture := mcptest.Connect(t, server, client)
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			if err := fixture.Close(context.Background()); err != nil {
				t.Errorf("Close: %v", err)
			}
		})
	}
	wg.Wait()
}

func TestNewNilPeer(t *testing.T) {
	server, client := peers()
	for _, test := range []struct {
		name   string
		server *mcp.Server
		client *mcp.Client
	}{
		{"server", nil, client}, {"client", server, nil}, {"both", nil, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			if fixture, err := mcptest.New(context.Background(), test.server, test.client); err == nil || fixture != nil {
				t.Fatalf("New = %v, %v; want nil fixture and error", fixture, err)
			}
		})
	}
}

func TestCloseDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		server, client := peers()
		entered, release := make(chan struct{}), make(chan struct{})
		mcp.AddTool(server, &mcp.Tool{Name: "blocked"}, func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, struct{}, error) {
			close(entered)
			<-release
			return nil, struct{}{}, nil
		})
		fixture := mcptest.Connect(t, server, client)
		done := make(chan struct{})
		go func() {
			defer close(done)
			fixture.Client.CallTool(context.Background(), &mcp.CallToolParams{Name: "blocked"})
		}()
		<-entered
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := fixture.Close(ctx); !errors.Is(err, context.Canceled) {
			t.Fatalf("Close = %v, want context.Canceled", err)
		}
		close(release)
		if err := fixture.Close(context.Background()); err != nil {
			t.Fatalf("Close after release: %v", err)
		}
		<-done
	})
}

func TestClientCallbacks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		server, _ := peers()
		progress := make(chan string, 1)
		client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, &mcp.ClientOptions{
			ProgressNotificationHandler: func(_ context.Context, req *mcp.ProgressNotificationClientRequest) { progress <- req.Params.Message },
		})
		mcp.AddTool(server, &mcp.Tool{Name: "progress"}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, struct{}, error) {
			err := req.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{ProgressToken: req.Params.GetProgressToken(), Message: "ready", Progress: 1})
			return nil, struct{}{}, err
		})
		fixture := mcptest.Connect(t, server, client)
		if _, err := fixture.Client.CallTool(context.Background(), &mcp.CallToolParams{Name: "progress", Meta: mcp.Meta{"progressToken": "test"}}); err != nil {
			t.Fatal(err)
		}
		if got := <-progress; got != "ready" {
			t.Fatalf("progress = %q, want ready", got)
		}
	})
}
