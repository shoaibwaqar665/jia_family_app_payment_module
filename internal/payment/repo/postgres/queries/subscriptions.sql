-- name: CreateSubscription :one
INSERT INTO subscriptions (
    user_id, family_id, plan_id, status, current_period_start, 
    current_period_end, cancel_at_period_end, external_subscription_id, metadata
) VALUES (
    sqlc.arg(user_id), sqlc.narg(family_id), sqlc.arg(plan_id), 
    sqlc.arg(status), sqlc.arg(current_period_start), sqlc.arg(current_period_end),
    sqlc.arg(cancel_at_period_end), sqlc.narg(external_subscription_id), sqlc.arg(metadata)
) RETURNING *;

-- name: GetSubscriptionByID :one
SELECT * FROM subscriptions WHERE id = sqlc.arg(id);

-- name: GetSubscriptionByExternalID :one
SELECT * FROM subscriptions WHERE external_subscription_id = sqlc.arg(external_id);

-- name: GetSubscriptionsByUserID :many
SELECT * FROM subscriptions WHERE user_id = sqlc.arg(user_id) ORDER BY created_at DESC;

-- name: GetSubscriptionsByStatus :many
SELECT * FROM subscriptions WHERE status = sqlc.arg(status) ORDER BY created_at DESC;

-- name: UpdateSubscription :one
UPDATE subscriptions SET
    status = sqlc.arg(status),
    current_period_start = sqlc.arg(current_period_start),
    current_period_end = sqlc.arg(current_period_end),
    cancel_at_period_end = sqlc.arg(cancel_at_period_end),
    cancelled_at = sqlc.narg(cancelled_at),
    metadata = sqlc.arg(metadata),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteSubscription :exec
DELETE FROM subscriptions WHERE id = sqlc.arg(id);

-- name: GetExpiringSubscriptions :many
SELECT * FROM subscriptions 
WHERE current_period_end <= sqlc.arg(before_date) 
  AND status = 'active'
ORDER BY current_period_end ASC;

-- name: GetActiveSubscriptions :many
SELECT * FROM subscriptions WHERE status = 'active' ORDER BY created_at DESC;

-- name: GetSubscriptionsByPlan :many
SELECT * FROM subscriptions WHERE plan_id = sqlc.arg(plan_id) ORDER BY created_at DESC;

-- name: UpdateSubscriptionStatus :one
UPDATE subscriptions SET
    status = sqlc.arg(status),
    cancelled_at = sqlc.narg(cancelled_at),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: RenewSubscription :one
UPDATE subscriptions SET
    current_period_start = sqlc.arg(new_period_start),
    current_period_end = sqlc.arg(new_period_end),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING *;
