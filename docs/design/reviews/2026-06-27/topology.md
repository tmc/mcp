<!-- focus: topology | notebook: 767b8a67-abd6-4981-8fc6-907c8ed110e4 | head: a2499b9cb | generated: 2026-06-27T08:12:01Z -->

## Verdict

This codebase **fails Go-team review** today and is **not ready to tag a stable release**. While individual components show significant technical effort, the package topology suffers from **severe layering violations, namespace pollution, and structural duplication**. Specifically, the root package (`package mcp`) acts as a monolithic god-package [1], forcing consumers to import transports, OAuth2/security frameworks, and a heavy middleware engine just to use basic client/server abstractions [2, 3]. Most critically, the repository is paralyzed by **extreme type triplication**, maintaining three competing and semantically drifting definitions of identical protocol schemas across `mcp` (`types.go`) [4], `modelcontextprotocol` (`modelcontextprotocol/types.go`) [5], and `protocol` (`protocol/types.go`) [6]. Until the codebase is strictly modularized—separating core client/server engines, transports, middleware, and schemas into clean package boundaries—it cannot be held up as exemplary Go.

## Findings

1. **Core Type Triplication and Version Drift** (severity: **blocker**)
    - **Location**: `types.go` [4], `modelcontextprotocol/types.go` [5], and `protocol/types.go` [6]
    - **Smell / problem**: The codebase defines the core Model Context Protocol (MCP) data structures (such as `Tool`, `CallToolResult`, `Resource`, `Prompt`) **three separate times** across different packages with varying field types and JSON tags. This structural redundancy has already caused **semantic version drift**: the root `types.go` defines `LATEST_PROTOCOL_VERSION` as `"2025-11-25"` [7], while `modelcontextprotocol/constants.go` defines it as `"2025-03-26"` [8]. This fragmentation forces consumers to convert between incompatible types representing the same wire data, destroying compile-time type safety.
    - **Recommendation**: **Russ** weighs in: We must have a single source of truth for the protocol types. **Delete the duplicate types** in the root `types.go` and `protocol/types.go`. Promote `package modelcontextprotocol` as the **sole canonical schema package**, and force the root client and server to import and consume those types directly.
    - **Why it matters**: Go standard library packages do not maintain competing definitions of their wire protocols. For example, `net/http` has exactly one `http.Request` struct; there are no parallel, drifting definitions in `net/` or `protocol/`.

2. **Root Package Namespace Pollution** (severity: **high**)
    - **Location**: `/workspace/` root files (e.g., `middleware.go` [9], `security.go` [10], `transport.go` [11])
    * **Smell / problem**: The root `mcp` package is a **monolithic god-package** that mixes core client/server abstractions with middleware chains, security handlers, and specific transport implementations (stdio, SSE, WebSockets, streamable HTTP). This violates proper package boundaries and forces heavy transitive dependencies onto lightweight clients.
    * **Recommendation**: **Rob** weighs in: "Keep package boundaries clean; less is more." We must **fold the root package down** to only contain `Client`, `Server`, and the core `Transport` interfaces. Move specialized subsystems to their own packages:
        * `github.com/tmc/mcp/transport/sse`
        * `github.com/tmc/mcp/transport/websocket`
        * `github.com/tmc/mcp/middleware`
        * `github.com/tmc/mcp/security`
    * **Why it matters**: Go packages should have a single, focused responsibility. Shoving unrelated domains into a flat namespace makes code navigation difficult and bloats public Go documentation.

3. **Platform-Specific Types Leaked in Public Root API** (severity: **high**)
    - **Location**: `platform_darwin.go` [12], `server.go` [13] (symbol: `WithAppleOptimizations`)
    - **Smell / problem**: The `platform_darwin.go` file implements macOS/Apple-specific performance optimizations. However, it exports platform-specific types (such as `ApplePlatformInfo` [14] and `DarwinTransportOptimizations` [15]) and server options (`WithAppleOptimizations` [13]) **directly in the root package namespace**. This makes the root public API target-dependent, cluttering the API for non-Apple users and creating dead weight on Linux/Windows builds.
    - **Recommendation**: **Ian** weighs in: Platform-specific performance optimizations should be **completely hidden** behind runtime-detectable interfaces or encapsulated in internal packages (e.g., `internal/platform/darwin`). The public root API must remain fully portable and target-agnostic.
    - **Why it matters**: Go values **portability**. Standard library packages like `net` handle platform-specific socket polling (kqueue, epoll, Windows IOCP) internally without exposing platform-specific types in the public root package namespace.

4. **Inconsistent and Fragile Concurrency Primitives** (severity: **medium**)
    - **Location**: `ratelimit.go` [16] (symbol: `TokenBucketRateLimiter`), `progress.go` [17] (symbol: `ProgressManager`)
    - **Smell / problem**: The codebase displays an inconsistent approach to concurrency management, alternating arbitrarily between `sync.Map` and `sync.RWMutex` to protect similar map-based structures. For instance, `ProgressManager` uses a map protected by `sync.RWMutex` [17], while the rate limiter uses a raw `sync.Map` [16], which can lead to race conditions or duplicate allocations during double-checked bucket initialization.
    - **Recommendation**: **Brad** weighs in: Standardize on **mutex-protected Go maps** for structured managers where read/write ratios are dynamic and initialization order matters. Reserve `sync.Map` strictly for cache-like structures that are read-heavy, append-only, and do not suffer from initialization races.
    - **Why it matters**: "Do not communicate by sharing memory; instead, share memory by communicating." When shared memory is necessary, consistent and predictable locking patterns prevent hard-to-debug deadlock and race foot-guns in production.

5. **Java-esque Class-Mimicry and Over-Engineered Naming** (severity: **medium**)
    - **Location**: `middleware_integration.go` [18] (symbol: `SuccessResponseImpl`), `middleware_advanced.go` [19] (symbol: `JSONMinifierTransformer`)
    - **Smell / problem**: The middleware and adapter systems suffer from highly verbose, object-oriented naming conventions. Identifiers like `SuccessResponseImpl` [18] and `JSONMinifierTransformer` [19] mimic Java class structures rather than idiomatic Go designs.
    - **Recommendation**: **Robert** weighs in: Simplify the naming. **Rename `SuccessResponseImpl` to `successResponse`** and keep it unexported. Go uses structural subtyping; there is no need to write `Impl` suffixes, as interfaces are satisfied implicitly.
    - **Why it matters**: Go proverbs dictate: "Interfaces are for satisfying, not for implementing." Go naming should be short, punchy, and reflect behavior rather than implementation details.

6. **Goroutine Leak Danger in SSE Transport readLoop** (severity: **medium**)
    - **Location**: `transport_sse.go` [20] (symbol: `sseRWCAdapter.Close` / `readLoop`)
    - **Smell / problem**: The Server-Sent Events (SSE) adapter runs a background `readLoop` [20] goroutine that streams data from the network. While `Close()` [20] closes the underlying body, if the goroutine is blocked on a channel send to `readChan` and the consumer has abandoned the channel, the goroutine will **leak indefinitely**.
    - **Recommendation**: **Brad** weighs in: Ensure the `readLoop` selects on both the read channel and a dedicated `closed` channel. Always coordinate goroutine lifecycles using context cancellation or explicit select blocks during channel writes to guarantee garbage collection.
    - **Why it matters**: Goroutine leaks are a major source of memory exhaustion in long-lived production Go servers.

## Patterns to keep

* **Sealed Interfaces for Closed Unions**: The use of unexported marker methods (e.g., `isContent()` on `Content` [21]) to seal protocol interfaces is excellent [22]. This prevents external users from implementing invalid protocol types, guaranteeing protocol-compliant marshaling at compile time.
* **Functional Options for Server and Client Configuration**: The use of the functional options pattern (`WithServerName`, `WithServerVersion`) in `options.go` [23] is highly idiomatic, clean, and easily extensible.
* **Pure Go Platform Gating**: Gating platform optimizations using pure Go build tags (`//go:build darwin` [12]) rather than resorting to CGO is an outstanding practice that preserves Go's cross-compilation simplicity.

## Open questions

* **What is the migration strategy regarding the official Go SDK?**: Since `cmd/mcp-probe` has been updated to depend on the official Anthropic/Google `github.com/modelcontextprotocol/go-sdk` [24], does the owner intend to deprecate the root `mcp` package's client/server implementations entirely and reposition `tmc/mcp` strictly as an enterprise middleware extension?
* **What is the purpose of the legacy `protocol/` package?**: The `protocol` subpackage [6] appears to carry a third, highly simplified copy of the protocol types. Is this package left over from a previous refactoring, and can it be deleted immediately to prevent consumer confusion?

***

🎧 Want me to turn this architecture review into a multi-host audio overview comparing the three parallel type systems?
