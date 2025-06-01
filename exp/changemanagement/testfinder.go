package changemanagement

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// TestFinderResult contains the results of finding affected tests
type TestFinderResult struct {
	DefinitelyAffected []string `json:"definitely_affected"`
	PossiblyAffected   []string `json:"possibly_affected"`
	RelatedTests       []string `json:"related_tests"`
	NewTestsNeeded     []string `json:"new_tests_needed"`
}

// TestFinder finds tests affected by changes
type TestFinder struct {
	codebase string
}

// NewTestFinder creates a new test finder
func NewTestFinder(codebase string) *TestFinder {
	return &TestFinder{
		codebase: codebase,
	}
}

// FindAffectedTests finds tests affected by a change
func (f *TestFinder) FindAffectedTests(analysis *AnalysisResult) (*TestFinderResult, error) {
	result := &TestFinderResult{
		DefinitelyAffected: []string{},
		PossiblyAffected:   []string{},
		RelatedTests:       []string{},
		NewTestsNeeded:     []string{},
	}

	// Walk the codebase to find test files
	err := filepath.Walk(f.codebase, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip non-test files
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Skip vendor directory
		if strings.Contains(path, "vendor/") {
			return nil
		}

		// Analyze test file
		affected, err := f.analyzeTestFile(path, analysis)
		if err != nil {
			// Log error but continue
			fmt.Printf("Error analyzing %s: %v\n", path, err)
			return nil
		}

		if affected {
			result.DefinitelyAffected = append(result.DefinitelyAffected, path)
		} else if f.isPossiblyAffected(path, analysis) {
			result.PossiblyAffected = append(result.PossiblyAffected, path)
		} else if f.isRelated(path, analysis) {
			result.RelatedTests = append(result.RelatedTests, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk codebase: %w", err)
	}

	// Determine new tests needed
	result.NewTestsNeeded = f.determineNewTests(analysis, result)

	return result, nil
}

func (f *TestFinder) analyzeTestFile(path string, analysis *AnalysisResult) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return false, err
	}

	// Simple keyword matching for now
	contentStr := string(content)
	
	// Check for affected components
	for _, component := range analysis.Components {
		if strings.Contains(strings.ToLower(contentStr), strings.ToLower(component)) {
			return true, nil
		}
	}

	// Check for affected areas
	for _, area := range analysis.AffectedAreas {
		// Convert path patterns to actual checks
		areaPattern := strings.ReplaceAll(area, "/", string(os.PathSeparator))
		if strings.Contains(path, areaPattern) {
			return true, nil
		}
	}

	// Parse Go AST for more accurate detection
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, content, parser.ParseComments)
	if err != nil {
		// Fallback to simple matching if parsing fails
		return false, nil
	}

	// Look for test functions that might be affected
	affected := false
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if strings.HasPrefix(x.Name.Name, "Test") {
				// Check if function name or comments mention affected components
				funcName := strings.ToLower(x.Name.Name)
				for _, component := range analysis.Components {
					if strings.Contains(funcName, strings.ToLower(component)) {
						affected = true
						return false
					}
				}
			}
		}
		return true
	})

	return affected, nil
}

func (f *TestFinder) isPossiblyAffected(path string, analysis *AnalysisResult) bool {
	// Check if file is in same package as affected areas
	dir := filepath.Dir(path)
	for _, area := range analysis.AffectedAreas {
		if strings.Contains(dir, filepath.Dir(area)) {
			return true
		}
	}

	// Check for integration or e2e tests
	if strings.Contains(path, "integration") || strings.Contains(path, "e2e") {
		return true
	}

	return false
}

func (f *TestFinder) isRelated(path string, analysis *AnalysisResult) bool {
	// Check for example tests
	if strings.Contains(path, "example") {
		return true
	}

	// Check for tests in related categories
	pathLower := strings.ToLower(path)
	categoryLower := strings.ToLower(analysis.Category)
	if strings.Contains(pathLower, categoryLower) {
		return true
	}

	return false
}

func (f *TestFinder) determineNewTests(analysis *AnalysisResult, existing *TestFinderResult) []string {
	needed := []string{}

	// Check coverage of requirements
	for _, req := range analysis.Requirements.Functional {
		testName := f.suggestTestName(req)
		if !f.testExists(testName, existing) {
			needed = append(needed, testName)
		}
	}

	// Add tests for specific change types
	switch analysis.Type {
	case ChangeTypeSecurity:
		needed = append(needed, "Security validation test")
		needed = append(needed, "Penetration test")
	case ChangeTypePerformance:
		needed = append(needed, "Performance benchmark test")
		needed = append(needed, "Load test")
	case ChangeTypeMigration:
		needed = append(needed, "Migration test")
		needed = append(needed, "Rollback test")
	}

	// Add tests for breaking changes
	if analysis.Breaking {
		needed = append(needed, "Backward compatibility test")
		needed = append(needed, "API version test")
	}

	return needed
}

func (f *TestFinder) suggestTestName(requirement string) string {
	// Simple heuristic to generate test name from requirement
	words := strings.Fields(requirement)
	if len(words) > 3 {
		words = words[:3]
	}
	
	testName := "Test"
	for _, word := range words {
		testName += strings.Title(strings.ToLower(word))
	}
	
	return testName
}

func (f *TestFinder) testExists(testName string, existing *TestFinderResult) bool {
	allTests := append(existing.DefinitelyAffected, existing.PossiblyAffected...)
	allTests = append(allTests, existing.RelatedTests...)
	
	for _, test := range allTests {
		if strings.Contains(test, testName) {
			return true
		}
	}
	
	return false
}