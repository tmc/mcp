package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/tmc/mcp"
)

const (
	ServerName    = "mcp-system-server"
	ServerVersion = "0.1.0"
)

type SystemInfo struct {
	OS           string            `json:"os"`
	Architecture string            `json:"architecture"`
	NumCPU       int               `json:"num_cpu"`
	GoVersion    string            `json:"go_version"`
	Hostname     string            `json:"hostname"`
	Username     string            `json:"username"`
	HomeDir      string            `json:"home_dir"`
	WorkingDir   string            `json:"working_dir"`
	Environment  map[string]string `json:"environment"`
	Uptime       string            `json:"uptime,omitempty"`
}

type ProcessInfo struct {
	PID         int               `json:"pid"`
	Name        string            `json:"name"`
	Command     string            `json:"command,omitempty"`
	CPU         string            `json:"cpu_usage,omitempty"`
	Memory      string            `json:"memory_usage,omitempty"`
	Status      string            `json:"status,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
}

type DiskInfo struct {
	Path        string `json:"path"`
	Size        string `json:"size,omitempty"`
	Used        string `json:"used,omitempty"`
	Available   string `json:"available,omitempty"`
	UsePercent  string `json:"use_percent,omitempty"`
	Filesystem  string `json:"filesystem,omitempty"`
	Mountpoint  string `json:"mountpoint,omitempty"`
}

type NetworkInfo struct {
	Interface   string   `json:"interface"`
	IPAddresses []string `json:"ip_addresses,omitempty"`
	Status      string   `json:"status,omitempty"`
}

func main() {
	log.SetOutput(os.Stderr)
	log.Println("Starting MCP System Server...")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	server := mcp.NewServer(ServerName, ServerVersion,
		mcp.WithServerInstructions("A system information server providing details about the host system"),
	)

	registerSystemTools(server)

	log.Println("Starting protocol server via stdio...")
	if err := server.Serve(ctx, nil); err != nil {
		if err != context.Canceled {
			log.Fatalf("Error serving: %v", err)
		}
		log.Println("Server terminated.")
	}
}

func registerSystemTools(server *mcp.Server) {
	// System info tool
	systemInfoTool := mcp.Tool{
		Name:        "get_system_info",
		Description: "Get general system information",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"include_env": {
					"type": "boolean",
					"description": "Whether to include environment variables",
					"default": false
				}
			}
		}`),
	}

	server.RegisterTool(systemInfoTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		includeEnv := false
		if env, ok := params["include_env"].(bool); ok {
			includeEnv = env
		}

		info, err := getSystemInfo(includeEnv)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Error getting system info: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		resultJSON, _ := json.MarshalIndent(info, "", "  ")
		return &mcp.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": string(resultJSON),
				},
			},
		}, nil
	})

	// Process list tool
	processListTool := mcp.Tool{
		Name:        "list_processes",
		Description: "List running processes",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"limit": {
					"type": "number",
					"description": "Maximum number of processes to return",
					"default": 20
				},
				"filter": {
					"type": "string",
					"description": "Filter processes by name (case-insensitive substring match)"
				}
			}
		}`),
	}

	server.RegisterTool(processListTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		limit := 20
		if l, ok := params["limit"].(float64); ok {
			limit = int(l)
		}

		filter := ""
		if f, ok := params["filter"].(string); ok {
			filter = f
		}

		processes, err := listProcesses(limit, filter)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Error listing processes: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		result := map[string]interface{}{
			"processes": processes,
			"count":     len(processes),
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

	// Disk usage tool
	diskUsageTool := mcp.Tool{
		Name:        "get_disk_usage",
		Description: "Get disk usage information",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Path to check disk usage for (default: current directory)"
				}
			}
		}`),
	}

	server.RegisterTool(diskUsageTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		path := "."
		if p, ok := params["path"].(string); ok && p != "" {
			path = p
		}

		diskInfo, err := getDiskUsage(path)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Error getting disk usage: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		resultJSON, _ := json.MarshalIndent(diskInfo, "", "  ")
		return &mcp.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": string(resultJSON),
				},
			},
		}, nil
	})

	// Environment variables tool
	envVarsTool := mcp.Tool{
		Name:        "get_env_vars",
		Description: "Get environment variables",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"filter": {
					"type": "string",
					"description": "Filter environment variables by name (case-insensitive substring match)"
				}
			}
		}`),
	}

	server.RegisterTool(envVarsTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		filter := ""
		if f, ok := params["filter"].(string); ok {
			filter = f
		}

		envVars := getEnvironmentVariables(filter)

		result := map[string]interface{}{
			"environment_variables": envVars,
			"count":                 len(envVars),
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

	// Execute command tool (be careful with this one!)
	execTool := mcp.Tool{
		Name:        "execute_command",
		Description: "Execute a system command (use with caution)",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"command": {
					"type": "string",
					"description": "The command to execute"
				},
				"args": {
					"type": "array",
					"items": {
						"type": "string"
					},
					"description": "Command arguments"
				},
				"timeout": {
					"type": "number",
					"description": "Timeout in seconds (default: 30)",
					"default": 30
				}
			},
			"required": ["command"]
		}`),
	}

	server.RegisterTool(execTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		command, ok := params["command"].(string)
		if !ok || command == "" {
			return nil, fmt.Errorf("command is required and must be a string")
		}

		var args []string
		if a, ok := params["args"].([]interface{}); ok {
			for _, arg := range a {
				if argStr, ok := arg.(string); ok {
					args = append(args, argStr)
				}
			}
		}

		timeout := 30
		if t, ok := params["timeout"].(float64); ok {
			timeout = int(t)
		}

		result, err := executeCommand(command, args, timeout)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Error executing command: %v", err),
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

	log.Println("Registered system tools: get_system_info, list_processes, get_disk_usage, get_env_vars, execute_command")
}

func getSystemInfo(includeEnv bool) (SystemInfo, error) {
	hostname, _ := os.Hostname()
	
	currentUser, _ := user.Current()
	username := currentUser.Username
	homeDir := currentUser.HomeDir
	
	workingDir, _ := os.Getwd()
	
	info := SystemInfo{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		NumCPU:       runtime.NumCPU(),
		GoVersion:    runtime.Version(),
		Hostname:     hostname,
		Username:     username,
		HomeDir:      homeDir,
		WorkingDir:   workingDir,
	}
	
	if includeEnv {
		info.Environment = make(map[string]string)
		for _, env := range os.Environ() {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				info.Environment[parts[0]] = parts[1]
			}
		}
	}
	
	// Try to get uptime (Unix-like systems only)
	if runtime.GOOS != "windows" {
		if uptime, err := getUptime(); err == nil {
			info.Uptime = uptime
		}
	}
	
	return info, nil
}

func listProcesses(limit int, filter string) ([]ProcessInfo, error) {
	var processes []ProcessInfo
	
	if runtime.GOOS == "windows" {
		return getWindowsProcesses(limit, filter)
	}
	
	return getUnixProcesses(limit, filter)
}

func getUnixProcesses(limit int, filter string) ([]ProcessInfo, error) {
	cmd := exec.Command("ps", "aux")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run ps command: %v", err)
	}
	
	lines := strings.Split(string(output), "\n")
	var processes []ProcessInfo
	
	// Skip header line
	for i, line := range lines[1:] {
		if i >= limit {
			break
		}
		
		fields := strings.Fields(line)
		if len(fields) < 11 {
			continue
		}
		
		name := fields[10]
		if filter != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(filter)) {
			continue
		}
		
		pid, _ := strconv.Atoi(fields[1])
		
		process := ProcessInfo{
			PID:    pid,
			Name:   name,
			CPU:    fields[2] + "%",
			Memory: fields[3] + "%",
			Status: fields[7],
		}
		
		if len(fields) > 11 {
			process.Command = strings.Join(fields[10:], " ")
		}
		
		processes = append(processes, process)
	}
	
	return processes, nil
}

func getWindowsProcesses(limit int, filter string) ([]ProcessInfo, error) {
	cmd := exec.Command("tasklist", "/fo", "csv")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run tasklist command: %v", err)
	}
	
	lines := strings.Split(string(output), "\n")
	var processes []ProcessInfo
	
	// Skip header line
	for i, line := range lines[1:] {
		if i >= limit {
			break
		}
		
		if strings.TrimSpace(line) == "" {
			continue
		}
		
		// Parse CSV-like output
		fields := strings.Split(line, ",")
		if len(fields) < 5 {
			continue
		}
		
		name := strings.Trim(fields[0], "\"")
		if filter != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(filter)) {
			continue
		}
		
		pidStr := strings.Trim(fields[1], "\"")
		pid, _ := strconv.Atoi(pidStr)
		
		memory := strings.Trim(fields[4], "\"")
		
		process := ProcessInfo{
			PID:    pid,
			Name:   name,
			Memory: memory,
		}
		
		processes = append(processes, process)
	}
	
	return processes, nil
}

func getDiskUsage(path string) (DiskInfo, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return DiskInfo{}, err
	}
	
	info := DiskInfo{
		Path: absPath,
	}
	
	if runtime.GOOS == "windows" {
		return getWindowsDiskUsage(absPath)
	}
	
	return getUnixDiskUsage(absPath)
}

func getUnixDiskUsage(path string) (DiskInfo, error) {
	cmd := exec.Command("df", "-h", path)
	output, err := cmd.Output()
	if err != nil {
		return DiskInfo{Path: path}, nil // Return basic info even if df fails
	}
	
	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return DiskInfo{Path: path}, nil
	}
	
	fields := strings.Fields(lines[1])
	if len(fields) >= 6 {
		return DiskInfo{
			Path:       path,
			Filesystem: fields[0],
			Size:       fields[1],
			Used:       fields[2],
			Available:  fields[3],
			UsePercent: fields[4],
			Mountpoint: fields[5],
		}, nil
	}
	
	return DiskInfo{Path: path}, nil
}

func getWindowsDiskUsage(path string) (DiskInfo, error) {
	// Get the drive letter from the path
	if len(path) < 2 {
		return DiskInfo{Path: path}, nil
	}
	
	drive := path[:2] // e.g., "C:"
	
	cmd := exec.Command("fsutil", "volume", "diskfree", drive)
	output, err := cmd.Output()
	if err != nil {
		return DiskInfo{Path: path}, nil
	}
	
	info := DiskInfo{
		Path: path,
	}
	
	// Parse fsutil output (very basic parsing)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "bytes") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				// This is a simplified parsing - in practice you'd want more robust parsing
				if strings.Contains(line, "free") {
					info.Available = parts[3] + " bytes"
				}
			}
		}
	}
	
	return info, nil
}

func getEnvironmentVariables(filter string) map[string]string {
	envVars := make(map[string]string)
	
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]
			
			if filter == "" || strings.Contains(strings.ToLower(key), strings.ToLower(filter)) {
				envVars[key] = value
			}
		}
	}
	
	return envVars
}

func executeCommand(command string, args []string, timeoutSecs int) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSecs)*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, command, args...)
	
	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)
	
	result := map[string]interface{}{
		"command":  command,
		"args":     args,
		"output":   string(output),
		"duration": duration.String(),
	}
	
	if err != nil {
		result["error"] = err.Error()
		if exitError, ok := err.(*exec.ExitError); ok {
			result["exit_code"] = exitError.ExitCode()
		}
	} else {
		result["exit_code"] = 0
	}
	
	return result, nil
}

func getUptime() (string, error) {
	if runtime.GOOS == "darwin" || runtime.GOOS == "freebsd" {
		cmd := exec.Command("uptime")
		output, err := cmd.Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(output)), nil
	}
	
	if runtime.GOOS == "linux" {
		data, err := os.ReadFile("/proc/uptime")
		if err != nil {
			return "", err
		}
		
		fields := strings.Fields(string(data))
		if len(fields) > 0 {
			uptimeSeconds, err := strconv.ParseFloat(fields[0], 64)
			if err != nil {
				return "", err
			}
			
			duration := time.Duration(uptimeSeconds) * time.Second
			return duration.String(), nil
		}
	}
	
	return "", fmt.Errorf("uptime not supported on this platform")
}