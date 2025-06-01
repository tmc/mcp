package changemanagement

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// MutationStrategy represents a type of mutation
type MutationStrategy string

const (
	MutationReorder MutationStrategy = "reorder"
	MutationFuzz    MutationStrategy = "fuzz"
	MutationTiming  MutationStrategy = "timing"
	MutationError   MutationStrategy = "error"
)

// Mutation represents a mutated test
type Mutation struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	Changes string `json:"changes"`
}

// TestMutator generates test mutations
type TestMutator struct {
	rand *rand.Rand
}

// NewTestMutator creates a new test mutator
func NewTestMutator() *TestMutator {
	return &TestMutator{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// MutateTest generates mutations of a test
func (m *TestMutator) MutateTest(content string, strategies []MutationStrategy, count int) ([]Mutation, error) {
	mutations := []Mutation{}
	lines := strings.Split(content, "\n")

	for i := 0; i < count; i++ {
		// Select random strategy
		strategy := strategies[m.rand.Intn(len(strategies))]
		
		mutation, err := m.applyStrategy(lines, strategy)
		if err != nil {
			continue // Skip failed mutations
		}
		
		mutations = append(mutations, mutation)
	}

	return mutations, nil
}

func (m *TestMutator) applyStrategy(lines []string, strategy MutationStrategy) (Mutation, error) {
	switch strategy {
	case MutationReorder:
		return m.reorderCommands(lines)
	case MutationFuzz:
		return m.fuzzInputs(lines)
	case MutationTiming:
		return m.mutateTiming(lines)
	case MutationError:
		return m.injectErrors(lines)
	default:
		return Mutation{}, fmt.Errorf("unknown strategy: %s", strategy)
	}
}

func (m *TestMutator) reorderCommands(lines []string) (Mutation, error) {
	newLines := make([]string, len(lines))
	copy(newLines, lines)
	
	// Find exec commands
	execLines := []int{}
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "exec ") {
			execLines = append(execLines, i)
		}
	}
	
	if len(execLines) < 2 {
		return Mutation{}, fmt.Errorf("not enough exec commands to reorder")
	}
	
	// Swap two random exec commands
	i := m.rand.Intn(len(execLines))
	j := m.rand.Intn(len(execLines))
	if i != j {
		newLines[execLines[i]], newLines[execLines[j]] = newLines[execLines[j]], newLines[execLines[i]]
	}
	
	return Mutation{
		Type:    "reorder",
		Content: strings.Join(newLines, "\n"),
		Changes: fmt.Sprintf("Swapped lines %d and %d", execLines[i]+1, execLines[j]+1),
	}, nil
}

func (m *TestMutator) fuzzInputs(lines []string) (Mutation, error) {
	newLines := make([]string, len(lines))
	copy(newLines, lines)
	
	// Find lines with JSON inputs
	for i, line := range lines {
		if strings.Contains(line, "{") && strings.Contains(line, "}") {
			// Simple JSON fuzzing
			fuzzed := m.fuzzJSON(line)
			newLines[i] = fuzzed
			
			return Mutation{
				Type:    "fuzz",
				Content: strings.Join(newLines, "\n"),
				Changes: fmt.Sprintf("Fuzzed input on line %d", i+1),
			}, nil
		}
	}
	
	return Mutation{}, fmt.Errorf("no JSON inputs found to fuzz")
}

func (m *TestMutator) fuzzJSON(line string) string {
	// Simple fuzzing strategies
	strategies := []func(string) string{
		// Add extra field
		func(s string) string {
			return strings.Replace(s, "}", `, "fuzz": "test"}`, 1)
		},
		// Change number
		func(s string) string {
			return strings.Replace(s, ": 1", ": 999", 1)
		},
		// Make value null
		func(s string) string {
			return strings.Replace(s, `: "`, `: null`, 1)
		},
		// Empty string
		func(s string) string {
			return strings.Replace(s, `"test"`, `""`, 1)
		},
	}
	
	strategy := strategies[m.rand.Intn(len(strategies))]
	return strategy(line)
}

func (m *TestMutator) mutateTiming(lines []string) (Mutation, error) {
	newLines := make([]string, len(lines))
	copy(newLines, lines)
	
	// Find sleep commands
	for i, line := range lines {
		if strings.Contains(line, "sleep") {
			// Modify sleep duration
			modifications := []string{
				"exec sleep 0.1",
				"exec sleep 0.01",
				"exec sleep 5",
				"# exec sleep 1  # Removed sleep",
			}
			
			newLines[i] = modifications[m.rand.Intn(len(modifications))]
			
			return Mutation{
				Type:    "timing",
				Content: strings.Join(newLines, "\n"),
				Changes: fmt.Sprintf("Modified timing on line %d", i+1),
			}, nil
		}
	}
	
	// If no sleep found, add one
	insertPos := m.rand.Intn(len(lines))
	newLines = append(newLines[:insertPos], append([]string{"exec sleep 1"}, newLines[insertPos:]...)...)
	
	return Mutation{
		Type:    "timing",
		Content: strings.Join(newLines, "\n"),
		Changes: fmt.Sprintf("Inserted sleep at line %d", insertPos+1),
	}, nil
}

func (m *TestMutator) injectErrors(lines []string) (Mutation, error) {
	newLines := make([]string, len(lines))
	copy(newLines, lines)
	
	// Find stdout assertions and change to stderr
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "stdout ") {
			newLines[i] = strings.Replace(line, "stdout", "stderr", 1)
			
			return Mutation{
				Type:    "error",
				Content: strings.Join(newLines, "\n"),
				Changes: fmt.Sprintf("Changed stdout to stderr on line %d", i+1),
			}, nil
		}
	}
	
	// Add error check
	insertPos := m.rand.Intn(len(lines))
	newLines = append(newLines[:insertPos], append([]string{"! stderr 'error'"}, newLines[insertPos:]...)...)
	
	return Mutation{
		Type:    "error",
		Content: strings.Join(newLines, "\n"),
		Changes: fmt.Sprintf("Inserted error check at line %d", insertPos+1),
	}, nil
}