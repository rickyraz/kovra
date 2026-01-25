-- +goose Up
-- +goose StatementBegin

-- Demo Tenants for Phase 1 testing
-- These tenants represent B2B clients under each legal entity

-- Tenant 1: European Fintech (under KOVRA_EU)
INSERT INTO tenants (id, display_name, legal_name, country, tenant_kind, tenant_status, kyc_level,
    legal_entity_id, netting_enabled, netting_window_minutes, metadata)
SELECT
    '019471a0-0000-7000-8000-000000000001'::uuid,
    'EuroFintech GmbH',
    'EuroFintech GmbH',
    'DE',
    'platform'::tenant_kind_enum,
    'active'::tenant_status_enum,
    'enhanced'::kyc_level_enum,
    id,
    true,
    60,
    '{"industry": "fintech", "employees": 50, "api_version": "v1"}'::jsonb
FROM legal_entities WHERE code = 'KOVRA_EU';

-- Tenant 2: UK Payment Provider (under KOVRA_UK)
INSERT INTO tenants (id, display_name, legal_name, country, tenant_kind, tenant_status, kyc_level,
    legal_entity_id, netting_enabled, netting_window_minutes, metadata)
SELECT
    '019471a0-0000-7000-8000-000000000002'::uuid,
    'BritPay Ltd',
    'BritPay Limited',
    'GB',
    'direct'::tenant_kind_enum,
    'active'::tenant_status_enum,
    'enhanced'::kyc_level_enum,
    id,
    true,
    30,
    '{"industry": "payments", "employees": 25, "api_version": "v1"}'::jsonb
FROM legal_entities WHERE code = 'KOVRA_UK';

-- Tenant 3: Indonesian Remittance (under KOVRA_ID)
INSERT INTO tenants (id, display_name, legal_name, country, tenant_kind, tenant_status, kyc_level,
    legal_entity_id, netting_enabled, netting_window_minutes, metadata)
SELECT
    '019471a0-0000-7000-8000-000000000003'::uuid,
    'IndoRemit',
    'PT IndoRemit Indonesia',
    'ID',
    'direct'::tenant_kind_enum,
    'active'::tenant_status_enum,
    'standard'::kyc_level_enum,
    id,
    false,
    0,
    '{"industry": "remittance", "employees": 100, "api_version": "v1"}'::jsonb
FROM legal_entities WHERE code = 'KOVRA_ID';

-- Tenant 4: Swedish E-commerce (sub-tenant under EuroFintech - under KOVRA_EU)
INSERT INTO tenants (id, display_name, legal_name, country, tenant_kind, tenant_status, kyc_level,
    legal_entity_id, parent_tenant_id, netting_enabled, metadata)
SELECT
    '019471a0-0000-7000-8000-000000000004'::uuid,
    'SwedeMart AB',
    'SwedeMart AB',
    'SE',
    'seller'::tenant_kind_enum,
    'active'::tenant_status_enum,
    'basic'::kyc_level_enum,
    le.id,
    '019471a0-0000-7000-8000-000000000001'::uuid,
    false,
    '{"industry": "e-commerce", "parent_contract": "EF-2024-001"}'::jsonb
FROM legal_entities le WHERE le.code = 'KOVRA_EU';

-- Pricing Policies for demo tenants
-- EuroFintech: Premium pricing (lower margin)
INSERT INTO pricing_policies (tenant_id, fx_margin_bps, fee_structure, corridor_overrides, valid_from)
VALUES (
    '019471a0-0000-7000-8000-000000000001'::uuid,
    100,  -- 1% margin
    '{"transfer_fee_flat": 0, "transfer_fee_percent": 0.1, "min_fee": 1, "max_fee": 50}'::jsonb,
    '{"EUR_IDR": {"fx_margin_bps": 80}, "EUR_GBP": {"fx_margin_bps": 50}}'::jsonb,
    NOW()
);

-- BritPay: Standard pricing
INSERT INTO pricing_policies (tenant_id, fx_margin_bps, fee_structure, corridor_overrides, valid_from)
VALUES (
    '019471a0-0000-7000-8000-000000000002'::uuid,
    150,  -- 1.5% margin
    '{"transfer_fee_flat": 2, "transfer_fee_percent": 0.15, "min_fee": 2, "max_fee": 100}'::jsonb,
    '{"GBP_IDR": {"fx_margin_bps": 120}}'::jsonb,
    NOW()
);

-- IndoRemit: Higher margin (outbound from ID)
INSERT INTO pricing_policies (tenant_id, fx_margin_bps, fee_structure, corridor_overrides, valid_from)
VALUES (
    '019471a0-0000-7000-8000-000000000003'::uuid,
    200,  -- 2% margin
    '{"transfer_fee_flat": 50000, "transfer_fee_percent": 0.2, "min_fee": 50000, "max_fee": 500000}'::jsonb,
    '{}'::jsonb,
    NOW()
);

-- SwedeMart: Inherits from parent but with sub-merchant rates
INSERT INTO pricing_policies (tenant_id, fx_margin_bps, fee_structure, corridor_overrides, valid_from)
VALUES (
    '019471a0-0000-7000-8000-000000000004'::uuid,
    120,  -- 1.2% margin
    '{"transfer_fee_flat": 5, "transfer_fee_percent": 0.12, "min_fee": 5, "max_fee": 25}'::jsonb,
    '{}'::jsonb,
    NOW()
);

-- Limit Policies for demo tenants
-- EuroFintech: High volume enterprise
INSERT INTO limit_policies (tenant_id, daily_limit_usd, monthly_limit_usd, per_transfer_limit_usd,
    rate_limit_rpm, rate_limit_burst, max_batch_size, max_batch_amount_usd)
VALUES (
    '019471a0-0000-7000-8000-000000000001'::uuid,
    1000000,    -- $1M daily
    10000000,   -- $10M monthly
    500000,     -- $500K per transfer
    1000,       -- 1000 rpm
    100,        -- 100 burst
    500,        -- 500 transfers per batch
    5000000     -- $5M per batch
);

-- BritPay: Medium volume
INSERT INTO limit_policies (tenant_id, daily_limit_usd, monthly_limit_usd, per_transfer_limit_usd,
    rate_limit_rpm, rate_limit_burst, max_batch_size, max_batch_amount_usd)
VALUES (
    '019471a0-0000-7000-8000-000000000002'::uuid,
    500000,     -- $500K daily
    5000000,    -- $5M monthly
    100000,     -- $100K per transfer
    500,        -- 500 rpm
    50,         -- 50 burst
    200,        -- 200 transfers per batch
    1000000     -- $1M per batch
);

-- IndoRemit: Standard volume with OJK compliance
INSERT INTO limit_policies (tenant_id, daily_limit_usd, monthly_limit_usd, per_transfer_limit_usd,
    rate_limit_rpm, rate_limit_burst, max_batch_size, max_batch_amount_usd)
VALUES (
    '019471a0-0000-7000-8000-000000000003'::uuid,
    100000,     -- $100K daily (OJK limit)
    1000000,    -- $1M monthly
    25000,      -- $25K per transfer
    200,        -- 200 rpm
    20,         -- 20 burst
    100,        -- 100 transfers per batch
    250000      -- $250K per batch
);

-- SwedeMart: Lower limits as sub-merchant
INSERT INTO limit_policies (tenant_id, daily_limit_usd, monthly_limit_usd, per_transfer_limit_usd,
    rate_limit_rpm, rate_limit_burst, max_batch_size, max_batch_amount_usd)
VALUES (
    '019471a0-0000-7000-8000-000000000004'::uuid,
    50000,      -- $50K daily
    500000,     -- $500K monthly
    10000,      -- $10K per transfer
    100,        -- 100 rpm
    10,         -- 10 burst
    50,         -- 50 transfers per batch
    100000      -- $100K per batch
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Remove in reverse order due to foreign keys
DELETE FROM limit_policies WHERE tenant_id IN (
    '019471a0-0000-7000-8000-000000000001'::uuid,
    '019471a0-0000-7000-8000-000000000002'::uuid,
    '019471a0-0000-7000-8000-000000000003'::uuid,
    '019471a0-0000-7000-8000-000000000004'::uuid
);

DELETE FROM pricing_policies WHERE tenant_id IN (
    '019471a0-0000-7000-8000-000000000001'::uuid,
    '019471a0-0000-7000-8000-000000000002'::uuid,
    '019471a0-0000-7000-8000-000000000003'::uuid,
    '019471a0-0000-7000-8000-000000000004'::uuid
);

DELETE FROM tenants WHERE id IN (
    '019471a0-0000-7000-8000-000000000001'::uuid,
    '019471a0-0000-7000-8000-000000000002'::uuid,
    '019471a0-0000-7000-8000-000000000003'::uuid,
    '019471a0-0000-7000-8000-000000000004'::uuid
);

-- +goose StatementEnd
