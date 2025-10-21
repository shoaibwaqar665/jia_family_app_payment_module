package auth

import (
	"context"
	"fmt"
)

// HybridValidator combines JWT and SPIFFE validation
type HybridValidator struct {
	jwtValidator    Validator
	spiffeValidator *SPIFFEValidator
	spiffeEnabled   bool
}

// NewHybridValidator creates a new hybrid validator
func NewHybridValidator(jwtValidator Validator, spiffeValidator *SPIFFEValidator, spiffeEnabled bool) *HybridValidator {
	return &HybridValidator{
		jwtValidator:    jwtValidator,
		spiffeValidator: spiffeValidator,
		spiffeEnabled:   spiffeEnabled,
	}
}

// Validate implements the Validator interface
// It tries SPIFFE validation first if enabled, then falls back to JWT validation
func (h *HybridValidator) Validate(ctx context.Context, token string) (string, error) {
	// If SPIFFE is enabled, try SPIFFE validation first
	if h.spiffeEnabled && h.spiffeValidator != nil {
		spiffeID, err := h.spiffeValidator.ValidatePeerFromContext(ctx)
		if err == nil && spiffeID != "" {
			// Extract service name from SPIFFE ID
			serviceName := ExtractServiceName(spiffeID)
			if serviceName != "" {
				return serviceName, nil
			}
			// If we can't extract service name, use the full SPIFFE ID
			return spiffeID, nil
		}
		// SPIFFE validation failed - log the error but continue to JWT fallback
		// This is expected behavior when SPIFFE is not available (e.g., non-mTLS connections)
	}

	// Fall back to JWT validation if available
	if h.jwtValidator != nil {
		return h.jwtValidator.Validate(ctx, token)
	}

	// If neither SPIFFE nor JWT validation is available, return an error
	return "", fmt.Errorf("no valid authentication method available")
}

// Close closes the hybrid validator
func (h *HybridValidator) Close() error {
	var err error

	if h.jwtValidator != nil {
		if closer, ok := h.jwtValidator.(interface{ Close() error }); ok {
			if closeErr := closer.Close(); closeErr != nil {
				err = closeErr
			}
		}
	}

	if h.spiffeValidator != nil {
		if closeErr := h.spiffeValidator.Close(); closeErr != nil {
			err = closeErr
		}
	}

	return err
}
