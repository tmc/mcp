# Local Authentication for mcpd

This document describes the local/offline authentication functionality for mcpd that requires **no third-party OAuth providers**.

## Overview

Local authentication provides a completely offline, self-contained authentication system for mcpd. This is perfect for:

- 🏠 **Local development environments** 
- 🔒 **Air-gapped/isolated networks**
- 🛡️ **Internal corporate environments**
- ⚡ **Quick testing and demos**
- 🌐 **Environments without internet access**

## Key Features

- ✅ **Zero external dependencies** - No Google, GitHub, or any third-party service needed
- ✅ **Multiple user management options** - Command-line, files, environment variables
- ✅ **Secure password hashing** - Uses bcrypt for password storage
- ✅ **Session management** - Secure HTTP-only cookies with expiration
- ✅ **Built-in web interface** - Clean login/logout pages
- ✅ **Persistent storage** - Optional JSON file for user persistence
- ✅ **Auto-admin creation** - Creates default admin user if none exist

## Quick Start

### 1. Basic Local Authentication

```bash
# Start with command-line users
./mcpd-local \
  -enable-oauth \
  -oauth-provider local \
  -local-auth-users 'admin:password,user:secret' \
  -http :8081 \
  -- your-mcp-server

# Access: http://localhost:8081/login
# Users: admin/password, user/secret
```

### 2. Environment Variable Users

```bash
# Set users via environment
export MCPD_USERS='dev:dev123,ops:ops456,admin:admin'

./mcpd-local \
  -enable-oauth \
  -oauth-provider local \
  -http :8081 \
  -- your-mcp-server
```

### 3. File-Based Users

```bash
# Create users file (users.txt)
echo "admin:securepassword" > users.txt
echo "developer:devpass" >> users.txt
echo "guest:welcome" >> users.txt

# Start with file-based auth
./mcpd-local \
  -enable-oauth \
  -oauth-provider local \
  -local-auth-file users.txt \
  -http :8081 \
  -- your-mcp-server
```

### 4. Persistent JSON Store

```bash
# Users will be saved to/loaded from JSON file
./mcpd-local \
  -enable-oauth \
  -oauth-provider local \
  -local-auth-users 'admin:admin' \
  -local-auth-persist users.json \
  -http :8081 \
  -- your-mcp-server
```

## Configuration Options

| Flag | Description | Example |
|------|-------------|---------|
| `-oauth-provider local` | Enable local authentication mode | Required |
| `-local-auth-users` | Command-line users (user:pass,user2:pass2) | `'admin:secret,dev:test'` |
| `-local-auth-file` | Path to users file (username:password per line) | `users.txt` |
| `-local-auth-persist` | Path for JSON user store persistence | `users.json` |
| `MCPD_USERS` | Environment variable for users | `'user1:pass1,user2:pass2'` |

## User Management Formats

### Command Line Format
```bash
-local-auth-users 'username1:password1,username2:password2'
```

### File Format (Plain Text)
```
# comments start with #
admin:securepassword123
developer:devpass456
guest:welcome
```

### Environment Variable Format
```bash
export MCPD_USERS='admin:admin,dev:devpass,ops:opspass'
```

### JSON Persistence Format (Auto-generated)
```json
[
  {
    "username": "admin",
    "password_hash": "$2a$10$...",
    "created_at": "2024-01-01T12:00:00Z",
    "last_login": "2024-01-01T13:00:00Z"
  }
]
```

## Security Features

### Password Security
- 🔐 **Bcrypt hashing** - Passwords are never stored in plaintext
- 🧂 **Salt included** - Each password gets a unique salt
- ⚡ **Adaptive cost** - Configurable work factor for future-proofing

### Session Security
- 🍪 **HTTP-only cookies** - Cannot be accessed via JavaScript
- ⏰ **Time-based expiration** - 24-hour session lifetime
- 🔒 **Secure flag** - Cookies marked secure when using HTTPS
- 🎲 **Random session IDs** - Cryptographically secure random generation

### Access Control
- 👤 **User-based authentication** - Each user has individual credentials
- 🚪 **Automatic redirects** - Unauthenticated users redirected to login
- 🔄 **Clean logout** - Sessions properly invalidated on logout

## Web Interface

### Login Page
- Clean, professional HTML interface
- Username and password fields
- Error message display
- Responsive design

### Available Endpoints
| Endpoint | Purpose | Auth Required |
|----------|---------|---------------|
| `/login` | Login page and form handler | No |
| `/logout` | Logout and session cleanup | No |
| `/stream` | MCP HTTP streaming | Yes |
| `/sse` | MCP Server-Sent Events | Yes |
| `/ws` | MCP WebSocket (experimental) | Yes |

## Use Cases

### Development Environment
```bash
# Quick local development with simple users
./mcpd-local -enable-oauth -oauth-provider local \
  -local-auth-users 'dev:dev' -http :8081 -- my-dev-server
```

### Team Development
```bash
# Team users in a file
echo "alice:alice123" > team.txt
echo "bob:bob456" >> team.txt
echo "charlie:charlie789" >> team.txt

./mcpd-local -enable-oauth -oauth-provider local \
  -local-auth-file team.txt -http :8081 -- team-server
```

### Production Internal
```bash
# Production with environment variables and persistence
export MCPD_USERS='admin:$(openssl rand -base64 32),ops:$(openssl rand -base64 32)'

./mcpd-local -enable-oauth -oauth-provider local \
  -local-auth-persist /etc/mcpd/users.json \
  -http :8081 -- production-server
```

### Air-Gapped Environment
```bash
# Completely offline deployment
./mcpd-local -enable-oauth -oauth-provider local \
  -local-auth-users 'operator:$(cat /dev/urandom | base64 | head -c 32)' \
  -local-auth-persist /var/lib/mcpd/users.json \
  -http :443 -- secure-server
```

## Default Behavior

If no users are configured through any method, mcpd will automatically:

1. Create a default admin user: `admin/admin`
2. Log a warning about the default credentials
3. Recommend changing the password for security

**⚠️ Warning**: Always change default credentials in production!

## Comparison: Local vs OAuth

| Feature | Local Auth | OAuth (Google/GitHub) |
|---------|------------|----------------------|
| **Setup Complexity** | ⭐ Simple | ⭐⭐⭐ Complex |
| **External Dependencies** | ❌ None | ✅ Required |
| **Internet Required** | ❌ No | ✅ Yes |
| **Third-party Service** | ❌ No | ✅ Yes |
| **Privacy** | ⭐⭐⭐ Excellent | ⭐⭐ Good |
| **Control** | ⭐⭐⭐ Full | ⭐ Limited |
| **Air-gap Compatible** | ✅ Yes | ❌ No |
| **Enterprise Ready** | ✅ Yes | ⭐⭐ Depends |

## Migration Examples

### From OAuth to Local
```bash
# Before (OAuth)
./mcpd -enable-oauth -oauth-provider google \
  -oauth-client-id "..." -oauth-secret "..." \
  -authorized-users "user@company.com" \
  -http :8081 -- server

# After (Local)
./mcpd-local -enable-oauth -oauth-provider local \
  -local-auth-users 'user:password' \
  -http :8081 -- server
```

### Adding Local to Existing Setup
```bash
# Can run both simultaneously on different ports
./mcpd -enable-oauth -oauth-provider google ... -http :8081 -- server &
./mcpd-local -enable-oauth -oauth-provider local \
  -local-auth-users 'admin:backup' -http :8082 -- server &
```

## Troubleshooting

### Common Issues

**Q: Login fails with correct credentials**  
A: Check logs for password hashing errors, ensure bcrypt is working

**Q: Session expires too quickly**  
A: Session lifetime is 24 hours, check system clock

**Q: Default admin user not created**  
A: Ensure no other user sources are configured

**Q: Users file not loading**  
A: Check file permissions and format (username:password per line)

### Debug Mode
```bash
# Enable verbose logging
./mcpd-local -v -enable-oauth -oauth-provider local ... -- server
```

### Testing Authentication
```bash
# Test login with curl
curl -c cookies.txt -d "username=admin&password=admin" \
  http://localhost:8081/login

# Test protected endpoint
curl -b cookies.txt http://localhost:8081/stream
```

## Best Practices

### Security
1. **Use strong passwords** - Generate random passwords for production
2. **Change defaults** - Never use admin/admin in production
3. **Use HTTPS** - Enable TLS for production deployments
4. **Rotate credentials** - Regularly update user passwords
5. **Monitor access** - Check logs for authentication attempts

### Management
1. **Environment variables** - Use for deployment automation
2. **File-based users** - Good for team environments
3. **JSON persistence** - Enables user management features
4. **Backup credentials** - Keep secure backup of user files

### Deployment
1. **Test locally first** - Verify authentication before production
2. **Use reverse proxy** - nginx/Apache for HTTPS termination
3. **Monitor logs** - Watch for authentication errors
4. **Health checks** - Verify login endpoint availability

## Implementation Notes

The local authentication system is built with:

- **Go's bcrypt package** - Industry-standard password hashing
- **Secure random generation** - Session IDs use crypto/rand
- **HTTP-only cookies** - XSS protection
- **Clean separation** - Local auth is separate from OAuth code
- **Zero external deps** - No third-party authentication libraries

This makes it ideal for environments where you need complete control over authentication without external dependencies.