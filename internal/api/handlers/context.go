package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

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
	svc     ContextService
	logRepo service.SearchLogRepository
}

func NewContextHandler(svc ContextService, logRepo service.SearchLogRepository) *ContextHandler {
	return &ContextHandler{svc: svc, logRepo: logRepo}
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
	Query      string `json:"query"`
	ProjectID  string `json:"project_id"`
	Type       string `json:"type,omitempty"`
	Status     string `json:"status,omitempty"`
	PathPrefix string `json:"path_prefix,omitempty"`
	SourceType string `json:"source_type,omitempty"`
	Mode       string `json:"mode,omitempty"`
	Exact      bool   `json:"exact,omitempty"`
	Limit      int    `json:"limit,omitempty"`
	Cursor     string `json:"cursor,omitempty"`
}

type SearchResultResponse struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Summary    string  `json:"summary,omitempty"`
	Scope      string  `json:"scope,omitempty"`
	Snippet    string  `json:"snippet,omitempty"`
	UpdatedAt  string  `json:"updated_at,omitempty"`
	Score      float32 `json:"score"`
	SourceType string  `json:"source_type"`
}

type SearchResponse struct {
	Results  []*SearchResultResponse `json:"results"`
	Cursor   string                  `json:"cursor,omitempty"`
	HasMore  bool                    `json:"has_more"`
	SearchID string                  `json:"search_id,omitempty"`
}

type SearchFeedbackRequest struct {
	SearchID   string `json:"search_id"`
	SelectedID string `json:"selected_id"`
	SourceType string `json:"source_type,omitempty"`
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

	start := time.Now()
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
	if req.Status != "" {
		filters.Status = domain.KnowledgeStatus(req.Status)
	}
	if req.PathPrefix != "" {
		filters.PathPrefix = req.PathPrefix
	}
	if req.SourceType != "" {
		filters.SourceType = req.SourceType
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}

	input := service.SearchInput{
		Query:   req.Query,
		Filters: filters,
		Mode:    service.SearchMode(req.Mode),
		Exact:   req.Exact,
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
		updatedAt := ""
		if !result.UpdatedAt.IsZero() {
			updatedAt = result.UpdatedAt.UTC().Format(time.RFC3339Nano)
		}
		responses[i] = &SearchResultResponse{
			ID:         result.ID,
			Title:      result.Title,
			Summary:    result.Summary,
			Scope:      result.Scope,
			Snippet:    result.Snippet,
			UpdatedAt:  updatedAt,
			Score:      result.Score,
			SourceType: result.SourceType,
		}
	}

	if h.logRepo != nil {
		logResults := make([]service.SearchLogResult, 0, len(output.Results))
		for _, result := range output.Results {
			if result == nil {
				continue
			}
			logResults = append(logResults, service.SearchLogResult{
				ID:         result.ID,
				SourceType: normalizeSourceType(result.SourceType),
				Score:      result.Score,
			})
		}
		entry := service.SearchLogEntry{
			OrgID:      orgID,
			ProjectID:  req.ProjectID,
			Query:      req.Query,
			Filters:    filters,
			Mode:       normalizeSearchMode(req.Mode),
			Exact:      req.Exact,
			Limit:      limit,
			DurationMs: int(time.Since(start).Milliseconds()),
			Results:    logResults,
		}
		if searchID, err := h.logRepo.CreateSearchLog(r.Context(), entry); err == nil {
			output.SearchID = searchID
		}
	}

	api.Success(w, http.StatusOK, SearchResponse{
		Results:  responses,
		Cursor:   output.Cursor,
		HasMore:  output.HasMore,
		SearchID: output.SearchID,
	})
}

// SearchFeedback records a selected result for a prior search.
func (h *ContextHandler) SearchFeedback(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgID(r.Context())
	if orgID == "" {
		api.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if h.logRepo == nil {
		api.Error(w, http.StatusNotImplemented, "search feedback not available")
		return
	}

	var req SearchFeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.SearchID == "" || req.SelectedID == "" {
		api.Error(w, http.StatusBadRequest, "search_id and selected_id are required")
		return
	}
	sourceType := normalizeSourceType(req.SourceType)
	if err := h.logRepo.RecordSearchSelection(r.Context(), orgID, req.SearchID, req.SelectedID, sourceType); err != nil {
		api.HandleError(w, err)
		return
	}

	api.Success(w, http.StatusOK, map[string]any{"status": "ok"})
}

func normalizeSourceType(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "asset" {
		return "asset"
	}
	return "knowledge"
}

func normalizeSearchMode(value string) service.SearchMode {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "semantic":
		return service.SearchModeSemantic
	case "lexical":
		return service.SearchModeLexical
	default:
		return service.SearchModeHybrid
	}
}
