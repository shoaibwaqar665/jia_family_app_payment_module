-- name: CreatePayment :one
INSERT INTO payments (
    amount, currency, status, payment_method, customer_id, order_id, description, external_payment_id, failure_reason, metadata
) VALUES (
    sqlc.arg(amount), sqlc.arg(currency), sqlc.arg(status), sqlc.arg(payment_method),
    sqlc.arg(customer_id), sqlc.arg(order_id), sqlc.narg(description), sqlc.narg(external_payment_id),
    sqlc.narg(failure_reason), sqlc.narg(metadata)
) RETURNING *;

-- name: GetPaymentByID :one
SELECT * FROM payments WHERE id = sqlc.arg(id);

-- name: GetPaymentByOrderID :one
SELECT * FROM payments WHERE order_id = sqlc.arg(order_id);

-- name: GetPaymentsByCustomerID :many
SELECT * FROM payments 
WHERE customer_id = sqlc.arg(customer_id)
ORDER BY created_at DESC;

-- name: UpdatePayment :one
UPDATE payments 
SET amount = sqlc.arg(amount),
    currency = sqlc.arg(currency),
    status = sqlc.arg(status),
    payment_method = sqlc.arg(payment_method),
    customer_id = sqlc.arg(customer_id),
    order_id = sqlc.arg(order_id),
    description = sqlc.narg(description),
    external_payment_id = sqlc.narg(external_payment_id),
    failure_reason = sqlc.narg(failure_reason),
    metadata = sqlc.narg(metadata),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: UpdatePaymentStatus :one
UPDATE payments 
SET status = sqlc.arg(status),
    failure_reason = sqlc.narg(failure_reason),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeletePayment :exec
DELETE FROM payments WHERE id = sqlc.arg(id);

-- name: ListPayments :many
SELECT * FROM payments 
ORDER BY created_at DESC;

-- name: CountPayments :one
SELECT COUNT(*) FROM payments;
