# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Comprehensive documentation suite (README, SECURITY, PERFORMANCE, TESTING_STATUS)
- Complete test coverage for `cmd/mcp-connect` (807 lines)
- Protocol compliance tests for cross-implementation validation
- Security audit documentation with vulnerability fixes
- Performance benchmarking suite with detailed metrics

### Fixed
- Critical benchmark test in `benchmark_test.go:216` - now tests handler directly
- ContentTransformationMiddleware implementation - replaced TODOs with working code
- Weak RNG fallback vulnerability - removed dangerous timestamp fallback
- Timing attack vulnerability - implemented constant-time comparisons
- Token validation race condition - added atomic operations
- Context value injection vulnerability - added input sanitization

### Changed
- Improved test coverage to ~49.4% with comprehensive test suites
- Enhanced middleware system with complete transformation logic
- Updated protocol interoperability tests with full serialization validation

## [0.3.0] - 2025-08-27

### Added
- 15 experimental command-line tools in `exp/cmd-experimental/`
- Comprehensive middleware system with 9 components
- Type-safe APIs using Go generics
- Streamable transport with SSE support
- Foundation libraries for configuration and output formatting

### Fixed
- Duplicate `mockReadWriteCloser` definition in tests
- CI build issues with tools having separate go.mod files
- Authentication and security test compilation errors
- Transport error handling with nil connections

### Security
- Implemented OAuth2 with PKCE support
- Added token rotation policies
- Secure session management with encryption
- Rate limiting with per-client tracking

## [0.2.0] - 2025-07-22

### Added
- WebSocket transport implementation (experimental)
- Advanced middleware components (compression, caching, validation)
- Security middleware with OAuth2 integration
- Performance benchmarking framework
- Integration testing framework

### Changed
- Migrated from conventional commits to Go project style
- Improved error handling across all packages
- Enhanced logging with structured output

### Fixed
- JSON-RPC marshaling issues in mcp-probe
- Build failures in experimental packages
- Test infrastructure stability improvements

## [0.1.0] - 2025-06-25

### Added
- Initial MCP Go implementation
- Core client and server libraries
- stdio and HTTP transport support
- Basic command-line tools (mcp-connect, mcp-probe, mcpspy)
- Example servers (filesystem, time, calculator, echo)
- JSON-RPC 2.0 protocol implementation
- Basic test suite with ~35% coverage

### Security
- Basic authentication framework
- TLS support for HTTP transport
- Input validation for JSON-RPC messages

### Known Issues
- High allocation count in server handler
- Limited middleware functionality
- No WebSocket support
- Basic test coverage

## [0.0.1] - 2025-05-17

### Added
- Project initialization
- Basic repository structure
- MIT License
- Initial README documentation

---

## Version History Summary

| Version | Date | Highlights |
|---------|------|------------|
| Unreleased | - | Documentation overhaul, security fixes, test improvements |
| 0.3.0 | 2025-08-27 | Experimental tools, comprehensive middleware |
| 0.2.0 | 2025-07-22 | WebSocket support, advanced middleware |
| 0.1.0 | 2025-06-25 | Initial implementation with core features |
| 0.0.1 | 2025-05-17 | Project initialization |

## Upgrade Guide

### From 0.2.x to 0.3.x

1. **Middleware Configuration Changes**
   ```go
   // Old
   middleware := NewMiddleware(opts)
   
   // New
   config := &ServerMiddlewareConfig{
       GlobalConfig: &MiddlewareConfig{...},
   }
   server.SetMiddlewareConfig(config)
   ```

2. **Transport API Changes**
   ```go
   // Old
   transport := NewTransport(url)
   
   // New
   transport := NewStreamableHTTPTransport(url)
   ```

### From 0.1.x to 0.2.x

1. **Error Handling Changes**
   - All handlers now return `(*Result, error)` instead of `Result`
   - Context is required for all operations

2. **Configuration Changes**
   - Environment variables now prefixed with `MCP_`
   - Configuration files moved to YAML format

## Deprecation Notices

### Deprecated in 0.3.0
- `SimpleMiddleware` - use `ServerMiddlewareConfig` instead
- `BasicTransport` - use specific transport types
- `LegacyAuth` - migrate to OAuth2Provider

### Removal Timeline
- 0.4.0: Remove deprecated middleware APIs
- 0.5.0: Remove legacy transport interfaces
- 1.0.0: Remove all deprecated features

## Release Process

1. Update `CHANGELOG.md` with release notes
2. Run release checks (`go test ./...`, `go test -race ./...`, core tool builds)
3. Create git tag: `git tag -a v0.x.x -m "Release v0.x.x"`
4. Push tag: `git push origin v0.x.x`
5. Let `.github/workflows/release.yml` publish artifacts and the GitHub release
6. Announce the release in project channels

## Support Policy

| Version | Support Status | End of Support |
|---------|---------------|----------------|
| 0.3.x | ✅ Active | - |
| 0.2.x | 🔧 Maintenance | 2026-02-01 |
| 0.1.x | ⚠️ Security only | 2025-12-01 |
| 0.0.x | ❌ Unsupported | 2025-09-01 |

For migration assistance, see [CONTRIBUTING.md](CONTRIBUTING.md) or open a discussion.
