package fuzz

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime/coverage"
	"time"
)

// Global coordinator instance
var globalCoordinator *MultiModalCoordinator

// InitializeEnhancedFuzzing sets up the enhanced fuzzing system
func InitializeEnhancedFuzzing(ctx context.Context) (*MultiModalCoordinator, error) {
	if globalCoordinator != nil {
		return globalCoordinator, nil
	}

	// Load configuration
	configPath := os.Getenv("FUZZ_COORDINATOR_CONFIG")
	config, err := LoadConfig(configPath)
	if err != nil {
		log.Printf("Warning: Failed to load coordinator config: %v", err)
		config = DefaultConfig()
	}

	// Create coordinator
	globalCoordinator = NewMultiModalCoordinator(ctx)

	// Configure with loaded settings
	if err := configureCoordinator(globalCoordinator, config); err != nil {
		return nil, fmt.Errorf("failed to configure coordinator: %w", err)
	}

	// Initialize coverage oracle integration
	if config.EnableLLMGuidance {
		if err := setupCoverageIntegration(globalCoordinator); err != nil {
			log.Printf("Warning: Failed to setup coverage integration: %v", err)
		}
	}

	// Set up event handlers for debugging and logging
	setupEventHandlers(globalCoordinator)

	// Initialize default targets and strategies
	if err := initializeDefaultTargets(globalCoordinator); err != nil {
		log.Printf("Warning: Failed to initialize default targets: %v", err)
	}

	log.Printf("Enhanced fuzzing coordinator initialized with config: %+v", config)
	return globalCoordinator, nil
}

// GetGlobalCoordinator returns the global coordinator instance
func GetGlobalCoordinator() *MultiModalCoordinator {
	return globalCoordinator
}

// configureCoordinator applies configuration to the coordinator
func configureCoordinator(coordinator *MultiModalCoordinator, config *CoordinatorConfig) error {
	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()

	coordinator.maxConcurrentSessions = config.MaxConcurrentSessions
	coordinator.sessionTimeout = config.SessionTimeout
	coordinator.adaptiveWeight = config.AdaptiveWeight
	coordinator.qualityThreshold = config.QualityThreshold

	return nil
}

// setupCoverageIntegration connects the coordinator with the coverage oracle
func setupCoverageIntegration(coordinator *MultiModalCoordinator) error {
	// Get the enhanced coverage coordinator
	coverageCoordinator := coverage.GetCoordinator()

	// Initialize the default LLM oracle
	oracleConfig := &coverage.OracleConfig{
		MaxRequestsPerMinute: 60,
		CacheTTL:             5 * time.Minute,
		MaxCacheSize:         1000,
		Model:                "gpt-4",
		Temperature:          0.3,
		MaxTokens:            1000,
		EnableHeuristics:     true,
		HeuristicWeight:      0.7,
	}

	oracle := coverage.NewDefaultLLMOracle(oracleConfig)
	coverageCoordinator.SetLLMOracle(oracle)
	coordinator.SetCoverageOracle(oracle)

	// Set up coverage-driven target refresh
	coverage.RegisterFuzzingHook(func(snapshot *coverage.CoverageSnapshot, guidance *coverage.CoverageGuidance) {
		// Convert coverage guidance to fuzz targets
		targets := coordinator.convertGuidanceToTargets(guidance)

		coordinator.mu.Lock()
		coordinator.targets = append(coordinator.targets, targets...)
		coordinator.mu.Unlock()

		if os.Getenv("FUZZ_DEBUG") != "" {
			log.Printf("Coverage-driven target update: added %d targets", len(targets))
		}
	})

	return nil
}

// setupEventHandlers configures event handlers for logging and debugging
func setupEventHandlers(coordinator *MultiModalCoordinator) {
	logger := NewLogger("FuzzCoordinator")

	coordinator.SetEventHandlers(
		// onSessionStart
		func(session *FuzzingSession) {
			logger.LogInfo("Fuzzing session started", map[string]interface{}{
				"session_id": session.ID,
				"target":     session.CurrentTarget.ID,
				"strategy":   session.ActiveStrategy.Name,
			})
		},
		// onSessionComplete
		func(session *FuzzingSession) {
			logger.LogInfo("Fuzzing session completed", map[string]interface{}{
				"session_id":       session.ID,
				"total_iterations": session.TotalIterations,
				"successful_tests": session.SuccessfulTests,
				"coverage_gained":  session.CoverageGained,
				"quality_score":    session.QualityScore,
				"status":           ConvertSessionStatus(session.Status),
			})
		},
		// onTargetChange
		func(target *FuzzTarget) {
			logger.LogDebug("Target updated", map[string]interface{}{
				"target_id":    target.ID,
				"priority":     target.Priority,
				"success_rate": target.SuccessRate,
				"last_tested":  target.LastTested,
			})
		},
		// onStrategyChange
		func(strategy *GuidanceStrategy) {
			logger.LogDebug("Strategy updated", map[string]interface{}{
				"strategy_name": strategy.Name,
				"mode":          ConvertGuidanceMode(strategy.Mode),
				"priority":      strategy.Priority,
				"enabled":       strategy.Enabled,
			})
		},
	)
}

// initializeDefaultTargets creates initial fuzzing targets
func initializeDefaultTargets(coordinator *MultiModalCoordinator) error {
	// Create some default targets based on common Go patterns
	defaultTargets := []*FuzzTarget{
		{
			ID:          "runtime.GC",
			Package:     "runtime",
			Function:    "GC",
			Priority:    0.6,
			Complexity:  0.8,
			Reason:      "Critical runtime function",
			LastTested:  time.Time{},
			SuccessRate: 0.0,
			Metadata: map[string]interface{}{
				"category": "runtime",
				"critical": true,
			},
		},
		{
			ID:          "net/http.HandleFunc",
			Package:     "net/http",
			Function:    "HandleFunc",
			Priority:    0.7,
			Complexity:  0.6,
			Reason:      "Common HTTP handler registration",
			LastTested:  time.Time{},
			SuccessRate: 0.0,
			Metadata: map[string]interface{}{
				"category": "networking",
				"common":   true,
			},
		},
		{
			ID:          "encoding/json.Marshal",
			Package:     "encoding/json",
			Function:    "Marshal",
			Priority:    0.8,
			Complexity:  0.7,
			Reason:      "Critical JSON serialization",
			LastTested:  time.Time{},
			SuccessRate: 0.0,
			Metadata: map[string]interface{}{
				"category": "serialization",
				"critical": true,
			},
		},
		{
			ID:          "os.Open",
			Package:     "os",
			Function:    "Open",
			Priority:    0.7,
			Complexity:  0.5,
			Reason:      "File system interaction",
			LastTested:  time.Time{},
			SuccessRate: 0.0,
			Metadata: map[string]interface{}{
				"category": "filesystem",
				"common":   true,
			},
		},
	}

	coordinator.mu.Lock()
	coordinator.targets = append(coordinator.targets, defaultTargets...)
	coordinator.mu.Unlock()

	return nil
}

// DemoEnhancedFuzzing demonstrates the enhanced fuzzing capabilities
func DemoEnhancedFuzzing(ctx context.Context) error {
	log.Println("=== Enhanced Fuzzing Demo ===")

	// Initialize the coordinator
	coordinator, err := InitializeEnhancedFuzzing(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize fuzzing: %w", err)
	}

	// Create a demo session
	sessionID := GenerateSessionID()
	session, err := coordinator.StartFuzzingSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}

	log.Printf("Started fuzzing session: %s", session.ID)
	log.Printf("Target: %s", session.CurrentTarget.ID)
	log.Printf("Strategy: %s (%s)", session.ActiveStrategy.Name, ConvertGuidanceMode(session.ActiveStrategy.Mode))

	// Simulate fuzzing progress
	for i := 0; i < 5; i++ {
		time.Sleep(2 * time.Second)

		// Simulate some fuzzing iterations
		iterations := int64(100 + i*50)
		successful := int64(20 + i*10)
		coverage := 0.05 + float64(i)*0.02
		quality := 0.6 + float64(i)*0.05

		err := coordinator.UpdateSessionProgress(sessionID, iterations, successful, coverage, quality)
		if err != nil {
			log.Printf("Failed to update session progress: %v", err)
			continue
		}

		log.Printf("Progress update %d: %d iterations, %d successful, %.2f coverage, %.2f quality",
			i+1, iterations, successful, coverage, quality)
	}

	// Get recommendations
	targets, strategies, err := coordinator.GetRecommendations(ctx)
	if err != nil {
		log.Printf("Failed to get recommendations: %v", err)
	} else {
		log.Printf("Recommendations:")
		log.Printf("  Priority Targets: %d", len(targets))
		for _, target := range targets {
			log.Printf("    - %s (priority: %.2f, reason: %s)", target.ID, target.Priority, target.Reason)
		}
		log.Printf("  Recommended Strategies: %d", len(strategies))
		for _, strategy := range strategies {
			log.Printf("    - %s (%s, priority: %.2f)", strategy.Name, ConvertGuidanceMode(strategy.Mode), strategy.Priority)
		}
	}

	// Complete the session
	err = coordinator.CompleteSession(sessionID, StatusCompleted)
	if err != nil {
		log.Printf("Failed to complete session: %v", err)
	} else {
		log.Printf("Session %s completed successfully", sessionID)
	}

	// Show final metrics
	metrics := coordinator.GetMetrics()
	log.Printf("Final Metrics:")
	log.Printf("  Total Sessions: %v", metrics["total_sessions"])
	log.Printf("  Total Iterations: %v", metrics["total_iterations"])
	log.Printf("  Total Coverage Gained: %v", metrics["total_coverage_gained"])
	log.Printf("  Targets Count: %v", metrics["targets_count"])
	log.Printf("  Strategies Count: %v", metrics["strategies_count"])

	log.Println("=== Demo Complete ===")
	return nil
}

// CreateFuzzingPipeline creates a complete fuzzing pipeline with all components
func CreateFuzzingPipeline(ctx context.Context) (*FuzzingPipeline, error) {
	// Initialize coordinator
	coordinator, err := InitializeEnhancedFuzzing(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize coordinator: %w", err)
	}

	// Create component managers
	sessionManager := NewSessionManager(coordinator, DefaultConfig())
	targetManager := NewTargetManager(coordinator)
	strategyManager := NewStrategyManager(coordinator)
	metricsCollector := NewMetricsCollector(coordinator)

	pipeline := &FuzzingPipeline{
		Coordinator:      coordinator,
		SessionManager:   sessionManager,
		TargetManager:    targetManager,
		StrategyManager:  strategyManager,
		MetricsCollector: metricsCollector,
		Logger:           NewLogger("FuzzingPipeline"),
	}

	return pipeline, nil
}

// FuzzingPipeline represents a complete fuzzing pipeline
type FuzzingPipeline struct {
	Coordinator      *MultiModalCoordinator
	SessionManager   *SessionManager
	TargetManager    *TargetManager
	StrategyManager  *StrategyManager
	MetricsCollector *MetricsCollector
	Logger           *Logger
}

// Start starts the fuzzing pipeline
func (fp *FuzzingPipeline) Start(ctx context.Context) error {
	fp.Logger.LogInfo("Starting fuzzing pipeline", map[string]interface{}{
		"max_concurrent_sessions": fp.Coordinator.maxConcurrentSessions,
		"targets_count":           len(fp.TargetManager.GetTargets()),
		"strategies_count":        len(fp.StrategyManager.GetStrategies()),
	})

	// Pipeline is now ready to accept fuzzing requests
	return nil
}

// Stop stops the fuzzing pipeline
func (fp *FuzzingPipeline) Stop() error {
	fp.Logger.LogInfo("Stopping fuzzing pipeline", nil)

	// Complete all active sessions
	sessions := fp.SessionManager.ListSessions()
	for _, session := range sessions {
		fp.SessionManager.CompleteSession(session.ID, StatusCompleted)
	}

	// Stop coordinator
	fp.Coordinator.Stop()

	fp.Logger.LogInfo("Fuzzing pipeline stopped", nil)
	return nil
}

// GetStatus returns pipeline status
func (fp *FuzzingPipeline) GetStatus() map[string]interface{} {
	metrics := fp.MetricsCollector.CollectMetrics()

	return map[string]interface{}{
		"active_sessions":  len(fp.SessionManager.ListSessions()),
		"total_targets":    len(fp.TargetManager.GetTargets()),
		"total_strategies": len(fp.StrategyManager.GetStrategies()),
		"metrics":          metrics,
	}
}

// Cleanup performs cleanup operations
func Cleanup() {
	if globalCoordinator != nil {
		globalCoordinator.Stop()
		globalCoordinator = nil
	}
}
