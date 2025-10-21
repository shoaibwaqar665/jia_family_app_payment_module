package auth

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTValidator validates JWT tokens
type JWTValidator struct {
	publicKey *rsa.PublicKey
}

// NewJWTValidator creates a new JWT validator from PEM string
func NewJWTValidator(publicKeyPEM string) (*JWTValidator, error) {
	if publicKeyPEM == "" {
		return nil, fmt.Errorf("public key PEM is required")
	}

	// Parse the PEM block
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block containing public key")
	}

	// Parse the public key
	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	// Type assert to RSA public key
	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not an RSA key")
	}

	return &JWTValidator{
		publicKey: rsaPublicKey,
	}, nil
}

// NewJWTValidatorFromFile creates a new JWT validator from file path (for backward compatibility)
func NewJWTValidatorFromFile(publicKeyPath string) (*JWTValidator, error) {
	// Read the public key file
	publicKeyPEM, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}

	return NewJWTValidator(string(publicKeyPEM))
}

// Validate validates a JWT token and returns the user ID
func (v *JWTValidator) Validate(ctx context.Context, token string) (userID string, err error) {
	if token == "" {
		return "", fmt.Errorf("empty token")
	}

	// Remove Bearer prefix if present
	token = strings.TrimPrefix(token, "Bearer ")
	token = strings.TrimSpace(token)

	// Validate token length (basic sanity check)
	if len(token) < 10 {
		return "", fmt.Errorf("token too short")
	}

	// Parse and validate the JWT token
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method - only allow RSA
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Validate algorithm in header
		if token.Header["alg"] != "RS256" {
			return nil, fmt.Errorf("unsupported algorithm: %v", token.Header["alg"])
		}

		return v.publicKey, nil
	}, jwt.WithValidMethods([]string{"RS256"}))

	if err != nil {
		return "", fmt.Errorf("failed to parse JWT token: %w", err)
	}

	// Check if token is valid
	if !parsedToken.Valid {
		return "", fmt.Errorf("invalid token")
	}

	// Extract claims
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("failed to extract claims from token")
	}

	// Validate required claims
	if err := v.validateClaims(claims); err != nil {
		return "", fmt.Errorf("claim validation failed: %w", err)
	}

	// Extract user ID from claims
	userID, ok = claims["sub"].(string)
	if !ok {
		userID, ok = claims["user_id"].(string)
		if !ok {
			userID, ok = claims["userId"].(string)
			if !ok {
				return "", fmt.Errorf("user ID not found in token claims")
			}
		}
	}

	// Validate user ID is not empty
	if strings.TrimSpace(userID) == "" {
		return "", fmt.Errorf("user ID is empty")
	}

	return userID, nil
}

// validateClaims validates JWT claims for security
func (v *JWTValidator) validateClaims(claims jwt.MapClaims) error {
	// Check expiration
	if exp, ok := claims["exp"].(float64); ok {
		expTime := time.Unix(int64(exp), 0)
		if time.Now().After(expTime) {
			return fmt.Errorf("token has expired at %v", expTime)
		}
	} else {
		return fmt.Errorf("expiration claim (exp) is missing")
	}

	// Check issued at time (iat)
	if iat, ok := claims["iat"].(float64); ok {
		iatTime := time.Unix(int64(iat), 0)
		// Reject tokens issued in the future (with 5 minute tolerance for clock skew)
		if time.Now().Before(iatTime.Add(-5 * time.Minute)) {
			return fmt.Errorf("token issued in the future: %v", iatTime)
		}
	}

	// Check not before time (nbf) if present
	if nbf, ok := claims["nbf"].(float64); ok {
		nbfTime := time.Unix(int64(nbf), 0)
		if time.Now().Before(nbfTime) {
			return fmt.Errorf("token not valid until %v", nbfTime)
		}
	}

	// Validate issuer if present
	if iss, ok := claims["iss"].(string); ok {
		if strings.TrimSpace(iss) == "" {
			return fmt.Errorf("issuer claim (iss) is empty")
		}
	}

	// Validate audience if present
	if aud, ok := claims["aud"].(string); ok {
		if strings.TrimSpace(aud) == "" {
			return fmt.Errorf("audience claim (aud) is empty")
		}
	} else if audSlice, ok := claims["aud"].([]interface{}); ok {
		if len(audSlice) == 0 {
			return fmt.Errorf("audience claim (aud) is empty")
		}
	}

	return nil
}
