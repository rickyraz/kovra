## Plan Mode

- Make the plan extremely concise. Sacrifice grammar for the sake of concision.
- At the end of each plan, give me a list of unresolved questions to answer, if any.

## Go Struct Pointer Guidelines

### Kapan HARUS pakai pointer (`*Struct` / `&Struct{}`)

1. **Method perlu mutasi receiver** → wajib pointer receiver
2. **Struct besar (>200-500 bytes)** → hindari copy mahal
3. **Butuh nil semantic** → represent "no value"
4. **Service/Handler dengan dependency injection** → konsistensi, lifetime management
5. **Interface implementation** → sering butuh pointer receiver untuk consistency

### Kapan JANGAN pakai pointer

1. **Struct kecil (1-4 field primitive)** read-only → value lebih cepat, zero heap alloc
2. **DTO / Request-Response struct** → sekali pakai, kecil
3. **Value objects immutable** → Price, Money, Coordinate, dll
4. **Return dari function yang tidak perlu mutasi** → prefer value

### Rules untuk Codebase Ini

```go
// ❌ BURUK - struct kecil, read-only, tidak perlu pointer
err := &ErrorInfo{Code: "ERR", Message: "msg"}
price := &Price{Amount: 100, Currency: "IDR"}

// ✅ BAIK - value untuk struct kecil read-only
err := ErrorInfo{Code: "ERR", Message: "msg"}
price := Price{Amount: 100, Currency: "IDR"}

// ✅ OK - pointer untuk service/handler (DI pattern, lifetime)
func NewWalletHandler(repo *WalletRepository) *WalletHandler {
    return &WalletHandler{repo: repo}
}

// ✅ OK - pointer untuk struct besar atau butuh mutasi
func (w *Wallet) UpdateBalance(amount decimal.Decimal) { ... }
```

### Checklist Review Pointer

- [ ] Struct < 5 field primitif? → coba value dulu
- [ ] Hanya dibaca setelah create? → value
- [ ] Perlu nil check? → pointer
- [ ] Method mengubah field? → pointer receiver
- [ ] Dependency injection? → pointer OK untuk consistency