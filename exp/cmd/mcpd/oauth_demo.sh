#!/bin/bash

# OAuth Demo Script for mcpd
# This demonstrates how to use OAuth authentication with mcpd

echo "=== MCP Daemon OAuth Demo ==="
echo

# Example 1: Basic OAuth setup with Google provider
echo "1. Basic OAuth with Google provider:"
echo "./mcpd-oauth -enable-oauth -oauth-client-id YOUR_CLIENT_ID -oauth-secret YOUR_SECRET -authorized-users user@example.com -http :8081 -- echo-server"
echo

# Example 2: OAuth with GitHub provider
echo "2. OAuth with GitHub provider:"
echo "./mcpd-oauth -enable-oauth -oauth-provider github -oauth-client-id YOUR_GITHUB_CLIENT_ID -oauth-secret YOUR_GITHUB_SECRET -authorized-users github-user@example.com -http :8081 -- echo-server"
echo

# Example 3: OAuth with multiple authorized users
echo "3. OAuth with multiple authorized users:"
echo "./mcpd-oauth -enable-oauth -oauth-client-id YOUR_CLIENT_ID -oauth-secret YOUR_SECRET -authorized-users 'user1@example.com,user2@example.com,admin@example.com' -http :8081 -- echo-server"
echo

# Example 4: OAuth with custom callback path
echo "4. OAuth with custom callback path:"
echo "./mcpd-oauth -enable-oauth -oauth-client-id YOUR_CLIENT_ID -oauth-secret YOUR_SECRET -oauth-callback '/custom/callback' -authorized-users user@example.com -http :8081 -- echo-server"
echo

echo "=== OAuth Setup Instructions ==="
echo
echo "Google OAuth Setup:"
echo "1. Go to Google Cloud Console (console.cloud.google.com)"
echo "2. Create a new project or select existing project"
echo "3. Enable Google+ API"
echo "4. Go to Credentials → Create Credentials → OAuth 2.0 Client ID"
echo "5. Set authorized redirect URI: http://localhost:8081/auth/callback"
echo "6. Copy Client ID and Client Secret"
echo
echo "GitHub OAuth Setup:"
echo "1. Go to GitHub Settings → Developer settings → OAuth Apps"
echo "2. Click 'New OAuth App'"
echo "3. Set Authorization callback URL: http://localhost:8081/auth/callback"
echo "4. Copy Client ID and Client Secret"
echo
echo "=== OAuth Flow ==="
echo "1. Start mcpd with OAuth enabled"
echo "2. Access http://localhost:8081/login to initiate OAuth"
echo "3. Complete OAuth flow with provider"
echo "4. Access protected MCP endpoints at http://localhost:8081/stream or /sse"
echo