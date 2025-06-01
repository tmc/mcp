/*
Mcpd manages and provides network access to MCP-compliant server commands
that communicate over stdin/stdout. It defaults to using Unix domain sockets
for client connections and logs all interactions to an MCP trace file.

Usage:
    mcpd [flags] -- <server_command> [server_args...]

The "--" is mandatory to separate mcpd's flags from the <server_command>.

Mcpd starts the <server_command>, making it accessible via a network endpoint
(Unix socket by default, or TCP). It pipes client requests to the server's
stdin and server responses from its stdout back to the client. All this traffic
is recorded in an MCP trace file (.mcp format).

On startup, mcpd prints its listening endpoint(s) (e.g., Unix socket path
or TCP host:port) to standard output, making it easy for test scripts or
other tools to discover how to connect.

Key Features:
  - Exposes stdio-based MCP server commands over Unix domain sockets (default) or TCP.
  - Automatically generates and manages unique socket paths.
  - Logs all client-server interactions to a specified .mcp trace file.
    This trace file serves as the primary record of the session.
  - Can manage the backend server process in different modes:
    - 'once' (default): Starts the server once; all clients connect to this instance.
    - 'per-connection': Starts a new server instance for each client connection.
  - Optional redirection of the backend server's own stderr.
  - PID file management for both mcpd and the managed server process.

Primary Use Case:
  Facilitating testing of MCP servers in isolated environments (like rsc.io/script)
  by abstracting network setup and providing detailed interaction logs.

Flags:
*/
package main