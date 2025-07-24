// Package main - Integration example showing how mcp-health, mcp-config, and mcp-deploy work together
//
// This file demonstrates how the three production operations tools integrate
// with each other and with the existing MCP middleware system.

package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/modelcontextprotocol"
)

// ProductionOperationsConfig represents the combined configuration for all production tools
type ProductionOperationsConfig struct {
	Health HealthConfig       `json:"health" yaml:"health"`
	Config ConfigManagementConfig `json:"config" yaml:"config"`
	Deploy DeploymentConfig   `json:"deploy" yaml:"deploy"`
}

// ProductionOperationsManager manages all production operations tools
type ProductionOperationsManager struct {
	logger      *slog.Logger
	healthApp   *HealthApp
	configApp   *ConfigApp
	deployApp   *DeployApp
	mcpServer   *mcp.Server
	middleware  []mcp.Middleware
}

// NewProductionOperationsManager creates a new production operations manager
func NewProductionOperationsManager(config *ProductionOperationsConfig) (*ProductionOperationsManager, error) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Initialize individual applications
	healthApp := NewHealthApp(&config.Health)
	configApp := NewConfigApp(&config.Config)
	deployApp := NewDeployApp(&config.Deploy)

	// Create MCP server with enhanced middleware
	server := mcp.NewServer(
		mcp.WithServerName("mcp-production-operations"),
		mcp.WithServerVersion("1.0.0"),
		mcp.WithServerInstructions("Production operations server with health, config, and deployment management"),
	)

	manager := &ProductionOperationsManager{
		logger:    logger,
		healthApp: healthApp,
		configApp: configApp,
		deployApp: deployApp,
		mcpServer: server,
	}

	// Setup middleware stack
	if err := manager.setupMiddleware(); err != nil {
		return nil, fmt.Errorf("failed to setup middleware: %w", err)
	}

	// Register tools
	if err := manager.registerTools(); err != nil {
		return nil, fmt.Errorf("failed to register tools: %w", err)
	}

	return manager, nil
}

// setupMiddleware sets up the middleware stack for production operations
func (m *ProductionOperationsManager) setupMiddleware() error {
	// Health checking middleware
	healthMiddleware := mcp.MiddlewareFunc(func(next mcp.Handler) mcp.Handler {
		return mcp.HandlerFunc(func(ctx context.Context, req mcp.Request) (mcp.Response, error) {
			// Check system health before processing requests
			if !m.isSystemHealthy(ctx) {
				return nil, fmt.Errorf("system is not healthy")
			}
			return next.Handle(ctx, req)
		})
	})

	// Configuration injection middleware
	configMiddleware := mcp.MiddlewareFunc(func(next mcp.Handler) mcp.Handler {
		return mcp.HandlerFunc(func(ctx context.Context, req mcp.Request) (mcp.Response, error) {
			// Inject current configuration into context
			config := m.configApp.getCurrentConfig(ctx)
			ctx = context.WithValue(ctx, "config", config)
			return next.Handle(ctx, req)
		})
	})

	// Deployment state middleware
	deployMiddleware := mcp.MiddlewareFunc(func(next mcp.Handler) mcp.Handler {
		return mcp.HandlerFunc(func(ctx context.Context, req mcp.Request) (mcp.Response, error) {
			// Check deployment state
			if m.isDeploymentInProgress(ctx) {
				return nil, fmt.Errorf("deployment in progress, please try again later")
			}
			return next.Handle(ctx, req)
		})
	})

	// Audit logging middleware
	auditMiddleware := mcp.MiddlewareFunc(func(next mcp.Handler) mcp.Handler {
		return mcp.HandlerFunc(func(ctx context.Context, req mcp.Request) (mcp.Response, error) {
			start := time.Now()
			
			// Log request
			m.logger.Info("Production operations request",
				"method", req.GetMethod(),
				"timestamp", start.Format(time.RFC3339),
			)

			// Process request
			response, err := next.Handle(ctx, req)

			// Log response
			m.logger.Info("Production operations response",
				"method", req.GetMethod(),
				"duration", time.Since(start),
				"success", err == nil,
			)

			return response, err
		})
	})

	// Add middleware to stack
	m.middleware = []mcp.Middleware{
		auditMiddleware,
		healthMiddleware,
		configMiddleware,
		deployMiddleware,
	}

	return nil
}

// registerTools registers all production operations tools
func (m *ProductionOperationsManager) registerTools() error {
	// Health management tools
	if err := m.registerHealthTools(); err != nil {
		return fmt.Errorf("failed to register health tools: %w", err)
	}

	// Configuration management tools
	if err := m.registerConfigTools(); err != nil {
		return fmt.Errorf("failed to register config tools: %w", err)
	}

	// Deployment management tools
	if err := m.registerDeployTools(); err != nil {
		return fmt.Errorf("failed to register deploy tools: %w", err)
	}

	return nil
}

// registerHealthTools registers health management tools
func (m *ProductionOperationsManager) registerHealthTools() error {
	// Health check tool
	healthCheckTool := mcp.Tool{
		Name:        "health_check",
		Description: "Perform health check on specified service",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"service": map[string]interface{}{
					"type":        "string",
					"description": "Service name to check",
				},
				"protocol": map[string]interface{}{
					"type":        "string",
					"description": "Health check protocol (http, tcp, mcp)",
					"enum":        []string{"http", "tcp", "mcp"},
				},
			},
			"required": []string{"service"},
		},
	}

	handler := func(ctx context.Context, req modelcontextprotocol.CallToolRequest) (*modelcontextprotocol.CallToolResult, error) {
		var params struct {
			Service  string `json:"service"`
			Protocol string `json:"protocol"`
		}

		if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %w", err)
		}

		// Perform health check
		result, err := m.performHealthCheck(ctx, params.Service, params.Protocol)
		if err != nil {
			return nil, fmt.Errorf("health check failed: %w", err)
		}

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				{
					Type: "text",
					Text: fmt.Sprintf("Health check result: %s", result),
				},
			},
		}, nil
	}

	return m.mcpServer.RegisterTool(healthCheckTool, handler)
}

// registerConfigTools registers configuration management tools
func (m *ProductionOperationsManager) registerConfigTools() error {
	// Get configuration tool
	getConfigTool := mcp.Tool{
		Name:        "get_config",
		Description: "Get current configuration for specified environment",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"environment": map[string]interface{}{
					"type":        "string",
					"description": "Environment name",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Configuration path (optional)",
				},
			},
			"required": []string{"environment"},
		},
	}

	getConfigHandler := func(ctx context.Context, req modelcontextprotocol.CallToolRequest) (*modelcontextprotocol.CallToolResult, error) {
		var params struct {
			Environment string `json:"environment"`
			Path        string `json:"path"`
		}

		if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %w", err)
		}

		// Get configuration
		config, err := m.getConfiguration(ctx, params.Environment, params.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to get configuration: %w", err)
		}

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				{
					Type: "text",
					Text: fmt.Sprintf("Configuration: %s", config),
				},
			},
		}, nil
	}

	return m.mcpServer.RegisterTool(getConfigTool, getConfigHandler)
}

// registerDeployTools registers deployment management tools
func (m *ProductionOperationsManager) registerDeployTools() error {
	// Deploy tool
	deployTool := mcp.Tool{
		Name:        "deploy",
		Description: "Deploy application to specified environment",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"environment": map[string]interface{}{
					"type":        "string",
					"description": "Target environment",
				},
				"version": map[string]interface{}{
					"type":        "string",
					"description": "Version to deploy",
				},
				"strategy": map[string]interface{}{
					"type":        "string",
					"description": "Deployment strategy",
					"enum":        []string{"rolling", "blue_green", "canary"},
				},
			},
			"required": []string{"environment", "version"},
		},
	}

	deployHandler := func(ctx context.Context, req modelcontextprotocol.CallToolRequest) (*modelcontextprotocol.CallToolResult, error) {
		var params struct {
			Environment string `json:"environment"`
			Version     string `json:"version"`
			Strategy    string `json:"strategy"`
		}

		if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %w", err)
		}

		// Perform deployment
		deployment, err := m.performDeployment(ctx, params.Environment, params.Version, params.Strategy)
		if err != nil {
			return nil, fmt.Errorf("deployment failed: %w", err)
		}

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				{
					Type: "text",
					Text: fmt.Sprintf("Deployment started: %s", deployment.ID),
				},
			},
		}, nil
	}

	return m.mcpServer.RegisterTool(deployTool, deployHandler)
}

// isSystemHealthy checks if the system is healthy
func (m *ProductionOperationsManager) isSystemHealthy(ctx context.Context) bool {
	// Check health of critical services
	allStatus := m.healthApp.monitor.GetAllStatus()
	
	criticalServices := []string{"database", "redis", "api-server"}
	for _, service := range criticalServices {
		if status, exists := allStatus[service]; exists {
			if status.Status == "unhealthy" {
				m.logger.Warn("Critical service unhealthy", "service", service)
				return false
			}
		}
	}

	return true
}

// isDeploymentInProgress checks if a deployment is in progress
func (m *ProductionOperationsManager) isDeploymentInProgress(ctx context.Context) bool {
	deployments := m.deployApp.deployer.ListDeployments()
	
	for _, deployment := range deployments {
		if deployment.Status == "deploying" || deployment.Status == "rolling_back" {
			return true
		}
	}

	return false
}

// performHealthCheck performs a health check on a service
func (m *ProductionOperationsManager) performHealthCheck(ctx context.Context, service, protocol string) (string, error) {
	// Find health check configuration
	var checkConfig *HealthCheckConfig
	for _, config := range m.healthApp.config.HealthChecks {
		if config.Name == service {
			checkConfig = &config
			break
		}
	}

	if checkConfig == nil {
		return "", fmt.Errorf("health check configuration not found for service: %s", service)
	}

	// Override protocol if specified
	if protocol != "" {
		checkConfig.Protocol = protocol
	}

	// Perform health check
	result, err := m.healthApp.checker.CheckHealth(ctx, *checkConfig)
	if err != nil {
		return "", fmt.Errorf("health check failed: %w", err)
	}

	return fmt.Sprintf("Status: %s, Duration: %s, Message: %s", 
		result.Status, result.Duration, result.Message), nil
}

// getConfiguration gets configuration for an environment
func (m *ProductionOperationsManager) getConfiguration(ctx context.Context, environment, path string) (string, error) {
	// Get environment configuration
	envConfig, exists := m.configApp.config.Environments[environment]
	if !exists {
		return "", fmt.Errorf("environment not found: %s", environment)
	}

	// Load configuration files
	config := make(map[string]interface{})
	for _, configPath := range envConfig.ConfigPaths {
		data, err := os.ReadFile(configPath)
		if err != nil {
			m.logger.Warn("Failed to read config file", "path", configPath, "error", err)
			continue
		}

		var fileConfig map[string]interface{}
		if err := yaml.Unmarshal(data, &fileConfig); err != nil {
			m.logger.Warn("Failed to parse config file", "path", configPath, "error", err)
			continue
		}

		// Merge configuration
		for key, value := range fileConfig {
			config[key] = value
		}
	}

	// Apply environment variables
	for key, value := range envConfig.Variables {
		config[key] = value
	}

	// Apply overrides
	for key, value := range envConfig.Overrides {
		config[key] = value
	}

	// Convert to JSON for output
	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal configuration: %w", err)
	}

	return string(configJSON), nil
}

// performDeployment performs a deployment
func (m *ProductionOperationsManager) performDeployment(ctx context.Context, environment, version, strategy string) (*Deployment, error) {
	// Validate environment
	if _, exists := m.deployApp.config.Environments[environment]; !exists {
		return nil, fmt.Errorf("environment not found: %s", environment)
	}

	// Override strategy if specified
	if strategy != "" {
		if envConfig, exists := m.deployApp.config.Environments[environment]; exists {
			envConfig.Strategy = strategy
		}
	}

	// Perform pre-deployment health checks
	if !m.isSystemHealthy(ctx) {
		return nil, fmt.Errorf("system is not healthy, deployment aborted")
	}

	// Validate configuration
	if err := m.configApp.validator.ValidateConfig(fmt.Sprintf("config/%s.yaml", environment)); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Start deployment
	deployment, err := m.deployApp.deployer.Deploy(ctx, environment, version)
	if err != nil {
		return nil, fmt.Errorf("deployment failed: %w", err)
	}

	// Start monitoring deployment
	go m.monitorDeployment(ctx, deployment)

	return deployment, nil
}

// monitorDeployment monitors a deployment and handles failures
func (m *ProductionOperationsManager) monitorDeployment(ctx context.Context, deployment *Deployment) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	timeout := time.After(30 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			return
		case <-timeout:
			m.logger.Error("Deployment timeout", "deployment", deployment.ID)
			m.rollbackDeployment(ctx, deployment)
			return
		case <-ticker.C:
			// Check deployment status
			currentDeployment, exists := m.deployApp.deployer.GetDeployment(deployment.ID)
			if !exists {
				m.logger.Error("Deployment not found", "deployment", deployment.ID)
				return
			}

			switch currentDeployment.Status {
			case "deployed":
				m.logger.Info("Deployment completed successfully", "deployment", deployment.ID)
				m.sendDeploymentNotification(ctx, deployment, "success")
				return
			case "failed":
				m.logger.Error("Deployment failed", "deployment", deployment.ID, "error", currentDeployment.Error)
				m.rollbackDeployment(ctx, currentDeployment)
				m.sendDeploymentNotification(ctx, deployment, "failed")
				return
			case "deploying":
				// Check health during deployment
				if !m.isSystemHealthy(ctx) {
					m.logger.Warn("System unhealthy during deployment", "deployment", deployment.ID)
					m.rollbackDeployment(ctx, currentDeployment)
					return
				}
			}
		}
	}
}

// rollbackDeployment rolls back a deployment
func (m *ProductionOperationsManager) rollbackDeployment(ctx context.Context, deployment *Deployment) {
	m.logger.Info("Rolling back deployment", "deployment", deployment.ID, "environment", deployment.Environment)

	if err := m.deployApp.deployer.Rollback(ctx, deployment.Environment, ""); err != nil {
		m.logger.Error("Rollback failed", "deployment", deployment.ID, "error", err)
		return
	}

	m.logger.Info("Rollback completed", "deployment", deployment.ID)
}

// sendDeploymentNotification sends deployment notifications
func (m *ProductionOperationsManager) sendDeploymentNotification(ctx context.Context, deployment *Deployment, status string) {
	message := fmt.Sprintf("Deployment %s: %s to %s (%s)", 
		status, deployment.Version, deployment.Environment, deployment.ID)

	// Send to configured alerting channels
	m.logger.Info("Deployment notification", "message", message)

	// Integration with alerting systems would go here
	// For example, send to Slack, email, PagerDuty, etc.
}

// Run starts the production operations manager
func (m *ProductionOperationsManager) Run(ctx context.Context) error {
	m.logger.Info("Starting production operations manager")

	// Start individual applications
	go func() {
		if err := m.healthApp.Run(ctx); err != nil {
			m.logger.Error("Health app failed", "error", err)
		}
	}()

	go func() {
		if err := m.configApp.Run(ctx, "serve", []string{}); err != nil {
			m.logger.Error("Config app failed", "error", err)
		}
	}()

	go func() {
		if err := m.deployApp.Run(ctx, "serve", []string{}); err != nil {
			m.logger.Error("Deploy app failed", "error", err)
		}
	}()

	// Start MCP server with middleware
	transport := mcp.NewStdioTransport()
	
	// Apply middleware
	handler := m.mcpServer.Handler()
	for i := len(m.middleware) - 1; i >= 0; i-- {
		handler = m.middleware[i].Apply(handler)
	}

	// Create server with middleware
	server := mcp.NewServer(
		mcp.WithServerName("mcp-production-operations"),
		mcp.WithServerVersion("1.0.0"),
	)

	// Set up server with middleware-wrapped handler
	// This would require extending the server to accept custom handlers
	
	m.logger.Info("Production operations manager started")

	// Wait for context cancellation
	<-ctx.Done()

	m.logger.Info("Production operations manager stopped")
	return nil
}

// Example usage showing how to use the production operations manager
func ExampleProductionOperationsUsage() {
	// Create configuration
	config := &ProductionOperationsConfig{
		Health: HealthConfig{
			ServiceName: "mcp-health",
			Port:        8080,
			HealthChecks: []HealthCheckConfig{
				{
					Name:     "database",
					Target:   "localhost:5432",
					Protocol: "tcp",
					Interval: 30 * time.Second,
					Timeout:  5 * time.Second,
				},
				{
					Name:     "api-server",
					Target:   "localhost:8080",
					Protocol: "http",
					HTTPPath: "/health",
					Interval: 10 * time.Second,
					Timeout:  5 * time.Second,
				},
			},
		},
		Config: ConfigManagementConfig{
			ServiceName: "mcp-config",
			Port:        8081,
			Environment: "production",
			Environments: map[string]*EnvConfig{
				"production": {
					Name:        "production",
					ConfigPaths: []string{"config/prod.yaml"},
					Variables: map[string]string{
						"LOG_LEVEL": "warn",
					},
				},
			},
		},
		Deploy: DeploymentConfig{
			ServiceName: "mcp-deploy",
			Port:        8082,
			Platform: PlatformConfig{
				Type: "kubernetes",
			},
			Environments: map[string]*EnvironmentConfig{
				"production": {
					Name:     "production",
					Strategy: "canary",
					Replicas: 5,
				},
			},
		},
	}

	// Create production operations manager
	manager, err := NewProductionOperationsManager(config)
	if err != nil {
		fmt.Printf("Failed to create production operations manager: %v\n", err)
		return
	}

	// Run the manager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := manager.Run(ctx); err != nil {
		fmt.Printf("Production operations manager failed: %v\n", err)
	}
}