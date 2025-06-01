package changeman

import (
	"testing"
)

func TestAnalyzer_Analyze(t *testing.T) {
	analyzer := NewAnalyzer()
	
	tests := []struct {
		name           string
		description    string
		expectedType   ChangeType
		expectedImpact ChangeImpact
		expectedKeywords []string
	}{
		{
			name:           "bug fix detection",
			description:    "Fix critical bug in authentication module",
			expectedType:   ChangeTypeBugFix,
			expectedImpact: ImpactHigh,
			expectedKeywords: []string{"fix", "bug"},
		},
		{
			name:           "feature detection",
			description:    "Add new feature to support OAuth2 authentication",
			expectedType:   ChangeTypeFeature,
			expectedImpact: ImpactMedium,
			expectedKeywords: []string{"add", "new", "feature", "support"},
		},
		{
			name:           "refactor detection",
			description:    "Refactor database connection handling to improve performance",
			expectedType:   ChangeTypeRefactor,
			expectedImpact: ImpactMedium,
			expectedKeywords: []string{"refactor", "improve", "performance"},
		},
		{
			name:           "security fix detection",
			description:    "Fix security vulnerability in user permissions",
			expectedType:   ChangeTypeSecurity,
			expectedImpact: ImpactHigh,
			expectedKeywords: []string{"fix", "security", "vulnerability"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			change := analyzer.Analyze(tt.description)
			
			if change.Type != tt.expectedType {
				t.Errorf("expected type %v, got %v", tt.expectedType, change.Type)
			}
			
			if change.Impact != tt.expectedImpact {
				t.Errorf("expected impact %v, got %v", tt.expectedImpact, change.Impact)
			}
			
			// Check if all expected keywords were found
			keywordMap := make(map[string]bool)
			for _, kw := range change.Keywords {
				keywordMap[kw] = true
			}
			
			for _, expectedKw := range tt.expectedKeywords {
				if !keywordMap[expectedKw] {
					t.Errorf("expected keyword '%s' not found in %v", expectedKw, change.Keywords)
				}
			}
		})
	}
}

func TestAnalyzer_ExtractComponents(t *testing.T) {
	analyzer := NewAnalyzer()
	
	tests := []struct {
		name              string
		description       string
		expectedComponents []string
	}{
		{
			name:              "path-like components",
			description:       "Update cmd/mcp-proxy to fix connection issues",
			expectedComponents: []string{"Update", "cmd/mcp-proxy"},
		},
		{
			name:              "capitalized components",
			description:       "Fix bug in TestFinder module",
			expectedComponents: []string{"Fix", "TestFinder"},
		},
		{
			name:              "mixed components",
			description:       "Refactor modelcontextprotocol/draft to support new schema",
			expectedComponents: []string{"Refactor", "modelcontextprotocol/draft"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			change := analyzer.Analyze(tt.description)
			
			// Create a map for easier checking
			componentMap := make(map[string]bool)
			for _, comp := range change.Components {
				componentMap[comp] = true
			}
			
			for _, expected := range tt.expectedComponents {
				if !componentMap[expected] {
					t.Errorf("expected component '%s' not found in %v", expected, change.Components)
				}
			}
		})
	}
}