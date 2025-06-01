package testcallgraph

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// Stitcher connects test scripts to the programs they execute
type Stitcher struct {
	// Map of test files to programs they execute
	TestToProgramMap map[string][]string
	
	// Map of programs to their main functions  
	ProgramToMainMap map[string]*MainFunc
}

// MainFunc represents a main function in a program
type MainFunc struct {
	Package  string
	File     string
	Line     int
	FullPath string
}

// NewStitcher creates a new stitcher
func NewStitcher() *Stitcher {
	return &Stitcher{
		TestToProgramMap: make(map[string][]string),
		ProgramToMainMap: make(map[string]*MainFunc),
	}
}

// AnalyzeScriptTest analyzes a scripttest file to find programs it executes
func (s *Stitcher) AnalyzeScriptTest(testFile string) error {
	file, err := os.Open(testFile)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	programs := []string{}
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Look for exec commands
		if strings.HasPrefix(line, "exec ") {
			cmd := strings.TrimPrefix(line, "exec ")
			// Extract the program name (first word)
			parts := strings.Fields(cmd)
			if len(parts) > 0 {
				prog := parts[0]
				// Remove any path prefix to get just the program name
				prog = filepath.Base(prog)
				programs = append(programs, prog)
			}
		}
	}
	
	s.TestToProgramMap[testFile] = programs
	return scanner.Err()
}

// FindProgramMain finds the main function of a program
func (s *Stitcher) FindProgramMain(programName string) (*MainFunc, error) {
	// Try to find the source code for the program
	// First, check if it's an MCP tool in the cmd directory

	possiblePaths := []string{
		fmt.Sprintf("cmd/%s/main.go", programName),
		fmt.Sprintf("../cmd/%s/main.go", programName),
		fmt.Sprintf("../../cmd/%s/main.go", programName),
		fmt.Sprintf("../../../cmd/%s/main.go", programName),
		fmt.Sprintf("../../../../cmd/%s/main.go", programName),
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			// Parse the file to find the main function
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if err == nil {
				for _, decl := range file.Decls {
					if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "main" {
						pos := fset.Position(fn.Pos())
						absPath, _ := filepath.Abs(path)
						return &MainFunc{
							Package:  file.Name.Name,
							File:     absPath,
							Line:     pos.Line,
							FullPath: fmt.Sprintf("%s:%d", absPath, pos.Line),
						}, nil
					}
				}
			}
		}
	}

	// If not found, return a placeholder
	return &MainFunc{
		Package:  "main",
		File:     fmt.Sprintf("cmd/%s/main.go", programName),
		Line:     1,
		FullPath: fmt.Sprintf("cmd/%s/main.go:1", programName),
	}, nil
}

// StitchTestToPrograms connects a test to all programs it executes
func (s *Stitcher) StitchTestToPrograms(testFile string) ([]*TestProgramConnection, error) {
	// First analyze the test file
	if err := s.AnalyzeScriptTest(testFile); err != nil {
		return nil, err
	}
	
	programs := s.TestToProgramMap[testFile]
	connections := []*TestProgramConnection{}
	
	for _, prog := range programs {
		// Find the main function for this program
		mainFunc, err := s.FindProgramMain(prog)
		if err != nil {
			// Skip if we can't find it
			continue
		}
		
		conn := &TestProgramConnection{
			TestFile:    testFile,
			TestLine:    0, // Would need to track line numbers in AnalyzeScriptTest
			Program:     prog,
			MainFunc:    mainFunc,
			CommandLine: fmt.Sprintf("exec %s", prog),
		}
		connections = append(connections, conn)
	}
	
	return connections, nil
}

// TestProgramConnection represents a connection from a test to a program
type TestProgramConnection struct {
	TestFile    string
	TestLine    int
	Program     string
	MainFunc    *MainFunc
	CommandLine string
}

// String returns a string representation of the connection
func (c *TestProgramConnection) String() string {
	return fmt.Sprintf("%s -> %s (%s)", c.TestFile, c.Program, c.MainFunc.FullPath)
}