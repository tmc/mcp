package changeman

import (
	"fmt"
)

// ChangeManager coordinates change analysis and test finding
type ChangeManager struct {
	analyzer   *Analyzer
	testFinder *TestFinder
}

// NewChangeManager creates a new change manager
func NewChangeManager(projectRoot string) *ChangeManager {
	return &ChangeManager{
		analyzer:   NewAnalyzer(),
		testFinder: NewTestFinder(projectRoot),
	}
}

// AnalyzeChange processes a change description and finds affected tests
func (cm *ChangeManager) AnalyzeChange(description string) (*ChangeAnalysis, error) {
	// Analyze the change description
	change := cm.analyzer.Analyze(description)
	
	// Find affected tests
	affectedTests, err := cm.testFinder.FindAffectedTests(change)
	if err != nil {
		return nil, fmt.Errorf("failed to find affected tests: %w", err)
	}
	
	return &ChangeAnalysis{
		Change:        change,
		AffectedTests: affectedTests,
	}, nil
}

// ChangeAnalysis contains the complete analysis of a change
type ChangeAnalysis struct {
	Change        *Change
	AffectedTests []TestInfo
}

// Summary returns a human-readable summary of the analysis
func (ca *ChangeAnalysis) Summary() string {
	summary := fmt.Sprintf("Change Analysis:\n")
	summary += fmt.Sprintf("  Type: %s\n", changeTypeString(ca.Change.Type))
	summary += fmt.Sprintf("  Impact: %s\n", impactString(ca.Change.Impact))
	summary += fmt.Sprintf("  Keywords: %v\n", ca.Change.Keywords)
	summary += fmt.Sprintf("  Components: %v\n", ca.Change.Components)
	summary += fmt.Sprintf("\nAffected Tests (%d):\n", len(ca.AffectedTests))
	
	for i, test := range ca.AffectedTests {
		if i >= 10 {
			summary += fmt.Sprintf("  ... and %d more\n", len(ca.AffectedTests)-10)
			break
		}
		summary += fmt.Sprintf("  %s.%s (relevance: %d%%)\n", 
			test.Package, test.Function, test.Relevance)
	}
	
	return summary
}

// Helper functions to convert enums to strings
func changeTypeString(ct ChangeType) string {
	switch ct {
	case ChangeTypeBugFix:
		return "Bug Fix"
	case ChangeTypeFeature:
		return "Feature"
	case ChangeTypeRefactor:
		return "Refactor"
	case ChangeTypeTest:
		return "Test"
	case ChangeTypeDocumentation:
		return "Documentation"
	case ChangeTypePerformance:
		return "Performance"
	case ChangeTypeSecurity:
		return "Security"
	default:
		return "Unknown"
	}
}

func impactString(impact ChangeImpact) string {
	switch impact {
	case ImpactHigh:
		return "High"
	case ImpactMedium:
		return "Medium"
	case ImpactLow:
		return "Low"
	default:
		return "Unknown"
	}
}