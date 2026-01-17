**Goal**: Production-ready demo untuk portfolio — B2B Cross-border Payment Rails (EU/UK ↔ Indonesia)

**Philosophy**: Aggressive timeline (Elon's Law) — target 8 minggu, accept 10-12 minggu reality.

---

## Rules & Conventions

### Time Estimation

|Symbol|Meaning|Example|
|---|---|---|
|`h`|Hours (focused coding time)|`4h` = 4 jam coding|
|`~Xh`|Approximate hours|`~23h` = sekitar 23 jam|
|`X days`|Working days|`4 days` = 4 hari kerja|

**Conversion**:

- 1 working day = **7 hours** focused coding
- Total hours ÷ 7 = working days
- Contoh: 29h ÷ 7 = ~4 days

**"Focused coding"** = waktu aktif nulis kode, tidak termasuk:

- Meeting, standup
- Research, baca dokumentasi
- Debugging unexpected issues
- Code review

### Priority Labels

|Label|Meaning|Rule|
|---|---|---|
|**P0**|Critical|Must complete dalam week itu. Blocker untuk minggu berikutnya.|
|**P1**|Important|Should complete. Boleh overflow ke buffer week jika perlu.|
|**P2**|Nice-to-have|Complete if time permits. Skip jika mepet.|

**Weekly target**: Selesaikan semua P0 + 80% P1.

### Demo Milestones

Setiap akhir minggu harus ada **demo yang bisa direkam**:

- Max 5 menit per demo
- Harus menunjukkan **working feature**, bukan slideshow
- Format: screen recording + voice narration
- Jika demo tidak bisa dilakukan = week tidak complete

### Definition of Done

Task dianggap selesai jika:

1. ✅ Code committed + pushed
2. ✅ Unit tests passing (>80% coverage untuk P0)
3. ✅ Bisa di-demo
4. ✅ Tidak ada known critical bugs

### Buffer Usage

**Week 9-10** adalah buffer untuk:

- Overflow dari P0/P1 yang tidak selesai
- Bug fixes dari testing
- Documentation
- Demo recording & editing

**Rule**: Jika masuk buffer week, tidak boleh ada feature baru. Focus hanya pada completion.

### Aggressive Timeline Principles

1. **Parkinson's Law**: Kerjaan akan memenuhi waktu yang tersedia. Compress timeline = force efficiency.
    
2. **Elon's Law**: Set deadline yang "impossible", bahkan jika terlambat, progress tetap lebih maju dari timeline konservatif.
    
3. **First Principles**: Setiap task harus dijustifikasi. Jika tidak essential untuk demo, cut.
    
4. **Velocity > Perfection**: Working code yang 80% sempurna > perfect code yang tidak selesai.
    

### Red Flags (Stop & Reassess)

🚨 Stop dan reassess jika:

- P0 task belum selesai di akhir minggu
- 2 minggu berturut-turut miss deadline
- Demo tidak bisa dilakukan
- Blocked oleh external dependency >2 hari

---

## Timeline Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      8-WEEK AGGRESSIVE DEVELOPMENT PLAN                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  PHASE 1: Core Ledger (Week 1-2)                                            │
│  ├── W1: PostgreSQL + Basic Ledger + Single Currency                        │
│  └── W2: TigerBeetle Cluster + Multi-Currency + Atomic FX                   │
│                                                                             │
│  PHASE 2: Business Logic (Week 3-4)                                         │
│  ├── W3: API + Auth + FX Engine                                             │
│  └── W4: Compliance + Payment Rails                                         │
│                                                                             │
│  PHASE 3: Operations (Week 5-6)                                             │
│  ├── W5: Webhooks + Real-time Tracking                                      │
│  └── W6: Reconciliation + Netting Engine + E2E Test Skeleton                │
│                                                                             │
│  PHASE 4: Production (Week 7-8)                                             │
│  ├── W7: Regional Security (FAPI 2.0 + SNAP)                                │
│  └── W8: Dashboard + Load Testing + Hardening                               │
│                                                                             │
│  BUFFER: Week 9-10 (for reality check)                                      │
│  └── Overflow, bug fixes, documentation, demo recording, CHAOS TESTING      │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Phase 1: Core Ledger (Week 1-2)

### Week 1: PostgreSQL + Basic Ledger

**Objective**: Database foundation dengan geo-partitioning + single-currency ledger

**Tasks**:

|Priority|Task|Est. Hours|
|---|---|---|
|P0|PostgreSQL 18 schema deployment|4h|
|P0|Geo-partitioned transfers table (ID/EU/UK)|3h|
|P0|Row-level security policies (OJK/GDPR/FCA)|2h|
|P0|TigerBeetle single node setup|2h|
|P1|Basic account types (TENANT_WALLET, FEE_REVENUE)|3h|
|P1|Simple EUR transfer (no FX)|4h|
|P1|Tenant + Legal Entity tables|3h|
|P2|Pricing policies with temporal constraints (PG 18 EXCLUDE)|2h|

**Total**: ~23 hours (3 days focused work)

**Demo Milestone**:

```
✅ WEEK 1 DEMO (2 min):
"Create tenant → Assign to KOVRA_EU → 
Fund EUR wallet €10,000 → Transfer €1,000 → 
Check partition (transfers_eu) → 
RLS blocks cross-region query → 
Show temporal constraint blocking overlapping pricing"
```

**Acceptance Criteria**:

- [ ] `SELECT * FROM transfers WHERE compliance_region = 'ID'` returns empty for EU-only tenant
- [ ] Temporal constraint rejects overlapping pricing periods
- [ ] TigerBeetle balance matches PostgreSQL cached_balance

---

### Week 2: Multi-Currency + Atomic FX

**Objective**: TigerBeetle cluster dengan atomic linked transfers

**Tasks**:

|Priority|Task|Est. Hours|
|---|---|---|
|P0|TigerBeetle 3-node cluster (docker-compose)|3h|
|P0|Multi-currency ledgers (EUR, GBP, IDR, SEK, DKK)|2h|
|P0|All account types (6 types per spec)|3h|
|P0|Atomic linked transfers (5-step FX chain)|6h|
|P1|Wallet service (TopUp, Hold, Release, Capture)|4h|
|P1|Balance cache sync (TigerBeetle → PostgreSQL)|2h|
|P1|Overdraft prevention + validation|2h|
|P2|Account ID encoding (128-bit structure)|1h|

**Total**: ~23 hours (3 days focused work)

**Linked Transfer Chain**:

```
┌──────────────────────────────────────────────────────────────────────────┐
│              5-STEP ATOMIC FX TRANSFER (EUR → IDR)                       │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  Step 1: TENANT_WALLET_EUR  ──debit──►  PENDING_OUTBOUND_EUR             │
│  Step 2: PENDING_OUTBOUND   ──convert─► FX_SETTLEMENT_EUR                │
│  Step 3: FX_SETTLEMENT_EUR  ──fx─────►  FX_SETTLEMENT_IDR                │
│  Step 4: FX_SETTLEMENT_IDR  ──fee────►  FEE_REVENUE_IDR                  │
│  Step 5: FX_SETTLEMENT_IDR  ──credit─►  REGIONAL_SETTLEMENT_ID           │
│                                                                          │
│  All 5 steps commit atomically or rollback entirely                      │
└──────────────────────────────────────────────────────────────────────────┘
```

**Demo Milestone**:

```
✅ WEEK 2 DEMO (3 min):
"EUR → IDR transfer €5,000 → 
Show 5-step linked chain in TigerBeetle → 
Simulate failure at step 4 → 
All 5 steps rollback → 
Balance unchanged → Retry succeeds"
```

**Acceptance Criteria**:

- [ ] Linked transfer chain executes atomically
- [ ] Partial failure triggers full rollback
- [ ] Hold → Capture flow works correctly
- [ ] Multi-currency balances accurate across 3 nodes

---

## Phase 2: Business Logic (Week 3-4)

### Week 3: API + Authentication + FX Engine

**Objective**: Production API dengan FX rate aggregation + tier-based policies

**Tasks**:

|Priority|Task|Est. Hours|
|---|---|---|
|P0|HTTP server setup (Chi router)|2h|
|P0|API key generation + hashing (bcrypt)|2h|
|P0|Tenant context middleware|2h|
|P0|Rate limiting per tier (sliding window, Redis)|3h|
|P0|Idempotency handling (Redis, 24h TTL)|2h|
|P0|Pricing policies service (tier-based, temporal)|3h|
|P0|Limit policies service (tier-based)|2h|
|P1|FX rate fetching (mock providers)|3h|
|P1|VWAP calculation + outlier removal|2h|
|P1|Quote API with rate locking (10min Redis TTL)|3h|
|P1|Corridor netting service (5-min window)|4h|
|P1|Request/response logging (structured)|2h|
|P2|OpenAPI spec generation|2h|

**Total**: ~32 hours (4-5 days focused work)

**Tier System (Policies, NOT Identity)**:

```
┌──────────────────────────────────────────────────────────────────────────┐
│                    TIER SYSTEM ARCHITECTURE                              │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ❌ WRONG: Tier in tenants table                                         │
│  ┌────────────────────────────────────────────────────────────────────┐  │
│  │ tenants: { id, name, tier: 'enterprise', margin: 30 }              │  │
│  └────────────────────────────────────────────────────────────────────┘  │
│                                                                          │
│  ✅ CORRECT: Tier in policies (separate tables)                          │
│  ┌────────────────────────────────────────────────────────────────────┐  │
│  │ tenants:          { id, name, tenant_kind, parent_tenant_id }      │  │
│  │ pricing_policies: { tenant_id, fx_margin_bps, valid_period }       │  │
│  │ limit_policies:   { tenant_id, rpm, daily_limit, per_transfer }    │  │
│  └────────────────────────────────────────────────────────────────────┘  │
│                                                                          │
│  WHY: Policies can change without touching identity.                     │
│       Full audit trail. Temporal constraints. Corridor overrides.        │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

**Tier Configurations**:

```
┌──────────────────────────────────────────────────────────────────────────┐
│                         TIER DEFINITIONS                                 │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  PRICING (stored in pricing_policies):                                   │
│  ┌─────────────┬─────────────┬─────────────────────────────────────────┐ │
│  │ Tier        │ FX Margin   │ Corridor Override Example               │ │
│  ├─────────────┼─────────────┼─────────────────────────────────────────┤ │
│  │ Starter     │ 150 bps     │ -                                       │ │
│  │ Growth      │ 80 bps      │ EUR_IDR: 60 bps (high volume)           │ │
│  │ Enterprise  │ 30 bps      │ EUR_IDR: 25 bps, GBP_IDR: 30 bps        │ │
│  └─────────────┴─────────────┴─────────────────────────────────────────┘ │
│                                                                          │
│  LIMITS (stored in limit_policies):                                      │
│  ┌─────────────┬────────┬─────────────┬──────────────┐                   │
│  │ Tier        │ RPM    │ Daily Limit │ Per-Transfer │                   │
│  ├─────────────┼────────┼─────────────┼──────────────┤                   │
│  │ Starter     │ 100    │ $10,000     │ $1,000       │                   │
│  │ Growth      │ 500    │ $100,000    │ $10,000      │                   │
│  │ Enterprise  │ 2,000  │ $1,000,000  │ $100,000     │                   │
│  └─────────────┴────────┴─────────────┴──────────────┘                   │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

**Tenant Hierarchy**:

```
┌──────────────────────────────────────────────────────────────────────────┐
│                       TENANT HIERARCHY                                   │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  tenant_kind:                                                            │
│  ├── 'platform'  → Tokopedia, Bukalapak (can create sub-tenants)        │
│  ├── 'seller'    → Toko under platform (parent_tenant_id = platform)    │
│  └── 'direct'    → Corporate client (no hierarchy)                      │
│                                                                          │
│  EXAMPLE:                                                                │
│  ┌─────────────────────────────────────────────────────────────────┐     │
│  │  Tokopedia (platform, parent_id: NULL)                          │     │
│  │  ├── pricing: 30 bps                                            │     │
│  │  ├── can_create_subtenants: true                                │     │
│  │  │                                                              │     │
│  │  └── Toko Sepatu (seller, parent_id: tokopedia)                 │     │
│  │      └── pricing: 50 bps (platform adds 20 bps markup)          │     │
│  └─────────────────────────────────────────────────────────────────┘     │
│                                                                          │
│  RULE: Seller margin >= Platform margin (platform can't undercut self)  │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

**Revenue Share Model**:

```
┌──────────────────────────────────────────────────────────────────────────┐
│                      REVENUE SHARE (Platform → Seller)                   │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  Transfer: €10,000 from EU buyer to Toko Sepatu (under Tokopedia)        │
│                                                                          │
│  Fee Calculation:                                                        │
│  ├── FX margin (seller rate): 0.5% = €50                                │
│  ├── Transfer fee: 0.8% = €80                                           │
│  └── Total fee: €130                                                    │
│                                                                          │
│  Revenue Split (configurable in platform settings):                      │
│  ┌─────────────────┬─────────┬──────────┐                                │
│  │ Party           │ Share   │ Amount   │                                │
│  ├─────────────────┼─────────┼──────────┤                                │
│  │ Tokopedia       │ 20%     │ €26      │                                │
│  │ Your Platform   │ 80%     │ €104     │                                │
│  └─────────────────┴─────────┴──────────┘                                │
│                                                                          │
│  Stored in: tenants.settings.revenue_share_model                         │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

**Corridor Netting**:

```
┌──────────────────────────────────────────────────────────────────────────┐
│                       CORRIDOR NETTING                                   │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  Window: 5 minutes                                                       │
│  Target Savings: 50-60% FX reduction                                     │
│                                                                          │
│  EXAMPLE (Tokopedia tenant):                                             │
│  ├── INBOUND:  €50,000 (EU buyers → IDR)                                │
│  ├── OUTBOUND: €20,000 (IDR → EU suppliers)                             │
│  │                                                                       │
│  │  Without netting: Convert €50k + €20k = €70k FX                      │
│  │  With netting:    Convert €30k only (net position)                   │
│  │  Savings:         €40k × 0.8% margin = €320 saved (57%)              │
│  │                                                                       │
│  └── RULE: Only convert NET position, not GROSS                         │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

**Demo Milestone**:

```
✅ WEEK 3 DEMO (4 min):
"Create tenant (platform) → Create sub-tenant (seller) →
Show pricing policy: platform 30 bps, seller 50 bps →
Generate API key → Make requests →
Show rate limit headers (X-RateLimit-Remaining) →
Hit limit (Growth: 500 RPM) → Get 429 →
Create FX quote EUR→IDR →
Rate locked for 10 min →
Show netting: €50k in, €20k out → Net €30k converted →
Savings displayed: €320 (57%)"
```

**Acceptance Criteria**:

- [ ] Tier stored in policies, NOT in tenants table
- [ ] Platform can create sellers with margin >= platform margin
- [ ] Rate limiting respects tier from limit_policies
- [ ] Idempotency key prevents duplicate transfers
- [ ] FX quote rate locked in Redis (10 min TTL)
- [ ] Corridor netting calculates correct net position
- [ ] Revenue share split calculated correctly
- [ ] API key hash not reversible (bcrypt)

---

### Week 4: Compliance + Payment Rails + Validation

**Objective**: IBAN validation + automated screening + multi-rail routing

**Tasks**:

|Priority|Task|Est. Hours|
|---|---|---|
|P0|IBAN validation service (MOD-97, country specs)|3h|
|P0|BIC lookup + bank directory|2h|
|P0|OFAC SDN list loader + fuzzy match (pg_trgm)|4h|
|P0|Sanctions screening (EU/UK lists)|2h|
|P0|Risk score calculation (0-100)|3h|
|P1|Auto-approve/flag/reject logic|2h|
|P1|Mock rail adapters (SEPA, BI-FAST, UK FPS)|4h|
|P1|Routing engine (cost vs speed)|3h|
|P1|ISO 20022 message stub (pacs.008)|2h|
|P1|Compliance logs (geo-partitioned)|2h|
|P2|Velocity checks (amount, frequency)|2h|

**Total**: ~29 hours (4 days focused work)

**Risk Score Matrix**:

```
┌──────────────────────────────────────────────────────────────────────────┐
│                        RISK SCORING ENGINE                               │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  Factor                    │ Weight │ Score Range                        │
│  ──────────────────────────┼────────┼────────────────────────────────    │
│  OFAC/Sanctions match      │ 40%    │ 0 (clear) - 100 (exact match)      │
│  PEP status                │ 20%    │ 0 (none) - 50 (direct PEP)         │
│  Transaction velocity      │ 15%    │ 0 (normal) - 30 (anomaly)          │
│  Amount deviation          │ 15%    │ 0 (typical) - 25 (10x average)     │
│  Country risk              │ 10%    │ 0 (low) - 20 (high-risk)           │
│                                                                          │
│  DECISION THRESHOLDS:                                                    │
│  ├── Score 0-30:   AUTO_APPROVE                                          │
│  ├── Score 31-70:  MANUAL_REVIEW                                         │
│  └── Score 71-100: AUTO_REJECT                                           │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

**Rail Routing Logic**:

```
┌──────────────────────────────────────────────────────────────────────────┐
│                      SMART RAIL ROUTING                                  │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  Input: EUR → IDR, €5,000                                                │
│                                                                          │
│  Available Rails:                                                        │
│  ┌─────────────┬─────────┬──────────┬────────────┐                       │
│  │ Rail        │ Cost    │ Speed    │ Available  │                       │
│  ├─────────────┼─────────┼──────────┼────────────┤                       │
│  │ BI-FAST     │ €0.35   │ 30 sec   │ ✓ (< €25K) │                       │
│  │ RTGS        │ €5.00   │ 2 hours  │ ✓          │                       │
│  │ SWIFT       │ €25.00  │ 1-2 days │ ✓          │                       │
│  └─────────────┴─────────┴──────────┴────────────┘                       │
│                                                                          │
│  Decision: BI-FAST (lowest cost, fastest, within limit)                  │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

**Demo Milestone**:

```
✅ WEEK 4 DEMO (4 min):
"Validate IBAN DE89370400440532013000 → Valid + BIC lookup →
Invalid IBAN rejected (MOD-97 fail) →
Transfer to clean recipient → Risk score: 12 → AUTO_APPROVE →
Transfer to OFAC match → Risk score: 85 → AUTO_REJECT →
Show compliance dashboard →
EUR→IDR routing: BI-FAST selected (€0.35, 30sec) →
Mock adapter simulates 5-30s settlement"
```

**Acceptance Criteria**:

- [ ] IBAN validation catches invalid checksums
- [ ] BIC lookup returns bank info from directory
- [ ] OFAC fuzzy match detects variations (e.g., "Al-Qaeda" vs "Al Qaeda")
- [ ] Risk score correctly weighted
- [ ] Mock rail adapters simulate realistic latency + failures
- [ ] Routing selects optimal rail
- [ ] Compliance logs in correct partition

---

## Phase 3: Operations (Week 5-6)

### Week 5: Webhooks + Real-time Tracking

**Objective**: Reliable delivery + WebSocket updates

**Tasks**:

|Priority|Task|Est. Hours|
|---|---|---|
|P0|Event emission on state changes|2h|
|P0|Webhook worker pool (10 goroutines)|3h|
|P0|JWS signature generation|2h|
|P0|Exponential backoff (1s → 32s, max 10 retries)|2h|
|P0|Dead letter queue (River)|2h|
|P1|PostgreSQL LISTEN/NOTIFY|2h|
|P1|WebSocket server (gorilla/websocket)|3h|
|P1|Connection management per tenant|2h|
|P1|Transaction timeline API|2h|
|P2|Delivery dashboard API|2h|

**Total**: ~22 hours

**Webhook Retry Strategy**:

```
┌──────────────────────────────────────────────────────────────────────────┐
│                    WEBHOOK DELIVERY ENGINE                               │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  Attempt │ Delay    │ Total Elapsed │ Action                             │
│  ────────┼──────────┼───────────────┼────────────────────────────────    │
│  1       │ 0s       │ 0s            │ Initial delivery                   │
│  2       │ 1s       │ 1s            │ First retry                        │
│  3       │ 2s       │ 3s            │ Second retry                       │
│  4       │ 4s       │ 7s            │ Third retry                        │
│  5       │ 8s       │ 15s           │ Fourth retry                       │
│  6       │ 16s      │ 31s           │ Fifth retry                        │
│  7       │ 32s      │ 63s           │ Sixth retry                        │
│  8       │ 32s      │ 95s           │ Seventh retry (capped)             │
│  9       │ 32s      │ 127s          │ Eighth retry                       │
│  10      │ 32s      │ 159s          │ Final retry → DLQ if fails         │
│                                                                          │
│  JWS Signature: RS256, kid in header, 5-min expiry                       │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

**WebSocket Message Flow**:

```
┌──────────────────────────────────────────────────────────────────────────┐
│                    REAL-TIME TRACKING                                    │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  Client                    Server                      Database          │
│    │                         │                            │              │
│    │──── WS Connect ────────►│                            │              │
│    │◄─── Auth OK ────────────│                            │              │
│    │                         │                            │              │
│    │──── Subscribe ─────────►│                            │              │
│    │     {transfer_id}       │──── LISTEN ───────────────►│              │
│    │                         │                            │              │
│    │                         │◄─── NOTIFY ────────────────│              │
│    │◄─── Status Update ──────│     (status changed)       │              │
│    │     {status: processing}│                            │              │
│    │                         │                            │              │
│    │◄─── Status Update ──────│◄─── NOTIFY ────────────────│              │
│    │     {status: completed} │                            │              │
│    │                         │                            │              │
│                                                                          │
│  Average E2E time: 30-45 seconds                                         │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

**Demo Milestone**:

```
✅ WEEK 5 DEMO (4 min):
"Create transfer → Open WebSocket →
Receive: created → validating → processing → completed →
Timeline shows 38 seconds E2E →
Webhook fired → Endpoint down →
Retry: 1s, 2s, 4s, 8s → Endpoint recovers →
Delivered with JWS signature"
```

**Acceptance Criteria**:

- [ ] WebSocket receives updates within 100ms of state change
- [ ] Webhook retries with correct exponential backoff
- [ ] JWS signature verifiable with public key
- [ ] DLQ captures failed deliveries after 10 attempts

---

### Week 6: Reconciliation + Netting Engine + E2E Test Skeleton

**Objective**: Daily recon + corridor netting + early E2E test foundation

**Tasks**:

| Priority | Task                                          | Est. Hours |
| -------- | --------------------------------------------- | ---------- |
| P0       | FBO reconciliation (TigerBeetle vs mock bank) | 4h         |
| P0       | Nostro reconciliation                         | 2h         |
| P0       | Discrepancy detection + alerting              | 2h         |
| P0       | Reconciliation report generation              | 2h         |
| P0       | **E2E test skeleton (mock externals)**        | 3h         |
| P1       | Corridor netting service                      | 4h         |
| P1       | Netting window management (5-min Redis)       | 2h         |
| P1       | Net position calculation                      | 2h         |
| P1       | Netting savings calculation                   | 2h         |
| P2       | Settlement file generation (CSV)              | 2h         |
| P2       | River scheduled jobs (daily 06:00 UTC)        | 2h         |

**Total**: ~27 hours

**E2E Test Skeleton** (NEW):

```go
// e2e/money_path_test.go

func TestFullTransferFlow(t *testing.T) {
    // Week 6-7: Mock all externals
    // Week 8+: Gradually replace with real calls
    
    // 1. Create tenant + fund wallet
    tenant := createTenant(t, "test_corp")
    fundWallet(t, tenant.ID, "EUR", 10000_00) // €10,000 in cents
    
    // 2. Create transfer EUR → IDR
    transfer := createTransfer(t, tenant.ID, TransferRequest{
        Amount:       5000_00,
        SourceCcy:    "EUR",
        DestCcy:      "IDR",
        Beneficiary:  mockBeneficiary(),
    })
    
    // 3. Verify ledger state
    assertBalance(t, tenant.ID, "EUR", 5000_00) // €5,000 remaining
    assertTransferStatus(t, transfer.ID, "completed")
    
    // 4. Verify compliance logged
    assertComplianceLog(t, transfer.ID, "AUTO_APPROVE")
    
    // 5. Verify webhook delivered
    assertWebhookDelivered(t, transfer.ID)
}
```

**Netting Engine Flow**:

```
┌──────────────────────────────────────────────────────────────────────────┐
│                     CORRIDOR NETTING ENGINE                              │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  Window: 5 minutes │ Corridor: EUR_IDR                                   │
│                                                                          │
│  INBOUND (EUR → IDR):           OUTBOUND (IDR → EUR):                    │
│  ├── Transfer 1: €10,000        ├── Transfer A: Rp 80M (~€5,000)         │
│  ├── Transfer 2: €5,000         └── Transfer B: Rp 48M (~€3,000)         │
│  └── Transfer 3: €8,000                                                  │
│      ─────────────                   ─────────────                       │
│      Total: €23,000                  Total: €8,000                       │
│                                                                          │
│  NETTING CALCULATION:                                                    │
│  ┌─────────────────────────────────────────────────────────┐             │
│  │ Gross Volume:    €31,000 (€23K + €8K)               │                 │
│  │ Net Position:    €15,000 (€23K - €8K) INBOUND       │                 │
│  │ FX Conversions:  1 (instead of 5)                   │                 │
│  │ FX Saved:        €256 (51.6% reduction)             │                 │
│  └─────────────────────────────────────────────────────────┘             │
│                                                                          │
│  Without netting: 5 FX conversions @ €0.32 margin each = €496            │
│  With netting:    1 FX conversion  @ €0.32 margin      = €240            │
│  Savings:         €256 (51.6%)                                           │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

**Demo Milestone**:

```
✅ WEEK 6 DEMO (4 min):
"Tenant has €50K inbound, €20K outbound in 5-min window →
Netting executes → Net: €30K converted →
Savings: €320 (53% reduction) →
Trigger daily reconciliation →
FBO: TigerBeetle €5M = Bank €5M ✓ →
Nostro: Match ✓ →
Generate settlement CSV →
Run E2E test suite → All green"
```

**Acceptance Criteria**:

- [ ] Netting correctly calculates net position
- [ ] Savings percentage accurate
- [ ] Reconciliation detects intentional mismatch
- [ ] Alert fires on discrepancy
- [ ] E2E test skeleton runs with mock externals

---

## Phase 4: Production (Week 7-8)

### Week 7: Regional Security

**Objective**: FAPI 2.0 (EU) + SNAP (Indonesia) authentication

**Tasks**:

|Priority|Task|Est. Hours|
|---|---|---|
|P0|FAPI 2.0 client implementation|6h|
|P0|PAR (Pushed Authorization Request)|2h|
|P0|PKCE + DPoP token binding|3h|
|P0|private_key_jwt client authentication|2h|
|P1|SNAP OAuth 2.0 implementation|3h|
|P1|X-SIGNATURE generation (HMAC-SHA512)|2h|
|P1|X-TIMESTAMP validation|1h|
|P1|mTLS setup (self-signed for demo)|2h|
|P2|Token refresh handling|2h|

**Total**: ~23 hours

**FAPI 2.0 Flow**:

```
┌──────────────────────────────────────────────────────────────────────────┐
│                      FAPI 2.0 AUTHORIZATION FLOW                         │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  Kovra                         Auth Server                    Bank       │
│    │                               │                            │        │
│    │── PAR Request ───────────────►│                            │        │
│    │   (client_assertion JWT)      │                            │        │
│    │◄── request_uri ───────────────│                            │        │
│    │                               │                            │        │
│    │── Authorize ─────────────────►│                            │        │
│    │   (request_uri + PKCE)        │── User Auth ──────────────►│        │
│    │                               │◄── Consent ────────────────│        │
│    │◄── code ──────────────────────│                            │        │
│    │                               │                            │        │
│    │── Token Request ─────────────►│                            │        │
│    │   (code + code_verifier)      │                            │        │
│    │◄── access_token + DPoP ───────│                            │        │
│    │                               │                            │        │
│    │── Payment Request ────────────────────────────────────────►│        │
│    │   (Authorization: DPoP + token)                            │        │
│    │◄── Payment Initiated ──────────────────────────────────────│        │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

**SNAP Authentication**:

```
┌──────────────────────────────────────────────────────────────────────────┐
│                      SNAP (BANK INDONESIA) AUTH                          │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  Headers Required:                                                       │
│  ├── Authorization: Bearer {access_token}                                │
│  ├── X-TIMESTAMP: 2025-01-15T10:30:00+07:00                              │
│  ├── X-SIGNATURE: {HMAC-SHA512 signature}                                │
│  ├── X-PARTNER-ID: {partner_id}                                          │
│  ├── X-EXTERNAL-ID: {unique_request_id}                                  │
│  └── CHANNEL-ID: {channel_id}                                            │
│                                                                          │
│  X-SIGNATURE = HMAC-SHA512(                                              │
│      key: client_secret,                                                 │
│      data: HTTP_METHOD + ":" + ENDPOINT + ":" + ACCESS_TOKEN + ":" +     │
│            SHA256(REQUEST_BODY) + ":" + TIMESTAMP                        │
│  )                                                                       │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

**Demo Milestone**:

```
✅ WEEK 7 DEMO (4 min):
"EU Bank connection:
PAR request with client_assertion →
User authenticates → Token with DPoP binding →
Payment initiated (FAPI 2.0 compliant) →

Indonesia connection:
SNAP OAuth → Generate X-SIGNATURE →
BI-FAST transfer sent → Signature verified"
```

**Acceptance Criteria**:

- [ ] FAPI 2.0 flow completes with DPoP
- [ ] SNAP signature validates correctly
- [ ] mTLS handshake succeeds
- [ ] Token refresh works before expiry

---

### Week 8: Dashboard + Load Testing + Hardening

**Objective**: Operations UI + 5K TPS validation

**Tasks**:

|Priority|Task|Est. Hours|
|---|---|---|
|P0|React dashboard setup (Vite + TanStack Query)|3h|
|P0|Real-time metrics (TPS, success rate, latency)|3h|
|P0|Transfer list with filters|3h|
|P0|k6 load test script|3h|
|P0|Performance tuning (connection pools, indexes)|4h|
|P1|Tenant management UI|2h|
|P1|Compliance review queue|2h|
|P1|WebSocket integration (live updates)|2h|
|P1|Security hardening checklist|3h|
|P2|Grafana dashboard setup|2h|

**Total**: ~27 hours

**Dashboard Layout**:

```
┌──────────────────────────────────────────────────────────────────────────┐
│  KOVRA ADMIN DASHBOARD                                         [logout] │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐      │
│  │  TPS        │  │  Success    │  │  P95        │  │  Pending    │      │
│  │  2,847      │  │  99.82%     │  │  35ms       │  │  12         │      │
│  │  ▲ +12%     │  │  ▲ +0.1%    │  │  ▼ -5ms     │  │  reviews    │      │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘      │
│                                                                          │
│  TRANSFERS                                          [+ New Transfer]     │
│  ┌────────────────────────────────────────────────────────────────────┐  │
│  │ ID          │ Amount      │ Route    │ Status    │ Time           │  │
│  ├─────────────┼─────────────┼──────────┼───────────┼────────────────┤  │
│  │ txn_abc123  │ €5,000→IDR  │ BI-FAST  │ completed │ 38s            │  │
│  │ txn_def456  │ £2,000→IDR  │ BI-FAST  │ processing│ 12s...         │  │
│  │ txn_ghi789  │ €10,000→IDR │ SWIFT    │ pending   │ compliance     │  │
│  └─────────────┴─────────────┴──────────┴───────────┴────────────────┘  │
│                                                                          │
│  COMPLIANCE QUEUE (3 pending)                                            │
│  ┌────────────────────────────────────────────────────────────────────┐  │
│  │ txn_ghi789 │ Risk: 65 │ Flag: Velocity anomaly │ [Approve] [Reject]│  │
│  └────────────────────────────────────────────────────────────────────┘  │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

**Load Test Targets**:

```
┌──────────────────────────────────────────────────────────────────────────┐
│                      PERFORMANCE TARGETS                                 │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  Metric              │ Target      │ Acceptable  │ Current              │
│  ────────────────────┼─────────────┼─────────────┼────────────────────  │
│  TPS                 │ 5,000       │ 3,000       │ TBD                  │
│  P50 Latency         │ < 15ms      │ < 25ms      │ TBD                  │
│  P95 Latency         │ < 50ms      │ < 75ms      │ TBD                  │
│  P99 Latency         │ < 100ms     │ < 150ms     │ TBD                  │
│  Success Rate        │ > 99.5%     │ > 99.0%     │ TBD                  │
│  Webhook Delivery    │ > 99%       │ > 98%       │ TBD                  │
│                                                                          │
│  k6 Script: scripts/k6/load-test.js                                      │
│  Duration: 5 min warmup → 10 min sustained → 5 min cooldown              │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

**Demo Milestone**:

```
✅ WEEK 8 DEMO - FINAL (5 min):
"Dashboard overview: Live TPS 2,847, Success 99.82% →
Filter transfers by status →
View transfer timeline (38 sec E2E) →
Approve compliance alert →
k6 load test: Ramp to 5,000 TPS →
P50: 12ms, P95: 35ms, P99: 48ms →
Zero partial failures →
Grafana shows healthy metrics"
```

**Acceptance Criteria**:

- [ ] Dashboard loads within 2 seconds
- [ ] Real-time updates via WebSocket
- [ ] k6 achieves target TPS
- [ ] No memory leaks during sustained load

---

## Buffer: Week 9-10

**Purpose**: Reality check — overflow, bugs, documentation, **CHAOS TESTING**

**Tasks**:

|Category|Items|Est. Hours|
|---|---|---|
|Overflow|Any incomplete P0/P1 from previous weeks|Variable|
|Bugs|Critical/High bugs from testing|Variable|
|Documentation|README, API docs, architecture diagrams|8h|
|Demo|Record 15-min video walkthrough|4h|
|Polish|UI fixes, error messages, edge cases|6h|
|**Chaos Testing**|See below|14h|

### Chaos & Failure Testing (NEW)

|Day|Focus|Tasks|Hours|
|---|---|---|---|
|Day 1|Network chaos|Inject 200ms latency antar service via `toxiproxy`, verify timeout handling|4h|
|Day 1|Pod kill|Random kill TigerBeetle node mid-transaction, verify rollback|3h|
|Day 2|Double idempotency|Fire same request 10x concurrent, verify single execution|3h|
|Day 2|Database failover|Kill primary PG, verify read replica promotion|4h|

**Tools**:

- `toxiproxy` untuk network chaos
- `kubectl delete pod --force` untuk pod kill
- Custom script untuk concurrent idempotency testing

**Acceptance Criteria**:

- [ ] System recovers from 200ms network latency without data loss
- [ ] TigerBeetle node kill triggers proper rollback
- [ ] 10x concurrent identical requests = 1 execution
- [ ] PG failover completes < 30 seconds

---

## Architectural Decision Records (ADR)

Buat folder `docs/adr/` dengan file berikut:

|ADR|Title|Key Reasoning|
|---|---|---|
|001|TigerBeetle over PostgreSQL for ledger|Financial-grade atomicity, 1M+ TPS benchmark, linked transfers|
|002|Policy tables terpisah dari identity|Temporal constraints, audit trail, corridor overrides tanpa touching tenant|
|003|Geo-partitioned transfers|OJK/GDPR/FCA compliance, data residency by design|
|004|River over Temporal for job queue|Simpler ops, PG-native, sufficient for webhook retry pattern|
|005|VWAP dengan outlier removal untuk FX|Reduce manipulation risk, statistical soundness|
|006|5-step linked transfer chain|Atomic FX + fee deduction, single failure = full rollback|
|007|Corridor netting window 5 menit|Balance antara savings (53%) dan settlement delay|

**ADR Template**:

```markdown
# ADR-001: TigerBeetle over PostgreSQL for Ledger

## Status
Accepted

## Context
Need atomic multi-currency transfers with guaranteed consistency.

## Decision
Use TigerBeetle as primary ledger, PostgreSQL for metadata/audit.

## Consequences
- Pro: Financial-grade atomicity, linked transfers
- Pro: 1M+ TPS benchmark
- Con: Additional operational complexity
- Con: Less mature ecosystem
```

---

## Interview Storytelling Prep

### Q: Apa bagian tersulit & kenapa?

> "Atomic 5-step linked transfer. Designing the chain was straightforward — the hard part was handling partial failures correctly. TigerBeetle's linked transfers helped, tapi edge case seperti 'step 3 succeeds, step 4 times out but actually succeeded on server' butuh careful idempotency design. Solusinya: every step punya unique transfer_id yang deterministic dari parent transfer, jadi retry safe."

### Q: Apa yang akan kamu ubah kalau ini production beneran?

> "Tiga hal:
> 
> 1. **Real bank adapters** — sekarang mock, production butuh certified connection ke SWIFT, SEPA Instant, BI-FAST. Ini 2-3 bulan sendiri per rail.
> 2. **HSM untuk signing** — saat ini private keys di env vars, production harus AWS CloudHSM atau Hashicorp Vault.
> 3. **Multi-region active-active** — sekarang single region dengan geo-partitioning. Production EU customer data harus physically di EU, bukan cuma logically partitioned."

---

## Build in Public — X/Twitter Strategy

### Weekly Thread Ideas

|Week|Thread Topic|Hook|
|---|---|---|
|1|"Building a cross-border payment system from scratch"|Day 1: PostgreSQL schema design for fintech. Here's why I'm using temporal constraints...|
|2|"Why TigerBeetle might replace your PostgreSQL for ledgers"|Most payment systems use PG for ledgers. Here's why I switched to TigerBeetle...|
|3|"The tier system design that most SaaS gets wrong"|Your tier shouldn't be in the users table. Here's why...|
|4|"How OFAC sanctions screening actually works"|Fuzzy matching "Al-Qaeda" vs "Al Qaeda" — harder than it sounds. Thread 🧵|
|5|"Webhook delivery is harder than you think"|10 retries, exponential backoff, JWS signatures. Here's my approach...|
|6|"Saved 53% on FX costs with one algorithm"|Corridor netting explained in 5 tweets.|
|7|"FAPI 2.0 vs regular OAuth — what banks actually require"|PSD2 compliance isn't just OAuth. DPoP, PAR, private_key_jwt...|
|8|"Load testing to 5K TPS — lessons learned"|k6 + TigerBeetle + PostgreSQL. Where the bottlenecks actually were.|

### Single Tweet Ideas

```
🔨 Day 3: First atomic transfer working. 
€100 debited, IDR credited, fee collected — all in one commit.
TigerBeetle's linked transfers are magic.
#buildinpublic #fintech

---

💡 TIL: PostgreSQL 18's EXCLUDE constraint + tstzrange 
= no overlapping pricing periods.
One line of DDL saved me from writing a complex validation layer.
#postgres #fintech

---

🤯 Just realized most payment startups handle FX wrong.
They convert GROSS volume instead of NET position.
With netting: €70k volume → €30k actual FX.
57% savings.
#buildinpublic

---

📊 Week 4 progress:
- IBAN validation: ✅ (MOD-97 + country specs)
- OFAC screening: ✅ (pg_trgm fuzzy match)
- Risk scoring: ✅ (0-100, auto-approve < 30)

Next: payment rail routing.
#fintech #buildinpublic

---

🏗️ Architecture decision I'm proud of:
Tenant identity ≠ Tenant policies.

Identity: who are you
Policy: what can you do, when, how much

Separate tables = audit trail + temporal overrides + no touching identity.
#systemdesign

---

⚡ First 5K TPS benchmark done.
P50: 12ms
P95: 35ms
P99: 48ms

Bottleneck was NOT TigerBeetle (surprise).
It was PostgreSQL connection pooling.
pgbouncer in transaction mode = solved.
#performance #golang
```

### Content Pillar Strategy

|Pillar|% of Content|Example|
|---|---|---|
|Technical deep-dives|40%|TigerBeetle linked transfers, FAPI 2.0 flow|
|Progress updates|30%|Weekly demo clips, metrics achieved|
|Lessons learned|20%|"What I'd do differently", gotchas|
|Industry context|10%|Why cross-border payments are broken, market size|

### Engagement Tactics

1. **Demo GIFs** — 10-15 sec clips of working features
2. **Code snippets** — Interesting patterns, not boilerplate
3. **Before/after** — "Before netting: 5 FX conversions. After: 1."
4. **Ask questions** — "How do you handle webhook delivery failures?"
5. **Tag relevant people** — TigerBeetle team, fintech founders, Go community

### Hashtags

Primary: `#buildinpublic` `#fintech` `#golang` Secondary: `#payments` `#systemdesign` `#startup` Occasional: `#postgresql` `#tigerbeetle` `#crossborder`

---

## Demo Videos Summary

|Week|Duration|Focus|Key Proof|
|---|---|---|---|
|1|2 min|PostgreSQL + Basic Ledger|RLS + Temporal constraints|
|2|3 min|Multi-Currency + Atomic FX|5-step rollback|
|3|4 min|API + FX + Tier Policies|Hierarchy + Netting + Rate limit|
|4|4 min|Compliance + Rails + Validation|IBAN + Risk score + Routing|
|5|4 min|Webhooks + Real-time|Retry + WebSocket|
|6|4 min|Reconciliation + Netting|53% savings + E2E test|
|7|4 min|Regional Security|FAPI 2.0 + SNAP|
|8|5 min|Dashboard + Load Test|5K TPS, <50ms P99|

**Total Demo Reel**: ~29 minutes (cut to 12-15 min highlight)

---

## Final Deliverables

### Codebase Structure

```
kovra/
├── cmd/
│   ├── api/              # Main API server
│   └── worker/           # Background workers (River)
├── internal/
│   ├── tenant/           # Multi-tenant management
│   ├── wallet/           # TigerBeetle wallet ops
│   ├── fx/               # FX engine + netting
│   ├── compliance/       # OFAC, sanctions, risk scoring
│   ├── validation/       # IBAN/BIC validation, bank directory
│   ├── rails/            # Mock adapters: SEPA, BI-FAST, UK FPS
│   ├── webhook/          # Delivery engine
│   ├── tracking/         # WebSocket + timeline
│   └── reconciliation/   # Daily recon
├── pkg/
│   ├── fapi/             # FAPI 2.0 client
│   ├── snap/             # SNAP Indonesia
│   └── tigerbeetle/      # TB client wrapper
├── web/
│   └── admin/            # React dashboard
├── scripts/
│   ├── k6/               # Load tests
│   └── demo.sh           # Demo script
├── e2e/                  # End-to-end tests
├── docs/
│   └── adr/              # Architectural Decision Records
├── migrations/           # PostgreSQL migrations
└── docker-compose.yml    # Full stack
```

### Technical Metrics

|Metric|Target|Status|
|---|---|---|
|TPS|5,000|⏳|
|P50 Latency|< 15ms|⏳|
|P95 Latency|< 50ms|⏳|
|P99 Latency|< 100ms|⏳|
|Success Rate|> 99.5%|⏳|
|Webhook Delivery|> 99%|⏳|
|FX Rate Freshness|< 30s|⏳|

### Security Checklist

- [ ] mTLS for external connections
- [ ] JWS webhook signatures (RS256)
- [ ] Rate limiting per tenant tier
- [ ] Parameterized queries (sqlc)
- [ ] HTTPS only (TLS 1.3)
- [ ] API keys hashed (bcrypt)
- [ ] Secrets in environment variables
- [ ] Audit logging (all mutations)
- [ ] Idempotency enforcement
- [ ] Request timeout (10s max)

### Compliance Features

- [ ] OFAC/EU/UK sanctions screening
- [ ] PEP check (mock)
- [ ] Velocity monitoring
- [ ] DHE tracking structure (Indonesia)
- [ ] CESOP reporting structure (EU)
- [ ] Daily FBO/Nostro reconciliation
- [ ] Immutable audit trail (geo-partitioned)

---

## Resume Statement

```
Built B2B cross-border payment rails platform (Kovra) processing 
5K TPS with <50ms P99 latency and 99.8% success rate.

Architecture:
• FBO/Nostro account model with TigerBeetle double-entry ledger
• PostgreSQL 18 geo-partitioning with RLS (OJK/GDPR/FCA compliance)
• Multi-tenant virtual wallets with atomic 5-step FX conversion
• Corridor netting engine (53% FX cost reduction)
• Smart routing across SEPA Instant, BI-FAST, SWIFT

Security:
• FAPI 2.0 (PSD2 compliant) + SNAP (Bank Indonesia)
• mTLS, JWS signing, comprehensive audit trail

Features:
• Real-time tracking (WebSocket), webhook delivery with exponential retry
• Automated compliance (OFAC, sanctions screening, risk scoring 0-100)
• Daily reconciliation, DHE/CESOP reporting structures

Tech Stack:
Go 1.23, TigerBeetle, PostgreSQL 18, Redis 8, Kafka, React 19

Portfolio: github.com/username/kovra
Demo: 15-minute video walkthrough
```

---

## Quick Start

```bash
# Clone
git clone https://github.com/username/kovra.git && cd kovra

# Start infrastructure
docker-compose up -d

# Run migrations
make migrate

# Start API + Workers
make run

# Run demo
./scripts/demo.sh

# Open dashboard
open http://localhost:3000
```