# Database Architecture — Kovra Payment Rails

B2B cross-border payment rails platform dengan data terdistribusi di PostgreSQL 18, TigerBeetle, Redis, dan Kafka.

---

## Design Principles

|#|Principle|Description|
|---|---|---|
|1|FBO/Nostro Alignment|TigerBeetle accounts map ke bank-level FBO/Nostro|
|2|Legal Entity First|Tenants assigned ke licensed entities per region|
|3|Identity vs Policy|Tenant identity terpisah dari pricing/limits|
|4|Compliance by Design|DHE, CESOP, AML/KYC built-in|
|5|Audit Everything|Immutable trail untuk regulatory requirements|
|6|Geo-Partitioning|Data residency enforced di database level|

---

## Data Distribution

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           DATA RESPONSIBILITY                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  PostgreSQL 18                    TigerBeetle                               │
│  ┌─────────────────────┐          ┌─────────────────────┐                   │
│  │ • Business metadata │          │ • TENANT_WALLET     │ ◄─ FBO sub-ledger │
│  │ • Legal entities    │          │ • REGIONAL_SETTLE   │ ◄─ Nostro balance │
│  │ • Tenant config     │          │ • FEE_REVENUE       │                   │
│  │ • Compliance logs   │          │ • FX_SETTLEMENT     │                   │
│  │ • Pricing policies  │          │ • PENDING_IN/OUT    │                   │
│  │ • River job queue   │          │ • Atomic guarantees │                   │
│  │ • Geo-partitioned   │          └─────────────────────┘                   │
│  └─────────────────────┘                                                    │
│                                                                             │
│  Redis 8                          Kafka                                     │
│  ┌─────────────────────┐          ┌─────────────────────┐                   │
│  │ • FX rate locks     │          │ • Transfer events   │                   │
│  │ • Rate limiting     │          │ • Audit stream      │                   │
│  │ • Idempotency keys  │          │ • Compliance events │                   │
│  │ • Netting window    │          │ • Webhook DLQ       │                   │
│  └─────────────────────┘          └─────────────────────┘                   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## TigerBeetle ↔ Bank Alignment

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    TIGERBEETLE ↔ BANK ALIGNMENT                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  BANK LEVEL (Regulatory)              TIGERBEETLE (Internal Ledger)         │
│  ────────────────────────             ─────────────────────────────         │
│                                                                             │
│  FBO EUR @ Deutsche Bank         ───► SUM(TENANT_WALLET where currency=EUR) │
│  FBO GBP @ Barclays              ───► SUM(TENANT_WALLET where currency=GBP) │
│  FBO IDR @ Bank Mandiri          ───► SUM(TENANT_WALLET where currency=IDR) │
│                                                                             │
│  Nostro EUR @ Deutsche Bank      ───► REGIONAL_SETTLEMENT_EU                │
│  Nostro GBP @ Barclays           ───► REGIONAL_SETTLEMENT_UK                │
│  Nostro IDR @ Bank Mandiri       ───► REGIONAL_SETTLEMENT_ID                │
│                                                                             │
│  Kovra Revenue Account           ───► FEE_REVENUE (per currency)            │
│                                                                             │
│  (Internal only - no bank)       ───► FX_SETTLEMENT                         │
│  (Internal only - no bank)       ───► PENDING_INBOUND                       │
│  (Internal only - no bank)       ───► PENDING_OUTBOUND                      │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Invariants:**

- `SUM(TENANT_WALLET per currency)` == FBO Bank Statement
- `REGIONAL_SETTLEMENT per region` == Nostro Bank Statement
- Daily reconciliation validates these invariants

---

## TigerBeetle Account Types

|Code|Type|Maps To|Owner|Purpose|
|---|---|---|---|---|
|0x01|TENANT_WALLET|FBO sub-ledger|Tenant (segregated)|Client funds per currency|
|0x02|FEE_REVENUE|Kovra operational|Kovra|Collected fees|
|0x03|FX_SETTLEMENT|FX Position/Exposure|System|Platform's currency position for FX operations|
|0x04|PENDING_INBOUND|Transit|System|Funds being collected|
|0x05|PENDING_OUTBOUND|Transit|System|Funds being disbursed|
|0x06|REGIONAL_SETTLEMENT|Nostro accounts|Kovra|Pre-funded settlement liquidity|

> **⚠️ TigerBeetle Constraint:** All accounts in a single transfer MUST be in the same ledger.
> Cross-currency FX requires two separate transfer chains (one per ledger) coordinated at application level.
> See "Cross-Currency FX Transfer" diagram below for the correct pattern.

### Account ID Structure (128-bit)

```
┌────────────────────────────────────────────────────────────────────────────┐
│  [tenant_id: 64 bits] [account_type: 8 bits] [currency: 24 bits] [reserved]│
└────────────────────────────────────────────────────────────────────────────┘
```

Examples:

- tokopedia_seller EUR wallet: `0x{tenant_hash}_01_978`
- REGIONAL_SETTLEMENT_EU: `0x{system}_06_978`
- FEE_REVENUE_IDR: `0x{kovra}_02_360`

### Ledger Codes (ISO 4217)

|Code|Currency|Region|
|---|---|---|
|978|EUR|EU|
|826|GBP|UK|
|360|IDR|Indonesia|
|752|SEK|Sweden (EU)|
|208|DKK|Denmark (EU)|

---

## Entity Relationship Diagram

```
┌──────────────────┐
│ legal_entities   │◄─────────────────────────────────────┐
│                  │                                      │
│ • Kovra Europe   │                                      │
│ • Kovra UK       │                                      │
│ • PT Kovra ID    │                                      │
└────────┬─────────┘                                      │
         │ 1:N                                            │
         ▼                                                │
┌──────────────────┐       ┌────────────────────────┐     │
│    tenants       │──────►│ tenant_capabilities    │     │
│                  │  1:1  │                        │     │
│ • legal_entity_id│       │ • can_have_children    │     │
│ • tenant_kind    │       │ • can_netting          │     │
│ • parent_id      │       │ • can_batch_transfer   │     │
│ • api_key_hash   │       │ • fees_responsibility  │     │
│ • webhook_url    │       │ • dashboard_access     │     │
│ • metadata       │       └────────────────────────┘     │
└────────┬─────────┘                                      │
         │                                                │
    ┌────┴────┬───────────────────┐                       │
    │         │                   │                       │
    ▼         ▼                   ▼                       │
┌────────┐ ┌────────┐       ┌──────────────────┐          │
│pricing │ │limits  │       │   transfers      │          │
│policies│ │policies│       │                  │          │
└────────┘ └────────┘       │ • source_entity ─┼──────────┘
                            │ • dest_entity    │
                            │ • netting_group  │
                            └──────────────────┘
```

---

## PostgreSQL 18 Geo-Partitioning

### Partitioned Tables Structure

```
transfers
├── transfers_id (compliance_region = 'ID')
│   └── Row-Level Security: OJK policy
├── transfers_eu (compliance_region = 'EU')
│   └── Row-Level Security: GDPR policy
└── transfers_uk (compliance_region = 'UK')
    └── Row-Level Security: FCA policy

compliance_logs
├── compliance_logs_id
├── compliance_logs_eu
└── compliance_logs_uk

audit_trail
├── audit_trail_id
├── audit_trail_eu
└── audit_trail_uk
```

### Partitioned Tables Summary

|Table|Partitions|RLS Policy|
|---|---|---|
|transfers|transfers_id, transfers_eu, transfers_uk|OJK, GDPR, FCA|
|compliance_logs|compliance_logs_id, compliance_logs_eu, compliance_logs_uk|Per-region|
|audit_trail|audit_trail_id, audit_trail_eu, audit_trail_uk|Per-region|

### Query Performance

```sql
-- ✅ GOOD: Partition pruning (70% faster)
SELECT * FROM transfers 
WHERE compliance_region = 'ID' AND status = 'completed';

-- Cross-region (auditor access only)
SELECT compliance_region, COUNT(*) FROM transfers GROUP BY compliance_region;
```

### Future Migration (YugabyteDB)

```
┌──────────────────────────────────────────────────────────────────────────┐
│                    YUGABYTEDB GEO-DISTRIBUTION                           │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   ap-southeast-3 (Jakarta)     eu-central-1 (Frankfurt)    eu-west-2     │
│   ┌─────────────────┐          ┌─────────────────┐     ┌─────────────┐   │
│   │ transfers_id    │          │ transfers_eu    │     │ transfers_uk│   │
│   │ compliance_id   │          │ compliance_eu   │     │ compliance_ │   │
│   │ audit_trail_id  │          │ audit_trail_eu  │     │ audit_trail │   │
│   │                 │          │                 │     │             │   │
│   │ id_tablespace   │          │ eu_tablespace   │     │ uk_tablespc │   │
│   └─────────────────┘          └─────────────────┘     └─────────────┘   │
│          │                            │                       │          │
│          └────────────────────────────┴───────────────────────┘          │
│                                  │                                       │
│                          Raft Consensus                                  │
│                    (Cross-region replication)                            │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

|Aspect|Detail|
|---|---|
|Migration Effort|2-3 weeks|
|Cost Impact|+$400/month (9 nodes vs 1 instance)|
|Code Changes|Zero (wire-compatible)|

---

## Transfer Flow Diagram

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   created   │────►│  validating │────►│ processing  │────►│  completed  │
└─────────────┘     └──────┬──────┘     └──────┬──────┘     └─────────────┘
                          │                   │
                          ▼                   ▼
                   ┌─────────────┐     ┌─────────────┐
                   │  rejected   │     │ rolled_back │
                   └─────────────┘     └─────────────┘
                          
                   ┌─────────────┐
                   │  cancelled  │ ◄── User initiated
                   └─────────────┘
```

### Transfer Settlement Flow (TigerBeetle)

#### Same-Currency Transfer (Atomic)

```
┌──────────────────────────────────────────────────────────────────────────┐
│                SAME-CURRENCY TRANSFER (Single Ledger)                    │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  All steps execute atomically via TigerBeetle linked transfers           │
│  Ledger: EUR (978)                                                       │
│                                                                          │
│  Step 1: Debit Source                                                    │
│  ┌─────────────────────┐                                                 │
│  │ TENANT_WALLET (EUR) │ ──── debit ────►  PENDING_OUTBOUND (EUR)        │
│  └─────────────────────┘                                                 │
│                                                                          │
│  Step 2: Fee Collection (if fee > 0)                                     │
│  ┌─────────────────────┐                                                 │
│  │ PENDING_OUTBOUND    │ ──── fee ──────►  FEE_REVENUE (EUR)             │
│  └─────────────────────┘                                                 │
│                                                                          │
│  Step 3: Settlement                                                      │
│  ┌─────────────────────┐                                                 │
│  │ PENDING_OUTBOUND    │ ──── payout ───►  REGIONAL_SETTLEMENT (EUR)     │
│  └─────────────────────┘                                                 │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

#### Cross-Currency FX Transfer (Two Coordinated Chains)

```
┌──────────────────────────────────────────────────────────────────────────┐
│              CROSS-CURRENCY FX TRANSFER (EUR → IDR)                      │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ⚠️  TigerBeetle Constraint: Accounts in a transfer MUST be same ledger  │
│  → FX requires TWO separate chains coordinated at application level      │
│  → CorrelationID (UserData128) links both chains for reconciliation      │
│                                                                          │
│  ═══════════════════════════════════════════════════════════════════════ │
│  SOURCE CHAIN (EUR Ledger = 978)                                         │
│  ═══════════════════════════════════════════════════════════════════════ │
│                                                                          │
│  Step 1: Hold source funds                                               │
│  ┌─────────────────────┐                                                 │
│  │ TENANT_WALLET (EUR) │ ──── debit ────►  PENDING_OUTBOUND (EUR)        │
│  └─────────────────────┘                                                 │
│                                                                          │
│  Step 2: Platform acquires EUR                                           │
│  ┌─────────────────────┐                                                 │
│  │ PENDING_OUTBOUND    │ ──── credit ───►  FX_POSITION (EUR)             │
│  └─────────────────────┘                   (platform's EUR exposure)     │
│                                                                          │
│  ═══════════════════════════════════════════════════════════════════════ │
│  DESTINATION CHAIN (IDR Ledger = 360)                                    │
│  ═══════════════════════════════════════════════════════════════════════ │
│                                                                          │
│  Step 1: Fee deduction (if fee > 0)                                      │
│  ┌─────────────────────┐                                                 │
│  │ FX_POSITION (IDR)   │ ──── fee ──────►  FEE_REVENUE (IDR)             │
│  └─────────────────────┘                                                 │
│                                                                          │
│  Step 2: Payout                                                          │
│  ┌─────────────────────┐                                                 │
│  │ FX_POSITION (IDR)   │ ──── payout ───►  REGIONAL_SETTLEMENT (IDR)     │
│  └─────────────────────┘                                                 │
│                                                                          │
│  ═══════════════════════════════════════════════════════════════════════ │
│  APPLICATION COORDINATION                                                │
│  ═══════════════════════════════════════════════════════════════════════ │
│                                                                          │
│  1. Execute Source Chain (atomic within EUR ledger)                      │
│  2. Execute Destination Chain (atomic within IDR ledger)                 │
│  3. If either fails → compensating transaction required                  │
│  4. FX rate & conversion stored in PostgreSQL transfers table            │
│  5. CorrelationID links chains for audit/reconciliation                  │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

**FX_POSITION Account Semantics:**

| Operation | Meaning |
|-----------|---------|
| Credit FX_POSITION_EUR | Platform receives EUR from user |
| Debit FX_POSITION_IDR | Platform pays out IDR to settlement |

The FX_POSITION accounts track platform's currency exposure. Treasury operations
(hedging, rebalancing) happen outside TigerBeetle and reconcile periodically.

---

## PostgreSQL Schema

### Enums

```sql
-- Tenant classification:
-- platform   → punya sub-tenant (Aggregator / super-merchant / e-commerce platform)
-- connected  → sub-merchant di bawah platform (sub-tenant)
-- direct     → B2B langsung (corporate/exporter standalone)
-- internal   → demo/sandbox/testing
CREATE TYPE tenant_kind_enum AS ENUM ('platform', 'connected', 'direct', 'internal');
CREATE TYPE tenant_status_enum AS ENUM ('pending_kyc', 'active', 'suspended', 'closed');
CREATE TYPE kyc_level_enum AS ENUM ('basic', 'standard', 'enhanced');
CREATE TYPE license_type_enum AS ENUM ('EMI', 'PI', 'BANK');
CREATE TYPE transfer_status_enum AS ENUM (
    'created', 'validating', 'rejected', 'processing',
    'completed', 'rolled_back', 'cancelled'
);
CREATE TYPE rail_enum AS ENUM (
    'SEPA_INSTANT', 'SEPA_SCT', 'FPS', 'CHAPS',
    'BI_FAST', 'RTGS', 'SWIFT'
);
```

### Legal Entities

```sql
CREATE TABLE legal_entities (
    id                      UUID PRIMARY KEY DEFAULT uuidv7(),
    code                    VARCHAR(20) NOT NULL UNIQUE,
    legal_name              VARCHAR(200) NOT NULL,
    jurisdiction            CHAR(2) NOT NULL,
    license_type            license_type_enum NOT NULL,
    license_number          VARCHAR(100),
    regulator               VARCHAR(100),
    fbo_bank_name           VARCHAR(100),
    fbo_account_iban        VARCHAR(34),
    nostro_bank_name        VARCHAR(100),
    nostro_account_iban     VARCHAR(34),
    supported_currencies    CHAR(3)[] NOT NULL,
    supported_rails         rail_enum[] NOT NULL,
    tax_id_requirements     JSONB NOT NULL DEFAULT '[]',
    reporting_obligations   JSONB NOT NULL DEFAULT '{}',
    ,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_legal_entities_jurisdiction ON legal_entities(jurisdiction);
```

### Tenants

```sql
CREATE TABLE tenants (
    id                      UUID PRIMARY KEY NOT NULL,  -- App-generated UUIDv7
    display_name            VARCHAR(100) NOT NULL,
    legal_name              VARCHAR(200) NOT NULL,
    country                 CHAR(2) NOT NULL,
    tenant_kind             tenant_kind_enum NOT NULL,
    parent_tenant_id        UUID REFERENCES tenants(id),
    legal_entity_id         UUID NOT NULL REFERENCES legal_entities(id),
    tenant_status           tenant_status_enum NOT NULL DEFAULT 'pending_kyc',
    kyc_level               kyc_level_enum NOT NULL DEFAULT 'basic',
    netting_enabled         BOOLEAN NOT NULL DEFAULT false,
    netting_window_minutes  INTEGER NOT NULL DEFAULT 5,
    -- API access
    api_key_hash            VARCHAR(64),
    webhook_url             VARCHAR(500),
    webhook_secret_hash     VARCHAR(64),
    -- Metadata
    metadata                JSONB NOT NULL DEFAULT '{}',
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Connected tenants must have parent
    CONSTRAINT chk_connected_parent CHECK (
        tenant_kind != 'connected' OR parent_tenant_id IS NOT NULL
    )
);

CREATE INDEX idx_tenants_legal_entity ON tenants(legal_entity_id);
CREATE INDEX idx_tenants_parent ON tenants(parent_tenant_id) WHERE parent_tenant_id IS NOT NULL;
CREATE INDEX idx_tenants_status ON tenants(tenant_status) WHERE tenant_status = 'active';
CREATE INDEX idx_tenants_kind ON tenants(tenant_kind);
CREATE INDEX idx_tenants_country ON tenants(country);
```

### Tenant Capabilities

```sql
-- Capability flags per tenant (1:1 with tenants)
CREATE TABLE tenant_capabilities (
    tenant_id           UUID PRIMARY KEY REFERENCES tenants(id),
    can_have_children   BOOLEAN NOT NULL DEFAULT false,  -- Platform-only
    can_netting         BOOLEAN NOT NULL DEFAULT false,
    can_batch_transfer  BOOLEAN NOT NULL DEFAULT false,
    fees_responsibility TEXT NOT NULL DEFAULT 'platform',  -- 'platform' | 'tenant'
    kyc_tier            TEXT NOT NULL DEFAULT 'basic',     -- 'basic' | 'standard' | 'enhanced'
    dashboard_access    TEXT NOT NULL DEFAULT 'none',      -- 'full' | 'limited' | 'none'
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Pricing & Limit Policies

```sql
-- PG 18 Temporal Constraint (no overlapping periods)
CREATE TABLE pricing_policies (
    id                      UUID PRIMARY KEY DEFAULT uuidv7(),
    tenant_id               UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    fx_margin_bps           INTEGER NOT NULL DEFAULT 150,
    fee_structure           JSONB NOT NULL DEFAULT '{"transfer_fee_flat": 0, "transfer_fee_percent": 0}',
    corridor_overrides      JSONB NOT NULL DEFAULT '{}',
    valid_from              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_until             TIMESTAMPTZ,
    CONSTRAINT pricing_no_overlap EXCLUDE USING gist (
        tenant_id WITH =,
        tstzrange(valid_from, COALESCE(valid_until, 'infinity'::timestamptz)) WITH &&
    )
);

CREATE TABLE limit_policies (
    id                      UUID PRIMARY KEY DEFAULT uuidv7(),
    tenant_id               UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    -- Volume limits (in USD equivalent)
    daily_limit_usd         NUMERIC(15,2) NOT NULL DEFAULT 10000,
    monthly_limit_usd       NUMERIC(15,2) NOT NULL DEFAULT 100000,
    per_transfer_limit_usd  NUMERIC(15,2) NOT NULL DEFAULT 50000,
    -- Rate limits
    rate_limit_rpm          INTEGER NOT NULL DEFAULT 100,
    rate_limit_burst        INTEGER NOT NULL DEFAULT 20,
    -- Batch limits
    max_batch_size          INTEGER NOT NULL DEFAULT 100,
    max_batch_amount_usd    NUMERIC(15,2) NOT NULL DEFAULT 100000,
    effective_from          DATE NOT NULL DEFAULT CURRENT_DATE,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_tenant_limit UNIQUE (tenant_id)
);
```

### Wallets (TigerBeetle Reference)

```sql
CREATE TABLE wallets (
    id                      UUID PRIMARY KEY NOT NULL,  -- App-generated UUIDv7
    tenant_id               UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    currency                CHAR(3) NOT NULL,
    tb_account_id           NUMERIC(39,0) NOT NULL UNIQUE,
    cached_balance          NUMERIC(20,2) NOT NULL DEFAULT 0,
    cached_pending          NUMERIC(20,2) NOT NULL DEFAULT 0,
    cached_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status                  VARCHAR(20) NOT NULL DEFAULT 'active',
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_tenant_currency UNIQUE (tenant_id, currency)
);
```

### Transfers (Geo-Partitioned)

```sql
-- Compliance region computed by trigger (not GENERATED)
CREATE OR REPLACE FUNCTION compute_compliance_region(from_curr CHAR(3), to_curr CHAR(3))
RETURNS TEXT AS $$
BEGIN
    IF from_curr = 'IDR' OR to_curr = 'IDR' THEN RETURN 'ID';
    ELSIF from_curr IN ('EUR', 'SEK', 'DKK') OR to_curr IN ('EUR', 'SEK', 'DKK') THEN RETURN 'EU';
    ELSIF from_curr = 'GBP' OR to_curr = 'GBP' THEN RETURN 'UK';
    ELSE RETURN 'UNKNOWN';
    END IF;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

CREATE TABLE transfers (
    id                      UUID NOT NULL,
    tenant_id               UUID NOT NULL,
    source_legal_entity_id  UUID,
    dest_legal_entity_id    UUID,
    quote_id                UUID,
    batch_id                UUID,
    recipient_id            UUID,
    idempotency_key         VARCHAR(64),
    from_currency           CHAR(3) NOT NULL,
    to_currency             CHAR(3) NOT NULL,
    from_amount             NUMERIC(20,2) NOT NULL,
    to_amount               NUMERIC(20,2) NOT NULL,
    fx_rate                 NUMERIC(20,8) NOT NULL DEFAULT 1.0,
    total_fee               NUMERIC(20,2) NOT NULL DEFAULT 0,
    status                  transfer_status_enum NOT NULL DEFAULT 'created',
    failure_reason          TEXT,
    rail                    rail_enum,
    rail_reference          VARCHAR(100),
    netting_group_id        UUID,
    is_netted               BOOLEAN NOT NULL DEFAULT false,
    tb_transfer_ids         NUMERIC(39,0)[],
    risk_score              INTEGER,
    compliance_status       VARCHAR(20) NOT NULL DEFAULT 'pending',
    screened_at             TIMESTAMPTZ,
    compliance_region       TEXT NOT NULL DEFAULT 'UNKNOWN',  -- Set by trigger
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at            TIMESTAMPTZ,
    PRIMARY KEY (id, compliance_region),
    CONSTRAINT unique_idempotency UNIQUE (tenant_id, idempotency_key, compliance_region)
) PARTITION BY LIST (compliance_region);

CREATE TABLE transfers_id PARTITION OF transfers FOR VALUES IN ('ID');
CREATE TABLE transfers_eu PARTITION OF transfers FOR VALUES IN ('EU');
CREATE TABLE transfers_uk PARTITION OF transfers FOR VALUES IN ('UK');
CREATE TABLE transfers_unknown PARTITION OF transfers FOR VALUES IN ('UNKNOWN');

-- Trigger to set compliance_region before insert
CREATE TRIGGER set_compliance_region_trigger
    BEFORE INSERT ON transfers
    FOR EACH ROW EXECUTE FUNCTION set_compliance_region();
```

### Netting Groups

```sql
CREATE TABLE netting_groups (
    id                      UUID PRIMARY KEY DEFAULT uuidv7(),
    tenant_id               UUID NOT NULL REFERENCES tenants(id),
    corridor                VARCHAR(7) NOT NULL,
    gross_inbound           NUMERIC(20,2) NOT NULL DEFAULT 0,
    gross_outbound          NUMERIC(20,2) NOT NULL DEFAULT 0,
    net_amount              NUMERIC(20,2) NOT NULL DEFAULT 0,
    net_direction           VARCHAR(10),
    fx_saved                NUMERIC(20,2) NOT NULL DEFAULT 0,
    window_start            TIMESTAMPTZ NOT NULL,
    window_end              TIMESTAMPTZ NOT NULL,
    status                  VARCHAR(20) NOT NULL DEFAULT 'open',
    executed_at             TIMESTAMPTZ,
    
);
```

### Regional Settlement Tracking

```sql
CREATE TABLE regional_settlements (
    id                      UUID PRIMARY KEY DEFAULT uuidv7(),
    legal_entity_id         UUID NOT NULL REFERENCES legal_entities(id),
    currency                CHAR(3) NOT NULL,
    tb_account_id           NUMERIC(39,0) NOT NULL UNIQUE,
    cached_balance          NUMERIC(20,2) NOT NULL DEFAULT 0,
    cached_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    min_balance             NUMERIC(20,2) NOT NULL,
    target_balance          NUMERIC(20,2) NOT NULL,
    alert_threshold         NUMERIC(20,2) NOT NULL,
    last_reconciled_at      TIMESTAMPTZ,
    reconciliation_status   VARCHAR(20) DEFAULT 'pending',
    ,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_entity_currency UNIQUE (legal_entity_id, currency)
);
```

### Reconciliation Reports

```sql
CREATE TABLE reconciliation_reports (
    id                      UUID PRIMARY KEY DEFAULT uuidv7(),
    legal_entity_id         UUID NOT NULL REFERENCES legal_entities(id),
    reconciliation_date     DATE NOT NULL,
    fbo_tigerbeetle_sum     NUMERIC(20,2),
    fbo_bank_statement      NUMERIC(20,2),
    fbo_match               BOOLEAN,
    nostro_tigerbeetle      NUMERIC(20,2),
    nostro_bank_statement   NUMERIC(20,2),
    nostro_match            BOOLEAN,
    discrepancy_count       INTEGER NOT NULL DEFAULT 0,
    discrepancies           JSONB,
    status                  VARCHAR(20) NOT NULL DEFAULT 'pending',
    reviewed_by             VARCHAR(100),
    reviewed_at             TIMESTAMPTZ,
    CONSTRAINT unique_recon_date UNIQUE (legal_entity_id, reconciliation_date)
);
```

### Compliance Tables

```sql
-- DHE Records (Indonesia Devisa Hasil Ekspor)
CREATE TABLE dhe_records (
    id                      UUID PRIMARY KEY DEFAULT uuidv7(),
    tenant_id               UUID NOT NULL REFERENCES tenants(id),
    legal_entity_id         UUID NOT NULL REFERENCES legal_entities(id),
    period_year             INTEGER NOT NULL,
    period_month            INTEGER NOT NULL,
    total_export_usd        NUMERIC(20,2) NOT NULL DEFAULT 0,
    dhe_required            BOOLEAN NOT NULL DEFAULT false,
    dhe_required_amount     NUMERIC(20,2),
    dhe_deposited           NUMERIC(20,2) DEFAULT 0,
    compliant               BOOLEAN NOT NULL DEFAULT false,
    reported_to_bi          BOOLEAN NOT NULL DEFAULT false,
    ,
    CONSTRAINT unique_dhe_period UNIQUE (tenant_id, period_year, period_month)
);

-- CESOP Reports (EU Central Electronic System of Payment Information)
CREATE TABLE cesop_reports (
    id                      UUID PRIMARY KEY DEFAULT uuidv7(),
    legal_entity_id         UUID NOT NULL REFERENCES legal_entities(id),
    period_year             INTEGER NOT NULL,
    period_quarter          INTEGER NOT NULL,
    total_payees            INTEGER NOT NULL DEFAULT 0,
    total_transactions      INTEGER NOT NULL DEFAULT 0,
    reportable_payees       INTEGER NOT NULL DEFAULT 0,
    report_status           VARCHAR(20) NOT NULL DEFAULT 'pending',
    submitted_at            TIMESTAMPTZ,
    ,
    CONSTRAINT unique_cesop_period UNIQUE (legal_entity_id, period_year, period_quarter)
);

-- Compliance Logs (Geo-Partitioned)
CREATE TABLE compliance_logs (
    id                      UUID PRIMARY KEY DEFAULT uuidv7(),
    transfer_id             UUID NOT NULL,
    screening_type          TEXT NOT NULL,
    result                  TEXT NOT NULL,
    risk_score              INTEGER,
    raw_response            JSONB,
    screened_by             TEXT,
    screened_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    compliance_region       TEXT GENERATED ALWAYS AS (
        (SELECT compliance_region FROM transfers WHERE id = transfer_id)
    ) STORED
) PARTITION BY LIST (compliance_region);

CREATE TABLE compliance_logs_id PARTITION OF compliance_logs FOR VALUES IN ('ID');
CREATE TABLE compliance_logs_eu PARTITION OF compliance_logs FOR VALUES IN ('EU');
CREATE TABLE compliance_logs_uk PARTITION OF compliance_logs FOR VALUES IN ('UK');
```

### Audit Trail (Immutable, Geo-Partitioned)

```sql
CREATE TABLE audit_trail (
    id                      UUID PRIMARY KEY DEFAULT uuidv7(),
    tenant_id               UUID,
    legal_entity_id         UUID,
    resource_type           VARCHAR(50) NOT NULL,
    resource_id             UUID NOT NULL,
    action                  VARCHAR(50) NOT NULL,
    actor_type              VARCHAR(20) NOT NULL,
    actor_id                VARCHAR(100) NOT NULL,
    before_state            JSONB,
    after_state             JSONB,
    ip_address              INET,
    request_id              VARCHAR(100),
    ,
    compliance_region       TEXT GENERATED ALWAYS AS (...) STORED
) PARTITION BY LIST (compliance_region);

-- Immutability Trigger
CREATE OR REPLACE FUNCTION prevent_audit_modification() RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Audit trail records are immutable';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_immutable BEFORE UPDATE OR DELETE ON audit_trail
FOR EACH ROW EXECUTE FUNCTION prevent_audit_modification();
```

---

## Netting Flow Diagram

```
┌──────────────────────────────────────────────────────────────────────────┐
│                         NETTING WINDOW (5 min)                           │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  Inbound Transfers (EUR → IDR)          Outbound Transfers (IDR → EUR)   │
│  ┌─────────────────────────┐            ┌─────────────────────────┐      │
│  │  Transfer 1: €10,000    │            │  Transfer A: Rp 50M     │      │
│  │  Transfer 2: €5,000     │            │  Transfer B: Rp 30M     │      │
│  │  Transfer 3: €8,000     │            │                         │      │
│  └──────────┬──────────────┘            └──────────┬──────────────┘      │
│             │                                      │                     │
│             ▼                                      ▼                     │
│       gross_inbound                          gross_outbound              │
│         €23,000                              Rp 80M (~€5,000)            │
│             │                                      │                     │
│             └──────────────┬───────────────────────┘                     │
│                            ▼                                             │
│                    ┌───────────────┐                                     │
│                    │   NETTING     │                                     │
│                    │   ENGINE      │                                     │
│                    └───────┬───────┘                                     │
│                            ▼                                             │
│                   net_amount: €18,000                                    │
│                   net_direction: inbound                                 │
│                   fx_saved: €150 (avoided double conversion)             │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## Redis Key Patterns

|Pattern|TTL|Purpose|
|---|---|---|
|`fx:lock:{quote_id}`|10min|Locked FX rate for quote|
|`fx:rate:{pair}`|1min|Cached FX rate|
|`rate_limit:{tenant_id}:{window}`|1min|Sliding window counter|
|`idempotency:{tenant_id}:{key}`|24h|Request deduplication|
|`netting:{tenant_id}:{corridor}`|5min|Active netting window|
|`balance:{wallet_id}`|30s|Cached wallet balance|

---

## Kafka Topics

|Topic|Partitions|Retention|Purpose|
|---|---|---|---|
|transfer.events|12|7 days|State changes, completions|
|compliance.events|6|30 days|Screening results, alerts|
|audit.trail|12|365 days|Immutable audit stream|
|reconciliation.alerts|3|30 days|FBO/Nostro mismatches|
|netting.executed|6|7 days|Netting completions|
|webhook.dlq|3|7 days|Failed webhook deliveries|

---

## Daily Reconciliation Flow

```
┌──────────────────────────────────────────────────────────────────────────┐
│                      DAILY RECONCILIATION PROCESS                        │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  06:00 UTC - Scheduled Job Starts                                        │
│                                                                          │
│  ┌─────────────────┐              ┌─────────────────┐                    │
│  │   TigerBeetle   │              │  Bank Statement │                    │
│  │   Balances      │              │  (SFTP/API)     │                    │
│  └────────┬────────┘              └────────┬────────┘                    │
│           │                                │                             │
│           ▼                                ▼                             │
│  ┌─────────────────────────────────────────────────────┐                 │
│  │              RECONCILIATION ENGINE                   │                │
│  │                                                      │                │
│  │  FBO Check:                                          │                │
│  │  SUM(TENANT_WALLET) == FBO Bank Statement?           │                │
│  │                                                      │                │
│  │  Nostro Check:                                       │                │
│  │  REGIONAL_SETTLEMENT == Nostro Bank Statement?       │                │
│  └──────────────────────┬───────────────────────────────┘                │
│                         │                                                │
│           ┌─────────────┴─────────────┐                                  │
│           ▼                           ▼                                  │
│  ┌─────────────────┐         ┌─────────────────┐                         │
│  │    MATCHED      │         │   DISCREPANCY   │                         │
│  │                 │         │                 │                         │
│  │ status: passed  │         │ → Kafka Alert   │                         │
│  │                 │         │ → Slack Notice  │                         │
│  │                 │         │ → Manual Review │                         │
│  └─────────────────┘         └─────────────────┘                         │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## Data Retention Policy

|Data Type|Hot (PG)|Warm|Cold|Total|
|---|---|---|---|---|
|Transfers|90 days|1 year|7 years|7 years|
|Audit trail|30 days|1 year|7 years|7 years|
|DHE records|3 years|7 years|-|7 years|
|CESOP reports|5 years|10 years|-|10 years|
|Reconciliation|90 days|2 years|7 years|7 years|
|FX rates|7 days|90 days|2 years|2 years|
|Compliance logs|90 days|1 year|7 years|7 years|

---

## Best Practices

### Query Optimization

```sql
-- ✅ Includes compliance_region (partition pruning)
SELECT * FROM transfers 
WHERE compliance_region = 'ID' 
  AND tenant_id = 'tenant_xyz'
  AND status = 'completed';

-- ❌ Missing compliance_region (scans all partitions)
SELECT * FROM transfers 
WHERE tenant_id = 'tenant_xyz'
  AND status = 'completed';
```

### Temporal Constraint Usage

```sql
-- Query current pricing
SELECT * FROM pricing_policies
WHERE tenant_id = 'tenant_abc'
  AND valid_period @> NOW()::timestamptz;
```

---

## Monitoring Queries

```sql
-- Partition sizes
SELECT tablename, pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables WHERE tablename LIKE 'transfers_%';

-- Active netting groups
SELECT corridor, COUNT(*) as open_groups, SUM(fx_saved) as total_saved
FROM netting_groups WHERE status = 'open' GROUP BY corridor;

-- Reconciliation status
SELECT le.code, rr.reconciliation_date, rr.fbo_match, rr.nostro_match
FROM reconciliation_reports rr
JOIN legal_entities le ON rr.legal_entity_id = le.id
WHERE rr.reconciliation_date >= CURRENT_DATE - INTERVAL '7 days';
```