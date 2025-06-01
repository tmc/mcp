package coverage

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"
)

// InitializeEnhancedCoverage sets up the enhanced coverage system with LLM oracle
func InitializeEnhancedCoverage() error {
	coordinator := GetCoordinator()
	
	// Load configuration
	configPath := os.Getenv("COVERAGE_ORACLE_CONFIG")
	if configPath == "" {
		// Look for default config in current directory
		if wd, err := os.Getwd(); err == nil {
			configPath = filepath.Join(wd, "coverage_oracle.json")
		}
	}
	
	config, err := LoadOracleConfig(configPath)
	if os.IsNotExist(err) {
		// Create default config
		config = &OracleConfig{
			MaxRequestsPerMinute: 60,
			CacheTTL:             5 * time.Minute,
			MaxCacheSize:         1000,
			Model:                "gpt-4",
			Temperature:          0.3,
			MaxTokens:            1000,
			EnableHeuristics:     true,
			HeuristicWeight:      0.7,
		}
		
		// Save default config for future use
		if configPath != "" {
			config.Save(configPath)
		}
	} else if err != nil {
		log.Printf("Warning: Failed to load LLM oracle config: %v", err)
		config = nil
	}
	
	// Initialize LLM oracle
	oracle := NewDefaultLLMOracle(config)
	coordinator.SetLLMOracle(oracle)
	
	// Set up event handlers for debugging
	if os.Getenv("COVERAGE_DEBUG") != "" {
		coordinator.SetEventHandlers(
			func(snapshot *CoverageSnapshot) {
				log.Printf("Coverage snapshot: %.2f%% (%d/%d lines)", 
					snapshot.CoverageRatio*100, 
					snapshot.CoveredLines, 
					snapshot.TotalLines)
			},
			func(assessment *TestQualityAssessment) {
				log.Printf("Test quality assessment: %.2f score, %s", 
					assessment.Score, assessment.Rationale)
			},
			func(guidance *CoverageGuidance) {
				log.Printf("Coverage guidance: %d targets, strategies: %v", 
					len(guidance.PriorityTargets), guidance.Strategies)
			},
		)
	}
	
	// Take initial snapshot
	coordinator.TakeSnapshot()
	
	log.Printf("Enhanced coverage system initialized with LLM oracle (heuristics=%v)", 
		config != nil && config.EnableHeuristics)
	
	return nil
}

// Demo function to showcase enhanced coverage capabilities
func DemoEnhancedCoverage() {
	log.Println("=== Enhanced Coverage Demo ===")
	
	coordinator := GetCoordinator()
	
	// Take a snapshot
	snapshot := coordinator.TakeSnapshot()
	log.Printf("Current coverage: %.2f%% (%d/%d lines)", 
		snapshot.CoverageRatio*100, 
		snapshot.CoveredLines, 
		snapshot.TotalLines)
	
	// Get test quality assessment
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	testCase := map[string]interface{}{
		"type": "demo_test",
		"data": []string{"test1", "test2", "test3"},
		"config": map[string]int{"iterations": 100, "timeout": 5000},
	}
	
	if assessment, err := coordinator.EvaluateTestQuality(ctx, testCase); err == nil {
		log.Printf("Test Quality Assessment:")
		log.Printf("  Score: %.2f", assessment.Score)
		log.Printf("  Rationale: %s", assessment.Rationale)
		log.Printf("  Suggestions: %v", assessment.Suggestions)
		log.Printf("  Rubric: %s", assessment.Rubric)
	} else {
		log.Printf("Test quality assessment failed: %v", err)
	}
	
	// Get coverage guidance
	if guidance, err := coordinator.GetCoverageGuidance(ctx); err == nil {
		log.Printf("Coverage Guidance:")
		log.Printf("  Priority Targets: %d", len(guidance.PriorityTargets))
		for _, target := range guidance.PriorityTargets {
			log.Printf("    - %s.%s (priority: %.2f, reason: %s)", 
				target.Package, target.Function, target.Priority, target.Reason)
		}
		log.Printf("  Strategies: %v", guidance.Strategies)
		log.Printf("  Rationale: %s", guidance.Rationale)
	} else {
		log.Printf("Coverage guidance failed: %v", err)
	}
	
	// Show metrics
	metrics := coordinator.GetMetrics()
	log.Printf("Coordinator Metrics: %+v", metrics)
	
	log.Println("=== Demo Complete ===")
}

// SetupFuzzingIntegration configures the enhanced coverage for fuzzing
func SetupFuzzingIntegration() {
	RegisterFuzzingHook(func(snapshot *CoverageSnapshot, guidance *CoverageGuidance) {
		log.Printf("Fuzzing hook triggered - Coverage: %.2f%%, Targets: %d", 
			snapshot.CoverageRatio*100, len(guidance.PriorityTargets))
		
		// This hook would typically:
		// 1. Analyze coverage improvements
		// 2. Adjust fuzzing parameters based on guidance
		// 3. Focus efforts on priority targets
		// 4. Report progress to fuzzing coordinator
		
		LogOracleActivity("fuzzing_integration", map[string]interface{}{
			"coverage_ratio": snapshot.CoverageRatio,
			"targets_count":  len(guidance.PriorityTargets),
			"strategies":     guidance.Strategies,
		})
	})
}

// CleanupEnhancedCoverage properly shuts down the enhanced coverage system
func CleanupEnhancedCoverage() {
	coordinator := GetCoordinator()
	coordinator.Stop()
	log.Println("Enhanced coverage system shut down")
}