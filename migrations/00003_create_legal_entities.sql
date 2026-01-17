-- +goose Up
-- +goose StatementBegin

-- Legal entities represent licensed Kovra entities per region
-- KOVRA_EU (Germany), KOVRA_UK (UK), KOVRA_ID (Indonesia)
CREATE TABLE legal_entities (
    id                      UUID PRIMARY KEY DEFAULT uuidv7(),
    code                    VARCHAR(20) NOT NULL UNIQUE,
    legal_name              VARCHAR(200) NOT NULL,
    jurisdiction            CHAR(2) NOT NULL,
    license_type            license_type_enum NOT NULL,
    license_number          VARCHAR(100),
    regulator               VARCHAR(100),
    -- FBO (For Benefit Of) account - pooled client funds
    fbo_bank_name           VARCHAR(100),
    fbo_account_iban        VARCHAR(34),
    fbo_account_number      VARCHAR(50),
    fbo_sort_code           VARCHAR(10),
    -- Nostro account - Kovra's pre-funded settlement account
    nostro_bank_name        VARCHAR(100),
    nostro_account_iban     VARCHAR(34),
    nostro_account_number   VARCHAR(50),
    nostro_sort_code        VARCHAR(10),
    -- Supported operations
    supported_currencies    CHAR(3)[] NOT NULL,
    supported_rails         rail_enum[] NOT NULL,
    -- Timestamps

    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_legal_entities_jurisdiction ON legal_entities(jurisdiction);
CREATE INDEX idx_legal_entities_code ON legal_entities(code);

-- Seed default legal entities
INSERT INTO legal_entities (code, legal_name, jurisdiction, license_type, license_number, regulator,
    fbo_bank_name, fbo_account_iban, nostro_bank_name, nostro_account_iban, supported_currencies, supported_rails)
VALUES
    ('KOVRA_EU', 'Kovra Europe GmbH', 'DE', 'EMI', 'EMI-DE-123456', 'BaFin',
     'Deutsche Bank', 'DE89370400440532013000', 'Deutsche Bank', 'DE89370400440532013001',
     ARRAY['EUR', 'SEK', 'DKK'], ARRAY['SEPA_INSTANT', 'SEPA_SCT']::rail_enum[]),
    ('KOVRA_UK', 'Kovra UK Ltd', 'GB', 'EMI', 'EMI-UK-789012', 'FCA',
     'Barclays', 'GB82WEST12345698765432', 'Barclays', 'GB82WEST12345698765433',
     ARRAY['GBP'], ARRAY['FPS', 'CHAPS']::rail_enum[]),
    ('KOVRA_ID', 'PT Kovra Indonesia', 'ID', 'PI', 'PI-OJK-345678', 'OJK',
     'Bank Mandiri', NULL, 'Bank Mandiri', NULL,
     ARRAY['IDR'], ARRAY['BI_FAST', 'RTGS']::rail_enum[]);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS legal_entities;

-- +goose StatementEnd
