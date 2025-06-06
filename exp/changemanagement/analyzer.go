package changemanagement

import (
	"fmt"
	"strings"
)

// ChangeType represents the type of change
type ChangeType string

const (
	ChangeTypeFeature     ChangeType = "feature"
	ChangeTypeRefactoring ChangeType = "refactoring"
	ChangeTypeBugFix      ChangeType = "bugfix"
	ChangeTypePerformance ChangeType = "performance"
	ChangeTypeSecurity    ChangeType = "security"
	ChangeTypeMigration   ChangeType = "migration"
	ChangeTypeUnknown     ChangeType = "unknown"
)

// RiskLevel represents the risk level of a change
type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

// Requirements holds functional and non-functional requirements
type Requirements struct {
	Functional    []string `json:"functional"`
	NonFunctional []string `json:"non_functional"`
}

// Recommendation represents a suggested action
type Recommendation struct {
	Type       string  `json:"type"`
	Suggestion string  `json:"suggestion"`
	Confidence float64 `json:"confidence"`
}

// AnalysisResult contains the results of analyzing a change
type AnalysisResult struct {
	Type            ChangeType       `json:"type"`
	Category        string           `json:"category"`
	Components      []string         `json:"components"`
	Breaking        bool             `json:"breaking"`
	Requirements    Requirements     `json:"requirements"`
	AffectedAreas   []string         `json:"affected_areas"`
	RiskLevel       RiskLevel        `json:"risk_level"`
	Confidence      float64          `json:"confidence"`
	Recommendations []Recommendation `json:"recommendations"`
}

// ChangeAnalyzer analyzes natural language change descriptions
type ChangeAnalyzer struct {
	patterns map[string]ChangeType
	keywords map[string]string
}

// NewChangeAnalyzer creates a new change analyzer
func NewChangeAnalyzer() *ChangeAnalyzer {
	return &ChangeAnalyzer{
		patterns: map[string]ChangeType{
			"add":        ChangeTypeFeature,
			"implement":  ChangeTypeFeature,
			"create":     ChangeTypeFeature,
			"introduce":  ChangeTypeFeature,
			"refactor":   ChangeTypeRefactoring,
			"reorganize": ChangeTypeRefactoring,
			"optimize":   ChangeTypePerformance,
			"improve":    ChangeTypePerformance,
			"fix":        ChangeTypeBugFix,
			"patch":      ChangeTypeBugFix,
			"secure":     ChangeTypeSecurity,
			"migrate":    ChangeTypeMigration,
			"upgrade":    ChangeTypeMigration,
		},
		keywords: map[string]string{
			"oauth":         "authentication",
			"auth":          "authentication",
			"login":         "authentication",
			"api":           "api",
			"endpoint":      "api",
			"database":      "database",
			"db":            "database",
			"performance":   "performance",
			"speed":         "performance",
			"security":      "security",
			"vulnerability": "security",
		},
	}
}

// AnalyzeChange analyzes a natural language change description
func (a *ChangeAnalyzer) AnalyzeChange(description string) (*AnalysisResult, error) {
	if description == "" {
		return nil, fmt.Errorf("empty change description")
	}

	result := &AnalysisResult{
		Type:            ChangeTypeUnknown,
		Category:        "general",
		Components:      []string{},
		Breaking:        false,
		Requirements:    Requirements{Functional: []string{}, NonFunctional: []string{}},
		AffectedAreas:   []string{},
		RiskLevel:       RiskLow,
		Confidence:      0.5,
		Recommendations: []Recommendation{},
	}

	// Normalize description
	desc := strings.ToLower(description)

	// Detect change type
	for pattern, changeType := range a.patterns {
		if strings.Contains(desc, pattern) {
			result.Type = changeType
			result.Confidence += 0.2
			break
		}
	}

	// Detect category and components
	for keyword, category := range a.keywords {
		if strings.Contains(desc, keyword) {
			result.Category = category
			result.Components = append(result.Components, category)
			result.Confidence += 0.1
		}
	}

	// Detect breaking changes
	if strings.Contains(desc, "breaking") || strings.Contains(desc, "incompatible") {
		result.Breaking = true
		result.RiskLevel = RiskHigh
	}

	// Extract requirements (simplified)
	if strings.Contains(desc, "must") || strings.Contains(desc, "require") {
		result.Requirements.Functional = append(result.Requirements.Functional,
			"Extracted from description: "+description)
	}

	// Detect affected areas (simplified)
	if strings.Contains(desc, "all") {
		result.AffectedAreas = append(result.AffectedAreas, "global")
		result.RiskLevel = RiskMedium
	}

	// Generate recommendations
	result.Recommendations = a.generateRecommendations(result)

	// Cap confidence at 1.0
	if result.Confidence > 1.0 {
		result.Confidence = 1.0
	}

	return result, nil
}

func (a *ChangeAnalyzer) generateRecommendations(result *AnalysisResult) []Recommendation {
	recommendations := []Recommendation{}

	// High-risk changes need gradual rollout
	if result.RiskLevel == RiskHigh {
		recommendations = append(recommendations, Recommendation{
			Type:       "deployment_strategy",
			Suggestion: "Use feature flags for gradual rollout",
			Confidence: 0.95,
		})
	}

	// Breaking changes need compatibility layer
	if result.Breaking {
		recommendations = append(recommendations, Recommendation{
			Type:       "compatibility",
			Suggestion: "Implement compatibility bridge pattern",
			Confidence: 0.88,
		})
	}

	// Authentication changes need security review
	if result.Category == "authentication" {
		recommendations = append(recommendations, Recommendation{
			Type:       "security",
			Suggestion: "Conduct security review before deployment",
			Confidence: 0.92,
		})
	}

	// Performance changes need benchmarking
	if result.Type == ChangeTypePerformance {
		recommendations = append(recommendations, Recommendation{
			Type:       "testing",
			Suggestion: "Create performance benchmarks",
			Confidence: 0.85,
		})
	}

	return recommendations
}
