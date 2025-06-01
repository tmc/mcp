This is a fascinating and very Plan 9-esque direction! You're essentially thinking about service discovery and process context in a hierarchical way, much like Plan 9's per-process namespaces and `/srv`.

Let's break down how `mcp-send` (which I'll interpret as a generic MCP client tool for sending one-off requests, perhaps we can call it `mcp-rpc` or `mcp-call` to distinguish from the trace file prefix) could discover the appropriate server socket managed by an `mcpd`.

**Method 1: Environment Variables (Simplest, Most Common)**

This is the most straightforward and common Unix/scripttest way.

*   **`mcpd` Behavior:** When `mcpd` starts, it determines/creates its listening Unix socket (e.g., `/tmp/.mcpd-1234.sock`). It then exports this path as an environment variable, say `MCP_SOCKET_PATH=/tmp/.mcpd-1234.sock`, *for the child processes it launches within a `scripttest` environment or for the current shell if run interactively*.
*   **`mcp-rpc` (or `mcp-send`) Behavior:**
    *   Looks for `MCP_SOCKET_PATH` in its environment.
    *   If found, connects to that socket.
    *   Optionally, could have a `-socket-path <path>` flag to override or use if the env var is not set.
*   **In `scripttest`:**
    ```
    # Start mcpd, it sets MCP_SOCKET_PATH for subsequent commands in this script
    exec mcpd -socket-env MCP_SOCKET_PATH -- ./my-server &
    waitsocket $MCP_SOCKET_PATH

    # mcp-rpc automatically uses $MCP_SOCKET_PATH
    exec mcp-rpc '{"method":"do"}'
    stdout '{"result":"done"}'
    ```

**Method 2: PID Hierarchy and Well-Known Relative Paths (More Plan 9-like)**

This is more complex but very much in the spirit of Plan 9's process namespaces.

*   **`mcpd` Behavior:**
    1.  When `mcpd` starts, it creates its listening Unix socket in a well-known *relative* location based on its own PID or a PID of a specific "session leader" process.
    2.  For example, under `/tmp/mcpns/` (MCP NameSpace) or `~/.mcpns/`:
        *   `/tmp/mcpns/<pid_of_mcpd>/service.sock`
        *   Or, if we have a concept of a "session leader" PID (e.g., the PID of the shell running the tests): `/tmp/mcpns/<session_pid>/<service_name_or_mcpd_pid>.sock`
*   **`mcp-rpc` Behavior:**
    1.  To discover the socket, `mcp-rpc` would need to:
        *   Traverse up its parent process tree (`getppid()`) to find a known `mcpd` PID or a "session leader" PID that might have an associated `mcpd`. This is tricky and not entirely reliable across all OSes without specific kernel support or a session manager.
        *   Alternatively, `mcp-rpc` could look for a socket in `/tmp/mcpns/<its_own_pid>/...` and then traverse upwards, assuming an `mcpd` in its parentage might have created a socket relevant to its child's PID.
    2.  **A more practical PID-hierarchy approach for `scripttest`:**
        *   `mcpd` writes its PID and socket path to a known file in `$WORK`.
        *   `mcp-rpc` (if run as `exec mcp-rpc ...` by `scripttest`) could check `$WORK` for this info. This is less about PID hierarchy and more about shared state within the test's working directory.

**Method 3: Service File System (`/srv`-like, True Plan 9 Style)**

This is the most "pure" Plan 9 approach but requires more infrastructure or a specific convention.

*   **`mcpd` Behavior:**
    1.  `mcpd` creates a service file in a conventional location. This location could be:
        *   A fixed path: `~/.srv/mcp/my_service_instance`
        *   A path relative to the current process's "namespace" (Plan 9 has this built-in; on Unix, we'd have to simulate it, perhaps using `$MCP_NAMESPACE_ROOT` env var).
    2.  The service file's *content* is the actual path to the Unix domain socket.
    *   Example: `mcpd` creates `~/.srv/mcp/weather_test` which contains the string `/tmp/.mcpd-1234.sock`.
*   **`mcp-rpc` Behavior:**
    1.  `mcp-rpc -service weather_test` would look for `~/.srv/mcp/weather_test`.
    2.  It reads the content of that file to get the socket path.
    3.  Connects to that socket.
*   **Nesting/Graph Structure:**
    *   If one `mcpd` (parent-mcpd) starts another `mcpd` (child-mcpd), the child-mcpd could announce its service *within a subdirectory* managed by the parent-mcpd.
    *   Parent-mcpd listens on `/tmp/parent.sock` and creates `~/.srv/mcp/parent_service`.
    *   Client requests `parent_service/child_service_request`.
    *   Parent-mcpd sees `child_service_request`, knows it needs to consult/start `child-mcpd`, which has announced itself in, say, `~/.srv/mcp/parent_service/child_mcpd_socket_file`.
    *   This creates a mount-like structure. `mcpd` could even implement a simple FUSE filesystem to expose these services.

**Simplifying for Unix/Linux without full Plan 9 Namespace Support:**

Given standard Unix/Linux, **Method 1 (Environment Variables)** is the most idiomatic and easiest to implement robustly for `scripttest`.

**For the "Nesting/Graph" idea:**

This is where `mcpd` truly shines as a more advanced component. One `mcpd` instance could indeed act as a frontend that routes or dispatches to other `mcpd` instances or directly to backend servers.

*   **Parent `mcpd` (Router/Dispatcher):**
    *   Listens on a primary Unix socket (e.g., `/tmp/main_mcp_router.sock`).
    *   Has a configuration (or dynamic registration mechanism) that maps parts of the MCP method path or a special parameter to downstream `mcpd` instances or server commands.
    *   Example request to router: `{"method": "serviceA/doWork", "params": {...}}`
    *   Router `mcpd` sees `serviceA`, looks up its config for `serviceA`, finds it's managed by another `mcpd` listening on `/tmp/mcpd_serviceA.sock` (or needs to start it).
    *   It then proxies the request to `/tmp/mcpd_serviceA.sock`.

*   **Child `mcpd` (Service Instance Manager):**
    *   Started by the parent `mcpd` (or manually, registering with the parent).
    *   Listens on its own socket (e.g., `/tmp/mcpd_serviceA.sock`).
    *   Manages the actual `<server_command>` via stdin/stdout.

**How `mcp-send` (or `mcp-rpc`) discovers in this nested scenario:**

1.  **Explicit Pathing in Method:** The client might send a request like `{"method": "path/to/service/actualMethod", ...}` to the root `mcpd`. The root `mcpd` parses the path to route.
2.  **Service Discovery Protocol with Root `mcpd`:** The client first queries the root `mcpd` for a list of available services and their endpoints (which could be other `mcpd` sockets).
3.  **Environment Variables for Local Context:** If a shell or script is "attached" to a specific part of the graph (i.e., a specific child `mcpd` has set an environment variable like `MCP_ACTIVE_SOCKET`), then `mcp-rpc` run in that shell would use that specific socket. Different shells could have different `MCP_ACTIVE_SOCKET` values pointing to different `mcpd` instances in the graph.

**Recommendation for "Shell Attachment" and Discovery:**

Let's combine Environment Variables with a convention for `mcpd` socket naming if no explicit path is given.

1.  **`mcpd` Default Socket Naming:**
    *   If no `-listen-uds` is given, `mcpd` creates a socket: `~/.mcpd/sock.<PID>` (or in `$XDG_RUNTIME_DIR/mcpd/sock.<PID>`).
    *   It also creates a symlink: `~/.mcpd/current -> sock.<PID>`. This `current` symlink always points to the *most recently started interactive `mcpd` by that user*.
    *   When `mcpd` starts, it can also set `MCP_SOCKET_PATH=~/.mcpd/sock.<PID>` for itself and its children.

2.  **`mcp-rpc` Discovery Logic:**
    *   **Priority 1: `-unix-socket <path>` flag.** Explicitly use this path.
    *   **Priority 2: `MCP_SOCKET_PATH` environment variable.** If set, use it. This is useful for scripting where a parent process (like `scripttest` or a specific `mcpd` starting a sub-shell) sets the context.
    *   **Priority 3: `~/.mcpd/current` symlink.** If it exists and points to a valid socket, use it. This targets the "current interactive" `mcpd`.
    *   **Error:** If none of the above, `mcp-rpc` fails, stating no MCP endpoint is defined.

3.  **Nesting `mcpd`s:**
    *   A parent `mcpd` can start a child `mcpd`.
    *   The parent `mcpd` would pass the child `mcpd` a specific `-listen-uds` path for the child to use (or let the child auto-generate and report it back).
    *   The parent `mcpd` would then know how to route to the child.
    *   If a user starts an interactive shell "within" the context of a child `mcpd`, that child `mcpd` could set `MCP_SOCKET_PATH` for that shell, overriding any `MCP_SOCKET_PATH` from the parent `mcpd`.

**`mcp-attach <service-name|pid|socket-path>` (New Conceptual Tool):**
This tool could be used to "attach" a new shell session to a specific `mcpd` instance.
*   `mcp-attach myweather`: Looks for a service named `myweather` (e.g., via `~/.srv/mcp/myweather`).
*   `mcp-attach 12345`: Attaches to `mcpd` with PID 12345 (looking for `~/.mcpd/sock.12345`).
*   It would start a new shell with `MCP_SOCKET_PATH` set to the target `mcpd`'s socket.

This creates a flexible system:
*   Simple cases use default discovery (`~/.mcpd/current`).
*   Scripted/nested cases use environment variables (`MCP_SOCKET_PATH`).
*   Users can explicitly target with flags.

The "graph-like structure" becomes a matter of how `mcpd` instances are configured to forward requests to other `mcpd` instances or backend services, and how clients (or `mcp-rpc`) discover the entry points into this graph (initially, likely a root `mcpd` or a well-known service name).

This feels very Plan 9 – services are files (sockets), context is inherited (env vars), and tools compose.
Yes, absolutely! That's a fantastic idea and aligns very well with the interactive, file-system-centric nature of Plan 9 and the capabilities of Unix terminals. You're essentially describing a way for the MCP system (via `mcpd` and a client tool) to "pause" a script or automated flow and request live, human input through the terminal.

Here's how this could be designed, combining `mcpd` with a terminal interaction tool, let's call it `mcp-prompt`:

**1. Special MCP Method for Requesting Human Input**

Define a special MCP method that a server (managed by `mcpd`) can "send" (i.e., write to its stdout, which `mcpd` intercepts) to indicate it needs human input.

*   **Method Name:** e.g., `interactive/promptUser`
*   **Params:**
    *   `prompt_message`: The string to display to the user.
    *   `input_id`: A unique ID for this input request, so the response can be correlated.
    *   `input_type`: (Optional) e.g., "text", "password", "confirm" (y/n).
    *   `default_value`: (Optional) A default value for the prompt.
    *   `timeout_seconds`: (Optional) How long to wait for input before the server might proceed with a default or error.

**Example message from server to `mcpd`:**
```json
{"jsonrpc":"2.0", "method":"interactive/promptUser", "params": {"prompt_message": "Enter API Key:", "input_id": "api_key_request_123", "input_type": "password"}}
```
This is logged by `mcpd` as an `mcp-send` (from server's perspective).

**2. `mcpd` Behavior on Receiving `interactive/promptUser`**

When `mcpd` (which is logging the server's stdout) sees this special `interactive/promptUser` message:

1.  **Log it:** Records the `mcp-send interactive/promptUser ...` message to its `.mcp` trace file.
2.  **Signal/Communicate with `mcp-prompt`:**
    *   `mcpd` forwards the `interactive/promptUser` request to a dedicated communication channel that `mcp-prompt` is listening on. This could be:
        *   A special Unix domain socket (`MCP_PROMPT_REQUEST_SOCKET`).
        *   A named pipe (FIFO).
3.  **Wait for Response:** `mcpd` then *pauses* forwarding I/O to/from the *actual client* that initiated the original request which led to this prompt. It waits for a response specifically for this `interactive/promptUser` request from `mcp-prompt`.

**3. `mcp-prompt` - Terminal Interaction Tool**

This tool runs in the user's "attached" terminal session.

*   **Purpose:** Listens for `interactive/promptUser` requests from `mcpd` and interacts with the TTY to get human input.
*   **Listening:** Connects to `MCP_PROMPT_REQUEST_SOCKET` or reads from the FIFO.
*   **Interaction:**
    1.  When it receives an `interactive/promptUser` message:
        *   It clears any current TTY input line (if possible, or just prints a newline).
        *   **Displays the prompt:** `[MCP PROMPT - my_service_instance - ID: api_key_request_123] Enter API Key:`
        *   **Handles input type:**
            *   `text`: Normal TTY read.
            *   `password`: Uses `golang.org/x/term` to read password without echoing.
            *   `confirm`: Reads y/n, potentially with a loop for valid input.
        *   Rings the terminal bell (`\a`) to get attention.
    *   **Sends Response back to `mcpd`:** After getting input, `mcp-prompt` sends an MCP message back to `mcpd` on a response channel (could be the same socket if bidirectional, or a reply pipe/socket).
        **Example response from `mcp-prompt` to `mcpd`:**
        ```json
        {"jsonrpc":"2.0", "result": {"input_id": "api_key_request_123", "user_input": "SECRET_KEY_VALUE"}}
        ```
        Or if the user cancels (e.g., Ctrl+C at the prompt, if `mcp-prompt` handles it):
        ```json
        {"jsonrpc":"2.0", "error": {"input_id": "api_key_request_123", "code": -32000, "message": "User cancelled input"}}
        ```

**4. `mcpd` Behavior on Receiving Response from `mcp-prompt`**

1.  `mcpd` receives the response from `mcp-prompt`.
2.  **Formats it as an MCP request to the backend server:**
    `mcpd` now takes the `user_input` and constructs a *new* MCP request that it sends to the backend server's stdin, as if it were a regular client request. This new request would typically use a method the server expects for providing this interactive input, or it could be a generic `interactive/userInput` method.
    **Example message from `mcpd` to server's stdin:**
    ```json
    {"jsonrpc":"2.0", "id": "input_resp_for_123", "method":"interactive/userInput", "params": {"original_input_id": "api_key_request_123", "value": "SECRET_KEY_VALUE"}}
    ```
3.  **Log it:** Records this `mcp-recv interactive/userInput ...` message to its `.mcp` trace file.
4.  **Resume:** `mcpd` then resumes normal forwarding of I/O for the original client connection. The backend server processes the `interactive/userInput` and continues its operation, eventually sending its actual response to the original client (which `mcpd` will then log and forward).

**Analogy to `script -r` (and `typescript` files):**

*   The `.mcp` file produced by `mcpd` becomes somewhat like a `typescript` file from the `script` command.
*   `mcpreplay` can play back this `.mcp` file. If `mcpreplay` encounters an `mcp-send interactive/promptUser ...` line (which was originally from the server), it would:
    *   **Option A (Non-interactive replay):** Look for the *next* `mcp-recv interactive/userInput ...` line in the recording (which was originally from `mcpd` forwarding `mcp-prompt`'s input) and use that payload directly. This allows fully automated replay of previously recorded interactive sessions.
    *   **Option B (Interactive replay with `mcpreplay -interactive`):** `mcpreplay` itself could invoke `mcp-prompt` (or have similar TTY logic) to ask the user for input again during replay, allowing the user to provide *different* answers during this specific replay.

**Workflow Example:**

1.  User has a shell attached via `mcp-attach some_service`. `mcp-prompt` is running.
2.  User runs a command: `mcp-rpc -s $MCP_SOCKET_PATH '{"method":"deployApp", "params":{...}}'`
3.  The `my_app_server` (managed by `mcpd`) needs an API key. It sends to its stdout:
    `{"jsonrpc":"2.0", "method":"interactive/promptUser", "params": {"prompt_message": "Enter API Key for deployment:", "input_id": "deploy_api_key_001"}}`
4.  `mcpd`:
    *   Logs: `mcp-send {"jsonrpc":"2.0", "method":"interactive/promptUser", ...} # ts1`
    *   Sends this to `mcp-prompt` via `MCP_PROMPT_REQUEST_SOCKET`.
    *   Waits.
5.  `mcp-prompt`:
    *   Rings bell.
    *   Displays: `[MCP PROMPT - some_service - ID: deploy_api_key_001] Enter API Key for deployment:`
    *   User types `mysecretkey`.
    *   Sends back to `mcpd`: `{"jsonrpc":"2.0", "result": {"input_id": "deploy_api_key_001", "user_input": "mysecretkey"}}`
6.  `mcpd`:
    *   Receives response from `mcp-prompt`.
    *   Sends to `my_app_server`'s stdin: `{"jsonrpc":"2.0", "id": "mcpd_input_1", "method":"interactive/userInput", "params": {"original_input_id": "deploy_api_key_001", "value": "mysecretkey"}}`
    *   Logs: `mcp-recv {"jsonrpc":"2.0", "id": "mcpd_input_1", "method":"interactive/userInput", ...} # ts2`
7.  `my_app_server` receives the API key, continues deployment, and eventually sends its final response for the `deployApp` method.
8.  `mcpd` logs and forwards this final response to the original `mcp-rpc` client.

**Benefits:**

*   **Interactive Debugging/Operation:** Allows human intervention in automated MCP flows.
*   **Recordable Interactions:** The human input and the prompt itself are captured in the `.mcp` trace.
*   **Testable Interactivity:** `mcpreplay` can automate these recorded interactive sessions.
*   **Separation of Concerns:** The server just signals it needs input; `mcpd` and `mcp-prompt` handle the TTY interaction.

This makes the system much more powerful for scenarios requiring occasional human input without breaking the overall automation and record/replay paradigm.Okay, this is a fantastic set of requirements that really leans into a Plan 9-inspired, tool-based workflow! Let's flesh out `mcpd` as the central piece for managing these interactive, proxied server sessions, and `mcp-prompt` as the TTY interaction handler.

## `mcpd` - MCP Daemon & Interactive Proxy

**Purpose:**
`mcpd` acts as a daemon to manage an underlying MCP server command. It exposes this server via a Unix domain socket (default) or TCP. `mcpd` logs all traffic, and crucially, it can intercept special "interactive prompt" requests from the server, delegate them to a TTY via `mcp-prompt`, and feed the user's response back to the server. This allows for sessions similar to `script -r` where a recorded session can include human interaction.

**Invocation:**

```bash
mcpd [mcpd-flags] -- <server_command> [server_args...]
```
The `--` is mandatory to separate `mcpd` flags from the server command.

**Communication Endpoints Managed by `mcpd`:**

1.  **Service Socket:**
    *   `-listen-uds <path>`: Unix domain socket path `mcpd` listens on for client connections.
        *   **Default:** Auto-generates in `os.TempDir()`, e.g., `/tmp/mcpd-svc-<PID>.sock`.
        *   The actual path used is printed to `mcpd`'s stdout on startup.
    *   `-listen-tcp <host:port>`: (Optional) Alternative TCP listen address.
2.  **Prompting Channel (FIFOs by default for simplicity):**
    *   `mcpd` creates two named pipes (FIFOs) for interaction with `mcp-prompt`:
        *   `<base_socket_name>.prompt-req.fifo`: `mcpd` writes `interactive/promptUser` requests here.
        *   `<base_socket_name>.prompt-resp.fifo`: `mcpd` reads user responses from `mcp-prompt` here.
    *   The paths to these FIFOs are also printed to `mcpd`'s stdout on startup.

**Output of `mcpd` on Startup (to its stdout):**
```
SERVICE_SOCKET=/tmp/mcpd-svc-12345.sock
PROMPT_REQ_FIFO=/tmp/mcpd-svc-12345.sock.prompt-req.fifo
PROMPT_RESP_FIFO=/tmp/mcpd-svc-12345.sock.prompt-resp.fifo
SERVER_PID=12346
```
This output is easily parsable by `scripttest` or wrapper scripts.

**Flags for `mcpd`:**

*   `-listen-uds <path>`: As above.
*   `-listen-tcp <host:port>`: As above.
*   `-log-file <path>`: Path to write the `.mcp` recording of interactions. **This is the primary state store.** Default: `mcpd-log-<PID>.mcp`.
*   `-server-stderr-log-file <path>`: (Optional) Redirect backend server's stderr here.
*   `-server-pid-file <path>`: (Optional) Write backend server's PID here (useful if `mcpd` doesn't print it).
*   `-mode <once|per-connection>`:
    *   `once` (default): Start `<server_command>` once. `mcpd` handles concurrent client connections by serializing requests to the single server instance if it's stdio-based.
    *   `per-connection`: Start a new server instance for each client.
*   `-v`: Verbose logging for `mcpd`'s own operations to its stderr.
*   `-vv`: Very verbose, logs raw data flow (client<->mcpd, mcpd<->server) to its stderr.

**Special MCP Methods Handled by `mcpd` <-> `mcp-prompt`:**

1.  **Server to `mcpd` (via server's stdout):**
    *   Method: `interactive/promptUser`
    *   Params:
        ```json
        {
          "prompt_message": "Enter API Key:",
          "input_id": "unique_req_id_from_server_001", // Server generates this
          "input_type": "text" | "password" | "confirm_yn", // Optional
          "default_value": "optional_default",            // Optional
          "timeout_seconds": 30                          // Optional
        }
        ```
    *   This is logged by `mcpd` as `mcp-send` (from server's perspective).

2.  **`mcpd` to `mcp-prompt` (via prompt-req FIFO):**
    *   `mcpd` writes the *params object* of the `interactive/promptUser` request as a JSON line to the FIFO.

3.  **`mcp-prompt` to `mcpd` (via prompt-resp FIFO):**
    *   After TTY interaction, `mcp-prompt` writes a JSON line:
        ```json
        {
          "input_id": "unique_req_id_from_server_001", // Echoed from prompt
          "status": "ok" | "cancelled" | "timeout",    // Status of user interaction
          "value": "user_typed_this",                 // If status="ok"
          "error_message": "User pressed Ctrl+C"      // If status!="ok"
        }
        ```

4.  **`mcpd` to Server (via server's stdin):**
    *   `mcpd` constructs an MCP request to deliver the user's input (or error) back to the server.
    *   Method: `interactive/userInput`
    *   Params:
        ```json
        {
          "original_input_id": "unique_req_id_from_server_001", // The ID from server's prompt
          "status": "ok" | "cancelled" | "timeout",
          "value": "user_typed_this",                // If status="ok"
          "error_message": "User pressed Ctrl+C"     // If status!="ok"
        }
        ```
    *   This is logged by `mcpd` as `mcp-recv` (from server's perspective).

**`mcpd` Behavior Flow:**

1.  **Startup:**
    *   Parse flags.
    *   Create service socket (UDS/TCP) and prompt FIFOs.
    *   Print endpoint information to stdout.
    *   If `mode` is `once`, start the `<server_command>`. Pipe its stdin/stdout. Redirect its stderr if specified.
2.  **Listen Loop:**
    *   Accept client connection on service socket.
    *   If `mode` is `per-connection`, start a new `<server_command>` instance for this client.
    *   Handle client connection in a new goroutine.
3.  **Client Connection Goroutine:**
    *   **Client -> Server:**
        *   Read MCP request from client socket.
        *   Log as `mcp-recv ... # timestamp` to `-log-file`.
        *   If `-vv`, print raw to `mcpd`'s stderr: `#> ...`
        *   Send to backend server's stdin.
    *   **Server -> Client (and Prompt Handling):**
        *   Read MCP response/message from server's stdout.
        *   Log as `mcp-send ... # timestamp` to `-log-file`.
        *   If `-vv`, print raw to `mcpd`'s stderr: `#< ...`
        *   **Check if it's `interactive/promptUser`:**
            *   If yes:
                1.  Extract `params` from this server message.
                2.  Write these `params` (as JSON) to the `prompt-req.fifo`.
                3.  Open `prompt-resp.fifo` for reading (with timeout based on `params.timeout_seconds` or a default).
                4.  Read the JSON response from `mcp-prompt`.
                5.  If timeout or error reading from FIFO, construct an error `interactive/userInput` message.
                6.  Construct the `interactive/userInput` request for the server using the data from `mcp-prompt`.
                7.  Log this new request as `mcp-recv interactive/userInput ... # timestamp`.
                8.  If `-vv`, print raw `interactive/userInput` as `#> ...`.
                9.  Send `interactive/userInput` to the server's stdin.
                10. *Do not* send the original `interactive/promptUser` message to the actual client. Instead, wait for the server's *next* response (which should be the response to the original client request that triggered the prompt).
            *   If no (it's a normal server response):
                1.  Forward to the client socket.

## `mcp-prompt` - TTY Interaction Tool

**Purpose:**
Listens for prompt requests from an `mcpd` instance (via FIFOs) and interacts with the user's TTY to get input.

**Invocation:**
Typically run in the background of an "attached" shell.
`mcp-prompt -req-fifo <path> -resp-fifo <path>`

**Flags:**

*   `-req-fifo <path>`: Path to the FIFO for receiving prompt requests from `mcpd`. (Required)
*   `-resp-fifo <path>`: Path to the FIFO for sending user input back to `mcpd`. (Required)
*   `-session-name <name>`: (Optional) Name of the service/session to display in prompts.

**Behavior:**

1.  Opens `req-fifo` for reading and `resp-fifo` for writing.
2.  Enters a loop:
    *   Blocks reading a JSON line from `req-fifo`. This JSON is the `params` object from an `interactive/promptUser` request.
    *   Parses the JSON: `prompt_message`, `input_id`, `input_type`, `default_value`.
    *   Rings terminal bell (`\a`).
    *   Displays prompt to user's TTY (stderr): `[MCP PROMPT - <session_name> - ID: <input_id>] <prompt_message>:`
    *   Reads user input based on `input_type`:
        *   `text`: Standard line read.
        *   `password`: Use `term.ReadPassword`.
        *   `confirm_yn`: Loop until 'y' or 'n' (case-insensitive), return "yes" or "no".
    *   Handles Ctrl+C (sends "cancelled" status) and Ctrl+D (EOF, could also be "cancelled").
    *   Constructs the response JSON: `{"input_id": ..., "status": ..., "value": ..., "error_message": ...}`.
    *   Writes this JSON response line to `resp-fifo`.

**Example `scripttest` using `mcpd` and simulating `mcp-prompt`:**

```txtar
# Test interactive prompt flow

-- mcpd_stdio_server.go --
package main
import ("bufio"; "fmt"; "os"; "encoding/json"; "log")
func main() {
    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        var req map[string]interface{}
        if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
            fmt.Fprintf(os.Stderr, "server: bad req: %v\n", err); continue
        }
        reqID := req["id"]
        method, _ := req["method"].(string)

        if method == "needsApiKey" {
            // Ask mcpd (via its stdout) to prompt the user
            promptReq := map[string]interface{}{
                "jsonrpc": "2.0",
                "method": "interactive/promptUser",
                "params": map[string]interface{}{
                    "prompt_message": "Enter API Key for needsApiKey:",
                    "input_id": fmt.Sprintf("api_key_for_%v", reqID),
                },
            }
            jsonPrompt, _ := json.Marshal(promptReq)
            fmt.Println(string(jsonPrompt))
            // Server now expects an interactive/userInput back
            continue 
        } else if method == "interactive/userInput" {
            params, _ := req["params"].(map[string]interface{})
            val, _ := params["value"].(string)
            origID, _ := params["original_input_id"].(string) // "api_key_for_X"
            // Here we'd find original_req_id from origID to use as response ID
            // For simplicity, assume we respond to the userInput req
            resp := map[string]interface{}{"jsonrpc":"2.0", "id":reqID, "result": "got api key: " + val}
            jsonResp, _ := json.Marshal(resp)
            fmt.Println(string(jsonResp))
            continue
        }
        
        // Default echo
        resp := map[string]interface{}{"jsonrpc":"2.0", "id":reqID, "result": req["params"]}
        jsonResp, _ := json.Marshal(resp)
        fmt.Println(string(jsonResp))
    }
}

# Build the server
exec go build -o $WORK/server $WORK/mcpd_stdio_server.go

# Start mcpd and get its endpoint info
env MCPD_ENDPOINTS_FILE=$WORK/mcpd_endpoints.env
exec mcpd -log-file $WORK/traffic.mcp -- $WORK/server > $MCPD_ENDPOINTS_FILE &
# mcpd_pid=$! # scripttest can manage this with 'stop'
. $MCPD_ENDPOINTS_FILE # source the env vars
[ -n "$SERVICE_SOCKET" ]
[ -n "$PROMPT_REQ_FIFO" ]
[ -n "$PROMPT_RESP_FIFO" ]
waitsocket $SERVICE_SOCKET
# We might need to wait for FIFOs too, or mcpd creates them before printing.

# Simulate mcp-prompt: read from PROMPT_REQ_FIFO, write to PROMPT_RESP_FIFO
# This part runs in background to respond to prompts from mcpd
exec sh -c ' \
  read -r prompt_json < "$PROMPT_REQ_FIFO"; \
  echo "mcp-prompt got: $prompt_json" >&2; \
  input_id=$(echo "$prompt_json" | jq -r .input_id); \
  echo "{\"input_id\":\"$input_id\", \"status\":\"ok\", \"value\":\"actual_api_key_from_prompt\"}" > "$PROMPT_RESP_FIFO" \
' &
prompt_sim_pid=$!

# Client sends a request that will trigger a prompt
exec echo '{"jsonrpc":"2.0","id":"client_req_001","method":"needsApiKey"}' | nc -U $SERVICE_SOCKET
# The server will send interactive/promptUser (logged by mcpd)
# mcpd will write to prompt-req.fifo
# prompt_sim.sh will read it, write "actual_api_key_from_prompt" to prompt-resp.fifo
# mcpd will read that, send interactive/userInput to server
# server will then send its final response.
stdout '{"jsonrpc":"2.0","id":"client_req_001_intermediate_after_prompt","result":"got api key: actual_api_key_from_prompt"}' # This needs server logic to match original ID

# Wait for the prompt simulator to finish (it reads one prompt)
wait $prompt_sim_pid

# Check mcpd's log
exec grep 'mcp-send {"jsonrpc":"2.0","method":"interactive/promptUser"' $WORK/traffic.mcp
exec grep 'mcp-recv {"jsonrpc":"2.0","method":"interactive/userInput","params":{"original_input_id":"api_key_for_client_req_001","status":"ok","value":"actual_api_key_from_prompt"}}' $WORK/traffic.mcp

# Stop mcpd
stop # stops the mcpd started with 'exec ... &'
# pkill -P $mcpd_pid # Or more specific cleanup if 'stop' isn't enough
```

This `mcpd` design is more complex but powerful for interactive scenarios and retains the Plan 9 spirit by using filesystem objects (sockets, FIFOs) for IPC and keeping the backend server simple (stdio-based). The trace files become invaluable for understanding these multi-step interactions.