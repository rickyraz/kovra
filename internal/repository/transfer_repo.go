package repository

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"kovra/internal/models"
)

// TransferRepository handles transfer data access.
type TransferRepository struct {
	pool *pgxpool.Pool
}

// NewTransferRepository creates a new transfer repository.
func NewTransferRepository(pool *pgxpool.Pool) *TransferRepository {
	return &TransferRepository{pool: pool}
}

// Create creates a new transfer.
func (r *TransferRepository) Create(ctx context.Context, params models.CreateTransferParams) (*models.Transfer, error) {
	query := `
		INSERT INTO transfers (
			tenant_id, source_legal_entity_id, dest_legal_entity_id, quote_id, recipient_id,
			idempotency_key, from_currency, to_currency, from_amount, to_amount, fx_rate, total_fee, rail
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, tenant_id, source_legal_entity_id, dest_legal_entity_id, quote_id, batch_id, recipient_id,
			idempotency_key, from_currency, to_currency, from_amount, to_amount, fx_rate, total_fee,
			status, failure_reason, rail, rail_reference, netting_group_id, is_netted,
			tb_transfer_ids, risk_score, compliance_status, screened_at, compliance_region,
			created_at, updated_at, completed_at`

	row := r.pool.QueryRow(ctx, query,
		params.TenantID,
		params.SourceLegalEntityID,
		params.DestLegalEntityID,
		params.QuoteID,
		params.RecipientID,
		params.IdempotencyKey,
		params.FromCurrency,
		params.ToCurrency,
		params.FromAmount,
		params.ToAmount,
		params.FXRate,
		params.TotalFee,
		params.Rail,
	)

	return r.scan(row)
}

// GetByID retrieves a transfer by ID.
func (r *TransferRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Transfer, error) {
	query := `
		SELECT id, tenant_id, source_legal_entity_id, dest_legal_entity_id, quote_id, batch_id, recipient_id,
			idempotency_key, from_currency, to_currency, from_amount, to_amount, fx_rate, total_fee,
			status, failure_reason, rail, rail_reference, netting_group_id, is_netted,
			tb_transfer_ids, risk_score, compliance_status, screened_at, compliance_region,
			created_at, updated_at, completed_at
		FROM transfers
		WHERE id = $1`

	row := r.pool.QueryRow(ctx, query, id)
	transfer, err := r.scan(row)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return transfer, err
}

// GetByIdempotencyKey retrieves a transfer by tenant and idempotency key.
func (r *TransferRepository) GetByIdempotencyKey(ctx context.Context, tenantID uuid.UUID, key string) (*models.Transfer, error) {
	query := `
		SELECT id, tenant_id, source_legal_entity_id, dest_legal_entity_id, quote_id, batch_id, recipient_id,
			idempotency_key, from_currency, to_currency, from_amount, to_amount, fx_rate, total_fee,
			status, failure_reason, rail, rail_reference, netting_group_id, is_netted,
			tb_transfer_ids, risk_score, compliance_status, screened_at, compliance_region,
			created_at, updated_at, completed_at
		FROM transfers
		WHERE tenant_id = $1 AND idempotency_key = $2`

	row := r.pool.QueryRow(ctx, query, tenantID, key)
	transfer, err := r.scan(row)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return transfer, err
}

// UpdateStatus updates the transfer status.
func (r *TransferRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.TransferStatus, failureReason *string) error {
	query := `
		UPDATE transfers
		SET status = $2, failure_reason = $3, updated_at = NOW(),
			completed_at = CASE WHEN $2 IN ('completed', 'rolled_back', 'cancelled', 'rejected') THEN NOW() ELSE completed_at END
		WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, id, status, failureReason)
	return err
}

// UpdateTBTransferIDs updates the TigerBeetle transfer IDs.
func (r *TransferRepository) UpdateTBTransferIDs(ctx context.Context, id uuid.UUID, tbIDs []*big.Int) error {
	// Convert to array of numeric strings
	numericIDs := make([]string, len(tbIDs))
	for i, n := range tbIDs {
		numericIDs[i] = n.String()
	}

	query := `
		UPDATE transfers
		SET tb_transfer_ids = $2::numeric[], updated_at = NOW()
		WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, id, numericIDs)
	return err
}

// ListByTenant retrieves transfers for a tenant with filters.
func (r *TransferRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, filter models.TransferFilter) ([]*models.Transfer, error) {
	var conditions []string
	var args []any
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argNum))
	args = append(args, tenantID)
	argNum++

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
		args = append(args, *filter.Status)
		argNum++
	}

	if filter.FromCurrency != nil {
		conditions = append(conditions, fmt.Sprintf("from_currency = $%d", argNum))
		args = append(args, *filter.FromCurrency)
		argNum++
	}

	if filter.ToCurrency != nil {
		conditions = append(conditions, fmt.Sprintf("to_currency = $%d", argNum))
		args = append(args, *filter.ToCurrency)
		argNum++
	}

	if filter.ComplianceRegion != nil {
		conditions = append(conditions, fmt.Sprintf("compliance_region = $%d", argNum))
		args = append(args, *filter.ComplianceRegion)
		argNum++
	}

	if filter.CreatedAfter != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argNum))
		args = append(args, *filter.CreatedAfter)
		argNum++
	}

	if filter.CreatedBefore != nil {
		conditions = append(conditions, fmt.Sprintf("created_at < $%d", argNum))
		args = append(args, *filter.CreatedBefore)
		argNum++
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, source_legal_entity_id, dest_legal_entity_id, quote_id, batch_id, recipient_id,
			idempotency_key, from_currency, to_currency, from_amount, to_amount, fx_rate, total_fee,
			status, failure_reason, rail, rail_reference, netting_group_id, is_netted,
			tb_transfer_ids, risk_score, compliance_status, screened_at, compliance_region,
			created_at, updated_at, completed_at
		FROM transfers
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`,
		strings.Join(conditions, " AND "),
		argNum,
		argNum+1,
	)
	args = append(args, limit, offset)

	return r.scanMany(ctx, query, args...)
}

func (r *TransferRepository) scanMany(ctx context.Context, query string, args ...any) ([]*models.Transfer, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query transfers: %w", err)
	}
	defer rows.Close()

	var transfers []*models.Transfer
	for rows.Next() {
		t, err := r.scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan transfer: %w", err)
		}
		transfers = append(transfers, t)
	}

	return transfers, rows.Err()
}

func (r *TransferRepository) scan(s scanner) (*models.Transfer, error) {
	var t models.Transfer
	var tbTransferIDs []pgtype.Numeric

	err := s.Scan(
		&t.ID,
		&t.TenantID,
		&t.SourceLegalEntityID,
		&t.DestLegalEntityID,
		&t.QuoteID,
		&t.BatchID,
		&t.RecipientID,
		&t.IdempotencyKey,
		&t.FromCurrency,
		&t.ToCurrency,
		&t.FromAmount,
		&t.ToAmount,
		&t.FXRate,
		&t.TotalFee,
		&t.Status,
		&t.FailureReason,
		&t.Rail,
		&t.RailReference,
		&t.NettingGroupID,
		&t.IsNetted,
		&tbTransferIDs,
		&t.RiskScore,
		&t.ComplianceStatus,
		&t.ScreenedAt,
		&t.ComplianceRegion,
		&t.CreatedAt,
		&t.UpdatedAt,
		&t.CompletedAt,
	)
	if err != nil {
		return nil, err
	}

	// Convert pgtype.Numeric array to []*big.Int
	if len(tbTransferIDs) > 0 {
		t.TBTransferIDs = make([]*big.Int, len(tbTransferIDs))
		for i, n := range tbTransferIDs {
			if n.Valid {
				t.TBTransferIDs[i] = new(big.Int)
				t.TBTransferIDs[i].SetString(n.Int.String(), 10)
			}
		}
	}

	return &t, nil
}
