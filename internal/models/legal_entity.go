package models

import (
	"time"

	"github.com/google/uuid"
)

// LegalEntity represents a licensed Kovra entity in a specific jurisdiction.
type LegalEntity struct {
	ID                   uuid.UUID
	Code                 string
	LegalName            string
	Jurisdiction         string
	LicenseType          LicenseType
	LicenseNumber        *string
	Regulator            *string
	FBOBankName          *string
	FBOAccountIBAN       *string
	FBOAccountNumber     *string
	FBOSortCode          *string
	NostroBankName       *string
	NostroAccountIBAN    *string
	NostroAccountNumber  *string
	NostroSortCode       *string
	SupportedCurrencies  []string
	SupportedRails       []Rail
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// SupportsСurrency checks if the legal entity supports a currency.
func (le *LegalEntity) SupportsCurrency(currency string) bool {
	for _, c := range le.SupportedCurrencies {
		if c == currency {
			return true
		}
	}
	return false
}

// SupportsRail checks if the legal entity supports a payment rail.
func (le *LegalEntity) SupportsRail(rail Rail) bool {
	for _, r := range le.SupportedRails {
		if r == rail {
			return true
		}
	}
	return false
}
