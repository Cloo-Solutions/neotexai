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

func TestOrgRepository_Create(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	repo := NewOrgRepository(pool)

	org := &domain.Organization{
		ID:        uuid.NewString(),
		Name:      "Test Org",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}

	err := repo.Create(ctx, org)
	require.NoError(t, err)

	retrieved, err := repo.GetByID(ctx, org.ID)
	require.NoError(t, err)
	assert.Equal(t, org.ID, retrieved.ID)
	assert.Equal(t, org.Name, retrieved.Name)
}

func TestOrgRepository_GetByID_NotFound(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	repo := NewOrgRepository(pool)

	_, err := repo.GetByID(ctx, uuid.NewString())
	assert.ErrorIs(t, err, domain.ErrOrganizationNotFound)
}

func TestOrgRepository_List(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	repo := NewOrgRepository(pool)

	org1 := &domain.Organization{ID: uuid.NewString(), Name: "Org 1", CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	org2 := &domain.Organization{ID: uuid.NewString(), Name: "Org 2", CreatedAt: time.Now().UTC().Add(time.Second).Truncate(time.Microsecond)}

	require.NoError(t, repo.Create(ctx, org1))
	require.NoError(t, repo.Create(ctx, org2))

	orgs, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, orgs, 2)
	assert.Equal(t, org2.Name, orgs[0].Name)
	assert.Equal(t, org1.Name, orgs[1].Name)
}

func TestOrgRepository_Update(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	repo := NewOrgRepository(pool)

	org := &domain.Organization{ID: uuid.NewString(), Name: "Original", CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	require.NoError(t, repo.Create(ctx, org))

	org.Name = "Updated"
	err := repo.Update(ctx, org)
	require.NoError(t, err)

	retrieved, err := repo.GetByID(ctx, org.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", retrieved.Name)
}

func TestOrgRepository_Update_NotFound(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	repo := NewOrgRepository(pool)

	org := &domain.Organization{ID: uuid.NewString(), Name: "Ghost"}
	err := repo.Update(ctx, org)
	assert.ErrorIs(t, err, domain.ErrOrganizationNotFound)
}

func TestOrgRepository_Delete(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	repo := NewOrgRepository(pool)

	org := &domain.Organization{ID: uuid.NewString(), Name: "To Delete", CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	require.NoError(t, repo.Create(ctx, org))

	err := repo.Delete(ctx, org.ID)
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, org.ID)
	assert.ErrorIs(t, err, domain.ErrOrganizationNotFound)
}

func TestOrgRepository_Delete_NotFound(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	repo := NewOrgRepository(pool)

	err := repo.Delete(ctx, uuid.NewString())
	assert.ErrorIs(t, err, domain.ErrOrganizationNotFound)
}
