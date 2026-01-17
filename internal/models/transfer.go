package models

import (
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Transfer represents a payment transfer.
type Transfer struct {
	ID                   uuid.UUID
	TenantID             uuid.UUID
	SourceLegalEntityID  *uuid.UUID
	DestLegalEntityID    *uuid.UUID
	QuoteID              *uuid.UUID
	BatchID              *uuid.UUID
	RecipientID          *uuid.UUID
	IdempotencyKey       *string
	FromCurrency         string
	ToCurrency           string
	FromAmount           decimal.Decimal
	ToAmount             decimal.Decimal
	FXRate               decimal.Decimal
	TotalFee             decimal.Decimal
	Status               TransferStatus
	FailureReason        *string
	Rail                 *Rail
	RailReference        *string
	NettingGroupID       *uuid.UUID
	IsNetted             bool
	TBTransferIDs        []*big.Int
	RiskScore            *int
	ComplianceStatus     string
	ScreenedAt           *time.Time
	ComplianceRegion     ComplianceRegion
	UpdatedAt            time.Time
	CompletedAt          *time.Time
}

// IsFXTransfer returns true if this is a cross-currency transfer.
func (t *Transfer) IsFXTransfer() bool {
	return t.FromCurrency != t.ToCurrency
}

// IsComplete returns true if the transfer has completed.
func (t *Transfer) IsComplete() bool {
	return t.Status == TransferStatusCompleted
}

// IsFailed returns true if the transfer has failed.
func (t *Transfer) IsFailed() bool {
	return t.Status == TransferStatusRejected ||
		t.Status == TransferStatusRolledBack ||
		t.Status == TransferStatusCancelled
}

// CreateTransferParams contains parameters for creating a new transfer.
type CreateTransferParams struct {
	TenantID             uuid.UUID
	SourceLegalEntityID  *uuid.UUID
	DestLegalEntityID    *uuid.UUID
	QuoteID              *uuid.UUID
	RecipientID          *uuid.UUID
	IdempotencyKey       *string
	FromCurrency         string
	ToCurrency           string
	FromAmount           decimal.Decimal
	ToAmount             decimal.Decimal
	FXRate               decimal.Decimal
	TotalFee             decimal.Decimal
	Rail                 *Rail
}

// TransferFilter contains filter parameters for querying transfers.
type TransferFilter struct {
	Status           *TransferStatus
	FromCurrency     *string
	ToCurrency       *string
	ComplianceRegion *ComplianceRegion
	UpdatedAfter     *time.Time
	UpdatedBefore    *time.Time
	Limit            int
	Offset           int
}
