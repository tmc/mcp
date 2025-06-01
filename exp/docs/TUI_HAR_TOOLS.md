# TUI and HAR Analysis Tools

Tools for testing Terminal User Interface applications and analyzing HTTP Archive data, designed as exp/ packages that can be automatically converted to MCP servers.

## 🖥️ TUI Testing Tools

### 1. **tui-record**: Terminal Session Recorder
**Purpose**: Record terminal interactions for replay and analysis
```go
// exp/tui/record/record.go
package record

type Session struct {
    ID        string
    StartTime time.Time
    Commands  []Command
    Outputs   []Output
    Timings   []Timing
}

type Recorder struct {
    pty      *os.File
    session  *Session
    encoding string // asciinema, typescript, ttyrec
}

func (r *Recorder) Start(cmd string, args []string) error
func (r *Recorder) Stop() (*Session, error)
func (r *Recorder) Export(format string) ([]byte, error)
```

**CLI Usage**:
```bash
# Record a session
tui-record start vim test.go

# Export to various formats
tui-record export --format=asciinema session.json
tui-record export --format=gif session.gif

# Convert to MCP trace
tui-record to-mcp session.json > tui-trace.mcp
```

### 2. **tui-replay**: Session Replay Engine
**Purpose**: Replay recorded sessions with modifications
```go
// exp/tui/replay/replay.go
package replay

type Player struct {
    session  *Session
    speed    float64
    paused   bool
    position time.Duration
}

func (p *Player) Play() error
func (p *Player) Pause()
func (p *Player) Seek(position time.Duration)
func (p *Player) SetSpeed(speed float64)
func (p *Player) InjectInput(input string) error
```

**Features**:
- Variable speed playback
- Pause/resume capability
- Seek to specific points
- Input injection for testing

### 3. **tui-test**: Automated TUI Testing
**Purpose**: Automated testing framework for TUI applications
```go
// exp/tui/test/test.go
package test

type TUITest struct {
    App      string
    Timeout  time.Duration
    Asserts  []Assert
}

type Assert struct {
    Type     string // contains, regex, position
    Expected string
    Position Position
}

func (t *TUITest) Run() (*Result, error)
func (t *TUITest) WaitForOutput(pattern string) error
func (t *TUITest) SendInput(input string) error
func (t *TUITest) Screenshot() ([]byte, error)
```

**Test Definition**:
```yaml
# tui-test.yaml
name: "Test Vim Navigation"
app: "vim test.txt"
timeout: 10s
steps:
  - wait: "test.txt"
  - input: "i"
  - wait: "-- INSERT --"
  - input: "Hello, World!"
  - input: "\x1b"  # ESC
  - assert:
      contains: "Hello, World!"
  - input: ":wq\n"
  - assert:
      exit_code: 0
```

### 4. **tui-analyze**: TUI Behavior Analysis
**Purpose**: Analyze TUI application behavior and patterns
```go
// exp/tui/analyze/analyze.go
package analyze

type Analyzer struct {
    sessions []Session
}

type Analysis struct {
    CommandFrequency map[string]int
    ResponseTimes    []time.Duration
    ErrorPatterns    []Pattern
    NavigationFlow   *Flow
}

func (a *Analyzer) Analyze() (*Analysis, error)
func (a *Analyzer) CompareSessionz(s1, s2 *Session) (*Diff, error)
func (a *Analyzer) ExtractPatterns() ([]Pattern, error)
```

### 5. **tui-fuzz**: TUI Fuzzing Tool
**Purpose**: Fuzz testing for TUI applications
```go
// exp/tui/fuzz/fuzz.go
package fuzz

type Fuzzer struct {
    App       string
    Seed      int64
    Mutations []Mutation
}

type Mutation struct {
    Type   string // random_char, sequence, control
    Weight float64
}

func (f *Fuzzer) Fuzz(duration time.Duration) (*Report, error)
func (f *Fuzzer) GenerateInput() string
func (f *Fuzzer) DetectCrash(output string) bool
```

## 🌐 HAR Analysis Tools

### 6. **har-parse**: HAR File Parser
**Purpose**: Parse and analyze HTTP Archive files
```go
// exp/har/parse/parse.go
package parse

type HAR struct {
    Log Log `json:"log"`
}

type Entry struct {
    StartedDateTime time.Time
    Request         Request
    Response        Response
    Timings         Timings
}

func Parse(data []byte) (*HAR, error)
func (h *HAR) Filter(predicate func(Entry) bool) *HAR
func (h *HAR) ToMCPTrace() (*mcp.Trace, error)
```

### 7. **har-analyze**: Performance Analysis
**Purpose**: Analyze HTTP performance from HAR files
```go
// exp/har/analyze/analyze.go
package analyze

type Analyzer struct {
    har *HAR
}

type Metrics struct {
    TotalRequests   int
    AverageLatency  time.Duration
    P95Latency      time.Duration
    ErrorRate       float64
    BytesTransferred int64
    Waterfall       []WaterfallItem
}

func (a *Analyzer) Analyze() (*Metrics, error)
func (a *Analyzer) FindBottlenecks() ([]Bottleneck, error)
func (a *Analyzer) GenerateReport() (*Report, error)
```

### 8. **har-diff**: HAR Comparison Tool
**Purpose**: Compare HAR files to identify differences
```go
// exp/har/diff/diff.go
package diff

type Differ struct {
    baseline *HAR
    current  *HAR
}

type Diff struct {
    AddedEndpoints   []string
    RemovedEndpoints []string
    LatencyChanges   map[string]float64
    SizeChanges      map[string]int64
}

func (d *Differ) Compare() (*Diff, error)
func (d *Differ) GenerateReport() (*Report, error)
```

### 9. **har-mock**: Mock Server from HAR
**Purpose**: Create mock servers from HAR files
```go
// exp/har/mock/mock.go
package mock

type MockServer struct {
    har      *HAR
    matching MatchStrategy
}

func (m *MockServer) Start(addr string) error
func (m *MockServer) HandleRequest(w http.ResponseWriter, r *http.Request)
func (m *MockServer) FindMatch(r *http.Request) (*Response, error)
```

## 🔍 Enhanced Reflection/Generation Tools

### 10. **reflect-mcp**: Advanced MCP Reflection
**Purpose**: Enhanced reflection for MCP type generation
```go
// exp/reflect/mcp/mcp.go
package mcp

type Reflector struct {
    pkg *types.Package
}

type MCPMapping struct {
    GoType      types.Type
    MCPSchema   *jsonschema.Schema
    Validations []Validation
    Examples    []interface{}
}

func (r *Reflector) ReflectType(t types.Type) (*MCPMapping, error)
func (r *Reflector) GenerateTool(fn *types.Func) (*mcp.Tool, error)
func (r *Reflector) ExtractCapabilities(t types.Type) ([]mcp.Capability, error)
```

### 11. **gen-validate**: Validation Code Generator
**Purpose**: Generate validation code from MCP schemas
```go
// exp/gen/validate/validate.go
package validate

type Generator struct {
    schema *jsonschema.Schema
}

func (g *Generator) GenerateValidator() (string, error)
func (g *Generator) GenerateTests() (string, error)
func (g *Generator) GenerateFuzzer() (string, error)
```

### 12. **gen-client**: MCP Client Generator
**Purpose**: Generate type-safe MCP clients from specs
```go
// exp/gen/client/client.go
package client

type ClientGenerator struct {
    spec *mcp.Specification
}

func (g *ClientGenerator) Generate() (string, error)
func (g *ClientGenerator) GenerateAsync() (string, error)
func (g *ClientGenerator) GenerateMock() (string, error)
```

### 13. **gen-server**: MCP Server Generator
**Purpose**: Generate complete MCP servers from interfaces
```go
// exp/gen/server/server.go
package server

type ServerGenerator struct {
    interfaces []types.Type
    options    GeneratorOptions
}

func (g *ServerGenerator) Generate() (string, error)
func (g *ServerGenerator) GenerateTests() (string, error)
func (g *ServerGenerator) GenerateDocumentation() (string, error)
```

## 🔄 MCP Server Conversion

All these tools can be automatically converted to MCP servers:

```go
// exp/cmd/auto-mcp/main.go
package main

func ConvertToMCPServer(tool interface{}) *mcp.Server {
    server := mcp.NewServer()
    
    // Reflect on tool methods
    t := reflect.TypeOf(tool)
    for i := 0; i < t.NumMethod(); i++ {
        method := t.Method(i)
        
        // Convert method to MCP tool
        server.AddTool(&mcp.Tool{
            Name:        strings.ToLower(method.Name),
            Description: extractDoc(method),
            InputSchema: generateSchema(method.Type.In),
            Handler:     wrapMethod(tool, method),
        })
    }
    
    return server
}
```

Example conversion:
```bash
# Convert TUI recorder to MCP server
auto-mcp tui-record > mcp-tui-record-server

# Run as MCP server
mcp-tui-record-server --stdio
```

## 📊 Integration Examples

### TUI + MCP Testing
```go
// Test a TUI app through MCP
func TestTUIApp(t *testing.T) {
    // Record baseline interaction
    recorder := tui.NewRecorder()
    session := recorder.Record("vim", "test.txt")
    
    // Convert to MCP test
    mcpTest := session.ToMCPTest()
    
    // Run through arena
    result := arena.Run(mcpTest)
    assert.Equal(t, result.Score, 1.0)
}
```

### HAR + Performance Analysis
```go
// Analyze API performance changes
func AnalyzeAPIChanges(before, after string) {
    harBefore := har.Parse(before)
    harAfter := har.Parse(after)
    
    diff := har.Diff(harBefore, harAfter)
    
    // Generate MCP-compatible report
    report := diff.ToMCPReport()
    server.SendNotification(report)
}
```

### Reflection + Code Generation
```go
// Generate MCP server from interface
func GenerateFromInterface(i interface{}) {
    reflector := reflect.NewMCPReflector()
    mapping := reflector.Reflect(i)
    
    generator := gen.NewServerGenerator(mapping)
    code := generator.Generate()
    
    // Output complete MCP server
    fmt.Println(code)
}
```

## 🚀 Advanced Features

### 1. **TUI Regression Testing**
```yaml
# tui-regression.yaml
baseline: v1.0.0
current: main
tests:
  - name: "Navigation Speed"
    app: "htop"
    actions:
      - input: "/"
      - input: "chrome"
    assert:
      response_time: "<100ms"
      
  - name: "Memory Usage"
    app: "top"
    assert:
      memory: "<50MB"
```

### 2. **HAR-based Load Testing**
```go
// Generate load test from HAR
func GenerateLoadTest(harFile string) *LoadTest {
    har := har.Parse(harFile)
    
    test := &LoadTest{
        Duration: 5 * time.Minute,
        RPS:      100,
    }
    
    for _, entry := range har.Entries {
        test.AddRequest(entry.ToRequest())
    }
    
    return test
}
```

### 3. **Automatic MCP Spec Coverage**
```go
// Check MCP spec coverage
func CheckSpecCoverage(pkg *types.Package) *Coverage {
    reflector := reflect.NewMCPReflector()
    spec := mcp.LoadSpec("2024-01-01")
    
    coverage := &Coverage{}
    
    for _, capability := range spec.Capabilities {
        if reflector.SupportsCapability(pkg, capability) {
            coverage.Supported = append(coverage.Supported, capability)
        } else {
            coverage.Missing = append(coverage.Missing, capability)
        }
    }
    
    return coverage
}
```

These tools provide comprehensive testing and analysis capabilities for both TUI applications and HTTP traffic, while the enhanced reflection/generation tools ensure complete MCP spec coverage. All tools are designed as modular exp/ packages that can be easily converted to MCP servers for integration into the larger ecosystem.