package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/cloo-solutions/neotexai/internal/api"
	"github.com/cloo-solutions/neotexai/internal/api/middleware"
	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ProjectRepository interface {
	Create(ctx context.Context, project *domain.Project) error
	GetByID(ctx context.Context, id string) (*domain.Project, error)
	ListByOrg(ctx context.Context, orgID string) ([]*domain.Project, error)
}

type ProjectHandler struct {
	repo ProjectRepository
}

func NewProjectHandler(repo ProjectRepository) *ProjectHandler {
	return &ProjectHandler{repo: repo}
}

type CreateProjectRequest struct {
	Name string `json:"name"`
}

func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgID(r.Context())
	if orgID == "" {
		api.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		api.Error(w, http.StatusBadRequest, "name is required")
		return
	}

	project := &domain.Project{
		ID:        uuid.NewString(),
		OrgID:     orgID,
		Name:      req.Name,
		CreatedAt: time.Now().UTC(),
	}

	if err := h.repo.Create(r.Context(), project); err != nil {
		api.Error(w, http.StatusInternalServerError, "failed to create project")
		return
	}

	api.Success(w, http.StatusCreated, project)
}

func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgID(r.Context())
	if orgID == "" {
		api.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	projectID := chi.URLParam(r, "id")
	project, err := h.repo.GetByID(r.Context(), projectID)
	if err != nil {
		if err == domain.ErrProjectNotFound {
			api.Error(w, http.StatusNotFound, "project not found")
			return
		}
		api.Error(w, http.StatusInternalServerError, "failed to get project")
		return
	}

	if project.OrgID != orgID {
		api.Error(w, http.StatusNotFound, "project not found")
		return
	}

	api.Success(w, http.StatusOK, project)
}

func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgID(r.Context())
	if orgID == "" {
		api.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	projects, err := h.repo.ListByOrg(r.Context(), orgID)
	if err != nil {
		api.Error(w, http.StatusInternalServerError, "failed to list projects")
		return
	}

	if projects == nil {
		projects = []*domain.Project{}
	}

	api.Success(w, http.StatusOK, projects)
}
