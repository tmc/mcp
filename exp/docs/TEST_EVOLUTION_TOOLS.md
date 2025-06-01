# Test Evolution and Mutation Tools

This document describes tools for evolving, mutating, and optimizing scripttest files for MCP development.

## Test Mutation Tools

### 1. mcp-test-mutate
```bash
mcp-test-mutate test.txt --strategies=all --count=10
```

Generates test variations to explore edge cases and improve robustness.

#### Mutation Strategies

**Command Mutations:**
```diff
# Original
exec mcp-tool call add '{"x": 1, "y": 2}'
stdout '{"result": 3}'

# Mutated: Reordered arguments
exec mcp-tool call add '{"y": 2, "x": 1}'
stdout '{"result": 3}'

# Mutated: Extra whitespace
exec mcp-tool call add '{"x":1,"y":2}'
stdout '{"result": 3}'

# Mutated: Type variations
exec mcp-tool call add '{"x": 1.0, "y": 2.0}'
stdout '{"result": 3.0}'
```

**Input Fuzzing:**
```diff
# Original
exec mcp-tool call divide '{"x": 10, "y": 2}'
stdout '{"result": 5}'

# Mutated: Zero division
exec mcp-tool call divide '{"x": 10, "y": 0}'
stderr 'division by zero'

# Mutated: Negative numbers
exec mcp-tool call divide '{"x": -10, "y": 2}'
stdout '{"result": -5}'

# Mutated: Large numbers
exec mcp-tool call divide '{"x": 1e308, "y": 1e-308}'
```

**Timing Mutations:**
```diff
# Original
exec mcp-server &
exec sleep 1
exec mcp-tool call ping

# Mutated: Faster timing
exec mcp-server &
exec sleep 0.1
exec mcp-tool call ping

# Mutated: No delay
exec mcp-server &
exec mcp-tool call ping
```

**Error Injection:**
```diff
# Original
exec mcp-tool call fetch '{"url": "http://example.com"}'
stdout 'content'

# Mutated: Network error
exec mcp-tool call fetch '{"url": "http://unreachable.test"}'
stderr 'network error'

# Mutated: Invalid URL
exec mcp-tool call fetch '{"url": "not-a-url"}'
stderr 'invalid URL'
```

### 2. mcp-test-breed
```bash
mcp-test-breed test1.txt test2.txt --offspring=5
```

Combines successful test patterns using genetic algorithms:

```
# Parent 1: Basic addition test
exec mcp-tool call add '{"x": 1, "y": 2}'
stdout '{"result": 3}'

# Parent 2: Error handling test
exec mcp-tool call add '{"x": "invalid"}'
stderr 'type error'

# Offspring: Combined test
exec mcp-tool call add '{"x": 1, "y": 2}'
stdout '{"result": 3}'
exec mcp-tool call add '{"x": "invalid"}'
stderr 'type error'
exec mcp-tool call add '{"x": 1, "y": "invalid"}'
stderr 'type error'
```

### 3. mcp-test-minimize
```bash
mcp-test-minimize failing_test.txt --output=minimal.txt
```

Reduces failing tests to minimal reproduction:

```diff
# Original failing test (50 lines)
exec mcp-server start
exec mcp-tool initialize
exec mcp-tool call setup
exec mcp-tool call configure '{"option": "value"}'
# ... many more lines ...
exec mcp-tool call process '{"data": "test"}'
! stderr 'error'

# Minimized (3 lines)
exec mcp-server start
exec mcp-tool call process '{"data": "test"}'
! stderr 'error'
```

## Test Evolution Tools

### 4. mcp-test-evolve
```bash
mcp-test-evolve --trace=server_trace.jsonl --test=basic.txt
```

Evolves tests based on observed behavior:

```diff
# Original test
exec mcp-server
exec mcp-tool list
stdout 'add'

# Evolved from trace
exec mcp-server
exec mcp-tool list
stdout 'add'
stdout 'subtract'
stdout 'multiply'
+
+# New tool discovered in trace
+exec mcp-tool call multiply '{"x": 3, "y": 4}'
+stdout '{"result": 12}'
+
+# Edge case from trace
+exec mcp-tool call multiply '{"x": 0, "y": 100}'
+stdout '{"result": 0}'
```

### 5. mcp-test-learn
```bash
mcp-test-learn --traces=traces/*.jsonl --output=learned_tests.txt
```

Learns test patterns from multiple traces:

```
# Learned pattern: Initialization sequence
exec mcp-server
exec mcp-tool initialize
exec mcp-tool capabilities
stdout 'tools'

# Learned pattern: Error recovery
exec mcp-tool call failing_operation
stderr 'temporary error'
exec sleep 1
exec mcp-tool call failing_operation
stdout 'success'

# Learned pattern: Resource cleanup
exec mcp-tool call create_resource
stdout 'resource_id'
exec mcp-tool call use_resource
exec mcp-tool call cleanup_resource
```

### 6. mcp-test-adapt
```bash
mcp-test-adapt old_test.txt --new-version=v2.0
```

Adapts tests to new API versions:

```diff
# Old test (v1.0)
exec mcp-tool call calculate '{"x": 1, "y": 2}'
stdout '3'

# Adapted for v2.0
exec mcp-tool call calculate '{"values": [1, 2]}'
stdout '{"result": 3, "operation": "sum"}'
```

## Coverage-Driven Evolution

### 7. mcp-test-cover
```bash
mcp-test-cover --binary=mcp-server --test=basic.txt --evolve
```

Evolves tests to increase code coverage:

```
Initial coverage: 45%

Generating test for uncovered branch at server.go:123
+ exec mcp-tool call process '{"mode": "async"}'
+ stdout 'processing'

Generating test for error handler at server.go:145
+ exec mcp-tool call process '{"data": null}'
+ stderr 'invalid data'

Final coverage: 78%
```

### 8. mcp-test-path
```bash
mcp-test-path --source=server.go --target="func HandleError"
```

Generates tests targeting specific code paths:

```
# Generated test reaching HandleError
exec mcp-server
exec mcp-tool call process '{"invalid": "json"}'
stderr 'HandleError: invalid request'

# Alternative path
exec mcp-server
exec mcp-tool initialize
! exec mcp-tool initialize  # Double initialization
stderr 'HandleError: already initialized'
```

## Property-Based Testing

### 9. mcp-test-property
```bash
mcp-test-property --tool=calculator --property="commutative"
```

Generates property-based tests:

```
# Property: Addition is commutative
exec mcp-tool call add '{"x": 5, "y": 3}'
stdout '{"result": 8}'
exec mcp-tool call add '{"x": 3, "y": 5}'
stdout '{"result": 8}'

# Property: Multiplication by zero
exec mcp-tool call multiply '{"x": 0, "y": 42}'
stdout '{"result": 0}'
exec mcp-tool call multiply '{"x": 42, "y": 0}'
stdout '{"result": 0}'

# Property: Identity element
exec mcp-tool call add '{"x": 7, "y": 0}'
stdout '{"result": 7}'
```

### 10. mcp-test-invariant
```bash
mcp-test-invariant --trace=trace.jsonl --find-invariants
```

Discovers and tests invariants:

```
# Discovered invariant: Response time < 100ms
exec mcp-server
exec mcp-tool call fast_operation
exec mcp-tool timestamp
stdout within 100ms

# Discovered invariant: State consistency
exec mcp-tool call set_state '{"value": 42}'
exec mcp-tool call get_state
stdout '{"value": 42}'
exec mcp-tool call increment_state
exec mcp-tool call get_state
stdout '{"value": 43}'
```

## Test Optimization

### 11. mcp-test-optimize
```bash
mcp-test-optimize test_suite.txt --target=speed
```

Optimizes test execution:

```diff
# Original: Sequential tests
exec mcp-server
exec mcp-tool call op1
exec mcp-tool call op2
exec mcp-tool call op3

# Optimized: Parallel execution
exec mcp-server
exec -parallel mcp-tool call op1 &
exec -parallel mcp-tool call op2 &
exec -parallel mcp-tool call op3 &
wait
```

### 12. mcp-test-dedupe
```bash
mcp-test-dedupe test_suite.txt --output=unique_tests.txt
```

Removes redundant tests:

```diff
# Test 1
exec mcp-tool call add '{"x": 1, "y": 2}'
stdout '{"result": 3}'

-# Test 2 (Redundant)
-exec mcp-tool call add '{"x": 1, "y": 2}'
-stdout '{"result": 3}'

# Test 3 (Unique)
exec mcp-tool call add '{"x": -1, "y": 2}'
stdout '{"result": 1}'
```

## Continuous Evolution

### 13. mcp-test-monitor
```bash
mcp-test-monitor --server=mcp-server --update-tests
```

Continuously evolves tests based on production behavior:

```
Monitoring MCP server at :8080

New pattern detected: Retry behavior
Generated test:
  exec mcp-tool call flaky_operation
  stderr 'temporary failure'
  exec sleep 1
  exec mcp-tool call flaky_operation
  stdout 'success'

New error detected: Invalid auth token
Generated test:
  exec mcp-tool --auth=expired call protected
  stderr 'authentication failed'
```

### 14. mcp-test-drift
```bash
mcp-test-drift --baseline=v1.0 --current=v2.0
```

Detects API drift and updates tests:

```
API Changes Detected:

1. Parameter renamed: 'x' -> 'value1'
   Updating 15 tests...

2. New required field: 'options'
   Adding default: '{"options": {}}'

3. Response format changed
   Old: '3'
   New: '{"result": 3}'
   Updating assertions...
```

## Test Synthesis

### 15. mcp-test-synthesize
```bash
mcp-test-synthesize --spec=api_spec.yaml --examples=5
```

Synthesizes tests from specifications:

```yaml
# Input: API Spec
endpoints:
  - name: calculate
    params:
      operation: enum[add, subtract, multiply, divide]
      values: array[number]
    returns: number
```

```
# Generated tests
exec mcp-tool call calculate '{"operation": "add", "values": [1, 2, 3]}'
stdout '{"result": 6}'

exec mcp-tool call calculate '{"operation": "multiply", "values": [2, 3, 4]}'
stdout '{"result": 24}'

exec mcp-tool call calculate '{"operation": "divide", "values": [10, 2]}'
stdout '{"result": 5}'
```

## Complete Example

Here's a complete workflow showing test evolution:

```bash
# 1. Start with basic test
cat > basic.txt << EOF
exec mcp-server
exec mcp-tool list
stdout 'echo'
EOF

# 2. Run and capture trace
mcp-trace-record mcp-test-run basic.txt > trace.jsonl

# 3. Evolve based on trace
mcp-test-evolve --trace=trace.jsonl --test=basic.txt > evolved.txt

# 4. Mutate for edge cases
mcp-test-mutate evolved.txt --count=20 > mutations/

# 5. Breed successful mutations
mcp-test-breed mutations/success_*.txt --offspring=10

# 6. Minimize failures
for f in mutations/fail_*.txt; do
  mcp-test-minimize "$f" > "minimal_$(basename $f)"
done

# 7. Optimize suite
cat evolved.txt mutations/success_*.txt | mcp-test-optimize > optimized_suite.txt

# 8. Remove redundancy
mcp-test-dedupe optimized_suite.txt > final_suite.txt

# 9. Check coverage
mcp-test-cover --binary=mcp-server --test=final_suite.txt

# 10. Generate property tests
mcp-test-property --trace=trace.jsonl >> final_suite.txt

# Result: Comprehensive, optimized test suite
```

## Best Practices

1. **Start Simple**: Begin with basic tests and evolve incrementally
2. **Use Traces**: Real execution traces provide the best evolution data
3. **Combine Strategies**: Use multiple mutation types for better coverage
4. **Minimize Failures**: Always reduce failing tests to minimal cases
4. **Monitor Coverage**: Track coverage improvements through evolution
5. **Version Control**: Track test evolution in git
6. **Regular Updates**: Re-evolve tests as APIs change
7. **Human Review**: Review evolved tests for readability

## Future Directions

1. **AI-Powered Evolution**: Use LLMs to generate smarter mutations
2. **Cross-Language**: Evolve tests across implementation languages
3. **Visual Evolution**: GUI for interactive test evolution
4. **Distributed Evolution**: Cloud-based test evolution at scale
5. **Behavioral Learning**: Learn from user interactions
6. **Automatic Repair**: Fix broken tests automatically

These tools enable continuous improvement of test suites through automated evolution and mutation strategies.