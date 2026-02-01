package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockAuthValidator struct {
	mock.Mock
}

func (m *MockAuthValidator) ValidateAPIKey(ctx context.Context, token string) (string, error) {
	args := m.Called(ctx, token)
	return args.String(0), args.Error(1)
}

func TestAPIKeyAuth_Success(t *testing.T) {
	mockValidator := new(MockAuthValidator)
	mockValidator.On("ValidateAPIKey", mock.Anything, "ntx_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef").Return("org-789", nil)

	var capturedOrgID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedOrgID = GetOrgID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	middleware := APIKeyAuth(mockValidator)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer ntx_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "org-789", capturedOrgID)
	mockValidator.AssertExpectations(t)
}

func TestAPIKeyAuth_MissingHeader(t *testing.T) {
	mockValidator := new(MockAuthValidator)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	middleware := APIKeyAuth(mockValidator)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "missing authorization header")
}

func TestAPIKeyAuth_InvalidFormat(t *testing.T) {
	mockValidator := new(MockAuthValidator)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	middleware := APIKeyAuth(mockValidator)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic abc123")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid authorization format")
}

func TestAPIKeyAuth_ValidationFails(t *testing.T) {
	mockValidator := new(MockAuthValidator)
	mockValidator.On("ValidateAPIKey", mock.Anything, "ntx_badtoken0123456789abcdef0123456789abcdef0123456789abcdef01234").Return("", errors.New("invalid key"))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	middleware := APIKeyAuth(mockValidator)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer ntx_badtoken0123456789abcdef0123456789abcdef0123456789abcdef01234")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid api key")
	mockValidator.AssertExpectations(t)
}

func TestGetOrgID_ValidContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), OrgIDKey, "org-123")
	orgID := GetOrgID(ctx)
	assert.Equal(t, "org-123", orgID)
}

func TestGetOrgID_MissingContext(t *testing.T) {
	ctx := context.Background()
	orgID := GetOrgID(ctx)
	assert.Equal(t, "", orgID)
}
