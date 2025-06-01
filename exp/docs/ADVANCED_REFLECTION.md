# Advanced Reflection: JSON Output & Subcommand Detection

## 🔍 JSON Output Schema Detection

### 1. **reflect-output**: Stdout JSON Detection
```go
// exp/reflect/output/detector.go
package output

import (
    "go/ast"
    "go/types"
    "golang.org/x/tools/go/ssa"
)

type OutputDetector struct {
    prog *ssa.Program
    pkg  *ssa.Package
}

// DetectJSONOutput finds types that are marshaled to stdout
func (d *OutputDetector) DetectJSONOutput() ([]*OutputSchema, error) {
    var schemas []*OutputSchema
    
    // Find all json.Marshal calls
    for _, fn := range d.pkg.Funcs {
        for _, block := range fn.Blocks {
            for _, instr := range block.Instrs {
                if call, ok := instr.(*ssa.Call); ok {
                    if schema := d.analyzeCall(call); schema != nil {
                        schemas = append(schemas, schema)
                    }
                }
            }
        }
    }
    
    return schemas, nil
}

// Analyze call to detect JSON marshaling to stdout
func (d *OutputDetector) analyzeCall(call *ssa.Call) *OutputSchema {
    // Check if this is json.Marshal
    if !d.isJSONMarshal(call) {
        return nil
    }
    
    // Get the type being marshaled
    marshaledType := d.getMarshaledType(call)
    if marshaledType == nil {
        return nil
    }
    
    // Trace data flow to see if it goes to stdout
    if !d.flowsToStdout(call) {
        return nil
    }
    
    // Generate schema from type
    return &OutputSchema{
        Type:   marshaledType,
        Schema: d.generateSchema(marshaledType),
        Usage:  d.findUsageContext(call),
    }
}

// Trace data flow analysis
func (d *OutputDetector) flowsToStdout(call *ssa.Call) bool {
    // Use SSA to trace where the marshaled data flows
    for _, ref := range *call.Referrers() {
        switch v := ref.(type) {
        case *ssa.Call:
            // Check for os.Stdout.Write, fmt.Print, etc.
            if d.isStdoutWrite(v) {
                return true
            }
        case *ssa.Store:
            // Check if stored value eventually goes to stdout
            if d.storedValueToStdout(v) {
                return true
            }
        }
    }
    return false
}
```

### 2. **reflect-pattern**: Pattern Recognition
```go
// exp/reflect/pattern/recognizer.go
package pattern

type PatternRecognizer struct {
    ast *ast.File
}

// Common stdout patterns to detect
var stdoutPatterns = []Pattern{
    {
        Name: "DirectMarshalPrint",
        // json.Marshal(v) followed by fmt.Print
        Pattern: `
            data, err := json.Marshal($VAR)
            if err != nil { $ERROR_HANDLING }
            fmt.Print(string(data))
        `,
    },
    {
        Name: "EncoderPattern", 
        // json.NewEncoder(os.Stdout)
        Pattern: `
            encoder := json.NewEncoder(os.Stdout)
            encoder.Encode($VAR)
        `,
    },
    {
        Name: "WriterPattern",
        // Using io.Writer interface
        Pattern: `
            w := os.Stdout
            json.NewEncoder(w).Encode($VAR)
        `,
    },
}

// DetectOutputPatterns finds JSON output patterns
func (r *PatternRecognizer) DetectOutputPatterns() ([]*OutputPattern, error) {
    var patterns []*OutputPattern
    
    ast.Inspect(r.ast, func(n ast.Node) bool {
        if fn, ok := n.(*ast.FuncDecl); ok {
            if pattern := r.matchFunction(fn); pattern != nil {
                patterns = append(patterns, pattern)
            }
        }
        return true
    })
    
    return patterns, nil
}
```

### 3. **reflect-schema**: Enhanced Schema Generation
```go
// exp/reflect/schema/generator.go
package schema

type SchemaGenerator struct {
    types map[string]*types.Type
}

// GenerateFromOutput creates schema from output detection
func (g *SchemaGenerator) GenerateFromOutput(output *OutputSchema) *mcp.OutputSchema {
    schema := &mcp.OutputSchema{
        Type: "object",
        Properties: make(map[string]interface{}),
    }
    
    // Analyze the Go type
    switch t := output.Type.(type) {
    case *types.Struct:
        for i := 0; i < t.NumFields(); i++ {
            field := t.Field(i)
            tag := reflect.StructTag(t.Tag(i))
            
            // Use JSON tag if available
            jsonTag := tag.Get("json")
            if jsonTag == "-" {
                continue
            }
            
            fieldName := jsonTag
            if fieldName == "" {
                fieldName = field.Name()
            }
            
            schema.Properties[fieldName] = g.typeToSchema(field.Type())
        }
    }
    
    return schema
}

// Handle complex types including slices, maps, etc.
func (g *SchemaGenerator) typeToSchema(t types.Type) interface{} {
    switch typ := t.(type) {
    case *types.Slice:
        return map[string]interface{}{
            "type": "array",
            "items": g.typeToSchema(typ.Elem()),
        }
    case *types.Map:
        return map[string]interface{}{
            "type": "object",
            "additionalProperties": g.typeToSchema(typ.Elem()),
        }
    case *types.Named:
        // Recursively handle named types
        return g.typeToSchema(typ.Underlying())
    default:
        return g.basicTypeToSchema(typ)
    }
}
```

## 🔧 Subcommand Detection

### 4. **reflect-subcommand**: CLI Structure Detection
```go
// exp/reflect/subcommand/detector.go
package subcommand

type SubcommandDetector struct {
    pkg  *types.Package
    prog *ssa.Program
}

// DetectSubcommands finds CLI subcommand structures
func (d *SubcommandDetector) DetectSubcommands() (*CommandStructure, error) {
    structure := &CommandStructure{
        Commands: make(map[string]*Command),
    }
    
    // Detect common CLI frameworks
    if cmds := d.detectCobraCommands(); cmds != nil {
        structure.Framework = "cobra"
        structure.Commands = cmds
    } else if cmds := d.detectUrfaveCommands(); cmds != nil {
        structure.Framework = "urfave/cli"
        structure.Commands = cmds
    } else if cmds := d.detectFlagSubcommands(); cmds != nil {
        structure.Framework = "flag"
        structure.Commands = cmds
    } else {
        // Fallback to pattern detection
        cmds := d.detectCustomPatterns()
        structure.Framework = "custom"
        structure.Commands = cmds
    }
    
    return structure, nil
}

// Detect Cobra command patterns
func (d *SubcommandDetector) detectCobraCommands() map[string]*Command {
    commands := make(map[string]*Command)
    
    // Look for cobra.Command initialization
    for _, decl := range d.findTypeDecls("cobra.Command") {
        if cmd := d.analyzeCobraCommand(decl); cmd != nil {
            commands[cmd.Name] = cmd
        }
    }
    
    // Look for AddCommand calls
    for _, call := range d.findMethodCalls("AddCommand") {
        if parent, child := d.extractCommandRelation(call); parent != "" {
            if cmd, ok := commands[child]; ok {
                cmd.Parent = parent
            }
        }
    }
    
    return commands
}

// Analyze command structure
func (d *SubcommandDetector) analyzeCobraCommand(decl ast.Node) *Command {
    cmd := &Command{}
    
    // Extract command configuration
    ast.Inspect(decl, func(n ast.Node) bool {
        switch v := n.(type) {
        case *ast.KeyValueExpr:
            key := d.getIdentName(v.Key)
            switch key {
            case "Use":
                cmd.Name = d.getStringValue(v.Value)
            case "Short":
                cmd.Short = d.getStringValue(v.Value)
            case "Long":
                cmd.Long = d.getStringValue(v.Value)
            case "Run", "RunE":
                cmd.Handler = d.analyzeFunctionLiteral(v.Value)
            }
        }
        return true
    })
    
    return cmd
}

// Detect flag-based subcommands
func (d *SubcommandDetector) detectFlagSubcommands() map[string]*Command {
    commands := make(map[string]*Command)
    
    // Look for os.Args parsing patterns
    mainFunc := d.findMainFunction()
    if mainFunc == nil {
        return nil
    }
    
    // Analyze main function for subcommand patterns
    ast.Inspect(mainFunc, func(n ast.Node) bool {
        switch v := n.(type) {
        case *ast.SwitchStmt:
            // Check if switching on os.Args[1]
            if d.isSwitchOnArgs(v) {
                for _, clause := range v.Body.List {
                    if cc, ok := clause.(*ast.CaseClause); ok {
                        cmd := d.extractCaseCommand(cc)
                        if cmd != nil {
                            commands[cmd.Name] = cmd
                        }
                    }
                }
            }
        case *ast.IfStmt:
            // Check for if len(os.Args) > 1 patterns
            if cmd := d.extractIfCommand(v); cmd != nil {
                commands[cmd.Name] = cmd
            }
        }
        return true
    })
    
    return commands
}
```

### 5. **reflect-flags**: Flag Detection
```go
// exp/reflect/flags/detector.go
package flags

type FlagDetector struct {
    pkg *types.Package
}

// DetectFlags finds command-line flags for each command
func (d *FlagDetector) DetectFlags(cmd *Command) ([]*Flag, error) {
    var flags []*Flag
    
    // Find flag definitions in command handler
    if cmd.Handler != nil {
        ast.Inspect(cmd.Handler, func(n ast.Node) bool {
            if call, ok := n.(*ast.CallExpr); ok {
                if flag := d.extractFlag(call); flag != nil {
                    flags = append(flags, flag)
                }
            }
            return true
        })
    }
    
    // Find persistent/global flags
    globalFlags := d.findGlobalFlags()
    flags = append(flags, globalFlags...)
    
    return flags, nil
}

// Extract flag information from function call
func (d *FlagDetector) extractFlag(call *ast.CallExpr) *Flag {
    funcName := d.getFunctionName(call)
    
    switch funcName {
    case "flag.StringVar", "pflag.StringVar":
        return d.extractStringFlag(call)
    case "flag.IntVar", "pflag.IntVar":
        return d.extractIntFlag(call)
    case "flag.BoolVar", "pflag.BoolVar":
        return d.extractBoolFlag(call)
    // ... other flag types
    }
    
    return nil
}

// Generate MCP tool input schema from flags
func (d *FlagDetector) GenerateInputSchema(flags []*Flag) *mcp.InputSchema {
    schema := &mcp.InputSchema{
        Type:       "object",
        Properties: make(map[string]interface{}),
        Required:   []string{},
    }
    
    for _, flag := range flags {
        prop := map[string]interface{}{
            "type":        flag.Type,
            "description": flag.Usage,
        }
        
        if flag.Default != nil {
            prop["default"] = flag.Default
        }
        
        schema.Properties[flag.Name] = prop
        
        if flag.Required {
            schema.Required = append(schema.Required, flag.Name)
        }
    }
    
    return schema
}
```

### 6. **Integration: Complete Tool Analysis**
```go
// exp/reflect/complete/analyzer.go
package complete

type CompleteAnalyzer struct {
    outputDetector     *output.OutputDetector
    subcommandDetector *subcommand.SubcommandDetector
    flagDetector       *flags.FlagDetector
    schemaGenerator    *schema.SchemaGenerator
}

// AnalyzeCLI performs complete analysis of a CLI tool
func (a *CompleteAnalyzer) AnalyzeCLI(pkg *types.Package) (*CLIAnalysis, error) {
    analysis := &CLIAnalysis{}
    
    // Detect subcommands
    cmdStructure, err := a.subcommandDetector.DetectSubcommands()
    if err != nil {
        return nil, err
    }
    analysis.Commands = cmdStructure
    
    // For each command, detect inputs and outputs
    for name, cmd := range cmdStructure.Commands {
        // Detect flags (inputs)
        flags, err := a.flagDetector.DetectFlags(cmd)
        if err != nil {
            return nil, err
        }
        cmd.InputSchema = a.flagDetector.GenerateInputSchema(flags)
        
        // Detect JSON output
        outputs, err := a.outputDetector.DetectJSONOutput()
        if err != nil {
            return nil, err
        }
        
        // Match outputs to commands based on context
        for _, output := range outputs {
            if a.isOutputForCommand(output, cmd) {
                cmd.OutputSchema = a.schemaGenerator.GenerateFromOutput(output)
            }
        }
    }
    
    return analysis, nil
}

// Generate MCP tools from CLI analysis
func (a *CompleteAnalyzer) GenerateMCPTools(analysis *CLIAnalysis) ([]*mcp.Tool, error) {
    var tools []*mcp.Tool
    
    for _, cmd := range analysis.Commands.Commands {
        tool := &mcp.Tool{
            Name:        cmd.Name,
            Description: cmd.Short,
            LongDescription: cmd.Long,
            InputSchema: cmd.InputSchema,
        }
        
        // Use detected output schema if available
        if cmd.OutputSchema != nil {
            tool.OutputSchema = cmd.OutputSchema
        } else {
            // Default to standard MCP output
            tool.OutputSchema = a.generateDefaultOutputSchema()
        }
        
        tools = append(tools, tool)
    }
    
    return tools, nil
}
```

## 📊 Usage Examples

### Detecting JSON Output
```go
// Example CLI tool
func main() {
    result := processData()
    
    // This pattern will be detected
    data, err := json.Marshal(result)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(string(data))
}

// Detected output schema:
// {
//   "type": "object",
//   "properties": {
//     "status": {"type": "string"},
//     "count": {"type": "integer"},
//     "items": {
//       "type": "array",
//       "items": {"type": "string"}
//     }
//   }
// }
```

### Detecting Subcommands
```go
// Example with flag-based subcommands
func main() {
    if len(os.Args) < 2 {
        usage()
        return
    }
    
    switch os.Args[1] {
    case "list":
        cmdList()
    case "get":
        cmdGet()
    case "create":
        cmdCreate()
    }
}

// Detected structure:
// Commands:
//   - list (no args)
//   - get (requires ID flag)
//   - create (requires name, optional description)
```

### Complete Analysis
```bash
# Analyze a CLI tool
mcp-reflect analyze kubectl

# Output:
Tool: kubectl
Commands:
  - get
    Input: {resource: string, name?: string, namespace?: string}
    Output: {apiVersion: string, kind: string, metadata: object, spec: object}
  - create
    Input: {file?: string, resource?: object}
    Output: {created: boolean, resource: object}
  - delete
    Input: {resource: string, name: string, namespace?: string}
    Output: {deleted: boolean}

# Generate MCP server
mcp-reflect generate kubectl > kubectl-mcp-server.go
```

## 🔍 Advanced Detection Patterns

### Complex Output Detection
```go
// Detect indirect JSON output
func processCommand() {
    result := compute()
    output := formatAsJSON(result) // Follow function calls
    writeToStdout(output)         // Track data flow
}

// Detect encoder patterns
encoder := json.NewEncoder(os.Stdout)
encoder.SetIndent("", "  ")
encoder.Encode(data)

// Detect template-based output
tmpl.Execute(os.Stdout, data) // Where template produces JSON
```

### Nested Subcommand Detection
```go
// Detect multi-level commands like "kubectl get pods"
rootCmd.AddCommand(getCmd)
getCmd.AddCommand(podsCmd)

// Detect command groups
cmds := &CommandGroup{
    Name: "cluster",
    Commands: []Command{
        {Name: "info"},
        {Name: "status"},
    },
}
```

### Dynamic Schema Detection
```go
// Detect schemas that vary based on flags
if verbose {
    output.Details = getDetails()
}

// Detect conditional fields
type Output struct {
    Basic  BasicInfo  `json:"basic"`
    Extra  *ExtraInfo `json:"extra,omitempty"`
}
```

This advanced reflection system can:
1. Detect JSON marshaling patterns to stdout
2. Analyze subcommand structures in various CLI frameworks
3. Extract input schemas from flags
4. Generate accurate output schemas
5. Handle complex patterns and edge cases
6. Create complete MCP tool definitions automatically

The key insight is using SSA (Static Single Assignment) analysis to trace data flow and detect when JSON-marshaled data reaches stdout, combined with AST pattern matching for subcommand detection.