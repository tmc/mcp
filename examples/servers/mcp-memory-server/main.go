package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/modelcontextprotocol"
)

type Entity struct {
	Name         string   `json:"name"`
	EntityType   string   `json:"entityType"`
	Observations []string `json:"observations"`
}

type Relation struct {
	From         string `json:"from"`
	To           string `json:"to"`
	RelationType string `json:"relationType"`
}

type KnowledgeGraph struct {
	Entities  []Entity   `json:"entities"`
	Relations []Relation `json:"relations"`
}

type KnowledgeGraphManager struct {
	filePath string
}

func NewKnowledgeGraphManager() *KnowledgeGraphManager {
	memoryPath := os.Getenv("MEMORY_FILE_PATH")
	if memoryPath == "" {
		wd, _ := os.Getwd()
		memoryPath = filepath.Join(wd, "memory.json")
	} else if !filepath.IsAbs(memoryPath) {
		wd, _ := os.Getwd()
		memoryPath = filepath.Join(wd, memoryPath)
	}
	
	return &KnowledgeGraphManager{
		filePath: memoryPath,
	}
}

func (kg *KnowledgeGraphManager) loadGraph() (*KnowledgeGraph, error) {
	data, err := os.ReadFile(kg.filePath)
	if os.IsNotExist(err) {
		return &KnowledgeGraph{
			Entities:  []Entity{},
			Relations: []Relation{},
		}, nil
	}
	if err != nil {
		return nil, err
	}

	var graph KnowledgeGraph
	if err := json.Unmarshal(data, &graph); err != nil {
		return nil, err
	}

	return &graph, nil
}

func (kg *KnowledgeGraphManager) saveGraph(graph *KnowledgeGraph) error {
	data, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(kg.filePath, data, 0644)
}

func (kg *KnowledgeGraphManager) createEntities(entities []Entity) ([]Entity, error) {
	graph, err := kg.loadGraph()
	if err != nil {
		return nil, err
	}

	var newEntities []Entity
	for _, entity := range entities {
		exists := false
		for _, existing := range graph.Entities {
			if existing.Name == entity.Name {
				exists = true
				break
			}
		}
		if !exists {
			if entity.Observations == nil {
				entity.Observations = []string{}
			}
			newEntities = append(newEntities, entity)
			graph.Entities = append(graph.Entities, entity)
		}
	}

	if err := kg.saveGraph(graph); err != nil {
		return nil, err
	}
	return newEntities, nil
}

func (kg *KnowledgeGraphManager) createRelations(relations []Relation) ([]Relation, error) {
	graph, err := kg.loadGraph()
	if err != nil {
		return nil, err
	}

	var newRelations []Relation
	for _, relation := range relations {
		exists := false
		for _, existing := range graph.Relations {
			if existing.From == relation.From && existing.To == relation.To && existing.RelationType == relation.RelationType {
				exists = true
				break
			}
		}
		if !exists {
			newRelations = append(newRelations, relation)
			graph.Relations = append(graph.Relations, relation)
		}
	}

	if err := kg.saveGraph(graph); err != nil {
		return nil, err
	}
	return newRelations, nil
}

func (kg *KnowledgeGraphManager) addObservations(observations []map[string]interface{}) ([]map[string]interface{}, error) {
	graph, err := kg.loadGraph()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for _, obs := range observations {
		entityName, ok := obs["entityName"].(string)
		if !ok {
			continue
		}

		contents, ok := obs["contents"].([]interface{})
		if !ok {
			continue
		}

		// Find the entity
		var targetEntity *Entity
		for i := range graph.Entities {
			if graph.Entities[i].Name == entityName {
				targetEntity = &graph.Entities[i]
				break
			}
		}

		if targetEntity == nil {
			return nil, fmt.Errorf("entity with name %s not found", entityName)
		}

		var newObservations []string
		for _, content := range contents {
			if contentStr, ok := content.(string); ok {
				// Check if observation already exists
				exists := false
				for _, existing := range targetEntity.Observations {
					if existing == contentStr {
						exists = true
						break
					}
				}
				if !exists {
					targetEntity.Observations = append(targetEntity.Observations, contentStr)
					newObservations = append(newObservations, contentStr)
				}
			}
		}

		results = append(results, map[string]interface{}{
			"entityName":        entityName,
			"addedObservations": newObservations,
		})
	}

	if err := kg.saveGraph(graph); err != nil {
		return nil, err
	}
	return results, nil
}

func (kg *KnowledgeGraphManager) searchMemory(query string) (*KnowledgeGraph, error) {
	graph, err := kg.loadGraph()
	if err != nil {
		return nil, err
	}

	if query == "" {
		return graph, nil
	}

	queryLower := strings.ToLower(query)
	filteredGraph := &KnowledgeGraph{
		Entities:  []Entity{},
		Relations: []Relation{},
	}

	// Search entities
	for _, entity := range graph.Entities {
		if strings.Contains(strings.ToLower(entity.Name), queryLower) ||
			strings.Contains(strings.ToLower(entity.EntityType), queryLower) {
			filteredGraph.Entities = append(filteredGraph.Entities, entity)
		} else {
			// Check observations
			for _, obs := range entity.Observations {
				if strings.Contains(strings.ToLower(obs), queryLower) {
					filteredGraph.Entities = append(filteredGraph.Entities, entity)
					break
				}
			}
		}
	}

	// Search relations
	for _, relation := range graph.Relations {
		if strings.Contains(strings.ToLower(relation.From), queryLower) ||
			strings.Contains(strings.ToLower(relation.To), queryLower) ||
			strings.Contains(strings.ToLower(relation.RelationType), queryLower) {
			filteredGraph.Relations = append(filteredGraph.Relations, relation)
		}
	}

	return filteredGraph, nil
}

func main() {
	// Create server with name and version
	srv := mcp.NewServer("memory-server", "1.0.0")
	
	// Initialize knowledge graph manager
	kgManager := NewKnowledgeGraphManager()

	// Register create_entities tool
	srv.RegisterTool("create_entities", "Create new entities in the knowledge graph", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var entitiesRaw json.RawMessage
		var exists bool
		if entitiesRaw, exists = args["entities"]; !exists {
			return nil, fmt.Errorf("missing required argument: entities")
		}

		var entities []Entity
		if err := json.Unmarshal(entitiesRaw, &entities); err != nil {
			return nil, fmt.Errorf("invalid entities argument: %w", err)
		}

		newEntities, err := kgManager.createEntities(entities)
		if err != nil {
			return nil, fmt.Errorf("failed to create entities: %w", err)
		}

		result := map[string]interface{}{
			"newEntities": newEntities,
			"count":       len(newEntities),
		}

		responseJSON, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response: %w", err)
		}

		log.Printf("Created %d new entities", len(newEntities))

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: string(responseJSON),
				},
			},
		}, nil
	})

	// Register create_relations tool
	srv.RegisterTool("create_relations", "Create new relations between entities in the knowledge graph", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var relationsRaw json.RawMessage
		var exists bool
		if relationsRaw, exists = args["relations"]; !exists {
			return nil, fmt.Errorf("missing required argument: relations")
		}

		var relations []Relation
		if err := json.Unmarshal(relationsRaw, &relations); err != nil {
			return nil, fmt.Errorf("invalid relations argument: %w", err)
		}

		newRelations, err := kgManager.createRelations(relations)
		if err != nil {
			return nil, fmt.Errorf("failed to create relations: %w", err)
		}

		result := map[string]interface{}{
			"newRelations": newRelations,
			"count":        len(newRelations),
		}

		responseJSON, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response: %w", err)
		}

		log.Printf("Created %d new relations", len(newRelations))

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: string(responseJSON),
				},
			},
		}, nil
	})

	// Register add_observations tool
	srv.RegisterTool("add_observations", "Add observations to existing entities", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var observationsRaw json.RawMessage
		var exists bool
		if observationsRaw, exists = args["observations"]; !exists {
			return nil, fmt.Errorf("missing required argument: observations")
		}

		var observations []map[string]interface{}
		if err := json.Unmarshal(observationsRaw, &observations); err != nil {
			return nil, fmt.Errorf("invalid observations argument: %w", err)
		}

		results, err := kgManager.addObservations(observations)
		if err != nil {
			return nil, fmt.Errorf("failed to add observations: %w", err)
		}

		responseJSON, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response: %w", err)
		}

		log.Printf("Added observations to %d entities", len(results))

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: string(responseJSON),
				},
			},
		}, nil
	})

	// Register search_memory tool
	srv.RegisterTool("search_memory", "Search the knowledge graph for entities and relations", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		var query string
		if queryRaw, exists := args["query"]; exists {
			if err := json.Unmarshal(queryRaw, &query); err != nil {
				return nil, fmt.Errorf("invalid query argument: %w", err)
			}
		}

		results, err := kgManager.searchMemory(query)
		if err != nil {
			return nil, fmt.Errorf("failed to search memory: %w", err)
		}

		responseJSON, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response: %w", err)
		}

		log.Printf("Memory search for query: '%s', found %d entities and %d relations", 
			query, len(results.Entities), len(results.Relations))

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: string(responseJSON),
				},
			},
		}, nil
	})

	// Start server with stdio transport
	transport := mcp.StdioTransport{}
	log.Printf("Memory server running on stdio, using file: %s", kgManager.filePath)

	if err := srv.Serve(context.Background(), transport); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}