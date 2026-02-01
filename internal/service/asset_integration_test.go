//go:build integration

package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/cloo-solutions/neotexai/internal/repository"
	"github.com/cloo-solutions/neotexai/internal/storage"
	"github.com/cloo-solutions/neotexai/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssetServiceIntegration_FullWorkflow(t *testing.T) {
	ctx := context.Background()

	pgContainer := testutil.NewPostgresContainer(ctx, t)
	defer pgContainer.Terminate(ctx)

	s3Container := testutil.NewRustFSContainer(ctx, t)
	defer s3Container.Terminate(ctx)

	pool := testutil.NewTestPool(ctx, t, pgContainer, "../../migrations")
	defer pool.Close()

	s3Client, err := storage.NewS3Client(ctx, storage.S3ClientConfig{
		Endpoint:        s3Container.Endpoint(),
		Region:          "us-east-1",
		AccessKeyID:     "rustfsadmin",
		SecretAccessKey: "rustfsadmin",
		Bucket:          "test-assets",
		UsePathStyle:    true,
	})
	require.NoError(t, err)

	require.NoError(t, s3Client.EnsureBucket(ctx))

	orgRepo := repository.NewOrgRepository(pool)
	assetRepo := repository.NewAssetRepository(pool)

	org := &domain.Organization{
		ID:        uuid.NewString(),
		Name:      "Test Org",
		CreatedAt: time.Now().UTC(),
	}
	require.NoError(t, orgRepo.Create(ctx, org))

	storageAdapter := &S3StorageAdapter{client: s3Client}
	assetService := NewAssetService(assetRepo, storageAdapter)

	t.Run("InitUpload returns presigned URL", func(t *testing.T) {
		input := InitUploadInput{
			OrgID:       org.ID,
			Filename:    "test-document.txt",
			ContentType: "text/plain",
		}

		result, err := assetService.InitUpload(ctx, input)

		require.NoError(t, err)
		assert.NotEmpty(t, result.AssetID)
		assert.NotEmpty(t, result.StorageKey)
		assert.Contains(t, result.UploadURL, s3Container.Endpoint())
	})

	t.Run("CompleteUpload creates asset after file upload", func(t *testing.T) {
		fileContent := []byte("Hello, this is test file content for RustFS!")
		hash := sha256.Sum256(fileContent)
		sha256Hash := hex.EncodeToString(hash[:])

		initResult, err := assetService.InitUpload(ctx, InitUploadInput{
			OrgID:       org.ID,
			Filename:    "uploaded-file.txt",
			ContentType: "text/plain",
		})
		require.NoError(t, err)

		req, err := http.NewRequestWithContext(ctx, "PUT", initResult.UploadURL, bytes.NewReader(fileContent))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "text/plain")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.True(t, resp.StatusCode >= 200 && resp.StatusCode < 300, "upload should succeed, got %d", resp.StatusCode)

		asset, err := assetService.CompleteUpload(ctx, CompleteUploadInput{
			AssetID:     initResult.AssetID,
			OrgID:       org.ID,
			Filename:    "uploaded-file.txt",
			ContentType: "text/plain",
			StorageKey:  initResult.StorageKey,
			SHA256:      sha256Hash,
			Keywords:    []string{"test", "integration"},
			Description: "Integration test file",
		})

		require.NoError(t, err)
		assert.NotNil(t, asset)
		assert.Equal(t, initResult.AssetID, asset.ID)
		assert.Equal(t, org.ID, asset.OrgID)
		assert.Equal(t, sha256Hash, asset.SHA256)

		retrieved, err := assetRepo.GetByID(ctx, asset.ID)
		require.NoError(t, err)
		assert.Equal(t, asset.ID, retrieved.ID)
	})

	t.Run("GetDownloadURL returns working presigned URL", func(t *testing.T) {
		fileContent := []byte("Download test content")
		hash := sha256.Sum256(fileContent)
		sha256Hash := hex.EncodeToString(hash[:])

		initResult, err := assetService.InitUpload(ctx, InitUploadInput{
			OrgID:       org.ID,
			Filename:    "download-test.txt",
			ContentType: "text/plain",
		})
		require.NoError(t, err)

		req, _ := http.NewRequestWithContext(ctx, "PUT", initResult.UploadURL, bytes.NewReader(fileContent))
		req.Header.Set("Content-Type", "text/plain")
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		resp.Body.Close()

		_, err = assetService.CompleteUpload(ctx, CompleteUploadInput{
			AssetID:     initResult.AssetID,
			OrgID:       org.ID,
			Filename:    "download-test.txt",
			ContentType: "text/plain",
			StorageKey:  initResult.StorageKey,
			SHA256:      sha256Hash,
		})
		require.NoError(t, err)

		downloadURL, err := assetService.GetDownloadURL(ctx, initResult.AssetID)
		require.NoError(t, err)
		assert.NotEmpty(t, downloadURL)

		downloadResp, err := http.Get(downloadURL)
		require.NoError(t, err)
		defer downloadResp.Body.Close()
		assert.Equal(t, http.StatusOK, downloadResp.StatusCode)

		downloadedContent, err := io.ReadAll(downloadResp.Body)
		require.NoError(t, err)
		assert.Equal(t, fileContent, downloadedContent)
	})

	t.Run("Delete removes asset from storage and database", func(t *testing.T) {
		fileContent := []byte("Delete test content")
		hash := sha256.Sum256(fileContent)
		sha256Hash := hex.EncodeToString(hash[:])

		initResult, err := assetService.InitUpload(ctx, InitUploadInput{
			OrgID:       org.ID,
			Filename:    "delete-test.txt",
			ContentType: "text/plain",
		})
		require.NoError(t, err)

		req, _ := http.NewRequestWithContext(ctx, "PUT", initResult.UploadURL, bytes.NewReader(fileContent))
		req.Header.Set("Content-Type", "text/plain")
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		resp.Body.Close()

		_, err = assetService.CompleteUpload(ctx, CompleteUploadInput{
			AssetID:     initResult.AssetID,
			OrgID:       org.ID,
			Filename:    "delete-test.txt",
			ContentType: "text/plain",
			StorageKey:  initResult.StorageKey,
			SHA256:      sha256Hash,
		})
		require.NoError(t, err)

		err = assetService.Delete(ctx, initResult.AssetID)
		require.NoError(t, err)

		_, err = assetRepo.GetByID(ctx, initResult.AssetID)
		assert.Equal(t, domain.ErrAssetNotFound, err)
	})

	t.Run("CompleteUpload fails if file not uploaded", func(t *testing.T) {
		initResult, err := assetService.InitUpload(ctx, InitUploadInput{
			OrgID:       org.ID,
			Filename:    "never-uploaded.txt",
			ContentType: "text/plain",
		})
		require.NoError(t, err)

		_, err = assetService.CompleteUpload(ctx, CompleteUploadInput{
			AssetID:     initResult.AssetID,
			OrgID:       org.ID,
			Filename:    "never-uploaded.txt",
			ContentType: "text/plain",
			StorageKey:  initResult.StorageKey,
			SHA256:      "any-hash-value",
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to verify uploaded file")

		_, err = assetRepo.GetByID(ctx, initResult.AssetID)
		assert.Equal(t, domain.ErrAssetNotFound, err)
	})
}

type S3StorageAdapter struct {
	client *storage.S3Client
}

func (a *S3StorageAdapter) GenerateUploadURL(ctx context.Context, key string, contentType string) (string, error) {
	return a.client.GenerateUploadURL(ctx, key, contentType)
}

func (a *S3StorageAdapter) GenerateDownloadURL(ctx context.Context, key string) (string, error) {
	return a.client.GenerateDownloadURL(ctx, key)
}

func (a *S3StorageAdapter) DeleteObject(ctx context.Context, key string) error {
	return a.client.DeleteObject(ctx, key)
}

func (a *S3StorageAdapter) HeadObject(ctx context.Context, key string) (*ObjectMetadata, error) {
	meta, err := a.client.HeadObject(ctx, key)
	if err != nil {
		return nil, err
	}
	return &ObjectMetadata{
		ContentLength: meta.ContentLength,
		ContentType:   meta.ContentType,
		ETag:          meta.ETag,
	}, nil
}
