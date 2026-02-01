package client

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// DownloadURLResponse represents the download URL API response.
type DownloadURLResponse struct {
	DownloadURL string `json:"download_url"`
}

// InitUploadRequest represents the init upload API request.
type InitUploadRequest struct {
	Filename  string `json:"filename"`
	MimeType  string `json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
	ProjectID string `json:"project_id"`
}

// InitUploadResponse represents the init upload API response.
type InitUploadResponse struct {
	AssetID    string `json:"asset_id"`
	StorageKey string `json:"storage_key"`
	UploadURL  string `json:"upload_url"`
}

// CompleteUploadRequest represents the complete upload API request.
type CompleteUploadRequest struct {
	AssetID     string   `json:"asset_id"`
	StorageKey  string   `json:"storage_key"`
	Filename    string   `json:"filename"`
	MimeType    string   `json:"mime_type"`
	SHA256      string   `json:"sha256"`
	ProjectID   string   `json:"project_id,omitempty"`
	KnowledgeID string   `json:"knowledge_id,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	Description string   `json:"description,omitempty"`
}

// AssetResponse represents the asset API response.
type AssetResponse struct {
	ID          string   `json:"id"`
	OrgID       string   `json:"org_id"`
	ProjectID   string   `json:"project_id"`
	Filename    string   `json:"filename"`
	MimeType    string   `json:"mime_type"`
	SHA256      string   `json:"sha256"`
	Keywords    []string `json:"keywords,omitempty"`
	Description string   `json:"description,omitempty"`
	CreatedAt   string   `json:"created_at"`
}

// AssetCmd creates the asset command group.
func AssetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "asset",
		Short: "Asset management commands",
		Long:  "Commands for managing assets (binary files) in the knowledge base.",
	}

	cmd.AddCommand(AssetAddCmd())
	cmd.AddCommand(AssetGetCmd())

	return cmd
}

// AssetAddCmd creates the asset add command.
func AssetAddCmd() *cobra.Command {
	var (
		description string
		keywords    string
		knowledgeID string
	)

	cmd := &cobra.Command{
		Use:   "add <filepath>",
		Short: "Upload an asset file",
		Long: `Upload a file as an asset to the knowledge base.

Examples:
  # Upload a reference image
  neotex asset add mockup.png --description "Login page mockup" --keywords "ui,login,mockup"

  # Upload and link to existing knowledge
  neotex asset add diagram.png --knowledge-id abc123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputJSON, _ := cmd.Flags().GetBool("output")
			var kw []string
			if keywords != "" {
				kw = strings.Split(keywords, ",")
				for i := range kw {
					kw[i] = strings.TrimSpace(kw[i])
				}
			}
			return runAssetAdd(args[0], description, kw, knowledgeID, outputJSON)
		},
	}

	cmd.Flags().StringVarP(&description, "description", "d", "", "Description of the asset")
	cmd.Flags().StringVarP(&keywords, "keywords", "k", "", "Comma-separated keywords for searchability")
	cmd.Flags().StringVar(&knowledgeID, "knowledge-id", "", "Link to existing knowledge item")

	return cmd
}

func runAssetAdd(filePath, description string, keywords []string, knowledgeID string, outputJSON bool) error {
	// Load config to get project ID
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	// Open file and get info
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	filename := filepath.Base(filePath)

	// Detect MIME type
	mimeType := mime.TypeByExtension(filepath.Ext(filePath))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Calculate SHA256
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("failed to calculate hash: %w", err)
	}
	sha256Hash := hex.EncodeToString(hash.Sum(nil))

	// Reset file for upload
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to reset file: %w", err)
	}

	// Create API client
	api, err := NewAPIClient()
	if err != nil {
		return err
	}

	// Step 1: Init upload
	initReq := InitUploadRequest{
		Filename:  filename,
		MimeType:  mimeType,
		SizeBytes: stat.Size(),
		ProjectID: config.ProjectID,
	}

	initResp, err := api.Post("/assets/init", initReq)
	if err != nil {
		return fmt.Errorf("failed to init upload: %w", err)
	}

	var uploadInfo InitUploadResponse
	if err := json.Unmarshal(initResp.Data, &uploadInfo); err != nil {
		return fmt.Errorf("failed to parse init response: %w", err)
	}

	// Step 2: Upload to presigned URL
	if err := api.UploadFile(uploadInfo.UploadURL, filePath, mimeType); err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	// Step 3: Complete upload
	completeReq := CompleteUploadRequest{
		AssetID:     uploadInfo.AssetID,
		StorageKey:  uploadInfo.StorageKey,
		Filename:    filename,
		MimeType:    mimeType,
		SHA256:      sha256Hash,
		ProjectID:   config.ProjectID,
		Keywords:    keywords,
		Description: description,
		KnowledgeID: knowledgeID,
	}

	completeResp, err := api.Post("/assets/complete", completeReq)
	if err != nil {
		return fmt.Errorf("failed to complete upload: %w", err)
	}

	var asset AssetResponse
	if err := json.Unmarshal(completeResp.Data, &asset); err != nil {
		return fmt.Errorf("failed to parse complete response: %w", err)
	}

	if outputJSON {
		output, _ := json.MarshalIndent(asset, "", "  ")
		fmt.Println(string(output))
	} else {
		fmt.Printf("Uploaded asset: %s\n", asset.ID)
		fmt.Printf("Filename: %s\n", asset.Filename)
		if asset.Description != "" {
			fmt.Printf("Description: %s\n", asset.Description)
		}
		if len(asset.Keywords) > 0 {
			fmt.Printf("Keywords: %s\n", strings.Join(asset.Keywords, ", "))
		}
	}

	return nil
}

// AssetGetCmd creates the asset get command.
func AssetGetCmd() *cobra.Command {
	var outputPath string

	cmd := &cobra.Command{
		Use:   "get <asset_id>",
		Short: "Download an asset by ID",
		Long:  "Downloads an asset from the knowledge base by its ID.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputJSON, _ := cmd.Flags().GetBool("output")
			return runAssetGet(args[0], outputPath, outputJSON)
		},
	}

	cmd.Flags().StringVarP(&outputPath, "dest", "o", "", "Output file path (default: current directory with original filename)")

	return cmd
}

func runAssetGet(assetID, outputPath string, outputJSON bool) error {
	// Create API client
	api, err := NewAPIClient()
	if err != nil {
		return err
	}

	// Get download URL
	resp, err := api.Get(fmt.Sprintf("/assets/%s/download", assetID))
	if err != nil {
		return fmt.Errorf("failed to get download URL: %w", err)
	}

	// Parse response
	var downloadResp DownloadURLResponse
	if err := json.Unmarshal(resp.Data, &downloadResp); err != nil {
		return fmt.Errorf("failed to parse download URL response: %w", err)
	}

	if downloadResp.DownloadURL == "" {
		return fmt.Errorf("no download URL returned")
	}

	// Determine output path
	if outputPath == "" {
		// Extract filename from URL or use asset ID
		outputPath = extractFilenameFromURL(downloadResp.DownloadURL)
		if outputPath == "" {
			outputPath = assetID
		}
	}

	// Download file
	if err := api.DownloadFile(downloadResp.DownloadURL, outputPath); err != nil {
		return fmt.Errorf("failed to download asset: %w", err)
	}

	if outputJSON {
		result := map[string]interface{}{
			"success":  true,
			"asset_id": assetID,
			"path":     outputPath,
		}
		output, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(output))
	} else {
		fmt.Printf("Downloaded asset to %s\n", outputPath)
	}

	return nil
}

// extractFilenameFromURL extracts the filename from a URL path.
func extractFilenameFromURL(url string) string {
	// Simple extraction - get the last path component before any query params
	path := url
	if idx := indexOf(path, '?'); idx != -1 {
		path = path[:idx]
	}
	return filepath.Base(path)
}

func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}
