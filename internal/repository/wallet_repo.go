package repository

import (
	"context"
	"math/big"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"kovra/internal/models"
	"kovra/internal/repository/queries"
)

// WalletRepository handles wallet data access.
type WalletRepository struct {
	q *queries.Queries
}

// NewWalletRepository creates a new wallet repository.
func NewWalletRepository(pool *pgxpool.Pool) *WalletRepository {
	return &WalletRepository{q: queries.New(pool)}
}

// Create creates a new wallet.
func (r *WalletRepository) Create(ctx context.Context, params models.CreateWalletParams) (*models.Wallet, error) {
	row, err := r.q.CreateWallet(ctx, queries.CreateWalletParams{
		TenantID:    params.TenantID,
		Currency:    params.Currency,
		TbAccountID: bigIntToNumeric(params.TBAccountID),
	})
	if err != nil {
		return nil, err
	}

	return r.toModel(row), nil
}

// GetByID retrieves a wallet by ID.
func (r *WalletRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
	row, err := r.q.GetWalletByID(ctx, id)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// GetByTenantAndCurrency retrieves a wallet by tenant ID and currency.
func (r *WalletRepository) GetByTenantAndCurrency(ctx context.Context, tenantID uuid.UUID, currency string) (*models.Wallet, error) {
	row, err := r.q.GetWalletByTenantAndCurrency(ctx, queries.GetWalletByTenantAndCurrencyParams{
		TenantID: tenantID,
		Currency: currency,
	})
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// ListByTenant retrieves all wallets for a tenant.
func (r *WalletRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.Wallet, error) {
	rows, err := r.q.ListWalletsByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return r.toModels(rows), nil
}

// UpdateCachedBalance updates the cached balance from TigerBeetle.
func (r *WalletRepository) UpdateCachedBalance(ctx context.Context, id uuid.UUID, balance, pending decimal.Decimal) error {
	return r.q.UpdateWalletCachedBalance(ctx, queries.UpdateWalletCachedBalanceParams{
		ID:            id,
		CachedBalance: decimalToNumeric(balance),
		CachedPending: decimalToNumeric(pending),
	})
}

// UpdateStatus updates the wallet status.
func (r *WalletRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return r.q.UpdateWalletStatus(ctx, queries.UpdateWalletStatusParams{
		ID:     id,
		Status: status,
	})
}

func (r *WalletRepository) toModel(row queries.Wallet) *models.Wallet {
	w := &models.Wallet{
		ID:            row.ID,
		TenantID:      row.TenantID,
		Currency:      row.Currency,
		CachedBalance: numericToDecimal(row.CachedBalance),
		CachedPending: numericToDecimal(row.CachedPending),
		CachedAt:      row.CachedAt,
		Status:        row.Status,
		UpdatedAt:     row.UpdatedAt,
	}

	// Convert TBAccountID
	w.TBAccountID = numericToBigInt(row.TbAccountID)

	return w
}

func (r *WalletRepository) toModels(rows []queries.Wallet) []*models.Wallet {
	result := make([]*models.Wallet, len(rows))
	for i, row := range rows {
		result[i] = r.toModel(row)
	}
	return result
}

// Helper functions for numeric conversions
func bigIntToNumeric(b *big.Int) pgtype.Numeric {
	if b == nil {
		return pgtype.Numeric{}
	}
	var n pgtype.Numeric
	n.Scan(b.String())
	return n
}

func numericToBigInt(n pgtype.Numeric) *big.Int {
	if !n.Valid {
		return nil
	}
	result := new(big.Int)
	result.SetString(n.Int.String(), 10)
	return result
}

func decimalToNumeric(d decimal.Decimal) pgtype.Numeric {
	var n pgtype.Numeric
	n.Scan(d.String())
	return n
}

func numericToDecimal(n pgtype.Numeric) decimal.Decimal {
	if !n.Valid {
		return decimal.Zero
	}
	d, _ := decimal.NewFromString(numericToString(n))
	return d
}

func numericToString(n pgtype.Numeric) string {
	if !n.Valid {
		return "0"
	}
	// Handle the numeric value properly
	if n.Exp == 0 {
		return n.Int.String()
	}
	// For decimals with exponent
	d := decimal.NewFromBigInt(n.Int, n.Exp)
	return d.String()
}
