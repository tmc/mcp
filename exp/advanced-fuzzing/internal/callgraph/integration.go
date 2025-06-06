package callgraph

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime/coverage"
	"sort"
	"sync"
	"time"
)

// CoverageCallGraphIntegrator connects call graph analysis with coverage tracking
type CoverageCallGraphIntegrator struct {
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc

	// Core components
	analyzer            *CallGraphAnalyzer
	coverageCoordinator *coverage.EnhancedCoverageCoordinator

	// Integration state
	lastIntegration    time.Time
	integrationResults []*IntegrationResult

	// Configuration
	integrationInterval time.Duration
	maxResults          int

	// Event handlers
	onTargetsGenerated  func([]*coverage.CoverageTarget)
	onInsightsGenerated func([]*CoverageInsight)

	// Background processing
	wg                sync.WaitGroup
	integrationTicker *time.Ticker
}

// IntegrationResult contains the results of call graph + coverage integration
type IntegrationResult struct {
	Timestamp        time.Time                    `json:"timestamp"`
	CoverageSnapshot *coverage.CoverageSnapshot   `json:"coverage_snapshot"`
	CallGraphSummary *GraphSummary                `json:"call_graph_summary"`
	GeneratedTargets []*coverage.CoverageTarget   `json:"generated_targets"`
	CoverageInsights []*CoverageInsight           `json:"coverage_insights"`
	Recommendations  []*IntegrationRecommendation `json:"recommendations"`
	Metrics          map[string]interface{}       `json:"metrics"`
}

// CoverageInsight provides insights from combining coverage and call graph data
type CoverageInsight struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Functions   []*FunctionNode        `json:"functions"`
	Paths       []*HotPath             `json:"paths"`
	Suggestions []string               `json:"suggestions"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// IntegrationRecommendation provides actionable recommendations from integrated analysis
type IntegrationRecommendation struct {
	ID             string                 `json:"id"`
	Priority       float64                `json:"priority"`
	Category       string                 `json:"category"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	ActionPlan     []string               `json:"action_plan"`
	TargetFuncs    []*FunctionNode        `json:"target_functions"`
	ExpectedImpact string                 `json:"expected_impact"`
	Effort         string                 `json:"effort"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// NewCoverageCallGraphIntegrator creates a new integrator
func NewCoverageCallGraphIntegrator(ctx context.Context, rootDir string, packagePaths []string) (*CoverageCallGraphIntegrator, error) {
	integratorCtx, cancel := context.WithCancel(ctx)

	// Create call graph analyzer
	analyzer := NewCallGraphAnalyzer(integratorCtx, rootDir, packagePaths)

	// Get coverage coordinator
	coverageCoordinator := coverage.GetCoordinator()

	integrator := &CoverageCallGraphIntegrator{
		ctx:                 integratorCtx,
		cancel:              cancel,
		analyzer:            analyzer,
		coverageCoordinator: coverageCoordinator,
		integrationResults:  make([]*IntegrationResult, 0),
		integrationInterval: 2 * time.Minute,
		maxResults:          50,
	}

	// Set up analyzer event handlers
	analyzer.SetEventHandlers(
		integrator.onCallGraphAnalysisComplete,
		integrator.onHotPathsDiscovered,
		integrator.onCriticalPathsDiscovered,
	)

	// Start background integration process
	integrator.startBackgroundProcesses()

	return integrator, nil
}

// SetEventHandlers configures event callbacks
func (i *CoverageCallGraphIntegrator) SetEventHandlers(
	onTargetsGenerated func([]*coverage.CoverageTarget),
	onInsightsGenerated func([]*CoverageInsight),
) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.onTargetsGenerated = onTargetsGenerated
	i.onInsightsGenerated = onInsightsGenerated
}

// PerformIntegration runs the complete integration analysis
func (i *CoverageCallGraphIntegrator) PerformIntegration() (*IntegrationResult, error) {
	startTime := time.Now()

	// Run call graph analysis
	analysisResult, err := i.analyzer.AnalyzeCallGraph()
	if err != nil {
		return nil, fmt.Errorf("call graph analysis failed: %w", err)
	}

	// Get current coverage snapshot
	coverageSnapshot := i.coverageCoordinator.TakeSnapshot()

	// Generate enhanced coverage targets using call graph insights
	targets, err := i.generateEnhancedCoverageTargets(analysisResult, coverageSnapshot)
	if err != nil {
		return nil, fmt.Errorf("target generation failed: %w", err)
	}

	// Generate coverage insights
	insights := i.generateCoverageInsights(analysisResult, coverageSnapshot)

	// Generate integration recommendations
	recommendations := i.generateIntegrationRecommendations(analysisResult, coverageSnapshot, insights)

	// Create integration result
	result := &IntegrationResult{
		Timestamp:        time.Now(),
		CoverageSnapshot: coverageSnapshot,
		CallGraphSummary: analysisResult.CallGraph,
		GeneratedTargets: targets,
		CoverageInsights: insights,
		Recommendations:  recommendations,
		Metrics: map[string]interface{}{
			"integration_time":   time.Since(startTime),
			"targets_generated":  len(targets),
			"insights_generated": len(insights),
			"recommendations":    len(recommendations),
			"call_graph_nodes":   analysisResult.CallGraph.TotalNodes,
			"call_graph_edges":   analysisResult.CallGraph.TotalEdges,
			"coverage_ratio":     coverageSnapshot.CoverageRatio,
		},
	}

	// Store result
	i.mu.Lock()
	i.integrationResults = append(i.integrationResults, result)
	if len(i.integrationResults) > i.maxResults {
		i.integrationResults = i.integrationResults[1:]
	}
	i.lastIntegration = time.Now()
	i.mu.Unlock()

	// Trigger event handlers
	if i.onTargetsGenerated != nil {
		go i.onTargetsGenerated(targets)
	}
	if i.onInsightsGenerated != nil {
		go i.onInsightsGenerated(insights)
	}

	return result, nil
}

// GetLatestResult returns the most recent integration result
func (i *CoverageCallGraphIntegrator) GetLatestResult() *IntegrationResult {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if len(i.integrationResults) == 0 {
		return nil
	}

	return i.integrationResults[len(i.integrationResults)-1]
}

// GetResults returns all integration results
func (i *CoverageCallGraphIntegrator) GetResults() []*IntegrationResult {
	i.mu.RLock()
	defer i.mu.RUnlock()

	results := make([]*IntegrationResult, len(i.integrationResults))
	copy(results, i.integrationResults)
	return results
}

// GetMetrics returns integrator metrics
func (i *CoverageCallGraphIntegrator) GetMetrics() map[string]interface{} {
	i.mu.RLock()
	defer i.mu.RUnlock()

	metrics := map[string]interface{}{
		"last_integration":     i.lastIntegration,
		"integration_count":    len(i.integrationResults),
		"integration_interval": i.integrationInterval,
	}

	// Add analyzer metrics
	analyzerMetrics := i.analyzer.GetMetrics()
	for k, v := range analyzerMetrics {
		metrics["analyzer_"+k] = v
	}

	// Add coverage coordinator metrics
	if i.coverageCoordinator != nil {
		coverageMetrics := i.coverageCoordinator.GetMetrics()
		for k, v := range coverageMetrics {
			metrics["coverage_"+k] = v
		}
	}

	return metrics
}

// Private methods

func (i *CoverageCallGraphIntegrator) generateEnhancedCoverageTargets(analysisResult *AnalysisResult, coverageSnapshot *coverage.CoverageSnapshot) ([]*coverage.CoverageTarget, error) {
	targets := make([]*coverage.CoverageTarget, 0)

	// Get basic targets from call graph
	basicTargets, err := i.analyzer.GetCoverageGuidance()
	if err == nil {
		targets = append(targets, basicTargets...)
	}

	// Enhance targets with coverage information
	for _, target := range targets {
		i.enhanceTargetWithCoverageData(target, coverageSnapshot)
	}

	// Add targets from critical paths with low coverage
	criticalTargets := i.generateCriticalPathTargets(analysisResult, coverageSnapshot)
	targets = append(targets, criticalTargets...)

	// Add targets from hot paths with coverage gaps
	hotPathTargets := i.generateHotPathTargets(analysisResult, coverageSnapshot)
	targets = append(targets, hotPathTargets...)

	// Add targets from coverage gaps in important functions
	gapTargets := i.generateCoverageGapTargets(analysisResult, coverageSnapshot)
	targets = append(targets, gapTargets...)

	// Sort by priority and remove duplicates
	targets = i.deduplicateAndSortTargets(targets)

	// Limit to reasonable number
	maxTargets := 25
	if len(targets) > maxTargets {
		targets = targets[:maxTargets]
	}

	return targets, nil
}

func (i *CoverageCallGraphIntegrator) enhanceTargetWithCoverageData(target *coverage.CoverageTarget, snapshot *coverage.CoverageSnapshot) {
	// Look up coverage information for this target
	if funcStats, exists := snapshot.FunctionStats[target.Function]; exists {
		target.Priority *= (1.0 + (1.0 - funcStats.CoverageRatio)) // Boost priority for low coverage

		// Add coverage metadata
		if target.Metadata == nil {
			target.Metadata = make(map[string]interface{})
		}
		target.Metadata["current_coverage"] = funcStats.CoverageRatio
		target.Metadata["call_count"] = funcStats.CallCount
		target.Metadata["last_hit"] = funcStats.LastHit
	}
}

func (i *CoverageCallGraphIntegrator) generateCriticalPathTargets(analysisResult *AnalysisResult, coverageSnapshot *coverage.CoverageSnapshot) []*coverage.CoverageTarget {
	targets := make([]*coverage.CoverageTarget, 0)

	for _, criticalPath := range analysisResult.CriticalPaths {
		if criticalPath.CoverageGap > 0.2 { // Significant coverage gap
			for _, funcNode := range criticalPath.Path {
				target := &coverage.CoverageTarget{
					Package:    funcNode.Package,
					Function:   funcNode.Name,
					Line:       funcNode.Line,
					Priority:   criticalPath.CriticalityScore * 0.8,
					Complexity: funcNode.Complexity,
					Reason:     fmt.Sprintf("Critical path with %.1f%% coverage gap", criticalPath.CoverageGap*100),
					Metadata: map[string]interface{}{
						"source":            "critical_path",
						"path_id":           criticalPath.ID,
						"risk_level":        criticalPath.RiskLevel,
						"criticality_score": criticalPath.CriticalityScore,
					},
				}
				targets = append(targets, target)
			}
		}
	}

	return targets
}

func (i *CoverageCallGraphIntegrator) generateHotPathTargets(analysisResult *AnalysisResult, coverageSnapshot *coverage.CoverageSnapshot) []*coverage.CoverageTarget {
	targets := make([]*coverage.CoverageTarget, 0)

	for _, hotPath := range analysisResult.HotPaths {
		if hotPath.CoverageRatio < 0.8 { // Below target coverage
			for _, funcNode := range hotPath.Path {
				target := &coverage.CoverageTarget{
					Package:    funcNode.Package,
					Function:   funcNode.Name,
					Line:       funcNode.Line,
					Priority:   hotPath.Importance * 0.6,
					Complexity: funcNode.Complexity,
					Reason:     fmt.Sprintf("Hot path with %.1f%% coverage (execution freq: %.1f)", hotPath.CoverageRatio*100, hotPath.ExecutionFreq),
					Metadata: map[string]interface{}{
						"source":         "hot_path",
						"path_id":        hotPath.ID,
						"execution_freq": hotPath.ExecutionFreq,
						"importance":     hotPath.Importance,
					},
				}
				targets = append(targets, target)
			}
		}
	}

	return targets
}

func (i *CoverageCallGraphIntegrator) generateCoverageGapTargets(analysisResult *AnalysisResult, coverageSnapshot *coverage.CoverageSnapshot) []*coverage.CoverageTarget {
	targets := make([]*coverage.CoverageTarget, 0)

	for _, gap := range analysisResult.CoverageGaps {
		target := &coverage.CoverageTarget{
			Package:    gap.Function.Package,
			Function:   gap.Function.Name,
			Line:       gap.Function.Line,
			Priority:   gap.Priority,
			Complexity: gap.Function.Complexity,
			Reason:     gap.Reason,
			Metadata: map[string]interface{}{
				"source":           "coverage_gap",
				"gap_id":           gap.ID,
				"current_coverage": gap.CurrentCoverage,
				"target_coverage":  gap.TargetCoverage,
				"suggested_tests":  gap.SuggestedTests,
			},
		}
		targets = append(targets, target)
	}

	return targets
}

func (i *CoverageCallGraphIntegrator) deduplicateAndSortTargets(targets []*coverage.CoverageTarget) []*coverage.CoverageTarget {
	// Create map to deduplicate by function
	targetMap := make(map[string]*coverage.CoverageTarget)

	for _, target := range targets {
		key := fmt.Sprintf("%s.%s", target.Package, target.Function)
		if existing, exists := targetMap[key]; exists {
			// Keep target with higher priority
			if target.Priority > existing.Priority {
				targetMap[key] = target
			}
		} else {
			targetMap[key] = target
		}
	}

	// Convert back to slice
	uniqueTargets := make([]*coverage.CoverageTarget, 0, len(targetMap))
	for _, target := range targetMap {
		uniqueTargets = append(uniqueTargets, target)
	}

	// Sort by priority
	sort.Slice(uniqueTargets, func(i, j int) bool {
		return uniqueTargets[i].Priority > uniqueTargets[j].Priority
	})

	return uniqueTargets
}

func (i *CoverageCallGraphIntegrator) generateCoverageInsights(analysisResult *AnalysisResult, coverageSnapshot *coverage.CoverageSnapshot) []*CoverageInsight {
	insights := make([]*CoverageInsight, 0)

	// Insight 1: Critical functions with low coverage
	criticalLowCov := i.findCriticalFunctionsWithLowCoverage(analysisResult, coverageSnapshot)
	if len(criticalLowCov) > 0 {
		insight := &CoverageInsight{
			ID:          "critical_low_coverage",
			Type:        "risk",
			Severity:    "high",
			Title:       "Critical Functions with Low Coverage",
			Description: fmt.Sprintf("Found %d critical functions with coverage below 70%%", len(criticalLowCov)),
			Functions:   criticalLowCov,
			Suggestions: []string{
				"Prioritize testing these critical functions",
				"Add integration tests for critical paths",
				"Consider property-based testing for complex functions",
			},
			Metadata: map[string]interface{}{
				"function_count": len(criticalLowCov),
				"avg_coverage":   i.calculateAverageCoverage(criticalLowCov, coverageSnapshot),
			},
		}
		insights = append(insights, insight)
	}

	// Insight 2: Hot paths with coverage gaps
	hotPathGaps := i.findHotPathsWithCoverageGaps(analysisResult)
	if len(hotPathGaps) > 0 {
		insight := &CoverageInsight{
			ID:          "hot_path_gaps",
			Type:        "performance",
			Severity:    "medium",
			Title:       "Hot Paths with Coverage Gaps",
			Description: fmt.Sprintf("Identified %d frequently executed paths with insufficient coverage", len(hotPathGaps)),
			Paths:       hotPathGaps,
			Suggestions: []string{
				"Add performance tests for hot paths",
				"Focus on edge cases in frequently used code",
				"Consider benchmark tests with coverage tracking",
			},
			Metadata: map[string]interface{}{
				"path_count":    len(hotPathGaps),
				"avg_exec_freq": i.calculateAverageExecutionFreq(hotPathGaps),
			},
		}
		insights = append(insights, insight)
	}

	// Insight 3: Orphaned functions (high coverage but no callers in analyzed paths)
	orphanedFuncs := i.findOrphanedFunctions(analysisResult, coverageSnapshot)
	if len(orphanedFuncs) > 0 {
		insight := &CoverageInsight{
			ID:          "orphaned_functions",
			Type:        "maintenance",
			Severity:    "low",
			Title:       "Potentially Orphaned Functions",
			Description: fmt.Sprintf("Found %d functions with coverage but no apparent callers", len(orphanedFuncs)),
			Functions:   orphanedFuncs,
			Suggestions: []string{
				"Review if these functions are still needed",
				"Check for dynamic or reflection-based calls",
				"Consider refactoring or removal if truly unused",
			},
			Metadata: map[string]interface{}{
				"function_count": len(orphanedFuncs),
			},
		}
		insights = append(insights, insight)
	}

	// Insight 4: Coverage vs. complexity mismatch
	complexityMismatch := i.findCoverageComplexityMismatches(analysisResult, coverageSnapshot)
	if len(complexityMismatch) > 0 {
		insight := &CoverageInsight{
			ID:          "complexity_mismatch",
			Type:        "quality",
			Severity:    "medium",
			Title:       "Coverage-Complexity Mismatches",
			Description: fmt.Sprintf("Found %d functions where coverage doesn't match complexity", len(complexityMismatch)),
			Functions:   complexityMismatch,
			Suggestions: []string{
				"High complexity functions need more thorough testing",
				"Simple functions may be over-tested",
				"Balance testing effort with function complexity",
			},
			Metadata: map[string]interface{}{
				"mismatch_count": len(complexityMismatch),
			},
		}
		insights = append(insights, insight)
	}

	return insights
}

func (i *CoverageCallGraphIntegrator) findCriticalFunctionsWithLowCoverage(analysisResult *AnalysisResult, coverageSnapshot *coverage.CoverageSnapshot) []*FunctionNode {
	criticalFuncs := make([]*FunctionNode, 0)

	// Check entry points
	for _, entryPoint := range analysisResult.CallGraph.EntryPoints {
		if funcStats, exists := coverageSnapshot.FunctionStats[entryPoint.Name]; exists {
			if funcStats.CoverageRatio < 0.7 { // Below 70% coverage
				entryPoint.CoverageRatio = funcStats.CoverageRatio
				criticalFuncs = append(criticalFuncs, entryPoint)
			}
		}
	}

	// Check functions in critical paths
	for _, criticalPath := range analysisResult.CriticalPaths {
		for _, funcNode := range criticalPath.Path {
			if funcStats, exists := coverageSnapshot.FunctionStats[funcNode.Name]; exists {
				if funcStats.CoverageRatio < 0.7 {
					funcNode.CoverageRatio = funcStats.CoverageRatio
					criticalFuncs = append(criticalFuncs, funcNode)
				}
			}
		}
	}

	return i.deduplicateFunctionNodes(criticalFuncs)
}

func (i *CoverageCallGraphIntegrator) findHotPathsWithCoverageGaps(analysisResult *AnalysisResult) []*HotPath {
	gapPaths := make([]*HotPath, 0)

	for _, hotPath := range analysisResult.HotPaths {
		if hotPath.CoverageRatio < 0.8 { // Below 80% coverage
			gapPaths = append(gapPaths, hotPath)
		}
	}

	return gapPaths
}

func (i *CoverageCallGraphIntegrator) findOrphanedFunctions(analysisResult *AnalysisResult, coverageSnapshot *coverage.CoverageSnapshot) []*FunctionNode {
	orphaned := make([]*FunctionNode, 0)

	// Look for functions with coverage but no callers in call graph
	for funcName, funcStats := range coverageSnapshot.FunctionStats {
		if funcStats.CoverageRatio > 0.1 { // Has some coverage
			// Check if function appears in call graph
			found := false
			for _, node := range analysisResult.CallGraph.EntryPoints {
				if node.Name == funcName {
					found = true
					break
				}
			}
			for _, node := range analysisResult.CallGraph.LeafNodes {
				if node.Name == funcName {
					found = true
					break
				}
			}

			if !found {
				funcNode := &FunctionNode{
					Name:          funcName,
					Package:       funcStats.Package,
					CoverageRatio: funcStats.CoverageRatio,
					CallCount:     int(funcStats.CallCount),
				}
				orphaned = append(orphaned, funcNode)
			}
		}
	}

	return orphaned
}

func (i *CoverageCallGraphIntegrator) findCoverageComplexityMismatches(analysisResult *AnalysisResult, coverageSnapshot *coverage.CoverageSnapshot) []*FunctionNode {
	mismatches := make([]*FunctionNode, 0)

	// Check entry points and leaf nodes
	allNodes := append(analysisResult.CallGraph.EntryPoints, analysisResult.CallGraph.LeafNodes...)

	for _, node := range allNodes {
		if funcStats, exists := coverageSnapshot.FunctionStats[node.Name]; exists {
			// High complexity, low coverage
			if node.Complexity > 0.7 && funcStats.CoverageRatio < 0.5 {
				node.CoverageRatio = funcStats.CoverageRatio
				mismatches = append(mismatches, node)
			}
			// Low complexity, very high coverage (potential over-testing)
			if node.Complexity < 0.2 && funcStats.CoverageRatio > 0.95 {
				node.CoverageRatio = funcStats.CoverageRatio
				mismatches = append(mismatches, node)
			}
		}
	}

	return mismatches
}

func (i *CoverageCallGraphIntegrator) deduplicateFunctionNodes(nodes []*FunctionNode) []*FunctionNode {
	seen := make(map[string]bool)
	unique := make([]*FunctionNode, 0)

	for _, node := range nodes {
		key := fmt.Sprintf("%s.%s", node.Package, node.Name)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, node)
		}
	}

	return unique
}

func (i *CoverageCallGraphIntegrator) calculateAverageCoverage(nodes []*FunctionNode, snapshot *coverage.CoverageSnapshot) float64 {
	if len(nodes) == 0 {
		return 0.0
	}

	total := 0.0
	count := 0

	for _, node := range nodes {
		if funcStats, exists := snapshot.FunctionStats[node.Name]; exists {
			total += funcStats.CoverageRatio
			count++
		}
	}

	if count == 0 {
		return 0.0
	}

	return total / float64(count)
}

func (i *CoverageCallGraphIntegrator) calculateAverageExecutionFreq(paths []*HotPath) float64 {
	if len(paths) == 0 {
		return 0.0
	}

	total := 0.0
	for _, path := range paths {
		total += path.ExecutionFreq
	}

	return total / float64(len(paths))
}

func (i *CoverageCallGraphIntegrator) generateIntegrationRecommendations(analysisResult *AnalysisResult, coverageSnapshot *coverage.CoverageSnapshot, insights []*CoverageInsight) []*IntegrationRecommendation {
	recommendations := make([]*IntegrationRecommendation, 0)

	// Recommendation 1: Focus on critical path coverage
	if len(analysisResult.CriticalPaths) > 0 {
		highRiskPaths := 0
		for _, path := range analysisResult.CriticalPaths {
			if path.RiskLevel == "HIGH" && path.CoverageGap > 0.2 {
				highRiskPaths++
			}
		}

		if highRiskPaths > 0 {
			rec := &IntegrationRecommendation{
				ID:          "critical_path_focus",
				Priority:    0.95,
				Category:    "testing",
				Title:       "Prioritize Critical Path Testing",
				Description: fmt.Sprintf("Focus testing efforts on %d high-risk critical paths with significant coverage gaps", highRiskPaths),
				ActionPlan: []string{
					"Identify test scenarios that exercise critical paths end-to-end",
					"Add integration tests for high-risk paths",
					"Implement error injection testing for resilience validation",
					"Set up continuous monitoring for critical path coverage",
				},
				ExpectedImpact: "High - reduces risk of failures in critical system flows",
				Effort:         "Medium - requires coordination across teams",
				Metadata: map[string]interface{}{
					"high_risk_paths": highRiskPaths,
					"total_critical":  len(analysisResult.CriticalPaths),
				},
			}
			recommendations = append(recommendations, rec)
		}
	}

	// Recommendation 2: Optimize test suite based on call graph insights
	rec := &IntegrationRecommendation{
		ID:          "test_suite_optimization",
		Priority:    0.7,
		Category:    "optimization",
		Title:       "Optimize Test Suite Using Call Graph Analysis",
		Description: "Restructure test suite to better align with actual code usage patterns",
		ActionPlan: []string{
			"Reduce over-testing of simple, low-risk functions",
			"Increase testing of complex functions with low coverage",
			"Add tests for frequently executed but under-tested paths",
			"Remove or consolidate redundant tests",
		},
		ExpectedImpact: "Medium - improves test efficiency and coverage quality",
		Effort:         "Medium - requires test suite refactoring",
		Metadata: map[string]interface{}{
			"total_functions": analysisResult.FunctionCount,
			"coverage_ratio":  coverageSnapshot.CoverageRatio,
			"hot_paths":       len(analysisResult.HotPaths),
		},
	}
	recommendations = append(recommendations, rec)

	// Recommendation 3: Address coverage-complexity mismatches
	for _, insight := range insights {
		if insight.ID == "complexity_mismatch" && len(insight.Functions) > 0 {
			rec := &IntegrationRecommendation{
				ID:          "complexity_coverage_alignment",
				Priority:    0.6,
				Category:    "quality",
				Title:       "Align Test Coverage with Function Complexity",
				Description: fmt.Sprintf("Rebalance testing effort for %d functions with coverage-complexity mismatches", len(insight.Functions)),
				ActionPlan: []string{
					"Increase test coverage for high-complexity, low-coverage functions",
					"Review over-tested simple functions for test reduction opportunities",
					"Implement complexity-based coverage targets",
					"Add automated checks for coverage-complexity alignment",
				},
				TargetFuncs:    insight.Functions,
				ExpectedImpact: "Medium - improves test suite efficiency and quality",
				Effort:         "Low - focused changes to existing tests",
				Metadata: map[string]interface{}{
					"mismatch_count": len(insight.Functions),
				},
			}
			recommendations = append(recommendations, rec)
		}
	}

	// Sort recommendations by priority
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Priority > recommendations[j].Priority
	})

	return recommendations
}

// Event handlers

func (i *CoverageCallGraphIntegrator) onCallGraphAnalysisComplete(result *AnalysisResult) {
	// Trigger integration when call graph analysis completes
	go func() {
		if _, err := i.PerformIntegration(); err != nil {
			log.Printf("Integration after call graph analysis failed: %v", err)
		}
	}()
}

func (i *CoverageCallGraphIntegrator) onHotPathsDiscovered(hotPaths []*HotPath) {
	if os.Getenv("CALLGRAPH_DEBUG") != "" {
		log.Printf("Discovered %d hot paths for coverage optimization", len(hotPaths))
	}
}

func (i *CoverageCallGraphIntegrator) onCriticalPathsDiscovered(criticalPaths []*CriticalPath) {
	if os.Getenv("CALLGRAPH_DEBUG") != "" {
		highRisk := 0
		for _, path := range criticalPaths {
			if path.RiskLevel == "HIGH" {
				highRisk++
			}
		}
		log.Printf("Discovered %d critical paths (%d high-risk) requiring attention", len(criticalPaths), highRisk)
	}
}

// Background processing

func (i *CoverageCallGraphIntegrator) startBackgroundProcesses() {
	i.integrationTicker = time.NewTicker(i.integrationInterval)
	i.wg.Add(1)
	go i.backgroundIntegrationProcess()
}

func (i *CoverageCallGraphIntegrator) backgroundIntegrationProcess() {
	defer i.wg.Done()

	for {
		select {
		case <-i.ctx.Done():
			return
		case <-i.integrationTicker.C:
			if os.Getenv("CALLGRAPH_AUTO_INTEGRATE") == "true" {
				i.PerformIntegration()
			}
		}
	}
}

// Stop shuts down the integrator
func (i *CoverageCallGraphIntegrator) Stop() {
	i.cancel()

	if i.integrationTicker != nil {
		i.integrationTicker.Stop()
	}

	i.analyzer.Stop()
	i.wg.Wait()
}
