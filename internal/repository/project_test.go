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

func setupOrgForProject(ctx context.Context, t *testing.T, orgRepo *OrgRepository) *domain.Organization {
	org := &domain.Organization{
		ID:        uuid.NewString(),
		Name:      "Test Org",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, orgRepo.Create(ctx, org))
	return org
}

func TestProjectRepository_Create(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	projectRepo := NewProjectRepository(pool)

	org := setupOrgForProject(ctx, t, orgRepo)

	project := &domain.Project{
		ID:        uuid.NewString(),
		OrgID:     org.ID,
		Name:      "Test Project",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}

	err := projectRepo.Create(ctx, project)
	require.NoError(t, err)

	retrieved, err := projectRepo.GetByID(ctx, project.ID)
	require.NoError(t, err)
	assert.Equal(t, project.ID, retrieved.ID)
	assert.Equal(t, project.OrgID, retrieved.OrgID)
	assert.Equal(t, project.Name, retrieved.Name)
}

func TestProjectRepository_Create_ForeignKeyViolation(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	projectRepo := NewProjectRepository(pool)

	project := &domain.Project{
		ID:        uuid.NewString(),
		OrgID:     uuid.NewString(),
		Name:      "Orphan Project",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}

	err := projectRepo.Create(ctx, project)
	assert.Error(t, err)
}

func TestProjectRepository_GetByID_NotFound(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	projectRepo := NewProjectRepository(pool)

	_, err := projectRepo.GetByID(ctx, uuid.NewString())
	assert.ErrorIs(t, err, domain.ErrProjectNotFound)
}

func TestProjectRepository_ListByOrg(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	projectRepo := NewProjectRepository(pool)

	org := setupOrgForProject(ctx, t, orgRepo)

	proj1 := &domain.Project{ID: uuid.NewString(), OrgID: org.ID, Name: "Project 1", CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	proj2 := &domain.Project{ID: uuid.NewString(), OrgID: org.ID, Name: "Project 2", CreatedAt: time.Now().UTC().Add(time.Second).Truncate(time.Microsecond)}

	require.NoError(t, projectRepo.Create(ctx, proj1))
	require.NoError(t, projectRepo.Create(ctx, proj2))

	projects, err := projectRepo.ListByOrg(ctx, org.ID)
	require.NoError(t, err)
	assert.Len(t, projects, 2)
	assert.Equal(t, proj2.Name, projects[0].Name)
	assert.Equal(t, proj1.Name, projects[1].Name)
}

func TestProjectRepository_ListByOrg_Empty(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	projectRepo := NewProjectRepository(pool)

	projects, err := projectRepo.ListByOrg(ctx, uuid.NewString())
	require.NoError(t, err)
	assert.Empty(t, projects)
}

func TestProjectRepository_Update(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	projectRepo := NewProjectRepository(pool)

	org := setupOrgForProject(ctx, t, orgRepo)
	project := &domain.Project{ID: uuid.NewString(), OrgID: org.ID, Name: "Original", CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	require.NoError(t, projectRepo.Create(ctx, project))

	project.Name = "Updated"
	err := projectRepo.Update(ctx, project)
	require.NoError(t, err)

	retrieved, err := projectRepo.GetByID(ctx, project.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", retrieved.Name)
}

func TestProjectRepository_Update_NotFound(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	projectRepo := NewProjectRepository(pool)

	project := &domain.Project{ID: uuid.NewString(), Name: "Ghost"}
	err := projectRepo.Update(ctx, project)
	assert.ErrorIs(t, err, domain.ErrProjectNotFound)
}

func TestProjectRepository_Delete(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	projectRepo := NewProjectRepository(pool)

	org := setupOrgForProject(ctx, t, orgRepo)
	project := &domain.Project{ID: uuid.NewString(), OrgID: org.ID, Name: "To Delete", CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	require.NoError(t, projectRepo.Create(ctx, project))

	err := projectRepo.Delete(ctx, project.ID)
	require.NoError(t, err)

	_, err = projectRepo.GetByID(ctx, project.ID)
	assert.ErrorIs(t, err, domain.ErrProjectNotFound)
}

func TestProjectRepository_Delete_NotFound(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	projectRepo := NewProjectRepository(pool)

	err := projectRepo.Delete(ctx, uuid.NewString())
	assert.ErrorIs(t, err, domain.ErrProjectNotFound)
}
