package auth

import (
	"context"
	"fmt"
	"strings"
)

// Validate validates a token and returns the user ID
// This is a stub implementation that treats any non-empty token as valid
// and extracts a fake spiff_id_* from a claim placeholder
func Validate(ctx context.Context, token string) (userID string, err error) {
	if token == "" {
		return "", fmt.Errorf("empty token")
	}
	
	// For now, treat any non-empty token as valid
	// In a real implementation, this would validate JWT, check expiration, etc.
	
	// Extract fake user ID from token (placeholder for real JWT claims)
	// Format: "Bearer spiff_id_12345" or just "spiff_id_12345"
	token = strings.TrimPrefix(token, "Bearer ")
	
	if strings.HasPrefix(token, "spiff_id_") {
		return token, nil
	}
	
	// If no spiff_id prefix, generate a fake one based on token
	// This is just for demonstration - real implementation would parse JWT claims
	if len(token) > 8 {
		fakeUserID := fmt.Sprintf("spiff_id_%s", token[:8])
		return fakeUserID, nil
	}
	// For short tokens, use the entire token
	fakeUserID := fmt.Sprintf("spiff_id_%s", token)
	return fakeUserID, nil
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
