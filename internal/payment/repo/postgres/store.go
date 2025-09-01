package postgres

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
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
	params := pgstore.CreatePaymentParams{
		Amount:            pgtype.Numeric{Int: big.NewInt(int64(payment.Amount * 100)), Valid: true}, // Convert dollars to cents for pgtype.Numeric
		Currency:          payment.Currency,
		Status:            payment.Status,
		PaymentMethod:     payment.PaymentMethod,
		CustomerID:        payment.CustomerID,
		OrderID:           payment.OrderID,
		Description:       pgtype.Text{String: payment.Description, Valid: payment.Description != ""},
		ExternalPaymentID: pgtype.Text{String: payment.ExternalPaymentID, Valid: payment.ExternalPaymentID != ""},
		FailureReason:     pgtype.Text{String: payment.FailureReason, Valid: payment.FailureReason != ""},
		Metadata:          payment.Metadata,
	}

	dbPayment, err := r.store.queries.CreatePayment(ctx, r.store.db, params)
	if err != nil {
		return fmt.Errorf("failed to create payment: %w", err)
	}

	// Update the payment with the generated ID and timestamps
	payment.ID = dbPayment.ID.Bytes
	if dbPayment.CreatedAt.Valid {
		payment.CreatedAt = dbPayment.CreatedAt.Time
	}
	if dbPayment.UpdatedAt.Valid {
		payment.UpdatedAt = dbPayment.UpdatedAt.Time
	}
	return nil
}

// GetByID retrieves a payment by ID
func (r *paymentRepository) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	paymentUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid payment ID: %w", err)
	}

	dbPayment, err := r.store.queries.GetPaymentByID(ctx, r.store.db, pgtype.UUID{Bytes: paymentUUID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	return convertPaymentFromDB(dbPayment), nil
}

// GetByOrderID retrieves a payment by order ID
func (r *paymentRepository) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	dbPayment, err := r.store.queries.GetPaymentByOrderID(ctx, r.store.db, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment by order ID: %w", err)
	}

	return convertPaymentFromDB(dbPayment), nil
}

// GetByCustomerID retrieves payments by customer ID
func (r *paymentRepository) GetByCustomerID(ctx context.Context, customerID string, limit, offset int) ([]*domain.Payment, error) {
	dbPayments, err := r.store.queries.GetPaymentsByCustomerID(ctx, r.store.db, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payments by customer ID: %w", err)
	}

	payments := make([]*domain.Payment, len(dbPayments))
	for i, dbPayment := range dbPayments {
		payments[i] = convertPaymentFromDB(dbPayment)
	}

	// Apply pagination manually since sqlc doesn't support it in this version
	if offset > 0 && offset < len(payments) {
		payments = payments[offset:]
	}
	if limit > 0 && limit < len(payments) {
		payments = payments[:limit]
	}

	return payments, nil
}

// Update updates an existing payment
func (r *paymentRepository) Update(ctx context.Context, payment *domain.Payment) error {
	params := pgstore.UpdatePaymentParams{
		ID:                pgtype.UUID{Bytes: payment.ID, Valid: true},
		Amount:            pgtype.Numeric{Int: big.NewInt(int64(payment.Amount * 100)), Valid: true}, // Convert dollars to cents for pgtype.Numeric
		Currency:          payment.Currency,
		Status:            payment.Status,
		PaymentMethod:     payment.PaymentMethod,
		CustomerID:        payment.CustomerID,
		OrderID:           payment.OrderID,
		Description:       pgtype.Text{String: payment.Description, Valid: payment.Description != ""},
		ExternalPaymentID: pgtype.Text{String: payment.ExternalPaymentID, Valid: payment.ExternalPaymentID != ""},
		FailureReason:     pgtype.Text{String: payment.FailureReason, Valid: payment.FailureReason != ""},
		Metadata:          payment.Metadata,
	}

	_, err := r.store.queries.UpdatePayment(ctx, r.store.db, params)
	if err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}

	return nil
}

// UpdateStatus updates only the status of a payment
func (r *paymentRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	paymentUUID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid payment ID: %w", err)
	}

	_, err = r.store.queries.UpdatePaymentStatus(ctx, r.store.db, pgstore.UpdatePaymentStatusParams{
		ID:            pgtype.UUID{Bytes: paymentUUID, Valid: true},
		Status:        status,
		FailureReason: pgtype.Text{String: "", Valid: false}, // No failure reason for status updates
	})
	if err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	return nil
}

// Delete deletes a payment (hard delete)
func (r *paymentRepository) Delete(ctx context.Context, id string) error {
	paymentUUID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid payment ID: %w", err)
	}

	err = r.store.queries.DeletePayment(ctx, r.store.db, pgtype.UUID{Bytes: paymentUUID, Valid: true})
	if err != nil {
		return fmt.Errorf("failed to delete payment: %w", err)
	}

	return nil
}

// List retrieves a list of payments with pagination
func (r *paymentRepository) List(ctx context.Context, limit, offset int) ([]*domain.Payment, error) {
	dbPayments, err := r.store.queries.ListPayments(ctx, r.store.db)
	if err != nil {
		return nil, fmt.Errorf("failed to list payments: %w", err)
	}

	payments := make([]*domain.Payment, len(dbPayments))
	for i, dbPayment := range dbPayments {
		payments[i] = convertPaymentFromDB(dbPayment)
	}

	// Apply pagination manually since sqlc doesn't support it in this version
	if offset > 0 && offset < len(payments) {
		payments = payments[offset:]
	}
	if limit > 0 && limit < len(payments) {
		payments = payments[:limit]
	}

	return payments, nil
}

// Count returns the total number of payments
func (r *paymentRepository) Count(ctx context.Context) (int64, error) {
	count, err := r.store.queries.CountPayments(ctx, r.store.db)
	if err != nil {
		return 0, fmt.Errorf("failed to count payments: %w", err)
	}

	return count, nil
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
	entitlements, err := r.store.queries.ListEntitlementsByUser(ctx, r.store.db, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list entitlements by user: %w", err)
	}

	var result []domain.Entitlement
	for _, ent := range entitlements {
		// Handle both UUID and string plan IDs
		var planID uuid.UUID
		if parsedUUID, err := uuid.Parse(ent.PlanID); err == nil {
			planID = parsedUUID
		} else {
			// It's a string plan ID, generate a deterministic UUID
			planID = uuid.NewSHA1(uuid.NameSpaceOID, []byte(ent.PlanID))
		}

		domainEnt := domain.Entitlement{
			ID:          ent.ID.Bytes,
			UserID:      ent.UserID,
			FeatureCode: ent.FeatureCode,
			PlanID:      planID,
			Status:      ent.Status,
			GrantedAt:   ent.GrantedAt.Time,
			CreatedAt:   ent.CreatedAt.Time,
			UpdatedAt:   ent.UpdatedAt.Time,
		}

		// Handle optional fields
		if ent.FamilyID.Valid {
			domainEnt.FamilyID = &ent.FamilyID.String
		}
		if ent.SubscriptionID.Valid {
			domainEnt.SubscriptionID = &ent.SubscriptionID.String
		}
		if ent.ExpiresAt.Valid {
			domainEnt.ExpiresAt = &ent.ExpiresAt.Time
		}

		result = append(result, domainEnt)
	}

	return result, nil
}

// Insert creates a new entitlement
func (r *entitlementRepository) Insert(ctx context.Context, e domain.Entitlement) (domain.Entitlement, error) {
	// For now, we'll use a hardcoded mapping since the domain model uses UUID but DB expects string
	// TODO: Refactor to use string plan IDs in domain model
	planIDString := e.PlanID.String()

	// Map UUID-based plan IDs to string plan IDs for database foreign key constraint
	switch e.PlanID.String() {
	case uuid.NewSHA1(uuid.NameSpaceOID, []byte("basic_monthly")).String():
		planIDString = "basic_monthly"
	case uuid.NewSHA1(uuid.NameSpaceOID, []byte("pro_monthly")).String():
		planIDString = "pro_monthly"
	case uuid.NewSHA1(uuid.NameSpaceOID, []byte("family_monthly")).String():
		planIDString = "family_monthly"
	}

	params := pgstore.InsertEntitlementParams{
		UserID:      e.UserID,
		FeatureCode: e.FeatureCode,
		PlanID:      planIDString,
		Status:      e.Status,
		GrantedAt:   pgtype.Timestamp{Time: e.GrantedAt, Valid: true},
		UsageLimits: []byte("{}"),
		Metadata:    []byte("{}"),
	}

	// Handle optional fields
	if e.FamilyID != nil {
		params.FamilyID = pgtype.Text{String: *e.FamilyID, Valid: true}
	}
	if e.SubscriptionID != nil {
		params.SubscriptionID = pgtype.Text{String: *e.SubscriptionID, Valid: true}
	}
	if e.ExpiresAt != nil {
		params.ExpiresAt = pgtype.Timestamp{Time: *e.ExpiresAt, Valid: true}
	}

	entitlement, err := r.store.queries.InsertEntitlement(ctx, r.store.db, params)
	if err != nil {
		return domain.Entitlement{}, fmt.Errorf("failed to insert entitlement: %w", err)
	}

	// Convert back to domain model
	// Handle both UUID and string plan IDs
	var resultPlanID uuid.UUID
	if parsedUUID, err := uuid.Parse(entitlement.PlanID); err == nil {
		resultPlanID = parsedUUID
	} else {
		// It's a string plan ID, generate a deterministic UUID
		resultPlanID = uuid.NewSHA1(uuid.NameSpaceOID, []byte(entitlement.PlanID))
	}

	result := domain.Entitlement{
		ID:          entitlement.ID.Bytes,
		UserID:      entitlement.UserID,
		FeatureCode: entitlement.FeatureCode,
		PlanID:      resultPlanID,
		Status:      entitlement.Status,
		GrantedAt:   entitlement.GrantedAt.Time,
		CreatedAt:   entitlement.CreatedAt.Time,
		UpdatedAt:   entitlement.UpdatedAt.Time,
	}

	// Handle optional fields
	if entitlement.FamilyID.Valid {
		result.FamilyID = &entitlement.FamilyID.String
	}
	if entitlement.SubscriptionID.Valid {
		result.SubscriptionID = &entitlement.SubscriptionID.String
	}
	if entitlement.ExpiresAt.Valid {
		result.ExpiresAt = &entitlement.ExpiresAt.Time
	}

	return result, nil
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
		ID:                      uuid.UUID(dbZone.ID.Bytes).String(),
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

// Helper function to convert payment from database model to domain model
func convertPaymentFromDB(dbPayment *pgstore.Payment) *domain.Payment {
	var description, externalPaymentID, failureReason string
	if dbPayment.Description.Valid {
		description = dbPayment.Description.String
	}
	if dbPayment.ExternalPaymentID.Valid {
		externalPaymentID = dbPayment.ExternalPaymentID.String
	}
	if dbPayment.FailureReason.Valid {
		failureReason = dbPayment.FailureReason.String
	}

	var createdAt, updatedAt time.Time
	if dbPayment.CreatedAt.Valid {
		createdAt = dbPayment.CreatedAt.Time
	}
	if dbPayment.UpdatedAt.Valid {
		updatedAt = dbPayment.UpdatedAt.Time
	}

	// Convert pgtype.Numeric to float64 (dollars)
	var amount float64
	if dbPayment.Amount.Valid {
		if val, err := dbPayment.Amount.Float64Value(); err == nil {
			amount = val.Float64 / 100.0 // Convert cents back to dollars
		}
	}

	return &domain.Payment{
		ID:                dbPayment.ID.Bytes,
		Amount:            amount,
		Currency:          dbPayment.Currency,
		Status:            dbPayment.Status,
		PaymentMethod:     dbPayment.PaymentMethod,
		CustomerID:        dbPayment.CustomerID,
		OrderID:           dbPayment.OrderID,
		Description:       description,
		ExternalPaymentID: externalPaymentID,
		FailureReason:     failureReason,
		Metadata:          dbPayment.Metadata,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	}
}
