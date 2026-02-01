package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

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

func TestAuthHandler_CreateOrg_Success(t *testing.T) {
	mockSvc := new(MockAuthService)
	handler := NewAuthHandler(mockSvc)

	expectedOrg := &domain.Organization{
		ID:        "org-123",
		Name:      "Test Org",
		CreatedAt: time.Now().UTC(),
	}
	mockSvc.On("CreateOrg", mock.Anything, "Test Org").Return(expectedOrg, nil)

	body := `{"name":"Test Org"}`
	req := httptest.NewRequest(http.MethodPost, "/orgs", bytes.NewReader([]byte(body)))
	w := httptest.NewRecorder()

	handler.CreateOrg(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "org-123", data["id"])
	assert.Equal(t, "Test Org", data["name"])
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_CreateOrg_MissingName(t *testing.T) {
	mockSvc := new(MockAuthService)
	handler := NewAuthHandler(mockSvc)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/orgs", bytes.NewReader([]byte(body)))
	w := httptest.NewRecorder()

	handler.CreateOrg(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "name is required")
}

func TestAuthHandler_CreateOrg_InvalidJSON(t *testing.T) {
	mockSvc := new(MockAuthService)
	handler := NewAuthHandler(mockSvc)

	body := `{invalid`
	req := httptest.NewRequest(http.MethodPost, "/orgs", bytes.NewReader([]byte(body)))
	w := httptest.NewRecorder()

	handler.CreateOrg(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request body")
}

func TestAuthHandler_CreateAPIKey_Success(t *testing.T) {
	mockSvc := new(MockAuthService)
	handler := NewAuthHandler(mockSvc)

	mockSvc.On("CreateAPIKey", mock.Anything, "org-123", "dev-key").Return("abc123secret", nil)

	body := `{"org_id":"org-123","name":"dev-key"}`
	req := httptest.NewRequest(http.MethodPost, "/apikeys", bytes.NewReader([]byte(body)))
	w := httptest.NewRecorder()

	handler.CreateAPIKey(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "abc123secret", data["token"])
	assert.Equal(t, "dev-key", data["name"])
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_CreateAPIKey_MissingOrgID(t *testing.T) {
	mockSvc := new(MockAuthService)
	handler := NewAuthHandler(mockSvc)

	body := `{"name":"dev-key"}`
	req := httptest.NewRequest(http.MethodPost, "/apikeys", bytes.NewReader([]byte(body)))
	w := httptest.NewRecorder()

	handler.CreateAPIKey(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "org_id is required")
}

func TestAuthHandler_CreateAPIKey_MissingName(t *testing.T) {
	mockSvc := new(MockAuthService)
	handler := NewAuthHandler(mockSvc)

	body := `{"org_id":"org-123"}`
	req := httptest.NewRequest(http.MethodPost, "/apikeys", bytes.NewReader([]byte(body)))
	w := httptest.NewRecorder()

	handler.CreateAPIKey(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "name is required")
}

func TestAuthHandler_CreateAPIKey_OrgNotFound(t *testing.T) {
	mockSvc := new(MockAuthService)
	handler := NewAuthHandler(mockSvc)

	mockSvc.On("CreateAPIKey", mock.Anything, "org-999", "dev-key").Return("", domain.ErrOrganizationNotFound)

	body := `{"org_id":"org-999","name":"dev-key"}`
	req := httptest.NewRequest(http.MethodPost, "/apikeys", bytes.NewReader([]byte(body)))
	w := httptest.NewRecorder()

	handler.CreateAPIKey(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}
