# Bash Command for mcpscripttest

The `bash` command has been added to mcpscripttest to allow running arbitrary bash commands within test scripts.

## Usage

```
bash 'command'
```

Runs a bash command with the `-c` flag. The command must be provided as a single quoted string argument.

## Examples

### Simple echo
```
bash 'echo "Hello from bash"'
stdout 'Hello from bash'
```

### Using environment variables
```
env TEST_VAR=World
bash 'echo "Hello $TEST_VAR"'
stdout 'Hello World'
```

### Piping commands
```
bash 'echo "one\ntwo\nthree" | grep two'
stdout 'two'
```

### Using stdin
```
setstdin 'input data'
bash 'cat'
stdout 'input data'
```

### Multiple commands
```
bash 'echo first; echo second'
stdout 'first'
stdout 'second'
```

### Error handling
```
! bash 'exit 1'
stderr
```

### File operations
```
bash 'echo "test content" > test.txt'
bash 'cat test.txt'
stdout 'test content'
bash 'rm test.txt'
```

## Implementation Details

- The command runs with a 30-second timeout
- If stdin is set using `setstdin`, it will be passed to the bash command
- The command inherits the test's working directory and environment
- Output is captured and returned appropriately (stdout for success, stderr for errors)
- The command supports all bash features including pipes, redirections, and command substitution

## Files Added

1. `bash_command.go` - Contains the implementation of the bash command and the missing mcpProbeCmd
2. `testdata/bash_command_test.txt` - Test file demonstrating bash command usage
3. `BASH_COMMAND.md` - This documentation file

## Integration

The bash command is registered in `scripttest.go` alongside other utility commands like `cat` and `setstdin`.