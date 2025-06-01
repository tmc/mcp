package changeman

import (
	"strings"
)

// ChangeType represents the type of change detected
type ChangeType int

const (
	ChangeTypeUnknown ChangeType = iota
	ChangeTypeBugFix
	ChangeTypeFeature
	ChangeTypeRefactor
	ChangeTypeTest
	ChangeTypeDocumentation
	ChangeTypePerformance
	ChangeTypeSecurity
)

// ChangeImpact represents the impact level of a change
type ChangeImpact int

const (
	ImpactLow ChangeImpact = iota
	ImpactMedium
	ImpactHigh
)

// Change represents an analyzed change request
type Change struct {
	Type        ChangeType
	Impact      ChangeImpact
	Description string
	Keywords    []string
	Components  []string // affected components/packages
}

// Analyzer analyzes natural language change descriptions
type Analyzer struct {
	// keyword mappings for change type detection
	typeKeywords map[ChangeType][]string
}

// NewAnalyzer creates a new change analyzer
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		typeKeywords: map[ChangeType][]string{
			ChangeTypeBugFix: {"fix", "bug", "issue", "error", "crash", "broken", "repair", "resolve", "patch"},
			ChangeTypeFeature: {"add", "feature", "implement", "new", "enhance", "extend", "create", "support"},
			ChangeTypeRefactor: {"refactor", "restructure", "reorganize", "cleanup", "simplify", "optimize", "improve"},
			ChangeTypeTest: {"test", "testing", "coverage", "unit test", "integration test", "spec"},
			ChangeTypeDocumentation: {"doc", "documentation", "readme", "comment", "explain", "clarify"},
			ChangeTypePerformance: {"performance", "speed", "optimize", "faster", "efficient", "cache", "latency"},
			ChangeTypeSecurity: {"security", "vulnerability", "auth", "encrypt", "permission", "access", "safe"},
		},
	}
}

// Analyze parses a natural language change description and returns structured change info
func (a *Analyzer) Analyze(description string) *Change {
	change := &Change{
		Type:        ChangeTypeUnknown,
		Impact:      ImpactLow,
		Description: description,
		Keywords:    []string{},
		Components:  []string{},
	}

	// Normalize description for analysis
	lowerDesc := strings.ToLower(description)
	words := strings.Fields(lowerDesc)
	
	// Detect change type based on keywords
	change.Type = a.detectChangeType(lowerDesc)
	
	// Extract keywords
	change.Keywords = a.extractKeywords(words)
	
	// Detect impact level
	change.Impact = a.detectImpact(lowerDesc)
	
	// Extract component names (simple heuristic: words containing "/" or starting with capitals)
	change.Components = a.extractComponents(description)
	
	return change
}

// detectChangeType identifies the type of change based on keywords
func (a *Analyzer) detectChangeType(description string) ChangeType {
	maxScore := 0
	detectedType := ChangeTypeUnknown
	
	for changeType, keywords := range a.typeKeywords {
		score := 0
		for _, keyword := range keywords {
			if strings.Contains(description, keyword) {
				score++
			}
		}
		if score > maxScore {
			maxScore = score
			detectedType = changeType
		}
	}
	
	return detectedType
}

// extractKeywords identifies important keywords from the description
func (a *Analyzer) extractKeywords(words []string) []string {
	keywords := []string{}
	keywordSet := make(map[string]bool)
	
	// Collect all keywords from type definitions
	allKeywords := make(map[string]bool)
	for _, typeKeywords := range a.typeKeywords {
		for _, keyword := range typeKeywords {
			allKeywords[keyword] = true
		}
	}
	
	// Find matching keywords
	for _, word := range words {
		if allKeywords[word] && !keywordSet[word] {
			keywords = append(keywords, word)
			keywordSet[word] = true
		}
	}
	
	return keywords
}

// detectImpact estimates the impact level of the change
func (a *Analyzer) detectImpact(description string) ChangeImpact {
	// High impact keywords
	if strings.Contains(description, "breaking") || strings.Contains(description, "major") ||
		strings.Contains(description, "critical") || strings.Contains(description, "security") {
		return ImpactHigh
	}
	
	// Medium impact keywords
	if strings.Contains(description, "refactor") || strings.Contains(description, "feature") ||
		strings.Contains(description, "significant") {
		return ImpactMedium
	}
	
	// Default to low impact
	return ImpactLow
}

// extractComponents attempts to identify component/package names from the description
func (a *Analyzer) extractComponents(description string) []string {
	components := []string{}
	words := strings.Fields(description)
	
	for _, word := range words {
		// Remove common punctuation
		word = strings.Trim(word, ".,!?;:")
		
		// Check for path-like structures
		if strings.Contains(word, "/") {
			components = append(components, word)
			continue
		}
		
		// Check for capitalized words that might be component names
		if len(word) > 0 && strings.ToUpper(word[0:1]) == word[0:1] && word != strings.ToUpper(word) {
			// Skip common words
			if !isCommonWord(strings.ToLower(word)) {
				components = append(components, word)
			}
		}
	}
	
	return components
}

// isCommonWord checks if a word is a common English word to filter out
func isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "from": true,
		"as": true, "is": true, "was": true, "are": true, "were": true,
		"been": true, "be": true, "will": true, "can": true, "could": true,
		"should": true, "would": true, "may": true, "might": true,
	}
	return commonWords[word]
}