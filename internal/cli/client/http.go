package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

const (
	envAPIKey = "NEOTEX_API_KEY"
	envAPIURL = "NEOTEX_API_URL"

	defaultAPIURL = "http://localhost:8080"
)

type APIClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewAPIClientWithCmd creates an APIClient with config cascade: flag → env → global config → default
// If cmd is nil, skips flag checking and goes directly to env → global config
func NewAPIClientWithCmd(cmd *cobra.Command) (*APIClient, error) {
	var apiKey, baseURL string

	// Priority 1: Check flag if cmd is provided
	if cmd != nil {
		if flagKey, err := cmd.Flags().GetString("api-key"); err == nil && flagKey != "" {
			apiKey = flagKey
		}
		if flagURL, err := cmd.Flags().GetString("api-url"); err == nil && flagURL != "" {
			baseURL = flagURL
		}
	}

	// Priority 2: Check environment variables (only if not found in flags)
	if apiKey == "" {
		apiKey = os.Getenv(envAPIKey)
	}
	if baseURL == "" {
		baseURL = os.Getenv(envAPIURL)
	}

	// Priority 3: Check global config (only if not found in env)
	if apiKey == "" || baseURL == "" {
		globalConfig, err := LoadGlobalConfig()
		if err != nil {
			return nil, err
		}
		if globalConfig != nil {
			if apiKey == "" && globalConfig.APIKey != "" {
				apiKey = globalConfig.APIKey
			}
			if baseURL == "" && globalConfig.APIURL != "" {
				baseURL = globalConfig.APIURL
			}
		}
	}

	// Validate API key is set
	if apiKey == "" {
		return nil, fmt.Errorf("%s not set (run 'neotex init' or set environment variable)", envAPIKey)
	}

	// Use default URL if still not set
	if baseURL == "" {
		baseURL = defaultAPIURL
	}

	return NewAPIClientWithConfig(apiKey, baseURL)
}

func NewAPIClient() (*APIClient, error) {
	_ = godotenv.Load()
	return NewAPIClientWithCmd(nil)
}

// NewAPIClientWithConfig creates an APIClient with explicit config (used by init before .env exists).
func NewAPIClientWithConfig(apiKey, baseURL string) (*APIClient, error) {
	return &APIClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// APIResponse represents the standard API response format.
type APIResponse struct {
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

// APIError represents an error from the API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (%d): %s", e.StatusCode, e.Message)
}

// Get performs a GET request.
func (c *APIClient) Get(path string) (*APIResponse, error) {
	return c.do("GET", path, nil)
}

// Post performs a POST request with JSON body.
func (c *APIClient) Post(path string, body interface{}) (*APIResponse, error) {
	return c.do("POST", path, body)
}

// Put performs a PUT request with JSON body.
func (c *APIClient) Put(path string, body interface{}) (*APIResponse, error) {
	return c.do("PUT", path, body)
}

// Delete performs a DELETE request.
func (c *APIClient) Delete(path string) (*APIResponse, error) {
	return c.do("DELETE", path, nil)
}

// RequestOptions contains optional settings for HTTP requests.
type RequestOptions struct {
	IdempotencyKey string
}

// PostWithOptions performs a POST request with JSON body and options.
func (c *APIClient) PostWithOptions(path string, body interface{}, opts RequestOptions) (*APIResponse, error) {
	return c.doWithOptions("POST", path, body, opts)
}

// DeleteWithOptions performs a DELETE request with options.
func (c *APIClient) DeleteWithOptions(path string, opts RequestOptions) (*APIResponse, error) {
	return c.doWithOptions("DELETE", path, nil, opts)
}

func (c *APIClient) do(method, path string, body interface{}) (*APIResponse, error) {
	return c.doWithOptions(method, path, body, RequestOptions{})
}

func (c *APIClient) doWithOptions(method, path string, body interface{}, opts RequestOptions) (*APIResponse, error) {
	url := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	if opts.IdempotencyKey != "" {
		req.Header.Set("Idempotency-Key", opts.IdempotencyKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		if resp.StatusCode >= 400 {
			return nil, &APIError{
				StatusCode: resp.StatusCode,
				Message:    string(respBody),
			}
		}
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    apiResp.Error,
		}
	}

	return &apiResp, nil
}

// UploadFile uploads a file to the given presigned URL.
func (c *APIClient) UploadFile(uploadURL, filePath, contentType string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info for content length
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	return c.UploadReader(uploadURL, file, stat.Size(), contentType)
}

// UploadReader uploads data from an io.Reader to the given presigned URL.
func (c *APIClient) UploadReader(uploadURL string, reader io.Reader, size int64, contentType string) error {
	req, err := http.NewRequest("PUT", uploadURL, reader)
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)
	req.ContentLength = size

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ProgressFunc is a callback for reporting upload/download progress.
type ProgressFunc func(current, total int64)

// progressReader wraps an io.Reader and reports progress.
type progressReader struct {
	reader     io.Reader
	total      int64
	current    int64
	onProgress ProgressFunc
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.current += int64(n)
	if pr.onProgress != nil {
		pr.onProgress(pr.current, pr.total)
	}
	return n, err
}

// UploadReaderWithProgress uploads data with progress reporting.
func (c *APIClient) UploadReaderWithProgress(uploadURL string, reader io.Reader, size int64, contentType string, onProgress ProgressFunc) error {
	pr := &progressReader{
		reader:     reader,
		total:      size,
		onProgress: onProgress,
	}

	req, err := http.NewRequest("PUT", uploadURL, pr)
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)
	req.ContentLength = size

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DownloadFile downloads a file from the given URL to the specified path.
func (c *APIClient) DownloadFile(url, outputPath string) error {
	return c.DownloadFileWithProgress(url, outputPath, nil)
}

// DownloadFileWithProgress downloads a file with progress reporting.
func (c *APIClient) DownloadFileWithProgress(url, outputPath string, onProgress ProgressFunc) error {
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer out.Close()

	var reader io.Reader = resp.Body
	if onProgress != nil {
		reader = &progressReader{
			reader:     resp.Body,
			total:      resp.ContentLength,
			onProgress: onProgress,
		}
	}

	_, err = io.Copy(out, reader)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
