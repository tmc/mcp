# MCP PostgreSQL Server

A Model Context Protocol (MCP) server that provides read-only access to PostgreSQL databases with schema inspection capabilities.

## Features

- **Read-Only Database Access**: Safely query PostgreSQL databases without modification risks
- **Schema Inspection**: List tables, describe table structures, and explore database schemas
- **Connection String Support**: Standard PostgreSQL connection strings
- **Query Execution**: Execute SELECT queries with configurable limits
- **Security**: Read-only mode prevents accidental writes

## Tools

### `list_tables`
List all tables in a PostgreSQL database schema.

**Parameters:**
- `schema` (optional): Schema name (default: "public")

### `describe_table`  
Describe the structure of a PostgreSQL table.

**Parameters:**
- `table` (required): Table name
- `schema` (optional): Schema name (default: "public")

### `execute_query`
Execute a read-only SQL query against the PostgreSQL database.

**Parameters:**
- `query` (required): SQL query to execute
- `limit` (optional): Maximum number of rows to return (default: 100)

### `list_schemas`
List all schemas in the PostgreSQL database.

**Parameters:** None

## Usage

```bash
# Basic usage with connection string
./mcp-postgresql-server "host=localhost user=username dbname=mydb sslmode=disable"

# Read-only mode (recommended)
./mcp-postgresql-server "host=localhost user=username dbname=mydb sslmode=disable" --read-only
```

## Connection String Format

Standard PostgreSQL connection string format:
```
host=localhost port=5432 user=username password=password dbname=database sslmode=disable
```

Or as URL:
```
postgres://username:password@localhost:5432/database?sslmode=disable
```

## Example Queries

### List tables in public schema
```json
{
  "name": "list_tables"
}
```

### Describe a specific table
```json
{
  "name": "describe_table",
  "arguments": {
    "table": "users",
    "schema": "public"
  }
}
```

### Execute a SELECT query
```json
{
  "name": "execute_query", 
  "arguments": {
    "query": "SELECT id, name, email FROM users WHERE active = true",
    "limit": 50
  }
}
```

## Security Considerations

- Use read-only database users when possible
- Enable `--read-only` flag to prevent write operations
- Validate connection strings to prevent SQL injection
- Consider network security and SSL connections
- Limit result set sizes to prevent memory issues

## Dependencies

- PostgreSQL driver: `github.com/lib/pq`
- MCP Go library: `github.com/tmc/mcp`