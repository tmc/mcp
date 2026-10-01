# MCP Go toolkit

This repository provides tools built on [`github.com/modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk). It does not implement the MCP protocol.

## Packages

- [`mcptrace`](./mcptrace) reads, writes, and records MCP traces at byte boundaries.
- [`mcpcatalog`](./mcpcatalog) captures tool catalogs and compares declared APIs.
- [`mcpcorpus`](./mcpcorpus) extracts recorded tool arguments into Go fuzz seeds or JSON files.
- [`mcpfixture`](./mcpfixture) converts an isolated recorded call into an editable regression fixture.
- [`mcptest`](./mcptest) connects SDK peers for tests with automatic cleanup.
- [`mcpanalysis/toolinputschema`](./mcpanalysis/toolinputschema) checks direct SDK tool registrations for missing input schemas.
- [`mcpmiddleware`](./mcpmiddleware) provides method middleware for the SDK. The v0.1 API includes `Log`.
- [`mcpscripttest`](./mcpscripttest) runs MCP-oriented script tests and installs toolkit commands.

## Commands

The trace tools are `mcpspy`, `mcpcat`, `mcpnorm`, `mcpdiff`, and `mcp-replay`. Experimental clients and development tools live under `exp/cmd`.
The interactive `mcp` client is a separate module so its terminal UI dependencies
stay outside the root toolkit module; build it with
`(cd exp/cmd/mcp && GOWORK=off go install .)`.

The following commands build on the toolkit packages:

```sh
go install ./cmd/mcpinspect ./cmd/mcpapi ./cmd/mcpvet ./cmd/mcpfixture ./cmd/mcpcorpus
mcpinspect session.mcp
mcpapi snapshot -scope development -principal local -client default -- ./server > api.json
mcpapi diff api.json candidate.json
go vet -vettool="$(command -v mcpvet)" ./...
```

`mcpinspect` pairs observed requests and responses without asserting protocol
lifecycle correctness. An unmarked trace must contain one session.
`mcpapi` captures tools without calling them. Its comparison checks a bounded
schema subset and reports unknown compatibility for unsupported schemas or
different capture contexts. `mcpvet` checks direct low-level `Server.AddTool`
literals; it does not follow dynamic registrations.

To turn a recorded handshake, optional catalog lists, and tool call into a script fixture:

```sh
mcpfixture -record 4 session.mcp > testdata/regression.txt
```

Review the captured result before adopting it as an assertion. The generated
script runs `mcpfixture check` against the executable named by `MCP_SERVER`.
Conversion supports legacy initialization and SDK discovery in protocol
`2026-07-28`. It preserves client configuration and checks recorded catalog
list prerequisites. Callbacks, earlier state-changing calls, concurrency, and
incomplete prerequisites are rejected.

Compare correlated calls across recordings with:

```sh
go install ./exp/cmd/mcpdiff
mcpdiff -calls before.mcp after.mcp
```

Call comparison ignores request IDs and compares complete parameters and
responses. Sequential repeated calls match by occurrence; overlapping identical
calls are ambiguous. Add `-timing` to compare known request durations. Exit codes
are 0 for equal, 1 for different, and 2 for invalid or ambiguous input.

The [regression walkthrough](./examples/workflow/README.md) records a bug,
reviews the expected result, checks the fix, and compares its API baseline.

To extract arguments for a Go fuzz target accepting one `[]byte` argument:

```sh
mkdir -p testdata/fuzz
mcpcorpus -tool search -out testdata/fuzz/FuzzSearch session.mcp
```

The output directory must be new. Extraction preserves argument bytes,
deduplicates identical inputs, and never calls tools. Use `-format json` for
ordinary argument files. Missing, null, and non-object arguments are rejected.

For in-memory tests, `mcptest.Connect(t, server, client)` returns connected SDK
sessions and registers cleanup. Configure callbacks on the SDK client and use
channels to synchronize handlers; see the package examples and cancellation tests.

Run the root module gates with:

```sh
go build ./...
go vet ./...
go test -race ./...
```

See each package's `doc.go` for usage and [examples](./examples/README.md) for practical workflows.

## License

ISC. See [LICENSE](LICENSE).
