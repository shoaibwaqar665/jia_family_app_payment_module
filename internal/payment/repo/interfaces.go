package repo

import (
	"context"
	"time"

	"github.com/jia-app/paymentservice/internal/payment/domain"
)

type PlanRepository interface {
	GetByID(ctx context.Context, id string) (domain.Plan, error)
	ListActive(ctx context.Context) ([]domain.Plan, error)
}

type EntitlementRepository interface {
	Check(ctx context.Context, userID, featureCode string) (domain.Entitlement, bool, error)
	ListByUser(ctx context.Context, userID string) ([]domain.Entitlement, error)
	Insert(ctx context.Context, e domain.Entitlement) (domain.Entitlement, error)
	UpdateStatus(ctx context.Context, id, status string) error
	UpdateExpiry(ctx context.Context, id string, expiresAt *time.Time) error
}

type PaymentRepository interface {
	// Create creates a new payment
	Create(ctx context.Context, payment *domain.Payment) error

	// GetByID retrieves a payment by ID
	GetByID(ctx context.Context, id string) (*domain.Payment, error)

	// GetByOrderID retrieves a payment by order ID
	GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)

	// GetByCustomerID retrieves payments by customer ID
	GetByCustomerID(ctx context.Context, customerID string, limit, offset int) ([]*domain.Payment, error)

	// Update updates an existing payment
	Update(ctx context.Context, payment *domain.Payment) error

	// UpdateStatus updates only the status of a payment
	UpdateStatus(ctx context.Context, id string, status string) error

	// Delete deletes a payment (soft delete)
	Delete(ctx context.Context, id string) error

	// List retrieves a list of payments with pagination
	List(ctx context.Context, limit, offset int) ([]*domain.Payment, error)

	// Count returns the total number of payments
	Count(ctx context.Context) (int64, error)
}

type PricingZoneRepository interface {
	// GetByISOCode retrieves a pricing zone by ISO country code
	GetByISOCode(ctx context.Context, isoCode string) (domain.PricingZone, error)

	// GetByCountry retrieves a pricing zone by country name
	GetByCountry(ctx context.Context, country string) (domain.PricingZone, error)

	// GetByZone retrieves all pricing zones for a specific zone type
	GetByZone(ctx context.Context, zone string) ([]domain.PricingZone, error)

	// List retrieves all pricing zones
	List(ctx context.Context) ([]domain.PricingZone, error)

	// Upsert creates or updates a pricing zone
	Upsert(ctx context.Context, zone domain.PricingZone) (domain.PricingZone, error)

	// BulkUpsert creates or updates multiple pricing zones
	BulkUpsert(ctx context.Context, zones []domain.PricingZone) error

	// Delete deletes a pricing zone by ISO code
	Delete(ctx context.Context, isoCode string) error
}
