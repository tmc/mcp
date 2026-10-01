// Package mcptracediff compares observed JSON-RPC calls across MCP traces.
// It matches direction, session label, method, and complete parameters, ignoring
// per-run IDs. Sequential repeated calls match by occurrence when counts agree;
// overlapping identical calls are ambiguous. Notifications are outside scope.
// This package makes no MCP lifecycle or protocol-version compatibility claims.
package mcptracediff
