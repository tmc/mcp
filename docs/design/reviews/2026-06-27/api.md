<!-- focus: api | notebook: 767b8a67-abd6-4981-8fc6-907c8ed110e4 | head: a2499b9cb | generated: 2026-06-27T08:13:43Z -->

## Verdict
This codebase **would not pass Go-team review today and is not ready for a stable v1.0 release.** While it contains excellent individual components, a sophisticated middleware framework, and highly optimized reflection-based schema generation, it falls short of production readiness due to **critical type-definition fragmentation, incompatible transport abstractions between tools and the core library, and severe concurrency and lifecycle design issues.** Specifically, the codebase duplicates core protocol structs across three distinct packages (`mcp`, `modelcontextprotocol`, and `protocol`), introduces global state side-effects in a library-level `init()` function, and contains silent goroutine leaks in its connection pooling. Before tagging a stable release, these architectural boundary violations, interface inconsistencies, and concurrency foot-guns must be systematically resolved.

## Findings

1. **Type Definition Redundancy and Namespace Fragmentation** (severity: blocker)
   - **Location**: `types.go`, `modelcontextprotocol/types.go`, and `protocol/types.go` [1-3]
   - **Smell / problem**: The codebase suffers from extreme type duplication. Core protocol structs like `Tool`, `CallToolResult`, `Resource`, and `Prompt` are defined independently across three different packages—the root `mcp` package, `modelcontextprotocol`, and `protocol`—with subtly different field types (e.g., `[]any` versus `[]Content`). This forces downstream developers to perform redundant type conversions and creates massive confusion about package boundaries and canonical representations.
   - **Recommendation**: **Delete the redundant `protocol` and `modelcontextprotocol` packages entirely.** Consolidate all canonical wire-format schemas and protocol structs into a single, well-documented package (such as `modelcontextprotocol`), and have the root `mcp` client and server packages import and consume these unified types directly.
   - **Why it matters**: *Go Proverb: "A little copying is better than a little dependency."* However, internal duplication of complex, overlapping wire-format types within the same library is not copying—it is fragmentation. In the standard library, `net/http` does not have a separate, third-party-looking `http` types package that duplicates `Request` and `Response`.

2. **Inconsistent Transport Interface Definition between Core and Tools** (severity: blocker)
   - **Location**: `transport.go` (`Transport` interface) [4] and `cmd/mcp-probe/main.go` (`Transport` interface) [5]
   - **Smell / problem**: The core library defines `Transport` as `Dial(context.Context) (io.ReadWriteCloser, error)` [4] to abstract communication channels. However, the `mcp-probe` command-line tool defines an incompatible `Transport` interface with `Send`, `Receive`, and `Close` methods [5]. This inconsistency prevents core transports (like `SSEClientTransport` or `WebSocketTransport`) from being used directly with diagnostic tools, breaking the expected interface uniformity of the project.
   - **Recommendation**: **Align the transport interfaces.** The `mcp-probe` tool should import and consume the core `mcp.Transport` interface and use the canonical `jsonrpc2` framer for message exchange, rather than inventing an ad-hoc, incompatible transport abstraction.
   - **Why it matters**: Interfaces should be small and uniform to encourage reuse across the entire codebase. Introducing duplicate, incompatible interfaces for the same logical concept in tools and library packages violates the clean design patterns seen in packages like `net` (where `Conn` is used universally).

3. **Global Error Sanitization and Side-Effects inside Library `init()`** (severity: high)
   - **Location**: `errors.go` (init, `SetErrorVerbosity`, `SanitizeError`) [6, 7]
   - **Smell / problem**: The library configures mutable global state during `init()` by reading the `MCP_ERROR_VERBOSITY` environment variable to toggle error sanitization [7]. It then automatically redacts filesystem paths and internal database/cryptographic details from returned error messages [8, 9]. Error sanitization and logging formatting are application-level concerns, and handling them inside a library's global state introduces hidden side-effects that make isolated, parallel testing impossible.
   - **Recommendation**: **Remove the global error verbosity state, `init()` environment checks, and the `SanitizeError` automatic filtering from the core library.** If error sanitization is desired for security, implement it as a configurable, explicit server middleware (e.g., `NewErrorSanitizerMiddleware(...)`) that applications can explicitly wire into their handler chains.
   - **Why it matters**: Libraries should be quiet, stateless, and side-effect-free. Go's standard library packages (such as `database/sql` or `crypto`) never globally sanitize or redact errors themselves; they return full contextual errors and let the calling application's logging or middleware layer decide how to format or redact them for production.

4. **Silent Goroutine Leak in Connection Pool Cleanup** (severity: high)
   - **Location**: `connection_pool.go` (struct `ConnectionPool`, method `cleanupRoutine`) [10, 11]
   - **Smell / problem**: The `ConnectionPool` spawns a background goroutine via `cleanupRoutine()` to periodically evict idle connections using a `cleanupTicker` [11]. However, if a client discards the `ConnectionPool` without explicitly calling `Close()`, the background goroutine will block indefinitely on `p.cleanupTicker.C`, preventing the garbage collector from reclaiming the pool and leaking the goroutine.
   - **Recommendation**: **Do not run un-cancellable background loops.** Pass a `context.Context` to the pool constructor to control the lifetime of the background goroutine, or use `runtime.SetFinalizer` on the connection pool to stop the ticker and exit the goroutine when the pool is garbage collected.
   - **Why it matters**: Goroutine leaks are severe runtime bugs that lead to progressive memory exhaustion in long-lived server processes. The standard library's `net/http` package goes to great lengths to ensure that idle connection cleanup goroutines are stopped and garbage collected correctly when a `Transport` is discarded.

5. **Context Deadlock Vulnerability in Framer Write** (severity: high)
   - **Location**: `framer.go` (method `Write` on `lineWriter`) [12]
   - **Smell / problem**: In `lineWriter.Write`, the context is checked only at the entry boundary of the call: `select { case <-ctx.Done(): return 0, ctx.Err() default: }` [12]. If the context is cancelled *during* the subsequent blocking write call `w.out.Write(...)` [12], the write will block indefinitely on the underlying socket or pipe, rendering the context cancellation completely ineffective.
   - **Recommendation**: Wrap the blocking write or utilize non-blocking I/O with deadlines (e.g., `net.Conn.SetWriteDeadline`) driven by the context's lifetime. For general `io.Writer` targets, you must document that the writer must be interruptible, or use channel-based coordination.
   - **Why it matters**: Context cancellation must be respected throughout the lifetime of an I/O operation, not just at its invocation boundary. The standard library's `net` and `crypto/tls` packages actively monitor context cancellation to set network connection deadlines and abort blocked operations promptly.

6. **API Constructor Asymmetry and Lack of Zero-Value Usability** (severity: medium)
   - **Location**: `client.go` (`NewClient`) [13] and `server.go` (`NewServer`) [14]
   - **Smell / problem**: There is a jarring asymmetry between client and server instantiation: `NewClient` returns `(*Client, error)` [13], while `NewServer` returns a bare `*Server` [14] (inferring missing config through build info [15]). Additionally, both `Client` and `Server` structs contain uninitialized maps and mutexes, making them completely unusable as zero-values (e.g., `var s mcp.Server` will panic on tool registration).
   - **Recommendation**: **Standardize constructors to return `(Type, error)` if they perform operations that can fail, or ensure they are safe to use via lazy initialization.** For zero-value usability, initialize maps on-demand (e.g., `if c.requestHandlers == nil { c.requestHandlers = make(...) }`) or hide internal state behind interface boundaries.
   - **Why it matters**: *Go Proverb: "Make the zero value useful."* If a struct cannot be useful as a zero-value, its constructor must be uniform and safe. Standard library types like `sync.Mutex` or `bytes.Buffer` are immediately usable, while complex types like `http.Client` are designed so their zero-value uses sensible defaults.

7. **Redundant Build Tag Runtime Guard and Stub Implementation** (severity: low)
   - **Location**: `platform_darwin.go` [16-18]
   - **Smell / problem**: `platform_darwin.go` uses the `//go:build darwin` build tag [16], yet includes a redundant runtime check at the bottom that panics if `runtime.GOOS != "darwin"` [18]. Furthermore, the `sysctlByName` function contains a placeholder comment ("In a full implementation, you'd use syscall.Syscall...") and lacks a complete, working implementation [17].
   - **Recommendation**: Remove the redundant runtime panic block [18] since the Go compiler's build tag ensures the file is only compiled on macOS. Fully implement `sysctlByName` using the standard `syscall.Syscall` or `unix.Sysctl` paths to make the Apple platform optimizations production-ready, or remove the file entirely if it is not ready for v1.0.
   - **Why it matters**: Placeholder stubs and redundant runtime panics detract from the "exemplary" quality expected of a Go team release. The standard library's platform-specific files (e.g., in `os` or `syscall`) rely purely on build tags for compilation hygiene and contain fully implemented, non-placeholder code.

## Patterns to keep

- **Polymorphic Content Handling via Sealed Interfaces**: The `Content` [19] and `ResourceContents` [20] interfaces use unexported marker methods (`isContent()`, `isResourceContents()`) to seal the interfaces [21]. This is excellent Go practice that prevents external packages from introducing unsupported protocol types while preserving compile-time type safety.
- **Fluent Options Pattern for Server/Client Configuration**: The use of functional options (e.g., `WithServerName` [22] and `WithNotificationHandler` [23]) is clean, self-documenting, and fully backwards-compatible, mirroring the standard design of packages like `crypto/tls`.
- **AST-Based Reflection & Struct Tag Validation**: The reflection-based schema generation with thread-safe caching (`SchemaCache` [24] in `mcp.go`) and the `StructValidator` [25] show thoughtful optimization that prevents runtime overhead while preserving developer ergonomics.
- **Graceful Shutdown in Connection Pooling**: The pool's ability to track active vs. idle connections and execute clean shutdowns is well-structured [26].

## Open questions

- **Why are there three identical wire-format packages (`mcp`, `protocol`, and `modelcontextprotocol`)?** Is this a historical artifact of a partial transition from a third-party specification? 
- **What is the security boundary of the global error sanitization?** If a library strips paths and SQL queries globally, how can parent applications debugging issues retrieve full diagnostics?
- **How is the `mcpscripttest` DSL intended to be versioned?** It seems highly complex and tightly coupled to testing CLI tools rather than being a standard Go testing harness.

***

👉 **Next Step**: Would you like me to generate a refactoring plan to consolidate the duplicate `Tool` and `CallToolResult` structures into a single canonical package?
