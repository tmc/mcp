# Improved Bash Coverage Strategies

This document outlines improved approaches for safely and properly covering bash code, building on the BASH_XTRACEFD implementation.

## Safe Coverage Collection

### 1. Non-Invasive Tracing with BASH_XTRACEFD

Our current approach using BASH_XTRACEFD is already non-invasive, but we can enhance it:

```bash
#!/bin/bash
# coverage_wrapper.sh - Safe wrapper for bash scripts

# Create trace file with proper permissions
TRACE_FILE=$(mktemp -t bash-trace-XXXXXX)
chmod 600 "$TRACE_FILE"

# Open FD 3 for trace output
exec 3>"$TRACE_FILE"

# Configure enhanced tracing
export BASH_XTRACEFD=3
export PS4='+${BASH_SOURCE[0]##*/}:${LINENO}:${FUNCNAME[0]:+${FUNCNAME[0]}():} '

# Enable tracing
set -x

# Run the actual script
"$@"
EXIT_CODE=$?

# Disable tracing and close FD
set +x
exec 3>&-

# Process trace file safely
if [[ -f "$TRACE_FILE" ]]; then
    # Parse and store coverage data
    process_trace_file "$TRACE_FILE"
    rm -f "$TRACE_FILE"
fi

exit $EXIT_CODE
```

### 2. Atomic Coverage Operations

Ensure coverage data is written atomically to prevent corruption:

```bash
write_coverage_atomically() {
    local coverage_file=$1
    local temp_file=$(mktemp)
    
    # Write to temp file first
    generate_coverage_report > "$temp_file"
    
    # Atomic move
    mv -f "$temp_file" "$coverage_file"
}
```

### 3. Signal-Safe Coverage Dumping

Handle signals properly to ensure coverage is saved:

```bash
#!/bin/bash
# Safe signal handling for coverage

declare -a COVERAGE_CLEANUP_FUNCS=()

register_coverage_cleanup() {
    COVERAGE_CLEANUP_FUNCS+=("$1")
}

coverage_signal_handler() {
    local sig=$1
    echo "Caught signal $sig, saving coverage..." >&2
    
    for cleanup_func in "${COVERAGE_CLEANUP_FUNCS[@]}"; do
        $cleanup_func || true
    done
    
    # Re-raise the signal
    trap - $sig
    kill -$sig $$
}

# Register handlers for common signals
for sig in INT TERM HUP; do
    trap "coverage_signal_handler $sig" $sig
done
```

## Enhanced Coverage Data Collection

### 1. Function Entry/Exit Tracking

Track function calls with timing and return codes:

```bash
declare -A FUNC_CALLS
declare -A FUNC_TIMES
declare -A FUNC_RETURNS

function_wrapper() {
    local func_name=$1
    shift
    
    local start_time=$(date +%s.%N)
    FUNC_CALLS[$func_name]=$((${FUNC_CALLS[$func_name]:-0} + 1))
    
    # Call the actual function
    $func_name "$@"
    local ret=$?
    
    local end_time=$(date +%s.%N)
    local duration=$(echo "$end_time - $start_time" | bc)
    
    FUNC_TIMES[$func_name]=$(echo "${FUNC_TIMES[$func_name]:-0} + $duration" | bc)
    FUNC_RETURNS[$func_name:$ret]=$((${FUNC_RETURNS[$func_name:$ret]:-0} + 1))
    
    return $ret
}

# Usage: Replace function calls
# function_wrapper checkpoint_create "My checkpoint"
```

### 2. Branch Coverage

Track conditional execution paths:

```bash
declare -A BRANCH_COVERAGE

track_branch() {
    local branch_id=$1
    local branch_taken=$2
    
    BRANCH_COVERAGE[$branch_id:$branch_taken]=$((${BRANCH_COVERAGE[$branch_id:$branch_taken]:-0} + 1))
}

# Usage in script
if [[ -f "$file" ]]; then
    track_branch "file_check" "true"
    process_file "$file"
else
    track_branch "file_check" "false"
    echo "File not found"
fi
```

### 3. Loop Coverage

Track loop iterations:

```bash
declare -A LOOP_COVERAGE

track_loop() {
    local loop_id=$1
    local iteration=$2
    
    LOOP_COVERAGE[$loop_id]=$iteration
}

# Usage
local i=0
for file in *.txt; do
    i=$((i + 1))
    track_loop "txt_files" $i
    process_file "$file"
done
```

## Integration with MCPScriptTest

### 1. Enhanced Bash Command

Improve the bash command in mcpscripttest:

```go
// bash_command_enhanced.go
func bashCmdEnhanced(s *script.State, args ...string) (script.WaitFunc, error) {
    // ... existing code ...
    
    if coverageEnabled {
        // Add safety features
        cmdStr = fmt.Sprintf(`
            # Safety wrapper
            set -euo pipefail
            
            # Enhanced tracing
            export PS4='+\${BASH_SOURCE[0]##*/}:\${LINENO}:\${FUNCNAME[0]:+\${FUNCNAME[0]}():} '
            set -x
            export BASH_XTRACEFD=3
            
            # Trap for cleanup
            trap 'echo "Coverage saved to %s" >&2' EXIT
            
            # Run command
            %s
        `, tracePath, bashCommand)
    }
    
    // ... rest of implementation
}
```

### 2. Coverage Aggregation

Aggregate coverage across multiple test runs:

```go
type BashCoverageAggregator struct {
    scriptCoverage map[string]*BashCoverage
    mu             sync.Mutex
}

func (a *BashCoverageAggregator) Merge(coverage *BashCoverage) {
    a.mu.Lock()
    defer a.mu.Unlock()
    
    existing, ok := a.scriptCoverage[coverage.ScriptPath]
    if !ok {
        a.scriptCoverage[coverage.ScriptPath] = coverage
        return
    }
    
    // Merge line coverage
    for line, _ := range coverage.ExecutedLines {
        existing.ExecutedLines[line] = true
    }
    
    // Recalculate percentage
    existing.CoveragePercent = float64(len(existing.ExecutedLines)) / 
                            float64(existing.TotalLines) * 100
}
```

### 3. Enhanced Trace Parser

Improve trace parsing with error recovery:

```go
func ParseBashTraceRobust(traceFile string) (*BashCoverage, error) {
    file, err := os.Open(traceFile)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    
    coverage := &BashCoverage{
        ExecutedLines: make(map[int]bool),
        Errors:        []string{},
    }
    
    scanner := bufio.NewScanner(file)
    scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // Handle large lines
    
    lineRegex := regexp.MustCompile(`^\+([^:]+):(\d+):(?:(\w+)\(\):)?\s*(.*)`)
    
    for scanner.Scan() {
        line := scanner.Text()
        
        if matches := lineRegex.FindStringSubmatch(line); matches != nil {
            script := matches[1]
            lineNum, _ := strconv.Atoi(matches[2])
            funcName := matches[3]
            command := matches[4]
            
            coverage.ExecutedLines[lineNum] = true
            
            // Track function calls
            if funcName != "" {
                coverage.FunctionCalls[funcName]++
            }
            
            // Detect errors
            if strings.Contains(command, "command not found") {
                coverage.Errors = append(coverage.Errors, 
                    fmt.Sprintf("Line %d: %s", lineNum, command))
            }
        }
    }
    
    return coverage, scanner.Err()
}
```

## Best Practices for Safe Bash Coverage

### 1. Isolation

Run coverage in isolated environments:

```bash
# Use namespaces for isolation
unshare --pid --mount --uts --ipc --net bash -c '
    # Run coverage collection in isolated namespace
    ./coverage_wrapper.sh ./script_under_test.sh
'
```

### 2. Resource Limits

Apply resource limits to prevent runaway scripts:

```bash
# coverage_limited.sh
ulimit -t 300  # CPU time limit (5 minutes)
ulimit -m 500000  # Memory limit (500MB)
ulimit -f 100000  # File size limit

# Run with timeout
timeout --kill-after=10s 5m ./script_under_test.sh
```

### 3. Sanitization

Sanitize coverage data before processing:

```go
func SanitizeCoverageData(data []byte) []byte {
    // Remove sensitive information
    sensitive := []string{
        `/home/\w+`,
        `password=\w+`,
        `token=\w+`,
    }
    
    result := data
    for _, pattern := range sensitive {
        re := regexp.MustCompile(pattern)
        result = re.ReplaceAll(result, []byte("[REDACTED]"))
    }
    
    return result
}
```

### 4. Coverage Validation

Validate coverage data integrity:

```go
func ValidateCoverage(coverage *BashCoverage) error {
    // Check for reasonable values
    if coverage.TotalLines == 0 {
        return errors.New("no lines in script")
    }
    
    if len(coverage.ExecutedLines) > coverage.TotalLines {
        return errors.New("executed lines exceed total lines")
    }
    
    if coverage.CoveragePercent > 100 {
        return errors.New("coverage percentage exceeds 100%")
    }
    
    // Check for suspicious patterns
    consecutiveLines := 0
    prev := -1
    for line := range coverage.ExecutedLines {
        if line == prev+1 {
            consecutiveLines++
            if consecutiveLines > 1000 {
                return errors.New("suspicious consecutive line pattern")
            }
        } else {
            consecutiveLines = 0
        }
        prev = line
    }
    
    return nil
}
```

## Performance Optimization

### 1. Lazy Trace Processing

Process traces incrementally:

```go
type LazyTraceProcessor struct {
    traceFile string
    processed int64
}

func (p *LazyTraceProcessor) ProcessNext(n int) ([]*TraceLine, error) {
    file, err := os.Open(p.traceFile)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    
    // Seek to last position
    file.Seek(p.processed, 0)
    
    scanner := bufio.NewScanner(file)
    lines := make([]*TraceLine, 0, n)
    
    for i := 0; i < n && scanner.Scan(); i++ {
        line := parseTraceLine(scanner.Text())
        if line != nil {
            lines = append(lines, line)
        }
        p.processed += int64(len(scanner.Bytes())) + 1
    }
    
    return lines, scanner.Err()
}
```

### 2. Concurrent Processing

Process multiple trace files concurrently:

```go
func ProcessTracesConcurrent(traceFiles []string) map[string]*BashCoverage {
    results := make(map[string]*BashCoverage)
    mu := sync.Mutex{}
    wg := sync.WaitGroup{}
    
    // Worker pool
    workers := runtime.NumCPU()
    files := make(chan string, len(traceFiles))
    
    for i := 0; i < workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for file := range files {
                if coverage, err := ProcessTraceFile(file); err == nil {
                    mu.Lock()
                    results[file] = coverage
                    mu.Unlock()
                }
            }
        }()
    }
    
    // Queue files
    for _, file := range traceFiles {
        files <- file
    }
    close(files)
    
    wg.Wait()
    return results
}
```

## Error Recovery

### 1. Partial Coverage Recovery

Recover partial coverage from corrupted traces:

```go
func RecoverPartialCoverage(traceFile string) (*BashCoverage, error) {
    file, err := os.Open(traceFile)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    
    coverage := &BashCoverage{
        ExecutedLines: make(map[int]bool),
        Partial:       true,
    }
    
    scanner := bufio.NewScanner(file)
    errors := 0
    
    for scanner.Scan() {
        line := scanner.Text()
        if parsed := parseTraceLineSafe(line); parsed != nil {
            coverage.ExecutedLines[parsed.LineNum] = true
        } else {
            errors++
            if errors > 100 {
                coverage.TooManyErrors = true
                break
            }
        }
    }
    
    return coverage, nil
}
```

### 2. Backup Mechanisms

Implement backup coverage collection:

```bash
# Primary method using BASH_XTRACEFD
primary_coverage() {
    export BASH_XTRACEFD=3
    exec 3>trace.primary
    set -x
}

# Fallback using DEBUG trap
fallback_coverage() {
    trap 'echo "${BASH_SOURCE[0]}:${LINENO}" >> trace.fallback' DEBUG
}

# Try primary, fall back if needed
if [[ -n "$BASH_XTRACEFD" ]]; then
    primary_coverage
else
    fallback_coverage
fi
```

## Conclusion

These improvements provide:

1. **Safety**: Signal handling, resource limits, isolation
2. **Reliability**: Error recovery, validation, atomicity
3. **Performance**: Lazy processing, concurrency
4. **Completeness**: Branch/loop coverage, function timing
5. **Security**: Sanitization, validation

The key principles are:
- Non-invasive instrumentation
- Fail-safe operation
- Comprehensive data collection
- Efficient processing
- Secure handling of sensitive data

These enhancements make bash coverage suitable for production use while maintaining the simplicity of the BASH_XTRACEFD approach.