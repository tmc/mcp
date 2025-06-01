package changemanagement_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/tmc/mcp/exp/changemanagement"
)

func TestIntegrationWorkflow(t *testing.T) {
	// Test the complete workflow
	tempDir := t.TempDir()

	// 1. Analyze a change
	t.Run("AnalyzeChange", func(t *testing.T) {
		analyzer := changemanagement.NewChangeAnalyzer()
		result, err := analyzer.AnalyzeChange("Add OAuth2 authentication to all API endpoints with rate limiting")
		if err != nil {
			t.Fatalf("Analysis failed: %v", err)
		}

		// Verify analysis results
		if result.Type != changemanagement.ChangeTypeFeature {
			t.Errorf("Expected feature type, got %s", result.Type)
		}

		if result.Category != "authentication" {
			t.Errorf("Expected authentication category, got %s", result.Category)
		}

		// Save for next steps
		data, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("Failed to marshal analysis: %v", err)
		}
		analysisPath := filepath.Join(tempDir, "analysis.json")
		if err := os.WriteFile(analysisPath, data, 0644); err != nil {
			t.Fatalf("Failed to save analysis: %v", err)
		}
	})

	// 2. Find affected tests
	t.Run("FindTests", func(t *testing.T) {
		finder := changemanagement.NewTestFinder(".")
		
		// Load saved analysis
		analysisPath := filepath.Join(tempDir, "analysis.json")
		data, err := os.ReadFile(analysisPath)
		if err != nil {
			t.Fatalf("Failed to read analysis: %v", err)
		}

		var analysis changemanagement.AnalysisResult
		if err := json.Unmarshal(data, &analysis); err != nil {
			t.Fatalf("Failed to unmarshal analysis: %v", err)
		}

		result, err := finder.FindAffectedTests(&analysis)
		if err != nil {
			t.Fatalf("Test finding failed: %v", err)
		}

		// We should find some tests
		totalTests := len(result.DefinitelyAffected) + 
			len(result.PossiblyAffected) + 
			len(result.RelatedTests)
		
		if totalTests == 0 {
			t.Log("Warning: No tests found (this might be expected in a new project)")
		}

		// Should suggest new tests for OAuth2
		if len(result.NewTestsNeeded) == 0 {
			t.Error("Expected new tests to be suggested")
		}
	})

	// 3. Generate documentation
	t.Run("GenerateDocumentation", func(t *testing.T) {
		generator := changemanagement.NewDocumentationGenerator()
		
		// Load saved analysis
		analysisPath := filepath.Join(tempDir, "analysis.json")
		data, err := os.ReadFile(analysisPath)
		if err != nil {
			t.Fatalf("Failed to read analysis: %v", err)
		}

		var analysis changemanagement.AnalysisResult
		if err := json.Unmarshal(data, &analysis); err != nil {
			t.Fatalf("Failed to unmarshal analysis: %v", err)
		}

		docs, err := generator.GenerateDocs(&analysis, "markdown")
		if err != nil {
			t.Fatalf("Documentation generation failed: %v", err)
		}

		// Should generate multiple documents
		if len(docs) < 2 {
			t.Errorf("Expected at least 2 documents, got %d", len(docs))
		}

		// Check for expected files
		foundOverview := false
		foundSecurity := false
		for _, doc := range docs {
			if doc.Type == "overview" {
				foundOverview = true
			}
			if doc.Type == "security" {
				foundSecurity = true
			}
		}

		if !foundOverview {
			t.Error("Expected overview documentation")
		}
		if !foundSecurity {
			t.Error("Expected security documentation for auth changes")
		}
	})

	// 4. Test mutations
	t.Run("MutateTests", func(t *testing.T) {
		mutator := changemanagement.NewTestMutator()
		
		// Create a sample test
		sampleTest := `exec mcp-server start
stdout 'Server started'
exec mcp-tool call auth '{"user": "test"}'
stdout '{"token": "'
exec sleep 1
exec mcp-server stop`

		mutations, err := mutator.MutateTest(sampleTest, 
			[]changemanagement.MutationStrategy{
				changemanagement.MutationReorder,
				changemanagement.MutationFuzz,
			}, 5)
		
		if err != nil {
			t.Fatalf("Mutation failed: %v", err)
		}

		if len(mutations) != 5 {
			t.Errorf("Expected 5 mutations, got %d", len(mutations))
		}

		// Check mutation types
		types := make(map[string]int)
		for _, m := range mutations {
			types[m.Type]++
		}

		if len(types) == 0 {
			t.Error("No mutation types recorded")
		}
	})
}

func TestChangeAnalyzerPatterns(t *testing.T) {
	analyzer := changemanagement.NewChangeAnalyzer()
	
	testCases := []struct {
		name         string
		description  string
		expectedType changemanagement.ChangeType
		expectedCat  string
		breaking     bool
	}{
		{
			name:         "Security Update",
			description:  "Fix security vulnerability in user authentication",
			expectedType: changemanagement.ChangeTypeSecurity,
			expectedCat:  "authentication",
			breaking:     false,
		},
		{
			name:         "Performance Optimization",
			description:  "Optimize database queries to improve API response time",
			expectedType: changemanagement.ChangeTypePerformance,
			expectedCat:  "database",
			breaking:     false,
		},
		{
			name:         "Breaking API Change",
			description:  "Refactor API endpoints with breaking changes to authentication",
			expectedType: changemanagement.ChangeTypeRefactoring,
			expectedCat:  "api",
			breaking:     true,
		},
		{
			name:         "Database Migration",
			description:  "Migrate from PostgreSQL to MongoDB",
			expectedType: changemanagement.ChangeTypeMigration,
			expectedCat:  "database",
			breaking:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := analyzer.AnalyzeChange(tc.description)
			if err != nil {
				t.Fatalf("Analysis failed: %v", err)
			}

			if result.Type != tc.expectedType {
				t.Errorf("Expected type %s, got %s", tc.expectedType, result.Type)
			}

			if result.Category != tc.expectedCat {
				t.Errorf("Expected category %s, got %s", tc.expectedCat, result.Category)
			}

			if result.Breaking != tc.breaking {
				t.Errorf("Expected breaking=%v, got %v", tc.breaking, result.Breaking)
			}
		})
	}
}

func TestDocumentationQuality(t *testing.T) {
	// Test that generated documentation meets quality standards
	generator := changemanagement.NewDocumentationGenerator()
	
	analysis := &changemanagement.AnalysisResult{
		Type:      changemanagement.ChangeTypeSecurity,
		Category:  "authentication",
		Breaking:  true,
		RiskLevel: changemanagement.RiskHigh,
		Components: []string{"auth", "api", "middleware"},
		Requirements: changemanagement.Requirements{
			Functional: []string{
				"Implement OAuth2 authentication",
				"Add rate limiting",
				"Support token refresh",
			},
		},
		Recommendations: []changemanagement.Recommendation{
			{
				Type:       "deployment_strategy",
				Suggestion: "Use feature flags for gradual rollout",
				Confidence: 0.95,
			},
		},
	}

	docs, err := generator.GenerateDocs(analysis, "markdown")
	if err != nil {
		t.Fatalf("Documentation generation failed: %v", err)
	}

	// Check documentation completeness
	foundElements := map[string]bool{
		"# ":            false, // Headers
		"**":            false, // Bold text
		"- ":            false, // Lists
		"```":           false, // Code blocks
		"Risk Level":    false, // Risk info
		"Breaking":      false, // Breaking change info
		"OAuth2":        false, // Specific requirement
		"feature flags": false, // Recommendation
	}

	for _, doc := range docs {
		for element := range foundElements {
			if strings.Contains(doc.Content, element) {
				foundElements[element] = true
			}
		}
	}

	for element, found := range foundElements {
		if !found {
			t.Errorf("Documentation missing expected element: %s", element)
		}
	}
}