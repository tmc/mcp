#!/bin/bash
# Demo script for mcpd with OAuth authentication

echo "🔐 mcpd OAuth Demo"
echo "=================="

# Example 1: Basic OAuth with Google
echo "📋 Example 1: Google OAuth with authorized users"
echo "./mcpd -http :8080 -enable-sse -enable-oauth \\"
echo "  -oauth-provider google \\"
echo "  -oauth-client-id YOUR_GOOGLE_CLIENT_ID \\"
echo "  -oauth-secret YOUR_GOOGLE_CLIENT_SECRET \\"
echo "  -authorized-users user1@gmail.com,user2@company.com \\"
echo "  -- ./your-mcp-server"
echo

# Example 2: GitHub OAuth
echo "📋 Example 2: GitHub OAuth"
echo "./mcpd -http :8080 -enable-sse -enable-oauth \\"
echo "  -oauth-provider github \\"
echo "  -oauth-client-id YOUR_GITHUB_CLIENT_ID \\"
echo "  -oauth-secret YOUR_GITHUB_CLIENT_SECRET \\"
echo "  -authorized-users developer@company.com \\"
echo "  -- ./your-mcp-server"
echo

# Example 3: Environment variables
echo "📋 Example 3: Using environment variables"
echo "export OAUTH_CLIENT_ID=your_client_id"
echo "export OAUTH_CLIENT_SECRET=your_client_secret"
echo "export AUTHORIZED_USERS=user1@example.com,user2@example.com"
echo "./mcpd -http :8080 -enable-sse -enable-oauth \\"
echo "  -oauth-provider google \\"
echo "  -oauth-client-id \$OAUTH_CLIENT_ID \\"
echo "  -oauth-secret \$OAUTH_CLIENT_SECRET \\"
echo "  -authorized-users \$AUTHORIZED_USERS \\"
echo "  -- ./your-mcp-server"
echo

# OAuth setup instructions
echo "🔧 OAuth Setup Instructions:"
echo "============================="
echo
echo "For Google OAuth:"
echo "1. Go to Google Cloud Console"
echo "2. Create/select a project"
echo "3. Enable Google+ API"
echo "4. Create OAuth 2.0 credentials"
echo "5. Add authorized redirect URI: http://localhost:8080/auth/callback"
echo "6. Copy Client ID and Client Secret"
echo
echo "For GitHub OAuth:"
echo "1. Go to GitHub Settings > Developer settings > OAuth Apps"
echo "2. Create a new OAuth App"
echo "3. Set Authorization callback URL: http://localhost:8080/auth/callback"
echo "4. Copy Client ID and Client Secret"
echo
echo "📱 Usage Flow:"
echo "1. Start mcpd with OAuth enabled"
echo "2. Visit http://localhost:8080 in browser"
echo "3. You'll be redirected to OAuth provider login"
echo "4. After authorization, you can access MCP endpoints"
echo "5. Available endpoints:"
echo "   - /sse (Server-Sent Events)"
echo "   - /stream (HTTP streaming)"
echo "   - /health (health check - no auth required)"
echo "   - /auth/login (manual login)"
echo "   - /auth/logout (logout)"
echo
echo "🔒 Security Features:"
echo "- Only authorized users can access MCP endpoints"
echo "- Session-based authentication with secure cookies"
echo "- CSRF protection with state parameter"
echo "- Support for Google, GitHub, and custom OAuth providers"
echo
echo "Deploy to Cloud Run with OAuth:"
echo "=============================="
echo "gcloud run deploy mcp-server \\"
echo "  --source . \\"
echo "  --set-env-vars=\"OAUTH_CLIENT_ID=\$CLIENT_ID,OAUTH_CLIENT_SECRET=\$SECRET\" \\"
echo "  --allow-unauthenticated \\"
echo "  --port 8080"