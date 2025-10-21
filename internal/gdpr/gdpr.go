package gdpr

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/jia-app/paymentservice/internal/audit"
	"github.com/jia-app/paymentservice/internal/log"
)

// Service provides GDPR compliance operations
type Service struct {
	paymentRepo     PaymentRepository
	entitlementRepo EntitlementRepository
	auditLogger     *audit.Manager
	logger          *zap.Logger
}

// PaymentRepository defines the interface for payment data operations
type PaymentRepository interface {
	GetByCustomerID(ctx context.Context, customerID string, limit, offset int) ([]*Payment, error)
	Delete(ctx context.Context, id string) error
}

// EntitlementRepository defines the interface for entitlement operations
type EntitlementRepository interface {
	ListByUser(ctx context.Context, userID string) ([]*Entitlement, error)
	Delete(ctx context.Context, userID, featureCode string) error
}

// Payment represents a payment transaction
type Payment struct {
	ID            string    `json:"id"`
	Amount        int64     `json:"amount"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	PaymentMethod string    `json:"payment_method"`
	CustomerID    string    `json:"customer_id"`
	OrderID       string    `json:"order_id"`
	Description   string    `json:"description"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Entitlement represents a user's entitlement
type Entitlement struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	FeatureCode string     `json:"feature_code"`
	PlanID      string     `json:"plan_id"`
	Status      string     `json:"status"`
	GrantedAt   time.Time  `json:"granted_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// NewService creates a new GDPR service
func NewService(
	paymentRepo PaymentRepository,
	entitlementRepo EntitlementRepository,
	auditLogger *audit.Manager,
	logger *zap.Logger,
) *Service {
	return &Service{
		paymentRepo:     paymentRepo,
		entitlementRepo: entitlementRepo,
		auditLogger:     auditLogger,
		logger:          logger,
	}
}

// UserData represents all user data for export
type UserData struct {
	UserID       string         `json:"user_id"`
	Payments     []*Payment     `json:"payments"`
	Entitlements []*Entitlement `json:"entitlements"`
	ExportedAt   time.Time      `json:"exported_at"`
}

// ExportUserData exports all data for a user (GDPR Right to Data Portability)
func (s *Service) ExportUserData(ctx context.Context, userID string) (*UserData, error) {
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	s.logger.Info("Exporting user data",
		zap.String("user_id", userID))

	// Export payments
	payments, err := s.exportPayments(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to export payments",
			zap.Error(err),
			zap.String("user_id", userID))
		return nil, status.Errorf(codes.Internal, "failed to export payments: %v", err)
	}

	// Export entitlements
	entitlements, err := s.exportEntitlements(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to export entitlements",
			zap.Error(err),
			zap.String("user_id", userID))
		return nil, status.Errorf(codes.Internal, "failed to export entitlements: %v", err)
	}

	userData := &UserData{
		UserID:       userID,
		Payments:     payments,
		Entitlements: entitlements,
		ExportedAt:   time.Now(),
	}

	// Log audit event
	if s.auditLogger != nil {
		s.auditLogger.LogEntitlementCreated(ctx, userID, "export", "user_data", "gdpr")
	}

	s.logger.Info("User data exported successfully",
		zap.String("user_id", userID),
		zap.Int("payments_count", len(payments)),
		zap.Int("entitlements_count", len(entitlements)))

	return userData, nil
}

// DeleteUserData deletes all data for a user (GDPR Right to be Forgotten)
func (s *Service) DeleteUserData(ctx context.Context, userID string) error {
	if userID == "" {
		return status.Error(codes.InvalidArgument, "user_id is required")
	}

	s.logger.Info("Deleting user data",
		zap.String("user_id", userID))

	// Delete entitlements
	if err := s.deleteEntitlements(ctx, userID); err != nil {
		s.logger.Error("Failed to delete entitlements",
			zap.Error(err),
			zap.String("user_id", userID))
		return status.Errorf(codes.Internal, "failed to delete entitlements: %v", err)
	}

	// Delete payments
	if err := s.deletePayments(ctx, userID); err != nil {
		s.logger.Error("Failed to delete payments",
			zap.Error(err),
			zap.String("user_id", userID))
		return status.Errorf(codes.Internal, "failed to delete payments: %v", err)
	}

	// Log audit event
	if s.auditLogger != nil {
		s.auditLogger.LogEntitlementDeleted(ctx, userID, "delete", "user_data")
	}

	s.logger.Info("User data deleted successfully",
		zap.String("user_id", userID))

	return nil
}

// exportPayments exports all payments for a user
func (s *Service) exportPayments(ctx context.Context, userID string) ([]*Payment, error) {
	var allPayments []*Payment
	limit := 100
	offset := 0

	for {
		payments, err := s.paymentRepo.GetByCustomerID(ctx, userID, limit, offset)
		if err != nil {
			return nil, fmt.Errorf("failed to get payments: %w", err)
		}

		if len(payments) == 0 {
			break
		}

		allPayments = append(allPayments, payments...)

		if len(payments) < limit {
			break
		}

		offset += limit
	}

	return allPayments, nil
}

// exportEntitlements exports all entitlements for a user
func (s *Service) exportEntitlements(ctx context.Context, userID string) ([]*Entitlement, error) {
	entitlements, err := s.entitlementRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entitlements: %w", err)
	}

	return entitlements, nil
}

// deletePayments deletes all payments for a user
func (s *Service) deletePayments(ctx context.Context, userID string) error {
	// Get all payments for the user
	payments, err := s.paymentRepo.GetByCustomerID(ctx, userID, 1000, 0)
	if err != nil {
		return fmt.Errorf("failed to get payments: %w", err)
	}

	// Delete each payment
	for _, payment := range payments {
		if err := s.paymentRepo.Delete(ctx, payment.ID); err != nil {
			log.Warn(ctx, "Failed to delete payment",
				zap.Error(err),
				zap.String("user_id", userID),
				zap.String("payment_id", payment.ID))
			// Continue deleting other payments even if one fails
		}
	}

	return nil
}

// deleteEntitlements deletes all entitlements for a user
func (s *Service) deleteEntitlements(ctx context.Context, userID string) error {
	// Get all entitlements for the user
	entitlements, err := s.entitlementRepo.ListByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get entitlements: %w", err)
	}

	// Delete each entitlement
	for _, entitlement := range entitlements {
		if err := s.entitlementRepo.Delete(ctx, userID, entitlement.FeatureCode); err != nil {
			log.Warn(ctx, "Failed to delete entitlement",
				zap.Error(err),
				zap.String("user_id", userID),
				zap.String("feature_code", entitlement.FeatureCode))
			// Continue deleting other entitlements even if one fails
		}
	}

	return nil
}

// FormatAsJSON formats user data as JSON
func (ud *UserData) FormatAsJSON() ([]byte, error) {
	return json.MarshalIndent(ud, "", "  ")
}

// FormatAsCSV formats user data as CSV
func (ud *UserData) FormatAsCSV() ([]byte, error) {
	// Simple CSV format
	csv := "type,id,amount,currency,status,created_at\n"

	for _, payment := range ud.Payments {
		csv += fmt.Sprintf("payment,%s,%d,%s,%s,%s\n",
			payment.ID,
			payment.Amount,
			payment.Currency,
			payment.Status,
			payment.CreatedAt.Format(time.RFC3339))
	}

	for _, entitlement := range ud.Entitlements {
		csv += fmt.Sprintf("entitlement,%s,%s,%s,%s,%s\n",
			entitlement.ID,
			entitlement.FeatureCode,
			entitlement.PlanID,
			entitlement.Status,
			entitlement.CreatedAt.Format(time.RFC3339))
	}

	return []byte(csv), nil
}
