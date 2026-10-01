// Package toolinputschema checks low-level SDK tool registrations for missing
// input schemas. It examines only Tool literals passed directly to
// Server.AddTool; generic mcp.AddTool infers schemas and is not checked.
package toolinputschema
