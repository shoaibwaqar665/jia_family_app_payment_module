package auth

import (
	"context"
	"fmt"
	"strings"
)

// Validator interface for token validation
type Validator interface {
	Validate(ctx context.Context, token string) (userID string, err error)
}

// Validate validates a token and returns the user ID
// This function should not be used in production - use specific validators instead
func Validate(ctx context.Context, token string) (userID string, err error) {
	// SECURITY: Mock validator fallback removed for production safety
	// Use NewJWTValidator or NewSPIFFEValidator directly in your application
	return "", fmt.Errorf("validator fallback removed for security - use specific validator")
}

// ExtractTokenFromAuthHeader extracts the token from an Authorization header
func ExtractTokenFromAuthHeader(authHeader string) string {
	if authHeader == "" {
		return ""
	}

	// Handle "Bearer <token>" format
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}

	// If no Bearer prefix, assume the entire header is the token
	return authHeader
}
