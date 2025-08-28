package postgres

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jia-app/paymentservice/internal/payment/domain"
	"github.com/jia-app/paymentservice/internal/payment/repo"
	"github.com/jia-app/paymentservice/internal/payment/repo/postgres/pgstore"
)

// Store represents the PostgreSQL store implementation
type Store struct {
	db      *pgxpool.Pool
	queries *pgstore.Queries
}

// NewStore creates a new PostgreSQL store
func NewStore(connString string) (*Store, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	queries := pgstore.New()
	return &Store{db: pool, queries: queries}, nil
}

// NewStoreWithPool creates a new PostgreSQL store with an existing pool
func NewStoreWithPool(pool *pgxpool.Pool) (*Store, error) {
	if pool == nil {
		return nil, fmt.Errorf("database pool cannot be nil")
	}

	queries := pgstore.New()
	return &Store{db: pool, queries: queries}, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	if s.db != nil {
		s.db.Close()
	}
	return nil
}

// Payment returns the payment repository implementation
func (s *Store) Payment() repo.PaymentRepository {
	// TODO: Return actual implementation
	return &paymentRepository{store: s}
}

// Plan returns the plan repository implementation
func (s *Store) Plan() repo.PlanRepository {
	return &planRepository{store: s}
}

// Entitlement returns the entitlement repository implementation
func (s *Store) Entitlement() repo.EntitlementRepository {
	return &entitlementRepository{store: s}
}

// PricingZone returns the pricing zone repository implementation
func (s *Store) PricingZone() repo.PricingZoneRepository {
	return &pricingZoneRepository{store: s}
}

// paymentRepository implements repository.PaymentRepository
type paymentRepository struct {
	store *Store
}

// Create creates a new payment
func (r *paymentRepository) Create(ctx context.Context, payment *domain.Payment) error {
	// TODO: Implement with sqlc generated queries
	return fmt.Errorf("not implemented")
}

// GetByID retrieves a payment by ID
func (r *paymentRepository) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	// TODO: Implement with sqlc generated queries
	return nil, fmt.Errorf("not implemented")
}

// GetByOrderID retrieves a payment by order ID
func (r *paymentRepository) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	// TODO: Implement with sqlc generated queries
	return nil, fmt.Errorf("not implemented")
}

// GetByCustomerID retrieves payments by customer ID
func (r *paymentRepository) GetByCustomerID(ctx context.Context, customerID string, limit, offset int) ([]*domain.Payment, error) {
	// TODO: Implement with sqlc generated queries
	return nil, fmt.Errorf("not implemented")
}

// Update updates an existing payment
func (r *paymentRepository) Update(ctx context.Context, payment *domain.Payment) error {
	// TODO: Implement with sqlc generated queries
	return fmt.Errorf("not implemented")
}

// UpdateStatus updates only the status of a payment
func (r *paymentRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	// TODO: Implement with sqlc generated queries
	return fmt.Errorf("not implemented")
}

// Delete deletes a payment (soft delete)
func (r *paymentRepository) Delete(ctx context.Context, id string) error {
	// TODO: Implement with sqlc generated queries
	return fmt.Errorf("not implemented")
}

// List retrieves a list of payments with pagination
func (r *paymentRepository) List(ctx context.Context, limit, offset int) ([]*domain.Payment, error) {
	// TODO: Implement with sqlc generated queries
	return nil, fmt.Errorf("not implemented")
}

// Count returns the total number of payments
func (r *paymentRepository) Count(ctx context.Context) (int64, error) {
	// TODO: Implement with sqlc generated queries
	return 0, fmt.Errorf("not implemented")
}

// planRepository implements repository.PlanRepository
type planRepository struct {
	store *Store
}

// GetByID retrieves a plan by ID
func (r *planRepository) GetByID(ctx context.Context, id string) (domain.Plan, error) {
	// TODO: Implement with sqlc generated queries
	return domain.Plan{}, fmt.Errorf("not implemented")
}

// ListActive retrieves all active plans
func (r *planRepository) ListActive(ctx context.Context) ([]domain.Plan, error) {
	// TODO: Implement with sqlc generated queries
	return nil, fmt.Errorf("not implemented")
}

// entitlementRepository implements repository.EntitlementRepository
type entitlementRepository struct {
	store *Store
}

// Check checks if a user has an active entitlement for a feature
func (r *entitlementRepository) Check(ctx context.Context, userID, featureCode string) (domain.Entitlement, bool, error) {
	// TODO: Implement with sqlc generated queries
	return domain.Entitlement{}, false, fmt.Errorf("not implemented")
}

// ListByUser retrieves all entitlements for a user
func (r *entitlementRepository) ListByUser(ctx context.Context, userID string) ([]domain.Entitlement, error) {
	// TODO: Implement with sqlc generated queries
	return nil, fmt.Errorf("not implemented")
}

// Insert creates a new entitlement
func (r *entitlementRepository) Insert(ctx context.Context, e domain.Entitlement) (domain.Entitlement, error) {
	// TODO: Implement with sqlc generated queries
	return domain.Entitlement{}, fmt.Errorf("not implemented")
}

// UpdateStatus updates the status of an entitlement
func (r *entitlementRepository) UpdateStatus(ctx context.Context, id, status string) error {
	// TODO: Implement with sqlc generated queries
	return fmt.Errorf("not implemented")
}

// UpdateExpiry updates the expiry time of an entitlement
func (r *entitlementRepository) UpdateExpiry(ctx context.Context, id string, expiresAt *time.Time) error {
	// TODO: Implement with sqlc generated queries
	return fmt.Errorf("not implemented")
}

// pricingZoneRepository implements repository.PricingZoneRepository
type pricingZoneRepository struct {
	store *Store
}

// GetByISOCode retrieves a pricing zone by ISO country code
func (r *pricingZoneRepository) GetByISOCode(ctx context.Context, isoCode string) (domain.PricingZone, error) {
	pricingZone, err := r.store.queries.GetPricingZoneByISOCode(ctx, r.store.db, isoCode)
	if err != nil {
		return domain.PricingZone{}, err
	}
	return convertPricingZoneFromDB(pricingZone), nil
}

// GetByCountry retrieves a pricing zone by country name
func (r *pricingZoneRepository) GetByCountry(ctx context.Context, country string) (domain.PricingZone, error) {
	pricingZone, err := r.store.queries.GetPricingZoneByCountry(ctx, r.store.db, country)
	if err != nil {
		return domain.PricingZone{}, err
	}
	return convertPricingZoneFromDB(pricingZone), nil
}

// GetByZone retrieves all pricing zones for a specific zone type
func (r *pricingZoneRepository) GetByZone(ctx context.Context, zone string) ([]domain.PricingZone, error) {
	pricingZones, err := r.store.queries.GetPricingZonesByZone(ctx, r.store.db, zone)
	if err != nil {
		return nil, err
	}
	return convertPricingZonesFromDB(pricingZones), nil
}

// List retrieves all pricing zones
func (r *pricingZoneRepository) List(ctx context.Context) ([]domain.PricingZone, error) {
	pricingZones, err := r.store.queries.ListPricingZones(ctx, r.store.db)
	if err != nil {
		return nil, err
	}
	return convertPricingZonesFromDB(pricingZones), nil
}

// Upsert creates or updates a pricing zone
func (r *pricingZoneRepository) Upsert(ctx context.Context, zone domain.PricingZone) (domain.PricingZone, error) {
	params := pgstore.UpsertPricingZoneParams{
		Country:                 zone.Country,
		IsoCode:                 zone.ISOCode,
		Zone:                    zone.Zone,
		ZoneName:                zone.ZoneName,
		WorldBankClassification: pgtype.Text{String: zone.WorldBankClassification, Valid: zone.WorldBankClassification != ""},
		GniPerCapitaThreshold:   pgtype.Text{String: zone.GNIPerCapitaThreshold, Valid: zone.GNIPerCapitaThreshold != ""},
		PricingMultiplier:       pgtype.Numeric{Int: big.NewInt(int64(zone.PricingMultiplier * 100)), Valid: true, Exp: -2},
	}

	pricingZone, err := r.store.queries.UpsertPricingZone(ctx, r.store.db, params)
	if err != nil {
		return domain.PricingZone{}, err
	}
	return convertPricingZoneFromDB(pricingZone), nil
}

// BulkUpsert creates or updates multiple pricing zones
func (r *pricingZoneRepository) BulkUpsert(ctx context.Context, zones []domain.PricingZone) error {
	for _, zone := range zones {
		_, err := r.Upsert(ctx, zone)
		if err != nil {
			return fmt.Errorf("failed to upsert zone %s: %w", zone.ISOCode, err)
		}
	}
	return nil
}

// Delete deletes a pricing zone by ISO code
func (r *pricingZoneRepository) Delete(ctx context.Context, isoCode string) error {
	return r.store.queries.DeletePricingZone(ctx, r.store.db, isoCode)
}

// Helper functions to convert between domain and database models
func convertPricingZoneFromDB(dbZone *pgstore.PricingZone) domain.PricingZone {
	var multiplier float64
	if dbZone.PricingMultiplier.Valid {
		if val, err := dbZone.PricingMultiplier.Float64Value(); err == nil {
			multiplier = val.Float64
		}
	}

	var worldBankClass, gniThreshold string
	if dbZone.WorldBankClassification.Valid {
		worldBankClass = dbZone.WorldBankClassification.String
	}
	if dbZone.GniPerCapitaThreshold.Valid {
		gniThreshold = dbZone.GniPerCapitaThreshold.String
	}

	var createdAt, updatedAt time.Time
	if dbZone.CreatedAt.Valid {
		createdAt = dbZone.CreatedAt.Time
	}
	if dbZone.UpdatedAt.Valid {
		updatedAt = dbZone.UpdatedAt.Time
	}

	return domain.PricingZone{
		ID:                      string(dbZone.ID.Bytes[:]),
		Country:                 dbZone.Country,
		ISOCode:                 dbZone.IsoCode,
		Zone:                    dbZone.Zone,
		ZoneName:                dbZone.ZoneName,
		WorldBankClassification: worldBankClass,
		GNIPerCapitaThreshold:   gniThreshold,
		PricingMultiplier:       multiplier,
		CreatedAt:               createdAt,
		UpdatedAt:               updatedAt,
	}
}

func convertPricingZonesFromDB(dbZones []*pgstore.PricingZone) []domain.PricingZone {
	zones := make([]domain.PricingZone, len(dbZones))
	for i, dbZone := range dbZones {
		zones[i] = convertPricingZoneFromDB(dbZone)
	}
	return zones
}
