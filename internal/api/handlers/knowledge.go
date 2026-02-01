package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/cloo-solutions/neotexai/internal/api"
	"github.com/cloo-solutions/neotexai/internal/api/middleware"
	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/cloo-solutions/neotexai/internal/service"
	"github.com/go-chi/chi/v5"
)

type KnowledgeService interface {
	Create(ctx context.Context, input service.CreateInput) (*domain.Knowledge, error)
	GetByID(ctx context.Context, id string) (*domain.Knowledge, error)
	Update(ctx context.Context, input service.UpdateInput) (*domain.Knowledge, *domain.KnowledgeVersion, error)
	Deprecate(ctx context.Context, knowledgeID string) (*domain.Knowledge, error)
	ListByOrg(ctx context.Context, orgID string) ([]*domain.Knowledge, error)
	ListByProject(ctx context.Context, projectID string) ([]*domain.Knowledge, error)
	ListKnowledge(ctx context.Context, input service.ListKnowledgeInput) (*service.ListKnowledgeOutput, error)
}

type KnowledgeHandler struct {
	svc KnowledgeService
}

func NewKnowledgeHandler(svc KnowledgeService) *KnowledgeHandler {
	return &KnowledgeHandler{svc: svc}
}

type CreateKnowledgeRequest struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Summary   string `json:"summary"`
	BodyMD    string `json:"body_md"`
	ProjectID string `json:"project_id"`
	Scope     string `json:"scope"`
}

type UpdateKnowledgeRequest struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	BodyMD  string `json:"body_md"`
	Scope   string `json:"scope"`
}

type KnowledgeResponse struct {
	ID        string `json:"id"`
	OrgID     string `json:"org_id"`
	ProjectID string `json:"project_id"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	Title     string `json:"title"`
	Summary   string `json:"summary"`
	BodyMD    string `json:"body_md"`
	Scope     string `json:"scope"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func knowledgeToResponse(k *domain.Knowledge) *KnowledgeResponse {
	return &KnowledgeResponse{
		ID:        k.ID,
		OrgID:     k.OrgID,
		ProjectID: k.ProjectID,
		Type:      string(k.Type),
		Status:    string(k.Status),
		Title:     k.Title,
		Summary:   k.Summary,
		BodyMD:    k.BodyMD,
		Scope:     k.Scope,
		CreatedAt: k.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: k.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func (h *KnowledgeHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgID(r.Context())
	if orgID == "" {
		api.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateKnowledgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" {
		api.Error(w, http.StatusBadRequest, "title is required")
		return
	}
	if req.BodyMD == "" {
		api.Error(w, http.StatusBadRequest, "body_md is required")
		return
	}
	if req.Type == "" {
		api.Error(w, http.StatusBadRequest, "type is required")
		return
	}

	knowledgeType := domain.KnowledgeType(req.Type)
	if !isValidKnowledgeType(knowledgeType) {
		api.Error(w, http.StatusBadRequest, "invalid knowledge type")
		return
	}

	input := service.CreateInput{
		OrgID:     orgID,
		ProjectID: req.ProjectID,
		Type:      knowledgeType,
		Title:     req.Title,
		Summary:   req.Summary,
		BodyMD:    req.BodyMD,
		Scope:     req.Scope,
	}

	knowledge, err := h.svc.Create(r.Context(), input)
	if err != nil {
		api.HandleError(w, err)
		return
	}

	api.Success(w, http.StatusCreated, knowledgeToResponse(knowledge))
}

func (h *KnowledgeHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		api.Error(w, http.StatusBadRequest, "id is required")
		return
	}

	knowledge, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		api.HandleError(w, err)
		return
	}

	api.Success(w, http.StatusOK, knowledgeToResponse(knowledge))
}

func (h *KnowledgeHandler) Update(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgID(r.Context())
	if orgID == "" {
		api.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		api.Error(w, http.StatusBadRequest, "id is required")
		return
	}

	var req UpdateKnowledgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" {
		api.Error(w, http.StatusBadRequest, "title is required")
		return
	}
	if req.BodyMD == "" {
		api.Error(w, http.StatusBadRequest, "body_md is required")
		return
	}

	input := service.UpdateInput{
		KnowledgeID: id,
		Title:       req.Title,
		Summary:     req.Summary,
		BodyMD:      req.BodyMD,
		Scope:       req.Scope,
	}

	knowledge, _, err := h.svc.Update(r.Context(), input)
	if err != nil {
		api.HandleError(w, err)
		return
	}

	api.Success(w, http.StatusOK, knowledgeToResponse(knowledge))
}

func (h *KnowledgeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgID(r.Context())
	if orgID == "" {
		api.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		api.Error(w, http.StatusBadRequest, "id is required")
		return
	}

	knowledge, err := h.svc.Deprecate(r.Context(), id)
	if err != nil {
		api.HandleError(w, err)
		return
	}

	api.Success(w, http.StatusOK, knowledgeToResponse(knowledge))
}

type KnowledgeListResponse struct {
	Items   []*KnowledgeResponse `json:"items"`
	Cursor  string               `json:"cursor,omitempty"`
	HasMore bool                 `json:"has_more"`
}

func (h *KnowledgeHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgID(r.Context())
	if orgID == "" {
		api.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	projectID := r.URL.Query().Get("project_id")
	cursor := r.URL.Query().Get("cursor")
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	input := service.ListKnowledgeInput{
		OrgID:     orgID,
		ProjectID: projectID,
		Cursor:    cursor,
		Limit:     limit,
	}

	output, err := h.svc.ListKnowledge(r.Context(), input)
	if err != nil {
		api.HandleError(w, err)
		return
	}

	responses := make([]*KnowledgeResponse, len(output.Items))
	for i, k := range output.Items {
		responses[i] = knowledgeToResponse(k)
	}

	api.Success(w, http.StatusOK, KnowledgeListResponse{
		Items:   responses,
		Cursor:  output.Cursor,
		HasMore: output.HasMore,
	})
}

func isValidKnowledgeType(t domain.KnowledgeType) bool {
	switch t {
	case domain.KnowledgeTypeGuideline, domain.KnowledgeTypeLearning, domain.KnowledgeTypeDecision,
		domain.KnowledgeTypeTemplate, domain.KnowledgeTypeChecklist, domain.KnowledgeTypeSnippet:
		return true
	}
	return false
}
