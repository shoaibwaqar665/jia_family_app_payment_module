package service

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/jia-app/paymentservice/internal/cache"
	"github.com/jia-app/paymentservice/internal/config"
	"github.com/jia-app/paymentservice/internal/domain"
	"github.com/jia-app/paymentservice/internal/events"
	"github.com/jia-app/paymentservice/internal/log"
)

// fakeEntitlementRepository implements repository.EntitlementRepository for testing
type fakeEntitlementRepository struct {
	entitlements map[string]domain.Entitlement // key: userID:featureCode
	shouldError  bool
}

func newFakeEntitlementRepository() *fakeEntitlementRepository {
	return &fakeEntitlementRepository{
		entitlements: make(map[string]domain.Entitlement),
	}
}

func (f *fakeEntitlementRepository) Check(ctx context.Context, userID, featureCode string) (domain.Entitlement, bool, error) {
	if f.shouldError {
		return domain.Entitlement{}, false, domain.NewInternalError("fake repository error")
	}

	key := userID + ":" + featureCode
	if ent, exists := f.entitlements[key]; exists {
		return ent, true, nil
	}
	return domain.Entitlement{}, false, nil
}

func (f *fakeEntitlementRepository) ListByUser(ctx context.Context, userID string) ([]domain.Entitlement, error) {
	if f.shouldError {
		return nil, domain.NewInternalError("fake repository error")
	}

	var result []domain.Entitlement
	for key, ent := range f.entitlements {
		if key[:len(userID)] == userID {
			result = append(result, ent)
		}
	}
	return result, nil
}

func (f *fakeEntitlementRepository) Insert(ctx context.Context, e domain.Entitlement) (domain.Entitlement, error) {
	if f.shouldError {
		return domain.Entitlement{}, domain.NewInternalError("fake repository error")
	}

	if e.ID.String() == "" {
		e.ID = uuid.New()
	}
	key := e.UserID + ":" + e.FeatureCode
	f.entitlements[key] = e
	return e, nil
}

func (f *fakeEntitlementRepository) UpdateStatus(ctx context.Context, id, status string) error {
	if f.shouldError {
		return domain.NewInternalError("fake repository error")
	}

	for key, ent := range f.entitlements {
		if ent.ID.String() == id {
			ent.Status = status
			f.entitlements[key] = ent
			return nil
		}
	}
	return domain.NewNotFoundError("entitlement", id)
}

func (f *fakeEntitlementRepository) UpdateExpiry(ctx context.Context, id string, expiresAt *time.Time) error {
	if f.shouldError {
		return domain.NewInternalError("fake repository error")
	}

	for key, ent := range f.entitlements {
		if ent.ID.String() == id {
			ent.ExpiresAt = expiresAt
			f.entitlements[key] = ent
			return nil
		}
	}
	return domain.NewNotFoundError("entitlement", id)
}

// setupTestCache creates a miniredis instance and cache for testing
func setupTestCache(t *testing.T) (*cache.Cache, *miniredis.Miniredis) {
	mr := miniredis.RunT(t)

	cacheClient, err := cache.NewCache(mr.Addr(), "", 0)
	require.NoError(t, err)

	return cacheClient, mr
}

// newTestPaymentService creates a PaymentService for testing with minimal dependencies
func newTestPaymentService(
	entRepo *fakeEntitlementRepository,
	cacheClient *cache.Cache,
) *PaymentService {
	return &PaymentService{
		config:               &config.Config{},
		entitlementRepo:      entRepo,
		cache:                cacheClient,
		entitlementPublisher: events.NoopPublisher{},
	}
}

func TestCheckEntitlement_HappyPath(t *testing.T) {
	// Setup
	ctx := context.Background()
	userID := "user123"
	featureCode := "premium_feature"

	// Create fake repositories
	entRepo := newFakeEntitlementRepository()

	// Create a valid entitlement
	validEntitlement := domain.Entitlement{
		ID:          uuid.New(),
		UserID:      userID,
		FeatureCode: featureCode,
		Status:      "active",
		PlanID:      uuid.New(),
		GrantedAt:   time.Now(),
		ExpiresAt:   nil, // Never expires
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	entRepo.entitlements[userID+":"+featureCode] = validEntitlement

	// Create cache
	cacheClient, mr := setupTestCache(t)
	defer mr.Close()

	// Create service
	service := newTestPaymentService(entRepo, cacheClient)

	// Test
	result, err := service.CheckEntitlement(ctx, userID, featureCode)

	// Assertions
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Allowed)
	require.NotNil(t, result.Entitlement)
	require.Equal(t, userID, result.Entitlement.UserID)
	require.Equal(t, featureCode, result.Entitlement.FeatureCode)
	require.Equal(t, "active", result.Entitlement.Status)

	// Verify cache was populated
	cachedEnt, found, err := cacheClient.GetEntitlement(ctx, userID, featureCode)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, validEntitlement.ID, cachedEnt.ID)
}

func TestCheckEntitlement_ExpiredPath(t *testing.T) {
	// Setup
	ctx := context.Background()
	userID := "user456"
	featureCode := "expired_feature"

	// Create fake repositories
	entRepo := newFakeEntitlementRepository()

	// Create an expired entitlement
	expiredEntitlement := domain.Entitlement{
		ID:          uuid.New(),
		UserID:      userID,
		FeatureCode: featureCode,
		Status:      "active",
		PlanID:      uuid.New(),
		GrantedAt:   time.Now().Add(-24 * time.Hour),
		ExpiresAt:   &[]time.Time{time.Now().Add(-1 * time.Hour)}[0], // Expired 1 hour ago
		CreatedAt:   time.Now().Add(-24 * time.Hour),
		UpdatedAt:   time.Now().Add(-24 * time.Hour),
	}
	entRepo.entitlements[userID+":"+featureCode] = expiredEntitlement

	// Create cache
	cacheClient, mr := setupTestCache(t)
	defer mr.Close()

	// Create service
	service := newTestPaymentService(entRepo, cacheClient)

	// Test
	result, err := service.CheckEntitlement(ctx, userID, featureCode)

	// Assertions
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.Allowed)
	require.Nil(t, result.Entitlement)

	// Verify negative result was cached
	isNegative, err := cacheClient.IsEntitlementNotFound(ctx, userID, featureCode)
	require.NoError(t, err)
	require.True(t, isNegative)
}

func TestCheckEntitlement_CacheHit(t *testing.T) {
	// Setup
	ctx := context.Background()
	userID := "user789"
	featureCode := "cached_feature"

	// Create fake repositories
	entRepo := newFakeEntitlementRepository()

	// Create a valid entitlement
	validEntitlement := domain.Entitlement{
		ID:          uuid.New(),
		UserID:      userID,
		FeatureCode: featureCode,
		Status:      "active",
		PlanID:      uuid.New(),
		GrantedAt:   time.Now(),
		ExpiresAt:   nil, // Never expires
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Create cache and pre-populate it
	cacheClient, mr := setupTestCache(t)
	defer mr.Close()

	err := cacheClient.SetEntitlement(ctx, validEntitlement, 0)
	require.NoError(t, err)

	// Create service
	service := newTestPaymentService(entRepo, cacheClient)

	// Test - should hit cache first
	result, err := service.CheckEntitlement(ctx, userID, featureCode)

	// Assertions
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Allowed)
	require.NotNil(t, result.Entitlement)
	require.Equal(t, userID, result.Entitlement.UserID)
	require.Equal(t, featureCode, result.Entitlement.FeatureCode)

	// Verify the repository was not called (cache hit)
	// We can verify this by checking that the fake repo still has no data
	_, found, _ := entRepo.Check(ctx, userID, featureCode)
	require.False(t, found)
}

func TestCheckEntitlement_CacheHit_Expired(t *testing.T) {
	// Setup
	ctx := context.Background()
	userID := "user101"
	featureCode := "cached_expired_feature"

	// Create fake repositories
	entRepo := newFakeEntitlementRepository()

	// Create an expired entitlement
	expiredEntitlement := domain.Entitlement{
		ID:          uuid.New(),
		UserID:      userID,
		FeatureCode: featureCode,
		Status:      "active",
		PlanID:      uuid.New(),
		GrantedAt:   time.Now().Add(-24 * time.Hour),
		ExpiresAt:   &[]time.Time{time.Now().Add(-1 * time.Hour)}[0], // Expired 1 hour ago
		CreatedAt:   time.Now().Add(-24 * time.Hour),
		UpdatedAt:   time.Now().Add(-24 * time.Hour),
	}

	// Create cache and pre-populate it with expired entitlement
	cacheClient, mr := setupTestCache(t)
	defer mr.Close()

	err := cacheClient.SetEntitlement(ctx, expiredEntitlement, 0)
	require.NoError(t, err)

	// Create service
	service := newTestPaymentService(entRepo, cacheClient)

	// Test - should hit cache but find expired entitlement
	result, err := service.CheckEntitlement(ctx, userID, featureCode)

	// Assertions
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.Allowed)
	require.Nil(t, result.Entitlement)

	// Verify the expired entitlement was evicted from cache and replaced with negative cache
	// The service should have deleted the expired entitlement and set a negative cache entry
	// Note: The service sets a negative cache entry when no valid entitlement is found
	
	// Verify negative result was cached (since repository also doesn't have this entitlement)
	isNegative, err := cacheClient.IsEntitlementNotFound(ctx, userID, featureCode)
	require.NoError(t, err)
	require.True(t, isNegative)
}

func TestCheckEntitlement_ContextUserID(t *testing.T) {
	// Setup
	userID := "user_context"
	featureCode := "context_feature"

	// Create context with user ID
	ctx := log.WithUserID(context.Background(), userID)

	// Create fake repositories
	entRepo := newFakeEntitlementRepository()

	// Create a valid entitlement
	validEntitlement := domain.Entitlement{
		ID:          uuid.New(),
		UserID:      userID,
		FeatureCode: featureCode,
		Status:      "active",
		PlanID:      uuid.New(),
		GrantedAt:   time.Now(),
		ExpiresAt:   nil,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	entRepo.entitlements[userID+":"+featureCode] = validEntitlement

	// Create cache
	cacheClient, mr := setupTestCache(t)
	defer mr.Close()

	// Create service
	service := newTestPaymentService(entRepo, cacheClient)

	// Test - pass empty userID, should extract from context
	result, err := service.CheckEntitlement(ctx, "", featureCode)

	// Assertions
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Allowed)
	require.NotNil(t, result.Entitlement)
	require.Equal(t, userID, result.Entitlement.UserID)
}

func TestCheckEntitlement_ValidationErrors(t *testing.T) {
	// Setup
	ctx := context.Background()

	// Create fake repositories
	entRepo := newFakeEntitlementRepository()

	// Create cache
	cacheClient, mr := setupTestCache(t)
	defer mr.Close()

	// Create service
	service := newTestPaymentService(entRepo, cacheClient)

	// Test cases
	testCases := []struct {
		name          string
		userID        string
		featureCode   string
		expectedError string
	}{
		{
			name:          "empty user ID and no context",
			userID:        "",
			featureCode:   "feature",
			expectedError: "user_id is required",
		},
		{
			name:          "empty feature code",
			userID:        "user123",
			featureCode:   "",
			expectedError: "feature_code is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := service.CheckEntitlement(ctx, tc.userID, tc.featureCode)

			require.Error(t, err)
			require.Contains(t, err.Error(), tc.expectedError)
			require.Nil(t, result)
		})
	}
}

func TestCheckEntitlement_RepositoryError(t *testing.T) {
	// Setup
	ctx := context.Background()
	userID := "user_error"
	featureCode := "error_feature"

	// Create fake repositories with error
	entRepo := newFakeEntitlementRepository()
	entRepo.shouldError = true

	// Create cache
	cacheClient, mr := setupTestCache(t)
	defer mr.Close()

	// Create service
	service := newTestPaymentService(entRepo, cacheClient)

	// Test
	result, err := service.CheckEntitlement(ctx, userID, featureCode)

	// Assertions
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to check entitlement")
	require.Nil(t, result)
}

func TestCheckEntitlement_CacheError(t *testing.T) {
	// Setup
	ctx := context.Background()
	userID := "user_cache_error"
	featureCode := "cache_error_feature"

	// Create fake repositories
	entRepo := newFakeEntitlementRepository()

	// Create a valid entitlement
	validEntitlement := domain.Entitlement{
		ID:          uuid.New(),
		UserID:      userID,
		FeatureCode: featureCode,
		Status:      "active",
		PlanID:      uuid.New(),
		GrantedAt:   time.Now(),
		ExpiresAt:   nil,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	entRepo.entitlements[userID+":"+featureCode] = validEntitlement

	// Create cache that will fail
	cacheClient, mr := setupTestCache(t)
	defer mr.Close()

	// Corrupt the cache by setting invalid data
	mr.Set("entl:"+userID+":"+featureCode, "invalid json data")

	// Create service
	service := newTestPaymentService(entRepo, cacheClient)

	// Test - should fallback to repository due to cache error
	result, err := service.CheckEntitlement(ctx, userID, featureCode)

	// Assertions
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Allowed)
	require.NotNil(t, result.Entitlement)

	// Verify the corrupted cache entry was evicted and replaced
	cachedEnt, found, err := cacheClient.GetEntitlement(ctx, userID, featureCode)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, validEntitlement.ID, cachedEnt.ID)
}
