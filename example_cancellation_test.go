package mcp_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/tmc/mcp"
)

// Example demonstrates how to use context.WithCancelCause to provide
// detailed cancellation reasons that are automatically propagated to the server
func Example_cancellationWithCause() {
	// Create a client (connection setup omitted for brevity)
	var client *mcp.Client

	// Create a context with cancel cause support
	ctx, cancel := context.WithCancelCause(context.Background())

	// Start a long-running operation
	go func() {
		result, err := client.CallTool(ctx, mcp.CallToolRequest{
			Name:      "analyze_data",
			Arguments: []byte(`{"dataset": "large_dataset.csv"}`),
		})

		if err != nil {
			// Check if it was cancelled
			if errors.Is(err, context.Canceled) {
				// The cancellation cause will have been automatically sent to the server
				fmt.Printf("Operation cancelled: %v\n", context.Cause(ctx))
			} else {
				fmt.Printf("Operation failed: %v\n", err)
			}
			return
		}

		fmt.Printf("Result: %v\n", result)
	}()

	// Simulate user cancelling after some time
	time.Sleep(2 * time.Second)

	// Cancel with a specific reason - this will be sent to the server automatically
	userReason := errors.New("user clicked the stop button")
	cancel(userReason)

	// The client will automatically send a notifications/cancelled message
	// to the server with the reason from the cause
}

// Example demonstrates how to handle different cancellation scenarios
func Example_cancellationScenarios() {
	var client *mcp.Client

	// Scenario 1: Simple cancellation (no specific reason)
	ctx1, cancel1 := context.WithCancel(context.Background())
	go func() {
		client.CallTool(ctx1, mcp.CallToolRequest{Name: "task1"})
	}()
	cancel1() // This sends a cancellation notification without a reason

	// Scenario 2: Cancellation with timeout
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	go func() {
		client.CallTool(ctx2, mcp.CallToolRequest{Name: "task2"})
		// If this times out, the cancellation notification will include
		// "context deadline exceeded" as the reason
	}()

	// Scenario 3: Cancellation with custom error
	ctx3, cancel3 := context.WithCancelCause(context.Background())
	go func() {
		client.CallTool(ctx3, mcp.CallToolRequest{Name: "task3"})
	}()

	// Cancel with a custom error that will be sent as the reason
	customError := fmt.Errorf("resource limit exceeded: memory usage at 95%%")
	cancel3(customError)

	// Scenario 4: Cancellation with structured error information
	ctx4, cancel4 := context.WithCancelCause(context.Background())
	go func() {
		client.CallTool(ctx4, mcp.CallToolRequest{Name: "task4"})
	}()

	// Use a custom error type for more structured information
	type CancellationReason struct {
		Code    string
		Message string
		Details map[string]interface{}
	}

	reason := &CancellationReason{
		Code:    "USER_ABORT",
		Message: "User aborted the operation",
		Details: map[string]interface{}{
			"elapsed_time": "45s",
			"progress":     "75%",
		},
	}

	cancel4(fmt.Errorf("%+v", reason))
}
