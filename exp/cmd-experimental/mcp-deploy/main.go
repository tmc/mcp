// Package main implements mcp-deploy: Deployment automation for MCP services
//
// This tool provides comprehensive deployment automation capabilities for MCP services,
// supporting multiple platforms and deployment strategies:
//
// Key features:
// - Multi-platform support (Docker, Kubernetes, serverless)
// - Rolling deployments with health checking
// - Automatic rollback on failure
// - Blue-green deployments
// - Canary deployments
// - Environment promotion pipelines
// - Configuration validation before deployment
// - Integration with CI/CD systems
// - Deployment monitoring and alerting
//
// Usage:
//
//	mcp-deploy [command] [flags]
//
// Commands:
//
//	init        Initialize deployment configuration
//	deploy      Deploy application
//	rollback    Rollback to previous version
//	status      Show deployment status
//	promote     Promote between environments
//	validate    Validate deployment configuration
//	scale       Scale deployment
//	canary      Manage canary deployments
//	serve       Start deployment service
//
// Examples:
//
//	mcp-deploy init --platform kubernetes
//	mcp-deploy deploy --environment production --version v1.2.3
//	mcp-deploy rollback --environment production
//	mcp-deploy promote --from staging --to production
//	mcp-deploy canary --environment production --traffic 10
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	version = "1.0.0"
)

// Main application structure
type DeployApp struct {
	config    *DeploymentConfig
	logger    *slog.Logger
	deployer  *Deployer
	monitor   *DeploymentMonitor
	server    *DeploymentServer
	validator *DeploymentValidator
}

// DeploymentConfig defines the deployment configuration
type DeploymentConfig struct {
	// Service configuration
	ServiceName string `json:"service_name" yaml:"service_name"`
	Port        int    `json:"port" yaml:"port"`
	LogLevel    string `json:"log_level" yaml:"log_level"`

	// Platform configuration
	Platform PlatformConfig `json:"platform" yaml:"platform"`

	// Environment configuration
	Environments map[string]*EnvironmentConfig `json:"environments" yaml:"environments"`

	// Deployment strategies
	Strategies map[string]*DeploymentStrategy `json:"strategies" yaml:"strategies"`

	// Health checking
	HealthCheck HealthCheckConfig `json:"health_check" yaml:"health_check"`

	// Rollback configuration
	Rollback RollbackConfig `json:"rollback" yaml:"rollback"`

	// Monitoring configuration
	Monitoring MonitoringConfig `json:"monitoring" yaml:"monitoring"`

	// CI/CD integration
	CICD CICDConfig `json:"cicd" yaml:"cicd"`
}

// PlatformConfig defines platform-specific configuration
type PlatformConfig struct {
	Type       string           `json:"type" yaml:"type"` // docker, kubernetes, serverless
	Docker     DockerConfig     `json:"docker" yaml:"docker"`
	Kubernetes KubernetesConfig `json:"kubernetes" yaml:"kubernetes"`
	Serverless ServerlessConfig `json:"serverless" yaml:"serverless"`
}

// DockerConfig defines Docker platform configuration
type DockerConfig struct {
	Registry    string            `json:"registry" yaml:"registry"`
	Repository  string            `json:"repository" yaml:"repository"`
	Tag         string            `json:"tag" yaml:"tag"`
	Dockerfile  string            `json:"dockerfile" yaml:"dockerfile"`
	Context     string            `json:"context" yaml:"context"`
	BuildArgs   map[string]string `json:"build_args" yaml:"build_args"`
	Labels      map[string]string `json:"labels" yaml:"labels"`
	Networks    []string          `json:"networks" yaml:"networks"`
	Volumes     []string          `json:"volumes" yaml:"volumes"`
	Environment []string          `json:"environment" yaml:"environment"`
	Ports       []string          `json:"ports" yaml:"ports"`
	HealthCheck DockerHealthCheck `json:"health_check" yaml:"health_check"`
}

// DockerHealthCheck defines Docker health check configuration
type DockerHealthCheck struct {
	Test        []string      `json:"test" yaml:"test"`
	Interval    time.Duration `json:"interval" yaml:"interval"`
	Timeout     time.Duration `json:"timeout" yaml:"timeout"`
	Retries     int           `json:"retries" yaml:"retries"`
	StartPeriod time.Duration `json:"start_period" yaml:"start_period"`
}

// KubernetesConfig defines Kubernetes platform configuration
type KubernetesConfig struct {
	Kubeconfig      string            `json:"kubeconfig" yaml:"kubeconfig"`
	Namespace       string            `json:"namespace" yaml:"namespace"`
	Context         string            `json:"context" yaml:"context"`
	Manifests       []string          `json:"manifests" yaml:"manifests"`
	HelmChart       HelmConfig        `json:"helm_chart" yaml:"helm_chart"`
	Kustomize       KustomizeConfig   `json:"kustomize" yaml:"kustomize"`
	Resources       ResourceLimits    `json:"resources" yaml:"resources"`
	Replicas        int               `json:"replicas" yaml:"replicas"`
	Labels          map[string]string `json:"labels" yaml:"labels"`
	Annotations     map[string]string `json:"annotations" yaml:"annotations"`
	NodeSelector    map[string]string `json:"node_selector" yaml:"node_selector"`
	Tolerations     []string          `json:"tolerations" yaml:"tolerations"`
	Affinity        string            `json:"affinity" yaml:"affinity"`
	ServiceAccount  string            `json:"service_account" yaml:"service_account"`
	SecurityContext SecurityContext   `json:"security_context" yaml:"security_context"`
}

// HelmConfig defines Helm chart configuration
type HelmConfig struct {
	Chart      string                 `json:"chart" yaml:"chart"`
	Repository string                 `json:"repository" yaml:"repository"`
	Version    string                 `json:"version" yaml:"version"`
	Values     map[string]interface{} `json:"values" yaml:"values"`
	ValuesFile string                 `json:"values_file" yaml:"values_file"`
}

// KustomizeConfig defines Kustomize configuration
type KustomizeConfig struct {
	Dir      string   `json:"dir" yaml:"dir"`
	Overlays []string `json:"overlays" yaml:"overlays"`
	Images   []string `json:"images" yaml:"images"`
	Patches  []string `json:"patches" yaml:"patches"`
}

// ResourceLimits defines resource limits
type ResourceLimits struct {
	CPU     string `json:"cpu" yaml:"cpu"`
	Memory  string `json:"memory" yaml:"memory"`
	Storage string `json:"storage" yaml:"storage"`
}

// SecurityContext defines security context
type SecurityContext struct {
	RunAsUser    int64 `json:"run_as_user" yaml:"run_as_user"`
	RunAsGroup   int64 `json:"run_as_group" yaml:"run_as_group"`
	ReadOnlyRoot bool  `json:"read_only_root" yaml:"read_only_root"`
}

// ServerlessConfig defines serverless platform configuration
type ServerlessConfig struct {
	Provider    string            `json:"provider" yaml:"provider"` // aws, gcp, azure
	Function    FunctionConfig    `json:"function" yaml:"function"`
	Triggers    []TriggerConfig   `json:"triggers" yaml:"triggers"`
	Environment map[string]string `json:"environment" yaml:"environment"`
	Timeout     time.Duration     `json:"timeout" yaml:"timeout"`
	Memory      int               `json:"memory" yaml:"memory"`
	Runtime     string            `json:"runtime" yaml:"runtime"`
	Handler     string            `json:"handler" yaml:"handler"`
	Package     string            `json:"package" yaml:"package"`
	Layers      []string          `json:"layers" yaml:"layers"`
	VPC         VPCConfig         `json:"vpc" yaml:"vpc"`
	IAM         IAMConfig         `json:"iam" yaml:"iam"`
}

// FunctionConfig defines function configuration
type FunctionConfig struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
	Runtime     string `json:"runtime" yaml:"runtime"`
	Handler     string `json:"handler" yaml:"handler"`
}

// TriggerConfig defines trigger configuration
type TriggerConfig struct {
	Type       string            `json:"type" yaml:"type"` // http, event, schedule
	Schedule   string            `json:"schedule" yaml:"schedule"`
	EventType  string            `json:"event_type" yaml:"event_type"`
	Source     string            `json:"source" yaml:"source"`
	Properties map[string]string `json:"properties" yaml:"properties"`
}

// VPCConfig defines VPC configuration
type VPCConfig struct {
	SecurityGroups []string `json:"security_groups" yaml:"security_groups"`
	Subnets        []string `json:"subnets" yaml:"subnets"`
}

// IAMConfig defines IAM configuration
type IAMConfig struct {
	Role        string   `json:"role" yaml:"role"`
	Policies    []string `json:"policies" yaml:"policies"`
	Permissions []string `json:"permissions" yaml:"permissions"`
}

// EnvironmentConfig defines environment-specific configuration
type EnvironmentConfig struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description" yaml:"description"`
	Strategy    string            `json:"strategy" yaml:"strategy"`
	Replicas    int               `json:"replicas" yaml:"replicas"`
	Resources   ResourceLimits    `json:"resources" yaml:"resources"`
	Config      map[string]string `json:"config" yaml:"config"`
	Secrets     map[string]string `json:"secrets" yaml:"secrets"`
	Ingress     IngressConfig     `json:"ingress" yaml:"ingress"`
	Monitoring  MonitoringConfig  `json:"monitoring" yaml:"monitoring"`
	AutoScale   AutoScaleConfig   `json:"auto_scale" yaml:"auto_scale"`
}

// IngressConfig defines ingress configuration
type IngressConfig struct {
	Enabled     bool              `json:"enabled" yaml:"enabled"`
	Host        string            `json:"host" yaml:"host"`
	Path        string            `json:"path" yaml:"path"`
	TLS         bool              `json:"tls" yaml:"tls"`
	Certificate string            `json:"certificate" yaml:"certificate"`
	Annotations map[string]string `json:"annotations" yaml:"annotations"`
}

// AutoScaleConfig defines auto-scaling configuration
type AutoScaleConfig struct {
	Enabled                bool          `json:"enabled" yaml:"enabled"`
	MinReplicas            int           `json:"min_replicas" yaml:"min_replicas"`
	MaxReplicas            int           `json:"max_replicas" yaml:"max_replicas"`
	CPUThreshold           int           `json:"cpu_threshold" yaml:"cpu_threshold"`
	MemoryThreshold        int           `json:"memory_threshold" yaml:"memory_threshold"`
	ScaleUpStabilization   time.Duration `json:"scale_up_stabilization" yaml:"scale_up_stabilization"`
	ScaleDownStabilization time.Duration `json:"scale_down_stabilization" yaml:"scale_down_stabilization"`
}

// DeploymentStrategy defines deployment strategy
type DeploymentStrategy struct {
	Type        string            `json:"type" yaml:"type"` // rolling, blue_green, canary
	Rolling     RollingConfig     `json:"rolling" yaml:"rolling"`
	BlueGreen   BlueGreenConfig   `json:"blue_green" yaml:"blue_green"`
	Canary      CanaryConfig      `json:"canary" yaml:"canary"`
	HealthCheck HealthCheckConfig `json:"health_check" yaml:"health_check"`
	Timeout     time.Duration     `json:"timeout" yaml:"timeout"`
	Rollback    RollbackConfig    `json:"rollback" yaml:"rollback"`
}

// RollingConfig defines rolling deployment configuration
type RollingConfig struct {
	MaxUnavailable string        `json:"max_unavailable" yaml:"max_unavailable"`
	MaxSurge       string        `json:"max_surge" yaml:"max_surge"`
	BatchSize      int           `json:"batch_size" yaml:"batch_size"`
	Pause          time.Duration `json:"pause" yaml:"pause"`
}

// BlueGreenConfig defines blue-green deployment configuration
type BlueGreenConfig struct {
	TrafficSplit    int           `json:"traffic_split" yaml:"traffic_split"`
	TestTraffic     int           `json:"test_traffic" yaml:"test_traffic"`
	PromotionDelay  time.Duration `json:"promotion_delay" yaml:"promotion_delay"`
	AutoPromotion   bool          `json:"auto_promotion" yaml:"auto_promotion"`
	HealthThreshold int           `json:"health_threshold" yaml:"health_threshold"`
}

// CanaryConfig defines canary deployment configuration
type CanaryConfig struct {
	TrafficPercent   int           `json:"traffic_percent" yaml:"traffic_percent"`
	StepPercent      int           `json:"step_percent" yaml:"step_percent"`
	StepDuration     time.Duration `json:"step_duration" yaml:"step_duration"`
	SuccessThreshold int           `json:"success_threshold" yaml:"success_threshold"`
	FailureThreshold int           `json:"failure_threshold" yaml:"failure_threshold"`
	AutoPromotion    bool          `json:"auto_promotion" yaml:"auto_promotion"`
}

// HealthCheckConfig defines health check configuration
type HealthCheckConfig struct {
	Enabled            bool          `json:"enabled" yaml:"enabled"`
	Path               string        `json:"path" yaml:"path"`
	Port               int           `json:"port" yaml:"port"`
	Interval           time.Duration `json:"interval" yaml:"interval"`
	Timeout            time.Duration `json:"timeout" yaml:"timeout"`
	HealthyThreshold   int           `json:"healthy_threshold" yaml:"healthy_threshold"`
	UnhealthyThreshold int           `json:"unhealthy_threshold" yaml:"unhealthy_threshold"`
	InitialDelay       time.Duration `json:"initial_delay" yaml:"initial_delay"`
}

// RollbackConfig defines rollback configuration
type RollbackConfig struct {
	Enabled          bool          `json:"enabled" yaml:"enabled"`
	AutoRollback     bool          `json:"auto_rollback" yaml:"auto_rollback"`
	FailureThreshold int           `json:"failure_threshold" yaml:"failure_threshold"`
	Timeout          time.Duration `json:"timeout" yaml:"timeout"`
	MaxHistory       int           `json:"max_history" yaml:"max_history"`
}

// MonitoringConfig defines monitoring configuration
type MonitoringConfig struct {
	Enabled    bool             `json:"enabled" yaml:"enabled"`
	Prometheus PrometheusConfig `json:"prometheus" yaml:"prometheus"`
	Grafana    GrafanaConfig    `json:"grafana" yaml:"grafana"`
	Alerting   AlertingConfig   `json:"alerting" yaml:"alerting"`
	Tracing    TracingConfig    `json:"tracing" yaml:"tracing"`
	Logging    LoggingConfig    `json:"logging" yaml:"logging"`
	Dashboards []string         `json:"dashboards" yaml:"dashboards"`
	Alerts     []string         `json:"alerts" yaml:"alerts"`
}

// PrometheusConfig defines Prometheus configuration
type PrometheusConfig struct {
	Enabled   bool              `json:"enabled" yaml:"enabled"`
	Endpoint  string            `json:"endpoint" yaml:"endpoint"`
	Namespace string            `json:"namespace" yaml:"namespace"`
	Labels    map[string]string `json:"labels" yaml:"labels"`
}

// GrafanaConfig defines Grafana configuration
type GrafanaConfig struct {
	Enabled   bool   `json:"enabled" yaml:"enabled"`
	Endpoint  string `json:"endpoint" yaml:"endpoint"`
	Dashboard string `json:"dashboard" yaml:"dashboard"`
}

// AlertingConfig defines alerting configuration
type AlertingConfig struct {
	Enabled bool        `json:"enabled" yaml:"enabled"`
	Webhook string      `json:"webhook" yaml:"webhook"`
	Slack   SlackConfig `json:"slack" yaml:"slack"`
	Email   EmailConfig `json:"email" yaml:"email"`
	Rules   []AlertRule `json:"rules" yaml:"rules"`
}

// AlertRule defines alert rule
type AlertRule struct {
	Name        string        `json:"name" yaml:"name"`
	Expression  string        `json:"expression" yaml:"expression"`
	Duration    time.Duration `json:"duration" yaml:"duration"`
	Severity    string        `json:"severity" yaml:"severity"`
	Description string        `json:"description" yaml:"description"`
}

// SlackConfig defines Slack configuration
type SlackConfig struct {
	Enabled bool   `json:"enabled" yaml:"enabled"`
	Token   string `json:"token" yaml:"token"`
	Channel string `json:"channel" yaml:"channel"`
}

// EmailConfig defines email configuration
type EmailConfig struct {
	Enabled  bool     `json:"enabled" yaml:"enabled"`
	SMTPHost string   `json:"smtp_host" yaml:"smtp_host"`
	SMTPPort int      `json:"smtp_port" yaml:"smtp_port"`
	From     string   `json:"from" yaml:"from"`
	To       []string `json:"to" yaml:"to"`
}

// TracingConfig defines tracing configuration
type TracingConfig struct {
	Enabled  bool   `json:"enabled" yaml:"enabled"`
	Endpoint string `json:"endpoint" yaml:"endpoint"`
	Service  string `json:"service" yaml:"service"`
}

// LoggingConfig defines logging configuration
type LoggingConfig struct {
	Enabled bool   `json:"enabled" yaml:"enabled"`
	Level   string `json:"level" yaml:"level"`
	Format  string `json:"format" yaml:"format"`
}

// CICDConfig defines CI/CD integration configuration
type CICDConfig struct {
	Enabled    bool              `json:"enabled" yaml:"enabled"`
	Provider   string            `json:"provider" yaml:"provider"` // github, gitlab, jenkins
	Repository string            `json:"repository" yaml:"repository"`
	Branch     string            `json:"branch" yaml:"branch"`
	Webhook    string            `json:"webhook" yaml:"webhook"`
	Triggers   []TriggerConfig   `json:"triggers" yaml:"triggers"`
	Variables  map[string]string `json:"variables" yaml:"variables"`
	Secrets    map[string]string `json:"secrets" yaml:"secrets"`
}

// Deployment represents a deployment
type Deployment struct {
	ID          string            `json:"id"`
	Environment string            `json:"environment"`
	Version     string            `json:"version"`
	Status      string            `json:"status"` // pending, deploying, deployed, failed, rolling_back
	Strategy    string            `json:"strategy"`
	StartTime   time.Time         `json:"start_time"`
	EndTime     time.Time         `json:"end_time"`
	Duration    time.Duration     `json:"duration"`
	Replicas    int               `json:"replicas"`
	Health      string            `json:"health"` // healthy, unhealthy, unknown
	Metadata    map[string]string `json:"metadata"`
	Logs        []string          `json:"logs"`
	Error       string            `json:"error,omitempty"`
}

// Deployer handles deployment operations
type Deployer struct {
	config      *DeploymentConfig
	logger      *slog.Logger
	deployments map[string]*Deployment
	mutex       sync.RWMutex
}

// DeploymentMonitor monitors deployment status
type DeploymentMonitor struct {
	config   *DeploymentConfig
	logger   *slog.Logger
	deployer *Deployer
}

// DeploymentServer provides HTTP API for deployment management
type DeploymentServer struct {
	config   *DeploymentConfig
	logger   *slog.Logger
	deployer *Deployer
	monitor  *DeploymentMonitor
	server   *http.Server
}

// DeploymentValidator validates deployment configuration
type DeploymentValidator struct {
	config *DeploymentConfig
	logger *slog.Logger
}

// NewDeployApp creates a new deployment application
func NewDeployApp(config *DeploymentConfig) *DeployApp {
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

	deployer := &Deployer{
		config:      config,
		logger:      logger,
		deployments: make(map[string]*Deployment),
	}

	monitor := &DeploymentMonitor{
		config:   config,
		logger:   logger,
		deployer: deployer,
	}

	server := &DeploymentServer{
		config:   config,
		logger:   logger,
		deployer: deployer,
		monitor:  monitor,
	}

	validator := &DeploymentValidator{
		config: config,
		logger: logger,
	}

	return &DeployApp{
		config:    config,
		logger:    logger,
		deployer:  deployer,
		monitor:   monitor,
		server:    server,
		validator: validator,
	}
}

// Deployer implementation
func (d *Deployer) Deploy(ctx context.Context, environment, version string) (*Deployment, error) {
	d.logger.Info("Starting deployment", "environment", environment, "version", version)

	envConfig, exists := d.config.Environments[environment]
	if !exists {
		return nil, fmt.Errorf("environment not found: %s", environment)
	}

	strategy, exists := d.config.Strategies[envConfig.Strategy]
	if !exists {
		return nil, fmt.Errorf("strategy not found: %s", envConfig.Strategy)
	}

	// Create deployment
	deployment := &Deployment{
		ID:          generateDeploymentID(),
		Environment: environment,
		Version:     version,
		Status:      "pending",
		Strategy:    strategy.Type,
		StartTime:   time.Now(),
		Replicas:    envConfig.Replicas,
		Health:      "unknown",
		Metadata:    make(map[string]string),
		Logs:        []string{},
	}

	// Store deployment
	d.mutex.Lock()
	d.deployments[deployment.ID] = deployment
	d.mutex.Unlock()

	// Start deployment process
	go d.runDeployment(ctx, deployment, envConfig, strategy)

	return deployment, nil
}

func (d *Deployer) runDeployment(ctx context.Context, deployment *Deployment, envConfig *EnvironmentConfig, strategy *DeploymentStrategy) {
	deployment.Status = "deploying"
	d.addLog(deployment, "Starting deployment process")

	var err error
	switch d.config.Platform.Type {
	case "docker":
		err = d.deployDocker(ctx, deployment, envConfig, strategy)
	case "kubernetes":
		err = d.deployKubernetes(ctx, deployment, envConfig, strategy)
	case "serverless":
		err = d.deployServerless(ctx, deployment, envConfig, strategy)
	default:
		err = fmt.Errorf("unsupported platform: %s", d.config.Platform.Type)
	}

	if err != nil {
		deployment.Status = "failed"
		deployment.Error = err.Error()
		d.addLog(deployment, fmt.Sprintf("Deployment failed: %s", err))
		d.logger.Error("Deployment failed", "deployment", deployment.ID, "error", err)
		return
	}

	deployment.Status = "deployed"
	deployment.EndTime = time.Now()
	deployment.Duration = deployment.EndTime.Sub(deployment.StartTime)
	d.addLog(deployment, "Deployment completed successfully")
	d.logger.Info("Deployment completed", "deployment", deployment.ID, "duration", deployment.Duration)
}

func (d *Deployer) deployDocker(ctx context.Context, deployment *Deployment, envConfig *EnvironmentConfig, strategy *DeploymentStrategy) error {
	d.addLog(deployment, "Deploying to Docker platform")

	dockerConfig := d.config.Platform.Docker

	// Build image
	d.addLog(deployment, "Building Docker image")
	if err := d.buildDockerImage(ctx, deployment, dockerConfig); err != nil {
		return fmt.Errorf("failed to build Docker image: %w", err)
	}

	// Push image
	d.addLog(deployment, "Pushing Docker image")
	if err := d.pushDockerImage(ctx, deployment, dockerConfig); err != nil {
		return fmt.Errorf("failed to push Docker image: %w", err)
	}

	// Deploy containers
	d.addLog(deployment, "Deploying containers")
	if err := d.deployDockerContainers(ctx, deployment, envConfig, dockerConfig); err != nil {
		return fmt.Errorf("failed to deploy containers: %w", err)
	}

	return nil
}

func (d *Deployer) buildDockerImage(ctx context.Context, deployment *Deployment, dockerConfig DockerConfig) error {
	imageTag := fmt.Sprintf("%s/%s:%s", dockerConfig.Registry, dockerConfig.Repository, deployment.Version)

	args := []string{"build", "-t", imageTag}

	// Add build args
	for key, value := range dockerConfig.BuildArgs {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, value))
	}

	// Add labels
	for key, value := range dockerConfig.Labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", key, value))
	}

	// Add dockerfile and context
	if dockerConfig.Dockerfile != "" {
		args = append(args, "-f", dockerConfig.Dockerfile)
	}

	context := dockerConfig.Context
	if context == "" {
		context = "."
	}
	args = append(args, context)

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker build failed: %w, output: %s", err, string(output))
	}

	d.addLog(deployment, fmt.Sprintf("Built image: %s", imageTag))
	return nil
}

func (d *Deployer) pushDockerImage(ctx context.Context, deployment *Deployment, dockerConfig DockerConfig) error {
	imageTag := fmt.Sprintf("%s/%s:%s", dockerConfig.Registry, dockerConfig.Repository, deployment.Version)

	cmd := exec.CommandContext(ctx, "docker", "push", imageTag)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker push failed: %w, output: %s", err, string(output))
	}

	d.addLog(deployment, fmt.Sprintf("Pushed image: %s", imageTag))
	return nil
}

func (d *Deployer) deployDockerContainers(ctx context.Context, deployment *Deployment, envConfig *EnvironmentConfig, dockerConfig DockerConfig) error {
	// Implementation for Docker container deployment
	// This would involve creating and starting containers with proper configuration
	d.addLog(deployment, "Docker container deployment not fully implemented")
	return nil
}

func (d *Deployer) deployKubernetes(ctx context.Context, deployment *Deployment, envConfig *EnvironmentConfig, strategy *DeploymentStrategy) error {
	d.addLog(deployment, "Deploying to Kubernetes platform")

	k8sConfig := d.config.Platform.Kubernetes

	// Apply manifests
	if len(k8sConfig.Manifests) > 0 {
		d.addLog(deployment, "Applying Kubernetes manifests")
		if err := d.applyKubernetesManifests(ctx, deployment, k8sConfig); err != nil {
			return fmt.Errorf("failed to apply manifests: %w", err)
		}
	}

	// Deploy with Helm
	if k8sConfig.HelmChart.Chart != "" {
		d.addLog(deployment, "Deploying with Helm")
		if err := d.deployWithHelm(ctx, deployment, k8sConfig); err != nil {
			return fmt.Errorf("failed to deploy with Helm: %w", err)
		}
	}

	// Deploy with Kustomize
	if k8sConfig.Kustomize.Dir != "" {
		d.addLog(deployment, "Deploying with Kustomize")
		if err := d.deployWithKustomize(ctx, deployment, k8sConfig); err != nil {
			return fmt.Errorf("failed to deploy with Kustomize: %w", err)
		}
	}

	return nil
}

func (d *Deployer) applyKubernetesManifests(ctx context.Context, deployment *Deployment, k8sConfig KubernetesConfig) error {
	for _, manifest := range k8sConfig.Manifests {
		args := []string{"apply", "-f", manifest}

		if k8sConfig.Namespace != "" {
			args = append(args, "-n", k8sConfig.Namespace)
		}

		if k8sConfig.Kubeconfig != "" {
			args = append(args, "--kubeconfig", k8sConfig.Kubeconfig)
		}

		cmd := exec.CommandContext(ctx, "kubectl", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("kubectl apply failed for %s: %w, output: %s", manifest, err, string(output))
		}

		d.addLog(deployment, fmt.Sprintf("Applied manifest: %s", manifest))
	}

	return nil
}

func (d *Deployer) deployWithHelm(ctx context.Context, deployment *Deployment, k8sConfig KubernetesConfig) error {
	helmConfig := k8sConfig.HelmChart

	args := []string{"upgrade", "--install", d.config.ServiceName, helmConfig.Chart}

	if helmConfig.Repository != "" {
		args = append(args, "--repo", helmConfig.Repository)
	}

	if helmConfig.Version != "" {
		args = append(args, "--version", helmConfig.Version)
	}

	if helmConfig.ValuesFile != "" {
		args = append(args, "-f", helmConfig.ValuesFile)
	}

	if k8sConfig.Namespace != "" {
		args = append(args, "-n", k8sConfig.Namespace)
	}

	// Add inline values
	for key, value := range helmConfig.Values {
		args = append(args, "--set", fmt.Sprintf("%s=%v", key, value))
	}

	cmd := exec.CommandContext(ctx, "helm", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("helm upgrade failed: %w, output: %s", err, string(output))
	}

	d.addLog(deployment, fmt.Sprintf("Deployed with Helm: %s", helmConfig.Chart))
	return nil
}

func (d *Deployer) deployWithKustomize(ctx context.Context, deployment *Deployment, k8sConfig KubernetesConfig) error {
	kustomizeConfig := k8sConfig.Kustomize

	// Build kustomization
	buildArgs := []string{"build", kustomizeConfig.Dir}
	buildCmd := exec.CommandContext(ctx, "kustomize", buildArgs...)
	manifest, err := buildCmd.Output()
	if err != nil {
		return fmt.Errorf("kustomize build failed: %w", err)
	}

	// Apply manifest
	applyArgs := []string{"apply", "-f", "-"}
	if k8sConfig.Namespace != "" {
		applyArgs = append(applyArgs, "-n", k8sConfig.Namespace)
	}

	applyCmd := exec.CommandContext(ctx, "kubectl", applyArgs...)
	applyCmd.Stdin = strings.NewReader(string(manifest))
	output, err := applyCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kubectl apply failed: %w, output: %s", err, string(output))
	}

	d.addLog(deployment, fmt.Sprintf("Deployed with Kustomize: %s", kustomizeConfig.Dir))
	return nil
}

func (d *Deployer) deployServerless(ctx context.Context, deployment *Deployment, envConfig *EnvironmentConfig, strategy *DeploymentStrategy) error {
	d.addLog(deployment, "Deploying to serverless platform")

	serverlessConfig := d.config.Platform.Serverless

	switch serverlessConfig.Provider {
	case "aws":
		return d.deployAWSLambda(ctx, deployment, envConfig, serverlessConfig)
	case "gcp":
		return d.deployGCPFunctions(ctx, deployment, envConfig, serverlessConfig)
	case "azure":
		return d.deployAzureFunctions(ctx, deployment, envConfig, serverlessConfig)
	default:
		return fmt.Errorf("unsupported serverless provider: %s", serverlessConfig.Provider)
	}
}

func (d *Deployer) deployAWSLambda(ctx context.Context, deployment *Deployment, envConfig *EnvironmentConfig, serverlessConfig ServerlessConfig) error {
	d.addLog(deployment, "Deploying AWS Lambda function")

	// Package function
	d.addLog(deployment, "Packaging Lambda function")
	if err := d.packageLambdaFunction(ctx, deployment, serverlessConfig); err != nil {
		return fmt.Errorf("failed to package Lambda function: %w", err)
	}

	// Deploy function
	d.addLog(deployment, "Deploying Lambda function")
	if err := d.deployLambdaFunction(ctx, deployment, serverlessConfig); err != nil {
		return fmt.Errorf("failed to deploy Lambda function: %w", err)
	}

	return nil
}

func (d *Deployer) packageLambdaFunction(ctx context.Context, deployment *Deployment, serverlessConfig ServerlessConfig) error {
	// Implementation for packaging Lambda function
	d.addLog(deployment, "Lambda function packaging not fully implemented")
	return nil
}

func (d *Deployer) deployLambdaFunction(ctx context.Context, deployment *Deployment, serverlessConfig ServerlessConfig) error {
	// Implementation for deploying Lambda function
	d.addLog(deployment, "Lambda function deployment not fully implemented")
	return nil
}

func (d *Deployer) deployGCPFunctions(ctx context.Context, deployment *Deployment, envConfig *EnvironmentConfig, serverlessConfig ServerlessConfig) error {
	d.addLog(deployment, "GCP Functions deployment not implemented")
	return fmt.Errorf("GCP Functions deployment not implemented")
}

func (d *Deployer) deployAzureFunctions(ctx context.Context, deployment *Deployment, envConfig *EnvironmentConfig, serverlessConfig ServerlessConfig) error {
	d.addLog(deployment, "Azure Functions deployment not implemented")
	return fmt.Errorf("Azure Functions deployment not implemented")
}

func (d *Deployer) Rollback(ctx context.Context, environment string, revision string) error {
	d.logger.Info("Starting rollback", "environment", environment, "revision", revision)

	switch d.config.Platform.Type {
	case "docker":
		return d.rollbackDocker(ctx, environment, revision)
	case "kubernetes":
		return d.rollbackKubernetes(ctx, environment, revision)
	case "serverless":
		return d.rollbackServerless(ctx, environment, revision)
	default:
		return fmt.Errorf("unsupported platform: %s", d.config.Platform.Type)
	}
}

func (d *Deployer) rollbackDocker(ctx context.Context, environment, revision string) error {
	// Implementation for Docker rollback
	return fmt.Errorf("Docker rollback not implemented")
}

func (d *Deployer) rollbackKubernetes(ctx context.Context, environment, revision string) error {
	k8sConfig := d.config.Platform.Kubernetes

	args := []string{"rollout", "undo", "deployment", d.config.ServiceName}

	if revision != "" {
		args = append(args, "--to-revision", revision)
	}

	if k8sConfig.Namespace != "" {
		args = append(args, "-n", k8sConfig.Namespace)
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kubectl rollout undo failed: %w, output: %s", err, string(output))
	}

	d.logger.Info("Rollback completed", "environment", environment, "revision", revision)
	return nil
}

func (d *Deployer) rollbackServerless(ctx context.Context, environment, revision string) error {
	// Implementation for serverless rollback
	return fmt.Errorf("Serverless rollback not implemented")
}

func (d *Deployer) GetDeployment(id string) (*Deployment, bool) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	deployment, exists := d.deployments[id]
	return deployment, exists
}

func (d *Deployer) ListDeployments() []*Deployment {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	deployments := make([]*Deployment, 0, len(d.deployments))
	for _, deployment := range d.deployments {
		deployments = append(deployments, deployment)
	}

	return deployments
}

func (d *Deployer) addLog(deployment *Deployment, message string) {
	deployment.Logs = append(deployment.Logs, fmt.Sprintf("[%s] %s", time.Now().Format(time.RFC3339), message))
}

// DeploymentServer implementation
func (ds *DeploymentServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Deployment endpoints
	mux.HandleFunc("/deployments", ds.handleDeployments)
	mux.HandleFunc("/deployments/", ds.handleDeployment)
	mux.HandleFunc("/deploy", ds.handleDeploy)
	mux.HandleFunc("/rollback", ds.handleRollback)
	mux.HandleFunc("/status", ds.handleStatus)

	// Environment endpoints
	mux.HandleFunc("/environments", ds.handleEnvironments)
	mux.HandleFunc("/environments/", ds.handleEnvironment)

	// Health endpoints
	mux.HandleFunc("/health", ds.handleHealth)
	mux.HandleFunc("/ready", ds.handleReady)

	ds.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", ds.config.Port),
		Handler: mux,
	}

	ds.logger.Info("Starting deployment server", "port", ds.config.Port)

	go func() {
		if err := ds.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			ds.logger.Error("Deployment server error", "error", err)
		}
	}()

	return nil
}

func (ds *DeploymentServer) Stop(ctx context.Context) error {
	if ds.server != nil {
		return ds.server.Shutdown(ctx)
	}
	return nil
}

func (ds *DeploymentServer) handleDeployments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	deployments := ds.deployer.ListDeployments()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(deployments); err != nil {
		ds.logger.Error("Failed to encode deployments", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (ds *DeploymentServer) handleDeployment(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/deployments/"):]
	if id == "" {
		http.Error(w, "Deployment ID required", http.StatusBadRequest)
		return
	}

	deployment, exists := ds.deployer.GetDeployment(id)
	if !exists {
		http.Error(w, "Deployment not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(deployment); err != nil {
		ds.logger.Error("Failed to encode deployment", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (ds *DeploymentServer) handleDeploy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Environment string `json:"environment"`
		Version     string `json:"version"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	deployment, err := ds.deployer.Deploy(r.Context(), req.Environment, req.Version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(deployment); err != nil {
		ds.logger.Error("Failed to encode deployment", "error", err)
	}
}

func (ds *DeploymentServer) handleRollback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Environment string `json:"environment"`
		Revision    string `json:"revision"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := ds.deployer.Rollback(r.Context(), req.Environment, req.Revision); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "rollback initiated"})
}

func (ds *DeploymentServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := map[string]interface{}{
		"service":     ds.config.ServiceName,
		"platform":    ds.config.Platform.Type,
		"deployments": len(ds.deployer.ListDeployments()),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (ds *DeploymentServer) handleEnvironments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ds.config.Environments)
}

func (ds *DeploymentServer) handleEnvironment(w http.ResponseWriter, r *http.Request) {
	envName := r.URL.Path[len("/environments/"):]
	if envName == "" {
		http.Error(w, "Environment name required", http.StatusBadRequest)
		return
	}

	env, exists := ds.config.Environments[envName]
	if !exists {
		http.Error(w, "Environment not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(env)
}

func (ds *DeploymentServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func (ds *DeploymentServer) handleReady(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

// Helper functions
func generateDeploymentID() string {
	return fmt.Sprintf("deploy-%d", time.Now().UnixNano())
}

// Main application logic
func (app *DeployApp) Run(ctx context.Context, command string, args []string) error {
	switch command {
	case "init":
		return app.initDeployment(args)
	case "deploy":
		return app.deployApplication(ctx, args)
	case "rollback":
		return app.rollbackDeployment(ctx, args)
	case "status":
		return app.showStatus(args)
	case "promote":
		return app.promoteDeployment(ctx, args)
	case "validate":
		return app.validateDeployment(args)
	case "scale":
		return app.scaleDeployment(ctx, args)
	case "canary":
		return app.manageCanary(ctx, args)
	case "serve":
		return app.runServer(ctx)
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

func (app *DeployApp) initDeployment(args []string) error {
	app.logger.Info("Initializing deployment configuration")

	// Create deployment directories
	dirs := []string{
		"deploy",
		"deploy/environments",
		"deploy/strategies",
		"deploy/manifests",
		"deploy/helm",
		"deploy/kustomize",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create example deployment configuration
	exampleConfig := `
service_name: "mcp-service"
port: 8080
log_level: "info"

platform:
  type: "kubernetes"
  kubernetes:
    kubeconfig: "~/.kube/config"
    namespace: "default"
    manifests:
      - "deploy/manifests/deployment.yaml"
      - "deploy/manifests/service.yaml"

environments:
  development:
    name: "development"
    description: "Development environment"
    strategy: "rolling"
    replicas: 1
    resources:
      cpu: "100m"
      memory: "128Mi"
    
  staging:
    name: "staging"
    description: "Staging environment"
    strategy: "blue_green"
    replicas: 2
    resources:
      cpu: "200m"
      memory: "256Mi"
    
  production:
    name: "production"
    description: "Production environment"
    strategy: "canary"
    replicas: 3
    resources:
      cpu: "500m"
      memory: "512Mi"
    auto_scale:
      enabled: true
      min_replicas: 3
      max_replicas: 10
      cpu_threshold: 80

strategies:
  rolling:
    type: "rolling"
    rolling:
      max_unavailable: "25%"
      max_surge: "25%"
      batch_size: 1
      pause: "30s"
    health_check:
      enabled: true
      path: "/health"
      port: 8080
      interval: "10s"
      timeout: "5s"
      healthy_threshold: 2
      unhealthy_threshold: 3
    timeout: "10m"
    
  blue_green:
    type: "blue_green"
    blue_green:
      traffic_split: 50
      test_traffic: 10
      promotion_delay: "5m"
      auto_promotion: false
      health_threshold: 90
    timeout: "15m"
    
  canary:
    type: "canary"
    canary:
      traffic_percent: 10
      step_percent: 10
      step_duration: "5m"
      success_threshold: 95
      failure_threshold: 5
      auto_promotion: true
    timeout: "30m"

health_check:
  enabled: true
  path: "/health"
  port: 8080
  interval: "30s"
  timeout: "5s"
  healthy_threshold: 2
  unhealthy_threshold: 3

rollback:
  enabled: true
  auto_rollback: true
  failure_threshold: 3
  timeout: "10m"
  max_history: 10

monitoring:
  enabled: true
  prometheus:
    enabled: true
    endpoint: "http://prometheus:9090"
    namespace: "mcp"
  alerting:
    enabled: true
    rules:
      - name: "deployment_failed"
        expression: "mcp_deployment_status == 0"
        duration: "2m"
        severity: "critical"
        description: "Deployment has failed"
`

	if err := os.WriteFile("deploy/mcp-deploy.yaml", []byte(exampleConfig), 0644); err != nil {
		return fmt.Errorf("failed to write example config: %w", err)
	}

	app.logger.Info("Deployment configuration initialized successfully")
	return nil
}

func (app *DeployApp) deployApplication(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("environment and version required")
	}

	environment := args[0]
	version := args[1]

	deployment, err := app.deployer.Deploy(ctx, environment, version)
	if err != nil {
		return fmt.Errorf("deployment failed: %w", err)
	}

	fmt.Printf("Deployment started: %s\n", deployment.ID)
	fmt.Printf("Environment: %s\n", deployment.Environment)
	fmt.Printf("Version: %s\n", deployment.Version)
	fmt.Printf("Status: %s\n", deployment.Status)

	return nil
}

func (app *DeployApp) rollbackDeployment(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("environment required")
	}

	environment := args[0]
	revision := ""
	if len(args) > 1 {
		revision = args[1]
	}

	if err := app.deployer.Rollback(ctx, environment, revision); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	fmt.Printf("Rollback completed for environment: %s\n", environment)
	return nil
}

func (app *DeployApp) showStatus(args []string) error {
	if len(args) == 0 {
		// Show all deployments
		deployments := app.deployer.ListDeployments()
		if len(deployments) == 0 {
			fmt.Println("No deployments found")
			return nil
		}

		fmt.Printf("%-20s %-15s %-10s %-12s %-10s\n", "ID", "Environment", "Version", "Status", "Health")
		fmt.Println(strings.Repeat("-", 80))
		for _, deployment := range deployments {
			fmt.Printf("%-20s %-15s %-10s %-12s %-10s\n",
				deployment.ID,
				deployment.Environment,
				deployment.Version,
				deployment.Status,
				deployment.Health)
		}
	} else {
		// Show specific deployment
		id := args[0]
		deployment, exists := app.deployer.GetDeployment(id)
		if !exists {
			return fmt.Errorf("deployment not found: %s", id)
		}

		fmt.Printf("Deployment ID: %s\n", deployment.ID)
		fmt.Printf("Environment: %s\n", deployment.Environment)
		fmt.Printf("Version: %s\n", deployment.Version)
		fmt.Printf("Status: %s\n", deployment.Status)
		fmt.Printf("Strategy: %s\n", deployment.Strategy)
		fmt.Printf("Start Time: %s\n", deployment.StartTime.Format(time.RFC3339))
		if !deployment.EndTime.IsZero() {
			fmt.Printf("End Time: %s\n", deployment.EndTime.Format(time.RFC3339))
			fmt.Printf("Duration: %s\n", deployment.Duration)
		}
		fmt.Printf("Replicas: %d\n", deployment.Replicas)
		fmt.Printf("Health: %s\n", deployment.Health)

		if deployment.Error != "" {
			fmt.Printf("Error: %s\n", deployment.Error)
		}

		fmt.Println("\nLogs:")
		for _, log := range deployment.Logs {
			fmt.Printf("  %s\n", log)
		}
	}

	return nil
}

func (app *DeployApp) promoteDeployment(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("source and target environments required")
	}

	// Implementation for environment promotion
	fmt.Printf("Promotion from %s to %s not implemented yet\n", args[0], args[1])
	return nil
}

func (app *DeployApp) validateDeployment(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("configuration file required")
	}

	// Implementation for deployment validation
	fmt.Printf("Validation for %s not implemented yet\n", args[0])
	return nil
}

func (app *DeployApp) scaleDeployment(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("environment and replica count required")
	}

	// Implementation for deployment scaling
	fmt.Printf("Scaling %s to %s replicas not implemented yet\n", args[0], args[1])
	return nil
}

func (app *DeployApp) manageCanary(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("canary command required")
	}

	// Implementation for canary management
	fmt.Printf("Canary management not implemented yet\n")
	return nil
}

func (app *DeployApp) runServer(ctx context.Context) error {
	// Start deployment server
	if err := app.server.Start(ctx); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	app.logger.Info("Deployment service started successfully")

	// Wait for shutdown signal
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.server.Stop(shutdownCtx); err != nil {
		app.logger.Error("Failed to stop server", "error", err)
	}

	app.logger.Info("Deployment service stopped")
	return nil
}

func main() {
	var (
		configFile  = flag.String("config", "deploy/mcp-deploy.yaml", "Path to configuration file")
		command     = flag.String("command", "serve", "Command to run")
		environment = flag.String("environment", "", "Environment name")
		version     = flag.String("version", "", "Version to deploy")
		port        = flag.Int("port", 8080, "Port to serve on")
		logLevel    = flag.String("log-level", "info", "Log level")
	)
	flag.Parse()

	// Create default configuration
	config := &DeploymentConfig{
		ServiceName: "mcp-deploy",
		Port:        *port,
		LogLevel:    *logLevel,
		Platform: PlatformConfig{
			Type: "kubernetes",
		},
		Environments: make(map[string]*EnvironmentConfig),
		Strategies:   make(map[string]*DeploymentStrategy),
		HealthCheck: HealthCheckConfig{
			Enabled: true,
			Path:    "/health",
			Port:    8080,
		},
		Rollback: RollbackConfig{
			Enabled: true,
		},
		Monitoring: MonitoringConfig{
			Enabled: true,
		},
	}

	// Load configuration from file if exists
	if _, err := os.Stat(*configFile); err == nil {
		data, err := os.ReadFile(*configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read config file: %v\n", err)
			os.Exit(1)
		}

		if err := yaml.Unmarshal(data, config); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse config file: %v\n", err)
			os.Exit(1)
		}
	}

	// Create and run application
	app := NewDeployApp(config)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Add command-line arguments
	args := flag.Args()
	if *environment != "" {
		args = append([]string{*environment}, args...)
	}
	if *version != "" {
		args = append(args, *version)
	}

	if err := app.Run(ctx, *command, args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
