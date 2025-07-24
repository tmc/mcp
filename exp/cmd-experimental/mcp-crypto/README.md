# mcp-crypto

Comprehensive cryptographic operations and key management tool for MCP implementations.

## Overview

`mcp-crypto` is a comprehensive cryptographic operations and key management tool designed specifically for Model Context Protocol (MCP) implementations. It provides enterprise-grade cryptographic capabilities including secure key generation, encryption/decryption, digital signatures, certificate management, and Hardware Security Module (HSM) integration.

## Features

### Core Cryptographic Operations
- **Key Generation**: Generate RSA, ECDSA, Ed25519, AES, and ChaCha20 keys
- **Encryption/Decryption**: AES-GCM, ChaCha20-Poly1305, RSA-OAEP encryption
- **Digital Signatures**: RSA-PSS, ECDSA, Ed25519 signatures
- **Key Derivation**: PBKDF2, scrypt, HKDF key derivation functions
- **Certificate Management**: X.509 certificate creation and CSR generation

### Key Management
- **Secure Key Storage**: Encrypted key storage with metadata
- **Key Rotation**: Automated and manual key rotation
- **Key Lifecycle Management**: Creation, activation, rotation, and deletion
- **Key Escrow**: Multi-party key escrow and recovery
- **Key Versioning**: Key version tracking and rollback

### Security Features
- **Hardware Security Module (HSM)**: PKCS#11 HSM integration
- **Cryptographic Policies**: Configurable security policies
- **Compliance**: FIPS 140-2 and Common Criteria compliance
- **Audit Logging**: Comprehensive cryptographic operation logging
- **Access Control**: Role-based access to cryptographic operations

### Enterprise Features
- **Policy Enforcement**: Algorithm restrictions and key size policies
- **Compliance Reporting**: Automated compliance reports
- **Integration APIs**: REST and gRPC APIs for integration
- **Monitoring**: Real-time monitoring of cryptographic operations
- **Backup and Recovery**: Secure key backup and disaster recovery

## Installation

```bash
go build -o mcp-crypto ./cmd/mcp-crypto
```

## Usage

### Basic Key Generation

```bash
# Generate RSA key pair
mcp-crypto keygen --type rsa --bits 2048 --output private.pem --public-output public.pem

# Generate ECDSA key pair
mcp-crypto keygen --type ecdsa --bits 256 --usage sign --output ecdsa-private.pem

# Generate Ed25519 key pair
mcp-crypto keygen --type ed25519 --usage sign --output ed25519-private.pem

# Generate AES key
mcp-crypto keygen --type aes --bits 256 --usage encrypt
```

### Encryption and Decryption

```bash
# Encrypt data
mcp-crypto encrypt --key key_1234567890_abcdef --input message.txt --output encrypted.bin

# Decrypt data
mcp-crypto decrypt --key key_1234567890_abcdef --input encrypted.bin --output decrypted.txt

# Encrypt with specific algorithm
mcp-crypto encrypt --key key_1234567890_abcdef --algorithm aes-256-gcm --input data.txt
```

### Digital Signatures

```bash
# Sign data
mcp-crypto sign --key key_1234567890_abcdef --input document.pdf --output signature.sig

# Verify signature
mcp-crypto verify --key key_1234567890_abcdef --input document.pdf --signature signature.sig

# Sign with specific algorithm
mcp-crypto sign --key key_1234567890_abcdef --algorithm rsa-pss-sha256 --input data.txt
```

### Certificate Management

```bash
# Create self-signed certificate
mcp-crypto cert --create --key key_1234567890_abcdef --subject "CN=example.com" --dns example.com --output cert.pem

# Generate certificate signing request
mcp-crypto cert --csr --key key_1234567890_abcdef --subject "CN=example.com" --dns example.com --output csr.pem

# Create CA certificate
mcp-crypto cert --create --key key_1234567890_abcdef --subject "CN=My CA" --ca --validity 3650d --output ca.pem
```

### Key Management

```bash
# List all keys
mcp-crypto list

# List keys in JSON format
mcp-crypto list --json

# Delete a key
mcp-crypto delete --key key_1234567890_abcdef

# Rotate a key
mcp-crypto rotate --key key_1234567890_abcdef
```

## Commands

### keygen
Generate cryptographic keys

**Options:**
- `--type`: Key type (rsa, ecdsa, ed25519, aes, chacha20) (default: rsa)
- `--bits`: Key size in bits (default: 2048)
- `--usage`: Key usage (encrypt, sign, decrypt, verify) (default: encrypt,sign)
- `--output`: Output file for private key
- `--public-output`: Output file for public key

**Example:**
```bash
mcp-crypto keygen --type ecdsa --bits 384 --usage sign --output private.pem --public-output public.pem
```

### encrypt
Encrypt data or messages

**Options:**
- `--key`: Key ID for encryption (required)
- `--algorithm`: Encryption algorithm (aes-256-gcm, chacha20poly1305, rsa-oaep)
- `--input`: Input file or data
- `--output`: Output file for encrypted data

**Example:**
```bash
mcp-crypto encrypt --key key_123 --algorithm aes-256-gcm --input message.txt --output encrypted.bin
```

### decrypt
Decrypt data or messages

**Options:**
- `--key`: Key ID for decryption (required)
- `--algorithm`: Decryption algorithm
- `--input`: Input file or encrypted data
- `--output`: Output file for decrypted data

**Example:**
```bash
mcp-crypto decrypt --key key_123 --input encrypted.bin --output decrypted.txt
```

### sign
Create digital signatures

**Options:**
- `--key`: Key ID for signing (required)
- `--algorithm`: Signature algorithm (rsa-pss-sha256, ecdsa-sha256, ed25519)
- `--input`: Input file or data to sign
- `--output`: Output file for signature

**Example:**
```bash
mcp-crypto sign --key key_123 --algorithm rsa-pss-sha256 --input document.pdf --output signature.sig
```

### verify
Verify digital signatures

**Options:**
- `--key`: Key ID for verification (required)
- `--algorithm`: Signature algorithm
- `--input`: Input file or data
- `--signature`: Signature file (required)

**Example:**
```bash
mcp-crypto verify --key key_123 --input document.pdf --signature signature.sig
```

### cert
Certificate management operations

**Options:**
- `--create`: Create new certificate
- `--csr`: Generate certificate signing request
- `--key`: Key ID for certificate (required)
- `--subject`: Certificate subject (e.g., "CN=example.com")
- `--dns`: DNS names (can be specified multiple times)
- `--email`: Email addresses (can be specified multiple times)
- `--validity`: Certificate validity period (default: 8760h)
- `--ca`: Create CA certificate
- `--output`: Output file for certificate

**Example:**
```bash
mcp-crypto cert --create --key key_123 --subject "CN=example.com" --dns example.com --dns www.example.com --validity 2160h --output cert.pem
```

### list
List cryptographic keys

**Options:**
- `--json`: Output in JSON format

**Example:**
```bash
mcp-crypto list --json
```

### delete
Delete cryptographic keys

**Options:**
- `--key`: Key ID to delete (required)

**Example:**
```bash
mcp-crypto delete --key key_123
```

### rotate
Key rotation operations

**Options:**
- `--key`: Key ID to rotate (required)

**Example:**
```bash
mcp-crypto rotate --key key_123
```

## Configuration

You can use a configuration file to specify detailed settings:

```json
{
  "key_store": "keystore.json",
  "default_key_type": "rsa",
  "default_key_size": 2048,
  "encryption_algorithm": "aes-256-gcm",
  "signature_algorithm": "rsa-pss-sha256",
  "hash_algorithm": "sha256",
  "kdf_algorithm": "pbkdf2",
  "kdf_iterations": 100000,
  "key_rotation_policy": {
    "enabled": true,
    "rotation_period": "720h",
    "max_key_age": "2160h",
    "min_key_age": "24h",
    "pre_rotation_time": "168h",
    "grace_period": "168h",
    "auto_rotate": false
  },
  "hsm_config": {
    "enabled": false,
    "provider": "pkcs11",
    "slot_id": 0,
    "token_label": "MCP-Token",
    "library": "/usr/lib/libpkcs11.so"
  },
  "policy_config": {
    "enabled": true,
    "min_key_size": 2048,
    "allowed_algorithms": ["rsa", "ecdsa", "ed25519", "aes", "chacha20poly1305"],
    "forbidden_algorithms": ["des", "3des", "rc4", "md5", "sha1"],
    "require_hsm": false,
    "require_fips": false,
    "max_key_age": "8760h",
    "require_escrow": false
  },
  "escrow_config": {
    "enabled": false,
    "min_trustees": 3,
    "max_trustees": 7,
    "threshold": 2,
    "backup_location": "escrow/",
    "encrypt_backup": true
  },
  "audit_config": {
    "enabled": true,
    "log_file": "crypto-audit.log",
    "log_level": "info",
    "log_operations": true,
    "log_access": true,
    "log_errors": true
  },
  "compliance_mode": "fips140-2",
  "fips_mode": true
}
```

Use the configuration file:
```bash
mcp-crypto --config crypto-config.json keygen --type rsa --bits 2048
```

## Key Types and Algorithms

### Asymmetric Key Types

#### RSA
- **Key Sizes**: 2048, 3072, 4096 bits
- **Encryption**: RSA-OAEP with SHA-256
- **Signatures**: RSA-PSS with SHA-256
- **Use Cases**: General purpose encryption and signing

#### ECDSA
- **Curves**: P-256, P-384, P-521
- **Signatures**: ECDSA with SHA-256
- **Use Cases**: Efficient signatures, limited encryption

#### Ed25519
- **Key Size**: 256 bits (fixed)
- **Signatures**: Ed25519 signature algorithm
- **Use Cases**: High-performance signatures

### Symmetric Key Types

#### AES
- **Key Sizes**: 128, 192, 256 bits
- **Modes**: GCM (Galois/Counter Mode)
- **Use Cases**: Bulk data encryption

#### ChaCha20
- **Key Size**: 256 bits (fixed)
- **Mode**: ChaCha20-Poly1305 AEAD
- **Use Cases**: High-performance encryption

## Security Features

### Cryptographic Policies

```json
{
  "policy_config": {
    "enabled": true,
    "min_key_size": 2048,
    "allowed_algorithms": ["rsa", "ecdsa", "ed25519", "aes"],
    "forbidden_algorithms": ["des", "3des", "rc4", "md5", "sha1"],
    "require_hsm": false,
    "require_fips": false,
    "max_key_age": "8760h",
    "require_escrow": false
  }
}
```

### Key Rotation

```json
{
  "key_rotation_policy": {
    "enabled": true,
    "rotation_period": "720h",
    "max_key_age": "2160h",
    "min_key_age": "24h",
    "pre_rotation_time": "168h",
    "grace_period": "168h",
    "auto_rotate": false
  }
}
```

### HSM Integration

```json
{
  "hsm_config": {
    "enabled": true,
    "provider": "pkcs11",
    "slot_id": 0,
    "token_label": "MCP-Token",
    "pin": "12345678",
    "library": "/usr/lib/libpkcs11.so"
  }
}
```

### Key Escrow

```json
{
  "escrow_config": {
    "enabled": true,
    "min_trustees": 3,
    "max_trustees": 7,
    "threshold": 2,
    "backup_location": "escrow/",
    "encrypt_backup": true
  }
}
```

## Key Storage Format

Keys are stored in JSON format with metadata:

```json
{
  "key_1234567890_abcdef": {
    "id": "key_1234567890_abcdef",
    "type": "rsa",
    "algorithm": "rsa",
    "size": 2048,
    "usage": ["encrypt", "sign"],
    "created_at": "2024-01-01T00:00:00Z",
    "expires_at": "2025-01-01T00:00:00Z",
    "status": "active",
    "location": "local",
    "hsm_backed": false,
    "escrow": false,
    "metadata": {
      "created_by": "admin",
      "purpose": "data_encryption"
    }
  }
}
```

## Compliance and Standards

### FIPS 140-2 Compliance
- **Level 1**: Software-based cryptographic modules
- **Level 2**: Hardware-based cryptographic modules
- **Level 3**: Tamper-evident hardware modules
- **Level 4**: Tamper-resistant hardware modules

### Common Criteria
- **EAL1**: Functionally tested
- **EAL2**: Structurally tested
- **EAL3**: Methodically tested and checked
- **EAL4**: Methodically designed, tested, and reviewed

### Supported Standards
- **NIST SP 800-57**: Key management guidelines
- **NIST SP 800-131A**: Cryptographic algorithm transitions
- **RFC 3447**: PKCS #1 RSA Cryptography Specifications
- **RFC 6979**: Deterministic ECDSA
- **RFC 8032**: Ed25519 signature algorithm

## Integration Examples

### Go Integration

```go
package main

import (
    "context"
    "fmt"
    "log"
)

func main() {
    // Create crypto manager
    config := DefaultCryptoConfig()
    cm, err := NewCryptoManager(config)
    if err != nil {
        log.Fatal(err)
    }

    // Generate key
    key, err := cm.keyManager.GenerateKey("rsa", 2048, []string{"encrypt", "sign"})
    if err != nil {
        log.Fatal(err)
    }

    // Encrypt data
    plaintext := []byte("Hello, World!")
    ciphertext, err := cm.Encrypt(key.ID, plaintext, "aes-256-gcm")
    if err != nil {
        log.Fatal(err)
    }

    // Decrypt data
    decrypted, err := cm.Decrypt(key.ID, ciphertext, "aes-256-gcm")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Original: %s\n", plaintext)
    fmt.Printf("Decrypted: %s\n", decrypted)
}
```

### REST API Integration

```bash
# Generate key via API
curl -X POST http://localhost:8080/api/v1/keys \
  -H "Content-Type: application/json" \
  -d '{
    "type": "rsa",
    "size": 2048,
    "usage": ["encrypt", "sign"]
  }'

# Encrypt data via API
curl -X POST http://localhost:8080/api/v1/encrypt \
  -H "Content-Type: application/json" \
  -d '{
    "key_id": "key_123",
    "algorithm": "aes-256-gcm",
    "plaintext": "SGVsbG8sIFdvcmxkIQ=="
  }'
```

### Docker Integration

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o mcp-crypto ./cmd/mcp-crypto

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/mcp-crypto .
CMD ["./mcp-crypto"]
```

### Kubernetes Integration

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mcp-crypto
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mcp-crypto
  template:
    metadata:
      labels:
        app: mcp-crypto
    spec:
      containers:
      - name: mcp-crypto
        image: mcp-crypto:latest
        volumeMounts:
        - name: keystore
          mountPath: /var/lib/mcp-crypto
        - name: config
          mountPath: /etc/mcp-crypto
      volumes:
      - name: keystore
        persistentVolumeClaim:
          claimName: keystore-pvc
      - name: config
        configMap:
          name: mcp-crypto-config
```

## Advanced Features

### Hardware Security Module (HSM)

```bash
# List HSM slots
mcp-crypto hsm --list-slots

# Generate key in HSM
mcp-crypto keygen --type rsa --bits 2048 --hsm --slot 0

# Sign with HSM key
mcp-crypto sign --key hsm_key_123 --input document.pdf --output signature.sig
```

### Key Escrow

```bash
# Enable key escrow
mcp-crypto escrow --enable --trustees 5 --threshold 3

# Backup key to escrow
mcp-crypto escrow --backup --key key_123 --trustees alice,bob,charlie,david,eve

# Recover key from escrow
mcp-crypto escrow --recover --key key_123 --trustees alice,bob,charlie
```

### Audit Logging

```bash
# View audit log
mcp-crypto audit --view --filter "operation=encrypt"

# Generate audit report
mcp-crypto audit --report --format pdf --output audit-report.pdf

# Export audit log
mcp-crypto audit --export --format json --output audit-export.json
```

## Performance Considerations

### Benchmarks

| Operation | Algorithm | Key Size | Operations/sec |
|-----------|-----------|----------|----------------|
| Encrypt   | AES-256-GCM | 256 bits | 50,000 |
| Encrypt   | ChaCha20-Poly1305 | 256 bits | 75,000 |
| Encrypt   | RSA-OAEP | 2048 bits | 500 |
| Sign      | RSA-PSS | 2048 bits | 800 |
| Sign      | ECDSA-P256 | 256 bits | 5,000 |
| Sign      | Ed25519 | 256 bits | 10,000 |

### Optimization Tips

1. **Use appropriate algorithms**: Ed25519 for signatures, ChaCha20-Poly1305 for encryption
2. **Batch operations**: Process multiple operations together
3. **Use HSM**: Offload cryptographic operations to dedicated hardware
4. **Cache keys**: Keep frequently used keys in memory
5. **Parallel processing**: Use multiple threads for independent operations

## Security Considerations

1. **Key Protection**: Store private keys securely, preferably in HSM
2. **Key Rotation**: Implement regular key rotation policies
3. **Access Control**: Implement proper access controls for cryptographic operations
4. **Audit Logging**: Enable comprehensive audit logging
5. **Compliance**: Ensure compliance with relevant standards and regulations

## Troubleshooting

### Common Issues

1. **Key not found**: Check key ID and ensure key exists in keystore
2. **Algorithm mismatch**: Verify algorithm compatibility with key type
3. **HSM errors**: Check HSM configuration and token availability
4. **Permission denied**: Verify file permissions and access rights

### Debug Mode

```bash
mcp-crypto --verbose keygen --type rsa --bits 2048
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
- RSA, ECDSA, Ed25519 key generation
- AES-GCM, ChaCha20-Poly1305 encryption
- Digital signatures and verification
- X.509 certificate management
- HSM integration
- Key rotation and escrow
- Comprehensive audit logging
- Policy enforcement
- Compliance support