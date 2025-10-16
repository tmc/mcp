package mcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockConnection simulates a network connection for testing
type mockConnection struct {
	mu          sync.Mutex
	closed      bool
	readDelay   time.Duration
	writeDelay  time.Duration
	failNext    bool
	pingCounter int32
}

func (m *mockConnection) Read(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return 0, ErrTransportClosed
	}
	if m.readDelay > 0 {
		time.Sleep(m.readDelay)
	}
	if m.failNext {
		m.failNext = false
		return 0, errors.New("simulated read error")
	}
	return copy(p, []byte("test data")), nil
}

func (m *mockConnection) Write(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return 0, ErrTransportClosed
	}
	if m.writeDelay > 0 {
		time.Sleep(m.writeDelay)
	}
	if m.failNext {
		m.failNext = false
		return 0, errors.New("simulated write error")
	}
	return len(p), nil
}

func (m *mockConnection) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockConnection) Ping(ctx context.Context) error {
	atomic.AddInt32(&m.pingCounter, 1)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return errors.New("connection closed")
	}
	return nil
}

// mockTransport creates mock connections for testing
type mockPoolTransport struct {
	mu          sync.Mutex
	connections []*mockConnection
	dialDelay   time.Duration
	failNext    bool
}

func (t *mockPoolTransport) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.failNext {
		t.failNext = false
		return nil, errors.New("simulated dial error")
	}

	if t.dialDelay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(t.dialDelay):
		}
	}

	conn := &mockConnection{}
	t.connections = append(t.connections, conn)
	return conn, nil
}

func (t *mockPoolTransport) ConnectionCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.connections)
}

// TestConnectionPool_BasicOperations tests basic pool functionality
func TestConnectionPool_BasicOperations(t *testing.T) {
	transport := &mockPoolTransport{}
	config := &ConnectionPoolConfig{
		MaxConnections:     5,
		MaxIdleTime:        1 * time.Minute,
		MaxConnectionAge:   5 * time.Minute,
		ConnectionTimeout:  5 * time.Second,
		HealthCheckTimeout: 1 * time.Second,
		CleanupInterval:    10 * time.Second,
		EnableHealthChecks: true,
	}

	pool := NewConnectionPool(transport, config, slog.Default())
	defer pool.Close()

	// Test getting a connection
	ctx := context.Background()
	conn1, err := pool.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to get connection: %v", err)
	}

	// Verify stats
	stats := pool.Stats()
	if stats.TotalConnections != 1 {
		t.Errorf("Expected 1 total connection, got %d", stats.TotalConnections)
	}
	if stats.ActiveConnections != 1 {
		t.Errorf("Expected 1 active connection, got %d", stats.ActiveConnections)
	}

	// Return connection
	conn1.Close()

	// Verify connection returned
	stats = pool.Stats()
	if stats.ActiveConnections != 0 {
		t.Errorf("Expected 0 active connections after close, got %d", stats.ActiveConnections)
	}
	if stats.IdleConnections != 1 {
		t.Errorf("Expected 1 idle connection, got %d", stats.IdleConnections)
	}

	// Reuse connection
	conn2, err := pool.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to reuse connection: %v", err)
	}
	defer conn2.Close()

	// Should still have only 1 total connection (reused)
	if transport.ConnectionCount() != 1 {
		t.Errorf("Expected connection reuse, but got %d connections", transport.ConnectionCount())
	}
}

// TestConnectionPool_MaxConnections tests the connection limit
func TestConnectionPool_MaxConnections(t *testing.T) {
	transport := &mockPoolTransport{}
	config := &ConnectionPoolConfig{
		MaxConnections:     3,
		MaxIdleTime:        1 * time.Minute,
		MaxConnectionAge:   5 * time.Minute,
		ConnectionTimeout:  5 * time.Second,
		HealthCheckTimeout: 1 * time.Second,
		CleanupInterval:    10 * time.Second,
		EnableHealthChecks: false,
	}

	pool := NewConnectionPool(transport, config, slog.Default())
	defer pool.Close()

	ctx := context.Background()
	var connections []io.ReadWriteCloser

	// Get max connections
	for i := 0; i < 3; i++ {
		conn, err := pool.Get(ctx)
		if err != nil {
			t.Fatalf("Failed to get connection %d: %v", i, err)
		}
		connections = append(connections, conn)
	}

	// Try to get one more (should fail)
	_, err := pool.Get(ctx)
	if err == nil {
		t.Error("Expected error when exceeding max connections")
	}

	// Return one connection
	connections[0].Close()

	// Now should be able to get another
	conn, err := pool.Get(ctx)
	if err != nil {
		t.Errorf("Should be able to get connection after one returned: %v", err)
	}
	defer conn.Close()

	// Cleanup
	for _, c := range connections[1:] {
		c.Close()
	}
}

// TestConnectionPool_HealthChecks tests health checking functionality
func TestConnectionPool_HealthChecks(t *testing.T) {
	transport := &mockPoolTransport{}
	config := &ConnectionPoolConfig{
		MaxConnections:     5,
		MaxIdleTime:        100 * time.Millisecond,
		MaxConnectionAge:   5 * time.Minute,
		ConnectionTimeout:  5 * time.Second,
		HealthCheckTimeout: 1 * time.Second,
		CleanupInterval:    50 * time.Millisecond,
		EnableHealthChecks: true,
	}

	pool := NewConnectionPool(transport, config, slog.Default())
	defer pool.Close()

	ctx := context.Background()

	// Get and return a connection
	conn, err := pool.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to get connection: %v", err)
	}
	conn.Close()

	// Wait for connection to become idle
	time.Sleep(150 * time.Millisecond)

	// Cleanup should remove idle connection
	pool.cleanup()

	stats := pool.Stats()
	if stats.TotalConnections != 0 {
		t.Errorf("Expected idle connection to be cleaned up, but got %d connections", stats.TotalConnections)
	}
}

// TestConnectionPool_WriteError tests that write errors mark connections unhealthy
func TestConnectionPool_WriteError(t *testing.T) {
	transport := &mockPoolTransport{}
	config := DefaultConnectionPoolConfig()
	config.EnableHealthChecks = false

	pool := NewConnectionPool(transport, config, slog.Default())
	defer pool.Close()

	ctx := context.Background()
	conn, err := pool.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to get connection: %v", err)
	}

	// Force write error on underlying connection
	mockConn := transport.connections[0]
	mockConn.failNext = true

	// Write should fail and mark connection unhealthy
	_, writeErr := conn.Write([]byte("test"))
	if writeErr == nil {
		t.Error("Expected write error")
	}

	conn.Close()

	// Connection should be marked unhealthy
	poolConn := pool.connections[0]
	if poolConn.healthy {
		t.Error("Expected connection to be marked unhealthy after write error")
	}
}

// TestConnectionPool_GracefulShutdown tests graceful shutdown
func TestConnectionPool_GracefulShutdown(t *testing.T) {
	transport := &mockPoolTransport{}
	config := DefaultConnectionPoolConfig()

	pool := NewConnectionPool(transport, config, slog.Default())

	ctx := context.Background()
	conn, err := pool.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to get connection: %v", err)
	}

	// Start graceful shutdown in goroutine
	shutdownDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		shutdownDone <- pool.GracefulShutdown(ctx)
	}()

	// Wait a bit then return connection
	time.Sleep(100 * time.Millisecond)
	conn.Close()

	// Shutdown should complete
	err = <-shutdownDone
	if err != nil {
		t.Errorf("Graceful shutdown failed: %v", err)
	}

	stats := pool.Stats()
	if stats.TotalConnections != 0 {
		t.Errorf("Expected no connections after shutdown, got %d", stats.TotalConnections)
	}
}

// TestConnectionPool_ConcurrentAccess tests concurrent access patterns
func TestConnectionPool_ConcurrentAccess(t *testing.T) {
	transport := &mockPoolTransport{}
	config := &ConnectionPoolConfig{
		MaxConnections:     50, // Match concurrency to avoid exhaustion
		MaxIdleTime:        1 * time.Minute,
		MaxConnectionAge:   5 * time.Minute,
		ConnectionTimeout:  5 * time.Second,
		HealthCheckTimeout: 1 * time.Second,
		CleanupInterval:    1 * time.Second,
		EnableHealthChecks: false,
	}

	pool := NewConnectionPool(transport, config, slog.Default())
	defer pool.Close()

	var wg sync.WaitGroup
	var successCount, errorCount int64
	concurrency := 50
	iterations := 10

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				ctx := context.Background()
				conn, err := pool.Get(ctx)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					continue
				}

				atomic.AddInt64(&successCount, 1)

				// Simulate some work
				time.Sleep(time.Millisecond)

				conn.Close()
			}
		}(i)
	}

	wg.Wait()

	stats := pool.Stats()
	if stats.TotalConnections > config.MaxConnections {
		t.Errorf("Pool exceeded max connections: %d > %d", stats.TotalConnections, config.MaxConnections)
	}

	// All connections should be idle
	if stats.ActiveConnections != 0 {
		t.Errorf("Expected 0 active connections, got %d", stats.ActiveConnections)
	}

	// Should have mostly successful connections
	totalAttempts := int64(concurrency * iterations)
	if successCount < totalAttempts*8/10 { // At least 80% success rate
		t.Errorf("Low success rate: %d/%d (errors: %d)", successCount, totalAttempts, errorCount)
	}
}

// TestPooledTransport tests the PooledTransport wrapper
func TestPooledTransport(t *testing.T) {
	baseTransport := &mockPoolTransport{}
	config := DefaultConnectionPoolConfig()
	config.MaxConnections = 5

	transport := NewPooledTransport(baseTransport, config, slog.Default())
	defer transport.Close()

	ctx := context.Background()
	conn, err := transport.Dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	// Test read/write
	testData := []byte("test message")
	n, err := conn.Write(testData)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(testData), n)
	}

	buffer := make([]byte, 100)
	n, err = conn.Read(buffer)
	if err != nil && err != io.EOF {
		t.Errorf("Read failed: %v", err)
	}
	if n == 0 {
		t.Error("Expected to read data")
	}

	// Check stats
	stats := transport.Stats()
	if stats.TotalConnections == 0 {
		t.Error("Expected at least one connection")
	}
}

// BenchmarkConnectionPool_NoPooling benchmarks without connection pooling
func BenchmarkConnectionPool_NoPooling(b *testing.B) {
	transport := &mockPoolTransport{dialDelay: 1 * time.Millisecond}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn, err := transport.Dial(ctx)
		if err != nil {
			b.Fatalf("Dial failed: %v", err)
		}

		// Simulate work
		conn.Write([]byte("benchmark data"))
		conn.Read(make([]byte, 100))

		conn.Close()
	}
}

// BenchmarkConnectionPool_WithPooling benchmarks with connection pooling
func BenchmarkConnectionPool_WithPooling(b *testing.B) {
	transport := &mockPoolTransport{dialDelay: 1 * time.Millisecond}
	config := &ConnectionPoolConfig{
		MaxConnections:     10,
		MaxIdleTime:        1 * time.Minute,
		MaxConnectionAge:   5 * time.Minute,
		ConnectionTimeout:  5 * time.Second,
		HealthCheckTimeout: 1 * time.Second,
		CleanupInterval:    10 * time.Second,
		EnableHealthChecks: false,
	}

	pool := NewConnectionPool(transport, config, slog.Default())
	defer pool.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn, err := pool.Get(ctx)
		if err != nil {
			b.Fatalf("Get failed: %v", err)
		}

		// Simulate work
		conn.Write([]byte("benchmark data"))
		conn.Read(make([]byte, 100))

		conn.Close() // Returns to pool
	}
}

// BenchmarkConnectionPool_ConcurrentNoPooling benchmarks concurrent access without pooling
func BenchmarkConnectionPool_ConcurrentNoPooling(b *testing.B) {
	transport := &mockPoolTransport{dialDelay: 1 * time.Millisecond}

	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		for pb.Next() {
			conn, err := transport.Dial(ctx)
			if err != nil {
				b.Fatalf("Dial failed: %v", err)
			}

			conn.Write([]byte("concurrent benchmark"))
			conn.Read(make([]byte, 100))
			conn.Close()
		}
	})
}

// BenchmarkConnectionPool_ConcurrentWithPooling benchmarks concurrent access with pooling
func BenchmarkConnectionPool_ConcurrentWithPooling(b *testing.B) {
	transport := &mockPoolTransport{dialDelay: 1 * time.Millisecond}
	config := &ConnectionPoolConfig{
		MaxConnections:     100, // Higher limit for concurrent benchmarks
		MaxIdleTime:        1 * time.Minute,
		MaxConnectionAge:   5 * time.Minute,
		ConnectionTimeout:  5 * time.Second,
		HealthCheckTimeout: 1 * time.Second,
		CleanupInterval:    10 * time.Second,
		EnableHealthChecks: false,
	}

	pool := NewConnectionPool(transport, config, slog.Default())
	defer pool.Close()

	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		for pb.Next() {
			conn, err := pool.Get(ctx)
			if err != nil {
				// Pool exhausted, wait and retry once
				time.Sleep(time.Microsecond)
				conn, err = pool.Get(ctx)
				if err != nil {
					continue // Skip this iteration
				}
			}

			conn.Write([]byte("concurrent benchmark"))
			conn.Read(make([]byte, 100))
			conn.Close()
		}
	})
}

// BenchmarkConnectionPool_HighThroughput benchmarks high-throughput scenarios
func BenchmarkConnectionPool_HighThroughput(b *testing.B) {
	for _, poolSize := range []int{5, 10, 20, 50} {
		b.Run(fmt.Sprintf("PoolSize_%d", poolSize), func(b *testing.B) {
			transport := &mockPoolTransport{dialDelay: 500 * time.Microsecond}
			config := &ConnectionPoolConfig{
				MaxConnections:     poolSize,
				MaxIdleTime:        1 * time.Minute,
				MaxConnectionAge:   5 * time.Minute,
				ConnectionTimeout:  5 * time.Second,
				HealthCheckTimeout: 1 * time.Second,
				CleanupInterval:    10 * time.Second,
				EnableHealthChecks: false,
			}

			pool := NewConnectionPool(transport, config, slog.Default())
			defer pool.Close()

			b.RunParallel(func(pb *testing.PB) {
				ctx := context.Background()
				for pb.Next() {
					conn, err := pool.Get(ctx)
					if err != nil {
						continue // Pool exhausted, skip
					}

					conn.Write([]byte("high throughput test"))
					conn.Close()
				}
			})

			stats := pool.Stats()
			b.ReportMetric(float64(stats.TotalConnections), "connections")
			b.ReportMetric(float64(transport.ConnectionCount()), "created")
		})
	}
}
