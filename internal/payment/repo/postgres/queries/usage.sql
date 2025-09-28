-- name: CreateUsage :exec
INSERT INTO usage (
    user_id, family_id, feature_code, resource_type, resource_size, operation, metadata
) VALUES (
    sqlc.arg(user_id), sqlc.narg(family_id), sqlc.arg(feature_code), 
    sqlc.arg(resource_type), sqlc.arg(resource_size), sqlc.narg(operation), sqlc.arg(metadata)
);

-- name: GetCurrentUsage :one
SELECT COALESCE(SUM(resource_size), 0) as total_usage
FROM usage 
WHERE user_id = sqlc.arg(user_id) 
  AND feature_code = sqlc.arg(feature_code) 
  AND resource_type = sqlc.arg(resource_type)
  AND created_at >= sqlc.arg(since);

-- name: GetUsageHistory :many
SELECT * FROM usage 
WHERE user_id = sqlc.arg(user_id) 
  AND feature_code = sqlc.arg(feature_code) 
  AND resource_type = sqlc.arg(resource_type)
  AND created_at >= sqlc.arg(since)
ORDER BY created_at DESC;

-- name: DeleteUsage :exec
DELETE FROM usage 
WHERE user_id = sqlc.arg(user_id) 
  AND feature_code = sqlc.arg(feature_code) 
  AND resource_type = sqlc.arg(resource_type);

-- name: GetUsageByID :one
SELECT * FROM usage WHERE id = sqlc.arg(id);

-- name: ListUsageByUser :many
SELECT * FROM usage 
WHERE user_id = sqlc.arg(user_id) 
ORDER BY created_at DESC
LIMIT sqlc.arg(limit_count) OFFSET sqlc.arg(offset_count);

-- name: GetUsageStats :many
SELECT 
    feature_code,
    resource_type,
    COUNT(*) as usage_count,
    SUM(resource_size) as total_size,
    MIN(created_at) as first_usage,
    MAX(created_at) as last_usage
FROM usage 
WHERE user_id = sqlc.arg(user_id) 
  AND created_at >= sqlc.arg(since)
GROUP BY feature_code, resource_type
ORDER BY total_size DESC;
