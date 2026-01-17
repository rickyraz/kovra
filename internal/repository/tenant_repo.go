package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"kovra/internal/models"
)

// TenantRepository handles tenant data access.
type TenantRepository struct {
	pool *pgxpool.Pool
}

// NewTenantRepository creates a new tenant repository.
func NewTenantRepository(pool *pgxpool.Pool) *TenantRepository {
	return &TenantRepository{pool: pool}
}

// Create creates a new tenant.
func (r *TenantRepository) Create(ctx context.Context, params models.CreateTenantParams) (*models.Tenant, error) {
	query := `
		INSERT INTO tenants (display_name, legal_name, country, tenant_kind, parent_tenant_id, legal_entity_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, display_name, legal_name, country, tenant_kind, parent_tenant_id, legal_entity_id,
			tenant_status, kyc_level, netting_enabled, netting_window_minutes,
			api_key_hash, webhook_url, webhook_secret_hash, metadata, created_at, updated_at`

	metadata := params.Metadata
	if metadata == nil {
		metadata = []byte("{}")
	}

	row := r.pool.QueryRow(ctx, query,
		params.DisplayName,
		params.LegalName,
		params.Country,
		params.TenantKind,
		params.ParentTenantID,
		params.LegalEntityID,
		metadata,
	)

	return r.scan(row)
}

// GetByID retrieves a tenant by ID.
func (r *TenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	query := `
		SELECT id, display_name, legal_name, country, tenant_kind, parent_tenant_id, legal_entity_id,
			tenant_status, kyc_level, netting_enabled, netting_window_minutes,
			api_key_hash, webhook_url, webhook_secret_hash, metadata, created_at, updated_at
		FROM tenants
		WHERE id = $1`

	row := r.pool.QueryRow(ctx, query, id)
	tenant, err := r.scan(row)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return tenant, err
}

// Update updates a tenant.
func (r *TenantRepository) Update(ctx context.Context, id uuid.UUID, params models.UpdateTenantParams) (*models.Tenant, error) {
	query := `
		UPDATE tenants SET
			display_name = COALESCE($2, display_name),
			legal_name = COALESCE($3, legal_name),
			tenant_status = COALESCE($4, tenant_status),
			kyc_level = COALESCE($5, kyc_level),
			netting_enabled = COALESCE($6, netting_enabled),
			netting_window_minutes = COALESCE($7, netting_window_minutes),
			webhook_url = COALESCE($8, webhook_url),
			metadata = COALESCE($9, metadata),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, display_name, legal_name, country, tenant_kind, parent_tenant_id, legal_entity_id,
			tenant_status, kyc_level, netting_enabled, netting_window_minutes,
			api_key_hash, webhook_url, webhook_secret_hash, metadata, created_at, updated_at`

	row := r.pool.QueryRow(ctx, query,
		id,
		params.DisplayName,
		params.LegalName,
		params.TenantStatus,
		params.KYCLevel,
		params.NettingEnabled,
		params.NettingWindowMinutes,
		params.WebhookURL,
		params.Metadata,
	)

	return r.scan(row)
}

// ListByLegalEntity retrieves tenants by legal entity.
func (r *TenantRepository) ListByLegalEntity(ctx context.Context, legalEntityID uuid.UUID) ([]*models.Tenant, error) {
	query := `
		SELECT id, display_name, legal_name, country, tenant_kind, parent_tenant_id, legal_entity_id,
			tenant_status, kyc_level, netting_enabled, netting_window_minutes,
			api_key_hash, webhook_url, webhook_secret_hash, metadata, created_at, updated_at
		FROM tenants
		WHERE legal_entity_id = $1
		ORDER BY created_at DESC`

	return r.scanMany(ctx, query, legalEntityID)
}

// ListByParent retrieves child tenants.
func (r *TenantRepository) ListByParent(ctx context.Context, parentID uuid.UUID) ([]*models.Tenant, error) {
	query := `
		SELECT id, display_name, legal_name, country, tenant_kind, parent_tenant_id, legal_entity_id,
			tenant_status, kyc_level, netting_enabled, netting_window_minutes,
			api_key_hash, webhook_url, webhook_secret_hash, metadata, created_at, updated_at
		FROM tenants
		WHERE parent_tenant_id = $1
		ORDER BY created_at DESC`

	return r.scanMany(ctx, query, parentID)
}

// ListActive retrieves active tenants.
func (r *TenantRepository) ListActive(ctx context.Context, limit, offset int) ([]*models.Tenant, error) {
	query := `
		SELECT id, display_name, legal_name, country, tenant_kind, parent_tenant_id, legal_entity_id,
			tenant_status, kyc_level, netting_enabled, netting_window_minutes,
			api_key_hash, webhook_url, webhook_secret_hash, metadata, created_at, updated_at
		FROM tenants
		WHERE tenant_status = 'active'
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	return r.scanMany(ctx, query, limit, offset)
}

func (r *TenantRepository) scanMany(ctx context.Context, query string, args ...any) ([]*models.Tenant, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*models.Tenant
	for rows.Next() {
		t, err := r.scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan tenant: %w", err)
		}
		tenants = append(tenants, t)
	}

	return tenants, rows.Err()
}

func (r *TenantRepository) scan(s scanner) (*models.Tenant, error) {
	var t models.Tenant

	err := s.Scan(
		&t.ID,
		&t.DisplayName,
		&t.LegalName,
		&t.Country,
		&t.TenantKind,
		&t.ParentTenantID,
		&t.LegalEntityID,
		&t.TenantStatus,
		&t.KYCLevel,
		&t.NettingEnabled,
		&t.NettingWindowMinutes,
		&t.APIKeyHash,
		&t.WebhookURL,
		&t.WebhookSecretHash,
		&t.Metadata,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &t, nil
}
