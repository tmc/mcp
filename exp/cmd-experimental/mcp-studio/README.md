# mcp-studio

Visual development environment for MCP (Model Context Protocol) servers with web-based interface, visual flow designer, real-time testing, and debugging capabilities.

## Features

- **Visual Flow Designer**: Drag-and-drop interface for building MCP workflows
- **Real-time Collaboration**: Multiple users can work on the same project simultaneously
- **Server Management**: Connect to and manage multiple MCP servers
- **Project Management**: Organize workflows into projects with version control
- **Debugging Interface**: Step-through debugging with breakpoints and variable inspection
- **Real-time Testing**: Test MCP operations in real-time with immediate feedback
- **WebSocket Integration**: Real-time updates and live collaboration
- **Responsive Design**: Works on desktop and mobile devices
- **Modern UI**: Clean, intuitive interface with dark/light mode support

## Installation

```bash
go install github.com/tmc/mcp/cmd/mcp-studio@latest
```

## Usage

### Starting the Studio

```bash
# Start with default settings
mcp-studio

# Start on custom port
mcp-studio -port 3000

# Start with custom workspace
mcp-studio -workspace ~/my-mcp-workspace

# Start with configuration file
mcp-studio -config ~/mcp-studio.json
```

### Configuration

The studio uses a JSON configuration file (default: `~/.mcp-studio-config.json`):

```json
{
  "port": 8080,
  "workspace_dir": "~/.mcp-studio",
  "projects": {},
  "servers": {
    "timeserver": {
      "id": "timeserver",
      "name": "Time Server",
      "description": "Example time server",
      "transport": "stdio",
      "command": ["go", "run", "./examples/servers/mcp-time-server"],
      "auto_start": true,
      "health_check": {
        "enabled": true,
        "interval": "30s",
        "timeout": "5s",
        "retries": 3
      }
    }
  },
  "settings": {
    "theme": "light",
    "auto_save": true,
    "auto_save_interval": 30,
    "show_grid_lines": true,
    "enable_collaboration": true,
    "enable_debugger": true,
    "max_recent_projects": 10,
    "enable_hot_reload": true
  }
}
```

## Key Components

### Projects

Projects are the main organizational unit in MCP Studio. Each project contains:

- **Flows**: Visual workflows composed of nodes and connections
- **Servers**: MCP server connections and configurations
- **Variables**: Project-wide variables and settings
- **Settings**: Project-specific configuration options

### Flows

Flows are visual representations of MCP operations:

- **Nodes**: Individual operations (tool calls, resource reads, prompts)
- **Edges**: Connections between nodes representing data flow
- **Variables**: Flow-specific variables and parameters
- **Execution**: Real-time execution with status tracking

### Servers

MCP servers provide the backend functionality:

- **Connection Management**: Automatic connection and health monitoring
- **Transport Support**: stdio, HTTP, and SSE transports
- **Tool Discovery**: Automatic discovery of available tools, resources, and prompts
- **Health Checks**: Regular health monitoring with retry logic

## Visual Flow Designer

### Node Types

The flow designer supports various node types:

#### MCP Operations
- **Tool Call**: Execute MCP tools with parameters
- **Resource Read**: Read MCP resources
- **Prompt Get**: Retrieve MCP prompts

#### Control Flow
- **Condition**: Conditional branching based on data
- **Loop**: Iterate over data or repeat operations
- **Delay**: Add delays between operations

#### Data Operations
- **Variable**: Store and retrieve variables
- **Transform**: Transform data between operations
- **Output**: Display results and debugging information

### Flow Execution

Flows can be executed in real-time with:

- **Step-by-step execution**: Run nodes individually
- **Breakpoints**: Pause execution at specific nodes
- **Variable inspection**: View variable values at runtime
- **Error handling**: Comprehensive error reporting and recovery

### Collaboration Features

- **Real-time updates**: Changes are synchronized across all connected users
- **Multi-user editing**: Multiple users can edit flows simultaneously
- **Live cursors**: See where other users are working
- **Change notifications**: Get notified of changes made by others

## Web Interface

### Dashboard

The main dashboard provides:

- **Project overview**: Quick access to all projects
- **Recent projects**: Recently opened projects
- **Server status**: Overview of all connected servers
- **Quick actions**: Common operations and shortcuts

### Flow Editor

The flow editor includes:

- **Canvas**: Main editing area with zoom and pan
- **Node library**: Drag-and-drop node components
- **Properties panel**: Edit node properties and configuration
- **Toolbar**: Common actions and tools
- **Minimap**: Navigate large flows easily

### Server Management

- **Server list**: All configured servers with status
- **Connection controls**: Connect/disconnect servers
- **Health monitoring**: Real-time health status
- **Tool discovery**: Browse available tools and resources

## API Reference

### WebSocket API

The studio provides a WebSocket API for real-time communication:

#### Connection
```javascript
const ws = new WebSocket('ws://localhost:8080/ws');
```

#### Message Types
- `ping/pong`: Heartbeat messages
- `subscribe_project`: Subscribe to project updates
- `flow_update`: Flow changes
- `node_update`: Node status changes
- `server_status`: Server connectivity changes

#### Example Usage
```javascript
ws.send(JSON.stringify({
    type: 'subscribe_project',
    data: 'project-id'
}));
```

### REST API

The studio provides a REST API for programmatic access:

#### Projects
- `GET /api/projects` - List all projects
- `POST /api/projects` - Create a new project
- `GET /api/projects/{id}` - Get project details
- `PUT /api/projects/{id}` - Update project
- `DELETE /api/projects/{id}` - Delete project

#### Flows
- `GET /api/projects/{id}/flows` - List project flows
- `POST /api/projects/{id}/flows` - Create new flow
- `GET /api/projects/{id}/flows/{flowId}` - Get flow details
- `PUT /api/projects/{id}/flows/{flowId}` - Update flow
- `POST /api/projects/{id}/flows/{flowId}/run` - Execute flow

#### Servers
- `GET /api/servers` - List all servers
- `POST /api/servers` - Create server configuration
- `POST /api/servers/{id}/connect` - Connect to server
- `POST /api/servers/{id}/disconnect` - Disconnect from server
- `POST /api/servers/{id}/ping` - Ping server

## Development

### Building

```bash
cd cmd/mcp-studio
go build
```

### Running in Development

```bash
# Enable debug mode
mcp-studio -debug

# Use custom workspace for development
mcp-studio -workspace ./dev-workspace -port 3000
```

### Frontend Development

The web interface is built with:

- **HTML5**: Modern semantic HTML
- **CSS3**: Custom CSS with CSS Grid and Flexbox
- **JavaScript**: ES6+ with WebSocket support
- **Canvas API**: For flow visualization
- **Web APIs**: File API, WebSocket, etc.

### Project Structure

```
cmd/mcp-studio/
├── main.go              # Main application
├── static/              # Static web assets
│   ├── css/            # Stylesheets
│   ├── js/             # JavaScript files
│   └── images/         # Images and icons
├── templates/          # HTML templates
├── README.md           # This file
└── go.mod              # Go module file
```

## Configuration Options

### Command Line Flags

- `-port <int>` - Port to listen on (default: 8080)
- `-workspace <path>` - Workspace directory (default: `~/.mcp-studio`)
- `-config <path>` - Configuration file path
- `-debug` - Enable debug mode
- `-version` - Show version information

### Environment Variables

- `MCP_STUDIO_PORT` - Override default port
- `MCP_STUDIO_WORKSPACE` - Override workspace directory
- `MCP_STUDIO_CONFIG` - Override configuration file path

### Studio Settings

- `theme` - UI theme (light/dark)
- `auto_save` - Enable automatic saving
- `auto_save_interval` - Auto-save interval in seconds
- `show_grid_lines` - Show grid lines in flow editor
- `enable_collaboration` - Enable real-time collaboration
- `enable_debugger` - Enable debugging features
- `max_recent_projects` - Maximum recent projects to remember
- `enable_hot_reload` - Enable hot reload for development

## Security

### Authentication

Currently, MCP Studio runs locally without authentication. For production use, consider:

- Adding authentication middleware
- Implementing user management
- Setting up HTTPS
- Configuring CORS policies

### Network Security

- Studio binds to localhost by default
- All MCP server connections are isolated
- WebSocket connections use same-origin policy
- No external dependencies by default

## Troubleshooting

### Common Issues

1. **Port already in use**
   ```bash
   mcp-studio -port 3000
   ```

2. **WebSocket connection failed**
   - Check firewall settings
   - Ensure studio is running
   - Try refreshing the page

3. **Server connection failed**
   - Verify server command is correct
   - Check server logs for errors
   - Test server independently

4. **Project not saving**
   - Check workspace directory permissions
   - Ensure enough disk space
   - Check for file system errors

### Debug Mode

Enable debug mode for detailed logging:

```bash
mcp-studio -debug
```

This provides:
- Detailed WebSocket message logging
- Server connection debugging
- Flow execution tracing
- Performance metrics

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

### Development Guidelines

- Follow Go best practices
- Write comprehensive tests
- Document new features
- Maintain backward compatibility
- Use semantic versioning

## License

Part of the MCP Go implementation project.

## Roadmap

- [ ] Plugin system for custom nodes
- [ ] Advanced debugging features
- [ ] Performance monitoring
- [ ] Export/import functionality
- [ ] Template library
- [ ] Team collaboration features
- [ ] Cloud deployment options
- [ ] Mobile app support