# MCP Testing Infrastructure Refactoring TODO

## Current State Analysis
- ✅ Core mcpscripttest split into minimal core + extensions
- ✅ Integration testing framework created
- ✅ mcptestutil package working
- ⚠️ Some tests failing due to missing commands
- ⚠️ Extension packages have placeholder implementations
- ⚠️ Integration test modules need proper setup

## TODO List (Priority Order)

### 1. Fix Critical Test Failures (HIGH PRIORITY)
- [x] Fix race condition in `auth_security_test.go` (Resolved: Added mutex locks in TestConcurrentTokenOperations)
- [ ] Fix chmod mode issue in TestAllTestdata
- [ ] Add proper skip conditions for tests requiring missing tools
- [ ] Fix or skip tests that use unavailable commands (testgraph, testcallgraph)
- [ ] Update test scripts to use available commands

### 2. Implement Core Extension Functionality (HIGH PRIORITY)
- [ ] Move actual command implementations from internal to extensions
- [ ] Implement setstdin sharing between core and extensions
- [ ] Wire up server management state between serverext and core
- [ ] Test that extensions actually work with real commands

### 3. Complete Integration Testing Setup (MEDIUM PRIORITY)
- [ ] Add proper dependencies to integration test go.mod files
- [ ] Create basic interop tests between implementations
- [ ] Add README with setup instructions for each module
- [ ] Add CI-friendly skip conditions

### 4. Documentation and Examples (MEDIUM PRIORITY)
- [ ] Create example showing minimal mode usage
- [ ] Create example showing extension usage
- [ ] Update main README with new architecture
- [ ] Document migration path for existing tests

### 5. Clean Up and Polish (LOW PRIORITY)
- [ ] Remove debug/test files
- [ ] Consolidate duplicate code
- [ ] Add package-level documentation
- [ ] Create comprehensive test coverage report

### 6. Git Operations (FINAL)
- [ ] Stage all changes in logical groups
- [ ] Create descriptive commit messages
- [ ] Update CLAUDE.md files if needed