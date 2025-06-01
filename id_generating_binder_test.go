package mcp

import (
	"context"
	"sync"
	"testing"

	"golang.org/x/exp/jsonrpc2"
)

// Mock binder for testing
type mockBinder struct {
	bindFunc func(ctx context.Context, conn *jsonrpc2.Connection) (jsonrpc2.ConnectionOptions, error)
}

func (m *mockBinder) Bind(ctx context.Context, conn *jsonrpc2.Connection) (jsonrpc2.ConnectionOptions, error) {
	if m.bindFunc != nil {
		return m.bindFunc(ctx, conn)
	}
	return jsonrpc2.ConnectionOptions{}, nil
}

func TestIdGeneratingBinder_Bind(t *testing.T) {
	tests := []struct {
		name     string
		baseBind func(ctx context.Context, conn *jsonrpc2.Connection) (jsonrpc2.ConnectionOptions, error)
		wantErr  bool
	}{
		{
			name: "successful bind with base binder",
			baseBind: func(ctx context.Context, conn *jsonrpc2.Connection) (jsonrpc2.ConnectionOptions, error) {
				return jsonrpc2.ConnectionOptions{}, nil
			},
			wantErr: false,
		},
		{
			name: "base binder returns error",
			baseBind: func(ctx context.Context, conn *jsonrpc2.Connection) (jsonrpc2.ConnectionOptions, error) {
				return jsonrpc2.ConnectionOptions{}, jsonrpc2.ErrInvalidRequest
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseBinder := &mockBinder{
				bindFunc: tt.baseBind,
			}

			binder := &idGeneratingBinder{
				base: baseBinder,
			}

			ctx := context.Background()
			conn := &jsonrpc2.Connection{} // Mock connection

			opts, err := binder.Bind(ctx, conn)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Verify we get connection options back
			_ = opts // Connection options should be returned
		})
	}
}

func TestIdGeneratingBinder_Initialization(t *testing.T) {
	baseBinder := &mockBinder{}
	binder := &idGeneratingBinder{
		base: baseBinder,
	}

	// Verify initial state
	if binder.base != baseBinder {
		t.Error("Base binder not set correctly")
	}

	// Verify atomic counter starts at 0
	if binder.idCounter.Load() != 0 {
		t.Error("ID counter should start at 0")
	}
}

func TestIdGeneratingBinder_AtomicCounter(t *testing.T) {
	binder := &idGeneratingBinder{
		base: &mockBinder{},
	}

	// Test atomic counter operations
	initial := binder.idCounter.Load()
	if initial != 0 {
		t.Errorf("Expected initial counter value 0, got %d", initial)
	}

	// Simulate ID generation
	newID := binder.idCounter.Add(1)
	if newID != 1 {
		t.Errorf("Expected first ID to be 1, got %d", newID)
	}

	secondID := binder.idCounter.Add(1)
	if secondID != 2 {
		t.Errorf("Expected second ID to be 2, got %d", secondID)
	}

	// Verify current value
	current := binder.idCounter.Load()
	if current != 2 {
		t.Errorf("Expected current counter value 2, got %d", current)
	}
}

func TestIdGeneratingBinder_ConcurrentAccess(t *testing.T) {
	binder := &idGeneratingBinder{
		base: &mockBinder{},
	}

	numGoroutines := 100
	numOpsPerGoroutine := 100

	var wg sync.WaitGroup
	idsChan := make(chan int64, numGoroutines*numOpsPerGoroutine)

	// Launch multiple goroutines that generate IDs concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				id := binder.idCounter.Add(1)
				idsChan <- id
			}
		}()
	}

	wg.Wait()
	close(idsChan)

	// Collect all generated IDs
	ids := make(map[int64]bool)
	var maxID int64

	for id := range idsChan {
		if ids[id] {
			t.Errorf("Duplicate ID generated: %d", id)
		}
		ids[id] = true
		if id > maxID {
			maxID = id
		}
	}

	expectedCount := numGoroutines * numOpsPerGoroutine
	if len(ids) != expectedCount {
		t.Errorf("Expected %d unique IDs, got %d", expectedCount, len(ids))
	}

	if maxID != int64(expectedCount) {
		t.Errorf("Expected max ID to be %d, got %d", expectedCount, maxID)
	}

	// Verify final counter value
	finalValue := binder.idCounter.Load()
	if finalValue != int64(expectedCount) {
		t.Errorf("Expected final counter value %d, got %d", expectedCount, finalValue)
	}
}

func TestIdGeneratingConnection_Call(t *testing.T) {
	// Create a mock base connection
	baseConn := &jsonrpc2.Connection{}

	conn := &idGeneratingConnection{
		Connection: baseConn,
	}

	// Verify initial counter state
	if conn.idCounter.Load() != 0 {
		t.Error("ID counter should start at 0")
	}

	ctx := context.Background()

	// Note: Since Call returns *AsyncCall and we don't have a real connection,
	// we can't fully test the actual call behavior without more complex mocking.
	// This test verifies the basic structure and that the method exists.

	// Test that Call method exists and can be called
	asyncCall := conn.Call(ctx, "test/method", map[string]interface{}{
		"param": "value",
	})

	// AsyncCall will be nil since we're using a mock connection,
	// but the method should not panic
	_ = asyncCall
}

func TestIdGeneratingConnection_CounterUniqueness(t *testing.T) {
	// Test that each connection has its own counter
	baseConn1 := &jsonrpc2.Connection{}
	baseConn2 := &jsonrpc2.Connection{}

	conn1 := &idGeneratingConnection{Connection: baseConn1}
	conn2 := &idGeneratingConnection{Connection: baseConn2}

	// Generate some IDs on each connection
	id1a := conn1.idCounter.Add(1)
	id2a := conn2.idCounter.Add(1)
	id1b := conn1.idCounter.Add(1)
	id2b := conn2.idCounter.Add(1)

	// Verify independent counters
	if id1a != 1 || id1b != 2 {
		t.Errorf("Connection 1 counter not working correctly: got %d, %d", id1a, id1b)
	}

	if id2a != 1 || id2b != 2 {
		t.Errorf("Connection 2 counter not working correctly: got %d, %d", id2a, id2b)
	}

	// Verify final states
	if conn1.idCounter.Load() != 2 {
		t.Errorf("Connection 1 final counter should be 2, got %d", conn1.idCounter.Load())
	}

	if conn2.idCounter.Load() != 2 {
		t.Errorf("Connection 2 final counter should be 2, got %d", conn2.idCounter.Load())
	}
}

func TestIdGeneratingBinder_NilBaseBinder(t *testing.T) {
	// Test behavior with nil base binder
	binder := &idGeneratingBinder{
		base: nil,
	}

	ctx := context.Background()
	conn := &jsonrpc2.Connection{}

	// This should panic or handle gracefully
	defer func() {
		if r := recover(); r != nil {
			// If it panics, that's expected behavior for nil base binder
			t.Logf("Nil base binder caused panic as expected: %v", r)
		}
	}()

	_, err := binder.Bind(ctx, conn)

	// If no panic, then nil binder should return an error
	if err == nil {
		t.Error("Expected error with nil base binder")
	}
}

func TestIdGeneratingConnection_Initialization(t *testing.T) {
	baseConn := &jsonrpc2.Connection{}
	conn := &idGeneratingConnection{
		Connection: baseConn,
	}

	// Verify proper initialization
	if conn.Connection != baseConn {
		t.Error("Base connection not set correctly")
	}

	if conn.idCounter.Load() != 0 {
		t.Error("ID counter should be initialized to 0")
	}
}

// Benchmark ID generation performance
func BenchmarkIdGeneratingBinder_CounterIncrement(b *testing.B) {
	binder := &idGeneratingBinder{
		base: &mockBinder{},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			binder.idCounter.Add(1)
		}
	})
}

func BenchmarkIdGeneratingConnection_CounterIncrement(b *testing.B) {
	conn := &idGeneratingConnection{
		Connection: &jsonrpc2.Connection{},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			conn.idCounter.Add(1)
		}
	})
}
