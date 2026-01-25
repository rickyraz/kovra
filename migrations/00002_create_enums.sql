-- +goose Up
-- +goose StatementBegin

-- > Tenant classification
-- platform    → punya sub-tenant = Aggregator / super-merchant / e-commerce platform yang punya banyak sub-merchant 
--              | Tokopedia, Bukalapak, Shopee seller group
-- connected   → nempel ke platform = Sub-merchant / individual seller di bawah platform (sub-tenant) 
--              | Tokopedia seller individu, SwedeMart AB
-- direct      → B2B langsung = Corporate / bisnis langsung (bukan lewat platform), biasanya exporter/importer besar 
--              | Manufacturer Indonesia, EuroFintech GmbH

CREATE TYPE tenant_kind_enum AS ENUM (
    'platform',     -- punya sub-merchant (Tokopedia, Bukalapak)
    'connected',    -- sub-merchant di bawah platform
    'direct',       -- corporate/exporter standalone
    'internal'      -- demo/sandbox/testing
);

-- Tenant lifecycle status
CREATE TYPE tenant_status_enum AS ENUM ('pending_kyc', 'active', 'suspended', 'closed');

-- Transfer state machine
CREATE TYPE transfer_status_enum AS ENUM (
    'created',
    'validating',
    'rejected',
    'processing',
    'completed',
    'rolled_back',
    'cancelled'
);

-- Payment rails
CREATE TYPE rail_enum AS ENUM (
    'SEPA_INSTANT',
    'SEPA_SCT',
    'FPS',
    'CHAPS',
    'BI_FAST',
    'RTGS',
    'SWIFT'
);

-- License types for legal entities
CREATE TYPE license_type_enum AS ENUM ('EMI', 'PI', 'BANK');
-- KYC verification levels
CREATE TYPE kyc_level_enum AS ENUM ('basic', 'standard', 'enhanced');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TYPE IF EXISTS kyc_level_enum;
DROP TYPE IF EXISTS license_type_enum;
DROP TYPE IF EXISTS rail_enum;
DROP TYPE IF EXISTS transfer_status_enum;
DROP TYPE IF EXISTS tenant_status_enum;
DROP TYPE IF EXISTS tenant_kind_enum;

-- +goose StatementEnd
