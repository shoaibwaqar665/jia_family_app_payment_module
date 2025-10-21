-- name: InsertOutboxEvent :one
INSERT INTO outbox (
    event_type, payload
) VALUES (
    sqlc.arg(event_type), sqlc.arg(payload)
) RETURNING *;

-- name: GetPendingOutboxEvents :many
SELECT * FROM outbox 
WHERE status = 'pending'
ORDER BY created_at ASC
LIMIT sqlc.arg('limit');

-- name: MarkOutboxEventPublished :one
UPDATE outbox 
SET status = 'published', published_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: MarkOutboxEventFailed :one
UPDATE outbox 
SET status = 'failed', error_message = sqlc.arg(error_message), retry_count = retry_count + 1
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: GetOutboxEventByID :one
SELECT * FROM outbox WHERE id = sqlc.arg(id);

-- name: ListFailedOutboxEvents :many
SELECT * FROM outbox 
WHERE status = 'failed' AND retry_count < 5
ORDER BY created_at ASC
LIMIT sqlc.arg('limit');
