## Overview

Kovra adalah **B2B Cross-border Payment Rails** untuk Indonesian exporters dan e-commerce platforms yang menerima pembayaran dari EU/UK markets.

```
┌─────────────────────────────────────────────────────────────────┐
│                      KOVRA ARCHITECTURE                         │
│                                                                 │
│   Bank Level              Internal Level         Rail Level     │
│   (Regulatory)            (TigerBeetle)          (Settlement)   │
│                                                                 │
│   ┌───────────┐          ┌──────────────┐       ┌───────────┐   │
│   │ FBO       │ ◄──────► │ TENANT_      │       │ SEPA      │   │
│   │ Accounts  │          │ WALLET       │       │ Instant   │   │
│   └───────────┘          └──────────────┘       └───────────┘   │
│                                │                      │         │
│   ┌───────────┐          ┌─────▼────────┐       ┌─────▼─────┐   │
│   │ Nostro    │ ◄──────► │ REGIONAL_    │ ◄───► │ BI-FAST   │   │
│   │ Accounts  │          │ SETTLEMENT   │       │ RTGS      │   │
│   └───────────┘          └──────────────┘       └───────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 1. FBO Accounts (For Benefit Of)

### Definition

FBO Account adalah **pooled bank account** yang menyimpan dana klien (tenants) secara segregated dari operational funds Kovra.

### Bank-Level Structure

```
┌─────────────────────────────────────────────────────────────────┐
│                    FBO ACCOUNTS (BANK LEVEL)                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  FBO EUR Account                                                │
│  Bank: Deutsche Bank (Germany)                                  │
│  Account Name: "Kovra Pte Ltd FBO Clients"                      │
│  IBAN: DE89 3704 0044 0532 0130 00                              │
│  Purpose: Hold EUR funds dari EU buyers sebelum conversion      │
│                                                                 │
│  FBO GBP Account                                                │
│  Bank: Barclays (UK)                                            │
│  Account Name: "Kovra Pte Ltd FBO Clients"                      │
│  Sort Code: 20-00-00 | Account: 12345678                        │
│  Purpose: Hold GBP funds dari UK buyers sebelum conversion      │
│                                                                 │
│  FBO IDR Account                                                │
│  Bank: Bank Mandiri (Indonesia)                                 │
│  Account Name: "PT Kovra Indonesia FBO Clients"                 │
│  Account No: 123-00-1234567-8                                   │
│  Purpose: Hold IDR funds untuk payout ke beneficiaries          │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### TigerBeetle Mapping: TENANT_WALLET

FBO bank account adalah **single pooled account**. Breakdown per tenant di-track di TigerBeetle sebagai `TENANT_WALLET`:

```
┌─────────────────────────────────────────────────────────────────┐
│              TIGERBEETLE TENANT_WALLET (0x01)                   │
│              Internal sub-ledger of FBO Account                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  FBO EUR (Deutsche Bank) Total: €5,000,000                      │
│  ├── TENANT_WALLET [tokopedia_seller] EUR: €2,500,000           │
│  ├── TENANT_WALLET [bukalapak_intl]   EUR: €1,500,000           │
│  └── TENANT_WALLET [corp_treasury_a]  EUR: €1,000,000           │
│                                                                 │
│  FBO GBP (Barclays) Total: £3,000,000                           │
│  ├── TENANT_WALLET [tokopedia_seller] GBP: £1,800,000           │
│  ├── TENANT_WALLET [bukalapak_intl]   GBP: £700,000             │
│  └── TENANT_WALLET [corp_treasury_a]  GBP: £500,000             │
│                                                                 │
│  FBO IDR (Mandiri) Total: IDR 50,000,000,000                    │
│  ├── TENANT_WALLET [tokopedia_seller] IDR: 30,000,000,000       │
│  ├── TENANT_WALLET [bukalapak_intl]   IDR: 12,000,000,000       │
│  └── TENANT_WALLET [corp_treasury_a]  IDR: 8,000,000,000        │
│                                                                 │
│  Invariant: SUM(TENANT_WALLET per currency) == FBO Bank Balance │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Account ID Format

```
TENANT_WALLET Account ID (128-bit):
┌────────────────────────────────────────────────────────────────┐
│  [tenant_id: 64 bits] [type: 0x01] [currency: ISO 4217 code]   │
└────────────────────────────────────────────────────────────────┘

Examples:
- tokopedia_seller EUR wallet: 0x{tenant_hash}_01_978
- tokopedia_seller IDR wallet: 0x{tenant_hash}_01_360
- bukalapak_intl GBP wallet:   0x{tenant_hash}_01_826
```

### FBO Regulatory Requirements

| Jurisdiction  | Regulator       | Key Requirements                                     | Kovra Implementation                          |
| ------------- | --------------- | ---------------------------------------------------- | --------------------------------------------- |
| **EU**        | EBA (PSD2)      | Safeguarding wajib, segregasi dari operational funds | FBO di Deutsche Bank, daily reconciliation    |
| **UK**        | FCA (PSRs 2017) | Safeguarding, resolution pack, daily reconciliation  | FBO di Barclays, CASS 15 compliance           |
| **Indonesia** | Bank Indonesia  | 30% di BUKU 4, 70% di BI/SBN, escrow requirements    | FBO di Mandiri (BUKU 4), placement compliance |

### Indonesia-Specific: Floating Fund Placement

```
┌─────────────────────────────────────────────────────────────────┐
│           INDONESIA FBO PLACEMENT REQUIREMENTS                  │
│                  (BI Regulation PBI 20/6/2018)                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  FBO IDR Total: IDR 50,000,000,000                              │
│                                                                 │
│  Required Placement:                                            │
│  ├── 30% minimum di BUKU 4 Bank                                 │
│  │   └── IDR 15,000,000,000 @ Bank Mandiri (checking)           │
│  │                                                              │
│  └── 70% maximum di BI atau Government Securities               │
│      ├── IDR 20,000,000,000 @ Bank Indonesia (FASBI)            │
│      └── IDR 15,000,000,000 @ SBN (Government Bonds)            │
│                                                                 │
│  Compliance Check: Daily automated verification                 │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. Nostro Accounts (Settlement Accounts)

### Definition

Nostro Account adalah **Kovra's own pre-funded settlement accounts** di berbagai correspondent banks untuk instant cross-border settlement.

### Bank-Level Structure

```
┌─────────────────────────────────────────────────────────────────┐
│                 NOSTRO ACCOUNTS (BANK LEVEL)                    │
│                   Kovra Operational Funds                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Nostro EUR Account                                             │
│  Bank: Deutsche Bank (Germany)                                  │
│  Account Name: "Kovra Pte Ltd - Settlement"                     │
│  IBAN: DE89 3704 0044 0532 0131 00                              │
│  Balance: €10,000,000 (pre-funded)                              │
│  Purpose: SEPA Instant inbound/outbound settlement              │
│  Rails: SEPA Instant (< €100K), SEPA Credit Transfer            │
│                                                                 │
│  Nostro GBP Account                                             │
│  Bank: Barclays (UK)                                            │
│  Account Name: "Kovra Pte Ltd - Settlement"                     │
│  Sort Code: 20-00-00 | Account: 87654321                        │
│  Balance: £5,000,000 (pre-funded)                               │
│  Purpose: Faster Payments / CHAPS settlement                    │
│  Rails: FPS (< £1M), CHAPS (> £1M)                              │
│                                                                 │
│  Nostro IDR Account                                             │
│  Bank: Bank Mandiri (Indonesia)                                 │
│  Account Name: "PT Kovra Indonesia - Settlement"                │
│  Account No: 123-00-7654321-0                                   │
│  Balance: IDR 100,000,000,000 (pre-funded)                      │
│  Purpose: BI-FAST / RTGS payout to beneficiaries                │
│  Rails: BI-FAST (< IDR 250M), RTGS (> IDR 250M)                 │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### TigerBeetle Mapping: REGIONAL_SETTLEMENT

Nostro bank balances di-track di TigerBeetle sebagai `REGIONAL_SETTLEMENT`:

```
┌─────────────────────────────────────────────────────────────────┐
│           TIGERBEETLE REGIONAL_SETTLEMENT (0x06)                │
│              Tracks Nostro Account Balances                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  REGIONAL_SETTLEMENT_EU (Ledger 978 - EUR)                      │
│  └── Balance: €10,000,000                                       │
│      Linked to: Nostro EUR @ Deutsche Bank                      │
│                                                                 │
│  REGIONAL_SETTLEMENT_UK (Ledger 826 - GBP)                      │
│  └── Balance: £5,000,000                                        │
│      Linked to: Nostro GBP @ Barclays                           │
│                                                                 │
│  REGIONAL_SETTLEMENT_ID (Ledger 360 - IDR)                      │
│  └── Balance: IDR 100,000,000,000                               │
│      Linked to: Nostro IDR @ Bank Mandiri                       │
│                                                                 │
│  Invariant: REGIONAL_SETTLEMENT == Nostro Bank Balance          │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Nostro Regulatory Requirements

|Jurisdiction|Requirement|Kovra Implementation|
|---|---|---|
|**Global**|Correspondent banking due diligence|Annual KYC review on all correspondent banks|
|**US (OFAC)**|Sanctions screening on all transactions|Compliance Service dengan real-time OFAC check|
|**EU (AMLD)**|AML transaction monitoring|Continuous monitoring + suspicious activity reporting|
|**Indonesia**|BI PJP License untuk operate settlement|PT Kovra Indonesia dengan PJP Category 1 license|

---

## 3. Complete Account Structure

### TigerBeetle Account Types

```
┌─────────────────────────────────────────────────────────────────┐
│              TIGERBEETLE ACCOUNT TYPES                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  0x01 - TENANT_WALLET                                           │
│         Maps to: FBO sub-ledger (client funds)                  │
│         Owner: Tenant (segregated)                              │
│         Purpose: Hold client funds per currency                 │
│                                                                 │
│  0x02 - FEE_REVENUE                                             │
│         Maps to: Kovra operational account                      │
│         Owner: Kovra                                            │
│         Purpose: Collected fees (FX margin, transfer fees)      │
│                                                                 │
│  0x03 - FX_SETTLEMENT                                           │
│         Maps to: Internal clearing account                      │
│         Owner: Kovra (system)                                   │
│         Purpose: Atomic FX conversion bridge                    │
│                                                                 │
│  0x04 - PENDING_INBOUND                                         │
│         Maps to: Transit account                                │
│         Owner: Kovra (system)                                   │
│         Purpose: Funds being collected (not yet confirmed)      │
│                                                                 │
│  0x05 - PENDING_OUTBOUND                                        │
│         Maps to: Transit account                                │
│         Owner: Kovra (system)                                   │
│         Purpose: Funds being sent (not yet delivered)           │
│                                                                 │
│  0x06 - REGIONAL_SETTLEMENT                                     │
│         Maps to: Nostro accounts (settlement funds)             │
│         Owner: Kovra                                            │
│         Purpose: Pre-funded liquidity for instant settlement    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Bank ↔ TigerBeetle Mapping Summary

```
┌─────────────────────────────────────────────────────────────────┐
│              BANK ↔ TIGERBEETLE MAPPING                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  BANK LEVEL                    TIGERBEETLE LEVEL                │
│  ──────────                    ─────────────────                │
│                                                                 │
│  FBO EUR @ Deutsche Bank  ───► SUM(TENANT_WALLET EUR)           │
│  FBO GBP @ Barclays       ───► SUM(TENANT_WALLET GBP)           │
│  FBO IDR @ Mandiri        ───► SUM(TENANT_WALLET IDR)           │
│                                                                 │
│  Nostro EUR @ Deutsche    ───► REGIONAL_SETTLEMENT_EU           │
│  Nostro GBP @ Barclays    ───► REGIONAL_SETTLEMENT_UK           │
│  Nostro IDR @ Mandiri     ───► REGIONAL_SETTLEMENT_ID           │
│                                                                 │
│  Kovra OpEx Account       ───► FEE_REVENUE (per currency)       │
│                                                                 │
│  (No bank account)        ───► FX_SETTLEMENT (internal only)    │
│  (No bank account)        ───► PENDING_INBOUND (transit)        │
│  (No bank account)        ───► PENDING_OUTBOUND (transit)       │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 4. Transaction Flow: EU → Indonesia (Export Receipt)

### Scenario

Indonesian exporter (Tokopedia seller) menerima €10,000 dari EU buyer untuk export goods.

### Flow Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│         EU → INDONESIA EXPORT RECEIPT FLOW                      │
│         €10,000 EUR → IDR (Rate: 17,250)                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  EU Buyer's Bank                                                │
│       │                                                         │
│       │ SEPA Instant (€10,000)                                  │
│       ▼                                                         │
│  ┌─────────────────┐                                            │
│  │ Nostro EUR      │  Kovra's settlement account                │
│  │ @ Deutsche Bank │  receives €10,000                          │
│  └────────┬────────┘                                            │
│           │                                                     │
│           │ TigerBeetle: Credit REGIONAL_SETTLEMENT_EU          │
│           ▼                                                     │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              KOVRA PROCESSING                           │    │
│  │                                                         │    │
│  │  1. Compliance Check (OFAC, EU Sanctions, PEP)          │    │
│  │  2. Fee Calculation: 0.8% = €80                         │    │
│  │  3. FX Conversion: €9,920 × 17,250 = IDR 171,120,000    │    │
│  │  4. TigerBeetle atomic linked transfers                 │    │
│  │                                                         │    │
│  └─────────────────────────────────────────────────────────┘    │
│           │                                                     │
│           │ TigerBeetle: Debit REGIONAL_SETTLEMENT_ID           │
│           ▼                                                     │
│  ┌─────────────────┐                                            │
│  │ Nostro IDR      │  Kovra's settlement account                │
│  │ @ Bank Mandiri  │  sends IDR 171,120,000                     │
│  └────────┬────────┘                                            │
│           │                                                     │
│           │ BI-FAST (IDR 171,120,000)                           │
│           ▼                                                     │
│  Indonesian Exporter's Bank Account                             │
│  (Beneficiary receives IDR 171,120,000)                         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### TigerBeetle Ledger Entries

```
┌─────────────────────────────────────────────────────────────────┐
│              TIGERBEETLE ATOMIC LINKED TRANSFERS                │
│                    (All-or-nothing execution)                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Transfer 1 (LINKED): Receive EUR from SEPA                     │
│  ├── Debit:  REGIONAL_SETTLEMENT_EU        €10,000              │
│  └── Credit: PENDING_INBOUND_EUR           €10,000              │
│                                                                 │
│  Transfer 2 (LINKED): Collect platform fee                      │
│  ├── Debit:  PENDING_INBOUND_EUR           €80                  │
│  └── Credit: FEE_REVENUE_EUR               €80                  │
│                                                                 │
│  Transfer 3 (LINKED): Move to FX settlement (EUR side)          │
│  ├── Debit:  PENDING_INBOUND_EUR           €9,920               │
│  └── Credit: FX_SETTLEMENT_EUR             €9,920               │
│                                                                 │
│  Transfer 4 (LINKED): FX conversion (IDR side)                  │
│  ├── Debit:  FX_SETTLEMENT_IDR             IDR 171,120,000      │
│  └── Credit: PENDING_OUTBOUND_IDR          IDR 171,120,000      │
│                                                                 │
│  Transfer 5 (FINAL): Payout via BI-FAST                         │
│  ├── Debit:  PENDING_OUTBOUND_IDR          IDR 171,120,000      │
│  └── Credit: REGIONAL_SETTLEMENT_ID        IDR 171,120,000      │
│                                                                 │
│  Result:                                                        │
│  • REGIONAL_SETTLEMENT_EU: -€10,000 (sent to beneficiary)       │
│  • REGIONAL_SETTLEMENT_ID: +IDR 171,120,000 (needs rebalance)   │
│  • FEE_REVENUE_EUR: +€80 (Kovra revenue)                        │
│                                                                 │
│  If ANY transfer fails → ALL automatically rollback             │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 5. Reconciliation Process

### Daily Reconciliation (River Scheduled Job)

```
┌─────────────────────────────────────────────────────────────────┐
│              DAILY RECONCILIATION PROCESS                       │
│              Scheduled: 00:00 UTC via River                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. FBO RECONCILIATION                                          │
│     ┌───────────────────────────────────────────────────────┐   │
│     │ For each currency (EUR, GBP, IDR):                    │   │
│     │                                                       │   │
│     │ TigerBeetle:                                          │   │
│     │   SELECT SUM(balance)                                 │   │
│     │   FROM accounts                                       │   │
│     │   WHERE type = TENANT_WALLET AND currency = X         │   │
│     │                                                       │   │
│     │ Bank Statement:                                       │   │
│     │   Fetch via API (Deutsche Bank, Barclays, Mandiri)    │   │
│     │                                                       │   │
│     │ Validation:                                           │   │
│     │   IF TigerBeetle_sum != Bank_balance THEN             │   │
│     │     → Alert: FBO_MISMATCH                             │   │
│     │     → Block new transactions                          │   │
│     │     → Trigger investigation                           │   │
│     └───────────────────────────────────────────────────────┘   │
│                                                                 │
│  2. NOSTRO RECONCILIATION                                       │
│     ┌───────────────────────────────────────────────────────┐   │
│     │ For each REGIONAL_SETTLEMENT account:                 │   │
│     │                                                       │   │
│     │ TigerBeetle:                                          │   │
│     │   SELECT balance                                      │   │
│     │   FROM accounts                                       │   │
│     │   WHERE type = REGIONAL_SETTLEMENT AND currency = X   │   │
│     │                                                       │   │
│     │ Bank Statement:                                       │   │
│     │   Fetch Nostro balance via API                        │   │
│     │                                                       │   │
│     │ Validation:                                           │   │
│     │   IF TigerBeetle != Bank_balance THEN                 │   │
│     │     → Alert: NOSTRO_MISMATCH                          │   │
│     │     → Investigate pending transactions                │   │
│     └───────────────────────────────────────────────────────┘   │
│                                                                 │
│  3. TENANT RECONCILIATION                                       │
│     ┌───────────────────────────────────────────────────────┐   │
│     │ For each tenant:                                      │   │
│     │                                                       │   │
│     │ TigerBeetle:                                          │   │
│     │   Tenant balance per currency                         │   │
│     │                                                       │   │
│     │ Tenant's Records:                                     │   │
│     │   Via reconciliation API (optional)                   │   │
│     │                                                       │   │
│     │ Generate:                                             │   │
│     │   Daily statement per tenant (webhook or download)    │   │
│     └───────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Reconciliation Metrics

```yaml
# Prometheus metrics for reconciliation
reconciliation_runs_total{type="fbo|nostro|tenant", status="success|mismatch"}
reconciliation_mismatch_amount{currency, type}
reconciliation_duration_seconds{type}
reconciliation_last_success_timestamp{type}
```

---

## 6. Regulatory Compliance Matrix

### License Requirements

```
┌─────────────────────────────────────────────────────────────────┐
│              KOVRA LICENSE REQUIREMENTS                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  EUROPEAN UNION                                                 │
│  ├── License: EMI (Electronic Money Institution)                │
│  ├── Regulator: National competent authority (e.g., BaFin)      │
│  ├── Capital: Minimum €350,000                                  │
│  ├── Passporting: Available across EU/EEA                       │
│  └── Requirements:                                              │
│      • PSD2 safeguarding compliance                             │
│      • Daily reconciliation                                     │
│      • AMLD compliance (KYC/AML)                                │
│                                                                 │
│  UNITED KINGDOM (Post-Brexit)                                   │
│  ├── License: EMI or PI (Payment Institution)                   │
│  ├── Regulator: FCA (Financial Conduct Authority)               │
│  ├── Capital: Based on payment volume                           │
│  ├── Passporting: NOT available to EU (separate license)        │
│  └── Requirements:                                              │
│      • PSRs 2017 safeguarding                                   │
│      • FCA CP24/20 new requirements (2025)                      │
│      • Resolution pack maintenance                              │
│      • Annual safeguarding audit                                │
│                                                                 │
│  INDONESIA                                                      │
│  ├── License: PJP (Penyelenggara Jasa Pembayaran) from BI       │
│  ├── Regulator: Bank Indonesia                                  │
│  ├── Capital: Minimum Rp 3 billion (scales with volume)         │
│  ├── Foreign ownership: Maximum 49%                             │
│  └── Requirements:                                              │
│      • 30% floating funds di BUKU 4 bank                        │
│      • 70% di BI atau SBN                                       │
│      • SNAP compliance untuk BI-FAST                            │
│      • OJK reporting untuk AML                                  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Compliance Checklist per Account Type

|Account Type|EU Compliance|UK Compliance|Indonesia Compliance|
|---|---|---|---|
|**FBO (TENANT_WALLET)**|PSD2 safeguarding, segregation|PSRs 2017, CASS 15, resolution pack|30/70 placement rule|
|**Nostro (REGIONAL_SETTLEMENT)**|AMLD transaction monitoring|UK AML Regs|BI PJP license|
|**Fee Revenue**|Standard corporate accounting|Standard corporate accounting|OJK reporting|

---

## 7. Liquidity Management

### Nostro Rebalancing

```
┌─────────────────────────────────────────────────────────────────┐
│              NOSTRO LIQUIDITY MANAGEMENT                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  SCENARIO: High EU → ID volume depletes EUR, accumulates IDR    │
│                                                                 │
│  Before rebalance:                                              │
│  ├── Nostro EUR: €2,000,000 (LOW - threshold €3M)               │
│  └── Nostro IDR: IDR 180,000,000,000 (HIGH)                     │
│                                                                 │
│  Rebalance Action:                                              │
│  ├── Treasury team initiates FX trade                           │
│  ├── Sell IDR 85,000,000,000                                    │
│  ├── Buy €5,000,000                                             │
│  └── Execute via correspondent bank                             │
│                                                                 │
│  After rebalance:                                               │
│  ├── Nostro EUR: €7,000,000 ✓                                   │
│  └── Nostro IDR: IDR 95,000,000,000 ✓                           │
│                                                                 │
│  Automation:                                                    │
│  ├── Alert when Nostro < threshold                              │
│  ├── Daily liquidity report                                     │
│  └── Projected runway based on volume trends                    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Liquidity Thresholds

|Currency|Minimum Balance|Target Balance|Alert Threshold|
|---|---|---|---|
|EUR|€3,000,000|€10,000,000|€5,000,000|
|GBP|£2,000,000|£5,000,000|£3,000,000|
|IDR|IDR 30B|IDR 100B|IDR 50B|

---

## 8. Corridor Netting (Wise Logic)

### Why Netting Matters

Tenants dengan **bidirectional flows** dapat menghemat FX cost signifikan melalui internal netting.

```
┌─────────────────────────────────────────────────────────────────┐
│              CORRIDOR NETTING OPTIMIZATION                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  SCENARIO: Tokopedia Seller (Monthly)                           │
│  ├── INBOUND:  €50,000 from Amazon EU buyers                    │
│  └── OUTBOUND: €20,000 to European suppliers                    │
│                                                                 │
│  ❌ WITHOUT NETTING:                                            │
│  ├── Convert €50,000 → IDR (0.8% fee = €400)                    │
│  ├── Convert €20,000 ← IDR (0.8% fee = €160)                    │
│  └── Total FX fees: €560                                        │
│                                                                 │
│  ✅ WITH NETTING:                                               │
│  ├── Net position: €50,000 - €20,000 = €30,000 inbound          │
│  ├── Internal book transfer: €20,000 (ZERO FX cost)             │
│  ├── Only convert €30,000 → IDR (0.8% fee = €240)               │
│  └── Total FX fees: €240                                        │
│                                                                 │
│  SAVINGS: €320/month (57% reduction)                            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### TigerBeetle Netting Flow

```
┌─────────────────────────────────────────────────────────────────┐
│              NETTING vs STANDARD FLOW                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  STANDARD FLOW (No netting opportunity):                        │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ REGIONAL_SETTLEMENT_EU                                  │    │
│  │        ↓                                                │    │
│  │ PENDING_INBOUND_EUR                                     │    │
│  │        ↓                                                │    │
│  │ FX_SETTLEMENT_EUR ←→ FX_SETTLEMENT_IDR (FX conversion)  │    │
│  │                              ↓                          │    │
│  │                      PENDING_OUTBOUND_IDR               │    │
│  │                              ↓                          │    │
│  │                      REGIONAL_SETTLEMENT_ID             │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  NETTING FLOW (Opposite flows detected):                        │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ Inbound €50k:                                           │    │
│  │   REGIONAL_SETTLEMENT_EU → PENDING_INBOUND_EUR          │    │
│  │                                                         │    │
│  │ Outbound €20k (NETTED - no FX):                         │    │
│  │   PENDING_INBOUND_EUR → PENDING_OUTBOUND_EUR            │    │
│  │   (Internal book transfer, skip FX_SETTLEMENT)          │    │
│  │                                                         │    │
│  │ Net €30k only (FX conversion):                          │    │
│  │   PENDING_INBOUND_EUR → FX_SETTLEMENT → IDR payout      │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Netting Service Implementation

```go
// corridor_netting_service.go
type NettingWindow struct {
    Duration    time.Duration  // 5 minutes default
    MinAmount   int64          // Minimum €1,000 to trigger netting
}

func (s *NettingService) FindNettingOpportunity(transfer *Transfer) *NettingMatch {
    // Find opposite direction transfers in window
    opposites := s.db.Query(`
        SELECT * FROM pending_transfers
        WHERE tenant_id = $1
          AND from_currency = $2      -- Opposite: their from = our to
          AND to_currency = $3        -- Opposite: their to = our from
          AND status = 'pending_netting'
          AND created_at >= NOW() - INTERVAL '5 minutes'
        ORDER BY amount DESC
    `, transfer.TenantID, transfer.ToCurrency, transfer.FromCurrency)
    
    if len(opposites) > 0 {
        return &NettingMatch{
            MatchedTransfer: opposites[0],
            NetAmount:       abs(transfer.Amount - opposites[0].Amount),
            SavedFXCost:     min(transfer.Amount, opposites[0].Amount) * fxMargin,
        }
    }
    return nil
}
```

---

## 9. Rail Selection Logic (Wise Logic)

### Optimal Rail Routing

```
┌─────────────────────────────────────────────────────────────────┐
│              RAIL SELECTION DECISION TREE                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  INBOUND (Collection):                                          │
│                                                                 │
│  Source: EU/EEA Country?                                        │
│  ├── YES: Amount < €100,000?                                    │
│  │   ├── YES → SEPA Instant (€0.20, <10 sec)                    │
│  │   └── NO  → SEPA Credit Transfer (FREE, T+1)                 │
│  │                                                              │
│  └── NO: Source = UK?                                           │
│      ├── YES: Amount < £1,000,000?                              │
│      │   ├── YES → Faster Payments (FREE, <2 hours)             │
│      │   └── NO  → CHAPS (£25, same day)                        │
│      │                                                          │
│      └── NO → SWIFT (€25-45, 2-3 days)                          │
│                                                                 │
│  OUTBOUND (Payout to Indonesia):                                │
│                                                                 │
│  Amount < IDR 250,000,000?                                      │
│  ├── YES → BI-FAST (IDR 2,500, <30 sec)                         │
│  └── NO  → RTGS (IDR 25,000, same day)                          │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Cost Comparison Matrix

|Route|Rail|Cost|Speed|When to Use|
|---|---|---|---|---|
|**EU → Kovra**|SEPA Instant|€0.20|<10 sec|Default for <€100k|
|**EU → Kovra**|SEPA SCT|FREE|T+1|Batch/large amounts|
|**UK → Kovra**|Faster Payments|FREE|<2 hours|Default for <£1M|
|**UK → Kovra**|CHAPS|£25|Same day|>£1M urgent|
|**Non-EU/UK → Kovra**|SWIFT|€25-45|2-3 days|Fallback only|
|**Kovra → Indonesia**|BI-FAST|IDR 2,500|<30 sec|Default for <IDR 250M|
|**Kovra → Indonesia**|RTGS|IDR 25,000|Same day|>IDR 250M|

### Batching Optimization

```
┌─────────────────────────────────────────────────────────────────┐
│              BATCH SETTLEMENT OPTIMIZATION                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  SCENARIO: 10 transfers to Indonesia dalam 5 menit              │
│  ├── Transfer 1: IDR 15,000,000                                 │
│  ├── Transfer 2: IDR 8,000,000                                  │
│  ├── ...                                                        │
│  └── Transfer 10: IDR 12,000,000                                │
│  Total: IDR 95,000,000                                          │
│                                                                 │
│  ❌ INDIVIDUAL SETTLEMENT:                                      │
│  10 × BI-FAST = 10 × IDR 2,500 = IDR 25,000                     │
│                                                                 │
│  ✅ BATCHED SETTLEMENT:                                         │
│  1 × BI-FAST (IDR 95M) = IDR 2,500                              │
│  Internal split via TigerBeetle (FREE)                          │
│                                                                 │
│  SAVINGS: IDR 22,500 (90% reduction)                            │
│                                                                 │
│  Implementation:                                                │
│  ├── Collect transfers in 5-min window                          │
│  ├── Single BI-FAST to Kovra Nostro IDR                         │
│  ├── TigerBeetle atomic split to beneficiaries                  │
│  └── Individual BI-FAST payouts from Nostro                     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 10. Legal Entity Structure

### Multi-Entity Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│              KOVRA LEGAL ENTITY STRUCTURE                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│                    Kovra Holdings Pte Ltd                       │
│                        (Singapore)                              │
│                            │                                    │
│            ┌───────────────┼───────────────┐                    │
│            │               │               │                    │
│            ▼               ▼               ▼                    │
│  ┌─────────────────┐ ┌─────────────┐ ┌─────────────────┐        │
│  │ Kovra Europe SA │ │ Kovra UK Ltd│ │ PT Kovra        │        │
│  │    (Belgium)    │ │  (London)   │ │ Indonesia       │        │
│  ├─────────────────┤ ├─────────────┤ ├─────────────────┤        │
│  │                 │ │             │ │                 │        │
│  │ License:        │ │ License:    │ │ License:        │        │
│  │ EMI (NBB)       │ │ EMI (FCA)   │ │ PJP (BI)        │        │
│  │                 │ │             │ │                 │        │
│  │ Rails:          │ │ Rails:      │ │ Rails:          │        │
│  │ • SEPA Instant  │ │ • FPS       │ │ • BI-FAST       │        │
│  │ • SEPA SCT      │ │ • CHAPS     │ │ • RTGS          │        │
│  │ • TARGET2       │ │ • SWIFT     │ │ • SNAP API      │        │
│  │                 │ │             │ │                 │        │
│  │ Accounts:       │ │ Accounts:   │ │ Accounts:       │        │
│  │ • FBO EUR       │ │ • FBO GBP   │ │ • FBO IDR       │        │
│  │ • Nostro EUR    │ │ • Nostro GBP│ │ • Nostro IDR    │        │
│  │                 │ │             │ │                 │        │
│  │ Coverage:       │ │ Coverage:   │ │ Coverage:       │        │
│  │ 36 EEA countries│ │ UK only     │ │ Indonesia       │        │
│  │ (passporting)   │ │ (no EU)     │ │                 │        │
│  │                 │ │             │ │                 │        │
│  └─────────────────┘ └─────────────┘ └─────────────────┘        │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Entity ↔ Account Mapping

|Legal Entity|FBO Account|Nostro Account|TigerBeetle Mapping|
|---|---|---|---|
|**Kovra Europe SA**|FBO EUR @ Deutsche Bank|Nostro EUR @ Deutsche Bank|TENANT_WALLET (EUR), REGIONAL_SETTLEMENT_EU|
|**Kovra UK Ltd**|FBO GBP @ Barclays|Nostro GBP @ Barclays|TENANT_WALLET (GBP), REGIONAL_SETTLEMENT_UK|
|**PT Kovra Indonesia**|FBO IDR @ Mandiri|Nostro IDR @ Mandiri|TENANT_WALLET (IDR), REGIONAL_SETTLEMENT_ID|

### Inter-Entity Settlement

```
┌─────────────────────────────────────────────────────────────────┐
│              INTER-ENTITY FUND FLOW                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  SCENARIO: Swedish buyer → Indonesian exporter                  │
│                                                                 │
│  Step 1: Collection (Kovra Europe SA)                           │
│  ├── Swedish buyer's bank (Swedbank)                            │
│  ├── → SEPA Instant                                             │
│  └── → Nostro EUR @ Deutsche Bank (Kovra Europe SA)             │
│                                                                 │
│  Step 2: Internal Transfer (Book entry)                         │
│  ├── Kovra Europe SA Nostro EUR                                 │
│  ├── → Intercompany receivable/payable                          │
│  └── → PT Kovra Indonesia (internal accounting)                 │
│                                                                 │
│  Step 3: FX Conversion                                          │
│  ├── TigerBeetle: FX_SETTLEMENT_EUR → FX_SETTLEMENT_IDR         │
│  └── Rate locked from Quote Service                             │
│                                                                 │
│  Step 4: Payout (PT Kovra Indonesia)                            │
│  ├── Nostro IDR @ Bank Mandiri (PT Kovra Indonesia)             │
│  ├── → BI-FAST                                                  │
│  └── → Indonesian exporter's bank account                       │
│                                                                 │
│  Settlement: All within 42 seconds                              │
│  Cost: €0.20 (SEPA) + IDR 2,500 (BI-FAST) = ~€0.35 total        │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 11. Summary

### Key Principles

1. **Segregation**: FBO accounts (client funds) NEVER mix dengan Nostro accounts (operational funds)
    
2. **Atomicity**: All TigerBeetle transfers use linked flags untuk all-or-nothing execution
    
3. **Reconciliation**: Daily automated reconciliation antara TigerBeetle ledger dan bank statements
    
4. **Compliance**: Per-jurisdiction license dan regulatory requirements
    
5. **Liquidity**: Active Nostro management untuk ensure instant settlement capability
    
6. **Netting**: Internal corridor netting untuk minimize FX costs (up to 57% savings)
    
7. **Rail Optimization**: Always use cheapest local rail (SEPA/FPS/BI-FAST over SWIFT)
    
8. **Multi-Entity**: Separate legal entities per region untuk direct rail access
    

### Account Hierarchy

```
KOVRA ACCOUNT STRUCTURE
│
├── CLIENT FUNDS (FBO) ─────────────────────────────────────────┐
│   │                                                           │
│   ├── FBO EUR @ Deutsche Bank                                 │
│   │   └── TENANT_WALLET (per tenant) ← TigerBeetle            │
│   │                                                           │
│   ├── FBO GBP @ Barclays                                      │
│   │   └── TENANT_WALLET (per tenant) ← TigerBeetle            │
│   │                                                           │
│   └── FBO IDR @ Bank Mandiri                                  │
│       └── TENANT_WALLET (per tenant) ← TigerBeetle            │
│                                                               │
├── SETTLEMENT FUNDS (NOSTRO) ──────────────────────────────────┤
│   │                                                           │
│   ├── Nostro EUR @ Deutsche Bank                              │
│   │   └── REGIONAL_SETTLEMENT_EU ← TigerBeetle                │
│   │                                                           │
│   ├── Nostro GBP @ Barclays                                   │
│   │   └── REGIONAL_SETTLEMENT_UK ← TigerBeetle                │
│   │                                                           │
│   └── Nostro IDR @ Bank Mandiri                               │
│       └── REGIONAL_SETTLEMENT_ID ← TigerBeetle                │
│                                                               │
└── OPERATIONAL (KOVRA) ────────────────────────────────────────┤
    │                                                           │
    ├── FEE_REVENUE (per currency) ← TigerBeetle                │
    │                                                           │
    └── INTERNAL CLEARING                                       │
        ├── FX_SETTLEMENT ← TigerBeetle                         │
        ├── PENDING_INBOUND ← TigerBeetle                       │
        └── PENDING_OUTBOUND ← TigerBeetle                      │
```