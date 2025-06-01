# MCP Git Server

A Model Context Protocol (MCP) server that provides Git repository management capabilities.

## Features

- **Repository Management**: Secure access to specified Git repositories
- **Status & History**: Check repository status and view commit logs
- **Branch Operations**: List, create, and switch between branches
- **Diff Viewing**: Show changes between commits or working directory
- **Staging & Commits**: Add files to staging area and create commits
- **Security**: Only operates on explicitly allowed repositories

## Available Tools

### git_status
Get the status of a git repository
- `repository` (string): Path to the git repository

### git_log
Get the commit log of a git repository
- `repository` (string): Path to the git repository
- `max_count` (number, optional): Maximum number of commits to show (default: 10)

### git_branches
List branches in a git repository
- `repository` (string): Path to the git repository

### git_diff
Show differences in a git repository
- `repository` (string): Path to the git repository
- `ref1` (string, optional): First reference for diff
- `ref2` (string, optional): Second reference for diff

### git_add
Add files to git staging area
- `repository` (string): Path to the git repository
- `files` (array): List of file paths to add

### git_commit
Create a git commit
- `repository` (string): Path to the git repository
- `message` (string): Commit message

### git_create_branch
Create a new git branch
- `repository` (string): Path to the git repository
- `branch` (string): Name of the new branch

### git_switch_branch
Switch to a different git branch
- `repository` (string): Path to the git repository
- `branch` (string): Name of the branch to switch to

## Usage

```bash
# Allow access to a single repository
./mcp-git-server /path/to/repo

# Allow access to multiple repositories
./mcp-git-server /path/to/repo1 /path/to/repo2 ~/my-project

# Use with home directory expansion
./mcp-git-server ~/projects/my-repo
```

## Security Features

- Only operates on explicitly allowed repositories specified at startup
- Path validation prevents directory traversal attacks
- Symlink resolution ensures access controls aren't bypassed
- All repository paths are validated before any Git operations

## Installation

```bash
go build -o mcp-git-server main.go
```

## Integration

This server can be integrated with MCP clients like Claude Desktop by adding it to your configuration:

```json
{
  "mcpServers": {
    "git": {
      "command": "/path/to/mcp-git-server",
      "args": ["/path/to/your/repo"]
    }
  }
}
```