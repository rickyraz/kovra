-- +goose Up
-- +goose StatementBegin

-- Limit policies define rate limits and volume caps per tenant
CREATE TABLE limit_policies (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    -- Volume limits (in USD equivalent)
    daily_limit_usd         NUMERIC(15,2) NOT NULL DEFAULT 10000,
    monthly_limit_usd       NUMERIC(15,2) NOT NULL DEFAULT 100000,
    per_transfer_limit_usd  NUMERIC(15,2) NOT NULL DEFAULT 50000,
    -- Rate limits
    rate_limit_rpm          INTEGER NOT NULL DEFAULT 100,  -- requests per minute
    rate_limit_burst        INTEGER NOT NULL DEFAULT 20,   -- burst allowance
    -- Batch limits
    max_batch_size          INTEGER NOT NULL DEFAULT 100,
    max_batch_amount_usd    NUMERIC(15,2) NOT NULL DEFAULT 100000,
    -- Effective date
    effective_from          DATE NOT NULL DEFAULT CURRENT_DATE,
    -- Timestamps
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Only one active limit policy per tenant
    CONSTRAINT unique_tenant_limit UNIQUE (tenant_id)
);

-- Indexes
CREATE INDEX idx_limit_policies_tenant ON limit_policies(tenant_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS limit_policies;

-- +goose StatementEnd
