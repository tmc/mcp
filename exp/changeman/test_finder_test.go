package changeman

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTestFinder_FindAffectedTests(t *testing.T) {
	// Create a temporary test directory structure
	tempDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"auth/auth_test.go": `
package auth

import "testing"

func TestAuthenticate(t *testing.T) {
	// Test authentication
}

func TestAuthorize(t *testing.T) {
	// Test authorization  
}
`,
		"database/db_test.go": `
package database

import "testing"

func TestConnect(t *testing.T) {
	// Test database connection
}

func TestQuery(t *testing.T) {
	// Test database queries
}
`,
		"utils/utils_test.go": `
package utils

import "testing"

func TestStringHelper(t *testing.T) {
	// Test string utilities
}
`,
	}

	// Write test files
	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create test finder
	finder := NewTestFinder(tempDir)

	// Test various change scenarios
	tests := []struct {
		name          string
		change        *Change
		expectedTests []string // function names
		minRelevance  int
	}{
		{
			name: "authentication bug fix",
			change: &Change{
				Type:        ChangeTypeBugFix,
				Description: "Fix bug in authentication module",
				Keywords:    []string{"fix", "bug"},
				Components:  []string{"auth"},
			},
			expectedTests: []string{"TestAuthenticate", "TestAuthorize"},
			minRelevance:  30,
		},
		{
			name: "database refactor",
			change: &Change{
				Type:        ChangeTypeRefactor,
				Description: "Refactor database connection handling",
				Keywords:    []string{"refactor", "database"},
				Components:  []string{"database"},
			},
			expectedTests: []string{"TestConnect", "TestQuery"},
			minRelevance:  40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			affectedTests, err := finder.FindAffectedTests(tt.change)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check that expected tests were found
			foundTests := make(map[string]bool)
			for _, test := range affectedTests {
				foundTests[test.Function] = true

				// Check minimum relevance
				if test.Relevance < tt.minRelevance {
					t.Errorf("test %s has relevance %d, expected at least %d",
						test.Function, test.Relevance, tt.minRelevance)
				}
			}

			for _, expectedTest := range tt.expectedTests {
				if !foundTests[expectedTest] {
					t.Errorf("expected test %s not found", expectedTest)
				}
			}
		})
	}
}

func TestTestFinder_CalculateRelevance(t *testing.T) {
	finder := NewTestFinder(".")

	tests := []struct {
		name              string
		testInfo          TestInfo
		change            *Change
		expectedRelevance int
	}{
		{
			name: "exact package match",
			testInfo: TestInfo{
				Package:  "auth",
				Function: "TestLogin",
			},
			change: &Change{
				Type:       ChangeTypeBugFix,
				Components: []string{"auth"},
			},
			expectedRelevance: 40, // 30 for package match + 10 for bug fix
		},
		{
			name: "keyword match in function name",
			testInfo: TestInfo{
				Package:  "utils",
				Function: "TestDatabaseHelper",
			},
			change: &Change{
				Type:     ChangeTypeFeature,
				Keywords: []string{"database"},
			},
			expectedRelevance: 20, // 15 for keyword match + 5 for feature
		},
		{
			name: "refactor with component match",
			testInfo: TestInfo{
				Package:  "database",
				Function: "TestConnection",
			},
			change: &Change{
				Type:       ChangeTypeRefactor,
				Components: []string{"database"},
			},
			expectedRelevance: 45, // 30 for package match + 15 for refactor
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relevance := finder.calculateRelevance(tt.testInfo, tt.change, nil)
			if relevance != tt.expectedRelevance {
				t.Errorf("expected relevance %d, got %d", tt.expectedRelevance, relevance)
			}
		})
	}
}
