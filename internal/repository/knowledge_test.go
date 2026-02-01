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

func setupOrgProjectForKnowledge(ctx context.Context, t *testing.T, pool interface {
	Exec(context.Context, string, ...interface{}) (interface{}, error)
}, orgRepo *OrgRepository, projectRepo *ProjectRepository) (*domain.Organization, *domain.Project) {
	org := &domain.Organization{
		ID:        uuid.NewString(),
		Name:      "Test Org for Knowledge",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, orgRepo.Create(ctx, org))

	project := &domain.Project{
		ID:        uuid.NewString(),
		OrgID:     org.ID,
		Name:      "Test Project for Knowledge",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, projectRepo.Create(ctx, project))

	return org, project
}

func TestKnowledgeRepository_Create(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	projectRepo := NewProjectRepository(pool)
	knowledgeRepo := NewKnowledgeRepository(pool)

	org, project := setupOrgProjectForKnowledge(ctx, t, nil, orgRepo, projectRepo)

	k := &domain.Knowledge{
		ID:        uuid.NewString(),
		OrgID:     org.ID,
		ProjectID: project.ID,
		Type:      domain.KnowledgeTypeGuideline,
		Status:    domain.KnowledgeStatusDraft,
		Title:     "Test Knowledge",
		Summary:   "Test Summary",
		BodyMD:    "# Test Body",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}

	err := knowledgeRepo.Create(ctx, k)
	require.NoError(t, err)

	retrieved, err := knowledgeRepo.GetByID(ctx, k.ID)
	require.NoError(t, err)
	assert.Equal(t, k.ID, retrieved.ID)
	assert.Equal(t, k.OrgID, retrieved.OrgID)
	assert.Equal(t, k.ProjectID, retrieved.ProjectID)
	assert.Equal(t, k.Type, retrieved.Type)
	assert.Equal(t, k.Status, retrieved.Status)
	assert.Equal(t, k.Title, retrieved.Title)
}

func TestKnowledgeRepository_Create_WithoutProject(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	knowledgeRepo := NewKnowledgeRepository(pool)

	org := &domain.Organization{ID: uuid.NewString(), Name: "Org", CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	require.NoError(t, orgRepo.Create(ctx, org))

	k := &domain.Knowledge{
		ID:        uuid.NewString(),
		OrgID:     org.ID,
		Type:      domain.KnowledgeTypeLearning,
		Status:    domain.KnowledgeStatusApproved,
		Title:     "No Project Knowledge",
		Summary:   "Summary",
		BodyMD:    "Body",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}

	err := knowledgeRepo.Create(ctx, k)
	require.NoError(t, err)

	retrieved, err := knowledgeRepo.GetByID(ctx, k.ID)
	require.NoError(t, err)
	assert.Empty(t, retrieved.ProjectID)
}

func TestKnowledgeRepository_GetByID_NotFound(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	knowledgeRepo := NewKnowledgeRepository(pool)

	_, err := knowledgeRepo.GetByID(ctx, uuid.NewString())
	assert.ErrorIs(t, err, domain.ErrKnowledgeNotFound)
}

func TestKnowledgeRepository_ListByOrg(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	knowledgeRepo := NewKnowledgeRepository(pool)

	org := &domain.Organization{ID: uuid.NewString(), Name: "Org", CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	require.NoError(t, orgRepo.Create(ctx, org))

	k1 := &domain.Knowledge{
		ID:        uuid.NewString(),
		OrgID:     org.ID,
		Type:      domain.KnowledgeTypeGuideline,
		Status:    domain.KnowledgeStatusDraft,
		Title:     "Knowledge 1",
		Summary:   "Summary 1",
		BodyMD:    "Body 1",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	k2 := &domain.Knowledge{
		ID:        uuid.NewString(),
		OrgID:     org.ID,
		Type:      domain.KnowledgeTypeLearning,
		Status:    domain.KnowledgeStatusApproved,
		Title:     "Knowledge 2",
		Summary:   "Summary 2",
		BodyMD:    "Body 2",
		CreatedAt: time.Now().UTC().Add(time.Second).Truncate(time.Microsecond),
		UpdatedAt: time.Now().UTC().Add(time.Second).Truncate(time.Microsecond),
	}

	require.NoError(t, knowledgeRepo.Create(ctx, k1))
	require.NoError(t, knowledgeRepo.Create(ctx, k2))

	list, err := knowledgeRepo.ListByOrg(ctx, org.ID)
	require.NoError(t, err)
	assert.Len(t, list, 2)
	assert.Equal(t, k2.Title, list[0].Title)
	assert.Equal(t, k1.Title, list[1].Title)
}

func TestKnowledgeRepository_ListByProject(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	projectRepo := NewProjectRepository(pool)
	knowledgeRepo := NewKnowledgeRepository(pool)

	org, project := setupOrgProjectForKnowledge(ctx, t, nil, orgRepo, projectRepo)

	k := &domain.Knowledge{
		ID:        uuid.NewString(),
		OrgID:     org.ID,
		ProjectID: project.ID,
		Type:      domain.KnowledgeTypeTemplate,
		Status:    domain.KnowledgeStatusDraft,
		Title:     "Project Knowledge",
		Summary:   "Summary",
		BodyMD:    "Body",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}

	require.NoError(t, knowledgeRepo.Create(ctx, k))

	list, err := knowledgeRepo.ListByProject(ctx, project.ID)
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, k.Title, list[0].Title)
}

func TestKnowledgeRepository_Update(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	knowledgeRepo := NewKnowledgeRepository(pool)

	org := &domain.Organization{ID: uuid.NewString(), Name: "Org", CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	require.NoError(t, orgRepo.Create(ctx, org))

	k := &domain.Knowledge{
		ID:        uuid.NewString(),
		OrgID:     org.ID,
		Type:      domain.KnowledgeTypeGuideline,
		Status:    domain.KnowledgeStatusDraft,
		Title:     "Original",
		Summary:   "Original Summary",
		BodyMD:    "Original Body",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, knowledgeRepo.Create(ctx, k))

	k.Title = "Updated"
	k.Status = domain.KnowledgeStatusApproved
	err := knowledgeRepo.Update(ctx, k)
	require.NoError(t, err)

	retrieved, err := knowledgeRepo.GetByID(ctx, k.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", retrieved.Title)
	assert.Equal(t, domain.KnowledgeStatusApproved, retrieved.Status)
}

func TestKnowledgeRepository_Update_NotFound(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	knowledgeRepo := NewKnowledgeRepository(pool)

	k := &domain.Knowledge{ID: uuid.NewString(), Title: "Ghost"}
	err := knowledgeRepo.Update(ctx, k)
	assert.ErrorIs(t, err, domain.ErrKnowledgeNotFound)
}

func TestKnowledgeRepository_Delete(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	knowledgeRepo := NewKnowledgeRepository(pool)

	org := &domain.Organization{ID: uuid.NewString(), Name: "Org", CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	require.NoError(t, orgRepo.Create(ctx, org))

	k := &domain.Knowledge{
		ID:        uuid.NewString(),
		OrgID:     org.ID,
		Type:      domain.KnowledgeTypeGuideline,
		Status:    domain.KnowledgeStatusDraft,
		Title:     "To Delete",
		Summary:   "Summary",
		BodyMD:    "Body",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, knowledgeRepo.Create(ctx, k))

	err := knowledgeRepo.Delete(ctx, k.ID)
	require.NoError(t, err)

	_, err = knowledgeRepo.GetByID(ctx, k.ID)
	assert.ErrorIs(t, err, domain.ErrKnowledgeNotFound)
}

func TestKnowledgeRepository_Delete_NotFound(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	knowledgeRepo := NewKnowledgeRepository(pool)

	err := knowledgeRepo.Delete(ctx, uuid.NewString())
	assert.ErrorIs(t, err, domain.ErrKnowledgeNotFound)
}

func TestKnowledgeRepository_CreateVersion(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	knowledgeRepo := NewKnowledgeRepository(pool)

	org := &domain.Organization{ID: uuid.NewString(), Name: "Org", CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	require.NoError(t, orgRepo.Create(ctx, org))

	k := &domain.Knowledge{
		ID:        uuid.NewString(),
		OrgID:     org.ID,
		Type:      domain.KnowledgeTypeGuideline,
		Status:    domain.KnowledgeStatusDraft,
		Title:     "Versioned Knowledge",
		Summary:   "Summary",
		BodyMD:    "Body v1",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, knowledgeRepo.Create(ctx, k))

	v := &domain.KnowledgeVersion{
		ID:            uuid.NewString(),
		KnowledgeID:   k.ID,
		VersionNumber: 1,
		Title:         k.Title,
		Summary:       k.Summary,
		BodyMD:        k.BodyMD,
		CreatedAt:     time.Now().UTC().Truncate(time.Microsecond),
	}

	err := knowledgeRepo.CreateVersion(ctx, v)
	require.NoError(t, err)

	versions, err := knowledgeRepo.GetVersions(ctx, k.ID)
	require.NoError(t, err)
	assert.Len(t, versions, 1)
	assert.Equal(t, int64(1), versions[0].VersionNumber)
}

func TestKnowledgeRepository_GetVersions(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	knowledgeRepo := NewKnowledgeRepository(pool)

	org := &domain.Organization{ID: uuid.NewString(), Name: "Org", CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	require.NoError(t, orgRepo.Create(ctx, org))

	k := &domain.Knowledge{
		ID:        uuid.NewString(),
		OrgID:     org.ID,
		Type:      domain.KnowledgeTypeGuideline,
		Status:    domain.KnowledgeStatusDraft,
		Title:     "Versioned Knowledge",
		Summary:   "Summary",
		BodyMD:    "Body",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, knowledgeRepo.Create(ctx, k))

	for i := int64(1); i <= 3; i++ {
		v := &domain.KnowledgeVersion{
			ID:            uuid.NewString(),
			KnowledgeID:   k.ID,
			VersionNumber: i,
			Title:         k.Title,
			Summary:       k.Summary,
			BodyMD:        "Body v" + string(rune('0'+i)),
			CreatedAt:     time.Now().UTC().Truncate(time.Microsecond),
		}
		require.NoError(t, knowledgeRepo.CreateVersion(ctx, v))
	}

	versions, err := knowledgeRepo.GetVersions(ctx, k.ID)
	require.NoError(t, err)
	assert.Len(t, versions, 3)
	assert.Equal(t, int64(3), versions[0].VersionNumber)
	assert.Equal(t, int64(2), versions[1].VersionNumber)
	assert.Equal(t, int64(1), versions[2].VersionNumber)
}

func TestKnowledgeRepository_GetLatestVersion(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	knowledgeRepo := NewKnowledgeRepository(pool)

	org := &domain.Organization{ID: uuid.NewString(), Name: "Org", CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	require.NoError(t, orgRepo.Create(ctx, org))

	k := &domain.Knowledge{
		ID:        uuid.NewString(),
		OrgID:     org.ID,
		Type:      domain.KnowledgeTypeGuideline,
		Status:    domain.KnowledgeStatusDraft,
		Title:     "Versioned Knowledge",
		Summary:   "Summary",
		BodyMD:    "Body",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, knowledgeRepo.Create(ctx, k))

	for i := int64(1); i <= 2; i++ {
		v := &domain.KnowledgeVersion{
			ID:            uuid.NewString(),
			KnowledgeID:   k.ID,
			VersionNumber: i,
			Title:         "Title v" + string(rune('0'+i)),
			Summary:       k.Summary,
			BodyMD:        "Body v" + string(rune('0'+i)),
			CreatedAt:     time.Now().UTC().Truncate(time.Microsecond),
		}
		require.NoError(t, knowledgeRepo.CreateVersion(ctx, v))
	}

	latest, err := knowledgeRepo.GetLatestVersion(ctx, k.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), latest.VersionNumber)
	assert.Equal(t, "Title v2", latest.Title)
}

func TestKnowledgeRepository_GetLatestVersion_NotFound(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	knowledgeRepo := NewKnowledgeRepository(pool)

	_, err := knowledgeRepo.GetLatestVersion(ctx, uuid.NewString())
	assert.ErrorIs(t, err, domain.ErrKnowledgeNotFound)
}
