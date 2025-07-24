# mcp-audit

Comprehensive audit logging and analysis tool for MCP implementations.

## Overview

`mcp-audit` is a comprehensive audit logging and analysis tool designed specifically for Model Context Protocol (MCP) implementations. It provides enterprise-grade audit trail generation, real-time analysis, compliance reporting, and security monitoring capabilities.

## Features

### Core Audit Capabilities
- **Comprehensive Audit Trail**: Complete audit logging of all MCP activities
- **Real-time Logging**: Live audit event capture and processing
- **Structured Logging**: JSON-based structured audit events
- **Event Correlation**: Intelligent correlation of related audit events
- **Anomaly Detection**: Behavioral and statistical anomaly detection
- **PII Detection**: Automatic detection and redaction of personally identifiable information
- **Integrity Protection**: Cryptographic hashing and digital signatures for audit integrity

### Analysis and Search
- **Advanced Search**: Full-text search with filters and time ranges
- **Statistical Analysis**: Comprehensive audit statistics and metrics
- **Pattern Recognition**: Detection of security patterns and threats
- **Behavioral Analysis**: User and system behavior analysis
- **Trend Analysis**: Historical trend analysis and reporting

### Compliance Frameworks
- **SOC2**: Service Organization Control 2 compliance monitoring
- **ISO 27001**: Information Security Management System compliance
- **GDPR**: General Data Protection Regulation compliance
- **HIPAA**: Health Insurance Portability and Accountability Act compliance
- **PCI DSS**: Payment Card Industry Data Security Standard compliance

### Export and Integration
- **Multiple Formats**: JSON, CSV, XML export capabilities
- **SIEM Integration**: Integration with Security Information and Event Management systems
- **Real-time Alerts**: Configurable alerting for security events
- **API Integration**: RESTful API for external integrations

## Installation

```bash
go build -o mcp-audit ./cmd/mcp-audit
```

## Usage

### Basic Audit Logging

```bash
mcp-audit log --target "stdio://./server" --output audit.log
```

### Real-time Monitoring

```bash
mcp-audit log \
  --target "stdio://./server" \
  --output audit.log \
  --real-time \
  --pii-detection \
  --anomaly \
  --compliance soc2,gdpr
```

### Log Analysis

```bash
mcp-audit analyze \
  --input audit.log \
  --output analysis.json \
  --compliance soc2,iso27001
```

### Search and Filter

```bash
mcp-audit search \
  --input audit.log \
  --query "authentication failed" \
  --timerange "24h" \
  --filter "severity=high"
```

### Compliance Reporting

```bash
mcp-audit report \
  --input audit.log \
  --compliance soc2 \
  --format json \
  --output compliance-report.json
```

### Anomaly Detection

```bash
mcp-audit anomaly \
  --input audit.log \
  --model statistical \
  --threshold 0.8 \
  --output anomalies.json
```

### Privacy Analysis

```bash
mcp-audit privacy \
  --input audit.log \
  --scan-pii \
  --redact-output \
  --output privacy-report.json
```

## Commands

### log
Generate and manage audit logs

**Options:**
- `--output`: Output file (default: audit.log)
- `--format`: Log format (json, csv) (default: json)
- `--real-time`: Enable real-time logging
- `--buffer-size`: Buffer size (default: 1000)
- `--flush-interval`: Flush interval (default: 5s)
- `--pii-detection`: Enable PII detection (default: true)
- `--redaction`: Enable PII redaction (default: true)
- `--anomaly`: Enable anomaly detection (default: true)
- `--encryption`: Enable encryption
- `--compliance`: Compliance frameworks (default: soc2)

**Example:**
```bash
mcp-audit log --target "stdio://./server" --real-time --pii-detection --anomaly
```

### analyze
Analyze existing audit logs

**Options:**
- `--input`: Input audit log file (default: audit.log)
- `--output`: Output analysis file (default: analysis.json)
- `--compliance`: Compliance frameworks (default: soc2)

**Example:**
```bash
mcp-audit analyze --input audit.log --compliance soc2,gdpr --output analysis.json
```

### search
Search through audit logs

**Options:**
- `--input`: Input audit log file (default: audit.log)
- `--query`: Search query
- `--timerange`: Time range (e.g., 1h, 1d, 7d)
- `--filter`: Filters (key=value pairs)
- `--output`: Output file (default: stdout)

**Example:**
```bash
mcp-audit search --query "failed login" --timerange "1h" --filter "user=admin"
```

### report
Generate compliance reports

**Options:**
- `--input`: Input audit log file (default: audit.log)
- `--compliance`: Compliance framework (default: soc2)
- `--format`: Report format (json, html, pdf) (default: json)
- `--output`: Output report file (default: compliance-report.json)

**Example:**
```bash
mcp-audit report --compliance gdpr --format html --output gdpr-report.html
```

### monitor
Real-time monitoring and alerting

**Options:**
- `--alerts`: Alert endpoints
- `--interval`: Monitoring interval (default: 1m)
- `--threshold`: Alert threshold (default: 0.8)

**Example:**
```bash
mcp-audit monitor --alerts "email:security@company.com" --interval 30s --threshold 0.9
```

### anomaly
Anomaly detection and analysis

**Options:**
- `--input`: Input audit log file (default: audit.log)
- `--model`: Anomaly detection model (default: statistical)
- `--threshold`: Anomaly threshold (default: 0.8)
- `--output`: Output anomalies file (default: anomalies.json)

**Example:**
```bash
mcp-audit anomaly --model behavioral --threshold 0.7 --output anomalies.json
```

### privacy
Data privacy compliance checking

**Options:**
- `--input`: Input audit log file (default: audit.log)
- `--scan-pii`: Scan for PII (default: true)
- `--redact-output`: Redact PII in output (default: true)
- `--output`: Output privacy report file (default: privacy-report.json)

**Example:**
```bash
mcp-audit privacy --scan-pii --redact-output --output privacy-report.json
```

### export
Export audit data in various formats

**Options:**
- `--input`: Input audit log file (default: audit.log)
- `--format`: Export format (json, csv, xml) (default: json)
- `--output`: Output export file (default: export.json)
- `--filter`: Export filters

**Example:**
```bash
mcp-audit export --format csv --filter "severity=high" --output high-severity.csv
```

## Configuration

You can use a configuration file to specify detailed settings:

```json
{
  "target": "stdio://./server",
  "output_file": "audit.log",
  "format": "json",
  "log_level": "info",
  "retention": "90d",
  "compression": true,
  "encryption": true,
  "real_time": true,
  "buffer_size": 1000,
  "flush_interval": "5s",
  "compliance_frameworks": ["soc2", "gdpr", "hipaa"],
  "pii_detection": true,
  "redaction": true,
  "anomaly": true,
  "alert_endpoints": ["email:security@company.com"],
  "metadata": {
    "environment": "production",
    "datacenter": "us-east-1"
  }
}
```

## Audit Event Structure

Each audit event follows a comprehensive structure:

```json
{
  "id": "audit_1234567890_12345",
  "timestamp": "2024-01-01T00:00:00Z",
  "event_type": "authentication",
  "source": "mcp-server",
  "target": "auth-service",
  "user": "john.doe",
  "session_id": "sess_abcdef123456",
  "action": "login",
  "resource": "user_account",
  "method": "POST",
  "parameters": {
    "username": "john.doe",
    "client_ip": "192.168.1.100"
  },
  "result": "success",
  "status_code": 200,
  "duration": "150ms",
  "ip_address": "192.168.1.100",
  "user_agent": "Mozilla/5.0...",
  "severity": "info",
  "category": "security",
  "message": "User logged in successfully",
  "pii_detected": false,
  "pii_types": [],
  "compliance": {
    "soc2": true,
    "gdpr": true,
    "hipaa": true
  },
  "risk_score": 0.2,
  "anomaly": false,
  "anomaly_score": 0.1,
  "correlation": ["auth_001", "auth_002"],
  "metadata": {
    "request_id": "req_123",
    "trace_id": "trace_456"
  },
  "hash": "sha256:abc123...",
  "signature": "sig_def456..."
}
```

## Compliance Frameworks

### SOC2 (Service Organization Control 2)
- **CC6.1**: Logical and Physical Access Controls
- **CC6.2**: Authentication and Authorization
- **CC6.7**: Data Transmission Controls
- **CC6.8**: System Monitoring Activities

### ISO 27001
- **A.9.1**: Business Requirements for Access Control
- **A.12.4**: Logging and Monitoring
- **A.14.2**: Security in Development and Support Processes

### GDPR (General Data Protection Regulation)
- **Article 32**: Security of Processing
- **Article 33**: Notification of Personal Data Breach
- **Article 34**: Communication of Personal Data Breach
- **Article 35**: Data Protection Impact Assessment

### HIPAA
- **164.312(a)**: Access Control
- **164.312(d)**: Person or Entity Authentication
- **164.312(e)**: Transmission Security

### PCI DSS
- **Requirement 10**: Track and Monitor Access
- **Requirement 11**: Regularly Test Security Systems

## PII Detection

The tool automatically detects various types of PII:

- **Social Security Numbers**: 123-45-6789
- **Credit Card Numbers**: 4111-1111-1111-1111
- **Email Addresses**: user@example.com
- **Phone Numbers**: (555) 123-4567
- **IP Addresses**: 192.168.1.1
- **Dates of Birth**: 01/01/1990
- **Driver's License Numbers**: DL123456789
- **Bank Account Numbers**: 1234567890123456
- **Passport Numbers**: A12345678
- **Medical Record Numbers**: MRN123456789

## Anomaly Detection

### Statistical Model
- **Event Rate Anomalies**: Unusual event frequency
- **Error Rate Anomalies**: Abnormal error patterns
- **User Activity Anomalies**: Unusual user behavior
- **Resource Usage Anomalies**: Abnormal resource access patterns

### Behavioral Model
- **User Behavior Profiling**: Learning normal user patterns
- **Time-based Analysis**: Detecting unusual timing patterns
- **Access Pattern Analysis**: Identifying abnormal access sequences
- **Geolocation Analysis**: Detecting location-based anomalies

### Machine Learning Models
- **Clustering**: Grouping similar events and detecting outliers
- **Classification**: Classifying events as normal or anomalous
- **Time Series Analysis**: Detecting trends and seasonal patterns
- **Deep Learning**: Neural network-based anomaly detection

## Search and Query

### Query Syntax
- **Text Search**: `authentication failed`
- **Field Search**: `user:admin`
- **Wildcard Search**: `user:john*`
- **Regex Search**: `user:/john\\.doe/`
- **Range Search**: `timestamp:[2024-01-01 TO 2024-01-31]`

### Filters
- **Event Type**: `event_type=authentication`
- **Severity**: `severity=high`
- **User**: `user=admin`
- **Status Code**: `status=404`
- **Time Range**: `timerange=1h`
- **PII Detection**: `pii_detected=true`
- **Anomaly**: `anomaly=true`

### Time Range Formats
- **Minutes**: `5m`, `30m`
- **Hours**: `1h`, `6h`, `24h`
- **Days**: `1d`, `7d`, `30d`
- **Weeks**: `1w`, `2w`
- **Months**: `1M`, `3M`, `6M`
- **Years**: `1y`

## Export Formats

### JSON
```json
[
  {
    "id": "audit_123",
    "timestamp": "2024-01-01T00:00:00Z",
    "event_type": "authentication",
    "user": "john.doe",
    "message": "User logged in successfully"
  }
]
```

### CSV
```csv
ID,Timestamp,EventType,User,Action,Resource,StatusCode,Severity,Message,PIIDetected,Anomaly,RiskScore
audit_123,2024-01-01T00:00:00Z,authentication,john.doe,login,user_account,200,info,User logged in successfully,false,false,0.2
```

### XML
```xml
<audit_events>
  <event>
    <id>audit_123</id>
    <timestamp>2024-01-01T00:00:00Z</timestamp>
    <event_type>authentication</event_type>
    <user>john.doe</user>
    <message>User logged in successfully</message>
  </event>
</audit_events>
```

## Integration

### CI/CD Pipeline

```yaml
name: Audit Compliance Check
on: [push, pull_request]

jobs:
  audit-compliance:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Build MCP Audit
        run: go build -o mcp-audit ./cmd/mcp-audit
        
      - name: Run Audit Analysis
        run: |
          ./mcp-audit analyze \
            --input audit.log \
            --compliance soc2,gdpr \
            --output analysis.json
            
      - name: Generate Compliance Report
        run: |
          ./mcp-audit report \
            --input audit.log \
            --compliance soc2 \
            --format json \
            --output compliance-report.json
            
      - name: Upload Reports
        uses: actions/upload-artifact@v2
        with:
          name: audit-reports
          path: |
            analysis.json
            compliance-report.json
```

### SIEM Integration

```bash
# Splunk
mcp-audit export --format json --output /splunk/logs/mcp-audit.json

# ELK Stack
mcp-audit export --format json | curl -X POST "localhost:9200/audit/_doc" -H "Content-Type: application/json" -d @-

# QRadar
mcp-audit export --format csv --output /qradar/import/mcp-audit.csv
```

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o mcp-audit ./cmd/mcp-audit

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/mcp-audit .
CMD ["./mcp-audit"]
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mcp-audit
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mcp-audit
  template:
    metadata:
      labels:
        app: mcp-audit
    spec:
      containers:
      - name: mcp-audit
        image: mcp-audit:latest
        command: ["./mcp-audit", "monitor"]
        volumeMounts:
        - name: audit-logs
          mountPath: /var/log/audit
      volumes:
      - name: audit-logs
        persistentVolumeClaim:
          claimName: audit-logs-pvc
```

## Advanced Features

### Real-time Alerting

```bash
# Email alerts
mcp-audit monitor --alerts "email:security@company.com,email:admin@company.com"

# Slack integration
mcp-audit monitor --alerts "slack:https://hooks.slack.com/services/..."

# Webhook alerts
mcp-audit monitor --alerts "webhook:https://api.company.com/alerts"

# Multiple alert types
mcp-audit monitor --alerts "email:security@company.com,slack:webhook_url,webhook:api_endpoint"
```

### Custom Correlation Rules

```json
{
  "rules": [
    {
      "id": "brute_force_detection",
      "name": "Brute Force Attack Detection",
      "description": "Detect brute force login attempts",
      "conditions": [
        {
          "field": "event_type",
          "operator": "equals",
          "value": "authentication"
        },
        {
          "field": "status_code",
          "operator": "gte",
          "value": 400
        }
      ],
      "window": "5m",
      "threshold": 5,
      "severity": "high",
      "action": "alert"
    }
  ]
}
```

### Performance Monitoring

```bash
# Monitor audit performance
mcp-audit monitor --metrics-port 9090

# Grafana dashboard
# Import dashboard for audit metrics visualization
```

## Security Considerations

1. **Log Integrity**: Use cryptographic hashing and digital signatures
2. **Encryption**: Encrypt audit logs at rest and in transit
3. **Access Control**: Implement proper access controls for audit logs
4. **Retention**: Follow compliance requirements for log retention
5. **Backup**: Implement secure backup and recovery procedures

## Troubleshooting

### Common Issues

1. **Permission Denied**: Ensure proper file permissions for log files
2. **Disk Space**: Monitor disk space for audit log storage
3. **Performance**: Adjust buffer size and flush interval for optimal performance
4. **Network**: Ensure network connectivity for real-time monitoring

### Debug Mode

```bash
mcp-audit log --verbose --target "stdio://./server" --output debug.log
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Implement your changes
4. Add tests for new functionality
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For support and questions:
- Open an issue on GitHub
- Contact the development team
- Review the documentation

## Changelog

### v1.0.0
- Initial release
- Comprehensive audit logging
- Multi-framework compliance support
- Real-time monitoring and alerting
- PII detection and redaction
- Anomaly detection
- Advanced search and analysis
- Multiple export formats