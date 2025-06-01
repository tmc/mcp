# Temporary Commit Plan for MCP Repository

## Overview

This document outlines a logical series of commits based on file creation/modification times and semantic relevance. The goal is to create a coherent commit history that groups related changes together while maintaining proper dependencies.

## Approach

### 1. Commit Grouping Principles
- **Semantic Coherence**: Group files that work together functionally
- **Dependency Order**: Earlier commits should not depend on later ones
- **File Timestamps**: Respect the chronological order when possible
- **Atomic Changes**: Each commit should represent a complete, working change
- **Build Integrity**: Each commit should compile and pass tests

### 2. Analysis Method
1. Identify all modified/new files from `git status`
2. Group files by functional area (core protocol, draft extensions, tooling, tests)
3. Order groups by dependencies (core → extensions → tools → tests)
4. Within groups, order by file modification time where relevant
5. Ensure each commit is self-contained and buildable

## Commit Sequence

### Phase 1: Core Protocol Refactoring

#### Commit 1: Remove Rate Limiting
**Files:**
- Delete `ratelimit.go`
- Delete `ratelimit_test.go`

**Message:** "refactor: remove rate limiting functionality"

**Rationale:** Start by removing deprecated functionality to clean the codebase.

#### Commit 2: Refactor Core Protocol Types
**Files:**
- Delete `modelcontextprotocol/schema_20250326.go`
- Add `modelcontextprotocol/types.go`
- Add `modelcontextprotocol/constants.go`
- Add `modelcontextprotocol/helpers.go`
- Add `modelcontextprotocol/options.go`
- Modify `modelcontextprotocol/marshaling.go`
- Modify `modelcontextprotocol/doc.go`

**Message:** "refactor: split monolithic schema into modular components"

**Rationale:** This is the foundational refactoring that splits the large schema file into focused, modular components.

#### Commit 3: Add Core Protocol Tests
**Files:**
- Add `modelcontextprotocol/example_test.go`
- Add `modelcontextprotocol/helpers_test.go`
- Add `modelcontextprotocol/marshaling_test.go`
- Add `modelcontextprotocol/marshaling_fuzz_test.go`
- Add `modelcontextprotocol/marshaling_null_test.go`
- Add `modelcontextprotocol/run_all_fuzz_tests.sh`

**Message:** "test: add comprehensive tests for refactored protocol components"

**Rationale:** Tests for the refactored core should be committed immediately after the refactoring.

### Phase 2: Draft Protocol Extension

#### Commit 4: Add Draft Protocol Types
**Files:**
- Delete `modelcontextprotocol/draft/schema_draft.go`
- Delete `modelcontextprotocol/draft/marshaling.go`
- Add `modelcontextprotocol/draft/types_draft.go`
- Add `modelcontextprotocol/draft/constants_draft.go`
- Add `modelcontextprotocol/draft/helpers_draft.go`
- Add `modelcontextprotocol/draft/marshaling_draft.go`
- Add `modelcontextprotocol/draft/options_draft.go`
- Modify `modelcontextprotocol/draft/doc.go`

**Message:** "feat: add draft protocol extensions with modular structure"

**Rationale:** Draft types build on the stable protocol and should follow the same modular pattern.

#### Commit 5: Add Draft Protocol Tests
**Files:**
- Add `modelcontextprotocol/draft/helpers_draft_test.go`
- Add `modelcontextprotocol/draft/marshaling_draft_test.go`
- Add `modelcontextprotocol/draft/marshaling_draft_fuzz_test.go`
- Add `modelcontextprotocol/draft/marshaling_draft_null_test.go`

**Message:** "test: add comprehensive tests for draft protocol components"

**Rationale:** Tests for draft protocol should follow immediately after the implementation.

### Phase 3: Core Library Updates

#### Commit 6: Update Core Implementation
**Files:**
- Modify `client.go`
- Modify `server.go`
- Modify `types.go`
- Modify `options.go`
- Modify `doc.go`
- Add `id_generating_binder.go`
- Add `preempter.go`
- Add `transport_sse.go`

**Message:** "feat: update core implementation with new protocol structure"

**Rationale:** Core library updates that use the refactored protocol types.

#### Commit 7: Update Core Tests
**Files:**
- Modify `mcp_test.go`
- Add `example_cancellation_test.go`

**Message:** "test: update core tests for new implementation"

**Rationale:** Tests for the updated core implementation.

### Phase 4: Internal Components

#### Commit 8: Add Internal Utilities
**Files:**
- `internal/jsonrpc2util/*` (all files in this directory)
- `internal/jsonrpc2shim/*`
- `jsonrpc2/jsonrpc2.go`

**Message:** "feat: add internal JSON-RPC utilities and shims"

**Rationale:** Internal components that support the core implementation.

### Phase 5: Command Line Tools

#### Commit 9: Add Core CLI Tools
**Files:**
- `cmd/mcp-proxy/*` (entire directory)
- `cmd/mcp-serve/*` (entire directory)
- `cmd/mcp-send/*` (entire directory)
- `cmd/mcp-connect/*` (entire directory)

**Message:** "feat: add core MCP command line tools"

**Rationale:** Essential CLI tools for working with MCP.

#### Commit 10: Update Existing Tools
**Files:**
- Modify `cmd/mcpdiff/main.go`
- Add `cmd/mcpdiff/README.md`
- Add other mcpdiff test files
- Modify `exp/cmd/mcpcolor/main.go`
- Modify `exp/mcpscripttest/scripttest.go`

**Message:** "feat: update existing tools for new protocol structure"

**Rationale:** Updates to existing tools to work with the refactored code.

### Phase 6: Additional Tools and Infrastructure

#### Commit 11: Add Development Tools
**Files:**
- `cmd/mcp-debug/*`
- `cmd/inspect-id/*`
- `cmd/json-test/*`
- Other development and debugging tools

**Message:** "feat: add development and debugging tools"

**Rationale:** Development tools that aid in working with MCP.

### Phase 7: Documentation and Build

#### Commit 12: Add Documentation and Build Files
**Files:**
- `.gitignore`
- `Makefile`
- `*.md` files (README updates, documentation)
- Shell scripts and test scripts

**Message:** "docs: add documentation and build infrastructure"

**Rationale:** Documentation and build support files.

## Implementation Notes

### Verification Steps for Each Commit
1. Run `go build ./...` to ensure compilation
2. Run `go test ./...` to ensure tests pass
3. Check that imports are satisfied
4. Verify no circular dependencies

### Handling Dependencies
- If a file in a later commit is needed by an earlier one, move it to the earlier commit
- Keep test files with their implementation files when possible
- Ensure each commit represents a complete, working state

### Special Considerations
1. The `go.mod` and `go.sum` files should be updated in the commit where dependencies change
2. Binary files and generated code should be committed separately
3. Large refactorings might need to be split into smaller, more manageable commits

## Execution Plan

1. Create a new branch for the reorganization
2. Use `git reset --soft` to unstage all changes
3. Stage and commit files according to this plan
4. Test each commit individually
5. Create a clean commit history

This approach ensures a logical, readable commit history that accurately reflects the evolution of the codebase while maintaining build integrity at each step.