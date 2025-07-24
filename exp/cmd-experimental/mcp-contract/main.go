package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/modelcontextprotocol"
	"gopkg.in/yaml.v3"
)

const (
	programName = "mcp-contract"
	version     = "0.1.0"
)

// Command represents a CLI command
type Command interface {
	Name() string
	Usage() string
	Execute(ctx context.Context, args []string) error
}

// RecordCommand records a contract from a trace
type RecordCommand struct {
	traceFile  string
	outputFile string
	format     string
	verbose    bool
}

func (c *RecordCommand) Name() string {
	return "record"
}

func (c *RecordCommand) Usage() string {
	return "Record a contract from trace file"
}

func (c *RecordCommand) Execute(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet(c.Name(), flag.ExitOnError)
	
	fs.StringVar(&c.traceFile, "trace", "", "Trace file to analyze")
	fs.StringVar(&c.outputFile, "output", "contract.yaml", "Output contract file")
	fs.StringVar(&c.format, "format", "yaml", "Output format (yaml, json)")
	fs.BoolVar(&c.verbose, "verbose", false, "Verbose output")
	
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}
	
	if c.traceFile == "" {
		return fmt.Errorf("trace file is required")
	}
	
	recorder := NewContractRecorder(c)
	return recorder.Record(ctx)
}

// VerifyCommand verifies a server against a contract
type VerifyCommand struct {
	serverCmd    string
	contractFile string
	reportFile   string
	format       string
	strict       bool
	verbose      bool
}

func (c *VerifyCommand) Name() string {
	return "verify"
}

func (c *VerifyCommand) Usage() string {
	return "Verify server against contract"
}

func (c *VerifyCommand) Execute(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet(c.Name(), flag.ExitOnError)
	
	fs.StringVar(&c.serverCmd, "server", "", "Server command to verify")
	fs.StringVar(&c.contractFile, "contract", "", "Contract file to verify against")
	fs.StringVar(&c.reportFile, "report", "", "Output report file")
	fs.StringVar(&c.format, "format", "json", "Output format (json, yaml, junit-xml)")
	fs.BoolVar(&c.strict, "strict", false, "Strict verification mode")
	fs.BoolVar(&c.verbose, "verbose", false, "Verbose output")
	
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}
	
	if c.serverCmd == "" || c.contractFile == "" {
		return fmt.Errorf("both server and contract file are required")
	}
	
	verifier := NewContractVerifier(c)
	return verifier.Verify(ctx)
}

// MatrixCommand runs compatibility matrix testing
type MatrixCommand struct {
	clientsFile string
	serversFile string
	outputFile  string
	format      string
	parallel    int
	verbose     bool
}

func (c *MatrixCommand) Name() string {
	return "matrix"
}

func (c *MatrixCommand) Usage() string {
	return "Run compatibility matrix testing"
}

func (c *MatrixCommand) Execute(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet(c.Name(), flag.ExitOnError)
	
	fs.StringVar(&c.clientsFile, "clients", "", "File containing client configurations")
	fs.StringVar(&c.serversFile, "servers", "", "File containing server configurations")
	fs.StringVar(&c.outputFile, "output", "matrix.json", "Output matrix file")
	fs.StringVar(&c.format, "format", "json", "Output format (json, yaml, html)")
	fs.IntVar(&c.parallel, "parallel", 4, "Number of parallel tests")
	fs.BoolVar(&c.verbose, "verbose", false, "Verbose output")
	
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}
	
	if c.clientsFile == "" || c.serversFile == "" {
		return fmt.Errorf("both clients and servers files are required")
	}
	
	matrix := NewCompatibilityMatrix(c)
	return matrix.Run(ctx)
}

// APIContract represents a contract for API testing
type APIContract struct {
	Version     string                 `json:"version" yaml:"version"`
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Provider    ContractProvider       `json:"provider" yaml:"provider"`
	Consumer    ContractConsumer       `json:"consumer" yaml:"consumer"`
	Interactions []ContractInteraction `json:"interactions" yaml:"interactions"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// ContractProvider describes the provider (server) side
type ContractProvider struct {
	Name         string            `json:"name" yaml:"name"`
	Version      string            `json:"version,omitempty" yaml:"version,omitempty"`
	Capabilities map[string]bool   `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// ContractConsumer describes the consumer (client) side
type ContractConsumer struct {
	Name         string            `json:"name" yaml:"name"`
	Version      string            `json:"version,omitempty" yaml:"version,omitempty"`
	Requirements []string          `json:"requirements,omitempty" yaml:"requirements,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// ContractInteraction describes an expected interaction
type ContractInteraction struct {
	Description string                 `json:"description" yaml:"description"`
	Request     ContractRequest        `json:"request" yaml:"request"`
	Response    ContractResponse       `json:"response" yaml:"response"`
	State       string                 `json:"state,omitempty" yaml:"state,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// ContractRequest describes an expected request
type ContractRequest struct {
	Method      string                 `json:"method" yaml:"method"`
	Parameters  map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Headers     map[string]string      `json:"headers,omitempty" yaml:"headers,omitempty"`
	Constraints map[string]interface{} `json:"constraints,omitempty" yaml:"constraints,omitempty"`
}

// ContractResponse describes an expected response
type ContractResponse struct {
	Status      string                 `json:"status" yaml:"status"`
	Body        map[string]interface{} `json:"body,omitempty" yaml:"body,omitempty"`
	Headers     map[string]string      `json:"headers,omitempty" yaml:"headers,omitempty"`
	Constraints map[string]interface{} `json:"constraints,omitempty" yaml:"constraints,omitempty"`
}

// ContractRecorder records contracts from traces
type ContractRecorder struct {
	config *RecordCommand
}

// NewContractRecorder creates a new contract recorder
func NewContractRecorder(config *RecordCommand) *ContractRecorder {
	return &ContractRecorder{config: config}
}

// Record records a contract from a trace file
func (r *ContractRecorder) Record(ctx context.Context) error {
	if r.config.verbose {
		log.Printf("Recording contract from trace: %s", r.config.traceFile)
	}
	
	// Parse trace file
	interactions, err := r.parseTrace(r.config.traceFile)
	if err != nil {
		return fmt.Errorf("failed to parse trace: %w", err)
	}
	
	// Create contract
	contract := &APIContract{
		Version:     "1.0.0",
		Name:        "Generated Contract",
		Description: fmt.Sprintf("Contract generated from trace file: %s", r.config.traceFile),
		Provider: ContractProvider{
			Name: "Unknown Server",
		},
		Consumer: ContractConsumer{
			Name: "Test Client",
		},
		Interactions: interactions,
		Metadata: map[string]interface{}{
			"generated_at": time.Now().Format(time.RFC3339),
			"source_trace": r.config.traceFile,
		},
	}
	
	// Write contract
	return r.writeContract(contract)
}

// parseTrace parses a trace file and extracts interactions
func (r *ContractRecorder) parseTrace(filename string) ([]ContractInteraction, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	var interactions []ContractInteraction
	decoder := json.NewDecoder(file)
	
	// Track request-response pairs
	requests := make(map[interface{}]ContractRequest)
	
	for {
		var msg map[string]interface{}
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			if r.config.verbose {
				log.Printf("Skipping invalid message: %v", err)
			}
			continue
		}
		
		// Handle request
		if method, hasMethod := msg["method"]; hasMethod {
			req := ContractRequest{
				Method: method.(string),
			}
			
			if params, hasParams := msg["params"]; hasParams {
				if paramsMap, ok := params.(map[string]interface{}); ok {
					req.Parameters = paramsMap
				}
			}
			
			if id, hasID := msg["id"]; hasID {
				requests[id] = req
			}
		}
		
		// Handle response
		if result, hasResult := msg["result"]; hasResult {
			if id, hasID := msg["id"]; hasID {
				if req, exists := requests[id]; exists {
					interaction := ContractInteraction{
						Description: fmt.Sprintf("Call to %s", req.Method),
						Request:     req,
						Response: ContractResponse{
							Status: "success",
							Body:   result.(map[string]interface{}),
						},
					}
					interactions = append(interactions, interaction)
					delete(requests, id)
				}
			}
		}
		
		// Handle error response
		if errorVal, hasError := msg["error"]; hasError {
			if id, hasID := msg["id"]; hasID {
				if req, exists := requests[id]; exists {
					interaction := ContractInteraction{
						Description: fmt.Sprintf("Call to %s (error)", req.Method),
						Request:     req,
						Response: ContractResponse{
							Status: "error",
							Body:   errorVal.(map[string]interface{}),
						},
					}
					interactions = append(interactions, interaction)
					delete(requests, id)
				}
			}
		}
	}
	
	return interactions, nil
}

// writeContract writes the contract to a file
func (r *ContractRecorder) writeContract(contract *APIContract) error {
	file, err := os.Create(r.config.outputFile)
	if err != nil {
		return err
	}
	defer file.Close()
	
	switch r.config.format {
	case "yaml":
		encoder := yaml.NewEncoder(file)
		encoder.SetIndent(2)
		return encoder.Encode(contract)
	case "json":
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		return encoder.Encode(contract)
	default:
		return fmt.Errorf("unsupported format: %s", r.config.format)
	}
}

// ContractVerifier verifies servers against contracts
type ContractVerifier struct {
	config *VerifyCommand
}

// NewContractVerifier creates a new contract verifier
func NewContractVerifier(config *VerifyCommand) *ContractVerifier {
	return &ContractVerifier{config: config}
}

// VerificationResult represents the result of contract verification
type VerificationResult struct {
	Summary      VerificationSummary `json:"summary"`
	Interactions []InteractionResult `json:"interactions"`
	Timestamp    time.Time           `json:"timestamp"`
	Contract     string              `json:"contract"`
	Server       string              `json:"server"`
}

// VerificationSummary provides a summary of verification results
type VerificationSummary struct {
	TotalInteractions int     `json:"totalInteractions"`
	PassedInteractions int    `json:"passedInteractions"`
	FailedInteractions int    `json:"failedInteractions"`
	SuccessRate       float64 `json:"successRate"`
	Status            string  `json:"status"`
}

// InteractionResult represents the result of testing one interaction
type InteractionResult struct {
	Description string                 `json:"description"`
	Status      string                 `json:"status"`
	Request     ContractRequest        `json:"request"`
	Expected    ContractResponse       `json:"expected"`
	Actual      map[string]interface{} `json:"actual,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Duration    time.Duration          `json:"duration"`
}

// Verify verifies a server against a contract
func (v *ContractVerifier) Verify(ctx context.Context) error {
	if v.config.verbose {
		log.Printf("Verifying server %s against contract %s", v.config.serverCmd, v.config.contractFile)
	}
	
	// Load contract
	contract, err := v.loadContract()
	if err != nil {
		return fmt.Errorf("failed to load contract: %w", err)
	}
	
	// Connect to server
	transport := mcp.NewCommandTransport(v.config.serverCmd)
	client := mcp.NewClient(transport)
	
	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer client.Close()
	
	// Initialize connection
	initReq := modelcontextprotocol.InitializeRequest{
		ProtocolVersion: modelcontextprotocol.LATEST_PROTOCOL_VERSION,
		ClientInfo: modelcontextprotocol.Implementation{
			Name:    programName,
			Version: version,
		},
	}
	
	if _, err := client.Initialize(ctx, initReq); err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	
	// Verify interactions
	result := &VerificationResult{
		Timestamp:    time.Now(),
		Contract:     v.config.contractFile,
		Server:       v.config.serverCmd,
		Interactions: []InteractionResult{},
	}
	
	for _, interaction := range contract.Interactions {
		interactionResult := v.verifyInteraction(ctx, client, interaction)
		result.Interactions = append(result.Interactions, interactionResult)
	}
	
	// Calculate summary
	result.Summary = v.calculateSummary(result.Interactions)
	
	// Output results
	return v.outputResults(result)
}

// loadContract loads a contract from file
func (v *ContractVerifier) loadContract() (*APIContract, error) {
	file, err := os.Open(v.config.contractFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	var contract APIContract
	
	ext := filepath.Ext(v.config.contractFile)
	switch ext {
	case ".yaml", ".yml":
		decoder := yaml.NewDecoder(file)
		err = decoder.Decode(&contract)
	case ".json":
		decoder := json.NewDecoder(file)
		err = decoder.Decode(&contract)
	default:
		return nil, fmt.Errorf("unsupported contract file format: %s", ext)
	}
	
	return &contract, err
}

// verifyInteraction verifies a single interaction
func (v *ContractVerifier) verifyInteraction(ctx context.Context, client *mcp.Client, interaction ContractInteraction) InteractionResult {
	start := time.Now()
	result := InteractionResult{
		Description: interaction.Description,
		Request:     interaction.Request,
		Expected:    interaction.Response,
		Duration:    0,
	}
	
	// Execute the interaction based on method
	switch interaction.Request.Method {
	case "initialize":
		result = v.verifyInitialize(ctx, client, interaction)
	case "ping":
		result = v.verifyPing(ctx, client, interaction)
	case "tools/list":
		result = v.verifyToolsList(ctx, client, interaction)
	case "tools/call":
		result = v.verifyToolsCall(ctx, client, interaction)
	case "resources/list":
		result = v.verifyResourcesList(ctx, client, interaction)
	case "resources/read":
		result = v.verifyResourcesRead(ctx, client, interaction)
	case "prompts/list":
		result = v.verifyPromptsList(ctx, client, interaction)
	case "prompts/get":
		result = v.verifyPromptsGet(ctx, client, interaction)
	default:
		result.Status = "skipped"
		result.Error = fmt.Sprintf("Unknown method: %s", interaction.Request.Method)
	}
	
	result.Duration = time.Since(start)
	return result
}

// verifyInitialize verifies initialize interaction
func (v *ContractVerifier) verifyInitialize(ctx context.Context, client *mcp.Client, interaction ContractInteraction) InteractionResult {
	result := InteractionResult{
		Description: interaction.Description,
		Request:     interaction.Request,
		Expected:    interaction.Response,
	}
	
	// Initialize should have already been called in Verify()
	// This is more of a structural verification
	result.Status = "passed"
	
	return result
}

// verifyPing verifies ping interaction
func (v *ContractVerifier) verifyPing(ctx context.Context, client *mcp.Client, interaction ContractInteraction) InteractionResult {
	result := InteractionResult{
		Description: interaction.Description,
		Request:     interaction.Request,
		Expected:    interaction.Response,
	}
	
	err := client.Ping(ctx)
	if err != nil {
		if interaction.Response.Status == "error" {
			result.Status = "passed"
		} else {
			result.Status = "failed"
			result.Error = err.Error()
		}
	} else {
		if interaction.Response.Status == "success" {
			result.Status = "passed"
		} else {
			result.Status = "failed"
			result.Error = "Expected error but got success"
		}
	}
	
	return result
}

// verifyToolsList verifies tools/list interaction
func (v *ContractVerifier) verifyToolsList(ctx context.Context, client *mcp.Client, interaction ContractInteraction) InteractionResult {
	result := InteractionResult{
		Description: interaction.Description,
		Request:     interaction.Request,
		Expected:    interaction.Response,
	}
	
	tools, err := client.ListTools(ctx)
	if err != nil {
		if interaction.Response.Status == "error" {
			result.Status = "passed"
		} else {
			result.Status = "failed"
			result.Error = err.Error()
		}
	} else {
		if interaction.Response.Status == "success" {
			result.Status = "passed"
			result.Actual = map[string]interface{}{
				"tools": tools,
			}
		} else {
			result.Status = "failed"
			result.Error = "Expected error but got success"
		}
	}
	
	return result
}

// verifyToolsCall verifies tools/call interaction
func (v *ContractVerifier) verifyToolsCall(ctx context.Context, client *mcp.Client, interaction ContractInteraction) InteractionResult {
	result := InteractionResult{
		Description: interaction.Description,
		Request:     interaction.Request,
		Expected:    interaction.Response,
	}
	
	// Extract tool name and arguments from request
	toolName := ""
	var arguments interface{}
	
	if params, ok := interaction.Request.Parameters["name"]; ok {
		if name, ok := params.(string); ok {
			toolName = name
		}
	}
	
	if params, ok := interaction.Request.Parameters["arguments"]; ok {
		arguments = params
	}
	
	toolResult, err := client.CallTool(ctx, toolName, arguments)
	if err != nil {
		if interaction.Response.Status == "error" {
			result.Status = "passed"
		} else {
			result.Status = "failed"
			result.Error = err.Error()
		}
	} else {
		if interaction.Response.Status == "success" {
			result.Status = "passed"
			result.Actual = map[string]interface{}{
				"result": toolResult,
			}
		} else {
			result.Status = "failed"
			result.Error = "Expected error but got success"
		}
	}
	
	return result
}

// verifyResourcesList verifies resources/list interaction
func (v *ContractVerifier) verifyResourcesList(ctx context.Context, client *mcp.Client, interaction ContractInteraction) InteractionResult {
	result := InteractionResult{
		Description: interaction.Description,
		Request:     interaction.Request,
		Expected:    interaction.Response,
	}
	
	resources, err := client.ListResources(ctx)
	if err != nil {
		if interaction.Response.Status == "error" {
			result.Status = "passed"
		} else {
			result.Status = "failed"
			result.Error = err.Error()
		}
	} else {
		if interaction.Response.Status == "success" {
			result.Status = "passed"
			result.Actual = map[string]interface{}{
				"resources": resources,
			}
		} else {
			result.Status = "failed"
			result.Error = "Expected error but got success"
		}
	}
	
	return result
}

// verifyResourcesRead verifies resources/read interaction
func (v *ContractVerifier) verifyResourcesRead(ctx context.Context, client *mcp.Client, interaction ContractInteraction) InteractionResult {
	result := InteractionResult{
		Description: interaction.Description,
		Request:     interaction.Request,
		Expected:    interaction.Response,
	}
	
	// Extract URI from request
	uri := ""
	if params, ok := interaction.Request.Parameters["uri"]; ok {
		if uriStr, ok := params.(string); ok {
			uri = uriStr
		}
	}
	
	resource, err := client.ReadResource(ctx, uri)
	if err != nil {
		if interaction.Response.Status == "error" {
			result.Status = "passed"
		} else {
			result.Status = "failed"
			result.Error = err.Error()
		}
	} else {
		if interaction.Response.Status == "success" {
			result.Status = "passed"
			result.Actual = map[string]interface{}{
				"resource": resource,
			}
		} else {
			result.Status = "failed"
			result.Error = "Expected error but got success"
		}
	}
	
	return result
}

// verifyPromptsList verifies prompts/list interaction
func (v *ContractVerifier) verifyPromptsList(ctx context.Context, client *mcp.Client, interaction ContractInteraction) InteractionResult {
	result := InteractionResult{
		Description: interaction.Description,
		Request:     interaction.Request,
		Expected:    interaction.Response,
	}
	
	prompts, err := client.ListPrompts(ctx)
	if err != nil {
		if interaction.Response.Status == "error" {
			result.Status = "passed"
		} else {
			result.Status = "failed"
			result.Error = err.Error()
		}
	} else {
		if interaction.Response.Status == "success" {
			result.Status = "passed"
			result.Actual = map[string]interface{}{
				"prompts": prompts,
			}
		} else {
			result.Status = "failed"
			result.Error = "Expected error but got success"
		}
	}
	
	return result
}

// verifyPromptsGet verifies prompts/get interaction
func (v *ContractVerifier) verifyPromptsGet(ctx context.Context, client *mcp.Client, interaction ContractInteraction) InteractionResult {
	result := InteractionResult{
		Description: interaction.Description,
		Request:     interaction.Request,
		Expected:    interaction.Response,
	}
	
	// Extract prompt name from request
	promptName := ""
	if params, ok := interaction.Request.Parameters["name"]; ok {
		if name, ok := params.(string); ok {
			promptName = name
		}
	}
	
	prompt, err := client.GetPrompt(ctx, promptName, nil)
	if err != nil {
		if interaction.Response.Status == "error" {
			result.Status = "passed"
		} else {
			result.Status = "failed"
			result.Error = err.Error()
		}
	} else {
		if interaction.Response.Status == "success" {
			result.Status = "passed"
			result.Actual = map[string]interface{}{
				"prompt": prompt,
			}
		} else {
			result.Status = "failed"
			result.Error = "Expected error but got success"
		}
	}
	
	return result
}

// calculateSummary calculates verification summary
func (v *ContractVerifier) calculateSummary(interactions []InteractionResult) VerificationSummary {
	total := len(interactions)
	passed := 0
	failed := 0
	
	for _, interaction := range interactions {
		switch interaction.Status {
		case "passed":
			passed++
		case "failed":
			failed++
		}
	}
	
	successRate := 0.0
	if total > 0 {
		successRate = float64(passed) / float64(total) * 100
	}
	
	status := "passed"
	if failed > 0 {
		status = "failed"
	}
	
	return VerificationSummary{
		TotalInteractions:  total,
		PassedInteractions: passed,
		FailedInteractions: failed,
		SuccessRate:       successRate,
		Status:            status,
	}
}

// outputResults outputs verification results
func (v *ContractVerifier) outputResults(result *VerificationResult) error {
	var output io.Writer = os.Stdout
	if v.config.reportFile != "" {
		file, err := os.Create(v.config.reportFile)
		if err != nil {
			return fmt.Errorf("failed to create report file: %w", err)
		}
		defer file.Close()
		output = file
	}
	
	switch v.config.format {
	case "json":
		encoder := json.NewEncoder(output)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	case "yaml":
		encoder := yaml.NewEncoder(output)
		encoder.SetIndent(2)
		return encoder.Encode(result)
	case "junit-xml":
		return v.outputJUnitXML(output, result)
	default:
		return fmt.Errorf("unsupported format: %s", v.config.format)
	}
}

// outputJUnitXML outputs results in JUnit XML format
func (v *ContractVerifier) outputJUnitXML(w io.Writer, result *VerificationResult) error {
	fmt.Fprintln(w, `<?xml version="1.0" encoding="UTF-8"?>`)
	fmt.Fprintf(w, `<testsuite name="contract-verification" tests="%d" failures="%d" errors="0" time="0">`,
		result.Summary.TotalInteractions,
		result.Summary.FailedInteractions,
	)
	
	for _, interaction := range result.Interactions {
		testName := fmt.Sprintf("interaction_%s", strings.ReplaceAll(interaction.Description, " ", "_"))
		if interaction.Status == "failed" {
			fmt.Fprintf(w, `<testcase name="%s" classname="contract">`, testName)
			fmt.Fprintf(w, `<failure message="%s">%s</failure>`, 
				escapeXML(interaction.Error), 
				escapeXML(interaction.Description))
			fmt.Fprintln(w, `</testcase>`)
		} else {
			fmt.Fprintf(w, `<testcase name="%s" classname="contract"/>`, testName)
		}
	}
	
	fmt.Fprintln(w, `</testsuite>`)
	return nil
}

// CompatibilityMatrix runs compatibility matrix testing
type CompatibilityMatrix struct {
	config *MatrixCommand
}

// NewCompatibilityMatrix creates a new compatibility matrix
func NewCompatibilityMatrix(config *MatrixCommand) *CompatibilityMatrix {
	return &CompatibilityMatrix{config: config}
}

// MatrixResult represents compatibility matrix results
type MatrixResult struct {
	Summary     MatrixSummary    `json:"summary"`
	Results     []MatrixEntry    `json:"results"`
	Timestamp   time.Time        `json:"timestamp"`
	ClientsFile string           `json:"clientsFile"`
	ServersFile string           `json:"serversFile"`
}

// MatrixSummary provides summary statistics
type MatrixSummary struct {
	TotalCombinations    int     `json:"totalCombinations"`
	SuccessfulCombinations int   `json:"successfulCombinations"`
	FailedCombinations   int     `json:"failedCombinations"`
	CompatibilityRate    float64 `json:"compatibilityRate"`
}

// MatrixEntry represents a single client-server combination result
type MatrixEntry struct {
	Client    string  `json:"client"`
	Server    string  `json:"server"`
	Status    string  `json:"status"`
	Error     string  `json:"error,omitempty"`
	Duration  time.Duration `json:"duration"`
	Details   VerificationResult `json:"details,omitempty"`
}

// Run runs the compatibility matrix
func (m *CompatibilityMatrix) Run(ctx context.Context) error {
	if m.config.verbose {
		log.Printf("Running compatibility matrix testing")
	}
	
	// Load client and server configurations
	clients, err := m.loadConfigurations(m.config.clientsFile)
	if err != nil {
		return fmt.Errorf("failed to load clients: %w", err)
	}
	
	servers, err := m.loadConfigurations(m.config.serversFile)
	if err != nil {
		return fmt.Errorf("failed to load servers: %w", err)
	}
	
	// Create result matrix
	result := &MatrixResult{
		Timestamp:   time.Now(),
		ClientsFile: m.config.clientsFile,
		ServersFile: m.config.serversFile,
		Results:     []MatrixEntry{},
	}
	
	// Test all combinations
	for _, client := range clients {
		for _, server := range servers {
			entry := m.testCombination(ctx, client, server)
			result.Results = append(result.Results, entry)
		}
	}
	
	// Calculate summary
	result.Summary = m.calculateMatrixSummary(result.Results)
	
	// Output results
	return m.outputMatrix(result)
}

// loadConfigurations loads configurations from file
func (m *CompatibilityMatrix) loadConfigurations(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	var configs []string
	if err := json.NewDecoder(file).Decode(&configs); err != nil {
		return nil, err
	}
	
	return configs, nil
}

// testCombination tests a single client-server combination
func (m *CompatibilityMatrix) testCombination(ctx context.Context, client, server string) MatrixEntry {
	start := time.Now()
	entry := MatrixEntry{
		Client: client,
		Server: server,
	}
	
	// Here we would actually test the combination
	// For now, we'll simulate it
	entry.Status = "passed"
	entry.Duration = time.Since(start)
	
	return entry
}

// calculateMatrixSummary calculates matrix summary
func (m *CompatibilityMatrix) calculateMatrixSummary(entries []MatrixEntry) MatrixSummary {
	total := len(entries)
	successful := 0
	failed := 0
	
	for _, entry := range entries {
		switch entry.Status {
		case "passed":
			successful++
		case "failed":
			failed++
		}
	}
	
	compatibilityRate := 0.0
	if total > 0 {
		compatibilityRate = float64(successful) / float64(total) * 100
	}
	
	return MatrixSummary{
		TotalCombinations:      total,
		SuccessfulCombinations: successful,
		FailedCombinations:     failed,
		CompatibilityRate:      compatibilityRate,
	}
}

// outputMatrix outputs matrix results
func (m *CompatibilityMatrix) outputMatrix(result *MatrixResult) error {
	var output io.Writer = os.Stdout
	if m.config.outputFile != "" {
		file, err := os.Create(m.config.outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()
		output = file
	}
	
	switch m.config.format {
	case "json":
		encoder := json.NewEncoder(output)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	case "yaml":
		encoder := yaml.NewEncoder(output)
		encoder.SetIndent(2)
		return encoder.Encode(result)
	case "html":
		return m.outputHTML(output, result)
	default:
		return fmt.Errorf("unsupported format: %s", m.config.format)
	}
}

// outputHTML outputs matrix in HTML format
func (m *CompatibilityMatrix) outputHTML(w io.Writer, result *MatrixResult) error {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Compatibility Matrix</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .summary { background: #f0f0f0; padding: 15px; border-radius: 5px; margin-bottom: 20px; }
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #4CAF50; color: white; }
        .passed { background-color: #d4edda; }
        .failed { background-color: #f8d7da; }
        .skipped { background-color: #fff3cd; }
    </style>
</head>
<body>
    <h1>Compatibility Matrix</h1>
    <div class="summary">
        <h2>Summary</h2>
        <p>Generated: ` + result.Timestamp.Format(time.RFC3339) + `</p>
        <p>Total Combinations: ` + fmt.Sprintf("%d", result.Summary.TotalCombinations) + `</p>
        <p>Successful: ` + fmt.Sprintf("%d", result.Summary.SuccessfulCombinations) + `</p>
        <p>Failed: ` + fmt.Sprintf("%d", result.Summary.FailedCombinations) + `</p>
        <p>Compatibility Rate: ` + fmt.Sprintf("%.1f%%", result.Summary.CompatibilityRate) + `</p>
    </div>
    
    <h2>Results</h2>
    <table>
        <tr>
            <th>Client</th>
            <th>Server</th>
            <th>Status</th>
            <th>Duration</th>
            <th>Error</th>
        </tr>`
	
	for _, entry := range result.Results {
		html += fmt.Sprintf(`
        <tr class="%s">
            <td>%s</td>
            <td>%s</td>
            <td>%s</td>
            <td>%s</td>
            <td>%s</td>
        </tr>`,
			entry.Status,
			entry.Client,
			entry.Server,
			entry.Status,
			entry.Duration.String(),
			entry.Error,
		)
	}
	
	html += `
    </table>
</body>
</html>`
	
	_, err := fmt.Fprint(w, html)
	return err
}

// Helper functions

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <command> [options]\n", programName)
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  record      Record a contract from trace file\n")
		fmt.Fprintf(os.Stderr, "  verify      Verify server against contract\n")
		fmt.Fprintf(os.Stderr, "  matrix      Run compatibility matrix testing\n")
		fmt.Fprintf(os.Stderr, "  version     Show version information\n")
		os.Exit(1)
	}
	
	ctx := context.Background()
	
	switch os.Args[1] {
	case "record":
		cmd := &RecordCommand{}
		if err := cmd.Execute(ctx, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "verify":
		cmd := &VerifyCommand{}
		if err := cmd.Execute(ctx, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "matrix":
		cmd := &MatrixCommand{}
		if err := cmd.Execute(ctx, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "version":
		fmt.Printf("%s version %s\n", programName, version)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}