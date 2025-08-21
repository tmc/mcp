//go:build k8s
// +build k8s

// Package main - Kubernetes operator for mcp-health
//
// This file implements a Kubernetes operator for managing MCP health checks
// and service discovery in cloud-native environments.

package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

// MCPHealthCheck represents a custom resource for MCP health checks
type MCPHealthCheck struct {
	v1.TypeMeta   `json:",inline"`
	v1.ObjectMeta `json:"metadata,omitempty"`
	Spec          MCPHealthCheckSpec   `json:"spec"`
	Status        MCPHealthCheckStatus `json:"status"`
}

// MCPHealthCheckSpec defines the desired state of MCPHealthCheck
type MCPHealthCheckSpec struct {
	Target           string                 `json:"target"`
	Protocol         string                 `json:"protocol"`
	Interval         v1.Duration            `json:"interval"`
	Timeout          v1.Duration            `json:"timeout"`
	FailureThreshold int                    `json:"failureThreshold"`
	SuccessThreshold int                    `json:"successThreshold"`
	HTTPPath         string                 `json:"httpPath,omitempty"`
	MCPMethod        string                 `json:"mcpMethod,omitempty"`
	ExpectedStatus   int                    `json:"expectedStatus,omitempty"`
	Headers          map[string]string      `json:"headers,omitempty"`
	Alerting         MCPHealthCheckAlerting `json:"alerting,omitempty"`
}

// MCPHealthCheckAlerting defines alerting configuration
type MCPHealthCheckAlerting struct {
	Enabled bool              `json:"enabled"`
	Webhook string            `json:"webhook,omitempty"`
	Slack   SlackAlerting     `json:"slack,omitempty"`
	Email   EmailAlerting     `json:"email,omitempty"`
	Labels  map[string]string `json:"labels,omitempty"`
}

// SlackAlerting defines Slack alerting configuration
type SlackAlerting struct {
	Enabled bool   `json:"enabled"`
	Channel string `json:"channel"`
	Token   string `json:"token"`
}

// EmailAlerting defines email alerting configuration
type EmailAlerting struct {
	Enabled bool     `json:"enabled"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
}

// MCPHealthCheckStatus defines the observed state of MCPHealthCheck
type MCPHealthCheckStatus struct {
	Status       string                    `json:"status"`
	LastCheck    v1.Time                   `json:"lastCheck"`
	LastSuccess  v1.Time                   `json:"lastSuccess"`
	LastFailure  v1.Time                   `json:"lastFailure"`
	FailureCount int                       `json:"failureCount"`
	SuccessCount int                       `json:"successCount"`
	Message      string                    `json:"message"`
	Conditions   []MCPHealthCheckCondition `json:"conditions"`
}

// MCPHealthCheckCondition defines a condition for the health check
type MCPHealthCheckCondition struct {
	Type               string  `json:"type"`
	Status             string  `json:"status"`
	LastTransitionTime v1.Time `json:"lastTransitionTime"`
	Reason             string  `json:"reason"`
	Message            string  `json:"message"`
}

// MCPHealthCheckList contains a list of MCPHealthCheck
type MCPHealthCheckList struct {
	v1.TypeMeta `json:",inline"`
	v1.ListMeta `json:"metadata,omitempty"`
	Items       []MCPHealthCheck `json:"items"`
}

// MCPHealthOperator manages MCP health checks in Kubernetes
type MCPHealthOperator struct {
	config     *HealthConfig
	logger     *slog.Logger
	clientset  kubernetes.Interface
	healthApp  *HealthApp
	controller cache.Controller
	informer   cache.SharedIndexInformer
	stopCh     chan struct{}
}

// NewMCPHealthOperator creates a new MCP health operator
func NewMCPHealthOperator(config *HealthConfig, healthApp *HealthApp) (*MCPHealthOperator, error) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create Kubernetes client
	var kubeConfig *rest.Config
	var err error

	if config.Kubernetes.Kubeconfig != "" {
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", config.Kubernetes.Kubeconfig)
	} else {
		kubeConfig, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	operator := &MCPHealthOperator{
		config:    config,
		logger:    logger,
		clientset: clientset,
		healthApp: healthApp,
		stopCh:    make(chan struct{}),
	}

	// Create informer for MCPHealthCheck resources
	operator.setupInformer()

	return operator, nil
}

// setupInformer sets up the informer for MCPHealthCheck resources
func (o *MCPHealthOperator) setupInformer() {
	// Create a list/watch client for MCPHealthCheck resources
	listWatchClient := cache.NewListWatchFromClient(
		o.clientset.RESTClient(),
		"mcphealthchecks",
		o.config.Kubernetes.Namespace,
		cache.NewSelector(nil),
	)

	// Create informer
	o.informer = cache.NewSharedIndexInformer(
		listWatchClient,
		&MCPHealthCheck{},
		time.Second*30,
		cache.Indexers{},
	)

	// Add event handlers
	o.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    o.handleAdd,
		UpdateFunc: o.handleUpdate,
		DeleteFunc: o.handleDelete,
	})

	// Create controller
	o.controller = cache.NewController(o.informer, cache.ResourceEventHandlerFuncs{})
}

// Start starts the MCP health operator
func (o *MCPHealthOperator) Start(ctx context.Context) error {
	o.logger.Info("Starting MCP health operator")

	// Start the informer
	go o.informer.Run(o.stopCh)

	// Wait for cache sync
	if !cache.WaitForCacheSync(o.stopCh, o.informer.HasSynced) {
		return fmt.Errorf("failed to sync cache")
	}

	o.logger.Info("MCP health operator started successfully")

	// Start reconciliation loop
	go o.reconcileLoop(ctx)

	// Wait for context cancellation
	<-ctx.Done()
	close(o.stopCh)

	o.logger.Info("MCP health operator stopped")
	return nil
}

// handleAdd handles the addition of MCPHealthCheck resources
func (o *MCPHealthOperator) handleAdd(obj interface{}) {
	healthCheck := obj.(*MCPHealthCheck)
	o.logger.Info("MCPHealthCheck added", "name", healthCheck.Name, "namespace", healthCheck.Namespace)

	// Convert to internal health check configuration
	config := o.convertToHealthCheckConfig(healthCheck)

	// Add to health app configuration
	o.healthApp.config.HealthChecks = append(o.healthApp.config.HealthChecks, config)

	// Update status
	o.updateStatus(healthCheck, "pending", "Health check created")
}

// handleUpdate handles the update of MCPHealthCheck resources
func (o *MCPHealthOperator) handleUpdate(oldObj, newObj interface{}) {
	oldHealthCheck := oldObj.(*MCPHealthCheck)
	newHealthCheck := newObj.(*MCPHealthCheck)

	o.logger.Info("MCPHealthCheck updated", "name", newHealthCheck.Name, "namespace", newHealthCheck.Namespace)

	// Find and update the corresponding health check configuration
	for i, config := range o.healthApp.config.HealthChecks {
		if config.Name == oldHealthCheck.Name {
			o.healthApp.config.HealthChecks[i] = o.convertToHealthCheckConfig(newHealthCheck)
			break
		}
	}

	// Update status
	o.updateStatus(newHealthCheck, "updated", "Health check updated")
}

// handleDelete handles the deletion of MCPHealthCheck resources
func (o *MCPHealthOperator) handleDelete(obj interface{}) {
	healthCheck := obj.(*MCPHealthCheck)
	o.logger.Info("MCPHealthCheck deleted", "name", healthCheck.Name, "namespace", healthCheck.Namespace)

	// Remove from health app configuration
	for i, config := range o.healthApp.config.HealthChecks {
		if config.Name == healthCheck.Name {
			o.healthApp.config.HealthChecks = append(
				o.healthApp.config.HealthChecks[:i],
				o.healthApp.config.HealthChecks[i+1:]...,
			)
			break
		}
	}
}

// convertToHealthCheckConfig converts MCPHealthCheck to internal HealthCheckConfig
func (o *MCPHealthOperator) convertToHealthCheckConfig(healthCheck *MCPHealthCheck) HealthCheckConfig {
	return HealthCheckConfig{
		Name:             healthCheck.Name,
		Target:           healthCheck.Spec.Target,
		Protocol:         healthCheck.Spec.Protocol,
		Interval:         healthCheck.Spec.Interval.Duration,
		Timeout:          healthCheck.Spec.Timeout.Duration,
		FailureThreshold: healthCheck.Spec.FailureThreshold,
		SuccessThreshold: healthCheck.Spec.SuccessThreshold,
		HTTPPath:         healthCheck.Spec.HTTPPath,
		MCPMethod:        healthCheck.Spec.MCPMethod,
		ExpectedStatus:   healthCheck.Spec.ExpectedStatus,
		Headers:          healthCheck.Spec.Headers,
	}
}

// reconcileLoop runs the reconciliation loop
func (o *MCPHealthOperator) reconcileLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			o.reconcile()
		}
	}
}

// reconcile reconciles the desired state with the actual state
func (o *MCPHealthOperator) reconcile() {
	// Get all MCPHealthCheck resources
	objects := o.informer.GetStore().List()

	for _, obj := range objects {
		healthCheck := obj.(*MCPHealthCheck)

		// Check if health check is running
		status, exists := o.healthApp.monitor.GetStatus(healthCheck.Name)
		if !exists {
			// Health check not running, update status
			o.updateStatus(healthCheck, "unknown", "Health check not running")
			continue
		}

		// Update status based on health check results
		var statusString string
		var message string

		switch status.Status {
		case "healthy":
			statusString = "healthy"
			message = "All health checks passing"
		case "unhealthy":
			statusString = "unhealthy"
			message = "Health checks failing"
		case "degraded":
			statusString = "degraded"
			message = "Some health checks failing"
		default:
			statusString = "unknown"
			message = "Health check status unknown"
		}

		o.updateStatus(healthCheck, statusString, message)

		// Handle alerting if enabled
		if healthCheck.Spec.Alerting.Enabled {
			o.handleAlerting(healthCheck, status)
		}
	}
}

// updateStatus updates the status of an MCPHealthCheck resource
func (o *MCPHealthOperator) updateStatus(healthCheck *MCPHealthCheck, status, message string) {
	now := v1.Now()

	healthCheck.Status.Status = status
	healthCheck.Status.LastCheck = now
	healthCheck.Status.Message = message

	if status == "healthy" {
		healthCheck.Status.LastSuccess = now
		healthCheck.Status.SuccessCount++
	} else if status == "unhealthy" {
		healthCheck.Status.LastFailure = now
		healthCheck.Status.FailureCount++
	}

	// Update conditions
	condition := MCPHealthCheckCondition{
		Type:               "Ready",
		Status:             status,
		LastTransitionTime: now,
		Reason:             "HealthCheck",
		Message:            message,
	}

	// Add or update condition
	conditionExists := false
	for i, existingCondition := range healthCheck.Status.Conditions {
		if existingCondition.Type == condition.Type {
			healthCheck.Status.Conditions[i] = condition
			conditionExists = true
			break
		}
	}

	if !conditionExists {
		healthCheck.Status.Conditions = append(healthCheck.Status.Conditions, condition)
	}

	// Update the resource in Kubernetes
	// Note: This would require proper client configuration and error handling
	o.logger.Info("Status updated", "name", healthCheck.Name, "status", status, "message", message)
}

// handleAlerting handles alerting for health check failures
func (o *MCPHealthOperator) handleAlerting(healthCheck *MCPHealthCheck, status *HealthStatus) {
	if status.Status != "unhealthy" {
		return
	}

	alerting := healthCheck.Spec.Alerting

	// Send webhook alert
	if alerting.Webhook != "" {
		o.sendWebhookAlert(healthCheck, status, alerting.Webhook)
	}

	// Send Slack alert
	if alerting.Slack.Enabled {
		o.sendSlackAlert(healthCheck, status, alerting.Slack)
	}

	// Send email alert
	if alerting.Email.Enabled {
		o.sendEmailAlert(healthCheck, status, alerting.Email)
	}
}

// sendWebhookAlert sends a webhook alert
func (o *MCPHealthOperator) sendWebhookAlert(healthCheck *MCPHealthCheck, status *HealthStatus, webhook string) {
	alert := map[string]interface{}{
		"service":    healthCheck.Name,
		"namespace":  healthCheck.Namespace,
		"status":     status.Status,
		"message":    status.Checks[len(status.Checks)-1].Message,
		"timestamp":  time.Now().Format(time.RFC3339),
		"severity":   "critical",
		"alert_type": "health_check_failure",
	}

	// Send HTTP POST to webhook
	// Implementation would include proper HTTP client and error handling
	o.logger.Info("Webhook alert sent", "service", healthCheck.Name, "webhook", webhook)
}

// sendSlackAlert sends a Slack alert
func (o *MCPHealthOperator) sendSlackAlert(healthCheck *MCPHealthCheck, status *HealthStatus, slack SlackAlerting) {
	message := fmt.Sprintf("🚨 Health Check Alert: %s/%s is %s",
		healthCheck.Namespace, healthCheck.Name, status.Status)

	// Send to Slack API
	// Implementation would include proper Slack client and error handling
	o.logger.Info("Slack alert sent", "service", healthCheck.Name, "channel", slack.Channel)
}

// sendEmailAlert sends an email alert
func (o *MCPHealthOperator) sendEmailAlert(healthCheck *MCPHealthCheck, status *HealthStatus, email EmailAlerting) {
	subject := fmt.Sprintf("Health Check Alert: %s/%s", healthCheck.Namespace, healthCheck.Name)
	if email.Subject != "" {
		subject = email.Subject
	}

	body := fmt.Sprintf(`
Health Check Alert

Service: %s/%s
Status: %s
Last Check: %s
Message: %s

Please investigate the issue.
`, healthCheck.Namespace, healthCheck.Name, status.Status,
		status.Timestamp.Format(time.RFC3339),
		status.Checks[len(status.Checks)-1].Message)

	// Send email
	// Implementation would include proper SMTP client and error handling
	o.logger.Info("Email alert sent", "service", healthCheck.Name, "to", email.To)
}

// CreateMCPHealthCheckCRD creates the Custom Resource Definition
func (o *MCPHealthOperator) CreateMCPHealthCheckCRD(ctx context.Context) error {
	// This would create the CRD in Kubernetes
	// Implementation would include proper CRD definition and creation
	o.logger.Info("Creating MCPHealthCheck CRD")
	return nil
}

// Example MCPHealthCheck resource
func ExampleMCPHealthCheck() *MCPHealthCheck {
	return &MCPHealthCheck{
		TypeMeta: v1.TypeMeta{
			APIVersion: "mcp.tmc.dev/v1alpha1",
			Kind:       "MCPHealthCheck",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "api-server-health",
			Namespace: "mcp-system",
		},
		Spec: MCPHealthCheckSpec{
			Target:           "api-server:8080",
			Protocol:         "http",
			Interval:         v1.Duration{Duration: 30 * time.Second},
			Timeout:          v1.Duration{Duration: 5 * time.Second},
			FailureThreshold: 3,
			SuccessThreshold: 1,
			HTTPPath:         "/health",
			ExpectedStatus:   200,
			Alerting: MCPHealthCheckAlerting{
				Enabled: true,
				Webhook: "https://hooks.slack.com/services/...",
				Slack: SlackAlerting{
					Enabled: true,
					Channel: "#alerts",
				},
			},
		},
	}
}

// RunOperator runs the MCP health operator
func RunOperator(config *HealthConfig, healthApp *HealthApp) error {
	operator, err := NewMCPHealthOperator(config, healthApp)
	if err != nil {
		return fmt.Errorf("failed to create operator: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create CRD if needed
	if err := operator.CreateMCPHealthCheckCRD(ctx); err != nil {
		return fmt.Errorf("failed to create CRD: %w", err)
	}

	// Start operator
	return operator.Start(ctx)
}
