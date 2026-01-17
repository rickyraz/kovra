package ledger

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/google/uuid"
)

// AccountID represents a 128-bit TigerBeetle account ID.
// Structure: [tenant_id: 64 bits][account_type: 8 bits][currency: 24 bits][reserved: 32 bits]
type AccountID [16]byte

// SystemTenantID is used for system-owned accounts (fees, FX settlement, regional settlement).
const SystemTenantID uint64 = 0

// NewAccountID creates a new AccountID from components.
func NewAccountID(tenantID uint64, accountType AccountType, currency Currency) AccountID {
	var id AccountID

	// Bytes 0-7: Tenant ID (big-endian)
	binary.BigEndian.PutUint64(id[0:8], tenantID)

	// Byte 8: Account Type
	id[8] = byte(accountType)

	// Bytes 9-11: Currency (24 bits, big-endian)
	id[9] = byte(currency >> 16)
	id[10] = byte(currency >> 8)
	id[11] = byte(currency)

	// Bytes 12-15: Reserved (zero)
	// Already zero from initialization

	return id
}

// NewAccountIDFromUUID creates an AccountID using a UUID's lower 64 bits as tenant ID.
func NewAccountIDFromUUID(tenantUUID uuid.UUID, accountType AccountType, currency Currency) AccountID {
	// Use the lower 64 bits of the UUID as tenant ID
	tenantID := binary.BigEndian.Uint64(tenantUUID[8:16])
	return NewAccountID(tenantID, accountType, currency)
}

// TenantID returns the tenant ID component.
func (id AccountID) TenantID() uint64 {
	return binary.BigEndian.Uint64(id[0:8])
}

// AccountType returns the account type component.
func (id AccountID) AccountType() AccountType {
	return AccountType(id[8])
}

// Currency returns the currency component.
func (id AccountID) Currency() Currency {
	return Currency(uint32(id[9])<<16 | uint32(id[10])<<8 | uint32(id[11]))
}

// IsSystemAccount returns true if this is a system-owned account.
func (id AccountID) IsSystemAccount() bool {
	return id.TenantID() == SystemTenantID
}

// Bytes returns the raw bytes of the AccountID.
func (id AccountID) Bytes() []byte {
	return id[:]
}

// ToUint128 returns the AccountID as a uint128 represented by two uint64s.
func (id AccountID) ToUint128() (hi, lo uint64) {
	hi = binary.BigEndian.Uint64(id[0:8])
	lo = binary.BigEndian.Uint64(id[8:16])
	return
}

// ToBigInt returns the AccountID as a big.Int for database storage.
func (id AccountID) ToBigInt() *big.Int {
	return new(big.Int).SetBytes(id[:])
}

// FromBigInt creates an AccountID from a big.Int.
func FromBigInt(n *big.Int) AccountID {
	var id AccountID
	bytes := n.Bytes()

	// Pad to 16 bytes (big.Int omits leading zeros)
	if len(bytes) < 16 {
		copy(id[16-len(bytes):], bytes)
	} else {
		copy(id[:], bytes[len(bytes)-16:])
	}

	return id
}

// String returns a human-readable representation of the AccountID.
func (id AccountID) String() string {
	return fmt.Sprintf("%s:%s:%016x",
		id.AccountType().String(),
		id.Currency().String(),
		id.TenantID(),
	)
}

// Hex returns the hexadecimal representation of the AccountID.
func (id AccountID) Hex() string {
	return fmt.Sprintf("%032x", id[:])
}
