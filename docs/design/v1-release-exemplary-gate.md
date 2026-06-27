# v1 Release Exemplary Gate

This document is the binary checklist for tagging `v1.0.0`.

The tag is allowed only when every open item below is closed with
in-tree evidence. Narrative status pages and historical notebook notes
do not close gates on their own.

## Phase-1 prerequisites already closed

These items were the hygiene and stabilization work required before a
serious v1 review could begin.

- `H1` orphan-tag purge: local no-op in the current clone
- `H2` tracked binary and artifact purge: closed
- `H3` `go.work` hygiene: closed
- `H4` root-doc sprawl cleanup: effectively closed
- `H5` duplicate `exp/` subtree removal: closed
- `B1` absorb `testing/mcpscripttest` into the root module: closed
- `B2` standardize transport closure errors on `ErrTransportClosed`: closed
- `B3` commit `framer.go` and `framer_test.go` at root: closed
- `B4` dirty core `.go` files resolved: closed
- `cmd/` trim: closed; only `cmd/mcp` and `cmd/mcp-probe` remain

## Closed hard blockers

These were verified against the tree on 2026-06-27 (HEAD `bb582aa97`) by
running each gate's recorded command; the scoping pass is captured in
[`b5-b10-scoping-2026-06-27.md`](b5-b10-scoping-2026-06-27.md).

- `B5` security evidence and auth hardening: closed. All ten named auth
  tests pass and `go test -race ./...` is CI-wired
  (`.github/workflows/ci.yml`). `SECURITY.md` was rewritten so every claim
  maps to a code location, a named check, or an explicit deferral
  (criterion 3). Full criteria below under "B5".
- `B6` root-module dependency contract: closed.
  `scripts/check-root-dep-contract.sh` exits 0
  (`root runtime dependency contract satisfied`) and is wired in CI
  (`.github/workflows/ci.yml`) and `make check-deps`. Runtime modules are
  stdlib + `golang.org/x/*` + the two approved R4 exceptions
  (`gorilla/websocket`, `santhosh-tekuri/jsonschema/v5`); test-only modules
  are correctly segregated. Full criteria below under "Closed-blocker
  evidence: B6".
- `B9` performance baseline in CI: closed. `scripts/bench-gate.sh` exits 0
  with both gated benchmarks inside tolerance
  (`BenchmarkServer_HandleRequest/PayloadSize_1024` 0.89x ns/op;
  `BenchmarkTokenValidation` 0.93x ns/op, 0 B/op, 0 allocs/op) against the
  committed `testdata/benchmarks/b9-baseline.txt`, wired in
  `.github/workflows/ci.yml`. Full criteria below under "Closed-blocker
  evidence: B9".
- `B10` `jsonrpc2` boundary decision: closed. The repo-local `jsonrpc2/`
  package is removed; no non-test root `.go` file references
  `github.com/tmc/mcp/jsonrpc2`; the root build uses
  `golang.org/x/exp/jsonrpc2` and `cmd/mcp-probe` consumes the upstream
  public `github.com/modelcontextprotocol/go-sdk/jsonrpc` from its own
  nested module. Full criteria below under "Closed-blocker evidence: B10".

## Open hard blockers

### B5. Security evidence and auth hardening — CLOSED

Closed 2026-06-27. Current state:

- `SECURITY.md` was rewritten so every retained claim maps to a code
  location or a named regression test, and unbacked claims (TLS
  enforcement, SQL/XXE/path/command-injection "prevention", SOC2/GDPR/HIPAA
  compliance checkmarks, a dated aspirational roadmap, a non-existent
  `FuzzInputValidation` target) were removed or moved to an explicit
  "not provided / deferred" list. Criterion 3 is now satisfied.
- Security claims touch `auth.go`, `auth_security.go`, `security.go`,
  and `middleware.go`.
- The auth path has direct tests for entropy failure handling,
  constant-time secret validation, token-race behavior, context-value
  sanitization, rate-limit granularity, key derivation, production
  error sanitization, and CORS defaults.
- The named subset and `go test -race ./...` pass at HEAD (verified
  2026-06-27); see [`b5-b10-scoping-2026-06-27.md`](b5-b10-scoping-2026-06-27.md).

Acceptance criteria:

1. `auth.go` does not silently degrade random token or client-secret
   generation. Entropy failures must return errors, not fall back to
   timestamps or other predictable values.
2. Client-secret comparison in `auth.go` is constant time.
3. Every issue claimed in `SECURITY.md` is mapped to one of:
   - a named automated check
   - an explicit non-v1 deferral
   - a removed or corrected claim
4. Minimum verification is recorded and passes:
   - targeted auth tests, including random-generation failure handling
   - token-integrity and concurrent-token validation tests
   - `go test -race ./...`

Claim-to-evidence map:

1. Weak random generation fallback
   Evidence: `TestGenerateRandomString_ReadError`, `TestMemoryOAuthProvider_RegisterClient_ReadError`, `TestMemoryOAuthProvider_CreateAccessToken_ReadError`
2. Client-secret timing attack
   Evidence: `TestMemoryOAuthProvider_ValidateClient`
3. Token validation race condition
   Evidence: `TestConcurrentTokenOperations`, `go test -race ./...`
4. Context value injection into token metadata
   Evidence: `TestSecureOAuthProvider_ExtractClientInfoSanitizesContextValues`
5. Per-endpoint rate-limit granularity
   Evidence: `TestRateLimitMiddleware_PerEndpointLimiting`, `TestEnhancedRateLimitMiddleware`
6. Key derivation hardening
   Evidence: `TestDeriveKeyMethods`
7. Production error sanitization
   Evidence: `TestSanitizeErrorModes`
8. Secure CORS defaults
   Evidence: `TestNewCORSMiddlewareDefaults`

Minimum recorded verification:

```bash
go test -run 'TestGenerateRandomString_ReadError|TestMemoryOAuthProvider_RegisterClient_ReadError|TestMemoryOAuthProvider_CreateAccessToken_ReadError|TestMemoryOAuthProvider_ValidateClient|TestSecureOAuthProvider_ExtractClientInfoSanitizesContextValues|TestRateLimitMiddleware_PerEndpointLimiting|TestDeriveKeyMethods|TestSanitizeErrorModes|TestNewCORSMiddlewareDefaults|TestConcurrentTokenOperations' ./
go test -race ./...
```

Evidence anchors:

- `auth.go`
- `auth_security.go`
- `auth_security_test.go`
- `security.go`
- `SECURITY.md`

### Closed-blocker evidence: B6 — Root-module dependency contract — CLOSED

Closed 2026-06-27. Current state:

- `cmd/mcp` is now an explicit submodule, so the Cobra/TUI dependency
  tree no longer pollutes the root module.
- `scripts/check-root-dep-contract.sh` is the reproducible local
  verifier for this gate and is exposed as `make check-deps`.
- The root `go.mod` still carries two non-stdlib, non-`golang.org/x/*`
  dependencies, and both are part of the root package surface today.
- `github.com/gorilla/websocket` is required by the exported
  `WebSocketTransport` in `transport_websocket.go`, including the
  `WithDialer(*websocket.Dialer)` method that exposes the gorilla type
  directly in the root API.
- `github.com/santhosh-tekuri/jsonschema/v5` is required by the
  exported `JSONSchemaValidator` in `security.go` and by the
  validation-middleware helpers that construct it.
- Neither dependency is honest-to-remove as a hygiene-only cleanup.
  Removing either one is an API decision, not a pre-tag prune.

Approved v1 exceptions to `R4`:

1. `github.com/gorilla/websocket`
   Rationale: root WebSocket transport support is still part of the
   public `mcp` package.
   `go mod why -m`:
   `# github.com/gorilla/websocket`
   `github.com/tmc/mcp`
   `github.com/gorilla/websocket`
2. `github.com/santhosh-tekuri/jsonschema/v5`
   Rationale: JSON schema validation is optional in practice, but the
   validator type and middleware entry points are exported from the
   root package today.
   `go mod why -m`:
   `# github.com/santhosh-tekuri/jsonschema/v5`
   `github.com/tmc/mcp`
   `github.com/santhosh-tekuri/jsonschema/v5`

Acceptance criteria:

1. Root runtime dependencies are limited to the stdlib plus
   `golang.org/x/*`, except for the named `R4` exceptions recorded
   above.
2. `go mod why -m` output is captured for every approved exception.
3. Test-only and `exp/`-only dependencies are not justified as root
   runtime requirements.

Evidence anchors:

- `go.mod`
- `security.go`
- `transport_websocket.go`
- `cmd/mcp/`
- `scripts/check-root-dep-contract.sh`

### B7. Upstream conformance harness — OPEN

Current state:

- `mcpscripttest` is valuable, but it is a project-local harness.
- The canonical upstream conformance target is
  `@modelcontextprotocol/conformance@0.1.16`.
- `scripts/mcp-conformance.sh` runs that harness against an
  already-running HTTP MCP endpoint and is exposed as `make conformance`.
- A live release-gate run still requires a `tmc/mcp` server URL via
  `MCP_CONFORMANCE_URL`.

Acceptance criteria:

1. One canonical conformance target is chosen and documented.
2. The repo contains a reproducible command or script that runs that
   harness against the v1 surface.
3. The result is part of the release path and not a manual,
   one-off notebook exercise.

Evidence anchors:

- `testing/mcpscripttest/`
- `docs/design/release-readiness-synthesis.md`
- `scripts/mcp-conformance.sh`

### B8. Non-Go interop baseline — OPEN

Current state:

- The baseline non-Go client is the official TypeScript SDK
  `@modelcontextprotocol/sdk`.
- `internal/integration_testing/typescript-sdk-interop` provides the
  first executable smoke path: TypeScript SDK client initialization,
  `tools/list`, and `tools/call` against a `tmc/mcp` stdio server.
- B8 remains open until the smoke path covers every transport still in
  scope for the root v1 surface at tag time.

Acceptance criteria:

1. One baseline client is chosen and documented.
2. The repo contains a reproducible smoke path for that client against
   a `tmc/mcp` server.
3. The smoke covers every transport still in scope for the root v1
   surface at tag time.

Evidence anchors:

- `internal/integration_testing/`
- `docs/design/release-readiness-synthesis.md`
- `internal/integration_testing/typescript-sdk-interop/`

### Closed-blocker evidence: B9 — Performance baseline in CI — CLOSED

Closed 2026-06-27. Current state:

- Benchmarks exist, and the repo now has a narrow regression gate for
  the v1-critical request and auth paths.
- The v1 gate now uses two stable root benchmarks:
  `BenchmarkServer_HandleRequest/PayloadSize_1024` and
  `BenchmarkTokenValidation`.
- The bootstrap baseline lives at `testdata/benchmarks/b9-baseline.txt`.
- CI runs `scripts/bench-gate.sh` on `ubuntu-latest` with Go 1.25.9.

Acceptance criteria:

1. The benchmark subset that matters for v1 is named.
2. Baselines and tolerances are recorded in-tree.
3. CI or a release script fails when those baselines regress beyond
   tolerance.

Recorded subset and tolerance:

1. `BenchmarkServer_HandleRequest/PayloadSize_1024`
   Baseline: `219643 ns/op` best observed latency sample,
   `27077 B/op` median, `6162 allocs/op` median
   Tolerance: `ns/op <= 5.0x baseline`, `B/op <= 1.10x baseline`,
   `allocs/op <= 1.10x baseline`
2. `BenchmarkTokenValidation`
   Baseline: `43.62 ns/op` best observed latency sample,
   `0 B/op` median, `0 allocs/op` median
   Tolerance: `ns/op <= 5.0x baseline`, `B/op == 0`,
   `allocs/op == 0`

Recorded gate command:

```bash
bash ./scripts/bench-gate.sh
```

The committed baseline is a bootstrap capture from `darwin/arm64`.
The gate compares best observed latency samples for `ns/op` so shared
or overloaded runners do not fail solely because of scheduler pauses,
and compares median allocation metrics. The tolerances are intentionally
conservative so CI can gate on large regressions before a dedicated
Linux refresh is recorded.

Evidence anchors:

- `benchmark_test.go`
- `benchmark_auth_test.go`

### Closed-blocker evidence: B10 — `jsonrpc2` boundary decision — CLOSED

Closed 2026-06-27. Current state:

- The repo-local `jsonrpc2/` package was removed after auditing direct
  consumers.
- `cmd/mcp-probe` now uses the official SDK public package
  `github.com/modelcontextprotocol/go-sdk/jsonrpc` from its own nested
  module.
- No root-package public API depends on `github.com/tmc/mcp/jsonrpc2`.

Acceptance criteria:

1. Consumer usage is audited.
2. If there are no external consumers, the repo-local `jsonrpc2/`
   package is removed or made private.
3. The remaining direct consumers build and test against the upstream
   public package.

Evidence anchors:

- `cmd/mcp-probe/`
- `docs/design/release-readiness-synthesis.md`

## Stale items that should stay closed

These should not re-enter the live v1 checklist unless the tree
regresses:

- orphan-tag purge in the current local/remote state
- nested-module breakage from `testing/mcpscripttest`
- duplicate `exp/` subtrees
- pre-cleanup `cmd/` sprawl
- tracked-binary and tracked-artifact cleanup

## Release gate commands

The release candidate must at minimum pass:

```bash
go build ./...
go vet ./...
go test ./...
go test -race ./...
(
  cd cmd/mcp &&
  GOWORK=off go build ./... &&
  GOWORK=off go vet ./... &&
  GOWORK=off go test ./...
)
git status --short
```

Additional commands for conformance, interop, and performance are part
of `B7`, `B8`, and `B9` and must be named before tag time.
