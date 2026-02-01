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

func setupOrgForAsset(ctx context.Context, t *testing.T, orgRepo *OrgRepository) *domain.Organization {
	org := &domain.Organization{
		ID:        uuid.NewString(),
		Name:      "Test Org for Asset",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, orgRepo.Create(ctx, org))
	return org
}

func TestAssetRepository_Create(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	assetRepo := NewAssetRepository(pool)

	org := setupOrgForAsset(ctx, t, orgRepo)

	asset := &domain.Asset{
		ID:          uuid.NewString(),
		OrgID:       org.ID,
		Filename:    "test.pdf",
		MimeType:    "application/pdf",
		SHA256:      "abc123hash",
		StorageKey:  "bucket/test.pdf",
		Keywords:    []string{"test", "document"},
		Description: "A test document",
		CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
	}

	err := assetRepo.Create(ctx, asset)
	require.NoError(t, err)

	retrieved, err := assetRepo.GetByID(ctx, asset.ID)
	require.NoError(t, err)
	assert.Equal(t, asset.ID, retrieved.ID)
	assert.Equal(t, asset.OrgID, retrieved.OrgID)
	assert.Equal(t, asset.Filename, retrieved.Filename)
	assert.Equal(t, asset.MimeType, retrieved.MimeType)
	assert.Equal(t, asset.SHA256, retrieved.SHA256)
	assert.Equal(t, asset.StorageKey, retrieved.StorageKey)
	assert.Equal(t, asset.Keywords, retrieved.Keywords)
	assert.Equal(t, asset.Description, retrieved.Description)
}

func TestAssetRepository_Create_WithProject(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	projectRepo := NewProjectRepository(pool)
	assetRepo := NewAssetRepository(pool)

	org := setupOrgForAsset(ctx, t, orgRepo)
	project := &domain.Project{ID: uuid.NewString(), OrgID: org.ID, Name: "Project", CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	require.NoError(t, projectRepo.Create(ctx, project))

	asset := &domain.Asset{
		ID:          uuid.NewString(),
		OrgID:       org.ID,
		ProjectID:   project.ID,
		Filename:    "project-file.png",
		MimeType:    "image/png",
		SHA256:      "xyz789hash",
		StorageKey:  "bucket/project-file.png",
		Keywords:    []string{"image"},
		Description: "Project image",
		CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
	}

	err := assetRepo.Create(ctx, asset)
	require.NoError(t, err)

	retrieved, err := assetRepo.GetByID(ctx, asset.ID)
	require.NoError(t, err)
	assert.Equal(t, project.ID, retrieved.ProjectID)
}

func TestAssetRepository_GetByID_NotFound(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	assetRepo := NewAssetRepository(pool)

	_, err := assetRepo.GetByID(ctx, uuid.NewString())
	assert.ErrorIs(t, err, domain.ErrAssetNotFound)
}

func TestAssetRepository_ListByKnowledge(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	knowledgeRepo := NewKnowledgeRepository(pool)
	assetRepo := NewAssetRepository(pool)

	org := setupOrgForAsset(ctx, t, orgRepo)

	knowledge := &domain.Knowledge{
		ID:        uuid.NewString(),
		OrgID:     org.ID,
		Type:      domain.KnowledgeTypeGuideline,
		Status:    domain.KnowledgeStatusDraft,
		Title:     "Knowledge with Assets",
		Summary:   "Summary",
		BodyMD:    "Body",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, knowledgeRepo.Create(ctx, knowledge))

	asset1 := &domain.Asset{
		ID:         uuid.NewString(),
		OrgID:      org.ID,
		Filename:   "file1.pdf",
		MimeType:   "application/pdf",
		SHA256:     "hash1",
		StorageKey: "bucket/file1.pdf",
		Keywords:   []string{},
		CreatedAt:  time.Now().UTC().Truncate(time.Microsecond),
	}
	asset2 := &domain.Asset{
		ID:         uuid.NewString(),
		OrgID:      org.ID,
		Filename:   "file2.pdf",
		MimeType:   "application/pdf",
		SHA256:     "hash2",
		StorageKey: "bucket/file2.pdf",
		Keywords:   []string{},
		CreatedAt:  time.Now().UTC().Add(time.Second).Truncate(time.Microsecond),
	}

	require.NoError(t, assetRepo.Create(ctx, asset1))
	require.NoError(t, assetRepo.Create(ctx, asset2))
	require.NoError(t, assetRepo.LinkToKnowledge(ctx, knowledge.ID, asset1.ID))
	require.NoError(t, assetRepo.LinkToKnowledge(ctx, knowledge.ID, asset2.ID))

	assets, err := assetRepo.ListByKnowledge(ctx, knowledge.ID)
	require.NoError(t, err)
	assert.Len(t, assets, 2)
	assert.Equal(t, asset2.Filename, assets[0].Filename)
	assert.Equal(t, asset1.Filename, assets[1].Filename)
}

func TestAssetRepository_ListByKnowledge_Empty(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	assetRepo := NewAssetRepository(pool)

	assets, err := assetRepo.ListByKnowledge(ctx, uuid.NewString())
	require.NoError(t, err)
	assert.Empty(t, assets)
}

func TestAssetRepository_Delete(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	assetRepo := NewAssetRepository(pool)

	org := setupOrgForAsset(ctx, t, orgRepo)

	asset := &domain.Asset{
		ID:         uuid.NewString(),
		OrgID:      org.ID,
		Filename:   "to-delete.pdf",
		MimeType:   "application/pdf",
		SHA256:     "hash",
		StorageKey: "bucket/to-delete.pdf",
		Keywords:   []string{},
		CreatedAt:  time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, assetRepo.Create(ctx, asset))

	err := assetRepo.Delete(ctx, asset.ID)
	require.NoError(t, err)

	_, err = assetRepo.GetByID(ctx, asset.ID)
	assert.ErrorIs(t, err, domain.ErrAssetNotFound)
}

func TestAssetRepository_Delete_NotFound(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	assetRepo := NewAssetRepository(pool)

	err := assetRepo.Delete(ctx, uuid.NewString())
	assert.ErrorIs(t, err, domain.ErrAssetNotFound)
}

func TestAssetRepository_LinkUnlink(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	knowledgeRepo := NewKnowledgeRepository(pool)
	assetRepo := NewAssetRepository(pool)

	org := setupOrgForAsset(ctx, t, orgRepo)

	knowledge := &domain.Knowledge{
		ID:        uuid.NewString(),
		OrgID:     org.ID,
		Type:      domain.KnowledgeTypeGuideline,
		Status:    domain.KnowledgeStatusDraft,
		Title:     "Knowledge",
		Summary:   "Summary",
		BodyMD:    "Body",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, knowledgeRepo.Create(ctx, knowledge))

	asset := &domain.Asset{
		ID:         uuid.NewString(),
		OrgID:      org.ID,
		Filename:   "link-test.pdf",
		MimeType:   "application/pdf",
		SHA256:     "hash",
		StorageKey: "bucket/link-test.pdf",
		Keywords:   []string{},
		CreatedAt:  time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, assetRepo.Create(ctx, asset))

	require.NoError(t, assetRepo.LinkToKnowledge(ctx, knowledge.ID, asset.ID))
	assets, err := assetRepo.ListByKnowledge(ctx, knowledge.ID)
	require.NoError(t, err)
	assert.Len(t, assets, 1)

	require.NoError(t, assetRepo.UnlinkFromKnowledge(ctx, knowledge.ID, asset.ID))
	assets, err = assetRepo.ListByKnowledge(ctx, knowledge.ID)
	require.NoError(t, err)
	assert.Empty(t, assets)
}
