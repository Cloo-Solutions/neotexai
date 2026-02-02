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

// MockVFSService for testing Open and List handlers
type MockVFSService struct {
	mock.Mock
}

func (m *MockVFSService) Open(ctx context.Context, input service.OpenInput) (*service.OpenResult, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.OpenResult), args.Error(1)
}

func (m *MockVFSService) List(ctx context.Context, input service.ListInput) (*service.ListOutput, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.ListOutput), args.Error(1)
}

func TestContextHandler_Open_Knowledge(t *testing.T) {
	mockSvc := new(MockContextService)
	mockVFS := new(MockVFSService)
	handler := NewContextHandlerWithVFS(mockSvc, mockVFS, nil)

	expectedResult := &service.OpenResult{
		ID:         "k-123",
		SourceType: "knowledge",
		Title:      "Test Knowledge",
		Content:    "Content here",
		TotalLines: 10,
		TotalChars: 100,
		ChunkCount: 3,
		ChunkIndex: -1,
	}
	mockVFS.On("Open", mock.Anything, mock.MatchedBy(func(input service.OpenInput) bool {
		return input.ID == "k-123" && input.SourceType == "knowledge"
	})).Return(expectedResult, nil)

	body := `{"id":"k-123","source_type":"knowledge"}`
	req := requestWithOrgID(http.MethodPost, "/context/open", []byte(body))
	w := httptest.NewRecorder()

	handler.Open(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "k-123", data["id"])
	assert.Equal(t, "knowledge", data["source_type"])
	assert.Equal(t, "Test Knowledge", data["title"])
	assert.Equal(t, "Content here", data["content"])
	assert.Equal(t, float64(10), data["total_lines"])
	assert.Equal(t, float64(3), data["chunk_count"])
	mockVFS.AssertExpectations(t)
}

func TestContextHandler_Open_Chunk(t *testing.T) {
	mockSvc := new(MockContextService)
	mockVFS := new(MockVFSService)
	handler := NewContextHandlerWithVFS(mockSvc, mockVFS, nil)

	expectedResult := &service.OpenResult{
		ID:         "k-123",
		SourceType: "knowledge",
		Title:      "Chunk Title",
		Content:    "Chunk content",
		ChunkID:    "c-456",
		ChunkIndex: 2,
		ChunkCount: 5,
	}
	mockVFS.On("Open", mock.Anything, mock.MatchedBy(func(input service.OpenInput) bool {
		return input.ID == "k-123" && input.ChunkID == "c-456"
	})).Return(expectedResult, nil)

	body := `{"id":"k-123","chunk_id":"c-456"}`
	req := requestWithOrgID(http.MethodPost, "/context/open", []byte(body))
	w := httptest.NewRecorder()

	handler.Open(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "c-456", data["chunk_id"])
	assert.Equal(t, float64(2), data["chunk_index"])
	assert.Equal(t, float64(5), data["chunk_count"])
}

func TestContextHandler_Open_Asset(t *testing.T) {
	mockSvc := new(MockContextService)
	mockVFS := new(MockVFSService)
	handler := NewContextHandlerWithVFS(mockSvc, mockVFS, nil)

	expectedResult := &service.OpenResult{
		ID:          "a-789",
		SourceType:  "asset",
		Title:       "report.pdf",
		Filename:    "report.pdf",
		MimeType:    "application/pdf",
		Description: "Quarterly report",
		Keywords:    []string{"finance", "quarterly"},
	}
	mockVFS.On("Open", mock.Anything, mock.MatchedBy(func(input service.OpenInput) bool {
		return input.ID == "a-789" && input.SourceType == "asset"
	})).Return(expectedResult, nil)

	body := `{"id":"a-789","source_type":"asset"}`
	req := requestWithOrgID(http.MethodPost, "/context/open", []byte(body))
	w := httptest.NewRecorder()

	handler.Open(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "a-789", data["id"])
	assert.Equal(t, "asset", data["source_type"])
	assert.Equal(t, "report.pdf", data["filename"])
	assert.Equal(t, "application/pdf", data["mime_type"])
}

func TestContextHandler_Open_AssetWithDownloadURL(t *testing.T) {
	mockSvc := new(MockContextService)
	mockVFS := new(MockVFSService)
	handler := NewContextHandlerWithVFS(mockSvc, mockVFS, nil)

	expectedResult := &service.OpenResult{
		ID:          "a-789",
		SourceType:  "asset",
		Filename:    "report.pdf",
		DownloadURL: "https://s3.example.com/presigned-url",
	}
	mockVFS.On("Open", mock.Anything, mock.MatchedBy(func(input service.OpenInput) bool {
		return input.ID == "a-789" && input.IncludeURL == true
	})).Return(expectedResult, nil)

	body := `{"id":"a-789","source_type":"asset","include_url":true}`
	req := requestWithOrgID(http.MethodPost, "/context/open", []byte(body))
	w := httptest.NewRecorder()

	handler.Open(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "https://s3.example.com/presigned-url", data["download_url"])
}

func TestContextHandler_Open_WithLineRange(t *testing.T) {
	mockSvc := new(MockContextService)
	mockVFS := new(MockVFSService)
	handler := NewContextHandlerWithVFS(mockSvc, mockVFS, nil)

	expectedResult := &service.OpenResult{
		ID:         "k-123",
		SourceType: "knowledge",
		Content:    "Lines 10-20",
		TotalLines: 100,
	}
	mockVFS.On("Open", mock.Anything, mock.MatchedBy(func(input service.OpenInput) bool {
		return input.Range != nil && input.Range.StartLine == 10 && input.Range.EndLine == 20
	})).Return(expectedResult, nil)

	body := `{"id":"k-123","range":{"start_line":10,"end_line":20}}`
	req := requestWithOrgID(http.MethodPost, "/context/open", []byte(body))
	w := httptest.NewRecorder()

	handler.Open(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockVFS.AssertExpectations(t)
}

func TestContextHandler_Open_MissingID(t *testing.T) {
	mockSvc := new(MockContextService)
	mockVFS := new(MockVFSService)
	handler := NewContextHandlerWithVFS(mockSvc, mockVFS, nil)

	body := `{"source_type":"knowledge"}`
	req := requestWithOrgID(http.MethodPost, "/context/open", []byte(body))
	w := httptest.NewRecorder()

	handler.Open(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "id is required")
}

func TestContextHandler_Open_Unauthorized(t *testing.T) {
	mockSvc := new(MockContextService)
	mockVFS := new(MockVFSService)
	handler := NewContextHandlerWithVFS(mockSvc, mockVFS, nil)

	req := httptest.NewRequest(http.MethodPost, "/context/open", nil)
	w := httptest.NewRecorder()

	handler.Open(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestContextHandler_Open_VFSNotConfigured(t *testing.T) {
	mockSvc := new(MockContextService)
	// No VFS service
	handler := NewContextHandler(mockSvc, nil)

	body := `{"id":"k-123"}`
	req := requestWithOrgID(http.MethodPost, "/context/open", []byte(body))
	w := httptest.NewRecorder()

	handler.Open(w, req)

	assert.Equal(t, http.StatusNotImplemented, w.Code)
	assert.Contains(t, w.Body.String(), "open not available")
}

func TestContextHandler_List_Success(t *testing.T) {
	mockSvc := new(MockContextService)
	mockVFS := new(MockVFSService)
	handler := NewContextHandlerWithVFS(mockSvc, mockVFS, nil)

	expectedOutput := &service.ListOutput{
		Items: []*service.ListItem{
			{
				ID:         "k-1",
				Title:      "Knowledge 1",
				Scope:      "/docs",
				Type:       domain.KnowledgeTypeGuideline,
				Status:     domain.KnowledgeStatusApproved,
				SourceType: "knowledge",
				ChunkCount: 3,
			},
			{
				ID:         "a-1",
				Title:      "asset.pdf",
				SourceType: "asset",
				Filename:   "asset.pdf",
				MimeType:   "application/pdf",
			},
		},
		HasMore: false,
	}
	mockVFS.On("List", mock.Anything, mock.MatchedBy(func(input service.ListInput) bool {
		return input.OrgID == "org-456"
	})).Return(expectedOutput, nil)

	body := `{}`
	req := requestWithOrgID(http.MethodPost, "/context/list", []byte(body))
	w := httptest.NewRecorder()

	handler.List(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	data := resp["data"].(map[string]interface{})
	items := data["items"].([]interface{})
	assert.Len(t, items, 2)
	mockVFS.AssertExpectations(t)
}

func TestContextHandler_List_WithFilters(t *testing.T) {
	mockSvc := new(MockContextService)
	mockVFS := new(MockVFSService)
	handler := NewContextHandlerWithVFS(mockSvc, mockVFS, nil)

	mockVFS.On("List", mock.Anything, mock.MatchedBy(func(input service.ListInput) bool {
		return input.PathPrefix == "/docs" &&
			input.Type == domain.KnowledgeTypeGuideline &&
			input.SourceType == "knowledge" &&
			input.Limit == 25
	})).Return(&service.ListOutput{Items: []*service.ListItem{}}, nil)

	body := `{"path_prefix":"/docs","type":"guideline","source_type":"knowledge","limit":25}`
	req := requestWithOrgID(http.MethodPost, "/context/list", []byte(body))
	w := httptest.NewRecorder()

	handler.List(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockVFS.AssertExpectations(t)
}

func TestContextHandler_List_WithPagination(t *testing.T) {
	mockSvc := new(MockContextService)
	mockVFS := new(MockVFSService)
	handler := NewContextHandlerWithVFS(mockSvc, mockVFS, nil)

	expectedOutput := &service.ListOutput{
		Items: []*service.ListItem{
			{ID: "k-3", Title: "Knowledge 3", SourceType: "knowledge"},
		},
		Cursor:  "next-cursor",
		HasMore: true,
	}
	mockVFS.On("List", mock.Anything, mock.MatchedBy(func(input service.ListInput) bool {
		return input.Cursor == "previous-cursor"
	})).Return(expectedOutput, nil)

	body := `{"cursor":"previous-cursor"}`
	req := requestWithOrgID(http.MethodPost, "/context/list", []byte(body))
	w := httptest.NewRecorder()

	handler.List(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "next-cursor", data["cursor"])
	assert.Equal(t, true, data["has_more"])
}

func TestContextHandler_List_Unauthorized(t *testing.T) {
	mockSvc := new(MockContextService)
	mockVFS := new(MockVFSService)
	handler := NewContextHandlerWithVFS(mockSvc, mockVFS, nil)

	req := httptest.NewRequest(http.MethodPost, "/context/list", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestContextHandler_List_VFSNotConfigured(t *testing.T) {
	mockSvc := new(MockContextService)
	// No VFS service
	handler := NewContextHandler(mockSvc, nil)

	body := `{}`
	req := requestWithOrgID(http.MethodPost, "/context/list", []byte(body))
	w := httptest.NewRecorder()

	handler.List(w, req)

	assert.Equal(t, http.StatusNotImplemented, w.Code)
	assert.Contains(t, w.Body.String(), "list not available")
}
