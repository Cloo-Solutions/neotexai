package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/cloo-solutions/neotexai/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockAssetService struct {
	mock.Mock
}

func (m *MockAssetService) InitUpload(ctx context.Context, input service.InitUploadInput) (*service.InitUploadResult, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.InitUploadResult), args.Error(1)
}

func (m *MockAssetService) CompleteUpload(ctx context.Context, input service.CompleteUploadInput) (*domain.Asset, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Asset), args.Error(1)
}

func (m *MockAssetService) GetDownloadURL(ctx context.Context, assetID string) (string, error) {
	args := m.Called(ctx, assetID)
	return args.String(0), args.Error(1)
}

func (m *MockAssetService) GetByID(ctx context.Context, assetID string) (*domain.Asset, error) {
	args := m.Called(ctx, assetID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Asset), args.Error(1)
}

func newTestAsset() *domain.Asset {
	return &domain.Asset{
		ID:          "asset-123",
		OrgID:       "org-456",
		ProjectID:   "proj-789",
		Filename:    "document.pdf",
		MimeType:    "application/pdf",
		SHA256:      "abc123hash",
		StorageKey:  "org-456/asset-123/document.pdf",
		Keywords:    []string{"doc", "test"},
		Description: "A test document",
		CreatedAt:   time.Now().UTC(),
	}
}

func TestAssetHandler_InitUpload_Success(t *testing.T) {
	mockSvc := new(MockAssetService)
	handler := NewAssetHandler(mockSvc)

	expectedResult := &service.InitUploadResult{
		AssetID:    "asset-123",
		StorageKey: "org-456/asset-123/document.pdf",
		UploadURL:  "https://storage.example.com/upload",
	}
	mockSvc.On("InitUpload", mock.Anything, mock.MatchedBy(func(input service.InitUploadInput) bool {
		return input.OrgID == "org-456" && input.Filename == "document.pdf"
	})).Return(expectedResult, nil)

	body := `{"filename":"document.pdf","mime_type":"application/pdf","size_bytes":1024,"project_id":"proj-789"}`
	req := requestWithOrgID(http.MethodPost, "/assets/init", []byte(body))
	w := httptest.NewRecorder()

	handler.InitUpload(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "asset-123", data["asset_id"])
	assert.Equal(t, "https://storage.example.com/upload", data["upload_url"])
	mockSvc.AssertExpectations(t)
}

func TestAssetHandler_InitUpload_Unauthorized(t *testing.T) {
	mockSvc := new(MockAssetService)
	handler := NewAssetHandler(mockSvc)

	body := `{"filename":"document.pdf","mime_type":"application/pdf"}`
	req := httptest.NewRequest(http.MethodPost, "/assets/init", bytes.NewReader([]byte(body)))
	w := httptest.NewRecorder()

	handler.InitUpload(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAssetHandler_InitUpload_MissingFilename(t *testing.T) {
	mockSvc := new(MockAssetService)
	handler := NewAssetHandler(mockSvc)

	body := `{"mime_type":"application/pdf"}`
	req := requestWithOrgID(http.MethodPost, "/assets/init", []byte(body))
	w := httptest.NewRecorder()

	handler.InitUpload(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "filename is required")
}

func TestAssetHandler_CompleteUpload_Success(t *testing.T) {
	mockSvc := new(MockAssetService)
	handler := NewAssetHandler(mockSvc)

	expectedAsset := newTestAsset()
	mockSvc.On("CompleteUpload", mock.Anything, mock.MatchedBy(func(input service.CompleteUploadInput) bool {
		return input.AssetID == "asset-123" &&
			input.StorageKey == "org-456/asset-123/document.pdf" &&
			input.Filename == "document.pdf" &&
			input.ContentType == "application/pdf" &&
			input.SHA256 == "abc123hash"
	})).Return(expectedAsset, nil)

	body := `{"asset_id":"asset-123","storage_key":"org-456/asset-123/document.pdf","filename":"document.pdf","mime_type":"application/pdf","sha256":"abc123hash"}`
	req := requestWithOrgID(http.MethodPost, "/assets/complete", []byte(body))
	w := httptest.NewRecorder()

	handler.CompleteUpload(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "asset-123", data["id"])
	mockSvc.AssertExpectations(t)
}

func TestAssetHandler_CompleteUpload_MissingAssetID(t *testing.T) {
	mockSvc := new(MockAssetService)
	handler := NewAssetHandler(mockSvc)

	body := `{"sha256":"abc123hash"}`
	req := requestWithOrgID(http.MethodPost, "/assets/complete", []byte(body))
	w := httptest.NewRecorder()

	handler.CompleteUpload(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "asset_id is required")
}

func TestAssetHandler_CompleteUpload_WithKnowledgeLink(t *testing.T) {
	mockSvc := new(MockAssetService)
	handler := NewAssetHandler(mockSvc)

	expectedAsset := newTestAsset()
	mockSvc.On("CompleteUpload", mock.Anything, mock.MatchedBy(func(input service.CompleteUploadInput) bool {
		return input.AssetID == "asset-123" &&
			input.StorageKey == "org-456/asset-123/document.pdf" &&
			input.KnowledgeID != nil && *input.KnowledgeID == "k-456"
	})).Return(expectedAsset, nil)

	body := `{"asset_id":"asset-123","storage_key":"org-456/asset-123/document.pdf","filename":"document.pdf","mime_type":"application/pdf","sha256":"abc123hash","knowledge_id":"k-456"}`
	req := requestWithOrgID(http.MethodPost, "/assets/complete", []byte(body))
	w := httptest.NewRecorder()

	handler.CompleteUpload(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestAssetHandler_GetDownloadURL_Success(t *testing.T) {
	mockSvc := new(MockAssetService)
	handler := NewAssetHandler(mockSvc)

	mockSvc.On("GetDownloadURL", mock.Anything, "asset-123").Return("https://storage.example.com/download", nil)

	req := httptest.NewRequest(http.MethodGet, "/assets/asset-123/download", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "asset-123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.GetDownloadURL(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "https://storage.example.com/download", data["download_url"])
	mockSvc.AssertExpectations(t)
}

func TestAssetHandler_GetDownloadURL_NotFound(t *testing.T) {
	mockSvc := new(MockAssetService)
	handler := NewAssetHandler(mockSvc)

	mockSvc.On("GetDownloadURL", mock.Anything, "asset-999").Return("", domain.ErrAssetNotFound)

	req := httptest.NewRequest(http.MethodGet, "/assets/asset-999/download", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "asset-999")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.GetDownloadURL(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}
