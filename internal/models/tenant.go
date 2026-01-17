package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Tenant represents a B2B client.
type Tenant struct {
	ID                   uuid.UUID
	DisplayName          string
	LegalName            string
	Country              string
	TenantKind           TenantKind
	ParentTenantID       *uuid.UUID
	LegalEntityID        uuid.UUID
	TenantStatus         TenantStatus
	KYCLevel             KYCLevel
	NettingEnabled       bool
	NettingWindowMinutes int
	APIKeyHash           *string
	WebhookURL           *string
	WebhookSecretHash    *string
	Metadata             json.RawMessage
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// IsActive returns true if the tenant is active.
func (t *Tenant) IsActive() bool {
	return t.TenantStatus == TenantStatusActive
}

// CanTransact returns true if the tenant can perform transactions.
func (t *Tenant) CanTransact() bool {
	return t.TenantStatus == TenantStatusActive && t.KYCLevel != KYCLevelBasic
}

// CreateTenantParams contains parameters for creating a new tenant.
type CreateTenantParams struct {
	DisplayName    string
	LegalName      string
	Country        string
	TenantKind     TenantKind
	ParentTenantID *uuid.UUID
	LegalEntityID  uuid.UUID
	Metadata       json.RawMessage
}

// UpdateTenantParams contains parameters for updating a tenant.
type UpdateTenantParams struct {
	DisplayName          *string
	LegalName            *string
	TenantStatus         *TenantStatus
	KYCLevel             *KYCLevel
	NettingEnabled       *bool
	NettingWindowMinutes *int
	WebhookURL           *string
	Metadata             json.RawMessage
}
