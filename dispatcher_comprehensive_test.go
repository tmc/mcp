package mcp

import (
	"context"
	"encoding/json"
	"testing"
)

func TestDispatcherBasics(t *testing.T) {
	dispatcher := NewDispatcher()

	t.Run("handle and dispatch notification", func(t *testing.T) {
		called := false
		handler := func(method string, params json.RawMessage) error {
			called = true
			if method != "test/notification" {
				t.Errorf("Expected method test/notification, got %s", method)
			}

			err := dispatcher.Dispatch(ctx, tt.method, tt.params)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.setupHandler != nil {
				if callCount == 0 {
					t.Error("Handler should have been called")
				}

				if lastMethod != tt.method {
					t.Errorf("Method mismatch: expected %s, got %s", tt.method, lastMethod)
				}

				if tt.params != nil && string(lastParams) != string(tt.params) {
					t.Errorf("Params mismatch: expected %s, got %s", string(tt.params), string(lastParams))
				}
			}
		})
	}
}

func TestDispatcher_DispatchMultipleHandlers(t *testing.T) {
	dispatcher := NewDispatcher()
	ctx := context.Background()

	var callOrder []int
	var mu sync.Mutex

	// Register multiple handlers that track call order
	handler1 := func(method string, params json.RawMessage) error {
		mu.Lock()
		callOrder = append(callOrder, 1)
		mu.Unlock()
		return nil
	}

	handler2 := func(method string, params json.RawMessage) error {
		mu.Lock()
		callOrder = append(callOrder, 2)
		mu.Unlock()
		return nil
	}

	handler3 := func(method string, params json.RawMessage) error {
		mu.Lock()
		callOrder = append(callOrder, 3)
		mu.Unlock()
		return errors.New("error from handler 3")
	}

	dispatcher.Handle("multi.dispatch", handler1)
	dispatcher.Handle("multi.dispatch", handler2)
	dispatcher.Handle("multi.dispatch", handler3)

	err := dispatcher.Dispatch(ctx, "multi.dispatch", json.RawMessage(`{}`))

	// Should get error because handler3 returns error
	if err == nil {
		t.Error("Expected error from handler3")
	}

	// All handlers should still be called
	mu.Lock()
	expectedOrder := []int{1, 2, 3}
	mu.Unlock()

	if len(callOrder) != len(expectedOrder) {
		t.Errorf("Call order length mismatch: expected %d, got %d", len(expectedOrder), len(callOrder))
	}

	for i, expected := range expectedOrder {
		if i >= len(callOrder) || callOrder[i] != expected {
			t.Errorf("Call order mismatch at index %d: expected %d, got %d", i, expected, callOrder[i])
		}
	}
}

func TestDispatcher_ConcurrentAccess(t *testing.T) {
	dispatcher := NewDispatcher()
	ctx := context.Background()

	// Counter to track handler invocations
	var counter int64
	var mu sync.Mutex

	// Handler that increments counter
	testHandler := func(method string, params json.RawMessage) error {
		mu.Lock()
		counter++
		mu.Unlock()
		return nil
	}

	dispatcher.Handle("concurrent.test", testHandler)

	// Launch multiple goroutines to test concurrent dispatch
	const numGoroutines = 50
	const numDispatchesPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numDispatchesPerGoroutine; j++ {
				params := json.RawMessage(fmt.Sprintf(`{"goroutine": %d, "iteration": %d}`, id, j))
				err := dispatcher.Dispatch(ctx, "concurrent.test", params)
				if err != nil {
					t.Errorf("Dispatch error: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	mu.Lock()
	finalCounter := counter
	mu.Unlock()

	expectedCount := int64(numGoroutines * numDispatchesPerGoroutine)
	if finalCounter != expectedCount {
		t.Errorf("Counter mismatch: expected %d, got %d", expectedCount, finalCounter)
	}
}

func TestDispatcher_NotifyListChanged(t *testing.T) {
	dispatcher := NewDispatcher()
	ctx := context.Background()

	var receivedMethod string
	var receivedParams json.RawMessage

	// Register handler for list changed notification
	dispatcher.Handle(string(MethodToolListChanged), func(method string, params json.RawMessage) error {
		receivedMethod = method
		receivedParams = params
		return nil
	})

	err := dispatcher.NotifyListChanged(ctx, MethodToolListChanged)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if receivedMethod != string(MethodToolListChanged) {
		t.Errorf("Method mismatch: expected %s, got %s", string(MethodToolListChanged), receivedMethod)
	}

	// Should receive empty object as params
	expectedParams := "{}"
	if string(receivedParams) != expectedParams {
		t.Errorf("Params mismatch: expected %s, got %s", expectedParams, string(receivedParams))
	}
}

func TestDispatcher_NotifyProgress(t *testing.T) {
	dispatcher := NewDispatcher()
	ctx := context.Background()

	var receivedMethod string
	var receivedParams json.RawMessage

	// Register handler for progress notification
	dispatcher.Handle(string(MethodProgress), func(method string, params json.RawMessage) error {
		receivedMethod = method
		receivedParams = params
		return nil
	})

	token := "test-token"
	progress := 0.5
	total := 1.0

	err := dispatcher.NotifyProgress(ctx, token, progress, &total)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if receivedMethod != string(MethodProgress) {
		t.Errorf("Method mismatch: expected %s, got %s", string(MethodProgress), receivedMethod)
	}

	// Parse and validate params
	var params map[string]interface{}
	if err := json.Unmarshal(receivedParams, &params); err != nil {
		t.Errorf("Failed to unmarshal params: %v", err)
	}

	if params["progressToken"] != token {
		t.Errorf("Token mismatch: expected %s, got %v", token, params["progressToken"])
	}

	if params["progress"] != progress {
		t.Errorf("Progress mismatch: expected %f, got %v", progress, params["progress"])
	}

	if params["total"] != total {
		t.Errorf("Total mismatch: expected %f, got %v", total, params["total"])
	}
}

func TestDispatcher_NotifyLoggingMessage(t *testing.T) {
	dispatcher := NewDispatcher()
	ctx := context.Background()

	var receivedMethod string
	var receivedParams json.RawMessage

	// Register handler for logging notification
	dispatcher.Handle(string(MethodLogging), func(method string, params json.RawMessage) error {
		receivedMethod = method
		receivedParams = params
		return nil
	})

	level := LogLevelInfo
	logger := "test-logger"
	data := "test message"

	err := dispatcher.NotifyLoggingMessage(ctx, level, logger, data)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if receivedMethod != string(MethodLogging) {
		t.Errorf("Method mismatch: expected %s, got %s", string(MethodLogging), receivedMethod)
	}

	// Parse and validate params
	var params map[string]interface{}
	if err := json.Unmarshal(receivedParams, &params); err != nil {
		t.Errorf("Failed to unmarshal params: %v", err)
	}

	if params["level"] != string(level) {
		t.Errorf("Level mismatch: expected %s, got %v", string(level), params["level"])
	}

	if params["logger"] != logger {
		t.Errorf("Logger mismatch: expected %s, got %v", logger, params["logger"])
	}
}

func TestDispatcher_InvalidHandlerType(t *testing.T) {
	dispatcher := NewDispatcher()
	ctx := context.Background()

	// Add a non-NotificationHandler to the handlers slice directly
	dispatcher.mu.Lock()
	dispatcher.handlers["invalid.test"] = []any{"not a handler"}
	dispatcher.mu.Unlock()

	// Should not panic and should return no error
	err := dispatcher.Dispatch(ctx, "invalid.test", json.RawMessage(`{}`))
	if err != nil {
		t.Errorf("Unexpected error for invalid handler type: %v", err)
	}
}

func TestDispatcher_ContextCancellation(t *testing.T) {
	dispatcher := NewDispatcher()

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	var handlerStarted bool

	// Handler that checks context but doesn't use it (current implementation doesn't pass context to handlers)
	slowHandler := func(method string, params json.RawMessage) error {
		handlerStarted = true
		// Simulate some work
		time.Sleep(100 * time.Millisecond)
		return nil
	}

	dispatcher.Handle("slow.test", slowHandler)

	// Start dispatch in goroutine
	var err error
	done := make(chan struct{})
	go func() {
		defer close(done)
		err = dispatcher.Dispatch(ctx, "slow.test", json.RawMessage(`{}`))
	}()

	// Cancel context after a short delay
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Wait for dispatch to complete
	<-done

	if !handlerStarted {
		t.Error("Handler should have started")
	}

	// Note: Current implementation doesn't pass context to handlers,
	// so cancellation won't interrupt handler execution
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestDispatcher_EmptyMethod(t *testing.T) {
	dispatcher := NewDispatcher()
	ctx := context.Background()

	// Test empty method name - should not panic
	err := dispatcher.Dispatch(ctx, "", json.RawMessage(`{}`))
	if err != nil {
		t.Errorf("Unexpected error for empty method: %v", err)
	}
}

func TestDispatcher_LargeParams(t *testing.T) {
	dispatcher := NewDispatcher()
	ctx := context.Background()

	var receivedSize int

	// Handler that checks param size
	sizeHandler := func(method string, params json.RawMessage) error {
		receivedSize = len(params)
		return nil
	}

	dispatcher.Handle("size.test", sizeHandler)

	// Create large JSON params
	largeData := make(map[string]string)
	for i := 0; i < 1000; i++ {
		largeData[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
	}

	largeParams, err := json.Marshal(largeData)
	if err != nil {
		t.Fatalf("Failed to marshal large params: %v", err)
	}

	// Dispatch with large params
	err = dispatcher.Dispatch(ctx, "size.test", largeParams)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if receivedSize != len(largeParams) {
		t.Errorf("Size mismatch: expected %d, got %d", len(largeParams), receivedSize)
	}
}

func BenchmarkDispatcher_Dispatch(b *testing.B) {
	dispatcher := NewDispatcher()
	ctx := context.Background()

	// Simple handler
	handler := func(method string, params json.RawMessage) error {
		return nil
	}

	dispatcher.Handle("bench.test", handler)
	params := json.RawMessage(`{"test": "data"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := dispatcher.Dispatch(ctx, "bench.test", params)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDispatcher_ConcurrentDispatch(b *testing.B) {
	dispatcher := NewDispatcher()
	ctx := context.Background()

	// Simple handler
	handler := func(method string, params json.RawMessage) error {
		return nil
	}

	dispatcher.Handle("bench.concurrent", handler)
	params := json.RawMessage(`{"test": "data"}`)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := dispatcher.Dispatch(ctx, "bench.concurrent", params)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkDispatcher_MultipleHandlers(b *testing.B) {
	dispatcher := NewDispatcher()
	ctx := context.Background()

	// Register multiple handlers
	for i := 0; i < 10; i++ {
		dispatcher.Handle("bench.multi", func(method string, params json.RawMessage) error {
			return nil
		}
	}
}
