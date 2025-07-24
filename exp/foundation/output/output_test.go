package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestJSONFormatting(t *testing.T) {
	var buf bytes.Buffer
	
	config := Config{
		Format: FormatJSON,
		Writer: &buf,
		Pretty: true,
	}
	
	formatter, err := NewFormatter(config)
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}
	
	data := map[string]interface{}{
		"name":    "test",
		"value":   42,
		"enabled": true,
		"items":   []string{"a", "b", "c"},
	}
	
	if err := formatter.Format(data); err != nil {
		t.Fatalf("Failed to format data: %v", err)
	}
	
	// Verify JSON output
	var parsed map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}
	
	if parsed["name"] != "test" {
		t.Errorf("Expected name 'test', got %v", parsed["name"])
	}
	
	if parsed["value"] != float64(42) {
		t.Errorf("Expected value 42, got %v", parsed["value"])
	}
	
	if parsed["enabled"] != true {
		t.Errorf("Expected enabled true, got %v", parsed["enabled"])
	}
	
	// Check pretty formatting
	output := buf.String()
	if !strings.Contains(output, "\n") {
		t.Error("Expected pretty formatted JSON to contain newlines")
	}
}

func TestYAMLFormatting(t *testing.T) {
	var buf bytes.Buffer
	
	config := Config{
		Format: FormatYAML,
		Writer: &buf,
		Pretty: true,
	}
	
	formatter, err := NewFormatter(config)
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}
	
	data := map[string]interface{}{
		"name":    "test",
		"value":   42,
		"enabled": true,
		"nested": map[string]interface{}{
			"key": "value",
		},
	}
	
	if err := formatter.Format(data); err != nil {
		t.Fatalf("Failed to format data: %v", err)
	}
	
	// Verify YAML output
	var parsed map[string]interface{}
	if err := yaml.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("Failed to parse YAML output: %v", err)
	}
	
	if parsed["name"] != "test" {
		t.Errorf("Expected name 'test', got %v", parsed["name"])
	}
	
	if parsed["value"] != 42 {
		t.Errorf("Expected value 42, got %v", parsed["value"])
	}
	
	if parsed["enabled"] != true {
		t.Errorf("Expected enabled true, got %v", parsed["enabled"])
	}
}

func TestTableFormatting(t *testing.T) {
	var buf bytes.Buffer
	
	config := Config{
		Format: FormatTable,
		Writer: &buf,
		Table: TableConfig{
			Headers: true,
			Borders: true,
		},
	}
	
	formatter, err := NewFormatter(config)
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}
	
	// Test slice of structs
	type TestStruct struct {
		Name    string `json:"name"`
		Value   int    `json:"value"`
		Enabled bool   `json:"enabled"`
	}
	
	data := []TestStruct{
		{Name: "test1", Value: 42, Enabled: true},
		{Name: "test2", Value: 24, Enabled: false},
	}
	
	if err := formatter.Format(data); err != nil {
		t.Fatalf("Failed to format data: %v", err)
	}
	
	output := buf.String()
	
	// Check headers
	if !strings.Contains(output, "Name") {
		t.Error("Expected table to contain 'Name' header")
	}
	
	if !strings.Contains(output, "Value") {
		t.Error("Expected table to contain 'Value' header")
	}
	
	if !strings.Contains(output, "Enabled") {
		t.Error("Expected table to contain 'Enabled' header")
	}
	
	// Check data
	if !strings.Contains(output, "test1") {
		t.Error("Expected table to contain 'test1' data")
	}
	
	if !strings.Contains(output, "42") {
		t.Error("Expected table to contain '42' data")
	}
	
	if !strings.Contains(output, "true") {
		t.Error("Expected table to contain 'true' data")
	}
	
	// Check borders
	if !strings.Contains(output, "---") {
		t.Error("Expected table to contain separator line")
	}
}

func TestCSVFormatting(t *testing.T) {
	var buf bytes.Buffer
	
	config := Config{
		Format: FormatCSV,
		Writer: &buf,
		CSV: CSVConfig{
			Headers:   true,
			Separator: ',',
		},
	}
	
	formatter, err := NewFormatter(config)
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}
	
	// Test slice of maps
	data := []map[string]interface{}{
		{"name": "test1", "value": 42, "enabled": true},
		{"name": "test2", "value": 24, "enabled": false},
	}
	
	if err := formatter.Format(data); err != nil {
		t.Fatalf("Failed to format data: %v", err)
	}
	
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	
	if len(lines) < 2 {
		t.Fatalf("Expected at least 2 lines, got %d", len(lines))
	}
	
	// Check headers
	headerLine := lines[0]
	if !strings.Contains(headerLine, "name") {
		t.Error("Expected CSV to contain 'name' header")
	}
	
	if !strings.Contains(headerLine, "value") {
		t.Error("Expected CSV to contain 'value' header")
	}
	
	if !strings.Contains(headerLine, "enabled") {
		t.Error("Expected CSV to contain 'enabled' header")
	}
	
	// Check data
	dataLine := lines[1]
	if !strings.Contains(dataLine, "test1") {
		t.Error("Expected CSV to contain 'test1' data")
	}
	
	if !strings.Contains(dataLine, "42") {
		t.Error("Expected CSV to contain '42' data")
	}
}

func TestTextFormatting(t *testing.T) {
	var buf bytes.Buffer
	
	config := Config{
		Format: FormatText,
		Writer: &buf,
		Text: TextConfig{
			ShowNames: true,
		},
	}
	
	formatter, err := NewFormatter(config)
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}
	
	data := map[string]interface{}{
		"name":    "test",
		"value":   42,
		"enabled": true,
		"items":   []interface{}{"a", "b", "c"},
	}
	
	if err := formatter.Format(data); err != nil {
		t.Fatalf("Failed to format data: %v", err)
	}
	
	output := buf.String()
	
	// Check that field names are shown
	if !strings.Contains(output, "name:") {
		t.Error("Expected text to contain 'name:' label")
	}
	
	if !strings.Contains(output, "value:") {
		t.Error("Expected text to contain 'value:' label")
	}
	
	if !strings.Contains(output, "enabled:") {
		t.Error("Expected text to contain 'enabled:' label")
	}
	
	// Check data
	if !strings.Contains(output, "test") {
		t.Error("Expected text to contain 'test' data")
	}
	
	if !strings.Contains(output, "42") {
		t.Error("Expected text to contain '42' data")
	}
	
	if !strings.Contains(output, "true") {
		t.Error("Expected text to contain 'true' data")
	}
}

func TestTableSorting(t *testing.T) {
	var buf bytes.Buffer
	
	config := Config{
		Format: FormatTable,
		Writer: &buf,
		Table: TableConfig{
			Headers:   true,
			SortBy:    "Name",
			SortOrder: "asc",
		},
	}
	
	formatter, err := NewFormatter(config)
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}
	
	// Test with unsorted data
	data := []map[string]interface{}{
		{"Name": "charlie", "Value": 3},
		{"Name": "alice", "Value": 1},
		{"Name": "bob", "Value": 2},
	}
	
	if err := formatter.Format(data); err != nil {
		t.Fatalf("Failed to format data: %v", err)
	}
	
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	
	// Find data lines (skip header and separator)
	var dataLines []string
	for _, line := range lines {
		if !strings.Contains(line, "Name") && !strings.Contains(line, "---") && strings.TrimSpace(line) != "" {
			dataLines = append(dataLines, line)
		}
	}
	
	if len(dataLines) < 3 {
		t.Fatalf("Expected at least 3 data lines, got %d", len(dataLines))
	}
	
	// Check sorting order
	if !strings.Contains(dataLines[0], "alice") {
		t.Error("Expected first row to contain 'alice'")
	}
	
	if !strings.Contains(dataLines[1], "bob") {
		t.Error("Expected second row to contain 'bob'")
	}
	
	if !strings.Contains(dataLines[2], "charlie") {
		t.Error("Expected third row to contain 'charlie'")
	}
}

func TestTableAlignment(t *testing.T) {
	var buf bytes.Buffer
	
	config := Config{
		Format: FormatTable,
		Writer: &buf,
		Table: TableConfig{
			Headers: true,
			Align:   []string{"left", "right", "center"},
		},
	}
	
	formatter, err := NewFormatter(config)
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}
	
	data := []map[string]interface{}{
		{"Name": "test", "Value": 42, "Status": "ok"},
	}
	
	if err := formatter.Format(data); err != nil {
		t.Fatalf("Failed to format data: %v", err)
	}
	
	output := buf.String()
	
	// Basic check - alignment is hard to test precisely without knowing exact widths
	if !strings.Contains(output, "test") {
		t.Error("Expected table to contain 'test' data")
	}
	
	if !strings.Contains(output, "42") {
		t.Error("Expected table to contain '42' data")
	}
	
	if !strings.Contains(output, "ok") {
		t.Error("Expected table to contain 'ok' data")
	}
}

func TestDataToTableConversion(t *testing.T) {
	config := Config{
		Format: FormatTable,
		Writer: &bytes.Buffer{},
	}
	
	formatter, err := NewFormatter(config)
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}
	
	// Test struct
	type TestStruct struct {
		Name    string `json:"name"`
		Value   int    `json:"value"`
		Enabled bool   `json:"enabled"`
	}
	
	structData := TestStruct{Name: "test", Value: 42, Enabled: true}
	table, err := formatter.dataToTable(structData)
	if err != nil {
		t.Fatalf("Failed to convert struct to table: %v", err)
	}
	
	if len(table.Headers) != 3 {
		t.Errorf("Expected 3 headers, got %d", len(table.Headers))
	}
	
	if len(table.Rows) != 1 {
		t.Errorf("Expected 1 row, got %d", len(table.Rows))
	}
	
	// Test map
	mapData := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	
	table, err = formatter.dataToTable(mapData)
	if err != nil {
		t.Fatalf("Failed to convert map to table: %v", err)
	}
	
	if len(table.Headers) != 2 {
		t.Errorf("Expected 2 headers, got %d", len(table.Headers))
	}
	
	if len(table.Rows) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(table.Rows))
	}
	
	// Test slice of primitives
	sliceData := []int{1, 2, 3}
	table, err = formatter.dataToTable(sliceData)
	if err != nil {
		t.Fatalf("Failed to convert slice to table: %v", err)
	}
	
	if len(table.Headers) != 1 {
		t.Errorf("Expected 1 header, got %d", len(table.Headers))
	}
	
	if len(table.Rows) != 3 {
		t.Errorf("Expected 3 rows, got %d", len(table.Rows))
	}
}

func TestFormatFieldValue(t *testing.T) {
	config := Config{
		Format: FormatTable,
		Writer: &bytes.Buffer{},
	}
	
	formatter, err := NewFormatter(config)
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}
	
	tests := []struct {
		input    interface{}
		expected string
	}{
		{"test", "test"},
		{42, "42"},
		{true, "true"},
		{false, "false"},
		{3.14, "3.14"},
		{[]int{1, 2, 3}, "[1, 2, 3]"},
		{map[string]int{"a": 1}, "{a: 1}"},
		{time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), "2023-01-01T00:00:00Z"},
	}
	
	for _, test := range tests {
		result := formatter.formatFieldValue(reflect.ValueOf(test.input))
		if result != test.expected {
			t.Errorf("Expected %s for input %v, got %s", test.expected, test.input, result)
		}
	}
}

func TestDefaultFormatterFunctions(t *testing.T) {
	data := map[string]interface{}{
		"name":  "test",
		"value": 42,
	}
	
	// Test default functions don't panic
	if err := FormatJSON(data); err != nil {
		t.Errorf("FormatJSON failed: %v", err)
	}
	
	if err := FormatYAML(data); err != nil {
		t.Errorf("FormatYAML failed: %v", err)
	}
	
	if err := FormatTable(data); err != nil {
		t.Errorf("FormatTable failed: %v", err)
	}
	
	if err := FormatCSV(data); err != nil {
		t.Errorf("FormatCSV failed: %v", err)
	}
}

func TestUnsupportedFormat(t *testing.T) {
	config := Config{
		Format: Format("unsupported"),
		Writer: &bytes.Buffer{},
	}
	
	formatter, err := NewFormatter(config)
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}
	
	if err := formatter.Format(map[string]interface{}{}); err == nil {
		t.Error("Expected error for unsupported format")
	}
}

func TestEmptyData(t *testing.T) {
	var buf bytes.Buffer
	
	config := Config{
		Format: FormatTable,
		Writer: &buf,
	}
	
	formatter, err := NewFormatter(config)
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}
	
	// Test empty slice
	if err := formatter.Format([]interface{}{}); err != nil {
		t.Errorf("Failed to format empty slice: %v", err)
	}
	
	// Test nil
	if err := formatter.Format(nil); err != nil {
		t.Errorf("Failed to format nil: %v", err)
	}
}

func BenchmarkJSONFormatting(b *testing.B) {
	var buf bytes.Buffer
	
	config := Config{
		Format: FormatJSON,
		Writer: &buf,
	}
	
	formatter, err := NewFormatter(config)
	if err != nil {
		b.Fatalf("Failed to create formatter: %v", err)
	}
	
	data := map[string]interface{}{
		"name":    "test",
		"value":   42,
		"enabled": true,
		"items":   []string{"a", "b", "c"},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		if err := formatter.Format(data); err != nil {
			b.Fatalf("Failed to format data: %v", err)
		}
	}
}

func BenchmarkTableFormatting(b *testing.B) {
	var buf bytes.Buffer
	
	config := Config{
		Format: FormatTable,
		Writer: &buf,
	}
	
	formatter, err := NewFormatter(config)
	if err != nil {
		b.Fatalf("Failed to create formatter: %v", err)
	}
	
	data := []map[string]interface{}{
		{"name": "test1", "value": 42, "enabled": true},
		{"name": "test2", "value": 24, "enabled": false},
		{"name": "test3", "value": 13, "enabled": true},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		if err := formatter.Format(data); err != nil {
			b.Fatalf("Failed to format data: %v", err)
		}
	}
}

func BenchmarkDataToTable(b *testing.B) {
	config := Config{
		Format: FormatTable,
		Writer: &bytes.Buffer{},
	}
	
	formatter, err := NewFormatter(config)
	if err != nil {
		b.Fatalf("Failed to create formatter: %v", err)
	}
	
	data := []map[string]interface{}{
		{"name": "test1", "value": 42, "enabled": true},
		{"name": "test2", "value": 24, "enabled": false},
		{"name": "test3", "value": 13, "enabled": true},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := formatter.dataToTable(data); err != nil {
			b.Fatalf("Failed to convert data to table: %v", err)
		}
	}
}