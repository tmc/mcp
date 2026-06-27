# Go-team review — 2026-06-27 (HEAD a2499b9cb)

A multi-lens "Go team panel" review (Russ Cox / Rob Pike / Ian Lance Taylor /
Robert Griesemer / Brad Fitzpatrick personas, via NotebookLM) of the `tmc/mcp`
codebase, run after the server↔client request feature work landed. Six lenses:
topology, naming, api, consistency, smells, completeness.

This doc records only **filesystem-verified** findings. The panel hallucinates
(it reads source from a txtar archive where brackets get mangled), so every claim
was triaged against the tree. `go build ./...`, `go vet ./...`, and `go test ./...`
are clean at HEAD, so all "does not compile / empty body / broken generics"
claims were discarded as false. Full per-lens output, triage, and meta-synthesis:
`~/.mcp-go-team-review/run-a2499b9cb/`.

## Relationship to the existing v1 gate

This review is **additive** to [`v1-release-exemplary-gate.md`](v1-release-exemplary-gate.md)
(B5–B10) and [`release-readiness-synthesis.md`](release-readiness-synthesis.md).
That gate covers hygiene, security evidence, the dep contract, conformance/interop
harnesses, and the perf baseline. It does **not** cover the correctness, API-shape,
and spec-completeness issues this review surfaces. The items below are proposed as
new blockers (B11–B14) and a post-1.0 polish list.

The panel raised several items the existing gate already closed — recorded here so
they are not re-litigated:

- Repo-local `jsonrpc2/` "duplication": already removed (B10). Root uses
  `golang.org/x/exp/jsonrpc2`. The panel was reading stale notebook state.
- `cmd/` sprawl: already trimmed to `cmd/mcp` + `cmd/mcp-probe`.
- Security stubs (`deriveKey`, error sanitization): the panel called these empty;
  they are implemented and covered by B5 evidence tests. FALSE.

## Discarded as hallucination (do not action)

| Panel claim | Reality |
|---|---|
| "Broken generics syntax `func createJSONSchemaT any`" | `mcp.go:118` is `func createJSONSchema[T any]()` — valid. |
| "Empty/truncated transport bodies (SSE, streamable)" | Implemented; build is clean. |
| "`CancellablePreempter` is a logging-only no-op" | `preempter.go:65` calls `p.Conn.Cancel(rpcID)`. |
| "`ConnectionPool` leaks, no cancellation" | Has `cleanupDone` + `Close()`. |
| "`removeInternalDetails` / `containsSensitiveInfo` are stubs" | Real bodies, `errors.go:102-146`. |

## Proposed new v1 blockers

> **Status (2026-06-27): B11–B14 all resolved** on branch `next`
> (commits 83b168a06..e304a90a6, each with git-notes). The execution workflow
> — ordering, the surgical B11 scope, and the v2 deferrals — was designed with
> the panel and is recorded in
> [`go-team-fix-execution-plan-2026-06-27.md`](go-team-fix-execution-plan-2026-06-27.md).
> The non-breaking polish items below (NoOp factories, string context keys,
> encoder-pool theater, the deadlocking Example) were also fixed in the same
> pass. NotificationHandler-ctx and the constructor/naming churn remain deferred
> to v2. B5–B10 are unaffected and remain the open v1 gate.

### B11. Single canonical protocol-type package — DONE
The repo carries three drifting definitions of the same wire types:
`types.go` (root, 1186 L), `modelcontextprotocol/types.go` (559 L),
`protocol/types.go` (173 L). The drift is already concrete:
`LATEST_PROTOCOL_VERSION` = `"2025-11-25"` in `types.go:14` but `"2025-03-26"` in
`modelcontextprotocol/constants.go:4`. Consumers cannot tell which is canonical,
and identical structs are not assignable across packages.

Acceptance: exactly one package defines each wire type; the others are deleted or
become thin re-export shims with a deprecation note; one `LATEST_PROTOCOL_VERSION`.
The root `mcp` package's own types are the working canonical set today (they are
what `Client`/`Server` use), so the cheapest correct move is to delete `protocol/`
and reduce `modelcontextprotocol/` to a compatibility alias or remove it.
Cost: L. API-breaking for anyone importing `protocol`/`modelcontextprotocol`.

### B12. Honor list pagination cursors — DONE
`ListTools` / `ListPrompts` / `ListResources` accept a `Cursor` but the handlers
return the full set and never populate `NextCursor` (server.go list handlers, with
an in-code "future version" comment). A parameter that is accepted but ignored is a
dishonest API and a memory-exhaustion trap for large registries.

Acceptance: handlers slice by cursor and return `NextCursor`; a test registers >1
page and asserts paging. If pagination is deliberately deferred, the `Cursor` field
must be documented as accepted-but-ignored and the gate records the deferral.
Cost: M. Not breaking (additive behavior).

### B13. `io.Reader` contract violation in the Darwin transport — DONE
`platform_darwin.go:132-139` (`AppleOptimizedTransport.Read`) reslices the caller's
buffer `p` up to `cap(p)`/BufferSize when `len(p)` is smaller. `io.Reader` must
never write beyond `len(p)`; this can corrupt caller-owned memory. Correctness bug,
small fix.

Acceptance: `Read` respects `len(p)`; use an internal buffer if a larger read is
wanted. Also delete the redundant `runtime.GOOS != "darwin"` panic guard
(`platform_darwin.go:326-327`) that sits behind a `//go:build darwin` tag, and
either implement or remove the placeholder `sysctlByName` (`:66-77`).
Cost: S. Not breaking.

### B14. Background-goroutine / timer lifecycle — DONE
`InMemoryCache.startCleanup` (`middleware_advanced.go:312-318`) reschedules itself
via `time.AfterFunc` forever with no stop path; `TokenBucketRateLimiter`
(`ratelimit.go:92-97`) has a cleanup timer with no `Stop()`. Long-lived servers and
test loops leak timers/goroutines. (`ConnectionPool` already does this right — use
it as the model: a `done` channel closed by `Close()`.)

Acceptance: every type that starts a background ticker/timer exposes `Close()`
(or takes a lifecycle `context.Context`) that stops it; a leak test (or
`-race`/goroutine-count assertion) covers it.
Cost: S. Not breaking (additive `Close()`).

## Post-1.0 polish (do not block the tag)

These are real but are API-breaking churn or cosmetic; batch them behind the v1 tag
or into a v2 surface decision.

- **Wire up the NoOp middleware factories.** `middleware_registry.go` Compression/
  Validation/Caching `Create()` return `NoOpMiddleware` though real impls exist in
  `middleware_advanced.go`. Either wire them up or remove the factory entries so
  config-driven middleware isn't silently a no-op. Cost: S. (Borderline blocker —
  promote to B-list if config-driven middleware is in the v1 surface.)
- **`NotificationHandler` should take `context.Context`.** `types.go:647` is
  `func(method string, params json.RawMessage) error`; no ctx for tracing/cancel.
  API-breaking — bundle with any other handler-signature changes.
- **Unexported context keys.** `progress.go:285,296` use string keys
  (`"mcp_progress"`, `"mcp_cancel_manager"`); switch to an unexported key type.
- **Constructor symmetry.** `NewClient` returns `(*Client, error)`,
  `NewServer` returns `*Server`. Pick one; or document why they differ.
- **Type stutter.** `MCPMethod`, `JSONRPCRequest/Response/Notification/Error` read
  as `mcp.MCPMethod` etc. Renames are pure churn; do them only as part of the B11
  consolidation when the surface is already moving.
- **`Get`-prefixed getters.** `GetProgressToken`, `GetValue/GetMessage/GetTotal`,
  `GetMethod/GetID/GetParams`. Drop the prefix when the surrounding type changes.
- **Performance theater.** `performance.go:449-461` JSON encoder/decoder pool fetches
  then discards and allocates fresh; `Put` is a no-op; comment wrongly says
  `json.Encoder` has no `Reset` (it has since Go 1.12). Delete the pool or use
  `Reset`.
- **`mcp-probe` defines its own `Transport` (Send/Receive/Close)** vs core
  `mcp.Transport` (Dial), and carries tracked `.mcp` trace files + shell scripts in
  its dir. It is a tool, not the library surface — clean opportunistically.
- **Deadlocking package `Example`.** `example_test.go:9` skips the headline example
  "as it's deadlocking". A skipped headline example erodes trust; fix and unskip, or
  replace with a runnable one.

## Patterns the panel said to keep
- Sealed `Content` / `ResourceContents` interfaces via unexported marker methods.
- Generics-based typed registration (`RegisterTypedToolWithServer`, `CallToolTyped`).
- Single-`Dial` `Transport` abstraction.
- `context.Context` threaded through the protocol surface.
- `synctest`-based deterministic concurrency tests.
- The `ConnectionPool` lifecycle (the model B14 should copy).

## Judgment calls
- **Consolidate local types vs adopt the official `go-sdk`.** Recommend: consolidate
  internally (B11) for v1; do not rebase the core onto the young official SDK, which
  would cost the generics-based registration. Revisit SDK adoption post-1.0 as a
  separate decision (see `docs/OFFICIAL_SDK_ANALYSIS.md`).
- **Move transports/middleware/security into subpackages.** Recommend: defer. It is
  a large breaking reshuffle; the flat `package mcp` is usable. Decide at the same
  time as B11, not before.
