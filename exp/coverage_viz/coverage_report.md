# Coverage Visualization Example

## File: example_test.go

### Coverage Summary
- **Total Coverage**: 83.3% (20/24 lines)
- **Function Coverage**: 100% (1/1 functions)
- **Branch Coverage**: 75% (6/8 branches)

### Enhanced Source View

```go
// Line coverage indicators:
// ✅ = Covered
// ❌ = Not covered
// ⚠️ = Partially covered

10  ✅ func ParseMessage(msg string) (string, error) {    // Tests: 5 | Hits: 7
11  ✅     if msg == "" {                                 // Tests: 1 | Hits: 1
12  ✅         return "", fmt.Errorf("empty message")     // Tests: 1 | Hits: 1
13  ✅     }
14  ✅     
15  ✅     parts := strings.Split(msg, ":")               // Tests: 4 | Hits: 6
16  ✅     if len(parts) < 2 {                           // Tests: 1 | Hits: 1
17  ✅         return "", fmt.Errorf("invalid format")    // Tests: 1 | Hits: 1
18  ✅     }
19  ✅     
20  ⚠️     switch parts[0] {                              // Tests: 3 | Hits: 5
21  ✅     case "info":                                   // Tests: 2 | Hits: 3
22  ✅         return fmt.Sprintf("INFO: %s", parts[1]), nil
23  ✅     case "warn":                                   // Tests: 2 | Hits: 2
24  ✅         return fmt.Sprintf("WARNING: %s", parts[1]), nil
25  ❌     case "error":                                  // Tests: 0 | Hits: 0
26  ❌         return fmt.Sprintf("ERROR: %s", parts[1]), nil
27  ⚠️     default:                                       // Tests: 0 | Hits: 0
28  ❌         return fmt.Sprintf("UNKNOWN: %s", parts[1]), nil
29  ✅     }
30  ✅ }
```

### Test Impact Analysis

| Test Name | Lines Covered | Unique Coverage | Execution Time | Impact Score |
|-----------|---------------|-----------------|----------------|--------------|
| TestParseMessage_Info | 9 (37.5%) | 0 (0%) | 1.2ms | 🟡 Medium |
| TestParseMessage_Warn | 9 (37.5%) | 0 (0%) | 0.8ms | 🟡 Medium |
| TestParseMessage_Empty | 3 (12.5%) | 3 (12.5%) | 0.5ms | 🟢 High |
| TestParseMessage_InvalidFormat | 6 (25%) | 3 (12.5%) | 0.7ms | 🟢 High |
| TestParseMessage_PartialCoverage | 11 (45.8%) | 0 (0%) | 1.5ms | 🔴 Low |

### Coverage Gaps

#### 1. Uncovered Error Case (Line 25-26)
**Priority**: High  
**Suggested Test**:
```go
func TestParseMessage_Error(t *testing.T) {
    result, err := ParseMessage("error:critical failure")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result != "ERROR: critical failure" {
        t.Errorf("expected 'ERROR: critical failure', got %s", result)
    }
}
```

#### 2. Uncovered Default Case (Line 27-28)
**Priority**: Medium  
**Suggested Test**:
```go
func TestParseMessage_Unknown(t *testing.T) {
    result, err := ParseMessage("debug:verbose output")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result != "UNKNOWN: verbose output" {
        t.Errorf("expected 'UNKNOWN: verbose output', got %s", result)
    }
}
```

### Test Redundancy Analysis

🔴 **High Redundancy Detected**

Tests `TestParseMessage_Info` and `TestParseMessage_PartialCoverage` have 100% overlap for the "info" case.

**Recommendation**: Consider removing the "info" test case from `TestParseMessage_PartialCoverage` or expanding it to cover the missing cases (error, default).

### Call Graph Visualization

```
TestParseMessage_Info
    └── ParseMessage("info:test message")
        ├── strings.Split("info:test message", ":")
        └── fmt.Sprintf("INFO: %s", "test message")

TestParseMessage_Empty
    └── ParseMessage("")
        └── fmt.Errorf("empty message")

TestParseMessage_InvalidFormat
    └── ParseMessage("no colon here")
        ├── strings.Split("no colon here", ":")
        └── fmt.Errorf("invalid format")
```

### Coverage Trend

```
Coverage History (Last 5 commits):
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

100% ┤                                    
 90% ┤                      ╭─────        
 80% ┤           ╭─────────╯    83.3%     
 70% ┤    ╭─────╯                         
 60% ┤────╯  70.8%                        
 50% ┤62.5%                               
     └────┴────┴────┴────┴────┴────       
     5d   4d   3d   2d   1d   now        
```

### Interactive Features

In the actual web interface, you would have:

1. **Hover over line numbers** to see:
   - Which tests cover this line
   - How many times the line was executed
   - Call stack from each test

2. **Click on test names** to:
   - View the test source code
   - See full execution trace
   - Navigate to test file

3. **Coverage filters**:
   - Show only uncovered lines
   - Filter by test
   - Show branch coverage only

4. **Test suggestions**:
   - Auto-generate test templates
   - Find similar tests as examples
   - Estimate coverage impact

### Command Line Output

```bash
$ mcp-coverage report example_test.go

Coverage Report for example_test.go
==================================

Overall: 83.3% (20/24 lines)
Functions: 100% (1/1)
Branches: 75% (6/8)

Uncovered Code:
  Line 25-26: case "error" - No tests
  Line 27-28: default case - No tests

Test Effectiveness:
  ✅ TestParseMessage_Empty: High impact (unique coverage)
  ✅ TestParseMessage_InvalidFormat: High impact (unique coverage)
  ⚠️ TestParseMessage_Info: Medium impact (no unique coverage)
  ⚠️ TestParseMessage_Warn: Medium impact (no unique coverage)
  ❌ TestParseMessage_PartialCoverage: Low impact (redundant)

Suggestions:
  1. Add test for error case (high priority)
  2. Add test for default case (medium priority)
  3. Consider removing redundant tests

Run 'mcp-coverage suggest example_test.go' for detailed test templates.
```

### IDE Integration Preview

In VS Code, you would see:

```
example_test.go
───────────────────────────────────────────────────
10  func ParseMessage(msg string) (string, error) {  │ 5 tests │ 83.3% │
11      if msg == "" {                              │ 1 test  │ ✅     │
12          return "", fmt.Errorf("empty message")  │ 1 test  │ ✅     │
...
25      case "error":                               │ 0 tests │ ❌     │
26          return fmt.Sprintf("ERROR:...           │ 0 tests │ ❌     │
───────────────────────────────────────────────────

💡 2 uncovered branches detected. Click to add tests.
```