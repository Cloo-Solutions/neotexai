package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/cloo-solutions/neotexai/internal/api"
	"github.com/cloo-solutions/neotexai/internal/domain"
)

type AuthService interface {
	CreateOrg(ctx context.Context, name string) (*domain.Organization, error)
	CreateAPIKey(ctx context.Context, orgID, name string) (string, error)
}

type AuthHandler struct {
	svc AuthService
}

func NewAuthHandler(svc AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

type CreateOrgRequest struct {
	Name string `json:"name"`
}

type OrgResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

type CreateAPIKeyRequest struct {
	OrgID string `json:"org_id"`
	Name  string `json:"name"`
}

type APIKeyResponse struct {
	ID    string `json:"id"`
	Token string `json:"token"`
	Name  string `json:"name"`
}

func (h *AuthHandler) CreateOrg(w http.ResponseWriter, r *http.Request) {
	var req CreateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		api.Error(w, http.StatusBadRequest, "name is required")
		return
	}

	org, err := h.svc.CreateOrg(r.Context(), req.Name)
	if err != nil {
		api.HandleError(w, err)
		return
	}

	api.Success(w, http.StatusCreated, OrgResponse{
		ID:        org.ID,
		Name:      org.Name,
		CreatedAt: org.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *AuthHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.OrgID == "" {
		api.Error(w, http.StatusBadRequest, "org_id is required")
		return
	}
	if req.Name == "" {
		api.Error(w, http.StatusBadRequest, "name is required")
		return
	}

	token, err := h.svc.CreateAPIKey(r.Context(), req.OrgID, req.Name)
	if err != nil {
		api.HandleError(w, err)
		return
	}

	api.Success(w, http.StatusCreated, APIKeyResponse{
		Token: token,
		Name:  req.Name,
	})
}
