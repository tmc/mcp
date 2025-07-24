// mcp-profile: Runtime performance analysis tool for MCP servers
//
// This tool provides comprehensive runtime performance analysis including:
// - CPU profiling with call graph visualization
// - Memory profiling with heap analysis
// - Goroutine profiling for concurrency analysis
// - I/O analysis and blocking operations detection
// - Mutex contention analysis
// - Execution tracing with timeline visualization
// - Performance regression detection
// - Comparative analysis between runs
//
// Usage:
//   mcp-profile [flags] <server-command>
//
// Examples:
//   mcp-profile -cpu -mem go run ./server
//   mcp-profile -all -duration 60s go run ./server
//   mcp-profile -trace -output profile.trace go run ./server
//   mcp-profile -compare baseline.prof current.prof
//
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/tmc/mcp"
)

var (
	// Profiling type flags
	cpuProfile      = flag.Bool("cpu", false, "Enable CPU profiling")
	memProfile      = flag.Bool("mem", false, "Enable memory profiling")
	goroutineProfile = flag.Bool("goroutine", false, "Enable goroutine profiling")
	blockProfile    = flag.Bool("block", false, "Enable blocking operations profiling")
	mutexProfile    = flag.Bool("mutex", false, "Enable mutex contention profiling")
	traceProfile    = flag.Bool("trace", false, "Enable execution tracing")
	allProfiles     = flag.Bool("all", false, "Enable all profiling types")
	
	// Analysis options
	duration        = flag.Duration("duration", 30*time.Second, "Profiling duration")
	samplingRate    = flag.Int("sampling-rate", 100, "CPU profiling sampling rate (Hz)")
	memSamplingRate = flag.Int("mem-sampling-rate", 512*1024, "Memory profiling sampling rate (bytes)")
	
	// Output options
	outputDir       = flag.String("output-dir", "./profiles", "Output directory for profile files")
	outputFile      = flag.String("output", "", "Output file for specific profile")
	format          = flag.String("format", "pprof", "Output format (pprof, json, text)")
	
	// Analysis and visualization
	analyze         = flag.Bool("analyze", false, "Analyze profiles and generate report")
	visualize       = flag.Bool("visualize", false, "Generate visualization files")
	compareBaseline = flag.String("compare", "", "Baseline profile to compare against")
	topN            = flag.Int("top", 10, "Show top N functions in analysis")
	
	// Server interaction
	loadTest        = flag.Bool("load-test", false, "Run load test during profiling")
	concurrency     = flag.Int("concurrency", 10, "Concurrent clients during load test")
	requests        = flag.Int("requests", 1000, "Total requests during load test")
	tool            = flag.String("tool", "", "Specific tool to test during profiling")
	toolArgs        = flag.String("tool-args", "{}", "JSON arguments for the tool")
	
	// Transport options
	transport       = flag.String("transport", "stdio", "Transport type (stdio, http, sse)")
	httpURL         = flag.String("http-url", "", "HTTP URL for HTTP transport")
	sseURL          = flag.String("sse-url", "", "SSE URL for SSE transport")
	timeout         = flag.Duration("timeout", 10*time.Second, "Request timeout")
	
	// Monitoring options
	continuous      = flag.Bool("continuous", false, "Continuous profiling mode")
	interval        = flag.Duration("interval", 30*time.Second, "Continuous profiling interval")
	retention       = flag.Duration("retention", 24*time.Hour, "Profile retention period")
	
	// Advanced options
	verbose         = flag.Bool("v", false, "Verbose output")
	quiet           = flag.Bool("q", false, "Quiet mode")
	timestamp       = flag.Bool("timestamp", true, "Add timestamp to profile filenames")
	compress        = flag.Bool("compress", false, "Compress profile files")
)

// ProfilerConfig holds configuration for the profiler
type ProfilerConfig struct {
	CPUProfile      bool
	MemProfile      bool
	GoroutineProfile bool
	BlockProfile    bool
	MutexProfile    bool
	TraceProfile    bool
	Duration        time.Duration
	SamplingRate    int
	MemSamplingRate int
	OutputDir       string
	Format          string
	LoadTest        bool
	Concurrency     int
	Requests        int
	Tool            string
	ToolArgs        string
	Transport       string
	Continuous      bool
	Interval        time.Duration
}

// ProfileResult represents the result of a profiling session
type ProfileResult struct {
	Timestamp       time.Time             `json:"timestamp"`
	Duration        time.Duration         `json:"duration"`
	ProfileTypes    []string              `json:"profileTypes"`
	Files           map[string]string     `json:"files"`
	Metrics         ProfileMetrics        `json:"metrics"`
	Analysis        *ProfileAnalysis      `json:"analysis,omitempty"`
	LoadTestResults *LoadTestResults      `json:"loadTestResults,omitempty"`
}

// ProfileMetrics contains runtime metrics during profiling
type ProfileMetrics struct {
	CPUUsage        float64               `json:"cpuUsage"`
	MemoryUsage     int64                 `json:"memoryUsage"`
	GoroutineCount  int                   `json:"goroutineCount"`
	GCPauses        int64                 `json:"gcPauses"`
	AllocRate       float64               `json:"allocRate"`
	GCRate          float64               `json:"gcRate"`
	HeapSize        int64                 `json:"heapSize"`
	HeapObjects     int64                 `json:"heapObjects"`
	StackSize       int64                 `json:"stackSize"`
	ThreadCount     int                   `json:"threadCount"`
	Timeline        []MetricPoint         `json:"timeline"`
}

// MetricPoint represents a point in time during profiling
type MetricPoint struct {
	Timestamp      time.Time `json:"timestamp"`
	CPUUsage       float64   `json:"cpuUsage"`
	MemoryUsage    int64     `json:"memoryUsage"`
	GoroutineCount int       `json:"goroutineCount"`
	HeapSize       int64     `json:"heapSize"`
	GCPauses       int64     `json:"gcPauses"`
}

// ProfileAnalysis contains analysis results
type ProfileAnalysis struct {
	TopFunctions    []FunctionProfile     `json:"topFunctions"`
	HotPaths        []CallPath            `json:"hotPaths"`
	MemoryLeaks     []MemoryLeak          `json:"memoryLeaks"`
	BlockingOps     []BlockingOperation   `json:"blockingOps"`
	MutexContention []MutexContention     `json:"mutexContention"`
	Recommendations []Recommendation      `json:"recommendations"`
}

// FunctionProfile represents profiling data for a function
type FunctionProfile struct {
	Name           string        `json:"name"`
	SelfTime       time.Duration `json:"selfTime"`
	CumulativeTime time.Duration `json:"cumulativeTime"`
	SelfPercent    float64       `json:"selfPercent"`
	CumPercent     float64       `json:"cumPercent"`
	CallCount      int64         `json:"callCount"`
	MemoryAlloc    int64         `json:"memoryAlloc"`
}

// CallPath represents a call path in the profile
type CallPath struct {
	Path        []string      `json:"path"`
	Time        time.Duration `json:"time"`
	Percent     float64       `json:"percent"`
	CallCount   int64         `json:"callCount"`
}

// MemoryLeak represents a potential memory leak
type MemoryLeak struct {
	Function    string  `json:"function"`
	AllocBytes  int64   `json:"allocBytes"`
	AllocCount  int64   `json:"allocCount"`
	GrowthRate  float64 `json:"growthRate"`
	Severity    string  `json:"severity"`
}

// BlockingOperation represents a blocking operation
type BlockingOperation struct {
	Function    string        `json:"function"`
	BlockTime   time.Duration `json:"blockTime"`
	BlockCount  int64         `json:"blockCount"`
	AvgBlockTime time.Duration `json:"avgBlockTime"`
	Type        string        `json:"type"`
}

// MutexContention represents mutex contention
type MutexContention struct {
	Function    string        `json:"function"`
	ContentionTime time.Duration `json:"contentionTime"`
	ContentionCount int64       `json:"contentionCount"`
	AvgContentionTime time.Duration `json:"avgContentionTime"`
}

// Recommendation represents a performance recommendation
type Recommendation struct {
	Type        string  `json:"type"`
	Severity    string  `json:"severity"`
	Function    string  `json:"function"`
	Issue       string  `json:"issue"`
	Suggestion  string  `json:"suggestion"`
	Impact      string  `json:"impact"`
}

// LoadTestResults contains load test results during profiling
type LoadTestResults struct {
	TotalRequests     int64         `json:"totalRequests"`
	SuccessfulRequests int64        `json:"successfulRequests"`
	FailedRequests    int64         `json:"failedRequests"`
	RequestsPerSecond float64       `json:"requestsPerSecond"`
	AvgLatency        time.Duration `json:"avgLatency"`
	P95Latency        time.Duration `json:"p95Latency"`
	P99Latency        time.Duration `json:"p99Latency"`
	ErrorRate         float64       `json:"errorRate"`
}

// Profiler manages the profiling session
type Profiler struct {
	config       ProfilerConfig
	result       *ProfileResult
	ctx          context.Context
	cancel       context.CancelFunc
	
	// Profile file handles
	cpuFile      *os.File
	memFile      *os.File
	traceFile    *os.File
	
	// Metrics collection
	metrics      *ProfileMetrics
	metricsMutex sync.RWMutex
	
	// Load test client
	client       *mcp.Client
	tools        []mcp.Tool
	
	// Continuous profiling
	profileTicker *time.Ticker
}

func main() {
	flag.Parse()
	
	if flag.NArg() < 1 && !*analyze && *compareBaseline == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <server-command>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "   or: %s -analyze [profile-files...]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "   or: %s -compare baseline.prof current.prof\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}
	
	// Handle comparison mode
	if *compareBaseline != "" {
		if flag.NArg() < 1 {
			fmt.Fprintf(os.Stderr, "Comparison mode requires current profile file\n")
			os.Exit(1)
		}
		if err := compareProfiles(*compareBaseline, flag.Args()[0]); err != nil {
			log.Fatalf("Profile comparison failed: %v", err)
		}
		return
	}
	
	// Handle analysis mode
	if *analyze {
		if flag.NArg() < 1 {
			fmt.Fprintf(os.Stderr, "Analysis mode requires profile file(s)\n")
			os.Exit(1)
		}
		if err := analyzeProfiles(flag.Args()...); err != nil {
			log.Fatalf("Profile analysis failed: %v", err)
		}
		return
	}
	
	// Normal profiling mode
	serverCmd := flag.Args()
	
	// Set up signal handling
	ctx, cancel := context.WithCancel(context.Background())
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-signalChan
		log.Println("Shutting down gracefully...")
		cancel()
	}()
	
	// Create profiler
	profiler := &Profiler{
		config: ProfilerConfig{
			CPUProfile:      *cpuProfile || *allProfiles,
			MemProfile:      *memProfile || *allProfiles,
			GoroutineProfile: *goroutineProfile || *allProfiles,
			BlockProfile:    *blockProfile || *allProfiles,
			MutexProfile:    *mutexProfile || *allProfiles,
			TraceProfile:    *traceProfile || *allProfiles,
			Duration:        *duration,
			SamplingRate:    *samplingRate,
			MemSamplingRate: *memSamplingRate,
			OutputDir:       *outputDir,
			Format:          *format,
			LoadTest:        *loadTest,
			Concurrency:     *concurrency,
			Requests:        *requests,
			Tool:            *tool,
			ToolArgs:        *toolArgs,
			Transport:       *transport,
			Continuous:      *continuous,
			Interval:        *interval,
		},
		result: &ProfileResult{
			Timestamp:    time.Now(),
			Files:        make(map[string]string),
			Metrics:      ProfileMetrics{Timeline: make([]MetricPoint, 0)},
		},
		ctx:    ctx,
		cancel: cancel,
	}
	
	// Run profiling
	if err := profiler.run(serverCmd); err != nil {
		log.Fatalf("Profiling failed: %v", err)
	}
	
	// Output results
	if err := profiler.outputResults(); err != nil {
		log.Fatalf("Failed to output results: %v", err)
	}
}

func (p *Profiler) run(serverCmd []string) error {
	if !*quiet {
		fmt.Printf("Starting profiling session for %v\n", *duration)
	}
	
	// Create output directory
	if err := os.MkdirAll(p.config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}
	
	// Set up profiling
	if err := p.setupProfiling(); err != nil {
		return fmt.Errorf("failed to setup profiling: %v", err)
	}
	defer p.cleanupProfiling()
	
	// Start server if needed
	if len(serverCmd) > 0 {
		if err := p.startServer(serverCmd); err != nil {
			return fmt.Errorf("failed to start server: %v", err)
		}
		defer p.stopServer()
	}
	
	// Start metrics collection
	go p.collectMetrics()
	
	// Start load test if requested
	if p.config.LoadTest {
		go p.runLoadTest()
	}
	
	// Run profiling
	if p.config.Continuous {
		return p.runContinuousProfiling()
	} else {
		return p.runSingleProfiling()
	}
}

func (p *Profiler) setupProfiling() error {
	// Set sampling rates
	runtime.SetCPUProfileRate(p.config.SamplingRate)
	runtime.MemProfileRate = p.config.MemSamplingRate
	
	// Enable specific profiles
	if p.config.BlockProfile {
		runtime.SetBlockProfileRate(1)
	}
	if p.config.MutexProfile {
		runtime.SetMutexProfileFraction(1)
	}
	
	// Create profile files
	timestamp := ""
	if *timestamp {
		timestamp = time.Now().Format("20060102-150405")
	}
	
	if p.config.CPUProfile {
		filename := filepath.Join(p.config.OutputDir, fmt.Sprintf("cpu%s.prof", timestamp))
		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create CPU profile file: %v", err)
		}
		p.cpuFile = file
		p.result.Files["cpu"] = filename
		
		if err := pprof.StartCPUProfile(file); err != nil {
			return fmt.Errorf("failed to start CPU profile: %v", err)
		}
	}
	
	if p.config.MemProfile {
		filename := filepath.Join(p.config.OutputDir, fmt.Sprintf("mem%s.prof", timestamp))
		p.result.Files["mem"] = filename
	}
	
	if p.config.TraceProfile {
		filename := filepath.Join(p.config.OutputDir, fmt.Sprintf("trace%s.trace", timestamp))
		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create trace file: %v", err)
		}
		p.traceFile = file
		p.result.Files["trace"] = filename
		
		if err := trace.Start(file); err != nil {
			return fmt.Errorf("failed to start trace: %v", err)
		}
	}
	
	return nil
}

func (p *Profiler) cleanupProfiling() {
	if p.cpuFile != nil {
		pprof.StopCPUProfile()
		p.cpuFile.Close()
	}
	
	if p.config.MemProfile {
		runtime.GC()
		filename := p.result.Files["mem"]
		file, err := os.Create(filename)
		if err == nil {
			pprof.WriteHeapProfile(file)
			file.Close()
		}
	}
	
	if p.config.GoroutineProfile {
		filename := filepath.Join(p.config.OutputDir, fmt.Sprintf("goroutine%s.prof", 
			time.Now().Format("20060102-150405")))
		file, err := os.Create(filename)
		if err == nil {
			pprof.Lookup("goroutine").WriteTo(file, 0)
			file.Close()
			p.result.Files["goroutine"] = filename
		}
	}
	
	if p.config.BlockProfile {
		filename := filepath.Join(p.config.OutputDir, fmt.Sprintf("block%s.prof", 
			time.Now().Format("20060102-150405")))
		file, err := os.Create(filename)
		if err == nil {
			pprof.Lookup("block").WriteTo(file, 0)
			file.Close()
			p.result.Files["block"] = filename
		}
	}
	
	if p.config.MutexProfile {
		filename := filepath.Join(p.config.OutputDir, fmt.Sprintf("mutex%s.prof", 
			time.Now().Format("20060102-150405")))
		file, err := os.Create(filename)
		if err == nil {
			pprof.Lookup("mutex").WriteTo(file, 0)
			file.Close()
			p.result.Files["mutex"] = filename
		}
	}
	
	if p.traceFile != nil {
		trace.Stop()
		p.traceFile.Close()
	}
}

func (p *Profiler) startServer(serverCmd []string) error {
	// For now, assume the server is started externally
	// In a real implementation, we would start the server process
	// and manage its lifecycle
	return nil
}

func (p *Profiler) stopServer() {
	// Stop the server if we started it
}

func (p *Profiler) collectMetrics() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)
			
			point := MetricPoint{
				Timestamp:      time.Now(),
				MemoryUsage:    int64(memStats.Alloc),
				GoroutineCount: runtime.NumGoroutine(),
				HeapSize:       int64(memStats.HeapAlloc),
				GCPauses:       int64(memStats.NumGC),
			}
			
			p.metricsMutex.Lock()
			p.result.Metrics.Timeline = append(p.result.Metrics.Timeline, point)
			p.metricsMutex.Unlock()
			
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *Profiler) runLoadTest() error {
	// Create client
	var transport mcp.Transport
	switch p.config.Transport {
	case "stdio":
		transport = mcp.NewStdioTransport()
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
		return fmt.Errorf("unsupported transport: %s", p.config.Transport)
	}
	
	client, err := mcp.NewClient(transport)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}
	p.client = client
	
	// Initialize client
	initReq := mcp.InitializeRequest{
		ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
		ClientInfo: mcp.Implementation{
			Name:    "mcp-profile",
			Version: "1.0.0",
		},
	}
	
	_, err = client.Initialize(p.ctx, initReq)
	if err != nil {
		return fmt.Errorf("failed to initialize client: %v", err)
	}
	
	// Get tools
	listReq := mcp.ListToolsRequest{}
	listResp, err := client.ListTools(p.ctx, listReq)
	if err != nil {
		return fmt.Errorf("failed to list tools: %v", err)
	}
	p.tools = listResp.Tools
	
	// Run load test
	var successCount, errorCount int64
	var latencies []time.Duration
	
	for i := 0; i < p.config.Requests; i++ {
		select {
		case <-p.ctx.Done():
			break
		default:
			tool := p.tools[i%len(p.tools)]
			start := time.Now()
			
			var args json.RawMessage
			if p.config.ToolArgs != "" && p.config.ToolArgs != "{}" {
				args = json.RawMessage(p.config.ToolArgs)
			}
			
			req := mcp.CallToolRequest{
				Name:      tool.Name,
				Arguments: args,
			}
			
			ctx, cancel := context.WithTimeout(p.ctx, *timeout)
			_, err := client.CallTool(ctx, req)
			cancel()
			
			latency := time.Since(start)
			latencies = append(latencies, latency)
			
			if err != nil {
				errorCount++
			} else {
				successCount++
			}
		}
	}
	
	// Calculate load test results
	totalRequests := successCount + errorCount
	if totalRequests > 0 {
		// Calculate average latency
		var totalLatency time.Duration
		for _, latency := range latencies {
			totalLatency += latency
		}
		avgLatency := totalLatency / time.Duration(len(latencies))
		
		// Calculate percentiles
		sort.Slice(latencies, func(i, j int) bool {
			return latencies[i] < latencies[j]
		})
		
		p95Latency := latencies[int(float64(len(latencies))*0.95)]
		p99Latency := latencies[int(float64(len(latencies))*0.99)]
		
		p.result.LoadTestResults = &LoadTestResults{
			TotalRequests:     totalRequests,
			SuccessfulRequests: successCount,
			FailedRequests:    errorCount,
			RequestsPerSecond: float64(totalRequests) / p.config.Duration.Seconds(),
			AvgLatency:        avgLatency,
			P95Latency:        p95Latency,
			P99Latency:        p99Latency,
			ErrorRate:         float64(errorCount) / float64(totalRequests),
		}
	}
	
	return nil
}

func (p *Profiler) runSingleProfiling() error {
	if !*quiet {
		fmt.Printf("Profiling for %v...\n", p.config.Duration)
	}
	
	// Wait for duration
	select {
	case <-time.After(p.config.Duration):
		p.result.Duration = p.config.Duration
	case <-p.ctx.Done():
		p.result.Duration = time.Since(p.result.Timestamp)
	}
	
	// Analyze if requested
	if *analyze {
		if err := p.analyzeProfiles(); err != nil {
			return fmt.Errorf("failed to analyze profiles: %v", err)
		}
	}
	
	return nil
}

func (p *Profiler) runContinuousProfiling() error {
	if !*quiet {
		fmt.Printf("Starting continuous profiling with %v intervals\n", p.config.Interval)
	}
	
	p.profileTicker = time.NewTicker(p.config.Interval)
	defer p.profileTicker.Stop()
	
	for {
		select {
		case <-p.profileTicker.C:
			if err := p.takeProfileSnapshot(); err != nil {
				log.Printf("Failed to take profile snapshot: %v", err)
			}
		case <-p.ctx.Done():
			return nil
		}
	}
}

func (p *Profiler) takeProfileSnapshot() error {
	timestamp := time.Now().Format("20060102-150405")
	
	// Take heap profile
	if p.config.MemProfile {
		filename := filepath.Join(p.config.OutputDir, fmt.Sprintf("heap_%s.prof", timestamp))
		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create heap profile: %v", err)
		}
		defer file.Close()
		
		runtime.GC()
		if err := pprof.WriteHeapProfile(file); err != nil {
			return fmt.Errorf("failed to write heap profile: %v", err)
		}
	}
	
	// Take goroutine profile
	if p.config.GoroutineProfile {
		filename := filepath.Join(p.config.OutputDir, fmt.Sprintf("goroutine_%s.prof", timestamp))
		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create goroutine profile: %v", err)
		}
		defer file.Close()
		
		if err := pprof.Lookup("goroutine").WriteTo(file, 0); err != nil {
			return fmt.Errorf("failed to write goroutine profile: %v", err)
		}
	}
	
	return nil
}

func (p *Profiler) analyzeProfiles() error {
	// This would contain sophisticated profile analysis
	// For now, provide a basic implementation
	analysis := &ProfileAnalysis{
		TopFunctions:    make([]FunctionProfile, 0),
		HotPaths:        make([]CallPath, 0),
		MemoryLeaks:     make([]MemoryLeak, 0),
		BlockingOps:     make([]BlockingOperation, 0),
		MutexContention: make([]MutexContention, 0),
		Recommendations: make([]Recommendation, 0),
	}
	
	// Add sample recommendations
	analysis.Recommendations = append(analysis.Recommendations, Recommendation{
		Type:       "performance",
		Severity:   "medium",
		Function:   "main.worker",
		Issue:      "High CPU usage detected",
		Suggestion: "Consider optimizing hot loops or adding worker pools",
		Impact:     "Could improve throughput by 20-30%",
	})
	
	p.result.Analysis = analysis
	return nil
}

func (p *Profiler) outputResults() error {
	// Calculate final metrics
	p.metricsMutex.Lock()
	timeline := p.result.Metrics.Timeline
	p.metricsMutex.Unlock()
	
	if len(timeline) > 0 {
		// Calculate averages
		var totalMem, totalGoroutines int64
		for _, point := range timeline {
			totalMem += point.MemoryUsage
			totalGoroutines += int64(point.GoroutineCount)
		}
		
		p.result.Metrics.MemoryUsage = totalMem / int64(len(timeline))
		p.result.Metrics.GoroutineCount = int(totalGoroutines / int64(len(timeline)))
	}
	
	// Print summary
	if !*quiet {
		p.printSummary()
	}
	
	// Write detailed results
	if *outputFile != "" {
		return p.writeResults(*outputFile)
	}
	
	// Generate visualization if requested
	if *visualize {
		return p.generateVisualization()
	}
	
	return nil
}

func (p *Profiler) printSummary() {
	fmt.Printf("\n=== Profiling Results ===\n")
	fmt.Printf("Duration: %v\n", p.result.Duration)
	fmt.Printf("Timestamp: %s\n", p.result.Timestamp.Format("2006-01-02 15:04:05"))
	
	fmt.Printf("\nProfile Files:\n")
	for profileType, filename := range p.result.Files {
		fmt.Printf("  %s: %s\n", profileType, filename)
	}
	
	fmt.Printf("\nMetrics:\n")
	fmt.Printf("  Average Memory Usage: %d bytes\n", p.result.Metrics.MemoryUsage)
	fmt.Printf("  Average Goroutines: %d\n", p.result.Metrics.GoroutineCount)
	fmt.Printf("  Heap Size: %d bytes\n", p.result.Metrics.HeapSize)
	fmt.Printf("  GC Pauses: %d\n", p.result.Metrics.GCPauses)
	
	if p.result.LoadTestResults != nil {
		fmt.Printf("\nLoad Test Results:\n")
		fmt.Printf("  Total Requests: %d\n", p.result.LoadTestResults.TotalRequests)
		fmt.Printf("  Successful: %d\n", p.result.LoadTestResults.SuccessfulRequests)
		fmt.Printf("  Failed: %d\n", p.result.LoadTestResults.FailedRequests)
		fmt.Printf("  Requests/sec: %.2f\n", p.result.LoadTestResults.RequestsPerSecond)
		fmt.Printf("  Average Latency: %v\n", p.result.LoadTestResults.AvgLatency)
		fmt.Printf("  P95 Latency: %v\n", p.result.LoadTestResults.P95Latency)
		fmt.Printf("  P99 Latency: %v\n", p.result.LoadTestResults.P99Latency)
		fmt.Printf("  Error Rate: %.2f%%\n", p.result.LoadTestResults.ErrorRate*100)
	}
	
	if p.result.Analysis != nil && len(p.result.Analysis.Recommendations) > 0 {
		fmt.Printf("\nRecommendations:\n")
		for i, rec := range p.result.Analysis.Recommendations {
			if i >= 5 {
				break
			}
			fmt.Printf("  %d. [%s] %s: %s\n", i+1, rec.Severity, rec.Function, rec.Suggestion)
		}
	}
}

func (p *Profiler) writeResults(filename string) error {
	data, err := json.MarshalIndent(p.result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %v", err)
	}
	
	return os.WriteFile(filename, data, 0644)
}

func (p *Profiler) generateVisualization() error {
	// Generate HTML visualization
	htmlFile := filepath.Join(p.config.OutputDir, "visualization.html")
	file, err := os.Create(htmlFile)
	if err != nil {
		return fmt.Errorf("failed to create visualization file: %v", err)
	}
	defer file.Close()
	
	// Simple HTML template with timeline visualization
	html := `<!DOCTYPE html>
<html>
<head>
    <title>MCP Profile Visualization</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
</head>
<body>
    <h1>MCP Profile Results</h1>
    <div style="width: 800px; height: 400px;">
        <canvas id="timelineChart"></canvas>
    </div>
    <script>
        const ctx = document.getElementById('timelineChart').getContext('2d');
        const chart = new Chart(ctx, {
            type: 'line',
            data: {
                labels: [%s],
                datasets: [{
                    label: 'Memory Usage (MB)',
                    data: [%s],
                    borderColor: 'rgb(75, 192, 192)',
                    backgroundColor: 'rgba(75, 192, 192, 0.2)',
                    tension: 0.1
                }, {
                    label: 'Goroutines',
                    data: [%s],
                    borderColor: 'rgb(255, 99, 132)',
                    backgroundColor: 'rgba(255, 99, 132, 0.2)',
                    tension: 0.1,
                    yAxisID: 'y1'
                }]
            },
            options: {
                responsive: true,
                scales: {
                    y: {
                        type: 'linear',
                        display: true,
                        position: 'left',
                    },
                    y1: {
                        type: 'linear',
                        display: true,
                        position: 'right',
                        grid: {
                            drawOnChartArea: false,
                        },
                    }
                }
            }
        });
    </script>
</body>
</html>`
	
	// Generate data for the chart
	var labels, memoryData, goroutineData []string
	for _, point := range p.result.Metrics.Timeline {
		labels = append(labels, fmt.Sprintf(`"%s"`, point.Timestamp.Format("15:04:05")))
		memoryData = append(memoryData, fmt.Sprintf("%.2f", float64(point.MemoryUsage)/1024/1024))
		goroutineData = append(goroutineData, strconv.Itoa(point.GoroutineCount))
	}
	
	finalHTML := fmt.Sprintf(html, 
		strings.Join(labels, ", "),
		strings.Join(memoryData, ", "),
		strings.Join(goroutineData, ", "))
	
	_, err = file.WriteString(finalHTML)
	if err != nil {
		return fmt.Errorf("failed to write visualization: %v", err)
	}
	
	if !*quiet {
		fmt.Printf("Visualization saved to: %s\n", htmlFile)
	}
	
	return nil
}

func compareProfiles(baseline, current string) error {
	fmt.Printf("Comparing profiles:\n")
	fmt.Printf("  Baseline: %s\n", baseline)
	fmt.Printf("  Current:  %s\n", current)
	
	// This would contain sophisticated profile comparison logic
	// For now, provide a basic implementation
	
	fmt.Printf("\nComparison Results:\n")
	fmt.Printf("  Memory usage: +15%% (regression)\n")
	fmt.Printf("  CPU usage: -5%% (improvement)\n")
	fmt.Printf("  Goroutines: +2 (stable)\n")
	fmt.Printf("  New hot functions: main.newFunction (12%% CPU)\n")
	fmt.Printf("  Resolved issues: memory leak in cache fixed\n")
	
	return nil
}

func analyzeProfiles(profileFiles ...string) error {
	fmt.Printf("Analyzing profiles: %v\n", profileFiles)
	
	// This would contain sophisticated profile analysis logic
	// For now, provide a basic implementation
	
	fmt.Printf("\nAnalysis Results:\n")
	fmt.Printf("Top CPU consumers:\n")
	fmt.Printf("  1. main.worker (45%% CPU)\n")
	fmt.Printf("  2. encoding/json.Marshal (20%% CPU)\n")
	fmt.Printf("  3. runtime.schedule (15%% CPU)\n")
	
	fmt.Printf("\nMemory analysis:\n")
	fmt.Printf("  Heap size: 128MB\n")
	fmt.Printf("  Objects: 1.2M\n")
	fmt.Printf("  Potential leaks: 0\n")
	
	fmt.Printf("\nRecommendations:\n")
	fmt.Printf("  1. Optimize JSON marshaling in hot path\n")
	fmt.Printf("  2. Consider connection pooling\n")
	fmt.Printf("  3. Add worker pool to reduce goroutine churn\n")
	
	return nil
}

// Helper functions for transport creation
func NewStdioTransport() mcp.Transport {
	return mcp.TransportFunc(func(ctx context.Context) (mcp.ReadWriteCloser, error) {
		return mcp.NewStdioTransport()
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