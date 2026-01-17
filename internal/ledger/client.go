package ledger

import (
	"fmt"

	tb "github.com/tigerbeetle/tigerbeetle-go"
	tbtypes "github.com/tigerbeetle/tigerbeetle-go/pkg/types"

	"kovra/internal/config"
)

// Client wraps the TigerBeetle client with domain-specific operations.
type Client struct {
	tb        tb.Client
	clusterID uint64
}

// NewClient creates a new TigerBeetle client.
func NewClient(cfg config.TigerBeetleConfig) (*Client, error) {
	// Convert addresses to TigerBeetle format
	addresses := make([]string, len(cfg.Addresses))
	copy(addresses, cfg.Addresses)

	client, err := tb.NewClient(tbtypes.ToUint128(cfg.ClusterID), addresses)
	if err != nil {
		return nil, fmt.Errorf("create TigerBeetle client: %w", err)
	}

	return &Client{
		tb:        client,
		clusterID: cfg.ClusterID,
	}, nil
}

// Close closes the TigerBeetle client connection.
func (c *Client) Close() {
	c.tb.Close()
}

// CreateAccount creates a new account in TigerBeetle.
func (c *Client) CreateAccount(id AccountID, ledger uint32, code uint16) error {
	accounts := []tbtypes.Account{{
		ID:     tbtypes.BytesToUint128(id),
		Ledger: ledger,
		Code:   code,
		Flags:  0,
	}}

	results, err := c.tb.CreateAccounts(accounts)
	if err != nil {
		return fmt.Errorf("create account: %w", err)
	}

	for _, result := range results {
		if result.Result != tbtypes.AccountOK {
			return fmt.Errorf("create account failed: %s", createAccountResultString(result.Result))
		}
	}

	return nil
}

// CreateAccountWithFlags creates a new account with custom flags.
func (c *Client) CreateAccountWithFlags(id AccountID, ledger uint32, code uint16, flags tbtypes.AccountFlags) error {
	accounts := []tbtypes.Account{{
		ID:     tbtypes.BytesToUint128(id),
		Ledger: ledger,
		Code:   code,
		Flags:  flags.ToUint16(),
	}}

	results, err := c.tb.CreateAccounts(accounts)
	if err != nil {
		return fmt.Errorf("create account: %w", err)
	}

	for _, result := range results {
		if result.Result != tbtypes.AccountOK {
			return fmt.Errorf("create account failed: %s", createAccountResultString(result.Result))
		}
	}

	return nil
}

// GetAccount retrieves an account from TigerBeetle.
func (c *Client) GetAccount(id AccountID) (*tbtypes.Account, error) {
	accounts, err := c.tb.LookupAccounts([]tbtypes.Uint128{tbtypes.BytesToUint128(id)})
	if err != nil {
		return nil, fmt.Errorf("lookup account: %w", err)
	}

	if len(accounts) == 0 {
		return nil, nil // Account not found
	}

	return &accounts[0], nil
}

// GetBalance retrieves the balance for an account.
func (c *Client) GetBalance(id AccountID) (Balance, error) {
	account, err := c.GetAccount(id)
	if err != nil {
		return Balance{}, err
	}

	if account == nil {
		return Balance{}, fmt.Errorf("account not found")
	}

	return Balance{
		Debits:   uint128ToUint64(account.DebitsPosted),
		Credits:  uint128ToUint64(account.CreditsPosted),
		Pending:  uint128ToUint64(account.DebitsPending),
		Reserved: 0,
	}, nil
}

// CreateTransfer creates a single transfer between accounts.
func (c *Client) CreateTransfer(transfer Transfer) error {
	transfers := []tbtypes.Transfer{transfer.toTigerBeetle()}

	results, err := c.tb.CreateTransfers(transfers)
	if err != nil {
		return fmt.Errorf("create transfer: %w", err)
	}

	for _, result := range results {
		if result.Result != tbtypes.TransferOK {
			return fmt.Errorf("create transfer failed: %s", createTransferResultString(result.Result))
		}
	}

	return nil
}

// CreateTransfers creates multiple transfers atomically.
func (c *Client) CreateTransfers(transfers []Transfer) error {
	tbTransfers := make([]tbtypes.Transfer, len(transfers))
	for i, t := range transfers {
		tbTransfers[i] = t.toTigerBeetle()
	}

	results, err := c.tb.CreateTransfers(tbTransfers)
	if err != nil {
		return fmt.Errorf("create transfers: %w", err)
	}

	for _, result := range results {
		if result.Result != tbtypes.TransferOK {
			return fmt.Errorf("create transfer %d failed: %s", result.Index, createTransferResultString(result.Result))
		}
	}

	return nil
}

// CreateLinkedTransfers creates a chain of linked transfers (all-or-nothing).
func (c *Client) CreateLinkedTransfers(transfers []Transfer) error {
	if len(transfers) == 0 {
		return nil
	}

	// Mark all but the last transfer as linked
	for i := range transfers[:len(transfers)-1] {
		transfers[i].Flags |= TransferFlagLinked
	}

	return c.CreateTransfers(transfers)
}

// uint128ToUint64 converts TigerBeetle Uint128 to uint64.
// Note: This may overflow for very large values.
func uint128ToUint64(v tbtypes.Uint128) uint64 {
	bi := v.BigInt()
	return bi.Uint64()
}

// createAccountResultString converts account creation result to string.
func createAccountResultString(result tbtypes.CreateAccountResult) string {
	if result == tbtypes.AccountOK {
		return "OK"
	}
	// Use the built-in String() method for other results
	return result.String()
}

// createTransferResultString converts transfer creation result to string.
func createTransferResultString(result tbtypes.CreateTransferResult) string {
	if result == tbtypes.TransferOK {
		return "OK"
	}
	// Use the built-in String() method for other results
	return result.String()
}
