# From a trace to a regression

The `square` tool accidentally adds its argument to itself. Calling it with
`n=3` returns `6`; its documented behavior requires `9`. This walkthrough uses
real SDK messages and toolkit commands to turn that observation into a test.
The `-fixed` flag selects the corrected implementation without changing its API.

From the toolkit checkout, build the commands into a fresh working directory:

```sh
mkdir -p "$HOME/tmp"
WORK=$(mktemp -d "$HOME/tmp/mcp-workflow.XXXXXX")
GOWORK=off go build -o "$WORK/mcpinspect" ./cmd/mcpinspect
GOWORK=off go build -o "$WORK/mcpfixture" ./cmd/mcpfixture
GOWORK=off go build -o "$WORK/mcpapi" ./cmd/mcpapi
GOWORK=off go build -o "$WORK/mcpdiff" ./exp/cmd/mcpdiff
(cd examples && GOWORK=off go build -o "$WORK/workflow" ./workflow)
```

Record the failure, then inspect the conversation:

```sh
"$WORK/workflow" record "$WORK/bad.mcp"
# 6
"$WORK/mcpinspect" "$WORK/bad.mcp"
CALL=$("$WORK/mcpinspect" "$WORK/bad.mcp" | awk '/send request.*tools\/call/ {print $1}')
"$WORK/mcpfixture" -record "$CALL" "$WORK/bad.mcp" > "$WORK/observed.txt"
```

The recorder uses the SDK's `IOTransport` with `mcptrace.ReadCloser` and
`WriteCloser` at the byte boundary. Client writes are `send`; reads are `recv`.
It first discovers the server and lists tools, then calls `square`. It defaults
to MCP `2026-07-28`; `record -protocol 2025-11-25` exercises legacy initialization.
It opens new output files exclusively so rerunning a command cannot replace a
trace accidentally.

The generated txtar contains the **observed** response `6`. Keeping that response
as the assertion would preserve the bug. Review the fixture and explicitly set
its expected text to the correct result:

```sh
"$WORK/workflow" expect -text 9 "$WORK/observed.txt" "$WORK/regression.json"
"$WORK/mcpfixture" check "$WORK/regression.json" -- "$WORK/workflow" server
# exits 1: observed result changed; got 6, want 9
"$WORK/mcpfixture" check "$WORK/regression.json" -- "$WORK/workflow" server -fixed
# exits 0
```

The example's `expect` command extracts `fixture.json` from the generated txtar
and changes only the expected result to the developer-supplied text. For your
own server, edit the generated fixture to express its intended behavior.

Record the corrected run and compare calls. This comparison ignores run-specific
request IDs while retaining method, parameters and result differences:

```sh
"$WORK/workflow" record -fixed "$WORK/good.mcp"
# 9
"$WORK/mcpdiff" -calls "$WORK/bad.mcp" "$WORK/good.mcp"
# exits 1: the square response changed
"$WORK/mcpdiff" -calls "$WORK/good.mcp" "$WORK/good.mcp"
# exits 0
```

The behavior changed, but the public discovery contract stayed the same:

```sh
"$WORK/mcpapi" snapshot -scope local -principal anonymous -client workflow -- \
  "$WORK/workflow" server > "$WORK/before.json"
"$WORK/mcpapi" snapshot -scope local -principal anonymous -client workflow -- \
  "$WORK/workflow" server -fixed > "$WORK/after.json"
"$WORK/mcpapi" diff "$WORK/before.json" "$WORK/after.json"
# exits 0: no API changes
```

`workflow_test.go` builds and runs these actual binaries for both protocol
versions. It requires the faulty server to fail the reviewed fixture, the fixed
server to pass, the call comparison to detect the changed response, and the API
comparison to remain equal. Subprocesses have deadlines; the recorder closes its
SDK sessions and pipe endpoints. No sleeps, remote services or credentials are
needed.

```sh
(cd examples && GOWORK=off go test -race ./workflow)
```
