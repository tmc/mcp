package mcp

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// These tests stress the concurrency and security fixes from the pre-v1
// hardening pass. The map-write paths would, before the fix, trigger an
// unrecoverable "fatal error: concurrent map writes" and kill the test binary;
// run under -race they also surface as data races. They are deliberately
// goroutine-heavy and short.

// TestValidateAccessToken_ConcurrentExpired exercises the expired-token
// eviction path (auth.go) from many goroutines at once. The delete must happen
// under a write lock, not the read lock, or this races / crashes.
func TestValidateAccessToken_ConcurrentExpired(t *testing.T) {
	p := NewMemoryOAuthProvider()
	const n = 200
	// Seed n already-expired tokens directly (internal test).
	for i := 0; i < n; i++ {
		tok := tokenName(i)
		p.accessTokens[tok] = &AccessToken{
			AccessToken: tok,
			ExpiresAt:   time.Now().Add(-time.Hour),
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		tok := tokenName(i)
		// Two validators race on the same expired token, plus a revoker.
		wg.Add(3)
		go func() { defer wg.Done(); p.ValidateAccessToken(context.Background(), tok) }()
		go func() { defer wg.Done(); p.ValidateAccessToken(context.Background(), tok) }()
		go func() { defer wg.Done(); p.RevokeToken(context.Background(), tok) }()
	}
	wg.Wait()
}

// TestTokenTransmissionGuard_NonceReplay asserts the replay protection holds
// under concurrency: of many concurrent validations of the SAME transmission
// (same nonce), exactly one is accepted. A Load-then-Store would let several
// through.
func TestTokenTransmissionGuard_NonceReplay(t *testing.T) {
	g := NewTokenTransmissionGuard(time.Minute)
	defer g.Close()

	transmitted, err := g.PrepareTokenForTransmission("secret-token")
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}

	const n = 64
	var accepted atomic.Int64
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			if _, err := g.ValidateTokenTransmission(transmitted); err == nil {
				accepted.Add(1)
			}
		}()
	}
	wg.Wait()

	if got := accepted.Load(); got != 1 {
		t.Fatalf("nonce replay: accepted %d concurrent replays, want exactly 1", got)
	}
}

// TestTokenTransmissionGuard_Close confirms Close stops the cleanup goroutine
// and is idempotent.
func TestTokenTransmissionGuard_Close(t *testing.T) {
	g := NewTokenTransmissionGuard(time.Minute)
	if err := g.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	// Second close must not panic on a closed channel.
	if err := g.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
}

// TestPooledConnectionWrapper_ConcurrentClose drives Read/Write/Close on the
// pooled wrapper from multiple goroutines. The closed flag must be atomic and
// Close must return the connection at most once.
func TestPooledConnectionWrapper_ConcurrentClose(t *testing.T) {
	pool := NewConnectionPool(nil, nil, nil)
	w := &pooledConnectionWrapper{
		conn: &PooledConnection{conn: nopConn{}, inUse: true},
		pool: pool,
	}
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(3)
		go func() { defer wg.Done(); w.Read(make([]byte, 1)) }()
		go func() { defer wg.Done(); w.Write([]byte{0}) }()
		go func() { defer wg.Done(); w.Close() }()
	}
	wg.Wait()
}

// TestStreamMessages_ConcurrentStreams runs streamMessages on distinct streams
// concurrently while Write publishes, stressing the t.signals map registration
// that previously happened under a read lock.
func TestStreamMessages_ConcurrentStreams(t *testing.T) {
	tr := newStreamableServerTransport("s", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		sid := streamID(i)
		wg.Add(2)
		go func() {
			defer wg.Done()
			rw := &flushRecorder{}
			tr.streamMessages(ctx, rw, rw, sid, 0)
		}()
		go func() {
			defer wg.Done()
			tr.Write(context.Background(), JSONRPCMessage{JSONRPC: "2.0", Method: "note"})
		}()
	}
	wg.Wait()
}

// helpers

func tokenName(i int) string { return "tok-" + strconv.Itoa(i) }

type nopConn struct{}

func (nopConn) Read(p []byte) (int, error)  { return 0, nil }
func (nopConn) Write(p []byte) (int, error) { return len(p), nil }
func (nopConn) Close() error                { return nil }

// flushRecorder is a minimal http.ResponseWriter + http.Flusher for driving
// streamMessages in tests.
type flushRecorder struct {
	mu sync.Mutex
	h  http.Header
}

func (f *flushRecorder) Header() http.Header {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.h == nil {
		f.h = make(http.Header)
	}
	return f.h
}
func (f *flushRecorder) Write(p []byte) (int, error) { return len(p), nil }
func (f *flushRecorder) WriteHeader(int)             {}
func (f *flushRecorder) Flush()                      {}
