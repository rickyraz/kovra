package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kovra/internal/repository"
)

// LegalEntityHandler handles legal entity endpoints.
type LegalEntityHandler struct {
	repo *repository.LegalEntityRepository
}

// NewLegalEntityHandler creates a new legal entity handler.
func NewLegalEntityHandler(repo *repository.LegalEntityRepository) *LegalEntityHandler {
	return &LegalEntityHandler{repo: repo}
}

// List returns all legal entities.
// GET /api/v1/legal-entities
func (h *LegalEntityHandler) List(w http.ResponseWriter, r *http.Request) {
	entities, err := h.repo.List(r.Context())
	if err != nil {
		InternalError(w, "failed to list legal entities")
		return
	}

	JSON(w, http.StatusOK, entities)
}

// Get returns a legal entity by ID.
// GET /api/v1/legal-entities/{id}
func (h *LegalEntityHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		BadRequest(w, "invalid legal entity ID")
		return
	}

	entity, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		InternalError(w, "failed to get legal entity")
		return
	}

	if entity == nil {
		NotFound(w, "legal entity not found")
		return
	}

	JSON(w, http.StatusOK, entity)
}

// GetByCode returns a legal entity by code.
// GET /api/v1/legal-entities/code/{code}
func (h *LegalEntityHandler) GetByCode(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		BadRequest(w, "code is required")
		return
	}

	entity, err := h.repo.GetByCode(r.Context(), code)
	if err != nil {
		InternalError(w, "failed to get legal entity")
		return
	}

	if entity == nil {
		NotFound(w, "legal entity not found")
		return
	}

	JSON(w, http.StatusOK, entity)
}
