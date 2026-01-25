# Fix #002: Mengapa "Atomic Transfer Settlement" Tidak Cocok untuk FX

**Date:** 2025-01-25
**Related:** Fix #001 (FX Transfer Chain Redesign)

---

## Pertanyaan Utama

> "Kenapa desain lama dengan satu atomic chain tidak bisa dipakai untuk FX?"

Dokumen ini menjelaskan **alasan fundamental** — bukan hanya "TigerBeetle tidak support", tapi **mengapa secara konseptual memang tidak seharusnya begitu**.

---

## 1. Apa Arti "Atomic" di TigerBeetle?

### Definisi Atomic

Dalam konteks TigerBeetle, **atomic** berarti:

```
Semua transfer dalam linked chain BERHASIL SEMUA atau GAGAL SEMUA.
Tidak ada state "setengah jadi".
```

### Contoh Atomic yang Valid

```
┌─────────────────────────────────────────────────────────────┐
│  ATOMIC CHAIN: Transfer IDR 1,000,000 dengan fee           │
│  Ledger: IDR (360)                                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Transfer 1 (linked=true)                                   │
│  WALLET_IDR ────── 1,000,000 ──────► PENDING_OUT_IDR       │
│                                                             │
│  Transfer 2 (linked=true)                                   │
│  PENDING_OUT_IDR ──── 15,000 ──────► FEE_REVENUE_IDR       │
│                                                             │
│  Transfer 3 (linked=false, last)                            │
│  PENDING_OUT_IDR ─── 985,000 ──────► SETTLEMENT_IDR        │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│  Hasil:                                                     │
│  - Jika semua valid → semua execute                         │
│  - Jika satu gagal → semua rollback                         │
│  - Balance selalu konsisten                                 │
└─────────────────────────────────────────────────────────────┘
```

**Ini valid karena:**
- Semua account di ledger yang sama (IDR = 360)
- Semua amount dalam unit yang sama (IDR minor units)
- Double-entry balance: total debit = total credit

---

## 2. Mengapa FX Tidak Bisa Atomic dalam Satu Chain?

### Masalah Fundamental: Unit Tidak Sama

FX melibatkan **dua unit berbeda** yang tidak bisa dijumlahkan:

```
100 EUR ≠ 1,700,000 IDR (dalam konteks ledger)
```

Meskipun secara bisnis kita tahu nilainya "equivalent", dalam ledger:
- 100 EUR adalah 100 EUR
- 1,700,000 IDR adalah 1,700,000 IDR
- Keduanya **tidak bisa** dijumlahkan atau di-net

### Analogi: Apel dan Jeruk

```
┌─────────────────────────────────────────────────────────────┐
│  ANALOGI                                                    │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Kamu punya 2 gudang:                                       │
│  - Gudang A: menyimpan APEL                                 │
│  - Gudang B: menyimpan JERUK                                │
│                                                             │
│  Pertanyaan: Bisakah kamu "transfer" 10 apel dari Gudang A  │
│  dan langsung jadi 50 jeruk di Gudang B dalam SATU operasi? │
│                                                             │
│  Jawaban: TIDAK.                                            │
│                                                             │
│  Yang terjadi sebenarnya:                                   │
│  1. Keluarkan 10 apel dari Gudang A                         │
│  2. (Di luar gudang) Tukar apel dengan jeruk                │
│  3. Masukkan 50 jeruk ke Gudang B                           │
│                                                             │
│  Step 1 dan 3 adalah operasi TERPISAH di gudang berbeda.    │
│  Step 2 adalah "bisnis" yang terjadi di luar sistem gudang. │
└─────────────────────────────────────────────────────────────┘
```

### Diterjemahkan ke TigerBeetle

```
Gudang A     = EUR Ledger (978)
Gudang B     = IDR Ledger (360)
Apel         = EUR balance
Jeruk        = IDR balance
Pertukaran   = FX conversion (bisnis, bukan ledger operation)
```

---

## 3. Double-Entry Accounting dan FX

### Prinsip Double-Entry

Setiap transaksi harus balance:

```
Total Debit = Total Credit (dalam unit yang sama)
```

### Same-Currency: Balance Terjaga

```
Transfer IDR 1,000,000:

  Account          | Debit      | Credit
  -----------------|------------|----------
  WALLET_IDR       | 1,000,000  |
  SETTLEMENT_IDR   |            | 1,000,000
  -----------------|------------|----------
  TOTAL            | 1,000,000  | 1,000,000  ✓ BALANCE
```

### Cross-Currency: Tidak Bisa Balance dalam Satu Transaksi

```
Transfer EUR 100 → IDR 1,700,000:

  Account          | Debit      | Credit     | Unit
  -----------------|------------|------------|------
  WALLET_EUR       | 100        |            | EUR
  SETTLEMENT_IDR   |            | 1,700,000  | IDR
  -----------------|------------|------------|------
  TOTAL            | 100 EUR    | 1,700,000 IDR

  ❌ TIDAK BALANCE - unit berbeda!
```

### Solusi Akuntansi yang Benar

Dalam akuntansi profesional, FX dicatat sebagai **DUA transaksi terpisah**:

```
TRANSAKSI 1 (EUR Ledger):
  Account          | Debit      | Credit     | Unit
  -----------------|------------|------------|------
  WALLET_EUR       | 100        |            | EUR
  FX_POSITION_EUR  |            | 100        | EUR
  -----------------|------------|------------|------
  TOTAL            | 100        | 100        | ✓ BALANCE (EUR)

TRANSAKSI 2 (IDR Ledger):
  Account          | Debit      | Credit     | Unit
  -----------------|------------|------------|------
  FX_POSITION_IDR  | 1,700,000  |            | IDR
  SETTLEMENT_IDR   |            | 1,700,000  | IDR
  -----------------|------------|------------|------
  TOTAL            | 1,700,000  | 1,700,000  | ✓ BALANCE (IDR)
```

**Ini adalah cara yang benar dan dipakai oleh bank/financial institution.**

---

## 4. Desain Lama vs Desain Baru

### Desain Lama: "Magic Conversion"

```
┌─────────────────────────────────────────────────────────────┐
│  DESAIN LAMA                                                │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Asumsi: FX bisa dilakukan dalam satu atomic chain          │
│                                                             │
│  WALLET_EUR ──► PENDING_EUR ──► FX_EUR ──► FX_IDR ──► OUT   │
│                                      │         │            │
│                                      └────?────┘            │
│                                      Cross-ledger           │
│                                      "Magic happens here"   │
│                                                             │
│  Problems:                                                  │
│  1. TigerBeetle rejects cross-ledger transfer               │
│  2. Secara akuntansi tidak make sense                       │
│  3. Menyembunyikan kompleksitas yang seharusnya explicit    │
└─────────────────────────────────────────────────────────────┘
```

### Desain Baru: Explicit Two-Chain

```
┌─────────────────────────────────────────────────────────────┐
│  DESAIN BARU                                                │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Explicit: FX = dua operasi terpisah + koordinasi bisnis    │
│                                                             │
│  CHAIN 1 (EUR):                                             │
│  WALLET_EUR ──► PENDING_EUR ──► FX_POSITION_EUR             │
│                                       │                     │
│                                       │ (Platform now       │
│                                       │  holds EUR)         │
│                                       ▼                     │
│                              ┌─────────────────┐            │
│                              │ Business Logic  │            │
│                              │ - FX Rate       │            │
│                              │ - Correlation   │            │
│                              │ - PostgreSQL    │            │
│                              └────────┬────────┘            │
│                                       │                     │
│                                       ▼                     │
│  CHAIN 2 (IDR):              (Platform pays out IDR)        │
│  FX_POSITION_IDR ──► FEE_IDR ──► SETTLEMENT_IDR             │
│                                                             │
│  Benefits:                                                  │
│  1. Setiap chain balance dalam ledger masing-masing         │
│  2. FX conversion explicit di business layer                │
│  3. Auditable: bisa trace kedua sisi                        │
│  4. Sesuai standar akuntansi                                │
└─────────────────────────────────────────────────────────────┘
```

---

## 5. Bagaimana Bank Melakukan FX?

### Realita di Dunia Perbankan

Bank tidak pernah melakukan FX dalam "satu transaksi atomic". Prosesnya:

```
┌─────────────────────────────────────────────────────────────┐
│  BANK FX PROCESS                                            │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. CUSTOMER REQUEST                                        │
│     "Saya mau kirim EUR 100 ke Indonesia"                   │
│                                                             │
│  2. BANK DEBITS CUSTOMER (EUR Ledger)                       │
│     Customer EUR Account: -100 EUR                          │
│     Bank FX Position EUR: +100 EUR                          │
│                                                             │
│  3. BANK INTERNAL FX (Off-ledger / Treasury)                │
│     - Lock FX rate                                          │
│     - Calculate IDR amount                                  │
│     - Record FX P&L                                         │
│                                                             │
│  4. BANK CREDITS DESTINATION (IDR Ledger)                   │
│     Bank FX Position IDR: -1,700,000 IDR                    │
│     Nostro/Settlement:    +1,700,000 IDR                    │
│                                                             │
│  5. RECONCILIATION                                          │
│     - Match EUR debit with IDR credit                       │
│     - Verify FX rate applied correctly                      │
│     - Regulatory reporting                                  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Kovra sekarang mengikuti pola yang sama.**

---

## 6. FX Position: Platform sebagai Counterparty

### Konsep Counterparty

Dalam FX, selalu ada **dua pihak**:
- Pihak yang memberikan currency A
- Pihak yang memberikan currency B

Platform (Kovra) bertindak sebagai **counterparty**:

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│  USER                         KOVRA                         │
│  ─────                        ─────                         │
│                                                             │
│  "Saya kasih EUR 100"    ──►  "OK, saya terima EUR 100"     │
│                               FX_POSITION_EUR += 100        │
│                                                             │
│                          ◄──  "Ini IDR 1,700,000"           │
│  "Saya terima IDR"            FX_POSITION_IDR -= 1,700,000  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### FX Position Accounting

```
FX_POSITION_EUR (Platform's EUR holdings):
┌──────────────────────────────────────────┐
│  Credits (+)    │  Debits (-)            │
├──────────────────────────────────────────┤
│  User deposits  │  Platform sells EUR    │
│  FX receipts    │  User withdrawals      │
└──────────────────────────────────────────┘

FX_POSITION_IDR (Platform's IDR holdings):
┌──────────────────────────────────────────┐
│  Credits (+)    │  Debits (-)            │
├──────────────────────────────────────────┤
│  Treasury fund  │  FX payouts            │
│  IDR receipts   │  Platform expenses     │
└──────────────────────────────────────────┘
```

Platform harus **maintain balance** di kedua position melalui treasury operations.

---

## 7. Mengapa Error Handling Juga Diperbaiki?

### Konteks

Saat memperbaiki FX chain, saya juga menemukan bahwa error dari `TransferBuilder.Add()` diabaikan:

```go
// ❌ LAMA: Error diabaikan
builder.Add(wallet, pending, amount, ledger, code)  // returns error, ignored!
```

### Kenapa Ini Berbahaya?

`Add()` memanggil `NewTransfer()` yang generate UUID v7:

```go
func NewTransfer(...) (Transfer, error) {
    id, err := uuid.NewV7()  // Bisa fail!
    if err != nil {
        return Transfer{}, fmt.Errorf("generate transfer id: %w", err)
    }
    // ...
}
```

UUID v7 generation bisa fail karena:
- Clock issues (system time tidak available)
- Random source exhausted (rare, tapi possible)

### Dalam Konteks FX yang Baru

Dengan desain baru, kita build **dua chain terpisah**. Jika error diabaikan:

```
Chain 1: Build success (atau kita pikir success)
Chain 2: Build fails silently

Execute Chain 1: Success
Execute Chain 2: ??? (malformed transfers)

Result: Partial execution, inconsistent state
```

### Fix

```go
// ✅ BARU: Error di-handle explicit
if _, err := builder.Add(wallet, pending, amount, ledger, code); err != nil {
    return FXTransferPair{}, fmt.Errorf("add source step: %w", err)
}
```

---

## 8. Diagram Perbandingan Final

### Sebelum (Incorrect Mental Model)

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│   EUR World                              IDR World          │
│   ─────────                              ─────────          │
│                                                             │
│   [WALLET] ──► [PENDING] ──► [FX] ═══════► [FX] ──► [OUT]  │
│                               │             │               │
│                               └─────────────┘               │
│                               "Teleportation"               │
│                               (IMPOSSIBLE)                  │
│                                                             │
│   Mindset: "FX adalah satu transaksi"                       │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Sesudah (Correct Mental Model)

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│   EUR World                              IDR World          │
│   ─────────                              ─────────          │
│                                                             │
│   [WALLET]                               [FX_POS]           │
│      │                                      │               │
│      ▼                                      ▼               │
│   [PENDING]                              [FEE]              │
│      │                                      │               │
│      ▼                                      ▼               │
│   [FX_POS] ◄─────── Platform ───────► [SETTLE]             │
│            receives    │     pays out                       │
│            EUR         │     IDR                            │
│                        │                                    │
│                   ┌────┴────┐                               │
│                   │ Business│                               │
│                   │  Logic  │                               │
│                   │ ─────── │                               │
│                   │ FX Rate │                               │
│                   │ Margin  │                               │
│                   │ Logging │                               │
│                   └─────────┘                               │
│                                                             │
│   Mindset: "FX adalah exchange between two parties"         │
│            "Platform adalah counterparty"                   │
│            "Dua ledger movement + business coordination"    │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 9. Summary

| Aspect | Desain Lama | Desain Baru |
|--------|-------------|-------------|
| Mental Model | FX = satu transaksi | FX = dua transaksi + koordinasi |
| Ledger Operation | Cross-ledger (invalid) | Single-ledger per chain |
| Accounting | Tidak balance | Balance per ledger |
| Platform Role | Passthrough | Counterparty |
| Error Handling | Ignored | Explicit |
| Auditability | Hidden complexity | Transparent |
| Industry Standard | No | Yes (follows bank pattern) |

### Kesimpulan

**"Atomic Transfer Settlement" dalam konteks single chain tidak cocok untuk FX karena:**

1. **Physically impossible** — TigerBeetle constraint
2. **Conceptually wrong** — FX bukan "teleportasi", tapi exchange
3. **Accounting violation** — Debit EUR ≠ Credit IDR
4. **Hides complexity** — Seharusnya explicit di business layer

**Desain baru lebih baik karena:**

1. **Correct** — Sesuai constraint dan akuntansi
2. **Transparent** — Complexity terlihat, bisa di-audit
3. **Industry standard** — Sama seperti bank melakukan FX
4. **Maintainable** — Jelas dimana FX logic berada
