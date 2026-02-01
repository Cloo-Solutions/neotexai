package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloo-solutions/neotexai/internal/api/middleware"
	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/cloo-solutions/neotexai/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockKnowledgeService struct {
	mock.Mock
}

func (m *MockKnowledgeService) Create(ctx context.Context, input service.CreateInput) (*domain.Knowledge, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Knowledge), args.Error(1)
}

func (m *MockKnowledgeService) GetByID(ctx context.Context, id string) (*domain.Knowledge, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Knowledge), args.Error(1)
}

func (m *MockKnowledgeService) Update(ctx context.Context, input service.UpdateInput) (*domain.Knowledge, *domain.KnowledgeVersion, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).(*domain.Knowledge), args.Get(1).(*domain.KnowledgeVersion), args.Error(2)
}

func (m *MockKnowledgeService) Deprecate(ctx context.Context, knowledgeID string) (*domain.Knowledge, error) {
	args := m.Called(ctx, knowledgeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Knowledge), args.Error(1)
}

func (m *MockKnowledgeService) ListByOrg(ctx context.Context, orgID string) ([]*domain.Knowledge, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Knowledge), args.Error(1)
}

func (m *MockKnowledgeService) ListByProject(ctx context.Context, projectID string) ([]*domain.Knowledge, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Knowledge), args.Error(1)
}

func (m *MockKnowledgeService) ListKnowledge(ctx context.Context, input service.ListKnowledgeInput) (*service.ListKnowledgeOutput, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.ListKnowledgeOutput), args.Error(1)
}

func newTestKnowledge() *domain.Knowledge {
	now := time.Now().UTC()
	return &domain.Knowledge{
		ID:        "k-123",
		OrgID:     "org-456",
		ProjectID: "proj-789",
		Type:      domain.KnowledgeTypeGuideline,
		Status:    domain.KnowledgeStatusDraft,
		Title:     "Test Knowledge",
		Summary:   "A test summary",
		BodyMD:    "# Test\nBody content",
		Scope:     "src/",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func requestWithOrgID(method, url string, body []byte) *http.Request {
	req := httptest.NewRequest(method, url, bytes.NewReader(body))
	ctx := context.WithValue(req.Context(), middleware.OrgIDKey, "org-456")
	return req.WithContext(ctx)
}

func TestKnowledgeHandler_Create_Success(t *testing.T) {
	mockSvc := new(MockKnowledgeService)
	handler := NewKnowledgeHandler(mockSvc)

	expectedKnowledge := newTestKnowledge()
	mockSvc.On("Create", mock.Anything, mock.MatchedBy(func(input service.CreateInput) bool {
		return input.OrgID == "org-456" && input.Title == "Test Knowledge"
	})).Return(expectedKnowledge, nil)

	body := `{"type":"guideline","title":"Test Knowledge","summary":"A test summary","body_md":"# Test\nBody content","project_id":"proj-789","scope":"src/"}`
	req := requestWithOrgID(http.MethodPost, "/knowledge", []byte(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Create(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "k-123", data["id"])
	mockSvc.AssertExpectations(t)
}

func TestKnowledgeHandler_Create_Unauthorized(t *testing.T) {
	mockSvc := new(MockKnowledgeService)
	handler := NewKnowledgeHandler(mockSvc)

	body := `{"type":"guideline","title":"Test Knowledge","body_md":"# Test"}`
	req := httptest.NewRequest(http.MethodPost, "/knowledge", bytes.NewReader([]byte(body)))
	w := httptest.NewRecorder()

	handler.Create(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestKnowledgeHandler_Create_InvalidJSON(t *testing.T) {
	mockSvc := new(MockKnowledgeService)
	handler := NewKnowledgeHandler(mockSvc)

	req := requestWithOrgID(http.MethodPost, "/knowledge", []byte(`{invalid`))
	w := httptest.NewRecorder()

	handler.Create(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request body")
}

func TestKnowledgeHandler_Create_MissingTitle(t *testing.T) {
	mockSvc := new(MockKnowledgeService)
	handler := NewKnowledgeHandler(mockSvc)

	body := `{"type":"guideline","body_md":"# Test"}`
	req := requestWithOrgID(http.MethodPost, "/knowledge", []byte(body))
	w := httptest.NewRecorder()

	handler.Create(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "title is required")
}

func TestKnowledgeHandler_Create_InvalidType(t *testing.T) {
	mockSvc := new(MockKnowledgeService)
	handler := NewKnowledgeHandler(mockSvc)

	body := `{"type":"invalid","title":"Test","body_md":"# Test"}`
	req := requestWithOrgID(http.MethodPost, "/knowledge", []byte(body))
	w := httptest.NewRecorder()

	handler.Create(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid knowledge type")
}

func TestKnowledgeHandler_Get_Success(t *testing.T) {
	mockSvc := new(MockKnowledgeService)
	handler := NewKnowledgeHandler(mockSvc)

	expectedKnowledge := newTestKnowledge()
	mockSvc.On("GetByID", mock.Anything, "k-123").Return(expectedKnowledge, nil)

	req := httptest.NewRequest(http.MethodGet, "/knowledge/k-123", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "k-123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.Get(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestKnowledgeHandler_Get_NotFound(t *testing.T) {
	mockSvc := new(MockKnowledgeService)
	handler := NewKnowledgeHandler(mockSvc)

	mockSvc.On("GetByID", mock.Anything, "k-999").Return(nil, domain.ErrKnowledgeNotFound)

	req := httptest.NewRequest(http.MethodGet, "/knowledge/k-999", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "k-999")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.Get(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestKnowledgeHandler_Update_Success(t *testing.T) {
	mockSvc := new(MockKnowledgeService)
	handler := NewKnowledgeHandler(mockSvc)

	expectedKnowledge := newTestKnowledge()
	expectedVersion := &domain.KnowledgeVersion{ID: "v-1", KnowledgeID: "k-123", VersionNumber: 2}
	mockSvc.On("Update", mock.Anything, mock.MatchedBy(func(input service.UpdateInput) bool {
		return input.KnowledgeID == "k-123" && input.Title == "Updated Title"
	})).Return(expectedKnowledge, expectedVersion, nil)

	body := `{"title":"Updated Title","summary":"Updated summary","body_md":"# Updated"}`
	req := requestWithOrgID(http.MethodPut, "/knowledge/k-123", []byte(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "k-123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.Update(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestKnowledgeHandler_Delete_Success(t *testing.T) {
	mockSvc := new(MockKnowledgeService)
	handler := NewKnowledgeHandler(mockSvc)

	expectedKnowledge := newTestKnowledge()
	expectedKnowledge.Status = domain.KnowledgeStatusDeprecated
	mockSvc.On("Deprecate", mock.Anything, "k-123").Return(expectedKnowledge, nil)

	req := requestWithOrgID(http.MethodDelete, "/knowledge/k-123", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "k-123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.Delete(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "deprecated", data["status"])
	mockSvc.AssertExpectations(t)
}

func TestKnowledgeHandler_List_ByOrg(t *testing.T) {
	mockSvc := new(MockKnowledgeService)
	handler := NewKnowledgeHandler(mockSvc)

	expectedOutput := &service.ListKnowledgeOutput{
		Items:   []*domain.Knowledge{newTestKnowledge()},
		HasMore: false,
	}
	mockSvc.On("ListKnowledge", mock.Anything, mock.MatchedBy(func(input service.ListKnowledgeInput) bool {
		return input.OrgID == "org-456" && input.ProjectID == ""
	})).Return(expectedOutput, nil)

	req := requestWithOrgID(http.MethodGet, "/knowledge", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestKnowledgeHandler_List_ByProject(t *testing.T) {
	mockSvc := new(MockKnowledgeService)
	handler := NewKnowledgeHandler(mockSvc)

	expectedOutput := &service.ListKnowledgeOutput{
		Items:   []*domain.Knowledge{newTestKnowledge()},
		HasMore: false,
	}
	mockSvc.On("ListKnowledge", mock.Anything, mock.MatchedBy(func(input service.ListKnowledgeInput) bool {
		return input.OrgID == "org-456" && input.ProjectID == "proj-789"
	})).Return(expectedOutput, nil)

	req := requestWithOrgID(http.MethodGet, "/knowledge?project_id=proj-789", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}
