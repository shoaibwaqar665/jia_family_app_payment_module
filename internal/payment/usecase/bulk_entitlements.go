package usecase

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/jia-app/paymentservice/internal/payment/domain"
	"github.com/jia-app/paymentservice/internal/payment/repo"
	"github.com/jia-app/paymentservice/internal/shared/cache"
	"github.com/jia-app/paymentservice/internal/shared/log"
)

// BulkEntitlementUseCase provides business logic for bulk entitlement operations
type BulkEntitlementUseCase struct {
	entitlementRepo repo.EntitlementRepository
	cache           *cache.Cache
}

// NewBulkEntitlementUseCase creates a new bulk entitlement use case
func NewBulkEntitlementUseCase(
	entitlementRepo repo.EntitlementRepository,
	cache *cache.Cache,
) *BulkEntitlementUseCase {
	return &BulkEntitlementUseCase{
		entitlementRepo: entitlementRepo,
		cache:           cache,
	}
}

// BulkCheckRequest represents a request for bulk entitlement checking
type BulkCheckRequest struct {
	UserID string          `json:"user_id"`
	Checks []BulkCheckItem `json:"checks"`
}

// BulkCheckItem represents a single entitlement check
type BulkCheckItem struct {
	FeatureCode  string                 `json:"feature_code"`
	Operation    string                 `json:"operation,omitempty"`
	ResourceSize int64                  `json:"resource_size,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// BulkCheckResponse represents the response from bulk entitlement checking
type BulkCheckResponse struct {
	Results []BulkCheckResult `json:"results"`
	Summary BulkCheckSummary  `json:"summary"`
}

// BulkCheckResult represents the result of a single entitlement check
type BulkCheckResult struct {
	FeatureCode string                 `json:"feature_code"`
	Authorized  bool                   `json:"authorized"`
	Entitlement *domain.Entitlement    `json:"entitlement,omitempty"`
	Reason      string                 `json:"reason,omitempty"`
	UpgradeURL  string                 `json:"upgrade_url,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// BulkCheckSummary provides summary statistics for bulk checks
type BulkCheckSummary struct {
	TotalChecks    int   `json:"total_checks"`
	Authorized     int   `json:"authorized"`
	NotAuthorized  int   `json:"not_authorized"`
	CacheHits      int   `json:"cache_hits"`
	CacheMisses    int   `json:"cache_misses"`
	ProcessingTime int64 `json:"processing_time_ms"`
}

// BulkCheckEntitlements performs bulk entitlement checking with parallel processing
func (uc *BulkEntitlementUseCase) BulkCheckEntitlements(ctx context.Context, req BulkCheckRequest) (*BulkCheckResponse, error) {
	startTime := time.Now()

	// Validate input
	if req.UserID == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	if len(req.Checks) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one check is required")
	}

	if len(req.Checks) > 100 {
		return nil, status.Error(codes.InvalidArgument, "maximum 100 checks allowed per request")
	}

	// Process checks in parallel with controlled concurrency
	results := make([]BulkCheckResult, len(req.Checks))
	var wg sync.WaitGroup
	var mu sync.Mutex
	cacheHits := 0
	cacheMisses := 0

	// Use a semaphore to limit concurrent goroutines
	semaphore := make(chan struct{}, 10) // Max 10 concurrent checks

	for i, check := range req.Checks {
		wg.Add(1)
		go func(index int, checkItem BulkCheckItem) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result := uc.checkSingleEntitlement(ctx, req.UserID, checkItem)

			mu.Lock()
			results[index] = result
			if result.Entitlement != nil {
				cacheHits++
			} else {
				cacheMisses++
			}
			mu.Unlock()
		}(i, check)
	}

	wg.Wait()

	// Calculate summary
	authorized := 0
	notAuthorized := 0
	for _, result := range results {
		if result.Authorized {
			authorized++
		} else {
			notAuthorized++
		}
	}

	processingTime := time.Since(startTime).Milliseconds()

	response := &BulkCheckResponse{
		Results: results,
		Summary: BulkCheckSummary{
			TotalChecks:    len(req.Checks),
			Authorized:     authorized,
			NotAuthorized:  notAuthorized,
			CacheHits:      cacheHits,
			CacheMisses:    cacheMisses,
			ProcessingTime: processingTime,
		},
	}

	log.Info(ctx, "Bulk entitlement check completed",
		zap.String("user_id", req.UserID),
		zap.Int("total_checks", len(req.Checks)),
		zap.Int("authorized", authorized),
		zap.Int("not_authorized", notAuthorized),
		zap.Int64("processing_time_ms", processingTime))

	return response, nil
}

// checkSingleEntitlement checks a single entitlement
func (uc *BulkEntitlementUseCase) checkSingleEntitlement(ctx context.Context, userID string, check BulkCheckItem) BulkCheckResult {
	// Validate feature code
	if check.FeatureCode == "" {
		return BulkCheckResult{
			FeatureCode: check.FeatureCode,
			Authorized:  false,
			Reason:      "feature_code is required",
		}
	}

	// Try cache first
	if uc.cache != nil {
		cachedEnt, found, err := uc.cache.GetEntitlement(ctx, userID, check.FeatureCode)
		if err != nil {
			log.Warn(ctx, "Failed to get entitlement from cache",
				zap.Error(err), zap.String("user_id", userID), zap.String("feature_code", check.FeatureCode))
		} else if found {
			// Check if it's a negative cache result
			if isNegative, err := uc.cache.IsEntitlementNotFound(ctx, userID, check.FeatureCode); err == nil && isNegative {
				return BulkCheckResult{
					FeatureCode: check.FeatureCode,
					Authorized:  false,
					Reason:      "No active entitlement found",
					UpgradeURL:  uc.generateUpgradeURL(check.FeatureCode),
				}
			}

			// Validate cached entitlement is still active and not expired
			if uc.isValidEntitlement(cachedEnt) {
				return BulkCheckResult{
					FeatureCode: check.FeatureCode,
					Authorized:  true,
					Entitlement: cachedEnt,
					Metadata:    check.Metadata,
				}
			}

			// Cached entitlement is invalid, evict from cache and fallback to repo
			uc.cache.DeleteEntitlement(ctx, userID, check.FeatureCode)
		}
	}

	// Fallback to repository
	entitlement, found, err := uc.entitlementRepo.Check(ctx, userID, check.FeatureCode)
	if err != nil {
		log.Error(ctx, "Failed to check entitlement",
			zap.Error(err), zap.String("user_id", userID), zap.String("feature_code", check.FeatureCode))
		return BulkCheckResult{
			FeatureCode: check.FeatureCode,
			Authorized:  false,
			Reason:      "Internal error checking entitlement",
		}
	}

	if !found {
		// Cache negative result
		if uc.cache != nil {
			uc.cache.SetEntitlementNotFound(ctx, userID, check.FeatureCode)
		}
		return BulkCheckResult{
			FeatureCode: check.FeatureCode,
			Authorized:  false,
			Reason:      "No active entitlement found",
			UpgradeURL:  uc.generateUpgradeURL(check.FeatureCode),
		}
	}

	// Validate entitlement is active and not expired
	if !uc.isValidEntitlement(&entitlement) {
		// Cache negative result for invalid entitlements
		if uc.cache != nil {
			uc.cache.SetEntitlementNotFound(ctx, userID, check.FeatureCode)
		}
		return BulkCheckResult{
			FeatureCode: check.FeatureCode,
			Authorized:  false,
			Reason:      "Entitlement expired or inactive",
			UpgradeURL:  uc.generateUpgradeURL(check.FeatureCode),
		}
	}

	// Cache the valid entitlement
	if uc.cache != nil {
		uc.cache.SetEntitlement(ctx, entitlement, 300*time.Second) // 5 minutes
	}

	return BulkCheckResult{
		FeatureCode: check.FeatureCode,
		Authorized:  true,
		Entitlement: &entitlement,
		Metadata:    check.Metadata,
	}
}

// isValidEntitlement checks if an entitlement is valid
func (uc *BulkEntitlementUseCase) isValidEntitlement(ent *domain.Entitlement) bool {
	if ent == nil {
		return false
	}

	// Check if status is active
	if ent.Status != "active" {
		return false
	}

	// Check if not expired
	if ent.ExpiresAt != nil && ent.ExpiresAt.Before(time.Now()) {
		return false
	}

	return true
}

// generateUpgradeURL generates an upgrade URL for a feature
func (uc *BulkEntitlementUseCase) generateUpgradeURL(featureCode string) string {
	// This would typically be generated based on the feature code and current pricing
	return fmt.Sprintf("https://pay.jia.app/checkout/%s", featureCode)
}
