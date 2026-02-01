//go:build integration

package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/cloo-solutions/neotexai/internal/repository"
	"github.com/cloo-solutions/neotexai/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestOrg creates a test organization for integration tests
func setupTestOrg(ctx context.Context, t *testing.T, orgRepo *repository.OrgRepository) *domain.Organization {
	org := &domain.Organization{
		ID:        uuid.NewString(),
		Name:      "Test Org " + uuid.NewString()[:8],
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, orgRepo.Create(ctx, org))
	return org
}

// setupTestProject creates a test project for integration tests
func setupTestProject(ctx context.Context, t *testing.T, projectRepo *repository.ProjectRepository, orgID string) *domain.Project {
	project := &domain.Project{
		ID:        uuid.NewString(),
		OrgID:     orgID,
		Name:      "Test Project " + uuid.NewString()[:8],
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	require.NoError(t, projectRepo.Create(ctx, project))
	return project
}

func TestKnowledgeServiceIntegration_Create(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	// Create repositories
	orgRepo := repository.NewOrgRepository(pool)
	projectRepo := repository.NewProjectRepository(pool)
	knowledgeRepo := repository.NewKnowledgeRepository(pool)
	embeddingJobRepo := repository.NewEmbeddingJobRepository(pool)

	// Setup test data
	org := setupTestOrg(ctx, t, orgRepo)
	project := setupTestProject(ctx, t, projectRepo, org.ID)

	// Create service
	service := NewKnowledgeService(knowledgeRepo, embeddingJobRepo)

	t.Run("creates knowledge with first version and queues embedding job", func(t *testing.T) {
		input := CreateInput{
			OrgID:     org.ID,
			ProjectID: project.ID,
			Type:      domain.KnowledgeTypeGuideline,
			Title:     "Integration Test Guideline",
			Summary:   "Test summary for integration test",
			BodyMD:    "# Integration Test\n\nThis is a test guideline.",
			Scope:     "/src/main.go",
		}

		// Execute
		knowledge, err := service.Create(ctx, input)

		// Assert
		require.NoError(t, err)
		assert.NotEmpty(t, knowledge.ID)
		assert.Equal(t, org.ID, knowledge.OrgID)
		assert.Equal(t, project.ID, knowledge.ProjectID)
		assert.Equal(t, domain.KnowledgeTypeGuideline, knowledge.Type)
		assert.Equal(t, domain.KnowledgeStatusDraft, knowledge.Status)
		assert.Equal(t, "Integration Test Guideline", knowledge.Title)
		assert.Equal(t, "/src/main.go", knowledge.Scope)

		// Verify knowledge was persisted
		retrieved, err := knowledgeRepo.GetByID(ctx, knowledge.ID)
		require.NoError(t, err)
		assert.Equal(t, knowledge.ID, retrieved.ID)
		assert.Equal(t, knowledge.Title, retrieved.Title)

		// Verify first version was created
		version, err := knowledgeRepo.GetLatestVersion(ctx, knowledge.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(1), version.VersionNumber)
		assert.Equal(t, "Integration Test Guideline", version.Title)
		assert.Equal(t, "# Integration Test\n\nThis is a test guideline.", version.BodyMD)

		// Verify embedding job was created
		jobs, err := embeddingJobRepo.GetPending(ctx, 10)
		require.NoError(t, err)
		var foundJob bool
		for _, job := range jobs {
			if job.KnowledgeID == knowledge.ID {
				foundJob = true
				assert.Equal(t, domain.EmbeddingJobStatusPending, job.Status)
				assert.Equal(t, int32(0), job.Retries)
				break
			}
		}
		assert.True(t, foundJob, "embedding job should be created")
	})

	t.Run("creates knowledge without project", func(t *testing.T) {
		input := CreateInput{
			OrgID:   org.ID,
			Type:    domain.KnowledgeTypeLearning,
			Title:   "Org-level Learning",
			Summary: "Learning at org level",
			BodyMD:  "# Org Learning",
		}

		knowledge, err := service.Create(ctx, input)

		require.NoError(t, err)
		assert.NotEmpty(t, knowledge.ID)
		assert.Equal(t, org.ID, knowledge.OrgID)
		assert.Empty(t, knowledge.ProjectID)
		assert.Equal(t, domain.KnowledgeTypeLearning, knowledge.Type)
	})
}

func TestKnowledgeServiceIntegration_GetByID(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := repository.NewOrgRepository(pool)
	knowledgeRepo := repository.NewKnowledgeRepository(pool)
	embeddingJobRepo := repository.NewEmbeddingJobRepository(pool)

	org := setupTestOrg(ctx, t, orgRepo)
	service := NewKnowledgeService(knowledgeRepo, embeddingJobRepo)

	t.Run("retrieves existing knowledge", func(t *testing.T) {
		// Create knowledge first
		input := CreateInput{
			OrgID:   org.ID,
			Type:    domain.KnowledgeTypeDecision,
			Title:   "Test Decision",
			Summary: "Test decision summary",
			BodyMD:  "# Decision\n\nWe decided to...",
		}
		created, err := service.Create(ctx, input)
		require.NoError(t, err)

		// Retrieve it
		retrieved, err := service.GetByID(ctx, created.ID)

		require.NoError(t, err)
		assert.Equal(t, created.ID, retrieved.ID)
		assert.Equal(t, created.Title, retrieved.Title)
		assert.Equal(t, created.Type, retrieved.Type)
	})

	t.Run("returns error for non-existent knowledge", func(t *testing.T) {
		_, err := service.GetByID(ctx, uuid.NewString())

		require.Error(t, err)
		assert.Equal(t, domain.ErrKnowledgeNotFound, err)
	})
}

func TestKnowledgeServiceIntegration_Update(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := repository.NewOrgRepository(pool)
	knowledgeRepo := repository.NewKnowledgeRepository(pool)
	embeddingJobRepo := repository.NewEmbeddingJobRepository(pool)

	org := setupTestOrg(ctx, t, orgRepo)
	service := NewKnowledgeService(knowledgeRepo, embeddingJobRepo)

	t.Run("creates new version on update (immutable versioning)", func(t *testing.T) {
		// Create initial knowledge
		createInput := CreateInput{
			OrgID:   org.ID,
			Type:    domain.KnowledgeTypeTemplate,
			Title:   "Original Template",
			Summary: "Original summary",
			BodyMD:  "# Original Body",
		}
		created, err := service.Create(ctx, createInput)
		require.NoError(t, err)

		// Update it
		updateInput := UpdateInput{
			KnowledgeID: created.ID,
			Title:       "Updated Template",
			Summary:     "Updated summary",
			BodyMD:      "# Updated Body\n\nWith more content.",
			Scope:       "/templates/new.go",
		}
		updated, newVersion, err := service.Update(ctx, updateInput)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, created.ID, updated.ID)
		assert.Equal(t, "Updated Template", updated.Title)
		assert.Equal(t, "Updated summary", updated.Summary)
		assert.Equal(t, "/templates/new.go", updated.Scope)

		// Verify new version was created
		assert.Equal(t, int64(2), newVersion.VersionNumber)
		assert.Equal(t, "Updated Template", newVersion.Title)
		assert.Equal(t, "# Updated Body\n\nWith more content.", newVersion.BodyMD)

		// Verify old version is still intact
		versions, err := knowledgeRepo.GetVersions(ctx, created.ID)
		require.NoError(t, err)
		assert.Len(t, versions, 2)

		// Versions are returned in descending order
		assert.Equal(t, int64(2), versions[0].VersionNumber)
		assert.Equal(t, "Updated Template", versions[0].Title)
		assert.Equal(t, int64(1), versions[1].VersionNumber)
		assert.Equal(t, "Original Template", versions[1].Title)

		// Verify embedding job was queued for update
		jobs, err := embeddingJobRepo.GetPending(ctx, 10)
		require.NoError(t, err)
		// Should have 2 jobs - one from create, one from update
		jobCount := 0
		for _, job := range jobs {
			if job.KnowledgeID == created.ID {
				jobCount++
			}
		}
		assert.Equal(t, 2, jobCount, "should have 2 embedding jobs (create + update)")
	})

	t.Run("cannot update deprecated knowledge", func(t *testing.T) {
		// Create and deprecate
		createInput := CreateInput{
			OrgID:   org.ID,
			Type:    domain.KnowledgeTypeGuideline,
			Title:   "Soon Deprecated",
			Summary: "Will be deprecated",
			BodyMD:  "# Deprecated",
		}
		created, err := service.Create(ctx, createInput)
		require.NoError(t, err)

		_, err = service.Deprecate(ctx, created.ID)
		require.NoError(t, err)

		// Try to update
		updateInput := UpdateInput{
			KnowledgeID: created.ID,
			Title:       "Cannot Update",
			Summary:     "This should fail",
			BodyMD:      "# Fail",
		}
		_, _, err = service.Update(ctx, updateInput)

		require.Error(t, err)
		assert.Equal(t, domain.ErrCannotModifyDeprecated, err)
	})
}

func TestKnowledgeServiceIntegration_Deprecate(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := repository.NewOrgRepository(pool)
	knowledgeRepo := repository.NewKnowledgeRepository(pool)
	embeddingJobRepo := repository.NewEmbeddingJobRepository(pool)

	org := setupTestOrg(ctx, t, orgRepo)
	service := NewKnowledgeService(knowledgeRepo, embeddingJobRepo)

	t.Run("sets status to deprecated", func(t *testing.T) {
		// Create knowledge
		createInput := CreateInput{
			OrgID:   org.ID,
			Type:    domain.KnowledgeTypeSnippet,
			Title:   "To Be Deprecated",
			Summary: "Will be deprecated",
			BodyMD:  "# Code Snippet",
		}
		created, err := service.Create(ctx, createInput)
		require.NoError(t, err)
		assert.Equal(t, domain.KnowledgeStatusDraft, created.Status)

		// Deprecate it
		deprecated, err := service.Deprecate(ctx, created.ID)

		require.NoError(t, err)
		assert.Equal(t, domain.KnowledgeStatusDeprecated, deprecated.Status)

		// Verify persistence
		retrieved, err := service.GetByID(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.KnowledgeStatusDeprecated, retrieved.Status)
	})
}

func TestKnowledgeServiceIntegration_ListByOrg(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := repository.NewOrgRepository(pool)
	knowledgeRepo := repository.NewKnowledgeRepository(pool)
	embeddingJobRepo := repository.NewEmbeddingJobRepository(pool)

	org := setupTestOrg(ctx, t, orgRepo)
	service := NewKnowledgeService(knowledgeRepo, embeddingJobRepo)

	t.Run("returns all knowledge for organization", func(t *testing.T) {
		// Create multiple knowledge items
		for i := 0; i < 3; i++ {
			input := CreateInput{
				OrgID:   org.ID,
				Type:    domain.KnowledgeTypeGuideline,
				Title:   "Guideline " + string(rune('A'+i)),
				Summary: "Summary",
				BodyMD:  "# Body",
			}
			_, err := service.Create(ctx, input)
			require.NoError(t, err)
			time.Sleep(10 * time.Millisecond) // Ensure different timestamps
		}

		// List by org
		list, err := service.ListByOrg(ctx, org.ID)

		require.NoError(t, err)
		assert.Len(t, list, 3)
	})

	t.Run("returns empty list for org with no knowledge", func(t *testing.T) {
		emptyOrg := setupTestOrg(ctx, t, orgRepo)

		list, err := service.ListByOrg(ctx, emptyOrg.ID)

		require.NoError(t, err)
		assert.Empty(t, list)
	})
}

func TestKnowledgeServiceIntegration_ListByProject(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := repository.NewOrgRepository(pool)
	projectRepo := repository.NewProjectRepository(pool)
	knowledgeRepo := repository.NewKnowledgeRepository(pool)
	embeddingJobRepo := repository.NewEmbeddingJobRepository(pool)

	org := setupTestOrg(ctx, t, orgRepo)
	project1 := setupTestProject(ctx, t, projectRepo, org.ID)
	project2 := setupTestProject(ctx, t, projectRepo, org.ID)

	service := NewKnowledgeService(knowledgeRepo, embeddingJobRepo)

	t.Run("returns only knowledge for specified project", func(t *testing.T) {
		// Create knowledge for project1
		for i := 0; i < 2; i++ {
			input := CreateInput{
				OrgID:     org.ID,
				ProjectID: project1.ID,
				Type:      domain.KnowledgeTypeTemplate,
				Title:     "Project1 Template " + string(rune('A'+i)),
				Summary:   "Summary",
				BodyMD:    "# Body",
			}
			_, err := service.Create(ctx, input)
			require.NoError(t, err)
		}

		// Create knowledge for project2
		input := CreateInput{
			OrgID:     org.ID,
			ProjectID: project2.ID,
			Type:      domain.KnowledgeTypeTemplate,
			Title:     "Project2 Template",
			Summary:   "Summary",
			BodyMD:    "# Body",
		}
		_, err := service.Create(ctx, input)
		require.NoError(t, err)

		// List by project1
		list1, err := service.ListByProject(ctx, project1.ID)
		require.NoError(t, err)
		assert.Len(t, list1, 2)

		// List by project2
		list2, err := service.ListByProject(ctx, project2.ID)
		require.NoError(t, err)
		assert.Len(t, list2, 1)
	})
}

func TestKnowledgeServiceIntegration_GetLatestVersion(t *testing.T) {
	ctx := context.Background()
	pc := testutil.NewPostgresContainer(ctx, t)
	defer pc.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pc, "../../migrations")
	defer pool.Close()

	orgRepo := repository.NewOrgRepository(pool)
	knowledgeRepo := repository.NewKnowledgeRepository(pool)
	embeddingJobRepo := repository.NewEmbeddingJobRepository(pool)

	org := setupTestOrg(ctx, t, orgRepo)
	service := NewKnowledgeService(knowledgeRepo, embeddingJobRepo)

	t.Run("returns latest version after multiple updates", func(t *testing.T) {
		// Create
		createInput := CreateInput{
			OrgID:   org.ID,
			Type:    domain.KnowledgeTypeChecklist,
			Title:   "Checklist v1",
			Summary: "Initial checklist",
			BodyMD:  "- [ ] Item 1",
		}
		created, err := service.Create(ctx, createInput)
		require.NoError(t, err)

		// Update twice
		for i := 2; i <= 3; i++ {
			updateInput := UpdateInput{
				KnowledgeID: created.ID,
				Title:       "Checklist v" + string(rune('0'+i)),
				Summary:     "Checklist version " + string(rune('0'+i)),
				BodyMD:      "- [x] Item 1\n- [ ] Item " + string(rune('0'+i)),
			}
			_, _, err = service.Update(ctx, updateInput)
			require.NoError(t, err)
		}

		// Get latest version
		latest, err := service.GetLatestVersion(ctx, created.ID)

		require.NoError(t, err)
		assert.Equal(t, int64(3), latest.VersionNumber)
		assert.Equal(t, "Checklist v3", latest.Title)
		assert.Contains(t, latest.BodyMD, "Item 3")
	})
}
