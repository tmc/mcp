package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
	"github.com/tmc/mcp"
	"github.com/tmc/mcp/modelcontextprotocol"
)

type PostgreSQLServer struct {
	db               *sql.DB
	connectionString string
	readOnlyMode     bool
}

func NewPostgreSQLServer(connectionString string, readOnlyMode bool) (*PostgreSQLServer, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &PostgreSQLServer{
		db:               db,
		connectionString: connectionString,
		readOnlyMode:     readOnlyMode,
	}, nil
}

func (ps *PostgreSQLServer) Close() error {
	return ps.db.Close()
}

func (ps *PostgreSQLServer) listTables(schema string) (string, error) {
	if schema == "" {
		schema = "public"
	}

	query := `
		SELECT table_name, table_type 
		FROM information_schema.tables 
		WHERE table_schema = $1 
		ORDER BY table_name
	`

	rows, err := ps.db.Query(query, schema)
	if err != nil {
		return "", fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Tables in schema '%s':\n", schema))
	result.WriteString("Type\t\tName\n")
	result.WriteString("----\t\t----\n")

	for rows.Next() {
		var tableName, tableType string
		if err := rows.Scan(&tableName, &tableType); err != nil {
			return "", fmt.Errorf("failed to scan row: %w", err)
		}
		result.WriteString(fmt.Sprintf("%s\t\t%s\n", tableType, tableName))
	}

	return result.String(), nil
}

func (ps *PostgreSQLServer) describeTable(tableName, schema string) (string, error) {
	if schema == "" {
		schema = "public"
	}

	query := `
		SELECT column_name, data_type, is_nullable, column_default, character_maximum_length
		FROM information_schema.columns 
		WHERE table_name = $1 AND table_schema = $2
		ORDER BY ordinal_position
	`

	rows, err := ps.db.Query(query, tableName, schema)
	if err != nil {
		return "", fmt.Errorf("failed to describe table: %w", err)
	}
	defer rows.Close()

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Table: %s.%s\n", schema, tableName))
	result.WriteString("Column\t\tType\t\tNullable\tDefault\t\tLength\n")
	result.WriteString("------\t\t----\t\t--------\t-------\t\t------\n")

	for rows.Next() {
		var columnName, dataType, isNullable string
		var columnDefault sql.NullString
		var charMaxLength sql.NullInt64

		if err := rows.Scan(&columnName, &dataType, &isNullable, &columnDefault, &charMaxLength); err != nil {
			return "", fmt.Errorf("failed to scan row: %w", err)
		}

		defaultVal := "NULL"
		if columnDefault.Valid {
			defaultVal = columnDefault.String
		}

		lengthVal := "N/A"
		if charMaxLength.Valid {
			lengthVal = fmt.Sprintf("%d", charMaxLength.Int64)
		}

		result.WriteString(fmt.Sprintf("%s\t\t%s\t\t%s\t\t%s\t\t%s\n",
			columnName, dataType, isNullable, defaultVal, lengthVal))
	}

	return result.String(), nil
}

func (ps *PostgreSQLServer) executeQuery(query string, limit int) (string, error) {
	if ps.readOnlyMode {
		// Basic check to prevent writes in read-only mode
		queryLower := strings.ToLower(strings.TrimSpace(query))
		if strings.HasPrefix(queryLower, "insert") ||
			strings.HasPrefix(queryLower, "update") ||
			strings.HasPrefix(queryLower, "delete") ||
			strings.HasPrefix(queryLower, "drop") ||
			strings.HasPrefix(queryLower, "create") ||
			strings.HasPrefix(queryLower, "alter") {
			return "", fmt.Errorf("write operations not allowed in read-only mode")
		}
	}

	if limit <= 0 {
		limit = 100 // default limit
	}

	// Add LIMIT if not present and it's a SELECT query
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(query)), "select") &&
		!strings.Contains(strings.ToLower(query), "limit") {
		query = fmt.Sprintf("%s LIMIT %d", query, limit)
	}

	rows, err := ps.db.Query(query)
	if err != nil {
		return "", fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return "", fmt.Errorf("failed to get columns: %w", err)
	}

	var result strings.Builder

	// Write header
	for i, col := range columns {
		if i > 0 {
			result.WriteString("\t")
		}
		result.WriteString(col)
	}
	result.WriteString("\n")

	// Write separator
	for i := range columns {
		if i > 0 {
			result.WriteString("\t")
		}
		result.WriteString("----")
	}
	result.WriteString("\n")

	// Write data
	values := make([]interface{}, len(columns))
	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	rowCount := 0
	for rows.Next() {
		if err := rows.Scan(scanArgs...); err != nil {
			return "", fmt.Errorf("failed to scan row: %w", err)
		}

		for i, val := range values {
			if i > 0 {
				result.WriteString("\t")
			}
			if val == nil {
				result.WriteString("NULL")
			} else {
				result.WriteString(fmt.Sprintf("%v", val))
			}
		}
		result.WriteString("\n")
		rowCount++
	}

	result.WriteString(fmt.Sprintf("\n(%d rows)\n", rowCount))
	return result.String(), nil
}

func (ps *PostgreSQLServer) listSchemas() (string, error) {
	query := `
		SELECT schema_name 
		FROM information_schema.schemata 
		WHERE schema_name NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
		ORDER BY schema_name
	`

	rows, err := ps.db.Query(query)
	if err != nil {
		return "", fmt.Errorf("failed to query schemas: %w", err)
	}
	defer rows.Close()

	var result strings.Builder
	result.WriteString("Available schemas:\n")

	for rows.Next() {
		var schemaName string
		if err := rows.Scan(&schemaName); err != nil {
			return "", fmt.Errorf("failed to scan row: %w", err)
		}
		result.WriteString(fmt.Sprintf("- %s\n", schemaName))
	}

	return result.String(), nil
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: mcp-postgresql-server <connection-string> [--read-only]")
	}

	connectionString := os.Args[1]
	readOnlyMode := len(os.Args) > 2 && os.Args[2] == "--read-only"

	// Initialize PostgreSQL server
	ps, err := NewPostgreSQLServer(connectionString, readOnlyMode)
	if err != nil {
		log.Fatalf("Failed to initialize PostgreSQL server: %v", err)
	}
	defer ps.Close()

	// Create server with name and version
	srv := mcp.NewServer("postgresql-server", "1.0.0")

	// Register list_tables tool
	srv.RegisterTool("list_tables", "List all tables in a PostgreSQL database schema", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var schema string = "public" // default
		if schemaRaw, exists := args["schema"]; exists {
			if err := json.Unmarshal(schemaRaw, &schema); err != nil {
				return nil, fmt.Errorf("invalid schema argument: %w", err)
			}
		}

		tables, err := ps.listTables(schema)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error listing tables: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}

		log.Printf("Listed tables for schema: %s", schema)

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: tables,
				},
			},
		}, nil
	})

	// Register describe_table tool
	srv.RegisterTool("describe_table", "Describe the structure of a PostgreSQL table", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var tableNameRaw json.RawMessage
		var exists bool
		if tableNameRaw, exists = args["table"]; !exists {
			return nil, fmt.Errorf("missing required argument: table")
		}

		var tableName string
		if err := json.Unmarshal(tableNameRaw, &tableName); err != nil {
			return nil, fmt.Errorf("invalid table argument: %w", err)
		}

		var schema string = "public" // default
		if schemaRaw, exists := args["schema"]; exists {
			if err := json.Unmarshal(schemaRaw, &schema); err != nil {
				return nil, fmt.Errorf("invalid schema argument: %w", err)
			}
		}

		description, err := ps.describeTable(tableName, schema)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error describing table: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}

		log.Printf("Described table: %s.%s", schema, tableName)

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: description,
				},
			},
		}, nil
	})

	// Register execute_query tool
	srv.RegisterTool("execute_query", "Execute a read-only SQL query against the PostgreSQL database", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var queryRaw json.RawMessage
		var exists bool
		if queryRaw, exists = args["query"]; !exists {
			return nil, fmt.Errorf("missing required argument: query")
		}

		var query string
		if err := json.Unmarshal(queryRaw, &query); err != nil {
			return nil, fmt.Errorf("invalid query argument: %w", err)
		}

		var limit int = 100 // default
		if limitRaw, exists := args["limit"]; exists {
			if err := json.Unmarshal(limitRaw, &limit); err != nil {
				return nil, fmt.Errorf("invalid limit argument: %w", err)
			}
		}

		result, err := ps.executeQuery(query, limit)
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error executing query: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}

		log.Printf("Executed query: %s", query)

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: result,
				},
			},
		}, nil
	})

	// Register list_schemas tool
	srv.RegisterTool("list_schemas", "List all schemas in the PostgreSQL database", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		schemas, err := ps.listSchemas()
		if err != nil {
			return &modelcontextprotocol.CallToolResult{
				Content: []modelcontextprotocol.Content{
					modelcontextprotocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error listing schemas: %s", err.Error()),
					},
				},
				IsError: boolPtr(true),
			}, nil
		}

		log.Printf("Listed database schemas")

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: schemas,
				},
			},
		}, nil
	})

	// Start server with stdio transport
	transport := mcp.StdioTransport{}
	readOnlyStatus := ""
	if ps.readOnlyMode {
		readOnlyStatus = " (read-only mode)"
	}
	log.Printf("PostgreSQL server running on stdio%s", readOnlyStatus)

	if err := srv.Serve(context.Background(), transport); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func boolPtr(b bool) *bool {
	return &b
}
