# mcp-docs

Documentation generator for MCP (Model Context Protocol) servers with API documentation generation, interactive examples, multi-format output, version management, and integration guides.

## Features

- **Multi-source Documentation**: Generate docs from MCP servers, Go packages, and REST APIs
- **Multiple Output Formats**: HTML, Markdown, JSON, and YAML output
- **Interactive Examples**: Generate interactive examples with code snippets
- **Version Management**: Support for multiple documentation versions
- **Theme Support**: Customizable themes and templates
- **Real-time Generation**: Watch for changes and regenerate automatically
- **Local Development Server**: Built-in server for local documentation preview
- **Search Integration**: Full-text search with multiple providers
- **Analytics Support**: Integration with analytics platforms
- **Responsive Design**: Mobile-friendly documentation

## Installation

```bash
go install github.com/tmc/mcp/cmd/mcp-docs@latest
```

## Usage

### Basic Usage

```bash
# Generate documentation for an MCP server
mcp-docs -server "go run ./examples/servers/mcp-time-server"

# Generate documentation for a Go package
mcp-docs -package ./mcp

# Generate documentation from configuration file
mcp-docs -config mcp-docs.yaml

# Generate in different formats
mcp-docs -format markdown -output ./docs/markdown
mcp-docs -format json -output ./docs/json
```

### Advanced Usage

```bash
# Watch for changes and regenerate
mcp-docs -watch -config mcp-docs.yaml

# Start local development server
mcp-docs -serve -port 8080

# Generate and serve
mcp-docs -serve -watch -config mcp-docs.yaml
```

## Configuration

The documentation generator uses a YAML configuration file (default: `mcp-docs.yaml`):

```yaml
# Source configuration
sources:
  - type: server
    command: ["go", "run", "./examples/servers/mcp-time-server"]
    description: "Time server example"
  - type: package
    path: "./mcp"
    description: "MCP Go client library"
  - type: api
    url: "http://localhost:8080"
    description: "REST API documentation"

# Output configuration
output:
  directory: "./docs"
  formats: ["html", "markdown", "json"]
  assets: "./static"
  templates: "./templates"

# Documentation settings
documentation:
  title: "MCP Documentation"
  description: "Comprehensive MCP implementation documentation"
  version: "1.0.0"
  author: "Your Name"
  base_url: "https://docs.example.com"
  logo: "/static/logo.png"
  favicon: "/static/favicon.ico"
  language: "en"
  theme: "default"
  
  sidebar:
    enabled: true
    collapsible: true
    sections: ["servers", "packages", "apis"]
    
  navigation:
    enabled: true
    items:
      - title: "Home"
        url: "/"
      - title: "Servers"
        url: "/servers"
      - title: "Packages"
        url: "/packages"
        
  search:
    enabled: true
    provider: "local"
    
  analytics:
    enabled: true
    provider: "google"
    config:
      tracking_id: "GA-XXXXXXXXX"
      
  social:
    github: "tmc/mcp"
    twitter: "@example"
    email: "contact@example.com"

# Template configuration
templates:
  directory: "./templates"
  custom:
    layout: "custom-layout.html"
    server: "custom-server.html"
  partials: ["header", "footer", "sidebar"]
  
# Version management
versions:
  enabled: true
  current: "1.0.0"
  available: ["1.0.0", "0.9.0", "0.8.0"]
  directory: "./versions"
  
# Integration settings
integration:
  examples:
    enabled: true
    directory: "./examples"
    languages: ["go", "python", "javascript"]
    interactive: true
    
  playground:
    enabled: true
    url: "https://playground.example.com"
    embed: true
    
  sdk:
    enabled: true
    languages: ["go", "python", "javascript"]
    generate: true
```

## Output Formats

### HTML

Generates a complete HTML documentation website with:

- **Responsive design**: Mobile-friendly interface
- **Interactive navigation**: Sidebar and search
- **Syntax highlighting**: Code examples with highlighting
- **Theme support**: Customizable themes
- **Analytics integration**: Track usage and engagement

### Markdown

Generates comprehensive Markdown documentation:

- **README.md**: Main documentation file
- **Server documentation**: Individual server files
- **Package documentation**: Go package documentation
- **API documentation**: REST API documentation

### JSON

Generates structured JSON documentation:

- **Machine-readable**: For programmatic consumption
- **Complete data**: All extracted information
- **API integration**: Easy integration with other tools
- **Version control**: Track changes over time

### YAML

Generates YAML documentation:

- **Human-readable**: Easy to read and edit
- **Configuration-friendly**: Similar to source configuration
- **Tool integration**: Works with YAML-based tools
- **Version management**: Track versions and changes

## Source Types

### MCP Servers

Automatically documents MCP servers by:

1. **Connecting to the server** using the specified command
2. **Discovering capabilities** through the initialize handshake
3. **Listing available tools**, resources, and prompts
4. **Generating examples** for each operation
5. **Testing connectivity** and health status

Example server configuration:

```yaml
sources:
  - type: server
    command: ["go", "run", "./examples/servers/mcp-time-server"]
    description: "Time server providing current time operations"
    include: ["get_time", "format_time"]
    exclude: ["internal_*"]
```

### Go Packages

Documents Go packages by:

1. **Parsing Go source code** using the `go/parser` package
2. **Extracting documentation** from comments
3. **Analyzing types, functions, and methods**
4. **Generating usage examples**
5. **Cross-referencing dependencies**

Example package configuration:

```yaml
sources:
  - type: package
    path: "./mcp"
    description: "MCP Go client library"
    include: ["*.go"]
    exclude: ["*_test.go", "internal/*"]
```

### REST APIs

Documents REST APIs by:

1. **Parsing OpenAPI/Swagger specifications**
2. **Discovering endpoints** through API exploration
3. **Extracting schemas** and data models
4. **Generating request/response examples**
5. **Testing API endpoints**

Example API configuration:

```yaml
sources:
  - type: api
    url: "http://localhost:8080"
    description: "MCP REST API"
    headers:
      Authorization: "Bearer token"
    include: ["/api/v1/*"]
    exclude: ["/api/v1/internal/*"]
```

## Templates

### Built-in Templates

The generator includes several built-in templates:

- **Default HTML**: Clean, responsive design
- **Markdown**: GitHub-compatible markdown
- **JSON**: Structured data format
- **YAML**: Human-readable configuration format

### Custom Templates

Create custom templates using Go's template syntax:

```html
<!-- templates/custom-layout.html -->
<!DOCTYPE html>
<html>
<head>
    <title>{{.Config.Documentation.Title}}</title>
    <link rel="stylesheet" href="/static/css/custom.css">
</head>
<body>
    <header>
        <h1>{{.Config.Documentation.Title}}</h1>
        <nav>{{template "navigation" .}}</nav>
    </header>
    <main>
        {{template "content" .}}
    </main>
    <footer>
        {{template "footer" .}}
    </footer>
</body>
</html>
```

### Template Functions

Available template functions:

- `toJSON`: Convert data to JSON
- `toYAML`: Convert data to YAML
- `markdown`: Process markdown text
- `code`: Generate code blocks with syntax highlighting
- `formatType`: Format Go type names
- `join`: Join strings with separator
- `lower`, `upper`, `title`: String case conversion

## Development Server

The built-in development server provides:

- **Live reloading**: Automatically reload when files change
- **Hot module replacement**: Update content without full page reload
- **Error reporting**: Display build errors in browser
- **Asset serving**: Serve static assets and resources
- **Search functionality**: Real-time search across documentation

Start the development server:

```bash
mcp-docs -serve -watch -port 8080
```

## Version Management

Support for multiple documentation versions:

### Configuration

```yaml
versions:
  enabled: true
  current: "1.0.0"
  available: ["1.0.0", "0.9.0", "0.8.0"]
  directory: "./versions"
```

### Directory Structure

```
docs/
├── current/           # Current version
├── versions/
│   ├── 1.0.0/        # Version 1.0.0
│   ├── 0.9.0/        # Version 0.9.0
│   └── 0.8.0/        # Version 0.8.0
└── versions.json     # Version metadata
```

### Version Switching

Generated documentation includes:

- **Version selector**: Dropdown to switch between versions
- **Version comparison**: Compare changes between versions
- **Migration guides**: Generated migration information
- **Deprecated features**: Highlight deprecated functionality

## Integration

### Examples

Generate interactive examples:

```yaml
integration:
  examples:
    enabled: true
    directory: "./examples"
    languages: ["go", "python", "javascript"]
    interactive: true
```

Example generation includes:

- **Code snippets**: Working code examples
- **Interactive execution**: Run examples in browser
- **Multiple languages**: Examples in different languages
- **Live editing**: Edit and test examples

### Playground

Integrate with online playgrounds:

```yaml
integration:
  playground:
    enabled: true
    url: "https://playground.example.com"
    embed: true
```

Features:

- **Embedded playground**: Run code directly in documentation
- **Share examples**: Share examples with others
- **Live collaboration**: Real-time collaborative editing
- **Multiple environments**: Support different runtime environments

### SDK Generation

Generate SDK documentation:

```yaml
integration:
  sdk:
    enabled: true
    languages: ["go", "python", "javascript"]
    generate: true
```

SDK features:

- **Auto-generated clients**: Generate client libraries
- **API bindings**: Language-specific bindings
- **Documentation**: Complete SDK documentation
- **Examples**: SDK usage examples

## Command Line Options

### Basic Options

- `-config <file>` - Configuration file path (default: `mcp-docs.yaml`)
- `-output <dir>` - Output directory (default: `./docs`)
- `-format <format>` - Output format (html, markdown, json, yaml)
- `-server <cmd>` - MCP server command to document
- `-package <path>` - Go package path to document

### Advanced Options

- `-watch` - Watch for changes and regenerate
- `-serve` - Start local development server
- `-port <port>` - Port for development server (default: 8080)
- `-debug` - Enable debug mode
- `-version` - Show version information

### Examples

```bash
# Generate HTML documentation
mcp-docs -server "go run ./server" -format html

# Generate all formats
mcp-docs -config docs.yaml -format html,markdown,json

# Development mode
mcp-docs -serve -watch -debug

# Custom output directory
mcp-docs -output /var/www/docs -format html
```

## Analytics and Tracking

### Google Analytics

```yaml
analytics:
  enabled: true
  provider: "google"
  config:
    tracking_id: "GA-XXXXXXXXX"
    anonymize_ip: true
    cookie_domain: "auto"
```

### Plausible Analytics

```yaml
analytics:
  enabled: true
  provider: "plausible"
  config:
    domain: "docs.example.com"
    src: "https://plausible.io/js/plausible.js"
```

### Custom Analytics

```yaml
analytics:
  enabled: true
  provider: "custom"
  config:
    script_url: "https://analytics.example.com/script.js"
    site_id: "12345"
```

## Search Integration

### Local Search

```yaml
search:
  enabled: true
  provider: "local"
  config:
    index_content: true
    search_fields: ["title", "content", "tags"]
    max_results: 10
```

### Algolia Search

```yaml
search:
  enabled: true
  provider: "algolia"
  config:
    app_id: "YOUR_APP_ID"
    search_key: "YOUR_SEARCH_KEY"
    index_name: "docs"
```

### Elasticsearch

```yaml
search:
  enabled: true
  provider: "elasticsearch"
  config:
    endpoint: "https://search.example.com"
    index: "documentation"
    auth_token: "your-token"
```

## Themes

### Built-in Themes

- **Default**: Clean, modern design
- **Dark**: Dark mode variant
- **Minimal**: Minimalist design
- **Corporate**: Professional business theme

### Custom Themes

Create custom themes by:

1. **Creating theme directory**: `themes/mytheme/`
2. **Adding CSS files**: `themes/mytheme/style.css`
3. **Customizing templates**: `themes/mytheme/templates/`
4. **Configuring theme**: Set `theme: "mytheme"` in config

### Theme Structure

```
themes/
└── mytheme/
    ├── style.css          # Main styles
    ├── variables.css      # CSS variables
    ├── components.css     # Component styles
    ├── templates/         # Custom templates
    │   ├── layout.html
    │   └── server.html
    └── assets/           # Theme assets
        ├── images/
        └── fonts/
```

## Performance

### Optimization Features

- **Lazy loading**: Load content on demand
- **Code splitting**: Split large documentation
- **Image optimization**: Compress and optimize images
- **Caching**: Cache generated content
- **Minification**: Minify CSS and JavaScript

### Build Performance

- **Incremental builds**: Only rebuild changed content
- **Parallel processing**: Process multiple sources in parallel
- **Memory optimization**: Efficient memory usage
- **Fast templates**: Optimized template rendering

## Troubleshooting

### Common Issues

1. **Server connection failed**
   ```bash
   # Check if server is running
   mcp-docs -server "go run ./server" -debug
   ```

2. **Template parsing error**
   ```bash
   # Validate template syntax
   mcp-docs -config docs.yaml -debug
   ```

3. **Build errors**
   ```bash
   # Enable debug mode for detailed logs
   mcp-docs -debug -config docs.yaml
   ```

### Debug Mode

Enable debug mode for detailed information:

```bash
mcp-docs -debug -config docs.yaml
```

Debug mode provides:

- **Detailed logging**: Step-by-step process information
- **Error traces**: Full stack traces for errors
- **Performance metrics**: Build time and memory usage
- **Template debugging**: Template parsing and execution details

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

### Development Setup

```bash
# Clone repository
git clone https://github.com/tmc/mcp
cd mcp/cmd/mcp-docs

# Install dependencies
go mod download

# Run tests
go test ./...

# Build
go build
```

## License

Part of the MCP Go implementation project.

## Roadmap

- [ ] Plugin system for custom generators
- [ ] Multi-language support
- [ ] Advanced search features
- [ ] Real-time collaboration
- [ ] Cloud hosting integration
- [ ] Mobile app support
- [ ] AI-powered documentation
- [ ] Integration with popular tools