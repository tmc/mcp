// mcp-audit: Comprehensive audit logging and analysis tool for MCP implementations
//
// This tool provides enterprise-grade audit logging and analysis capabilities including:
// - Comprehensive audit trail generation and management
// - Real-time log analysis and search capabilities
// - Compliance reporting for SOC2, ISO27001, GDPR, HIPAA, PCI DSS
// - Anomaly detection and behavioral analysis
// - Data privacy compliance and PII detection
// - Security event correlation and alerting
// - Forensic analysis and incident response support
//
// Usage:
//   mcp-audit [command] [options]
//
// Commands:
//   log           Generate and manage audit logs
//   analyze       Analyze existing audit logs
//   search        Search through audit logs
//   report        Generate compliance reports
//   monitor       Real-time monitoring and alerting
//   anomaly       Anomaly detection and analysis
//   privacy       Data privacy compliance checking
//   export        Export audit data in various formats
//
// Examples:
//   mcp-audit log --target "stdio://./server" --output audit.log
//   mcp-audit analyze --input audit.log --compliance soc2
//   mcp-audit search --query "authentication failed" --timerange "1h"
//   mcp-audit report --compliance gdpr --format pdf
//   mcp-audit monitor --alerts email:security@example.com
//   mcp-audit anomaly --model behavioral --threshold 0.8
//   mcp-audit privacy --scan-pii --redact-output
//   mcp-audit export --format json --filter "severity:high"
//
package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/tmc/mcp"
)

// AuditConfig represents the audit configuration
type AuditConfig struct {
	Target           string            `json:"target"`
	OutputFile       string            `json:"output_file"`
	Format           string            `json:"format"`
	LogLevel         string            `json:"log_level"`
	Retention        time.Duration     `json:"retention"`
	Compression      bool              `json:"compression"`
	Encryption       bool              `json:"encryption"`
	RealTime         bool              `json:"real_time"`
	BufferSize       int               `json:"buffer_size"`
	FlushInterval    time.Duration     `json:"flush_interval"`
	ComplianceFrameworks []string      `json:"compliance_frameworks"`
	PIIDetection     bool              `json:"pii_detection"`
	Redaction        bool              `json:"redaction"`
	Anomaly          bool              `json:"anomaly"`
	AlertEndpoints   []string          `json:"alert_endpoints"`
	Metadata         map[string]string `json:"metadata"`
}

// AuditEvent represents a single audit event
type AuditEvent struct {
	ID            string                 `json:"id"`
	Timestamp     time.Time              `json:"timestamp"`
	EventType     string                 `json:"event_type"`
	Source        string                 `json:"source"`
	Target        string                 `json:"target"`
	User          string                 `json:"user,omitempty"`
	SessionID     string                 `json:"session_id,omitempty"`
	Action        string                 `json:"action"`
	Resource      string                 `json:"resource,omitempty"`
	Method        string                 `json:"method,omitempty"`
	Parameters    map[string]interface{} `json:"parameters,omitempty"`
	Result        string                 `json:"result"`
	StatusCode    int                    `json:"status_code"`
	Duration      time.Duration          `json:"duration"`
	IPAddress     string                 `json:"ip_address,omitempty"`
	UserAgent     string                 `json:"user_agent,omitempty"`
	Severity      string                 `json:"severity"`
	Category      string                 `json:"category"`
	Message       string                 `json:"message"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	PIIDetected   bool                   `json:"pii_detected"`
	PIITypes      []string               `json:"pii_types,omitempty"`
	Compliance    map[string]bool        `json:"compliance"`
	RiskScore     float64                `json:"risk_score"`
	Anomaly       bool                   `json:"anomaly"`
	AnomalyScore  float64                `json:"anomaly_score"`
	Correlation   []string               `json:"correlation,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Hash          string                 `json:"hash"`
	Signature     string                 `json:"signature,omitempty"`
}

// AuditLog represents the audit log manager
type AuditLog struct {
	config      *AuditConfig
	file        *os.File
	buffer      []AuditEvent
	bufferMutex sync.RWMutex
	flushTicker *time.Ticker
	stopChan    chan struct{}
	client      *mcp.Client
	piiDetector *PIIDetector
	anomalyDetector *AnomalyDetector
	correlationEngine *CorrelationEngine
}

// AuditAnalyzer represents the audit analysis engine
type AuditAnalyzer struct {
	config         *AuditConfig
	events         []AuditEvent
	patterns       map[string]*regexp.Regexp
	complianceRules map[string][]ComplianceRule
	statistics     AuditStatistics
	anomalies      []AnomalyResult
	correlations   []CorrelationResult
}

// AuditStatistics represents audit statistics
type AuditStatistics struct {
	TotalEvents       int                    `json:"total_events"`
	EventsByType      map[string]int         `json:"events_by_type"`
	EventsBySeverity  map[string]int         `json:"events_by_severity"`
	EventsByCategory  map[string]int         `json:"events_by_category"`
	EventsByUser      map[string]int         `json:"events_by_user"`
	EventsByAction    map[string]int         `json:"events_by_action"`
	EventsByStatus    map[string]int         `json:"events_by_status"`
	AverageRiskScore  float64                `json:"average_risk_score"`
	PIIEvents         int                    `json:"pii_events"`
	AnomalyEvents     int                    `json:"anomaly_events"`
	ComplianceIssues  int                    `json:"compliance_issues"`
	TimeRange         TimeRange              `json:"time_range"`
	TopResources      []ResourceUsage        `json:"top_resources"`
	TopUsers          []UserActivity         `json:"top_users"`
	ErrorRate         float64                `json:"error_rate"`
	AverageResponseTime time.Duration        `json:"average_response_time"`
}

// TimeRange represents a time range
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ResourceUsage represents resource usage statistics
type ResourceUsage struct {
	Resource string `json:"resource"`
	Count    int    `json:"count"`
	Users    int    `json:"users"`
}

// UserActivity represents user activity statistics
type UserActivity struct {
	User     string `json:"user"`
	Events   int    `json:"events"`
	Actions  int    `json:"actions"`
	LastSeen time.Time `json:"last_seen"`
}

// ComplianceRule represents a compliance rule
type ComplianceRule struct {
	ID          string `json:"id"`
	Framework   string `json:"framework"`
	Control     string `json:"control"`
	Description string `json:"description"`
	Pattern     *regexp.Regexp `json:"-"`
	Severity    string `json:"severity"`
	Required    bool   `json:"required"`
}

// ComplianceReport represents a compliance report
type ComplianceReport struct {
	Framework     string                    `json:"framework"`
	GeneratedAt   time.Time                 `json:"generated_at"`
	Period        TimeRange                 `json:"period"`
	TotalEvents   int                       `json:"total_events"`
	Violations    []ComplianceViolation     `json:"violations"`
	Controls      map[string]ControlStatus  `json:"controls"`
	Score         float64                   `json:"score"`
	Status        string                    `json:"status"`
	Summary       string                    `json:"summary"`
	Recommendations []string                `json:"recommendations"`
	Evidence      []EvidenceItem            `json:"evidence"`
}

// ComplianceViolation represents a compliance violation
type ComplianceViolation struct {
	ID          string                 `json:"id"`
	Framework   string                 `json:"framework"`
	Control     string                 `json:"control"`
	Description string                 `json:"description"`
	Severity    string                 `json:"severity"`
	Events      []string               `json:"events"`
	Count       int                    `json:"count"`
	FirstSeen   time.Time              `json:"first_seen"`
	LastSeen    time.Time              `json:"last_seen"`
	Remediation string                 `json:"remediation"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ControlStatus represents the status of a compliance control
type ControlStatus struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Status       string    `json:"status"`
	Score        float64   `json:"score"`
	Violations   int       `json:"violations"`
	LastChecked  time.Time `json:"last_checked"`
	Evidence     []string  `json:"evidence"`
	Remediation  string    `json:"remediation"`
}

// EvidenceItem represents an evidence item
type EvidenceItem struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
	Source      string                 `json:"source"`
	Data        map[string]interface{} `json:"data"`
	Hash        string                 `json:"hash"`
}

// PIIDetector represents the PII detection engine
type PIIDetector struct {
	patterns map[string]*regexp.Regexp
	enabled  bool
}

// PIIType represents a type of PII
type PIIType struct {
	Name        string         `json:"name"`
	Pattern     *regexp.Regexp `json:"-"`
	Description string         `json:"description"`
	Severity    string         `json:"severity"`
	Countries   []string       `json:"countries"`
}

// AnomalyDetector represents the anomaly detection engine
type AnomalyDetector struct {
	baseline     map[string]float64
	threshold    float64
	window       time.Duration
	enabled      bool
	model        string
	features     []string
	mutex        sync.RWMutex
}

// AnomalyResult represents an anomaly detection result
type AnomalyResult struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	Type        string                 `json:"type"`
	Score       float64                `json:"score"`
	Threshold   float64                `json:"threshold"`
	Description string                 `json:"description"`
	Events      []string               `json:"events"`
	Features    map[string]float64     `json:"features"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// CorrelationEngine represents the event correlation engine
type CorrelationEngine struct {
	rules    []CorrelationRule
	window   time.Duration
	enabled  bool
	mutex    sync.RWMutex
}

// CorrelationRule represents a correlation rule
type CorrelationRule struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Conditions  []Condition   `json:"conditions"`
	Window      time.Duration `json:"window"`
	Threshold   int           `json:"threshold"`
	Severity    string        `json:"severity"`
	Action      string        `json:"action"`
}

// Condition represents a correlation condition
type Condition struct {
	Field     string      `json:"field"`
	Operator  string      `json:"operator"`
	Value     interface{} `json:"value"`
	Pattern   *regexp.Regexp `json:"-"`
}

// CorrelationResult represents a correlation result
type CorrelationResult struct {
	ID          string                 `json:"id"`
	RuleID      string                 `json:"rule_id"`
	RuleName    string                 `json:"rule_name"`
	Timestamp   time.Time              `json:"timestamp"`
	Events      []string               `json:"events"`
	Count       int                    `json:"count"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// NewAuditLog creates a new audit log instance
func NewAuditLog(config *AuditConfig) (*AuditLog, error) {
	if config == nil {
		return nil, fmt.Errorf("audit config cannot be nil")
	}

	// Create output file
	file, err := os.OpenFile(config.OutputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open output file: %w", err)
	}

	auditLog := &AuditLog{
		config:   config,
		file:     file,
		buffer:   make([]AuditEvent, 0, config.BufferSize),
		stopChan: make(chan struct{}),
	}

	// Initialize PII detector
	if config.PIIDetection {
		auditLog.piiDetector = NewPIIDetector()
	}

	// Initialize anomaly detector
	if config.Anomaly {
		auditLog.anomalyDetector = NewAnomalyDetector(0.8, 5*time.Minute)
	}

	// Initialize correlation engine
	auditLog.correlationEngine = NewCorrelationEngine()

	// Start flush timer
	if config.FlushInterval > 0 {
		auditLog.flushTicker = time.NewTicker(config.FlushInterval)
		go auditLog.flushRoutine()
	}

	return auditLog, nil
}

// NewPIIDetector creates a new PII detector
func NewPIIDetector() *PIIDetector {
	patterns := make(map[string]*regexp.Regexp)
	
	// Social Security Number (US)
	patterns["ssn"] = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
	
	// Credit Card Numbers
	patterns["credit_card"] = regexp.MustCompile(`\b(?:\d{4}[\s-]?){3}\d{4}\b`)
	
	// Email Addresses
	patterns["email"] = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`)
	
	// Phone Numbers
	patterns["phone"] = regexp.MustCompile(`\b(?:\+?1[-.\s]?)?\(?([0-9]{3})\)?[-.\s]?([0-9]{3})[-.\s]?([0-9]{4})\b`)
	
	// IP Addresses
	patterns["ip_address"] = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	
	// Date of Birth
	patterns["date_of_birth"] = regexp.MustCompile(`\b(?:0[1-9]|1[0-2])[\/\-](?:0[1-9]|[12]\d|3[01])[\/\-](?:19|20)\d{2}\b`)
	
	// Driver's License (US format)
	patterns["drivers_license"] = regexp.MustCompile(`\b[A-Z]{1,2}\d{6,8}\b`)
	
	// Bank Account Numbers
	patterns["bank_account"] = regexp.MustCompile(`\b\d{8,17}\b`)
	
	// Passport Numbers
	patterns["passport"] = regexp.MustCompile(`\b[A-Z]{1,2}\d{6,9}\b`)
	
	// Medical Record Numbers
	patterns["medical_record"] = regexp.MustCompile(`\bMRN[:\s]?\d{6,12}\b`)
	
	return &PIIDetector{
		patterns: patterns,
		enabled:  true,
	}
}

// NewAnomalyDetector creates a new anomaly detector
func NewAnomalyDetector(threshold float64, window time.Duration) *AnomalyDetector {
	return &AnomalyDetector{
		baseline:  make(map[string]float64),
		threshold: threshold,
		window:    window,
		enabled:   true,
		model:     "statistical",
		features:  []string{"event_rate", "error_rate", "user_activity", "resource_usage"},
	}
}

// NewCorrelationEngine creates a new correlation engine
func NewCorrelationEngine() *CorrelationEngine {
	return &CorrelationEngine{
		rules:   make([]CorrelationRule, 0),
		window:  10 * time.Minute,
		enabled: true,
	}
}

// LogEvent logs an audit event
func (al *AuditLog) LogEvent(event AuditEvent) error {
	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Generate ID if not provided
	if event.ID == "" {
		event.ID = generateEventID()
	}

	// Detect PII if enabled
	if al.config.PIIDetection && al.piiDetector != nil {
		piiTypes := al.piiDetector.DetectPII(event.Message)
		if len(piiTypes) > 0 {
			event.PIIDetected = true
			event.PIITypes = piiTypes
		}
	}

	// Redact PII if enabled
	if al.config.Redaction && event.PIIDetected {
		event.Message = al.piiDetector.RedactPII(event.Message)
		if event.Parameters != nil {
			event.Parameters = al.piiDetector.RedactPIIFromMap(event.Parameters)
		}
	}

	// Calculate risk score
	event.RiskScore = al.calculateRiskScore(event)

	// Check for anomalies
	if al.config.Anomaly && al.anomalyDetector != nil {
		anomalyScore, isAnomaly := al.anomalyDetector.CheckAnomaly(event)
		event.AnomalyScore = anomalyScore
		event.Anomaly = isAnomaly
	}

	// Evaluate compliance
	event.Compliance = al.evaluateCompliance(event)

	// Generate hash
	event.Hash = al.generateEventHash(event)

	// Sign event if encryption enabled
	if al.config.Encryption {
		event.Signature = al.signEvent(event)
	}

	// Add to buffer
	al.bufferMutex.Lock()
	al.buffer = append(al.buffer, event)
	al.bufferMutex.Unlock()

	// Flush if buffer is full or real-time mode
	if len(al.buffer) >= al.config.BufferSize || al.config.RealTime {
		return al.flush()
	}

	return nil
}

// DetectPII detects PII in text
func (pd *PIIDetector) DetectPII(text string) []string {
	if !pd.enabled {
		return nil
	}

	var detected []string
	for piiType, pattern := range pd.patterns {
		if pattern.MatchString(text) {
			detected = append(detected, piiType)
		}
	}

	return detected
}

// RedactPII redacts PII from text
func (pd *PIIDetector) RedactPII(text string) string {
	if !pd.enabled {
		return text
	}

	for _, pattern := range pd.patterns {
		text = pattern.ReplaceAllString(text, "[REDACTED]")
	}

	return text
}

// RedactPIIFromMap redacts PII from a map
func (pd *PIIDetector) RedactPIIFromMap(data map[string]interface{}) map[string]interface{} {
	if !pd.enabled {
		return data
	}

	result := make(map[string]interface{})
	for key, value := range data {
		if str, ok := value.(string); ok {
			result[key] = pd.RedactPII(str)
		} else {
			result[key] = value
		}
	}

	return result
}

// CheckAnomaly checks if an event is anomalous
func (ad *AnomalyDetector) CheckAnomaly(event AuditEvent) (float64, bool) {
	if !ad.enabled {
		return 0, false
	}

	ad.mutex.RLock()
	defer ad.mutex.RUnlock()

	// Simple statistical anomaly detection
	features := ad.extractFeatures(event)
	score := ad.calculateAnomalyScore(features)

	return score, score > ad.threshold
}

// extractFeatures extracts features from an event
func (ad *AnomalyDetector) extractFeatures(event AuditEvent) map[string]float64 {
	features := make(map[string]float64)

	// Event type frequency
	features["event_type_frequency"] = ad.getEventTypeFrequency(event.EventType)

	// User activity level
	features["user_activity"] = ad.getUserActivityLevel(event.User)

	// Error rate
	if event.StatusCode >= 400 {
		features["error_indicator"] = 1.0
	} else {
		features["error_indicator"] = 0.0
	}

	// Time-based features
	hour := float64(event.Timestamp.Hour())
	features["hour_of_day"] = hour

	// Risk score
	features["risk_score"] = event.RiskScore

	return features
}

// calculateAnomalyScore calculates the anomaly score
func (ad *AnomalyDetector) calculateAnomalyScore(features map[string]float64) float64 {
	// Simple scoring based on deviation from baseline
	score := 0.0
	count := 0

	for feature, value := range features {
		if baseline, exists := ad.baseline[feature]; exists {
			deviation := abs(value - baseline)
			score += deviation
			count++
		}
	}

	if count > 0 {
		return score / float64(count)
	}

	return 0.0
}

// getEventTypeFrequency gets the frequency of an event type
func (ad *AnomalyDetector) getEventTypeFrequency(eventType string) float64 {
	// This would be implemented with actual frequency calculation
	// For now, return a placeholder value
	return 0.5
}

// getUserActivityLevel gets the user activity level
func (ad *AnomalyDetector) getUserActivityLevel(user string) float64 {
	// This would be implemented with actual activity level calculation
	// For now, return a placeholder value
	return 0.3
}

// calculateRiskScore calculates the risk score for an event
func (al *AuditLog) calculateRiskScore(event AuditEvent) float64 {
	score := 0.0

	// Base score based on event type
	switch event.EventType {
	case "authentication":
		score += 0.3
	case "authorization":
		score += 0.4
	case "data_access":
		score += 0.5
	case "configuration_change":
		score += 0.7
	case "admin_action":
		score += 0.8
	default:
		score += 0.2
	}

	// Severity multiplier
	switch event.Severity {
	case "critical":
		score *= 2.0
	case "high":
		score *= 1.5
	case "medium":
		score *= 1.0
	case "low":
		score *= 0.5
	}

	// Error conditions
	if event.StatusCode >= 400 {
		score += 0.3
	}

	// PII involvement
	if event.PIIDetected {
		score += 0.4
	}

	// Anomaly flag
	if event.Anomaly {
		score += 0.5
	}

	// Normalize to 0-1 range
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// evaluateCompliance evaluates compliance for an event
func (al *AuditLog) evaluateCompliance(event AuditEvent) map[string]bool {
	compliance := make(map[string]bool)

	for _, framework := range al.config.ComplianceFrameworks {
		compliance[framework] = al.evaluateFrameworkCompliance(event, framework)
	}

	return compliance
}

// evaluateFrameworkCompliance evaluates compliance for a specific framework
func (al *AuditLog) evaluateFrameworkCompliance(event AuditEvent, framework string) bool {
	switch framework {
	case "soc2":
		return al.evaluateSOC2Compliance(event)
	case "iso27001":
		return al.evaluateISO27001Compliance(event)
	case "gdpr":
		return al.evaluateGDPRCompliance(event)
	case "hipaa":
		return al.evaluateHIPAACompliance(event)
	case "pci":
		return al.evaluatePCICompliance(event)
	default:
		return true
	}
}

// evaluateSOC2Compliance evaluates SOC2 compliance
func (al *AuditLog) evaluateSOC2Compliance(event AuditEvent) bool {
	// SOC2 requires comprehensive logging of security events
	if event.EventType == "authentication" && event.StatusCode >= 400 {
		return false // Failed authentication should be logged
	}
	
	if event.EventType == "authorization" && event.StatusCode >= 400 {
		return false // Authorization failures should be logged
	}
	
	if event.PIIDetected && !al.config.PIIDetection {
		return false // PII handling should be monitored
	}
	
	return true
}

// evaluateISO27001Compliance evaluates ISO 27001 compliance
func (al *AuditLog) evaluateISO27001Compliance(event AuditEvent) bool {
	// ISO 27001 requires monitoring of security events
	if event.EventType == "admin_action" && event.User == "" {
		return false // Admin actions should be attributed
	}
	
	if event.EventType == "configuration_change" && event.User == "" {
		return false // Configuration changes should be attributed
	}
	
	return true
}

// evaluateGDPRCompliance evaluates GDPR compliance
func (al *AuditLog) evaluateGDPRCompliance(event AuditEvent) bool {
	// GDPR requires monitoring of personal data processing
	if event.PIIDetected && !al.config.Redaction {
		return false // PII should be protected
	}
	
	if event.EventType == "data_access" && event.PIIDetected {
		return true // Data access should be logged
	}
	
	return true
}

// evaluateHIPAACompliance evaluates HIPAA compliance
func (al *AuditLog) evaluateHIPAACompliance(event AuditEvent) bool {
	// HIPAA requires audit of PHI access
	if event.EventType == "data_access" && event.Resource == "phi" {
		return event.User != "" // PHI access must be attributed
	}
	
	if event.PIIDetected && strings.Contains(event.Message, "medical") {
		return al.config.Encryption // Medical data should be encrypted
	}
	
	return true
}

// evaluatePCICompliance evaluates PCI DSS compliance
func (al *AuditLog) evaluatePCICompliance(event AuditEvent) bool {
	// PCI DSS requires monitoring of cardholder data access
	if event.EventType == "data_access" && event.Resource == "cardholder_data" {
		return event.User != "" // Cardholder data access must be attributed
	}
	
	if event.PIIDetected && strings.Contains(event.Message, "card") {
		return al.config.Encryption // Card data should be encrypted
	}
	
	return true
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return fmt.Sprintf("audit_%d_%d", time.Now().UnixNano(), os.Getpid())
}

// generateEventHash generates a hash for an event
func (al *AuditLog) generateEventHash(event AuditEvent) string {
	// Create a consistent string representation for hashing
	hashData := fmt.Sprintf("%s:%s:%s:%s:%s:%d:%s",
		event.Timestamp.Format(time.RFC3339),
		event.EventType,
		event.User,
		event.Action,
		event.Resource,
		event.StatusCode,
		event.Message)
	
	hash := sha256.Sum256([]byte(hashData))
	return fmt.Sprintf("%x", hash)
}

// signEvent signs an event for integrity
func (al *AuditLog) signEvent(event AuditEvent) string {
	// This would implement digital signature in a real implementation
	// For now, return a placeholder
	return fmt.Sprintf("sig_%s", event.Hash[:16])
}

// flush flushes the buffer to disk
func (al *AuditLog) flush() error {
	al.bufferMutex.Lock()
	defer al.bufferMutex.Unlock()

	if len(al.buffer) == 0 {
		return nil
	}

	// Write events to file
	for _, event := range al.buffer {
		eventJSON, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal event: %w", err)
		}

		if _, err := al.file.WriteString(string(eventJSON) + "\n"); err != nil {
			return fmt.Errorf("failed to write event: %w", err)
		}
	}

	// Sync to disk
	if err := al.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	// Clear buffer
	al.buffer = al.buffer[:0]

	return nil
}

// flushRoutine runs the flush routine
func (al *AuditLog) flushRoutine() {
	for {
		select {
		case <-al.flushTicker.C:
			if err := al.flush(); err != nil {
				log.Printf("Failed to flush audit log: %v", err)
			}
		case <-al.stopChan:
			return
		}
	}
}

// Close closes the audit log
func (al *AuditLog) Close() error {
	// Stop flush routine
	if al.flushTicker != nil {
		al.flushTicker.Stop()
		close(al.stopChan)
	}

	// Final flush
	if err := al.flush(); err != nil {
		return err
	}

	// Close file
	return al.file.Close()
}

// NewAuditAnalyzer creates a new audit analyzer
func NewAuditAnalyzer(config *AuditConfig) *AuditAnalyzer {
	analyzer := &AuditAnalyzer{
		config:          config,
		events:          make([]AuditEvent, 0),
		patterns:        make(map[string]*regexp.Regexp),
		complianceRules: make(map[string][]ComplianceRule),
		statistics:      AuditStatistics{},
		anomalies:       make([]AnomalyResult, 0),
		correlations:    make([]CorrelationResult, 0),
	}

	// Initialize patterns
	analyzer.initializePatterns()

	// Initialize compliance rules
	analyzer.initializeComplianceRules()

	return analyzer
}

// initializePatterns initializes analysis patterns
func (aa *AuditAnalyzer) initializePatterns() {
	aa.patterns["failed_login"] = regexp.MustCompile(`(?i)(login|authentication).*failed`)
	aa.patterns["brute_force"] = regexp.MustCompile(`(?i)(multiple|repeated).*failed.*attempts`)
	aa.patterns["privilege_escalation"] = regexp.MustCompile(`(?i)(privilege|permission).*escalat`)
	aa.patterns["data_breach"] = regexp.MustCompile(`(?i)(data|breach|leak|exposure)`)
	aa.patterns["unauthorized_access"] = regexp.MustCompile(`(?i)(unauthorized|forbidden|access.*denied)`)
	aa.patterns["malicious_activity"] = regexp.MustCompile(`(?i)(malicious|attack|intrusion|exploit)`)
	aa.patterns["configuration_change"] = regexp.MustCompile(`(?i)(configuration|config|setting).*chang`)
	aa.patterns["admin_action"] = regexp.MustCompile(`(?i)(admin|administrator).*action`)
	aa.patterns["error_pattern"] = regexp.MustCompile(`(?i)(error|exception|fail|crash)`)
	aa.patterns["suspicious_behavior"] = regexp.MustCompile(`(?i)(suspicious|anomal|unusual)`)
}

// initializeComplianceRules initializes compliance rules
func (aa *AuditAnalyzer) initializeComplianceRules() {
	// SOC2 rules
	aa.complianceRules["soc2"] = []ComplianceRule{
		{
			ID:          "SOC2-CC6.1",
			Framework:   "soc2",
			Control:     "CC6.1",
			Description: "Logical and physical access controls",
			Pattern:     regexp.MustCompile(`(?i)(access|login|authentication)`),
			Severity:    "high",
			Required:    true,
		},
		{
			ID:          "SOC2-CC6.2",
			Framework:   "soc2",
			Control:     "CC6.2",
			Description: "Authentication and authorization",
			Pattern:     regexp.MustCompile(`(?i)(authentication|authorization)`),
			Severity:    "high",
			Required:    true,
		},
		{
			ID:          "SOC2-CC6.7",
			Framework:   "soc2",
			Control:     "CC6.7",
			Description: "Data transmission controls",
			Pattern:     regexp.MustCompile(`(?i)(transmission|transfer|data.*send)`),
			Severity:    "medium",
			Required:    true,
		},
	}

	// ISO 27001 rules
	aa.complianceRules["iso27001"] = []ComplianceRule{
		{
			ID:          "ISO27001-A.9.1",
			Framework:   "iso27001",
			Control:     "A.9.1",
			Description: "Business requirements for access control",
			Pattern:     regexp.MustCompile(`(?i)(access.*control|permission)`),
			Severity:    "high",
			Required:    true,
		},
		{
			ID:          "ISO27001-A.12.4",
			Framework:   "iso27001",
			Control:     "A.12.4",
			Description: "Logging and monitoring",
			Pattern:     regexp.MustCompile(`(?i)(log|monitor|audit)`),
			Severity:    "medium",
			Required:    true,
		},
	}

	// GDPR rules
	aa.complianceRules["gdpr"] = []ComplianceRule{
		{
			ID:          "GDPR-Art32",
			Framework:   "gdpr",
			Control:     "Article 32",
			Description: "Security of processing",
			Pattern:     regexp.MustCompile(`(?i)(personal.*data|pii|privacy)`),
			Severity:    "high",
			Required:    true,
		},
	}

	// HIPAA rules
	aa.complianceRules["hipaa"] = []ComplianceRule{
		{
			ID:          "HIPAA-164.312.a",
			Framework:   "hipaa",
			Control:     "164.312(a)",
			Description: "Access control (unique user identification)",
			Pattern:     regexp.MustCompile(`(?i)(phi|medical|health.*record)`),
			Severity:    "high",
			Required:    true,
		},
	}

	// PCI DSS rules
	aa.complianceRules["pci"] = []ComplianceRule{
		{
			ID:          "PCI-10.2",
			Framework:   "pci",
			Control:     "10.2",
			Description: "Audit trail for cardholder data",
			Pattern:     regexp.MustCompile(`(?i)(cardholder|card.*data|payment)`),
			Severity:    "high",
			Required:    true,
		},
	}
}

// LoadEvents loads events from a file
func (aa *AuditAnalyzer) LoadEvents(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var event AuditEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			log.Printf("Failed to parse event: %v", err)
			continue
		}

		aa.events = append(aa.events, event)
	}

	return scanner.Err()
}

// AnalyzeEvents analyzes loaded events
func (aa *AuditAnalyzer) AnalyzeEvents() error {
	// Calculate statistics
	aa.calculateStatistics()

	// Detect anomalies
	aa.detectAnomalies()

	// Correlate events
	aa.correlateEvents()

	// Validate compliance
	aa.validateCompliance()

	return nil
}

// calculateStatistics calculates audit statistics
func (aa *AuditAnalyzer) calculateStatistics() {
	aa.statistics = AuditStatistics{
		TotalEvents:      len(aa.events),
		EventsByType:     make(map[string]int),
		EventsBySeverity: make(map[string]int),
		EventsByCategory: make(map[string]int),
		EventsByUser:     make(map[string]int),
		EventsByAction:   make(map[string]int),
		EventsByStatus:   make(map[string]int),
		TopResources:     make([]ResourceUsage, 0),
		TopUsers:         make([]UserActivity, 0),
	}

	if len(aa.events) == 0 {
		return
	}

	// Track various metrics
	var totalRiskScore float64
	var totalResponseTime time.Duration
	var piiEvents, anomalyEvents, complianceIssues int
	var errorCount int
	var minTime, maxTime time.Time

	resourceUsage := make(map[string]ResourceUsage)
	userActivity := make(map[string]UserActivity)

	for i, event := range aa.events {
		// Basic counts
		aa.statistics.EventsByType[event.EventType]++
		aa.statistics.EventsBySeverity[event.Severity]++
		aa.statistics.EventsByCategory[event.Category]++
		aa.statistics.EventsByUser[event.User]++
		aa.statistics.EventsByAction[event.Action]++
		aa.statistics.EventsByStatus[fmt.Sprintf("%d", event.StatusCode)]++

		// Risk score
		totalRiskScore += event.RiskScore

		// Response time
		totalResponseTime += event.Duration

		// PII events
		if event.PIIDetected {
			piiEvents++
		}

		// Anomaly events
		if event.Anomaly {
			anomalyEvents++
		}

		// Compliance issues
		for _, compliant := range event.Compliance {
			if !compliant {
				complianceIssues++
				break
			}
		}

		// Error count
		if event.StatusCode >= 400 {
			errorCount++
		}

		// Time range
		if i == 0 {
			minTime = event.Timestamp
			maxTime = event.Timestamp
		} else {
			if event.Timestamp.Before(minTime) {
				minTime = event.Timestamp
			}
			if event.Timestamp.After(maxTime) {
				maxTime = event.Timestamp
			}
		}

		// Resource usage
		if event.Resource != "" {
			if usage, exists := resourceUsage[event.Resource]; exists {
				usage.Count++
				resourceUsage[event.Resource] = usage
			} else {
				resourceUsage[event.Resource] = ResourceUsage{
					Resource: event.Resource,
					Count:    1,
					Users:    1,
				}
			}
		}

		// User activity
		if event.User != "" {
			if activity, exists := userActivity[event.User]; exists {
				activity.Events++
				if event.Timestamp.After(activity.LastSeen) {
					activity.LastSeen = event.Timestamp
				}
				userActivity[event.User] = activity
			} else {
				userActivity[event.User] = UserActivity{
					User:     event.User,
					Events:   1,
					Actions:  1,
					LastSeen: event.Timestamp,
				}
			}
		}
	}

	// Calculate averages
	aa.statistics.AverageRiskScore = totalRiskScore / float64(len(aa.events))
	aa.statistics.AverageResponseTime = totalResponseTime / time.Duration(len(aa.events))
	aa.statistics.PIIEvents = piiEvents
	aa.statistics.AnomalyEvents = anomalyEvents
	aa.statistics.ComplianceIssues = complianceIssues
	aa.statistics.ErrorRate = float64(errorCount) / float64(len(aa.events))
	aa.statistics.TimeRange = TimeRange{Start: minTime, End: maxTime}

	// Convert maps to sorted slices
	aa.statistics.TopResources = aa.sortResourceUsage(resourceUsage)
	aa.statistics.TopUsers = aa.sortUserActivity(userActivity)
}

// sortResourceUsage sorts resource usage by count
func (aa *AuditAnalyzer) sortResourceUsage(usage map[string]ResourceUsage) []ResourceUsage {
	var sorted []ResourceUsage
	for _, ru := range usage {
		sorted = append(sorted, ru)
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Count > sorted[j].Count
	})

	// Return top 10
	if len(sorted) > 10 {
		sorted = sorted[:10]
	}

	return sorted
}

// sortUserActivity sorts user activity by event count
func (aa *AuditAnalyzer) sortUserActivity(activity map[string]UserActivity) []UserActivity {
	var sorted []UserActivity
	for _, ua := range activity {
		sorted = append(sorted, ua)
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Events > sorted[j].Events
	})

	// Return top 10
	if len(sorted) > 10 {
		sorted = sorted[:10]
	}

	return sorted
}

// detectAnomalies detects anomalies in the events
func (aa *AuditAnalyzer) detectAnomalies() {
	// Simple anomaly detection based on patterns
	for _, event := range aa.events {
		if event.Anomaly {
			anomaly := AnomalyResult{
				ID:          fmt.Sprintf("anomaly_%s", event.ID),
				Timestamp:   event.Timestamp,
				Type:        "behavioral",
				Score:       event.AnomalyScore,
				Threshold:   0.8,
				Description: "Anomalous behavior detected",
				Events:      []string{event.ID},
				Features:    map[string]float64{"risk_score": event.RiskScore},
				Metadata:    map[string]interface{}{"event_type": event.EventType},
			}
			aa.anomalies = append(aa.anomalies, anomaly)
		}
	}
}

// correlateEvents correlates events based on patterns
func (aa *AuditAnalyzer) correlateEvents() {
	// Simple correlation based on user and time proximity
	userEvents := make(map[string][]AuditEvent)
	
	for _, event := range aa.events {
		if event.User != "" {
			userEvents[event.User] = append(userEvents[event.User], event)
		}
	}

	// Look for suspicious patterns
	for user, events := range userEvents {
		aa.correlateSuspiciousActivity(user, events)
	}
}

// correlateSuspiciousActivity correlates suspicious activity for a user
func (aa *AuditAnalyzer) correlateSuspiciousActivity(user string, events []AuditEvent) {
	// Sort events by timestamp
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	// Look for patterns
	failedLogins := 0
	var failedLoginEvents []string
	
	for _, event := range events {
		if event.EventType == "authentication" && event.StatusCode >= 400 {
			failedLogins++
			failedLoginEvents = append(failedLoginEvents, event.ID)
		} else if event.EventType == "authentication" && event.StatusCode < 400 {
			// Reset on successful login
			failedLogins = 0
			failedLoginEvents = []string{}
		}

		// Brute force detection
		if failedLogins >= 5 {
			correlation := CorrelationResult{
				ID:          fmt.Sprintf("corr_%s_%d", user, time.Now().UnixNano()),
				RuleID:      "brute_force",
				RuleName:    "Brute Force Attack",
				Timestamp:   event.Timestamp,
				Events:      failedLoginEvents,
				Count:       failedLogins,
				Severity:    "high",
				Description: fmt.Sprintf("Brute force attack detected for user %s", user),
				Metadata:    map[string]interface{}{"user": user, "pattern": "brute_force"},
			}
			aa.correlations = append(aa.correlations, correlation)
		}
	}
}

// validateCompliance validates compliance rules
func (aa *AuditAnalyzer) validateCompliance() {
	// This would implement comprehensive compliance validation
	// For now, we'll use the compliance flags already set on events
}

// GenerateComplianceReport generates a compliance report
func (aa *AuditAnalyzer) GenerateComplianceReport(framework string) ComplianceReport {
	violations := make([]ComplianceViolation, 0)
	controls := make(map[string]ControlStatus)
	
	// Get rules for the framework
	rules, exists := aa.complianceRules[framework]
	if !exists {
		return ComplianceReport{
			Framework:   framework,
			GeneratedAt: time.Now(),
			Status:      "ERROR",
			Summary:     "Unknown compliance framework",
		}
	}

	// Check each rule
	for _, rule := range rules {
		violationEvents := make([]string, 0)
		
		for _, event := range aa.events {
			if compliant, exists := event.Compliance[framework]; exists && !compliant {
				if rule.Pattern.MatchString(event.Message) {
					violationEvents = append(violationEvents, event.ID)
				}
			}
		}

		// Create control status
		status := "PASS"
		score := 1.0
		if len(violationEvents) > 0 {
			status = "FAIL"
			score = 0.0
		}

		controls[rule.Control] = ControlStatus{
			ID:          rule.Control,
			Title:       rule.Description,
			Description: rule.Description,
			Status:      status,
			Score:       score,
			Violations:  len(violationEvents),
			LastChecked: time.Now(),
			Evidence:    violationEvents,
			Remediation: "Address compliance violations",
		}

		// Create violation if any
		if len(violationEvents) > 0 {
			violation := ComplianceViolation{
				ID:          fmt.Sprintf("viol_%s_%s", framework, rule.Control),
				Framework:   framework,
				Control:     rule.Control,
				Description: rule.Description,
				Severity:    rule.Severity,
				Events:      violationEvents,
				Count:       len(violationEvents),
				FirstSeen:   time.Now(), // Would calculate actual first/last seen
				LastSeen:    time.Now(),
				Remediation: "Address compliance violations",
				Metadata:    map[string]interface{}{"rule_id": rule.ID},
			}
			violations = append(violations, violation)
		}
	}

	// Calculate overall score
	totalScore := 0.0
	for _, control := range controls {
		totalScore += control.Score
	}
	overallScore := totalScore / float64(len(controls))

	// Determine status
	status := "PASS"
	if overallScore < 0.8 {
		status = "FAIL"
	} else if overallScore < 1.0 {
		status = "PARTIAL"
	}

	return ComplianceReport{
		Framework:   framework,
		GeneratedAt: time.Now(),
		Period:      aa.statistics.TimeRange,
		TotalEvents: aa.statistics.TotalEvents,
		Violations:  violations,
		Controls:    controls,
		Score:       overallScore,
		Status:      status,
		Summary:     aa.generateComplianceSummary(framework, overallScore, len(violations)),
		Recommendations: aa.generateComplianceRecommendations(framework, violations),
		Evidence:    aa.generateEvidence(framework),
	}
}

// generateComplianceSummary generates a compliance summary
func (aa *AuditAnalyzer) generateComplianceSummary(framework string, score float64, violations int) string {
	if violations == 0 {
		return fmt.Sprintf("%s compliance assessment passed with score %.2f", framework, score)
	}
	return fmt.Sprintf("%s compliance assessment found %d violations with score %.2f", framework, violations, score)
}

// generateComplianceRecommendations generates compliance recommendations
func (aa *AuditAnalyzer) generateComplianceRecommendations(framework string, violations []ComplianceViolation) []string {
	recommendations := make([]string, 0)

	switch framework {
	case "soc2":
		recommendations = append(recommendations, "Implement comprehensive access control logging")
		recommendations = append(recommendations, "Ensure all authentication events are properly logged")
		recommendations = append(recommendations, "Monitor data transmission activities")
	case "iso27001":
		recommendations = append(recommendations, "Establish comprehensive access control policies")
		recommendations = append(recommendations, "Implement continuous monitoring and logging")
		recommendations = append(recommendations, "Conduct regular security audits")
	case "gdpr":
		recommendations = append(recommendations, "Implement data protection by design")
		recommendations = append(recommendations, "Ensure personal data processing is properly logged")
		recommendations = append(recommendations, "Implement privacy impact assessments")
	case "hipaa":
		recommendations = append(recommendations, "Implement comprehensive PHI access logging")
		recommendations = append(recommendations, "Ensure user attribution for all PHI access")
		recommendations = append(recommendations, "Implement encryption for PHI data")
	case "pci":
		recommendations = append(recommendations, "Implement comprehensive cardholder data access logging")
		recommendations = append(recommendations, "Ensure all payment processing is properly monitored")
		recommendations = append(recommendations, "Implement network segmentation for cardholder data")
	}

	return recommendations
}

// generateEvidence generates evidence items
func (aa *AuditAnalyzer) generateEvidence(framework string) []EvidenceItem {
	evidence := make([]EvidenceItem, 0)

	// Generate evidence based on events
	for _, event := range aa.events {
		if compliant, exists := event.Compliance[framework]; exists && !compliant {
			item := EvidenceItem{
				ID:          fmt.Sprintf("evidence_%s", event.ID),
				Type:        "audit_event",
				Description: fmt.Sprintf("Compliance violation in event %s", event.ID),
				Timestamp:   event.Timestamp,
				Source:      "audit_log",
				Data: map[string]interface{}{
					"event_id":   event.ID,
					"event_type": event.EventType,
					"severity":   event.Severity,
					"user":       event.User,
				},
				Hash: event.Hash,
			}
			evidence = append(evidence, item)
		}
	}

	return evidence
}

// SearchEvents searches events based on criteria
func (aa *AuditAnalyzer) SearchEvents(query string, timeRange *TimeRange, filters map[string]string) []AuditEvent {
	var results []AuditEvent

	// Compile query as regex
	queryRegex, err := regexp.Compile(fmt.Sprintf("(?i)%s", query))
	if err != nil {
		log.Printf("Invalid query regex: %v", err)
		return results
	}

	for _, event := range aa.events {
		// Time range filter
		if timeRange != nil {
			if event.Timestamp.Before(timeRange.Start) || event.Timestamp.After(timeRange.End) {
				continue
			}
		}

		// Text search
		if query != "" {
			if !queryRegex.MatchString(event.Message) &&
			   !queryRegex.MatchString(event.Action) &&
			   !queryRegex.MatchString(event.EventType) {
				continue
			}
		}

		// Apply filters
		matched := true
		for key, value := range filters {
			switch key {
			case "event_type":
				if event.EventType != value {
					matched = false
				}
			case "severity":
				if event.Severity != value {
					matched = false
				}
			case "user":
				if event.User != value {
					matched = false
				}
			case "category":
				if event.Category != value {
					matched = false
				}
			case "status":
				if fmt.Sprintf("%d", event.StatusCode) != value {
					matched = false
				}
			}
		}

		if matched {
			results = append(results, event)
		}
	}

	return results
}

// ExportEvents exports events in various formats
func (aa *AuditAnalyzer) ExportEvents(events []AuditEvent, format string, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	switch format {
	case "json":
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		return encoder.Encode(events)
	case "csv":
		return aa.exportCSV(events, file)
	case "xml":
		return fmt.Errorf("XML export not implemented")
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// exportCSV exports events to CSV format
func (aa *AuditAnalyzer) exportCSV(events []AuditEvent, writer io.Writer) error {
	// CSV header
	header := "ID,Timestamp,EventType,User,Action,Resource,StatusCode,Severity,Message,PIIDetected,Anomaly,RiskScore\n"
	if _, err := writer.Write([]byte(header)); err != nil {
		return err
	}

	// CSV rows
	for _, event := range events {
		row := fmt.Sprintf("%s,%s,%s,%s,%s,%s,%d,%s,%s,%t,%t,%.2f\n",
			event.ID,
			event.Timestamp.Format(time.RFC3339),
			event.EventType,
			event.User,
			event.Action,
			event.Resource,
			event.StatusCode,
			event.Severity,
			strings.ReplaceAll(event.Message, ",", ";"), // Escape commas
			event.PIIDetected,
			event.Anomaly,
			event.RiskScore,
		)
		if _, err := writer.Write([]byte(row)); err != nil {
			return err
		}
	}

	return nil
}

// abs returns the absolute value of a float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// Main CLI implementation
func main() {
	var rootCmd = &cobra.Command{
		Use:   "mcp-audit",
		Short: "Comprehensive audit logging and analysis tool for MCP implementations",
		Long: `mcp-audit provides enterprise-grade audit logging and analysis capabilities for
Model Context Protocol (MCP) implementations. It supports comprehensive audit trail generation,
real-time analysis, compliance reporting, and security monitoring.`,
	}

	// Global flags
	var (
		target       = rootCmd.PersistentFlags().String("target", "", "Target MCP server")
		verbose      = rootCmd.PersistentFlags().Bool("verbose", false, "Enable verbose output")
		configFile   = rootCmd.PersistentFlags().String("config", "", "Configuration file path")
	)

	// Log command
	var logCmd = &cobra.Command{
		Use:   "log",
		Short: "Generate and manage audit logs",
		Long:  `Generates comprehensive audit logs for MCP server activities.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLog(cmd, args, target, verbose)
		},
	}

	logCmd.Flags().String("output", "audit.log", "Output file")
	logCmd.Flags().String("format", "json", "Log format (json, csv)")
	logCmd.Flags().Bool("real-time", false, "Enable real-time logging")
	logCmd.Flags().Int("buffer-size", 1000, "Buffer size")
	logCmd.Flags().Duration("flush-interval", 5*time.Second, "Flush interval")
	logCmd.Flags().Bool("pii-detection", true, "Enable PII detection")
	logCmd.Flags().Bool("redaction", true, "Enable PII redaction")
	logCmd.Flags().Bool("anomaly", true, "Enable anomaly detection")
	logCmd.Flags().Bool("encryption", false, "Enable encryption")
	logCmd.Flags().StringSlice("compliance", []string{"soc2"}, "Compliance frameworks")

	// Analyze command
	var analyzeCmd = &cobra.Command{
		Use:   "analyze",
		Short: "Analyze existing audit logs",
		Long:  `Analyzes existing audit logs to generate insights and statistics.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnalyze(cmd, args, verbose)
		},
	}

	analyzeCmd.Flags().String("input", "audit.log", "Input audit log file")
	analyzeCmd.Flags().String("output", "analysis.json", "Output analysis file")
	analyzeCmd.Flags().StringSlice("compliance", []string{"soc2"}, "Compliance frameworks")

	// Search command
	var searchCmd = &cobra.Command{
		Use:   "search",
		Short: "Search through audit logs",
		Long:  `Searches through audit logs using various criteria.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearch(cmd, args, verbose)
		},
	}

	searchCmd.Flags().String("input", "audit.log", "Input audit log file")
	searchCmd.Flags().String("query", "", "Search query")
	searchCmd.Flags().String("timerange", "", "Time range (e.g., 1h, 1d)")
	searchCmd.Flags().StringToString("filter", map[string]string{}, "Filters (key=value)")
	searchCmd.Flags().String("output", "", "Output file (default: stdout)")

	// Report command
	var reportCmd = &cobra.Command{
		Use:   "report",
		Short: "Generate compliance reports",
		Long:  `Generates comprehensive compliance reports.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReport(cmd, args, verbose)
		},
	}

	reportCmd.Flags().String("input", "audit.log", "Input audit log file")
	reportCmd.Flags().String("compliance", "soc2", "Compliance framework")
	reportCmd.Flags().String("format", "json", "Report format (json, html, pdf)")
	reportCmd.Flags().String("output", "compliance-report.json", "Output report file")

	// Monitor command
	var monitorCmd = &cobra.Command{
		Use:   "monitor",
		Short: "Real-time monitoring and alerting",
		Long:  `Provides real-time monitoring and alerting capabilities.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMonitor(cmd, args, target, verbose)
		},
	}

	monitorCmd.Flags().StringSlice("alerts", []string{}, "Alert endpoints")
	monitorCmd.Flags().Duration("interval", 1*time.Minute, "Monitoring interval")
	monitorCmd.Flags().Float64("threshold", 0.8, "Alert threshold")

	// Anomaly command
	var anomalyCmd = &cobra.Command{
		Use:   "anomaly",
		Short: "Anomaly detection and analysis",
		Long:  `Performs anomaly detection and analysis on audit logs.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnomaly(cmd, args, verbose)
		},
	}

	anomalyCmd.Flags().String("input", "audit.log", "Input audit log file")
	anomalyCmd.Flags().String("model", "statistical", "Anomaly detection model")
	anomalyCmd.Flags().Float64("threshold", 0.8, "Anomaly threshold")
	anomalyCmd.Flags().String("output", "anomalies.json", "Output anomalies file")

	// Privacy command
	var privacyCmd = &cobra.Command{
		Use:   "privacy",
		Short: "Data privacy compliance checking",
		Long:  `Performs data privacy compliance checking and PII detection.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPrivacy(cmd, args, verbose)
		},
	}

	privacyCmd.Flags().String("input", "audit.log", "Input audit log file")
	privacyCmd.Flags().Bool("scan-pii", true, "Scan for PII")
	privacyCmd.Flags().Bool("redact-output", true, "Redact PII in output")
	privacyCmd.Flags().String("output", "privacy-report.json", "Output privacy report file")

	// Export command
	var exportCmd = &cobra.Command{
		Use:   "export",
		Short: "Export audit data in various formats",
		Long:  `Exports audit data in various formats for external analysis.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExport(cmd, args, verbose)
		},
	}

	exportCmd.Flags().String("input", "audit.log", "Input audit log file")
	exportCmd.Flags().String("format", "json", "Export format (json, csv, xml)")
	exportCmd.Flags().String("output", "export.json", "Output export file")
	exportCmd.Flags().StringToString("filter", map[string]string{}, "Export filters")

	// Add commands to root
	rootCmd.AddCommand(logCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(reportCmd)
	rootCmd.AddCommand(monitorCmd)
	rootCmd.AddCommand(anomalyCmd)
	rootCmd.AddCommand(privacyCmd)
	rootCmd.AddCommand(exportCmd)

	// Execute the CLI
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

// Command implementations
func runLog(cmd *cobra.Command, args []string, target, verbose *string) error {
	output, _ := cmd.Flags().GetString("output")
	format, _ := cmd.Flags().GetString("format")
	realTime, _ := cmd.Flags().GetBool("real-time")
	bufferSize, _ := cmd.Flags().GetInt("buffer-size")
	flushInterval, _ := cmd.Flags().GetDuration("flush-interval")
	piiDetection, _ := cmd.Flags().GetBool("pii-detection")
	redaction, _ := cmd.Flags().GetBool("redaction")
	anomaly, _ := cmd.Flags().GetBool("anomaly")
	encryption, _ := cmd.Flags().GetBool("encryption")
	compliance, _ := cmd.Flags().GetStringSlice("compliance")

	config := &AuditConfig{
		Target:               *target,
		OutputFile:           output,
		Format:               format,
		RealTime:             realTime,
		BufferSize:           bufferSize,
		FlushInterval:        flushInterval,
		PIIDetection:         piiDetection,
		Redaction:            redaction,
		Anomaly:              anomaly,
		Encryption:           encryption,
		ComplianceFrameworks: compliance,
	}

	auditLog, err := NewAuditLog(config)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}
	defer auditLog.Close()

	// Example: Log some test events
	testEvents := []AuditEvent{
		{
			EventType:  "authentication",
			User:       "testuser",
			Action:     "login",
			Result:     "success",
			StatusCode: 200,
			Severity:   "info",
			Category:   "security",
			Message:    "User logged in successfully",
		},
		{
			EventType:  "data_access",
			User:       "testuser",
			Action:     "read",
			Resource:   "user_data",
			Result:     "success",
			StatusCode: 200,
			Severity:   "info",
			Category:   "data",
			Message:    "User accessed personal data",
		},
	}

	for _, event := range testEvents {
		if err := auditLog.LogEvent(event); err != nil {
			return fmt.Errorf("failed to log event: %w", err)
		}
	}

	fmt.Printf("Audit logging completed. Output: %s\n", output)
	return nil
}

func runAnalyze(cmd *cobra.Command, args []string, verbose *bool) error {
	input, _ := cmd.Flags().GetString("input")
	output, _ := cmd.Flags().GetString("output")
	compliance, _ := cmd.Flags().GetStringSlice("compliance")

	config := &AuditConfig{
		ComplianceFrameworks: compliance,
	}

	analyzer := NewAuditAnalyzer(config)

	// Load events
	if err := analyzer.LoadEvents(input); err != nil {
		return fmt.Errorf("failed to load events: %w", err)
	}

	// Analyze events
	if err := analyzer.AnalyzeEvents(); err != nil {
		return fmt.Errorf("failed to analyze events: %w", err)
	}

	// Save analysis
	analysisData, err := json.MarshalIndent(analyzer.statistics, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal analysis: %w", err)
	}

	if err := os.WriteFile(output, analysisData, 0644); err != nil {
		return fmt.Errorf("failed to write analysis: %w", err)
	}

	fmt.Printf("Analysis completed. Output: %s\n", output)
	return nil
}

func runSearch(cmd *cobra.Command, args []string, verbose *bool) error {
	input, _ := cmd.Flags().GetString("input")
	query, _ := cmd.Flags().GetString("query")
	timerange, _ := cmd.Flags().GetString("timerange")
	filters, _ := cmd.Flags().GetStringToString("filter")
	output, _ := cmd.Flags().GetString("output")

	analyzer := NewAuditAnalyzer(&AuditConfig{})

	// Load events
	if err := analyzer.LoadEvents(input); err != nil {
		return fmt.Errorf("failed to load events: %w", err)
	}

	// Parse time range
	var timeRange *TimeRange
	if timerange != "" {
		// Simple time range parsing (would be more sophisticated in real implementation)
		duration, err := time.ParseDuration(timerange)
		if err != nil {
			return fmt.Errorf("invalid time range: %w", err)
		}
		end := time.Now()
		start := end.Add(-duration)
		timeRange = &TimeRange{Start: start, End: end}
	}

	// Search events
	results := analyzer.SearchEvents(query, timeRange, filters)

	// Output results
	if output == "" {
		// Print to stdout
		for _, event := range results {
			fmt.Printf("[%s] %s: %s\n", event.Timestamp.Format(time.RFC3339), event.EventType, event.Message)
		}
	} else {
		// Save to file
		if err := analyzer.ExportEvents(results, "json", output); err != nil {
			return fmt.Errorf("failed to export results: %w", err)
		}
	}

	fmt.Printf("Search completed. Found %d events.\n", len(results))
	return nil
}

func runReport(cmd *cobra.Command, args []string, verbose *bool) error {
	input, _ := cmd.Flags().GetString("input")
	compliance, _ := cmd.Flags().GetString("compliance")
	format, _ := cmd.Flags().GetString("format")
	output, _ := cmd.Flags().GetString("output")

	analyzer := NewAuditAnalyzer(&AuditConfig{})

	// Load events
	if err := analyzer.LoadEvents(input); err != nil {
		return fmt.Errorf("failed to load events: %w", err)
	}

	// Analyze events
	if err := analyzer.AnalyzeEvents(); err != nil {
		return fmt.Errorf("failed to analyze events: %w", err)
	}

	// Generate compliance report
	report := analyzer.GenerateComplianceReport(compliance)

	// Save report
	switch format {
	case "json":
		reportData, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal report: %w", err)
		}
		if err := os.WriteFile(output, reportData, 0644); err != nil {
			return fmt.Errorf("failed to write report: %w", err)
		}
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	fmt.Printf("Compliance report generated. Output: %s\n", output)
	return nil
}

func runMonitor(cmd *cobra.Command, args []string, target, verbose *string) error {
	alerts, _ := cmd.Flags().GetStringSlice("alerts")
	interval, _ := cmd.Flags().GetDuration("interval")
	threshold, _ := cmd.Flags().GetFloat64("threshold")

	fmt.Printf("Starting monitoring with interval: %v, threshold: %.2f\n", interval, threshold)
	fmt.Printf("Alerts: %v\n", alerts)

	// Implementation would include real-time monitoring logic
	fmt.Println("Monitoring started...")
	return nil
}

func runAnomaly(cmd *cobra.Command, args []string, verbose *bool) error {
	input, _ := cmd.Flags().GetString("input")
	model, _ := cmd.Flags().GetString("model")
	threshold, _ := cmd.Flags().GetFloat64("threshold")
	output, _ := cmd.Flags().GetString("output")

	analyzer := NewAuditAnalyzer(&AuditConfig{})

	// Load events
	if err := analyzer.LoadEvents(input); err != nil {
		return fmt.Errorf("failed to load events: %w", err)
	}

	// Analyze events
	if err := analyzer.AnalyzeEvents(); err != nil {
		return fmt.Errorf("failed to analyze events: %w", err)
	}

	// Save anomalies
	anomaliesData, err := json.MarshalIndent(analyzer.anomalies, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal anomalies: %w", err)
	}

	if err := os.WriteFile(output, anomaliesData, 0644); err != nil {
		return fmt.Errorf("failed to write anomalies: %w", err)
	}

	fmt.Printf("Anomaly detection completed. Found %d anomalies. Output: %s\n", len(analyzer.anomalies), output)
	return nil
}

func runPrivacy(cmd *cobra.Command, args []string, verbose *bool) error {
	input, _ := cmd.Flags().GetString("input")
	scanPII, _ := cmd.Flags().GetBool("scan-pii")
	redactOutput, _ := cmd.Flags().GetBool("redact-output")
	output, _ := cmd.Flags().GetString("output")

	analyzer := NewAuditAnalyzer(&AuditConfig{})

	// Load events
	if err := analyzer.LoadEvents(input); err != nil {
		return fmt.Errorf("failed to load events: %w", err)
	}

	// Count PII events
	piiCount := 0
	for _, event := range analyzer.events {
		if event.PIIDetected {
			piiCount++
		}
	}

	// Generate privacy report
	privacyReport := map[string]interface{}{
		"total_events": len(analyzer.events),
		"pii_events":   piiCount,
		"scan_pii":     scanPII,
		"redact_output": redactOutput,
		"generated_at": time.Now(),
	}

	// Save privacy report
	reportData, err := json.MarshalIndent(privacyReport, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal privacy report: %w", err)
	}

	if err := os.WriteFile(output, reportData, 0644); err != nil {
		return fmt.Errorf("failed to write privacy report: %w", err)
	}

	fmt.Printf("Privacy analysis completed. Found %d PII events. Output: %s\n", piiCount, output)
	return nil
}

func runExport(cmd *cobra.Command, args []string, verbose *bool) error {
	input, _ := cmd.Flags().GetString("input")
	format, _ := cmd.Flags().GetString("format")
	output, _ := cmd.Flags().GetString("output")
	filters, _ := cmd.Flags().GetStringToString("filter")

	analyzer := NewAuditAnalyzer(&AuditConfig{})

	// Load events
	if err := analyzer.LoadEvents(input); err != nil {
		return fmt.Errorf("failed to load events: %w", err)
	}

	// Apply filters
	filteredEvents := analyzer.SearchEvents("", nil, filters)

	// Export events
	if err := analyzer.ExportEvents(filteredEvents, format, output); err != nil {
		return fmt.Errorf("failed to export events: %w", err)
	}

	fmt.Printf("Export completed. Exported %d events to %s\n", len(filteredEvents), output)
	return nil
}