package models

import (
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Wallet represents a tenant's currency wallet linked to TigerBeetle.
type Wallet struct {
	ID            uuid.UUID
	TenantID      uuid.UUID
	Currency      string
	TBAccountID   *big.Int // 128-bit TigerBeetle account ID
	CachedBalance decimal.Decimal
	CachedPending decimal.Decimal
	CachedAt      time.Time
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// IsActive returns true if the wallet is active.
func (w *Wallet) IsActive() bool {
	return w.Status == "active"
}

// AvailableBalance returns the available balance (cached - pending).
func (w *Wallet) AvailableBalance() decimal.Decimal {
	return w.CachedBalance.Sub(w.CachedPending)
}

// HasSufficientBalance checks if the wallet has enough available balance.
func (w *Wallet) HasSufficientBalance(amount decimal.Decimal) bool {
	return w.AvailableBalance().GreaterThanOrEqual(amount)
}

// CreateWalletParams contains parameters for creating a new wallet.
type CreateWalletParams struct {
	TenantID    uuid.UUID
	Currency    string
	TBAccountID *big.Int
}

// WalletBalance represents a wallet's balance state.
type WalletBalance struct {
	Available decimal.Decimal
	Pending   decimal.Decimal
	Total     decimal.Decimal
}
