-- +goose Up
-- +goose StatementBegin

-- Enable btree_gist extension for temporal EXCLUDE constraint
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- Pricing policies with temporal constraints
-- Each tenant can have different pricing at different time periods
-- EXCLUDE constraint prevents overlapping valid periods
CREATE TABLE pricing_policies (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    -- FX margin in basis points (100 bps = 1%)
    fx_margin_bps           INTEGER NOT NULL DEFAULT 150,
    -- Fee structure as JSON for flexibility
    -- {"transfer_fee_flat": 0, "transfer_fee_percent": 0, "min_fee": 0, "max_fee": null}
    fee_structure           JSONB NOT NULL DEFAULT '{"transfer_fee_flat": 0, "transfer_fee_percent": 0}'::jsonb,
    -- Corridor-specific overrides
    -- {"EUR_IDR": {"fx_margin_bps": 100}, "GBP_IDR": {"fx_margin_bps": 120}}
    corridor_overrides      JSONB NOT NULL DEFAULT '{}'::jsonb,
    -- Valid time range for this pricing
    valid_from              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_until             TIMESTAMPTZ,
    -- Timestamps
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Temporal constraint: prevent overlapping pricing periods for same tenant
    CONSTRAINT pricing_no_overlap EXCLUDE USING gist (
        tenant_id WITH =,
        tstzrange(valid_from, COALESCE(valid_until, 'infinity'::timestamptz)) WITH &&
    )
);

-- Indexes
CREATE INDEX idx_pricing_policies_tenant ON pricing_policies(tenant_id);
CREATE INDEX idx_pricing_policies_valid ON pricing_policies(valid_from, valid_until);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS pricing_policies;
DROP EXTENSION IF EXISTS btree_gist;

-- +goose StatementEnd
