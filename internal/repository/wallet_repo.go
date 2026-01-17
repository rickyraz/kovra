package repository

import (
	"context"
	"fmt"
	"math/big"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"kovra/internal/models"
)

// WalletRepository handles wallet data access.
type WalletRepository struct {
	pool *pgxpool.Pool
}

// NewWalletRepository creates a new wallet repository.
func NewWalletRepository(pool *pgxpool.Pool) *WalletRepository {
	return &WalletRepository{pool: pool}
}

// Create creates a new wallet.
func (r *WalletRepository) Create(ctx context.Context, params models.CreateWalletParams) (*models.Wallet, error) {
	query := `
		INSERT INTO wallets (tenant_id, currency, tb_account_id)
		VALUES ($1, $2, $3)
		RETURNING id, tenant_id, currency, tb_account_id, cached_balance, cached_pending, cached_at, status, created_at, updated_at`

	// Convert big.Int to pgtype.Numeric
	tbAccountID := pgtype.Numeric{}
	tbAccountID.Scan(params.TBAccountID.String())

	row := r.pool.QueryRow(ctx, query,
		params.TenantID,
		params.Currency,
		tbAccountID,
	)

	return r.scan(row)
}

// GetByID retrieves a wallet by ID.
func (r *WalletRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
	query := `
		SELECT id, tenant_id, currency, tb_account_id, cached_balance, cached_pending, cached_at, status, created_at, updated_at
		FROM wallets
		WHERE id = $1`

	row := r.pool.QueryRow(ctx, query, id)
	wallet, err := r.scan(row)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return wallet, err
}

// GetByTenantAndCurrency retrieves a wallet by tenant ID and currency.
func (r *WalletRepository) GetByTenantAndCurrency(ctx context.Context, tenantID uuid.UUID, currency string) (*models.Wallet, error) {
	query := `
		SELECT id, tenant_id, currency, tb_account_id, cached_balance, cached_pending, cached_at, status, created_at, updated_at
		FROM wallets
		WHERE tenant_id = $1 AND currency = $2`

	row := r.pool.QueryRow(ctx, query, tenantID, currency)
	wallet, err := r.scan(row)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return wallet, err
}

// ListByTenant retrieves all wallets for a tenant.
func (r *WalletRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.Wallet, error) {
	query := `
		SELECT id, tenant_id, currency, tb_account_id, cached_balance, cached_pending, cached_at, status, created_at, updated_at
		FROM wallets
		WHERE tenant_id = $1
		ORDER BY currency`

	return r.scanMany(ctx, query, tenantID)
}

// UpdateCachedBalance updates the cached balance from TigerBeetle.
func (r *WalletRepository) UpdateCachedBalance(ctx context.Context, id uuid.UUID, balance, pending decimal.Decimal) error {
	query := `
		UPDATE wallets
		SET cached_balance = $2, cached_pending = $3, cached_at = NOW(), updated_at = NOW()
		WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, id, balance, pending)
	return err
}

// UpdateStatus updates the wallet status.
func (r *WalletRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `
		UPDATE wallets
		SET status = $2, updated_at = NOW()
		WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, id, status)
	return err
}

func (r *WalletRepository) scanMany(ctx context.Context, query string, args ...any) ([]*models.Wallet, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query wallets: %w", err)
	}
	defer rows.Close()

	var wallets []*models.Wallet
	for rows.Next() {
		w, err := r.scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan wallet: %w", err)
		}
		wallets = append(wallets, w)
	}

	return wallets, rows.Err()
}

func (r *WalletRepository) scan(s scanner) (*models.Wallet, error) {
	var w models.Wallet
	var tbAccountID pgtype.Numeric

	err := s.Scan(
		&w.ID,
		&w.TenantID,
		&w.Currency,
		&tbAccountID,
		&w.CachedBalance,
		&w.CachedPending,
		&w.CachedAt,
		&w.Status,
		&w.CreatedAt,
		&w.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Convert pgtype.Numeric to big.Int
	if tbAccountID.Valid {
		// Get the string representation and parse it
		numStr := tbAccountID.Int.String()
		w.TBAccountID = new(big.Int)
		w.TBAccountID.SetString(numStr, 10)
	}

	return &w, nil
}
