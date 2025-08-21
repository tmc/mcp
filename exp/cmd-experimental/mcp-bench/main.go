// mcp-bench: Comprehensive performance testing tool for MCP servers
//
// This tool provides comprehensive performance testing capabilities including:
// - Load testing with configurable concurrent clients
// - Stress testing with automatic scaling
// - Latency analysis with percentile reporting
// - Throughput measurement and monitoring
// - Resource utilization tracking
// - Real-time performance visualization
// - Export to standard formats (JMeter, k6, Prometheus)
//
// Usage:
//
//	mcp-bench [flags] <server-command>
//
// Examples:
//
//	mcp-bench -c 10 -r 100 -d 30s go run ./examples/servers/mcp-time-server
//	mcp-bench -load-test -stress-test -output results.json go run ./server
//	mcp-bench -profile -export-prometheus go run ./server
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/tmc/mcp"
)

var (
	// Core benchmarking flags
	concurrency = flag.Int("c", 1, "Number of concurrent clients")
	requests    = flag.Int("r", 100, "Total number of requests per client")
	duration    = flag.Duration("d", 30*time.Second, "Duration of the test")
	warmup      = flag.Duration("warmup", 5*time.Second, "Warmup duration")
	cooldown    = flag.Duration("cooldown", 2*time.Second, "Cooldown duration")

	// Test type flags
	loadTest      = flag.Bool("load-test", false, "Run load test (default behavior)")
	stressTest    = flag.Bool("stress-test", false, "Run stress test with automatic scaling")
	spikeTest     = flag.Bool("spike-test", false, "Run spike test with sudden load increases")
	enduranceTest = flag.Bool("endurance-test", false, "Run endurance test for extended periods")

	// Tool selection flags
	tool     = flag.String("tool", "", "Specific tool to test (if empty, tests all available tools)")
	toolArgs = flag.String("tool-args", "{}", "JSON arguments to pass to the tool")

	// Output and reporting flags
	output   = flag.String("output", "", "Output file for results (JSON format)")
	verbose  = flag.Bool("v", false, "Verbose output")
	quiet    = flag.Bool("q", false, "Quiet mode (minimal output)")
	realtime = flag.Bool("realtime", false, "Enable real-time monitoring dashboard")

	// Performance analysis flags
	profile    = flag.Bool("profile", false, "Enable CPU and memory profiling")
	profileDir = flag.String("profile-dir", "./profiles", "Directory for profile outputs")
	memProfile = flag.Bool("mem-profile", false, "Enable memory profiling")
	cpuProfile = flag.Bool("cpu-profile", false, "Enable CPU profiling")
	traceFile  = flag.String("trace", "", "Enable execution tracing to file")

	// Export flags
	exportPrometheus = flag.Bool("export-prometheus", false, "Export metrics in Prometheus format")
	exportJMeter     = flag.Bool("export-jmeter", false, "Export test plan in JMeter format")
	exportK6         = flag.Bool("export-k6", false, "Export test script in k6 format")

	// Rate limiting and throttling
	rateLimit = flag.Float64("rate-limit", 0, "Rate limit in requests per second (0 = no limit)")
	throttle  = flag.Duration("throttle", 0, "Throttle delay between requests")

	// Monitoring and metrics
	monitorInterval = flag.Duration("monitor-interval", 1*time.Second, "Monitoring interval")
	metricsPort     = flag.Int("metrics-port", 8080, "Port for metrics HTTP server")

	// Connection and transport options
	transport = flag.String("transport", "stdio", "Transport type (stdio, http, sse)")
	httpURL   = flag.String("http-url", "", "HTTP URL for HTTP transport")
	sseURL    = flag.String("sse-url", "", "SSE URL for SSE transport")
	timeout   = flag.Duration("timeout", 10*time.Second, "Request timeout")

	// Distributed testing
	distributed     = flag.Bool("distributed", false, "Enable distributed load generation")
	workerNodes     = flag.String("worker-nodes", "", "Comma-separated list of worker node addresses")
	coordinatorMode = flag.Bool("coordinator", false, "Run as coordinator for distributed testing")
	workerMode      = flag.Bool("worker", false, "Run as worker for distributed testing")
)

// BenchmarkResult represents the results of a benchmark run
type BenchmarkResult struct {
	// Test configuration
	Config BenchmarkConfig `json:"config"`

	// Timing information
	StartTime    time.Time     `json:"startTime"`
	EndTime      time.Time     `json:"endTime"`
	Duration     time.Duration `json:"duration"`
	WarmupTime   time.Duration `json:"warmupTime"`
	CooldownTime time.Duration `json:"cooldownTime"`

	// Request statistics
	TotalRequests      int64   `json:"totalRequests"`
	SuccessfulRequests int64   `json:"successfulRequests"`
	FailedRequests     int64   `json:"failedRequests"`
	RequestsPerSecond  float64 `json:"requestsPerSecond"`

	// Latency statistics
	LatencyStats LatencyStats `json:"latencyStats"`

	// Error information
	Errors []ErrorInfo `json:"errors"`

	// Resource utilization
	ResourceStats ResourceStats `json:"resourceStats"`

	// Per-tool breakdown
	ToolStats map[string]ToolStats `json:"toolStats"`

	// Timeline data for visualization
	Timeline []TimelinePoint `json:"timeline"`
}

type BenchmarkConfig struct {
	Concurrency int           `json:"concurrency"`
	Requests    int           `json:"requests"`
	Duration    time.Duration `json:"duration"`
	Tool        string        `json:"tool"`
	ToolArgs    string        `json:"toolArgs"`
	Transport   string        `json:"transport"`
	TestType    string        `json:"testType"`
	RateLimit   float64       `json:"rateLimit"`
}

type LatencyStats struct {
	Min    time.Duration `json:"min"`
	Max    time.Duration `json:"max"`
	Mean   time.Duration `json:"mean"`
	Median time.Duration `json:"median"`
	P90    time.Duration `json:"p90"`
	P95    time.Duration `json:"p95"`
	P99    time.Duration `json:"p99"`
	P999   time.Duration `json:"p999"`
	StdDev time.Duration `json:"stdDev"`
}

type ErrorInfo struct {
	Error     string    `json:"error"`
	Count     int64     `json:"count"`
	FirstSeen time.Time `json:"firstSeen"`
	LastSeen  time.Time `json:"lastSeen"`
}

type ResourceStats struct {
	CPUUsage       float64 `json:"cpuUsage"`
	MemoryUsage    int64   `json:"memoryUsage"`
	GoroutineCount int     `json:"goroutineCount"`
	GCPauses       int64   `json:"gcPauses"`
}

type ToolStats struct {
	Name             string        `json:"name"`
	Requests         int64         `json:"requests"`
	Success          int64         `json:"success"`
	Errors           int64         `json:"errors"`
	AvgLatency       time.Duration `json:"avgLatency"`
	MinLatency       time.Duration `json:"minLatency"`
	MaxLatency       time.Duration `json:"maxLatency"`
	ThroughputPerSec float64       `json:"throughputPerSec"`
}

type TimelinePoint struct {
	Timestamp         time.Time     `json:"timestamp"`
	RequestsPerSecond float64       `json:"requestsPerSecond"`
	LatencyP95        time.Duration `json:"latencyP95"`
	ErrorRate         float64       `json:"errorRate"`
	ActiveConnections int           `json:"activeConnections"`
}

// BenchmarkRunner manages benchmark execution
type BenchmarkRunner struct {
	config  BenchmarkConfig
	results *BenchmarkResult
	clients []*mcp.Client
	tools   []mcp.Tool

	// Synchronization
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Metrics collection
	requestCount int64
	successCount int64
	errorCount   int64
	latencies    []time.Duration
	latencyMutex sync.RWMutex
	errorMap     map[string]*ErrorInfo
	errorMutex   sync.RWMutex

	// Timeline tracking
	timeline      []TimelinePoint
	timelineMutex sync.RWMutex

	// Rate limiting
	rateLimiter *time.Ticker

	// Profiling
	cpuFile   *os.File
	memFile   *os.File
	traceFile *os.File
}

func main() {
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <server-command>\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	serverCmd := flag.Args()

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signalChan
		log.Println("Shutting down gracefully...")
		cancel()
	}()

	// Create and configure benchmark runner
	runner := &BenchmarkRunner{
		config: BenchmarkConfig{
			Concurrency: *concurrency,
			Requests:    *requests,
			Duration:    *duration,
			Tool:        *tool,
			ToolArgs:    *toolArgs,
			Transport:   *transport,
			RateLimit:   *rateLimit,
		},
		results: &BenchmarkResult{
			StartTime: time.Now(),
			Errors:    make([]ErrorInfo, 0),
			ToolStats: make(map[string]ToolStats),
			Timeline:  make([]TimelinePoint, 0),
		},
		ctx:      ctx,
		cancel:   cancel,
		errorMap: make(map[string]*ErrorInfo),
	}

	// Determine test type
	runner.config.TestType = "load"
	if *stressTest {
		runner.config.TestType = "stress"
	} else if *spikeTest {
		runner.config.TestType = "spike"
	} else if *enduranceTest {
		runner.config.TestType = "endurance"
	}

	// Set up profiling if requested
	if *profile || *cpuProfile || *memProfile {
		if err := runner.setupProfiling(); err != nil {
			log.Fatalf("Failed to setup profiling: %v", err)
		}
		defer runner.cleanupProfiling()
	}

	// Run the benchmark
	if err := runner.run(serverCmd); err != nil {
		log.Fatalf("Benchmark failed: %v", err)
	}

	// Output results
	if err := runner.outputResults(); err != nil {
		log.Fatalf("Failed to output results: %v", err)
	}
}

func (r *BenchmarkRunner) run(serverCmd []string) error {
	if !*quiet {
		fmt.Printf("Starting %s test with %d clients for %v\n",
			r.config.TestType, r.config.Concurrency, r.config.Duration)
	}

	// Start the server and create clients
	if err := r.setupClients(serverCmd); err != nil {
		return fmt.Errorf("failed to setup clients: %v", err)
	}
	defer r.cleanupClients()

	// Get available tools
	if err := r.discoverTools(); err != nil {
		return fmt.Errorf("failed to discover tools: %v", err)
	}

	// Set up rate limiter if needed
	if r.config.RateLimit > 0 {
		interval := time.Duration(float64(time.Second) / r.config.RateLimit)
		r.rateLimiter = time.NewTicker(interval)
		defer r.rateLimiter.Stop()
	}

	// Start monitoring
	if *realtime {
		go r.startRealtimeMonitoring()
	}

	// Run warmup
	if *warmup > 0 {
		if !*quiet {
			fmt.Printf("Running warmup for %v\n", *warmup)
		}
		r.runWarmup()
	}

	// Run main test based on type
	switch r.config.TestType {
	case "load":
		return r.runLoadTest()
	case "stress":
		return r.runStressTest()
	case "spike":
		return r.runSpikeTest()
	case "endurance":
		return r.runEnduranceTest()
	default:
		return fmt.Errorf("unknown test type: %s", r.config.TestType)
	}
}

func (r *BenchmarkRunner) setupClients(serverCmd []string) error {
	r.clients = make([]*mcp.Client, r.config.Concurrency)

	for i := 0; i < r.config.Concurrency; i++ {
		var transport mcp.Transport
		var err error

		switch r.config.Transport {
		case "stdio":
			transport = mcp.NewStdioTransport(serverCmd...)
		case "http":
			if *httpURL == "" {
				return fmt.Errorf("http-url required for HTTP transport")
			}
			transport = mcp.NewHTTPTransport(*httpURL)
		case "sse":
			if *sseURL == "" {
				return fmt.Errorf("sse-url required for SSE transport")
			}
			transport = mcp.NewSSETransport(*sseURL)
		default:
			return fmt.Errorf("unsupported transport: %s", r.config.Transport)
		}

		client, err := mcp.NewClient(transport)
		if err != nil {
			return fmt.Errorf("failed to create client %d: %v", i, err)
		}

		// Initialize client
		initReq := mcp.InitializeRequest{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "mcp-bench",
				Version: "1.0.0",
			},
			Capabilities: mcp.ClientCapabilities{
				Roots: &mcp.RootsCapability{
					ListChanged: true,
				},
				Sampling: &mcp.SamplingCapability{},
			},
		}

		_, err = client.Initialize(r.ctx, initReq)
		if err != nil {
			return fmt.Errorf("failed to initialize client %d: %v", i, err)
		}

		r.clients[i] = client
	}

	return nil
}

func (r *BenchmarkRunner) cleanupClients() {
	for i, client := range r.clients {
		if client != nil {
			if err := client.Close(); err != nil {
				log.Printf("Error closing client %d: %v", i, err)
			}
		}
	}
}

func (r *BenchmarkRunner) discoverTools() error {
	if len(r.clients) == 0 {
		return fmt.Errorf("no clients available")
	}

	// Use the first client to discover tools
	client := r.clients[0]

	listReq := mcp.ListToolsRequest{}
	listResp, err := client.ListTools(r.ctx, listReq)
	if err != nil {
		return fmt.Errorf("failed to list tools: %v", err)
	}

	r.tools = listResp.Tools

	// Filter tools if specific tool requested
	if r.config.Tool != "" {
		var filteredTools []mcp.Tool
		for _, tool := range r.tools {
			if tool.Name == r.config.Tool {
				filteredTools = append(filteredTools, tool)
				break
			}
		}
		if len(filteredTools) == 0 {
			return fmt.Errorf("tool %s not found", r.config.Tool)
		}
		r.tools = filteredTools
	}

	if len(r.tools) == 0 {
		return fmt.Errorf("no tools available")
	}

	if !*quiet {
		fmt.Printf("Found %d tools to test\n", len(r.tools))
		for _, tool := range r.tools {
			fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
		}
	}

	return nil
}

func (r *BenchmarkRunner) runLoadTest() error {
	r.results.Config = r.config
	testStart := time.Now()

	// Start timeline monitoring
	go r.monitorTimeline()

	// Launch worker goroutines
	for i := 0; i < r.config.Concurrency; i++ {
		r.wg.Add(1)
		go r.worker(i)
	}

	// Wait for duration or completion
	select {
	case <-time.After(r.config.Duration):
		r.cancel()
	case <-r.ctx.Done():
		// Cancelled by signal
	}

	// Wait for all workers to complete
	r.wg.Wait()

	r.results.EndTime = time.Now()
	r.results.Duration = r.results.EndTime.Sub(testStart)
	r.calculateStats()

	return nil
}

func (r *BenchmarkRunner) runStressTest() error {
	// Start with minimum load and gradually increase
	startConcurrency := 1
	maxConcurrency := r.config.Concurrency
	stepSize := maxConcurrency / 10
	if stepSize < 1 {
		stepSize = 1
	}

	for currentConcurrency := startConcurrency; currentConcurrency <= maxConcurrency; currentConcurrency += stepSize {
		if !*quiet {
			fmt.Printf("Stress test: %d concurrent clients\n", currentConcurrency)
		}

		// Run sub-test with current concurrency
		subRunner := &BenchmarkRunner{
			config: r.config,
			results: &BenchmarkResult{
				StartTime: time.Now(),
				Errors:    make([]ErrorInfo, 0),
				ToolStats: make(map[string]ToolStats),
				Timeline:  make([]TimelinePoint, 0),
			},
			ctx:      r.ctx,
			cancel:   r.cancel,
			tools:    r.tools,
			errorMap: make(map[string]*ErrorInfo),
		}

		subRunner.config.Concurrency = currentConcurrency
		subRunner.config.Duration = time.Minute // Fixed duration for each stress level

		// Create clients for this stress level
		if err := subRunner.setupClients([]string{}); err != nil {
			return fmt.Errorf("failed to setup clients for stress level %d: %v", currentConcurrency, err)
		}

		// Run the test
		if err := subRunner.runLoadTest(); err != nil {
			subRunner.cleanupClients()
			return fmt.Errorf("stress test failed at concurrency %d: %v", currentConcurrency, err)
		}

		subRunner.cleanupClients()

		// Check if we've reached the breaking point
		if subRunner.results.LatencyStats.P95 > time.Second*5 ||
			float64(subRunner.results.FailedRequests)/float64(subRunner.results.TotalRequests) > 0.1 {
			if !*quiet {
				fmt.Printf("Breaking point reached at %d concurrent clients\n", currentConcurrency)
			}
			break
		}

		// Merge results
		r.mergeResults(subRunner.results)
	}

	return nil
}

func (r *BenchmarkRunner) runSpikeTest() error {
	// Spike test: sudden increases in load
	baseLoad := r.config.Concurrency / 4
	if baseLoad < 1 {
		baseLoad = 1
	}

	spikeLoad := r.config.Concurrency

	phases := []struct {
		concurrency int
		duration    time.Duration
	}{
		{baseLoad, time.Minute},       // Base load
		{spikeLoad, 30 * time.Second}, // Spike
		{baseLoad, time.Minute},       // Recovery
		{spikeLoad, 30 * time.Second}, // Another spike
		{baseLoad, time.Minute},       // Final recovery
	}

	for i, phase := range phases {
		if !*quiet {
			fmt.Printf("Spike test phase %d: %d clients for %v\n", i+1, phase.concurrency, phase.duration)
		}

		// Run phase
		subRunner := &BenchmarkRunner{
			config: r.config,
			results: &BenchmarkResult{
				StartTime: time.Now(),
				Errors:    make([]ErrorInfo, 0),
				ToolStats: make(map[string]ToolStats),
				Timeline:  make([]TimelinePoint, 0),
			},
			ctx:      r.ctx,
			cancel:   r.cancel,
			tools:    r.tools,
			errorMap: make(map[string]*ErrorInfo),
		}

		subRunner.config.Concurrency = phase.concurrency
		subRunner.config.Duration = phase.duration

		if err := subRunner.setupClients([]string{}); err != nil {
			return fmt.Errorf("failed to setup clients for spike phase %d: %v", i+1, err)
		}

		if err := subRunner.runLoadTest(); err != nil {
			subRunner.cleanupClients()
			return fmt.Errorf("spike test failed at phase %d: %v", i+1, err)
		}

		subRunner.cleanupClients()
		r.mergeResults(subRunner.results)
	}

	return nil
}

func (r *BenchmarkRunner) runEnduranceTest() error {
	// Endurance test: steady load over extended period
	if !*quiet {
		fmt.Printf("Endurance test: %d clients for %v\n", r.config.Concurrency, r.config.Duration)
	}

	return r.runLoadTest()
}

func (r *BenchmarkRunner) worker(workerID int) {
	defer r.wg.Done()

	client := r.clients[workerID]
	requestCount := 0

	for {
		select {
		case <-r.ctx.Done():
			return
		default:
			// Rate limiting
			if r.rateLimiter != nil {
				<-r.rateLimiter.C
			}

			// Throttling
			if *throttle > 0 {
				time.Sleep(*throttle)
			}

			// Select tool to test
			tool := r.tools[requestCount%len(r.tools)]

			// Execute request
			startTime := time.Now()
			err := r.executeToolRequest(client, tool)
			endTime := time.Now()

			latency := endTime.Sub(startTime)

			// Record metrics
			atomic.AddInt64(&r.requestCount, 1)

			if err != nil {
				atomic.AddInt64(&r.errorCount, 1)
				r.recordError(err)
			} else {
				atomic.AddInt64(&r.successCount, 1)
			}

			r.recordLatency(latency)

			requestCount++

			// Check if we've reached the request limit
			if r.config.Requests > 0 && requestCount >= r.config.Requests {
				return
			}
		}
	}
}

func (r *BenchmarkRunner) executeToolRequest(client *mcp.Client, tool mcp.Tool) error {
	// Parse tool arguments
	var args json.RawMessage
	if r.config.ToolArgs != "" && r.config.ToolArgs != "{}" {
		args = json.RawMessage(r.config.ToolArgs)
	}

	req := mcp.CallToolRequest{
		Name:      tool.Name,
		Arguments: args,
	}

	ctx, cancel := context.WithTimeout(r.ctx, *timeout)
	defer cancel()

	_, err := client.CallTool(ctx, req)
	return err
}

func (r *BenchmarkRunner) recordLatency(latency time.Duration) {
	r.latencyMutex.Lock()
	r.latencies = append(r.latencies, latency)
	r.latencyMutex.Unlock()
}

func (r *BenchmarkRunner) recordError(err error) {
	r.errorMutex.Lock()
	defer r.errorMutex.Unlock()

	errStr := err.Error()
	if info, exists := r.errorMap[errStr]; exists {
		info.Count++
		info.LastSeen = time.Now()
	} else {
		r.errorMap[errStr] = &ErrorInfo{
			Error:     errStr,
			Count:     1,
			FirstSeen: time.Now(),
			LastSeen:  time.Now(),
		}
	}
}

func (r *BenchmarkRunner) calculateStats() {
	r.results.TotalRequests = r.requestCount
	r.results.SuccessfulRequests = r.successCount
	r.results.FailedRequests = r.errorCount

	if r.results.Duration > 0 {
		r.results.RequestsPerSecond = float64(r.results.TotalRequests) / r.results.Duration.Seconds()
	}

	// Calculate latency statistics
	r.latencyMutex.RLock()
	latencies := make([]time.Duration, len(r.latencies))
	copy(latencies, r.latencies)
	r.latencyMutex.RUnlock()

	if len(latencies) > 0 {
		sort.Slice(latencies, func(i, j int) bool {
			return latencies[i] < latencies[j]
		})

		r.results.LatencyStats = LatencyStats{
			Min:    latencies[0],
			Max:    latencies[len(latencies)-1],
			Median: latencies[len(latencies)/2],
			P90:    latencies[int(float64(len(latencies))*0.9)],
			P95:    latencies[int(float64(len(latencies))*0.95)],
			P99:    latencies[int(float64(len(latencies))*0.99)],
			P999:   latencies[int(float64(len(latencies))*0.999)],
		}

		// Calculate mean
		var total time.Duration
		for _, latency := range latencies {
			total += latency
		}
		r.results.LatencyStats.Mean = total / time.Duration(len(latencies))

		// Calculate standard deviation
		var sumSquares float64
		meanFloat := float64(r.results.LatencyStats.Mean)
		for _, latency := range latencies {
			diff := float64(latency) - meanFloat
			sumSquares += diff * diff
		}
		variance := sumSquares / float64(len(latencies))
		r.results.LatencyStats.StdDev = time.Duration(variance)
	}

	// Convert error map to slice
	r.errorMutex.RLock()
	for _, info := range r.errorMap {
		r.results.Errors = append(r.results.Errors, *info)
	}
	r.errorMutex.RUnlock()

	// Resource statistics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	r.results.ResourceStats = ResourceStats{
		MemoryUsage:    int64(memStats.Alloc),
		GoroutineCount: runtime.NumGoroutine(),
		GCPauses:       int64(memStats.NumGC),
	}
}

func (r *BenchmarkRunner) monitorTimeline() {
	ticker := time.NewTicker(*monitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			point := TimelinePoint{
				Timestamp:         time.Now(),
				ActiveConnections: r.config.Concurrency,
			}

			// Calculate current RPS
			if r.results.Duration > 0 {
				point.RequestsPerSecond = float64(atomic.LoadInt64(&r.requestCount)) /
					time.Since(r.results.StartTime).Seconds()
			}

			// Calculate current error rate
			totalReqs := atomic.LoadInt64(&r.requestCount)
			if totalReqs > 0 {
				point.ErrorRate = float64(atomic.LoadInt64(&r.errorCount)) / float64(totalReqs)
			}

			// Get current P95 latency
			r.latencyMutex.RLock()
			if len(r.latencies) > 0 {
				latencies := make([]time.Duration, len(r.latencies))
				copy(latencies, r.latencies)
				sort.Slice(latencies, func(i, j int) bool {
					return latencies[i] < latencies[j]
				})
				point.LatencyP95 = latencies[int(float64(len(latencies))*0.95)]
			}
			r.latencyMutex.RUnlock()

			r.timelineMutex.Lock()
			r.results.Timeline = append(r.results.Timeline, point)
			r.timelineMutex.Unlock()

		case <-r.ctx.Done():
			return
		}
	}
}

func (r *BenchmarkRunner) runWarmup() {
	// Simple warmup: make a few requests to establish connections
	warmupClient := r.clients[0]

	for i := 0; i < 5; i++ {
		if len(r.tools) > 0 {
			tool := r.tools[i%len(r.tools)]
			r.executeToolRequest(warmupClient, tool)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (r *BenchmarkRunner) mergeResults(other *BenchmarkResult) {
	r.results.TotalRequests += other.TotalRequests
	r.results.SuccessfulRequests += other.SuccessfulRequests
	r.results.FailedRequests += other.FailedRequests

	// Merge latencies
	r.latencyMutex.Lock()
	other.latencyMutex.RLock()
	r.latencies = append(r.latencies, other.latencies...)
	other.latencyMutex.RUnlock()
	r.latencyMutex.Unlock()

	// Merge errors
	r.errorMutex.Lock()
	for _, err := range other.Errors {
		if existing, exists := r.errorMap[err.Error]; exists {
			existing.Count += err.Count
			if err.FirstSeen.Before(existing.FirstSeen) {
				existing.FirstSeen = err.FirstSeen
			}
			if err.LastSeen.After(existing.LastSeen) {
				existing.LastSeen = err.LastSeen
			}
		} else {
			r.errorMap[err.Error] = &ErrorInfo{
				Error:     err.Error,
				Count:     err.Count,
				FirstSeen: err.FirstSeen,
				LastSeen:  err.LastSeen,
			}
		}
	}
	r.errorMutex.Unlock()

	// Merge timeline
	r.timelineMutex.Lock()
	r.results.Timeline = append(r.results.Timeline, other.Timeline...)
	r.timelineMutex.Unlock()
}

func (r *BenchmarkRunner) startRealtimeMonitoring() {
	// Simple real-time monitoring output
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			totalReqs := atomic.LoadInt64(&r.requestCount)
			successReqs := atomic.LoadInt64(&r.successCount)
			errorReqs := atomic.LoadInt64(&r.errorCount)

			elapsed := time.Since(r.results.StartTime).Seconds()
			rps := float64(totalReqs) / elapsed
			errorRate := float64(errorReqs) / float64(totalReqs) * 100

			fmt.Printf("[%s] Requests: %d, Success: %d, Errors: %d, RPS: %.2f, Error Rate: %.2f%%\n",
				time.Now().Format("15:04:05"), totalReqs, successReqs, errorReqs, rps, errorRate)

		case <-r.ctx.Done():
			return
		}
	}
}

func (r *BenchmarkRunner) setupProfiling() error {
	// Create profile directory
	if err := os.MkdirAll(*profileDir, 0755); err != nil {
		return fmt.Errorf("failed to create profile directory: %v", err)
	}

	// CPU profiling
	if *profile || *cpuProfile {
		cpuFile, err := os.Create(fmt.Sprintf("%s/cpu.prof", *profileDir))
		if err != nil {
			return fmt.Errorf("failed to create CPU profile: %v", err)
		}
		r.cpuFile = cpuFile

		if err := pprof.StartCPUProfile(cpuFile); err != nil {
			return fmt.Errorf("failed to start CPU profile: %v", err)
		}
	}

	// Memory profiling
	if *profile || *memProfile {
		memFile, err := os.Create(fmt.Sprintf("%s/mem.prof", *profileDir))
		if err != nil {
			return fmt.Errorf("failed to create memory profile: %v", err)
		}
		r.memFile = memFile
	}

	// Execution tracing
	if *traceFile != "" {
		traceFile, err := os.Create(*traceFile)
		if err != nil {
			return fmt.Errorf("failed to create trace file: %v", err)
		}
		r.traceFile = traceFile
	}

	return nil
}

func (r *BenchmarkRunner) cleanupProfiling() {
	if r.cpuFile != nil {
		pprof.StopCPUProfile()
		r.cpuFile.Close()
	}

	if r.memFile != nil {
		runtime.GC()
		if err := pprof.WriteHeapProfile(r.memFile); err != nil {
			log.Printf("Failed to write heap profile: %v", err)
		}
		r.memFile.Close()
	}

	if r.traceFile != nil {
		r.traceFile.Close()
	}
}

func (r *BenchmarkRunner) outputResults() error {
	// Print summary to console
	if !*quiet {
		r.printSummary()
	}

	// Write detailed results to file
	if *output != "" {
		if err := r.writeResultsToFile(*output); err != nil {
			return fmt.Errorf("failed to write results to file: %v", err)
		}
	}

	// Export to various formats
	if *exportPrometheus {
		if err := r.exportPrometheusMetrics(); err != nil {
			return fmt.Errorf("failed to export Prometheus metrics: %v", err)
		}
	}

	if *exportJMeter {
		if err := r.exportJMeterPlan(); err != nil {
			return fmt.Errorf("failed to export JMeter plan: %v", err)
		}
	}

	if *exportK6 {
		if err := r.exportK6Script(); err != nil {
			return fmt.Errorf("failed to export k6 script: %v", err)
		}
	}

	return nil
}

func (r *BenchmarkRunner) printSummary() {
	fmt.Printf("\n=== Benchmark Results ===\n")
	fmt.Printf("Test Type: %s\n", r.config.TestType)
	fmt.Printf("Duration: %v\n", r.results.Duration)
	fmt.Printf("Concurrency: %d\n", r.config.Concurrency)
	fmt.Printf("\nRequests:\n")
	fmt.Printf("  Total: %d\n", r.results.TotalRequests)
	fmt.Printf("  Successful: %d\n", r.results.SuccessfulRequests)
	fmt.Printf("  Failed: %d\n", r.results.FailedRequests)
	fmt.Printf("  Requests/sec: %.2f\n", r.results.RequestsPerSecond)

	if r.results.TotalRequests > 0 {
		fmt.Printf("  Error Rate: %.2f%%\n", float64(r.results.FailedRequests)/float64(r.results.TotalRequests)*100)
	}

	fmt.Printf("\nLatency:\n")
	fmt.Printf("  Min: %v\n", r.results.LatencyStats.Min)
	fmt.Printf("  Max: %v\n", r.results.LatencyStats.Max)
	fmt.Printf("  Mean: %v\n", r.results.LatencyStats.Mean)
	fmt.Printf("  Median: %v\n", r.results.LatencyStats.Median)
	fmt.Printf("  P90: %v\n", r.results.LatencyStats.P90)
	fmt.Printf("  P95: %v\n", r.results.LatencyStats.P95)
	fmt.Printf("  P99: %v\n", r.results.LatencyStats.P99)
	fmt.Printf("  P99.9: %v\n", r.results.LatencyStats.P999)

	if len(r.results.Errors) > 0 {
		fmt.Printf("\nTop Errors:\n")
		for i, err := range r.results.Errors {
			if i >= 5 {
				break
			}
			fmt.Printf("  %d: %s (count: %d)\n", i+1, err.Error, err.Count)
		}
	}

	fmt.Printf("\nResource Usage:\n")
	fmt.Printf("  Memory: %d bytes\n", r.results.ResourceStats.MemoryUsage)
	fmt.Printf("  Goroutines: %d\n", r.results.ResourceStats.GoroutineCount)
	fmt.Printf("  GC Pauses: %d\n", r.results.ResourceStats.GCPauses)
}

func (r *BenchmarkRunner) writeResultsToFile(filename string) error {
	data, err := json.MarshalIndent(r.results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %v", err)
	}

	return os.WriteFile(filename, data, 0644)
}

func (r *BenchmarkRunner) exportPrometheusMetrics() error {
	filename := "prometheus_metrics.txt"
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	timestamp := time.Now().Unix()

	fmt.Fprintf(file, "# HELP mcp_benchmark_requests_total Total number of requests\n")
	fmt.Fprintf(file, "# TYPE mcp_benchmark_requests_total counter\n")
	fmt.Fprintf(file, "mcp_benchmark_requests_total{status=\"success\"} %d %d\n",
		r.results.SuccessfulRequests, timestamp)
	fmt.Fprintf(file, "mcp_benchmark_requests_total{status=\"error\"} %d %d\n",
		r.results.FailedRequests, timestamp)

	fmt.Fprintf(file, "# HELP mcp_benchmark_requests_per_second Requests per second\n")
	fmt.Fprintf(file, "# TYPE mcp_benchmark_requests_per_second gauge\n")
	fmt.Fprintf(file, "mcp_benchmark_requests_per_second %.2f %d\n",
		r.results.RequestsPerSecond, timestamp)

	fmt.Fprintf(file, "# HELP mcp_benchmark_latency_seconds Request latency in seconds\n")
	fmt.Fprintf(file, "# TYPE mcp_benchmark_latency_seconds histogram\n")
	fmt.Fprintf(file, "mcp_benchmark_latency_seconds{quantile=\"0.5\"} %.6f %d\n",
		r.results.LatencyStats.Median.Seconds(), timestamp)
	fmt.Fprintf(file, "mcp_benchmark_latency_seconds{quantile=\"0.9\"} %.6f %d\n",
		r.results.LatencyStats.P90.Seconds(), timestamp)
	fmt.Fprintf(file, "mcp_benchmark_latency_seconds{quantile=\"0.95\"} %.6f %d\n",
		r.results.LatencyStats.P95.Seconds(), timestamp)
	fmt.Fprintf(file, "mcp_benchmark_latency_seconds{quantile=\"0.99\"} %.6f %d\n",
		r.results.LatencyStats.P99.Seconds(), timestamp)

	return nil
}

func (r *BenchmarkRunner) exportJMeterPlan() error {
	filename := "jmeter_plan.jmx"
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Basic JMeter XML structure
	fmt.Fprintf(file, `<?xml version="1.0" encoding="UTF-8"?>
<jmeterTestPlan version="1.2" properties="5.0" jmeter="5.4">
  <hashTree>
    <TestPlan guiclass="TestPlanGui" testclass="TestPlan" testname="MCP Benchmark" enabled="true">
      <stringProp name="TestPlan.comments">Generated by mcp-bench</stringProp>
      <boolProp name="TestPlan.functional_mode">false</boolProp>
      <boolProp name="TestPlan.serialize_threadgroups">false</boolProp>
      <elementProp name="TestPlan.arguments" elementType="Arguments" guiclass="ArgumentsPanel" testclass="Arguments" testname="User Defined Variables" enabled="true">
        <collectionProp name="Arguments.arguments"/>
      </elementProp>
      <stringProp name="TestPlan.user_define_classpath"></stringProp>
    </TestPlan>
    <hashTree>
      <ThreadGroup guiclass="ThreadGroupGui" testclass="ThreadGroup" testname="Thread Group" enabled="true">
        <stringProp name="ThreadGroup.on_sample_error">continue</stringProp>
        <elementProp name="ThreadGroup.main_controller" elementType="LoopController" guiclass="LoopControllerGui" testclass="LoopController" testname="Loop Controller" enabled="true">
          <boolProp name="LoopController.continue_forever">false</boolProp>
          <stringProp name="LoopController.loops">%d</stringProp>
        </elementProp>
        <stringProp name="ThreadGroup.num_threads">%d</stringProp>
        <stringProp name="ThreadGroup.ramp_time">10</stringProp>
        <boolProp name="ThreadGroup.scheduler">false</boolProp>
        <stringProp name="ThreadGroup.duration">%d</stringProp>
        <stringProp name="ThreadGroup.delay"></stringProp>
      </ThreadGroup>
    </hashTree>
  </hashTree>
</jmeterTestPlan>`, r.config.Requests, r.config.Concurrency, int(r.config.Duration.Seconds()))

	return nil
}

func (r *BenchmarkRunner) exportK6Script() error {
	filename := "k6_script.js"
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintf(file, `import http from 'k6/http';
import { check } from 'k6';

export let options = {
  stages: [
    { duration: '10s', target: %d },
    { duration: '%ds', target: %d },
    { duration: '10s', target: 0 },
  ],
};

export default function() {
  // Generated by mcp-bench
  let response = http.get('http://localhost:8080/mcp');
  check(response, {
    'status is 200': (r) => r.status === 200,
  });
}`, r.config.Concurrency, int(r.config.Duration.Seconds()), r.config.Concurrency)

	return nil
}

// Helper functions for different transports
func NewStdioTransport(args ...string) mcp.Transport {
	return mcp.TransportFunc(func(ctx context.Context) (mcp.ReadWriteCloser, error) {
		return mcp.NewStdioTransport(args...)
	})
}

func NewHTTPTransport(url string) mcp.Transport {
	return mcp.TransportFunc(func(ctx context.Context) (mcp.ReadWriteCloser, error) {
		return mcp.NewHTTPTransport(url)
	})
}

func NewSSETransport(url string) mcp.Transport {
	return mcp.TransportFunc(func(ctx context.Context) (mcp.ReadWriteCloser, error) {
		return mcp.NewSSETransport(url)
	})
}
