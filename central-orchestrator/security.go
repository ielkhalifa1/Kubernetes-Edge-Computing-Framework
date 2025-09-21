package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// Certificate validity period
	CertValidityPeriod = 365 * 24 * time.Hour // 1 year
	
	// RSA key size
	RSAKeySize = 2048
)

// AuthMiddleware provides authentication middleware
func (sm *SecurityManager) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip auth for health check
		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}

		// For now, implement basic token authentication
		// In production, this should use proper mTLS client certificate validation
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Extract Bearer token
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Bearer token required"})
			c.Abort()
			return
		}

		token := strings.TrimPrefix(authHeader, bearerPrefix)
		
		// For demo purposes, accept any non-empty token
		// In production, validate JWT tokens or client certificates
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Set user context (in production, extract from validated token)
		c.Set("user", "edge-node")
		c.Set("role", "node")
		
		c.Next()
	}
}

// IssueCertificate issues a new certificate for a node
func (co *CentralOrchestrator) IssueCertificate(c *gin.Context) {
	var req CertificateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cert, err := co.SecurityManager.GenerateCertificate(req.NodeID, req.CommonName, req.DNSNames, req.IPAddresses)
	if err != nil {
		co.Logger.Errorf("Failed to generate certificate: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate certificate"})
		return
	}

	co.Logger.Infof("Certificate issued for node %s", req.NodeID)
	
	c.JSON(http.StatusCreated, gin.H{
		"certificate_id": cert.ID,
		"certificate":    string(cert.Certificate),
		"issued_at":     cert.IssuedAt,
		"expires_at":    cert.ExpiresAt,
	})
}

// RevokeCertificate revokes an existing certificate
func (co *CentralOrchestrator) RevokeCertificate(c *gin.Context) {
	var req CertificateRevocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := co.SecurityManager.RevokeCertificate(req.CertificateID)
	if err != nil {
		co.Logger.Errorf("Failed to revoke certificate: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	co.Logger.Infof("Certificate %s revoked", req.CertificateID)
	
	c.JSON(http.StatusOK, gin.H{"message": "Certificate revoked successfully"})
}

// GenerateCertificate generates a new TLS certificate
func (sm *SecurityManager) GenerateCertificate(nodeID, commonName string, dnsNames, ipAddresses []string) (*Certificate, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, RSAKeySize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %v", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Kubernetes Edge Framework"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
			CommonName:    commonName,
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(CertValidityPeriod),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		IPAddresses:  []x509.IPAddress{},
		DNSNames:     dnsNames,
	}

	// Add IP addresses if provided
	for _, ipStr := range ipAddresses {
		if ip := x509.ParseIP(ipStr); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		}
	}

	// For demo purposes, use self-signed certificates
	// In production, use a proper CA
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %v", err)
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Encode private key to PEM
	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %v", err)
	}

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyDER,
	})

	// Create certificate record
	certID := generateID()
	cert := &Certificate{
		ID:          certID,
		NodeID:      nodeID,
		Certificate: certPEM,
		PrivateKey:  privateKeyPEM,
		IssuedAt:    template.NotBefore,
		ExpiresAt:   template.NotAfter,
	}

	// Store certificate
	sm.certificates[certID] = cert

	return cert, nil
}

// RevokeCertificate revokes a certificate
func (sm *SecurityManager) RevokeCertificate(certificateID string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if _, exists := sm.certificates[certificateID]; !exists {
		return fmt.Errorf("certificate not found")
	}

	// For now, just remove from storage
	// In production, maintain a certificate revocation list (CRL)
	delete(sm.certificates, certificateID)
	
	return nil
}

// ValidateClientCertificate validates a client certificate
func (sm *SecurityManager) ValidateClientCertificate(certPEM []byte) error {
	// Parse certificate
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return fmt.Errorf("failed to parse certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %v", err)
	}

	// Check expiration
	now := time.Now()
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return fmt.Errorf("certificate is not valid at current time")
	}

	// Additional validation logic can be added here
	// For example, checking against a certificate revocation list

	return nil
}

// GetTLSConfig returns TLS configuration for secure communication
func (sm *SecurityManager) GetTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
		ClientAuth: tls.RequireAndVerifyClientCert,
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			// Custom certificate verification logic
			if len(rawCerts) == 0 {
				return fmt.Errorf("no client certificate provided")
			}
			
			// In production, implement proper certificate chain validation
			return nil
		},
	}
}

// Certificate request structures
type CertificateRequest struct {
	NodeID      string   `json:"node_id" binding:"required"`
	CommonName  string   `json:"common_name" binding:"required"`
	DNSNames    []string `json:"dns_names"`
	IPAddresses []string `json:"ip_addresses"`
}

type CertificateRevocationRequest struct {
	CertificateID string `json:"certificate_id" binding:"required"`
}
