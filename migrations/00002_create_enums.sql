-- +goose Up
-- +goose StatementBegin

-- Tenant classification
CREATE TYPE tenant_kind_enum AS ENUM ('platform', 'seller', 'direct');

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
