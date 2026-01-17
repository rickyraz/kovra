-- +goose Up
-- +goose StatementBegin

-- Tenants are B2B clients (e-commerce platforms, sellers, corporate treasury)
CREATE TABLE tenants (
    id                      UUID PRIMARY KEY DEFAULT uuidv7(),
    display_name            VARCHAR(100) NOT NULL,
    legal_name              VARCHAR(200) NOT NULL,
    country                 CHAR(2) NOT NULL,
    tenant_kind             tenant_kind_enum NOT NULL,
    -- Parent tenant for hierarchical relationships (platform -> sellers)
    parent_tenant_id        UUID REFERENCES tenants(id),
    -- Legal entity this tenant is assigned to
    legal_entity_id         UUID NOT NULL REFERENCES legal_entities(id),
    -- Status and verification
    tenant_status           tenant_status_enum NOT NULL DEFAULT 'pending_kyc',
    kyc_level               kyc_level_enum NOT NULL DEFAULT 'basic',
    -- Netting configuration
    netting_enabled         BOOLEAN NOT NULL DEFAULT false,
    netting_window_minutes  INTEGER NOT NULL DEFAULT 5,
    -- API access
    api_key_hash            VARCHAR(64),
    webhook_url             VARCHAR(500),
    webhook_secret_hash     VARCHAR(64),
    -- Metadata
    metadata                JSONB NOT NULL DEFAULT '{}',
    -- Timestamps

    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for common queries
CREATE INDEX idx_tenants_legal_entity ON tenants(legal_entity_id);
CREATE INDEX idx_tenants_parent ON tenants(parent_tenant_id) WHERE parent_tenant_id IS NOT NULL;
CREATE INDEX idx_tenants_status ON tenants(tenant_status) WHERE tenant_status = 'active';
CREATE INDEX idx_tenants_kind ON tenants(tenant_kind);
CREATE INDEX idx_tenants_country ON tenants(country);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS tenants;

-- +goose StatementEnd
