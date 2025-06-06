// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package fuzz provides enhanced fuzzing coordination with multi-modal guidance mechanisms.
// This extends Go's built-in fuzzing with LLM integration, coverage analysis, and intelligent guidance.
package fuzz

import (
	"context"
	"fmt"
	"math"
	"runtime/coverage"
	"sort"
	"sync"
	"time"
)

// GuidanceMode represents different fuzzing guidance strategies
type GuidanceMode int

const (
	GuidanceCoverage GuidanceMode = iota // Traditional coverage-guided fuzzing
	GuidanceSemantic                     // Semantic understanding of inputs
	GuidanceLLM                          // LLM-assisted guidance
	GuidanceHybrid                       // Multi-modal combination
	GuidanceAdaptive                     // Adaptive strategy selection
)

// FuzzTarget represents a specific area to focus fuzzing efforts
type FuzzTarget struct {
	ID          string                 `json:"id"`
	Package     string                 `json:"package"`
	Function    string                 `json:"function"`
	Priority    float64                `json:"priority"`
	Complexity  float64                `json:"complexity"`
	LastTested  time.Time              `json:"last_tested"`
	SuccessRate float64                `json:"success_rate"`
	Reason      string                 `json:"reason"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// GuidanceStrategy defines how to approach fuzzing a target
type GuidanceStrategy struct {
	Name        string                 `json:"name"`
	Mode        GuidanceMode           `json:"mode"`
	Priority    float64                `json:"priority"`
	Parameters  map[string]interface{} `json:"parameters"`
	Description string                 `json:"description"`
	Enabled     bool                   `json:"enabled"`
}

// FuzzingSession represents an active fuzzing session
type FuzzingSession struct {
	ID              string                 `json:"id"`
	StartTime       time.Time              `json:"start_time"`
	CurrentTarget   *FuzzTarget            `json:"current_target"`
	ActiveStrategy  *GuidanceStrategy      `json:"active_strategy"`
	TotalIterations int64                  `json:"total_iterations"`
	SuccessfulTests int64                  `json:"successful_tests"`
	CoverageGained  float64                `json:"coverage_gained"`
	QualityScore    float64                `json:"quality_score"`
	Status          SessionStatus          `json:"status"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// SessionStatus represents the current state of a fuzzing session
type SessionStatus int

const (
	StatusPending SessionStatus = iota
	StatusRunning
	StatusPaused
	StatusCompleted
	StatusFailed
)

// MultiModalCoordinator manages fuzzing with multiple guidance mechanisms
type MultiModalCoordinator struct {
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc

	// Core components
	coverageOracle coverage.LLMOracle
	strategies     []*GuidanceStrategy
	targets        []*FuzzTarget
	activeSessions map[string]*FuzzingSession

	// Configuration
	maxConcurrentSessions int
	sessionTimeout        time.Duration
	adaptiveWeight        float64
	qualityThreshold      float64

	// Metrics and state
	totalSessions       int64
	totalIterations     int64
	totalCoverageGained float64
	adaptiveWeights     map[GuidanceMode]float64
	strategyPerformance map[string]*StrategyMetrics

	// Event handlers
	onSessionStart    func(*FuzzingSession)
	onSessionComplete func(*FuzzingSession)
	onTargetChange    func(*FuzzTarget)
	onStrategyChange  func(*GuidanceStrategy)

	// Background processing
	wg                  sync.WaitGroup
	targetRefreshTicker *time.Ticker
	metricsUpdateTicker *time.Ticker
}

// StrategyMetrics tracks performance of guidance strategies
type StrategyMetrics struct {
	TotalAttempts      int64         `json:"total_attempts"`
	SuccessfulAttempts int64         `json:"successful_attempts"`
	SuccessRate        float64       `json:"success_rate"`
	AverageCoverage    float64       `json:"average_coverage"`
	AverageQuality     float64       `json:"average_quality"`
	AverageTime        time.Duration `json:"average_time"`
	LastUsed           time.Time     `json:"last_used"`
	Enabled            bool          `json:"enabled"`
}

// NewMultiModalCoordinator creates a new multi-modal fuzzing coordinator
func NewMultiModalCoordinator(ctx context.Context) *MultiModalCoordinator {
	coordinatorCtx, cancel := context.WithCancel(ctx)

	coordinator := &MultiModalCoordinator{
		ctx:    coordinatorCtx,
		cancel: cancel,

		// Initialize collections
		strategies:          make([]*GuidanceStrategy, 0),
		targets:             make([]*FuzzTarget, 0),
		activeSessions:      make(map[string]*FuzzingSession),
		adaptiveWeights:     make(map[GuidanceMode]float64),
		strategyPerformance: make(map[string]*StrategyMetrics),

		// Default configuration
		maxConcurrentSessions: 5,
		sessionTimeout:        30 * time.Minute,
		adaptiveWeight:        0.3,
		qualityThreshold:      0.7,

		// Initialize adaptive weights
		adaptiveWeights: map[GuidanceMode]float64{
			GuidanceCoverage: 0.4,
			GuidanceSemantic: 0.3,
			GuidanceLLM:      0.2,
			GuidanceHybrid:   0.1,
		},
	}

	// Initialize default strategies
	coordinator.initializeDefaultStrategies()

	// Start background processes
	coordinator.startBackgroundProcesses()

	return coordinator
}

// SetCoverageOracle configures the LLM oracle for coverage analysis
func (c *MultiModalCoordinator) SetCoverageOracle(oracle coverage.LLMOracle) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.coverageOracle = oracle
}

// SetEventHandlers configures event callbacks
func (c *MultiModalCoordinator) SetEventHandlers(
	onSessionStart func(*FuzzingSession),
	onSessionComplete func(*FuzzingSession),
	onTargetChange func(*FuzzTarget),
	onStrategyChange func(*GuidanceStrategy),
) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onSessionStart = onSessionStart
	c.onSessionComplete = onSessionComplete
	c.onTargetChange = onTargetChange
	c.onStrategyChange = onStrategyChange
}

// StartFuzzingSession begins a new fuzzing session with intelligent target selection
func (c *MultiModalCoordinator) StartFuzzingSession(sessionID string) (*FuzzingSession, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check concurrent session limit
	if len(c.activeSessions) >= c.maxConcurrentSessions {
		return nil, fmt.Errorf("maximum concurrent sessions (%d) reached", c.maxConcurrentSessions)
	}

	// Select optimal target
	target, err := c.selectOptimalTarget()
	if err != nil {
		return nil, fmt.Errorf("failed to select target: %w", err)
	}

	// Select best strategy for target
	strategy, err := c.selectStrategyForTarget(target)
	if err != nil {
		return nil, fmt.Errorf("failed to select strategy: %w", err)
	}

	// Create fuzzing session
	session := &FuzzingSession{
		ID:             sessionID,
		StartTime:      time.Now(),
		CurrentTarget:  target,
		ActiveStrategy: strategy,
		Status:         StatusRunning,
		Metadata:       make(map[string]interface{}),
	}

	c.activeSessions[sessionID] = session
	c.totalSessions++

	// Update target last tested time
	target.LastTested = time.Now()

	// Trigger event handler
	if c.onSessionStart != nil {
		go c.onSessionStart(session)
	}

	return session, nil
}

// UpdateSessionProgress updates the progress of an active fuzzing session
func (c *MultiModalCoordinator) UpdateSessionProgress(sessionID string, iterations int64, successfulTests int64, coverageGained float64, qualityScore float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	session, exists := c.activeSessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	// Update session metrics
	session.TotalIterations += iterations
	session.SuccessfulTests += successfulTests
	session.CoverageGained += coverageGained
	session.QualityScore = qualityScore

	// Update global metrics
	c.totalIterations += iterations
	c.totalCoverageGained += coverageGained

	// Update strategy performance
	if metrics, exists := c.strategyPerformance[session.ActiveStrategy.Name]; exists {
		metrics.TotalAttempts += iterations
		metrics.SuccessfulAttempts += successfulTests
		metrics.SuccessRate = float64(metrics.SuccessfulAttempts) / float64(metrics.TotalAttempts)
		metrics.AverageCoverage = (metrics.AverageCoverage + coverageGained) / 2
		metrics.AverageQuality = (metrics.AverageQuality + qualityScore) / 2
		metrics.LastUsed = time.Now()
	}

	// Update target success rate
	if session.CurrentTarget != nil {
		session.CurrentTarget.SuccessRate = float64(session.SuccessfulTests) / float64(session.TotalIterations)
	}

	return nil
}

// CompleteSession marks a fuzzing session as completed
func (c *MultiModalCoordinator) CompleteSession(sessionID string, status SessionStatus) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	session, exists := c.activeSessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.Status = status
	delete(c.activeSessions, sessionID)

	// Update adaptive weights based on session performance
	c.updateAdaptiveWeights(session)

	// Trigger event handler
	if c.onSessionComplete != nil {
		go c.onSessionComplete(session)
	}

	return nil
}

// GetRecommendations provides intelligent fuzzing recommendations
func (c *MultiModalCoordinator) GetRecommendations(ctx context.Context) ([]*FuzzTarget, []*GuidanceStrategy, error) {
	c.mu.RLock()
	coverageOracle := c.coverageOracle
	c.mu.RUnlock()

	if coverageOracle == nil {
		return c.getHeuristicRecommendations(), c.getTopStrategies(3), nil
	}

	// Get coverage snapshot
	coverageCoordinator := coverage.GetCoordinator()
	snapshot := coverageCoordinator.TakeSnapshot()

	// Get LLM guidance
	guidance, err := coverageOracle.SuggestCoverageTargets(ctx, snapshot)
	if err != nil {
		return c.getHeuristicRecommendations(), c.getTopStrategies(3), nil
	}

	// Convert guidance to fuzz targets
	targets := c.convertGuidanceToTargets(guidance)
	strategies := c.selectStrategiesForTargets(targets)

	return targets, strategies, nil
}

// GetMetrics returns comprehensive coordinator metrics
func (c *MultiModalCoordinator) GetMetrics() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"total_sessions":        c.totalSessions,
		"active_sessions":       len(c.activeSessions),
		"total_iterations":      c.totalIterations,
		"total_coverage_gained": c.totalCoverageGained,
		"adaptive_weights":      c.adaptiveWeights,
		"strategy_performance":  c.strategyPerformance,
		"targets_count":         len(c.targets),
		"strategies_count":      len(c.strategies),
	}
}

// Private methods

func (c *MultiModalCoordinator) initializeDefaultStrategies() {
	defaultStrategies := []*GuidanceStrategy{
		{
			Name:        "CoverageFocused",
			Mode:        GuidanceCoverage,
			Priority:    0.8,
			Description: "Traditional coverage-guided fuzzing",
			Enabled:     true,
			Parameters: map[string]interface{}{
				"coverage_threshold": 0.1,
				"max_iterations":     10000,
			},
		},
		{
			Name:        "SemanticAware",
			Mode:        GuidanceSemantic,
			Priority:    0.7,
			Description: "Semantic understanding of input structure",
			Enabled:     true,
			Parameters: map[string]interface{}{
				"grammar_guided":     true,
				"type_aware":         true,
				"constraint_solving": true,
			},
		},
		{
			Name:        "LLMAssisted",
			Mode:        GuidanceLLM,
			Priority:    0.6,
			Description: "LLM-guided intelligent fuzzing",
			Enabled:     true,
			Parameters: map[string]interface{}{
				"llm_suggestions":  true,
				"quality_feedback": true,
				"pattern_learning": true,
			},
		},
		{
			Name:        "HybridApproach",
			Mode:        GuidanceHybrid,
			Priority:    0.9,
			Description: "Multi-modal combination of all approaches",
			Enabled:     true,
			Parameters: map[string]interface{}{
				"coverage_weight": 0.4,
				"semantic_weight": 0.3,
				"llm_weight":      0.3,
			},
		},
		{
			Name:        "AdaptiveLearning",
			Mode:        GuidanceAdaptive,
			Priority:    0.5,
			Description: "Adaptive strategy selection based on performance",
			Enabled:     true,
			Parameters: map[string]interface{}{
				"learning_rate":     0.1,
				"exploration_rate":  0.2,
				"adaptation_window": 1000,
			},
		},
	}

	c.strategies = defaultStrategies

	// Initialize strategy metrics
	for _, strategy := range defaultStrategies {
		c.strategyPerformance[strategy.Name] = &StrategyMetrics{
			Enabled: strategy.Enabled,
		}
	}
}

func (c *MultiModalCoordinator) selectOptimalTarget() (*FuzzTarget, error) {
	if len(c.targets) == 0 {
		return nil, fmt.Errorf("no targets available")
	}

	// Sort targets by priority and other factors
	sort.Slice(c.targets, func(i, j int) bool {
		target1, target2 := c.targets[i], c.targets[j]

		// Calculate composite score
		score1 := c.calculateTargetScore(target1)
		score2 := c.calculateTargetScore(target2)

		return score1 > score2
	})

	return c.targets[0], nil
}

func (c *MultiModalCoordinator) calculateTargetScore(target *FuzzTarget) float64 {
	score := target.Priority

	// Boost score for high complexity, low success rate targets
	if target.Complexity > 0.7 && target.SuccessRate < 0.3 {
		score += 0.2
	}

	// Reduce score for recently tested targets
	timeSinceLastTest := time.Since(target.LastTested)
	if timeSinceLastTest < time.Hour {
		score *= 0.8
	}

	return score
}

func (c *MultiModalCoordinator) selectStrategyForTarget(target *FuzzTarget) (*GuidanceStrategy, error) {
	if len(c.strategies) == 0 {
		return nil, fmt.Errorf("no strategies available")
	}

	// Select strategy based on target characteristics and adaptive weights
	bestStrategy := c.strategies[0]
	bestScore := 0.0

	for _, strategy := range c.strategies {
		if !strategy.Enabled {
			continue
		}

		score := c.calculateStrategyScore(strategy, target)
		if score > bestScore {
			bestScore = score
			bestStrategy = strategy
		}
	}

	return bestStrategy, nil
}

func (c *MultiModalCoordinator) calculateStrategyScore(strategy *GuidanceStrategy, target *FuzzTarget) float64 {
	baseScore := strategy.Priority

	// Apply adaptive weight
	if weight, exists := c.adaptiveWeights[strategy.Mode]; exists {
		baseScore *= weight
	}

	// Consider strategy performance
	if metrics, exists := c.strategyPerformance[strategy.Name]; exists {
		baseScore *= (1.0 + metrics.SuccessRate)
	}

	// Target-specific adjustments
	if target.Complexity > 0.8 {
		// High complexity targets benefit from LLM and hybrid approaches
		if strategy.Mode == GuidanceLLM || strategy.Mode == GuidanceHybrid {
			baseScore *= 1.2
		}
	}

	return baseScore
}

func (c *MultiModalCoordinator) updateAdaptiveWeights(session *FuzzingSession) {
	mode := session.ActiveStrategy.Mode

	// Calculate performance factor
	performanceFactor := 1.0
	if session.TotalIterations > 0 {
		successRate := float64(session.SuccessfulTests) / float64(session.TotalIterations)
		performanceFactor = 0.5 + successRate // Range: 0.5 to 1.5
	}

	// Update adaptive weight
	currentWeight := c.adaptiveWeights[mode]
	newWeight := currentWeight + c.adaptiveWeight*(performanceFactor-1.0)

	// Ensure weight stays in reasonable bounds
	if newWeight < 0.1 {
		newWeight = 0.1
	} else if newWeight > 2.0 {
		newWeight = 2.0
	}

	c.adaptiveWeights[mode] = newWeight

	// Normalize weights to sum to 1.0
	c.normalizeAdaptiveWeights()
}

func (c *MultiModalCoordinator) normalizeAdaptiveWeights() {
	totalWeight := 0.0
	for _, weight := range c.adaptiveWeights {
		totalWeight += weight
	}

	if totalWeight > 0 {
		for mode := range c.adaptiveWeights {
			c.adaptiveWeights[mode] /= totalWeight
		}
	}
}

func (c *MultiModalCoordinator) getHeuristicRecommendations() []*FuzzTarget {
	// Return top targets based on heuristics
	recommendations := make([]*FuzzTarget, 0)

	// Sort by priority and add top targets
	sort.Slice(c.targets, func(i, j int) bool {
		return c.calculateTargetScore(c.targets[i]) > c.calculateTargetScore(c.targets[j])
	})

	maxRecommendations := int(math.Min(5, float64(len(c.targets))))
	for i := 0; i < maxRecommendations; i++ {
		recommendations = append(recommendations, c.targets[i])
	}

	return recommendations
}

func (c *MultiModalCoordinator) getTopStrategies(count int) []*GuidanceStrategy {
	// Sort strategies by performance and return top ones
	strategies := make([]*GuidanceStrategy, 0)

	sort.Slice(c.strategies, func(i, j int) bool {
		return c.calculateStrategyScore(c.strategies[i], nil) > c.calculateStrategyScore(c.strategies[j], nil)
	})

	maxStrategies := int(math.Min(float64(count), float64(len(c.strategies))))
	for i := 0; i < maxStrategies; i++ {
		if c.strategies[i].Enabled {
			strategies = append(strategies, c.strategies[i])
		}
	}

	return strategies
}

func (c *MultiModalCoordinator) convertGuidanceToTargets(guidance *coverage.CoverageGuidance) []*FuzzTarget {
	targets := make([]*FuzzTarget, 0)

	for _, coverageTarget := range guidance.PriorityTargets {
		target := &FuzzTarget{
			ID:         fmt.Sprintf("%s.%s", coverageTarget.Package, coverageTarget.Function),
			Package:    coverageTarget.Package,
			Function:   coverageTarget.Function,
			Priority:   coverageTarget.Priority,
			Complexity: coverageTarget.Complexity,
			Reason:     coverageTarget.Reason,
			Metadata: map[string]interface{}{
				"line":            coverageTarget.Line,
				"llm_recommended": true,
				"guidance_source": "coverage_oracle",
			},
		}
		targets = append(targets, target)
	}

	return targets
}

func (c *MultiModalCoordinator) selectStrategiesForTargets(targets []*FuzzTarget) []*GuidanceStrategy {
	strategies := make([]*GuidanceStrategy, 0)

	for _, target := range targets {
		if strategy, err := c.selectStrategyForTarget(target); err == nil {
			strategies = append(strategies, strategy)
		}
	}

	return strategies
}

func (c *MultiModalCoordinator) startBackgroundProcesses() {
	// Start target refresh process
	c.targetRefreshTicker = time.NewTicker(1 * time.Minute)
	c.wg.Add(1)
	go c.targetRefreshProcess()

	// Start metrics update process
	c.metricsUpdateTicker = time.NewTicker(30 * time.Second)
	c.wg.Add(1)
	go c.metricsUpdateProcess()
}

func (c *MultiModalCoordinator) targetRefreshProcess() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-c.targetRefreshTicker.C:
			c.refreshTargets()
		}
	}
}

func (c *MultiModalCoordinator) metricsUpdateProcess() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-c.metricsUpdateTicker.C:
			c.updateMetrics()
		}
	}
}

func (c *MultiModalCoordinator) refreshTargets() {
	// Refresh targets based on coverage data
	if c.coverageOracle != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		coverageCoordinator := coverage.GetCoordinator()
		snapshot := coverageCoordinator.TakeSnapshot()

		if guidance, err := c.coverageOracle.SuggestCoverageTargets(ctx, snapshot); err == nil {
			c.mu.Lock()
			newTargets := c.convertGuidanceToTargets(guidance)
			c.targets = append(c.targets, newTargets...)
			c.mu.Unlock()
		}
	}
}

func (c *MultiModalCoordinator) updateMetrics() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Update strategy metrics based on recent performance
	for _, metrics := range c.strategyPerformance {
		if metrics.TotalAttempts > 0 {
			metrics.SuccessRate = float64(metrics.SuccessfulAttempts) / float64(metrics.TotalAttempts)
		}
	}
}

// Stop shuts down the coordinator
func (c *MultiModalCoordinator) Stop() {
	c.cancel()

	if c.targetRefreshTicker != nil {
		c.targetRefreshTicker.Stop()
	}
	if c.metricsUpdateTicker != nil {
		c.metricsUpdateTicker.Stop()
	}

	c.wg.Wait()
}
