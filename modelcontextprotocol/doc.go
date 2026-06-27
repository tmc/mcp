// Package modelcontextprotocol implements the Model Context Protocol (MCP) schema
// defined by https://github.com/modelcontextprotocol/modelcontextprotocol.
//
// This is a schema-faithful reference rendering of the protocol types, with
// custom JSON marshaling/unmarshaling, functional options, and sealed union
// interfaces (like Content) built from unexported marker methods. It is
// distinct from the root [github.com/tmc/mcp] package: that package defines
// the canonical wire types used by the mcp Client and Server, while this
// package is a standalone schema model used by code generation, validation,
// and example servers. When the two disagree, the root package is canonical.
//
// For the latest DRAFT version of the protocol, see the ./draft subdirectory.
package modelcontextprotocol
