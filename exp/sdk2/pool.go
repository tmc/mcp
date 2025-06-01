package sdk2

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// PoolConfig configures connection pooling behavior.
// This follows the database/sql DB configuration pattern.
type PoolConfig struct {
	// MaxIdleConns sets the maximum number of connections in the idle
	// connection pool.
	MaxIdleConns int
	
	// MaxOpenConns sets the maximum number of open connections to the server.
	// If MaxIdleConns is greater than 0 and the new MaxOpenConns is less than
	// MaxIdleConns, then MaxIdleConns will be reduced to match the new
	// MaxOpenConns limit.
	MaxOpenConns int
	
	// ConnMaxLifetime sets the maximum amount of time a connection may be reused.
	// Expired connections may be closed lazily before reuse.
	// If d <= 0, connections are not closed due to a connection's age.
	ConnMaxLifetime time.Duration
	
	// ConnMaxIdleTime sets the maximum amount of time a connection may be idle.
	// Expired connections may be closed lazily before reuse.
	// If d <= 0, connections are not closed due to a connection's idle time.
	ConnMaxIdleTime time.Duration
}

// DefaultPoolConfig returns a default pool configuration.
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 10 * time.Minute,
	}
}

// ConnectionPool manages a pool of connections following database/sql patterns.
type ConnectionPool struct {
	config   *PoolConfig
	factory  func(context.Context) (Conn, error)
	
	mu          sync.RWMutex
	idle        []*pooledConn
	active      map[*pooledConn]struct{}
	numOpen     int32
	cleanerStop chan struct{}
	cleanerDone chan struct{}
	
	stats ConnectionStats
}

// ConnectionStats contains connection pool statistics.
type ConnectionStats struct {
	MaxOpenConnections int   // Maximum number of open connections to the database.
	OpenConnections    int   // The number of established connections both in use and idle.
	InUse              int   // The number of connections currently in use.
	Idle               int   // The number of idle connections.
	WaitCount          int64 // The total number of connections waited for.
	WaitDuration       time.Duration // The total time blocked waiting for a new connection.
	MaxIdleClosed      int64 // The total number of connections closed due to SetMaxIdleConns.
	MaxIdleTimeClosed  int64 // The total number of connections closed due to ConnMaxIdleTime.
	MaxLifetimeClosed  int64 // The total number of connections closed due to ConnMaxLifetime.
}

// pooledConn wraps a connection with pool management metadata.
type pooledConn struct {
	Conn
	createdAt time.Time
	lastUsed  time.Time
	pool      *ConnectionPool
	inUse     bool
}

// NewConnectionPool creates a new connection pool.
func NewConnectionPool(config *PoolConfig, factory func(context.Context) (Conn, error)) *ConnectionPool {
	if config == nil {
		config = DefaultPoolConfig()
	}
	
	pool := &ConnectionPool{
		config:      config,
		factory:     factory,
		active:      make(map[*pooledConn]struct{}),
		cleanerStop: make(chan struct{}),
		cleanerDone: make(chan struct{}),
	}
	
	// Start the connection cleaner
	go pool.connectionCleaner()
	
	return pool
}

// Get retrieves a connection from the pool.
func (p *ConnectionPool) Get(ctx context.Context) (Conn, error) {
	conn, err := p.getConn(ctx)
	if err != nil {
		return nil, WrapError("pool.get", err)
	}
	return conn, nil
}

// getConn is the internal connection retrieval method.
func (p *ConnectionPool) getConn(ctx context.Context) (*pooledConn, error) {
	waitStart := time.Now()
	
	// Try to get an idle connection first
	if conn := p.getIdleConn(); conn != nil {
		return conn, nil
	}
	
	// Check if we can create a new connection
	if p.canOpenNewConn() {
		conn, err := p.createConn(ctx)
		if err != nil {
			return nil, err
		}
		return conn, nil
	}
	
	// Wait for a connection to become available
	p.mu.Lock()
	p.stats.WaitCount++
	p.mu.Unlock()
	
	defer func() {
		p.mu.Lock()
		p.stats.WaitDuration += time.Since(waitStart)
		p.mu.Unlock()
	}()
	
	// Wait for connection or context cancellation
	for {
		select {
		case <-ctx.Done():
			return nil, NewError("pool.get", StatusRequestTimeout, "connection timeout", ctx.Err())
		default:
			if conn := p.getIdleConn(); conn != nil {
				return conn, nil
			}
			// Brief wait before retrying
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// getIdleConn retrieves an idle connection from the pool.
func (p *ConnectionPool) getIdleConn() *pooledConn {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if len(p.idle) == 0 {
		return nil
	}
	
	// Get the most recently used connection (LIFO)
	conn := p.idle[len(p.idle)-1]
	p.idle = p.idle[:len(p.idle)-1]
	
	// Mark as in use
	conn.inUse = true
	conn.lastUsed = time.Now()
	p.active[conn] = struct{}{}
	
	p.updateStats()
	return conn
}

// canOpenNewConn checks if a new connection can be opened.
func (p *ConnectionPool) canOpenNewConn() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	return int(atomic.LoadInt32(&p.numOpen)) < p.config.MaxOpenConns
}

// createConn creates a new connection.
func (p *ConnectionPool) createConn(ctx context.Context) (*pooledConn, error) {
	rawConn, err := p.factory(ctx)
	if err != nil {
		return nil, err
	}
	
	conn := &pooledConn{
		Conn:      rawConn,
		createdAt: time.Now(),
		lastUsed:  time.Now(),
		pool:      p,
		inUse:     true,
	}
	
	p.mu.Lock()
	p.active[conn] = struct{}{}
	atomic.AddInt32(&p.numOpen, 1)
	p.updateStats()
	p.mu.Unlock()
	
	return conn, nil
}

// Put returns a connection to the pool.
func (p *ConnectionPool) Put(conn Conn) error {
	pooledConn, ok := conn.(*pooledConn)
	if !ok {
		// Not a pooled connection, just close it
		return conn.Close()
	}
	
	return p.putConn(pooledConn)
}

// putConn returns a pooled connection to the pool.
func (p *ConnectionPool) putConn(conn *pooledConn) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if !conn.inUse {
		// Connection already returned
		return nil
	}
	
	// Remove from active connections
	delete(p.active, conn)
	conn.inUse = false
	
	// Check if connection should be discarded
	if p.shouldDiscardConn(conn) {
		p.closeConnLocked(conn)
		return nil
	}
	
	// Check if we have too many idle connections
	if len(p.idle) >= p.config.MaxIdleConns {
		p.stats.MaxIdleClosed++
		p.closeConnLocked(conn)
		return nil
	}
	
	// Add to idle pool
	p.idle = append(p.idle, conn)
	p.updateStats()
	
	return nil
}

// shouldDiscardConn determines if a connection should be discarded.
func (p *ConnectionPool) shouldDiscardConn(conn *pooledConn) bool {
	now := time.Now()
	
	// Check max lifetime
	if p.config.ConnMaxLifetime > 0 && now.Sub(conn.createdAt) > p.config.ConnMaxLifetime {
		p.stats.MaxLifetimeClosed++
		return true
	}
	
	// Check max idle time
	if p.config.ConnMaxIdleTime > 0 && now.Sub(conn.lastUsed) > p.config.ConnMaxIdleTime {
		p.stats.MaxIdleTimeClosed++
		return true
	}
	
	return false
}

// closeConnLocked closes a connection while holding the lock.
func (p *ConnectionPool) closeConnLocked(conn *pooledConn) {
	conn.Conn.Close()
	atomic.AddInt32(&p.numOpen, -1)
}

// updateStats updates connection statistics.
func (p *ConnectionPool) updateStats() {
	p.stats.MaxOpenConnections = p.config.MaxOpenConns
	p.stats.OpenConnections = int(atomic.LoadInt32(&p.numOpen))
	p.stats.InUse = len(p.active)
	p.stats.Idle = len(p.idle)
}

// Stats returns connection pool statistics.
func (p *ConnectionPool) Stats() ConnectionStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	stats := p.stats
	stats.OpenConnections = int(atomic.LoadInt32(&p.numOpen))
	stats.InUse = len(p.active)
	stats.Idle = len(p.idle)
	
	return stats
}

// Close closes all connections in the pool.
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Stop the cleaner
	close(p.cleanerStop)
	<-p.cleanerDone
	
	// Close all idle connections
	for _, conn := range p.idle {
		conn.Conn.Close()
	}
	p.idle = nil
	
	// Close all active connections
	for conn := range p.active {
		conn.Conn.Close()
	}
	p.active = make(map[*pooledConn]struct{})
	
	atomic.StoreInt32(&p.numOpen, 0)
	p.updateStats()
	
	return nil
}

// connectionCleaner periodically cleans up expired connections.
func (p *ConnectionPool) connectionCleaner() {
	defer close(p.cleanerDone)
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			p.cleanExpiredConns()
		case <-p.cleanerStop:
			return
		}
	}
}

// cleanExpiredConns removes expired connections from the idle pool.
func (p *ConnectionPool) cleanExpiredConns() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	now := time.Now()
	var keepIdle []*pooledConn
	
	for _, conn := range p.idle {
		if p.shouldDiscardConn(conn) {
			p.closeConnLocked(conn)
		} else {
			keepIdle = append(keepIdle, conn)
		}
	}
	
	p.idle = keepIdle
	p.updateStats()
}

// Ping tests the connection pool.
func (p *ConnectionPool) Ping(ctx context.Context) error {
	conn, err := p.Get(ctx)
	if err != nil {
		return err
	}
	defer p.Put(conn)
	
	// In a real implementation, this would test the connection
	// For now, just check if we can get a connection
	return nil
}

// SetMaxIdleConns sets the maximum number of connections in the idle pool.
func (p *ConnectionPool) SetMaxIdleConns(n int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.config.MaxIdleConns = n
	
	// Close excess idle connections
	for len(p.idle) > n {
		conn := p.idle[len(p.idle)-1]
		p.idle = p.idle[:len(p.idle)-1]
		p.stats.MaxIdleClosed++
		p.closeConnLocked(conn)
	}
	
	p.updateStats()
}

// SetMaxOpenConns sets the maximum number of open connections.
func (p *ConnectionPool) SetMaxOpenConns(n int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.config.MaxOpenConns = n
	p.updateStats()
}

// SetConnMaxLifetime sets the maximum lifetime for connections.
func (p *ConnectionPool) SetConnMaxLifetime(d time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.config.ConnMaxLifetime = d
}

// SetConnMaxIdleTime sets the maximum idle time for connections.
func (p *ConnectionPool) SetConnMaxIdleTime(d time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.config.ConnMaxIdleTime = d
}

// pooledConn methods

// Close returns the connection to the pool.
func (c *pooledConn) Close() error {
	return c.pool.putConn(c)
}

// PooledTransport wraps a transport with connection pooling.
type PooledTransport struct {
	pool *ConnectionPool
}

// NewPooledTransport creates a transport with connection pooling.
func NewPooledTransport(config *PoolConfig, factory func(context.Context) (Conn, error)) *PooledTransport {
	return &PooledTransport{
		pool: NewConnectionPool(config, factory),
	}
}

// Dial gets a connection from the pool.
func (t *PooledTransport) Dial(ctx context.Context) (Conn, error) {
	return t.pool.Get(ctx)
}

// Close closes the connection pool.
func (t *PooledTransport) Close() error {
	return t.pool.Close()
}

// Stats returns pool statistics.
func (t *PooledTransport) Stats() ConnectionStats {
	return t.pool.Stats()
}

// Ping tests the transport.
func (t *PooledTransport) Ping(ctx context.Context) error {
	return t.pool.Ping(ctx)
}