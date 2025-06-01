# Proximity Analysis Example

## Real-World Scenario: Reaching Uncovered Error Handler

### Initial Situation

```go
// parser.go - Current Coverage: 72%
func parseMCPFile(filename string) (*MCPData, error) {
    data, err := os.ReadFile(filename)
    if err != nil {
        return nil, err  // ✓ Covered by file_not_found_test.txt
    }
    
    var mcp MCPData
    err = json.Unmarshal(data, &mcp)
    if err != nil {
        return nil, handleParseError(err)  // ✗ Line 89: UNCOVERED
    }
    
    if err := validateMCPData(&mcp); err != nil {
        return nil, err  // ✓ Covered by validation_test.txt
    }
    
    return &mcp, nil
}

func handleParseError(err error) error {  // ✗ UNCOVERED FUNCTION
    log.Printf("Parse error: %v", err)
    
    if syntaxErr, ok := err.(*json.SyntaxError); ok {
        return fmt.Errorf("JSON syntax error at offset %d: %v", 
            syntaxErr.Offset, err)
    }
    
    return fmt.Errorf("invalid JSON format: %v", err)
}
```

### Proximity Analysis

```
$ mcpscripttest proximity parser.go:89

Analyzing proximity to: parser.go:89 (handleParseError call)
==========================================================

Found 3 tests within reach:

1. basic_test.txt:5 - Distance: 2 calls ⭐ CLOSEST
   Current test:
     exec mcpcat valid.mcp
     stdout 'timestamp'
   
   Execution path:
     mcpcat.main()
     └─ parseMCPFile("valid.mcp")
         ├─ os.ReadFile() ✓
         ├─ json.Unmarshal() ✓ (succeeds)
         └─ [BRANCHES HERE]
             ├─ Success path ✓ (currently taken)
             └─ Error path ✗ (2 calls to target)
                 └─ handleParseError()
   
   Why it doesn't reach target:
     - Input file has valid JSON
     - json.Unmarshal() succeeds
     - Never enters error condition

2. validation_test.txt:8 - Distance: 3 calls
   Current test:
     exec mcpdiff schema.mcp data.mcp
     stdout 'schema validation'
   
   Path to target:
     mcpdiff.main()
     └─ loadFiles()
         └─ parseMCPFile()
             └─ [3 calls to handleParseError]

3. integration_test.txt:15 - Distance: 4 calls
   Current test:
     exec mcpspy -- mcpcat test.mcp
     stdout 'spy output'
   
   Path involves process spawning, further from target
```

### Suggested Modifications

```
Modification Suggestions for basic_test.txt:5
===========================================

Current Test:
-------------
exec mcpcat valid.mcp
stdout 'timestamp'

-- valid.mcp --
{"timestamp": "2024-01-01", "data": "test"}


Suggested Modification 1: Invalid JSON Syntax
--------------------------------------------
exec mcpcat invalid.mcp
stderr 'JSON syntax error at offset 14'
! stdout 'timestamp'

-- invalid.mcp --
{"timestamp": "2024-01-01" "data": "test"}
              Missing comma ↑

Why this works:
- Triggers json.Unmarshal() error
- Calls handleParseError() directly
- Tests syntax error branch
- Distance: 0 (direct hit)


Suggested Modification 2: Malformed JSON
---------------------------------------
exec mcpcat malformed.mcp
stderr 'invalid JSON format'
! stdout 'timestamp'

-- malformed.mcp --
{timestamp: 2024-01-01, data: test}

Why this works:
- Unquoted keys trigger parse error
- Tests generic error branch
- Distance: 0 (direct hit)


Suggested Modification 3: Partial JSON
-------------------------------------
exec mcpcat truncated.mcp
stderr 'unexpected end of JSON'
! stdout 'timestamp'

-- truncated.mcp --
{"timestamp": "2024-01-01", "data"

Why this works:
- Incomplete JSON structure
- Different error type testing
- Distance: 0 (direct hit)
```

### Visual Call Graph with Distances

```
                mcpcat.main()
                     |
                     | (1 call)
                     ↓
              parseMCPFile()
                     |
          ┌──────────┴──────────┐
          |                     |
          | (1 call)            | (1 call)
          ↓                     ↓
    os.ReadFile()         json.Unmarshal()
          |                     |
          |              ┌──────┴──────┐
          |              |             |
          |         (success)     (error)
          |              |             |
          |              |      (1 call)
          |              |             ↓
          |              |    handleParseError() ← TARGET
          |              |             |
          ↓              ↓             ↓
       (return)       (return)     (return)

Current test path: ━━━━━━━━━━━━━━
Suggested path: - - - - - - - - -
Distance to target: 2 calls
```

### IDE Integration View

```
┌─ parser.go ─────────────────────────────────────────────────────┐
│ 85:  err = json.Unmarshal(data, &mcp)                          │
│ 86:  if err != nil {                                           │
│ 87:      return nil, handleParseError(err)  ⚠️ Uncovered       │
│      ┌─────────────────────────────────────────────────────┐   │
│      │ Nearest test: basic_test.txt:5 (2 calls away)       │   │
│      │                                                     │   │
│      │ Suggested change:                                   │   │
│      │   Replace: valid.mcp                               │   │
│      │   With: invalid.mcp (malformed JSON)               │   │
│      │                                                     │   │
│      │ [Apply Change] [Show Path] [Find More Tests]        │   │
│      └─────────────────────────────────────────────────────┘   │
│ 88:  }                                                          │
│ 89:                                                             │
│ 90:  if err := validateMCPData(&mcp); err != nil {             │
└─────────────────────────────────────────────────────────────────┘
```

### Command-Line Workflow

```bash
# 1. Find closest test to uncovered line
$ mcpscripttest closest parser.go:89
basic_test.txt:5 (distance: 2 calls)

# 2. Analyze why test doesn't reach target
$ mcpscripttest analyze-path basic_test.txt:5 parser.go:89
Path blocked at: json.Unmarshal() success branch
Reason: Input file contains valid JSON

# 3. Generate modification
$ mcpscripttest suggest-change basic_test.txt:5 --target parser.go:89
Suggestion: Change input to invalid JSON
Example: {"invalid": json content"}

# 4. Apply and test
$ mcpscripttest apply-suggestion basic_test.txt:5
Modified test:
  exec mcpcat invalid.mcp
  stderr 'JSON syntax error'

Coverage increased: 72% → 78% (+6%)
Target reached: ✓ parser.go:89
```

### Machine Learning Enhancement

Over time, the system could learn patterns:

```
Pattern Recognition: JSON Parse Errors
====================================

Based on 1,842 similar cases across 47 projects:

Most effective approaches:
1. Missing quotes (87% success rate)
2. Unclosed brackets (85% success rate)
3. Invalid escape sequences (82% success rate)
4. Trailing commas (79% success rate)

Suggested test for your case:
exec mcpcat invalid_quotes.mcp
stderr 'JSON syntax error'

-- invalid_quotes.mcp --
{"key": value}  // Missing quotes on value

Confidence: 94% (based on similar code patterns)
```

This proximity analysis feature would dramatically reduce the effort required to improve test coverage by guiding developers directly to the most efficient path for reaching uncovered code.