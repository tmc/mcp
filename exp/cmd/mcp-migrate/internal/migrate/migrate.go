// Package migrate provides MCP migration and upgrade functionality
package migrate

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Config represents migration configuration
type Config struct {
	From        string
	To          string
	Language    string
	Path        string
	ConfigFile  string
	Backup      bool
	Verbose     bool
	DryRun      bool
	Interactive bool
}

// Migrator provides migration functionality
type Migrator struct {
	config       *Config
	analyzers    map[string]Analyzer
	transformers map[string]Transformer
	validators   map[string]Validator
	fileSet      *token.FileSet
}

// Analyzer analyzes code for migration opportunities
type Analyzer interface {
	Analyze(ctx context.Context, path string) (*AnalysisResult, error)
}

// Transformer transforms code between versions
type Transformer interface {
	Transform(ctx context.Context, files []string) (*TransformResult, error)
}

// Validator validates migration results
type Validator interface {
	Validate(ctx context.Context, result *TransformResult) error
}

// AnalysisResult contains analysis results
type AnalysisResult struct {
	Language        string                 `json:"language"`
	CurrentVersion  string                 `json:"current_version"`
	TargetVersion   string                 `json:"target_version"`
	Issues          []Issue                `json:"issues"`
	Opportunities   []Opportunity          `json:"opportunities"`
	Dependencies    []Dependency           `json:"dependencies"`
	Compatibility   CompatibilityReport    `json:"compatibility"`
	EstimatedEffort string                 `json:"estimated_effort"`
	Recommendations []Recommendation       `json:"recommendations"`
}

// Issue represents a migration issue
type Issue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	Column      int    `json:"column"`
	Message     string `json:"message"`
	Description string `json:"description"`
	Fix         string `json:"fix,omitempty"`
}

// Opportunity represents a migration opportunity
type Opportunity struct {
	Type        string `json:"type"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	Description string `json:"description"`
	Benefit     string `json:"benefit"`
	Effort      string `json:"effort"`
}

// Dependency represents a project dependency
type Dependency struct {
	Name           string `json:"name"`
	CurrentVersion string `json:"current_version"`
	TargetVersion  string `json:"target_version"`
	Compatible     bool   `json:"compatible"`
	UpdateRequired bool   `json:"update_required"`
}

// CompatibilityReport provides compatibility information
type CompatibilityReport struct {
	Compatible    bool     `json:"compatible"`
	BreakingChanges []string `json:"breaking_changes"`
	Warnings      []string `json:"warnings"`
	Notes         []string `json:"notes"`
}

// Recommendation provides migration recommendations
type Recommendation struct {
	Type        string `json:"type"`
	Priority    string `json:"priority"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Reason      string `json:"reason"`
}

// TransformResult contains transformation results
type TransformResult struct {
	Files       []TransformedFile `json:"files"`
	Changes     []Change          `json:"changes"`
	Warnings    []string          `json:"warnings"`
	Errors      []string          `json:"errors"`
	Summary     TransformSummary  `json:"summary"`
}

// TransformedFile represents a transformed file
type TransformedFile struct {
	Path         string `json:"path"`
	OriginalSize int64  `json:"original_size"`
	NewSize      int64  `json:"new_size"`
	Changes      int    `json:"changes"`
	Backed       bool   `json:"backed_up"`
}

// Change represents a code change
type Change struct {
	Type        string `json:"type"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	Column      int    `json:"column"`
	Before      string `json:"before"`
	After       string `json:"after"`
	Description string `json:"description"`
}

// TransformSummary provides transformation summary
type TransformSummary struct {
	FilesProcessed int    `json:"files_processed"`
	FilesChanged   int    `json:"files_changed"`
	TotalChanges   int    `json:"total_changes"`
	Errors         int    `json:"errors"`
	Warnings       int    `json:"warnings"`
	Duration       string `json:"duration"`
}

// MigrationPlan represents a migration plan
type MigrationPlan struct {
	From        string       `json:"from"`
	To          string       `json:"to"`
	Language    string       `json:"language"`
	Path        string       `json:"path"`
	Created     time.Time    `json:"created"`
	Steps       []Step       `json:"steps"`
	Estimate    string       `json:"estimate"`
	Risks       []Risk       `json:"risks"`
	Rollback    RollbackPlan `json:"rollback"`
}

// Step represents a migration step
type Step struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	Order       int      `json:"order"`
	Required    bool     `json:"required"`
	Files       []string `json:"files"`
	Commands    []string `json:"commands"`
	Validation  string   `json:"validation"`
}

// Risk represents a migration risk
type Risk struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Mitigation  string `json:"mitigation"`
	Probability string `json:"probability"`
}

// RollbackPlan represents a rollback plan
type RollbackPlan struct {
	Enabled bool     `json:"enabled"`
	Steps   []Step   `json:"steps"`
	Backups []string `json:"backups"`
}

// Version represents a protocol version
type Version struct {
	Major    int    `json:"major"`
	Minor    int    `json:"minor"`
	Patch    int    `json:"patch"`
	Date     string `json:"date"`
	Features []string `json:"features"`
	Breaking []string `json:"breaking"`
}

// New creates a new migrator
func New(config *Config) (*Migrator, error) {
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	migrator := &Migrator{
		config:       config,
		analyzers:    make(map[string]Analyzer),
		transformers: make(map[string]Transformer),
		validators:   make(map[string]Validator),
		fileSet:      token.NewFileSet(),
	}

	// Register language-specific components
	migrator.registerAnalyzers()
	migrator.registerTransformers()
	migrator.registerValidators()

	return migrator, nil
}

// Analyze analyzes project for migration opportunities
func (m *Migrator) Analyze(ctx context.Context) error {
	if m.config.Verbose {
		fmt.Printf("Analyzing project at %s\n", m.config.Path)
	}

	// Detect language if not specified
	if m.config.Language == "" {
		lang, err := m.detectLanguage(m.config.Path)
		if err != nil {
			return fmt.Errorf("failed to detect language: %w", err)
		}
		m.config.Language = lang
	}

	// Get analyzer for language
	analyzer, exists := m.analyzers[m.config.Language]
	if !exists {
		return fmt.Errorf("no analyzer for language: %s", m.config.Language)
	}

	// Perform analysis
	result, err := analyzer.Analyze(ctx, m.config.Path)
	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	// Output results
	return m.outputAnalysisResult(result)
}

// Upgrade upgrades MCP protocol version
func (m *Migrator) Upgrade(ctx context.Context) error {
	if m.config.Verbose {
		fmt.Printf("Upgrading from %s to %s\n", m.config.From, m.config.To)
	}

	// First analyze the project
	if err := m.Analyze(ctx); err != nil {
		return fmt.Errorf("pre-upgrade analysis failed: %w", err)
	}

	// Create backup if requested
	if m.config.Backup {
		if err := m.createBackup(); err != nil {
			return fmt.Errorf("backup creation failed: %w", err)
		}
	}

	// Get files to transform
	files, err := m.getProjectFiles()
	if err != nil {
		return fmt.Errorf("failed to get project files: %w", err)
	}

	// Get transformer for language
	transformer, exists := m.transformers[m.config.Language]
	if !exists {
		return fmt.Errorf("no transformer for language: %s", m.config.Language)
	}

	// Perform transformation
	result, err := transformer.Transform(ctx, files)
	if err != nil {
		return fmt.Errorf("transformation failed: %w", err)
	}

	// Validate results
	if err := m.validateTransform(ctx, result); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Output results
	return m.outputTransformResult(result)
}

// Transform transforms code between versions
func (m *Migrator) Transform(ctx context.Context, files []string) error {
	if m.config.Verbose {
		fmt.Printf("Transforming %d files\n", len(files))
	}

	// Get transformer for language
	transformer, exists := m.transformers[m.config.Language]
	if !exists {
		return fmt.Errorf("no transformer for language: %s", m.config.Language)
	}

	// Perform transformation
	result, err := transformer.Transform(ctx, files)
	if err != nil {
		return fmt.Errorf("transformation failed: %w", err)
	}

	// Output results
	return m.outputTransformResult(result)
}

// Validate validates migration results
func (m *Migrator) Validate(ctx context.Context) error {
	if m.config.Verbose {
		fmt.Println("Validating migration results")
	}

	// Get validator for language
	validator, exists := m.validators[m.config.Language]
	if !exists {
		return fmt.Errorf("no validator for language: %s", m.config.Language)
	}

	// Create dummy result for validation
	result := &TransformResult{
		Summary: TransformSummary{
			FilesProcessed: 0,
			FilesChanged:   0,
			TotalChanges:   0,
		},
	}

	// Perform validation
	if err := validator.Validate(ctx, result); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	fmt.Println("Validation successful")
	return nil
}

// CreatePlan creates a migration plan
func (m *Migrator) CreatePlan(ctx context.Context) error {
	if m.config.Verbose {
		fmt.Println("Creating migration plan")
	}

	plan := &MigrationPlan{
		From:     m.config.From,
		To:       m.config.To,
		Language: m.config.Language,
		Path:     m.config.Path,
		Created:  time.Now(),
		Steps:    m.createMigrationSteps(),
		Estimate: m.estimateEffort(),
		Risks:    m.assessRisks(),
		Rollback: m.createRollbackPlan(),
	}

	// Save plan to file
	planPath := filepath.Join(m.config.Path, "migration-plan.json")
	if err := m.savePlan(plan, planPath); err != nil {
		return fmt.Errorf("failed to save plan: %w", err)
	}

	fmt.Printf("Migration plan created: %s\n", planPath)
	return nil
}

// ApplyPlan applies a migration plan
func (m *Migrator) ApplyPlan(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("plan file path required")
	}

	planPath := args[0]
	if m.config.Verbose {
		fmt.Printf("Applying migration plan: %s\n", planPath)
	}

	// Load plan
	plan, err := m.loadPlan(planPath)
	if err != nil {
		return fmt.Errorf("failed to load plan: %w", err)
	}

	// Execute steps
	for _, step := range plan.Steps {
		if err := m.executeStep(ctx, step); err != nil {
			return fmt.Errorf("failed to execute step %s: %w", step.Name, err)
		}
	}

	fmt.Println("Migration plan applied successfully")
	return nil
}

// Helper methods

func (m *Migrator) registerAnalyzers() {
	m.analyzers["go"] = &GoAnalyzer{fileSet: m.fileSet}
	m.analyzers["typescript"] = &TypeScriptAnalyzer{}
	m.analyzers["python"] = &PythonAnalyzer{}
	m.analyzers["rust"] = &RustAnalyzer{}
	m.analyzers["java"] = &JavaAnalyzer{}
}

func (m *Migrator) registerTransformers() {
	m.transformers["go"] = &GoTransformer{fileSet: m.fileSet}
	m.transformers["typescript"] = &TypeScriptTransformer{}
	m.transformers["python"] = &PythonTransformer{}
	m.transformers["rust"] = &RustTransformer{}
	m.transformers["java"] = &JavaTransformer{}
}

func (m *Migrator) registerValidators() {
	m.validators["go"] = &GoValidator{}
	m.validators["typescript"] = &TypeScriptValidator{}
	m.validators["python"] = &PythonValidator{}
	m.validators["rust"] = &RustValidator{}
	m.validators["java"] = &JavaValidator{}
}

func (m *Migrator) detectLanguage(path string) (string, error) {
	// Check for language-specific files
	if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
		return "go", nil
	}
	if _, err := os.Stat(filepath.Join(path, "package.json")); err == nil {
		return "typescript", nil
	}
	if _, err := os.Stat(filepath.Join(path, "pyproject.toml")); err == nil {
		return "python", nil
	}
	if _, err := os.Stat(filepath.Join(path, "Cargo.toml")); err == nil {
		return "rust", nil
	}
	if _, err := os.Stat(filepath.Join(path, "pom.xml")); err == nil {
		return "java", nil
	}

	return "", fmt.Errorf("unable to detect language")
}

func (m *Migrator) createBackup() error {
	backupPath := m.config.Path + ".backup." + time.Now().Format("20060102-150405")
	if m.config.Verbose {
		fmt.Printf("Creating backup at %s\n", backupPath)
	}

	if m.config.DryRun {
		fmt.Printf("Would create backup: %s\n", backupPath)
		return nil
	}

	// TODO: Implement backup creation
	return nil
}

func (m *Migrator) getProjectFiles() ([]string, error) {
	var files []string

	err := filepath.Walk(m.config.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && m.isSourceFile(path) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

func (m *Migrator) isSourceFile(path string) bool {
	switch m.config.Language {
	case "go":
		return strings.HasSuffix(path, ".go")
	case "typescript":
		return strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".tsx")
	case "python":
		return strings.HasSuffix(path, ".py")
	case "rust":
		return strings.HasSuffix(path, ".rs")
	case "java":
		return strings.HasSuffix(path, ".java")
	}
	return false
}

func (m *Migrator) validateTransform(ctx context.Context, result *TransformResult) error {
	validator, exists := m.validators[m.config.Language]
	if !exists {
		return fmt.Errorf("no validator for language: %s", m.config.Language)
	}

	return validator.Validate(ctx, result)
}

func (m *Migrator) outputAnalysisResult(result *AnalysisResult) error {
	if m.config.DryRun {
		fmt.Println("Analysis Results (dry run):")
	} else {
		fmt.Println("Analysis Results:")
	}

	fmt.Printf("  Language: %s\n", result.Language)
	fmt.Printf("  Current Version: %s\n", result.CurrentVersion)
	fmt.Printf("  Target Version: %s\n", result.TargetVersion)
	fmt.Printf("  Issues: %d\n", len(result.Issues))
	fmt.Printf("  Opportunities: %d\n", len(result.Opportunities))
	fmt.Printf("  Estimated Effort: %s\n", result.EstimatedEffort)

	// Output issues
	if len(result.Issues) > 0 {
		fmt.Println("\nIssues:")
		for _, issue := range result.Issues {
			fmt.Printf("  - %s: %s (%s:%d)\n", issue.Type, issue.Message, issue.File, issue.Line)
		}
	}

	// Output opportunities
	if len(result.Opportunities) > 0 {
		fmt.Println("\nOpportunities:")
		for _, opp := range result.Opportunities {
			fmt.Printf("  - %s: %s (%s)\n", opp.Type, opp.Description, opp.Benefit)
		}
	}

	return nil
}

func (m *Migrator) outputTransformResult(result *TransformResult) error {
	if m.config.DryRun {
		fmt.Println("Transform Results (dry run):")
	} else {
		fmt.Println("Transform Results:")
	}

	fmt.Printf("  Files Processed: %d\n", result.Summary.FilesProcessed)
	fmt.Printf("  Files Changed: %d\n", result.Summary.FilesChanged)
	fmt.Printf("  Total Changes: %d\n", result.Summary.TotalChanges)
	fmt.Printf("  Errors: %d\n", result.Summary.Errors)
	fmt.Printf("  Warnings: %d\n", result.Summary.Warnings)
	fmt.Printf("  Duration: %s\n", result.Summary.Duration)

	return nil
}

func (m *Migrator) createMigrationSteps() []Step {
	// Create basic migration steps
	return []Step{
		{
			ID:          "backup",
			Name:        "Create Backup",
			Description: "Create backup of current project",
			Type:        "backup",
			Order:       1,
			Required:    true,
		},
		{
			ID:          "analyze",
			Name:        "Analyze Project",
			Description: "Analyze project for migration issues",
			Type:        "analysis",
			Order:       2,
			Required:    true,
		},
		{
			ID:          "transform",
			Name:        "Transform Code",
			Description: "Transform code to target version",
			Type:        "transform",
			Order:       3,
			Required:    true,
		},
		{
			ID:          "validate",
			Name:        "Validate Results",
			Description: "Validate migration results",
			Type:        "validation",
			Order:       4,
			Required:    true,
		},
	}
}

func (m *Migrator) estimateEffort() string {
	// Simple effort estimation
	return "Medium (4-8 hours)"
}

func (m *Migrator) assessRisks() []Risk {
	return []Risk{
		{
			Type:        "compatibility",
			Severity:    "medium",
			Description: "Potential compatibility issues with dependencies",
			Mitigation:  "Test thoroughly and update dependencies",
			Probability: "medium",
		},
	}
}

func (m *Migrator) createRollbackPlan() RollbackPlan {
	return RollbackPlan{
		Enabled: true,
		Steps: []Step{
			{
				ID:          "restore",
				Name:        "Restore Backup",
				Description: "Restore from backup",
				Type:        "restore",
				Order:       1,
				Required:    true,
			},
		},
	}
}

func (m *Migrator) savePlan(plan *MigrationPlan, path string) error {
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, data, 0644)
}

func (m *Migrator) loadPlan(path string) (*MigrationPlan, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var plan MigrationPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, err
	}

	return &plan, nil
}

func (m *Migrator) executeStep(ctx context.Context, step Step) error {
	if m.config.Verbose {
		fmt.Printf("Executing step: %s\n", step.Name)
	}

	// Execute commands
	for _, cmd := range step.Commands {
		if m.config.Verbose {
			fmt.Printf("  Running: %s\n", cmd)
		}
		// TODO: Execute command
	}

	return nil
}

func validateConfig(config *Config) error {
	if config.Path == "" {
		return fmt.Errorf("path is required")
	}

	if config.From == "" || config.To == "" {
		return fmt.Errorf("source and target versions are required")
	}

	return nil
}

// Language-specific analyzers, transformers, and validators

// GoAnalyzer analyzes Go projects
type GoAnalyzer struct {
	fileSet *token.FileSet
}

func (a *GoAnalyzer) Analyze(ctx context.Context, path string) (*AnalysisResult, error) {
	result := &AnalysisResult{
		Language:       "go",
		CurrentVersion: "unknown",
		TargetVersion:  "unknown",
		Issues:         []Issue{},
		Opportunities:  []Opportunity{},
	}

	// Parse Go files
	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(filePath, ".go") {
			return nil
		}

		// Parse file
		src, err := ioutil.ReadFile(filePath)
		if err != nil {
			return err
		}

		file, err := parser.ParseFile(a.fileSet, filePath, src, parser.ParseComments)
		if err != nil {
			result.Issues = append(result.Issues, Issue{
				Type:     "parse_error",
				Severity: "high",
				File:     filePath,
				Message:  err.Error(),
			})
			return nil
		}

		// Analyze AST
		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.CallExpr:
				if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
					if id, ok := sel.X.(*ast.Ident); ok && id.Name == "mcp" {
						// Found MCP call - analyze for migration opportunities
						result.Opportunities = append(result.Opportunities, Opportunity{
							Type:        "api_call",
							File:        filePath,
							Line:        a.fileSet.Position(node.Pos()).Line,
							Description: fmt.Sprintf("MCP API call: %s", sel.Sel.Name),
							Benefit:     "Can be migrated to newer API",
							Effort:      "low",
						})
					}
				}
			}
			return true
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	result.EstimatedEffort = "Low (1-2 hours)"
	return result, nil
}

// GoTransformer transforms Go code
type GoTransformer struct {
	fileSet *token.FileSet
}

func (t *GoTransformer) Transform(ctx context.Context, files []string) (*TransformResult, error) {
	start := time.Now()
	
	result := &TransformResult{
		Files:    []TransformedFile{},
		Changes:  []Change{},
		Warnings: []string{},
		Errors:   []string{},
	}

	for _, file := range files {
		if !strings.HasSuffix(file, ".go") {
			continue
		}

		// Read file
		content, err := ioutil.ReadFile(file)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to read %s: %v", file, err))
			continue
		}

		// Simple string replacement transformations
		originalContent := string(content)
		newContent := originalContent

		// Example transformations
		transformations := map[string]string{
			"mcp.OldAPI":      "mcp.NewAPI",
			"mcp.CallTool":    "mcp.CallToolWithContext",
			"mcp.Initialize":  "mcp.InitializeWithConfig",
		}

		changes := 0
		for old, new := range transformations {
			if strings.Contains(newContent, old) {
				newContent = strings.ReplaceAll(newContent, old, new)
				changes++
				
				result.Changes = append(result.Changes, Change{
					Type:        "api_migration",
					File:        file,
					Before:      old,
					After:       new,
					Description: fmt.Sprintf("Migrated %s to %s", old, new),
				})
			}
		}

		if changes > 0 {
			result.Files = append(result.Files, TransformedFile{
				Path:         file,
				OriginalSize: int64(len(originalContent)),
				NewSize:      int64(len(newContent)),
				Changes:      changes,
			})

			// Write transformed content (if not dry run)
			// TODO: Implement actual file writing
		}
	}

	result.Summary = TransformSummary{
		FilesProcessed: len(files),
		FilesChanged:   len(result.Files),
		TotalChanges:   len(result.Changes),
		Duration:       time.Since(start).String(),
	}

	return result, nil
}

// GoValidator validates Go migration results
type GoValidator struct{}

func (v *GoValidator) Validate(ctx context.Context, result *TransformResult) error {
	// TODO: Implement Go validation
	return nil
}

// Placeholder implementations for other languages
type TypeScriptAnalyzer struct{}
func (a *TypeScriptAnalyzer) Analyze(ctx context.Context, path string) (*AnalysisResult, error) {
	return &AnalysisResult{Language: "typescript"}, nil
}

type TypeScriptTransformer struct{}
func (t *TypeScriptTransformer) Transform(ctx context.Context, files []string) (*TransformResult, error) {
	return &TransformResult{}, nil
}

type TypeScriptValidator struct{}
func (v *TypeScriptValidator) Validate(ctx context.Context, result *TransformResult) error {
	return nil
}

type PythonAnalyzer struct{}
func (a *PythonAnalyzer) Analyze(ctx context.Context, path string) (*AnalysisResult, error) {
	return &AnalysisResult{Language: "python"}, nil
}

type PythonTransformer struct{}
func (t *PythonTransformer) Transform(ctx context.Context, files []string) (*TransformResult, error) {
	return &TransformResult{}, nil
}

type PythonValidator struct{}
func (v *PythonValidator) Validate(ctx context.Context, result *TransformResult) error {
	return nil
}

type RustAnalyzer struct{}
func (a *RustAnalyzer) Analyze(ctx context.Context, path string) (*AnalysisResult, error) {
	return &AnalysisResult{Language: "rust"}, nil
}

type RustTransformer struct{}
func (t *RustTransformer) Transform(ctx context.Context, files []string) (*TransformResult, error) {
	return &TransformResult{}, nil
}

type RustValidator struct{}
func (v *RustValidator) Validate(ctx context.Context, result *TransformResult) error {
	return nil
}

type JavaAnalyzer struct{}
func (a *JavaAnalyzer) Analyze(ctx context.Context, path string) (*AnalysisResult, error) {
	return &AnalysisResult{Language: "java"}, nil
}

type JavaTransformer struct{}
func (t *JavaTransformer) Transform(ctx context.Context, files []string) (*TransformResult, error) {
	return &TransformResult{}, nil
}

type JavaValidator struct{}
func (v *JavaValidator) Validate(ctx context.Context, result *TransformResult) error {
	return nil
}