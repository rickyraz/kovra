package ledger

// AccountType represents the type of TigerBeetle account.
type AccountType uint8

const (
	// AccountTypeTenantWallet holds client funds per currency (FBO sub-ledger)
	AccountTypeTenantWallet AccountType = 0x01

	// AccountTypeFeeRevenue holds collected platform fees
	AccountTypeFeeRevenue AccountType = 0x02

	// AccountTypeFXSettlement is an internal clearing account for FX conversions
	AccountTypeFXSettlement AccountType = 0x03

	// AccountTypePendingInbound holds funds being collected
	AccountTypePendingInbound AccountType = 0x04

	// AccountTypePendingOutbound holds funds being disbursed
	AccountTypePendingOutbound AccountType = 0x05

	// AccountTypeRegionalSettlement represents Nostro accounts for pre-funded settlement
	AccountTypeRegionalSettlement AccountType = 0x06
)

// String returns a human-readable name for the account type.
func (t AccountType) String() string {
	switch t {
	case AccountTypeTenantWallet:
		return "TENANT_WALLET"
	case AccountTypeFeeRevenue:
		return "FEE_REVENUE"
	case AccountTypeFXSettlement:
		return "FX_SETTLEMENT"
	case AccountTypePendingInbound:
		return "PENDING_INBOUND"
	case AccountTypePendingOutbound:
		return "PENDING_OUTBOUND"
	case AccountTypeRegionalSettlement:
		return "REGIONAL_SETTLEMENT"
	default:
		return "UNKNOWN"
	}
}

// Currency represents ISO 4217 currency codes as ledger IDs.
type Currency uint32

const (
	CurrencyEUR Currency = 978
	CurrencyGBP Currency = 826
	CurrencyIDR Currency = 360
	CurrencySEK Currency = 752
	CurrencyDKK Currency = 208
	CurrencyUSD Currency = 840
)

// String returns the ISO 4217 code for the currency.
func (c Currency) String() string {
	switch c {
	case CurrencyEUR:
		return "EUR"
	case CurrencyGBP:
		return "GBP"
	case CurrencyIDR:
		return "IDR"
	case CurrencySEK:
		return "SEK"
	case CurrencyDKK:
		return "DKK"
	case CurrencyUSD:
		return "USD"
	default:
		return "UNKNOWN"
	}
}

// CurrencyFromString converts a currency code string to Currency.
func CurrencyFromString(s string) Currency {
	switch s {
	case "EUR":
		return CurrencyEUR
	case "GBP":
		return CurrencyGBP
	case "IDR":
		return CurrencyIDR
	case "SEK":
		return CurrencySEK
	case "DKK":
		return CurrencyDKK
	case "USD":
		return CurrencyUSD
	default:
		return 0
	}
}

// TransferFlags represents TigerBeetle transfer flags.
type TransferFlags uint16

const (
	// TransferFlagLinked links this transfer with the next one (all-or-nothing)
	TransferFlagLinked TransferFlags = 1 << 0

	// TransferFlagPending makes this a two-phase transfer (requires void/post)
	TransferFlagPending TransferFlags = 1 << 1

	// TransferFlagPostPending posts (completes) a pending transfer
	TransferFlagPostPending TransferFlags = 1 << 2

	// TransferFlagVoidPending voids (cancels) a pending transfer
	TransferFlagVoidPending TransferFlags = 1 << 3
)

// Balance represents an account balance.
type Balance struct {
	Debits   uint64 // Total debits posted
	Credits  uint64 // Total credits posted
	Pending  uint64 // Pending debits (holds)
	Reserved uint64 // Reserved for future use
}

// Available returns the available balance (credits - debits - pending).
func (b Balance) Available() int64 {
	return int64(b.Credits) - int64(b.Debits) - int64(b.Pending)
}

// Total returns the total balance (credits - debits).
func (b Balance) Total() int64 {
	return int64(b.Credits) - int64(b.Debits)
}
