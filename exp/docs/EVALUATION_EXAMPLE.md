# Evaluation Tools in Practice

## Real-World Example: Optimizing MCP Code Server

Let's walk through how we'd use the evaluation tools to improve an MCP code server by comparing different implementations and configurations.

### Scenario Setup

We have three different MCP code server implementations:
1. **Basic Code Server**: Simple file operations and syntax checking
2. **Enhanced Code Server**: Adds AST analysis and refactoring
3. **AI-Powered Code Server**: Includes ML-based suggestions

### Step 1: Define Evaluation Tasks

```yaml
# tasks/refactoring.yaml
name: "Extract Method Refactoring"
description: "Extract a complex method into smaller, reusable functions"
type: "coding"
language: "go"
difficulty: "medium"
timeout: 300s
initial_code: |
  func ProcessOrder(order Order) error {
      // Validate order
      if order.ID == "" {
          return errors.New("order ID required")
      }
      if order.CustomerID == "" {
          return errors.New("customer ID required")
      }
      if len(order.Items) == 0 {
          return errors.New("order must have items")
      }
      
      // Calculate total
      var total float64
      for _, item := range order.Items {
          if item.Quantity <= 0 {
              return errors.New("invalid quantity")
          }
          total += item.Price * float64(item.Quantity)
      }
      
      // Apply discount
      if order.DiscountCode != "" {
          discount, err := GetDiscount(order.DiscountCode)
          if err != nil {
              return err
          }
          total = total * (1 - discount.Percentage)
      }
      
      // Process payment
      payment := Payment{
          Amount: total,
          Method: order.PaymentMethod,
      }
      if err := ProcessPayment(payment); err != nil {
          return err
      }
      
      // Update inventory
      for _, item := range order.Items {
          if err := UpdateInventory(item.ProductID, -item.Quantity); err != nil {
              return err
          }
      }
      
      return nil
  }
expected_refactoring:
  extracted_methods: 3
  max_method_lines: 15
  test_coverage: 90
```

### Step 2: Configure Agents

```json
// agents/claude-basic.json
{
  "name": "Claude-Basic",
  "type": "anthropic",
  "model": "claude-3-opus",
  "mcp_servers": [
    {
      "name": "basic-code-server",
      "command": "mcp-code-server",
      "args": ["--mode=basic"]
    }
  ]
}

// agents/claude-enhanced.json
{
  "name": "Claude-Enhanced",
  "type": "anthropic",
  "model": "claude-3-opus",
  "mcp_servers": [
    {
      "name": "enhanced-code-server",
      "command": "mcp-code-server",
      "args": ["--mode=enhanced", "--ast=true"]
    }
  ]
}

// agents/claude-ai.json
{
  "name": "Claude-AI",
  "type": "anthropic",
  "model": "claude-3-opus",
  "mcp_servers": [
    {
      "name": "ai-code-server",
      "command": "mcp-code-server",
      "args": ["--mode=ai", "--ml-suggestions=true"]
    }
  ]
}
```

### Step 3: Run Competition

```bash
# Run head-to-head comparison
mcp-arena compete agents/*.json --task=tasks/refactoring.yaml

# Output:
Competition ID: c4f2a8b1-3d4e-4f5a-9b6c-7d8e9f0a1b2c
Task: Extract Method Refactoring
Agents: Claude-Basic, Claude-Enhanced, Claude-AI

Running competitions...
[████████████████████████████████] 100%

Results:
┌─────────────────┬─────────┬───────┬──────────┬─────────────┐
│ Agent           │ Success │ Score │ Duration │ Interactions│
├─────────────────┼─────────┼───────┼──────────┼─────────────┤
│ Claude-Basic    │ ✓       │ 0.65  │ 2m 34s   │ 12          │
│ Claude-Enhanced │ ✓       │ 0.87  │ 1m 47s   │ 8           │
│ Claude-AI       │ ✓       │ 0.94  │ 1m 23s   │ 6           │
└─────────────────┴─────────┴───────┴──────────┴─────────────┘
```

### Step 4: Analyze Results

```bash
# Deep dive into the results
mcp-eval analyze c4f2a8b1-3d4e-4f5a-9b6c-7d8e9f0a1b2c

# Output:
## Analysis Report

### Claude-Basic
- Extracted 2/3 expected methods
- Average method length: 18 lines (over limit)
- Test coverage: 75%
- Used 12 MCP tool calls:
  - ReadFile: 3
  - WriteFile: 4
  - RunTests: 5
- Struggled with complex refactoring patterns

### Claude-Enhanced  
- Extracted all 3 methods correctly
- Average method length: 12 lines
- Test coverage: 88%
- Used 8 MCP tool calls:
  - ReadFile: 1
  - AST.Parse: 2
  - AST.Refactor: 2
  - WriteFile: 2
  - RunTests: 1
- Efficient use of AST tools

### Claude-AI
- Extracted 3 methods + 1 helper
- Average method length: 10 lines  
- Test coverage: 95%
- Used 6 MCP tool calls:
  - ReadFile: 1
  - AI.SuggestRefactoring: 1
  - WriteFile: 3
  - RunTests: 1
- ML suggestions reduced trial-and-error
```

### Step 5: Compare Tool Usage Patterns

```bash
# Differential analysis
mcp-diff patterns c4f2a8b1-3d4e-4f5a-9b6c-7d8e9f0a1b2c

# Output:
## Tool Usage Patterns

### Pattern 1: Initial Analysis
Claude-Basic:    ReadFile → ReadFile → ReadFile
Claude-Enhanced: ReadFile → AST.Parse
Claude-AI:       ReadFile → AI.SuggestRefactoring

### Pattern 2: Refactoring Strategy
Claude-Basic:    Manual extraction → Test → Fix → Test
Claude-Enhanced: AST analysis → Targeted refactor → Test
Claude-AI:       AI suggestion → Apply → Test

### Pattern 3: Error Recovery
Claude-Basic:    Test failure → Re-read → Manual fix → Test
Claude-Enhanced: Test failure → AST.analyze → Fix → Test
Claude-AI:       Test failure → AI.debug → Fix → Test

### Efficiency Gain:
- Enhanced vs Basic: 33% fewer interactions
- AI vs Basic: 50% fewer interactions
- AI vs Enhanced: 25% fewer interactions
```

### Step 6: Generate Performance Profile

```bash
# Benchmark specific operations
mcp-bench profile code-servers --operation=refactor

# Output:
## Performance Profile: Refactoring Operation

### Basic Code Server
- Parse time: 45ms
- Analysis time: N/A
- Refactor time: N/A (manual)
- Total: 45ms + manual work

### Enhanced Code Server  
- Parse time: 52ms
- AST analysis: 128ms
- Refactor time: 89ms
- Total: 269ms

### AI Code Server
- Parse time: 48ms
- ML inference: 340ms
- Refactor time: 72ms
- Total: 460ms

### Trade-offs:
- Basic: Fast but requires more agent work
- Enhanced: Balanced speed and capability
- AI: Slower but highest success rate
```

### Step 7: Run A/B Test on Configuration

```bash
# Test configuration changes
mcp-replay ab-test c4f2a8b1 \
  --server-a="ai-code-server" \
  --server-b="ai-code-server --cache=true --parallel=true"

# Output:
## A/B Test Results

Configuration A (baseline):
- Average latency: 460ms
- Success rate: 94%
- Tool calls: 6

Configuration B (optimized):
- Average latency: 285ms (-38%)
- Success rate: 93% (-1%)
- Tool calls: 6

Statistical significance: p=0.003
Recommendation: Deploy configuration B
```

### Step 8: Generate Training Dataset

```bash
# Create dataset from successful runs
mcp-dataset generate \
  --from=competitions/*.json \
  --filter="score>0.85" \
  --output=refactoring-dataset.jsonl

# Output:
Generated dataset with 147 examples
- Successful refactorings: 147
- Average score: 0.91
- Unique patterns: 23
- Edge cases captured: 8

Dataset saved to: refactoring-dataset.jsonl
```

### Step 9: Learn from Patterns

```bash
# Extract insights
mcp-learn strategies --from=refactoring-dataset.jsonl

# Output:
## Learned Strategies

### Successful Patterns:
1. AST-guided refactoring (87% success)
   - Parse → Identify boundaries → Extract → Test
   
2. AI-assisted planning (94% success)
   - Get suggestions → Validate → Apply → Test

3. Incremental extraction (82% success)
   - Extract one method → Test → Repeat

### Failure Patterns:
1. Over-extraction (12% of failures)
   - Creating too many small methods
   
2. Missing dependencies (34% of failures)
   - Not identifying shared state

3. Test coverage gaps (28% of failures)
   - Not testing edge cases

### Recommendations:
- Use AI suggestions for complex refactoring
- Always validate AST analysis results
- Test after each extraction
- Check for shared state before extracting
```

### Step 10: Continuous Improvement

```bash
# Set up continuous evaluation
mcp-arena continuous \
  --config=continuous-eval.yaml \
  --notify=slack \
  --threshold=0.02

# continuous-eval.yaml
schedule: "0 */6 * * *"  # Every 6 hours
agents:
  - claude-enhanced
  - claude-ai
tasks: 
  - tasks/refactoring/*.yaml
  - tasks/debugging/*.yaml
baseline: claude-enhanced
metrics:
  - success_rate
  - average_score
  - tool_efficiency
alerts:
  regression: -0.02
  improvement: +0.05
```

## Key Insights from Evaluation

1. **Tool Effectiveness**:
   - AST tools reduce interactions by 33%
   - AI suggestions reduce interactions by 50%
   - Caching improves latency by 38%

2. **Configuration Impact**:
   - Parallel processing helps AI server
   - Cache critical for repeated operations
   - Trade-off between latency and accuracy

3. **Learning Outcomes**:
   - Clear patterns in successful refactoring
   - Specific failure modes identified
   - Actionable improvements discovered

4. **Dataset Value**:
   - 147 high-quality examples generated
   - Can train better agents
   - Captures edge cases

## Next Steps

1. Deploy optimized AI server configuration
2. Train specialized refactoring model
3. Add more task types to evaluation suite
4. Monitor production performance
5. Iterate based on continuous evaluation

This example shows how evaluation tools create a data-driven development cycle for MCP servers, enabling systematic improvement based on real performance data.