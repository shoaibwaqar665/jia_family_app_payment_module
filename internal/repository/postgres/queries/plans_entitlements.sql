-- Plans queries
-- name: CreatePlan :one
INSERT INTO plans (
    id, name, description, feature_codes, billing_cycle, 
    price_cents, currency, max_users, usage_limits, metadata, active
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: GetPlan :one
SELECT * FROM plans WHERE id = $1 AND active = true;

-- name: ListPlans :many
SELECT * FROM plans WHERE active = true ORDER BY created_at DESC;

-- name: UpdatePlan :one
UPDATE plans SET 
    name = $2,
    description = $3,
    feature_codes = $4,
    billing_cycle = $5,
    price_cents = $6,
    currency = $7,
    max_users = $8,
    usage_limits = $9,
    metadata = $10,
    active = $11
WHERE id = $1 RETURNING *;

-- name: DeletePlan :exec
UPDATE plans SET active = false WHERE id = $1;

-- Entitlements queries
-- name: CreateEntitlement :one
INSERT INTO entitlements (
    user_id, family_id, feature_code, plan_id, subscription_id,
    status, granted_at, expires_at, usage_limits, metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetEntitlement :one
SELECT * FROM entitlements WHERE id = $1;

-- name: GetUserEntitlements :many
SELECT * FROM entitlements 
WHERE user_id = $1 AND status = 'active'
ORDER BY granted_at DESC;

-- name: GetUserFeatureEntitlement :one
SELECT * FROM entitlements 
WHERE user_id = $1 AND feature_code = $2 AND status = 'active'
ORDER BY granted_at DESC LIMIT 1;

-- name: GetFamilyEntitlements :many
SELECT * FROM entitlements 
WHERE family_id = $1 AND status = 'active'
ORDER BY granted_at DESC;

-- name: GetFamilyFeatureEntitlement :one
SELECT * FROM entitlements 
WHERE family_id = $1 AND feature_code = $2 AND status = 'active'
ORDER BY granted_at DESC LIMIT 1;

-- name: UpdateEntitlementStatus :one
UPDATE entitlements SET status = $2 WHERE id = $1 RETURNING *;

-- name: UpdateEntitlementExpiry :one
UPDATE entitlements SET expires_at = $2 WHERE id = $1 RETURNING *;

-- name: ListExpiringEntitlements :many
SELECT * FROM entitlements 
WHERE expires_at IS NOT NULL 
AND expires_at <= $1 
AND status = 'active'
ORDER BY expires_at ASC;
