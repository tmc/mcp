# MCPTrace Format Specification

## Overview

MCPTrace is a standardized file format for recording and replaying Model Context Protocol (MCP) interactions between clients and servers. This document describes the format specification, including the header structure, line format, timestamp normalization, and compatibility considerations.

## Version

Current format version: **v1**

## Header Structure

MCPTrace files begin with a header line that identifies the file format and version:

```
# mcptrace version 1
```

The header must appear as the first line in a new MCPTrace file and follows these conventions:
- Begins with `#` to indicate it's a comment/metadata line
- Contains `mcptrace version` followed by the version identifier (currently `1`)
- May contain additional key=value metadata pairs separated by spaces
- Is followed by a newline character (`\n`)

### Header with Metadata (Optional)

The header can include additional metadata:

```
# mcptrace version 1 compare=true
```

Common header metadata:
- `compare=true`: Indicates the trace contains shadow server comparisons

### OpenTelemetry (OTEL) Trace Context (Optional)

MCPTrace files can optionally include W3C OpenTelemetry trace context information in the header line. This allows MCPTrace files to participate in distributed tracing ecosystems and link MCP interactions with other system components.

Format with trace context:

```
# mcptrace version 1 traceparent=00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
```

Where:
- `traceparent` follows the W3C Trace Context specification format: `version-trace-id-parent-id-flags`
  - `version`: 2-hex digit version (usually `00`)
  - `trace-id`: 32-hex digit trace identifier
  - `parent-id`: 16-hex digit span identifier
  - `flags`: 2-hex digit flags (01 = sampled)

Additional trace context information can be included as well:

```
# mcptrace version 1 traceparent=00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01 tracestate=congo=t61rcWkgMzE
```

### Trace-Level Baggage (Optional)

Trace-level baggage can be added to the header to provide additional context that applies to the entire trace:

```
# mcptrace version 1 traceparent=00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01 baggage=environment=production,service=mcpserver
```

The baggage follows a simple key-value format:
- Key-value pairs are separated by commas
- No spaces between pairs
- Values that contain special characters should be URL-encoded

Tools supporting trace context should:
- Generate new trace IDs when recording MCP interactions
- Propagate existing trace context if available in the environment
- Preserve trace context when processing MCPTrace files
- Propagate baggage items when appropriate

## Line Format

After the header, each line in an MCPTrace file represents a single MCP message with the following format:

```
mcp-<direction> <json-content> # <timestamp>
```

Where:
- `<direction>` is one of:
  - `recv` for messages received by the server (client → server)
  - `send` for messages sent by the server (server → client)
  - `send-shadow` for messages sent by a shadow server (in compare mode)
- `<json-content>` is the complete JSON-RPC message
- `<timestamp>` is a Unix timestamp with millisecond precision in the format `<seconds>.<milliseconds>`

Example:
```
mcp-recv {"jsonrpc":"2.0","method":"initialize","id":1} # 1683000000.550
mcp-send {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"server":true}}} # 1683000001.120
mcp-send-shadow {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"server":true,"shadow":true}}} # 1683000001.121
```

Note: There is no `recv-shadow` direction since shadow servers receive identical input to primary servers.

### Per-Row Span Context (Optional)

Each line can optionally include OpenTelemetry span context information, allowing for detailed tracing of individual messages and linking between related messages. This is particularly valuable in traffic splitting/shadowing scenarios where messages are duplicated across multiple systems.

Extended format with per-row span:
```
mcp-<direction> <json-content> # <timestamp> spanid=<span-id> [linksto=<related-span-id>] [baggage=<key>=<value>]
```

Where:
- `spanid` is a 16-hex digit span identifier unique to this message
- `linksto` (optional) references another span ID that this message is related to
- `baggage` (optional) contains message-specific key-value pairs

Examples:

Simple span identifier:
```
mcp-recv {"jsonrpc":"2.0","method":"initialize","id":1} # 1683000000.550 spanid=00f067aa0ba902b7
```

Message in a shadow/split scenario linking to the original message:
```
mcp-recv {"jsonrpc":"2.0","method":"initialize","id":1} # 1683000000.550 spanid=a1b2c3d4e5f6a7b8 linksto=00f067aa0ba902b7
```

Message with additional context:
```
mcp-send {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"server":true}}} # 1683000001.120 spanid=00f067cc0ba902d9 baggage=priority=high,region=us-west-2
```

## Timestamp Format

Timestamps in MCPTrace files follow these rules:
- Format: `<seconds>.<milliseconds>`
- Seconds part: Integer value representing Unix time (seconds since epoch)
- Milliseconds part: Exactly 3 digits representing milliseconds (padded with zeros if needed)
- Example: `1683000000.550` means 1,683,000,000 seconds and 550 milliseconds after epoch

## Tools and Utilities

The MCP toolchain includes several utilities for working with MCPTrace files:

### mcpspy

Records MCP interactions and writes them to MCPTrace files:
- Automatically adds the `# mcptrace version 1` header to new files
- Records timestamps with millisecond precision
- Can operate in pipe mode, command mode, or as a standalone logger
- Can generate and include W3C OpenTelemetry trace context (optional)
- Supports per-row spans for message-level tracing

```bash
mcpspy -f output.mcp [command]                                # Record execution of [command]
mcpspy -pipe < input.json                                     # Pipe mode for recording stdin/stdout
mcpspy -trace -f output.mcp [command]                         # Record with new trace context
mcpspy -trace-parent $TRACEPARENT -f output.mcp               # Record and propagate existing trace context
mcpspy -trace -per-row-spans -f output.mcp [command]          # Generate span IDs for each message
mcpspy -baggage "env=prod,region=us-west" -f output.mcp       # Add trace-level baggage
mcpspy -shadow-mode -original-id abc123 -f shadow.mcp         # Record with links to original traffic
```

### mcp-shadow

Records primary and shadow server interactions for comparison:
- Runs both primary and shadow servers simultaneously
- Can output enhanced mcptrace format with `-compare` flag
- Records shadow responses with `send-shadow` direction
- Links shadow responses to primary responses using span metadata

```bash
mcp-shadow -primary "server1" -shadow "server2" > trace.mcp             # Basic mode
mcp-shadow -primary "server1" -shadow "server2" -compare > shadow.mcp   # Compare mode
```

### mcp-tsnorm

Normalizes timestamps in MCPTrace files:
- Preserves the mcptrace header (configurable with `-preserve-header`)
- Can rebase timestamps to start at a specified offset (`-start` flag)
- Can rebase to an absolute Unix timestamp (`-absolute` flag)
- Preserves W3C trace context when present in headers
- Maintains per-row span information

```bash
mcp-tsnorm -start 10s -o normalized.mcp input.mcp                  # Start at 10 seconds
mcp-tsnorm -absolute 1683000000 -o absolute.mcp input.mcp          # Use absolute timestamp
mcp-tsnorm -preserve-header=false -o noheader.mcp input.mcp        # Strip header
mcp-tsnorm -preserve-trace=false -o notrace.mcp input.mcp          # Remove trace context but keep header
mcp-tsnorm -preserve-spans=false -o nospans.mcp input.mcp          # Remove per-row span information
mcp-tsnorm -preserve-baggage=false -o nobaggage.mcp input.mcp      # Remove baggage but keep spans
mcp-tsnorm -add-baggage "region=eu-west" -o with-region.mcp input.mcp  # Add to existing baggage
```

### mcp-replay

Replays MCPTrace files to simulate MCP interactions:
- Supports header detection in mock-server mode
- Can replay at variable speeds with the `-speed` flag
- Supports various replay modes including mock client and mock server
- Can replay shadow responses with `-shadow` flag in mock-server mode
- Handles per-row spans and links between messages

```bash
mcp-replay -mock-server recording.mcp                    # Act as mock server
mcp-replay -mock-server -shadow shadow.mcp               # Replay shadow responses
mcp-replay -mock-client recording.mcp                    # Act as mock client
mcp-replay -speed 2.0 recording.mcp                      # Replay at 2x speed
mcp-replay -propagate-trace recording.mcp                # Make trace context available to child processes
mcp-replay -respect-spans recording.mcp                  # Use span information during replay
mcp-replay -compare-mode original.mcp shadow.mcp         # Compare linked messages between files
mcp-replay -export-otel recording.mcp                    # Export spans to OpenTelemetry collector
mcp-replay -filter "baggage.environment=production"      # Filter messages by baggage values
```

### mcpdiff

Compares MCP trace files to identify differences:
- Can compare two separate trace files
- Automatically detects and compares primary vs shadow responses in a single file
- Supports various comparison modes (timestamps, IDs, semantic JSON)
- Shows unified diff format output

```bash
mcpdiff trace1.mcp trace2.mcp                            # Compare two trace files
mcpdiff shadow-trace.mcp                                 # Auto-compare primary vs shadow
mcpdiff -t false trace1.mcp trace2.mcp                   # Include timestamps in comparison
mcpdiff -s trace1.mcp trace2.mcp                         # Semantic JSON comparison
```

## Pipeline Processing

MCPTrace files are designed to be processable in Unix-style pipelines:

```bash
mcpspy -pipe < input.json | mcp-tsnorm -start 5s | mcp-replay -mock-server
```

Tracing context can be propagated through the pipeline:

```bash
# Generate and propagate a new trace
mcpspy -trace -pipe < input.json | mcp-tsnorm -start 5s | mcp-replay -propagate-trace -mock-server

# Use an existing trace from the environment
mcpspy -trace-parent $TRACEPARENT -pipe < input.json | mcp-tsnorm | mcp-replay -propagate-trace
```

Advanced use cases with per-row spans and baggage:

```bash
# Generate spans for messages, then analyze in shadow mode
mcpspy -trace -per-row-spans -pipe < input.json > original.mcp
mcpspy -shadow-mode -original-id $(cat original.mcp | grep spanid | head -1 | cut -d= -f2) -pipe < input.json > shadow.mcp
mcp-replay -compare-mode original.mcp shadow.mcp -export-otel

# Traffic splitting with context propagation
mcpspy -trace -per-row-spans -baggage "split=true,test=v2" -pipe < input.json | tee original.mcp |
mcp-tsnorm -add-baggage "target=prod" | mcp-replay -mock-server &
cat original.mcp | mcp-tsnorm -add-baggage "target=canary" | mcp-replay -mock-server -port 8081 &

# Analyze differences between responses
cat original.mcp | grep spanid |
mcp-replay -filter "baggage.target=prod" original.mcp > prod-results.mcp
mcp-replay -filter "baggage.target=canary" original.mcp > canary-results.mcp
mcpdiff prod-results.mcp canary-results.mcp
```

## Compatibility Considerations

- **Header Detection**: Tools should check for and handle the header line, but should also work with legacy files that don't have headers
- **Version Evolution**: Future versions may introduce additional metadata lines starting with `#`
- **Backward Compatibility**: Newer tools should be able to read older format versions
- **Forward Compatibility**: When unknown format versions are encountered, tools should warn but attempt to process the file
- **Trace Context**: Tools should gracefully handle headers with or without trace context
- **Trace Context Propagation**: Tools should preserve trace context when processing files, but provide options to strip or modify it if needed

## Best Practices

1. **Always Include Headers**: New MCPTrace files should always include the format header
2. **Timestamp Precision**: Always use millisecond precision in timestamps
3. **Direction Consistency**: Use `mcp-recv` for client→server messages and `mcp-send` for server→client messages
4. **Avoid Timestamp Reordering**: Messages should generally be in timestamp order, though replay tools should handle out-of-order timestamps
5. **Include Complete Messages**: Each line should contain a complete, valid JSON-RPC message
6. **Trace Context Generation**: When creating new trace files, generate valid W3C trace context identifiers using a suitable algorithm (such as random UUIDs)
7. **Trace Context Propagation**: When operating in a larger system, respect existing trace context from the environment when available
8. **Trace Sampling**: Set the sampling flag appropriately based on your observability needs (typically "01" for sampled)

## Extension Mechanism

The MCPTrace format can be extended in future versions by:
1. Incrementing the version number (e.g., `# mcptrace:v2`)
2. Adding additional metadata lines (prefixed with `#`)
3. Enhancing the line format while maintaining backward compatibility

## Examples

### Complete MCPTrace file examples:

#### Basic Example (No Trace Context)
```
# mcptrace version 1
mcp-recv {"jsonrpc":"2.0","method":"initialize","id":1} # 1683000000.000
mcp-send {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"server":true}}} # 1683000000.550
mcp-recv {"jsonrpc":"2.0","method":"ping","id":2} # 1683000001.200
mcp-send {"jsonrpc":"2.0","id":2,"result":"pong"} # 1683000001.750
mcp-send {"method":"notifications/message","params":{"level":"info","data":"Test notification"},"jsonrpc":"2.0"} # 1683000002.100
mcp-recv {"jsonrpc":"2.0","method":"exit"} # 1683000003.000
```

#### With Trace Context in Header
```
# mcptrace version 1 traceparent=00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
mcp-recv {"jsonrpc":"2.0","method":"initialize","id":1} # 1683000000.000
mcp-send {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"server":true}}} # 1683000000.550
mcp-recv {"jsonrpc":"2.0","method":"ping","id":2} # 1683000001.200
mcp-send {"jsonrpc":"2.0","id":2,"result":"pong"} # 1683000001.750
mcp-send {"method":"notifications/message","params":{"level":"info","data":"Test notification"},"jsonrpc":"2.0"} # 1683000002.100
mcp-recv {"jsonrpc":"2.0","method":"exit"} # 1683000003.000
```

#### With Trace Context and Per-Row Spans
```
# mcptrace version 1 traceparent=00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01 baggage=environment=production
mcp-recv {"jsonrpc":"2.0","method":"initialize","id":1} # 1683000000.000 spanid=a1b2c3d4e5f6a7b8
mcp-send {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"server":true}}} # 1683000000.550 spanid=b2c3d4e5f6a7b8a9
mcp-recv {"jsonrpc":"2.0","method":"ping","id":2} # 1683000001.200 spanid=c3d4e5f6a7b8a9b0
mcp-send {"jsonrpc":"2.0","id":2,"result":"pong"} # 1683000001.750 spanid=d4e5f6a7b8a9b0c1
mcp-send {"method":"notifications/message","params":{"level":"info","data":"Test notification"},"jsonrpc":"2.0"} # 1683000002.100 spanid=e5f6a7b8a9b0c1d2
mcp-recv {"jsonrpc":"2.0","method":"exit"} # 1683000003.000 spanid=f6a7b8a9b0c1d2e3
```

#### Shadow Mode Comparison Example
```
# mcptrace version 1 compare=true
mcp-recv {"jsonrpc":"2.0","method":"initialize","id":1} # 1683000000.000 spanid=aaaaaaaaaaaaaaa1
mcp-send {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"server":true}}} # 1683000000.550 spanid=aaaaaaaaaaaaaaa2
mcp-send-shadow {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"server":true,"shadow":true}}} # 1683000000.560 spanid=bbbbbbbbbbbbbbbb2 linksto=aaaaaaaaaaaaaaa2
mcp-recv {"jsonrpc":"2.0","method":"execute","id":2,"params":{"code":"calculate(42)"}} # 1683000001.200 spanid=aaaaaaaaaaaaaaa3
mcp-send {"jsonrpc":"2.0","id":2,"result":{"value":84}} # 1683000001.750 spanid=aaaaaaaaaaaaaaa4
mcp-send-shadow {"jsonrpc":"2.0","id":2,"result":{"value":86}} # 1683000001.770 spanid=bbbbbbbbbbbbbbbb4 linksto=aaaaaaaaaaaaaaa4 baggage=diff=true
```

## Implementation Notes

1. **Header Writing**: When creating new files, tools should write the header before any other content and immediately flush to ensure it's written properly
2. **Header Checking**: When reading files, tools should check if the first line matches the header pattern and handle it appropriately
3. **File Processing**: Tools should be able to process both files with and without headers
4. **Timestamp Handling**: Tools should be able to handle timestamps with varying precision but normalize to millisecond precision (3 decimal places)
5. **Trace Context Generation**: When generating new traces, use a secure random source for trace ID generation to ensure uniqueness and prevent collisions
6. **Trace Context Parsing**: When reading trace context from headers, gracefully handle malformed trace context strings
7. **Trace Context Propagation**: Support environment variables like `TRACEPARENT` and `TRACESTATE` for trace context propagation between processes
8. **Trace Context Preservation**: When processing MCPTrace files, preserve the original trace context unless explicitly requested to modify or remove it
9. **Per-Row Span Generation**: When splitting or shadowing traffic, generate new span IDs for each message copy while linking back to the original
10. **Span Linking**: Use the `linksto` parameter to establish causal relationships between messages, especially in diagnostic or A/B testing scenarios
11. **Baggage Scoping**: Use header-level baggage for trace-wide context and per-row baggage for message-specific context
12. **Span Aggregation**: When processing MCPTrace files, tools should be able to aggregate related spans based on linking relationships

## Shadow Mode

Shadow mode enables comparing responses from multiple server implementations:

### Key Features
- Primary server handles actual client requests
- Shadow server receives identical input for comparison
- Shadow responses are recorded with `send-shadow` direction
- No `recv-shadow` entries (both servers get same input)
- Header includes `compare=true` metadata
- Shadow responses link to primary responses via `linksto` metadata

### Use Cases
1. **A/B Testing**: Compare behavior between different implementations
2. **Migration Validation**: Ensure new implementation matches old behavior
3. **Performance Comparison**: Measure response time differences
4. **Regression Testing**: Detect changes in behavior across versions

### Recording Shadow Traffic
```bash
# Using mcp-shadow tool
mcp-shadow -primary "server1" -shadow "server2" -compare > shadow.mcp

# Manual recording with mcpspy
mcpspy -f primary.mcp server1 &
mcpspy -f shadow.mcp -shadow-mode -original-trace primary.mcp server2
```

### Analyzing Shadow Traffic
```bash
# Compare primary vs shadow responses
mcpdiff shadow-trace.mcp                 # Auto-detects shadow mode

# Replay only shadow responses
mcp-replay -mock-server -shadow shadow.mcp

# Filter for differences
mcpdiff shadow.mcp | grep "diff=true"
```

### Shadow Mode Format Example
```
# mcptrace version 1 compare=true
mcp-recv {"jsonrpc":"2.0","method":"tools/list","id":1} # 1000.000 spanid=req1
mcp-send {"jsonrpc":"2.0","result":["tool1","tool2"],"id":1} # 1001.000 spanid=resp1
mcp-send-shadow {"jsonrpc":"2.0","result":["tool1","tool3"],"id":1} # 1001.100 spanid=shadow1 linksto=resp1
```

## OpenTelemetry Integration

MCPTrace's trace context can integrate with OpenTelemetry collectors and visualization tools:

1. **Exporting to OpenTelemetry Collectors**: Tools can convert MCPTrace files to OpenTelemetry spans and export them to collectors
2. **Span Generation**: Each request/response pair in MCPTrace can be represented as a span in an OpenTelemetry trace
3. **Span Attributes**: JSON-RPC fields can be mapped to span attributes (method, id, etc.)
4. **Context Propagation**: Trace context can be propagated to and from other systems via the W3C Trace Context standard
5. **Visualization**: Traces can be visualized in tools like Jaeger, Zipkin, or other OpenTelemetry-compatible UIs

### Advanced Tracing Scenarios

MCPTrace's per-row span context enables several advanced observability patterns:

1. **Traffic Splitting/Shadowing**:
   - Original request gets a span ID
   - Shadow copies get their own span IDs with `linksto` referencing the original
   - Visualizations can show fan-out patterns and compare response characteristics

2. **A/B Testing**:
   - Route requests to different implementations based on traffic rules
   - Link related spans to compare response times, error rates, and behavior differences
   - Use baggage to carry test-specific parameters

3. **Request Transformation**:
   - Track a request as it's transformed through multiple processing steps
   - Each transformation gets a new span ID linked to its predecessor
   - Baggage carries transformation-specific metadata

4. **Failure Analysis**:
   - When errors occur, link to related requests or system events
   - Capture diagnostic information in per-row baggage
   - Generate synthetic spans for system events to provide context