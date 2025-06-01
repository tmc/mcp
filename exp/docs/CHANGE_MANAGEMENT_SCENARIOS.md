# Real-World Change Management Scenarios

This document demonstrates how the intelligent change management system handles various real-world development scenarios.

## Scenario 1: Breaking API Change

### Change Description
"Replace our custom JSON serialization with standard library implementation to improve performance"

### System Analysis
```bash
mcp-change-analyze "Replace custom JSON with standard library"
```

```json
{
  "type": "refactoring",
  "breaking": true,
  "risk": "high",
  "affected": {
    "files": 47,
    "tests": 132,
    "apis": 23
  },
  "compatibility_issues": [
    "Custom date format differs",
    "Null handling changes",
    "Field naming conventions"
  ]
}
```

### Automated Resolution Process

#### Phase 1: Compatibility Layer
```go
// Generated compatibility layer
type JSONCompat struct {
    UseCustomFormat bool
}

func (j *JSONCompat) Marshal(v interface{}) ([]byte, error) {
    if j.UseCustomFormat {
        return customJSON.Marshal(v)
    }
    return json.Marshal(v)
}
```

#### Phase 2: Test Updates
```diff
// Automatically updated test
func TestAPIResponse(t *testing.T) {
-   resp := customJSON.Marshal(data)
+   resp := jsonCompat.Marshal(data)
    
-   assert.Contains(t, string(resp), "custom_field")
+   assert.Contains(t, string(resp), "customField")
}
```

#### Phase 3: Migration Path
```bash
Migration Plan Generated:
1. Add compatibility layer (2 hours)
2. Update 132 tests (1 day)
3. Feature flag rollout (1 week)
4. Remove old implementation (2 days)

Total estimated time: 1.5 weeks
Risk mitigation: Feature flags + gradual rollout
```

## Scenario 2: Security Enhancement

### Change Description
"Implement zero-trust security model for all internal services"

### System Analysis
```bash
mcp-change-analyze "Implement zero-trust security model"
```

```yaml
analysis:
  complexity: extreme
  phases: 5
  estimated_effort: 3-4 weeks
  
  security_requirements:
    - mTLS between all services
    - Service identity verification
    - Request-level authorization
    - Audit logging
    
  affected_components:
    - networking layer
    - service discovery
    - authentication system
    - all microservices (43)
```

### Automated Implementation

#### Phase 1: Infrastructure
```go
// Generated service mesh configuration
type ZeroTrustConfig struct {
    ServiceName string
    Certificate tls.Certificate
    AuthzPolicy Policy
}

func NewZeroTrustService(name string) *Service {
    config := &ZeroTrustConfig{
        ServiceName: name,
        Certificate: generateServiceCert(name),
        AuthzPolicy: loadPolicy(name),
    }
    return wrapWithZeroTrust(config)
}
```

#### Phase 2: Service Updates
```bash
# Automated service modification
mcp-zero-trust-convert --service=user-service

Generated changes:
- Added mTLS client
- Implemented service identity
- Added request signing
- Updated health checks
- Modified all external calls
```

#### Phase 3: Test Generation
```go
// Generated security tests
func TestZeroTrustCommunication(t *testing.T) {
    // Test: Unauthorized service cannot connect
    client := NewClient("unauthorized-service")
    err := client.Call("user-service", "GetUser", req)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "unauthorized")
    
    // Test: Authorized service can connect
    client = NewClient("auth-service", WithCertificate(cert))
    err = client.Call("user-service", "GetUser", req)
    assert.NoError(t, err)
}
```

### Rollout Strategy
```yaml
rollout_plan:
  week_1:
    - Enable mTLS in test environment
    - Update 5 non-critical services
    - Monitor performance impact
    
  week_2:
    - Rollout to 50% of services
    - Enable request authorization
    - Update client libraries
    
  week_3:
    - Complete service migration
    - Enable audit logging
    - Deprecate old auth system
    
  week_4:
    - Remove legacy code
    - Performance optimization
    - Documentation finalization
```

## Scenario 3: Database Migration

### Change Description
"Migrate from PostgreSQL to CockroachDB for global distribution"

### System Analysis
```json
{
  "migration_type": "database",
  "complexity": "high",
  "data_volume": "2.5TB",
  "downtime_required": false,
  "compatibility_score": 0.85,
  
  "incompatibilities": [
    "Serial columns → UUID migration",
    "Stored procedures not supported",
    "Different transaction isolation levels",
    "Geographic partitioning syntax"
  ],
  
  "migration_strategy": "dual-write"
}
```

### Automated Migration Process

#### Phase 1: Schema Conversion
```sql
-- Generated schema modifications
-- PostgreSQL
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE,
    created_at TIMESTAMP DEFAULT NOW()
);

-- CockroachDB equivalent
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email STRING UNIQUE,
    created_at TIMESTAMP DEFAULT current_timestamp()
);
```

#### Phase 2: Dual-Write Implementation
```go
// Generated dual-write wrapper
type DualDBWriter struct {
    postgres   *sql.DB
    cockroach  *sql.DB
    migration  MigrationState
}

func (d *DualDBWriter) CreateUser(user User) error {
    // Write to PostgreSQL (primary)
    if err := d.postgres.Create(&user); err != nil {
        return err
    }
    
    // Async write to CockroachDB
    go func() {
        if err := d.cockroach.Create(&user); err != nil {
            d.logMigrationError(err)
        }
        d.migration.RecordSync("users", user.ID)
    }()
    
    return nil
}
```

#### Phase 3: Test Adaptation
```go
// Automatically adapted tests
func TestUserCreation(t *testing.T) {
    db := getDatabaseConnection()
    
    user := User{Email: "test@example.com"}
    err := db.CreateUser(user)
    assert.NoError(t, err)
    
    // Additional validation for migration
    if migrations.IsDualWriteEnabled() {
        eventually(t, func() bool {
            return checkSyncStatus(user.ID) == "synced"
        }, 5*time.Second)
    }
}
```

### Migration Validation
```bash
mcp-db-migrate-validate --from=postgres --to=cockroach

Validation Report:
✓ Schema compatibility: 85%
✓ Data integrity: 100%
✓ Performance baseline: Established
⚠ Incompatible features: 3 (with workarounds)
✓ Rollback plan: Verified

Ready for production migration? YES
Recommended approach: Gradual geographic rollout
```

## Scenario 4: Performance Optimization

### Change Description
"Optimize API response time to under 100ms for 95th percentile"

### System Analysis
```yaml
current_performance:
  p50: 45ms
  p95: 280ms
  p99: 520ms

bottlenecks_identified:
  - database_queries: 55%
  - json_serialization: 20%
  - network_latency: 15%
  - business_logic: 10%

optimization_targets:
  - query_optimization
  - caching_layer
  - connection_pooling
  - response_compression
```

### Automated Optimization

#### Phase 1: Query Optimization
```go
// Generated optimized queries
// Before
func GetUserOrders(userID string) ([]Order, error) {
    return db.Query(`
        SELECT * FROM orders 
        WHERE user_id = $1 
        ORDER BY created_at DESC
    `, userID)
}

// After (with analysis-driven indexes)
func GetUserOrdersOptimized(userID string) ([]Order, error) {
    // Added: Composite index on (user_id, created_at)
    // Added: Partial index for active orders
    return db.Query(`
        SELECT id, user_id, status, total, created_at 
        FROM orders 
        WHERE user_id = $1 
          AND status != 'cancelled'
        ORDER BY created_at DESC
        LIMIT 100
    `, userID)
}
```

#### Phase 2: Caching Layer
```go
// Generated caching implementation
type CachedAPI struct {
    cache   Cache
    service Service
    config  CacheConfig
}

func (c *CachedAPI) GetUserOrders(ctx context.Context, userID string) ([]Order, error) {
    key := fmt.Sprintf("orders:%s", userID)
    
    // Try cache first
    var orders []Order
    if err := c.cache.Get(ctx, key, &orders); err == nil {
        return orders, nil
    }
    
    // Fetch from service
    orders, err := c.service.GetUserOrders(ctx, userID)
    if err != nil {
        return nil, err
    }
    
    // Cache with intelligent TTL
    ttl := c.calculateTTL(orders)
    c.cache.Set(ctx, key, orders, ttl)
    
    return orders, nil
}
```

#### Phase 3: Performance Tests
```go
// Generated performance benchmarks
func BenchmarkAPIResponse(b *testing.B) {
    scenarios := []struct {
        name string
        req  Request
    }{
        {"small_payload", smallRequest},
        {"medium_payload", mediumRequest},
        {"large_payload", largeRequest},
    }
    
    for _, s := range scenarios {
        b.Run(s.name, func(b *testing.B) {
            for i := 0; i < b.N; i++ {
                resp, err := api.Handle(s.req)
                require.NoError(b, err)
                require.NotNil(b, resp)
            }
        })
    }
}
```

### Performance Validation
```bash
mcp-perf-validate --target="p95 < 100ms"

Performance Test Results:
========================
Before optimization:
  p50: 45ms  | p95: 280ms | p99: 520ms

After optimization:
  p50: 22ms  | p95: 85ms  | p99: 142ms

Target achieved: ✓ (p95: 85ms < 100ms)
Improvement: 69.6% reduction in p95 latency

Recommendations:
- Enable caching in production with monitoring
- Set up alerts for cache hit ratio < 80%
- Schedule weekly performance reviews
```

## Scenario 5: Feature Addition

### Change Description
"Add real-time notifications for all user actions using WebSockets"

### System Analysis
```json
{
  "feature_type": "real-time",
  "complexity": "medium",
  "new_dependencies": ["websocket", "message-queue"],
  "scalability_concerns": ["connection_limit", "message_throughput"],
  
  "implementation_requirements": {
    "websocket_server": true,
    "message_broker": "redis_pubsub",
    "client_libraries": ["js", "ios", "android"],
    "fallback_mechanism": "polling"
  }
}
```

### Automated Implementation

#### Phase 1: WebSocket Server
```go
// Generated WebSocket server
type NotificationServer struct {
    upgrader websocket.Upgrader
    hub      *Hub
    broker   MessageBroker
}

func (s *NotificationServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := s.upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Error("WebSocket upgrade failed:", err)
        return
    }
    
    client := &Client{
        conn:   conn,
        send:   make(chan []byte, 256),
        hub:    s.hub,
        userID: extractUserID(r),
    }
    
    s.hub.register <- client
    
    go client.writePump()
    go client.readPump()
}
```

#### Phase 2: Event Integration
```go
// Generated event publisher
type EventPublisher struct {
    broker MessageBroker
}

func (p *EventPublisher) PublishUserAction(action UserAction) error {
    notification := Notification{
        Type:      "user_action",
        UserID:    action.UserID,
        Action:    action.Type,
        Timestamp: time.Now(),
        Data:      action.Data,
    }
    
    return p.broker.Publish(
        fmt.Sprintf("user:%s:notifications", action.UserID),
        notification,
    )
}

// Automatically added to existing code
func (s *UserService) UpdateProfile(ctx context.Context, req UpdateRequest) error {
    // Existing logic
    if err := s.db.UpdateUser(req); err != nil {
        return err
    }
    
    // Added: Publish notification
    s.events.PublishUserAction(UserAction{
        UserID: req.UserID,
        Type:   "profile_updated",
        Data:   map[string]interface{}{"fields": req.Changes},
    })
    
    return nil
}
```

#### Phase 3: Client Libraries
```javascript
// Generated JavaScript client
class NotificationClient {
    constructor(url, options = {}) {
        this.url = url;
        this.options = options;
        this.ws = null;
        this.listeners = new Map();
        this.reconnectAttempts = 0;
    }
    
    connect() {
        this.ws = new WebSocket(this.url);
        
        this.ws.onmessage = (event) => {
            const notification = JSON.parse(event.data);
            this.emit(notification.type, notification);
        };
        
        this.ws.onclose = () => {
            this.scheduleReconnect();
        };
        
        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
            this.fallbackToPolling();
        };
    }
    
    on(event, handler) {
        if (!this.listeners.has(event)) {
            this.listeners.set(event, []);
        }
        this.listeners.get(event).push(handler);
    }
    
    emit(event, data) {
        const handlers = this.listeners.get(event) || [];
        handlers.forEach(handler => handler(data));
    }
}
```

### Integration Tests
```go
func TestRealTimeNotifications(t *testing.T) {
    // Setup WebSocket client
    wsClient := NewTestWSClient(t)
    defer wsClient.Close()
    
    // Subscribe to notifications
    notifications := make(chan Notification, 10)
    wsClient.OnNotification(func(n Notification) {
        notifications <- n
    })
    
    // Trigger user action
    err := userService.UpdateProfile(ctx, UpdateRequest{
        UserID: testUser.ID,
        Changes: map[string]interface{}{
            "name": "New Name",
        },
    })
    require.NoError(t, err)
    
    // Verify notification received
    select {
    case notification := <-notifications:
        assert.Equal(t, "profile_updated", notification.Type)
        assert.Equal(t, testUser.ID, notification.UserID)
    case <-time.After(5 * time.Second):
        t.Fatal("Notification not received within timeout")
    }
}
```

## Common Patterns

### 1. Gradual Rollout Pattern
```yaml
pattern: gradual_rollout
applicable_to: [breaking_changes, performance_critical, high_risk]

stages:
  - name: canary
    percentage: 1%
    duration: 1 day
    rollback_threshold: 5% error rate
    
  - name: pilot
    percentage: 10%
    duration: 3 days
    rollback_threshold: 2% error rate
    
  - name: general
    percentage: 100%
    duration: permanent
    monitoring: enhanced
```

### 2. Compatibility Bridge Pattern
```go
// Used for API changes
type CompatibilityBridge struct {
    oldImpl OldImplementation
    newImpl NewImplementation
    version string
}

func (b *CompatibilityBridge) Handle(req Request) Response {
    if b.shouldUseOld(req) {
        return b.adaptOldResponse(b.oldImpl.Handle(req))
    }
    return b.newImpl.Handle(req)
}
```

### 3. Test Evolution Pattern
```yaml
pattern: test_evolution
triggers: [api_change, refactoring, feature_addition]

steps:
  - identify_affected_tests
  - generate_compatibility_layer
  - update_assertions
  - add_migration_tests
  - verify_backward_compatibility
  - cleanup_obsolete_tests
```

## Lessons Learned

1. **AI-Driven Analysis**
   - Natural language processing accurately identifies change scope
   - Pattern matching from historical data improves predictions
   - Automated test generation catches edge cases humans miss

2. **Incremental Implementation**
   - Breaking large changes into phases reduces risk
   - Feature flags enable safe rollouts
   - Compatibility layers preserve functionality during transitions

3. **Comprehensive Testing**
   - Automated test updates maintain coverage
   - Performance benchmarks catch regressions early
   - Integration tests verify system-wide impacts

4. **Continuous Learning**
   - System improves with each change
   - Patterns emerge from successful implementations
   - Failure analysis prevents repeated mistakes

These real-world scenarios demonstrate how intelligent change management transforms complex development tasks into manageable, automated workflows.