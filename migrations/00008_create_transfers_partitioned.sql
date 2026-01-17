-- +goose Up
-- +goose StatementBegin

-- Function to compute compliance region from currencies
CREATE OR REPLACE FUNCTION compute_compliance_region(from_curr CHAR(3), to_curr CHAR(3))
RETURNS TEXT AS $$
BEGIN
    IF from_curr = 'IDR' OR to_curr = 'IDR' THEN
        RETURN 'ID';
    ELSIF from_curr IN ('EUR', 'SEK', 'DKK') OR to_curr IN ('EUR', 'SEK', 'DKK') THEN
        RETURN 'EU';
    ELSIF from_curr = 'GBP' OR to_curr = 'GBP' THEN
        RETURN 'UK';
    ELSE
        RETURN 'UNKNOWN';
    END IF;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Trigger function to set compliance_region before insert
CREATE OR REPLACE FUNCTION set_compliance_region()
RETURNS TRIGGER AS $$
BEGIN
    NEW.compliance_region := compute_compliance_region(NEW.from_currency, NEW.to_currency);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Transfers table with geo-partitioning by compliance_region
-- compliance_region is derived from currencies and determines data residency
CREATE TABLE transfers (
    id                      UUID NOT NULL DEFAULT uuidv7(),
    tenant_id               UUID NOT NULL,
    -- Legal entity routing
    source_legal_entity_id  UUID,
    dest_legal_entity_id    UUID,
    -- Quote reference (for FX transfers)
    quote_id                UUID,
    -- Batch reference (for bulk transfers)
    batch_id                UUID,
    -- Recipient reference
    recipient_id            UUID,
    -- Idempotency
    idempotency_key         VARCHAR(64),
    -- Currency and amounts
    from_currency           CHAR(3) NOT NULL,
    to_currency             CHAR(3) NOT NULL,
    from_amount             NUMERIC(20,2) NOT NULL,
    to_amount               NUMERIC(20,2) NOT NULL,
    fx_rate                 NUMERIC(20,8) NOT NULL DEFAULT 1.0,
    total_fee               NUMERIC(20,2) NOT NULL DEFAULT 0,
    -- Status
    status                  transfer_status_enum NOT NULL DEFAULT 'created',
    failure_reason          TEXT,
    -- Payment rail
    rail                    rail_enum,
    rail_reference          VARCHAR(100),
    -- Netting
    netting_group_id        UUID,
    is_netted               BOOLEAN NOT NULL DEFAULT false,
    -- TigerBeetle transfer IDs (for audit trail)
    tb_transfer_ids         NUMERIC(39,0)[],
    -- Compliance
    risk_score              INTEGER,
    compliance_status       VARCHAR(20) NOT NULL DEFAULT 'pending',
    screened_at             TIMESTAMPTZ,
    -- Partition key (computed by trigger, not GENERATED)
    -- Default is 'UNKNOWN' but trigger will set correct value
    compliance_region       TEXT NOT NULL DEFAULT 'UNKNOWN',
    -- Timestamps
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at            TIMESTAMPTZ,

    -- Primary key includes partition key
    PRIMARY KEY (id, compliance_region),
    -- Idempotency constraint
    CONSTRAINT unique_idempotency UNIQUE (tenant_id, idempotency_key, compliance_region)
) PARTITION BY LIST (compliance_region);

-- Create partitions for each compliance region
CREATE TABLE transfers_id PARTITION OF transfers FOR VALUES IN ('ID');
CREATE TABLE transfers_eu PARTITION OF transfers FOR VALUES IN ('EU');
CREATE TABLE transfers_uk PARTITION OF transfers FOR VALUES IN ('UK');
CREATE TABLE transfers_unknown PARTITION OF transfers FOR VALUES IN ('UNKNOWN');

-- Create trigger on the parent table (PostgreSQL 11+)
-- This fires BEFORE partition routing, allowing us to set compliance_region
CREATE TRIGGER set_compliance_region_trigger
    BEFORE INSERT ON transfers
    FOR EACH ROW EXECUTE FUNCTION set_compliance_region();

-- Indexes (created on parent, inherited by partitions)
CREATE INDEX idx_transfers_tenant ON transfers(tenant_id);
CREATE INDEX idx_transfers_status ON transfers(status);
CREATE INDEX idx_transfers_updated ON transfers(updated_at DESC);
CREATE INDEX idx_transfers_tenant_status ON transfers(tenant_id, status);
CREATE INDEX idx_transfers_corridor ON transfers(from_currency, to_currency);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS transfers;
DROP FUNCTION IF EXISTS set_compliance_region();
DROP FUNCTION IF EXISTS compute_compliance_region(CHAR(3), CHAR(3));

-- +goose StatementEnd
