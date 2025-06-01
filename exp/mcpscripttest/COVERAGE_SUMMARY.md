# Coverage Collection Summary

This document shows how to run scripttests with coverage collection that works properly.

## The Problem

When GOCOVERDIR propagates through test hierarchies, coverage data gets scattered across subdirectories, making it hard to analyze.

## The Solution

Control where coverage data goes by setting GOCOVERDIR explicitly in your tests.

## Simple Example

Here's the simplest way to run a scripttest and collect coverage:

```go
func TestWithCoverage(t *testing.T) {
    // 1. Create a controlled coverage directory
    coverDir := t.TempDir()
    os.Setenv("GOCOVERDIR", coverDir)
    
    // 2. Install coverage-enabled tools
    cleanup := InstallMCPTools(t, nil)
    defer cleanup()
    
    // 3. Run scripttest
    opts := DefaultOptions()
    opts.AdditionalEnvVars = []string{"GOCOVERDIR"}
    Test(t, "testdata/mytest.txt", opts)
    
    // 4. Coverage is now in coverDir
    t.Logf("Coverage saved to: %s", coverDir)
}
```

## Command Line Usage

```bash
# Create coverage directory
COVERDIR=$(mktemp -d)

# Run test with coverage
GOCOVERDIR=$COVERDIR go test -v -run TestName

# Analyze coverage
go tool covdata percent -i $COVERDIR
```

## Key Points

1. **Set GOCOVERDIR explicitly** to control where coverage goes
2. **Tools auto-detect coverage** when GOCOVERDIR is set
3. **Pass GOCOVERDIR to scripttest** via AdditionalEnvVars
4. **Coverage files use hashed names** (e.g., covmeta.e05e7149...)
5. **Use go tool covdata** to analyze results

## Working Examples

- `TestExplicitCoverage` - Shows direct control over coverage
- `TestPracticalCoverageExample` - Simple, practical example
- `explicit_coverage_test.go` - Complete working implementation

## Scripts

- `complete_coverage_example.sh` - Full workflow example
- `isolated_coverage_run.sh` - Prevents GOCOVERDIR propagation

## Results

When run correctly, you'll see:
- Coverage files created in your controlled directory
- Coverage percentages for tools (e.g., "mcpdiff coverage: 11.1%")
- Ability to generate detailed reports

This approach ensures all coverage data is collected in one place, making analysis straightforward.

## Future Enhancement: Per-Line Coverage Analysis

An interesting enhancement would be to collect coverage data for each line in a scripttest file separately. This would enable fine-grained coverage analysis showing exactly which test lines contribute to coverage.

### Concept

```
# testdata/detailed_coverage.txt
exec mcpdiff file1.mcp file2.mcp    # Coverage: mcpdiff 12.1%
stdout 'Files match'

exec mcpcat file1.mcp               # Coverage: mcpcat 8.3%
stderr 'mcp-send'

exec mcpspy -- echo test            # Coverage: mcpspy 15.7%
stdout 'test'

-- file1.mcp --
mcp-send {"jsonrpc":"2.0","method":"test","id":1}

-- file2.mcp --
mcp-send {"jsonrpc":"2.0","method":"test","id":1}
```

### Implementation Approach

1. **Execute each line separately** with its own GOCOVERDIR
2. **Always include txtar content** at the end of each execution
3. **Collect coverage after each line**
4. **Annotate results** with per-line coverage percentages

### Benefits

- **Identify high-value test lines** that provide most coverage
- **Optimize test suites** by removing redundant lines
- **Debug coverage gaps** by seeing exactly which lines contribute
- **Generate coverage heat maps** for test files

### Example Implementation

```go
func TestPerLineCoverage(t *testing.T) {
    // Parse scripttest file
    lines := parseScriptFile("testdata/test.txt")
    txtar := extractTxtar("testdata/test.txt")

    results := []LineCoverage{}

    for i, line := range lines {
        // Create per-line coverage dir
        lineDir := filepath.Join(baseDir, fmt.Sprintf("line_%d", i))
        os.Setenv("GOCOVERDIR", lineDir)

        // Execute line with txtar content
        executeLineWithTxtar(line, txtar)

        // Collect coverage
        coverage := analyzeCoverage(lineDir)
        results = append(results, LineCoverage{
            Line:     line,
            Coverage: coverage,
        })
    }

    // Generate annotated test file
    generateAnnotatedTest(results)
}
```

This would produce detailed reports showing exactly how each test line contributes to overall coverage, enabling much more targeted test optimization.

## Test Isolation and Advanced Execution Strategies

### Complete Test Isolation

For truly accurate per-line coverage analysis, each test line should run in complete isolation to prevent state pollution between lines.

#### Current Challenges

- Shared file system state between test lines
- Environment variable persistence
- Process state carryover
- Coverage data mixing

#### Isolation Approaches

1. **Process-level isolation**: Run each line in a fresh process
2. **Container isolation**: Use lightweight containers per line
3. **VM-style branching**: Similar to Firecracker VM snapshots

### Firecracker-Style VM Branching for Tests

An advanced approach inspired by Firecracker VMs could enable:

1. **Snapshot at test start**: Create a base VM/container state
2. **Branch for each line**: Fork from the snapshot for each test line
3. **Parallel execution**: Run lines concurrently in isolated branches
4. **Fail-fast iteration**: Quickly retry failing lines without full re-execution

#### Example Concept

```go
type TestBrancher struct {
    baseSnapshot Snapshot
}

func (tb *TestBrancher) RunWithBranching(testFile string) {
    // Create base snapshot with tools and txtar files
    tb.baseSnapshot = createSnapshot(testFile)

    lines := parseTestLines(testFile)
    results := make([]LineCoverage, len(lines))

    // Execute each line in parallel branches
    var wg sync.WaitGroup
    for i, line := range lines {
        wg.Add(1)
        go func(idx int, testLine string) {
            defer wg.Done()

            // Branch from snapshot
            branch := tb.baseSnapshot.Fork()
            defer branch.Cleanup()

            // Run line in isolated branch
            coverage := branch.Execute(testLine)
            results[idx] = coverage

            // If failed, could retry with modifications
            if coverage.Failed {
                retryBranch := tb.baseSnapshot.Fork()
                // Modify test conditions
                coverage = retryBranch.ExecuteWithDebug(testLine)
            }
        }(i, line)
    }
    wg.Wait()
}
```

### Benefits of Branching Approach

1. **True isolation**: Each line runs in pristine environment
2. **Parallel execution**: All lines can run simultaneously
3. **Fast iteration**: Failed tests can be retried quickly
4. **State exploration**: Can try different initial states
5. **Debugging support**: Pause and inspect failed branches

### Implementation Options

1. **Docker branching**: Use overlay filesystems
2. **QEMU snapshots**: Full VM isolation
3. **Firecracker microVMs**: Lightweight, fast branching
4. **User-mode Linux**: Process-level VM isolation
5. **gVisor**: Container runtime with strong isolation

### Future Research Areas

- Optimal snapshot granularity
- Branch caching strategies
- Parallel execution scheduling
- Failure analysis automation
- State-space exploration for testing

This approach would transform scripttest from sequential execution to parallel, isolated testing with powerful debugging capabilities.

## Synthetic Coverage and Call Graph Analysis

### Connecting Test Lines to Source Code

An advanced enhancement would be to create synthetic coverage data and call graphs that directly link scripttest lines to the source code they exercise.

### Call Graph Integration

```go
type CallGraphAnalyzer struct {
    coverage  CoverageData
    callGraph *CallGraph
}

// AnalyzeTestLine traces execution from test line to source
func (cga *CallGraphAnalyzer) AnalyzeTestLine(line string) *ExecutionTrace {
    // Execute line and capture call stack
    trace := cga.TraceExecution(line)

    // Build call graph from entry point
    graph := cga.BuildCallGraph(trace)

    // Map to source code locations
    sourceMap := cga.MapToSource(graph)

    return &ExecutionTrace{
        TestLine:   line,
        CallGraph:  graph,
        SourceMap:  sourceMap,
        Coverage:   cga.CalculateSyntheticCoverage(graph),
    }
}
```

### Example Output

```
Test Line: exec mcpdiff file1.mcp file2.mcp

Call Graph:
└── mcpdiff.main()
    ├── flags.Parse()
    ├── loadMCPFile("file1.mcp")
    │   ├── os.Open()
    │   └── parseMCPFormat()
    │       └── json.Unmarshal()
    ├── loadMCPFile("file2.mcp")
    └── compareMCPFiles()
        ├── sortMessages()
        └── diffAlgorithm()
            └── computeLCS()

Source Coverage Impact:
- cmd/mcpdiff/main.go:15-47 (main function)
- internal/mcp/parser.go:23-89 (file parsing)
- internal/diff/compare.go:12-156 (comparison logic)
- internal/diff/lcs.go:8-67 (LCS algorithm)

Synthetic Coverage: 23.4% of mcpdiff codebase
```

### Visual Call Graph

```
# testdata/coverage_test.txt with call graphs

exec mcpdiff file1.mcp file2.mcp
├─→ main.go:15 (main)            # Entry point
├─→ parser.go:23 (loadMCPFile)   # File loading
├─→ parser.go:45 (parseMCPFormat) # Format parsing
└─→ compare.go:12 (compareMCPFiles) # Core logic
    └─→ lcs.go:8 (computeLCS)    # Diff algorithm

stdout 'Files match'

exec mcpcat -color=never file1.mcp
├─→ main.go:12 (main)
├─→ display.go:34 (formatMCP)
└─→ color.go:56 (disableColor)
```

### Implementation Concepts

#### 1. Dynamic Tracing

```go
// TraceExecution captures runtime call information
func TraceExecution(testLine string) *Trace {
    // Use runtime/trace or bpf to capture execution
    trace.Start()
    defer trace.Stop()

    executeTestLine(testLine)

    return parseTraceData()
}
```

#### 2. Static Analysis Integration

```go
// StaticAnalyzer combines runtime data with static analysis
type StaticAnalyzer struct {
    ast     *ast.Package
    ssa     *ssa.Program
    runtime *RuntimeTrace
}

func (sa *StaticAnalyzer) BuildCompleteGraph() *CallGraph {
    // Combine static call graph with runtime execution
    static := sa.BuildStaticCallGraph()
    runtime := sa.runtime.GetExecutedPaths()

    return MergeGraphs(static, runtime)
}
```

#### 3. Source Mapping

```go
// SourceMapper links test lines to source locations
type SourceMapper struct {
    callGraph *CallGraph
    sourceAST map[string]*ast.File
}

func (sm *SourceMapper) GetImpactedCode(testLine string) []SourceRange {
    calls := sm.callGraph.GetCallsFrom(testLine)

    var ranges []SourceRange
    for _, call := range calls {
        ast := sm.sourceAST[call.File]
        ranges = append(ranges, sm.GetFunctionRange(ast, call.Function))
    }

    return ranges
}
```

### Advanced Features

#### 1. Test Impact Analysis

```
$ mcpscripttest impact analysis_test.txt

Test Impact Analysis
===================

Line 2: exec mcpdiff file1.mcp file2.mcp
  Direct impact:
    - mcpdiff/main.go:15-47 (32 lines)
    - internal/parser.go:23-89 (66 lines)
  Transitive impact:
    - 156 total lines across 8 files
    - Key algorithms: LCS diff, JSON parsing
    - Error paths: 12 untested error conditions

Line 5: exec mcpspy -- mcpcat file1.mcp
  Creates process pipeline:
    - mcpspy/main.go:20-180 (spy logic)
    - mcpcat/main.go:15-95 (display logic)
  Unique coverage:
    - Pipe handling code (45 lines)
    - Process monitoring (78 lines)
```

#### 2. Coverage Gaps Visualization

```go
// GapAnalyzer identifies untested code paths
func (ga *GapAnalyzer) FindGaps(testFile string) []CoverageGap {
    executed := ga.GetExecutedPaths(testFile)
    allPaths := ga.GetAllPossiblePaths()

    gaps := []CoverageGap{}
    for _, path := range allPaths {
        if !executed.Contains(path) {
            gaps = append(gaps, CoverageGap{
                Path:        path,
                Reason:      ga.AnalyzeWhyNotCovered(path),
                Suggestion:  ga.SuggestTestLine(path),
            })
        }
    }
    return gaps
}
```

#### 3. Test Generation Suggestions

```
Coverage Gap Analysis
====================

Uncovered Path: mcpdiff error handling for malformed JSON
Reason: No test provides invalid JSON input
Suggested test line:
  exec mcpdiff invalid.mcp valid.mcp
  stderr 'parse error'
  ! stdout 'Files match'

Uncovered Path: mcpcat with binary input files
Reason: All tests use text-based MCP files
Suggested test line:
  exec mcpcat binary.mcp
  stderr 'binary content detected'
```

### Benefits

1. **Precise Impact Analysis**: Know exactly what code each test line exercises
2. **Coverage Optimization**: Identify redundant tests and gaps
3. **Test Generation**: Suggest new tests for uncovered paths
4. **Debugging Aid**: Trace from test failure to source code
5. **Refactoring Safety**: Understand test dependencies before changing code

### Integration with IDE

```go
// IDE plugin shows test coverage inline
type IDEIntegration struct {
    analyzer *CallGraphAnalyzer
}

func (ide *IDEIntegration) GetTestsForLine(file string, line int) []TestReference {
    // Find all test lines that execute this source line
    return ide.analyzer.GetTestsCovering(file, line)
}

// In IDE: Hover over source line shows:
// "Covered by: coverage_test.txt:5, integration_test.txt:12"
```

This synthetic coverage and call graph approach would provide unprecedented visibility into the relationship between tests and source code, enabling data-driven test optimization and maintenance.

### Proximity Analysis: Finding Tests Closest to Uncovered Code

Another powerful enhancement would be analyzing which existing tests get closest to uncovered lines through call graph analysis. This helps identify the best starting points for extending coverage.

#### Proximity Analyzer

```go
type ProximityAnalyzer struct {
    callGraph *CallGraph
    coverage  *CoverageData
}

// FindClosestTests finds tests that get nearest to an uncovered line
func (pa *ProximityAnalyzer) FindClosestTests(file string, line int) []TestProximity {
    target := SourceLocation{File: file, Line: line}

    // Find all call paths that get close to the target
    var proximities []TestProximity

    for _, test := range pa.getAllTests() {
        trace := pa.callGraph.GetExecutionTrace(test)
        distance := pa.calculateDistance(trace, target)

        proximities = append(proximities, TestProximity{
            Test:         test,
            Distance:     distance,
            NearestPoint: pa.findNearestPoint(trace, target),
            PathToTarget: pa.suggestPath(trace, target),
        })
    }

    // Sort by distance (closest first)
    sort.Slice(proximities, func(i, j int) bool {
        return proximities[i].Distance < proximities[j].Distance
    })

    return proximities
}
```

#### Example Analysis

```
$ mcpscripttest proximity --file parser.go --line 89

Proximity Analysis: parser.go:89 (uncovered error handler)
========================================================

Closest Tests by Call Graph Distance:

1. coverage_test.txt:5 (Distance: 2 calls)
   exec mcpcat malformed.mcp

   Call Path:
   └─ mcpcat.main()
       └─ parseMCPFile() [parser.go:45]
           └─ validateJSON() [parser.go:67]
               ↓ (2 calls away)
           └─ handleParseError() [parser.go:89] ← TARGET

   Suggestion: Modify input to trigger validation failure

2. integration_test.txt:12 (Distance: 3 calls)
   exec mcpdiff corrupt.mcp valid.mcp

   Call Path:
   └─ mcpdiff.main()
       └─ loadFile() [loader.go:23]
           └─ parseContent() [parser.go:34]
               └─ validateFormat() [parser.go:56]
                   ↓ (3 calls away)
               └─ handleParseError() [parser.go:89] ← TARGET

3. edge_cases_test.txt:8 (Distance: 4 calls)
   exec mcpspy -- cat invalid.json

   Near Miss: This test executes parser.go:78 (11 lines away)
   Could reach target with different input
```

#### Visual Proximity Map

```
Call Graph Proximity to parser.go:89
====================================

Legend: ● Executed  ○ Not executed  ★ Target

                    main()
                      ●
                      |
                loadFile()
                      ●
                      |
                parseContent()
                      ●
                    / | \
                   /  |  \
     validateJSON() validateSchema() validateFormat()
           ●              ○                ●
           |              |                |
           ▼              ▼                ▼
    handleParseError() errorResponse() formatError()
           ★              ○               ○
        [TARGET]    [3 calls away]  [3 calls away]

Nearest executed: validateJSON() (2 calls from target)
Test: coverage_test.txt:5
```

#### Intelligent Suggestions

```go
// SuggestModification suggests how to modify a test to reach uncovered code
func (pa *ProximityAnalyzer) SuggestModification(test TestLine, target SourceLocation) TestModification {
    trace := pa.callGraph.GetExecutionTrace(test)
    nearestPoint := pa.findNearestPoint(trace, target)

    // Analyze why the path doesn't continue to target
    blocker := pa.findBlocker(nearestPoint, target)

    switch blocker.Type {
    case ConditionalBlock:
        return TestModification{
            Original: test,
            Suggested: pa.modifyToSatisfyCondition(test, blocker.Condition),
            Reason: fmt.Sprintf("Change input to satisfy condition: %s", blocker.Condition),
        }
    case ErrorPath:
        return TestModification{
            Original: test,
            Suggested: pa.modifyToTriggerError(test, blocker.Error),
            Reason: fmt.Sprintf("Modify to trigger error: %s", blocker.Error),
        }
    case MissingInput:
        return TestModification{
            Original: test,
            Suggested: pa.addRequiredInput(test, blocker.Required),
            Reason: fmt.Sprintf("Add input: %s", blocker.Required),
        }
    }
}
```

#### Practical Example Output

```
$ mcpscripttest suggest-path --to parser.go:89

Path Suggestions to Reach parser.go:89
=====================================

Option 1: Modify coverage_test.txt:5 (Closest - 2 calls away)
Current:  exec mcpcat valid.mcp
Suggested: exec mcpcat malformed.mcp
          stderr 'parse error'

Why: Add malformed JSON to trigger error path
Distance: 2 function calls
Confidence: High (85%)

-- malformed.mcp --
{ "invalid": json content"


Option 2: Extend integration_test.txt:12 (3 calls away)
Current:  exec mcpdiff file1.mcp file2.mcp
Suggested: exec mcpdiff --strict corrupt.mcp valid.mcp
          stderr 'validation failed'

Why: Use --strict flag to enable validation
Distance: 3 function calls
Confidence: Medium (67%)


Option 3: Create new test (Direct path)
Suggested: exec mcpcat --validate invalid.json
          stderr 'JSON parse error at line 1'

Why: Direct path to error handler
Distance: Direct
Confidence: High (92%)

-- invalid.json --
{ unclosed": "bracket
```

#### IDE Integration

```
// IDE shows proximity information inline
type ProximityIDE struct {
    analyzer *ProximityAnalyzer
}

func (ide *ProximityIDE) ShowUncoveredLine(file string, line int) {
    proximities := ide.analyzer.FindClosestTests(file, line)

    // Show inline annotation
    ide.ShowAnnotation(line, fmt.Sprintf(
        "Uncovered - Nearest test: %s (distance: %d calls)",
        proximities[0].Test,
        proximities[0].Distance,
    ))

    // Show hover info with suggestions
    ide.ShowHover(line, formatProximitySuggestions(proximities))
}
```

### Benefits of Proximity Analysis

1. **Efficient Coverage Extension**: Start from the nearest existing test
2. **Minimal Test Modifications**: Small changes to reach uncovered code
3. **Path Visualization**: See exactly how to reach target code
4. **Confidence Metrics**: Know likelihood of success for each approach
5. **Learning Tool**: Understand code flow and dependencies

This proximity analysis would make it dramatically easier to improve coverage by showing developers exactly which tests to modify and how to modify them to reach uncovered code.