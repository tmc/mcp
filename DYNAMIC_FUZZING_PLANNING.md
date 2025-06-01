# Dynamic Fuzzing Plan for MCP-Replay

This document outlines comprehensive strategies for dynamic feedback-driven fuzzing of the MCP-Replay tool, combining Go's built-in coverage-guided fuzzing with custom extensions for directing attention and evaluation.

## Table of Contents

1. [Current Implementation](#current-implementation)
2. [Enhanced Coverage Visualization](#enhanced-coverage-visualization)
3. [Call Graph Analysis](#call-graph-analysis)
4. [User-Directed Attention](#user-directed-attention)
5. [Fuzzing Architecture](#fuzzing-architecture)
6. [Implementation Plan](#implementation-plan)
7. [Evaluation Framework](#evaluation-framework)

## Current Implementation

Our current fuzzing implementation (`main_test.go`) already incorporates several important features:

- Detection of "interesting" inputs that expand coverage
- Separation of invalid inputs vs. real bugs
- Coverage tracking and visualization
- Progress reporting with `GODEBUG=fuzzdebug=1`
- Corpus management of valuable inputs

This foundation gives us a solid starting point for more advanced techniques.

## Enhanced Coverage Visualization

The existing implementation includes:

```go
// Enhanced debug output
if fuzzdebugEnabled {
    // Print detailed coverage information
    report += fmt.Sprintf("Coverage bits: %d → %d (+%d)\n",
        oldBits, newBits, newBits-oldBits)
    
    // Show uncovered functions
    uncoveredFuncs := getUncoveredFunctions(currentStats)
    // ...
}
```

This can be further enhanced with:

1. **Color-coded coverage maps**: Visual representation of function coverage
2. **Time-series coverage tracking**: Graphs showing coverage growth over time
3. **Path analysis visualization**: Display execution paths through the codebase

## Call Graph Analysis

We will implement static analysis to build and leverage the call graph:

```go
type CallGraph struct {
    // Maps function names to their nodes
    Functions map[string]*FunctionNode
    
    // Entry points in the program (like main())
    EntryPoints []*FunctionNode
    
    // Gateway metrics
    GatewayFunctions []*FunctionNode
}

type FunctionNode struct {
    Name string
    Package string
    
    // Functions this one calls
    OutEdges []*FunctionNode
    
    // Functions that call this one
    InEdges []*FunctionNode
    
    // Coverage metrics
    Coverage float64
    
    // Reachability - how many functions can be reached from here
    ReachabilityScore int
    
    // How valuable this function is as a gateway to unexplored code
    GatewayScore float64
}
```

The call graph will enable several key capabilities:
- Identifying "gateway" functions that can lead to unexplored code
- Finding the shortest paths to uncovered functions
- Creating structured mutations that target specific code paths

## User-Directed Attention

We'll add a mechanism for users to direct the fuzzer's attention to specific parts of the code:

```go
// Focus.go - The user-facing API for directing fuzzer attention

// Target a specific function for increased fuzzing attention
func Target(functionName string, weight float64) {
    director.SetFunctionWeight(functionName, weight)
}

// Prioritize exploration of a specific execution path
func PrioritizePath(functions []string, weight float64) {
    director.AddPriorityPath(functions, weight)
}

// Direct the fuzzer to explore functions near this one
func ExploreNear(functionName string, depth int) {
    callGraph := director.GetCallGraph()
    nearbyFunctions := callGraph.GetFunctionsWithinDistance(functionName, depth)
    for _, f := range nearbyFunctions {
        director.SetFunctionWeight(f, 1.5) // Boost nearby functions
    }
}

// Mark a region of code as critical (high priority)
func MarkCritical(packagePath string, lineStart, lineEnd int) {
    functionsInRegion := director.GetFunctionsInRegion(packagePath, lineStart, lineEnd)
    for _, f := range functionsInRegion {
        director.SetFunctionWeight(f, 2.0) // Double attention to critical regions
    }
}
```

## Fuzzing Architecture

The extended fuzzing architecture will be organized as follows:

```
┌────────────────┐      ┌─────────────────┐       ┌────────────────┐
│ Call Graph     │◄────►│ Coverage Oracle │◄─────►│ Corpus Manager │
│ Analyzer       │      │                 │       │                │
└───────┬────────┘      └────────┬────────┘       └───────┬────────┘
        │                        │                        │
        │                        ▼                        │
        │               ┌─────────────────┐               │
        └──────────────►│ Fuzzing         │◄──────────────┘
                        │ Director        │
                        └────────┬────────┘
                                 │
                                 ▼
                        ┌─────────────────┐
                        │ Script          │
                        │ Generator       │
                        └─────────────────┘
```

### Component Responsibilities

1. **Call Graph Analyzer**:
   - Build complete call graph from source code
   - Calculate gateway scores and path distances
   - Identify strategic points for fuzzing focus

2. **Coverage Oracle**:
   - Track per-test coverage details
   - Monitor progress toward complete coverage
   - Identify uncovered areas and access paths

3. **Corpus Manager**:
   - Maintain and organize valuable test inputs
   - Prioritize inputs based on coverage patterns
   - Implement intelligent deduplication

4. **Fuzzing Director**:
   - Coordinate fuzzing strategies
   - Apply weights from user directives
   - Adapt strategies based on progress

5. **Script Generator**:
   - Create targeted test scripts for specific paths
   - Apply knowledge of MCP-Replay's format
   - Implement constraint-based generation

## Implementation Plan

The implementation will proceed in these phases:

### Phase 1: Call Graph Analysis (2 weeks)

1. Build the call graph analyzer tool
2. Integrate with Go's static analysis tools
3. Create visualization of the call graph
4. Implement gateway function identification

### Phase 2: Coverage Enhancement (2 weeks)

1. Extend coverage tracking for per-testcase data
2. Implement coverage comparison algorithms
3. Build the coverage oracle component
4. Create access path analysis

### Phase 3: Intelligent Corpus Management (2 weeks)

1. Implement the corpus manager with scoring
2. Create template extraction algorithms
3. Build corpus optimization techniques
4. Implement genetic algorithms for corpus evolution

### Phase 4: Targeted Fuzzing Director (3 weeks)

1. Build the fuzzing director
2. Implement multiple fuzzing strategies
3. Create the adaptive strategy switching
4. Integrate all components

### Phase 5: Script Generator (2 weeks)

1. Implement the script generator
2. Create semantic-aware mutators for mcp-replay
3. Build constraint solving for path conditions
4. Integrate with the fuzzing director

### Phase 6: Integration and Evaluation (3 weeks)

1. Integrate all components
2. Create a unified UI/CLI
3. Run comprehensive benchmarks
4. Optimize performance bottlenecks

## Influencing Go's Fuzzer

Key strategies for extending Go's fuzzing without modifying it:

```go
// The core of our coverage influencing mechanism
func (d *FuzzDirector) runWithGuidance(f *testing.F, testFunc func(t *testing.T, data []byte)) {
    // 1. Initialize with user-defined weights and priorities
    d.initializeWeights()
    
    // 2. Prepare coverage analysis hook
    oldCoverage := make(map[string]float64)
    
    // 3. Define a wrapper that evaluates and influences inputs
    fuzzWrapper := func(t *testing.T, data []byte) {
        // Begin tracking coverage for this input
        testID := d.EvaluationRecorder.StartTest(data)
        
        // Parse the input to extract structural information
        structureInfo := d.analyzeInputStructure(data)
        d.EvaluationRecorder.RecordStructure(testID, structureInfo)
        
        // Run the actual test with coverage tracking
        start := time.Now()
        testFunc(t, data)
        duration := time.Since(start)
        
        // Get coverage for this test
        currentCoverage := getCoverageForCurrentTest()
        
        // Check if this input hit any of our targeted areas
        weightedCoverage := d.calculateWeightedCoverage(currentCoverage)
        
        // A key insight: we don't manipulate the fuzzer directly,
        // but if this input hit a weighted function, we "clone" it
        // in our corpus with minor variations to bias future selection
        if weightedCoverage > d.weightThreshold {
            // Signal to the fuzzer this is "interesting" by creating
            // subtle variations that will also be added to the corpus
            for i := 0; i < int(weightedCoverage); i++ {
                subtleVariation := d.createSubtleVariation(data, i)
                // This will be added to the corpus if it maintains coverage
                f.Add(subtleVariation)
            }
        }
    }
    
    // 4. Start the fuzzer with our wrapper
    f.Fuzz(fuzzWrapper)
}
```

## Evaluation Framework

To track and evaluate fuzzing performance:

```go
type EvaluationRecorder struct {
    // Test results organized by test ID
    TestResults map[string]*TestResult
    
    // Time-series coverage data
    CoverageOverTime []CoverageSnapshot
    
    // Path execution frequencies
    PathFrequencies map[string]int
    
    // Integration with external analysis tools
    ExportHandlers map[string]ExportHandler
}

type TestResult struct {
    // Input that was tested
    Input []byte
    
    // Structure analysis of the input
    Structure InputStructure
    
    // Coverage achieved by this test
    Coverage map[string]float64
    
    // How this test affected weighted areas
    WeightedCoverageScore float64
    
    // Performance metrics
    ExecutionTime time.Duration
    MemoryUsage int64
    
    // Whether this test found new coverage
    FoundNewCoverage bool
    
    // Priority paths hit by this test
    HitPriorityPaths []string
}
```

The evaluation framework will provide:

- Detailed per-test metrics
- Time-series analysis of coverage growth
- Identification of effective vs. ineffective strategies
- Exportable reports in multiple formats (HTML, JSON, CSV)

## MCP-Replay Specific Strategies

For MCP-Replay specifically, we'll implement:

1. **Command Structure Analysis**: Extract the supported command structures
2. **JSON-RPC Schema Extraction**: Build models of valid JSON-RPC patterns
3. **Mode Transition Coverage**: Ensure coverage of transitions between modes
4. **Error Path Targeting**: Generate inputs that trigger specific error conditions
5. **Configuration Space Exploration**: Systematically explore valid configuration combinations

## Expected Results

By implementing this comprehensive approach, we expect to achieve:

1. **Increased Coverage**: From ~75% to 90%+ by finding paths to previously unreachable code
2. **Efficiency**: 5-10x faster discovery of new code paths compared to random fuzzing
3. **Higher Quality Tests**: Generated tests that are semantically meaningful and maintainable
4. **Bug Discovery**: Identification of edge cases in error handling paths
5. **Documentation**: Automated generation of path diagrams showing code flow

This plan combines static analysis, dynamic coverage feedback, and intelligent fuzzing to create a highly efficient targeted fuzzing system tailored specifically for MCP-Replay.