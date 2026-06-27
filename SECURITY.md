# Security Policy

## Reporting Security Vulnerabilities

If you discover a security vulnerability in this project, please report it responsibly:

1. **DO NOT** open a public issue
2. Open a private report at: https://github.com/tmc/mcp/security/advisories/new
3. Include: description, steps to reproduce, potential impact
4. We aim to respond within 48 hours

## Security Audit Results

**Last Audit**: August 31, 2025
**Security Update**: October 6, 2025

All critical and medium-risk issues from the audit are resolved, each with a
named regression test (see the claim-to-evidence mapping below).

### Critical Vulnerabilities (Fixed)

#### 1. Weak Random Number Generation (CVE-PENDING)
- **Location**: `auth.go:478-484`
- **Issue**: Fallback to timestamp-based generation if crypto/rand fails
- **Impact**: Predictable tokens vulnerable to timing attacks
- **Status**: ✅ Fixed - removed fallback; entropy failures now return errors
- **Severity**: CRITICAL
- **Gate evidence**: `TestGenerateRandomString_ReadError`, `TestMemoryOAuthProvider_RegisterClient_ReadError`, `TestMemoryOAuthProvider_CreateAccessToken_ReadError`

#### 2. Timing Attack on Secret Comparison
- **Location**: `auth.go:198-213`
- **Issue**: Non-constant time comparison of client secrets
- **Impact**: Client secrets could be revealed through response timing
- **Status**: ✅ Fixed - using `subtle.ConstantTimeCompare`
- **Severity**: HIGH
- **Gate evidence**: `TestMemoryOAuthProvider_ValidateClient`

#### 3. Token Validation Race Condition
- **Location**: `auth_security.go:160-221`
- **Issue**: Race between revocation check and metadata access
- **Impact**: Revoked tokens could be validated in race window
- **Status**: ✅ Fixed - added atomic operations
- **Severity**: HIGH
- **Gate evidence**: `TestConcurrentTokenOperations`, `go test -race ./...`

#### 4. Context Value Injection
- **Location**: `auth_security.go:360-373`
- **Issue**: Direct extraction of unsanitized context values
- **Impact**: Log poisoning, potential code execution
- **Status**: ✅ Fixed - added input sanitization
- **Severity**: MEDIUM
- **Gate evidence**: `TestSecureOAuthProvider_ExtractClientInfoSanitizesContextValues`

### Medium-Risk Issues (All Resolved)

| Issue | Status | Priority |
|-------|--------|----------|
| Insufficient rate limiting granularity | ✅ Fixed | High |
| Token encryption key derivation | ✅ Fixed | Medium |
| Verbose error messages in production | ✅ Fixed | Low |
| Permissive CORS defaults | ✅ Fixed | Medium |

#### 5. Rate Limiting Granularity (FIXED)
- **Location**: `ratelimit.go:714-776`, `middleware.go:315-323`
- **Issue**: Rate limiting only per-client, not per-endpoint
- **Impact**: Unable to apply different limits to different endpoints
- **Fix**: Added `PerEndpointLimiting` configuration option
- **Status**: ✅ Fixed - supports per-endpoint (client:method) granularity
- **Gate evidence**: `TestRateLimitMiddleware_PerEndpointLimiting`, `TestEnhancedRateLimitMiddleware`

#### 6. Key Derivation (FIXED)
- **Location**: `auth_security.go:87-145`
- **Issue**: Simple SHA256 for key derivation from encryption key
- **Impact**: Weaker key derivation than recommended standards
- **Fix**: Implemented Argon2id and PBKDF2 with proper parameters
- **Status**: ✅ Fixed - uses Argon2id (64MB memory, 3 iterations)
- **Gate evidence**: `TestDeriveKeyMethods`

#### 7. Production Error Verbosity (FIXED)
- **Location**: `errors.go:1-145`, `middleware.go:690-695`
- **Issue**: Verbose error messages leak implementation details
- **Impact**: Information disclosure through error messages
- **Fix**: Created comprehensive error sanitization system
- **Status**: ✅ Fixed - environment-aware error sanitization
- **Gate evidence**: `TestSanitizeErrorModes`

#### 8. CORS Policy (FIXED)
- **Location**: `middleware_advanced.go:471-497`
- **Issue**: Default CORS allows "*" (all origins)
- **Impact**: Overly permissive CORS in production
- **Fix**: Secure defaults (no origins in prod, localhost in dev)
- **Status**: ✅ Fixed - strict defaults, environment-aware
- **Gate evidence**: `TestNewCORSMiddlewareDefaults`

### Security mechanisms implemented in this library

Each item below maps to a concrete location in the source tree. Claims that
could not be grounded in code were removed (see the deferral list that
follows).

#### Cryptography
- Constant-time secret comparison — `subtle.ConstantTimeCompare`
  (`auth.go:215`, `auth_security.go:449`)
- AES-GCM for token/payload encryption — `aes.NewCipher` + `cipher.NewGCM`
  (`security.go:547`, `security.go:644`)
- HMAC-SHA256 for signatures — `hmac.New(sha256.New, ...)`
  (`auth_security.go:440`)
- Argon2id / PBKDF2 key derivation — `argon2.IDKey`
  (`auth_security.go:107`); covered by `TestDeriveKeyMethods` and
  `security_gate_test.go`
- crypto/rand token generation with no predictable fallback — covered by
  `TestGenerateRandomString_ReadError` (see Critical Vulnerability #1)

#### Authentication & authorization
- OAuth2 provider with client authentication — `auth.go`, `auth_security.go`
- Token validation hardened against revocation races — covered by
  `TestConcurrentTokenOperations` and `go test -race ./...` (#3)
- Context-value sanitization on client-info extraction — covered by
  `TestSecureOAuthProvider_ExtractClientInfoSanitizesContextValues` (#4)

#### Request handling
- Per-endpoint rate limiting — `ratelimit.go`, `middleware.go`; covered by
  `TestRateLimitMiddleware_PerEndpointLimiting` (#5)
- Environment-aware error sanitization — `errors.go`; covered by
  `TestSanitizeErrorModes` (#7)
- Secure CORS defaults — `middleware_advanced.go`; covered by
  `TestNewCORSMiddlewareDefaults` (#8)
- JSON schema validation — `security.go` (`JSONSchemaValidator`)

### Explicitly not provided (do not claim these)

These are out of scope for the library or not yet implemented. They are
listed so the security posture is honest and so the v1 gate (B5) can map
every claim to a named check, a deferral, or a removed claim.

- **TLS enforcement / `MinVersion`.** The library does not configure TLS;
  transport TLS is the embedding application's responsibility. Removed the
  prior "TLS 1.2+ enforcement" claim — there is no `tls.Config` with
  `MinVersion` in this package.
- **SQL injection, XXE, path-traversal, and command-injection "prevention."**
  Removed. This is a protocol library with no SQL, XML, filesystem, or shell
  surface of its own; those prior claims were not backed by any code.
- **MFA, HSM integration, IP allowlisting, DDoS protection.** Not
  implemented; deferred beyond v1.
- **Formal compliance (SOC 2 / GDPR / HIPAA).** No certification or audit is
  asserted by this library. Any prior compliance checkmarks were aspirational
  and have been removed.

## Security Best Practices

### For Developers

1. **Never commit secrets** - Use environment variables
2. **Validate all inputs** - Use the validation middleware
3. **Sanitize logs** - Don't log sensitive data
4. **Update dependencies** - Run `govulncheck ./...` regularly

### For Operators

1. **Use TLS everywhere** - Never run production without TLS
2. **Rotate secrets** - Change keys every 90 days
3. **Monitor logs** - Watch for suspicious patterns
4. **Limit access** - Use principle of least privilege
5. **Regular audits** - Schedule quarterly security reviews

## Security Configuration

### Recommended Production Settings

```go
// Secure middleware configuration
config := &ServerMiddlewareConfig{
    GlobalConfig: &MiddlewareConfig{
        Enabled: true,
        RateLimit: &RateLimitConfig{
            RequestsPerSecond: 100,
            BurstSize:         10,
            PerClient:         true,  // Important!
        },
        Authentication: &AuthConfig{
            Required:     true,
            TokenExpiry:  15 * time.Minute,
            RefreshToken: true,
        },
        Logging: &LoggingConfig{
            Level:           slog.LevelWarn,  // Not Debug!
            SanitizeSensitive: true,
        },
        CORS: &CORSConfig{
            AllowOrigins: []string{"https://yourdomain.com"},  // Not "*"!
            Credentials:  true,
        },
    },
}
```

### Environment Variables

```bash
# Required for production
MCP_TLS_CERT=/path/to/cert.pem
MCP_TLS_KEY=/path/to/key.pem
MCP_AUTH_SECRET=$(openssl rand -hex 32)
MCP_ENCRYPTION_KEY=$(openssl rand -hex 32)

# Recommended
MCP_LOG_LEVEL=warn
MCP_RATE_LIMIT=100
MCP_MAX_REQUEST_SIZE=1048576
MCP_SESSION_TIMEOUT=900
```

## Security Tools

### Static Analysis
```bash
# Run security checks
go install github.com/securego/gosec/v2/cmd/gosec@latest
gosec ./...

# Check for vulnerabilities
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

### Dynamic Analysis
```bash
# Fuzzing (protocol marshaling targets live in modelcontextprotocol/)
go test -fuzz=FuzzCallToolResultUnmarshal -fuzztime=10s ./modelcontextprotocol/

# Race detection
go test -race ./...
```

### Dependency Scanning
```bash
# Check dependencies
go list -m -json all | nancy sleuth

# Update vulnerable dependencies
go get -u ./...
go mod tidy
```

## Security Contacts

- **Private Reports**: https://github.com/tmc/mcp/security/advisories/new
- **Security Updates**: https://github.com/tmc/mcp/security

## Acknowledgments

We thank the following researchers for responsible disclosure:

- [Pending - be the first!]

---

**Remember**: Security is everyone's responsibility. If you see something, say something!
