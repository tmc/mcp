package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/tmc/mcp/modelcontextprotocol"
)

// TestProgressNotificationBasic tests basic progress notification functionality
func TestProgressNotificationBasic(t *testing.T) {
	dispatcher := NewDispatcher()

	var receivedNotifications []modelcontextprotocol.ProgressNotificationParams
	var mu sync.Mutex

	// Register handler
	dispatcher.Handle(string(MethodProgress), func(method string, params json.RawMessage) error {
		var progress modelcontextprotocol.ProgressNotificationParams
		if err := json.Unmarshal(params, &progress); err != nil {
			return err
		}

		mu.Lock()
		receivedNotifications = append(receivedNotifications, progress)
		mu.Unlock()

		return nil
	})

	ctx := context.Background()
	token := "test-token-123"
	progress := 50.0
	total := 100.0

	// Send progress notification
	err := dispatcher.NotifyProgress(ctx, token, progress, &total)
	if err != nil {
		t.Fatalf("Failed to send progress notification: %v", err)
	}

	// Verify notification was received
	mu.Lock()
	defer mu.Unlock()

	if len(receivedNotifications) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(receivedNotifications))
	}

	notification := receivedNotifications[0]
	if notification.ProgressToken != token {
		t.Errorf("Expected token %v, got %v", token, notification.ProgressToken)
	}

	if notification.Progress != progress {
		t.Errorf("Expected progress %f, got %f", progress, notification.Progress)
	}

	if notification.Total == nil || *notification.Total != total {
		t.Errorf("Expected total %f, got %v", total, notification.Total)
	}
}

// TestProgressNotificationWithoutTotal tests progress notification without total
func TestProgressNotificationWithoutTotal(t *testing.T) {
	dispatcher := NewDispatcher()

	var receivedParams modelcontextprotocol.ProgressNotificationParams
	var received bool

	dispatcher.Handle(string(MethodProgress), func(method string, params json.RawMessage) error {
		received = true
		return json.Unmarshal(params, &receivedParams)
	})

	ctx := context.Background()
	token := "indeterminate-progress"
	progress := 0.0

	// Send progress notification without total
	err := dispatcher.NotifyProgress(ctx, token, progress, nil)
	if err != nil {
		t.Fatalf("Failed to send progress notification: %v", err)
	}

	if !received {
		t.Fatal("Expected notification to be received")
	}

	if receivedParams.Total != nil {
		t.Errorf("Expected nil total, got %v", receivedParams.Total)
	}

	if receivedParams.Progress != progress {
		t.Errorf("Expected progress %f, got %f", progress, receivedParams.Progress)
	}
}

// TestProgressNotificationDifferentTokenTypes tests different token types
func TestProgressNotificationDifferentTokenTypes(t *testing.T) {
	dispatcher := NewDispatcher()

	var receivedTokens []any
	var mu sync.Mutex

	dispatcher.Handle(string(MethodProgress), func(method string, params json.RawMessage) error {
		var progress modelcontextprotocol.ProgressNotificationParams
		if err := json.Unmarshal(params, &progress); err != nil {
			return err
		}

		mu.Lock()
		receivedTokens = append(receivedTokens, progress.ProgressToken)
		mu.Unlock()

		return nil
	})

	ctx := context.Background()

	// Test different token types
	testCases := []struct {
		name  string
		token any
	}{
		{"string token", "string-token"},
		{"integer token", 12345},
		{"float token", 123.45},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := dispatcher.NotifyProgress(ctx, tc.token, 50.0, nil)
			if err != nil {
				t.Fatalf("Failed to send progress notification: %v", err)
			}
		})
	}

	// Verify all tokens were received
	mu.Lock()
	defer mu.Unlock()

	if len(receivedTokens) != len(testCases) {
		t.Fatalf("Expected %d tokens, got %d", len(testCases), len(receivedTokens))
	}
}

// TestProgressNotificationMultipleHandlers tests multiple handlers for progress
func TestProgressNotificationMultipleHandlers(t *testing.T) {
	dispatcher := NewDispatcher()

	var handler1Called, handler2Called bool

	dispatcher.Handle(string(MethodProgress), func(method string, params json.RawMessage) error {
		handler1Called = true
		return nil
	})

	dispatcher.Handle(string(MethodProgress), func(method string, params json.RawMessage) error {
		handler2Called = true
		return nil
	})

	ctx := context.Background()
	err := dispatcher.NotifyProgress(ctx, "test", 50.0, nil)
	if err != nil {
		t.Fatalf("Failed to send progress notification: %v", err)
	}

	if !handler1Called {
		t.Error("Handler 1 was not called")
	}

	if !handler2Called {
		t.Error("Handler 2 was not called")
	}
}

// TestProgressNotificationSequence tests a sequence of progress notifications
func TestProgressNotificationSequence(t *testing.T) {
	dispatcher := NewDispatcher()

	var progressSequence []float64
	var mu sync.Mutex

	dispatcher.Handle(string(MethodProgress), func(method string, params json.RawMessage) error {
		var progress modelcontextprotocol.ProgressNotificationParams
		if err := json.Unmarshal(params, &progress); err != nil {
			return err
		}

		mu.Lock()
		progressSequence = append(progressSequence, progress.Progress)
		mu.Unlock()

		return nil
	})

	ctx := context.Background()
	token := "sequence-token"
	total := 100.0

	// Send sequence of progress updates
	expectedSequence := []float64{0.0, 25.0, 50.0, 75.0, 100.0}

	for _, prog := range expectedSequence {
		err := dispatcher.NotifyProgress(ctx, token, prog, &total)
		if err != nil {
			t.Fatalf("Failed to send progress notification: %v", err)
		}
	}

	// Verify sequence
	mu.Lock()
	defer mu.Unlock()

	if len(progressSequence) != len(expectedSequence) {
		t.Fatalf("Expected %d progress updates, got %d", len(expectedSequence), len(progressSequence))
	}

	for i, expected := range expectedSequence {
		if progressSequence[i] != expected {
			t.Errorf("Progress %d: expected %f, got %f", i, expected, progressSequence[i])
		}
	}
}

// TestProgressNotificationConcurrency tests concurrent progress notifications
func TestProgressNotificationConcurrency(t *testing.T) {
	dispatcher := NewDispatcher()

	var notificationCount int
	var mu sync.Mutex

	dispatcher.Handle(string(MethodProgress), func(method string, params json.RawMessage) error {
		mu.Lock()
		notificationCount++
		mu.Unlock()
		return nil
	})

	ctx := context.Background()
	numGoroutines := 10
	notificationsPerGoroutine := 5

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < notificationsPerGoroutine; j++ {
				token := fmt.Sprintf("token-%d-%d", goroutineID, j)
				progress := float64(j) * 20.0

				err := dispatcher.NotifyProgress(ctx, token, progress, nil)
				if err != nil {
					t.Errorf("Failed to send progress notification: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify all notifications were received
	mu.Lock()
	defer mu.Unlock()

	expected := numGoroutines * notificationsPerGoroutine
	if notificationCount != expected {
		t.Errorf("Expected %d notifications, got %d", expected, notificationCount)
	}
}

// TestProgressNotificationWithMessage tests progress with message field
func TestProgressNotificationWithMessage(t *testing.T) {
	dispatcher := NewDispatcher()

	var receivedParams modelcontextprotocol.ProgressNotificationParams
	var received bool

	dispatcher.Handle(string(MethodProgress), func(method string, params json.RawMessage) error {
		received = true
		return json.Unmarshal(params, &receivedParams)
	})

	// Create progress notification with message manually
	// (since the current NotifyProgress doesn't support message field)
	progressParams := modelcontextprotocol.ProgressNotificationParams{
		ProgressToken: "test-token",
		Progress:      75.0,
		Total:         &[]float64{100.0}[0],
		Message:       &[]string{"Processing files..."}[0],
	}

	data, err := json.Marshal(progressParams)
	if err != nil {
		t.Fatalf("Failed to marshal progress params: %v", err)
	}

	ctx := context.Background()
	err = dispatcher.Dispatch(ctx, string(MethodProgress), data)
	if err != nil {
		t.Fatalf("Failed to dispatch progress notification: %v", err)
	}

	if !received {
		t.Fatal("Expected notification to be received")
	}

	if receivedParams.Message == nil || *receivedParams.Message != "Processing files..." {
		t.Errorf("Expected message 'Processing files...', got %v", receivedParams.Message)
	}
}

// TestProgressNotificationJSONSerialization tests JSON serialization/deserialization
func TestProgressNotificationJSONSerialization(t *testing.T) {
	original := modelcontextprotocol.ProgressNotificationParams{
		ProgressToken: "serialization-test",
		Progress:      42.5,
		Total:         &[]float64{100.0}[0],
		Message:       &[]string{"Test message"}[0],
	}

	// Serialize
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Deserialize
	var restored modelcontextprotocol.ProgressNotificationParams
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify
	if restored.ProgressToken != original.ProgressToken {
		t.Errorf("Token mismatch: %v != %v", restored.ProgressToken, original.ProgressToken)
	}

	if restored.Progress != original.Progress {
		t.Errorf("Progress mismatch: %f != %f", restored.Progress, original.Progress)
	}

	if restored.Total == nil || *restored.Total != *original.Total {
		t.Errorf("Total mismatch: %v != %v", restored.Total, original.Total)
	}

	if restored.Message == nil || *restored.Message != *original.Message {
		t.Errorf("Message mismatch: %v != %v", restored.Message, original.Message)
	}
}

// TestProgressNotificationBoundaryValues tests boundary values
func TestProgressNotificationBoundaryValues(t *testing.T) {
	dispatcher := NewDispatcher()

	var receivedProgress []float64
	var mu sync.Mutex

	dispatcher.Handle(string(MethodProgress), func(method string, params json.RawMessage) error {
		var progress modelcontextprotocol.ProgressNotificationParams
		if err := json.Unmarshal(params, &progress); err != nil {
			return err
		}

		mu.Lock()
		receivedProgress = append(receivedProgress, progress.Progress)
		mu.Unlock()

		return nil
	})

	ctx := context.Background()

	// Test boundary values
	boundaryValues := []float64{0.0, 1.0, 99.0, 100.0, -1.0, 150.0}

	for _, value := range boundaryValues {
		err := dispatcher.NotifyProgress(ctx, "boundary-test", value, nil)
		if err != nil {
			t.Fatalf("Failed to send progress %f: %v", value, err)
		}
	}

	// Verify all values were received
	mu.Lock()
	defer mu.Unlock()

	if len(receivedProgress) != len(boundaryValues) {
		t.Fatalf("Expected %d values, got %d", len(boundaryValues), len(receivedProgress))
	}

	for i, expected := range boundaryValues {
		if receivedProgress[i] != expected {
			t.Errorf("Value %d: expected %f, got %f", i, expected, receivedProgress[i])
		}
	}
}

// BenchmarkProgressNotification benchmarks progress notification performance
func BenchmarkProgressNotification(b *testing.B) {
	dispatcher := NewDispatcher()

	dispatcher.Handle(string(MethodProgress), func(method string, params json.RawMessage) error {
		return nil // No-op handler
	})

	ctx := context.Background()
	total := 100.0

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		token := "benchmark-token"
		progress := float64(i % 101)

		err := dispatcher.NotifyProgress(ctx, token, progress, &total)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestProgressNotificationErrorHandling tests error handling in handlers
func TestProgressNotificationErrorHandling(t *testing.T) {
	dispatcher := NewDispatcher()

	// Handler that always returns an error
	dispatcher.Handle(string(MethodProgress), func(method string, params json.RawMessage) error {
		return fmt.Errorf("test error")
	})

	// Handler that succeeds
	var successHandlerCalled bool
	dispatcher.Handle(string(MethodProgress), func(method string, params json.RawMessage) error {
		successHandlerCalled = true
		return nil
	})

	ctx := context.Background()
	err := dispatcher.NotifyProgress(ctx, "error-test", 50.0, nil)

	// Should return error from failing handler
	if err == nil {
		t.Error("Expected error from progress notification")
	}

	// But success handler should still be called
	if !successHandlerCalled {
		t.Error("Success handler should have been called despite error in other handler")
	}
}
