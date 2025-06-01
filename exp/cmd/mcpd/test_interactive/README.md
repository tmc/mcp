# mcpd Interactive Mode and Signal Handling Tests

This directory contains test tools for verifying the interactive mode functionality and signal handling capabilities of mcpd.

## Components

- `server.go`: A test MCP server that sends interactive prompts
- `always_running_server.go`: A simple server that stays running indefinitely
- `test.sh`: A test script that runs the server with mcpd in interactive mode
- `test_server.sh`: Sets up an MCP server using socat, mcpspy, and the @modelcontextprotocol/server-everything backend
- `test_client.sh`: Connects to the test server and logs the interaction
- `test_signals.sh`: Tests mcpd signal handling by sending various signals

## Testing Interactive Mode

To run the interactive tests:

```sh
./test.sh
```

The test script:
1. Builds the test server and mcpd
2. Starts mcpd in interactive mode with the test server
3. Sends various test requests to trigger prompts
4. Requires manual input to respond to the prompts
5. Verifies the server continues to function after interactive prompts

## Testing Signal Handling

To test the new signal handling improvements:

1. Start the test server:
   ```sh
   ./test_server.sh
   ```

2. In another terminal, run the signal test script:
   ```sh
   ./test_signals.sh
   ```

3. Follow the prompts to send different signals to mcpd and observe how it handles them

## Alternative Test Using socat and mcpspy

You can also set up the test environment manually:

```sh
export SPYCMD="npx @modelcontextprotocol/server-everything stdio"
socat TCP-LISTEN:7000,fork,reuseaddr EXEC:"mcpspy -v -vv -- ${SPYCMD}"
```

Then in another terminal, connect to the server:

```sh
nc localhost 7000
```

## Test Server Features

The test server implements several methods to test interactive prompting:

- `echo`: Simple echo method for basic connectivity testing
- `prompt`: Sends a single interactive prompt
- `prompt_chain`: Sends a series of 3 prompts in sequence
- `interactive/userInput`: Handles user input responses from prompts

## Signal Handling Tests

The `test_signals.sh` script tests the following signal handling capabilities:

- **SIGWINCH**: Terminal window resize
- **SIGTSTP**: Terminal suspend (Ctrl+Z)
- **SIGCONT**: Continue after suspend
- **SIGINT**: Interrupt (Ctrl+C)
- **SIGTERM**: Termination signal
- **SIGHUP**: Hangup signal
- **SIGTTIN**: Terminal read from background
- **SIGTTOU**: Terminal write to background

## Test Output

When the tests run, you'll be prompted to enter input in the terminal. The server will receive this input via mcpd's interactive prompt handling and respond accordingly.

Log files are generated to record all interactions:
- `test_server.log`: Output from the test server
- `test_client.log`: Log of client interactions
- `mcpd.log`: Log output from mcpd during signal tests

## Cleanup

The test scripts include signal handling to ensure proper cleanup of processes, sockets, and temporary files on exit (even if interrupted).