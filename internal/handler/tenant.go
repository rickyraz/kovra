package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kovra/internal/models"
	"kovra/internal/repository"
)

// TenantHandler handles tenant endpoints.
type TenantHandler struct {
	repo *repository.TenantRepository
}

// NewTenantHandler creates a new tenant handler.
func NewTenantHandler(repo *repository.TenantRepository) *TenantHandler {
	return &TenantHandler{repo: repo}
}

// CreateTenantRequest represents a tenant creation request.
type CreateTenantRequest struct {
	DisplayName    string          `json:"display_name"`
	LegalName      string          `json:"legal_name"`
	Country        string          `json:"country"`
	TenantKind     string          `json:"tenant_kind"`
	ParentTenantID *uuid.UUID      `json:"parent_tenant_id,omitempty"`
	LegalEntityID  uuid.UUID       `json:"legal_entity_id"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
}

// Create creates a new tenant.
// POST /api/v1/tenants
func (h *TenantHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "invalid request body")
		return
	}

	// Validate required fields
	if req.DisplayName == "" || req.LegalName == "" || req.Country == "" {
		BadRequest(w, "display_name, legal_name, and country are required")
		return
	}

	if req.LegalEntityID == uuid.Nil {
		BadRequest(w, "legal_entity_id is required")
		return
	}

	params := models.CreateTenantParams{
		DisplayName:    req.DisplayName,
		LegalName:      req.LegalName,
		Country:        req.Country,
		TenantKind:     models.TenantKind(req.TenantKind),
		ParentTenantID: req.ParentTenantID,
		LegalEntityID:  req.LegalEntityID,
		Metadata:       req.Metadata,
	}

	tenant, err := h.repo.Create(r.Context(), params)
	if err != nil {
		InternalError(w, "failed to create tenant")
		return
	}

	JSON(w, http.StatusCreated, tenant)
}

// Get returns a tenant by ID.
// GET /api/v1/tenants/{id}
func (h *TenantHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		BadRequest(w, "invalid tenant ID")
		return
	}

	tenant, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		InternalError(w, "failed to get tenant")
		return
	}

	if tenant == nil {
		NotFound(w, "tenant not found")
		return
	}

	JSON(w, http.StatusOK, tenant)
}

// UpdateTenantRequest represents a tenant update request.
type UpdateTenantRequest struct {
	DisplayName          *string          `json:"display_name,omitempty"`
	LegalName            *string          `json:"legal_name,omitempty"`
	TenantStatus         *string          `json:"tenant_status,omitempty"`
	KYCLevel             *string          `json:"kyc_level,omitempty"`
	NettingEnabled       *bool            `json:"netting_enabled,omitempty"`
	NettingWindowMinutes *int             `json:"netting_window_minutes,omitempty"`
	WebhookURL           *string          `json:"webhook_url,omitempty"`
	Metadata             *json.RawMessage `json:"metadata,omitempty"`
}

// Update updates a tenant.
// PATCH /api/v1/tenants/{id}
func (h *TenantHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		BadRequest(w, "invalid tenant ID")
		return
	}

	var req UpdateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "invalid request body")
		return
	}

	params := models.UpdateTenantParams{
		DisplayName:          req.DisplayName,
		LegalName:            req.LegalName,
		NettingEnabled:       req.NettingEnabled,
		NettingWindowMinutes: req.NettingWindowMinutes,
		WebhookURL:           req.WebhookURL,
	}

	if req.TenantStatus != nil {
		status := models.TenantStatus(*req.TenantStatus)
		params.TenantStatus = &status
	}

	if req.KYCLevel != nil {
		level := models.KYCLevel(*req.KYCLevel)
		params.KYCLevel = &level
	}

	if req.Metadata != nil {
		params.Metadata = *req.Metadata
	}

	tenant, err := h.repo.Update(r.Context(), id, params)
	if err != nil {
		InternalError(w, "failed to update tenant")
		return
	}

	JSON(w, http.StatusOK, tenant)
}

// ListByLegalEntity returns tenants by legal entity.
// GET /api/v1/legal-entities/{id}/tenants
func (h *TenantHandler) ListByLegalEntity(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		BadRequest(w, "invalid legal entity ID")
		return
	}

	tenants, err := h.repo.ListByLegalEntity(r.Context(), id)
	if err != nil {
		InternalError(w, "failed to list tenants")
		return
	}

	JSON(w, http.StatusOK, tenants)
}
