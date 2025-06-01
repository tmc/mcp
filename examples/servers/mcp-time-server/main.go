package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/tmc/mcp"
)

const (
	ServerName    = "mcp-time-server"
	ServerVersion = "0.1.0"
)

type TimeResult struct {
	Timezone string `json:"timezone"`
	DateTime string `json:"datetime"`
	IsDST    bool   `json:"is_dst"`
}

type TimeConversionResult struct {
	Source         TimeResult `json:"source"`
	Target         TimeResult `json:"target"`
	TimeDifference string     `json:"time_difference"`
}

func main() {
	// Redirect logs to stderr to keep stdout clean for the protocol
	log.SetOutput(os.Stderr)
	log.Println("Starting MCP Time Server...")

	// Create a context that can be canceled
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Get local timezone
	localTz := time.Now().Location().String()

	// Create a server
	server := mcp.NewServer(ServerName, ServerVersion,
		mcp.WithServerInstructions("A time and timezone conversion server for MCP"),
	)

	// Register time tools
	registerTimeTools(server, localTz)

	// Serve via stdio
	log.Println("Starting protocol server via stdio...")
	if err := server.Serve(ctx, nil); err != nil {
		if err != context.Canceled {
			log.Fatalf("Error serving: %v", err)
		}
		log.Println("Server terminated.")
	}
}

func registerTimeTools(server *mcp.Server, localTz string) {
	// Register get_current_time tool
	getCurrentTimeTool := mcp.Tool{
		Name:        "get_current_time",
		Description: "Get current time in a specific timezone",
		InputSchema: json.RawMessage(fmt.Sprintf(`{
			"type": "object",
			"properties": {
				"timezone": {
					"type": "string",
					"description": "IANA timezone name (e.g., 'America/New_York', 'Europe/London'). Use '%s' as local timezone if no timezone provided by the user."
				}
			},
			"required": ["timezone"]
		}`, localTz)),
	}

	server.RegisterTool(getCurrentTimeTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		timezone, ok := params["timezone"].(string)
		if !ok || timezone == "" {
			return nil, fmt.Errorf("timezone is required and must be a string")
		}

		result, err := getCurrentTime(timezone)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Error getting current time: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		resultJSON, _ := json.MarshalIndent(result, "", "  ")
		return &mcp.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": string(resultJSON),
				},
			},
		}, nil
	})

	// Register convert_time tool
	convertTimeTool := mcp.Tool{
		Name:        "convert_time",
		Description: "Convert time between timezones",
		InputSchema: json.RawMessage(fmt.Sprintf(`{
			"type": "object",
			"properties": {
				"source_timezone": {
					"type": "string",
					"description": "Source IANA timezone name (e.g., 'America/New_York', 'Europe/London'). Use '%s' as local timezone if no source timezone provided by the user."
				},
				"time": {
					"type": "string",
					"description": "Time to convert in 24-hour format (HH:MM)"
				},
				"target_timezone": {
					"type": "string",
					"description": "Target IANA timezone name (e.g., 'Asia/Tokyo', 'America/San_Francisco'). Use '%s' as local timezone if no target timezone provided by the user."
				}
			},
			"required": ["source_timezone", "time", "target_timezone"]
		}`, localTz, localTz)),
	}

	server.RegisterTool(convertTimeTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		sourceTz, ok := params["source_timezone"].(string)
		if !ok || sourceTz == "" {
			return nil, fmt.Errorf("source_timezone is required and must be a string")
		}

		timeStr, ok := params["time"].(string)
		if !ok || timeStr == "" {
			return nil, fmt.Errorf("time is required and must be a string")
		}

		targetTz, ok := params["target_timezone"].(string)
		if !ok || targetTz == "" {
			return nil, fmt.Errorf("target_timezone is required and must be a string")
		}

		result, err := convertTime(sourceTz, timeStr, targetTz)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Error converting time: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		resultJSON, _ := json.MarshalIndent(result, "", "  ")
		return &mcp.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": string(resultJSON),
				},
			},
		}, nil
	})

	log.Println("Registered time tools: get_current_time, convert_time")
}

func getCurrentTime(timezone string) (TimeResult, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return TimeResult{}, fmt.Errorf("invalid timezone: %v", err)
	}

	now := time.Now().In(loc)
	
	// Check if DST is in effect
	isDST := isDaylightSaving(now)

	return TimeResult{
		Timezone: timezone,
		DateTime: now.Format(time.RFC3339),
		IsDST:    isDST,
	}, nil
}

func convertTime(sourceTz, timeStr, targetTz string) (TimeConversionResult, error) {
	sourceLoc, err := time.LoadLocation(sourceTz)
	if err != nil {
		return TimeConversionResult{}, fmt.Errorf("invalid source timezone: %v", err)
	}

	targetLoc, err := time.LoadLocation(targetTz)
	if err != nil {
		return TimeConversionResult{}, fmt.Errorf("invalid target timezone: %v", err)
	}

	// Parse time in HH:MM format
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return TimeConversionResult{}, fmt.Errorf("invalid time format. Expected HH:MM [24-hour format]")
	}

	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return TimeConversionResult{}, fmt.Errorf("invalid hour: %s", parts[0])
	}

	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return TimeConversionResult{}, fmt.Errorf("invalid minute: %s", parts[1])
	}

	// Create source time using today's date
	now := time.Now().In(sourceLoc)
	sourceTime := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, sourceLoc)

	// Convert to target timezone
	targetTime := sourceTime.In(targetLoc)

	// Calculate time difference
	_, sourceOffset := sourceTime.Zone()
	_, targetOffset := targetTime.Zone()
	diffSeconds := targetOffset - sourceOffset
	diffHours := float64(diffSeconds) / 3600.0

	var timeDiff string
	if diffHours == float64(int(diffHours)) {
		timeDiff = fmt.Sprintf("%+.1fh", diffHours)
	} else {
		timeDiff = fmt.Sprintf("%+.2fh", diffHours)
		timeDiff = strings.TrimSuffix(strings.TrimSuffix(timeDiff, "0"), ".") + "h"
	}

	return TimeConversionResult{
		Source: TimeResult{
			Timezone: sourceTz,
			DateTime: sourceTime.Format(time.RFC3339),
			IsDST:    isDaylightSaving(sourceTime),
		},
		Target: TimeResult{
			Timezone: targetTz,
			DateTime: targetTime.Format(time.RFC3339),
			IsDST:    isDaylightSaving(targetTime),
		},
		TimeDifference: timeDiff,
	}, nil
}

func isDaylightSaving(t time.Time) bool {
	// Get the time zone offset for January 1st (definitely not DST in northern hemisphere)
	jan1 := time.Date(t.Year(), 1, 1, 12, 0, 0, 0, t.Location())
	_, jan1Offset := jan1.Zone()
	
	// Get the current offset
	_, currentOffset := t.Zone()
	
	// If current offset is greater than January offset, we're likely in DST
	return currentOffset > jan1Offset
}