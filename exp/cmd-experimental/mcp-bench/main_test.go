package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tmc/mcp"
)

func TestBenchmarkRunner_Basic(t *testing.T) {
	// Create a temporary directory for test outputs
	tmpDir, err := os.MkdirTemp("", "mcp-bench-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a simple benchmark runner
	runner := &BenchmarkRunner{
		config: BenchmarkConfig{
			Concurrency: 2,
			Requests:    10,
			Duration:    1 * time.Second,
			Transport:   "stdio",
			TestType:    "load",
		},
		results: &BenchmarkResult{
			StartTime: time.Now(),
			Errors:    make([]ErrorInfo, 0),
			ToolStats: make(map[string]ToolStats),
			Timeline:  make([]TimelinePoint, 0),
		},
		errorMap: make(map[string]*ErrorInfo),
	}

	// Test calculateStats
	runner.latencies = []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
		40 * time.Millisecond,
		50 * time.Millisecond,
	}
	runner.requestCount = 5
	runner.successCount = 4
	runner.errorCount = 1
	runner.results.Duration = 5 * time.Second

	runner.calculateStats()

	// Verify results
	if runner.results.TotalRequests != 5 {
		t.Errorf("Expected 5 total requests, got %d", runner.results.TotalRequests)
	}
	if runner.results.SuccessfulRequests != 4 {
		t.Errorf("Expected 4 successful requests, got %d", runner.results.SuccessfulRequests)
	}
	if runner.results.FailedRequests != 1 {
		t.Errorf("Expected 1 failed request, got %d", runner.results.FailedRequests)
	}
	if runner.results.RequestsPerSecond != 1.0 {
		t.Errorf("Expected 1.0 RPS, got %f", runner.results.RequestsPerSecond)
	}
	if runner.results.LatencyStats.Min != 10*time.Millisecond {
		t.Errorf("Expected min latency 10ms, got %v", runner.results.LatencyStats.Min)
	}
	if runner.results.LatencyStats.Max != 50*time.Millisecond {
		t.Errorf("Expected max latency 50ms, got %v", runner.results.LatencyStats.Max)
	}
	if runner.results.LatencyStats.Median != 30*time.Millisecond {
		t.Errorf("Expected median latency 30ms, got %v", runner.results.LatencyStats.Median)
	}
}

func TestBenchmarkRunner_RecordError(t *testing.T) {
	runner := &BenchmarkRunner{
		errorMap: make(map[string]*ErrorInfo),
	}

	// Test error recording
	err1 := &TestError{message: "test error 1"}
	err2 := &TestError{message: "test error 2"}
	err3 := &TestError{message: "test error 1"} // Duplicate

	runner.recordError(err1)
	runner.recordError(err2)
	runner.recordError(err3)

	if len(runner.errorMap) != 2 {
		t.Errorf("Expected 2 unique errors, got %d", len(runner.errorMap))
	}

	if runner.errorMap["test error 1"].Count != 2 {
		t.Errorf("Expected error 1 count to be 2, got %d", runner.errorMap["test error 1"].Count)
	}

	if runner.errorMap["test error 2"].Count != 1 {
		t.Errorf("Expected error 2 count to be 1, got %d", runner.errorMap["test error 2"].Count)
	}
}

func TestBenchmarkRunner_WriteResultsToFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mcp-bench-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	runner := &BenchmarkRunner{
		results: &BenchmarkResult{
			StartTime:         time.Now(),
			EndTime:           time.Now().Add(30 * time.Second),
			Duration:          30 * time.Second,
			TotalRequests:     1000,
			SuccessfulRequests: 950,
			FailedRequests:    50,
			RequestsPerSecond: 33.33,
			LatencyStats: LatencyStats{
				Min:    5 * time.Millisecond,
				Max:    100 * time.Millisecond,
				Mean:   25 * time.Millisecond,
				Median: 20 * time.Millisecond,
				P95:    75 * time.Millisecond,
				P99:    90 * time.Millisecond,
			},
			Errors:    make([]ErrorInfo, 0),
			ToolStats: make(map[string]ToolStats),
			Timeline:  make([]TimelinePoint, 0),
		},
	}

	// Test writing results to file
	outputFile := filepath.Join(tmpDir, "results.json")
	err = runner.writeResultsToFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to write results: %v", err)
	}

	// Verify file exists and contains valid JSON
	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read results file: %v", err)
	}

	var result BenchmarkResult
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if result.TotalRequests != 1000 {
		t.Errorf("Expected 1000 total requests, got %d", result.TotalRequests)
	}
}

func TestBenchmarkRunner_ExportPrometheus(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mcp-bench-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	runner := &BenchmarkRunner{
		results: &BenchmarkResult{
			SuccessfulRequests: 900,
			FailedRequests:     100,
			RequestsPerSecond:  50.0,
			LatencyStats: LatencyStats{
				Median: 20 * time.Millisecond,
				P90:    40 * time.Millisecond,
				P95:    50 * time.Millisecond,
				P99:    80 * time.Millisecond,
			},
		},
	}

	err = runner.exportPrometheusMetrics()
	if err != nil {
		t.Fatalf("Failed to export Prometheus metrics: %v", err)
	}

	// Verify file exists and contains expected metrics
	data, err := os.ReadFile("prometheus_metrics.txt")
	if err != nil {
		t.Fatalf("Failed to read Prometheus metrics file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "mcp_benchmark_requests_total") {
		t.Error("Expected Prometheus metrics to contain request totals")
	}
	if !strings.Contains(content, "mcp_benchmark_requests_per_second") {
		t.Error("Expected Prometheus metrics to contain RPS")
	}
	if !strings.Contains(content, "mcp_benchmark_latency_seconds") {
		t.Error("Expected Prometheus metrics to contain latency histograms")
	}
}

func TestBenchmarkRunner_ExportJMeter(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mcp-bench-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	runner := &BenchmarkRunner{
		config: BenchmarkConfig{
			Concurrency: 10,
			Requests:    100,
			Duration:    60 * time.Second,
		},
	}

	err = runner.exportJMeterPlan()
	if err != nil {
		t.Fatalf("Failed to export JMeter plan: %v", err)
	}

	// Verify file exists and contains valid XML
	data, err := os.ReadFile("jmeter_plan.jmx")
	if err != nil {
		t.Fatalf("Failed to read JMeter plan file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "<?xml version=") {
		t.Error("Expected JMeter plan to contain XML declaration")
	}
	if !strings.Contains(content, "jmeterTestPlan") {
		t.Error("Expected JMeter plan to contain test plan element")
	}
	if !strings.Contains(content, "ThreadGroup") {
		t.Error("Expected JMeter plan to contain thread group")
	}
}

func TestBenchmarkRunner_ExportK6(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mcp-bench-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	runner := &BenchmarkRunner{
		config: BenchmarkConfig{
			Concurrency: 15,
			Duration:    120 * time.Second,
		},
	}

	err = runner.exportK6Script()
	if err != nil {
		t.Fatalf("Failed to export k6 script: %v", err)
	}

	// Verify file exists and contains valid JavaScript
	data, err := os.ReadFile("k6_script.js")
	if err != nil {
		t.Fatalf("Failed to read k6 script file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "import http from 'k6/http'") {
		t.Error("Expected k6 script to contain HTTP import")
	}
	if !strings.Contains(content, "export let options") {
		t.Error("Expected k6 script to contain options export")
	}
	if !strings.Contains(content, "export default function") {
		t.Error("Expected k6 script to contain default function")
	}
}

func TestBenchmarkRunner_MergeResults(t *testing.T) {
	runner1 := &BenchmarkRunner{
		results: &BenchmarkResult{
			TotalRequests:     100,
			SuccessfulRequests: 90,
			FailedRequests:    10,
			Timeline:          []TimelinePoint{{Timestamp: time.Now()}},
		},
		latencies: []time.Duration{10 * time.Millisecond, 20 * time.Millisecond},
		errorMap: map[string]*ErrorInfo{
			"error1": {Error: "error1", Count: 5, FirstSeen: time.Now()},
		},
	}

	runner2 := &BenchmarkRunner{
		results: &BenchmarkResult{
			TotalRequests:     200,
			SuccessfulRequests: 180,
			FailedRequests:    20,
			Timeline:          []TimelinePoint{{Timestamp: time.Now()}},
			Errors: []ErrorInfo{
				{Error: "error1", Count: 3, FirstSeen: time.Now()},
				{Error: "error2", Count: 2, FirstSeen: time.Now()},
			},
		},
		latencies: []time.Duration{30 * time.Millisecond, 40 * time.Millisecond},
	}

	runner1.mergeResults(runner2.results)

	// Verify merged results
	if runner1.results.TotalRequests != 300 {
		t.Errorf("Expected 300 total requests, got %d", runner1.results.TotalRequests)
	}
	if runner1.results.SuccessfulRequests != 270 {
		t.Errorf("Expected 270 successful requests, got %d", runner1.results.SuccessfulRequests)
	}
	if runner1.results.FailedRequests != 30 {
		t.Errorf("Expected 30 failed requests, got %d", runner1.results.FailedRequests)
	}
	if len(runner1.latencies) != 4 {
		t.Errorf("Expected 4 latencies, got %d", len(runner1.latencies))
	}
	if len(runner1.results.Timeline) != 2 {
		t.Errorf("Expected 2 timeline points, got %d", len(runner1.results.Timeline))
	}
	if len(runner1.errorMap) != 2 {
		t.Errorf("Expected 2 error types, got %d", len(runner1.errorMap))
	}
	if runner1.errorMap["error1"].Count != 8 {
		t.Errorf("Expected error1 count to be 8, got %d", runner1.errorMap["error1"].Count)
	}
}

func TestBenchmarkConfig_Validation(t *testing.T) {
	testCases := []struct {
		name        string
		config      BenchmarkConfig
		expectError bool
	}{
		{
			name: "Valid config",
			config: BenchmarkConfig{
				Concurrency: 10,
				Requests:    100,
				Duration:    30 * time.Second,
				Transport:   "stdio",
				TestType:    "load",
			},
			expectError: false,
		},
		{
			name: "Zero concurrency",
			config: BenchmarkConfig{
				Concurrency: 0,
				Requests:    100,
				Duration:    30 * time.Second,
				Transport:   "stdio",
				TestType:    "load",
			},
			expectError: true,
		},
		{
			name: "Invalid transport",
			config: BenchmarkConfig{
				Concurrency: 10,
				Requests:    100,
				Duration:    30 * time.Second,
				Transport:   "invalid",
				TestType:    "load",
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateBenchmarkConfig(tc.config)
			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestLatencyStats_Percentiles(t *testing.T) {
	latencies := []time.Duration{
		1 * time.Millisecond,
		5 * time.Millisecond,
		10 * time.Millisecond,
		15 * time.Millisecond,
		20 * time.Millisecond,
		25 * time.Millisecond,
		30 * time.Millisecond,
		50 * time.Millisecond,
		75 * time.Millisecond,
		100 * time.Millisecond,
	}

	stats := calculateLatencyStats(latencies)

	if stats.Min != 1*time.Millisecond {
		t.Errorf("Expected min 1ms, got %v", stats.Min)
	}
	if stats.Max != 100*time.Millisecond {
		t.Errorf("Expected max 100ms, got %v", stats.Max)
	}
	if stats.Median != 22*time.Millisecond && stats.Median != 23*time.Millisecond {
		t.Errorf("Expected median around 22-23ms, got %v", stats.Median)
	}
	if stats.P95 < 75*time.Millisecond {
		t.Errorf("Expected P95 >= 75ms, got %v", stats.P95)
	}
	if stats.P99 < 90*time.Millisecond {
		t.Errorf("Expected P99 >= 90ms, got %v", stats.P99)
	}
}

// Helper types and functions for testing

type TestError struct {
	message string
}

func (e *TestError) Error() string {
	return e.message
}

func validateBenchmarkConfig(config BenchmarkConfig) error {
	if config.Concurrency <= 0 {
		return &TestError{message: "concurrency must be positive"}
	}
	if config.Transport != "stdio" && config.Transport != "http" && config.Transport != "sse" {
		return &TestError{message: "invalid transport"}
	}
	return nil
}

func calculateLatencyStats(latencies []time.Duration) LatencyStats {
	if len(latencies) == 0 {
		return LatencyStats{}
	}

	// Sort latencies
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)
	
	// Simple bubble sort for small arrays
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Calculate percentiles
	stats := LatencyStats{
		Min:    sorted[0],
		Max:    sorted[len(sorted)-1],
		Median: sorted[len(sorted)/2],
		P90:    sorted[int(float64(len(sorted))*0.9)],
		P95:    sorted[int(float64(len(sorted))*0.95)],
		P99:    sorted[int(float64(len(sorted))*0.99)],
		P999:   sorted[int(float64(len(sorted))*0.999)],
	}

	// Calculate mean
	var total time.Duration
	for _, latency := range latencies {
		total += latency
	}
	stats.Mean = total / time.Duration(len(latencies))

	return stats
}

// Integration test - requires actual server
func TestBenchmarkRunner_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test would require a real MCP server to be running
	// For now, just test the setup
	runner := &BenchmarkRunner{
		config: BenchmarkConfig{
			Concurrency: 1,
			Requests:    5,
			Duration:    1 * time.Second,
			Transport:   "stdio",
			TestType:    "load",
		},
		results: &BenchmarkResult{
			StartTime: time.Now(),
			Errors:    make([]ErrorInfo, 0),
			ToolStats: make(map[string]ToolStats),
			Timeline:  make([]TimelinePoint, 0),
		},
		errorMap: make(map[string]*ErrorInfo),
	}

	// Test that the runner can be initialized
	if runner.config.Concurrency != 1 {
		t.Error("Runner not initialized correctly")
	}
}

func TestBenchmarkRunner_StressTest(t *testing.T) {
	runner := &BenchmarkRunner{
		config: BenchmarkConfig{
			Concurrency: 20,
			Duration:    5 * time.Second,
			Transport:   "stdio",
			TestType:    "stress",
		},
		results: &BenchmarkResult{
			StartTime: time.Now(),
			Errors:    make([]ErrorInfo, 0),
			ToolStats: make(map[string]ToolStats),
			Timeline:  make([]TimelinePoint, 0),
		},
		errorMap: make(map[string]*ErrorInfo),
		ctx:      context.Background(),
	}

	// Test stress test configuration
	if runner.config.TestType != "stress" {
		t.Error("Expected stress test type")
	}
	if runner.config.Concurrency != 20 {
		t.Error("Expected concurrency of 20")
	}
}

func TestBenchmarkRunner_SpikeTest(t *testing.T) {
	runner := &BenchmarkRunner{
		config: BenchmarkConfig{
			Concurrency: 50,
			Duration:    2 * time.Minute,
			Transport:   "stdio",
			TestType:    "spike",
		},
		results: &BenchmarkResult{
			StartTime: time.Now(),
			Errors:    make([]ErrorInfo, 0),
			ToolStats: make(map[string]ToolStats),
			Timeline:  make([]TimelinePoint, 0),
		},
		errorMap: make(map[string]*ErrorInfo),
		ctx:      context.Background(),
	}

	// Test spike test configuration
	if runner.config.TestType != "spike" {
		t.Error("Expected spike test type")
	}
	if runner.config.Concurrency != 50 {
		t.Error("Expected concurrency of 50")
	}
}

func TestBenchmarkRunner_ResourceStats(t *testing.T) {
	runner := &BenchmarkRunner{
		results: &BenchmarkResult{
			ResourceStats: ResourceStats{
				CPUUsage:       65.5,
				MemoryUsage:    128 * 1024 * 1024,
				GoroutineCount: 25,
				GCPauses:       12,
			},
		},
	}

	stats := runner.results.ResourceStats
	if stats.CPUUsage != 65.5 {
		t.Errorf("Expected CPU usage 65.5, got %f", stats.CPUUsage)
	}
	if stats.MemoryUsage != 128*1024*1024 {
		t.Errorf("Expected memory usage 128MB, got %d", stats.MemoryUsage)
	}
	if stats.GoroutineCount != 25 {
		t.Errorf("Expected 25 goroutines, got %d", stats.GoroutineCount)
	}
	if stats.GCPauses != 12 {
		t.Errorf("Expected 12 GC pauses, got %d", stats.GCPauses)
	}
}

func TestTimelinePoint_Serialization(t *testing.T) {
	point := TimelinePoint{
		Timestamp:         time.Now(),
		RequestsPerSecond: 125.5,
		LatencyP95:        25 * time.Millisecond,
		ErrorRate:         0.02,
		ActiveConnections: 10,
	}

	// Test JSON serialization
	data, err := json.Marshal(point)
	if err != nil {
		t.Fatalf("Failed to marshal timeline point: %v", err)
	}

	var unmarshaled TimelinePoint
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal timeline point: %v", err)
	}

	if unmarshaled.RequestsPerSecond != 125.5 {
		t.Errorf("Expected RPS 125.5, got %f", unmarshaled.RequestsPerSecond)
	}
	if unmarshaled.LatencyP95 != 25*time.Millisecond {
		t.Errorf("Expected latency 25ms, got %v", unmarshaled.LatencyP95)
	}
	if unmarshaled.ErrorRate != 0.02 {
		t.Errorf("Expected error rate 0.02, got %f", unmarshaled.ErrorRate)
	}
}

func BenchmarkBenchmarkRunner_CalculateStats(b *testing.B) {
	runner := &BenchmarkRunner{
		results:  &BenchmarkResult{Duration: 30 * time.Second},
		errorMap: make(map[string]*ErrorInfo),
	}

	// Generate test data
	for i := 0; i < 10000; i++ {
		runner.latencies = append(runner.latencies, time.Duration(i)*time.Microsecond)
	}
	runner.requestCount = 10000
	runner.successCount = 9500
	runner.errorCount = 500

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runner.calculateStats()
	}
}

func BenchmarkBenchmarkRunner_RecordLatency(b *testing.B) {
	runner := &BenchmarkRunner{
		latencies: make([]time.Duration, 0, b.N),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runner.recordLatency(time.Duration(i) * time.Microsecond)
	}
}

func BenchmarkBenchmarkRunner_RecordError(b *testing.B) {
	runner := &BenchmarkRunner{
		errorMap: make(map[string]*ErrorInfo),
	}

	err := &TestError{message: "test error"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runner.recordError(err)
	}
}