# Security policy

Report vulnerabilities through a [private security advisory](https://github.com/tmc/mcp/security/advisories/new). Include the affected revision, reproduction steps, and impact.

This toolkit uses the official Go MCP SDK for protocol handling. Applications
remain responsible for server authorization, transport configuration, and the
permissions of tools they expose.

Trace recordings, catalog snapshots, and fixtures can contain sensitive request
arguments and response content. Review them before storing or sharing them.
Fixture checks and replay commands execute operations against a server; use a
server intended for the test and review captured expectations first.

Experimental components under `exp` have separate limitations documented in
their packages. This repository does not claim a completed security audit or
compliance certification.
