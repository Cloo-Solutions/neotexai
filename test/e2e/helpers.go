//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloo-solutions/neotexai/internal/api/handlers"
	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/cloo-solutions/neotexai/internal/repository"
	"github.com/cloo-solutions/neotexai/internal/server"
	"github.com/cloo-solutions/neotexai/internal/service"
	"github.com/cloo-solutions/neotexai/internal/storage"
	"github.com/cloo-solutions/neotexai/internal/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
)

// E2ETestEnv holds all resources needed for E2E tests
type E2ETestEnv struct {
	T            *testing.T
	Ctx          context.Context
	PostgresC    *testutil.PostgresContainer
	RustFSC      *testutil.RustFSContainer
	Pool         *pgxpool.Pool
	ServerURL    string
	ServerCloser func()
	S3Client     *storage.S3Client
	BinaryDir    string
	OrgID        string
	APIKeyID     string
	APIKeyToken  string
	AuthToken    string // keyID.plaintext format
	HTTPClient   *http.Client
}

// SetupE2EEnv creates a full E2E test environment with containers and server
func SetupE2EEnv(t *testing.T) *E2ETestEnv {
	ctx := context.Background()

	// Start PostgreSQL container
	pgC := testutil.NewPostgresContainer(ctx, t)

	// Start RustFS container
	s3C := testutil.NewRustFSContainer(ctx, t)

	// Create connection pool and run migrations
	pool := testutil.NewTestPool(ctx, t, pgC, "../../migrations")

	// Create S3 client
	s3Client, err := storage.NewS3Client(ctx, storage.S3ClientConfig{
		Endpoint:        s3C.Endpoint(),
		Region:          "us-east-1",
		AccessKeyID:     "rustfsadmin",
		SecretAccessKey: "rustfsadmin",
		Bucket:          "test-assets",
		UsePathStyle:    true,
	})
	if err != nil {
		t.Fatalf("failed to create S3 client: %v", err)
	}

	if err := s3Client.EnsureBucket(ctx); err != nil {
		t.Fatalf("failed to create bucket: %v", err)
	}

	// Find free port for server
	port, err := getFreePort()
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}

	// Start HTTP server
	serverURL, serverCloser := startServer(t, pool, s3Client, port)

	env := &E2ETestEnv{
		T:            t,
		Ctx:          ctx,
		PostgresC:    pgC,
		RustFSC:      s3C,
		Pool:         pool,
		ServerURL:    serverURL,
		ServerCloser: serverCloser,
		S3Client:     s3Client,
		HTTPClient:   &http.Client{Timeout: 30 * time.Second},
	}

	return env
}

// Cleanup releases all resources
func (e *E2ETestEnv) Cleanup() {
	if e.ServerCloser != nil {
		e.ServerCloser()
	}
	if e.Pool != nil {
		e.Pool.Close()
	}
	if e.RustFSC != nil {
		e.RustFSC.Terminate(e.Ctx)
	}
	if e.PostgresC != nil {
		e.PostgresC.Terminate(e.Ctx)
	}
	// Clean up binaries
	if e.BinaryDir != "" {
		os.RemoveAll(e.BinaryDir)
	}
}

// Bootstrap creates an organization and API key for testing
func (e *E2ETestEnv) Bootstrap() {
	// Create organization
	orgResp, err := e.Post("/orgs", map[string]string{"name": "E2E Test Org"}, "")
	if err != nil {
		e.T.Fatalf("failed to create org: %v", err)
	}

	var orgData struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(orgResp.Data, &orgData); err != nil {
		e.T.Fatalf("failed to parse org response: %v", err)
	}
	e.OrgID = orgData.ID

	// Create API key
	keyResp, err := e.Post("/apikeys", map[string]string{
		"org_id": e.OrgID,
		"name":   "e2e-test-key",
	}, "")
	if err != nil {
		e.T.Fatalf("failed to create API key: %v", err)
	}

	var keyData struct {
		ID    string `json:"id"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(keyResp.Data, &keyData); err != nil {
		e.T.Fatalf("failed to parse key response: %v", err)
	}
	e.APIKeyID = keyData.ID
	e.APIKeyToken = keyData.Token

	// Fetch the key ID from database (we need the actual key ID for token format)
	rows, err := e.Pool.Query(e.Ctx, "SELECT id FROM api_keys WHERE org_id = $1 ORDER BY created_at DESC LIMIT 1", e.OrgID)
	if err != nil {
		e.T.Fatalf("failed to query API key: %v", err)
	}
	defer rows.Close()
	if rows.Next() {
		if err := rows.Scan(&e.APIKeyID); err != nil {
			e.T.Fatalf("failed to scan API key ID: %v", err)
		}
	}

	e.AuthToken = fmt.Sprintf("%s.%s", e.APIKeyID, e.APIKeyToken)
}

// BuildBinaries builds the neotex and neotexd binaries
func (e *E2ETestEnv) BuildBinaries() {
	tmpDir, err := os.MkdirTemp("", "neotex-e2e-*")
	if err != nil {
		e.T.Fatalf("failed to create temp dir: %v", err)
	}
	e.BinaryDir = tmpDir

	// Build neotexd
	cmd := exec.Command("go", "build", "-o", filepath.Join(tmpDir, "neotexd"), "./cmd/neotexd")
	cmd.Dir = "../.."
	if out, err := cmd.CombinedOutput(); err != nil {
		e.T.Fatalf("failed to build neotexd: %v\n%s", err, out)
	}

	// Build neotex
	cmd = exec.Command("go", "build", "-o", filepath.Join(tmpDir, "neotex"), "./cmd/neotex")
	cmd.Dir = "../.."
	if out, err := cmd.CombinedOutput(); err != nil {
		e.T.Fatalf("failed to build neotex: %v\n%s", err, out)
	}
}

// RunNeotex runs the neotex CLI command
func (e *E2ETestEnv) RunNeotex(workDir string, args ...string) (string, error) {
	cmd := exec.Command(filepath.Join(e.BinaryDir, "neotex"), args...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("NEOTEX_API_KEY=%s", e.AuthToken),
		fmt.Sprintf("NEOTEX_API_URL=%s", e.ServerURL),
	)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// RunNeotexWithInput runs the neotex CLI command with stdin input
func (e *E2ETestEnv) RunNeotexWithInput(workDir string, input string, args ...string) (string, error) {
	cmd := exec.Command(filepath.Join(e.BinaryDir, "neotex"), args...)
	cmd.Dir = workDir
	cmd.Stdin = bytes.NewReader([]byte(input))
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("NEOTEX_API_KEY=%s", e.AuthToken),
		fmt.Sprintf("NEOTEX_API_URL=%s", e.ServerURL),
	)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// APIResponse represents a standard API response
type APIResponse struct {
	Data  json.RawMessage `json:"data"`
	Error string          `json:"error,omitempty"`
}

// Get performs a GET request
func (e *E2ETestEnv) Get(path, authToken string) (*APIResponse, error) {
	return e.doRequest("GET", path, nil, authToken)
}

// Post performs a POST request
func (e *E2ETestEnv) Post(path string, body interface{}, authToken string) (*APIResponse, error) {
	return e.doRequest("POST", path, body, authToken)
}

// Put performs a PUT request
func (e *E2ETestEnv) Put(path string, body interface{}, authToken string) (*APIResponse, error) {
	return e.doRequest("PUT", path, body, authToken)
}

// Delete performs a DELETE request
func (e *E2ETestEnv) Delete(path, authToken string) (*APIResponse, error) {
	return e.doRequest("DELETE", path, nil, authToken)
}

func (e *E2ETestEnv) doRequest(method, path string, body interface{}, authToken string) (*APIResponse, error) {
	url := e.ServerURL + path

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
		}
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, apiResp.Error)
	}

	return &apiResp, nil
}

// UploadFile uploads a file to the presigned URL
func (e *E2ETestEnv) UploadFile(uploadURL string, content []byte, contentType string) error {
	req, err := http.NewRequest("PUT", uploadURL, bytes.NewReader(content))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := e.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, body)
	}

	return nil
}

// DownloadFile downloads a file from the presigned URL
func (e *E2ETestEnv) DownloadFile(downloadURL string) ([]byte, error) {
	resp, err := e.HTTPClient.Get(downloadURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// SHA256Sum calculates SHA256 hash of data
func SHA256Sum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// startServer starts the HTTP server with all handlers
func startServer(t *testing.T, pool *pgxpool.Pool, s3Client *storage.S3Client, port int) (string, func()) {
	// Initialize repositories
	knowledgeRepo := repository.NewKnowledgeRepository(pool)
	embeddingJobRepo := repository.NewEmbeddingJobRepository(pool)
	assetRepo := repository.NewAssetRepository(pool)
	orgRepo := repository.NewOrgRepository(pool)
	apiKeyRepo := repository.NewAPIKeyRepository(pool)

	// Initialize services
	uuidGen := &service.DefaultUUIDGenerator{}
	knowledgeSvc := service.NewKnowledgeService(knowledgeRepo, embeddingJobRepo)
	assetSvc := service.NewAssetService(assetRepo, &s3StorageAdapter{client: s3Client})
	authSvc := service.NewAuthService(orgRepo, apiKeyRepo, uuidGen)

	// Initialize handlers
	knowledgeHandler := handlers.NewKnowledgeHandler(knowledgeSvc)
	assetHandler := handlers.NewAssetHandler(assetSvc)
	authHandler := handlers.NewAuthHandler(authSvc)
	contextHandler := handlers.NewContextHandler(&simpleContextService{repo: knowledgeRepo})

	cfg := server.RouterConfig{
		AuthValidator:    authSvc,
		KnowledgeHandler: knowledgeHandler,
		AssetHandler:     assetHandler,
		ContextHandler:   contextHandler,
		AuthHandler:      authHandler,
	}

	router := server.NewRouter(cfg)
	addr := fmt.Sprintf(":%d", port)

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Logf("server error: %v", err)
		}
	}()

	// Wait for server to start
	serverURL := fmt.Sprintf("http://localhost:%d", port)
	waitForServer(t, serverURL, 10*time.Second)

	return serverURL, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}
}

func waitForServer(t *testing.T, url string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("server did not start within %v", timeout)
}

func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// s3StorageAdapter adapts S3Client to StorageClientInterface
type s3StorageAdapter struct {
	client *storage.S3Client
}

func (a *s3StorageAdapter) GenerateUploadURL(ctx context.Context, key string, contentType string) (string, error) {
	return a.client.GenerateUploadURL(ctx, key, contentType)
}

func (a *s3StorageAdapter) GenerateDownloadURL(ctx context.Context, key string) (string, error) {
	return a.client.GenerateDownloadURL(ctx, key)
}

func (a *s3StorageAdapter) DeleteObject(ctx context.Context, key string) error {
	return a.client.DeleteObject(ctx, key)
}

func (a *s3StorageAdapter) HeadObject(ctx context.Context, key string) (*service.ObjectMetadata, error) {
	meta, err := a.client.HeadObject(ctx, key)
	if err != nil {
		return nil, err
	}
	return &service.ObjectMetadata{
		ContentLength: meta.ContentLength,
		ContentType:   meta.ContentType,
		ETag:          meta.ETag,
	}, nil
}

// simpleContextService provides basic context functionality for E2E tests
type simpleContextService struct {
	repo service.KnowledgeRepositoryInterface
}

func (s *simpleContextService) GetManifest(ctx context.Context, orgID, projectID string) ([]*service.KnowledgeManifestItem, error) {
	var knowledgeList []*domain.Knowledge
	var err error
	if projectID != "" {
		knowledgeList, err = s.repo.ListByProject(ctx, projectID)
	} else {
		knowledgeList, err = s.repo.ListByOrg(ctx, orgID)
	}
	if err != nil {
		return nil, err
	}

	items := make([]*service.KnowledgeManifestItem, len(knowledgeList))
	for i, k := range knowledgeList {
		items[i] = &service.KnowledgeManifestItem{
			ID:      k.ID,
			Title:   k.Title,
			Summary: k.Summary,
			Type:    k.Type,
			Scope:   k.Scope,
		}
	}
	return items, nil
}

func (s *simpleContextService) Search(ctx context.Context, input service.SearchInput) (*service.SearchOutput, error) {
	var knowledgeList []*domain.Knowledge
	var err error

	if input.Filters.ProjectID != "" {
		knowledgeList, err = s.repo.ListByProject(ctx, input.Filters.ProjectID)
	} else {
		knowledgeList, err = s.repo.ListByOrg(ctx, input.Filters.OrgID)
	}
	if err != nil {
		return nil, err
	}

	results := make([]*service.SearchResult, 0)
	for _, k := range knowledgeList {
		if containsIgnoreCase(k.Title, input.Query) || containsIgnoreCase(k.BodyMD, input.Query) {
			results = append(results, &service.SearchResult{
				ID:      k.ID,
				Title:   k.Title,
				Summary: k.Summary,
				Scope:   k.Scope,
				Score:   0.9,
			})
		}
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 20
	}

	hasMore := len(results) > limit
	if hasMore {
		results = results[:limit]
	}

	return &service.SearchOutput{
		Results: results,
		HasMore: hasMore,
	}, nil
}

func containsIgnoreCase(s, substr string) bool {
	if substr == "" {
		return true
	}
	// Simple case-insensitive contains
	return bytes.Contains(
		bytes.ToLower([]byte(s)),
		bytes.ToLower([]byte(substr)),
	)
}
