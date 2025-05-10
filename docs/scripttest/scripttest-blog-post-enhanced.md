# Mastering Command-Line Testing with rsc.io/script/scripttest

Testing command-line applications presents unique challenges. Setting up environments, running commands, capturing outputs, and verifying results often require complex, hard-to-maintain test code. The `rsc.io/script` and `rsc.io/script/scripttest` packages provide an elegant solution by offering a declarative, readable approach to testing command-line tools.

This blog post explores how to leverage scripttest to create powerful integration tests for your command-line applications.

## What is rsc.io/script/scripttest?

The `scripttest` package adapts the `rsc.io/script` engine specifically for Go's testing package. It provides:

1. A simple yet powerful scripting language for test automation
2. Integration with Go's `testing` package
3. Platform-agnostic command execution and verification
4. A declarative approach to defining test scenarios

## Key Concepts

Understanding the following concepts is crucial to effectively using scripttest:

- **Engine**: Interprets and executes script commands and conditions
- **State**: Manages the working directory, environment, and command outputs
- **Commands**: Actions performed by scripts (e.g., running programs, comparing files)
- **Conditions**: Logic that determines whether commands should execute
- **txtar format**: Combines script code and data files in a single text file

## Getting Started

Let's walk through a simple example to demonstrate how scripttest works:

### 1. Setting Up Dependencies

First, install the required packages:

```bash
go get rsc.io/script
go get rsc.io/script/scripttest
go get golang.org/x/tools/txtar
```

### 2. Writing a Basic Test Script

Create a test script file (e.g., `testdata/basic.txt`) in txtar format:

```
# Test a simple echo command
echo "Hello, World!"
stdout "Hello, World!"

# Test file creation and content verification
echo "test content" > test.txt
cat test.txt
stdout "test content"

# Create a data file for the test to use
-- input.txt --
This is input data
```

The script above:
- Runs `echo` and verifies its output
- Creates a file and checks its content
- Includes a data file (`input.txt`) that will be created in the test directory

### 3. Implementing the Go Test Function

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
	// Create an engine with default commands and conditions
	engine := &script.Engine{
		Cmds:  scripttest.DefaultCmds(),
		Conds: scripttest.DefaultConds(),
	}

	// Use the host environment variables
	env := os.Environ()

	// Run all test scripts in the testdata directory
	scripttest.Test(t, context.Background(), engine, env, "testdata/*.txt")
}
```

This function:
1. Creates a script engine with default commands and conditions
2. Uses the host environment
3. Runs all `.txt` files in the `testdata` directory as test scripts

## Practical Example 1: Testing a JSON Processor

Let's create a test for a hypothetical CLI tool called `jsonproc` that processes JSON files:

### Test Script (testdata/jsonproc_test.txt):

```
# Test basic JSON transformation
jsonproc transform input.json
cmp stdout expected_output.json
! stderr .

# Test JSON validation of valid file
jsonproc validate valid.json
stdout "JSON is valid"

# Test JSON validation of invalid file
! jsonproc validate invalid.json
stderr "Error: Invalid JSON"

# Test with unsupported operation
! jsonproc unsupported input.json
stderr "Error: Unsupported operation"

# Test with missing file
! jsonproc transform missing.json
stderr "Error: File not found"

-- input.json --
{
  "name": "John Doe",
  "age": 30,
  "email": "john@example.com"
}

-- expected_output.json --
{
  "user": {
    "fullName": "John Doe",
    "age": 30,
    "contact": "john@example.com"
  }
}

-- valid.json --
{
  "valid": true,
  "numbers": [1, 2, 3]
}

-- invalid.json --
{
  "invalid": true,
  "unclosed": "string
}
```

### Go Test Implementation:

```go
package jsonproc_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"rsc.io/script"
	"rsc.io/script/scripttest"
)

func TestJsonProc(t *testing.T) {
	// Find the jsonproc executable
	executable, err := exec.LookPath("jsonproc")
	if err != nil {
		// Fall back to local build if not in PATH
		executable = "./jsonproc"
	}

	// Create an engine with default commands and conditions
	engine := script.NewEngine()
	for k, v := range scripttest.DefaultCmds() {
		engine.Cmds[k] = v
	}
	for k, v := range scripttest.DefaultConds() {
		engine.Conds[k] = v
	}

	// Add a custom command for jsonproc
	engine.Cmds["jsonproc"] = script.Program(executable, nil, 0)

	// Use the host environment variables
	env := os.Environ()

	// Run all test scripts
	scripttest.Test(t, context.Background(), engine, env, "testdata/*.txt")
}
```

## Practical Example 2: Testing an API Client CLI

Let's test an API client tool that interacts with a REST service. We'll use a mock server for testing:

### Test Script (testdata/apiclient_test.txt):

```
# Start a mock server in the background
mockserver 8080 &
sleep 1

# Test successful GET request
apiclient get users
stdout '{"users":\[{"id":1,"name":"User 1"},{"id":2,"name":"User 2"}\]'

# Test filtered GET request
apiclient get users --filter="name=User 1"
stdout '{"users":\[{"id":1,"name":"User 1"}\]'

# Test creating a resource
apiclient create user --data=create_user.json
stdout '{"id":3,"name":"New User","status":"created"}'

# Test error handling (resource not found)
! apiclient get nonexistent
stderr "Error: Resource not found"
exit 1

# Test authentication failure
! apiclient --token=invalid get users
stderr "Error: Authentication failed"
exit 2

# Stop the mock server
kill mockserver

-- create_user.json --
{
  "name": "New User",
  "email": "new@example.com"
}

-- mockserver.responses --
GET /users -> 200 {"users":[{"id":1,"name":"User 1"},{"id":2,"name":"User 2"}]}
GET /users?filter=name=User%201 -> 200 {"users":[{"id":1,"name":"User 1"}]}
POST /user -> 201 {"id":3,"name":"New User","status":"created"}
GET /nonexistent -> 404 {"error":"Resource not found"}
```

### Go Test Implementation:

```go
package apiclient_test

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"rsc.io/script"
	"rsc.io/script/scripttest"
)

func TestApiClient(t *testing.T) {
	// Create an engine with default commands and conditions
	engine := script.NewEngine()
	for k, v := range scripttest.DefaultCmds() {
		engine.Cmds[k] = v
	}
	for k, v := range scripttest.DefaultConds() {
		engine.Conds[k] = v
	}

	// Add apiclient command
	engine.Cmds["apiclient"] = script.Program("./apiclient", nil, 0)

	// Add mock server command
	engine.Cmds["mockserver"] = script.Command(
		script.CmdUsage{
			Summary: "Start a mock HTTP server",
			Args:    "port",
		},
		func(s *script.State, args ...string) (script.WaitFunc, error) {
			if len(args) != 1 {
				return nil, script.ErrUsage
			}
			
			// Start mock server on specified port
			cmd := exec.CommandContext(s.Context(), "./mockserver", args[0])
			cmd.Dir = s.Getwd()
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				return nil, err
			}
			stderr, err := cmd.StderrPipe()
			if err != nil {
				return nil, err
			}
			if err := cmd.Start(); err != nil {
				return nil, err
			}
			
			// Save PID for kill command
			s.Setenv("MOCKSERVER_PID", fmt.Sprintf("%d", cmd.Process.Pid))
			
			return script.ReadOutput(stdout, stderr, func() error {
				return cmd.Wait()
			}), nil
		},
	)

	// Add kill command to stop mock server
	engine.Cmds["kill"] = script.Command(
		script.CmdUsage{
			Summary: "Kill a background process",
			Args:    "process",
		},
		func(s *script.State, args ...string) (script.WaitFunc, error) {
			if len(args) != 1 || args[0] != "mockserver" {
				return nil, script.ErrUsage
			}
			
			// Get PID from environment
			pid, ok := s.LookupEnv("MOCKSERVER_PID")
			if !ok {
				return nil, fmt.Errorf("no mockserver running")
			}
			
			// Parse PID
			pidInt, err := strconv.Atoi(pid)
			if err != nil {
				return nil, err
			}
			
			// Kill process
			process, err := os.FindProcess(pidInt)
			if err != nil {
				return nil, err
			}
			if err := process.Signal(os.Interrupt); err != nil {
				return nil, err
			}
			
			return nil, nil
		},
	)

	// Use the host environment variables
	env := os.Environ()

	// Run all test scripts
	scripttest.Test(t, context.Background(), engine, env, "testdata/*.txt")
}
```

## Practical Example 3: Testing a File Processing Pipeline

Let's test a pipeline of commands that process log files:

### Test Script (testdata/logprocessor_test.txt):

```
# Test parsing and filtering a log file

# First generate a test log file with timestamp patterns
echo "2021-01-01 12:00:00 INFO  Message 1" > input.log
echo "2021-01-01 12:01:00 ERROR Error message" >> input.log
echo "2021-01-01 12:02:00 WARN  Warning message" >> input.log
echo "2021-01-01 12:03:00 INFO  Message 2" >> input.log

# Filter only ERROR entries
logproc filter --level=ERROR input.log
stdout "2021-01-01 12:01:00 ERROR Error message"
! stdout "INFO"
! stdout "WARN"

# Count entries by log level
logproc count-by-level input.log
stdout "INFO: 2"
stdout "ERROR: 1"
stdout "WARN: 1"

# Export to JSON format
logproc export --format=json input.log
cmp stdout expected.json

# Time range filtering
logproc filter --from="2021-01-01 12:01:00" --to="2021-01-01 12:02:00" input.log
stdout "2021-01-01 12:01:00 ERROR Error message"
stdout "2021-01-01 12:02:00 WARN  Warning message"
! stdout "12:00:00"
! stdout "12:03:00"

# Test error handling for invalid time format
! logproc filter --from="invalid-time" input.log
stderr "Error: Invalid time format"

# Test processing a non-existent file
! logproc filter nonexistent.log
stderr "Error: File not found"

-- expected.json --
[
  {
    "timestamp": "2021-01-01 12:00:00",
    "level": "INFO",
    "message": "Message 1"
  },
  {
    "timestamp": "2021-01-01 12:01:00",
    "level": "ERROR",
    "message": "Error message"
  },
  {
    "timestamp": "2021-01-01 12:02:00",
    "level": "WARN",
    "message": "Warning message"
  },
  {
    "timestamp": "2021-01-01 12:03:00",
    "level": "INFO",
    "message": "Message 2"
  }
]
```

### Go Test Implementation:

```go
package logproc_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"rsc.io/script"
	"rsc.io/script/scripttest"
)

func TestLogProcessor(t *testing.T) {
	// Locate the logproc binary
	executable := "./logproc"
	if path, err := exec.LookPath("logproc"); err == nil {
		executable = path
	}

	// Create an engine with default commands and conditions
	engine := script.NewEngine()
	for k, v := range scripttest.DefaultCmds() {
		engine.Cmds[k] = v
	}
	for k, v := range scripttest.DefaultConds() {
		engine.Conds[k] = v
	}

	// Add logproc command
	engine.Cmds["logproc"] = script.Program(executable, nil, 0)

	// Use the host environment variables
	env := os.Environ()

	// Run all test scripts
	scripttest.Test(t, context.Background(), engine, env, "testdata/logprocessor_test.txt")
}
```

## Practical Example 4: Testing with External Dependencies

Test a CLI tool that requires external dependencies like a database:

### Test Script (testdata/dbcli_test.txt):

```
# Skip if sqlite3 is not available
[!exec:sqlite3] skip "sqlite3 not found in PATH"

# Initialize a test database
sqlite3 test.db < schema.sql

# Test creating records
dbcli --db test.db create user --name="John Doe" --email="john@example.com"
stdout "User created with ID: 1"

dbcli --db test.db create user --name="Jane Smith" --email="jane@example.com"
stdout "User created with ID: 2"

# Test listing records
dbcli --db test.db list users
stdout "ID\s+Name\s+Email"
stdout "1\s+John Doe\s+john@example.com"
stdout "2\s+Jane Smith\s+jane@example.com"

# Test querying specific records
dbcli --db test.db get user 1
stdout "ID: 1"
stdout "Name: John Doe"
stdout "Email: john@example.com"

# Test record not found
! dbcli --db test.db get user 999
stderr "Error: User not found with ID 999"

# Test updating records
dbcli --db test.db update user 1 --name="John Updated"
stdout "User updated successfully"

dbcli --db test.db get user 1
stdout "Name: John Updated"

# Test deleting records
dbcli --db test.db delete user 2
stdout "User deleted successfully"

dbcli --db test.db list users
stdout "John Updated"
! stdout "Jane Smith"

# Test transaction rollback on error
! dbcli --db test.db create user --name="Invalid" --invalid-flag
stderr "Error: Unknown flag: --invalid-flag"

# Create a user with the same email to test unique constraint
dbcli --db test.db create user --name="Another John" --email="john@example.com"
! dbcli --db test.db create user --name="Duplicate Email" --email="john@example.com"
stderr "Error: Email already exists"

# Clean up
rm test.db

-- schema.sql --
CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  email TEXT UNIQUE NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX users_email_idx ON users(email);
```

### Go Test Implementation:

```go
package dbcli_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"rsc.io/script"
	"rsc.io/script/scripttest"
)

func TestDatabaseCLI(t *testing.T) {
	// Create an engine with default commands and conditions
	engine := script.NewEngine()
	for k, v := range scripttest.DefaultCmds() {
		engine.Cmds[k] = v
	}
	for k, v := range scripttest.DefaultConds() {
		engine.Conds[k] = v
	}

	// Add dbcli command
	engine.Cmds["dbcli"] = script.Program("./dbcli", nil, 0)

	// Add sqlite3 command if it exists in PATH
	if sqlitePath, err := exec.LookPath("sqlite3"); err == nil {
		engine.Cmds["sqlite3"] = script.Program(sqlitePath, nil, 0)
	}

	// Use the host environment variables
	env := os.Environ()

	// Run all test scripts
	scripttest.Test(t, context.Background(), engine, env, "testdata/dbcli_test.txt")
}
```

## Core Script Commands

Scripttest provides various built-in commands:

- **Command execution**: Running any command (`ls`, `grep`, etc.)
- **`stdout`/`stderr`**: Verifying command output against patterns
- **`cmp`**: Comparing output or files
- **`exists`**: Checking if files or directories exist
- **`env`**: Setting environment variables
- **`cat`**: Displaying file contents
- **`cp`/`mv`/`rm`**: File operations
- **`skip`**: Skipping a test with optional message

## Script Conditions

Conditions determine whether commands execute:

- **OS conditions**: `[linux]`, `[darwin]`, `[windows]`
- **Architecture conditions**: `[amd64]`, `[arm64]`, etc.
- **Go test conditions**: `[short]`, `[verbose]`
- **Executable conditions**: `[exec:command]` (checks if command exists in PATH)

For example:

```
[windows] echo "Windows only"
[exec:go] go version
[short] skip "Skipping in short mode"
```

## Real-World Example: Testing a Multi-Command CLI

Let's examine how to test a CLI application with multiple subcommands:

### Test Script (testdata/multicmd.txt):

```
# Test the help command
toolkit help
stdout "Available commands:"
stdout "  config"
stdout "  process"
stdout "  validate"

# Test config listing
toolkit config list
stdout "No configurations found"

# Test config creation
toolkit config create --name="default" --value="test value"
stdout "Configuration 'default' created successfully"

# Test config retrieval
toolkit config get default
stdout "test value"

# Test invalid config name
! toolkit config get nonexistent
stderr "Error: Configuration 'nonexistent' not found"
exit 1

# Test processing with valid configuration
echo "sample data" > input.txt
toolkit process input.txt --config=default
stdout "Processing complete"
exists output.txt

# Validate the output
toolkit validate output.txt
stdout "Validation passed"

# Test processing with invalid input
! toolkit process nonexistent.txt --config=default
stderr "Error: Cannot read input file"

# Test processing with invalid configuration
! toolkit process input.txt --config=nonexistent
stderr "Error: Configuration 'nonexistent' not found"

# Test version command
toolkit version
stdout "toolkit v[0-9]+\.[0-9]+\.[0-9]+"
```

### Go Test Implementation:

```go
package toolkit_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"rsc.io/script"
	"rsc.io/script/scripttest"
)

func TestToolkit(t *testing.T) {
	// Find toolkit executable
	executable := "./toolkit"
	if path, err := exec.LookPath("toolkit"); err == nil {
		executable = path
	}

	// Create an engine with default commands and conditions
	engine := script.NewEngine()
	for k, v := range scripttest.DefaultCmds() {
		engine.Cmds[k] = v
	}
	for k, v := range scripttest.DefaultConds() {
		engine.Conds[k] = v
	}

	// Add toolkit command
	engine.Cmds["toolkit"] = script.Program(executable, nil, 0)

	// Set up temporary home directory for config files
	env := append(os.Environ(), "TOOLKIT_HOME=${WORK}/toolkit-home")

	// Run all test scripts
	scripttest.Test(t, context.Background(), engine, env, "testdata/multicmd.txt")
}
```

## Advanced Patterns

### Custom Commands

You can extend scripttest with custom commands:

```go
engine.Cmds["mycommand"] = script.Command(
	script.CmdUsage{
		Summary: "Execute a custom operation",
		Args:    "[args...]",
	},
	func(s *script.State, args ...string) (script.WaitFunc, error) {
		// Custom command implementation
		// ...
		return nil, nil
	},
)
```

### Custom Conditions

Similarly, you can add custom conditions:

```go
engine.Conds["mycondition"] = script.CachedCondition(
	"Description of the condition",
	func(suffix string) (bool, error) {
		// Custom condition logic
		return true, nil
	},
)
```

### Background Commands

Run commands in the background with the `&` suffix:

```
# Start a server in the background
server &
sleep 1
curl localhost:8080
wait
```

### Error Handling

Use prefixes to control error handling:

- **`!`**: Command must fail (test passes if command fails)
- **`?`**: Ignore errors (continue even if command fails)

```
# This should fail, and we expect that
! invalid_command
# Continue even if this fails
? might_fail
```

## Best Practices

1. **Keep scripts focused**: Each script should test a specific aspect of your application
2. **Use descriptive comments**: Document your tests with clear comments
3. **Take advantage of the txtar format**: Store test data alongside scripts
4. **Test edge cases**: Test invalid inputs, special cases, and error conditions
5. **Use conditions for platform-specific tests**: Make your tests portable across systems
6. **Create custom commands for readability**: Abstract complex operations into simple commands
7. **Clean up resources**: Ensure your tests clean up any created resources

## When to Use scripttest

Scripttest is particularly useful for:

- **Integration testing**: Verifying components work together
- **Command-line tools**: Testing CLI applications
- **Cross-platform testing**: Ensuring your code works on different systems
- **End-to-end testing**: Validating complete workflows
- **Environment-sensitive testing**: Testing with specific environment configurations

## Conclusion

The `rsc.io/script/scripttest` package provides a powerful, declarative approach to testing command-line applications. By adopting a script-based testing strategy, you can create more readable, maintainable, and comprehensive tests for your CLI tools.

The combination of simple syntax, powerful commands, and flexible conditions makes scripttest an excellent choice for Go developers who want to ensure their command-line applications function correctly in various environments.

Start leveraging scripttest in your projects today to create more robust and reliable command-line applications!

## Further Reading

- [rsc.io/script GitHub repository](https://github.com/rogpeppe/go-internal/tree/master/txtar)
- [txtar package documentation](https://pkg.go.dev/golang.org/x/tools/txtar)
- [Go testing package documentation](https://pkg.go.dev/testing)