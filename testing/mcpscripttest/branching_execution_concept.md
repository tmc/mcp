# Branching Execution Concept for Scripttest

## Visual Representation

```
                    Base Snapshot
                    (tools + txtar)
                          |
          +---------------+---------------+
          |               |               |
      Branch 1        Branch 2        Branch 3
         |               |               |
   exec mcpdiff     exec mcpcat     exec mcpspy
         |               |               |
     Coverage:        Coverage:       Coverage:
     11.2%            8.5%            15.3%
         |               |               |
         ✓               ✓               ✗ (failed)
                                         |
                                    Retry Branch
                                         |
                                    Debug Mode
                                         |
                                    Coverage: 15.8%
                                         ✓
```

## Implementation Example

```go
package scripttest

import (
    "sync"
    "time"
)

// TestSnapshot represents a snapshot of test environment
type TestSnapshot struct {
    BaseDir    string
    Tools      map[string]string  // tool -> path
    TxtarFiles map[string][]byte  // filename -> content
    Env        []string
}

// Branch represents an isolated test execution branch
type Branch struct {
    ID       string
    Snapshot *TestSnapshot
    WorkDir  string
}

// BranchingExecutor runs scripttest lines in isolated branches
type BranchingExecutor struct {
    snapshot    *TestSnapshot
    maxRetries  int
    parallelism int
}

// ExecuteScripttest runs an entire test file with branching
func (be *BranchingExecutor) ExecuteScripttest(testFile string) *TestResults {
    // Parse test file
    lines, txtar := parseTestFile(testFile)
    
    // Create base snapshot
    be.snapshot = be.createSnapshot(txtar)
    
    // Execute lines in parallel branches
    results := make([]*LineResult, len(lines))
    semaphore := make(chan struct{}, be.parallelism)
    var wg sync.WaitGroup
    
    for i, line := range lines {
        wg.Add(1)
        semaphore <- struct{}{} // Limit parallelism
        
        go func(idx int, testLine TestLine) {
            defer wg.Done()
            defer func() { <-semaphore }()
            
            // Initial execution
            result := be.executeLine(testLine)
            
            // Retry failed lines
            if result.Failed && be.maxRetries > 0 {
                for retry := 0; retry < be.maxRetries; retry++ {
                    // Create fresh branch for retry
                    result = be.executeLineWithDebug(testLine, retry)
                    if !result.Failed {
                        break
                    }
                }
            }
            
            results[idx] = result
        }(i, line)
    }
    
    wg.Wait()
    return &TestResults{Lines: results}
}

// executeLine runs a single test line in an isolated branch
func (be *BranchingExecutor) executeLine(line TestLine) *LineResult {
    branch := be.createBranch()
    defer branch.Cleanup()
    
    start := time.Now()
    
    // Set up isolated environment
    branch.SetupEnvironment()
    branch.ExtractTxtarFiles()
    
    // Execute the line
    output, err := branch.Execute(line.Command)
    
    // Collect coverage
    coverage := branch.CollectCoverage()
    
    return &LineResult{
        Line:     line,
        Output:   output,
        Error:    err,
        Coverage: coverage,
        Duration: time.Since(start),
    }
}

// createBranch creates an isolated execution branch
func (be *BranchingExecutor) createBranch() *Branch {
    // This could use various isolation mechanisms:
    // - Docker container with overlay fs
    // - Firecracker microVM
    // - Filesystem namespace + chroot
    // - User-mode Linux
    
    return &Branch{
        ID:       generateID(),
        Snapshot: be.snapshot,
        WorkDir:  createIsolatedWorkdir(),
    }
}

// Example usage with different isolation backends
type IsolationBackend interface {
    CreateBranch(snapshot *TestSnapshot) *Branch
    Fork(branch *Branch) *Branch
    Cleanup(branch *Branch)
}

// DockerBackend uses Docker with overlay filesystem
type DockerBackend struct{}

func (d *DockerBackend) CreateBranch(snapshot *TestSnapshot) *Branch {
    // docker run --rm -v overlay:/work ...
    return &Branch{}
}

// FirecrackerBackend uses Firecracker microVMs
type FirecrackerBackend struct{}

func (f *FirecrackerBackend) CreateBranch(snapshot *TestSnapshot) *Branch {
    // Create microVM from snapshot
    return &Branch{}
}

// Benefits in Practice

/*
1. True Isolation:
   - No state pollution between test lines
   - Clean environment for each execution
   - Accurate coverage measurement

2. Parallel Execution:
   - Run all test lines simultaneously
   - Limited only by available resources
   - Dramatic speedup for large test suites

3. Smart Retries:
   - Failed tests can be retried with debugging
   - Different retry strategies (timeouts, resources)
   - Automatic failure analysis

4. Advanced Debugging:
   - Pause and inspect failed branches
   - Step through test execution
   - Compare successful vs failed branches

5. Coverage Optimization:
   - Accurate per-line coverage
   - No interference between lines
   - Clear incremental coverage data
*/
```

## Execution Timeline

```
Time →

T0: Create base snapshot
    |
T1: |--Branch 1--|    (exec mcpdiff file1 file2)
    |--Branch 2------|    (exec mcpcat -v file1)
    |--Branch 3-----------|    (exec mcpspy -- server)
    |
T2: |--Retry Branch 3-----|    (failed, retry with debug)
    |
T3: Aggregate results
```

## Future Possibilities

1. **State Space Exploration**: Try different input combinations
2. **Fuzzing Integration**: Generate test variations automatically
3. **Performance Profiling**: Compare execution across branches
4. **Resource Optimization**: Share common resources between branches
5. **CI/CD Integration**: Parallel test execution in cloud environments

This branching approach would revolutionize how we think about test execution, moving from sequential, stateful testing to parallel, isolated, and debuggable test runs.