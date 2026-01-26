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

// id := NewAccountID(
//     1,        // Tenant ID = 1
//     2,        // Account Type = 2 (Savings)
//     840,      // Currency = 840 (USD)
// )

// NewAccountID creates a new AccountID from components.
func NewAccountID(tenantID uint64, accountType AccountType, currency Currency) AccountID {
	var id AccountID

	// Bytes 0-7: Tenant ID (big-endian)
	// Siapa yang punya akun ini? (Perusahaan/organisasi mana)
	// Contoh: Tenant 1 = Bank A, Tenant 2 = Bank B
	// uint64 = bisa nyimpan angka sampai 18 kuadriliun (9,223,372,036,854,775,807)
	// Jadi, bisa nyimpen sampai 18 kuadriliun tenant yang berbeda

	// BigEndian: cara nyimpan angka besar di memory (urutan normal) - // SIMPAN (Put)
	binary.BigEndian.PutUint64(id[0:8], tenantID)

	// Byte 8: Account Type
	// Jenis akun apa? Contoh:
	// 1 = Checking Account (Tabungan)
	// 2 = Savings Account (Giro)
	// 3 = Credit Account (Pinjaman)
	// byte = bisa nyimpan 0-255 (satu tipe cukup satu byte)
	id[8] = byte(accountType)

	// Bytes 9-11: Currency (24 bits, big-endian)
	id[9] = byte(currency >> 16)
	id[10] = byte(currency >> 8)
	id[11] = byte(currency)

	// 	- **Mata uang** apa? Contoh:
	//   -> `840` = USD
	//   -> `360` = IDR
	//   -> `156` = CNY
	//  - **3 byte = 24 bit** bisa nyimpan sampai **16 juta** kode mata uang (lebih dari cukup)

	// **Mengapa pakai `>> 16`, `>> 8`?** Itu **bit shifting** — caranya "potong" angka besar jadi piece kecil:
	// ```
	// Currency = 840 (biner: 000000000000001101001000)

	// >> 16 : potong 16 bit → 0        (byte 9)
	// >> 8  : potong 8 bit  → 3        (byte 10)
	// >> 0  : sisa          → 72       (byte 11)

	// Bytes 12-15: Reserved (zero)
	// Already zero from initialization

	return id
}

// NewAccountIDFromUUID creates an AccountID using a UUID's lower 64 bits as tenant ID.
func NewAccountIDFromUUID(tenantUUID uuid.UUID, accountType AccountType, currency Currency) AccountID {
	// UUID = Universally Unique Identifier - Sebuah angka 128-bit (16 byte) yang unik di dunia, dipakai sebagai ID yang tidak akan bentrok.
	// Byte 0-7     | Byte 8-15
	// (8 byte)     | (8 byte)
	// ─────────────┼──────────
	// Upper 64 bit | Lower 64 bit

	// Use the lower 64 bits of the UUID as tenant ID
	tenantID := binary.BigEndian.Uint64(tenantUUID[8:16])
	return NewAccountID(tenantID, accountType, currency)
}

// TenantID returns the tenant ID component.
func (id AccountID) TenantID() uint64 {
	// Ambil byte 0-7 dari AccountID
	// Convert ke uint64 menggunakan BigEndian
	// Return hasilnya  - // BACA (Get)
	return binary.BigEndian.Uint64(id[0:8])
}

// | Operasi | Arah | Fungsi |
// |---------|------|--------|
// | **PutUint64** | uint64 → byte array | **Simpan** angka jadi 8 byte |
// | **Uint64** | byte array → uint64 | **Baca** 8 byte jadi angka |

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
