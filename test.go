package main

import (
	"fmt"
	"log"
	"math/big"

	tigerbeetle "github.com/tigerbeetle/tigerbeetle-go"
	"github.com/tigerbeetle/tigerbeetle-go/pkg/types"
)

func main() {
	client, err := tigerbeetle.NewClient(types.Uint128{}, []string{"127.0.0.1:3000"})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	log.Println("✓ Connected to TigerBeetle")

	// Buat akun Bank (ID=0)
	bankAccount := []types.Account{
		{
			ID:     toUint128(100), // Changed from 0
			Ledger: 1,
			Code:   1,
			Flags:  types.AccountFlags{CreditsMustNotExceedDebits: true}.ToUint16(),
		},
	}
	client.CreateAccounts(bankAccount)
	log.Println("✓ Akun Bank (ID=0) dibuat")

	// Buat User A (ID=1) dan User B (ID=2)
	accounts := []types.Account{
		{
			ID:     toUint128(1),
			Ledger: 1,
			Code:   1,
			Flags:  types.AccountFlags{DebitsMustNotExceedCredits: true}.ToUint16(),
		},
		{
			ID:     toUint128(2),
			Ledger: 1,
			Code:   1,
			Flags:  types.AccountFlags{DebitsMustNotExceedCredits: true}.ToUint16(),
		},
	}

	accountResults, err := client.CreateAccounts(accounts)
	if err != nil {
		log.Fatal(err)
	}
	if len(accountResults) == 0 {
		log.Println("✓ Akun User A (ID=1) dan User B (ID=2) berhasil dibuat")
	} else {
		log.Printf("Account creation results: %+v", accountResults)
	}

	// Top-up User A: Rp 500.000
	topup := []types.Transfer{
		{
			ID:              toUint128(200), // New ID since 100 failed
			DebitAccountID:  toUint128(100), // Changed from 0
			CreditAccountID: toUint128(1),
			Amount:          toUint128(500000),
			Ledger:          1,
			Code:            1,
		},
	}

	topupResults, err := client.CreateTransfers(topup)
	if err != nil {
		log.Fatal(err)
	}
	if len(topupResults) == 0 {
		log.Println("✓ Top-up Rp 500.000 ke User A berhasil")
	} else {
		log.Printf("Top-up failed: %+v", topupResults) // Add this
	}

	// Transfer dari User A ke User B: Rp 150.000
	transfers := []types.Transfer{
		{
			ID:              toUint128(201), // Changed from 101
			DebitAccountID:  toUint128(1),
			CreditAccountID: toUint128(2),
			Amount:          toUint128(150000),
			Ledger:          1,
			Code:            2,
		},
	}

	transferResults, err := client.CreateTransfers(transfers)
	if err != nil {
		log.Fatal(err)
	}
	if len(transferResults) == 0 {
		log.Println("✓ Transfer Rp 150.000 dari User A ke User B berhasil")
	} else {
		log.Printf("Transfer failed: %+v", transferResults)
	}

	// Cek saldo
	lookupAccounts, err := client.LookupAccounts([]types.Uint128{
		toUint128(100), // Changed from 0
		toUint128(1),
		toUint128(2),
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n====== SALDO AKHIR ======")
	for _, acc := range lookupAccounts {
		name := getName(acc.ID)
		credits := acc.CreditsPosted.BigInt()
		debits := acc.DebitsPosted.BigInt()
		saldo := new(big.Int).Sub(&credits, &debits)
		fmt.Printf("%s: Rp %d\n", name, saldo)
		fmt.Printf("  - Credits: %d\n", &credits)
		fmt.Printf("  - Debits:  %d\n", &debits)
	}
}

func toUint128(id uint64) types.Uint128 {
	return types.ToUint128(id)
}

func getName(id types.Uint128) string {
	idVal := id.BigInt()
	idUint64 := (&idVal).Uint64()
	switch idUint64 {
	case 100: // Changed from 0
		return "Bank"
	case 1:
		return "User A"
	case 2:
		return "User B"
	default:
		return fmt.Sprintf("Unknown (ID=%d)", idVal)
	}
}
