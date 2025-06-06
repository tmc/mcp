package changeman

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
)

// TestInfo represents information about a test
type TestInfo struct {
	Package  string
	Function string
	FilePath string
	Line     int
	// Relevance score (0-100) indicating how likely this test is affected by the change
	Relevance int
}

// TestFinder locates tests that might be affected by a change
type TestFinder struct {
	rootDir string
	// cache of package imports to identify dependencies
	packageImports map[string][]string
}

// NewTestFinder creates a new test finder
func NewTestFinder(rootDir string) *TestFinder {
	return &TestFinder{
		rootDir:        rootDir,
		packageImports: make(map[string][]string),
	}
}

// FindAffectedTests finds tests that might be affected by the given change
func (tf *TestFinder) FindAffectedTests(change *Change) ([]TestInfo, error) {
	var tests []TestInfo

	// First, scan for all test files
	testFiles, err := tf.findTestFiles()
	if err != nil {
		return nil, err
	}

	// Analyze each test file
	for _, testFile := range testFiles {
		fileTests, err := tf.analyzeTestFile(testFile, change)
		if err != nil {
			continue // Skip files with errors
		}
		tests = append(tests, fileTests...)
	}

	// Sort tests by relevance score
	sortTestsByRelevance(tests)

	return tests, nil
}

// findTestFiles locates all test files in the project
func (tf *TestFinder) findTestFiles() ([]string, error) {
	var testFiles []string

	err := filepath.WalkDir(tf.rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip directories with errors
		}

		// Skip vendor directories
		if strings.Contains(path, "vendor/") {
			return filepath.SkipDir
		}

		// Find test files
		if !d.IsDir() && strings.HasSuffix(path, "_test.go") {
			testFiles = append(testFiles, path)
		}

		return nil
	})

	return testFiles, err
}

// analyzeTestFile analyzes a single test file for relevant tests
func (tf *TestFinder) analyzeTestFile(filePath string, change *Change) ([]TestInfo, error) {
	var tests []TestInfo

	// Parse the file
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// Extract package name
	packageName := node.Name.Name

	// Extract imports for dependency analysis
	var imports []string
	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		imports = append(imports, importPath)
	}
	tf.packageImports[packageName] = imports

	// Find test functions
	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}

		// Check if it's a test function
		if strings.HasPrefix(fn.Name.Name, "Test") {
			position := fset.Position(fn.Pos())
			testInfo := TestInfo{
				Package:  packageName,
				Function: fn.Name.Name,
				FilePath: filePath,
				Line:     position.Line,
			}

			// Calculate relevance score
			testInfo.Relevance = tf.calculateRelevance(testInfo, change, node)

			// Only include tests with non-zero relevance
			if testInfo.Relevance > 0 {
				tests = append(tests, testInfo)
			}
		}

		return true
	})

	return tests, nil
}

// calculateRelevance calculates how relevant a test is to the given change
func (tf *TestFinder) calculateRelevance(test TestInfo, change *Change, file *ast.File) int {
	relevance := 0

	// Check if the test package matches any of the change components
	for _, component := range change.Components {
		if strings.Contains(test.Package, strings.ToLower(component)) {
			relevance += 30
		}
		if strings.Contains(test.Function, component) {
			relevance += 20
		}
	}

	// Check if test name contains relevant keywords
	testNameLower := strings.ToLower(test.Function)
	for _, keyword := range change.Keywords {
		if strings.Contains(testNameLower, keyword) {
			relevance += 15
		}
	}

	// Analyze imports for relevance
	for _, imp := range tf.packageImports[test.Package] {
		for _, component := range change.Components {
			if strings.Contains(imp, strings.ToLower(component)) {
				relevance += 10
			}
		}
	}

	// Boost relevance based on change type
	switch change.Type {
	case ChangeTypeBugFix:
		// Bug fixes are more likely to affect existing tests
		relevance += 10
	case ChangeTypeFeature:
		// New features might need new tests
		relevance += 5
	case ChangeTypeRefactor:
		// Refactoring often affects many tests
		relevance += 15
	case ChangeTypeTest:
		// Test changes highly affect other tests in the same package
		if strings.Contains(test.FilePath, "_test.go") {
			relevance += 20
		}
	}

	// Cap relevance at 100
	if relevance > 100 {
		relevance = 100
	}

	return relevance
}

// sortTestsByRelevance sorts tests by their relevance score in descending order
func sortTestsByRelevance(tests []TestInfo) {
	// Simple bubble sort for now (can be optimized later)
	for i := 0; i < len(tests)-1; i++ {
		for j := 0; j < len(tests)-i-1; j++ {
			if tests[j].Relevance < tests[j+1].Relevance {
				tests[j], tests[j+1] = tests[j+1], tests[j]
			}
		}
	}
}
