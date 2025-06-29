// Package mcp - Performance Monitoring and Profiling Utilities
//
// This file provides comprehensive performance monitoring, profiling, and
// optimization utilities for the MCP Go implementation.

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
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
	TotalRequests        int64         `json:"totalRequests"`
	ActiveRequests       int64         `json:"activeRequests"`
	AverageResponseTime  time.Duration `json:"averageResponseTime"`
	MedianResponseTime   time.Duration `json:"medianResponseTime"`
	P95ResponseTime      time.Duration `json:"p95ResponseTime"`
	P99ResponseTime      time.Duration `json:"p99ResponseTime"`
	MaxResponseTime      time.Duration `json:"maxResponseTime"`
	
	// Error metrics
	TotalErrors         int64 `json:"totalErrors"`
	ErrorRate          float64 `json:"errorRate"`
	TimeoutErrors      int64 `json:"timeoutErrors"`
	ConnectionErrors   int64 `json:"connectionErrors"`
	
	// Throughput metrics
	RequestsPerSecond  float64 `json:"requestsPerSecond"`
	BytesPerSecond     float64 `json:"bytesPerSecond"`
	
	// Memory metrics
	AllocatedMemory    uint64 `json:"allocatedMemory"`
	TotalAllocations   uint64 `json:"totalAllocations"`
	GCPauses          uint64 `json:"gcPauses"`
	
	// Concurrency metrics
	MaxConcurrency     int64 `json:"maxConcurrency"`
	AvgConcurrency     float64 `json:"avgConcurrency"`
	
	// Transport metrics
	BytesSent         int64 `json:"bytesSent"`
	BytesReceived     int64 `json:"bytesReceived"`
	ConnectionsOpened int64 `json:"connectionsOpened"`
	ConnectionsClosed int64 `json:"connectionsClosed"`
}

// PerformanceMonitor provides real-time performance monitoring
type PerformanceMonitor struct {
	mu                sync.RWMutex
	startTime         time.Time
	responseTimes     []time.Duration
	metrics           PerformanceMetrics
	requestChan       chan requestMetric
	stopChan          chan struct{}
	samplingRate      float64
	maxSamples        int
}

type requestMetric struct {
	startTime    time.Time
	endTime      time.Time
	success      bool
	error        error
	bytesSent    int64
	bytesReceived int64
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(samplingRate float64, maxSamples int) *PerformanceMonitor {
	pm := &PerformanceMonitor{
		startTime:    time.Now(),
		responseTimes: make([]time.Duration, 0, maxSamples),
		requestChan:  make(chan requestMetric, 1000),
		stopChan:     make(chan struct{}),
		samplingRate: samplingRate,
		maxSamples:   maxSamples,
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
	bufferPool   sync.Pool
	contextPool  sync.Pool
	requestPool  sync.Pool
	responsePool sync.Pool
}

// NewResourcePool creates a new resource pool
func NewResourcePool() *ResourcePool {
	return &ResourcePool{
		bufferPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 4096)
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
	}
}

// GetBuffer gets a buffer from the pool
func (rp *ResourcePool) GetBuffer() []byte {
	return rp.bufferPool.Get().([]byte)[:0]
}

// PutBuffer returns a buffer to the pool
func (rp *ResourcePool) PutBuffer(buf []byte) {
	if cap(buf) <= 65536 { // Don't pool very large buffers
		rp.bufferPool.Put(buf)
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

// =============================================================================
// Connection Pool for Performance
// =============================================================================

// ConnectionPool manages a pool of reusable connections
type ConnectionPool struct {
	factory     func() (Transport, error)
	connections chan Transport
	maxSize     int
	created     int64
	inUse       int64
	mu          sync.Mutex
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(factory func() (Transport, error), maxSize int) *ConnectionPool {
	return &ConnectionPool{
		factory:     factory,
		connections: make(chan Transport, maxSize),
		maxSize:     maxSize,
	}
}

// Get gets a connection from the pool
func (cp *ConnectionPool) Get() (Transport, error) {
	select {
	case conn := <-cp.connections:
		atomic.AddInt64(&cp.inUse, 1)
		return conn, nil
	default:
		// No available connection, create new one if under limit
		cp.mu.Lock()
		if int(cp.created) < cp.maxSize {
			cp.created++
			cp.mu.Unlock()
			
			conn, err := cp.factory()
			if err != nil {
				cp.mu.Lock()
				cp.created--
				cp.mu.Unlock()
				return nil, err
			}
			
			atomic.AddInt64(&cp.inUse, 1)
			return conn, nil
		}
		cp.mu.Unlock()
		
		// Wait for available connection
		conn := <-cp.connections
		atomic.AddInt64(&cp.inUse, 1)
		return conn, nil
	}
}

// Put returns a connection to the pool
func (cp *ConnectionPool) Put(conn Transport) {
	atomic.AddInt64(&cp.inUse, -1)
	
	select {
	case cp.connections <- conn:
		// Connection returned to pool
	default:
		// Pool is full, close the connection
		conn.Close()
		cp.mu.Lock()
		cp.created--
		cp.mu.Unlock()
	}
}

// Close closes all connections in the pool
func (cp *ConnectionPool) Close() error {
	close(cp.connections)
	
	for conn := range cp.connections {
		conn.Close()
	}
	
	return nil
}

// Stats returns pool statistics
func (cp *ConnectionPool) Stats() (created, inUse, available int64) {
	cp.mu.Lock()
	created = cp.created
	cp.mu.Unlock()
	
	inUse = atomic.LoadInt64(&cp.inUse)
	available = int64(len(cp.connections))
	
	return created, inUse, available
}

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
	errStr := fmt.Sprintf("%v", err)
	return fmt.Sprintf("%s", errStr) == "connection refused" ||
		   fmt.Sprintf("%s", errStr) == "connection reset" ||
		   fmt.Sprintf("%s", errStr) == "broken pipe"
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