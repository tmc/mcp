# MCP GitHub Server

A Model Context Protocol (MCP) server that provides integration with GitHub's API for repository management, file operations, and issue tracking.

## Features

- **Repository Management**: Get repository information, list repositories
- **File Operations**: Read file contents from repositories  
- **Branch Management**: List branches and their commit information
- **Issue Management**: Create and list issues
- **GitHub API Integration**: Full OAuth2 authentication with GitHub

## Tools

### `get_repository`
Get detailed information about a GitHub repository.

**Parameters:**
- `owner` (required): Repository owner (username or organization)
- `repo` (required): Repository name

### `list_repositories`
List GitHub repositories for a user or organization.

**Parameters:**
- `owner` (optional): Repository owner (if omitted, lists authenticated user's repos)
- `limit` (optional): Maximum number of repositories to return (default: 30)

### `get_file_content`
Get the content of a file from a GitHub repository.

**Parameters:**
- `owner` (required): Repository owner
- `repo` (required): Repository name
- `path` (required): File path within the repository
- `ref` (optional): Branch, tag, or commit SHA (default: default branch)

### `list_branches`
List branches in a GitHub repository.

**Parameters:**
- `owner` (required): Repository owner
- `repo` (required): Repository name

### `create_issue`
Create a new GitHub issue.

**Parameters:**
- `owner` (required): Repository owner
- `repo` (required): Repository name
- `title` (required): Issue title
- `body` (optional): Issue description

### `list_issues`
List issues in a GitHub repository.

**Parameters:**
- `owner` (required): Repository owner
- `repo` (required): Repository name
- `state` (optional): Issue state ("open", "closed", "all") (default: "open")
- `limit` (optional): Maximum number of issues to return (default: 30)

## Usage

### Prerequisites

1. Create a GitHub Personal Access Token:
   - Go to GitHub Settings > Developer settings > Personal access tokens
   - Generate a new token with appropriate permissions:
     - `repo` - Full control of private repositories (for private repos)
     - `public_repo` - Access public repositories (for public repos only)
     - `issues` - Read and write access to issues

2. Set the environment variable:
   ```bash
   export GITHUB_TOKEN=your_github_token_here
   ```

### Running the Server

```bash
# Set your GitHub token
export GITHUB_TOKEN=ghp_your_token_here

# Run the server
./mcp-github-server
```

## Example Queries

### Get repository information
```json
{
  "name": "get_repository",
  "arguments": {
    "owner": "microsoft",
    "repo": "vscode"
  }
}
```

### List user's repositories
```json
{
  "name": "list_repositories",
  "arguments": {
    "owner": "octocat",
    "limit": 10
  }
}
```

### Get file content
```json
{
  "name": "get_file_content",
  "arguments": {
    "owner": "microsoft",
    "repo": "vscode", 
    "path": "README.md",
    "ref": "main"
  }
}
```

### Create an issue
```json
{
  "name": "create_issue",
  "arguments": {
    "owner": "myorg",
    "repo": "myrepo",
    "title": "Bug: Application crashes on startup",
    "body": "Steps to reproduce:\n1. Start the application\n2. Click on File menu\n3. Application crashes"
  }
}
```

### List open issues
```json
{
  "name": "list_issues",
  "arguments": {
    "owner": "microsoft",
    "repo": "vscode",
    "state": "open",
    "limit": 20
  }
}
```

## Security Considerations

- Store GitHub tokens securely using environment variables
- Use tokens with minimal required permissions
- Consider using GitHub Apps for organization-wide access
- Be aware of API rate limits (5000 requests/hour for authenticated users)
- Validate repository access permissions before operations

## API Rate Limits

GitHub API has rate limits:
- **Authenticated users**: 5,000 requests per hour
- **Unauthenticated**: 60 requests per hour

The server will return appropriate error messages when rate limits are exceeded.

## Dependencies

- GitHub API client: `github.com/google/go-github/v57`
- OAuth2: `golang.org/x/oauth2`
- MCP Go library: `github.com/tmc/mcp`