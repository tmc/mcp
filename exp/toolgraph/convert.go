package toolgraph

import (
	"fmt"
	"strings"
)

// ReactFlowNode represents a node in React Flow format
type ReactFlowNode struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Position Position               `json:"position"`
	Data     ReactFlowNodeData      `json:"data"`
}

// Position represents the position of a node
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// ReactFlowNodeData contains the data for a React Flow node
type ReactFlowNodeData struct {
	Label    string                 `json:"label"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ReactFlowEdge represents an edge in React Flow format
type ReactFlowEdge struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	Target   string `json:"target"`
	Type     string `json:"type"`
	Animated bool   `json:"animated,omitempty"`
	Label    string `json:"label,omitempty"`
}

// ReactFlowData contains the complete React Flow data
type ReactFlowData struct {
	Nodes []ReactFlowNode `json:"nodes"`
	Edges []ReactFlowEdge `json:"edges"`
}

// ConvertToReactFlow converts a graph to React Flow format
func ConvertToReactFlow(graph *Graph) *ReactFlowData {
	data := &ReactFlowData{
		Nodes: []ReactFlowNode{},
		Edges: []ReactFlowEdge{},
	}

	// Layout nodes using a simple algorithm
	layout := calculateLayout(graph)

	// Convert nodes
	for id, node := range graph.Nodes {
		pos := layout[id]
		rfNode := ReactFlowNode{
			ID:   id,
			Type: string(node.Type),
			Position: Position{
				X: pos.X,
				Y: pos.Y,
			},
			Data: ReactFlowNodeData{
				Label:    node.Label,
				Metadata: node.Metadata,
			},
		}
		data.Nodes = append(data.Nodes, rfNode)
	}

	// Convert edges
	for _, edge := range graph.Edges {
		rfEdge := ReactFlowEdge{
			ID:       edge.ID,
			Source:   edge.Source,
			Target:   edge.Target,
			Type:     "default",
			Animated: edge.Type == "executes",
			Label:    edge.Label,
		}
		data.Edges = append(data.Edges, rfEdge)
	}

	return data
}

// calculateLayout calculates node positions using a simple hierarchical layout
func calculateLayout(graph *Graph) map[string]Position {
	layout := make(map[string]Position)
	levels := calculateLevels(graph)
	
	// Group nodes by level
	nodesByLevel := make(map[int][]*Node)
	for id, node := range graph.Nodes {
		level := levels[id]
		nodesByLevel[level] = append(nodesByLevel[level], node)
	}
	
	// Position nodes
	ySpacing := 120.0
	xSpacing := 200.0
	
	for level, nodes := range nodesByLevel {
		y := float64(level) * ySpacing
		startX := -float64(len(nodes)-1) * xSpacing / 2
		
		for i, node := range nodes {
			x := startX + float64(i)*xSpacing
			layout[node.ID] = Position{X: x, Y: y}
		}
	}
	
	return layout
}

// calculateLevels calculates the hierarchical level of each node
func calculateLevels(graph *Graph) map[string]int {
	levels := make(map[string]int)
	visited := make(map[string]bool)
	
	// Build adjacency list
	adj := make(map[string][]string)
	for _, edge := range graph.Edges {
		adj[edge.Source] = append(adj[edge.Source], edge.Target)
	}
	
	// Find root nodes (nodes with no incoming edges)
	hasIncoming := make(map[string]bool)
	for _, edge := range graph.Edges {
		hasIncoming[edge.Target] = true
	}
	
	roots := []string{}
	for id := range graph.Nodes {
		if !hasIncoming[id] {
			roots = append(roots, id)
		}
	}
	
	// If no roots found, use the specified root or first node
	if len(roots) == 0 {
		if graph.Root != "" {
			roots = []string{graph.Root}
		} else {
			for id := range graph.Nodes {
				roots = []string{id}
				break
			}
		}
	}
	
	// BFS to assign levels
	queue := []string{}
	for _, root := range roots {
		queue = append(queue, root)
		levels[root] = 0
		visited[root] = true
	}
	
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		currentLevel := levels[current]
		
		for _, neighbor := range adj[current] {
			if !visited[neighbor] {
				visited[neighbor] = true
				levels[neighbor] = currentLevel + 1
				queue = append(queue, neighbor)
			}
		}
	}
	
	// Handle unvisited nodes
	for id := range graph.Nodes {
		if !visited[id] {
			levels[id] = 0
		}
	}
	
	return levels
}

// ConvertToDot converts a graph to Graphviz DOT format
func ConvertToDot(graph *Graph) string {
	var sb strings.Builder
	
	sb.WriteString("digraph ToolGraph {\n")
	sb.WriteString("  rankdir=TB;\n")
	sb.WriteString("  node [shape=box];\n\n")
	
	// Define node styles by type
	sb.WriteString("  // Node styles\n")
	sb.WriteString("  node [style=filled];\n")
	
	// Add nodes
	sb.WriteString("\n  // Nodes\n")
	for id, node := range graph.Nodes {
		color := getNodeColor(node.Type)
		shape := getNodeShape(node.Type)
		
		label := node.Label
		if node.Metadata != nil {
			if line, ok := node.Metadata["line"]; ok {
				label = fmt.Sprintf("%s\\n(line %v)", label, line)
			}
		}
		
		sb.WriteString(fmt.Sprintf("  \"%s\" [label=\"%s\", fillcolor=\"%s\", shape=%s];\n",
			id, label, color, shape))
	}
	
	// Add edges
	sb.WriteString("\n  // Edges\n")
	for _, edge := range graph.Edges {
		style := ""
		if edge.Type == "executes" {
			style = ", style=dashed"
		}
		
		label := ""
		if edge.Label != "" {
			label = fmt.Sprintf(", label=\"%s\"", edge.Label)
		}
		
		sb.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\" [label=\"%s\"%s%s];\n",
			edge.Source, edge.Target, edge.Type, label, style))
	}
	
	sb.WriteString("}\n")
	return sb.String()
}

func getNodeColor(nodeType NodeType) string {
	switch nodeType {
	case NodeTypeTest:
		return "#e3f2fd"
	case NodeTypeTool:
		return "#f3e5f5"
	case NodeTypeCommand:
		return "#e8f5e9"
	case NodeTypeFile:
		return "#fff3e0"
	case NodeTypePackage:
		return "#fce4ec"
	default:
		return "#ffffff"
	}
}

func getNodeShape(nodeType NodeType) string {
	switch nodeType {
	case NodeTypeTest:
		return "box"
	case NodeTypeTool:
		return "ellipse"
	case NodeTypeCommand:
		return "diamond"
	case NodeTypeFile:
		return "note"
	case NodeTypePackage:
		return "folder"
	default:
		return "box"
	}
}