package ledger

import (
	"fmt"

	"github.com/google/uuid"
	tbtypes "github.com/tigerbeetle/tigerbeetle-go/pkg/types"
)

// Transfer is the smallest immutable unit of value movement in the ledger.
//
// It represents a single double-entry accounting operation where value is
// atomically debited from one account and credited to another.
// Transfers must always balance and are executed within a specific ledger
// (currency/book) and business context (Code).
//
// UserData fields are reserved for correlation, idempotency, and tracing.
// They must not affect accounting semantics.
type Transfer struct {
	ID            uuid.UUID
	DebitAccount  AccountID
	CreditAccount AccountID
	Amount        uint64
	Ledger        uint32
	Code          uint16
	Flags         TransferFlags
	UserData128   [16]byte
	UserData64    uint64
	UserData32    uint32
}

// NewTransfer creates a new transfer.
func NewTransfer(debit, credit AccountID, amount uint64, ledger uint32, code uint16) (Transfer, error) {
	id, err := uuid.NewV7()

	if err != nil {
		return Transfer{}, fmt.Errorf("generate transfer id: %w", err)
	}

	return Transfer{
		ID:            id,
		DebitAccount:  debit,
		CreditAccount: credit,
		Amount:        amount,
		Ledger:        ledger,
		Code:          code,
	}, nil
}

// WithID sets a specific transfer ID.
func (t Transfer) WithID(id uuid.UUID) Transfer {
	t.ID = id
	return t
}

// WithFlags sets transfer flags.
func (t Transfer) WithFlags(flags TransferFlags) Transfer {
	t.Flags = flags
	return t
}

// WithUserData sets user data fields for correlation.
func (t Transfer) WithUserData(data128 [16]byte, data64 uint64, data32 uint32) Transfer {
	t.UserData128 = data128
	t.UserData64 = data64
	t.UserData32 = data32
	return t
}

// Linked returns a copy of the transfer with the linked flag set.
func (t Transfer) Linked() Transfer {
	t.Flags |= TransferFlagLinked
	return t
}

// Pending returns a copy of the transfer as a pending (two-phase) transfer.
func (t Transfer) Pending() Transfer {
	t.Flags |= TransferFlagPending
	return t
}

// toTigerBeetle converts the transfer to TigerBeetle format.
func (t Transfer) toTigerBeetle() tbtypes.Transfer {
	// Convert UUID to bytes and then to Uint128
	idBytes := [16]byte(t.ID)

	// Alternative : Langsung slice, tanpa intermediate variable
	// tbtypes.BytesToUint128(t.ID[:])

	// Build transfer flags struct
	flags := tbtypes.TransferFlags{
		Linked:              t.Flags&TransferFlagLinked != 0,
		Pending:             t.Flags&TransferFlagPending != 0,
		PostPendingTransfer: t.Flags&TransferFlagPostPending != 0,
		VoidPendingTransfer: t.Flags&TransferFlagVoidPending != 0,
	}

	return tbtypes.Transfer{
		ID:              tbtypes.BytesToUint128(idBytes),
		DebitAccountID:  tbtypes.BytesToUint128(t.DebitAccount),
		CreditAccountID: tbtypes.BytesToUint128(t.CreditAccount),
		Amount:          tbtypes.ToUint128(t.Amount),
		Ledger:          t.Ledger,
		Code:            t.Code,
		Flags:           flags.ToUint16(),
		UserData128:     tbtypes.BytesToUint128(t.UserData128),
		UserData64:      t.UserData64,
		UserData32:      t.UserData32,
	}
}

// TransferBuilder helps construct linked transfer chains.
type TransferBuilder struct {
	transfers []Transfer
}

// NewTransferBuilder creates a new transfer builder.
func NewTransferBuilder() *TransferBuilder {
	return &TransferBuilder{
		transfers: make([]Transfer, 0),
	}
}

// Add adds a transfer to the chain.
func (b *TransferBuilder) Add(debit, credit AccountID, amount uint64, ledger uint32, code uint16) (*TransferBuilder, error) {
	transfer, err := NewTransfer(debit, credit, amount, ledger, code)
	if err != nil {
		return nil, err
	}
	b.transfers = append(b.transfers, transfer)
	return b, nil
}

// AddTransfer adds a pre-built transfer to the chain.
func (b *TransferBuilder) AddTransfer(t Transfer) *TransferBuilder {
	b.transfers = append(b.transfers, t)
	return b
}

// Build returns the list of transfers.
func (b *TransferBuilder) Build() []Transfer {
	return b.transfers
}

// BuildLinked returns the list of transfers with linked flags set.
// All transfers except the last one will have the linked flag.
func (b *TransferBuilder) BuildLinked() []Transfer {
	if len(b.transfers) == 0 {
		return nil
	}

	result := make([]Transfer, len(b.transfers))
	copy(result, b.transfers)

	// Set linked flag on all but the last transfer
	for i := range result[:len(result)-1] {
		result[i].Flags |= TransferFlagLinked
	}

	return result
}

// FXTransferPair represents a pair of transfer chains for cross-currency FX operations.
// TigerBeetle requires all accounts in a transfer to be in the same ledger,
// so FX conversions must be split into two separate chains coordinated at application level.
type FXTransferPair struct {
	// SourceChain debits source currency (e.g., EUR)
	// Executes in source currency ledger
	SourceChain []Transfer

	// DestinationChain credits destination currency (e.g., IDR)
	// Executes in destination currency ledger
	DestinationChain []Transfer

	// CorrelationID links both chains for reconciliation (stored in UserData128)
	CorrelationID [16]byte
}

// FXTransferChains creates two separate transfer chains for cross-currency FX operations.
//
// TigerBeetle Constraint: Accounts in a transfer MUST be in the same ledger.
// Therefore, cross-currency FX requires two coordinated chains:
//
// Source Chain (srcCurrency ledger):
//  1. TENANT_WALLET → PENDING_OUTBOUND (hold source funds)
//  2. PENDING_OUTBOUND → FX_POSITION (platform acquires source currency)
//
// Destination Chain (dstCurrency ledger):
//  1. FX_POSITION → FEE_REVENUE (fee deduction, if any)
//  2. FX_POSITION → REGIONAL_SETTLEMENT (payout to destination)
//
// The FX_POSITION accounts track the platform's currency exposure:
//   - Credit to FX_POSITION_SRC = platform receives source currency
//   - Debit from FX_POSITION_DST = platform pays out destination currency
//
// Application-level coordination required:
//   - Both chains must succeed, or both must be compensated
//   - Use CorrelationID to link chains in PostgreSQL for reconciliation
//   - FX rate and conversion details are stored in PostgreSQL, not TigerBeetle
func FXTransferChains(
	tenantID uint64,
	srcCurrency Currency,
	dstCurrency Currency,
	srcAmount uint64,
	dstAmount uint64,
	feeAmount uint64,
	transferCode uint16,
) (FXTransferPair, error) {
	// Generate correlation ID to link both chains
	correlationUUID, err := uuid.NewV7()
	if err != nil {
		return FXTransferPair{}, fmt.Errorf("generate correlation id: %w", err)
	}
	correlationID := [16]byte(correlationUUID)

	// === SOURCE CHAIN (srcCurrency ledger) ===
	srcBuilder := NewTransferBuilder()

	srcWallet := NewAccountID(tenantID, AccountTypeTenantWallet, srcCurrency)
	srcPendingOut := NewAccountID(SystemTenantID, AccountTypePendingOutbound, srcCurrency)
	srcFXPosition := NewAccountID(SystemTenantID, AccountTypeFXSettlement, srcCurrency)
	srcLedger := uint32(srcCurrency)

	// Step 1: Debit tenant wallet → pending outbound (hold)
	if _, err := srcBuilder.Add(srcWallet, srcPendingOut, srcAmount, srcLedger, transferCode); err != nil {
		return FXTransferPair{}, fmt.Errorf("add source step 1: %w", err)
	}

	// Step 2: Pending outbound → FX position (platform acquires source currency)
	if _, err := srcBuilder.Add(srcPendingOut, srcFXPosition, srcAmount, srcLedger, transferCode); err != nil {
		return FXTransferPair{}, fmt.Errorf("add source step 2: %w", err)
	}

	// Set correlation ID on source chain
	srcChain := srcBuilder.BuildLinked()
	for i := range srcChain {
		srcChain[i].UserData128 = correlationID
	}

	// === DESTINATION CHAIN (dstCurrency ledger) ===
	dstBuilder := NewTransferBuilder()

	dstFXPosition := NewAccountID(SystemTenantID, AccountTypeFXSettlement, dstCurrency)
	dstFeeRevenue := NewAccountID(SystemTenantID, AccountTypeFeeRevenue, dstCurrency)
	dstRegionalSettlement := NewAccountID(SystemTenantID, AccountTypeRegionalSettlement, dstCurrency)
	dstLedger := uint32(dstCurrency)

	// Step 1: FX position → fee revenue (fee deduction)
	if feeAmount > 0 {
		if _, err := dstBuilder.Add(dstFXPosition, dstFeeRevenue, feeAmount, dstLedger, transferCode); err != nil {
			return FXTransferPair{}, fmt.Errorf("add dest fee step: %w", err)
		}
	}

	// Step 2: FX position → regional settlement (payout)
	finalAmount := dstAmount - feeAmount
	if _, err := dstBuilder.Add(dstFXPosition, dstRegionalSettlement, finalAmount, dstLedger, transferCode); err != nil {
		return FXTransferPair{}, fmt.Errorf("add dest payout step: %w", err)
	}

	// Set correlation ID on destination chain
	dstChain := dstBuilder.BuildLinked()
	for i := range dstChain {
		dstChain[i].UserData128 = correlationID
	}

	return FXTransferPair{
		SourceChain:      srcChain,
		DestinationChain: dstChain,
		CorrelationID:    correlationID,
	}, nil
}

// SimpleTransferChain creates a simple same-currency transfer chain.
// All transfers execute atomically in the same ledger via TigerBeetle linked transfers.
//
// Steps:
//  1. TENANT_WALLET → PENDING_OUTBOUND (debit source wallet)
//  2. PENDING_OUTBOUND → FEE_REVENUE (fee deduction, if feeAmount > 0)
//  3. PENDING_OUTBOUND → REGIONAL_SETTLEMENT (final payout)
func SimpleTransferChain(
	tenantID uint64,
	currency Currency,
	amount uint64,
	feeAmount uint64,
	transferCode uint16,
) ([]Transfer, error) {
	builder := NewTransferBuilder()

	wallet := NewAccountID(tenantID, AccountTypeTenantWallet, currency)
	pendingOut := NewAccountID(SystemTenantID, AccountTypePendingOutbound, currency)
	feeRevenue := NewAccountID(SystemTenantID, AccountTypeFeeRevenue, currency)
	regionalSettlement := NewAccountID(SystemTenantID, AccountTypeRegionalSettlement, currency)

	ledger := uint32(currency)

	// Step 1: Debit tenant wallet → pending outbound
	if _, err := builder.Add(wallet, pendingOut, amount, ledger, transferCode); err != nil {
		return nil, fmt.Errorf("add debit step: %w", err)
	}

	// Step 2: Fee deduction (if any)
	if feeAmount > 0 {
		if _, err := builder.Add(pendingOut, feeRevenue, feeAmount, ledger, transferCode); err != nil {
			return nil, fmt.Errorf("add fee step: %w", err)
		}
	}

	// Step 3: Final settlement
	finalAmount := amount - feeAmount
	if _, err := builder.Add(pendingOut, regionalSettlement, finalAmount, ledger, transferCode); err != nil {
		return nil, fmt.Errorf("add settlement step: %w", err)
	}

	return builder.BuildLinked(), nil
}
