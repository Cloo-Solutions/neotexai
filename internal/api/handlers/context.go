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

type VFSService interface {
	Open(ctx context.Context, input service.OpenInput) (*service.OpenResult, error)
	List(ctx context.Context, input service.ListInput) (*service.ListOutput, error)
}

type ContextHandler struct {
	svc     ContextService
	vfs     VFSService
	logRepo service.SearchLogRepository
}

func NewContextHandler(svc ContextService, logRepo service.SearchLogRepository) *ContextHandler {
	return &ContextHandler{svc: svc, logRepo: logRepo}
}

// NewContextHandlerWithVFS creates a context handler with VFS support
func NewContextHandlerWithVFS(svc ContextService, vfs VFSService, logRepo service.SearchLogRepository) *ContextHandler {
	return &ContextHandler{svc: svc, vfs: vfs, logRepo: logRepo}
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
	ChunkID    string  `json:"chunk_id,omitempty"`
	ChunkIndex int     `json:"chunk_index,omitempty"`
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

type OpenRequest struct {
	ID         string        `json:"id"`
	SourceType string        `json:"source_type,omitempty"`
	ChunkID    string        `json:"chunk_id,omitempty"`
	Range      *ContentRange `json:"range,omitempty"`
	IncludeURL bool          `json:"include_url,omitempty"`
}

type ContentRange struct {
	StartLine int `json:"start_line,omitempty"`
	EndLine   int `json:"end_line,omitempty"`
	MaxChars  int `json:"max_chars,omitempty"`
}

type OpenResponse struct {
	ID          string   `json:"id"`
	SourceType  string   `json:"source_type"`
	Title       string   `json:"title"`
	Content     string   `json:"content,omitempty"`
	TotalLines  int      `json:"total_lines,omitempty"`
	TotalChars  int      `json:"total_chars,omitempty"`
	ChunkID     string   `json:"chunk_id,omitempty"`
	ChunkIndex  int      `json:"chunk_index,omitempty"`
	ChunkCount  int      `json:"chunk_count,omitempty"`
	UpdatedAt   string   `json:"updated_at,omitempty"`
	Filename    string   `json:"filename,omitempty"`
	MimeType    string   `json:"mime_type,omitempty"`
	SizeBytes   int64    `json:"size_bytes,omitempty"`
	Description string   `json:"description,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	DownloadURL string   `json:"download_url,omitempty"`
}

type ListRequest struct {
	ProjectID    string `json:"project_id,omitempty"`
	PathPrefix   string `json:"path_prefix,omitempty"`
	Type         string `json:"type,omitempty"`
	Status       string `json:"status,omitempty"`
	SourceType   string `json:"source_type,omitempty"`
	UpdatedSince string `json:"updated_since,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	Cursor       string `json:"cursor,omitempty"`
}

type ListItemResponse struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Scope      string `json:"scope,omitempty"`
	Type       string `json:"type,omitempty"`
	Status     string `json:"status,omitempty"`
	SourceType string `json:"source_type"`
	UpdatedAt  string `json:"updated_at,omitempty"`
	ChunkCount int    `json:"chunk_count,omitempty"`
	Filename   string `json:"filename,omitempty"`
	MimeType   string `json:"mime_type,omitempty"`
}

type ListResponse struct {
	Items   []*ListItemResponse `json:"items"`
	Cursor  string              `json:"cursor,omitempty"`
	HasMore bool                `json:"has_more"`
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
			ChunkID:    result.ChunkID,
			ChunkIndex: result.ChunkIndex,
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

// Open retrieves content for a knowledge item, chunk, or asset
func (h *ContextHandler) Open(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgID(r.Context())
	if orgID == "" {
		api.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if h.vfs == nil {
		api.Error(w, http.StatusNotImplemented, "open not available")
		return
	}

	var req OpenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ID == "" {
		api.Error(w, http.StatusBadRequest, "id is required")
		return
	}

	input := service.OpenInput{
		ID:         req.ID,
		SourceType: req.SourceType,
		ChunkID:    req.ChunkID,
		IncludeURL: req.IncludeURL,
	}

	if req.Range != nil {
		input.Range = &service.ContentRange{
			StartLine: req.Range.StartLine,
			EndLine:   req.Range.EndLine,
			MaxChars:  req.Range.MaxChars,
		}
	}

	result, err := h.vfs.Open(r.Context(), input)
	if err != nil {
		api.HandleError(w, err)
		return
	}

	updatedAt := ""
	if !result.UpdatedAt.IsZero() {
		updatedAt = result.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}

	resp := OpenResponse{
		ID:          result.ID,
		SourceType:  result.SourceType,
		Title:       result.Title,
		Content:     result.Content,
		TotalLines:  result.TotalLines,
		TotalChars:  result.TotalChars,
		ChunkID:     result.ChunkID,
		ChunkIndex:  result.ChunkIndex,
		ChunkCount:  result.ChunkCount,
		UpdatedAt:   updatedAt,
		Filename:    result.Filename,
		MimeType:    result.MimeType,
		SizeBytes:   result.SizeBytes,
		Description: result.Description,
		Keywords:    result.Keywords,
		DownloadURL: result.DownloadURL,
	}

	api.Success(w, http.StatusOK, resp)
}

// List retrieves metadata for knowledge items and/or assets
func (h *ContextHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgID(r.Context())
	if orgID == "" {
		api.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if h.vfs == nil {
		api.Error(w, http.StatusNotImplemented, "list not available")
		return
	}

	var req ListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	input := service.ListInput{
		OrgID:      orgID,
		ProjectID:  req.ProjectID,
		PathPrefix: req.PathPrefix,
		SourceType: req.SourceType,
		Limit:      req.Limit,
		Cursor:     req.Cursor,
	}

	if req.Type != "" {
		input.Type = domain.KnowledgeType(req.Type)
	}
	if req.Status != "" {
		input.Status = domain.KnowledgeStatus(req.Status)
	}
	if req.UpdatedSince != "" {
		t, err := time.Parse(time.RFC3339, req.UpdatedSince)
		if err == nil {
			input.UpdatedSince = &t
		}
	}

	result, err := h.vfs.List(r.Context(), input)
	if err != nil {
		api.HandleError(w, err)
		return
	}

	items := make([]*ListItemResponse, len(result.Items))
	for i, item := range result.Items {
		updatedAt := ""
		if !item.UpdatedAt.IsZero() {
			updatedAt = item.UpdatedAt.UTC().Format(time.RFC3339Nano)
		}
		items[i] = &ListItemResponse{
			ID:         item.ID,
			Title:      item.Title,
			Scope:      item.Scope,
			Type:       string(item.Type),
			Status:     string(item.Status),
			SourceType: item.SourceType,
			UpdatedAt:  updatedAt,
			ChunkCount: item.ChunkCount,
			Filename:   item.Filename,
			MimeType:   item.MimeType,
		}
	}

	api.Success(w, http.StatusOK, ListResponse{
		Items:   items,
		Cursor:  result.Cursor,
		HasMore: result.HasMore,
	})
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
