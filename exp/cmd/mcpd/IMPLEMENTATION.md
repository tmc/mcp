# mcpd Interactive Mode Implementation

This document describes the implementation of the interactive mode for `mcpd`.

## Overview

The interactive mode in `mcpd` allows the daemon to handle interactive prompts from MCP servers directly via the TTY. This creates a seamless experience similar to `script -r`, where a recorded interactive session can include human interaction.

## Architecture

The implementation follows these key components:

1. **Configuration Support**: Added `-i` and `-no-tty-prompt` flags to enable interactive mode and control TTY prompting behavior.

2. **Prompt Handler**: Created a dedicated `InteractivePromptHandler` in `transport/prompt.go` that:
   - Captures and parses `interactive/promptUser` requests from servers
   - Displays prompts to the user via terminal
   - Reads user input from stdin
   - Formats the response as an `interactive/userInput` message
   - Sends the response back to the server

3. **Transport Layer Integration**: Modified the connection handling to intercept `interactive/promptUser` messages and not forward them to clients when in interactive mode.

4. **Protocol Definition**: Defined a standard protocol for interactive prompting:
   - Server → Client: `interactive/promptUser` with parameters for prompt text, input type, etc.
   - Client → Server: `interactive/userInput` with the user's response

## Implementation Details

### 1. Configuration

Extended `config.Config` to include:
- `Interactive bool`: Flag indicating if interactive mode is enabled
- `NoTTYPrompt bool`: Flag to disable TTY prompting even in interactive mode

### 2. Prompt Handler

The `InteractivePromptHandler` in `transport/prompt.go` handles:
- TTY detection via `term.IsTerminal`
- Different input types (text, password, confirmation)
- Timeout handling
- Error reporting

### 3. Transport Layer Integration

Modified `transport/handler.go` to:
- Detect `interactive/promptUser` messages from servers
- Invoke the prompt handler to get user input
- Send responses back to the server
- Skip forwarding prompt requests to clients

### 4. Protocol Definition

Defined standard message formats:

**Server → mcpd (promptUser):**
```json
{
  "jsonrpc": "2.0",
  "method": "interactive/promptUser",
  "params": {
    "prompt_message": "Enter API Key:",
    "input_id": "unique_req_id_001",
    "input_type": "text|password|confirm_yn",
    "default_value": "optional_default",
    "timeout_seconds": 60
  }
}
```

**mcpd → Server (userInput):**
```json
{
  "jsonrpc": "2.0",
  "id": "prompt_response_001",
  "method": "interactive/userInput",
  "params": {
    "original_input_id": "unique_req_id_001",
    "status": "ok|cancelled|timeout",
    "value": "user_response",
    "error_message": "error details if status is not ok"
  }
}
```

## Testing

A test server (`test_interactive_server.go`) and test script (`test_interactive.sh`) were created to demonstrate the interactive mode.

The test server provides methods to:
- Send interactive prompts
- Chain multiple prompts together
- Process user responses

## Integration with mcp-attach

This implementation complements the `mcp-attach` tool, providing two ways to interact with MCP servers:

1. **Direct Interactive Mode (`mcpd -i`)**: For direct terminal interaction with a server
2. **Attachment Mode (`mcp-attach`)**: For connecting shell sessions to existing mcpd instances

## Future Enhancements

Potential future improvements:
1. More sophisticated terminal UI using a library like Bubble Tea
2. Support for more input types (selection lists, file selection, etc.)
3. Improved handling of terminal state and raw mode