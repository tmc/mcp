// depgraph transforms callgraph data into various dependency graph formats
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
)

var (
	inputFile   string
	outputFile  string
	format      string
	graphType   string
	direction   string
	includeMeta bool
	groupByType bool
	verbose     bool
)

func init() {
	flag.StringVar(&inputFile, "input", "", "Input callgraph file (JSON format)")
	flag.StringVar(&outputFile, "output", "", "Output file (default: stdout)")
	flag.StringVar(&format, "format", "digraph", "Output format: digraph, dot, json, matrix")
	flag.StringVar(&graphType, "type", "direct", "Graph type: direct, transitive, reverse")
	flag.StringVar(&direction, "direction", "test-to-program", "Direction: test-to-program, program-to-test, bidirectional")
	flag.BoolVar(&includeMeta, "include-meta", false, "Include metadata in output")
	flag.BoolVar(&groupByType, "group", false, "Group nodes by type")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "depgraph - Transform callgraph data into dependency graphs\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  depgraph [options] -input <callgraph.json>\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Create transitive dependency graph\n")
		fmt.Fprintf(os.Stderr, "  depgraph -input graph.json -type transitive\n")
		fmt.Fprintf(os.Stderr, "  \n")
		fmt.Fprintf(os.Stderr, "  # Create reverse dependency graph (program-to-test)\n")
		fmt.Fprintf(os.Stderr, "  depgraph -input graph.json -direction program-to-test\n")
		fmt.Fprintf(os.Stderr, "  \n")
		fmt.Fprintf(os.Stderr, "  # Output as adjacency matrix\n")
		fmt.Fprintf(os.Stderr, "  depgraph -input graph.json -format matrix\n")
		fmt.Fprintf(os.Stderr, "  \n")
		fmt.Fprintf(os.Stderr, "  # Create DOT file with metadata\n")
		fmt.Fprintf(os.Stderr, "  depgraph -input graph.json -format dot -include-meta\n")
	}
}

type CallGraph struct {
	Nodes map[string]Node `json:"nodes"`
	Edges []Edge          `json:"edges"`
}

type Node struct {
	ID       string                 `json:"ID"`
	Type     string                 `json:"Type"`
	Path     string                 `json:"Path"`
	Metadata map[string]interface{} `json:"Metadata,omitempty"`
}

type Edge struct {
	From     string                 `json:"From"`
	To       string                 `json:"To"`
	Type     string                 `json:"Type"`
	Metadata map[string]interface{} `json:"Metadata,omitempty"`
}

type DepGraph struct {
	Nodes       map[string]*DepNode
	Edges       map[string]map[string]*DepEdge
	NodesByType map[string][]string
}

type DepNode struct {
	ID         string
	Type       string
	Properties map[string]interface{}
	InDegree   int
	OutDegree  int
}

type DepEdge struct {
	From       string
	To         string
	Weight     int
	Properties map[string]interface{}
}

func main() {
	flag.Parse()

	if inputFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Load callgraph
	callgraph, err := loadCallGraph(inputFile)
	if err != nil {
		log.Fatalf("Error loading callgraph: %v", err)
	}

	// Transform to dependency graph
	depgraph := transformGraph(callgraph)

	// Setup output
	var out io.Writer = os.Stdout
	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			log.Fatalf("Error creating output file: %v", err)
		}
		defer f.Close()
		out = f
	}

	// Output the graph
	switch format {
	case "json":
		outputJSON(out, depgraph)
	case "dot":
		outputDot(out, depgraph)
	case "matrix":
		outputMatrix(out, depgraph)
	default:
		outputDigraph(out, depgraph)
	}
}

func loadCallGraph(filename string) (*CallGraph, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var callgraph CallGraph
	if err := json.Unmarshal(data, &callgraph); err != nil {
		return nil, err
	}

	return &callgraph, nil
}

func transformGraph(callgraph *CallGraph) *DepGraph {
	depgraph := &DepGraph{
		Nodes:       make(map[string]*DepNode),
		Edges:       make(map[string]map[string]*DepEdge),
		NodesByType: make(map[string][]string),
	}

	// Add nodes
	for id, node := range callgraph.Nodes {
		depNode := &DepNode{
			ID:         id,
			Type:       node.Type,
			Properties: node.Metadata,
		}
		depgraph.Nodes[id] = depNode
		depgraph.NodesByType[node.Type] = append(depgraph.NodesByType[node.Type], id)
	}

	// Add edges based on direction
	for _, edge := range callgraph.Edges {
		switch direction {
		case "program-to-test":
			addEdge(depgraph, edge.To, edge.From, edge)
		case "bidirectional":
			addEdge(depgraph, edge.From, edge.To, edge)
			addEdge(depgraph, edge.To, edge.From, edge)
		default: // test-to-program
			addEdge(depgraph, edge.From, edge.To, edge)
		}
	}

	// Calculate transitive closure if requested
	if graphType == "transitive" {
		calculateTransitiveClosure(depgraph)
	}

	// Update degrees
	for _, node := range depgraph.Nodes {
		if edges, ok := depgraph.Edges[node.ID]; ok {
			node.OutDegree = len(edges)
		}
		for _, edges := range depgraph.Edges {
			if _, ok := edges[node.ID]; ok {
				node.InDegree++
			}
		}
	}

	return depgraph
}

func addEdge(depgraph *DepGraph, from, to string, edge Edge) {
	if depgraph.Edges[from] == nil {
		depgraph.Edges[from] = make(map[string]*DepEdge)
	}

	if existing, ok := depgraph.Edges[from][to]; ok {
		existing.Weight++
	} else {
		depgraph.Edges[from][to] = &DepEdge{
			From:       from,
			To:         to,
			Weight:     1,
			Properties: edge.Metadata,
		}
	}
}

func calculateTransitiveClosure(depgraph *DepGraph) {
	// Floyd-Warshall algorithm
	nodes := make([]string, 0, len(depgraph.Nodes))
	for id := range depgraph.Nodes {
		nodes = append(nodes, id)
	}

	for _, k := range nodes {
		for _, i := range nodes {
			for _, j := range nodes {
				if hasPath(depgraph, i, k) && hasPath(depgraph, k, j) {
					if depgraph.Edges[i] == nil {
						depgraph.Edges[i] = make(map[string]*DepEdge)
					}
					if _, ok := depgraph.Edges[i][j]; !ok {
						depgraph.Edges[i][j] = &DepEdge{
							From:   i,
							To:     j,
							Weight: 0,
							Properties: map[string]interface{}{
								"transitive": true,
							},
						}
					}
				}
			}
		}
	}
}

func hasPath(depgraph *DepGraph, from, to string) bool {
	if edges, ok := depgraph.Edges[from]; ok {
		_, exists := edges[to]
		return exists
	}
	return false
}

func outputDigraph(w io.Writer, depgraph *DepGraph) {
	// Output in digraph format (simple adjacency list)
	for from, edges := range depgraph.Edges {
		for to := range edges {
			fmt.Fprintf(w, "%s %s\n", from, to)
		}
	}
}

func outputDot(w io.Writer, depgraph *DepGraph) {
	fmt.Fprintf(w, "digraph dependencies {\n")
	fmt.Fprintf(w, "  rankdir=LR;\n")
	fmt.Fprintf(w, "  compound=true;\n")
	fmt.Fprintf(w, "  node [shape=box];\n\n")

	if groupByType {
		// Group nodes by type
		for nodeType, nodes := range depgraph.NodesByType {
			fmt.Fprintf(w, "  subgraph cluster_%s {\n", nodeType)
			fmt.Fprintf(w, "    label=\"%s\";\n", strings.Title(nodeType))
			fmt.Fprintf(w, "    style=filled;\n")
			if nodeType == "test" {
				fmt.Fprintf(w, "    fillcolor=lightblue;\n")
			} else {
				fmt.Fprintf(w, "    fillcolor=lightgreen;\n")
			}
			for _, node := range nodes {
				fmt.Fprintf(w, "    \"%s\";\n", node)
			}
			fmt.Fprintf(w, "  }\n\n")
		}
	} else {
		// Individual nodes
		for id, node := range depgraph.Nodes {
			attrs := []string{}
			if node.Type == "test" {
				attrs = append(attrs, "style=filled,fillcolor=lightblue")
			} else if node.Type == "program" {
				attrs = append(attrs, "style=filled,fillcolor=lightgreen")
			}

			if includeMeta {
				attrs = append(attrs, fmt.Sprintf("tooltip=\"In: %d, Out: %d\"", node.InDegree, node.OutDegree))
			}

			if len(attrs) > 0 {
				fmt.Fprintf(w, "  \"%s\" [%s];\n", id, strings.Join(attrs, ","))
			} else {
				fmt.Fprintf(w, "  \"%s\";\n", id)
			}
		}
		fmt.Fprintf(w, "\n")
	}

	// Define edges
	for from, edges := range depgraph.Edges {
		for to, edge := range edges {
			attrs := []string{}

			if edge.Weight > 1 {
				attrs = append(attrs, fmt.Sprintf("label=\"%d\"", edge.Weight))
				attrs = append(attrs, "penwidth=2")
			}

			if trans, ok := edge.Properties["transitive"].(bool); ok && trans {
				attrs = append(attrs, "style=dashed")
			}

			if includeMeta {
				if line, ok := edge.Properties["line"].(float64); ok {
					attrs = append(attrs, fmt.Sprintf("tooltip=\"Line: %d\"", int(line)))
				}
			}

			if len(attrs) > 0 {
				fmt.Fprintf(w, "  \"%s\" -> \"%s\" [%s];\n", from, to, strings.Join(attrs, ","))
			} else {
				fmt.Fprintf(w, "  \"%s\" -> \"%s\";\n", from, to)
			}
		}
	}

	fmt.Fprintf(w, "}\n")
}

func outputMatrix(w io.Writer, depgraph *DepGraph) {
	// Get sorted list of all nodes
	nodes := make([]string, 0, len(depgraph.Nodes))
	for id := range depgraph.Nodes {
		nodes = append(nodes, id)
	}
	sort.Strings(nodes)

	// Header
	fmt.Fprintf(w, "           ")
	for _, node := range nodes {
		fmt.Fprintf(w, "%-10s ", node)
	}
	fmt.Fprintf(w, "\n")

	// Rows
	for _, from := range nodes {
		fmt.Fprintf(w, "%-10s ", from)
		for _, to := range nodes {
			weight := 0
			if edges, ok := depgraph.Edges[from]; ok {
				if edge, ok := edges[to]; ok {
					weight = edge.Weight
				}
			}
			fmt.Fprintf(w, "%-10d ", weight)
		}
		fmt.Fprintf(w, "\n")
	}
}

func outputJSON(w io.Writer, depgraph *DepGraph) {
	// Convert edges map to list for JSON
	edgeList := []DepEdge{}
	for _, edges := range depgraph.Edges {
		for _, edge := range edges {
			edgeList = append(edgeList, *edge)
		}
	}

	data := map[string]interface{}{
		"nodes":       depgraph.Nodes,
		"edges":       edgeList,
		"nodesByType": depgraph.NodesByType,
		"statistics": map[string]interface{}{
			"nodeCount":    len(depgraph.Nodes),
			"edgeCount":    len(edgeList),
			"testCount":    len(depgraph.NodesByType["test"]),
			"programCount": len(depgraph.NodesByType["program"]),
		},
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(data)
}
