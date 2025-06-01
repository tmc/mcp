# OAuth Authentication for mcpd

This document describes the OAuth authentication functionality added to mcpd (MCP Server Daemon).

## Overview

mcpd now supports OAuth 2.0 authentication to protect MCP streaming endpoints. When enabled, users must authenticate through Google, GitHub, or a custom OAuth provider before accessing MCP services.

## Features

- ✅ Google OAuth 2.0 integration
- ✅ GitHub OAuth 2.0 integration  
- ✅ Custom OAuth provider support
- ✅ Email-based authorization control
- ✅ Secure session management
- ✅ Configurable callback URLs
- ✅ Multiple authorized users support

## Quick Start

1. **Build OAuth-enabled mcpd:**
   ```bash
   cd /Volumes/tmc/go/src/github.com/tmc/mcp/exp/cmd/mcpd
   go build -o mcpd-oauth .
   ```

2. **Set up OAuth provider (Google example):**
   - Go to [Google Cloud Console](https://console.cloud.google.com)
   - Create/select project and enable Google+ API
   - Create OAuth 2.0 Client ID credentials
   - Set redirect URI: `http://localhost:8081/auth/callback`

3. **Run with OAuth protection:**
   ```bash
   ./mcpd-oauth \
     -enable-oauth \
     -oauth-client-id "your-google-client-id" \
     -oauth-secret "your-google-secret" \
     -authorized-users "user@example.com" \
     -http :8081 \
     -- your-mcp-server
   ```

4. **Access protected endpoints:**
   - Navigate to `http://localhost:8081/login` to authenticate
   - After OAuth flow, access `http://localhost:8081/stream` or `/sse`

## Command Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `-enable-oauth` | Enable OAuth authentication | `false` |
| `-oauth-client-id` | OAuth Client ID from provider | `""` |
| `-oauth-secret` | OAuth Client Secret from provider | `""` |
| `-oauth-provider` | Provider: `google`, `github`, `custom` | `"google"` |
| `-oauth-callback` | OAuth callback path | `"/auth/callback"` |
| `-authorized-users` | Comma-separated list of authorized emails | `""` |

## OAuth Providers

### Google OAuth Setup

1. Go to [Google Cloud Console](https://console.cloud.google.com)
2. Create a new project or select existing
3. Navigate to APIs & Services → Credentials
4. Click "Create Credentials" → "OAuth 2.0 Client ID"
5. Select "Web application"
6. Add authorized redirect URI: `http://localhost:8081/auth/callback`
7. Copy Client ID and Client Secret

### GitHub OAuth Setup

1. Go to GitHub Settings → Developer settings → OAuth Apps
2. Click "New OAuth App"
3. Fill in application details
4. Set Authorization callback URL: `http://localhost:8081/auth/callback`
5. Copy Client ID and Client Secret

### Custom OAuth Provider

For custom providers, ensure they support OAuth 2.0 standard endpoints:
- Authorization endpoint
- Token endpoint  
- User info endpoint (returning email field)

## Usage Examples

### Basic Google OAuth
```bash
./mcpd-oauth \
  -enable-oauth \
  -oauth-client-id "123456789.apps.googleusercontent.com" \
  -oauth-secret "your-secret" \
  -authorized-users "admin@company.com" \
  -http :8081 \
  -- echo-server
```

### GitHub OAuth with Multiple Users
```bash
./mcpd-oauth \
  -enable-oauth \
  -oauth-provider github \
  -oauth-client-id "your-github-client-id" \
  -oauth-secret "your-github-secret" \
  -authorized-users "dev1@company.com,dev2@company.com,admin@company.com" \
  -http :8081 \
  -- mcp-time-server
```

### Custom Callback Path
```bash
./mcpd-oauth \
  -enable-oauth \
  -oauth-callback "/auth/custom-callback" \
  -oauth-client-id "your-client-id" \
  -oauth-secret "your-secret" \
  -authorized-users "user@example.com" \
  -http :8081 \
  -- your-server
```

## Security Features

- **Session Management**: Secure HTTP-only cookies with CSRF protection
- **Email Authorization**: Only specified email addresses can access services
- **HTTPS Ready**: Configure with reverse proxy for production HTTPS
- **State Validation**: OAuth state parameter prevents CSRF attacks
- **Token Validation**: Proper OAuth token validation and user info retrieval

## OAuth Flow

1. User accesses protected endpoint (e.g., `/stream`)
2. Middleware redirects to `/login` if not authenticated
3. User clicks OAuth provider button
4. Redirected to provider's authorization page
5. After approval, provider redirects to callback URL
6. mcpd exchanges code for access token
7. mcpd fetches user info and validates email
8. If authorized, user session is created
9. User can now access protected MCP endpoints

## Endpoints

| Endpoint | Description | Auth Required |
|----------|-------------|---------------|
| `/login` | OAuth login page | No |
| `/auth/callback` | OAuth callback handler | No |
| `/logout` | Clear session | No |
| `/stream` | MCP HTTP streaming | Yes |
| `/sse` | MCP Server-Sent Events | Yes |
| `/ws` | MCP WebSocket (experimental) | Yes |

## Environment Variables

OAuth credentials can also be provided via environment variables:
```bash
export OAUTH_CLIENT_ID="your-client-id"
export OAUTH_CLIENT_SECRET="your-secret"
export AUTHORIZED_USERS="user1@example.com,user2@example.com"

./mcpd-oauth -enable-oauth -http :8081 -- your-server
```

## Production Deployment

For production use:
1. Use HTTPS with proper TLS certificates
2. Set secure OAuth redirect URIs (https://)
3. Use environment variables for secrets
4. Configure proper session timeouts
5. Consider using a reverse proxy (nginx, Cloudflare, etc.)

## Troubleshooting

### Common Issues

1. **OAuth callback mismatch**: Ensure redirect URI in OAuth app matches `-oauth-callback` flag
2. **Unauthorized user**: Check that user email is in `-authorized-users` list
3. **Provider errors**: Verify client ID and secret are correct
4. **Session issues**: Clear browser cookies and try again

### Debug Mode

Enable verbose logging to troubleshoot:
```bash
./mcpd-oauth -v -enable-oauth ... -- your-server
```

## Implementation Details

The OAuth implementation consists of:

- `transport/oauth.go`: Core OAuth middleware and handlers
- `config/config.go`: OAuth configuration management  
- `transport/streaming.go`: Integration with HTTP server
- `main.go`: Command-line flag definitions

The middleware integrates seamlessly with existing mcpd transports while adding robust authentication without affecting the core MCP protocol handling.