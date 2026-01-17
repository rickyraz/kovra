package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kovra/internal/ledger"
	"kovra/internal/models"
	"kovra/internal/repository"
)

// WalletHandler handles wallet endpoints.
type WalletHandler struct {
	repo         *repository.WalletRepository
	ledgerClient *ledger.Client
}

// NewWalletHandler creates a new wallet handler.
func NewWalletHandler(repo *repository.WalletRepository, ledgerClient *ledger.Client) *WalletHandler {
	return &WalletHandler{
		repo:         repo,
		ledgerClient: ledgerClient,
	}
}

// CreateWalletRequest represents a wallet creation request.
type CreateWalletRequest struct {
	TenantID uuid.UUID `json:"tenant_id"`
	Currency string    `json:"currency"`
}

// Create creates a new wallet.
// POST /api/v1/wallets
func (h *WalletHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateWalletRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "invalid request body")
		return
	}

	if req.TenantID == uuid.Nil {
		BadRequest(w, "tenant_id is required")
		return
	}

	if req.Currency == "" || len(req.Currency) != 3 {
		BadRequest(w, "currency must be a 3-letter ISO code")
		return
	}

	// Check if wallet already exists
	existing, err := h.repo.GetByTenantAndCurrency(r.Context(), req.TenantID, req.Currency)
	if err != nil {
		InternalError(w, "failed to check existing wallet")
		return
	}
	if existing != nil {
		Conflict(w, "wallet already exists for this tenant and currency")
		return
	}

	// Create TigerBeetle account
	currency := ledger.CurrencyFromString(req.Currency)
	if currency == 0 {
		BadRequest(w, "unsupported currency")
		return
	}

	// Use lower 64 bits of tenant UUID as tenant ID for TigerBeetle
	tenantID := uint64(req.TenantID[8])<<56 | uint64(req.TenantID[9])<<48 |
		uint64(req.TenantID[10])<<40 | uint64(req.TenantID[11])<<32 |
		uint64(req.TenantID[12])<<24 | uint64(req.TenantID[13])<<16 |
		uint64(req.TenantID[14])<<8 | uint64(req.TenantID[15])

	accountID := ledger.NewAccountID(tenantID, ledger.AccountTypeTenantWallet, currency)

	// Create account in TigerBeetle
	err = h.ledgerClient.CreateAccount(accountID, uint32(currency), uint16(ledger.AccountTypeTenantWallet))
	if err != nil {
		InternalError(w, "failed to create ledger account")
		return
	}

	// Create wallet in PostgreSQL
	params := models.CreateWalletParams{
		TenantID:    req.TenantID,
		Currency:    req.Currency,
		TBAccountID: accountID.ToBigInt(),
	}

	wallet, err := h.repo.Create(r.Context(), params)
	if err != nil {
		InternalError(w, "failed to create wallet")
		return
	}

	JSON(w, http.StatusCreated, wallet)
}

// Get returns a wallet by ID.
// GET /api/v1/wallets/{id}
func (h *WalletHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		BadRequest(w, "invalid wallet ID")
		return
	}

	wallet, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		InternalError(w, "failed to get wallet")
		return
	}

	if wallet == nil {
		NotFound(w, "wallet not found")
		return
	}

	JSON(w, http.StatusOK, wallet)
}

// ListByTenant returns all wallets for a tenant.
// GET /api/v1/tenants/{id}/wallets
func (h *WalletHandler) ListByTenant(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		BadRequest(w, "invalid tenant ID")
		return
	}

	wallets, err := h.repo.ListByTenant(r.Context(), id)
	if err != nil {
		InternalError(w, "failed to list wallets")
		return
	}

	JSON(w, http.StatusOK, wallets)
}

// WalletBalanceResponse represents wallet balance.
type WalletBalanceResponse struct {
	WalletID  uuid.UUID `json:"wallet_id"`
	Currency  string    `json:"currency"`
	Available string    `json:"available"`
	Pending   string    `json:"pending"`
	Total     string    `json:"total"`
}

// GetBalance returns the current balance from TigerBeetle.
// GET /api/v1/wallets/{id}/balance
func (h *WalletHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		BadRequest(w, "invalid wallet ID")
		return
	}

	wallet, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		InternalError(w, "failed to get wallet")
		return
	}

	if wallet == nil {
		NotFound(w, "wallet not found")
		return
	}

	// Get balance from TigerBeetle
	accountID := ledger.FromBigInt(wallet.TBAccountID)
	balance, err := h.ledgerClient.GetBalance(accountID)
	if err != nil {
		InternalError(w, "failed to get balance from ledger")
		return
	}

	resp := WalletBalanceResponse{
		WalletID:  wallet.ID,
		Currency:  wallet.Currency,
		Available: formatAmount(balance.Available()),
		Pending:   formatAmount(int64(balance.Pending)),
		Total:     formatAmount(balance.Total()),
	}

	JSON(w, http.StatusOK, resp)
}

// formatAmount formats an amount in minor units to major units with 2 decimals.
func formatAmount(minorUnits int64) string {
	major := minorUnits / 100
	minor := minorUnits % 100
	if minor < 0 {
		minor = -minor
	}
	return fmt.Sprintf("%d.%02d", major, minor)
}
