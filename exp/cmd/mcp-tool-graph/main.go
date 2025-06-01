package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/tmc/mcp/exp/toolgraph"
)

func main() {
	var (
		target      = flag.String("target", "", "Target test file or directory to analyze")
		output      = flag.String("output", "tool-graph", "Output directory for visualization")
		format      = flag.String("format", "react", "Output format: react, dot, json")
		maxDepth    = flag.Int("max-depth", 10, "Maximum depth to traverse")
		includeStd  = flag.Bool("include-std", false, "Include standard library tools")
		server      = flag.Bool("server", false, "Start a web server for the visualization")
		port        = flag.Int("port", 8080, "Port for the web server")
		verbose     = flag.Bool("verbose", false, "Verbose output")
	)
	flag.Parse()

	if *target == "" {
		log.Fatal("Please provide a target test file or directory via -target")
	}

	// Create output directory
	if err := os.MkdirAll(*output, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Create graph builder
	builder := toolgraph.NewGraphBuilder(toolgraph.GraphOptions{
		MaxDepth:   *maxDepth,
		IncludeStd: *includeStd,
		Verbose:    *verbose,
	})

	// Build the graph
	graph, err := builder.BuildFromTarget(*target)
	if err != nil {
		log.Fatalf("Failed to build graph: %v", err)
	}

	// Generate output based on format
	switch *format {
	case "react":
		if err := generateReactFlow(graph, *output); err != nil {
			log.Fatalf("Failed to generate React Flow: %v", err)
		}
		fmt.Printf("React Flow visualization generated in %s\n", *output)

		if *server {
			fmt.Printf("Starting web server on http://localhost:%d\n", *port)
			if err := serveVisualization(*output, *port); err != nil {
				log.Fatalf("Failed to start server: %v", err)
			}
		}

	case "dot":
		if err := generateDot(graph, filepath.Join(*output, "graph.dot")); err != nil {
			log.Fatalf("Failed to generate DOT: %v", err)
		}
		fmt.Printf("DOT file generated: %s/graph.dot\n", *output)

	case "json":
		if err := generateJSON(graph, filepath.Join(*output, "graph.json")); err != nil {
			log.Fatalf("Failed to generate JSON: %v", err)
		}
		fmt.Printf("JSON file generated: %s/graph.json\n", *output)

	default:
		log.Fatalf("Unknown format: %s", *format)
	}

	// Print summary
	fmt.Printf("\nGraph Summary:\n")
	fmt.Printf("- Nodes: %d\n", len(graph.Nodes))
	fmt.Printf("- Edges: %d\n", len(graph.Edges))
	fmt.Printf("- Root: %s\n", graph.Root)
	fmt.Printf("- Max depth: %d\n", graph.MaxDepth)
}

func generateReactFlow(graph *toolgraph.Graph, outputDir string) error {
	// Convert to React Flow format
	reactData := toolgraph.ConvertToReactFlow(graph)

	// Save the data
	dataPath := filepath.Join(outputDir, "graph-data.json")
	data, err := json.MarshalIndent(reactData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := os.WriteFile(dataPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	// Generate HTML visualization
	htmlPath := filepath.Join(outputDir, "index.html")
	if err := os.WriteFile(htmlPath, []byte(getReactFlowHTML()), 0644); err != nil {
		return fmt.Errorf("failed to write HTML: %w", err)
	}

	// Generate CSS
	cssPath := filepath.Join(outputDir, "styles.css")
	if err := os.WriteFile(cssPath, []byte(getReactFlowCSS()), 0644); err != nil {
		return fmt.Errorf("failed to write CSS: %w", err)
	}

	// Generate JS
	jsPath := filepath.Join(outputDir, "app.js")
	if err := os.WriteFile(jsPath, []byte(getReactFlowJS()), 0644); err != nil {
		return fmt.Errorf("failed to write JS: %w", err)
	}

	return nil
}

func generateDot(graph *toolgraph.Graph, outputPath string) error {
	dot := toolgraph.ConvertToDot(graph)
	return os.WriteFile(outputPath, []byte(dot), 0644)
}

func generateJSON(graph *toolgraph.Graph, outputPath string) error {
	data, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, data, 0644)
}

func serveVisualization(dir string, port int) error {
	http.Handle("/", http.FileServer(http.Dir(dir)))
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func getReactFlowHTML() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>MCP Tool Graph Visualization</title>
    <link rel="stylesheet" href="styles.css">
    <script crossorigin src="https://unpkg.com/react@18/umd/react.development.js"></script>
    <script crossorigin src="https://unpkg.com/react-dom@18/umd/react-dom.development.js"></script>
    <script crossorigin src="https://unpkg.com/reactflow@11/dist/umd/index.js"></script>
    <link href="https://unpkg.com/reactflow@11/dist/style.css" rel="stylesheet">
</head>
<body>
    <div id="root"></div>
    <script src="app.js"></script>
</body>
</html>`
}

func getReactFlowCSS() string {
	return `body {
    margin: 0;
    padding: 0;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

#root {
    width: 100vw;
    height: 100vh;
}

.tool-graph-container {
    width: 100%;
    height: 100%;
}

.react-flow__node {
    font-size: 12px;
}

.react-flow__node-test {
    background: #e3f2fd;
    border: 1px solid #1976d2;
    border-radius: 8px;
    padding: 10px;
}

.react-flow__node-tool {
    background: #f3e5f5;
    border: 1px solid #7b1fa2;
    border-radius: 8px;
    padding: 10px;
}

.react-flow__node-command {
    background: #e8f5e9;
    border: 1px solid #388e3c;
    border-radius: 8px;
    padding: 10px;
}

.react-flow__node-file {
    background: #fff3e0;
    border: 1px solid #f57c00;
    border-radius: 8px;
    padding: 10px;
}

.react-flow__edge-path {
    stroke: #b1b1b7;
}

.react-flow__edge-animated {
    animation: dashdraw 0.5s linear infinite;
}

@keyframes dashdraw {
    to {
        stroke-dashoffset: -10;
    }
}

.controls {
    position: absolute;
    top: 10px;
    left: 10px;
    z-index: 4;
    background: white;
    padding: 10px;
    border-radius: 8px;
    box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.controls button {
    margin: 0 5px;
    padding: 5px 10px;
    border: 1px solid #ddd;
    border-radius: 4px;
    background: white;
    cursor: pointer;
}

.controls button:hover {
    background: #f5f5f5;
}

.node-info {
    position: absolute;
    bottom: 10px;
    left: 10px;
    background: white;
    padding: 10px;
    border-radius: 8px;
    box-shadow: 0 2px 4px rgba(0,0,0,0.1);
    max-width: 300px;
}

.legend {
    position: absolute;
    top: 10px;
    right: 10px;
    background: white;
    padding: 10px;
    border-radius: 8px;
    box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.legend-item {
    display: flex;
    align-items: center;
    margin: 5px 0;
}

.legend-color {
    width: 20px;
    height: 20px;
    border-radius: 4px;
    margin-right: 10px;
    border: 1px solid #ddd;
}`
}

func getReactFlowJS() string {
	return `const { ReactFlow, Controls, Background, MiniMap, useNodesState, useEdgesState } = ReactFlowLibrary;

function ToolGraphFlow() {
    const [nodes, setNodes, onNodesChange] = useNodesState([]);
    const [edges, setEdges, onEdgesChange] = useEdgesState([]);
    const [selectedNode, setSelectedNode] = React.useState(null);
    const [loading, setLoading] = React.useState(true);

    React.useEffect(() => {
        // Load graph data
        fetch('graph-data.json')
            .then(res => res.json())
            .then(data => {
                setNodes(data.nodes);
                setEdges(data.edges);
                setLoading(false);
            })
            .catch(err => {
                console.error('Failed to load graph data:', err);
                setLoading(false);
            });
    }, []);

    const onNodeClick = React.useCallback((event, node) => {
        setSelectedNode(node);
    }, []);

    const nodeTypes = React.useMemo(() => ({
        test: TestNode,
        tool: ToolNode,
        command: CommandNode,
        file: FileNode,
    }), []);

    if (loading) {
        return React.createElement('div', { className: 'loading' }, 'Loading graph...');
    }

    return React.createElement('div', { className: 'tool-graph-container' },
        React.createElement(ReactFlow, {
            nodes: nodes,
            edges: edges,
            onNodesChange: onNodesChange,
            onEdgesChange: onEdgesChange,
            onNodeClick: onNodeClick,
            nodeTypes: nodeTypes,
            fitView: true,
            attributionPosition: 'bottom-left'
        },
            React.createElement(Controls),
            React.createElement(Background, { variant: 'dots', gap: 12, size: 1 }),
            React.createElement(MiniMap, {
                nodeStrokeColor: (n) => {
                    switch (n.type) {
                        case 'test': return '#1976d2';
                        case 'tool': return '#7b1fa2';
                        case 'command': return '#388e3c';
                        case 'file': return '#f57c00';
                        default: return '#000';
                    }
                },
                nodeColor: (n) => {
                    switch (n.type) {
                        case 'test': return '#e3f2fd';
                        case 'tool': return '#f3e5f5';
                        case 'command': return '#e8f5e9';
                        case 'file': return '#fff3e0';
                        default: return '#fff';
                    }
                }
            })
        ),
        React.createElement(Legend),
        selectedNode && React.createElement(NodeInfo, { node: selectedNode })
    );
}

function TestNode({ data }) {
    return React.createElement('div', { className: 'test-node' },
        React.createElement('div', { className: 'node-title' }, '🧪 Test'),
        React.createElement('div', { className: 'node-label' }, data.label)
    );
}

function ToolNode({ data }) {
    return React.createElement('div', { className: 'tool-node' },
        React.createElement('div', { className: 'node-title' }, '🔧 Tool'),
        React.createElement('div', { className: 'node-label' }, data.label)
    );
}

function CommandNode({ data }) {
    return React.createElement('div', { className: 'command-node' },
        React.createElement('div', { className: 'node-title' }, '⚡ Command'),
        React.createElement('div', { className: 'node-label' }, data.label)
    );
}

function FileNode({ data }) {
    return React.createElement('div', { className: 'file-node' },
        React.createElement('div', { className: 'node-title' }, '📄 File'),
        React.createElement('div', { className: 'node-label' }, data.label)
    );
}

function Legend() {
    const items = [
        { type: 'test', label: 'Test', color: '#e3f2fd', border: '#1976d2' },
        { type: 'tool', label: 'Tool', color: '#f3e5f5', border: '#7b1fa2' },
        { type: 'command', label: 'Command', color: '#e8f5e9', border: '#388e3c' },
        { type: 'file', label: 'File', color: '#fff3e0', border: '#f57c00' },
    ];

    return React.createElement('div', { className: 'legend' },
        React.createElement('h4', null, 'Legend'),
        items.map(item =>
            React.createElement('div', { key: item.type, className: 'legend-item' },
                React.createElement('div', {
                    className: 'legend-color',
                    style: { backgroundColor: item.color, borderColor: item.border }
                }),
                React.createElement('span', null, item.label)
            )
        )
    );
}

function NodeInfo({ node }) {
    return React.createElement('div', { className: 'node-info' },
        React.createElement('h4', null, 'Node Info'),
        React.createElement('p', null, React.createElement('strong', null, 'ID: '), node.id),
        React.createElement('p', null, React.createElement('strong', null, 'Type: '), node.type),
        React.createElement('p', null, React.createElement('strong', null, 'Label: '), node.data.label),
        node.data.metadata && Object.entries(node.data.metadata).map(([key, value]) =>
            React.createElement('p', { key: key },
                React.createElement('strong', null, key + ': '), String(value)
            )
        )
    );
}

// Mount the app
const container = document.getElementById('root');
const root = ReactDOM.createRoot(container);
root.render(React.createElement(ToolGraphFlow));`
}

// Add missing import
import (
	"net/http"
)