// coverage-hotspots analyzes coverage data to find the most hit but uncovered lines
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"
)

var (
	coverageDir string
	profileFile string
	threshold   int
	topN        int
	outputFile  string
	format      string
	verbose     bool
)

func init() {
	flag.StringVar(&coverageDir, "coverage", "", "Coverage data directory")
	flag.StringVar(&profileFile, "profile", "", "Coverage profile file")
	flag.IntVar(&threshold, "threshold", 10, "Minimum hit count to consider")
	flag.IntVar(&topN, "top", 20, "Show top N hotspots")
	flag.StringVar(&outputFile, "output", "", "Output file (default: stdout)")
	flag.StringVar(&format, "format", "text", "Output format: text, json")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")
	
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "coverage-hotspots - Find most hit but uncovered code\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  coverage-hotspots [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Find hotspots from coverage directory\n")
		fmt.Fprintf(os.Stderr, "  coverage-hotspots -coverage /tmp/coverage\n")
		fmt.Fprintf(os.Stderr, "  \n")
		fmt.Fprintf(os.Stderr, "  # Find top 10 hotspots with at least 100 hits\n")
		fmt.Fprintf(os.Stderr, "  coverage-hotspots -coverage /tmp/coverage -top 10 -threshold 100\n")
		fmt.Fprintf(os.Stderr, "  \n")
		fmt.Fprintf(os.Stderr, "  # Output as JSON for tooling\n")
		fmt.Fprintf(os.Stderr, "  coverage-hotspots -coverage /tmp/coverage -format json\n")
	}
}

type Hotspot struct {
	Package    string `json:"package"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	HitCount   int64  `json:"hit_count"`
	Function   string `json:"function"`
	SourceLine string `json:"source_line"`
}

type ProfileLine struct {
	File      string
	Line      int
	Covered   bool
	HitCount  int64
	Function  string
}

func main() {
	flag.Parse()
	
	if coverageDir == "" && profileFile == "" {
		flag.Usage()
		os.Exit(1)
	}
	
	var hotspots []Hotspot
	
	if profileFile != "" {
		// Analyze profile file
		lines, err := analyzeProfile(profileFile)
		if err != nil {
			log.Fatalf("Error analyzing profile: %v", err)
		}
		hotspots = findHotspots(lines)
	} else {
		// Convert covdata to profile and analyze
		profile, err := convertCovdataToProfile(coverageDir)
		if err != nil {
			log.Fatalf("Error converting coverage data: %v", err)
		}
		
		lines, err := analyzeProfile(profile)
		if err != nil {
			log.Fatalf("Error analyzing profile: %v", err)
		}
		hotspots = findHotspots(lines)
		
		// Clean up temp file
		os.Remove(profile)
	}
	
	// Sort by hit count
	sort.Slice(hotspots, func(i, j int) bool {
		return hotspots[i].HitCount > hotspots[j].HitCount
	})
	
	// Limit to topN
	if len(hotspots) > topN {
		hotspots = hotspots[:topN]
	}
	
	// Output results
	var out = os.Stdout
	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			log.Fatalf("Error creating output file: %v", err)
		}
		defer f.Close()
		out = f
	}
	
	switch format {
	case "json":
		outputJSON(out, hotspots)
	default:
		outputText(out, hotspots)
	}
}

func convertCovdataToProfile(coverDir string) (string, error) {
	// Create temp file for profile
	tmpfile, err := os.CreateTemp("", "coverage-*.out")
	if err != nil {
		return "", err
	}
	tmpfile.Close()
	
	// Convert using go tool covdata
	cmd := exec.Command("go", "tool", "covdata", "textfmt", "-i", coverDir, "-o", tmpfile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error converting coverage: %v\n%s", err, output)
	}
	
	return tmpfile.Name(), nil
}

func analyzeProfile(profilePath string) ([]*ProfileLine, error) {
	file, err := os.Open(profilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	var lines []*ProfileLine
	scanner := bufio.NewScanner(file)
	
	// Skip mode line
	if scanner.Scan() {
		mode := scanner.Text()
		if verbose {
			log.Printf("Coverage mode: %s", mode)
		}
	}
	
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		
		// Parse coverage line
		// Format: name.com/package/file.go:start.col,end.col count statements
		var path string
		var startLine, startCol, endLine, endCol int
		var count int64
		var statements int
		
		parts := strings.Fields(line)
		if len(parts) != 3 {
			continue
		}
		
		// Parse location
		locParts := strings.Split(parts[0], ":")
		if len(locParts) != 2 {
			continue
		}
		path = locParts[0]
		
		// Parse line ranges
		rangeParts := strings.Split(locParts[1], ",")
		if len(rangeParts) != 2 {
			continue
		}
		
		fmt.Sscanf(rangeParts[0], "%d.%d", &startLine, &startCol)
		fmt.Sscanf(rangeParts[1], "%d.%d", &endLine, &endCol)
		fmt.Sscanf(parts[1], "%d", &statements)
		fmt.Sscanf(parts[2], "%d", &count)
		
		// Extract package from path
		pkg := extractPackage(path)
		
		for line := startLine; line <= endLine; line++ {
			lines = append(lines, &ProfileLine{
				File:     path,
				Line:     line,
				Covered:  count > 0,
				HitCount: count,
				Function: fmt.Sprintf("%s:%d", pkg, line),
			})
		}
	}
	
	return lines, scanner.Err()
}

func findHotspots(lines []*ProfileLine) []Hotspot {
	// Group by file and line
	lineMap := make(map[string]*ProfileLine)
	
	for _, line := range lines {
		key := fmt.Sprintf("%s:%d", line.File, line.Line)
		if existing, ok := lineMap[key]; ok {
			// Keep the one with highest hit count
			if line.HitCount > existing.HitCount {
				lineMap[key] = line
			}
		} else {
			lineMap[key] = line
		}
	}
	
	// Find uncovered lines with high hit counts
	var hotspots []Hotspot
	
	for _, line := range lineMap {
		if !line.Covered && line.HitCount >= int64(threshold) {
			hotspot := Hotspot{
				Package:  extractPackage(line.File),
				File:     line.File,
				Line:     line.Line,
				HitCount: line.HitCount,
				Function: line.Function,
			}
			
			// Try to get source line
			if sourceLine, err := getSourceLine(line.File, line.Line); err == nil {
				hotspot.SourceLine = sourceLine
			}
			
			hotspots = append(hotspots, hotspot)
		}
	}
	
	return hotspots
}

func extractPackage(path string) string {
	// Extract package from file path
	parts := strings.Split(path, "/")
	if len(parts) > 1 {
		// Remove filename
		parts = parts[:len(parts)-1]
		return strings.Join(parts, "/")
	}
	return path
}

func getSourceLine(file string, line int) (string, error) {
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()
	
	scanner := bufio.NewScanner(f)
	currentLine := 0
	
	for scanner.Scan() {
		currentLine++
		if currentLine == line {
			return strings.TrimSpace(scanner.Text()), nil
		}
	}
	
	return "", fmt.Errorf("line %d not found", line)
}

func outputText(out *os.File, hotspots []Hotspot) {
	fmt.Fprintf(out, "=== Coverage Hotspots ===\n")
	fmt.Fprintf(out, "Most hit but uncovered lines:\n\n")
	
	for i, hotspot := range hotspots {
		fmt.Fprintf(out, "#%d: %s:%d (hit %d times)\n", i+1, hotspot.File, hotspot.Line, hotspot.HitCount)
		if hotspot.SourceLine != "" {
			fmt.Fprintf(out, "    %s\n", hotspot.SourceLine)
		}
		fmt.Fprintf(out, "\n")
	}
	
	fmt.Fprintf(out, "Total hotspots found: %d\n", len(hotspots))
}

func outputJSON(out *os.File, hotspots []Hotspot) {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	encoder.Encode(map[string]interface{}{
		"hotspots": hotspots,
		"count":    len(hotspots),
	})
}