-- +goose Up
-- +goose StatementBegin

-- Additional performance indexes for common query patterns

-- Composite indexes for tenant queries
CREATE INDEX idx_transfers_tenant_created ON transfers(tenant_id, created_at DESC);

-- Partial index for active transfers (most common query pattern)
CREATE INDEX idx_transfers_active ON transfers(tenant_id, created_at)
    WHERE status IN ('created', 'validating', 'processing');

-- Index for compliance queries
CREATE INDEX idx_transfers_compliance ON transfers(compliance_status, screened_at)
    WHERE compliance_status = 'pending';

-- BRIN index for time-series queries (efficient for large tables)
CREATE INDEX idx_transfers_created_brin ON transfers USING BRIN (created_at)
    WITH (pages_per_range = 128);

-- Index for batch processing
CREATE INDEX idx_transfers_batch ON transfers(batch_id)
    WHERE batch_id IS NOT NULL;

-- Index for netting queries
CREATE INDEX idx_transfers_netting ON transfers(netting_group_id, is_netted)
    WHERE netting_group_id IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_transfers_netting;
DROP INDEX IF EXISTS idx_transfers_batch;
DROP INDEX IF EXISTS idx_transfers_created_brin;
DROP INDEX IF EXISTS idx_transfers_compliance;
DROP INDEX IF EXISTS idx_transfers_active;
DROP INDEX IF EXISTS idx_transfers_tenant_created;

-- +goose StatementEnd
