# MCP SQLite Server

A Model Context Protocol (MCP) server that provides access to SQLite databases.

## Features

- Execute SQL queries and return JSON results
- Execute DDL and DML statements
- Get database schema information
- Support for transactions
- Safe error handling

## Tools

### execute_query
Execute a SELECT query and return results as JSON.

**Parameters:**
- `query` (string): SQL SELECT query to execute

### execute_statement
Execute INSERT, UPDATE, DELETE, or DDL statements.

**Parameters:**
- `statement` (string): SQL statement to execute

### get_schema
Get the database schema including tables, views, and indexes.

**Parameters:** None

## Configuration

Set the `SQLITE_DB_PATH` environment variable to specify the database file path. Defaults to `./test.db`.

## Usage

```bash
# Start the server with default database
go run .

# Start with custom database
SQLITE_DB_PATH=/path/to/database.db go run .
```

## Examples

### Create a table
```json
{
  "tool": "execute_statement",
  "arguments": {
    "statement": "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT)"
  }
}
```

### Insert data
```json
{
  "tool": "execute_statement", 
  "arguments": {
    "statement": "INSERT INTO users (name, email) VALUES ('John Doe', 'john@example.com')"
  }
}
```

### Query data
```json
{
  "tool": "execute_query",
  "arguments": {
    "query": "SELECT * FROM users WHERE name LIKE '%John%'"
  }
}
```

### Get schema
```json
{
  "tool": "get_schema",
  "arguments": {}
}
```