// Package mcp - Performance Monitoring and Profiling Utilities
//
// This file provides comprehensive performance monitoring, profiling, and
// optimization utilities for the MCP Go implementation.

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// =============================================================================
// Performance Metrics Collection
// =============================================================================

// PerformanceMetrics tracks various performance indicators
type PerformanceMetrics struct {
	// Request metrics
	TotalRequests       int64         `json:"totalRequests"`
	ActiveRequests      int64         `json:"activeRequests"`
	AverageResponseTime time.Duration `json:"averageResponseTime"`
	MedianResponseTime  time.Duration `json:"medianResponseTime"`
	P95ResponseTime     time.Duration `json:"p95ResponseTime"`
	P99ResponseTime     time.Duration `json:"p99ResponseTime"`
	MaxResponseTime     time.Duration `json:"maxResponseTime"`

	// Error metrics
	TotalErrors      int64   `json:"totalErrors"`
	ErrorRate        float64 `json:"errorRate"`
	TimeoutErrors    int64   `json:"timeoutErrors"`
	ConnectionErrors int64   `json:"connectionErrors"`

	// Throughput metrics
	RequestsPerSecond float64 `json:"requestsPerSecond"`
	BytesPerSecond    float64 `json:"bytesPerSecond"`

	// Memory metrics
	AllocatedMemory  uint64 `json:"allocatedMemory"`
	TotalAllocations uint64 `json:"totalAllocations"`
	GCPauses         uint64 `json:"gcPauses"`

	// Concurrency metrics
	MaxConcurrency int64   `json:"maxConcurrency"`
	AvgConcurrency float64 `json:"avgConcurrency"`

	// Transport metrics
	BytesSent         int64 `json:"bytesSent"`
	BytesReceived     int64 `json:"bytesReceived"`
	ConnectionsOpened int64 `json:"connectionsOpened"`
	ConnectionsClosed int64 `json:"connectionsClosed"`
}

// PerformanceMonitor provides real-time performance monitoring
type PerformanceMonitor struct {
	mu            sync.RWMutex
	startTime     time.Time
	responseTimes []time.Duration
	metrics       PerformanceMetrics
	requestChan   chan requestMetric
	stopChan      chan struct{}
	samplingRate  float64
	maxSamples    int
}

type requestMetric struct {
	startTime     time.Time
	endTime       time.Time
	success       bool
	error         error
	bytesSent     int64
	bytesReceived int64
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(samplingRate float64, maxSamples int) *PerformanceMonitor {
	pm := &PerformanceMonitor{
		startTime:     time.Now(),
		responseTimes: make([]time.Duration, 0, maxSamples),
		requestChan:   make(chan requestMetric, 1000),
		stopChan:      make(chan struct{}),
		samplingRate:  samplingRate,
		maxSamples:    maxSamples,
	}

	go pm.collectMetrics()
	go pm.updateMemoryMetrics()

	return pm
}

// RecordRequest records a completed request for performance tracking
func (pm *PerformanceMonitor) RecordRequest(startTime, endTime time.Time, success bool, err error, bytesSent, bytesReceived int64) {
	select {
	case pm.requestChan <- requestMetric{
		startTime:     startTime,
		endTime:       endTime,
		success:       success,
		error:         err,
		bytesSent:     bytesSent,
		bytesReceived: bytesReceived,
	}:
	default:
		// Channel full, drop metric (should be rare with proper sizing)
	}
}

// GetMetrics returns current performance metrics
func (pm *PerformanceMonitor) GetMetrics() PerformanceMetrics {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.metrics
}

// GetMetricsJSON returns performance metrics as JSON
func (pm *PerformanceMonitor) GetMetricsJSON() ([]byte, error) {
	metrics := pm.GetMetrics()
	return json.MarshalIndent(metrics, "", "  ")
}

// Stop stops the performance monitor
func (pm *PerformanceMonitor) Stop() {
	close(pm.stopChan)
}

func (pm *PerformanceMonitor) collectMetrics() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case metric := <-pm.requestChan:
			pm.processRequestMetric(metric)
		case <-ticker.C:
			pm.calculateDerivedMetrics()
		case <-pm.stopChan:
			return
		}
	}
}

func (pm *PerformanceMonitor) processRequestMetric(metric requestMetric) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	duration := metric.endTime.Sub(metric.startTime)

	// Update request counters
	atomic.AddInt64(&pm.metrics.TotalRequests, 1)

	// Track response times
	if len(pm.responseTimes) >= pm.maxSamples {
		// Remove oldest sample
		pm.responseTimes = pm.responseTimes[1:]
	}
	pm.responseTimes = append(pm.responseTimes, duration)

	// Update max response time
	if duration > pm.metrics.MaxResponseTime {
		pm.metrics.MaxResponseTime = duration
	}

	// Update error counters
	if !metric.success {
		atomic.AddInt64(&pm.metrics.TotalErrors, 1)

		if metric.error != nil {
			// Classify error types
			switch {
			case isTimeoutError(metric.error):
				atomic.AddInt64(&pm.metrics.TimeoutErrors, 1)
			case isConnectionError(metric.error):
				atomic.AddInt64(&pm.metrics.ConnectionErrors, 1)
			}
		}
	}

	// Update byte counters
	atomic.AddInt64(&pm.metrics.BytesSent, metric.bytesSent)
	atomic.AddInt64(&pm.metrics.BytesReceived, metric.bytesReceived)
}

func (pm *PerformanceMonitor) calculateDerivedMetrics() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.responseTimes) == 0 {
		return
	}

	// Calculate response time percentiles
	sortedTimes := make([]time.Duration, len(pm.responseTimes))
	copy(sortedTimes, pm.responseTimes)

	// Simple sort for percentile calculation
	for i := 0; i < len(sortedTimes)-1; i++ {
		for j := i + 1; j < len(sortedTimes); j++ {
			if sortedTimes[i] > sortedTimes[j] {
				sortedTimes[i], sortedTimes[j] = sortedTimes[j], sortedTimes[i]
			}
		}
	}

	// Calculate percentiles
	count := len(sortedTimes)
	pm.metrics.MedianResponseTime = sortedTimes[count/2]
	pm.metrics.P95ResponseTime = sortedTimes[int(float64(count)*0.95)]
	pm.metrics.P99ResponseTime = sortedTimes[int(float64(count)*0.99)]

	// Calculate average
	var total time.Duration
	for _, t := range pm.responseTimes {
		total += t
	}
	pm.metrics.AverageResponseTime = total / time.Duration(len(pm.responseTimes))

	// Calculate rates
	elapsed := time.Since(pm.startTime).Seconds()
	if elapsed > 0 {
		pm.metrics.RequestsPerSecond = float64(pm.metrics.TotalRequests) / elapsed
		totalBytes := float64(pm.metrics.BytesSent + pm.metrics.BytesReceived)
		pm.metrics.BytesPerSecond = totalBytes / elapsed
	}

	// Calculate error rate
	if pm.metrics.TotalRequests > 0 {
		pm.metrics.ErrorRate = float64(pm.metrics.TotalErrors) / float64(pm.metrics.TotalRequests)
	}
}

func (pm *PerformanceMonitor) updateMemoryMetrics() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			pm.mu.Lock()
			pm.metrics.AllocatedMemory = m.Alloc
			pm.metrics.TotalAllocations = m.TotalAlloc
			pm.metrics.GCPauses = uint64(m.NumGC)
			pm.mu.Unlock()

		case <-pm.stopChan:
			return
		}
	}
}

// =============================================================================
// Performance-Aware Client
// =============================================================================

// PerformanceAwareClient wraps a regular client with performance monitoring
type PerformanceAwareClient struct {
	*Client
	monitor *PerformanceMonitor
}

// NewPerformanceAwareClient creates a new performance-aware client
func NewPerformanceAwareClient(transport Transport, options ...ClientOption) (*PerformanceAwareClient, error) {
	client, err := NewClient(transport, options...)
	if err != nil {
		return nil, err
	}

	return &PerformanceAwareClient{
		Client:  client,
		monitor: NewPerformanceMonitor(1.0, 10000),
	}, nil
}

// CallTool wraps the regular CallTool with performance monitoring
func (pac *PerformanceAwareClient) CallTool(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
	startTime := time.Now()

	// Estimate request size
	reqData, _ := json.Marshal(req)
	bytesSent := int64(len(reqData))

	result, err := pac.Client.CallTool(ctx, req)

	endTime := time.Now()
	success := err == nil

	// Estimate response size
	var bytesReceived int64
	if result != nil {
		respData, _ := json.Marshal(result)
		bytesReceived = int64(len(respData))
	}

	pac.monitor.RecordRequest(startTime, endTime, success, err, bytesSent, bytesReceived)

	return result, err
}

// GetPerformanceMetrics returns performance metrics for the client
func (pac *PerformanceAwareClient) GetPerformanceMetrics() PerformanceMetrics {
	return pac.monitor.GetMetrics()
}

// Close closes the client and stops monitoring
func (pac *PerformanceAwareClient) Close() error {
	pac.monitor.Stop()
	return pac.Client.Close()
}

// =============================================================================
// Performance-Aware Server
// =============================================================================

// PerformanceAwareServer wraps a regular server with performance monitoring
type PerformanceAwareServer struct {
	*Server
	monitor *PerformanceMonitor
}

// NewPerformanceAwareServer creates a new performance-aware server
func NewPerformanceAwareServer(name, version string, options ...ServerOption) *PerformanceAwareServer {
	server := NewServer(name, version, options...)

	return &PerformanceAwareServer{
		Server:  server,
		monitor: NewPerformanceMonitor(1.0, 10000),
	}
}

// =============================================================================
// Resource Pool for Performance Optimization
// =============================================================================

// ResourcePool manages reusable resources to reduce allocations
type ResourcePool struct {
	bufferPool     sync.Pool
	contextPool    sync.Pool
	requestPool    sync.Pool
	responsePool   sync.Pool
	rawMessagePool sync.Pool
}

// NewResourcePool creates a new resource pool
func NewResourcePool() *ResourcePool {
	return &ResourcePool{
		bufferPool: sync.Pool{
			// Store *[]byte rather than []byte: putting a slice header by value
			// boxes it into an interface and allocates on every Put (SA6002).
			New: func() interface{} {
				b := make([]byte, 0, 4096)
				return &b
			},
		},
		contextPool: sync.Pool{
			New: func() interface{} {
				return context.Background()
			},
		},
		requestPool: sync.Pool{
			New: func() interface{} {
				return &CallToolRequest{}
			},
		},
		responsePool: sync.Pool{
			New: func() interface{} {
				return &CallToolResult{}
			},
		},
		rawMessagePool: sync.Pool{
			New: func() interface{} {
				msg := make(json.RawMessage, 0, 1024)
				return &msg
			},
		},
	}
}

// GetBuffer gets a buffer from the pool
func (rp *ResourcePool) GetBuffer() []byte {
	return (*rp.bufferPool.Get().(*[]byte))[:0]
}

// PutBuffer returns a buffer to the pool
func (rp *ResourcePool) PutBuffer(buf []byte) {
	if cap(buf) <= 65536 { // Don't pool very large buffers
		rp.bufferPool.Put(&buf)
	}
}

// GetRequest gets a request object from the pool
func (rp *ResourcePool) GetRequest() *CallToolRequest {
	req := rp.requestPool.Get().(*CallToolRequest)
	// Reset the request
	req.Name = ""
	req.Arguments = nil
	return req
}

// PutRequest returns a request object to the pool
func (rp *ResourcePool) PutRequest(req *CallToolRequest) {
	rp.requestPool.Put(req)
}

// GetResponse gets a response object from the pool
func (rp *ResourcePool) GetResponse() *CallToolResult {
	resp := rp.responsePool.Get().(*CallToolResult)
	// Reset the response
	resp.Content = nil
	resp.IsError = false
	return resp
}

// PutResponse returns a response object to the pool
func (rp *ResourcePool) PutResponse(resp *CallToolResult) {
	rp.responsePool.Put(resp)
}

// GetRawMessage gets a json.RawMessage from the pool
func (rp *ResourcePool) GetRawMessage() *json.RawMessage {
	msg := rp.rawMessagePool.Get().(*json.RawMessage)
	*msg = (*msg)[:0] // Reset length while keeping capacity
	return msg
}

// PutRawMessage returns a json.RawMessage to the pool
func (rp *ResourcePool) PutRawMessage(msg *json.RawMessage) {
	if cap(*msg) <= 65536 { // Don't pool very large messages
		rp.rawMessagePool.Put(msg)
	}
}

// GetJSONEncoder returns a JSON encoder writing to w.
//
// json.Encoder binds to its writer at construction and has no Reset, so it
// cannot be pooled across writers; this is a plain constructor kept for API
// symmetry with the pooled resources above.
func (rp *ResourcePool) GetJSONEncoder(w io.Writer) *json.Encoder {
	return json.NewEncoder(w)
}

// GetJSONDecoder returns a JSON decoder reading from r.
//
// json.Decoder binds to its reader at construction and has no Reset, so it
// cannot be pooled across readers; this is a plain constructor kept for API
// symmetry with the pooled resources above.
func (rp *ResourcePool) GetJSONDecoder(r io.Reader) *json.Decoder {
	return json.NewDecoder(r)
}

// =============================================================================
// Performance Optimization Utilities
// =============================================================================

// OptimizedJSONEncoder provides optimized JSON encoding with pooled buffers
type OptimizedJSONEncoder struct {
	pool *ResourcePool
}

// NewOptimizedJSONEncoder creates a new optimized JSON encoder
func NewOptimizedJSONEncoder() *OptimizedJSONEncoder {
	return &OptimizedJSONEncoder{
		pool: NewResourcePool(),
	}
}

// Marshal marshals the given value to JSON using a pooled buffer
func (oje *OptimizedJSONEncoder) Marshal(v interface{}) ([]byte, error) {
	buf := oje.pool.GetBuffer()
	defer oje.pool.PutBuffer(buf)

	encoder := json.NewEncoder(&bufferWriter{buf: &buf})
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}

	// Return a copy since we're returning the buffer to the pool
	result := make([]byte, len(buf))
	copy(result, buf)
	return result, nil
}

type bufferWriter struct {
	buf *[]byte
}

func (bw *bufferWriter) Write(p []byte) (n int, err error) {
	*bw.buf = append(*bw.buf, p...)
	return len(p), nil
}

// Connection Pool functionality is now provided by connection_pool.go

// =============================================================================
// Utility Functions
// =============================================================================

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	return err == context.DeadlineExceeded ||
		fmt.Sprintf("%v", err) == "context deadline exceeded"
}

func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return errStr == "connection refused" ||
		errStr == "connection reset" ||
		errStr == "broken pipe"
}

// Global performance monitor instance
var globalPerformanceMonitor *PerformanceMonitor
var globalResourcePool *ResourcePool

func init() {
	globalPerformanceMonitor = NewPerformanceMonitor(0.1, 1000) // 10% sampling
	globalResourcePool = NewResourcePool()
}

// GetGlobalPerformanceMetrics returns global performance metrics
func GetGlobalPerformanceMetrics() PerformanceMetrics {
	if globalPerformanceMonitor != nil {
		return globalPerformanceMonitor.GetMetrics()
	}
	return PerformanceMetrics{}
}

// GetGlobalResourcePool returns the global resource pool
func GetGlobalResourcePool() *ResourcePool {
	return globalResourcePool
}
