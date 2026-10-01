// Package mcpcorpus extracts tools/call arguments from MCP traces as fuzz seeds.
// Extraction requires an explicit tool name and never executes tools. It reads
// the entire trace before returning seeds, so malformed or truncated records
// cannot produce a partial corpus.
//
// Seeds preserve the raw JSON object argument bytes, including property order
// and interior whitespace. Deduplication compares bytes, not JSON meaning.
// Missing or null selected arguments are errors: extraction does not invent an
// empty object. Requests across all recorded directions and sessions are included;
// callers should provide a trace appropriate to their target and visibility.
// Extraction validates request shapes, not session lifecycle or tool schemas.
package mcpcorpus
