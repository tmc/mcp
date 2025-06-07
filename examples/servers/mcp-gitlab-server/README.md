# MCP GitLab Server

A Model Context Protocol (MCP) server for GitLab integration. This server provides tools to interact with GitLab projects, issues, and repositories.

## Features

- **Search Repositories**: Find GitLab repositories by query
- **Get Project Details**: Retrieve detailed information about GitLab projects
- **List Issues**: View project issues with filtering by state
- **Create Issues**: Create new issues in GitLab projects

## Tools

### search_repositories
Search for GitLab repositories.

**Parameters:**
- `query` (string, required): Search query for repositories
- `limit` (integer, optional): Maximum number of results (default: 10)

### get_project
Get detailed information about a GitLab project.

**Parameters:**
- `project_id` (string, required): GitLab project ID or path

### list_issues
List issues from a GitLab project.

**Parameters:**
- `project_id` (string, required): GitLab project ID or path
- `state` (string, optional): Issue state - "opened", "closed", "all" (default: "opened")
- `limit` (integer, optional): Maximum number of issues (default: 10)

### create_issue
Create a new issue in a GitLab project.

**Parameters:**
- `project_id` (string, required): GitLab project ID or path
- `title` (string, required): Issue title
- `description` (string, optional): Issue description
- `labels` (array of strings, optional): Issue labels

## Configuration

Set the following environment variables:
- `GITLAB_TOKEN`: GitLab personal access token (optional for mock mode)
- `GITLAB_URL`: GitLab instance URL (optional, defaults to gitlab.com)

## Usage

```bash
go run main.go
```

## Example Usage

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "search_repositories",
    "arguments": {
      "query": "machine-learning",
      "limit": 5
    }
  }
}
```

## Note

This implementation includes mock data for demonstration purposes. For production use, integrate with the actual GitLab API using the GitLab Go client library.