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

## Open hard blockers

### B5. Security evidence and auth hardening

Current state:

- `SECURITY.md` is narrative, not gate evidence.
- Security claims touch `auth.go`, `auth_security.go`, `security.go`,
  and `middleware.go`.
- The auth path now needs the rest of the security claims reconciled to
  code and tests, not just asserted in markdown.

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

Evidence anchors:

- `auth.go`
- `auth_security.go`
- `auth_security_test.go`
- `security.go`
- `SECURITY.md`

### B6. Root-module dependency contract

Current state:

- `cmd/mcp` is now an explicit submodule, so the Cobra/TUI dependency
  tree no longer pollutes the root module.
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

### B7. Upstream conformance harness

Current state:

- `mcpscripttest` is valuable, but it is a project-local harness.
- The repo still lacks a named conformance gate against the stable
  upstream MCP spec.

Acceptance criteria:

1. One canonical conformance target is chosen and documented.
2. The repo contains a reproducible command or script that runs that
   harness against the v1 surface.
3. The result is part of the release path and not a manual,
   one-off notebook exercise.

Evidence anchors:

- `testing/mcpscripttest/`
- `docs/design/release-readiness-synthesis.md`

### B8. Non-Go interop baseline

Current state:

- The repo does not yet name one non-Go client as the release interop
  baseline.
- The current release path does not prove wire compatibility from a
  non-Go implementation into a `tmc/mcp` server across the transports
  kept in the root v1 surface.

Acceptance criteria:

1. One baseline client is chosen and documented.
2. The repo contains a reproducible smoke path for that client against
   a `tmc/mcp` server.
3. The smoke covers every transport still in scope for the root v1
   surface at tag time.

Evidence anchors:

- `internal/integration_testing/`
- `docs/design/release-readiness-synthesis.md`

### B9. Performance baseline in CI

Current state:

- Benchmarks exist, but the repo does not yet treat performance
  regression as a release gate.

Acceptance criteria:

1. The benchmark subset that matters for v1 is named.
2. Baselines and tolerances are recorded in-tree.
3. CI or a release script fails when those baselines regress beyond
   tolerance.

Evidence anchors:

- `benchmark_test.go`
- `benchmark_auth_test.go`

### B10. `jsonrpc2` boundary decision

Current state:

- `jsonrpc2/` is still public in the repo layout.
- The release pathway still needs a final decision on whether this
  package is part of the stable v1 contract or an implementation
  detail that should move under `internal/`.

Acceptance criteria:

1. Consumer usage is audited.
2. If there are no external consumers, `jsonrpc2/` moves under
   `internal/`.
3. If it stays public, package documentation states its stability
   contract and intended use.

Evidence anchors:

- `jsonrpc2/`
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
