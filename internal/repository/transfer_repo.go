package repository

import (
	"context"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"kovra/internal/models"
	"kovra/internal/repository/queries"
)

// TransferRepository handles transfer data access.
type TransferRepository struct {
	q *queries.Queries
}

// NewTransferRepository creates a new transfer repository.
func NewTransferRepository(pool *pgxpool.Pool) *TransferRepository {
	return &TransferRepository{q: queries.New(pool)}
}

// Create creates a new transfer.
func (r *TransferRepository) Create(ctx context.Context, params models.CreateTransferParams) (*models.Transfer, error) {
	row, err := r.q.CreateTransfer(ctx, queries.CreateTransferParams{
		TenantID:            params.TenantID,
		SourceLegalEntityID: uuidToNullable(params.SourceLegalEntityID),
		DestLegalEntityID:   uuidToNullable(params.DestLegalEntityID),
		QuoteID:             uuidToNullable(params.QuoteID),
		RecipientID:         uuidToNullable(params.RecipientID),
		IdempotencyKey:      stringToNullable(params.IdempotencyKey),
		FromCurrency:        params.FromCurrency,
		ToCurrency:          params.ToCurrency,
		FromAmount:          decimalToNumeric(params.FromAmount),
		ToAmount:            decimalToNumeric(params.ToAmount),
		FxRate:              decimalToNumeric(params.FXRate),
		TotalFee:            decimalToNumeric(params.TotalFee),
		Rail:                railToNullable(params.Rail),
	})
	if err != nil {
		return nil, err
	}

	return r.toModel(row), nil
}

// GetByID retrieves a transfer by ID.
func (r *TransferRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Transfer, error) {
	row, err := r.q.GetTransferByID(ctx, id)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// GetByIdempotencyKey retrieves a transfer by tenant and idempotency key.
func (r *TransferRepository) GetByIdempotencyKey(ctx context.Context, tenantID uuid.UUID, key string) (*models.Transfer, error) {
	row, err := r.q.GetTransferByIdempotencyKey(ctx, queries.GetTransferByIdempotencyKeyParams{
		TenantID:       tenantID,
		IdempotencyKey: pgtype.Text{String: key, Valid: true},
	})
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// UpdateStatus updates the transfer status.
func (r *TransferRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.TransferStatus, failureReason *string) error {
	return r.q.UpdateTransferStatus(ctx, queries.UpdateTransferStatusParams{
		ID:            id,
		Status:        queries.TransferStatusEnum(status),
		FailureReason: stringToNullable(failureReason),
	})
}

// UpdateTBTransferIDs updates the TigerBeetle transfer IDs.
func (r *TransferRepository) UpdateTBTransferIDs(ctx context.Context, id uuid.UUID, tbIDs []*big.Int) error {
	numericIDs := make([]pgtype.Numeric, len(tbIDs))
	for i, n := range tbIDs {
		numericIDs[i] = bigIntToNumeric(n)
	}

	return r.q.UpdateTransferTBTransferIDs(ctx, queries.UpdateTransferTBTransferIDsParams{
		ID:            id,
		TbTransferIds: numericIDs,
	})
}

// ListByTenant retrieves transfers for a tenant with filters.
func (r *TransferRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, filter models.TransferFilter) ([]*models.Transfer, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	rows, err := r.q.ListTransfersByTenant(ctx, queries.ListTransfersByTenantParams{
		TenantID:         tenantID,
		Limit:            int32(limit),
		Offset:           int32(offset),
		Status:           transferStatusToNullable(filter.Status),
		FromCurrency:     stringPtrToNullable(filter.FromCurrency),
		ToCurrency:       stringPtrToNullable(filter.ToCurrency),
		ComplianceRegion: complianceRegionToNullable(filter.ComplianceRegion),
		UpdatedAfter:     timeToNullable(filter.UpdatedAfter),
		UpdatedBefore:    timeToNullable(filter.UpdatedBefore),
	})
	if err != nil {
		return nil, err
	}

	return r.toModels(rows), nil
}

func (r *TransferRepository) toModel(row queries.Transfer) *models.Transfer {
	t := &models.Transfer{
		ID:               row.ID,
		TenantID:         row.TenantID,
		FromCurrency:     row.FromCurrency,
		ToCurrency:       row.ToCurrency,
		FromAmount:       numericToDecimal(row.FromAmount),
		ToAmount:         numericToDecimal(row.ToAmount),
		FXRate:           numericToDecimal(row.FxRate),
		TotalFee:         numericToDecimal(row.TotalFee),
		Status:           models.TransferStatus(row.Status),
		IsNetted:         row.IsNetted,
		ComplianceStatus: row.ComplianceStatus,
		ComplianceRegion: models.ComplianceRegion(row.ComplianceRegion),
		UpdatedAt:        row.UpdatedAt,
	}

	if row.SourceLegalEntityID.Valid {
		id := uuid.UUID(row.SourceLegalEntityID.Bytes)
		t.SourceLegalEntityID = &id
	}
	if row.DestLegalEntityID.Valid {
		id := uuid.UUID(row.DestLegalEntityID.Bytes)
		t.DestLegalEntityID = &id
	}
	if row.QuoteID.Valid {
		id := uuid.UUID(row.QuoteID.Bytes)
		t.QuoteID = &id
	}
	if row.BatchID.Valid {
		id := uuid.UUID(row.BatchID.Bytes)
		t.BatchID = &id
	}
	if row.RecipientID.Valid {
		id := uuid.UUID(row.RecipientID.Bytes)
		t.RecipientID = &id
	}
	if row.IdempotencyKey.Valid {
		t.IdempotencyKey = &row.IdempotencyKey.String
	}
	if row.FailureReason.Valid {
		t.FailureReason = &row.FailureReason.String
	}
	if row.Rail.Valid {
		rail := models.Rail(row.Rail.RailEnum)
		t.Rail = &rail
	}
	if row.RailReference.Valid {
		t.RailReference = &row.RailReference.String
	}
	if row.NettingGroupID.Valid {
		id := uuid.UUID(row.NettingGroupID.Bytes)
		t.NettingGroupID = &id
	}
	if row.RiskScore.Valid {
		score := int(row.RiskScore.Int32)
		t.RiskScore = &score
	}
	if row.ScreenedAt.Valid {
		t.ScreenedAt = &row.ScreenedAt.Time
	}
	if row.CompletedAt.Valid {
		t.CompletedAt = &row.CompletedAt.Time
	}

	// Convert TBTransferIDs
	if len(row.TbTransferIds) > 0 {
		t.TBTransferIDs = make([]*big.Int, len(row.TbTransferIds))
		for i, n := range row.TbTransferIds {
			t.TBTransferIDs[i] = numericToBigInt(n)
		}
	}

	return t
}

func (r *TransferRepository) toModels(rows []queries.Transfer) []*models.Transfer {
	result := make([]*models.Transfer, len(rows))
	for i, row := range rows {
		result[i] = r.toModel(row)
	}
	return result
}

// Additional helper functions
func stringToNullable(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func stringPtrToNullable(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func railToNullable(r *models.Rail) queries.NullRailEnum {
	if r == nil {
		return queries.NullRailEnum{}
	}
	return queries.NullRailEnum{RailEnum: queries.RailEnum(*r), Valid: true}
}

func transferStatusToNullable(s *models.TransferStatus) queries.NullTransferStatusEnum {
	if s == nil {
		return queries.NullTransferStatusEnum{}
	}
	return queries.NullTransferStatusEnum{TransferStatusEnum: queries.TransferStatusEnum(*s), Valid: true}
}

func complianceRegionToNullable(r *models.ComplianceRegion) pgtype.Text {
	if r == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: string(*r), Valid: true}
}

func timeToNullable(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}
