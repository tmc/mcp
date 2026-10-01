// Package mcpfixture converts a recorded linear tool call into an editable
// script regression fixture. Convert reads a trace and writes txtar; it never
// starts a process or calls a tool. Review observed results before adopting them
// as assertions. The emitted script runs mcpfixture check explicitly and requires
// MCP_SERVER to name the server executable. Check also supports HTTP endpoints.
//
// Conversion supports legacy initialize handshakes and the 2026-07-28
// server/discover lifecycle. Client identity, requested and negotiated versions,
// supported client configuration, and request metadata are retained. Callback
// capabilities, progress callbacks, continuations, and concurrent requests are
// rejected because their prerequisites cannot be reconstructed safely.
//
// Sequential tools/list, prompts/list, resources/list, and
// resources/templates/list exchanges may precede the selected tools/call.
// These observed list results are checked before replaying the selected call.
// No earlier tool calls or resource reads are accepted. Conversion validates the
// handshake through the selected response; later exchanges are outside this
// extraction boundary. The full trace must still be syntactically complete.
package mcpfixture
