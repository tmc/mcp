package mcp

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"
)

// PooledConnection represents a connection in the pool with metadata
type PooledConnection struct {
	conn       io.ReadWriteCloser
	lastUsed   time.Time
	inUse      bool
	healthy    bool
	createdAt  time.Time
}

// ConnectionPool manages a pool of reusable connections
type ConnectionPool struct {
	transport Transport
	config    *ConnectionPoolConfig
	logger    *slog.Logger

	mu          sync.RWMutex
	connections []*PooledConnection
	activeCount int

	// Cleanup management
	cleanupTicker *time.Ticker
	cleanupDone   chan struct{}
	closed        bool
}

// NewConnectionPool creates a new connection pool with the given transport and configuration
func NewConnectionPool(transport Transport, config *ConnectionPoolConfig, logger *slog.Logger) *ConnectionPool {
	if config == nil {
		config = DefaultConnectionPoolConfig()
	}
	if logger == nil {
		logger = slog.Default()
	}

	pool := &ConnectionPool{
		transport:   transport,
		config:      config,
		logger:      logger,
		connections: make([]*PooledConnection, 0, config.MaxConnections),
		cleanupDone: make(chan struct{}),
	}

	// Start cleanup routine
	if config.CleanupInterval > 0 {
		pool.cleanupTicker = time.NewTicker(config.CleanupInterval)
		go pool.cleanupRoutine()
	}

	return pool
}

// Get retrieves a connection from the pool or creates a new one
func (p *ConnectionPool) Get(ctx context.Context) (io.ReadWriteCloser, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, ErrTransportClosed
	}

	// Try to find an available healthy connection
	for _, conn := range p.connections {
		if !conn.inUse && conn.healthy {
			// Check if connection is still valid
			if p.config.EnableHealthChecks && !p.isConnectionHealthy(conn) {
				p.logger.Debug("Connection failed health check, marking as unhealthy")
				conn.healthy = false
				continue
			}

			// Mark as in use and update last used time
			conn.inUse = true
			conn.lastUsed = time.Now()
			p.activeCount++

			p.logger.Debug("Reusing pooled connection", "active_count", p.activeCount)
			return &pooledConnectionWrapper{conn: conn, pool: p}, nil
		}
	}

	// No available connection, create a new one if under limit
	if len(p.connections) < p.config.MaxConnections {
		// Create connection with timeout
		ctx, cancel := context.WithTimeout(ctx, p.config.ConnectionTimeout)
		defer cancel()

		rawConn, err := p.transport.Dial(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create new connection: %w", err)
		}

		now := time.Now()
		pooledConn := &PooledConnection{
			conn:      rawConn,
			lastUsed:  now,
			inUse:     true,
			healthy:   true,
			createdAt: now,
		}

		p.connections = append(p.connections, pooledConn)
		p.activeCount++

		p.logger.Debug("Created new pooled connection", 
			"total_connections", len(p.connections),
			"active_count", p.activeCount)

		return &pooledConnectionWrapper{conn: pooledConn, pool: p}, nil
	}

	// Pool is full, return error
	return nil, fmt.Errorf("connection pool exhausted (max: %d)", p.config.MaxConnections)
}

// Return marks a connection as available for reuse
func (p *ConnectionPool) Return(conn *PooledConnection) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conn.inUse {
		conn.inUse = false
		conn.lastUsed = time.Now()
		p.activeCount--

		p.logger.Debug("Returned connection to pool", "active_count", p.activeCount)
	}
}

// Close closes all connections in the pool and stops the cleanup routine
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true

	// Stop cleanup routine
	if p.cleanupTicker != nil {
		p.cleanupTicker.Stop()
		close(p.cleanupDone)
	}

	// Close all connections
	var errs []error
	for _, conn := range p.connections {
		if err := conn.conn.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	p.connections = nil
	p.activeCount = 0

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}

	p.logger.Debug("Connection pool closed")
	return nil
}

// Stats returns statistics about the connection pool
func (p *ConnectionPool) Stats() ConnectionPoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	healthy := 0
	idle := 0
	for _, conn := range p.connections {
		if conn.healthy {
			healthy++
		}
		if !conn.inUse {
			idle++
		}
	}

	return ConnectionPoolStats{
		TotalConnections:   len(p.connections),
		ActiveConnections:  p.activeCount,
		IdleConnections:    idle,
		HealthyConnections: healthy,
		MaxConnections:     p.config.MaxConnections,
	}
}

// ConnectionPoolStats provides statistics about the connection pool
type ConnectionPoolStats struct {
	TotalConnections   int `json:"totalConnections"`
	ActiveConnections  int `json:"activeConnections"`
	IdleConnections    int `json:"idleConnections"`
	HealthyConnections int `json:"healthyConnections"`
	MaxConnections     int `json:"maxConnections"`
}

// cleanupRoutine periodically removes idle and unhealthy connections
func (p *ConnectionPool) cleanupRoutine() {
	for {
		select {
		case <-p.cleanupTicker.C:
			p.cleanup()
		case <-p.cleanupDone:
			return
		}
	}
}

// cleanup removes idle and unhealthy connections
func (p *ConnectionPool) cleanup() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}

	now := time.Now()
	var newConnections []*PooledConnection
	var closedCount int

	for _, conn := range p.connections {
		shouldRemove := false

		// Remove if unhealthy
		if !conn.healthy {
			shouldRemove = true
		}

		// Remove if idle for too long
		if !conn.inUse && now.Sub(conn.lastUsed) > p.config.MaxIdleTime {
			shouldRemove = true
		}

		if shouldRemove && !conn.inUse {
			if err := conn.conn.Close(); err != nil {
				p.logger.Warn("Error closing connection during cleanup", "error", err)
			}
			closedCount++
		} else {
			newConnections = append(newConnections, conn)
		}
	}

	p.connections = newConnections

	if closedCount > 0 {
		p.logger.Debug("Cleaned up idle connections", 
			"closed_count", closedCount, 
			"remaining_connections", len(p.connections))
	}
}

// isConnectionHealthy performs a health check on a connection
func (p *ConnectionPool) isConnectionHealthy(conn *PooledConnection) bool {
	if conn.conn == nil {
		return false
	}
	
	// Check if connection has been idle too long
	if time.Since(conn.lastUsed) > p.config.MaxIdleTime {
		p.logger.Debug("Connection idle too long", "idle_duration", time.Since(conn.lastUsed))
		return false
	}
	
	// Check if connection is too old
	if time.Since(conn.createdAt) > p.config.MaxConnectionAge {
		p.logger.Debug("Connection too old", "age", time.Since(conn.createdAt))
		return false
	}
	
	// For connections that support ping, send a ping
	if pinger, ok := conn.conn.(ConnectionPinger); ok {
		ctx, cancel := context.WithTimeout(context.Background(), p.config.HealthCheckTimeout)
		defer cancel()
		
		if err := pinger.Ping(ctx); err != nil {
			p.logger.Debug("Connection ping failed", "error", err)
			return false
		}
	}
	
	return true
}

// ConnectionPinger interface for connections that support ping operations
type ConnectionPinger interface {
	Ping(ctx context.Context) error
}

// GracefulShutdown performs a graceful shutdown of the connection pool
func (p *ConnectionPool) GracefulShutdown(ctx context.Context) error {
	p.logger.Info("Starting graceful shutdown of connection pool")
	
	// Set a deadline for shutdown
	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		// Default 30 second timeout
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		deadline, _ = ctx.Deadline()
	}
	
	// Wait for active connections to become idle
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			p.logger.Warn("Graceful shutdown timeout, forcing close")
			return p.Close()
		case <-ticker.C:
			p.mu.RLock()
			activeCount := p.activeCount
			closed := p.closed
			p.mu.RUnlock()
			
			if closed {
				return nil
			}
			
			if activeCount == 0 {
				p.logger.Info("All connections idle, proceeding with shutdown")
				return p.Close()
			}
			
			remaining := time.Until(deadline)
			p.logger.Debug("Waiting for connections to become idle", 
				"active_connections", activeCount,
				"remaining_time", remaining)
		}
	}
}

// pooledConnectionWrapper wraps a pooled connection to automatically return it on close
type pooledConnectionWrapper struct {
	conn   *PooledConnection
	pool   *ConnectionPool
	closed bool
}

func (w *pooledConnectionWrapper) Read(p []byte) (n int, err error) {
	if w.closed {
		return 0, ErrTransportClosed
	}
	return w.conn.conn.Read(p)
}

func (w *pooledConnectionWrapper) Write(p []byte) (n int, err error) {
	if w.closed {
		return 0, ErrTransportClosed
	}
	n, err = w.conn.conn.Write(p)
	if err != nil {
		// Mark connection as unhealthy on write error
		w.pool.mu.Lock()
		w.conn.healthy = false
		w.pool.mu.Unlock()
	}
	return n, err
}

func (w *pooledConnectionWrapper) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true

	// Return connection to pool instead of closing it
	w.pool.Return(w.conn)
	return nil
}

// PooledTransport wraps a transport with connection pooling
type PooledTransport struct {
	pool *ConnectionPool
}

// NewPooledTransport creates a new pooled transport wrapper
func NewPooledTransport(transport Transport, config *ConnectionPoolConfig, logger *slog.Logger) *PooledTransport {
	return &PooledTransport{
		pool: NewConnectionPool(transport, config, logger),
	}
}

// Dial implements the Transport interface using the connection pool
func (t *PooledTransport) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	return t.pool.Get(ctx)
}

// Close closes the underlying connection pool
func (t *PooledTransport) Close() error {
	return t.pool.Close()
}

// Stats returns statistics about the connection pool
func (t *PooledTransport) Stats() ConnectionPoolStats {
	return t.pool.Stats()
}