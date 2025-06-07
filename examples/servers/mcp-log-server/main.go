package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/modelcontextprotocol"
)

func main() {
	server := mcp.NewServer(mcp.ServerOptions{
		Name:        "log-server",
		Version:     "1.0.0",
		Description: "A server for log file analysis and monitoring operations",
	})

	// Add log file reading tool
	server.AddTool(modelcontextprotocol.Tool{
		Name:        "read_log",
		Description: "Reads and filters log file content",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the log file",
				},
				"lines": map[string]interface{}{
					"type":        "integer",
					"description": "Number of lines to read (default: 100)",
					"default":     100,
				},
				"from_end": map[string]interface{}{
					"type":        "boolean",
					"description": "Read from end of file (tail behavior, default: true)",
					"default":     true,
				},
			},
			"required": []string{"file_path"},
		},
	}, func(ctx context.Context, req modelcontextprotocol.CallToolRequest) (modelcontextprotocol.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		filePath := args["file_path"].(string)

		lines := 100
		if linesVal, ok := args["lines"]; ok {
			if linesFloat, ok := linesVal.(float64); ok {
				lines = int(linesFloat)
			}
		}

		fromEnd := true
		if fromEndVal, ok := args["from_end"]; ok {
			if fromEndBool, ok := fromEndVal.(bool); ok {
				fromEnd = fromEndBool
			}
		}

		content, err := readLogFile(filePath, lines, fromEnd)
		if err != nil {
			return modelcontextprotocol.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Error reading log file: %v", err),
					},
				},
			}, nil
		}

		return modelcontextprotocol.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": content,
				},
			},
		}, nil
	})

	// Add log filtering tool
	server.AddTool(modelcontextprotocol.Tool{
		Name:        "filter_logs",
		Description: "Filters log entries by pattern, level, or time range",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the log file",
				},
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "Regular expression pattern to match",
				},
				"level": map[string]interface{}{
					"type":        "string",
					"description": "Log level to filter (ERROR, WARN, INFO, DEBUG)",
				},
				"since": map[string]interface{}{
					"type":        "string",
					"description": "Start time (e.g., '2024-01-01 12:00:00')",
				},
				"until": map[string]interface{}{
					"type":        "string",
					"description": "End time (e.g., '2024-01-01 13:00:00')",
				},
			},
			"required": []string{"file_path"},
		},
	}, func(ctx context.Context, req modelcontextprotocol.CallToolRequest) (modelcontextprotocol.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		filePath := args["file_path"].(string)

		var pattern string
		if patternVal, ok := args["pattern"]; ok {
			pattern = patternVal.(string)
		}

		var level string
		if levelVal, ok := args["level"]; ok {
			level = levelVal.(string)
		}

		var since string
		if sinceVal, ok := args["since"]; ok {
			since = sinceVal.(string)
		}

		var until string
		if untilVal, ok := args["until"]; ok {
			until = untilVal.(string)
		}

		result, err := filterLogs(filePath, pattern, level, since, until)
		if err != nil {
			return modelcontextprotocol.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Error filtering logs: %v", err),
					},
				},
			}, nil
		}

		return modelcontextprotocol.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": result,
				},
			},
		}, nil
	})

	// Add log statistics tool
	server.AddTool(modelcontextprotocol.Tool{
		Name:        "log_stats",
		Description: "Generates statistics about log file content",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the log file",
				},
				"include_levels": map[string]interface{}{
					"type":        "boolean",
					"description": "Include log level statistics (default: true)",
					"default":     true,
				},
			},
			"required": []string{"file_path"},
		},
	}, func(ctx context.Context, req modelcontextprotocol.CallToolRequest) (modelcontextprotocol.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		filePath := args["file_path"].(string)

		includeLevels := true
		if includeLevelsVal, ok := args["include_levels"]; ok {
			if includeLevelsBool, ok := includeLevelsVal.(bool); ok {
				includeLevels = includeLevelsBool
			}
		}

		stats, err := generateLogStats(filePath, includeLevels)
		if err != nil {
			return modelcontextprotocol.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Error generating log statistics: %v", err),
					},
				},
			}, nil
		}

		return modelcontextprotocol.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": stats,
				},
			},
		}, nil
	})

	// Add log monitoring tool
	server.AddTool(modelcontextprotocol.Tool{
		Name:        "monitor_logs",
		Description: "Monitors log file for new entries (like tail -f)",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the log file",
				},
				"duration": map[string]interface{}{
					"type":        "integer",
					"description": "Monitoring duration in seconds (default: 10)",
					"default":     10,
				},
			},
			"required": []string{"file_path"},
		},
	}, func(ctx context.Context, req modelcontextprotocol.CallToolRequest) (modelcontextprotocol.CallToolResult, error) {
		args := req.Params.Arguments.(map[string]interface{})
		filePath := args["file_path"].(string)

		duration := 10
		if durationVal, ok := args["duration"]; ok {
			if durationFloat, ok := durationVal.(float64); ok {
				duration = int(durationFloat)
			}
		}

		result, err := monitorLogFile(filePath, duration)
		if err != nil {
			return modelcontextprotocol.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Error monitoring log file: %v", err),
					},
				},
			}, nil
		}

		return modelcontextprotocol.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": result,
				},
			},
		}, nil
	})

	if err := server.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func readLogFile(filePath string, lines int, fromEnd bool) (string, error) {
	// Simple implementation - in production would use more efficient methods
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	allLines := strings.Split(string(content), "\n")

	if fromEnd {
		start := len(allLines) - lines
		if start < 0 {
			start = 0
		}
		return strings.Join(allLines[start:], "\n"), nil
	} else {
		end := lines
		if end > len(allLines) {
			end = len(allLines)
		}
		return strings.Join(allLines[:end], "\n"), nil
	}
}

func filterLogs(filePath, pattern, level, since, until string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	var filtered []string

	var regex *regexp.Regexp
	if pattern != "" {
		regex, err = regexp.Compile(pattern)
		if err != nil {
			return "", fmt.Errorf("invalid regex pattern: %v", err)
		}
	}

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Apply pattern filter
		if regex != nil && !regex.MatchString(line) {
			continue
		}

		// Apply level filter
		if level != "" && !strings.Contains(strings.ToUpper(line), strings.ToUpper(level)) {
			continue
		}

		// Time filtering would require parsing log timestamps - simplified here

		filtered = append(filtered, line)
	}

	return strings.Join(filtered, "\n"), nil
}

func generateLogStats(filePath string, includeLevels bool) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	totalLines := len(lines) - 1 // Subtract 1 for final empty line

	stats := fmt.Sprintf("Log File Statistics for: %s\n", filepath.Base(filePath))
	stats += fmt.Sprintf("========================================\n")
	stats += fmt.Sprintf("Total Lines: %d\n", totalLines)
	stats += fmt.Sprintf("File Size: %d bytes\n", len(content))

	if includeLevels {
		levels := map[string]int{
			"ERROR": 0,
			"WARN":  0,
			"INFO":  0,
			"DEBUG": 0,
		}

		for _, line := range lines {
			upperLine := strings.ToUpper(line)
			for level := range levels {
				if strings.Contains(upperLine, level) {
					levels[level]++
				}
			}
		}

		stats += "\nLog Levels:\n"
		for level, count := range levels {
			if count > 0 {
				percentage := float64(count) / float64(totalLines) * 100
				stats += fmt.Sprintf("  %s: %d (%.1f%%)\n", level, count, percentage)
			}
		}
	}

	return stats, nil
}

func monitorLogFile(filePath string, duration int) (string, error) {
	// Simple monitoring simulation - in production would use file watchers
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to stat file: %v", err)
	}

	initialSize := fileInfo.Size()

	time.Sleep(time.Duration(duration) * time.Second)

	fileInfo, err = os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to stat file after monitoring: %v", err)
	}

	finalSize := fileInfo.Size()

	result := fmt.Sprintf("Monitoring Results for: %s\n", filepath.Base(filePath))
	result += fmt.Sprintf("Duration: %d seconds\n", duration)
	result += fmt.Sprintf("Initial Size: %d bytes\n", initialSize)
	result += fmt.Sprintf("Final Size: %d bytes\n", finalSize)
	result += fmt.Sprintf("Size Change: %+d bytes\n", finalSize-initialSize)

	if finalSize > initialSize {
		// Read the new content
		content, err := os.ReadFile(filePath)
		if err == nil {
			allLines := strings.Split(string(content), "\n")
			// Estimate new lines (simplified)
			result += "\nNew activity detected!\n"
		}
	} else {
		result += "\nNo new activity detected.\n"
	}

	return result, nil
}
