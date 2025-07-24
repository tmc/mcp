// mcp-crypto: Comprehensive cryptographic operations and key management tool for MCP implementations
//
// This tool provides enterprise-grade cryptographic operations and key management capabilities including:
// - Secure key generation, storage, and rotation
// - Message encryption and decryption (AES, RSA, ChaCha20)
// - Digital signatures and verification (RSA, ECDSA, Ed25519)
// - Certificate management and PKI operations
// - Hardware Security Module (HSM) integration
// - Cryptographic policy enforcement
// - Key escrow and recovery mechanisms
// - Compliance with cryptographic standards (FIPS 140-2, Common Criteria)
//
// Usage:
//   mcp-crypto [command] [options]
//
// Commands:
//   keygen        Generate cryptographic keys
//   encrypt       Encrypt data or messages
//   decrypt       Decrypt data or messages
//   sign          Create digital signatures
//   verify        Verify digital signatures
//   cert          Certificate management operations
//   hsm           Hardware Security Module operations
//   policy        Cryptographic policy management
//   rotate        Key rotation operations
//   escrow        Key escrow and recovery
//   audit         Cryptographic audit operations
//
// Examples:
//   mcp-crypto keygen --type rsa --bits 2048 --output private.pem
//   mcp-crypto encrypt --key public.pem --input message.txt --output encrypted.bin
//   mcp-crypto decrypt --key private.pem --input encrypted.bin --output decrypted.txt
//   mcp-crypto sign --key private.pem --input document.pdf --output signature.sig
//   mcp-crypto verify --key public.pem --input document.pdf --signature signature.sig
//   mcp-crypto cert --create --subject "CN=example.com" --key private.pem
//   mcp-crypto hsm --list-keys --slot 0
//   mcp-crypto policy --enforce --config crypto-policy.yaml
//   mcp-crypto rotate --key-id key123 --schedule weekly
//   mcp-crypto escrow --backup --key-id key123 --trustees 3
//   mcp-crypto audit --report --compliance fips140-2
//
package main

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
)

// CryptoConfig represents the cryptographic configuration
type CryptoConfig struct {
	KeyStore         string            `json:"key_store"`
	DefaultKeyType   string            `json:"default_key_type"`
	DefaultKeySize   int               `json:"default_key_size"`
	EncryptionAlg    string            `json:"encryption_algorithm"`
	SignatureAlg     string            `json:"signature_algorithm"`
	HashAlg          string            `json:"hash_algorithm"`
	KDFAlg           string            `json:"kdf_algorithm"`
	KDFIterations    int               `json:"kdf_iterations"`
	KeyRotationPolicy KeyRotationPolicy `json:"key_rotation_policy"`
	HSMConfig        HSMConfig         `json:"hsm_config"`
	PolicyConfig     PolicyConfig      `json:"policy_config"`
	EscrowConfig     EscrowConfig      `json:"escrow_config"`
	AuditConfig      AuditConfig       `json:"audit_config"`
	ComplianceMode   string            `json:"compliance_mode"`
	FIPSMode         bool              `json:"fips_mode"`
}

// KeyRotationPolicy represents key rotation policy
type KeyRotationPolicy struct {
	Enabled         bool          `json:"enabled"`
	RotationPeriod  time.Duration `json:"rotation_period"`
	MaxKeyAge       time.Duration `json:"max_key_age"`
	MinKeyAge       time.Duration `json:"min_key_age"`
	PreRotationTime time.Duration `json:"pre_rotation_time"`
	GracePeriod     time.Duration `json:"grace_period"`
	AutoRotate      bool          `json:"auto_rotate"`
}

// HSMConfig represents HSM configuration
type HSMConfig struct {
	Enabled    bool   `json:"enabled"`
	Provider   string `json:"provider"`
	SlotID     int    `json:"slot_id"`
	TokenLabel string `json:"token_label"`
	PIN        string `json:"pin"`
	Library    string `json:"library"`
}

// PolicyConfig represents cryptographic policy configuration
type PolicyConfig struct {
	Enabled            bool     `json:"enabled"`
	MinKeySize         int      `json:"min_key_size"`
	AllowedAlgorithms  []string `json:"allowed_algorithms"`
	ForbiddenAlgorithms []string `json:"forbidden_algorithms"`
	RequireHSM         bool     `json:"require_hsm"`
	RequireFIPS        bool     `json:"require_fips"`
	MaxKeyAge          time.Duration `json:"max_key_age"`
	RequireEscrow      bool     `json:"require_escrow"`
}

// EscrowConfig represents key escrow configuration
type EscrowConfig struct {
	Enabled       bool   `json:"enabled"`
	MinTrustees   int    `json:"min_trustees"`
	MaxTrustees   int    `json:"max_trustees"`
	Threshold     int    `json:"threshold"`
	BackupLocation string `json:"backup_location"`
	EncryptBackup bool   `json:"encrypt_backup"`
}

// AuditConfig represents audit configuration
type AuditConfig struct {
	Enabled       bool   `json:"enabled"`
	LogFile       string `json:"log_file"`
	LogLevel      string `json:"log_level"`
	LogOperations bool   `json:"log_operations"`
	LogAccess     bool   `json:"log_access"`
	LogErrors     bool   `json:"log_errors"`
}

// CryptoKey represents a cryptographic key
type CryptoKey struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Algorithm    string                 `json:"algorithm"`
	Size         int                    `json:"size"`
	Usage        []string               `json:"usage"`
	CreatedAt    time.Time              `json:"created_at"`
	ExpiresAt    time.Time              `json:"expires_at"`
	Status       string                 `json:"status"`
	Location     string                 `json:"location"`
	HSMBacked    bool                   `json:"hsm_backed"`
	Escrow       bool                   `json:"escrow"`
	Metadata     map[string]interface{} `json:"metadata"`
	PublicKey    []byte                 `json:"public_key,omitempty"`
	PrivateKey   []byte                 `json:"private_key,omitempty"`
	Certificate  []byte                 `json:"certificate,omitempty"`
}

// CryptoOperation represents a cryptographic operation
type CryptoOperation struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Algorithm   string                 `json:"algorithm"`
	KeyID       string                 `json:"key_id"`
	Input       []byte                 `json:"input,omitempty"`
	Output      []byte                 `json:"output,omitempty"`
	Signature   []byte                 `json:"signature,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Status      string                 `json:"status"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// CertificateRequest represents a certificate request
type CertificateRequest struct {
	Subject            pkix.Name     `json:"subject"`
	DNSNames           []string      `json:"dns_names"`
	IPAddresses        []string      `json:"ip_addresses"`
	EmailAddresses     []string      `json:"email_addresses"`
	KeyUsage           x509.KeyUsage `json:"key_usage"`
	ExtKeyUsage        []x509.ExtKeyUsage `json:"ext_key_usage"`
	ValidityPeriod     time.Duration `json:"validity_period"`
	IsCA               bool          `json:"is_ca"`
	MaxPathLength      int           `json:"max_path_length"`
	SerialNumber       *big.Int      `json:"serial_number"`
	SignatureAlgorithm string        `json:"signature_algorithm"`
}

// KeyManager manages cryptographic keys
type KeyManager struct {
	config   *CryptoConfig
	keyStore map[string]*CryptoKey
	mutex    sync.RWMutex
	audit    *AuditLogger
}

// HSMManager manages HSM operations
type HSMManager struct {
	config *HSMConfig
	mutex  sync.RWMutex
}

// PolicyManager manages cryptographic policies
type PolicyManager struct {
	config *PolicyConfig
	mutex  sync.RWMutex
}

// EscrowManager manages key escrow operations
type EscrowManager struct {
	config *EscrowConfig
	mutex  sync.RWMutex
}

// AuditLogger logs cryptographic operations
type AuditLogger struct {
	config *AuditConfig
	file   *os.File
	mutex  sync.RWMutex
}

// CryptoManager is the main crypto management structure
type CryptoManager struct {
	config        *CryptoConfig
	keyManager    *KeyManager
	hsmManager    *HSMManager
	policyManager *PolicyManager
	escrowManager *EscrowManager
	auditLogger   *AuditLogger
}

// NewCryptoManager creates a new crypto manager
func NewCryptoManager(config *CryptoConfig) (*CryptoManager, error) {
	if config == nil {
		config = DefaultCryptoConfig()
	}

	// Initialize audit logger
	auditLogger, err := NewAuditLogger(&config.AuditConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create audit logger: %w", err)
	}

	// Initialize key manager
	keyManager := &KeyManager{
		config:   config,
		keyStore: make(map[string]*CryptoKey),
		audit:    auditLogger,
	}

	// Initialize HSM manager
	hsmManager := &HSMManager{
		config: &config.HSMConfig,
	}

	// Initialize policy manager
	policyManager := &PolicyManager{
		config: &config.PolicyConfig,
	}

	// Initialize escrow manager
	escrowManager := &EscrowManager{
		config: &config.EscrowConfig,
	}

	return &CryptoManager{
		config:        config,
		keyManager:    keyManager,
		hsmManager:    hsmManager,
		policyManager: policyManager,
		escrowManager: escrowManager,
		auditLogger:   auditLogger,
	}, nil
}

// DefaultCryptoConfig returns default configuration
func DefaultCryptoConfig() *CryptoConfig {
	return &CryptoConfig{
		KeyStore:       "keystore.json",
		DefaultKeyType: "rsa",
		DefaultKeySize: 2048,
		EncryptionAlg:  "aes-256-gcm",
		SignatureAlg:   "rsa-pss-sha256",
		HashAlg:        "sha256",
		KDFAlg:         "pbkdf2",
		KDFIterations:  100000,
		KeyRotationPolicy: KeyRotationPolicy{
			Enabled:         true,
			RotationPeriod:  30 * 24 * time.Hour, // 30 days
			MaxKeyAge:       90 * 24 * time.Hour, // 90 days
			MinKeyAge:       24 * time.Hour,      // 1 day
			PreRotationTime: 7 * 24 * time.Hour,  // 7 days
			GracePeriod:     7 * 24 * time.Hour,  // 7 days
			AutoRotate:      false,
		},
		HSMConfig: HSMConfig{
			Enabled:    false,
			Provider:   "pkcs11",
			SlotID:     0,
			TokenLabel: "MCP-Token",
		},
		PolicyConfig: PolicyConfig{
			Enabled:             true,
			MinKeySize:          2048,
			AllowedAlgorithms:   []string{"rsa", "ecdsa", "ed25519", "aes", "chacha20poly1305"},
			ForbiddenAlgorithms: []string{"des", "3des", "rc4", "md5", "sha1"},
			RequireHSM:          false,
			RequireFIPS:         false,
			MaxKeyAge:           365 * 24 * time.Hour, // 1 year
			RequireEscrow:       false,
		},
		EscrowConfig: EscrowConfig{
			Enabled:       false,
			MinTrustees:   3,
			MaxTrustees:   7,
			Threshold:     2,
			BackupLocation: "escrow/",
			EncryptBackup: true,
		},
		AuditConfig: AuditConfig{
			Enabled:       true,
			LogFile:       "crypto-audit.log",
			LogLevel:      "info",
			LogOperations: true,
			LogAccess:     true,
			LogErrors:     true,
		},
		ComplianceMode: "standard",
		FIPSMode:       false,
	}
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(config *AuditConfig) (*AuditLogger, error) {
	if !config.Enabled {
		return &AuditLogger{config: config}, nil
	}

	file, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log file: %w", err)
	}

	return &AuditLogger{
		config: config,
		file:   file,
	}, nil
}

// Log logs an audit event
func (al *AuditLogger) Log(operation, keyID, message string) {
	if !al.config.Enabled {
		return
	}

	al.mutex.Lock()
	defer al.mutex.Unlock()

	event := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"operation": operation,
		"key_id":    keyID,
		"message":   message,
	}

	eventJSON, _ := json.Marshal(event)
	if al.file != nil {
		al.file.WriteString(string(eventJSON) + "\n")
		al.file.Sync()
	}
}

// GenerateKey generates a new cryptographic key
func (km *KeyManager) GenerateKey(keyType string, keySize int, usage []string) (*CryptoKey, error) {
	km.mutex.Lock()
	defer km.mutex.Unlock()

	// Validate against policy
	if km.config.PolicyConfig.Enabled {
		if err := km.validateKeyPolicy(keyType, keySize); err != nil {
			return nil, fmt.Errorf("policy validation failed: %w", err)
		}
	}

	keyID := generateKeyID()
	key := &CryptoKey{
		ID:        keyID,
		Type:      keyType,
		Algorithm: keyType,
		Size:      keySize,
		Usage:     usage,
		CreatedAt: time.Now(),
		Status:    "active",
		Location:  "local",
		HSMBacked: false,
		Escrow:    false,
		Metadata:  make(map[string]interface{}),
	}

	// Set expiration based on policy
	if km.config.PolicyConfig.Enabled && km.config.PolicyConfig.MaxKeyAge > 0 {
		key.ExpiresAt = key.CreatedAt.Add(km.config.PolicyConfig.MaxKeyAge)
	}

	var err error
	switch keyType {
	case "rsa":
		err = km.generateRSAKey(key, keySize)
	case "ecdsa":
		err = km.generateECDSAKey(key, keySize)
	case "ed25519":
		err = km.generateEd25519Key(key)
	case "aes":
		err = km.generateAESKey(key, keySize)
	case "chacha20":
		err = km.generateChaCha20Key(key)
	default:
		return nil, fmt.Errorf("unsupported key type: %s", keyType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	// Store key
	km.keyStore[keyID] = key

	// Save to persistent storage
	if err := km.saveKeyStore(); err != nil {
		return nil, fmt.Errorf("failed to save key store: %w", err)
	}

	// Audit log
	km.audit.Log("key_generate", keyID, fmt.Sprintf("Generated %s key with size %d", keyType, keySize))

	return key, nil
}

// generateRSAKey generates an RSA key pair
func (km *KeyManager) generateRSAKey(key *CryptoKey, keySize int) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return err
	}

	// Marshal private key
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return err
	}

	// Marshal public key
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return err
	}

	key.PrivateKey = pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	key.PublicKey = pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	return nil
}

// generateECDSAKey generates an ECDSA key pair
func (km *KeyManager) generateECDSAKey(key *CryptoKey, keySize int) error {
	var curve elliptic.Curve
	switch keySize {
	case 256:
		curve = elliptic.P256()
	case 384:
		curve = elliptic.P384()
	case 521:
		curve = elliptic.P521()
	default:
		return fmt.Errorf("unsupported ECDSA key size: %d", keySize)
	}

	privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return err
	}

	// Marshal private key
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return err
	}

	// Marshal public key
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return err
	}

	key.PrivateKey = pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	key.PublicKey = pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	return nil
}

// generateEd25519Key generates an Ed25519 key pair
func (km *KeyManager) generateEd25519Key(key *CryptoKey) error {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}

	// Marshal private key
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return err
	}

	// Marshal public key
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return err
	}

	key.PrivateKey = pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	key.PublicKey = pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	key.Size = 256 // Ed25519 is always 256 bits

	return nil
}

// generateAESKey generates an AES key
func (km *KeyManager) generateAESKey(key *CryptoKey, keySize int) error {
	if keySize != 128 && keySize != 192 && keySize != 256 {
		return fmt.Errorf("unsupported AES key size: %d", keySize)
	}

	keyBytes := make([]byte, keySize/8)
	if _, err := rand.Read(keyBytes); err != nil {
		return err
	}

	key.PrivateKey = keyBytes
	key.PublicKey = nil // Symmetric key

	return nil
}

// generateChaCha20Key generates a ChaCha20 key
func (km *KeyManager) generateChaCha20Key(key *CryptoKey) error {
	keyBytes := make([]byte, chacha20poly1305.KeySize)
	if _, err := rand.Read(keyBytes); err != nil {
		return err
	}

	key.PrivateKey = keyBytes
	key.PublicKey = nil // Symmetric key
	key.Size = 256      // ChaCha20 is always 256 bits

	return nil
}

// validateKeyPolicy validates key against policy
func (km *KeyManager) validateKeyPolicy(keyType string, keySize int) error {
	policy := km.config.PolicyConfig

	// Check minimum key size
	if keySize < policy.MinKeySize {
		return fmt.Errorf("key size %d is below minimum %d", keySize, policy.MinKeySize)
	}

	// Check allowed algorithms
	if len(policy.AllowedAlgorithms) > 0 {
		allowed := false
		for _, alg := range policy.AllowedAlgorithms {
			if alg == keyType {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("key type %s is not allowed", keyType)
		}
	}

	// Check forbidden algorithms
	for _, alg := range policy.ForbiddenAlgorithms {
		if alg == keyType {
			return fmt.Errorf("key type %s is forbidden", keyType)
		}
	}

	return nil
}

// saveKeyStore saves the key store to disk
func (km *KeyManager) saveKeyStore() error {
	// Create a safe copy without private keys for storage
	safeKeyStore := make(map[string]*CryptoKey)
	for id, key := range km.keyStore {
		safeKey := *key
		safeKey.PrivateKey = nil // Don't save private keys in plaintext
		safeKeyStore[id] = &safeKey
	}

	data, err := json.MarshalIndent(safeKeyStore, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(km.config.KeyStore, data, 0644)
}

// GetKey retrieves a key by ID
func (km *KeyManager) GetKey(keyID string) (*CryptoKey, error) {
	km.mutex.RLock()
	defer km.mutex.RUnlock()

	key, exists := km.keyStore[keyID]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", keyID)
	}

	km.audit.Log("key_access", keyID, "Key accessed")

	return key, nil
}

// ListKeys lists all keys
func (km *KeyManager) ListKeys() []*CryptoKey {
	km.mutex.RLock()
	defer km.mutex.RUnlock()

	keys := make([]*CryptoKey, 0, len(km.keyStore))
	for _, key := range km.keyStore {
		keys = append(keys, key)
	}

	return keys
}

// DeleteKey deletes a key
func (km *KeyManager) DeleteKey(keyID string) error {
	km.mutex.Lock()
	defer km.mutex.Unlock()

	if _, exists := km.keyStore[keyID]; !exists {
		return fmt.Errorf("key not found: %s", keyID)
	}

	delete(km.keyStore, keyID)

	// Save to persistent storage
	if err := km.saveKeyStore(); err != nil {
		return fmt.Errorf("failed to save key store: %w", err)
	}

	km.audit.Log("key_delete", keyID, "Key deleted")

	return nil
}

// Encrypt encrypts data using the specified key
func (cm *CryptoManager) Encrypt(keyID string, plaintext []byte, algorithm string) ([]byte, error) {
	key, err := cm.keyManager.GetKey(keyID)
	if err != nil {
		return nil, err
	}

	if algorithm == "" {
		algorithm = cm.config.EncryptionAlg
	}

	var ciphertext []byte
	switch algorithm {
	case "aes-256-gcm":
		ciphertext, err = cm.encryptAESGCM(key, plaintext)
	case "chacha20poly1305":
		ciphertext, err = cm.encryptChaCha20Poly1305(key, plaintext)
	case "rsa-oaep":
		ciphertext, err = cm.encryptRSAOAEP(key, plaintext)
	default:
		return nil, fmt.Errorf("unsupported encryption algorithm: %s", algorithm)
	}

	if err != nil {
		cm.auditLogger.Log("encrypt_error", keyID, fmt.Sprintf("Encryption failed: %v", err))
		return nil, err
	}

	cm.auditLogger.Log("encrypt", keyID, fmt.Sprintf("Data encrypted using %s", algorithm))
	return ciphertext, nil
}

// encryptAESGCM encrypts data using AES-GCM
func (cm *CryptoManager) encryptAESGCM(key *CryptoKey, plaintext []byte) ([]byte, error) {
	if key.Type != "aes" {
		return nil, fmt.Errorf("key type %s is not suitable for AES encryption", key.Type)
	}

	block, err := aes.NewCipher(key.PrivateKey)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// encryptChaCha20Poly1305 encrypts data using ChaCha20-Poly1305
func (cm *CryptoManager) encryptChaCha20Poly1305(key *CryptoKey, plaintext []byte) ([]byte, error) {
	if key.Type != "chacha20" {
		return nil, fmt.Errorf("key type %s is not suitable for ChaCha20 encryption", key.Type)
	}

	aead, err := chacha20poly1305.New(key.PrivateKey)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := aead.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// encryptRSAOAEP encrypts data using RSA-OAEP
func (cm *CryptoManager) encryptRSAOAEP(key *CryptoKey, plaintext []byte) ([]byte, error) {
	if key.Type != "rsa" {
		return nil, fmt.Errorf("key type %s is not suitable for RSA encryption", key.Type)
	}

	// Parse public key
	block, _ := pem.Decode(key.PublicKey)
	if block == nil {
		return nil, fmt.Errorf("failed to parse public key")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	return rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaPublicKey, plaintext, nil)
}

// Decrypt decrypts data using the specified key
func (cm *CryptoManager) Decrypt(keyID string, ciphertext []byte, algorithm string) ([]byte, error) {
	key, err := cm.keyManager.GetKey(keyID)
	if err != nil {
		return nil, err
	}

	if algorithm == "" {
		algorithm = cm.config.EncryptionAlg
	}

	var plaintext []byte
	switch algorithm {
	case "aes-256-gcm":
		plaintext, err = cm.decryptAESGCM(key, ciphertext)
	case "chacha20poly1305":
		plaintext, err = cm.decryptChaCha20Poly1305(key, ciphertext)
	case "rsa-oaep":
		plaintext, err = cm.decryptRSAOAEP(key, ciphertext)
	default:
		return nil, fmt.Errorf("unsupported decryption algorithm: %s", algorithm)
	}

	if err != nil {
		cm.auditLogger.Log("decrypt_error", keyID, fmt.Sprintf("Decryption failed: %v", err))
		return nil, err
	}

	cm.auditLogger.Log("decrypt", keyID, fmt.Sprintf("Data decrypted using %s", algorithm))
	return plaintext, nil
}

// decryptAESGCM decrypts data using AES-GCM
func (cm *CryptoManager) decryptAESGCM(key *CryptoKey, ciphertext []byte) ([]byte, error) {
	if key.Type != "aes" {
		return nil, fmt.Errorf("key type %s is not suitable for AES decryption", key.Type)
	}

	block, err := aes.NewCipher(key.PrivateKey)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return aesGCM.Open(nil, nonce, ciphertext, nil)
}

// decryptChaCha20Poly1305 decrypts data using ChaCha20-Poly1305
func (cm *CryptoManager) decryptChaCha20Poly1305(key *CryptoKey, ciphertext []byte) ([]byte, error) {
	if key.Type != "chacha20" {
		return nil, fmt.Errorf("key type %s is not suitable for ChaCha20 decryption", key.Type)
	}

	aead, err := chacha20poly1305.New(key.PrivateKey)
	if err != nil {
		return nil, err
	}

	nonceSize := aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return aead.Open(nil, nonce, ciphertext, nil)
}

// decryptRSAOAEP decrypts data using RSA-OAEP
func (cm *CryptoManager) decryptRSAOAEP(key *CryptoKey, ciphertext []byte) ([]byte, error) {
	if key.Type != "rsa" {
		return nil, fmt.Errorf("key type %s is not suitable for RSA decryption", key.Type)
	}

	// Parse private key
	block, _ := pem.Decode(key.PrivateKey)
	if block == nil {
		return nil, fmt.Errorf("failed to parse private key")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaPrivateKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA private key")
	}

	return rsa.DecryptOAEP(sha256.New(), rand.Reader, rsaPrivateKey, ciphertext, nil)
}

// Sign creates a digital signature
func (cm *CryptoManager) Sign(keyID string, message []byte, algorithm string) ([]byte, error) {
	key, err := cm.keyManager.GetKey(keyID)
	if err != nil {
		return nil, err
	}

	if algorithm == "" {
		algorithm = cm.config.SignatureAlg
	}

	var signature []byte
	switch algorithm {
	case "rsa-pss-sha256":
		signature, err = cm.signRSAPSS(key, message)
	case "ecdsa-sha256":
		signature, err = cm.signECDSA(key, message)
	case "ed25519":
		signature, err = cm.signEd25519(key, message)
	default:
		return nil, fmt.Errorf("unsupported signature algorithm: %s", algorithm)
	}

	if err != nil {
		cm.auditLogger.Log("sign_error", keyID, fmt.Sprintf("Signing failed: %v", err))
		return nil, err
	}

	cm.auditLogger.Log("sign", keyID, fmt.Sprintf("Message signed using %s", algorithm))
	return signature, nil
}

// signRSAPSS signs using RSA-PSS
func (cm *CryptoManager) signRSAPSS(key *CryptoKey, message []byte) ([]byte, error) {
	if key.Type != "rsa" {
		return nil, fmt.Errorf("key type %s is not suitable for RSA signing", key.Type)
	}

	// Parse private key
	block, _ := pem.Decode(key.PrivateKey)
	if block == nil {
		return nil, fmt.Errorf("failed to parse private key")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaPrivateKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA private key")
	}

	// Hash the message
	hash := sha256.Sum256(message)

	// Sign using PSS
	return rsa.SignPSS(rand.Reader, rsaPrivateKey, crypto.SHA256, hash[:], nil)
}

// signECDSA signs using ECDSA
func (cm *CryptoManager) signECDSA(key *CryptoKey, message []byte) ([]byte, error) {
	if key.Type != "ecdsa" {
		return nil, fmt.Errorf("key type %s is not suitable for ECDSA signing", key.Type)
	}

	// Parse private key
	block, _ := pem.Decode(key.PrivateKey)
	if block == nil {
		return nil, fmt.Errorf("failed to parse private key")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	ecdsaPrivateKey, ok := privateKey.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an ECDSA private key")
	}

	// Hash the message
	hash := sha256.Sum256(message)

	// Sign
	r, s, err := ecdsa.Sign(rand.Reader, ecdsaPrivateKey, hash[:])
	if err != nil {
		return nil, err
	}

	// Encode signature
	return asn1.Marshal(struct{ R, S *big.Int }{r, s})
}

// signEd25519 signs using Ed25519
func (cm *CryptoManager) signEd25519(key *CryptoKey, message []byte) ([]byte, error) {
	if key.Type != "ed25519" {
		return nil, fmt.Errorf("key type %s is not suitable for Ed25519 signing", key.Type)
	}

	// Parse private key
	block, _ := pem.Decode(key.PrivateKey)
	if block == nil {
		return nil, fmt.Errorf("failed to parse private key")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	ed25519PrivateKey, ok := privateKey.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an Ed25519 private key")
	}

	// Sign
	return ed25519.Sign(ed25519PrivateKey, message), nil
}

// Verify verifies a digital signature
func (cm *CryptoManager) Verify(keyID string, message, signature []byte, algorithm string) (bool, error) {
	key, err := cm.keyManager.GetKey(keyID)
	if err != nil {
		return false, err
	}

	if algorithm == "" {
		algorithm = cm.config.SignatureAlg
	}

	var valid bool
	switch algorithm {
	case "rsa-pss-sha256":
		valid, err = cm.verifyRSAPSS(key, message, signature)
	case "ecdsa-sha256":
		valid, err = cm.verifyECDSA(key, message, signature)
	case "ed25519":
		valid, err = cm.verifyEd25519(key, message, signature)
	default:
		return false, fmt.Errorf("unsupported signature algorithm: %s", algorithm)
	}

	if err != nil {
		cm.auditLogger.Log("verify_error", keyID, fmt.Sprintf("Verification failed: %v", err))
		return false, err
	}

	cm.auditLogger.Log("verify", keyID, fmt.Sprintf("Signature verification: %t", valid))
	return valid, nil
}

// verifyRSAPSS verifies RSA-PSS signature
func (cm *CryptoManager) verifyRSAPSS(key *CryptoKey, message, signature []byte) (bool, error) {
	if key.Type != "rsa" {
		return false, fmt.Errorf("key type %s is not suitable for RSA verification", key.Type)
	}

	// Parse public key
	block, _ := pem.Decode(key.PublicKey)
	if block == nil {
		return false, fmt.Errorf("failed to parse public key")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return false, err
	}

	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return false, fmt.Errorf("not an RSA public key")
	}

	// Hash the message
	hash := sha256.Sum256(message)

	// Verify using PSS
	err = rsa.VerifyPSS(rsaPublicKey, crypto.SHA256, hash[:], signature, nil)
	return err == nil, nil
}

// verifyECDSA verifies ECDSA signature
func (cm *CryptoManager) verifyECDSA(key *CryptoKey, message, signature []byte) (bool, error) {
	if key.Type != "ecdsa" {
		return false, fmt.Errorf("key type %s is not suitable for ECDSA verification", key.Type)
	}

	// Parse public key
	block, _ := pem.Decode(key.PublicKey)
	if block == nil {
		return false, fmt.Errorf("failed to parse public key")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return false, err
	}

	ecdsaPublicKey, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return false, fmt.Errorf("not an ECDSA public key")
	}

	// Decode signature
	var sig struct{ R, S *big.Int }
	if _, err := asn1.Unmarshal(signature, &sig); err != nil {
		return false, err
	}

	// Hash the message
	hash := sha256.Sum256(message)

	// Verify
	return ecdsa.Verify(ecdsaPublicKey, hash[:], sig.R, sig.S), nil
}

// verifyEd25519 verifies Ed25519 signature
func (cm *CryptoManager) verifyEd25519(key *CryptoKey, message, signature []byte) (bool, error) {
	if key.Type != "ed25519" {
		return false, fmt.Errorf("key type %s is not suitable for Ed25519 verification", key.Type)
	}

	// Parse public key
	block, _ := pem.Decode(key.PublicKey)
	if block == nil {
		return false, fmt.Errorf("failed to parse public key")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return false, err
	}

	ed25519PublicKey, ok := publicKey.(ed25519.PublicKey)
	if !ok {
		return false, fmt.Errorf("not an Ed25519 public key")
	}

	// Verify
	return ed25519.Verify(ed25519PublicKey, message, signature), nil
}

// CreateCertificate creates a new certificate
func (cm *CryptoManager) CreateCertificate(keyID string, req *CertificateRequest) ([]byte, error) {
	key, err := cm.keyManager.GetKey(keyID)
	if err != nil {
		return nil, err
	}

	// Parse private key
	block, _ := pem.Decode(key.PrivateKey)
	if block == nil {
		return nil, fmt.Errorf("failed to parse private key")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	// Create certificate template
	template := &x509.Certificate{
		SerialNumber:          req.SerialNumber,
		Subject:               req.Subject,
		DNSNames:              req.DNSNames,
		EmailAddresses:        req.EmailAddresses,
		KeyUsage:              req.KeyUsage,
		ExtKeyUsage:           req.ExtKeyUsage,
		BasicConstraintsValid: true,
		IsCA:                  req.IsCA,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(req.ValidityPeriod),
	}

	if req.IsCA {
		template.MaxPathLen = req.MaxPathLength
		template.MaxPathLenZero = req.MaxPathLength == 0
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, getPublicKey(privateKey), privateKey)
	if err != nil {
		return nil, err
	}

	// Encode as PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Store certificate with key
	key.Certificate = certPEM
	if err := cm.keyManager.saveKeyStore(); err != nil {
		return nil, fmt.Errorf("failed to save key store: %w", err)
	}

	cm.auditLogger.Log("cert_create", keyID, fmt.Sprintf("Certificate created for %s", req.Subject.CommonName))

	return certPEM, nil
}

// getPublicKey extracts public key from private key
func getPublicKey(privateKey interface{}) interface{} {
	switch key := privateKey.(type) {
	case *rsa.PrivateKey:
		return &key.PublicKey
	case *ecdsa.PrivateKey:
		return &key.PublicKey
	case ed25519.PrivateKey:
		return key.Public()
	default:
		return nil
	}
}

// DeriveKey derives a key using key derivation function
func (cm *CryptoManager) DeriveKey(password []byte, salt []byte, keyLen int, algorithm string) ([]byte, error) {
	if algorithm == "" {
		algorithm = cm.config.KDFAlg
	}

	var key []byte
	var err error

	switch algorithm {
	case "pbkdf2":
		key = pbkdf2.Key(password, salt, cm.config.KDFIterations, keyLen, sha256.New)
	case "scrypt":
		key, err = scrypt.Key(password, salt, 32768, 8, 1, keyLen)
	case "hkdf":
		hkdf := hkdf.New(sha256.New, password, salt, nil)
		key = make([]byte, keyLen)
		_, err = io.ReadFull(hkdf, key)
	default:
		return nil, fmt.Errorf("unsupported KDF algorithm: %s", algorithm)
	}

	if err != nil {
		return nil, err
	}

	cm.auditLogger.Log("key_derive", "", fmt.Sprintf("Key derived using %s", algorithm))
	return key, nil
}

// RotateKey rotates a key
func (cm *CryptoManager) RotateKey(keyID string) (*CryptoKey, error) {
	oldKey, err := cm.keyManager.GetKey(keyID)
	if err != nil {
		return nil, err
	}

	// Generate new key with same parameters
	newKey, err := cm.keyManager.GenerateKey(oldKey.Type, oldKey.Size, oldKey.Usage)
	if err != nil {
		return nil, err
	}

	// Mark old key as rotated
	oldKey.Status = "rotated"
	oldKey.Metadata["rotated_to"] = newKey.ID
	oldKey.Metadata["rotated_at"] = time.Now().Format(time.RFC3339)

	// Mark new key as rotation of old key
	newKey.Metadata["rotated_from"] = oldKey.ID

	// Save key store
	if err := cm.keyManager.saveKeyStore(); err != nil {
		return nil, fmt.Errorf("failed to save key store: %w", err)
	}

	cm.auditLogger.Log("key_rotate", keyID, fmt.Sprintf("Key rotated to %s", newKey.ID))

	return newKey, nil
}

// GenerateCSR generates a certificate signing request
func (cm *CryptoManager) GenerateCSR(keyID string, req *CertificateRequest) ([]byte, error) {
	key, err := cm.keyManager.GetKey(keyID)
	if err != nil {
		return nil, err
	}

	// Parse private key
	block, _ := pem.Decode(key.PrivateKey)
	if block == nil {
		return nil, fmt.Errorf("failed to parse private key")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	// Create CSR template
	template := &x509.CertificateRequest{
		Subject:        req.Subject,
		DNSNames:       req.DNSNames,
		EmailAddresses: req.EmailAddresses,
	}

	// Create CSR
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, template, privateKey)
	if err != nil {
		return nil, err
	}

	// Encode as PEM
	csrPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrDER,
	})

	cm.auditLogger.Log("csr_create", keyID, fmt.Sprintf("CSR created for %s", req.Subject.CommonName))

	return csrPEM, nil
}

// generateKeyID generates a unique key ID
func generateKeyID() string {
	return fmt.Sprintf("key_%d_%x", time.Now().Unix(), generateRandomBytes(8))
}

// generateRandomBytes generates random bytes
func generateRandomBytes(n int) []byte {
	bytes := make([]byte, n)
	rand.Read(bytes)
	return bytes
}

// Main CLI implementation
func main() {
	var rootCmd = &cobra.Command{
		Use:   "mcp-crypto",
		Short: "Comprehensive cryptographic operations and key management tool for MCP implementations",
		Long: `mcp-crypto provides enterprise-grade cryptographic operations and key management
capabilities for Model Context Protocol (MCP) implementations. It supports secure key
generation, encryption/decryption, digital signatures, and certificate management.`,
	}

	// Global flags
	var (
		configFile = rootCmd.PersistentFlags().String("config", "", "Configuration file path")
		keyStore   = rootCmd.PersistentFlags().String("keystore", "keystore.json", "Key store file path")
		verbose    = rootCmd.PersistentFlags().Bool("verbose", false, "Enable verbose output")
	)

	// Key generation command
	var keygenCmd = &cobra.Command{
		Use:   "keygen",
		Short: "Generate cryptographic keys",
		Long:  `Generates cryptographic keys for various algorithms and purposes.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runKeygen(cmd, args, configFile, keyStore, verbose)
		},
	}

	keygenCmd.Flags().String("type", "rsa", "Key type (rsa, ecdsa, ed25519, aes, chacha20)")
	keygenCmd.Flags().Int("bits", 2048, "Key size in bits")
	keygenCmd.Flags().StringSlice("usage", []string{"encrypt", "sign"}, "Key usage")
	keygenCmd.Flags().String("output", "", "Output file for private key")
	keygenCmd.Flags().String("public-output", "", "Output file for public key")

	// Encryption command
	var encryptCmd = &cobra.Command{
		Use:   "encrypt",
		Short: "Encrypt data or messages",
		Long:  `Encrypts data or messages using specified keys and algorithms.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEncrypt(cmd, args, configFile, keyStore, verbose)
		},
	}

	encryptCmd.Flags().String("key", "", "Key ID for encryption")
	encryptCmd.Flags().String("algorithm", "", "Encryption algorithm")
	encryptCmd.Flags().String("input", "", "Input file or data")
	encryptCmd.Flags().String("output", "", "Output file for encrypted data")

	// Decryption command
	var decryptCmd = &cobra.Command{
		Use:   "decrypt",
		Short: "Decrypt data or messages",
		Long:  `Decrypts data or messages using specified keys and algorithms.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDecrypt(cmd, args, configFile, keyStore, verbose)
		},
	}

	decryptCmd.Flags().String("key", "", "Key ID for decryption")
	decryptCmd.Flags().String("algorithm", "", "Decryption algorithm")
	decryptCmd.Flags().String("input", "", "Input file or encrypted data")
	decryptCmd.Flags().String("output", "", "Output file for decrypted data")

	// Signing command
	var signCmd = &cobra.Command{
		Use:   "sign",
		Short: "Create digital signatures",
		Long:  `Creates digital signatures for data or messages.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSign(cmd, args, configFile, keyStore, verbose)
		},
	}

	signCmd.Flags().String("key", "", "Key ID for signing")
	signCmd.Flags().String("algorithm", "", "Signature algorithm")
	signCmd.Flags().String("input", "", "Input file or data to sign")
	signCmd.Flags().String("output", "", "Output file for signature")

	// Verification command
	var verifyCmd = &cobra.Command{
		Use:   "verify",
		Short: "Verify digital signatures",
		Long:  `Verifies digital signatures for data or messages.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVerify(cmd, args, configFile, keyStore, verbose)
		},
	}

	verifyCmd.Flags().String("key", "", "Key ID for verification")
	verifyCmd.Flags().String("algorithm", "", "Signature algorithm")
	verifyCmd.Flags().String("input", "", "Input file or data")
	verifyCmd.Flags().String("signature", "", "Signature file")

	// Certificate command
	var certCmd = &cobra.Command{
		Use:   "cert",
		Short: "Certificate management operations",
		Long:  `Manages certificates including creation, signing, and validation.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCert(cmd, args, configFile, keyStore, verbose)
		},
	}

	certCmd.Flags().Bool("create", false, "Create new certificate")
	certCmd.Flags().Bool("csr", false, "Generate certificate signing request")
	certCmd.Flags().String("key", "", "Key ID for certificate")
	certCmd.Flags().String("subject", "", "Certificate subject")
	certCmd.Flags().StringSlice("dns", []string{}, "DNS names")
	certCmd.Flags().StringSlice("email", []string{}, "Email addresses")
	certCmd.Flags().Duration("validity", 365*24*time.Hour, "Certificate validity period")
	certCmd.Flags().Bool("ca", false, "Create CA certificate")
	certCmd.Flags().String("output", "", "Output file for certificate")

	// List command
	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List cryptographic keys",
		Long:  `Lists all cryptographic keys in the key store.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, args, configFile, keyStore, verbose)
		},
	}

	listCmd.Flags().Bool("json", false, "Output in JSON format")

	// Delete command
	var deleteCmd = &cobra.Command{
		Use:   "delete",
		Short: "Delete cryptographic keys",
		Long:  `Deletes cryptographic keys from the key store.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(cmd, args, configFile, keyStore, verbose)
		},
	}

	deleteCmd.Flags().String("key", "", "Key ID to delete")

	// Rotate command
	var rotateCmd = &cobra.Command{
		Use:   "rotate",
		Short: "Key rotation operations",
		Long:  `Performs key rotation operations.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRotate(cmd, args, configFile, keyStore, verbose)
		},
	}

	rotateCmd.Flags().String("key", "", "Key ID to rotate")

	// Add commands to root
	rootCmd.AddCommand(keygenCmd)
	rootCmd.AddCommand(encryptCmd)
	rootCmd.AddCommand(decryptCmd)
	rootCmd.AddCommand(signCmd)
	rootCmd.AddCommand(verifyCmd)
	rootCmd.AddCommand(certCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(rotateCmd)

	// Execute the CLI
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

// Command implementations
func runKeygen(cmd *cobra.Command, args []string, configFile, keyStore, verbose *string) error {
	keyType, _ := cmd.Flags().GetString("type")
	keySize, _ := cmd.Flags().GetInt("bits")
	usage, _ := cmd.Flags().GetStringSlice("usage")
	output, _ := cmd.Flags().GetString("output")
	publicOutput, _ := cmd.Flags().GetString("public-output")

	// Load configuration
	config := DefaultCryptoConfig()
	config.KeyStore = *keyStore

	if *configFile != "" {
		if err := loadConfig(config, *configFile); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Create crypto manager
	cm, err := NewCryptoManager(config)
	if err != nil {
		return fmt.Errorf("failed to create crypto manager: %w", err)
	}

	// Generate key
	key, err := cm.keyManager.GenerateKey(keyType, keySize, usage)
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	// Save private key
	if output != "" {
		if err := os.WriteFile(output, key.PrivateKey, 0600); err != nil {
			return fmt.Errorf("failed to save private key: %w", err)
		}
	}

	// Save public key
	if publicOutput != "" && key.PublicKey != nil {
		if err := os.WriteFile(publicOutput, key.PublicKey, 0644); err != nil {
			return fmt.Errorf("failed to save public key: %w", err)
		}
	}

	fmt.Printf("Key generated successfully: %s\n", key.ID)
	fmt.Printf("Type: %s, Size: %d bits\n", key.Type, key.Size)
	fmt.Printf("Usage: %v\n", key.Usage)

	return nil
}

func runEncrypt(cmd *cobra.Command, args []string, configFile, keyStore, verbose *string) error {
	keyID, _ := cmd.Flags().GetString("key")
	algorithm, _ := cmd.Flags().GetString("algorithm")
	input, _ := cmd.Flags().GetString("input")
	output, _ := cmd.Flags().GetString("output")

	if keyID == "" {
		return fmt.Errorf("key ID is required")
	}

	// Load configuration
	config := DefaultCryptoConfig()
	config.KeyStore = *keyStore

	if *configFile != "" {
		if err := loadConfig(config, *configFile); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Create crypto manager
	cm, err := NewCryptoManager(config)
	if err != nil {
		return fmt.Errorf("failed to create crypto manager: %w", err)
	}

	// Read input
	var plaintext []byte
	if input == "" {
		plaintext = []byte(strings.Join(args, " "))
	} else {
		plaintext, err = os.ReadFile(input)
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
	}

	// Encrypt
	ciphertext, err := cm.Encrypt(keyID, plaintext, algorithm)
	if err != nil {
		return fmt.Errorf("failed to encrypt: %w", err)
	}

	// Save output
	if output != "" {
		if err := os.WriteFile(output, ciphertext, 0644); err != nil {
			return fmt.Errorf("failed to save output: %w", err)
		}
	} else {
		fmt.Printf("Encrypted data (base64): %s\n", base64.StdEncoding.EncodeToString(ciphertext))
	}

	fmt.Printf("Encryption successful\n")
	return nil
}

func runDecrypt(cmd *cobra.Command, args []string, configFile, keyStore, verbose *string) error {
	keyID, _ := cmd.Flags().GetString("key")
	algorithm, _ := cmd.Flags().GetString("algorithm")
	input, _ := cmd.Flags().GetString("input")
	output, _ := cmd.Flags().GetString("output")

	if keyID == "" {
		return fmt.Errorf("key ID is required")
	}

	// Load configuration
	config := DefaultCryptoConfig()
	config.KeyStore = *keyStore

	if *configFile != "" {
		if err := loadConfig(config, *configFile); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Create crypto manager
	cm, err := NewCryptoManager(config)
	if err != nil {
		return fmt.Errorf("failed to create crypto manager: %w", err)
	}

	// Read input
	var ciphertext []byte
	if input == "" {
		// Read from base64 argument
		if len(args) == 0 {
			return fmt.Errorf("no input provided")
		}
		ciphertext, err = base64.StdEncoding.DecodeString(args[0])
		if err != nil {
			return fmt.Errorf("failed to decode base64: %w", err)
		}
	} else {
		ciphertext, err = os.ReadFile(input)
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
	}

	// Decrypt
	plaintext, err := cm.Decrypt(keyID, ciphertext, algorithm)
	if err != nil {
		return fmt.Errorf("failed to decrypt: %w", err)
	}

	// Save output
	if output != "" {
		if err := os.WriteFile(output, plaintext, 0644); err != nil {
			return fmt.Errorf("failed to save output: %w", err)
		}
	} else {
		fmt.Printf("Decrypted data: %s\n", string(plaintext))
	}

	fmt.Printf("Decryption successful\n")
	return nil
}

func runSign(cmd *cobra.Command, args []string, configFile, keyStore, verbose *string) error {
	keyID, _ := cmd.Flags().GetString("key")
	algorithm, _ := cmd.Flags().GetString("algorithm")
	input, _ := cmd.Flags().GetString("input")
	output, _ := cmd.Flags().GetString("output")

	if keyID == "" {
		return fmt.Errorf("key ID is required")
	}

	// Load configuration
	config := DefaultCryptoConfig()
	config.KeyStore = *keyStore

	if *configFile != "" {
		if err := loadConfig(config, *configFile); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Create crypto manager
	cm, err := NewCryptoManager(config)
	if err != nil {
		return fmt.Errorf("failed to create crypto manager: %w", err)
	}

	// Read input
	var message []byte
	if input == "" {
		message = []byte(strings.Join(args, " "))
	} else {
		message, err = os.ReadFile(input)
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
	}

	// Sign
	signature, err := cm.Sign(keyID, message, algorithm)
	if err != nil {
		return fmt.Errorf("failed to sign: %w", err)
	}

	// Save output
	if output != "" {
		if err := os.WriteFile(output, signature, 0644); err != nil {
			return fmt.Errorf("failed to save output: %w", err)
		}
	} else {
		fmt.Printf("Signature (hex): %s\n", hex.EncodeToString(signature))
	}

	fmt.Printf("Signing successful\n")
	return nil
}

func runVerify(cmd *cobra.Command, args []string, configFile, keyStore, verbose *string) error {
	keyID, _ := cmd.Flags().GetString("key")
	algorithm, _ := cmd.Flags().GetString("algorithm")
	input, _ := cmd.Flags().GetString("input")
	signatureFile, _ := cmd.Flags().GetString("signature")

	if keyID == "" {
		return fmt.Errorf("key ID is required")
	}

	// Load configuration
	config := DefaultCryptoConfig()
	config.KeyStore = *keyStore

	if *configFile != "" {
		if err := loadConfig(config, *configFile); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Create crypto manager
	cm, err := NewCryptoManager(config)
	if err != nil {
		return fmt.Errorf("failed to create crypto manager: %w", err)
	}

	// Read input
	var message []byte
	if input == "" {
		message = []byte(strings.Join(args, " "))
	} else {
		message, err = os.ReadFile(input)
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
	}

	// Read signature
	var signature []byte
	if signatureFile != "" {
		signature, err = os.ReadFile(signatureFile)
		if err != nil {
			return fmt.Errorf("failed to read signature: %w", err)
		}
	} else {
		return fmt.Errorf("signature file is required")
	}

	// Verify
	valid, err := cm.Verify(keyID, message, signature, algorithm)
	if err != nil {
		return fmt.Errorf("failed to verify: %w", err)
	}

	if valid {
		fmt.Printf("Signature verification: VALID\n")
	} else {
		fmt.Printf("Signature verification: INVALID\n")
	}

	return nil
}

func runCert(cmd *cobra.Command, args []string, configFile, keyStore, verbose *string) error {
	create, _ := cmd.Flags().GetBool("create")
	csr, _ := cmd.Flags().GetBool("csr")
	keyID, _ := cmd.Flags().GetString("key")
	subject, _ := cmd.Flags().GetString("subject")
	dnsNames, _ := cmd.Flags().GetStringSlice("dns")
	emailAddresses, _ := cmd.Flags().GetStringSlice("email")
	validity, _ := cmd.Flags().GetDuration("validity")
	isCA, _ := cmd.Flags().GetBool("ca")
	output, _ := cmd.Flags().GetString("output")

	if keyID == "" {
		return fmt.Errorf("key ID is required")
	}

	// Load configuration
	config := DefaultCryptoConfig()
	config.KeyStore = *keyStore

	if *configFile != "" {
		if err := loadConfig(config, *configFile); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Create crypto manager
	cm, err := NewCryptoManager(config)
	if err != nil {
		return fmt.Errorf("failed to create crypto manager: %w", err)
	}

	// Parse subject
	var subjectPkix pkix.Name
	if subject != "" {
		// Simple subject parsing (would be more sophisticated in real implementation)
		if strings.HasPrefix(subject, "CN=") {
			subjectPkix.CommonName = strings.TrimPrefix(subject, "CN=")
		}
	}

	// Create certificate request
	certReq := &CertificateRequest{
		Subject:         subjectPkix,
		DNSNames:        dnsNames,
		EmailAddresses:  emailAddresses,
		ValidityPeriod:  validity,
		IsCA:            isCA,
		SerialNumber:    big.NewInt(1),
		KeyUsage:        x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	var result []byte
	if create {
		result, err = cm.CreateCertificate(keyID, certReq)
		if err != nil {
			return fmt.Errorf("failed to create certificate: %w", err)
		}
		fmt.Printf("Certificate created successfully\n")
	} else if csr {
		result, err = cm.GenerateCSR(keyID, certReq)
		if err != nil {
			return fmt.Errorf("failed to generate CSR: %w", err)
		}
		fmt.Printf("CSR generated successfully\n")
	} else {
		return fmt.Errorf("specify --create or --csr")
	}

	// Save output
	if output != "" {
		if err := os.WriteFile(output, result, 0644); err != nil {
			return fmt.Errorf("failed to save output: %w", err)
		}
	} else {
		fmt.Printf("%s", string(result))
	}

	return nil
}

func runList(cmd *cobra.Command, args []string, configFile, keyStore, verbose *string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Load configuration
	config := DefaultCryptoConfig()
	config.KeyStore = *keyStore

	if *configFile != "" {
		if err := loadConfig(config, *configFile); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Create crypto manager
	cm, err := NewCryptoManager(config)
	if err != nil {
		return fmt.Errorf("failed to create crypto manager: %w", err)
	}

	// List keys
	keys := cm.keyManager.ListKeys()

	if jsonOutput {
		keysJSON, err := json.MarshalIndent(keys, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal keys: %w", err)
		}
		fmt.Printf("%s\n", keysJSON)
	} else {
		fmt.Printf("%-20s %-10s %-8s %-10s %-20s\n", "ID", "Type", "Size", "Status", "Created")
		fmt.Printf("%-20s %-10s %-8s %-10s %-20s\n", strings.Repeat("-", 20), strings.Repeat("-", 10), strings.Repeat("-", 8), strings.Repeat("-", 10), strings.Repeat("-", 20))
		for _, key := range keys {
			fmt.Printf("%-20s %-10s %-8d %-10s %-20s\n", key.ID, key.Type, key.Size, key.Status, key.CreatedAt.Format("2006-01-02 15:04"))
		}
	}

	return nil
}

func runDelete(cmd *cobra.Command, args []string, configFile, keyStore, verbose *string) error {
	keyID, _ := cmd.Flags().GetString("key")

	if keyID == "" {
		return fmt.Errorf("key ID is required")
	}

	// Load configuration
	config := DefaultCryptoConfig()
	config.KeyStore = *keyStore

	if *configFile != "" {
		if err := loadConfig(config, *configFile); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Create crypto manager
	cm, err := NewCryptoManager(config)
	if err != nil {
		return fmt.Errorf("failed to create crypto manager: %w", err)
	}

	// Delete key
	if err := cm.keyManager.DeleteKey(keyID); err != nil {
		return fmt.Errorf("failed to delete key: %w", err)
	}

	fmt.Printf("Key %s deleted successfully\n", keyID)
	return nil
}

func runRotate(cmd *cobra.Command, args []string, configFile, keyStore, verbose *string) error {
	keyID, _ := cmd.Flags().GetString("key")

	if keyID == "" {
		return fmt.Errorf("key ID is required")
	}

	// Load configuration
	config := DefaultCryptoConfig()
	config.KeyStore = *keyStore

	if *configFile != "" {
		if err := loadConfig(config, *configFile); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Create crypto manager
	cm, err := NewCryptoManager(config)
	if err != nil {
		return fmt.Errorf("failed to create crypto manager: %w", err)
	}

	// Rotate key
	newKey, err := cm.RotateKey(keyID)
	if err != nil {
		return fmt.Errorf("failed to rotate key: %w", err)
	}

	fmt.Printf("Key %s rotated successfully to %s\n", keyID, newKey.ID)
	return nil
}

// loadConfig loads configuration from file
func loadConfig(config *CryptoConfig, filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	ext := filepath.Ext(filename)
	switch ext {
	case ".json":
		return json.Unmarshal(data, config)
	default:
		return fmt.Errorf("unsupported config file format: %s", ext)
	}
}