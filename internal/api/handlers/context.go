package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/cloo-solutions/neotexai/internal/api"
	"github.com/cloo-solutions/neotexai/internal/api/middleware"
	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/cloo-solutions/neotexai/internal/service"
)

type ContextService interface {
	GetManifest(ctx context.Context, orgID, projectID string) ([]*service.KnowledgeManifestItem, error)
	Search(ctx context.Context, input service.SearchInput) (*service.SearchOutput, error)
}

type ContextHandler struct {
	svc ContextService
}

func NewContextHandler(svc ContextService) *ContextHandler {
	return &ContextHandler{svc: svc}
}

type ManifestItemResponse struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Type    string `json:"type"`
	Scope   string `json:"scope,omitempty"`
}

type ManifestResponse struct {
	Manifest []*ManifestItemResponse `json:"manifest"`
}

type SearchRequest struct {
	Query     string `json:"query"`
	ProjectID string `json:"project_id"`
	Type      string `json:"type,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Cursor    string `json:"cursor,omitempty"`
}

type SearchResultResponse struct {
	ID      string  `json:"id"`
	Title   string  `json:"title"`
	Summary string  `json:"summary,omitempty"`
	Scope   string  `json:"scope,omitempty"`
	Score   float32 `json:"score"`
}

type SearchResponse struct {
	Results []*SearchResultResponse `json:"results"`
	Cursor  string                  `json:"cursor,omitempty"`
	HasMore bool                    `json:"has_more"`
}

func (h *ContextHandler) GetManifest(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgID(r.Context())
	if orgID == "" {
		api.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	projectID := r.URL.Query().Get("project_id")

	items, err := h.svc.GetManifest(r.Context(), orgID, projectID)
	if err != nil {
		api.HandleError(w, err)
		return
	}

	responses := make([]*ManifestItemResponse, len(items))
	for i, item := range items {
		responses[i] = &ManifestItemResponse{
			ID:      item.ID,
			Title:   item.Title,
			Summary: item.Summary,
			Type:    string(item.Type),
			Scope:   item.Scope,
		}
	}

	api.Success(w, http.StatusOK, ManifestResponse{Manifest: responses})
}

func (h *ContextHandler) Search(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgID(r.Context())
	if orgID == "" {
		api.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Query == "" {
		api.Error(w, http.StatusBadRequest, "query is required")
		return
	}

	filters := service.SearchFilters{
		OrgID:     orgID,
		ProjectID: req.ProjectID,
	}

	if req.Type != "" {
		filters.Type = domain.KnowledgeType(req.Type)
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}

	input := service.SearchInput{
		Query:   req.Query,
		Filters: filters,
		Limit:   limit,
		Cursor:  req.Cursor,
	}

	output, err := h.svc.Search(r.Context(), input)
	if err != nil {
		api.HandleError(w, err)
		return
	}

	responses := make([]*SearchResultResponse, len(output.Results))
	for i, result := range output.Results {
		responses[i] = &SearchResultResponse{
			ID:      result.ID,
			Title:   result.Title,
			Summary: result.Summary,
			Scope:   result.Scope,
			Score:   result.Score,
		}
	}

	api.Success(w, http.StatusOK, SearchResponse{
		Results: responses,
		Cursor:  output.Cursor,
		HasMore: output.HasMore,
	})
}
