package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Event represents an audit event
type Event struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	UserID     string                 `json:"user_id"`
	Action     string                 `json:"action"`
	Resource   string                 `json:"resource"`
	ResourceID string                 `json:"resource_id"`
	Details    map[string]interface{} `json:"details"`
	IPAddress  string                 `json:"ip_address,omitempty"`
	UserAgent  string                 `json:"user_agent,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Result     string                 `json:"result"` // success, failure, error
	Error      string                 `json:"error,omitempty"`
}

// Logger defines the interface for audit logging
type Logger interface {
	Log(ctx context.Context, event Event) error
}

// ZapAuditLogger implements audit logging using zap
type ZapAuditLogger struct {
	logger *zap.Logger
}

// NewZapAuditLogger creates a new zap-based audit logger
func NewZapAuditLogger(logger *zap.Logger) *ZapAuditLogger {
	return &ZapAuditLogger{
		logger: logger,
	}
}

// Log logs an audit event
func (l *ZapAuditLogger) Log(ctx context.Context, event Event) error {
	// Log audit event as structured JSON
	fields := []zap.Field{
		zap.String("audit_id", event.ID),
		zap.String("audit_type", event.Type),
		zap.String("audit_action", event.Action),
		zap.String("audit_resource", event.Resource),
		zap.String("audit_resource_id", event.ResourceID),
		zap.String("audit_result", event.Result),
		zap.Time("audit_timestamp", event.Timestamp),
	}

	if event.UserID != "" {
		fields = append(fields, zap.String("audit_user_id", event.UserID))
	}

	if event.IPAddress != "" {
		fields = append(fields, zap.String("audit_ip_address", event.IPAddress))
	}

	if event.UserAgent != "" {
		fields = append(fields, zap.String("audit_user_agent", event.UserAgent))
	}

	if event.Error != "" {
		fields = append(fields, zap.String("audit_error", event.Error))
	}

	if len(event.Details) > 0 {
		detailsJSON, _ := json.Marshal(event.Details)
		fields = append(fields, zap.String("audit_details", string(detailsJSON)))
	}

	// Log at info level for successful events, error level for failures
	if event.Result == "success" {
		l.logger.Info("Audit event", fields...)
	} else {
		l.logger.Error("Audit event", fields...)
	}

	return nil
}

// Manager manages audit logging
type Manager struct {
	logger Logger
}

// NewManager creates a new audit manager
func NewManager(logger Logger) *Manager {
	return &Manager{
		logger: logger,
	}
}

// LogEntitlementCreated logs an entitlement creation event
func (m *Manager) LogEntitlementCreated(ctx context.Context, userID, entitlementID, featureCode, planID string) error {
	event := Event{
		ID:         generateEventID(),
		Type:       "entitlement",
		UserID:     userID,
		Action:     "create",
		Resource:   "entitlement",
		ResourceID: entitlementID,
		Details: map[string]interface{}{
			"feature_code": featureCode,
			"plan_id":      planID,
		},
		Timestamp: time.Now(),
		Result:    "success",
	}

	return m.logger.Log(ctx, event)
}

// LogEntitlementUpdated logs an entitlement update event
func (m *Manager) LogEntitlementUpdated(ctx context.Context, userID, entitlementID, featureCode string, changes map[string]interface{}) error {
	event := Event{
		ID:         generateEventID(),
		Type:       "entitlement",
		UserID:     userID,
		Action:     "update",
		Resource:   "entitlement",
		ResourceID: entitlementID,
		Details: map[string]interface{}{
			"feature_code": featureCode,
			"changes":      changes,
		},
		Timestamp: time.Now(),
		Result:    "success",
	}

	return m.logger.Log(ctx, event)
}

// LogEntitlementDeleted logs an entitlement deletion event
func (m *Manager) LogEntitlementDeleted(ctx context.Context, userID, entitlementID, featureCode string) error {
	event := Event{
		ID:         generateEventID(),
		Type:       "entitlement",
		UserID:     userID,
		Action:     "delete",
		Resource:   "entitlement",
		ResourceID: entitlementID,
		Details: map[string]interface{}{
			"feature_code": featureCode,
		},
		Timestamp: time.Now(),
		Result:    "success",
	}

	return m.logger.Log(ctx, event)
}

// LogPaymentCreated logs a payment creation event
func (m *Manager) LogPaymentCreated(ctx context.Context, userID, paymentID, orderID string, amount int64, currency string) error {
	event := Event{
		ID:         generateEventID(),
		Type:       "payment",
		UserID:     userID,
		Action:     "create",
		Resource:   "payment",
		ResourceID: paymentID,
		Details: map[string]interface{}{
			"order_id": orderID,
			"amount":   amount,
			"currency": currency,
		},
		Timestamp: time.Now(),
		Result:    "success",
	}

	return m.logger.Log(ctx, event)
}

// LogPaymentUpdated logs a payment update event
func (m *Manager) LogPaymentUpdated(ctx context.Context, userID, paymentID, orderID string, oldStatus, newStatus string) error {
	event := Event{
		ID:         generateEventID(),
		Type:       "payment",
		UserID:     userID,
		Action:     "update",
		Resource:   "payment",
		ResourceID: paymentID,
		Details: map[string]interface{}{
			"order_id":   orderID,
			"old_status": oldStatus,
			"new_status": newStatus,
		},
		Timestamp: time.Now(),
		Result:    "success",
	}

	return m.logger.Log(ctx, event)
}

// LogWebhookReceived logs a webhook reception event
func (m *Manager) LogWebhookReceived(ctx context.Context, eventType, eventID string, success bool, errorMsg string) error {
	result := "success"
	if !success {
		result = "failure"
	}

	event := Event{
		ID:         generateEventID(),
		Type:       "webhook",
		Action:     "receive",
		Resource:   "webhook",
		ResourceID: eventID,
		Details: map[string]interface{}{
			"event_type": eventType,
		},
		Timestamp: time.Now(),
		Result:    result,
		Error:     errorMsg,
	}

	return m.logger.Log(ctx, event)
}

// LogAccessDenied logs an access denied event
func (m *Manager) LogAccessDenied(ctx context.Context, userID, resource, resourceID, reason string) error {
	event := Event{
		ID:         generateEventID(),
		Type:       "security",
		UserID:     userID,
		Action:     "access_denied",
		Resource:   resource,
		ResourceID: resourceID,
		Details: map[string]interface{}{
			"reason": reason,
		},
		Timestamp: time.Now(),
		Result:    "failure",
	}

	return m.logger.Log(ctx, event)
}

// LogAuthentication logs an authentication event
func (m *Manager) LogAuthentication(ctx context.Context, userID string, success bool, method string) error {
	result := "success"
	if !success {
		result = "failure"
	}

	event := Event{
		ID:       generateEventID(),
		Type:     "authentication",
		UserID:   userID,
		Action:   "authenticate",
		Resource: "user",
		Details: map[string]interface{}{
			"method": method,
		},
		Timestamp: time.Now(),
		Result:    result,
	}

	return m.logger.Log(ctx, event)
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return fmt.Sprintf("audit_%d", time.Now().UnixNano())
}

// WithAuditContext adds audit information to the context
func WithAuditContext(ctx context.Context, ipAddress, userAgent string) context.Context {
	return context.WithValue(ctx, "audit_ip_address", ipAddress)
}

// GetAuditInfo extracts audit information from the context
func GetAuditInfo(ctx context.Context) (ipAddress, userAgent string) {
	if ip, ok := ctx.Value("audit_ip_address").(string); ok {
		ipAddress = ip
	}
	if ua, ok := ctx.Value("audit_user_agent").(string); ok {
		userAgent = ua
	}
	return
}
