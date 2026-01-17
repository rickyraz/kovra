package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"kovra/internal/models"
)

// LegalEntityRepository handles legal entity data access.
type LegalEntityRepository struct {
	pool *pgxpool.Pool
}

// NewLegalEntityRepository creates a new legal entity repository.
func NewLegalEntityRepository(pool *pgxpool.Pool) *LegalEntityRepository {
	return &LegalEntityRepository{pool: pool}
}

// GetByID retrieves a legal entity by ID.
func (r *LegalEntityRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.LegalEntity, error) {
	query := `
		SELECT id, code, legal_name, jurisdiction, license_type, license_number, regulator,
			fbo_bank_name, fbo_account_iban, fbo_account_number, fbo_sort_code,
			nostro_bank_name, nostro_account_iban, nostro_account_number, nostro_sort_code,
			supported_currencies, supported_rails, created_at, updated_at
		FROM legal_entities
		WHERE id = $1`

	return r.scanOne(ctx, query, id)
}

// GetByCode retrieves a legal entity by code.
func (r *LegalEntityRepository) GetByCode(ctx context.Context, code string) (*models.LegalEntity, error) {
	query := `
		SELECT id, code, legal_name, jurisdiction, license_type, license_number, regulator,
			fbo_bank_name, fbo_account_iban, fbo_account_number, fbo_sort_code,
			nostro_bank_name, nostro_account_iban, nostro_account_number, nostro_sort_code,
			supported_currencies, supported_rails, created_at, updated_at
		FROM legal_entities
		WHERE code = $1`

	return r.scanOne(ctx, query, code)
}

// ListByJurisdiction retrieves legal entities by jurisdiction.
func (r *LegalEntityRepository) ListByJurisdiction(ctx context.Context, jurisdiction string) ([]*models.LegalEntity, error) {
	query := `
		SELECT id, code, legal_name, jurisdiction, license_type, license_number, regulator,
			fbo_bank_name, fbo_account_iban, fbo_account_number, fbo_sort_code,
			nostro_bank_name, nostro_account_iban, nostro_account_number, nostro_sort_code,
			supported_currencies, supported_rails, created_at, updated_at
		FROM legal_entities
		WHERE jurisdiction = $1
		ORDER BY code`

	return r.scanMany(ctx, query, jurisdiction)
}

// List retrieves all legal entities.
func (r *LegalEntityRepository) List(ctx context.Context) ([]*models.LegalEntity, error) {
	query := `
		SELECT id, code, legal_name, jurisdiction, license_type, license_number, regulator,
			fbo_bank_name, fbo_account_iban, fbo_account_number, fbo_sort_code,
			nostro_bank_name, nostro_account_iban, nostro_account_number, nostro_sort_code,
			supported_currencies, supported_rails, created_at, updated_at
		FROM legal_entities
		ORDER BY code`

	return r.scanMany(ctx, query)
}

func (r *LegalEntityRepository) scanOne(ctx context.Context, query string, args ...any) (*models.LegalEntity, error) {
	row := r.pool.QueryRow(ctx, query, args...)
	le, err := r.scan(row)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan legal entity: %w", err)
	}
	return le, nil
}

func (r *LegalEntityRepository) scanMany(ctx context.Context, query string, args ...any) ([]*models.LegalEntity, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query legal entities: %w", err)
	}
	defer rows.Close()

	var entities []*models.LegalEntity
	for rows.Next() {
		le, err := r.scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan legal entity: %w", err)
		}
		entities = append(entities, le)
	}

	return entities, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func (r *LegalEntityRepository) scan(s scanner) (*models.LegalEntity, error) {
	var le models.LegalEntity
	var rails []string

	err := s.Scan(
		&le.ID,
		&le.Code,
		&le.LegalName,
		&le.Jurisdiction,
		&le.LicenseType,
		&le.LicenseNumber,
		&le.Regulator,
		&le.FBOBankName,
		&le.FBOAccountIBAN,
		&le.FBOAccountNumber,
		&le.FBOSortCode,
		&le.NostroBankName,
		&le.NostroAccountIBAN,
		&le.NostroAccountNumber,
		&le.NostroSortCode,
		&le.SupportedCurrencies,
		&rails,
		&le.CreatedAt,
		&le.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Convert string rails to Rail type
	le.SupportedRails = make([]models.Rail, len(rails))
	for i, r := range rails {
		le.SupportedRails[i] = models.Rail(r)
	}

	return &le, nil
}
