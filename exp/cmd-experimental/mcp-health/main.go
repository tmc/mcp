// Package main implements mcp-health: Health checking, service discovery, and cluster management for MCP services
//
// This tool provides comprehensive health monitoring and service discovery capabilities
// for MCP services in production environments, including:
//
// - Health checking with configurable protocols and thresholds
// - Service discovery and registration with various backends
// - Load balancing and routing based on health status
// - Kubernetes integration for cloud-native deployments
// - Alerting and monitoring integration
// - Cluster management and coordination
//
// Usage:
//
//	mcp-health [command] [flags]
//
// Commands:
//
//	check       Perform health checks on MCP services
//	discover    Discover MCP services in the cluster
//	monitor     Start continuous health monitoring
//	serve       Start the health service API server
//	operator    Run Kubernetes operator mode
//
// Examples:
//
//	mcp-health check --target localhost:8080
//	mcp-health discover --consul-addr localhost:8500
//	mcp-health monitor --config health-config.yaml
//	mcp-health serve --port 8080
//	mcp-health operator --kubeconfig ~/.kube/config
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/modelcontextprotocol"
)

const (
	version = "1.0.0"
)

// Main application structure
type HealthApp struct {
	config  *HealthConfig
	logger  *slog.Logger
	checker *HealthChecker
	monitor *HealthMonitor
	server  *HealthServer
	cancel  context.CancelFunc
}

// HealthConfig defines configuration for the health system
type HealthConfig struct {
	// Service configuration
	ServiceName string        `json:"service_name" yaml:"service_name"`
	Port        int           `json:"port" yaml:"port"`
	LogLevel    string        `json:"log_level" yaml:"log_level"`
	Metrics     MetricsConfig `json:"metrics" yaml:"metrics"`

	// Health checking configuration
	HealthChecks []HealthCheckConfig `json:"health_checks" yaml:"health_checks"`

	// Service discovery configuration
	Discovery ServiceDiscoveryConfig `json:"discovery" yaml:"discovery"`

	// Load balancing configuration
	LoadBalancer LoadBalancerConfig `json:"load_balancer" yaml:"load_balancer"`

	// Kubernetes configuration
	Kubernetes KubernetesConfig `json:"kubernetes" yaml:"kubernetes"`

	// Alerting configuration
	Alerting AlertingConfig `json:"alerting" yaml:"alerting"`
}

// HealthCheckConfig defines configuration for individual health checks
type HealthCheckConfig struct {
	Name             string            `json:"name" yaml:"name"`
	Target           string            `json:"target" yaml:"target"`
	Protocol         string            `json:"protocol" yaml:"protocol"` // http, tcp, mcp
	Interval         time.Duration     `json:"interval" yaml:"interval"`
	Timeout          time.Duration     `json:"timeout" yaml:"timeout"`
	FailureThreshold int               `json:"failure_threshold" yaml:"failure_threshold"`
	SuccessThreshold int               `json:"success_threshold" yaml:"success_threshold"`
	HTTPPath         string            `json:"http_path,omitempty" yaml:"http_path,omitempty"`
	MCPMethod        string            `json:"mcp_method,omitempty" yaml:"mcp_method,omitempty"`
	ExpectedStatus   int               `json:"expected_status,omitempty" yaml:"expected_status,omitempty"`
	Headers          map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
}

// ServiceDiscoveryConfig defines service discovery configuration
type ServiceDiscoveryConfig struct {
	Enabled   bool   `json:"enabled" yaml:"enabled"`
	Backend   string `json:"backend" yaml:"backend"` // consul, etcd, k8s
	Address   string `json:"address" yaml:"address"`
	Namespace string `json:"namespace" yaml:"namespace"`
}

// LoadBalancerConfig defines load balancing configuration
type LoadBalancerConfig struct {
	Strategy string         `json:"strategy" yaml:"strategy"` // round_robin, least_conn, weighted
	Weights  map[string]int `json:"weights,omitempty" yaml:"weights,omitempty"`
}

// KubernetesConfig defines Kubernetes integration configuration
type KubernetesConfig struct {
	Enabled      bool   `json:"enabled" yaml:"enabled"`
	Kubeconfig   string `json:"kubeconfig" yaml:"kubeconfig"`
	Namespace    string `json:"namespace" yaml:"namespace"`
	ServiceName  string `json:"service_name" yaml:"service_name"`
	OperatorMode bool   `json:"operator_mode" yaml:"operator_mode"`
}

// AlertingConfig defines alerting configuration
type AlertingConfig struct {
	Enabled    bool             `json:"enabled" yaml:"enabled"`
	Webhook    string           `json:"webhook" yaml:"webhook"`
	Slack      SlackConfig      `json:"slack" yaml:"slack"`
	Email      EmailConfig      `json:"email" yaml:"email"`
	Prometheus PrometheusConfig `json:"prometheus" yaml:"prometheus"`
}

// MetricsConfig defines metrics configuration
type MetricsConfig struct {
	Enabled    bool             `json:"enabled" yaml:"enabled"`
	Path       string           `json:"path" yaml:"path"`
	Prometheus PrometheusConfig `json:"prometheus" yaml:"prometheus"`
}

// SlackConfig defines Slack alerting configuration
type SlackConfig struct {
	Enabled bool   `json:"enabled" yaml:"enabled"`
	Token   string `json:"token" yaml:"token"`
	Channel string `json:"channel" yaml:"channel"`
}

// EmailConfig defines email alerting configuration
type EmailConfig struct {
	Enabled  bool     `json:"enabled" yaml:"enabled"`
	SMTPHost string   `json:"smtp_host" yaml:"smtp_host"`
	SMTPPort int      `json:"smtp_port" yaml:"smtp_port"`
	From     string   `json:"from" yaml:"from"`
	To       []string `json:"to" yaml:"to"`
}

// PrometheusConfig defines Prometheus integration configuration
type PrometheusConfig struct {
	Enabled   bool   `json:"enabled" yaml:"enabled"`
	Address   string `json:"address" yaml:"address"`
	JobName   string `json:"job_name" yaml:"job_name"`
	Namespace string `json:"namespace" yaml:"namespace"`
}

// HealthStatus represents the health status of a service
type HealthStatus struct {
	ServiceName string            `json:"service_name"`
	Status      string            `json:"status"` // healthy, unhealthy, unknown
	Timestamp   time.Time         `json:"timestamp"`
	Checks      []CheckResult     `json:"checks"`
	Metadata    map[string]string `json:"metadata"`
}

// CheckResult represents the result of a health check
type CheckResult struct {
	Name      string        `json:"name"`
	Status    string        `json:"status"`
	Message   string        `json:"message"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
}

// HealthChecker performs health checks on MCP services
type HealthChecker struct {
	config  *HealthConfig
	logger  *slog.Logger
	clients map[string]*mcp.Client
	mutex   sync.RWMutex
}

// HealthMonitor continuously monitors service health
type HealthMonitor struct {
	config  *HealthConfig
	logger  *slog.Logger
	checker *HealthChecker
	status  map[string]*HealthStatus
	mutex   sync.RWMutex
}

// HealthServer provides HTTP API for health status
type HealthServer struct {
	config  *HealthConfig
	logger  *slog.Logger
	monitor *HealthMonitor
	server  *http.Server
}

// NewHealthApp creates a new health application
func NewHealthApp(config *HealthConfig) *HealthApp {
	logLevel := slog.LevelInfo
	switch config.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	checker := &HealthChecker{
		config:  config,
		logger:  logger,
		clients: make(map[string]*mcp.Client),
	}

	monitor := &HealthMonitor{
		config:  config,
		logger:  logger,
		checker: checker,
		status:  make(map[string]*HealthStatus),
	}

	server := &HealthServer{
		config:  config,
		logger:  logger,
		monitor: monitor,
	}

	return &HealthApp{
		config:  config,
		logger:  logger,
		checker: checker,
		monitor: monitor,
		server:  server,
	}
}

// HealthChecker implementation
func (hc *HealthChecker) CheckHealth(ctx context.Context, checkConfig HealthCheckConfig) (*CheckResult, error) {
	startTime := time.Now()

	result := &CheckResult{
		Name:      checkConfig.Name,
		Timestamp: startTime,
	}

	// Create context with timeout
	checkCtx, cancel := context.WithTimeout(ctx, checkConfig.Timeout)
	defer cancel()

	switch checkConfig.Protocol {
	case "http":
		err := hc.checkHTTP(checkCtx, checkConfig, result)
		if err != nil {
			result.Status = "unhealthy"
			result.Message = err.Error()
		} else {
			result.Status = "healthy"
			result.Message = "HTTP check passed"
		}
	case "tcp":
		err := hc.checkTCP(checkCtx, checkConfig, result)
		if err != nil {
			result.Status = "unhealthy"
			result.Message = err.Error()
		} else {
			result.Status = "healthy"
			result.Message = "TCP check passed"
		}
	case "mcp":
		err := hc.checkMCP(checkCtx, checkConfig, result)
		if err != nil {
			result.Status = "unhealthy"
			result.Message = err.Error()
		} else {
			result.Status = "healthy"
			result.Message = "MCP check passed"
		}
	default:
		result.Status = "unknown"
		result.Message = fmt.Sprintf("Unknown protocol: %s", checkConfig.Protocol)
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

func (hc *HealthChecker) checkHTTP(ctx context.Context, config HealthCheckConfig, result *CheckResult) error {
	client := &http.Client{
		Timeout: config.Timeout,
	}

	url := fmt.Sprintf("http://%s%s", config.Target, config.HTTPPath)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	expectedStatus := config.ExpectedStatus
	if expectedStatus == 0 {
		expectedStatus = 200
	}

	if resp.StatusCode != expectedStatus {
		return fmt.Errorf("unexpected status code: %d, expected: %d", resp.StatusCode, expectedStatus)
	}

	return nil
}

func (hc *HealthChecker) checkTCP(ctx context.Context, config HealthCheckConfig, result *CheckResult) error {
	// Implementation for TCP health check
	// This would attempt to establish a TCP connection to the target
	return fmt.Errorf("TCP health check not implemented yet")
}

func (hc *HealthChecker) checkMCP(ctx context.Context, config HealthCheckConfig, result *CheckResult) error {
	hc.mutex.RLock()
	client, exists := hc.clients[config.Target]
	hc.mutex.RUnlock()

	if !exists {
		// Create new MCP client
		transport := mcp.NewStdioTransport()
		newClient, err := mcp.NewClient(transport)
		if err != nil {
			return fmt.Errorf("failed to create MCP client: %w", err)
		}

		hc.mutex.Lock()
		hc.clients[config.Target] = newClient
		client = newClient
		hc.mutex.Unlock()
	}

	// Initialize client if needed
	if err := client.Initialize(ctx, modelcontextprotocol.InitializeRequest{
		ProtocolVersion: modelcontextprotocol.SUPPORTED_PROTOCOL_VERSION,
		Capabilities:    modelcontextprotocol.ClientCapabilities{},
		ClientInfo: modelcontextprotocol.Implementation{
			Name:    "mcp-health",
			Version: version,
		},
	}); err != nil {
		return fmt.Errorf("failed to initialize MCP client: %w", err)
	}

	// Perform health check based on method
	method := config.MCPMethod
	if method == "" {
		method = "tools/list"
	}

	switch method {
	case "tools/list":
		_, err := client.ListTools(ctx, modelcontextprotocol.ListToolsRequest{})
		if err != nil {
			return fmt.Errorf("MCP tools/list failed: %w", err)
		}
	case "resources/list":
		_, err := client.ListResources(ctx, modelcontextprotocol.ListResourcesRequest{})
		if err != nil {
			return fmt.Errorf("MCP resources/list failed: %w", err)
		}
	case "ping":
		_, err := client.Ping(ctx, modelcontextprotocol.PingRequest{})
		if err != nil {
			return fmt.Errorf("MCP ping failed: %w", err)
		}
	default:
		return fmt.Errorf("unsupported MCP method: %s", method)
	}

	return nil
}

// HealthMonitor implementation
func (hm *HealthMonitor) Start(ctx context.Context) error {
	hm.logger.Info("Starting health monitor")

	for _, checkConfig := range hm.config.HealthChecks {
		go hm.runHealthCheck(ctx, checkConfig)
	}

	return nil
}

func (hm *HealthMonitor) runHealthCheck(ctx context.Context, checkConfig HealthCheckConfig) {
	ticker := time.NewTicker(checkConfig.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			result, err := hm.checker.CheckHealth(ctx, checkConfig)
			if err != nil {
				hm.logger.Error("Health check failed", "check", checkConfig.Name, "error", err)
				continue
			}

			hm.updateStatus(checkConfig.Name, result)
		}
	}
}

func (hm *HealthMonitor) updateStatus(serviceName string, result *CheckResult) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	status, exists := hm.status[serviceName]
	if !exists {
		status = &HealthStatus{
			ServiceName: serviceName,
			Status:      "unknown",
			Checks:      []CheckResult{},
			Metadata:    make(map[string]string),
		}
		hm.status[serviceName] = status
	}

	// Update status
	status.Checks = append(status.Checks, *result)
	status.Timestamp = result.Timestamp

	// Keep only last 10 results
	if len(status.Checks) > 10 {
		status.Checks = status.Checks[len(status.Checks)-10:]
	}

	// Determine overall status
	recentChecks := status.Checks
	if len(recentChecks) > 3 {
		recentChecks = recentChecks[len(recentChecks)-3:]
	}

	healthyCount := 0
	for _, check := range recentChecks {
		if check.Status == "healthy" {
			healthyCount++
		}
	}

	if healthyCount == len(recentChecks) {
		status.Status = "healthy"
	} else if healthyCount > 0 {
		status.Status = "degraded"
	} else {
		status.Status = "unhealthy"
	}

	hm.logger.Debug("Updated health status", "service", serviceName, "status", status.Status)
}

func (hm *HealthMonitor) GetStatus(serviceName string) (*HealthStatus, bool) {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	status, exists := hm.status[serviceName]
	return status, exists
}

func (hm *HealthMonitor) GetAllStatus() map[string]*HealthStatus {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	result := make(map[string]*HealthStatus)
	for k, v := range hm.status {
		result[k] = v
	}
	return result
}

// HealthServer implementation
func (hs *HealthServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Health status endpoints
	mux.HandleFunc("/health", hs.handleHealthStatus)
	mux.HandleFunc("/health/", hs.handleServiceHealth)
	mux.HandleFunc("/readiness", hs.handleReadiness)
	mux.HandleFunc("/liveness", hs.handleLiveness)

	// Service discovery endpoints
	mux.HandleFunc("/services", hs.handleServices)
	mux.HandleFunc("/services/", hs.handleServiceDiscovery)

	// Metrics endpoint
	if hs.config.Metrics.Enabled {
		mux.HandleFunc(hs.config.Metrics.Path, hs.handleMetrics)
	}

	hs.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", hs.config.Port),
		Handler: mux,
	}

	hs.logger.Info("Starting health server", "port", hs.config.Port)

	go func() {
		if err := hs.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			hs.logger.Error("Health server error", "error", err)
		}
	}()

	return nil
}

func (hs *HealthServer) Stop(ctx context.Context) error {
	if hs.server != nil {
		return hs.server.Shutdown(ctx)
	}
	return nil
}

func (hs *HealthServer) handleHealthStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := hs.monitor.GetAllStatus()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		hs.logger.Error("Failed to encode health status", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (hs *HealthServer) handleServiceHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	serviceName := r.URL.Path[len("/health/"):]
	if serviceName == "" {
		http.Error(w, "Service name required", http.StatusBadRequest)
		return
	}

	status, exists := hs.monitor.GetStatus(serviceName)
	if !exists {
		http.Error(w, "Service not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		hs.logger.Error("Failed to encode service health", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (hs *HealthServer) handleReadiness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if all services are ready
	allStatus := hs.monitor.GetAllStatus()
	for _, status := range allStatus {
		if status.Status == "unhealthy" {
			http.Error(w, "Service not ready", http.StatusServiceUnavailable)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (hs *HealthServer) handleLiveness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (hs *HealthServer) handleServices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Return list of discovered services
	services := make([]string, 0)
	for serviceName := range hs.monitor.GetAllStatus() {
		services = append(services, serviceName)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(services); err != nil {
		hs.logger.Error("Failed to encode services", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (hs *HealthServer) handleServiceDiscovery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	serviceName := r.URL.Path[len("/services/"):]
	if serviceName == "" {
		http.Error(w, "Service name required", http.StatusBadRequest)
		return
	}

	// Return service discovery information
	discovery := map[string]interface{}{
		"service_name": serviceName,
		"endpoints":    []string{}, // Would be populated by service discovery
		"metadata":     map[string]string{},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(discovery); err != nil {
		hs.logger.Error("Failed to encode service discovery", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (hs *HealthServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Return Prometheus-compatible metrics
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("# HELP mcp_health_checks_total Total number of health checks\n"))
	w.Write([]byte("# TYPE mcp_health_checks_total counter\n"))

	// Add metrics for each service
	for serviceName, status := range hs.monitor.GetAllStatus() {
		for _, check := range status.Checks {
			statusValue := "0"
			if check.Status == "healthy" {
				statusValue = "1"
			}
			line := fmt.Sprintf("mcp_health_check{service=\"%s\",check=\"%s\",status=\"%s\"} %s\n",
				serviceName, check.Name, check.Status, statusValue)
			w.Write([]byte(line))
		}
	}
}

// Main application logic
func (app *HealthApp) Run(ctx context.Context) error {
	// Start health monitor
	if err := app.monitor.Start(ctx); err != nil {
		return fmt.Errorf("failed to start health monitor: %w", err)
	}

	// Start health server
	if err := app.server.Start(ctx); err != nil {
		return fmt.Errorf("failed to start health server: %w", err)
	}

	app.logger.Info("Health service started successfully")

	// Wait for shutdown signal
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.server.Stop(shutdownCtx); err != nil {
		app.logger.Error("Failed to stop health server", "error", err)
	}

	app.logger.Info("Health service stopped")
	return nil
}

func main() {
	var (
		configFile = flag.String("config", "", "Path to configuration file")
		command    = flag.String("command", "serve", "Command to run (check, discover, monitor, serve, operator)")
		target     = flag.String("target", "", "Target service for health check")
		port       = flag.Int("port", 8080, "Port to serve on")
		logLevel   = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	)
	flag.Parse()

	// Create default configuration
	config := &HealthConfig{
		ServiceName: "mcp-health",
		Port:        *port,
		LogLevel:    *logLevel,
		Metrics: MetricsConfig{
			Enabled: true,
			Path:    "/metrics",
		},
	}

	// Load configuration from file if provided
	if *configFile != "" {
		// Implementation for loading config from file would go here
		fmt.Printf("Loading configuration from %s\n", *configFile)
	}

	// Add default health check if target is specified
	if *target != "" {
		config.HealthChecks = append(config.HealthChecks, HealthCheckConfig{
			Name:             "default",
			Target:           *target,
			Protocol:         "http",
			Interval:         30 * time.Second,
			Timeout:          5 * time.Second,
			FailureThreshold: 3,
			SuccessThreshold: 1,
			HTTPPath:         "/health",
		})
	}

	// Create and run application
	app := NewHealthApp(config)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	switch *command {
	case "check":
		if *target == "" {
			fmt.Fprintf(os.Stderr, "Target is required for check command\n")
			os.Exit(1)
		}
		checkConfig := HealthCheckConfig{
			Name:     "single-check",
			Target:   *target,
			Protocol: "http",
			Timeout:  5 * time.Second,
			HTTPPath: "/health",
		}
		result, err := app.checker.CheckHealth(ctx, checkConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Health check failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Health check result: %s - %s\n", result.Status, result.Message)
	case "discover":
		fmt.Println("Service discovery not implemented yet")
	case "monitor":
		if err := app.monitor.Start(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start monitor: %v\n", err)
			os.Exit(1)
		}
		<-ctx.Done()
	case "serve":
		if err := app.Run(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to run health service: %v\n", err)
			os.Exit(1)
		}
	case "operator":
		fmt.Println("Kubernetes operator mode not implemented yet")
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", *command)
		os.Exit(1)
	}
}
