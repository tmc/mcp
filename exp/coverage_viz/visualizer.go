package coverage_viz

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"sort"
	"strings"
)

// CoverageVisualizer provides web-based coverage visualization
type CoverageVisualizer struct {
	coverage *CoverageData
	server   *http.Server
}

// CoverageData represents the coverage information
type CoverageData struct {
	Files map[string]*FileCoverage
	Tests map[string]*TestImpact
}

// FileCoverage represents coverage for a single file
type FileCoverage struct {
	Path         string
	Lines        map[int]*LineCoverage
	TotalLines   int
	CoveredLines int
}

// LineCoverage represents coverage for a single line
type LineCoverage struct {
	Number   int
	Covered  bool
	HitCount int
	Tests    []string
}

// TestImpact represents the impact of a single test
type TestImpact struct {
	Name           string
	CoveredLines   int
	UniqueCoverage int
	Files          []string
}

// NewVisualizer creates a new coverage visualizer
func NewVisualizer(coverage *CoverageData) *CoverageVisualizer {
	return &CoverageVisualizer{
		coverage: coverage,
	}
}

// Serve starts the web server for visualization
func (v *CoverageVisualizer) Serve(port int) error {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/coverage/files", v.handleFiles)
	mux.HandleFunc("/api/coverage/file/", v.handleFile)
	mux.HandleFunc("/api/coverage/tests", v.handleTests)
	mux.HandleFunc("/api/coverage/test/", v.handleTest)

	// Web UI
	mux.HandleFunc("/", v.handleIndex)
	mux.HandleFunc("/file/", v.handleFileView)

	v.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	fmt.Printf("Coverage visualization server starting on http://localhost:%d\n", port)
	return v.server.ListenAndServe()
}

// handleIndex serves the main page
func (v *CoverageVisualizer) handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>MCP Coverage Visualization</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .coverage-bar { 
            width: 200px; 
            height: 20px; 
            background: #f0f0f0; 
            border: 1px solid #ccc; 
            position: relative; 
        }
        .coverage-fill {
            height: 100%;
            background: #4CAF50;
            position: absolute;
            left: 0;
            top: 0;
        }
        .file-list { margin-top: 20px; }
        .file-item { 
            margin: 10px 0; 
            padding: 10px; 
            border: 1px solid #ddd; 
            border-radius: 4px;
        }
        .covered { background-color: #e8f5e9; }
        .uncovered { background-color: #ffebee; }
        .partial { background-color: #fff3e0; }
    </style>
</head>
<body>
    <h1>MCP Coverage Report</h1>
    <div class="coverage-summary">
        <h2>Overall Coverage: {{.OverallCoverage}}%</h2>
        <div class="coverage-bar">
            <div class="coverage-fill" style="width: {{.OverallCoverage}}%"></div>
        </div>
    </div>
    
    <div class="file-list">
        <h2>Files</h2>
        {{range .Files}}
        <div class="file-item">
            <a href="/file/{{.Path}}">{{.Path}}</a>
            <div class="coverage-bar">
                <div class="coverage-fill" style="width: {{.Coverage}}%"></div>
            </div>
            <span>{{.Coverage}}% ({{.CoveredLines}}/{{.TotalLines}} lines)</span>
        </div>
        {{end}}
    </div>
    
    <div class="test-list">
        <h2>Tests</h2>
        {{range .Tests}}
        <div class="test-item">
            <a href="/test/{{.Name}}">{{.Name}}</a>
            <span>Covers {{.CoveredLines}} lines</span>
        </div>
        {{end}}
    </div>
</body>
</html>
`

	// Calculate overall coverage
	totalLines := 0
	coveredLines := 0
	for _, file := range v.coverage.Files {
		totalLines += file.TotalLines
		coveredLines += file.CoveredLines
	}

	overallCoverage := 0.0
	if totalLines > 0 {
		overallCoverage = float64(coveredLines) / float64(totalLines) * 100
	}

	// Prepare file data
	var files []struct {
		Path         string
		Coverage     float64
		CoveredLines int
		TotalLines   int
	}

	for path, file := range v.coverage.Files {
		coverage := 0.0
		if file.TotalLines > 0 {
			coverage = float64(file.CoveredLines) / float64(file.TotalLines) * 100
		}
		files = append(files, struct {
			Path         string
			Coverage     float64
			CoveredLines int
			TotalLines   int
		}{
			Path:         path,
			Coverage:     coverage,
			CoveredLines: file.CoveredLines,
			TotalLines:   file.TotalLines,
		})
	}

	// Sort files by path
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	// Prepare test data
	var tests []struct {
		Name         string
		CoveredLines int
	}

	for name, test := range v.coverage.Tests {
		tests = append(tests, struct {
			Name         string
			CoveredLines int
		}{
			Name:         name,
			CoveredLines: test.CoveredLines,
		})
	}

	// Sort tests by name
	sort.Slice(tests, func(i, j int) bool {
		return tests[i].Name < tests[j].Name
	})

	// Execute template
	t := template.Must(template.New("index").Parse(tmpl))
	data := struct {
		OverallCoverage float64
		Files           interface{}
		Tests           interface{}
	}{
		OverallCoverage: overallCoverage,
		Files:           files,
		Tests:           tests,
	}

	t.Execute(w, data)
}

// handleFileView displays coverage for a specific file
func (v *CoverageVisualizer) handleFileView(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/file/")
	file, ok := v.coverage.Files[path]
	if !ok {
		http.NotFound(w, r)
		return
	}

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>{{.Path}} - Coverage</title>
    <style>
        body { font-family: monospace; margin: 20px; }
        .line { margin: 2px 0; }
        .line-number { 
            display: inline-block; 
            width: 40px; 
            text-align: right; 
            padding-right: 10px;
            color: #666;
        }
        .covered { background-color: #c8e6c9; }
        .uncovered { background-color: #ffcdd2; }
        .coverage-info { 
            position: absolute; 
            right: 20px; 
            color: #666;
            font-size: 12px;
        }
        pre { margin: 0; display: inline; }
    </style>
</head>
<body>
    <h1>{{.Path}}</h1>
    <h2>Coverage: {{.Coverage}}% ({{.CoveredLines}}/{{.TotalLines}} lines)</h2>
    
    <div class="source-code">
        {{range .Lines}}
        <div class="line {{if .Covered}}covered{{else}}uncovered{{end}}">
            <span class="line-number">{{.Number}}</span>
            <pre>{{.Content}}</pre>
            {{if .Covered}}
            <span class="coverage-info">Tests: {{.TestCount}} | Hits: {{.HitCount}}</span>
            {{end}}
        </div>
        {{end}}
    </div>
    
    <div class="test-mapping">
        <h3>Tests covering this file:</h3>
        <ul>
        {{range .Tests}}
            <li>{{.}}</li>
        {{end}}
        </ul>
    </div>
</body>
</html>
`

	// Calculate file coverage
	coverage := 0.0
	if file.TotalLines > 0 {
		coverage = float64(file.CoveredLines) / float64(file.TotalLines) * 100
	}

	// Prepare line data
	var lines []struct {
		Number    int
		Content   string
		Covered   bool
		HitCount  int
		TestCount int
	}

	// Mock source code content
	sourceLines := []string{
		"package example",
		"",
		"import (",
		`    "fmt"`,
		`    "strings"`,
		")",
		"",
		"func ParseMessage(msg string) (string, error) {",
		"    if msg == \"\" {",
		"        return \"\", fmt.Errorf(\"empty message\")",
		"    }",
		"    ",
		"    parts := strings.Split(msg, \":\")",
		"    if len(parts) < 2 {",
		"        return \"\", fmt.Errorf(\"invalid format\")",
		"    }",
		"    ",
		"    switch parts[0] {",
		"    case \"info\":",
		"        return fmt.Sprintf(\"INFO: %s\", parts[1]), nil",
		"    case \"warn\":",
		"        return fmt.Sprintf(\"WARNING: %s\", parts[1]), nil",
		"    case \"error\":",
		"        return fmt.Sprintf(\"ERROR: %s\", parts[1]), nil",
		"    default:",
		"        return fmt.Sprintf(\"UNKNOWN: %s\", parts[1]), nil",
		"    }",
		"}",
	}

	for i, content := range sourceLines {
		lineNum := i + 1
		lineCov, exists := file.Lines[lineNum]
		if exists {
			lines = append(lines, struct {
				Number    int
				Content   string
				Covered   bool
				HitCount  int
				TestCount int
			}{
				Number:    lineNum,
				Content:   content,
				Covered:   lineCov.Covered,
				HitCount:  lineCov.HitCount,
				TestCount: len(lineCov.Tests),
			})
		} else {
			lines = append(lines, struct {
				Number    int
				Content   string
				Covered   bool
				HitCount  int
				TestCount int
			}{
				Number:  lineNum,
				Content: content,
				Covered: true, // Non-executable lines
			})
		}
	}

	// Get tests covering this file
	var tests []string
	testMap := make(map[string]bool)
	for _, line := range file.Lines {
		for _, test := range line.Tests {
			testMap[test] = true
		}
	}
	for test := range testMap {
		tests = append(tests, test)
	}
	sort.Strings(tests)

	// Execute template
	t := template.Must(template.New("file").Parse(tmpl))
	data := struct {
		Path         string
		Coverage     float64
		CoveredLines int
		TotalLines   int
		Lines        interface{}
		Tests        []string
	}{
		Path:         path,
		Coverage:     coverage,
		CoveredLines: file.CoveredLines,
		TotalLines:   file.TotalLines,
		Lines:        lines,
		Tests:        tests,
	}

	t.Execute(w, data)
}

// API handlers
func (v *CoverageVisualizer) handleFiles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Implementation would return JSON data
	fmt.Fprint(w, `{"files": []}`)
}

func (v *CoverageVisualizer) handleFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Implementation would return JSON data for specific file
	fmt.Fprint(w, `{"coverage": {}}`)
}

func (v *CoverageVisualizer) handleTests(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Implementation would return JSON data
	fmt.Fprint(w, `{"tests": []}`)
}

func (v *CoverageVisualizer) handleTest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Implementation would return JSON data for specific test
	fmt.Fprint(w, `{"impact": {}}`)
}

// GenerateReport creates a text-based coverage report
func (v *CoverageVisualizer) GenerateReport(w io.Writer) {
	// Calculate overall coverage
	totalLines := 0
	coveredLines := 0
	for _, file := range v.coverage.Files {
		totalLines += file.TotalLines
		coveredLines += file.CoveredLines
	}

	overallCoverage := 0.0
	if totalLines > 0 {
		overallCoverage = float64(coveredLines) / float64(totalLines) * 100
	}

	fmt.Fprintf(w, "MCP Coverage Report\n")
	fmt.Fprintf(w, "==================\n\n")
	fmt.Fprintf(w, "Overall Coverage: %.1f%% (%d/%d lines)\n\n", overallCoverage, coveredLines, totalLines)

	fmt.Fprintf(w, "File Coverage:\n")
	fmt.Fprintf(w, "--------------\n")

	// Sort files for consistent output
	var paths []string
	for path := range v.coverage.Files {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		file := v.coverage.Files[path]
		coverage := 0.0
		if file.TotalLines > 0 {
			coverage = float64(file.CoveredLines) / float64(file.TotalLines) * 100
		}
		fmt.Fprintf(w, "%-40s %6.1f%% (%d/%d)\n", path, coverage, file.CoveredLines, file.TotalLines)
	}

	fmt.Fprintf(w, "\nTest Impact:\n")
	fmt.Fprintf(w, "------------\n")

	// Sort tests for consistent output
	var testNames []string
	for name := range v.coverage.Tests {
		testNames = append(testNames, name)
	}
	sort.Strings(testNames)

	for _, name := range testNames {
		test := v.coverage.Tests[name]
		fmt.Fprintf(w, "%-40s Covers %d lines\n", name, test.CoveredLines)
	}
}
