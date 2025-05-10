# Guide for LLMs: Writing Tests with rsc.io/script/scripttest

This guide provides structured instructions for an LLM to create effective tests using the `rsc.io/script/scripttest` package. Follow this framework to generate Go tests that verify command-line application behavior.

## 1. Understanding scripttest Architecture

When writing tests using scripttest, focus on:

- **Engine**: Interprets commands and conditions
- **State**: Maintains environment, working directory, and captured outputs
- **txtar format**: Combines script code and test data in a single file
- **Commands and conditions**: Control script execution flow

## 2. Test Script Structure Framework

Script test files (typically .txt) should follow this structure:

```
# [Comment describing the test]
[command to run]
[verification command]

# [Comment for next test case]
[command to run]
[verification command]

-- [filename] --
[file content]
```

Example patterns to follow:

```
# Test basic functionality
myapp argument
stdout "Expected output"

# Test with file input
myapp -f input.txt
cmp stdout expected_output.txt

# Test error cases
! myapp invalid
stderr "Error message"

-- input.txt --
Test input data
-- expected_output.txt --
Expected output data
```

## 3. Go Test Function Template

Use this template to implement the Go test function:

```go
package myapp_test

import (
    "context"
    "os"
    "testing"
    
    "rsc.io/script"
    "rsc.io/script/scripttest"
)

func TestMyApp(t *testing.T) {
    // Create engine with default commands and conditions
    engine := script.NewEngine()
    for k, v := range scripttest.DefaultCmds() {
        engine.Cmds[k] = v
    }
    for k, v := range scripttest.DefaultConds() {
        engine.Conds[k] = v
    }
    
    // Add custom commands if needed
    engine.Cmds["run"] = script.Command(
        script.CmdUsage{
            Summary: "run the application",
            Args:    "[args...]",
        },
        func(s *script.State, args ...string) (script.WaitFunc, error) {
            // Implementation to run the application
            return s.Exec("./myapp", args...)
        },
    )
    
    // Set environment and run tests
    env := []string{"PATH=" + os.Getenv("PATH")}
    scripttest.Test(t, context.Background(), engine, env, "testdata/*.txt")
}
```

## 4. Core Commands Reference

Include these essential commands in test scripts:

- **Application execution**: `myapp [args]` or custom command like `run [args]`
- **Output verification**:
  - `stdout "pattern"` - Check stdout contains pattern
  - `stderr "pattern"` - Check stderr contains pattern 
  - `cmp stdout file.txt` - Compare stdout with file content
  - `cmp stderr file.txt` - Compare stderr with file content
- **File operations**:
  - `cat file.txt` - Display file content
  - `exists file.txt` - Verify file exists
  - `echo "content" > file.txt` - Create file with content
- **Flow control**:
  - `!` prefix - Expect command to fail
  - `?` prefix - Ignore command errors
  - `[condition] command` - Conditional execution
  - `skip "reason"` - Skip test with reason

## 5. Test Case Generation Strategy

When generating test cases, systematically cover:

1. **Happy path scenarios** - Normal, expected usage
2. **Edge cases** - Boundary conditions and special inputs
3. **Error handling** - Invalid inputs and expected errors
4. **Environment variations** - Different platform behaviors

## 6. Comprehensive Example: Command-Line Parser Tool

Let's create a complete test suite for a command-line argument parser tool called `clparse`:

```
# Test help command
clparse --help
stdout "Usage: clparse \[options\] \[arguments\]"
stdout "Options:"
stdout "  --format=FORMAT"
stdout "  --output=FILE"

# Test version display
clparse --version
stdout "clparse v[0-9]\.[0-9]\.[0-9]"

# Test basic argument parsing (default format)
clparse arg1 arg2 "argument with spaces"
stdout "Arguments:"
stdout "1: arg1"
stdout "2: arg2"
stdout "3: argument with spaces"

# Test JSON output format
clparse --format=json arg1 arg2
cmp stdout expected.json

# Test YAML output format
clparse --format=yaml arg1 arg2
cmp stdout expected.yaml

# Test XML output format
clparse --format=xml arg1 arg2
cmp stdout expected.xml

# Test writing output to a file
clparse --output=output.txt arg1 arg2
! stdout .
stderr "Output written to output.txt"
exists output.txt
cat output.txt
stdout "Arguments:"
stdout "1: arg1"
stdout "2: arg2"

# Test invalid format
! clparse --format=invalid arg1
stderr "Error: Invalid format 'invalid'"
stderr "Valid formats: text, json, yaml, xml"
exit 1

# Test file output when file already exists (should overwrite)
echo "Old content" > existing.txt
clparse --output=existing.txt arg1
exists existing.txt
cat existing.txt
stdout "Arguments:"
stdout "1: arg1"
! stdout "Old content"

# Test file output with unwritable path
[unix] mkdir -p test_dir
[unix] chmod 000 test_dir
[unix] ! clparse --output=test_dir/output.txt arg1
[unix] stderr "Error: Unable to write to file"
[unix] chmod 755 test_dir

# Test file output to directory (should fail)
[unix] mkdir -p test_dir2
[unix] ! clparse --output=test_dir2 arg1
[unix] stderr "Error: Cannot write to directory"

# Test with environment variables
env CLPARSE_FORMAT=json
clparse arg1
cmp stdout expected_single.json

# Test with empty arguments
clparse
stdout "No arguments provided"

# Test with quoted and escaped arguments
clparse "quoted \"argument\"" "escaped\\backslash"
stdout "Arguments:"
stdout "1: quoted \"argument\""
stdout "2: escaped\\backslash"

# Test with numeric arguments
clparse 123 456 -789
stdout "Arguments:"
stdout "1: 123"
stdout "2: 456"
stdout "3: -789"

# Test with Unicode arguments
clparse "español" "中文" "🚀"
stdout "Arguments:"
stdout "1: español"
stdout "2: 中文"
stdout "3: 🚀"

# Test input from stdin
echo "stdin input" | clparse --stdin
stdout "Arguments:"
stdout "1: stdin input"

# Test input from stdin with args
echo "stdin input" | clparse --stdin arg1
stdout "Arguments:"
stdout "1: stdin input"
stdout "2: arg1"

# Test very long argument
clparse "$(echo 'x' | head -c 10000)"
stdout "Arguments:"
stdout "1: xxxxx"

-- expected.json --
{
  "arguments": [
    "arg1",
    "arg2"
  ]
}

-- expected.yaml --
arguments:
  - arg1
  - arg2

-- expected.xml --
<arguments>
  <argument>arg1</argument>
  <argument>arg2</argument>
</arguments>

-- expected_single.json --
{
  "arguments": [
    "arg1"
  ]
}
```

## 7. Custom Command Implementation Examples

Here are examples of different types of custom commands for testing specialized behaviors:

### Simple Output Command

```go
engine.Cmds["generate"] = script.Command(
    script.CmdUsage{
        Summary: "Generate test data",
        Args:    "[type] [count]",
    },
    func(s *script.State, args ...string) (script.WaitFunc, error) {
        if len(args) != 2 {
            return nil, script.ErrUsage
        }
        
        dataType := args[0]
        count, err := strconv.Atoi(args[1])
        if err != nil {
            return nil, fmt.Errorf("invalid count: %v", err)
        }
        
        var result strings.Builder
        switch dataType {
        case "number":
            for i := 0; i < count; i++ {
                result.WriteString(fmt.Sprintf("%d\n", i))
            }
        case "uuid":
            for i := 0; i < count; i++ {
                result.WriteString(fmt.Sprintf("test-uuid-%d\n", i))
            }
        default:
            return nil, fmt.Errorf("unknown data type: %s", dataType)
        }
        
        s.WriteStdout([]byte(result.String()))
        return nil, nil
    },
)
```

### Database Interaction Command

```go
engine.Cmds["db"] = script.Command(
    script.CmdUsage{
        Summary: "Interact with test database",
        Args:    "[action] [params...]",
    },
    func(s *script.State, args ...string) (script.WaitFunc, error) {
        if len(args) < 1 {
            return nil, script.ErrUsage
        }
        
        dbPath := filepath.Join(s.Getwd(), "test.db")
        action := args[0]
        
        switch action {
        case "init":
            // Initialize test database
            db, err := sql.Open("sqlite3", dbPath)
            if err != nil {
                return nil, err
            }
            defer db.Close()
            
            _, err = db.Exec("CREATE TABLE IF NOT EXISTS items (id INTEGER PRIMARY KEY, name TEXT)")
            if err != nil {
                return nil, err
            }
            
            s.WriteStdout([]byte("Database initialized\n"))
            return nil, nil
            
        case "insert":
            if len(args) < 2 {
                return nil, fmt.Errorf("missing name parameter")
            }
            
            db, err := sql.Open("sqlite3", dbPath)
            if err != nil {
                return nil, err
            }
            defer db.Close()
            
            res, err := db.Exec("INSERT INTO items (name) VALUES (?)", args[1])
            if err != nil {
                return nil, err
            }
            
            id, _ := res.LastInsertId()
            s.WriteStdout([]byte(fmt.Sprintf("Inserted item with ID: %d\n", id)))
            return nil, nil
            
        case "list":
            db, err := sql.Open("sqlite3", dbPath)
            if err != nil {
                return nil, err
            }
            defer db.Close()
            
            rows, err := db.Query("SELECT id, name FROM items ORDER BY id")
            if err != nil {
                return nil, err
            }
            defer rows.Close()
            
            var result strings.Builder
            for rows.Next() {
                var id int
                var name string
                if err := rows.Scan(&id, &name); err != nil {
                    return nil, err
                }
                result.WriteString(fmt.Sprintf("%d: %s\n", id, name))
            }
            
            if result.Len() == 0 {
                result.WriteString("No items found\n")
            }
            
            s.WriteStdout([]byte(result.String()))
            return nil, nil
            
        default:
            return nil, fmt.Errorf("unknown action: %s", action)
        }
    },
)
```

### HTTP Server Command

```go
engine.Cmds["httpserver"] = script.Command(
    script.CmdUsage{
        Summary: "Start a test HTTP server",
        Args:    "[port]",
    },
    func(s *script.State, args ...string) (script.WaitFunc, error) {
        if len(args) != 1 {
            return nil, script.ErrUsage
        }
        
        port := args[0]
        
        // Channel to signal server is ready
        ready := make(chan struct{})
        
        // Start HTTP server in a goroutine
        go func() {
            mux := http.NewServeMux()
            
            // Add test endpoints
            mux.HandleFunc("/api/items", func(w http.ResponseWriter, r *http.Request) {
                w.Header().Set("Content-Type", "application/json")
                fmt.Fprintf(w, `[{"id":1,"name":"Item 1"},{"id":2,"name":"Item 2"}]`)
            })
            
            mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
                w.Header().Set("Content-Type", "application/json")
                fmt.Fprintf(w, `{"status":"ok","version":"1.0.0"}`)
            })
            
            server := &http.Server{
                Addr:    "127.0.0.1:" + port,
                Handler: mux,
            }
            
            // Signal that the server is ready
            close(ready)
            
            // Context for shutdown
            ctx := s.Context()
            
            // Start server
            go server.ListenAndServe()
            
            // Wait for context to be done and then shutdown
            <-ctx.Done()
            server.Shutdown(context.Background())
        }()
        
        // Wait for server to be ready
        <-ready
        
        // Store the port in an environment variable
        s.Setenv("HTTP_SERVER_PORT", port)
        s.WriteStdout([]byte(fmt.Sprintf("Server started on port %s\n", port)))
        
        // Return a wait func that returns an error when the context is done
        return func(s *script.State) (string, string, error) {
            <-s.Context().Done()
            return "", "", nil
        }, nil
    },
)
```

## 8. Real-World Example: Complex CLI Tool

Here's a comprehensive test for a configuration management CLI that supports multiple file formats, templates, and validation:

```
# Test help command
configctl --help
stdout "Usage: configctl COMMAND \[OPTIONS\]"
stdout "Commands:"
stdout "  init"
stdout "  validate"
stdout "  apply"
stdout "  diff"
stdout "  template"

# Test initialization
configctl init --type=app
stdout "Created default configuration in config.yaml"
exists config.yaml
cat config.yaml
stdout "app:"
stdout "  name: myapp"
stdout "  environment: development"

# Test initialization with specific format
configctl init --type=app --format=json
stdout "Created default configuration in config.json"
exists config.json

# Test template rendering
configctl template --input=config.yaml --template=template.tpl --output=output.conf
stdout "Template rendered to output.conf"
exists output.conf
cat output.conf
stdout "APP_NAME=myapp"
stdout "APP_ENV=development"

# Test validation of valid config
configctl validate config.yaml
stdout "Configuration is valid"

# Test validation of invalid config
echo "app:\n  badfield: value" > invalid.yaml
! configctl validate invalid.yaml
stderr "Error: Missing required field 'name'"

# Test validation with schema
configctl validate config.yaml --schema=schema.json
stdout "Configuration is valid"

# Test validation with invalid schema
! configctl validate config.yaml --schema=invalid_schema.json
stderr "Error: Could not parse schema file"

# Test diff between configurations
configctl diff config.yaml modified.yaml
stdout "Differences found:"
stdout "+.*environment: production"
stdout "-.*environment: development"

# Test diff with no differences
cp config.yaml same.yaml
configctl diff config.yaml same.yaml
stdout "No differences found"

# Test apply configuration
configctl apply config.yaml
stdout "Configuration applied successfully"
stdout "Environment: development"
exists ".configctl.state"

# Test apply with dry run
configctl apply modified.yaml --dry-run
stdout "Dry run: Configuration would be applied with these changes:"
stdout "environment: development -> production"
! exists ".configctl.state.new"

# Test apply with variables
configctl apply config.yaml --var="app.port=8080" --var="app.timeout=30s"
stdout "Configuration applied successfully"
stdout "Port: 8080"
stdout "Timeout: 30s"

# Test apply with variables file
cat > vars.yaml <<EOF
app:
  port: 9090
  host: localhost
EOF
configctl apply config.yaml --vars-file=vars.yaml
stdout "Configuration applied successfully"
stdout "Port: 9090"
stdout "Host: localhost"

# Test apply with invalid variables format
echo "invalid-format" > invalid_vars.yaml
! configctl apply config.yaml --vars-file=invalid_vars.yaml
stderr "Error: Could not parse variables file"

# Test export to different format
configctl export config.yaml --format=json --output=exported.json
stdout "Exported configuration to exported.json"
exists exported.json
cat exported.json
stdout '{\s*"app":\s*{\s*"name":\s*"myapp"'

# Test export to standard output
configctl export config.yaml --format=json
stdout '{\s*"app":\s*{\s*"name":\s*"myapp"'

# Test environment-specific configurations
env CONFIGCTL_ENV=production
configctl apply config.yaml
stdout "Configuration applied successfully"
stdout "Using production environment"

# Test with profile
configctl apply config.yaml --profile=staging
stdout "Configuration applied successfully"
stdout "Using staging profile"

# Test concurrent apply (should fail with lock)
configctl apply config.yaml --lock-timeout=1 &
sleep 1
! configctl apply modified.yaml --lock-timeout=1
stderr "Error: Configuration is locked by another process"
wait

-- template.tpl --
APP_NAME={{.app.name}}
APP_ENV={{.app.environment}}

-- schema.json --
{
  "type": "object",
  "required": ["app"],
  "properties": {
    "app": {
      "type": "object",
      "required": ["name", "environment"],
      "properties": {
        "name": {"type": "string"},
        "environment": {"type": "string"}
      }
    }
  }
}

-- invalid_schema.json --
{
  "type": "object",
  "required": ["app",
  "properties": {
    "app": {
      "type": "object"
    }
  }
}

-- modified.yaml --
app:
  name: myapp
  environment: production
```

## 9. Debugging Failed Tests

When a scripttest test fails, follow these debugging steps:

1. **Examine logs**: Look at the full test output to see the exact command that failed and why.

2. **Inspect environment**: Check if environment variables are set correctly and paths are properly constructed.

3. **Verify file paths**: Ensure the test is looking for files in the correct working directory.

4. **Check exit codes**: Some failures might be due to unexpected exit codes rather than output.

5. **Use verbose mode**: Run with `-v` flag to see detailed command execution.

6. **Trace custom commands**: Add logging to custom command implementations to trace execution.

7. **Simplify test cases**: Break down complex tests into smaller, focused tests to isolate issues.

8. **Test regex patterns**: Verify that your stdout/stderr regex patterns correctly match expected outputs.

## 10. Handling Cross-Platform Tests

To make tests work across different platforms:

```
# Create a directory (platform-specific approach)
[windows] mkdir testdir
[!windows] mkdir -p testdir

# Use appropriate path separators
[windows] exists testdir\\file.txt
[!windows] exists testdir/file.txt

# Check for platform-specific executables
[exec:powershell] powershell -Command "Write-Host 'Hello from PowerShell'"
[exec:bash] bash -c "echo 'Hello from Bash'"

# Skip tests that don't apply to current platform
[windows] skip "Test not applicable on Windows"

# Handle different line endings
[windows] cmp stdout win_expected.txt
[!windows] cmp stdout unix_expected.txt
```

Remember these key points:
- Use platform detection conditions liberally
- Provide platform-specific test files when needed
- Be careful with path separators and line endings
- Consider using the `skip` command for platform-specific tests rather than attempting complex workarounds