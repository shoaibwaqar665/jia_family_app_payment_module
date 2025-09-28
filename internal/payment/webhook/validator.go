package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Validator handles webhook signature validation
type Validator struct {
	webhookSecret string
	tolerance     time.Duration
}

// NewValidator creates a new webhook validator
func NewValidator(webhookSecret string) *Validator {
	return &Validator{
		webhookSecret: webhookSecret,
		tolerance:     5 * time.Minute, // 5 minutes tolerance for timestamp
	}
}

// ValidateStripeWebhook validates a Stripe webhook signature
func (v *Validator) ValidateStripeWebhook(payload []byte, signature string) error {
	if signature == "" {
		return fmt.Errorf("missing signature")
	}

	if v.webhookSecret == "" {
		return fmt.Errorf("webhook secret not configured")
	}

	// Parse signature elements
	sigElements := strings.Split(signature, ",")
	var timestamp, providedSignature string

	for _, element := range sigElements {
		if strings.HasPrefix(element, "t=") {
			timestamp = strings.TrimPrefix(element, "t=")
		} else if strings.HasPrefix(element, "v1=") {
			providedSignature = strings.TrimPrefix(element, "v1=")
		}
	}

	if timestamp == "" || providedSignature == "" {
		return fmt.Errorf("invalid signature format")
	}

	// Validate timestamp to prevent replay attacks
	if err := v.validateTimestamp(timestamp); err != nil {
		return fmt.Errorf("timestamp validation failed: %w", err)
	}

	// Compute expected signature
	expectedSignature, err := v.computeSignature(timestamp, payload)
	if err != nil {
		return fmt.Errorf("failed to compute signature: %w", err)
	}

	// Compare signatures using constant time comparison
	if !v.constantTimeCompare(expectedSignature, providedSignature) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

// validateTimestamp checks if the timestamp is within tolerance
func (v *Validator) validateTimestamp(timestampStr string) error {
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp format: %w", err)
	}

	now := time.Now().Unix()
	diff := now - timestamp

	if diff < 0 {
		return fmt.Errorf("timestamp is in the future")
	}

	if time.Duration(diff)*time.Second > v.tolerance {
		return fmt.Errorf("timestamp too old: %d seconds", diff)
	}

	return nil
}

// computeSignature computes the HMAC-SHA256 signature
func (v *Validator) computeSignature(timestamp string, payload []byte) (string, error) {
	// Create the signed payload: timestamp + "." + payload
	signedPayload := timestamp + "." + string(payload)

	// Compute HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(v.webhookSecret))
	mac.Write([]byte(signedPayload))
	signature := mac.Sum(nil)

	return hex.EncodeToString(signature), nil
}

// constantTimeCompare performs constant time comparison to prevent timing attacks
func (v *Validator) constantTimeCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}

	result := 0
	for i := 0; i < len(a); i++ {
		result |= int(a[i]) ^ int(b[i])
	}

	return result == 0
}

// ValidatePayPalWebhook validates a PayPal webhook signature (placeholder)
func (v *Validator) ValidatePayPalWebhook(payload []byte, signature string) error {
	// TODO: Implement PayPal webhook validation
	// PayPal uses different signature validation method
	return fmt.Errorf("PayPal webhook validation not implemented")
}

// ValidateAdyenWebhook validates an Adyen webhook signature (placeholder)
func (v *Validator) ValidateAdyenWebhook(payload []byte, signature string) error {
	// TODO: Implement Adyen webhook validation
	// Adyen uses HMAC-SHA256 with different format
	return fmt.Errorf("Adyen webhook validation not implemented")
}
