# Call Graph Visualization Example

## Interactive Test Explorer

```
mcpscripttest explore coverage_test.txt
```

### Visual Output

```
┌─────────────────────────────────────────────────────────────────────────┐
│ Test File: coverage_test.txt                                            │
├─────────────────────────────────────────────────────────────────────────┤
│ Line 2: exec mcpdiff file1.mcp file2.mcp                                │
│                                                                         │
│ Call Graph:                                                             │
│ ┌─ mcpdiff.main() ──────────────────────────────────────┐              │
│ │   ├─ flags.Parse()                                    │              │
│ │   ├─ loadMCPFile("file1.mcp") ─────┐                 │              │
│ │   │                                 ├─ os.Open()      │              │
│ │   │                                 └─ parseMCP()     │              │
│ │   ├─ loadMCPFile("file2.mcp")                        │              │
│ │   └─ compareMCPFiles() ────────────┐                 │              │
│ │                                     ├─ sortMessages() │              │
│ │                                     └─ diffAlgorithm()│              │
│ └─────────────────────────────────────┴─────────────────┘              │
│                                                                         │
│ Source Impact: 243 lines across 6 files                                 │
│ Coverage: 23.4% of mcpdiff                                             │
│                                                                         │
│ [View Source] [View Gaps] [Suggest Tests] [Next Line]                  │
└─────────────────────────────────────────────────────────────────────────┘
```

## Source Code View with Test Mapping

```go
// cmd/mcpdiff/main.go
15  func main() {                    // ← Tested by: coverage_test.txt:2,5,8
16      var (
17          ignoreTimestamps = flag.Bool("t", true, "ignore timestamps")
18          ignoreIDs       = flag.Bool("i", false, "ignore IDs")    // ← No tests
19          verbose         = flag.Bool("v", false, "verbose")
20      )
21      
22      flag.Parse()                 // ← Tested by: coverage_test.txt:2,5,8
23      
24      if flag.NArg() != 2 {        // ← Tested by: edge_cases.txt:3
25          log.Fatal("Usage: mcpdiff <file1> <file2>")
26      }
27      
28      file1 := loadMCPFile(flag.Arg(0))  // ← Tested by: coverage_test.txt:2
29      file2 := loadMCPFile(flag.Arg(1))  // ← Tested by: coverage_test.txt:2
30      
31      if err := compareMCPFiles(file1, file2, *ignoreTimestamps); err != nil {
32          log.Fatal(err)           // ← Tested by: error_test.txt:5
33      }
34  }
```

## Test Impact Report

```
Test Impact Analysis: coverage_test.txt
======================================

Overview:
- Total lines: 12
- Lines with executable commands: 4
- Total coverage generated: 31.2%
- Unique coverage contribution: 18.7%

Per-Line Analysis:

Line 2: exec mcpdiff file1.mcp file2.mcp
  Functions called:
    - main() → 100%
    - loadMCPFile() → 85%
    - compareMCPFiles() → 72%
    - parseMCPFormat() → 90%
  Unique contribution: 11.2%
  Execution time: 134ms
  
Line 5: exec mcpcat -color=never file1.mcp
  Functions called:
    - main() → 100%
    - formatMCP() → 65%
    - disableColor() → 100%
  Unique contribution: 7.5%
  Execution time: 89ms

Line 8: exec mcpspy -- mcpcat file1.mcp
  Functions called:
    - main() → 100%
    - setupPipe() → 100%
    - monitorProcess() → 45%
  Unique contribution: 12.3%
  Execution time: 156ms

Optimization Opportunities:
1. Line 11 provides minimal additional coverage (+0.3%)
2. Consider adding tests for error paths in parseMCPFormat()
3. monitorProcess() has low coverage (45%) - add edge cases
```

## Coverage Gap Analysis with Suggestions

```
Coverage Gaps Found: 8
=====================

Gap 1: Error handling in loadMCPFile()
  Location: cmd/mcpdiff/parser.go:45-52
  Reason: No test provides non-existent file
  Suggested test:
    exec mcpdiff nonexistent.mcp file2.mcp
    stderr 'no such file'
    ! stdout 'Files match'

Gap 2: Binary file detection
  Location: internal/mcp/parser.go:78-85
  Reason: All tests use text files
  Suggested test:
    >binary.mcp
    \x00\x01\x02\x03
    exec mcpdiff binary.mcp text.mcp
    stderr 'binary content'

Gap 3: Concurrent access handling
  Location: internal/mcp/lock.go:12-34
  Reason: No tests trigger concurrent access
  Suggested test:
    exec mcpspy -- bash -c 'mcpdiff file1.mcp file2.mcp & mcpdiff file2.mcp file3.mcp'
    stdout 'Files match'
```

## IDE Integration Preview

```
┌─ VSCode: cmd/mcpdiff/main.go ───────────────────────────────────────┐
│ 28  │ file1 := loadMCPFile(flag.Arg(0))  ⚡ 3 tests              │
│ 29  │ file2 := loadMCPFile(flag.Arg(1))  ⚡ 3 tests              │
│ 30  │                                                            │
│ 31  │ if err := compareMCPFiles(file1, file2, *ignoreTimestamps);│
│     │                           └─ Click to see calling tests     │
├─────┴────────────────────────────────────────────────────────────────┤
│ Test Coverage:                                                      │
│ • coverage_test.txt:2  - exec mcpdiff file1.mcp file2.mcp          │
│ • coverage_test.txt:8  - exec mcpdiff --no-timestamps f1 f2        │
│ • integration_test.txt:15 - exec mcpdiff large1.mcp large2.mcp     │
│                                                                     │
│ [Run Tests] [Debug Test] [Add Test] [View Call Graph]              │
└─────────────────────────────────────────────────────────────────────┘
```

## Command-Line Tool Examples

```bash
# Find which tests cover a specific function
$ mcpscripttest which-tests --function compareMCPFiles
coverage_test.txt:2
coverage_test.txt:8
integration_test.txt:15
edge_cases_test.txt:3

# Generate test for uncovered code
$ mcpscripttest suggest-test --file parser.go --line 45
Suggested test for error handling at parser.go:45:

exec mcpdiff malformed.json valid.mcp
stderr 'invalid JSON'
! stdout 'Files match'

-- malformed.json --
{ invalid json content

# Analyze test redundancy
$ mcpscripttest analyze redundancy coverage_test.txt
Line 11: exec mcpdiff file1.mcp file3.mcp
  Redundancy: 94% (only +0.6% unique coverage)
  Overlaps with: Line 2 (11.2% coverage)
  Recommendation: Consider removing or modifying for different path
```

This synthetic coverage and call graph integration would transform how developers understand and optimize their tests, providing deep insights into the relationship between test code and source code.