// Package output provides a unified output formatting library for all MCP tools.
// It supports JSON, YAML, Table, and CSV formats with color support and follows
// the Russ Cox coding style for consistency across the MCP toolkit.
package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Format represents an output format.
type Format string

const (
	// FormatJSON outputs data in JSON format
	FormatJSON Format = "json"
	
	// FormatYAML outputs data in YAML format
	FormatYAML Format = "yaml"
	
	// FormatTable outputs data in table format
	FormatTable Format = "table"
	
	// FormatCSV outputs data in CSV format
	FormatCSV Format = "csv"
	
	// FormatText outputs data in plain text format
	FormatText Format = "text"
)

// Config represents output configuration.
type Config struct {
	// Output format
	Format Format `json:"format" yaml:"format"`
	
	// Output destination
	Writer io.Writer `json:"-" yaml:"-"`
	
	// Enable color output
	Color bool `json:"color" yaml:"color"`
	
	// Pretty print JSON/YAML
	Pretty bool `json:"pretty" yaml:"pretty"`
	
	// Table configuration
	Table TableConfig `json:"table" yaml:"table"`
	
	// CSV configuration
	CSV CSVConfig `json:"csv" yaml:"csv"`
	
	// Text configuration
	Text TextConfig `json:"text" yaml:"text"`
}

// TableConfig represents table formatting configuration.
type TableConfig struct {
	// Show table headers
	Headers bool `json:"headers" yaml:"headers"`
	
	// Table borders
	Borders bool `json:"borders" yaml:"borders"`
	
	// Column separator
	Separator string `json:"separator" yaml:"separator"`
	
	// Maximum column width
	MaxWidth int `json:"max_width" yaml:"max_width"`
	
	// Column alignment
	Align []string `json:"align" yaml:"align"`
	
	// Sort by column
	SortBy string `json:"sort_by" yaml:"sort_by"`
	
	// Sort order (asc/desc)
	SortOrder string `json:"sort_order" yaml:"sort_order"`
}

// CSVConfig represents CSV formatting configuration.
type CSVConfig struct {
	// Field separator
	Separator rune `json:"separator" yaml:"separator"`
	
	// Include headers
	Headers bool `json:"headers" yaml:"headers"`
	
	// Quote character
	Quote rune `json:"quote" yaml:"quote"`
	
	// Escape character
	Escape rune `json:"escape" yaml:"escape"`
}

// TextConfig represents text formatting configuration.
type TextConfig struct {
	// Template for formatting
	Template string `json:"template" yaml:"template"`
	
	// Field separator
	Separator string `json:"separator" yaml:"separator"`
	
	// Show field names
	ShowNames bool `json:"show_names" yaml:"show_names"`
}

// ColorConfig represents color configuration.
type ColorConfig struct {
	// Enable color output
	Enabled bool `json:"enabled" yaml:"enabled"`
	
	// Color scheme
	Scheme ColorScheme `json:"scheme" yaml:"scheme"`
}

// ColorScheme represents color scheme configuration.
type ColorScheme struct {
	// Primary color
	Primary string `json:"primary" yaml:"primary"`
	
	// Secondary color
	Secondary string `json:"secondary" yaml:"secondary"`
	
	// Success color
	Success string `json:"success" yaml:"success"`
	
	// Warning color
	Warning string `json:"warning" yaml:"warning"`
	
	// Error color
	Error string `json:"error" yaml:"error"`
	
	// Info color
	Info string `json:"info" yaml:"info"`
	
	// Muted color
	Muted string `json:"muted" yaml:"muted"`
}

// Formatter handles output formatting.
type Formatter struct {
	config Config
	colors *ColorProvider
}

// NewFormatter creates a new output formatter.
func NewFormatter(config Config) (*Formatter, error) {
	if config.Writer == nil {
		config.Writer = os.Stdout
	}
	
	// Set default values
	if config.Format == "" {
		config.Format = FormatJSON
	}
	
	if config.Table.Separator == "" {
		config.Table.Separator = "  "
	}
	
	if config.CSV.Separator == 0 {
		config.CSV.Separator = ','
	}
	
	if config.CSV.Quote == 0 {
		config.CSV.Quote = '"'
	}
	
	if config.Text.Separator == "" {
		config.Text.Separator = "\t"
	}
	
	f := &Formatter{
		config: config,
		colors: NewColorProvider(config.Color),
	}
	
	return f, nil
}

// Format formats the given data according to the configured format.
func (f *Formatter) Format(data interface{}) error {
	switch f.config.Format {
	case FormatJSON:
		return f.formatJSON(data)
	case FormatYAML:
		return f.formatYAML(data)
	case FormatTable:
		return f.formatTable(data)
	case FormatCSV:
		return f.formatCSV(data)
	case FormatText:
		return f.formatText(data)
	default:
		return fmt.Errorf("unsupported format: %s", f.config.Format)
	}
}

// formatJSON formats data as JSON.
func (f *Formatter) formatJSON(data interface{}) error {
	encoder := json.NewEncoder(f.config.Writer)
	
	if f.config.Pretty {
		encoder.SetIndent("", "  ")
	}
	
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	
	return nil
}

// formatYAML formats data as YAML.
func (f *Formatter) formatYAML(data interface{}) error {
	encoder := yaml.NewEncoder(f.config.Writer)
	defer encoder.Close()
	
	if f.config.Pretty {
		encoder.SetIndent(2)
	}
	
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode YAML: %w", err)
	}
	
	return nil
}

// formatTable formats data as a table.
func (f *Formatter) formatTable(data interface{}) error {
	table, err := f.dataToTable(data)
	if err != nil {
		return fmt.Errorf("failed to convert data to table: %w", err)
	}
	
	if len(table.Rows) == 0 {
		return nil
	}
	
	// Sort table if requested
	if f.config.Table.SortBy != "" {
		if err := f.sortTable(table); err != nil {
			return fmt.Errorf("failed to sort table: %w", err)
		}
	}
	
	// Calculate column widths
	widths := f.calculateColumnWidths(table)
	
	// Print headers
	if f.config.Table.Headers && len(table.Headers) > 0 {
		if err := f.printTableRow(table.Headers, widths, true); err != nil {
			return fmt.Errorf("failed to print table headers: %w", err)
		}
		
		if f.config.Table.Borders {
			if err := f.printTableSeparator(widths); err != nil {
				return fmt.Errorf("failed to print table separator: %w", err)
			}
		}
	}
	
	// Print rows
	for _, row := range table.Rows {
		if err := f.printTableRow(row, widths, false); err != nil {
			return fmt.Errorf("failed to print table row: %w", err)
		}
	}
	
	return nil
}

// formatCSV formats data as CSV.
func (f *Formatter) formatCSV(data interface{}) error {
	table, err := f.dataToTable(data)
	if err != nil {
		return fmt.Errorf("failed to convert data to table: %w", err)
	}
	
	writer := csv.NewWriter(f.config.Writer)
	defer writer.Flush()
	
	// Configure CSV writer
	writer.Comma = f.config.CSV.Separator
	writer.Quote = f.config.CSV.Quote
	
	// Write headers
	if f.config.CSV.Headers && len(table.Headers) > 0 {
		if err := writer.Write(table.Headers); err != nil {
			return fmt.Errorf("failed to write CSV headers: %w", err)
		}
	}
	
	// Write rows
	for _, row := range table.Rows {
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}
	
	return nil
}

// formatText formats data as plain text.
func (f *Formatter) formatText(data interface{}) error {
	if f.config.Text.Template != "" {
		return f.formatTextTemplate(data)
	}
	
	return f.formatTextDefault(data)
}

// formatTextTemplate formats data using a template.
func (f *Formatter) formatTextTemplate(data interface{}) error {
	// Simple template implementation
	// In a real implementation, you'd use text/template
	template := f.config.Text.Template
	
	// Replace placeholders with actual values
	if reflect.TypeOf(data).Kind() == reflect.Map {
		if m, ok := data.(map[string]interface{}); ok {
			for key, value := range m {
				placeholder := fmt.Sprintf("{{.%s}}", key)
				template = strings.ReplaceAll(template, placeholder, fmt.Sprintf("%v", value))
			}
		}
	}
	
	_, err := fmt.Fprint(f.config.Writer, template)
	return err
}

// formatTextDefault formats data as plain text with default formatting.
func (f *Formatter) formatTextDefault(data interface{}) error {
	return f.formatValue(data, 0)
}

// formatValue formats a value recursively.
func (f *Formatter) formatValue(value interface{}, depth int) error {
	indent := strings.Repeat("  ", depth)
	
	switch v := value.(type) {
	case nil:
		_, err := fmt.Fprintf(f.config.Writer, "%s<nil>\n", indent)
		return err
	case bool:
		_, err := fmt.Fprintf(f.config.Writer, "%s%t\n", indent, v)
		return err
	case int, int8, int16, int32, int64:
		_, err := fmt.Fprintf(f.config.Writer, "%s%d\n", indent, v)
		return err
	case uint, uint8, uint16, uint32, uint64:
		_, err := fmt.Fprintf(f.config.Writer, "%s%d\n", indent, v)
		return err
	case float32, float64:
		_, err := fmt.Fprintf(f.config.Writer, "%s%f\n", indent, v)
		return err
	case string:
		_, err := fmt.Fprintf(f.config.Writer, "%s%s\n", indent, v)
		return err
	case []interface{}:
		for i, item := range v {
			if f.config.Text.ShowNames {
				fmt.Fprintf(f.config.Writer, "%s[%d]:\n", indent, i)
			}
			if err := f.formatValue(item, depth+1); err != nil {
				return err
			}
		}
		return nil
	case map[string]interface{}:
		for key, val := range v {
			if f.config.Text.ShowNames {
				fmt.Fprintf(f.config.Writer, "%s%s:\n", indent, key)
			}
			if err := f.formatValue(val, depth+1); err != nil {
				return err
			}
		}
		return nil
	default:
		// Use reflection for structs
		return f.formatStruct(value, depth)
	}
}

// formatStruct formats a struct using reflection.
func (f *Formatter) formatStruct(value interface{}, depth int) error {
	v := reflect.ValueOf(value)
	t := reflect.TypeOf(value)
	
	// Handle pointers
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			_, err := fmt.Fprintf(f.config.Writer, "%s<nil>\n", strings.Repeat("  ", depth))
			return err
		}
		v = v.Elem()
		t = t.Elem()
	}
	
	if v.Kind() != reflect.Struct {
		_, err := fmt.Fprintf(f.config.Writer, "%s%v\n", strings.Repeat("  ", depth), value)
		return err
	}
	
	indent := strings.Repeat("  ", depth)
	
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		
		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}
		
		if f.config.Text.ShowNames {
			fmt.Fprintf(f.config.Writer, "%s%s:\n", indent, fieldType.Name)
		}
		
		if err := f.formatValue(field.Interface(), depth+1); err != nil {
			return err
		}
	}
	
	return nil
}

// Table represents a table structure.
type Table struct {
	Headers []string
	Rows    [][]string
}

// dataToTable converts data to a table structure.
func (f *Formatter) dataToTable(data interface{}) (*Table, error) {
	v := reflect.ValueOf(data)
	t := reflect.TypeOf(data)
	
	// Handle pointers
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return &Table{}, nil
		}
		v = v.Elem()
		t = t.Elem()
	}
	
	switch v.Kind() {
	case reflect.Slice:
		return f.sliceToTable(v, t)
	case reflect.Map:
		return f.mapToTable(v, t)
	case reflect.Struct:
		return f.structToTable(v, t)
	default:
		// Single value
		return &Table{
			Headers: []string{"Value"},
			Rows:    [][]string{{fmt.Sprintf("%v", data)}},
		}, nil
	}
}

// sliceToTable converts a slice to a table.
func (f *Formatter) sliceToTable(v reflect.Value, t reflect.Type) (*Table, error) {
	if v.Len() == 0 {
		return &Table{}, nil
	}
	
	// Get element type
	elemType := t.Elem()
	
	// Handle slice of structs
	if elemType.Kind() == reflect.Struct {
		return f.structSliceToTable(v, elemType)
	}
	
	// Handle slice of maps
	if elemType.Kind() == reflect.Map {
		return f.mapSliceToTable(v)
	}
	
	// Handle slice of primitives
	table := &Table{
		Headers: []string{"Value"},
		Rows:    make([][]string, v.Len()),
	}
	
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)
		table.Rows[i] = []string{fmt.Sprintf("%v", item.Interface())}
	}
	
	return table, nil
}

// structSliceToTable converts a slice of structs to a table.
func (f *Formatter) structSliceToTable(v reflect.Value, elemType reflect.Type) (*Table, error) {
	if v.Len() == 0 {
		return &Table{}, nil
	}
	
	// Get headers from struct fields
	var headers []string
	for i := 0; i < elemType.NumField(); i++ {
		field := elemType.Field(i)
		if field.IsExported() {
			headers = append(headers, field.Name)
		}
	}
	
	// Create table
	table := &Table{
		Headers: headers,
		Rows:    make([][]string, v.Len()),
	}
	
	// Fill rows
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)
		row := make([]string, len(headers))
		
		fieldIndex := 0
		for j := 0; j < elemType.NumField(); j++ {
			field := elemType.Field(j)
			if field.IsExported() {
				fieldValue := item.Field(j)
				row[fieldIndex] = f.formatFieldValue(fieldValue)
				fieldIndex++
			}
		}
		
		table.Rows[i] = row
	}
	
	return table, nil
}

// mapSliceToTable converts a slice of maps to a table.
func (f *Formatter) mapSliceToTable(v reflect.Value) (*Table, error) {
	if v.Len() == 0 {
		return &Table{}, nil
	}
	
	// Collect all unique keys
	keySet := make(map[string]bool)
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)
		if item.Kind() == reflect.Map {
			for _, key := range item.MapKeys() {
				keySet[fmt.Sprintf("%v", key.Interface())] = true
			}
		}
	}
	
	// Sort keys for consistent output
	var headers []string
	for key := range keySet {
		headers = append(headers, key)
	}
	sort.Strings(headers)
	
	// Create table
	table := &Table{
		Headers: headers,
		Rows:    make([][]string, v.Len()),
	}
	
	// Fill rows
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)
		row := make([]string, len(headers))
		
		if item.Kind() == reflect.Map {
			for j, header := range headers {
				key := reflect.ValueOf(header)
				value := item.MapIndex(key)
				if value.IsValid() {
					row[j] = fmt.Sprintf("%v", value.Interface())
				} else {
					row[j] = ""
				}
			}
		}
		
		table.Rows[i] = row
	}
	
	return table, nil
}

// mapToTable converts a map to a table.
func (f *Formatter) mapToTable(v reflect.Value, t reflect.Type) (*Table, error) {
	if v.Len() == 0 {
		return &Table{}, nil
	}
	
	table := &Table{
		Headers: []string{"Key", "Value"},
		Rows:    make([][]string, v.Len()),
	}
	
	keys := v.MapKeys()
	for i, key := range keys {
		value := v.MapIndex(key)
		table.Rows[i] = []string{
			fmt.Sprintf("%v", key.Interface()),
			fmt.Sprintf("%v", value.Interface()),
		}
	}
	
	return table, nil
}

// structToTable converts a struct to a table.
func (f *Formatter) structToTable(v reflect.Value, t reflect.Type) (*Table, error) {
	var headers []string
	var row []string
	
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		
		if fieldType.IsExported() {
			headers = append(headers, fieldType.Name)
			row = append(row, f.formatFieldValue(field))
		}
	}
	
	return &Table{
		Headers: headers,
		Rows:    [][]string{row},
	}, nil
}

// formatFieldValue formats a field value for table display.
func (f *Formatter) formatFieldValue(v reflect.Value) string {
	if !v.IsValid() {
		return ""
	}
	
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
	case reflect.Ptr:
		if v.IsNil() {
			return "<nil>"
		}
		return f.formatFieldValue(v.Elem())
	case reflect.Slice:
		if v.Len() == 0 {
			return "[]"
		}
		var items []string
		for i := 0; i < v.Len(); i++ {
			items = append(items, f.formatFieldValue(v.Index(i)))
		}
		return "[" + strings.Join(items, ", ") + "]"
	case reflect.Map:
		if v.Len() == 0 {
			return "{}"
		}
		var items []string
		for _, key := range v.MapKeys() {
			value := v.MapIndex(key)
			items = append(items, fmt.Sprintf("%v: %v", key.Interface(), f.formatFieldValue(value)))
		}
		return "{" + strings.Join(items, ", ") + "}"
	case reflect.Struct:
		if v.Type() == reflect.TypeOf(time.Time{}) {
			return v.Interface().(time.Time).Format(time.RFC3339)
		}
		return fmt.Sprintf("%v", v.Interface())
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

// calculateColumnWidths calculates column widths for table formatting.
func (f *Formatter) calculateColumnWidths(table *Table) []int {
	if len(table.Headers) == 0 {
		return nil
	}
	
	widths := make([]int, len(table.Headers))
	
	// Calculate width based on headers
	for i, header := range table.Headers {
		widths[i] = len(header)
	}
	
	// Calculate width based on rows
	for _, row := range table.Rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	
	// Apply maximum width limit
	if f.config.Table.MaxWidth > 0 {
		for i := range widths {
			if widths[i] > f.config.Table.MaxWidth {
				widths[i] = f.config.Table.MaxWidth
			}
		}
	}
	
	return widths
}

// printTableRow prints a table row.
func (f *Formatter) printTableRow(row []string, widths []int, isHeader bool) error {
	var parts []string
	
	for i, cell := range row {
		width := widths[i]
		
		// Truncate if too long
		if len(cell) > width {
			cell = cell[:width-3] + "..."
		}
		
		// Apply alignment
		align := "left"
		if i < len(f.config.Table.Align) {
			align = f.config.Table.Align[i]
		}
		
		switch align {
		case "right":
			cell = fmt.Sprintf("%*s", width, cell)
		case "center":
			padding := width - len(cell)
			leftPad := padding / 2
			rightPad := padding - leftPad
			cell = strings.Repeat(" ", leftPad) + cell + strings.Repeat(" ", rightPad)
		default: // left
			cell = fmt.Sprintf("%-*s", width, cell)
		}
		
		// Apply color
		if f.config.Color && isHeader {
			cell = f.colors.Primary(cell)
		}
		
		parts = append(parts, cell)
	}
	
	line := strings.Join(parts, f.config.Table.Separator)
	_, err := fmt.Fprintln(f.config.Writer, line)
	return err
}

// printTableSeparator prints a table separator line.
func (f *Formatter) printTableSeparator(widths []int) error {
	var parts []string
	
	for _, width := range widths {
		parts = append(parts, strings.Repeat("-", width))
	}
	
	line := strings.Join(parts, f.config.Table.Separator)
	_, err := fmt.Fprintln(f.config.Writer, line)
	return err
}

// sortTable sorts a table by the specified column.
func (f *Formatter) sortTable(table *Table) error {
	if f.config.Table.SortBy == "" {
		return nil
	}
	
	// Find column index
	var columnIndex int = -1
	for i, header := range table.Headers {
		if header == f.config.Table.SortBy {
			columnIndex = i
			break
		}
	}
	
	if columnIndex == -1 {
		return fmt.Errorf("column %s not found", f.config.Table.SortBy)
	}
	
	// Sort rows
	sort.Slice(table.Rows, func(i, j int) bool {
		if columnIndex >= len(table.Rows[i]) || columnIndex >= len(table.Rows[j]) {
			return false
		}
		
		a := table.Rows[i][columnIndex]
		b := table.Rows[j][columnIndex]
		
		if f.config.Table.SortOrder == "desc" {
			return a > b
		}
		return a < b
	})
	
	return nil
}

// Default formatter instance
var defaultFormatter *Formatter

// SetDefault sets the default formatter.
func SetDefault(formatter *Formatter) {
	defaultFormatter = formatter
}

// GetDefault returns the default formatter.
func GetDefault() *Formatter {
	if defaultFormatter == nil {
		config := Config{
			Format: FormatJSON,
			Writer: os.Stdout,
			Pretty: true,
			Color:  true,
		}
		
		var err error
		defaultFormatter, err = NewFormatter(config)
		if err != nil {
			panic(fmt.Sprintf("failed to create default formatter: %v", err))
		}
	}
	
	return defaultFormatter
}

// Format formats data using the default formatter.
func Format(data interface{}) error {
	return GetDefault().Format(data)
}

// FormatJSON formats data as JSON using the default formatter.
func FormatJSON(data interface{}) error {
	formatter := GetDefault()
	formatter.config.Format = FormatJSON
	return formatter.Format(data)
}

// FormatYAML formats data as YAML using the default formatter.
func FormatYAML(data interface{}) error {
	formatter := GetDefault()
	formatter.config.Format = FormatYAML
	return formatter.Format(data)
}

// FormatTable formats data as a table using the default formatter.
func FormatTable(data interface{}) error {
	formatter := GetDefault()
	formatter.config.Format = FormatTable
	return formatter.Format(data)
}

// FormatCSV formats data as CSV using the default formatter.
func FormatCSV(data interface{}) error {
	formatter := GetDefault()
	formatter.config.Format = FormatCSV
	return formatter.Format(data)
}