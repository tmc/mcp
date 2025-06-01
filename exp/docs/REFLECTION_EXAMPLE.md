# Reflection Example: Docker CLI to MCP Server

Let's see how the advanced reflection tools would analyze the Docker CLI and convert it to an MCP server.

## Docker CLI Analysis

### 1. Subcommand Detection

```go
// Docker uses a custom command structure
func main() {
    if len(os.Args) < 2 {
        showUsage()
        return
    }
    
    switch os.Args[1] {
    case "ps":
        listContainers()
    case "run":
        runContainer()
    case "stop":
        stopContainer()
    case "logs":
        showLogs()
    case "inspect":
        inspectContainer()
    }
}
```

### 2. Flag Detection

```go
// docker ps flags
func listContainers() {
    var all bool
    var format string
    var quiet bool
    
    flag.BoolVar(&all, "a", false, "Show all containers")
    flag.StringVar(&format, "format", "", "Pretty-print containers using a Go template")
    flag.BoolVar(&quiet, "q", false, "Only display container IDs")
    flag.Parse()
    
    // ... implementation
}

// docker run flags
func runContainer() {
    var detach bool
    var name string
    var env []string
    var port []string
    var volume []string
    
    flag.BoolVar(&detach, "d", false, "Run container in background")
    flag.StringVar(&name, "name", "", "Assign a name to the container")
    flag.Var(&env, "e", "Set environment variables")
    flag.Var(&port, "p", "Publish container's port(s)")
    flag.Var(&volume, "v", "Bind mount a volume")
    flag.Parse()
    
    // ... implementation
}
```

### 3. JSON Output Detection

```go
// docker ps output
func listContainers() {
    // ... flag parsing ...
    
    containers := getContainers(all)
    
    if format != "" {
        tmpl, _ := template.New("").Parse(format)
        tmpl.Execute(os.Stdout, containers)
    } else if quiet {
        for _, c := range containers {
            fmt.Println(c.ID)
        }
    } else {
        // Default table output
        data, _ := json.Marshal(containers)
        fmt.Println(string(data))
    }
}

// docker inspect output
func inspectContainer() {
    containerID := os.Args[2]
    info := getContainerInfo(containerID)
    
    // Always outputs JSON
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    encoder.Encode(info)
}

// Types that get marshaled
type Container struct {
    ID      string   `json:"Id"`
    Names   []string `json:"Names"`
    Image   string   `json:"Image"`
    Command string   `json:"Command"`
    Created int64    `json:"Created"`
    Status  string   `json:"Status"`
    Ports   []Port   `json:"Ports"`
}

type ContainerInfo struct {
    Container
    Config     *Config     `json:"Config"`
    HostConfig *HostConfig `json:"HostConfig"`
    Mounts     []Mount     `json:"Mounts"`
    State      *State      `json:"State"`
}
```

## Reflection Analysis Results

### Detected Command Structure

```yaml
tool: docker
framework: custom
commands:
  ps:
    description: "List containers"
    flags:
      - name: all
        type: bool
        short: a
        description: "Show all containers"
      - name: format
        type: string
        description: "Pretty-print containers using a Go template"
      - name: quiet
        type: bool
        short: q
        description: "Only display container IDs"
    output_schema:
      type: array
      items:
        type: object
        properties:
          Id: { type: string }
          Names: { type: array, items: { type: string } }
          Image: { type: string }
          Command: { type: string }
          Created: { type: integer }
          Status: { type: string }
          Ports:
            type: array
            items:
              type: object
              properties:
                IP: { type: string }
                PrivatePort: { type: integer }
                PublicPort: { type: integer }
                Type: { type: string }

  run:
    description: "Run a new container"
    flags:
      - name: detach
        type: bool
        short: d
        description: "Run container in background"
      - name: name
        type: string
        description: "Assign a name to the container"
      - name: env
        type: array
        short: e
        description: "Set environment variables"
      - name: port
        type: array
        short: p
        description: "Publish container's port(s)"
      - name: volume
        type: array
        short: v
        description: "Bind mount a volume"
    arguments:
      - name: image
        type: string
        required: true
      - name: command
        type: string
        required: false
    output_schema:
      type: object
      properties:
        Id: { type: string }
        Warnings: { type: array, items: { type: string } }

  inspect:
    description: "Return low-level information on Docker objects"
    arguments:
      - name: name
        type: string
        required: true
    output_schema:
      type: object
      properties:
        Id: { type: string }
        Created: { type: string }
        Path: { type: string }
        Args: { type: array, items: { type: string } }
        State:
          type: object
          properties:
            Status: { type: string }
            Running: { type: boolean }
            Paused: { type: boolean }
            # ... more state properties
        Config:
          type: object
          properties:
            Hostname: { type: string }
            Image: { type: string }
            Env: { type: array, items: { type: string } }
            # ... more config properties
```

## Generated MCP Server

```go
// Generated by mcp-reflect
package main

import (
    "context"
    "encoding/json"
    "github.com/tmc/mcp"
)

type DockerMCPServer struct {
    *mcp.Server
}

func NewDockerMCPServer() *DockerMCPServer {
    server := &DockerMCPServer{
        Server: mcp.NewServer(),
    }
    
    // Register tools based on detected commands
    server.AddTool(&mcp.Tool{
        Name:        "docker_ps",
        Description: "List containers",
        InputSchema: &mcp.InputSchema{
            Type: "object",
            Properties: map[string]interface{}{
                "all": {
                    "type":        "boolean",
                    "description": "Show all containers",
                    "default":     false,
                },
                "format": {
                    "type":        "string",
                    "description": "Pretty-print containers using a Go template",
                },
                "quiet": {
                    "type":        "boolean",
                    "description": "Only display container IDs",
                    "default":     false,
                },
            },
        },
        OutputSchema: &mcp.OutputSchema{
            Type: "array",
            Items: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "Id":      map[string]string{"type": "string"},
                    "Names":   map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
                    "Image":   map[string]string{"type": "string"},
                    "Command": map[string]string{"type": "string"},
                    "Created": map[string]string{"type": "integer"},
                    "Status":  map[string]string{"type": "string"},
                    "Ports": map[string]interface{}{
                        "type": "array",
                        "items": map[string]interface{}{
                            "type": "object",
                            "properties": map[string]interface{}{
                                "IP":          map[string]string{"type": "string"},
                                "PrivatePort": map[string]string{"type": "integer"},
                                "PublicPort":  map[string]string{"type": "integer"},
                                "Type":        map[string]string{"type": "string"},
                            },
                        },
                    },
                },
            },
        },
        Handler: server.handleDockerPs,
    })
    
    server.AddTool(&mcp.Tool{
        Name:        "docker_run",
        Description: "Run a new container",
        InputSchema: &mcp.InputSchema{
            Type: "object",
            Properties: map[string]interface{}{
                "image": {
                    "type":        "string",
                    "description": "Container image",
                },
                "command": {
                    "type":        "string",
                    "description": "Command to run",
                },
                "detach": {
                    "type":        "boolean",
                    "description": "Run container in background",
                    "default":     false,
                },
                "name": {
                    "type":        "string",
                    "description": "Assign a name to the container",
                },
                "env": {
                    "type":        "array",
                    "items":       map[string]string{"type": "string"},
                    "description": "Set environment variables",
                },
                "port": {
                    "type":        "array",
                    "items":       map[string]string{"type": "string"},
                    "description": "Publish container's port(s)",
                },
                "volume": {
                    "type":        "array",
                    "items":       map[string]string{"type": "string"},
                    "description": "Bind mount a volume",
                },
            },
            Required: []string{"image"},
        },
        Handler: server.handleDockerRun,
    })
    
    server.AddTool(&mcp.Tool{
        Name:        "docker_inspect",
        Description: "Return low-level information on Docker objects",
        InputSchema: &mcp.InputSchema{
            Type: "object",
            Properties: map[string]interface{}{
                "name": {
                    "type":        "string",
                    "description": "Container name or ID",
                },
            },
            Required: []string{"name"},
        },
        OutputSchema: detectedInspectSchema, // Complex nested schema
        Handler:      server.handleDockerInspect,
    })
    
    return server
}

// Handler implementations
func (s *DockerMCPServer) handleDockerPs(ctx context.Context, params json.RawMessage) (*mcp.CallToolResult, error) {
    var input struct {
        All    bool   `json:"all"`
        Format string `json:"format"`
        Quiet  bool   `json:"quiet"`
    }
    
    if err := json.Unmarshal(params, &input); err != nil {
        return nil, err
    }
    
    // Execute actual docker ps command
    result, err := executeDockerCommand("ps", dockerPsFlags(input))
    if err != nil {
        return nil, err
    }
    
    // Parse JSON output
    var containers []Container
    if err := json.Unmarshal(result, &containers); err != nil {
        return nil, err
    }
    
    return &mcp.CallToolResult{
        Content: []mcp.Content{
            {
                Type: "text",
                Text: string(result),
            },
        },
    }, nil
}

func (s *DockerMCPServer) handleDockerRun(ctx context.Context, params json.RawMessage) (*mcp.CallToolResult, error) {
    var input struct {
        Image   string   `json:"image"`
        Command string   `json:"command"`
        Detach  bool     `json:"detach"`
        Name    string   `json:"name"`
        Env     []string `json:"env"`
        Port    []string `json:"port"`
        Volume  []string `json:"volume"`
    }
    
    if err := json.Unmarshal(params, &input); err != nil {
        return nil, err
    }
    
    // Build docker run command
    args := dockerRunArgs(input)
    result, err := executeDockerCommand("run", args)
    if err != nil {
        return nil, err
    }
    
    return &mcp.CallToolResult{
        Content: []mcp.Content{
            {
                Type: "text",
                Text: string(result),
            },
        },
    }, nil
}

func (s *DockerMCPServer) handleDockerInspect(ctx context.Context, params json.RawMessage) (*mcp.CallToolResult, error) {
    var input struct {
        Name string `json:"name"`
    }
    
    if err := json.Unmarshal(params, &input); err != nil {
        return nil, err
    }
    
    // Execute docker inspect
    result, err := executeDockerCommand("inspect", []string{input.Name})
    if err != nil {
        return nil, err
    }
    
    // Result is already JSON
    return &mcp.CallToolResult{
        Content: []mcp.Content{
            {
                Type: "text",
                Text: string(result),
            },
        },
    }, nil
}

// Helper functions
func executeDockerCommand(cmd string, args []string) ([]byte, error) {
    // Implementation to execute docker command
    // This would use os/exec to run the actual docker binary
}

func dockerPsFlags(input struct{All bool; Format string; Quiet bool}) []string {
    var args []string
    if input.All {
        args = append(args, "-a")
    }
    if input.Format != "" {
        args = append(args, "--format", input.Format)
    }
    if input.Quiet {
        args = append(args, "-q")
    }
    return args
}

func dockerRunArgs(input struct{
    Image string
    Command string
    Detach bool
    Name string
    Env []string
    Port []string
    Volume []string
}) []string {
    var args []string
    
    if input.Detach {
        args = append(args, "-d")
    }
    if input.Name != "" {
        args = append(args, "--name", input.Name)
    }
    for _, env := range input.Env {
        args = append(args, "-e", env)
    }
    for _, port := range input.Port {
        args = append(args, "-p", port)
    }
    for _, vol := range input.Volume {
        args = append(args, "-v", vol)
    }
    
    args = append(args, input.Image)
    if input.Command != "" {
        args = append(args, input.Command)
    }
    
    return args
}

func main() {
    server := NewDockerMCPServer()
    
    // Run with stdio transport by default
    transport := mcp.NewStdioTransport()
    server.Serve(transport)
}
```

## Benefits of Advanced Reflection

1. **Automatic Schema Generation**: The tool detected the JSON output structures and created proper schemas
2. **Subcommand Mapping**: Each Docker subcommand became an MCP tool
3. **Flag Translation**: Command-line flags became tool parameters with proper types
4. **Output Detection**: JSON output patterns were identified and schemas generated
5. **Complete Tool Definition**: No manual schema writing required

This example shows how the enhanced reflection tools can automatically convert complex CLI tools like Docker into fully-functional MCP servers with accurate input/output schemas.