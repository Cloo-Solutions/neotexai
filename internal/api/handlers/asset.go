package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/cloo-solutions/neotexai/internal/api"
	"github.com/cloo-solutions/neotexai/internal/api/middleware"
	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/cloo-solutions/neotexai/internal/service"
	"github.com/go-chi/chi/v5"
)

type AssetService interface {
	InitUpload(ctx context.Context, input service.InitUploadInput) (*service.InitUploadResult, error)
	CompleteUpload(ctx context.Context, input service.CompleteUploadInput) (*domain.Asset, error)
	GetDownloadURL(ctx context.Context, assetID string) (string, error)
	GetByID(ctx context.Context, assetID string) (*domain.Asset, error)
}

type AssetHandler struct {
	svc AssetService
}

func NewAssetHandler(svc AssetService) *AssetHandler {
	return &AssetHandler{svc: svc}
}

type InitUploadRequest struct {
	Filename  string `json:"filename"`
	MimeType  string `json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
	ProjectID string `json:"project_id"`
}

type InitUploadResponse struct {
	AssetID    string `json:"asset_id"`
	StorageKey string `json:"storage_key"`
	UploadURL  string `json:"upload_url"`
}

type CompleteUploadRequest struct {
	AssetID     string   `json:"asset_id"`
	StorageKey  string   `json:"storage_key"`
	Filename    string   `json:"filename"`
	MimeType    string   `json:"mime_type"`
	SHA256      string   `json:"sha256"`
	ProjectID   string   `json:"project_id,omitempty"`
	KnowledgeID string   `json:"knowledge_id,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	Description string   `json:"description,omitempty"`
}

type AssetResponse struct {
	ID          string   `json:"id"`
	OrgID       string   `json:"org_id"`
	ProjectID   string   `json:"project_id"`
	Filename    string   `json:"filename"`
	MimeType    string   `json:"mime_type"`
	SHA256      string   `json:"sha256"`
	Keywords    []string `json:"keywords,omitempty"`
	Description string   `json:"description,omitempty"`
	CreatedAt   string   `json:"created_at"`
}

type DownloadURLResponse struct {
	DownloadURL string `json:"download_url"`
}

func assetToResponse(a *domain.Asset) *AssetResponse {
	return &AssetResponse{
		ID:          a.ID,
		OrgID:       a.OrgID,
		ProjectID:   a.ProjectID,
		Filename:    a.Filename,
		MimeType:    a.MimeType,
		SHA256:      a.SHA256,
		Keywords:    a.Keywords,
		Description: a.Description,
		CreatedAt:   a.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func (h *AssetHandler) InitUpload(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgID(r.Context())
	if orgID == "" {
		api.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req InitUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Filename == "" {
		api.Error(w, http.StatusBadRequest, "filename is required")
		return
	}
	if req.MimeType == "" {
		api.Error(w, http.StatusBadRequest, "mime_type is required")
		return
	}

	input := service.InitUploadInput{
		OrgID:       orgID,
		ProjectID:   req.ProjectID,
		Filename:    req.Filename,
		ContentType: req.MimeType,
	}

	result, err := h.svc.InitUpload(r.Context(), input)
	if err != nil {
		api.HandleError(w, err)
		return
	}

	api.Success(w, http.StatusOK, InitUploadResponse{
		AssetID:    result.AssetID,
		StorageKey: result.StorageKey,
		UploadURL:  result.UploadURL,
	})
}

func (h *AssetHandler) CompleteUpload(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgID(r.Context())
	if orgID == "" {
		api.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CompleteUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.AssetID == "" {
		api.Error(w, http.StatusBadRequest, "asset_id is required")
		return
	}
	if req.StorageKey == "" {
		api.Error(w, http.StatusBadRequest, "storage_key is required")
		return
	}
	if req.Filename == "" {
		api.Error(w, http.StatusBadRequest, "filename is required")
		return
	}
	if req.MimeType == "" {
		api.Error(w, http.StatusBadRequest, "mime_type is required")
		return
	}
	if req.SHA256 == "" {
		api.Error(w, http.StatusBadRequest, "sha256 is required")
		return
	}

	var knowledgeID *string
	if req.KnowledgeID != "" {
		knowledgeID = &req.KnowledgeID
	}

	input := service.CompleteUploadInput{
		AssetID:     req.AssetID,
		OrgID:       orgID,
		ProjectID:   req.ProjectID,
		StorageKey:  req.StorageKey,
		Filename:    req.Filename,
		ContentType: req.MimeType,
		SHA256:      req.SHA256,
		Keywords:    req.Keywords,
		Description: req.Description,
		KnowledgeID: knowledgeID,
	}

	asset, err := h.svc.CompleteUpload(r.Context(), input)
	if err != nil {
		api.HandleError(w, err)
		return
	}

	api.Success(w, http.StatusOK, assetToResponse(asset))
}

func (h *AssetHandler) GetDownloadURL(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		api.Error(w, http.StatusBadRequest, "id is required")
		return
	}

	downloadURL, err := h.svc.GetDownloadURL(r.Context(), id)
	if err != nil {
		api.HandleError(w, err)
		return
	}

	api.Success(w, http.StatusOK, DownloadURLResponse{
		DownloadURL: downloadURL,
	})
}
