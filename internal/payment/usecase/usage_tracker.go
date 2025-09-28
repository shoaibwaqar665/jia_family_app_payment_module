package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/jia-app/paymentservice/internal/payment/domain"
	"github.com/jia-app/paymentservice/internal/payment/repo"
	"github.com/jia-app/paymentservice/internal/shared/cache"
	"github.com/jia-app/paymentservice/internal/shared/events"
	"github.com/jia-app/paymentservice/internal/shared/log"
)

// UsageTracker handles usage tracking and quota management
type UsageTracker struct {
	usageRepo       repo.UsageRepository
	entitlementRepo repo.EntitlementRepository
	cache           *cache.Cache
	eventPublisher  events.UsagePublisher
}

// NewUsageTracker creates a new usage tracker
func NewUsageTracker(
	usageRepo repo.UsageRepository,
	entitlementRepo repo.EntitlementRepository,
	cache *cache.Cache,
	eventPublisher events.UsagePublisher,
) *UsageTracker {
	return &UsageTracker{
		usageRepo:       usageRepo,
		entitlementRepo: entitlementRepo,
		cache:           cache,
		eventPublisher:  eventPublisher,
	}
}

// TrackUsageRequest represents a request to track usage
type TrackUsageRequest struct {
	UserID       string                 `json:"user_id"`
	FamilyID     *string                `json:"family_id,omitempty"`
	FeatureCode  string                 `json:"feature_code"`
	ResourceType string                 `json:"resource_type"`
	ResourceSize int64                  `json:"resource_size"`
	Operation    string                 `json:"operation"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// TrackUsageResponse represents the response from tracking usage
type TrackUsageResponse struct {
	Allowed        bool                   `json:"allowed"`
	RemainingQuota int64                  `json:"remaining_quota"`
	QuotaLimit     int64                  `json:"quota_limit"`
	ResetTime      *time.Time             `json:"reset_time,omitempty"`
	UsageID        uuid.UUID              `json:"usage_id"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// CheckQuotaRequest represents a request to check quota
type CheckQuotaRequest struct {
	UserID       string  `json:"user_id"`
	FamilyID     *string `json:"family_id,omitempty"`
	FeatureCode  string  `json:"feature_code"`
	ResourceType string  `json:"resource_type"`
	ResourceSize int64   `json:"resource_size"`
}

// CheckQuotaResponse represents the response from checking quota
type CheckQuotaResponse struct {
	Allowed        bool       `json:"allowed"`
	RemainingQuota int64      `json:"remaining_quota"`
	QuotaLimit     int64      `json:"quota_limit"`
	ResetTime      *time.Time `json:"reset_time,omitempty"`
	Reason         string     `json:"reason,omitempty"`
}

// TrackUsage tracks usage for a user and checks against quota limits
func (ut *UsageTracker) TrackUsage(ctx context.Context, req TrackUsageRequest) (*TrackUsageResponse, error) {
	// Validate input
	if req.UserID == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.FeatureCode == "" {
		return nil, status.Error(codes.InvalidArgument, "feature_code is required")
	}
	if req.ResourceType == "" {
		return nil, status.Error(codes.InvalidArgument, "resource_type is required")
	}

	// Get user's entitlement for the feature
	entitlement, found, err := ut.entitlementRepo.Check(ctx, req.UserID, req.FeatureCode)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check entitlement: %v", err)
	}

	if !found {
		return nil, status.Errorf(codes.PermissionDenied, "no entitlement found for feature %s", req.FeatureCode)
	}

	// Parse usage limits from entitlement
	quotaLimit, resetPeriod, err := ut.parseUsageLimits(&entitlement, req.ResourceType)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to parse usage limits: %v", err)
	}

	// Check current usage
	currentUsage, err := ut.getCurrentUsage(ctx, req.UserID, req.FeatureCode, req.ResourceType, resetPeriod)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current usage: %v", err)
	}

	// Check if usage would exceed quota
	if currentUsage+req.ResourceSize > quotaLimit {
		return &TrackUsageResponse{
			Allowed:        false,
			RemainingQuota: max(0, quotaLimit-currentUsage),
			QuotaLimit:     quotaLimit,
			ResetTime:      ut.getResetTime(resetPeriod),
			Metadata:       req.Metadata,
		}, nil
	}

	// Record the usage
	usageID := uuid.New()
	usage := domain.Usage{
		ID:           usageID,
		UserID:       req.UserID,
		FamilyID:     req.FamilyID,
		FeatureCode:  req.FeatureCode,
		ResourceType: req.ResourceType,
		ResourceSize: req.ResourceSize,
		Operation:    req.Operation,
		Metadata:     req.Metadata,
		CreatedAt:    time.Now(),
	}

	if err := ut.usageRepo.Create(ctx, usage); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to record usage: %v", err)
	}

	// Update cache
	if ut.cache != nil {
		cacheKey := fmt.Sprintf("usage:%s:%s:%s", req.UserID, req.FeatureCode, req.ResourceType)
		newUsage := currentUsage + req.ResourceSize
		ut.cache.Set(ctx, cacheKey, newUsage, resetPeriod)
	}

	// Publish usage event
	if ut.eventPublisher != nil {
		if err := ut.eventPublisher.PublishUsageTracked(ctx, &usage); err != nil {
			log.L(ctx).Warn("Failed to publish usage tracked event", zap.Error(err))
		}
	}

	// Calculate remaining quota
	remainingQuota := quotaLimit - (currentUsage + req.ResourceSize)

	log.Info(ctx, "Usage tracked successfully",
		zap.String("user_id", req.UserID),
		zap.String("feature_code", req.FeatureCode),
		zap.String("resource_type", req.ResourceType),
		zap.Int64("resource_size", req.ResourceSize),
		zap.Int64("remaining_quota", remainingQuota),
		zap.Int64("quota_limit", quotaLimit))

	return &TrackUsageResponse{
		Allowed:        true,
		RemainingQuota: remainingQuota,
		QuotaLimit:     quotaLimit,
		ResetTime:      ut.getResetTime(resetPeriod),
		UsageID:        usageID,
		Metadata:       req.Metadata,
	}, nil
}

// CheckQuota checks if a user has quota available for a resource
func (ut *UsageTracker) CheckQuota(ctx context.Context, req CheckQuotaRequest) (*CheckQuotaResponse, error) {
	// Validate input
	if req.UserID == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.FeatureCode == "" {
		return nil, status.Error(codes.InvalidArgument, "feature_code is required")
	}
	if req.ResourceType == "" {
		return nil, status.Error(codes.InvalidArgument, "resource_type is required")
	}

	// Get user's entitlement for the feature
	entitlement, found, err := ut.entitlementRepo.Check(ctx, req.UserID, req.FeatureCode)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check entitlement: %v", err)
	}

	if !found {
		return &CheckQuotaResponse{
			Allowed: false,
			Reason:  "No entitlement found for feature",
		}, nil
	}

	// Parse usage limits from entitlement
	quotaLimit, resetPeriod, err := ut.parseUsageLimits(&entitlement, req.ResourceType)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to parse usage limits: %v", err)
	}

	// Check current usage
	currentUsage, err := ut.getCurrentUsage(ctx, req.UserID, req.FeatureCode, req.ResourceType, resetPeriod)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current usage: %v", err)
	}

	// Check if usage would exceed quota
	allowed := currentUsage+req.ResourceSize <= quotaLimit
	remainingQuota := max(0, quotaLimit-currentUsage)

	return &CheckQuotaResponse{
		Allowed:        allowed,
		RemainingQuota: remainingQuota,
		QuotaLimit:     quotaLimit,
		ResetTime:      ut.getResetTime(resetPeriod),
		Reason:         ut.getReason(allowed, remainingQuota),
	}, nil
}

// GetUsageStats returns usage statistics for a user and feature
func (ut *UsageTracker) GetUsageStats(ctx context.Context, userID, featureCode, resourceType string) (*UsageStats, error) {
	// Get user's entitlement
	entitlement, found, err := ut.entitlementRepo.Check(ctx, userID, featureCode)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check entitlement: %v", err)
	}

	if !found {
		return nil, status.Errorf(codes.NotFound, "no entitlement found for feature %s", featureCode)
	}

	// Parse usage limits
	quotaLimit, resetPeriod, err := ut.parseUsageLimits(&entitlement, resourceType)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to parse usage limits: %v", err)
	}

	// Get current usage
	currentUsage, err := ut.getCurrentUsage(ctx, userID, featureCode, resourceType, resetPeriod)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current usage: %v", err)
	}

	// Get usage history
	usageHistory, err := ut.usageRepo.GetUsageHistory(ctx, userID, featureCode, resourceType, resetPeriod)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get usage history: %v", err)
	}

	return &UsageStats{
		UserID:         userID,
		FeatureCode:    featureCode,
		ResourceType:   resourceType,
		CurrentUsage:   currentUsage,
		QuotaLimit:     quotaLimit,
		RemainingQuota: max(0, quotaLimit-currentUsage),
		ResetTime:      ut.getResetTime(resetPeriod),
		UsageHistory:   usageHistory,
		CreatedAt:      time.Now(),
	}, nil
}

// ResetUsage resets usage for a user and feature (admin function)
func (ut *UsageTracker) ResetUsage(ctx context.Context, userID, featureCode, resourceType string) error {
	// Delete usage records
	if err := ut.usageRepo.DeleteUsage(ctx, userID, featureCode, resourceType); err != nil {
		return status.Errorf(codes.Internal, "failed to reset usage: %v", err)
	}

	// Clear cache
	if ut.cache != nil {
		cacheKey := fmt.Sprintf("usage:%s:%s:%s", userID, featureCode, resourceType)
		ut.cache.Delete(ctx, cacheKey)
	}

	log.Info(ctx, "Usage reset successfully",
		zap.String("user_id", userID),
		zap.String("feature_code", featureCode),
		zap.String("resource_type", resourceType))

	return nil
}

// Helper methods

// parseUsageLimits parses usage limits from entitlement metadata
func (ut *UsageTracker) parseUsageLimits(entitlement *domain.Entitlement, resourceType string) (int64, time.Duration, error) {
	// Parse usage limits from entitlement metadata
	var usageLimits map[string]interface{}
	if err := json.Unmarshal(entitlement.UsageLimits, &usageLimits); err != nil {
		return 0, 0, fmt.Errorf("failed to parse usage limits: %w", err)
	}

	// Get limits for the specific resource type
	resourceLimits, ok := usageLimits[resourceType].(map[string]interface{})
	if !ok {
		return 0, 0, fmt.Errorf("no limits found for resource type %s", resourceType)
	}

	// Parse quota limit
	quotaLimit, ok := resourceLimits["quota_limit"].(float64)
	if !ok {
		return 0, 0, fmt.Errorf("invalid quota_limit for resource type %s", resourceType)
	}

	// Parse reset period
	resetPeriodStr, ok := resourceLimits["reset_period"].(string)
	if !ok {
		return 0, 0, fmt.Errorf("invalid reset_period for resource type %s", resourceType)
	}

	resetPeriod, err := time.ParseDuration(resetPeriodStr)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid reset_period format: %w", err)
	}

	return int64(quotaLimit), resetPeriod, nil
}

// getCurrentUsage gets the current usage for a user, feature, and resource type
func (ut *UsageTracker) getCurrentUsage(ctx context.Context, userID, featureCode, resourceType string, resetPeriod time.Duration) (int64, error) {
	// Try cache first
	if ut.cache != nil {
		cacheKey := fmt.Sprintf("usage:%s:%s:%s", userID, featureCode, resourceType)
		var cachedUsage int64
		if err := ut.cache.Get(ctx, cacheKey, &cachedUsage); err == nil {
			return cachedUsage, nil
		}
	}

	// Get from database
	usage, err := ut.usageRepo.GetCurrentUsage(ctx, userID, featureCode, resourceType, resetPeriod)
	if err != nil {
		return 0, err
	}

	// Cache the result
	if ut.cache != nil {
		cacheKey := fmt.Sprintf("usage:%s:%s:%s", userID, featureCode, resourceType)
		ut.cache.Set(ctx, cacheKey, usage, resetPeriod)
	}

	return usage, nil
}

// getResetTime calculates when the usage quota will reset
func (ut *UsageTracker) getResetTime(resetPeriod time.Duration) *time.Time {
	now := time.Now()
	resetTime := now.Add(resetPeriod)
	return &resetTime
}

// getReason returns a human-readable reason for quota check result
func (ut *UsageTracker) getReason(allowed bool, remainingQuota int64) string {
	if allowed {
		return "Quota available"
	}
	return "Quota exceeded"
}

// Helper function for max
func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// UsageStats represents usage statistics
type UsageStats struct {
	UserID         string         `json:"user_id"`
	FeatureCode    string         `json:"feature_code"`
	ResourceType   string         `json:"resource_type"`
	CurrentUsage   int64          `json:"current_usage"`
	QuotaLimit     int64          `json:"quota_limit"`
	RemainingQuota int64          `json:"remaining_quota"`
	ResetTime      *time.Time     `json:"reset_time,omitempty"`
	UsageHistory   []domain.Usage `json:"usage_history"`
	CreatedAt      time.Time      `json:"created_at"`
}
