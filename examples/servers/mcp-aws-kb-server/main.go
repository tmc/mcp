package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime/types"
	"github.com/tmc/mcp"
)

const (
	ServerName    = "aws-kb-server"
	ServerVersion = "1.0.0"
)

type AWSKBServer struct {
	client *bedrockagentruntime.Client
}

func main() {
	// Redirect logs to stderr to keep stdout clean for the protocol
	log.SetOutput(os.Stderr)
	log.Println("Starting MCP AWS Knowledge Base Server...")

	// Create a context that can be canceled
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Initialize AWS config
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Unable to load AWS config: %v", err)
	}

	// Create AWS KB server instance
	awsKBServer := &AWSKBServer{
		client: bedrockagentruntime.NewFromConfig(cfg),
	}

	// Create MCP server
	server := mcp.NewServer(ServerName, ServerVersion,
		mcp.WithServerInstructions("AWS Knowledge Base retrieval server using Bedrock Agent Runtime for querying knowledge bases"),
	)

	// Register tools
	awsKBServer.registerTools(server)

	// Serve via stdio
	log.Println("Starting protocol server via stdio...")
	if err := server.Serve(ctx, nil); err != nil {
		if err != context.Canceled {
			log.Fatalf("Error serving: %v", err)
		}
		log.Println("Server terminated.")
	}
}

func (s *AWSKBServer) registerTools(server *mcp.Server) {
	// Query Knowledge Base tool
	queryTool := mcp.Tool{
		Name:        "query_knowledge_base",
		Description: "Query an AWS Knowledge Base using Bedrock Agent Runtime",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"knowledge_base_id": {
					"type": "string",
					"description": "The ID of the knowledge base to query"
				},
				"query": {
					"type": "string",
					"description": "The query text to search for in the knowledge base"
				},
				"max_results": {
					"type": "integer",
					"description": "Maximum number of results to return (default: 10)",
					"minimum": 1,
					"maximum": 100,
					"default": 10
				},
				"retrieval_filter": {
					"type": "object",
					"description": "Optional filter to apply to the retrieval",
					"properties": {
						"equals": {
							"type": "object",
							"description": "Equality filter"
						},
						"notEquals": {
							"type": "object", 
							"description": "Not equals filter"
						}
					}
				}
			},
			"required": ["knowledge_base_id", "query"]
		}`),
	}

	server.RegisterTool(queryTool, s.handleQueryKnowledgeBase)

	// Retrieve and Generate tool
	retrieveAndGenerateTool := mcp.Tool{
		Name:        "retrieve_and_generate",
		Description: "Retrieve from knowledge base and generate a response using the retrieved context",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"knowledge_base_id": {
					"type": "string",
					"description": "The ID of the knowledge base to query"
				},
				"input_text": {
					"type": "string",
					"description": "The input text/query for generation"
				},
				"model_arn": {
					"type": "string",
					"description": "ARN of the model to use for generation (optional)"
				},
				"retrieval_configuration": {
					"type": "object",
					"description": "Configuration for retrieval",
					"properties": {
						"vector_search_configuration": {
							"type": "object",
							"properties": {
								"number_of_results": {
									"type": "integer",
									"minimum": 1,
									"maximum": 100,
									"default": 10
								},
								"override_search_type": {
									"type": "string",
									"enum": ["HYBRID", "SEMANTIC"]
								}
							}
						}
					}
				}
			},
			"required": ["knowledge_base_id", "input_text"]
		}`),
	}

	server.RegisterTool(retrieveAndGenerateTool, s.handleRetrieveAndGenerate)
}

type QueryKnowledgeBaseRequest struct {
	KnowledgeBaseID string                 `json:"knowledge_base_id"`
	Query           string                 `json:"query"`
	MaxResults      *int32                 `json:"max_results,omitempty"`
	RetrievalFilter map[string]interface{} `json:"retrieval_filter,omitempty"`
}

func (s *AWSKBServer) handleQueryKnowledgeBase(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params QueryKnowledgeBaseRequest
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if params.KnowledgeBaseID == "" {
		return nil, fmt.Errorf("knowledge_base_id is required")
	}
	if params.Query == "" {
		return nil, fmt.Errorf("query is required")
	}

	// Set default max results
	maxResults := int32(10)
	if params.MaxResults != nil {
		maxResults = *params.MaxResults
	}

	// Build retrieve request
	input := &bedrockagentruntime.RetrieveInput{
		KnowledgeBaseId: aws.String(params.KnowledgeBaseID),
		RetrievalQuery: &types.KnowledgeBaseQuery{
			Text: aws.String(params.Query),
		},
		RetrievalConfiguration: &types.KnowledgeBaseRetrievalConfiguration{
			VectorSearchConfiguration: &types.KnowledgeBaseVectorSearchConfiguration{
				NumberOfResults: aws.Int32(maxResults),
			},
		},
	}

	// Apply retrieval filter if provided
	if len(params.RetrievalFilter) > 0 {
		// Note: This is a simplified filter implementation
		// In practice, you'd need to properly convert the filter map to AWS filter types
		log.Printf("Retrieval filter provided but not fully implemented: %+v", params.RetrievalFilter)
	}

	// Execute retrieval
	result, err := s.client.Retrieve(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to query knowledge base: %w", err)
	}

	// Format results
	var resultTexts []string
	for i, retrievalResult := range result.RetrievalResults {
		if retrievalResult.Content != nil && retrievalResult.Content.Text != nil {
			score := "N/A"
			if retrievalResult.Score != nil {
				score = fmt.Sprintf("%.3f", *retrievalResult.Score)
			}

			location := "N/A"
			if retrievalResult.Location != nil && retrievalResult.Location.S3Location != nil {
				location = fmt.Sprintf("s3://%s/%s",
					aws.ToString(retrievalResult.Location.S3Location.Uri),
					aws.ToString(retrievalResult.Location.S3Location.BucketOwnerAccountId))
			}

			resultTexts = append(resultTexts, fmt.Sprintf("Result %d (Score: %s, Location: %s):\n%s",
				i+1, score, location, aws.ToString(retrievalResult.Content.Text)))
		}
	}

	if len(resultTexts) == 0 {
		resultTexts = []string{"No results found for the query."}
	}

	responseText := fmt.Sprintf("Retrieved %d results from knowledge base %s:\n\n%s",
		len(result.RetrievalResults), params.KnowledgeBaseID, strings.Join(resultTexts, "\n\n"))

	return &mcp.CallToolResult{
		Content: []any{
			mcp.TextContent{
				Type: "text",
				Text: responseText,
			},
		},
	}, nil
}

type RetrieveAndGenerateRequest struct {
	KnowledgeBaseID        string                 `json:"knowledge_base_id"`
	InputText              string                 `json:"input_text"`
	ModelArn               string                 `json:"model_arn,omitempty"`
	RetrievalConfiguration map[string]interface{} `json:"retrieval_configuration,omitempty"`
}

func (s *AWSKBServer) handleRetrieveAndGenerate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params RetrieveAndGenerateRequest
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if params.KnowledgeBaseID == "" {
		return nil, fmt.Errorf("knowledge_base_id is required")
	}
	if params.InputText == "" {
		return nil, fmt.Errorf("input_text is required")
	}

	// Build retrieve and generate request
	input := &bedrockagentruntime.RetrieveAndGenerateInput{
		Input: &types.RetrieveAndGenerateInput{
			Text: aws.String(params.InputText),
		},
		RetrieveAndGenerateConfiguration: &types.RetrieveAndGenerateConfiguration{
			Type: types.RetrieveAndGenerateTypeKnowledgeBase,
			KnowledgeBaseConfiguration: &types.KnowledgeBaseRetrieveAndGenerateConfiguration{
				KnowledgeBaseId: aws.String(params.KnowledgeBaseID),
			},
		},
	}

	// Set model ARN if provided
	if params.ModelArn != "" {
		input.RetrieveAndGenerateConfiguration.KnowledgeBaseConfiguration.ModelArn = aws.String(params.ModelArn)
	}

	// Execute retrieve and generate
	result, err := s.client.RetrieveAndGenerate(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve and generate: %w", err)
	}

	// Format response
	responseText := "No response generated."
	if result.Output != nil && result.Output.Text != nil {
		responseText = aws.ToString(result.Output.Text)
	}

	// Add citation information if available
	var citations []string
	if result.Citations != nil {
		for i, citation := range result.Citations {
			if citation.GeneratedResponsePart != nil && citation.GeneratedResponsePart.TextResponsePart != nil {
				citationText := fmt.Sprintf("Citation %d: %s",
					i+1, aws.ToString(citation.GeneratedResponsePart.TextResponsePart.Text))
				if len(citation.RetrievedReferences) > 0 {
					var refs []string
					for _, ref := range citation.RetrievedReferences {
						if ref.Location != nil && ref.Location.S3Location != nil {
							refs = append(refs, aws.ToString(ref.Location.S3Location.Uri))
						}
					}
					if len(refs) > 0 {
						citationText += fmt.Sprintf(" (Sources: %s)", strings.Join(refs, ", "))
					}
				}
				citations = append(citations, citationText)
			}
		}
	}

	finalResponse := responseText
	if len(citations) > 0 {
		finalResponse += "\n\nCitations:\n" + strings.Join(citations, "\n")
	}

	return &mcp.CallToolResult{
		Content: []any{
			mcp.TextContent{
				Type: "text",
				Text: finalResponse,
			},
		},
	}, nil
}
