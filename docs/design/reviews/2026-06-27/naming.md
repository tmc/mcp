<!-- focus: naming | notebook: 767b8a67-abd6-4981-8fc6-907c8ed110e4 | head: a2499b9cb | generated: 2026-06-27T08:13:43Z -->

## Verdict

No, this codebase would not pass a Go team review today and is not ready to be tagged as a stable release. While the project exhibits impressive functional breadth, rich testing utilities, and modern Go generic patterns, it falls critically short in **API coherence**, **package boundaries**, and **naming idioms**. The coexistence of three redundant, competing type packages (`mcp`, `modelcontextprotocol`, and `protocol`), coupled with verbose package names, getter-method naming violations, and constructor sprawl, makes the public API surface feel like a disjointed assembly of different development eras rather than a clean, unified, production-ready Go product.

## Findings

1. **Package Name Complexity and Verbosity**
   - **Severity**: blocker
   - **Location**: `modelcontextprotocol/`
   - **Smell / problem**: The package name `modelcontextprotocol` is far too long (20 characters) and redundant given that the parent repository and module are named `mcp`. Having users import and write `modelcontextprotocol.InitializeRequestParams` instead of a concise package prefix is unidiomatic and causes unnecessary visual noise [1, 2].
   - **Recommendation**: Delete the `modelcontextprotocol` subpackage and fold its schema types directly into the root `mcp` package, or if a separate subpackage is desired for wire-level schemas, rename it to a clean, singular noun like `wire` or `schema`.
   - **Why it matters**: Rob: "Package names should be short, lowercase, and singular. See `json`, `http`, `zip`—not `javascriptobjectnotation` or `hypertexttransferprotocol`. Package names should be self-explanatory and never stutter with the module's name."

2. **Severe Type Duplication and Package Sprawl**
   - **Severity**: blocker
   - **Location**: `types.go`, `modelcontextprotocol/types.go`, `protocol/types.go`
   - **Smell / problem**: The repository defines three duplicate, conflicting versions of core protocol types (such as `Tool`, `CallToolResult`, and `ResourceContents`) [3-9]. For example, `mcp.Tool` is defined differently than `modelcontextprotocol.Tool`, which itself differs from `protocol.Tool` [3, 10, 11]. This duplication creates massive confusion for users and leads to a bloated API surface with redundant adapter code [12].
   - **Recommendation**: Establish a single source of truth for the protocol types. Consolidate all canonical structs into the main `mcp` package, delete the duplicate definitions, and ensure that both internal code and client tools build against a single, unified type system.
   - **Why it matters**: Russ: "API surface hygiene is paramount. Having multiple structs representing the same protocol entity with different fields forces constant converting and adapter glue. Go libraries should have one clear, canonical representation of their core domain."

3. **Getter Methods Violating Go Naming Conventions**
   - **Severity**: high
   - **Location**: `middleware.go` (interfaces `MCPRequest`, `MCPResponse` [cite: 212]), `progress.go` (`Progress` methods [cite: 261-262]), `client.go` (`CallToolRequest.GetProgressToken` [cite: 397])
   - **Smell / problem**: Exported methods on interfaces and structs prefix their retrieval methods with `Get` (e.g., `GetMethod()`, `GetID()`, `GetParams()`, `GetValue()`, `GetProgressToken()`). This is a direct violation of Go standard library style.
   - **Recommendation**: Rename these methods to omit the `Get` prefix: `Method()`, `ID()`, `Params()`, `Value()`, and `ProgressToken()`.
   - **Why it matters**: Robert: "The Go style is clear: do not put 'Get' in the getter's name. It's redundant and unidiomatic. A method to get a value should be named after the value itself, just like `Request.Context()` in the `net/http` package."

4. **Constructor Overloading and Public API Sprawl**
   - **Severity**: high
   - **Location**: `middleware_integration.go` (e.g., `NewEnhancedServer` vs `NewEnhancedServerWithName` [cite: 174-175]), and multiple middleware/transport packages
   - **Smell / problem**: The API suffers from constructor sprawl and exports initialization functions for internal implementation details. For example, having both `NewEnhancedServer` and `NewEnhancedServerWithName` as separate exports violates Go patterns, while exporting constructors for every intermediate middleware struct clutters the public namespace [13-16].
   - **Recommendation**: Consolidate overloaded constructors into a single constructor using functional options (e.g., `NewEnhancedServer(name, version, opts...)`) and keep implementation-specific constructors unexported.
   - **Why it matters**: Brad: "API ergonomics suffer when users are confronted with dozens of `New...` functions. We should leverage Go's functional options pattern to keep constructors singular and clean, while hiding internal types behind package boundaries."

5. **Type Name and Package Stuttering**
   - **Severity**: medium
   - **Location**: `dispatcher.go` (`MCPMethod` [cite: 116]), `types.go` (`JSONRPCNotification`, `JSONRPCRequest`, `JSONRPCResponse`, `JSONRPCError` [cite: 388, 389])
   - **Smell / problem**: Several types stutter by repeating the package or protocol name in their type identifiers within the `mcp` package. For example, `mcp.MCPMethod` and `mcp.JSONRPCRequest` repeat information already implied by their package namespace.
   - **Recommendation**: Rename `MCPMethod` to `Method` and `JSONRPCRequest`/`JSONRPCResponse` to `Request`/`Response` (or `RPCRequest`/`RPCResponse` if a conflict exists with the middleware interface).
   - **Why it matters**: Rob: "Stuttering makes code read like a broken record. Avoid `mcp.MCPMethod`; use `mcp.Method`. The package name already provides the context."

6. **Inconsistent Initialism and Acronym Casing**
   - **Severity**: medium
   - **Location**: `auth_security.go` (`SecureOAuthProvider`, `KeyDerivationPBKDF2`, `KeyDerivationArgon2` [cite: 64]), `types.go` (`isValidURI`, `isValidIdentifier` [cite: 423])
   - **Smell / problem**: Acronyms and initialisms are inconsistently cased throughout the codebase. We see `OAuth` (mixed-case initialism) alongside all-uppercase `JSONRPC` and `URI`, and camel-cased `jsonSchema` in some files, violating the Go style rule of uniform casing for acronyms.
   - **Recommendation**: Enforce consistent casing across the entire repository: use `JSON` instead of `json`, `Oauth` or `OAuth` consistently, and keep all acronyms (e.g., URL, ID, SSE) consistently capitalized in exported identifiers.
   - **Why it matters**: Ian: "Acronyms in Go must be consistently cased—e.g., `HTTP`, `URL`, `ID`. See `net/http` which uses `URL` and `IP`, never `Url` or `Ip`. This preserves portability of style."

7. **Concurrency Foot-guns in Shared State Stores**
   - **Severity**: high
   - **Location**: `internal/mcpcli/state.go` (`StateStore.AddRoot` and `StateStore.RemoveRoot` [cite: 1386, 1387])
   - **Smell / problem**: The `StateStore` implements concurrent safety using a local `sync.Mutex` to coordinate writes to `state.json` [17, 18]. However, because `state.json` can be modified by separate concurrent CLI process invocations or parallel test runs, the memory-bound mutex is completely ineffective at preventing filesystem-level race conditions.
   - **Recommendation**: Use a filesystem lock (like `syscall.Flock` or a lock file) when modifying shared state files on disk, or leverage atomic file replacement.
   - **Why it matters**: Brad: "A filesystem-backed state store shared across multiple CLI invocations or parallel test runs is a classic concurrency foot-gun. A local process mutex won't protect you from concurrent processes overwriting each other's state."

## Patterns to keep

- **Grounded Context Propagation**: The consistent adoption of `context.Context` [cite: 119] across all client handshakes, tool calls, and background transport readers is exemplary. 
- **Type-Safe Generics Integration**: The type-safe generic API in `typed.go` (`RegisterTypedToolWithServer`, `CallToolTyped`) [cite: 370, 371] represents excellent type-system fit.
- **Pure-Go Platform Optimizations**: The darwin optimizations in `platform_darwin.go` [cite: 248] are cleanly isolated without dragging in CGO dependencies, preserving pure-Go portability.

## Open questions

1. **Why do we have multiple competing packages (`mcp`, `protocol`, and `modelcontextprotocol`) defining duplicate versions of the exact same protocol objects?** This design suggests a historical struggle to coordinate various forks or standard libraries. We must unify this.
2. **What are the compatibility requirements with the official `@modelcontextprotocol/sdk`?** Should the Go implementation deprecate its custom type schemas entirely in favor of importing the official SDK's types to minimize duplication?

👉 Want me to draft a pull request script that consolidates `protocol/types.go` and `modelcontextprotocol/types.go` into a unified `schema` package to resolve the blocker-level duplication?
