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

func setupKnowledgeForEmbeddingJob(ctx context.Context, t *testing.T, orgRepo *OrgRepository, knowledgeRepo *KnowledgeRepository) *domain.Knowledge {
	org := &domain.Organization{
		ID:        uuid.NewString(),
		Name:      "Test Org for EmbeddingJob",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, orgRepo.Create(ctx, org))

	k := &domain.Knowledge{
		ID:        uuid.NewString(),
		OrgID:     org.ID,
		Type:      domain.KnowledgeTypeGuideline,
		Status:    domain.KnowledgeStatusDraft,
		Title:     "Knowledge for Embedding",
		Summary:   "Summary",
		BodyMD:    "Body",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, knowledgeRepo.Create(ctx, k))
	return k
}

func TestEmbeddingJobRepository_Create(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	knowledgeRepo := NewKnowledgeRepository(pool)
	jobRepo := NewEmbeddingJobRepository(pool)

	k := setupKnowledgeForEmbeddingJob(ctx, t, orgRepo, knowledgeRepo)

	job := &domain.EmbeddingJob{
		ID:          uuid.NewString(),
		KnowledgeID: k.ID,
		Status:      domain.EmbeddingJobStatusPending,
		Retries:     0,
		CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
	}

	err := jobRepo.Create(ctx, job)
	require.NoError(t, err)

	retrieved, err := jobRepo.GetByID(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, job.ID, retrieved.ID)
	assert.Equal(t, job.KnowledgeID, retrieved.KnowledgeID)
	assert.Equal(t, domain.EmbeddingJobStatusPending, retrieved.Status)
	assert.Equal(t, int32(0), retrieved.Retries)
	assert.Empty(t, retrieved.Error)
	assert.Nil(t, retrieved.ProcessedAt)
}

func TestEmbeddingJobRepository_GetByID_NotFound(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	jobRepo := NewEmbeddingJobRepository(pool)

	_, err := jobRepo.GetByID(ctx, uuid.NewString())
	assert.ErrorIs(t, err, ErrEmbeddingJobNotFound)
}

func TestEmbeddingJobRepository_GetPending(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	knowledgeRepo := NewKnowledgeRepository(pool)
	jobRepo := NewEmbeddingJobRepository(pool)

	k := setupKnowledgeForEmbeddingJob(ctx, t, orgRepo, knowledgeRepo)

	job1 := &domain.EmbeddingJob{ID: uuid.NewString(), KnowledgeID: k.ID, Status: domain.EmbeddingJobStatusPending, CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	job2 := &domain.EmbeddingJob{ID: uuid.NewString(), KnowledgeID: k.ID, Status: domain.EmbeddingJobStatusPending, CreatedAt: time.Now().UTC().Add(time.Second).Truncate(time.Microsecond)}
	job3 := &domain.EmbeddingJob{ID: uuid.NewString(), KnowledgeID: k.ID, Status: domain.EmbeddingJobStatusProcessing, CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}

	require.NoError(t, jobRepo.Create(ctx, job1))
	require.NoError(t, jobRepo.Create(ctx, job2))
	require.NoError(t, jobRepo.Create(ctx, job3))

	pending, err := jobRepo.GetPending(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, pending, 2)
	assert.Equal(t, job1.ID, pending[0].ID)
	assert.Equal(t, job2.ID, pending[1].ID)
}

func TestEmbeddingJobRepository_GetPending_WithLimit(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	knowledgeRepo := NewKnowledgeRepository(pool)
	jobRepo := NewEmbeddingJobRepository(pool)

	k := setupKnowledgeForEmbeddingJob(ctx, t, orgRepo, knowledgeRepo)

	for i := 0; i < 5; i++ {
		job := &domain.EmbeddingJob{
			ID:          uuid.NewString(),
			KnowledgeID: k.ID,
			Status:      domain.EmbeddingJobStatusPending,
			CreatedAt:   time.Now().UTC().Add(time.Duration(i) * time.Second).Truncate(time.Microsecond),
		}
		require.NoError(t, jobRepo.Create(ctx, job))
	}

	pending, err := jobRepo.GetPending(ctx, 2)
	require.NoError(t, err)
	assert.Len(t, pending, 2)
}

func TestEmbeddingJobRepository_ClaimPending(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	knowledgeRepo := NewKnowledgeRepository(pool)
	jobRepo := NewEmbeddingJobRepository(pool)

	k := setupKnowledgeForEmbeddingJob(ctx, t, orgRepo, knowledgeRepo)

	job1 := &domain.EmbeddingJob{ID: uuid.NewString(), KnowledgeID: k.ID, Status: domain.EmbeddingJobStatusPending, CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	job2 := &domain.EmbeddingJob{ID: uuid.NewString(), KnowledgeID: k.ID, Status: domain.EmbeddingJobStatusPending, CreatedAt: time.Now().UTC().Add(time.Second).Truncate(time.Microsecond)}
	job3 := &domain.EmbeddingJob{ID: uuid.NewString(), KnowledgeID: k.ID, Status: domain.EmbeddingJobStatusProcessing, CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}

	require.NoError(t, jobRepo.Create(ctx, job1))
	require.NoError(t, jobRepo.Create(ctx, job2))
	require.NoError(t, jobRepo.Create(ctx, job3))

	claimed, err := jobRepo.ClaimPending(ctx, 2)
	require.NoError(t, err)
	assert.Len(t, claimed, 2)

	for _, job := range claimed {
		retrieved, err := jobRepo.GetByID(ctx, job.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.EmbeddingJobStatusProcessing, retrieved.Status)
	}
}

func TestEmbeddingJobRepository_UpdateStatus(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	knowledgeRepo := NewKnowledgeRepository(pool)
	jobRepo := NewEmbeddingJobRepository(pool)

	k := setupKnowledgeForEmbeddingJob(ctx, t, orgRepo, knowledgeRepo)

	job := &domain.EmbeddingJob{
		ID:          uuid.NewString(),
		KnowledgeID: k.ID,
		Status:      domain.EmbeddingJobStatusPending,
		CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, jobRepo.Create(ctx, job))

	err := jobRepo.UpdateStatus(ctx, job.ID, domain.EmbeddingJobStatusProcessing, "")
	require.NoError(t, err)

	retrieved, err := jobRepo.GetByID(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.EmbeddingJobStatusProcessing, retrieved.Status)
	assert.Nil(t, retrieved.ProcessedAt)
}

func TestEmbeddingJobRepository_UpdateStatus_Completed(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	knowledgeRepo := NewKnowledgeRepository(pool)
	jobRepo := NewEmbeddingJobRepository(pool)

	k := setupKnowledgeForEmbeddingJob(ctx, t, orgRepo, knowledgeRepo)

	job := &domain.EmbeddingJob{
		ID:          uuid.NewString(),
		KnowledgeID: k.ID,
		Status:      domain.EmbeddingJobStatusProcessing,
		CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, jobRepo.Create(ctx, job))

	err := jobRepo.UpdateStatus(ctx, job.ID, domain.EmbeddingJobStatusCompleted, "")
	require.NoError(t, err)

	retrieved, err := jobRepo.GetByID(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.EmbeddingJobStatusCompleted, retrieved.Status)
	assert.NotNil(t, retrieved.ProcessedAt)
}

func TestEmbeddingJobRepository_UpdateStatus_Failed(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	knowledgeRepo := NewKnowledgeRepository(pool)
	jobRepo := NewEmbeddingJobRepository(pool)

	k := setupKnowledgeForEmbeddingJob(ctx, t, orgRepo, knowledgeRepo)

	job := &domain.EmbeddingJob{
		ID:          uuid.NewString(),
		KnowledgeID: k.ID,
		Status:      domain.EmbeddingJobStatusProcessing,
		CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, jobRepo.Create(ctx, job))

	err := jobRepo.UpdateStatus(ctx, job.ID, domain.EmbeddingJobStatusFailed, "embedding API error")
	require.NoError(t, err)

	retrieved, err := jobRepo.GetByID(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.EmbeddingJobStatusFailed, retrieved.Status)
	assert.Equal(t, "embedding API error", retrieved.Error)
	assert.NotNil(t, retrieved.ProcessedAt)
}

func TestEmbeddingJobRepository_UpdateStatus_NotFound(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	jobRepo := NewEmbeddingJobRepository(pool)

	err := jobRepo.UpdateStatus(ctx, uuid.NewString(), domain.EmbeddingJobStatusCompleted, "")
	assert.ErrorIs(t, err, ErrEmbeddingJobNotFound)
}

func TestEmbeddingJobRepository_IncrementRetries(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := NewOrgRepository(pool)
	knowledgeRepo := NewKnowledgeRepository(pool)
	jobRepo := NewEmbeddingJobRepository(pool)

	k := setupKnowledgeForEmbeddingJob(ctx, t, orgRepo, knowledgeRepo)

	job := &domain.EmbeddingJob{
		ID:          uuid.NewString(),
		KnowledgeID: k.ID,
		Status:      domain.EmbeddingJobStatusPending,
		Retries:     0,
		CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, jobRepo.Create(ctx, job))

	require.NoError(t, jobRepo.IncrementRetries(ctx, job.ID))
	require.NoError(t, jobRepo.IncrementRetries(ctx, job.ID))

	retrieved, err := jobRepo.GetByID(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, int32(2), retrieved.Retries)
}

func TestEmbeddingJobRepository_IncrementRetries_NotFound(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	jobRepo := NewEmbeddingJobRepository(pool)

	err := jobRepo.IncrementRetries(ctx, uuid.NewString())
	assert.ErrorIs(t, err, ErrEmbeddingJobNotFound)
}
