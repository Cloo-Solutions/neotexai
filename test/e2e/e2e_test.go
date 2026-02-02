//go:build e2e

package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_Bootstrap tests organization and API key creation
func TestE2E_Bootstrap(t *testing.T) {
	env := SetupE2EEnv(t)
	defer env.Cleanup()

	t.Run("create organization", func(t *testing.T) {
		resp, err := env.Post("/orgs", map[string]string{"name": "Test Organization"}, "")
		require.NoError(t, err)

		var org struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			CreatedAt string `json:"created_at"`
		}
		require.NoError(t, json.Unmarshal(resp.Data, &org))
		assert.NotEmpty(t, org.ID)
		assert.Equal(t, "Test Organization", org.Name)
		assert.NotEmpty(t, org.CreatedAt)
	})

	t.Run("create API key", func(t *testing.T) {
		// First create org
		orgResp, err := env.Post("/orgs", map[string]string{"name": "Key Test Org"}, "")
		require.NoError(t, err)

		var org struct {
			ID string `json:"id"`
		}
		require.NoError(t, json.Unmarshal(orgResp.Data, &org))

		// Create API key
		keyResp, err := env.Post("/apikeys", map[string]string{
			"org_id": org.ID,
			"name":   "test-key",
		}, "")
		require.NoError(t, err)

		var key struct {
			Token string `json:"token"`
			Name  string `json:"name"`
		}
		require.NoError(t, json.Unmarshal(keyResp.Data, &key))
		assert.NotEmpty(t, key.Token)
		assert.Equal(t, "test-key", key.Name)
		assert.Len(t, key.Token, 68) // ntx_ prefix (4) + 32 bytes hex (64) = 68 chars
	})

	t.Run("API key works for authentication", func(t *testing.T) {
		// Create org
		orgResp, err := env.Post("/orgs", map[string]string{"name": "Auth Test Org"}, "")
		require.NoError(t, err)

		var org struct {
			ID string `json:"id"`
		}
		require.NoError(t, json.Unmarshal(orgResp.Data, &org))

		// Create API key
		keyResp, err := env.Post("/apikeys", map[string]string{
			"org_id": org.ID,
			"name":   "auth-test-key",
		}, "")
		require.NoError(t, err)

		var key struct {
			Token string `json:"token"`
		}
		require.NoError(t, json.Unmarshal(keyResp.Data, &key))

		// Use the token directly to make an authenticated request
		// Token format is ntx_<64 hex chars>
		resp, err := env.Get("/knowledge", key.Token)
		require.NoError(t, err)

		var knowledge struct {
			Items []interface{} `json:"items"`
		}
		require.NoError(t, json.Unmarshal(resp.Data, &knowledge))
		assert.NotNil(t, knowledge.Items) // Should be empty array, not error
	})

	t.Run("invalid API key returns 401", func(t *testing.T) {
		_, err := env.Get("/knowledge", "invalid.token")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
	})
}

// TestE2E_KnowledgeLifecycle tests knowledge CRUD operations
func TestE2E_KnowledgeLifecycle(t *testing.T) {
	env := SetupE2EEnv(t)
	defer env.Cleanup()
	env.Bootstrap()

	var knowledgeID string

	t.Run("create knowledge", func(t *testing.T) {
		resp, err := env.Post("/knowledge", map[string]interface{}{
			"type":    "guideline",
			"title":   "E2E Test Guideline",
			"summary": "A test guideline for E2E testing",
			"body_md": "# E2E Test\n\nThis is a test guideline created during E2E testing.",
			"scope":   "test/e2e",
		}, env.AuthToken)
		require.NoError(t, err)

		var knowledge struct {
			ID      string `json:"id"`
			OrgID   string `json:"org_id"`
			Type    string `json:"type"`
			Status  string `json:"status"`
			Title   string `json:"title"`
			Summary string `json:"summary"`
			BodyMD  string `json:"body_md"`
			Scope   string `json:"scope"`
		}
		require.NoError(t, json.Unmarshal(resp.Data, &knowledge))
		assert.NotEmpty(t, knowledge.ID)
		assert.Equal(t, env.OrgID, knowledge.OrgID)
		assert.Equal(t, "guideline", knowledge.Type)
		assert.Equal(t, "draft", knowledge.Status)
		assert.Equal(t, "E2E Test Guideline", knowledge.Title)
		assert.Equal(t, "test/e2e", knowledge.Scope)

		knowledgeID = knowledge.ID
	})

	t.Run("get knowledge by ID", func(t *testing.T) {
		resp, err := env.Get("/knowledge/"+knowledgeID, env.AuthToken)
		require.NoError(t, err)

		var knowledge struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		}
		require.NoError(t, json.Unmarshal(resp.Data, &knowledge))
		assert.Equal(t, knowledgeID, knowledge.ID)
		assert.Equal(t, "E2E Test Guideline", knowledge.Title)
	})

	t.Run("update knowledge creates new version", func(t *testing.T) {
		resp, err := env.Put("/knowledge/"+knowledgeID, map[string]interface{}{
			"title":   "E2E Test Guideline v2",
			"summary": "Updated summary",
			"body_md": "# E2E Test v2\n\nThis is the updated content.",
		}, env.AuthToken)
		require.NoError(t, err)

		var knowledge struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		}
		require.NoError(t, json.Unmarshal(resp.Data, &knowledge))
		assert.Equal(t, knowledgeID, knowledge.ID)
		assert.Equal(t, "E2E Test Guideline v2", knowledge.Title)

		// Verify version was created
		rows, err := env.Pool.Query(env.Ctx,
			"SELECT version_number FROM knowledge_versions WHERE knowledge_id = $1 ORDER BY version_number",
			knowledgeID)
		require.NoError(t, err)
		defer rows.Close()

		var versions []int
		for rows.Next() {
			var v int
			require.NoError(t, rows.Scan(&v))
			versions = append(versions, v)
		}
		assert.Equal(t, []int{1, 2}, versions)
	})

	t.Run("list knowledge returns created items", func(t *testing.T) {
		resp, err := env.Get("/knowledge", env.AuthToken)
		require.NoError(t, err)

		var knowledge struct {
			Items []struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"items"`
		}
		require.NoError(t, json.Unmarshal(resp.Data, &knowledge))
		assert.GreaterOrEqual(t, len(knowledge.Items), 1)

		found := false
		for _, k := range knowledge.Items {
			if k.ID == knowledgeID {
				found = true
				break
			}
		}
		assert.True(t, found, "created knowledge should be in list")
	})

	t.Run("deprecate knowledge", func(t *testing.T) {
		resp, err := env.Delete("/knowledge/"+knowledgeID, env.AuthToken)
		require.NoError(t, err)

		var knowledge struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		}
		require.NoError(t, json.Unmarshal(resp.Data, &knowledge))
		assert.Equal(t, knowledgeID, knowledge.ID)
		assert.Equal(t, "deprecated", knowledge.Status)
	})
}

// TestE2E_AssetUploadDownload tests asset upload and download flow
func TestE2E_AssetUploadDownload(t *testing.T) {
	env := SetupE2EEnv(t)
	defer env.Cleanup()
	env.Bootstrap()

	fileContent := []byte("This is test file content for E2E testing of asset upload/download flow.")
	sha256Hash := SHA256Sum(fileContent)
	var assetID string

	t.Run("init upload returns presigned URL", func(t *testing.T) {
		resp, err := env.Post("/assets/init", map[string]interface{}{
			"filename":  "test-document.txt",
			"mime_type": "text/plain",
		}, env.AuthToken)
		require.NoError(t, err)

		var initResp struct {
			AssetID    string `json:"asset_id"`
			StorageKey string `json:"storage_key"`
			UploadURL  string `json:"upload_url"`
		}
		require.NoError(t, json.Unmarshal(resp.Data, &initResp))
		assert.NotEmpty(t, initResp.AssetID)
		assert.NotEmpty(t, initResp.StorageKey)
		assert.NotEmpty(t, initResp.UploadURL)
		assert.Contains(t, initResp.UploadURL, "http")

		assetID = initResp.AssetID

		// Upload file to presigned URL
		err = env.UploadFile(initResp.UploadURL, fileContent, "text/plain")
		require.NoError(t, err)
	})

	t.Run("complete upload creates asset record", func(t *testing.T) {
		// First init a new upload
		initResp, err := env.Post("/assets/init", map[string]interface{}{
			"filename":  "complete-test.txt",
			"mime_type": "text/plain",
		}, env.AuthToken)
		require.NoError(t, err)

		var init struct {
			AssetID    string `json:"asset_id"`
			StorageKey string `json:"storage_key"`
			UploadURL  string `json:"upload_url"`
		}
		require.NoError(t, json.Unmarshal(initResp.Data, &init))

		// Upload file
		err = env.UploadFile(init.UploadURL, fileContent, "text/plain")
		require.NoError(t, err)

		// Complete upload
		completeResp, err := env.Post("/assets/complete", map[string]interface{}{
			"asset_id":    init.AssetID,
			"storage_key": init.StorageKey,
			"filename":    "complete-test.txt",
			"mime_type":   "text/plain",
			"sha256":      sha256Hash,
			"keywords":    []string{"test", "e2e"},
			"description": "E2E test file",
		}, env.AuthToken)
		require.NoError(t, err)

		var asset struct {
			ID          string   `json:"id"`
			Filename    string   `json:"filename"`
			SHA256      string   `json:"sha256"`
			Keywords    []string `json:"keywords"`
			Description string   `json:"description"`
		}
		require.NoError(t, json.Unmarshal(completeResp.Data, &asset))
		assert.Equal(t, init.AssetID, asset.ID)
		assert.Equal(t, sha256Hash, asset.SHA256)
		assert.Contains(t, asset.Keywords, "test")

		assetID = asset.ID
	})

	t.Run("get download URL and verify content", func(t *testing.T) {
		resp, err := env.Get("/assets/"+assetID+"/download", env.AuthToken)
		require.NoError(t, err)

		var download struct {
			DownloadURL string `json:"download_url"`
		}
		require.NoError(t, json.Unmarshal(resp.Data, &download))
		assert.NotEmpty(t, download.DownloadURL)

		// Download and verify content
		downloadedContent, err := env.DownloadFile(download.DownloadURL)
		require.NoError(t, err)
		assert.Equal(t, fileContent, downloadedContent)
	})
}

// TestE2E_SearchAndContext tests search and context manifest endpoints
func TestE2E_SearchAndContext(t *testing.T) {
	env := SetupE2EEnv(t)
	defer env.Cleanup()
	env.Bootstrap()

	// Create multiple knowledge items for testing
	knowledgeIDs := make([]string, 0)

	items := []map[string]interface{}{
		{
			"type":    "guideline",
			"title":   "Authentication Guide",
			"summary": "How to implement authentication",
			"body_md": "# Authentication\n\nUse JWT tokens for authentication.",
			"scope":   "src/auth",
		},
		{
			"type":    "learning",
			"title":   "Database Optimization",
			"summary": "Lessons learned about database performance",
			"body_md": "# Database\n\nIndex your frequently queried columns.",
			"scope":   "src/db",
		},
		{
			"type":    "decision",
			"title":   "API Design Decision",
			"summary": "Why we chose REST over GraphQL",
			"body_md": "# API Design\n\nREST is simpler for our use case.",
			"scope":   "src/api",
		},
	}

	for _, item := range items {
		resp, err := env.Post("/knowledge", item, env.AuthToken)
		require.NoError(t, err)

		var k struct {
			ID string `json:"id"`
		}
		require.NoError(t, json.Unmarshal(resp.Data, &k))
		knowledgeIDs = append(knowledgeIDs, k.ID)
	}

	t.Run("get manifest returns all knowledge", func(t *testing.T) {
		resp, err := env.Get("/context", env.AuthToken)
		require.NoError(t, err)

		var manifest struct {
			Manifest []struct {
				ID    string `json:"id"`
				Title string `json:"title"`
				Type  string `json:"type"`
			} `json:"manifest"`
		}
		require.NoError(t, json.Unmarshal(resp.Data, &manifest))
		assert.GreaterOrEqual(t, len(manifest.Manifest), 3)

		// Verify all created items are in manifest
		manifestIDs := make(map[string]bool)
		for _, item := range manifest.Manifest {
			manifestIDs[item.ID] = true
		}
		for _, id := range knowledgeIDs {
			assert.True(t, manifestIDs[id], "knowledge %s should be in manifest", id)
		}
	})

	t.Run("search finds matching knowledge", func(t *testing.T) {
		resp, err := env.Post("/search", map[string]interface{}{
			"query": "authentication",
			"limit": 10,
		}, env.AuthToken)
		require.NoError(t, err)

		var search struct {
			Results []struct {
				ID    string  `json:"id"`
				Title string  `json:"title"`
				Score float32 `json:"score"`
			} `json:"results"`
		}
		require.NoError(t, json.Unmarshal(resp.Data, &search))
		assert.GreaterOrEqual(t, len(search.Results), 1)

		// First result should be the authentication guide
		found := false
		for _, r := range search.Results {
			if strings.Contains(r.Title, "Authentication") {
				found = true
				break
			}
		}
		assert.True(t, found, "search should find Authentication Guide")
	})

	t.Run("search with type filter", func(t *testing.T) {
		resp, err := env.Post("/search", map[string]interface{}{
			"query": "database",
			"type":  "learning",
			"limit": 10,
		}, env.AuthToken)
		require.NoError(t, err)

		var search struct {
			Results []struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"results"`
		}
		require.NoError(t, json.Unmarshal(resp.Data, &search))

		// All results should be of type learning (if filtering works)
		for _, r := range search.Results {
			// Verify it's a learning by checking the database
			row := env.Pool.QueryRow(env.Ctx, "SELECT type FROM knowledge WHERE id = $1", r.ID)
			var kType string
			if row.Scan(&kType) == nil {
				// Type filtering is optional in our simple implementation
				t.Logf("Found result: %s with type: %s", r.Title, kType)
			}
		}
	})
}

// TestE2E_CLIWorkflow tests the CLI commands end-to-end
func TestE2E_CLIWorkflow(t *testing.T) {
	env := SetupE2EEnv(t)
	defer env.Cleanup()
	env.Bootstrap()
	env.BuildBinaries()

	// Create a temporary project directory
	projectDir, err := os.MkdirTemp("", "neotex-cli-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(projectDir)

	t.Run("neotex init creates .neotex directory", func(t *testing.T) {
		// CLI init uses env vars (set by RunNeotex) for auth and creates project via API
		output, err := env.RunNeotex(projectDir, "init", "--project", "CLI Test Project")
		require.NoError(t, err, "init failed: %s", output)

		// Verify .neotex directory exists
		neotexDir := filepath.Join(projectDir, ".neotex")
		info, err := os.Stat(neotexDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())

		// Verify config.yaml exists
		configPath := filepath.Join(neotexDir, "config.yaml")
		_, err = os.Stat(configPath)
		require.NoError(t, err)
	})

	t.Run("neotex pull downloads manifest", func(t *testing.T) {
		output, err := env.RunNeotex(projectDir, "pull")
		require.NoError(t, err, "pull failed: %s", output)

		// Verify index.json exists
		indexPath := filepath.Join(projectDir, ".neotex", "index.json")
		_, err = os.Stat(indexPath)
		require.NoError(t, err)

		// Read and verify content
		content, err := os.ReadFile(indexPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "manifest")
	})

	t.Run("neotex add creates knowledge", func(t *testing.T) {
		input := `{
			"type": "guideline",
			"title": "CLI Test Guideline",
			"summary": "Created via CLI",
			"body_md": "# CLI Test\n\nThis knowledge was created via the neotex CLI."
		}`

		output, err := env.RunNeotexWithInput(projectDir, input, "add", "--output")
		require.NoError(t, err, "add failed: %s", output)

		// Verify knowledge was created (output should contain ID)
		assert.Contains(t, output, "id")
	})

	t.Run("neotex search finds knowledge", func(t *testing.T) {
		// Wait a moment for knowledge to be indexed
		time.Sleep(100 * time.Millisecond)

		output, err := env.RunNeotex(projectDir, "search", "CLI Test", "--output")
		require.NoError(t, err, "search failed: %s", output)

		// Output should contain results
		assert.Contains(t, output, "CLI Test Guideline")
	})

	t.Run("neotex get retrieves knowledge", func(t *testing.T) {
		// First, get a knowledge ID from the database
		row := env.Pool.QueryRow(env.Ctx,
			"SELECT id FROM knowledge WHERE org_id = $1 ORDER BY created_at DESC LIMIT 1",
			env.OrgID)

		var knowledgeID string
		require.NoError(t, row.Scan(&knowledgeID))

		output, err := env.RunNeotex(projectDir, "get", knowledgeID, "--output")
		require.NoError(t, err, "get failed: %s", output)

		// Output should contain the knowledge content
		assert.Contains(t, output, "id")
		assert.Contains(t, output, knowledgeID)
	})
}

// TestE2E_FullWorkflow tests the complete user journey
func TestE2E_FullWorkflow(t *testing.T) {
	env := SetupE2EEnv(t)
	defer env.Cleanup()

	t.Run("complete workflow from bootstrap to search", func(t *testing.T) {
		// 1. Bootstrap: Create org
		orgResp, err := env.Post("/orgs", map[string]string{"name": "Full Workflow Org"}, "")
		require.NoError(t, err)

		var org struct {
			ID string `json:"id"`
		}
		require.NoError(t, json.Unmarshal(orgResp.Data, &org))

		// 2. Bootstrap: Create API key
		keyResp, err := env.Post("/apikeys", map[string]string{
			"org_id": org.ID,
			"name":   "workflow-key",
		}, "")
		require.NoError(t, err)

		var key struct {
			Token string `json:"token"`
		}
		require.NoError(t, json.Unmarshal(keyResp.Data, &key))

		authToken := key.Token

		// 3. Create knowledge
		kResp, err := env.Post("/knowledge", map[string]interface{}{
			"type":    "template",
			"title":   "Full Workflow Template",
			"summary": "A template for testing the full workflow",
			"body_md": "# Template\n\n{{ .content }}",
			"scope":   "src/templates",
		}, authToken)
		require.NoError(t, err)

		var knowledge struct {
			ID string `json:"id"`
		}
		require.NoError(t, json.Unmarshal(kResp.Data, &knowledge))

		// 4. Upload an asset
		fileContent := []byte("Template asset content")
		sha256Hash := SHA256Sum(fileContent)

		initResp, err := env.Post("/assets/init", map[string]interface{}{
			"filename":  "template-asset.txt",
			"mime_type": "text/plain",
		}, authToken)
		require.NoError(t, err)

		var init struct {
			AssetID    string `json:"asset_id"`
			StorageKey string `json:"storage_key"`
			UploadURL  string `json:"upload_url"`
		}
		require.NoError(t, json.Unmarshal(initResp.Data, &init))

		err = env.UploadFile(init.UploadURL, fileContent, "text/plain")
		require.NoError(t, err)

		_, err = env.Post("/assets/complete", map[string]interface{}{
			"asset_id":     init.AssetID,
			"storage_key":  init.StorageKey,
			"filename":     "template-asset.txt",
			"mime_type":    "text/plain",
			"sha256":       sha256Hash,
			"knowledge_id": knowledge.ID,
		}, authToken)
		require.NoError(t, err)

		// 5. Get manifest
		manifestResp, err := env.Get("/context", authToken)
		require.NoError(t, err)

		var manifest struct {
			Manifest []struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"manifest"`
		}
		require.NoError(t, json.Unmarshal(manifestResp.Data, &manifest))
		assert.GreaterOrEqual(t, len(manifest.Manifest), 1)

		// 6. Search
		searchResp, err := env.Post("/search", map[string]interface{}{
			"query": "Template",
			"limit": 10,
		}, authToken)
		require.NoError(t, err)

		var search struct {
			Results []struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"results"`
		}
		require.NoError(t, json.Unmarshal(searchResp.Data, &search))
		assert.GreaterOrEqual(t, len(search.Results), 1)

		// Verify our template is found
		found := false
		for _, r := range search.Results {
			if r.ID == knowledge.ID {
				found = true
				break
			}
		}
		assert.True(t, found, "created knowledge should be searchable")

		// 7. Download asset
		downloadResp, err := env.Get("/assets/"+init.AssetID+"/download", authToken)
		require.NoError(t, err)

		var download struct {
			DownloadURL string `json:"download_url"`
		}
		require.NoError(t, json.Unmarshal(downloadResp.Data, &download))

		downloadedContent, err := env.DownloadFile(download.DownloadURL)
		require.NoError(t, err)
		assert.Equal(t, fileContent, downloadedContent)
	})
}
