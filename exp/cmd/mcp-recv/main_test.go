package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestProcessTypeOnly(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "request",
			input: `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`,
			want:  "request\n",
		},
		{
			name:  "response",
			input: `{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`,
			want:  "response\n",
		},
		{
			name:  "notification",
			input: `{"jsonrpc":"2.0","method":"tools/changed","params":{"added":[]}}`,
			want:  "notification\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var msg jsonrpc2Message
			if err := json.Unmarshal([]byte(tt.input), &msg); err != nil {
				t.Fatalf("Failed to unmarshal test input: %v", err)
			}

			var buf bytes.Buffer
			err := processTypeOnly(&buf, &msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("processTypeOnly() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got := buf.String(); got != tt.want {
				t.Errorf("processTypeOnly() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProcessMethodOnly(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "request with method",
			input: `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`,
			want:  "tools/list\n",
		},
		{
			name:    "response without method",
			input:   `{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`,
			wantErr: true,
		},
		{
			name:  "notification with method",
			input: `{"jsonrpc":"2.0","method":"tools/changed","params":{"added":[]}}`,
			want:  "tools/changed\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var msg jsonrpc2Message
			if err := json.Unmarshal([]byte(tt.input), &msg); err != nil {
				t.Fatalf("Failed to unmarshal test input: %v", err)
			}

			var buf bytes.Buffer
			err := processMethodOnly(&buf, &msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("processMethodOnly() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && buf.String() != tt.want {
				t.Errorf("processMethodOnly() = %q, want %q", buf.String(), tt.want)
			}
		})
	}
}

func TestExtractField(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		path    string
		want    interface{}
		wantErr bool
	}{
		{
			name:    "simple field",
			json:    `{"name":"test","value":42}`,
			path:    "name",
			want:    "test",
			wantErr: false,
		},
		{
			name:    "nested field",
			json:    `{"outer":{"inner":"value"}}`,
			path:    "outer.inner",
			want:    "value",
			wantErr: false,
		},
		{
			name:    "array index",
			json:    `{"items":[10,20,30]}`,
			path:    "items.1",
			want:    float64(20),
			wantErr: false,
		},
		{
			name:    "deep nesting",
			json:    `{"a":{"b":{"c":{"d":"deep"}}}}`,
			path:    "a.b.c.d",
			want:    "deep",
			wantErr: false,
		},
		{
			name:    "field not found",
			json:    `{"name":"test"}`,
			path:    "missing",
			wantErr: true,
		},
		{
			name:    "invalid array index",
			json:    `{"items":[10,20,30]}`,
			path:    "items.xyz",
			wantErr: true,
		},
		{
			name:    "array index out of bounds",
			json:    `{"items":[10,20,30]}`,
			path:    "items.5",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractField(json.RawMessage(tt.json), tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractField() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRun(t *testing.T) {
	// Create test input files
	requestFile, err := os.CreateTemp("", "mcp-recv-request-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(requestFile.Name())

	responseFile, err := os.CreateTemp("", "mcp-recv-response-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(responseFile.Name())

	// Write test data
	requestJSON := `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`
	if _, err := requestFile.Write([]byte(requestJSON)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	requestFile.Close()

	responseJSON := `{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"echo","description":"Echo input"}]}}`
	if _, err := responseFile.Write([]byte(responseJSON)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	responseFile.Close()

	// Create output file
	outputFile, err := os.CreateTemp("", "mcp-recv-output-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp output file: %v", err)
	}
	defer os.Remove(outputFile.Name())
	outputFile.Close()

	tests := []struct {
		name       string
		inputFile  string
		input      string
		outputFile string
		format     bool
		extract    string
		typeOnly   bool
		methodOnly bool
		wantErr    bool
		contains   string
	}{
		{
			name:      "request from file",
			inputFile: requestFile.Name(),
			format:    false,
			wantErr:   false,
			contains:  "Method: tools/list",
		},
		{
			name:      "response from file with formatting",
			inputFile: responseFile.Name(),
			format:    true,
			wantErr:   false,
			contains:  "Result:",
		},
		{
			name:      "extract tool name",
			inputFile: responseFile.Name(),
			extract:   "result.tools.0.name",
			wantErr:   false,
			contains:  "echo",
		},
		{
			name:       "write to output file",
			inputFile:  requestFile.Name(),
			outputFile: outputFile.Name(),
			wantErr:    false,
			contains:   "Method: tools/list",
		},
		{
			name:      "type only",
			inputFile: requestFile.Name(),
			typeOnly:  true,
			wantErr:   false,
			contains:  "request",
		},
		{
			name:       "method only",
			inputFile:  requestFile.Name(),
			methodOnly: true,
			wantErr:    false,
			contains:   "tools/list",
		},
		{
			name:     "extract from string input",
			input:    responseJSON,
			extract:  "result.tools.0.description",
			wantErr:  false,
			contains: "Echo input",
		},
		{
			name:      "non-existent input file",
			inputFile: filepath.Join(os.TempDir(), "this-file-does-not-exist.json"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			var in io.Reader

			if tt.input != "" {
				in = strings.NewReader(tt.input)
				// Save stdin
				oldStdin := os.Stdin
				// Create a pipe
				r, w, _ := os.Pipe()
				// Set stdin to the read end
				os.Stdin = r
				// Write test input to the write end
				w.Write([]byte(tt.input))
				w.Close()
				// Defer restoring stdin
				defer func() { os.Stdin = oldStdin }()

				err := run("", tt.outputFile, tt.format, tt.extract, tt.typeOnly, tt.methodOnly, false)
				if (err != nil) != tt.wantErr {
					t.Errorf("run() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				err := run(tt.inputFile, tt.outputFile, tt.format, tt.extract, tt.typeOnly, tt.methodOnly, false)
				if (err != nil) != tt.wantErr {
					t.Errorf("run() error = %v, wantErr %v", err, tt.wantErr)
				}
			}

			// For non-error cases, check the output
			if !tt.wantErr && tt.outputFile != "" {
				// Read from output file
				data, err := os.ReadFile(tt.outputFile)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}
				if !strings.Contains(string(data), tt.contains) {
					t.Errorf("Output file missing expected content. Want %q in %q", tt.contains, string(data))
				}
			}
		})
	}
}

// jsonrpc2Message is a simplified version of jsonrpc2.Message for testing
type jsonrpc2Message struct {
	Method string           `json:"method,omitempty"`
	ID     *json.RawMessage `json:"id,omitempty"`
	Bytes  *json.RawMessage
}

func (m *jsonrpc2Message) UnmarshalJSON(data []byte) error {
	type msg jsonrpc2Message
	var rm msg
	if err := json.Unmarshal(data, &rm); err != nil {
		return err
	}
	*m = jsonrpc2Message(rm)
	raw := json.RawMessage(data)
	m.Bytes = &raw
	return nil
}
