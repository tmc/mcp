package fuzzing

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/tmc/mcp/exp/mcpscripttest"
)

// SpecializedGenerator generates scripts based on available commands and constraints
type SpecializedGenerator struct {
	rng      *rand.Rand
	schema   *mcpscripttest.Schema
	config   GeneratorConfig
	commands []GeneratorCommand
}

// GeneratorConfig configures the specialized generator
type GeneratorConfig struct {
	// DisabledCommands lists commands that should not be generated
	DisabledCommands map[string]bool

	// CommandWeights allows custom weighting of commands
	CommandWeights map[string]float64

	// MaxScriptLength limits the script size
	MaxScriptLength int

	// MinScriptLength ensures minimum script size
	MinScriptLength int

	// AllowDirectives controls whether to include directives like !, ?, [platform]
	AllowDirectives bool

	// FocusArea can be "mcp", "file", "exec", "mixed"
	FocusArea string
}

// GeneratorCommand represents a command that can be generated
type GeneratorCommand struct {
	Name      string
	Generator func(*SpecializedGenerator) string
	Weight    float64
}

// NewSpecializedGenerator creates a new specialized generator
func NewSpecializedGenerator(seed int64, config GeneratorConfig) *SpecializedGenerator {
	schema := mcpscripttest.GetSchema()

	g := &SpecializedGenerator{
		rng:    rand.New(rand.NewSource(seed)),
		schema: schema,
		config: config,
	}

	// Set defaults
	if g.config.MaxScriptLength == 0 {
		g.config.MaxScriptLength = 20
	}
	if g.config.MinScriptLength == 0 {
		g.config.MinScriptLength = 3
	}

	// Build command list based on schema and config
	g.buildCommandList()

	return g
}

// buildCommandList creates the list of available commands based on config
func (g *SpecializedGenerator) buildCommandList() {
	g.commands = []GeneratorCommand{}

	// Add core commands
	for _, cmd := range g.schema.CoreCommands {
		if g.config.DisabledCommands[cmd.Name] {
			continue
		}

		weight := g.config.CommandWeights[cmd.Name]
		if weight == 0 {
			weight = 1.0 // Default weight
		}

		g.commands = append(g.commands, GeneratorCommand{
			Name:      cmd.Name,
			Generator: g.getCoreCommandGenerator(cmd.Name),
			Weight:    weight,
		})
	}

	// Add MCP commands
	for _, cmd := range g.schema.MCPCommands {
		if g.config.DisabledCommands[cmd.Name] {
			continue
		}

		weight := g.config.CommandWeights[cmd.Name]
		if weight == 0 {
			weight = 1.0
		}

		// Adjust weight based on focus area
		if g.config.FocusArea == "mcp" {
			weight *= 3
		}

		g.commands = append(g.commands, GeneratorCommand{
			Name:      cmd.Name,
			Generator: g.getMCPCommandGenerator(cmd.Name),
			Weight:    weight,
		})
	}

	// Add directives if allowed
	if g.config.AllowDirectives {
		for _, dir := range g.schema.Directives {
			if g.config.DisabledCommands[dir.Name] {
				continue
			}

			weight := g.config.CommandWeights[dir.Name]
			if weight == 0 {
				weight = 0.5 // Lower weight for directives
			}

			g.commands = append(g.commands, GeneratorCommand{
				Name:      dir.Name,
				Generator: g.getDirectiveGenerator(dir.Name),
				Weight:    weight,
			})
		}
	}
}

// getCoreCommandGenerator returns a generator function for core commands
func (g *SpecializedGenerator) getCoreCommandGenerator(name string) func(*SpecializedGenerator) string {
	switch name {
	case "exec":
		return func(g *SpecializedGenerator) string {
			commands := []string{
				"echo test",
				"echo 'hello world'",
				"true",
				"false",
				"date",
				"pwd",
			}
			return fmt.Sprintf("exec %s", commands[g.rng.Intn(len(commands))])
		}
	case "stdin":
		return func(g *SpecializedGenerator) string {
			inputs := []string{
				"test input",
				`{"jsonrpc":"2.0","method":"test","id":1}`,
				"line1\nline2",
				"",
			}
			return fmt.Sprintf("stdin %s", inputs[g.rng.Intn(len(inputs))])
		}
	case "stdout", "stderr":
		return func(g *SpecializedGenerator) string {
			expectations := []string{
				"'success'",
				"result",
				"error",
				"test",
				"[0-9]+",
			}
			return fmt.Sprintf("%s %s", name, expectations[g.rng.Intn(len(expectations))])
		}
	case "cat", "rm":
		return func(g *SpecializedGenerator) string {
			files := []string{"test.txt", "output.json", "config.yaml"}
			return fmt.Sprintf("%s %s", name, files[g.rng.Intn(len(files))])
		}
	case "cp", "mv":
		return func(g *SpecializedGenerator) string {
			files := []string{"test.txt", "output.json", "config.yaml"}
			src := files[g.rng.Intn(len(files))]
			dst := files[g.rng.Intn(len(files))]
			return fmt.Sprintf("%s %s %s", name, src, dst)
		}
	case "mkdir", "cd":
		return func(g *SpecializedGenerator) string {
			dirs := []string{"test", "output", "temp", ".", ".."}
			return fmt.Sprintf("%s %s", name, dirs[g.rng.Intn(len(dirs))])
		}
	case "sleep":
		return func(g *SpecializedGenerator) string {
			return fmt.Sprintf("sleep %dms", g.rng.Intn(1000)+100)
		}
	case "wait":
		return func(g *SpecializedGenerator) string {
			if g.rng.Intn(2) == 0 {
				return "wait"
			}
			return "wait $server"
		}
	case "env":
		return func(g *SpecializedGenerator) string {
			vars := []string{"DEBUG", "PATH", "TEST_VAR"}
			values := []string{"1", "true", "/tmp/test"}
			return fmt.Sprintf("env %s=%s",
				vars[g.rng.Intn(len(vars))],
				values[g.rng.Intn(len(values))])
		}
	default:
		return func(g *SpecializedGenerator) string {
			// Unknown command - just echo it
			return fmt.Sprintf("echo unknown command: %s", name)
		}
	}
}

// getMCPCommandGenerator returns a generator function for MCP commands
func (g *SpecializedGenerator) getMCPCommandGenerator(name string) func(*SpecializedGenerator) string {
	switch name {
	case "mcp-send":
		return func(g *SpecializedGenerator) string {
			methods := []string{"initialize", "tools/list", "tools/call", "resources/list"}
			id := g.rng.Intn(100) + 1
			method := methods[g.rng.Intn(len(methods))]
			return fmt.Sprintf(`mcp-send {"jsonrpc":"2.0","method":"%s","id":%d}`, method, id)
		}
	case "mcp-recv":
		return func(g *SpecializedGenerator) string {
			expectations := []string{"result", "error", "'success'", "jsonrpc"}
			return fmt.Sprintf("mcp-recv %s", expectations[g.rng.Intn(len(expectations))])
		}
	case "mcp-serve":
		return func(g *SpecializedGenerator) string {
			servers := []string{
				"go run server.go",
				"node server.js",
				"./server",
			}
			return fmt.Sprintf("mcp-serve -- %s", servers[g.rng.Intn(len(servers))])
		}
	case "mcp-trace":
		return func(g *SpecializedGenerator) string {
			return fmt.Sprintf("mcp-trace trace_%d.mcp", g.rng.Intn(1000))
		}
	default:
		return func(g *SpecializedGenerator) string {
			return name
		}
	}
}

// getDirectiveGenerator returns a generator function for directives
func (g *SpecializedGenerator) getDirectiveGenerator(name string) func(*SpecializedGenerator) string {
	switch name {
	case "!":
		return func(g *SpecializedGenerator) string {
			return fmt.Sprintf("! exec %s", []string{"true", "false"}[g.rng.Intn(2)])
		}
	case "?":
		return func(g *SpecializedGenerator) string {
			return fmt.Sprintf("? exec %s", []string{"true", "false"}[g.rng.Intn(2)])
		}
	case "[linux]", "[!windows]":
		return func(g *SpecializedGenerator) string {
			return fmt.Sprintf("%s exec echo test", name)
		}
	case "skip":
		return func(g *SpecializedGenerator) string {
			platforms := []string{"windows", "!linux", "darwin"}
			return fmt.Sprintf("skip %s", platforms[g.rng.Intn(len(platforms))])
		}
	default:
		return func(g *SpecializedGenerator) string {
			return name
		}
	}
}

// Generate creates a test script using weighted command selection
func (g *SpecializedGenerator) Generate() string {
	var lines []string

	// Start with a comment
	lines = append(lines, "# Generated test script")
	lines = append(lines, "")

	// Calculate total weight
	totalWeight := 0.0
	for _, cmd := range g.commands {
		totalWeight += cmd.Weight
	}

	// Generate commands
	numCommands := g.config.MinScriptLength + g.rng.Intn(g.config.MaxScriptLength-g.config.MinScriptLength+1)

	for i := 0; i < numCommands; i++ {
		// Select command based on weight
		r := g.rng.Float64() * totalWeight
		cumWeight := 0.0

		for _, cmd := range g.commands {
			cumWeight += cmd.Weight
			if r <= cumWeight {
				lines = append(lines, cmd.Generator(g))
				break
			}
		}
	}

	// Ensure we have a completion marker for exec-based tests
	if !g.config.DisabledCommands["exec"] {
		hasExec := false
		for _, line := range lines {
			if strings.HasPrefix(line, "exec ") {
				hasExec = true
				break
			}
		}

		if !hasExec {
			lines = append(lines, "exec echo 'test complete'")
			lines = append(lines, "stdout 'test complete'")
		}
	}

	return strings.Join(lines, "\n")
}

// MCPTraceGenerator creates scripts focused on MCP trace testing
type MCPTraceGenerator struct {
	*SpecializedGenerator
}

// NewMCPTraceGenerator creates a generator focused on MCP trace operations
func NewMCPTraceGenerator(seed int64) *MCPTraceGenerator {
	config := GeneratorConfig{
		DisabledCommands: map[string]bool{
			"exec": true, // Disable exec for pure MCP testing
		},
		CommandWeights: map[string]float64{
			"mcp-trace": 5.0,
			"mcp-send":  3.0,
			"mcp-recv":  3.0,
			"mcp-serve": 2.0,
		},
		FocusArea:       "mcp",
		MinScriptLength: 5,
		MaxScriptLength: 15,
	}

	return &MCPTraceGenerator{
		SpecializedGenerator: NewSpecializedGenerator(seed, config),
	}
}

// SafeFileOperationsGenerator creates scripts without dangerous operations
type SafeFileOperationsGenerator struct {
	*SpecializedGenerator
}

// NewSafeFileOperationsGenerator creates a generator for safe file operations
func NewSafeFileOperationsGenerator(seed int64) *SafeFileOperationsGenerator {
	config := GeneratorConfig{
		DisabledCommands: map[string]bool{
			"exec": true,
			"rm":   true, // No deletions
		},
		CommandWeights: map[string]float64{
			"cat":    2.0,
			"cp":     1.5,
			"mv":     1.0,
			"mkdir":  1.5,
			"stdout": 2.0,
			"stderr": 1.0,
		},
		FocusArea:       "file",
		AllowDirectives: false,
	}

	return &SafeFileOperationsGenerator{
		SpecializedGenerator: NewSpecializedGenerator(seed, config),
	}
}
