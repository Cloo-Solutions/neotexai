//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/cloo-solutions/neotexai/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupOrgForAPIKey(ctx context.Context, t *testing.T, orgRepo *OrgRepository) *domain.Organization {
	org := &domain.Organization{
		ID:        uuid.NewString(),
		Name:      "Test Org for APIKey",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, orgRepo.Create(ctx, org))
	return org
}

func TestAPIKeyRepository_Create(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	keyRepo := NewAPIKeyRepository(pool)

	org := setupOrgForAPIKey(ctx, t, orgRepo)

	key := &domain.APIKey{
		ID:        uuid.NewString(),
		OrgID:     org.ID,
		Name:      "Test Key",
		KeyHash:   "hashed_key_value",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}

	err := keyRepo.Create(ctx, key)
	require.NoError(t, err)

	retrieved, err := keyRepo.GetByID(ctx, key.ID)
	require.NoError(t, err)
	assert.Equal(t, key.ID, retrieved.ID)
	assert.Equal(t, key.OrgID, retrieved.OrgID)
	assert.Equal(t, key.Name, retrieved.Name)
	assert.Equal(t, key.KeyHash, retrieved.KeyHash)
	assert.Nil(t, retrieved.RevokedAt)
}

func TestAPIKeyRepository_Create_ForeignKeyViolation(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	keyRepo := NewAPIKeyRepository(pool)

	key := &domain.APIKey{
		ID:        uuid.NewString(),
		OrgID:     uuid.NewString(),
		Name:      "Orphan Key",
		KeyHash:   "hashed",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}

	err := keyRepo.Create(ctx, key)
	assert.Error(t, err)
}

func TestAPIKeyRepository_GetByID_NotFound(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	keyRepo := NewAPIKeyRepository(pool)

	_, err := keyRepo.GetByID(ctx, uuid.NewString())
	assert.ErrorIs(t, err, domain.ErrAPIKeyNotFound)
}

func TestAPIKeyRepository_GetByOrgID(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	keyRepo := NewAPIKeyRepository(pool)

	org := setupOrgForAPIKey(ctx, t, orgRepo)

	key1 := &domain.APIKey{ID: uuid.NewString(), OrgID: org.ID, Name: "Key 1", KeyHash: "hash1", CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	key2 := &domain.APIKey{ID: uuid.NewString(), OrgID: org.ID, Name: "Key 2", KeyHash: "hash2", CreatedAt: time.Now().UTC().Add(time.Second).Truncate(time.Microsecond)}

	require.NoError(t, keyRepo.Create(ctx, key1))
	require.NoError(t, keyRepo.Create(ctx, key2))

	keys, err := keyRepo.GetByOrgID(ctx, org.ID)
	require.NoError(t, err)
	assert.Len(t, keys, 2)
	assert.Equal(t, key2.Name, keys[0].Name)
	assert.Equal(t, key1.Name, keys[1].Name)
}

func TestAPIKeyRepository_GetByOrgID_Empty(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	keyRepo := NewAPIKeyRepository(pool)

	keys, err := keyRepo.GetByOrgID(ctx, uuid.NewString())
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestAPIKeyRepository_Revoke(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	keyRepo := NewAPIKeyRepository(pool)

	org := setupOrgForAPIKey(ctx, t, orgRepo)
	key := &domain.APIKey{ID: uuid.NewString(), OrgID: org.ID, Name: "To Revoke", KeyHash: "hash", CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	require.NoError(t, keyRepo.Create(ctx, key))

	err := keyRepo.Revoke(ctx, key.ID)
	require.NoError(t, err)

	retrieved, err := keyRepo.GetByID(ctx, key.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved.RevokedAt)
	assert.True(t, retrieved.IsRevoked())
}

func TestAPIKeyRepository_Revoke_NotFound(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	keyRepo := NewAPIKeyRepository(pool)

	err := keyRepo.Revoke(ctx, uuid.NewString())
	assert.ErrorIs(t, err, domain.ErrAPIKeyNotFound)
}

func TestAPIKeyRepository_Revoke_AlreadyRevoked(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	keyRepo := NewAPIKeyRepository(pool)

	org := setupOrgForAPIKey(ctx, t, orgRepo)
	key := &domain.APIKey{ID: uuid.NewString(), OrgID: org.ID, Name: "Already Revoked", KeyHash: "hash", CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	require.NoError(t, keyRepo.Create(ctx, key))

	require.NoError(t, keyRepo.Revoke(ctx, key.ID))
	err := keyRepo.Revoke(ctx, key.ID)
	assert.ErrorIs(t, err, domain.ErrAPIKeyNotFound)
}

func TestAPIKeyRepository_Delete(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	keyRepo := NewAPIKeyRepository(pool)

	org := setupOrgForAPIKey(ctx, t, orgRepo)
	key := &domain.APIKey{ID: uuid.NewString(), OrgID: org.ID, Name: "To Delete", KeyHash: "hash", CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	require.NoError(t, keyRepo.Create(ctx, key))

	err := keyRepo.Delete(ctx, key.ID)
	require.NoError(t, err)

	_, err = keyRepo.GetByID(ctx, key.ID)
	assert.ErrorIs(t, err, domain.ErrAPIKeyNotFound)
}

func TestAPIKeyRepository_Delete_NotFound(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	keyRepo := NewAPIKeyRepository(pool)

	err := keyRepo.Delete(ctx, uuid.NewString())
	assert.ErrorIs(t, err, domain.ErrAPIKeyNotFound)
}
