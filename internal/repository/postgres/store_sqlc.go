package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jia-app/paymentservice/internal/domain"
	"github.com/jia-app/paymentservice/internal/repository"
	"github.com/jia-app/paymentservice/internal/repository/postgres/pgstore"
	"github.com/sqlc-dev/pqtype"
)

// ============================================================================
// Payment Repository Implementation (using sqlc generated code)
// ============================================================================

// GetByID retrieves a payment by ID using sqlc generated code
func (r *paymentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error) {
	payment, err := r.store.queries.GetPaymentByID(ctx, id)
	if err == sql.ErrNoRows {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get payment by ID: %w", err)
	}

	return convertPaymentFromPGStore(payment), nil
}

// GetByOrderID retrieves a payment by order ID using sqlc generated code
func (r *paymentRepository) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	payment, err := r.store.queries.GetPaymentByOrderID(ctx, orderID)
	if err == sql.ErrNoRows {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get payment by order ID: %w", err)
	}

	return convertPaymentFromPGStore(payment), nil
}

// GetByCustomerID retrieves payments for a customer using sqlc generated code with pagination
func (r *paymentRepository) GetByCustomerID(ctx context.Context, customerID uuid.UUID, limit, offset int) ([]*domain.Payment, int, error) {
	payments, err := r.store.queries.GetPaymentsByCustomerID(ctx, pgstore.GetPaymentsByCustomerIDParams{
		CustomerID: customerID.String(),
		Limit:      int32(limit),
		Offset:     int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get payments by customer ID: %w", err)
	}

	result := make([]*domain.Payment, len(payments))
	for i, payment := range payments {
		result[i] = convertPaymentFromPGStore(payment)
	}

	// Get total count for pagination
	total, err := r.store.queries.CountPayments(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count payments: %w", err)
	}

	return result, int(total), nil
}

// UpdateStatus updates the status of a payment using sqlc generated code
func (r *paymentRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.store.queries.UpdatePaymentStatus(ctx, pgstore.UpdatePaymentStatusParams{
		ID:     id,
		Status: status,
	})
	if err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}
	return nil
}

// Update updates a payment using sqlc generated code
func (r *paymentRepository) Update(ctx context.Context, payment *domain.Payment) error {
	var description sql.NullString
	if payment.Description != "" {
		description = sql.NullString{String: payment.Description, Valid: true}
	}

	var stripePaymentIntentID sql.NullString
	if payment.StripePaymentIntentID != nil {
		stripePaymentIntentID = sql.NullString{String: *payment.StripePaymentIntentID, Valid: true}
	}

	var stripeSessionID sql.NullString
	if payment.StripeSessionID != nil {
		stripeSessionID = sql.NullString{String: *payment.StripeSessionID, Valid: true}
	}

	_, err := r.store.queries.UpdatePayment(ctx, pgstore.UpdatePaymentParams{
		ID:                    payment.ID,
		Amount:                int32(payment.Amount),
		Currency:              payment.Currency,
		Status:                payment.Status,
		PaymentMethod:         payment.PaymentMethod,
		Description:           description,
		StripePaymentIntentID: stripePaymentIntentID,
		StripeSessionID:       stripeSessionID,
		Metadata:              pqtype.NullRawMessage{RawMessage: payment.Metadata, Valid: len(payment.Metadata) > 0},
	})
	if err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}
	return nil
}

// Delete deletes a payment using sqlc generated code
func (r *paymentRepository) Delete(ctx context.Context, id string) error {
	idUUID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid payment ID format: %w", err)
	}

	err = r.store.queries.DeletePayment(ctx, idUUID)
	if err != nil {
		return fmt.Errorf("failed to delete payment: %w", err)
	}
	return nil
}

// List retrieves a list of payments with pagination using sqlc generated code
func (r *paymentRepository) List(ctx context.Context, limit, offset int) ([]*domain.Payment, int, error) {
	payments, err := r.store.queries.ListPayments(ctx, pgstore.ListPaymentsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list payments: %w", err)
	}

	result := make([]*domain.Payment, len(payments))
	for i, payment := range payments {
		result[i] = convertPaymentFromPGStore(payment)
	}

	// Get total count for pagination
	total, err := r.store.queries.CountPayments(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count payments: %w", err)
	}

	return result, int(total), nil
}

// Count returns the total number of payments using sqlc generated code
func (r *paymentRepository) Count(ctx context.Context) (int64, error) {
	return r.store.queries.CountPayments(ctx)
}

// ============================================================================
// Plan Repository Implementation (using sqlc generated code)
// ============================================================================

// GetByID retrieves a plan by ID using sqlc generated code
func (r *planRepository) GetByID(ctx context.Context, id string) (domain.Plan, error) {
	plan, err := r.store.queries.GetPlanByID(ctx, id)
	if err == sql.ErrNoRows {
		return domain.Plan{}, repository.ErrNotFound
	}
	if err != nil {
		return domain.Plan{}, fmt.Errorf("failed to get plan by ID: %w", err)
	}

	return *convertPlanFromPGStore(plan), nil
}

// ListActive retrieves all active plans using sqlc generated code
func (r *planRepository) ListActive(ctx context.Context) ([]domain.Plan, error) {
	plans, err := r.store.queries.ListActivePlans(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list active plans: %w", err)
	}

	result := make([]domain.Plan, len(plans))
	for i, plan := range plans {
		result[i] = *convertPlanFromPGStore(plan)
	}

	return result, nil
}

// Create creates a new plan using sqlc generated code
func (r *planRepository) Create(ctx context.Context, plan *domain.Plan) error {
	var description sql.NullString
	if plan.Description != "" {
		description = sql.NullString{String: plan.Description, Valid: true}
	}

	var billingCycle sql.NullString
	if plan.BillingCycle != "" {
		billingCycle = sql.NullString{String: plan.BillingCycle, Valid: true}
	}

	var maxUsers sql.NullInt32
	if plan.MaxUsers > 0 {
		maxUsers = sql.NullInt32{Int32: plan.MaxUsers, Valid: true}
	}

	_, err := r.store.queries.InsertPlan(ctx, pgstore.InsertPlanParams{
		ID:           plan.ID.String(),
		Name:         plan.Name,
		Description:  description,
		FeatureCodes: plan.FeatureCodes,
		BillingCycle: billingCycle,
		PriceCents:   int32(plan.PriceCents),
		Currency:     plan.Currency,
		MaxUsers:     maxUsers,
		UsageLimits:  pqtype.NullRawMessage{RawMessage: plan.UsageLimits, Valid: len(plan.UsageLimits) > 0},
		Metadata:     pqtype.NullRawMessage{RawMessage: plan.Metadata, Valid: len(plan.Metadata) > 0},
		Active:       plan.Active,
	})
	if err != nil {
		return fmt.Errorf("failed to create plan: %w", err)
	}
	return nil
}

// UpdateActive updates the active status of a plan using sqlc generated code
func (r *planRepository) UpdateActive(ctx context.Context, id string, active bool) error {
	_, err := r.store.queries.UpdatePlanActive(ctx, pgstore.UpdatePlanActiveParams{
		ID:     id,
		Active: active,
	})
	if err != nil {
		return fmt.Errorf("failed to update plan active status: %w", err)
	}
	return nil
}

// ============================================================================
// Entitlement Repository Implementation (using sqlc generated code)
// ============================================================================

// Check checks if a user has an entitlement for a feature using sqlc generated code
func (r *entitlementRepository) Check(ctx context.Context, userID, featureCode string) (domain.Entitlement, bool, error) {
	entitlement, err := r.store.queries.CheckEntitlement(ctx, pgstore.CheckEntitlementParams{
		UserID:      userID,
		FeatureCode: featureCode,
	})
	if err == sql.ErrNoRows {
		return domain.Entitlement{}, false, nil
	}
	if err != nil {
		return domain.Entitlement{}, false, fmt.Errorf("failed to check entitlement: %w", err)
	}

	return convertEntitlementFromPGStore(entitlement), true, nil
}

// ListByUser lists all entitlements for a user using sqlc generated code
func (r *entitlementRepository) ListByUser(ctx context.Context, userID string) ([]domain.Entitlement, error) {
	entitlements, err := r.store.queries.ListEntitlementsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list entitlements by user: %w", err)
	}

	result := make([]domain.Entitlement, len(entitlements))
	for i, entitlement := range entitlements {
		result[i] = convertEntitlementFromPGStore(entitlement)
	}

	return result, nil
}

// Insert creates a new entitlement using sqlc generated code
func (r *entitlementRepository) Insert(ctx context.Context, entitlement domain.Entitlement) (domain.Entitlement, error) {
	var familyID sql.NullString
	if entitlement.FamilyID != nil {
		familyID = sql.NullString{String: *entitlement.FamilyID, Valid: true}
	}

	var subscriptionID sql.NullString
	if entitlement.SubscriptionID != nil {
		subscriptionID = sql.NullString{String: *entitlement.SubscriptionID, Valid: true}
	}

	var expiresAt sql.NullTime
	if entitlement.ExpiresAt != nil {
		expiresAt = sql.NullTime{Time: *entitlement.ExpiresAt, Valid: true}
	}

	inserted, err := r.store.queries.InsertEntitlement(ctx, pgstore.InsertEntitlementParams{
		UserID:         entitlement.UserID,
		FamilyID:       familyID,
		FeatureCode:    entitlement.FeatureCode,
		PlanID:         entitlement.PlanID.String(),
		SubscriptionID: subscriptionID,
		Status:         entitlement.Status,
		GrantedAt:      entitlement.GrantedAt,
		ExpiresAt:      expiresAt,
		UsageLimits:    pqtype.NullRawMessage{RawMessage: entitlement.UsageLimits, Valid: len(entitlement.UsageLimits) > 0},
		Metadata:       pqtype.NullRawMessage{RawMessage: entitlement.Metadata, Valid: len(entitlement.Metadata) > 0},
	})
	if err != nil {
		return domain.Entitlement{}, fmt.Errorf("failed to insert entitlement: %w", err)
	}

	result := convertEntitlementFromPGStore(inserted)
	return result, nil
}

// UpdateStatus updates the status of an entitlement using sqlc generated code
func (r *entitlementRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	idUUID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid entitlement ID format: %w", err)
	}

	_, err = r.store.queries.UpdateEntitlementStatus(ctx, pgstore.UpdateEntitlementStatusParams{
		ID:     idUUID,
		Status: status,
	})
	if err != nil {
		return fmt.Errorf("failed to update entitlement status: %w", err)
	}
	return nil
}

// UpdateExpiry updates the expiry of an entitlement using sqlc generated code
func (r *entitlementRepository) UpdateExpiry(ctx context.Context, id string, expiresAt *time.Time) error {
	idUUID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid entitlement ID format: %w", err)
	}

	var expiresAtNull sql.NullTime
	if expiresAt != nil {
		expiresAtNull = sql.NullTime{Time: *expiresAt, Valid: true}
	}

	_, err = r.store.queries.UpdateEntitlementExpiry(ctx, pgstore.UpdateEntitlementExpiryParams{
		ID:        idUUID,
		ExpiresAt: expiresAtNull,
	})
	if err != nil {
		return fmt.Errorf("failed to update entitlement expiry: %w", err)
	}
	return nil
}

// GetByID retrieves an entitlement by ID using sqlc generated code
func (r *entitlementRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Entitlement, error) {
	entitlement, err := r.store.queries.GetEntitlementByID(ctx, id)
	if err == sql.ErrNoRows {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get entitlement by ID: %w", err)
	}

	result := convertEntitlementFromPGStore(entitlement)
	return &result, nil
}

// ListExpiring lists entitlements that are expiring using sqlc generated code
func (r *entitlementRepository) ListExpiring(ctx context.Context) ([]*domain.Entitlement, error) {
	entitlements, err := r.store.queries.ListExpiringEntitlements(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list expiring entitlements: %w", err)
	}

	result := make([]*domain.Entitlement, len(entitlements))
	for i, entitlement := range entitlements {
		converted := convertEntitlementFromPGStore(entitlement)
		result[i] = &converted
	}

	return result, nil
}

// ============================================================================
// Webhook Events Repository Implementation (using sqlc generated code)
// ============================================================================

// Insert creates a new webhook event using sqlc generated code
func (r *webhookEventsRepository) Insert(ctx context.Context, eventID, eventType, signature string, payload []byte) error {
	_, err := r.store.queries.InsertWebhookEvent(ctx, pgstore.InsertWebhookEventParams{
		EventID:   eventID,
		EventType: eventType,
		Payload:   payload,
		Signature: signature,
	})
	if err != nil {
		return fmt.Errorf("failed to insert webhook event: %w", err)
	}
	return nil
}

// GetByEventID retrieves a webhook event by event ID using sqlc generated code
func (r *webhookEventsRepository) GetByEventID(ctx context.Context, eventID string) (*domain.WebhookEvent, error) {
	webhookEvent, err := r.store.queries.GetWebhookEventByEventID(ctx, eventID)
	if err == sql.ErrNoRows {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook event by event ID: %w", err)
	}

	return convertWebhookEventFromPGStore(webhookEvent), nil
}

// MarkProcessed marks a webhook event as processed using sqlc generated code
func (r *webhookEventsRepository) MarkProcessed(ctx context.Context, eventID string) error {
	_, err := r.store.queries.MarkWebhookEventProcessed(ctx, eventID)
	if err != nil {
		return fmt.Errorf("failed to mark webhook event as processed: %w", err)
	}
	return nil
}

// ============================================================================
// Outbox Repository Implementation (using sqlc generated code)
// ============================================================================

// Insert creates a new outbox event using sqlc generated code
func (r *outboxRepository) Insert(ctx context.Context, eventType string, payload []byte) error {
	_, err := r.store.queries.InsertOutboxEvent(ctx, pgstore.InsertOutboxEventParams{
		EventType: eventType,
		Payload:   payload,
	})
	if err != nil {
		return fmt.Errorf("failed to insert outbox event: %w", err)
	}
	return nil
}

// GetPending retrieves pending outbox events using sqlc generated code
func (r *outboxRepository) GetPending(ctx context.Context, limit int) ([]domain.OutboxEvent, error) {
	outboxEvents, err := r.store.queries.GetPendingOutboxEvents(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("failed to get pending outbox events: %w", err)
	}

	result := make([]domain.OutboxEvent, len(outboxEvents))
	for i, event := range outboxEvents {
		result[i] = convertOutboxEventFromPGStore(event)
	}

	return result, nil
}

// MarkPublished marks an outbox event as published using sqlc generated code
func (r *outboxRepository) MarkPublished(ctx context.Context, id string) error {
	idUUID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid outbox event ID format: %w", err)
	}

	_, err = r.store.queries.MarkOutboxEventPublished(ctx, idUUID)
	if err != nil {
		return fmt.Errorf("failed to mark outbox event as published: %w", err)
	}
	return nil
}

// MarkFailed marks an outbox event as failed using sqlc generated code
func (r *outboxRepository) MarkFailed(ctx context.Context, id string, errorMessage string) error {
	idUUID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid outbox event ID format: %w", err)
	}

	var errorMessageNull sql.NullString
	if errorMessage != "" {
		errorMessageNull = sql.NullString{String: errorMessage, Valid: true}
	}

	_, err = r.store.queries.MarkOutboxEventFailed(ctx, pgstore.MarkOutboxEventFailedParams{
		ID:           idUUID,
		ErrorMessage: errorMessageNull,
	})
	if err != nil {
		return fmt.Errorf("failed to mark outbox event as failed: %w", err)
	}
	return nil
}

// ============================================================================
// Conversion functions from pgstore models to domain models
// ============================================================================

func convertPaymentFromPGStore(p *pgstore.Payment) *domain.Payment {
	var stripePaymentIntentID *string
	if p.StripePaymentIntentID.Valid {
		stripePaymentIntentID = &p.StripePaymentIntentID.String
	}

	var stripeSessionID *string
	if p.StripeSessionID.Valid {
		stripeSessionID = &p.StripeSessionID.String
	}

	var metadata json.RawMessage
	if p.Metadata.Valid {
		metadata = json.RawMessage(p.Metadata.RawMessage)
	}

	return &domain.Payment{
		ID:                    p.ID,
		Amount:                int64(p.Amount),
		Currency:              p.Currency,
		Status:                p.Status,
		PaymentMethod:         p.PaymentMethod,
		CustomerID:            p.CustomerID,
		OrderID:               p.OrderID,
		Description:           p.Description.String,
		StripePaymentIntentID: stripePaymentIntentID,
		StripeSessionID:       stripeSessionID,
		Metadata:              metadata,
		CreatedAt:             p.CreatedAt,
		UpdatedAt:             p.UpdatedAt,
	}
}

func convertPlanFromPGStore(p *pgstore.Plan) *domain.Plan {
	var description string
	if p.Description.Valid {
		description = p.Description.String
	}

	var billingCycle string
	if p.BillingCycle.Valid {
		billingCycle = p.BillingCycle.String
	}

	var maxUsers int32
	if p.MaxUsers.Valid {
		maxUsers = p.MaxUsers.Int32
	}

	var usageLimits json.RawMessage
	if p.UsageLimits.Valid {
		usageLimits = json.RawMessage(p.UsageLimits.RawMessage)
	}

	var metadata json.RawMessage
	if p.Metadata.Valid {
		metadata = json.RawMessage(p.Metadata.RawMessage)
	}

	return &domain.Plan{
		ID:           uuid.MustParse(p.ID),
		Name:         p.Name,
		Description:  description,
		FeatureCodes: p.FeatureCodes,
		BillingCycle: billingCycle,
		PriceCents:   int64(p.PriceCents),
		Currency:     p.Currency,
		MaxUsers:     maxUsers,
		UsageLimits:  usageLimits,
		Metadata:     metadata,
		Active:       p.Active,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}

func convertEntitlementFromPGStore(e *pgstore.Entitlement) domain.Entitlement {
	var familyID *string
	if e.FamilyID.Valid {
		familyID = &e.FamilyID.String
	}

	var subscriptionID *string
	if e.SubscriptionID.Valid {
		subscriptionID = &e.SubscriptionID.String
	}

	var expiresAt *time.Time
	if e.ExpiresAt.Valid {
		expiresAt = &e.ExpiresAt.Time
	}

	var usageLimits json.RawMessage
	if e.UsageLimits.Valid {
		usageLimits = json.RawMessage(e.UsageLimits.RawMessage)
	}

	var metadata json.RawMessage
	if e.Metadata.Valid {
		metadata = json.RawMessage(e.Metadata.RawMessage)
	}

	return domain.Entitlement{
		ID:             e.ID,
		UserID:         e.UserID,
		FamilyID:       familyID,
		FeatureCode:    e.FeatureCode,
		PlanID:         uuid.MustParse(e.PlanID),
		SubscriptionID: subscriptionID,
		Status:         e.Status,
		GrantedAt:      e.GrantedAt,
		ExpiresAt:      expiresAt,
		UsageLimits:    usageLimits,
		Metadata:       metadata,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}
}

func convertWebhookEventFromPGStore(w *pgstore.WebhookEvent) *domain.WebhookEvent {
	var processedAt *time.Time
	if w.ProcessedAt.Valid {
		processedAt = &w.ProcessedAt.Time
	}

	return &domain.WebhookEvent{
		ID:          w.ID,
		EventID:     w.EventID,
		EventType:   w.EventType,
		Payload:     w.Payload,
		Signature:   w.Signature,
		Processed:   w.Processed,
		ProcessedAt: processedAt,
		CreatedAt:   w.CreatedAt,
	}
}

func convertOutboxEventFromPGStore(o *pgstore.Outbox) domain.OutboxEvent {
	var publishedAt *time.Time
	if o.PublishedAt.Valid {
		publishedAt = &o.PublishedAt.Time
	}

	var errorMessage *string
	if o.ErrorMessage.Valid {
		errorMessage = &o.ErrorMessage.String
	}

	return domain.OutboxEvent{
		ID:           o.ID.String(),
		EventType:    o.EventType,
		Payload:      o.Payload,
		Status:       o.Status,
		RetryCount:   int(o.RetryCount),
		CreatedAt:    o.CreatedAt,
		PublishedAt:  publishedAt,
		ErrorMessage: errorMessage,
	}
}
