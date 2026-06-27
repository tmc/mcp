<!-- triage of go-team panel findings vs filesystem | head: a2499b9cb | 2026-06-27 -->
# Triage — panel findings vs filesystem (HARD RULE: filesystem is ground truth)

Ground truth: `go build ./...` and `go vet ./...` are CLEAN at HEAD a2499b9cb. Therefore every
"does not compile / empty body / truncated function / broken generics / missing return" claim is a
NotebookLM hallucination from txtar-mangled source, UNLESS a real syntax error is shown.

## FALSE — discard (panel hallucinations)
- **"Broken generics syntax"** (`func createJSONSchemaT any`): FALSE. `mcp.go:118` is `func createJSONSchema[T any]() (json.RawMessage, error)` — valid Go. Brackets mangled in the archive.
- **"Empty function bodies / truncated transports"** (SSE Dial, Streamable Write, etc.): FALSE. All implemented; build is clean.
- **"CancellablePreempter is a logging-only no-op"**: FALSE. `preempter.go:65` calls `p.Conn.Cancel(rpcID)` — real cancellation.
- **"ConnectionPool.cleanupRoutine leaks, no cancellation"**: FALSE. Selects on `cleanupDone`; `Close()` stops the ticker and closes the channel.
- **"errors.go removeInternalDetails/containsSensitiveInfo are empty stubs"**: FALSE. Both have real bodies (errors.go:102-146).
- **"auth_security.go deriveKey/needsRotation/... are empty stubs"**: FALSE. deriveKey has real Argon2/PBKDF2 logic.

## VERIFIED — real, actionable
| # | Issue | Location | Severity |
|---|---|---|---|
| V1 | Three drifting protocol type packages (`mcp` root, `modelcontextprotocol`, `protocol`) defining the same wire types | types.go (1186 L), modelcontextprotocol/types.go (559 L), protocol/types.go (173 L) | high (architecture) |
| V2 | LATEST_PROTOCOL_VERSION drift: root = "2025-11-25", modelcontextprotocol = "2025-03-26" | types.go:14 vs modelcontextprotocol/constants.go:4 | high |
| V3 | Placeholder middleware factories return NoOpMiddleware despite real impls existing | middleware_registry.go:543-545,562-564,581-583 | high (silent foot-gun) |
| V4 | Background timer leaks: no Stop/Close on cache + rate limiter cleanup loops | middleware_advanced.go:312-318 (InMemoryCache.startCleanup), ratelimit.go:92-97 | high |
| V5 | io.Reader contract violation: AppleOptimizedTransport.Read reslices caller buffer to BufferSize | platform_darwin.go:132-139 | high (correctness) |
| V6 | Pagination accepted but ignored: Cursor in, NextCursor never populated | server.go list handlers (~577-595) | high (spec gap) |
| V7 | Server-side tasks: types/constants exist, no handlers, capability never advertised | types.go (TaskInfo block), server.go | medium (spec gap, documented) |
| V8 | global error sanitization via package-level mutable state + init() env read | errors.go:22-28 | medium (library-as-app concern) |
| V9 | NewClient returns (*Client,error) but NewServer returns bare *Server — asymmetry | client.go:60 vs server.go:155 | medium (API) |
| V10 | NotificationHandler signature lacks context.Context | types.go:647 | medium (API, breaking) |
| V11 | String context keys instead of unexported key type | progress.go:285,296 | medium |
| V12 | Type stutter: MCPMethod, JSONRPCRequest/Response/Notification/Error in package mcp | types.go:82,123,128,131,138,141,145 | medium (API) |
| V13 | Get-prefixed getters: GetProgressToken, GetValue/GetMessage/GetTotal, GetMethod/GetID/GetParams | client.go, progress.go:69-88, middleware.go | low-medium (API) |
| V14 | JSON encoder/decoder pool is performance theater (fetch discarded, Put no-op); stale comment re json.Encoder.Reset | performance.go:449-461 | low |
| V15 | platform_darwin.go redundant runtime panic guard behind //go:build darwin | platform_darwin.go:326-327 | low |
| V16 | platform_darwin.go sysctlByName is a placeholder returning 0,nil | platform_darwin.go:66-77 | low |
| V17 | cmd/mcp-probe defines its own incompatible Transport (Send/Receive/Close) vs core mcp.Transport (Dial) | cmd/mcp-probe/main.go:33-37 | low (tool, not library) |
| V18 | Constructor sprawl: NewEnhancedServer + NewEnhancedServerWithName | middleware_integration.go:112-117 | low |
| V19 | Deadlocking package Example is commented out / skipped | example_test.go:9 | medium (docs/trust) |

## PARTIAL / context
- **framer.go ctx-mid-write**: ctx checked at entry only; blocking write not interruptible. This is normal io.Writer behavior — document the requirement rather than treat as a bug. low.
- **sseRWCAdapter stores ctx in struct**: discouraged, but used only to drive its own readLoop; low priority.

## Stale/scope
- Panel repeatedly suggests adopting/wrapping the official `github.com/modelcontextprotocol/go-sdk`. That is a strategic decision for the owner, not a triage item.
