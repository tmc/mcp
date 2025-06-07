// digraph-compat provides a simple implementation of common digraph operations
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: digraph-compat <command> [args...]\n")
		fmt.Fprintf(os.Stderr, "Commands: nodes, sources, sinks, successors, predecessors\n")
		os.Exit(1)
	}

	command := args[0]

	// Read graph from stdin
	scanner := bufio.NewScanner(os.Stdin)
	edges := [][]string{}
	nodes := make(map[string]bool)

	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) >= 2 {
			edges = append(edges, parts[:2])
			nodes[parts[0]] = true
			nodes[parts[1]] = true
		}
	}

	switch command {
	case "nodes":
		for node := range nodes {
			fmt.Println(node)
		}

	case "sources":
		sources := make(map[string]bool)
		for node := range nodes {
			sources[node] = true
		}
		for _, edge := range edges {
			delete(sources, edge[1])
		}
		for source := range sources {
			fmt.Println(source)
		}

	case "sinks":
		sinks := make(map[string]bool)
		for node := range nodes {
			sinks[node] = true
		}
		for _, edge := range edges {
			delete(sinks, edge[0])
		}
		for sink := range sinks {
			fmt.Println(sink)
		}

	case "successors":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: digraph-compat successors <node>\n")
			os.Exit(1)
		}
		node := args[1]
		for _, edge := range edges {
			if edge[0] == node {
				fmt.Println(edge[1])
			}
		}

	case "predecessors":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: digraph-compat predecessors <node>\n")
			os.Exit(1)
		}
		node := args[1]
		for _, edge := range edges {
			if edge[1] == node {
				fmt.Println(edge[0])
			}
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		os.Exit(1)
	}
}
