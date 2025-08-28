-- name: GetPlanByID :one
SELECT * FROM plans 
WHERE id = sqlc.arg(id) AND active = true;

-- name: ListActivePlans :many
SELECT * FROM plans 
WHERE active = true 
ORDER BY created_at DESC;

-- name: InsertPlan :one
INSERT INTO plans (
    id, name, description, feature_codes, billing_cycle,
    price_cents, currency, max_users, usage_limits, metadata, active
) VALUES (
    sqlc.arg(id), sqlc.arg(name), sqlc.arg(description), 
    sqlc.arg(feature_codes), sqlc.arg(billing_cycle),
    sqlc.arg(price_cents), sqlc.arg(currency), sqlc.arg(max_users),
    sqlc.arg(usage_limits), sqlc.arg(metadata), sqlc.arg(active)
) RETURNING *;

-- name: UpdatePlanActive :one
UPDATE plans 
SET active = sqlc.arg(active), updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING *;
