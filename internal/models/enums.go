package models

// TenantKind represents the type of tenant.
type TenantKind string

const (
	TenantKindPlatform TenantKind = "platform"
	TenantKindSeller   TenantKind = "seller"
	TenantKindDirect   TenantKind = "direct"
)

// TenantStatus represents the lifecycle status of a tenant.
type TenantStatus string

const (
	TenantStatusPendingKYC TenantStatus = "pending_kyc"
	TenantStatusActive     TenantStatus = "active"
	TenantStatusSuspended  TenantStatus = "suspended"
	TenantStatusClosed     TenantStatus = "closed"
)

// TransferStatus represents the state machine status of a transfer.
type TransferStatus string

const (
	TransferStatusCreated    TransferStatus = "created"
	TransferStatusValidating TransferStatus = "validating"
	TransferStatusRejected   TransferStatus = "rejected"
	TransferStatusProcessing TransferStatus = "processing"
	TransferStatusCompleted  TransferStatus = "completed"
	TransferStatusRolledBack TransferStatus = "rolled_back"
	TransferStatusCancelled  TransferStatus = "cancelled"
)

// IsTerminal returns true if the status is a terminal state.
func (s TransferStatus) IsTerminal() bool {
	switch s {
	case TransferStatusCompleted, TransferStatusRolledBack, TransferStatusCancelled, TransferStatusRejected:
		return true
	default:
		return false
	}
}

// Rail represents a payment rail.
type Rail string

const (
	RailSEPAInstant Rail = "SEPA_INSTANT"
	RailSEPASCT     Rail = "SEPA_SCT"
	RailFPS         Rail = "FPS"
	RailCHAPS       Rail = "CHAPS"
	RailBIFast      Rail = "BI_FAST"
	RailRTGS        Rail = "RTGS"
	RailSWIFT       Rail = "SWIFT"
)

// LicenseType represents the type of financial license.
type LicenseType string

const (
	LicenseTypeEMI  LicenseType = "EMI"
	LicenseTypePI   LicenseType = "PI"
	LicenseTypeBank LicenseType = "BANK"
)

// KYCLevel represents the verification level of a tenant.
type KYCLevel string

const (
	KYCLevelBasic    KYCLevel = "basic"
	KYCLevelStandard KYCLevel = "standard"
	KYCLevelEnhanced KYCLevel = "enhanced"
)

// ComplianceRegion represents the compliance jurisdiction.
type ComplianceRegion string

const (
	ComplianceRegionID      ComplianceRegion = "ID"
	ComplianceRegionEU      ComplianceRegion = "EU"
	ComplianceRegionUK      ComplianceRegion = "UK"
	ComplianceRegionUnknown ComplianceRegion = "UNKNOWN"
)

// DeriveComplianceRegion determines the compliance region from currencies.
func DeriveComplianceRegion(fromCurrency, toCurrency string) ComplianceRegion {
	if fromCurrency == "IDR" || toCurrency == "IDR" {
		return ComplianceRegionID
	}
	if fromCurrency == "EUR" || toCurrency == "EUR" ||
		fromCurrency == "SEK" || toCurrency == "SEK" ||
		fromCurrency == "DKK" || toCurrency == "DKK" {
		return ComplianceRegionEU
	}
	if fromCurrency == "GBP" || toCurrency == "GBP" {
		return ComplianceRegionUK
	}
	return ComplianceRegionUnknown
}
