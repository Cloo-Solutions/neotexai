package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockOrgRepository struct {
	mock.Mock
}

func (m *MockOrgRepository) Create(ctx context.Context, org *domain.Organization) error {
	args := m.Called(ctx, org)
	return args.Error(0)
}

func (m *MockOrgRepository) GetByID(ctx context.Context, id string) (*domain.Organization, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Organization), args.Error(1)
}

func (m *MockOrgRepository) List(ctx context.Context) ([]*domain.Organization, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Organization), args.Error(1)
}

func (m *MockOrgRepository) Update(ctx context.Context, org *domain.Organization) error {
	args := m.Called(ctx, org)
	return args.Error(0)
}

func (m *MockOrgRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOrgRepository) GetByName(ctx context.Context, name string) (*domain.Organization, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Organization), args.Error(1)
}

type MockAPIKeyRepository struct {
	mock.Mock
}

func (m *MockAPIKeyRepository) Create(ctx context.Context, key *domain.APIKey) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockAPIKeyRepository) GetByID(ctx context.Context, id string) (*domain.APIKey, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.APIKey), args.Error(1)
}

func (m *MockAPIKeyRepository) GetByHash(ctx context.Context, hash string) (*domain.APIKey, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.APIKey), args.Error(1)
}

func (m *MockAPIKeyRepository) GetByOrgID(ctx context.Context, orgID string) ([]*domain.APIKey, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.APIKey), args.Error(1)
}

func (m *MockAPIKeyRepository) Revoke(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAPIKeyRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestAuthService_CreateOrg(t *testing.T) {
	ctx := context.Background()
	mockOrgRepo := new(MockOrgRepository)
	mockAPIKeyRepo := new(MockAPIKeyRepository)
	mockUUIDGen := NewMockUUIDGenerator("org-123")

	mockOrgRepo.On("Create", ctx, mock.MatchedBy(func(org *domain.Organization) bool {
		return org.Name == "Test Org" && org.ID == "org-123"
	})).Return(nil)

	service := NewAuthService(mockOrgRepo, mockAPIKeyRepo, mockUUIDGen)
	org, err := service.CreateOrg(ctx, "Test Org")

	require.NoError(t, err)
	assert.Equal(t, "org-123", org.ID)
	assert.Equal(t, "Test Org", org.Name)
	mockOrgRepo.AssertExpectations(t)
}

func TestAuthService_CreateOrg_EmptyName(t *testing.T) {
	ctx := context.Background()
	mockOrgRepo := new(MockOrgRepository)
	mockAPIKeyRepo := new(MockAPIKeyRepository)
	mockUUIDGen := NewMockUUIDGenerator()

	service := NewAuthService(mockOrgRepo, mockAPIKeyRepo, mockUUIDGen)
	_, err := service.CreateOrg(ctx, "")

	assert.Error(t, err)
	mockOrgRepo.AssertNotCalled(t, "Create")
}

func TestAuthService_CreateAPIKey_GeneratesNtxToken(t *testing.T) {
	ctx := context.Background()
	mockOrgRepo := new(MockOrgRepository)
	mockAPIKeyRepo := new(MockAPIKeyRepository)
	mockUUIDGen := NewMockUUIDGenerator("key-123")

	mockOrgRepo.On("GetByID", ctx, "org-123").Return(&domain.Organization{
		ID:        "org-123",
		Name:      "Test Org",
		CreatedAt: time.Now().UTC(),
	}, nil)

	mockAPIKeyRepo.On("Create", ctx, mock.MatchedBy(func(key *domain.APIKey) bool {
		return key.ID == "key-123" && key.KeyHash != "" && len(key.KeyHash) == 64
	})).Return(nil)

	service := NewAuthService(mockOrgRepo, mockAPIKeyRepo, mockUUIDGen)
	token, err := service.CreateAPIKey(ctx, "org-123", "test-key")

	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(token, "ntx_"), "token should start with ntx_")
	assert.Equal(t, 68, len(token), "token should be ntx_ + 64 hex chars")
	mockAPIKeyRepo.AssertExpectations(t)
}

func TestAuthService_CreateAPIKey_StoresSHA256Hash(t *testing.T) {
	ctx := context.Background()
	mockOrgRepo := new(MockOrgRepository)
	mockAPIKeyRepo := new(MockAPIKeyRepository)
	mockUUIDGen := NewMockUUIDGenerator("key-123")

	mockOrgRepo.On("GetByID", ctx, "org-123").Return(&domain.Organization{
		ID:        "org-123",
		Name:      "Test Org",
		CreatedAt: time.Now().UTC(),
	}, nil)

	var capturedKey *domain.APIKey
	mockAPIKeyRepo.On("Create", ctx, mock.MatchedBy(func(key *domain.APIKey) bool {
		capturedKey = key
		return true
	})).Return(nil)

	service := NewAuthService(mockOrgRepo, mockAPIKeyRepo, mockUUIDGen)
	token, err := service.CreateAPIKey(ctx, "org-123", "test-key")

	require.NoError(t, err)
	require.NotNil(t, capturedKey)
	assert.NotEqual(t, token, capturedKey.KeyHash)
	assert.Equal(t, 64, len(capturedKey.KeyHash), "SHA256 hash should be 64 hex chars")
}

func TestAuthService_ValidateAPIKey_ValidToken(t *testing.T) {
	ctx := context.Background()
	mockOrgRepo := new(MockOrgRepository)
	mockAPIKeyRepo := new(MockAPIKeyRepository)
	mockUUIDGen := NewMockUUIDGenerator("key-123")

	mockOrgRepo.On("GetByID", ctx, "org-123").Return(&domain.Organization{
		ID:        "org-123",
		Name:      "Test Org",
		CreatedAt: time.Now().UTC(),
	}, nil)

	var storedHash string
	mockAPIKeyRepo.On("Create", ctx, mock.MatchedBy(func(key *domain.APIKey) bool {
		storedHash = key.KeyHash
		return true
	})).Return(nil)

	service := NewAuthService(mockOrgRepo, mockAPIKeyRepo, mockUUIDGen)
	token, _ := service.CreateAPIKey(ctx, "org-123", "test-key")

	mockAPIKeyRepo.On("GetByHash", ctx, storedHash).Return(&domain.APIKey{
		ID:        "key-123",
		OrgID:     "org-123",
		Name:      "test-key",
		KeyHash:   storedHash,
		CreatedAt: time.Now().UTC(),
		RevokedAt: nil,
	}, nil)

	orgID, err := service.ValidateAPIKey(ctx, token)
	require.NoError(t, err)
	assert.Equal(t, "org-123", orgID)
}

func TestAuthService_ValidateAPIKey_InvalidToken(t *testing.T) {
	ctx := context.Background()
	mockOrgRepo := new(MockOrgRepository)
	mockAPIKeyRepo := new(MockAPIKeyRepository)
	mockUUIDGen := NewMockUUIDGenerator()

	service := NewAuthService(mockOrgRepo, mockAPIKeyRepo, mockUUIDGen)
	_, err := service.ValidateAPIKey(ctx, "invalid-token")

	assert.ErrorIs(t, err, domain.ErrInvalidAPIKey)
}

func TestAuthService_ValidateAPIKey_NotFound(t *testing.T) {
	ctx := context.Background()
	mockOrgRepo := new(MockOrgRepository)
	mockAPIKeyRepo := new(MockAPIKeyRepository)
	mockUUIDGen := NewMockUUIDGenerator()

	mockAPIKeyRepo.On("GetByHash", ctx, mock.Anything).Return(nil, domain.ErrAPIKeyNotFound)

	service := NewAuthService(mockOrgRepo, mockAPIKeyRepo, mockUUIDGen)
	_, err := service.ValidateAPIKey(ctx, "ntx_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")

	assert.ErrorIs(t, err, domain.ErrInvalidAPIKey)
}

func TestAuthService_ValidateAPIKey_RevokedKey(t *testing.T) {
	ctx := context.Background()
	mockOrgRepo := new(MockOrgRepository)
	mockAPIKeyRepo := new(MockAPIKeyRepository)
	mockUUIDGen := NewMockUUIDGenerator()

	revokedAt := time.Now().UTC()
	mockAPIKeyRepo.On("GetByHash", ctx, mock.Anything).Return(&domain.APIKey{
		ID:        "key-123",
		OrgID:     "org-123",
		Name:      "test-key",
		KeyHash:   "somehash",
		CreatedAt: time.Now().UTC(),
		RevokedAt: &revokedAt,
	}, nil)

	service := NewAuthService(mockOrgRepo, mockAPIKeyRepo, mockUUIDGen)
	_, err := service.ValidateAPIKey(ctx, "ntx_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")

	assert.ErrorIs(t, err, domain.ErrAPIKeyRevoked)
}

func TestAuthService_RevokeAPIKey(t *testing.T) {
	ctx := context.Background()
	mockOrgRepo := new(MockOrgRepository)
	mockAPIKeyRepo := new(MockAPIKeyRepository)
	mockUUIDGen := NewMockUUIDGenerator()

	mockAPIKeyRepo.On("Revoke", ctx, "key-123").Return(nil)

	service := NewAuthService(mockOrgRepo, mockAPIKeyRepo, mockUUIDGen)
	err := service.RevokeAPIKey(ctx, "key-123")

	require.NoError(t, err)
	mockAPIKeyRepo.AssertExpectations(t)
}

func TestAuthService_RevokeAPIKey_NotFound(t *testing.T) {
	ctx := context.Background()
	mockOrgRepo := new(MockOrgRepository)
	mockAPIKeyRepo := new(MockAPIKeyRepository)
	mockUUIDGen := NewMockUUIDGenerator()

	mockAPIKeyRepo.On("Revoke", ctx, "key-123").Return(domain.ErrAPIKeyNotFound)

	service := NewAuthService(mockOrgRepo, mockAPIKeyRepo, mockUUIDGen)
	err := service.RevokeAPIKey(ctx, "key-123")

	assert.ErrorIs(t, err, domain.ErrAPIKeyNotFound)
}

func TestAuthService_ListAPIKeys(t *testing.T) {
	ctx := context.Background()
	mockOrgRepo := new(MockOrgRepository)
	mockAPIKeyRepo := new(MockAPIKeyRepository)
	mockUUIDGen := NewMockUUIDGenerator()

	keys := []*domain.APIKey{
		{ID: "key-1", OrgID: "org-123", Name: "key1", KeyHash: "hash1", CreatedAt: time.Now().UTC()},
		{ID: "key-2", OrgID: "org-123", Name: "key2", KeyHash: "hash2", CreatedAt: time.Now().UTC()},
	}

	mockAPIKeyRepo.On("GetByOrgID", ctx, "org-123").Return(keys, nil)

	service := NewAuthService(mockOrgRepo, mockAPIKeyRepo, mockUUIDGen)
	result, err := service.ListAPIKeys(ctx, "org-123")

	require.NoError(t, err)
	assert.Len(t, result, 2)
	mockAPIKeyRepo.AssertExpectations(t)
}

func TestAuthService_CreateAPIKey_EmptyOrgID(t *testing.T) {
	ctx := context.Background()
	mockOrgRepo := new(MockOrgRepository)
	mockAPIKeyRepo := new(MockAPIKeyRepository)
	mockUUIDGen := NewMockUUIDGenerator()

	service := NewAuthService(mockOrgRepo, mockAPIKeyRepo, mockUUIDGen)
	_, err := service.CreateAPIKey(ctx, "", "test-key")

	assert.Error(t, err)
}

func TestAuthService_CreateAPIKey_EmptyName(t *testing.T) {
	ctx := context.Background()
	mockOrgRepo := new(MockOrgRepository)
	mockAPIKeyRepo := new(MockAPIKeyRepository)
	mockUUIDGen := NewMockUUIDGenerator()

	service := NewAuthService(mockOrgRepo, mockAPIKeyRepo, mockUUIDGen)
	_, err := service.CreateAPIKey(ctx, "org-123", "")

	assert.Error(t, err)
}

func TestAuthService_RevokeAPIKey_EmptyID(t *testing.T) {
	ctx := context.Background()
	mockOrgRepo := new(MockOrgRepository)
	mockAPIKeyRepo := new(MockAPIKeyRepository)
	mockUUIDGen := NewMockUUIDGenerator()

	service := NewAuthService(mockOrgRepo, mockAPIKeyRepo, mockUUIDGen)
	err := service.RevokeAPIKey(ctx, "")

	assert.Error(t, err)
}

func TestAuthService_ListAPIKeys_EmptyOrgID(t *testing.T) {
	ctx := context.Background()
	mockOrgRepo := new(MockOrgRepository)
	mockAPIKeyRepo := new(MockAPIKeyRepository)
	mockUUIDGen := NewMockUUIDGenerator()

	service := NewAuthService(mockOrgRepo, mockAPIKeyRepo, mockUUIDGen)
	_, err := service.ListAPIKeys(ctx, "")

	assert.Error(t, err)
}

func TestIsValidAPIToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  bool
	}{
		{"valid token", "ntx_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", true},
		{"valid uppercase", "ntx_0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF", true},
		{"missing prefix", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", false},
		{"wrong prefix", "abc_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", false},
		{"too short", "ntx_0123456789abcdef", false},
		{"too long", "ntx_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef00", false},
		{"invalid chars", "ntx_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdeg", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidAPIToken(tt.token)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAuthService_CreateAPIKeyWithToken(t *testing.T) {
	ctx := context.Background()
	mockOrgRepo := new(MockOrgRepository)
	mockAPIKeyRepo := new(MockAPIKeyRepository)
	mockUUIDGen := NewMockUUIDGenerator("key-123")

	mockOrgRepo.On("GetByID", ctx, "org-123").Return(&domain.Organization{
		ID:        "org-123",
		Name:      "Test Org",
		CreatedAt: time.Now().UTC(),
	}, nil)

	mockAPIKeyRepo.On("Create", ctx, mock.MatchedBy(func(key *domain.APIKey) bool {
		return key.OrgID == "org-123" && key.Name == "test-key"
	})).Return(nil)

	service := NewAuthService(mockOrgRepo, mockAPIKeyRepo, mockUUIDGen)
	err := service.CreateAPIKeyWithToken(ctx, "org-123", "test-key", "ntx_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")

	require.NoError(t, err)
	mockAPIKeyRepo.AssertExpectations(t)
}

func TestAuthService_CreateAPIKeyWithToken_InvalidFormat(t *testing.T) {
	ctx := context.Background()
	mockOrgRepo := new(MockOrgRepository)
	mockAPIKeyRepo := new(MockAPIKeyRepository)
	mockUUIDGen := NewMockUUIDGenerator()

	service := NewAuthService(mockOrgRepo, mockAPIKeyRepo, mockUUIDGen)
	err := service.CreateAPIKeyWithToken(ctx, "org-123", "test-key", "invalid-token")

	assert.Error(t, err)
}
