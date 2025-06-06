// Package callgraph provides call graph analysis for coverage-guided test exploration.
// This integrates with Go's toolchain to analyze code structure and guide fuzzing efforts.
package callgraph

import (
	"context"
	"fmt"
	"os"
	"runtime/coverage"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/callgraph/static"
	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/pointer"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// AnalysisMode represents different call graph analysis approaches
type AnalysisMode int

const (
	ModeStatic   AnalysisMode = iota // Static analysis (fast, less precise)
	ModeCHA                          // Class Hierarchy Analysis (medium speed/precision)
	ModeRTA                          // Rapid Type Analysis (slower, more precise)
	ModePointer                      // Pointer analysis (slowest, most precise)
	ModeAdaptive                     // Adaptive selection based on code characteristics
)

// CallGraphAnalyzer provides call graph analysis capabilities
type CallGraphAnalyzer struct {
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc

	// Configuration
	mode         AnalysisMode
	packagePaths []string
	rootDir      string

	// Analysis results
	callGraph  *callgraph.Graph
	ssaProgram *ssa.Program
	typeInfo   *loader.Program

	// Metrics and caching
	lastAnalysis time.Time
	analysisTime time.Duration
	nodeCount    int
	edgeCount    int

	// Event handlers
	onAnalysisComplete   func(*AnalysisResult)
	onHotPathsFound      func([]*HotPath)
	onCriticalPathsFound func([]*CriticalPath)

	// Background processing
	wg            sync.WaitGroup
	refreshTicker *time.Ticker
}

// AnalysisResult contains the results of call graph analysis
type AnalysisResult struct {
	Timestamp     time.Time     `json:"timestamp"`
	Mode          AnalysisMode  `json:"mode"`
	PackageCount  int           `json:"package_count"`
	FunctionCount int           `json:"function_count"`
	CallSiteCount int           `json:"call_site_count"`
	AnalysisTime  time.Duration `json:"analysis_time"`

	// Analysis outputs
	CallGraph       *GraphSummary             `json:"call_graph"`
	HotPaths        []*HotPath                `json:"hot_paths"`
	CriticalPaths   []*CriticalPath           `json:"critical_paths"`
	CoverageGaps    []*CoverageGap            `json:"coverage_gaps"`
	Recommendations []*AnalysisRecommendation `json:"recommendations"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata"`
}

// GraphSummary provides a high-level summary of the call graph
type GraphSummary struct {
	TotalNodes  int                     `json:"total_nodes"`
	TotalEdges  int                     `json:"total_edges"`
	MaxDepth    int                     `json:"max_depth"`
	CyclicNodes int                     `json:"cyclic_nodes"`
	EntryPoints []*FunctionNode         `json:"entry_points"`
	LeafNodes   []*FunctionNode         `json:"leaf_nodes"`
	Packages    map[string]*PackageInfo `json:"packages"`
}

// FunctionNode represents a function in the call graph
type FunctionNode struct {
	ID            string  `json:"id"`
	Package       string  `json:"package"`
	Name          string  `json:"name"`
	File          string  `json:"file"`
	Line          int     `json:"line"`
	Column        int     `json:"column"`
	Signature     string  `json:"signature"`
	IsEntryPoint  bool    `json:"is_entry_point"`
	IsLeaf        bool    `json:"is_leaf"`
	CallCount     int     `json:"call_count"`
	CallerCount   int     `json:"caller_count"`
	Complexity    float64 `json:"complexity"`
	CoverageRatio float64 `json:"coverage_ratio"`
}

// PackageInfo provides information about a package
type PackageInfo struct {
	Name          string  `json:"name"`
	Path          string  `json:"path"`
	FunctionCount int     `json:"function_count"`
	CallSiteCount int     `json:"call_site_count"`
	CoverageRatio float64 `json:"coverage_ratio"`
	Complexity    float64 `json:"complexity"`
}

// HotPath represents a frequently executed call path
type HotPath struct {
	ID            string          `json:"id"`
	Path          []*FunctionNode `json:"path"`
	ExecutionFreq float64         `json:"execution_frequency"`
	CoverageRatio float64         `json:"coverage_ratio"`
	Importance    float64         `json:"importance"`
	Reason        string          `json:"reason"`
}

// CriticalPath represents a path critical for system functionality
type CriticalPath struct {
	ID               string          `json:"id"`
	Path             []*FunctionNode `json:"path"`
	CriticalityScore float64         `json:"criticality_score"`
	RiskLevel        string          `json:"risk_level"`
	CoverageGap      float64         `json:"coverage_gap"`
	Reason           string          `json:"reason"`
}

// CoverageGap represents areas with insufficient test coverage
type CoverageGap struct {
	ID              string        `json:"id"`
	Function        *FunctionNode `json:"function"`
	CurrentCoverage float64       `json:"current_coverage"`
	TargetCoverage  float64       `json:"target_coverage"`
	Priority        float64       `json:"priority"`
	Reason          string        `json:"reason"`
	SuggestedTests  []string      `json:"suggested_tests"`
}

// AnalysisRecommendation provides actionable insights from call graph analysis
type AnalysisRecommendation struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Priority    float64                `json:"priority"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Actions     []string               `json:"actions"`
	TargetFuncs []*FunctionNode        `json:"target_functions"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// NewCallGraphAnalyzer creates a new call graph analyzer
func NewCallGraphAnalyzer(ctx context.Context, rootDir string, packagePaths []string) *CallGraphAnalyzer {
	analyzerCtx, cancel := context.WithCancel(ctx)

	analyzer := &CallGraphAnalyzer{
		ctx:          analyzerCtx,
		cancel:       cancel,
		mode:         ModeAdaptive,
		packagePaths: packagePaths,
		rootDir:      rootDir,
	}

	// Start background refresh process
	analyzer.startBackgroundProcesses()

	return analyzer
}

// SetAnalysisMode configures the analysis mode
func (a *CallGraphAnalyzer) SetAnalysisMode(mode AnalysisMode) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.mode = mode
}

// SetEventHandlers configures event callbacks
func (a *CallGraphAnalyzer) SetEventHandlers(
	onAnalysisComplete func(*AnalysisResult),
	onHotPathsFound func([]*HotPath),
	onCriticalPathsFound func([]*CriticalPath),
) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onAnalysisComplete = onAnalysisComplete
	a.onHotPathsFound = onHotPathsFound
	a.onCriticalPathsFound = onCriticalPathsFound
}

// AnalyzeCallGraph performs call graph analysis
func (a *CallGraphAnalyzer) AnalyzeCallGraph() (*AnalysisResult, error) {
	startTime := time.Now()

	a.mu.Lock()
	defer a.mu.Unlock()

	// Load packages
	program, err := a.loadPackages()
	if err != nil {
		return nil, fmt.Errorf("failed to load packages: %w", err)
	}

	a.typeInfo = program

	// Build SSA representation
	ssaProg, ssaPkgs := ssautil.AllPackages(program.AllPackages, 0)
	ssaProg.Build()
	a.ssaProgram = ssaProg

	// Select analysis mode
	mode := a.selectAnalysisMode(ssaProg)

	// Perform call graph analysis
	cg, err := a.buildCallGraph(ssaProg, ssaPkgs, mode)
	if err != nil {
		return nil, fmt.Errorf("failed to build call graph: %w", err)
	}

	a.callGraph = cg
	a.lastAnalysis = time.Now()
	a.analysisTime = time.Since(startTime)
	a.nodeCount = len(cg.Nodes)

	// Count edges
	edgeCount := 0
	for _, node := range cg.Nodes {
		edgeCount += len(node.Out)
	}
	a.edgeCount = edgeCount

	// Analyze results
	result := a.analyzeResults(cg, mode, startTime)

	// Trigger event handlers
	if a.onAnalysisComplete != nil {
		go a.onAnalysisComplete(result)
	}
	if a.onHotPathsFound != nil && len(result.HotPaths) > 0 {
		go a.onHotPathsFound(result.HotPaths)
	}
	if a.onCriticalPathsFound != nil && len(result.CriticalPaths) > 0 {
		go a.onCriticalPathsFound(result.CriticalPaths)
	}

	return result, nil
}

// GetCoverageGuidance provides coverage guidance based on call graph analysis
func (a *CallGraphAnalyzer) GetCoverageGuidance() ([]*coverage.CoverageTarget, error) {
	a.mu.RLock()
	cg := a.callGraph
	a.mu.RUnlock()

	if cg == nil {
		return nil, fmt.Errorf("no call graph available - run analysis first")
	}

	targets := make([]*coverage.CoverageTarget, 0)

	// Convert call graph nodes to coverage targets
	for _, node := range cg.Nodes {
		if node.Func == nil {
			continue
		}

		fn := node.Func
		pkg := fn.Pkg.Pkg

		// Calculate priority based on call graph metrics
		priority := a.calculateNodePriority(node)
		complexity := a.calculateNodeComplexity(node)

		target := &coverage.CoverageTarget{
			Package:    pkg.Path(),
			Function:   fn.Name(),
			Priority:   priority,
			Complexity: complexity,
			Reason:     fmt.Sprintf("Call graph analysis: %d callers, %d callees", len(node.In), len(node.Out)),
		}

		// Add line information if available
		if fn.Pos().IsValid() {
			fset := a.ssaProgram.Fset
			pos := fset.Position(fn.Pos())
			target.Line = pos.Line
		}

		targets = append(targets, target)
	}

	// Sort by priority
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].Priority > targets[j].Priority
	})

	// Return top targets
	maxTargets := 20
	if len(targets) > maxTargets {
		targets = targets[:maxTargets]
	}

	return targets, nil
}

// GetHotPaths identifies frequently executed paths
func (a *CallGraphAnalyzer) GetHotPaths(maxPaths int) ([]*HotPath, error) {
	a.mu.RLock()
	cg := a.callGraph
	a.mu.RUnlock()

	if cg == nil {
		return nil, fmt.Errorf("no call graph available")
	}

	hotPaths := make([]*HotPath, 0)

	// Find paths with high call frequency or importance
	for _, node := range cg.Nodes {
		if len(node.Out) == 0 { // Leaf nodes
			continue
		}

		// Build paths from this node
		paths := a.buildPathsFromNode(node, 5) // Max depth of 5

		for _, path := range paths {
			if len(path) < 2 {
				continue
			}

			hotPath := &HotPath{
				ID:            a.generatePathID(path),
				Path:          a.convertToFunctionNodes(path),
				ExecutionFreq: a.estimateExecutionFrequency(path),
				CoverageRatio: a.calculatePathCoverage(path),
				Importance:    a.calculatePathImportance(path),
				Reason:        "High execution frequency and importance",
			}

			hotPaths = append(hotPaths, hotPath)
		}
	}

	// Sort by importance
	sort.Slice(hotPaths, func(i, j int) bool {
		return hotPaths[i].Importance > hotPaths[j].Importance
	})

	// Return top paths
	if len(hotPaths) > maxPaths {
		hotPaths = hotPaths[:maxPaths]
	}

	return hotPaths, nil
}

// GetCriticalPaths identifies paths critical for system functionality
func (a *CallGraphAnalyzer) GetCriticalPaths(maxPaths int) ([]*CriticalPath, error) {
	a.mu.RLock()
	cg := a.callGraph
	a.mu.RUnlock()

	if cg == nil {
		return nil, fmt.Errorf("no call graph available")
	}

	criticalPaths := make([]*CriticalPath, 0)

	// Find entry points and critical functions
	entryPoints := a.findEntryPoints(cg)

	for _, entry := range entryPoints {
		// Build critical paths from entry points
		paths := a.buildCriticalPathsFromNode(entry, 4) // Max depth of 4

		for _, path := range paths {
			if len(path) < 2 {
				continue
			}

			criticalPath := &CriticalPath{
				ID:               a.generatePathID(path),
				Path:             a.convertToFunctionNodes(path),
				CriticalityScore: a.calculateCriticalityScore(path),
				RiskLevel:        a.assessRiskLevel(path),
				CoverageGap:      a.calculateCoverageGap(path),
				Reason:           "Critical system path requiring thorough testing",
			}

			criticalPaths = append(criticalPaths, criticalPath)
		}
	}

	// Sort by criticality score
	sort.Slice(criticalPaths, func(i, j int) bool {
		return criticalPaths[i].CriticalityScore > criticalPaths[j].CriticalityScore
	})

	// Return top paths
	if len(criticalPaths) > maxPaths {
		criticalPaths = criticalPaths[:maxPaths]
	}

	return criticalPaths, nil
}

// GetMetrics returns analyzer metrics
func (a *CallGraphAnalyzer) GetMetrics() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return map[string]interface{}{
		"last_analysis": a.lastAnalysis,
		"analysis_time": a.analysisTime,
		"node_count":    a.nodeCount,
		"edge_count":    a.edgeCount,
		"package_count": len(a.packagePaths),
		"analysis_mode": a.mode,
	}
}

// Private methods

func (a *CallGraphAnalyzer) loadPackages() (*loader.Program, error) {
	conf := loader.Config{
		Build: nil, // Use default build context
	}

	// Add packages to load
	for _, pkgPath := range a.packagePaths {
		conf.Import(pkgPath)
	}

	// If no specific packages, try to load from root directory
	if len(a.packagePaths) == 0 && a.rootDir != "" {
		conf.ImportWithTests(".")
	}

	return conf.Load()
}

func (a *CallGraphAnalyzer) selectAnalysisMode(prog *ssa.Program) AnalysisMode {
	if a.mode != ModeAdaptive {
		return a.mode
	}

	// Adaptive mode selection based on program characteristics
	funcCount := 0
	for _, pkg := range prog.AllPackages() {
		funcCount += len(pkg.Members)
	}

	// Select mode based on size and complexity
	if funcCount < 100 {
		return ModePointer // Small programs can afford precise analysis
	} else if funcCount < 500 {
		return ModeRTA // Medium programs use RTA
	} else if funcCount < 2000 {
		return ModeCHA // Large programs use CHA
	} else {
		return ModeStatic // Very large programs use static analysis
	}
}

func (a *CallGraphAnalyzer) buildCallGraph(prog *ssa.Program, pkgs []*ssa.Package, mode AnalysisMode) (*callgraph.Graph, error) {
	switch mode {
	case ModeStatic:
		return static.CallGraph(prog), nil

	case ModeCHA:
		return cha.CallGraph(prog), nil

	case ModeRTA:
		main := ssautil.MainPackages(pkgs)
		if len(main) == 0 {
			// No main package, use all packages
			var mains []*ssa.Function
			for _, pkg := range pkgs {
				if pkg.Func("init") != nil {
					mains = append(mains, pkg.Func("init"))
				}
			}
			if len(mains) > 0 {
				return rta.Analyze(mains, true).CallGraph, nil
			}
			return static.CallGraph(prog), nil
		}
		return rta.Analyze(main, true).CallGraph, nil

	case ModePointer:
		main := ssautil.MainPackages(pkgs)
		if len(main) == 0 {
			return static.CallGraph(prog), nil
		}

		config := &pointer.Config{
			Mains:          main,
			BuildCallGraph: true,
		}
		result, err := pointer.Analyze(config)
		if err != nil {
			return nil, err
		}
		return result.CallGraph, nil

	default:
		return static.CallGraph(prog), nil
	}
}

func (a *CallGraphAnalyzer) analyzeResults(cg *callgraph.Graph, mode AnalysisMode, startTime time.Time) *AnalysisResult {
	result := &AnalysisResult{
		Timestamp:       time.Now(),
		Mode:            mode,
		AnalysisTime:    time.Since(startTime),
		CallGraph:       a.buildGraphSummary(cg),
		HotPaths:        make([]*HotPath, 0),
		CriticalPaths:   make([]*CriticalPath, 0),
		CoverageGaps:    make([]*CoverageGap, 0),
		Recommendations: make([]*AnalysisRecommendation, 0),
		Metadata:        make(map[string]interface{}),
	}

	// Count functions and call sites
	funcCount := 0
	callSiteCount := 0
	pkgSet := make(map[string]bool)

	for _, node := range cg.Nodes {
		if node.Func != nil {
			funcCount++
			pkg := node.Func.Pkg.Pkg
			pkgSet[pkg.Path()] = true
		}
		callSiteCount += len(node.Out)
	}

	result.PackageCount = len(pkgSet)
	result.FunctionCount = funcCount
	result.CallSiteCount = callSiteCount

	// Find hot paths
	if hotPaths, err := a.GetHotPaths(10); err == nil {
		result.HotPaths = hotPaths
	}

	// Find critical paths
	if criticalPaths, err := a.GetCriticalPaths(10); err == nil {
		result.CriticalPaths = criticalPaths
	}

	// Identify coverage gaps
	result.CoverageGaps = a.identifyCoverageGaps(cg)

	// Generate recommendations
	result.Recommendations = a.generateRecommendations(result)

	return result
}

func (a *CallGraphAnalyzer) buildGraphSummary(cg *callgraph.Graph) *GraphSummary {
	summary := &GraphSummary{
		TotalNodes:  len(cg.Nodes),
		TotalEdges:  0,
		EntryPoints: make([]*FunctionNode, 0),
		LeafNodes:   make([]*FunctionNode, 0),
		Packages:    make(map[string]*PackageInfo),
	}

	pkgStats := make(map[string]*PackageInfo)
	maxDepth := 0
	cyclicNodes := 0

	for _, node := range cg.Nodes {
		summary.TotalEdges += len(node.Out)

		if node.Func == nil {
			continue
		}

		fn := node.Func
		pkg := fn.Pkg.Pkg
		pkgPath := pkg.Path()

		// Update package stats
		if _, exists := pkgStats[pkgPath]; !exists {
			pkgStats[pkgPath] = &PackageInfo{
				Name: pkg.Name(),
				Path: pkgPath,
			}
		}
		pkgStats[pkgPath].FunctionCount++
		pkgStats[pkgPath].CallSiteCount += len(node.Out)

		// Create function node
		funcNode := a.createFunctionNode(node)

		// Check if entry point (no callers)
		if len(node.In) == 0 && len(node.Out) > 0 {
			funcNode.IsEntryPoint = true
			summary.EntryPoints = append(summary.EntryPoints, funcNode)
		}

		// Check if leaf node (no callees)
		if len(node.Out) == 0 && len(node.In) > 0 {
			funcNode.IsLeaf = true
			summary.LeafNodes = append(summary.LeafNodes, funcNode)
		}

		// Calculate depth and check for cycles
		depth := a.calculateNodeDepth(node, make(map[*callgraph.Node]bool))
		if depth > maxDepth {
			maxDepth = depth
		}
		if depth == -1 { // Cycle detected
			cyclicNodes++
		}
	}

	summary.MaxDepth = maxDepth
	summary.CyclicNodes = cyclicNodes
	summary.Packages = pkgStats

	return summary
}

func (a *CallGraphAnalyzer) createFunctionNode(node *callgraph.Node) *FunctionNode {
	if node.Func == nil {
		return &FunctionNode{
			ID:   "unknown",
			Name: "unknown",
		}
	}

	fn := node.Func
	pkg := fn.Pkg.Pkg

	funcNode := &FunctionNode{
		ID:          fmt.Sprintf("%s.%s", pkg.Path(), fn.Name()),
		Package:     pkg.Path(),
		Name:        fn.Name(),
		Signature:   fn.Signature.String(),
		CallCount:   len(node.Out),
		CallerCount: len(node.In),
		Complexity:  a.calculateFunctionComplexity(fn),
	}

	// Add position information if available
	if fn.Pos().IsValid() && a.ssaProgram != nil {
		fset := a.ssaProgram.Fset
		pos := fset.Position(fn.Pos())
		funcNode.File = pos.Filename
		funcNode.Line = pos.Line
		funcNode.Column = pos.Column
	}

	return funcNode
}

func (a *CallGraphAnalyzer) calculateNodePriority(node *callgraph.Node) float64 {
	if node.Func == nil {
		return 0.0
	}

	// Base priority from call connectivity
	priority := 0.0

	// Higher priority for functions with many callers (widely used)
	callerCount := float64(len(node.In))
	priority += callerCount * 0.1

	// Higher priority for functions that call many others (complex)
	calleeCount := float64(len(node.Out))
	priority += calleeCount * 0.05

	// Boost priority for entry points
	if len(node.In) == 0 && len(node.Out) > 0 {
		priority += 0.5
	}

	// Boost priority for functions in critical packages
	pkg := node.Func.Pkg.Pkg
	if a.isCriticalPackage(pkg.Path()) {
		priority += 0.3
	}

	return priority
}

func (a *CallGraphAnalyzer) calculateNodeComplexity(node *callgraph.Node) float64 {
	if node.Func == nil {
		return 0.0
	}

	return a.calculateFunctionComplexity(node.Func)
}

func (a *CallGraphAnalyzer) calculateFunctionComplexity(fn *ssa.Function) float64 {
	if fn == nil {
		return 0.0
	}

	complexity := 0.0

	// Count basic blocks (cyclomatic complexity approximation)
	blockCount := float64(len(fn.Blocks))
	complexity += blockCount * 0.1

	// Count instructions
	instrCount := 0
	for _, block := range fn.Blocks {
		instrCount += len(block.Instrs)
	}
	complexity += float64(instrCount) * 0.01

	// Count parameters (interface complexity)
	if fn.Signature != nil {
		paramCount := fn.Signature.Params().Len()
		complexity += float64(paramCount) * 0.05
	}

	return complexity
}

func (a *CallGraphAnalyzer) isCriticalPackage(pkgPath string) bool {
	criticalPkgs := []string{
		"runtime",
		"syscall",
		"os",
		"net",
		"crypto",
		"encoding",
		"database",
	}

	for _, critical := range criticalPkgs {
		if strings.HasPrefix(pkgPath, critical) {
			return true
		}
	}

	return false
}

func (a *CallGraphAnalyzer) findEntryPoints(cg *callgraph.Graph) []*callgraph.Node {
	entryPoints := make([]*callgraph.Node, 0)

	for _, node := range cg.Nodes {
		// Entry points have no callers but have callees
		if len(node.In) == 0 && len(node.Out) > 0 {
			entryPoints = append(entryPoints, node)
		}
	}

	return entryPoints
}

func (a *CallGraphAnalyzer) buildPathsFromNode(node *callgraph.Node, maxDepth int) [][]*callgraph.Node {
	paths := make([][]*callgraph.Node, 0)
	visited := make(map[*callgraph.Node]bool)

	var dfs func(*callgraph.Node, []*callgraph.Node, int)
	dfs = func(current *callgraph.Node, path []*callgraph.Node, depth int) {
		if depth >= maxDepth || visited[current] {
			if len(path) > 1 {
				paths = append(paths, append([]*callgraph.Node(nil), path...))
			}
			return
		}

		visited[current] = true
		newPath := append(path, current)

		for _, edge := range current.Out {
			dfs(edge.Callee, newPath, depth+1)
		}

		visited[current] = false
	}

	dfs(node, nil, 0)
	return paths
}

func (a *CallGraphAnalyzer) buildCriticalPathsFromNode(node *callgraph.Node, maxDepth int) [][]*callgraph.Node {
	// For critical paths, we focus on paths that include error handling or critical functions
	return a.buildPathsFromNode(node, maxDepth)
}

func (a *CallGraphAnalyzer) convertToFunctionNodes(path []*callgraph.Node) []*FunctionNode {
	nodes := make([]*FunctionNode, len(path))
	for i, node := range path {
		nodes[i] = a.createFunctionNode(node)
	}
	return nodes
}

func (a *CallGraphAnalyzer) generatePathID(path []*callgraph.Node) string {
	if len(path) == 0 {
		return "empty_path"
	}

	parts := make([]string, len(path))
	for i, node := range path {
		if node.Func != nil {
			parts[i] = fmt.Sprintf("%s.%s", node.Func.Pkg.Pkg.Path(), node.Func.Name())
		} else {
			parts[i] = "unknown"
		}
	}

	return strings.Join(parts, "->")
}

func (a *CallGraphAnalyzer) estimateExecutionFrequency(path []*callgraph.Node) float64 {
	// Simplified heuristic - in practice, this would use profiling data
	freq := 1.0

	for _, node := range path {
		if node.Func == nil {
			continue
		}

		// Functions with more callers are likely called more frequently
		callerCount := float64(len(node.In))
		freq *= (1.0 + callerCount*0.1)
	}

	return freq
}

func (a *CallGraphAnalyzer) calculatePathCoverage(path []*callgraph.Node) float64 {
	// Simplified coverage calculation - would integrate with actual coverage data
	totalCoverage := 0.0
	validNodes := 0

	for _, node := range path {
		if node.Func != nil {
			// Placeholder - would use real coverage data
			totalCoverage += 0.7 // Assume 70% coverage
			validNodes++
		}
	}

	if validNodes == 0 {
		return 0.0
	}

	return totalCoverage / float64(validNodes)
}

func (a *CallGraphAnalyzer) calculatePathImportance(path []*callgraph.Node) float64 {
	importance := 0.0

	for _, node := range path {
		if node.Func == nil {
			continue
		}

		// Add importance based on function characteristics
		importance += a.calculateNodePriority(node)

		// Boost for critical packages
		pkg := node.Func.Pkg.Pkg
		if a.isCriticalPackage(pkg.Path()) {
			importance += 0.2
		}
	}

	return importance
}

func (a *CallGraphAnalyzer) calculateCriticalityScore(path []*callgraph.Node) float64 {
	score := 0.0

	for _, node := range path {
		if node.Func == nil {
			continue
		}

		// Higher score for functions in critical packages
		pkg := node.Func.Pkg.Pkg
		if a.isCriticalPackage(pkg.Path()) {
			score += 0.5
		}

		// Higher score for functions with many dependencies
		score += float64(len(node.Out)) * 0.1

		// Check for error handling patterns
		if a.hasErrorHandling(node.Func) {
			score += 0.3
		}
	}

	return score
}

func (a *CallGraphAnalyzer) assessRiskLevel(path []*callgraph.Node) string {
	score := a.calculateCriticalityScore(path)

	if score > 2.0 {
		return "HIGH"
	} else if score > 1.0 {
		return "MEDIUM"
	} else {
		return "LOW"
	}
}

func (a *CallGraphAnalyzer) calculateCoverageGap(path []*callgraph.Node) float64 {
	currentCoverage := a.calculatePathCoverage(path)
	targetCoverage := 0.9 // 90% target

	gap := targetCoverage - currentCoverage
	if gap < 0 {
		gap = 0
	}

	return gap
}

func (a *CallGraphAnalyzer) hasErrorHandling(fn *ssa.Function) bool {
	if fn == nil {
		return false
	}

	// Look for error-related patterns in function name or package
	name := strings.ToLower(fn.Name())
	pkg := strings.ToLower(fn.Pkg.Pkg.Path())

	errorKeywords := []string{"error", "err", "panic", "recover", "fail", "exception"}

	for _, keyword := range errorKeywords {
		if strings.Contains(name, keyword) || strings.Contains(pkg, keyword) {
			return true
		}
	}

	return false
}

func (a *CallGraphAnalyzer) calculateNodeDepth(node *callgraph.Node, visited map[*callgraph.Node]bool) int {
	if visited[node] {
		return -1 // Cycle detected
	}

	if len(node.Out) == 0 {
		return 0 // Leaf node
	}

	visited[node] = true
	maxDepth := 0

	for _, edge := range node.Out {
		depth := a.calculateNodeDepth(edge.Callee, visited)
		if depth == -1 {
			visited[node] = false
			return -1 // Cycle propagated
		}
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	visited[node] = false
	return maxDepth + 1
}

func (a *CallGraphAnalyzer) identifyCoverageGaps(cg *callgraph.Graph) []*CoverageGap {
	gaps := make([]*CoverageGap, 0)

	for _, node := range cg.Nodes {
		if node.Func == nil {
			continue
		}

		// Estimate current coverage (placeholder)
		currentCoverage := 0.7 // Would use real coverage data
		targetCoverage := 0.9

		if currentCoverage < targetCoverage {
			gap := &CoverageGap{
				ID:              fmt.Sprintf("gap_%s", node.Func.Name()),
				Function:        a.createFunctionNode(node),
				CurrentCoverage: currentCoverage,
				TargetCoverage:  targetCoverage,
				Priority:        a.calculateNodePriority(node),
				Reason:          fmt.Sprintf("Coverage %.1f%% below target %.1f%%", currentCoverage*100, targetCoverage*100),
				SuggestedTests:  a.generateTestSuggestions(node),
			}
			gaps = append(gaps, gap)
		}
	}

	// Sort by priority
	sort.Slice(gaps, func(i, j int) bool {
		return gaps[i].Priority > gaps[j].Priority
	})

	// Return top gaps
	maxGaps := 15
	if len(gaps) > maxGaps {
		gaps = gaps[:maxGaps]
	}

	return gaps
}

func (a *CallGraphAnalyzer) generateTestSuggestions(node *callgraph.Node) []string {
	if node.Func == nil {
		return []string{}
	}

	suggestions := []string{
		fmt.Sprintf("Add unit test for %s", node.Func.Name()),
		"Test error conditions and edge cases",
		"Add integration tests for call chains",
	}

	// Add specific suggestions based on function characteristics
	if len(node.Out) > 5 {
		suggestions = append(suggestions, "Test complex call patterns")
	}

	if a.hasErrorHandling(node.Func) {
		suggestions = append(suggestions, "Test error handling paths")
	}

	return suggestions
}

func (a *CallGraphAnalyzer) generateRecommendations(result *AnalysisResult) []*AnalysisRecommendation {
	recommendations := make([]*AnalysisRecommendation, 0)

	// Recommendation for coverage gaps
	if len(result.CoverageGaps) > 0 {
		rec := &AnalysisRecommendation{
			ID:          "coverage_gaps",
			Type:        "coverage",
			Priority:    0.8,
			Title:       "Address Coverage Gaps",
			Description: fmt.Sprintf("Found %d functions with insufficient test coverage", len(result.CoverageGaps)),
			Actions: []string{
				"Prioritize testing functions with low coverage",
				"Focus on critical and complex functions first",
				"Implement automated coverage tracking",
			},
			Metadata: map[string]interface{}{
				"gap_count": len(result.CoverageGaps),
			},
		}
		recommendations = append(recommendations, rec)
	}

	// Recommendation for critical paths
	if len(result.CriticalPaths) > 0 {
		rec := &AnalysisRecommendation{
			ID:          "critical_paths",
			Type:        "testing",
			Priority:    0.9,
			Title:       "Test Critical Execution Paths",
			Description: fmt.Sprintf("Identified %d critical paths requiring thorough testing", len(result.CriticalPaths)),
			Actions: []string{
				"Create comprehensive test suites for critical paths",
				"Implement integration tests for end-to-end flows",
				"Add error injection testing for resilience",
			},
			Metadata: map[string]interface{}{
				"path_count":      len(result.CriticalPaths),
				"high_risk_count": a.countHighRiskPaths(result.CriticalPaths),
			},
		}
		recommendations = append(recommendations, rec)
	}

	// Recommendation for hot paths
	if len(result.HotPaths) > 0 {
		rec := &AnalysisRecommendation{
			ID:          "hot_paths",
			Type:        "performance",
			Priority:    0.7,
			Title:       "Optimize Hot Execution Paths",
			Description: fmt.Sprintf("Found %d frequently executed paths for optimization focus", len(result.HotPaths)),
			Actions: []string{
				"Profile hot paths for performance bottlenecks",
				"Add performance regression tests",
				"Consider caching or optimization strategies",
			},
			Metadata: map[string]interface{}{
				"hot_path_count": len(result.HotPaths),
			},
		}
		recommendations = append(recommendations, rec)
	}

	return recommendations
}

func (a *CallGraphAnalyzer) countHighRiskPaths(paths []*CriticalPath) int {
	count := 0
	for _, path := range paths {
		if path.RiskLevel == "HIGH" {
			count++
		}
	}
	return count
}

func (a *CallGraphAnalyzer) startBackgroundProcesses() {
	// Start periodic refresh
	a.refreshTicker = time.NewTicker(5 * time.Minute)
	a.wg.Add(1)
	go a.backgroundRefreshProcess()
}

func (a *CallGraphAnalyzer) backgroundRefreshProcess() {
	defer a.wg.Done()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-a.refreshTicker.C:
			if os.Getenv("CALLGRAPH_AUTO_REFRESH") == "true" {
				a.AnalyzeCallGraph()
			}
		}
	}
}

// Stop shuts down the analyzer
func (a *CallGraphAnalyzer) Stop() {
	a.cancel()

	if a.refreshTicker != nil {
		a.refreshTicker.Stop()
	}

	a.wg.Wait()
}
