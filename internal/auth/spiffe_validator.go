package auth

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

// SPIFFEValidator validates SPIFFE peer certificates
type SPIFFEValidator struct {
	trustDomain string
	caPool      *x509.CertPool
}

// NewSPIFFEValidator creates a new SPIFFE validator
func NewSPIFFEValidator(trustDomain string, caCertPEM string) (*SPIFFEValidator, error) {
	if trustDomain == "" {
		return nil, fmt.Errorf("trust domain is required")
	}

	// Create certificate pool for CA validation
	caPool := x509.NewCertPool()
	if caCertPEM != "" {
		if !caPool.AppendCertsFromPEM([]byte(caCertPEM)) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
	}

	return &SPIFFEValidator{
		trustDomain: trustDomain,
		caPool:      caPool,
	}, nil
}

// ValidatePeerCertificate validates a peer certificate against SPIFFE requirements
func (v *SPIFFEValidator) ValidatePeerCertificate(ctx context.Context, certPEM string) (string, error) {
	if certPEM == "" {
		return "", fmt.Errorf("empty certificate")
	}

	// Parse the PEM block
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return "", fmt.Errorf("failed to decode PEM block containing certificate")
	}

	// Validate PEM block type
	if block.Type != "CERTIFICATE" {
		return "", fmt.Errorf("invalid PEM block type: expected CERTIFICATE, got %s", block.Type)
	}

	// Parse the certificate
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Validate certificate is not expired (with grace period)
	now := time.Now()
	gracePeriod := 5 * time.Minute // 5 minute grace period for clock skew
	if now.After(cert.NotAfter.Add(gracePeriod)) {
		return "", fmt.Errorf("certificate has expired (not after: %v, current time: %v)", cert.NotAfter, now)
	}
	if now.Before(cert.NotBefore.Add(-gracePeriod)) {
		return "", fmt.Errorf("certificate is not yet valid (not before: %v, current time: %v)", cert.NotBefore, now)
	}

	// Validate certificate key usage
	if cert.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		return "", fmt.Errorf("certificate does not have digital signature key usage")
	}

	// Validate certificate chain if CA pool is provided
	if v.caPool != nil {
		opts := x509.VerifyOptions{
			Roots:         v.caPool,
			Intermediates: x509.NewCertPool(),
			KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}
		if _, err := cert.Verify(opts); err != nil {
			return "", fmt.Errorf("certificate verification failed: %w", err)
		}
	}

	// Extract SPIFFE ID from certificate URI SAN
	spiffeIDs := make([]string, 0)
	for _, uri := range cert.URIs {
		if strings.HasPrefix(uri.String(), "spiffe://") {
			spiffeIDs = append(spiffeIDs, uri.String())
		}
	}

	if len(spiffeIDs) == 0 {
		return "", fmt.Errorf("no SPIFFE ID found in certificate URI SAN")
	}

	// Validate that exactly one SPIFFE ID is present and it matches our trust domain
	if len(spiffeIDs) > 1 {
		return "", fmt.Errorf("multiple SPIFFE IDs found in certificate: %v", spiffeIDs)
	}

	spiffeID := spiffeIDs[0]

	// Validate trust domain
	if !strings.HasPrefix(spiffeID, "spiffe://"+v.trustDomain+"/") {
		return "", fmt.Errorf("certificate trust domain does not match expected %s, got %s", v.trustDomain, spiffeID)
	}

	// Validate SPIFFE ID format (basic validation)
	if !isValidSPIFFEID(spiffeID) {
		return "", fmt.Errorf("invalid SPIFFE ID format: %s", spiffeID)
	}

	return spiffeID, nil
}

// isValidSPIFFEID validates the format of a SPIFFE ID
func isValidSPIFFEID(spiffeID string) bool {
	// Basic format validation: spiffe://trust-domain/path
	if !strings.HasPrefix(spiffeID, "spiffe://") {
		return false
	}

	// Remove spiffe:// prefix
	path := strings.TrimPrefix(spiffeID, "spiffe://")

	// Split by first slash to get trust domain and path
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 {
		return false // Must have both trust domain and path
	}

	trustDomain := parts[0]
	spiffePath := parts[1]

	// Validate trust domain (basic validation)
	if trustDomain == "" || strings.Contains(trustDomain, "/") {
		return false
	}

	// Validate path is not empty
	if spiffePath == "" {
		return false
	}

	return true
}

// GetTLSConfig returns a TLS configuration for SPIFFE validation
func (v *SPIFFEValidator) GetTLSConfig() *tls.Config {
	return &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		RootCAs:    v.caPool,
		MinVersion: tls.VersionTLS12, // Enforce minimum TLS version
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},
		PreferServerCipherSuites: true,
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return fmt.Errorf("no client certificate provided")
			}

			// Convert raw certificate to PEM format for validation
			certPEM := pem.EncodeToMemory(&pem.Block{
				Type:  "CERTIFICATE",
				Bytes: rawCerts[0],
			})

			// Validate SPIFFE ID
			_, err := v.ValidatePeerCertificate(context.Background(), string(certPEM))
			return err
		},
	}
}

// ValidatePeerFromContext validates a peer certificate from gRPC context
func (v *SPIFFEValidator) ValidatePeerFromContext(ctx context.Context) (string, error) {
	// Get peer information from gRPC context
	p, ok := peer.FromContext(ctx)
	if !ok {
		return "", fmt.Errorf("no peer information found in context")
	}

	// Extract TLS information
	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return "", fmt.Errorf("peer does not have TLS information")
	}

	// Get the peer certificate
	if len(tlsInfo.State.PeerCertificates) == 0 {
		return "", fmt.Errorf("no peer certificate found")
	}

	// Convert certificate to PEM format
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: tlsInfo.State.PeerCertificates[0].Raw,
	})

	// Validate the certificate
	return v.ValidatePeerCertificate(ctx, string(certPEM))
}

// Validate implements the Validator interface for SPIFFE
func (v *SPIFFEValidator) Validate(ctx context.Context, token string) (string, error) {
	// For SPIFFE, we validate the peer certificate from context, not a token
	return v.ValidatePeerFromContext(ctx)
}

// Close closes the SPIFFE validator
func (v *SPIFFEValidator) Close() error {
	// No resources to close
	return nil
}

// ExtractServiceName extracts the service name from a SPIFFE ID
func ExtractServiceName(spiffeID string) string {
	if spiffeID == "" {
		return ""
	}

	// SPIFFE ID format: spiffe://trust-domain/path
	// Extract the path part
	parts := strings.Split(spiffeID, "/")
	if len(parts) >= 4 {
		path := strings.Join(parts[3:], "/") // Everything after trust-domain/

		// Handle different SPIFFE path formats:
		// /ns/{namespace}/sa/{service-account} -> return service-account
		// /workload/{service} -> return service
		// /{service} -> return service
		pathParts := strings.Split(path, "/")

		// Check for service account format: /ns/{namespace}/sa/{service-account}
		if len(pathParts) >= 4 && pathParts[0] == "ns" && pathParts[2] == "sa" {
			return pathParts[3]
		}

		// Check for workload format: /workload/{service}
		if len(pathParts) >= 2 && pathParts[0] == "workload" {
			return pathParts[1]
		}

		// For simple path format: /{service}
		if len(pathParts) >= 1 {
			return pathParts[0]
		}

		return path
	}

	return ""
}

// ValidateTrustDomain validates that a SPIFFE ID belongs to the expected trust domain
func (v *SPIFFEValidator) ValidateTrustDomain(spiffeID string) error {
	if !strings.HasPrefix(spiffeID, "spiffe://"+v.trustDomain+"/") {
		return fmt.Errorf("SPIFFE ID %s does not belong to trust domain %s", spiffeID, v.trustDomain)
	}
	return nil
}
