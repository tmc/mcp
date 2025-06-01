package mcpcoverageviz_test

import (
	"strings"
	"testing"
	"time"
	
	mcpcoverageviz "github.com/tmc/mcp/exp/mcp-coverage-viz"
)

func TestCoverageVisualization(t *testing.T) {
	// Create sample MCP trace data
	traces := []mcpcoverageviz.MCPTrace{
		{
			ID:        "1",
			Type:      "request",
			Direction: "->",
			Timestamp: time.Now(),
			Method:    "testing/start",
			Params: map[string]interface{}{
				"name":    "TestExample",
				"package": "example",
			},
		},
		{
			ID:        "2",
			Type:      "response",
			Direction: "<-",
			Timestamp: time.Now().Add(100 * time.Millisecond),
			Result:    map[string]interface{}{"status": "ok"},
		},
		{
			ID:        "3",
			Type:      "request",
			Direction: "->",
			Timestamp: time.Now().Add(200 * time.Millisecond),
			Method:    "testing/end",
			Params: map[string]interface{}{
				"name":   "TestExample",
				"result": "passed",
			},
		},
	}
	
	// Extract test info
	tests := mcpcoverageviz.ExtractTestInfo(traces)
	if len(tests) != 1 {
		t.Fatalf("Expected 1 test, got %d", len(tests))
	}
	
	test := tests[0]
	if test.TestName != "TestExample" {
		t.Errorf("Expected test name 'TestExample', got %s", test.TestName)
	}
	if test.Result != mcpcoverageviz.TestPassed {
		t.Errorf("Expected test result 'passed', got %s", test.Result)
	}
}

func TestCoverageParser(t *testing.T) {
	// Sample coverage profile
	coverageData := `mode: set
example.go:10.2,12.3 1 1
example.go:14.2,16.3 2 0
example.go:18.2,20.3 1 1`
	
	reader := strings.NewReader(coverageData)
	integrator := mcpcoverageviz.NewCoverageIntegrator()
	
	if err := integrator.ParseCoverageProfile(reader); err != nil {
		t.Fatalf("Failed to parse coverage profile: %v", err)
	}
	
	// Verify coverage was parsed
	summary := integrator.CalculateSummary()
	if summary.TotalLines == 0 {
		t.Error("Expected coverage data to be parsed")
	}
}

func TestTraceParser(t *testing.T) {
	// Sample MCP trace
	traceData := `# mcptrace:v1
2024-01-01T12:00:00Z -> {"id":"1","method":"initialize","params":{}}
2024-01-01T12:00:01Z <- {"id":"1","result":{"version":"1.0"}}
2024-01-01T12:00:02Z -> {"method":"testing/start","params":{"name":"TestFoo"}}`
	
	reader := strings.NewReader(traceData)
	parser := mcpcoverageviz.NewTraceParser(reader)
	
	traces, err := parser.ParseMCPTrace()
	if err != nil {
		t.Fatalf("Failed to parse MCP trace: %v", err)
	}
	
	if len(traces) != 3 {
		t.Fatalf("Expected 3 traces, got %d", len(traces))
	}
	
	// Verify first trace
	if traces[0].Method != "initialize" {
		t.Errorf("Expected method 'initialize', got %s", traces[0].Method)
	}
	if traces[0].Direction != "->" {
		t.Errorf("Expected direction '->', got %s", traces[0].Direction)
	}
}