package repository

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"kovra/internal/models"
	"kovra/internal/repository/queries"
)

// TenantRepository handles tenant data access.
type TenantRepository struct {
	q *queries.Queries
}

// NewTenantRepository creates a new tenant repository.
func NewTenantRepository(pool *pgxpool.Pool) *TenantRepository {
	return &TenantRepository{q: queries.New(pool)}
}

// Create creates a new tenant.
func (r *TenantRepository) Create(ctx context.Context, params models.CreateTenantParams) (*models.Tenant, error) {
	metadata := params.Metadata
	if metadata == nil {
		metadata = []byte("{}")
	}

	row, err := r.q.CreateTenant(ctx, queries.CreateTenantParams{
		DisplayName:    params.DisplayName,
		LegalName:      params.LegalName,
		Country:        params.Country,
		TenantKind:     queries.TenantKindEnum(params.TenantKind),
		ParentTenantID: uuidToNullable(params.ParentTenantID),
		LegalEntityID:  params.LegalEntityID,
		Metadata:       metadata,
	})
	if err != nil {
		return nil, err
	}

	return r.toModel(row), nil
}

// GetByID retrieves a tenant by ID.
func (r *TenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	row, err := r.q.GetTenantByID(ctx, id)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// Update updates a tenant.
func (r *TenantRepository) Update(ctx context.Context, id uuid.UUID, params models.UpdateTenantParams) (*models.Tenant, error) {
	row, err := r.q.UpdateTenant(ctx, queries.UpdateTenantParams{
		ID:                   id,
		DisplayName:          pgtype.Text{String: ptrToString(params.DisplayName), Valid: params.DisplayName != nil},
		LegalName:            pgtype.Text{String: ptrToString(params.LegalName), Valid: params.LegalName != nil},
		TenantStatus:         toNullTenantStatus(params.TenantStatus),
		KycLevel:             toNullKYCLevel(params.KYCLevel),
		NettingEnabled:       pgtype.Bool{Bool: ptrToBool(params.NettingEnabled), Valid: params.NettingEnabled != nil},
		NettingWindowMinutes: pgtype.Int4{Int32: int32(ptrToInt(params.NettingWindowMinutes)), Valid: params.NettingWindowMinutes != nil},
		WebhookUrl:           pgtype.Text{String: ptrToString(params.WebhookURL), Valid: params.WebhookURL != nil},
		Metadata:             params.Metadata,
	})
	if err != nil {
		return nil, err
	}

	return r.toModel(row), nil
}

// ListByLegalEntity retrieves tenants by legal entity.
func (r *TenantRepository) ListByLegalEntity(ctx context.Context, legalEntityID uuid.UUID) ([]*models.Tenant, error) {
	rows, err := r.q.ListTenantsByLegalEntity(ctx, legalEntityID)
	if err != nil {
		return nil, err
	}
	return r.toModels(rows), nil
}

// ListByParent retrieves child tenants.
func (r *TenantRepository) ListByParent(ctx context.Context, parentID uuid.UUID) ([]*models.Tenant, error) {
	rows, err := r.q.ListTenantsByParent(ctx, pgtype.UUID{Bytes: parentID, Valid: true})
	if err != nil {
		return nil, err
	}
	return r.toModels(rows), nil
}

// ListActive retrieves active tenants.
func (r *TenantRepository) ListActive(ctx context.Context, limit, offset int) ([]*models.Tenant, error) {
	rows, err := r.q.ListActiveTenants(ctx, queries.ListActiveTenantsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return r.toModels(rows), nil
}

func (r *TenantRepository) toModel(row queries.Tenant) *models.Tenant {
	t := &models.Tenant{
		ID:                   row.ID,
		DisplayName:          row.DisplayName,
		LegalName:            row.LegalName,
		Country:              row.Country,
		TenantKind:           models.TenantKind(row.TenantKind),
		LegalEntityID:        row.LegalEntityID,
		TenantStatus:         models.TenantStatus(row.TenantStatus),
		KYCLevel:             models.KYCLevel(row.KycLevel),
		NettingEnabled:       row.NettingEnabled,
		NettingWindowMinutes: int(row.NettingWindowMinutes),
		Metadata:             json.RawMessage(row.Metadata),
		UpdatedAt:            row.UpdatedAt,
	}

	if row.ParentTenantID.Valid {
		id := uuid.UUID(row.ParentTenantID.Bytes)
		t.ParentTenantID = &id
	}
	if row.ApiKeyHash.Valid {
		t.APIKeyHash = &row.ApiKeyHash.String
	}
	if row.WebhookUrl.Valid {
		t.WebhookURL = &row.WebhookUrl.String
	}
	if row.WebhookSecretHash.Valid {
		t.WebhookSecretHash = &row.WebhookSecretHash.String
	}

	return t
}

func (r *TenantRepository) toModels(rows []queries.Tenant) []*models.Tenant {
	result := make([]*models.Tenant, len(rows))
	for i, row := range rows {
		result[i] = r.toModel(row)
	}
	return result
}

// Helper functions
func uuidToNullable(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

func ptrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ptrToBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func ptrToInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func toNullTenantStatus(s *models.TenantStatus) queries.NullTenantStatusEnum {
	if s == nil {
		return queries.NullTenantStatusEnum{}
	}
	return queries.NullTenantStatusEnum{TenantStatusEnum: queries.TenantStatusEnum(*s), Valid: true}
}

func toNullKYCLevel(k *models.KYCLevel) queries.NullKycLevelEnum {
	if k == nil {
		return queries.NullKycLevelEnum{}
	}
	return queries.NullKycLevelEnum{KycLevelEnum: queries.KycLevelEnum(*k), Valid: true}
}
