-- Rollback: 0003_add_payments_table

DROP TRIGGER IF EXISTS update_payments_updated_at ON payments;
DROP TABLE IF EXISTS payments;

