package client

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveAssetInput_FilePath(t *testing.T) {
	// Create a temp file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("hello world")
	require.NoError(t, os.WriteFile(testFile, testContent, 0644))

	input, err := resolveAssetInput(testFile, "", false, "raw", "")
	require.NoError(t, err)
	assert.Equal(t, testContent, input.data)
	assert.Equal(t, "test.txt", input.filename)
	assert.Equal(t, int64(len(testContent)), input.size)
}

func TestResolveAssetInput_FilePath_NotExists(t *testing.T) {
	input, err := resolveAssetInput("/nonexistent/file.txt", "", false, "raw", "")
	assert.Error(t, err)
	assert.Nil(t, input)
	assert.Contains(t, err.Error(), "failed to read file")
}

func TestResolveAssetInput_Base64(t *testing.T) {
	content := []byte("hello world")
	b64 := base64.StdEncoding.EncodeToString(content)

	input, err := resolveAssetInput("", b64, false, "raw", "test.txt")
	require.NoError(t, err)
	assert.Equal(t, content, input.data)
	assert.Equal(t, "test.txt", input.filename)
	assert.Equal(t, int64(len(content)), input.size)
}

func TestResolveAssetInput_Base64_MissingFilename(t *testing.T) {
	content := []byte("hello world")
	b64 := base64.StdEncoding.EncodeToString(content)

	input, err := resolveAssetInput("", b64, false, "raw", "")
	assert.Error(t, err)
	assert.Nil(t, input)
	assert.Contains(t, err.Error(), "--filename is required")
}

func TestResolveAssetInput_Base64_Invalid(t *testing.T) {
	input, err := resolveAssetInput("", "not-valid-base64!!!", false, "raw", "test.txt")
	assert.Error(t, err)
	assert.Nil(t, input)
	assert.Contains(t, err.Error(), "failed to decode base64")
}

func TestResolveAssetInput_NoInput(t *testing.T) {
	input, err := resolveAssetInput("", "", false, "raw", "")
	assert.Error(t, err)
	assert.Nil(t, input)
	assert.Contains(t, err.Error(), "must provide filepath, --base64, or --stdin")
}

func TestDetectMimeType_ExplicitOverride(t *testing.T) {
	mimeType := detectMimeType("image.png", "application/custom")
	assert.Equal(t, "application/custom", mimeType)
}

func TestDetectMimeType_FromExtension(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"image.png", "image/png"},
		{"document.pdf", "application/pdf"},
		{"script.js", "text/javascript; charset=utf-8"},
		{"data.json", "application/json"},
		{"style.css", "text/css; charset=utf-8"},
		{"page.html", "text/html; charset=utf-8"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			mimeType := detectMimeType(tt.filename, "")
			assert.Equal(t, tt.expected, mimeType)
		})
	}
}

func TestDetectMimeType_UnknownExtension(t *testing.T) {
	mimeType := detectMimeType("file.xyz123", "")
	assert.Equal(t, "application/octet-stream", mimeType)
}

func TestDetectMimeType_NoExtension(t *testing.T) {
	mimeType := detectMimeType("noextension", "")
	assert.Equal(t, "application/octet-stream", mimeType)
}

func TestExtractFilenameFromURL(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://example.com/files/test.png", "test.png"},
		{"https://example.com/files/test.png?token=abc", "test.png"},
		{"https://example.com/test.pdf", "test.pdf"},
		{"https://example.com/path/to/file.txt?a=1&b=2", "file.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := extractFilenameFromURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}
