package coverageviz

import (
	"time"
)

// CoverageVisualization represents the complete coverage and test data
type CoverageVisualization struct {
	Sessions  []TestSession         `json:"sessions"`
	Files     map[string]*FileData  `json:"files"`
	Summary   Summary              `json:"summary"`
	Generated time.Time            `json:"generated"`
}

// TestSession represents a single test execution session
type TestSession struct {
	ID        string               `json:"id"`
	Name      string               `json:"name"`
	StartTime time.Time            `json:"startTime"`
	EndTime   time.Time            `json:"endTime"`
	Tests     []TestExecution      `json:"tests"`
	Traces    []MCPTrace           `json:"traces,omitempty"`
	Coverage  map[string]FileCoverage `json:"coverage"`
	Transport string               `json:"transport,omitempty"`
}

// TestExecution represents an individual test run
type TestExecution struct {
	Package   string               `json:"package"`
	TestName  string               `json:"testName"`
	StartTime time.Time            `json:"startTime"`
	Duration  time.Duration        `json:"duration"`
	Result    TestResult           `json:"result"`
	Output    string               `json:"output,omitempty"`
	Error     string               `json:"error,omitempty"`
	Coverage  map[string]FileCoverage `json:"coverage"`
	Traces    []MCPTrace           `json:"traces,omitempty"`
}

// MCPTrace represents parsed MCP trace data
type MCPTrace struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Direction string                 `json:"direction"`
	Timestamp time.Time              `json:"timestamp"`
	Method    string                 `json:"method,omitempty"`
	Params    map[string]interface{} `json:"params,omitempty"`
	Result    interface{}            `json:"result,omitempty"`
	Error     interface{}            `json:"error,omitempty"`
}

// FileData represents source file information and coverage
type FileData struct {
	Path         string             `json:"path"`
	Package      string             `json:"package"`
	Lines        []LineData         `json:"lines"`
	Functions    []FunctionData     `json:"functions"`
	Coverage     FileCoverage       `json:"coverage"`
	TestImpact   []TestImpact       `json:"testImpact"`
}

// LineData represents a single line in a source file
type LineData struct {
	Number     int                `json:"number"`
	Content    string             `json:"content"`
	Covered    bool               `json:"covered"`
	HitCount   int                `json:"hitCount"`
	Tests      []string           `json:"tests"`
	Conditions []ConditionCoverage `json:"conditions,omitempty"`
}

// FunctionData represents a function or method in the source
type FunctionData struct {
	Name       string             `json:"name"`
	Package    string             `json:"package"`
	StartLine  int                `json:"startLine"`
	EndLine    int                `json:"endLine"`
	Covered    bool               `json:"covered"`
	Tests      []string           `json:"tests"`
	Complexity int                `json:"complexity,omitempty"`
}

// FileCoverage represents coverage statistics for a file
type FileCoverage struct {
	Path            string  `json:"path"`
	TotalLines      int     `json:"totalLines"`
	CoveredLines    int     `json:"coveredLines"`
	TotalBranches   int     `json:"totalBranches"`
	CoveredBranches int     `json:"coveredBranches"`
	CoveragePercent float64 `json:"coveragePercent"`
}

// ConditionCoverage represents branch coverage information
type ConditionCoverage struct {
	Type       string `json:"type"`
	Expression string `json:"expression"`
	Covered    int    `json:"covered"`
	Total      int    `json:"total"`
}

// TestImpact shows which tests cover specific code
type TestImpact struct {
	TestID       string `json:"testId"`
	TestName     string `json:"testName"`
	CoveredLines []int  `json:"coveredLines"`
	Impact       float64 `json:"impact"` // Percentage of file covered by this test
}

// Summary provides overall coverage statistics
type Summary struct {
	TotalTests      int                   `json:"totalTests"`
	PassedTests     int                   `json:"passedTests"`
	FailedTests     int                   `json:"failedTests"`
	TotalFiles      int                   `json:"totalFiles"`
	CoveredFiles    int                   `json:"coveredFiles"`
	TotalLines      int                   `json:"totalLines"`
	CoveredLines    int                   `json:"coveredLines"`
	TotalBranches   int                   `json:"totalBranches"`
	CoveredBranches int                   `json:"coveredBranches"`
	Coverage        CoverageStats         `json:"coverage"`
	ByPackage       map[string]CoverageStats `json:"byPackage"`
}

// CoverageStats represents coverage percentages
type CoverageStats struct {
	Line   float64 `json:"line"`
	Branch float64 `json:"branch"`
	Function float64 `json:"function"`
}

// TestResult represents the outcome of a test
type TestResult string

const (
	TestPassed  TestResult = "passed"
	TestFailed  TestResult = "failed"
	TestSkipped TestResult = "skipped"
	TestError   TestResult = "error"
)