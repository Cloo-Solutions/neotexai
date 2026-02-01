//go:build integration

package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/cloo-solutions/neotexai/internal/repository"
	"github.com/cloo-solutions/neotexai/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthService_Integration_CreateOrg(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := repository.NewOrgRepository(pool)
	keyRepo := repository.NewAPIKeyRepository(pool)
	uuidGen := &DefaultUUIDGenerator{}

	service := NewAuthService(orgRepo, keyRepo, uuidGen)

	org, err := service.CreateOrg(ctx, "Integration Test Org")
	require.NoError(t, err)
	assert.NotEmpty(t, org.ID)
	assert.Equal(t, "Integration Test Org", org.Name)

	retrieved, err := orgRepo.GetByID(ctx, org.ID)
	require.NoError(t, err)
	assert.Equal(t, org.ID, retrieved.ID)
	assert.Equal(t, org.Name, retrieved.Name)
}

func TestAuthService_Integration_CreateAPIKey(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := repository.NewOrgRepository(pool)
	keyRepo := repository.NewAPIKeyRepository(pool)
	uuidGen := &DefaultUUIDGenerator{}

	service := NewAuthService(orgRepo, keyRepo, uuidGen)

	org, err := service.CreateOrg(ctx, "Test Org")
	require.NoError(t, err)

	plaintext, err := service.CreateAPIKey(ctx, org.ID, "test-key")
	require.NoError(t, err)
	assert.Equal(t, 64, len(plaintext))

	keys, err := keyRepo.GetByOrgID(ctx, org.ID)
	require.NoError(t, err)
	assert.Len(t, keys, 1)
	assert.Equal(t, "test-key", keys[0].Name)
	assert.NotEqual(t, plaintext, keys[0].KeyHash)
}

func TestAuthService_Integration_CreateAPIKey_Plaintext(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := repository.NewOrgRepository(pool)
	keyRepo := repository.NewAPIKeyRepository(pool)
	uuidGen := &DefaultUUIDGenerator{}

	service := NewAuthService(orgRepo, keyRepo, uuidGen)

	org, err := service.CreateOrg(ctx, "Test Org")
	require.NoError(t, err)

	plaintext, err := service.CreateAPIKey(ctx, org.ID, "test-key")
	require.NoError(t, err)

	keys, err := keyRepo.GetByOrgID(ctx, org.ID)
	require.NoError(t, err)
	assert.Len(t, keys, 1)
	assert.Equal(t, "test-key", keys[0].Name)
	assert.NotEqual(t, plaintext, keys[0].KeyHash)
}

func TestAuthService_Integration_ValidateAPIKey(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := repository.NewOrgRepository(pool)
	keyRepo := repository.NewAPIKeyRepository(pool)
	uuidGen := &DefaultUUIDGenerator{}

	service := NewAuthService(orgRepo, keyRepo, uuidGen)

	org, err := service.CreateOrg(ctx, "Test Org")
	require.NoError(t, err)

	plaintext, err := service.CreateAPIKey(ctx, org.ID, "test-key")
	require.NoError(t, err)

	keys, err := keyRepo.GetByOrgID(ctx, org.ID)
	require.NoError(t, err)
	keyID := keys[0].ID

	orgID, err := service.ValidateAPIKey(ctx, keyID, plaintext)
	require.NoError(t, err)
	assert.Equal(t, org.ID, orgID)
}

func TestAuthService_Integration_ValidateAPIKey_InvalidToken(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := repository.NewOrgRepository(pool)
	keyRepo := repository.NewAPIKeyRepository(pool)
	uuidGen := &DefaultUUIDGenerator{}

	service := NewAuthService(orgRepo, keyRepo, uuidGen)

	org, err := service.CreateOrg(ctx, "Test Org")
	require.NoError(t, err)

	_, err = service.CreateAPIKey(ctx, org.ID, "test-key")
	require.NoError(t, err)

	keys, err := keyRepo.GetByOrgID(ctx, org.ID)
	require.NoError(t, err)
	keyID := keys[0].ID

	_, err = service.ValidateAPIKey(ctx, keyID, "wrong-token")
	assert.ErrorIs(t, err, domain.ErrInvalidAPIKey)
}

func TestAuthService_Integration_ValidateAPIKey_RevokedKey(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := repository.NewOrgRepository(pool)
	keyRepo := repository.NewAPIKeyRepository(pool)
	uuidGen := &DefaultUUIDGenerator{}

	service := NewAuthService(orgRepo, keyRepo, uuidGen)

	org, err := service.CreateOrg(ctx, "Test Org")
	require.NoError(t, err)

	plaintext, err := service.CreateAPIKey(ctx, org.ID, "test-key")
	require.NoError(t, err)

	keys, err := keyRepo.GetByOrgID(ctx, org.ID)
	require.NoError(t, err)
	keyID := keys[0].ID

	err = service.RevokeAPIKey(ctx, keyID)
	require.NoError(t, err)

	_, err = service.ValidateAPIKey(ctx, keyID, plaintext)
	assert.ErrorIs(t, err, domain.ErrAPIKeyRevoked)
}

func TestAuthService_Integration_RevokeAPIKey(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := repository.NewOrgRepository(pool)
	keyRepo := repository.NewAPIKeyRepository(pool)
	uuidGen := &DefaultUUIDGenerator{}

	service := NewAuthService(orgRepo, keyRepo, uuidGen)

	org, err := service.CreateOrg(ctx, "Test Org")
	require.NoError(t, err)

	_, err = service.CreateAPIKey(ctx, org.ID, "test-key")
	require.NoError(t, err)

	keys, err := keyRepo.GetByOrgID(ctx, org.ID)
	require.NoError(t, err)
	keyID := keys[0].ID

	err = service.RevokeAPIKey(ctx, keyID)
	require.NoError(t, err)

	retrieved, err := keyRepo.GetByID(ctx, keyID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved.RevokedAt)
	assert.True(t, retrieved.IsRevoked())
}

func TestAuthService_Integration_ListAPIKeys(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := repository.NewOrgRepository(pool)
	keyRepo := repository.NewAPIKeyRepository(pool)
	uuidGen := &DefaultUUIDGenerator{}

	service := NewAuthService(orgRepo, keyRepo, uuidGen)

	org, err := service.CreateOrg(ctx, "Test Org")
	require.NoError(t, err)

	_, err = service.CreateAPIKey(ctx, org.ID, "key-1")
	require.NoError(t, err)

	_, err = service.CreateAPIKey(ctx, org.ID, "key-2")
	require.NoError(t, err)

	keys, err := service.ListAPIKeys(ctx, org.ID)
	require.NoError(t, err)
	assert.Len(t, keys, 2)
	assert.Equal(t, "key-2", keys[0].Name)
	assert.Equal(t, "key-1", keys[1].Name)
}

func TestAuthService_Integration_MultipleOrgs(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := repository.NewOrgRepository(pool)
	keyRepo := repository.NewAPIKeyRepository(pool)
	uuidGen := &DefaultUUIDGenerator{}

	service := NewAuthService(orgRepo, keyRepo, uuidGen)

	org1, err := service.CreateOrg(ctx, "Org 1")
	require.NoError(t, err)

	org2, err := service.CreateOrg(ctx, "Org 2")
	require.NoError(t, err)

	plaintext1, err := service.CreateAPIKey(ctx, org1.ID, "key-1")
	require.NoError(t, err)

	plaintext2, err := service.CreateAPIKey(ctx, org2.ID, "key-2")
	require.NoError(t, err)

	keys1, err := service.ListAPIKeys(ctx, org1.ID)
	require.NoError(t, err)
	assert.Len(t, keys1, 1)

	keys2, err := service.ListAPIKeys(ctx, org2.ID)
	require.NoError(t, err)
	assert.Len(t, keys2, 1)

	orgID1, err := service.ValidateAPIKey(ctx, keys1[0].ID, plaintext1)
	require.NoError(t, err)
	assert.Equal(t, org1.ID, orgID1)

	orgID2, err := service.ValidateAPIKey(ctx, keys2[0].ID, plaintext2)
	require.NoError(t, err)
	assert.Equal(t, org2.ID, orgID2)
}

func TestAuthService_Integration_CreateAPIKey_OrgNotFound(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := repository.NewOrgRepository(pool)
	keyRepo := repository.NewAPIKeyRepository(pool)
	uuidGen := &DefaultUUIDGenerator{}

	service := NewAuthService(orgRepo, keyRepo, uuidGen)

	_, err := service.CreateAPIKey(ctx, uuid.NewString(), "test-key")
	assert.ErrorIs(t, err, domain.ErrOrganizationNotFound)
}

func TestAuthService_Integration_APIKeyTokenUniqueness(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := repository.NewOrgRepository(pool)
	keyRepo := repository.NewAPIKeyRepository(pool)
	uuidGen := &DefaultUUIDGenerator{}

	service := NewAuthService(orgRepo, keyRepo, uuidGen)

	org, err := service.CreateOrg(ctx, "Test Org")
	require.NoError(t, err)

	plaintext1, err := service.CreateAPIKey(ctx, org.ID, "key-1")
	require.NoError(t, err)

	plaintext2, err := service.CreateAPIKey(ctx, org.ID, "key-2")
	require.NoError(t, err)

	assert.NotEqual(t, plaintext1, plaintext2)

	keys, err := keyRepo.GetByOrgID(ctx, org.ID)
	require.NoError(t, err)
	assert.NotEqual(t, keys[0].KeyHash, keys[1].KeyHash)
}

func TestAuthService_Integration_ValidateAPIKey_NotFound(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := repository.NewOrgRepository(pool)
	keyRepo := repository.NewAPIKeyRepository(pool)
	uuidGen := &DefaultUUIDGenerator{}

	service := NewAuthService(orgRepo, keyRepo, uuidGen)

	_, err := service.ValidateAPIKey(ctx, uuid.NewString(), "token")
	assert.ErrorIs(t, err, domain.ErrAPIKeyNotFound)
}
