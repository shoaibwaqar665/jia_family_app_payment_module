-- name: InsertWebhookEvent :one
INSERT INTO webhook_events (
    event_id, event_type, payload, signature
) VALUES (
    sqlc.arg(event_id), sqlc.arg(event_type), sqlc.arg(payload), sqlc.arg(signature)
) RETURNING *;

-- name: GetWebhookEventByEventID :one
SELECT * FROM webhook_events WHERE event_id = sqlc.arg(event_id);

-- name: MarkWebhookEventProcessed :one
UPDATE webhook_events 
SET processed = TRUE, processed_at = NOW()
WHERE event_id = sqlc.arg(event_id)
RETURNING *;

-- name: ListUnprocessedWebhookEvents :many
SELECT * FROM webhook_events 
WHERE processed = FALSE
ORDER BY created_at ASC
LIMIT sqlc.arg('limit');
