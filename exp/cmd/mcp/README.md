# Experimental MCP client

This command connects to MCP servers and provides tool, resource, prompt, root,
log, and interactive dashboard commands. It is a separate Go module to keep
Bubble Tea and Lip Gloss out of the root toolkit dependencies.

Build or install it from this checkout:

```sh
cd exp/cmd/mcp
GOWORK=off go install .
```

Run `mcp --help` for commands and connection options. The local replacement of
`github.com/tmc/mcp` points to the repository root for the shared client support
package. The root `go build ./...` excludes this module; `make gates` and
the CI module matrix check it separately.
