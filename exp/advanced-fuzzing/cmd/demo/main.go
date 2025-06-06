// Demo showcasing the advanced fuzzing infrastructure with grammar-guided generation
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"internal/callgraph"
	"internal/fuzz"
	"log"
	"os"
	"runtime/coverage"
	"time"

	"../grammar"
)

func main() {
	log.Println("=== Advanced Fuzzing Infrastructure Demo ===")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Initialize the complete fuzzing pipeline
	if err := runCompletePipeline(ctx); err != nil {
		log.Fatalf("Demo failed: %v", err)
	}

	log.Println("=== Demo Complete ===")
}

func runCompletePipeline(ctx context.Context) error {
	log.Println("\n--- Phase 1: Initialize Enhanced Coverage System ---")

	// Initialize enhanced coverage with LLM oracle
	if err := coverage.InitializeEnhancedCoverage(); err != nil {
		return fmt.Errorf("failed to initialize coverage: %w", err)
	}

	// Get coverage coordinator
	coverageCoordinator := coverage.GetCoordinator()
	log.Printf("✓ Enhanced coverage coordinator initialized")

	log.Println("\n--- Phase 2: Initialize Multi-Modal Fuzzing Coordinator ---")

	// Initialize fuzzing coordinator
	fuzzCoordinator, err := fuzz.InitializeEnhancedFuzzing(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize fuzzing: %w", err)
	}
	log.Printf("✓ Multi-modal fuzzing coordinator initialized")

	log.Println("\n--- Phase 3: Initialize Call Graph Analysis ---")

	// Initialize call graph integration
	packagePaths := []string{"runtime", "testing", "os", "net/http"}
	callgraphIntegrator, err := callgraph.NewCoverageCallGraphIntegrator(ctx, ".", packagePaths)
	if err != nil {
		return fmt.Errorf("failed to initialize call graph: %w", err)
	}
	log.Printf("✓ Call graph analyzer initialized for %d packages", len(packagePaths))

	log.Println("\n--- Phase 4: Initialize Grammar Engine ---")

	// Initialize grammar engine
	grammarEngine := grammar.NewGrammarEngine(ctx)
	grammarEngine.SetCoordinator(fuzzCoordinator)
	grammarEngine.SetCoverageOracle(coverageCoordinator.GetOracle())
	log.Printf("✓ Grammar engine initialized")

	log.Println("\n--- Phase 5: Demonstrate Integrated Pipeline ---")

	// Set up event handlers for demonstration
	setupEventHandlers(coverageCoordinator, fuzzCoordinator, callgraphIntegrator, grammarEngine)

	// Perform call graph analysis
	log.Println("\nPerforming call graph analysis...")
	integrationResult, err := callgraphIntegrator.PerformIntegration()
	if err != nil {
		log.Printf("Warning: Call graph integration failed: %v", err)
	} else {
		log.Printf("✓ Call graph analysis complete: %d targets, %d insights",
			len(integrationResult.GeneratedTargets), len(integrationResult.CoverageInsights))
	}

	// Generate test scenarios using grammar engine
	log.Println("\nGenerating test scenarios...")
	scenarios := []string{"basic_build", "test_coverage", "module_ops", "error_handling"}

	for i, scenario := range scenarios {
		log.Printf("Generating scenario %d/%d: %s", i+1, len(scenarios), scenario)

		result, err := grammarEngine.GenerateTestScenario(ctx, scenario)
		if err != nil {
			log.Printf("  Warning: Failed to generate %s: %v", scenario, err)
			continue
		}

		log.Printf("  ✓ Generated (%s, quality: %.2f, coverage: %.2f)",
			scenario, result.Quality, result.Coverage)
		log.Printf("  Content preview: %s", truncateString(result.Content, 100))

		// Start a fuzzing session for this scenario
		sessionID := fmt.Sprintf("demo_%s_%d", scenario, time.Now().Unix())
		session, err := fuzzCoordinator.StartFuzzingSession(sessionID)
		if err != nil {
			log.Printf("  Warning: Failed to start fuzzing session: %v", err)
			continue
		}

		// Simulate some fuzzing progress
		time.Sleep(200 * time.Millisecond)
		fuzzCoordinator.UpdateSessionProgress(sessionID, 50, 10, 0.05, result.Quality)

		// Complete the session
		fuzzCoordinator.CompleteSession(sessionID, fuzz.StatusCompleted)
		log.Printf("  ✓ Completed fuzzing session %s", sessionID)
	}

	log.Println("\n--- Phase 6: Show Comprehensive Results ---")

	// Show coverage metrics
	log.Println("\nCoverage Metrics:")
	coverageMetrics := coverageCoordinator.GetMetrics()
	printMetrics(coverageMetrics, "  ")

	// Show fuzzing metrics
	log.Println("\nFuzzing Metrics:")
	fuzzingMetrics := fuzzCoordinator.GetMetrics()
	printMetrics(fuzzingMetrics, "  ")

	// Show call graph metrics
	log.Println("\nCall Graph Metrics:")
	callgraphMetrics := callgraphIntegrator.GetMetrics()
	printMetrics(callgraphMetrics, "  ")

	// Show grammar engine metrics
	log.Println("\nGrammar Engine Metrics:")
	grammarMetrics := grammarEngine.GetMetrics()
	printMetrics(grammarMetrics, "  ")

	// Get final recommendations
	log.Println("\n--- Phase 7: Generate Final Recommendations ---")

	targets, strategies, err := fuzzCoordinator.GetRecommendations(ctx)
	if err != nil {
		log.Printf("Warning: Failed to get recommendations: %v", err)
	} else {
		log.Printf("\nFinal Recommendations:")
		log.Printf("  Priority Targets: %d", len(targets))
		for i, target := range targets[:min(3, len(targets))] {
			log.Printf("    %d. %s.%s (priority: %.2f, reason: %s)",
				i+1, target.Package, target.Function, target.Priority, target.Reason)
		}

		log.Printf("  Recommended Strategies: %d", len(strategies))
		for i, strategy := range strategies[:min(3, len(strategies))] {
			log.Printf("    %d. %s (priority: %.2f)",
				i+1, strategy.Name, strategy.Priority)
		}
	}

	// Show integration insights
	if integrationResult != nil && len(integrationResult.CoverageInsights) > 0 {
		log.Printf("\nKey Insights:")
		for i, insight := range integrationResult.CoverageInsights[:min(2, len(integrationResult.CoverageInsights))] {
			log.Printf("  %d. %s (%s severity)", i+1, insight.Title, insight.Severity)
			log.Printf("     %s", insight.Description)
		}
	}

	// Save detailed results to file
	if err := saveDetailedResults(coverageMetrics, fuzzingMetrics, callgraphMetrics, grammarMetrics); err != nil {
		log.Printf("Warning: Failed to save detailed results: %v", err)
	} else {
		log.Println("\n✓ Detailed results saved to demo_results.json")
	}

	return nil
}

func setupEventHandlers(
	coverageCoordinator *coverage.EnhancedCoverageCoordinator,
	fuzzCoordinator *fuzz.MultiModalCoordinator,
	callgraphIntegrator *callgraph.CoverageCallGraphIntegrator,
	grammarEngine *grammar.GrammarEngine,
) {
	// Coverage event handlers
	coverageCoordinator.SetEventHandlers(
		func(snapshot *coverage.CoverageSnapshot) {
			if os.Getenv("DEMO_VERBOSE") == "true" {
				log.Printf("[Coverage] Snapshot: %.1f%% coverage (%d/%d lines)",
					snapshot.CoverageRatio*100, snapshot.CoveredLines, snapshot.TotalLines)
			}
		},
		func(assessment *coverage.TestQualityAssessment) {
			if os.Getenv("DEMO_VERBOSE") == "true" {
				log.Printf("[Coverage] Quality assessment: %.2f score (%s)",
					assessment.Score, assessment.Rationale)
			}
		},
		func(guidance *coverage.CoverageGuidance) {
			if os.Getenv("DEMO_VERBOSE") == "true" {
				log.Printf("[Coverage] Guidance: %d targets, strategies: %v",
					len(guidance.PriorityTargets), guidance.Strategies)
			}
		},
	)

	// Fuzzing event handlers
	fuzzCoordinator.SetEventHandlers(
		func(session *fuzz.FuzzingSession) {
			log.Printf("[Fuzzing] Session started: %s (target: %s, strategy: %s)",
				session.ID, session.CurrentTarget.ID, session.ActiveStrategy.Name)
		},
		func(session *fuzz.FuzzingSession) {
			log.Printf("[Fuzzing] Session completed: %s (%d iterations, %.2f quality)",
				session.ID, session.TotalIterations, session.QualityScore)
		},
		nil, // onTargetChange
		nil, // onStrategyChange
	)

	// Call graph event handlers
	callgraphIntegrator.SetEventHandlers(
		func(targets []*coverage.CoverageTarget) {
			log.Printf("[CallGraph] Generated %d coverage targets from analysis", len(targets))
		},
		func(insights []*callgraph.CoverageInsight) {
			log.Printf("[CallGraph] Generated %d coverage insights", len(insights))
		},
	)

	// Grammar engine event handlers
	grammarEngine.SetEventHandlers(
		func(result *grammar.GenerationResult) {
			if os.Getenv("DEMO_VERBOSE") == "true" {
				log.Printf("[Grammar] Generated content: %s (quality: %.2f)",
					result.ID, result.Quality)
			}
		},
		func(pattern *grammar.Pattern) {
			if os.Getenv("DEMO_VERBOSE") == "true" {
				log.Printf("[Grammar] Discovered pattern: %s (score: %.2f)",
					pattern.Name, pattern.QualityScore)
			}
		},
	)
}

func printMetrics(metrics map[string]interface{}, prefix string) {
	for key, value := range metrics {
		switch v := value.(type) {
		case int, int64:
			log.Printf("%s%s: %d", prefix, key, v)
		case float64:
			log.Printf("%s%s: %.3f", prefix, key, v)
		case bool:
			log.Printf("%s%s: %t", prefix, key, v)
		case time.Time:
			if !v.IsZero() {
				log.Printf("%s%s: %s", prefix, key, v.Format("15:04:05"))
			}
		case time.Duration:
			log.Printf("%s%s: %s", prefix, key, v.String())
		default:
			log.Printf("%s%s: %v", prefix, key, v)
		}
	}
}

func saveDetailedResults(coverageMetrics, fuzzingMetrics, callgraphMetrics, grammarMetrics map[string]interface{}) error {
	results := map[string]interface{}{
		"timestamp":         time.Now(),
		"coverage_metrics":  coverageMetrics,
		"fuzzing_metrics":   fuzzingMetrics,
		"callgraph_metrics": callgraphMetrics,
		"grammar_metrics":   grammarMetrics,
	}

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile("demo_results.json", data, 0644)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
