package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/cloo-solutions/neotexai/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockContextService struct {
	mock.Mock
}

func (m *MockContextService) GetManifest(ctx context.Context, orgID, projectID string) ([]*service.KnowledgeManifestItem, error) {
	args := m.Called(ctx, orgID, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*service.KnowledgeManifestItem), args.Error(1)
}

func (m *MockContextService) Search(ctx context.Context, input service.SearchInput) (*service.SearchOutput, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.SearchOutput), args.Error(1)
}

func TestContextHandler_GetManifest_Success(t *testing.T) {
	mockSvc := new(MockContextService)
	handler := NewContextHandler(mockSvc, nil)

	expectedItems := []*service.KnowledgeManifestItem{
		{ID: "k-1", Title: "Guideline 1", Summary: "Summary 1", Type: domain.KnowledgeTypeGuideline, Scope: "src/"},
		{ID: "k-2", Title: "Learning 1", Summary: "Summary 2", Type: domain.KnowledgeTypeLearning, Scope: ""},
	}
	mockSvc.On("GetManifest", mock.Anything, "org-456", "").Return(expectedItems, nil)

	req := requestWithOrgID(http.MethodGet, "/context", nil)
	w := httptest.NewRecorder()

	handler.GetManifest(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	data := resp["data"].(map[string]interface{})
	manifest := data["manifest"].([]interface{})
	assert.Len(t, manifest, 2)
	mockSvc.AssertExpectations(t)
}

func TestContextHandler_GetManifest_WithProjectID(t *testing.T) {
	mockSvc := new(MockContextService)
	handler := NewContextHandler(mockSvc, nil)

	expectedItems := []*service.KnowledgeManifestItem{
		{ID: "k-1", Title: "Guideline 1", Summary: "Summary 1", Type: domain.KnowledgeTypeGuideline},
	}
	mockSvc.On("GetManifest", mock.Anything, "org-456", "proj-789").Return(expectedItems, nil)

	req := requestWithOrgID(http.MethodGet, "/context?project_id=proj-789", nil)
	w := httptest.NewRecorder()

	handler.GetManifest(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestContextHandler_GetManifest_Unauthorized(t *testing.T) {
	mockSvc := new(MockContextService)
	handler := NewContextHandler(mockSvc, nil)

	req := httptest.NewRequest(http.MethodGet, "/context", nil)
	w := httptest.NewRecorder()

	handler.GetManifest(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestContextHandler_Search_Success(t *testing.T) {
	mockSvc := new(MockContextService)
	handler := NewContextHandler(mockSvc, nil)

	expectedOutput := &service.SearchOutput{
		Results: []*service.SearchResult{
			{ID: "k-1", Title: "Result 1", Summary: "Summary 1", Score: 0.95},
			{ID: "k-2", Title: "Result 2", Summary: "Summary 2", Score: 0.85},
		},
		HasMore: false,
	}
	mockSvc.On("Search", mock.Anything, mock.MatchedBy(func(input service.SearchInput) bool {
		return input.Query == "test query" && input.Filters.OrgID == "org-456"
	})).Return(expectedOutput, nil)

	body := `{"query":"test query","project_id":"proj-789"}`
	req := requestWithOrgID(http.MethodPost, "/search", []byte(body))
	w := httptest.NewRecorder()

	handler.Search(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	data := resp["data"].(map[string]interface{})
	results := data["results"].([]interface{})
	assert.Len(t, results, 2)
	mockSvc.AssertExpectations(t)
}

func TestContextHandler_Search_WithTypeFilter(t *testing.T) {
	mockSvc := new(MockContextService)
	handler := NewContextHandler(mockSvc, nil)

	expectedOutput := &service.SearchOutput{
		Results: []*service.SearchResult{
			{ID: "k-1", Title: "Guideline", Score: 0.90},
		},
		HasMore: false,
	}
	mockSvc.On("Search", mock.Anything, mock.MatchedBy(func(input service.SearchInput) bool {
		return input.Filters.Type == domain.KnowledgeTypeGuideline
	})).Return(expectedOutput, nil)

	body := `{"query":"test","type":"guideline"}`
	req := requestWithOrgID(http.MethodPost, "/search", []byte(body))
	w := httptest.NewRecorder()

	handler.Search(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestContextHandler_Search_CustomLimit(t *testing.T) {
	mockSvc := new(MockContextService)
	handler := NewContextHandler(mockSvc, nil)

	mockSvc.On("Search", mock.Anything, mock.MatchedBy(func(input service.SearchInput) bool {
		return input.Limit == 5
	})).Return(&service.SearchOutput{Results: []*service.SearchResult{}, HasMore: false}, nil)

	body := `{"query":"test","limit":5}`
	req := requestWithOrgID(http.MethodPost, "/search", []byte(body))
	w := httptest.NewRecorder()

	handler.Search(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestContextHandler_Search_MissingQuery(t *testing.T) {
	mockSvc := new(MockContextService)
	handler := NewContextHandler(mockSvc, nil)

	body := `{"project_id":"proj-789"}`
	req := requestWithOrgID(http.MethodPost, "/search", []byte(body))
	w := httptest.NewRecorder()

	handler.Search(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "query is required")
}

func TestContextHandler_Search_Unauthorized(t *testing.T) {
	mockSvc := new(MockContextService)
	handler := NewContextHandler(mockSvc, nil)

	req := httptest.NewRequest(http.MethodPost, "/search", nil)
	w := httptest.NewRecorder()

	handler.Search(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
