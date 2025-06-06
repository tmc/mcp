package fuzz

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// CoordinatorConfig holds configuration for the multi-modal coordinator
type CoordinatorConfig struct {
	MaxConcurrentSessions int           `json:"max_concurrent_sessions"`
	SessionTimeout        time.Duration `json:"session_timeout"`
	AdaptiveWeight        float64       `json:"adaptive_weight"`
	QualityThreshold      float64       `json:"quality_threshold"`
	TargetRefreshInterval time.Duration `json:"target_refresh_interval"`
	MetricsUpdateInterval time.Duration `json:"metrics_update_interval"`
	EnableLLMGuidance     bool          `json:"enable_llm_guidance"`
	EnableAdaptiveWeights bool          `json:"enable_adaptive_weights"`
}

// DefaultConfig returns default coordinator configuration
func DefaultConfig() *CoordinatorConfig {
	return &CoordinatorConfig{
		MaxConcurrentSessions: 5,
		SessionTimeout:        30 * time.Minute,
		AdaptiveWeight:        0.3,
		QualityThreshold:      0.7,
		TargetRefreshInterval: 1 * time.Minute,
		MetricsUpdateInterval: 30 * time.Second,
		EnableLLMGuidance:     true,
		EnableAdaptiveWeights: true,
	}
}

// LoadConfig loads configuration from file
func LoadConfig(configPath string) (*CoordinatorConfig, error) {
	if configPath == "" {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config CoordinatorConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// SaveConfig saves configuration to file
func (c *CoordinatorConfig) SaveConfig(configPath string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	return os.WriteFile(configPath, data, 0644)
}

// SessionManager manages fuzzing sessions
type SessionManager struct {
	coordinator *MultiModalCoordinator
	sessions    map[string]*FuzzingSession
	config      *CoordinatorConfig
}

// NewSessionManager creates a new session manager
func NewSessionManager(coordinator *MultiModalCoordinator, config *CoordinatorConfig) *SessionManager {
	return &SessionManager{
		coordinator: coordinator,
		sessions:    make(map[string]*FuzzingSession),
		config:      config,
	}
}

// CreateSession creates a new fuzzing session
func (sm *SessionManager) CreateSession(ctx context.Context, sessionID string, options map[string]interface{}) (*FuzzingSession, error) {
	session, err := sm.coordinator.StartFuzzingSession(sessionID)
	if err != nil {
		return nil, err
	}

	// Apply session options
	if options != nil {
		session.Metadata = options
	}

	sm.sessions[sessionID] = session
	return session, nil
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(sessionID string) (*FuzzingSession, bool) {
	session, exists := sm.sessions[sessionID]
	return session, exists
}

// ListSessions returns all active sessions
func (sm *SessionManager) ListSessions() []*FuzzingSession {
	sessions := make([]*FuzzingSession, 0, len(sm.sessions))
	for _, session := range sm.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// UpdateSession updates session progress
func (sm *SessionManager) UpdateSession(sessionID string, progress *SessionProgress) error {
	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	return sm.coordinator.UpdateSessionProgress(
		sessionID,
		progress.Iterations,
		progress.SuccessfulTests,
		progress.CoverageGained,
		progress.QualityScore,
	)
}

// CompleteSession completes a session
func (sm *SessionManager) CompleteSession(sessionID string, status SessionStatus) error {
	if err := sm.coordinator.CompleteSession(sessionID, status); err != nil {
		return err
	}

	delete(sm.sessions, sessionID)
	return nil
}

// SessionProgress represents progress update for a session
type SessionProgress struct {
	Iterations      int64   `json:"iterations"`
	SuccessfulTests int64   `json:"successful_tests"`
	CoverageGained  float64 `json:"coverage_gained"`
	QualityScore    float64 `json:"quality_score"`
}

// TargetManager manages fuzzing targets
type TargetManager struct {
	targets     []*FuzzTarget
	coordinator *MultiModalCoordinator
}

// NewTargetManager creates a new target manager
func NewTargetManager(coordinator *MultiModalCoordinator) *TargetManager {
	return &TargetManager{
		targets:     make([]*FuzzTarget, 0),
		coordinator: coordinator,
	}
}

// AddTarget adds a new fuzzing target
func (tm *TargetManager) AddTarget(target *FuzzTarget) {
	tm.targets = append(tm.targets, target)
}

// GetTargets returns all targets
func (tm *TargetManager) GetTargets() []*FuzzTarget {
	return tm.targets
}

// GetTargetByID retrieves a target by ID
func (tm *TargetManager) GetTargetByID(id string) (*FuzzTarget, bool) {
	for _, target := range tm.targets {
		if target.ID == id {
			return target, true
		}
	}
	return nil, false
}

// UpdateTarget updates a target's information
func (tm *TargetManager) UpdateTarget(target *FuzzTarget) error {
	for i, t := range tm.targets {
		if t.ID == target.ID {
			tm.targets[i] = target
			return nil
		}
	}
	return fmt.Errorf("target %s not found", target.ID)
}

// RemoveTarget removes a target
func (tm *TargetManager) RemoveTarget(id string) error {
	for i, target := range tm.targets {
		if target.ID == id {
			tm.targets = append(tm.targets[:i], tm.targets[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("target %s not found", id)
}

// StrategyManager manages fuzzing strategies
type StrategyManager struct {
	strategies  []*GuidanceStrategy
	coordinator *MultiModalCoordinator
}

// NewStrategyManager creates a new strategy manager
func NewStrategyManager(coordinator *MultiModalCoordinator) *StrategyManager {
	return &StrategyManager{
		strategies:  make([]*GuidanceStrategy, 0),
		coordinator: coordinator,
	}
}

// AddStrategy adds a new strategy
func (sm *StrategyManager) AddStrategy(strategy *GuidanceStrategy) {
	sm.strategies = append(sm.strategies, strategy)
}

// GetStrategies returns all strategies
func (sm *StrategyManager) GetStrategies() []*GuidanceStrategy {
	return sm.strategies
}

// GetStrategyByName retrieves a strategy by name
func (sm *StrategyManager) GetStrategyByName(name string) (*GuidanceStrategy, bool) {
	for _, strategy := range sm.strategies {
		if strategy.Name == name {
			return strategy, true
		}
	}
	return nil, false
}

// EnableStrategy enables a strategy
func (sm *StrategyManager) EnableStrategy(name string) error {
	strategy, exists := sm.GetStrategyByName(name)
	if !exists {
		return fmt.Errorf("strategy %s not found", name)
	}
	strategy.Enabled = true
	return nil
}

// DisableStrategy disables a strategy
func (sm *StrategyManager) DisableStrategy(name string) error {
	strategy, exists := sm.GetStrategyByName(name)
	if !exists {
		return fmt.Errorf("strategy %s not found", name)
	}
	strategy.Enabled = false
	return nil
}

// MetricsCollector collects and manages metrics
type MetricsCollector struct {
	coordinator *MultiModalCoordinator
	metrics     map[string]interface{}
	lastUpdate  time.Time
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(coordinator *MultiModalCoordinator) *MetricsCollector {
	return &MetricsCollector{
		coordinator: coordinator,
		metrics:     make(map[string]interface{}),
		lastUpdate:  time.Now(),
	}
}

// CollectMetrics collects current metrics
func (mc *MetricsCollector) CollectMetrics() map[string]interface{} {
	metrics := mc.coordinator.GetMetrics()

	// Add collection metadata
	metrics["collection_time"] = time.Now()
	metrics["uptime"] = time.Since(mc.lastUpdate)

	mc.metrics = metrics
	mc.lastUpdate = time.Now()

	return metrics
}

// GetMetrics returns cached metrics
func (mc *MetricsCollector) GetMetrics() map[string]interface{} {
	return mc.metrics
}

// ExportMetrics exports metrics to file
func (mc *MetricsCollector) ExportMetrics(filepath string) error {
	metrics := mc.CollectMetrics()

	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	return os.WriteFile(filepath, data, 0644)
}

// Logger provides structured logging for fuzzing operations
type Logger struct {
	enabled bool
	prefix  string
}

// NewLogger creates a new logger
func NewLogger(prefix string) *Logger {
	return &Logger{
		enabled: os.Getenv("FUZZ_DEBUG") != "",
		prefix:  prefix,
	}
}

// Log logs a message with context
func (l *Logger) Log(level, message string, context map[string]interface{}) {
	if !l.enabled {
		return
	}

	contextJSON, _ := json.Marshal(context)
	log.Printf("[%s] %s: %s - %s", l.prefix, level, message, string(contextJSON))
}

// LogInfo logs an info message
func (l *Logger) LogInfo(message string, context map[string]interface{}) {
	l.Log("INFO", message, context)
}

// LogWarn logs a warning message
func (l *Logger) LogWarn(message string, context map[string]interface{}) {
	l.Log("WARN", message, context)
}

// LogError logs an error message
func (l *Logger) LogError(message string, context map[string]interface{}) {
	l.Log("ERROR", message, context)
}

// LogDebug logs a debug message
func (l *Logger) LogDebug(message string, context map[string]interface{}) {
	l.Log("DEBUG", message, context)
}

// Utility functions

// GenerateSessionID generates a unique session ID
func GenerateSessionID() string {
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}

// GenerateTargetID generates a unique target ID
func GenerateTargetID(pkg, fn string) string {
	return fmt.Sprintf("%s.%s_%d", pkg, fn, time.Now().UnixNano())
}

// ValidateSession validates session parameters
func ValidateSession(session *FuzzingSession) error {
	if session.ID == "" {
		return fmt.Errorf("session ID is required")
	}
	if session.CurrentTarget == nil {
		return fmt.Errorf("session target is required")
	}
	if session.ActiveStrategy == nil {
		return fmt.Errorf("session strategy is required")
	}
	return nil
}

// ValidateTarget validates target parameters
func ValidateTarget(target *FuzzTarget) error {
	if target.ID == "" {
		return fmt.Errorf("target ID is required")
	}
	if target.Package == "" {
		return fmt.Errorf("target package is required")
	}
	if target.Function == "" {
		return fmt.Errorf("target function is required")
	}
	if target.Priority < 0 || target.Priority > 1 {
		return fmt.Errorf("target priority must be between 0 and 1")
	}
	return nil
}

// ValidateStrategy validates strategy parameters
func ValidateStrategy(strategy *GuidanceStrategy) error {
	if strategy.Name == "" {
		return fmt.Errorf("strategy name is required")
	}
	if strategy.Priority < 0 || strategy.Priority > 1 {
		return fmt.Errorf("strategy priority must be between 0 and 1")
	}
	return nil
}

// ConvertSessionStatus converts session status to string
func ConvertSessionStatus(status SessionStatus) string {
	switch status {
	case StatusPending:
		return "pending"
	case StatusRunning:
		return "running"
	case StatusPaused:
		return "paused"
	case StatusCompleted:
		return "completed"
	case StatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// ConvertGuidanceMode converts guidance mode to string
func ConvertGuidanceMode(mode GuidanceMode) string {
	switch mode {
	case GuidanceCoverage:
		return "coverage"
	case GuidanceSemantic:
		return "semantic"
	case GuidanceLLM:
		return "llm"
	case GuidanceHybrid:
		return "hybrid"
	case GuidanceAdaptive:
		return "adaptive"
	default:
		return "unknown"
	}
}
