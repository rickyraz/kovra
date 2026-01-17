package ledger

import (
	"github.com/google/uuid"
	tbtypes "github.com/tigerbeetle/tigerbeetle-go/pkg/types"
)

// Transfer represents a TigerBeetle transfer.
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
func NewTransfer(debit, credit AccountID, amount uint64, ledger uint32, code uint16) Transfer {
	return Transfer{
		ID:            uuid.New(),
		DebitAccount:  debit,
		CreditAccount: credit,
		Amount:        amount,
		Ledger:        ledger,
		Code:          code,
	}
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
func (b *TransferBuilder) Add(debit, credit AccountID, amount uint64, ledger uint32, code uint16) *TransferBuilder {
	b.transfers = append(b.transfers, NewTransfer(debit, credit, amount, ledger, code))
	return b
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

// FXTransferChain creates a 5-step FX transfer chain for cross-currency transfers.
// This is the atomic EUR → IDR (or similar) conversion flow.
//
// Steps:
// 1. TENANT_WALLET_SRC → PENDING_OUTBOUND_SRC (debit source wallet)
// 2. PENDING_OUTBOUND_SRC → FX_SETTLEMENT_SRC (move to FX clearing)
// 3. FX_SETTLEMENT_SRC → FX_SETTLEMENT_DST (FX conversion)
// 4. FX_SETTLEMENT_DST → FEE_REVENUE_DST (fee deduction)
// 5. FX_SETTLEMENT_DST → REGIONAL_SETTLEMENT_DST (final settlement)
func FXTransferChain(
	tenantID uint64,
	srcCurrency Currency,
	dstCurrency Currency,
	srcAmount uint64,
	dstAmount uint64,
	feeAmount uint64,
	transferCode uint16,
) []Transfer {
	builder := NewTransferBuilder()

	// Source currency accounts
	srcWallet := NewAccountID(tenantID, AccountTypeTenantWallet, srcCurrency)
	srcPendingOut := NewAccountID(SystemTenantID, AccountTypePendingOutbound, srcCurrency)
	srcFXSettlement := NewAccountID(SystemTenantID, AccountTypeFXSettlement, srcCurrency)

	// Destination currency accounts
	dstFXSettlement := NewAccountID(SystemTenantID, AccountTypeFXSettlement, dstCurrency)
	dstFeeRevenue := NewAccountID(SystemTenantID, AccountTypeFeeRevenue, dstCurrency)
	dstRegionalSettlement := NewAccountID(SystemTenantID, AccountTypeRegionalSettlement, dstCurrency)

	srcLedger := uint32(srcCurrency)
	dstLedger := uint32(dstCurrency)

	// Step 1: Debit tenant wallet → pending outbound
	builder.Add(srcWallet, srcPendingOut, srcAmount, srcLedger, transferCode)

	// Step 2: Pending outbound → FX settlement (source currency)
	builder.Add(srcPendingOut, srcFXSettlement, srcAmount, srcLedger, transferCode)

	// Step 3: FX settlement source → FX settlement dest (cross-currency)
	// Note: This requires accounts in different ledgers, handled specially
	builder.Add(srcFXSettlement, dstFXSettlement, dstAmount, dstLedger, transferCode)

	// Step 4: FX settlement → fee revenue (fee deduction)
	if feeAmount > 0 {
		builder.Add(dstFXSettlement, dstFeeRevenue, feeAmount, dstLedger, transferCode)
	}

	// Step 5: FX settlement → regional settlement (final payout)
	finalAmount := dstAmount - feeAmount
	builder.Add(dstFXSettlement, dstRegionalSettlement, finalAmount, dstLedger, transferCode)

	return builder.BuildLinked()
}

// SimpleTransferChain creates a simple same-currency transfer chain.
// Steps:
// 1. TENANT_WALLET → PENDING_OUTBOUND (debit source)
// 2. PENDING_OUTBOUND → REGIONAL_SETTLEMENT (settlement)
func SimpleTransferChain(
	tenantID uint64,
	currency Currency,
	amount uint64,
	feeAmount uint64,
	transferCode uint16,
) []Transfer {
	builder := NewTransferBuilder()

	wallet := NewAccountID(tenantID, AccountTypeTenantWallet, currency)
	pendingOut := NewAccountID(SystemTenantID, AccountTypePendingOutbound, currency)
	feeRevenue := NewAccountID(SystemTenantID, AccountTypeFeeRevenue, currency)
	regionalSettlement := NewAccountID(SystemTenantID, AccountTypeRegionalSettlement, currency)

	ledger := uint32(currency)

	// Step 1: Debit tenant wallet
	builder.Add(wallet, pendingOut, amount, ledger, transferCode)

	// Step 2: Fee deduction (if any)
	if feeAmount > 0 {
		builder.Add(pendingOut, feeRevenue, feeAmount, ledger, transferCode)
	}

	// Step 3: Final settlement
	finalAmount := amount - feeAmount
	builder.Add(pendingOut, regionalSettlement, finalAmount, ledger, transferCode)

	return builder.BuildLinked()
}
