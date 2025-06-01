package mcpcoverageviz

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

//go:embed static/*
var staticFiles embed.FS

//go:embed templates/*
var templateFiles embed.FS

// WebServer serves the coverage visualization
type WebServer struct {
	viz      *CoverageVisualization
	addr     string
	template *template.Template
}

// NewWebServer creates a new web server
func NewWebServer(viz *CoverageVisualization, addr string) (*WebServer, error) {
	// Parse templates
	tmpl, err := template.ParseFS(templateFiles, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}
	
	return &WebServer{
		viz:      viz,
		addr:     addr,
		template: tmpl,
	}, nil
}

// Serve starts the web server
func (s *WebServer) Serve() error {
	mux := http.NewServeMux()
	
	// Static files
	mux.Handle("/static/", http.FileServer(http.FS(staticFiles)))
	
	// API endpoints
	mux.HandleFunc("/api/coverage", s.handleCoverageAPI)
	mux.HandleFunc("/api/files/", s.handleFileAPI)
	mux.HandleFunc("/api/tests", s.handleTestsAPI)
	mux.HandleFunc("/api/sessions", s.handleSessionsAPI)
	
	// HTML pages
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/file/", s.handleFile)
	mux.HandleFunc("/test/", s.handleTest)
	mux.HandleFunc("/timeline", s.handleTimeline)
	
	fmt.Printf("Starting server on %s\n", s.addr)
	return http.ListenAndServe(s.addr, mux)
}

// handleIndex serves the main dashboard
func (s *WebServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Title   string
		Summary Summary
		Sessions []TestSession
	}{
		Title:   "MCP Coverage Visualization",
		Summary: s.viz.Summary,
		Sessions: s.viz.Sessions,
	}
	
	if err := s.template.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleFile serves the file coverage view
func (s *WebServer) handleFile(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/file/")
	
	file, ok := s.viz.Files[path]
	if !ok {
		http.NotFound(w, r)
		return
	}
	
	data := struct {
		Title string
		File  *FileData
	}{
		Title: fmt.Sprintf("Coverage: %s", path),
		File:  file,
	}
	
	if err := s.template.ExecuteTemplate(w, "file.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleTest serves the test detail view
func (s *WebServer) handleTest(w http.ResponseWriter, r *http.Request) {
	testID := strings.TrimPrefix(r.URL.Path, "/test/")
	
	// Find test
	var test *TestExecution
	for _, session := range s.viz.Sessions {
		for _, t := range session.Tests {
			if t.TestName == testID {
				test = &t
				break
			}
		}
	}
	
	if test == nil {
		http.NotFound(w, r)
		return
	}
	
	data := struct {
		Title string
		Test  *TestExecution
	}{
		Title: fmt.Sprintf("Test: %s", testID),
		Test:  test,
	}
	
	if err := s.template.ExecuteTemplate(w, "test.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleTimeline serves the timeline view
func (s *WebServer) handleTimeline(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Title    string
		Sessions []TestSession
	}{
		Title:    "Test Timeline",
		Sessions: s.viz.Sessions,
	}
	
	if err := s.template.ExecuteTemplate(w, "timeline.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// API handlers

func (s *WebServer) handleCoverageAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.viz)
}

func (s *WebServer) handleFileAPI(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/files/")
	
	file, ok := s.viz.Files[path]
	if !ok {
		http.NotFound(w, r)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(file)
}

func (s *WebServer) handleTestsAPI(w http.ResponseWriter, r *http.Request) {
	var tests []TestExecution
	for _, session := range s.viz.Sessions {
		tests = append(tests, session.Tests...)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tests)
}

func (s *WebServer) handleSessionsAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.viz.Sessions)
}