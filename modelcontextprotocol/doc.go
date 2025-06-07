// Package modelcontextprotocol implements the Model Context Protocol (MCP) schema
// defined by https://github.com/modelcontextprotocol/modelcontextprotocol.
// This package specifically targets the 2025-03-26 version of the schema.
//
// It provides Go types for all messages, parameters, and results, along with
// custom JSON marshaling/unmarshaling logic, functional options for construction,
// and helper methods for common operations. Types that form closed unions
// (like Content) are implemented as "sealed" interfaces using unexported marker methods.
// Grouping interfaces (like ClientRequest) are also sealed to ensure type safety
// when working with known MCP methods.
//
// For the latest DRAFT version of the protocol, see the ./draft subdirectory.
package modelcontextprotocol
