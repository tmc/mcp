# Changelog

## Unreleased

This repository provides development tools built on the official Go MCP SDK.

- Record and inspect MCP conversations with `mcptrace` and `mcpinspect`.
- Capture tool API baselines and compare supported schemas with `mcpapi`.
- Check direct low-level tool registrations with `mcpvet`.
- Connect SDK peers in tests with `mcptest`.
- Convert recorded calls and catalog prerequisites into editable regression
  fixtures with `mcpfixture`.
- Extract tool arguments into fuzz seeds with `mcpcorpus`.
- Compare correlated calls with `mcpdiff -calls`.
- Log SDK method traffic with `mcpmiddleware`.
- Run MCP script tests with `mcpscripttest`.

See [examples](examples/README.md) for recording and regression workflows.
Experimental commands and packages under `exp` may change without notice.
