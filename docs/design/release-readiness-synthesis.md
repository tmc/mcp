# Release Readiness Synthesis: API, Package Design, and Release Pathway

Status: updated after cleanup sweep and cmd/ trim (session 2)
Date: 2026-04-22

## Why this doc exists

tmc/mcp has no releasable v1 tag yet, and the session-1 kickoff found a
tree that still needed hygiene cleanup before API review could mean
anything: duplicate `exp/` subtrees, a root test package hidden behind a
nested module boundary, inconsistent transport-close errors, and a `cmd/`
tree that mixed core protocol tooling with experimental utilities. As of
2026-04-22, H2/H3/H5, B1/B2/B3/B4, and the `cmd/` trim are closed locally.
H1 is a no-op locally: `origin` has zero tags and the local clone has no
tracked tags to purge. H4 is effectively closed: no tracked ALL_CAPS status
docs remain at the repo root.

This doc is the single source of truth for:

1. The API design rules that the v1 public surface must satisfy.
2. The package-design pathway: what ships under v1, what's deferred,
   what's out of scope.
3. The release pathway: hygiene gates first (the tree is not releasable
   as-is), then the code gates, then tag mechanics.
4. The contributor-onboarding pathway — what a first-timer needs to
   ship a PR.

Related docs (most still to be written):

- `v1-release-exemplary-gate.md` — B1..Bn (code) and H1..Hn (hygiene) blockers.
- `v1-subsystem-reviews.md` — per-tier review (core, transport, middleware,
  schema, cmd/, testing, exp/).
- `codex-v1-execution-prompt.md` — handoff doc for automated execution
  of the blocker list.

Notebook panel scratch: `/tmp/7e91-mcp-panel-kickoff-session1.md`
(also uploaded to notebook `c442cd0b-ca26-47da-8b12-24db2a1c2613`
as source `panel-session1-kickoff-2026-04-21`).

## Part 1 — API design rules

The rules below are what the public surface must satisfy at tag time.
Each is grounded in concrete code, not aspiration.

### R1. Minimal Transport interface; one signature at each layer

Transport is a two-level contract:

- `Transport` — `Dial(ctx) (io.ReadWriteCloser, error)`. The atomic
  connection shape used by stdio, SSE, WebSocket, and streamable
  adapters in `transport.go`, `transport_sse.go`,
  `transport_streamable.go`, `transport_websocket.go`.
- `StreamableTransport` — extends `Transport` with bidirectional
  `Connection` semantics for SSE-style persistent streams. Lives in
  `transport_streamable.go`.

The rule is not "one transport everywhere." Each altitude has exactly
one canonical signature; adapters (`ReadWriteCloserTransport`,
`TransportFunc`) conform to the lower one. Custom transports implement
`Transport`; streaming-aware transports implement `StreamableTransport`.

### R2. Constructor-per-struct, no hidden global state

`NewClient(transport)`, `NewServer(...)`, `NewEnhancedServer(...)` are
the standard forms. No singleton registries, no global middleware
ambient state, no `init()`-side-effects for feature enable/disable.
Middleware config is a value (`ServerMiddlewareConfig`) passed in; the
registry in `middleware_registry.go` is per-server, not process-wide.

Why: a v1 server must be usable in a test harness that spins up 20
isolated servers in parallel without cross-contamination.

### R3. No panics at protocol boundaries; transport layer may still panic on programmer bugs

Handler surfaces (`CallToolHandlerFunc`, `ReadResourceHandlerFunc`,
`GetPromptHandlerFunc`) return errors. The server's panic-recovery
middleware (`middleware.go`) converts any handler panic into a
JSON-RPC error response. Transport framing (`framer.go`; see B3) is
allowed to panic on programmer misuse because a
misframed message is a bug, not a runtime condition.

Zero manual `IsNil` panic wrappers in the public surface.

### R4. Dependency hygiene: stdlib + golang.org/x/* only at runtime

Runtime dependencies stay limited to the stdlib plus `golang.org/x/*`,
except for explicitly approved v1 exceptions. Test dependencies
(`rsc.io/script` for mcpscripttest, `google/go-cmp` for diffs) are
acceptable and do not propagate to consumers. No `replace` directives
in the root `go.mod` at tag time.

Why: MCP is a protocol library. The moment it pulls in Prometheus,
OTel exporters, or Redis adapters at runtime, it becomes a framework.
Those integrations live in `ext/` or `exp/` subpackages that consumers
opt into.

Current state: `cmd/mcp` has been split into its own module, so the
root no longer carries Cobra or Bubble Tea. Two named runtime
exceptions remain at v1:

- `github.com/gorilla/websocket`, because `transport_websocket.go`
  exports `WebSocketTransport` from the root package and exposes
  `*websocket.Dialer` in the public API.
- `github.com/santhosh-tekuri/jsonschema/v5`, because `security.go`
  exports `JSONSchemaValidator` and the validation middleware helpers
  that construct it from the root package.

Those two deps are not honest-to-remove as a last-minute prune; doing
so would be an API decision rather than dependency hygiene.

### R5. Internal is actually internal

`internal/` packages (`internal/mcpcli`, `internal/mcpspy`,
`internal/jsonrpc2shim`, `internal/jsonrpc2util`,
`internal/integration_testing`) are private. They can churn. The
public surface is the root `mcp` package plus `modelcontextprotocol/`,
plus any deliberately-promoted packages (see R8 below).

Why: contributors reading godoc need to know what they can depend on
and what's fair game for next-cycle redesign.

### R6. Decomposed extension points over god-objects

MCP's extension contracts:

- Handler types: `CallToolHandlerFunc`, `ReadResourceHandlerFunc`,
  `GetPromptHandlerFunc`, notification handlers.
- Typed APIs via generics: `RegisterTypedToolWithServer`,
  `CallToolTyped` in `typed.go`.
- Middleware factories in `middleware_registry.go` implement a
  registration contract; consumers register custom middleware by
  implementing the factory interface, not by patching the chain.

Why: a v1 API where "to add a middleware you fork the dispatcher"
kills extensibility.

### R7. Generated code is visible and regeneratable

`modelcontextprotocol/` types are hand-maintained against the upstream
spec; any future codegen (see `exp/schema2go`, `exp/json2go`) must use
`.gen.go` suffix, `//go:generate` directives adjacent, and must be
regeneratable from a checked-in schema version.

The draft-spec types under `modelcontextprotocol/draft/` are explicitly
**not part of the v1 stable surface**.

### R8. Examples graduate through the core interface, not by back-door import

`examples/servers/*` are demonstrators. `exp/*` is research tier. When
a pattern becomes load-bearing (e.g., `cmd/mcp` umbrella CLI consuming
an `exp/*` package), graduation is mandatory and happens through the
core interface.

Concrete rule: `cmd/mcp` (the umbrella tool) imports only the root
module plus `modelcontextprotocol/`. It does not import `exp/*`
directly. If it needs a capability currently in `exp/`, either the
capability graduates first (two consumers, API review, test harness),
or the cmd-tool ships that capability inline.

Why: "hardcoded wire-in" is where API contracts die.

## Part 2 — Package design pathway

### At v1 tag — on-disk layout

```
. (root module: github.com/tmc/mcp)
  client.go, server.go, transport.go, middleware.go, ...   Core API
  typed.go                                                 Generic helpers
  modelcontextprotocol/                                    Stable protocol types
    (no draft/)                                            Draft types excluded from v1
  internal/                                                Private helpers
    jsonrpc2shim/ jsonrpc2util/ mcpcli/ mcpspy/ ...
  cmd/
    mcp/                                                   The umbrella CLI (v1 in-scope)
    (optionally) mcp-probe/                                Diagnostic tool if kept
  examples/servers/                                        Demonstrators, not a contract
  ext/                                                     Opt-in integrations (if any)
  testing/                                                 Deferred — see Part 3
  exp/                                                     Research tier — no compat promise
```

### Phase 1 cleanups (pre-tag, hygiene-first)

Executed in the order below. Each is a single-concern commit.

1. **H1** Tag purge is a no-op locally. `origin` has zero tags and the
   local clone has no tags to delete, so there is no destructive cleanup
   to perform.
2. **H2** Committed binaries, logs, trace dumps, screenshots,
   `node_modules/`, and similar artifacts have been purged from VCS.
3. **H3** `go.work` and `go.work.sum` are no longer tracked. Workspace
   files remain local-only.
4. **H4** The ALL_CAPS root-status-doc sweep is effectively closed. No
   tracked ALL_CAPS status docs remain at the repo root; `AGENTS.md` is
   a repository instruction file, not a status report.
5. **H5** Duplicate `exp/` subtrees are gone. Keep
   `exp/changemanagement` and `exp/coverage_viz`.

Then, code-side:

6. **B1** `testing/mcpscripttest` is now a plain subpackage in the root
   module. That was the release blocker. Other intentionally-separate
   modules (`exp/`, `testing/mcptestutil`, `testing/mcpscripttest/fuzzing`,
   selected examples, and tool-specific nested modules) remain.
7. **B3** `framer.go` and `framer_test.go` are committed at the repo root.
8. **B4** The in-flight diffs in `client.go`, `server.go`, and
   `internal/mcpcli/session.go` are no longer an outstanding gate.
9. **B2** `ErrTransportClosed` is standardized across transports via
   `errors.Is`.
10. **cmd/ trim** is closed. Keep `cmd/mcp` and `cmd/mcp-probe`; move the
    remaining non-core tools under `exp/cmd/`.

### Phase 2 deferred to v1.x (stabilize-then-extract)

- **exp/** stays in-repo through v1 under the `github.com/tmc/mcp/exp`
  module. Consumers importing
  `github.com/tmc/mcp/exp/...` accept the "no compatibility promise"
  that the path implies — same convention as `golang.org/x/exp/`.
  Re-evaluate extraction to `github.com/tmc/mcp-exp` once any exp/
  subpackage acquires a second stable consumer.
- **testing/mcpscripttest** stays in-repo through v1 but is not part
  of the v1 public surface. Graduates to its own repository
  (`github.com/tmc/mcp-test` or similar) when the scripttest dialect
  stabilizes. Mills: "incredible piece of engineering, but shouldn't
  block a protocol library release."
- **Advanced middleware** (caching, compression, OTel, Prometheus)
  mentioned in `MIDDLEWARE_README.md` lives in `ext/` or `exp/`
  through v1 to prevent bloating `go.mod`. Only the v1 core —
  logging, recovery, timeout, basic auth — ships under the root.
- **17 cmd/ tools** move to `exp/cmd/` with the understanding that
  any of them can graduate back to `cmd/` by passing R8's graduation
  bar (two consumers, API review, test harness).
- **modelcontextprotocol/draft/** stays isolated through v1. Draft
  spec types are not part of the stable surface and churn with the
  upstream MCP spec.
- **Streamable transport polish** — current implementation is
  experimental. Landing as stable requires a second implementation
  (alternate server) and a session-resumption conformance suite.
  Deferred to v1.x.

### Non-goals for v1 package design

- Windows support beyond what CI validates for the root package.
- Public stabilization of the mcpscripttest DSL.
- Plugin architectures from `exp/foundation`.
- Kubernetes operator mode (currently gated behind `//go:build k8s`;
  stays gated).
- Full OpenTelemetry integration. Metrics hooks exist; the default
  build doesn't depend on OTel packages.

## Part 3 — Release pathway

### Module strategy — corrected 2026-04-22

**Root module plus explicit submodules.** The repository is not a single
module, and the synthesis should stop pretending otherwise.

Current state:

- Root module: `github.com/tmc/mcp`
- `exp/` module: `github.com/tmc/mcp/exp`
- `testing/mcpscripttest` is now part of the root module
- Separate modules still present by design: `exp/cmd/mcp2go`,
  `exp/cmd/mcptrace-to-otel`, `testing/mcpscripttest/fuzzing`,
  `testing/mcptestutil`, and several example-server directories

The release blocker was not "delete every nested `go.mod`". The blocker
was that root tests imported `testing/mcpscripttest` across a module
boundary and therefore failed on a fresh clone unless the developer had a
workspace. Closing B1 required absorbing that package into the root
module. It did not require flattening the entire repo.

The Amsterdam concern ("exp/ breakage forces a major bump of the
entire module") is still handled by convention as much as by layout:
`github.com/tmc/mcp/exp/...` import paths signal "no compatibility
promise" to consumers — same contract as `golang.org/x/exp/`. v1 does
not ship breaking changes *from exp/* under the stable surface; if a
consumer depends on an exp/ path and it breaks, that's explicitly
within the no-compat-promise zone.

### Gates (status)

**Hygiene gates** (H1..H5 — from Part 2 Phase 1):

- H1: tag purge → NO-OP locally (`origin` has zero tags; local clone has none)
- H2: binary/log/trace/node_modules purge → CLOSED
- H3: go.work de-track → CLOSED
- H4: ALL_CAPS.md → docs/archive/ move → EFFECTIVELY CLOSED
- H5: exp/ duplicate-subtree delete → CLOSED

**Code gates** (B1..B4 — from Part 2 Phase 1):

- B1: absorb `testing/mcpscripttest` into the root module → CLOSED
- B2: ErrTransportClosed standardization → CLOSED
- B3: framer.go/framer_test.go committed at root → CLOSED
- B4: dirty core .go files resolved → CLOSED
- cmd/ trim: keep `cmd/mcp` + `cmd/mcp-probe`, move the rest to `exp/cmd/` → CLOSED

**Second-round gates** — defined after H1..H5 land and the panel
re-consults against a clean tree. Expected to include: v1 API freeze,
conformance harness against upstream spec, cross-language
interop test (at least one non-Go client), performance baseline.

### Stabilize-then-extract: four items deferred

- **S1 `testing/mcpscripttest` extraction to own repo — deferred.**
  Stays in-repo as a non-public build target through v1. Graduates
  when the scripttest dialect is stable and a second external
  consumer adopts it.
- **S2 `exp/` extraction to `github.com/tmc/mcp-exp` — deferred.**
  No compat promise inside `exp/` per module-strategy decision;
  extraction happens only if/when exp/ subpackages stabilize enough
  to need semver.
- **S3 Streamable transport v1-stable promotion — deferred.**
  Implementation lives in `transport_streamable*.go`; needs a second
  implementation and a session-resumption conformance harness before
  it can be part of the stable contract.
- **S4 Advanced middleware (caching, compression, OTel) in core —
  deferred.** Stays under `ext/` or `exp/` to keep `go.mod` lean.

### v1 hard blockers that remain (beyond Phase 1)

To be enumerated in `v1-release-exemplary-gate.md` after H1..H5 land.
Expected categories:

- **API shape:** freeze root-package exports; confirm the
  `ReadResourceResult` custom unmarshaling; typed-API generics
  stability.
- **Protocol compliance:** pass a conformance suite against the
  stable upstream MCP spec.
- **Interop:** at least one working non-Go client (Python SDK or
  TypeScript SDK) driving a tmc/mcp server through every transport.
- **Performance baseline:** benchmarks in `benchmark_test.go` locked
  to tracked baselines with tolerance.
- **Auth/security:** the race fix in `auth_security_test.go` already
  lands (2-line mutex add, verified 2026-04-21); broader audit of
  timing attacks and RNG fallback per existing TODOs.

### Tag mechanics

1. `./scripts/check-release-version-coherence.sh` (to be written)
2. `go test ./...` (clean, race-free)
3. External-consumer smoke: `go get github.com/tmc/mcp@<rev>` in a
   scratch module; `go build` works against the public surface.
4. Interop smoke: Python SDK client → tmc/mcp server, all transports.
5. Conformance harness pass: `mcpscripttest` full suite.
6. `git tag -a v1.0.0 -m "mcp v1.0.0"`; push.

### Merge strategy

Interactive rebase of the 199 `next`-ahead commits into 6–8 macro
commits before tag, grouped by theme: hygiene purge, module
consolidation, transport stabilization, middleware core, typed API
finalization, cmd/mcp, docs.

Preserves architectural context without squashing 199 commits into
one unreviewable diff.

## Part 4 — Contributor onboarding pathway

### C1. CONTRIBUTING.md that says what's in scope

Three tiers:

- **Core tier**: root `mcp` package, `modelcontextprotocol/`,
  `internal/*` (private but in-scope for issue reports), `cmd/mcp`.
  Changes require
  design-doc justification for any API surface change. New transport
  implementations are in scope via the `Transport` /
  `StreamableTransport` interfaces.
- **Examples tier**: `examples/servers/*`. Lower bar. Demonstrators
  land with tests + short README. Graduation to core via R8's bar.
- **Research tier**: `exp/*`, `testing/mcpscripttest`, draft-spec
  types under `modelcontextprotocol/draft/`. Unstable intentionally.

### C2. Issue templates that route

- Bug: reproducible transcript (trace dump via `mcp-probe` or
  `mcpspy`) + Go version + OS + transport.
- Feature: mapped to a design doc under `docs/design/` or a clear
  extension-interface proposal.
- Performance: benchmark name + baseline + observed delta.
- Spec compliance: a failing `mcpscripttest` test case, or a pointer
  to the upstream MCP spec section.

### C3. Extension points for outside contribution

The interfaces most likely to attract PRs:

- `Transport` — new transport implementations (QUIC, named-pipes,
  etc.).
- Middleware factory interface — new cross-cutting concerns.
- `CallToolHandlerFunc` and the typed-API generics — tool-author
  ergonomics.
- Custom codecs: wiring alternate framers into `framer.go`.

Each needs: godoc with a minimal example, a scripttest conformance
entry, a benchmark entry.

### C4. Performance claim methodology

Benchmarks in `benchmark_test.go`, `benchmark_middleware_test.go`,
`benchmark_auth_test.go`. v1 requires:

- A tracked baseline per benchmark (committed JSON).
- A `benchstat` comparison gate in CI.
- Hardware normalization: ratio to local baseline, not absolute
  throughput.

Currently ad-hoc. To be wired into CI before tag.

### C5. Fast path for new-transport contribution

Without a template, new-transport PRs turn into multi-round reviews
teaching the contributor how `Dial`, `Connection`, and session
management fit together. Ship `docs/contributing/adding-a-transport.md`
with a minimal working example and a scripttest conformance fixture.

## Part 5 — Resolved questions (from panel session 1)

1. **Module strategy?** Root module plus explicit submodules. The
   blocker was the `testing/mcpscripttest` boundary, which is now
   gone; `exp/` remains a separate in-repo module with the usual
   `golang.org/x/exp`-style no-compat-promise convention.

2. **Tag purge?** No-op locally. `origin` has zero tags and the local
   clone has no tags to delete, so there is no release-hygiene work
   left here unless a future remote state changes.

3. **auth_security_test.go diff?** Verified 2026-04-21. The
   modification is a 2-line addition of `baseProvider.mu.Lock()` /
   `Unlock()` around a map write in `TestConcurrentTokenOperations`
   at line 397. Matches the TESTING_STATUS claim of a mutex-lock race
   fix precisely. Not a new blocker — stash or commit as-is.

4. **cmd/ trim strategy — resolved 2026-04-22.** Keep `cmd/mcp` (the
   umbrella CLI) and `cmd/mcp-probe`. Move the remaining non-core
   commands to `exp/cmd/`.

## Part 6 — Remaining gaps and open questions

1. **Q3 — resolved.** Keep `cmd/mcp-probe` at the repo root alongside
   `cmd/mcp`. It remains a standard first-contact diagnostic flow and
   is still close enough to core protocol interaction to justify
   staying out of `exp/cmd/`.

2. **Q4 — resolved.** The repo-local `jsonrpc2/` package was removed
   after auditing direct consumers. `cmd/mcp-probe` now uses the
   official SDK public `github.com/modelcontextprotocol/go-sdk/jsonrpc`
   package from its own nested module, and no root-package public API
   depended on the old path.

3. **Conformance harness against upstream spec.** mcpscripttest is
   powerful but it's a tmc/mcp dialect. A v1 library needs a
   conformance suite that proves compliance against the canonical
   MCP spec, ideally by running a canonical-spec-authored test suite.
   Does one exist upstream? To investigate.

4. **Interop baseline.** Panel expects a non-Go client driving a
   tmc/mcp server through every transport as a v1 gate. Which client
   do we pick (Python SDK, TypeScript SDK, Anthropic's reference)?
   Needs a call.

5. **go.mod runtime-dep audit.** `cmd/mcp` is out of the root module.
   Confirm the remaining named `R4` exceptions still match the public
   root API, and avoid adding new ones.

6. **Auth/security broader audit.** The 2-line race fix is done, but
   an earlier audit identified 4 critical issues (RNG fallback,
   timing attacks, race conditions) and 8 medium-risk issues per
   `SECURITY_COMPLIANCE_REPORT.md`. Status of each at v1 time needs
   to be named explicitly in `v1-release-exemplary-gate.md`.

7. **Performance baselines in CI.** No current automated gate.
   Panel (Mills) expects one before v1.

## Decision log

- **2026-04-21 draft.** Initial synthesis written from notebook panel
  session 1 (notebook `c442cd0b-ca26-47da-8b12-24db2a1c2613`). Seven
  voices: Pike, Cox, Cheney, Mills, Ajmani, Amsterdam, Fitzpatrick
  (adding the 7th voice for repo hygiene vs mlx-go's six was the
  right call — tree state is half the problem here).

- **2026-04-21 module-strategy draft.** Session 1 proposed a single
  root module. Session 2 corrected this: the repo still has explicit
  submodules, but `testing/mcpscripttest` no longer crosses a module
  boundary from root tests.

- **2026-04-21 tag-purge dry-run completed.** Session 1 treated orphan
  local tags as a possible release gate. Session 2 corrected the
  ground truth: `origin` has zero tags and the local clone has no tags,
  so H1 is a no-op in the current tree.

- **2026-04-22 cleanup sweep.** Corrected the synthesis to match the
  tree instead of the session-1 assumptions. `testing/mcpscripttest`
  was absorbed into the root module, `ErrTransportClosed` was
  standardized, the framer pair and dirty core files were resolved,
  duplicate `exp/` subtrees were deleted, and the non-core `cmd/*`
  tools moved to `exp/cmd/` while `cmd/mcp` and `cmd/mcp-probe`
  stayed at the repo root. The repo still contains explicit submodules;
  the blocker was the `testing/mcpscripttest` boundary, not the mere
  existence of nested `go.mod` files.

- **2026-04-21 auth_security_test.go diff reviewed.** 2-line mutex
  addition around a map write. Matches TESTING_STATUS claim; not a
  new blocker. Cheney can sign off in session 2.
