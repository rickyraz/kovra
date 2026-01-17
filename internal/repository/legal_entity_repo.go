package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"kovra/internal/models"
	"kovra/internal/repository/queries"
)

// LegalEntityRepository handles legal entity data access.
type LegalEntityRepository struct {
	q *queries.Queries
}

// NewLegalEntityRepository creates a new legal entity repository.
func NewLegalEntityRepository(pool *pgxpool.Pool) *LegalEntityRepository {
	return &LegalEntityRepository{q: queries.New(pool)}
}

// GetByID retrieves a legal entity by ID.
func (r *LegalEntityRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.LegalEntity, error) {
	row, err := r.q.GetLegalEntityByID(ctx, id)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// GetByCode retrieves a legal entity by code.
func (r *LegalEntityRepository) GetByCode(ctx context.Context, code string) (*models.LegalEntity, error) {
	row, err := r.q.GetLegalEntityByCode(ctx, code)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// ListByJurisdiction retrieves legal entities by jurisdiction.
func (r *LegalEntityRepository) ListByJurisdiction(ctx context.Context, jurisdiction string) ([]*models.LegalEntity, error) {
	rows, err := r.q.ListLegalEntitiesByJurisdiction(ctx, jurisdiction)
	if err != nil {
		return nil, err
	}
	return r.toModels(rows), nil
}

// List retrieves all legal entities.
func (r *LegalEntityRepository) List(ctx context.Context) ([]*models.LegalEntity, error) {
	rows, err := r.q.ListLegalEntities(ctx)
	if err != nil {
		return nil, err
	}
	return r.toModels(rows), nil
}

func (r *LegalEntityRepository) toModel(row queries.LegalEntity) *models.LegalEntity {
	le := &models.LegalEntity{
		ID:                  row.ID,
		Code:                row.Code,
		LegalName:           row.LegalName,
		Jurisdiction:        row.Jurisdiction,
		LicenseType:         models.LicenseType(row.LicenseType),
		SupportedCurrencies: row.SupportedCurrencies,
		UpdatedAt:           row.UpdatedAt,
	}

	if row.LicenseNumber.Valid {
		le.LicenseNumber = &row.LicenseNumber.String
	}
	if row.Regulator.Valid {
		le.Regulator = &row.Regulator.String
	}
	if row.FboBankName.Valid {
		le.FBOBankName = &row.FboBankName.String
	}
	if row.FboAccountIban.Valid {
		le.FBOAccountIBAN = &row.FboAccountIban.String
	}
	if row.FboAccountNumber.Valid {
		le.FBOAccountNumber = &row.FboAccountNumber.String
	}
	if row.FboSortCode.Valid {
		le.FBOSortCode = &row.FboSortCode.String
	}
	if row.NostroBankName.Valid {
		le.NostroBankName = &row.NostroBankName.String
	}
	if row.NostroAccountIban.Valid {
		le.NostroAccountIBAN = &row.NostroAccountIban.String
	}
	if row.NostroAccountNumber.Valid {
		le.NostroAccountNumber = &row.NostroAccountNumber.String
	}
	if row.NostroSortCode.Valid {
		le.NostroSortCode = &row.NostroSortCode.String
	}

	// Convert rails
	le.SupportedRails = make([]models.Rail, len(row.SupportedRails))
	for i, r := range row.SupportedRails {
		le.SupportedRails[i] = models.Rail(r)
	}

	return le
}

func (r *LegalEntityRepository) toModels(rows []queries.LegalEntity) []*models.LegalEntity {
	result := make([]*models.LegalEntity, len(rows))
	for i, row := range rows {
		result[i] = r.toModel(row)
	}
	return result
}
