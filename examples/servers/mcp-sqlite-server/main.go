package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/modelcontextprotocol"
	_ "modernc.org/sqlite"
)

type SQLiteServer struct {
	db *sql.DB
}

func NewSQLiteServer(dbPath string) (*SQLiteServer, error) {
	// Ensure directory exists
	if dir := filepath.Dir(dbPath); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &SQLiteServer{db: db}, nil
}

func (s *SQLiteServer) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *SQLiteServer) handleExecuteQuery(args map[string]interface{}) (*modelcontextprotocol.CallToolResult, error) {
	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query parameter is required and must be a string")
	}

	rows, err := s.db.Query(query)
	if err != nil {
		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				{
					Type: "text",
					Text: fmt.Sprintf("Error executing query: %v", err),
				},
			},
			IsError: true,
		}, nil
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}
		results = append(results, row)
	}

	resultJSON, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal results: %w", err)
	}

	return &modelcontextprotocol.CallToolResult{
		Content: []modelcontextprotocol.Content{
			{
				Type: "text",
				Text: string(resultJSON),
			},
		},
	}, nil
}

func (s *SQLiteServer) handleExecuteStatement(args map[string]interface{}) (*modelcontextprotocol.CallToolResult, error) {
	statement, ok := args["statement"].(string)
	if !ok {
		return nil, fmt.Errorf("statement parameter is required and must be a string")
	}

	result, err := s.db.Exec(statement)
	if err != nil {
		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				{
					Type: "text",
					Text: fmt.Sprintf("Error executing statement: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	rowsAffected, _ := result.RowsAffected()
	lastInsertID, _ := result.LastInsertId()

	resultText := fmt.Sprintf("Statement executed successfully.\nRows affected: %d\nLast insert ID: %d", rowsAffected, lastInsertID)

	return &modelcontextprotocol.CallToolResult{
		Content: []modelcontextprotocol.Content{
			{
				Type: "text",
				Text: resultText,
			},
		},
	}, nil
}

func (s *SQLiteServer) handleGetSchema(args map[string]interface{}) (*modelcontextprotocol.CallToolResult, error) {
	query := `SELECT name, type, sql FROM sqlite_master WHERE type IN ('table', 'view', 'index') ORDER BY type, name`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}
	defer rows.Close()

	var schema []map[string]interface{}
	for rows.Next() {
		var name, objType, sql string
		if err := rows.Scan(&name, &objType, &sql); err != nil {
			return nil, fmt.Errorf("failed to scan schema row: %w", err)
		}
		schema = append(schema, map[string]interface{}{
			"name": name,
			"type": objType,
			"sql":  sql,
		})
	}

	schemaJSON, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	return &modelcontextprotocol.CallToolResult{
		Content: []modelcontextprotocol.Content{
			{
				Type: "text",
				Text: string(schemaJSON),
			},
		},
	}, nil
}

func main() {
	dbPath := os.Getenv("SQLITE_DB_PATH")
	if dbPath == "" {
		dbPath = "./test.db"
	}

	server, err := NewSQLiteServer(dbPath)
	if err != nil {
		log.Fatalf("Failed to create SQLite server: %v", err)
	}
	defer server.Close()

	mcpServer := mcp.NewServer("mcp-sqlite-server", "1.0.0")

	// Add tools
	mcpServer.AddTool(modelcontextprotocol.Tool{
		Name:        "execute_query",
		Description: "Execute a SELECT query and return results as JSON",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "SQL SELECT query to execute",
				},
			},
			"required": []string{"query"},
		},
	})

	mcpServer.AddTool(modelcontextprotocol.Tool{
		Name:        "execute_statement",
		Description: "Execute an INSERT, UPDATE, DELETE, or DDL statement",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"statement": map[string]interface{}{
					"type":        "string",
					"description": "SQL statement to execute",
				},
			},
			"required": []string{"statement"},
		},
	})

	mcpServer.AddTool(modelcontextprotocol.Tool{
		Name:        "get_schema",
		Description: "Get the database schema (tables, views, indexes)",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	})

	// Add tool handlers
	mcpServer.OnToolCall("execute_query", server.handleExecuteQuery)
	mcpServer.OnToolCall("execute_statement", server.handleExecuteStatement)
	mcpServer.OnToolCall("get_schema", server.handleGetSchema)

	if err := mcpServer.Serve(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
