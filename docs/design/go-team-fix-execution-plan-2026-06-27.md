# Execution plan — go-team review fixes (2026-06-27)

Workflow for landing the verified findings from
[`go-team-review-2026-06-27.md`](go-team-review-2026-06-27.md) (B11–B14 + a
non-breaking polish subset). Ordering and scope were designed with the
NotebookLM Go-team panel; the raw verdict is in
[`reviews/2026-06-27/execution-workflow-verdict.md`](reviews/2026-06-27/execution-workflow-verdict.md).

Every claim below was re-verified against the filesystem before adoption (the
panel hallucinates from a mangled source archive — see
[`reviews/2026-06-27/triage.md`](reviews/2026-06-27/triage.md)). Two findings
turned out *wider* than the original triage recorded:

- B12 pagination: **four** list handlers (`server.go` ~592/651/710/799 —
  tools, prompts, resources, resource-templates), not three.
- B14 timer leaks: **both** `TokenBucketRateLimiter` and
  `SlidingWindowRateLimiter` (ratelimit.go) lack `Close()`, plus
  `InMemoryCache`.

## Ordering — leaf → additive → internal → high-churn

Each commit must keep `go build ./...` and `go test ./...` green and
bisect-safe. The sequence lands the lowest-risk, most-isolated changes first
and the cross-package churn last:

```
B13 (Darwin, build-tag isolated)
  → B14 (Close() — additive, leaf, multi-file)
    → B12 (server pagination — internal behavior)
      → polish (non-breaking cleanups, independent)
        → B11 (protocol/ deletion — cross-package, last)
```

B13, B14, B12 and the polish items are mutually independent (parallelizable in
principle); B11 is serialized last so the high-churn import rewrite happens on a
tree already verified correct.

## Commit sequence

1. **B13** — `platform_darwin.go`: `AppleOptimizedTransport.Read` respects
   `len(p)` (no reslice past caller length); delete the redundant
   `runtime.GOOS != "darwin"` panic guard sitting behind a `//go:build darwin`
   tag; drop the placeholder `sysctlByName`. Pure correctness, macOS-only.
2. **B14** — add `Close()` to `InMemoryCache`, `TokenBucketRateLimiter`,
   `SlidingWindowRateLimiter`, each stopping its cleanup timer. Modeled on
   `ConnectionPool` (done-channel + `Close()`). Additive, non-breaking.
3. **B12** — the four `server.go` list handlers slice by `Cursor` and populate
   `NextCursor`; a test registers >1 page and asserts paging. `Cursor` already
   exists on the public request structs, so this is additive behavior.
4. **Polish** (folded — all non-breaking, verified internal):
   - `progress.go`: unexported `contextKey` type instead of the bare strings
     `"mcp_progress"` / `"mcp_cancel_manager"`. Consumers use
     `ProgressFromContext` / `CancelManagerFromContext`, so no signature change.
   - `middleware_registry.go`: wire the Compression / Validation / Caching
     factories to the real implementations in `middleware_advanced.go` instead
     of returning `NoOpMiddleware` (config-driven middleware was silently a
     no-op).
   - `performance.go`: fix the encoder/decoder pool theater (the `Get` result is
     discarded and a fresh value allocated; `Put` is a no-op) — use `Reset` or
     delete the pool.
   - `example_test.go`: fix and unskip the deadlocking package `Example`.
5. **B11** — delete `protocol/` (only two `exp/` importers; migrate them to the
   root `mcp` types), align `modelcontextprotocol`'s `LATEST_PROTOCOL_VERSION`
   to the canonical `"2025-11-25"`, and document the root `types.go` as the
   canonical wire surface with `modelcontextprotocol/` as the
   reference/draft-compat package.

### B11 scope decision — surgical, not full consolidation

`protocol/` (173 L, 2 `exp/` importers) is deadweight: delete it. But
`modelcontextprotocol/` is **load-bearing** — imported by 17 example servers,
the root `dispatcher.go` and `security.go`, with its own `draft/` subpackage,
custom unmarshaling, and fuzz tests. Collapsing it into the root package now is
a large breaking churn for no v1 benefit. The v1 move is therefore: one
canonical version constant, one deleted redundant package, and a documented
boundary — which closes the "consumers can't tell which is canonical" complaint
without destabilizing the tag.

## Deferred to v2 (API-breaking — do not fold)

- `NotificationHandler` gaining a `context.Context` parameter (`types.go:647`) —
  breaks every registered notification handler.
- `NewServer` returning `(*Server, error)` to match `NewClient` — breaking
  signature change. Document the asymmetry for v1.
- Type-stutter renames (`MCPMethod`, `JSONRPCRequest/...`) and `Get`-prefix
  getter renames — pure churn; do them only if/when the surface moves in v2.
