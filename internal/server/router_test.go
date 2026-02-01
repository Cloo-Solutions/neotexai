package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloo-solutions/neotexai/internal/api/handlers"
	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/cloo-solutions/neotexai/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockAuthValidator struct {
	mock.Mock
}

func (m *MockAuthValidator) ValidateAPIKey(ctx context.Context, token string) (string, error) {
	args := m.Called(ctx, token)
	return args.String(0), args.Error(1)
}

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

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) CreateOrg(ctx context.Context, name string) (*domain.Organization, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Organization), args.Error(1)
}

func (m *MockAuthService) CreateAPIKey(ctx context.Context, orgID, name string) (string, error) {
	args := m.Called(ctx, orgID, name)
	return args.String(0), args.Error(1)
}

func setupRouter() (http.Handler, *MockAuthValidator, *MockKnowledgeService, *MockAssetService, *MockContextService, *MockAuthService) {
	authValidator := new(MockAuthValidator)
	knowledgeSvc := new(MockKnowledgeService)
	assetSvc := new(MockAssetService)
	contextSvc := new(MockContextService)
	authSvc := new(MockAuthService)

	cfg := RouterConfig{
		AuthValidator:    authValidator,
		KnowledgeHandler: handlers.NewKnowledgeHandler(knowledgeSvc),
		AssetHandler:     handlers.NewAssetHandler(assetSvc),
		ContextHandler:   handlers.NewContextHandler(contextSvc),
		AuthHandler:      handlers.NewAuthHandler(authSvc),
	}

	router := NewRouter(cfg)
	return router, authValidator, knowledgeSvc, assetSvc, contextSvc, authSvc
}

func TestRouter_HealthEndpoint(t *testing.T) {
	router, _, _, _, _, _ := setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "ok", data["status"])
}

func TestRouter_AuthenticatedRoutes_RequireAuth(t *testing.T) {
	router, authValidator, _, _, _, _ := setupRouter()

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/knowledge"},
		{http.MethodGet, "/knowledge/123"},
		{http.MethodPost, "/knowledge"},
		{http.MethodPut, "/knowledge/123"},
		{http.MethodDelete, "/knowledge/123"},
		{http.MethodPost, "/assets/init"},
		{http.MethodPost, "/assets/complete"},
		{http.MethodGet, "/assets/123/download"},
		{http.MethodGet, "/context"},
		{http.MethodPost, "/search"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}

	authValidator.AssertExpectations(t)
}

func TestRouter_AuthenticatedRoutes_WithValidAuth(t *testing.T) {
	router, authValidator, knowledgeSvc, _, _, _ := setupRouter()

	authValidator.On("ValidateAPIKey", mock.Anything, "ntx_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef").Return("org-789", nil)

	expectedKnowledge := &domain.Knowledge{
		ID:        "k-123",
		OrgID:     "org-789",
		ProjectID: "proj-1",
		Type:      domain.KnowledgeTypeGuideline,
		Status:    domain.KnowledgeStatusDraft,
		Title:     "Test",
		Summary:   "Summary",
		BodyMD:    "Body",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	knowledgeSvc.On("GetByID", mock.Anything, "k-123").Return(expectedKnowledge, nil)

	req := httptest.NewRequest(http.MethodGet, "/knowledge/k-123", nil)
	req.Header.Set("Authorization", "Bearer ntx_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	authValidator.AssertExpectations(t)
	knowledgeSvc.AssertExpectations(t)
}

func TestRouter_InternalRoutes_NoAuthRequired(t *testing.T) {
	router, _, _, _, _, authSvc := setupRouter()

	expectedOrg := &domain.Organization{
		ID:        "org-123",
		Name:      "Test Org",
		CreatedAt: time.Now().UTC(),
	}
	authSvc.On("CreateOrg", mock.Anything, "Test Org").Return(expectedOrg, nil)

	req := httptest.NewRequest(http.MethodPost, "/orgs", nil)
	req.Body = httptest.NewRequest(http.MethodPost, "/orgs", nil).Body
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
