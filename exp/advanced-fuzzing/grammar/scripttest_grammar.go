// Package grammar provides comprehensive grammar-guided generation for ScriptTest content.
// This coordinates with Go tools to generate maximum coverage test scenarios.
package grammar

import (
	"context"
	"fmt"
	"internal/fuzz"
	"math/rand"
	"runtime/coverage"
	"strings"
	"sync"
	"time"
)

// GrammarEngine generates structured test content using grammar rules
type GrammarEngine struct {
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc

	// Core components
	rules          map[string]*GrammarRule
	generators     map[string]Generator
	coordinator    *fuzz.MultiModalCoordinator
	coverageOracle coverage.LLMLOracle

	// Generation state
	generationHistory []*GenerationResult
	activePatterns    map[string]*Pattern

	// Configuration
	maxDepth         int
	maxVariations    int
	creativityLevel  float64
	useSemanticGuide bool

	// Metrics
	totalGenerations int64
	successfulGens   int64
	averageQuality   float64

	// Event handlers
	onGenerationComplete func(*GenerationResult)
	onPatternDiscovered  func(*Pattern)
}

// GrammarRule defines a rule for generating content
type GrammarRule struct {
	Name         string                 `json:"name"`
	Pattern      string                 `json:"pattern"`
	Alternatives []string               `json:"alternatives"`
	Constraints  []Constraint           `json:"constraints"`
	Weight       float64                `json:"weight"`
	Context      string                 `json:"context"`
	Examples     []string               `json:"examples"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// Generator interface for content generation
type Generator interface {
	Generate(ctx context.Context, rule *GrammarRule, depth int) (string, error)
	GetName() string
	GetCapabilities() []string
}

// Pattern represents a discovered pattern in generated content
type Pattern struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Template     string    `json:"template"`
	Frequency    int       `json:"frequency"`
	SuccessRate  float64   `json:"success_rate"`
	QualityScore float64   `json:"quality_score"`
	LastUsed     time.Time `json:"last_used"`
	Context      []string  `json:"context"`
}

// GenerationResult contains the results of content generation
type GenerationResult struct {
	ID             string                 `json:"id"`
	Content        string                 `json:"content"`
	Rules          []*GrammarRule         `json:"rules"`
	Quality        float64                `json:"quality"`
	Coverage       float64                `json:"coverage"`
	GeneratedAt    time.Time              `json:"generated_at"`
	GenerationTime time.Duration          `json:"generation_time"`
	Success        bool                   `json:"success"`
	Patterns       []*Pattern             `json:"patterns"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// Constraint defines constraints for rule application
type Constraint struct {
	Type     string      `json:"type"`
	Value    interface{} `json:"value"`
	Operator string      `json:"operator"`
	Context  string      `json:"context"`
}

// Built-in generators

// GoToolGenerator generates Go tool command sequences
type GoToolGenerator struct {
	name         string
	capabilities []string
	toolChains   map[string][]string
}

// ScriptTestGenerator generates ScriptTest-specific content
type ScriptTestGenerator struct {
	name         string
	capabilities []string
	commands     map[string]*CommandTemplate
}

// CommandTemplate defines templates for command generation
type CommandTemplate struct {
	Command    string            `json:"command"`
	Args       []string          `json:"args"`
	Flags      map[string]string `json:"flags"`
	Context    string            `json:"context"`
	Complexity float64           `json:"complexity"`
	Examples   []string          `json:"examples"`
}

// NewGrammarEngine creates a new grammar engine
func NewGrammarEngine(ctx context.Context) *GrammarEngine {
	engineCtx, cancel := context.WithCancel(ctx)

	engine := &GrammarEngine{
		ctx:               engineCtx,
		cancel:            cancel,
		rules:             make(map[string]*GrammarRule),
		generators:        make(map[string]Generator),
		generationHistory: make([]*GenerationResult, 0),
		activePatterns:    make(map[string]*Pattern),
		maxDepth:          5,
		maxVariations:     10,
		creativityLevel:   0.7,
		useSemanticGuide:  true,
	}

	// Initialize built-in generators
	engine.initializeBuiltinGenerators()

	// Initialize default grammar rules
	engine.initializeDefaultRules()

	return engine
}

// SetCoordinator sets the fuzzing coordinator
func (e *GrammarEngine) SetCoordinator(coordinator *fuzz.MultiModalCoordinator) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.coordinator = coordinator
}

// SetCoverageOracle sets the coverage oracle
func (e *GrammarEngine) SetCoverageOracle(oracle coverage.LLMOracle) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.coverageOracle = oracle
}

// SetEventHandlers configures event callbacks
func (e *GrammarEngine) SetEventHandlers(
	onGenerationComplete func(*GenerationResult),
	onPatternDiscovered func(*Pattern),
) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onGenerationComplete = onGenerationComplete
	e.onPatternDiscovered = onPatternDiscovered
}

// GenerateContent generates content using grammar rules
func (e *GrammarEngine) GenerateContent(ctx context.Context, targetRule string, options map[string]interface{}) (*GenerationResult, error) {
	startTime := time.Now()

	e.mu.RLock()
	rule, exists := e.rules[targetRule]
	e.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("rule %s not found", targetRule)
	}

	// Apply options
	e.applyGenerationOptions(options)

	// Generate content
	content, usedRules, err := e.generateWithRule(ctx, rule, 0)
	if err != nil {
		return nil, fmt.Errorf("generation failed: %w", err)
	}

	// Evaluate quality
	quality, err := e.evaluateQuality(ctx, content)
	if err != nil {
		quality = 0.5 // Default quality if evaluation fails
	}

	// Calculate coverage impact
	coverage := e.estimateCoverageImpact(content, usedRules)

	// Detect patterns
	patterns := e.detectPatterns(content, usedRules)

	// Create result
	result := &GenerationResult{
		ID:             e.generateResultID(),
		Content:        content,
		Rules:          usedRules,
		Quality:        quality,
		Coverage:       coverage,
		GeneratedAt:    time.Now(),
		GenerationTime: time.Since(startTime),
		Success:        quality > 0.3,
		Patterns:       patterns,
		Metadata: map[string]interface{}{
			"target_rule":   targetRule,
			"depth_used":    e.calculateMaxDepthUsed(usedRules),
			"rules_applied": len(usedRules),
		},
	}

	// Store result
	e.mu.Lock()
	e.generationHistory = append(e.generationHistory, result)
	e.totalGenerations++
	if result.Success {
		e.successfulGens++
		e.averageQuality = (e.averageQuality + quality) / 2.0
	}
	// Update patterns
	for _, pattern := range patterns {
		e.activePatterns[pattern.ID] = pattern
	}
	e.mu.Unlock()

	// Trigger event handlers
	if e.onGenerationComplete != nil {
		go e.onGenerationComplete(result)
	}
	for _, pattern := range patterns {
		if e.onPatternDiscovered != nil {
			go e.onPatternDiscovered(pattern)
		}
	}

	return result, nil
}

// GenerateTestScenario generates a complete test scenario
func (e *GrammarEngine) GenerateTestScenario(ctx context.Context, scenario string) (*GenerationResult, error) {
	scenarios := map[string]string{
		"basic_build":    "go_build_sequence",
		"test_coverage":  "coverage_test_sequence",
		"module_ops":     "module_operation_sequence",
		"cross_compile":  "cross_compile_sequence",
		"performance":    "performance_test_sequence",
		"integration":    "integration_test_sequence",
		"error_handling": "error_handling_sequence",
	}

	ruleName, exists := scenarios[scenario]
	if !exists {
		return nil, fmt.Errorf("unknown scenario: %s", scenario)
	}

	return e.GenerateContent(ctx, ruleName, map[string]interface{}{
		"scenario":              scenario,
		"optimize_for_coverage": true,
	})
}

// GetRecommendedScenarios returns scenarios recommended by coverage analysis
func (e *GrammarEngine) GetRecommendedScenarios(ctx context.Context) ([]string, error) {
	if e.coordinator == nil {
		return []string{"basic_build", "test_coverage", "module_ops"}, nil
	}

	// Get recommendations from coordinator
	targets, strategies, err := e.coordinator.GetRecommendations(ctx)
	if err != nil {
		return nil, err
	}

	scenarios := make([]string, 0)

	// Convert targets and strategies to scenarios
	for _, target := range targets {
		if strings.Contains(target.Package, "testing") {
			scenarios = append(scenarios, "test_coverage")
		} else if strings.Contains(target.Package, "build") {
			scenarios = append(scenarios, "basic_build")
		} else if strings.Contains(target.Package, "mod") {
			scenarios = append(scenarios, "module_ops")
		}
	}

	for _, strategy := range strategies {
		switch strategy.Mode {
		case fuzz.GuidanceCoverage:
			scenarios = append(scenarios, "test_coverage")
		case fuzz.GuidanceSemantic:
			scenarios = append(scenarios, "integration")
		case fuzz.GuidanceLLM:
			scenarios = append(scenarios, "error_handling")
		}
	}

	// Remove duplicates and limit
	unique := e.removeDuplicateStrings(scenarios)
	if len(unique) > 5 {
		unique = unique[:5]
	}

	return unique, nil
}

// Private methods

func (e *GrammarEngine) initializeBuiltinGenerators() {
	// Go Tool Generator
	goToolGen := &GoToolGenerator{
		name:         "go_tools",
		capabilities: []string{"build", "test", "mod", "run", "install"},
		toolChains: map[string][]string{
			"build_chain":   {"go", "mod", "tidy", "go", "build"},
			"test_chain":    {"go", "mod", "tidy", "go", "test"},
			"install_chain": {"go", "mod", "tidy", "go", "install"},
		},
	}

	// ScriptTest Generator
	scriptTestGen := &ScriptTestGenerator{
		name:         "scripttest",
		capabilities: []string{"commands", "env", "files", "output"},
		commands: map[string]*CommandTemplate{
			"go_build": {
				Command:    "go",
				Args:       []string{"build"},
				Flags:      map[string]string{"-v": "", "-x": ""},
				Context:    "build",
				Complexity: 0.3,
			},
			"go_test": {
				Command:    "go",
				Args:       []string{"test"},
				Flags:      map[string]string{"-v": "", "-cover": "", "-race": ""},
				Context:    "test",
				Complexity: 0.5,
			},
		},
	}

	e.generators["go_tools"] = goToolGen
	e.generators["scripttest"] = scriptTestGen
}

func (e *GrammarEngine) initializeDefaultRules() {
	// Go build sequence rule
	buildRule := &GrammarRule{
		Name:    "go_build_sequence",
		Pattern: "env GOOS=${OS} GOARCH=${ARCH}\n${BUILD_CMD}\n${VALIDATION}",
		Alternatives: []string{
			"go mod tidy\ngo build -v .",
			"go build -o ${OUTPUT} ${SOURCE}",
			"go build -ldflags='-s -w' .",
		},
		Weight:  0.8,
		Context: "build",
		Examples: []string{
			"env GOOS=linux GOARCH=amd64\ngo build -v .\ntest -f main",
		},
	}

	// Coverage test sequence rule
	coverageRule := &GrammarRule{
		Name:    "coverage_test_sequence",
		Pattern: "go mod tidy\ngo test -cover -v ${PACKAGES}\n${COVERAGE_CHECK}",
		Alternatives: []string{
			"go test -coverprofile=coverage.out -v ./...\ngo tool cover -html=coverage.out",
			"go test -race -cover -v .\ngrep -E 'coverage: [0-9]+' output.txt",
		},
		Weight:  0.9,
		Context: "coverage",
		Examples: []string{
			"go test -cover -v ./...\ngrep 'coverage:' stdout",
		},
	}

	// Module operation rule
	moduleRule := &GrammarRule{
		Name:    "module_operation_sequence",
		Pattern: "go mod init ${MODULE_NAME}\n${MOD_OPERATIONS}\ngo mod tidy",
		Alternatives: []string{
			"go mod init example.com/test\ngo get -u ./...\ngo mod tidy",
			"go mod download\ngo mod verify\ngo list -m all",
		},
		Weight:  0.7,
		Context: "module",
	}

	// Error handling sequence
	errorRule := &GrammarRule{
		Name:    "error_handling_sequence",
		Pattern: "${ERROR_CMD}\n! ${SUCCESS_CMD}\n${ERROR_CHECK}",
		Alternatives: []string{
			"go build ./nonexistent\n! stdout 'success'\nstderr 'cannot find'",
			"go test ./missing\nstderr 'no Go files'\n! stdout 'PASS'",
		},
		Weight:  0.6,
		Context: "error",
	}

	e.rules["go_build_sequence"] = buildRule
	e.rules["coverage_test_sequence"] = coverageRule
	e.rules["module_operation_sequence"] = moduleRule
	e.rules["error_handling_sequence"] = errorRule
}

func (e *GrammarEngine) generateWithRule(ctx context.Context, rule *GrammarRule, depth int) (string, []*GrammarRule, error) {
	if depth > e.maxDepth {
		return "", nil, fmt.Errorf("maximum depth exceeded")
	}

	usedRules := []*GrammarRule{rule}

	// Choose between pattern and alternatives
	var template string
	if len(rule.Alternatives) > 0 && rand.Float64() < e.creativityLevel {
		template = rule.Alternatives[rand.Intn(len(rule.Alternatives))]
	} else {
		template = rule.Pattern
	}

	// Find and replace placeholders
	content := template
	placeholders := e.findPlaceholders(template)

	for _, placeholder := range placeholders {
		replacement, subRules, err := e.generatePlaceholder(ctx, placeholder, depth+1)
		if err != nil {
			// Use fallback if generation fails
			replacement = e.getFallbackForPlaceholder(placeholder)
		} else {
			usedRules = append(usedRules, subRules...)
		}

		content = strings.ReplaceAll(content, "${"+placeholder+"}", replacement)
	}

	return content, usedRules, nil
}

func (e *GrammarEngine) generatePlaceholder(ctx context.Context, placeholder string, depth int) (string, []*GrammarRule, error) {
	// Check if we have a specific rule for this placeholder
	if rule, exists := e.rules[placeholder]; exists {
		return e.generateWithRule(ctx, rule, depth)
	}

	// Try generators
	for _, gen := range e.generators {
		if e.canGeneratePlaceholder(gen, placeholder) {
			if mockRule := e.createMockRule(placeholder); mockRule != nil {
				content, err := gen.Generate(ctx, mockRule, depth)
				if err == nil {
					return content, []*GrammarRule{mockRule}, nil
				}
			}
		}
	}

	// Use semantic generation if enabled
	if e.useSemanticGuide {
		return e.generateSemanticPlaceholder(placeholder), nil, nil
	}

	return e.getFallbackForPlaceholder(placeholder), nil, nil
}

func (e *GrammarEngine) findPlaceholders(template string) []string {
	var placeholders []string
	parts := strings.Split(template, "${")

	for i := 1; i < len(parts); i++ {
		if idx := strings.Index(parts[i], "}"); idx != -1 {
			placeholder := parts[i][:idx]
			placeholders = append(placeholders, placeholder)
		}
	}

	return e.removeDuplicateStrings(placeholders)
}

func (e *GrammarEngine) canGeneratePlaceholder(gen Generator, placeholder string) bool {
	capabilities := gen.GetCapabilities()
	lowerPlaceholder := strings.ToLower(placeholder)

	for _, cap := range capabilities {
		if strings.Contains(lowerPlaceholder, cap) {
			return true
		}
	}

	return false
}

func (e *GrammarEngine) createMockRule(placeholder string) *GrammarRule {
	return &GrammarRule{
		Name:    placeholder,
		Pattern: placeholder,
		Weight:  0.5,
		Context: "generated",
	}
}

func (e *GrammarEngine) generateSemanticPlaceholder(placeholder string) string {
	// Semantic generation based on placeholder meaning
	semanticMap := map[string]string{
		"OS":             "linux",
		"ARCH":           "amd64",
		"MODULE_NAME":    "example.com/test",
		"PACKAGES":       "./...",
		"OUTPUT":         "main",
		"SOURCE":         ".",
		"BUILD_CMD":      "go build -v .",
		"VALIDATION":     "test -f main",
		"COVERAGE_CHECK": "grep 'coverage:' stdout",
		"MOD_OPERATIONS": "go get -u ./...",
		"ERROR_CMD":      "go build ./nonexistent",
		"SUCCESS_CMD":    "echo 'success'",
		"ERROR_CHECK":    "stderr 'cannot find'",
	}

	if value, exists := semanticMap[placeholder]; exists {
		return value
	}

	// Generate based on context
	return e.generateContextualPlaceholder(placeholder)
}

func (e *GrammarEngine) generateContextualPlaceholder(placeholder string) string {
	lower := strings.ToLower(placeholder)

	if strings.Contains(lower, "file") {
		return "main.go"
	} else if strings.Contains(lower, "dir") {
		return "./test"
	} else if strings.Contains(lower, "cmd") {
		return "go build"
	} else if strings.Contains(lower, "flag") {
		return "-v"
	} else if strings.Contains(lower, "name") {
		return "test"
	}

	return "placeholder_" + strings.ToLower(placeholder)
}

func (e *GrammarEngine) getFallbackForPlaceholder(placeholder string) string {
	fallbacks := map[string]string{
		"OS":          "linux",
		"ARCH":        "amd64",
		"MODULE_NAME": "example.com/test",
		"PACKAGES":    "./...",
		"OUTPUT":      "main",
		"SOURCE":      ".",
	}

	if fallback, exists := fallbacks[placeholder]; exists {
		return fallback
	}

	return "test_value"
}

func (e *GrammarEngine) evaluateQuality(ctx context.Context, content string) (float64, error) {
	if e.coverageOracle == nil {
		return e.heuristicQualityEvaluation(content), nil
	}

	// Use LLM oracle for quality evaluation
	assessment, err := e.coverageOracle.EvaluateTestQuality(ctx, nil, content)
	if err != nil {
		return e.heuristicQualityEvaluation(content), nil
	}

	return assessment.Score, nil
}

func (e *GrammarEngine) heuristicQualityEvaluation(content string) float64 {
	score := 0.5 // Base score

	// Check content length
	if len(content) > 20 && len(content) < 1000 {
		score += 0.1
	}

	// Check for Go commands
	if strings.Contains(content, "go ") {
		score += 0.2
	}

	// Check for test patterns
	if strings.Contains(content, "test") {
		score += 0.1
	}

	// Check for validation
	if strings.Contains(content, "stdout") || strings.Contains(content, "stderr") {
		score += 0.1
	}

	// Check for error handling
	if strings.Contains(content, "!") || strings.Contains(content, "stderr") {
		score += 0.1
	}

	return score
}

func (e *GrammarEngine) estimateCoverageImpact(content string, rules []*GrammarRule) float64 {
	impact := 0.0

	// Base impact from content analysis
	if strings.Contains(content, "go build") {
		impact += 0.3
	}
	if strings.Contains(content, "go test") {
		impact += 0.4
	}
	if strings.Contains(content, "-cover") {
		impact += 0.2
	}
	if strings.Contains(content, "-race") {
		impact += 0.1
	}

	// Impact from rule complexity
	for _, rule := range rules {
		impact += rule.Weight * 0.1
	}

	return impact
}

func (e *GrammarEngine) detectPatterns(content string, rules []*GrammarRule) []*Pattern {
	patterns := make([]*Pattern, 0)

	// Detect command patterns
	if strings.Contains(content, "go mod tidy") && strings.Contains(content, "go build") {
		pattern := &Pattern{
			ID:           "tidy_build_pattern",
			Name:         "Tidy Before Build",
			Template:     "go mod tidy\ngo build ${ARGS}",
			Frequency:    1,
			SuccessRate:  0.8,
			QualityScore: 0.7,
			LastUsed:     time.Now(),
			Context:      []string{"build", "module"},
		}
		patterns = append(patterns, pattern)
	}

	// Detect test patterns
	if strings.Contains(content, "go test") && strings.Contains(content, "-cover") {
		pattern := &Pattern{
			ID:           "coverage_test_pattern",
			Name:         "Coverage Test",
			Template:     "go test -cover ${FLAGS} ${PACKAGES}",
			Frequency:    1,
			SuccessRate:  0.9,
			QualityScore: 0.8,
			LastUsed:     time.Now(),
			Context:      []string{"test", "coverage"},
		}
		patterns = append(patterns, pattern)
	}

	return patterns
}

func (e *GrammarEngine) applyGenerationOptions(options map[string]interface{}) {
	if options == nil {
		return
	}

	if creativity, ok := options["creativity"]; ok {
		if level, ok := creativity.(float64); ok {
			e.creativityLevel = level
		}
	}

	if depth, ok := options["max_depth"]; ok {
		if maxDepth, ok := depth.(int); ok {
			e.maxDepth = maxDepth
		}
	}

	if semantic, ok := options["use_semantic"]; ok {
		if useSemantic, ok := semantic.(bool); ok {
			e.useSemanticGuide = useSemantic
		}
	}
}

func (e *GrammarEngine) calculateMaxDepthUsed(rules []*GrammarRule) int {
	// Simple approximation - in practice would track actual depth
	return len(rules)
}

func (e *GrammarEngine) generateResultID() string {
	return fmt.Sprintf("gen_%d_%d", time.Now().UnixNano(), rand.Intn(1000))
}

func (e *GrammarEngine) removeDuplicateStrings(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// GetMetrics returns engine metrics
func (e *GrammarEngine) GetMetrics() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return map[string]interface{}{
		"total_generations":      e.totalGenerations,
		"successful_generations": e.successfulGens,
		"average_quality":        e.averageQuality,
		"rules_count":            len(e.rules),
		"generators_count":       len(e.generators),
		"active_patterns":        len(e.activePatterns),
		"creativity_level":       e.creativityLevel,
	}
}

// Generator implementations

func (g *GoToolGenerator) Generate(ctx context.Context, rule *GrammarRule, depth int) (string, error) {
	if strings.Contains(rule.Context, "build") {
		return "go build -v .", nil
	} else if strings.Contains(rule.Context, "test") {
		return "go test -v ./...", nil
	} else if strings.Contains(rule.Context, "mod") {
		return "go mod tidy", nil
	}

	return "go version", nil
}

func (g *GoToolGenerator) GetName() string {
	return g.name
}

func (g *GoToolGenerator) GetCapabilities() []string {
	return g.capabilities
}

func (g *ScriptTestGenerator) Generate(ctx context.Context, rule *GrammarRule, depth int) (string, error) {
	if template, exists := g.commands[rule.Context]; exists {
		cmd := template.Command
		if len(template.Args) > 0 {
			cmd += " " + strings.Join(template.Args, " ")
		}
		return cmd, nil
	}

	return "echo 'test'", nil
}

func (g *ScriptTestGenerator) GetName() string {
	return g.name
}

func (g *ScriptTestGenerator) GetCapabilities() []string {
	return g.capabilities
}

// Stop shuts down the engine
func (e *GrammarEngine) Stop() {
	e.cancel()
}
