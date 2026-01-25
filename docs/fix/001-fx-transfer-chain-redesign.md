# Fix #001: FX Transfer Chain Redesign

**Date:** 2025-01-25
**Severity:** P0 (Critical)
**Files Changed:** `internal/ledger/transfer.go`, `docs/DATABASE.md`

---

## TL;DR

TigerBeetle **tidak mengizinkan** transfer antar ledger berbeda. Desain lama `FXTransferChain()` mencoba transfer EUR→IDR dalam satu chain — ini akan **selalu gagal**. Solusinya: split jadi dua chain terpisah yang dikoordinasi di application level.

---

## 1. Masalah: Constraint TigerBeetle yang Diabaikan

### Apa itu Ledger di TigerBeetle?

TigerBeetle menggunakan konsep **ledger** sebagai isolasi untuk mata uang atau buku besar yang berbeda:

```
Ledger 978 = EUR (semua akun EUR hidup di sini)
Ledger 360 = IDR (semua akun IDR hidup di sini)
Ledger 840 = USD (semua akun USD hidup di sini)
```

### Constraint yang Dilanggar

TigerBeetle memiliki invariant keras:

> **"Both the debit and credit accounts must have the same ledger"**
> — TigerBeetle Documentation

Ini bukan bug, ini **by design**. Alasannya:

1. **Double-entry accounting** mengharuskan debit = credit dalam unit yang sama
2. Tidak masuk akal mendebit 100 EUR dan mengkredit 100 IDR — nilainya berbeda
3. FX rate adalah konsep bisnis, bukan konsep ledger

### Kode Lama yang Salah

```go
// ❌ INVALID: srcFXSettlement di ledger EUR, dstFXSettlement di ledger IDR
builder.Add(srcFXSettlement, dstFXSettlement, dstAmount, dstLedger, transferCode)
```

Error yang akan terjadi: `TransferAccountsMustHaveTheSameLedger`

---

## 2. Mengapa Desain Lama Terlihat Masuk Akal (Tapi Salah)

Desain lama mengasumsikan FX conversion bisa dilakukan dalam satu atomic operation:

```
┌─────────────────────────────────────────────────────────┐
│  DESAIN LAMA (SALAH)                                    │
│                                                         │
│  EUR Ledger        "Magic FX"         IDR Ledger        │
│  ───────────       ──────────         ───────────       │
│                                                         │
│  WALLET_EUR ─────────────────────────► FX_IDR ──► PAYOUT│
│       │                 ↑                               │
│       │                 │                               │
│       └── PENDING ──► FX_EUR ─── ??? ──┘                │
│                              Cross-ledger               │
│                              (IMPOSSIBLE!)              │
└─────────────────────────────────────────────────────────┘
```

**Kesalahan mental model:**

Kita berpikir TigerBeetle seperti database relasional yang bisa join antar tabel. Tidak. TigerBeetle adalah **ledger engine** yang menjaga integritas akuntansi.

---

## 3. Solusi: Dua Chain Terkoordinasi

### Prinsip Dasar

FX conversion sebenarnya adalah **dua operasi terpisah**:

1. **Source side:** User memberikan EUR ke platform
2. **Destination side:** Platform memberikan IDR ke recipient

Platform bertindak sebagai **counterparty** yang memegang posisi di kedua mata uang.

### Desain Baru

```
┌─────────────────────────────────────────────────────────────────┐
│  DESAIN BARU (CORRECT)                                          │
│                                                                 │
│  ╔═══════════════════════════════════════════════════════════╗  │
│  ║  SOURCE CHAIN (EUR Ledger = 978)                          ║  │
│  ║  Atomic within EUR ledger                                 ║  │
│  ╠═══════════════════════════════════════════════════════════╣  │
│  ║                                                           ║  │
│  ║  TENANT_WALLET_EUR ──debit──► PENDING_OUTBOUND_EUR        ║  │
│  ║                                      │                    ║  │
│  ║                                      ▼                    ║  │
│  ║                               FX_POSITION_EUR             ║  │
│  ║                               (platform receives EUR)     ║  │
│  ╚═══════════════════════════════════════════════════════════╝  │
│                           │                                     │
│                           │ CorrelationID (UUID v7)             │
│                           │ Links both chains                   │
│                           ▼                                     │
│  ╔═══════════════════════════════════════════════════════════╗  │
│  ║  DESTINATION CHAIN (IDR Ledger = 360)                     ║  │
│  ║  Atomic within IDR ledger                                 ║  │
│  ╠═══════════════════════════════════════════════════════════╣  │
│  ║                                                           ║  │
│  ║  FX_POSITION_IDR ──fee──► FEE_REVENUE_IDR                 ║  │
│  ║        │                                                  ║  │
│  ║        └──────payout────► REGIONAL_SETTLEMENT_IDR         ║  │
│  ║        (platform pays out IDR)                            ║  │
│  ╚═══════════════════════════════════════════════════════════╝  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Kode Baru

```go
type FXTransferPair struct {
    SourceChain      []Transfer  // Executes in EUR ledger
    DestinationChain []Transfer  // Executes in IDR ledger
    CorrelationID    [16]byte    // Links both for reconciliation
}

pair, err := FXTransferChains(tenantID, CurrencyEUR, CurrencyIDR,
    srcAmount, dstAmount, feeAmount, transferCode)
```

---

## 4. FX_POSITION Account: Platform Currency Exposure

### Apa itu FX_POSITION?

FX_POSITION (sebelumnya FX_SETTLEMENT) adalah akun yang merepresentasikan **posisi mata uang platform**.

```
┌────────────────────────────────────────────────────────────────┐
│  FX_POSITION SEMANTICS                                         │
├────────────────────────────────────────────────────────────────┤
│                                                                │
│  Credit FX_POSITION_EUR = Platform menerima EUR dari user      │
│                           Platform punya "long EUR position"   │
│                                                                │
│  Debit FX_POSITION_IDR  = Platform membayar IDR ke recipient   │
│                           Platform "menggunakan" IDR position  │
│                                                                │
└────────────────────────────────────────────────────────────────┘
```

### Analogi Dunia Nyata

Bayangkan Kovra seperti money changer:

1. **Customer datang dengan EUR 100**
   - Money changer terima EUR 100 → masuk laci EUR
   - Ini = Credit FX_POSITION_EUR

2. **Money changer kasih IDR 1,700,000**
   - Ambil dari laci IDR → keluar ke customer
   - Ini = Debit FX_POSITION_IDR

3. **Laci EUR dan IDR adalah dua entitas terpisah**
   - Tidak bisa "transfer" dari laci EUR ke laci IDR
   - Harus diproses sebagai dua transaksi terpisah

---

## 5. Application-Level Coordination

### Kenapa Perlu Koordinasi di Application Level?

TigerBeetle menjamin atomicity **dalam satu ledger**. Tapi untuk cross-ledger:

```
Source Chain  ──► TigerBeetle EUR ──► Success/Fail
                                          │
                        ┌─────────────────┘
                        ▼
Dest Chain    ──► TigerBeetle IDR ──► Success/Fail
```

Dua kemungkinan failure:
1. Source succeeds, Destination fails → **Perlu rollback source**
2. Source fails → **Stop, don't execute destination**

### Strategi Koordinasi

```go
// Pseudocode untuk FX execution
func ExecuteFXTransfer(pair FXTransferPair) error {
    // 1. Execute source chain
    err := ledger.CreateLinkedTransfers(pair.SourceChain)
    if err != nil {
        return fmt.Errorf("source chain failed: %w", err)
    }

    // 2. Execute destination chain
    err = ledger.CreateLinkedTransfers(pair.DestinationChain)
    if err != nil {
        // CRITICAL: Source sudah executed, perlu compensate
        compensateErr := rollbackSourceChain(pair)
        if compensateErr != nil {
            // Alert ops team, manual intervention needed
            alertOpsTeam(pair.CorrelationID, err, compensateErr)
        }
        return fmt.Errorf("dest chain failed, compensation attempted: %w", err)
    }

    // 3. Record FX details in PostgreSQL
    return recordFXInPostgres(pair.CorrelationID, fxRate, srcAmount, dstAmount)
}
```

### Kenapa Tidak Pakai Two-Phase Commit?

TigerBeetle mendukung **pending transfers** (two-phase), tapi:

1. Pending transfers untuk **single transfer**, bukan cross-ledger
2. Menambah complexity dan latency
3. Untuk most cases, compensating transaction cukup

---

## 6. Dimana FX Rate Disimpan?

### Bukan di TigerBeetle

TigerBeetle hanya menyimpan:
- Amount dalam minor units (cents/sen)
- Account IDs
- Transfer metadata (UserData fields)

TigerBeetle **tidak tahu** bahwa 100 EUR = 1,700,000 IDR.

### Di PostgreSQL

```sql
-- transfers table
INSERT INTO transfers (
    id,
    correlation_id,      -- Links to TigerBeetle UserData128
    from_currency,       -- 'EUR'
    to_currency,         -- 'IDR'
    from_amount,         -- 100.00
    to_amount,           -- 1700000.00
    fx_rate,             -- 17000.00
    fx_margin_bps,       -- 150 (1.5%)
    status
) VALUES (...);
```

### Kenapa Begini?

1. **Separation of concerns:** TigerBeetle = ledger integrity, PostgreSQL = business logic
2. **Queryability:** Mudah query FX history, calculate P&L, regulatory reporting
3. **Flexibility:** FX rate bisa dari berbagai provider, dengan berbagai margin

---

## 7. Trade-offs

### Kelebihan Desain Baru

| Aspect | Benefit |
|--------|---------|
| Correctness | Tidak violate TigerBeetle constraints |
| Clarity | Jelas bahwa FX = dua operasi terpisah |
| Auditability | CorrelationID memudahkan tracing |
| Flexibility | Bisa handle partial failure gracefully |

### Kekurangan

| Aspect | Drawback | Mitigation |
|--------|----------|------------|
| Not atomic cross-ledger | Bisa partial failure | Compensating transactions |
| More complex | 2 chains instead of 1 | Well-documented, clear API |
| Coordination overhead | Application must orchestrate | Service layer handles this |

### Acceptable Trade-off?

**Ya.** Karena:

1. TigerBeetle constraint adalah **hard requirement** — tidak ada pilihan lain
2. Financial systems selalu butuh compensation mechanisms anyway
3. Complexity ada di service layer, bukan di ledger layer

---

## 8. Testing Checklist

```
[ ] Source chain executes atomically in source ledger
[ ] Destination chain executes atomically in dest ledger
[ ] CorrelationID sama di semua transfers
[ ] Partial failure triggers compensation
[ ] FX details recorded in PostgreSQL
[ ] Reconciliation can link both chains via CorrelationID
```

---

## 9. References

- [TigerBeetle Transfer Constraints](https://docs.tigerbeetle.com/reference/transfer#constraints)
- [Double-Entry Bookkeeping](https://en.wikipedia.org/wiki/Double-entry_bookkeeping)
- [Saga Pattern for Distributed Transactions](https://microservices.io/patterns/data/saga.html)

---

## Appendix: Error Handling Improvement

Sebagai bagian dari fix ini, `SimpleTransferChain` dan builder methods juga diperbaiki untuk **properly handle errors**:

```go
// ❌ OLD: Error diabaikan
builder.Add(wallet, pendingOut, amount, ledger, code)  // returns (*Builder, error)

// ✅ NEW: Error di-handle
if _, err := builder.Add(wallet, pendingOut, amount, ledger, code); err != nil {
    return nil, fmt.Errorf("add debit step: %w", err)
}
```

Ini penting karena `NewTransfer()` bisa fail saat generate UUID v7.
