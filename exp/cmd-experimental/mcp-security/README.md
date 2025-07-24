# mcp-security

Enterprise-grade security analysis and validation tool for MCP implementations.

## Overview

`mcp-security` is a comprehensive security testing and validation tool designed specifically for Model Context Protocol (MCP) implementations. It provides enterprise-grade security assessment capabilities including vulnerability scanning, authentication testing, access control validation, and compliance reporting against multiple frameworks.

## Features

### Core Security Testing
- **Vulnerability Scanning**: Comprehensive vulnerability assessment using pattern matching and behavior analysis
- **Input Validation Testing**: Extensive input validation and sanitization testing
- **Authentication Security**: OAuth2 security testing, session management analysis, and authentication bypass detection
- **Authorization Testing**: Access control validation and privilege escalation detection
- **Transport Security**: TLS configuration analysis and certificate validation
- **Error Handling Analysis**: Information disclosure detection in error messages
- **Rate Limiting Testing**: Rate limiting effectiveness and bypass detection

### Compliance Frameworks
- **SOC2**: Service Organization Control 2 compliance assessment
- **ISO 27001**: Information Security Management System compliance
- **GDPR**: General Data Protection Regulation compliance
- **HIPAA**: Health Insurance Portability and Accountability Act compliance
- **PCI DSS**: Payment Card Industry Data Security Standard compliance

### Advanced Features
- **Fuzzing**: Automated fuzzing for input validation testing
- **Continuous Monitoring**: Real-time security monitoring with alerting
- **Policy Validation**: Security policy compliance checking
- **Custom Checks**: Support for custom security checks and patterns
- **Multiple Output Formats**: JSON, XML, HTML, and PDF report generation

## Installation

```bash
go build -o mcp-security ./cmd/mcp-security
```

## Usage

### Basic Security Scan

```bash
mcp-security scan --target "stdio://./server" --verbose
```

### Comprehensive Scan with Compliance

```bash
mcp-security scan \
  --target "stdio://./server" \
  --compliance soc2,iso27001,gdpr \
  --output json \
  --report security-report.json \
  --verbose
```

### Authentication Testing

```bash
mcp-security auth \
  --target "sse://localhost:8080" \
  --oauth2-endpoint "https://auth.example.com/oauth2/token" \
  --client-id "your-client-id" \
  --verbose
```

### Fuzzing

```bash
mcp-security fuzz \
  --target "stdio://./server" \
  --method "tools/call" \
  --duration 10m \
  --threads 10
```

### Continuous Monitoring

```bash
mcp-security monitor \
  --target "stdio://./server" \
  --alerts "email:security@example.com,slack:webhook_url" \
  --interval 5m
```

## Commands

### scan
Perform comprehensive security scanning

**Options:**
- `--vuln-scan`: Enable vulnerability scanning (default: true)
- `--auth-test`: Enable authentication testing (default: true)
- `--fuzz-test`: Enable fuzzing tests (default: true)
- `--compliance`: Compliance frameworks to assess (default: soc2,iso27001)
- `--report`: Output report file (default: security-report.json)

**Example:**
```bash
mcp-security scan --target "stdio://./server" --compliance soc2,pci --report scan-results.json
```

### audit
Security audit and compliance checking

**Options:**
- `--compliance`: Compliance frameworks to audit (default: soc2)
- `--report`: Output audit report file (default: audit-report.json)

**Example:**
```bash
mcp-security audit --target "stdio://./server" --compliance gdpr,hipaa
```

### fuzz
Fuzzing and input validation testing

**Options:**
- `--method`: Method to fuzz (default: tools/call)
- `--duration`: Fuzzing duration (default: 5m)
- `--threads`: Number of fuzzing threads (default: 5)

**Example:**
```bash
mcp-security fuzz --target "stdio://./server" --duration 15m --threads 20
```

### auth
Authentication and authorization testing

**Options:**
- `--oauth2-endpoint`: OAuth2 endpoint URL
- `--client-id`: OAuth2 client ID
- `--client-secret`: OAuth2 client secret

**Example:**
```bash
mcp-security auth --target "sse://localhost:8080" --oauth2-endpoint "https://auth.example.com/token"
```

### policy
Security policy validation

**Options:**
- `--policy-config`: Security policy configuration file (default: security-policy.yaml)

**Example:**
```bash
mcp-security policy --target "stdio://./server" --policy-config custom-policy.yaml
```

### report
Generate compliance reports

**Options:**
- `--compliance`: Compliance frameworks to report (default: all)
- `--template`: Custom report template

**Example:**
```bash
mcp-security report --compliance soc2,iso27001 --output pdf --template custom-template.html
```

### monitor
Continuous security monitoring

**Options:**
- `--alerts`: Alert endpoints (email:user@domain.com, slack:webhook_url)
- `--interval`: Monitoring interval (default: 5m)

**Example:**
```bash
mcp-security monitor --target "stdio://./server" --alerts "email:team@company.com" --interval 2m
```

## Configuration

You can use a configuration file to specify detailed settings:

```json
{
  "target": "stdio://./server",
  "vuln_scan_enabled": true,
  "auth_test_enabled": true,
  "fuzz_test_enabled": true,
  "compliance_frameworks": ["soc2", "iso27001", "gdpr"],
  "output_format": "json",
  "verbose": true,
  "timeout": "30s",
  "max_concurrency": 10,
  "custom_checks": ["check1", "check2"],
  "policy_file": "security-policy.yaml",
  "report_file": "security-report.json",
  "alert_endpoints": ["email:security@example.com"],
  "tls_config": {
    "min_version": 771,
    "cert_file": "/path/to/cert.pem",
    "key_file": "/path/to/key.pem",
    "ca_file": "/path/to/ca.pem"
  }
}
```

Use the configuration file:
```bash
mcp-security scan --config security-config.json
```

## Security Checks

### Vulnerability Patterns

The tool includes detection for:

- **Input Validation**: XSS, SQL injection, command injection
- **Authentication**: Weak credentials, authentication bypass
- **Authorization**: Privilege escalation, access control bypass
- **Transport Security**: Insecure protocols, weak cipher suites
- **Information Disclosure**: Sensitive data exposure
- **Session Management**: Weak session tokens, session fixation
- **Configuration**: Insecure defaults, configuration exposure

### OWASP Top 10 Coverage

- A1: Injection
- A2: Broken Authentication
- A3: Sensitive Data Exposure
- A4: XML External Entities (XXE)
- A5: Broken Access Control
- A6: Security Misconfiguration
- A7: Cross-Site Scripting (XSS)
- A8: Insecure Deserialization
- A9: Using Components with Known Vulnerabilities
- A10: Insufficient Logging & Monitoring

## Compliance Frameworks

### SOC2 (Service Organization Control 2)
- CC6.1: Logical and Physical Access Controls
- CC6.2: Authentication and Authorization
- CC6.7: Data Transmission Security
- CC6.8: System Monitoring Activities

### ISO 27001
- A.9: Access Control
- A.10: Cryptography
- A.12: Operations Security
- A.14: System Acquisition, Development and Maintenance

### GDPR (General Data Protection Regulation)
- Article 32: Security of Processing
- Article 33: Notification of Personal Data Breach
- Article 34: Communication of Personal Data Breach to Data Subject
- Article 35: Data Protection Impact Assessment

### HIPAA (Health Insurance Portability and Accountability Act)
- 164.312(a)(1): Access Control
- 164.312(d): Person or Entity Authentication
- 164.312(e)(1): Transmission Security
- 164.312(a)(2)(iv): Encryption and Decryption

### PCI DSS (Payment Card Industry Data Security Standard)
- Requirement 6: Develop and Maintain Secure Systems
- Requirement 7: Restrict Access to Cardholder Data
- Requirement 8: Identify and Authenticate Access
- Requirement 11: Regularly Test Security Systems

## Report Format

### JSON Report Structure

```json
{
  "target": "stdio://./server",
  "timestamp": "2024-01-01T00:00:00Z",
  "duration": "5m30s",
  "total_issues": 15,
  "issues_by_severity": {
    "CRITICAL": 2,
    "HIGH": 5,
    "MEDIUM": 6,
    "LOW": 2
  },
  "issues": [
    {
      "id": "MCP-001-123456789",
      "title": "Cross-Site Scripting (XSS)",
      "description": "User input is reflected in output without proper encoding",
      "severity": "HIGH",
      "category": "Cross-Site Scripting",
      "cwe": "CWE-79",
      "cvss": 6.1,
      "location": "tool:example_tool",
      "evidence": "<script>alert('xss')</script>",
      "remediation": "Implement proper output encoding and Content Security Policy",
      "references": ["https://owasp.org/www-community/attacks/xss/"],
      "compliance": {
        "SOC2": false,
        "ISO27001": false,
        "GDPR": true
      },
      "metadata": {
        "tool_name": "example_tool",
        "payload": "<script>alert('xss')</script>"
      },
      "timestamp": "2024-01-01T00:00:00Z"
    }
  ],
  "compliance": {
    "frameworks": {
      "soc2": {
        "name": "soc2",
        "version": "1.0",
        "score": 0.75,
        "status": "PARTIAL",
        "controls": {
          "CC6.1": {
            "id": "CC6.1",
            "title": "Logical and Physical Access Controls",
            "description": "System implements logical and physical access controls",
            "status": "PASS",
            "score": 0.9,
            "evidence": "Access control mechanisms implemented",
            "remediation": "Strengthen access control policies",
            "last_tested": "2024-01-01T00:00:00Z"
          }
        },
        "gaps": ["Insufficient access control mechanisms"],
        "recommendations": ["Implement comprehensive access control policies"]
      }
    },
    "overall_score": 0.75,
    "passed_tests": 35,
    "failed_tests": 15,
    "total_tests": 50
  },
  "summary": "Security scan identified 15 issues across multiple categories...",
  "recommendations": [
    "Address critical and high-severity security issues immediately",
    "Implement comprehensive input validation and sanitization"
  ],
  "metadata": {
    "scanner_version": "1.0.0",
    "config": {...}
  }
}
```

## Integration

### CI/CD Pipeline

```yaml
# .github/workflows/security.yml
name: Security Scan
on: [push, pull_request]

jobs:
  security-scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Setup Go
        uses: actions/setup-go@v2
        with:
          go-version: 1.21
          
      - name: Build MCP Security
        run: go build -o mcp-security ./cmd/mcp-security
        
      - name: Run Security Scan
        run: |
          ./mcp-security scan \
            --target "stdio://./server" \
            --compliance soc2,iso27001 \
            --report security-report.json \
            --output json
            
      - name: Upload Security Report
        uses: actions/upload-artifact@v2
        with:
          name: security-report
          path: security-report.json
```

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o mcp-security ./cmd/mcp-security

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/mcp-security .
CMD ["./mcp-security"]
```

## Advanced Usage

### Custom Security Checks

Create custom security patterns:

```json
{
  "custom_patterns": [
    {
      "id": "CUSTOM-001",
      "name": "Custom Vulnerability",
      "description": "Custom vulnerability pattern",
      "pattern": "(?i)(custom|pattern)",
      "severity": "HIGH",
      "category": "Custom",
      "remediation": "Apply custom remediation"
    }
  ]
}
```

### Monitoring Integration

```bash
# Prometheus metrics endpoint
mcp-security monitor --metrics-port 9090

# Grafana dashboard
# Import dashboard ID: 12345
```

### Alerting

```bash
# Email alerts
mcp-security monitor --alerts "email:security@company.com"

# Slack integration
mcp-security monitor --alerts "slack:https://hooks.slack.com/services/..."

# Webhook
mcp-security monitor --alerts "webhook:https://api.company.com/alerts"
```

## Security Considerations

1. **Credentials**: Never include credentials in configuration files
2. **Network Access**: Ensure proper network segmentation for security testing
3. **Rate Limiting**: Respect target system rate limits during testing
4. **Data Handling**: Securely handle and store security test results
5. **Compliance**: Ensure testing activities comply with organizational policies

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
- Contact the security team
- Review the documentation

## Changelog

### v1.0.0
- Initial release
- Comprehensive vulnerability scanning
- Multi-framework compliance assessment
- Authentication and authorization testing
- Fuzzing capabilities
- Continuous monitoring
- Multiple output formats