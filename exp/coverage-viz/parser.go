package coverageviz

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// TraceParser parses MCP trace files
type TraceParser struct {
	reader io.Reader
}

// NewTraceParser creates a new trace parser
func NewTraceParser(r io.Reader) *TraceParser {
	return &TraceParser{reader: r}
}

// ParseMCPTrace parses an MCP trace file and returns MCPTrace entries
func (p *TraceParser) ParseMCPTrace() ([]MCPTrace, error) {
	scanner := bufio.NewScanner(p.reader)
	var traces []MCPTrace
	
	// Skip header line
	if scanner.Scan() {
		header := scanner.Text()
		if !strings.HasPrefix(header, "# mcptrace:v1") {
			return nil, fmt.Errorf("invalid MCP trace format")
		}
	}
	
	lineNum := 1
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Parse timestamp and direction
		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 3 {
			continue
		}
		
		timestamp, err := time.Parse(time.RFC3339Nano, parts[0])
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid timestamp: %w", lineNum, err)
		}
		
		direction := parts[1]
		jsonData := parts[2]
		
		trace := MCPTrace{
			Timestamp: timestamp,
			Direction: direction,
		}
		
		// Parse JSON data
		var msg map[string]interface{}
		if err := json.Unmarshal([]byte(jsonData), &msg); err != nil {
			return nil, fmt.Errorf("line %d: invalid JSON: %w", lineNum, err)
		}
		
		// Extract common fields
		if id, ok := msg["id"]; ok {
			trace.ID = fmt.Sprintf("%v", id)
		}
		
		if method, ok := msg["method"].(string); ok {
			trace.Method = method
			trace.Type = "request"
		} else if _, ok := msg["result"]; ok {
			trace.Type = "response"
			trace.Result = msg["result"]
		} else if _, ok := msg["error"]; ok {
			trace.Type = "error"
			trace.Error = msg["error"]
		} else if msg["id"] == nil && msg["method"] != nil {
			trace.Type = "notification"
			trace.Method = msg["method"].(string)
		}
		
		if params, ok := msg["params"].(map[string]interface{}); ok {
			trace.Params = params
		}
		
		traces = append(traces, trace)
	}
	
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}
	
	return traces, nil
}

// ExtractTestInfo extracts test execution information from MCP traces
func ExtractTestInfo(traces []MCPTrace) []TestExecution {
	var tests []TestExecution
	testMap := make(map[string]*TestExecution)
	
	for _, trace := range traces {
		// Look for test-related methods
		switch trace.Method {
		case "testing/start":
			if params := trace.Params; params != nil {
				testName := fmt.Sprintf("%v", params["name"])
				pkg := fmt.Sprintf("%v", params["package"])
				test := &TestExecution{
					TestName:  testName,
					Package:   pkg,
					StartTime: trace.Timestamp,
					Traces:    []MCPTrace{trace},
				}
				testMap[testName] = test
			}
			
		case "testing/end", "testing/complete":
			if params := trace.Params; params != nil {
				testName := fmt.Sprintf("%v", params["name"])
				if test, ok := testMap[testName]; ok {
					test.Duration = trace.Timestamp.Sub(test.StartTime)
					test.Traces = append(test.Traces, trace)
					
					// Determine result
					if result, ok := params["result"].(string); ok {
						switch result {
						case "pass", "passed":
							test.Result = TestPassed
						case "fail", "failed":
							test.Result = TestFailed
						case "skip", "skipped":
							test.Result = TestSkipped
						default:
							test.Result = TestError
						}
					}
					
					if output, ok := params["output"].(string); ok {
						test.Output = output
					}
					
					if err, ok := params["error"].(string); ok {
						test.Error = err
					}
				}
			}
		}
		
		// Add trace to all active tests
		for _, test := range testMap {
			if trace.Timestamp.After(test.StartTime) {
				test.Traces = append(test.Traces, trace)
			}
		}
	}
	
	// Convert map to slice
	for _, test := range testMap {
		tests = append(tests, *test)
	}
	
	return tests
}

// GroupTracesBySession groups traces into test sessions
func GroupTracesBySession(traces []MCPTrace) []TestSession {
	if len(traces) == 0 {
		return nil
	}
	
	// Simple grouping by time gaps
	var sessions []TestSession
	sessionGap := 5 * time.Minute
	
	currentSession := TestSession{
		ID:        fmt.Sprintf("session-%d", 1),
		StartTime: traces[0].Timestamp,
		Traces:    []MCPTrace{traces[0]},
	}
	
	for i := 1; i < len(traces); i++ {
		trace := traces[i]
		
		// Check if there's a significant time gap
		if trace.Timestamp.Sub(currentSession.Traces[len(currentSession.Traces)-1].Timestamp) > sessionGap {
			// Finalize current session
			currentSession.EndTime = currentSession.Traces[len(currentSession.Traces)-1].Timestamp
			currentSession.Tests = ExtractTestInfo(currentSession.Traces)
			sessions = append(sessions, currentSession)
			
			// Start new session
			currentSession = TestSession{
				ID:        fmt.Sprintf("session-%d", len(sessions)+1),
				StartTime: trace.Timestamp,
				Traces:    []MCPTrace{trace},
			}
		} else {
			currentSession.Traces = append(currentSession.Traces, trace)
		}
	}
	
	// Add final session
	if len(currentSession.Traces) > 0 {
		currentSession.EndTime = currentSession.Traces[len(currentSession.Traces)-1].Timestamp
		currentSession.Tests = ExtractTestInfo(currentSession.Traces)
		sessions = append(sessions, currentSession)
	}
	
	return sessions
}