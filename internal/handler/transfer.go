package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"kovra/internal/models"
	"kovra/internal/repository"
)

// TransferHandler handles transfer endpoints.
type TransferHandler struct {
	repo       *repository.TransferRepository
	walletRepo *repository.WalletRepository
}

// NewTransferHandler creates a new transfer handler.
func NewTransferHandler(repo *repository.TransferRepository, walletRepo *repository.WalletRepository) *TransferHandler {
	return &TransferHandler{
		repo:       repo,
		walletRepo: walletRepo,
	}
}

// CreateTransferRequest represents a transfer creation request.
type CreateTransferRequest struct {
	TenantID            uuid.UUID  `json:"tenant_id"`
	SourceLegalEntityID *uuid.UUID `json:"source_legal_entity_id,omitempty"`
	DestLegalEntityID   *uuid.UUID `json:"dest_legal_entity_id,omitempty"`
	QuoteID             *uuid.UUID `json:"quote_id,omitempty"`
	RecipientID         *uuid.UUID `json:"recipient_id,omitempty"`
	IdempotencyKey      *string    `json:"idempotency_key,omitempty"`
	FromCurrency        string     `json:"from_currency"`
	ToCurrency          string     `json:"to_currency"`
	FromAmount          string     `json:"from_amount"`
	ToAmount            string     `json:"to_amount"`
	FXRate              string     `json:"fx_rate"`
	TotalFee            string     `json:"total_fee,omitempty"`
	Rail                *string    `json:"rail,omitempty"`
}

// Create creates a new transfer.
// POST /api/v1/transfers
func (h *TransferHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "invalid request body")
		return
	}

	// Validate required fields
	if req.TenantID == uuid.Nil {
		BadRequest(w, "tenant_id is required")
		return
	}

	if req.FromCurrency == "" || req.ToCurrency == "" {
		BadRequest(w, "from_currency and to_currency are required")
		return
	}

	// Parse amounts
	fromAmount, err := decimal.NewFromString(req.FromAmount)
	if err != nil {
		BadRequest(w, "invalid from_amount")
		return
	}

	toAmount, err := decimal.NewFromString(req.ToAmount)
	if err != nil {
		BadRequest(w, "invalid to_amount")
		return
	}

	fxRate, err := decimal.NewFromString(req.FXRate)
	if err != nil {
		BadRequest(w, "invalid fx_rate")
		return
	}

	var totalFee decimal.Decimal
	if req.TotalFee != "" {
		totalFee, err = decimal.NewFromString(req.TotalFee)
		if err != nil {
			BadRequest(w, "invalid total_fee")
			return
		}
	}

	// Check idempotency
	if req.IdempotencyKey != nil {
		existing, err := h.repo.GetByIdempotencyKey(r.Context(), req.TenantID, *req.IdempotencyKey)
		if err != nil {
			InternalError(w, "failed to check idempotency")
			return
		}
		if existing != nil {
			JSON(w, http.StatusOK, existing) // Return existing transfer
			return
		}
	}

	var rail *models.Rail
	if req.Rail != nil {
		r := models.Rail(*req.Rail)
		rail = &r
	}

	params := models.CreateTransferParams{
		TenantID:            req.TenantID,
		SourceLegalEntityID: req.SourceLegalEntityID,
		DestLegalEntityID:   req.DestLegalEntityID,
		QuoteID:             req.QuoteID,
		RecipientID:         req.RecipientID,
		IdempotencyKey:      req.IdempotencyKey,
		FromCurrency:        req.FromCurrency,
		ToCurrency:          req.ToCurrency,
		FromAmount:          fromAmount,
		ToAmount:            toAmount,
		FXRate:              fxRate,
		TotalFee:            totalFee,
		Rail:                rail,
	}

	transfer, err := h.repo.Create(r.Context(), params)
	if err != nil {
		InternalError(w, "failed to create transfer")
		return
	}

	JSON(w, http.StatusCreated, transfer)
}

// Get returns a transfer by ID.
// GET /api/v1/transfers/{id}
func (h *TransferHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		BadRequest(w, "invalid transfer ID")
		return
	}

	transfer, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		InternalError(w, "failed to get transfer")
		return
	}

	if transfer == nil {
		NotFound(w, "transfer not found")
		return
	}

	JSON(w, http.StatusOK, transfer)
}

// ListByTenant returns transfers for a tenant.
// GET /api/v1/tenants/{id}/transfers
func (h *TransferHandler) ListByTenant(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		BadRequest(w, "invalid tenant ID")
		return
	}

	// Parse query params
	filter := models.TransferFilter{
		Limit:  100,
		Offset: 0,
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 1000 {
			filter.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	if status := r.URL.Query().Get("status"); status != "" {
		s := models.TransferStatus(status)
		filter.Status = &s
	}

	if fromCurrency := r.URL.Query().Get("from_currency"); fromCurrency != "" {
		filter.FromCurrency = &fromCurrency
	}

	if toCurrency := r.URL.Query().Get("to_currency"); toCurrency != "" {
		filter.ToCurrency = &toCurrency
	}

	transfers, err := h.repo.ListByTenant(r.Context(), id, filter)
	if err != nil {
		InternalError(w, "failed to list transfers")
		return
	}

	JSON(w, http.StatusOK, transfers)
}
