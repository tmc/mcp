// Package mcpcatalog captures tool catalogs and compares their declared APIs.
// Snapshots contain the negotiated protocol, server capabilities, and a caller
// supplied visibility context. They do not include tool results or credentials.
// Prompts, resources, and resource templates are outside this package's scope.
//
// Compare checks a bounded JSON Schema subset: flat objects with primitive
// properties, required fields, string enums, and boolean additionalProperties.
// Other constructs produce Unknown findings, even when their schemas are
// unchanged. Comparisons describe declared schemas, not tool behavior or a
// complete protocol contract.
package mcpcatalog
