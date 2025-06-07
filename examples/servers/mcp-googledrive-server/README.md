# MCP Google Drive Server

A Model Context Protocol (MCP) server that provides read-only access to Google Drive with file browsing and search capabilities.

## Features

- **File Browsing**: List files and folders in Google Drive
- **File Information**: Get detailed metadata about files
- **Search Capabilities**: Search files by name and content
- **Folder Navigation**: Browse folder structure
- **OAuth2 Authentication**: Secure authentication with Google Drive API

## Tools

### `list_files`
List files in Google Drive with optional filtering.

**Parameters:**
- `query` (optional): Google Drive API query filter (e.g., "mimeType='image/jpeg'")
- `limit` (optional): Maximum number of files to return (default: 20)

### `get_file_info`
Get detailed information about a specific Google Drive file.

**Parameters:**
- `file_id` (required): Google Drive file ID

### `search_files`
Search for files in Google Drive by name or content.

**Parameters:**
- `query` (required): Search query string
- `limit` (optional): Maximum number of results to return (default: 20)

### `list_folders`
List folders in Google Drive.

**Parameters:**
- `parent_id` (optional): Parent folder ID (if omitted, lists root folders)

## Setup

### 1. Google Cloud Console Setup

1. Go to the [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Enable the Google Drive API:
   - Go to "APIs & Services" > "Library"
   - Search for "Google Drive API" and enable it
4. Create credentials:
   - Go to "APIs & Services" > "Credentials"
   - Click "Create Credentials" > "OAuth 2.0 Client IDs"
   - Choose "Desktop application"
   - Download the credentials JSON file

### 2. OAuth2 Token Generation

Create a simple script to generate your OAuth2 token:

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"

    "golang.org/x/oauth2"
    "golang.org/x/oauth2/google"
    "google.golang.org/api/drive/v3"
)

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
    authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
    fmt.Printf("Go to the following link in your browser: \n%v\n", authURL)
    fmt.Print("Enter the authorization code: ")
    
    var authCode string
    if _, err := fmt.Scan(&authCode); err != nil {
        log.Fatalf("Unable to read authorization code: %v", err)
    }

    tok, err := config.Exchange(context.TODO(), authCode)
    if err != nil {
        log.Fatalf("Unable to retrieve token from web: %v", err)
    }
    return tok
}

func main() {
    b, err := os.ReadFile("credentials.json")
    if err != nil {
        log.Fatalf("Unable to read client secret file: %v", err)
    }

    config, err := google.ConfigFromJSON(b, drive.DriveReadonlyScope)
    if err != nil {
        log.Fatalf("Unable to parse client secret file to config: %v", err)
    }

    tok := getTokenFromWeb(config)
    
    tokFile, err := os.Create("token.json")
    if err != nil {
        log.Fatalf("Unable to cache oauth token: %v", err)
    }
    defer tokFile.Close()
    json.NewEncoder(tokFile).Encode(tok)
    fmt.Println("Token saved to token.json")
}
```

### 3. Environment Variables

Set the required environment variables:

```bash
export GOOGLE_CREDENTIALS=path/to/credentials.json
export GOOGLE_TOKEN=path/to/token.json
```

## Usage

```bash
# Set environment variables
export GOOGLE_CREDENTIALS=/path/to/credentials.json
export GOOGLE_TOKEN=/path/to/token.json

# Run the server
./mcp-googledrive-server
```

## Example Queries

### List recent files
```json
{
  "name": "list_files",
  "arguments": {
    "limit": 10
  }
}
```

### Search for documents
```json
{
  "name": "search_files",
  "arguments": {
    "query": "project proposal",
    "limit": 15
  }
}
```

### Get file information
```json
{
  "name": "get_file_info",
  "arguments": {
    "file_id": "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms"
  }
}
```

### List folders
```json
{
  "name": "list_folders",
  "arguments": {
    "parent_id": "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms"
  }
}
```

### Advanced file filtering
```json
{
  "name": "list_files",
  "arguments": {
    "query": "mimeType='application/pdf' and modifiedTime > '2023-01-01T00:00:00'",
    "limit": 20
  }
}
```

## Google Drive API Query Syntax

The `list_files` tool supports Google Drive API query syntax:

- `name contains 'hello'` - Files containing "hello" in name
- `mimeType='image/jpeg'` - JPEG images only
- `modifiedTime > '2023-01-01T00:00:00'` - Files modified after date
- `parents in 'folder_id'` - Files in specific folder
- `fullText contains 'keyword'` - Files containing keyword in content

## Security Considerations

- Store credentials and tokens securely
- Use read-only scope (`drive.readonly`) for safety
- Consider using service accounts for production deployment
- Be aware of API quotas and rate limits
- Validate file IDs to prevent unauthorized access

## API Limits

Google Drive API has the following limits:
- **Queries per day**: 1,000,000,000
- **Queries per 100 seconds per user**: 1,000
- **Queries per 100 seconds**: 10,000

## Dependencies

- Google Drive API: `google.golang.org/api/drive/v3`
- OAuth2: `golang.org/x/oauth2`
- MCP Go library: `github.com/tmc/mcp`