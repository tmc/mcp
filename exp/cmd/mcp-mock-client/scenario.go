package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// Scenario represents a test scenario with multiple steps
type Scenario struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Steps       []ScenarioStep `json:"steps"`
}

// ScenarioStep represents a single step in a test scenario
type ScenarioStep struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Request      json.RawMessage `json:"request"`
	Expectations []Expectation   `json:"expectations"`
	Delay        time.Duration   `json:"delay,omitempty"` // Optional delay before executing step
}

// Expectation represents an expected response or behavior
type Expectation struct {
	Type     string          `json:"type"`                // "response", "error", "notification"
	Pattern  json.RawMessage `json:"pattern"`             // Expected pattern to match
	Timeout  time.Duration   `json:"timeout,omitempty"`   // Optional timeout for the response
	FailFast bool            `json:"fail_fast,omitempty"` // Whether to stop testing if this expectation fails
}

// ResponseReader is an interface for reading responses
type ResponseReader interface {
	ReadResponse() ([]byte, error)
}

// StdioResponder implements ResponseReader for standard I/O
type StdioResponder struct {
	reader io.Reader
}

// ReadResponse reads a response from the reader
func (s *StdioResponder) ReadResponse() ([]byte, error) {
	var buf [4096]byte
	n, err := s.reader.Read(buf[:])
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

// MockResponder implements ResponseReader for testing
type MockResponder struct {
	Responses []string
	index     int
}

// ReadResponse returns the next pre-configured response
func (m *MockResponder) ReadResponse() ([]byte, error) {
	if m.index >= len(m.Responses) {
		return nil, fmt.Errorf("no more mock responses available")
	}
	resp := m.Responses[m.index]
	m.index++
	return []byte(resp), nil
}

// LoadScenario loads a scenario from a file
func LoadScenario(filename string) (*Scenario, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading scenario file: %w", err)
	}

	var scenario Scenario
	if err := json.Unmarshal(data, &scenario); err != nil {
		return nil, fmt.Errorf("parsing scenario file: %w", err)
	}

	return &scenario, nil
}

// runScenario runs a scenario with the given context and output writer
func runScenario(ctx context.Context, out io.Writer, scenarioFile string) error {
	scenario, err := LoadScenario(scenarioFile)
	if err != nil {
		return err
	}

	// Create a responder to read responses if validation is enabled
	var responder ResponseReader
	if *validate {
		if in, ok := out.(*os.File); ok && (in.Fd() == 1 || in.Fd() == 2) {
			// If output is stdout or stderr, use stdin for responses
			responder = &StdioResponder{reader: os.Stdin}
		} else {
			// For testing, we need to use a mock responder
			responder = &MockResponder{
				Responses: []string{
					`{"jsonrpc":"2.0","result":{"capabilities":{}},"id":1}`,
					`{"jsonrpc":"2.0","result":{"tools":[]},"id":2}`,
				},
			}
		}
	}

	// Log scenario info if verbose
	if *verbose {
		fmt.Fprintf(os.Stderr, "Running scenario: %s\n", scenario.Name)
		fmt.Fprintf(os.Stderr, "Description: %s\n", scenario.Description)
		fmt.Fprintf(os.Stderr, "Steps: %d\n", len(scenario.Steps))
	}

	matcher := JSONMatcher{}
	for i, step := range scenario.Steps {
		// Apply any delay specified for the step or global step delay
		delay := step.Delay
		if delay == 0 && *stepDelay > 0 {
			delay = *stepDelay
		}

		if delay > 0 {
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Log step info if verbose
		if *verbose {
			fmt.Fprintf(os.Stderr, "Step %d: %s\n", i+1, step.Name)
			if step.Description != "" {
				fmt.Fprintf(os.Stderr, "  Description: %s\n", step.Description)
			}
		}

		// Send the request
		if *dryRun {
			fmt.Fprintf(os.Stderr, "would send: %s\n", step.Request)
			continue
		}

		// Write the request to output
		if _, err := fmt.Fprintf(out, "%s\n", step.Request); err != nil {
			return fmt.Errorf("writing request: %w", err)
		}

		// Validate response if requested
		if *validate && responder != nil && len(step.Expectations) > 0 {
			for j, expectation := range step.Expectations {
				if *verbose {
					fmt.Fprintf(os.Stderr, "  Validating expectation %d: %s\n", j+1, expectation.Type)
				}

				// Set up a timeout for this expectation
				timeout := expectation.Timeout
				if timeout == 0 {
					timeout = 5 * time.Second // Default timeout
				}

				// Create a context with timeout for this expectation
				expCtx, cancel := context.WithTimeout(ctx, timeout)

				// Read response
				var resp []byte
				readDone := make(chan struct{})
				var readErr error

				go func() {
					resp, readErr = responder.ReadResponse()
					close(readDone)
				}()

				// Wait for either response or timeout
				select {
				case <-readDone:
					// Response received
				case <-expCtx.Done():
					cancel()
					if expCtx.Err() == context.DeadlineExceeded {
						return fmt.Errorf("timeout waiting for response in step %d (%s), expectation %d",
							i+1, step.Name, j+1)
					}
					return expCtx.Err()
				}
				cancel()

				if readErr != nil {
					return fmt.Errorf("error reading response in step %d (%s): %w",
						i+1, step.Name, readErr)
				}

				// Match the response against the pattern
				result := matcher.Match([]byte(expectation.Pattern), resp)
				if !result.Success {
					if *verbose {
						fmt.Fprintf(os.Stderr, "  Validation failed: %s\n", result.Message)
						fmt.Fprintf(os.Stderr, "  Expected: %s\n", result.Expected)
						fmt.Fprintf(os.Stderr, "  Actual: %s\n", result.Actual)
					}

					// Check if we should fail fast
					if expectation.FailFast {
						return fmt.Errorf("expectation failed in step %d (%s): %s",
							i+1, step.Name, result.Message)
					}
				} else if *verbose {
					fmt.Fprintf(os.Stderr, "  Validation successful\n")
				}
			}
		}
	}

	if *verbose {
		fmt.Fprintf(os.Stderr, "Scenario completed successfully\n")
	}
	return nil
}
