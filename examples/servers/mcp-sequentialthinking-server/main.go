package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/mcptel"
)

var (
	verbose = flag.Bool("v", false, "Enable verbose logging")
	quiet   = flag.Bool("q", false, "Enable quiet mode")
)

const (
	ServerName    = "mcp-sequentialthinking-server"
	ServerVersion = "0.1.0"
)

func main() {
	flag.Parse()
	slog.SetDefault(slog.Default())
	// Redirect logs to stderr to keep stdout clean for the protocol
	if *verbose {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	} else if *quiet {
		slog.SetLogLoggerLevel(slog.LevelError)
	}
	log.Println("starting MCP Sequential Thinking Server...")

	// Create a context that can be canceled
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Create a server
	server := mcp.NewServer(ServerName, ServerVersion,
		mcp.WithServerInstructions("A MCP server for structured reasoning and sequential thinking processes"),
		// mcp.WithDefaultTracer()
	)

	// Register the sequential thinking tools
	registerTools(server)

	// Serve via stdio
	log.Println("Starting protocol server via stdio...")
	if err := server.Serve(ctx, nil); err != nil {
		if err != context.Canceled {
			log.Fatalf("Error serving: %v", err)
		}
		log.Println("Server terminated.")
	}
}

func registerTools(server *mcp.Server) {
	// Register think_step_by_step tool
	thinkStepByStepTool := mcp.Tool{
		Name:        "think_step_by_step",
		Description: "Break down a problem into sequential thinking steps",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"problem": {
					"type": "string",
					"description": "The problem or question to analyze"
				},
				"context": {
					"type": "string",
					"description": "Additional context for the problem"
				},
				"max_steps": {
					"type": "integer",
					"description": "Maximum number of thinking steps",
					"default": 5
				}
			},
			"required": ["problem"]
		}`),
	}

	server.RegisterTool(thinkStepByStepTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_ = mcptel.CurrentSpan
		server.GetLogger().InfoContext(ctx, "think_step_by_step tool called")
		var params map[string]any
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		problem, ok := params["problem"].(string)
		if !ok || problem == "" {
			return nil, fmt.Errorf("problem is required and must be a string")
		}

		maxSteps := 5
		if ms, ok := params["max_steps"].(float64); ok {
			maxSteps = int(ms)
		}

		context := ""
		if c, ok := params["context"].(string); ok {
			context = c
		}

		// Generate sequential thinking steps
		steps := generateThinkingSteps(problem, context, maxSteps)

		return &mcp.CallToolResult{
			Content: []any{
				map[string]any{
					"type": "text",
					"text": fmt.Sprintf("Sequential thinking analysis for: %s\n\n%s", problem, steps),
				},
			},
		}, nil
	})

	// Register analyze_reasoning tool
	analyzeReasoningTool := mcp.Tool{
		Name:        "analyze_reasoning",
		Description: "Analyze the logical structure of a reasoning process",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"reasoning": {
					"type": "string",
					"description": "The reasoning text to analyze"
				},
				"check_logic": {
					"type": "boolean",
					"description": "Whether to check for logical fallacies",
					"default": true
				}
			},
			"required": ["reasoning"]
		}`),
	}

	server.RegisterTool(analyzeReasoningTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_ = mcptel.CurrentSpan
		server.GetLogger().InfoContext(ctx, "analyze_reasoning tool called")
		var params map[string]any
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		reasoning, ok := params["reasoning"].(string)
		if !ok || reasoning == "" {
			return nil, fmt.Errorf("reasoning is required and must be a string")
		}

		checkLogic := true
		if cl, ok := params["check_logic"].(bool); ok {
			checkLogic = cl
		}

		analysis := analyzeReasoningStructure(reasoning, checkLogic)

		return &mcp.CallToolResult{
			Content: []any{
				map[string]any{
					"type": "text",
					"text": fmt.Sprintf("Reasoning analysis:\n%s", analysis),
				},
			},
		}, nil
	})

	// Register create_decision_tree tool
	createDecisionTreeTool := mcp.Tool{
		Name:        "create_decision_tree",
		Description: "Create a decision tree for a complex problem",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"decision": {
					"type": "string",
					"description": "The decision to be made"
				},
				"criteria": {
					"type": "array",
					"items": {"type": "string"},
					"description": "Decision criteria to consider"
				},
				"alternatives": {
					"type": "array",
					"items": {"type": "string"},
					"description": "Available alternatives"
				}
			},
			"required": ["decision"]
		}`),
	}

	server.RegisterTool(createDecisionTreeTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_ = mcptel.CurrentSpan
		server.GetLogger().InfoContext(ctx, "create_decision_tree tool called")
		var params map[string]any
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		decision, ok := params["decision"].(string)
		if !ok || decision == "" {
			return nil, fmt.Errorf("decision is required and must be a string")
		}

		var criteria []string
		if c, ok := params["criteria"].([]interface{}); ok {
			for _, item := range c {
				if str, ok := item.(string); ok {
					criteria = append(criteria, str)
				}
			}
		}

		var alternatives []string
		if a, ok := params["alternatives"].([]interface{}); ok {
			for _, item := range a {
				if str, ok := item.(string); ok {
					alternatives = append(alternatives, str)
				}
			}
		}

		tree := createDecisionTreeStructure(decision, criteria, alternatives)

		return &mcp.CallToolResult{
			Content: []any{
				map[string]any{
					"type": "text",
					"text": fmt.Sprintf("Decision tree for: %s\n\n%s", decision, tree),
				},
			},
		}, nil
	})

	log.Println("Registered Sequential Thinking tools: think_step_by_step, analyze_reasoning, create_decision_tree")
}

func generateThinkingSteps(problem, context string, maxSteps int) string {
	steps := []string{
		"1. Problem Understanding: " + strings.TrimSpace(strings.Split(problem, ".")[0]),
		"2. Information Gathering: Identify key facts and constraints",
		"3. Analysis: Break down the problem into components",
		"4. Solution Generation: Consider multiple approaches",
		"5. Evaluation: Assess pros and cons of each approach",
	}

	if context != "" {
		steps = append([]string{"0. Context Review: " + context}, steps...)
	}

	if maxSteps < len(steps) {
		steps = steps[:maxSteps]
	}

	result := strings.Join(steps, "\n")
	result += fmt.Sprintf("\n\nGenerated at: %s", time.Now().Format(time.RFC3339))
	return result
}

func analyzeReasoningStructure(reasoning string, checkLogic bool) string {
	sentences := strings.Split(reasoning, ".")
	analysis := []string{
		fmt.Sprintf("Structure Analysis: %d sentences found", len(sentences)),
		fmt.Sprintf("Word count: %d words", len(strings.Fields(reasoning))),
	}

	if checkLogic {
		// Simple logical structure analysis
		hasConclusion := strings.Contains(strings.ToLower(reasoning), "therefore") ||
			strings.Contains(strings.ToLower(reasoning), "conclusion") ||
			strings.Contains(strings.ToLower(reasoning), "thus")

		hasPremises := strings.Contains(strings.ToLower(reasoning), "because") ||
			strings.Contains(strings.ToLower(reasoning), "since") ||
			strings.Contains(strings.ToLower(reasoning), "given")

		analysis = append(analysis,
			fmt.Sprintf("Has clear premises: %t", hasPremises),
			fmt.Sprintf("Has conclusion indicators: %t", hasConclusion),
		)
	}

	return strings.Join(analysis, "\n")
}

func createDecisionTreeStructure(decision string, criteria, alternatives []string) string {
	tree := []string{
		fmt.Sprintf("Decision: %s", decision),
		"",
		"Decision Tree Structure:",
		"├── Criteria:",
	}

	if len(criteria) == 0 {
		criteria = []string{"Cost", "Time", "Quality", "Risk"}
	}

	for i, criterion := range criteria {
		prefix := "│   ├── "
		if i == len(criteria)-1 {
			prefix = "│   └── "
		}
		tree = append(tree, prefix+criterion)
	}

	tree = append(tree, "", "├── Alternatives:")

	if len(alternatives) == 0 {
		alternatives = []string{"Option A", "Option B", "Option C"}
	}

	for i, alt := range alternatives {
		prefix := "│   ├── "
		if i == len(alternatives)-1 {
			prefix = "│   └── "
		}
		tree = append(tree, prefix+alt)
	}

	tree = append(tree, "", "└── Evaluation Matrix:")
	tree = append(tree, "    (Rate each alternative against each criterion)")

	return strings.Join(tree, "\n")
}
