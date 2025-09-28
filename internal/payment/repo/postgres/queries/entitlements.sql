-- name: CheckEntitlement :one
SELECT * FROM entitlements 
WHERE user_id = sqlc.arg(user_id) 
  AND feature_code = sqlc.arg(feature_code)
  AND status = 'active'
  AND (expires_at IS NULL OR expires_at > NOW())
ORDER BY granted_at DESC
LIMIT 1;

-- name: ListEntitlementsByUser :many
SELECT * FROM entitlements 
WHERE user_id = sqlc.arg(user_id)
ORDER BY granted_at DESC;

-- name: InsertEntitlement :one
INSERT INTO entitlements (
    user_id, family_id, feature_code, plan_id, subscription_id,
    status, granted_at, expires_at, usage_limits, metadata
) VALUES (
    sqlc.arg(user_id), sqlc.narg(family_id), sqlc.arg(feature_code),
    sqlc.arg(plan_id), sqlc.narg(subscription_id),
    sqlc.arg(status), sqlc.arg(granted_at), sqlc.narg(expires_at),
    sqlc.arg(usage_limits), sqlc.arg(metadata)
) RETURNING *;

-- name: UpdateEntitlementStatus :one
UPDATE entitlements 
SET status = sqlc.arg(status), updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: UpdateEntitlementExpiry :one
UPDATE entitlements 
SET expires_at = sqlc.narg(expires_at), updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: GetEntitlementByID :one
SELECT * FROM entitlements 
WHERE id = sqlc.arg(id);

-- name: ListExpiringEntitlements :many
SELECT * FROM entitlements 
WHERE expires_at IS NOT NULL 
  AND expires_at <= NOW() 
  AND status = 'active'
ORDER BY expires_at ASC;

-- name: GetEntitlementsBySubscriptionID :many
SELECT * FROM entitlements 
WHERE subscription_id = sqlc.arg(subscription_id)
ORDER BY granted_at DESC;

-- name: UpdateEntitlement :one
UPDATE entitlements 
SET user_id = sqlc.arg(user_id),
    family_id = sqlc.narg(family_id),
    feature_code = sqlc.arg(feature_code),
    plan_id = sqlc.arg(plan_id),
    subscription_id = sqlc.narg(subscription_id),
    status = sqlc.arg(status),
    granted_at = sqlc.arg(granted_at),
    expires_at = sqlc.narg(expires_at),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING *;
