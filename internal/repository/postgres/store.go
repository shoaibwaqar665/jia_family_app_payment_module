package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jia-app/paymentservice/internal/domain"
	"github.com/jia-app/paymentservice/internal/repository"
	"github.com/jia-app/paymentservice/internal/repository/postgres/pgstore"
	_ "github.com/lib/pq"
	"github.com/sqlc-dev/pqtype"
)

// Store represents the PostgreSQL store implementation
type Store struct {
	db      *sql.DB
	queries *pgstore.Queries
}

// NewStore creates a new PostgreSQL store
func NewStore(connString string) (*Store, error) {
	db, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Store{db: db, queries: pgstore.New(db)}, nil
}

// NewStoreWithDB creates a new PostgreSQL store with an existing database connection
func NewStoreWithDB(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection cannot be nil")
	}

	return &Store{db: db, queries: pgstore.New(db)}, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Transaction represents a database transaction
type Transaction struct {
	tx      *sql.Tx
	queries *pgstore.Queries
}

// BeginTx starts a new transaction
func (s *Store) BeginTx(ctx context.Context) (*Transaction, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &Transaction{
		tx:      tx,
		queries: pgstore.New(tx),
	}, nil
}

// Commit commits the transaction
func (t *Transaction) Commit() error {
	if err := t.tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// Rollback rolls back the transaction
func (t *Transaction) Rollback() error {
	if err := t.tx.Rollback(); err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}
	return nil
}

// Payment returns the payment repository implementation for this transaction
func (t *Transaction) Payment() repository.PaymentRepository {
	return &paymentRepository{store: &Store{db: nil, queries: t.queries}}
}

// Plan returns the plan repository implementation for this transaction
func (t *Transaction) Plan() repository.PlanRepository {
	return &planRepository{store: &Store{db: nil, queries: t.queries}}
}

// Entitlement returns the entitlement repository implementation for this transaction
func (t *Transaction) Entitlement() repository.EntitlementRepository {
	return &entitlementRepository{store: &Store{db: nil, queries: t.queries}}
}

// WebhookEvents returns the webhook events repository implementation for this transaction
func (t *Transaction) WebhookEvents() repository.WebhookEventsRepository {
	return &webhookEventsRepository{store: &Store{db: nil, queries: t.queries}}
}

// Outbox returns the outbox repository implementation for this transaction
func (t *Transaction) Outbox() repository.OutboxRepository {
	return &outboxRepository{store: &Store{db: nil, queries: t.queries}}
}

// WithTransaction executes a function within a transaction (implements TransactionManager interface)
func (s *Store) WithTransaction(ctx context.Context, fn func(repository.Transaction) error) error {
	tx, err := s.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			// Rollback on panic
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				// Log rollback error but don't mask the original panic
				fmt.Printf("Failed to rollback transaction after panic: %v\n", rollbackErr)
			}
			panic(p) // Re-throw the panic
		}
	}()

	if err := fn(tx); err != nil {
		// Rollback on error
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("failed to rollback transaction: %w (original error: %v)", rollbackErr, err)
		}
		return err
	}

	// Commit on success
	return tx.Commit()
}

// Payment returns the payment repository implementation
func (s *Store) Payment() repository.PaymentRepository {
	return &paymentRepository{store: s}
}

// Plan returns the plan repository implementation
func (s *Store) Plan() repository.PlanRepository {
	return &planRepository{store: s}
}

// Entitlement returns the entitlement repository implementation
func (s *Store) Entitlement() repository.EntitlementRepository {
	return &entitlementRepository{store: s}
}

// WebhookEvents returns the webhook events repository implementation
func (s *Store) WebhookEvents() repository.WebhookEventsRepository {
	return &webhookEventsRepository{store: s}
}

// Outbox returns the outbox repository implementation
func (s *Store) Outbox() repository.OutboxRepository {
	return &outboxRepository{store: s}
}

// paymentRepository implements repository.PaymentRepository
type paymentRepository struct {
	store *Store
}

// Create creates a new payment using sqlc generated code
func (r *paymentRepository) Create(ctx context.Context, payment *domain.Payment) error {
	// Handle nullable Stripe fields safely
	var stripePaymentIntentID sql.NullString
	if payment.StripePaymentIntentID != nil {
		stripePaymentIntentID = sql.NullString{String: *payment.StripePaymentIntentID, Valid: true}
	}

	var stripeSessionID sql.NullString
	if payment.StripeSessionID != nil {
		stripeSessionID = sql.NullString{String: *payment.StripeSessionID, Valid: true}
	}

	_, err := r.store.queries.CreatePayment(ctx, pgstore.CreatePaymentParams{
		ID:                    payment.ID,
		Amount:                int32(payment.Amount),
		Currency:              payment.Currency,
		Status:                payment.Status,
		PaymentMethod:         payment.PaymentMethod,
		CustomerID:            payment.CustomerID,
		OrderID:               payment.OrderID,
		Description:           sql.NullString{String: payment.Description, Valid: payment.Description != ""},
		StripePaymentIntentID: stripePaymentIntentID,
		StripeSessionID:       stripeSessionID,
		Metadata:              pqtype.NullRawMessage{RawMessage: payment.Metadata, Valid: len(payment.Metadata) > 0},
	})
	if err != nil {
		return fmt.Errorf("failed to create payment: %w", err)
	}
	return nil
}

// Payment repository methods are implemented in store_sqlc.go

// planRepository implements repository.PlanRepository
type planRepository struct {
	store *Store
}

// Plan repository methods are implemented in store_sqlc.go

// entitlementRepository implements repository.EntitlementRepository
type entitlementRepository struct {
	store *Store
}

// Entitlement repository methods are implemented in store_sqlc.go

// webhookEventsRepository implements repository.WebhookEventsRepository
type webhookEventsRepository struct {
	store *Store
}

// Outbox repository methods are implemented in store_sqlc.go

// outboxRepository implements repository.OutboxRepository
type outboxRepository struct {
	store *Store
}

// Outbox repository methods are implemented in store_sqlc.go
