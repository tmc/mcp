package sourcereflect_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tmc/mcp/exp/sourcereflect"
)

func TestAnalyzeSourceHints(t *testing.T) {
	// Create test file
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "test.go")

	testCode := `
package main

import (
	"os"
	"net/http"
	"io/ioutil"
)

func ReadOnlyFunc(data string) string {
	return data + "_processed"
}

func NetworkFunc(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func DiskWriteFunc(filename string, data []byte) error {
	return os.WriteFile(filename, data, 0644)
}

func DiskReadFunc(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

func StateChangeFunc(data *[]string) {
	*data = append(*data, "new")
}
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	tests := []struct {
		funcName string
		expected struct {
			readOnly    bool
			destructive bool
			openWorld   bool
		}
	}{
		{
			funcName: "ReadOnlyFunc",
			expected: struct {
				readOnly    bool
				destructive bool
				openWorld   bool
			}{readOnly: true, destructive: false, openWorld: false},
		},
		{
			funcName: "NetworkFunc",
			expected: struct {
				readOnly    bool
				destructive bool
				openWorld   bool
			}{readOnly: false, destructive: false, openWorld: true},
		},
		{
			funcName: "DiskWriteFunc",
			expected: struct {
				readOnly    bool
				destructive bool
				openWorld   bool
			}{readOnly: false, destructive: true, openWorld: false},
		},
		{
			funcName: "StateChangeFunc",
			expected: struct {
				readOnly    bool
				destructive bool
				openWorld   bool
			}{readOnly: false, destructive: false, openWorld: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.funcName, func(t *testing.T) {
			hints, err := sourcereflect.AnalyzeSourceHints(testFile, tt.funcName)
			if err != nil {
				t.Fatalf("Failed to analyze source hints: %v", err)
			}

			if hints.ReadOnlyHint != nil && *hints.ReadOnlyHint != tt.expected.readOnly {
				t.Errorf("Expected readOnly=%v, got %v", tt.expected.readOnly, *hints.ReadOnlyHint)
			}

			if hints.DestructiveHint != nil && *hints.DestructiveHint != tt.expected.destructive {
				t.Errorf("Expected destructive=%v, got %v", tt.expected.destructive, *hints.DestructiveHint)
			}

			if hints.OpenWorldHint != nil && *hints.OpenWorldHint != tt.expected.openWorld {
				t.Errorf("Expected openWorld=%v, got %v", tt.expected.openWorld, *hints.OpenWorldHint)
			}
		})
	}
}
