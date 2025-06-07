package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"syscall"
	"time"

	"github.com/tmc/mcp"
)

const (
	ServerName    = "mcp-todo-server"
	ServerVersion = "0.1.0"
)

type TodoItem struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status"`   // "pending", "in_progress", "completed", "cancelled"
	Priority    string    `json:"priority"` // "low", "medium", "high", "urgent"
	DueDate     *string   `json:"due_date,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Tags        []string  `json:"tags,omitempty"`
}

type TodoManager struct {
	todos    []TodoItem
	nextID   int
	filePath string
}

func NewTodoManager() *TodoManager {
	homeDir, _ := os.UserHomeDir()
	filePath := filepath.Join(homeDir, ".mcp-todo-server.json")

	tm := &TodoManager{
		todos:    []TodoItem{},
		nextID:   1,
		filePath: filePath,
	}

	tm.loadFromFile()
	return tm
}

func (tm *TodoManager) loadFromFile() {
	data, err := os.ReadFile(tm.filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Warning: Could not read todo file: %v", err)
		}
		return
	}

	var fileData struct {
		NextID int        `json:"next_id"`
		Todos  []TodoItem `json:"todos"`
	}

	if err := json.Unmarshal(data, &fileData); err != nil {
		log.Printf("Warning: Could not parse todo file: %v", err)
		return
	}

	tm.nextID = fileData.NextID
	tm.todos = fileData.Todos
}

func (tm *TodoManager) saveToFile() error {
	fileData := struct {
		NextID int        `json:"next_id"`
		Todos  []TodoItem `json:"todos"`
	}{
		NextID: tm.nextID,
		Todos:  tm.todos,
	}

	data, err := json.MarshalIndent(fileData, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(tm.filePath, data, 0644)
}

func (tm *TodoManager) addTodo(title, description, priority, dueDate string, tags []string) TodoItem {
	now := time.Now()

	todo := TodoItem{
		ID:          tm.nextID,
		Title:       title,
		Description: description,
		Status:      "pending",
		Priority:    priority,
		CreatedAt:   now,
		UpdatedAt:   now,
		Tags:        tags,
	}

	if dueDate != "" {
		todo.DueDate = &dueDate
	}

	tm.todos = append(tm.todos, todo)
	tm.nextID++
	tm.saveToFile()

	return todo
}

func (tm *TodoManager) getTodos(status, priority string, tags []string) []TodoItem {
	var filtered []TodoItem

	for _, todo := range tm.todos {
		// Filter by status
		if status != "" && todo.Status != status {
			continue
		}

		// Filter by priority
		if priority != "" && todo.Priority != priority {
			continue
		}

		// Filter by tags
		if len(tags) > 0 {
			hasTag := false
			for _, filterTag := range tags {
				for _, todoTag := range todo.Tags {
					if todoTag == filterTag {
						hasTag = true
						break
					}
				}
				if hasTag {
					break
				}
			}
			if !hasTag {
				continue
			}
		}

		filtered = append(filtered, todo)
	}

	// Sort by priority and creation date
	sort.Slice(filtered, func(i, j int) bool {
		// Priority order: urgent > high > medium > low
		priorities := map[string]int{"urgent": 4, "high": 3, "medium": 2, "low": 1}
		priI := priorities[filtered[i].Priority]
		priJ := priorities[filtered[j].Priority]

		if priI != priJ {
			return priI > priJ
		}
		return filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
	})

	return filtered
}

func (tm *TodoManager) updateTodo(id int, status, priority, title, description, dueDate *string, tags []string) (*TodoItem, error) {
	for i, todo := range tm.todos {
		if todo.ID == id {
			updated := todo
			updated.UpdatedAt = time.Now()

			if status != nil {
				if !isValidStatus(*status) {
					return nil, fmt.Errorf("invalid status: %s", *status)
				}
				updated.Status = *status
			}

			if priority != nil {
				if !isValidPriority(*priority) {
					return nil, fmt.Errorf("invalid priority: %s", *priority)
				}
				updated.Priority = *priority
			}

			if title != nil {
				updated.Title = *title
			}

			if description != nil {
				updated.Description = *description
			}

			if dueDate != nil {
				if *dueDate == "" {
					updated.DueDate = nil
				} else {
					updated.DueDate = dueDate
				}
			}

			if tags != nil {
				updated.Tags = tags
			}

			tm.todos[i] = updated
			tm.saveToFile()
			return &updated, nil
		}
	}

	return nil, fmt.Errorf("todo with ID %d not found", id)
}

func (tm *TodoManager) deleteTodo(id int) error {
	for i, todo := range tm.todos {
		if todo.ID == id {
			tm.todos = append(tm.todos[:i], tm.todos[i+1:]...)
			tm.saveToFile()
			return nil
		}
	}

	return fmt.Errorf("todo with ID %d not found", id)
}

func (tm *TodoManager) getStats() map[string]interface{} {
	stats := map[string]interface{}{
		"total":       len(tm.todos),
		"pending":     0,
		"in_progress": 0,
		"completed":   0,
		"cancelled":   0,
		"by_priority": map[string]int{
			"urgent": 0,
			"high":   0,
			"medium": 0,
			"low":    0,
		},
	}

	for _, todo := range tm.todos {
		switch todo.Status {
		case "pending":
			stats["pending"] = stats["pending"].(int) + 1
		case "in_progress":
			stats["in_progress"] = stats["in_progress"].(int) + 1
		case "completed":
			stats["completed"] = stats["completed"].(int) + 1
		case "cancelled":
			stats["cancelled"] = stats["cancelled"].(int) + 1
		}

		if byPriority, ok := stats["by_priority"].(map[string]int); ok {
			byPriority[todo.Priority]++
		}
	}

	return stats
}

func isValidStatus(status string) bool {
	validStatuses := []string{"pending", "in_progress", "completed", "cancelled"}
	for _, valid := range validStatuses {
		if status == valid {
			return true
		}
	}
	return false
}

func isValidPriority(priority string) bool {
	validPriorities := []string{"low", "medium", "high", "urgent"}
	for _, valid := range validPriorities {
		if priority == valid {
			return true
		}
	}
	return false
}

func main() {
	log.SetOutput(os.Stderr)
	log.Println("Starting MCP Todo Server...")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	todoManager := NewTodoManager()

	server := mcp.NewServer(ServerName, ServerVersion,
		mcp.WithServerInstructions("A todo/task management server for organizing and tracking tasks"),
	)

	registerTodoTools(server, todoManager)

	log.Println("Starting protocol server via stdio...")
	if err := server.Serve(ctx, nil); err != nil {
		if err != context.Canceled {
			log.Fatalf("Error serving: %v", err)
		}
		log.Println("Server terminated.")
	}
}

func registerTodoTools(server *mcp.Server, tm *TodoManager) {
	// Add todo tool
	addTodoTool := mcp.Tool{
		Name:        "add_todo",
		Description: "Add a new todo item",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"title": {
					"type": "string",
					"description": "The title of the todo item"
				},
				"description": {
					"type": "string",
					"description": "Optional description of the todo item"
				},
				"priority": {
					"type": "string",
					"description": "Priority level",
					"enum": ["low", "medium", "high", "urgent"],
					"default": "medium"
				},
				"due_date": {
					"type": "string",
					"description": "Due date in YYYY-MM-DD format (optional)"
				},
				"tags": {
					"type": "array",
					"items": {
						"type": "string"
					},
					"description": "Optional tags for the todo item"
				}
			},
			"required": ["title"]
		}`),
	}

	server.RegisterTool(addTodoTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		title, ok := params["title"].(string)
		if !ok || title == "" {
			return nil, fmt.Errorf("title is required and must be a string")
		}

		description := ""
		if d, ok := params["description"].(string); ok {
			description = d
		}

		priority := "medium"
		if p, ok := params["priority"].(string); ok && p != "" {
			if !isValidPriority(p) {
				return nil, fmt.Errorf("invalid priority: %s", p)
			}
			priority = p
		}

		dueDate := ""
		if d, ok := params["due_date"].(string); ok {
			dueDate = d
		}

		var tags []string
		if t, ok := params["tags"].([]interface{}); ok {
			for _, tag := range t {
				if tagStr, ok := tag.(string); ok {
					tags = append(tags, tagStr)
				}
			}
		}

		todo := tm.addTodo(title, description, priority, dueDate, tags)

		resultJSON, _ := json.MarshalIndent(todo, "", "  ")
		return &mcp.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": string(resultJSON),
				},
			},
		}, nil
	})

	// List todos tool
	listTodosTool := mcp.Tool{
		Name:        "list_todos",
		Description: "List todo items with optional filtering",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"status": {
					"type": "string",
					"description": "Filter by status",
					"enum": ["pending", "in_progress", "completed", "cancelled"]
				},
				"priority": {
					"type": "string",
					"description": "Filter by priority",
					"enum": ["low", "medium", "high", "urgent"]
				},
				"tags": {
					"type": "array",
					"items": {
						"type": "string"
					},
					"description": "Filter by tags (items with any of these tags will be included)"
				}
			}
		}`),
	}

	server.RegisterTool(listTodosTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		status := ""
		if s, ok := params["status"].(string); ok {
			status = s
		}

		priority := ""
		if p, ok := params["priority"].(string); ok {
			priority = p
		}

		var tags []string
		if t, ok := params["tags"].([]interface{}); ok {
			for _, tag := range t {
				if tagStr, ok := tag.(string); ok {
					tags = append(tags, tagStr)
				}
			}
		}

		todos := tm.getTodos(status, priority, tags)

		result := map[string]interface{}{
			"todos": todos,
			"count": len(todos),
		}

		resultJSON, _ := json.MarshalIndent(result, "", "  ")
		return &mcp.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": string(resultJSON),
				},
			},
		}, nil
	})

	// Update todo tool
	updateTodoTool := mcp.Tool{
		Name:        "update_todo",
		Description: "Update an existing todo item",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id": {
					"type": "number",
					"description": "The ID of the todo item to update"
				},
				"title": {
					"type": "string",
					"description": "New title for the todo item"
				},
				"description": {
					"type": "string",
					"description": "New description for the todo item"
				},
				"status": {
					"type": "string",
					"description": "New status",
					"enum": ["pending", "in_progress", "completed", "cancelled"]
				},
				"priority": {
					"type": "string",
					"description": "New priority level",
					"enum": ["low", "medium", "high", "urgent"]
				},
				"due_date": {
					"type": "string",
					"description": "New due date in YYYY-MM-DD format (empty string to remove)"
				},
				"tags": {
					"type": "array",
					"items": {
						"type": "string"
					},
					"description": "New tags for the todo item"
				}
			},
			"required": ["id"]
		}`),
	}

	server.RegisterTool(updateTodoTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		idFloat, ok := params["id"].(float64)
		if !ok {
			return nil, fmt.Errorf("id is required and must be a number")
		}
		id := int(idFloat)

		var status, priority, title, description, dueDate *string
		var tags []string

		if s, ok := params["status"].(string); ok {
			status = &s
		}

		if p, ok := params["priority"].(string); ok {
			priority = &p
		}

		if t, ok := params["title"].(string); ok {
			title = &t
		}

		if d, ok := params["description"].(string); ok {
			description = &d
		}

		if d, ok := params["due_date"].(string); ok {
			dueDate = &d
		}

		if t, ok := params["tags"].([]interface{}); ok {
			for _, tag := range t {
				if tagStr, ok := tag.(string); ok {
					tags = append(tags, tagStr)
				}
			}
		}

		todo, err := tm.updateTodo(id, status, priority, title, description, dueDate, tags)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Error updating todo: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		resultJSON, _ := json.MarshalIndent(todo, "", "  ")
		return &mcp.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": string(resultJSON),
				},
			},
		}, nil
	})

	// Delete todo tool
	deleteTodoTool := mcp.Tool{
		Name:        "delete_todo",
		Description: "Delete a todo item",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id": {
					"type": "number",
					"description": "The ID of the todo item to delete"
				}
			},
			"required": ["id"]
		}`),
	}

	server.RegisterTool(deleteTodoTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		idFloat, ok := params["id"].(float64)
		if !ok {
			return nil, fmt.Errorf("id is required and must be a number")
		}
		id := int(idFloat)

		err := tm.deleteTodo(id)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Error deleting todo: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		return &mcp.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": fmt.Sprintf("Todo item with ID %d has been deleted", id),
				},
			},
		}, nil
	})

	// Statistics tool
	statsTool := mcp.Tool{
		Name:        "todo_stats",
		Description: "Get statistics about todos",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {}
		}`),
	}

	server.RegisterTool(statsTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		stats := tm.getStats()

		resultJSON, _ := json.MarshalIndent(stats, "", "  ")
		return &mcp.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": string(resultJSON),
				},
			},
		}, nil
	})

	log.Println("Registered todo tools: add_todo, list_todos, update_todo, delete_todo, todo_stats")
}
