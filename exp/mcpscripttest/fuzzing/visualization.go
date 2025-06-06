package fuzzing

import (
	"github.com/tmc/mcp/exp/mcpscripttest"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Visualizer provides live visualization of fuzzing activity
type Visualizer struct {
	mu          sync.Mutex
	writer      io.Writer
	enabled     bool
	showRejected bool
	
	// Statistics
	totalTested    int64
	totalAccepted  int64
	totalRejected  int64
	lastUpdate     time.Time
	
	// Display options
	clearScreen    bool
	showStats      bool
	showScript     bool
	maxScriptLines int
	updateInterval time.Duration
	
	// Current state
	currentScript  string
	currentStatus  string
	lastError      error
}

// NewVisualizer creates a new fuzzing visualizer
func NewVisualizer(opts VisualizerOptions) *Visualizer {
	return &Visualizer{
		writer:         opts.Writer,
		enabled:        opts.Enabled,
		showRejected:   opts.ShowRejected,
		clearScreen:    opts.ClearScreen,
		showStats:      opts.ShowStats,
		showScript:     opts.ShowScript,
		maxScriptLines: opts.MaxScriptLines,
		updateInterval: opts.UpdateInterval,
		lastUpdate:     time.Now(),
	}
}

// VisualizerOptions configures the visualizer
type VisualizerOptions struct {
	// Writer is where to write the visualization (default: os.Stdout)
	Writer io.Writer
	
	// Enabled determines if visualization is active
	Enabled bool
	
	// ShowRejected shows scripts that fail validation
	ShowRejected bool
	
	// ClearScreen clears the terminal between updates
	ClearScreen bool
	
	// ShowStats displays fuzzing statistics
	ShowStats bool
	
	// ShowScript displays the current script being tested
	ShowScript bool
	
	// MaxScriptLines limits how many lines of script to show
	MaxScriptLines int
	
	// UpdateInterval controls how often to update the display
	UpdateInterval time.Duration
}

// DefaultVisualizerOptions returns default visualization options
func DefaultVisualizerOptions() VisualizerOptions {
	return VisualizerOptions{
		Writer:         os.Stdout,
		Enabled:        false,
		ShowRejected:   false,
		ClearScreen:    false,
		ShowStats:      true,
		ShowScript:     true,
		MaxScriptLines: 20,
		UpdateInterval: 100 * time.Millisecond,
	}
}

// StartTest records the beginning of a test
func (v *Visualizer) StartTest(script string) {
	if !v.enabled {
		return
	}
	
	v.mu.Lock()
	defer v.mu.Unlock()
	
	v.totalTested++
	v.currentScript = script
	v.currentStatus = "testing"
	v.lastError = nil
}

// AcceptScript records a script that passed validation
func (v *Visualizer) AcceptScript(script string) {
	if !v.enabled {
		return
	}
	
	v.mu.Lock()
	defer v.mu.Unlock()
	
	v.totalAccepted++
	v.currentStatus = "accepted"
	
	// Only update display for accepted scripts (or if ShowRejected is true)
	if v.shouldUpdate() {
		v.updateDisplay()
	}
}

// RejectScript records a script that failed validation
func (v *Visualizer) RejectScript(script string, err error) {
	if !v.enabled {
		return
	}
	
	v.mu.Lock()
	defer v.mu.Unlock()
	
	v.totalRejected++
	v.currentStatus = "rejected"
	v.lastError = err
	
	// Only update display if ShowRejected is true
	if v.showRejected && v.shouldUpdate() {
		v.updateDisplay()
	}
}

// shouldUpdate checks if it's time to update the display
func (v *Visualizer) shouldUpdate() bool {
	now := time.Now()
	if now.Sub(v.lastUpdate) >= v.updateInterval {
		v.lastUpdate = now
		return true
	}
	return false
}

// updateDisplay refreshes the visualization
func (v *Visualizer) updateDisplay() {
	var output strings.Builder
	
	// Clear screen if requested
	if v.clearScreen {
		output.WriteString("\033[H\033[2J") // ANSI escape codes to clear screen
	}
	
	// Header
	output.WriteString("=== MCPScriptTest Fuzzer ===\n\n")
	
	// Statistics
	if v.showStats {
		output.WriteString(fmt.Sprintf("Total Tested:  %d\n", v.totalTested))
		output.WriteString(fmt.Sprintf("Accepted:      %d (%.1f%%)\n", 
			v.totalAccepted, 
			float64(v.totalAccepted)/float64(max(v.totalTested, 1))*100))
		output.WriteString(fmt.Sprintf("Rejected:      %d (%.1f%%)\n", 
			v.totalRejected,
			float64(v.totalRejected)/float64(max(v.totalTested, 1))*100))
		output.WriteString(fmt.Sprintf("Status:        %s\n", v.currentStatus))
		output.WriteString("\n")
	}
	
	// Current script
	if v.showScript && v.currentScript != "" {
		output.WriteString("Current Script:\n")
		output.WriteString("---------------\n")
		
		lines := strings.Split(v.currentScript, "\n")
		displayLines := lines
		if len(lines) > v.maxScriptLines {
			displayLines = lines[:v.maxScriptLines]
		}
		
		for i, line := range displayLines {
			output.WriteString(fmt.Sprintf("%3d: %s\n", i+1, line))
		}
		
		if len(lines) > v.maxScriptLines {
			output.WriteString(fmt.Sprintf("... (%d more lines)\n", len(lines)-v.maxScriptLines))
		}
		output.WriteString("\n")
	}
	
	// Error if rejected and showing rejected
	if v.currentStatus == "rejected" && v.lastError != nil && v.showRejected {
		output.WriteString(fmt.Sprintf("Error: %v\n\n", v.lastError))
	}
	
	// Write to output
	fmt.Fprint(v.writer, output.String())
}

// UpdateCoverage updates the display with coverage information
func (v *Visualizer) UpdateCoverage(coverage float64, newLines int) {
	if !v.enabled {
		return
	}
	
	v.mu.Lock()
	defer v.mu.Unlock()
	
	// Add coverage info to the current status
	v.currentStatus = fmt.Sprintf("accepted (coverage: %.1f%%, +%d lines)", coverage, newLines)
	
	if v.shouldUpdate() {
		v.updateDisplay()
	}
}

// Close cleans up the visualizer
func (v *Visualizer) Close() {
	if !v.enabled {
		return
	}
	
	v.mu.Lock()
	defer v.mu.Unlock()
	
	// Final statistics
	var output strings.Builder
	output.WriteString("\n=== Final Statistics ===\n")
	output.WriteString(fmt.Sprintf("Total Tested:  %d\n", v.totalTested))
	output.WriteString(fmt.Sprintf("Accepted:      %d (%.1f%%)\n", 
		v.totalAccepted, 
		float64(v.totalAccepted)/float64(max(v.totalTested, 1))*100))
	output.WriteString(fmt.Sprintf("Rejected:      %d (%.1f%%)\n", 
		v.totalRejected,
		float64(v.totalRejected)/float64(max(v.totalTested, 1))*100))
	
	fmt.Fprint(v.writer, output.String())
}

// max returns the larger of two int64 values
func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}