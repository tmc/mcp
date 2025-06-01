# MCP Todo Server

A Model Context Protocol (MCP) server that provides comprehensive todo/task management capabilities with persistent storage.

## Features

- **Task Management**: Create, read, update, and delete todo items
- **Status Tracking**: Track tasks through different states (pending, in_progress, completed, cancelled)
- **Priority System**: Organize tasks by priority levels (low, medium, high, urgent)
- **Tagging**: Add and filter tasks by custom tags
- **Due Dates**: Set and track due dates for tasks
- **Statistics**: Get insights about your task completion and distribution
- **Persistent Storage**: All tasks are saved to a local JSON file

## Setup

1. **Build the Server**:
   ```bash
   go build -o mcp-todo-server
   ```

2. **Run the Server**:
   ```bash
   ./mcp-todo-server
   ```

The server will automatically create a `.mcp-todo-server.json` file in your home directory to store todos persistently.

## Tools

### `add_todo`
Create a new todo item.

**Parameters:**
- `title` (required): The title of the todo item
- `description` (optional): Detailed description of the task
- `priority` (optional): "low", "medium", "high", or "urgent" (default: "medium")
- `due_date` (optional): Due date in YYYY-MM-DD format
- `tags` (optional): Array of tags for categorization

**Example:**
```json
{
  "title": "Complete project proposal",
  "description": "Write and review the Q1 project proposal",
  "priority": "high",
  "due_date": "2024-02-15",
  "tags": ["work", "proposal", "deadline"]
}
```

### `list_todos`
List todo items with optional filtering.

**Parameters:**
- `status` (optional): Filter by status ("pending", "in_progress", "completed", "cancelled")
- `priority` (optional): Filter by priority ("low", "medium", "high", "urgent")
- `tags` (optional): Array of tags to filter by (items with any of these tags will be included)

**Example:**
```json
{
  "status": "pending",
  "priority": "high"
}
```

**Response:**
```json
{
  "todos": [
    {
      "id": 1,
      "title": "Complete project proposal",
      "description": "Write and review the Q1 project proposal",
      "status": "pending",
      "priority": "high",
      "due_date": "2024-02-15",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:00Z",
      "tags": ["work", "proposal", "deadline"]
    }
  ],
  "count": 1
}
```

### `update_todo`
Update an existing todo item.

**Parameters:**
- `id` (required): The ID of the todo item to update
- `title` (optional): New title
- `description` (optional): New description
- `status` (optional): New status ("pending", "in_progress", "completed", "cancelled")
- `priority` (optional): New priority ("low", "medium", "high", "urgent")
- `due_date` (optional): New due date in YYYY-MM-DD format (empty string to remove)
- `tags` (optional): New tags array

**Example:**
```json
{
  "id": 1,
  "status": "in_progress",
  "priority": "urgent"
}
```

### `delete_todo`
Delete a todo item.

**Parameters:**
- `id` (required): The ID of the todo item to delete

**Example:**
```json
{
  "id": 1
}
```

### `todo_stats`
Get statistics about your todos.

**Response:**
```json
{
  "total": 10,
  "pending": 5,
  "in_progress": 2,
  "completed": 3,
  "cancelled": 0,
  "by_priority": {
    "urgent": 1,
    "high": 3,
    "medium": 4,
    "low": 2
  }
}
```

## Task Statuses

- **pending**: Task created but not yet started
- **in_progress**: Task is currently being worked on
- **completed**: Task has been finished successfully
- **cancelled**: Task has been abandoned or is no longer needed

## Priority Levels

Tasks are automatically sorted by priority (urgent > high > medium > low) and then by creation date.

- **urgent**: Critical tasks that need immediate attention
- **high**: Important tasks with significant impact
- **medium**: Regular tasks with moderate importance
- **low**: Nice-to-have tasks or low-impact items

## Configuration for Claude Desktop

Add this to your Claude Desktop configuration:

```json
{
  "mcpServers": {
    "todo": {
      "command": "/path/to/mcp-todo-server"
    }
  }
}
```

## Example Usage

Once connected, you can ask Claude:
- "Add a new todo: 'Buy groceries' with high priority"
- "Show me all pending tasks"
- "Mark todo ID 5 as completed"
- "List all high priority tasks"
- "What are my todo statistics?"
- "Update todo 3 to add the tag 'urgent'"
- "Show me all todos tagged with 'work'"

## Data Storage

All todos are stored in `~/.mcp-todo-server.json`. This file contains:
- All todo items with their complete information
- Auto-incrementing ID counter
- Timestamps for creation and updates

The file is automatically created on first use and updated whenever todos are modified.

## Error Handling

The server includes comprehensive error handling for:
- Invalid todo IDs
- Invalid status or priority values
- Missing required parameters
- File system errors
- JSON parsing errors